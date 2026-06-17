"use client";

import { useEffect, useRef, useState } from "react";

import { Loader2, Trash2 } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";

const CONTAINERS = ["all", "titan-node1", "titan-node2", "titan-node3"];

export default function LogsPage() {
  const [selected, setSelected] = useState("all");
  const [lines, setLines] = useState<string[]>([]);
  const [streaming, setStreaming] = useState(false);
  const topRef = useRef<HTMLDivElement>(null);
  const esRef = useRef<EventSource | null>(null);

  function connect(container: string) {
    if (esRef.current) {
      esRef.current.close();
      esRef.current = null;
    }
    setLines([]);
    setStreaming(true);
    const es = new EventSource(`/api/titan/logs?container=${container}`);
    esRef.current = es;
    es.onmessage = (e) => {
      // Newest first (top of list).
      setLines((prev) => [e.data, ...prev].slice(0, 3000));
    };
    es.onerror = () => {
      setStreaming(false);
      es.close();
    };
  }

  useEffect(() => {
    connect(selected);
    return () => {
      esRef.current?.close();
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [selected]);

  useEffect(() => {
    topRef.current?.scrollIntoView({ behavior: "smooth", block: "start" });
  }, [lines]);

  return (
    <div className="flex flex-col gap-4" style={{ height: "calc(100vh - 8rem)" }}>
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Docker Logs</h1>
          <p className="text-sm text-muted-foreground">
            Live streaming from containers · newest at top
          </p>
        </div>
        <div className="flex items-center gap-2">
          {streaming ? (
            <Badge className="bg-green-500 text-white flex gap-1 items-center">
              <Loader2 className="h-3 w-3 animate-spin" />
              Live
            </Badge>
          ) : (
            <Badge variant="secondary">Disconnected</Badge>
          )}
          <Button
            variant="ghost"
            size="icon"
            onClick={() => setLines([])}
            title="Clear logs"
          >
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      </div>

      <div className="flex gap-2">
        {CONTAINERS.map((c) => (
          <Button
            key={c}
            variant={selected === c ? "default" : "outline"}
            size="sm"
            onClick={() => setSelected(c)}
          >
            {c === "all" ? "combined" : c}
          </Button>
        ))}
      </div>

      <Card className="flex-1 flex flex-col overflow-hidden min-h-0">
        <CardHeader className="pb-2 shrink-0">
          <CardTitle className="text-sm font-mono text-muted-foreground">
            {selected === "all" ? "combined stream: all titan containers" : selected}
          </CardTitle>
        </CardHeader>
        <CardContent className="flex-1 overflow-y-auto bg-black rounded-b-lg p-3 min-h-0">
          <div ref={topRef} />
          <pre className="text-xs font-mono text-green-400 whitespace-pre-wrap break-all">
            {lines.length === 0
              ? "Waiting for log output…"
              : lines.join("\n")}
          </pre>
        </CardContent>
      </Card>
    </div>
  );
}
