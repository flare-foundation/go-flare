// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package cchain

import (
	"context"
	"testing"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core/rawdb"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/database/memdb"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/snow/engine/snowman/block"
	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ava-labs/avalanchego/vms/saevm/blocks"
	"github.com/ava-labs/avalanchego/vms/saevm/cchain/tx/txtest"
)

// TestFactoryVMImplementsHeightByID pins the method set of the VM that
// [Factory.New] hands to the chain manager, which is the VM the proposervm
// wraps on a real node. It is composed of interface values, so a method of
// [VM] reaches the proposervm only if the wrapper forwards it explicitly.
// [block.HeightByIDChainVM] was not forwarded until 2026-11-19: the proposervm
// proposer lookup then fell back to GetBlock on every node and restored the
// block from disk under the context lock (FYEO-FLARE-06).
func TestFactoryVMImplementsHeightByID(t *testing.T) {
	vm, err := (&Factory{}).New(logging.NoLog{})
	require.NoError(t, err, "Factory.New()")

	_, ok := vm.(block.HeightByIDChainVM)
	require.Truef(t, ok, "%T must implement block.HeightByIDChainVM", vm)

	// The interfaces of upstream's wrapper must survive the addition.
	_, ok = vm.(block.ChainVM)
	require.Truef(t, ok, "%T must implement block.ChainVM", vm)
	_, ok = vm.(block.BuildBlockWithContextChainVM)
	require.Truef(t, ok, "%T must implement block.BuildBlockWithContextChainVM", vm)
	_, ok = vm.(block.SetPreferenceWithContextChainVM)
	require.Truef(t, ok, "%T must implement block.SetPreferenceWithContextChainVM", vm)
	_, ok = vm.(block.StateSyncableVM)
	require.Truef(t, ok, "%T must implement block.StateSyncableVM", vm)
}

// TestVMGetBlockHeight checks [VM.GetBlockHeight], the method the factory VM
// forwards to the proposervm, in every state a node can be queried in: it
// reports the height of each accepted block and [database.ErrNotFound] for an
// unknown ID. Once the SAE VM exists it answers from the SAE VM's indexes
// alone, never through [VM.GetBlock]; the index-only entry shows it, as
// [VM.GetBlock] has no block to return for it and its height still resolves.
func TestVMGetBlockHeight(t *testing.T) {
	key := txtest.NewKey(t)
	alloc := withMaxAllocFor(key.EthAddress())
	db := memdb.New()

	ctx, node := newSUT(t, alloc, withDB(db))
	w := newWallet(key, node.ctx, node.Client)

	genesis, err := node.GetBlock(ctx, node.lastAccepted(ctx, t))
	require.NoErrorf(t, err, "%T.GetBlock()", node.VM)
	want := []*blocks.Block{genesis}
	for range 3 {
		want = append(want, node.issueAndExecute(ctx, t, w.newMinimalTx(t)))
	}

	requireHeights := func(ctx context.Context, t *testing.T, s *SUT) {
		t.Helper()
		for _, b := range want {
			got, err := s.GetBlockHeight(ctx, b.ID())
			require.NoErrorf(t, err, "%T.GetBlockHeight(%s)", s.VM, b.ID())
			require.Equalf(t, b.Height(), got, "%T.GetBlockHeight(%s)", s.VM, b.ID())
		}
		_, err := s.GetBlockHeight(ctx, ids.GenerateTestID())
		require.ErrorIsf(t, err, database.ErrNotFound, "%T.GetBlockHeight(unknown)", s.VM)
	}

	// The node that built the blocks still holds the recent ones in memory.
	requireHeights(ctx, t, node)
	require.NoErrorf(t, node.Shutdown(ctx), "%T.Shutdown()", node.VM)

	// After a restart every block is read back from the database.
	for _, state := range []snow.State{
		snow.StateSyncing,
		snow.Bootstrapping,
		snow.NormalOp,
	} {
		t.Run(state.String(), func(t *testing.T) {
			ctx, s := newSUT(t, alloc, withDB(db), withState(state))
			requireHeights(ctx, t, s)

			if state < snow.Bootstrapping {
				return // no SAE VM yet: the documented GetBlock fallback
			}

			// A header-number entry without a block: far above every stored
			// block, so GetBlock rejects it before touching the disk.
			const indexOnlyHeight = uint64(1 << 40)
			var (
				chainDB   = s.VM.VM.GethRPCBackends().ChainDb()
				indexOnly = common.Hash{0xf1, 0xa2, 0xe0}
				id        = ids.ID(indexOnly)
			)
			_, err := s.GetBlockHeight(ctx, id)
			require.ErrorIsf(t, err, database.ErrNotFound, "test setup: %T.GetBlockHeight(index-only ID) before the entry exists", s.VM)

			rawdb.WriteHeaderNumber(chainDB, indexOnly, indexOnlyHeight)
			defer rawdb.DeleteHeaderNumber(chainDB, indexOnly)

			_, err = s.GetBlock(ctx, id)
			require.ErrorIsf(t, err, database.ErrNotFound, "%T.GetBlock(index-only ID)", s.VM)
			got, err := s.GetBlockHeight(ctx, id)
			require.NoErrorf(t, err, "%T.GetBlockHeight(index-only ID)", s.VM)
			require.Equalf(t, indexOnlyHeight, got, "%T.GetBlockHeight(index-only ID)", s.VM)
		})
	}
}
