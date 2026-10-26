// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

package pchaingateway

import (
	"errors"

	"github.com/ava-labs/avalanchego/graft/coreth/precompile/precompileconfig"
)

var (
	_ precompileconfig.Config = (*Config)(nil)

	errNoTimestamp = errors.New("P-chain gateway config requires a block timestamp")
)

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

// Verify checks the config. The gateway reads its bound from the header field
// added at Helicon; before Helicon getPChainHeight simply reverts, so there is
// no fork requirement to check beyond a set timestamp.
func (c *Config) Verify(precompileconfig.ChainConfig) error {
	if c.Timestamp() == nil {
		return errNoTimestamp
	}
	return nil
}

// Equal returns true if s is a gateway config with the same upgrade.
func (c *Config) Equal(s precompileconfig.Config) bool {
	other, ok := s.(*Config)
	if !ok {
		return false
	}
	return c.Upgrade.Equal(&other.Upgrade)
}
