"use client";

import { useCallback, useEffect, useRef, useState } from "react";
import { useSearchParams } from "next/navigation";
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

const BLOCKS_PAGE_SIZE = 20;

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

function isTxHash(s: string) {
  return /^0x[0-9a-fA-F]{64}$/.test(s);
}
function isAddress(s: string) {
  return /^0x[0-9a-fA-F]{40}$/.test(s);
}
function isBlockNumber(s: string) {
  if (/^\d+$/.test(s)) return true;
  // 64-char hex values are block/tx hashes, not block numbers (RPC max is 64 bits).
  if (/^0x[0-9a-fA-F]{1,16}$/i.test(s)) return true;
  return false;
}

export default function ExplorerPage() {
  const searchParams = useSearchParams();
  const [nodes, setNodes] = useState<NodeInfo[]>([]);
  const [nodesLoading, setNodesLoading] = useState(true);
  const [lastUpdated, setLastUpdated] = useState<Date | null>(null);

  const [blocks, setBlocks] = useState<Block[]>([]);
  const [blocksLoading, setBlocksLoading] = useState(true);
  const [loadingMore, setLoadingMore] = useState(false);
  const [hasMoreBlocks, setHasMoreBlocks] = useState(true);
  const [blocksError, setBlocksError] = useState<string>("");
  const loadMoreRef = useRef<HTMLDivElement>(null);

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
        ? String(identifier)
        : typeof identifier === "number"
          ? `0x${identifier.toString(16)}`
          : identifier; // can be "latest" or hex block number
      const method = isHash ? "eth_getBlockByHash" : "eth_getBlockByNumber";
      const raw = await rpc(method, [param, true]);
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
  const loadRecentBlocks = useCallback(async (count = BLOCKS_PAGE_SIZE) => {
    setBlocksLoading(true);
    setBlocksError("");
    setHasMoreBlocks(true);
    try {
      const headHex = (await rpc("eth_blockNumber")) as string | null;
      if (!headHex) throw new Error("Could not read head block");
      const head = Number.parseInt(headHex, 16);

      const toFetch: number[] = [];
      for (let i = 0; i < count; i++) {
        const n = head - i;
        if (n < 0) break;
        toFetch.push(n);
      }

      const results = await Promise.all(toFetch.map((n) => fetchBlock(n)));
      const valid = results.filter(Boolean) as Block[];
      setBlocks(valid);
      setHasMoreBlocks(head - count > 0);
    } catch (e: unknown) {
      setBlocksError(e instanceof Error ? e.message : "Failed to load blocks");
      setBlocks([]);
      setHasMoreBlocks(false);
    } finally {
      setBlocksLoading(false);
    }
  }, [fetchBlock]);

  // Load older blocks (paginated batches for infinite scroll)
  const loadOlder = useCallback(
    async (count = BLOCKS_PAGE_SIZE) => {
      if (loadingMore || blocksLoading) return;

      if (blocks.length === 0) {
        await loadRecentBlocks(count);
        return;
      }

      const lowest = hexToNumber(blocks[blocks.length - 1].number) ?? 0;
      if (lowest <= 0) {
        setHasMoreBlocks(false);
        return;
      }

      setLoadingMore(true);
      try {
        const toFetch: number[] = [];
        for (let i = 1; i <= count; i++) {
          const n = lowest - i;
          if (n < 0) break;
          toFetch.push(n);
        }

        if (toFetch.length === 0) {
          setHasMoreBlocks(false);
          return;
        }

        const results = await Promise.all(toFetch.map((n) => fetchBlock(n)));
        const older = results.filter(Boolean) as Block[];
        if (older.length) {
          setBlocks((prev) => [...prev, ...older]);
        }
        if (lowest - count <= 0 || older.length < count) {
          setHasMoreBlocks(false);
        }
      } finally {
        setLoadingMore(false);
      }
    },
    [blocks, blocksLoading, fetchBlock, loadRecentBlocks, loadingMore],
  );

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
      if (isAddress(q)) {
        // Simple address lookup: show balance
        const balHex = (await rpc("eth_getBalance", [q, "latest"])) as string;
        const titan = formatWeiToTitan(balHex);
        // For now surface in search error area (or we could open a mini panel)
        setSearchError(`Address balance: ${titan} TITAN  (${balHex})`);
        // optionally clear after a bit, but leave it visible
        setTimeout(() => setSearchError((prev) => (prev.includes("Address balance") ? "" : prev)), 8000);
      } else if (isTxHash(q)) {
        const txRaw = await rpc("eth_getTransactionByHash", [q]);
        if (txRaw) {
          await loadTx(q);
          setSearchValue("");
        } else {
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
            setSearchError("Transaction or block not found");
          }
        }
      } else if (isBlockNumber(q)) {
        const num = q.startsWith("0x") ? Number.parseInt(q, 16) : Number.parseInt(q, 10);
        const b = await fetchBlock(num);
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

  useEffect(() => {
    const q = searchParams.get("q")?.trim();
    if (!q) return;
    setSearchValue(q);
    void performSearch(q);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [searchParams]);

  // Auto-refresh nodes + detect new head and refresh blocks
  useEffect(() => {
    loadNodes();
    const id = setInterval(loadNodes, 10000);
    return () => clearInterval(id);
  }, []);

  // Initial blocks + periodic head check
  useEffect(() => {
    loadRecentBlocks(BLOCKS_PAGE_SIZE);

    const headPoll = setInterval(async () => {
      try {
        const headHex = (await rpc("eth_blockNumber")) as string;
        const head = hexToNumber(headHex);
        if (head == null || blocks.length === 0) return;

        const currentHead = hexToNumber(blocks[0]?.number);
        if (head > (currentHead ?? -1)) {
          await loadRecentBlocks(BLOCKS_PAGE_SIZE);
        }
      } catch {
        /* ignore poll errors */
      }
    }, 8000);

    return () => clearInterval(headPoll);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  // Infinite scroll: load older blocks when sentinel enters view
  useEffect(() => {
    const sentinel = loadMoreRef.current;
    if (!sentinel || !hasMoreBlocks || loadingMore || blocksLoading) return;

    const observer = new IntersectionObserver(
      (entries) => {
        if (entries[0]?.isIntersecting) {
          void loadOlder(BLOCKS_PAGE_SIZE);
        }
      },
      { rootMargin: "240px" },
    );

    observer.observe(sentinel);
    return () => observer.disconnect();
  }, [blocks.length, blocksLoading, hasMoreBlocks, loadOlder, loadingMore]);

  const headBlock = blocks[0]?.number ? hexToNumber(blocks[0].number)?.toLocaleString() : nodes.find((n) => n.blockNumber)?.blockNumber;
  const chainId = nodes.find((n) => n.chainId)?.chainId ?? "—";
  const gasPrice = nodes.find((n) => n.gasPrice)?.gasPrice ?? "—";
  const highestBlock = blocks[0]?.number ? hexToNumber(blocks[0].number) : null;
  const lowestBlock = blocks[blocks.length - 1]?.number ? hexToNumber(blocks[blocks.length - 1].number) : null;
  const loadedPages = Math.max(1, Math.ceil(blocks.length / BLOCKS_PAGE_SIZE));
  const currentDetailBlock = selectedBlock;

  return (
    <div className="flex flex-col gap-5">
      {/* Header */}
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight flex items-center gap-2">
            <Hash className="h-6 w-6" /> Explorer
          </h1>
          <p className="text-sm text-muted-foreground">Browse the Titan C-Chain · live blocks &amp; transactions</p>
        </div>

        <Button
          variant="outline"
          size="sm"
          onClick={() => {
            loadNodes();
            loadRecentBlocks(BLOCKS_PAGE_SIZE);
            setSelectedBlock(null);
            setSelectedTx(null);
            setSelectedTxHash(null);
            setSelectedReceipt(null);
          }}
          disabled={blocksLoading || nodesLoading}
        >
          {blocksLoading || nodesLoading ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
          Refresh
        </Button>
      </div>

      {/* Compact network strip */}
      <div className="flex flex-wrap items-center gap-x-6 gap-y-2 rounded-lg border bg-muted/20 px-4 py-3 text-sm">
        <div className="flex items-center gap-2">
          <Zap className="h-4 w-4 text-muted-foreground" />
          <span className="text-muted-foreground">Head</span>
          <span className="font-mono font-semibold tabular-nums">{headBlock ?? "—"}</span>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-muted-foreground">Chain</span>
          <span className="font-mono font-semibold">{chainId}</span>
        </div>
        <div className="flex items-center gap-2">
          <span className="text-muted-foreground">Gas</span>
          <span className="font-mono text-xs">{gasPrice}</span>
        </div>
        <div className="flex flex-wrap items-center gap-1.5">
          {NODES.map(({ node, port }) => {
            const info = nodes.find((n) => n.node === node);
            const ok = info?.healthy;
            return (
              <Badge key={node} variant={ok ? "default" : "secondary"} className={ok ? "bg-green-600" : ""}>
                {node}:{port}
              </Badge>
            );
          })}
          {nodesLoading && <Loader2 className="h-3 w-3 animate-spin" />}
        </div>
        {lastUpdated && (
          <span className="text-xs text-muted-foreground ml-auto">
            <Clock className="inline h-3 w-3 mr-1" />
            {lastUpdated.toLocaleTimeString()}
          </span>
        )}
      </div>

      {/* Search */}
      <div className="rounded-lg border px-4 py-3">
        <div className="flex flex-col gap-2 sm:flex-row">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-2.5 h-4 w-4 text-muted-foreground" />
            <Input
              value={searchValue}
              onChange={(e) => setSearchValue(e.target.value)}
              onKeyDown={onSearchKey}
              placeholder="Block number, block/tx hash (0x...), or address"
              className="pl-9 font-mono text-sm"
              disabled={searchLoading}
            />
          </div>
          <Button onClick={() => performSearch(searchValue)} disabled={searchLoading || !searchValue.trim()}>
            {searchLoading ? <Loader2 className="h-4 w-4 animate-spin" /> : <Search className="h-4 w-4" />}
            Search
          </Button>
        </div>
        {searchError && <p className="mt-2 text-xs text-amber-600 break-all">{searchError}</p>}
      </div>

      {/* Blocks feed — single column, infinite scroll */}
      <section className="rounded-lg border overflow-hidden">
        <div className="flex flex-wrap items-center justify-between gap-2 border-b bg-muted/30 px-4 py-3">
          <div className="flex items-center gap-2 text-sm font-semibold">
            <Activity className="h-4 w-4" />
            Blocks
          </div>
          <div className="text-xs text-muted-foreground">
            {blocks.length > 0 && highestBlock != null && lowestBlock != null ? (
              <>
                #{highestBlock.toLocaleString()} → #{lowestBlock.toLocaleString()} · {blocks.length} loaded · page {loadedPages}
              </>
            ) : (
              "Loading chain history…"
            )}
          </div>
        </div>

        <table className="w-full text-sm">
          <thead className="bg-muted/40 text-muted-foreground text-xs uppercase tracking-wider border-b">
            <tr>
              <th className="px-4 py-2.5 text-left font-medium w-28">Block</th>
              <th className="px-4 py-2.5 text-left font-medium w-24">Age</th>
              <th className="px-4 py-2.5 text-right font-medium w-16">Txs</th>
              <th className="px-4 py-2.5 text-right font-medium hidden sm:table-cell">Gas Used</th>
              <th className="px-4 py-2.5 text-left font-medium hidden md:table-cell">Hash</th>
            </tr>
          </thead>
          <tbody className="divide-y">
            {blocksLoading && blocks.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-4 py-12 text-center text-muted-foreground">
                  <Loader2 className="mx-auto h-5 w-5 animate-spin" />
                </td>
              </tr>
            ) : blocks.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-4 py-8 text-center text-muted-foreground">
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
                    className={`cursor-pointer transition-colors hover:bg-muted/50 ${isSel ? "bg-muted/70" : ""}`}
                  >
                    <td className="px-4 py-2.5 font-mono font-semibold tabular-nums">#{num?.toLocaleString()}</td>
                    <td className="px-4 py-2.5 text-muted-foreground text-xs">{ago}</td>
                    <td className="px-4 py-2.5 text-right font-mono tabular-nums">{txCount}</td>
                    <td className="px-4 py-2.5 text-right font-mono text-xs text-muted-foreground hidden sm:table-cell">
                      {hexToNumber(b.gasUsed)?.toLocaleString() ?? "—"}
                    </td>
                    <td className="px-4 py-2.5 font-mono text-xs text-muted-foreground hidden md:table-cell">
                      {shortHash(b.hash)}
                    </td>
                  </tr>
                );
              })
            )}
          </tbody>
        </table>

        <div ref={loadMoreRef} className="border-t bg-muted/20 px-4 py-3 flex flex-wrap items-center justify-between gap-3">
          <div className="text-xs text-muted-foreground">
            {loadingMore ? (
              <span className="inline-flex items-center gap-1.5">
                <Loader2 className="h-3.5 w-3.5 animate-spin" />
                Loading {BLOCKS_PAGE_SIZE} older blocks…
              </span>
            ) : hasMoreBlocks ? (
              "Scroll down to load older blocks"
            ) : (
              "Reached the earliest loaded block"
            )}
          </div>
          <div className="flex items-center gap-2">
            <Button
              variant="outline"
              size="sm"
              onClick={() => loadOlder(BLOCKS_PAGE_SIZE)}
              disabled={!hasMoreBlocks || loadingMore || blocksLoading || blocks.length === 0}
            >
              Load {BLOCKS_PAGE_SIZE} more
            </Button>
            <Button variant="ghost" size="sm" onClick={() => loadRecentBlocks(BLOCKS_PAGE_SIZE)} disabled={blocksLoading}>
              Back to latest
            </Button>
          </div>
        </div>
      </section>

      {/* Block detail — full width below feed */}
      {currentDetailBlock && (
        <section className="rounded-lg border">
          <div className="flex items-start justify-between gap-3 border-b bg-muted/30 px-4 py-3">
            <div>
              <h2 className="text-base font-semibold">Block #{hexToNumber(currentDetailBlock.number)?.toLocaleString()}</h2>
              <p className="text-xs text-muted-foreground font-mono mt-0.5 break-all">{currentDetailBlock.hash}</p>
            </div>
            <Button size="icon" variant="ghost" onClick={() => selectBlock(null)} title="Close block details">
              <ArrowLeft className="h-4 w-4" />
            </Button>
          </div>

          <div className="px-4 py-4">
            {selectedBlockLoading ? (
              <div className="flex items-center gap-2 text-sm text-muted-foreground py-4">
                <Loader2 className="h-4 w-4 animate-spin" /> Loading block…
              </div>
            ) : (
              <div className="space-y-4 text-sm">
                <div className="grid grid-cols-1 gap-x-8 gap-y-1 sm:grid-cols-2 lg:grid-cols-3">
                  <DetailRow label="Timestamp" value={formatTimestamp(currentDetailBlock.timestamp).full} />
                  <DetailRow label="Age" value={formatTimestamp(currentDetailBlock.timestamp).ago} />
                  <DetailRow
                    label="Transactions"
                    value={String(
                      currentDetailBlock.transactionCount ??
                        (Array.isArray(currentDetailBlock.transactions) ? currentDetailBlock.transactions.length : 0),
                    )}
                  />
                  <DetailRow
                    label="Gas Used / Limit"
                    value={`${hexToNumber(currentDetailBlock.gasUsed)?.toLocaleString() ?? "—"} / ${hexToNumber(currentDetailBlock.gasLimit)?.toLocaleString() ?? "—"}`}
                  />
                  {currentDetailBlock.baseFeePerGas && (
                    <DetailRow label="Base Fee" value={formatGwei(currentDetailBlock.baseFeePerGas)} />
                  )}
                  <DetailRow label="Parent Hash" value={shortHash(currentDetailBlock.parentHash)} mono copyValue={currentDetailBlock.parentHash} />
                </div>

                <div>
                  <h3 className="mb-2 text-xs font-medium uppercase tracking-widest text-muted-foreground">
                    Transactions (
                    {currentDetailBlock.transactionCount ??
                      (Array.isArray(currentDetailBlock.transactions) ? currentDetailBlock.transactions.length : 0)}
                    )
                  </h3>

                  {Array.isArray(currentDetailBlock.transactions) && currentDetailBlock.transactions.length > 0 ? (
                    <div className="overflow-hidden rounded-md border">
                      <table className="w-full text-xs">
                        <thead>
                          <tr className="bg-muted/60 text-muted-foreground">
                            <th className="px-3 py-1.5 text-left font-normal">#</th>
                            <th className="px-3 py-1.5 text-left font-normal">Hash</th>
                            <th className="px-3 py-1.5 text-left font-normal hidden md:table-cell">From → To</th>
                            <th className="px-3 py-1.5 text-right font-normal">Value</th>
                            <th className="px-3 py-1.5 text-right font-normal hidden sm:table-cell">Gas</th>
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
                                <td className="px-3 py-1.5 font-mono tabular-nums text-muted-foreground">{idx}</td>
                                <td className="px-3 py-1.5 font-mono text-primary hover:underline">{shortHash(tx.hash)}</td>
                                <td className="px-3 py-1.5 text-muted-foreground hidden md:table-cell font-mono text-[10px] truncate max-w-65">
                                  {shortHash(tx.from, 4, 4)} → {tx.to ? shortHash(tx.to, 4, 4) : "contract"}
                                </td>
                                <td className="px-3 py-1.5 text-right font-medium tabular-nums">{val}</td>
                                <td className="px-3 py-1.5 text-right text-muted-foreground hidden sm:table-cell">{g.toLocaleString()}</td>
                              </tr>
                            );
                          })}
                        </tbody>
                      </table>
                    </div>
                  ) : (
                    <p className="text-xs text-muted-foreground py-2">No transactions in this block.</p>
                  )}
                </div>
              </div>
            )}
          </div>
        </section>
      )}

      {/* Transaction detail — full width below block detail */}
      {(selectedTxHash || selectedTx) && (
        <section className="rounded-lg border">
          <div className="flex items-center justify-between gap-3 border-b bg-muted/30 px-4 py-3">
            <div>
              <h2 className="text-base font-semibold">Transaction</h2>
              <p className="font-mono text-xs text-muted-foreground break-all mt-0.5">{selectedTxHash}</p>
            </div>
            <Button
              size="sm"
              variant="ghost"
              onClick={() => {
                setSelectedTxHash(null);
                setSelectedTx(null);
                setSelectedReceipt(null);
              }}
            >
              Close
            </Button>
          </div>

          <div className="px-4 py-4">
            {txLoading ? (
              <div className="flex gap-2 text-sm py-4 text-muted-foreground">
                <Loader2 className="h-4 w-4 animate-spin" /> Loading transaction…
              </div>
            ) : !selectedTx ? (
              <div className="text-sm text-muted-foreground">Transaction not found or still loading.</div>
            ) : (
              <div className="space-y-3 text-sm">
                <div className="grid grid-cols-1 gap-x-8 gap-y-1 md:grid-cols-2 lg:grid-cols-3">
                  <DetailRow label="From" value={shortHash(selectedTx.from)} mono copyValue={selectedTx.from} />
                  <DetailRow label="To" value={selectedTx.to ? shortHash(selectedTx.to) : "Contract creation"} mono copyValue={selectedTx.to ?? undefined} />
                  <DetailRow label="Value" value={`${formatWeiToTitan(selectedTx.value)} TITAN`} />
                  <DetailRow label="Gas Price" value={selectedTx.gasPrice ? formatGwei(selectedTx.gasPrice) : "—"} />
                  <DetailRow
                    label="Gas Limit / Used"
                    value={`${hexToNumber(selectedTx.gas)?.toLocaleString() ?? "—"} ${selectedReceipt ? ` / ${hexToNumber(selectedReceipt.gasUsed)?.toLocaleString()}` : ""}`}
                  />
                  {selectedReceipt && (
                    <DetailRow
                      label="Status"
                      value={
                        <span
                          className={
                            selectedReceipt.status === "0x1" || selectedReceipt.status === "0x01"
                              ? "text-green-600 font-medium"
                              : "text-red-600 font-medium"
                          }
                        >
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
                    <Badge variant="secondary">
                      {selectedReceipt.logs?.length ?? 0} log{selectedReceipt.logs?.length === 1 ? "" : "s"}
                    </Badge>
                  </div>
                )}

                <div>
                  <div className="uppercase text-[10px] tracking-widest text-muted-foreground mb-1">Input data</div>
                  <pre className="text-[10px] bg-muted p-2 rounded overflow-auto max-h-24 font-mono break-all">
                    {selectedTx.input && selectedTx.input !== "0x" ? selectedTx.input : "(empty)"}
                  </pre>
                </div>

                <div className="flex flex-wrap gap-2">
                  <Button size="sm" variant="outline" onClick={() => navigator.clipboard.writeText(selectedTxHash!)}>
                    <Copy className="h-3.5 w-3.5 mr-1.5" /> Copy tx hash
                  </Button>
                  {selectedTx.blockHash && (
                    <Button size="sm" variant="outline" onClick={() => selectBlock({ hash: selectedTx.blockHash } as Block)}>
                      View containing block
                    </Button>
                  )}
                </div>
              </div>
            )}
          </div>
        </section>
      )}

      <p className="text-[10px] text-muted-foreground px-1">
        Data is fetched live from local Titan nodes via C-Chain JSON-RPC. Blocks load in batches of {BLOCKS_PAGE_SIZE}.
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
