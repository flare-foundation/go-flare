// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package granite defines the ACP-176 parameter set used on Flare-family
// chains after the Granite upgrade. It equals the default acp176 parameter
// set except for the two overrides defined below. These values are live
// consensus parameters on all Flare-family networks; changing them requires
// a coordinated network upgrade. This is the only place that ACP-176
// parameters are defined for Granite.
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

	// TimeToFillCapacity is the seconds it takes to refill the gas capacity
	// from zero to its maximum (C = R * TimeToFillCapacity).
	// Diverges from the default acp176 value of 5.
	TimeToFillCapacity gas.Gas = 4
)

// DefaultParams is the ACP-176 parameter set to use on Flare-family chains
// after the Granite upgrade. All fields other than MinGasPrice and
// TimeToFillCapacity equal the default acp176 values.
var DefaultParams = &acp176.Params{
	MinTargetPerSecond:            acp176.MinTargetPerSecond,
	TargetConversion:              acp176.TargetConversion,
	MaxTargetExcessDiff:           acp176.MaxTargetExcessDiff,
	MinGasPrice:                   MinGasPrice,
	TimeToFillCapacity:            TimeToFillCapacity,
	TargetToMax:                   acp176.TargetToMax,
	TargetToPriceUpdateConversion: acp176.TargetToPriceUpdateConversion,
}
