// Copyright (C) 2019, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package block

import (
	"context"

	"github.com/ava-labs/avalanchego/ids"
)

// HeightByIDChainVM is an optional interface a [ChainVM] can implement to
// report the height of a block by ID without materialising the block.
//
// Flare addition: the proposervm proposer lookup API
// (proposervm.getProposerByEVMBlockHash) resolves a hash to a height with it,
// so that the lookup does not restore an SAE block, receipts included, from
// disk while it holds the context lock.
type HeightByIDChainVM interface {
	// GetBlockHeight returns the height of the block with the given ID, or
	// database.ErrNotFound. It is called with the context lock held and MUST
	// be cheap: an index read, never a block restore.
	GetBlockHeight(ctx context.Context, blkID ids.ID) (uint64, error)
}
