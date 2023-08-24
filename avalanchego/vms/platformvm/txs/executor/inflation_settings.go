package executor

import (
	"time"

	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/units"
	"github.com/ava-labs/avalanchego/vms/platformvm/config"
)

// The value of currentTimestamp is used to return new inflation settings over time
func GetCurrentInflationSettings(startTime time.Time, networkID uint32, config *config.Config) (uint64, uint64, uint64, uint32, time.Duration, time.Duration, time.Duration, time.Duration, uint64, time.Time) {
	switch {
	case startTime.Before(getPhaseTwoStakingStartTime(networkID)):
		return getPhaseOneInflationSettings(networkID, config)
	default:
		return getPhaseTwoInflationSettings(networkID, config)
	}
}

func GetMinStake(networkID uint32, config *config.Config) (minValidatorStake, minDelegatorStake uint64) {
	minValidatorStake1, _, minDelegatorStake1, _, _, _, _, _, _, _ := getPhaseOneInflationSettings(networkID, config)
	minValidatorStake2, _, minDelegatorStake2, _, _, _, _, _, _, _ := getPhaseTwoInflationSettings(networkID, config)
	if minValidatorStake1 < minValidatorStake2 {
		minValidatorStake = minValidatorStake1
	} else {
		minValidatorStake = minValidatorStake2
	}
	if minDelegatorStake1 < minDelegatorStake2 {
		minDelegatorStake = minDelegatorStake1
	} else {
		minDelegatorStake = minDelegatorStake2
	}
	return
}

func getPhaseTwoStakingStartTime(networkID uint32) time.Time {
	switch networkID {
	case constants.FlareID:
		return time.Date(2023, time.October, 1, 0, 0, 0, 0, time.UTC)
	case constants.CostwoID:
		return time.Date(2023, time.August, 15, 0, 0, 0, 0, time.UTC)
	default:
		return time.Date(2023, time.August, 1, 0, 0, 0, 0, time.UTC)
	}
}

func getPhaseOneInflationSettings(networkID uint32, config *config.Config) (uint64, uint64, uint64, uint32, time.Duration, time.Duration, time.Duration, time.Duration, uint64, time.Time) {
	switch networkID {
	case constants.FlareID:
		return 10 * units.MegaAvax, // minValidatorStake
			50 * units.MegaAvax, // maxValidatorStake
			1 * units.KiloAvax, // minDelegatorStake
			0, // minDelegationFee
			2 * 7 * 24 * time.Hour, // minStakeDuration
			2 * 7 * 24 * time.Hour, // minDelegateDuration
			365 * 24 * time.Hour, // maxStakeDuration
			3 * 24 * time.Hour, // minFutureStartTimeOffset
			MaxValidatorWeightFactor, // maxValidatorWeightFactor
			time.Date(2023, time.July, 5, 15, 0, 0, 0, time.UTC) // minStakeStartTime
	case constants.CostwoID:
		return 100 * units.KiloAvax,
			50 * units.MegaAvax,
			1 * units.KiloAvax,
			0,
			2 * 7 * 24 * time.Hour,
			2 * 7 * 24 * time.Hour,
			365 * 24 * time.Hour,
			MaxFutureStartTime,
			MaxValidatorWeightFactor,
			time.Date(2023, time.May, 25, 15, 0, 0, 0, time.UTC)
	case constants.StagingID:
		return 100 * units.KiloAvax,
			50 * units.MegaAvax,
			1 * units.KiloAvax,
			0,
			2 * 7 * 24 * time.Hour,
			2 * 7 * 24 * time.Hour,
			365 * 24 * time.Hour,
			MaxFutureStartTime,
			MaxValidatorWeightFactor,
			time.Date(2023, time.May, 10, 15, 0, 0, 0, time.UTC)
	case constants.LocalFlareID:
		return 10 * units.KiloAvax,
			50 * units.MegaAvax,
			10 * units.KiloAvax,
			0,
			2 * 7 * 24 * time.Hour,
			2 * 7 * 24 * time.Hour,
			365 * 24 * time.Hour,
			24 * time.Hour,
			MaxValidatorWeightFactor,
			time.Date(2023, time.April, 10, 15, 0, 0, 0, time.UTC)
	default:
		return config.MinValidatorStake,
			config.MaxValidatorStake,
			config.MinDelegatorStake,
			config.MinDelegationFee,
			config.MinStakeDuration,
			config.MinStakeDuration,
			config.MaxStakeDuration,
			MaxFutureStartTime,
			MaxValidatorWeightFactor,
			time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC)
	}
}

func getPhaseTwoInflationSettings(networkID uint32, config *config.Config) (uint64, uint64, uint64, uint32, time.Duration, time.Duration, time.Duration, time.Duration, uint64, time.Time) {
	switch networkID {
	case constants.FlareID:
		return 1 * units.MegaAvax, // minValidatorStake
			200 * units.MegaAvax, // maxValidatorStake
			50 * units.KiloAvax, // minDelegatorStake
			0, // minDelegationFee
			60 * 24 * time.Hour, // minStakeDuration
			2 * 7 * 24 * time.Hour, // minDelegateDuration
			365 * 24 * time.Hour, // maxStakeDuration
			MaxFutureStartTime, // minFutureStartTimeOffset
			15, // maxValidatorWeightFactor
			time.Date(2023, time.October, 1, 0, 0, 0, 0, time.UTC) // minStakeStartTime
	case constants.CostwoID:
		return 1 * units.MegaAvax,
			200 * units.MegaAvax,
			50 * units.KiloAvax,
			0,
			60 * 24 * time.Hour,
			2 * 7 * 24 * time.Hour,
			365 * 24 * time.Hour,
			MaxFutureStartTime,
			15,
			time.Date(2023, time.August, 15, 0, 0, 0, 0, time.UTC)
	default:
		return getPhaseOneInflationSettings(networkID, config)
	}
}
