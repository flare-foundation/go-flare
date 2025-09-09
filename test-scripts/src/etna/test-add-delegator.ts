import { networkIDs, pvm, utils } from "@flarenetwork/flarejs";
import { issuePChainTx, localFlareContext } from "../utils";
import { runTest } from "../runner";

async function addDelegator(nodeId: string, endTime: number, weight: number) {
  const ctx = await localFlareContext();

  // Create the transaction to add a delegator
  console.log(
    `Creating add delegator transaction for node ${nodeId} with weight ${weight}...`,
  );

  const feeState = await ctx.pvmapi.getFeeState();
  const { utxos } = await ctx.pvmapi.getUTXOs({
    addresses: [ctx.addressP],
  });

  const tx = pvm.e.newAddPermissionlessDelegatorTx(
    {
      end: BigInt(endTime),
      feeState,
      fromAddressesBytes: [utils.bech32ToBytes(ctx.addressP)],
      nodeId,
      rewardAddresses: [utils.bech32ToBytes(ctx.addressP)],
      start: BigInt(Date.now()) / 1000n,
      subnetId: networkIDs.PrimaryNetworkID.toString(),
      utxos,
      weight: BigInt(weight * 1e9),
    },
    ctx.context,
  );

  await issuePChainTx(ctx.pvmapi, tx, ctx.privateKey);
}

runTest(() =>
  addDelegator(
    "NodeID-7Xhw2mDxuDS44j42TCB6U5579esbSt3Lg",
    Math.ceil(Date.now() / 1000) + 60 * 60 + 5, // 1 hour (+ 5 seconds) from now
    10_000,
  ),
);
