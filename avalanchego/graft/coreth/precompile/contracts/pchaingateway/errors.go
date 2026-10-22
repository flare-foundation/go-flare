// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package pchaingateway

import (
	"errors"
	"fmt"

	"github.com/ava-labs/libevm/accounts/abi"
	"github.com/ava-labs/libevm/core/vm"
	"github.com/ava-labs/libevm/crypto"

	"github.com/ava-labs/avalanchego/graft/coreth/precompile/contract"
	"github.com/ava-labs/avalanchego/snow/pchain"
)

// Revert reasons for caller errors. They are returned as a Solidity
// Error(string) revert so that calling contracts can handle them.
const (
	reasonHeightNotCommitted    = "PChainGateway: P-chain height not committed"
	reasonHeightAboveBound      = "PChainGateway: height above committed P-chain height"
	reasonInvalidInput          = "PChainGateway: invalid input"
	reasonPageSizeExceedsCap    = "PChainGateway: page size exceeds cap"
	reasonTxIndexOutOfRange     = "PChainGateway: tx index out of range"
	reasonUnsupportedTx         = "PChainGateway: unsupported tx type"
	reasonInputIndexOutOfRange  = "PChainGateway: input index out of range"
	reasonOutputIndexOutOfRange = "PChainGateway: output index out of range"
)

var (
	// errPChainReaderUnavailable is a node-local failure: the chain context
	// carries no P-chain reader (never the case for the in-process C-chain).
	errPChainReaderUnavailable = errors.New("P-chain reader unavailable")
	// errChainContextUnavailable is a node-local failure: the EVM carries no
	// chain context (never the case for the in-process C-chain).
	errChainContextUnavailable = errors.New("chain context unavailable")

	errorSelector = crypto.Keccak256([]byte("Error(string)"))[:4]
	stringType    = mustNewType("string")
)

func mustNewType(t string) abi.Type {
	typ, err := abi.NewType(t, "", nil)
	if err != nil {
		panic(err)
	}
	return typ
}

// revertReason encodes reason as the data of a Solidity Error(string) revert.
func revertReason(reason string) []byte {
	packed, err := abi.Arguments{{Type: stringType}}.Pack(reason)
	if err != nil {
		panic(err) // a string always packs
	}
	return append(append([]byte{}, errorSelector...), packed...)
}

// revert returns a Solidity revert with reason, keeping remainingGas: libevm
// preserves the return data and the gas on vm.ErrExecutionReverted.
func revert(remainingGas uint64, reason string) ([]byte, uint64, error) {
	return revertReason(reason), remainingGas, vm.ErrExecutionReverted
}

// invalidate reports a node-local failure. It marks the whole execution as
// invalid so that canonical execution halts loudly instead of diverging from
// the other validators and RPC replays surface the error; it never becomes a
// Solidity-visible revert that a daemon could swallow.
func invalidate(accessibleState contract.AccessibleState, err error) ([]byte, uint64, error) {
	err = fmt.Errorf("P-chain gateway: %w", err)
	accessibleState.GetPrecompileEnv().InvalidateExecution(err)
	return nil, 0, err
}

// readerError classifies an error returned by the P-chain reader: caller
// errors map to a revert reason, everything else (including a height the
// local P-chain has not accepted yet) is node-local.
func readerError(err error) (reason string, callerError bool) {
	switch {
	case errors.Is(err, pchain.ErrTxIndexOutOfRange):
		return reasonTxIndexOutOfRange, true
	case errors.Is(err, pchain.ErrUnsupportedTx):
		return reasonUnsupportedTx, true
	case errors.Is(err, pchain.ErrInputIndexOutOfRange):
		return reasonInputIndexOutOfRange, true
	case errors.Is(err, pchain.ErrOutputIndexOutOfRange):
		return reasonOutputIndexOutOfRange, true
	default:
		return "", false
	}
}
