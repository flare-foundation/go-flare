// Copyright (C) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package reward

import (
	"math/big"
	"time"
)

var _ Calculator = &calculator{}

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
