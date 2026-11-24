// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package upgrade_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	. "github.com/ava-labs/avalanchego/upgrade"
)

// TestIsProposerVMActivated pins the ProposerVM activation floor: a config
// with Apricot Phase 4 at Unix 0 (Flare, Coston2, the local networks) does
// not count a Unix-0 parent — the C-chain genesis — as activated, so the
// genesis child is a pre-fork block, while a parent at or after
// InitiallyActiveTime is. Explicit later Apricot Phase 4 times are untouched.
func TestIsProposerVMActivated(t *testing.T) {
	var (
		costwoBlock1 = time.Date(2022, time.September, 7, 11, 18, 19, 0, time.UTC)
		flareBlock1  = time.Date(2022, time.July, 13, 19, 32, 41, 0, time.UTC)
	)
	tests := []struct {
		name    string
		upgrade Config
		time    time.Time
		want    bool
	}{
		{name: "flare_genesis", upgrade: Flare, time: ZeroTime, want: false},
		{name: "flare_just_before_floor", upgrade: Flare, time: InitiallyActiveTime.Add(-time.Second), want: false},
		{name: "flare_floor", upgrade: Flare, time: InitiallyActiveTime, want: true},
		{name: "flare_block_1", upgrade: Flare, time: flareBlock1, want: true},
		{name: "costwo_genesis", upgrade: Costwo, time: ZeroTime, want: false},
		{name: "costwo_block_1", upgrade: Costwo, time: costwoBlock1, want: true},
		{name: "localflare_genesis", upgrade: LocalFlare, time: ZeroTime, want: false},
		{name: "local_genesis", upgrade: Local, time: ZeroTime, want: false},
		{name: "songbird_genesis", upgrade: Songbird, time: ZeroTime, want: false},
		{name: "songbird_floor_before_ap4", upgrade: Songbird, time: InitiallyActiveTime, want: false},
		{name: "songbird_just_before_ap4", upgrade: Songbird, time: Songbird.ApricotPhase4Time.Add(-time.Second), want: false},
		{name: "songbird_ap4", upgrade: Songbird, time: Songbird.ApricotPhase4Time, want: true},
		{name: "coston_just_before_ap4", upgrade: Coston, time: Coston.ApricotPhase4Time.Add(-time.Second), want: false},
		{name: "coston_ap4", upgrade: Coston, time: Coston.ApricotPhase4Time, want: true},
		{name: "mainnet_just_before_ap4", upgrade: Mainnet, time: Mainnet.ApricotPhase4Time.Add(-time.Second), want: false},
		{name: "mainnet_ap4", upgrade: Mainnet, time: Mainnet.ApricotPhase4Time, want: true},
		{name: "default_genesis", upgrade: Default, time: ZeroTime, want: false},
		{name: "default_floor", upgrade: Default, time: InitiallyActiveTime, want: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			require.Equal(t, test.want, test.upgrade.IsProposerVMActivated(test.time))
		})
	}
}
