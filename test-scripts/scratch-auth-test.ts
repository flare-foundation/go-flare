// Temporary scratch test: does addTxSignatures sign the auth credential of
// SetAutoRenewedValidatorConfigTx?
import {
  pvm,
  utils,
  addTxSignatures,
  secp256k1,
  Utxo,
  avaxSerial,
  Id,
  Int,
  BigIntPr,
  TransferOutput,
  OutputOwners,
} from "@flarenetwork/flarejs";

const TestPrivateKey =
  "0x56289e99c94b6912bfc12adc093c9b51124f0dc54ac7a766b2bc5ccf558d8027";

async function main() {
  const privateKey = utils.hexToBuffer(TestPrivateKey);
  const publicKey = secp256k1.getPublicKey(privateKey);
  const addrBytes = secp256k1.publicKeyBytesToAddress(publicKey);

  const context: any = {
    networkID: 162,
    hrp: "localflare",
    pBlockchainID: "11111111111111111111111111111111LpoYY",
    avaxAssetID: "FvwEAhmxKfeiG8SnEvq42hc6whRyY3EFYAvebMqDNDGCgxN5Z",
    baseTxFee: 1000000n,
    platformFeeConfig: {
      weights: { 0: 1, 1: 1, 2: 1, 3: 1 },
      maxCapacity: 1000000n,
      maxPerSecond: 1000n,
      targetPerSecond: 500n,
      minPrice: 1n,
      excessConversionConstant: 5000n,
    },
  };

  const utxo = new Utxo(
    new avaxSerial.UTXOID(
      Id.fromString("2QGB9NgqRDs2XtWhLtoTFyyLNYDvpSFtwbFAP1vDM7uWiwqoib"),
      new Int(0),
    ),
    Id.fromString(context.avaxAssetID),
    new TransferOutput(
      new BigIntPr(50_000_000_000n),
      OutputOwners.fromNative([addrBytes]),
    ),
  );

  const feeState: any = {
    capacity: 1000000n,
    excess: 0n,
    price: 1n,
    timestamp: new Date().toISOString(),
  };

  const tx = pvm.newSetAutoRenewedValidatorConfigTx(
    {
      feeState,
      utxos: [utxo],
      fromAddressesBytes: [addrBytes],
      validatorTxId: "2QGB9NgqRDs2XtWhLtoTFyyLNYDvpSFtwbFAP1vDM7uWiwqoib",
      auth: [0],
      autoCompoundRewardShares: 0,
      period: 3600n,
    },
    context,
  );

  await addTxSignatures({
    unsignedTx: tx,
    privateKeys: [privateKey],
  });

  const creds = tx.getCredentials();
  console.log("number of credentials:", creds.length);
  creds.forEach((c, i) => {
    console.log(`credential ${i}:`, JSON.stringify(c.toJSON()));
  });
  console.log("hasAllSignatures:", tx.hasAllSignatures());

  // Now manually sign the auth credential (last credential) and re-check
  const sig = await secp256k1.sign(tx.toBytes(), privateKey);
  tx.addSignatureAt(sig, creds.length - 1, 0);
  console.log("after manual auth signature:");
  creds.forEach((c, i) => {
    console.log(`credential ${i}:`, JSON.stringify(c.toJSON()));
  });
}

main().catch((e) => {
  console.error(e);
  process.exit(1);
});
