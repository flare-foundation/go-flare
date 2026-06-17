"use client";

import { useEffect, useMemo, useState } from "react";

import Link from "next/link";

import {
  AlertCircle,
  CheckCircle2,
  ChevronRight,
  Circle,
  Code2,
  ExternalLink,
  FileCode2,
  FlaskConical,
  History,
  Loader2,
  Play,
  Plus,
  Rocket,
  Trash2,
  Wallet,
} from "lucide-react";
import type { Abi, AbiParameter } from "viem";

import { ContractPlayground } from "@/app/(main)/dashboard/contracts/_components/contract-playground";
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
import { Tabs, TabsContent } from "@/components/ui/tabs";
import { Textarea } from "@/components/ui/textarea";
import { APP_CONFIG } from "@/config/app-config";
import type { AbiConstructorInput, CompiledContract } from "@/lib/titan/compile-contract";
import {
  CONTRACT_TEMPLATES,
  DEFAULT_TEMPLATE_ID,
} from "@/lib/titan/contract-templates";
import { isSandboxContract, templateIdForContractName } from "@/lib/titan/contract-sandbox";
import {
  deployContract,
  getConstructorAbi,
  parseConstructorArgValue,
} from "@/lib/titan/deploy-contract";
import {
  addDeployedContract,
  type DeployedContractRecord,
  isContractAddress,
  loadDeployedContracts,
  removeDeployedContract,
} from "@/lib/titan/deployed-contracts-storage";
import { shortAddress } from "@/lib/titan/format";
import { parseWalletError } from "@/lib/titan/wallet-errors";
import { cn } from "@/lib/utils";
import { isOnTitanChain, isWalletConnected, useWalletStore } from "@/stores/wallet/wallet-store";

type DeployResult = {
  transactionHash: string;
  contractAddress: string;
};

type StudioStep = "edit" | "compile" | "deploy" | "deployed";

const STEPS: Array<{ id: StudioStep; label: string; step: number }> = [
  { id: "edit", label: "Edit", step: 1 },
  { id: "compile", label: "Compile", step: 2 },
  { id: "deploy", label: "Deploy", step: 3 },
  { id: "deployed", label: "Deployed", step: 4 },
];

export function ContractStudio() {
  const defaultTemplate = CONTRACT_TEMPLATES.find((t) => t.id === DEFAULT_TEMPLATE_ID) ?? CONTRACT_TEMPLATES[0];

  const [activeStep, setActiveStep] = useState<StudioStep>("edit");
  const [templateId, setTemplateId] = useState(defaultTemplate.id);
  const [source, setSource] = useState(defaultTemplate.source);
  const [compiled, setCompiled] = useState<CompiledContract | null>(null);
  const [compileErrors, setCompileErrors] = useState<string[]>([]);
  const [hasCompiledOnce, setHasCompiledOnce] = useState(false);
  const [constructorArgValues, setConstructorArgValues] = useState<Record<string, string>>({});
  const [isCompiling, setIsCompiling] = useState(false);
  const [isDeploying, setIsDeploying] = useState(false);
  const [deployError, setDeployError] = useState("");
  const [deployResult, setDeployResult] = useState<DeployResult | null>(null);
  const [deployedContracts, setDeployedContracts] = useState<DeployedContractRecord[]>([]);
  const [trackAddress, setTrackAddress] = useState("");
  const [trackError, setTrackError] = useState("");
  const [isTracking, setIsTracking] = useState(false);
  const [activePlayground, setActivePlayground] = useState<string | null>(null);

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

  const stepStatus = useMemo(() => {
    const editDone = source.trim().length > 0;
    const compileDone = Boolean(compiled);
    const compileFailed = hasCompiledOnce && compileErrors.length > 0;
    const deployDone = Boolean(deployResult) || deployedContracts.length > 0;

    return {
      edit: editDone ? "done" : "current",
      compile: compileFailed ? "error" : compileDone ? "done" : hasCompiledOnce ? "current" : editDone ? "ready" : "locked",
      deploy: compileDone ? (deployDone ? "done" : "current") : "locked",
      deployed: deployDone ? "done" : compileDone ? "ready" : "locked",
    } as Record<StudioStep, "locked" | "ready" | "current" | "done" | "error">;
  }, [compiled, compileErrors.length, deployResult, deployedContracts.length, hasCompiledOnce, source]);

  useEffect(() => {
    setDeployedContracts(loadDeployedContracts());
  }, []);

  function rememberDeployment(result: DeployResult, contractName: string) {
    const sandboxTemplateId = templateIdForContractName(contractName);
    setDeployedContracts(
      addDeployedContract({
        contractName,
        contractAddress: result.contractAddress,
        transactionHash: result.transactionHash,
        deployer: address,
        templateId: sandboxTemplateId,
      }),
    );
    if (sandboxTemplateId) {
      setActivePlayground(result.contractAddress);
    }
    setActiveStep("deployed");
  }

  async function handleTrackAddress() {
    const candidate = trackAddress.trim();
    setTrackError("");

    if (!/^0x[0-9a-fA-F]{40}$/.test(candidate)) {
      setTrackError("Enter a valid 0x contract address.");
      return;
    }

    setIsTracking(true);
    try {
      const exists = await isContractAddress(candidate);
      if (!exists) {
        setTrackError("No contract bytecode at this address on Titan.");
        return;
      }

      setDeployedContracts(
        addDeployedContract({
          contractName: "Tracked contract",
          contractAddress: candidate,
          transactionHash: null,
          deployer: null,
        }),
      );
      setTrackAddress("");
    } catch (error) {
      setTrackError(error instanceof Error ? error.message : "Could not verify contract address.");
    } finally {
      setIsTracking(false);
    }
  }

  function handleRemoveContract(contractAddress: string) {
    setDeployedContracts(removeDeployedContract(contractAddress));
  }

  function applyTemplate(nextTemplateId: string) {
    const template = CONTRACT_TEMPLATES.find((t) => t.id === nextTemplateId);
    if (!template) return;
    setTemplateId(template.id);
    setSource(template.source);
    setCompiled(null);
    setCompileErrors([]);
    setHasCompiledOnce(false);
    setConstructorArgValues(template.constructorDefaults ?? {});
    setDeployError("");
    setDeployResult(null);
    setActiveStep("edit");
  }

  function handleSourceChange(value: string) {
    setSource(value);
    if (compiled || compileErrors.length > 0) {
      setCompiled(null);
      setCompileErrors([]);
      setDeployResult(null);
      setHasCompiledOnce(false);
    }
  }

  async function handleCompile(goToStep: StudioStep = "compile") {
    setIsCompiling(true);
    setCompileErrors([]);
    setCompiled(null);
    setDeployError("");
    setDeployResult(null);
    setHasCompiledOnce(true);
    setActiveStep(goToStep === "compile" ? "compile" : goToStep);

    try {
      const response = await fetch("/api/titan/compile", {
        method: "POST",
        headers: { "content-type": "application/json" },
        body: JSON.stringify({
          source,
          fileName: selectedTemplate.fileName ?? `${selectedTemplate.name}.sol`,
        }),
      });

      const result = (await response.json()) as
        | { ok: true; contract: CompiledContract }
        | { ok: false; errors: string[] };

      if (!result.ok) {
        setCompileErrors(result.errors);
        setActiveStep("compile");
        return;
      }

      setCompiled(result.contract);
      const initialArgs: Record<string, string> = {};
      for (const input of result.contract.constructorInputs) {
        const key = input.name || input.type;
        initialArgs[key] =
          selectedTemplate.constructorDefaults?.[key] ?? defaultArgValue(input);
      }
      setConstructorArgValues(initialArgs);
      setActiveStep("deploy");
    } catch (error) {
      setCompileErrors([error instanceof Error ? error.message : "Compilation request failed."]);
      setActiveStep("compile");
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
      rememberDeployment(result, compiled.contractName);
    } catch (error) {
      setDeployError(parseWalletError(error, "Deployment failed."));
    } finally {
      setIsDeploying(false);
    }
  }

  function canOpenStep(step: StudioStep): boolean {
    if (step === "edit") return true;
    if (step === "compile") return source.trim().length > 0;
    if (step === "deploy") return Boolean(compiled);
    return true;
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
            Edit → compile → deploy → interact on {APP_CONFIG.titan.networkName}.
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

      <Tabs
        value={activeStep}
        onValueChange={(value) => {
          const step = value as StudioStep;
          if (canOpenStep(step)) setActiveStep(step);
        }}
        className="gap-4"
      >
        <div className="rounded-lg border bg-muted/15 px-3 py-2.5">
          <div className="flex w-full items-center gap-1.5 sm:gap-2">
            {STEPS.map((step, index) => {
              const status = stepStatus[step.id];
              const isActive = activeStep === step.id;
              const clickable = canOpenStep(step.id);

              return (
                <div key={step.id} className="flex min-w-0 flex-1 items-center gap-1.5 sm:gap-2">
                  <button
                    type="button"
                    disabled={!clickable}
                    onClick={() => clickable && setActiveStep(step.id)}
                    className={cn(
                      "group flex h-8 w-full min-w-0 items-center justify-center gap-1.5 rounded-full border px-2.5 text-xs font-medium transition-all sm:h-9 sm:gap-2 sm:px-3",
                      isActive && "border-primary bg-primary/10 text-foreground shadow-sm",
                      !isActive &&
                        status === "done" &&
                        clickable &&
                        "border-emerald-500/40 bg-emerald-500/5 text-emerald-800 dark:text-emerald-300 hover:bg-emerald-500/10",
                      !isActive &&
                        status === "error" &&
                        "border-red-500/40 bg-red-500/5 text-red-600",
                      !isActive &&
                        clickable &&
                        status !== "done" &&
                        status !== "error" &&
                        "border-border bg-background text-muted-foreground hover:border-primary/40 hover:text-foreground",
                      !clickable && "cursor-not-allowed border-transparent bg-muted/40 text-muted-foreground opacity-60",
                    )}
                  >
                    <span
                      className={cn(
                        "flex h-5 w-5 shrink-0 items-center justify-center rounded-full border text-[10px] font-semibold sm:h-6 sm:w-6 sm:text-[11px]",
                        isActive && "border-primary bg-primary text-primary-foreground",
                        !isActive &&
                          status === "done" &&
                          "border-emerald-500/60 bg-emerald-500/15 text-emerald-700 dark:text-emerald-300",
                        !isActive &&
                          status === "error" &&
                          "border-red-500/60 bg-red-500/15 text-red-600",
                        !isActive &&
                          status !== "done" &&
                          status !== "error" &&
                          "border-muted-foreground/35 bg-muted/50 text-muted-foreground group-hover:border-primary/50",
                      )}
                    >
                      <RoundStepGlyph status={status} step={step.step} isActive={isActive} />
                    </span>
                    <span className="truncate">{step.label}</span>
                    {step.id === "deployed" && deployedContracts.length > 0 && (
                      <Badge variant="secondary" className="h-4 shrink-0 px-1 text-[10px]">
                        {deployedContracts.length}
                      </Badge>
                    )}
                  </button>
                  {index < STEPS.length - 1 && (
                    <div
                      className={cn(
                        "h-px w-2 shrink-0 rounded-full sm:w-3",
                        stepStatus[step.id] === "done" ? "bg-emerald-500/40" : "bg-border",
                      )}
                    />
                  )}
                </div>
              );
            })}
          </div>
        </div>

        <TabsContent value="edit" className="mt-0">
          <section className="flex flex-col gap-3 rounded-lg border">
            <div className="flex flex-wrap items-center justify-between gap-2 border-b bg-muted/30 px-4 py-3">
              <div className="flex items-center gap-2 text-sm font-semibold">
                <FileCode2 className="h-4 w-4" />
                1. Edit source
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
                onChange={(e) => handleSourceChange(e.target.value)}
                className="min-h-105 font-mono text-xs leading-relaxed"
                spellCheck={false}
              />
            </div>

            <div className="flex flex-wrap gap-2 border-t px-4 py-3">
              <Button onClick={() => void handleCompile("compile")} disabled={isCompiling || !source.trim()}>
                {isCompiling ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />}
                Compile &amp; check
              </Button>
              <Button variant="outline" onClick={() => applyTemplate(templateId)} disabled={isCompiling}>
                Reset template
              </Button>
            </div>
          </section>
        </TabsContent>

        <TabsContent value="compile" className="mt-0">
          <section className="rounded-lg border">
            <div className="border-b bg-muted/30 px-4 py-3 text-sm font-semibold">2. Compiler</div>
            <div className="space-y-4 px-4 py-4 text-sm">
              {isCompiling && (
                <div className="flex items-center gap-2 text-muted-foreground">
                  <Loader2 className="h-4 w-4 animate-spin" />
                  Compiling Solidity for Paris EVM…
                </div>
              )}

              {!isCompiling && !hasCompiledOnce && (
                <p className="text-muted-foreground">
                  Run compile from the Edit step to check your contract for errors.
                </p>
              )}

              {!isCompiling && compileErrors.length > 0 && (
                <div className="rounded-md border border-red-500/30 bg-red-500/5 p-3">
                  <p className="font-medium text-red-600 mb-2 flex items-center gap-1.5">
                    <AlertCircle className="h-4 w-4" />
                    Compilation failed
                  </p>
                  <pre className="whitespace-pre-wrap text-xs text-red-600/90 font-mono">{compileErrors.join("\n\n")}</pre>
                  <Button size="sm" variant="outline" className="mt-3" onClick={() => setActiveStep("edit")}>
                    Back to edit
                  </Button>
                </div>
              )}

              {!isCompiling && compiled && (
                <div className="rounded-md border border-emerald-500/30 bg-emerald-500/5 p-3 space-y-2">
                  <p className="font-medium text-emerald-700 dark:text-emerald-400 flex items-center gap-1.5">
                    <CheckCircle2 className="h-4 w-4" />
                    Compilation successful
                  </p>
                  <p className="text-xs text-muted-foreground">
                    <span className="font-mono font-medium text-foreground">{compiled.contractName}</span> is ready for
                    deployment.
                  </p>
                  <Button size="sm" onClick={() => setActiveStep("deploy")}>
                    Continue to deploy
                    <ChevronRight className="h-4 w-4" />
                  </Button>
                </div>
              )}

              <div className="flex flex-wrap gap-2 border-t pt-4">
                <Button variant="outline" size="sm" onClick={() => setActiveStep("edit")}>
                  Back to edit
                </Button>
                <Button size="sm" onClick={() => void handleCompile("compile")} disabled={isCompiling || !source.trim()}>
                  {isCompiling ? <Loader2 className="h-4 w-4 animate-spin" /> : <Play className="h-4 w-4" />}
                  Recompile
                </Button>
              </div>
            </div>
          </section>
        </TabsContent>

        <TabsContent value="deploy" className="mt-0">
          <section className="flex flex-col gap-4">
            <div className="rounded-lg border">
              <div className="border-b bg-muted/30 px-4 py-3 text-sm font-semibold">3. Build output</div>
              <div className="space-y-3 px-4 py-4 text-sm">
                {!compiled ? (
                  <p className="text-muted-foreground">Compile successfully first to unlock deployment.</p>
                ) : (
                  <div className="space-y-3">
                    <div className="flex flex-wrap items-center gap-2">
                      <Badge>{compiled.contractName}</Badge>
                      <Badge variant="outline">Paris EVM</Badge>
                      <span className="text-xs text-muted-foreground">
                        Bytecode: {(compiled.bytecode.length / 2).toLocaleString()} bytes
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
                Deploy to Titan
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
                  onClick={() => void handleDeploy()}
                  disabled={!compiled || !walletReady || !onTitanChain || isDeploying}
                  className="w-full sm:w-auto"
                >
                  {isDeploying ? <Loader2 className="h-4 w-4 animate-spin" /> : <Rocket className="h-4 w-4" />}
                  Deploy to Titan
                </Button>

                {!onTitanChain && walletReady && (
                  <p className="text-xs text-amber-600">
                    Switch MetaMask to {APP_CONFIG.titan.networkName} before deploying.
                  </p>
                )}

                {walletReady && (titanBalance === "0" || titanBalance === "—") && (
                  <p className="text-xs text-amber-600">
                    Your wallet has no TITAN for gas. Import a prefunded genesis account into MetaMask to deploy on
                    local UAT.
                  </p>
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
                    <Button size="sm" onClick={() => setActiveStep("deployed")}>
                      View in Deployed tab
                      <ChevronRight className="h-4 w-4" />
                    </Button>
                  </div>
                )}
              </div>
            </div>
          </section>
        </TabsContent>

        <TabsContent value="deployed" className="mt-0">
          <section className="rounded-lg border">
            <div className="flex flex-wrap items-center justify-between gap-2 border-b bg-muted/30 px-4 py-3">
              <div className="flex items-center gap-2 text-sm font-semibold">
                <History className="h-4 w-4" />
                4. Deployed contracts
              </div>
              <span className="text-xs text-muted-foreground">Saved in this browser · sandbox for templates</span>
            </div>

            <div className="space-y-4 px-4 py-4">
              <div className="flex flex-col gap-2 sm:flex-row sm:items-end">
                <div className="flex-1 space-y-1.5">
                  <Label htmlFor="track-contract" className="text-xs">
                    Track an existing contract address
                  </Label>
                  <Input
                    id="track-contract"
                    value={trackAddress}
                    onChange={(e) => setTrackAddress(e.target.value)}
                    placeholder="0x…"
                    className="font-mono text-xs"
                  />
                </div>
                <Button
                  variant="outline"
                  onClick={() => void handleTrackAddress()}
                  disabled={isTracking || !trackAddress.trim()}
                >
                  {isTracking ? <Loader2 className="h-4 w-4 animate-spin" /> : <Plus className="h-4 w-4" />}
                  Add
                </Button>
              </div>
              {trackError && <p className="text-xs text-red-600 break-all">{trackError}</p>}

              {deployedContracts.length === 0 ? (
                <p className="text-sm text-muted-foreground">
                  No contracts yet. Complete the deploy step or paste an address above.
                </p>
              ) : (
                <div className="divide-y rounded-md border">
                  {deployedContracts.map((record) => {
                    const sandboxReady = isSandboxContract(record);
                    const playgroundOpen = activePlayground === record.contractAddress;

                    return (
                      <div key={record.id}>
                        <div className="flex flex-col gap-2 px-3 py-3 text-sm sm:flex-row sm:items-center sm:justify-between">
                          <div className="min-w-0 space-y-1">
                            <div className="flex flex-wrap items-center gap-2">
                              <span className="font-medium">{record.contractName}</span>
                              {sandboxReady && (
                                <Badge variant="outline" className="text-[10px]">
                                  Sandbox
                                </Badge>
                              )}
                              <span className="font-mono text-xs text-muted-foreground">
                                {shortAddress(record.contractAddress)}
                              </span>
                            </div>
                            <p className="font-mono text-xs break-all text-muted-foreground">{record.contractAddress}</p>
                            <p className="text-[10px] text-muted-foreground">
                              {new Date(record.deployedAt).toLocaleString()}
                              {record.deployer ? ` · deployer ${shortAddress(record.deployer)}` : ""}
                            </p>
                          </div>
                          <div className="flex flex-wrap items-center gap-2 shrink-0">
                            {sandboxReady && (
                              <Button
                                size="sm"
                                variant={playgroundOpen ? "default" : "outline"}
                                onClick={() =>
                                  setActivePlayground((current) =>
                                    current === record.contractAddress ? null : record.contractAddress,
                                  )
                                }
                              >
                                <FlaskConical className="h-3.5 w-3.5" />
                                {playgroundOpen ? "Hide sandbox" : "Try it"}
                              </Button>
                            )}
                            <Button asChild size="sm" variant="outline">
                              <Link href={`/dashboard/activity?q=${encodeURIComponent(record.contractAddress)}`}>
                                Explorer
                                <ExternalLink className="h-3.5 w-3.5" />
                              </Link>
                            </Button>
                            {record.transactionHash && (
                              <Button asChild size="sm" variant="ghost">
                                <Link href={`/dashboard/activity?q=${encodeURIComponent(record.transactionHash)}`}>
                                  Deploy tx
                                </Link>
                              </Button>
                            )}
                            <Button
                              size="icon"
                              variant="ghost"
                              onClick={() => {
                                if (playgroundOpen) setActivePlayground(null);
                                handleRemoveContract(record.contractAddress);
                              }}
                              title="Remove from list"
                            >
                              <Trash2 className="h-3.5 w-3.5" />
                            </Button>
                          </div>
                        </div>
                        {playgroundOpen && (
                          <div className="border-t bg-muted/10 px-3 py-3">
                            <ContractPlayground record={record} />
                          </div>
                        )}
                      </div>
                    );
                  })}
                </div>
              )}
            </div>
          </section>
        </TabsContent>
      </Tabs>
    </div>
  );
}

function RoundStepGlyph({
  status,
  step,
  isActive,
}: {
  status: "locked" | "ready" | "current" | "done" | "error";
  step: number;
  isActive: boolean;
}) {
  if (status === "done" && !isActive) {
    return <CheckCircle2 className="h-3 w-3 sm:h-3.5 sm:w-3.5" />;
  }
  if (status === "error") {
    return <AlertCircle className="h-3 w-3 sm:h-3.5 sm:w-3.5" />;
  }
  if (status === "locked" && !isActive) {
    return <Circle className="h-2.5 w-2.5 sm:h-3 sm:w-3" />;
  }
  return <span>{step}</span>;
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