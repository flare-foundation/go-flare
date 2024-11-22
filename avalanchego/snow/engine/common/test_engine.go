// Copyright (C) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package common

import (
	"errors"
	"testing"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/version"
)

var (
	errTimeout                       = errors.New("unexpectedly called Timeout")
	errGossip                        = errors.New("unexpectedly called Gossip")
	errNotify                        = errors.New("unexpectedly called Notify")
	errGetStateSummaryFrontier       = errors.New("unexpectedly called GetStateSummaryFrontier")
	errGetStateSummaryFrontierFailed = errors.New("unexpectedly called GetStateSummaryFrontierFailed")
	errStateSummaryFrontier          = errors.New("unexpectedly called StateSummaryFrontier")
	errGetAcceptedStateSummary       = errors.New("unexpectedly called GetAcceptedStateSummary")
	errGetAcceptedStateSummaryFailed = errors.New("unexpectedly called GetAcceptedStateSummaryFailed")
	errAcceptedStateSummary          = errors.New("unexpectedly called AcceptedStateSummary")
	errGetAcceptedFrontier           = errors.New("unexpectedly called GetAcceptedFrontier")
	errGetAcceptedFrontierFailed     = errors.New("unexpectedly called GetAcceptedFrontierFailed")
	errAcceptedFrontier              = errors.New("unexpectedly called AcceptedFrontier")
	errGetAccepted                   = errors.New("unexpectedly called GetAccepted")
	errGetAcceptedFailed             = errors.New("unexpectedly called GetAcceptedFailed")
	errAccepted                      = errors.New("unexpectedly called Accepted")
	errGet                           = errors.New("unexpectedly called Get")
	errGetAncestors                  = errors.New("unexpectedly called GetAncestors")
	errGetFailed                     = errors.New("unexpectedly called GetFailed")
	errGetAncestorsFailed            = errors.New("unexpectedly called GetAncestorsFailed")
	errPut                           = errors.New("unexpectedly called Put")
	errAncestors                     = errors.New("unexpectedly called Ancestors")
	errPushQuery                     = errors.New("unexpectedly called PushQuery")
	errPullQuery                     = errors.New("unexpectedly called PullQuery")
	errQueryFailed                   = errors.New("unexpectedly called QueryFailed")
	errChits                         = errors.New("unexpectedly called Chits")
	errStart                         = errors.New("unexpectedly called Start")

	_ Engine = &EngineTest{}
)

// EngineTest is a test engine
type EngineTest struct {
	T *testing.T

	CantStart,

	CantIsBootstrapped,
	CantTimeout,
	CantGossip,
	CantHalt,
	CantShutdown,

	CantContext,

	CantNotify,

	CantGetStateSummaryFrontier,
	CantGetStateSummaryFrontierFailed,
	CantStateSummaryFrontier,

	CantGetAcceptedStateSummary,
	CantGetAcceptedStateSummaryFailed,
	CantAcceptedStateSummary,

	CantGetAcceptedFrontier,
	CantGetAcceptedFrontierFailed,
	CantAcceptedFrontier,

	CantGetAccepted,
	CantGetAcceptedFailed,
	CantAccepted,

	CantGet,
	CantGetAncestors,
	CantGetFailed,
	CantGetAncestorsFailed,
	CantPut,
	CantAncestors,

	CantPushQuery,
	CantPullQuery,
	CantQueryFailed,
	CantChits,

	CantConnected,
	CantDisconnected,

	CantHealth,

	CantAppRequest,
	CantAppResponse,
	CantAppGossip,
	CantAppRequestFailed,

	CantGetVM bool

	StartF                                             func(startReqID uint32) error
	IsBootstrappedF                                    func() bool
	ContextF                                           func() *snow.ConsensusContext
	HaltF                                              func()
	TimeoutF, GossipF, ShutdownF                       func() error
	NotifyF                                            func(Message) error
	GetF, GetAncestorsF, PullQueryF                    func(nodeID ids.NodeID, requestID uint32, containerID ids.ID) error
	PutF, PushQueryF                                   func(nodeID ids.NodeID, requestID uint32, container []byte) error
	AncestorsF                                         func(nodeID ids.NodeID, requestID uint32, containers [][]byte) error
	AcceptedFrontierF, GetAcceptedF, AcceptedF, ChitsF func(nodeID ids.NodeID, requestID uint32, containerIDs []ids.ID) error
	GetStateSummaryFrontierF, GetStateSummaryFrontierFailedF, GetAcceptedStateSummaryFailedF,
	GetAcceptedFrontierF, GetFailedF, GetAncestorsFailedF,
	QueryFailedF, GetAcceptedFrontierFailedF, GetAcceptedFailedF, AppRequestFailedF func(nodeID ids.NodeID, requestID uint32) error
	StateSummaryFrontierF     func(nodeID ids.NodeID, requestID uint32, summary []byte) error
	GetAcceptedStateSummaryF  func(nodeID ids.NodeID, requestID uint32, keys []uint64) error
	AcceptedStateSummaryF     func(nodeID ids.NodeID, requestID uint32, summaryIDs []ids.ID) error
	ConnectedF                func(nodeID ids.NodeID, nodeVersion *version.Application) error
	DisconnectedF             func(nodeID ids.NodeID) error
	HealthF                   func() (interface{}, error)
	GetVMF                    func() VM
	AppRequestF, AppResponseF func(nodeID ids.NodeID, requestID uint32, msg []byte) error
	AppGossipF                func(nodeID ids.NodeID, msg []byte) error
}

func (e *EngineTest) Default(cant bool) {
	e.CantStart = cant
	e.CantIsBootstrapped = cant
	e.CantTimeout = cant
	e.CantGossip = cant
	e.CantHalt = cant
	e.CantShutdown = cant
	e.CantContext = cant
	e.CantNotify = cant
	e.CantGetStateSummaryFrontier = cant
	e.CantGetStateSummaryFrontierFailed = cant
	e.CantStateSummaryFrontier = cant
	e.CantGetAcceptedStateSummary = cant
	e.CantGetAcceptedStateSummaryFailed = cant
	e.CantAcceptedStateSummary = cant
	e.CantGetAcceptedFrontier = cant
	e.CantGetAcceptedFrontierFailed = cant
	e.CantAcceptedFrontier = cant
	e.CantGetAccepted = cant
	e.CantGetAcceptedFailed = cant
	e.CantAccepted = cant
	e.CantGet = cant
	e.CantGetAncestors = cant
	e.CantGetAncestorsFailed = cant
	e.CantGetFailed = cant
	e.CantPut = cant
	e.CantAncestors = cant
	e.CantPushQuery = cant
	e.CantPullQuery = cant
	e.CantQueryFailed = cant
	e.CantChits = cant
	e.CantConnected = cant
	e.CantDisconnected = cant
	e.CantHealth = cant
	e.CantAppRequest = cant
	e.CantAppRequestFailed = cant
	e.CantAppResponse = cant
	e.CantAppGossip = cant
	e.CantGetVM = cant
}

func (e *EngineTest) Start(startReqID uint32) error {
	if e.StartF != nil {
		return e.StartF(startReqID)
	}
	if e.CantStart && e.T != nil {
		e.T.Fatalf("Unexpectedly called Start")
	}
	return errStart
}

func (e *EngineTest) Context() *snow.ConsensusContext {
	if e.ContextF != nil {
		return e.ContextF()
	}
	if e.CantContext && e.T != nil {
		e.T.Fatalf("Unexpectedly called Context")
	}
	return nil
}

func (e *EngineTest) Timeout() error {
	if e.TimeoutF != nil {
		return e.TimeoutF()
	}
	if !e.CantTimeout {
		return nil
	}
	if e.T != nil {
		e.T.Fatal(errTimeout)
	}
	return errTimeout
}

func (e *EngineTest) Gossip() error {
	if e.GossipF != nil {
		return e.GossipF()
	}
	if !e.CantGossip {
		return nil
	}
	if e.T != nil {
		e.T.Fatal(errGossip)
	}
	return errGossip
}

func (e *EngineTest) Halt() {
	if e.HaltF != nil {
		e.HaltF()
	} else if e.CantHalt && e.T != nil {
		e.T.Fatalf("Unexpectedly called Halt")
	}
}

func (e *EngineTest) Shutdown() error {
	if e.ShutdownF != nil {
		return e.ShutdownF()
	}
	if !e.CantShutdown {
		return nil
	}
	if e.T != nil {
		e.T.Fatal(errShutdown)
	}
	return errShutdown
}

func (e *EngineTest) Notify(msg Message) error {
	if e.NotifyF != nil {
		return e.NotifyF(msg)
	}
	if !e.CantNotify {
		return nil
	}
	if e.T != nil {
		e.T.Fatal(errNotify)
	}
	return errNotify
}

func (e *EngineTest) GetStateSummaryFrontier(validatorID ids.NodeID, requestID uint32) error {
	if e.GetStateSummaryFrontierF != nil {
		return e.GetStateSummaryFrontierF(validatorID, requestID)
	}
	if e.CantGetStateSummaryFrontier && e.T != nil {
		e.T.Fatalf("Unexpectedly called GetStateSummaryFrontier")
	}
	return errGetStateSummaryFrontier
}

func (e *EngineTest) StateSummaryFrontier(validatorID ids.NodeID, requestID uint32, summary []byte) error {
	if e.StateSummaryFrontierF != nil {
		return e.StateSummaryFrontierF(validatorID, requestID, summary)
	}
	if e.CantGetStateSummaryFrontier && e.T != nil {
		e.T.Fatalf("Unexpectedly called CantStateSummaryFrontier")
	}
	return errStateSummaryFrontier
}

func (e *EngineTest) GetStateSummaryFrontierFailed(validatorID ids.NodeID, requestID uint32) error {
	if e.GetStateSummaryFrontierFailedF != nil {
		return e.GetStateSummaryFrontierFailedF(validatorID, requestID)
	}
	if e.CantGetStateSummaryFrontierFailed && e.T != nil {
		e.T.Fatalf("Unexpectedly called GetStateSummaryFrontierFailed")
	}
	return errGetStateSummaryFrontierFailed
}

func (e *EngineTest) GetAcceptedStateSummary(validatorID ids.NodeID, requestID uint32, keys []uint64) error {
	if e.GetAcceptedStateSummaryF != nil {
		return e.GetAcceptedStateSummaryF(validatorID, requestID, keys)
	}
	if e.CantGetAcceptedStateSummary && e.T != nil {
		e.T.Fatalf("Unexpectedly called GetAcceptedStateSummary")
	}
	return errGetAcceptedStateSummary
}

func (e *EngineTest) AcceptedStateSummary(validatorID ids.NodeID, requestID uint32, summaryIDs []ids.ID) error {
	if e.AcceptedStateSummaryF != nil {
		return e.AcceptedStateSummaryF(validatorID, requestID, summaryIDs)
	}
	if e.CantAcceptedStateSummary && e.T != nil {
		e.T.Fatalf("Unexpectedly called AcceptedStateSummary")
	}
	return errAcceptedStateSummary
}

func (e *EngineTest) GetAcceptedStateSummaryFailed(validatorID ids.NodeID, requestID uint32) error {
	if e.GetAcceptedStateSummaryFailedF != nil {
		return e.GetAcceptedStateSummaryFailedF(validatorID, requestID)
	}
	if e.CantGetAcceptedStateSummaryFailed && e.T != nil {
		e.T.Fatalf("Unexpectedly called GetAcceptedStateSummaryFailed")
	}
	return errGetAcceptedStateSummaryFailed
}

func (e *EngineTest) GetAcceptedFrontier(nodeID ids.NodeID, requestID uint32) error {
	if e.GetAcceptedFrontierF != nil {
		return e.GetAcceptedFrontierF(nodeID, requestID)
	}
	if !e.CantGetAcceptedFrontier {
		return nil
	}
	if e.T != nil {
		e.T.Fatal(errGetAcceptedFrontier)
	}
	return errGetAcceptedFrontier
}

func (e *EngineTest) GetAcceptedFrontierFailed(nodeID ids.NodeID, requestID uint32) error {
	if e.GetAcceptedFrontierFailedF != nil {
		return e.GetAcceptedFrontierFailedF(nodeID, requestID)
	}
	if !e.CantGetAcceptedFrontierFailed {
		return nil
	}
	if e.T != nil {
		e.T.Fatal(errGetAcceptedFrontierFailed)
	}
	return errGetAcceptedFrontierFailed
}

func (e *EngineTest) AcceptedFrontier(nodeID ids.NodeID, requestID uint32, containerIDs []ids.ID) error {
	if e.AcceptedFrontierF != nil {
		return e.AcceptedFrontierF(nodeID, requestID, containerIDs)
	}
	if !e.CantAcceptedFrontier {
		return nil
	}
	if e.T != nil {
		e.T.Fatal(errAcceptedFrontier)
	}
	return errAcceptedFrontier
}

func (e *EngineTest) GetAccepted(nodeID ids.NodeID, requestID uint32, containerIDs []ids.ID) error {
	if e.GetAcceptedF != nil {
		return e.GetAcceptedF(nodeID, requestID, containerIDs)
	}
	if !e.CantGetAccepted {
		return nil
	}
	if e.T != nil {
		e.T.Fatal(errGetAccepted)
	}
	return errGetAccepted
}

func (e *EngineTest) GetAcceptedFailed(nodeID ids.NodeID, requestID uint32) error {
	if e.GetAcceptedFailedF != nil {
		return e.GetAcceptedFailedF(nodeID, requestID)
	}
	if !e.CantGetAcceptedFailed {
		return nil
	}
	if e.T != nil {
		e.T.Fatal(errGetAcceptedFailed)
	}
	return errGetAcceptedFailed
}

func (e *EngineTest) Accepted(nodeID ids.NodeID, requestID uint32, containerIDs []ids.ID) error {
	if e.AcceptedF != nil {
		return e.AcceptedF(nodeID, requestID, containerIDs)
	}
	if !e.CantAccepted {
		return nil
	}
	if e.T != nil {
		e.T.Fatal(errAccepted)
	}
	return errAccepted
}

func (e *EngineTest) Get(nodeID ids.NodeID, requestID uint32, containerID ids.ID) error {
	if e.GetF != nil {
		return e.GetF(nodeID, requestID, containerID)
	}
	if !e.CantGet {
		return nil
	}
	if e.T != nil {
		e.T.Fatal(errGet)
	}
	return errGet
}

func (e *EngineTest) GetAncestors(nodeID ids.NodeID, requestID uint32, containerID ids.ID) error {
	if e.GetAncestorsF != nil {
		return e.GetAncestorsF(nodeID, requestID, containerID)
	}
	if !e.CantGetAncestors {
		return nil
	}
	if e.T != nil {
		e.T.Fatal(errGetAncestors)
	}
	return errGetAncestors
}

func (e *EngineTest) GetFailed(nodeID ids.NodeID, requestID uint32) error {
	if e.GetFailedF != nil {
		return e.GetFailedF(nodeID, requestID)
	}
	if !e.CantGetFailed {
		return nil
	}
	if e.T != nil {
		e.T.Fatal(errGetFailed)
	}
	return errGetFailed
}

func (e *EngineTest) GetAncestorsFailed(nodeID ids.NodeID, requestID uint32) error {
	if e.GetAncestorsFailedF != nil {
		return e.GetAncestorsFailedF(nodeID, requestID)
	}
	if e.CantGetAncestorsFailed {
		return nil
	}
	if e.T != nil {
		e.T.Fatal(errGetAncestorsFailed)
	}
	return errGetAncestorsFailed
}

func (e *EngineTest) Put(nodeID ids.NodeID, requestID uint32, container []byte) error {
	if e.PutF != nil {
		return e.PutF(nodeID, requestID, container)
	}
	if !e.CantPut {
		return nil
	}
	if e.T != nil {
		e.T.Fatal(errPut)
	}
	return errPut
}

func (e *EngineTest) Ancestors(nodeID ids.NodeID, requestID uint32, containers [][]byte) error {
	if e.AncestorsF != nil {
		return e.AncestorsF(nodeID, requestID, containers)
	}
	if !e.CantAncestors {
		return nil
	}
	if e.T != nil {
		e.T.Fatal(errAncestors)
	}
	return errAncestors
}

func (e *EngineTest) PushQuery(nodeID ids.NodeID, requestID uint32, container []byte) error {
	if e.PushQueryF != nil {
		return e.PushQueryF(nodeID, requestID, container)
	}
	if !e.CantPushQuery {
		return nil
	}
	if e.T != nil {
		e.T.Fatal(errPushQuery)
	}
	return errPushQuery
}

func (e *EngineTest) PullQuery(nodeID ids.NodeID, requestID uint32, containerID ids.ID) error {
	if e.PullQueryF != nil {
		return e.PullQueryF(nodeID, requestID, containerID)
	}
	if !e.CantPullQuery {
		return nil
	}
	if e.T != nil {
		e.T.Fatal(errPullQuery)
	}
	return errPullQuery
}

func (e *EngineTest) QueryFailed(nodeID ids.NodeID, requestID uint32) error {
	if e.QueryFailedF != nil {
		return e.QueryFailedF(nodeID, requestID)
	}
	if !e.CantQueryFailed {
		return nil
	}
	if e.T != nil {
		e.T.Fatal(errQueryFailed)
	}
	return errQueryFailed
}

func (e *EngineTest) AppRequest(nodeID ids.NodeID, requestID uint32, deadline time.Time, request []byte) error {
	if e.AppRequestF != nil {
		return e.AppRequestF(nodeID, requestID, request)
	}
	if !e.CantAppRequest {
		return nil
	}
	if e.T != nil {
		e.T.Fatal(errAppRequest)
	}
	return errAppRequest
}

func (e *EngineTest) AppResponse(nodeID ids.NodeID, requestID uint32, response []byte) error {
	if e.AppResponseF != nil {
		return e.AppResponseF(nodeID, requestID, response)
	}
	if !e.CantAppResponse {
		return nil
	}
	if e.T != nil {
		e.T.Fatal(errAppResponse)
	}
	return errAppResponse
}

func (e *EngineTest) AppRequestFailed(nodeID ids.NodeID, requestID uint32) error {
	if e.AppRequestFailedF != nil {
		return e.AppRequestFailedF(nodeID, requestID)
	}
	if !e.CantAppRequestFailed {
		return nil
	}
	if e.T != nil {
		e.T.Fatal(errAppRequestFailed)
	}
	return errAppRequestFailed
}

func (e *EngineTest) AppGossip(nodeID ids.NodeID, msg []byte) error {
	if e.AppGossipF != nil {
		return e.AppGossipF(nodeID, msg)
	}
	if !e.CantAppGossip {
		return nil
	}
	if e.T != nil {
		e.T.Fatal(errAppGossip)
	}
	return errAppGossip
}

func (e *EngineTest) Chits(nodeID ids.NodeID, requestID uint32, containerIDs []ids.ID) error {
	if e.ChitsF != nil {
		return e.ChitsF(nodeID, requestID, containerIDs)
	}
	if !e.CantChits {
		return nil
	}
	if e.T != nil {
		e.T.Fatal(errChits)
	}
	return errChits
}

func (e *EngineTest) Connected(nodeID ids.NodeID, nodeVersion *version.Application) error {
	if e.ConnectedF != nil {
		return e.ConnectedF(nodeID, nodeVersion)
	}
	if !e.CantConnected {
		return nil
	}
	if e.T != nil {
		e.T.Fatal(errConnected)
	}
	return errConnected
}

func (e *EngineTest) Disconnected(nodeID ids.NodeID) error {
	if e.DisconnectedF != nil {
		return e.DisconnectedF(nodeID)
	}
	if !e.CantDisconnected {
		return nil
	}
	if e.T != nil {
		e.T.Fatal(errDisconnected)
	}
	return errDisconnected
}

func (e *EngineTest) HealthCheck() (interface{}, error) {
	if e.HealthF != nil {
		return e.HealthF()
	}
	if e.CantHealth && e.T != nil {
		e.T.Fatal(errHealthCheck)
	}
	return nil, errHealthCheck
}

func (e *EngineTest) GetVM() VM {
	if e.GetVMF != nil {
		return e.GetVMF()
	}
	if e.CantGetVM && e.T != nil {
		e.T.Fatalf("Unexpectedly called GetVM")
	}
	return nil
}
