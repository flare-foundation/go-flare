// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package pchaingateway

import (
	"bytes"
	"errors"
	"fmt"
	"math/big"
	"testing"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/core/vm"
	"github.com/ava-labs/libevm/crypto"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ava-labs/avalanchego/graft/coreth/precompile/contract"
	"github.com/ava-labs/avalanchego/graft/coreth/precompile/precompiletest"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/snow/pchain"
	"github.com/ava-labs/avalanchego/snow/snowtest"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/crypto/secp256k1"
	"github.com/ava-labs/avalanchego/utils/formatting/address"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
)

// fakeReader is an in-memory pchain.Reader that charges the configured
// stored sizes exactly like the real reader: the block's bytes when it is
// read, then the producing transaction's bytes for each input returned.
type fakeReader struct {
	last   uint64
	blocks map[uint64]*pchain.Block
	// inputs by height and transaction index, in canonical order
	inputs map[uint64]map[uint32][]pchain.Input
	// outputs by height and transaction index
	outputs map[uint64]map[uint32][]pchain.Output
	// blockBytes by height; inputBytes by height, transaction index and
	// canonical input position (0 for genesis UTXOs)
	blockBytes map[uint64]uint64
	inputBytes map[uint64]map[uint32][]uint64
	// err, if set, is returned by every method (a node-local failure)
	err error
}

func (f *fakeReader) LastAcceptedHeight() (uint64, error) {
	return f.last, f.err
}

func (f *fakeReader) GetBlock(height uint64, charge pchain.ByteCharger) (pchain.BlockReader, error) {
	if f.err != nil {
		return nil, f.err
	}
	if height > f.last {
		return nil, pchain.ErrNotAccepted
	}
	blk, ok := f.blocks[height]
	if !ok {
		return nil, fmt.Errorf("block %d missing from fake store", height)
	}
	if charge != nil {
		if err := charge(f.blockBytes[height]); err != nil {
			return nil, err
		}
	}
	return &fakeBlockReader{blk: blk, inputs: f.inputs[height], outputs: f.outputs[height], sizes: f.inputBytes[height]}, nil
}

// fakeBlockReader serves the inputs and outputs of one fake block, charging
// each input's producing-transaction size like the real reader.
type fakeBlockReader struct {
	blk     *pchain.Block
	inputs  map[uint32][]pchain.Input
	outputs map[uint32][]pchain.Output
	sizes   map[uint32][]uint64
}

func (b *fakeBlockReader) Block() *pchain.Block {
	return b.blk
}

func (b *fakeBlockReader) GetInputs(txIndex, from, max uint32, charge pchain.ByteCharger) ([]pchain.Input, uint32, error) {
	if int(txIndex) >= len(b.blk.Txs) {
		return nil, 0, pchain.ErrTxIndexOutOfRange
	}
	if b.blk.Txs[txIndex].Type == pchain.TxUnsupported {
		return nil, 0, pchain.ErrUnsupportedTx
	}
	ins := b.inputs[txIndex]
	sizes := b.sizes[txIndex]
	total := uint32(len(ins))
	if from > total {
		return nil, total, pchain.ErrInputIndexOutOfRange
	}
	end := total
	if max < total-from {
		end = from + max
	}
	page := make([]pchain.Input, 0, end-from)
	for i := from; i < end; i++ {
		if charge != nil && sizes[i] > 0 { // genesis UTXOs charge nothing
			if err := charge(sizes[i]); err != nil {
				return nil, total, err
			}
		}
		page = append(page, ins[i])
	}
	return page, total, nil
}

func (b *fakeBlockReader) GetOutputs(txIndex, from, max uint32) ([]pchain.Output, uint32, error) {
	if int(txIndex) >= len(b.blk.Txs) {
		return nil, 0, pchain.ErrTxIndexOutOfRange
	}
	if b.blk.Txs[txIndex].Type == pchain.TxUnsupported {
		return nil, 0, pchain.ErrUnsupportedTx
	}
	outs := b.outputs[txIndex]
	total := uint32(len(outs))
	if from > total {
		return nil, total, pchain.ErrOutputIndexOutOfRange
	}
	end := total
	if max < total-from {
		end = from + max
	}
	return outs[from:end], total, nil
}

// fakeEnv records execution invalidation.
type fakeEnv struct {
	vm.PrecompileEnvironment
	invalidated error
}

func (e *fakeEnv) InvalidateExecution(err error) {
	e.invalidated = err
}

const (
	bound       = uint64(10)
	blockHeight = uint64(10)
	genesisH    = uint64(0)
)

var (
	// byteFirst sorts before byteSecond by UTXO tx id bytes: the inputs' canonical order.
	byteFirst  = ids.ID{0x01}
	byteSecond = ids.ID{0x03}

	// Owners are the P-chain addresses of deterministic keys, so that the
	// fixture inputs can carry the owners' signatures and getStakeOwner can
	// recover their C-chain addresses; keyE owns nothing.
	testKeys                     = secp256k1.TestKeys()
	keyA, keyB, keyC, keyD, keyE = testKeys[0], testKeys[1], testKeys[2], testKeys[3], testKeys[4]
	addrA                        = keyA.Address()
	addrB                        = keyB.Address()
	addrC                        = keyC.Address()
	addrD                        = keyD.Address()

	validatorTxID   = ids.ID{0x11}
	delegatorTxID   = ids.ID{0x12}
	subnetTxID      = ids.ID{0x13}
	autoRenewedTxID = ids.ID{0x14}
	rewardTxID      = ids.ID{0x15}
	genesisTxID     = ids.ID{0x16}
	unresolvedTxID  = ids.ID{0x17}

	importTxID          = ids.ID{0x18}
	exportTxID          = ids.ID{0x19}
	baseTxID            = ids.ID{0x1a}
	rewardValidatorTxID = ids.ID{0x1b}
	cChainID            = ids.ID{0x41}
	nodeID              = ids.NodeID{0x21}
	blockID             = ids.ID{0x31}
	parentID            = ids.ID{0x30}

	// nativeAssetID is the native asset of the test chain context (FLR);
	// customAssetID an asset created on the X-chain and imported to the
	// P-chain.
	nativeAssetID = ids.ID{0x50}
	customAssetID = ids.ID{0x51}
)

// The fixture transactions use their ID bytes as unsigned bytes (any distinct
// bytes would do); their inputs carry the owners' signatures over them, in
// the plain scheme unless noted.
var testInputs = map[uint32][]pchain.Input{
	0: { // validator: canonical order byteFirst, byteSecond
		{UTXOTxID: byteFirst, UTXOOutputIndex: 0, AssetID: nativeAssetID, Amount: 700, Resolved: true, Threshold: 2, Addresses: []ids.ShortID{addrA, addrB, addrC}, SigIndices: []uint32{0, 2}, Signatures: sigs(validatorTxID[:], keyA, keyC)},
		{UTXOTxID: byteSecond, UTXOOutputIndex: 1, AssetID: nativeAssetID, Amount: 300, Resolved: true, Threshold: 1, Addresses: []ids.ShortID{addrD}, SigIndices: []uint32{0}, Signatures: sigs(validatorTxID[:], keyD)},
	},
	2: { // delegator: single owner
		{UTXOTxID: byteFirst, UTXOOutputIndex: 5, AssetID: nativeAssetID, Amount: 500, Resolved: true, Threshold: 1, Addresses: []ids.ShortID{addrB}, SigIndices: []uint32{0}, Signatures: sigs(delegatorTxID[:], keyB)},
	},
	3: { // auto-renewed: genesis input
		{UTXOTxID: ids.Empty, UTXOOutputIndex: 2, AssetID: nativeAssetID, Amount: 1000, Resolved: true, Genesis: true, Threshold: 1, Addresses: []ids.ShortID{addrA}, SigIndices: []uint32{0}, Signatures: sigs(autoRenewedTxID[:], keyA)},
	},
	5: { // legacy validator: unresolved input
		{UTXOTxID: unresolvedTxID, UTXOOutputIndex: 0, AssetID: nativeAssetID, Amount: 2000, SigIndices: []uint32{0}, Signatures: sigs(genesisTxID[:], keyA)},
	},
}

var testBlock = &pchain.Block{
	Height:    blockHeight,
	ID:        blockID,
	ParentID:  parentID,
	Timestamp: 1_700_000_000,
	Kind:      pchain.BlockStandard,
	Txs: []pchain.Tx{
		{Type: pchain.TxAddPermissionlessValidator, Index: 0, ID: validatorTxID, NodeID: nodeID, StartTime: 1_700_000_000, EndTime: 1_800_000_000, Weight: 1000, DelegationShares: 20_000, InputCount: 2, OutputCount: 2, UnsignedBytes: validatorTxID[:]},
		{Type: pchain.TxUnsupported, Index: 1, ID: subnetTxID},
		{Type: pchain.TxAddPermissionlessDelegator, Index: 2, ID: delegatorTxID, NodeID: nodeID, StartTime: 1_700_000_000, EndTime: 1_750_000_000, Weight: 500, InputCount: 1, OutputCount: 1, UnsignedBytes: delegatorTxID[:]},
		{Type: pchain.TxAddAutoRenewedValidator, Index: 3, ID: autoRenewedTxID, NodeID: nodeID, StartTime: 1_700_000_000, EndTime: 1_700_000_000 + 3600, Period: 3600, Weight: 1000, DelegationShares: 30_000, AutoCompoundRewardShares: 5, InputCount: 1, OutputCount: 1, UnsignedBytes: autoRenewedTxID[:]},
		{Type: pchain.TxRewardAutoRenewedValidator, Index: 4, ID: rewardTxID, StartTime: 1_700_000_000, EndTime: 1_700_000_000, StakerTxID: autoRenewedTxID, UnsignedBytes: rewardTxID[:]},
		{Type: pchain.TxAddValidator, Index: 5, ID: genesisTxID, NodeID: nodeID, StartTime: 5, EndTime: 6, Weight: 2000, InputCount: 1, OutputCount: 1, UnsignedBytes: genesisTxID[:]},
	},
}

// Stored sizes charged by the fake reader.
const (
	testBlockBytes    uint64 = 4_000
	genesisBlockBytes uint64 = 100
)

// testInputBytes are the producing-tx sizes of testInputs, by tx index and
// canonical input position.
var testInputBytes = map[uint32][]uint64{
	0: {300, 400},
	2: {200},
	3: {0}, // genesis UTXO: no producing transaction
	5: {700},
}

// testOutputs are the produced outputs of testBlock's transactions: the
// validator's change output, then the stake outputs.
var testOutputs = map[uint32][]pchain.Output{
	0: {
		{Kind: pchain.OutputTransfer, AssetID: nativeAssetID, Amount: 5, Threshold: 1, Addresses: []ids.ShortID{addrD}},
		{Kind: pchain.OutputStake, AssetID: nativeAssetID, Amount: 1000, Threshold: 1, Addresses: []ids.ShortID{addrA}},
	},
	2: {{Kind: pchain.OutputStake, AssetID: nativeAssetID, Amount: 500, Threshold: 1, Addresses: []ids.ShortID{addrB}}},
	3: {{Kind: pchain.OutputStake, AssetID: nativeAssetID, Amount: 1000, Threshold: 1, Addresses: []ids.ShortID{addrA}}},
	5: {{Kind: pchain.OutputStake, AssetID: nativeAssetID, Amount: 2000, Threshold: 2, Addresses: []ids.ShortID{addrA, addrB}}},
}

// decodedBytes is the block size plus the producing-tx sizes of the given
// canonical input positions of tx txIndex.
func decodedBytes(txIndex uint32, inputs ...int) uint64 {
	total := testBlockBytes
	for _, i := range inputs {
		total += testInputBytes[txIndex][i]
	}
	return total
}

// The transfer block, a Proposal block whose proposal is a reward-validator
// transaction, holds the value-moving types: an import of a custom asset from
// the C-chain paid for by a native P-chain input, an export to the C-chain
// and a plain transfer.
const (
	transferHeight     = uint64(9)
	transferBlockBytes = uint64(2_500)
	transferTimestamp  = uint64(1_699_999_000)
)

var transferBlock = &pchain.Block{
	Height:    transferHeight,
	ID:        ids.ID{0x2f},
	ParentID:  ids.ID{0x2e},
	Timestamp: transferTimestamp,
	Kind:      pchain.BlockProposal,
	Txs: []pchain.Tx{
		{Type: pchain.TxImport, Index: 0, ID: importTxID, StartTime: transferTimestamp, RemoteChainID: cChainID, InputCount: 2, OutputCount: 1, UnsignedBytes: importTxID[:]},
		{Type: pchain.TxExport, Index: 1, ID: exportTxID, StartTime: transferTimestamp, RemoteChainID: cChainID, InputCount: 1, OutputCount: 2, UnsignedBytes: exportTxID[:]},
		{Type: pchain.TxBase, Index: 2, ID: baseTxID, StartTime: transferTimestamp, InputCount: 1, OutputCount: 2, UnsignedBytes: baseTxID[:]},
		{Type: pchain.TxRewardValidator, Index: 3, ID: rewardValidatorTxID, StartTime: transferTimestamp, EndTime: transferTimestamp, StakerTxID: validatorTxID, UnsignedBytes: rewardValidatorTxID[:]},
	},
}

// transferInputs are in canonical order: the P-chain inputs first, then the
// imported inputs. The import brings in a custom asset: its native P-chain
// input pays the fee, the imported input and the output carry customAssetID.
// The plain transfer is signed in the Ethereum scheme.
var transferInputs = map[uint32][]pchain.Input{
	0: {
		{UTXOTxID: byteFirst, UTXOOutputIndex: 7, AssetID: nativeAssetID, Amount: 10, Resolved: true, Threshold: 1, Addresses: []ids.ShortID{addrA}, SigIndices: []uint32{0}, Signatures: sigs(importTxID[:], keyA)},
		{UTXOTxID: byteSecond, UTXOOutputIndex: 0, AssetID: customAssetID, Amount: 990, Imported: true, SigIndices: []uint32{0}, Signatures: sigs(importTxID[:], keyB)},
	},
	1: {{UTXOTxID: byteFirst, UTXOOutputIndex: 8, AssetID: nativeAssetID, Amount: 1000, Resolved: true, Threshold: 1, Addresses: []ids.ShortID{addrC}, SigIndices: []uint32{0}, Signatures: sigs(exportTxID[:], keyC)}},
	2: {{UTXOTxID: byteSecond, UTXOOutputIndex: 2, AssetID: nativeAssetID, Amount: 50, Resolved: true, Threshold: 1, Addresses: []ids.ShortID{addrB}, SigIndices: []uint32{0}, Signatures: ethSigs(baseTxID[:], keyB)}},
}

// transferInputBytes are the producing-tx sizes of transferInputs; imported
// inputs have no producing P-chain transaction.
var transferInputBytes = map[uint32][]uint64{
	0: {300, 0},
	1: {300},
	2: {400},
}

var transferOutputs = map[uint32][]pchain.Output{
	0: {{Kind: pchain.OutputTransfer, AssetID: customAssetID, Amount: 990, Threshold: 1, Addresses: []ids.ShortID{addrB}}},
	1: {
		{Kind: pchain.OutputTransfer, AssetID: nativeAssetID, Amount: 100, Threshold: 1, Addresses: []ids.ShortID{addrC}},
		{Kind: pchain.OutputExported, AssetID: nativeAssetID, Amount: 899, Threshold: 1, Addresses: []ids.ShortID{addrD}},
	},
	2: {
		{Kind: pchain.OutputTransfer, AssetID: nativeAssetID, Amount: 30, Threshold: 1, Addresses: []ids.ShortID{addrA}},
		{Kind: pchain.OutputTransfer, AssetID: nativeAssetID, Amount: 19, Locktime: 7, StakeableLocktime: 99, Threshold: 2, Addresses: []ids.ShortID{addrA, addrB}},
	},
}

// transferDecodedBytes is the transfer block's size plus the producing-tx
// sizes of the given canonical input positions of tx txIndex.
func transferDecodedBytes(txIndex uint32, inputs ...int) uint64 {
	total := transferBlockBytes
	for _, i := range inputs {
		total += transferInputBytes[txIndex][i]
	}
	return total
}

// sigs signs unsignedBytes with keys, in order, in the plain scheme (over the
// sha256 of the bytes, as the P-chain wallets do); ethSigs in the Ethereum
// scheme (over the Ethereum signed-message hash of the hex-encoded digest, as
// Ethereum wallets do).
func sigs(unsignedBytes []byte, keys ...*secp256k1.PrivateKey) [][pchain.SignatureLen]byte {
	return signAll(secp256k1fx.NewSigningHashes(unsignedBytes).Plain, keys)
}

func ethSigs(unsignedBytes []byte, keys ...*secp256k1.PrivateKey) [][pchain.SignatureLen]byte {
	return signAll(secp256k1fx.NewSigningHashes(unsignedBytes).EthPrefixed, keys)
}

func signAll(hash []byte, keys []*secp256k1.PrivateKey) [][pchain.SignatureLen]byte {
	signatures := make([][pchain.SignatureLen]byte, len(keys))
	for i, key := range keys {
		sig, err := key.SignHash(hash)
		if err != nil {
			panic(err)
		}
		copy(signatures[i][:], sig)
	}
	return signatures
}

// ownerlessInput is a located output with threshold 0: it has no owner
// addresses and can be spent without signatures (secp256k1fx permits it).
var ownerlessInput = pchain.Input{UTXOTxID: byteFirst, UTXOOutputIndex: 9, AssetID: nativeAssetID, Amount: 500, Resolved: true}

// newReaderWithInput serves the fixture block with transaction txIndex funded
// by input alone instead of its usual inputs.
func newReaderWithInput(txIndex uint32, input pchain.Input) *fakeReader {
	r := newFakeReader()
	r.inputs = map[uint64]map[uint32][]pchain.Input{blockHeight: {txIndex: {input}}}
	r.inputBytes = map[uint64]map[uint32][]uint64{blockHeight: {txIndex: {testInputBytes[txIndex][0]}}}
	return r
}

// newOwnerlessReader serves the fixture block with the delegator (tx 2)
// funded by ownerlessInput instead of its usual single-owner input.
func newOwnerlessReader() *fakeReader {
	return newReaderWithInput(2, ownerlessInput)
}

func newFakeReader() *fakeReader {
	return &fakeReader{
		last: bound,
		blocks: map[uint64]*pchain.Block{
			blockHeight:    testBlock,
			transferHeight: transferBlock,
			genesisH:       {Height: 0, ID: parentID, Kind: pchain.BlockStandard},
		},
		inputs:     map[uint64]map[uint32][]pchain.Input{blockHeight: testInputs, transferHeight: transferInputs},
		outputs:    map[uint64]map[uint32][]pchain.Output{blockHeight: testOutputs, transferHeight: transferOutputs},
		blockBytes: map[uint64]uint64{blockHeight: testBlockBytes, transferHeight: transferBlockBytes, genesisH: genesisBlockBytes},
		inputBytes: map[uint64]map[uint32][]uint64{blockHeight: testInputBytes, transferHeight: transferInputBytes},
	}
}

func snowContextWithReader(t testing.TB, reader pchain.Reader) *snow.Context {
	ctx := snowtest.Context(t, snowtest.CChainID)
	ctx.AVAXAssetID = nativeAssetID
	ctx.PChainReader = reader
	return ctx
}

func withBound(height uint64) func(*contract.MockBlockContext) {
	return func(mbc *contract.MockBlockContext) {
		mbc.EXPECT().PChainHeight().Return(height, true).AnyTimes()
	}
}

func withoutBound(mbc *contract.MockBlockContext) {
	mbc.EXPECT().PChainHeight().Return(uint64(0), false).AnyTimes()
}

// mustPack returns a function that fails the test on a packing error, so that
// multi-value Pack* results can be used inline: mustPack(t)(PackGetBlock(1)).
func mustPack(t testing.TB) func(packed []byte, err error) []byte {
	return func(packed []byte, err error) []byte {
		t.Helper()
		require.NoError(t, err)
		return packed
	}
}

// stripSelector removes the 4-byte function selector from a packed call.
func stripSelector(packed []byte) []byte {
	return packed[contract.SelectorLen:]
}

// expectedTx is transaction index of the staking fixture block with its
// first maxInputs inputs and maxOutputs outputs inline.
func expectedTx(t testing.TB, index, maxInputs, maxOutputs uint32) Tx {
	t.Helper()
	return fixtureTx(testBlock, testInputs, testOutputs, index, maxInputs, maxOutputs)
}

// expectedTransferTx is transaction index of the transfer fixture block.
func expectedTransferTx(t testing.TB, index, maxInputs, maxOutputs uint32) Tx {
	t.Helper()
	return fixtureTx(transferBlock, transferInputs, transferOutputs, index, maxInputs, maxOutputs)
}

func fixtureTx(blk *pchain.Block, inputs map[uint32][]pchain.Input, outputs map[uint32][]pchain.Output, index, maxInputs, maxOutputs uint32) Tx {
	ins := inputs[index]
	if uint32(len(ins)) > maxInputs {
		ins = ins[:maxInputs]
	}
	outs := outputs[index]
	if uint32(len(outs)) > maxOutputs {
		outs = outs[:maxOutputs]
	}
	return toTx(blk.Txs[index], ins, outs)
}

func TestABISelectorsAndSignatures(t *testing.T) {
	require := require.New(t)
	want := map[string]string{
		"getPChainHeight":       "getPChainHeight()",
		"getNativeAssetId":      "getNativeAssetId()",
		"getBlock":              "getBlock(uint64)",
		"getTransactions":       "getTransactions(uint64,uint32,uint32,uint256,uint32,uint32)",
		"getTransactionInputs":  "getTransactionInputs(uint64,uint32,uint32,uint32)",
		"getTransactionOutputs": "getTransactionOutputs(uint64,uint32,uint32,uint32)",
		"getStakeOwner":         "getStakeOwner(uint64,uint32)",
	}
	require.Len(PChainGatewayABI.Methods, len(want))
	for name, sig := range want {
		method, ok := PChainGatewayABI.Methods[name]
		require.True(ok, name)
		require.Equal(sig, method.Sig)
		require.Equal(crypto.Keccak256([]byte(sig))[:4], method.ID)
	}
}

func TestPackUnpackRoundTrips(t *testing.T) {
	require := require.New(t)

	// Nested dynamic tuples must survive a round trip through the hand-written ABI.
	txsOut := GetTransactionsOutput{
		Txs:       []Tx{expectedTx(t, 0, 8, 8), expectedTx(t, 3, 8, 0), toTx(testBlock.Txs[4], nil, nil), expectedTransferTx(t, 0, 8, 8), expectedTransferTx(t, 1, 0, 8)},
		NextIndex: 5,
	}
	packed, err := PackGetTransactionsOutput(txsOut)
	require.NoError(err)
	unpacked, err := UnpackGetTransactionsOutput(packed)
	require.NoError(err)
	require.Equal(txsOut, unpacked)

	inputsOut := GetTransactionInputsOutput{Inputs: []Input{toInput(testInputs[0][0]), toInput(testInputs[5][0])}, InputCount: 7}
	packed, err = PackGetTransactionInputsOutput(inputsOut)
	require.NoError(err)
	unpackedInputs, err := UnpackGetTransactionInputsOutput(packed)
	require.NoError(err)
	require.Equal(inputsOut, unpackedInputs)

	outputsOut := GetTransactionOutputsOutput{Outputs: []Output{toOutput(transferOutputs[2][0]), toOutput(transferOutputs[2][1])}, OutputCount: 9}
	packed, err = PackGetTransactionOutputsOutput(outputsOut)
	require.NoError(err)
	unpackedOutputs, err := UnpackGetTransactionOutputsOutput(packed)
	require.NoError(err)
	require.Equal(outputsOut, unpackedOutputs)

	blockOut := GetBlockOutput{BlockId: blockID, ParentId: parentID, Timestamp: 7, Kind: 1, TxCount: 9}
	packed, err = PackGetBlockOutput(blockOut)
	require.NoError(err)
	unpackedBlock, err := UnpackGetBlockOutput(packed)
	require.NoError(err)
	require.Equal(blockOut, unpackedBlock)

	ownerOut := GetStakeOwnerOutput{PChainOwner: addrA, CChainOwner: keyA.EthAddress(), Resolved: true}
	packed, err = PackGetStakeOwnerOutput(ownerOut)
	require.NoError(err)
	unpackedOwner, err := UnpackGetStakeOwnerOutput(packed)
	require.NoError(err)
	require.Equal(ownerOut, unpackedOwner)

	in := GetTransactionsInput{Height: 1, FromIndex: 2, MaxTxs: 3, TypeFilter: big.NewInt(4), MaxInputsPerTx: 5, MaxOutputsPerTx: 6}
	packed, err = PackGetTransactions(in)
	require.NoError(err)
	unpackedIn, err := UnpackGetTransactionsInput(stripSelector(packed))
	require.NoError(err)
	require.Equal(in, unpackedIn)

	outputsIn := GetTransactionOutputsInput{Height: 1, TxIndex: 2, FromOutput: 3, MaxOutputs: 4}
	packed, err = PackGetTransactionOutputs(outputsIn)
	require.NoError(err)
	unpackedOutputsIn, err := UnpackGetTransactionOutputsInput(stripSelector(packed))
	require.NoError(err)
	require.Equal(outputsIn, unpackedOutputsIn)

	height, err := UnpackGetPChainHeightOutput(mustPack(t)(PackGetPChainHeightOutput(42)))
	require.NoError(err)
	require.Equal(uint64(42), height)

	assetID, err := UnpackGetNativeAssetIdOutput(mustPack(t)(PackGetNativeAssetIdOutput(nativeAssetID)))
	require.NoError(err)
	require.Equal([32]byte(nativeAssetID), assetID)
}

func TestGetPChainHeight(t *testing.T) {
	tests := []precompiletest.PrecompileTest{
		{
			Name:              "returns committed height",
			InputFn:           func(t testing.TB) []byte { return mustPack(t)(PackGetPChainHeight()) },
			SetupBlockContext: withBound(bound),
			SuppliedGas:       GetPChainHeightGas,
			ReadOnly:          true,
			ExpectedRes:       mustPack(t)(PackGetPChainHeightOutput(bound)),
		},
		{
			Name:              "reverts without committed height",
			InputFn:           func(t testing.TB) []byte { return mustPack(t)(PackGetPChainHeight()) },
			SetupBlockContext: withoutBound,
			SuppliedGas:       GetPChainHeightGas,
			ReadOnly:          true,
			ExpectedRes:       revertReason(reasonHeightNotCommitted),
			ExpectedErr:       vm.ErrExecutionReverted,
		},
		{
			Name:              "out of gas",
			InputFn:           func(t testing.TB) []byte { return mustPack(t)(PackGetPChainHeight()) },
			SetupBlockContext: withBound(bound),
			SuppliedGas:       GetPChainHeightGas - 1,
			ReadOnly:          true,
			ExpectedErr:       vm.ErrOutOfGas,
		},
	}
	precompiletest.RunPrecompileTests(t, Module, tests)
}

func TestGetNativeAssetId(t *testing.T) {
	snowCtx := snowContextWithReader(t, newFakeReader())

	tests := []precompiletest.PrecompileTest{
		{
			Name:        "returns the chain context's native asset",
			InputFn:     func(t testing.TB) []byte { return mustPack(t)(PackGetNativeAssetId()) },
			SnowContext: snowCtx,
			SuppliedGas: GetNativeAssetIdGas,
			ReadOnly:    true,
			ExpectedRes: mustPack(t)(PackGetNativeAssetIdOutput(nativeAssetID)),
		},
		{
			Name:        "out of gas",
			InputFn:     func(t testing.TB) []byte { return mustPack(t)(PackGetNativeAssetId()) },
			SnowContext: snowCtx,
			SuppliedGas: GetNativeAssetIdGas - 1,
			ReadOnly:    true,
			ExpectedErr: vm.ErrOutOfGas,
		},
	}
	precompiletest.RunPrecompileTests(t, Module, tests)
}

// TestGetNativeAssetIdWithoutChainContextInvalidates covers the node-local
// failure path directly: the test harness always supplies a chain context.
func TestGetNativeAssetIdWithoutChainContextInvalidates(t *testing.T) {
	require := require.New(t)
	ctrl := gomock.NewController(t)
	env := &fakeEnv{}
	accessibleState := contract.NewMockAccessibleState(ctrl)
	accessibleState.EXPECT().GetSnowContext().Return(nil).AnyTimes()
	accessibleState.EXPECT().GetPrecompileEnv().Return(env).AnyTimes()

	input := stripSelector(mustPack(t)(PackGetNativeAssetId()))
	res, remainingGas, err := getNativeAssetId(accessibleState, common.Address{}, ContractAddress, input, GetNativeAssetIdGas, true)
	require.ErrorIs(err, errChainContextUnavailable)
	require.ErrorIs(env.invalidated, errChainContextUnavailable)
	require.Nil(res)
	require.Zero(remainingGas)
}

// TestAssetIdsAreExposed pins the asset IDs of the transfer block's import
// against literal values on every path that returns inputs or outputs: the
// native fee input and the custom imported input (and its output) look alike
// in every other field.
func TestAssetIdsAreExposed(t *testing.T) {
	snowCtx := snowContextWithReader(t, newFakeReader())
	importOnly := new(big.Int).SetBit(big.NewInt(0), int(pchain.TxImport), 1)

	feeInput := Input{UtxoTxId: byteFirst, UtxoOutputIndex: 7, AssetId: nativeAssetID, Amount: 10, Resolved: true, Threshold: 1, Addresses: [][20]byte{addrA}, SigIndices: []uint32{0}}
	importedInput := Input{UtxoTxId: byteSecond, UtxoOutputIndex: 0, AssetId: customAssetID, Amount: 990, Imported: true, Addresses: [][20]byte{}, SigIndices: []uint32{0}}
	importedOutput := Output{Kind: uint8(pchain.OutputTransfer), AssetId: customAssetID, Amount: 990, Threshold: 1, Addresses: [][20]byte{addrB}}

	tests := []precompiletest.PrecompileTest{
		{
			Name: "inline inputs and outputs of getTransactions",
			InputFn: func(t testing.TB) []byte {
				return mustPack(t)(PackGetTransactions(GetTransactionsInput{Height: transferHeight, FromIndex: 0, MaxTxs: 1, TypeFilter: importOnly, MaxInputsPerTx: 8, MaxOutputsPerTx: 8}))
			},
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionsGas(1, 8, 8, transferDecodedBytes(0, 0), 1+1),
			ReadOnly:          true,
			ExpectedRes: mustPack(t)(PackGetTransactionsOutput(GetTransactionsOutput{
				Txs: []Tx{{
					TxType:        uint8(pchain.TxImport),
					Index:         0,
					TxId:          importTxID,
					StartTime:     transferTimestamp,
					RemoteChainId: cChainID,
					InputCount:    2,
					OutputCount:   1,
					Inputs:        []Input{feeInput, importedInput},
					Outputs:       []Output{importedOutput},
				}},
				NextIndex: 1,
			})),
		},
		{
			Name: "getTransactionInputs",
			InputFn: func(t testing.TB) []byte {
				return mustPack(t)(PackGetTransactionInputs(GetTransactionInputsInput{Height: transferHeight, TxIndex: 0, FromInput: 0, MaxInputs: 64}))
			},
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionInputsGas(64, transferDecodedBytes(0, 0), 1),
			ReadOnly:          true,
			ExpectedRes: mustPack(t)(PackGetTransactionInputsOutput(GetTransactionInputsOutput{
				Inputs:     []Input{feeInput, importedInput},
				InputCount: 2,
			})),
		},
		{
			Name: "getTransactionOutputs",
			InputFn: func(t testing.TB) []byte {
				return mustPack(t)(PackGetTransactionOutputs(GetTransactionOutputsInput{Height: transferHeight, TxIndex: 0, FromOutput: 0, MaxOutputs: 64}))
			},
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionOutputsGas(64, transferBlockBytes, 1),
			ReadOnly:          true,
			ExpectedRes: mustPack(t)(PackGetTransactionOutputsOutput(GetTransactionOutputsOutput{
				Outputs:     []Output{importedOutput},
				OutputCount: 1,
			})),
		},
	}
	precompiletest.RunPrecompileTests(t, Module, tests)
}

func TestGetBlock(t *testing.T) {
	reader := newFakeReader()
	snowCtx := snowContextWithReader(t, reader)
	env := &fakeEnv{}

	tests := []precompiletest.PrecompileTest{
		{
			Name:              "block at bound",
			InputFn:           func(t testing.TB) []byte { return mustPack(t)(PackGetBlock(blockHeight)) },
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetBlockGas(testBlockBytes),
			ReadOnly:          true,
			ExpectedRes: mustPack(t)(PackGetBlockOutput(GetBlockOutput{
				BlockId: blockID, ParentId: parentID, Timestamp: 1_700_000_000, Kind: uint8(pchain.BlockStandard), TxCount: 6,
			})),
		},
		{
			Name:              "genesis block",
			InputFn:           func(t testing.TB) []byte { return mustPack(t)(PackGetBlock(0)) },
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetBlockGas(genesisBlockBytes),
			ReadOnly:          true,
			ExpectedRes:       mustPack(t)(PackGetBlockOutput(GetBlockOutput{BlockId: parentID})),
		},
		{
			Name:              "insufficient gas for the block bytes",
			InputFn:           func(t testing.TB) []byte { return mustPack(t)(PackGetBlock(blockHeight)) },
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetBlockGas(testBlockBytes) - 1,
			ReadOnly:          true,
			ExpectedErr:       vm.ErrOutOfGas,
		},
		{
			Name:              "height above bound reverts",
			InputFn:           func(t testing.TB) []byte { return mustPack(t)(PackGetBlock(bound + 1)) },
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetBlockGas(0),
			ReadOnly:          true,
			ExpectedRes:       revertReason(reasonHeightAboveBound),
			ExpectedErr:       vm.ErrExecutionReverted,
		},
		{
			Name:              "no committed height reverts",
			InputFn:           func(t testing.TB) []byte { return mustPack(t)(PackGetBlock(1)) },
			SetupBlockContext: withoutBound,
			SnowContext:       snowCtx,
			SuppliedGas:       GetBlockGas(0),
			ReadOnly:          true,
			ExpectedRes:       revertReason(reasonHeightNotCommitted),
			ExpectedErr:       vm.ErrExecutionReverted,
		},
		{
			Name:              "malformed input reverts",
			InputFn:           func(t testing.TB) []byte { return mustPack(t)(PackGetBlock(1))[:6] },
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetBlockGas(0),
			ReadOnly:          true,
			ExpectedRes:       revertReason(reasonInvalidInput),
			ExpectedErr:       vm.ErrExecutionReverted,
		},
		{
			Name:              "height not accepted locally invalidates execution",
			InputFn:           func(t testing.TB) []byte { return mustPack(t)(PackGetBlock(bound)) },
			SetupBlockContext: withBound(bound + 5), // header bound ahead of the lagging local P-chain
			SnowContext: snowContextWithReader(t, &fakeReader{
				last:   bound - 1,
				blocks: map[uint64]*pchain.Block{},
			}),
			PrecompileEnv: env,
			SuppliedGas:   GetBlockGas(0),
			ReadOnly:      true,
			ExpectedErr:   pchain.ErrNotAccepted,
			AfterHook: func(t testing.TB, _ contract.StateDB) {
				require.ErrorIs(t, env.invalidated, pchain.ErrNotAccepted)
			},
		},
		{
			Name:              "missing reader invalidates execution",
			InputFn:           func(t testing.TB) []byte { return mustPack(t)(PackGetBlock(1)) },
			SetupBlockContext: withBound(bound),
			SnowContext:       snowtest.Context(t, snowtest.CChainID),
			PrecompileEnv:     env,
			SuppliedGas:       GetBlockGas(0),
			ReadOnly:          true,
			ExpectedErr:       errPChainReaderUnavailable,
		},
	}
	precompiletest.RunPrecompileTests(t, Module, tests)
}

func TestGetTransactions(t *testing.T) {
	reader := newFakeReader()
	snowCtx := snowContextWithReader(t, reader)

	call := func(in GetTransactionsInput) func(testing.TB) []byte {
		return func(t testing.TB) []byte { return mustPack(t)(PackGetTransactions(in)) }
	}
	allTypes := big.NewInt(0)
	validatorsOnly := new(big.Int).SetBit(big.NewInt(0), int(pchain.TxAddPermissionlessValidator), 1)
	autoRenew := new(big.Int).SetBit(big.NewInt(0), int(pchain.TxAddAutoRenewedValidator), 1)
	autoRenew.SetBit(autoRenew, int(pchain.TxRewardAutoRenewedValidator), 1)
	transfers := new(big.Int).SetBit(big.NewInt(0), int(pchain.TxImport), 1)
	transfers.SetBit(transfers, int(pchain.TxExport), 1)
	transfers.SetBit(transfers, int(pchain.TxBase), 1)
	rewardValidators := new(big.Int).SetBit(big.NewInt(0), int(pchain.TxRewardValidator), 1)

	tests := []precompiletest.PrecompileTest{
		{
			Name:              "whole block, all types, inline inputs",
			InputFn:           call(GetTransactionsInput{Height: blockHeight, FromIndex: 0, MaxTxs: 32, TypeFilter: allTypes, MaxInputsPerTx: 8}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionsGas(32, 8, 0, decodedBytes(0, 0, 1)+testInputBytes[2][0]+testInputBytes[3][0]+testInputBytes[5][0], 3+1+1+1), // all inline inputs
			ReadOnly:          true,
			ExpectedRes: mustPack(t)(PackGetTransactionsOutput(GetTransactionsOutput{
				Txs:       []Tx{expectedTx(t, 0, 8, 0), expectedTx(t, 2, 8, 0), expectedTx(t, 3, 8, 0), toTx(testBlock.Txs[4], nil, nil), expectedTx(t, 5, 8, 0)},
				NextIndex: 6,
			})),
		},
		{
			Name:              "type filter",
			InputFn:           call(GetTransactionsInput{Height: blockHeight, FromIndex: 0, MaxTxs: 32, TypeFilter: autoRenew, MaxInputsPerTx: 0}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionsGas(32, 0, 0, testBlockBytes, 0),
			ReadOnly:          true,
			ExpectedRes: mustPack(t)(PackGetTransactionsOutput(GetTransactionsOutput{
				Txs:       []Tx{toTx(testBlock.Txs[3], nil, nil), toTx(testBlock.Txs[4], nil, nil)},
				NextIndex: 6,
			})),
		},
		{
			Name:              "pagination stops after maxTxs and reports nextIndex",
			InputFn:           call(GetTransactionsInput{Height: blockHeight, FromIndex: 0, MaxTxs: 2, TypeFilter: allTypes, MaxInputsPerTx: 1}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionsGas(2, 1, 0, decodedBytes(0, 0)+testInputBytes[2][0], 3+1),
			ReadOnly:          true,
			ExpectedRes: mustPack(t)(PackGetTransactionsOutput(GetTransactionsOutput{
				Txs:       []Tx{expectedTx(t, 0, 1, 0), expectedTx(t, 2, 1, 0)},
				NextIndex: 3,
			})),
		},
		{
			Name:              "resume from nextIndex",
			InputFn:           call(GetTransactionsInput{Height: blockHeight, FromIndex: 3, MaxTxs: 2, TypeFilter: validatorsOnly, MaxInputsPerTx: 8}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionsGas(2, 8, 0, testBlockBytes, 0),
			ReadOnly:          true,
			ExpectedRes:       mustPack(t)(PackGetTransactionsOutput(GetTransactionsOutput{Txs: []Tx{}, NextIndex: 6})),
		},
		{
			Name:              "fromIndex equal to txCount is an empty page",
			InputFn:           call(GetTransactionsInput{Height: blockHeight, FromIndex: 6, MaxTxs: 1, TypeFilter: allTypes, MaxInputsPerTx: 0}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionsGas(1, 0, 0, testBlockBytes, 0),
			ReadOnly:          true,
			ExpectedRes:       mustPack(t)(PackGetTransactionsOutput(GetTransactionsOutput{Txs: []Tx{}, NextIndex: 6})),
		},
		{
			Name:              "fromIndex above txCount reverts",
			InputFn:           call(GetTransactionsInput{Height: blockHeight, FromIndex: 7, MaxTxs: 1, TypeFilter: allTypes, MaxInputsPerTx: 0}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionsGas(1, 0, 0, testBlockBytes, 0),
			ReadOnly:          true,
			ExpectedRes:       revertReason(reasonTxIndexOutOfRange),
			ExpectedErr:       vm.ErrExecutionReverted,
		},
		{
			Name:              "insufficient gas for a producing transaction",
			InputFn:           call(GetTransactionsInput{Height: blockHeight, FromIndex: 0, MaxTxs: 1, TypeFilter: allTypes, MaxInputsPerTx: 8}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionsGas(1, 8, 0, decodedBytes(0, 0, 1), 0) - 1, // one byte short of the second input's tx
			ReadOnly:          true,
			ExpectedErr:       vm.ErrOutOfGas,
		},
		{
			Name:              "maxTxs above cap reverts",
			InputFn:           call(GetTransactionsInput{Height: blockHeight, FromIndex: 0, MaxTxs: MaxTxsPerCall + 1, TypeFilter: allTypes, MaxInputsPerTx: 0}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       BlockReadGas,
			ReadOnly:          true,
			ExpectedRes:       revertReason(reasonPageSizeExceedsCap),
			ExpectedErr:       vm.ErrExecutionReverted,
		},
		{
			Name:              "maxInputsPerTx above cap reverts",
			InputFn:           call(GetTransactionsInput{Height: blockHeight, FromIndex: 0, MaxTxs: 1, TypeFilter: allTypes, MaxInputsPerTx: MaxInlineInputsPerTx + 1}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       BlockReadGas,
			ReadOnly:          true,
			ExpectedRes:       revertReason(reasonPageSizeExceedsCap),
			ExpectedErr:       vm.ErrExecutionReverted,
		},
		{
			Name:              "height above bound reverts after charging the page",
			InputFn:           call(GetTransactionsInput{Height: bound + 1, FromIndex: 0, MaxTxs: 4, TypeFilter: allTypes, MaxInputsPerTx: 2}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionsGas(4, 2, 0, 0, 0),
			ReadOnly:          true,
			ExpectedRes:       revertReason(reasonHeightAboveBound),
			ExpectedErr:       vm.ErrExecutionReverted,
		},
		{
			Name:              "insufficient gas for the page",
			InputFn:           call(GetTransactionsInput{Height: blockHeight, FromIndex: 0, MaxTxs: 4, TypeFilter: allTypes, MaxInputsPerTx: 2}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionsGas(4, 2, 0, 0, 0) - 1,
			ReadOnly:          true,
			ExpectedErr:       vm.ErrOutOfGas,
		},
		{
			Name:              "inline outputs",
			InputFn:           call(GetTransactionsInput{Height: blockHeight, FromIndex: 0, MaxTxs: 32, TypeFilter: allTypes, MaxInputsPerTx: 0, MaxOutputsPerTx: 8}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionsGas(32, 0, 8, testBlockBytes, 2+1+1+2), // output owner addresses of txs 0, 2, 3, 5
			ReadOnly:          true,
			ExpectedRes: mustPack(t)(PackGetTransactionsOutput(GetTransactionsOutput{
				Txs:       []Tx{expectedTx(t, 0, 0, 8), expectedTx(t, 2, 0, 8), expectedTx(t, 3, 0, 8), toTx(testBlock.Txs[4], nil, nil), expectedTx(t, 5, 0, 8)},
				NextIndex: 6,
			})),
		},
		{
			Name:              "maxOutputsPerTx truncates the inline outputs",
			InputFn:           call(GetTransactionsInput{Height: blockHeight, FromIndex: 0, MaxTxs: 1, TypeFilter: allTypes, MaxInputsPerTx: 0, MaxOutputsPerTx: 1}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionsGas(1, 0, 1, testBlockBytes, 1), // the change output's single owner
			ReadOnly:          true,
			ExpectedRes: mustPack(t)(PackGetTransactionsOutput(GetTransactionsOutput{
				Txs:       []Tx{expectedTx(t, 0, 0, 1)},
				NextIndex: 1,
			})),
		},
		{
			Name:              "transfers with inline inputs and outputs",
			InputFn:           call(GetTransactionsInput{Height: transferHeight, FromIndex: 0, MaxTxs: 32, TypeFilter: allTypes, MaxInputsPerTx: 8, MaxOutputsPerTx: 8}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			// The imported input has no producing transaction and no owner;
			// 3 input owners plus 6 output owners are returned.
			SuppliedGas: GetTransactionsGas(32, 8, 8, transferDecodedBytes(0, 0, 1)+transferInputBytes[1][0]+transferInputBytes[2][0], 3+6),
			ReadOnly:    true,
			ExpectedRes: mustPack(t)(PackGetTransactionsOutput(GetTransactionsOutput{
				Txs:       []Tx{expectedTransferTx(t, 0, 8, 8), expectedTransferTx(t, 1, 8, 8), expectedTransferTx(t, 2, 8, 8), toTx(transferBlock.Txs[3], nil, nil)},
				NextIndex: 4,
			})),
		},
		{
			Name:              "type filter selects the transfer types",
			InputFn:           call(GetTransactionsInput{Height: transferHeight, FromIndex: 0, MaxTxs: 32, TypeFilter: transfers, MaxInputsPerTx: 0, MaxOutputsPerTx: 0}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionsGas(32, 0, 0, transferBlockBytes, 0),
			ReadOnly:          true,
			ExpectedRes: mustPack(t)(PackGetTransactionsOutput(GetTransactionsOutput{
				Txs:       []Tx{toTx(transferBlock.Txs[0], nil, nil), toTx(transferBlock.Txs[1], nil, nil), toTx(transferBlock.Txs[2], nil, nil)},
				NextIndex: 4,
			})),
		},
		{
			Name:              "type filter selects reward-validator transactions",
			InputFn:           call(GetTransactionsInput{Height: transferHeight, FromIndex: 0, MaxTxs: 32, TypeFilter: rewardValidators, MaxInputsPerTx: 8, MaxOutputsPerTx: 8}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionsGas(32, 8, 8, transferBlockBytes, 0),
			ReadOnly:          true,
			ExpectedRes: mustPack(t)(PackGetTransactionsOutput(GetTransactionsOutput{
				Txs:       []Tx{toTx(transferBlock.Txs[3], nil, nil)},
				NextIndex: 4,
			})),
		},
		{
			Name:              "maxOutputsPerTx above cap reverts",
			InputFn:           call(GetTransactionsInput{Height: blockHeight, FromIndex: 0, MaxTxs: 1, TypeFilter: allTypes, MaxInputsPerTx: 0, MaxOutputsPerTx: MaxInlineOutputsPerTx + 1}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       BlockReadGas,
			ReadOnly:          true,
			ExpectedRes:       revertReason(reasonPageSizeExceedsCap),
			ExpectedErr:       vm.ErrExecutionReverted,
		},
	}
	precompiletest.RunPrecompileTests(t, Module, tests)
}

func TestGetTransactionInputs(t *testing.T) {
	reader := newFakeReader()
	snowCtx := snowContextWithReader(t, reader)

	call := func(in GetTransactionInputsInput) func(testing.TB) []byte {
		return func(t testing.TB) []byte { return mustPack(t)(PackGetTransactionInputs(in)) }
	}
	tests := []precompiletest.PrecompileTest{
		{
			Name:              "all inputs, in canonical order",
			InputFn:           call(GetTransactionInputsInput{Height: blockHeight, TxIndex: 0, FromInput: 0, MaxInputs: 64}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionInputsGas(64, decodedBytes(0, 0, 1), 4),
			ReadOnly:          true,
			ExpectedRes: mustPack(t)(PackGetTransactionInputsOutput(GetTransactionInputsOutput{
				Inputs:     []Input{toInput(testInputs[0][0]), toInput(testInputs[0][1])},
				InputCount: 2,
			})),
		},
		{
			Name:              "paging",
			InputFn:           call(GetTransactionInputsInput{Height: blockHeight, TxIndex: 0, FromInput: 1, MaxInputs: 1}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionInputsGas(1, decodedBytes(0, 1), 1),
			ReadOnly:          true,
			ExpectedRes: mustPack(t)(PackGetTransactionInputsOutput(GetTransactionInputsOutput{
				Inputs:     []Input{toInput(testInputs[0][1])},
				InputCount: 2,
			})),
		},
		{
			Name:              "unresolved input is returned as such",
			InputFn:           call(GetTransactionInputsInput{Height: blockHeight, TxIndex: 5, FromInput: 0, MaxInputs: 8}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionInputsGas(8, decodedBytes(5, 0), 0),
			ReadOnly:          true,
			ExpectedRes: mustPack(t)(PackGetTransactionInputsOutput(GetTransactionInputsOutput{
				Inputs:     []Input{toInput(testInputs[5][0])},
				InputCount: 1,
			})),
		},
		{
			Name:              "maxInputs above cap reverts",
			InputFn:           call(GetTransactionInputsInput{Height: blockHeight, TxIndex: 0, FromInput: 0, MaxInputs: MaxInputsPerCall + 1}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       BlockReadGas,
			ReadOnly:          true,
			ExpectedRes:       revertReason(reasonPageSizeExceedsCap),
			ExpectedErr:       vm.ErrExecutionReverted,
		},
		{
			Name:              "empty page",
			InputFn:           call(GetTransactionInputsInput{Height: blockHeight, TxIndex: 0, FromInput: 0, MaxInputs: 0}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionInputsGas(0, testBlockBytes, 0),
			ReadOnly:          true,
			ExpectedRes:       mustPack(t)(PackGetTransactionInputsOutput(GetTransactionInputsOutput{Inputs: []Input{}, InputCount: 2})),
		},
		{
			Name:              "page starting at the input count is empty",
			InputFn:           call(GetTransactionInputsInput{Height: blockHeight, TxIndex: 0, FromInput: 2, MaxInputs: 5}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionInputsGas(5, testBlockBytes, 0),
			ReadOnly:          true,
			ExpectedRes:       mustPack(t)(PackGetTransactionInputsOutput(GetTransactionInputsOutput{Inputs: []Input{}, InputCount: 2})),
		},
		{
			Name:              "ownerless output resolves with no addresses",
			InputFn:           call(GetTransactionInputsInput{Height: blockHeight, TxIndex: 2, FromInput: 0, MaxInputs: 8}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowContextWithReader(t, newOwnerlessReader()),
			SuppliedGas:       GetTransactionInputsGas(8, decodedBytes(2, 0), 0),
			ReadOnly:          true,
			ExpectedRes: mustPack(t)(PackGetTransactionInputsOutput(GetTransactionInputsOutput{
				Inputs:     []Input{toInput(ownerlessInput)},
				InputCount: 1,
			})),
		},
		{
			Name:              "unsupported tx reverts",
			InputFn:           call(GetTransactionInputsInput{Height: blockHeight, TxIndex: 1, FromInput: 0, MaxInputs: 1}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionInputsGas(1, testBlockBytes, 0),
			ReadOnly:          true,
			ExpectedRes:       revertReason(reasonUnsupportedTx),
			ExpectedErr:       vm.ErrExecutionReverted,
		},
		{
			Name:              "tx index out of range reverts",
			InputFn:           call(GetTransactionInputsInput{Height: blockHeight, TxIndex: 6, FromInput: 0, MaxInputs: 1}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionInputsGas(1, testBlockBytes, 0),
			ReadOnly:          true,
			ExpectedRes:       revertReason(reasonTxIndexOutOfRange),
			ExpectedErr:       vm.ErrExecutionReverted,
		},
		{
			Name:              "input index out of range reverts",
			InputFn:           call(GetTransactionInputsInput{Height: blockHeight, TxIndex: 0, FromInput: 3, MaxInputs: 1}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionInputsGas(1, testBlockBytes, 0),
			ReadOnly:          true,
			ExpectedRes:       revertReason(reasonInputIndexOutOfRange),
			ExpectedErr:       vm.ErrExecutionReverted,
		},
		{
			Name:              "imported input is unresolved and charges no producing transaction",
			InputFn:           call(GetTransactionInputsInput{Height: transferHeight, TxIndex: 0, FromInput: 0, MaxInputs: 64}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionInputsGas(64, transferDecodedBytes(0, 0), 1),
			ReadOnly:          true,
			ExpectedRes: mustPack(t)(PackGetTransactionInputsOutput(GetTransactionInputsOutput{
				Inputs:     []Input{toInput(transferInputs[0][0]), toInput(transferInputs[0][1])},
				InputCount: 2,
			})),
		},
	}
	precompiletest.RunPrecompileTests(t, Module, tests)
}

func TestGetStakeOwner(t *testing.T) {
	require := require.New(t)
	reader := newFakeReader()
	snowCtx := snowContextWithReader(t, reader)
	hrp := constants.GetHRP(snowCtx.NetworkID)

	// The validator's first canonical input is owned by A, B, C with A and C
	// signing; the indexer picks the lexically smallest bech32 string of the
	// signers.
	bech := func(addr ids.ShortID) string {
		str, err := address.FormatBech32(hrp, addr.Bytes())
		require.NoError(err)
		return str
	}
	canonicalOwner, canonicalKey := addrA, keyA
	if bech(addrC) < bech(addrA) {
		canonicalOwner, canonicalKey = addrC, keyC
	}

	// Variants of the delegator's single-owner input (tx 2) and of the
	// validator's multisig input (tx 0) whose signatures do not recover to
	// the attributed owner.
	singleOwner := testInputs[2][0]
	wrongKey, unsigned, malformed := singleOwner, singleOwner, singleOwner
	wrongKey.Signatures = sigs(delegatorTxID[:], keyE)
	unsigned.Signatures = nil
	malformed.Signatures = make([][pchain.SignatureLen]byte, 1)
	swapped := testInputs[0][0]
	swapped.Signatures = sigs(validatorTxID[:], keyC, keyA)

	call := func(in GetStakeOwnerInput) func(testing.TB) []byte {
		return func(t testing.TB) []byte { return mustPack(t)(PackGetStakeOwner(in)) }
	}
	env := &fakeEnv{}

	tests := []precompiletest.PrecompileTest{
		{
			Name:              "picks the smallest bech32 signer of the first canonical input",
			InputFn:           call(GetStakeOwnerInput{Height: blockHeight, TxIndex: 0}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetStakeOwnerGas(decodedBytes(0, 0), 2),
			ReadOnly:          true,
			ExpectedRes:       mustPack(t)(PackGetStakeOwnerOutput(GetStakeOwnerOutput{PChainOwner: canonicalOwner, CChainOwner: canonicalKey.EthAddress(), Resolved: true})),
		},
		{
			Name:              "single owner",
			InputFn:           call(GetStakeOwnerInput{Height: blockHeight, TxIndex: 2}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetStakeOwnerGas(decodedBytes(2, 0), 1),
			ReadOnly:          true,
			ExpectedRes:       mustPack(t)(PackGetStakeOwnerOutput(GetStakeOwnerOutput{PChainOwner: addrB, CChainOwner: keyB.EthAddress(), Resolved: true})),
		},
		{
			Name:              "genesis input has no owner",
			InputFn:           call(GetStakeOwnerInput{Height: blockHeight, TxIndex: 3}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetStakeOwnerGas(decodedBytes(3, 0), 0),
			ReadOnly:          true,
			ExpectedRes:       mustPack(t)(PackGetStakeOwnerOutput(GetStakeOwnerOutput{})),
		},
		{
			Name:              "no inputs has no owner",
			InputFn:           call(GetStakeOwnerInput{Height: blockHeight, TxIndex: 4}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetStakeOwnerGas(testBlockBytes, 0),
			ReadOnly:          true,
			ExpectedRes:       mustPack(t)(PackGetStakeOwnerOutput(GetStakeOwnerOutput{})),
		},
		{
			Name:              "unresolved input has no owner",
			InputFn:           call(GetStakeOwnerInput{Height: blockHeight, TxIndex: 5}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetStakeOwnerGas(decodedBytes(5, 0), 0),
			ReadOnly:          true,
			ExpectedRes:       mustPack(t)(PackGetStakeOwnerOutput(GetStakeOwnerOutput{})),
		},
		{
			Name:              "ownerless output has no owner",
			InputFn:           call(GetStakeOwnerInput{Height: blockHeight, TxIndex: 2}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowContextWithReader(t, newOwnerlessReader()),
			SuppliedGas:       GetStakeOwnerGas(decodedBytes(2, 0), 0),
			ReadOnly:          true,
			ExpectedRes:       mustPack(t)(PackGetStakeOwnerOutput(GetStakeOwnerOutput{})),
		},
		{
			Name:              "unsupported tx reverts",
			InputFn:           call(GetStakeOwnerInput{Height: blockHeight, TxIndex: 1}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetStakeOwnerGas(testBlockBytes, 0),
			ReadOnly:          true,
			ExpectedRes:       revertReason(reasonUnsupportedTx),
			ExpectedErr:       vm.ErrExecutionReverted,
		},
		{
			Name:              "reader failure invalidates execution",
			InputFn:           call(GetStakeOwnerInput{Height: blockHeight, TxIndex: 0}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowContextWithReader(t, &fakeReader{err: errDatabaseCorrupted}),
			PrecompileEnv:     env,
			SuppliedGas:       GetStakeOwnerGas(0, 0),
			ReadOnly:          true,
			ExpectedErr:       errDatabaseCorrupted,
			AfterHook: func(testing.TB, contract.StateDB) {
				require.ErrorIs(env.invalidated, errDatabaseCorrupted)
			},
		},
		{
			Name:              "import: attributes the fee-paying P-chain input, first in canonical order",
			InputFn:           call(GetStakeOwnerInput{Height: transferHeight, TxIndex: 0}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetStakeOwnerGas(transferDecodedBytes(0, 0), 1),
			ReadOnly:          true,
			ExpectedRes:       mustPack(t)(PackGetStakeOwnerOutput(GetStakeOwnerOutput{PChainOwner: addrA, CChainOwner: keyA.EthAddress(), Resolved: true})),
		},
		{
			Name:              "transfer attributes its sender, who signed in the Ethereum scheme",
			InputFn:           call(GetStakeOwnerInput{Height: transferHeight, TxIndex: 2}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetStakeOwnerGas(transferDecodedBytes(2, 0), 1),
			ReadOnly:          true,
			ExpectedRes:       mustPack(t)(PackGetStakeOwnerOutput(GetStakeOwnerOutput{PChainOwner: addrB, CChainOwner: keyB.EthAddress(), Resolved: true})),
		},
		{
			Name:              "reward-validator transaction has no inputs",
			InputFn:           call(GetStakeOwnerInput{Height: transferHeight, TxIndex: 3}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetStakeOwnerGas(transferBlockBytes, 0),
			ReadOnly:          true,
			ExpectedRes:       mustPack(t)(PackGetStakeOwnerOutput(GetStakeOwnerOutput{})),
		},
		// Signatures that do not recover to the attributed owner leave the
		// stake unattributed (never for accepted transactions: consensus
		// verified them) and still pay for the recovery; none of them
		// reverts or invalidates execution.
		{
			Name:              "wrong-key signature leaves the stake unattributed",
			InputFn:           call(GetStakeOwnerInput{Height: blockHeight, TxIndex: 2}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowContextWithReader(t, newReaderWithInput(2, wrongKey)),
			SuppliedGas:       GetStakeOwnerGas(decodedBytes(2, 0), 1),
			ReadOnly:          true,
			ExpectedRes:       mustPack(t)(PackGetStakeOwnerOutput(GetStakeOwnerOutput{})),
		},
		{
			Name:              "missing signature leaves the stake unattributed",
			InputFn:           call(GetStakeOwnerInput{Height: blockHeight, TxIndex: 2}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowContextWithReader(t, newReaderWithInput(2, unsigned)),
			SuppliedGas:       GetStakeOwnerGas(decodedBytes(2, 0), 1),
			ReadOnly:          true,
			ExpectedRes:       mustPack(t)(PackGetStakeOwnerOutput(GetStakeOwnerOutput{})),
		},
		{
			Name:              "malformed signature leaves the stake unattributed",
			InputFn:           call(GetStakeOwnerInput{Height: blockHeight, TxIndex: 2}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowContextWithReader(t, newReaderWithInput(2, malformed)),
			SuppliedGas:       GetStakeOwnerGas(decodedBytes(2, 0), 1),
			ReadOnly:          true,
			ExpectedRes:       mustPack(t)(PackGetStakeOwnerOutput(GetStakeOwnerOutput{})),
		},
		{
			// The owner's signature is the one at the owner's slot of the
			// signature indices, not the first: swapping the two signatures
			// of the multisig input attributes neither owner.
			Name:              "swapped multisig signatures leave the stake unattributed",
			InputFn:           call(GetStakeOwnerInput{Height: blockHeight, TxIndex: 0}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowContextWithReader(t, newReaderWithInput(0, swapped)),
			SuppliedGas:       GetStakeOwnerGas(decodedBytes(0, 0), 2),
			ReadOnly:          true,
			ExpectedRes:       mustPack(t)(PackGetStakeOwnerOutput(GetStakeOwnerOutput{})),
		},
		{
			Name:              "insufficient gas for signature recovery",
			InputFn:           call(GetStakeOwnerInput{Height: blockHeight, TxIndex: 2}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetStakeOwnerGas(decodedBytes(2, 0), 1) - 1,
			ReadOnly:          true,
			ExpectedErr:       vm.ErrOutOfGas,
		},
	}
	precompiletest.RunPrecompileTests(t, Module, tests)
}

func TestGetTransactionOutputs(t *testing.T) {
	reader := newFakeReader()
	snowCtx := snowContextWithReader(t, reader)
	env := &fakeEnv{}

	call := func(in GetTransactionOutputsInput) func(testing.TB) []byte {
		return func(t testing.TB) []byte { return mustPack(t)(PackGetTransactionOutputs(in)) }
	}

	tests := []precompiletest.PrecompileTest{
		{
			Name:              "staker: change output then stake output",
			InputFn:           call(GetTransactionOutputsInput{Height: blockHeight, TxIndex: 0, FromOutput: 0, MaxOutputs: 64}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionOutputsGas(64, testBlockBytes, 2),
			ReadOnly:          true,
			ExpectedRes: mustPack(t)(PackGetTransactionOutputsOutput(GetTransactionOutputsOutput{
				Outputs:     []Output{toOutput(testOutputs[0][0]), toOutput(testOutputs[0][1])},
				OutputCount: 2,
			})),
		},
		{
			Name:              "export: change output then exported output",
			InputFn:           call(GetTransactionOutputsInput{Height: transferHeight, TxIndex: 1, FromOutput: 0, MaxOutputs: 64}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionOutputsGas(64, transferBlockBytes, 2),
			ReadOnly:          true,
			ExpectedRes: mustPack(t)(PackGetTransactionOutputsOutput(GetTransactionOutputsOutput{
				Outputs:     []Output{toOutput(transferOutputs[1][0]), toOutput(transferOutputs[1][1])},
				OutputCount: 2,
			})),
		},
		{
			Name:              "paging returns the locked change output with both owners",
			InputFn:           call(GetTransactionOutputsInput{Height: transferHeight, TxIndex: 2, FromOutput: 1, MaxOutputs: 1}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionOutputsGas(1, transferBlockBytes, 2),
			ReadOnly:          true,
			ExpectedRes: mustPack(t)(PackGetTransactionOutputsOutput(GetTransactionOutputsOutput{
				Outputs:     []Output{toOutput(transferOutputs[2][1])},
				OutputCount: 2,
			})),
		},
		{
			Name:              "empty page",
			InputFn:           call(GetTransactionOutputsInput{Height: blockHeight, TxIndex: 0, FromOutput: 2, MaxOutputs: 5}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionOutputsGas(5, testBlockBytes, 0),
			ReadOnly:          true,
			ExpectedRes:       mustPack(t)(PackGetTransactionOutputsOutput(GetTransactionOutputsOutput{Outputs: []Output{}, OutputCount: 2})),
		},
		{
			Name:              "reward-validator transaction has no outputs",
			InputFn:           call(GetTransactionOutputsInput{Height: transferHeight, TxIndex: 3, FromOutput: 0, MaxOutputs: 8}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionOutputsGas(8, transferBlockBytes, 0),
			ReadOnly:          true,
			ExpectedRes:       mustPack(t)(PackGetTransactionOutputsOutput(GetTransactionOutputsOutput{Outputs: []Output{}, OutputCount: 0})),
		},
		{
			Name:              "output index out of range reverts",
			InputFn:           call(GetTransactionOutputsInput{Height: blockHeight, TxIndex: 0, FromOutput: 3, MaxOutputs: 1}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionOutputsGas(1, testBlockBytes, 0),
			ReadOnly:          true,
			ExpectedRes:       revertReason(reasonOutputIndexOutOfRange),
			ExpectedErr:       vm.ErrExecutionReverted,
		},
		{
			Name:              "unsupported tx reverts",
			InputFn:           call(GetTransactionOutputsInput{Height: blockHeight, TxIndex: 1, FromOutput: 0, MaxOutputs: 1}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionOutputsGas(1, testBlockBytes, 0),
			ReadOnly:          true,
			ExpectedRes:       revertReason(reasonUnsupportedTx),
			ExpectedErr:       vm.ErrExecutionReverted,
		},
		{
			Name:              "tx index out of range reverts",
			InputFn:           call(GetTransactionOutputsInput{Height: blockHeight, TxIndex: 6, FromOutput: 0, MaxOutputs: 1}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionOutputsGas(1, testBlockBytes, 0),
			ReadOnly:          true,
			ExpectedRes:       revertReason(reasonTxIndexOutOfRange),
			ExpectedErr:       vm.ErrExecutionReverted,
		},
		{
			Name:              "maxOutputs above cap reverts",
			InputFn:           call(GetTransactionOutputsInput{Height: blockHeight, TxIndex: 0, FromOutput: 0, MaxOutputs: MaxOutputsPerCall + 1}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       BlockReadGas,
			ReadOnly:          true,
			ExpectedRes:       revertReason(reasonPageSizeExceedsCap),
			ExpectedErr:       vm.ErrExecutionReverted,
		},
		{
			Name:              "height above bound reverts after charging the page",
			InputFn:           call(GetTransactionOutputsInput{Height: bound + 1, TxIndex: 0, FromOutput: 0, MaxOutputs: 4}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionOutputsGas(4, 0, 0),
			ReadOnly:          true,
			ExpectedRes:       revertReason(reasonHeightAboveBound),
			ExpectedErr:       vm.ErrExecutionReverted,
		},
		{
			Name:              "insufficient gas for the page",
			InputFn:           call(GetTransactionOutputsInput{Height: blockHeight, TxIndex: 0, FromOutput: 0, MaxOutputs: 4}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionOutputsGas(4, 0, 0) - 1,
			ReadOnly:          true,
			ExpectedErr:       vm.ErrOutOfGas,
		},
		{
			Name:              "insufficient gas for the addresses",
			InputFn:           call(GetTransactionOutputsInput{Height: blockHeight, TxIndex: 0, FromOutput: 0, MaxOutputs: 64}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       GetTransactionOutputsGas(64, testBlockBytes, 2) - 1,
			ReadOnly:          true,
			ExpectedErr:       vm.ErrOutOfGas,
		},
		{
			Name: "malformed input reverts",
			InputFn: func(t testing.TB) []byte {
				return mustPack(t)(PackGetTransactionOutputs(GetTransactionOutputsInput{Height: 1}))[:6]
			},
			SetupBlockContext: withBound(bound),
			SnowContext:       snowCtx,
			SuppliedGas:       BlockReadGas,
			ReadOnly:          true,
			ExpectedRes:       revertReason(reasonInvalidInput),
			ExpectedErr:       vm.ErrExecutionReverted,
		},
		{
			Name:              "reader failure invalidates execution",
			InputFn:           call(GetTransactionOutputsInput{Height: blockHeight, TxIndex: 0, FromOutput: 0, MaxOutputs: 1}),
			SetupBlockContext: withBound(bound),
			SnowContext:       snowContextWithReader(t, &fakeReader{err: errDatabaseCorrupted}),
			PrecompileEnv:     env,
			SuppliedGas:       GetTransactionOutputsGas(1, 0, 0),
			ReadOnly:          true,
			ExpectedErr:       errDatabaseCorrupted,
			AfterHook: func(t testing.TB, _ contract.StateDB) {
				require.ErrorIs(t, env.invalidated, errDatabaseCorrupted)
			},
		},
	}
	precompiletest.RunPrecompileTests(t, Module, tests)
}

// errDatabaseCorrupted stands in for a node-local reader failure.
var errDatabaseCorrupted = errors.New("database corrupted")

func TestStakeOwnerCandidates(t *testing.T) {
	require := require.New(t)

	in := pchain.Input{Addresses: []ids.ShortID{addrA, addrB, addrC}, SigIndices: []uint32{1}}
	require.Equal([]ids.ShortID{addrB}, stakeOwnerCandidates(in))

	// Without signature indices every owner is a candidate.
	in.SigIndices = nil
	require.Equal([]ids.ShortID{addrA, addrB, addrC}, stakeOwnerCandidates(in))

	// Out-of-range indices fall back to every owner.
	in.SigIndices = []uint32{9}
	require.Equal([]ids.ShortID{addrA, addrB, addrC}, stakeOwnerCandidates(in))

	// A single owner needs no signatures to be attributed.
	require.Equal([]ids.ShortID{addrD}, stakeOwnerCandidates(pchain.Input{Addresses: []ids.ShortID{addrD}}))
}

func TestSignatureOf(t *testing.T) {
	require := require.New(t)

	sigA, sigC := sigs(validatorTxID[:], keyA)[0], sigs(validatorTxID[:], keyC)[0]
	in := pchain.Input{
		Addresses:  []ids.ShortID{addrA, addrB, addrC},
		SigIndices: []uint32{0, 2},
		Signatures: [][pchain.SignatureLen]byte{sigA, sigC},
	}
	got, ok := signatureOf(in, addrA)
	require.True(ok)
	require.Equal(sigA[:], got)
	got, ok = signatureOf(in, addrC)
	require.True(ok)
	require.Equal(sigC[:], got)

	// An owner that did not sign has no signature.
	_, ok = signatureOf(in, addrB)
	require.False(ok)

	// Fewer signatures than signature indices: the rest have none.
	in.Signatures = in.Signatures[:1]
	_, ok = signatureOf(in, addrC)
	require.False(ok)
	_, ok = signatureOf(pchain.Input{Addresses: []ids.ShortID{addrA}, SigIndices: []uint32{0}}, addrA)
	require.False(ok)

	// An out-of-range index is skipped; the position still counts.
	in = pchain.Input{
		Addresses:  []ids.ShortID{addrA},
		SigIndices: []uint32{5, 0},
		Signatures: [][pchain.SignatureLen]byte{sigC, sigA},
	}
	got, ok = signatureOf(in, addrA)
	require.True(ok)
	require.Equal(sigA[:], got)
}

func TestCChainOwner(t *testing.T) {
	require := require.New(t)

	unsignedBytes := []byte("some unsigned transaction")
	owner := func(signatures [][pchain.SignatureLen]byte) pchain.Input {
		return pchain.Input{Addresses: []ids.ShortID{addrA}, SigIndices: []uint32{0}, Signatures: signatures}
	}

	// Both signing schemes recover the same key and hence the same EVM
	// address.
	got, ok := cChainOwner(unsignedBytes, owner(sigs(unsignedBytes, keyA)), addrA)
	require.True(ok)
	require.Equal(keyA.EthAddress(), got)
	got, ok = cChainOwner(unsignedBytes, owner(ethSigs(unsignedBytes, keyA)), addrA)
	require.True(ok)
	require.Equal(keyA.EthAddress(), got)

	// A signature over other bytes, by another key, or none at all does not
	// recover to the owner.
	_, ok = cChainOwner([]byte("another transaction"), owner(sigs(unsignedBytes, keyA)), addrA)
	require.False(ok)
	_, ok = cChainOwner(unsignedBytes, owner(sigs(unsignedBytes, keyE)), addrA)
	require.False(ok)
	_, ok = cChainOwner(unsignedBytes, owner(nil), addrA)
	require.False(ok)
}

// TestSmallestBech32PrefersStringOrder mirrors the indexer's
// TestOwnerOrderIsByAddressBytesNotBech32: byte order and bech32 string order
// disagree, and the attribution follows the string order.
func TestSmallestBech32PrefersStringOrder(t *testing.T) {
	require := require.New(t)
	const hrp = "costwo"

	var first, second ids.ShortID
	found := false
	for i := 0; i < 10_000 && !found; i++ {
		a, b := ids.GenerateTestShortID(), ids.GenerateTestShortID()
		strA, err := address.FormatBech32(hrp, a.Bytes())
		require.NoError(err)
		strB, err := address.FormatBech32(hrp, b.Bytes())
		require.NoError(err)
		switch {
		case bytes.Compare(a.Bytes(), b.Bytes()) < 0 && strA > strB:
			first, second, found = a, b, true // first is smaller by bytes, larger by string
		case bytes.Compare(b.Bytes(), a.Bytes()) < 0 && strB > strA:
			first, second, found = b, a, true
		}
	}
	require.True(found, "could not find a pair whose byte and bech32 orderings disagree")

	owner, err := smallestBech32(hrp, []ids.ShortID{first, second})
	require.NoError(err)
	require.Equal(second, owner)
}
