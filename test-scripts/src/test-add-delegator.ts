import { networkIDs, pvm, UnsignedTx, utils } from "@flarenetwork/flarejs";
import { issuePChainTx, localFlareContext } from "./utils";
import { runTest } from "./runner";

async function addDelegator(nodeId: string, endTime: number, weight: number) {
    const ctx = await localFlareContext();

    // Create the transaction to add a delegator
    console.log(`Creating add delegator transaction for node ${nodeId} with weight ${weight}...`);

    const { utxos } = await ctx.pvmapi.getUTXOs({
        addresses: [ctx.addressP]
    });

    let tx: UnsignedTx;
    if (ctx.isEtnaForkActive) {
        console.log("Etna fork is active, using new transaction format.");
        const feeState = await ctx.pvmapi.getFeeState();
        tx = pvm.e.newAddPermissionlessDelegatorTx(
            {
                feeState,
                utxos,
                nodeId,
                subnetId: networkIDs.PrimaryNetworkID.toString(),
                start: BigInt(Date.now()) / 1000n,
                end: BigInt(endTime),
                weight: BigInt(weight * 1e9),
                fromAddressesBytes: [utils.bech32ToBytes(ctx.addressP)],
                rewardAddresses: [utils.bech32ToBytes(ctx.addressP)],
            },
            ctx.context,
        );
    } else {
        console.log("Etna fork is not active, using legacy transaction format.");
        tx = pvm.newAddPermissionlessDelegatorTx(
        ctx.context,
        utxos,
        [utils.bech32ToBytes(ctx.addressP)],
        nodeId,
        networkIDs.PrimaryNetworkID.toString(),
        BigInt(Date.now()) / 1000n,
        BigInt(endTime),
        BigInt(weight * 1e9),
        [utils.bech32ToBytes(ctx.addressP)],
        );
    }
    await issuePChainTx(ctx.pvmapi, tx, ctx.privateKey);
}

runTest(() => addDelegator(
    'NodeID-7Xhw2mDxuDS44j42TCB6U5579esbSt3Lg',
    Math.ceil(Date.now() / 1000) + 60 * 60 + 5, // 1 hour (+ 5 seconds) from now
    10_000
))