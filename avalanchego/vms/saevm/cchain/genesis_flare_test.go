// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package cchain

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/graft/coreth/params/extras"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/upgrade"

	avalanchegenesis "github.com/ava-labs/avalanchego/genesis"
	corethparams "github.com/ava-labs/avalanchego/graft/coreth/params"
)

// TestFlareForkBlocks pins the Berlin (AP2) and London (AP3) activation
// heights that [parseGenesis] derives for the Flare and Songbird networks.
// Songbird and Coston activated AP3 mid-chain; the heights used to come from
// coreth's SetEthUpgrades (deleted with the pre-SAE VM) and MUST match the
// chain config the pre-Helicon node stored, or the SAE VM refuses to start on
// an existing database.
func TestFlareForkBlocks(t *testing.T) {
	tests := []struct {
		name        string
		upgrades    upgrade.Config
		genesis     string
		wantChainID int64
		wantBerlin  int64
		wantLondon  int64
	}{
		{
			name:        "flare",
			upgrades:    upgrade.Flare,
			genesis:     avalanchegenesis.FlareConfig.CChainGenesis,
			wantChainID: 14,
		},
		{
			name:        "songbird",
			upgrades:    upgrade.Songbird,
			genesis:     avalanchegenesis.SongbirdConfig.CChainGenesis,
			wantChainID: 19,
			wantLondon:  12_349_716,
		},
		{
			name:        "costwo",
			upgrades:    upgrade.Costwo,
			genesis:     avalanchegenesis.CostwoConfig.CChainGenesis,
			wantChainID: 114,
		},
		{
			name:        "coston",
			upgrades:    upgrade.Coston,
			genesis:     avalanchegenesis.CostonConfig.CChainGenesis,
			wantChainID: 16,
			wantLondon:  55_188,
		},
		{
			name:        "localflare",
			upgrades:    upgrade.LocalFlare,
			genesis:     avalanchegenesis.LocalFlareConfig.CChainGenesis,
			wantChainID: 162,
		},
		{
			name:        "local",
			upgrades:    upgrade.Local,
			genesis:     avalanchegenesis.LocalConfig.CChainGenesis,
			wantChainID: 4294967295,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require := require.New(t)
			g, err := parseGenesis(&snow.Context{NetworkUpgrades: test.upgrades}, []byte(test.genesis))
			require.NoError(err)
			require.Equal(big.NewInt(test.wantChainID), g.Config.ChainID)
			require.Equal(big.NewInt(test.wantBerlin), g.Config.BerlinBlock, "BerlinBlock")
			require.Equal(big.NewInt(test.wantLondon), g.Config.LondonBlock, "LondonBlock")
		})
	}
}

// TestFlareNetworkUpgradesMatchStoredConfig checks, for every Flare and
// Songbird network, that the upgrade schedule [parseGenesis] derives is
// exactly the one the pre-Helicon coreth VM stored in the chain database
// (extras.GetNetworkUpgrades, coreth's deleted plugin/evm/vm.go parseGenesis)
// and that the stored config accepts it. The SAE VM runs CheckCompatible
// against the stored config on every start ([genesis.verifyAndWriteBlock]), so
// a fork the SAE schedule lacks refuses to start the VM. The Songbird
// transition — an optional fork that only Songbird, Coston and Local carry —
// was missing until 2026-11-11, when Coston failed its Helicon transition with
// "incompatible chain config: mismatching SongbirdTransition fork block
// timestamp in database".
func TestFlareNetworkUpgradesMatchStoredConfig(t *testing.T) {
	tests := []struct {
		name     string
		upgrades upgrade.Config
		genesis  string
	}{
		{name: "flare", upgrades: upgrade.Flare, genesis: avalanchegenesis.FlareConfig.CChainGenesis},
		{name: "songbird", upgrades: upgrade.Songbird, genesis: avalanchegenesis.SongbirdConfig.CChainGenesis},
		{name: "costwo", upgrades: upgrade.Costwo, genesis: avalanchegenesis.CostwoConfig.CChainGenesis},
		{name: "coston", upgrades: upgrade.Coston, genesis: avalanchegenesis.CostonConfig.CChainGenesis},
		{name: "localflare", upgrades: upgrade.LocalFlare, genesis: avalanchegenesis.LocalFlareConfig.CChainGenesis},
		{name: "local", upgrades: upgrade.Local, genesis: avalanchegenesis.LocalConfig.CChainGenesis},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require := require.New(t)
			ctx := &snow.Context{NetworkUpgrades: test.upgrades}
			g, err := parseGenesis(ctx, []byte(test.genesis))
			require.NoError(err)

			want := extras.GetNetworkUpgrades(test.upgrades)
			require.Equal(want, corethparams.GetExtra(g.Config).NetworkUpgrades)

			// The config the pre-Helicon node stored: an independent parse of
			// the same genesis carrying coreth's schedule.
			stored, err := parseGenesis(ctx, []byte(test.genesis))
			require.NoError(err)
			corethparams.GetExtra(stored.Config).NetworkUpgrades = want

			// The SAE VM is initialised against the Helicon transition block
			// (v1.15.1) or any earlier pre-Helicon head (v1.15.2+, SAE-only);
			// the transition block is the strictest head.
			head := upgrade.UnixTimestamp(test.upgrades.HeliconTime)
			compatErr := stored.Config.CheckCompatible(g.Config, 1, head)
			require.Nil(compatErr, "stored chain config rejects the SAE one: %v", compatErr)

			// Negative control on the networks that carry the transition: the
			// upstream schedule has no Songbird transition and the stored
			// config must reject it — that is what v1.15.0/v1.15.1 shipped.
			if want.SongbirdTransitionTimestamp == nil {
				return
			}
			upstream, err := parseGenesis(ctx, []byte(test.genesis))
			require.NoError(err)
			corethparams.GetExtra(upstream.Config).SongbirdTransitionTimestamp = nil
			compatErr = stored.Config.CheckCompatible(upstream.Config, 1, head)
			require.NotNil(compatErr)
			require.ErrorContains(compatErr, "SongbirdTransition fork block timestamp")
		})
	}
}
