// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

// Package pchain is the read-only, in-process view of accepted P-chain blocks
// and their transactions (primary-network staking transactions, imports,
// exports and plain transfers) that other chains consume, in particular the
// C-chain "P-chain gateway" precompile.
//
// It is a leaf package (it imports only ids) so that snow, the platformvm and
// coreth can all depend on it: the platformvm implements [Reader],
// chains/manager wires it into snow.Context.PChainReader and coreth's
// precompile only sees these plain types.
//
// Every value returned by a [Reader] is derived from accepted P-chain blocks
// only and is therefore identical on every node that has accepted the same
// height; callers are responsible for bounding the heights they read by a
// consensus-committed value.
package pchain

import (
	"errors"
	"sync"

	"github.com/ava-labs/avalanchego/ids"
)

// SignatureLen is the length of a secp256k1 credential signature (r||s||v).
// It equals utils/crypto/secp256k1.SignatureLen; it is kept local so that this
// package stays a leaf (the reader's assignment of credential signatures to
// [Input.Signatures] does not compile if the two ever differ).
const SignatureLen = 65

var (
	// ErrNotAccepted is returned when a height above the local last accepted
	// P-chain height is requested.
	ErrNotAccepted           = errors.New("P-chain height not accepted yet")
	ErrTxIndexOutOfRange     = errors.New("transaction index out of range")
	ErrUnsupportedTx         = errors.New("unsupported transaction type")
	ErrInputIndexOutOfRange  = errors.New("input index out of range")
	ErrOutputIndexOutOfRange = errors.New("output index out of range")
)

// BlockKind is the kind of an accepted P-chain block.
type BlockKind uint8

const (
	BlockStandard BlockKind = iota
	// BlockProposal blocks carry decision transactions followed by exactly
	// one proposal transaction (the last element of [Block.Txs]). The
	// proposal's outcome is the kind of the block at height+1: [BlockCommit]
	// or [BlockAbort].
	BlockProposal
	BlockCommit
	BlockAbort
	// BlockAtomic is the pre-Apricot-Phase-5 atomic block (historical only).
	BlockAtomic
)

// TxType identifies the decoded transaction types. Bit `t` of a type filter
// mask selects type `t`.
type TxType uint8

const (
	// TxUnsupported marks transactions the gateway does not decode. They are
	// still counted in [Block.Txs] so that indexes match the block's raw
	// transaction list.
	TxUnsupported TxType = iota
	TxAddPermissionlessValidator
	TxAddPermissionlessDelegator
	TxAddAutoRenewedValidator
	TxSetAutoRenewedValidatorConfig
	TxRewardAutoRenewedValidator
	// TxAddValidator and TxAddDelegator are the pre-Durango staking
	// transactions (historical blocks only).
	TxAddValidator
	TxAddDelegator
	// TxImport moves funds from another chain of the primary network (the
	// C-chain or the X-chain) to the P-chain: it consumes UTXOs exported by
	// that chain ([Input.Imported]) and produces P-chain outputs.
	TxImport
	// TxExport moves funds from the P-chain to another chain of the primary
	// network: its exported outputs ([OutputExported]) become UTXOs of the
	// destination chain.
	TxExport
	// TxBase is a plain P-chain transfer (Durango and later).
	TxBase
	// TxRewardValidator is the proposal transaction that removes a bounded
	// staker (a permissionless or legacy validator or delegator) once its end
	// time is reached; the outcome is the kind of the block at height+1.
	TxRewardValidator

	// NumTxTypes is one past the highest [TxType].
	NumTxTypes
)

// OutputKind is the kind of a produced output; see [Output].
type OutputKind uint8

const (
	// OutputTransfer is a P-chain UTXO: (transaction ID, output position).
	OutputTransfer OutputKind = iota
	// OutputStake is a staked output: locked while the staker is active and
	// returned as a P-chain UTXO with the same transaction ID and position
	// when the staker is removed (bounded stakers at their end time,
	// auto-renewed validators on exit or abort; never on renewal).
	OutputStake
	// OutputExported is a UTXO created on the destination chain of an export
	// transaction, with the same transaction ID and position. Its owner
	// addresses are secp256k1fx addresses of that chain, not EVM addresses.
	OutputExported
)

// Block is an accepted P-chain block.
type Block struct {
	Height   uint64
	ID       ids.ID
	ParentID ids.ID
	// Timestamp is the block time in Unix seconds; 0 for Apricot-era blocks,
	// which carry no timestamp.
	Timestamp uint64
	Kind      BlockKind
	// Txs has one entry per raw transaction of the block, in block order.
	Txs []Tx
}

// Tx is a decoded transaction. Fields that do not apply to a type are zero.
// Weight is in units of the native asset (nAVAX, i.e. nFLR): the P-chain
// only accepts primary-network stakes of that asset. Input and output
// amounts are in units of their own [Input.AssetID] / [Output.AssetID]. All
// times are in Unix seconds.
type Tx struct {
	Type TxType
	// Index is the position in the block's raw transaction list.
	Index uint32
	ID    ids.ID

	NodeID ids.NodeID
	// StartTime is the effective start of the stake for bounded stakers: the
	// block timestamp when Durango is active at that block (the chain time
	// at acceptance), else the transaction's own start field. For every
	// other type it is the block timestamp (0 for Apricot-era blocks).
	StartTime uint64
	// EndTime is the stake end (bounded stakers), the end of the first cycle
	// (auto-renewed validators), the cycle end ([TxRewardAutoRenewedValidator])
	// or the removal time ([TxRewardValidator]: the block timestamp, which is
	// the removed staker's end time).
	EndTime uint64
	// Period is the auto-renew cycle length; for config transactions the
	// next cycle length (0 requests an exit at the end of the cycle).
	Period uint64
	// Weight is the staked amount.
	Weight uint64
	// DelegationShares is the validator's delegation fee, times 10,000.
	DelegationShares uint32
	// AutoCompoundRewardShares is the auto-renewed validator's restake share
	// in millionths.
	AutoCompoundRewardShares uint32
	// StakerTxID references the staker transaction of config and reward
	// transactions: the AddAutoRenewedValidatorTx for
	// [TxSetAutoRenewedValidatorConfig] and [TxRewardAutoRenewedValidator],
	// the removed bounded staker for [TxRewardValidator].
	StakerTxID ids.ID
	// RemoteChainID is the source chain of an import and the destination
	// chain of an export; zero for every other type.
	RemoteChainID ids.ID
	// InputCount is the number of consumed inputs; see [BlockReader.GetInputs].
	InputCount uint32
	// OutputCount is the number of produced outputs; see [BlockReader.GetOutputs].
	OutputCount uint32
	// UnsignedBytes is the serialized unsigned transaction: the message its
	// credentials sign, either as its plain sha256 digest or as the Ethereum
	// signed-message hash of that digest's hex encoding (see
	// vms/secp256k1fx.SigningHashes). Nil for [TxUnsupported].
	UnsignedBytes []byte
}

// Input is a consumed UTXO of a transaction together with the owner of the
// consumed output, resolved from the transaction that produced it.
type Input struct {
	UTXOTxID        ids.ID
	UTXOOutputIndex uint32
	// AssetID is the asset of the consumed output and Amount its quantity.
	// The P-chain UTXO set holds assets other than the native one (any
	// X-chain asset can be imported, transferred and exported), so an amount
	// is a native amount only when AssetID is the native asset ID
	// (snow.Context.AVAXAssetID). Consensus verifies that the asset an input
	// claims is the asset of the consumed output, for imported inputs too.
	AssetID ids.ID
	Amount  uint64
	// Imported is true for the inputs of an import transaction that consume
	// UTXOs exported by the source chain. They have no producing P-chain
	// transaction and are never resolved: only the UTXO ID, the amount and
	// the signature indices are known.
	Imported bool
	// Resolved is false when the producing output could not be located; the
	// owner fields are then zero.
	Resolved bool
	// Genesis is true when the consumed output is a genesis UTXO, which has
	// no producing transaction (production genesis UTXOs carry the empty
	// transaction ID).
	Genesis  bool
	Locktime uint64
	// StakeableLocktime is the locktime of the stakeable lock wrapping the
	// consumed output, 0 when the output is not stakeable-locked.
	StakeableLocktime uint64
	Threshold         uint32
	// Addresses are all owners of the consumed output in on-chain (byte)
	// order.
	Addresses []ids.ShortID
	// SigIndices are the positions in Addresses whose signatures authorized
	// the spend.
	SigIndices []uint32
	// Signatures are the signatures (r||s||v) of the input's secp256k1fx
	// credential, aligned with SigIndices: Signatures[k] was produced by the
	// owner Addresses[SigIndices[k]] over the transaction's
	// [Tx.UnsignedBytes]. Nil when the transaction carries no secp256k1fx
	// credential at the input's position. Imported inputs carry theirs too.
	Signatures [][SignatureLen]byte
}

// Output is a produced output of a transaction. Its position in the
// transaction's output list (the BaseTx outputs first, then the stake outputs
// of a staker or the exported outputs of an export) is the output index of
// the UTXO it becomes ([OutputTransfer]; [OutputStake] once returned) or
// creates on the destination chain ([OutputExported]), i.e. the
// [Input.UTXOOutputIndex] of an input consuming it.
type Output struct {
	Kind OutputKind
	// AssetID is the asset of the output and Amount its quantity. Stake
	// outputs ([OutputStake]) are always the native asset; transfer and
	// exported outputs may carry any asset (see [Input.AssetID]).
	AssetID ids.ID
	Amount  uint64
	// Locktime, Threshold and Addresses are the secp256k1fx owner of the
	// output; an output of any other type (which the P-chain never produces)
	// reports threshold 0 and no addresses.
	Locktime uint64
	// StakeableLocktime is the locktime of the stakeable lock wrapping the
	// output, 0 when the output is not stakeable-locked.
	StakeableLocktime uint64
	Threshold         uint32
	// Addresses are all owners in on-chain (byte) order.
	Addresses []ids.ShortID
}

// ByteCharger accounts for the P-chain consensus bytes a read decodes: the
// stored bytes of the block being read ([Reader.GetBlock]) and of every
// producing transaction resolved for an input ([BlockReader.GetInputs]). It
// is called before the bytes are parsed, and with the same amount when the
// reader serves a cached copy, so that callers can charge gas that never
// depends on node-local cache state. Returning an error aborts the read; the
// error is returned unchanged. A nil ByteCharger charges nothing.
type ByteCharger func(bytes uint64) error

// Reader reads accepted P-chain blocks. Implementations are safe for
// concurrent use. Returned values must be treated as read-only.
type Reader interface {
	// LastAcceptedHeight returns the height of the last accepted P-chain
	// block of this node.
	LastAcceptedHeight() (uint64, error)
	// GetBlock returns the accepted block at height, or [ErrNotAccepted] if
	// the local P-chain has not accepted it yet. charge is called with the
	// block's stored size before it is decoded. The returned [BlockReader]
	// holds the decoded block, so reading its inputs never decodes (or
	// charges) the block again.
	GetBlock(height uint64, charge ByteCharger) (BlockReader, error)
}

// BlockReader is a decoded accepted block. It stays valid for as long as the
// caller holds it, independently of the [Reader]'s caches.
type BlockReader interface {
	// Block returns the decoded block.
	Block() *Block
	// GetInputs returns up to max inputs of the transaction at txIndex,
	// starting at position from, and the total number of inputs. Inputs are
	// in the transaction's serialized (canonical) order: the P-chain inputs
	// followed, for import transactions, by the imported inputs, each group
	// sorted by UTXO transaction ID bytes, then by output index (the P-chain
	// requires that order; it is the Flare P-chain indexer's input index).
	// Only decoded (non-[TxUnsupported]) transactions have inputs. charge is
	// called with the stored size of each producing transaction that is
	// resolved (genesis UTXOs and imported inputs have none) before it is
	// decoded.
	GetInputs(txIndex, from, max uint32, charge ByteCharger) ([]Input, uint32, error)
	// GetOutputs returns up to max outputs of the transaction at txIndex,
	// starting at position from, and the total number of outputs. Only
	// decoded transactions have outputs. Outputs are part of the decoded
	// block, so nothing is charged.
	GetOutputs(txIndex, from, max uint32) ([]Output, uint32, error)
}

// ReaderProvider is implemented by the P-chain VM. The returned Reader takes
// lock around every access to the P-chain state.
type ReaderProvider interface {
	NewPChainReader(lock sync.Locker) Reader
}
