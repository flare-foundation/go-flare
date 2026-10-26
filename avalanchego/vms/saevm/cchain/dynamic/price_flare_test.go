// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package dynamic

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/graft/coreth/plugin/evm/upgrade/granite"
	"github.com/ava-labs/avalanchego/vms/components/gas"
)

// TestFlareInitialPriceExponent pins the Flare-family seed to the granite
// base-fee floor: it is the smallest exponent whose price is exactly
// granite.MinGasPrice (500 GWei), so seeding it at the Helicon boundary keeps
// the fork fee-neutral on Flare-family networks.
func TestFlareInitialPriceExponent(t *testing.T) {
	const want = gas.Price(granite.MinGasPrice)
	require.Equal(t, want, FlareInitialPriceExponent.Price(), "FlareInitialPriceExponent.Price()")
	require.Less(t, (FlareInitialPriceExponent - 1).Price(), want, "(FlareInitialPriceExponent-1).Price()")
}
