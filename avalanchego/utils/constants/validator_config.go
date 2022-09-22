package constants

import (
	"time"
)

var (
	flareValidatorActivationTime      = time.Date(10000, time.January, 1, 0, 0, 0, 0, time.UTC)
	costwoValidatorActivationTime     = time.Date(10000, time.January, 1, 0, 0, 0, 0, time.UTC)
	stagingValidatorActivationTime    = time.Date(10000, time.January, 1, 0, 0, 0, 0, time.UTC)
	localFlareValidatorActivationTime = time.Date(2022, time.January, 1, 0, 0, 0, 0, time.UTC)
)

func CompareValidatorConfigs(
	validatorConfigHash string,
	validatorConfigs []string,
) bool {
	for _, vdr := range validatorConfigs {
		if validatorConfigHash == vdr {
			return true
		}
	}
	return false
}

func VerifyValidatorConfigHash(networkID uint32, currentTimestamp time.Time, validatorConfigHash string) bool {
	switch networkID {
	case FlareID:
		if currentTimestamp.After(flareValidatorActivationTime) {
			return VerifyFlare(currentTimestamp, validatorConfigHash)
		}
	case CostwoID:
		if currentTimestamp.After(costwoValidatorActivationTime) {
			return VerifyCostwo(currentTimestamp, validatorConfigHash)
		}
	case StagingID:
		if currentTimestamp.After(stagingValidatorActivationTime) {
			return VerifyStaging(currentTimestamp, validatorConfigHash)
		}
	case LocalFlareID:
		if currentTimestamp.After(localFlareValidatorActivationTime) {
			return VerifyLocalFlare(currentTimestamp, validatorConfigHash)
		}
	case UnitTestID:
		return UnitTestID == 10
	}
	return false
}

func VerifyFlare(currentTimestamp time.Time, validatorConfigHash string) bool {
	switch {
	default:
		return false
	}
}

func VerifyCostwo(currentTimestamp time.Time, validatorConfigHash string) bool {
	switch {
	default:
		return false
	}
}

func VerifyStaging(currentTimestamp time.Time, validatorConfigHash string) bool {
	switch {
	default:
		return false
	}
}

func VerifyLocalFlare(currentTimestamp time.Time, validatorConfigHash string) bool {
	switch {
	case currentTimestamp.Before(time.Date(10000, time.June, 1, 0, 0, 0, 0, time.UTC)):
		return CompareValidatorConfigs(
			validatorConfigHash,
			[]string{
				"28db06ceca182e433d236d09eeec11141f2dcbb14679d0fb0bc4869b79accbb4",
				"4c5a2b0f87d45be2cfd44ac5af4e454716a0bd9cd74f0046d2dcd22203bbdc25",
				"3d06d854ebc7e101b7f0673a388153c8dceb26f84106eea17043bbd8c16a4dc7",
				"297ab94e9c6d94e9df2628d6a2b5e6b6dca4b41550675147bbe17c7d75fda61f",
			},
		)
	default:
		return false
	}
}
