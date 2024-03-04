package validators

import (
	"fmt"
	"os"
	"strings"

	"github.com/ava-labs/avalanchego/ids"
)

const (
	customValidatorWeight = 200_000
	customValidatorEnv    = "CUSTOM_VALIDATORS"
)

func loadCustomValidators() Set {
	set := NewSet()
	weight := uint64(customValidatorWeight)
	customValidatorList := os.Getenv(customValidatorEnv)
	nodeIDs := strings.Split(customValidatorList, ",")
	for _, nodeID := range nodeIDs {
		if nodeID == "" {
			continue
		}
		shortID, err := ids.ShortFromPrefixedString(nodeID, ids.NodeIDPrefix)
		if err != nil {
			panic(fmt.Sprintf("invalid custom validator node ID: %s", nodeID))
		}
		err = set.AddWeight(ids.NodeID(shortID), weight)
		if err != nil {
			panic(fmt.Sprintf("could not add weight for validator (node: %x, weight: %d): %s", shortID, weight, err))
		}
	}
	return set
}
