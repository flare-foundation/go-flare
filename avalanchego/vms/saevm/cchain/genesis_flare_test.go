// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package cchain

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/upgrade"

	avalanchegenesis "github.com/ava-labs/avalanchego/genesis"
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
