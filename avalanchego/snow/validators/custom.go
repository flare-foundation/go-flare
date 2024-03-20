package validators

// SGB-MERGE

import (
	"errors"
	"fmt"
	"os"
	"strings"

	"github.com/ava-labs/avalanchego/ids"
	"github.com/ava-labs/avalanchego/utils/constants"
)

const (
	songbirdValidatorWeight = 50_000
	costonValidatorWeight   = 200_000
	customValidatorWeight   = 200_000
	customValidatorEnv      = "CUSTOM_VALIDATORS"
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

func InitializeDefaultValidators(networkID uint32) {
	defaultValidators.initialize(networkID)
}

type defaultValidatorSet struct {
	initialzed bool
	vdrMap     map[ids.NodeID]Validator
}

func (dvs *defaultValidatorSet) initialize(networkID uint32) {
	if dvs.initialzed {
		return
	}

	var vdrs []Validator
	switch networkID {
	case constants.LocalID:
		vdrs = loadCustomValidators()
	case constants.SongbirdID:
		vdrs = loadSongbirdValidators()
	case constants.CostonID:
		vdrs = loadCostonValidators()
	}
	dvs.vdrMap = make(map[ids.NodeID]Validator)
	for _, vdr := range vdrs {
		dvs.vdrMap[vdr.ID()] = vdr
	}
	dvs.initialzed = true
}

func (dvs *defaultValidatorSet) list() []Validator {
	if !dvs.initialzed {
		panic(errNotInitialized)
	}
	vdrs := make([]Validator, 0, len(dvs.vdrMap))
	for _, vdr := range dvs.vdrMap {
		vdrs = append(vdrs, vdr)
	}
	return vdrs
}

func (dvs *defaultValidatorSet) isValidator(vdrID ids.NodeID) bool {
	if !dvs.initialzed {
		panic(errNotInitialized)
	}
	_, ok := dvs.vdrMap[vdrID]
	return ok
}

func loadCustomValidators() []Validator {
	customValidatorList := os.Getenv(customValidatorEnv)
	nodeIDs := strings.Split(customValidatorList, ",")
	return createValidators(nodeIDs, uint64(customValidatorWeight))
}

func loadCostonValidators() []Validator {
	nodeIDs := []string{
		"NodeID-5dDZXn99LCkDoEi6t9gTitZuQmhokxQTc",
		"NodeID-EkH8wyEshzEQBToAdR7Fexxcj9rrmEEHZ",
		"NodeID-FPAwqHjs8Mw8Cuki5bkm3vSVisZr8t2Lu",
		"NodeID-AQghDJTU3zuQj73itPtfTZz6CxsTQVD3R",
		"NodeID-HaZ4HpanjndqSuN252chFsTysmdND5meA",
	}
	return createValidators(nodeIDs, uint64(costonValidatorWeight))
}

func loadSongbirdValidators() []Validator {
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
