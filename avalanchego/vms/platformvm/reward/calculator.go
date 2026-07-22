// Copyright (C) 2019, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package reward

import (
	"math/big"
	"time"

	"github.com/ava-labs/avalanchego/upgrade"
	"github.com/ava-labs/avalanchego/utils/math"
)

var _ Calculator = (*calculator)(nil)

type Calculator interface {
	Calculate(stakeStartTime time.Time, stakedDuration time.Duration, stakedAmount, currentSupply uint64) uint64
}

type calculator struct {
	maxSubMinConsumptionRate *big.Int
	minConsumptionRate       *big.Int
	mintingPeriod            *big.Int
	supplyCap                uint64
}

// NewCalculator returns a calculator for the provided reward config as-is.
// It does not account for reward changes introduced by network upgrades.
func NewCalculator(c Config) Calculator {
	return &calculator{
		maxSubMinConsumptionRate: new(big.Int).SetUint64(c.MaxConsumptionRate - c.MinConsumptionRate),
		minConsumptionRate:       new(big.Int).SetUint64(c.MinConsumptionRate),
		mintingPeriod:            new(big.Int).SetUint64(uint64(c.MintingPeriod)),
		supplyCap:                c.SupplyCap,
	}
}

// Reward returns the amount of tokens to reward the staker with.
func (c *calculator) Calculate(stakeStartTime time.Time, stakedDuration time.Duration, stakedAmount, currentSupply uint64) uint64 {
	return uint64(0)
}

type primaryNetworkCalculator struct {
	config        Config
	upgradeConfig upgrade.Config
}

var _ Calculator = (*primaryNetworkCalculator)(nil)

// NewPrimaryNetworkCalculator returns a calculator for primary network staking
// rewards. It applies primary network reward upgrades.
func NewPrimaryNetworkCalculator(c Config, upgradeConfig upgrade.Config) Calculator {
	return &primaryNetworkCalculator{
		config:        c,
		upgradeConfig: upgradeConfig,
	}
}

func (c *primaryNetworkCalculator) Calculate(stakeStartTime time.Time, stakedDuration time.Duration, stakedAmount, currentSupply uint64) uint64 {
	cfg := configForStakeStart(c.config, c.upgradeConfig, stakeStartTime)
	return NewCalculator(cfg).Calculate(stakeStartTime, stakedDuration, stakedAmount, currentSupply)
}

func configForStakeStart(
	rewardConfig Config,
	upgradeConfig upgrade.Config,
	stakeStartTime time.Time,
) Config {
	const (
		// ACP-285 lowers primary network MinConsumptionRate to 7.5%.
		heliconMinConsumptionRateTarget          uint64 = 75_000
		heliconMinConsumptionRateReductionPeriod        = 90 * 24 * time.Hour
	)

	if !upgradeConfig.IsHeliconActivated(stakeStartTime) ||
		rewardConfig.MinConsumptionRate == heliconMinConsumptionRateTarget {
		return rewardConfig
	}

	// Custom networks may configure a maximum below the Helicon target. Do not
	// increase the minimum above their configured maximum.
	if rewardConfig.MaxConsumptionRate < heliconMinConsumptionRateTarget {
		return rewardConfig
	}

	fullChange := math.AbsDiff(
		rewardConfig.MinConsumptionRate,
		heliconMinConsumptionRateTarget,
	)

	change := fullChange
	reductionEndTime := upgradeConfig.HeliconTime.Add(heliconMinConsumptionRateReductionPeriod)
	if stakeStartTime.Before(reductionEndTime) {
		elapsed := stakeStartTime.Sub(upgradeConfig.HeliconTime)
		rampedChange := new(big.Int).SetUint64(fullChange)
		rampedChange.Mul(rampedChange, big.NewInt(int64(elapsed)))
		rampedChange.Div(rampedChange, big.NewInt(int64(heliconMinConsumptionRateReductionPeriod)))
		change = rampedChange.Uint64()
	}

	if rewardConfig.MinConsumptionRate < heliconMinConsumptionRateTarget {
		rewardConfig.MinConsumptionRate += change
	} else {
		rewardConfig.MinConsumptionRate -= change
	}
	return rewardConfig
}

// Split [totalAmount] into [totalAmount * shares percentage] and the remainder.
//
// Invariant: [shares] <= [PercentDenominator]
func Split(totalAmount uint64, shares uint32) (uint64, uint64) {
	remainderShares := PercentDenominator - uint64(shares)
	remainderAmount := remainderShares * (totalAmount / PercentDenominator)

	// Delay rounding as long as possible for small numbers
	if optimisticReward, err := math.Mul(remainderShares, totalAmount); err == nil {
		remainderAmount = optimisticReward / PercentDenominator
	}

	amountFromShares := totalAmount - remainderAmount
	return amountFromShares, remainderAmount
}
