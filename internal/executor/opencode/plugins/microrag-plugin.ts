/**
 * microrag-plugin.ts — opencode plugin that injects μRAG context on tool events.
 *
 * Install:
 *   Copy to ~/.config/opencode/plugins/microrag-plugin.ts (or symlink).
 *   Verify: opencode plugins list
 *
 * Toggle:
 *   AILANG_MICRORAG_ENABLED=1   (default: enabled)
 *   AILANG_MICRORAG_ENABLED=0   (disable without uninstalling)
 *
 * How it works:
 *   preToolUse fires before Edit/Write/Read/MultiEdit tool calls.
 *   It shells out to `ailang micro-rag context`, which queries the μRAG engine
 *   and returns a snippet. That snippet is returned from preToolUse as the
 *   additionalContext string, which opencode prepends to the next LLM prompt.
 *
 * Session isolation:
 *   AILANG_MICRORAG_SESSION is set from sessionInfo.id so the engine's
 *   dedup ledger is contiguous across hooks within one opencode session.
 */

import { spawnSync } from "child_process";

// Tool names that trigger μRAG context injection.
const WATCHED_TOOLS = new Set(["Edit", "Write", "Read", "MultiEdit"]);

// File paths / extensions that are never worth indexing.
const SKIP_PATH_RE =
  /\/(node_modules|\.git|vendor|__pycache__)\//;
const SKIP_EXT_RE =
  /\.(png|jpe?g|gif|ico|svg|webp|pdf|wasm|bin|exe|dylib|so|zip|tar|gz|bz2)$/i;

// Module-level session id, populated once from sessionStart.
let sessionId = "";

export function sessionStart(info: { id?: string }): void {
  if (info?.id) {
    sessionId = info.id;
    // Propagate for any child processes spawned in this Node instance.
    process.env["AILANG_MICRORAG_SESSION"] = sessionId;
  }
}

export function preToolUse(
  toolName: string,
  toolInput: Record<string, unknown>
): string | undefined {
  if (process.env["AILANG_MICRORAG_ENABLED"] === "0") return undefined;

  if (!WATCHED_TOOLS.has(toolName)) return undefined;

  const filePath = resolveFilePath(toolName, toolInput);
  if (!filePath) return undefined;

  if (SKIP_PATH_RE.test(filePath) || SKIP_EXT_RE.test(filePath)) {
    return undefined;
  }

  const content = resolveContent(toolName, toolInput);

  const args = ["micro-rag", "context", "--tool", toolName, "--file", filePath];
  if (content) {
    args.push("--content", content.slice(0, 4096));
  }

  const result = spawnSync("ailang", args, {
    encoding: "utf-8",
    timeout: 3000,
    env: { ...process.env },
  });

  if (result.status !== 0 || !result.stdout) return undefined;

  let parsed: { injection?: { injection_text?: string } };
  try {
    parsed = JSON.parse(result.stdout) as typeof parsed;
  } catch {
    return undefined;
  }

  const injectionText = parsed?.injection?.injection_text;
  if (!injectionText) return undefined;

  // 🧠 μRAG marker is embedded in the engine's injection_text; transcript
  // grepping works across all harnesses.
  return injectionText;
}

export function postToolUse(
  _toolName: string,
  _toolInput: Record<string, unknown>,
  _toolOutput: unknown
): void {
  // Reserved for future lint/quality hooks analogous to microrag_lint.sh.
}

// --- helpers ---

function resolveFilePath(
  toolName: string,
  toolInput: Record<string, unknown>
): string | undefined {
  if (typeof toolInput["file_path"] === "string") {
    return toolInput["file_path"] || undefined;
  }
  // MultiEdit carries edits[] with per-edit file_path.
  if (toolName === "MultiEdit") {
    const edits = toolInput["edits"];
    if (Array.isArray(edits) && edits.length > 0) {
      const first = edits[0] as Record<string, unknown>;
      if (typeof first["file_path"] === "string") return first["file_path"];
    }
  }
  return undefined;
}

function resolveContent(
  toolName: string,
  toolInput: Record<string, unknown>
): string | undefined {
  switch (toolName) {
    case "Write":
      return typeof toolInput["content"] === "string"
        ? toolInput["content"]
        : undefined;
    case "Edit":
      return typeof toolInput["new_string"] === "string"
        ? toolInput["new_string"]
        : undefined;
    case "MultiEdit": {
      const edits = toolInput["edits"];
      if (!Array.isArray(edits)) return undefined;
      return edits
        .map((e) => {
          const edit = e as Record<string, unknown>;
          return typeof edit["new_string"] === "string" ? edit["new_string"] : "";
        })
        .filter(Boolean)
        .join("\n");
    }
    case "Read":
    default:
      return undefined;
  }
}
