// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package pchaingateway

import (
	"fmt"

	"github.com/ava-labs/libevm/common"

	"github.com/ava-labs/avalanchego/graft/coreth/precompile/contract"
	"github.com/ava-labs/avalanchego/graft/coreth/precompile/modules"
	"github.com/ava-labs/avalanchego/graft/coreth/precompile/precompileconfig"
)

var _ contract.Configurator = (*configurator)(nil)

// ConfigKey is the key used in json config files to specify this precompile
// config. It must be unique across all precompiles.
const ConfigKey = "pChainGatewayConfig"

// ContractAddress is the address of the P-chain gateway. Flare-only
// precompiles start at 0x0300...; the 0x0200... range is reserved for
// upstream (coreth/subnet-evm) precompiles.
var ContractAddress = common.HexToAddress("0x0300000000000000000000000000000000000000")

// Module is the precompile module. It is registered by init and enabled by a
// PrecompileUpgrade keyed on the Helicon activation time.
var Module = modules.Module{
	ConfigKey:    ConfigKey,
	Address:      ContractAddress,
	Contract:     PChainGatewayPrecompile,
	Configurator: &configurator{},
}

type configurator struct{}

func init() {
	if err := modules.RegisterModule(Module); err != nil {
		panic(err)
	}
}

// MakeConfig returns a new precompile config instance.
func (*configurator) MakeConfig() precompileconfig.Config {
	return new(Config)
}

// Configure is a no-op: the gateway keeps no state; the account itself is
// activated by the SAE VM (vms/saevm/cchain) at the Helicon boundary.
func (*configurator) Configure(_ precompileconfig.ChainConfig, cfg precompileconfig.Config, _ contract.StateDB, _ contract.ConfigurationBlockContext) error {
	if _, ok := cfg.(*Config); !ok {
		return fmt.Errorf("expected config type %T, got %T: %v", &Config{}, cfg, cfg)
	}
	return nil
}
