// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package sae

import (
	"testing"
	"time"

	"github.com/arr4n/shed/testerr"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/saevm/blocks"

	saeparams "github.com/ava-labs/avalanchego/vms/saevm/params"
	"github.com/ava-labs/avalanchego/vms/saevm/saetest"
)

// TestGetBlockHeight mirrors [TestBlockSources]: GetBlockHeight must report
// the height of every block GetBlock can return, from the in-memory
// consensus-critical set or from the header-number index, and must do so
// without touching the execution results — the "corrupted" block, whose
// execution results cannot be read and which therefore cannot be restored by
// GetBlock, still resolves.
func TestGetBlockHeight(t *testing.T) {
	opt, vmTime := withVMTime(t, time.Unix(saeparams.TauSeconds, 0))
	xdb := &corruptableHeightIndex{HeightIndex: saetest.NewHeightIndexDB()}
	ctx, sut := newSUT(t, 1, opt, withExecResultsDB(xdb))

	genesis := sut.lastAcceptedBlock(t)
	corrupted := sut.runConsensusLoop(t)
	onDisk := sut.runConsensusLoop(t)
	settled := sut.runConsensusLoop(t)
	vmTime.AdvanceToSettle(ctx, t, settled)
	unsettled := sut.runConsensusLoop(t)
	require.NoErrorf(t, unsettled.WaitUntilExecuted(ctx), "%T.WaitUntilExecuted()", unsettled)

	verified := sut.createAndVerifyBlock(t, unsettled)
	unverified := sut.buildAndParseBlock(t, unwrap(t, verified))

	xdb.corrupt(corrupted.Height())

	tests := []struct {
		name            string
		block           *blocks.Block
		wantGetBlockErr testerr.Want
		wantErr         testerr.Want
	}{
		{"genesis", genesis, nil, nil},
		{"corrupted", corrupted, testerr.Is(errInjectedRead), nil},
		{"on_disk", onDisk, nil, nil},
		{"settled_in_memory", settled, nil, nil},
		{"unsettled", unsettled, nil, nil},
		{"verified", unwrap(t, verified), nil, nil},
		{"unverified", unwrap(t, unverified), testerr.Equals(database.ErrNotFound), testerr.Equals(database.ErrNotFound)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := sut.GetBlock(ctx, tt.block.ID())
			if diff := testerr.Diff(err, tt.wantGetBlockErr); diff != "" {
				t.Fatalf("GetBlock(...) %s", diff)
			}

			got, err := sut.rawVM.GetBlockHeight(ctx, tt.block.ID())
			if diff := testerr.Diff(err, tt.wantErr); diff != "" {
				t.Fatalf("GetBlockHeight(...) %s", diff)
			}
			if tt.wantErr != nil {
				return
			}
			require.Equal(t, tt.block.Height(), got, "GetBlockHeight(...)")
		})
	}

	t.Run("unknown", func(t *testing.T) {
		_, err := sut.rawVM.GetBlockHeight(ctx, ids.GenerateTestID())
		require.ErrorIs(t, err, database.ErrNotFound)
	})
}
