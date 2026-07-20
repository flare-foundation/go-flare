// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package vm

import (
	"testing"

	"github.com/ava-labs/avalanchego/api"
	"github.com/ava-labs/avalanchego/database"
	"github.com/ava-labs/avalanchego/database/memdb"
	"github.com/ava-labs/avalanchego/database/versiondb"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/snow/snowtest"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/graft/coreth/plugin/evm/atomic"
	atomicstate "github.com/ava-labs/avalanchego/graft/coreth/plugin/evm/atomic/state"
	"github.com/ava-labs/avalanchego/graft/coreth/plugin/evm/client"
)

func TestGetAtomicTxStatusPropagatesRepositoryError(t *testing.T) {
	require := require.New(t)

	db := versiondb.New(memdb.New())
	repository, err := atomicstate.NewAtomicTxRepository(db, atomic.Codec, 0)
	require.NoError(err)
	require.NoError(db.Close())

	service := &AvaxAPI{
		Context:     snowtest.Context(t, snowtest.CChainID),
		AcceptedTxs: repository,
	}
	reply := &client.GetAtomicTxStatusReply{}
	err = service.GetAtomicTxStatus(nil, &api.JSONTxID{
		TxID: ids.GenerateTestID(),
	}, reply)
	require.ErrorIs(err, database.ErrClosed)
}
