"use client";

import { useCallback, useEffect, useState } from "react";
import {
  Activity,
  ArrowLeft,
  Clock,
  Copy,
  Hash,
  Loader2,
  RefreshCw,
  Search,
  Zap,
} from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";

interface NodeInfo {
  node: string;
  port: number;
  healthy: boolean;
  peers: number;
  chainId?: string;
  blockNumber?: string;
  gasPrice?: string;
}

interface Tx {
  blockHash?: string;
  blockNumber?: string;
  from: string;
  gas: string;
  gasPrice?: string;
  hash: string;
  input: string;
  nonce: string;
  to: string | null;
  transactionIndex?: string;
  value: string;
  type?: string;
  // extended when full
}

interface Block {
  number: string;
  hash: string;
  parentHash: string;
  timestamp: string;
  miner?: string; // coinbase
  gasUsed: string;
  gasLimit: string;
  baseFeePerGas?: string;
  difficulty?: string;
  extraData?: string;
  transactions: Tx[] | string[]; // full objects when requested with true
  transactionCount?: number;
}

interface Receipt {
  status: string;
  gasUsed: string;
  effectiveGasPrice?: string;
  logs: unknown[];
  blockNumber: string;
  transactionHash: string;
}

const NODES = [
  { node: "node1", port: 9650 },
  { node: "node2", port: 9652 },
  { node: "node3", port: 9654 },
];

// Small RPC helper via our proxy (targets C-Chain by default)
async function rpc(method: string, params: unknown[] = [], node = "node1"): Promise<unknown> {
  const res = await fetch("/api/titan/rpc", {
    method: "POST",
    headers: { "content-type": "application/json" },
    body: JSON.stringify({ method, params, node, chain: "C" }),
  });
  const j = await res.json();
  if (j?.error) {
    const msg = typeof j.error === "string" ? j.error : j.error?.message || JSON.stringify(j.error);
    throw new Error(msg);
  }
  return j?.result;
}

function shortHash(h?: string | null, left = 6, right = 4): string {
  if (!h) return "—";
  if (h.length <= left + right + 2) return h;
  return `${h.slice(0, left)}…${h.slice(-right)}`;
}

function formatWeiToTitan(hexOrBig?: string | bigint): string {
  if (hexOrBig == null) return "0";
  try {
    const wei = typeof hexOrBig === "string" ? BigInt(hexOrBig) : hexOrBig;
    const titan = Number(wei) / 1e18;
    if (Math.abs(titan) < 0.0001 && titan !== 0) return titan.toExponential(2);
    return titan.toFixed(4);
  } catch {
    return "0";
  }
}

function formatGwei(hex?: string): string {
  if (!hex) return "—";
  try {
    const wei = BigInt(hex);
    const g = Number(wei) / 1e9;
    return g.toFixed(g < 1 ? 4 : 2) + " gwei";
  } catch {
    return "—";
  }
}

function hexToNumber(hex?: string): number | null {
  if (!hex) return null;
  try {
    return parseInt(hex, 16);
  } catch {
    return null;
  }
}

function formatTimestamp(tsHex?: string): { full: string; ago: string } {
  if (!tsHex) return { full: "—", ago: "—" };
  const sec = hexToNumber(tsHex) ?? 0;
  const d = new Date(sec * 1000);
  const full = d.toLocaleString();
  const diff = Math.floor((Date.now() - d.getTime()) / 1000);
  let ago = "";
  if (diff < 60) ago = `${diff}s ago`;
  else if (diff < 3600) ago = `${Math.floor(diff / 60)}m ago`;
  else if (diff < 86400) ago = `${Math.floor(diff / 3600)}h ago`;
  else ago = `${Math.floor(diff / 86400)}d ago`;
  return { full, ago };
}

function isBlockHash(s: string) {
  return /^0x[0-9a-fA-F]{64}$/.test(s);
}
function isTxHash(s: string) {
  return /^0x[0-9a-fA-F]{64}$/.test(s);
}
function isAddress(s: string) {
  return /^0x[0-9a-fA-F]{40}$/.test(s);
}
function isBlockNumber(s: string) {
  return /^\d+$/.test(s) || /^0x[0-9a-fA-F]+$/.test(s);
}

export default function ExplorerPage() {
  const [nodes, setNodes] = useState<NodeInfo[]>([]);
  const [nodesLoading, setNodesLoading] = useState(true);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

  const [blocks, setBlocks] = useState<Block[]>([]);
  const [blocksLoading, setBlocksLoading] = useState(true);
  const [blocksError, setBlocksError] = useState<string>("");

  const [selectedBlock, setSelectedBlock] = useState<Block | null>(null);
  const [selectedBlockLoading, setSelectedBlockLoading] = useState(false);

  const [selectedTxHash, setSelectedTxHash] = useState<string | null>(null);
  const [selectedTx, setSelectedTx] = useState<Tx | null>(null);
  const [selectedReceipt, setSelectedReceipt] = useState<Receipt | null>(null);
  const [txLoading, setTxLoading] = useState(false);

  const [searchValue, setSearchValue] = useState("");
  const [searchLoading, setSearchLoading] = useState(false);
  const [searchError, setSearchError] = useState("");

  // Load node overview (reused from old activity)
  async function loadNodes() {
    setNodesLoading(true);
    try {
      const r = await fetch("/api/titan/rpc");
      const j = await r.json();
      setNodes(j.nodes ?? []);
      setLastUpdated(new Date());
    } catch {
      setNodes([]);
    } finally {
      setNodesLoading(false);
    }
  }

  // Fetch a single block (with full tx objects)
  const fetchBlock = useCallback(async (identifier: string | number, isHash = false): Promise<Block | null> => {
    try {
      const param = isHash
        ? identifier
        : typeof identifier === "number"
          ? "0x" + identifier.toString(16)
          : identifier; // can be "latest" or hex
      const raw = await rpc("eth_getBlockByNumber", [param, true]);
      if (!raw) return null;
      const b = raw as Block;
      // normalize tx count
      const txArr = Array.isArray(b.transactions) ? b.transactions : [];
      return {
        ...b,
        transactionCount: txArr.length,
      } as Block;
    } catch (e: unknown) {
      console.error("fetchBlock error", e);
      return null;
    }
  }, []);

  // Load recent N blocks starting from head
  const loadRecentBlocks = useCallback(async (count = 20) => {
    setBlocksLoading(true);
    setBlocksError("");
    try {
      const headHex = (await rpc("eth_blockNumber")) as string | null;
      if (!headHex) throw new Error("Could not read head block");
      const head = parseInt(headHex, 16);

      const toFetch: number[] = [];
      for (let i = 0; i < count; i++) {
        const n = head - i;
        if (n < 0) break;
        toFetch.push(n);
      }

      const results = await Promise.all(toFetch.map((n) => fetchBlock(n)));
      const valid = results.filter(Boolean) as Block[];
      setBlocks(valid);
      // auto select latest if nothing selected
      if (valid.length > 0 && !selectedBlock) {
        setSelectedBlock(valid[0]);
        // preload its first tx if any? (optional)
      }
    } catch (e: unknown) {
      setBlocksError(e instanceof Error ? e.message : "Failed to load blocks");
      setBlocks([]);
    } finally {
      setBlocksLoading(false);
    }
  }, [fetchBlock, selectedBlock]);

  // Load older blocks (before the current lowest)
  async function loadOlder(count = 10) {
    if (blocks.length === 0) {
      await loadRecentBlocks();
      return;
    }
    const lowestHex = blocks[blocks.length - 1].number;
    const lowest = hexToNumber(lowestHex) ?? 0;
    if (lowest <= 0) return;

    setBlocksLoading(true);
    try {
      const toFetch: number[] = [];
      for (let i = 1; i <= count; i++) {
        const n = lowest - i;
        if (n < 0) break;
        toFetch.push(n);
      }
      const results = await Promise.all(toFetch.map((n) => fetchBlock(n)));
      const older = results.filter(Boolean) as Block[];
      if (older.length) {
        setBlocks((prev) => [...prev, ...older]);
      }
    } catch (e) {
      // silent for load older
    } finally {
      setBlocksLoading(false);
    }
  }

  // Select and fully load a block (ensures full tx objects)
  async function selectBlock(b: Block | null) {
    if (!b) {
      setSelectedBlock(null);
      setSelectedTxHash(null);
      setSelectedTx(null);
      setSelectedReceipt(null);
      return;
    }
    // if we already have full txs, use it
    const needsRefetch = !b.transactions || (b.transactions.length > 0 && typeof b.transactions[0] === "string");
    if (needsRefetch) {
      setSelectedBlockLoading(true);
      const full = await fetchBlock(b.hash, true);
      setSelectedBlock(full ?? b);
      setSelectedBlockLoading(false);
    } else {
      setSelectedBlock(b);
    }
    // clear tx selection
    setSelectedTxHash(null);
    setSelectedTx(null);
    setSelectedReceipt(null);
  }

  // Fetch a transaction + receipt
  async function loadTx(hash: string) {
    setTxLoading(true);
    setSelectedTxHash(hash);
    setSelectedTx(null);
    setSelectedReceipt(null);
    setSearchError("");
    try {
      const [txRaw, receiptRaw] = await Promise.all([
        rpc("eth_getTransactionByHash", [hash]),
        rpc("eth_getTransactionReceipt", [hash]),
      ]);
      setSelectedTx(txRaw as Tx | null);
      setSelectedReceipt(receiptRaw as Receipt | null);

      // If we can, also select/ensure the parent block is loaded in the list/details
      if (txRaw && (txRaw as Tx).blockNumber) {
        const bn = (txRaw as Tx).blockNumber as string;
        // try to find in current blocks
        const existing = blocks.find((bb) => bb.number === bn);
        if (existing) {
          setSelectedBlock(existing);
        } else {
          // fetch it
          const bl = await fetchBlock(bn);
          if (bl) {
            setBlocks((prev) => {
              // insert sorted if possible, else just put at front
              const exists = prev.some((p) => p.number === bl.number);
              return exists ? prev : [bl, ...prev].sort((a, b) => hexToNumber(b.number)! - hexToNumber(a.number)!);
            });
            setSelectedBlock(bl);
          }
        }
      }
    } catch (e: unknown) {
      setSearchError(e instanceof Error ? e.message : "Failed to load transaction");
      setSelectedTxHash(null);
    } finally {
      setTxLoading(false);
    }
  }

  // Generic search dispatcher
  async function performSearch(raw: string) {
    const q = raw.trim();
    if (!q) return;
    setSearchLoading(true);
    setSearchError("");

    try {
      if (isBlockNumber(q)) {
        const num = q.startsWith("0x") ? parseInt(q, 16) : parseInt(q, 10);
        const b = await fetchBlock(num);
        if (b) {
          // put it at top of list if new
          setBlocks((prev) => {
            const exists = prev.some((bb) => bb.hash === b.hash);
            const next = exists ? prev : [b, ...prev];
            return next.sort((a, bb) => (hexToNumber(bb.number) ?? 0) - (hexToNumber(a.number) ?? 0));
          });
          await selectBlock(b);
          setSearchValue("");
        } else {
          setSearchError("Block not found");
        }
      } else if (isTxHash(q)) {
        await loadTx(q);
        setSearchValue("");
      } else if (isBlockHash(q)) {
        const b = await fetchBlock(q, true);
        if (b) {
          setBlocks((prev) => {
            const exists = prev.some((bb) => bb.hash === b.hash);
            const next = exists ? prev : [b, ...prev];
            return next.sort((a, bb) => (hexToNumber(bb.number) ?? 0) - (hexToNumber(a.number) ?? 0));
          });
          await selectBlock(b);
          setSearchValue("");
        } else {
          setSearchError("Block not found");
        }
      } else if (isAddress(q)) {
        // Simple address lookup: show balance
        const balHex = (await rpc("eth_getBalance", [q, "latest"])) as string;
        const titan = formatWeiToTitan(balHex);
        // For now surface in search error area (or we could open a mini panel)
        setSearchError(`Address balance: ${titan} TITAN  (${balHex})`);
        // optionally clear after a bit, but leave it visible
        setTimeout(() => setSearchError((prev) => (prev.includes("Address balance") ? "" : prev)), 8000);
      } else {
        setSearchError("Unrecognized input. Use block number, 0x-block/tx hash, or 0x-address.");
      }
    } catch (e: unknown) {
      setSearchError(e instanceof Error ? e.message : "Search failed");
    } finally {
      setSearchLoading(false);
    }
  }

  function onSearchKey(e: React.KeyboardEvent<HTMLInputElement>) {
    if (e.key === "Enter") {
      performSearch(searchValue);
    }
  }

  // Auto-refresh nodes + detect new head and refresh blocks
  useEffect(() => {
    loadNodes();
    const id = setInterval(loadNodes, 10000);
    return () => clearInterval(id);
  }, []);

  // Initial blocks + periodic head check
  useEffect(() => {
    loadRecentBlocks(18);

    const headPoll = setInterval(async () => {
      try {
        const headHex = (await rpc("eth_blockNumber")) as string;
        const head = hexToNumber(headHex);
        if (head == null || blocks.length === 0) return;

        const currentHead = hexToNumber(blocks[0]?.number);
        if (head > (currentHead ?? -1)) {
          // new blocks! reload recent
          await loadRecentBlocks(18);
        }
      } catch {
        /* ignore poll errors */
      }
    }, 8000);

    return () => clearInterval(headPoll);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // When nodes update, if we have no blocks yet, ensure load
  useEffect(() => {
    if (blocks.length === 0 && !blocksLoading) {
      // already handled by initial
    }
  }, [blocks.length, blocksLoading]);

  const headBlock = blocks[0]?.number ? hexToNumber(blocks[0].number)?.toLocaleString() : nodes.find((n) => n.blockNumber)?.blockNumber;
  const chainId = nodes.find((n) => n.chainId)?.chainId ?? "—";
  const gasPrice = nodes.find((n) => n.gasPrice)?.gasPrice ?? "—";

  const currentDetailBlock = selectedBlock;

  return (
    <div className="flex flex-col gap-6">
      {/* Header */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight flex items-center gap-2">
            <Hash className="h-6 w-6" /> Explorer
          </h1>
          <p className="text-sm text-muted-foreground">
            Browse the Titan C-Chain • live blocks &amp; transactions
          </p>
        </div>

        <div className="flex items-center gap-2">
          <Button
            variant="outline"
            size="sm"
            onClick={() => {
              loadNodes();
              loadRecentBlocks(18);
              setSelectedBlock(null);
              setSelectedTx(null);
              setSelectedTxHash(null);
              setSelectedReceipt(null);
            }}
            disabled={blocksLoading || nodesLoading}
          >
            {blocksLoading || nodesLoading ? (
              <Loader2 className="h-4 w-4 animate-spin" />
            ) : (
              <RefreshCw className="h-4 w-4" />
            )}
            Refresh
          </Button>
          <Button
            size="sm"
            variant="secondary"
            onClick={() => loadRecentBlocks(18)}
            disabled={blocksLoading}
          >
            Latest Blocks
          </Button>
        </div>
      </div>

      {/* Live head summary */}
      <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
        <Card>
          <CardHeader className="pb-1.5">
            <CardTitle className="text-xs font-medium text-muted-foreground flex items-center gap-1.5">
              <Zap className="h-3.5 w-3.5" /> HEAD BLOCK
            </CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-semibold font-mono tabular-nums">{headBlock ?? "—"}</div>
            <p className="text-[11px] text-muted-foreground mt-0.5">Latest confirmed on C-Chain</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-1.5">
            <CardTitle className="text-xs font-medium text-muted-foreground">CHAIN ID</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-semibold font-mono">{chainId}</div>
            <p className="text-[11px] text-muted-foreground mt-0.5">781337 (local UAT)</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-1.5">
            <CardTitle className="text-xs font-medium text-muted-foreground">GAS PRICE (SAMPLE)</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="text-2xl font-semibold font-mono">{gasPrice}</div>
            <p className="text-[11px] text-muted-foreground mt-0.5">Across healthy nodes</p>
          </CardContent>
        </Card>
        <Card>
          <CardHeader className="pb-1.5">
            <CardTitle className="text-xs font-medium text-muted-foreground">NODES</CardTitle>
          </CardHeader>
          <CardContent className="flex flex-wrap gap-1.5">
            {NODES.map(({ node, port }) => {
              const info = nodes.find((n) => n.node === node);
              const ok = info?.healthy;
              return (
                <Badge key={node} variant={ok ? "default" : "secondary"} className={ok ? "bg-green-600" : ""}>
                  {node}:{port} {ok ? "●" : "○"}
                </Badge>
              );
            })}
            {nodesLoading && <Loader2 className="h-3 w-3 animate-spin mt-1" />}
          </CardContent>
        </Card>
      </div>

      {/* Search */}
      <Card>
        <CardContent className="pt-4">
          <div className="flex flex-col gap-2 sm:flex-row">
            <div className="relative flex-1">
              <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
              <Input
                value={searchValue}
                onChange={(e) => setSearchValue(e.target.value)}
                onKeyDown={onSearchKey}
                placeholder="Search block number (e.g. 3120), block/tx hash (0x...), or address"
                className="pl-9 font-mono text-sm"
                disabled={searchLoading}
              />
            </div>
            <Button onClick={() => performSearch(searchValue)} disabled={searchLoading || !searchValue.trim()}>
              {searchLoading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Search className="h-4 w-4" />}
              Search
            </Button>
          </div>
          {searchError && (
            <p className="mt-2 text-xs text-amber-600 break-all">{searchError}</p>
          )}
          <p className="mt-1 text-[10px] text-muted-foreground">
            Tip: paste a tx hash to jump straight to it. Numbers are treated as block heights.
          </p>
        </CardContent>
      </Card>

      {/* Main content: blocks list + detail */}
      <div className="grid gap-6 lg:grid-cols-5">
        {/* Recent blocks */}
        <div className="lg:col-span-2">
          <div className="mb-2 flex items-center justify-between">
            <div className="text-sm font-semibold tracking-tight flex items-center gap-2">
              <Activity className="h-4 w-4" /> Recent Blocks
            </div>
            <div className="text-xs text-muted-foreground">
              {blocks.length > 0 ? `${hexToNumber(blocks[blocks.length - 1].number)} — ${hexToNumber(blocks[0].number)}` : ""}
            </div>
          </div>

          <Card className="overflow-hidden">
            <div className="max-h-140 overflow-auto">
              <table className="w-full text-sm">
                <thead className="sticky top-0 bg-card text-muted-foreground text-xs uppercase tracking-wider border-b">
                  <tr>
                    <th className="px-3 py-2 text-left font-medium">Block</th>
                    <th className="px-3 py-2 text-left font-medium">Age</th>
                    <th className="px-3 py-2 text-right font-medium">Txs</th>
                    <th className="px-3 py-2 text-right font-medium hidden sm:table-cell">Gas Used</th>
                    <th className="px-3 py-2 text-left font-medium hidden md:table-cell">Hash</th>
                  </tr>
                </thead>
                <tbody className="divide-y">
                  {blocksLoading && blocks.length === 0 ? (
                    <tr>
                      <td colSpan={5} className="px-3 py-8 text-center text-muted-foreground">
                        <Loader2 className="mx-auto h-4 w-4 animate-spin" />
                      </td>
                    </tr>
                  ) : blocks.length === 0 ? (
                    <tr>
                      <td colSpan={5} className="px-3 py-6 text-center text-muted-foreground">
                        {blocksError || "No blocks loaded"}
                      </td>
                    </tr>
                  ) : (
                    blocks.map((b) => {
                      const num = hexToNumber(b.number);
                      const { ago } = formatTimestamp(b.timestamp);
                      const isSel = selectedBlock?.hash === b.hash;
                      const txCount = b.transactionCount ?? (Array.isArray(b.transactions) ? b.transactions.length : 0);
                      return (
                        <tr
                          key={b.hash}
                          onClick={() => selectBlock(b)}
                          className={`cursor-pointer hover:bg-muted/60 transition-colors ${isSel ? "bg-muted/70" : ""}`}
                        >
                          <td className="px-3 py-2 font-mono font-semibold tabular-nums">#{num?.toLocaleString()}</td>
                          <td className="px-3 py-2 text-muted-foreground text-xs">{ago}</td>
                          <td className="px-3 py-2 text-right font-mono">{txCount}</td>
                          <td className="px-3 py-2 text-right font-mono text-xs text-muted-foreground hidden sm:table-cell">
                            {hexToNumber(b.gasUsed)?.toLocaleString() ?? "—"}
                          </td>
                          <td className="px-3 py-2 font-mono text-xs text-muted-foreground hidden md:table-cell">
                            {shortHash(b.hash)}
                          </td>
                        </tr>
                      );
                    })
                  )}
                </tbody>
              </table>
            </div>

            <div className="border-t p-2 flex items-center justify-between bg-muted/30">
              <Button variant="ghost" size="sm" onClick={() => loadOlder()} disabled={blocksLoading || blocks.length === 0}>
                Load older
              </Button>
              <div className="text-[10px] text-muted-foreground pr-1">
                {blocksLoading ? "Loading…" : "Click row to inspect"}
              </div>
            </div>
          </Card>

          {lastUpdated && (
            <p className="mt-1.5 text-[10px] text-muted-foreground px-1">
              Nodes updated {lastUpdated.toLocaleTimeString()} · blocks auto-refresh on new head
            </p>
          )}
        </div>

        {/* Details panel */}
        <div className="lg:col-span-3 space-y-4">
          {/* Block detail */}
          <Card>
            <CardHeader className="pb-3 flex flex-row items-start justify-between">
              <div>
                <CardTitle className="text-base flex items-center gap-2">
                  Block {currentDetailBlock ? `#${hexToNumber(currentDetailBlock.number)?.toLocaleString()}` : "—"}
                </CardTitle>
                {currentDetailBlock && (
                  <p className="text-xs text-muted-foreground font-mono mt-0.5">{currentDetailBlock.hash}</p>
                )}
              </div>
              {currentDetailBlock && (
                <Button size="icon" variant="ghost" onClick={() => selectBlock(null)} title="Clear selection">
                  <ArrowLeft className="h-4 w-4" />
                </Button>
              )}
            </CardHeader>

            <CardContent>
              {!currentDetailBlock ? (
                <div className="text-sm text-muted-foreground py-4">Select a block from the list or search for one.</div>
              ) : selectedBlockLoading ? (
                <div className="flex items-center gap-2 text-sm text-muted-foreground py-6">
                  <Loader2 className="h-4 w-4 animate-spin" /> Loading block…
                </div>
              ) : (
                <div className="space-y-4 text-sm">
                  <div className="grid grid-cols-1 sm:grid-cols-2 gap-x-6 gap-y-1.5">
                    <DetailRow label="Timestamp" value={formatTimestamp(currentDetailBlock.timestamp).full} />
                    <DetailRow label="Age" value={formatTimestamp(currentDetailBlock.timestamp).ago} />
                    <DetailRow label="Transactions" value={String(currentDetailBlock.transactionCount ?? (Array.isArray(currentDetailBlock.transactions) ? currentDetailBlock.transactions.length : 0))} />
                    <DetailRow label="Gas Used / Limit" value={`${hexToNumber(currentDetailBlock.gasUsed)?.toLocaleString() ?? "—"} / ${hexToNumber(currentDetailBlock.gasLimit)?.toLocaleString() ?? "—"}`} />
                    {currentDetailBlock.baseFeePerGas && (
                      <DetailRow label="Base Fee" value={formatGwei(currentDetailBlock.baseFeePerGas)} />
                    )}
                    <DetailRow label="Parent Hash" value={shortHash(currentDetailBlock.parentHash)} mono copyValue={currentDetailBlock.parentHash} />
                  </div>

                  <div>
                    <div className="font-medium mb-1.5 flex items-center gap-1 text-xs uppercase tracking-widest text-muted-foreground">
                      Transactions in block ({currentDetailBlock.transactionCount ?? (Array.isArray(currentDetailBlock.transactions) ? currentDetailBlock.transactions.length : 0)})
                    </div>

                    {Array.isArray(currentDetailBlock.transactions) && currentDetailBlock.transactions.length > 0 ? (
                      <div className="border rounded-md overflow-hidden">
                        <table className="w-full text-xs">
                          <thead>
                            <tr className="bg-muted/60 text-muted-foreground">
                              <th className="px-2 py-1 text-left font-normal">#</th>
                              <th className="px-2 py-1 text-left font-normal">Hash</th>
                              <th className="px-2 py-1 text-left font-normal hidden md:table-cell">From → To</th>
                              <th className="px-2 py-1 text-right font-normal">Value</th>
                              <th className="px-2 py-1 text-right font-normal hidden sm:table-cell">Gas</th>
                            </tr>
                          </thead>
                          <tbody className="divide-y">
                            {currentDetailBlock.transactions.map((t, idx) => {
                              const tx = typeof t === "string" ? ({ hash: t } as Tx) : (t as Tx);
                              const val = formatWeiToTitan(tx.value);
                              const g = hexToNumber(tx.gas) ?? 0;
                              return (
                                <tr
                                  key={tx.hash}
                                  onClick={() => loadTx(tx.hash)}
                                  className={`cursor-pointer hover:bg-muted/50 ${selectedTxHash === tx.hash ? "bg-muted" : ""}`}
                                >
                                  <td className="px-2 py-1 font-mono tabular-nums text-muted-foreground">{idx}</td>
                                  <td className="px-2 py-1 font-mono text-primary hover:underline">{shortHash(tx.hash)}</td>
                                  <td className="px-2 py-1 text-muted-foreground hidden md:table-cell font-mono text-[10px] truncate max-w-65">
                                    {shortHash(tx.from, 4, 4)} → {tx.to ? shortHash(tx.to, 4, 4) : "contract"}
                                  </td>
                                  <td className="px-2 py-1 text-right font-medium tabular-nums">{val}</td>
                                  <td className="px-2 py-1 text-right text-muted-foreground hidden sm:table-cell">{g.toLocaleString()}</td>
                                </tr>
                              );
                            })}
                          </tbody>
                        </table>
                      </div>
                    ) : (
                      <div className="text-muted-foreground text-xs py-2">No transactions in this block.</div>
                    )}
                    <p className="text-[10px] text-muted-foreground mt-1">Click any transaction row to view full details + receipt.</p>
                  </div>
                </div>
              )}
            </CardContent>
          </Card>

          {/* Transaction detail */}
          {(selectedTxHash || selectedTx) && (
            <Card>
              <CardHeader className="pb-2">
                <div className="flex items-center justify-between">
                  <CardTitle className="text-base">Transaction</CardTitle>
                  {selectedTxHash && (
                    <Button size="sm" variant="ghost" onClick={() => { setSelectedTxHash(null); setSelectedTx(null); setSelectedReceipt(null); }}>
                      Close
                    </Button>
                  )}
                </div>
                <div className="font-mono text-xs text-muted-foreground break-all">{selectedTxHash}</div>
              </CardHeader>
              <CardContent>
                {txLoading ? (
                  <div className="flex gap-2 text-sm py-4 text-muted-foreground"><Loader2 className="h-4 w-4 animate-spin" /> Loading transaction…</div>
                ) : !selectedTx ? (
                  <div className="text-sm text-muted-foreground">Transaction not found or still loading.</div>
                ) : (
                  <div className="space-y-3 text-sm">
                    <div className="grid grid-cols-1 md:grid-cols-2 gap-x-8 gap-y-1">
                      <DetailRow label="From" value={shortHash(selectedTx.from)} mono copyValue={selectedTx.from} />
                      <DetailRow label="To" value={selectedTx.to ? shortHash(selectedTx.to) : "Contract creation"} mono copyValue={selectedTx.to ?? undefined} />
                      <DetailRow label="Value" value={`${formatWeiToTitan(selectedTx.value)} TITAN`} />
                      <DetailRow label="Gas Price" value={selectedTx.gasPrice ? formatGwei(selectedTx.gasPrice) : "—"} />
                      <DetailRow label="Gas Limit / Used" value={`${hexToNumber(selectedTx.gas)?.toLocaleString() ?? "—"} ${selectedReceipt ? ` / ${hexToNumber(selectedReceipt.gasUsed)?.toLocaleString()}` : ""}`} />
                      {selectedReceipt && (
                        <DetailRow
                          label="Status"
                          value={
                            <span className={selectedReceipt.status === "0x1" || selectedReceipt.status === "0x01" ? "text-green-600 font-medium" : "text-red-600 font-medium"}>
                              {selectedReceipt.status === "0x1" || selectedReceipt.status === "0x01" ? "Success" : "Failed"}
                            </span>
                          }
                        />
                      )}
                      <DetailRow label="Nonce" value={String(hexToNumber(selectedTx.nonce) ?? selectedTx.nonce)} />
                      {selectedReceipt?.effectiveGasPrice && (
                        <DetailRow label="Effective Gas Price" value={formatGwei(selectedReceipt.effectiveGasPrice)} />
                      )}
                    </div>

                    {selectedReceipt && (
                      <div>
                        <div className="uppercase text-[10px] tracking-widest text-muted-foreground mb-1">Logs emitted</div>
                        <Badge variant="secondary">{selectedReceipt.logs?.length ?? 0} log{selectedReceipt.logs?.length === 1 ? "" : "s"}</Badge>
                      </div>
                    )}

                    <div>
                      <div className="uppercase text-[10px] tracking-widest text-muted-foreground mb-1">Input data</div>
                      <pre className="text-[10px] bg-muted p-2 rounded overflow-auto max-h-24 font-mono break-all">
                        {selectedTx.input && selectedTx.input !== "0x" ? selectedTx.input : "(empty)"}
                      </pre>
                    </div>

                    <div className="pt-1 flex flex-wrap gap-2">
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => navigator.clipboard.writeText(selectedTxHash!)}
                      >
                        <Copy className="h-3.5 w-3.5 mr-1.5" /> Copy tx hash
                      </Button>
                      {selectedTx.blockHash && (
                        <Button size="sm" variant="outline" onClick={() => selectBlock({ hash: selectedTx.blockHash } as any)}>
                          View containing block
                        </Button>
                      )}
                    </div>
                  </div>
                )}
              </CardContent>
            </Card>
          )}
        </div>
      </div>

      <p className="text-[10px] text-muted-foreground px-1 -mt-2">
        Data is fetched live from the local Titan nodes via the C-Chain JSON-RPC. Some fields (baseFee, effective price) are post-London.
      </p>
    </div>
  );
}

function DetailRow({
  label,
  value,
  mono,
  copyValue,
}: {
  label: string;
  value: React.ReactNode;
  mono?: boolean;
  copyValue?: string;
}) {
  const [copied, setCopied] = useState(false);

  async function doCopy() {
    if (!copyValue) return;
    await navigator.clipboard.writeText(copyValue);
    setCopied(true);
    setTimeout(() => setCopied(false), 1200);
  }

  return (
    <div className="flex justify-between gap-3 py-0.5 border-b border-dashed last:border-none border-border/60">
      <span className="text-muted-foreground shrink-0">{label}</span>
      <span className={`${mono ? "font-mono text-xs break-all" : "font-medium break-all"} text-right flex items-center gap-1.5`}>
        {value}
        {copyValue && (
          <button onClick={doCopy} className="text-muted-foreground hover:text-foreground" title="Copy">
            <Copy className="h-3 w-3" />
          </button>
        )}
        {copied && <span className="text-[10px] text-emerald-600">copied</span>}
      </span>
    </div>
  );
}
