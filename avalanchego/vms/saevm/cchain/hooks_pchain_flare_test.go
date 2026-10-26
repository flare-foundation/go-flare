// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package cchain

import (
	"context"
	"errors"
	"math/big"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ava-labs/libevm/common/hexutil"
	"github.com/ava-labs/libevm/core/types"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/graft/coreth/plugin/evm/customtypes"
	"github.com/ava-labs/avalanchego/snow/engine/snowman/block"
	"github.com/ava-labs/avalanchego/snow/pchain"
	"github.com/ava-labs/avalanchego/snow/snowtest"
	"github.com/ava-labs/avalanchego/vms/saevm/cchain/tx/txtest"
	"github.com/ava-labs/avalanchego/vms/saevm/hook"
)

var (
	_ pchain.Reader      = (*fakePChainReader)(nil)
	_ hook.ExecutionGate = (*hooks)(nil)
)

// fakePChainReader only serves LastAcceptedHeight, which is all the execution
// gate needs.
type fakePChainReader struct {
	height atomic.Uint64
	err    atomic.Pointer[error]
}

func (r *fakePChainReader) LastAcceptedHeight() (uint64, error) {
	if err := r.err.Load(); err != nil {
		return 0, *err
	}
	return r.height.Load(), nil
}

func (*fakePChainReader) GetBlock(uint64, pchain.ByteCharger) (pchain.BlockReader, error) {
	return nil, errors.New("not implemented")
}

// TestBuildBlockCommitsPChainHeight asserts that the builder commits the block
// context's P-chain height into the header extra, and leaves it unset without
// a context.
func TestBuildBlockCommitsPChainHeight(t *testing.T) {
	key := txtest.NewKey(t)
	_, sut := newSUT(t, withMaxAllocFor(key.EthAddress()))
	hooks := sut.hooks(t)

	stx := newWallet(key, sut.ctx, sut.Client).newMinimalTx(t)
	htx, err := newHookTx(stx, sut.ctx.AVAXAssetID)
	require.NoError(t, err, "newHookTx()")

	built := func(t *testing.T, blockCtx *block.Context) *customtypes.HeaderExtra {
		t.Helper()

		block, err := hooks.BuildBlock(
			&types.Header{},
			blockCtx,
			nil, // ethTxs
			nil, // receipts
			[]*hookTx{htx},
			hook.Settled{},
		)
		require.NoError(t, err, "builder.BuildBlock()")
		return customtypes.GetHeaderExtra(block.Header())
	}

	t.Run("with_context", func(t *testing.T) {
		extra := built(t, &block.Context{ProposerPChainHeight: 7})
		require.NotNilf(t, extra.PChainHeight, "%T.PChainHeight", extra)
		require.Equalf(t, uint64(7), *extra.PChainHeight, "%T.PChainHeight", extra)
	})

	// The epoch height (block.Context.PChainHeight) is what warp verifies
	// against; the header must carry the ProposerVM block's own height.
	t.Run("epoch_height_is_not_committed", func(t *testing.T) {
		extra := built(t, &block.Context{PChainHeight: 3, ProposerPChainHeight: 7})
		require.NotNilf(t, extra.PChainHeight, "%T.PChainHeight", extra)
		require.Equalf(t, uint64(7), *extra.PChainHeight, "%T.PChainHeight", extra)
	})

	t.Run("nil_context", func(t *testing.T) {
		extra := built(t, nil)
		require.Nilf(t, extra.PChainHeight, "%T.PChainHeight", extra)
	})
}

func blockWithPChainHeight(height *uint64) *types.Block {
	return types.NewBlockWithHeader(customtypes.WithHeaderExtra(
		&types.Header{Number: big.NewInt(1)},
		&customtypes.HeaderExtra{
			SettledHeight:       new(uint64(0)),
			SettledGasUnix:      new(uint64(0)),
			SettledGasNumerator: new(uint64(0)),
			SettledExcess:       new(uint64(0)),
			PChainHeight:        height,
		},
	))
}

// TestAwaitExecutionPrerequisites covers the execution gate: no committed
// height or no reader returns immediately; otherwise the gate polls the local
// P-chain height until it reaches the committed one or the context is
// cancelled.
func TestAwaitExecutionPrerequisites(t *testing.T) {
	origPoll := pChainHeightPollInterval
	pChainHeightPollInterval = time.Millisecond
	t.Cleanup(func() { pChainHeightPollInterval = origPoll })

	newGate := func(reader pchain.Reader) *hooks {
		snowCtx := snowtest.Context(t, snowtest.CChainID)
		snowCtx.PChainReader = reader
		return &hooks{builder: builder{ctx: snowCtx}}
	}

	// await runs the gate in the background and returns a channel carrying its
	// result.
	await := func(ctx context.Context, h *hooks, b *types.Block) <-chan error {
		done := make(chan error, 1)
		go func() {
			done <- h.AwaitExecutionPrerequisites(ctx, b)
		}()
		return done
	}

	requireBlocked := func(t *testing.T, done <-chan error) {
		t.Helper()
		select {
		case err := <-done:
			t.Fatalf("AwaitExecutionPrerequisites() returned early with %v", err)
		case <-time.After(20 * time.Millisecond):
		}
	}
	requireReturned := func(t *testing.T, done <-chan error) error {
		t.Helper()
		select {
		case err := <-done:
			return err
		case <-time.After(5 * time.Second):
			t.Fatal("AwaitExecutionPrerequisites() did not return")
			return nil
		}
	}

	t.Run("no_committed_height", func(t *testing.T) {
		reader := &fakePChainReader{}
		gate := newGate(reader)
		require.NoError(t, gate.AwaitExecutionPrerequisites(t.Context(), blockWithPChainHeight(nil)))

		// Even without a reader.
		gate = newGate(nil)
		require.NoError(t, gate.AwaitExecutionPrerequisites(t.Context(), blockWithPChainHeight(nil)))
	})

	t.Run("no_reader", func(t *testing.T) {
		gate := newGate(nil)
		require.NoError(t, gate.AwaitExecutionPrerequisites(t.Context(), blockWithPChainHeight(new(uint64(7)))))
	})

	t.Run("already_reached", func(t *testing.T) {
		reader := &fakePChainReader{}
		reader.height.Store(7)
		gate := newGate(reader)
		require.NoError(t, gate.AwaitExecutionPrerequisites(t.Context(), blockWithPChainHeight(new(uint64(7)))))
	})

	t.Run("waits_until_reached", func(t *testing.T) {
		reader := &fakePChainReader{}
		reader.height.Store(5)
		gate := newGate(reader)

		done := await(t.Context(), gate, blockWithPChainHeight(new(uint64(7))))
		requireBlocked(t, done)

		reader.height.Store(6)
		requireBlocked(t, done)

		reader.height.Store(7)
		require.NoError(t, requireReturned(t, done))
	})

	t.Run("retries_reader_errors", func(t *testing.T) {
		reader := &fakePChainReader{}
		readErr := errors.New("boom")
		reader.err.Store(&readErr)
		reader.height.Store(9)
		gate := newGate(reader)

		done := await(t.Context(), gate, blockWithPChainHeight(new(uint64(7))))
		requireBlocked(t, done)

		reader.err.Store(nil)
		require.NoError(t, requireReturned(t, done))
	})

	t.Run("cancelled", func(t *testing.T) {
		reader := &fakePChainReader{}
		gate := newGate(reader)

		ctx, cancel := context.WithCancel(t.Context())
		done := await(ctx, gate, blockWithPChainHeight(new(uint64(7))))
		requireBlocked(t, done)

		cancel()
		require.ErrorIs(t, requireReturned(t, done), context.Canceled)
	})
}

// TestVerifyBlockPChainHeight asserts that the committed P-chain height is
// enforced by rebuild-and-compare: a block built under one context is
// rejected when tampered or when verified under a different (or no) context.
func TestVerifyBlockPChainHeight(t *testing.T) {
	key := txtest.NewKey(t)
	ctx, sut := newSUT(t, withMaxAllocFor(key.EthAddress()))

	stx := newWallet(key, sut.ctx, sut.Client).newMinimalTx(t)
	require.NoErrorf(t, sut.IssueTx(ctx, stx), "%T.IssueTx()", sut.Client)

	const committed = 7
	bCtx := &block.Context{ProposerPChainHeight: committed}
	valid := sut.buildVerify(ctx, t, sut.lastAccepted(ctx, t), withBlockContext(bCtx))

	extra := customtypes.GetHeaderExtra(valid.Header())
	require.NotNilf(t, extra.PChainHeight, "%T.PChainHeight", extra)
	require.Equalf(t, uint64(committed), *extra.PChainHeight, "%T.PChainHeight", extra)

	t.Run("tampered_height", func(t *testing.T) {
		err := sut.verifyTampered(ctx, t, valid, tamperExtra(func(e *customtypes.HeaderExtra) {
			e.PChainHeight = new(uint64(committed + 1))
		}), withBlockContext(bCtx))
		require.ErrorContainsf(t, err, "hash mismatch", "%T.VerifyBlock(tampered block)", sut.VM)
	})

	t.Run("removed_height", func(t *testing.T) {
		err := sut.verifyTampered(ctx, t, valid, tamperExtra(func(e *customtypes.HeaderExtra) {
			e.PChainHeight = nil
		}), withBlockContext(bCtx))
		require.ErrorContainsf(t, err, "hash mismatch", "%T.VerifyBlock(tampered block)", sut.VM)
	})

	t.Run("different_context", func(t *testing.T) {
		parsed, err := sut.ParseBlock(ctx, valid.Bytes())
		require.NoErrorf(t, err, "%T.ParseBlock()", sut.VM)
		err = sut.VerifyBlock(ctx, &block.Context{ProposerPChainHeight: committed + 1}, parsed)
		require.ErrorContainsf(t, err, "hash mismatch", "%T.VerifyBlock(other context)", sut.VM)
	})

	t.Run("nil_context", func(t *testing.T) {
		parsed, err := sut.ParseBlock(ctx, valid.Bytes())
		require.NoErrorf(t, err, "%T.ParseBlock()", sut.VM)
		err = sut.VerifyBlock(ctx, nil, parsed)
		require.ErrorContainsf(t, err, "hash mismatch", "%T.VerifyBlock(nil context)", sut.VM)
	})
}

// TestBuildBlockWithoutContextHasNoPChainHeight asserts that a block built
// without a block context carries no P-chain height, verifies without one, and
// is rejected under a context.
func TestBuildBlockWithoutContextHasNoPChainHeight(t *testing.T) {
	key := txtest.NewKey(t)
	ctx, sut := newSUT(t, withMaxAllocFor(key.EthAddress()))

	stx := newWallet(key, sut.ctx, sut.Client).newMinimalTx(t)
	require.NoErrorf(t, sut.IssueTx(ctx, stx), "%T.IssueTx()", sut.Client)

	valid := sut.buildVerify(ctx, t, sut.lastAccepted(ctx, t))
	extra := customtypes.GetHeaderExtra(valid.Header())
	require.Nilf(t, extra.PChainHeight, "%T.PChainHeight", extra)

	parsed, err := sut.ParseBlock(ctx, valid.Bytes())
	require.NoErrorf(t, err, "%T.ParseBlock()", sut.VM)
	err = sut.VerifyBlock(ctx, &block.Context{ProposerPChainHeight: 7}, parsed)
	require.ErrorContainsf(t, err, "hash mismatch", "%T.VerifyBlock(with context)", sut.VM)
}

// TestRPCNoPChainHeightWithoutContext asserts that headers of blocks built
// without a block context expose no pChainHeight field over JSON-RPC.
func TestRPCNoPChainHeightWithoutContext(t *testing.T) {
	key := txtest.NewKey(t)
	ctx, sut := newSUT(t, withMaxAllocFor(key.EthAddress()))
	w := newWallet(key, sut.ctx, sut.Client)

	blk := sut.issueAndExecute(ctx, t, w.newMinimalTx(t))
	require.Nilf(t, customtypes.GetHeaderExtra(blk.Header()).PChainHeight, "PChainHeight of block built without context")

	var got map[string]any
	client := sut.ethclient.Client()
	require.NoError(t, client.CallContext(ctx, &got, "eth_getHeaderByNumber", hexutil.EncodeUint64(blk.NumberU64())))
	_, ok := got["pChainHeight"]
	require.Falsef(t, ok, "eth_getHeaderByNumber returned pChainHeight %v for a block without a block context", got["pChainHeight"])
}
