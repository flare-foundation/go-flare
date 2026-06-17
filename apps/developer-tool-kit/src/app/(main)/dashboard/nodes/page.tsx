"use client";

import { useEffect, useState } from "react";

import { Loader2, RefreshCw, Server, Users } from "lucide-react";

import { Badge } from "@/components/ui/badge";
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
  error?: string;
}

const NODES = [
  { node: "node1", port: 9650 },
  { node: "node2", port: 9652 },
  { node: "node3", port: 9654 },
];

export default function NodesPage() {
  const [nodes, setNodes] = useState<NodeInfo[]>([]);
  const [loading, setLoading] = useState(true);

  async function load() {
    setLoading(true);
    try {
      const r = await fetch("/api/titan/rpc");
      const j = await r.json();
      setNodes(j.nodes ?? []);
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
          <h1 className="text-2xl font-bold tracking-tight">Nodes</h1>
          <p className="text-sm text-muted-foreground">
            Live status for all Titan network nodes
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
        {NODES.map(({ node, port }) => {
          const info = nodes.find((n) => n.node === node);
          return (
            <Card key={node}>
              <CardHeader className="pb-2">
                <div className="flex items-center gap-3">
                  <Server className="h-5 w-5 text-muted-foreground" />
                  <div className="flex-1">
                    <CardTitle className="text-base capitalize">{node}</CardTitle>
                    <p className="text-xs text-muted-foreground font-mono">
                      localhost:{port}
                    </p>
                  </div>
                  {info === undefined ? (
                    <Badge variant="secondary">…</Badge>
                  ) : info.healthy ? (
                    <Badge className="bg-green-500 text-white">Healthy</Badge>
                  ) : (
                    <Badge variant="destructive">Down</Badge>
                  )}
                </div>
              </CardHeader>
              <CardContent className="space-y-2 text-sm">
                {info === undefined ? (
                  <div className="text-muted-foreground flex gap-2">
                    <Loader2 className="h-3 w-3 animate-spin mt-0.5" />
                    Loading…
                  </div>
                ) : (
                  <>
                    <Row
                      label="Peers"
                      value={String(info.peers)}
                      icon={<Users className="h-3 w-3" />}
                    />
                    <Row label="Chain ID" value={info.chainId ?? "—"} mono />
                    <Row label="Latest Block" value={info.blockNumber ?? "—"} mono />
                    <Row label="Gas Price" value={info.gasPrice ?? "—"} mono />
                    <Row
                      label="RPC"
                      value={`http://localhost:${port}/ext/bc/C/rpc`}
                      mono
                      small
                    />
                    {info.error && (
                      <p className="text-xs text-red-500 break-all">{info.error}</p>
                    )}
                  </>
                )}
              </CardContent>
            </Card>
          );
        })}
      </div>
    </div>
  );
}

function Row({
  label,
  value,
  mono,
  small,
  icon,
}: {
  label: string;
  value: string;
  mono?: boolean;
  small?: boolean;
  icon?: React.ReactNode;
}) {
  return (
    <div className="flex justify-between items-start gap-2">
      <span className="text-muted-foreground shrink-0 flex items-center gap-1">
        {icon}
        {label}
      </span>
      <span
        className={`${mono ? "font-mono" : "font-medium"} ${small ? "text-xs" : ""} break-all text-right`}
      >
        {value}
      </span>
    </div>
  );
}
