// Copyright (C) 2019-2021, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package version

import (
	"time"

	"github.com/ava-labs/avalanchego/utils/constants"
)

// These are globals that describe network upgrades and node versions
var (
	Current = &Semantic{
		Major: 1,
		Minor: 7,
		Patch: 1807,
	}
	CurrentApp = &Application{
		Major: Current.Major,
		Minor: Current.Minor,
		Patch: Current.Patch,
	}
	MinimumCompatibleVersion = &Application{
		Major: 1,
		Minor: 7,
		Patch: 0,
	}
	PrevMinimumCompatibleVersion = &Application{
		Major: 1,
		Minor: 6,
		Patch: 0,
	}

	CurrentSgb = &Semantic{
		Major: 0,
		Minor: 6,
		Patch: 6,
	}
	CurrentSgbApp = &Application{
		Major: CurrentSgb.Major,
		Minor: CurrentSgb.Minor,
		Patch: CurrentSgb.Patch,
	}
	MinimumCompatibleSgbVersion = &Application{
		Major: 0,
		Minor: 6,
		Patch: 5,
	}
	PrevMinimumCompatibleSgbVersion = &Application{
		Major: 0,
		Minor: 6,
		Patch: 4,
	}

	CurrentDatabase = DatabaseVersion1_4_5
	PrevDatabase    = DatabaseVersion1_0_0

	DatabaseVersion1_4_5 = &Semantic{
		Major: 1,
		Minor: 4,
		Patch: 5,
	}
	DatabaseVersion1_0_0 = &Semantic{
		Major: 1,
		Minor: 0,
		Patch: 0,
	}

	ApricotPhase3Times = map[uint32]time.Time{
		constants.MainnetID:    time.Date(2021, time.August, 24, 14, 0, 0, 0, time.UTC),
		constants.FlareID:      time.Date(2022, time.June, 1, 0, 0, 0, 0, time.UTC),
		constants.CostwoID:     time.Date(2022, time.June, 1, 0, 0, 0, 0, time.UTC),
		constants.StagingID:    time.Date(2022, time.June, 1, 0, 0, 0, 0, time.UTC),
		constants.LocalFlareID: time.Date(2022, time.June, 1, 0, 0, 0, 0, time.UTC),
		constants.CostonID:     time.Date(2022, time.February, 25, 14, 0, 0, 0, time.UTC),
		constants.SongbirdID:   time.Date(2022, time.March, 7, 14, 0, 0, 0, time.UTC),
	}
	ApricotPhase3DefaultTime = time.Date(2020, time.December, 5, 5, 0, 0, 0, time.UTC)

	ApricotPhase4Times = map[uint32]time.Time{
		constants.MainnetID:    time.Date(2021, time.September, 22, 21, 0, 0, 0, time.UTC),
		constants.FlareID:      time.Date(2022, time.June, 1, 0, 0, 0, 0, time.UTC),
		constants.CostwoID:     time.Date(2022, time.June, 1, 0, 0, 0, 0, time.UTC),
		constants.StagingID:    time.Date(2022, time.June, 1, 0, 0, 0, 0, time.UTC),
		constants.LocalFlareID: time.Date(2022, time.June, 1, 0, 0, 0, 0, time.UTC),
		constants.CostonID:     time.Date(2022, time.February, 25, 15, 0, 0, 0, time.UTC),
		constants.SongbirdID:   time.Date(2022, time.March, 7, 15, 0, 0, 0, time.UTC),
	}
	ApricotPhase4DefaultTime     = time.Date(2020, time.December, 5, 5, 0, 0, 0, time.UTC)
	ApricotPhase4MinPChainHeight = map[uint32]uint64{
		constants.MainnetID: 793005,
	}
	ApricotPhase4DefaultMinPChainHeight uint64

	ApricotPhase5Times = map[uint32]time.Time{
		constants.MainnetID:    time.Date(2021, time.December, 2, 18, 0, 0, 0, time.UTC),
		constants.FlareID:      time.Date(2022, time.June, 1, 0, 0, 0, 0, time.UTC),
		constants.CostwoID:     time.Date(2022, time.June, 1, 0, 0, 0, 0, time.UTC),
		constants.StagingID:    time.Date(2022, time.June, 1, 0, 0, 0, 0, time.UTC),
		constants.LocalFlareID: time.Date(2022, time.June, 1, 0, 0, 0, 0, time.UTC),
		constants.CostonID:     time.Date(2022, time.February, 25, 16, 0, 0, 0, time.UTC),
		constants.SongbirdID:   time.Date(2022, time.March, 7, 16, 0, 0, 0, time.UTC),
	}
	ApricotPhase5DefaultTime = time.Date(2020, time.December, 5, 5, 0, 0, 0, time.UTC)

	// FIXME: update this before release
	BlueberryTimes = map[uint32]time.Time{
		constants.MainnetID:    time.Date(10000, time.December, 1, 0, 0, 0, 0, time.UTC),
		constants.FlareID:      time.Date(10000, time.December, 1, 0, 0, 0, 0, time.UTC),
		constants.CostwoID:     time.Date(10000, time.December, 1, 0, 0, 0, 0, time.UTC),
		constants.StagingID:    time.Date(10000, time.December, 1, 0, 0, 0, 0, time.UTC),
		constants.LocalFlareID: time.Date(10000, time.December, 1, 0, 0, 0, 0, time.UTC),
		constants.CostonID:     time.Date(10000, time.December, 1, 0, 0, 0, 0, time.UTC),
		constants.SongbirdID:   time.Date(10000, time.December, 1, 0, 0, 0, 0, time.UTC),
	}
	BlueberryDefaultTime = time.Date(2020, time.December, 5, 5, 0, 0, 0, time.UTC)

	// FIXME: update this before release
	XChainMigrationTimes = map[uint32]time.Time{
		constants.MainnetID:    time.Date(10000, time.December, 1, 0, 0, 0, 0, time.UTC),
		constants.FlareID:      time.Date(10000, time.December, 1, 0, 0, 0, 0, time.UTC),
		constants.CostwoID:     time.Date(10000, time.December, 1, 0, 0, 0, 0, time.UTC),
		constants.StagingID:    time.Date(10000, time.December, 1, 0, 0, 0, 0, time.UTC),
		constants.LocalFlareID: time.Date(10000, time.December, 1, 0, 0, 0, 0, time.UTC),
		constants.CostonID:     time.Date(10000, time.December, 1, 0, 0, 0, 0, time.UTC),
		constants.SongbirdID:   time.Date(10000, time.December, 1, 0, 0, 0, 0, time.UTC),
	}
	XChainMigrationDefaultTime = time.Date(2020, time.December, 5, 5, 0, 0, 0, time.UTC)
)

func GetApricotPhase3Time(networkID uint32) time.Time {
	if upgradeTime, exists := ApricotPhase3Times[networkID]; exists {
		return upgradeTime
	}
	return ApricotPhase3DefaultTime
}

func GetApricotPhase4Time(networkID uint32) time.Time {
	if upgradeTime, exists := ApricotPhase4Times[networkID]; exists {
		return upgradeTime
	}
	return ApricotPhase4DefaultTime
}

func GetApricotPhase4MinPChainHeight(networkID uint32) uint64 {
	if minHeight, exists := ApricotPhase4MinPChainHeight[networkID]; exists {
		return minHeight
	}
	return ApricotPhase4DefaultMinPChainHeight
}

func GetApricotPhase5Time(networkID uint32) time.Time {
	if upgradeTime, exists := ApricotPhase5Times[networkID]; exists {
		return upgradeTime
	}
	return ApricotPhase5DefaultTime
}

func GetBlueberryTime(networkID uint32) time.Time {
	if upgradeTime, exists := BlueberryTimes[networkID]; exists {
		return upgradeTime
	}
	return BlueberryDefaultTime
}

func GetXChainMigrationTime(networkID uint32) time.Time {
	if upgradeTime, exists := XChainMigrationTimes[networkID]; exists {
		return upgradeTime
	}
	return XChainMigrationDefaultTime
}

func GetCompatibility(networkID uint32) Compatibility {
	if networkID == constants.SongbirdID || networkID == constants.CostonID || networkID == constants.LocalID {
		return NewCompatibility(
			CurrentSgbApp,
			MinimumCompatibleSgbVersion,
			GetApricotPhase5Time(networkID),
			PrevMinimumCompatibleSgbVersion,
		)
	}
	return NewCompatibility(
		CurrentApp,
		MinimumCompatibleVersion,
		GetApricotPhase5Time(networkID),
		PrevMinimumCompatibleVersion,
	)
}
