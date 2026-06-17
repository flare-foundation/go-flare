"use client";

import { useEffect, useMemo, useRef, useState } from "react";

import { Loader2, Pause, Play, ScrollText, Trash2 } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import {
  containerBadgeClass,
  formatLogTime,
  levelTextClass,
  parseLogLine,
  type ParsedLogLine,
  shortContainerLabel,
  type TitanLogContainer,
} from "@/lib/titan/log-parser";
import { cn } from "@/lib/utils";

const CONTAINERS = ["all", "titan-node1", "titan-node2", "titan-node3"] as const;
type ContainerFilter = (typeof CONTAINERS)[number];

const MAX_LINES = 3000;

export function LogViewer() {
  const [selected, setSelected] = useState<ContainerFilter>("all");
  const [lines, setLines] = useState<string[]>([]);
  const [streaming, setStreaming] = useState(false);
  const [paused, setPaused] = useState(false);
  const [search, setSearch] = useState("");
  const listRef = useRef<HTMLDivElement>(null);
  const esRef = useRef<EventSource | null>(null);
  const pausedRef = useRef(paused);

  useEffect(() => {
    pausedRef.current = paused;
  }, [paused]);

  function connect(container: ContainerFilter) {
    if (esRef.current) {
      esRef.current.close();
      esRef.current = null;
    }
    setLines([]);
    setStreaming(true);
    const es = new EventSource(`/api/titan/logs?container=${container}`);
    esRef.current = es;
    es.onmessage = (event) => {
      if (pausedRef.current) return;
      setLines((prev) => [event.data, ...prev].slice(0, MAX_LINES));
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
  }, [selected]);

  const parsedLines = useMemo(() => lines.map((line, index) => parseLogLine(line, index)), [lines]);

  const filteredLines = useMemo(() => {
    const query = search.trim().toLowerCase();
    return parsedLines.filter((entry) => {
      if (selected !== "all" && entry.container !== selected) return false;
      if (!query) return true;
      return (
        entry.message.toLowerCase().includes(query) ||
        entry.container.toLowerCase().includes(query) ||
        (entry.timestamp?.toLowerCase().includes(query) ?? false)
      );
    });
  }, [parsedLines, search, selected]);

  const counts = useMemo(() => {
    const tally: Record<TitanLogContainer, number> = {
      "titan-node1": 0,
      "titan-node2": 0,
      "titan-node3": 0,
      unknown: 0,
    };
    for (const entry of parsedLines) {
      tally[entry.container] += 1;
    }
    return tally;
  }, [parsedLines]);

  return (
    <div className="flex flex-col gap-5" style={{ height: "calc(100vh - 8rem)" }}>
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight flex items-center gap-2">
            <ScrollText className="h-6 w-6" />
            Docker Logs
          </h1>
          <p className="text-sm text-muted-foreground">
            Live Titan node output · parsed by timestamp and container
          </p>
        </div>
        <div className="flex items-center gap-2">
          {streaming ? (
            <Badge variant="secondary" className="gap-1.5">
              <span className="size-1.5 rounded-full bg-emerald-500 animate-pulse" />
              Live
            </Badge>
          ) : (
            <Badge variant="outline">Disconnected</Badge>
          )}
          <Button
            variant="outline"
            size="sm"
            onClick={() => setPaused((value) => !value)}
            title={paused ? "Resume stream" : "Pause stream"}
          >
            {paused ? <Play className="h-4 w-4" /> : <Pause className="h-4 w-4" />}
            {paused ? "Resume" : "Pause"}
          </Button>
          <Button variant="ghost" size="icon" onClick={() => setLines([])} title="Clear logs">
            <Trash2 className="h-4 w-4" />
          </Button>
        </div>
      </div>

      <div className="flex flex-wrap items-center gap-2 rounded-lg border bg-muted/20 px-3 py-2 text-xs">
        {(["titan-node1", "titan-node2", "titan-node3"] as const).map((node) => (
          <span key={node} className="inline-flex items-center gap-1.5">
            <span className={cn("rounded-md border px-1.5 py-0.5 font-mono", containerBadgeClass(node))}>
              {shortContainerLabel(node)}
            </span>
            <span className="text-muted-foreground tabular-nums">{counts[node]}</span>
          </span>
        ))}
        <span className="text-muted-foreground ml-auto">{filteredLines.length.toLocaleString()} lines shown</span>
      </div>

      <div className="flex flex-col gap-3 sm:flex-row sm:items-center">
        <div className="flex flex-wrap gap-2">
          {CONTAINERS.map((container) => (
            <Button
              key={container}
              variant={selected === container ? "default" : "outline"}
              size="sm"
              onClick={() => setSelected(container)}
            >
              {container === "all" ? "Combined" : shortContainerLabel(container)}
            </Button>
          ))}
        </div>
        <Input
          value={search}
          onChange={(e) => setSearch(e.target.value)}
          placeholder="Filter message, node, or timestamp…"
          className="sm:max-w-xs font-mono text-xs"
        />
      </div>

      <section className="flex min-h-0 flex-1 flex-col overflow-hidden rounded-lg border">
        <div className="hidden gap-3 border-b bg-muted/30 px-4 py-2 text-[10px] font-medium uppercase tracking-wider text-muted-foreground sm:grid sm:grid-cols-[5.5rem_4.5rem_1fr]">
          <span>Time</span>
          <span>Node</span>
          <span>Message</span>
        </div>

        <div ref={listRef} className="min-h-0 flex-1 overflow-y-auto">
          {filteredLines.length === 0 ? (
            <div className="flex items-center justify-center gap-2 px-4 py-16 text-sm text-muted-foreground">
              {streaming ? <Loader2 className="h-4 w-4 animate-spin" /> : null}
              {streaming ? "Waiting for log output…" : "No logs to show"}
            </div>
          ) : (
            <div className="divide-y divide-border/60">
              {filteredLines.map((entry) => (
                <LogRow key={entry.id} entry={entry} />
              ))}
            </div>
          )}
        </div>
      </section>
    </div>
  );
}

function LogRow({ entry }: { entry: ParsedLogLine }) {
  return (
    <div className="gap-2 px-4 py-2 text-xs hover:bg-muted/30 sm:grid sm:grid-cols-[5.5rem_4.5rem_1fr] sm:gap-3">
      <div className="mb-1 flex items-center gap-2 sm:mb-0 sm:block">
        <span
          className={cn(
            "inline-flex rounded-md border px-1.5 py-0.5 font-mono text-[10px] font-medium sm:hidden",
            containerBadgeClass(entry.container),
          )}
        >
          {shortContainerLabel(entry.container)}
        </span>
        <span className="font-mono tabular-nums text-muted-foreground">
          {formatLogTime(entry.timestamp, entry.timestampMs)}
        </span>
      </div>
      <span className="hidden sm:inline">
        <span
          className={cn(
            "inline-flex rounded-md border px-1.5 py-0.5 font-mono text-[10px] font-medium",
            containerBadgeClass(entry.container),
          )}
        >
          {shortContainerLabel(entry.container)}
        </span>
      </span>
      <span className={cn("font-mono break-all leading-relaxed", levelTextClass(entry.level))}>{entry.message}</span>
    </div>
  );
}