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
	customValidatorWeight = 200_000
	customValidatorEnv    = "CUSTOM_VALIDATORS"
)

var (
	defaultValidators = defaultValidatorSet{}
	errNotInitialized = errors.New("default validator set not initialized")
)

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
		// Todo: Add songbird network validators
		vdrs = []Validator{}
	case constants.CostonID:
		// Todo: Add coston network validators
		vdrs = []Validator{}
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

func loadCustomValidators() (vdrs []Validator) {
	weight := uint64(customValidatorWeight)
	customValidatorList := os.Getenv(customValidatorEnv)
	nodeIDs := strings.Split(customValidatorList, ",")
	for _, nodeID := range nodeIDs {
		if nodeID == "" {
			continue
		}
		shortID, err := ids.ShortFromPrefixedString(nodeID, ids.NodeIDPrefix)
		if err != nil {
			panic(fmt.Errorf("invalid custom validator node ID: %s", nodeID))
		}
		vdrs = append(vdrs, &validator{
			nodeID: ids.NodeID(shortID),
			weight: weight,
		})
	}
	return
}

func DefaultValidatorList() []Validator {
	return defaultValidators.list()
}

func IsDefaultValidator(vdrID ids.NodeID) bool {
	return defaultValidators.isValidator(vdrID)
}

func InitializeDefaultValidators(networkID uint32) {
	defaultValidators.initialize(networkID)
}
