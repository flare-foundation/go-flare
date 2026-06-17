"use client";

import { useCallback, useEffect, useState } from "react";

import { Copy, EllipsisVertical, Loader2, LogOut, RefreshCw, Wallet } from "lucide-react";

import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { SidebarMenu, SidebarMenuButton, SidebarMenuItem, useSidebar } from "@/components/ui/sidebar";
import { APP_CONFIG } from "@/config/app-config";
import { connectMetaMask, getEthereumProvider } from "@/lib/titan/ethereum";
import { formatWeiToTitan, shortAddress } from "@/lib/titan/format";
import { titanRpc } from "@/lib/titan/rpc";
import { cn } from "@/lib/utils";

export function NavWallet() {
  const { isMobile } = useSidebar();
  const [address, setAddress] = useState<string>("");
  const [chainId, setChainId] = useState<string>("");
  const [titanBalance, setTitanBalance] = useState<string>("");
  const [isLoading, setIsLoading] = useState(false);
  const [isRefreshingBalance, setIsRefreshingBalance] = useState(false);
  const [error, setError] = useState<string>("");

  const fetchBalance = useCallback(async (walletAddress: string) => {
    setIsRefreshingBalance(true);
    try {
      const balHex = (await titanRpc("eth_getBalance", [walletAddress, "latest"])) as string;
      setTitanBalance(formatWeiToTitan(balHex));
    } catch {
      setTitanBalance("—");
    } finally {
      setIsRefreshingBalance(false);
    }
  }, []);

  const syncFromProvider = useCallback(
    async (accounts: string[]) => {
      const selectedAddress = accounts?.[0] ?? "";
      setAddress(selectedAddress);

      if (!selectedAddress) {
        setChainId("");
        setTitanBalance("");
        return;
      }

      const provider = getEthereumProvider();
      if (provider) {
        const selectedChain = (await provider.request({ method: "eth_chainId" })) as string;
        setChainId(selectedChain);
      }

      await fetchBalance(selectedAddress);
    },
    [fetchBalance],
  );

  useEffect(() => {
    const provider = getEthereumProvider();
    if (!provider?.on) return;

    const handleAccountsChanged = (accounts: unknown) => {
      void syncFromProvider(accounts as string[]);
    };

    const handleChainChanged = (nextChainId: unknown) => {
      setChainId(String(nextChainId));
      if (address) {
        void fetchBalance(address);
      }
    };

    provider.on("accountsChanged", handleAccountsChanged);
    provider.on("chainChanged", handleChainChanged);

    return () => {
      provider.removeListener?.("accountsChanged", handleAccountsChanged);
      provider.removeListener?.("chainChanged", handleChainChanged);
    };
  }, [address, fetchBalance, syncFromProvider]);

  useEffect(() => {
    const provider = getEthereumProvider();
    if (!provider) return;

    void (async () => {
      try {
        const accounts = (await provider.request({ method: "eth_accounts" })) as string[];
        if (accounts?.[0]) {
          await syncFromProvider(accounts);
        }
      } catch {
        // MetaMask not available or permission not granted yet.
      }
    })();
  }, [syncFromProvider]);

  async function handleConnect() {
    setError("");
    setIsLoading(true);
    try {
      const { address: connectedAddress, chainId: connectedChainId } = await connectMetaMask();
      setAddress(connectedAddress);
      setChainId(connectedChainId);
      await fetchBalance(connectedAddress);
    } catch (connectError) {
      setError(connectError instanceof Error ? connectError.message : "Wallet connection failed.");
    } finally {
      setIsLoading(false);
    }
  }

  function handleDisconnect() {
    setAddress("");
    setChainId("");
    setTitanBalance("");
    setError("");
  }

  async function handleCopyAddress() {
    if (!address) return;
    try {
      await navigator.clipboard.writeText(address);
    } catch {
      // Clipboard unavailable.
    }
  }

  if (!address) {
    return (
      <SidebarMenu>
        <SidebarMenuItem>
          <SidebarMenuButton
            size="lg"
            onClick={handleConnect}
            disabled={isLoading}
            className="cursor-pointer"
            tooltip="Connect MetaMask"
          >
            <div className="flex size-8 items-center justify-center rounded-lg bg-sidebar-primary text-sidebar-primary-foreground">
              {isLoading ? <Loader2 className="size-4 animate-spin" /> : <Wallet className="size-4" />}
            </div>
            <div className="grid flex-1 text-left text-sm leading-tight">
              <span className="truncate font-medium">Connect MetaMask</span>
              <span className="truncate text-muted-foreground text-xs">
                {error || `Sign in to ${APP_CONFIG.titan.networkName}`}
              </span>
            </div>
          </SidebarMenuButton>
        </SidebarMenuItem>
      </SidebarMenu>
    );
  }

  const isOnTitanChain = chainId.toLowerCase() === APP_CONFIG.titan.chainIdHex.toLowerCase();

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <SidebarMenuButton
              size="lg"
              className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
            >
              <div className="flex size-8 items-center justify-center rounded-lg bg-emerald-600/15 text-emerald-600 dark:text-emerald-400">
                <Wallet className="size-4" />
              </div>
              <div className="grid flex-1 text-left text-sm leading-tight">
                <span className="truncate font-medium font-mono">{shortAddress(address)}</span>
                <span className="truncate text-muted-foreground text-xs">
                  {isRefreshingBalance ? (
                    <span className="inline-flex items-center gap-1">
                      <Loader2 className="size-3 animate-spin" />
                      Loading balance…
                    </span>
                  ) : (
                    <>
                      {titanBalance} {APP_CONFIG.titan.nativeToken.symbol}
                    </>
                  )}
                </span>
              </div>
              <EllipsisVertical className="ml-auto size-4" />
            </SidebarMenuButton>
          </DropdownMenuTrigger>
          <DropdownMenuContent
            className="w-(--radix-dropdown-menu-trigger-width) min-w-56 rounded-lg"
            side={isMobile ? "bottom" : "right"}
            align="end"
            sideOffset={4}
          >
            <DropdownMenuLabel className="p-0 font-normal">
              <div className="flex items-center gap-2 px-1 py-1.5 text-left text-sm">
                <div className="flex size-8 items-center justify-center rounded-lg bg-emerald-600/15 text-emerald-600 dark:text-emerald-400">
                  <Wallet className="size-4" />
                </div>
                <div className="grid flex-1 text-left text-sm leading-tight">
                  <span className="truncate font-medium font-mono">{shortAddress(address)}</span>
                  <span className="truncate text-muted-foreground text-xs">
                    {titanBalance} {APP_CONFIG.titan.nativeToken.symbol}
                  </span>
                </div>
              </div>
            </DropdownMenuLabel>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={handleCopyAddress}>
              <Copy />
              Copy address
            </DropdownMenuItem>
            <DropdownMenuItem onClick={() => fetchBalance(address)} disabled={isRefreshingBalance}>
              <RefreshCw className={cn(isRefreshingBalance && "animate-spin")} />
              Refresh balance
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              className={cn(!isOnTitanChain && "text-amber-600 focus:text-amber-600 dark:text-amber-400")}
              disabled
            >
              Network: {isOnTitanChain ? APP_CONFIG.titan.networkName : `Chain ${chainId}`}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={handleDisconnect}>
              <LogOut />
              Disconnect
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}