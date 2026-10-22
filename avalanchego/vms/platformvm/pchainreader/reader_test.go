// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package pchainreader

import (
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/pchain"
	"github.com/ava-labs/avalanchego/upgrade"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/crypto/secp256k1"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/platformvm/platform"
	"github.com/ava-labs/avalanchego/vms/platformvm/signer"
	"github.com/ava-labs/avalanchego/vms/platformvm/stakeable"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
	"github.com/ava-labs/avalanchego/vms/types"
)

const (
	durangoTime     = 1_000_000
	preDurangoTime  = durangoTime - 10
	postDurangoTime = durangoTime + 10
)

var (
	testUpgrades = upgrade.Config{DurangoTime: time.Unix(durangoTime, 0)}
	avaxAssetID  = ids.GenerateTestID()
	addrA        = ids.ShortID{1}
	addrB        = ids.ShortID{2}
	addrC        = ids.ShortID{3}

	// genesisMarkerID stands in for a test genesis that gives its UTXOs an
	// arbitrary transaction ID instead of ids.Empty.
	genesisMarkerID = ids.ID{0x9e, 0x5e}
)

// fakeStore is an in-memory Store recording how often it is called.
type fakeStore struct {
	lastAccepted ids.ID
	heights      map[uint64]ids.ID
	blocks       map[ids.ID]platform.Block
	txs          map[ids.ID]*platform.Tx
	rewardUTXOs  map[ids.ID][]*avax.UTXO

	calls       int // every store call
	heightCalls int // GetBlockIDAtHeight calls
	bytesCalls  int // GetBlockBytes and GetTxBytes calls
	// staleLastAccepted is returned by the next staleReads GetLastAccepted
	// calls, simulating a tip that moves between the reader's lookups.
	staleLastAccepted ids.ID
	staleReads        int
}

func newFakeStore() *fakeStore {
	return &fakeStore{
		heights:     map[uint64]ids.ID{},
		blocks:      map[ids.ID]platform.Block{},
		txs:         map[ids.ID]*platform.Tx{},
		rewardUTXOs: map[ids.ID][]*avax.UTXO{},
	}
}

func (s *fakeStore) accept(blk platform.Block) {
	s.heights[blk.Height()] = blk.ID()
	s.blocks[blk.ID()] = blk
	s.lastAccepted = blk.ID()
	for _, tx := range blk.Txs() {
		s.txs[tx.ID()] = tx
	}
}

func (s *fakeStore) GetLastAccepted() ids.ID {
	s.calls++
	if s.staleReads > 0 {
		s.staleReads--
		return s.staleLastAccepted
	}
	return s.lastAccepted
}

func (s *fakeStore) GetBlockIDAtHeight(height uint64) (ids.ID, error) {
	s.calls++
	s.heightCalls++
	id, ok := s.heights[height]
	if !ok {
		return ids.Empty, database.ErrNotFound
	}
	return id, nil
}

func (s *fakeStore) GetBlockBytes(blockID ids.ID) ([]byte, error) {
	s.calls++
	s.bytesCalls++
	blk, ok := s.blocks[blockID]
	if !ok {
		return nil, database.ErrNotFound
	}
	return blk.Bytes(), nil
}

func (s *fakeStore) GetTxBytes(txID ids.ID) ([]byte, error) {
	s.calls++
	s.bytesCalls++
	tx, ok := s.txs[txID]
	if !ok {
		return nil, database.ErrNotFound
	}
	return tx.Bytes(), nil
}

func (s *fakeStore) GetRewardUTXOs(txID ids.ID) ([]*avax.UTXO, error) {
	s.calls++
	return s.rewardUTXOs[txID], nil
}

func genesisList(utxos ...*avax.UTXO) func() []*avax.UTXO {
	return func() []*avax.UTXO { return utxos }
}

// countingLocker records Lock/Unlock calls.
type countingLocker struct {
	locks, unlocks int
}

func (l *countingLocker) Lock()   { l.locks++ }
func (l *countingLocker) Unlock() { l.unlocks++ }

func signedTx(t *testing.T, unsigned platform.UnsignedTx) *platform.Tx {
	t.Helper()
	tx, err := platform.NewSignedTx(unsigned, platform.Codec, nil)
	require.NoError(t, err)
	return tx
}

func owners(addrs ...ids.ShortID) secp256k1fx.OutputOwners {
	return secp256k1fx.OutputOwners{Threshold: 1, Addrs: addrs}
}

func transferOutput(amount uint64, o secp256k1fx.OutputOwners) *avax.TransferableOutput {
	return &avax.TransferableOutput{
		Asset: avax.Asset{ID: avaxAssetID},
		Out:   &secp256k1fx.TransferOutput{Amt: amount, OutputOwners: o},
	}
}

func transferInput(utxoTxID ids.ID, index uint32, amount uint64, sigIndices ...uint32) *avax.TransferableInput {
	return &avax.TransferableInput{
		UTXOID: avax.UTXOID{TxID: utxoTxID, OutputIndex: index},
		Asset:  avax.Asset{ID: avaxAssetID},
		In: &secp256k1fx.TransferInput{
			Amt:   amount,
			Input: secp256k1fx.Input{SigIndices: sigIndices},
		},
	}
}

func baseTx(ins []*avax.TransferableInput, outs []*avax.TransferableOutput) platform.BaseTx {
	return platform.BaseTx{BaseTx: avax.BaseTx{
		NetworkID:    constants.UnitTestID,
		BlockchainID: constants.PlatformChainID,
		Ins:          ins,
		Outs:         outs,
		Memo:         types.JSONByteSlice{},
	}}
}

func permissionlessValidator(subnetID ids.ID, nodeID ids.NodeID, ins []*avax.TransferableInput) *platform.AddPermissionlessValidatorTx {
	return &platform.AddPermissionlessValidatorTx{
		BaseTx:                baseTx(ins, nil),
		Validator:             platform.Validator{NodeID: nodeID, Start: 111, End: 222, Wght: 1000},
		Subnet:                subnetID,
		Signer:                &signer.Empty{},
		StakeOuts:             []*avax.TransferableOutput{transferOutput(1000, owners(addrA))},
		ValidatorRewardsOwner: &secp256k1fx.OutputOwners{},
		DelegatorRewardsOwner: &secp256k1fx.OutputOwners{},
		DelegationShares:      20_000,
	}
}

func permissionlessDelegator(subnetID ids.ID, nodeID ids.NodeID, ins []*avax.TransferableInput) *platform.AddPermissionlessDelegatorTx {
	return &platform.AddPermissionlessDelegatorTx{
		BaseTx:                 baseTx(ins, nil),
		Validator:              platform.Validator{NodeID: nodeID, Start: 333, End: 444, Wght: 500},
		Subnet:                 subnetID,
		StakeOuts:              []*avax.TransferableOutput{transferOutput(500, owners(addrB))},
		DelegationRewardsOwner: &secp256k1fx.OutputOwners{},
	}
}

func autoRenewedValidator(nodeID ids.NodeID, ins []*avax.TransferableInput) *platform.AddAutoRenewedValidatorTx {
	return &platform.AddAutoRenewedValidatorTx{
		BaseTx:                   baseTx(ins, nil),
		ValidatorNodeID:          nodeID.Bytes(),
		Signer:                   &signer.Empty{},
		StakeOuts:                []*avax.TransferableOutput{transferOutput(700, owners(addrA)), transferOutput(300, owners(addrB))},
		ValidatorRewardsOwner:    &secp256k1fx.OutputOwners{},
		DelegatorRewardsOwner:    &secp256k1fx.OutputOwners{},
		ValidatorAuthority:       &secp256k1fx.OutputOwners{},
		DelegationShares:         30_000,
		AutoCompoundRewardShares: 250_000,
		Period:                   90 * 24 * 3600,
	}
}

func importTx(sourceChain ids.ID, ins, imported []*avax.TransferableInput, outs ...*avax.TransferableOutput) *platform.ImportTx {
	return &platform.ImportTx{
		BaseTx:         baseTx(ins, outs),
		SourceChain:    sourceChain,
		ImportedInputs: imported,
	}
}

func exportTx(destinationChain ids.ID, ins []*avax.TransferableInput, outs, exported []*avax.TransferableOutput) *platform.ExportTx {
	return &platform.ExportTx{
		BaseTx:           baseTx(ins, outs),
		DestinationChain: destinationChain,
		ExportedOutputs:  exported,
	}
}

func plainTransfer(ins []*avax.TransferableInput, outs ...*avax.TransferableOutput) *platform.BaseTx {
	tx := baseTx(ins, outs)
	return &tx
}

// createSubnetTx is a transaction type the gateway does not decode.
func createSubnetTx(ins []*avax.TransferableInput) *platform.CreateSubnetTx {
	return &platform.CreateSubnetTx{
		BaseTx: baseTx(ins, nil),
		Owner:  &secp256k1fx.OutputOwners{},
	}
}

func newBanffStandard(t *testing.T, timestamp int64, parent ids.ID, height uint64, blkTxs ...*platform.Tx) *platform.BanffStandardBlock {
	t.Helper()
	blk, err := platform.NewBanffStandardBlock(time.Unix(timestamp, 0), parent, height, blkTxs)
	require.NoError(t, err)
	return blk
}

func TestDecodeBanffStandardBlock(t *testing.T) {
	require := require.New(t)

	nodeA, nodeB, nodeC := ids.GenerateTestNodeID(), ids.GenerateTestNodeID(), ids.GenerateTestNodeID()
	fundingTxID := ids.GenerateTestID()
	validatorTx := signedTx(t, permissionlessValidator(constants.PrimaryNetworkID, nodeA, []*avax.TransferableInput{
		transferInput(fundingTxID, 0, 1000, 0),
		transferInput(fundingTxID, 1, 1, 0),
	}))
	delegatorTx := signedTx(t, permissionlessDelegator(constants.PrimaryNetworkID, nodeA, []*avax.TransferableInput{
		transferInput(fundingTxID, 2, 500, 0),
	}))
	subnetValidatorTx := signedTx(t, permissionlessValidator(ids.GenerateTestID(), nodeB, []*avax.TransferableInput{
		transferInput(fundingTxID, 3, 1000, 0),
	}))
	autoRenewedTx := signedTx(t, autoRenewedValidator(nodeC, []*avax.TransferableInput{
		transferInput(fundingTxID, 4, 1000, 0),
	}))
	autoRenewedTxID := autoRenewedTx.ID()
	configTx := signedTx(t, &platform.SetAutoRenewedValidatorConfigTx{
		BaseTx:                   baseTx([]*avax.TransferableInput{transferInput(fundingTxID, 5, 1, 0)}, nil),
		TxID:                     autoRenewedTxID,
		Auth:                     &secp256k1fx.Input{},
		AutoCompoundRewardShares: 1_000_000,
		Period:                   0,
	})
	unsupportedTx := signedTx(t, createSubnetTx([]*avax.TransferableInput{transferInput(fundingTxID, 6, 1, 0)}))

	parent := ids.GenerateTestID()
	blk := newBanffStandard(t, postDurangoTime, parent, 7, validatorTx, delegatorTx, subnetValidatorTx, autoRenewedTx, configTx, unsupportedTx)

	store := newFakeStore()
	store.accept(blk)
	reader := New(store, nil, testUpgrades)

	got, err := getBlock(reader, 7, nil)
	require.NoError(err)
	require.Equal(uint64(7), got.Height)
	require.Equal(blk.ID(), got.ID)
	require.Equal(parent, got.ParentID)
	require.Equal(uint64(postDurangoTime), got.Timestamp)
	require.Equal(pchain.BlockStandard, got.Kind)
	require.Len(got.Txs, 6)

	require.Equal(pchain.Tx{
		Type:             pchain.TxAddPermissionlessValidator,
		Index:            0,
		ID:               validatorTx.ID(),
		UnsignedBytes:    validatorTx.Unsigned.Bytes(),
		NodeID:           nodeA,
		StartTime:        postDurangoTime, // Durango: chain time, not the tx's start
		EndTime:          222,
		Weight:           1000,
		DelegationShares: 20_000,
		InputCount:       2,
		OutputCount:      1, // the stake output
	}, got.Txs[0])
	require.Equal(pchain.Tx{
		Type:          pchain.TxAddPermissionlessDelegator,
		Index:         1,
		ID:            delegatorTx.ID(),
		UnsignedBytes: delegatorTx.Unsigned.Bytes(),
		NodeID:        nodeA,
		StartTime:     postDurangoTime,
		EndTime:       444,
		Weight:        500,
		InputCount:    1,
		OutputCount:   1,
	}, got.Txs[1])
	// A subnet staker is not a primary-network staker: skipped but counted.
	require.Equal(pchain.Tx{Type: pchain.TxUnsupported, Index: 2, ID: subnetValidatorTx.ID()}, got.Txs[2])
	require.Equal(pchain.Tx{
		Type:                     pchain.TxAddAutoRenewedValidator,
		Index:                    3,
		ID:                       autoRenewedTxID,
		UnsignedBytes:            autoRenewedTx.Unsigned.Bytes(),
		NodeID:                   nodeC,
		StartTime:                postDurangoTime,
		EndTime:                  postDurangoTime + 90*24*3600,
		Period:                   90 * 24 * 3600,
		Weight:                   1000, // sum of the stake outputs
		DelegationShares:         30_000,
		AutoCompoundRewardShares: 250_000,
		InputCount:               1,
		OutputCount:              2, // two stake outputs
	}, got.Txs[3])
	require.Equal(pchain.Tx{
		Type:                     pchain.TxSetAutoRenewedValidatorConfig,
		Index:                    4,
		ID:                       configTx.ID(),
		UnsignedBytes:            configTx.Unsigned.Bytes(),
		StartTime:                postDurangoTime,
		Period:                   0,
		AutoCompoundRewardShares: 1_000_000,
		StakerTxID:               autoRenewedTxID,
		InputCount:               1,
	}, got.Txs[4])
	require.Equal(pchain.Tx{Type: pchain.TxUnsupported, Index: 5, ID: unsupportedTx.ID()}, got.Txs[5])

	// Decoded blocks are cached: a second read makes no store calls.
	calls := store.calls
	again, err := getBlock(reader, 7, nil)
	require.NoError(err)
	require.Equal(got, again)
	require.Equal(calls, store.calls)
}

func TestDecodeProposalAndOptionBlocks(t *testing.T) {
	require := require.New(t)

	nodeID := ids.GenerateTestNodeID()
	fundingTxID := ids.GenerateTestID()
	stakerTxID := ids.GenerateTestID()
	decisionTx := signedTx(t, permissionlessDelegator(constants.PrimaryNetworkID, nodeID, []*avax.TransferableInput{
		transferInput(fundingTxID, 0, 500, 0),
	}))
	rewardTx := signedTx(t, &platform.RewardAutoRenewedValidatorTx{TxID: stakerTxID, Timestamp: postDurangoTime})

	genesisID := ids.GenerateTestID()
	proposal, err := platform.NewBanffProposalBlock(time.Unix(postDurangoTime, 0), genesisID, 1, rewardTx, []*platform.Tx{decisionTx})
	require.NoError(err)
	commit, err := platform.NewBanffCommitBlock(time.Unix(postDurangoTime, 0), proposal.ID(), 2)
	require.NoError(err)
	abort, err := platform.NewBanffAbortBlock(time.Unix(postDurangoTime, 0), proposal.ID(), 3)
	require.NoError(err)

	store := newFakeStore()
	store.accept(proposal)
	store.accept(commit)
	store.accept(abort) // not a valid chain, but exercises the kind mapping
	reader := New(store, nil, testUpgrades)

	got, err := getBlock(reader, 1, nil)
	require.NoError(err)
	require.Equal(pchain.BlockProposal, got.Kind)
	require.Len(got.Txs, 2)
	require.Equal(pchain.TxAddPermissionlessDelegator, got.Txs[0].Type)
	// The proposal transaction is always last.
	require.Equal(pchain.Tx{
		Type:          pchain.TxRewardAutoRenewedValidator,
		Index:         1,
		ID:            rewardTx.ID(),
		UnsignedBytes: rewardTx.Unsigned.Bytes(),
		StartTime:     postDurangoTime,
		EndTime:       postDurangoTime,
		StakerTxID:    stakerTxID,
	}, got.Txs[1])

	got, err = getBlock(reader, 2, nil)
	require.NoError(err)
	require.Equal(pchain.BlockCommit, got.Kind)
	require.Empty(got.Txs)

	got, err = getBlock(reader, 3, nil)
	require.NoError(err)
	require.Equal(pchain.BlockAbort, got.Kind)
	require.Empty(got.Txs)

	// The reward transaction has no inputs.
	inputs, total, err := getInputs(reader, 1, 1, 0, 10, nil)
	require.NoError(err)
	require.Zero(total)
	require.Empty(inputs)
}

func TestDecodeApricotBlocks(t *testing.T) {
	require := require.New(t)

	nodeID := ids.GenerateTestNodeID()
	fundingTxID := ids.GenerateTestID()
	legacyValidator := signedTx(t, &platform.AddValidatorTx{
		BaseTx:           baseTx([]*avax.TransferableInput{transferInput(fundingTxID, 0, 2000, 0)}, nil),
		Validator:        platform.Validator{NodeID: nodeID, Start: 555, End: 666, Wght: 2000},
		StakeOuts:        []*avax.TransferableOutput{transferOutput(2000, owners(addrA))},
		RewardsOwner:     &secp256k1fx.OutputOwners{},
		DelegationShares: 50_000,
	})
	legacyDelegator := signedTx(t, &platform.AddDelegatorTx{
		BaseTx:                 baseTx([]*avax.TransferableInput{transferInput(fundingTxID, 1, 25, 0)}, nil),
		Validator:              platform.Validator{NodeID: nodeID, Start: 777, End: 888, Wght: 25},
		StakeOuts:              []*avax.TransferableOutput{transferOutput(25, owners(addrB))},
		DelegationRewardsOwner: &secp256k1fx.OutputOwners{},
	})
	sourceChain := ids.GenerateTestID()
	atomicTx := signedTx(t, importTx(sourceChain,
		[]*avax.TransferableInput{transferInput(fundingTxID, 2, 1, 0)},
		[]*avax.TransferableInput{transferInput(ids.GenerateTestID(), 0, 5, 0)},
		transferOutput(5, owners(addrA)),
	))

	genesisID := ids.GenerateTestID()
	proposal, err := platform.NewApricotProposalBlock(genesisID, 1, legacyValidator)
	require.NoError(err)
	commit, err := platform.NewApricotCommitBlock(proposal.ID(), 2)
	require.NoError(err)
	standard, err := platform.NewApricotStandardBlock(commit.ID(), 3, []*platform.Tx{legacyDelegator})
	require.NoError(err)
	atomic, err := platform.NewApricotAtomicBlock(standard.ID(), 4, atomicTx)
	require.NoError(err)
	abort, err := platform.NewApricotAbortBlock(atomic.ID(), 5)
	require.NoError(err)

	store := newFakeStore()
	for _, blk := range []platform.Block{proposal, commit, standard, atomic, abort} {
		store.accept(blk)
	}
	reader := New(store, nil, testUpgrades)

	got, err := getBlock(reader, 1, nil)
	require.NoError(err)
	require.Equal(pchain.BlockProposal, got.Kind)
	require.Zero(got.Timestamp) // Apricot blocks carry no timestamp
	require.Equal(pchain.Tx{
		Type:             pchain.TxAddValidator,
		Index:            0,
		ID:               legacyValidator.ID(),
		UnsignedBytes:    legacyValidator.Unsigned.Bytes(),
		NodeID:           nodeID,
		StartTime:        555, // pre-Durango: the transaction's own start
		EndTime:          666,
		Weight:           2000,
		DelegationShares: 50_000,
		InputCount:       1,
		OutputCount:      1,
	}, got.Txs[0])

	got, err = getBlock(reader, 2, nil)
	require.NoError(err)
	require.Equal(pchain.BlockCommit, got.Kind)

	got, err = getBlock(reader, 3, nil)
	require.NoError(err)
	require.Equal(pchain.BlockStandard, got.Kind)
	require.Equal(pchain.Tx{
		Type:          pchain.TxAddDelegator,
		Index:         0,
		ID:            legacyDelegator.ID(),
		UnsignedBytes: legacyDelegator.Unsigned.Bytes(),
		NodeID:        nodeID,
		StartTime:     777,
		EndTime:       888,
		Weight:        25,
		InputCount:    1,
		OutputCount:   1,
	}, got.Txs[0])

	got, err = getBlock(reader, 4, nil)
	require.NoError(err)
	require.Equal(pchain.BlockAtomic, got.Kind)
	// Atomic blocks carry one import or export; Apricot blocks have no
	// timestamp, so the transfer reports none.
	require.Equal(pchain.Tx{
		Type:          pchain.TxImport,
		Index:         0,
		ID:            atomicTx.ID(),
		UnsignedBytes: atomicTx.Unsigned.Bytes(),
		RemoteChainID: sourceChain,
		InputCount:    2, // the P-chain input, then the imported input
		OutputCount:   1,
	}, got.Txs[0])

	got, err = getBlock(reader, 5, nil)
	require.NoError(err)
	require.Equal(pchain.BlockAbort, got.Kind)
}

func TestPreDurangoBanffStartTime(t *testing.T) {
	require := require.New(t)

	nodeID := ids.GenerateTestNodeID()
	validatorTx := signedTx(t, permissionlessValidator(constants.PrimaryNetworkID, nodeID, []*avax.TransferableInput{
		transferInput(ids.GenerateTestID(), 0, 1000, 0),
	}))
	blk := newBanffStandard(t, preDurangoTime, ids.GenerateTestID(), 1, validatorTx)

	store := newFakeStore()
	store.accept(blk)
	reader := New(store, nil, testUpgrades)

	got, err := getBlock(reader, 1, nil)
	require.NoError(err)
	require.Equal(uint64(111), got.Txs[0].StartTime)
}

func TestGetBlockErrors(t *testing.T) {
	require := require.New(t)

	store := newFakeStore()
	acceptChain(t, store, 5)
	reader := New(store, nil, testUpgrades)

	last, err := reader.LastAcceptedHeight()
	require.NoError(err)
	require.Equal(uint64(5), last)

	_, err = getBlock(reader, 6, nil)
	require.ErrorIs(err, pchain.ErrNotAccepted)

	// A height below the tip that is missing from the index is a database
	// problem, not a not-yet-accepted height.
	delete(store.heights, 4)
	_, err = getBlock(reader, 4, nil)
	require.ErrorIs(err, database.ErrNotFound)
	require.NotErrorIs(err, pchain.ErrNotAccepted)
	require.ErrorContains(err, "last accepted 5")

	_, _, err = getInputs(reader, 5, 0, 0, 1, nil)
	require.ErrorIs(err, pchain.ErrTxIndexOutOfRange)
}

// inputFixture builds a staking transaction whose inputs exercise every owner
// resolution rule.
type inputFixture struct {
	store      *fakeStore
	reader     *Reader
	height     uint64
	stakingTx  *platform.Tx
	producerID ids.ID // BaseTx outputs
	stakerID   ids.ID // returned stake + reward UTXO
	missingID  ids.ID // no producing tx in the store
}

func newInputFixture(t *testing.T) *inputFixture {
	t.Helper()
	require := require.New(t)

	// A plain transaction producing two outputs, the second one locked.
	producer := signedTx(t, &platform.BaseTx{BaseTx: avax.BaseTx{
		NetworkID:    constants.UnitTestID,
		BlockchainID: constants.PlatformChainID,
		Ins:          []*avax.TransferableInput{transferInput(ids.GenerateTestID(), 0, 5, 0)},
		Outs: []*avax.TransferableOutput{
			transferOutput(10, owners(addrA)),
			{
				Asset: avax.Asset{ID: avaxAssetID},
				Out: &stakeable.LockOut{
					Locktime: 99,
					TransferableOut: &secp256k1fx.TransferOutput{
						Amt:          20,
						OutputOwners: secp256k1fx.OutputOwners{Locktime: 7, Threshold: 2, Addrs: []ids.ShortID{addrA, addrB, addrC}},
					},
				},
			},
		},
		Memo: types.JSONByteSlice{},
	}})
	// A staker with one BaseTx output and two stake outputs; its reward UTXO
	// is stored separately, keyed by the staker tx.
	staker := signedTx(t, &platform.AddPermissionlessValidatorTx{
		BaseTx:                baseTx([]*avax.TransferableInput{transferInput(ids.GenerateTestID(), 0, 5, 0)}, []*avax.TransferableOutput{transferOutput(1, owners(addrC))}),
		Validator:             platform.Validator{NodeID: ids.GenerateTestNodeID(), Start: 1, End: 2, Wght: 300},
		Subnet:                constants.PrimaryNetworkID,
		Signer:                &signer.Empty{},
		StakeOuts:             []*avax.TransferableOutput{transferOutput(100, owners(addrA)), transferOutput(200, owners(addrB))},
		ValidatorRewardsOwner: &secp256k1fx.OutputOwners{},
		DelegatorRewardsOwner: &secp256k1fx.OutputOwners{},
	})
	stakerID := staker.ID()

	genesisUTXO := &avax.UTXO{
		UTXOID: avax.UTXOID{TxID: ids.Empty, OutputIndex: 3},
		Asset:  avax.Asset{ID: avaxAssetID},
		Out:    &secp256k1fx.TransferOutput{Amt: 42, OutputOwners: owners(addrC)},
	}
	oddGenesisUTXO := &avax.UTXO{
		UTXOID: avax.UTXOID{TxID: genesisMarkerID, OutputIndex: 0},
		Asset:  avax.Asset{ID: avaxAssetID},
		Out:    &secp256k1fx.TransferOutput{Amt: 7, OutputOwners: owners(addrB)},
	}

	missingID := ids.GenerateTestID()
	ins := []*avax.TransferableInput{
		transferInput(ids.Empty, 3, 42, 0),        // genesis
		transferInput(producer.ID(), 0, 10, 0),    // BaseTx output
		transferInput(producer.ID(), 1, 20, 0, 2), // locked BaseTx output, signed by slots 0 and 2
		transferInput(producer.ID(), 5, 1, 0),     // no such output
		transferInput(stakerID, 1, 100, 0),        // returned stake (index len(Outs)+0)
		transferInput(stakerID, 3, 9, 0),          // reward UTXO (index len(Outs)+len(Stake)+0)
		transferInput(genesisMarkerID, 0, 7, 0),   // genesis UTXO with a non-empty tx id
		transferInput(missingID, 0, 1, 0),         // producing tx unknown
	}
	// The last input is wrapped in a stakeable lock.
	ins[4] = &avax.TransferableInput{
		UTXOID: ins[4].UTXOID,
		Asset:  ins[4].Asset,
		In:     &stakeable.LockIn{Locktime: 99, TransferableIn: ins[4].In},
	}
	stakingTx := signedTx(t, permissionlessValidator(constants.PrimaryNetworkID, ids.GenerateTestNodeID(), ins))

	store := newFakeStore()
	store.txs[producer.ID()] = producer
	store.txs[stakerID] = staker
	store.rewardUTXOs[stakerID] = []*avax.UTXO{{
		UTXOID: avax.UTXOID{TxID: stakerID, OutputIndex: 3},
		Asset:  avax.Asset{ID: avaxAssetID},
		Out:    &secp256k1fx.TransferOutput{Amt: 9, OutputOwners: owners(addrB)},
	}}
	blk := newBanffStandard(t, postDurangoTime, ids.GenerateTestID(), 1, stakingTx)
	store.accept(blk)
	require.NotNil(store.txs[stakingTx.ID()])

	return &inputFixture{
		store:      store,
		reader:     New(store, genesisList(genesisUTXO, oddGenesisUTXO), testUpgrades),
		height:     1,
		stakingTx:  stakingTx,
		producerID: producer.ID(),
		stakerID:   stakerID,
		missingID:  missingID,
	}
}

func TestGetInputsResolvesOwners(t *testing.T) {
	require := require.New(t)
	f := newInputFixture(t)

	// The unknown producing transaction is a local error, not an unresolved input.
	_, total, err := getInputs(f.reader, f.height, 0, 0, 10, nil)
	require.ErrorIs(err, database.ErrNotFound)
	require.Equal(uint32(8), total)

	inputs, total, err := getInputs(f.reader, f.height, 0, 0, 7, nil)
	require.NoError(err)
	require.Equal(uint32(8), total)
	require.Equal([]pchain.Input{
		{UTXOTxID: ids.Empty, UTXOOutputIndex: 3, AssetID: avaxAssetID, Amount: 42, Resolved: true, Genesis: true, Threshold: 1, Addresses: []ids.ShortID{addrC}, SigIndices: []uint32{0}},
		{UTXOTxID: f.producerID, UTXOOutputIndex: 0, AssetID: avaxAssetID, Amount: 10, Resolved: true, Threshold: 1, Addresses: []ids.ShortID{addrA}, SigIndices: []uint32{0}},
		{UTXOTxID: f.producerID, UTXOOutputIndex: 1, AssetID: avaxAssetID, Amount: 20, Resolved: true, Locktime: 7, StakeableLocktime: 99, Threshold: 2, Addresses: []ids.ShortID{addrA, addrB, addrC}, SigIndices: []uint32{0, 2}},
		{UTXOTxID: f.producerID, UTXOOutputIndex: 5, AssetID: avaxAssetID, Amount: 1, SigIndices: []uint32{0}},
		{UTXOTxID: f.stakerID, UTXOOutputIndex: 1, AssetID: avaxAssetID, Amount: 100, Resolved: true, Threshold: 1, Addresses: []ids.ShortID{addrA}, SigIndices: []uint32{0}},
		{UTXOTxID: f.stakerID, UTXOOutputIndex: 3, AssetID: avaxAssetID, Amount: 9, Resolved: true, Threshold: 1, Addresses: []ids.ShortID{addrB}, SigIndices: []uint32{0}},
		{UTXOTxID: genesisMarkerID, UTXOOutputIndex: 0, AssetID: avaxAssetID, Amount: 7, Resolved: true, Genesis: true, Threshold: 1, Addresses: []ids.ShortID{addrB}, SigIndices: []uint32{0}},
	}, inputs)

	// Paging.
	inputs, total, err = getInputs(f.reader, f.height, 0, 4, 2, nil)
	require.NoError(err)
	require.Equal(uint32(8), total)
	require.Len(inputs, 2)
	require.Equal(f.stakerID, inputs[0].UTXOTxID)
	require.Equal(uint32(3), inputs[1].UTXOOutputIndex)

	inputs, _, err = getInputs(f.reader, f.height, 0, 7, 0, nil)
	require.NoError(err)
	require.Empty(inputs)

	inputs, _, err = getInputs(f.reader, f.height, 0, 8, 5, nil)
	require.NoError(err)
	require.Empty(inputs)

	_, _, err = getInputs(f.reader, f.height, 0, 9, 1, nil)
	require.ErrorIs(err, pchain.ErrInputIndexOutOfRange)

	// Owners are cached after the first resolution.
	calls := f.store.calls
	_, _, err = getInputs(f.reader, f.height, 0, 0, 7, nil)
	require.NoError(err)
	require.Equal(calls, f.store.calls)
}

func TestGetInputsUnsupportedTx(t *testing.T) {
	require := require.New(t)

	tx := signedTx(t, createSubnetTx([]*avax.TransferableInput{transferInput(ids.GenerateTestID(), 0, 1, 0)}))
	blk := newBanffStandard(t, postDurangoTime, ids.GenerateTestID(), 1, tx)
	store := newFakeStore()
	store.accept(blk)
	reader := New(store, nil, testUpgrades)

	_, _, err := getInputs(reader, 1, 0, 0, 1, nil)
	require.ErrorIs(err, pchain.ErrUnsupportedTx)
	_, _, err = getOutputs(reader, 1, 0, 0, 1)
	require.ErrorIs(err, pchain.ErrUnsupportedTx)
}

func TestLockedStoreHoldsLockPerCall(t *testing.T) {
	require := require.New(t)
	f := newInputFixture(t)

	locker := &countingLocker{}
	reader := New(NewLockedStore(locker, f.store), nil, testUpgrades)

	_, err := getBlock(reader, f.height, nil)
	require.NoError(err)
	_, _, err = getInputs(reader, f.height, 0, 1, 5, nil)
	require.NoError(err)

	require.Positive(locker.locks)
	require.Equal(locker.locks, locker.unlocks)
}

// recordingCharger records every charged byte count and can fail.
type recordingCharger struct {
	charges []uint64
	fail    error
}

func (c *recordingCharger) charge(bytes uint64) error {
	c.charges = append(c.charges, bytes)
	return c.fail
}

func TestGetBlockChargesStoredBytesOnMissAndHit(t *testing.T) {
	require := require.New(t)

	tx := signedTx(t, permissionlessValidator(constants.PrimaryNetworkID, ids.GenerateTestNodeID(), []*avax.TransferableInput{
		transferInput(ids.GenerateTestID(), 0, 1000, 0),
	}))
	blk := newBanffStandard(t, postDurangoTime, ids.GenerateTestID(), 1, tx)
	store := newFakeStore()
	store.accept(blk)
	reader := New(store, nil, testUpgrades)
	size := uint64(len(blk.Bytes()))

	miss := &recordingCharger{}
	_, err := getBlock(reader, 1, miss.charge)
	require.NoError(err)
	require.Equal([]uint64{size}, miss.charges)

	// A cached read charges exactly the same amount without touching the store.
	calls := store.calls
	hit := &recordingCharger{}
	_, err = getBlock(reader, 1, hit.charge)
	require.NoError(err)
	require.Equal([]uint64{size}, hit.charges)
	require.Equal(calls, store.calls)
}

func TestGetBlockChargeFailureAbortsBeforeDecoding(t *testing.T) {
	require := require.New(t)

	blk := newBanffStandard(t, postDurangoTime, ids.GenerateTestID(), 1)
	store := newFakeStore()
	store.accept(blk)
	reader := New(store, nil, testUpgrades)

	errNoGas := errors.New("out of gas")
	failing := &recordingCharger{fail: errNoGas}
	_, err := getBlock(reader, 1, failing.charge)
	require.ErrorIs(err, errNoGas)
	require.Equal([]uint64{uint64(len(blk.Bytes()))}, failing.charges)

	// Nothing was cached: the next read goes back to the store.
	calls := store.calls
	_, err = getBlock(reader, 1, nil)
	require.NoError(err)
	require.Greater(store.calls, calls)
}

func TestGetInputsChargesProducingTxBytes(t *testing.T) {
	require := require.New(t)
	f := newInputFixture(t)

	producer := f.store.txs[f.producerID]
	staker := f.store.txs[f.stakerID]
	blockSize := uint64(len(f.store.blocks[f.store.heights[f.height]].Bytes()))

	// Inputs 0..2: genesis (no producing tx, nothing charged), then two
	// outputs of the same producing tx (charged on each resolution, the
	// second one from cache).
	c := &recordingCharger{}
	inputs, _, err := getInputs(f.reader, f.height, 0, 0, 3, c.charge)
	require.NoError(err)
	require.Len(inputs, 3)
	require.Equal([]uint64{blockSize, uint64(len(producer.Bytes())), uint64(len(producer.Bytes()))}, c.charges)

	// Returned stake and reward UTXO both parse the staker transaction.
	c = &recordingCharger{}
	_, _, err = getInputs(f.reader, f.height, 0, 4, 2, c.charge)
	require.NoError(err)
	require.Equal([]uint64{blockSize, uint64(len(staker.Bytes())), uint64(len(staker.Bytes()))}, c.charges)

	// Reading through the block handle charges the producing transactions
	// only: the block was paid for by GetBlock.
	br, err := f.reader.GetBlock(f.height, nil)
	require.NoError(err)
	c = &recordingCharger{}
	_, _, err = br.GetInputs(0, 0, 3, c.charge)
	require.NoError(err)
	require.Equal([]uint64{uint64(len(producer.Bytes())), uint64(len(producer.Bytes()))}, c.charges)

	// A failing charge on a producing tx aborts the page.
	errNoGas := errors.New("out of gas")
	var charged int
	failSecond := func(uint64) error {
		charged++
		if charged > 1 {
			return errNoGas
		}
		return nil
	}
	_, _, err = getInputs(f.reader, f.height, 0, 0, 3, failSecond)
	require.ErrorIs(err, errNoGas)
	require.Equal(2, charged) // block bytes, then the first producing tx
}

// getBlock returns the decoded block at height.
func getBlock(r *Reader, height uint64, charge pchain.ByteCharger) (*pchain.Block, error) {
	br, err := r.GetBlock(height, charge)
	if err != nil {
		return nil, err
	}
	return br.Block(), nil
}

// getOutputs reads the block at height and then a page of the outputs of its
// transaction txIndex.
func getOutputs(r *Reader, height uint64, txIndex, from, max uint32) ([]pchain.Output, uint32, error) {
	br, err := r.GetBlock(height, nil)
	if err != nil {
		return nil, 0, err
	}
	return br.GetOutputs(txIndex, from, max)
}

// getInputs reads the block at height and then a page of the inputs of its
// transaction txIndex, charging the block and then the producing transactions
// through charge, exactly like a single gateway call does.
func getInputs(r *Reader, height uint64, txIndex, from, max uint32, charge pchain.ByteCharger) ([]pchain.Input, uint32, error) {
	br, err := r.GetBlock(height, charge)
	if err != nil {
		return nil, 0, err
	}
	return br.GetInputs(txIndex, from, max, charge)
}

// acceptChain accepts n empty Banff standard blocks at heights 1..n.
func acceptChain(t *testing.T, store *fakeStore, n uint64) []*platform.BanffStandardBlock {
	t.Helper()
	blocks := make([]*platform.BanffStandardBlock, 0, n)
	parent := ids.GenerateTestID()
	for h := uint64(1); h <= n; h++ {
		blk := newBanffStandard(t, postDurangoTime, parent, h)
		store.accept(blk)
		blocks = append(blocks, blk)
		parent = blk.ID()
	}
	return blocks
}

func TestLastAcceptedHeightUsesHeightIndexOnly(t *testing.T) {
	require := require.New(t)
	store := newFakeStore()
	acceptChain(t, store, 300)
	reader := New(store, nil, testUpgrades)

	// A cold lookup gallops and bisects over the height index: no block is
	// fetched, let alone parsed, and the number of index probes is
	// logarithmic (1,2,4,…,512 then a bisection of at most 9 steps, plus
	// the tip identity check).
	last, err := reader.LastAcceptedHeight()
	require.NoError(err)
	require.Equal(uint64(300), last)
	require.Zero(store.bytesCalls)
	require.LessOrEqual(store.heightCalls, 10+9+1)

	// An unchanged tip is answered from the cached pair with a single call.
	calls, heightCalls := store.calls, store.heightCalls
	last, err = reader.LastAcceptedHeight()
	require.NoError(err)
	require.Equal(uint64(300), last)
	require.Equal(calls+1, store.calls)
	require.Equal(heightCalls, store.heightCalls)

	// A tip that advanced by one costs a handful of probes from the known
	// height, still without touching any block.
	acceptChain(t, store, 301)
	heightCalls = store.heightCalls
	last, err = reader.LastAcceptedHeight()
	require.NoError(err)
	require.Equal(uint64(301), last)
	require.LessOrEqual(store.heightCalls-heightCalls, 4)
	require.Zero(store.bytesCalls)

	// Reads above the tip are rejected through the index as well, and the
	// error names the tip.
	_, err = getBlock(reader, 302, nil)
	require.ErrorIs(err, pchain.ErrNotAccepted)
	require.ErrorContains(err, "last accepted 301")
	require.Zero(store.bytesCalls)
}

func TestLastAcceptedHeightWhenTipMovesDuringLookup(t *testing.T) {
	require := require.New(t)
	store := newFakeStore()
	blocks := acceptChain(t, store, 3)
	reader := New(store, nil, testUpgrades)

	// The tip ID read first is stale (block 2) while the index already holds
	// block 3: the index is authoritative, and the stale pair is not cached.
	store.staleLastAccepted = blocks[1].ID()
	store.staleReads = 1
	last, err := reader.LastAcceptedHeight()
	require.NoError(err)
	require.Equal(uint64(3), last)
	require.Zero(store.staleReads)

	// The next call sees the real tip ID, searches the index again and
	// caches the consistent pair.
	heightCalls := store.heightCalls
	last, err = reader.LastAcceptedHeight()
	require.NoError(err)
	require.Equal(uint64(3), last)
	require.Greater(store.heightCalls, heightCalls)

	heightCalls = store.heightCalls
	last, err = reader.LastAcceptedHeight()
	require.NoError(err)
	require.Equal(uint64(3), last)
	require.Equal(heightCalls, store.heightCalls)
	require.Zero(store.bytesCalls)
}

// TestOwnerlessOutputResolves covers an output with threshold 0 and no
// addresses, which secp256k1fx permits: it is located, so it resolves, with an
// empty owner list.
func TestOwnerlessOutputResolves(t *testing.T) {
	require := require.New(t)

	ownerless := baseTx(
		[]*avax.TransferableInput{transferInput(ids.GenerateTestID(), 0, 5, 0)},
		[]*avax.TransferableOutput{transferOutput(10, secp256k1fx.OutputOwners{Threshold: 0})},
	)
	producer := signedTx(t, &ownerless)
	stakingTx := signedTx(t, permissionlessValidator(constants.PrimaryNetworkID, ids.GenerateTestNodeID(), []*avax.TransferableInput{
		transferInput(producer.ID(), 0, 10), // spent without signatures
	}))
	store := newFakeStore()
	store.txs[producer.ID()] = producer
	store.accept(newBanffStandard(t, postDurangoTime, ids.GenerateTestID(), 1, stakingTx))
	reader := New(store, nil, testUpgrades)

	inputs, total, err := getInputs(reader, 1, 0, 0, 1, nil)
	require.NoError(err)
	require.Equal(uint32(1), total)
	require.Len(inputs, 1)
	in := inputs[0]
	require.True(in.Resolved)
	require.False(in.Genesis)
	require.Zero(in.Threshold)
	require.Empty(in.Addresses)
	require.Empty(in.SigIndices)
	require.Equal(uint64(10), in.Amount)
}

func utxoTxIDs(inputs []pchain.Input) []ids.ID {
	out := make([]ids.ID, len(inputs))
	for i, in := range inputs {
		out[i] = in.UTXOTxID
	}
	return out
}

// TestDecodeTransferTransactions covers the value-moving types (import,
// export, plain transfer) and their outputs, which are reported at the
// positions the P-chain assigns the produced UTXOs.
func TestDecodeTransferTransactions(t *testing.T) {
	require := require.New(t)

	fundingTxID := ids.GenerateTestID()
	sourceChain, destinationChain := ids.GenerateTestID(), ids.GenerateTestID()
	importedTxID := ids.GenerateTestID()
	importedTx := signedTx(t, importTx(sourceChain,
		[]*avax.TransferableInput{transferInput(fundingTxID, 0, 1, 0)},
		[]*avax.TransferableInput{transferInput(importedTxID, 0, 500, 0), transferInput(importedTxID, 1, 250, 1)},
		transferOutput(750, owners(addrA)),
	))
	exportedTx := signedTx(t, exportTx(destinationChain,
		[]*avax.TransferableInput{transferInput(fundingTxID, 1, 1000, 0)},
		[]*avax.TransferableOutput{transferOutput(100, owners(addrA))},
		[]*avax.TransferableOutput{transferOutput(600, owners(addrB)), transferOutput(299, owners(addrC))},
	))
	locked := &avax.TransferableOutput{
		Asset: avax.Asset{ID: avaxAssetID},
		Out: &stakeable.LockOut{
			Locktime: 99,
			TransferableOut: &secp256k1fx.TransferOutput{
				Amt:          20,
				OutputOwners: secp256k1fx.OutputOwners{Locktime: 7, Threshold: 2, Addrs: []ids.ShortID{addrA, addrB}},
			},
		},
	}
	transferTx := signedTx(t, plainTransfer(
		[]*avax.TransferableInput{transferInput(fundingTxID, 2, 30, 0)},
		transferOutput(9, owners(addrC)), locked,
	))

	blk := newBanffStandard(t, postDurangoTime, ids.GenerateTestID(), 1, importedTx, exportedTx, transferTx)
	store := newFakeStore()
	store.accept(blk)
	reader := New(store, nil, testUpgrades)

	got, err := getBlock(reader, 1, nil)
	require.NoError(err)
	require.Equal([]pchain.Tx{
		{Type: pchain.TxImport, Index: 0, ID: importedTx.ID(), UnsignedBytes: importedTx.Unsigned.Bytes(), StartTime: postDurangoTime, RemoteChainID: sourceChain, InputCount: 3, OutputCount: 1},
		{Type: pchain.TxExport, Index: 1, ID: exportedTx.ID(), UnsignedBytes: exportedTx.Unsigned.Bytes(), StartTime: postDurangoTime, RemoteChainID: destinationChain, InputCount: 1, OutputCount: 3},
		{Type: pchain.TxBase, Index: 2, ID: transferTx.ID(), UnsignedBytes: transferTx.Unsigned.Bytes(), StartTime: postDurangoTime, InputCount: 1, OutputCount: 2},
	}, got.Txs)

	outputs, total, err := getOutputs(reader, 1, 0, 0, 10)
	require.NoError(err)
	require.Equal(uint32(1), total)
	require.Equal([]pchain.Output{{Kind: pchain.OutputTransfer, AssetID: avaxAssetID, Amount: 750, Threshold: 1, Addresses: []ids.ShortID{addrA}}}, outputs)

	// The BaseTx output comes first, then the exported outputs: the
	// destination chain creates its UTXOs at indexes len(Outs)+i.
	outputs, total, err = getOutputs(reader, 1, 1, 0, 10)
	require.NoError(err)
	require.Equal(uint32(3), total)
	require.Equal([]pchain.Output{
		{Kind: pchain.OutputTransfer, AssetID: avaxAssetID, Amount: 100, Threshold: 1, Addresses: []ids.ShortID{addrA}},
		{Kind: pchain.OutputExported, AssetID: avaxAssetID, Amount: 600, Threshold: 1, Addresses: []ids.ShortID{addrB}},
		{Kind: pchain.OutputExported, AssetID: avaxAssetID, Amount: 299, Threshold: 1, Addresses: []ids.ShortID{addrC}},
	}, outputs)

	// A stakeable lock is unwrapped into StakeableLocktime; the owner's own
	// locktime is reported separately.
	outputs, total, err = getOutputs(reader, 1, 2, 0, 10)
	require.NoError(err)
	require.Equal(uint32(2), total)
	require.Equal([]pchain.Output{
		{Kind: pchain.OutputTransfer, AssetID: avaxAssetID, Amount: 9, Threshold: 1, Addresses: []ids.ShortID{addrC}},
		{Kind: pchain.OutputTransfer, AssetID: avaxAssetID, Amount: 20, Locktime: 7, StakeableLocktime: 99, Threshold: 2, Addresses: []ids.ShortID{addrA, addrB}},
	}, outputs)
}

func TestDecodeRewardValidatorProposal(t *testing.T) {
	require := require.New(t)

	stakerTxID := ids.GenerateTestID()
	decisionTx := signedTx(t, permissionlessDelegator(constants.PrimaryNetworkID, ids.GenerateTestNodeID(), []*avax.TransferableInput{
		transferInput(ids.GenerateTestID(), 0, 500, 0),
	}))
	rewardTx := signedTx(t, &platform.RewardValidatorTx{TxID: stakerTxID})

	proposal, err := platform.NewBanffProposalBlock(time.Unix(postDurangoTime, 0), ids.GenerateTestID(), 1, rewardTx, []*platform.Tx{decisionTx})
	require.NoError(err)
	commit, err := platform.NewBanffCommitBlock(time.Unix(postDurangoTime, 0), proposal.ID(), 2)
	require.NoError(err)

	store := newFakeStore()
	store.accept(proposal)
	store.accept(commit)
	reader := New(store, nil, testUpgrades)

	got, err := getBlock(reader, 1, nil)
	require.NoError(err)
	require.Equal(pchain.BlockProposal, got.Kind)
	require.Len(got.Txs, 2)
	// The removal takes effect at the block time, which is the removed
	// staker's end time; the proposal transaction is last.
	require.Equal(pchain.Tx{
		Type:          pchain.TxRewardValidator,
		Index:         1,
		ID:            rewardTx.ID(),
		UnsignedBytes: rewardTx.Unsigned.Bytes(),
		StartTime:     postDurangoTime,
		EndTime:       postDurangoTime,
		StakerTxID:    stakerTxID,
	}, got.Txs[1])

	// A reward transaction has neither inputs nor outputs.
	inputs, total, err := getInputs(reader, 1, 1, 0, 10, nil)
	require.NoError(err)
	require.Zero(total)
	require.Empty(inputs)
	outputs, total, err := getOutputs(reader, 1, 1, 0, 10)
	require.NoError(err)
	require.Zero(total)
	require.Empty(outputs)

	// Apricot proposal blocks carry the reward transaction alone and no
	// timestamp.
	apricot, err := platform.NewApricotProposalBlock(commit.ID(), 3, rewardTx)
	require.NoError(err)
	store.accept(apricot)
	got, err = getBlock(reader, 3, nil)
	require.NoError(err)
	require.Equal(pchain.Tx{Type: pchain.TxRewardValidator, Index: 0, ID: rewardTx.ID(), StakerTxID: stakerTxID, UnsignedBytes: rewardTx.Unsigned.Bytes()}, got.Txs[0])
}

// TestGetInputsImportTx checks that the P-chain inputs of an import are
// resolved while its imported inputs, which consume UTXOs of the source
// chain, are reported without any lookup: the store does not know their
// producing transactions, and a genesis UTXO with the same key must not be
// mistaken for them.
func TestGetInputsImportTx(t *testing.T) {
	require := require.New(t)

	// producerID is the P-chain transaction that produced the fee input,
	// exportID the source chain's export transaction of the second imported
	// UTXO.
	producerID, exportID := ids.ID{0x03}, ids.ID{0x01}

	producer := signedTx(t, plainTransfer(
		[]*avax.TransferableInput{transferInput(ids.GenerateTestID(), 0, 5, 0)},
		transferOutput(3, owners(addrA)),
	))
	producer.TxID = producerID
	genesisUTXO := &avax.UTXO{
		UTXOID: avax.UTXOID{TxID: ids.Empty, OutputIndex: 3},
		Asset:  avax.Asset{ID: avaxAssetID},
		Out:    &secp256k1fx.TransferOutput{Amt: 42, OutputOwners: owners(addrC)},
	}
	tx := signedTx(t, importTx(ids.GenerateTestID(),
		[]*avax.TransferableInput{transferInput(producerID, 0, 3, 0)},
		[]*avax.TransferableInput{transferInput(ids.Empty, 3, 42, 0), transferInput(exportID, 0, 7, 0, 1)},
		transferOutput(50, owners(addrA)),
	))
	blk := newBanffStandard(t, postDurangoTime, ids.GenerateTestID(), 1, tx)
	store := newFakeStore()
	store.txs[producerID] = producer
	store.accept(blk)
	reader := New(store, genesisList(genesisUTXO), testUpgrades)

	// Canonical order: the P-chain input, then the imported inputs. Only the
	// block and the one producing transaction are fetched and charged.
	c := &recordingCharger{}
	inputs, total, err := getInputs(reader, 1, 0, 0, 10, c.charge)
	require.NoError(err)
	require.Equal(uint32(3), total)
	require.Equal([]pchain.Input{
		{UTXOTxID: producerID, UTXOOutputIndex: 0, AssetID: avaxAssetID, Amount: 3, Resolved: true, Threshold: 1, Addresses: []ids.ShortID{addrA}, SigIndices: []uint32{0}},
		{UTXOTxID: ids.Empty, UTXOOutputIndex: 3, AssetID: avaxAssetID, Amount: 42, Imported: true, SigIndices: []uint32{0}},
		{UTXOTxID: exportID, UTXOOutputIndex: 0, AssetID: avaxAssetID, Amount: 7, Imported: true, SigIndices: []uint32{0, 1}},
	}, inputs)
	require.Equal([]uint64{uint64(len(blk.Bytes())), uint64(len(producer.Bytes()))}, c.charges)
	require.Equal(2, store.bytesCalls)

	// Paging: position 1 is the first imported input, which resolves and
	// fetches nothing.
	page, _, err := getInputs(reader, 1, 0, 1, 1, nil)
	require.NoError(err)
	require.Equal([]ids.ID{ids.Empty}, utxoTxIDs(page))
	require.True(page[0].Imported)
	require.Equal(2, store.bytesCalls)

	// An import funded from imported UTXOs alone has nothing to resolve.
	pure := signedTx(t, importTx(ids.GenerateTestID(), nil,
		[]*avax.TransferableInput{transferInput(exportID, 1, 9, 0)},
		transferOutput(9, owners(addrB)),
	))
	store.accept(newBanffStandard(t, postDurangoTime, blk.ID(), 2, pure))
	inputs, total, err = getInputs(reader, 2, 0, 0, 10, nil)
	require.NoError(err)
	require.Equal(uint32(1), total)
	require.Equal([]pchain.Input{{UTXOTxID: exportID, UTXOOutputIndex: 1, AssetID: avaxAssetID, Amount: 9, Imported: true, SigIndices: []uint32{0}}}, inputs)
	require.Equal(3, store.bytesCalls) // the block only
}

func TestGetOutputs(t *testing.T) {
	require := require.New(t)

	staker := signedTx(t, &platform.AddPermissionlessValidatorTx{
		BaseTx:                baseTx([]*avax.TransferableInput{transferInput(ids.GenerateTestID(), 0, 5, 0)}, []*avax.TransferableOutput{transferOutput(1, owners(addrC))}),
		Validator:             platform.Validator{NodeID: ids.GenerateTestNodeID(), Start: 1, End: 2, Wght: 300},
		Subnet:                constants.PrimaryNetworkID,
		Signer:                &signer.Empty{},
		StakeOuts:             []*avax.TransferableOutput{transferOutput(100, owners(addrA)), transferOutput(200, owners(addrB))},
		ValidatorRewardsOwner: &secp256k1fx.OutputOwners{},
		DelegatorRewardsOwner: &secp256k1fx.OutputOwners{},
	})
	ownerless := signedTx(t, plainTransfer(
		[]*avax.TransferableInput{transferInput(ids.GenerateTestID(), 0, 5, 0)},
		transferOutput(10, secp256k1fx.OutputOwners{Threshold: 0}),
	))
	unsupported := signedTx(t, createSubnetTx([]*avax.TransferableInput{transferInput(ids.GenerateTestID(), 0, 1, 0)}))
	blk := newBanffStandard(t, postDurangoTime, ids.GenerateTestID(), 1, staker, ownerless, unsupported)
	store := newFakeStore()
	store.accept(blk)
	reader := New(store, nil, testUpgrades)

	got, err := getBlock(reader, 1, nil)
	require.NoError(err)
	require.Equal(uint32(3), got.Txs[0].OutputCount)

	// The BaseTx output comes first, then the stake outputs, at the UTXO
	// indexes the P-chain assigns them (and returns the stake at).
	want := []pchain.Output{
		{Kind: pchain.OutputTransfer, AssetID: avaxAssetID, Amount: 1, Threshold: 1, Addresses: []ids.ShortID{addrC}},
		{Kind: pchain.OutputStake, AssetID: avaxAssetID, Amount: 100, Threshold: 1, Addresses: []ids.ShortID{addrA}},
		{Kind: pchain.OutputStake, AssetID: avaxAssetID, Amount: 200, Threshold: 1, Addresses: []ids.ShortID{addrB}},
	}
	outputs, total, err := getOutputs(reader, 1, 0, 0, 10)
	require.NoError(err)
	require.Equal(uint32(3), total)
	require.Equal(want, outputs)

	// Paging.
	outputs, total, err = getOutputs(reader, 1, 0, 1, 1)
	require.NoError(err)
	require.Equal(uint32(3), total)
	require.Equal(want[1:2], outputs)
	outputs, _, err = getOutputs(reader, 1, 0, 3, 5)
	require.NoError(err)
	require.Empty(outputs)
	require.NotNil(outputs)
	outputs, _, err = getOutputs(reader, 1, 0, 0, 0)
	require.NoError(err)
	require.Empty(outputs)
	_, total, err = getOutputs(reader, 1, 0, 4, 1)
	require.ErrorIs(err, pchain.ErrOutputIndexOutOfRange)
	require.Equal(uint32(3), total)

	// An ownerless output (threshold 0) has an amount but no addresses.
	outputs, _, err = getOutputs(reader, 1, 1, 0, 10)
	require.NoError(err)
	require.Len(outputs, 1)
	require.Equal(pchain.OutputTransfer, outputs[0].Kind)
	require.Equal(uint64(10), outputs[0].Amount)
	require.Zero(outputs[0].Threshold)
	require.Empty(outputs[0].Addresses)

	_, _, err = getOutputs(reader, 1, 2, 0, 1)
	require.ErrorIs(err, pchain.ErrUnsupportedTx)
	_, _, err = getOutputs(reader, 1, 3, 0, 1)
	require.ErrorIs(err, pchain.ErrTxIndexOutOfRange)

	// Outputs come from the decoded block: nothing else is fetched.
	calls := store.calls
	_, _, err = getOutputs(reader, 1, 0, 0, 10)
	require.NoError(err)
	require.Equal(calls, store.calls)
}

// TestAssetIDsFollowInputsAndOutputs checks that every input and output
// reports the asset of its own bytes, so that a custom X-chain asset that is
// imported to the P-chain, transferred, exported and used on the fee side of
// a staking transaction stays distinguishable from native funds, while stakes
// are always native.
func TestAssetIDsFollowInputsAndOutputs(t *testing.T) {
	require := require.New(t)

	customAssetID := ids.GenerateTestID()
	customOutput := func(amount uint64, o secp256k1fx.OutputOwners) *avax.TransferableOutput {
		out := transferOutput(amount, o)
		out.Asset = avax.Asset{ID: customAssetID}
		return out
	}
	customInput := func(utxoTxID ids.ID, index uint32, amount uint64) *avax.TransferableInput {
		in := transferInput(utxoTxID, index, amount, 0)
		in.Asset = avax.Asset{ID: customAssetID}
		return in
	}
	type assetAmount struct {
		asset  ids.ID
		amount uint64
	}
	inputAssets := func(inputs []pchain.Input) []assetAmount {
		got := make([]assetAmount, len(inputs))
		for i, in := range inputs {
			got[i] = assetAmount{in.AssetID, in.Amount}
		}
		return got
	}
	outputAssets := func(outputs []pchain.Output) []assetAmount {
		got := make([]assetAmount, len(outputs))
		for i, out := range outputs {
			got[i] = assetAmount{out.AssetID, out.Amount}
		}
		return got
	}

	// Native funds for fees and the stake.
	funding := signedTx(t, plainTransfer(
		[]*avax.TransferableInput{transferInput(ids.GenerateTestID(), 0, 2000, 0)},
		transferOutput(1, owners(addrA)), transferOutput(30, owners(addrA)), transferOutput(10, owners(addrA)), transferOutput(1001, owners(addrA)),
	))
	remoteChain := ids.GenerateTestID()

	// The import's native P-chain input pays the fee; the imported input
	// brings in the custom asset, which the output keeps.
	imported := signedTx(t, importTx(remoteChain,
		[]*avax.TransferableInput{transferInput(funding.ID(), 0, 1, 0)},
		[]*avax.TransferableInput{customInput(ids.GenerateTestID(), 0, 500)},
		customOutput(500, owners(addrA)),
	))
	// A plain transfer moving both assets.
	transferred := signedTx(t, plainTransfer(
		[]*avax.TransferableInput{customInput(imported.ID(), 0, 500), transferInput(funding.ID(), 1, 30, 0)},
		customOutput(300, owners(addrB)), transferOutput(29, owners(addrA)), customOutput(200, owners(addrC)),
	))
	// An export of the custom asset with native change.
	exported := signedTx(t, exportTx(remoteChain,
		[]*avax.TransferableInput{customInput(transferred.ID(), 0, 300), transferInput(funding.ID(), 2, 10, 0)},
		[]*avax.TransferableOutput{transferOutput(9, owners(addrA))},
		[]*avax.TransferableOutput{customOutput(300, owners(addrB))},
	))
	// A staker passing the custom asset through on its fee side (the flow
	// checker only requires each asset to balance); the stake is native.
	staker := permissionlessValidator(constants.PrimaryNetworkID, ids.GenerateTestNodeID(), []*avax.TransferableInput{
		transferInput(funding.ID(), 3, 1001, 0), customInput(transferred.ID(), 2, 200),
	})
	staker.Outs = []*avax.TransferableOutput{customOutput(200, owners(addrA))}
	staked := signedTx(t, staker)

	blk := newBanffStandard(t, postDurangoTime, ids.GenerateTestID(), 1, imported, transferred, exported, staked)
	store := newFakeStore()
	store.txs[funding.ID()] = funding
	store.accept(blk)
	reader := New(store, nil, testUpgrades)

	got, err := getBlock(reader, 1, nil)
	require.NoError(err)
	require.Equal(uint64(1000), got.Txs[3].Weight)

	// Import: the native fee input, then the custom imported input.
	inputs, _, err := getInputs(reader, 1, 0, 0, 10, nil)
	require.NoError(err)
	require.Equal([]assetAmount{{avaxAssetID, 1}, {customAssetID, 500}}, inputAssets(inputs))
	require.False(inputs[0].Imported)
	require.True(inputs[1].Imported)
	outputs, _, err := getOutputs(reader, 1, 0, 0, 10)
	require.NoError(err)
	require.Equal([]assetAmount{{customAssetID, 500}}, outputAssets(outputs))

	// Transfer: assets follow each input and output, in serialized order.
	inputs, _, err = getInputs(reader, 1, 1, 0, 10, nil)
	require.NoError(err)
	require.Equal([]assetAmount{{customAssetID, 500}, {avaxAssetID, 30}}, inputAssets(inputs))
	require.True(inputs[0].Resolved) // resolved from the import's custom output
	outputs, _, err = getOutputs(reader, 1, 1, 0, 10)
	require.NoError(err)
	require.Equal([]assetAmount{{customAssetID, 300}, {avaxAssetID, 29}, {customAssetID, 200}}, outputAssets(outputs))

	// Export: native change, then the exported custom output.
	inputs, _, err = getInputs(reader, 1, 2, 0, 10, nil)
	require.NoError(err)
	require.Equal([]assetAmount{{customAssetID, 300}, {avaxAssetID, 10}}, inputAssets(inputs))
	outputs, _, err = getOutputs(reader, 1, 2, 0, 10)
	require.NoError(err)
	require.Equal([]assetAmount{{avaxAssetID, 9}, {customAssetID, 300}}, outputAssets(outputs))
	require.Equal(pchain.OutputTransfer, outputs[0].Kind)
	require.Equal(pchain.OutputExported, outputs[1].Kind)

	// Staker: the custom change output is a transfer output, the stake is
	// native.
	inputs, _, err = getInputs(reader, 1, 3, 0, 10, nil)
	require.NoError(err)
	require.Equal([]assetAmount{{avaxAssetID, 1001}, {customAssetID, 200}}, inputAssets(inputs))
	outputs, _, err = getOutputs(reader, 1, 3, 0, 10)
	require.NoError(err)
	require.Equal([]assetAmount{{customAssetID, 200}, {avaxAssetID, 1000}}, outputAssets(outputs))
	require.Equal(pchain.OutputTransfer, outputs[0].Kind)
	require.Equal(pchain.OutputStake, outputs[1].Kind)
}

// TestDecodeMalformedTxIsUnsupported covers the decoder's error path: a
// transaction it cannot decode (here an auto-renewed validator with a
// malformed node ID, which accepted blocks never contain) is exposed as
// unsupported, with none of its inputs or outputs.
func TestDecodeMalformedTxIsUnsupported(t *testing.T) {
	require := require.New(t)

	malformed := autoRenewedValidator(ids.GenerateTestNodeID(), []*avax.TransferableInput{transferInput(ids.GenerateTestID(), 0, 1000, 0)})
	malformed.ValidatorNodeID = []byte{1, 2, 3}
	tx := signedTx(t, malformed)
	blk := newBanffStandard(t, postDurangoTime, ids.GenerateTestID(), 1, tx)
	store := newFakeStore()
	store.accept(blk)
	reader := New(store, nil, testUpgrades)

	got, err := getBlock(reader, 1, nil)
	require.NoError(err)
	require.Equal(pchain.Tx{Type: pchain.TxUnsupported, Index: 0, ID: tx.ID()}, got.Txs[0])
	_, _, err = getInputs(reader, 1, 0, 0, 1, nil)
	require.ErrorIs(err, pchain.ErrUnsupportedTx)
	_, _, err = getOutputs(reader, 1, 0, 0, 1)
	require.ErrorIs(err, pchain.ErrUnsupportedTx)
}

// TestGetInputsSignatures checks that inputs carry the signatures of their
// credential, aligned with the signature indices, that decoded transactions
// expose their unsigned bytes, and that the two recover the signers, as the
// gateway's getStakeOwner needs.
func TestGetInputsSignatures(t *testing.T) {
	require := require.New(t)
	keys := secp256k1.TestKeys()
	k0, k1, authority := keys[0], keys[1], keys[2]

	// The producing transaction: two outputs, owned by k0 and k1.
	producer := signedTx(t, plainTransfer(
		[]*avax.TransferableInput{transferInput(ids.GenerateTestID(), 0, 10, 0)},
		transferOutput(5, owners(k0.Address())),
		transferOutput(5, owners(k1.Address())),
	))
	ins := func() []*avax.TransferableInput {
		return []*avax.TransferableInput{
			transferInput(producer.ID(), 0, 5, 0),
			transferInput(producer.ID(), 1, 5, 0),
		}
	}
	signedBy := func(unsigned platform.UnsignedTx, signers ...[]*secp256k1.PrivateKey) *platform.Tx {
		tx, err := platform.NewSignedTx(unsigned, platform.Codec, signers)
		require.NoError(err)
		return tx
	}
	// A transfer whose two inputs are signed by k0 and k1, one credential each.
	transfer := signedBy(plainTransfer(ins(), transferOutput(10, owners(k0.Address()))), []*secp256k1.PrivateKey{k0}, []*secp256k1.PrivateKey{k1})
	// A config transaction: the credential of its input, then the validator
	// authority's, which is not an input credential.
	config := signedBy(&platform.SetAutoRenewedValidatorConfigTx{
		BaseTx: baseTx(ins()[:1], nil),
		TxID:   ids.GenerateTestID(),
		Auth:   &secp256k1fx.Input{SigIndices: []uint32{0}},
		Period: 1,
	}, []*secp256k1.PrivateKey{k0}, []*secp256k1.PrivateKey{authority})
	// A transfer with fewer credentials than inputs, and one with none.
	short := signedBy(plainTransfer(ins(), transferOutput(10, owners(k1.Address()))), []*secp256k1.PrivateKey{k0})
	unsigned := signedTx(t, plainTransfer(ins(), transferOutput(10, owners(k0.Address()))))

	store := newFakeStore()
	store.txs[producer.ID()] = producer
	store.accept(newBanffStandard(t, postDurangoTime, ids.GenerateTestID(), 1, transfer, config, short, unsigned))
	reader := New(store, nil, testUpgrades)

	blk, err := getBlock(reader, 1, nil)
	require.NoError(err)
	for i, tx := range []*platform.Tx{transfer, config, short, unsigned} {
		require.Equal(tx.Unsigned.Bytes(), blk.Txs[i].UnsignedBytes)
	}
	sigsOf := func(tx *platform.Tx, credIndex int) [][pchain.SignatureLen]byte {
		cred, ok := tx.Creds[credIndex].(*secp256k1fx.Credential)
		require.True(ok)
		return cred.Sigs
	}

	// The transfer: each input carries its own credential's signature, which
	// recovers to its owner over the transaction's unsigned bytes.
	inputs, _, err := getInputs(reader, 1, 0, 0, 2, nil)
	require.NoError(err)
	require.Len(inputs, 2)
	require.Equal(sigsOf(transfer, 0), inputs[0].Signatures)
	require.Equal(sigsOf(transfer, 1), inputs[1].Signatures)
	hashes := secp256k1fx.NewSigningHashes(blk.Txs[0].UnsignedBytes)
	for i, key := range []*secp256k1.PrivateKey{k0, k1} {
		require.Equal([]ids.ShortID{key.Address()}, inputs[i].Addresses)
		pk, err := hashes.RecoverSigner(nil, inputs[i].Signatures[0][:], key.Address())
		require.NoError(err)
		require.Equal(key.EthAddress(), pk.EthAddress())
	}

	// The config transaction: its input credential only; the trailing
	// authority credential is not an input credential.
	inputs, _, err = getInputs(reader, 1, 1, 0, 1, nil)
	require.NoError(err)
	require.Len(inputs, 1)
	require.Equal(sigsOf(config, 0), inputs[0].Signatures)
	pk, err := secp256k1fx.NewSigningHashes(blk.Txs[1].UnsignedBytes).RecoverSigner(nil, inputs[0].Signatures[0][:], k0.Address())
	require.NoError(err)
	require.Equal(k0.Address(), pk.Address())

	// Fewer credentials than inputs: the inputs beyond them have none.
	inputs, _, err = getInputs(reader, 1, 2, 0, 2, nil)
	require.NoError(err)
	require.Len(inputs, 2)
	require.Equal(sigsOf(short, 0), inputs[0].Signatures)
	require.Nil(inputs[1].Signatures)

	// No credentials at all.
	inputs, _, err = getInputs(reader, 1, 3, 0, 2, nil)
	require.NoError(err)
	require.Len(inputs, 2)
	for _, in := range inputs {
		require.Nil(in.Signatures)
	}
}
