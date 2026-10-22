// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package pchainreader

import (
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/pchain"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/components/verify"
	"github.com/ava-labs/avalanchego/vms/platformvm/platform"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
)

var (
	_ platform.BlockVisitor = (*kindVisitor)(nil)
	_ platform.TxVisitor    = (*txDecoder)(nil)
)

// decodedBlock is the cached, decoded form of an accepted block.
type decodedBlock struct {
	block pchain.Block
	// bytes is the stored size of the block, charged on every read.
	bytes uint64
	// ins holds the consumed inputs of every decoded transaction, indexed
	// like block.Txs; nil for unsupported transactions. For import
	// transactions the P-chain inputs come first, then the imported inputs.
	ins [][]*avax.TransferableInput
	// importedFrom is, per transaction, the position in ins at which the
	// imported inputs start (len(ins[i]) when there are none).
	importedFrom []uint32
	// creds holds, per decoded transaction and aligned with ins, the
	// secp256k1fx credential of each input (nil for a missing or other
	// credential type); see inputCredentials. Together with the unsigned
	// bytes kept in block.Txs they let a reader recover the signers, and keep
	// about the transactions' serialized size alive per cached block.
	creds [][]*secp256k1fx.Credential
	// outs references the produced outputs of every decoded transaction
	// inside the parsed transaction; they are converted per requested page.
	outs []txOutputs
}

// txOutputs references the produced outputs of one transaction: the BaseTx
// outputs, then the type-specific extra outputs (the stake outputs of a
// staker, the exported outputs of an export) at the positions the P-chain
// assigns them (avax.Produce, unstakeUTXOs, the export executor).
type txOutputs struct {
	outs      []*avax.TransferableOutput
	extra     []*avax.TransferableOutput
	extraKind pchain.OutputKind
}

func (o txOutputs) count() uint32 {
	return uint32(len(o.outs) + len(o.extra))
}

// at returns the output at position i and its kind.
func (o txOutputs) at(i int) (*avax.TransferableOutput, pchain.OutputKind) {
	if i < len(o.outs) {
		return o.outs[i], pchain.OutputTransfer
	}
	return o.extra[i-len(o.outs)], o.extraKind
}

func (r *Reader) decode(blk platform.Block) *decodedBlock {
	kind := &kindVisitor{}
	_ = blk.Visit(kind) // the visitor never fails

	var timestamp uint64
	if banff, ok := blk.(platform.BanffBlock); ok {
		timestamp = uint64(banff.Timestamp().Unix())
	}
	// Apricot blocks (timestamp 0) predate Durango on every network.
	durango := r.upgrades.IsDurangoActivated(time.Unix(int64(timestamp), 0))

	rawTxs := blk.Txs()
	db := &decodedBlock{
		block: pchain.Block{
			Height:    blk.Height(),
			ID:        blk.ID(),
			ParentID:  blk.Parent(),
			Timestamp: timestamp,
			Kind:      kind.kind,
			Txs:       make([]pchain.Tx, len(rawTxs)),
		},
		ins:          make([][]*avax.TransferableInput, len(rawTxs)),
		importedFrom: make([]uint32, len(rawTxs)),
		creds:        make([][]*secp256k1fx.Credential, len(rawTxs)),
		outs:         make([]txOutputs, len(rawTxs)),
	}
	for i, tx := range rawTxs {
		d := newTxDecoder(timestamp, durango, uint32(i), tx.ID())
		if err := tx.Unsigned.Visit(d); err != nil {
			// Only a malformed auto-renewed node ID can fail, and accepted
			// transactions are syntactically verified; expose the transaction
			// as unsupported rather than failing the whole block.
			d = newTxDecoder(timestamp, durango, uint32(i), tx.ID())
		}
		if d.tx.Type != pchain.TxUnsupported {
			d.tx.UnsignedBytes = tx.Unsigned.Bytes()
			db.creds[i] = inputCredentials(tx.Creds, len(d.ins))
		}
		db.block.Txs[i] = d.tx
		db.ins[i] = d.ins
		db.importedFrom[i] = d.importedFrom
		db.outs[i] = d.outs
	}
	return db
}

// inputCredentials returns the secp256k1fx credentials of the first n inputs
// of a transaction, in input order. Consensus pairs credential i with input i
// (VerifySpendUTXOs; an import's imported inputs follow its P-chain inputs,
// as in setInputs) and treats any credential beyond the inputs as an
// authorization (the validator-authority credential of a config
// transaction), which is left out. A missing or non-secp256k1fx credential
// leaves a nil entry.
func inputCredentials(creds []verify.Verifiable, n int) []*secp256k1fx.Credential {
	if n == 0 {
		return nil
	}
	out := make([]*secp256k1fx.Credential, n)
	for i := 0; i < n && i < len(creds); i++ {
		if cred, ok := creds[i].(*secp256k1fx.Credential); ok {
			out[i] = cred
		}
	}
	return out
}

// kindVisitor maps a block's concrete type to its [pchain.BlockKind].
type kindVisitor struct {
	kind pchain.BlockKind
}

func (v *kindVisitor) BanffAbortBlock(*platform.BanffAbortBlock) error {
	v.kind = pchain.BlockAbort
	return nil
}

func (v *kindVisitor) BanffCommitBlock(*platform.BanffCommitBlock) error {
	v.kind = pchain.BlockCommit
	return nil
}

func (v *kindVisitor) BanffProposalBlock(*platform.BanffProposalBlock) error {
	v.kind = pchain.BlockProposal
	return nil
}

func (v *kindVisitor) BanffStandardBlock(*platform.BanffStandardBlock) error {
	v.kind = pchain.BlockStandard
	return nil
}

func (v *kindVisitor) ApricotAbortBlock(*platform.ApricotAbortBlock) error {
	v.kind = pchain.BlockAbort
	return nil
}

func (v *kindVisitor) ApricotCommitBlock(*platform.ApricotCommitBlock) error {
	v.kind = pchain.BlockCommit
	return nil
}

func (v *kindVisitor) ApricotProposalBlock(*platform.ApricotProposalBlock) error {
	v.kind = pchain.BlockProposal
	return nil
}

func (v *kindVisitor) ApricotStandardBlock(*platform.ApricotStandardBlock) error {
	v.kind = pchain.BlockStandard
	return nil
}

func (v *kindVisitor) ApricotAtomicBlock(*platform.ApricotAtomicBlock) error {
	v.kind = pchain.BlockAtomic
	return nil
}

// txDecoder decodes one transaction into a [pchain.Tx] plus references to
// its inputs and outputs. Transactions the gateway does not expose leave
// tx.Type at [pchain.TxUnsupported].
type txDecoder struct {
	blockTime uint64
	durango   bool

	tx           pchain.Tx
	ins          []*avax.TransferableInput
	importedFrom uint32
	outs         txOutputs
}

func newTxDecoder(blockTime uint64, durango bool, index uint32, id ids.ID) *txDecoder {
	return &txDecoder{
		blockTime: blockTime,
		durango:   durango,
		tx:        pchain.Tx{Index: index, ID: id},
	}
}

// startTime applies the Durango rule: post-Durango stakers start at the chain
// time of the accepting block; before, at the transaction's start field.
func (d *txDecoder) startTime(txStart uint64) uint64 {
	if d.durango {
		return d.blockTime
	}
	return txStart
}

// setInputs records the consumed inputs: the P-chain inputs ins followed by
// the imported inputs of an import transaction. The parsed slices are never
// appended to in place.
func (d *txDecoder) setInputs(ins, imported []*avax.TransferableInput) {
	if len(imported) == 0 {
		d.ins = ins
	} else {
		all := make([]*avax.TransferableInput, 0, len(ins)+len(imported))
		all = append(all, ins...)
		d.ins = append(all, imported...)
	}
	d.importedFrom = uint32(len(ins))
	d.tx.InputCount = uint32(len(d.ins))
}

// setOutputs records the produced outputs: the BaseTx outputs outs followed
// by the type-specific extra outputs of kind extraKind.
func (d *txDecoder) setOutputs(outs, extra []*avax.TransferableOutput, extraKind pchain.OutputKind) {
	d.outs = txOutputs{outs: outs, extra: extra, extraKind: extraKind}
	d.tx.OutputCount = d.outs.count()
}

func (d *txDecoder) boundedStaker(
	txType pchain.TxType,
	validator platform.Validator,
	delegationShares uint32,
	ins []*avax.TransferableInput,
	outs []*avax.TransferableOutput,
	stake []*avax.TransferableOutput,
) {
	d.tx.Type = txType
	d.tx.NodeID = validator.NodeID
	d.tx.StartTime = d.startTime(validator.Start)
	d.tx.EndTime = validator.End
	d.tx.Weight = validator.Wght
	d.tx.DelegationShares = delegationShares
	d.setInputs(ins, nil)
	d.setOutputs(outs, stake, pchain.OutputStake)
}

// transfer decodes a value-moving transaction: it takes effect at the block
// time and carries nothing but inputs and outputs.
func (d *txDecoder) transfer(
	txType pchain.TxType,
	ins, imported []*avax.TransferableInput,
	outs, extra []*avax.TransferableOutput,
	extraKind pchain.OutputKind,
) {
	d.tx.Type = txType
	d.tx.StartTime = d.blockTime
	d.setInputs(ins, imported)
	d.setOutputs(outs, extra, extraKind)
}

// Apricot transactions.

func (d *txDecoder) AddValidatorTx(tx *platform.AddValidatorTx) error {
	d.boundedStaker(pchain.TxAddValidator, tx.Validator, tx.DelegationShares, tx.Ins, tx.Outs, tx.StakeOuts)
	return nil
}

func (*txDecoder) AddSubnetValidatorTx(*platform.AddSubnetValidatorTx) error {
	return nil
}

func (d *txDecoder) AddDelegatorTx(tx *platform.AddDelegatorTx) error {
	d.boundedStaker(pchain.TxAddDelegator, tx.Validator, 0, tx.Ins, tx.Outs, tx.StakeOuts)
	return nil
}

func (*txDecoder) CreateChainTx(*platform.CreateChainTx) error {
	return nil
}

func (*txDecoder) CreateSubnetTx(*platform.CreateSubnetTx) error {
	return nil
}

func (d *txDecoder) ImportTx(tx *platform.ImportTx) error {
	d.transfer(pchain.TxImport, tx.Ins, tx.ImportedInputs, tx.Outs, nil, pchain.OutputTransfer)
	d.tx.RemoteChainID = tx.SourceChain
	return nil
}

func (d *txDecoder) ExportTx(tx *platform.ExportTx) error {
	d.transfer(pchain.TxExport, tx.Ins, nil, tx.Outs, tx.ExportedOutputs, pchain.OutputExported)
	d.tx.RemoteChainID = tx.DestinationChain
	return nil
}

func (*txDecoder) AdvanceTimeTx(*platform.AdvanceTimeTx) error {
	return nil
}

// RewardValidatorTx removes a bounded staker at its end time; it is only
// valid once the chain time (the block timestamp) has reached that end.
func (d *txDecoder) RewardValidatorTx(tx *platform.RewardValidatorTx) error {
	d.tx.Type = pchain.TxRewardValidator
	d.tx.StartTime = d.blockTime
	d.tx.EndTime = d.blockTime
	d.tx.StakerTxID = tx.TxID
	return nil
}

// Banff transactions.

func (*txDecoder) RemoveSubnetValidatorTx(*platform.RemoveSubnetValidatorTx) error {
	return nil
}

func (*txDecoder) TransformSubnetTx(*platform.TransformSubnetTx) error {
	return nil
}

func (d *txDecoder) AddPermissionlessValidatorTx(tx *platform.AddPermissionlessValidatorTx) error {
	if tx.Subnet != constants.PrimaryNetworkID {
		return nil
	}
	d.boundedStaker(pchain.TxAddPermissionlessValidator, tx.Validator, tx.DelegationShares, tx.Ins, tx.Outs, tx.StakeOuts)
	return nil
}

func (d *txDecoder) AddPermissionlessDelegatorTx(tx *platform.AddPermissionlessDelegatorTx) error {
	if tx.Subnet != constants.PrimaryNetworkID {
		return nil
	}
	d.boundedStaker(pchain.TxAddPermissionlessDelegator, tx.Validator, 0, tx.Ins, tx.Outs, tx.StakeOuts)
	return nil
}

// Durango transactions.

func (*txDecoder) TransferSubnetOwnershipTx(*platform.TransferSubnetOwnershipTx) error {
	return nil
}

func (d *txDecoder) BaseTx(tx *platform.BaseTx) error {
	d.transfer(pchain.TxBase, tx.Ins, nil, tx.Outs, nil, pchain.OutputTransfer)
	return nil
}

// Etna transactions.

func (*txDecoder) ConvertSubnetToL1Tx(*platform.ConvertSubnetToL1Tx) error {
	return nil
}

func (*txDecoder) RegisterL1ValidatorTx(*platform.RegisterL1ValidatorTx) error {
	return nil
}

func (*txDecoder) SetL1ValidatorWeightTx(*platform.SetL1ValidatorWeightTx) error {
	return nil
}

func (*txDecoder) IncreaseL1ValidatorBalanceTx(*platform.IncreaseL1ValidatorBalanceTx) error {
	return nil
}

func (*txDecoder) DisableL1ValidatorTx(*platform.DisableL1ValidatorTx) error {
	return nil
}

// Helicon transactions.

func (d *txDecoder) AddAutoRenewedValidatorTx(tx *platform.AddAutoRenewedValidatorTx) error {
	nodeID, err := ids.ToNodeID(tx.ValidatorNodeID)
	if err != nil {
		return err
	}
	d.tx.Type = pchain.TxAddAutoRenewedValidator
	d.tx.NodeID = nodeID
	d.tx.StartTime = d.blockTime
	d.tx.Period = tx.Period
	d.tx.EndTime = d.blockTime + tx.Period
	d.tx.Weight = tx.Weight()
	d.tx.DelegationShares = tx.DelegationShares
	d.tx.AutoCompoundRewardShares = tx.AutoCompoundRewardShares
	d.setInputs(tx.Ins, nil)
	d.setOutputs(tx.Outs, tx.StakeOuts, pchain.OutputStake)
	return nil
}

func (d *txDecoder) SetAutoRenewedValidatorConfigTx(tx *platform.SetAutoRenewedValidatorConfigTx) error {
	d.tx.Type = pchain.TxSetAutoRenewedValidatorConfig
	d.tx.StartTime = d.blockTime
	d.tx.Period = tx.Period
	d.tx.AutoCompoundRewardShares = tx.AutoCompoundRewardShares
	d.tx.StakerTxID = tx.TxID
	d.setInputs(tx.Ins, nil)
	d.setOutputs(tx.Outs, nil, pchain.OutputTransfer)
	return nil
}

func (d *txDecoder) RewardAutoRenewedValidatorTx(tx *platform.RewardAutoRenewedValidatorTx) error {
	d.tx.Type = pchain.TxRewardAutoRenewedValidator
	d.tx.StartTime = d.blockTime
	d.tx.EndTime = tx.Timestamp
	d.tx.StakerTxID = tx.TxID
	return nil
}
