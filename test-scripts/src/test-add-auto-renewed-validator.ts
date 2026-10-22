import { pvm, utils } from "@flarenetwork/flarejs";
import { issuePChainTx, localFlareContext } from "./utils";
import { runTest } from "./runner";

// Arbitrary valid node ID (the node does not have to be running); different from
// the one in test-add-validator.ts so both scripts can run on the same network.
const nodeID = 'NodeID-GWPcbFJZFfZreETSoWjPimr846mXEKCtu';
const blsPublicKey = '0xa926858ebfba247e6d759e40bc82f92c20f359769e3538d58562bf30077f5e4c62b3b149f11b8e68a2f5599acd782de2';
const blsSignature = '0xb48df13f6e0167e2f40cc7c02cf81bf4a2c0b1b686395a56690deb6b5876c0a654ac41253455f41a6c65ac2fbf712d980d9012375edf73efb341d0961ac09197bf5f807192b6cd9f44c9e9032b0e035e4d07ba7eebccf434688610a2f51bc492'

async function addAutoRenewedValidator(nodeId: string, period: number, weight: number) {
    const ctx = await localFlareContext();

    // Create the transaction to add an auto-renewed validator (ACP-236)
    console.log(`Creating add auto-renewed validator transaction for node ${nodeId} with weight ${weight} and period ${period}s...`);

    const { utxos } = await ctx.pvmapi.getUTXOs({
        addresses: [ctx.addressP]
    });
    const feeState = await ctx.pvmapi.getFeeState();

    // Unlike AddPermissionlessValidatorTx there is no start/end time: validation
    // starts at the current chain time and is renewed every `period` seconds
    // until the period is set to 0 via SetAutoRenewedValidatorConfigTx.
    const tx = pvm.newAddAutoRenewedValidatorTx(
        {
            feeState,
            utxos,
            nodeId,
            period: BigInt(period),
            weight: BigInt(weight * 1e9),
            shares: 10_0000,
            // Fraction of rewards restaked at each cycle end, in millionths.
            // Rewards are 0 on Flare networks, so the value has no effect.
            autoCompoundRewardShares: 0,
            fromAddressesBytes: [utils.bech32ToBytes(ctx.addressP)],
            rewardAddresses: [utils.bech32ToBytes(ctx.addressP)],
            delegatorRewardsOwner: [utils.bech32ToBytes(ctx.addressP)],
            // Owner(s) allowed to change the auto-renew config later
            // (see test-set-auto-renewed-validator-config.ts).
            ownerAddresses: [utils.bech32ToBytes(ctx.addressP)],
            publicKey: utils.hexToBuffer(blsPublicKey),
            signature: utils.hexToBuffer(blsSignature),
        },
        ctx.context,
    );
    await issuePChainTx(ctx.pvmapi, tx, ctx.privateKey);

    // The auto-renew config fields are flattened into the validator object
    // (not part of the flarejs response type yet, hence the cast)
    const { validators } = await ctx.pvmapi.getCurrentValidators({ nodeIDs: [nodeId] });
    const validator = validators[0] as any;
    console.log(`Validator added with txID ${validator?.txID}, next period: ${validator?.nextPeriod}s, auto compound reward shares: ${validator?.autoCompoundRewardShares}`);
}

runTest(() => addAutoRenewedValidator(
    nodeID,
    15 * 60, // 15 minutes
    10_000
))
