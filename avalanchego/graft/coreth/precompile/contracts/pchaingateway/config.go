// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package pchaingateway

import "github.com/ava-labs/avalanchego/graft/coreth/precompile/precompileconfig"

var _ precompileconfig.Config = (*Config)(nil)

// Config implements precompileconfig.Config. The gateway has no parameters;
// the config only carries the activation (or deactivation) timestamp.
type Config struct {
	precompileconfig.Upgrade
}

// NewDefaultConfig returns a config enabling the gateway at blockTimestamp.
func NewDefaultConfig(blockTimestamp *uint64) *Config {
	return &Config{
		Upgrade: precompileconfig.Upgrade{BlockTimestamp: blockTimestamp},
	}
}

// NewDisableConfig returns a config disabling the gateway at blockTimestamp.
func NewDisableConfig(blockTimestamp *uint64) *Config {
	return &Config{
		Upgrade: precompileconfig.Upgrade{
			BlockTimestamp: blockTimestamp,
			Disable:        true,
		},
	}
}

// Key returns the key for the gateway precompile config.
func (*Config) Key() string {
	return ConfigKey
}
