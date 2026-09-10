// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package cchain

import (
	"math/big"
	"testing"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/common/hexutil"
	"github.com/ava-labs/libevm/core/types"
	"github.com/ava-labs/libevm/eth/tracers"
	"github.com/ava-labs/libevm/eth/tracers/native"
	"github.com/ava-labs/libevm/libevm/options"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/utils"
	"github.com/ava-labs/avalanchego/vms/saevm/saetest"

	cparams "github.com/ava-labs/avalanchego/graft/coreth/params"
	ethparams "github.com/ava-labs/libevm/params"
)

// prestateTracer configures the block tracers to report the state each
// transaction read, which pins the intra-block fee settlement.
var prestateTracer = tracers.TraceConfig{Tracer: utils.PointerTo("prestateTracer")}

// TestFlareTraceBlockReplaysFlareState asserts that the block-level tracing
// endpoints (debug_traceBlock*, debug_intermediateRoots) replay transactions
// through the Flare execution pipeline ([saexec.ApplyTransaction]) rather
// than libevm's plain core.ApplyMessage.
//
// On a Flare chain ID, live execution relocates each transaction's full fee
// from the coinbase to the 0x…dEaD burn address after the EVM run. A plain
// upstream replay leaves the fee at the coinbase, so on a block with two
// transactions it diverges observably in both assertions below:
//
//   - the last intermediate root would not equal the block's post-execution
//     state root, and
//   - the prestate of tx[1] would show the coinbase credited with tx[0]'s
//     fee, which the canonical state never contained.
func TestFlareTraceBlockReplaysFlareState(t *testing.T) {
	w := saetest.NewUNSAFEWallet(t, 1, types.LatestSignerForChainID(cparams.LocalFlareChainID))
	sender := w.Addresses()[0]

	ctx, sut := newSUT(t,
		options.Func[sutConfig](func(c *sutConfig) {
			// The Flare fee/daemon mechanisms are chain-ID-gated; the network
			// ID (a test one here) is irrelevant to them.
			c.genesis.Config.ChainID = cparams.LocalFlareChainID
		}),
		withMaxAllocFor(sender),
		withArchival(),
	)

	txs := make([]*types.Transaction, 2)
	for i := range txs {
		txs[i] = w.SetNonceAndSign(t, 0, &types.LegacyTx{
			To:       &sender,
			Gas:      ethparams.TxGas,
			GasPrice: big.NewInt(1),
		})
		require.NoErrorf(t, sut.ethclient.SendTransaction(ctx, txs[i]), "%T.SendTransaction(%d)", sut.ethclient, i)
	}
	sut.waitForPendingEthTxs(ctx, t, txs...)

	blk := sut.runConsensusLoop(ctx, t)
	receipts := blk.Receipts()
	require.Lenf(t, receipts, len(txs), "%T.Receipts()", blk)

	fee0 := new(big.Int).SetUint64(receipts[0].GasUsed)
	fee0.Mul(fee0, receipts[0].EffectiveGasPrice)
	require.Positive(t, fee0.Sign(), "tx[0] must pay a non-zero fee for the burn to be observable")

	// Live execution burned both fees to dEaD; otherwise the assertions below
	// would be vacuous.
	deadAddr := common.HexToAddress("0x000000000000000000000000000000000000dEaD")
	fee1 := new(big.Int).SetUint64(receipts[1].GasUsed)
	fee1.Mul(fee1, receipts[1].EffectiveGasPrice)
	burned := sut.balance(t, deadAddr)
	assert.Equal(t, new(big.Int).Add(fee0, fee1).String(), burned.ToBig().String(), "fees burned to dEaD by live execution")

	t.Run("debug_intermediateRoots", func(t *testing.T) {
		var roots []common.Hash
		require.NoError(t,
			sut.ethclient.Client().CallContext(ctx, &roots, "debug_intermediateRoots", blk.Hash()),
			"debug_intermediateRoots(%s)", blk.Hash())
		require.Len(t, roots, len(txs), "one root per transaction")
		assert.NotEqual(t, roots[0], roots[1], "each transaction changes state")
		// Only holds if the replay applied the Flare fee relocation (and
		// daemon call) of both transactions; the block has no cross-chain txs,
		// so its end-of-block ops are state-neutral.
		assert.Equal(t, blk.PostExecutionStateRoot(), roots[1], "last intermediate root is the post-execution state root")
	})

	t.Run("debug_traceBlockByNumber_prestate", func(t *testing.T) {
		var prestates []struct {
			Result map[common.Address]native.Account `json:"result"`
		}
		require.NoError(t,
			sut.ethclient.Client().CallContext(ctx, &prestates, "debug_traceBlockByNumber", hexutil.Uint64(blk.NumberU64()), prestateTracer),
			"debug_traceBlockByNumber(%d, prestateTracer)", blk.NumberU64())
		require.Len(t, prestates, len(txs), "one prestate per transaction")

		// The prestateTracer always records the coinbase. Live execution
		// relocated tx[0]'s fee from the coinbase to dEaD before tx[1] ran, so
		// tx[1]'s prestate must show the coinbase unchanged; an upstream
		// replay would show it credited with fee0.
		coinbase := blk.EthBlock().Coinbase()
		balance := func(i int) *big.Int {
			acc, ok := prestates[i].Result[coinbase]
			require.Truef(t, ok, "coinbase %s in prestate of tx %d", coinbase, i)
			if acc.Balance == nil { // zero balances are omitted from the JSON
				return new(big.Int)
			}
			return acc.Balance
		}
		assert.Zero(t, balance(0).Cmp(balance(1)), "coinbase must not accrue tx[0]'s fee between transactions: prestate(tx0)=%s, prestate(tx1)=%s", balance(0), balance(1))
	})
}
