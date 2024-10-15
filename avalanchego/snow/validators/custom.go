package validators

import (
	"errors"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/constants"
)

const (
	songbirdValidatorWeight = 50_000
	costonValidatorWeight   = 200_000
	customValidatorWeight   = 200_000
	customValidatorEnv      = "CUSTOM_VALIDATORS"
	customValidatorExpEnv   = "CUSTOM_VALIDATORS_EXPIRATION"
)

var (
	// Set dates before release
	songbirdValidatorsExpTime = time.Date(10000, time.January, 1, 0, 0, 0, 0, time.UTC)
	costonValidatorsExpTime   = time.Date(10000, time.January, 1, 0, 0, 0, 0, time.UTC)
	customValidatorsExpTime   = time.Date(10000, time.January, 1, 0, 0, 0, 0, time.UTC)
)

var (
	defaultValidators = defaultValidatorSet{}
	errNotInitialized = errors.New("default validator set not initialized")
)

func DefaultValidatorList() []Validator {
	return defaultValidators.list()
}

func IsDefaultValidator(vdrID ids.NodeID) bool {
	return defaultValidators.isValidator(vdrID)
}

func InitializeDefaultValidators(networkID uint32, timestamp time.Time) {
	defaultValidators.initialize(networkID, timestamp)
}

func ExpiredDefaultValidators(networkID uint32, timestamp time.Time) []Validator {
	return defaultValidators.expiredValidators(networkID, timestamp)
}

type defaultValidatorSet struct {
	initialized bool
	vdrMap      map[ids.NodeID]Validator
}

func (dvs *defaultValidatorSet) initialize(networkID uint32, timestamp time.Time) {
	if dvs.initialized {
		return
	}

	var vdrs []Validator
	switch networkID {
	case constants.LocalID:
		vdrs = loadCustomValidators(timestamp)
	case constants.SongbirdID:
		vdrs = loadSongbirdValidators(timestamp)
	case constants.CostonID:
		vdrs = loadCostonValidators(timestamp)
	}
	dvs.vdrMap = make(map[ids.NodeID]Validator)
	for _, vdr := range vdrs {
		dvs.vdrMap[vdr.ID()] = vdr
	}
	dvs.initialized = true
}

func (dvs *defaultValidatorSet) expiredValidators(networkID uint32, timestamp time.Time) []Validator {
	if !dvs.initialized {
		panic(errNotInitialized)
	}

	switch networkID {
	case constants.LocalID:
		if !timestamp.Before(customValidatorsExpTime) {
			return dvs.list()
		}
	case constants.SongbirdID:
		if !timestamp.Before(songbirdValidatorsExpTime) {
			return dvs.list()
		}
	case constants.CostonID:
		if !timestamp.Before(costonValidatorsExpTime) {
			return dvs.list()
		}
	}
	return nil
}

func (dvs *defaultValidatorSet) list() []Validator {
	if !dvs.initialized {
		panic(errNotInitialized)
	}
	vdrs := make([]Validator, 0, len(dvs.vdrMap))
	for _, vdr := range dvs.vdrMap {
		vdrs = append(vdrs, vdr)
	}
	return vdrs
}

func (dvs *defaultValidatorSet) isValidator(vdrID ids.NodeID) bool {
	if !dvs.initialized {
		panic(errNotInitialized)
	}
	_, ok := dvs.vdrMap[vdrID]
	return ok
}

func loadCustomValidators(timestamp time.Time) []Validator {
	customValidatorList := os.Getenv(customValidatorEnv)
	customValidatorExpString := os.Getenv(customValidatorExpEnv)
	if len(customValidatorExpString) > 0 {
		if t, err := time.Parse(time.RFC3339, customValidatorExpString); err == nil {
			customValidatorsExpTime = t
		}
		// Ignore if error occurs, use default expiration time
	}
	if !timestamp.Before(customValidatorsExpTime) {
		return nil
	}
	nodeIDs := strings.Split(customValidatorList, ",")
	return createValidators(nodeIDs, uint64(customValidatorWeight))
}

func loadCostonValidators(timestamp time.Time) []Validator {
	if !timestamp.Before(costonValidatorsExpTime) {
		return nil
	}
	nodeIDs := []string{
		"NodeID-5dDZXn99LCkDoEi6t9gTitZuQmhokxQTc",
		"NodeID-EkH8wyEshzEQBToAdR7Fexxcj9rrmEEHZ",
		"NodeID-FPAwqHjs8Mw8Cuki5bkm3vSVisZr8t2Lu",
		"NodeID-AQghDJTU3zuQj73itPtfTZz6CxsTQVD3R",
		"NodeID-HaZ4HpanjndqSuN252chFsTysmdND5meA",
	}
	return createValidators(nodeIDs, uint64(costonValidatorWeight))
}

func loadSongbirdValidators(timestamp time.Time) []Validator {
	if !timestamp.Before(songbirdValidatorsExpTime) {
		return nil
	}
	nodeIDs := []string{
		"NodeID-3M9KVT6ixi4gVMisbm5TnPXYXgFN5LHuv",
		"NodeID-NnX4fajAmyvpL9RLfheNdc47FKKDuQW8i",
		"NodeID-AzdF8JNU468uwZYGquHt7bhDrsggZpK67",
		"NodeID-FqeGcnLAXbDTthd382aP9uyu1i47paRRh",
		"NodeID-B9HuZ5hDkRodyRRsiMEHWgMmmMF7xSKbj",
		"NodeID-Jx3E1F7mfkseZmqnFgDUFV3eusMxVdT6Z",
		"NodeID-FnvWuwvJGezs4uaBLujkfeM8U3gmAUY3Z",
		"NodeID-LhVs6hzHjBcEkzA1Eu8Qxb9nEQAk1Qbgf",
		"NodeID-9SqDo3MxpvEDN4bE4rLTyM7HkkKAw4h96",
		"NodeID-4tStYRTi3KDxFmv1YHTZAQxbzeyMA7z52",
		"NodeID-8XnMh17zo6pB8Pa2zptRBi9TbbMZgij2t",
		"NodeID-Cn9P5wgg7d9RNLqm4dFLCUV2diCxpkj7f",
		"NodeID-PEDdah7g7Efiii1xw8ex2dH58oMfByzjb",
		"NodeID-QCt9AxMPt5nn445CQGoA3yktqkChnKmPY",
		"NodeID-9bWz6J61B8WbQtzeSyA1jsXosyVbuUJd1",
		"NodeID-DLMnewsEwtSH8Qk7p9RGzUVyZAaZVMKsk",
		"NodeID-7meEpyjmGbL577th58dm4nvvtVZiJusFp",
		"NodeID-JeYnnrUkuArAAe2Sjo47Z3X5yfeF7cw43",
		"NodeID-Fdwp9Wtjh5rxzuTCF9z4zrSM31y7ZzBQS",
		"NodeID-JdEBRLS98PansyFKQUzFKqk4xqrVZ41nC",
	}
	return createValidators(nodeIDs, uint64(songbirdValidatorWeight))
}

func createValidators(nodeIDs []string, weight uint64) (vdrs []Validator) {
	for _, nodeID := range nodeIDs {
		if nodeID == "" {
			continue
		}

		shortID, err := ids.ShortFromPrefixedString(nodeID, ids.NodeIDPrefix)
		if err != nil {
			panic(fmt.Sprintf("invalid validator node ID: %s", nodeID))
		}
		vdrs = append(vdrs, &validator{
			nodeID: ids.NodeID(shortID),
			weight: weight,
		})
	}
	return
}
