// Copyright (C) 2019-2024, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package sampler

import (
	"math"
	"math/big"
)

type weightedWithoutReplacementGeneric struct {
	u           Uniform
	w           Weighted
	totalWeight uint64
}

func (s *weightedWithoutReplacementGeneric) InitializeWithAdjustedWeights(weights []uint64) error {
	totalUnadjustedWeight := big.NewInt(0)
	for _, weight := range weights {
		totalUnadjustedWeight.Add(totalUnadjustedWeight, new(big.Int).SetUint64(weight))
	}

	var adjustedWeights []uint64
	var totalAdjustedWeight uint64
	if totalUnadjustedWeight.IsUint64() {
		adjustedWeights = weights
		totalAdjustedWeight = totalUnadjustedWeight.Uint64()
	} else {
		// Adjust weights to fit within uint64
		adjustedWeights = make([]uint64, len(weights))
		totalAdjustedWeight = 0
		weightFactor := totalUnadjustedWeight.Div(totalUnadjustedWeight, new(big.Int).SetUint64(math.MaxUint64)).Uint64() + 1
		for i, weight := range weights {
			adjustedWeights[i] = weight / weightFactor
			totalAdjustedWeight += adjustedWeights[i]
		}
	}
	s.totalWeight = totalAdjustedWeight
	s.u.Initialize(totalAdjustedWeight)
	return s.w.Initialize(adjustedWeights)
}

func (s *weightedWithoutReplacementGeneric) Sample(count int) ([]int, bool) {
	s.u.Reset()

	indices := make([]int, count)
	for i := 0; i < count; i++ {
		weight, ok := s.u.Next()
		if !ok {
			return nil, false
		}

		indices[i], ok = s.w.Sample(weight)
		if !ok {
			return nil, false
		}
	}
	return indices, true
}

func (s *weightedWithoutReplacementGeneric) TotalAdjustedWeight() uint64 {
	return s.totalWeight
}
