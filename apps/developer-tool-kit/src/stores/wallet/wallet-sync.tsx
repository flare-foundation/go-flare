"use client";

import { useEffect } from "react";

import { getEthereumProvider } from "@/lib/titan/ethereum";

import { useWalletStore } from "./wallet-store";

export function WalletSync() {
  const hydrate = useWalletStore((s) => s.hydrate);
  const syncFromAccounts = useWalletStore((s) => s.syncFromAccounts);
  const refreshBalance = useWalletStore((s) => s.refreshBalance);
  const address = useWalletStore((s) => s.address);

  useEffect(() => {
    void hydrate();
  }, [hydrate]);

  useEffect(() => {
    const provider = getEthereumProvider();
    if (!provider?.on) return;

    const handleAccountsChanged = (accounts: unknown) => {
      void syncFromAccounts(accounts as string[]);
    };

    const handleChainChanged = (nextChainId: unknown) => {
      useWalletStore.setState({ chainId: String(nextChainId) });
      if (address) {
        void refreshBalance();
      }
    };

    provider.on("accountsChanged", handleAccountsChanged);
    provider.on("chainChanged", handleChainChanged);

    return () => {
      provider.removeListener?.("accountsChanged", handleAccountsChanged);
      provider.removeListener?.("chainChanged", handleChainChanged);
    };
  }, [address, refreshBalance, syncFromAccounts]);

  return null;
}