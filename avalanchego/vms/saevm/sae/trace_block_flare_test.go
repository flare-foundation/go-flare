// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package sae

import (
	"encoding/json"
	"math/big"
	"sync"
	"testing"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/common/hexutil"
	"github.com/ava-labs/libevm/core/rawdb"
	"github.com/ava-labs/libevm/core/state"
	"github.com/ava-labs/libevm/core/types"
	"github.com/ava-labs/libevm/eth/tracers"
	"github.com/ava-labs/libevm/params"
	"github.com/ava-labs/libevm/rlp"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestDebugTraceBlockRLPBeforeBlockUsesSuppliedBlock pins that debug_traceBlock
// applies the before-block changes (EIP-4788 beacon root and the Flare
// before-block hooks) of the *supplied* block, not the canonical child at the
// same height. The RLP endpoint can be handed a non-canonical block, and its
// before-block state must key off that block's header, matching live execution.
//
// It fails against the previous implementation, which sourced the before-block
// changes from tracerBackend.StateAtBlock's canonical-child hook.
func TestDebugTraceBlockRLPBeforeBlockUsesSuppliedBlock(t *testing.T) {
	ctx, sut := newSUT(t, 1)

	// Record the timestamp of every block whose before-block hook runs. The
	// supplied block below is given a unique timestamp no real block has, so a
	// hook call at that timestamp can only come from tracing it.
	var (
		mu   sync.Mutex
		seen = map[uint64]common.Hash{}
	)
	sut.hooks.StartExecutingBlockFn = func(_ params.Rules, _ *state.StateDB, _ *types.Header, block *types.Block) error {
		mu.Lock()
		seen[block.Time()] = block.Hash()
		mu.Unlock()
		return nil
	}

	// Canonical block 1 (child of genesis). Wait for execution so its own
	// before-block hook call has settled.
	b1 := sut.runConsensusLoop(t)
	require.NoErrorf(t, b1.WaitUntilExecuted(ctx), "%T.WaitUntilExecuted()", b1)

	// A non-canonical variant of block 1: same parent (genesis) and body, but a
	// later timestamp, so it is NOT the canonical block at height 1.
	hdr := b1.Header()
	hdr.Time++
	supplied := b1.EthBlock().WithSeal(hdr)
	require.NotEqual(t, b1.Hash(), supplied.Hash(), "supplied block must be non-canonical")

	blockRLP, err := rlp.EncodeToBytes(supplied)
	require.NoErrorf(t, err, "rlp.EncodeToBytes(%T)", supplied)

	var out []json.RawMessage
	require.NoError(t, sut.CallContext(ctx, &out, "debug_traceBlock", hexutil.Bytes(blockRLP)), "CallContext(debug_traceBlock)")

	mu.Lock()
	gotHash, ran := seen[supplied.Time()]
	mu.Unlock()
	require.True(t, ran, "before-block hook must run for the supplied block's timestamp")
	assert.Equal(t, supplied.Hash(), gotHash, "before-block hook must key off the supplied block, not the canonical child")
}

// TestDebugTraceBlockJSTracerPerTxErrors pins the upstream traceBlockParallel
// contract for JS tracers: a per-transaction tracer error is reported in that
// transaction's TxTraceResult.Error and the block trace continues, rather than
// surfacing as a top-level RPC error. The Flare shadow keeps state advancement
// sequential and era-aware, but must still preserve these partial-result
// semantics.
//
// It fails against the previous serial loop, which returned the first JS result
// error as a top-level RPC error and aborted the whole block.
func TestDebugTraceBlockJSTracerPerTxErrors(t *testing.T) {
	ctx, sut := newSUT(t, 1)

	// Both from account 0 (nonces auto-increment) so they land in one block.
	txs := []*types.Transaction{
		sut.wallet.SetNonceAndSign(t, 0, &types.LegacyTx{To: &common.Address{'a'}, Gas: params.TxGas, GasPrice: big.NewInt(1), Value: big.NewInt(1)}),
		sut.wallet.SetNonceAndSign(t, 0, &types.LegacyTx{To: &common.Address{'b'}, Gas: params.TxGas, GasPrice: big.NewInt(1), Value: big.NewInt(1)}),
	}
	block := sut.runConsensusLoop(t, txs...)
	require.Lenf(t, block.Transactions(), len(txs), "%T.Transactions()", block)

	// A JS tracer that throws for the first transaction only. If tracing were
	// aborted on the first error, the second transaction's result would never
	// be produced.
	jsTracer := `{
		fault: function() {},
		result: function(ctx) {
			if (ctx.txIndex == 0) { throw new Error('boom at tx 0'); }
			return ctx.txIndex;
		}
	}`

	var results []struct {
		TxHash common.Hash     `json:"txHash"`
		Result json.RawMessage `json:"result"`
		Error  string          `json:"error"`
	}
	require.NoError(t,
		sut.CallContext(ctx, &results, "debug_traceBlockByNumber",
			hexutil.Uint64(block.NumberU64()),
			tracers.TraceConfig{Tracer: &jsTracer},
		),
		"debug_traceBlockByNumber must not fail the RPC on a per-transaction JS tracer error",
	)

	require.Len(t, results, len(txs), "one result per transaction")
	assert.Contains(t, results[0].Error, "boom at tx 0", "tx 0's tracer error is reported per-transaction")
	assert.Empty(t, results[0].Result, "tx 0 has no result")
	assert.Empty(t, results[1].Error, "tx 1 traced successfully after tx 0's error")
	assert.Equal(t, "1", string(results[1].Result), "tx 1's tracer ran and returned its index")
}

// bogusBaseFee is a base fee far above any transaction's fee cap used in these
// tests; replaying against it rejects the transaction with ErrFeeCapTooLow.
func bogusBaseFee() *big.Int { return new(big.Int).Exp(big.NewInt(10), big.NewInt(18), nil) }

// TestDebugTraceBlockNonCanonicalDerivesBaseFee pins that a non-canonical
// debug_traceBlock RLP block is replayed against the executed base fee derived
// from its parent, not the worst-case bound carried in its header. The variant
// below carries a bogus header base fee above the transaction's fee cap: with
// the fix the derived base fee (the minimum) lets the transaction trace; without
// it the supplied base fee rejects the transaction and fails the whole trace.
func TestDebugTraceBlockNonCanonicalDerivesBaseFee(t *testing.T) {
	ctx, sut := newSUT(t, 1)

	tx := sut.wallet.SetNonceAndSign(t, 0, &types.LegacyTx{
		To: &common.Address{'x'}, Gas: params.TxGas, GasPrice: big.NewInt(1), Value: big.NewInt(1),
	})
	block := sut.runConsensusLoop(t, tx)
	require.NoErrorf(t, block.WaitUntilExecuted(ctx), "%T.WaitUntilExecuted()", block)

	// Same parent, timestamp and body; only the header base fee differs, so the
	// block is non-canonical (no stored artifact) and takes the derive path.
	hdr := block.EthBlock().Header()
	hdr.BaseFee = bogusBaseFee()
	nonCanonical := block.EthBlock().WithSeal(hdr)
	require.NotEqual(t, block.Hash(), nonCanonical.Hash(), "variant must be non-canonical")

	blockRLP, err := rlp.EncodeToBytes(nonCanonical)
	require.NoErrorf(t, err, "rlp.EncodeToBytes(%T)", nonCanonical)

	var results []struct {
		TxHash common.Hash     `json:"txHash"`
		Result json.RawMessage `json:"result"`
		Error  string          `json:"error"`
	}
	require.NoError(t, sut.CallContext(ctx, &results, "debug_traceBlock", hexutil.Bytes(blockRLP)),
		"debug_traceBlock must derive the executed base fee for a non-canonical block")
	require.Len(t, results, 1, "one result for the single transaction")
	assert.Empty(t, results[0].Error, "the transaction traces at the derived base fee")
	assert.NotEmpty(t, results[0].Result, "a struct-log result is produced")
}

// TestIntermediateRootsBadBlockDerivesBaseFee pins the same normalisation for
// the debug_intermediateRoots bad-block path, which reads the block via
// ReadBadBlock and previously bypassed base-fee normalisation entirely. Without
// the fix the bogus header base fee rejects the transaction and no root is
// produced; with it the derived base fee yields one root.
func TestIntermediateRootsBadBlockDerivesBaseFee(t *testing.T) {
	ctx, sut := newSUT(t, 1)

	tx := sut.wallet.SetNonceAndSign(t, 0, &types.LegacyTx{
		To: &common.Address{'y'}, Gas: params.TxGas, GasPrice: big.NewInt(1), Value: big.NewInt(1),
	})
	block := sut.runConsensusLoop(t, tx)
	require.NoErrorf(t, block.WaitUntilExecuted(ctx), "%T.WaitUntilExecuted()", block)

	hdr := block.EthBlock().Header()
	hdr.BaseFee = bogusBaseFee()
	badBlock := block.EthBlock().WithSeal(hdr)
	rawdb.WriteBadBlock(sut.db, badBlock)

	var roots []common.Hash
	require.NoError(t, sut.CallContext(ctx, &roots, "debug_intermediateRoots", badBlock.Hash()),
		"debug_intermediateRoots(bad block)")
	require.Len(t, roots, 1, "one intermediate root per successfully replayed transaction")
}
