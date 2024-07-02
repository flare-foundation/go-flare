// Copyright (C) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package router

import (
	"encoding/binary"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"go.uber.org/zap"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/message"
	"github.com/ava-labs/avalanchego/snow/networking/benchlist"
	"github.com/ava-labs/avalanchego/snow/networking/handler"
	"github.com/ava-labs/avalanchego/snow/networking/timeout"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/hashing"
	"github.com/ava-labs/avalanchego/utils/linkedhashmap"
	"github.com/ava-labs/avalanchego/utils/logging"
	"github.com/ava-labs/avalanchego/utils/timer/mockable"
	"github.com/ava-labs/avalanchego/utils/wrappers"
	"github.com/ava-labs/avalanchego/version"
)

var (
	errUnknownChain = errors.New("received message for unknown chain")

	_ Router              = &ChainRouter{}
	_ benchlist.Benchable = &ChainRouter{}
)

type requestEntry struct {
	// When this request was registered
	time time.Time
	// The type of request that was made
	op message.Op
}

type peer struct {
	version        *version.Application
	trackedSubnets ids.Set
}

// ChainRouter routes incoming messages from the validator network
// to the consensus engines that the messages are intended for.
// Note that consensus engines are uniquely identified by the ID of the chain
// that they are working on.
type ChainRouter struct {
	clock      mockable.Clock
	log        logging.Logger
	msgCreator message.InternalMsgBuilder
	lock       sync.Mutex
	chains     map[ids.ID]handler.Handler

	// It is only safe to call [RegisterResponse] with the router lock held. Any
	// other calls to the timeout manager with the router lock held could cause
	// a deadlock because the timeout manager will call Benched and Unbenched.
	timeoutManager timeout.Manager

	closeTimeout time.Duration
	peers        map[ids.NodeID]*peer
	// node ID --> chains that node is benched on
	// invariant: if a node is benched on any chain, it is treated as disconnected on all chains
	benched        map[ids.NodeID]ids.Set
	criticalChains ids.Set
	onFatal        func(exitCode int)
	metrics        *routerMetrics
	// Parameters for doing health checks
	healthConfig HealthConfig
	// aggregator of requests based on their time
	timedRequests linkedhashmap.LinkedHashmap[ids.ID, requestEntry]
	// Must only be accessed in method [createRequestID].
	// [lock] must be held when [requestIDBytes] is accessed.
	requestIDBytes []byte
}

// Initialize the router.
//
// When this router receives an incoming message, it cancels the timeout in
// [timeouts] associated with the request that caused the incoming message, if
// applicable.
func (cr *ChainRouter) Initialize(
	nodeID ids.NodeID,
	log logging.Logger,
	msgCreator message.InternalMsgBuilder,
	timeoutManager timeout.Manager,
	closeTimeout time.Duration,
	criticalChains ids.Set,
	whitelistedSubnets ids.Set,
	onFatal func(exitCode int),
	healthConfig HealthConfig,
	metricsNamespace string,
	metricsRegisterer prometheus.Registerer,
) error {
	cr.log = log
	cr.msgCreator = msgCreator
	cr.chains = make(map[ids.ID]handler.Handler)
	cr.timeoutManager = timeoutManager
	cr.closeTimeout = closeTimeout
	cr.benched = make(map[ids.NodeID]ids.Set)
	cr.criticalChains = criticalChains
	cr.onFatal = onFatal
	cr.timedRequests = linkedhashmap.New[ids.ID, requestEntry]()
	cr.peers = make(map[ids.NodeID]*peer)
	cr.healthConfig = healthConfig
	cr.requestIDBytes = make([]byte, hashing.AddrLen+hashing.HashLen+wrappers.IntLen+wrappers.ByteLen) // Validator ID, Chain ID, Request ID, Msg Type

	// Mark myself as connected
	myself := &peer{
		version: version.CurrentApp,
	}
	myself.trackedSubnets.Union(whitelistedSubnets)
	myself.trackedSubnets.Add(constants.PrimaryNetworkID)
	cr.peers[nodeID] = myself

	// Register metrics
	rMetrics, err := newRouterMetrics(metricsNamespace, metricsRegisterer)
	if err != nil {
		return err
	}
	cr.metrics = rMetrics
	return nil
}

// RegisterRequest marks that we should expect to receive a reply from the given
// validator regarding the given chain and the reply should have the given
// requestID.
// The type of message we expect is [op].
// Every registered request must be cleared either by receiving a valid reply
// and passing it to the appropriate chain or by a timeout.
// This method registers a timeout that calls such methods if we don't get a
// reply in time.
func (cr *ChainRouter) RegisterRequest(
	nodeID ids.NodeID,
	chainID ids.ID,
	requestID uint32,
	op message.Op,
) {
	cr.lock.Lock()
	// When we receive a response message type (Chits, Put, Accepted, etc.)
	// we validate that we actually sent the corresponding request.
	// Give this request a unique ID so we can do that validation.
	uniqueRequestID := cr.createRequestID(nodeID, chainID, requestID, op)
	// Add to the set of unfulfilled requests
	cr.timedRequests.Put(uniqueRequestID, requestEntry{
		time: cr.clock.Time(),
		op:   op,
	})
	cr.metrics.outstandingRequests.Set(float64(cr.timedRequests.Len()))
	cr.lock.Unlock()

	failedOp, exists := message.ResponseToFailedOps[op]
	if !exists {
		// This should never happen
		cr.log.Error("failed to convert message operation",
			zap.Stringer("messageOp", op),
		)
		return
	}

	// Register a timeout to fire if we don't get a reply in time.
	cr.timeoutManager.RegisterRequest(nodeID, chainID, op, uniqueRequestID, func() {
		msg := cr.msgCreator.InternalFailedRequest(failedOp, nodeID, chainID, requestID)
		cr.HandleInbound(msg)
	})
}

func (cr *ChainRouter) HandleInbound(msg message.InboundMessage) {
	nodeID := msg.NodeID()
	op := msg.Op()

	chainIDIntf, err := msg.Get(message.ChainID)
	if err != nil {
		cr.log.Debug("dropping message with invalid field",
			zap.Stringer("nodeID", nodeID),
			zap.Stringer("messageOp", op),
			zap.Stringer("field", message.ChainID),
			zap.Error(err),
		)

		msg.OnFinishedHandling()
		return
	}
	chainIDBytes := chainIDIntf.([]byte)
	chainID, err := ids.ToID(chainIDBytes)
	if err != nil {
		cr.log.Debug("dropping message with invalid field",
			zap.Stringer("nodeID", nodeID),
			zap.Stringer("messageOp", op),
			zap.Stringer("field", message.ChainID),
			zap.Error(err),
		)

		msg.OnFinishedHandling()
		return
	}

	// AppGossip is the only message currently not containing a requestID
	// Here we assign the requestID already in use for gossiped containers
	// to allow a uniform handling of all messages
	var requestID uint32
	if op == message.AppGossip {
		requestID = constants.GossipMsgRequestID
	} else {
		// Invariant: Getting a [RequestID] must never error in the handler. Any
		//            verification performed by the message is done here.
		requestIDIntf, err := msg.Get(message.RequestID)
		if err != nil {
			cr.log.Debug("dropping message with invalid field",
				zap.Stringer("nodeID", nodeID),
				zap.Stringer("messageOp", op),
				zap.Stringer("field", message.RequestID),
				zap.Error(err),
			)

			msg.OnFinishedHandling()
			return
		}
		requestID = requestIDIntf.(uint32)
	}

	cr.lock.Lock()
	defer cr.lock.Unlock()

	// Get the chain, if it exists
	chain, exists := cr.chains[chainID]
	if !exists || !chain.IsValidator(nodeID) {
		cr.log.Debug("dropping message",
			zap.Stringer("messageOp", op),
			zap.Stringer("nodeID", nodeID),
			zap.Stringer("chainID", chainID),
			zap.Error(errUnknownChain),
		)

		msg.OnFinishedHandling()
		return
	}

	ctx := chain.Context()

	// TODO: [requestID] can overflow, which means a timeout on the request
	//       before the overflow may not be handled properly.
	if _, notRequested := message.UnrequestedOps[op]; notRequested ||
		(op == message.Put && requestID == constants.GossipMsgRequestID) {
		if ctx.IsExecuting() {
			cr.log.Debug("dropping message and skipping queue",
				zap.String("reason", "the chain is currently executing"),
				zap.Stringer("messageOp", op),
			)
			cr.metrics.droppedRequests.Inc()

			msg.OnFinishedHandling()
			return
		}
		chain.Push(msg)
		return
	}

	if expectedResponse, isFailed := message.FailedToResponseOps[op]; isFailed {
		// Create the request ID of the request we sent that this message is in
		// response to.
		uniqueRequestID, req := cr.clearRequest(expectedResponse, nodeID, chainID, requestID)
		if req == nil {
			// This was a duplicated response.
			msg.OnFinishedHandling()
			return
		}

		// Tell the timeout manager we are no longer expecting a response
		cr.timeoutManager.RemoveRequest(uniqueRequestID)

		// Pass the failure to the chain
		chain.Push(msg)
		return
	}

	if ctx.IsExecuting() {
		cr.log.Debug("dropping message and skipping queue",
			zap.String("reason", "the chain is currently executing"),
			zap.Stringer("messageOp", op),
		)
		cr.metrics.droppedRequests.Inc()

		msg.OnFinishedHandling()
		return
	}

	uniqueRequestID, req := cr.clearRequest(op, nodeID, chainID, requestID)
	if req == nil {
		// We didn't request this message.
		msg.OnFinishedHandling()
		return
	}

	// Calculate how long it took [nodeID] to reply
	latency := cr.clock.Time().Sub(req.time)

	// Tell the timeout manager we got a response
	cr.timeoutManager.RegisterResponse(nodeID, chainID, uniqueRequestID, req.op, latency)

	// Pass the response to the chain
	chain.Push(msg)
}

// Shutdown shuts down this router
func (cr *ChainRouter) Shutdown() {
	cr.log.Info("shutting down chain router")
	cr.lock.Lock()
	prevChains := cr.chains
	cr.chains = map[ids.ID]handler.Handler{}
	cr.lock.Unlock()

	for _, chain := range prevChains {
		chain.Stop()
	}

	ticker := time.NewTicker(cr.closeTimeout)
	defer ticker.Stop()

	for _, chain := range prevChains {
		select {
		case <-chain.Stopped():
		case <-ticker.C:
			cr.log.Warn("timed out while shutting down the chains")
			return
		}
	}
}

// AddChain registers the specified chain so that incoming
// messages can be routed to it
func (cr *ChainRouter) AddChain(chain handler.Handler) {
	cr.lock.Lock()
	defer cr.lock.Unlock()

	chainID := chain.Context().ChainID
	cr.log.Debug("registering chain with chain router",
		zap.Stringer("chainID", chainID),
	)
	chain.SetOnStopped(func() {
		cr.removeChain(chainID)
	})
	cr.chains[chainID] = chain

	// Notify connected validators
	subnetID := chain.Context().SubnetID
	for validatorID, peer := range cr.peers {
		// If this validator is benched on any chain, treat them as disconnected on all chains
		if _, benched := cr.benched[validatorID]; !benched && peer.trackedSubnets.Contains(subnetID) {
			msg := cr.msgCreator.InternalConnected(validatorID, peer.version)
			chain.Push(msg)
		}
	}
}

// Connected routes an incoming notification that a validator was just connected
func (cr *ChainRouter) Connected(nodeID ids.NodeID, nodeVersion *version.Application, subnetID ids.ID) {
	cr.lock.Lock()
	defer cr.lock.Unlock()

	connectedPeer, exists := cr.peers[nodeID]
	if !exists {
		connectedPeer = &peer{
			version: nodeVersion,
		}
		cr.peers[nodeID] = connectedPeer
	}
	connectedPeer.trackedSubnets.Add(subnetID)

	// If this validator is benched on any chain, treat them as disconnected on all chains
	if _, benched := cr.benched[nodeID]; benched {
		return
	}

	msg := cr.msgCreator.InternalConnected(nodeID, nodeVersion)

	// TODO: fire up an event when validator state changes i.e when they leave set, disconnect.
	// we cannot put a subnet-only validator check here since Disconnected would not be handled properly.
	for _, chain := range cr.chains {
		if subnetID == chain.Context().SubnetID {
			chain.Push(msg)
		}
	}
}

// Disconnected routes an incoming notification that a validator was connected
func (cr *ChainRouter) Disconnected(nodeID ids.NodeID) {
	cr.lock.Lock()
	defer cr.lock.Unlock()

	peer := cr.peers[nodeID]
	delete(cr.peers, nodeID)
	if _, benched := cr.benched[nodeID]; benched {
		return
	}

	msg := cr.msgCreator.InternalDisconnected(nodeID)

	// TODO: fire up an event when validator state changes i.e when they leave set, disconnect.
	// we cannot put a subnet-only validator check here since if a validator connects then it leaves validator-set, it would not be disconnected properly.
	for _, chain := range cr.chains {
		if peer.trackedSubnets.Contains(chain.Context().SubnetID) {
			chain.Push(msg)
		}
	}
}

// Benched routes an incoming notification that a validator was benched
func (cr *ChainRouter) Benched(chainID ids.ID, nodeID ids.NodeID) {
	cr.lock.Lock()
	defer cr.lock.Unlock()

	benchedChains, exists := cr.benched[nodeID]
	benchedChains.Add(chainID)
	cr.benched[nodeID] = benchedChains
	peer, hasPeer := cr.peers[nodeID]
	if exists || !hasPeer {
		// If the set already existed, then the node was previously benched.
		return
	}

	msg := cr.msgCreator.InternalDisconnected(nodeID)

	for _, chain := range cr.chains {
		if peer.trackedSubnets.Contains(chain.Context().SubnetID) {
			chain.Push(msg)
		}
	}
}

// Unbenched routes an incoming notification that a validator was just unbenched
func (cr *ChainRouter) Unbenched(chainID ids.ID, nodeID ids.NodeID) {
	cr.lock.Lock()
	defer cr.lock.Unlock()

	benchedChains := cr.benched[nodeID]
	benchedChains.Remove(chainID)
	if benchedChains.Len() == 0 {
		delete(cr.benched, nodeID)
	} else {
		cr.benched[nodeID] = benchedChains
		return // This node is still benched
	}

	peer, found := cr.peers[nodeID]
	if !found {
		return
	}

	msg := cr.msgCreator.InternalConnected(nodeID, peer.version)

	for _, chain := range cr.chains {
		if peer.trackedSubnets.Contains(chain.Context().SubnetID) {
			chain.Push(msg)
		}
	}
}

// HealthCheck returns results of router health checks. Returns:
// 1) Information about health check results
// 2) An error if the health check reports unhealthy
func (cr *ChainRouter) HealthCheck() (interface{}, error) {
	cr.lock.Lock()
	defer cr.lock.Unlock()

	numOutstandingReqs := cr.timedRequests.Len()
	isOutstandingReqs := numOutstandingReqs <= cr.healthConfig.MaxOutstandingRequests
	healthy := isOutstandingReqs
	details := map[string]interface{}{
		"outstandingRequests": numOutstandingReqs,
	}

	// check for long running requests
	now := cr.clock.Time()
	processingRequest := now
	if _, longestRunning, exists := cr.timedRequests.Oldest(); exists {
		processingRequest = longestRunning.time
	}
	timeReqRunning := now.Sub(processingRequest)
	isOutstanding := timeReqRunning <= cr.healthConfig.MaxOutstandingDuration
	healthy = healthy && isOutstanding
	details["longestRunningRequest"] = timeReqRunning.String()
	cr.metrics.longestRunningRequest.Set(float64(timeReqRunning))

	if !healthy {
		var errorReasons []string
		if !isOutstandingReqs {
			errorReasons = append(errorReasons, fmt.Sprintf("number of outstanding requests %d > %d", numOutstandingReqs, cr.healthConfig.MaxOutstandingRequests))
		}
		if !isOutstanding {
			errorReasons = append(errorReasons, fmt.Sprintf("time for outstanding requests %s > %s", timeReqRunning, cr.healthConfig.MaxOutstandingDuration))
		}
		// The router is not healthy
		return details, fmt.Errorf("the router is not healthy reason: %s", strings.Join(errorReasons, ", "))
	}
	return details, nil
}

// RemoveChain removes the specified chain so that incoming
// messages can't be routed to it
func (cr *ChainRouter) removeChain(chainID ids.ID) {
	cr.lock.Lock()
	chain, exists := cr.chains[chainID]
	if !exists {
		cr.log.Debug("can't remove unknown chain",
			zap.Stringer("chainID", chainID),
		)
		cr.lock.Unlock()
		return
	}
	delete(cr.chains, chainID)
	cr.lock.Unlock()

	chain.Stop()

	ticker := time.NewTicker(cr.closeTimeout)
	defer ticker.Stop()
	select {
	case <-chain.Stopped():
	case <-ticker.C:
		chain.Context().Log.Warn("timed out while shutting down")
	}

	if cr.onFatal != nil && cr.criticalChains.Contains(chainID) {
		go cr.onFatal(1)
	}
}

func (cr *ChainRouter) clearRequest(
	op message.Op,
	nodeID ids.NodeID,
	chainID ids.ID,
	requestID uint32,
) (ids.ID, *requestEntry) {
	// Create the request ID of the request we sent that this message is (allegedly) in response to.
	uniqueRequestID := cr.createRequestID(nodeID, chainID, requestID, op)
	// Mark that an outstanding request has been fulfilled
	request, exists := cr.timedRequests.Get(uniqueRequestID)
	if !exists {
		return uniqueRequestID, nil
	}

	cr.timedRequests.Delete(uniqueRequestID)
	cr.metrics.outstandingRequests.Set(float64(cr.timedRequests.Len()))
	return uniqueRequestID, &request
}

// Assumes [cr.lock] is held.
// Assumes [message.Op] is an alias of byte.
func (cr *ChainRouter) createRequestID(nodeID ids.NodeID, chainID ids.ID, requestID uint32, op message.Op) ids.ID {
	copy(cr.requestIDBytes, nodeID[:])
	copy(cr.requestIDBytes[hashing.AddrLen:], chainID[:])
	binary.BigEndian.PutUint32(cr.requestIDBytes[hashing.AddrLen+hashing.HashLen:], requestID)
	cr.requestIDBytes[hashing.AddrLen+hashing.HashLen+wrappers.IntLen] = byte(op)
	return hashing.ComputeHash256Array(cr.requestIDBytes)
}
