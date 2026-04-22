# Codex Executor

OpenAI Codex CLI executor for AILANG. Implements the `executor.Executor`
interface as a subprocess driver around the [OpenAI Codex CLI](https://developers.openai.com/codex/cli/),
mirroring the `internal/executor/claude` and `internal/executor/gemini` packages.

## Status

- **Milestone**: M-EXEC-EXPAND / M1 (core) + M2 (registration)
- **Target version**: v0.15.0
- **Default model**: `gpt-5-codex`

## Registration

The Codex executor self-registers with the global executor factory via `init()`:

```go
package codex

func init() {
    Register()
}

func Register() {
    executor.GlobalFactory().Register("codex", func(cfg *executor.Config) (executor.Executor, error) {
        return New(cfg)
    })
}
```

As a result, **any Go file that blank-imports this package automatically
discovers Codex** through `executor.ExecutorProvider`, with no changes to the
coordinator. This matches the pattern already used by `claude` and `gemini`.

```go
// internal/coordinator/provider_executor.go
import _ "github.com/sunholo-data/ailang/internal/executor/codex"
```

## Command-line Flags

The executor invokes Codex in non-interactive mode, streaming NDJSON to stdout:

| Flag                                          | Purpose                                       |
| --------------------------------------------- | --------------------------------------------- |
| `exec`                                        | Non-interactive subcommand (one-shot run)     |
| `--json`                                      | Emit NDJSON events on stdout                  |
| `--skip-git-repo-check`                       | Allow running in non-git workspaces           |
| `--dangerously-bypass-approvals-and-sandbox`  | Auto-approve tool calls (coordinator context) |
| `--model <name>`                              | Select model (default `gpt-5-codex`)          |

Example invocation:

```bash
codex exec --json --skip-git-repo-check \
  --dangerously-bypass-approvals-and-sandbox \
  --model gpt-5-codex \
  "write fizzbuzz in python"
```

**Source**: Flag surface documented at <https://developers.openai.com/codex/cli/>
and confirmed by the NDJSON fixture at `testdata/codex_response.jsonl`.

## Authentication

Codex CLI reads credentials from (in priority order):

1. `OPENAI_API_KEY` environment variable — recommended for CI/CD and cloud machines
2. The local credential cache created by `codex login` (ChatGPT Plus/Pro OAuth2 session)

The executor does **not** fail `HealthCheck` when `OPENAI_API_KEY` is unset —
Codex may be using cached auth. A `DEBUG_AGENT=1` trace prints a warning so
missing auth is still surfaced for the developer.

### Headless / Remote / Cloud Machines

On machines without a browser (cloud VMs, remote SSH sessions, coordinator workers):

```bash
# Device authorization flow — prints a URL + code; authorize on any device with a browser
codex login --device-auth
```

This is the OAuth2 Device Authorization Grant (RFC 8628). It avoids the browser redirect
requirement of the standard `codex login` flow. Once authorized, credentials are cached
at `~/.codex/auth.json` (or equivalent platform path) and reused by all subsequent
`codex` invocations on that machine.

**Which to use:**

| Environment | Method |
|---|---|
| Laptop / desktop with browser | `codex login` |
| Cloud VM / SSH session / CI runner | `codex login --device-auth` |
| Automated pipeline (no human) | `OPENAI_API_KEY` env var |
| AILANG coordinator daemon | `OPENAI_API_KEY` in coordinator env (preferred) |

For the AILANG coordinator, `OPENAI_API_KEY` in the process environment is the most
reliable approach — it survives container restarts and doesn't require cached session
files to be present on every worker node. `codex login --device-auth` is the right
choice for interactive developer machines that don't expose API keys.

## Event Schema

Codex uses a **flat** NDJSON schema, distinct from Claude/Gemini's nested
`stream_event` wrapping. See `internal/executor/codex_compat_test.go` for the
compatibility analysis. Event types handled:

| Type                                   | Handling                                      |
| -------------------------------------- | --------------------------------------------- |
| `session` / `session_start` / `init`   | Capture session id, start turn span           |
| `message`                              | Emit text, accumulate cumulative token totals |
| `tool_use` / `tool_call`               | Increment `ToolCallCount`, fire handler       |
| `tool_result`                          | Fire handler with output                      |
| `result` / `turn_complete` / `message_stop` | Mark success, finalize tokens             |

**Unknown event types** are preserved in `Result.ProviderData["codex_events"]`
as the original raw event map, so schema drift does not break the executor.

### Token Semantics

Codex emits **cumulative** `tokens_used` in each `message` event (matching
OpenAI API usage semantics), not per-turn deltas. The parser takes `max(…)`
rather than summing, so the final totals reflect the cumulative state from the
last event, not a double-counted sum.

## Cost Model

`gpt-5-codex`: `$1.25 / $10.00` per 1M tokens (input / output).

| Token kind | Cost per 1K |
| ---------- | ----------- |
| Input      | `$0.00125`  |
| Output     | `$0.0100`   |
| Cache read | `$0.000125` |

**Source**: <https://platform.openai.com/docs/pricing> (`gpt-5-codex`).

Update the `CostModel()` method in `codex.go` if pricing changes.

## Configuration

Codex is exposed through the shared `executor.Config` struct:

```go
cfg := &executor.Config{
    CodexPath:      "codex",            // Path to codex CLI binary
    CodexModel:     "gpt-5-codex",      // Default model
    TimeoutSeconds: 300,                // Hard ceiling per task
}
```

Per-task overrides via `executor.Task.Model` take precedence over the config
default.

## Timeouts

- **Hard ceiling**: `task.Timeout` or `cfg.TimeoutSeconds` (default 5 min).
  Kills the subprocess when exceeded, returns `Success: false`.
- **Idle timeout**: `task.IdleTimeout` (default 3 min). Kills the subprocess
  when no stdout activity for the full window; resets on every event.

Both timeouts are implemented using `time.NewTimer` + `sync/atomic` for the
last-activity clock.

## Health Check

`HealthCheck(ctx)` runs `codex --version`:

- Binary missing → returns error (`install with: npm i -g @openai/codex`)
- Binary present but `--version` fails → returns error
- Binary OK, `OPENAI_API_KEY` unset → warns under `DEBUG_AGENT=1`, does not fail
- Binary OK → returns `nil`

## Known Limits

- **No session resumption**: `executor.Task.ResumeSessionID` is accepted but
  currently unused. Codex CLI does not expose a resume flag.
- **No plugin/skill install flow**: unlike Claude Code, Codex has no plugin
  directory surface. `task.PluginDirs` and `task.Plugins` are ignored.
- **`--dangerously-bypass-approvals-and-sandbox` is required** for autonomous
  coordinator use. In a hardened environment this flag may need to be swapped
  for a policy-based approval mechanism (out of scope for this sprint).
- **Non-JSON stdout preamble** (e.g. `"Creating GCP exporters…"`) is tolerated:
  the parser skips lines that do not start with `{`.

## Testing

```bash
go test ./internal/executor/codex/...        # unit + mock-binary integration
go test -cover ./internal/executor/codex/... # expect ≥ 70% coverage
```

The test suite uses a POSIX shell script as a fake `codex` binary that replays
the fixture at `testdata/codex_response.jsonl`. No real OpenAI account or
network access is required to run the tests.

## References

- **Design doc**: `design_docs/planned/v0_15_0/m-exec-expand-codex-opencode.md`
- **Sprint plan**: `design_docs/planned/v0_15_0/m-exec-expand-codex-opencode-sprint-plan.md`
- **Compat test**: `internal/executor/codex_compat_test.go`
- **Codex CLI docs**: <https://developers.openai.com/codex/cli/>
- **Pricing**: <https://platform.openai.com/docs/pricing>
