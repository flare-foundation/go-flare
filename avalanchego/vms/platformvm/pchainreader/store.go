// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package pchainreader

import (
	"sync"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/platformvm/state"
)

// Store is the raw access to accepted P-chain state the [Reader] needs.
// *state.State satisfies it. The P-chain state has no internal
// synchronization and mutates its caches on reads, so a Store used from
// another chain must be wrapped with [NewLockedStore]. Blocks and
// transactions are handed out as stored bytes so that the Reader parses them
// outside the lock and after their size has been accounted for.
type Store interface {
	GetLastAccepted() ids.ID
	GetBlockIDAtHeight(height uint64) (ids.ID, error)
	GetBlockBytes(blockID ids.ID) ([]byte, error)
	GetTxBytes(txID ids.ID) ([]byte, error)
	GetRewardUTXOs(txID ids.ID) ([]*avax.UTXO, error)
}

var _ Store = (*state.State)(nil)

type lockedStore struct {
	lock sync.Locker
	s    Store
}

// NewLockedStore wraps s so that every call holds lock, mirroring
// validators.NewLockedState. The lock is held per call only, never across
// decoding, so P-chain consensus interleaves freely with gateway reads.
func NewLockedStore(lock sync.Locker, s Store) Store {
	return &lockedStore{lock: lock, s: s}
}

func (l *lockedStore) GetLastAccepted() ids.ID {
	l.lock.Lock()
	defer l.lock.Unlock()

	return l.s.GetLastAccepted()
}

func (l *lockedStore) GetBlockIDAtHeight(height uint64) (ids.ID, error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	return l.s.GetBlockIDAtHeight(height)
}

func (l *lockedStore) GetBlockBytes(blockID ids.ID) ([]byte, error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	return l.s.GetBlockBytes(blockID)
}

func (l *lockedStore) GetTxBytes(txID ids.ID) ([]byte, error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	return l.s.GetTxBytes(txID)
}

func (l *lockedStore) GetRewardUTXOs(txID ids.ID) ([]*avax.UTXO, error) {
	l.lock.Lock()
	defer l.lock.Unlock()

	return l.s.GetRewardUTXOs(txID)
}
