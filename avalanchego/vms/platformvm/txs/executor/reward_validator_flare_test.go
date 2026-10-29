// Copyright (C) 2019, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package executor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/upgrade/upgradetest"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/units"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/platformvm/genesis/genesistest"
	"github.com/ava-labs/avalanchego/vms/platformvm/platform"
	"github.com/ava-labs/avalanchego/vms/platformvm/reward"
	"github.com/ava-labs/avalanchego/vms/platformvm/state"
	"github.com/ava-labs/avalanchego/vms/platformvm/status"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
)

// TestRewardAutoRenewedValidatorTxFlareStakeLimits checks that restaking an
// auto-renewed validator is bounded by the Flare per-network maximum validator
// stake (inflation_settings.go) and not by backend.Config.MaxValidatorStake,
// which on the Flare networks is unrelated to the primary network's rules (it
// used to be the node's --max-validator-stake flag, defaulting to
// genesis.LocalParams' 3M). Coston2 accepts auto-renewed validators between 1M
// and 200M after Helicon and their reward tx is mandatory: a failing one would
// stall the P-chain. A validator at or above the maximum restakes nothing.
func TestRewardAutoRenewedValidatorTxFlareStakeLimits(t *testing.T) {
	const (
		nodeMaxValidatorStake   = 3 * units.MegaAvax   // genesis.LocalParams.MaxValidatorStake
		costwoMaxValidatorStake = 200 * units.MegaAvax // inflation_settings.go, Coston2 Helicon phase
		costwoMinDelegationFee  = 200_000
		costwoMinStakeDuration  = 7 * 24 * time.Hour
		delegateeReward         = 1 * units.MegaAvax
	)

	tests := []struct {
		name            string
		weight          uint64
		delegateeReward uint64
		// bypassVerification stages the validator without executing its tx, to
		// model a weight above the current maximum (a later phase lowered it).
		bypassVerification bool
		wantRestaked       uint64 // the part of delegateeReward that is restaked
	}{
		{
			name:   "zero_rewards_above_node_config",
			weight: 5 * units.MegaAvax,
		},
		{
			name:            "weight_above_node_config",
			weight:          5 * units.MegaAvax,
			delegateeReward: delegateeReward,
			wantRestaked:    delegateeReward,
		},
		{
			name:            "rewards_capped_at_network_max",
			weight:          costwoMaxValidatorStake - delegateeReward/2,
			delegateeReward: delegateeReward,
			wantRestaked:    delegateeReward / 2,
		},
		{
			name:            "weight_at_network_max",
			weight:          costwoMaxValidatorStake,
			delegateeReward: delegateeReward,
			wantRestaked:    0,
		},
		{
			name:               "weight_above_network_max",
			weight:             costwoMaxValidatorStake + 1*units.MegaAvax,
			delegateeReward:    delegateeReward,
			bypassVerification: true,
			wantRestaked:       0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			env := newEnvironment(t, upgradetest.Latest)
			env.ctx.NetworkID = constants.CostwoID
			env.config.MaxValidatorStake = nodeMaxValidatorStake

			// Fund a key with enough FLR for the Coston2 validator stakes.
			key := genesistest.DefaultFundedKeys[0]
			owner := &secp256k1fx.OutputOwners{Threshold: 1, Addrs: []ids.ShortID{key.Address()}}
			env.state.AddUTXO(&avax.UTXO{
				UTXOID: avax.UTXOID{TxID: ids.GenerateTestID()},
				Asset:  avax.Asset{ID: env.ctx.AVAXAssetID},
				Out: &secp256k1fx.TransferOutput{
					Amt:          2 * costwoMaxValidatorStake,
					OutputOwners: *owner,
				},
			})
			require.NoError(env.state.Commit())

			wallet := newWallet(t, env, walletConfig{keys: genesistest.DefaultFundedKeys[:1]})
			stakerTx, err := wallet.IssueAddAutoRenewedValidatorTx(
				ids.GenerateTestNodeID(),
				tt.weight,
				newProofOfPossession(t),
				owner,
				owner,
				&secp256k1fx.OutputOwners{},
				costwoMinDelegationFee,
				reward.PercentDenominator, // restake all rewards
				costwoMinStakeDuration,
			)
			require.NoError(err)

			cfg := autoRenewedValidatorConfig{
				weight:                   tt.weight,
				delegateeReward:          tt.delegateeReward,
				autoCompoundRewardShares: reward.PercentDenominator,
				restake:                  true,
			}
			if tt.bypassVerification {
				addAutoRenewedValidatorUnverified(t, env, stakerTx, cfg)
			} else {
				// Accepted by the Coston2 rules although above the node config.
				addAutoRenewedValidator(t, env, stakerTx, cfg)
			}

			uStakerTx := stakerTx.Unsigned.(*platform.AddAutoRenewedValidatorTx)
			staker, err := env.state.GetCurrentValidator(uStakerTx.SubnetID(), uStakerTx.NodeID())
			require.NoError(err)
			require.Zero(staker.PotentialReward) // Flare mints no staking rewards

			diff, err := state.NewDiffOn(env.state, state.StakerAdditionAfterDeletionAllowed)
			require.NoError(err)
			diff.SetTimestamp(staker.EndTime)
			// The shared test helper uses the node config's minimum duration.
			// Renew with the original period, which satisfies the Coston2 rules.
			stakingInfo, err := diff.GetStakingInfo(staker.SubnetID, staker.NodeID)
			require.NoError(err)
			stakingInfo.NextPeriod = uStakerTx.Period
			require.NoError(diff.SetStakingInfo(staker.SubnetID, staker.NodeID, stakingInfo))

			rewardTx := newRewardAutoRenewedValidatorTx(t, stakerTx.ID(), diff.GetTimestamp())
			onCommitState, err := state.NewDiffOn(diff, state.StakerAdditionAfterDeletionAllowed)
			require.NoError(err)
			onAbortState, err := state.NewDiffOn(diff, state.StakerAdditionAfterDeletionAllowed)
			require.NoError(err)

			require.NoError(ProposalTx(
				&env.backend,
				state.PickFeeCalculator(env.config, env.state),
				rewardTx,
				onCommitState,
				onAbortState,
			))

			assertRewardAutoRenewedValidator(
				t,
				env.state,
				stakerTx,
				rewardTx,
				onCommitState,
				onAbortState,
				wantRewardAutoRenewedValidator{
					commitRestaked:                true,
					commitWeight:                  tt.weight + tt.wantRestaked,
					commitAccruedDelegateeRewards: tt.wantRestaked,
					commitReward:                  wantReward{delegatee: tt.delegateeReward - tt.wantRestaked},
					abortReward:                   wantReward{delegatee: tt.delegateeReward},
				},
			)
		})
	}
}

// addAutoRenewedValidatorUnverified stages the validator of tx in env.state
// without executing tx, so that a weight the current rules would reject can be
// set up.
func addAutoRenewedValidatorUnverified(t testing.TB, env *environment, tx *platform.Tx, cfg autoRenewedValidatorConfig) {
	t.Helper()

	uTx := tx.Unsigned.(*platform.AddAutoRenewedValidatorTx)
	startTime := env.state.GetTimestamp()
	staker, err := state.NewCurrentStaker(
		tx.ID(),
		uTx,
		startTime,
		startTime.Add(time.Duration(uTx.Period)*time.Second),
		cfg.weight,
		0, // potentialReward: Flare mints no staking rewards
	)
	require.NoError(t, err)

	diff, err := state.NewDiffOn(env.state, state.StakerAdditionAfterDeletionAllowed)
	require.NoError(t, err)
	require.NoError(t, diff.PutCurrentValidator(staker))
	diff.AddTx(tx, status.Committed)
	require.NoError(t, diff.SetStakingInfo(staker.SubnetID, staker.NodeID, state.StakingInfo{
		DelegateeReward:          cfg.delegateeReward,
		AutoCompoundRewardShares: cfg.autoCompoundRewardShares,
		NextPeriod:               uTx.Period,
	}))
	require.NoError(t, diff.Apply(env.state))
	require.NoError(t, env.state.Commit())
}
