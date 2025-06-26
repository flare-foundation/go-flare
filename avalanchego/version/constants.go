// Copyright (C) 2019-2024, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package version

import (
	"encoding/json"
	"time"

	_ "embed"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/constants"
)

const (
	Client = "avalanchego"
	// RPCChainVMProtocol should be bumped anytime changes are made which
	// require the plugin vm to upgrade to latest avalanchego release to be
	// compatible.
	RPCChainVMProtocol uint = 33
)

// These are globals that describe network upgrades and node versions
var (
	Current = &Semantic{
		Major: 1,
		Minor: 11,
		Patch: 0,
	}
	CurrentApp = &Application{
		Name:  Client,
		Major: Current.Major,
		Minor: Current.Minor,
		Patch: Current.Patch,
	}
	MinimumCompatibleVersion = &Application{
		Name:  Client,
		Major: 1,
		Minor: 11,
		Patch: 0,
	}
	PrevMinimumCompatibleVersion = &Application{
		Name:  Client,
		Major: 1,
		Minor: 10,
		Patch: 0,
	}

	CurrentSgb = &Semantic{
		Major: 0,
		Minor: 9,
		Patch: 0,
	}
	CurrentSgbApp = &Application{
		Name:  Client,
		Major: CurrentSgb.Major,
		Minor: CurrentSgb.Minor,
		Patch: CurrentSgb.Patch,
	}
	MinimumCompatibleSgbVersion = &Application{
		Name:  Client,
		Major: 0,
		Minor: 9,
		Patch: 0,
	}
	PrevMinimumCompatibleSgbVersion = &Application{
		Name:  Client,
		Major: 0,
		Minor: 8,
		Patch: 0,
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

	//go:embed compatibility.json
	rpcChainVMProtocolCompatibilityBytes []byte
	// RPCChainVMProtocolCompatibility maps RPCChainVMProtocol versions to the
	// set of avalanchego versions that supported that version. This is not used
	// by avalanchego, but is useful for downstream libraries.
	RPCChainVMProtocolCompatibility map[uint][]*Semantic

	DefaultUpgradeTime = time.Date(2020, time.December, 5, 5, 0, 0, 0, time.UTC)

	ApricotPhase3Times = map[uint32]time.Time{
		constants.MainnetID:  time.Date(2021, time.August, 24, 14, 0, 0, 0, time.UTC),
		constants.CostonID:   time.Date(2022, time.February, 25, 14, 0, 0, 0, time.UTC),
		constants.SongbirdID: time.Date(2022, time.March, 7, 14, 0, 0, 0, time.UTC),
	}

	ApricotPhase4Times = map[uint32]time.Time{
		constants.MainnetID:  time.Date(2021, time.September, 22, 21, 0, 0, 0, time.UTC),
		constants.CostonID:   time.Date(2022, time.February, 25, 15, 0, 0, 0, time.UTC),
		constants.SongbirdID: time.Date(2022, time.March, 7, 15, 0, 0, 0, time.UTC),
	}

	ApricotPhase4MinPChainHeight = map[uint32]uint64{
		constants.MainnetID: 793005,
	}

	ApricotPhase5Times = map[uint32]time.Time{
		constants.MainnetID:  time.Date(2021, time.December, 2, 18, 0, 0, 0, time.UTC),
		constants.CostonID:   time.Date(2022, time.February, 25, 16, 0, 0, 0, time.UTC),
		constants.SongbirdID: time.Date(2022, time.March, 7, 16, 0, 0, 0, time.UTC),
	}

	ApricotPhase6Times = map[uint32]time.Time{
		constants.MainnetID:  time.Date(2022, time.September, 6, 20, 0, 0, 0, time.UTC),
		constants.FlareID:    time.Date(2024, time.December, 17, 13, 0, 0, 0, time.UTC),
		constants.CostwoID:   time.Date(2024, time.November, 26, 13, 0, 0, 0, time.UTC),
		constants.CostonID:   time.Date(2025, time.January, 7, 13, 0, 0, 0, time.UTC),
		constants.SongbirdID: time.Date(2025, time.January, 28, 13, 0, 0, 0, time.UTC),
	}

	BanffTimes = map[uint32]time.Time{
		constants.MainnetID:  time.Date(2022, time.October, 18, 16, 0, 0, 0, time.UTC),
		constants.FlareID:    time.Date(2024, time.December, 17, 15, 0, 0, 0, time.UTC),
		constants.CostwoID:   time.Date(2024, time.November, 26, 15, 0, 0, 0, time.UTC),
		constants.CostonID:   time.Date(2025, time.January, 7, 15, 0, 0, 0, time.UTC),
		constants.SongbirdID: time.Date(2025, time.January, 28, 15, 0, 0, 0, time.UTC),
	}

	CortinaTimes = map[uint32]time.Time{
		constants.MainnetID:  time.Date(2023, time.April, 25, 15, 0, 0, 0, time.UTC),
		constants.FlareID:    time.Date(2025, time.May, 13, 12, 0, 0, 0, time.UTC),
		constants.CostwoID:   time.Date(2025, time.April, 8, 12, 0, 0, 0, time.UTC),
		constants.CostonID:   time.Date(2025, time.March, 27, 13, 0, 0, 0, time.UTC),
		constants.SongbirdID: time.Date(2025, time.May, 6, 12, 0, 0, 0, time.UTC),
	}
	CortinaXChainStopVertexID = map[uint32]ids.ID{
		// The mainnet stop vertex is well known. It can be verified on any
		// fully synced node by looking at the parentID of the genesis block.
		//
		// Ref: https://subnets.avax.network/x-chain/block/0
		constants.MainnetID: ids.FromStringOrPanic("jrGWDh5Po9FMj54depyunNixpia5PN4aAYxfmNzU8n752Rjga"),
	}

	DurangoTimes = map[uint32]time.Time{
		constants.MainnetID:  time.Date(2024, time.March, 6, 16, 0, 0, 0, time.UTC),
		constants.FlareID:    time.Date(2025, time.August, 5, 12, 0, 0, 0, time.UTC),
		constants.CostwoID:   time.Date(2025, time.June, 24, 12, 0, 0, 0, time.UTC),
		constants.CostonID:   time.Date(2025, time.July, 1, 12, 0, 0, 0, time.UTC),
		constants.SongbirdID: time.Date(2025, time.July, 22, 12, 0, 0, 0, time.UTC),
		constants.LocalID:    time.Date(10000, time.December, 1, 0, 0, 0, 0, time.UTC),
	}
)

func init() {
	var parsedRPCChainVMCompatibility map[uint][]string
	err := json.Unmarshal(rpcChainVMProtocolCompatibilityBytes, &parsedRPCChainVMCompatibility)
	if err != nil {
		panic(err)
	}

	RPCChainVMProtocolCompatibility = make(map[uint][]*Semantic)
	for rpcChainVMProtocol, versionStrings := range parsedRPCChainVMCompatibility {
		versions := make([]*Semantic, len(versionStrings))
		for i, versionString := range versionStrings {
			version, err := Parse(versionString)
			if err != nil {
				panic(err)
			}
			versions[i] = version
		}
		RPCChainVMProtocolCompatibility[rpcChainVMProtocol] = versions
	}
}

func GetApricotPhase3Time(networkID uint32) time.Time {
	if upgradeTime, exists := ApricotPhase3Times[networkID]; exists {
		return upgradeTime
	}
	return DefaultUpgradeTime
}

func GetApricotPhase4Time(networkID uint32) time.Time {
	if upgradeTime, exists := ApricotPhase4Times[networkID]; exists {
		return upgradeTime
	}
	return DefaultUpgradeTime
}

func GetApricotPhase5Time(networkID uint32) time.Time {
	if upgradeTime, exists := ApricotPhase5Times[networkID]; exists {
		return upgradeTime
	}
	return DefaultUpgradeTime
}

func GetApricotPhase6Time(networkID uint32) time.Time {
	if upgradeTime, exists := ApricotPhase6Times[networkID]; exists {
		return upgradeTime
	}
	return DefaultUpgradeTime
}

func GetBanffTime(networkID uint32) time.Time {
	if upgradeTime, exists := BanffTimes[networkID]; exists {
		return upgradeTime
	}
	return DefaultUpgradeTime
}

func GetCortinaTime(networkID uint32) time.Time {
	if upgradeTime, exists := CortinaTimes[networkID]; exists {
		return upgradeTime
	}
	return DefaultUpgradeTime
}

func GetDurangoTime(networkID uint32) time.Time {
	if upgradeTime, exists := DurangoTimes[networkID]; exists {
		return upgradeTime
	}
	return DefaultUpgradeTime
}

func GetCompatibility(networkID uint32) Compatibility {
	if networkID == constants.SongbirdID || networkID == constants.CostonID || networkID == constants.LocalID {
		return NewCompatibility(
			CurrentSgbApp,
			MinimumCompatibleSgbVersion,
			GetDurangoTime(networkID),
			PrevMinimumCompatibleSgbVersion,
		)
	}
	return NewCompatibility(
		CurrentApp,
		MinimumCompatibleVersion,
		GetDurangoTime(networkID),
		PrevMinimumCompatibleVersion,
	)
}
