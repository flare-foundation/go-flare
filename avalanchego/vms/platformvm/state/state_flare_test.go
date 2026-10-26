// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package state

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/database/memdb"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/choices"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/vms/components/avax"
	"github.com/ava-labs/avalanchego/vms/platformvm/platform"
	"github.com/ava-labs/avalanchego/vms/platformvm/status"
	"github.com/ava-labs/avalanchego/vms/secp256k1fx"
)

// TestGetBlockBytes covers every place a block can live: added but not yet
// written, cached, stored raw, and stored in the legacy stateBlk format.
func TestGetBlockBytes(t *testing.T) {
	require := require.New(t)
	s := newTestState(t, memdb.New())
	blks := makeBlocks(require)
	require.GreaterOrEqual(len(blks), 3)

	added, raw, legacy := blks[0], blks[1], blks[2]

	s.AddStatelessBlock(added)
	rawID := raw.ID()
	require.NoError(s.blockDB.Put(rawID[:], raw.Bytes()))
	legacyID := legacy.ID()
	legacyBytes, err := platform.GenesisCodec.Marshal(platform.CodecVersion, &stateBlk{
		Bytes:  legacy.Bytes(),
		Status: choices.Accepted,
	})
	require.NoError(err)
	require.NoError(s.blockDB.Put(legacyID[:], legacyBytes))

	got, err := s.GetBlockBytes(added.ID())
	require.NoError(err)
	require.Equal(added.Bytes(), got)

	got, err = s.GetBlockBytes(rawID)
	require.NoError(err)
	require.Equal(raw.Bytes(), got)

	got, err = s.GetBlockBytes(legacyID)
	require.NoError(err)
	require.Equal(legacy.Bytes(), got)

	// Once GetStatelessBlock cached the parsed block, the bytes still match.
	_, err = s.GetStatelessBlock(rawID)
	require.NoError(err)
	got, err = s.GetBlockBytes(rawID)
	require.NoError(err)
	require.Equal(raw.Bytes(), got)

	_, err = s.GetBlockBytes(ids.GenerateTestID())
	require.ErrorIs(err, database.ErrNotFound)

	// The bytes parse to the same block as GetStatelessBlock returns.
	for _, blk := range blks[:3] {
		got, err := s.GetBlockBytes(blk.ID())
		require.NoError(err)
		parsed, err := platform.ParseBlock(platform.GenesisCodec, got)
		require.NoError(err)
		require.Equal(blk.ID(), parsed.ID())
	}
}

func TestGetTxBytes(t *testing.T) {
	require := require.New(t)
	s := newTestState(t, memdb.New())

	newTx := func() *platform.Tx {
		tx, err := platform.NewSignedTx(&platform.BaseTx{BaseTx: avax.BaseTx{
			NetworkID:    constants.UnitTestID,
			BlockchainID: constants.PlatformChainID,
			Outs: []*avax.TransferableOutput{{
				Asset: avax.Asset{ID: ids.GenerateTestID()},
				Out:   &secp256k1fx.TransferOutput{Amt: 1, OutputOwners: secp256k1fx.OutputOwners{Threshold: 1, Addrs: []ids.ShortID{ids.GenerateTestShortID()}}},
			}},
		}}, platform.Codec, nil)
		require.NoError(err)
		return tx
	}
	added, stored := newTx(), newTx()

	s.AddTx(added, status.Committed)
	storedBytes, err := platform.GenesisCodec.Marshal(platform.CodecVersion, &txBytesAndStatus{
		Tx:     stored.Bytes(),
		Status: status.Committed,
	})
	require.NoError(err)
	storedID := stored.ID()
	require.NoError(s.txDB.Put(storedID[:], storedBytes))

	got, err := s.GetTxBytes(added.ID())
	require.NoError(err)
	require.Equal(added.Bytes(), got)

	got, err = s.GetTxBytes(storedID)
	require.NoError(err)
	require.Equal(stored.Bytes(), got)

	// Cached by GetTx afterwards; the bytes still match and parse to the same tx.
	_, _, err = s.GetTx(storedID)
	require.NoError(err)
	got, err = s.GetTxBytes(storedID)
	require.NoError(err)
	parsed, err := platform.ParseTx(platform.GenesisCodec, got)
	require.NoError(err)
	require.Equal(storedID, parsed.ID())

	_, err = s.GetTxBytes(ids.GenerateTestID())
	require.ErrorIs(err, database.ErrNotFound)
}
