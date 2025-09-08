// Copyright (C) 2019-2024, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package upgrade

import (
	"errors"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/coreth/constants"
)

// Fork times: copied from avalanchego/upgrade/upgrade.go
// There is an "import cycle" between coreth and avalanchego on Avalanche GitHub repository which lacks
// times for Flare and Songbird networks.

var (
	InitiallyActiveTime       = time.Date(2020, time.December, 5, 5, 0, 0, 0, time.UTC)
	UnscheduledActivationTime = time.Date(9999, time.December, 1, 0, 0, 0, 0, time.UTC)
	ZeroTime                  = time.Unix(0, 0)

	Mainnet = Config{
		ApricotPhase1Time:            time.Date(2021, time.March, 31, 14, 0, 0, 0, time.UTC),
		ApricotPhase2Time:            time.Date(2021, time.May, 10, 11, 0, 0, 0, time.UTC),
		ApricotPhase3Time:            time.Date(2021, time.August, 24, 14, 0, 0, 0, time.UTC),
		ApricotPhase4Time:            time.Date(2021, time.September, 22, 21, 0, 0, 0, time.UTC),
		ApricotPhase4MinPChainHeight: 793005,
		ApricotPhase5Time:            time.Date(2021, time.December, 2, 18, 0, 0, 0, time.UTC),
		ApricotPhasePre6Time:         time.Date(2022, time.September, 5, 1, 30, 0, 0, time.UTC),
		ApricotPhase6Time:            time.Date(2022, time.September, 6, 20, 0, 0, 0, time.UTC),
		ApricotPhasePost6Time:        time.Date(2022, time.September, 7, 3, 0, 0, 0, time.UTC),
		BanffTime:                    time.Date(2022, time.October, 18, 16, 0, 0, 0, time.UTC),
		CortinaTime:                  time.Date(2023, time.April, 25, 15, 0, 0, 0, time.UTC),
		// The mainnet stop vertex is well known. It can be verified on any
		// fully synced node by looking at the parentID of the genesis block.
		//
		// Ref: https://subnets.avax.network/x-chain/block/0
		CortinaXChainStopVertexID: ids.FromStringOrPanic("jrGWDh5Po9FMj54depyunNixpia5PN4aAYxfmNzU8n752Rjga"),
		DurangoTime:               time.Date(2024, time.March, 6, 16, 0, 0, 0, time.UTC),
		EtnaTime:                  time.Date(2024, time.December, 16, 17, 0, 0, 0, time.UTC),
	}
	Flare = Config{
		ApricotPhase1Time:     ZeroTime,
		ApricotPhase2Time:     ZeroTime,
		ApricotPhase3Time:     ZeroTime,
		ApricotPhase4Time:     ZeroTime,
		ApricotPhase5Time:     ZeroTime,
		ApricotPhasePre6Time:  time.Date(2024, time.December, 17, 12, 0, 0, 0, time.UTC),
		ApricotPhase6Time:     time.Date(2024, time.December, 17, 13, 0, 0, 0, time.UTC),
		ApricotPhasePost6Time: time.Date(2024, time.December, 17, 14, 0, 0, 0, time.UTC),
		BanffTime:             time.Date(2024, time.December, 17, 15, 0, 0, 0, time.UTC),
		CortinaTime:           time.Date(2025, time.May, 13, 12, 0, 0, 0, time.UTC),
		DurangoTime:           time.Date(2025, time.August, 5, 12, 0, 0, 0, time.UTC),
		EtnaTime:              UnscheduledActivationTime,
	}
	Songbird = Config{
		ApricotPhase1Time:      ZeroTime,
		ApricotPhase2Time:      ZeroTime,
		ApricotPhase3Time:      time.Date(2022, time.March, 7, 14, 0, 0, 0, time.UTC),
		ApricotPhase4Time:      time.Date(2022, time.March, 7, 15, 0, 0, 0, time.UTC),
		ApricotPhase5Time:      time.Date(2022, time.March, 7, 16, 0, 0, 0, time.UTC),
		SongbirdTransitionTime: time.Date(2024, time.October, 29, 12, 0, 0, 0, time.UTC),
		ApricotPhasePre6Time:   time.Date(2025, time.January, 28, 12, 0, 0, 0, time.UTC),
		ApricotPhase6Time:      time.Date(2025, time.January, 28, 13, 0, 0, 0, time.UTC),
		ApricotPhasePost6Time:  time.Date(2025, time.January, 28, 14, 0, 0, 0, time.UTC),
		BanffTime:              time.Date(2025, time.January, 28, 15, 0, 0, 0, time.UTC),
		CortinaTime:            time.Date(2025, time.May, 6, 12, 0, 0, 0, time.UTC),
		DurangoTime:            time.Date(2025, time.July, 22, 12, 0, 0, 0, time.UTC),
		EtnaTime:               UnscheduledActivationTime,
	}
	Costwo = Config{
		ApricotPhase1Time:     ZeroTime,
		ApricotPhase2Time:     ZeroTime,
		ApricotPhase3Time:     ZeroTime,
		ApricotPhase4Time:     ZeroTime,
		ApricotPhase5Time:     ZeroTime,
		ApricotPhasePre6Time:  time.Date(2024, time.November, 26, 12, 0, 0, 0, time.UTC),
		ApricotPhase6Time:     time.Date(2024, time.November, 26, 13, 0, 0, 0, time.UTC),
		ApricotPhasePost6Time: time.Date(2024, time.November, 26, 14, 0, 0, 0, time.UTC),
		BanffTime:             time.Date(2024, time.November, 26, 15, 0, 0, 0, time.UTC),
		CortinaTime:           time.Date(2025, time.April, 8, 12, 0, 0, 0, time.UTC),
		DurangoTime:           time.Date(2025, time.June, 24, 12, 0, 0, 0, time.UTC),
		EtnaTime:              UnscheduledActivationTime,
	}
	Coston = Config{
		ApricotPhase1Time:      ZeroTime,
		ApricotPhase2Time:      ZeroTime,
		ApricotPhase3Time:      time.Date(2022, time.February, 25, 14, 0, 0, 0, time.UTC),
		ApricotPhase4Time:      time.Date(2022, time.February, 25, 15, 0, 0, 0, time.UTC),
		ApricotPhase5Time:      time.Date(2022, time.February, 25, 16, 0, 0, 0, time.UTC),
		SongbirdTransitionTime: time.Date(2024, time.July, 23, 12, 0, 0, 0, time.UTC),
		ApricotPhasePre6Time:   time.Date(2025, time.January, 7, 12, 0, 0, 0, time.UTC),
		ApricotPhase6Time:      time.Date(2025, time.January, 7, 13, 0, 0, 0, time.UTC),
		ApricotPhasePost6Time:  time.Date(2025, time.January, 7, 14, 0, 0, 0, time.UTC),
		BanffTime:              time.Date(2025, time.January, 7, 15, 0, 0, 0, time.UTC),
		CortinaTime:            time.Date(2025, time.March, 27, 13, 0, 0, 0, time.UTC),
		DurangoTime:            time.Date(2025, time.July, 1, 12, 0, 0, 0, time.UTC),
		EtnaTime:               UnscheduledActivationTime,
	}
	Default = Config{
		ApricotPhase1Time:            InitiallyActiveTime,
		ApricotPhase2Time:            InitiallyActiveTime,
		ApricotPhase3Time:            InitiallyActiveTime,
		ApricotPhase4Time:            InitiallyActiveTime,
		ApricotPhase4MinPChainHeight: 0,
		ApricotPhase5Time:            InitiallyActiveTime,
		ApricotPhasePre6Time:         InitiallyActiveTime,
		ApricotPhase6Time:            InitiallyActiveTime,
		ApricotPhasePost6Time:        InitiallyActiveTime,
		BanffTime:                    InitiallyActiveTime,
		CortinaTime:                  InitiallyActiveTime,
		DurangoTime:                  InitiallyActiveTime,
		EtnaTime:                     InitiallyActiveTime,
	}
	LocalFlare = Config{
		ApricotPhase1Time:            ZeroTime,
		ApricotPhase2Time:            ZeroTime,
		ApricotPhase3Time:            ZeroTime,
		ApricotPhase4Time:            ZeroTime,
		ApricotPhase4MinPChainHeight: 0,
		ApricotPhase5Time:            ZeroTime,
		ApricotPhasePre6Time:         ZeroTime,
		ApricotPhase6Time:            ZeroTime,
		ApricotPhasePost6Time:        ZeroTime,
		BanffTime:                    ZeroTime,
		CortinaTime:                  ZeroTime,
		DurangoTime:                  ZeroTime,
		EtnaTime:                     ZeroTime,
	}
	Local = Config{
		ApricotPhase1Time:            ZeroTime,
		ApricotPhase2Time:            ZeroTime,
		ApricotPhase3Time:            ZeroTime,
		ApricotPhase4Time:            ZeroTime,
		ApricotPhase4MinPChainHeight: 0,
		ApricotPhase5Time:            ZeroTime,
		ApricotPhasePre6Time:         ZeroTime,
		ApricotPhase6Time:            ZeroTime,
		ApricotPhasePost6Time:        ZeroTime,
		BanffTime:                    ZeroTime,
		CortinaTime:                  ZeroTime,
		DurangoTime:                  ZeroTime,
		EtnaTime:                     ZeroTime,
	}
	ErrInvalidUpgradeTimes = errors.New("invalid upgrade configuration")
)

type Config struct {
	ApricotPhase1Time            time.Time `json:"apricotPhase1Time"`
	ApricotPhase2Time            time.Time `json:"apricotPhase2Time"`
	ApricotPhase3Time            time.Time `json:"apricotPhase3Time"`
	ApricotPhase4Time            time.Time `json:"apricotPhase4Time"`
	ApricotPhase4MinPChainHeight uint64    `json:"apricotPhase4MinPChainHeight"`
	ApricotPhase5Time            time.Time `json:"apricotPhase5Time"`
	ApricotPhasePre6Time         time.Time `json:"apricotPhasePre6Time"`
	ApricotPhase6Time            time.Time `json:"apricotPhase6Time"`
	ApricotPhasePost6Time        time.Time `json:"apricotPhasePost6Time"`
	SongbirdTransitionTime       time.Time `json:"songbirdTransitionTime,omitempty"` // Only used for Songbird (and Coston) network
	BanffTime                    time.Time `json:"banffTime"`
	CortinaTime                  time.Time `json:"cortinaTime"`
	CortinaXChainStopVertexID    ids.ID    `json:"cortinaXChainStopVertexID"`
	DurangoTime                  time.Time `json:"durangoTime"`
	EtnaTime                     time.Time `json:"etnaTime"`
}

func (c *Config) IsApricotPhase1Activated(t time.Time) bool {
	return !t.Before(c.ApricotPhase1Time)
}

func (c *Config) IsApricotPhase2Activated(t time.Time) bool {
	return !t.Before(c.ApricotPhase2Time)
}

func (c *Config) IsApricotPhase3Activated(t time.Time) bool {
	return !t.Before(c.ApricotPhase3Time)
}

func (c *Config) IsApricotPhase4Activated(t time.Time) bool {
	return !t.Before(c.ApricotPhase4Time)
}

func (c *Config) IsApricotPhase5Activated(t time.Time) bool {
	return !t.Before(c.ApricotPhase5Time)
}

func (c *Config) IsSongbirdTransitionActivated(t time.Time) bool {
	return !t.Before(c.SongbirdTransitionTime)
}

func (c *Config) IsApricotPhasePre6Activated(t time.Time) bool {
	return !t.Before(c.ApricotPhasePre6Time)
}

func (c *Config) IsApricotPhase6Activated(t time.Time) bool {
	return !t.Before(c.ApricotPhase6Time)
}

func (c *Config) IsApricotPhasePost6Activated(t time.Time) bool {
	return !t.Before(c.ApricotPhasePost6Time)
}

func (c *Config) IsBanffActivated(t time.Time) bool {
	return !t.Before(c.BanffTime)
}

func (c *Config) IsCortinaActivated(t time.Time) bool {
	return !t.Before(c.CortinaTime)
}

func (c *Config) IsDurangoActivated(t time.Time) bool {
	return !t.Before(c.DurangoTime)
}

func (c *Config) IsEtnaActivated(t time.Time) bool {
	return !t.Before(c.EtnaTime)
}

func GetConfig(networkID uint32) Config {
	switch networkID {
	case constants.MainnetID:
		return Mainnet
	case constants.FlareID:
		return Flare
	case constants.SongbirdID:
		return Songbird
	case constants.CostwoID:
		return Costwo
	case constants.CostonID:
		return Coston
	case constants.LocalFlareID:
		return LocalFlare
	case constants.LocalID:
		return Local
	default:
		return Default
	}
}
