// Copyright (C) 2019, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package reward

import (
	"fmt"
	"testing"
	"time"

	"github.com/ava-labs/avalanchego/upgrade/upgradetest"
	"github.com/ava-labs/avalanchego/utils/units"
	"github.com/stretchr/testify/require"
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

type calculatorImplementation struct {
	name       string
	calculator Calculator
}

func newCalculatorsBeforeHelicon(config Config) []calculatorImplementation {
	heliconTime := time.Time{}.Add(time.Second)
	return []calculatorImplementation{
		{
			name:       "calculator",
			calculator: NewCalculator(config),
		},
		{
			name: "primary_network_before_helicon",
			calculator: NewPrimaryNetworkCalculator(
				config,
				upgradetest.GetConfigWithUpgradeTime(upgradetest.Helicon, heliconTime),
			),
		},
	}
}

func TestLongerDurationBonus(t *testing.T) {
	for _, impl := range newCalculatorsBeforeHelicon(defaultConfig) {
		t.Run(impl.name, func(t *testing.T) {
			shortDuration := 24 * time.Hour
			totalDuration := 365 * 24 * time.Hour
			shortBalance := units.KiloAvax
			for i := 0; i < int(totalDuration/shortDuration); i++ {
				reward := impl.calculator.Calculate(time.Time{}, shortDuration, shortBalance, 359*units.MegaAvax+shortBalance)
				shortBalance += reward
			}
			reward := impl.calculator.Calculate(time.Time{}, totalDuration%shortDuration, shortBalance, 359*units.MegaAvax+shortBalance)
			shortBalance += reward

			longBalance := units.KiloAvax
			longBalance += impl.calculator.Calculate(time.Time{}, totalDuration, longBalance, 359*units.MegaAvax+longBalance)
			// Rewards are always 0 on Flare, so there is no duration bonus.
			require.Equal(t, shortBalance, longBalance)
		})
	}
}

func TestRewards(t *testing.T) {
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
	for _, impl := range newCalculatorsBeforeHelicon(defaultConfig) {
		t.Run(impl.name, func(t *testing.T) {
			for _, test := range tests {
				name := fmt.Sprintf("reward(%s,%d,%d)==%d",
					test.duration,
					test.stakeAmount,
					test.existingAmount,
					test.expectedReward,
				)
				t.Run(name, func(t *testing.T) {
					reward := impl.calculator.Calculate(
						time.Time{},
						test.duration,
						test.stakeAmount,
						test.existingAmount,
					)
					require.Equal(t, test.expectedReward, reward)
				})
			}
		})
	}
}

func TestPrimaryNetworkCalculatorHeliconRewards(t *testing.T) {
	const (
		heliconReductionPeriod = 90 * 24 * time.Hour
		duration               = defaultMinStakingDuration
		amount                 = units.MegaAvax
		supply                 = 360 * units.MegaAvax
	)

	heliconTime := time.Unix(1_000_000, 0)
	upgradeConfig := upgradetest.GetConfigWithUpgradeTime(upgradetest.Helicon, heliconTime)
	config := defaultConfig

	// Rewards are always 0 on Flare; upstream's expected values are replaced
	// with 0. Upstream computes:
	// expectedReward = (config.SupplyCap-supply) *
	// (minRate*config.MintingPeriod +
	// (config.MaxConsumptionRate-minRate)*duration) * amount * duration /
	// (config.MintingPeriod*PercentDenominator) / supply / config.MintingPeriod.
	tests := []struct {
		name               string
		minConsumptionRate uint64
		maxConsumptionRate uint64
		stakeStartTime     time.Time
		expectedReward     uint64
	}{
		{
			// minRate = 10% at the start of the ramp.
			name:               "at_helicon",
			minConsumptionRate: 100_000,
			stakeStartTime:     heliconTime,
			expectedReward:     0,
		},
		{
			// minRate = 10% - (10% - 7.5%) / 3 = 9.1667%.
			name:               "one_third_ramp",
			minConsumptionRate: 100_000,
			stakeStartTime:     heliconTime.Add(heliconReductionPeriod / 3),
			expectedReward:     0,
		},
		{
			// minRate = 7.5% at the end of the ramp.
			name:               "at_ramp_end",
			minConsumptionRate: 100_000,
			stakeStartTime:     heliconTime.Add(heliconReductionPeriod),
			expectedReward:     0,
		},
		{
			// minRate = 8% - (8% - 7.5%) / 2 = 7.75%.
			name:               "custom_min_rate_above_target",
			minConsumptionRate: 80_000,
			stakeStartTime:     heliconTime.Add(heliconReductionPeriod / 2),
			expectedReward:     0,
		},
		{
			// minRate = 1% + (7.5% - 1%) / 2 = 4.25%.
			name:               "custom_min_rate_below_target",
			minConsumptionRate: 10_000,
			stakeStartTime:     heliconTime.Add(heliconReductionPeriod / 2),
			expectedReward:     0,
		},
		{
			name:               "below_target_at_helicon",
			minConsumptionRate: 74_999,
			stakeStartTime:     heliconTime,
			expectedReward:     0,
		},
		{
			// A half-ramped one-unit increase rounds down to zero.
			name:               "below_target_mid_ramp",
			minConsumptionRate: 74_999,
			stakeStartTime:     heliconTime.Add(heliconReductionPeriod / 2),
			expectedReward:     0,
		},
		{
			name:               "below_target_at_ramp_end",
			minConsumptionRate: 74_999,
			stakeStartTime:     heliconTime.Add(heliconReductionPeriod),
			expectedReward:     0,
		},
		{
			name:               "at_target_mid_ramp",
			minConsumptionRate: 75_000,
			stakeStartTime:     heliconTime.Add(heliconReductionPeriod / 2),
			expectedReward:     0,
		},
		{
			name:               "above_target_at_helicon",
			minConsumptionRate: 75_001,
			stakeStartTime:     heliconTime,
			expectedReward:     0,
		},
		{
			// A half-ramped one-unit reduction rounds down to zero.
			name:               "above_target_mid_ramp",
			minConsumptionRate: 75_001,
			stakeStartTime:     heliconTime.Add(heliconReductionPeriod / 2),
			expectedReward:     0,
		},
		{
			name:               "above_target_at_ramp_end",
			minConsumptionRate: 75_001,
			stakeStartTime:     heliconTime.Add(heliconReductionPeriod),
			expectedReward:     0,
		},
		{
			// Preserve minRate because the 7.5% target exceeds maxRate.
			name:               "target_above_max",
			minConsumptionRate: 10_000,
			maxConsumptionRate: 20_000,
			stakeStartTime:     heliconTime.Add(heliconReductionPeriod),
			expectedReward:     0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config
			cfg.MinConsumptionRate = tt.minConsumptionRate
			if tt.maxConsumptionRate != 0 {
				cfg.MaxConsumptionRate = tt.maxConsumptionRate
			}
			c := NewPrimaryNetworkCalculator(cfg, upgradeConfig)

			reward := c.Calculate(
				tt.stakeStartTime,
				duration,
				amount,
				supply,
			)
			require.Equal(t, tt.expectedReward, reward)
		})
	}
}
