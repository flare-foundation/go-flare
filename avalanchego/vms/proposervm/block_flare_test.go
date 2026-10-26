// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package proposervm

import (
	"bytes"
	"context"
	"testing"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/snow/consensus/snowman"
	"github.com/ava-labs/avalanchego/snow/consensus/snowman/snowmantest"
	"github.com/ava-labs/avalanchego/snow/engine/snowman/block"
	"github.com/ava-labs/avalanchego/snow/engine/snowman/block/blockmock"
	"github.com/ava-labs/avalanchego/snow/validators/validatorsmock"
	"github.com/ava-labs/avalanchego/upgrade/upgradetest"
	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ava-labs/avalanchego/vms/proposervm/proposer"
	"github.com/ava-labs/avalanchego/vms/proposervm/proposer/proposermock"

	statelessblock "github.com/ava-labs/avalanchego/vms/proposervm/block"
)

// TestBuildChildContextCarriesProposerPChainHeight checks that, post-Granite,
// the inner VM receives both heights: the epoch-pinned PChainHeight used by
// warp and the block's own ProposerPChainHeight, which is the height sealed
// into the outer block and the one the SAE C-chain commits into its header.
func TestBuildChildContextCarriesProposerPChainHeight(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)

	var (
		nodeID                     = ids.GenerateTestNodeID()
		epochPChainHeight   uint64 = 1000 // pinned at the start of the current epoch
		parentPChainHeight  uint64 = 1200
		minimumPChainHeight uint64 = 1337 // what GetMinimumHeight recommends now
		parentID                   = ids.GenerateTestID()
		parentTimestamp            = time.Now().Truncate(time.Second)
		// The parent did not seal its epoch, so the child inherits it.
		parentEpoch = statelessblock.Epoch{
			PChainHeight: epochPChainHeight,
			Number:       3,
			StartTime:    parentTimestamp.Unix(),
		}
	)

	innerChildBlock := snowmantest.BuildChild(snowmantest.Genesis)

	innerBlockBuilderVM := blockmock.NewBuildBlockWithContextChainVM(ctrl)
	innerBlockBuilderVM.EXPECT().BuildBlockWithContext(gomock.Any(), &block.Context{
		PChainHeight:         epochPChainHeight,
		ProposerPChainHeight: minimumPChainHeight,
	}).Return(innerChildBlock, nil).AnyTimes()

	vdrState := validatorsmock.NewState(ctrl)
	vdrState.EXPECT().GetMinimumHeight(t.Context()).Return(minimumPChainHeight, nil).AnyTimes()

	windower := proposermock.NewWindower(ctrl)
	windower.EXPECT().ExpectedProposer(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nodeID, nil).AnyTimes()

	vm := &VM{
		Config: Config{
			Upgrades:          upgradetest.GetConfig(upgradetest.Latest), // Granite active
			StakingCertLeaf:   pTestCert,
			StakingLeafSigner: pTestSigner,
			Registerer:        prometheus.NewRegistry(),
		},
		blockBuilderVM: innerBlockBuilderVM,
		ctx: &snow.Context{
			NodeID:         nodeID,
			ValidatorState: vdrState,
			Log:            logging.NoLog{},
		},
		Windower: windower,
	}

	blk := &postForkCommonComponents{
		innerBlk: innerChildBlock,
		vm:       vm,
	}

	gotChild, err := blk.buildChild(
		t.Context(),
		parentID,
		parentTimestamp,
		parentPChainHeight,
		parentEpoch,
	)
	require.NoError(err)

	child := gotChild.(*postForkBlock)
	require.Equal(innerChildBlock, child.innerBlk)
	// The context's ProposerPChainHeight is exactly what the outer block seals.
	require.Equal(minimumPChainHeight, child.PChainHeight())
	require.Equal(epochPChainHeight, child.PChainEpoch().PChainHeight)
}

// contextRecordingBlock is an inner block that records the contexts it is
// verified with.
type contextRecordingBlock struct {
	*snowmantest.Block

	contexts []*block.Context
}

func (*contextRecordingBlock) ShouldVerifyWithContext(context.Context) (bool, error) {
	return true, nil
}

func (b *contextRecordingBlock) VerifyWithContext(ctx context.Context, blockCtx *block.Context) error {
	b.contexts = append(b.contexts, blockCtx)
	return b.Block.Verify(ctx)
}

// TestVerifyChildContextCarriesProposerPChainHeight checks the verify path:
// the inner block is verified with the epoch height in PChainHeight and the
// outer block's own P-chain height in ProposerPChainHeight.
func TestVerifyChildContextCarriesProposerPChainHeight(t *testing.T) {
	require := require.New(t)

	coreVM, valState, proVM, _ := initTestProposerVM(t, upgradetest.Latest, 0)
	defer func() {
		require.NoError(proVM.Shutdown(t.Context()))
	}()

	// Block 1 is the first post-fork block; its pre-fork parent verifies it
	// without a context, so it only serves as a post-fork parent here.
	innerBlk1 := snowmantest.BuildChild(snowmantest.Genesis)
	innerBlk2 := &contextRecordingBlock{Block: snowmantest.BuildChild(innerBlk1)}
	coreVM.GetBlockF = func(_ context.Context, blkID ids.ID) (snowman.Block, error) {
		switch blkID {
		case snowmantest.GenesisID:
			return snowmantest.Genesis, nil
		case innerBlk1.ID():
			return innerBlk1, nil
		case innerBlk2.ID():
			return innerBlk2, nil
		default:
			return nil, errUnknownBlock
		}
	}
	// Accepted outer blocks are re-read from the database, which re-parses
	// their inner blocks.
	coreVM.ParseBlockF = func(_ context.Context, b []byte) (snowman.Block, error) {
		switch {
		case bytes.Equal(b, snowmantest.GenesisBytes):
			return snowmantest.Genesis, nil
		case bytes.Equal(b, innerBlk1.Bytes()):
			return innerBlk1, nil
		case bytes.Equal(b, innerBlk2.Bytes()):
			return innerBlk2, nil
		default:
			return nil, errUnknownBlock
		}
	}

	coreVM.BuildBlockF = func(context.Context) (snowman.Block, error) {
		return innerBlk1, nil
	}
	proVM.Set(proVM.Time().Add(proposer.MaxBuildDelay))
	blk1, err := proVM.BuildBlock(t.Context())
	require.NoError(err)
	require.NoError(blk1.Verify(t.Context()))
	require.NoError(blk1.Accept(t.Context()))
	require.NoError(proVM.SetPreference(t.Context(), blk1.ID()))
	parent := blk1.(*postForkBlock)

	// The proposer now recommends a P-chain height above the epoch pin.
	const proposerPChainHeight uint64 = 100
	valState.GetMinimumHeightF = func(context.Context) (uint64, error) {
		return proposerPChainHeight, nil
	}
	coreVM.BuildBlockF = func(context.Context) (snowman.Block, error) {
		return innerBlk2, nil
	}
	proVM.Set(proVM.Time().Add(proposer.MaxBuildDelay))
	blk2, err := proVM.BuildBlock(t.Context())
	require.NoError(err)
	child := blk2.(*postForkBlock)
	require.Equal(proposerPChainHeight, child.PChainHeight())
	require.LessOrEqual(parent.PChainHeight(), child.PChainEpoch().PChainHeight)
	require.NotEqual(proposerPChainHeight, child.PChainEpoch().PChainHeight)

	require.NoError(blk2.Verify(t.Context()))
	require.Len(innerBlk2.contexts, 1)
	require.Equal(&block.Context{
		PChainHeight:         child.PChainEpoch().PChainHeight,
		ProposerPChainHeight: proposerPChainHeight,
	}, innerBlk2.contexts[0])
}
