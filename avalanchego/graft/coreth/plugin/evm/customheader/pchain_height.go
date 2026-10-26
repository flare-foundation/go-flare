// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package customheader

import (
	"errors"
	"fmt"

	"github.com/ava-labs/libevm/core/types"

	"github.com/ava-labs/avalanchego/graft/coreth/plugin/evm/customtypes"
)

// errPChainHeightSet is returned when a coreth block carries the P-chain height
// marker. The marker belongs to SAE (Helicon and later) and must not appear on
// coreth blocks.
var errPChainHeightSet = errors.New("P-chain height marker should be nil")

// VerifyPChainHeight rejects any coreth block carrying a committed P-chain
// height. The field is written by the SAE block builder from the ProposerVM
// block context (Flare, Helicon and later); semanticVerify only runs on coreth's
// own block.Verify, so coreth never verifies an SAE block and the field must
// always be absent.
func VerifyPChainHeight(header *types.Header) error {
	if customtypes.GetHeaderExtra(header).PChainHeight != nil {
		return fmt.Errorf("%w: %s", errPChainHeightSet, header.Hash())
	}
	return nil
}
