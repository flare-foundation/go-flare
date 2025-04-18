// Copyright (C) 2019-2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package avalanche

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"go.uber.org/zap"

	"golang.org/x/exp/maps"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/snow/choices"
	"github.com/ava-labs/avalanchego/snow/consensus/metrics"
	"github.com/ava-labs/avalanchego/snow/consensus/snowstorm"
	"github.com/ava-labs/avalanchego/utils/bag"
	"github.com/ava-labs/avalanchego/utils/set"
)

const minMapSize = 16

var (
	errNoLeaves = errors.New("couldn't pop a leaf from leaf set")

	_ Factory   = (*TopologicalFactory)(nil)
	_ Consensus = (*Topological)(nil)
)

// TopologicalFactory implements Factory by returning a topological struct
type TopologicalFactory struct{}

func (TopologicalFactory) New() Consensus {
	return &Topological{}
}

// TODO: Implement pruning of decisions.
// To perfectly preserve the protocol, this implementation will need to store
// the hashes of all accepted decisions. It is possible to add a heuristic that
// removes sufficiently old decisions. However, that will need to be analyzed to
// ensure safety. It is doable when adding in a weak synchrony assumption.

// Topological performs the avalanche algorithm by utilizing a topological sort
// of the voting results. Assumes that vertices are inserted in topological
// order.
type Topological struct {
	metrics.Latency

	// pollNumber is the number of times RecordPolls has been called
	pollNumber uint64

	// Context used for logging
	ctx *snow.ConsensusContext
	// Threshold for confidence increases
	params Parameters

	// Maps vtxID -> transactionVertex wrapping the provided vertex as a
	// transaction
	nodes map[ids.ID]*transactionVertex

	// Tracks the conflict relations
	cg snowstorm.Consensus

	// preferred is the frontier of vtxIDs that are strongly preferred
	preferred set.Set[ids.ID]

	// virtuous is the frontier of vtxIDs that are strongly virtuous
	virtuous set.Set[ids.ID]

	// orphans are the txIDs that are virtuous, but not preferred
	orphans set.Set[ids.ID]

	// virtuousVoting are the txIDs that are virtuous and still awaiting
	// additional votes before acceptance. transactionVertices whose vertices
	// are not considered virtuous are removed from this set.
	virtuousVoting set.Set[ids.ID]

	// frontier is the set of vts that have no descendents
	//
	// Invariant: frontier never contains a rejected vertex
	frontier map[ids.ID]Vertex
	// preferenceCache is the cache for strongly preferred checks
	// virtuousCache is the cache for strongly virtuous checks
	preferenceCache, virtuousCache map[ids.ID]bool

	// Used in [calculateInDegree] and [markAncestorInDegrees].
	// Should only be accessed in those methods.
	// We use this one instance of set.Set instead of creating a
	// new set.Set during each call to [calculateInDegree].
	leaves set.Set[ids.ID]

	// Kahn nodes used in [calculateInDegree] and [markAncestorInDegrees].
	// Should only be accessed in those methods.
	// We use this one map instead of creating a new map
	// during each call to [calculateInDegree].
	kahnNodes map[ids.ID]kahnNode

	// Used in [pushVotes]. Should only be accessed in that method.
	// We use this one instance instead of creating a new bag.UniqueBag[ids.ID]
	// during each call to [pushVotes].
	votes bag.UniqueBag[ids.ID]
}

type kahnNode struct {
	inDegree int
	votes    set.Bits64
}

func (ta *Topological) Initialize(
	ctx context.Context,
	chainCtx *snow.ConsensusContext,
	params Parameters,
	frontier []Vertex,
) error {
	if err := params.Valid(); err != nil {
		return err
	}

	ta.ctx = chainCtx
	ta.params = params
	ta.leaves = set.Set[ids.ID]{}
	ta.votes = bag.UniqueBag[ids.ID]{}
	ta.kahnNodes = make(map[ids.ID]kahnNode)

	latencyMetrics, err := metrics.NewLatency("vtx", "vertex/vertices", chainCtx.Log, "", chainCtx.AvalancheRegisterer)
	if err != nil {
		return err
	}
	ta.Latency = latencyMetrics

	ta.nodes = make(map[ids.ID]*transactionVertex, minMapSize)

	ta.cg = &snowstorm.Directed{}
	if err := ta.cg.Initialize(chainCtx, params.Parameters); err != nil {
		return err
	}

	ta.frontier = make(map[ids.ID]Vertex, minMapSize)
	for _, vtx := range frontier {
		ta.frontier[vtx.ID()] = vtx
	}
	return ta.updateFrontiers(ctx)
}

func (ta *Topological) NumProcessing() int {
	return len(ta.nodes)
}

func (ta *Topological) IsVirtuous(tx snowstorm.Tx) bool {
	return ta.cg.IsVirtuous(tx)
}

func (ta *Topological) Add(ctx context.Context, vtx Vertex) error {
	if vtx.Status().Decided() {
		return nil // Already decided this vertex
	}

	vtxID := vtx.ID()
	if _, exists := ta.nodes[vtxID]; exists {
		return nil // Already inserted this vertex
	}

	txs, err := vtx.Txs(ctx)
	if err != nil {
		return err
	}
	for _, tx := range txs {
		if !tx.Status().Decided() {
			// Add the consumers to the conflict graph.
			if err := ta.cg.Add(ctx, tx); err != nil {
				return err
			}

			// If the added transaction is virtuous, add it to the set of
			// virtuous transactions that are still being voted on.
			if vs := ta.cg.VirtuousVoting(); vs.Contains(tx.ID()) {
				ta.virtuousVoting.Add(tx.ID())
			}
		}
	}

	txv := newTransactionVertex(vtx, ta.nodes)

	// Add the transaction vertex to the set of processing nodes.
	ta.nodes[vtxID] = txv

	// Also add the transaction vertex to the conflict graph to track conflicts.
	if err := ta.cg.Add(ctx, txv); err != nil {
		return err
	}

	// If the added transaction vertex is virtuous, add it to the set of
	// virtuous transactions that are still being voted on. If the vertex isn't
	// virtuous, then the ID will be removed on the subsequent call to update.
	if vs := ta.cg.VirtuousVoting(); vs.Contains(vtxID) {
		ta.virtuousVoting.Add(vtxID)
	}

	ta.Latency.Issued(vtxID, ta.pollNumber)

	// Because we don't call [updateFrontiers], previous vertices that were
	// marked as virtuous will not be updated to no longer being virtuous. Even
	// if this newly added vertex conflicts with them. This is an optimization
	// to avoid a re-traversal of the DAG in the issuance path. Their virtuous
	// status will be updated during a future poll. This is safe because the
	// virtuous frontier is only used optimistically to control when it is valid
	// to quiesce.
	return ta.update(ctx, vtx) // Update the vertices preference and virtuous status
}

func (ta *Topological) VertexIssued(vtx Vertex) bool {
	if vtx.Status().Decided() {
		return true
	}
	_, ok := ta.nodes[vtx.ID()]
	return ok
}

func (ta *Topological) TxIssued(tx snowstorm.Tx) bool {
	return ta.cg.Issued(tx)
}

func (ta *Topological) Orphans() set.Set[ids.ID] {
	return ta.orphans
}

func (ta *Topological) Virtuous() set.Set[ids.ID] {
	return ta.virtuous
}

func (ta *Topological) Preferences() set.Set[ids.ID] {
	return ta.preferred
}

func (ta *Topological) RecordPoll(ctx context.Context, responses bag.UniqueBag[ids.ID]) error {
	// Register a new poll call
	ta.pollNumber++

	// If it isn't possible to have alpha votes for any transaction, then we can
	// just reset the confidence values in the conflict graph and not perform
	// any traversals.
	partialVotes := set.Bits64(0)
	for vote := range responses {
		votes := responses.GetSet(vote)
		partialVotes.Union(votes)
		if partialVotes.Len() >= ta.params.Alpha {
			break
		}
	}
	if partialVotes.Len() < ta.params.Alpha {
		// Because there were less than alpha total returned votes, we can skip
		// the traversals and fail the poll.
		_, err := ta.cg.RecordPoll(ctx, bag.Bag[ids.ID]{})
		return err
	}

	// Set up the topological sort: O(|Live Set|)
	if err := ta.calculateInDegree(responses); err != nil {
		return err
	}

	// Collect the votes for each transaction: O(|Live Set|)
	votes, err := ta.pushVotes(ctx)
	if err != nil {
		return err
	}

	// Update the conflict graph: O(|Transactions|)
	if updated, err := ta.cg.RecordPoll(ctx, votes); !updated || err != nil {
		// If the transaction statuses weren't changed, there is no need to
		// perform a traversal.
		return err
	}

	// Update the dag: O(|Live Set|)
	return ta.updateFrontiers(ctx)
}

func (ta *Topological) Quiesce() bool {
	return ta.virtuousVoting.Len() == 0
}

func (ta *Topological) Finalized() bool {
	return ta.cg.Finalized()
}

// HealthCheck returns information about the consensus health.
func (ta *Topological) HealthCheck(ctx context.Context) (interface{}, error) {
	numOutstandingVtx := ta.Latency.NumProcessing()
	isOutstandingVtx := numOutstandingVtx <= ta.params.MaxOutstandingItems
	healthy := isOutstandingVtx
	details := map[string]interface{}{
		"outstandingVertices": numOutstandingVtx,
	}

	// check for long running vertices
	oldestProcessingDuration := ta.Latency.MeasureAndGetOldestDuration()
	processingTimeOK := oldestProcessingDuration <= ta.params.MaxItemProcessingTime
	healthy = healthy && processingTimeOK
	details["longestRunningVertex"] = oldestProcessingDuration.String()

	snowstormReport, err := ta.cg.HealthCheck(ctx)
	healthy = healthy && err == nil
	details["snowstorm"] = snowstormReport

	if !healthy {
		var errorReasons []string
		if isOutstandingVtx {
			errorReasons = append(errorReasons, fmt.Sprintf("number outstanding vertexes %d > %d", numOutstandingVtx, ta.params.MaxOutstandingItems))
		}
		if !processingTimeOK {
			errorReasons = append(errorReasons, fmt.Sprintf("vertex processing time %s > %s", oldestProcessingDuration, ta.params.MaxItemProcessingTime))
		}
		if err != nil {
			errorReasons = append(errorReasons, err.Error())
		}
		return details, fmt.Errorf("avalanche consensus is not healthy reason: %s", strings.Join(errorReasons, ", "))
	}
	return details, nil
}

// Takes in a list of votes and sets up the topological ordering. Returns the
// reachable section of the graph annotated with the number of inbound edges and
// the non-transitively applied votes. Also returns the list of leaf nodes.
func (ta *Topological) calculateInDegree(responses bag.UniqueBag[ids.ID]) error {
	// Clear the kahn node set
	maps.Clear(ta.kahnNodes)
	// Clear the leaf set
	ta.leaves.Clear()

	for vote := range responses {
		// If it is not found, then the vote is either for something decided,
		// or something we haven't heard of yet.
		if tv := ta.nodes[vote]; tv != nil {
			vtx := tv.vtx

			kahn, previouslySeen := ta.kahnNodes[vote]
			// Add this new vote to the current bag of votes
			kahn.votes.Union(responses.GetSet(vote))
			ta.kahnNodes[vote] = kahn

			if !previouslySeen {
				// If I've never seen this node before, it is currently a leaf.
				ta.leaves.Add(vote)
				parents, err := vtx.Parents()
				if err != nil {
					return err
				}
				ta.kahnNodes, err = ta.markAncestorInDegrees(ta.kahnNodes, parents)
				if err != nil {
					return err
				}
			}
		}
	}
	return nil
}

// adds a new in-degree reference for all nodes.
// should only be called from [calculateInDegree]
func (ta *Topological) markAncestorInDegrees(
	kahns map[ids.ID]kahnNode,
	deps []Vertex,
) (map[ids.ID]kahnNode, error) {
	frontier := make([]Vertex, 0, len(deps))
	for _, vtx := range deps {
		// The vertex may have been decided, no need to vote in that case
		if !vtx.Status().Decided() {
			frontier = append(frontier, vtx)
		}
	}

	for len(frontier) > 0 {
		newLen := len(frontier) - 1
		current := frontier[newLen]
		frontier = frontier[:newLen]

		currentID := current.ID()
		kahn, alreadySeen := kahns[currentID]
		// I got here through a transitive edge, so increase the in-degree
		kahn.inDegree++
		kahns[currentID] = kahn

		if kahn.inDegree == 1 {
			// If I am transitively seeing this node for the first
			// time, it is no longer a leaf.
			ta.leaves.Remove(currentID)
		}

		if !alreadySeen {
			// If I am seeing this node for the first time, I need to check its
			// parents
			parents, err := current.Parents()
			if err != nil {
				return nil, err
			}
			for _, depVtx := range parents {
				// No need to traverse to a decided vertex
				if !depVtx.Status().Decided() {
					frontier = append(frontier, depVtx)
				}
			}
		}
	}
	return kahns, nil
}

// Count the number of votes for each operation by pushing votes upwards through
// vertex ancestors.
func (ta *Topological) pushVotes(ctx context.Context) (bag.Bag[ids.ID], error) {
	ta.votes.Clear()
	txConflicts := make(map[ids.ID]set.Set[ids.ID], minMapSize)

	// A leaf is a node with no inbound edges. This removes each leaf and pushes
	// the votes upwards, potentially creating new leaves, until there are no
	// more leaves.
	for ta.leaves.Len() > 0 {
		// Pop one node from [leaves]
		leaf, ok := ta.leaves.Pop()
		if !ok {
			// Should never happen because we just checked that [ta.leaves] is
			// not empty.
			return bag.Bag[ids.ID]{}, errNoLeaves
		}

		kahn := ta.kahnNodes[leaf]

		if tv := ta.nodes[leaf]; tv != nil {
			vtx := tv.vtx
			txs, err := vtx.Txs(ctx)
			if err != nil {
				return bag.Bag[ids.ID]{}, err
			}
			for _, tx := range txs {
				// Give the votes to the consumer
				txID := tx.ID()
				ta.votes.UnionSet(txID, kahn.votes)

				// Map txID to set of Conflicts
				if _, exists := txConflicts[txID]; !exists {
					txConflicts[txID] = ta.cg.Conflicts(tx)
				}
			}

			// The leaf is the ID of the transaction vertex that was issued to
			// the conflict graph for this vertex. Adding this vote is required
			// to make progress toward accepting this transaction.
			ta.votes.UnionSet(leaf, kahn.votes)

			// Map the vertexID to the set of conflicts from the transaction
			// vertex.
			if _, exists := txConflicts[leaf]; !exists {
				txConflicts[leaf] = ta.cg.Conflicts(tv)
			}

			parents, err := vtx.Parents()
			if err != nil {
				return bag.Bag[ids.ID]{}, err
			}
			for _, dep := range parents {
				depID := dep.ID()
				if depNode, notPruned := ta.kahnNodes[depID]; notPruned {
					depNode.inDegree--
					// Give the votes to my parents
					depNode.votes.Union(kahn.votes)
					ta.kahnNodes[depID] = depNode

					if depNode.inDegree == 0 {
						// Only traverse into the leaves
						ta.leaves.Add(depID)
					}
				}
			}
		}
	}

	// Create bag of votes for conflicting transactions
	conflictingVotes := make(bag.UniqueBag[ids.ID])
	for txID, conflicts := range txConflicts {
		for conflictTxID := range conflicts {
			conflictingVotes.UnionSet(txID, ta.votes.GetSet(conflictTxID))
		}
	}

	ta.votes.Difference(&conflictingVotes)
	return ta.votes.Bag(ta.params.Alpha), nil
}

// If I've already checked, do nothing
// If I'm decided, cache the preference and return
// At this point, I must be live
// I now try to accept all my consumers
// I now update all my ancestors
// If any of my parents are rejected, reject myself
// If I'm preferred, remove all my ancestors from the preferred frontier, add
// myself to the preferred frontier
// If all my parents are accepted and I'm acceptable, accept myself
func (ta *Topological) update(ctx context.Context, vtx Vertex) error {
	vtxID := vtx.ID()
	if _, cached := ta.preferenceCache[vtxID]; cached {
		return nil // This vertex has already been updated
	}

	// Drop all transaction vertices from the orphan set, as they can never be
	// reissued.
	ta.orphans.Remove(vtxID)

	// Note: it is not possible for the status to be rejected here. Update is
	// only called when adding a new processing vertex and when updating the
	// frontiers. If update is called with a rejected vertex when updating the
	// frontiers, it is guaranteed that the vertex was rejected during the same
	// frontier update. This means that the rejected vertex must have already
	// been visited, which means update will have exited from the above
	// preferenceCache check.
	if vtx.Status() == choices.Accepted {
		ta.preferred.Add(vtxID) // I'm preferred
		ta.virtuous.Add(vtxID)  // Accepted is defined as virtuous

		ta.frontier[vtxID] = vtx // I have no descendents yet

		ta.preferenceCache[vtxID] = true
		ta.virtuousCache[vtxID] = true
		return nil
	}

	txs, err := vtx.Txs(ctx)
	if err != nil {
		return err
	}
	preferences := ta.cg.Preferences()
	virtuousTxs := ta.cg.Virtuous()

	txv, ok := ta.nodes[vtxID]
	if !ok {
		return fmt.Errorf("transaction vertex %s not found in processing nodes set", vtxID)
	}

	initialTxVStatus := txv.Status()

	// acceptable tracks if all transactions included in the vertex have been
	// accepted and if all the parent vertices have been accepted. The
	// transactions include the transaction vertex.
	acceptable := initialTxVStatus == choices.Accepted

	// rejectable tracks if any of the transactions included in the vertex have
	// been rejected or if any of the parent vertices have been rejected. The
	// transactions include the transaction vertex.
	rejectable := initialTxVStatus == choices.Rejected

	preferred := acceptable || preferences.Contains(vtxID)
	virtuous := acceptable || virtuousTxs.Contains(vtxID)

	for _, tx := range txs {
		txID := tx.ID()
		s := tx.Status()
		if s == choices.Rejected {
			// If I contain a rejected consumer, I am rejectable
			rejectable = true
			preferred = false
			virtuous = false
		}
		if s != choices.Accepted {
			// If I contain a non-accepted consumer, I am not acceptable
			acceptable = false
			preferred = preferred && preferences.Contains(txID)
			virtuous = virtuous && virtuousTxs.Contains(txID)
		}
	}

	deps, err := vtx.Parents()
	if err != nil {
		return err
	}
	// Update all of my dependencies
	for _, dep := range deps {
		if err := ta.update(ctx, dep); err != nil {
			return err
		}

		depID := dep.ID()
		preferred = preferred && ta.preferenceCache[depID]
		virtuous = virtuous && ta.virtuousCache[depID]
	}

	// Check my parent statuses
	for _, dep := range deps {
		switch status := dep.Status(); status {
		case choices.Rejected:
			// My parent is rejected, so I should be rejected
			ta.ctx.Log.Trace("rejecting vertex",
				zap.String("reason", "rejected parent"),
				zap.Stringer("vtxID", vtxID),
				zap.Stringer("parentID", dep.ID()),
			)
			// Note: because the parent was rejected, the transaction vertex
			// will have already been marked as rejected by the conflict graph.
			// However, we still need to remove it from the set of virtuous
			// transactions.
			ta.virtuousVoting.Remove(vtxID)
			if err := vtx.Reject(ctx); err != nil {
				return err
			}
			delete(ta.nodes, vtxID)
			ta.Latency.Rejected(vtxID, ta.pollNumber, len(vtx.Bytes()))

			ta.preferenceCache[vtxID] = false
			ta.virtuousCache[vtxID] = false
			return nil
		case choices.Accepted:
			// If the dependency is accepted, then the vertex's acceptability
			// doesn't change.
		default:
			acceptable = false // A parent isn't accepted, so I can't be
		}
	}

	// Technically, we could also check to see if there are direct conflicts
	// between this vertex and a vertex in it's ancestry. If there does exist
	// such a conflict, this vertex could also be rejected. However, this would
	// require a traversal. Therefore, this memory optimization is ignored.
	// Also, this will only happen from a byzantine node issuing the vertex.
	// Therefore, this is very unlikely to actually be triggered in practice.

	// If the vertex is going to be rejected, it and all of its children are
	// going to be removed from the graph. This means that the parents may still
	// exist in the frontier. If the vertex is not rejectable, then it will
	// still be in the graph and the parents can not be part of the frontier.
	if !rejectable {
		for _, dep := range deps {
			delete(ta.frontier, dep.ID())
		}
		ta.frontier[vtxID] = vtx // I have no descendents yet
	}

	ta.preferenceCache[vtxID] = preferred
	ta.virtuousCache[vtxID] = virtuous

	if preferred {
		ta.preferred.Add(vtxID) // I'm preferred
		for _, dep := range deps {
			ta.preferred.Remove(dep.ID()) // My parents aren't part of the frontier
		}

		// Transactions are marked as orphans if they are virtuous, but not
		// contained in a preferred vertex. Since this vertex is preferred,
		// remove all the internal transactions from the orphan set.
		//
		// As an optimization, we only iterate over the transactions if the set
		// of orphans isn't empty. As if the set is empty, nothing will be able
		// to be removed anyway.
		if ta.orphans.Len() > 0 {
			for _, tx := range txs {
				if tx.Status() != choices.Accepted {
					ta.orphans.Remove(tx.ID())
				}
			}
		}
	}

	if virtuous {
		ta.virtuous.Add(vtxID) // I'm virtuous
		for _, dep := range deps {
			ta.virtuous.Remove(dep.ID()) // My parents aren't part of the frontier
		}
	} else {
		// If the vertex isn't virtuous, then we can remove the transaction
		// vertex from the voting set to pessemistically quiesce early.
		ta.virtuousVoting.Remove(vtxID)
	}

	switch {
	case acceptable:
		// I'm acceptable, why not accept?
		// Note that VertexAcceptor.Accept must be called before vtx.Accept to
		// honor Acceptor.Accept's invariant.
		vtxBytes := vtx.Bytes()
		if err := ta.ctx.VertexAcceptor.Accept(ta.ctx, vtxID, vtxBytes); err != nil {
			return err
		}

		if err := vtx.Accept(ctx); err != nil {
			return err
		}
		delete(ta.nodes, vtxID)
		ta.Latency.Accepted(vtxID, ta.pollNumber, len(vtxBytes))
	case rejectable:
		// I'm rejectable, why not reject?
		ta.ctx.Log.Trace("rejecting vertex",
			zap.String("reason", "conflicting acceptance"),
			zap.Stringer("vtxID", vtxID),
		)
		if !txv.Status().Decided() {
			if err := ta.cg.Remove(ctx, vtxID); err != nil {
				return fmt.Errorf("failed to remove transaction vertex %s from snowstorm before rejecting vertex itself", vtxID)
			}
			ta.virtuousVoting.Remove(vtxID)
		}
		if err := vtx.Reject(ctx); err != nil {
			return err
		}
		delete(ta.nodes, vtxID)
		ta.Latency.Rejected(vtxID, ta.pollNumber, len(vtx.Bytes()))
	}
	return nil
}

// Update the frontier sets
func (ta *Topological) updateFrontiers(ctx context.Context) error {
	vts := ta.frontier

	ta.preferred.Clear()
	ta.virtuous.Clear()
	ta.virtuousVoting.Clear()

	ta.orphans.Clear()
	ta.frontier = make(map[ids.ID]Vertex, minMapSize)
	ta.preferenceCache = make(map[ids.ID]bool, minMapSize)
	ta.virtuousCache = make(map[ids.ID]bool, minMapSize)

	ta.virtuousVoting.Union(ta.cg.VirtuousVoting())
	ta.orphans.Union(ta.cg.Virtuous()) // Initially, nothing is preferred

	for _, vtx := range vts {
		// Update all the vertices that were in my previous frontier
		if err := ta.update(ctx, vtx); err != nil {
			return err
		}
	}
	return nil
}
