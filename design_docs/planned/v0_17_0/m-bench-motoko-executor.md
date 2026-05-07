---
name: M-BENCH-MOTOKO
description: Add motoko_agent (AILANG-native coding agent) as a benchmark executor in the eval harness alongside claude/gemini/codex/opencode/pi.
type: project
---

# M-BENCH-MOTOKO: Motoko Agent as a Benchmark Executor

> **⚠️ SUPERSEDED (2026-05-07)** by [`design_docs/planned/v0_18_0/m-motoko-executor-adapter.md`](../v0_18_0/m-motoko-executor-adapter.md).
>
> This doc is preserved for historical context — it captures the original strategic framing and the BLOCKING dependency on motoko stdlib catch-up (now resolved via M-MOTOKO-EXTENSION-INTEGRATION shipped 2026-05-07). The replacement doc reframes the work strictly around the EXECUTOR_SHAPE.md two-pillar contract (formalized v0.15.0, post-this-doc) and drops the now-unnecessary headless-mode fork-side spec.

**Status**: Superseded
**Target**: v0.17.0 (never shipped — see superseding doc for v0.18.0 plan)
**Priority**: P2 (Medium — closes the loop on dogfooding by benchmarking the first AILANG-native coding agent against the same task suite as Claude/Gemini/Codex; surfaces self-referential signal that no other executor can produce)
**Estimated**: 4-5 days (~26-32 hours) for local; +2 days for cloud
**Dependencies**:
- **Motoko stdlib upgrade to AILANG >v0.14.3 (BLOCKING)** — motoko's vendored AILANG runtime predates the M-AI-OPENROUTER and stdlib changes shipped through v0.14.3. The headless-mode PR (Day 1) must include or be preceded by a stdlib catch-up so motoko's agent loop compiles against current stdlib signatures. This is upstream-of-this-doc work and gates the entire milestone — none of the executor wiring is testable until motoko runs against current AILANG.
- M-EXTERNAL-CONSUMER-DX (same release; provides `error_codes.json` artifact that motoko already plans to consume — better diagnostics → fewer eval failures attributable to error message quality rather than agent quality)
- **M-AI-OPENROUTER (load-bearing)** — already merged; motoko routes through OpenRouter, so cost normalisation logic already lives in `internal/ai/`. This milestone is the first benchmark consumer of that integration and validates it under real eval traffic. It also unlocks **open-weight model benchmarking** (e.g. Gemma 4) which has no first-party agent CLI.
- Vendored fork: `sunholo-data/motoko_agent` (rebase-forward from `arniwesth/motoko_agent`)

**Author**: Claude + Mark
**Created**: 2026-05-04

---

## Executive Summary

[motoko_agent](https://github.com/arniwesth/motoko_agent) is the first production-scale coding agent built **on** AILANG (~5,200 LOC of `.ail` modules implementing the agent loop, tool dispatcher, extension system, and TUI bridge). Today our eval harness benchmarks five external CLIs (Claude Code, Gemini CLI, Codex, opencode, pi) — none of which exercise AILANG-the-language at runtime. Adding motoko as a sixth executor gives us:

1. **Dogfooding signal.** A regression in AILANG semantics that breaks motoko's agent loop will surface in our benchmark suite, not only in motoko's CI.
2. **Cross-harness comparison.** Motoko routes to multiple underlying models via OpenRouter, so we can pair `motoko-claude-sonnet-4-6` against bare `claude-sonnet-4-6` (Claude Code harness) on the same benchmark and isolate "agent shell quality" from "model quality" — a signal we cannot get from any other executor.
3. **First validation of the executor contract for an in-house agent.** Every existing executor is a third-party CLI. Motoko is the first agent we control end-to-end, which lets us iterate on the contract from both sides.
4. **Open-weight model coverage via OpenRouter.** With M-AI-OPENROUTER merged, the harness can now route to any model on OpenRouter's catalog — including open-weight models like **Gemma 4** that have no first-party coding-agent CLI. Motoko-as-host is the bridge: it gives Gemma 4 (and future open weights) an agent loop, so they appear in the same benchmark suite as Claude/GPT/Gemini-Pro on equal footing. This is the first time AILANG benchmarks can directly compare proprietary closed models against open weights on agentic coding tasks.

**Scope (in priority order):**

1. **Local executor `internal/executor/motoko/`** — new sub-package implementing the `executor.Executor` interface, parallel to gemini/codex/opencode.
2. **Headless mode in motoko fork** — `motoko run --task "..." --json` non-interactive mode that streams NDJSON events and writes a final `result.json` with usage/cost.
3. **`models.yml` entries** — four paired entries (`motoko-claude-sonnet-4-6`, `motoko-gemini-3-pro`, `motoko-gpt5`, `motoko-gemma-4`) so we can compare across both proprietary and open-weight model backends.
4. **OpenRouter cost reconciliation** — verify the M-AI-OPENROUTER cost-normalisation path correctly resolves `usage.cost_usd` for all four routed models, including open-weight pricing tiers where input/output costs differ by an order of magnitude from proprietary models.
5. **Eval harness wiring** — blank-import in `agent_runner_multi.go`; benchmark documentation update.
6. *(Stretch)* **Cloud executor** — Docker variant + Cloud Run Job definition for parity with cloud-eligible executors (`opencode`, `codex`).

Out of scope: changes to motoko's agent loop or extension system; benchmarking motoko-the-runtime-as-language (motoko already has its own benchmarks); self-modifying eval (where motoko would write AILANG code that improves AILANG itself — separate v0.18 thought).

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Adds an executor; no semantic change |
| A2: Replayability | +1 | Each motoko run emits NDJSON + `result.json` to the workspace, replayable from disk like other executor results |
| A3: Effect Legibility | 0 | None |
| A4: Explicit Authority | -1 | Motoko has "autonomous execution — plans and runs commands without pausing for approval"; we accept this within an isolated workspace but it widens the local executor authority surface |
| A5: Bounded Verification | +1 | Standard executor timeouts (idle/TTFT/hard) bound runs |
| A6: Safe Concurrency | 0 | One process per task; existing executor patterns apply |
| A7: Machines First | +2 | Headless mode + structured `result.json` is exactly the artefact contract; replaces TUI-only output |
| A8: Minimal Syntax | 0 | No language change |
| A9: Cost Visibility | +2 | Motoko routes through OpenRouter; `result.json` carries `{cost_usd, input_tokens, output_tokens, cached_tokens}` per run, surfaced into AgentBenchmarkResult |
| A10: Composability | +1 | First in-house executor; validates the contract for future internal agents |
| A11: Structured Failure | +1 | Failures emit a typed `result.json` with `{success: false, error_kind, error_message, last_step}` instead of TUI-only stack traces |
| A12: System Boundary | +1 | The motoko↔AILANG-eval-harness boundary is explicit (NDJSON events + `result.json`), not implicit |

**Net Score: +8** → **Decision: ✅ Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): No effect on language semantics; executor is a benchmark consumer
- [x] A3 (Effects): No new AILANG effect; motoko's runtime FS/Net authority is bounded by the workspace timeout pattern shared with other executors
- [x] A4 (Authority): Workspace is per-run tmpdir, deleted after benchmark unless `--keep-workspace` set. Acceptable for local. Cloud variant must add network egress allowlist (OpenRouter only) — tracked in stretch milestone.
- [x] A11 (Failure): `result.json` schema enforced; missing/malformed JSON treated as `Success=false` with `Error="motoko did not emit result.json"`

---

## Motivating Evidence

### 1. We have no in-house executor today

`internal/executor/` ships five executors, all wrapping third-party binaries:

| Executor | Binary | Source |
|----------|--------|--------|
| claude | `claude` (Node) | Anthropic |
| gemini | `gemini` (Node) | Google |
| codex | `codex` (Rust) | OpenAI |
| opencode | `opencode` | OpenAI / sst |
| pi | `pi` | sst |

When a benchmark fails, we cannot tell whether the failure is "the agent harness mishandled the task" vs "the model couldn't solve the task" vs "AILANG-the-target-language was hard to write" — the harness is opaque. Motoko's agent loop is in our repo (vendored), so we own all three layers.

### 2. M-EXTERNAL-CONSUMER-DX establishes that motoko is a real consumer

The companion v0.17.0 design doc M-EXTERNAL-CONSUMER-DX cites motoko's `.agent/learnings/` as evidence of three concrete DX gaps in AILANG core. That work is already approved for this release. It is wasteful to fix DX based on motoko's *manual* feedback while having no automated regression signal. Adding motoko to the eval harness closes that loop: a future DX regression would manifest as a drop in motoko's benchmark pass rate, not just an `.agent/learnings/` note discovered weeks later.

### 3. Cross-harness comparison is currently impossible for "agent quality"

`models.yml` already pairs codex and opencode entries via `model_family: "gpt5-5"` to enable cross-harness comparison of GPT-5.5 across the codex and opencode shells. There is no equivalent for "AILANG-native agent loop vs. Claude Code's agent loop" because no AILANG-native agent is in the harness. Motoko fills this gap:

```
model_family: "claude-sonnet-4-6"
  ├── claude-sonnet-4-6              (Claude Code harness)
  ├── opencode-claude-sonnet-4-6     (opencode harness)
  └── motoko-claude-sonnet-4-6       (motoko harness, NEW)
```

A consistent ordering across this trio measures *the agent shell*, not the model.

### 4. Open-weight models are absent from the agent benchmark suite

Today's `models.yml` lists only proprietary models. Open-weight models like Gemma have no first-party agent CLI — there is no `gemma run --task ...` analogue to `claude` or `gemini`. With M-AI-OPENROUTER merged, the `internal/ai/` provider can route to any OpenRouter-hosted model, but that integration is currently exercised only for *non-agent* benchmarks (single-shot text generation). Wiring motoko + OpenRouter together gives Gemma 4 (and any future open-weight model on OpenRouter) an agent loop, allowing it to compete on the same agentic-coding benchmark suite as Claude/GPT/Gemini-Pro. This is the only planned path to open-weight agent benchmarks in v0.17.0.

### 4. Motoko's current entrypoint is not benchmark-callable

From the upstream README: invocation is `make run TASK="..."` with output rendered to a TUI. No `--print`, no JSON, no headless flag. We cannot wire this to the eval harness without adding one. Since the fork is in `sunholo-data/`, the cost of adding it is one PR within an org we control rather than upstream coordination.

---

## Design

### Item 1: Headless mode in motoko fork

Add a non-interactive entrypoint to the vendored motoko fork. Two surfaces:

**A. CLI invocation:**

```
motoko run \
  --task "<directive>" \
  --workdir /path/to/workspace \
  --profile eval \
  --model anthropic/claude-sonnet-4-6 \
  --max-steps 50 \
  --json \
  --emit-result-file result.json
```

The `--json` flag swaps the TUI renderer for an NDJSON event stream on stdout (one event per line, mirroring the gemini stream-json shape so we can reuse our parsing patterns). `--emit-result-file` writes a final `result.json` to the workspace with end-of-run aggregates.

**B. NDJSON event schema** (compatible with our `EventHandler` callbacks):

```json
{"type":"init","session_id":"<uuid>","model":"anthropic/claude-sonnet-4-6","profile":"eval"}
{"type":"turn_start","turn":1}
{"type":"message","role":"assistant","content":"I'll start by reading the existing config..."}
{"type":"tool_use","tool_name":"read_file","parameters":{"path":"src/config.go"}}
{"type":"tool_result","tool_name":"read_file","output":"package config\n..."}
{"type":"turn_end","turn":1}
...
{"type":"result","status":"success","stats":{"input_tokens":3500,"output_tokens":1200,"cached_input_tokens":1800,"total_tokens":4700,"cost_usd":0.0142,"duration_ms":18450,"tool_calls":12,"num_turns":7}}
```

**C. `result.json` schema** (final aggregate file, redundant with the `result` event but easier to read post-hoc):

```json
{
  "schema": "motoko.eval_result/v1",
  "session_id": "<uuid>",
  "success": true,
  "model": "anthropic/claude-sonnet-4-6",
  "num_turns": 7,
  "tool_call_count": 12,
  "duration_ms": 18450,
  "usage": {
    "input_tokens": 3500,
    "output_tokens": 1200,
    "cached_input_tokens": 1800,
    "cost_usd": 0.0142,
    "provider_billing_source": "openrouter"
  },
  "files_created": ["main.ail"],
  "files_modified": [],
  "error": null
}
```

**Why two surfaces (NDJSON + result.json)?** NDJSON gives us live event callbacks and per-turn telemetry spans (matching gemini's pattern). `result.json` gives us authoritative aggregates that don't depend on parsing every line of stdout being non-lossy — defence in depth against stdout buffering issues we have hit with other Node-based binaries.

**Implementation:** add a thin renderer in `motoko_agent/src/tui/` that branches on `--json`. The agent loop already has all the data; this is a serialisation change, not a behaviour change. Estimated 1 day in motoko.

### Item 2: `internal/executor/motoko/` sub-package

New sub-package paralleling `internal/executor/gemini/`:

```
internal/executor/motoko/
  motoko.go          (~550 LOC: implements Executor interface)
  motoko_test.go     (~500 LOC: fixture replay + mock binary tests)
  README.md          (~80 LOC: flags, auth, schema, limits)
  testdata/
    fixtures/
      success_simple.ndjson   (recorded NDJSON stream, golden)
      success_multi_turn.ndjson
      failure_compile_error.ndjson
      failure_timeout.ndjson
    mock_motoko.sh            (test double: emits a fixture based on $MOTOKO_FIXTURE env var)
```

**Interface implementation:**

```go
type Executor struct {
    motokoPath string  // resolved at New() — checks PATH, then ~/.local/bin/motoko, then vendored ./bin/motoko
    config     *executor.Config
}

func (e *Executor) Name() string                              { return "motoko" }
func (e *Executor) Capabilities() []executor.Capability {
    return []executor.Capability{
        executor.CapStreaming,
        executor.CapLocalWorkspace,
        executor.CapCostReporting,
        executor.CapTokenReporting,
        // NOT CapSessionResume initially — motoko doesn't expose --resume yet
    }
}
func (e *Executor) HealthCheck(ctx context.Context) error {
    // Verify: motoko binary exists, Bun >= 1.0, Node >= 18, Go >= 1.22, OPENROUTER_API_KEY set
}
func (e *Executor) ExecuteStreaming(ctx context.Context, task *executor.Task, h executor.EventHandler) (*executor.Result, error) {
    // 1. Generate per-run profile at <workspace>/.motoko/config/eval/config.json from task fields
    // 2. Build args: motoko run --task <directive> --workdir <workspace> --profile eval --json --emit-result-file result.json
    // 3. exec.CommandContext + stdout/stderr pipes (mirror gemini.go pattern)
    // 4. Parse NDJSON in goroutine, dispatch to EventHandler
    // 5. After cmd.Wait, read <workspace>/result.json for authoritative aggregates
    // 6. Reconcile NDJSON-derived counters against result.json (warn on divergence > 5%)
    // 7. Convert to executor.Result
}
```

**Profile generation:** each task gets a fresh profile written to `<workspace>/.motoko/config/eval/config.json`:

```json
{
  "agent": {
    "model": "<task.Model>",
    "workdir": "<task.Workspace>",
    "max_steps": <derived from task.IdleTimeout / task.Timeout — default 50>
  },
  "extensions": {
    "order": [],
    "strict": false
  }
}
```

We start with **no extensions** — pure baseline agent loop. A follow-up can add `motoko-with-context_mode`, `motoko-with-omnigraph`, etc. as separate `models.yml` entries to benchmark extension impact.

**Process management:** identical pattern to gemini — `IdleTimeout` enforced via `lastActivity.Store(time.Now().UnixNano())` in the NDJSON loop; `Timeout` via context; `TTFTTimeout` via wait-for-first-event watchdog.

**File capture:** track files via `tool_use` events with tool_name in `{write_file, edit_file, create_file}`; cross-validate against a workspace dir-walk diff post-run for `FilesCreated`/`FilesModified`.

### Item 3: `models.yml` entries

Four initial entries — three paired to existing proprietary `model_family` groups for cross-harness comparison, plus one open-weight entry that introduces a new `gemma-4` family with no proprietary counterpart:

```yaml
motoko-claude-sonnet-4-6:
  api_name: "anthropic/claude-sonnet-4-6"
  provider: "openrouter"
  env_var: "OPENROUTER_API_KEY"
  agent_cli: "motoko"
  agent_model_name: "anthropic/claude-sonnet-4-6"
  model_family: "claude-sonnet-4-6"
  max_output_tokens: 8192
  pricing:
    input_per_1k: 0.003
    output_per_1k: 0.015
  notes: |
    Motoko agent harness backed by Claude Sonnet 4.6 via OpenRouter.
    Pairs with `claude-sonnet-4-6` (Claude Code harness) and
    `opencode-claude-sonnet-4-6` for cross-harness comparison.

motoko-gemini-3-pro:
  api_name: "google/gemini-3-pro"
  provider: "openrouter"
  env_var: "OPENROUTER_API_KEY"
  agent_cli: "motoko"
  agent_model_name: "google/gemini-3-pro"
  model_family: "gemini-3-pro"
  max_output_tokens: 8192
  pricing: { input_per_1k: 0.00125, output_per_1k: 0.005 }

motoko-gpt5:
  api_name: "openai/gpt-5"
  provider: "openrouter"
  env_var: "OPENROUTER_API_KEY"
  agent_cli: "motoko"
  agent_model_name: "openai/gpt-5"
  model_family: "gpt5"
  max_output_tokens: 8192
  pricing: { input_per_1k: 0.00125, output_per_1k: 0.01 }

motoko-gemma-4:
  api_name: "google/gemma-4"   # exact OpenRouter slug confirmed at integration time;
                                # current Gemma family on OpenRouter uses pattern `google/gemma-3-<variant>`
  provider: "openrouter"
  env_var: "OPENROUTER_API_KEY"
  agent_cli: "motoko"
  agent_model_name: "google/gemma-4"
  model_family: "gemma-4"       # new family; no proprietary counterpart
  max_output_tokens: 8192
  pricing:
    input_per_1k: 0.00005       # placeholder — Gemma open-weight pricing on OpenRouter
                                # is typically 1-2 orders of magnitude below proprietary
                                # frontier models; confirm against OpenRouter at wiring time
    output_per_1k: 0.00010
  notes: |
    Motoko agent harness backed by Gemma 4 (Google open weights) via OpenRouter.
    First open-weight entry in the agent benchmark suite. No first-party agent
    CLI exists for Gemma — motoko-as-host is the only path into the harness.
    Use as the price/capability floor for cross-family comparison: a benchmark
    that motoko-gemma-4 passes is plausibly solvable by *any* harnessed model;
    one only the proprietary entries pass quantifies the open-weight gap.
```

Pricing is informational only — actual cost comes from `result.json.usage.cost_usd` (OpenRouter passes through provider billing for both proprietary and open-weight models); the pricing block is the fallback if `cost_usd` is absent. The Gemma 4 entry will exercise the M-AI-OPENROUTER cost-normalisation path against a 100x cheaper price tier than the proprietary entries — a useful regression test for the OpenRouter integration's numerical handling at the low end.

### Item 4: Eval harness wiring

One blank import in `internal/eval_harness/agent_runner_multi.go`:

```go
import _ "github.com/sunholo-data/ailang/internal/executor/motoko"
```

Plus an entry in `EXECUTOR_SHAPE.md` (the formal contract document) listing motoko's capabilities and any deviations from the contract.

### Item 5 (Stretch): Cloud executor

Cloud variant requires:
- Dockerfile with Bun + Node 22 + Go 1.22 + the vendored motoko binary
- Cloud Run Job definition mirroring opencode/codex cloud variants
- Network egress: allow OpenRouter API, deny everything else (motoko's autonomous bash execution makes wider egress a concern)
- Workspace is a Cloud Storage volume, identical to other cloud executors

Defer to a follow-up unless items 1-4 ship with time to spare.

---

## Implementation Plan

### Day 0 (prerequisite): Motoko stdlib catch-up to AILANG >v0.14.3

- Bump motoko's vendored AILANG version to current (≥v0.14.3); resolve any stdlib signature breakages in motoko's `.ail` modules.
- Re-run motoko's existing test suite against the upgraded runtime.
- Tag the upgraded motoko commit; this becomes the pinned commit referenced in Risk #3 below.
- **Acceptance:** `make check_core` and `make test` in motoko pass against AILANG ≥v0.14.3; tagged release cut.
- **Note:** this work happens in `sunholo-data/motoko_agent`, not in AILANG. It's listed here because it's a hard prerequisite for everything that follows.

### Day 1: Motoko fork — headless mode

- Add `--json` and `--emit-result-file` flags to motoko's CLI (now running on the upgraded stdlib from Day 0).
- Implement NDJSON renderer (alternative to TUI renderer; same agent loop).
- Implement `result.json` writer at end of run (success and failure paths).
- Smoke test: `motoko run --task "echo hi" --json --emit-result-file r.json` produces parseable NDJSON + valid `r.json`.
- PR to `sunholo-data/motoko_agent`.
- **Acceptance:** all NDJSON event types parseable as the schema documented above; `result.json` validates against the v1 schema.

### Day 2: `internal/executor/motoko/` skeleton

- Sub-package + `Executor` interface implementation.
- `Register()` + `init()` registration.
- `HealthCheck()` verifying motoko binary + Bun/Node/Go versions + OPENROUTER_API_KEY.
- Profile generation in workspace.
- Process spawn + NDJSON parser (mirror gemini.go structure).
- Event dispatch to `EventHandler`.
- **Acceptance:** unit test with mock_motoko.sh fixture passes; HealthCheck fails with a clear message when any prerequisite is missing.

### Day 3: Result reconciliation + tests

- Read `result.json` after `cmd.Wait`; reconcile against NDJSON-derived counters.
- File capture (NDJSON tool events + workspace diff).
- Failure-path tests (timeout, motoko crash, malformed result.json, network failure → no result.json).
- Integration test in `internal/executor/integration_test.go` (skipped unless `MOTOKO_INTEGRATION=1` and a real binary is present).
- **Acceptance:** golden-fixture replay tests pass; integration test passes against a real motoko binary on the maintainer's box.

### Day 4: models.yml + harness wiring + first eval baseline

- Add four `motoko-*` entries to `models.yml` (claude-sonnet-4-6, gemini-3-pro, gpt5, gemma-4).
- Confirm exact OpenRouter slug for Gemma 4 against the live catalog (`curl https://openrouter.ai/api/v1/models | jq '.data[] | select(.id | contains("gemma"))'`); update `api_name` and `agent_model_name` accordingly.
- Blank-import in `agent_runner_multi.go`.
- Update `EXECUTOR_SHAPE.md`.
- Run smoke tier across all four entries: `ailang eval-suite --models motoko-claude-sonnet-4-6,motoko-gemini-3-pro,motoko-gpt5,motoko-gemma-4 --tier smoke`; capture baseline JSONs.
- Validate OpenRouter cost-normalisation: assert all four results have `cost_usd > 0` and that Gemma's value is at least 10x lower than Claude's on the same task (sanity check for the open-weight pricing path).
- Document the executor in `internal/executor/motoko/README.md`, including the open-weight comparison story.
- **Acceptance:** smoke tier completes for all four entries; results land in `eval_results/latest/results_motoko-*.json` with non-zero `cost_usd`; Gemma entry's `model_family: "gemma-4"` is the first open-weight family in the eval suite.

### Day 5-6 (stretch): Cloud executor

- Dockerfile, Cloud Build config, Cloud Run Job def.
- Egress allowlist via VPC connector + serverless VPC egress.
- End-to-end cloud run on one benchmark.

---

## Acceptance Criteria

- `make ci` passes with motoko sub-package included.
- `make test ./internal/executor/motoko/...` passes (fixture replay tests, no real binary required).
- `MOTOKO_INTEGRATION=1 make test ./internal/executor/motoko/...` passes against a real motoko binary on the maintainer's machine.
- `ailang eval-suite --models motoko-claude-sonnet-4-6 --tier smoke` completes without errors and produces a results JSON with `executor: "motoko"`, non-zero `cost_usd`, and matching `model_family` to the paired `claude-sonnet-4-6` entry.
- `result.json` schema is documented in `internal/executor/motoko/README.md` and pinned in motoko's repo at the same v1 version.
- `EXECUTOR_SHAPE.md` lists motoko with its capability set.
- CHANGELOG.md entry under v0.17.0.
- Design doc moved to `design_docs/implemented/v0_17_x/m-bench-motoko-executor.md` post-merge.

## Telemetry

- **Cross-harness delta**: track pass-rate delta on each benchmark tier between `motoko-claude-sonnet-4-6` and `claude-sonnet-4-6`. Hypothesis: motoko within 10pp of bare Claude on smoke tier; gap may widen on multi-turn tiers (motoko's loop is younger).
- **Cost ratio**: track `cost_usd` ratio between paired harnesses on the same model. Expectation: motoko within 1.5x of bare Claude (possibly higher due to less aggressive prompt caching initially).
- **Open-weight gap**: track pass-rate delta between `motoko-gemma-4` and `motoko-claude-sonnet-4-6` on each benchmark tier. Same harness, different model — isolates *model capability* on agentic coding from *agent shell quality*. This is the headline number for the open-weight comparison story.
- **OpenRouter cost-normalisation regression**: on every eval run, assert `result.json.usage.cost_usd > 0` for all four motoko entries (no silent zero-cost). The Gemma 4 entry's small absolute cost is the most sensitive to floating-point or unit-conversion bugs in the M-AI-OPENROUTER pricing path.
- **Self-referential failure flag**: if a benchmark passes for `claude-sonnet-4-6` but fails for `motoko-claude-sonnet-4-6` with `error_kind == "compile_error"` from motoko's own AILANG runtime, flag for AILANG-core regression review.
- Per-run NDJSON span emission via existing `telemetry.StartSpan(... "exec.turn", ...)` pattern from gemini executor — no new telemetry plumbing.

## Risks

1. **NDJSON stream divergence from spec.** Motoko's event renderer might emit slightly different shapes than gemini's, causing parser drift over time. *Mitigation:* the schema is owned by the executor sub-package's `testdata/` fixtures; any motoko change that breaks a fixture must update the fixture in the same PR (cross-repo gate enforced via CI).
2. **`result.json` missing on crash.** If motoko panics before writing the file, we have only the NDJSON event stream. *Mitigation:* `result.json` writer wraps the agent loop in `defer`; partial-state `result.json` is acceptable as long as it has `success: false` + `error_kind`. Reconciliation logic handles the missing-file case as `success=false, error="motoko did not emit result.json"`.
3. **Self-referential masking.** If AILANG regresses in a way that breaks motoko's loop, motoko's benchmark numbers crater across the board, masking the *agent quality* signal we're trying to measure. *Mitigation:* pin motoko to a specific commit per AILANG release (recorded in `EXECUTOR_SHAPE.md` and `result.json.motoko_commit`); rebase forward only between releases, not within a release. The Day 0 stdlib catch-up produces the **first such pinned commit** — its tag is recorded in `EXECUTOR_SHAPE.md` against AILANG v0.17.0.

7. **Stdlib upgrade surfaces latent motoko bugs.** Upgrading motoko to AILANG ≥v0.14.3 (Day 0) is non-trivial: signatures may have changed, and motoko's agent loop has not been exercised against current AILANG. Could surface bugs that block the rest of the milestone. *Mitigation:* Day 0 is sequenced first and gated behind motoko's own test suite passing; if catch-up takes >1 day, escalate scope back to maintainer and consider pinning to whichever older AILANG version motoko currently targets (the cross-harness signal is still valuable; only the M-AI-OPENROUTER stress-test value is lost). Track the upgrade as an issue in motoko's repo with the AILANG-side dependency cross-linked.
4. **OpenRouter as single billing source.** All motoko entries route through OpenRouter; a billing API change there would zero out our cost numbers. *Mitigation:* accept fallback to `pricing` block in `models.yml` when `cost_usd` is absent; alert on > 24h of absence.
5. **Autonomous bash execution.** Motoko's "no approval" bash tool means a hallucinating agent could touch anything reachable from the workspace. *Mitigation (local):* per-run tmpdir under the OS tmp root; documented as a trust boundary in README. *Mitigation (cloud, stretch):* network egress allowlist + ephemeral container.
6. **Bun/Node/Go prereq drift.** Three runtime prereqs vs. one for most executors. *Mitigation:* HealthCheck reports the version of each, fails fast with a single error message listing all missing prereqs.

## Documentation Impact

- New: `internal/executor/motoko/README.md` (flags, auth, schema, limits, trust boundary).
- Updated: `EXECUTOR_SHAPE.md` (add motoko row to the capability matrix; document v1 `result.json` schema).
- Updated: `docs/docs/guides/evaluation/executors.md` (or equivalent) — list motoko as an available executor with the cross-harness comparison story.
- Updated: motoko fork's README — document `--json` + `--emit-result-file` flags.
- CHANGELOG.md: entry under v0.17.0.

## Related Work

- M-EXTERNAL-CONSUMER-DX (same release) — improves AILANG diagnostics that motoko encounters; better diagnostics → fewer eval failures attributable to error message quality. The two milestones reinforce each other.
- M-AI-OPENROUTER (v0.14.x, merged) — **load-bearing dependency**. Motoko routes through OpenRouter; the cost normalisation logic established there is reused for motoko's `usage.cost_usd` reconciliation, and the integration's catalog access is what unlocks open-weight model entries (Gemma 4) in the agent harness. This milestone is the first eval-traffic stress test of M-AI-OPENROUTER under both proprietary and open-weight pricing paths.
- M-AI-TOOL-LOOP (same release) — orthogonal: that milestone is "AILANG drives an agent loop"; this milestone is "an AILANG-built agent loop is benchmarked". The two together demonstrate the full inversion.
- Future: M-BENCH-MOTOKO-EXTENSIONS (proposed v0.18) — add `motoko-with-context_mode`, `motoko-with-omnigraph`, etc. as separate `models.yml` entries to benchmark extension impact.
- Future: M-BENCH-MOTOKO-CLOUD (proposed v0.18 if stretch defers) — Docker + Cloud Run variant.
