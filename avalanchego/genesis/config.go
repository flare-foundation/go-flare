// Copyright (C) 2019-2022, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package genesis

import (
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/formatting/address"
	"github.com/ava-labs/avalanchego/utils/wrappers"

	safemath "github.com/ava-labs/avalanchego/utils/math"
)

type LockedAmount struct {
	Amount   uint64 `json:"amount"`
	Locktime uint64 `json:"locktime"`
}

type Allocation struct {
	ETHAddr        ids.ShortID    `json:"ethAddr"`
	AVAXAddr       ids.ShortID    `json:"avaxAddr"`
	InitialAmount  uint64         `json:"initialAmount"`
	UnlockSchedule []LockedAmount `json:"unlockSchedule"`
}

func (a Allocation) Unparse(networkID uint32) (UnparsedAllocation, error) {
	ua := UnparsedAllocation{
		InitialAmount:  a.InitialAmount,
		UnlockSchedule: a.UnlockSchedule,
		ETHAddr:        "0x" + hex.EncodeToString(a.ETHAddr.Bytes()),
	}
	avaxAddr, err := address.Format(
		"X",
		constants.GetHRP(networkID),
		a.AVAXAddr.Bytes(),
	)
	ua.AVAXAddr = avaxAddr
	return ua, err
}

type Staker struct {
	NodeID        ids.NodeID  `json:"nodeID"`
	RewardAddress ids.ShortID `json:"rewardAddress"`
	DelegationFee uint32      `json:"delegationFee"`
}

func (s Staker) Unparse(networkID uint32) (UnparsedStaker, error) {
	avaxAddr, err := address.Format(
		"X",
		constants.GetHRP(networkID),
		s.RewardAddress.Bytes(),
	)
	return UnparsedStaker{
		NodeID:        s.NodeID,
		RewardAddress: avaxAddr,
		DelegationFee: s.DelegationFee,
	}, err
}

// Config contains the genesis addresses used to construct a genesis
type Config struct {
	NetworkID uint32 `json:"networkID"`

	Allocations []Allocation `json:"allocations"`

	StartTime                  uint64        `json:"startTime"`
	InitialStakeDuration       uint64        `json:"initialStakeDuration"`
	InitialStakeDurationOffset uint64        `json:"initialStakeDurationOffset"`
	InitialStakedFunds         []ids.ShortID `json:"initialStakedFunds"`
	InitialStakers             []Staker      `json:"initialStakers"`

	CChainGenesis string `json:"cChainGenesis"`

	Message string `json:"message"`
}

func (c Config) Unparse() (UnparsedConfig, error) {
	uc := UnparsedConfig{
		NetworkID:                  c.NetworkID,
		Allocations:                make([]UnparsedAllocation, len(c.Allocations)),
		StartTime:                  c.StartTime,
		InitialStakeDuration:       c.InitialStakeDuration,
		InitialStakeDurationOffset: c.InitialStakeDurationOffset,
		InitialStakedFunds:         make([]string, len(c.InitialStakedFunds)),
		InitialStakers:             make([]UnparsedStaker, len(c.InitialStakers)),
		CChainGenesis:              c.CChainGenesis,
		Message:                    c.Message,
	}
	for i, a := range c.Allocations {
		ua, err := a.Unparse(uc.NetworkID)
		if err != nil {
			return uc, err
		}
		uc.Allocations[i] = ua
	}
	for i, isa := range c.InitialStakedFunds {
		avaxAddr, err := address.Format(
			"X",
			constants.GetHRP(uc.NetworkID),
			isa.Bytes(),
		)
		if err != nil {
			return uc, err
		}
		uc.InitialStakedFunds[i] = avaxAddr
	}
	for i, is := range c.InitialStakers {
		uis, err := is.Unparse(c.NetworkID)
		if err != nil {
			return uc, err
		}
		uc.InitialStakers[i] = uis
	}

	return uc, nil
}

func (c *Config) InitialSupply() (uint64, error) {
	// For songbird, coston and local networks, the initial supply is 1
	if c.NetworkID == constants.SongbirdID || c.NetworkID == constants.CostonID || c.NetworkID == constants.LocalID {
		return 1, nil
	}

	initialSupply := uint64(0)
	for _, allocation := range c.Allocations {
		newInitialSupply, err := safemath.Add64(initialSupply, allocation.InitialAmount)
		if err != nil {
			return 0, err
		}
		for _, unlock := range allocation.UnlockSchedule {
			newInitialSupply, err = safemath.Add64(newInitialSupply, unlock.Amount)
			if err != nil {
				return 0, err
			}
		}
		initialSupply = newInitialSupply
	}
	return initialSupply, nil
}

var (
	// MainnetConfig is the config that should be used to generate the mainnet
	// genesis.
	MainnetConfig Config

	// LocalConfig is the config that should be used to generate a local
	// genesis.
	LocalConfig Config

	// FlareConfig is the config that should be used to generate a flare
	// genesis.
	FlareConfig Config

	// CostwoConfig is the config that should be used to generate a costwo
	// genesis.
	CostwoConfig Config

	// StagingConfig is the config that should be used to generate a flare
	// staging genesis.
	StagingConfig Config

	// LocalFlareConfig is the config that should be used to generate a localFlare
	// genesis.
	LocalFlareConfig Config

	// SongbirdConfig is the config that should be used to generate the Songbird
	// canary network genesis.
	SongbirdConfig Config

	// CostonConfig is the config tat should be used to generate the Coston test
	// network genesis.
	CostonConfig Config
)

func init() {
	unparsedMainnetConfig := UnparsedConfig{}
	unparsedLocalConfig := UnparsedConfig{}
	unparsedFlareConfig := UnparsedConfig{}
	unparsedCostwoConfig := UnparsedConfig{}
	unparsedStagingConfig := UnparsedConfig{}
	unparsedLocalFlareConfig := UnparsedConfig{}
	unparsedSongbirdConfig := UnparsedConfig{}
	unparsedCostonConfig := UnparsedConfig{}

	errs := wrappers.Errs{}
	errs.Add(
		json.Unmarshal(mainnetGenesisConfigJSON, &unparsedMainnetConfig),
		json.Unmarshal([]byte(localGenesisConfigJSON), &unparsedLocalConfig),
		json.Unmarshal(flareGenesisConfigJSON, &unparsedFlareConfig),
		json.Unmarshal(costwoGenesisConfigJSON, &unparsedCostwoConfig),
		json.Unmarshal(stagingGenesisConfigJSON, &unparsedStagingConfig),
		json.Unmarshal(localFlareGenesisConfigJSON, &unparsedLocalFlareConfig),
		json.Unmarshal([]byte(songbirdGenesisConfigJSON), &unparsedSongbirdConfig),
		json.Unmarshal([]byte(costonGenesisConfigJSON), &unparsedCostonConfig),
	)
	if errs.Errored() {
		panic(errs.Err)
	}

	mainnetConfig, err := unparsedMainnetConfig.Parse()
	errs.Add(err)
	MainnetConfig = mainnetConfig

	localConfig, err := unparsedLocalConfig.Parse()
	localConfig.CChainGenesis = localCChainGenesis
	errs.Add(err)
	LocalConfig = localConfig

	flareConfig, err := unparsedFlareConfig.Parse()
	errs.Add(err)
	FlareConfig = flareConfig

	costwoConfig, err := unparsedCostwoConfig.Parse()
	errs.Add(err)
	CostwoConfig = costwoConfig

	stagingConfig, err := unparsedStagingConfig.Parse()
	errs.Add(err)
	StagingConfig = stagingConfig

	localFlareConfig, err := unparsedLocalFlareConfig.Parse()
	errs.Add(err)
	LocalFlareConfig = localFlareConfig

	songbirdConfig, err := unparsedSongbirdConfig.Parse()
	songbirdConfig.CChainGenesis = songbirdCChainGenesis
	errs.Add(err)
	SongbirdConfig = songbirdConfig

	costonConfig, err := unparsedCostonConfig.Parse()
	costonConfig.CChainGenesis = costonCChainGenesis
	errs.Add(err)
	CostonConfig = costonConfig

	if errs.Errored() {
		panic(errs.Err)
	}
}

func GetConfig(networkID uint32) *Config {
	switch networkID {
	case constants.MainnetID:
		return &MainnetConfig
	case constants.LocalID:
		return &LocalConfig
	case constants.FlareID:
		return &FlareConfig
	case constants.CostwoID:
		return &CostwoConfig
	case constants.StagingID:
		return &StagingConfig
	case constants.LocalFlareID:
		return &LocalFlareConfig
	case constants.SongbirdID:
		return &SongbirdConfig
	case constants.CostonID:
		return &CostonConfig
	default:
		tempConfig := LocalConfig
		tempConfig.NetworkID = networkID
		return &tempConfig
	}
}

// GetConfigFile loads a *Config from a provided filepath.
func GetConfigFile(fp string) (*Config, error) {
	bytes, err := os.ReadFile(filepath.Clean(fp))
	if err != nil {
		return nil, fmt.Errorf("unable to load file %s: %w", fp, err)
	}
	return parseGenesisJSONBytesToConfig(bytes)
}

// GetConfigContent loads a *Config from a provided environment variable
func GetConfigContent(genesisContent string) (*Config, error) {
	bytes, err := base64.StdEncoding.DecodeString(genesisContent)
	if err != nil {
		return nil, fmt.Errorf("unable to decode base64 content: %w", err)
	}
	return parseGenesisJSONBytesToConfig(bytes)
}

func parseGenesisJSONBytesToConfig(bytes []byte) (*Config, error) {
	var unparsedConfig UnparsedConfig
	if err := json.Unmarshal(bytes, &unparsedConfig); err != nil {
		return nil, fmt.Errorf("could not unmarshal JSON: %w", err)
	}

	config, err := unparsedConfig.Parse()
	if err != nil {
		return nil, fmt.Errorf("unable to parse config: %w", err)
	}
	return &config, nil
}
