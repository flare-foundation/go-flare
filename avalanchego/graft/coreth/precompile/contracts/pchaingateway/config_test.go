// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package pchaingateway

import (
	"testing"

	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"

	"github.com/ava-labs/avalanchego/graft/coreth/precompile/precompileconfig"
	"github.com/ava-labs/avalanchego/graft/coreth/precompile/precompiletest"
	"github.com/ava-labs/avalanchego/graft/evm/utils"
)

func TestVerify(t *testing.T) {
	tests := map[string]precompiletest.ConfigVerifyTest{
		"valid config": {
			Config:        NewDefaultConfig(utils.PointerTo[uint64](3)),
			ExpectedError: nil,
		},
		"valid disable config": {
			Config:        NewDisableConfig(utils.PointerTo[uint64](3)),
			ExpectedError: nil,
		},
		"missing timestamp": {
			Config:        &Config{},
			ExpectedError: errNoTimestamp,
		},
	}
	precompiletest.RunVerifyTests(t, tests)
}

func TestEqual(t *testing.T) {
	tests := map[string]precompiletest.ConfigEqualTest{
		"non-nil config and nil other": {
			Config:   NewDefaultConfig(utils.PointerTo[uint64](3)),
			Other:    nil,
			Expected: false,
		},
		"different type": {
			Config:   NewDefaultConfig(utils.PointerTo[uint64](3)),
			Other:    precompileconfig.NewMockConfig(gomock.NewController(t)),
			Expected: false,
		},
		"different timestamp": {
			Config:   NewDefaultConfig(utils.PointerTo[uint64](3)),
			Other:    NewDefaultConfig(utils.PointerTo[uint64](4)),
			Expected: false,
		},
		"enable vs disable": {
			Config:   NewDefaultConfig(utils.PointerTo[uint64](3)),
			Other:    NewDisableConfig(utils.PointerTo[uint64](3)),
			Expected: false,
		},
		"same config": {
			Config:   NewDefaultConfig(utils.PointerTo[uint64](3)),
			Other:    NewDefaultConfig(utils.PointerTo[uint64](3)),
			Expected: true,
		},
	}
	precompiletest.RunEqualTests(t, tests)
}

func TestKey(t *testing.T) {
	require.Equal(t, ConfigKey, NewDefaultConfig(nil).Key())
	require.Equal(t, ContractAddress, Module.Address)
}
