// Copyright (C) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package avalanche

import (
	"go.uber.org/zap"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/consensus/avalanche"
	"github.com/ava-labs/avalanchego/snow/consensus/snowstorm"
	"github.com/ava-labs/avalanchego/snow/engine/common"
)

// issuer issues [vtx] into consensus after its dependencies are met.
type issuer struct {
	t                 *Transitive
	vtx               avalanche.Vertex
	issued, abandoned bool
	vtxDeps, txDeps   ids.Set
}

// Register that a vertex we were waiting on has been issued to consensus.
func (i *issuer) FulfillVtx(id ids.ID) {
	i.vtxDeps.Remove(id)
	i.Update()
}

// Register that a transaction we were waiting on has been issued to consensus.
func (i *issuer) FulfillTx(id ids.ID) {
	i.txDeps.Remove(id)
	i.Update()
}

// Abandon this attempt to issue
func (i *issuer) Abandon() {
	if !i.abandoned {
		vtxID := i.vtx.ID()
		i.t.pending.Remove(vtxID)
		i.abandoned = true
		i.t.vtxBlocked.Abandon(vtxID) // Inform vertices waiting on this vtx that it won't be issued
		i.t.metrics.blockerVtxs.Set(float64(i.t.vtxBlocked.Len()))
	}
}

// Issue the poll when all dependencies are met
func (i *issuer) Update() {
	if i.abandoned || i.issued || i.vtxDeps.Len() != 0 || i.txDeps.Len() != 0 || i.t.Consensus.VertexIssued(i.vtx) || i.t.errs.Errored() {
		return
	}

	vtxID := i.vtx.ID()

	// All dependencies have been met
	i.issued = true

	// check stop vertex validity
	err := i.vtx.Verify()
	if err != nil {
		if i.vtx.HasWhitelist() {
			// do not update "i.t.errs" since it's only used for critical errors
			// which will cause chain shutdown in the engine
			// (see "handleSyncMsg" and "handleChanMsg")
			i.t.Ctx.Log.Debug("stop vertex verification failed",
				zap.Stringer("vtxID", vtxID),
				zap.Error(err),
			)
			i.t.metrics.whitelistVtxIssueFailure.Inc()
		} else {
			i.t.Ctx.Log.Debug("vertex verification failed",
				zap.Stringer("vtxID", vtxID),
				zap.Error(err),
			)
		}

		i.t.vtxBlocked.Abandon(vtxID)
		return
	}

	i.t.pending.Remove(vtxID) // Remove from set of vertices waiting to be issued.

	// Make sure the transactions in this vertex are valid
	txs, err := i.vtx.Txs()
	if err != nil {
		i.t.errs.Add(err)
		return
	}
	validTxs := make([]snowstorm.Tx, 0, len(txs))
	for _, tx := range txs {
		if err := tx.Verify(); err != nil {
			txID := tx.ID()
			i.t.Ctx.Log.Debug("transaction verification failed",
				zap.Stringer("txID", txID),
				zap.Error(err),
			)
			i.t.txBlocked.Abandon(txID)
		} else {
			validTxs = append(validTxs, tx)
		}
	}

	// Some of the transactions weren't valid. Abandon this vertex.
	// Take the valid transactions and issue a new vertex with them.
	if len(validTxs) != len(txs) {
		i.t.Ctx.Log.Debug("abandoning vertex",
			zap.String("reason", "transaction verification failed"),
			zap.Stringer("vtxID", vtxID),
		)
		if _, err := i.t.batch(validTxs, batchOption{}); err != nil {
			i.t.errs.Add(err)
		}
		i.t.vtxBlocked.Abandon(vtxID)
		i.t.metrics.blockerVtxs.Set(float64(i.t.vtxBlocked.Len()))
		return
	}

	i.t.Ctx.Log.Verbo("adding vertex to consensus",
		zap.Stringer("vtxID", vtxID),
	)

	// Add this vertex to consensus.
	if err := i.t.Consensus.Add(i.vtx); err != nil {
		i.t.errs.Add(err)
		return
	}

	// Issue a poll for this vertex.
	p := i.t.Consensus.Parameters()
	vdrs, err := i.t.Validators.Sample(p.K) // Validators to sample
	if err != nil {
		i.t.Ctx.Log.Error("dropped query",
			zap.String("reason", "insufficient number of validators"),
			zap.Stringer("vtxID", vtxID),
		)
	}

	vdrBag := ids.NodeIDBag{} // Validators to sample repr. as a set
	for _, vdr := range vdrs {
		vdrBag.Add(vdr.ID())
	}

	i.t.RequestID++
	if err == nil && i.t.polls.Add(i.t.RequestID, vdrBag) {
		numPushTo := i.t.Params.MixedQueryNumPushVdr
		if !i.t.Validators.Contains(i.t.Ctx.NodeID) {
			numPushTo = i.t.Params.MixedQueryNumPushNonVdr
		}
		common.SendMixedQuery(
			i.t.Sender,
			vdrBag.List(), // Note that this doesn't contain duplicates; length may be < k
			numPushTo,
			i.t.RequestID,
			vtxID,
			i.vtx.Bytes(),
		)
	}

	// Notify vertices waiting on this one that it (and its transactions) have been issued.
	i.t.vtxBlocked.Fulfill(vtxID)
	for _, tx := range txs {
		i.t.txBlocked.Fulfill(tx.ID())
	}
	i.t.metrics.blockerTxs.Set(float64(i.t.txBlocked.Len()))
	i.t.metrics.blockerVtxs.Set(float64(i.t.vtxBlocked.Len()))

	if i.vtx.HasWhitelist() {
		i.t.Ctx.Log.Info("successfully issued stop vertex",
			zap.Stringer("vtxID", vtxID),
		)
		i.t.metrics.whitelistVtxIssueSuccess.Inc()
	}

	// Issue a repoll
	i.t.repoll()
}

type vtxIssuer struct{ i *issuer }

func (vi *vtxIssuer) Dependencies() ids.Set { return vi.i.vtxDeps }
func (vi *vtxIssuer) Fulfill(id ids.ID)     { vi.i.FulfillVtx(id) }
func (vi *vtxIssuer) Abandon(ids.ID)        { vi.i.Abandon() }
func (vi *vtxIssuer) Update()               { vi.i.Update() }

type txIssuer struct{ i *issuer }

func (ti *txIssuer) Dependencies() ids.Set { return ti.i.txDeps }
func (ti *txIssuer) Fulfill(id ids.ID)     { ti.i.FulfillTx(id) }
func (ti *txIssuer) Abandon(ids.ID)        { ti.i.Abandon() }
func (ti *txIssuer) Update()               { ti.i.Update() }
