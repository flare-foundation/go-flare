import { encodeDeployData, type Abi } from "viem";

import { getEthereumProvider, switchToTitanNetwork } from "@/lib/titan/ethereum";

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

function hexToNumber(hex: string): bigint {
  return BigInt(hex);
}

async function waitForReceipt(txHash: string, attempts = 40, delayMs = 1500): Promise<{ contractAddress?: string }> {
  const provider = getEthereumProvider();
  if (!provider) {
    throw new Error("MetaMask not found.");
  }

  for (let i = 0; i < attempts; i++) {
    const receipt = (await provider.request({
      method: "eth_getTransactionReceipt",
      params: [txHash],
    })) as { contractAddress?: string | null; status?: string } | null;

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

  const gasEstimateHex = (await provider.request({
    method: "eth_estimateGas",
    params: [
      {
        from: input.from,
        data: deployData,
      },
    ],
  })) as string;

  const gasLimit = hexToNumber(gasEstimateHex) + BigInt(50_000);

  const txHash = (await provider.request({
    method: "eth_sendTransaction",
    params: [
      {
        from: input.from,
        data: deployData,
        gas: `0x${gasLimit.toString(16)}`,
      },
    ],
  })) as string;

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