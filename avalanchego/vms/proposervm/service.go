// Copyright (C) 2019, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package proposervm

import (
	"context"
	"encoding/hex"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"connectrpc.com/connect"
	"go.uber.org/zap"

	"github.com/ava-labs/avalanchego/api"
	"github.com/ava-labs/avalanchego/api/server"
	"github.com/ava-labs/avalanchego/connectproto/pb/proposervm/proposervmconnect"
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/proposervm/acp181"
	"github.com/ava-labs/avalanchego/vms/proposervm/block"

	pb "github.com/ava-labs/avalanchego/connectproto/pb/proposervm"
	avajson "github.com/ava-labs/avalanchego/utils/json"
)

var _ proposervmconnect.ProposerVMHandler = (*connectrpcService)(nil)

type connectrpcService struct {
	vm *VM
}

func (c *connectrpcService) GetProposedHeight(ctx context.Context, r *connect.Request[pb.GetProposedHeightRequest]) (*connect.Response[pb.GetProposedHeightReply], error) {
	log := c.vm.ctx.Log.With(
		zap.String("service", "proposervm"),
		zap.String("method", "GetProposedHeight"),
		zap.Strings("route", r.Header()[server.HTTPHeaderRoute]),
	)
	log.Debug("API called")

	c.vm.ctx.Lock.Lock()
	defer c.vm.ctx.Lock.Unlock()

	blk, err := c.vm.getBlock(ctx, c.vm.preferred)
	if err != nil {
		log.Error("failed to get preferred block", zap.Error(err))
		return nil, fmt.Errorf("failed to get preferred block: %w", err)
	}

	height, err := blk.selectChildPChainHeight(ctx)
	if err != nil {
		log.Error("failed to get child p-chain height", zap.Error(err))
		return nil, fmt.Errorf("failed to get child p-chain height: %w", err)
	}

	return connect.NewResponse(&pb.GetProposedHeightReply{
		Height: height,
	}), nil
}

func (c *connectrpcService) GetCurrentEpoch(ctx context.Context, r *connect.Request[pb.GetCurrentEpochRequest]) (*connect.Response[pb.GetCurrentEpochReply], error) {
	c.vm.ctx.Log.Debug("API called",
		zap.String("service", "proposervm"),
		zap.String("method", "getCurrentEpoch"),
		zap.Strings("route", r.Header()[server.HTTPHeaderRoute]),
	)

	epoch, err := c.vm.getCurrentEpoch(ctx)
	if err != nil {
		return nil, fmt.Errorf("couldn't get current epoch: %w", err)
	}

	return connect.NewResponse(&pb.GetCurrentEpochReply{
		Number:       epoch.Number,
		StartTime:    epoch.StartTime,
		PChainHeight: epoch.PChainHeight,
	}), nil
}

type jsonrpcService struct {
	vm *VM
}

func (j *jsonrpcService) GetProposedHeight(r *http.Request, _ *struct{}, reply *api.GetHeightResponse) error {
	log := j.vm.ctx.Log.With(
		zap.String("service", "proposervm"),
		zap.String("method", "GetProposedHeight"),
		zap.String("path", r.URL.Path),
	)
	log.Debug("API called")

	j.vm.ctx.Lock.Lock()
	defer j.vm.ctx.Lock.Unlock()

	ctx := r.Context()
	blk, err := j.vm.getBlock(ctx, j.vm.preferred)
	if err != nil {
		log.Error("failed to get preferred block", zap.Error(err))
		return fmt.Errorf("failed to get preferred block: %w", err)
	}

	height, err := blk.selectChildPChainHeight(ctx)
	if err != nil {
		log.Error("failed to get child p-chain height", zap.Error(err))
		return fmt.Errorf("failed to get child p-chain height: %w", err)
	}

	reply.Height = avajson.Uint64(height)
	return nil
}

type GetEpochResponse struct {
	Number       avajson.Uint64 `json:"number"`
	StartTime    avajson.Uint64 `json:"startTime"`
	PChainHeight avajson.Uint64 `json:"pChainHeight"`
}

func (j *jsonrpcService) GetCurrentEpoch(r *http.Request, _ *struct{}, reply *GetEpochResponse) error {
	j.vm.ctx.Log.Debug("API called",
		zap.String("service", "proposervm"),
		zap.String("method", "getCurrentEpoch"),
	)

	epoch, err := j.vm.getCurrentEpoch(r.Context())
	if err != nil {
		return fmt.Errorf("couldn't get current epoch: %w", err)
	}

	reply.Number = avajson.Uint64(epoch.Number)
	reply.StartTime = avajson.Uint64(epoch.StartTime)
	reply.PChainHeight = avajson.Uint64(epoch.PChainHeight)
	return nil
}

// ProposerStatus describes whether a proposer could be resolved for a given
// EVM block hash, and if not, why not.
type ProposerStatus string

const (
	ProposerStatusOK           ProposerStatus = "ok"
	ProposerStatusPreFork      ProposerStatus = "preFork"
	ProposerStatusNoProposer   ProposerStatus = "noProposer"
	ProposerStatusPruned       ProposerStatus = "pruned"
	ProposerStatusNotCanonical ProposerStatus = "notCanonical"
	ProposerStatusNotFound     ProposerStatus = "notFound"
)

type GetProposerByEVMBlockHashArgs struct {
	BlockHash string `json:"blockHash"`
}

const blockNumberLatest = "latest"

type GetProposerByEVMBlockNumberArgs struct {
	// BlockNumber is one of:
	//   - decimal number as JSON string, e.g. "12345"
	//   - 0x-prefixed hex number, e.g. "0x3039"
	//   - the tag "latest"
	BlockNumber string `json:"blockNumber"`
}

type GetProposerResponse struct {
	NodeID            ids.NodeID     `json:"nodeID"`
	Status            ProposerStatus `json:"status"`
	EVMBlockHash      string         `json:"evmBlockHash"` // 0x-prefixed hex
	Height            avajson.Uint64 `json:"height"`
	PChainHeight      avajson.Uint64 `json:"pChainHeight"`
	Timestamp         avajson.Uint64 `json:"timestamp"`
	ProposerVMBlockID ids.ID         `json:"proposerVMBlockID"`
}

// evmBlockHashString renders an inner block ID as a 0x-prefixed 32-byte hex
// string — the same format eth_getBlockBy{Hash,Number} returns for block
// hashes, so callers can correlate results across RPCs.
func evmBlockHashString(innerID ids.ID) string {
	return "0x" + hex.EncodeToString(innerID[:])
}

func (j *jsonrpcService) GetProposerByEVMBlockHash(r *http.Request, args *GetProposerByEVMBlockHashArgs, reply *GetProposerResponse) error {
	log := j.vm.ctx.Log.With(
		zap.String("service", "proposervm"),
		zap.String("method", "GetProposerByEVMBlockHash"),
		zap.String("blockHash", args.BlockHash),
	)
	log.Debug("API called")

	hashStr := strings.TrimPrefix(args.BlockHash, "0x")
	hashStr = strings.TrimPrefix(hashStr, "0X")
	hashBytes, err := hex.DecodeString(hashStr)
	if err != nil {
		return fmt.Errorf("invalid block hash %q: %w", args.BlockHash, err)
	}
	innerID, err := ids.ToID(hashBytes)
	if err != nil {
		return fmt.Errorf("invalid block hash %q: %w", args.BlockHash, err)
	}

	j.vm.ctx.Lock.Lock()
	defer j.vm.ctx.Lock.Unlock()

	resp, err := j.vm.getProposerByInnerBlockID(r.Context(), innerID)
	if err != nil {
		return err
	}
	*reply = resp
	return nil
}

func (j *jsonrpcService) GetProposerByEVMBlockNumber(r *http.Request, args *GetProposerByEVMBlockNumberArgs, reply *GetProposerResponse) error {
	log := j.vm.ctx.Log.With(
		zap.String("service", "proposervm"),
		zap.String("method", "GetProposerByEVMBlockNumber"),
		zap.String("blockNumber", args.BlockNumber),
	)
	log.Debug("API called")

	j.vm.ctx.Lock.Lock()
	defer j.vm.ctx.Lock.Unlock()

	var height uint64
	switch raw := strings.TrimSpace(args.BlockNumber); {
	case strings.EqualFold(raw, blockNumberLatest):
		height = j.vm.lastAcceptedHeight
	case strings.HasPrefix(raw, "0x") || strings.HasPrefix(raw, "0X"):
		h, err := strconv.ParseUint(raw[2:], 16, 64)
		if err != nil {
			return fmt.Errorf("invalid hex block number %q: %w", args.BlockNumber, err)
		}
		height = h
	default:
		h, err := strconv.ParseUint(raw, 10, 64)
		if err != nil {
			return fmt.Errorf("invalid block number %q: %w", args.BlockNumber, err)
		}
		height = h
	}

	resp, err := j.vm.getProposerByHeight(r.Context(), height)
	if err != nil {
		return err
	}
	*reply = resp
	return nil
}

// getProposerByInnerBlockID resolves the proposer of the inner (EVM) block
// whose ID equals innerID. It returns a populated response with Status
// describing the outcome; only unexpected database errors return a non-nil
// error.
func (vm *VM) getProposerByInnerBlockID(ctx context.Context, innerID ids.ID) (GetProposerResponse, error) {
	innerBlk, err := vm.ChainVM.GetBlock(ctx, innerID)
	if errors.Is(err, database.ErrNotFound) {
		return GetProposerResponse{
			Status:       ProposerStatusNotFound,
			EVMBlockHash: evmBlockHashString(innerID),
		}, nil
	}
	if err != nil {
		return GetProposerResponse{}, fmt.Errorf("failed to fetch inner block: %w", err)
	}
	height := innerBlk.Height()

	canonicalID, err := vm.ChainVM.GetBlockIDAtHeight(ctx, height)
	switch {
	case errors.Is(err, database.ErrNotFound), err == nil && canonicalID != innerID:
		return GetProposerResponse{
			Status:       ProposerStatusNotCanonical,
			EVMBlockHash: evmBlockHashString(innerID),
			Height:       avajson.Uint64(height),
		}, nil
	case err != nil:
		return GetProposerResponse{}, fmt.Errorf("failed to fetch canonical block at height %d: %w", height, err)
	}

	return vm.resolveProposerAtHeight(height, innerID)
}

// getProposerByHeight resolves the proposer for the canonical EVM block at
// the given height. Returns Status=notFound when no canonical block exists
// at that height yet.
func (vm *VM) getProposerByHeight(ctx context.Context, height uint64) (GetProposerResponse, error) {
	innerID, err := vm.ChainVM.GetBlockIDAtHeight(ctx, height)
	if errors.Is(err, database.ErrNotFound) {
		return GetProposerResponse{
			Status: ProposerStatusNotFound,
			Height: avajson.Uint64(height),
		}, nil
	}
	if err != nil {
		return GetProposerResponse{}, fmt.Errorf("failed to fetch canonical block at height %d: %w", height, err)
	}
	return vm.resolveProposerAtHeight(height, innerID)
}

// resolveProposerAtHeight fills a GetProposerResponse for the canonical inner
// block at the given height. evmBlockHash must already be the canonical inner
// ID at that height (verified by the caller).
func (vm *VM) resolveProposerAtHeight(height uint64, evmBlockHash ids.ID) (GetProposerResponse, error) {
	resp := GetProposerResponse{
		EVMBlockHash: evmBlockHashString(evmBlockHash),
		Height:       avajson.Uint64(height),
	}

	forkHeight, err := vm.State.GetForkHeight()
	switch {
	case errors.Is(err, database.ErrNotFound), err == nil && height < forkHeight:
		resp.Status = ProposerStatusPreFork
		return resp, nil
	case err != nil:
		return GetProposerResponse{}, fmt.Errorf("failed to fetch fork height: %w", err)
	}

	outerID, err := vm.State.GetBlockIDAtHeight(height)
	if errors.Is(err, database.ErrNotFound) {
		resp.Status = ProposerStatusPruned
		return resp, nil
	}
	if err != nil {
		return GetProposerResponse{}, fmt.Errorf("failed to fetch proposerVM block ID at height %d: %w", height, err)
	}
	resp.ProposerVMBlockID = outerID

	outerBlock, err := vm.State.GetBlock(outerID)
	if errors.Is(err, database.ErrNotFound) {
		resp.Status = ProposerStatusPruned
		return resp, nil
	}
	if err != nil {
		return GetProposerResponse{}, fmt.Errorf("failed to fetch proposerVM block %s: %w", outerID, err)
	}

	signedBlock, ok := outerBlock.(block.SignedBlock)
	if !ok {
		// Post-fork height (we passed the forkHeight check above), but the
		// outer block is unsigned — e.g. an oracle option block. C-chain
		// doesn't use option blocks in practice, but distinguish this from
		// a true pre-fork block so callers can tell them apart.
		resp.Status = ProposerStatusNoProposer
		return resp, nil
	}

	resp.PChainHeight = avajson.Uint64(signedBlock.PChainHeight())
	resp.Timestamp = avajson.Uint64(signedBlock.Timestamp().Unix())

	proposer := signedBlock.Proposer()
	if proposer == ids.EmptyNodeID {
		resp.Status = ProposerStatusNoProposer
		return resp, nil
	}

	resp.NodeID = proposer
	resp.Status = ProposerStatusOK
	return resp, nil
}

func (vm *VM) getCurrentEpoch(ctx context.Context) (block.Epoch, error) {
	vm.ctx.Lock.Lock()
	defer vm.ctx.Lock.Unlock()

	blk, err := vm.getBlock(ctx, vm.preferred)
	if err != nil {
		return block.Epoch{}, fmt.Errorf("couldn't get preferred block: %w", err)
	}

	epoch, err := blk.pChainEpoch(ctx)
	if err != nil {
		return block.Epoch{}, fmt.Errorf("couldn't get preferred block epoch: %w", err)
	}

	pChainHeight, err := blk.pChainHeight(ctx)
	if err != nil {
		return block.Epoch{}, fmt.Errorf("couldn't get preferred block p-chain height: %w", err)
	}

	timestamp := blk.Timestamp()
	newTimestamp := vm.Time().Truncate(time.Second)
	if newTimestamp.Before(timestamp) {
		newTimestamp = timestamp
	}

	return acp181.NewEpoch(
		vm.Upgrades,
		pChainHeight,
		epoch,
		timestamp,
		newTimestamp,
	), nil
}
