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
				"3b66ad21620fe6d0dd1665b89b7c5f7a3b18e34d7d18ca56b732f833b8259108",
				"0f750b09f2702ecea445657120e7dcb0cdb46a8c87d1eae9a508fa4e3bfa5a32",
				"39e596bdd2e00f5cb7fd86069c94159b98bdd79a4ed8684a88c0faf49f63bfab",
				"aa43e6ef2d60823406e7e3cb6fcdb148fdd3ffcd137a261457f9ee6d541f9ca9",
			},
		)
	default:
		return false
	}
}
