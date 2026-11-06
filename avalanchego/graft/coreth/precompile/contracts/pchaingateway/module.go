// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package pchaingateway

import (
	"github.com/ava-labs/libevm/common"

	"github.com/ava-labs/avalanchego/graft/coreth/precompile/modules"
)

// ConfigKey is the key used in json config files to specify this precompile
// config. It must be unique across all precompiles.
const ConfigKey = "pChainGatewayConfig"

// ContractAddress is the address of the P-chain gateway. Flare-only
// precompiles start at 0x0300...; the 0x0200... range is reserved for
// upstream (coreth/subnet-evm) precompiles.
var ContractAddress = common.HexToAddress("0x0300000000000000000000000000000000000000")

// Module is the precompile module. It is registered by init and enabled by a
// PrecompileUpgrade keyed on the Helicon activation time. The gateway keeps no
// state; the account itself is activated by the SAE VM (vms/saevm/cchain) at
// the Helicon boundary.
var Module = modules.Module{
	ConfigKey: ConfigKey,
	Address:   ContractAddress,
	Contract:  PChainGatewayPrecompile,
}

func init() {
	if err := modules.RegisterModule(Module); err != nil {
		panic(err)
	}
}
