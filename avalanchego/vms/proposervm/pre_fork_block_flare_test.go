// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package proposervm

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/consensus/snowman"
	"github.com/ava-labs/avalanchego/snow/consensus/snowman/snowmantest"
	"github.com/ava-labs/avalanchego/snow/snowtest"
	"github.com/ava-labs/avalanchego/upgrade"
	"github.com/ava-labs/avalanchego/upgrade/upgradetest"
	"github.com/ava-labs/avalanchego/vms/proposervm/proposer"

	statelessblock "github.com/ava-labs/avalanchego/vms/proposervm/block"
)

// TestPreForkChildOfZeroTimeGenesis reproduces the Flare and Coston2 C-chain
// history as a node bootstrapping from genesis sees it: every upgrade, Apricot
// Phase 4 included, is at Unix 0, the genesis has timestamp 0, and the
// ProposerVM was activated (InitiallyActiveTime, see
// upgrade.Config.IsProposerVMActivated) before block 1 was built in 2022.
// Block 1 is therefore a pre-fork block, whose verification must not require
// the genesis to be an oracle block, and every later block is post-fork.
func TestPreForkChildOfZeroTimeGenesis(t *testing.T) {
	require := require.New(t)

	coreVM, _, proVM, _ := initTestProposerVM(t, upgradetest.Latest, 0, upgrade.ZeroTime)
	defer func() {
		require.NoError(proVM.Shutdown(t.Context()))
	}()

	// The C-chain genesis has timestamp 0. Pre-fork blocks are always read
	// from the inner VM, so the genesis [initTestProposerVM] installed can be
	// replaced by one with that timestamp.
	genesis := &snowmantest.Block{
		Decidable: snowtest.Decidable{
			IDV:    snowmantest.GenesisID,
			Status: snowtest.Accepted,
		},
		HeightV:    snowmantest.GenesisHeight,
		TimestampV: upgrade.ZeroTime,
		BytesV:     snowmantest.GenesisBytes,
	}
	innerBlk1 := snowmantest.BuildChild(genesis)
	innerBlk1.TimestampV = time.Date(2022, time.September, 7, 11, 18, 19, 0, time.UTC) // Coston2 block 1
	innerBlk2 := snowmantest.BuildChild(innerBlk1)

	coreVM.GetBlockF = func(_ context.Context, blkID ids.ID) (snowman.Block, error) {
		switch blkID {
		case genesis.ID():
			return genesis, nil
		case innerBlk1.ID():
			return innerBlk1, nil
		case innerBlk2.ID():
			return innerBlk2, nil
		default:
			return nil, errUnknownBlock
		}
	}
	coreVM.ParseBlockF = func(_ context.Context, b []byte) (snowman.Block, error) {
		switch {
		case bytes.Equal(b, genesis.Bytes()):
			return genesis, nil
		case bytes.Equal(b, innerBlk1.Bytes()):
			return innerBlk1, nil
		case bytes.Equal(b, innerBlk2.Bytes()):
			return innerBlk2, nil
		default:
			return nil, errUnknownBlock
		}
	}

	// Block 1 arrives as a pre-fork block, exactly as peers serve it.
	blk1, err := proVM.ParseBlock(t.Context(), innerBlk1.Bytes())
	require.NoError(err)
	require.IsType(&preForkBlock{}, blk1)
	require.NoError(blk1.Verify(t.Context()))

	// A post-fork block 1 would have been built before the activation.
	postForkBlk1, err := statelessblock.BuildUnsigned(
		genesis.ID(),
		innerBlk1.Timestamp(),
		0,
		statelessblock.Epoch{},
		innerBlk1.Bytes(),
	)
	require.NoError(err)
	outerBlk1, err := proVM.ParseBlock(t.Context(), postForkBlk1.Bytes())
	require.NoError(err)
	require.IsType(&postForkBlock{}, outerBlk1)
	require.ErrorIs(outerBlk1.Verify(t.Context()), errProposersNotActivated)

	require.NoError(blk1.Accept(t.Context()))
	require.NoError(proVM.SetPreference(t.Context(), blk1.ID()))

	// The ProposerVM is active from block 1 on: block 2 is built and verified
	// as a post-fork block...
	coreVM.BuildBlockF = func(context.Context) (snowman.Block, error) {
		return innerBlk2, nil
	}
	proVM.Set(innerBlk1.Timestamp().Add(proposer.MaxBuildDelay))
	blk2, err := proVM.BuildBlock(t.Context())
	require.NoError(err)
	require.IsType(&postForkBlock{}, blk2)
	require.NoError(blk2.Verify(t.Context()))

	// ...and a pre-fork block 2 is rejected.
	preForkBlk2, err := proVM.ParseBlock(t.Context(), innerBlk2.Bytes())
	require.NoError(err)
	require.IsType(&preForkBlock{}, preForkBlk2)
	require.ErrorIs(preForkBlk2.Verify(t.Context()), errUnexpectedBlockType)
}
