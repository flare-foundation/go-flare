// Copyright (C) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package sender

import (
	"fmt"
	"time"

	"github.com/prometheus/client_golang/prometheus"

	"go.uber.org/zap"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/message"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/snow/engine/common"
	"github.com/ava-labs/avalanchego/snow/networking/router"
	"github.com/ava-labs/avalanchego/snow/networking/timeout"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/timer/mockable"
)

var _ common.Sender = &sender{}

type GossipConfig struct {
	AcceptedFrontierValidatorSize    uint `json:"gossipAcceptedFrontierValidatorSize" yaml:"gossipAcceptedFrontierValidatorSize"`
	AcceptedFrontierNonValidatorSize uint `json:"gossipAcceptedFrontierNonValidatorSize" yaml:"gossipAcceptedFrontierNonValidatorSize"`
	AcceptedFrontierPeerSize         uint `json:"gossipAcceptedFrontierPeerSize" yaml:"gossipAcceptedFrontierPeerSize"`
	OnAcceptValidatorSize            uint `json:"gossipOnAcceptValidatorSize" yaml:"gossipOnAcceptValidatorSize"`
	OnAcceptNonValidatorSize         uint `json:"gossipOnAcceptNonValidatorSize" yaml:"gossipOnAcceptNonValidatorSize"`
	OnAcceptPeerSize                 uint `json:"gossipOnAcceptPeerSize" yaml:"gossipOnAcceptPeerSize"`
	AppGossipValidatorSize           uint `json:"appGossipValidatorSize" yaml:"appGossipValidatorSize"`
	AppGossipNonValidatorSize        uint `json:"appGossipNonValidatorSize" yaml:"appGossipNonValidatorSize"`
	AppGossipPeerSize                uint `json:"appGossipPeerSize" yaml:"appGossipPeerSize"`
}

// sender is a wrapper around an ExternalSender.
// Messages to this node are put directly into [router] rather than
// being sent over the network via the wrapped ExternalSender.
// sender registers outbound requests with [router] so that [router]
// fires a timeout if we don't get a response to the request.
type sender struct {
	ctx                 *snow.ConsensusContext
	msgCreator          message.Creator
	msgCreatorWithProto message.Creator

	// TODO: remove this once we complete banff migration
	banffTime time.Time

	clock mockable.Clock

	sender   ExternalSender // Actually does the sending over the network
	router   router.Router
	timeouts timeout.Manager

	gossipConfig GossipConfig

	// Request message type --> Counts how many of that request
	// have failed because the node was benched
	failedDueToBench map[message.Op]prometheus.Counter
}

func New(
	ctx *snow.ConsensusContext,
	msgCreator message.Creator,
	msgCreatorWithProto message.Creator,
	banffTime time.Time,
	externalSender ExternalSender,
	router router.Router,
	timeouts timeout.Manager,
	gossipConfig GossipConfig,
) (common.Sender, error) {
	s := &sender{
		ctx:                 ctx,
		msgCreator:          msgCreator,
		msgCreatorWithProto: msgCreatorWithProto,
		banffTime:           banffTime,
		sender:              externalSender,
		router:              router,
		timeouts:            timeouts,
		gossipConfig:        gossipConfig,
		failedDueToBench:    make(map[message.Op]prometheus.Counter, len(message.ConsensusRequestOps)),
	}

	for _, op := range message.ConsensusRequestOps {
		counter := prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: fmt.Sprintf("%s_failed_benched", op),
				Help: fmt.Sprintf("# of times a %s request was not sent because the node was benched", op),
			},
		)
		if err := ctx.Registerer.Register(counter); err != nil {
			return nil, fmt.Errorf("couldn't register metric for %s: %w", op, err)
		}
		s.failedDueToBench[op] = counter
	}
	return s, nil
}

func (s *sender) getMsgCreator() message.Creator {
	now := s.clock.Time()
	if now.Before(s.banffTime) {
		return s.msgCreator
	}
	return s.msgCreatorWithProto
}

func (s *sender) SendGetStateSummaryFrontier(nodeIDs ids.NodeIDSet, requestID uint32) {
	// Note that this timeout duration won't exactly match the one that gets
	// registered. That's OK.
	deadline := s.timeouts.TimeoutDuration()

	// Tell the router to expect a response message or a message notifying
	// that we won't get a response from each of these nodes.
	// We register timeouts for all nodes, regardless of whether we fail
	// to send them a message, to avoid busy looping when disconnected from
	// the internet.
	for nodeID := range nodeIDs {
		s.router.RegisterRequest(nodeID, s.ctx.ChainID, requestID, message.StateSummaryFrontier)
	}

	msgCreator := s.getMsgCreator()

	// Sending a message to myself. No need to send it over the network.
	// Just put it right into the router. Asynchronously to avoid deadlock.
	if nodeIDs.Contains(s.ctx.NodeID) {
		nodeIDs.Remove(s.ctx.NodeID)
		inMsg := msgCreator.InboundGetStateSummaryFrontier(s.ctx.ChainID, requestID, deadline, s.ctx.NodeID)
		go s.router.HandleInbound(inMsg)
	}

	// Create the outbound message.
	outMsg, err := msgCreator.GetStateSummaryFrontier(s.ctx.ChainID, requestID, deadline)

	// Send the message over the network.
	var sentTo ids.NodeIDSet
	if err == nil {
		sentTo = s.sender.Send(outMsg, nodeIDs, s.ctx.SubnetID, s.ctx.IsValidatorOnly())
	} else {
		s.ctx.Log.Error("failed to build message",
			zap.Stringer("messageOp", message.GetStateSummaryFrontier),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Uint32("requestID", requestID),
			zap.Duration("deadline", deadline),
			zap.Error(err),
		)
	}

	for nodeID := range nodeIDs {
		if !sentTo.Contains(nodeID) {
			s.ctx.Log.Debug("failed to send message",
				zap.Stringer("messageOp", message.GetStateSummaryFrontier),
				zap.Stringer("nodeID", nodeID),
				zap.Stringer("chainID", s.ctx.ChainID),
				zap.Uint32("requestID", requestID),
			)
		}
	}
}

func (s *sender) SendStateSummaryFrontier(nodeID ids.NodeID, requestID uint32, summary []byte) {
	msgCreator := s.getMsgCreator()

	// Sending this message to myself.
	if nodeID == s.ctx.NodeID {
		inMsg := msgCreator.InboundStateSummaryFrontier(s.ctx.ChainID, requestID, summary, nodeID)
		go s.router.HandleInbound(inMsg)
		return
	}

	// Create the outbound message.
	outMsg, err := msgCreator.StateSummaryFrontier(s.ctx.ChainID, requestID, summary)
	if err != nil {
		s.ctx.Log.Error("failed to build message",
			zap.Stringer("messageOp", message.StateSummaryFrontier),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Uint32("requestID", requestID),
			zap.Binary("summaryBytes", summary),
			zap.Error(err),
		)
		return
	}

	// Send the message over the network.
	nodeIDs := ids.NewNodeIDSet(1)
	nodeIDs.Add(nodeID)
	if sentTo := s.sender.Send(outMsg, nodeIDs, s.ctx.SubnetID, s.ctx.IsValidatorOnly()); sentTo.Len() == 0 {
		s.ctx.Log.Debug("failed to send message",
			zap.Stringer("messageOp", message.StateSummaryFrontier),
			zap.Stringer("nodeID", nodeID),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Uint32("requestID", requestID),
		)
		s.ctx.Log.Verbo("failed to send message",
			zap.Stringer("messageOp", message.StateSummaryFrontier),
			zap.Stringer("nodeID", nodeID),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Uint32("requestID", requestID),
			zap.Binary("summary", summary),
		)
	}
}

func (s *sender) SendGetAcceptedStateSummary(nodeIDs ids.NodeIDSet, requestID uint32, heights []uint64) {
	// Note that this timeout duration won't exactly match the one that gets
	// registered. That's OK.
	deadline := s.timeouts.TimeoutDuration()

	// Tell the router to expect a response message or a message notifying
	// that we won't get a response from each of these nodes.
	// We register timeouts for all nodes, regardless of whether we fail
	// to send them a message, to avoid busy looping when disconnected from
	// the internet.
	for nodeID := range nodeIDs {
		s.router.RegisterRequest(nodeID, s.ctx.ChainID, requestID, message.AcceptedStateSummary)
	}

	msgCreator := s.getMsgCreator()

	// Sending a message to myself. No need to send it over the network.
	// Just put it right into the router. Asynchronously to avoid deadlock.
	if nodeIDs.Contains(s.ctx.NodeID) {
		nodeIDs.Remove(s.ctx.NodeID)
		inMsg := msgCreator.InboundGetAcceptedStateSummary(s.ctx.ChainID, requestID, heights, deadline, s.ctx.NodeID)
		go s.router.HandleInbound(inMsg)
	}

	// Create the outbound message.
	outMsg, err := msgCreator.GetAcceptedStateSummary(s.ctx.ChainID, requestID, deadline, heights)

	// Send the message over the network.
	var sentTo ids.NodeIDSet
	if err == nil {
		sentTo = s.sender.Send(outMsg, nodeIDs, s.ctx.SubnetID, s.ctx.IsValidatorOnly())
	} else {
		s.ctx.Log.Error("failed to build message",
			zap.Stringer("messageOp", message.GetAcceptedStateSummary),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Uint32("requestID", requestID),
			zap.Uint64s("heights", heights),
			zap.Error(err),
		)
	}

	for nodeID := range nodeIDs {
		if !sentTo.Contains(nodeID) {
			s.ctx.Log.Debug("failed to send message",
				zap.Stringer("messageOp", message.GetAcceptedStateSummary),
				zap.Stringer("nodeID", nodeID),
				zap.Stringer("chainID", s.ctx.ChainID),
				zap.Uint32("requestID", requestID),
				zap.Uint64s("heights", heights),
			)
		}
	}
}

func (s *sender) SendAcceptedStateSummary(nodeID ids.NodeID, requestID uint32, summaryIDs []ids.ID) {
	msgCreator := s.getMsgCreator()

	if nodeID == s.ctx.NodeID {
		inMsg := msgCreator.InboundAcceptedStateSummary(s.ctx.ChainID, requestID, summaryIDs, nodeID)
		go s.router.HandleInbound(inMsg)
		return
	}

	// Create the outbound message.
	outMsg, err := msgCreator.AcceptedStateSummary(s.ctx.ChainID, requestID, summaryIDs)
	if err != nil {
		s.ctx.Log.Error("failed to build message",
			zap.Stringer("messageOp", message.AcceptedStateSummary),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Uint32("requestID", requestID),
			zap.Stringer("summaryIDs", ids.SliceStringer(summaryIDs)),
			zap.Error(err),
		)
		return
	}

	// Send the message over the network.
	nodeIDs := ids.NewNodeIDSet(1)
	nodeIDs.Add(nodeID)
	if sentTo := s.sender.Send(outMsg, nodeIDs, s.ctx.SubnetID, s.ctx.IsValidatorOnly()); sentTo.Len() == 0 {
		s.ctx.Log.Debug("failed to send message",
			zap.Stringer("messageOp", message.AcceptedStateSummary),
			zap.Stringer("nodeID", nodeID),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Uint32("requestID", requestID),
			zap.Stringer("summaryIDs", ids.SliceStringer(summaryIDs)),
		)
	}
}

func (s *sender) SendGetAcceptedFrontier(nodeIDs ids.NodeIDSet, requestID uint32) {
	// Note that this timeout duration won't exactly match the one that gets
	// registered. That's OK.
	deadline := s.timeouts.TimeoutDuration()

	// Tell the router to expect a response message or a message notifying
	// that we won't get a response from each of these nodes.
	// We register timeouts for all nodes, regardless of whether we fail
	// to send them a message, to avoid busy looping when disconnected from
	// the internet.
	for nodeID := range nodeIDs {
		s.router.RegisterRequest(nodeID, s.ctx.ChainID, requestID, message.AcceptedFrontier)
	}

	msgCreator := s.getMsgCreator()

	// Sending a message to myself. No need to send it over the network.
	// Just put it right into the router. Asynchronously to avoid deadlock.
	if nodeIDs.Contains(s.ctx.NodeID) {
		nodeIDs.Remove(s.ctx.NodeID)
		inMsg := msgCreator.InboundGetAcceptedFrontier(s.ctx.ChainID, requestID, deadline, s.ctx.NodeID)
		go s.router.HandleInbound(inMsg)
	}

	// Create the outbound message.
	outMsg, err := msgCreator.GetAcceptedFrontier(s.ctx.ChainID, requestID, deadline)

	// Send the message over the network.
	var sentTo ids.NodeIDSet
	if err == nil {
		sentTo = s.sender.Send(outMsg, nodeIDs, s.ctx.SubnetID, s.ctx.IsValidatorOnly())
	} else {
		s.ctx.Log.Error("failed to build message",
			zap.Stringer("messageOp", message.GetAcceptedFrontier),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Uint32("requestID", requestID),
			zap.Duration("deadline", deadline),
			zap.Error(err),
		)
	}

	for nodeID := range nodeIDs {
		if !sentTo.Contains(nodeID) {
			s.ctx.Log.Debug("failed to send message",
				zap.Stringer("messageOp", message.GetAcceptedFrontier),
				zap.Stringer("nodeID", nodeID),
				zap.Stringer("chainID", s.ctx.ChainID),
				zap.Uint32("requestID", requestID),
			)
		}
	}
}

func (s *sender) SendAcceptedFrontier(nodeID ids.NodeID, requestID uint32, containerIDs []ids.ID) {
	msgCreator := s.getMsgCreator()

	// Sending this message to myself.
	if nodeID == s.ctx.NodeID {
		inMsg := msgCreator.InboundAcceptedFrontier(s.ctx.ChainID, requestID, containerIDs, nodeID)
		go s.router.HandleInbound(inMsg)
		return
	}

	// Create the outbound message.
	outMsg, err := msgCreator.AcceptedFrontier(s.ctx.ChainID, requestID, containerIDs)
	if err != nil {
		s.ctx.Log.Error("failed to build message",
			zap.Stringer("messageOp", message.AcceptedFrontier),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Uint32("requestID", requestID),
			zap.Stringer("containerIDs", ids.SliceStringer(containerIDs)),
			zap.Error(err),
		)
		return
	}

	// Send the message over the network.
	nodeIDs := ids.NewNodeIDSet(1)
	nodeIDs.Add(nodeID)
	if sentTo := s.sender.Send(outMsg, nodeIDs, s.ctx.SubnetID, s.ctx.IsValidatorOnly()); sentTo.Len() == 0 {
		s.ctx.Log.Debug("failed to send message",
			zap.Stringer("messageOp", message.AcceptedFrontier),
			zap.Stringer("nodeID", nodeID),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Uint32("requestID", requestID),
			zap.Stringer("containerIDs", ids.SliceStringer(containerIDs)),
		)
	}
}

func (s *sender) SendGetAccepted(nodeIDs ids.NodeIDSet, requestID uint32, containerIDs []ids.ID) {
	// Note that this timeout duration won't exactly match the one that gets
	// registered. That's OK.
	deadline := s.timeouts.TimeoutDuration()

	// Tell the router to expect a response message or a message notifying
	// that we won't get a response from each of these nodes.
	// We register timeouts for all nodes, regardless of whether we fail
	// to send them a message, to avoid busy looping when disconnected from
	// the internet.
	for nodeID := range nodeIDs {
		s.router.RegisterRequest(nodeID, s.ctx.ChainID, requestID, message.Accepted)
	}

	msgCreator := s.getMsgCreator()

	// Sending a message to myself. No need to send it over the network.
	// Just put it right into the router. Asynchronously to avoid deadlock.
	if nodeIDs.Contains(s.ctx.NodeID) {
		nodeIDs.Remove(s.ctx.NodeID)
		inMsg := msgCreator.InboundGetAccepted(s.ctx.ChainID, requestID, deadline, containerIDs, s.ctx.NodeID)
		go s.router.HandleInbound(inMsg)
	}

	// Create the outbound message.
	outMsg, err := msgCreator.GetAccepted(s.ctx.ChainID, requestID, deadline, containerIDs)

	// Send the message over the network.
	var sentTo ids.NodeIDSet
	if err == nil {
		sentTo = s.sender.Send(outMsg, nodeIDs, s.ctx.SubnetID, s.ctx.IsValidatorOnly())
	} else {
		s.ctx.Log.Error("failed to build message",
			zap.Stringer("messageOp", message.GetAccepted),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Uint32("requestID", requestID),
			zap.Stringer("containerIDs", ids.SliceStringer(containerIDs)),
			zap.Error(err),
		)
	}

	for nodeID := range nodeIDs {
		if !sentTo.Contains(nodeID) {
			s.ctx.Log.Debug("failed to send message",
				zap.Stringer("messageOp", message.GetAccepted),
				zap.Stringer("nodeID", nodeID),
				zap.Stringer("chainID", s.ctx.ChainID),
				zap.Uint32("requestID", requestID),
				zap.Stringer("containerIDs", ids.SliceStringer(containerIDs)),
			)
		}
	}
}

func (s *sender) SendAccepted(nodeID ids.NodeID, requestID uint32, containerIDs []ids.ID) {
	msgCreator := s.getMsgCreator()

	if nodeID == s.ctx.NodeID {
		inMsg := msgCreator.InboundAccepted(s.ctx.ChainID, requestID, containerIDs, nodeID)
		go s.router.HandleInbound(inMsg)
		return
	}

	// Create the outbound message.
	outMsg, err := msgCreator.Accepted(s.ctx.ChainID, requestID, containerIDs)
	if err != nil {
		s.ctx.Log.Error("failed to build message",
			zap.Stringer("messageOp", message.Accepted),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Uint32("requestID", requestID),
			zap.Stringer("containerIDs", ids.SliceStringer(containerIDs)),
			zap.Error(err),
		)
		return
	}

	// Send the message over the network.
	nodeIDs := ids.NewNodeIDSet(1)
	nodeIDs.Add(nodeID)
	if sentTo := s.sender.Send(outMsg, nodeIDs, s.ctx.SubnetID, s.ctx.IsValidatorOnly()); sentTo.Len() == 0 {
		s.ctx.Log.Debug("failed to send message",
			zap.Stringer("messageOp", message.Accepted),
			zap.Stringer("nodeID", nodeID),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Uint32("requestID", requestID),
			zap.Stringer("containerIDs", ids.SliceStringer(containerIDs)),
		)
	}
}

func (s *sender) SendGetAncestors(nodeID ids.NodeID, requestID uint32, containerID ids.ID) {
	// Tell the router to expect a response message or a message notifying
	// that we won't get a response from this node.
	s.router.RegisterRequest(nodeID, s.ctx.ChainID, requestID, message.Ancestors)

	msgCreator := s.getMsgCreator()

	// Sending a GetAncestors to myself always fails.
	if nodeID == s.ctx.NodeID {
		inMsg := msgCreator.InternalFailedRequest(message.GetAncestorsFailed, nodeID, s.ctx.ChainID, requestID)
		go s.router.HandleInbound(inMsg)
		return
	}

	// [nodeID] may be benched. That is, they've been unresponsive
	// so we don't even bother sending requests to them. We just have them immediately fail.
	if s.timeouts.IsBenched(nodeID, s.ctx.ChainID) {
		s.failedDueToBench[message.GetAncestors].Inc() // update metric
		s.timeouts.RegisterRequestToUnreachableValidator()
		inMsg := msgCreator.InternalFailedRequest(message.GetAncestorsFailed, nodeID, s.ctx.ChainID, requestID)
		go s.router.HandleInbound(inMsg)
		return
	}

	// Note that this timeout duration won't exactly match the one that gets
	// registered. That's OK.
	deadline := s.timeouts.TimeoutDuration()
	// Create the outbound message.
	outMsg, err := msgCreator.GetAncestors(s.ctx.ChainID, requestID, deadline, containerID)
	if err != nil {
		s.ctx.Log.Error("failed to build message",
			zap.Stringer("messageOp", message.GetAncestors),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Uint32("requestID", requestID),
			zap.Stringer("containerID", containerID),
			zap.Error(err),
		)

		inMsg := msgCreator.InternalFailedRequest(message.GetAncestorsFailed, nodeID, s.ctx.ChainID, requestID)
		go s.router.HandleInbound(inMsg)
		return
	}

	// Send the message over the network.
	nodeIDs := ids.NewNodeIDSet(1)
	nodeIDs.Add(nodeID)
	if sentTo := s.sender.Send(outMsg, nodeIDs, s.ctx.SubnetID, s.ctx.IsValidatorOnly()); sentTo.Len() == 0 {
		s.ctx.Log.Debug("failed to send message",
			zap.Stringer("messageOp", message.GetAncestors),
			zap.Stringer("nodeID", nodeID),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Uint32("requestID", requestID),
			zap.Stringer("containerID", containerID),
		)

		s.timeouts.RegisterRequestToUnreachableValidator()
		inMsg := msgCreator.InternalFailedRequest(message.GetAncestorsFailed, nodeID, s.ctx.ChainID, requestID)
		go s.router.HandleInbound(inMsg)
	}
}

// SendAncestors sends an Ancestors message to the consensus engine running on the specified chain
// on the specified node.
// The Ancestors message gives the recipient the contents of several containers.
func (s *sender) SendAncestors(nodeID ids.NodeID, requestID uint32, containers [][]byte) {
	msgCreator := s.getMsgCreator()

	// Create the outbound message.
	outMsg, err := msgCreator.Ancestors(s.ctx.ChainID, requestID, containers)
	if err != nil {
		s.ctx.Log.Error("failed to build message",
			zap.Stringer("messageOp", message.Ancestors),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Uint32("requestID", requestID),
			zap.Int("numContainers", len(containers)),
			zap.Error(err),
		)
		return
	}

	// Send the message over the network.
	nodeIDs := ids.NewNodeIDSet(1)
	nodeIDs.Add(nodeID)
	if sentTo := s.sender.Send(outMsg, nodeIDs, s.ctx.SubnetID, s.ctx.IsValidatorOnly()); sentTo.Len() == 0 {
		s.ctx.Log.Debug("failed to send message",
			zap.Stringer("messageOp", message.Ancestors),
			zap.Stringer("nodeID", nodeID),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Uint32("requestID", requestID),
			zap.Int("numContainers", len(containers)),
		)
	}
}

// SendGet sends a Get message to the consensus engine running on the specified
// chain to the specified node. The Get message signifies that this
// consensus engine would like the recipient to send this consensus engine the
// specified container.
func (s *sender) SendGet(nodeID ids.NodeID, requestID uint32, containerID ids.ID) {
	// Tell the router to expect a response message or a message notifying
	// that we won't get a response from this node.
	s.router.RegisterRequest(nodeID, s.ctx.ChainID, requestID, message.Put)

	msgCreator := s.getMsgCreator()

	// Sending a Get to myself always fails.
	if nodeID == s.ctx.NodeID {
		inMsg := msgCreator.InternalFailedRequest(message.GetFailed, nodeID, s.ctx.ChainID, requestID)
		go s.router.HandleInbound(inMsg)
		return
	}

	// [nodeID] may be benched. That is, they've been unresponsive
	// so we don't even bother sending requests to them. We just have them immediately fail.
	if s.timeouts.IsBenched(nodeID, s.ctx.ChainID) {
		s.failedDueToBench[message.Get].Inc() // update metric
		s.timeouts.RegisterRequestToUnreachableValidator()
		inMsg := msgCreator.InternalFailedRequest(message.GetFailed, nodeID, s.ctx.ChainID, requestID)
		go s.router.HandleInbound(inMsg)
		return
	}

	// Note that this timeout duration won't exactly match the one that gets
	// registered. That's OK.
	deadline := s.timeouts.TimeoutDuration()
	// Create the outbound message.
	outMsg, err := msgCreator.Get(s.ctx.ChainID, requestID, deadline, containerID)

	// Send the message over the network.
	var sentTo ids.NodeIDSet
	if err == nil {
		nodeIDs := ids.NewNodeIDSet(1)
		nodeIDs.Add(nodeID)
		sentTo = s.sender.Send(outMsg, nodeIDs, s.ctx.SubnetID, s.ctx.IsValidatorOnly())
	} else {
		s.ctx.Log.Error("failed to build message",
			zap.Stringer("messageOp", message.Get),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Uint32("requestID", requestID),
			zap.Duration("deadline", deadline),
			zap.Stringer("containerID", containerID),
			zap.Error(err),
		)
	}

	if sentTo.Len() == 0 {
		s.ctx.Log.Debug("failed to send message",
			zap.Stringer("messageOp", message.Get),
			zap.Stringer("nodeID", nodeID),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Uint32("requestID", requestID),
			zap.Stringer("containerID", containerID),
		)

		s.timeouts.RegisterRequestToUnreachableValidator()
		inMsg := msgCreator.InternalFailedRequest(message.GetFailed, nodeID, s.ctx.ChainID, requestID)
		go s.router.HandleInbound(inMsg)
	}
}

// SendPut sends a Put message to the consensus engine running on the specified chain
// on the specified node.
// The Put message signifies that this consensus engine is giving to the recipient
// the contents of the specified container.
func (s *sender) SendPut(nodeID ids.NodeID, requestID uint32, container []byte) {
	msgCreator := s.getMsgCreator()

	// Create the outbound message.
	outMsg, err := msgCreator.Put(s.ctx.ChainID, requestID, container)
	if err != nil {
		s.ctx.Log.Error("failed to build message",
			zap.Stringer("messageOp", message.Put),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Uint32("requestID", requestID),
			zap.Binary("container", container),
			zap.Error(err),
		)
		return
	}

	// Send the message over the network.
	nodeIDs := ids.NewNodeIDSet(1)
	nodeIDs.Add(nodeID)
	if sentTo := s.sender.Send(outMsg, nodeIDs, s.ctx.SubnetID, s.ctx.IsValidatorOnly()); sentTo.Len() == 0 {
		s.ctx.Log.Debug("failed to send message",
			zap.Stringer("messageOp", message.Put),
			zap.Stringer("nodeID", nodeID),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Uint32("requestID", requestID),
		)
		s.ctx.Log.Verbo("failed to send message",
			zap.Stringer("messageOp", message.Put),
			zap.Stringer("nodeID", nodeID),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Uint32("requestID", requestID),
			zap.Binary("container", container),
		)
	}
}

// SendPushQuery sends a PushQuery message to the consensus engines running on the specified chains
// on the specified nodes.
// The PushQuery message signifies that this consensus engine would like each node to send
// their preferred frontier given the existence of the specified container.
func (s *sender) SendPushQuery(nodeIDs ids.NodeIDSet, requestID uint32, container []byte) {
	// Tell the router to expect a response message or a message notifying
	// that we won't get a response from each of these nodes.
	// We register timeouts for all nodes, regardless of whether we fail
	// to send them a message, to avoid busy looping when disconnected from
	// the internet.
	for nodeID := range nodeIDs {
		s.router.RegisterRequest(nodeID, s.ctx.ChainID, requestID, message.Chits)
	}

	// Note that this timeout duration won't exactly match the one that gets
	// registered. That's OK.
	deadline := s.timeouts.TimeoutDuration()

	msgCreator := s.getMsgCreator()

	// Sending a message to myself. No need to send it over the network.
	// Just put it right into the router. Do so asynchronously to avoid deadlock.
	if nodeIDs.Contains(s.ctx.NodeID) {
		nodeIDs.Remove(s.ctx.NodeID)
		inMsg := msgCreator.InboundPushQuery(s.ctx.ChainID, requestID, deadline, container, s.ctx.NodeID)
		go s.router.HandleInbound(inMsg)
	}

	// Some of [nodeIDs] may be benched. That is, they've been unresponsive
	// so we don't even bother sending messages to them. We just have them immediately fail.
	for nodeID := range nodeIDs {
		if s.timeouts.IsBenched(nodeID, s.ctx.ChainID) {
			s.failedDueToBench[message.PushQuery].Inc() // update metric
			nodeIDs.Remove(nodeID)
			s.timeouts.RegisterRequestToUnreachableValidator()

			// Immediately register a failure. Do so asynchronously to avoid deadlock.
			inMsg := msgCreator.InternalFailedRequest(message.QueryFailed, nodeID, s.ctx.ChainID, requestID)
			go s.router.HandleInbound(inMsg)
		}
	}

	// Create the outbound message.
	// [sentTo] are the IDs of validators who may receive the message.
	outMsg, err := msgCreator.PushQuery(s.ctx.ChainID, requestID, deadline, container)

	// Send the message over the network.
	var sentTo ids.NodeIDSet
	if err == nil {
		sentTo = s.sender.Send(outMsg, nodeIDs, s.ctx.SubnetID, s.ctx.IsValidatorOnly())
	} else {
		s.ctx.Log.Error("failed to build message",
			zap.Stringer("messageOp", message.PushQuery),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Uint32("requestID", requestID),
			zap.Binary("container", container),
			zap.Error(err),
		)
	}

	for nodeID := range nodeIDs {
		if !sentTo.Contains(nodeID) {
			s.ctx.Log.Debug("failed to send message",
				zap.Stringer("messageOp", message.PushQuery),
				zap.Stringer("nodeID", nodeID),
				zap.Stringer("chainID", s.ctx.ChainID),
				zap.Uint32("requestID", requestID),
			)
			s.ctx.Log.Verbo("failed to send message",
				zap.Stringer("messageOp", message.PushQuery),
				zap.Stringer("nodeID", nodeID),
				zap.Stringer("chainID", s.ctx.ChainID),
				zap.Uint32("requestID", requestID),
				zap.Binary("container", container),
			)

			// Register failures for nodes we didn't send a request to.
			s.timeouts.RegisterRequestToUnreachableValidator()
			inMsg := msgCreator.InternalFailedRequest(message.QueryFailed, nodeID, s.ctx.ChainID, requestID)
			go s.router.HandleInbound(inMsg)
		}
	}
}

// SendPullQuery sends a PullQuery message to the consensus engines running on the specified chains
// on the specified nodes.
// The PullQuery message signifies that this consensus engine would like each node to send
// their preferred frontier.
func (s *sender) SendPullQuery(nodeIDs ids.NodeIDSet, requestID uint32, containerID ids.ID) {
	// Tell the router to expect a response message or a message notifying
	// that we won't get a response from each of these nodes.
	// We register timeouts for all nodes, regardless of whether we fail
	// to send them a message, to avoid busy looping when disconnected from
	// the internet.
	for nodeID := range nodeIDs {
		s.router.RegisterRequest(nodeID, s.ctx.ChainID, requestID, message.Chits)
	}

	// Note that this timeout duration won't exactly match the one that gets
	// registered. That's OK.
	deadline := s.timeouts.TimeoutDuration()

	msgCreator := s.getMsgCreator()

	// Sending a message to myself. No need to send it over the network.
	// Just put it right into the router. Do so asynchronously to avoid deadlock.
	if nodeIDs.Contains(s.ctx.NodeID) {
		nodeIDs.Remove(s.ctx.NodeID)
		inMsg := msgCreator.InboundPullQuery(s.ctx.ChainID, requestID, deadline, containerID, s.ctx.NodeID)
		go s.router.HandleInbound(inMsg)
	}

	// Some of the nodes in [nodeIDs] may be benched. That is, they've been unresponsive
	// so we don't even bother sending messages to them. We just have them immediately fail.
	for nodeID := range nodeIDs {
		if s.timeouts.IsBenched(nodeID, s.ctx.ChainID) {
			s.failedDueToBench[message.PullQuery].Inc() // update metric
			nodeIDs.Remove(nodeID)
			s.timeouts.RegisterRequestToUnreachableValidator()
			// Immediately register a failure. Do so asynchronously to avoid deadlock.
			inMsg := msgCreator.InternalFailedRequest(message.QueryFailed, nodeID, s.ctx.ChainID, requestID)
			go s.router.HandleInbound(inMsg)
		}
	}

	// Create the outbound message.
	outMsg, err := msgCreator.PullQuery(s.ctx.ChainID, requestID, deadline, containerID)

	// Send the message over the network.
	var sentTo ids.NodeIDSet
	if err == nil {
		sentTo = s.sender.Send(outMsg, nodeIDs, s.ctx.SubnetID, s.ctx.IsValidatorOnly())
	} else {
		s.ctx.Log.Error("failed to build message",
			zap.Stringer("messageOp", message.PullQuery),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Uint32("requestID", requestID),
			zap.Duration("deadline", deadline),
			zap.Stringer("containerID", containerID),
			zap.Error(err),
		)
	}

	for nodeID := range nodeIDs {
		if !sentTo.Contains(nodeID) {
			s.ctx.Log.Debug("failed to send message",
				zap.Stringer("messageOp", message.PullQuery),
				zap.Stringer("nodeID", nodeID),
				zap.Stringer("chainID", s.ctx.ChainID),
				zap.Uint32("requestID", requestID),
				zap.Stringer("containerID", containerID),
			)

			// Register failures for nodes we didn't send a request to.
			s.timeouts.RegisterRequestToUnreachableValidator()
			inMsg := msgCreator.InternalFailedRequest(message.QueryFailed, nodeID, s.ctx.ChainID, requestID)
			go s.router.HandleInbound(inMsg)
		}
	}
}

// SendChits sends chits
func (s *sender) SendChits(nodeID ids.NodeID, requestID uint32, votes []ids.ID) {
	msgCreator := s.getMsgCreator()

	// If [nodeID] is myself, send this message directly
	// to my own router rather than sending it over the network
	if nodeID == s.ctx.NodeID {
		inMsg := msgCreator.InboundChits(s.ctx.ChainID, requestID, votes, nodeID)
		go s.router.HandleInbound(inMsg)
		return
	}

	// Create the outbound message.
	outMsg, err := msgCreator.Chits(s.ctx.ChainID, requestID, votes)
	if err != nil {
		s.ctx.Log.Error("failed to build message",
			zap.Stringer("messageOp", message.Chits),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Uint32("requestID", requestID),
			zap.Stringer("containerIDs", ids.SliceStringer(votes)),
			zap.Error(err),
		)
		return
	}

	// Send the message over the network.
	nodeIDs := ids.NewNodeIDSet(1)
	nodeIDs.Add(nodeID)
	if sentTo := s.sender.Send(outMsg, nodeIDs, s.ctx.SubnetID, s.ctx.IsValidatorOnly()); sentTo.Len() == 0 {
		s.ctx.Log.Debug("failed to send message",
			zap.Stringer("messageOp", message.Chits),
			zap.Stringer("nodeID", nodeID),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Uint32("requestID", requestID),
			zap.Stringer("containerIDs", ids.SliceStringer(votes)),
		)
	}
}

// SendAppRequest sends an application-level request to the given nodes.
// The meaning of this request, and how it should be handled, is defined by the VM.
func (s *sender) SendAppRequest(nodeIDs ids.NodeIDSet, requestID uint32, appRequestBytes []byte) error {
	// Tell the router to expect a response message or a message notifying
	// that we won't get a response from each of these nodes.
	// We register timeouts for all nodes, regardless of whether we fail
	// to send them a message, to avoid busy looping when disconnected from
	// the internet.
	for nodeID := range nodeIDs {
		s.router.RegisterRequest(nodeID, s.ctx.ChainID, requestID, message.AppResponse)
	}

	// Note that this timeout duration won't exactly match the one that gets
	// registered. That's OK.
	deadline := s.timeouts.TimeoutDuration()

	msgCreator := s.getMsgCreator()

	// Sending a message to myself. No need to send it over the network.
	// Just put it right into the router. Do so asynchronously to avoid deadlock.
	if nodeIDs.Contains(s.ctx.NodeID) {
		nodeIDs.Remove(s.ctx.NodeID)
		inMsg := msgCreator.InboundAppRequest(s.ctx.ChainID, requestID, deadline, appRequestBytes, s.ctx.NodeID)
		go s.router.HandleInbound(inMsg)
	}

	// Some of the nodes in [nodeIDs] may be benched. That is, they've been unresponsive
	// so we don't even bother sending messages to them. We just have them immediately fail.
	for nodeID := range nodeIDs {
		if s.timeouts.IsBenched(nodeID, s.ctx.ChainID) {
			s.failedDueToBench[message.AppRequest].Inc() // update metric
			nodeIDs.Remove(nodeID)
			s.timeouts.RegisterRequestToUnreachableValidator()

			// Immediately register a failure. Do so asynchronously to avoid deadlock.
			inMsg := msgCreator.InternalFailedRequest(message.AppRequestFailed, nodeID, s.ctx.ChainID, requestID)
			go s.router.HandleInbound(inMsg)
		}
	}

	// Create the outbound message.
	// [sentTo] are the IDs of nodes who may receive the message.
	outMsg, err := msgCreator.AppRequest(s.ctx.ChainID, requestID, deadline, appRequestBytes)

	// Send the message over the network.
	var sentTo ids.NodeIDSet
	if err == nil {
		sentTo = s.sender.Send(outMsg, nodeIDs, s.ctx.SubnetID, s.ctx.IsValidatorOnly())
	} else {
		s.ctx.Log.Error("failed to build message",
			zap.Stringer("messageOp", message.AppRequest),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Uint32("requestID", requestID),
			zap.Binary("payload", appRequestBytes),
			zap.Error(err),
		)
	}

	for nodeID := range nodeIDs {
		if !sentTo.Contains(nodeID) {
			s.ctx.Log.Debug("failed to send message",
				zap.Stringer("messageOp", message.AppRequest),
				zap.Stringer("nodeID", nodeID),
				zap.Stringer("chainID", s.ctx.ChainID),
				zap.Uint32("requestID", requestID),
			)
			s.ctx.Log.Verbo("failed to send message",
				zap.Stringer("messageOp", message.AppRequest),
				zap.Stringer("nodeID", nodeID),
				zap.Stringer("chainID", s.ctx.ChainID),
				zap.Uint32("requestID", requestID),
				zap.Binary("payload", appRequestBytes),
			)

			// Register failures for nodes we didn't send a request to.
			s.timeouts.RegisterRequestToUnreachableValidator()
			inMsg := msgCreator.InternalFailedRequest(message.AppRequestFailed, nodeID, s.ctx.ChainID, requestID)
			go s.router.HandleInbound(inMsg)
		}
	}
	return nil
}

// SendAppResponse sends a response to an application-level request from the
// given node
func (s *sender) SendAppResponse(nodeID ids.NodeID, requestID uint32, appResponseBytes []byte) error {
	msgCreator := s.getMsgCreator()

	if nodeID == s.ctx.NodeID {
		inMsg := msgCreator.InboundAppResponse(s.ctx.ChainID, requestID, appResponseBytes, nodeID)
		go s.router.HandleInbound(inMsg)
		return nil
	}

	// Create the outbound message.
	outMsg, err := msgCreator.AppResponse(s.ctx.ChainID, requestID, appResponseBytes)
	if err != nil {
		s.ctx.Log.Error("failed to build message",
			zap.Stringer("messageOp", message.AppResponse),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Uint32("requestID", requestID),
			zap.Binary("payload", appResponseBytes),
			zap.Error(err),
		)
		return nil
	}

	// Send the message over the network.
	nodeIDs := ids.NewNodeIDSet(1)
	nodeIDs.Add(nodeID)
	if sentTo := s.sender.Send(outMsg, nodeIDs, s.ctx.SubnetID, s.ctx.IsValidatorOnly()); sentTo.Len() == 0 {
		s.ctx.Log.Debug("failed to send message",
			zap.Stringer("messageOp", message.AppResponse),
			zap.Stringer("nodeID", nodeID),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Uint32("requestID", requestID),
		)
		s.ctx.Log.Verbo("failed to send message",
			zap.Stringer("messageOp", message.AppResponse),
			zap.Stringer("nodeID", nodeID),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Uint32("requestID", requestID),
			zap.Binary("payload", appResponseBytes),
		)
	}
	return nil
}

func (s *sender) SendAppGossipSpecific(nodeIDs ids.NodeIDSet, appGossipBytes []byte) error {
	msgCreator := s.getMsgCreator()

	// Create the outbound message.
	outMsg, err := msgCreator.AppGossip(s.ctx.ChainID, appGossipBytes)
	if err != nil {
		s.ctx.Log.Error("failed to build message",
			zap.Stringer("messageOp", message.AppGossip),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Binary("payload", appGossipBytes),
			zap.Error(err),
		)
		return nil
	}

	// Send the message over the network.
	if sentTo := s.sender.Send(outMsg, nodeIDs, s.ctx.SubnetID, s.ctx.IsValidatorOnly()); sentTo.Len() == 0 {
		for nodeID := range nodeIDs {
			if !sentTo.Contains(nodeID) {
				s.ctx.Log.Debug("failed to send message",
					zap.Stringer("messageOp", message.AppGossip),
					zap.Stringer("nodeID", nodeID),
					zap.Stringer("chainID", s.ctx.ChainID),
				)
				s.ctx.Log.Verbo("failed to send message",
					zap.Stringer("messageOp", message.AppGossip),
					zap.Stringer("nodeID", nodeID),
					zap.Stringer("chainID", s.ctx.ChainID),
					zap.Binary("payload", appGossipBytes),
				)
			}
		}
	}
	return nil
}

// SendAppGossip sends an application-level gossip message.
func (s *sender) SendAppGossip(appGossipBytes []byte) error {
	msgCreator := s.getMsgCreator()

	// Create the outbound message.
	outMsg, err := msgCreator.AppGossip(s.ctx.ChainID, appGossipBytes)
	if err != nil {
		s.ctx.Log.Error("failed to build message",
			zap.Stringer("messageOp", message.AppGossip),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Binary("payload", appGossipBytes),
			zap.Error(err),
		)
		return nil
	}

	validatorSize := int(s.gossipConfig.AppGossipValidatorSize)
	nonValidatorSize := int(s.gossipConfig.AppGossipNonValidatorSize)
	peerSize := int(s.gossipConfig.AppGossipPeerSize)

	sentTo := s.sender.Gossip(outMsg, s.ctx.SubnetID, s.ctx.IsValidatorOnly(), validatorSize, nonValidatorSize, peerSize)
	if sentTo.Len() == 0 {
		s.ctx.Log.Debug("failed to send message",
			zap.Stringer("messageOp", message.AppGossip),
			zap.Stringer("chainID", s.ctx.ChainID),
		)
		s.ctx.Log.Verbo("failed to send message",
			zap.Stringer("messageOp", message.AppGossip),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Binary("payload", appGossipBytes),
		)
	}
	return nil
}

// SendGossip gossips the provided container
func (s *sender) SendGossip(container []byte) {
	msgCreator := s.getMsgCreator()

	// Create the outbound message.
	outMsg, err := msgCreator.Put(s.ctx.ChainID, constants.GossipMsgRequestID, container)
	if err != nil {
		s.ctx.Log.Error("failed to build message",
			zap.Stringer("messageOp", message.Put),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Binary("container", container),
			zap.Error(err),
		)
		return
	}

	sentTo := s.sender.Gossip(
		outMsg,
		s.ctx.SubnetID,
		s.ctx.IsValidatorOnly(),
		int(s.gossipConfig.AcceptedFrontierValidatorSize),
		int(s.gossipConfig.AcceptedFrontierNonValidatorSize),
		int(s.gossipConfig.AcceptedFrontierPeerSize),
	)
	if sentTo.Len() == 0 {
		s.ctx.Log.Debug("failed to send message",
			zap.Stringer("messageOp", message.Put),
			zap.Stringer("chainID", s.ctx.ChainID),
		)
		s.ctx.Log.Verbo("failed to send message",
			zap.Stringer("messageOp", message.Put),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Binary("container", container),
		)
	}
}

// Accept is called after every consensus decision
func (s *sender) Accept(ctx *snow.ConsensusContext, _ ids.ID, container []byte) error {
	if ctx.GetState() != snow.NormalOp {
		// don't gossip during bootstrapping
		return nil
	}

	msgCreator := s.getMsgCreator()

	// Create the outbound message.
	outMsg, err := msgCreator.Put(s.ctx.ChainID, constants.GossipMsgRequestID, container)
	if err != nil {
		s.ctx.Log.Error("failed to build message",
			zap.Stringer("messageOp", message.Put),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Binary("container", container),
			zap.Error(err),
		)
		return nil
	}

	sentTo := s.sender.Gossip(
		outMsg,
		s.ctx.SubnetID,
		s.ctx.IsValidatorOnly(),
		int(s.gossipConfig.OnAcceptValidatorSize),
		int(s.gossipConfig.OnAcceptNonValidatorSize),
		int(s.gossipConfig.OnAcceptPeerSize),
	)
	if sentTo.Len() == 0 {
		s.ctx.Log.Debug("failed to send message",
			zap.Stringer("messageOp", message.Put),
			zap.Stringer("chainID", s.ctx.ChainID),
		)
		s.ctx.Log.Verbo("failed to send message",
			zap.Stringer("messageOp", message.Put),
			zap.Stringer("chainID", s.ctx.ChainID),
			zap.Binary("container", container),
		)
	}
	return nil
}
