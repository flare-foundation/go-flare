// Copyright (C) 2019-2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package config

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/pflag"
	"github.com/spf13/viper"

	"github.com/stretchr/testify/require"

	"github.com/ava-labs/avalanchego/chains"
	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/subnets"
)

func TestGetChainConfigsFromFiles(t *testing.T) {
	tests := map[string]struct {
		configs    map[string]string
		upgrades   map[string]string
		errMessage string
		expected   map[string]chains.ChainConfig
	}{
		"no chain configs": {
			configs:  map[string]string{},
			upgrades: map[string]string{},
			expected: map[string]chains.ChainConfig{},
		},
		"valid chain-id": {
			configs:  map[string]string{"yH8D7ThNJkxmtkuv2jgBa4P1Rn3Qpr4pPr7QYNfcdoS6k6HWp": "hello", "2JVSBoinj9C2J33VntvzYtVJNZdN2NKiwwKjcumHUWEb5DbBrm": "world"},
			upgrades: map[string]string{"yH8D7ThNJkxmtkuv2jgBa4P1Rn3Qpr4pPr7QYNfcdoS6k6HWp": "helloUpgrades"},
			expected: func() map[string]chains.ChainConfig {
				m := map[string]chains.ChainConfig{}
				id1, err := ids.FromString("yH8D7ThNJkxmtkuv2jgBa4P1Rn3Qpr4pPr7QYNfcdoS6k6HWp")
				require.NoError(t, err)
				m[id1.String()] = chains.ChainConfig{Config: []byte("hello"), Upgrade: []byte("helloUpgrades")}

				id2, err := ids.FromString("2JVSBoinj9C2J33VntvzYtVJNZdN2NKiwwKjcumHUWEb5DbBrm")
				require.NoError(t, err)
				m[id2.String()] = chains.ChainConfig{Config: []byte("world"), Upgrade: []byte(nil)}

				return m
			}(),
		},
		"valid alias": {
			configs:  map[string]string{"C": "hello", "X": "world"},
			upgrades: map[string]string{"C": "upgradess"},
			expected: func() map[string]chains.ChainConfig {
				m := map[string]chains.ChainConfig{}
				m["C"] = chains.ChainConfig{Config: []byte("hello"), Upgrade: []byte("upgradess")}
				m["X"] = chains.ChainConfig{Config: []byte("world"), Upgrade: []byte(nil)}

				return m
			}(),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			root := t.TempDir()
			configJSON := fmt.Sprintf(`{%q: %q}`, ChainConfigDirKey, root)
			configFile := setupConfigJSON(t, root, configJSON)
			chainsDir := root
			// Create custom configs
			for key, value := range test.configs {
				chainDir := filepath.Join(chainsDir, key)
				setupFile(t, chainDir, chainConfigFileName+".ex", value)
			}
			for key, value := range test.upgrades {
				chainDir := filepath.Join(chainsDir, key)
				setupFile(t, chainDir, chainUpgradeFileName+".ex", value)
			}

			v := setupViper(configFile)

			// Parse config
			require.Equal(root, v.GetString(ChainConfigDirKey))
			chainConfigs, err := getChainConfigs(v)
			if len(test.errMessage) > 0 {
				require.Error(err)
				if err != nil {
					require.Contains(err.Error(), test.errMessage)
				}
			} else {
				require.NoError(err)
			}
			require.Equal(test.expected, chainConfigs)
		})
	}
}

func TestGetChainConfigsDirNotExist(t *testing.T) {
	tests := map[string]struct {
		structure  string
		file       map[string]string
		errMessage string
		expected   map[string]chains.ChainConfig
	}{
		"cdir not exist": {
			structure:  "/",
			file:       map[string]string{"config.ex": "noeffect"},
			errMessage: "cannot read directory",
			expected:   nil,
		},
		"cdir is file ": {
			structure:  "/",
			file:       map[string]string{"cdir": "noeffect"},
			errMessage: "cannot read directory",
			expected:   nil,
		},
		"chain subdir not exist": {
			structure: "/cdir/",
			file:      map[string]string{"config.ex": "noeffect"},
			expected:  map[string]chains.ChainConfig{},
		},
		"full structure": {
			structure: "/cdir/C/",
			file:      map[string]string{"config.ex": "hello"},
			expected:  map[string]chains.ChainConfig{"C": {Config: []byte("hello"), Upgrade: []byte(nil)}},
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			root := t.TempDir()
			chainConfigDir := filepath.Join(root, "cdir")
			configJSON := fmt.Sprintf(`{%q: %q}`, ChainConfigDirKey, chainConfigDir)
			configFile := setupConfigJSON(t, root, configJSON)

			dirToCreate := filepath.Join(root, test.structure)
			require.NoError(os.MkdirAll(dirToCreate, 0o700))

			for key, value := range test.file {
				setupFile(t, dirToCreate, key, value)
			}
			v := setupViper(configFile)

			// Parse config
			require.Equal(chainConfigDir, v.GetString(ChainConfigDirKey))

			// don't read with getConfigFromViper since it's very slow.
			chainConfigs, err := getChainConfigs(v)
			switch {
			case len(test.errMessage) > 0:
				require.Error(err)
				require.Contains(err.Error(), test.errMessage)
			default:
				require.NoError(err)
				require.Equal(test.expected, chainConfigs)
			}
		})
	}
}

func TestSetChainConfigDefaultDir(t *testing.T) {
	require := require.New(t)
	root := t.TempDir()
	// changes internal package variable, since using defaultDir (under user home) is risky.
	defaultChainConfigDir = filepath.Join(root, "cdir")
	configFilePath := setupConfigJSON(t, root, "{}")

	v := setupViper(configFilePath)
	require.Equal(defaultChainConfigDir, v.GetString(ChainConfigDirKey))

	chainsDir := filepath.Join(defaultChainConfigDir, "C")
	setupFile(t, chainsDir, chainConfigFileName+".ex", "helloworld")
	chainConfigs, err := getChainConfigs(v)
	require.NoError(err)
	expected := map[string]chains.ChainConfig{"C": {Config: []byte("helloworld"), Upgrade: []byte(nil)}}
	require.Equal(expected, chainConfigs)
}

func TestGetChainConfigsFromFlags(t *testing.T) {
	tests := map[string]struct {
		fullConfigs map[string]chains.ChainConfig
		errMessage  string
		expected    map[string]chains.ChainConfig
	}{
		"no chain configs": {
			fullConfigs: map[string]chains.ChainConfig{},
			expected:    map[string]chains.ChainConfig{},
		},
		"valid chain-id": {
			fullConfigs: func() map[string]chains.ChainConfig {
				m := map[string]chains.ChainConfig{}
				id1, err := ids.FromString("yH8D7ThNJkxmtkuv2jgBa4P1Rn3Qpr4pPr7QYNfcdoS6k6HWp")
				require.NoError(t, err)
				m[id1.String()] = chains.ChainConfig{Config: []byte("hello"), Upgrade: []byte("helloUpgrades")}

				id2, err := ids.FromString("2JVSBoinj9C2J33VntvzYtVJNZdN2NKiwwKjcumHUWEb5DbBrm")
				require.NoError(t, err)
				m[id2.String()] = chains.ChainConfig{Config: []byte("world"), Upgrade: []byte(nil)}

				return m
			}(),
			expected: func() map[string]chains.ChainConfig {
				m := map[string]chains.ChainConfig{}
				id1, err := ids.FromString("yH8D7ThNJkxmtkuv2jgBa4P1Rn3Qpr4pPr7QYNfcdoS6k6HWp")
				require.NoError(t, err)
				m[id1.String()] = chains.ChainConfig{Config: []byte("hello"), Upgrade: []byte("helloUpgrades")}

				id2, err := ids.FromString("2JVSBoinj9C2J33VntvzYtVJNZdN2NKiwwKjcumHUWEb5DbBrm")
				require.NoError(t, err)
				m[id2.String()] = chains.ChainConfig{Config: []byte("world"), Upgrade: []byte(nil)}

				return m
			}(),
		},
		"valid alias": {
			fullConfigs: map[string]chains.ChainConfig{
				"C": {Config: []byte("hello"), Upgrade: []byte("upgradess")},
				"X": {Config: []byte("world"), Upgrade: []byte(nil)},
			},
			expected: func() map[string]chains.ChainConfig {
				m := map[string]chains.ChainConfig{}
				m["C"] = chains.ChainConfig{Config: []byte("hello"), Upgrade: []byte("upgradess")}
				m["X"] = chains.ChainConfig{Config: []byte("world"), Upgrade: []byte(nil)}

				return m
			}(),
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			jsonMaps, err := json.Marshal(test.fullConfigs)
			require.NoError(err)
			encodedFileContent := base64.StdEncoding.EncodeToString(jsonMaps)

			// build viper config
			v := setupViperFlags()
			v.Set(ChainConfigContentKey, encodedFileContent)

			// Parse config
			chainConfigs, err := getChainConfigs(v)
			if len(test.errMessage) > 0 {
				require.Error(err)
				if err != nil {
					require.Contains(err.Error(), test.errMessage)
				}
			} else {
				require.NoError(err)
			}
			require.Equal(test.expected, chainConfigs)
		})
	}
}

func TestGetVMAliasesFromFile(t *testing.T) {
	tests := map[string]struct {
		givenJSON  string
		expected   map[ids.ID][]string
		errMessage string
	}{
		"wrong vm id": {
			givenJSON:  `{"wrongVmId": ["vm1","vm2"]}`,
			expected:   nil,
			errMessage: "problem unmarshaling vm aliases",
		},
		"vm id": {
			givenJSON: `{"2Ctt6eGAeo4MLqTmGa7AdRecuVMPGWEX9wSsCLBYrLhX4a394i": ["vm1","vm2"],
										"Gmt4fuNsGJAd2PX86LBvycGaBpgCYKbuULdCLZs3SEs1Jx1LU": ["vm3", "vm4"] }`,
			expected: func() map[ids.ID][]string {
				m := map[ids.ID][]string{}
				id1, _ := ids.FromString("2Ctt6eGAeo4MLqTmGa7AdRecuVMPGWEX9wSsCLBYrLhX4a394i")
				id2, _ := ids.FromString("Gmt4fuNsGJAd2PX86LBvycGaBpgCYKbuULdCLZs3SEs1Jx1LU")
				m[id1] = []string{"vm1", "vm2"}
				m[id2] = []string{"vm3", "vm4"}
				return m
			}(),
			errMessage: "",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			root := t.TempDir()
			aliasPath := filepath.Join(root, "aliases.json")
			configJSON := fmt.Sprintf(`{%q: %q}`, VMAliasesFileKey, aliasPath)
			configFilePath := setupConfigJSON(t, root, configJSON)
			setupFile(t, root, "aliases.json", test.givenJSON)
			v := setupViper(configFilePath)
			vmAliases, err := getVMAliases(v)
			if len(test.errMessage) > 0 {
				require.Error(err)
				require.Contains(err.Error(), test.errMessage)
			} else {
				require.NoError(err)
				require.Equal(test.expected, vmAliases)
			}
		})
	}
}

func TestGetVMAliasesFromFlag(t *testing.T) {
	tests := map[string]struct {
		givenJSON  string
		expected   map[ids.ID][]string
		errMessage string
	}{
		"wrong vm id": {
			givenJSON:  `{"wrongVmId": ["vm1","vm2"]}`,
			expected:   nil,
			errMessage: "problem unmarshaling vm aliases",
		},
		"vm id": {
			givenJSON: `{"2Ctt6eGAeo4MLqTmGa7AdRecuVMPGWEX9wSsCLBYrLhX4a394i": ["vm1","vm2"],
										"Gmt4fuNsGJAd2PX86LBvycGaBpgCYKbuULdCLZs3SEs1Jx1LU": ["vm3", "vm4"] }`,
			expected: func() map[ids.ID][]string {
				m := map[ids.ID][]string{}
				id1, _ := ids.FromString("2Ctt6eGAeo4MLqTmGa7AdRecuVMPGWEX9wSsCLBYrLhX4a394i")
				id2, _ := ids.FromString("Gmt4fuNsGJAd2PX86LBvycGaBpgCYKbuULdCLZs3SEs1Jx1LU")
				m[id1] = []string{"vm1", "vm2"}
				m[id2] = []string{"vm3", "vm4"}
				return m
			}(),
			errMessage: "",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			encodedFileContent := base64.StdEncoding.EncodeToString([]byte(test.givenJSON))

			// build viper config
			v := setupViperFlags()
			v.Set(VMAliasesContentKey, encodedFileContent)

			vmAliases, err := getVMAliases(v)
			if len(test.errMessage) > 0 {
				require.Error(err)
				require.Contains(err.Error(), test.errMessage)
			} else {
				require.NoError(err)
				require.Equal(test.expected, vmAliases)
			}
		})
	}
}

func TestGetVMAliasesDefaultDir(t *testing.T) {
	require := require.New(t)
	root := t.TempDir()
	// changes internal package variable, since using defaultDir (under user home) is risky.
	defaultVMAliasFilePath = filepath.Join(root, "aliases.json")
	configFilePath := setupConfigJSON(t, root, "{}")

	v := setupViper(configFilePath)
	require.Equal(defaultVMAliasFilePath, v.GetString(VMAliasesFileKey))

	setupFile(t, root, "aliases.json", `{"2Ctt6eGAeo4MLqTmGa7AdRecuVMPGWEX9wSsCLBYrLhX4a394i": ["vm1","vm2"]}`)
	vmAliases, err := getVMAliases(v)
	require.NoError(err)

	expected := map[ids.ID][]string{}
	id, _ := ids.FromString("2Ctt6eGAeo4MLqTmGa7AdRecuVMPGWEX9wSsCLBYrLhX4a394i")
	expected[id] = []string{"vm1", "vm2"}
	require.Equal(expected, vmAliases)
}

func TestGetVMAliasesDirNotExists(t *testing.T) {
	require := require.New(t)
	root := t.TempDir()
	aliasPath := "/not/exists"
	// set it explicitly
	configJSON := fmt.Sprintf(`{%q: %q}`, VMAliasesFileKey, aliasPath)
	configFilePath := setupConfigJSON(t, root, configJSON)
	v := setupViper(configFilePath)
	vmAliases, err := getVMAliases(v)
	require.Nil(vmAliases)
	require.Error(err)
	require.Contains(err.Error(), "vm aliases file does not exist")

	// do not set it explicitly
	configJSON = "{}"
	configFilePath = setupConfigJSON(t, root, configJSON)
	v = setupViper(configFilePath)
	vmAliases, err = getVMAliases(v)
	require.Nil(vmAliases)
	require.NoError(err)
}

func TestGetSubnetConfigsFromFile(t *testing.T) {
	tests := map[string]struct {
		givenJSON  string
		testF      func(*require.Assertions, map[ids.ID]subnets.Config)
		errMessage string
		fileName   string
	}{
		"wrong config": {
			fileName:  "2Ctt6eGAeo4MLqTmGa7AdRecuVMPGWEX9wSsCLBYrLhX4a394i.json",
			givenJSON: `thisisnotjson`,
			testF: func(require *require.Assertions, given map[ids.ID]subnets.Config) {
				require.Nil(given)
			},
			errMessage: "invalid character",
		},
		"subnet is not tracked": {
			fileName:  "Gmt4fuNsGJAd2PX86LBvycGaBpgCYKbuULdCLZs3SEs1Jx1LU.json",
			givenJSON: `{"validatorOnly": true}`,
			testF: func(require *require.Assertions, given map[ids.ID]subnets.Config) {
				require.Empty(given)
			},
		},
		"wrong extension": {
			fileName:  "2Ctt6eGAeo4MLqTmGa7AdRecuVMPGWEX9wSsCLBYrLhX4a394i.yaml",
			givenJSON: `{"validatorOnly": true}`,
			testF: func(require *require.Assertions, given map[ids.ID]subnets.Config) {
				require.Empty(given)
			},
		},
		"invalid consensus parameters": {
			fileName:  "2Ctt6eGAeo4MLqTmGa7AdRecuVMPGWEX9wSsCLBYrLhX4a394i.json",
			givenJSON: `{"consensusParameters":{"k": 111, "alpha":1234} }`,
			testF: func(require *require.Assertions, given map[ids.ID]subnets.Config) {
				require.Nil(given)
			},
			errMessage: "fails the condition that: alpha <= k",
		},
		"correct config": {
			fileName:  "2Ctt6eGAeo4MLqTmGa7AdRecuVMPGWEX9wSsCLBYrLhX4a394i.json",
			givenJSON: `{"validatorOnly": true, "consensusParameters":{"parents": 111, "alpha":16} }`,
			testF: func(require *require.Assertions, given map[ids.ID]subnets.Config) {
				id, _ := ids.FromString("2Ctt6eGAeo4MLqTmGa7AdRecuVMPGWEX9wSsCLBYrLhX4a394i")
				config, ok := given[id]
				require.True(ok)

				require.Equal(true, config.ValidatorOnly)
				require.Equal(111, config.ConsensusParameters.Parents)
				require.Equal(16, config.ConsensusParameters.Alpha)
				// must still respect defaults
				require.Equal(20, config.ConsensusParameters.K)
			},
			errMessage: "",
		},
		"gossip config": {
			fileName:  "2Ctt6eGAeo4MLqTmGa7AdRecuVMPGWEX9wSsCLBYrLhX4a394i.json",
			givenJSON: `{"appGossipNonValidatorSize": 100 }`,
			testF: func(require *require.Assertions, given map[ids.ID]subnets.Config) {
				id, _ := ids.FromString("2Ctt6eGAeo4MLqTmGa7AdRecuVMPGWEX9wSsCLBYrLhX4a394i")
				config, ok := given[id]
				require.True(ok)
				require.Equal(uint(100), config.GossipConfig.AppGossipNonValidatorSize)
				// must still respect defaults
				require.Equal(20, config.ConsensusParameters.K)
				require.Equal(uint(10), config.GossipConfig.AppGossipValidatorSize)
			},
			errMessage: "",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			root := t.TempDir()
			subnetPath := filepath.Join(root, "subnets")
			configJSON := fmt.Sprintf(`{%q: %q}`, SubnetConfigDirKey, subnetPath)
			configFilePath := setupConfigJSON(t, root, configJSON)
			subnetID, err := ids.FromString("2Ctt6eGAeo4MLqTmGa7AdRecuVMPGWEX9wSsCLBYrLhX4a394i")
			require.NoError(err)
			setupFile(t, subnetPath, test.fileName, test.givenJSON)
			v := setupViper(configFilePath)
			subnetConfigs, err := getSubnetConfigs(v, []ids.ID{subnetID})
			if len(test.errMessage) > 0 {
				require.Error(err)
				require.Contains(err.Error(), test.errMessage)
			} else {
				require.NoError(err)
				test.testF(require, subnetConfigs)
			}
		})
	}
}

func TestGetSubnetConfigsFromFlags(t *testing.T) {
	tests := map[string]struct {
		givenJSON  string
		testF      func(*require.Assertions, map[ids.ID]subnets.Config)
		errMessage string
	}{
		"no configs": {
			givenJSON: `{}`,
			testF: func(require *require.Assertions, given map[ids.ID]subnets.Config) {
				require.Empty(given)
			},
			errMessage: "",
		},
		"entry with no config": {
			givenJSON: `{"2Ctt6eGAeo4MLqTmGa7AdRecuVMPGWEX9wSsCLBYrLhX4a394i":{}}`,
			testF: func(require *require.Assertions, given map[ids.ID]subnets.Config) {
				require.True(len(given) == 1)
				id, _ := ids.FromString("2Ctt6eGAeo4MLqTmGa7AdRecuVMPGWEX9wSsCLBYrLhX4a394i")
				config, ok := given[id]
				require.True(ok)
				// should respect defaults
				require.Equal(20, config.ConsensusParameters.K)
			},
		},
		"subnet is not tracked": {
			givenJSON: `{"Gmt4fuNsGJAd2PX86LBvycGaBpgCYKbuULdCLZs3SEs1Jx1LU":{"validatorOnly":true}}`,
			testF: func(require *require.Assertions, given map[ids.ID]subnets.Config) {
				require.Empty(given)
			},
		},
		"invalid consensus parameters": {
			givenJSON: `{
				"2Ctt6eGAeo4MLqTmGa7AdRecuVMPGWEX9wSsCLBYrLhX4a394i": {
					"consensusParameters": {
						"k": 111,
						"alpha": 1234
					}
				}
			}`,
			testF: func(require *require.Assertions, given map[ids.ID]subnets.Config) {
				require.Empty(given)
			},
			errMessage: "fails the condition that: alpha <= k",
		},
		"correct config": {
			givenJSON: `{
				"2Ctt6eGAeo4MLqTmGa7AdRecuVMPGWEX9wSsCLBYrLhX4a394i": {
					"consensusParameters": {
						"k": 30,
						"alpha": 20,
						"parents": 111
					},
					"validatorOnly": true
				}
			}`,
			testF: func(require *require.Assertions, given map[ids.ID]subnets.Config) {
				id, _ := ids.FromString("2Ctt6eGAeo4MLqTmGa7AdRecuVMPGWEX9wSsCLBYrLhX4a394i")
				config, ok := given[id]
				require.True(ok)
				require.Equal(true, config.ValidatorOnly)
				require.Equal(111, config.ConsensusParameters.Parents)
				require.Equal(20, config.ConsensusParameters.Alpha)
				require.Equal(30, config.ConsensusParameters.K)
				// must still respect defaults
				require.Equal(uint(10), config.GossipConfig.AppGossipValidatorSize)
				require.Equal(256, config.ConsensusParameters.MaxOutstandingItems)
			},
			errMessage: "",
		},
	}

	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			require := require.New(t)
			subnetID, err := ids.FromString("2Ctt6eGAeo4MLqTmGa7AdRecuVMPGWEX9wSsCLBYrLhX4a394i")
			require.NoError(err)
			encodedFileContent := base64.StdEncoding.EncodeToString([]byte(test.givenJSON))

			// build viper config
			v := setupViperFlags()
			v.Set(SubnetConfigContentKey, encodedFileContent)

			subnetConfigs, err := getSubnetConfigs(v, []ids.ID{subnetID})
			if len(test.errMessage) > 0 {
				require.Error(err)
				require.Contains(err.Error(), test.errMessage)
			} else {
				require.NoError(err)
				test.testF(require, subnetConfigs)
			}
		})
	}
}

func TestCalcMinConnectedStake(t *testing.T) {
	v := setupViperFlags()
	defaultParams := getConsensusConfig(v)
	defaultExpectedMinStake := 0.8
	minStake := calcMinConnectedStake(defaultParams.Parameters)
	require.Equal(t, defaultExpectedMinStake, minStake)
}

// setups config json file and writes content
func setupConfigJSON(t *testing.T, rootPath string, value string) string {
	configFilePath := filepath.Join(rootPath, "config.json")
	require.NoError(t, os.WriteFile(configFilePath, []byte(value), 0o600))
	return configFilePath
}

// setups file creates necessary path and writes value to it.
func setupFile(t *testing.T, path string, fileName string, value string) {
	require.NoError(t, os.MkdirAll(path, 0o700))
	filePath := filepath.Join(path, fileName)
	require.NoError(t, os.WriteFile(filePath, []byte(value), 0o600))
}

func setupViperFlags() *viper.Viper {
	v := viper.New()
	fs := BuildFlagSet()
	pflag.Parse()
	if err := v.BindPFlags(fs); err != nil {
		log.Fatal(err)
	}
	return v
}

func setupViper(configFilePath string) *viper.Viper {
	v := setupViperFlags()
	v.SetConfigFile(configFilePath)
	if err := v.ReadInConfig(); err != nil {
		log.Fatal(err)
	}
	return v
}
