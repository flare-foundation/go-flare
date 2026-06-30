// Copyright (C) 2019, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// ACP176 implements the fee logic specified here:
// https://github.com/avalanche-foundation/ACPs/blob/main/ACPs/176-dynamic-evm-gas-limit-and-price-discovery-updates/README.md
package acp176

import (
	"encoding/binary"
	"errors"
	"fmt"
	"math"
	"math/big"
	"sort"

	"github.com/holiman/uint256"

	"github.com/ava-labs/avalanchego/utils/wrappers"
	"github.com/ava-labs/avalanchego/vms/components/gas"

	safemath "github.com/ava-labs/avalanchego/utils/math"
)

const (
	MinTargetPerSecond  = 1_000_000                                 // P
	TargetConversion    = MaxTargetChangeRate * MaxTargetExcessDiff // D
	MaxTargetExcessDiff = 1 << 15                                   // Q
	MinGasPrice         = 1                                         // M

	TimeToFillCapacity            = 5    // in seconds
	TargetToMax                   = 2    // multiplier to convert from target per second to max per second
	TargetToPriceUpdateConversion = 87   // 87 ~= 60 / ln(2) which makes the price double at most every ~60 seconds
	MaxTargetChangeRate           = 1024 // Controls the rate that the target can change per block.

	TargetToMaxCapacity = TargetToMax * TimeToFillCapacity
	MinMaxPerSecond     = MinTargetPerSecond * TargetToMax
	MinMaxCapacity      = MinMaxPerSecond * TimeToFillCapacity

	StateSize = 3 * wrappers.LongLen

	maxTargetExcess = 1_024_950_627 // TargetConversion * ln(MaxUint64 / MinTargetPerSecond) + 1
)

var ErrStateInsufficientLength = errors.New("insufficient length for fee state")

// Params holds the per-fork ACP-176 parameter set. The package-level
// constants below are the default acp176 parameters; callers may supply
// alternate values via copies of [DefaultParams] to the `*With` method
// variants below.
//
// Every field is consensus-critical: validators that compute different values
// for any of these will reject each other's blocks. Changing any of them
// therefore requires a coordinated hard fork.
//
// Note: MaxTargetChangeRate is implicit — it is the ratio
// TargetConversion / MaxTargetExcessDiff (i.e. D / Q). Tuning either of those
// fields independently changes the effective MaxTargetChangeRate.
type Params struct {
	MinTargetPerSecond            gas.Gas   // P
	TargetConversion              gas.Gas   // D = MaxTargetChangeRate * Q
	MaxTargetExcessDiff           gas.Gas   // Q
	MinGasPrice                   gas.Price // M
	TimeToFillCapacity            gas.Gas   // seconds
	TargetToMax                   gas.Gas   // R = T * TargetToMax
	TargetToPriceUpdateConversion gas.Gas   // K = T * TargetToPriceUpdateConversion
}

// DefaultParams is the default acp176 parameter set, matching the
// package-level constants.
var DefaultParams = &Params{
	MinTargetPerSecond:            MinTargetPerSecond,
	TargetConversion:              TargetConversion,
	MaxTargetExcessDiff:           MaxTargetExcessDiff,
	MinGasPrice:                   MinGasPrice,
	TimeToFillCapacity:            TimeToFillCapacity,
	TargetToMax:                   TargetToMax,
	TargetToPriceUpdateConversion: TargetToPriceUpdateConversion,
}

// State represents the current state of the gas pricing and constraints.
type State struct {
	Gas          gas.State
	TargetExcess gas.Gas // q
}

// ParseState returns the state from the provided bytes. It is the inverse of
// [State.Bytes]. This function allows for additional bytes to be padded at the
// end of the provided bytes.
func ParseState(bytes []byte) (State, error) {
	if len(bytes) < StateSize {
		return State{}, fmt.Errorf("%w: expected at least %d bytes but got %d bytes",
			ErrStateInsufficientLength,
			StateSize,
			len(bytes),
		)
	}

	return State{
		Gas: gas.State{
			Capacity: gas.Gas(binary.BigEndian.Uint64(bytes)),
			Excess:   gas.Gas(binary.BigEndian.Uint64(bytes[wrappers.LongLen:])),
		},
		TargetExcess: gas.Gas(binary.BigEndian.Uint64(bytes[2*wrappers.LongLen:])),
	}, nil
}

// Target returns the target gas consumed per second, `T`, using the default
// acp176 parameters.
//
// Target = MinTargetPerSecond * e^(TargetExcess / TargetConversion)
func (s *State) Target() gas.Gas {
	return s.TargetWith(DefaultParams)
}

// TargetWith returns the target gas consumed per second using a caller-supplied
// parameter set.
func (s *State) TargetWith(p *Params) gas.Gas {
	return gas.Gas(gas.CalculatePrice(
		gas.Price(p.MinTargetPerSecond),
		s.TargetExcess,
		p.TargetConversion,
	))
}

// MaxCapacity returns the maximum possible accrued gas capacity, `C`, using
// the default acp176 parameters.
func (s *State) MaxCapacity() gas.Gas {
	return s.MaxCapacityWith(DefaultParams)
}

// MaxCapacityWith returns the maximum possible accrued gas capacity using a
// caller-supplied parameter set.
func (s *State) MaxCapacityWith(p *Params) gas.Gas {
	targetPerSecond := s.TargetWith(p)
	targetToMaxCapacity := mulWithUpperBound(p.TargetToMax, p.TimeToFillCapacity)
	return mulWithUpperBound(targetPerSecond, targetToMaxCapacity)
}

// GasPrice returns the current required fee per gas using the default acp176
// parameters. Equivalent to GasPriceWith(DefaultParams).
func (s *State) GasPrice() gas.Price {
	return s.GasPriceWith(DefaultParams)
}

// GasPriceWith returns the current required fee per gas using a caller-supplied
// parameter set.
//
// Price = p.MinGasPrice * e^(Excess / (TargetWith(p) * p.TargetToPriceUpdateConversion))
func (s *State) GasPriceWith(p *Params) gas.Price {
	targetPerSecond := s.TargetWith(p)
	priceUpdateConversion := mulWithUpperBound(targetPerSecond, p.TargetToPriceUpdateConversion) // K
	return gas.CalculatePrice(p.MinGasPrice, s.Gas.Excess, priceUpdateConversion)
}

// AdvanceSeconds increases the gas capacity and decreases the gas excess based on
// the elapsed seconds, using the default acp176 parameters.
// This is used in Fortuna.
func (s *State) AdvanceSeconds(seconds uint64) {
	s.AdvanceSecondsWith(DefaultParams, seconds)
}

// AdvanceSecondsWith advances the gas state by `seconds` using a caller-supplied
// parameter set.
func (s *State) AdvanceSecondsWith(p *Params, seconds uint64) {
	targetPerSecond := s.TargetWith(p)
	maxPerSecond := mulWithUpperBound(targetPerSecond, p.TargetToMax)    // R
	maxCapacity := mulWithUpperBound(maxPerSecond, p.TimeToFillCapacity) // C
	s.Gas = s.Gas.AdvanceTime(
		maxCapacity,
		maxPerSecond,
		targetPerSecond,
		seconds,
	)
}

// AdvanceMilliseconds increases the gas capacity and decreases the gas excess based on
// the elapsed milliseconds, using the default acp176 parameters.
// This is used in Granite.
func (s *State) AdvanceMilliseconds(milliseconds uint64) {
	s.AdvanceMillisecondsWith(DefaultParams, milliseconds)
}

// AdvanceMillisecondsWith advances the gas state by `milliseconds` using a
// caller-supplied parameter set.
func (s *State) AdvanceMillisecondsWith(p *Params, milliseconds uint64) {
	targetPerSecond := s.TargetWith(p)
	targetPerMS := targetPerSecond / 1000
	maxPerMS := targetPerMS * p.TargetToMax                              // R - safe when 1000 > p.TargetToMax (true for the default of 2).
	maxPerSecond := mulWithUpperBound(targetPerSecond, p.TargetToMax)    // rate used for calculating maxCapacity
	maxCapacity := mulWithUpperBound(maxPerSecond, p.TimeToFillCapacity) // C
	s.Gas = s.Gas.AdvanceTime(
		maxCapacity,
		maxPerMS,
		targetPerMS,
		milliseconds,
	)
}

// ConsumeGas decreases the gas capacity and increases the gas excess by
// gasUsed + extraGasUsed. If the gas capacity is insufficient, an error is
// returned.
func (s *State) ConsumeGas(
	gasUsed uint64,
	extraGasUsed *big.Int,
) error {
	newGas, err := s.Gas.ConsumeGas(gas.Gas(gasUsed))
	if err != nil {
		return err
	}

	if extraGasUsed == nil {
		s.Gas = newGas
		return nil
	}
	if !extraGasUsed.IsUint64() {
		return fmt.Errorf("%w: extraGasUsed (%d) exceeds MaxUint64",
			gas.ErrInsufficientCapacity,
			extraGasUsed,
		)
	}
	newGas, err = newGas.ConsumeGas(gas.Gas(extraGasUsed.Uint64()))
	if err != nil {
		return err
	}

	s.Gas = newGas
	return nil
}

// UpdateTargetExcess updates the targetExcess to be as close as possible to the
// desiredTargetExcess without exceeding the maximum targetExcess change, using
// the default acp176 parameters.
func (s *State) UpdateTargetExcess(desiredTargetExcess gas.Gas) {
	s.UpdateTargetExcessWith(DefaultParams, desiredTargetExcess)
}

// UpdateTargetExcessWith updates the targetExcess using a caller-supplied
// parameter set; the per-block change is capped by p.MaxTargetExcessDiff.
func (s *State) UpdateTargetExcessWith(p *Params, desiredTargetExcess gas.Gas) {
	previousTargetPerSecond := s.TargetWith(p)
	s.TargetExcess = targetExcess(s.TargetExcess, desiredTargetExcess, p.MaxTargetExcessDiff)
	newTargetPerSecond := s.TargetWith(p)
	s.Gas.Excess = scaleExcess(
		s.Gas.Excess,
		newTargetPerSecond,
		previousTargetPerSecond,
	)

	// Ensure the gas capacity does not exceed the maximum capacity.
	targetToMaxCapacity := mulWithUpperBound(p.TargetToMax, p.TimeToFillCapacity)
	newMaxCapacity := mulWithUpperBound(newTargetPerSecond, targetToMaxCapacity) // C
	s.Gas.Capacity = min(s.Gas.Capacity, newMaxCapacity)
}

// Bytes returns the binary representation of the state.
func (s *State) Bytes() []byte {
	bytes := make([]byte, StateSize)
	binary.BigEndian.PutUint64(bytes, uint64(s.Gas.Capacity))
	binary.BigEndian.PutUint64(bytes[wrappers.LongLen:], uint64(s.Gas.Excess))
	binary.BigEndian.PutUint64(bytes[2*wrappers.LongLen:], uint64(s.TargetExcess))
	return bytes
}

// DesiredTargetExcess calculates the optimal desiredTargetExcess given the
// desired target, using the default acp176 parameters.
func DesiredTargetExcess(desiredTarget gas.Gas) gas.Gas {
	return DesiredTargetExcessWith(DefaultParams, desiredTarget)
}

// DesiredTargetExcessWith calculates the optimal desiredTargetExcess given the
// desired target, using a caller-supplied parameter set.
func DesiredTargetExcessWith(p *Params, desiredTarget gas.Gas) gas.Gas {
	// This could be solved directly by calculating
	// p.TargetConversion * ln(desiredTarget / p.MinTargetPerSecond) using
	// floating point math. However, it introduces inaccuracies. So, we use a
	// binary search to find the closest integer solution. The upper bound
	// [maxTargetExcess] is conservative enough to cover non-default parameter
	// sets that share the same order of magnitude as the default acp176
	// parameters; an excess past it would map to gas/sec > MaxUint64.
	return gas.Gas(sort.Search(maxTargetExcess, func(targetExcessGuess int) bool {
		state := State{
			TargetExcess: gas.Gas(targetExcessGuess),
		}
		return state.TargetWith(p) >= desiredTarget
	}))
}

// targetExcess calculates the optimal new targetExcess for a block proposer to
// include given the current and desired excess values, capped by maxChange.
func targetExcess(excess, desired, maxChange gas.Gas) gas.Gas {
	change := safemath.AbsDiff(excess, desired)
	change = min(change, maxChange)
	if excess < desired {
		return excess + change
	}
	return excess - change
}

// scaleExcess scales the excess during gas target modifications to keep the
// price constant.
func scaleExcess(
	excess,
	newTargetPerSecond,
	previousTargetPerSecond gas.Gas,
) gas.Gas {
	var bigExcess uint256.Int
	bigExcess.SetUint64(uint64(excess))

	var bigTarget uint256.Int
	bigTarget.SetUint64(uint64(newTargetPerSecond))
	bigExcess.Mul(&bigExcess, &bigTarget)

	bigTarget.SetUint64(uint64(previousTargetPerSecond))
	bigExcess.Div(&bigExcess, &bigTarget)
	if !bigExcess.IsUint64() {
		return math.MaxUint64
	}
	return gas.Gas(bigExcess.Uint64())
}

// mulWithUpperBound multiplies two numbers and returns the result. If the
// result overflows, it returns [math.MaxUint64].
func mulWithUpperBound(a, b gas.Gas) gas.Gas {
	product, err := safemath.Mul(a, b)
	if err != nil {
		return math.MaxUint64
	}
	return product
}
