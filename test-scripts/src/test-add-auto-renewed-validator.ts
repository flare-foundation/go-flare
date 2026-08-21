import { pvm, utils } from "@flarenetwork/flarejs";
import { issuePChainTx, localFlareContext } from "./utils";
import { runTest } from "./runner";

// Arbitrary valid node ID (the node does not have to be running); different from
// the one in test-add-validator.ts so both scripts can run on the same network.
const nodeID = 'NodeID-GWPcbFJZFfZreETSoWjPimr846mXEKCtu';
const blsPublicKey = '0xadb0203ebc76627d28fb9440272a2701b85f5c5d4266352686ea42666f5026f1fdabab59529932f1eddb317a6f7435f9';
const blsSignature = '0x926a1c21953babb12189ec88bfab5cb0060b385efa04c51abe4a3bc42266f80eebb84bcfaacfc8f075560ab444bda8de17bf9013ec20a80dfdfda4ae047a0b39669d153dcee94eb964fe194f5d55c5bba5939ff9ba07b728b6733d696c33ac5a'

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
    3600, // minimal staking period on localflare (1 hour)
    10_000
))
