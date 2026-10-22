// Copyright (C) 2019, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package cchain

import (
	"context"
	"testing"

	"github.com/ava-labs/libevm/common"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"

	"github.com/ava-labs/avalanchego/graft/coreth/plugin/evm/vmtest"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/snow/engine/snowman/block"
	"github.com/ava-labs/avalanchego/upgrade"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/logging"

	avalanchegenesis "github.com/ava-labs/avalanchego/genesis"
	corethfactory "github.com/ava-labs/avalanchego/graft/coreth/plugin/factory"
)

// corethVMGoleakOptions ignores the goroutines the Coreth VM leaves behind
// after [block.ChainVM.Shutdown]: its gas price oracle ranges over
// subscription channels that are never closed, and the ephemeral keystore's
// account manager and file watcher are never stopped (coreth's own tests do
// not run goleak). Nothing in the SAE VM starts these, so ignoring them cannot
// hide a leak in the code under test here.
func corethVMGoleakOptions() []goleak.Option {
	return []goleak.Option{
		goleak.IgnoreTopFunction("github.com/ava-labs/avalanchego/graft/coreth/eth/gasprice.NewOracle.func1"),
		goleak.IgnoreTopFunction("github.com/ava-labs/avalanchego/graft/coreth/eth/gasprice.newFeeInfoProvider.func1"),
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
