# opencode microrag plugin

TypeScript plugin for [opencode](https://opencode.ai) that injects AILANG μRAG
context on `Edit`, `Write`, `Read`, and `MultiEdit` tool events.

## What it does

On every file-touching tool call, the plugin shells out to `ailang micro-rag
context`. The μRAG engine queries the local vector index for snippets relevant
to the file being edited and returns an injection text prefixed with `🧠 μRAG`.
opencode prepends this text to the next LLM prompt as `additionalContext`.

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
| `preToolUse(tool, input)` | Before each tool call | Injects μRAG context; returns `string \| undefined` |
| `postToolUse(tool, input, output)` | After tool completes | No-op (reserved for future lint hooks) |

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
required. All 18 test cases verify subprocess args, content truncation,
MultiEdit handling, error resilience, and the `AILANG_MICRORAG_ENABLED=0` guard.

## Known limits

- opencode plugin loader version ≥ 1.14.0 required (TypeScript ESM support)
- `postToolUse` is a no-op stub; lint integration analogous to `microrag_lint.sh`
  is a future enhancement
- Session isolation via `AILANG_MICRORAG_SESSION` works only within a single
  Node.js process; multi-process opencode setups would need a different IPC path
