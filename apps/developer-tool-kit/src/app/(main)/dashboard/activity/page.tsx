"use client";

import { useEffect, useState } from "react";

import { Activity, Loader2, RefreshCw } from "lucide-react";

import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

interface NodeInfo {
  node: string;
  port: number;
  healthy: boolean;
  peers: number;
  chainId?: string;
  blockNumber?: string;
  gasPrice?: string;
}

export default function ActivityPage() {
  const [nodes, setNodes] = useState<NodeInfo[]>([]);
  const [loading, setLoading] = useState(true);
  const [ts, setTs] = useState<Date | null>(null);

  async function load() {
    setLoading(true);
    try {
      const r = await fetch("/api/titan/rpc");
      const j = await r.json();
      setNodes(j.nodes ?? []);
      setTs(new Date());
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
    const id = setInterval(load, 10_000);
    return () => clearInterval(id);
  }, []);

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Activity</h1>
          <p className="text-sm text-muted-foreground">
            Real-time chain metrics across all nodes
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={load} disabled={loading}>
          {loading ? (
            <Loader2 className="h-4 w-4 animate-spin" />
          ) : (
            <RefreshCw className="h-4 w-4" />
          )}
          Refresh
        </Button>
      </div>

      <div className="grid gap-4 sm:grid-cols-3">
        {nodes.map((n) => (
          <Card key={n.node}>
            <CardHeader className="pb-2">
              <div className="flex items-center gap-2">
                <Activity className="h-4 w-4 text-muted-foreground" />
                <CardTitle className="text-base capitalize">{n.node}</CardTitle>
              </div>
            </CardHeader>
            <CardContent className="text-sm space-y-2">
              <Row label="Block" value={n.blockNumber ?? "—"} />
              <Row label="Gas Price" value={n.gasPrice ?? "—"} />
              <Row label="Peers" value={String(n.peers)} />
              <Row label="Chain ID" value={n.chainId ?? "—"} />
            </CardContent>
          </Card>
        ))}
      </div>

      {ts && (
        <p className="text-xs text-muted-foreground">
          Last updated: {ts.toLocaleTimeString()} · auto-refreshes every 10 s
        </p>
      )}
    </div>
  );
}

function Row({ label, value }: { label: string; value: string }) {
  return (
    <div className="flex justify-between">
      <span className="text-muted-foreground">{label}</span>
      <span className="font-medium font-mono">{value}</span>
    </div>
  );
}
