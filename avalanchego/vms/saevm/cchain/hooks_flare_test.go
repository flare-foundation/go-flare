// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package cchain

import (
	"math/big"
	"testing"
	"time"

	"github.com/ava-labs/libevm/core/types"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/graft/coreth/params/extras"
	"github.com/ava-labs/avalanchego/graft/coreth/plugin/evm/customtypes"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/snow/snowtest"
	"github.com/ava-labs/avalanchego/upgrade"
	"github.com/ava-labs/avalanchego/upgrade/upgradetest"
	"github.com/ava-labs/avalanchego/utils"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/vms/saevm/cchain/dynamic"
	"github.com/ava-labs/avalanchego/vms/saevm/cchain/tx/txtest"
)

// TestPriceExponentFlareSeed verifies the fallback of [priceExponent] for
// headers that predate Helicon and therefore carry no MinPriceExponent: on
// Flare-family networks it is seeded to [dynamic.FlareInitialPriceExponent]
// (the granite 500 GWei floor), everywhere else it keeps the upstream
// [dynamic.InitialPriceExponent] (1 wei). A header that does carry the
// exponent always wins, regardless of network.
func TestPriceExponentFlareSeed(t *testing.T) {
	const carried dynamic.PriceExponent = 7
	withExponent := customtypes.WithHeaderExtra(
		&types.Header{},
		&customtypes.HeaderExtra{
			MinPriceExponent: utils.PointerTo(carried),
		},
	)

	tests := []struct {
		name      string
		networkID uint32
		header    *types.Header
		want      dynamic.PriceExponent
	}{
		{
			name:      "flare_seeds_granite_floor",
			networkID: constants.FlareID,
			header:    &types.Header{},
			want:      dynamic.FlareInitialPriceExponent,
		},
		{
			name:      "non_flare_keeps_upstream_initial",
			networkID: constants.UnitTestID,
			header:    &types.Header{},
			want:      dynamic.InitialPriceExponent,
		},
		{
			name:      "header_exponent_wins_on_flare_family",
			networkID: constants.FlareID,
			header:    withExponent,
			want:      carried,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := &extras.ChainConfig{
				AvalancheContext: extras.AvalancheContext{
					SnowCtx: &snow.Context{NetworkID: test.networkID},
				},
			}
			require.Equal(t, test.want, priceExponent(config, test.header))
		})
	}
}

// TestHeliconTransitionPriceExponent exercises the coreth->SAE transition: the
// first SAE block is built over a pre-Helicon parent that carries no
// MinPriceExponent. On a Flare-family network the block must start at the
// granite 500 GWei floor (both in its ACP-283 exponent and in its base fee,
// which the floor pulls up from the parent's 1 wei), while a non-Flare
// network keeps the upstream 1 wei behaviour.
func TestHeliconTransitionPriceExponent(t *testing.T) {
	tests := []struct {
		name      string
		networkID uint32
		want      dynamic.PriceExponent
	}{
		{
			name:      "flare_family_starts_at_granite_floor",
			networkID: constants.FlareID,
			want:      dynamic.FlareInitialPriceExponent,
		},
		{
			name:      "non_flare_starts_at_upstream_initial",
			networkID: constants.UnitTestID,
			want:      dynamic.InitialPriceExponent,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			key := txtest.NewKey(t)
			// Schedule Helicon after the other upgrades so the genesis block
			// predates it and carries no MinPriceExponent, mirroring the last
			// coreth block on a live network.
			heliconTime := upgrade.InitiallyActiveTime.Add(5 * time.Second)
			timeOpt, _ := withVMTime(heliconTime)
			ctx, sut := newSUT(t,
				withNetworkID(test.networkID),
				withMaxAllocFor(key.EthAddress()),
				withUpgradeTime(upgradetest.Helicon, heliconTime),
				timeOpt,
			)

			// The block needs a transaction, and the op's implied fee cap
			// (burned nAVAX * 1e9 / gas) must clear the 500 GWei floor for it
			// to be includable on the Flare-family variant.
			const exportFee = 1_000_000_000 // nAVAX; fee cap ~ 88,000 GWei
			w := newWallet(key, sut.ctx, sut.Client)
			stx, _ := w.newExportTx(t, snowtest.XChainID, exportFee, txtest.NewTransferOutput(1, key.Address()))

			blk := sut.issueAndExecute(ctx, t, stx)
			header := blk.Header()

			he := customtypes.GetHeaderExtra(header)
			require.NotNilf(t, he.MinPriceExponent, "first SAE block %T.MinPriceExponent", he)
			require.Equalf(t, test.want, *he.MinPriceExponent, "first SAE block %T.MinPriceExponent", he)

			wantBaseFee := new(big.Int).SetUint64(uint64(test.want.Price()))
			require.NotNilf(t, header.BaseFee, "first SAE block %T.BaseFee", header)
			require.Zerof(t, wantBaseFee.Cmp(header.BaseFee), "first SAE block %T.BaseFee: want %v, got %v", header, wantBaseFee, header.BaseFee)
		})
	}
}
