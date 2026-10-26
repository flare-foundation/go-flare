// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package pchaingateway

// Gas schedule and protocol caps. Gas is charged before any P-chain access and
// depends only on the call arguments plus, for owner addresses, on the
// returned data, so it is identical on every node. The schedule targets the
// ~100 Mgas/s of the Granite warp costs: a block read (height index + block
// fetch + decode, cached after the first read) and an input resolution (one
// producing-transaction fetch) are priced as database reads; an output is a
// conversion inside the decoded block and priced far below an input.
const (
	// GetPChainHeightGas is the cost of reading the header-committed height.
	GetPChainHeightGas uint64 = 200
	// GetNativeAssetIdGas is the cost of reading the native asset ID from
	// the chain context.
	GetNativeAssetIdGas uint64 = 200
	// BlockReadGas is charged by every call that reads a P-chain block.
	BlockReadGas uint64 = 20_000
	// TxGas is charged per requested transaction slot of getTransactions.
	TxGas uint64 = 5_000
	// InputGas is charged per requested input slot (inline inputs of
	// getTransactions, inputs of getTransactionInputs, the single input of
	// getStakeOwner).
	InputGas uint64 = 10_000
	// OutputGas is charged per requested output slot (inline outputs of
	// getTransactions, outputs of getTransactionOutputs). Outputs need no
	// database read: they are converted from the decoded block, whose bytes
	// the call has already paid for.
	OutputGas uint64 = 1_000
	// AddressGas is charged per returned owner address (per candidate
	// address for getStakeOwner) once the data is known.
	AddressGas uint64 = 200
	// ByteGas is charged per byte of P-chain data the node decodes for a
	// call: the stored bytes of the block read and of every producing
	// transaction resolved for an input. It is charged before the bytes are
	// parsed, and equally when the reader serves a cached copy, so gas never
	// depends on node-local caches. Calibration: parsing plus decoding runs at
	// ~65 MB/s (~15 ns/byte, transaction-dense blocks) on an M-series laptop
	// (BenchmarkParse* in vms/platformvm/pchainreader), i.e. ~1.5 gas/byte at
	// the 100 Mgas/s target; 10 gas/byte keeps a >6x margin for slower
	// validators. A 2 MiB block (the P-chain maximum) costs ~21M gas, a
	// 128 KiB block (the builder's target size) ~1.3M gas.
	ByteGas uint64 = 10
	// SignatureRecoveryGas is charged by getStakeOwner when its input has a
	// candidate owner: the owner's C-chain address is recovered from the
	// owner's signature on the transaction, which takes up to two secp256k1
	// public-key recoveries (one per signing scheme the P-chain accepts, see
	// vms/secp256k1fx.SigningHashes), each priced like the EVM ecrecover
	// precompile (3,000 gas). It is charged whichever scheme matches and
	// whether or not the recovery succeeds, so it depends on chain data only.
	SignatureRecoveryGas uint64 = 6_000

	// MaxTxsPerCall caps maxTxs of getTransactions.
	MaxTxsPerCall = 32
	// MaxInlineInputsPerTx caps maxInputsPerTx of getTransactions.
	MaxInlineInputsPerTx = 8
	// MaxInlineOutputsPerTx caps maxOutputsPerTx of getTransactions.
	MaxInlineOutputsPerTx = 8
	// MaxInputsPerCall caps maxInputs of getTransactionInputs.
	MaxInputsPerCall = 64
	// MaxOutputsPerCall caps maxOutputs of getTransactionOutputs.
	MaxOutputsPerCall = 64
)

// GetBlockGas is the cost of getBlock for a block of blockBytes stored bytes.
func GetBlockGas(blockBytes uint64) uint64 {
	return BlockReadGas + blockBytes*ByteGas
}

// GetTransactionsGas is the cost of getTransactions for the given page sizes,
// the total stored bytes decoded (the block plus the producing transactions
// of the inline inputs) and the number of owner addresses returned inline
// (inputs and outputs). Arguments must be within the caps.
func GetTransactionsGas(maxTxs, maxInputsPerTx, maxOutputsPerTx uint32, decodedBytes uint64, addresses int) uint64 {
	return BlockReadGas +
		uint64(maxTxs)*TxGas +
		uint64(maxTxs)*uint64(maxInputsPerTx)*InputGas +
		uint64(maxTxs)*uint64(maxOutputsPerTx)*OutputGas +
		decodedBytes*ByteGas +
		uint64(addresses)*AddressGas
}

// GetTransactionInputsGas is the cost of getTransactionInputs for the given
// page size, the total stored bytes decoded and the number of owner addresses
// returned.
func GetTransactionInputsGas(maxInputs uint32, decodedBytes uint64, addresses int) uint64 {
	return BlockReadGas + uint64(maxInputs)*InputGas + decodedBytes*ByteGas + uint64(addresses)*AddressGas
}

// GetTransactionOutputsGas is the cost of getTransactionOutputs for the given
// page size, the block's stored bytes and the number of owner addresses
// returned.
func GetTransactionOutputsGas(maxOutputs uint32, blockBytes uint64, addresses int) uint64 {
	return BlockReadGas + uint64(maxOutputs)*OutputGas + blockBytes*ByteGas + uint64(addresses)*AddressGas
}

// GetStakeOwnerGas is the cost of getStakeOwner given the total stored bytes
// decoded and the number of candidate owner addresses of the first input (the
// signature recovery is charged when there is one).
func GetStakeOwnerGas(decodedBytes uint64, candidates int) uint64 {
	gas := BlockReadGas + InputGas + decodedBytes*ByteGas + uint64(candidates)*AddressGas
	if candidates > 0 {
		gas += SignatureRecoveryGas
	}
	return gas
}
