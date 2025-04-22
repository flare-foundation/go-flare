# Release Notes: Flare and Songbird networks

Here are listed specific changes to the code for the Flare and Songbird networks. For a comprehensive list of general changes, see [here](./avalanchego/RELEASES.md) for the AvalancheGo project and [here](./coreth/RELEASES.md) for the Coreth project.

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
