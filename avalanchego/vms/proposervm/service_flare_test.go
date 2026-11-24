// Copyright (C) 2019, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package proposervm

import (
	"bytes"
	"context"
	"encoding/hex"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/consensus/snowman"
	"github.com/ava-labs/avalanchego/snow/consensus/snowman/snowmantest"
	"github.com/ava-labs/avalanchego/upgrade/upgradetest"
	"github.com/ava-labs/avalanchego/vms/proposervm/block"

	snowmanblock "github.com/ava-labs/avalanchego/snow/engine/snowman/block"
	avajson "github.com/ava-labs/avalanchego/utils/json"
)

// proposerLookupEnv is a proposervm over a two-block post-fork chain built,
// verified and accepted by this node. The first post-fork block (height 1,
// outer1 wrapping coreBlk1) is always built unsigned by the proposervm, so it
// has no proposer; the second (height 2, outer wrapping coreBlk) is built in
// this node's proposer slot and signed, so its proposer is the node itself.
// sibling is another inner child of genesis that the inner VM knows but never
// accepted.
type proposerLookupEnv struct {
	coreVM   *fullVM
	proVM    *VM
	coreBlk1 *snowmantest.Block
	coreBlk  *snowmantest.Block
	sibling  *snowmantest.Block
	outer1   *postForkBlock
	outer    *postForkBlock
	svc      *jsonrpcService
}

func newProposerLookupEnv(t *testing.T) *proposerLookupEnv {
	require := require.New(t)

	coreVM, _, proVM, _ := initTestProposerVM(t, upgradetest.Latest, 0)
	t.Cleanup(func() {
		require.NoError(proVM.Shutdown(context.Background()))
	})

	coreBlk1 := snowmantest.BuildChild(snowmantest.Genesis)
	coreBlk := snowmantest.BuildChild(coreBlk1)
	sibling := snowmantest.BuildChild(snowmantest.Genesis)
	coreHeights := []ids.ID{snowmantest.GenesisID}

	coreVM.GetBlockF = func(_ context.Context, blkID ids.ID) (snowman.Block, error) {
		switch blkID {
		case snowmantest.GenesisID:
			return snowmantest.Genesis, nil
		case coreBlk1.ID():
			return coreBlk1, nil
		case coreBlk.ID():
			return coreBlk, nil
		case sibling.ID():
			return sibling, nil
		default:
			return nil, database.ErrNotFound
		}
	}
	coreVM.ParseBlockF = func(_ context.Context, b []byte) (snowman.Block, error) {
		switch {
		case bytes.Equal(b, snowmantest.GenesisBytes):
			return snowmantest.Genesis, nil
		case bytes.Equal(b, coreBlk1.Bytes()):
			return coreBlk1, nil
		case bytes.Equal(b, coreBlk.Bytes()):
			return coreBlk, nil
		case bytes.Equal(b, sibling.Bytes()):
			return sibling, nil
		default:
			return nil, errUnknownBlock
		}
	}
	coreVM.GetBlockIDAtHeightF = func(_ context.Context, height uint64) (ids.ID, error) {
		if height >= uint64(len(coreHeights)) {
			return ids.Empty, database.ErrNotFound
		}
		return coreHeights[height], nil
	}

	buildAndAccept := func(inner *snowmantest.Block) *postForkBlock {
		coreVM.BuildBlockF = func(context.Context) (snowman.Block, error) {
			return inner, nil
		}
		// Advance the clock into this node's proposer slot so that a child
		// of a post-fork block is signed by this node.
		require.NoError(proVM.waitForProposerWindow())
		builtBlk, err := proVM.BuildBlock(t.Context())
		require.NoError(err)
		require.NoError(builtBlk.Verify(t.Context()))
		require.NoError(proVM.SetPreference(t.Context(), builtBlk.ID()))
		require.NoError(builtBlk.Accept(t.Context()))
		coreHeights = append(coreHeights, inner.ID())

		outer, ok := builtBlk.(*postForkBlock)
		require.True(ok)
		return outer
	}
	outer1 := buildAndAccept(coreBlk1)
	outer := buildAndAccept(coreBlk)

	return &proposerLookupEnv{
		coreVM:   coreVM,
		proVM:    proVM,
		coreBlk1: coreBlk1,
		coreBlk:  coreBlk,
		sibling:  sibling,
		outer1:   outer1,
		outer:    outer,
		svc:      &jsonrpcService{vm: proVM},
	}
}

func (e *proposerLookupEnv) byHash(hash string) (GetProposerResponse, error) {
	var reply GetProposerResponse
	err := e.svc.GetProposerByEVMBlockHash(
		httptest.NewRequest(http.MethodPost, "/", nil),
		&GetProposerByEVMBlockHashArgs{BlockHash: hash},
		&reply,
	)
	return reply, err
}

func (e *proposerLookupEnv) byNumber(number string) (GetProposerResponse, error) {
	var reply GetProposerResponse
	err := e.svc.GetProposerByEVMBlockNumber(
		httptest.NewRequest(http.MethodPost, "/", nil),
		&GetProposerByEVMBlockNumberArgs{BlockNumber: number},
		&reply,
	)
	return reply, err
}

func evmHash(id ids.ID) string {
	return "0x" + hex.EncodeToString(id[:])
}

func TestGetProposerByEVMBlock(t *testing.T) {
	require := require.New(t)
	env := newProposerLookupEnv(t)

	want := GetProposerResponse{
		NodeID:            env.proVM.ctx.NodeID,
		Status:            ProposerStatusOK,
		EVMBlockHash:      evmHash(env.coreBlk.ID()),
		Height:            2,
		PChainHeight:      avajson.Uint64(env.outer.PChainHeight()),
		Timestamp:         avajson.Uint64(uint64(env.outer.Timestamp().Unix())),
		ProposerVMBlockID: env.outer.ID(),
	}

	t.Run("by_hash", func(t *testing.T) {
		got, err := env.byHash(evmHash(env.coreBlk.ID()))
		require.NoError(err)
		require.Equal(want, got)
	})

	for _, number := range []string{"2", "0x2", "0X2", " 2 ", "latest", "LATEST"} {
		t.Run("by_number_"+number, func(t *testing.T) {
			got, err := env.byNumber(number)
			require.NoError(err)
			require.Equal(want, got)
		})
	}

	t.Run("genesis_is_pre_fork", func(t *testing.T) {
		got, err := env.byHash(evmHash(snowmantest.GenesisID))
		require.NoError(err)
		require.Equal(GetProposerResponse{
			Status:       ProposerStatusPreFork,
			EVMBlockHash: evmHash(snowmantest.GenesisID),
			Height:       0,
		}, got)

		got, err = env.byNumber("0")
		require.NoError(err)
		require.Equal(ProposerStatusPreFork, got.Status)
	})

	t.Run("first_post_fork_block_is_unsigned", func(t *testing.T) {
		got, err := env.byNumber("1")
		require.NoError(err)
		require.Equal(GetProposerResponse{
			Status:            ProposerStatusNoProposer,
			EVMBlockHash:      evmHash(env.coreBlk1.ID()),
			Height:            1,
			PChainHeight:      avajson.Uint64(env.outer1.PChainHeight()),
			Timestamp:         avajson.Uint64(uint64(env.outer1.Timestamp().Unix())),
			ProposerVMBlockID: env.outer1.ID(),
		}, got)
	})

	t.Run("unknown_block", func(t *testing.T) {
		unknown := ids.GenerateTestID()
		got, err := env.byHash(evmHash(unknown))
		require.NoError(err)
		require.Equal(GetProposerResponse{
			Status:       ProposerStatusNotFound,
			EVMBlockHash: evmHash(unknown),
		}, got)

		got, err = env.byNumber("7")
		require.NoError(err)
		require.Equal(GetProposerResponse{
			Status: ProposerStatusNotFound,
			Height: 7,
		}, got)
	})

	t.Run("known_but_not_canonical", func(t *testing.T) {
		got, err := env.byHash(evmHash(env.sibling.ID()))
		require.NoError(err)
		require.Equal(GetProposerResponse{
			Status:       ProposerStatusNotCanonical,
			EVMBlockHash: evmHash(env.sibling.ID()),
			Height:       1,
		}, got)
	})

	t.Run("invalid_arguments", func(t *testing.T) {
		for _, hash := range []string{"", "0x", "0xzz", "0x1234"} {
			_, err := env.byHash(hash)
			require.Error(err, hash)
		}
		for _, number := range []string{"", "abc", "0x", "0xg", "-1", "1.5"} {
			_, err := env.byNumber(number)
			require.Error(err, number)
		}
	})
}

// heightByIDVM wraps a ChainVM with [snowmanblock.HeightByIDChainVM].
type heightByIDVM struct {
	snowmanblock.ChainVM

	heights map[ids.ID]uint64
	calls   int
}

func (vm *heightByIDVM) GetBlockHeight(_ context.Context, blkID ids.ID) (uint64, error) {
	vm.calls++
	height, ok := vm.heights[blkID]
	if !ok {
		return 0, database.ErrNotFound
	}
	return height, nil
}

// TestGetProposerByEVMBlockHashPrefersHeightByID checks that the hash lookup
// resolves the height through the inner VM's optional HeightByIDChainVM and
// never fetches the inner block, which on the SAE VM restores it from disk.
func TestGetProposerByEVMBlockHashPrefersHeightByID(t *testing.T) {
	require := require.New(t)
	env := newProposerLookupEnv(t)

	hvm := &heightByIDVM{
		ChainVM: env.proVM.ChainVM,
		heights: map[ids.ID]uint64{env.coreBlk.ID(): 2},
	}
	env.proVM.ChainVM = hvm
	env.coreVM.GetBlockF = func(context.Context, ids.ID) (snowman.Block, error) {
		return nil, errors.New("GetBlock must not be called by the proposer lookup")
	}

	got, err := env.byHash(evmHash(env.coreBlk.ID()))
	require.NoError(err)
	require.Equal(ProposerStatusOK, got.Status)
	require.Equal(env.proVM.ctx.NodeID, got.NodeID)
	require.Equal(1, hvm.calls)

	unknown := ids.GenerateTestID()
	got, err = env.byHash(evmHash(unknown))
	require.NoError(err)
	require.Equal(ProposerStatusNotFound, got.Status)
	require.Equal(2, hvm.calls)
}

// TestGetProposerHeightIndexConsistency exercises the proposerVM-side branches
// by rewriting the height index: a proposerVM block that wraps a different
// inner block is an error, an unsigned one yields noProposer, and a missing
// block or index entry yields pruned.
func TestGetProposerHeightIndexConsistency(t *testing.T) {
	require := require.New(t)
	env := newProposerLookupEnv(t)

	unsignedWrapping := func(innerBytes []byte) block.SignedBlock {
		blk, err := block.BuildUnsigned(
			env.outer1.ID(),
			env.outer.Timestamp(),
			env.outer.PChainHeight(),
			block.Epoch{},
			innerBytes,
		)
		require.NoError(err)
		require.NoError(env.proVM.State.PutBlock(blk))
		return blk
	}

	t.Run("index_points_at_block_wrapping_another_inner", func(t *testing.T) {
		bogus := unsignedWrapping(env.sibling.Bytes())
		require.NoError(env.proVM.State.SetBlockIDAtHeight(2, bogus.ID()))

		_, err := env.byHash(evmHash(env.coreBlk.ID()))
		require.ErrorIs(err, errInnerBlockMismatch)
		_, err = env.byNumber("2")
		require.ErrorIs(err, errInnerBlockMismatch)
	})

	t.Run("unsigned_block_has_no_proposer", func(t *testing.T) {
		unsigned := unsignedWrapping(env.coreBlk.Bytes())
		require.NoError(env.proVM.State.SetBlockIDAtHeight(2, unsigned.ID()))

		got, err := env.byNumber("2")
		require.NoError(err)
		require.Equal(GetProposerResponse{
			Status:            ProposerStatusNoProposer,
			EVMBlockHash:      evmHash(env.coreBlk.ID()),
			Height:            2,
			PChainHeight:      avajson.Uint64(env.outer.PChainHeight()),
			Timestamp:         avajson.Uint64(uint64(env.outer.Timestamp().Unix())),
			ProposerVMBlockID: unsigned.ID(),
		}, got)
	})

	t.Run("index_points_at_missing_block", func(t *testing.T) {
		missing := ids.GenerateTestID()
		require.NoError(env.proVM.State.SetBlockIDAtHeight(2, missing))

		got, err := env.byHash(evmHash(env.coreBlk.ID()))
		require.NoError(err)
		require.Equal(ProposerStatusPruned, got.Status)
		require.Equal(missing, got.ProposerVMBlockID)
	})

	t.Run("index_entry_deleted", func(t *testing.T) {
		require.NoError(env.proVM.State.DeleteBlockIDAtHeight(2))

		got, err := env.byNumber("2")
		require.NoError(err)
		require.Equal(GetProposerResponse{
			Status:       ProposerStatusPruned,
			EVMBlockHash: evmHash(env.coreBlk.ID()),
			Height:       2,
		}, got)
	})
}
