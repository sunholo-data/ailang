# μRAG: Just-in-Time Knowledge Injection

**μRAG (micro-rag)** is a harness-agnostic engine that injects relevant AILANG
knowledge into your AI coding assistant *at the moment it edits a file*, not
just at session start. It extends the [Brain Cache](./brain-cache.md) with a
glob-routed, token-window-deduplicated lookup that runs as a tool-call hook.

The same engine drives three frontends:

- **Claude Code** — bash hooks (`PreToolUse` / `PostToolUse`)
- **Cursor / Continue / Cline** — MCP server (`ailang-microrag-mcp`)
- **Direct CLI** — `ailang micro-rag context` / `lint-builtin`

All of them shell into one Go engine, so policy (dedup window, session budget,
relevance floor, route allowlist) lives in **one place**: `~/.ailang/microrag.yaml`.

## Why μRAG

AILANG isn't in any model's training data. Brain Cache injects context at
SessionStart, but a 200K-token session quickly buries that prefix. μRAG
re-injects targeted snippets at the *moment of action* — when Claude calls
`Edit` on `foo.ail`, μRAG queries the corpus, returns the single most
relevant ≤150-token pointer, and dedups against this session's history so
the same snippet doesn't get re-injected for an hour.

Concretely, μRAG closes three gaps:

1. **Stale context.** SessionStart context drifts after the first 30K tokens.
2. **First-use builtin guesses.** Models guess `concat`/`++`/`<>` before
   importing. μRAG's `lint-builtin` returns the actual signature on the
   *first* call site this session.
3. **Breaking-change blindspots.** When a release changes operator semantics
   (e.g. `++` becoming list-only in v0.13.0), μRAG nudges the model the next
   time it touches a `.ail` file.

## Quick Start (Claude Code)

```bash
# 1. Build the corpus once (idempotent; reruns at release time)
make brain-index-syntax

# 2. Write a default config (only first time)
ailang micro-rag init

# 3. The hooks are pre-registered in .claude/settings.json
ls ~/.ailang/hooks/microrag_*.sh
# microrag_context.sh    PreToolUse(Edit|Write|Read|MultiEdit)
# microrag_lint.sh       PostToolUse(Edit|Write|MultiEdit)
```

After this, every `Edit`/`Write`/`Read` Claude makes against a `.ail` file
fires the hook. The injected snippet appears in Claude's context as a
`hookSpecificOutput.additionalContext` block, prefixed with `━━━ 🧠 μRAG`.

## Quick Start (Cursor / Continue / Cline)

These IDEs don't have bash hooks, so use the MCP server:

```bash
# Build and install the MCP frontend
make microrag-mcp-build microrag-mcp-install
which ailang-microrag-mcp
```

In the IDE's MCP config (Cursor: `~/.cursor/mcp.json`, Continue:
`~/.continue/config.json`):

```json
{
  "mcpServers": {
    "ailang-microrag": {
      "command": "ailang-microrag-mcp",
      "env": {
        "AILANG_MICRORAG_ENABLED": "1"
      }
    }
  }
}
```

The agent now has two new tools:

| Tool | Purpose |
|------|---------|
| `microrag_context_for_file` | Pre-edit retrieval of relevant syntax / change-log knowledge |
| `microrag_lint_builtin` | Post-edit first-use builtin signature nudge |

Verify the handshake:

```bash
ailang-microrag-mcp <<<'{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"smoke","version":"0"}}}'
```

You should see a `serverInfo.name = "ailang-microrag"` response.

## Configuration: `~/.ailang/microrag.yaml`

`ailang micro-rag init` writes the default. The shape:

```yaml
enabled: true

routes:
  - glob: "**/*.ail"
    kb: ailang-syntax
    max_tokens_per_injection: 150
    relevance_floor: 0.30
  - glob: "**/CHANGELOG.md"
    kb: ailang-breaking-changes
    max_tokens_per_injection: 150
    relevance_floor: 0.40
  - glob: "**/CLAUDE.md"
    kb: skip       # explicit no-op; never inject for meta files

dedup:
  windows:                     # tokens of session activity before re-inject
    ailang-breaking-changes: 15000
    ailang-syntax: 30000
    ailang-builtins: 80000
    default: 30000
  relevance_bypass:            # snippets above this score skip dedup window
    ailang-breaking-changes: 0.60
    ailang-syntax: 0.70
    default: 0.70
  wall_clock_max: 240          # seconds; stays under Anthropic's 5-min cache TTL

session_budget: 5000           # total injection tokens per session
marker_style: unicode          # unicode | ascii
```

**Why these defaults are calibrated this way:**

- **Token-window dedup, not wall-clock.** Modern context windows are 200K+;
  a 30K dedup window is 15% of that. Re-injecting more often thrashes the
  AI prompt cache. Re-injecting less often defeats the purpose.
- **Relevance bypass.** When a snippet scores extremely high — i.e., the
  model is *about* to make exactly the mistake the snippet warns about —
  bypass dedup. Contextual urgency wins over recall.
- **`wall_clock_max: 240`.** Anthropic's prompt cache has a 5-min TTL.
  μRAG's search-result cache deliberately stays under it so bursts of
  edits within ~4 minutes get the *same* `additionalContext` prefix
  (cache stays warm).

## Eval Toggle (Environment Variables)

| Variable | Effect |
|----------|--------|
| `AILANG_MICRORAG_ENABLED=0` | Master kill-switch. Engine returns `state=disabled` immediately. |
| `AILANG_MICRORAG_DRYRUN=1` | Run the full retrieval; log to ledger; suppress injection. Useful for A/B evals — captures what *would* have been injected. |
| `AILANG_MICRORAG_ROUTES=ailang-syntax,ailang-builtins` | Allowlist namespaces. Anything else routes but emits `state=on, reason=kb_not_in_allowlist`. |
| `AILANG_MICRORAG_SESSION=<id>` | Override session ID (defaults to `CLAUDE_SESSION_ID` / pid-fallback). Required when running parallel benchmarks so each agent has its own dedup ledger. |

## Eval Integration

The [eval suite](./evaluation/README.md) supports a `--microrag=on|off|auto`
flag (M-EVAL milestone). When set:

- `off` — `AILANG_MICRORAG_ENABLED=0` for the run; baseline.
- `on`  — full injection; tests "with brain assist" performance.
- `auto` — default; respect existing env.

The benchmark result schema records `microrag_state` per-trial so reports
can break down "with-vs-without" performance and surface knowledge-base
gaps (high-failure benchmarks where retrieval scored above floor → the
corpus has the answer but the model didn't apply it).

## Reindexing the Corpus (Release-Tied)

The corpus reflects the *active* AILANG prompt. Whenever a new prompt ships:

```bash
make brain-index-syntax-reset   # drops + rebuilds all three namespaces
```

This is a **required step** in the [`release-manager`](../../../.claude/skills/release-manager/SKILL.md)
skill (section 7.5) and verified by [`post-release`](../../../.claude/skills/post-release/SKILL.md)
(section 9). Skipping it leaves μRAG injecting outdated syntax — exactly
the bug the system is supposed to prevent.

## Troubleshooting

```bash
# Confirm the engine is happy
ailang micro-rag context --tool Edit --file /tmp/x.ail --content 'x ++ y' | jq .

# Inspect the session ledger
ls ~/.ailang/state/microrag/sessions/
cat ~/.ailang/state/microrag/sessions/<sid>/injections.jsonl | tail

# Inspect the corpus
ailang cache stats | grep ailang-

# Probe a specific query
ailang cache search "string interpolation" --namespace ailang-syntax --limit 3
```

If the hook output is empty:

1. `AILANG_MICRORAG_ENABLED=1` set?
2. `~/.ailang/microrag.yaml` exists and parses?
3. `ailang cache stats` shows non-zero `ailang-syntax`?
4. macOS only: `gtimeout` installed (`brew install coreutils`)? The hooks
   degrade to unbounded calls without it.

All hook failures exit `0` so the tool call is never blocked — μRAG is
**fail-quiet by design**. Check `stderr` of the IDE / Claude Code session
or the engine ledger for symptoms.

## See Also

- [Brain Cache](./brain-cache.md) — the underlying SQLite + embeddings store
- [Design doc: M-BRAIN-MICRORAG](https://github.com/sunholo-data/ailang/blob/dev/design_docs/planned/v0_15_0/m-brain-microrag.md)
- [Evaluation guide](./evaluation/README.md) — how `--microrag` flag fits into eval runs
