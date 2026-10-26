// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

// Package pchainreader implements snow/pchain.Reader on top of the P-chain
// state: it decodes accepted blocks into the gateway's plain view and resolves
// the owners of consumed inputs from the transactions that produced them.
package pchainreader

import (
	"errors"
	"fmt"
	"sync"

	"github.com/ava-labs/avalanchego/cache/lru"
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/pchain"
	"github.com/ava-labs/avalanchego/upgrade"
	"github.com/ava-labs/avalanchego/utils"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/platformvm/platform"
)

const (
	// decodedBlockCacheSize bounds the number of decoded blocks kept in
	// memory. Blocks are immutable once accepted, so entries never go stale.
	decodedBlockCacheSize = 256
	// ownerCacheSize bounds the number of resolved consumed-output owners.
	ownerCacheSize = 8192
)

var _ pchain.Reader = (*Reader)(nil)

type lastAccepted struct {
	id     ids.ID
	height uint64
}

// Reader implements [pchain.Reader]. It is safe for concurrent use; all
// P-chain state access goes through the (locked) [Store].
//
// A Reader may be constructed before the P-chain VM is initialized (the chain
// manager wires it while creating the P-chain), so the store and the genesis
// UTXOs are only consulted when a method is called.
type Reader struct {
	store    Store
	upgrades upgrade.Config

	genesisUTXOs func() []*avax.UTXO
	genesisOnce  sync.Once
	genesis      map[utxoKey]*avax.UTXO // genesis UTXOs, which have no producing tx

	lastAccepted utils.Atomic[lastAccepted]
	blocks       *lru.Cache[uint64, *decodedBlock]
	owners       *lru.Cache[utxoKey, ownerEntry]
}

// New returns a Reader over store. genesisUTXOs returns the P-chain genesis
// UTXOs, which have no producing transaction (on production networks their
// transaction ID is ids.Empty); it is called once, on first use. upgrades
// selects the Durango start-time rule.
func New(store Store, genesisUTXOs func() []*avax.UTXO, upgrades upgrade.Config) *Reader {
	return &Reader{
		store:        store,
		upgrades:     upgrades,
		genesisUTXOs: genesisUTXOs,
		blocks:       lru.NewCache[uint64, *decodedBlock](decodedBlockCacheSize),
		owners:       lru.NewCache[utxoKey, ownerEntry](ownerCacheSize),
	}
}

// genesisUTXO returns the genesis UTXO consumed by key, if any.
func (r *Reader) genesisUTXO(key utxoKey) (*avax.UTXO, bool) {
	r.genesisOnce.Do(func() {
		var utxos []*avax.UTXO
		if r.genesisUTXOs != nil {
			utxos = r.genesisUTXOs()
		}
		r.genesis = make(map[utxoKey]*avax.UTXO, len(utxos))
		for _, utxo := range utxos {
			if utxo != nil {
				r.genesis[utxoKey{txID: utxo.TxID, index: utxo.OutputIndex}] = utxo
			}
		}
	})
	utxo, ok := r.genesis[key]
	return utxo, ok
}

// chargeBytes applies a possibly nil [pchain.ByteCharger].
func chargeBytes(charge pchain.ByteCharger, bytes uint64) error {
	if charge == nil {
		return nil
	}
	return charge(bytes)
}

// LastAcceptedHeight returns the height of the last accepted block. The
// height is read from the accepted-height index, never by parsing the tip
// block, so a tip change costs a few index lookups (one lock acquisition
// each) instead of an unmetered parse of an arbitrarily large block:
// accepted heights are contiguous from genesis, so the tip is the highest
// indexed height, found by galloping up from the last known height and
// bisecting. The result is cached until the tip changes.
func (r *Reader) LastAcceptedHeight() (uint64, error) {
	id := r.store.GetLastAccepted()
	known := r.lastAccepted.Get()
	if known.id == id {
		return known.height, nil
	}
	height, err := r.highestIndexedHeight(known.height)
	if err != nil {
		return 0, err
	}
	// Heights only grow, so height is an accepted height even if the tip
	// moved during the search; the (id, height) pair is cached only when it
	// is known to be consistent, otherwise the next call searches again.
	if tipID, err := r.store.GetBlockIDAtHeight(height); err == nil && tipID == id {
		r.lastAccepted.Set(lastAccepted{id: id, height: height})
	}
	return height, nil
}

// highestIndexedHeight returns the highest accepted height, given that from
// is accepted: O(log distance) index lookups, no block parsing.
func (r *Reader) highestIndexedHeight(from uint64) (uint64, error) {
	const maxHeight = ^uint64(0)
	lo, hi, step := from, from, uint64(1) // lo is accepted
	for {
		hi = lo + step
		if step == 0 || hi < lo { // overflow: nothing is accepted that high
			hi = maxHeight
		}
		accepted, err := r.heightAccepted(hi)
		if err != nil {
			return 0, err
		}
		if !accepted {
			break
		}
		if hi == maxHeight {
			return hi, nil
		}
		lo = hi
		step <<= 1
	}
	// lo is accepted, hi is not.
	for hi-lo > 1 {
		mid := lo + (hi-lo)/2
		accepted, err := r.heightAccepted(mid)
		if err != nil {
			return 0, err
		}
		if accepted {
			lo = mid
		} else {
			hi = mid
		}
	}
	return lo, nil
}

// heightAccepted reports whether the accepted-height index has an entry for
// height.
func (r *Reader) heightAccepted(height uint64) (bool, error) {
	_, err := r.store.GetBlockIDAtHeight(height)
	switch {
	case err == nil:
		return true, nil
	case errors.Is(err, database.ErrNotFound):
		return false, nil
	default:
		return false, fmt.Errorf("fetching P-chain block ID at height %d: %w", height, err)
	}
}

func (r *Reader) GetBlock(height uint64, charge pchain.ByteCharger) (pchain.BlockReader, error) {
	db, err := r.decodedBlock(height, charge)
	if err != nil {
		return nil, err
	}
	return &blockReader{r: r, db: db}, nil
}

// blockReader implements [pchain.BlockReader] over a decoded block. Decoded
// blocks are immutable, so the reader stays valid after the block leaves the
// cache.
type blockReader struct {
	r  *Reader
	db *decodedBlock
}

func (b *blockReader) Block() *pchain.Block {
	return &b.db.block
}

// decodedBlock returns the decoded block at height. The block's stored size
// is charged before it is parsed, and again on every cached read, so the
// charge never depends on the cache.
func (r *Reader) decodedBlock(height uint64, charge pchain.ByteCharger) (*decodedBlock, error) {
	if db, ok := r.blocks.Get(height); ok {
		if err := chargeBytes(charge, db.bytes); err != nil {
			return nil, err
		}
		return db, nil
	}

	blkID, err := r.store.GetBlockIDAtHeight(height)
	if errors.Is(err, database.ErrNotFound) {
		// Accepted heights are contiguous from genesis, so a height missing
		// from the index is above the local tip; a hole below the tip is a
		// local database problem.
		last, lastErr := r.LastAcceptedHeight()
		if lastErr != nil {
			return nil, lastErr
		}
		if height > last {
			return nil, fmt.Errorf("%w: height %d, last accepted %d", pchain.ErrNotAccepted, height, last)
		}
		return nil, fmt.Errorf("fetching P-chain block ID at height %d (last accepted %d): %w", height, last, err)
	}
	if err != nil {
		return nil, fmt.Errorf("fetching P-chain block ID at height %d: %w", height, err)
	}
	blkBytes, err := r.store.GetBlockBytes(blkID)
	if err != nil {
		return nil, fmt.Errorf("fetching P-chain block %s at height %d: %w", blkID, height, err)
	}
	size := uint64(len(blkBytes))
	if err := chargeBytes(charge, size); err != nil {
		return nil, err
	}

	// Parsed outside the P-chain lock, after the caller paid for the size.
	blk, err := platform.ParseBlock(platform.GenesisCodec, blkBytes)
	if err != nil {
		return nil, fmt.Errorf("parsing P-chain block %s at height %d: %w", blkID, height, err)
	}
	db := r.decode(blk)
	db.bytes = size
	r.blocks.Put(height, db)
	return db, nil
}
