// Copyright (C) 2019, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package genesis

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/units"
	"github.com/ava-labs/avalanchego/vms/platformvm/reward"
)

// TestPublicNetworkStakingConfigs pins the staking parameters of the public
// Flare and Songbird networks that still apply directly on the P-chain (see
// GetStakingConfig): they must equal the LocalParams flag defaults every node
// ran with before config.getStakingConfig stopped honoring the staking flags
// on these networks, or fixing them would have changed the rules for
// permissioned subnet validators and TransformSubnetTx.
func TestPublicNetworkStakingConfigs(t *testing.T) {
	for networkID := range constants.ProductionNetworkIDs {
		t.Run(constants.NetworkName(networkID), func(t *testing.T) {
			require := require.New(t)
			cfg := GetStakingConfig(networkID)
			// Pin the historical defaults independently of LocalParams so that
			// changing both sets of parameters cannot hide a consensus change.
			require.Equal(0.8, cfg.UptimeRequirement)
			require.Equal(24*time.Hour, cfg.MinStakeDuration)
			require.Equal(365*24*time.Hour, cfg.MaxStakeDuration)
			require.Equal(time.Hour, cfg.HeliconMinStakeDuration)
			require.Equal(reward.Config{
				MaxConsumptionRate: 120_000,
				MinConsumptionRate: 100_000,
				MintingPeriod:      365 * 24 * time.Hour,
				SupplyCap:          720 * units.MegaAvax,
			}, cfg.RewardConfig)
			require.LessOrEqual(cfg.MinValidatorStake, cfg.MaxValidatorStake)
		})
	}
}
