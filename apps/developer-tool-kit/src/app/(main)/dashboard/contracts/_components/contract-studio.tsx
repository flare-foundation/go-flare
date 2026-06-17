"use client";

import { useMemo, useState } from "react";

import Link from "next/link";

import {
  CheckCircle2,
  Code2,
  ExternalLink,
  FileCode2,
  Loader2,
  Play,
  Rocket,
  Wallet,
} from "lucide-react";
import type { Abi, AbiParameter } from "viem";

import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Textarea } from "@/components/ui/textarea";
import { APP_CONFIG } from "@/config/app-config";
import type { AbiConstructorInput, CompiledContract } from "@/lib/titan/compile-contract";
import {
  CONTRACT_TEMPLATES,
  DEFAULT_TEMPLATE_ID,
} from "@/lib/titan/contract-templates";
import {
  deployContract,
  getConstructorAbi,
  parseConstructorArgValue,
} from "@/lib/titan/deploy-contract";
import { shortAddress } from "@/lib/titan/format";
import { isOnTitanChain, isWalletConnected, useWalletStore } from "@/stores/wallet/wallet-store";

type DeployResult = {
  transactionHash: string;
  contractAddress: string;
};

export function ContractStudio() {
  const defaultTemplate = CONTRACT_TEMPLATES.find((t) => t.id === DEFAULT_TEMPLATE_ID) ?? CONTRACT_TEMPLATES[0];

  const [templateId, setTemplateId] = useState(defaultTemplate.id);
  const [source, setSource] = useState(defaultTemplate.source);
  const [compiled, setCompiled] = useState<CompiledContract | null>(null);
  const [compileErrors, setCompileErrors] = useState<string[]>([]);
  const [constructorArgValues, setConstructorArgValues] = useState<Record<string, string>>({});
  const [isCompiling, setIsCompiling] = useState(false);
  const [isDeploying, setIsDeploying] = useState(false);
  const [deployError, setDeployError] = useState("");
  const [deployResult, setDeployResult] = useState<DeployResult | null>(null);

  const address = useWalletStore((s) => s.address);
  const chainId = useWalletStore((s) => s.chainId);
  const titanBalance = useWalletStore((s) => s.titanBalance);
  const connect = useWalletStore((s) => s.connect);

  const walletReady = isWalletConnected({ address });
  const onTitanChain = isOnTitanChain(chainId);

  const selectedTemplate = useMemo(
    () => CONTRACT_TEMPLATES.find((t) => t.id === templateId) ?? defaultTemplate,
    [defaultTemplate, templateId],
  );

  function applyTemplate(nextTemplateId: string) {
    const template = CONTRACT_TEMPLATES.find((t) => t.id === nextTemplateId);
    if (!template) return;
    setTemplateId(template.id);
    setSource(template.source);
    setCompiled(null);
    setCompileErrors([]);
    setConstructorArgValues({});
    setDeployError("");
    setDeployResult(null);
  }

  async function handleCompile() {
    setIsCompiling(true);
    setCompileErrors([]);
    setCompiled(null);
    setDeployError("");
    setDeployResult(null);

    try {
      const response = await fetch("/api/titan/compile", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({ source }),
      });

      const result = (await response.json()) as
        | { ok: true; contract: CompiledContract }
        | { ok: false; errors: string[] };

      if (!result.ok) {
        setCompileErrors(result.errors);
        return;
      }

      setCompiled(result.contract);
      const initialArgs: Record<string, string> = {};
      for (const input of result.contract.constructorInputs) {
        initialArgs[input.name || input.type] = defaultArgValue(input);
      }
      setConstructorArgValues(initialArgs);
    } catch (error) {
      setCompileErrors([error instanceof Error ? error.message : "Compilation request failed."]);
    } finally {
      setIsCompiling(false);
    }
  }

  async function handleDeploy() {
    if (!compiled || !walletReady) return;

    setDeployError("");
    setDeployResult(null);
    setIsDeploying(true);

    try {
      const constructorAbi = getConstructorAbi(compiled.abi as Abi);
      const args =
        constructorAbi?.inputs.map((input: AbiParameter) => {
          const key = input.name || input.type;
          const raw = constructorArgValues[key] ?? "";
          return parseConstructorArgValue(input.type, raw);
        }) ?? [];

      const result = await deployContract({
        from: address,
        abi: compiled.abi as Abi,
        bytecode: compiled.bytecode,
        constructorArgs: args,
      });

      setDeployResult(result);
    } catch (error) {
      setDeployError(error instanceof Error ? error.message : "Deployment failed.");
    } finally {
      setIsDeploying(false);
    }
  }

  return (
    <div className="flex flex-col gap-5">
      <div className="flex flex-col gap-3 sm:flex-row sm:items-end sm:justify-between">
        <div>
          <h1 className="text-2xl font-bold tracking-tight flex items-center gap-2">
            <Code2 className="h-6 w-6" />
            Contract Studio
          </h1>
          <p className="text-sm text-muted-foreground">
            Write Solidity, compile, and deploy to {APP_CONFIG.titan.networkName} with MetaMask.
          </p>
        </div>
        <Badge variant={walletReady && onTitanChain ? "default" : "secondary"}>
          {walletReady ? (onTitanChain ? "Wallet ready" : "Wrong network") : "Wallet not connected"}
        </Badge>
      </div>

      {!walletReady && (
        <div className="flex flex-wrap items-center justify-between gap-3 rounded-lg border border-dashed px-4 py-3 text-sm">
          <div className="flex items-center gap-2 text-muted-foreground">
            <Wallet className="h-4 w-4" />
            Connect MetaMask from the sidebar to deploy contracts.
          </div>
          <Button size="sm" variant="outline" onClick={() => void connect()}>
            Connect Wallet
          </Button>
        </div>
      )}

      <div className="grid gap-5 xl:grid-cols-2">
        <section className="flex flex-col gap-3 rounded-lg border">
          <div className="flex flex-wrap items-center justify-between gap-2 border-b bg-muted/30 px-4 py-3">
            <div className="flex items-center gap-2 text-sm font-semibold">
              <FileCode2 className="h-4 w-4" />
              Source
            </div>
            <Select value={templateId} onValueChange={applyTemplate}>
              <SelectTrigger className="w-44 h-8 text-xs">
                <SelectValue placeholder="Template" />
              </SelectTrigger>
              <SelectContent>
                {CONTRACT_TEMPLATES.map((template) => (
                  <SelectItem key={template.id} value={template.id}>
                    {template.name}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          <div className="px-4">
            <p className="text-xs text-muted-foreground mb-2">{selectedTemplate.description}</p>
            <Textarea
              value={source}
              onChange={(e) => setSource(e.target.value)}
              className="min-h-105 font-mono text-xs leading-relaxed"
              spellCheck={false}
            />
          </div>

          <div className="flex flex-wrap gap-2 border-t px-4 py-3">
            <Button onClick={handleCompile} disabled={isCompiling || !source.trim()}>
              {isCompiling ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />}
              Compile
            </Button>
            <Button variant="outline" onClick={() => applyTemplate(templateId)} disabled={isCompiling}>
              Reset template
            </Button>
          </div>
        </section>

        <section className="flex flex-col gap-4">
          <div className="rounded-lg border">
            <div className="border-b bg-muted/30 px-4 py-3 text-sm font-semibold">Compile output</div>
            <div className="space-y-3 px-4 py-4 text-sm">
              {compileErrors.length > 0 && (
                <div className="rounded-md border border-red-500/30 bg-red-500/5 p-3">
                  <p className="font-medium text-red-600 mb-2">Compilation failed</p>
                  <pre className="whitespace-pre-wrap text-xs text-red-600/90 font-mono">{compileErrors.join("\n\n")}</pre>
                </div>
              )}

              {!compiled && compileErrors.length === 0 && (
                <p className="text-muted-foreground text-sm">Compile your contract to see ABI and bytecode.</p>
              )}

              {compiled && (
                <div className="space-y-3">
                  <div className="flex flex-wrap items-center gap-2">
                    <Badge>{compiled.contractName}</Badge>
                    <span className="text-xs text-muted-foreground">
                      Bytecode size: {(compiled.bytecode.length / 2).toLocaleString()} bytes
                    </span>
                  </div>

                  <div>
                    <p className="text-xs uppercase tracking-wider text-muted-foreground mb-1">Constructor</p>
                    {compiled.constructorInputs.length === 0 ? (
                      <p className="text-sm text-muted-foreground">No constructor arguments.</p>
                    ) : (
                      <ul className="text-xs font-mono text-muted-foreground space-y-1">
                        {compiled.constructorInputs.map((input) => (
                          <li key={`${input.name}-${input.type}`}>
                            {input.type} {input.name}
                          </li>
                        ))}
                      </ul>
                    )}
                  </div>

                  <details className="rounded-md border">
                    <summary className="cursor-pointer px-3 py-2 text-xs font-medium">View ABI</summary>
                    <pre className="max-h-48 overflow-auto border-t p-3 text-[10px] font-mono">
                      {JSON.stringify(compiled.abi, null, 2)}
                    </pre>
                  </details>
                </div>
              )}
            </div>
          </div>

          <div className="rounded-lg border">
            <div className="border-b bg-muted/30 px-4 py-3 text-sm font-semibold flex items-center gap-2">
              <Rocket className="h-4 w-4" />
              Deploy
            </div>
            <div className="space-y-4 px-4 py-4">
              {walletReady && (
                <div className="rounded-md border bg-muted/20 p-3 text-xs space-y-1">
                  <p>
                    <span className="text-muted-foreground">Deployer:</span>{" "}
                    <span className="font-mono">{shortAddress(address)}</span>
                  </p>
                  <p>
                    <span className="text-muted-foreground">Balance:</span>{" "}
                    <span className="font-mono">
                      {titanBalance} {APP_CONFIG.titan.nativeToken.symbol}
                    </span>
                  </p>
                </div>
              )}

              {compiled?.constructorInputs.map((input) => {
                const key = input.name || input.type;
                return (
                  <div key={key} className="space-y-1.5">
                    <Label htmlFor={key} className="text-xs font-mono">
                      {input.name || "arg"} ({input.type})
                    </Label>
                    <Input
                      id={key}
                      value={constructorArgValues[key] ?? ""}
                      onChange={(e) =>
                        setConstructorArgValues((prev) => ({
                          ...prev,
                          [key]: e.target.value,
                        }))
                      }
                      placeholder={placeholderForType(input.type)}
                      className="font-mono text-xs"
                    />
                  </div>
                );
              })}

              <Button
                onClick={handleDeploy}
                disabled={!compiled || !walletReady || !onTitanChain || isDeploying}
                className="w-full sm:w-auto"
              >
                {isDeploying ? <Loader2 className="h-4 w-4 animate-spin" /> : <Rocket className="h-4 w-4" />}
                Deploy to Titan
              </Button>

              {!onTitanChain && walletReady && (
                <p className="text-xs text-amber-600">Switch MetaMask to {APP_CONFIG.titan.networkName} before deploying.</p>
              )}

              {deployError && <p className="text-xs text-red-600 break-all">{deployError}</p>}

              {deployResult && (
                <div className="rounded-md border border-emerald-500/30 bg-emerald-500/5 p-3 text-sm space-y-2">
                  <p className="flex items-center gap-1.5 font-medium text-emerald-700 dark:text-emerald-400">
                    <CheckCircle2 className="h-4 w-4" />
                    Contract deployed
                  </p>
                  <p className="text-xs break-all">
                    <span className="text-muted-foreground">Address:</span>{" "}
                    <span className="font-mono">{deployResult.contractAddress}</span>
                  </p>
                  <p className="text-xs break-all">
                    <span className="text-muted-foreground">Tx:</span>{" "}
                    <span className="font-mono">{deployResult.transactionHash}</span>
                  </p>
                  <div className="flex flex-wrap gap-2">
                    <Button asChild size="sm" variant="outline">
                      <Link href={`/dashboard/activity?q=${encodeURIComponent(deployResult.contractAddress)}`}>
                        View in Explorer
                        <ExternalLink className="h-3.5 w-3.5" />
                      </Link>
                    </Button>
                    <Button asChild size="sm" variant="ghost">
                      <Link href={`/dashboard/activity?q=${encodeURIComponent(deployResult.transactionHash)}`}>
                        View deployment tx
                      </Link>
                    </Button>
                  </div>
                </div>
              )}
            </div>
          </div>
        </section>
      </div>
    </div>
  );
}

function defaultArgValue(input: AbiConstructorInput): string {
  if (input.type === "bool") return "false";
  if (input.type === "string") return "Hello Titan";
  if (input.type === "address") return "0x0000000000000000000000000000000000000000";
  if (input.type.startsWith("uint") || input.type.startsWith("int")) return "0";
  return "";
}

function placeholderForType(type: string): string {
  if (type === "bool") return "true or false";
  if (type === "string") return "Hello Titan";
  if (type === "address") return "0x...";
  if (type.startsWith("uint") || type.startsWith("int")) return "0";
  return type;
}