// Copyright (C) 2019-2024, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package genesis

import (
	"cmp"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/formatting/address"
	"github.com/ava-labs/avalanchego/utils/math"
	"github.com/ava-labs/avalanchego/vms/platformvm/signer"
)

const localNetworkUpdateStartTimePeriod = 9 * 30 * 24 * time.Hour // 9 months

var (
	_ utils.Sortable[Allocation] = Allocation{}

	errInvalidGenesisJSON = errors.New("could not unmarshal genesis JSON")
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

func (a Allocation) Compare(other Allocation) int {
	if amountCmp := cmp.Compare(a.InitialAmount, other.InitialAmount); amountCmp != 0 {
		return amountCmp
	}
	return a.AVAXAddr.Compare(other.AVAXAddr)
}

type Staker struct {
	NodeID        ids.NodeID                `json:"nodeID"`
	RewardAddress ids.ShortID               `json:"rewardAddress"`
	DelegationFee uint32                    `json:"delegationFee"`
	Signer        *signer.ProofOfPossession `json:"signer,omitempty"`
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
		Signer:        s.Signer,
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
		newInitialSupply, err := math.Add(initialSupply, allocation.InitialAmount)
		if err != nil {
			return 0, err
		}
		for _, unlock := range allocation.UnlockSchedule {
			newInitialSupply, err = math.Add(newInitialSupply, unlock.Amount)
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
	unparsedLocalFlareConfig := UnparsedConfig{}
	unparsedSongbirdConfig := UnparsedConfig{}
	unparsedCostonConfig := UnparsedConfig{}

	err := errors.Join(
		json.Unmarshal(mainnetGenesisConfigJSON, &unparsedMainnetConfig),
		json.Unmarshal([]byte(localGenesisConfigJSON), &unparsedLocalConfig),
		json.Unmarshal(flareGenesisConfigJSON, &unparsedFlareConfig),
		json.Unmarshal(costwoGenesisConfigJSON, &unparsedCostwoConfig),
		json.Unmarshal(localFlareGenesisConfigJSON, &unparsedLocalFlareConfig),
		json.Unmarshal([]byte(songbirdGenesisConfigJSON), &unparsedSongbirdConfig),
		json.Unmarshal([]byte(costonGenesisConfigJSON), &unparsedCostonConfig),
	)
	if err != nil {
		panic(err)
	}

	MainnetConfig, err = unparsedMainnetConfig.Parse()
	if err != nil {
		panic(err)
	}

	LocalConfig, err = unparsedLocalConfig.Parse()
	if err != nil {
		panic(err)
	}
	LocalConfig.CChainGenesis = localCChainGenesis

	FlareConfig, err = unparsedFlareConfig.Parse()
	if err != nil {
		panic(err)
	}

	CostwoConfig, err = unparsedCostwoConfig.Parse()
	if err != nil {
		panic(err)
	}

	LocalFlareConfig, err = unparsedLocalFlareConfig.Parse()
	if err != nil {
		panic(err)
	}

	SongbirdConfig, err = unparsedSongbirdConfig.Parse()
	if err != nil {
		panic(err)
	}
	SongbirdConfig.CChainGenesis = songbirdCChainGenesis

	CostonConfig, err = unparsedCostonConfig.Parse()
	if err != nil {
		panic(err)
	}
	CostonConfig.CChainGenesis = costonCChainGenesis
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
		return nil, fmt.Errorf("%w: %w", errInvalidGenesisJSON, err)
	}

	config, err := unparsedConfig.Parse()
	if err != nil {
		return nil, fmt.Errorf("unable to parse config: %w", err)
	}
	return &config, nil
}

// getRecentStartTime advances [definedStartTime] in chunks of [period]. It
// returns the latest startTime that isn't after [now].
func getRecentStartTime(
	definedStartTime time.Time,
	now time.Time,
	period time.Duration,
) time.Time {
	startTime := definedStartTime
	for {
		nextStartTime := startTime.Add(period)
		if now.Before(nextStartTime) {
			break
		}
		startTime = nextStartTime
	}
	return startTime
}
