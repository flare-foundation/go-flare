// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package core

import (
	"os"
	"testing"

	"github.com/ava-labs/avalanchego/graft/coreth/core/extstate"
	"github.com/ava-labs/avalanchego/graft/coreth/params"
	"github.com/ava-labs/avalanchego/graft/coreth/plugin/evm/customtypes"
)

// TestMain registers the C-Chain's libevm hooks and payloads, as
// vms/saevm/cchain/libevm.RegisterExtras does for the node, so that the legacy
// StateTransition under test resolves the coreth rules extras.
func TestMain(m *testing.M) {
	RegisterExtras()
	customtypes.Register()
	extstate.RegisterExtras()
	params.RegisterExtras()
	os.Exit(m.Run())
}
