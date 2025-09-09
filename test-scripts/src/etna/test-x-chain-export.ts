import { avm, evm, utils, TransferableOutput } from "@flarenetwork/flarejs";
import { issueCChainTx, issueXChainTx, localFlareContext } from "../utils";
import { runTest } from "../runner";

async function XtoCExport(amountFLR: number) {
  const ctx = await localFlareContext();
  const fee = 1; // in FLR

  // Create and issue a X to C export transaction
  console.log(`Creating X to C export transaction for ${amountFLR} FLR...`);

  const { utxos: utxosp } = await ctx.avmapi.getUTXOs({
    addresses: [ctx.addressX],
  });
  const exportTx = avm.newExportTx(
    ctx.context,
    ctx.context.cBlockchainID,
    [utils.bech32ToBytes(ctx.addressX)],
    utxosp,
    [
      TransferableOutput.fromNative(
        ctx.context.avaxAssetID,
        BigInt(amountFLR * 1e9),
        [utils.bech32ToBytes(ctx.addressX)],
      ),
    ],
  );
  await issueXChainTx(ctx.avmapi, exportTx, ctx.privateKey);

  // Create and issue a X to C chain import transaction
  console.log("\nCreating X to C chain import transaction");

  const { utxos: utxosc } = await ctx.evmapi.getUTXOs({
    sourceChain: "X",
    addresses: ["C-" + ctx.addressX.slice(2)],
  });

  const importTx = evm.newImportTx(
    ctx.context,
    utils.hexToBuffer(ctx.addressC),
    [utils.bech32ToBytes(ctx.addressX)],
    utxosc,
    ctx.context.xBlockchainID,
    BigInt(fee * 1e9),
  );

  await issueCChainTx(ctx.evmapi, importTx, ctx.privateKey);
}

runTest(() => XtoCExport(100));
