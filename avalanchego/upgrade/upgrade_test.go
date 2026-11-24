// Copyright (C) 2019, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package upgrade_test

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/upgrade/upgradetest"

	. "github.com/ava-labs/avalanchego/upgrade"
)

func TestValidDefaultUpgrades(t *testing.T) {
	tests := []struct {
		name    string
		upgrade Config
	}{
		{
			name:    "Default",
			upgrade: Default,
		},
		{
			name:    "Flare",
			upgrade: Flare,
		},
		{
			name:    "Songbird",
			upgrade: Songbird,
		},
		{
			name:    "Costwo",
			upgrade: Costwo,
		},
		{
			name:    "Coston",
			upgrade: Coston,
		},
		{
			name:    "LocalFlare",
			upgrade: LocalFlare,
		},
		{
			name:    "Local",
			upgrade: Local,
		},
		{
			name:    "Mainnet",
			upgrade: Mainnet,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			require.NoError(t, tt.upgrade.Validate())
		})
	}
}

func TestInvalidUpgrade(t *testing.T) {
	now := time.Now()
	upgrade := Config{
		ApricotPhase1Time: now,
		ApricotPhase2Time: now.Add(-1 * time.Second),
	}

	err := upgrade.Validate()
	require.ErrorIs(t, err, ErrInvalidUpgradeTimes)
}

func TestLatestTime(t *testing.T) {
	now := time.Now()
	c := upgradetest.GetConfigWithUpgradeTime(upgradetest.Latest, now)
	require.Equal(t, now, c.LatestTime())
}

func TestInvalidSongbirdTransitionUpgrade(t *testing.T) {
	require := require.New(t)
	upgrade := Config{
		ApricotPhase1Time:      ZeroTime,
		ApricotPhase2Time:      ZeroTime,
		ApricotPhase3Time:      ZeroTime,
		ApricotPhase4Time:      ZeroTime,
		ApricotPhase5Time:      time.Unix(10, 0),
		SongbirdTransitionTime: time.Unix(30, 0),
		ApricotPhasePre6Time:   time.Unix(20, 0),
	}

	err := upgrade.Validate()
	require.ErrorIs(err, ErrInvalidUpgradeTimes)
}
