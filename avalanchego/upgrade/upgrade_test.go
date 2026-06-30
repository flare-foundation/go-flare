// Copyright (C) 2019, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package upgrade

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestValidDefaultUpgrades(t *testing.T) {
	for _, upgradeTest := range []struct {
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
	} {
		t.Run(upgradeTest.name, func(t *testing.T) {
			require := require.New(t)
			require.NoError(upgradeTest.upgrade.Validate())
		})
	}
}

func TestInvalidUpgrade(t *testing.T) {
	require := require.New(t)
	firstUpgradeTime := time.Now()
	invalidSecondUpgradeTime := firstUpgradeTime.Add(-1 * time.Second)
	upgrade := Config{
		ApricotPhase1Time: firstUpgradeTime,
		ApricotPhase2Time: invalidSecondUpgradeTime,
	}
	err := upgrade.Validate()
	require.ErrorIs(err, ErrInvalidUpgradeTimes)
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
