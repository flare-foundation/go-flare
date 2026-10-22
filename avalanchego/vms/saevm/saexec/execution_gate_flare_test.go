// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package saexec

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/ava-labs/libevm/core/types"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ava-labs/avalanchego/utils/logging/loggingtest"
	"github.com/ava-labs/avalanchego/vms/saevm/blocks"
	"github.com/ava-labs/avalanchego/vms/saevm/hook"
	"github.com/ava-labs/avalanchego/vms/saevm/saedb"

	saehookstest "github.com/ava-labs/avalanchego/vms/saevm/hook/hookstest"
)

// gatedHooks are [hook.Points] that additionally implement the Flare
// [hook.ExecutionGate]: every block presented to the gate is reported on
// [gatedHooks.entered] and held until [gatedHooks.release] is closed or the
// executor's context is cancelled.
type gatedHooks struct {
	*saehookstest.Stub

	entered chan *types.Block
	release chan struct{}

	mu  sync.Mutex
	err error
}

var (
	_ hook.Points        = (*gatedHooks)(nil)
	_ hook.ExecutionGate = (*gatedHooks)(nil)
)

func newGatedHooks(stub *saehookstest.Stub) *gatedHooks {
	return &gatedHooks{
		Stub:    stub,
		entered: make(chan *types.Block, 16),
		release: make(chan struct{}),
	}
}

func (g *gatedHooks) AwaitExecutionPrerequisites(ctx context.Context, b *types.Block) error {
	g.entered <- b
	select {
	case <-g.release:
		return nil
	case <-ctx.Done():
		g.mu.Lock()
		g.err = ctx.Err()
		g.mu.Unlock()
		return ctx.Err()
	}
}

func (g *gatedHooks) gateErr() error {
	g.mu.Lock()
	defer g.mu.Unlock()
	return g.err
}

// newGatedExecutor replaces the SUT's [Executor] with one whose hooks are the
// provided gate, sharing the SUT's chain, database and chain-data directory.
// The returned [Executor] is closed (once) during test cleanup.
func newGatedExecutor(t *testing.T, sut *SUT, gate *gatedHooks) (*Executor, func()) {
	t.Helper()
	require.NoErrorf(t, sut.Close(), "%T.Close()", sut)

	src := blocks.Source(sut.chain.GetBlock)
	log := loggingtest.New(t, logging.Warn)
	last := sut.LastExecuted()
	tr, err := saedb.NewTracker(sut.db, sut.saedbConfig, last.PostExecutionStateRoot(), sut.chainDataDir, log)
	require.NoError(t, err, "saedb.NewTracker()")
	e, err := New(last, src.AsHeaderSource(), sut.chainConfig, sut.db, sut.xdb, tr, gate, log, prometheus.NewRegistry())
	require.NoError(t, err, "New()")

	closeOnce := sync.OnceFunc(func() {
		e.Close()
		require.NoErrorf(t, tr.Close(e.LastExecuted().PostExecutionStateRoot()), "%T.Close()", tr)
	})
	t.Cleanup(closeOnce)
	return e, closeOnce
}

func awaitGateEntered(t *testing.T, gate *gatedHooks, want *blocks.Block) {
	t.Helper()
	select {
	case got := <-gate.entered:
		require.Equalf(t, want.Hash(), got.Hash(), "block presented to %T", gate)
	case <-time.After(10 * time.Second):
		t.Fatalf("%T never entered the execution gate", gate)
	}
}

func TestExecutionGateHoldsBlockUntilReleased(t *testing.T) {
	stub := defaultHooks()
	ctx, sut := newSUT(t, withHooks(stub))
	gate := newGatedHooks(stub)
	e, _ := newGatedExecutor(t, sut, gate)

	parent := e.LastExecuted()
	b := sut.chain.NewBlock(t, nil)
	require.NoError(t, e.Enqueue(ctx, b), "Enqueue()")

	awaitGateEntered(t, gate, b)
	// Give the executor a chance to (wrongly) proceed while the gate is held.
	time.Sleep(100 * time.Millisecond)
	require.Falsef(t, b.Executed(), "%T.Executed() while gate is held", b)
	require.Equalf(t, parent.Hash(), e.LastExecuted().Hash(), "%T.LastExecuted() while gate is held", e)

	close(gate.release)
	require.NoErrorf(t, b.WaitUntilExecuted(ctx), "%T.WaitUntilExecuted() after gate release", b)
	require.Equalf(t, b.Hash(), e.LastExecuted().Hash(), "%T.LastExecuted() after gate release", e)
	require.NoError(t, gate.gateErr(), "gate observed a context error")

	// Subsequent blocks pass through the (now open) gate.
	next := sut.chain.NewBlock(t, nil)
	require.NoError(t, e.Enqueue(ctx, next), "Enqueue()")
	awaitGateEntered(t, gate, next)
	require.NoErrorf(t, next.WaitUntilExecuted(ctx), "%T.WaitUntilExecuted()", next)
}

func TestExecutionGateAbortedByClose(t *testing.T) {
	stub := defaultHooks()
	ctx, sut := newSUT(t, withHooks(stub))
	gate := newGatedHooks(stub) // never released
	e, closeExecutor := newGatedExecutor(t, sut, gate)

	parent := e.LastExecuted()
	b := sut.chain.NewBlock(t, nil)
	require.NoError(t, e.Enqueue(ctx, b), "Enqueue()")
	awaitGateEntered(t, gate, b)

	closed := make(chan struct{})
	go func() {
		defer close(closed)
		closeExecutor()
	}()
	select {
	case <-closed:
	case <-time.After(10 * time.Second):
		t.Fatalf("%T.Close() blocked on a block waiting in the execution gate", e)
	}

	require.Falsef(t, b.Executed(), "%T.Executed() after shutdown", b)
	require.Equalf(t, parent.Hash(), e.LastExecuted().Hash(), "%T.LastExecuted() after shutdown", e)
	require.ErrorIs(t, gate.gateErr(), context.Canceled, "gate context error")

	select {
	case <-e.done:
	default:
		t.Errorf("%T.done not closed after Close()", e)
	}
}
