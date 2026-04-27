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
 *   preToolUse currently returns undefined for all tools — embedding-based
 *   PreToolUse retrieval on `.ail` files was disabled per ADR-002 (queries
 *   built from file content average over too many tokens). Engine is left
 *   wired so a future hook redesign can re-enable it without code churn.
 *
 *   postToolUse fires after Edit/Write/MultiEdit and runs the targeted
 *   builtin-lint pass on `.ail` files: regex-extract builtin call sites,
 *   then emit one short signature nudge per first-use builtin. This is
 *   the working part of μRAG on the agent loop and gives opencode-driven
 *   eval runs the same harness-fairness signal that Claude Code / Gemini /
 *   Codex agents already get.
 *
 * Session isolation:
 *   AILANG_MICRORAG_SESSION is set from sessionInfo.id so the engine's
 *   dedup ledger is contiguous across hooks within one opencode session.
 */

import { spawnSync } from "child_process";

// Tool names that trigger μRAG context injection.
const WATCHED_TOOLS = new Set(["Edit", "Write", "Read", "MultiEdit"]);

// Tools whose post-state is worth scanning for first-use builtin nudges.
// Read is excluded — no new code is being introduced.
const POST_LINT_TOOLS = new Set(["Edit", "Write", "MultiEdit"]);

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

/**
 * postToolUse runs the builtin-lint nudge pass on `.ail` files. Returns the
 * joined nudge text so opencode can prepend it to the next LLM turn — same
 * harness-fairness behaviour as the bash microrag_lint.sh shims used by
 * Claude Code / Gemini / Codex. Returns undefined if no nudges fire.
 */
export function postToolUse(
  toolName: string,
  toolInput: Record<string, unknown>,
  _toolOutput: unknown
): string | undefined {
  if (process.env["AILANG_MICRORAG_ENABLED"] === "0") return undefined;

  if (!POST_LINT_TOOLS.has(toolName)) return undefined;

  const filePath = resolveFilePath(toolName, toolInput);
  if (!filePath) return undefined;

  // Builtin lint is AILANG-specific. Skip everything else.
  if (!filePath.endsWith(".ail")) return undefined;

  const content = resolveContent(toolName, toolInput);
  if (!content) return undefined;

  // Engine accepts up to ~8KB of code body for the regex pass; truncate so we
  // never feed multi-megabyte payloads.
  const code = content.slice(0, 8192);

  const args = [
    "micro-rag",
    "lint-builtin",
    "--file",
    filePath,
    "--code",
    code,
  ];
  const result = spawnSync("ailang", args, {
    encoding: "utf-8",
    timeout: 3000,
    env: { ...process.env },
  });
  if (result.status !== 0 || !result.stdout) return undefined;

  let parsed: { nudges?: { injection_text?: string }[] };
  try {
    parsed = JSON.parse(result.stdout) as typeof parsed;
  } catch {
    return undefined;
  }
  const nudges = parsed?.nudges ?? [];
  if (nudges.length === 0) return undefined;

  const joined = nudges
    .map((n) => n?.injection_text)
    .filter((t): t is string => typeof t === "string" && t.length > 0)
    .join("\n");
  return joined.length > 0 ? joined : undefined;
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
