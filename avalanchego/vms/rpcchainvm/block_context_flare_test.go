// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package rpcchainvm

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/snow/engine/snowman/block"
)

// TestBlockContextRequestRoundTrip guards the gRPC transport of block.Context:
// every exported field must survive blockContextRequestFields followed by
// blockContextFromRequest, and adding a field must force updating both.
func TestBlockContextRequestRoundTrip(t *testing.T) {
	require := require.New(t)

	exported := 0
	typ := reflect.TypeOf(block.Context{})
	for i := range typ.NumField() {
		if typ.Field(i).IsExported() {
			exported++
		}
	}
	require.Equal(2, exported, "block.Context gained a field: extend blockContextRequestFields, blockContextFromRequest and this test")

	want := &block.Context{
		PChainHeight:         1,
		ProposerPChainHeight: 2,
	}
	pChainHeight, proposerPChainHeight := blockContextRequestFields(want)
	require.Equal(want, blockContextFromRequest(pChainHeight, proposerPChainHeight))

	// Clients without the Flare field leave it unset.
	require.Equal(&block.Context{PChainHeight: 1}, blockContextFromRequest(pChainHeight, nil))
}
