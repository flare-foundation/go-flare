import { avm, evm, utils } from "@flarenetwork/flarejs";
import { issueCChainTx, issueXChainTx, localFlareContext } from "./utils";
import { runTest } from "./runner";

async function CtoPExport(amountFLR: number) {
  const ctx = await localFlareContext();
  const fee = 1; // in FLR
  const txCount = await ctx.provider.getTransactionCount(ctx.addressC);

  // Create and issue a C to X export transaction
  console.log(`Creating C to X export transaction for ${amountFLR} FLR...`);

  const exportTx = evm.newExportTx(
    ctx.context,
    BigInt(amountFLR * 1e9),
    ctx.context.xBlockchainID,
    utils.hexToBuffer(ctx.addressC),
    [utils.bech32ToBytes(ctx.addressX)],
    BigInt(fee * 1e9),
    BigInt(txCount),
  );

  await issueCChainTx(ctx.evmapi, exportTx, ctx.privateKey);

  // Create and issue a C to X chain import transaction
  console.log("\nCreating C to X chain import transaction");

  const { utxos } = await ctx.avmapi.getUTXOs({
    sourceChain: "C",
    addresses: [ctx.addressX],
  });
  const importTx = avm.newImportTx(
    ctx.context,
    ctx.context.cBlockchainID,
    utxos,
    [utils.bech32ToBytes(ctx.addressX)],
    [utils.bech32ToBytes(ctx.addressX)],
  );

  await issueXChainTx(ctx.avmapi, importTx, ctx.privateKey);
}

runTest(() => CtoPExport(100));
