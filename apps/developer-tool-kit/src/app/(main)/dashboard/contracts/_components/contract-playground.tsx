"use client";

import { useCallback, useEffect, useState } from "react";

import { FlaskConical, Loader2, Minus, Plus, RefreshCw } from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import type { DeployedContractRecord } from "@/lib/titan/deployed-contracts-storage";
import { readContractFunction, writeContractFunction } from "@/lib/titan/contract-interact";
import {
  resolveSandboxTemplateId,
  SANDBOX_ABIS,
  sandboxLabel,
  type SandboxTemplateId,
} from "@/lib/titan/contract-sandbox";
import { shortAddress } from "@/lib/titan/format";
import { parseWalletError } from "@/lib/titan/wallet-errors";
import { isOnTitanChain, isWalletConnected, useWalletStore } from "@/stores/wallet/wallet-store";

type ContractPlaygroundProps = {
  record: DeployedContractRecord;
};

export function ContractPlayground({ record }: ContractPlaygroundProps) {
  const templateId = resolveSandboxTemplateId(record);
  if (!templateId) return null;

  return <SandboxPanel record={record} templateId={templateId} />;
}

function SandboxPanel({
  record,
  templateId,
}: {
  record: DeployedContractRecord;
  templateId: SandboxTemplateId;
}) {
  const address = useWalletStore((s) => s.address);
  const chainId = useWalletStore((s) => s.chainId);
  const walletReady = isWalletConnected({ address });
  const onTitanChain = isOnTitanChain(chainId);

  const [loading, setLoading] = useState(false);
  const [writing, setWriting] = useState(false);
  const [error, setError] = useState("");
  const [lastTx, setLastTx] = useState<string | null>(null);

  const [greeting, setGreeting] = useState<string | null>(null);
  const [newGreeting, setNewGreeting] = useState("Hello from Titan sandbox");

  const [count, setCount] = useState<string | null>(null);
  const [storedValue, setStoredValue] = useState<string | null>(null);
  const [newStoredValue, setNewStoredValue] = useState("42");

  const abi = SANDBOX_ABIS[templateId];
  const contractAddress = record.contractAddress;

  const refresh = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      if (templateId === "greeter") {
        const value = await readContractFunction<string>({
          contractAddress,
          abi,
          functionName: "greet",
        });
        setGreeting(value);
      } else if (templateId === "counter") {
        const value = await readContractFunction<bigint>({
          contractAddress,
          abi,
          functionName: "count",
        });
        setCount(value.toString());
      } else {
        const value = await readContractFunction<bigint>({
          contractAddress,
          abi,
          functionName: "get",
        });
        setStoredValue(value.toString());
      }
    } catch (err) {
      setError(err instanceof Error ? err.message : "Failed to read contract.");
    } finally {
      setLoading(false);
    }
  }, [abi, contractAddress, templateId]);

  useEffect(() => {
    void refresh();
  }, [refresh]);

  async function runWrite(functionName: string, args: readonly unknown[] = []) {
    if (!walletReady) {
      setError("Connect MetaMask to send transactions.");
      return;
    }
    if (!onTitanChain) {
      setError("Switch MetaMask to Titan Local UAT.");
      return;
    }

    setWriting(true);
    setError("");
    setLastTx(null);
    try {
      const txHash = await writeContractFunction({
        from: address,
        contractAddress,
        abi,
        functionName,
        args,
      });
      setLastTx(txHash);
      await refresh();
    } catch (err) {
      setError(parseWalletError(err, "Transaction failed."));
    } finally {
      setWriting(false);
    }
  }

  return (
    <div className="rounded-md border border-dashed border-primary/30 bg-primary/5 p-4 space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2 text-sm font-medium">
          <FlaskConical className="h-4 w-4 text-primary" />
          Sandbox · {sandboxLabel(templateId)}
        </div>
        <Badge variant="secondary" className="font-mono text-[10px]">
          {shortAddress(contractAddress)}
        </Badge>
      </div>

      <p className="text-xs text-muted-foreground">
        Demo controls for built-in templates only. Read calls use the local RPC; writes go through MetaMask.
      </p>

      {templateId === "greeter" && (
        <div className="space-y-3">
          <div className="rounded-md border bg-background/80 px-3 py-3">
            <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1">greet()</p>
            <p className="text-lg font-medium break-words">
              {loading && greeting === null ? "Loading…" : (greeting ?? "—")}
            </p>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button size="sm" variant="outline" onClick={() => void refresh()} disabled={loading}>
              {loading ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RefreshCw className="h-3.5 w-3.5" />}
              Refresh
            </Button>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor={`greeting-${record.id}`} className="text-xs">
              setGreeting(string)
            </Label>
            <div className="flex flex-col gap-2 sm:flex-row">
              <Input
                id={`greeting-${record.id}`}
                value={newGreeting}
                onChange={(e) => setNewGreeting(e.target.value)}
                className="font-mono text-xs"
              />
              <Button
                size="sm"
                onClick={() => void runWrite("setGreeting", [newGreeting])}
                disabled={writing || !newGreeting.trim()}
              >
                {writing ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : null}
                Update greeting
              </Button>
            </div>
          </div>
        </div>
      )}

      {templateId === "counter" && (
        <div className="space-y-3">
          <div className="rounded-md border bg-background/80 px-3 py-3">
            <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1">count</p>
            <p className="text-3xl font-semibold tabular-nums">{loading && count === null ? "…" : (count ?? "—")}</p>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button size="sm" variant="outline" onClick={() => void refresh()} disabled={loading}>
              {loading ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RefreshCw className="h-3.5 w-3.5" />}
              Refresh
            </Button>
            <Button size="sm" onClick={() => void runWrite("increment")} disabled={writing}>
              {writing ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Plus className="h-3.5 w-3.5" />}
              increment()
            </Button>
            <Button size="sm" variant="secondary" onClick={() => void runWrite("decrement")} disabled={writing}>
              {writing ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <Minus className="h-3.5 w-3.5" />}
              decrement()
            </Button>
          </div>
        </div>
      )}

      {templateId === "simple-storage" && (
        <div className="space-y-3">
          <div className="rounded-md border bg-background/80 px-3 py-3">
            <p className="text-[10px] uppercase tracking-wider text-muted-foreground mb-1">get()</p>
            <p className="text-3xl font-semibold tabular-nums">
              {loading && storedValue === null ? "…" : (storedValue ?? "—")}
            </p>
          </div>
          <div className="flex flex-wrap gap-2">
            <Button size="sm" variant="outline" onClick={() => void refresh()} disabled={loading}>
              {loading ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : <RefreshCw className="h-3.5 w-3.5" />}
              Refresh
            </Button>
          </div>
          <div className="space-y-1.5">
            <Label htmlFor={`value-${record.id}`} className="text-xs">
              set(uint256)
            </Label>
            <div className="flex flex-col gap-2 sm:flex-row">
              <Input
                id={`value-${record.id}`}
                value={newStoredValue}
                onChange={(e) => setNewStoredValue(e.target.value)}
                className="font-mono text-xs"
                inputMode="numeric"
              />
              <Button
                size="sm"
                onClick={() => void runWrite("set", [BigInt(newStoredValue)])}
                disabled={writing || !/^\d+$/.test(newStoredValue.trim())}
              >
                {writing ? <Loader2 className="h-3.5 w-3.5 animate-spin" /> : null}
                Store value
              </Button>
            </div>
          </div>
        </div>
      )}

      {error && <p className="text-xs text-red-600 break-all">{error}</p>}
      {lastTx && (
        <p className="text-[10px] font-mono text-muted-foreground break-all">
          Last tx: {lastTx}
        </p>
      )}
    </div>
  );
}