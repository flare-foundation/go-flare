import { execFile, spawn } from "node:child_process";
import { promisify } from "node:util";

import { type NextRequest, NextResponse } from "next/server";

const execFileAsync = promisify(execFile);

export const dynamic = "force-dynamic";

const ALLOWED = ["titan-node1", "titan-node2", "titan-node3"];

export async function GET(request: NextRequest) {
  const { searchParams } = request.nextUrl;

  // Container info / status list mode
  if (searchParams.get("info") === "1") {
    try {
      const { stdout } = await execFileAsync("docker", [
        "ps",
        "-a",
        "--filter",
        "name=titan-node",
        "--format",
        "{{.Names}}\t{{.Status}}\t{{.Ports}}\t{{.Image}}",
      ]);
      const containers = stdout
        .trim()
        .split("\n")
        .filter(Boolean)
        .map((line) => {
          const [name, status, ports, image] = line.split("\t");
          return { name, status, ports, image };
        });
      return NextResponse.json({ containers });
    } catch (e) {
      return NextResponse.json({ containers: [], error: String(e) });
    }
  }

  // Live log streaming via Server-Sent Events
  const container = searchParams.get("container") ?? "all";
  const isCombined = container === "all";
  if (!isCombined && !ALLOWED.includes(container)) {
    return NextResponse.json({ error: "invalid container" }, { status: 400 });
  }

  const encoder = new TextEncoder();
  let procs: Array<ReturnType<typeof spawn>> = [];
  let isClosed = false;
  let cleanup = () => {};

  const closeSafely = (controller: ReadableStreamDefaultController<Uint8Array>) => {
    if (isClosed) {
      return;
    }
    isClosed = true;
    // Next.js/runtime may already own stream shutdown lifecycle.
    // Avoid calling controller.close() directly to prevent close races.
    void controller;
  };

  const killProcs = () => {
    for (const proc of procs) {
      if (!proc.killed) {
        try {
          proc.kill();
        } catch {}
      }
    }
    procs = [];
  };

  const stream = new ReadableStream({
    start(controller) {
      const containers = isCombined ? ALLOWED : [container];
      const listeners: Array<{
        proc: ReturnType<typeof spawn>;
        send: (data: Buffer) => void;
        onClose: () => void;
        onError: () => void;
      }> = [];

      let activeProcCount = containers.length;

      for (const currentContainer of containers) {
        const proc = spawn("docker", [
          "logs",
          "--follow",
          "--tail",
          "200",
          currentContainer,
        ]);
        procs.push(proc);

        if (!proc.stdout || !proc.stderr) {
          activeProcCount -= 1;
          continue;
        }

        const send = (data: Buffer) => {
          if (isClosed || controller.desiredSize === null) {
            isClosed = true;
            return;
          }
          for (const line of data.toString().split("\n")) {
            if (!line.trim()) {
              continue;
            }
            const stampedLine = `[${new Date().toISOString()}] [${currentContainer}] ${line}`;
            try {
              controller.enqueue(encoder.encode(`data: ${stampedLine}\n\n`));
            } catch {
              cleanup();
              closeSafely(controller);
              killProcs();
              return;
            }
          }
        };

        const onClose = () => {
          activeProcCount -= 1;
          if (activeProcCount <= 0) {
            cleanup();
            closeSafely(controller);
          }
        };

        const onError = () => {
          activeProcCount -= 1;
          if (activeProcCount <= 0) {
            cleanup();
            closeSafely(controller);
          }
        };

        proc.stdout.on("data", send);
        proc.stderr.on("data", send);
        proc.on("close", onClose);
        proc.on("error", onError);

        listeners.push({ proc, send, onClose, onError });
      }

      const onAbort = () => {
        cleanup();
        closeSafely(controller);
        killProcs();
      };

      cleanup = () => {
        for (const { proc, send, onClose, onError } of listeners) {
          proc.stdout?.off("data", send);
          proc.stderr?.off("data", send);
          proc.off("close", onClose);
          proc.off("error", onError);
        }
        request.signal.removeEventListener("abort", onAbort);
      };

      // In case the incoming request is aborted before stream cancellation is propagated.
      request.signal.addEventListener("abort", onAbort);

      if (request.signal.aborted) {
        onAbort();
      }
    },
    cancel() {
      isClosed = true;
      cleanup();
      killProcs();
    },
  });

  return new Response(stream, {
    headers: {
      "Content-Type": "text/event-stream",
      "Cache-Control": "no-cache",
      Connection: "keep-alive",
    },
  });
}
