// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package granite defines the ACP-176 parameter set used on Flare-family
// chains after the Granite upgrade. Values not yet finalized may be tuned
// based on simulation; the constants here are the per-fork knobs and should
// be the only place that ACP-176 parameters are tweaked for Granite.
package granite

import (
	"github.com/ava-labs/avalanchego/graft/evm/utils"
	"github.com/ava-labs/avalanchego/vms/components/gas"
	"github.com/ava-labs/avalanchego/vms/evm/acp176"
)

const (
	// MinGasPrice (M) is the dynamic ACP-176 base fee floor. Diverges from
	// the default acp176 MinGasPrice = 1 Wei.
	MinGasPrice = 500 * utils.GWei

	// MinTargetPerSecond (P) is the target gas/sec at TargetExcess = 0.
	// Equal to the default acp176 value
	MinTargetPerSecond gas.Gas = 1_000_000
	// MaxTargetExcessDiff (Q) caps the per-block change in TargetExcess.
	// Equal to the default acp176 value
	MaxTargetExcessDiff gas.Gas = 1 << 15
	// TargetConversion (D) is the conversion constant used in the Target()
	// formula.
	// Equal to the default acp176 value (MaxTargetChangeRate * MaxTargetExcessDiff = 1024 * 32768).
	// MaxTargetChangeRate is implicit as D / Q.
	TargetConversion gas.Gas = 1024 * 32768

	// TimeToFillCapacity is the seconds it takes to refill the gas capacity
	// from zero to its maximum (C = R * TimeToFillCapacity).
	// Diverges from the default acp176 value of 5.
	TimeToFillCapacity gas.Gas = 4
	// TargetToMax is the multiplier applied to the target rate to obtain the
	// maximum gas rate per second (R = T * TargetToMax).
	// Equal to the default acp176 value
	TargetToMax gas.Gas = 2
	// TargetToPriceUpdateConversion controls how fast the dynamic base fee
	// reacts to excess gas: K = T * TargetToPriceUpdateConversion appears as
	// the denominator in MinGasPrice * e^(Excess / K).
	// Equal to the default acp176 value
	TargetToPriceUpdateConversion gas.Gas = 87
)

// DefaultParams is the ACP-176 parameter set to use on Flare-family chains
// after the Granite upgrade.
var DefaultParams = &acp176.Params{
	MinTargetPerSecond:            MinTargetPerSecond,
	TargetConversion:              TargetConversion,
	MaxTargetExcessDiff:           MaxTargetExcessDiff,
	MinGasPrice:                   MinGasPrice,
	TimeToFillCapacity:            TimeToFillCapacity,
	TargetToMax:                   TargetToMax,
	TargetToPriceUpdateConversion: TargetToPriceUpdateConversion,
}
