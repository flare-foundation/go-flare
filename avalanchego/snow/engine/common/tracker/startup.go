// Copyright (C) 2019-2024, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package tracker

import (
	"context"
	"math/big"
	"sync"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/crypto/bls"
	"github.com/ava-labs/avalanchego/version"
)

var _ Startup = (*startup)(nil)

type Startup interface {
	Peers

	ShouldStart() bool
}

type startup struct {
	Peers

	lock          sync.RWMutex
	startupWeight *big.Int
	shouldStart   bool
}

func NewStartup(peers Peers, startupWeight *big.Int) Startup {
	return &startup{
		Peers:         peers,
		startupWeight: startupWeight,
		shouldStart:   peers.ConnectedWeight().Cmp(startupWeight) >= 0,
	}
}

func (s *startup) OnValidatorAdded(nodeID ids.NodeID, pk *bls.PublicKey, txID ids.ID, weight uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.Peers.OnValidatorAdded(nodeID, pk, txID, weight)
	s.shouldStart = s.shouldStart || s.Peers.ConnectedWeight().Cmp(s.startupWeight) >= 0
}

func (s *startup) OnValidatorWeightChanged(nodeID ids.NodeID, oldWeight, newWeight uint64) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.Peers.OnValidatorWeightChanged(nodeID, oldWeight, newWeight)
	s.shouldStart = s.shouldStart || s.Peers.ConnectedWeight().Cmp(s.startupWeight) >= 0
}

func (s *startup) Connected(ctx context.Context, nodeID ids.NodeID, nodeVersion *version.Application) error {
	s.lock.Lock()
	defer s.lock.Unlock()

	if err := s.Peers.Connected(ctx, nodeID, nodeVersion); err != nil {
		return err
	}

	s.shouldStart = s.shouldStart || s.Peers.ConnectedWeight().Cmp(s.startupWeight) >= 0
	return nil
}

func (s *startup) ShouldStart() bool {
	s.lock.RLock()
	defer s.lock.RUnlock()

	return s.shouldStart
}
