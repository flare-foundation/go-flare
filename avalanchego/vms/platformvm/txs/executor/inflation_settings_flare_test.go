// Copyright (C) 2019-2024, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package executor

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/genesis"
	"github.com/ava-labs/avalanchego/upgrade"
	"github.com/ava-labs/avalanchego/upgrade/upgradetest"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/units"
	"github.com/ava-labs/avalanchego/vms/platformvm/config"
)

// Switching from flag defaults to public genesis params must preserve the
// primary network's inflation rules throughout each network's history.
func TestPublicNetworkInflationSettingsPreserveDefaults(t *testing.T) {
	for _, networkID := range []uint32{constants.FlareID, constants.SongbirdID, constants.CostwoID, constants.CostonID} {
		t.Run(constants.NetworkName(networkID), func(t *testing.T) {
			upgrades := upgrade.GetConfig(networkID)
			toInternal := func(staking genesis.StakingConfig) *config.Internal {
				return &config.Internal{
					MinValidatorStake:       staking.MinValidatorStake,
					MaxValidatorStake:       staking.MaxValidatorStake,
					MinDelegatorStake:       staking.MinDelegatorStake,
					MinDelegationFee:        staking.MinDelegationFee,
					MinStakeDuration:        staking.MinStakeDuration,
					MaxStakeDuration:        staking.MaxStakeDuration,
					HeliconMinStakeDuration: staking.HeliconMinStakeDuration,
					UpgradeConfig:           upgrades,
				}
			}
			previous := toInternal(genesis.LocalParams.StakingConfig)
			current := toInternal(genesis.GetStakingConfig(networkID))
			genesisTime := time.Unix(int64(genesis.GetConfig(networkID).StartTime), 0)
			timestamps := map[string]time.Time{"genesis": genesisTime}
			boundaries := map[string]time.Time{
				"granite": upgrades.GraniteTime,
				"helicon": upgrades.HeliconTime,
			}
			switch networkID {
			case constants.FlareID:
				boundaries["phase_2"] = time.Date(2023, time.October, 1, 0, 0, 0, 0, time.UTC)
			case constants.CostwoID:
				boundaries["phase_2"] = time.Date(2023, time.September, 7, 0, 0, 0, 0, time.UTC)
			}
			for name, boundary := range boundaries {
				if boundary.Equal(upgrade.UnscheduledActivationTime) || boundary.Before(genesisTime) {
					continue
				}
				timestamps[name+"_before"] = boundary.Add(-time.Second)
				timestamps[name] = boundary
				timestamps[name+"_after"] = boundary.Add(time.Second)
			}
			settings := inflationSettingsVariants.GetValue(networkID)
			for name, timestamp := range timestamps {
				t.Run(name, func(t *testing.T) {
					require.Equal(t, settings(timestamp, previous), settings(timestamp, current))
				})
			}
		})
	}
}

// TestCostwoHeliconStakingDurations pins the Coston2 minimum staking
// durations on both sides of the Helicon activation: Helicon shortens the
// validator minimum from 60 days to 7 days and the delegation minimum from
// 14 days to 24 hours, leaving every other parameter unchanged.
func TestCostwoHeliconStakingDurations(t *testing.T) {
	ts := upgrade.InitiallyActiveTime

	tests := []struct {
		name                string
		fork                upgradetest.Fork
		minStakeDuration    time.Duration
		minDelegateDuration time.Duration
	}{
		{
			name:                "granite",
			fork:                upgradetest.Granite,
			minStakeDuration:    60 * 24 * time.Hour,
			minDelegateDuration: 14 * 24 * time.Hour,
		},
		{
			name:                "helicon",
			fork:                upgradetest.Helicon,
			minStakeDuration:    7 * 24 * time.Hour,
			minDelegateDuration: 24 * time.Hour,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require := require.New(t)
			cfg := &config.Internal{UpgradeConfig: upgradetest.GetConfig(tt.fork)}
			s := inflationSettingsVariants.GetValue(constants.CostwoID)(ts, cfg)

			require.Equal(tt.minStakeDuration, s.MinStakeDuration)
			require.Equal(tt.minDelegateDuration, s.MinDelegateDuration)

			// Unchanged across the Helicon activation.
			require.Equal(uint64(1*units.MegaAvax), s.MinValidatorStake)
			require.Equal(uint64(200*units.MegaAvax), s.MaxValidatorStake)
			require.Equal(uint64(50*units.KiloAvax), s.MinDelegatorStake)
			require.Equal(uint32(200_000), s.MinDelegationFee)
			require.Equal(365*24*time.Hour, s.MaxStakeDuration)
			require.Equal(uint64(15), s.MaxValidatorWeightFactor)
		})
	}

	// Flare has no Helicon phase yet: its Granite durations still apply.
	cfg := &config.Internal{UpgradeConfig: upgradetest.GetConfig(upgradetest.Helicon)}
	s := inflationSettingsVariants.GetValue(constants.FlareID)(ts, cfg)
	require.Equal(t, 60*24*time.Hour, s.MinStakeDuration)
	require.Equal(t, 14*24*time.Hour, s.MinDelegateDuration)
}
