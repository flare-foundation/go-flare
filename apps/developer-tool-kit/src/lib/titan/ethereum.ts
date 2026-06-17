import { APP_CONFIG } from "@/config/app-config";

export type EthereumProvider = {
  request: (args: { method: string; params?: unknown[] }) => Promise<unknown>;
  on?: (event: string, handler: (...args: unknown[]) => void) => void;
  removeListener?: (event: string, handler: (...args: unknown[]) => void) => void;
};

export function getEthereumProvider(): EthereumProvider | undefined {
  return (window as Window & { ethereum?: EthereumProvider }).ethereum;
}

export async function switchToTitanNetwork(provider: EthereumProvider) {
  try {
    await provider.request({
      method: "wallet_switchEthereumChain",
      params: [{ chainId: APP_CONFIG.titan.chainIdHex }],
    });
  } catch (error) {
    const shouldAddChain =
      typeof error === "object" &&
      error !== null &&
      "code" in error &&
      Number((error as { code?: unknown }).code) === 4902;

    if (!shouldAddChain) {
      throw error;
    }

    await provider.request({
      method: "wallet_addEthereumChain",
      params: [
        {
          chainId: APP_CONFIG.titan.chainIdHex,
          chainName: APP_CONFIG.titan.networkName,
          nativeCurrency: {
            name: APP_CONFIG.titan.nativeToken.name,
            symbol: APP_CONFIG.titan.nativeToken.symbol,
            decimals: APP_CONFIG.titan.nativeToken.decimals,
          },
          rpcUrls: [APP_CONFIG.titan.rpcUrl],
          blockExplorerUrls: [APP_CONFIG.titan.explorerUrl],
        },
      ],
    });
  }
}

export async function connectMetaMask(): Promise<{ address: string; chainId: string }> {
  const provider = getEthereumProvider();
  if (!provider) {
    throw new Error("MetaMask not found. Install MetaMask and refresh the page.");
  }

  await switchToTitanNetwork(provider);
  const accounts = (await provider.request({ method: "eth_requestAccounts" })) as string[];
  const address = accounts?.[0];

  if (!address) {
    throw new Error("No wallet account returned by MetaMask.");
  }

  const chainId = (await provider.request({ method: "eth_chainId" })) as string;
  return { address, chainId };
}