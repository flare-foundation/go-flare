package executor

import (
	"time"

	"github.com/ava-labs/avalanchego/utils"
	"github.com/ava-labs/avalanchego/utils/constants"
	"github.com/ava-labs/avalanchego/utils/units"
	"github.com/ava-labs/avalanchego/vms/platformvm/config"
)

var inflationSettingsVariants = utils.NewNetworkValue(getDefaultInflationSettings).
	AddValue(constants.FlareID, getFlareInflationSettings).
	AddValue(constants.CostwoID, getCostwoInflationSettings).
	AddValue(constants.LocalFlareID, getLocalFlareInflationSettings).
	AddValue(constants.StagingID, getStagingInflationSettings).
	AddValue(constants.SongbirdID, getSongbirdInflationSettings).
	AddValue(constants.CostonID, getCostonInflationSettings).
	AddValue(constants.LocalID, getLocalInflationSettings)

type InflationSettings struct {
	MinValidatorStake        uint64
	MaxValidatorStake        uint64
	MinDelegatorStake        uint64
	MinDelegationFee         uint32
	MinStakeDuration         time.Duration
	MinDelegateDuration      time.Duration
	MaxStakeDuration         time.Duration
	MinFutureStartTimeOffset time.Duration // Will not be checked when addPermissionlessValidator tx is used
	MaxValidatorWeightFactor uint64
	MinStakeStartTime        time.Time
}

// The value of currentTimestamp is used to return new inflation settings over time
func GetCurrentInflationSettings(currentTimestamp time.Time, networkID uint32, config *config.Config) (uint64, uint64, uint64, uint32, time.Duration, time.Duration, time.Duration, time.Duration, uint64, time.Time) {
	s := inflationSettingsVariants.GetValue(networkID)(currentTimestamp, config)
	return s.MinValidatorStake, s.MaxValidatorStake, s.MinDelegatorStake, s.MinDelegationFee, s.MinStakeDuration, s.MinDelegateDuration, s.MaxStakeDuration, s.MinFutureStartTimeOffset, s.MaxValidatorWeightFactor, s.MinStakeStartTime
}

func getCurrentValidatorRules(currentTimestamp time.Time, backend *Backend) *addValidatorRules {
	s := inflationSettingsVariants.GetValue(backend.Ctx.NetworkID)(currentTimestamp, backend.Config)
	return &addValidatorRules{
		assetID:           backend.Ctx.AVAXAssetID,
		minValidatorStake: s.MinValidatorStake,
		maxValidatorStake: s.MaxValidatorStake,
		minStakeDuration:  s.MinStakeDuration,
		maxStakeDuration:  s.MaxStakeDuration,
		minDelegationFee:  s.MinDelegationFee,
		minStakeStartTime: s.MinStakeStartTime,
	}
}

func getCurrentDelegatorRules(currentTimestamp time.Time, backend *Backend) *addDelegatorRules {
	s := inflationSettingsVariants.GetValue(backend.Ctx.NetworkID)(currentTimestamp, backend.Config)
	return &addDelegatorRules{
		assetID:                  backend.Ctx.AVAXAssetID,
		minDelegatorStake:        s.MinDelegatorStake,
		maxValidatorStake:        s.MaxValidatorStake,
		minStakeDuration:         s.MinDelegateDuration,
		maxStakeDuration:         s.MaxStakeDuration,
		maxValidatorWeightFactor: byte(s.MaxValidatorWeightFactor),
	}
}

func getFlareInflationSettings(currentTimestamp time.Time, _ *config.Config) InflationSettings {
	switch {
	case currentTimestamp.Before(time.Date(2023, time.October, 1, 0, 0, 0, 0, time.UTC)):
		// Phase 1
		return InflationSettings{
			MinValidatorStake:        10 * units.MegaAvax,
			MaxValidatorStake:        50 * units.MegaAvax,
			MinDelegatorStake:        1 * units.KiloAvax,
			MinDelegationFee:         0,
			MinStakeDuration:         2 * 7 * 24 * time.Hour,
			MinDelegateDuration:      2 * 7 * 24 * time.Hour,
			MaxStakeDuration:         365 * 24 * time.Hour,
			MinFutureStartTimeOffset: 3 * 24 * time.Hour,
			MaxValidatorWeightFactor: MaxValidatorWeightFactor,
			MinStakeStartTime:        time.Date(2023, time.July, 5, 15, 0, 0, 0, time.UTC),
		}
	default:
		// Phase 2
		return InflationSettings{
			MinValidatorStake:        1 * units.MegaAvax,
			MaxValidatorStake:        200 * units.MegaAvax,
			MinDelegatorStake:        50 * units.KiloAvax,
			MinDelegationFee:         0,
			MinStakeDuration:         60 * 24 * time.Hour,
			MinDelegateDuration:      2 * 7 * 24 * time.Hour,
			MaxStakeDuration:         365 * 24 * time.Hour,
			MinFutureStartTimeOffset: MaxFutureStartTime,
			MaxValidatorWeightFactor: 15,
			MinStakeStartTime:        time.Date(2023, time.October, 1, 0, 0, 0, 0, time.UTC),
		}
	}
}

func getCostwoInflationSettings(currentTimestamp time.Time, _ *config.Config) InflationSettings {
	switch {
	case currentTimestamp.Before(time.Date(2023, time.September, 7, 0, 0, 0, 0, time.UTC)):
		// Phase 1
		return InflationSettings{
			MinValidatorStake:        100 * units.KiloAvax,
			MaxValidatorStake:        50 * units.MegaAvax,
			MinDelegatorStake:        1 * units.KiloAvax,
			MinDelegationFee:         0,
			MinStakeDuration:         2 * 7 * 24 * time.Hour,
			MinDelegateDuration:      2 * 7 * 24 * time.Hour,
			MaxStakeDuration:         365 * 24 * time.Hour,
			MinFutureStartTimeOffset: MaxFutureStartTime,
			MaxValidatorWeightFactor: MaxValidatorWeightFactor,
			MinStakeStartTime:        time.Date(2023, time.May, 25, 15, 0, 0, 0, time.UTC),
		}
	default:
		// Phase 2
		return InflationSettings{
			MinValidatorStake:        1 * units.MegaAvax,
			MaxValidatorStake:        200 * units.MegaAvax,
			MinDelegatorStake:        50 * units.KiloAvax,
			MinDelegationFee:         0,
			MinStakeDuration:         60 * 24 * time.Hour,
			MinDelegateDuration:      2 * 7 * 24 * time.Hour,
			MaxStakeDuration:         365 * 24 * time.Hour,
			MinFutureStartTimeOffset: MaxFutureStartTime,
			MaxValidatorWeightFactor: 15,
			MinStakeStartTime:        time.Date(2023, time.September, 7, 0, 0, 0, 0, time.UTC),
		}
	}
}

func getLocalFlareInflationSettings(currentTimestamp time.Time, _ *config.Config) InflationSettings {
	switch {
	case currentTimestamp.Before(time.Date(2023, time.August, 1, 0, 0, 0, 0, time.UTC)):
		// Phase 1
		return InflationSettings{
			MinValidatorStake:        10 * units.KiloAvax,
			MaxValidatorStake:        50 * units.MegaAvax,
			MinDelegatorStake:        10 * units.KiloAvax,
			MinDelegationFee:         0,
			MinStakeDuration:         2 * 7 * 24 * time.Hour,
			MinDelegateDuration:      1 * time.Hour,
			MaxStakeDuration:         365 * 24 * time.Hour,
			MinFutureStartTimeOffset: MaxFutureStartTime,
			MaxValidatorWeightFactor: MaxValidatorWeightFactor,
			MinStakeStartTime:        time.Date(2023, time.April, 10, 15, 0, 0, 0, time.UTC),
		}
	default:
		// Phase 2
		return InflationSettings{
			MinValidatorStake:        10 * units.KiloAvax,
			MaxValidatorStake:        50 * units.MegaAvax,
			MinDelegatorStake:        10 * units.KiloAvax,
			MinDelegationFee:         0,
			MinStakeDuration:         2 * 7 * 24 * time.Hour,
			MinDelegateDuration:      1 * time.Hour,
			MaxStakeDuration:         365 * 24 * time.Hour,
			MinFutureStartTimeOffset: MaxFutureStartTime,
			MaxValidatorWeightFactor: MaxValidatorWeightFactor,
			MinStakeStartTime:        time.Date(2023, time.April, 10, 15, 0, 0, 0, time.UTC),
		}
	}
}

func getStagingInflationSettings(_ time.Time, _ *config.Config) InflationSettings {
	// Phase 1
	return InflationSettings{
		MinValidatorStake:        100 * units.KiloAvax,
		MaxValidatorStake:        50 * units.MegaAvax,
		MinDelegatorStake:        1 * units.KiloAvax,
		MinDelegationFee:         0,
		MinStakeDuration:         2 * 7 * 24 * time.Hour,
		MinDelegateDuration:      2 * 7 * 24 * time.Hour,
		MaxStakeDuration:         365 * 24 * time.Hour,
		MinFutureStartTimeOffset: MaxFutureStartTime,
		MaxValidatorWeightFactor: MaxValidatorWeightFactor,
		MinStakeStartTime:        time.Date(2023, time.May, 10, 15, 0, 0, 0, time.UTC),
	}
}

func getSongbirdInflationSettings(currentTimestamp time.Time, config *config.Config) InflationSettings {
	switch {
	case currentTimestamp.Before(time.Date(2000, time.March, 1, 0, 0, 0, 0, time.UTC)):
		return getDefaultInflationSettings(currentTimestamp, config)
	default:
		// Phase 2
		return InflationSettings{
			MinValidatorStake:        1 * units.MegaAvax,
			MaxValidatorStake:        200 * units.MegaAvax,
			MinDelegatorStake:        50 * units.KiloAvax,
			MinDelegationFee:         0,
			MinStakeDuration:         60 * 24 * time.Hour,
			MinDelegateDuration:      2 * 7 * 24 * time.Hour,
			MaxStakeDuration:         365 * 24 * time.Hour,
			MinFutureStartTimeOffset: MaxFutureStartTime,
			MaxValidatorWeightFactor: 15,
			MinStakeStartTime:        time.Date(2024, time.November, 19, 12, 0, 0, 0, time.UTC),
		}
	}
}

func getCostonInflationSettings(currentTimestamp time.Time, config *config.Config) InflationSettings {
	switch {
	case currentTimestamp.Before(time.Date(2000, time.March, 1, 0, 0, 0, 0, time.UTC)):
		return getDefaultInflationSettings(currentTimestamp, config)
	default:
		return InflationSettings{
			MinValidatorStake:        100 * units.KiloAvax,
			MaxValidatorStake:        1000 * units.MegaAvax,
			MinDelegatorStake:        10 * units.KiloAvax,
			MinDelegationFee:         0,
			MinStakeDuration:         24 * time.Hour,
			MinDelegateDuration:      1 * time.Hour,
			MaxStakeDuration:         365 * 24 * time.Hour,
			MinFutureStartTimeOffset: MaxFutureStartTime,
			MaxValidatorWeightFactor: 15,
			MinStakeStartTime:        time.Date(2024, time.July, 30, 12, 0, 0, 0, time.UTC),
		}
	}
}

func getLocalInflationSettings(currentTimestamp time.Time, config *config.Config) InflationSettings {
	switch {
	case currentTimestamp.Before(time.Date(2000, time.March, 1, 0, 0, 0, 0, time.UTC)):
		return getDefaultInflationSettings(currentTimestamp, config)
	default:
		return InflationSettings{
			MinValidatorStake:        10 * units.KiloAvax,
			MaxValidatorStake:        50 * units.MegaAvax,
			MinDelegatorStake:        10 * units.KiloAvax,
			MinDelegationFee:         0,
			MinStakeDuration:         2 * time.Hour,
			MinDelegateDuration:      20 * time.Minute,
			MaxStakeDuration:         365 * 24 * time.Hour,
			MinFutureStartTimeOffset: MaxFutureStartTime,
			MaxValidatorWeightFactor: 15,
			MinStakeStartTime:        time.Date(2024, time.April, 22, 15, 0, 0, 0, time.UTC),
		}
	}
}

func getDefaultInflationSettings(_ time.Time, config *config.Config) InflationSettings {
	return InflationSettings{
		MinValidatorStake:        config.MinValidatorStake,
		MaxValidatorStake:        config.MaxValidatorStake,
		MinDelegatorStake:        config.MinDelegatorStake,
		MinDelegationFee:         config.MinDelegationFee,
		MinStakeDuration:         config.MinStakeDuration,
		MinDelegateDuration:      config.MinStakeDuration,
		MaxStakeDuration:         config.MaxStakeDuration,
		MinFutureStartTimeOffset: MaxFutureStartTime,
		MaxValidatorWeightFactor: MaxValidatorWeightFactor,
		MinStakeStartTime:        time.Date(1970, time.January, 1, 0, 0, 0, 0, time.UTC),
	}
}
