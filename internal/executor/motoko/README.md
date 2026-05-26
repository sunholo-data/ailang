# motoko Executor

AILANG eval-harness adapter for [motoko_agent](https://github.com/arniwesth/motoko_agent) — the first production-scale coding agent built on AILANG itself (~5,200 LOC of `.ail` modules + 9 published `motoko-ext-*` extension packages, wrapped in a bun TUI).

Sprint: [M-MOTOKO-EXECUTOR-ADAPTER](../../../design_docs/planned/v0_18_0/m-motoko-executor-adapter.md) (target v0.18.0)

## What this adapter does

Spawns `motoko "<task>"` as a subprocess with `WORKDIR` / `MODEL` / `MOTOKO_CONFIG` / `MOTOKO_SESSION_ID` env vars. After the subprocess exits, locates the session JSONL motoko wrote at `${WORKDIR}/.motoko/logfile/<session_id>.jsonl`, parses it via [`parser.go::parseSessionJSONL`](parser.go), and folds it into the standard `executor.Result` shape.

## CLI flags / invocation

```bash
motoko "<directive>"           # task as positional arg
# Env var inputs:
#   WORKDIR=/path/to/workspace          # working directory motoko writes JSONL into
#   MODEL=openrouter/anthropic/claude-haiku-4-5   # model id (always openrouter/* for motoko)
#   MOTOKO_CONFIG=dogfood                          # profile name (matches a directory under .motoko/config/)
#   MOTOKO_SESSION_ID=session_<task_id>            # the adapter sets this so it can find the JSONL
```

The adapter never invokes any motoko sub-command beyond `--version` / `--help` (used by HealthCheck).

## Auth

- **Required**: `OPENROUTER_API_KEY` — motoko routes ALL models via OpenRouter, so this is the canonical auth surface.
- **NOT bound in cloud Job** (per [EXECUTOR_SHAPE.md §8](../../../docs/internal/EXECUTOR_SHAPE.md) cost-control rule): `ANTHROPIC_API_KEY`. Anthropic API-key billing is pay-per-token; the cloud Job binds only OpenRouter + OpenAI + Gemini keys, matching the Pi-precedent. `motoko-claude-*` models stay LOCAL-only for cost-control.

For local dev, `OPENROUTER_API_KEY` in your shell environment is sufficient. The adapter does not manage credentials — it inherits the parent process's environment via `executor.BuildEnvironment`.

## Event schema (motoko session JSONL v1)

motoko emits a structured JSONL stream documented in motoko_agent's [M-MOTOKO-EVAL-INSTRUMENTATION design doc](https://github.com/sunholo-data/motoko_agent/blob/motoko-dx-compaction-pending/design_docs/implemented/motoko_agent/m-motoko-eval-instrumentation.md). Every event carries:

- `schema_version: "1"` — forward-compat marker
- `session_id: "session_<id>"` — adapter sets this via env so it can find the file
- `type: "<event-type>"`

Event types parsed by this adapter:

| Type | Maps to `Result` field |
|---|---|
| `session_start` | `SessionID`, `ProviderData["motoko_commit"]`, `ProviderData["motoko_model"]` |
| `thinking` (per-step) | summed token/cost counters (fallback when `run_summary` is absent); `NumTurns++` |
| `native_tool_calls` | `ToolCallCount += len(tool_calls[])` |
| `done` (per-step success) | `Output` (last seen) |
| `dp7_verifier_rejected` | `ProviderData["dp7_rejections"]++` |
| `error` (top-level) | feeds `Error` field when no `run_summary` |
| `run_summary` (terminal) | **authoritative totals** — overrides per-step sums for `InputTokens`, `OutputTokens`, `CacheReadInputTokens`, `CacheCreationInputTokens`, `CostUSD`, `DurationMS`, `NumTurns`; sets `Success` from `finish_reason == "stop"` |

Unknown event types are silently captured into `ProviderData["motoko_events"]` for forward-compat.

## Cost model

Cost is read directly from motoko's `run_summary.total_cost_usd` (computed motoko-side from `cost_rates` per profile, in millicents → USD). The adapter does NOT call `CostModel.CalculateCost()` — motoko's number is authoritative because it includes the OpenRouter passthrough markup that pricing tables don't capture.

When `run_summary` is absent (crash mid-run), the adapter falls back to summing per-step `cost_usd` fields from `thinking` events. `Result.Success` is set to `false` in that case with a clear `Error` message.

## Cache token telemetry (Anthropic prompt caching)

motoko's schema v1 emits `cache_read_input_tokens` + `cache_creation_input_tokens` per-step (Anthropic populates both; OpenAI/Gemini populate `cache_read` only; other providers leave both at 0 — omit-when-absent on the wire). The adapter folds these into `Result.CacheReadInputTokens` + `Result.CacheCreationInputTokens` from the `run_summary.usage` block.

This was a deliberate cross-repo dependency: AILANG `std/ai.ail::StepResult` had to surface these fields first (commit `0dc4835a`), then motoko plumbed them into its emit pipeline (commit `84fa449`), then this adapter consumes them. Without the cache telemetry, "cheap because the prompt cached" looks identical to "cheap because the model is fast" in cost-arbitrage analysis.

## Known limits

- **Streaming**: Currently `ExecuteStreaming` parses the JSONL only after subprocess exit (M2 of M-MOTOKO-EXECUTOR-ADAPTER). True line-by-line streaming via file-tailing is a follow-up.
- **Session resume**: Not supported in v0.1. Each Execute spawns a fresh motoko run with a new session_id.
- **Tool-call drill-down**: We count tool calls but don't surface per-tool input/output. The `ProviderData["motoko_events"]` round-trip preserves the data for downstream consumers that need it.
- **Wall-clock vs motoko-internal time**: When `run_summary.duration_ms` is present, we use it (motoko measures from `started_at_ms` to terminal); fallback to subprocess wall time only when `run_summary` is absent.

## Trust boundary

motoko has an autonomous bash tool (no per-call approval prompt by default — the v0.6.x configuration trades safety for autonomy). When invoked from the eval harness:

- **Local mode**: motoko runs in a per-task tmpdir under the developer's user account. It can touch anything the developer can. Treat eval workspaces as untrusted: don't run against a workspace containing secrets or production data.
- **Cloud mode** (Cloud Run Job, M4 of this sprint): per-Job ephemeral container with VPC egress allowlist + secret bindings limited to OpenRouter + OpenAI + Gemini keys. The cost-control rule (no ANTHROPIC_API_KEY) is itself a trust-boundary mitigation — a runaway agent can't burn unbudgeted Anthropic spend.

## Pinned motoko revision

This adapter is tested against motoko_agent commit `f7b26c8` on branch `feature/v021-effect-row-migration` (sunholo-voight-kampff fork, PR pending against arniwesth/motoko_agent main). The schema v1 contract this adapter depends on shipped in commits `0c006be` (initial) + `84fa449` (cache token closure). Subsequent work that brought motoko_agent up on AILANG v0.21+ → v0.22+: `29a1fed` (ai_compat → std/ai.stepWithStream), `e960592` (bump 12 ext deps), `3b72542` (jnum/intToFloat), `8834a47` (regen lock), `ada0ae9` (effect-row workarounds), `f7b26c8` (ai_compat 0.2.1 + compose 0.2.4 dep bumps after registry republish).

**AILANG version floor: v0.22.0+** (the iface bug fix M-IFACE-NESTED-EFFECTS is required to type-check `agent_loop_v2.ail`'s call to `dispatch_step` when `dispatch_step` is imported from another module). Older AILANG will see `dispatch_step`'s on_chunk parameter as `(StreamChunk) -> ()` (closed empty effect) due to the nested-function-type effect-row stripping bug, and fail at "incompatible closed rows".

Verified date: 2026-05-26 on macOS Tahoe 26.5 (Studio eval rig). `make build` exits 0; full eval-smoke run deferred to a workstation with `OPENROUTER_API_KEY` in env.

When upgrading the pinned commit:

1. Re-capture the snapshot fixtures (`testdata/session_*.jsonl`) from a fresh motoko run.
2. Run `go test ./internal/executor/motoko/ -count=20` (catches map-iteration nondeterminism in `ProviderData`).
3. Update `Result.ProviderData["motoko_commit"]` will surface the new commit at runtime.

## Cross-references

- **Sprint design doc**: [`design_docs/planned/v0_18_0/m-motoko-executor-adapter.md`](../../../design_docs/planned/v0_18_0/m-motoko-executor-adapter.md)
- **Sprint plan**: [`design_docs/planned/v0_18_0/m-motoko-executor-adapter-sprint-plan.md`](../../../design_docs/planned/v0_18_0/m-motoko-executor-adapter-sprint-plan.md)
- **EXECUTOR_SHAPE.md**: [`docs/internal/EXECUTOR_SHAPE.md`](../../../docs/internal/EXECUTOR_SHAPE.md) — the two-pillar contract this adapter conforms to
- **motoko_agent design doc**: [M-MOTOKO-EVAL-INSTRUMENTATION](https://github.com/sunholo-data/motoko_agent/blob/motoko-dx-compaction-pending/design_docs/implemented/motoko_agent/m-motoko-eval-instrumentation.md) — the schema v1 contract
- **Closest analogue**: [`internal/executor/opencode/`](../opencode/) — both parse top-level NDJSON events
- **Closest deployment analogue**: [`docker/Dockerfile.agent-pi`](../../../docker/Dockerfile.agent-pi) — CLI install pattern (no Go toolchain)
