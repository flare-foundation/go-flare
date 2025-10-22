import { networkIDs, pvm, UnsignedTx, utils } from "@flarenetwork/flarejs";
import { issuePChainTx, localFlareContext } from "./utils";
import { runTest } from "./runner";

const nodeID = 'NodeID-MFrZFVCXPv5iCn6M9K6XduxGTYp891xXZ';
const blsPublicKey = '0xadb0203ebc76627d28fb9440272a2701b85f5c5d4266352686ea42666f5026f1fdabab59529932f1eddb317a6f7435f9';
const blsSignature = '0x926a1c21953babb12189ec88bfab5cb0060b385efa04c51abe4a3bc42266f80eebb84bcfaacfc8f075560ab444bda8de17bf9013ec20a80dfdfda4ae047a0b39669d153dcee94eb964fe194f5d55c5bba5939ff9ba07b728b6733d696c33ac5a'

async function addValidator(nodeId: string, endTime: number, weight: number) {
    const ctx = await localFlareContext();

    // Create the transaction to add a validator
    console.log(`Creating add validator transaction for node ${nodeId} with weight ${weight}...`);

    const { utxos } = await ctx.pvmapi.getUTXOs({
        addresses: [ctx.addressP]
    });

    let tx: UnsignedTx;
    if (ctx.isEtnaForkActive) {
        console.log("Etna fork is active, using new transaction format.");
        const feeState = await ctx.pvmapi.getFeeState();
        tx = pvm.e.newAddPermissionlessValidatorTx(
            {
                feeState,
                utxos,
                nodeId,
                subnetId: networkIDs.PrimaryNetworkID.toString(),
                start: BigInt(Date.now()) / 1000n,
                end: BigInt(endTime),
                weight: BigInt(weight * 1e9),
                shares: 10_0000,
                fromAddressesBytes: [utils.bech32ToBytes(ctx.addressP)],
                rewardAddresses: [utils.bech32ToBytes(ctx.addressP)],
                delegatorRewardsOwner: [utils.bech32ToBytes(ctx.addressP)],
                publicKey: utils.hexToBuffer(blsPublicKey),
                signature: utils.hexToBuffer(blsSignature),
            },
            ctx.context,
        );
    } else {
        console.log("Etna fork is not active, using legacy transaction format.");
        tx = pvm.newAddPermissionlessValidatorTx(
            ctx.context,
            utxos,
            [utils.bech32ToBytes(ctx.addressP)],
            nodeId,
            networkIDs.PrimaryNetworkID.toString(),
            BigInt(Date.now()) / 1000n,
            BigInt(endTime),
            BigInt(weight * 1e9),
            [utils.bech32ToBytes(ctx.addressP)],
            [utils.bech32ToBytes(ctx.addressP)],
            10_0000,
            undefined,
            1,
            0n,
            utils.hexToBuffer(blsPublicKey),
            utils.hexToBuffer(blsSignature)
        );
    }

    await issuePChainTx(ctx.pvmapi, tx, ctx.privateKey);
}

runTest(() => addValidator(
    nodeID,
    Math.ceil(Date.now() / 1000) + 14 * 24 * 60 * 60 + 5, // 14 days (+ 5 seconds) from now
    10_000
))