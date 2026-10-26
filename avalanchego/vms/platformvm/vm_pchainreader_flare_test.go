// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package platformvm

import (
	"sync"
	"testing"

	"github.com/ava-labs/libevm/common"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/pchain"
	"github.com/ava-labs/avalanchego/upgrade/upgradetest"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/crypto/bls/signer/localsigner"
	"github.com/ava-labs/avalanchego/utils/set"
	"github.com/ava-labs/avalanchego/vms/platformvm/config"
	"github.com/ava-labs/avalanchego/vms/platformvm/genesis/genesistest"
	"github.com/ava-labs/avalanchego/vms/platformvm/platform"
	"github.com/ava-labs/avalanchego/vms/platformvm/reward"
	"github.com/ava-labs/avalanchego/vms/platformvm/signer"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
)

// TestNewPChainReaderBeforeInitialize mirrors chains/manager, which wires the
// reader while creating the P-chain, before the VM is initialized.
func TestNewPChainReaderBeforeInitialize(t *testing.T) {
	vm := &VM{Internal: config.Internal{UpgradeConfig: upgradetest.GetConfig(upgradetest.Latest)}}
	require.NotNil(t, vm.NewPChainReader(&sync.Mutex{}))
}

// TestPChainReaderOverVM reads an accepted AddPermissionlessValidatorTx through
// the reader the VM hands to other chains, including the owners of its
// genesis-funded inputs.
func TestPChainReaderOverVM(t *testing.T) {
	require := require.New(t)
	vm, _, _ := defaultVM(t, upgradetest.Latest)
	vm.ctx.Lock.Lock()
	defer vm.ctx.Lock.Unlock()

	wallet := newWallet(t, vm, walletConfig{})

	var (
		endTime      = vm.clock.Time().Add(defaultMinStakingDuration)
		nodeID       = ids.GenerateTestNodeID()
		rewardsOwner = &secp256k1fx.OutputOwners{
			Threshold: 1,
			Addrs:     []ids.ShortID{ids.GenerateTestShortID()},
		}
	)

	sk, err := localsigner.New()
	require.NoError(err)
	pop, err := signer.NewProofOfPossession(sk)
	require.NoError(err)

	tx, err := wallet.IssueAddPermissionlessValidatorTx(
		&platform.SubnetValidator{
			Validator: platform.Validator{
				NodeID: nodeID,
				End:    uint64(endTime.Unix()),
				Wght:   vm.MinValidatorStake,
			},
			Subnet: constants.PrimaryNetworkID,
		},
		pop,
		vm.ctx.AVAXAssetID,
		rewardsOwner,
		rewardsOwner,
		reward.PercentDenominator,
	)
	require.NoError(err)

	vm.ctx.Lock.Unlock()
	require.NoError(vm.issueTxFromRPC(tx))
	vm.ctx.Lock.Lock()
	require.NoError(buildAndAcceptStandardBlock(vm))

	// The reader takes the P-chain lock around every state access, exactly
	// as chains/manager wires it; release the lock held by this test.
	reader := vm.NewPChainReader(&vm.ctx.Lock)
	vm.ctx.Lock.Unlock()
	defer vm.ctx.Lock.Lock()

	// defaultVM already accepted a subnet-creation block, so the staking
	// block is the last accepted one.
	last, err := reader.LastAcceptedHeight()
	require.NoError(err)
	require.GreaterOrEqual(last, uint64(1))

	// The genesis block is an Apricot commit block without transactions.
	genesisReader, err := reader.GetBlock(0, nil)
	require.NoError(err)
	genesisBlk := genesisReader.Block()
	require.Equal(pchain.BlockCommit, genesisBlk.Kind)
	require.Zero(genesisBlk.Height)
	require.Empty(genesisBlk.Txs)

	parentReader, err := reader.GetBlock(last-1, nil)
	require.NoError(err)
	parent := parentReader.Block()

	blkReader, err := reader.GetBlock(last, nil)
	require.NoError(err)
	blk := blkReader.Block()
	require.Equal(pchain.BlockStandard, blk.Kind)
	require.Equal(last, blk.Height)
	require.Equal(parent.ID, blk.ParentID)
	require.Len(blk.Txs, 1)

	got := blk.Txs[0]
	require.Equal(pchain.TxAddPermissionlessValidator, got.Type)
	require.Equal(tx.ID(), got.ID)
	require.Equal(nodeID, got.NodeID)
	require.Equal(blk.Timestamp, got.StartTime) // Durango: the chain time at acceptance
	require.Equal(uint64(endTime.Unix()), got.EndTime)
	require.Equal(vm.MinValidatorStake, got.Weight)
	require.Equal(uint32(reward.PercentDenominator), got.DelegationShares)
	require.NotZero(got.InputCount)
	require.NotZero(got.OutputCount)

	// The stake is funded either from genesis UTXOs, whose owners are
	// resolved from the genesis allocation rather than from a producing
	// transaction, or from the change of the subnet-creation transaction
	// defaultVM accepted (the wallet picks UTXOs in ID order, so which ones
	// are spent varies between runs); both are owned by the funded keys.
	inputs, total, err := blkReader.GetInputs(0, 0, 64, nil)
	require.NoError(err)
	require.Equal(got.InputCount, total)
	require.Len(inputs, int(total))

	fundedAddrs := set.NewSet[ids.ShortID](len(genesistest.DefaultFundedKeys))
	fundedEthAddrs := make(map[ids.ShortID]common.Address, len(genesistest.DefaultFundedKeys))
	for _, key := range genesistest.DefaultFundedKeys {
		fundedAddrs.Add(key.Address())
		fundedEthAddrs[key.Address()] = key.EthAddress()
	}
	// The wallet signed the unsigned bytes the reader exposes: every input's
	// signature recovers to its (funded) owner, whose C-chain address the
	// gateway derives from the recovered key.
	require.Equal(tx.Unsigned.Bytes(), got.UnsignedBytes)
	hashes := secp256k1fx.NewSigningHashes(got.UnsignedBytes)
	var funded uint64
	for _, in := range inputs {
		if !in.Genesis {
			require.Equal(testSubnet1.ID(), in.UTXOTxID)
		}
		require.True(in.Resolved)
		require.Len(in.Addresses, 1)
		require.True(fundedAddrs.Contains(in.Addresses[0]))
		require.Equal([]uint32{0}, in.SigIndices)
		require.Len(in.Signatures, 1)
		pk, err := hashes.RecoverSigner(nil, in.Signatures[0][:], in.Addresses[0])
		require.NoError(err)
		require.Equal(fundedEthAddrs[in.Addresses[0]], pk.EthAddress())
		funded += in.Amount
	}
	require.GreaterOrEqual(funded, vm.MinValidatorStake)

	// The produced outputs are the wallet's change outputs followed by the
	// stake outputs, at the indexes the P-chain assigns the UTXOs; all are
	// owned by the funded keys and the stake adds up to the weight.
	outputs, outputTotal, err := blkReader.GetOutputs(0, 0, 64)
	require.NoError(err)
	require.Equal(got.OutputCount, outputTotal)
	require.Len(outputs, int(outputTotal))
	var (
		staked    uint64
		seenStake bool
	)
	for _, out := range outputs {
		require.Len(out.Addresses, 1)
		require.True(fundedAddrs.Contains(out.Addresses[0]))
		switch out.Kind {
		case pchain.OutputStake:
			seenStake = true
			staked += out.Amount
		case pchain.OutputTransfer:
			require.False(seenStake, "change outputs precede stake outputs")
		default:
			require.FailNowf("unexpected output kind", "%d", out.Kind)
		}
	}
	require.Equal(got.Weight, staked)

	_, err = reader.GetBlock(last+1, nil)
	require.ErrorIs(err, pchain.ErrNotAccepted)
}
