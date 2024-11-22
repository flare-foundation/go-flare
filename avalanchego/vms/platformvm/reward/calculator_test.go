// Copyright (C) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package reward

import (
	"fmt"
	"testing"
	"time"

	"github.com/ava-labs/avalanchego/utils/units"
)

const (
	defaultMinStakingDuration = 24 * time.Hour
	defaultMaxStakingDuration = 365 * 24 * time.Hour

	defaultMinValidatorStake = 5 * units.MilliAvax
)

var defaultConfig = Config{
	MaxConsumptionRate: .12 * PercentDenominator,
	MinConsumptionRate: .10 * PercentDenominator,
	MintingPeriod:      365 * 24 * time.Hour,
	SupplyCap:          0 * units.MegaAvax,
}

func TestLongerDurationBonus(t *testing.T) {
	c := NewCalculator(defaultConfig)
	shortDuration := 24 * time.Hour
	totalDuration := 365 * 24 * time.Hour
	shortBalance := units.KiloAvax
	for i := 0; i < int(totalDuration/shortDuration); i++ {
		r := c.Calculate(shortDuration, shortBalance, 359*units.MegaAvax+shortBalance)
		shortBalance += r
	}
	r := c.Calculate(totalDuration%shortDuration, shortBalance, 359*units.MegaAvax+shortBalance)
	shortBalance += r

	longBalance := units.KiloAvax
	longBalance += c.Calculate(totalDuration, longBalance, 359*units.MegaAvax+longBalance)

	if shortBalance != longBalance {
		t.Fatalf("non-zero rewards")
	}
}

func TestRewards(t *testing.T) {
	c := NewCalculator(defaultConfig)
	tests := []struct {
		duration       time.Duration
		stakeAmount    uint64
		existingAmount uint64
		expectedReward uint64
	}{
		// Max duration:
		{ // (720M - 360M) * (1M / 360M) * 12%
			duration:       defaultMaxStakingDuration,
			stakeAmount:    units.MegaAvax,
			existingAmount: 360 * units.MegaAvax,
			expectedReward: 0 * units.KiloAvax,
		},
		{ // (720M - 400M) * (1M / 400M) * 12%
			duration:       defaultMaxStakingDuration,
			stakeAmount:    units.MegaAvax,
			existingAmount: 400 * units.MegaAvax,
			expectedReward: 0 * units.KiloAvax,
		},
		{ // (720M - 400M) * (2M / 400M) * 12%
			duration:       defaultMaxStakingDuration,
			stakeAmount:    2 * units.MegaAvax,
			existingAmount: 400 * units.MegaAvax,
			expectedReward: 0 * units.KiloAvax,
		},
		{ // (720M - 720M) * (1M / 720M) * 12%
			duration:       defaultMaxStakingDuration,
			stakeAmount:    units.MegaAvax,
			existingAmount: defaultConfig.SupplyCap,
			expectedReward: 0,
		},
		// Min duration:
		// (720M - 360M) * (1M / 360M) * (10% + 2% * MinimumStakingDuration / MaximumStakingDuration) * MinimumStakingDuration / MaximumStakingDuration
		{
			duration:       defaultMinStakingDuration,
			stakeAmount:    units.MegaAvax,
			existingAmount: 360 * units.MegaAvax,
			expectedReward: 0,
		},
		// (720M - 360M) * (.005 / 360M) * (10% + 2% * MinimumStakingDuration / MaximumStakingDuration) * MinimumStakingDuration / MaximumStakingDuration
		{
			duration:       defaultMinStakingDuration,
			stakeAmount:    defaultMinValidatorStake,
			existingAmount: 360 * units.MegaAvax,
			expectedReward: 0,
		},
		// (720M - 400M) * (1M / 400M) * (10% + 2% * MinimumStakingDuration / MaximumStakingDuration) * MinimumStakingDuration / MaximumStakingDuration
		{
			duration:       defaultMinStakingDuration,
			stakeAmount:    units.MegaAvax,
			existingAmount: 400 * units.MegaAvax,
			expectedReward: 0,
		},
		// (720M - 400M) * (2M / 400M) * (10% + 2% * MinimumStakingDuration / MaximumStakingDuration) * MinimumStakingDuration / MaximumStakingDuration
		{
			duration:       defaultMinStakingDuration,
			stakeAmount:    2 * units.MegaAvax,
			existingAmount: 400 * units.MegaAvax,
			expectedReward: 0,
		},
		// (720M - 720M) * (1M / 720M) * (10% + 2% * MinimumStakingDuration / MaximumStakingDuration) * MinimumStakingDuration / MaximumStakingDuration
		{
			duration:       defaultMinStakingDuration,
			stakeAmount:    units.MegaAvax,
			existingAmount: defaultConfig.SupplyCap,
			expectedReward: 0,
		},
	}
	for _, test := range tests {
		name := fmt.Sprintf("reward(%s,%d,%d)==%d",
			test.duration,
			test.stakeAmount,
			test.existingAmount,
			test.expectedReward,
		)
		t.Run(name, func(t *testing.T) {
			r := c.Calculate(
				test.duration,
				test.stakeAmount,
				test.existingAmount,
			)
			if r != test.expectedReward {
				t.Fatalf("expected %d; got %d", test.expectedReward, r)
			}
		})
	}
}

// func TestRewardsOverflow(t *testing.T) {
// 	require := require.New(t)

// 	var (
// 		maxSupply     uint64 = math.MaxUint64
// 		initialSupply uint64 = 1
// 	)
// 	c := NewCalculator(Config{
// 		MaxConsumptionRate: PercentDenominator,
// 		MinConsumptionRate: PercentDenominator,
// 		MintingPeriod:      defaultMinStakingDuration,
// 		SupplyCap:          maxSupply,
// 	})
// 	rewards := c.Calculate(
// 		defaultMinStakingDuration,
// 		maxSupply, // The staked amount is larger than the current supply
// 		initialSupply,
// 	)
// 	require.Equal(maxSupply-initialSupply, rewards)
// }

// func TestRewardsMint(t *testing.T) {
// 	require := require.New(t)

// 	var (
// 		maxSupply     uint64 = 1000
// 		initialSupply uint64 = 1
// 	)
// 	c := NewCalculator(Config{
// 		MaxConsumptionRate: PercentDenominator,
// 		MinConsumptionRate: PercentDenominator,
// 		MintingPeriod:      defaultMinStakingDuration,
// 		SupplyCap:          maxSupply,
// 	})
// 	rewards := c.Calculate(
// 		defaultMinStakingDuration,
// 		maxSupply, // The staked amount is larger than the current supply
// 		initialSupply,
// 	)
// 	require.Equal(maxSupply-initialSupply, rewards)
// }
