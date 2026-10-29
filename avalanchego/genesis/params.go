// Copyright (C) 2019, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

package genesis

import (
	"time"

	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/vms/components/gas"
	"github.com/ava-labs/avalanchego/vms/platformvm/reward"
	"github.com/ava-labs/avalanchego/vms/platformvm/validators/fee"
)

// ACP267UptimeRequirement is the 90% Primary Network uptime requirement
// introduced by ACP-267.
const ACP267UptimeRequirement = .9

type StakingConfig struct {
	// Staking uptime requirements
	UptimeRequirement float64 `json:"uptimeRequirement"`
	// Minimum stake, in nAVAX, required to validate the primary network
	MinValidatorStake uint64 `json:"minValidatorStake"`
	// Maximum stake, in nAVAX, allowed to be placed on a single validator in
	// the primary network
	MaxValidatorStake uint64 `json:"maxValidatorStake"`
	// Minimum stake, in nAVAX, that can be delegated on the primary network
	MinDelegatorStake uint64 `json:"minDelegatorStake"`
	// Minimum delegation fee, in the range [0, 1000000], that can be charged
	// for delegation on the primary network.
	MinDelegationFee uint32 `json:"minDelegationFee"`
	// MinStakeDuration is the minimum amount of time a validator can validate
	// for in a single period.
	MinStakeDuration time.Duration `json:"minStakeDuration"`
	// MaxStakeDuration is the maximum amount of time a validator can validate
	// for in a single period.
	MaxStakeDuration time.Duration `json:"maxStakeDuration"`
	// HeliconMinStakeDuration is the minimum staking duration for the primary
	// network after the Helicon upgrade (ACP-273).
	HeliconMinStakeDuration time.Duration `json:"heliconMinStakeDuration"`
	// RewardConfig is the config for the reward function.
	RewardConfig reward.Config `json:"rewardConfig"`
}

type TxFeeConfig struct {
	CreateAssetTxFee   uint64     `json:"createAssetTxFee"`
	TxFee              uint64     `json:"txFee"`
	DynamicFeeConfig   gas.Config `json:"dynamicFeeConfig"`
	ValidatorFeeConfig fee.Config `json:"validatorFeeConfig"`
}

type Params struct {
	StakingConfig
	TxFeeConfig
}

func GetTxFeeConfig(networkID uint32) TxFeeConfig {
	// TxFee configs are ommitted for LocalFlare and Local networks since they
	// are set from program arguments (getTxFeeConfig)
	switch networkID {
	case constants.MainnetID:
		return MainnetParams.TxFeeConfig
	case constants.FlareID:
		return FlareParams.TxFeeConfig
	case constants.CostwoID:
		return CostwoParams.TxFeeConfig
	case constants.LocalFlareID:
		return LocalFlareParams.TxFeeConfig
	case constants.SongbirdID:
		return SongbirdParams.TxFeeConfig
	case constants.CostonID:
		return CostonParams.TxFeeConfig
	default:
		// for LocalID and any other networkID
		return LocalParams.TxFeeConfig
	}
}

// GetStakingConfig returns the staking parameters of the given network.
//
// Flare: config.getStakingConfig uses these for the public networks
// (constants.ProductionNetworkIDs and Avalanche Mainnet), where the staking
// flags are ignored; local and custom networks take the flags, which default
// to LocalParams. On the Flare networks the primary network's stake limits and
// durations are the time-phased rules in
// vms/platformvm/txs/executor/inflation_settings.go, not these values: the
// stake limits here only mirror the latest phase of those rules and are kept
// in step with them. UptimeRequirement and the stake durations do apply
// directly (reward votes, permissioned subnet validators, TransformSubnetTx)
// and equal the LocalParams values every node ran with while the flags were
// still honored on these networks, so fixing them changed no behavior.
// RewardConfig also retains the former flag defaults; staking rewards are zero.
func GetStakingConfig(networkID uint32) StakingConfig {
	switch networkID {
	case constants.MainnetID:
		return MainnetParams.StakingConfig
	case constants.LocalID:
		return LocalParams.StakingConfig
	case constants.FlareID:
		return FlareParams.StakingConfig
	case constants.CostwoID:
		return CostwoParams.StakingConfig
	case constants.LocalFlareID:
		return LocalFlareParams.StakingConfig
	case constants.SongbirdID:
		return SongbirdParams.StakingConfig
	case constants.CostonID:
		return CostonParams.StakingConfig
	default:
		return LocalParams.StakingConfig
	}
}
