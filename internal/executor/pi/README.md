# Pi Coding Harness Executor

CLI-subprocess executor for [pi](https://pi.dev/) (npm: `@mariozechner/pi-coding-agent`),
a deliberately minimal Claude Agent SDK-based coding harness with broad multi-provider reach.

**Status**: M1 complete (fixtures captured); M2+ pending.

## Pinned Version

```
pi 0.70.2
@mariozechner/pi-coding-agent
```

Tested: 2026-04-27.

## Installation

```bash
npm install -g @mariozechner/pi-coding-agent
```

Requires a user-local npm prefix (no sudo needed). Tested with Node 25.5.0 via nvm.

## Invocation

```bash
pi --mode json --model <provider>/<model> -p "<prompt>"
```

The executor uses `--mode json` (NDJSON event stream) — **not** `-p` print mode.
Print mode emits final text only; JSON mode is required for streaming events,
token counts, and tool-use telemetry.

### Flags Used by the Executor

| Flag | Purpose |
|---|---|
| `--mode json` | NDJSON event output to stdout |
| `--print` / `-p` | Non-interactive (process prompt and exit) |
| `--model <provider>/<id>` | Model selection via provider-prefix shorthand |
| `--no-tools` | Disable all tools (when AllowedTools is empty) |
| `--tools <list>` | Allowlist tools (when AllowedTools is non-empty) |
| `--api-key <key>` | Optional explicit key (default: env vars) |
| `--no-session` | Ephemeral session (avoids polluting `~/.pi/sessions/`) |

### Authentication

Pi reads provider keys from environment variables — same as the underlying
SDKs. The executor passes through the host environment unchanged:

| Provider | Env var |
|---|---|
| Anthropic | `ANTHROPIC_API_KEY` |
| OpenAI | `OPENAI_API_KEY` |
| Google (Gemini) | `GEMINI_API_KEY` (or ADC for Vertex) |

## NDJSON Event Schema

Pi emits one JSON object per line on stdout. Top-level discriminator is `type`.

### Top-Level Event Types (observed)

| `type` | When | Key fields |
|---|---|---|
| `session` | Start of run | `id`, `version`, `cwd`, `timestamp` |
| `agent_start` | After session init | — |
| `turn_start` | Each turn begins | — |
| `message_start` | Per message in turn | `message: {role, content, ...}` |
| `message_update` | Incremental delta | `assistantMessageEvent: {type, ...}`, `partial`, `message` |
| `message_end` | Per-message complete | `message: {..., usage}` |
| `tool_execution_start` | Tool call begins | (tool name, args — TBD) |
| `tool_execution_end` | Tool call completes | (result — TBD) |
| `turn_end` | Turn boundary | `message`, `toolResults: []` |
| `agent_end` | **Terminal** | `messages: [...]` (full conversation) |

### `assistantMessageEvent.type` (inside `message_update`)

| Sub-type | Meaning |
|---|---|
| `text_start` / `text_delta` / `text_end` | Streaming text content |
| `thinking_start` / `thinking_delta` / `thinking_end` | Extended thinking content |
| `toolcall_start` / `toolcall_delta` / `toolcall_end` | Tool-call construction |

### Token / Cost Fields

In `message_end` and `turn_end`:

```json
"usage": {
  "input": 480,
  "output": 205,
  "cacheRead": 0,
  "cacheWrite": 0,
  "totalTokens": 685,
  "cost": {
    "input": 0.00048,
    "output": 0.001025,
    "cacheRead": 0,
    "cacheWrite": 0,
    "total": 0.001505
  }
}
```

**Pi reports cost directly** — the executor uses `usage.cost.total` for
`Result.CostUSD` rather than recomputing from token counts.

### Stop Reason

`message` carries a `stopReason` on `message_start` / `message_end` /
`turn_end` (and mirrored on `assistantMessageEvent.partial` during
`message_update`). Values observed in captured fixtures at 0.70.2:

| Value | Meaning | Maps to |
|---|---|---|
| `stop` | Model finished its turn | `stop` |
| `toolUse` | Model stopped to call a tool (**intermediate**) | `tool_calls` |

The executor takes the **last settled** value (`message_end` / `turn_end`
only — streaming `message_update` events carry cumulative partial state) and
normalizes it into `Result.FinishReason` via `normalizePiFinishReason`. A
tool-using run ends `toolUse` mid-run and `stop` at the end, so only the last
value is meaningful at run level.

The vocabulary is **not documented upstream** and only these two values have
been observed; unrecognized values are passed through verbatim rather than
coerced, so they surface in banked eval JSON. An explicit kill
(cost budget, timeout, cancellation) overrides the stream value — the eval
harness trusts `FinishReason` over the error string when categorizing.
Re-check this mapping when bumping the pinned pi version.

### Parser Strategy

- Use `message_end` (assistant role) as the source of truth for usage / cost.
- Use `agent_end` as the terminal event signalling completion.
- For `ExecuteStreaming`, fire `OnText` from `text_delta` events, `OnToolUse` from
  `toolcall_end` (full call assembled), `OnToolResult` from `tool_execution_end`.
- Skip `message_update` events that are intermediate cumulative state — only
  the deltas (`text_delta`, `thinking_delta`, `toolcall_delta`) carry new info.
- Preserve unknown top-level fields in `Result.ProviderData` for forward-compat.

## Schema Drift Notes

Pinned: `@earendil-works/pi-coding-agent@0.85.1` (`ExpectedPackage`/`ExpectedVersion`
in pi.go). `HealthCheck` asserts the running `pi --version` against the pin and errors
naming both. The upstream package moved from the abandoned `@mariozechner` name at
0.73.1; the fleet cut over in M-PI-HARNESS-UPGRADE (design doc carries the measured
0.73.1 → 0.84.4 → 0.85.1 drift table).

Drift is recorded or fatal, never silent (D4):
- unknown event `type` → counted into `ProviderData.pi_unknown_events`
- non-JSON lines → `ProviderData.pi_unparsed_lines`
- an assistant `message_end` with no `usage` → the run fails as `wire_drift`
- `auto_retry_*` → `ProviderData.pi_retries {count, max_attempts, exhausted}`
- `usage.reasoning` (0.84+) is a subset of `usage.output` on the wire; the executor banks
  `OutputTokens = output − reasoning`, `ReasonTokens = reasoning`
- `rawStopReason` (0.84+) → `ProviderData.pi_raw_stop_reason`; consulted for the finish
  reason only when `stopReason` is unrecognised

## Fixtures

One pair per wire version the fleet has actually run, captured live on the rig
2026-09-16 with the same directives (`ollama/glm-5.3-flash:cloud`):

- `testdata/v0_73_1/fizzbuzz.ndjson` — 38 events, no tools; cumulative `message_update` (~1.2 KB each)
- `testdata/v0_73_1/tool_use.ndjson` — 51 events, write + read tool calls, 3 turns
- `testdata/v0_85_1/fizzbuzz.ndjson` — 33 events, no tools; delta `message_update` (~280 B), `agent_settled`
- `testdata/v0_85_1/tool_use.ndjson` — 56 events, write + read, `rawStopReason`
- `testdata/v0_85_1/reasoning.ndjson` — `openrouter/z-ai/glm-5.3-flash --thinking medium`; `usage.reasoning` = 8 of `output` = 41

Tests that prove summation patch non-zero `cacheWrite`/`cost` into the pinned fixture
(`patchAssistantUsage` in fixtures_test.go) rather than trusting a hand-written stream.

## Cost Model

Pi uses the underlying provider's pricing and reports it inline. The executor's
`CostModel()` returns a placeholder (cost calculation is bypassed in favor of
`usage.cost.total` from each `message_end`).

## Known Limits

- Pi creates a stateful session under `~/.pi/sessions/` by default; the executor
  passes `--no-session` to keep runs ephemeral.
- The default provider is `google` if `--provider`/`--model` is not specified;
  the executor always provides an explicit `--model` to avoid surprises.
- Long output streams emit large `partial` payloads in every `message_update`;
  the parser must handle multi-megabyte single-line JSON without copying.
- **No plugin / extension surface for μRAG injection.** Pi runs the agent loop
  inside its own process and emits NDJSON events for *observation* — there is
  no PreToolUse / PostToolUse callback hook the host can interpose on to feed
  context back into the next turn. Cross-harness fairness with the
  builtin-first-use μRAG nudge that Claude Code / Gemini / Codex / opencode
  agents receive is therefore currently unachievable on Pi.
  Tracked under [M-MICRORAG-EXPAND](../../../design_docs/planned/v0_15_0/m-microrag-hook-expansion.md);
  unblocked once Pi adds upstream support for a passive PostToolUse hook
  (or alternatively, when SessionStart-style up-front context prepending
  proves valuable enough to ship as M3).

## References

- [Design doc](../../../design_docs/planned/v0_14_2/m-exec-pi-harness.md)
- [Sprint plan](../../../design_docs/planned/v0_14_2/m-exec-pi-harness-sprint-plan.md)
- [EXECUTOR_SHAPE.md](../../../docs/internal/EXECUTOR_SHAPE.md) — uniform contract
- [pi.dev](https://pi.dev/) | [pi-mono on GitHub](https://github.com/badlogic/pi-mono/tree/main/packages/coding-agent)
