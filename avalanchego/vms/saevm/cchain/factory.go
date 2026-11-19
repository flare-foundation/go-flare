// Copyright (C) 2019, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cchain

import (
	"time"

	"github.com/ava-labs/avalanchego/snow/engine/snowman/block"
	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ava-labs/avalanchego/vms"
	"github.com/ava-labs/avalanchego/vms/saevm/adaptor"
)

var _ vms.Factory = (*Factory)(nil)

// Factory creates new C-Chain VMs.
type Factory struct{}

// New creates a new C-Chain VM.
func (*Factory) New(log logging.Logger) (interface{}, error) {
	log.Info("Creating new C-Chain SAE VM")
	vm := &VM{
		pullGossipPeriod: time.Second,
		pushGossipPeriod: 100 * time.Millisecond,
		now:              time.Now,
	}
	type fullVM struct {
		adaptor.ChainVMWithContext
		block.StateSyncableVM
		// Flare: the two wrappers above are interface values and hide every
		// other method of vm, so [VM.GetBlockHeight] is forwarded explicitly.
		// Without it the proposervm proposer lookup falls back to GetBlock.
		block.HeightByIDChainVM
	}
	return fullVM{
		adaptor.Convert(vm),
		adaptor.ConvertStateSync(vm),
		vm,
	}, nil
}
