// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package core

import ethcore "github.com/ava-labs/libevm/core"

// The legacy (pre-Helicon) StateTransition in state_transition.go is coreth's
// copy of libevm's and returns the same sentinel errors. coreth's core/error.go
// was deleted together with the pre-SAE VM, so the names resolve to libevm's
// values directly; callers comparing against either package see the same
// errors.
var (
	ErrNonceTooLow                  = ethcore.ErrNonceTooLow
	ErrNonceTooHigh                 = ethcore.ErrNonceTooHigh
	ErrNonceMax                     = ethcore.ErrNonceMax
	ErrGasUintOverflow              = ethcore.ErrGasUintOverflow
	ErrIntrinsicGas                 = ethcore.ErrIntrinsicGas
	ErrInsufficientFunds            = ethcore.ErrInsufficientFunds
	ErrInsufficientFundsForTransfer = ethcore.ErrInsufficientFundsForTransfer
	ErrSenderNoEOA                  = ethcore.ErrSenderNoEOA
	ErrTipAboveFeeCap               = ethcore.ErrTipAboveFeeCap
	ErrTipVeryHigh                  = ethcore.ErrTipVeryHigh
	ErrFeeCapVeryHigh               = ethcore.ErrFeeCapVeryHigh
	ErrFeeCapTooLow                 = ethcore.ErrFeeCapTooLow
	ErrBlobFeeCapTooLow             = ethcore.ErrBlobFeeCapTooLow
	ErrMissingBlobHashes            = ethcore.ErrMissingBlobHashes
	ErrBlobTxCreate                 = ethcore.ErrBlobTxCreate
)
