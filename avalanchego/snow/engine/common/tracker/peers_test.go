// Copyright (C) 2019, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package tracker

import (
	"math/big"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/version"
)

func TestPeers(t *testing.T) {
	require := require.New(t)

	nodeID := ids.GenerateTestNodeID()

	p := NewPeers()

	require.Equal(big.NewInt(0), p.ConnectedWeight())

	p.OnValidatorAdded(nodeID, nil, ids.Empty, 5)
	require.Equal(big.NewInt(0), p.ConnectedWeight())

	require.NoError(p.Connected(t.Context(), nodeID, version.Current))
	require.Equal(big.NewInt(5), p.ConnectedWeight())

	p.OnValidatorWeightChanged(nodeID, 5, 10)
	require.Equal(big.NewInt(10), p.ConnectedWeight())

	p.OnValidatorRemoved(nodeID, 10)
	require.Equal(big.NewInt(0), p.ConnectedWeight())

	p.OnValidatorAdded(nodeID, nil, ids.Empty, 5)
	require.Equal(big.NewInt(5), p.ConnectedWeight())

	require.NoError(p.Disconnected(t.Context(), nodeID))
	require.Equal(big.NewInt(0), p.ConnectedWeight())
}
