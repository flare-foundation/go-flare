import { decodeFunctionResult, encodeFunctionData, type Abi } from "viem";

import { estimateGasViaRpc, fetchTitanGasPriceWei, toHex } from "@/lib/titan/gas";
import { getEthereumProvider, switchToTitanNetwork } from "@/lib/titan/ethereum";
import { parseWalletError } from "@/lib/titan/wallet-errors";

async function rpcCall<T>(method: string, params: unknown[]): Promise<T> {
  const res = await fetch("/api/titan/rpc", {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ method, params, node: "node1", chain: "C" }),
  });

  const json = (await res.json()) as { result?: T; error?: { message?: string } };
  if (json.error) {
    throw new Error(json.error.message ?? "RPC call failed.");
  }
  return json.result as T;
}

export async function readContractFunction<T>(input: {
  contractAddress: string;
  abi: Abi;
  functionName: string;
  args?: readonly unknown[];
}): Promise<T> {
  const data = encodeFunctionData({
    abi: input.abi,
    functionName: input.functionName,
    args: input.args ?? [],
  });

  const result = await rpcCall<string>("eth_call", [
    {
      to: input.contractAddress,
      data,
    },
    "latest",
  ]);

  return decodeFunctionResult({
    abi: input.abi,
    functionName: input.functionName,
    data: result as `0x${string}`,
  }) as T;
}

export async function writeContractFunction(input: {
  from: string;
  contractAddress: string;
  abi: Abi;
  functionName: string;
  args?: readonly unknown[];
}): Promise<string> {
  const provider = getEthereumProvider();
  if (!provider) {
    throw new Error("MetaMask not found. Connect your wallet first.");
  }

  await switchToTitanNetwork(provider);

  const data = encodeFunctionData({
    abi: input.abi,
    functionName: input.functionName,
    args: input.args ?? [],
  });

  let gasLimit: bigint;
  try {
    gasLimit = (await estimateGasViaRpc(input.from, data)) + BigInt(50_000);
  } catch (error) {
    throw new Error(parseWalletError(error, "Gas estimation failed."));
  }

  const gasPrice = await fetchTitanGasPriceWei();

  let txHash: string;
  try {
    txHash = (await provider.request({
      method: "eth_sendTransaction",
      params: [
        {
          from: input.from,
          to: input.contractAddress,
          data,
          value: "0x0",
          gas: toHex(gasLimit),
          gasPrice: toHex(gasPrice),
        },
      ],
    })) as string;
  } catch (error) {
    throw new Error(parseWalletError(error, "Transaction failed."));
  }

  await waitForReceipt(txHash);
  return txHash;
}

async function waitForReceipt(txHash: string, attempts = 40, delayMs = 1500): Promise<void> {
  for (let i = 0; i < attempts; i++) {
    const receipt = await rpcCall<{ status?: string } | null>("eth_getTransactionReceipt", [txHash]);
    if (receipt) {
      if (receipt.status === "0x0") {
        throw new Error("Transaction reverted.");
      }
      return;
    }
    await new Promise((resolve) => setTimeout(resolve, delayMs));
  }
  throw new Error("Timed out waiting for transaction confirmation.");
}