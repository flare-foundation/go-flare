// Copyright (C) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package executor

import (
	"fmt"
	"math"
	"testing"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/crypto"
	"github.com/ava-labs/avalanchego/utils/hashing"
	"github.com/ava-labs/avalanchego/vms/platformvm/reward"
	"github.com/ava-labs/avalanchego/vms/platformvm/state"
	"github.com/ava-labs/avalanchego/vms/platformvm/status"
	"github.com/ava-labs/avalanchego/vms/platformvm/txs"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
)

func TestProposalTxExecuteAddDelegator(t *testing.T) {
	dummyHeight := uint64(1)
	rewardAddress := preFundedKeys[0].PublicKey().Address()
	nodeID := ids.NodeID(rewardAddress)

	newValidatorID := ids.GenerateTestNodeID()
	newValidatorStartTime := uint64(defaultValidateStartTime.Add(5 * time.Second).Unix())
	newValidatorEndTime := uint64(defaultValidateEndTime.Add(-5 * time.Second).Unix())

	// [addMinStakeValidator] adds a new validator to the primary network's
	// pending validator set with the minimum staking amount
	addMinStakeValidator := func(target *environment) {
		tx, err := target.txBuilder.NewAddValidatorTx(
			target.config.MinValidatorStake, // stake amount
			newValidatorStartTime,           // start time
			newValidatorEndTime,             // end time
			newValidatorID,                  // node ID
			rewardAddress,                   // Reward Address
			reward.PercentDenominator,       // Shares
			[]*crypto.PrivateKeySECP256K1R{preFundedKeys[0]},
			ids.ShortEmpty,
		)
		if err != nil {
			t.Fatal(err)
		}

		staker := state.NewCurrentStaker(
			tx.ID(),
			tx.Unsigned.(*txs.AddValidatorTx),
			0,
		)

		target.state.PutCurrentValidator(staker)
		target.state.AddTx(tx, status.Committed)
		target.state.SetHeight(dummyHeight)
		if err := target.state.Commit(); err != nil {
			t.Fatal(err)
		}
	}

	// [addMaxStakeValidator] adds a new validator to the primary network's
	// pending validator set with the maximum staking amount
	addMaxStakeValidator := func(target *environment) {
		tx, err := target.txBuilder.NewAddValidatorTx(
			target.config.MaxValidatorStake, // stake amount
			newValidatorStartTime,           // start time
			newValidatorEndTime,             // end time
			newValidatorID,                  // node ID
			rewardAddress,                   // Reward Address
			reward.PercentDenominator,       // Shared
			[]*crypto.PrivateKeySECP256K1R{preFundedKeys[0]},
			ids.ShortEmpty,
		)
		if err != nil {
			t.Fatal(err)
		}

		staker := state.NewCurrentStaker(
			tx.ID(),
			tx.Unsigned.(*txs.AddValidatorTx),
			0,
		)

		target.state.PutCurrentValidator(staker)
		target.state.AddTx(tx, status.Committed)
		target.state.SetHeight(dummyHeight)
		if err := target.state.Commit(); err != nil {
			t.Fatal(err)
		}
	}

	dummyH := newEnvironment()
	currentTimestamp := dummyH.state.GetTimestamp()

	type test struct {
		stakeAmount   uint64
		startTime     uint64
		endTime       uint64
		nodeID        ids.NodeID
		rewardAddress ids.ShortID
		feeKeys       []*crypto.PrivateKeySECP256K1R
		setup         func(*environment)
		AP3Time       time.Time
		shouldErr     bool
		description   string
	}

	tests := []test{
		{
			stakeAmount:   dummyH.config.MinDelegatorStake,
			startTime:     uint64(defaultValidateStartTime.Unix()),
			endTime:       uint64(defaultValidateEndTime.Unix()) + 1,
			nodeID:        nodeID,
			rewardAddress: rewardAddress,
			feeKeys:       []*crypto.PrivateKeySECP256K1R{preFundedKeys[0]},
			setup:         nil,
			AP3Time:       defaultGenesisTime,
			shouldErr:     true,
			description:   "validator stops validating primary network earlier than subnet",
		},
		{
			stakeAmount:   dummyH.config.MinDelegatorStake,
			startTime:     uint64(currentTimestamp.Add(MaxFutureStartTime + time.Second).Unix()),
			endTime:       uint64(currentTimestamp.Add(MaxFutureStartTime * 2).Unix()),
			nodeID:        nodeID,
			rewardAddress: rewardAddress,
			feeKeys:       []*crypto.PrivateKeySECP256K1R{preFundedKeys[0]},
			setup:         nil,
			AP3Time:       defaultGenesisTime,
			shouldErr:     true,
			description:   fmt.Sprintf("validator should not be added more than (%s) in the future", MaxFutureStartTime),
		},
		{
			stakeAmount:   dummyH.config.MinDelegatorStake,
			startTime:     uint64(defaultValidateStartTime.Unix()),
			endTime:       uint64(defaultValidateEndTime.Unix()) + 1,
			nodeID:        nodeID,
			rewardAddress: rewardAddress,
			feeKeys:       []*crypto.PrivateKeySECP256K1R{preFundedKeys[0]},
			setup:         nil,
			AP3Time:       defaultGenesisTime,
			shouldErr:     true,
			description:   "end time is after the primary network end time",
		},
		{
			stakeAmount:   dummyH.config.MinDelegatorStake,
			startTime:     uint64(defaultValidateStartTime.Add(5 * time.Second).Unix()),
			endTime:       uint64(defaultValidateEndTime.Add(-5 * time.Second).Unix()),
			nodeID:        newValidatorID,
			rewardAddress: rewardAddress,
			feeKeys:       []*crypto.PrivateKeySECP256K1R{preFundedKeys[0]},
			setup:         nil,
			AP3Time:       defaultGenesisTime,
			shouldErr:     true,
			description:   "validator not in the current or pending validator sets of the subnet",
		},
		{
			stakeAmount:   dummyH.config.MinDelegatorStake,
			startTime:     newValidatorStartTime - 1, // start validating subnet before primary network
			endTime:       newValidatorEndTime,
			nodeID:        newValidatorID,
			rewardAddress: rewardAddress,
			feeKeys:       []*crypto.PrivateKeySECP256K1R{preFundedKeys[0]},
			setup:         addMinStakeValidator,
			AP3Time:       defaultGenesisTime,
			shouldErr:     true,
			description:   "validator starts validating subnet before primary network",
		},
		{
			stakeAmount:   dummyH.config.MinDelegatorStake,
			startTime:     newValidatorStartTime,
			endTime:       newValidatorEndTime + 1, // stop validating subnet after stopping validating primary network
			nodeID:        newValidatorID,
			rewardAddress: rewardAddress,
			feeKeys:       []*crypto.PrivateKeySECP256K1R{preFundedKeys[0]},
			setup:         addMinStakeValidator,
			AP3Time:       defaultGenesisTime,
			shouldErr:     true,
			description:   "validator stops validating primary network before subnet",
		},
		{
			stakeAmount:   dummyH.config.MinDelegatorStake,
			startTime:     newValidatorStartTime, // same start time as for primary network
			endTime:       newValidatorEndTime,   // same end time as for primary network
			nodeID:        newValidatorID,
			rewardAddress: rewardAddress,
			feeKeys:       []*crypto.PrivateKeySECP256K1R{preFundedKeys[0]},
			setup:         addMinStakeValidator,
			AP3Time:       defaultGenesisTime,
			shouldErr:     false,
			description:   "valid",
		},
		{
			stakeAmount:   dummyH.config.MinDelegatorStake,                  // weight
			startTime:     uint64(currentTimestamp.Unix()),                  // start time
			endTime:       uint64(defaultValidateEndTime.Unix()),            // end time
			nodeID:        nodeID,                                           // node ID
			rewardAddress: rewardAddress,                                    // Reward Address
			feeKeys:       []*crypto.PrivateKeySECP256K1R{preFundedKeys[0]}, // tx fee payer
			setup:         nil,
			AP3Time:       defaultGenesisTime,
			shouldErr:     true,
			description:   "starts validating at current timestamp",
		},
		{
			stakeAmount:   dummyH.config.MinDelegatorStake,                  // weight
			startTime:     uint64(defaultValidateStartTime.Unix()),          // start time
			endTime:       uint64(defaultValidateEndTime.Unix()),            // end time
			nodeID:        nodeID,                                           // node ID
			rewardAddress: rewardAddress,                                    // Reward Address
			feeKeys:       []*crypto.PrivateKeySECP256K1R{preFundedKeys[1]}, // tx fee payer
			setup: func(target *environment) { // Remove all UTXOs owned by keys[1]
				utxoIDs, err := target.state.UTXOIDs(
					preFundedKeys[1].PublicKey().Address().Bytes(),
					ids.Empty,
					math.MaxInt32)
				if err != nil {
					t.Fatal(err)
				}
				for _, utxoID := range utxoIDs {
					target.state.DeleteUTXO(utxoID)
				}
				target.state.SetHeight(dummyHeight)
				if err := target.state.Commit(); err != nil {
					t.Fatal(err)
				}
			},
			AP3Time:     defaultGenesisTime,
			shouldErr:   true,
			description: "tx fee paying key has no funds",
		},
		{
			stakeAmount:   dummyH.config.MinDelegatorStake,
			startTime:     newValidatorStartTime, // same start time as for primary network
			endTime:       newValidatorEndTime,   // same end time as for primary network
			nodeID:        newValidatorID,
			rewardAddress: rewardAddress,
			feeKeys:       []*crypto.PrivateKeySECP256K1R{preFundedKeys[0]},
			setup:         addMaxStakeValidator,
			AP3Time:       defaultValidateEndTime,
			shouldErr:     false,
			description:   "over delegation before AP3",
		},
		{
			stakeAmount:   dummyH.config.MinDelegatorStake,
			startTime:     newValidatorStartTime, // same start time as for primary network
			endTime:       newValidatorEndTime,   // same end time as for primary network
			nodeID:        newValidatorID,
			rewardAddress: rewardAddress,
			feeKeys:       []*crypto.PrivateKeySECP256K1R{preFundedKeys[0]},
			setup:         addMaxStakeValidator,
			AP3Time:       defaultGenesisTime,
			shouldErr:     true,
			description:   "over delegation after AP3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.description, func(t *testing.T) {
			freshTH := newEnvironment()
			freshTH.config.ApricotPhase3Time = tt.AP3Time
			defer func() {
				if err := shutdownEnvironment(freshTH); err != nil {
					t.Fatal(err)
				}
			}()

			tx, err := freshTH.txBuilder.NewAddDelegatorTx(
				tt.stakeAmount,
				tt.startTime,
				tt.endTime,
				tt.nodeID,
				tt.rewardAddress,
				tt.feeKeys,
				ids.ShortEmpty,
			)
			if err != nil {
				t.Fatalf("couldn't build tx: %s", err)
			}
			if tt.setup != nil {
				tt.setup(freshTH)
			}

			onCommitState, err := state.NewDiff(lastAcceptedID, freshTH)
			if err != nil {
				t.Fatal(err)
			}

			onAbortState, err := state.NewDiff(lastAcceptedID, freshTH)
			if err != nil {
				t.Fatal(err)
			}

			executor := ProposalTxExecutor{
				OnCommitState: onCommitState,
				OnAbortState:  onAbortState,
				Backend:       &freshTH.backend,
				Tx:            tx,
			}
			err = tx.Unsigned.Visit(&executor)
			if err != nil && !tt.shouldErr {
				t.Fatalf("shouldn't have errored but got %s", err)
			} else if err == nil && tt.shouldErr {
				t.Fatalf("expected test to error but got none")
			}

			mempoolExecutor := MempoolTxVerifier{
				Backend:       &freshTH.backend,
				ParentID:      lastAcceptedID,
				StateVersions: freshTH,
				Tx:            tx,
			}
			err = tx.Unsigned.Visit(&mempoolExecutor)
			if err != nil && !tt.shouldErr {
				t.Fatalf("shouldn't have errored but got %s", err)
			} else if err == nil && tt.shouldErr {
				t.Fatalf("expected test to error but got none")
			}
		})
	}
}

func TestProposalTxExecuteAddSubnetValidator(t *testing.T) {
	env := newEnvironment()
	env.ctx.Lock.Lock()
	defer func() {
		if err := shutdownEnvironment(env); err != nil {
			t.Fatal(err)
		}
	}()

	nodeID := preFundedKeys[0].PublicKey().Address()

	{
		// Case: Proposed validator currently validating primary network
		// but stops validating subnet after stops validating primary network
		// (note that keys[0] is a genesis validator)
		tx, err := env.txBuilder.NewAddSubnetValidatorTx(
			defaultWeight,
			uint64(defaultValidateStartTime.Unix()),
			uint64(defaultValidateEndTime.Unix())+1,
			ids.NodeID(nodeID),
			testSubnet1.ID(),
			[]*crypto.PrivateKeySECP256K1R{testSubnet1ControlKeys[0], testSubnet1ControlKeys[1]},
			ids.ShortEmpty, // change addr
		)
		if err != nil {
			t.Fatal(err)
		}

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		executor := ProposalTxExecutor{
			OnCommitState: onCommitState,
			OnAbortState:  onAbortState,
			Backend:       &env.backend,
			Tx:            tx,
		}
		err = tx.Unsigned.Visit(&executor)
		if err == nil {
			t.Fatal("should have failed because validator stops validating primary network earlier than subnet")
		}
	}

	{
		// Case: Proposed validator currently validating primary network
		// and proposed subnet validation period is subset of
		// primary network validation period
		// (note that keys[0] is a genesis validator)
		tx, err := env.txBuilder.NewAddSubnetValidatorTx(
			defaultWeight,
			uint64(defaultValidateStartTime.Unix()+1),
			uint64(defaultValidateEndTime.Unix()),
			ids.NodeID(nodeID),
			testSubnet1.ID(),
			[]*crypto.PrivateKeySECP256K1R{testSubnet1ControlKeys[0], testSubnet1ControlKeys[1]},
			ids.ShortEmpty, // change addr
		)
		if err != nil {
			t.Fatal(err)
		}

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		executor := ProposalTxExecutor{
			OnCommitState: onCommitState,
			OnAbortState:  onAbortState,
			Backend:       &env.backend,
			Tx:            tx,
		}
		err = tx.Unsigned.Visit(&executor)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Add a validator to pending validator set of primary network
	key, err := testKeyfactory.NewPrivateKey()
	if err != nil {
		t.Fatal(err)
	}
	pendingDSValidatorID := ids.NodeID(key.PublicKey().Address())

	// starts validating primary network 10 seconds after genesis
	DSStartTime := defaultGenesisTime.Add(10 * time.Second)
	DSEndTime := DSStartTime.Add(5 * defaultMinStakingDuration)

	addDSTx, err := env.txBuilder.NewAddValidatorTx(
		env.config.MinValidatorStake, // stake amount
		uint64(DSStartTime.Unix()),   // start time
		uint64(DSEndTime.Unix()),     // end time
		pendingDSValidatorID,         // node ID
		nodeID,                       // reward address
		reward.PercentDenominator,    // shares
		[]*crypto.PrivateKeySECP256K1R{preFundedKeys[0]},
		ids.ShortEmpty,
	)
	if err != nil {
		t.Fatal(err)
	}

	{
		// Case: Proposed validator isn't in pending or current validator sets
		tx, err := env.txBuilder.NewAddSubnetValidatorTx(
			defaultWeight,
			uint64(DSStartTime.Unix()), // start validating subnet before primary network
			uint64(DSEndTime.Unix()),
			pendingDSValidatorID,
			testSubnet1.ID(),
			[]*crypto.PrivateKeySECP256K1R{testSubnet1ControlKeys[0], testSubnet1ControlKeys[1]},
			ids.ShortEmpty, // change addr
		)
		if err != nil {
			t.Fatal(err)
		}

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		executor := ProposalTxExecutor{
			OnCommitState: onCommitState,
			OnAbortState:  onAbortState,
			Backend:       &env.backend,
			Tx:            tx,
		}
		err = tx.Unsigned.Visit(&executor)
		if err == nil {
			t.Fatal("should have failed because validator not in the current or pending validator sets of the primary network")
		}
	}

	staker := state.NewCurrentStaker(
		addDSTx.ID(),
		addDSTx.Unsigned.(*txs.AddValidatorTx),
		0,
	)

	env.state.PutCurrentValidator(staker)
	env.state.AddTx(addDSTx, status.Committed)
	dummyHeight := uint64(1)
	env.state.SetHeight(dummyHeight)
	if err := env.state.Commit(); err != nil {
		t.Fatal(err)
	}

	// Node with ID key.PublicKey().Address() now a pending validator for primary network

	{
		// Case: Proposed validator is pending validator of primary network
		// but starts validating subnet before primary network
		tx, err := env.txBuilder.NewAddSubnetValidatorTx(
			defaultWeight,
			uint64(DSStartTime.Unix())-1, // start validating subnet before primary network
			uint64(DSEndTime.Unix()),
			pendingDSValidatorID,
			testSubnet1.ID(),
			[]*crypto.PrivateKeySECP256K1R{testSubnet1ControlKeys[0], testSubnet1ControlKeys[1]},
			ids.ShortEmpty, // change addr
		)
		if err != nil {
			t.Fatal(err)
		}

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		executor := ProposalTxExecutor{
			OnCommitState: onCommitState,
			OnAbortState:  onAbortState,
			Backend:       &env.backend,
			Tx:            tx,
		}
		err = tx.Unsigned.Visit(&executor)
		if err == nil {
			t.Fatal("should have failed because validator starts validating primary network before starting to validate primary network")
		}
	}

	{
		// Case: Proposed validator is pending validator of primary network
		// but stops validating subnet after primary network
		tx, err := env.txBuilder.NewAddSubnetValidatorTx(
			defaultWeight,
			uint64(DSStartTime.Unix()),
			uint64(DSEndTime.Unix())+1, // stop validating subnet after stopping validating primary network
			pendingDSValidatorID,
			testSubnet1.ID(),
			[]*crypto.PrivateKeySECP256K1R{testSubnet1ControlKeys[0], testSubnet1ControlKeys[1]},
			ids.ShortEmpty, // change addr
		)
		if err != nil {
			t.Fatal(err)
		}

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		executor := ProposalTxExecutor{
			OnCommitState: onCommitState,
			OnAbortState:  onAbortState,
			Backend:       &env.backend,
			Tx:            tx,
		}
		err = tx.Unsigned.Visit(&executor)
		if err == nil {
			t.Fatal("should have failed because validator stops validating primary network after stops validating primary network")
		}
	}

	{
		// Case: Proposed validator is pending validator of primary network and
		// period validating subnet is subset of time validating primary network
		tx, err := env.txBuilder.NewAddSubnetValidatorTx(
			defaultWeight,
			uint64(DSStartTime.Unix()), // same start time as for primary network
			uint64(DSEndTime.Unix()),   // same end time as for primary network
			pendingDSValidatorID,
			testSubnet1.ID(),
			[]*crypto.PrivateKeySECP256K1R{testSubnet1ControlKeys[0], testSubnet1ControlKeys[1]},
			ids.ShortEmpty, // change addr
		)
		if err != nil {
			t.Fatal(err)
		}

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		executor := ProposalTxExecutor{
			OnCommitState: onCommitState,
			OnAbortState:  onAbortState,
			Backend:       &env.backend,
			Tx:            tx,
		}
		err = tx.Unsigned.Visit(&executor)
		if err != nil {
			t.Fatal(err)
		}
	}

	// Case: Proposed validator start validating at/before current timestamp
	// First, advance the timestamp
	newTimestamp := defaultGenesisTime.Add(2 * time.Second)
	env.state.SetTimestamp(newTimestamp)

	{
		tx, err := env.txBuilder.NewAddSubnetValidatorTx(
			defaultWeight,               // weight
			uint64(newTimestamp.Unix()), // start time
			uint64(newTimestamp.Add(defaultMinStakingDuration).Unix()), // end time
			ids.NodeID(nodeID), // node ID
			testSubnet1.ID(),   // subnet ID
			[]*crypto.PrivateKeySECP256K1R{testSubnet1ControlKeys[0], testSubnet1ControlKeys[1]},
			ids.ShortEmpty, // change addr
		)
		if err != nil {
			t.Fatal(err)
		}

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		executor := ProposalTxExecutor{
			OnCommitState: onCommitState,
			OnAbortState:  onAbortState,
			Backend:       &env.backend,
			Tx:            tx,
		}
		err = tx.Unsigned.Visit(&executor)
		if err == nil {
			t.Fatal("should have failed verification because starts validating at current timestamp")
		}
	}

	// reset the timestamp
	env.state.SetTimestamp(defaultGenesisTime)

	// Case: Proposed validator already validating the subnet
	// First, add validator as validator of subnet
	subnetTx, err := env.txBuilder.NewAddSubnetValidatorTx(
		defaultWeight,                           // weight
		uint64(defaultValidateStartTime.Unix()), // start time
		uint64(defaultValidateEndTime.Unix()),   // end time
		ids.NodeID(nodeID),                      // node ID
		testSubnet1.ID(),                        // subnet ID
		[]*crypto.PrivateKeySECP256K1R{testSubnet1ControlKeys[0], testSubnet1ControlKeys[1]},
		ids.ShortEmpty,
	)
	if err != nil {
		t.Fatal(err)
	}

	staker = state.NewCurrentStaker(
		subnetTx.ID(),
		subnetTx.Unsigned.(*txs.AddSubnetValidatorTx),
		0,
	)

	env.state.PutCurrentValidator(staker)
	env.state.AddTx(subnetTx, status.Committed)
	env.state.SetHeight(dummyHeight)
	if err := env.state.Commit(); err != nil {
		t.Fatal(err)
	}

	{
		// Node with ID nodeIDKey.PublicKey().Address() now validating subnet with ID testSubnet1.ID
		duplicateSubnetTx, err := env.txBuilder.NewAddSubnetValidatorTx(
			defaultWeight,                           // weight
			uint64(defaultValidateStartTime.Unix()), // start time
			uint64(defaultValidateEndTime.Unix()),   // end time
			ids.NodeID(nodeID),                      // node ID
			testSubnet1.ID(),                        // subnet ID
			[]*crypto.PrivateKeySECP256K1R{testSubnet1ControlKeys[0], testSubnet1ControlKeys[1]},
			ids.ShortEmpty, // change addr
		)
		if err != nil {
			t.Fatal(err)
		}

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		executor := ProposalTxExecutor{
			OnCommitState: onCommitState,
			OnAbortState:  onAbortState,
			Backend:       &env.backend,
			Tx:            duplicateSubnetTx,
		}
		err = duplicateSubnetTx.Unsigned.Visit(&executor)
		if err == nil {
			t.Fatal("should have failed verification because validator already validating the specified subnet")
		}
	}

	env.state.DeleteCurrentValidator(staker)
	env.state.SetHeight(dummyHeight)
	if err := env.state.Commit(); err != nil {
		t.Fatal(err)
	}

	{
		// Case: Too many signatures
		tx, err := env.txBuilder.NewAddSubnetValidatorTx(
			defaultWeight,                     // weight
			uint64(defaultGenesisTime.Unix()), // start time
			uint64(defaultGenesisTime.Add(defaultMinStakingDuration).Unix())+1, // end time
			ids.NodeID(nodeID), // node ID
			testSubnet1.ID(),   // subnet ID
			[]*crypto.PrivateKeySECP256K1R{testSubnet1ControlKeys[0], testSubnet1ControlKeys[1], testSubnet1ControlKeys[2]},
			ids.ShortEmpty, // change addr
		)
		if err != nil {
			t.Fatal(err)
		}

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		executor := ProposalTxExecutor{
			OnCommitState: onCommitState,
			OnAbortState:  onAbortState,
			Backend:       &env.backend,
			Tx:            tx,
		}
		err = tx.Unsigned.Visit(&executor)
		if err == nil {
			t.Fatal("should have failed verification because tx has 3 signatures but only 2 needed")
		}
	}

	{
		// Case: Too few signatures
		tx, err := env.txBuilder.NewAddSubnetValidatorTx(
			defaultWeight,                     // weight
			uint64(defaultGenesisTime.Unix()), // start time
			uint64(defaultGenesisTime.Add(defaultMinStakingDuration).Unix()), // end time
			ids.NodeID(nodeID), // node ID
			testSubnet1.ID(),   // subnet ID
			[]*crypto.PrivateKeySECP256K1R{testSubnet1ControlKeys[0], testSubnet1ControlKeys[2]},
			ids.ShortEmpty, // change addr
		)
		if err != nil {
			t.Fatal(err)
		}

		// Remove a signature
		addSubnetValidatorTx := tx.Unsigned.(*txs.AddSubnetValidatorTx)
		input := addSubnetValidatorTx.SubnetAuth.(*secp256k1fx.Input)
		input.SigIndices = input.SigIndices[1:]
		// This tx was syntactically verified when it was created...pretend it wasn't so we don't use cache
		addSubnetValidatorTx.SyntacticallyVerified = false

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		executor := ProposalTxExecutor{
			OnCommitState: onCommitState,
			OnAbortState:  onAbortState,
			Backend:       &env.backend,
			Tx:            tx,
		}
		err = tx.Unsigned.Visit(&executor)
		if err == nil {
			t.Fatal("should have failed verification because not enough control sigs")
		}
	}

	{
		// Case: Control Signature from invalid key (keys[3] is not a control key)
		tx, err := env.txBuilder.NewAddSubnetValidatorTx(
			defaultWeight,                     // weight
			uint64(defaultGenesisTime.Unix()), // start time
			uint64(defaultGenesisTime.Add(defaultMinStakingDuration).Unix()), // end time
			ids.NodeID(nodeID), // node ID
			testSubnet1.ID(),   // subnet ID
			[]*crypto.PrivateKeySECP256K1R{testSubnet1ControlKeys[0], preFundedKeys[1]},
			ids.ShortEmpty, // change addr
		)
		if err != nil {
			t.Fatal(err)
		}
		// Replace a valid signature with one from keys[3]
		sig, err := preFundedKeys[3].SignHash(hashing.ComputeHash256(tx.Unsigned.Bytes()))
		if err != nil {
			t.Fatal(err)
		}
		copy(tx.Creds[0].(*secp256k1fx.Credential).Sigs[0][:], sig)

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		executor := ProposalTxExecutor{
			OnCommitState: onCommitState,
			OnAbortState:  onAbortState,
			Backend:       &env.backend,
			Tx:            tx,
		}
		err = tx.Unsigned.Visit(&executor)
		if err == nil {
			t.Fatal("should have failed verification because a control sig is invalid")
		}
	}

	{
		// Case: Proposed validator in pending validator set for subnet
		// First, add validator to pending validator set of subnet
		tx, err := env.txBuilder.NewAddSubnetValidatorTx(
			defaultWeight,                       // weight
			uint64(defaultGenesisTime.Unix())+1, // start time
			uint64(defaultGenesisTime.Add(defaultMinStakingDuration).Unix())+1, // end time
			ids.NodeID(nodeID), // node ID
			testSubnet1.ID(),   // subnet ID
			[]*crypto.PrivateKeySECP256K1R{testSubnet1ControlKeys[0], testSubnet1ControlKeys[1]},
			ids.ShortEmpty, // change addr
		)
		if err != nil {
			t.Fatal(err)
		}

		staker = state.NewCurrentStaker(
			subnetTx.ID(),
			subnetTx.Unsigned.(*txs.AddSubnetValidatorTx),
			0,
		)

		env.state.PutCurrentValidator(staker)
		env.state.AddTx(tx, status.Committed)
		env.state.SetHeight(dummyHeight)
		if err := env.state.Commit(); err != nil {
			t.Fatal(err)
		}

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		executor := ProposalTxExecutor{
			OnCommitState: onCommitState,
			OnAbortState:  onAbortState,
			Backend:       &env.backend,
			Tx:            tx,
		}
		err = tx.Unsigned.Visit(&executor)
		if err == nil {
			t.Fatal("should have failed verification because validator already in pending validator set of the specified subnet")
		}
	}
}

func TestProposalTxExecuteAddValidator(t *testing.T) {
	env := newEnvironment()
	env.ctx.Lock.Lock()
	defer func() {
		if err := shutdownEnvironment(env); err != nil {
			t.Fatal(err)
		}
	}()

	nodeID := ids.GenerateTestNodeID()

	{
		// Case: Validator's start time too early
		tx, err := env.txBuilder.NewAddValidatorTx(
			env.config.MinValidatorStake,
			uint64(defaultValidateStartTime.Unix())-1,
			uint64(defaultValidateEndTime.Unix()),
			nodeID,
			ids.ShortEmpty,
			reward.PercentDenominator,
			[]*crypto.PrivateKeySECP256K1R{preFundedKeys[0]},
			ids.ShortEmpty, // change addr
		)
		if err != nil {
			t.Fatal(err)
		}

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		executor := ProposalTxExecutor{
			OnCommitState: onCommitState,
			OnAbortState:  onAbortState,
			Backend:       &env.backend,
			Tx:            tx,
		}
		err = tx.Unsigned.Visit(&executor)
		if err == nil {
			t.Fatal("should've errored because start time too early")
		}
	}

	{
		// Case: Validator's start time too far in the future
		tx, err := env.txBuilder.NewAddValidatorTx(
			env.config.MinValidatorStake,
			uint64(defaultValidateStartTime.Add(MaxFutureStartTime).Unix()+1),
			uint64(defaultValidateStartTime.Add(MaxFutureStartTime).Add(defaultMinStakingDuration).Unix()+1),
			nodeID,
			ids.ShortEmpty,
			reward.PercentDenominator,
			[]*crypto.PrivateKeySECP256K1R{preFundedKeys[0]},
			ids.ShortEmpty, // change addr
		)
		if err != nil {
			t.Fatal(err)
		}

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		executor := ProposalTxExecutor{
			OnCommitState: onCommitState,
			OnAbortState:  onAbortState,
			Backend:       &env.backend,
			Tx:            tx,
		}
		err = tx.Unsigned.Visit(&executor)
		if err == nil {
			t.Fatal("should've errored because start time too far in the future")
		}
	}

	{
		// Case: Validator already validating primary network
		tx, err := env.txBuilder.NewAddValidatorTx(
			env.config.MinValidatorStake,
			uint64(defaultValidateStartTime.Unix()),
			uint64(defaultValidateEndTime.Unix()),
			nodeID,
			ids.ShortEmpty,
			reward.PercentDenominator,
			[]*crypto.PrivateKeySECP256K1R{preFundedKeys[0]},
			ids.ShortEmpty, // change addr
		)
		if err != nil {
			t.Fatal(err)
		}

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		executor := ProposalTxExecutor{
			OnCommitState: onCommitState,
			OnAbortState:  onAbortState,
			Backend:       &env.backend,
			Tx:            tx,
		}
		err = tx.Unsigned.Visit(&executor)
		if err == nil {
			t.Fatal("should've errored because validator already validating")
		}
	}

	{
		// Case: Validator in pending validator set of primary network
		startTime := defaultGenesisTime.Add(1 * time.Second)
		tx, err := env.txBuilder.NewAddValidatorTx(
			env.config.MinValidatorStake,                            // stake amount
			uint64(startTime.Unix()),                                // start time
			uint64(startTime.Add(defaultMinStakingDuration).Unix()), // end time
			nodeID,
			ids.ShortEmpty,
			reward.PercentDenominator, // shares
			[]*crypto.PrivateKeySECP256K1R{preFundedKeys[0]},
			ids.ShortEmpty, // change addr
		)
		if err != nil {
			t.Fatal(err)
		}

		staker := state.NewCurrentStaker(
			tx.ID(),
			tx.Unsigned.(*txs.AddValidatorTx),
			0,
		)

		env.state.PutCurrentValidator(staker)
		env.state.AddTx(tx, status.Committed)
		dummyHeight := uint64(1)
		env.state.SetHeight(dummyHeight)
		if err := env.state.Commit(); err != nil {
			t.Fatal(err)
		}

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		executor := ProposalTxExecutor{
			OnCommitState: onCommitState,
			OnAbortState:  onAbortState,
			Backend:       &env.backend,
			Tx:            tx,
		}
		err = tx.Unsigned.Visit(&executor)
		if err == nil {
			t.Fatal("should have failed because validator in pending validator set")
		}
	}

	{
		// Case: Validator doesn't have enough tokens to cover stake amount
		tx, err := env.txBuilder.NewAddValidatorTx( // create the tx
			env.config.MinValidatorStake,
			uint64(defaultValidateStartTime.Unix()),
			uint64(defaultValidateEndTime.Unix()),
			nodeID,
			ids.ShortEmpty,
			reward.PercentDenominator,
			[]*crypto.PrivateKeySECP256K1R{preFundedKeys[0]},
			ids.ShortEmpty, // change addr
		)
		if err != nil {
			t.Fatal(err)
		}

		// Remove all UTXOs owned by preFundedKeys[0]
		utxoIDs, err := env.state.UTXOIDs(preFundedKeys[0].PublicKey().Address().Bytes(), ids.Empty, math.MaxInt32)
		if err != nil {
			t.Fatal(err)
		}
		for _, utxoID := range utxoIDs {
			env.state.DeleteUTXO(utxoID)
		}

		onCommitState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		onAbortState, err := state.NewDiff(lastAcceptedID, env)
		if err != nil {
			t.Fatal(err)
		}

		executor := ProposalTxExecutor{
			OnCommitState: onCommitState,
			OnAbortState:  onAbortState,
			Backend:       &env.backend,
			Tx:            tx,
		}
		err = tx.Unsigned.Visit(&executor)
		if err == nil {
			t.Fatal("should have failed because tx fee paying key has no funds")
		}
	}
}
