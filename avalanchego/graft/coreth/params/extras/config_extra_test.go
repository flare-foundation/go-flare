// Copyright (C) 2019, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package extras

import (
	"testing"
	"time"

	"github.com/ava-labs/avalanchego/upgrade"
	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/utils"
)

func TestIsTimestampForked(t *testing.T) {
	type test struct {
		fork     *uint64
		block    uint64
		isForked bool
	}

	for name, test := range map[string]test{
		"nil fork at 0": {
			fork:     nil,
			block:    0,
			isForked: false,
		},
		"nil fork at non-zero": {
			fork:     nil,
			block:    100,
			isForked: false,
		},
		"zero fork at genesis": {
			fork:     utils.PointerTo[uint64](0),
			block:    0,
			isForked: true,
		},
		"pre fork timestamp": {
			fork:     utils.PointerTo[uint64](100),
			block:    50,
			isForked: false,
		},
		"at fork timestamp": {
			fork:     utils.PointerTo[uint64](100),
			block:    100,
			isForked: true,
		},
		"post fork timestamp": {
			fork:     utils.PointerTo[uint64](100),
			block:    150,
			isForked: true,
		},
	} {
		t.Run(name, func(t *testing.T) {
			res := isTimestampForked(test.fork, test.block)
			require.Equal(t, test.isForked, res)
		})
	}
}

func TestIsForkTransition(t *testing.T) {
	type test struct {
		fork, parent *uint64
		current      uint64
		transitioned bool
	}

	for name, test := range map[string]test{
		"not active at genesis": {
			fork:         nil,
			parent:       nil,
			current:      0,
			transitioned: false,
		},
		"activate at genesis": {
			fork:         utils.PointerTo[uint64](0),
			parent:       nil,
			current:      0,
			transitioned: true,
		},
		"nil fork arbitrary transition": {
			fork:         nil,
			parent:       utils.PointerTo[uint64](100),
			current:      101,
			transitioned: false,
		},
		"nil fork transition same timestamp": {
			fork:         nil,
			parent:       utils.PointerTo[uint64](100),
			current:      100,
			transitioned: false,
		},
		"exact match on current timestamp": {
			fork:         utils.PointerTo[uint64](100),
			parent:       utils.PointerTo[uint64](99),
			current:      100,
			transitioned: true,
		},
		"current same as parent does not transition twice": {
			fork:         utils.PointerTo[uint64](100),
			parent:       utils.PointerTo[uint64](101),
			current:      101,
			transitioned: false,
		},
		"current, parent, and fork same should not transition twice": {
			fork:         utils.PointerTo[uint64](100),
			parent:       utils.PointerTo[uint64](100),
			current:      100,
			transitioned: false,
		},
		"current transitions after fork": {
			fork:         utils.PointerTo[uint64](100),
			parent:       utils.PointerTo[uint64](99),
			current:      101,
			transitioned: true,
		},
		"current and parent come after fork": {
			fork:         utils.PointerTo[uint64](100),
			parent:       utils.PointerTo[uint64](101),
			current:      102,
			transitioned: false,
		},
	} {
		t.Run(name, func(t *testing.T) {
			res := IsForkTransition(test.fork, test.parent, test.current)
			require.Equal(t, test.transitioned, res)
		})
	}
}

func TestCheckConfigForkOrderSongbirdTransition(t *testing.T) {
	baseUpgrades := NetworkUpgrades{
		ApricotPhase1BlockTimestamp: utils.PointerTo[uint64](0),
		ApricotPhase2BlockTimestamp: utils.PointerTo[uint64](0),
		ApricotPhase3BlockTimestamp: utils.PointerTo[uint64](0),
		ApricotPhase4BlockTimestamp: utils.PointerTo[uint64](0),
		ApricotPhase5BlockTimestamp: utils.PointerTo[uint64](0),
	}

	tests := map[string]struct {
		songbirdTransitionTimestamp *uint64
		apricotPhasePre6Timestamp   *uint64
		wantErr                     bool
	}{
		"omitted songbird transition is allowed before pre6": {
			apricotPhasePre6Timestamp: utils.PointerTo[uint64](10),
		},
		"songbird transition before pre6 is allowed": {
			songbirdTransitionTimestamp: utils.PointerTo[uint64](5),
			apricotPhasePre6Timestamp:   utils.PointerTo[uint64](10),
		},
		"songbird transition after pre6 is rejected": {
			songbirdTransitionTimestamp: utils.PointerTo[uint64](15),
			apricotPhasePre6Timestamp:   utils.PointerTo[uint64](10),
			wantErr:                     true,
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			config := &ChainConfig{
				NetworkUpgrades: baseUpgrades,
			}
			config.SongbirdTransitionTimestamp = test.songbirdTransitionTimestamp
			config.ApricotPhasePre6BlockTimestamp = test.apricotPhasePre6Timestamp

			err := config.CheckConfigForkOrder()
			if test.wantErr {
				require.Error(t, err)
				return
			}
			require.NoError(t, err)
		})
	}
}

func TestGetNetworkUpgradesSongbirdTransition(t *testing.T) {
	t.Run("omitted transition remains nil", func(t *testing.T) {
		networkUpgrades := GetNetworkUpgrades(upgrade.Config{})
		require.Nil(t, networkUpgrades.SongbirdTransitionTimestamp)
	})

	t.Run("zero transition is initially active", func(t *testing.T) {
		networkUpgrades := GetNetworkUpgrades(upgrade.Config{
			SongbirdTransitionTime: upgrade.ZeroTime,
		})
		require.Equal(t, utils.PointerTo[uint64](0), networkUpgrades.SongbirdTransitionTimestamp)
	})

	t.Run("configured transition is preserved", func(t *testing.T) {
		transitionTime := time.Unix(100, 0)
		networkUpgrades := GetNetworkUpgrades(upgrade.Config{
			SongbirdTransitionTime: transitionTime,
		})
		require.Equal(t, utils.PointerTo[uint64](uint64(transitionTime.Unix())), networkUpgrades.SongbirdTransitionTimestamp)
	})
}
