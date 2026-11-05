// Copyright (C) 2019, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cchain

import (
	"context"
	"math/big"
	"testing"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core/rawdb"
	"github.com/ava-labs/libevm/core/types"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/ava-labs/avalanchego/graft/coreth/plugin/evm/vmtest"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/snow/engine/snowman/block"
	"github.com/ava-labs/avalanchego/upgrade"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/logging"

	avalanchegenesis "github.com/ava-labs/avalanchego/genesis"
	corethparams "github.com/ava-labs/avalanchego/graft/coreth/params"
	corethfactory "github.com/ava-labs/avalanchego/graft/coreth/plugin/factory"
)

// corethVMGoleakOptions ignores the goroutines the Coreth VM leaves behind
// after [block.ChainVM.Shutdown]: its gas price oracle ranges over
// subscription channels that are never closed, the ephemeral keystore's
// account manager and file watcher are never stopped, and the tx gossip
// subscription can be orphaned by a shutdown race (coreth's own VM tests do
// not run goleak). Nothing in the SAE VM starts these, so ignoring them cannot
// hide a leak in the code under test here.
func corethVMGoleakOptions() []goleak.Option {
	return []goleak.Option{
		goleak.IgnoreTopFunction("github.com/ava-labs/avalanchego/graft/coreth/eth/gasprice.NewOracle.func1"),
		goleak.IgnoreTopFunction("github.com/ava-labs/avalanchego/graft/coreth/eth/gasprice.newFeeInfoProvider.func1"),
		// initBlockBuilding subscribes GossipEthTxPool to the tx pool from a
		// goroutine. When Shutdown closes the pool first, SubscriptionScope.Track
		// returns nil without unsubscribing the joined subscription, so its two
		// goroutines (the joiner and its per-subpool waiter) block forever. Seen
		// in CI under CPU starvation; reproducible with GOMAXPROCS=1 -cpu 1.
		goleak.IgnoreTopFunction("github.com/ava-labs/avalanchego/graft/coreth/core/txpool.(*TxPool).SubscribeTransactions.JoinSubscriptions.func1"),
		goleak.IgnoreTopFunction("github.com/ava-labs/avalanchego/graft/coreth/core/txpool.(*TxPool).SubscribeTransactions.JoinSubscriptions.func1.2"),
		goleak.IgnoreTopFunction("github.com/ava-labs/libevm/accounts.(*Manager).update"),
		goleak.IgnoreTopFunction("github.com/ava-labs/libevm/accounts/keystore.(*KeyStore).updater"),
		goleak.IgnoreTopFunction("github.com/ava-labs/libevm/accounts/keystore.(*watcher).loop"),
		// The keystore watcher's fsnotify backend blocks in a syscall; match
		// the backend frame rather than the platform-specific top function.
		goleak.IgnoreAnyFunction("github.com/fsnotify/fsnotify.(*kqueue).readEvents"),
		goleak.IgnoreAnyFunction("github.com/fsnotify/fsnotify.(*inotify).readEvents"),
	}
}

// TestGenesisMatchesCoreth verifies that, for every Flare and Songbird
// network, the genesis block Coreth commits is the one the SAE VM expects.
//
// The transition VM initializes Coreth first and hands over to SAE at the
// first block whose timestamp reaches the Helicon time. On the local networks
// Helicon is active at the genesis timestamp, so the hand-over happens at
// height 0 on a fresh database: Coreth writes its genesis block, then SAE
// re-derives the genesis from the same bytes and refuses to start if the
// hashes differ. Any drift between Coreth's genesis parsing (precompile
// schedule, header fields) and [parseGenesis]/[genesis.block] therefore makes
// a clean local network unstartable.
func TestGenesisMatchesCoreth(t *testing.T) {
	networkIDs := []uint32{
		constants.FlareID,
		constants.CostwoID,
		constants.LocalFlareID,
		constants.SongbirdID,
		constants.CostonID,
		constants.LocalID,
	}
	for _, networkID := range networkIDs {
		t.Run(constants.NetworkName(networkID), func(t *testing.T) {
			var (
				upgrades = upgrade.GetConfig(networkID)
				genesis  = avalanchegenesis.GetConfig(networkID).CChainGenesis
			)

			// The SAE genesis, as [verifyAndWriteBlock] would compare it with
			// the stored block.
			ctx := &snow.Context{
				NetworkID:       networkID,
				NetworkUpgrades: upgrades,
			}
			g, err := parseGenesis(ctx, []byte(genesis))
			require.NoErrorf(t, err, "parseGenesis(%s)", genesis)
			want, err := g.block()
			require.NoErrorf(t, err, "%T.block()", g)

			// The genesis Coreth commits on a fresh database, built by the
			// same factory the node hands to the transition VM.
			factory := &corethfactory.Factory{}
			intf, err := factory.New(logging.NoLog{})
			require.NoErrorf(t, err, "%T.New()", factory)
			vm, ok := intf.(block.ChainVM)
			require.Truef(t, ok, "%T.New() returned %T, want block.ChainVM", factory, intf)

			vmtest.SetupTestVM(t, vm, vmtest.TestVMConfig{
				Upgrades:    &upgrades,
				GenesisJSON: genesis,
			})
			t.Cleanup(func() {
				// [testing.T.Context] is already cancelled when cleanups run.
				require.NoErrorf(t, vm.Shutdown(context.Background()), "%T.Shutdown()", vm)
			})

			lastAcceptedID, err := vm.LastAccepted(t.Context())
			require.NoErrorf(t, err, "%T.LastAccepted()", vm)
			got, err := vm.GetBlock(t.Context(), lastAcceptedID)
			require.NoErrorf(t, err, "%T.GetBlock(%s)", vm, lastAcceptedID)
			require.Zerof(t, got.Height(), "%T.LastAccepted() height", vm)

			require.Equalf(t, want.Hash(), common.Hash(lastAcceptedID),
				"Coreth genesis hash differs from the SAE genesis hash (SAE state root %s)", want.Root())
		})
	}
}

// TestFlareForkBlocks pins the Berlin (AP2) and London (AP3) activation
// heights that [parseGenesis] derives for the Flare and Songbird networks and
// checks them against coreth's [corethparams.SetEthUpgrades], which pins them
// in the chain config the pre-Helicon node stores. Songbird and Coston
// activated AP3 mid-chain. The transition VM starts the SAE VM on coreth's
// database, where [verifyAndWriteBlock] checks the derived config against the
// stored one, so a mismatch stops the C-chain at Helicon.
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
			ctx := &snow.Context{NetworkUpgrades: test.upgrades}
			g, err := parseGenesis(ctx, []byte(test.genesis))
			require.NoError(err)
			require.Equal(big.NewInt(test.wantChainID), g.Config.ChainID)
			require.Equal(big.NewInt(test.wantBerlin), g.Config.BerlinBlock, "BerlinBlock")
			require.Equal(big.NewInt(test.wantLondon), g.Config.LondonBlock, "LondonBlock")

			// Coreth's view of the same chain. For the Flare and Songbird
			// chain IDs SetEthUpgrades depends on the chain ID only.
			coreth := corethparams.Copy(g.Config)
			require.NoError(corethparams.SetEthUpgrades(&coreth))
			require.Zerof(coreth.BerlinBlock.Cmp(g.Config.BerlinBlock), "BerlinBlock: coreth SetEthUpgrades %v, SAE %v", coreth.BerlinBlock, g.Config.BerlinBlock)
			require.Zerof(coreth.LondonBlock.Cmp(g.Config.LondonBlock), "LondonBlock: coreth SetEthUpgrades %v, SAE %v", coreth.LondonBlock, g.Config.LondonBlock)

			// The Helicon hand-over: the SAE VM starts on the database coreth
			// wrote, and verifyAndWriteBlock checks the derived config against
			// the stored one at the head block. Model the stored config as the
			// SAE config after SetEthUpgrades (TestGenesisMatchesCoreth covers
			// the rest of the derivation) and check at a head past every block
			// fork and every scheduled timestamp fork.
			db := rawdb.NewMemoryDatabase()
			require.NoError(g.verifyAndWriteBlock(db))
			block, err := g.block()
			require.NoError(err)
			rawdb.WriteChainConfig(db, block.Hash(), &coreth)
			head := &types.Header{
				Number: big.NewInt(1 << 62),
				Time:   uint64(upgrade.UnscheduledActivationTime.Unix()) - 1,
			}
			rawdb.WriteHeader(db, head)
			rawdb.WriteHeadHeaderHash(db, head.Hash())

			g, err = parseGenesis(ctx, []byte(test.genesis))
			require.NoError(err)
			require.NoError(g.verifyAndWriteBlock(db), "verifyAndWriteBlock on coreth's database")
		})
	}
}
