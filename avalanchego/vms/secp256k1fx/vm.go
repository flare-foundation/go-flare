// Copyright (C) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package secp256k1fx

import (
	"time"

	"github.com/ava-labs/avalanchego/codec"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ava-labs/avalanchego/utils/timer/mockable"
)

// VM that this Fx must be run by
type VM interface {
	CodecRegistry() codec.Registry
	Clock() *mockable.Clock
	Logger() logging.Logger
	GetTimestamp() time.Time
	GetNetworkID() uint32
}

var _ VM = &TestVM{}

// TestVM is a minimal implementation of a VM
type TestVM struct {
	CLK   mockable.Clock
	Codec codec.Registry
	Log   logging.Logger
}

func (vm *TestVM) Clock() *mockable.Clock        { return &vm.CLK }
func (vm *TestVM) CodecRegistry() codec.Registry { return vm.Codec }
func (vm *TestVM) Logger() logging.Logger        { return vm.Log }
func (vm *TestVM) GetTimestamp() time.Time       { return vm.CLK.Time() }
func (vm *TestVM) GetNetworkID() uint32          { return constants.FlareID }
