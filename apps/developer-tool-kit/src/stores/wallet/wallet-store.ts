import { create } from "zustand";

import { APP_CONFIG } from "@/config/app-config";
import { connectMetaMask, getEthereumProvider, switchToTitanNetwork } from "@/lib/titan/ethereum";
import { formatWeiToTitan } from "@/lib/titan/format";
import { titanRpc } from "@/lib/titan/rpc";

type WalletState = {
  address: string;
  chainId: string;
  titanBalance: string;
  signedIn: boolean;
  signMessage: string;
  isLoading: boolean;
  isRefreshingBalance: boolean;
  error: string;
  isHydrated: boolean;
  connect: () => Promise<void>;
  disconnect: () => void;
  signIn: () => Promise<void>;
  refreshBalance: () => Promise<void>;
  syncFromAccounts: (accounts: string[]) => Promise<void>;
  hydrate: () => Promise<void>;
};

async function fetchBalanceForAddress(walletAddress: string): Promise<string> {
  try {
    const balHex = (await titanRpc("eth_getBalance", [walletAddress, "latest"])) as string;
    return formatWeiToTitan(balHex);
  } catch {
    return "—";
  }
}

export const useWalletStore = create<WalletState>((set, get) => ({
  address: "",
  chainId: "",
  titanBalance: "",
  signedIn: false,
  signMessage: "",
  isLoading: false,
  isRefreshingBalance: false,
  error: "",
  isHydrated: false,

  refreshBalance: async () => {
    const { address } = get();
    if (!address) return;

    set({ isRefreshingBalance: true });
    const balance = await fetchBalanceForAddress(address);
    set({ titanBalance: balance, isRefreshingBalance: false });
  },

  syncFromAccounts: async (accounts: string[]) => {
    const selectedAddress = accounts?.[0] ?? "";
    if (!selectedAddress) {
      set({ address: "", chainId: "", titanBalance: "", signedIn: false, signMessage: "" });
      return;
    }

    const provider = getEthereumProvider();
    let selectedChain = get().chainId;
    if (provider) {
      selectedChain = (await provider.request({ method: "eth_chainId" })) as string;
    }

    const balance = await fetchBalanceForAddress(selectedAddress);
    set({ address: selectedAddress, chainId: selectedChain, titanBalance: balance });
  },

  hydrate: async () => {
    const provider = getEthereumProvider();
    if (!provider) {
      set({ isHydrated: true });
      return;
    }

    try {
      const accounts = (await provider.request({ method: "eth_accounts" })) as string[];
      if (accounts?.[0]) {
        await get().syncFromAccounts(accounts);
      }
    } catch {
      // MetaMask not available or permission not granted yet.
    } finally {
      set({ isHydrated: true });
    }
  },

  connect: async () => {
    set({ error: "", isLoading: true });
    try {
      const { address, chainId } = await connectMetaMask();
      const balance = await fetchBalanceForAddress(address);
      set({ address, chainId, titanBalance: balance });
    } catch (connectError) {
      set({ error: connectError instanceof Error ? connectError.message : "Wallet connection failed." });
    } finally {
      set({ isLoading: false });
    }
  },

  disconnect: () => {
    set({
      address: "",
      chainId: "",
      titanBalance: "",
      signedIn: false,
      signMessage: "",
      error: "",
    });
  },

  signIn: async () => {
    set({ error: "" });
    const provider = getEthereumProvider();
    if (!provider) {
      set({ error: "MetaMask not found. Install MetaMask and refresh the page." });
      return;
    }

    set({ isLoading: true });
    try {
      await switchToTitanNetwork(provider);
      const accounts = (await provider.request({ method: "eth_requestAccounts" })) as string[];
      const selectedAddress = accounts?.[0];

      if (!selectedAddress) {
        throw new Error("No wallet account returned by MetaMask.");
      }

      const message = `Titan Explorer sign-in\nAddress: ${selectedAddress}\nTimestamp: ${new Date().toISOString()}\nOrigin: ${window.location.origin}`;
      const signature = (await provider.request({
        method: "personal_sign",
        params: [message, selectedAddress],
      })) as string;

      const selectedChain = (await provider.request({ method: "eth_chainId" })) as string;
      const balance = await fetchBalanceForAddress(selectedAddress);
      set({
        address: selectedAddress,
        chainId: selectedChain,
        titanBalance: balance,
        signedIn: true,
        signMessage: `Signed in with wallet. Signature: ${signature.slice(0, 14)}...${signature.slice(-10)}`,
      });
    } catch (signInError) {
      set({ error: signInError instanceof Error ? signInError.message : "Wallet sign-in failed." });
    } finally {
      set({ isLoading: false });
    }
  },
}));

export function isWalletConnected(state: Pick<WalletState, "address">): boolean {
  return Boolean(state.address);
}

export function isOnTitanChain(chainId: string): boolean {
  return chainId.toLowerCase() === APP_CONFIG.titan.chainIdHex.toLowerCase();
}