// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package transitionvm

import (
	"reflect"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/snow"
	"github.com/ava-labs/avalanchego/snow/snowtest"
	"github.com/ava-labs/avalanchego/vms/platformvm/pchainreader"
)

// TestCopyContextCopiesEveryField guards copyContext against new snow.Context
// fields: every field except the lock must be carried over to the
// post-transition chain. The P-chain gateway reader was missed once, which
// left the SAE C-chain without P-chain access on a live network.
func TestCopyContextCopiesEveryField(t *testing.T) {
	require := require.New(t)

	src := snowtest.Context(t, snowtest.CChainID)
	src.PChainReader = pchainreader.New(nil, nil, src.NetworkUpgrades)

	got := copyContext(src)

	srcV, gotV := reflect.ValueOf(src).Elem(), reflect.ValueOf(got).Elem()
	typ := reflect.TypeOf(snow.Context{})
	for i := range typ.NumField() {
		field := typ.Field(i)
		if field.Name == "Lock" {
			continue // deliberately not copied
		}
		require.Truef(
			srcV.Field(i).IsValid() && reflect.DeepEqual(srcV.Field(i).Interface(), gotV.Field(i).Interface()),
			"snow.Context.%s is not copied by copyContext", field.Name,
		)
		if field.Type.Kind() == reflect.Interface || field.Type.Kind() == reflect.Pointer {
			require.Falsef(srcV.Field(i).IsNil(), "snow.Context.%s is nil in the test fixture; populate it so the copy is checked", field.Name)
		}
	}
}
