// Copyright (C) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package constants

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/ava-labs/avalanchego/ids"
)

// Const variables to be exported
const (
	MainnetID uint32 = 1
	CascadeID uint32 = 2
	DenaliID  uint32 = 3
	EverestID uint32 = 4

	UnitTestID   uint32 = 10
	LocalID      uint32 = 12345
	FlareID      uint32 = 14
	CostwoID     uint32 = 114
	StagingID    uint32 = 161
	LocalFlareID uint32 = 162
	SongbirdID   uint32 = 5
	CostonID     uint32 = 7

	MainnetName    = "mainnet"
	CascadeName    = "cascade"
	DenaliName     = "denali"
	EverestName    = "everest"
	UnitTestName   = "testing"
	LocalName      = "local"
	FlareName      = "flare"
	CostwoName     = "costwo"
	StagingName    = "staging"
	LocalFlareName = "localflare"
	SongbirdName   = "songbird"
	CostonName     = "coston"

	MainnetHRP    = "avax"
	CascadeHRP    = "cascade"
	DenaliHRP     = "denali"
	EverestHRP    = "everest"
	UnitTestHRP   = "testing"
	LocalHRP      = "local"
	FallbackHRP   = "custom"
	FlareHRP      = "flare"
	CostwoHRP     = "costwo"
	StagingHRP    = "staging"
	LocalFlareHRP = "localflare"
	SongbirdHRP   = "songbird"
	CostonHRP     = "coston"
)

// Variables to be exported
var (
	PrimaryNetworkID = ids.Empty
	PlatformChainID  = ids.Empty

	NetworkIDToNetworkName = map[uint32]string{
		MainnetID:    MainnetName,
		CascadeID:    CascadeName,
		DenaliID:     DenaliName,
		EverestID:    EverestName,
		UnitTestID:   UnitTestName,
		LocalID:      LocalName,
		FlareID:      FlareName,
		CostwoID:     CostwoName,
		StagingID:    StagingName,
		LocalFlareID: LocalFlareName,
		SongbirdID:   SongbirdName,
		CostonID:     CostonName,
	}
	NetworkNameToNetworkID = map[string]uint32{
		MainnetName:    MainnetID,
		CascadeName:    CascadeID,
		DenaliName:     DenaliID,
		EverestName:    EverestID,
		UnitTestName:   UnitTestID,
		LocalName:      LocalID,
		FlareName:      FlareID,
		CostwoName:     CostwoID,
		StagingName:    StagingID,
		LocalFlareName: LocalFlareID,
		SongbirdName:   SongbirdID,
		CostonName:     CostonID,
	}

	NetworkIDToHRP = map[uint32]string{
		MainnetID:    MainnetHRP,
		CascadeID:    CascadeHRP,
		DenaliID:     DenaliHRP,
		EverestID:    EverestHRP,
		UnitTestID:   UnitTestHRP,
		LocalID:      LocalHRP,
		FlareID:      FlareHRP,
		CostwoID:     CostwoHRP,
		StagingID:    StagingHRP,
		LocalFlareID: LocalFlareHRP,
		SongbirdID:   SongbirdHRP,
		CostonID:     CostonHRP,
	}
	NetworkHRPToNetworkID = map[string]uint32{
		MainnetHRP:    MainnetID,
		CascadeHRP:    CascadeID,
		DenaliHRP:     DenaliID,
		EverestHRP:    EverestID,
		UnitTestHRP:   UnitTestID,
		LocalHRP:      LocalID,
		FlareHRP:      FlareID,
		CostwoHRP:     CostwoID,
		StagingHRP:    StagingID,
		LocalFlareHRP: LocalFlareID,
		SongbirdHRP:   SongbirdID,
		CostonHRP:     CostonID,
	}

	ValidNetworkPrefix = "network-"
)

// GetHRP returns the Human-Readable-Part of bech32 addresses for a networkID
func GetHRP(networkID uint32) string {
	if hrp, ok := NetworkIDToHRP[networkID]; ok {
		return hrp
	}
	return FallbackHRP
}

// NetworkName returns a human readable name for the network with
// ID [networkID]
func NetworkName(networkID uint32) string {
	if name, exists := NetworkIDToNetworkName[networkID]; exists {
		return name
	}
	return fmt.Sprintf("network-%d", networkID)
}

// NetworkID returns the ID of the network with name [networkName]
func NetworkID(networkName string) (uint32, error) {
	networkName = strings.ToLower(networkName)
	if id, exists := NetworkNameToNetworkID[networkName]; exists {
		return id, nil
	}

	idStr := networkName
	if strings.HasPrefix(networkName, ValidNetworkPrefix) {
		idStr = networkName[len(ValidNetworkPrefix):]
	}
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("failed to parse %q as a network name", networkName)
	}
	return uint32(id), nil
}

func IsFlareNetworkID(networkID uint32) bool {
	return networkID == FlareID || networkID == CostwoID || networkID == StagingID || networkID == LocalFlareID
}

func IsSgbNetworkID(networkID uint32) bool {
	return networkID == SongbirdID || networkID == CostonID || networkID == LocalID
}
