// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package pchaingateway

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/graft/evm/utils"
)

func TestKey(t *testing.T) {
	require.Equal(t, ConfigKey, NewDefaultConfig(nil).Key())
	require.Equal(t, ContractAddress, Module.Address)
}

func TestTimestampAndDisable(t *testing.T) {
	enabled := NewDefaultConfig(utils.PointerTo[uint64](3))
	require.Equal(t, uint64(3), *enabled.Timestamp())
	require.False(t, enabled.IsDisabled())

	disabled := NewDisableConfig(utils.PointerTo[uint64](4))
	require.Equal(t, uint64(4), *disabled.Timestamp())
	require.True(t, disabled.IsDisabled())
}
