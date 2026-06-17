"use client";

import { useEffect, useState } from "react";
import {
  Activity,
  AlertCircle,
  CheckCircle2,
  Copy,
  Loader2,
  RefreshCw,
  ShieldCheck,
  Wallet,
} from "lucide-react";
import { APP_CONFIG } from "@/config/app-config";
import { shortAddress } from "@/lib/titan/format";
import { Badge } from "@/components/ui/badge";
import { isOnTitanChain, isWalletConnected, useWalletStore } from "@/stores/wallet/wallet-store";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

interface NodeHealth {
  node: string;
  port: number;
  healthy: boolean;
  peers: number;
  chainId?: string;
  blockNumber?: string;
  error?: string;
}

const NODES = [
  { node: "node1", port: 9650 },
  { node: "node2", port: 9652 },
  { node: "node3", port: 9654 },
];

function StatCard({ title, value, sub, ok }: { title: string; value: string; sub?: string; ok?: boolean }) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">{title}</CardTitle>
        {ok !== undefined && (ok ? <CheckCircle2 className="h-4 w-4 text-green-500" /> : <AlertCircle className="h-4 w-4 text-red-500" />)}
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold">{value}</div>
        {sub && <p className="text-xs text-muted-foreground mt-1">{sub}</p>}
      </CardContent>
    </Card>
  );
}

export function NetworkOverview() {
  const [nodes, setNodes] = useState<NodeHealth[]>([]);
  const [loading, setLoading] = useState(true);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);
  const [copiedField, setCopiedField] = useState<string>("");
  const address = useWalletStore((s) => s.address);
  const walletChainId = useWalletStore((s) => s.chainId);
  const titanBalance = useWalletStore((s) => s.titanBalance);
  const signMessage = useWalletStore((s) => s.signMessage);
  const walletError = useWalletStore((s) => s.error);
  const isWalletConnectedNow = isWalletConnected({ address });

  async function fetchAll() {
    setLoading(true);
    try {
      const res = await fetch("/api/titan/rpc");
      const data = await res.json();
      setNodes(data.nodes ?? []);
      setLastUpdated(new Date());
    } catch { setNodes([]); }
    finally { setLoading(false); }
  }

  useEffect(() => {
    fetchAll();
    const id = setInterval(fetchAll, 10_000);
    return () => clearInterval(id);
  }, []);

  const healthyCount = nodes.filter((n) => n.healthy).length;
  const totalPeers = nodes.reduce((a, n) => a + n.peers, 0);
  const chainId = nodes.find((n) => n.chainId)?.chainId ?? "—";
  const blockNumber = nodes.find((n) => n.blockNumber)?.blockNumber ?? "—";

  async function copyValue(label: string, value: string) {
    try {
      await navigator.clipboard.writeText(value);
      setCopiedField(label);
      setTimeout(() => setCopiedField(""), 1500);
    } catch {
      setCopiedField("");
    }
  }

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Titan Network</h1>
          <p className="text-sm text-muted-foreground">UAT · Chain ID {chainId} · Network ID 781337</p>
        </div>
        <Button variant="outline" size="sm" onClick={fetchAll} disabled={loading}>
          {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
          Refresh
        </Button>
      </div>
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <StatCard title="Nodes Online" value={`${healthyCount} / ${nodes.length}`} sub={`${nodes.length - healthyCount} unhealthy`} ok={healthyCount === nodes.length} />
        <StatCard title="Chain ID" value={chainId} sub="C-Chain" />
        <StatCard title="Latest Block" value={blockNumber} sub="C-Chain head" />
        <StatCard title="Total Peers" value={String(totalPeers)} sub="across all nodes" />
      </div>
      <div className="grid gap-4 sm:grid-cols-3">
        {NODES.map(({ node, port }) => {
          const info = nodes.find((n) => n.node === node);
          return (
            <Card key={node}>
              <CardHeader className="pb-2">
                <div className="flex items-center justify-between">
                  <CardTitle className="text-base font-semibold capitalize">{node}</CardTitle>
                </div>
                <p className="text-xs text-muted-foreground">localhost:{port}</p>
              </CardHeader>
              <CardContent className="text-sm space-y-1">
                {!info ? (
                  <div className="flex items-center gap-2 text-muted-foreground"><Loader2 className="h-3 w-3 animate-spin" /> loading…</div>
                ) : (
                  <>
                    <div className="flex justify-between"><span className="text-muted-foreground">Peers</span><span className="font-medium">{info.peers}</span></div>
                    <div className="flex justify-between"><span className="text-muted-foreground">Block</span><span className="font-medium font-mono">{info.blockNumber ?? "—"}</span></div>
                    {info.error && <p className="text-xs text-red-500 break-all">{info.error}</p>}
                  </>
                )}
              </CardContent>
            </Card>
          );
        })}
      </div>
      <Card>
        <CardHeader>
          <div className="flex items-center justify-between gap-2">
            <CardTitle className="text-base font-semibold">Developer Connection</CardTitle>
            <Badge variant="secondary">MetaMask Ready</Badge>
          </div>
          <p className="text-sm text-muted-foreground">
            {isWalletConnectedNow
              ? "Your wallet is connected. Use these values to configure MetaMask or other Titan tooling."
              : "Use these values to add Titan Local UAT to MetaMask. Connect your wallet from the sidebar."}
          </p>
        </CardHeader>
        <CardContent className="space-y-4">
          <div className="grid gap-3 sm:grid-cols-2">
            <div className="rounded-md border p-3 text-sm">
              <p className="text-xs uppercase text-muted-foreground">Dashboard URL</p>
              <div className="mt-1 flex items-center justify-between gap-2">
                <p className="font-mono break-all">{APP_CONFIG.titan.dashboardUrl}</p>
                <Button size="icon" variant="ghost" onClick={() => copyValue("dashboard", APP_CONFIG.titan.dashboardUrl)}>
                  <Copy className="h-4 w-4" />
                </Button>
              </div>
            </div>
            <div className="rounded-md border p-3 text-sm">
              <p className="text-xs uppercase text-muted-foreground">RPC URL</p>
              <div className="mt-1 flex items-center justify-between gap-2">
                <p className="font-mono break-all">{APP_CONFIG.titan.rpcUrl}</p>
                <Button size="icon" variant="ghost" onClick={() => copyValue("rpc", APP_CONFIG.titan.rpcUrl)}>
                  <Copy className="h-4 w-4" />
                </Button>
              </div>
            </div>
            <div className="rounded-md border p-3 text-sm">
              <p className="text-xs uppercase text-muted-foreground">Chain / Network</p>
              <p className="mt-1 font-mono">
                {APP_CONFIG.titan.chainIdDec} ({APP_CONFIG.titan.chainIdHex}) / {APP_CONFIG.titan.networkId}
              </p>
            </div>
            <div className="rounded-md border p-3 text-sm">
              <p className="text-xs uppercase text-muted-foreground">Explorer URL</p>
              <div className="mt-1 flex items-center justify-between gap-2">
                <p className="font-mono break-all">{APP_CONFIG.titan.explorerUrl}</p>
                <Button size="icon" variant="ghost" onClick={() => copyValue("explorer", APP_CONFIG.titan.explorerUrl)}>
                  <Copy className="h-4 w-4" />
                </Button>
              </div>
            </div>
            <div className="rounded-md border p-3 text-sm sm:col-span-2">
              <p className="text-xs uppercase text-muted-foreground">Native Token</p>
              <p className="mt-1 font-mono">
                {APP_CONFIG.titan.nativeToken.name} ({APP_CONFIG.titan.nativeToken.symbol}) · {APP_CONFIG.titan.nativeToken.decimals} decimals
              </p>
            </div>
          </div>

          {copiedField && <p className="text-xs text-muted-foreground">Copied {copiedField} to clipboard.</p>}
          {isWalletConnectedNow && (
            <div className="rounded-md border bg-muted/30 p-3 text-sm space-y-1.5">
              <p className="flex items-center gap-1.5 text-emerald-600 font-medium">
                <Wallet className="h-4 w-4" />
                Wallet connected
              </p>
              <p>
                <span className="text-muted-foreground">Address:</span>{" "}
                <span className="font-mono">{shortAddress(address)}</span>
              </p>
              <p>
                <span className="text-muted-foreground">Balance:</span>{" "}
                <span className="font-mono">
                  {titanBalance} {APP_CONFIG.titan.nativeToken.symbol}
                </span>
              </p>
              <p>
                <span className="text-muted-foreground">Network:</span>{" "}
                {isOnTitanChain(walletChainId) ? APP_CONFIG.titan.networkName : `Chain ${walletChainId}`}
              </p>
              {signMessage && (
                <p className="text-emerald-600 flex items-center gap-1 pt-1">
                  <ShieldCheck className="h-4 w-4" /> {signMessage}
                </p>
              )}
            </div>
          )}
          {walletError && !isWalletConnectedNow && <p className="text-sm text-red-500 break-all">{walletError}</p>}
        </CardContent>
      </Card>
      {lastUpdated && <p className="text-xs text-muted-foreground flex items-center gap-1"><Activity className="h-3 w-3" /> Last updated {lastUpdated.toLocaleTimeString()} · auto-refreshes every 10 s</p>}
    </div>
  );
}
