// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package rpc

import (
	"context"
	"errors"
	"fmt"
	"math/big"
	"os"
	"sync/atomic"
	"time"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/common/hexutil"
	"github.com/ava-labs/libevm/core"
	"github.com/ava-labs/libevm/core/rawdb"
	"github.com/ava-labs/libevm/core/state"
	"github.com/ava-labs/libevm/core/types"
	"github.com/ava-labs/libevm/core/vm"
	"github.com/ava-labs/libevm/eth/tracers"
	"github.com/ava-labs/libevm/eth/tracers/logger"
	"github.com/ava-labs/libevm/rlp"
	"github.com/ava-labs/libevm/rpc"
	"go.uber.org/zap"

	"github.com/ava-labs/avalanchego/vms/saevm/saexec"
)

// This file shadows libevm's block-level tracing endpoints on [tracerAPI] so
// that every transaction is replayed through [saexec.ApplyTransaction] — the
// same era-splitting pipeline used by live execution and by
// [backend.StateAtTransaction] — instead of libevm's plain core.ApplyMessage.
// On Flare/Songbird chains the plain replay misses the Flare per-transaction
// mechanisms (fee burn/refund, daemon minting, and, pre-Helicon, coreth's
// full StateTransition including the one-time transition-contract calls), so
// traces and intermediate roots would diverge from the canonical state.
//
// Everything else mirrors libevm's eth/tracers/api.go behavior byte-for-byte
// (error strings, tracer construction, timeouts, partial-result semantics).
// Check for differences in the upstream implementation when updating libevm.
//
// Not shadowed (still libevm's plain-ApplyMessage replay): debug_traceChain,
// debug_standardTraceBlockToFile, debug_traceBadBlock, and
// debug_standardTraceBadBlockToFile.

// Mirrors of libevm eth/tracers unexported defaults.
const (
	defaultTraceTimeout = 5 * time.Second
	defaultTraceReexec  = uint64(128) // resolved for parity; [backend.StateAtBlock] ignores it
)

// TraceBlockByNumber shadows [tracers.API.TraceBlockByNumber] to replay
// transactions with the Flare execution pipeline.
func (a *tracerAPI) TraceBlockByNumber(ctx context.Context, number rpc.BlockNumber, config *tracers.TraceConfig) ([]*tracers.TxTraceResult, error) {
	block, err := a.blockByNumber(ctx, number)
	if err != nil {
		return nil, err
	}
	return a.traceBlockReplay(ctx, block, config)
}

// TraceBlockByHash shadows [tracers.API.TraceBlockByHash] to replay
// transactions with the Flare execution pipeline.
func (a *tracerAPI) TraceBlockByHash(ctx context.Context, hash common.Hash, config *tracers.TraceConfig) ([]*tracers.TxTraceResult, error) {
	block, err := a.blockByHash(ctx, hash)
	if err != nil {
		return nil, err
	}
	return a.traceBlockReplay(ctx, block, config)
}

// TraceBlock shadows [tracers.API.TraceBlock] to replay transactions with the
// Flare execution pipeline. The block's base fee is normalised to the executed
// value during shared replay setup ([replayStateAtParent] →
// [executedBaseFeeBlock]).
func (a *tracerAPI) TraceBlock(ctx context.Context, blob hexutil.Bytes, config *tracers.TraceConfig) ([]*tracers.TxTraceResult, error) {
	block := new(types.Block)
	if err := rlp.DecodeBytes(blob, block); err != nil {
		return nil, fmt.Errorf("could not decode block: %v", err)
	}
	return a.traceBlockReplay(ctx, block, config)
}

// executedBaseFeeBlock returns block re-sealed so its header carries the base
// fee its transactions actually pay (the executed base fee) rather than SAE's
// worst-case pre-execution bound. Replaying against the bound charges a higher
// effective base fee, diverging the Flare fee settlement, later transactions'
// interim state, and the intermediate roots from canonical execution.
//
// The correct base fee comes from one of two sources:
//   - the stored artifact of the canonical block at this height (authoritative,
//     and already applied for the by-number/by-hash endpoints); or
//   - otherwise, deterministically derived from the parent's executed gas clock
//     and the supplied block's timestamp — the same value [saexec.Execute]
//     computes for a freshly built block. This covers non-canonical blocks
//     (a debug_traceBlock RLP block, an IntermediateRoots bad block).
//
// Pre-Helicon coreth blocks are replayed through the legacy path, which uses
// the header base fee directly, so it is already the executed value and the
// block is returned unchanged.
func (a *tracerAPI) executedBaseFeeBlock(ctx context.Context, block *types.Block) *types.Block {
	if saexec.IsLegacyCorethBlock(a.b.ChainConfig(), block.Time()) {
		return block
	}
	num := rpc.BlockNumber(block.NumberU64()) // #nosec G115 -- won't overflow for a while.
	if bl, err := a.b.restoreExecutedBlock(ctx, rpc.BlockNumberOrHashWithNumber(num)); err == nil {
		// The canonical block's hash is over its raw consensus header; the
		// by-number/by-hash endpoints instead serve it re-sealed with the
		// executed header. Match either form and prefer the stored artifact.
		executed := block.WithSeal(executedHeader(bl))
		switch block.Hash() {
		case bl.Hash():
			return executed // raw canonical (e.g. debug_traceBlock RLP): re-seal.
		case executed.Hash():
			return block // already carries the executed header.
		}
	}
	// Non-canonical block: derive from the parent's executed gas clock.
	parent, err := a.b.restoreExecutedBlock(ctx, rpc.BlockNumberOrHashWithHash(block.ParentHash(), true /* canonical */))
	if err != nil {
		return block // parent unavailable: best-effort, keep the supplied header.
	}
	hdr := block.Header()
	hdr.BaseFee = saexec.DeriveExecutedBaseFee(parent, a.b.Hooks(), hdr)
	return block.WithSeal(hdr)
}

// TraceBlockFromFile shadows [tracers.API.TraceBlockFromFile] to replay
// transactions with the Flare execution pipeline.
func (a *tracerAPI) TraceBlockFromFile(ctx context.Context, file string, config *tracers.TraceConfig) ([]*tracers.TxTraceResult, error) {
	blob, err := os.ReadFile(file)
	if err != nil {
		return nil, fmt.Errorf("could not read file: %v", err)
	}
	return a.TraceBlock(ctx, blob, config)
}

// IntermediateRoots shadows [tracers.API.IntermediateRoots] to replay
// transactions with the Flare execution pipeline. Like upstream, a failing
// transaction ends the replay and the roots collected so far are returned
// with a nil error.
func (a *tracerAPI) IntermediateRoots(ctx context.Context, hash common.Hash, config *tracers.TraceConfig) ([]common.Hash, error) {
	block, _ := a.blockByHash(ctx, hash)
	if block == nil {
		// Check in the bad blocks
		block = rawdb.ReadBadBlock(a.b.ChainDb(), hash)
	}
	if block == nil {
		return nil, fmt.Errorf("block %#x not found", hash)
	}
	block, statedb, release, err := a.replayStateAtParent(ctx, block, config)
	if err != nil {
		return nil, err
	}
	defer release()

	var (
		roots              []common.Hash
		usedGas            uint64
		header             = block.Header()
		deleteEmptyObjects = a.b.ChainConfig().IsEIP158(block.Number())
	)
	for i, tx := range block.Transactions() {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		if err := a.applyReplayTx(statedb, header, tx, i, &usedGas, vm.Config{}); err != nil {
			a.b.Logger().Warn("Tracing intermediate roots did not complete",
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
// era-aware [tracerAPI.applyReplayTx].
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
func (a *tracerAPI) traceBlockReplay(ctx context.Context, block *types.Block, config *tracers.TraceConfig) ([]*tracers.TxTraceResult, error) {
	block, statedb, release, err := a.replayStateAtParent(ctx, block, config)
	if err != nil {
		return nil, err
	}
	defer release()

	var (
		txs       = block.Transactions()
		blockHash = a.b.BlockHash(block)
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
	_, err := saexec.ApplyTransaction(a.b.ChainConfig(), a.b.ChainContext(), &header.Coinbase, gp, statedb, header, tx, usedGas, cfg)
	return err
}

// replayStateAtParent mirrors the preamble shared by libevm's traceBlock and
// IntermediateRoots: it rejects the genesis block, resolves the parent, and
// returns the replay base state — the parent's post-execution state with
// block's own before-block changes (EIP-4788 beacon root and the Flare
// before-block hooks) applied.
//
// It normalises the block's base fee to the executed value (via
// [executedBaseFeeBlock]) and returns the resulting block, so canonical and
// non-canonical SAE blocks alike replay against the base fee their transactions
// actually pay rather than SAE's worst-case bound.
//
// It also deliberately bypasses [tracerBackend.StateAtBlock], whose before-block
// hook applies the changes of the *canonical* child at parent+1. That is only
// correct when the traced block IS that canonical child (the by-number and
// by-hash endpoints). For an arbitrary debug_traceBlock RLP block or an
// IntermediateRoots bad block, the supplied block may differ from the canonical
// child — with a different timestamp or parent-beacon-root — so the before-block
// changes must key off the supplied block. This applies them via
// [saexec.BeforeExecutingBlock] on the parent's clean post-execution state,
// exactly as [saexec.Execute] does for live execution.
func (a *tracerAPI) replayStateAtParent(ctx context.Context, block *types.Block, config *tracers.TraceConfig) (*types.Block, *state.StateDB, tracers.StateReleaseFunc, error) {
	if block.NumberU64() == 0 {
		return nil, nil, nil, errors.New("genesis is not traceable")
	}
	parent, err := a.blockByNumberAndHash(ctx, rpc.BlockNumber(block.NumberU64()-1), block.ParentHash()) // #nosec G115 -- won't overflow for a while.
	if err != nil {
		return nil, nil, nil, err
	}
	reexec := defaultTraceReexec
	if config != nil && config.Reexec != nil {
		reexec = *config.Reexec
	}
	// a.b.backend.StateAtBlock is the base method WITHOUT tracerBackend's
	// canonical-child before-block hook (see the doc comment).
	sdb, release, err := a.b.backend.StateAtBlock(ctx, parent, reexec, nil, true, false)
	if err != nil {
		return nil, nil, nil, err
	}
	block = a.executedBaseFeeBlock(ctx, block)
	rules := a.b.ChainConfig().Rules(block.Number(), true /*isMerge*/, block.Time())
	if err := saexec.BeforeExecutingBlock(a.b.Hooks(), rules, sdb, parent.Header(), block); err != nil {
		release()
		return nil, nil, nil, err
	}
	return block, sdb, release, nil
}

// blockByNumber mirrors libevm's unexported API.blockByNumber over
// [tracerBackend], including the error strings.
func (a *tracerAPI) blockByNumber(ctx context.Context, number rpc.BlockNumber) (*types.Block, error) {
	block, err := a.b.BlockByNumber(ctx, number)
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
	block, err := a.b.BlockByHash(ctx, hash)
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
	if a.b.BlockHash(block) == hash {
		return block, nil
	}
	return a.blockByHash(ctx, hash)
}
