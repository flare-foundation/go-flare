// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package pchainreader

import (
	"fmt"

	"github.com/ava-labs/avalanchego/snow/pchain"
	"github.com/ava-labs/avalanchego/vms/components/avax"
)

func (b *blockReader) GetOutputs(txIndex, from, max uint32) ([]pchain.Output, uint32, error) {
	db := b.db
	if uint64(txIndex) >= uint64(len(db.block.Txs)) {
		return nil, 0, fmt.Errorf("%w: index %d, block has %d transactions", pchain.ErrTxIndexOutOfRange, txIndex, len(db.block.Txs))
	}
	if db.block.Txs[txIndex].Type == pchain.TxUnsupported {
		return nil, 0, fmt.Errorf("%w: transaction %d", pchain.ErrUnsupportedTx, txIndex)
	}

	outs := db.outs[txIndex]
	total := outs.count()
	if from > total {
		return nil, total, fmt.Errorf("%w: from %d, transaction has %d outputs", pchain.ErrOutputIndexOutOfRange, from, total)
	}
	end := total
	if max < total-from {
		end = from + max
	}
	// Only the requested page is converted; the decoded block keeps
	// references into the parsed transaction.
	outputs := make([]pchain.Output, 0, end-from)
	for i := from; i < end; i++ {
		out, kind := outs.at(int(i))
		outputs = append(outputs, outputFrom(out, kind))
	}
	return outputs, total, nil
}

// outputFrom converts a produced output of kind kind. The owner is read the
// same way as for consumed outputs (ownersFromOutput); an output of an
// unknown type has an asset and an amount but no owner.
func outputFrom(out *avax.TransferableOutput, kind pchain.OutputKind) pchain.Output {
	output := pchain.Output{Kind: kind}
	if out == nil {
		return output
	}
	output.AssetID = out.AssetID()
	if out.Out == nil {
		return output
	}
	output.Amount = out.Out.Amount()
	owner := ownersFromOutput(out.Out)
	if owner.resolved {
		output.Locktime = owner.locktime
		output.StakeableLocktime = owner.stakeableLocktime
		output.Threshold = owner.threshold
		output.Addresses = owner.addrs
	}
	return output
}
