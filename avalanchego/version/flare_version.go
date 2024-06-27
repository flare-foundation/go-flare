package version

import "github.com/ava-labs/avalanchego/utils/constants"

var applicationPrefix = "avalanche"

func InitApplicationPrefix(networkID uint32) {
	if networkID == constants.CostonID || networkID == constants.SongbirdID || networkID == constants.LocalID {
		applicationPrefix = "flare"
	}
}

func GetApplicationPrefix() string {
	if applicationPrefix == "" {
		panic("application prefix not set")
	}
	return applicationPrefix
}
