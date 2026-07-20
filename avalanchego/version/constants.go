// Copyright (C) 2019, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package version

import (
	"encoding/json"
	"time"

	_ "embed"

	"github.com/ava-labs/avalanchego/utils/constants"
)

const (
	Client = "avalanchego"
	// RPCChainVMProtocol should be bumped anytime changes are made which
	// require the plugin vm to upgrade to latest avalanchego release to be
	// compatible.
	RPCChainVMProtocol uint = 45

	CurrentDatabase = "v1.4.5"
	PrevDatabase    = "v1.0.0"
)

// These are globals that describe network upgrades and node versions
var (
	Current = &Application{
		Name:  Client,
		Major: 1,
		Minor: 14,
		Patch: 2,
	}
	MinimumCompatibleVersion = &Application{
		Name:  Client,
		Major: 1,
		Minor: 14,
		Patch: 0,
	}
	PrevMinimumCompatibleVersion = &Application{
		Name:  Client,
		Major: 1,
		Minor: 13,
		Patch: 0,
	}

	CurrentSgb = &Application{
		Major: 0,
		Minor: 12,
		Patch: 2,
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
		Minor: 12,
		Patch: 0,
	}
	PrevMinimumCompatibleSgbVersion = &Application{
		Name:  Client,
		Major: 0,
		Minor: 11,
		Patch: 0,
	}

	//go:embed compatibility.json
	rpcChainVMProtocolCompatibilityBytes []byte
	// RPCChainVMProtocolCompatibility maps RPCChainVMProtocol versions to the
	// set of avalanchego versions that supported that version. This is not used
	// by avalanchego, but is useful for downstream libraries.
	RPCChainVMProtocolCompatibility map[uint][]string
)

func init() {
	err := json.Unmarshal(rpcChainVMProtocolCompatibilityBytes, &RPCChainVMProtocolCompatibility)
	if err != nil {
		panic(err)
	}
}

func GetCompatibility(networkID uint32, upgradeTime time.Time) *Compatibility {
	if networkID == constants.SongbirdID || networkID == constants.CostonID || networkID == constants.LocalID {
		return &Compatibility{
			Current:                   CurrentSgbApp,
			MinCompatibleAfterUpgrade: MinimumCompatibleSgbVersion,
			MinCompatible:             PrevMinimumCompatibleSgbVersion,
			UpgradeTime:               upgradeTime,
		}
	}
	return &Compatibility{
		Current:                   Current,
		MinCompatibleAfterUpgrade: MinimumCompatibleVersion,
		MinCompatible:             PrevMinimumCompatibleVersion,
		UpgradeTime:               upgradeTime,
	}
}
