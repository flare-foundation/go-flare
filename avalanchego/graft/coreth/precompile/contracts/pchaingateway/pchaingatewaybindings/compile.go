// (c) 2026, Flare Network. All rights reserved.
// See the file LICENSE for licensing terms.

// Package pchaingatewaybindings holds the Solidity interface of the P-chain
// gateway precompile. The ABI in the parent directory is generated from it.
package pchaingatewaybindings

// Compile IPChainGateway to generate the ABI at the parent level (requires solc >= 0.8).
//go:generate solc -o .. --overwrite --abi --pretty-json --evm-version cancun IPChainGateway.sol
