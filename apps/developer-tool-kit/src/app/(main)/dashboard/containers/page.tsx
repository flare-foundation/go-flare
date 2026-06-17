"use client";

import { useEffect, useState } from "react";

import { Container, Loader2, RefreshCw } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

interface ContainerInfo {
  name: string;
  status: string;
  ports: string;
  image: string;
}

export default function ContainersPage() {
  const [containers, setContainers] = useState<ContainerInfo[]>([]);
  const [loading, setLoading] = useState(true);

  async function load() {
    setLoading(true);
    try {
      const r = await fetch("/api/titan/logs?info=1");
      const j = await r.json();
      setContainers(j.containers ?? []);
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
    const id = setInterval(load, 15_000);
    return () => clearInterval(id);
  }, []);

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Containers</h1>
          <p className="text-sm text-muted-foreground">
            Docker container status (auto-refreshes every 15 s)
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

      {loading && containers.length === 0 ? (
        <div className="flex items-center gap-2 text-muted-foreground">
          <Loader2 className="h-4 w-4 animate-spin" />
          Loading containers…
        </div>
      ) : containers.length === 0 ? (
        <p className="text-muted-foreground">
          No titan containers found. Are the nodes running?
        </p>
      ) : (
        <div className="grid gap-4 sm:grid-cols-3">
          {containers.map((c) => {
            const isUp = c.status?.includes("Up");
            return (
              <Card key={c.name}>
                <CardHeader className="pb-2">
                  <div className="flex items-center gap-2">
                    <Container className="h-4 w-4 text-muted-foreground" />
                    <CardTitle className="text-base font-mono">{c.name}</CardTitle>
                  </div>
                </CardHeader>
                <CardContent className="text-sm space-y-2">
                  <div className="flex justify-between items-center">
                    <span className="text-muted-foreground">Status</span>
                    <Badge
                      className={
                        isUp ? "bg-green-500 text-white" : ""
                      }
                      variant={isUp ? "default" : "destructive"}
                    >
                      {c.status}
                    </Badge>
                  </div>
                  <div className="flex justify-between items-start gap-2">
                    <span className="text-muted-foreground shrink-0">Ports</span>
                    <span className="font-mono text-xs text-right break-all">
                      {c.ports || "—"}
                    </span>
                  </div>
                  <div className="flex justify-between items-start gap-2">
                    <span className="text-muted-foreground shrink-0">Image</span>
                    <span className="font-mono text-xs text-right break-all">
                      {c.image}
                    </span>
                  </div>
                </CardContent>
              </Card>
            );
          })}
        </div>
      )}
    </div>
  );
}
