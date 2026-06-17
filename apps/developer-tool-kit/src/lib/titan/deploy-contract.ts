import { encodeDeployData, type Abi } from "viem";

import { estimateGasViaRpc, fetchTitanGasPriceWei, toHex } from "@/lib/titan/gas";
import { getEthereumProvider, switchToTitanNetwork } from "@/lib/titan/ethereum";
import { parseWalletError } from "@/lib/titan/wallet-errors";

export type DeployContractInput = {
  from: string;
  abi: Abi;
  bytecode: string;
  constructorArgs: unknown[];
};

export type DeployContractResult = {
  transactionHash: string;
  contractAddress: string;
};

async function providerRequest<T>(method: string, params: unknown[]): Promise<T> {
  const provider = getEthereumProvider();
  if (!provider) {
    throw new Error("MetaMask not found. Install MetaMask and refresh the page.");
  }

  try {
    return (await provider.request({ method, params })) as T;
  } catch (error) {
    throw new Error(parseWalletError(error, `${method} failed.`));
  }
}

async function waitForReceipt(txHash: string, attempts = 40, delayMs = 1500): Promise<{ contractAddress?: string }> {
  for (let i = 0; i < attempts; i++) {
    const receipt = await providerRequest<{
      contractAddress?: string | null;
      status?: string;
    } | null>("eth_getTransactionReceipt", [txHash]);

    if (receipt) {
      if (receipt.status === "0x0") {
        throw new Error("Contract deployment transaction reverted.");
      }
      if (receipt.contractAddress) {
        return { contractAddress: receipt.contractAddress };
      }
    }

    await new Promise((resolve) => setTimeout(resolve, delayMs));
  }

  throw new Error("Timed out waiting for deployment receipt.");
}

export async function deployContract(input: DeployContractInput): Promise<DeployContractResult> {
  const provider = getEthereumProvider();
  if (!provider) {
    throw new Error("MetaMask not found. Install MetaMask and refresh the page.");
  }

  await switchToTitanNetwork(provider);

  const bytecodeHex = input.bytecode.startsWith("0x") ? input.bytecode : `0x${input.bytecode}`;
  const deployData = encodeDeployData({
    abi: input.abi,
    bytecode: bytecodeHex as `0x${string}`,
    args: input.constructorArgs as readonly unknown[],
  });

  let gasLimit: bigint;
  try {
    gasLimit = (await estimateGasViaRpc(input.from, deployData)) + BigInt(100_000);
  } catch (error) {
    throw new Error(
      parseWalletError(
        error,
        "Gas estimation failed. Check constructor args, wallet balance, and that Titan nodes are running.",
      ),
    );
  }

  const gasPrice = await fetchTitanGasPriceWei();

  const txHash = await providerRequest<string>("eth_sendTransaction", [
    {
      from: input.from,
      data: deployData,
      value: "0x0",
      gas: toHex(gasLimit),
      gasPrice: toHex(gasPrice),
    },
  ]);

  const receipt = await waitForReceipt(txHash);
  if (!receipt.contractAddress) {
    throw new Error("Deployment transaction mined but no contract address was returned.");
  }

  return {
    transactionHash: txHash,
    contractAddress: receipt.contractAddress,
  };
}

export function parseConstructorArgValue(type: string, raw: string): unknown {
  const trimmed = raw.trim();
  if (trimmed === "") {
    throw new Error("Constructor argument cannot be empty.");
  }

  if (type === "bool") {
    if (trimmed === "true") return true;
    if (trimmed === "false") return false;
    throw new Error(`Invalid bool value: ${trimmed}`);
  }

  if (type.startsWith("uint") || type.startsWith("int")) {
    if (!/^\d+$/.test(trimmed)) {
      throw new Error(`Invalid integer value for ${type}: ${trimmed}`);
    }
    return BigInt(trimmed);
  }

  if (type === "address") {
    if (!/^0x[0-9a-fA-F]{40}$/.test(trimmed)) {
      throw new Error(`Invalid address: ${trimmed}`);
    }
    return trimmed;
  }

  if (type === "string") {
    return trimmed;
  }

  throw new Error(`Unsupported constructor type in MVP: ${type}`);
}

type ConstructorAbiItem = Extract<Abi[number], { type: "constructor" }>;

export function getConstructorAbi(abi: Abi): ConstructorAbiItem | undefined {
  return abi.find((item): item is ConstructorAbiItem => item.type === "constructor");
}