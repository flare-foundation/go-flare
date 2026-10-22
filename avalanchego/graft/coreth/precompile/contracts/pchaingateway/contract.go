// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

// Package pchaingateway implements the P-chain gateway, a Flare-only stateful
// precompile that lets contracts read accepted P-chain blocks and transactions
// (primary-network staking transactions, imports, exports and plain transfers)
// during EVM execution (pull model).
//
// Determinism: every read is bounded by the P-chain height the SAE block
// builder committed into the block header (HeaderExtra.PChainHeight, the
// ProposerVM epoch height, verified by block rebuild), and the canonical
// executor waits for the local P-chain to reach that height before executing
// the block. See README.md.
package pchaingateway

import (
	_ "embed"
	"fmt"
	"math/big"

	"github.com/ava-labs/libevm/common"

	"github.com/ava-labs/avalanchego/graft/coreth/precompile/contract"
	"github.com/ava-labs/avalanchego/snow/pchain"
)

var (
	// PChainGatewayRawABI contains the raw ABI of the gateway contract.
	//go:embed IPChainGateway.abi
	PChainGatewayRawABI string

	PChainGatewayABI = contract.ParseABI(PChainGatewayRawABI)

	PChainGatewayPrecompile = createPChainGatewayPrecompile()
)

// Input mirrors IPChainGateway.Input.
type Input struct {
	UtxoTxId          [32]byte
	UtxoOutputIndex   uint32
	AssetId           [32]byte
	Amount            uint64
	Imported          bool
	Resolved          bool
	Locktime          uint64
	StakeableLocktime uint64
	Threshold         uint32
	Addresses         [][20]byte
	SigIndices        []uint32
}

// Output mirrors IPChainGateway.Output.
type Output struct {
	Kind              uint8
	AssetId           [32]byte
	Amount            uint64
	Locktime          uint64
	StakeableLocktime uint64
	Threshold         uint32
	Addresses         [][20]byte
}

// Tx mirrors IPChainGateway.Tx.
type Tx struct {
	TxType                   uint8
	Index                    uint32
	TxId                     [32]byte
	NodeId                   [20]byte
	StartTime                uint64
	EndTime                  uint64
	Period                   uint64
	Weight                   uint64
	DelegationShares         uint32
	AutoCompoundRewardShares uint32
	StakerTxId               [32]byte
	RemoteChainId            [32]byte
	InputCount               uint32
	OutputCount              uint32
	Inputs                   []Input
	Outputs                  []Output
}

// GetBlockOutput is the output of getBlock.
type GetBlockOutput struct {
	BlockId   [32]byte
	ParentId  [32]byte
	Timestamp uint64
	Kind      uint8
	TxCount   uint32
}

// GetTransactionsInput is the input of getTransactions.
type GetTransactionsInput struct {
	Height          uint64
	FromIndex       uint32
	MaxTxs          uint32
	TypeFilter      *big.Int
	MaxInputsPerTx  uint32
	MaxOutputsPerTx uint32
}

// GetTransactionsOutput is the output of getTransactions.
type GetTransactionsOutput struct {
	Txs       []Tx
	NextIndex uint32
}

// GetTransactionInputsInput is the input of getTransactionInputs.
type GetTransactionInputsInput struct {
	Height    uint64
	TxIndex   uint32
	FromInput uint32
	MaxInputs uint32
}

// GetTransactionInputsOutput is the output of getTransactionInputs.
type GetTransactionInputsOutput struct {
	Inputs     []Input
	InputCount uint32
}

// GetTransactionOutputsInput is the input of getTransactionOutputs.
type GetTransactionOutputsInput struct {
	Height     uint64
	TxIndex    uint32
	FromOutput uint32
	MaxOutputs uint32
}

// GetTransactionOutputsOutput is the output of getTransactionOutputs.
type GetTransactionOutputsOutput struct {
	Outputs     []Output
	OutputCount uint32
}

// GetStakeOwnerInput is the input of getStakeOwner.
type GetStakeOwnerInput struct {
	Height  uint64
	TxIndex uint32
}

// GetStakeOwnerOutput is the output of getStakeOwner: the attributed owner's
// P-chain address and the C-chain (EVM) address of the same key.
type GetStakeOwnerOutput struct {
	PChainOwner [20]byte
	CChainOwner common.Address
	Resolved    bool
}

// PackGetPChainHeight packs the getPChainHeight call (selector included).
func PackGetPChainHeight() ([]byte, error) {
	return PChainGatewayABI.Pack("getPChainHeight")
}

// PackGetPChainHeightOutput packs the getPChainHeight output.
func PackGetPChainHeightOutput(height uint64) ([]byte, error) {
	return PChainGatewayABI.PackOutput("getPChainHeight", height)
}

// UnpackGetPChainHeightOutput unpacks the getPChainHeight output.
func UnpackGetPChainHeightOutput(output []byte) (uint64, error) {
	var height uint64
	err := PChainGatewayABI.UnpackIntoInterface(&height, "getPChainHeight", output)
	return height, err
}

// PackGetNativeAssetId packs the getNativeAssetId call (selector included).
func PackGetNativeAssetId() ([]byte, error) {
	return PChainGatewayABI.Pack("getNativeAssetId")
}

// PackGetNativeAssetIdOutput packs the getNativeAssetId output.
func PackGetNativeAssetIdOutput(assetID [32]byte) ([]byte, error) {
	return PChainGatewayABI.PackOutput("getNativeAssetId", assetID)
}

// UnpackGetNativeAssetIdOutput unpacks the getNativeAssetId output.
func UnpackGetNativeAssetIdOutput(output []byte) ([32]byte, error) {
	var assetID [32]byte
	err := PChainGatewayABI.UnpackIntoInterface(&assetID, "getNativeAssetId", output)
	return assetID, err
}

// PackGetBlock packs the getBlock call (selector included).
func PackGetBlock(height uint64) ([]byte, error) {
	return PChainGatewayABI.Pack("getBlock", height)
}

// UnpackGetBlockInput unpacks the getBlock arguments (selector omitted).
func UnpackGetBlockInput(input []byte) (uint64, error) {
	var height uint64
	err := PChainGatewayABI.UnpackInputIntoInterface(&height, "getBlock", input)
	return height, err
}

// PackGetBlockOutput packs the getBlock output.
func PackGetBlockOutput(out GetBlockOutput) ([]byte, error) {
	return PChainGatewayABI.PackOutput("getBlock", out.BlockId, out.ParentId, out.Timestamp, out.Kind, out.TxCount)
}

// UnpackGetBlockOutput unpacks the getBlock output.
func UnpackGetBlockOutput(output []byte) (GetBlockOutput, error) {
	var out GetBlockOutput
	err := PChainGatewayABI.UnpackIntoInterface(&out, "getBlock", output)
	return out, err
}

// PackGetTransactions packs the getTransactions call (selector included).
func PackGetTransactions(in GetTransactionsInput) ([]byte, error) {
	return PChainGatewayABI.Pack("getTransactions", in.Height, in.FromIndex, in.MaxTxs, in.TypeFilter, in.MaxInputsPerTx, in.MaxOutputsPerTx)
}

// UnpackGetTransactionsInput unpacks the getTransactions arguments (selector omitted).
func UnpackGetTransactionsInput(input []byte) (GetTransactionsInput, error) {
	var in GetTransactionsInput
	err := PChainGatewayABI.UnpackInputIntoInterface(&in, "getTransactions", input)
	return in, err
}

// PackGetTransactionsOutput packs the getTransactions output.
func PackGetTransactionsOutput(out GetTransactionsOutput) ([]byte, error) {
	return PChainGatewayABI.PackOutput("getTransactions", out.Txs, out.NextIndex)
}

// UnpackGetTransactionsOutput unpacks the getTransactions output.
func UnpackGetTransactionsOutput(output []byte) (GetTransactionsOutput, error) {
	var out GetTransactionsOutput
	err := PChainGatewayABI.UnpackIntoInterface(&out, "getTransactions", output)
	return out, err
}

// PackGetTransactionInputs packs the getTransactionInputs call (selector included).
func PackGetTransactionInputs(in GetTransactionInputsInput) ([]byte, error) {
	return PChainGatewayABI.Pack("getTransactionInputs", in.Height, in.TxIndex, in.FromInput, in.MaxInputs)
}

// UnpackGetTransactionInputsInput unpacks the getTransactionInputs arguments (selector omitted).
func UnpackGetTransactionInputsInput(input []byte) (GetTransactionInputsInput, error) {
	var in GetTransactionInputsInput
	err := PChainGatewayABI.UnpackInputIntoInterface(&in, "getTransactionInputs", input)
	return in, err
}

// PackGetTransactionInputsOutput packs the getTransactionInputs output.
func PackGetTransactionInputsOutput(out GetTransactionInputsOutput) ([]byte, error) {
	return PChainGatewayABI.PackOutput("getTransactionInputs", out.Inputs, out.InputCount)
}

// UnpackGetTransactionInputsOutput unpacks the getTransactionInputs output.
func UnpackGetTransactionInputsOutput(output []byte) (GetTransactionInputsOutput, error) {
	var out GetTransactionInputsOutput
	err := PChainGatewayABI.UnpackIntoInterface(&out, "getTransactionInputs", output)
	return out, err
}

// PackGetTransactionOutputs packs the getTransactionOutputs call (selector included).
func PackGetTransactionOutputs(in GetTransactionOutputsInput) ([]byte, error) {
	return PChainGatewayABI.Pack("getTransactionOutputs", in.Height, in.TxIndex, in.FromOutput, in.MaxOutputs)
}

// UnpackGetTransactionOutputsInput unpacks the getTransactionOutputs arguments (selector omitted).
func UnpackGetTransactionOutputsInput(input []byte) (GetTransactionOutputsInput, error) {
	var in GetTransactionOutputsInput
	err := PChainGatewayABI.UnpackInputIntoInterface(&in, "getTransactionOutputs", input)
	return in, err
}

// PackGetTransactionOutputsOutput packs the getTransactionOutputs output.
func PackGetTransactionOutputsOutput(out GetTransactionOutputsOutput) ([]byte, error) {
	return PChainGatewayABI.PackOutput("getTransactionOutputs", out.Outputs, out.OutputCount)
}

// UnpackGetTransactionOutputsOutput unpacks the getTransactionOutputs output.
func UnpackGetTransactionOutputsOutput(output []byte) (GetTransactionOutputsOutput, error) {
	var out GetTransactionOutputsOutput
	err := PChainGatewayABI.UnpackIntoInterface(&out, "getTransactionOutputs", output)
	return out, err
}

// PackGetStakeOwner packs the getStakeOwner call (selector included).
func PackGetStakeOwner(in GetStakeOwnerInput) ([]byte, error) {
	return PChainGatewayABI.Pack("getStakeOwner", in.Height, in.TxIndex)
}

// UnpackGetStakeOwnerInput unpacks the getStakeOwner arguments (selector omitted).
func UnpackGetStakeOwnerInput(input []byte) (GetStakeOwnerInput, error) {
	var in GetStakeOwnerInput
	err := PChainGatewayABI.UnpackInputIntoInterface(&in, "getStakeOwner", input)
	return in, err
}

// PackGetStakeOwnerOutput packs the getStakeOwner output.
func PackGetStakeOwnerOutput(out GetStakeOwnerOutput) ([]byte, error) {
	return PChainGatewayABI.PackOutput("getStakeOwner", out.PChainOwner, out.CChainOwner, out.Resolved)
}

// UnpackGetStakeOwnerOutput unpacks the getStakeOwner output.
func UnpackGetStakeOwnerOutput(output []byte) (GetStakeOwnerOutput, error) {
	var out GetStakeOwnerOutput
	err := PChainGatewayABI.UnpackIntoInterface(&out, "getStakeOwner", output)
	return out, err
}

// toInput converts a reader input to its ABI form.
func toInput(in pchain.Input) Input {
	out := Input{
		UtxoTxId:          in.UTXOTxID,
		UtxoOutputIndex:   in.UTXOOutputIndex,
		AssetId:           in.AssetID,
		Amount:            in.Amount,
		Imported:          in.Imported,
		Resolved:          in.Resolved,
		Locktime:          in.Locktime,
		StakeableLocktime: in.StakeableLocktime,
		Threshold:         in.Threshold,
		Addresses:         make([][20]byte, len(in.Addresses)),
		SigIndices:        make([]uint32, len(in.SigIndices)),
	}
	for i, addr := range in.Addresses {
		out.Addresses[i] = addr
	}
	copy(out.SigIndices, in.SigIndices)
	return out
}

// toOutput converts a reader output to its ABI form.
func toOutput(output pchain.Output) Output {
	out := Output{
		Kind:              uint8(output.Kind),
		AssetId:           output.AssetID,
		Amount:            output.Amount,
		Locktime:          output.Locktime,
		StakeableLocktime: output.StakeableLocktime,
		Threshold:         output.Threshold,
		Addresses:         make([][20]byte, len(output.Addresses)),
	}
	for i, addr := range output.Addresses {
		out.Addresses[i] = addr
	}
	return out
}

// toTx converts a reader transaction and its inline inputs and outputs to the
// ABI form.
func toTx(tx pchain.Tx, inputs []pchain.Input, outputs []pchain.Output) Tx {
	out := Tx{
		TxType:                   uint8(tx.Type),
		Index:                    tx.Index,
		TxId:                     tx.ID,
		NodeId:                   tx.NodeID,
		StartTime:                tx.StartTime,
		EndTime:                  tx.EndTime,
		Period:                   tx.Period,
		Weight:                   tx.Weight,
		DelegationShares:         tx.DelegationShares,
		AutoCompoundRewardShares: tx.AutoCompoundRewardShares,
		StakerTxId:               tx.StakerTxID,
		RemoteChainId:            tx.RemoteChainID,
		InputCount:               tx.InputCount,
		OutputCount:              tx.OutputCount,
		Inputs:                   make([]Input, len(inputs)),
		Outputs:                  make([]Output, len(outputs)),
	}
	for i, in := range inputs {
		out.Inputs[i] = toInput(in)
	}
	for i, output := range outputs {
		out.Outputs[i] = toOutput(output)
	}
	return out
}

// createPChainGatewayPrecompile returns the precompile with all functions
// registered by their ABI selectors.
func createPChainGatewayPrecompile() contract.StatefulPrecompiledContract {
	abiFunctionMap := map[string]contract.RunStatefulPrecompileFunc{
		"getPChainHeight":       getPChainHeight,
		"getNativeAssetId":      getNativeAssetId,
		"getBlock":              getBlock,
		"getTransactions":       getTransactions,
		"getTransactionInputs":  getTransactionInputs,
		"getTransactionOutputs": getTransactionOutputs,
		"getStakeOwner":         getStakeOwner,
	}

	functions := make([]*contract.StatefulPrecompileFunction, 0, len(abiFunctionMap))
	for name, function := range abiFunctionMap {
		method, ok := PChainGatewayABI.Methods[name]
		if !ok {
			panic(fmt.Errorf("given method (%s) does not exist in the ABI", name))
		}
		functions = append(functions, contract.NewStatefulPrecompileFunction(method.ID, function))
	}
	// Construct the contract with no fallback function.
	statefulContract, err := contract.NewStatefulPrecompileContract(nil, functions)
	if err != nil {
		panic(err)
	}
	return statefulContract
}
