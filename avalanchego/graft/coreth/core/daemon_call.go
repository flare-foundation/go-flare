// (c) 2025, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package core

import (
	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core/vm"
	"github.com/holiman/uint256"
)

// DaemonCall executes a contract call for Flare's daemon/mint mechanism. It
// disables tracing, takes a StateDB snapshot before the call, and delegates
// to evm.Call with a zero value.
//
// The snapshot is returned so the caller can revert the combined daemon+mint
// state transition if a subsequent mint fails (see atomicDaemonAndMint in
// daemon.go). Note that evm.Call itself takes an additional inner snapshot
// and handles revert-on-error internally; the outer snapshot returned here
// remains valid in both the success and error paths.
//
// This function replaces the previous EVM.DaemonCall method that lived in
// the Flare libevm fork (see prior go-flare/libevm changes). Keeping this
// logic in coreth allows use of the unmodified upstream ava-labs/libevm.
func DaemonCall(evm *vm.EVM, caller vm.ContractRef, addr common.Address, input []byte, gas uint64) (snapshot int, ret []byte, leftOverGas uint64, err error) {
	// Temporarily disable EVM debugging for the duration of the daemon call.
	oldTracer := evm.Config.Tracer
	defer func() { evm.Config.Tracer = oldTracer }()
	evm.Config.Tracer = nil

	// Snapshot before the call so the entire daemon+mint sequence can be
	// reverted by the caller if mint() subsequently fails.
	snapshot = evm.StateDB.Snapshot()

	// evm.Call handles its own internal snapshot and revert-on-error. Daemon
	// calls never transfer value.
	ret, leftOverGas, err = evm.Call(caller, addr, input, gas, uint256.NewInt(0))

	return snapshot, ret, leftOverGas, err
}
