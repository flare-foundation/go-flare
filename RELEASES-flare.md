# Release Notes: Flare and Songbird networks

Here are listed specific changes to the code for the Flare and Songbird networks. For a comprehensive list of general changes, see [here](./avalanchego/RELEASES.md) for the AvalancheGo project — which, since v1.14.x, also contains Coreth (grafted in at `avalanchego/graft/coreth`).

## v1.14.2

This release updates the Flare and Songbird codebase to AvalancheGo v1.14.2.

The upgrade is optional but encouraged. It is backwards compatible with v1.14.0 (the minimum compatible version remains v1.14.0) and does not schedule any new fork times.

### Note:

 * Coreth and Subnet-EVM are now grafted into the AvalancheGo tree — there is no separate `coreth` module anymore. Go import paths change from `github.com/ava-labs/coreth/...` to `github.com/ava-labs/avalanchego/graft/coreth/...`, which affects downstream tools that import Coreth packages.
 * Building from source now requires Go 1.25.11 or later.

## v1.14.0

This release updates the Flare and Songbird codebase to AvalancheGo v1.14.0 (Granite upgrade).

The changes go into effect
 * June 11, 2026 at 12 PM UTC for the Coston network,
 * June 16, 2026 at 12 PM UTC for the Coston2 network,
 * July 7, 2026 at 12 PM UTC for the Songbird network,
 * July 14, 2026 at 12 PM UTC for the Flare network.

Nodes should be updated to this version before these times.

### Note:

 * The minimum C-chain gas price (base fee floor) is raised to 500 GWei.
 * For the Flare network, the maximum validator stake amount is increased from 200 million to 300 million FLR.
 * For the Flare and Coston2 networks, the minimum delegation fee for validators is set to 20% (previously 0%).
 * The deprecated top-level `rewardOwner` field has been removed from validator entries returned by `platform.getCurrentValidators`. It previously duplicated `validationRewardOwner`.

## v1.13.0

The changes go into effect
 * March 17, 2026 at 12 PM UTC for the Coston network,
 * March 24, 2026 at 12 PM UTC for the Coston2 network,
 * March 31, 2026 at 12 PM UTC for the Songbird network,
 * April 14, 2026 at 12 PM UTC for the Flare network.

## v1.12.0

The changes go into effect
 * November 13, 2025 at 10 AM UTC for the Coston network,
 * November 13, 2025 at 2 PM UTC for the Coston2 network,
 * November 25, 2025 at 12 PM UTC for the Songbird network,
 * December 2, 2025 at 12 PM UTC for the Flare network.

### Note:

 * In contrast to Avalanche, the minimum C-chain base fee after the fork remains at 25 GWei.
 * Minimum price per unit of gas used in P-chain [dynamic transaction fee calculation](https://github.com/avalanche-foundation/ACPs/blob/main/ACPs/103-dynamic-fees) is set to 250 nFLR.

## v1.11.0

The changes go into effect
 * June 24, 2025 at 12 PM UTC for the Coston2 network,
 * July 1, 2025 at 12 PM UTC for the Coston network,
 * July 22, 2025 at 12 PM UTC for the Songbird network,
 * August 5, 2025 at 12 PM UTC for the Flare network.

### Note:

- Avalanche added in v1.10.3 a new config `--http-allowed-hosts` with a default value of `localhost`. Set `--http-allowed-hosts="*"` to allow RPC calls for all hosts.

## v1.10.0

The changes go into effect on May 6, 2025 at 12 PM UTC for the Songbird network and on May 13, 2025 at 12 PM UTC for the Flare network.

### Specific changes:

- Return delegators in `platform.getCurrentValidators` when the environment variable `COMPLETE_GET_VALIDATORS` is set to `true`. (Only the number of delegators is returned by default unless only one node is specified.)

- `AddPermissionlessValidatorTx` and `AddPermissionlessDelegatorTx` transactions are enabled for the primary network (they were disabled in 1.9.0) and are the preferred way to add delegators and validators as we plan to disable `AddValidatorTx` and `AddDelegatorTx` transactions in v1.11.0 to align with Avalanche.

- Reduce the target gas (for the 10-second rolling window used for dynamic fee calculation) to 15,000,000 from 150,000,000 for the Songbird network to be consistent with the Flare network.

### Note:

- We recommend using `AddPermissionlessValidatorTx` and `AddPermissionlessDelegatorTx` transactions for adding validators and delegators after the fork. When creating `AddPermissionlessValidatorTx` you need to specify the BLS proof of posession (BLS public key and signature). You can find the BLS proof of posession in the response of `info.getNodeID` API. Path to the BLS private key is specified with `--staking-signer-key-file`. The default is `~/.avalanchego/staking/signer.key`. If it does not exist, it will be populated with a new key. You should copy it to your persistent storage along with the existing `staker.crt` and `staker.key` files.

## v1.9.0

### Specific changes:

- `AddPermissionlessValidatorTx` and `AddPermissionlessDelegatorTx` transactions are disabled.

- Subnets are disabled.

- We allow Ethereum-style signatures for P-chain transactions ([EIP-191](https://eips.ethereum.org/EIPS/eip-191)).
