export type TitanLogContainer = "titan-node1" | "titan-node2" | "titan-node3" | "unknown";

export type ParsedLogLine = {
  id: string;
  raw: string;
  timestamp: string | null;
  timestampMs: number | null;
  container: TitanLogContainer;
  message: string;
  level: "error" | "warn" | "info" | "debug" | "default";
};

const LOG_LINE_RE = /^\[([^\]]+)\] \[([^\]]+)\] (.*)$/;

const CONTAINER_IDS = new Set<TitanLogContainer>(["titan-node1", "titan-node2", "titan-node3"]);

export function parseLogLine(raw: string, index = 0): ParsedLogLine {
  const trimmed = raw.trim();
  const match = trimmed.match(LOG_LINE_RE);

  if (!match) {
    return {
      id: `log-${index}-${hashString(trimmed)}`,
      raw: trimmed,
      timestamp: null,
      timestampMs: null,
      container: "unknown",
      message: trimmed,
      level: detectLogLevel(trimmed),
    };
  }

  const [, timestamp, containerRaw, message] = match;
  const container = normalizeContainer(containerRaw);
  const timestampMs = Date.parse(timestamp);

  return {
    id: `log-${index}-${hashString(trimmed)}`,
    raw: trimmed,
    timestamp,
    timestampMs: Number.isNaN(timestampMs) ? null : timestampMs,
    container,
    message,
    level: detectLogLevel(message),
  };
}

export function parseLogLines(lines: string[]): ParsedLogLine[] {
  return lines.map((line, index) => parseLogLine(line, index));
}

function normalizeContainer(value: string): TitanLogContainer {
  const normalized = value.trim().toLowerCase() as TitanLogContainer;
  return CONTAINER_IDS.has(normalized) ? normalized : "unknown";
}

function detectLogLevel(message: string): ParsedLogLine["level"] {
  // AvalancheGo: [MM-DD|HH:MM:SS.mmm] LEVEL <component> …
  const avalancheMatch = message.match(
    /^\[[^\]]+\]\s+(VERBO|DEBUG|TRACE|INFO|WARN|WARNING|ERROR|FATAL|CRITICAL)\b/i,
  );
  if (avalancheMatch) {
    return mapLevelToken(avalancheMatch[1]);
  }

  // Leading bracketed level, e.g. [WARN] or [ERROR]
  const bracketLevelMatch = message.match(/^\[(WARN(?:ING)?|ERROR|ERR|FATAL|CRITICAL|INFO|DEBUG|TRACE)\]/i);
  if (bracketLevelMatch) {
    return mapLevelToken(bracketLevelMatch[1]);
  }

  const upper = message.toUpperCase();
  // Check WARN before ERROR — warn lines often mention "error" in the message body.
  if (/\b(WARN|WARNING)\b/.test(upper)) return "warn";
  if (/\b(ERROR|FATAL|PANIC|CRITICAL)\b/.test(upper)) return "error";
  if (/\b(INFO)\b/.test(upper)) return "info";
  if (/\b(DEBUG|TRACE|VERBO)\b/.test(upper)) return "debug";
  return "default";
}

function mapLevelToken(token: string): ParsedLogLine["level"] {
  const upper = token.toUpperCase();
  if (upper === "WARN" || upper === "WARNING") return "warn";
  if (["ERROR", "ERR", "FATAL", "CRITICAL", "PANIC"].includes(upper)) return "error";
  if (upper === "INFO") return "info";
  if (["DEBUG", "TRACE", "VERBO"].includes(upper)) return "debug";
  return "default";
}

export function formatLogTime(timestamp: string | null, timestampMs: number | null): string {
  if (timestampMs != null) {
    return new Date(timestampMs).toLocaleTimeString(undefined, {
      hour: "2-digit",
      minute: "2-digit",
      second: "2-digit",
      hour12: false,
    });
  }
  if (!timestamp) return "—";
  const parsed = Date.parse(timestamp);
  if (Number.isNaN(parsed)) return timestamp;
  return new Date(parsed).toLocaleTimeString(undefined, {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
    hour12: false,
  });
}

export function shortContainerLabel(container: TitanLogContainer): string {
  switch (container) {
    case "titan-node1":
      return "node1";
    case "titan-node2":
      return "node2";
    case "titan-node3":
      return "node3";
    default:
      return "unknown";
  }
}

export function containerBadgeClass(container: TitanLogContainer): string {
  switch (container) {
    case "titan-node1":
      return "border-sky-500/30 bg-sky-500/10 text-sky-700 dark:text-sky-300";
    case "titan-node2":
      return "border-amber-500/30 bg-amber-500/10 text-amber-700 dark:text-amber-300";
    case "titan-node3":
      return "border-violet-500/30 bg-violet-500/10 text-violet-700 dark:text-violet-300";
    default:
      return "border-border bg-muted text-muted-foreground";
  }
}

export function levelTextClass(level: ParsedLogLine["level"]): string {
  switch (level) {
    case "error":
      return "text-red-600 dark:text-red-400";
    case "warn":
      return "text-orange-600 dark:text-orange-400";
    case "info":
      return "text-foreground";
    case "debug":
      return "text-muted-foreground";
    default:
      return "text-foreground/90";
  }
}

export function levelRowClass(level: ParsedLogLine["level"]): string {
  switch (level) {
    case "error":
      return "bg-red-500/5";
    case "warn":
      return "bg-orange-500/5";
    default:
      return "";
  }
}

function hashString(value: string): string {
  let hash = 0;
  for (let i = 0; i < value.length; i++) {
    hash = (hash << 5) - hash + value.charCodeAt(i);
    hash |= 0;
  }
  return Math.abs(hash).toString(36);
}