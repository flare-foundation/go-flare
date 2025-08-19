import {
  CChainBalance,
  formatDecimal,
  PChainBalance,
  XChainBalance,
} from "./utils";

async function runWithBalances(func: () => Promise<void>) {
  // Log the current balances before calling the function
  let cBalanceFLR = await CChainBalance();
  let pBalanceFLR = await PChainBalance();
  let xBalanceFLR = await XChainBalance();

  console.log(`Current C chain balance: ${formatDecimal(cBalanceFLR, 18)} FLR`);
  console.log(`Current P chain balance: ${formatDecimal(pBalanceFLR, 9)} FLR`);
  console.log(
    `Current X chain balance: ${formatDecimal(xBalanceFLR, 9)} FLR\n`,
  );

  await func();

  // Log the balances after calling the function
  cBalanceFLR = await CChainBalance();
  pBalanceFLR = await PChainBalance();
  xBalanceFLR = await XChainBalance();

  console.log(`\nNew C chain balance: ${formatDecimal(cBalanceFLR, 18)} FLR`);
  console.log(`New P chain balance: ${formatDecimal(pBalanceFLR, 9)} FLR`);
  console.log(`New X chain balance: ${formatDecimal(xBalanceFLR, 9)} FLR\n`);
}

export async function runTest(func: () => Promise<void>) {
  try {
    await runWithBalances(func);
    console.log("Script completed successfully");
  } catch (error) {
    console.error("Script failed:", error);
  }
}
