"use client";

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
import { shortAddress } from "@/lib/titan/format";
import { cn } from "@/lib/utils";
import { isOnTitanChain, useWalletStore } from "@/stores/wallet/wallet-store";

export function NavWallet() {
  const { isMobile } = useSidebar();
  const address = useWalletStore((s) => s.address);
  const chainId = useWalletStore((s) => s.chainId);
  const titanBalance = useWalletStore((s) => s.titanBalance);
  const isLoading = useWalletStore((s) => s.isLoading);
  const isRefreshingBalance = useWalletStore((s) => s.isRefreshingBalance);
  const error = useWalletStore((s) => s.error);
  const connect = useWalletStore((s) => s.connect);
  const disconnect = useWalletStore((s) => s.disconnect);
  const refreshBalance = useWalletStore((s) => s.refreshBalance);

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
            size="default"
            onClick={() => void connect()}
            disabled={isLoading}
            className="cursor-pointer"
            tooltip="Connect MetaMask"
          >
            {isLoading ? <Loader2 className="animate-spin" /> : <Wallet />}
            <span className="truncate font-medium group-data-[collapsible=icon]:hidden">Connect MetaMask</span>
            <span className="truncate text-muted-foreground text-xs group-data-[collapsible=icon]:hidden">
              {error || `Sign in to ${APP_CONFIG.titan.networkName}`}
            </span>
          </SidebarMenuButton>
        </SidebarMenuItem>
      </SidebarMenu>
    );
  }

  const onTitanChain = isOnTitanChain(chainId);
  const walletTooltip = `${shortAddress(address)} · ${titanBalance} ${APP_CONFIG.titan.nativeToken.symbol}`;

  return (
    <SidebarMenu>
      <SidebarMenuItem>
        <DropdownMenu>
          <DropdownMenuTrigger asChild>
            <SidebarMenuButton
              size="default"
              tooltip={walletTooltip}
              className="data-[state=open]:bg-sidebar-accent data-[state=open]:text-sidebar-accent-foreground"
            >
              <Wallet className="text-emerald-600 dark:text-emerald-400" />
              <span className="truncate font-medium font-mono group-data-[collapsible=icon]:hidden">
                {shortAddress(address)}
              </span>
              <span className="truncate text-muted-foreground text-xs group-data-[collapsible=icon]:hidden">
                {isRefreshingBalance ? (
                  <span className="inline-flex items-center gap-1">
                    <Loader2 className="size-3 animate-spin" />
                    Loading…
                  </span>
                ) : (
                  <>
                    {titanBalance} {APP_CONFIG.titan.nativeToken.symbol}
                  </>
                )}
              </span>
              <EllipsisVertical className="ml-auto size-4 group-data-[collapsible=icon]:hidden" />
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
            <DropdownMenuItem onClick={() => void refreshBalance()} disabled={isRefreshingBalance}>
              <RefreshCw className={cn(isRefreshingBalance && "animate-spin")} />
              Refresh balance
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem
              className={cn(!onTitanChain && "text-amber-600 focus:text-amber-600 dark:text-amber-400")}
              disabled
            >
              Network: {onTitanChain ? APP_CONFIG.titan.networkName : `Chain ${chainId}`}
            </DropdownMenuItem>
            <DropdownMenuSeparator />
            <DropdownMenuItem onClick={disconnect}>
              <LogOut />
              Disconnect
            </DropdownMenuItem>
          </DropdownMenuContent>
        </DropdownMenu>
      </SidebarMenuItem>
    </SidebarMenu>
  );
}