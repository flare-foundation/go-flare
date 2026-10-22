// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package pchaingateway

import (
	"errors"
	"math/big"

	"github.com/ava-labs/libevm/common"
	"github.com/ava-labs/libevm/common/math"
	"github.com/ava-labs/libevm/core/vm"

	"github.com/ava-labs/avalanchego/graft/coreth/precompile/contract"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/pchain"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/formatting/address"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
)

// pchainReader returns the P-chain reader of the chain context.
func pchainReader(accessibleState contract.AccessibleState) (pchain.Reader, error) {
	snowCtx := accessibleState.GetSnowContext()
	if snowCtx == nil || snowCtx.PChainReader == nil {
		return nil, errPChainReaderUnavailable
	}
	return snowCtx.PChainReader, nil
}

// checkBound returns a revert reason unless height is at or below the P-chain
// height committed in the block header.
func checkBound(accessibleState contract.AccessibleState, height uint64) (string, bool) {
	bound, ok := accessibleState.GetBlockContext().PChainHeight()
	switch {
	case !ok:
		return reasonHeightNotCommitted, false
	case height > bound:
		return reasonHeightAboveBound, false
	default:
		return "", true
	}
}

// readerFailure turns a reader error into an out-of-gas result (the byte
// charge could not be paid), a revert (caller error) or an execution
// invalidation (node-local error).
func readerFailure(accessibleState contract.AccessibleState, remainingGas uint64, err error) ([]byte, uint64, error) {
	if errors.Is(err, vm.ErrOutOfGas) {
		return nil, 0, vm.ErrOutOfGas
	}
	if reason, callerError := readerError(err); callerError {
		return revert(remainingGas, reason)
	}
	return invalidate(accessibleState, err)
}

// byteCharger returns the reader's byte charger: it deducts ByteGas per byte
// of P-chain data from *remainingGas before the reader parses it, and fails
// with vm.ErrOutOfGas (leaving no gas) when the caller cannot pay.
func byteCharger(remainingGas *uint64) pchain.ByteCharger {
	return func(bytes uint64) error {
		cost, overflow := math.SafeMul(bytes, ByteGas)
		if overflow {
			*remainingGas = 0
			return vm.ErrOutOfGas
		}
		gas, err := contract.DeductGas(*remainingGas, cost)
		*remainingGas = gas
		return err
	}
}

// typeSelected reports whether filter selects txType; a zero filter selects
// every supported type.
func typeSelected(filter *big.Int, txType pchain.TxType) bool {
	return filter == nil || filter.Sign() == 0 || filter.Bit(int(txType)) == 1
}

func countInputAddresses(inputs []pchain.Input) uint64 {
	var n uint64
	for _, in := range inputs {
		n += uint64(len(in.Addresses))
	}
	return n
}

func countOutputAddresses(outputs []pchain.Output) uint64 {
	var n uint64
	for _, out := range outputs {
		n += uint64(len(out.Addresses))
	}
	return n
}

// getPChainHeight returns the P-chain height committed in the block header.
//
//nolint:revive // General-purpose types lose the meaning of args if unused ones are removed
func getPChainHeight(accessibleState contract.AccessibleState, caller common.Address, addr common.Address, input []byte, suppliedGas uint64, readOnly bool) ([]byte, uint64, error) {
	remainingGas, err := contract.DeductGas(suppliedGas, GetPChainHeightGas)
	if err != nil {
		return nil, 0, err
	}
	height, ok := accessibleState.GetBlockContext().PChainHeight()
	if !ok {
		return revert(remainingGas, reasonHeightNotCommitted)
	}
	packed, err := PackGetPChainHeightOutput(height)
	if err != nil {
		return nil, remainingGas, err
	}
	return packed, remainingGas, nil
}

// getNativeAssetId returns the asset ID of the native token (the primary
// network's AVAX asset ID: FLR on Flare, SGB on Songbird), the value an
// input's or output's assetId must equal for its amount to be native funds.
// It is a genesis constant, identical on every node.
//
//nolint:revive // General-purpose types lose the meaning of args if unused ones are removed
func getNativeAssetId(accessibleState contract.AccessibleState, caller common.Address, addr common.Address, input []byte, suppliedGas uint64, readOnly bool) ([]byte, uint64, error) {
	remainingGas, err := contract.DeductGas(suppliedGas, GetNativeAssetIdGas)
	if err != nil {
		return nil, 0, err
	}
	snowCtx := accessibleState.GetSnowContext()
	if snowCtx == nil {
		return invalidate(accessibleState, errChainContextUnavailable)
	}
	packed, err := PackGetNativeAssetIdOutput(snowCtx.AVAXAssetID)
	if err != nil {
		return nil, remainingGas, err
	}
	return packed, remainingGas, nil
}

// getBlock returns the accepted P-chain block at the requested height.
//
//nolint:revive // General-purpose types lose the meaning of args if unused ones are removed
func getBlock(accessibleState contract.AccessibleState, caller common.Address, addr common.Address, input []byte, suppliedGas uint64, readOnly bool) ([]byte, uint64, error) {
	remainingGas, err := contract.DeductGas(suppliedGas, BlockReadGas)
	if err != nil {
		return nil, 0, err
	}
	height, err := UnpackGetBlockInput(input)
	if err != nil {
		return revert(remainingGas, reasonInvalidInput)
	}
	if reason, ok := checkBound(accessibleState, height); !ok {
		return revert(remainingGas, reason)
	}
	reader, err := pchainReader(accessibleState)
	if err != nil {
		return invalidate(accessibleState, err)
	}

	blkReader, err := reader.GetBlock(height, byteCharger(&remainingGas))
	if err != nil {
		return readerFailure(accessibleState, remainingGas, err)
	}
	blk := blkReader.Block()
	packed, err := PackGetBlockOutput(GetBlockOutput{
		BlockId:   blk.ID,
		ParentId:  blk.ParentID,
		Timestamp: blk.Timestamp,
		Kind:      uint8(blk.Kind),
		TxCount:   uint32(len(blk.Txs)),
	})
	if err != nil {
		return nil, remainingGas, err
	}
	return packed, remainingGas, nil
}

// getTransactions returns a page of decoded transactions.
//
//nolint:revive // General-purpose types lose the meaning of args if unused ones are removed
func getTransactions(accessibleState contract.AccessibleState, caller common.Address, addr common.Address, input []byte, suppliedGas uint64, readOnly bool) ([]byte, uint64, error) {
	remainingGas, err := contract.DeductGas(suppliedGas, BlockReadGas)
	if err != nil {
		return nil, 0, err
	}
	in, err := UnpackGetTransactionsInput(input)
	if err != nil {
		return revert(remainingGas, reasonInvalidInput)
	}
	if in.MaxTxs > MaxTxsPerCall || in.MaxInputsPerTx > MaxInlineInputsPerTx || in.MaxOutputsPerTx > MaxInlineOutputsPerTx {
		return revert(remainingGas, reasonPageSizeExceedsCap)
	}
	// Charge the requested page before any P-chain access (caps make the
	// multiplication safe).
	pageGas := uint64(in.MaxTxs)*TxGas +
		uint64(in.MaxTxs)*uint64(in.MaxInputsPerTx)*InputGas +
		uint64(in.MaxTxs)*uint64(in.MaxOutputsPerTx)*OutputGas
	if remainingGas, err = contract.DeductGas(remainingGas, pageGas); err != nil {
		return nil, 0, err
	}
	if reason, ok := checkBound(accessibleState, in.Height); !ok {
		return revert(remainingGas, reason)
	}
	reader, err := pchainReader(accessibleState)
	if err != nil {
		return invalidate(accessibleState, err)
	}

	// The block is decoded and paid for once; the per-transaction input and
	// output reads below work on that decoded block and charge only the
	// producing transactions the inputs resolve.
	charge := byteCharger(&remainingGas)
	blkReader, err := reader.GetBlock(in.Height, charge)
	if err != nil {
		return readerFailure(accessibleState, remainingGas, err)
	}
	blk := blkReader.Block()
	txCount := uint32(len(blk.Txs))
	if in.FromIndex > txCount {
		return revert(remainingGas, reasonTxIndexOutOfRange)
	}

	var (
		txs       = make([]Tx, 0, min(in.MaxTxs, txCount-in.FromIndex))
		addresses uint64
		next      = in.FromIndex
	)
	for ; next < txCount && uint32(len(txs)) < in.MaxTxs; next++ {
		tx := blk.Txs[next]
		if tx.Type == pchain.TxUnsupported || !typeSelected(in.TypeFilter, tx.Type) {
			continue
		}
		var inputs []pchain.Input
		if in.MaxInputsPerTx > 0 && tx.InputCount > 0 {
			inputs, _, err = blkReader.GetInputs(next, 0, in.MaxInputsPerTx, charge)
			if err != nil {
				return readerFailure(accessibleState, remainingGas, err)
			}
		}
		var outputs []pchain.Output
		if in.MaxOutputsPerTx > 0 && tx.OutputCount > 0 {
			outputs, _, err = blkReader.GetOutputs(next, 0, in.MaxOutputsPerTx)
			if err != nil {
				return readerFailure(accessibleState, remainingGas, err)
			}
		}
		addresses += countInputAddresses(inputs) + countOutputAddresses(outputs)
		txs = append(txs, toTx(tx, inputs, outputs))
	}
	if remainingGas, err = contract.DeductGas(remainingGas, addresses*AddressGas); err != nil {
		return nil, 0, err
	}

	packed, err := PackGetTransactionsOutput(GetTransactionsOutput{Txs: txs, NextIndex: next})
	if err != nil {
		return nil, remainingGas, err
	}
	return packed, remainingGas, nil
}

// getTransactionInputs returns a page of the inputs of one transaction.
//
//nolint:revive // General-purpose types lose the meaning of args if unused ones are removed
func getTransactionInputs(accessibleState contract.AccessibleState, caller common.Address, addr common.Address, input []byte, suppliedGas uint64, readOnly bool) ([]byte, uint64, error) {
	remainingGas, err := contract.DeductGas(suppliedGas, BlockReadGas)
	if err != nil {
		return nil, 0, err
	}
	in, err := UnpackGetTransactionInputsInput(input)
	if err != nil {
		return revert(remainingGas, reasonInvalidInput)
	}
	if in.MaxInputs > MaxInputsPerCall {
		return revert(remainingGas, reasonPageSizeExceedsCap)
	}
	if remainingGas, err = contract.DeductGas(remainingGas, uint64(in.MaxInputs)*InputGas); err != nil {
		return nil, 0, err
	}
	if reason, ok := checkBound(accessibleState, in.Height); !ok {
		return revert(remainingGas, reason)
	}
	reader, err := pchainReader(accessibleState)
	if err != nil {
		return invalidate(accessibleState, err)
	}

	charge := byteCharger(&remainingGas)
	blkReader, err := reader.GetBlock(in.Height, charge)
	if err != nil {
		return readerFailure(accessibleState, remainingGas, err)
	}
	inputs, total, err := blkReader.GetInputs(in.TxIndex, in.FromInput, in.MaxInputs, charge)
	if err != nil {
		return readerFailure(accessibleState, remainingGas, err)
	}
	if remainingGas, err = contract.DeductGas(remainingGas, countInputAddresses(inputs)*AddressGas); err != nil {
		return nil, 0, err
	}

	out := GetTransactionInputsOutput{
		Inputs:     make([]Input, len(inputs)),
		InputCount: total,
	}
	for i, input := range inputs {
		out.Inputs[i] = toInput(input)
	}
	packed, err := PackGetTransactionInputsOutput(out)
	if err != nil {
		return nil, remainingGas, err
	}
	return packed, remainingGas, nil
}

// getTransactionOutputs returns a page of the outputs of one transaction.
//
//nolint:revive // General-purpose types lose the meaning of args if unused ones are removed
func getTransactionOutputs(accessibleState contract.AccessibleState, caller common.Address, addr common.Address, input []byte, suppliedGas uint64, readOnly bool) ([]byte, uint64, error) {
	remainingGas, err := contract.DeductGas(suppliedGas, BlockReadGas)
	if err != nil {
		return nil, 0, err
	}
	in, err := UnpackGetTransactionOutputsInput(input)
	if err != nil {
		return revert(remainingGas, reasonInvalidInput)
	}
	if in.MaxOutputs > MaxOutputsPerCall {
		return revert(remainingGas, reasonPageSizeExceedsCap)
	}
	if remainingGas, err = contract.DeductGas(remainingGas, uint64(in.MaxOutputs)*OutputGas); err != nil {
		return nil, 0, err
	}
	if reason, ok := checkBound(accessibleState, in.Height); !ok {
		return revert(remainingGas, reason)
	}
	reader, err := pchainReader(accessibleState)
	if err != nil {
		return invalidate(accessibleState, err)
	}

	blkReader, err := reader.GetBlock(in.Height, byteCharger(&remainingGas))
	if err != nil {
		return readerFailure(accessibleState, remainingGas, err)
	}
	// Outputs live in the decoded block, which the call has already paid
	// for; only the returned addresses are charged on top of the page.
	outputs, total, err := blkReader.GetOutputs(in.TxIndex, in.FromOutput, in.MaxOutputs)
	if err != nil {
		return readerFailure(accessibleState, remainingGas, err)
	}
	if remainingGas, err = contract.DeductGas(remainingGas, countOutputAddresses(outputs)*AddressGas); err != nil {
		return nil, 0, err
	}

	out := GetTransactionOutputsOutput{
		Outputs:     make([]Output, len(outputs)),
		OutputCount: total,
	}
	for i, output := range outputs {
		out.Outputs[i] = toOutput(output)
	}
	packed, err := PackGetTransactionOutputsOutput(out)
	if err != nil {
		return nil, remainingGas, err
	}
	return packed, remainingGas, nil
}

// getStakeOwner attributes a transaction to one owner the way the Flare
// P-chain indexer does: the first input in canonical (serialized) order,
// restricted to the owners whose signatures authorized the spend, and among
// those the lexically smallest bech32 address string. It returns the owner's
// P-chain address together with the C-chain (EVM) address of the same key,
// recovered from the owner's signature on the transaction. Genesis and
// unresolved inputs have no owner.
//
//nolint:revive // General-purpose types lose the meaning of args if unused ones are removed
func getStakeOwner(accessibleState contract.AccessibleState, caller common.Address, addr common.Address, input []byte, suppliedGas uint64, readOnly bool) ([]byte, uint64, error) {
	remainingGas, err := contract.DeductGas(suppliedGas, BlockReadGas+InputGas)
	if err != nil {
		return nil, 0, err
	}
	in, err := UnpackGetStakeOwnerInput(input)
	if err != nil {
		return revert(remainingGas, reasonInvalidInput)
	}
	if reason, ok := checkBound(accessibleState, in.Height); !ok {
		return revert(remainingGas, reason)
	}
	reader, err := pchainReader(accessibleState)
	if err != nil {
		return invalidate(accessibleState, err)
	}

	charge := byteCharger(&remainingGas)
	blkReader, err := reader.GetBlock(in.Height, charge)
	if err != nil {
		return readerFailure(accessibleState, remainingGas, err)
	}
	inputs, _, err := blkReader.GetInputs(in.TxIndex, 0, 1, charge)
	if err != nil {
		return readerFailure(accessibleState, remainingGas, err)
	}

	// A located output is not necessarily attributable: an output with
	// threshold 0 has no owner addresses (it is spendable without
	// signatures), so it resolves for getTransactionInputs but yields no
	// stake owner here.
	var out GetStakeOwnerOutput
	if len(inputs) == 1 && inputs[0].Resolved && !inputs[0].Genesis {
		candidates := stakeOwnerCandidates(inputs[0])
		gas := uint64(len(candidates)) * AddressGas
		if len(candidates) > 0 {
			gas += SignatureRecoveryGas
		}
		if remainingGas, err = contract.DeductGas(remainingGas, gas); err != nil {
			return nil, 0, err
		}
		if len(candidates) > 0 {
			hrp := constants.GetHRP(accessibleState.GetSnowContext().NetworkID)
			owner, err := smallestBech32(hrp, candidates)
			if err != nil {
				return invalidate(accessibleState, err)
			}
			// Consensus verified the owner's signature (under either
			// signing scheme) before accepting the transaction, so it only
			// fails to recover for data the P-chain never accepts; that
			// leaves the stake unattributed, deterministically, rather than
			// reverting.
			unsignedBytes := unsignedBytesOf(blkReader.Block(), in.TxIndex)
			if cOwner, ok := cChainOwner(unsignedBytes, inputs[0], owner); ok {
				out = GetStakeOwnerOutput{PChainOwner: owner, CChainOwner: cOwner, Resolved: true}
			}
		}
	}

	packed, err := PackGetStakeOwnerOutput(out)
	if err != nil {
		return nil, remainingGas, err
	}
	return packed, remainingGas, nil
}

// stakeOwnerCandidates returns the owners of the consumed output that signed
// the input; all owners when the signature indices are absent or unusable
// (the indexer keeps all owners in that case).
func stakeOwnerCandidates(in pchain.Input) []ids.ShortID {
	if len(in.Addresses) <= 1 || len(in.SigIndices) == 0 {
		return in.Addresses
	}
	candidates := make([]ids.ShortID, 0, len(in.SigIndices))
	for _, index := range in.SigIndices {
		if uint64(index) < uint64(len(in.Addresses)) {
			candidates = append(candidates, in.Addresses[index])
		}
	}
	if len(candidates) == 0 {
		return in.Addresses
	}
	return candidates
}

// unsignedBytesOf returns the unsigned bytes of transaction txIndex of blk,
// nil when there is no such transaction.
func unsignedBytesOf(blk *pchain.Block, txIndex uint32) []byte {
	if uint64(txIndex) >= uint64(len(blk.Txs)) {
		return nil
	}
	return blk.Txs[txIndex].UnsignedBytes
}

// cChainOwner recovers the C-chain (EVM) address of the key that signed the
// input for owner: the owner's signature (see signatureOf) over the
// transaction's unsigned bytes, under either signing scheme the P-chain
// accepts (vms/secp256k1fx.SigningHashes). ok is false when the owner has no
// signature on the input or its signature does not recover to it.
func cChainOwner(unsignedBytes []byte, in pchain.Input, owner ids.ShortID) (common.Address, bool) {
	sig, ok := signatureOf(in, owner)
	if !ok {
		return common.Address{}, false
	}
	pk, err := secp256k1fx.NewSigningHashes(unsignedBytes).RecoverSigner(nil, sig, owner)
	if err != nil {
		return common.Address{}, false
	}
	return pk.EthAddress(), true
}

// signatureOf returns the signature of owner on the input: the signature at
// the position of the owner's slot in the input's signature indices.
func signatureOf(in pchain.Input, owner ids.ShortID) ([]byte, bool) {
	for k, index := range in.SigIndices {
		if k >= len(in.Signatures) {
			return nil, false
		}
		if uint64(index) < uint64(len(in.Addresses)) && in.Addresses[index] == owner {
			return in.Signatures[k][:], true
		}
	}
	return nil, false
}

// smallestBech32 returns the address whose bech32 string with hrp is the
// lexically smallest; the zero address for an empty addrs.
func smallestBech32(hrp string, addrs []ids.ShortID) (ids.ShortID, error) {
	var (
		best    ids.ShortID
		bestStr string
	)
	for i, addr := range addrs {
		str, err := address.FormatBech32(hrp, addr.Bytes())
		if err != nil {
			return ids.ShortID{}, err
		}
		if i == 0 || str < bestStr {
			best, bestStr = addr, str
		}
	}
	return best, nil
}
