// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package customheader

import (
	"testing"

	"github.com/ava-labs/libevm/core/types"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/graft/coreth/plugin/evm/customtypes"
)

func pChainHeightHeader(extra *customtypes.HeaderExtra) *types.Header {
	return customtypes.WithHeaderExtra(
		&types.Header{},
		extra,
	)
}

func TestVerifyPChainHeight(t *testing.T) {
	tests := []struct {
		name    string
		header  *types.Header
		wantErr error
	}{
		{
			name:    "p_chain_height_should_fail",
			header:  pChainHeightHeader(&customtypes.HeaderExtra{PChainHeight: new(uint64(1))}),
			wantErr: errPChainHeightSet,
		},
		{
			name:    "zero_p_chain_height_should_fail",
			header:  pChainHeightHeader(&customtypes.HeaderExtra{PChainHeight: new(uint64(0))}),
			wantErr: errPChainHeightSet,
		},
		{
			name: "settled_fields_without_p_chain_height_should_work",
			header: pChainHeightHeader(&customtypes.HeaderExtra{
				SettledHeight: new(uint64(1)),
			}),
		},
		{
			name:   "nil_should_work",
			header: pChainHeightHeader(&customtypes.HeaderExtra{}),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := VerifyPChainHeight(tt.header)
			require.ErrorIs(t, err, tt.wantErr, "VerifyPChainHeight()")
		})
	}
}
