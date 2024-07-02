// Copyright (C) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package genesis

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"path/filepath"
	"testing"

	_ "embed"

	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/hashing"
	"github.com/ava-labs/avalanchego/utils/perms"
	"github.com/ava-labs/avalanchego/vms/platformvm/genesis"
)

var (
	//go:embed genesis_test.json
	customGenesisConfigJSON  []byte
	invalidGenesisConfigJSON = []byte(`{
		"networkID": 9999}}}}
	}`)
)

func TestValidateConfig(t *testing.T) {
	tests := map[string]struct {
		networkID uint32
		config    *Config
		err       string
	}{
		"mainnet": {
			networkID: 1,
			config:    &MainnetConfig,
		},
		"local": {
			networkID: 162,
			config:    &LocalFlareConfig,
		},
		"mainnet (networkID mismatch)": {
			networkID: 2,
			config:    &MainnetConfig,
			err:       "networkID 2 specified but genesis config contains networkID 1",
		},
		"invalid start time": {
			networkID: 162,
			config: func() *Config {
				thisConfig := LocalFlareConfig
				thisConfig.StartTime = 999999999999999
				return &thisConfig
			}(),
			err: "start time cannot be in the future",
		},
		"no initial supply": {
			networkID: 162,
			config: func() *Config {
				thisConfig := LocalFlareConfig
				thisConfig.Allocations = []Allocation{}
				return &thisConfig
			}(),
			err: "initial supply must be > 0",
		},
		"no initial stakers": {
			networkID: 162,
			config: func() *Config {
				thisConfig := LocalFlareConfig
				thisConfig.InitialStakers = []Staker{}
				return &thisConfig
			}(),
			err: "initial stakers must be > 0",
		},
		"invalid initial stake duration": {
			networkID: 162,
			config: func() *Config {
				thisConfig := LocalFlareConfig
				thisConfig.InitialStakeDuration = 0
				return &thisConfig
			}(),
			err: "initial stake duration must be > 0",
		},
		"invalid stake offset": {
			networkID: 14,
			config: func() *Config {
				thisConfig := FlareConfig
				thisConfig.InitialStakeDurationOffset = 100000000
				return &thisConfig
			}(),
			err: "initial stake duration is 31536000 but need at least 1900000000 with offset of 100000000",
		},
		"empty initial staked funds": {
			networkID: 162,
			config: func() *Config {
				thisConfig := LocalFlareConfig
				thisConfig.InitialStakedFunds = []ids.ShortID(nil)
				return &thisConfig
			}(),
			err: "initial staked funds cannot be empty",
		},
		"duplicate initial staked funds": {
			networkID: 162,
			config: func() *Config {
				thisConfig := LocalFlareConfig
				thisConfig.InitialStakedFunds = append(thisConfig.InitialStakedFunds, thisConfig.InitialStakedFunds[0])
				return &thisConfig
			}(),
			err: "duplicated in initial staked funds",
		},
		"empty C-Chain genesis": {
			networkID: 162,
			config: func() *Config {
				thisConfig := LocalFlareConfig
				thisConfig.CChainGenesis = ""
				return &thisConfig
			}(),
			err: "C-Chain genesis cannot be empty",
		},
		"empty message": {
			networkID: 162,
			config: func() *Config {
				thisConfig := LocalFlareConfig
				thisConfig.Message = ""
				return &thisConfig
			}(),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)

			err := validateConfig(test.networkID, test.config)
			if len(test.err) > 0 {
				require.Error(err)
				require.Contains(err.Error(), test.err)
				return
			}
			require.NoError(err)
		})
	}
}

func TestGenesisFromFile(t *testing.T) {
	tests := map[string]struct {
		networkID       uint32
		customConfig    []byte
		missingFilepath string
		err             string
		expected        string
	}{
		"flare": {
			networkID:    constants.FlareID,
			customConfig: customGenesisConfigJSON,
			err:          "cannot override genesis config for standard network flare (14)",
		},
		"songbird": {
			networkID:    constants.SongbirdID,
			customConfig: customGenesisConfigJSON,
			err:          "cannot override genesis config for standard network songbird (5)",
		},
		"songbird (with custom specified)": {
			networkID:    constants.SongbirdID,
			customConfig: []byte(localGenesisConfigJSON), // won't load
			err:          "cannot override genesis config for standard network songbird (5)",
		},
		"local": {
			networkID:    constants.LocalID,
			customConfig: customGenesisConfigJSON,
			err:          "cannot override genesis config for standard network local (12345)",
		},
		"local (with custom specified)": {
			networkID:    constants.LocalID,
			customConfig: customGenesisConfigJSON,
			err:          "cannot override genesis config for standard network local (12345)",
		},
		"custom": {
			networkID:    9999,
			customConfig: customGenesisConfigJSON,
			expected:     "a1d1838586db85fe94ab1143560c3356df9ba2445794b796bba050be89f4fcb4",
		},
		"custom (networkID mismatch)": {
			networkID:    9999,
			customConfig: []byte(localGenesisConfigJSON),
			err:          "networkID 9999 specified but genesis config contains networkID 12345",
		},
		"custom (invalid format)": {
			networkID:    9999,
			customConfig: invalidGenesisConfigJSON,
			err:          "unable to load provided genesis config",
		},
		"custom (missing filepath)": {
			networkID:       9999,
			missingFilepath: "missing.json",
			err:             "unable to load provided genesis config",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// test loading of genesis from file

			require := require.New(t)
			var customFile string
			if len(test.customConfig) > 0 {
				customFile = filepath.Join(t.TempDir(), "config.json")
				require.NoError(perms.WriteFile(customFile, test.customConfig, perms.ReadWrite))
			}

			if len(test.missingFilepath) > 0 {
				customFile = test.missingFilepath
			}

			genesisBytes, _, err := FromFile(test.networkID, customFile)
			if len(test.err) > 0 {
				require.Error(err)
				require.Contains(err.Error(), test.err)
				return
			}
			require.NoError(err)

			genesisHash := fmt.Sprintf("%x", hashing.ComputeHash256(genesisBytes))
			require.Equal(test.expected, genesisHash, "genesis hash mismatch")

			_, err = genesis.Parse(genesisBytes)
			require.NoError(err)
		})
	}
}

func TestGenesisFromFlag(t *testing.T) {
	tests := map[string]struct {
		networkID    uint32
		customConfig []byte
		err          string
		expected     string
	}{
		"flare": {
			networkID: constants.FlareID,
			err:       "cannot override genesis config for standard network flare (14)",
		},
		"songbird": {
			networkID: constants.SongbirdID,
			err:       "cannot override genesis config for standard network songbird (5)",
		},
		"local": {
			networkID: constants.LocalID,
			err:       "cannot override genesis config for standard network local (12345)",
		},
		"local (with custom specified)": {
			networkID:    constants.LocalID,
			customConfig: customGenesisConfigJSON,
			err:          "cannot override genesis config for standard network local (12345)",
		},
		"custom": {
			networkID:    9999,
			customConfig: customGenesisConfigJSON,
			expected:     "a1d1838586db85fe94ab1143560c3356df9ba2445794b796bba050be89f4fcb4",
		},
		"custom (networkID mismatch)": {
			networkID:    9999,
			customConfig: []byte(localGenesisConfigJSON),
			err:          "networkID 9999 specified but genesis config contains networkID 12345",
		},
		"custom (invalid format)": {
			networkID:    9999,
			customConfig: invalidGenesisConfigJSON,
			err:          "unable to load genesis content from flag",
		},
		"custom (missing content)": {
			networkID: 9999,
			err:       "unable to load genesis content from flag",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			// test loading of genesis content from flag/env-var

			require := require.New(t)
			var genBytes []byte
			if len(test.customConfig) == 0 {
				// try loading a default config
				var err error
				switch test.networkID {
				case constants.MainnetID:
					genBytes, err = json.Marshal(&MainnetConfig)
					require.NoError(err)
				case constants.SongbirdID:
					genBytes, err = json.Marshal(&SongbirdConfig)
					require.NoError(err)
				case constants.LocalID:
					genBytes, err = json.Marshal(&LocalConfig)
					require.NoError(err)
				default:
					genBytes = make([]byte, 0)
				}
			} else {
				genBytes = test.customConfig
			}
			content := base64.StdEncoding.EncodeToString(genBytes)

			genesisBytes, _, err := FromFlag(test.networkID, content)
			if len(test.err) > 0 {
				require.Error(err)
				require.Contains(err.Error(), test.err)
				return
			}
			require.NoError(err)

			genesisHash := fmt.Sprintf("%x", hashing.ComputeHash256(genesisBytes))
			require.Equal(test.expected, genesisHash, "genesis hash mismatch")

			_, err = genesis.Parse(genesisBytes)
			require.NoError(err)
		})
	}
}

func TestGenesis(t *testing.T) {
	tests := []struct {
		networkID  uint32
		expectedID string
	}{
		{
			networkID:  constants.FlareID,
			expectedID: "frq8jezXkuL4PmuBt6FDcpULh2sCsFHPgWq3ZGP1G8R8UnnoU",
		},
		{
			networkID:  constants.SongbirdID,
			expectedID: "2ACyRqRc8H5VT7DDGn4qadKfct4iTPe9buQKhAjiDyotSVkeoi",
		},
		{
			networkID:  constants.LocalID,
			expectedID: "pA6uxpovoxuKFwNxGndoX9YTYDUWCFnqwDodVWvS43UWT6Zde",
		},
	}
	for _, test := range tests {
		t.Run(constants.NetworkIDToNetworkName[test.networkID], func(t *testing.T) {
			require := require.New(t)

			config := GetConfig(test.networkID)
			genesisBytes, _, err := FromConfig(config)
			require.NoError(err)

			var genesisID ids.ID = hashing.ComputeHash256Array(genesisBytes)
			require.Equal(test.expectedID, genesisID.String())
		})
	}
}

func TestVMGenesis(t *testing.T) {
	type vmTest struct {
		vmID       ids.ID
		expectedID string
	}
	tests := []struct {
		networkID uint32
		vmTest    []vmTest
	}{
		{
			networkID: constants.MainnetID,
			vmTest: []vmTest{
				{
					vmID:       constants.AVMID,
					expectedID: "2oYMBNV4eNHyqk2fjjV5nVQLDbtmNJzq5s3qs3Lo6ftnC6FByM",
				},
				{
					vmID:       constants.EVMID,
					expectedID: "2q9e4r6Mu3U68nU1fYjgbR6JvwrRx36CohpAX5UQxse55x1Q5",
				},
			},
		},
		{
			networkID: constants.SongbirdID,
			vmTest: []vmTest{
				{
					vmID:       constants.AVMID,
					expectedID: "7xKYhEvYuUekwDxozgEiMPufzJ3jJPypKbGE8ny6KL84z4RKB",
				},
				{
					vmID:       constants.EVMID,
					expectedID: "erCt5pSo5d4bM8fMrsB2dRM54PGssDAVqRg1jHedQzr6ayLiq",
				},
			},
		},
		{
			networkID: constants.LocalID,
			vmTest: []vmTest{
				{
					vmID:       constants.AVMID,
					expectedID: "ALRkp1tuy7ErVkWuEWFLVd657JAULWDDyQkQBkLKVE94jCaNu",
				},
				{
					vmID:       constants.EVMID,
					expectedID: "yHEy62ti66aY6p4gzGWd2d5DCgSCuuYEnHJUagQVxPm24gz94",
				},
			},
		},
	}

	for _, test := range tests {
		for _, vmTest := range test.vmTest {
			name := fmt.Sprintf("%s-%s",
				constants.NetworkIDToNetworkName[test.networkID],
				vmTest.vmID,
			)
			t.Run(name, func(t *testing.T) {
				require := require.New(t)

				config := GetConfig(test.networkID)
				genesisBytes, _, err := FromConfig(config)
				require.NoError(err)

				genesisTx, err := VMGenesis(genesisBytes, vmTest.vmID)
				require.NoError(err)

				require.Equal(
					vmTest.expectedID,
					genesisTx.ID().String(),
					"%s genesisID with networkID %d mismatch",
					vmTest.vmID,
					test.networkID,
				)
			})
		}
	}
}

func TestAVAXAssetID(t *testing.T) {
	tests := []struct {
		networkID  uint32
		expectedID string
	}{
		{
			networkID:  constants.MainnetID,
			expectedID: "FvwEAhmxKfeiG8SnEvq42hc6whRyY3EFYAvebMqDNDGCgxN5Z",
		},
		{
			networkID:  constants.SongbirdID,
			expectedID: "1S3PSi4VsVpD8iK2vdykuajxVeuCV2xhjPSkQ4K88mqWGozMP",
		},
		{
			networkID:  constants.LocalID,
			expectedID: "2RULRJVXVpQNAsV3sBpy4G8LWH1LN3z5Adokv5bVtnZmsBQDCX",
		},
	}

	for _, test := range tests {
		t.Run(constants.NetworkIDToNetworkName[test.networkID], func(t *testing.T) {
			require := require.New(t)

			config := GetConfig(test.networkID)
			_, avaxAssetID, err := FromConfig(config)
			require.NoError(err)

			require.Equal(
				test.expectedID,
				avaxAssetID.String(),
				"AVAX assetID with networkID %d mismatch",
				test.networkID,
			)
		})
	}
}
