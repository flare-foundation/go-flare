// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package rpc

import (
	"math/big"
	"testing"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core/vm"
	"github.com/ava-labs/libevm/eth/tracers/logger"
	"github.com/ava-labs/libevm/params"
	"github.com/stretchr/testify/require"
)

// TestCancellableTracerCancelsEVM pins the trace-timeout cancellation wiring:
// the wrapper must abort the EVM handed to it via CaptureStart, standing in for
// upstream traceTx's vmenv.Cancel(). Without this, a timed-out struct-logger
// trace of a high-gas transaction would run to gas exhaustion instead of
// aborting — the DoS surface debug_traceBlock* exposes. It must hold regardless
// of whether the watchdog fires before or after CaptureStart, since a "0s" or
// tiny timeout can fire the watchdog before execution reaches CaptureStart.
func TestCancellableTracerCancelsEVM(t *testing.T) {
	newEVM := func() *vm.EVM {
		return vm.NewEVM(
			vm.BlockContext{BlockNumber: big.NewInt(0)},
			vm.TxContext{},
			nil, // StructLogger.CaptureStart only stores env; no statedb access
			&params.ChainConfig{ChainID: big.NewInt(1)},
			vm.Config{},
		)
	}

	t.Run("capture_then_cancel", func(t *testing.T) {
		require := require.New(t)
		ct := &cancellableTracer{Tracer: logger.NewStructLogger(nil)}

		evm := newEVM()
		// The EVM hands itself to the tracer as the top call frame.
		ct.CaptureStart(evm, common.Address{}, common.Address{}, false, nil, 0, nil)
		require.False(evm.Cancelled(), "capturing alone must not cancel the EVM")

		ct.cancel()
		require.True(evm.Cancelled(), "cancel aborts the captured EVM's interpreter loop")
	})

	t.Run("cancel_before_capture", func(t *testing.T) {
		require := require.New(t)
		ct := &cancellableTracer{Tracer: logger.NewStructLogger(nil)}

		// The watchdog fires before execution reaches CaptureStart (no EVM yet);
		// cancel must record the intent without panicking.
		require.NotPanics(ct.cancel)

		evm := newEVM()
		require.False(evm.Cancelled())

		// CaptureStart must honour the already-fired cancel, otherwise the
		// transaction runs to completion despite the timeout.
		ct.CaptureStart(evm, common.Address{}, common.Address{}, false, nil, 0, nil)
		require.True(evm.Cancelled(), "CaptureStart must cancel an EVM stored after cancel already fired")
	})
}

