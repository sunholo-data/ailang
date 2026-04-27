# opencode microrag plugin

TypeScript plugin for [opencode](https://opencode.ai) that injects AILANG μRAG
context on `Edit`, `Write`, `Read`, and `MultiEdit` tool events.

## What it does

Two μRAG hooks are wired:

- **`preToolUse`** — calls the engine's `micro-rag context`, but the engine
  returns `kb_skip` for `.ail` files (per ADR-002: embedding-based PreToolUse
  retrieval on file content averages over too many tokens; cosine similarity
  dilutes single-character anti-patterns like `++`). The plugin stays wired so
  re-enabling the route in `~/.ailang/microrag.yaml` is the only change a
  future redesign would need.
- **`postToolUse`** — fires after `Edit`, `Write`, and `MultiEdit` on `.ail`
  files. Calls `ailang micro-rag lint-builtin` which regex-scans the just-written
  code for first-use builtin invocations and emits one short signature nudge
  per first-use. The plugin returns the joined nudge text so opencode can
  prepend it to the next LLM turn — same harness-fairness behaviour the bash
  `microrag_lint.sh` shims give Claude Code / Gemini / Codex agents.

Same engine as the Claude Code / Gemini CLI / Codex CLI shims — only the hook
interface differs (TypeScript module exports vs bash stdin/stdout).

## Install

**1. Copy (or symlink) the plugin:**

```bash
mkdir -p ~/.config/opencode/plugins
cp microrag-plugin.ts ~/.config/opencode/plugins/
```

**2. Register in `~/.config/opencode/opencode.jsonc`:**

```jsonc
{
  "plugins": [
    "~/.config/opencode/plugins/microrag-plugin.ts"
  ]
}
```

**3. Verify:**

```bash
opencode plugins list
# Should show: ailang-microrag-opencode-plugin  (enabled)
```

## Toggle

```bash
# Disable (without uninstalling):
export AILANG_MICRORAG_ENABLED=0

# Re-enable (default):
export AILANG_MICRORAG_ENABLED=1
```

The plugin exits immediately when `AILANG_MICRORAG_ENABLED=0` — no subprocess
spawned, no latency impact.

## Hook exports

| Export | Fires | Purpose |
|---|---|---|
| `sessionStart(info)` | Session init | Sets `AILANG_MICRORAG_SESSION` from opencode session ID |
| `preToolUse(tool, input)` | Before each tool call | Calls `micro-rag context`; engine returns `kb_skip` for `.ail` files (ADR-002) |
| `postToolUse(tool, input, output)` | After Edit/Write/MultiEdit on `.ail` files | Returns first-use builtin signature nudges |

## Skipped files

The plugin never queries for:
- Paths inside `node_modules/`, `.git/`, `vendor/`, `__pycache__/`
- Binary extensions: `.png .jpg .gif .ico .svg .webp .pdf .wasm .bin .exe .dylib .so .zip .tar .gz .bz2`

## Testing

```bash
cd internal/executor/opencode/plugins
npm install
npx vitest run
```

Tests use `vitest` with `vi.mock("child_process")` — no live `ailang` binary
required. All 26 test cases verify subprocess args, content truncation,
MultiEdit handling, error resilience, the `AILANG_MICRORAG_ENABLED=0` guard,
and `postToolUse` builtin-lint behaviour.

## Known limits

- opencode plugin loader version ≥ 1.14.0 required (TypeScript ESM support)
- `preToolUse` is intentionally a no-op stub — see ADR-002 for the reasoning
  behind disabling embedding-based PreToolUse retrieval on `.ail` file content.
  PostToolUse builtin-lint (an unrelated targeted code path) provides the
  on-agent signal.
- Session isolation via `AILANG_MICRORAG_SESSION` works only within a single
  Node.js process; multi-process opencode setups would need a different IPC path
