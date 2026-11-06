// Copyright (C) 2019-2025, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// Package granite defines the ACP-176 parameters that diverged on Flare-family
// chains after the Granite upgrade. Pre-Helicon, coreth's dynamic fee logic
// consumed a full parameter set from here (MinGasPrice and a shorter
// TimeToFillCapacity); that VM was deleted together with the Granite fee
// machinery in avalanchego v1.15.2, so only the base-fee floor remains. It is
// still consensus-relevant: the SAE VM seeds the ACP-283 minimum gas price at
// Helicon from it on Flare-family networks (see
// vms/saevm/cchain/dynamic.FlareInitialPriceExponent). Changing it requires a
// coordinated network upgrade.
package granite

import "github.com/ava-labs/avalanchego/graft/evm/utils"

// MinGasPrice (M) is the dynamic ACP-176 base-fee floor on Flare-family chains
// from Granite until Helicon, and the ACP-283 minimum gas price the SAE VM
// starts from at Helicon. Diverges from the default acp176 MinGasPrice of
// 1 Wei.
const MinGasPrice = 500 * utils.GWei
