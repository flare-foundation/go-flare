// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package rpc

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"sync/atomic"
	"time"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core"
	"github.com/ava-labs/libevm/core/rawdb"
	"github.com/ava-labs/libevm/core/state"
	"github.com/ava-labs/libevm/core/types"
	"github.com/ava-labs/libevm/core/vm"
	"github.com/ava-labs/libevm/eth/tracers"
	"github.com/ava-labs/libevm/eth/tracers/logger"
	"github.com/ava-labs/libevm/rpc"
	"go.uber.org/zap"

	"github.com/ava-labs/avalanchego/vms/saevm/hook"
	"github.com/ava-labs/avalanchego/vms/saevm/saexec"
)

// This file shadows libevm's block-level tracing endpoints on [tracerAPI] so
// that every transaction is replayed through [saexec.ApplyTransaction] — the
// same era-splitting pipeline used by live execution ([saexec.Execute]) and by
// [backend.StateAtTransaction] — instead of libevm's plain core.ApplyMessage.
// On Flare/Songbird chains the plain replay misses the Flare per-transaction
// mechanisms (fee burn/refund, daemon minting, and, pre-Helicon, coreth's
// full StateTransition including the one-time transition-contract calls), so
// traces and intermediate roots would diverge from the canonical state.
//
// Only the per-transaction application differs from upstream. The replay base
// state — the parent's post-execution state with the traced block's
// start-executing-block changes applied — and the reported block hash come
// from the same backends that serve upstream's [tracers.API]: [tracerBackend]
// for canonical blocks and [suppliedHashBackend] for caller-supplied ones (see
// the package README). Everything else mirrors libevm's eth/tracers/api.go
// byte-for-byte (error strings, tracer construction, timeouts, partial-result
// semantics). Check for differences in the upstream implementation when
// updating libevm.
//
// Shadowed here: debug_traceBlockByNumber, debug_traceBlockByHash and
// debug_intermediateRoots. debug_traceBlock and debug_traceBlockFromFile are
// shadowed in stateful.go (upstream re-seals the caller-supplied block's base
// fee there) and dispatch to [tracerAPI.traceBlockReplay].
//
// Not shadowed (still libevm's plain-ApplyMessage replay): debug_traceChain,
// debug_standardTraceBlockToFile, debug_traceBadBlock, and
// debug_standardTraceBadBlockToFile.

// Mirrors of libevm eth/tracers unexported defaults.
const (
	defaultTraceTimeout = 5 * time.Second
	defaultTraceReexec  = uint64(128) // resolved for parity; [backend.StateAtBlock] ignores it
)

// replayBackend is the part of the tracer backends that differs between
// tracing a canonical block ([tracerBackend]) and a caller-supplied one
// ([suppliedHashBackend]): the replay base state and the reported block hash.
// See the README table for the behaviour of each implementation.
type replayBackend interface {
	// StateAtBlock returns the parent's post-execution state with the traced
	// block's start-executing-block changes applied.
	StateAtBlock(ctx context.Context, parent *types.Block, reexec uint64, base *state.StateDB, readOnly bool, preferDisk bool) (*state.StateDB, tracers.StateReleaseFunc, error)
	// BlockHash returns the hash reported for the traced block.
	BlockHash(*types.Block) common.Hash
}

var (
	_ replayBackend = (*tracerBackend)(nil)
	_ replayBackend = (*suppliedHashBackend)(nil)
)

// TraceBlockByNumber shadows [tracers.API.TraceBlockByNumber] to replay
// transactions with the Flare execution pipeline.
func (a *tracerAPI) TraceBlockByNumber(ctx context.Context, number rpc.BlockNumber, config *tracers.TraceConfig) ([]*tracers.TxTraceResult, error) {
	block, err := a.blockByNumber(ctx, number)
	if err != nil {
		return nil, err
	}
	return a.traceBlockReplay(ctx, a.tracerBackend, block, config)
}

// TraceBlockByHash shadows [tracers.API.TraceBlockByHash] to replay
// transactions with the Flare execution pipeline.
func (a *tracerAPI) TraceBlockByHash(ctx context.Context, hash common.Hash, config *tracers.TraceConfig) ([]*tracers.TxTraceResult, error) {
	block, err := a.blockByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	return a.traceBlockReplay(ctx, a.tracerBackend, block, config)
}

// resealWithExecutedBaseFee returns a caller-supplied block re-sealed with the
// executed base fee, replacing the worst-case bound its header carries. The
// block need not be canonical, but its parent MUST be. Replaying against the
// bound would charge a higher effective base fee, diverging the Flare fee
// settlement, later transactions' interim state and the intermediate roots
// from canonical execution. Shared by [tracerAPI.TraceBlock] (where upstream
// inlines it) and the bad-block path of [tracerAPI.IntermediateRoots].
//
// A synchronous (pre-SAE) block carries the base fee its transactions actually
// paid, so it is returned as supplied.
func (a *tracerAPI) resealWithExecutedBaseFee(ctx context.Context, block *types.Block) (*types.Block, error) {
	hdr := block.Header()
	if hook.Synchronous(a.tracerBackend.Hooks(), hdr) {
		return block, nil
	}
	parent, err := a.tracerBackend.restoreExecutedParent(ctx, block)
	if err != nil {
		return nil, fmt.Errorf("restoring parent block: %w", err)
	}
	// The parent's gas clock, advanced to the start of the block, determines
	// the executed base fee, so the supplied one is discarded. This is the
	// same derivation [saexec.Execute] performs for a freshly built block.
	gasClock := parent.ExecutedByGasTime()
	gasClock.BeforeBlock(a.tracerBackend.Hooks().BlockTime(hdr))
	hdr.BaseFee = gasClock.BaseFee().ToBig()
	return block.WithSeal(hdr), nil
}

// IntermediateRoots shadows [tracers.API.IntermediateRoots] to replay
// transactions with the Flare execution pipeline. Like upstream, a failing
// transaction ends the replay and the roots collected so far are returned
// with a nil error.
//
// A canonical block is served with its executed header and replayed from the
// state [tracerBackend] supplies. A bad block is not: its header carries the
// worst-case base fee, and its start-executing-block changes may differ from
// those of the canonical block at the same height, so it is re-sealed with the
// executed base fee and replayed from the state [suppliedHashBackend] supplies,
// exactly like a debug_traceBlock RLP block.
func (a *tracerAPI) IntermediateRoots(ctx context.Context, hash common.Hash, config *tracers.TraceConfig) ([]common.Hash, error) {
	block, _ := a.blockByHash(ctx, hash)
	isBadBlock := block == nil
	if isBadBlock {
		// Check in the bad blocks
		block = rawdb.ReadBadBlock(a.tracerBackend.ChainDb(), hash)
	}
	if block == nil {
		return nil, fmt.Errorf("block %#x not found", hash)
	}
	if block.NumberU64() == 0 {
		return nil, errors.New("genesis is not traceable")
	}
	var be replayBackend = a.tracerBackend
	if isBadBlock {
		resealed, err := a.resealWithExecutedBaseFee(ctx, block)
		if err != nil {
			return nil, err
		}
		be = &suppliedHashBackend{
			tracerBackend: a.tracerBackend,
			supplied:      block,
			resealed:      resealed,
		}
		block = resealed
	}
	statedb, release, err := a.replayStateAtParent(ctx, be, block, config)
	if err != nil {
		return nil, err
	}
	defer release()

	var (
		roots              []common.Hash
		usedGas            uint64
		header             = block.Header()
		deleteEmptyObjects = a.tracerBackend.ChainConfig().IsEIP158(block.Number())
	)
	for i, tx := range block.Transactions() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := a.applyReplayTx(statedb, header, tx, i, &usedGas, vm.Config{}); err != nil {
			a.tracerBackend.Logger().Warn("Tracing intermediate roots did not complete",
				zap.Int("txindex", i),
				zap.Stringer("txhash", tx.Hash()),
				zap.Error(err),
			)
			// We intentionally don't return the error here: if we do, then the RPC server will not
			// return the roots. Most likely, the caller already knows that a certain transaction fails to
			// be included, but still want the intermediate roots that led to that point.
			// It may happen the tx_N causes an erroneous state, which in turn causes tx_N+M to not be
			// executable.
			// N.B: This should never happen while tracing canon blocks, only when tracing bad blocks.
			return roots, nil
		}
		// [saexec.ApplyTransaction] already finalised the state;
		// IntermediateRoot only computes the root of the pending changes.
		roots = append(roots, statedb.IntermediateRoot(deleteEmptyObjects))
	}
	return roots, nil
}

// traceBlockReplay is the Flare counterpart of libevm's unexported
// API.traceBlock and API.traceBlockParallel: it configures a tracer per
// transaction and replays all of the block's transactions through the
// era-aware [tracerAPI.applyReplayTx], from the base state and with the block
// hash that be supplies.
//
// State advancement is always sequential and era-aware, never upstream's
// parallel path: traceBlockParallel advances the shared state with plain,
// untraced core.ApplyMessage, which would drop the Flare per-transaction state
// changes this shadow exists to preserve. Per-transaction error handling still
// follows the upstream tracer-type contract:
//   - Native/struct-logger tracers trace on (and advance) the shared state; any
//     per-transaction error aborts the whole block (upstream API.traceBlock).
//   - JS tracers record a per-transaction TxTraceResult.Error and continue, so
//     a bad tracer construction, timeout, or result for one transaction does
//     not fail the RPC (upstream API.traceBlockParallel). Their transaction is
//     traced on a copy of the shared state, which is then advanced by an
//     untraced era-aware apply; only ctx cancellation or a genuine execution
//     failure (upstream's feeder ApplyMessage error) aborts the block.
//
// No explicit statedb.Finalise per transaction: [saexec.ApplyTransaction]
// finalises internally on both era paths, mirroring upstream's loop.
func (a *tracerAPI) traceBlockReplay(ctx context.Context, be replayBackend, block *types.Block, config *tracers.TraceConfig) ([]*tracers.TxTraceResult, error) {
	if block.NumberU64() == 0 {
		return nil, errors.New("genesis is not traceable")
	}
	statedb, release, err := a.replayStateAtParent(ctx, be, block, config)
	if err != nil {
		return nil, err
	}
	defer release()

	var (
		txs       = block.Transactions()
		blockHash = be.BlockHash(block)
		header    = block.Header()
		usedGas   uint64
		results   = make([]*tracers.TxTraceResult, len(txs))
	)
	// JS tracers report per-transaction errors rather than aborting the block.
	perTxErrors := config != nil && config.Tracer != nil && *config.Tracer != "" &&
		tracers.DefaultDirectory.IsJS(*config.Tracer)

	for i, tx := range txs {
		txctx := &tracers.Context{
			BlockHash:   blockHash,
			BlockNumber: block.Number(),
			TxIndex:     i,
			TxHash:      tx.Hash(),
		}

		if !perTxErrors {
			res, err := a.traceReplayTx(ctx, txctx, statedb, header, tx, &usedGas, config)
			if err != nil {
				return nil, err
			}
			results[i] = &tracers.TxTraceResult{TxHash: tx.Hash(), Result: res}
			continue
		}

		if err := ctx.Err(); err != nil {
			return nil, err
		}
		// Trace on a copy so a tracer error can neither corrupt the shared state
		// nor abort the block; the throwaway usedGas belongs to the copy.
		res, traceErr := a.traceReplayTx(ctx, txctx, statedb.Copy(), header, tx, new(uint64), config)
		if traceErr != nil {
			results[i] = &tracers.TxTraceResult{TxHash: tx.Hash(), Error: traceErr.Error()}
		} else {
			results[i] = &tracers.TxTraceResult{TxHash: tx.Hash(), Result: res}
		}
		// Advance the shared state through the era-aware pipeline, untraced. A
		// failure here is a genuine execution error and aborts the block.
		if err := a.applyReplayTx(statedb, header, tx, i, &usedGas, vm.Config{}); err != nil {
			return nil, err
		}
	}
	return results, nil
}

// traceReplayTx is the Flare counterpart of libevm's unexported API.traceTx:
// it constructs the tracer and timeout exactly like upstream but executes the
// transaction via [tracerAPI.applyReplayTx] instead of core.ApplyMessage.
//
// On timeout the watchdog matches upstream (eth/tracers/api.go traceTx) as
// closely as the shadow allows: it calls tracer.Stop (JS and native tracers
// abort at the next opportunity) and, via [cancellableTracer], vm.EVM.Cancel on
// the EVM that [saexec.ApplyTransaction] created internally — the abort that
// halts a struct-logger trace's interpreter loop. Cancellation is not
// immediate: like upstream it only takes effect at the interpreter's next
// abort check, so a timed-out trace still runs to that point.
func (a *tracerAPI) traceReplayTx(ctx context.Context, txctx *tracers.Context, statedb *state.StateDB, header *types.Header, tx *types.Transaction, usedGas *uint64, config *tracers.TraceConfig) (any, error) {
	var (
		tracer  tracers.Tracer
		err     error
		timeout = defaultTraceTimeout
	)
	if config == nil {
		config = &tracers.TraceConfig{}
	}
	// Default tracer is the struct logger
	tracer = logger.NewStructLogger(config.Config)
	if config.Tracer != nil {
		tracer, err = tracers.DefaultDirectory.New(*config.Tracer, txctx, config.TracerConfig)
		if err != nil {
			return nil, err
		}
	}
	// Wrap so the watchdog can reach the EVM that saexec.ApplyTransaction
	// creates internally and cancel it, replicating upstream's vmenv.Cancel().
	ctracer := &cancellableTracer{Tracer: tracer}

	// Define a meaningful timeout of a single transaction trace
	if config.Timeout != nil {
		if timeout, err = time.ParseDuration(*config.Timeout); err != nil {
			return nil, err
		}
	}
	deadlineCtx, cancel := context.WithTimeout(ctx, timeout)
	go func() {
		<-deadlineCtx.Done()
		if errors.Is(deadlineCtx.Err(), context.DeadlineExceeded) {
			ctracer.Stop(errors.New("execution timeout"))
			// Stop the EVM execution. Note cancellation is not necessarily immediate.
			ctracer.cancel()
		}
	}()
	defer cancel()

	if err := a.applyReplayTx(statedb, header, tx, txctx.TxIndex, usedGas, vm.Config{Tracer: ctracer, NoBaseFee: true}); err != nil {
		return nil, fmt.Errorf("tracing failed: %w", err)
	}
	return ctracer.GetResult()
}

// cancellableTracer wraps a tracer so the trace-timeout watchdog can abort the
// EVM created inside [saexec.ApplyTransaction], which is otherwise out of the
// shadow's reach (unlike upstream's traceTx, which holds the vm.EVM directly
// and calls vmenv.Cancel). The EVM hands itself to the tracer as the top call
// frame's CaptureStart argument; capturing it there lets [cancellableTracer.cancel]
// stand in for the missing vmenv.Cancel. All other tracer behaviour, including
// GetResult, is delegated to the wrapped tracer unchanged.
//
// cancel and CaptureStart race whenever the deadline fires before execution
// reaches CaptureStart (e.g. a "0s" or tiny timeout). The cancelled flag closes
// that race: cancel records it before loading the EVM, and CaptureStart cancels
// the EVM it just stored when the flag is already set. Because both sides store
// their field before loading the other's, sequentially-consistent atomics
// guarantee at least one observes the other — so a cancel is never lost, and a
// high-gas transaction cannot slip past the deadline uncancelled.
type cancellableTracer struct {
	tracers.Tracer
	evm       atomic.Pointer[vm.EVM]
	cancelled atomic.Bool
}

// CaptureStart records the executing EVM, honours a cancel that already fired,
// then delegates. It fires once per transaction (top call frame only; nested
// frames use CaptureEnter), so the stored pointer is the transaction's root
// EVM — the one whose Cancel aborts the interpreter loop.
func (t *cancellableTracer) CaptureStart(env *vm.EVM, from common.Address, to common.Address, create bool, input []byte, gas uint64, value *big.Int) {
	t.evm.Store(env)
	if t.cancelled.Load() {
		env.Cancel()
	}
	t.Tracer.CaptureStart(env, from, to, create, input, gas, value)
}

// cancel aborts the traced EVM. It records the cancellation before loading the
// EVM so a CaptureStart that has not yet run (the watchdog fired first) cancels
// the EVM itself; without the flag, such a cancel would be a permanent no-op
// and the transaction would run to completion despite the timeout.
func (t *cancellableTracer) cancel() {
	t.cancelled.Store(true)
	if evm := t.evm.Load(); evm != nil {
		evm.Cancel()
	}
}

// applyReplayTx executes one transaction on statedb through the era-splitting
// [saexec.ApplyTransaction], with cfg carrying the tracer when tracing.
//
// The explicit &header.Coinbase author matches both upstream (author=nil
// resolves via [coinbaseAsAuthor] to header.Coinbase) and [saexec.Execute].
// The per-transaction gas pool matches upstream's sizing; both era paths only
// ever draw at most tx.Gas() from it. The returned receipt is discarded.
func (a *tracerAPI) applyReplayTx(statedb *state.StateDB, header *types.Header, tx *types.Transaction, index int, usedGas *uint64, cfg vm.Config) error {
	statedb.SetTxContext(tx.Hash(), index)
	gp := new(core.GasPool).AddGas(tx.Gas())
	_, err := saexec.ApplyTransaction(a.tracerBackend.ChainConfig(), a.tracerBackend.ChainContext(), &header.Coinbase, gp, statedb, header, tx, usedGas, cfg)
	return err
}

// replayStateAtParent mirrors the preamble shared by libevm's traceBlock and
// IntermediateRoots: it resolves the (canonical) parent and returns the replay
// base state served by be — the parent's post-execution state with the traced
// block's start-executing-block changes (EIP-4788 beacon root and the
// StartExecutingBlock hook), applied by [saexec.Execute] exactly as for live
// execution. [tracerBackend] keys those changes off the canonical child, which
// is the block the by-number and by-hash endpoints trace; [suppliedHashBackend]
// keys them off the caller-supplied block, whose header (timestamp,
// parent-beacon-root) may differ from the canonical block's at that height.
func (a *tracerAPI) replayStateAtParent(ctx context.Context, be replayBackend, block *types.Block, config *tracers.TraceConfig) (*state.StateDB, tracers.StateReleaseFunc, error) {
	parent, err := a.blockByNumberAndHash(ctx, rpc.BlockNumber(block.NumberU64()-1), block.ParentHash()) // #nosec G115 -- won't overflow for a while.
	if err != nil {
		return nil, nil, err
	}
	reexec := defaultTraceReexec
	if config != nil && config.Reexec != nil {
		reexec = *config.Reexec
	}
	return be.StateAtBlock(ctx, parent, reexec, nil, true, false)
}

// blockByNumber mirrors libevm's unexported API.blockByNumber over
// [tracerBackend], including the error strings.
func (a *tracerAPI) blockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	block, err := a.tracerBackend.BlockByNumber(ctx, number)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, fmt.Errorf("block #%d not found", number)
	}
	return block, nil
}

// blockByHash mirrors libevm's unexported API.blockByHash over
// [tracerBackend], including the error strings.
func (a *tracerAPI) blockByHash(ctx context.Context, hash common.Hash) (*types.Block, error) {
	block, err := a.tracerBackend.BlockByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	if block == nil {
		return nil, fmt.Errorf("block %s not found", hash.Hex())
	}
	return block, nil
}

// blockByNumberAndHash mirrors libevm's unexported API.blockByNumberAndHash:
// resolve by number first and fall back to the hash lookup when the canonical
// hash at that height differs.
func (a *tracerAPI) blockByNumberAndHash(ctx context.Context, number rpc.BlockNumber, hash common.Hash) (*types.Block, error) {
	block, err := a.blockByNumber(ctx, number)
	if err != nil {
		return nil, err
	}
	if a.tracerBackend.BlockHash(block) == hash {
		return block, nil
	}
	return a.blockByHash(ctx, hash)
}
