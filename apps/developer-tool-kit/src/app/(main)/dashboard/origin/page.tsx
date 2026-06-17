"use client";

import { useEffect, useState } from "react";

import {
  Copy,
  Download,
  ExternalLink,
  FileJson,
  GitBranch,
  Loader2,
  Network,
  RefreshCw,
  Server,
  Wallet,
} from "lucide-react";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { shortAddress } from "@/lib/titan/format";

interface OriginResponse {
  origin: {
    networkID: number;
    message: string;
    startTime: number;
    initialStakeDuration: number;
    initialStakers: Array<{
      nodeID: string;
      rewardAddress: string;
      delegationFee: number;
    }>;
    allocations: Array<{ ethAddr: string; avaxAddr: string }>;
  };
  cChain: {
    config?: { chainId?: number };
  };
  prefundedAccounts: Array<{ address: string; balanceTitan: string }>;
  sourcePath: string;
  githubRawUrl: string;
  githubBlobUrl: string;
  apiRawUrl: string;
  dockerEnv: {
    ORIGIN_URL: string;
    GENESIS_FILE: string;
    TITAN_NETWORK_ID: string;
  };
  error?: string;
}

function CopyButton({ value, label }: { value: string; label: string }) {
  const [copied, setCopied] = useState(false);

  async function copy() {
    try {
      await navigator.clipboard.writeText(value);
      setCopied(true);
      setTimeout(() => setCopied(false), 1500);
    } catch {
      setCopied(false);
    }
  }

  return (
    <Button variant="ghost" size="icon" className="h-7 w-7 shrink-0" onClick={copy} title={`Copy ${label}`}>
      <Copy className="h-3.5 w-3.5" />
      {copied && <span className="sr-only">Copied</span>}
    </Button>
  );
}

export default function OriginPage() {
  const [data, setData] = useState<OriginResponse | null>(null);
  const [rawJson, setRawJson] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  async function load() {
    setLoading(true);
    setError("");
    try {
      const [summaryRes, rawRes] = await Promise.all([
        fetch("/api/titan/origin"),
        fetch("/api/titan/origin?format=raw"),
      ]);
      if (!summaryRes.ok) {
        throw new Error((await summaryRes.json()).error ?? "Failed to load origin");
      }
      const summary = (await summaryRes.json()) as OriginResponse;
      setData(summary);
      setRawJson(await rawRes.text());
    } catch (e) {
      setData(null);
      setRawJson("");
      setError(String(e));
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    load();
  }, []);

  const explorerOriginUrl =
    typeof window !== "undefined" ? `${window.location.origin}/api/titan/origin?format=raw` : "/api/titan/origin?format=raw";

  return (
    <div className="flex flex-col gap-6">
      <div className="flex items-center justify-between gap-4">
        <div>
          <h1 className="text-2xl font-bold tracking-tight">Network Origin</h1>
          <p className="text-sm text-muted-foreground">
            Genesis definition for Titan — browsable here and downloadable by Docker nodes
          </p>
        </div>
        <Button variant="outline" size="sm" onClick={load} disabled={loading}>
          {loading ? <Loader2 className="h-4 w-4 animate-spin" /> : <RefreshCw className="h-4 w-4" />}
          Refresh
        </Button>
      </div>

      {loading && !data ? (
        <div className="flex items-center gap-2 text-muted-foreground">
          <Loader2 className="h-4 w-4 animate-spin" />
          Loading origin.json…
        </div>
      ) : error ? (
        <Card className="border-destructive/40">
          <CardContent className="pt-6 text-sm text-destructive">{error}</CardContent>
        </Card>
      ) : data ? (
        <>
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-4">
            <StatCard
              icon={<Network className="h-4 w-4 text-muted-foreground" />}
              title="Network ID"
              value={String(data.origin.networkID)}
              sub={`0x${data.origin.networkID.toString(16)}`}
            />
            <StatCard
              icon={<GitBranch className="h-4 w-4 text-muted-foreground" />}
              title="C-Chain ID"
              value={String(data.cChain.config?.chainId ?? "—")}
              sub="EVM chain identifier"
            />
            <StatCard
              icon={<Server className="h-4 w-4 text-muted-foreground" />}
              title="Initial stakers"
              value={String(data.origin.initialStakers.length)}
              sub={`${data.origin.allocations.length} P-chain allocations`}
            />
            <StatCard
              icon={<Wallet className="h-4 w-4 text-muted-foreground" />}
              title="Prefunded accounts"
              value={String(data.prefundedAccounts.length)}
              sub="C-chain genesis alloc"
            />
          </div>

          <Card>
            <CardHeader>
              <CardTitle className="text-base">Genesis message</CardTitle>
              <CardDescription>
                {data.origin.message} · start {new Date(data.origin.startTime * 1000).toUTCString()}
              </CardDescription>
            </CardHeader>
          </Card>

          <div className="grid gap-4 lg:grid-cols-2">
            <Card>
              <CardHeader>
                <CardTitle className="text-base flex items-center gap-2">
                  <Download className="h-4 w-4" />
                  Download endpoints
                </CardTitle>
                <CardDescription>Nodes fetch origin.json from one of these URLs at container start</CardDescription>
              </CardHeader>
              <CardContent className="space-y-3 text-sm">
                <EndpointRow label="GitHub (raw)" value={data.githubRawUrl} href={data.githubRawUrl} />
                <EndpointRow label="Explorer API" value={explorerOriginUrl} href={explorerOriginUrl} />
                <EndpointRow label="GitHub (browse)" value={data.githubBlobUrl} href={data.githubBlobUrl} />
              </CardContent>
            </Card>

            <Card>
              <CardHeader>
                <CardTitle className="text-base flex items-center gap-2">
                  <FileJson className="h-4 w-4" />
                  Docker environment
                </CardTitle>
                <CardDescription>Used by docker compose and remote node deployments</CardDescription>
              </CardHeader>
              <CardContent>
                <pre className="rounded-lg bg-muted p-3 text-xs font-mono overflow-x-auto whitespace-pre-wrap">
                  {`ORIGIN_URL=${data.dockerEnv.ORIGIN_URL}
GENESIS_FILE=${data.dockerEnv.GENESIS_FILE}
TITAN_NETWORK_ID=${data.dockerEnv.TITAN_NETWORK_ID}`}
                </pre>
                <p className="text-xs text-muted-foreground mt-3">
                  Node 2+ also needs bootstrap settings pointing at node 1&apos;s IP and NodeID.
                </p>
              </CardContent>
            </Card>
          </div>

          <Tabs defaultValue="stakers">
            <TabsList>
              <TabsTrigger value="stakers">Initial stakers</TabsTrigger>
              <TabsTrigger value="accounts">Prefunded C-chain</TabsTrigger>
              <TabsTrigger value="raw">Raw JSON</TabsTrigger>
            </TabsList>

            <TabsContent value="stakers" className="mt-4">
              <Card>
                <CardContent className="pt-6">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Node ID</TableHead>
                        <TableHead>Reward address</TableHead>
                        <TableHead className="text-right">Delegation fee</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {data.origin.initialStakers.map((staker) => (
                        <TableRow key={staker.nodeID}>
                          <TableCell className="font-mono text-xs">{staker.nodeID}</TableCell>
                          <TableCell className="font-mono text-xs">{staker.rewardAddress}</TableCell>
                          <TableCell className="text-right tabular-nums">
                            {(staker.delegationFee / 1_000_000).toFixed(2)}%
                          </TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="accounts" className="mt-4">
              <Card>
                <CardContent className="pt-6">
                  <Table>
                    <TableHeader>
                      <TableRow>
                        <TableHead>Address</TableHead>
                        <TableHead className="text-right">Balance</TableHead>
                      </TableRow>
                    </TableHeader>
                    <TableBody>
                      {data.prefundedAccounts.map((account) => (
                        <TableRow key={account.address}>
                          <TableCell className="font-mono text-xs" title={account.address}>
                            {shortAddress(account.address)}
                          </TableCell>
                          <TableCell className="text-right font-medium">{account.balanceTitan}</TableCell>
                        </TableRow>
                      ))}
                    </TableBody>
                  </Table>
                </CardContent>
              </Card>
            </TabsContent>

            <TabsContent value="raw" className="mt-4">
              <Card>
                <CardHeader className="flex flex-row items-center justify-between">
                  <div>
                    <CardTitle className="text-base">origin.json</CardTitle>
                    <CardDescription className="font-mono text-xs break-all">{data.sourcePath}</CardDescription>
                  </div>
                  <Badge variant="secondary">JSON</Badge>
                </CardHeader>
                <CardContent>
                  <pre className="max-h-[480px] overflow-auto rounded-lg bg-muted p-4 text-xs font-mono">
                    {rawJson}
                  </pre>
                </CardContent>
              </Card>
            </TabsContent>
          </Tabs>
        </>
      ) : null}
    </div>
  );
}

function StatCard({
  icon,
  title,
  value,
  sub,
}: {
  icon: React.ReactNode;
  title: string;
  value: string;
  sub?: string;
}) {
  return (
    <Card>
      <CardHeader className="flex flex-row items-center justify-between pb-2">
        <CardTitle className="text-sm font-medium text-muted-foreground">{title}</CardTitle>
        {icon}
      </CardHeader>
      <CardContent>
        <div className="text-2xl font-bold tabular-nums">{value}</div>
        {sub && <p className="text-xs text-muted-foreground mt-1">{sub}</p>}
      </CardContent>
    </Card>
  );
}

function EndpointRow({ label, value, href }: { label: string; value: string; href: string }) {
  return (
    <div className="flex items-start gap-2">
      <div className="min-w-0 flex-1">
        <p className="text-xs text-muted-foreground mb-0.5">{label}</p>
        <a
          href={href}
          target="_blank"
          rel="noreferrer"
          className="font-mono text-xs break-all hover:underline inline-flex items-center gap-1"
        >
          {value}
          <ExternalLink className="h-3 w-3 shrink-0" />
        </a>
      </div>
      <CopyButton value={value} label={label} />
    </div>
  );
}