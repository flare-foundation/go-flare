// Copyright (C) 2019-2024, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package reward

import (
	"math/big"
	"time"

	"github.com/ava-labs/avalanchego/utils/math"
)

var _ Calculator = (*calculator)(nil)

type Calculator interface {
	Calculate(stakedDuration time.Duration, stakedAmount, currentSupply uint64) uint64
}

type calculator struct {
	maxSubMinConsumptionRate *big.Int
	minConsumptionRate       *big.Int
	mintingPeriod            *big.Int
	supplyCap                uint64
}

func NewCalculator(c Config) Calculator {
	return &calculator{
		maxSubMinConsumptionRate: new(big.Int).SetUint64(c.MaxConsumptionRate - c.MinConsumptionRate),
		minConsumptionRate:       new(big.Int).SetUint64(c.MinConsumptionRate),
		mintingPeriod:            new(big.Int).SetUint64(uint64(c.MintingPeriod)),
		supplyCap:                c.SupplyCap,
	}
}

// Reward returns the amount of tokens to reward the staker with.
func (c *calculator) Calculate(stakedDuration time.Duration, stakedAmount, currentSupply uint64) uint64 {
	return uint64(0)
}

// Split [totalAmount] into [totalAmount * shares percentage] and the remainder.
//
// Invariant: [shares] <= [PercentDenominator]
func Split(totalAmount uint64, shares uint32) (uint64, uint64) {
	remainderShares := PercentDenominator - uint64(shares)
	remainderAmount := remainderShares * (totalAmount / PercentDenominator)

	// Delay rounding as long as possible for small numbers
	if optimisticReward, err := math.Mul64(remainderShares, totalAmount); err == nil {
		remainderAmount = optimisticReward / PercentDenominator
	}

	amountFromShares := totalAmount - remainderAmount
	return amountFromShares, remainderAmount
}
