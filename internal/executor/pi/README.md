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

### Parser Strategy

- Use `message_end` (assistant role) as the source of truth for usage / cost.
- Use `agent_end` as the terminal event signalling completion.
- For `ExecuteStreaming`, fire `OnText` from `text_delta` events, `OnToolUse` from
  `toolcall_end` (full call assembled), `OnToolResult` from `tool_execution_end`.
- Skip `message_update` events that are intermediate cumulative state — only
  the deltas (`text_delta`, `thinking_delta`, `toolcall_delta`) carry new info.
- Preserve unknown top-level fields in `Result.ProviderData` for forward-compat.

## Schema Drift Notes

Pi is pre-1.0 (v0.70.x). The schema captured here is from 0.70.2.
The parser tolerates unknown fields and unknown `type` values (logs and skips).
Re-capture fixtures and re-run tests when bumping the pinned version.

## Fixtures

- `testdata/fizzbuzz.ndjson` — 20 events, no tool use, `--no-tools`, claude-haiku-4-5
- `testdata/tool_use.ndjson` — 37 events, file-create tool call, claude-haiku-4-5

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

## References

- [Design doc](../../../design_docs/planned/v0_14_2/m-exec-pi-harness.md)
- [Sprint plan](../../../design_docs/planned/v0_14_2/m-exec-pi-harness-sprint-plan.md)
- [EXECUTOR_SHAPE.md](../../../docs/internal/EXECUTOR_SHAPE.md) — uniform contract
- [pi.dev](https://pi.dev/) | [pi-mono on GitHub](https://github.com/badlogic/pi-mono/tree/main/packages/coding-agent)
