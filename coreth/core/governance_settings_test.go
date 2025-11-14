// (c) 2021, Flare Networks Limited. All rights reserved.
// Please see the file LICENSE for licensing terms.

package core

import (
	"testing"
	"time"

	"github.com/ava-labs/coreth/params"
)

// TestNewTimelockIsPermittedCostwo checks each new timelock update
// with permitted values and non-permitted values for the Costwo chainID
func TestNewTimelockIsPermittedCostwo(t *testing.T) {

	chainID := params.CostwoChainID

	// ====================================================================================
	// Test Case #1 --- timelock: 3600 seconds, valid from: September 8th, 2022 to present
	// ====================================================================================

	// Permitted timelock update:
	blockTime := uint64(time.Date(2022, time.September, 8, 0, 0, 0, 0, time.UTC).Unix())
	newTimelock := uint64(3600)
	want := true
	have := NewTimelockIsPermitted(chainID, blockTime, newTimelock)
	if want != have {
		t.Fatalf(`NewTimelockIsPermitted = %t, want %t.`, have, want)
	}

	// Non-permitted timelock update:
	blockTime = uint64(time.Date(2022, time.September, 8, 0, 0, 0, 0, time.UTC).Unix())
	newTimelock = uint64(0)
	want = false
	have = NewTimelockIsPermitted(chainID, blockTime, newTimelock)
	if want != have {
		t.Fatalf(`NewTimelockIsPermitted = %t, want %t.`, have, want)
	}

	// Non-permitted timelock update:
	blockTime = uint64(time.Date(2022, time.September, 8, 0, 0, 0, 0, time.UTC).Unix())
	newTimelock = uint64(1000000)
	want = false
	have = NewTimelockIsPermitted(chainID, blockTime, newTimelock)
	if want != have {
		t.Fatalf(`NewTimelockIsPermitted = %t, want %t.`, have, want)
	}

	// Non-permitted timelock update:
	blockTime = uint64(time.Date(2021, time.September, 8, 0, 0, 0, 0, time.UTC).Unix())
	newTimelock = uint64(3600)
	want = false
	have = NewTimelockIsPermitted(chainID, blockTime, newTimelock)
	if want != have {
		t.Fatalf(`NewTimelockIsPermitted = %t, want %t.`, have, want)
	}

}

// TestNewTimelockIsPermittedFlare checks each new timelock update
// with permitted values and non-permitted values for the Flare chainID
func TestNewTimelockIsPermittedFlare(t *testing.T) {

	chainID := params.FlareChainID

	// ====================================================================================
	// Test Case #1 --- timelock: 3600 seconds, valid from: September 9th, 2022 to present
	// ====================================================================================

	// Permitted timelock update:
	blockTime := uint64(time.Date(2022, time.September, 9, 0, 0, 0, 0, time.UTC).Unix())
	newTimelock := uint64(3600)
	want := true
	have := NewTimelockIsPermitted(chainID, blockTime, newTimelock)
	if want != have {
		t.Fatalf(`NewTimelockIsPermitted = %t, want %t.`, have, want)
	}

	// Non-permitted timelock update:
	blockTime = uint64(time.Date(2022, time.September, 9, 0, 0, 0, 0, time.UTC).Unix())
	newTimelock = uint64(0)
	want = false
	have = NewTimelockIsPermitted(chainID, blockTime, newTimelock)
	if want != have {
		t.Fatalf(`NewTimelockIsPermitted = %t, want %t.`, have, want)
	}

	// Non-permitted timelock update:
	blockTime = uint64(time.Date(2022, time.September, 9, 0, 0, 0, 0, time.UTC).Unix())
	newTimelock = uint64(1000000)
	want = false
	have = NewTimelockIsPermitted(chainID, blockTime, newTimelock)
	if want != have {
		t.Fatalf(`NewTimelockIsPermitted = %t, want %t.`, have, want)
	}

	// Non-permitted timelock update:
	blockTime = uint64(time.Date(2021, time.September, 9, 0, 0, 0, 0, time.UTC).Unix())
	newTimelock = uint64(3600)
	want = false
	have = NewTimelockIsPermitted(chainID, blockTime, newTimelock)
	if want != have {
		t.Fatalf(`NewTimelockIsPermitted = %t, want %t.`, have, want)
	}

}
