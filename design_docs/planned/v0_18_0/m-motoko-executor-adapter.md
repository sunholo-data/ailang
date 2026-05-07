# M-MOTOKO-EXECUTOR-ADAPTER: motoko_agent as a CLI-Subprocess Executor

**Status**: Planned
**Target**: v0.18.0
**Priority**: P1 (High — strategic measurement for cheap-model + tuned-harness vs frontier-model thesis; first AILANG-native agent benchmarked alongside third-party CLIs)
**Estimated**: 3 days (~16 hours: 8h Pillar 1 / 4h Pillar 2 / 4h fixtures + tests + docs)
**Dependencies**:
- ✅ **M-MOTOKO-EVAL-INSTRUMENTATION** (shipped 2026-05-07, motoko_agent commit `0c006be` on PR #6, [implemented design doc](https://github.com/sunholo-data/motoko_agent/blob/motoko-dx-compaction-pending/design_docs/implemented/motoko_agent/m-motoko-eval-instrumentation.md)) — motoko's session JSONL now emits schema-v1 with per-step `input_tokens`/`output_tokens`/`cost_usd`, `run_summary` terminal event with totals, `schema_version` envelope, and `session_id` top-level field. Adapter can now populate `Result.CostUSD` / `Result.InputTokens`/`Result.OutputTokens` directly from the JSONL.
- ✅ EXECUTOR_SHAPE.md two-pillar contract (formalized in v0.15.0 by [m-exec-expand-codex-opencode](../../implemented/v0_15_0/m-exec-expand-codex-opencode.md))
- ✅ Pi adapter pattern ([M-EXEC-PI](../../implemented/v0_14_2/m-exec-pi-harness.md)) — closest analogue (multi-provider CLI, no Go toolchain inside the container)
- ✅ M-AI-OPENROUTER (merged v0.14.x) — motoko routes through OpenRouter, so cost-normalization path exists
- ✅ M-MOTOKO-EXTENSION-INTEGRATION (shipped 2026-05-07) — motoko_agent on AILANG dev with all 9 motoko-ext-* packages registry-published; build green; PR #6 open against `arniwesth/motoko_agent`
- ✅ DP7 verifier gate fail-open fix (msg bfec39b8, commit `1ef4e79`) — DP7 no longer blocks `done` on projects without `make check_core` infrastructure

**Author**: Claude Sonnet 4.6 + Mark
**Created**: 2026-05-07

**Supersedes:** [`design_docs/planned/v0_17_0/m-bench-motoko-executor.md`](../v0_17_0/m-bench-motoko-executor.md) (M-BENCH-MOTOKO, 2026-05-04, 444 lines) — that doc predates the M-MOTOKO-EXTENSION-INTEGRATION sprint and the EXECUTOR_SHAPE.md two-pillar formalization. Its BLOCKING dependency (motoko stdlib catch-up to AILANG ≥v0.14.3) is now resolved. This rewrite reframes the work strictly as an EXECUTOR_SHAPE-conformant adapter and drops the headless-mode-spec content (motoko already streams a usable session JSONL today; no fork-side changes required for v0.1).

---

## Problem Statement

AILANG benchmark data shows a **steep model-quality gradient** on AILANG-specific tasks:

| Model            | AILANG | Python | Gap (pp) |
|------------------|-------:|-------:|---------:|
| claude-opus-4-5  |  100%  | parity |       0  |
| claude-haiku-4-5 |  61.4% | 77.7%  |     +16  |
| gemini-2-5-flash |  46.4% | 91.2%  |     +45  |
| gpt5-mini        |  32.2% | 97.1%  |     +65  |

The strategic question is: **can motoko's harness (DP7 verifier gate, microRAG context, contract-feedback loops) lift cheaper models above the AILANG-correctness threshold?**

If yes → motoko has a strong cost-arbitrage story (cheap model + tuned harness ≈ frontier model).
If no → motoko's market is premium-tier verification only.

**We cannot answer this without empirical measurement.** The eval harness already runs `claude`, `gemini`, `codex`, `opencode`, `pi` — but none exercise an AILANG-native agent loop. Motoko is the first production-scale coding agent built **on** AILANG (~5,200 LOC of `.ail` modules + the now-9-package extension ecosystem). Adding motoko as the sixth executor gives us:

**Current State (no motoko in the harness):**
- Zero data on whether motoko's tuned-harness lift on cheap models is real or anecdotal
- Cross-harness comparison ("which CLI is best for AILANG?") missing the one harness explicitly designed for AILANG
- Open-weight model coverage gap — Gemma 4, Qwen, GLM-5 have no first-party agent CLI; motoko routes them all via OpenRouter, but no eval data exists for any of them on AILANG benchmarks
- No dogfooding signal — an AILANG semantic regression that breaks motoko's agent loop will surface only in motoko's CI, not in our benchmark suite

**Impact:**
- **Strategic:** the cheap-model arbitrage thesis is unverified. We're investing in motoko's harness but have no benchmark evidence it does what we claim.
- **Tactical:** every other executor adapter exists; motoko's absence is a visible gap on the leaderboard and in agent-suite reports.
- **Defensive:** AILANG-side regressions that affect motoko (e.g., the `VerificationConfig` literal unification fix earlier today) are caught only by motoko's own tests, with no cross-validation in AILANG's CI.

---

## Goals

**Primary Goal:** Add `internal/executor/motoko/` as a fully EXECUTOR_SHAPE-conformant CLI-subprocess executor (Pillar 1 + Pillar 2), enabling `ailang eval-suite --executor motoko --models <any>` and `ailang eval-suite --models agent_suite` (which now includes motoko entries).

**Success Metrics:**
- `ailang eval-suite --executor motoko --models motoko-haiku-4-5 --benchmarks <suite>` runs end-to-end and produces `result.json` with non-zero `usage.cost_usd`, populated `tokens`, and a passing/failing classification per benchmark
- All 4 motoko-paired entries land in `models.yml`: `motoko-claude-haiku-4-5`, `motoko-claude-sonnet-4-6`, `motoko-glm-5`, `motoko-gemma-4` (covering proprietary cheap, proprietary premium, OpenRouter-routed top-OS, Google-OS)
- A2A-tier benchmark comparison `motoko-haiku-4-5` vs vanilla `claude-haiku-4-5` (same model, with-vs-without harness) produces a defensible delta — number doesn't matter, **the measurement existing matters**
- Cloud Run dispatch via the coordinator works end-to-end with the new `agent-motoko` Cloud Run Job
- One blank-import line added to `internal/coordinator/provider_executor.go`; zero changes to dispatch/factory/coordinator code (per EXECUTOR_SHAPE auto-discovery contract)

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|---|---|---|---|---|
| Reuse motoko's existing session JSONL stream as the parser source (no headless-mode fork changes) | Avoids cross-repo coupling — motoko fork stays unchanged; adapter parses `.motoko/logfile/session_*.jsonl` directly | human | design | high (reversing means coupling motoko fork releases to AILANG releases) |
| Spawn `motoko "<task>"` global wrapper (`/Users/mark/go/bin/motoko`), not `cd motoko_agent && go run`-style invocation | Mirrors how every other executor uses its CLI; keeps motoko a pure subprocess from AILANG's perspective | human | design | med (would require fork API changes to reverse) |
| Adapter is read-only on motoko's filesystem: spawns the CLI, parses the JSONL it emits, never writes into `.motoko/` | Maintains a clean trust boundary; failures in the adapter cannot corrupt motoko's session state | agent | implementation | low |
| Mirror Pi adapter's Dockerfile pattern (`Dockerfile.agent-motoko` = base image + binary install, no Go toolchain) | Pi precedent already proven in production; smallest viable cloud variant | agent | implementation | low |
| Models.yml entries use the `motoko-<model>` naming convention to keep paired-comparison rows obvious in leaderboards | Without the prefix, sorting and visual diffing breaks — it's the difference between immediate "harness lift" reading and manual cross-row comparison | agent | implementation | low |
| First release does NOT bind `ANTHROPIC_API_KEY` in the cloud Job (motoko-claude-* models are LOCAL-only) | Cost-control rule from EXECUTOR_SHAPE §8 — pi precedent: `agent_executor_pi` deliberately binds only `OPENAI_API_KEY` + `GEMINI_API_KEY`. Anthropic API-key billing is pay-per-token; a busy day = hundreds of dollars | human | design | high (changing this opens billing exposure that requires explicit policy) |
| Skip the `-go` Dockerfile variant (`Dockerfile.agent-motoko-go`) for v0.1 | Motoko doesn't compile Go inside the container; defer until a benchmark concretely requires it (per EXECUTOR_SHAPE §5 guidance) | agent | implementation | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Reuse session JSONL stream (no fork changes) — confirmed by today's review of motoko's logfile output (events: `session_start`, `native_tool_calls`, `cost_warning`, `dp7_verifier_rejected`, `done`)
- [x] Cost-control: cloud Job binds only OpenRouter + OpenAI + Gemini secrets (matches pi pattern)
- [x] Dockerfile is base-image + binary install style (single binary, no transitive runtime deps beyond what `agent-base` already provides)
- [ ] Naming: `motoko-<model>` vs `motoko/<model>` in `models.yml` keys — both visually distinct from vanilla rows; pick one before M3 (recommended: hyphen, matching `pi-claude-*` precedent)
- [ ] Whether the first release pins motoko_agent to a specific commit (yes, recommended) or floats `main` (gambles on stability) — recommend pinning per the M-BENCH-MOTOKO doc's risk #3 ("self-referential masking")

---

## Deferred Decisions

The following are intentionally left open for the implementer:

- Internal helper function naming inside `internal/executor/motoko/motoko.go` — agent may choose
- Test fixture organization under `internal/executor/motoko/testdata/` — agent may choose (recommend mirroring `internal/executor/opencode/testdata/` since both parse top-level NDJSON events with sub-events)
- Exact JSONL fields surfaced as `ProviderData` (above and beyond `tokens`/`cost`/`finish_reason`) — agent may choose; safe default is to round-trip everything not explicitly handled
- Whether to emit `dp7_verifier_rejected` as a per-turn telemetry event (would be a strong cross-harness signal — "motoko's gate fired N times before the model converged") — human at review
- Per-step span emission via `telemetry.StartSpan(... "exec.turn", ...)` — match gemini's pattern (one span per agent turn) — agent may choose

---

## Solution Design

### Overview

Adopt the EXECUTOR_SHAPE.md two-pillar contract verbatim. Pillar 1 (local executor) is a new package `internal/executor/motoko/` mirroring `internal/executor/opencode/` (closest structural analogue: both parse top-level NDJSON events from a long-running CLI). Pillar 2 (cloud) is a `Dockerfile.agent-motoko` mirroring `Dockerfile.agent-pi` (closest deployment analogue: multi-provider CLI, no Go toolchain in the container).

The adapter spawns `motoko "<task>"` as a subprocess with `WORKDIR`, `MODEL`, and `MOTOKO_CONFIG` env vars. It tails motoko's session JSONL log at `${WORKDIR}/.motoko/logfile/session_<id>.jsonl`, parses events of interest, and produces a `Result` struct conforming to the `executor.Executor` interface. No fork-side changes to motoko_agent are required for v0.1 — motoko already emits everything we need.

The strategic payoff is the **paired-comparison row** in agent-suite output: `motoko-haiku-4-5` next to `claude-haiku-4-5`, same task, same model, with-vs-without motoko's harness. The delta is the empirical answer to "does the harness lift cheap models above the AILANG-correctness threshold?"

### Architecture

**Components:**

1. **`internal/executor/motoko/motoko.go`** (~600 LOC) — CLI driver
   - `MotokoExecutor` struct implementing `executor.Executor` interface (all 7 methods)
   - `New(cfg *executor.Config)` constructor reading `cfg.MotokoPath` / `cfg.MotokoModel` / `cfg.MotokoProfile`
   - `Register()` + `init()` per EXECUTOR_SHAPE §2 contract
   - `Execute` (synchronous) and `ExecuteStreaming` (NDJSON event handler) methods
   - `parseSessionJSONL` — line-by-line parser with `json.RawMessage` payloads + `ProviderData` for unknown fields
   - `HealthCheck` — runs `motoko --version` (or equivalent) to verify the binary is on PATH
   - `CostModel` — derived from motoko's per-step `cost_warning` events + final `done` total

2. **`internal/executor/motoko/motoko_test.go`** (~400 LOC) — tests
   - Registration test (`init()` registers with factory; `Register()` is idempotent)
   - Fixture-driven parser test (replay a captured session JSONL, assert token counts + finish reason)
   - Mock-binary test (POSIX shell stand-in emits canned NDJSON; full streaming path exercised)
   - Gated live test `TestLiveRun_Motoko` — skips unless `AILANG_MOTOKO_LIVE=1` set AND `motoko` on PATH
   - HealthCheck test (positive with mock; negative with bad path)

3. **`internal/executor/motoko/README.md`** — flags, auth, event schema, cost model, known limits, trust boundary

4. **`internal/executor/motoko/testdata/`** — captured session JSONL fixtures (3-5 representative runs: success, failure, dp7-rejected, cost-exhausted)

5. **Pillar 1 wiring** (~5 LOC outside the package):
   - `internal/coordinator/provider_executor.go`: one blank-import line `_ "github.com/sunholo-data/ailang/internal/executor/motoko"`
   - `internal/eval_harness/models.yml`: 4 new model entries (`motoko-claude-haiku-4-5`, `motoko-claude-sonnet-4-6`, `motoko-glm-5`, `motoko-gemma-4`) with `agent_cli: motoko`
   - `internal/eval_harness/models.yml`: add the 4 entries to the `agent_suite` composite
   - `internal/executor/factory.go::Config` struct: add `MotokoPath` / `MotokoModel` / `MotokoProfile` fields

6. **Pillar 2 deployment** (~80 LOC + 2 ailang-multivac PRs):
   - `docker/Dockerfile.agent-motoko` (~10 lines, mirroring `Dockerfile.agent-pi`)
   - `internal/dispatch/cloudrun/dispatcher.go::knownVariants` — add `"motoko"` so the coordinator accepts `--executor-variant motoko`
   - `ailang-multivac/cloudbuild.yaml`: add `build-agent-motoko` + `push-agent-motoko` steps; update `push-images.waitFor` and top-level `images:` lists
   - `ailang-multivac/cloudbuild-images.yaml`: SAME — both files MUST stay in sync per EXECUTOR_SHAPE §6 (the historical drift between these two files for several months is exactly the failure mode the contract guards against)
   - `ailang-multivac/terraform/cloud_run_jobs.tf`: one `agent-motoko` Cloud Run Job block, secret bindings = `OPENROUTER_API_KEY` + `OPENAI_API_KEY` + `GEMINI_API_KEY` (NO `ANTHROPIC_API_KEY` per Design Freeze)

### Implementation Plan

**Phase 1: Pillar 1 Foundation** (~6 hours)
- [ ] M1: Scaffold `internal/executor/motoko/motoko.go` with `MotokoExecutor` struct + all 7 `executor.Executor` interface methods (return `errors.New("not implemented")` stubs initially)
- [ ] M2: Implement `New` constructor + `Register()` + `init()` per EXECUTOR_SHAPE §2; verify factory registration via `TestRegistration_Motoko`
- [ ] M3: Implement subprocess spawn in `Execute` (build args from cfg; pass `MODEL`/`MOTOKO_CONFIG`/`WORKDIR` env; capture stdout/stderr; wait for exit)
- [ ] M4: Implement `parseSessionJSONL` — locate `.motoko/logfile/session_*.jsonl` after the run completes; line-by-line parse with `json.RawMessage`; map known events to `Result` fields; round-trip unknowns into `ProviderData`
- [ ] M5: Implement `ExecuteStreaming` — same as Execute but tails the JSONL during the run via a goroutine + handler callback
- [ ] M6: Implement `HealthCheck` — invoke `motoko --version` (or fallback: check binary exists at configured path); positive + negative tests

**Phase 2: Pillar 1 Wiring + Tests** (~4 hours)
- [ ] M7: Add `MotokoPath`/`MotokoModel`/`MotokoProfile` to `executor.Config`; default `MotokoPath = "motoko"`
- [ ] M8: Capture 3-5 fixture JSONL files from real motoko runs (success, failure, dp7-rejected, cost-exhausted); commit to `testdata/`
- [ ] M9: Write fixture-replay test, mock-binary test, gated live test; achieve ≥80% coverage on `motoko.go`
- [ ] M10: Add blank import to `internal/coordinator/provider_executor.go`
- [ ] M11: Add 4 model entries to `models.yml` + add to `agent_suite` composite
- [ ] M12: Write `internal/executor/motoko/README.md` per EXECUTOR_SHAPE §1 schema (flags, auth, schema, limits, trust boundary noting motoko's autonomous bash tool)

**Phase 3: Pillar 2 Cloud Deployment** (~4 hours)
- [ ] M13: Author `docker/Dockerfile.agent-motoko` (FROM `agent-base`, USER root, install motoko binary [build-from-source OR vendor a release tarball — agent may choose], USER ailang); verify `docker build` + `motoko --version` locally
- [ ] M14: Add `"motoko"` to `knownVariants` in `internal/dispatch/cloudrun/dispatcher.go`
- [ ] M15: Open ailang-multivac PR #1: add `build-agent-motoko` + `push-agent-motoko` to BOTH `cloudbuild.yaml` AND `cloudbuild-images.yaml`; update `push-images.waitFor` + `images:` lists in both files
- [ ] M16: Open ailang-multivac PR #2: add `agent-motoko` Cloud Run Job to `terraform/cloud_run_jobs.tf` with `OPENROUTER_API_KEY` + `OPENAI_API_KEY` + `GEMINI_API_KEY` bindings (NO Anthropic per Design Freeze)
- [ ] M17: Smoke-test in dev: build pipeline → `terraform apply` → coordinator dispatch with `--executor motoko` → completion

**Phase 4: Threshold Measurement Run** (~2 hours)
- [ ] M18: Run `ailang eval-suite --executor motoko --models motoko-claude-haiku-4-5 --benchmarks <agent-tier suite>` locally; capture results
- [ ] M19: Run paired comparison: `motoko-claude-haiku-4-5` vs vanilla `claude-haiku-4-5` on the same suite; compute delta
- [ ] M20: Update CHANGELOG.md under `[Unreleased]` with M-MOTOKO-EXECUTOR-ADAPTER entry citing concrete paired-comparison numbers
- [ ] M21: Move design doc to `design_docs/implemented/v0_18_0/m-motoko-executor-adapter.md` with implementation report

### Files to Modify/Create

**New files (this repo):**
- `internal/executor/motoko/motoko.go` (~600 LOC) — core driver
- `internal/executor/motoko/motoko_test.go` (~400 LOC) — tests
- `internal/executor/motoko/README.md` (~150 lines) — flags, auth, schema, limits, trust boundary
- `internal/executor/motoko/testdata/session_success.jsonl` (~50 lines)
- `internal/executor/motoko/testdata/session_failure.jsonl` (~30 lines)
- `internal/executor/motoko/testdata/session_dp7_rejected.jsonl` (~40 lines)
- `internal/executor/motoko/testdata/session_cost_exhausted.jsonl` (~25 lines)
- `docker/Dockerfile.agent-motoko` (~10 lines) — Pillar 2 cloud variant

**Modified files (this repo):**
- `internal/coordinator/provider_executor.go` (+1 LOC) — blank import
- `internal/eval_harness/models.yml` (+~30 lines) — 4 new model entries + `agent_suite` membership
- `internal/executor/factory.go` (+~10 LOC) — `Config` struct gains `MotokoPath`/`MotokoModel`/`MotokoProfile`
- `internal/dispatch/cloudrun/dispatcher.go` (+1 LOC) — `"motoko"` in `knownVariants`
- `docs/internal/EXECUTOR_SHAPE.md` (+~3 lines) — add motoko to the canonical-references list
- `changelogs/v0.10-current.md` (+~30 lines) — `[Unreleased]` section M-MOTOKO-EXECUTOR-ADAPTER entry

**Modified files (`ailang-multivac` repo, separate PRs):**
- `cloudbuild.yaml` (+~25 lines) — `build-agent-motoko` + `push-agent-motoko` steps + waitFor/images updates
- `cloudbuild-images.yaml` (+~25 lines) — SAME (both files in sync per EXECUTOR_SHAPE §6)
- `terraform/cloud_run_jobs.tf` (+~50 lines) — `agent-motoko` Cloud Run Job block

---

## Examples

### Example 1: Local cross-harness comparison (the strategic measurement)

**Before (no motoko in the harness):**
```bash
$ ailang eval-suite --models claude-haiku-4-5 --benchmarks tier_a
# Single row: claude-haiku-4-5 → 61.4% pass rate
# No way to ask "what would motoko's harness do with the same model?"
```

**After (motoko adapter shipped):**
```bash
$ ailang eval-suite --models motoko-claude-haiku-4-5,claude-haiku-4-5 --benchmarks tier_a

Model                         | Pass | Fail | Cost   | Tokens
------------------------------|------|------|--------|-------
claude-haiku-4-5              | 25/41| 16   | $0.42  | 380K
motoko-claude-haiku-4-5       | 33/41|  8   | $1.18  | 920K   ← +20% lift, 2.8x cost
```

**Strategic read:** the harness lifts cheap-model AILANG correctness by 20pp at 2.8x cost. The cost-arbitrage thesis holds when `(claude-sonnet-4-6 cost / 2.8) > (claude-haiku-4-5 cost) AND (motoko-haiku pass% ≈ sonnet pass%)`. Without this measurement, that's a hypothesis; with it, a result.

### Example 2: Pillar 1 wiring — the entire blank-import diff

**Before (`internal/coordinator/provider_executor.go`):**
```go
import (
    _ "github.com/sunholo-data/ailang/internal/executor/claude"
    _ "github.com/sunholo-data/ailang/internal/executor/codex"
    _ "github.com/sunholo-data/ailang/internal/executor/gemini"
    _ "github.com/sunholo-data/ailang/internal/executor/opencode"
    _ "github.com/sunholo-data/ailang/internal/executor/pi"
)
```

**After:**
```go
import (
    _ "github.com/sunholo-data/ailang/internal/executor/claude"
    _ "github.com/sunholo-data/ailang/internal/executor/codex"
    _ "github.com/sunholo-data/ailang/internal/executor/gemini"
    _ "github.com/sunholo-data/ailang/internal/executor/motoko"  // <-- one line
    _ "github.com/sunholo-data/ailang/internal/executor/opencode"
    _ "github.com/sunholo-data/ailang/internal/executor/pi"
)
```

That single line + the package itself is the entire local wiring. `ExecutorProvider` auto-discovers via `NewExecutorProvider("motoko")`; no factory edits, no switch statements.

### Example 3: Pillar 2 — the entire Dockerfile

```dockerfile
# AILANG Agent Executor — motoko variant (motoko_agent, no Go toolchain)
# M-MOTOKO-EXECUTOR-ADAPTER: Use for AILANG-native agent runs against any
# OpenRouter-routable model. Use for: cross-harness comparisons (motoko-X vs
# vanilla-X) and open-weight model evaluation (Gemma 4, GLM-5, Qwen, etc.).
ARG PROJECT
FROM europe-west1-docker.pkg.dev/${PROJECT}/ailang/agent-base:latest

# motoko binary (built from sunholo-data/motoko_agent fork, pinned to a release tag)
USER root
ARG MOTOKO_VERSION=v0.6.0
RUN curl -fsSL "https://github.com/sunholo-data/motoko_agent/releases/download/${MOTOKO_VERSION}/motoko-linux-amd64" \
    -o /usr/local/bin/motoko \
    && chmod +x /usr/local/bin/motoko
USER ailang
```

10 lines. Mirrors `Dockerfile.agent-pi` structure exactly.

---

## Conflict Surface

This is **not** a parser/lexer/typechecker/codegen change — it's a new subprocess executor. Per EXECUTOR_SHAPE §1, the package is purely additive, with no modifications to compiler internals. The Conflict Surface section is therefore not strictly required.

However, the **executor framework** has its own surface that must be respected:

### Touchpoints in the executor framework

- `internal/executor/factory.go::Config` — adding fields is purely additive (other executors ignore them); no risk
- `internal/executor/factory.go::GlobalFactory().Register("motoko", ...)` — the registration map is keyed by string; collision with another `"motoko"` registration would be a build-time panic, not a silent bug
- `internal/coordinator/provider_executor.go` blank-import list — Go's import resolution is order-independent; no conflict possible

### Programs that MUST still work

The following must continue to pass after this change ships:

1. `ailang eval-suite --models claude-sonnet-4-6 --benchmarks tier_a` (vanilla executor unaffected)
2. `ailang eval-suite --models pi-haiku-4-5 --benchmarks tier_a` (Pi executor unaffected; both pi and motoko occupy the same NDJSON-parser pattern but different packages)
3. `ailang eval-suite --models agent_suite --benchmarks tier_a` (composite must include both old and new entries)
4. Cloud Run dispatch via the coordinator with `--executor opencode` (existing variant unaffected by the addition of `"motoko"` to `knownVariants`)
5. `make test` and `make ci` continue to pass (no test interference; new tests are isolated to `internal/executor/motoko/`)

These become regression test fixtures in M9 + M17.

### What deliberately changes

Nothing existing breaks. This is an additive change. The 4 new `models.yml` entries don't displace existing rows.

---

## Testing Strategy

**Unit tests** (`internal/executor/motoko/motoko_test.go`):
- `TestRegistration_Motoko` — `init()` registers `"motoko"` in the global factory; `Register()` is idempotent
- `TestParseSessionJSONL_Success` — replay `testdata/session_success.jsonl`; assert `Result.Success == true`, expected token counts, expected `cost_usd`
- `TestParseSessionJSONL_Failure` — replay `testdata/session_failure.jsonl`; assert `Result.Success == false`, populated `Result.Error`
- `TestParseSessionJSONL_DP7Rejected` — replay `testdata/session_dp7_rejected.jsonl`; assert the dp7 rejection event surfaces in `ProviderData` (or as a typed field if we choose to elevate it)
- `TestParseSessionJSONL_CostExhausted` — replay `testdata/session_cost_exhausted.jsonl`; assert `cost_exhausted` finish reason
- `TestParseSessionJSONL_TolerantToNonJSONLines` — interleave non-JSON preamble lines (motoko's TUI sometimes prints status before the JSONL starts); assert clean skip
- `TestExecute_MockBinary` — POSIX shell stand-in (`testdata/mock-motoko.sh`) emits canned NDJSON; full subprocess spawn + parse + `Result` round-trip
- `TestHealthCheck_Positive` — mock binary returns version string; `HealthCheck` returns nil
- `TestHealthCheck_Negative` — config with bad path; `HealthCheck` returns wrapped error
- `TestLiveRun_Motoko` (gated) — skips unless `AILANG_MOTOKO_LIVE=1` AND `motoko` is on PATH; runs a tiny canned task end-to-end against the real binary

**Integration tests:**
- `TestExecutorProvider_Motoko` (in `internal/coordinator/`) — `NewExecutorProvider("motoko")` succeeds after blank import (proves Pillar 1 §3 wiring)
- `TestEvalSuite_MotokoModelEntries` (in `internal/eval_harness/`) — parsing `models.yml` resolves all 4 motoko entries to `agent_cli: motoko` and they appear in `agent_suite` expansion

**Cloud smoke (manual after Pillar 2 lands):**
- `docker build -f docker/Dockerfile.agent-motoko --build-arg PROJECT=ailang-dev -t agent-motoko:dev .` succeeds
- `docker run --rm agent-motoko:dev motoko --version` returns the pinned version
- After `terraform apply` to dev: dispatch a coordinator task with `--executor motoko` and observe successful completion in Cloud Run logs

**Regression-surface tests** (per Conflict Surface above):
- `make test` covers all existing executor packages (claude, codex, gemini, opencode, pi); none should regress
- One paired smoke run before merge: `ailang eval-suite --models pi-claude-haiku-4-5,motoko-claude-haiku-4-5 --benchmarks <small-suite>` to prove both adapters operate side-by-side

**Coverage target:** ≥80% on `internal/executor/motoko/motoko.go`.

---

## Non-Goals

**Not in this feature:**
- Changes to motoko_agent's agent loop, extension system, or session JSONL schema — out of scope; adapter conforms to whatever motoko emits today
- A `Dockerfile.agent-motoko-go` variant — deferred until a benchmark concretely requires Go inside the container (per EXECUTOR_SHAPE §5 guidance)
- Self-modifying eval (motoko writing AILANG code that improves AILANG itself) — separate v0.19+ thought experiment; motoko_explore strategy doc §16
- M-BENCH-MOTOKO-EXTENSIONS (`motoko-with-context_mode`, `motoko-with-omnigraph`, etc. as separate `models.yml` entries to benchmark per-extension impact) — deferred to v0.19; would need motoko-side flag to disable extensions selectively
- Anthropic API-key binding in the cloud Job — explicitly excluded by Design Freeze cost-control rule; motoko-claude-* models stay LOCAL-only for now

---

## Timeline

**Day 1** (~8 hours):
- Phase 1 (M1–M6): Pillar 1 foundation — package skeleton through HealthCheck (~6h)
- Phase 2 start (M7–M9): Config wiring + fixture capture from real runs (~2h)

**Day 2** (~6 hours):
- Phase 2 finish (M10–M12): blank import + models.yml + README (~3h)
- Phase 3 start (M13–M14): Dockerfile + knownVariants (~3h)

**Day 3** (~4 hours):
- Phase 3 finish (M15–M17): cloudbuild + terraform PRs in ailang-multivac + dev smoke (~3h)
- Phase 4 (M18–M21): threshold measurement run + CHANGELOG + move-to-implemented (~1h, may extend if measurement reveals interesting numbers worth analyzing)

**Total: ~18 hours across 3 working days.**

Delivery as a single sprint executed by sprint-executor against this design doc; sister sprint plan to follow at `m-motoko-executor-adapter-sprint-plan.md`.

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Motoko's session JSONL schema drifts (an event field renamed, new required field added) | Medium | Per-package parser with `json.RawMessage` + `ProviderData` round-trip absorbs shape changes silently; only fields we explicitly map can break a fixture, and fixture failures are obvious in CI. Pin motoko_agent commit per release (Day 0 record in CHANGELOG). |
| Motoko's `done` event missing on crash | Medium | Adapter treats absence-of-`done`-by-deadline as `Success: false, Error: "motoko did not emit done event"`. Same pattern as opencode's missing-final-event handling. |
| OpenRouter cost-normalization regression for open-weight models (Gemma 4 / GLM-5) zeros out `cost_usd` | Low | Assert `cost_usd > 0` in fixture tests for open-weight runs; alert in CI if any motoko fixture loses its cost number. M-AI-OPENROUTER's pricing path is the load-bearing piece. |
| Self-referential masking — an AILANG regression breaks motoko's loop, motoko's benchmark numbers crater across the board, masks the *agent quality* signal | High | Pin motoko_agent to a specific commit per AILANG release (recorded in CHANGELOG.md and surfaced in `Result.ProviderData["motoko_commit"]`); rebase forward only between AILANG releases. Per M-BENCH-MOTOKO doc Risk #3. |
| Cloud Build pipeline drift between `cloudbuild.yaml` and `cloudbuild-images.yaml` (a new variant added to one but not the other) | Medium | EXECUTOR_SHAPE §6 explicitly calls this out as the historical failure mode; M15 acceptance criterion requires both files updated in the same PR. |
| Surprise Anthropic billing if `ANTHROPIC_API_KEY` accidentally bound to the cloud Job | High | Design Freeze rule: cloud Job binds only OpenRouter + OpenAI + Gemini secrets. motoko-claude-* models stay LOCAL-only. Pi precedent (`agent_executor_pi` deliberately omits Anthropic). |
| Motoko's autonomous bash execution from inside the cloud container | Medium | Cloud trust boundary: per-Job ephemeral container, network egress allowlist (per existing Cloud Run Job network policy), workspace under per-run tmpdir. Documented in `internal/executor/motoko/README.md` "Trust Boundary" section. |

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|---|---|---|
| A1: Determinism | 0 | No new nondeterminism introduced; subprocess spawning is already a deterministic boundary in the executor framework |
| A2: Replayability | +1 | Captured session JSONL fixtures are checked-in and replayable; eval runs produce reproducible `result.json` artifacts |
| A3: Effect Legibility | +1 | Adapter is a pure subprocess executor — all effects (file IO on `.motoko/`, process spawn, network via the spawned binary) are visible at the package boundary |
| A4: Explicit Authority | 0 | No new ambient authority; spawned binary inherits whatever the parent process has (matches all other executor adapters) |
| A5: Bounded Verification | +1 | Per-package parser + per-package fixtures — verification is local; a schema change in motoko cannot affect codex/gemini/etc. |
| A6: Safe Concurrency | 0 | Adapter spawns one subprocess per `Execute` call; no shared mutable state between concurrent calls |
| A7: Machines First | +1 | Cross-harness comparison rows in `models.yml` are mechanically diffable (`motoko-X` vs `X` is sortable + parseable); enables tooling to read the harness-lift signal |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | +1 | Per-step `cost_warning` events are surfaced in `Result.Cost`; cross-harness comparison enables empirical "$ per success" measurement |
| A10: Composability | +1 | Composes with existing executor framework, eval-suite, agent_suite, cloud dispatch — zero changes to any of those layers (per EXECUTOR_SHAPE auto-discovery contract) |
| A11: Structured Failure | +1 | `dp7_verifier_rejected` and `cost_exhausted` map to typed `Result.FinishReason` values; `done` with `error` field maps to `Success: false, Error: ...`; absence-of-`done`-by-deadline is a typed timeout |
| A12: System Boundary | 0 | Subprocess + filesystem boundary already established by other executors; no new boundary |

**Net Score: +7** ✅ Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced — subprocess spawning is the deterministic boundary
- [x] A3 (Effects): All side effects visible at the package boundary (process spawn, file IO on `.motoko/`, network via spawned binary)
- [x] A4 (Authority): No ambient authority granted; subprocess inherits parent's authority (matches existing executor pattern)
- [x] A7 (Machines First): Optimizing for mechanical diffability (paired-comparison rows), not human convenience

---

## References

- **Source proposal:** msg `5f2facd3` from motoko-explore (2026-05-07) — "Sprint proposal: add motoko as executor adapter in AILANG eval harness (alongside claude/pi/codex/opencode)"
- **Canonical contract:** [`docs/internal/EXECUTOR_SHAPE.md`](../../../docs/internal/EXECUTOR_SHAPE.md) — two-pillar executor recipe (this design doc is structured strictly around this contract)
- **Pillar 1 precedent:** [`design_docs/implemented/v0_15_0/m-exec-expand-codex-opencode.md`](../../implemented/v0_15_0/m-exec-expand-codex-opencode.md) + [sprint plan](../../implemented/v0_15_0/m-exec-expand-codex-opencode-sprint-plan.md) — added codex + opencode local executors; this is where the auto-discovery contract was hardened
- **Pillar 1 closest analogue:** [`internal/executor/opencode/opencode.go`](../../../internal/executor/opencode/opencode.go) (601 LOC, NDJSON parser with sub-events) — adopt as structural template
- **Pillar 2 precedent:** [`design_docs/implemented/v0_14_2/m-exec-pi-harness.md`](../../implemented/v0_14_2/m-exec-pi-harness.md) — Pi adapter, first design doc structured around the two-pillar recipe; CLI-only Dockerfile pattern (no Go toolchain) is the closest model for motoko
- **Pillar 2 closest analogue:** [`docker/Dockerfile.agent-pi`](../../../docker/Dockerfile.agent-pi) — 10-line CLI install pattern; mirror exactly
- **Variant framework:** [`design_docs/planned/v1_1_0/m-executor-variants.md`](../v1_1_0/m-executor-variants.md) — per-executor Docker variants + Cloud Run Jobs framework
- **Superseded predecessor:** [`design_docs/planned/v0_17_0/m-bench-motoko-executor.md`](../v0_17_0/m-bench-motoko-executor.md) (M-BENCH-MOTOKO, 2026-05-04, 444 lines) — predates EXECUTOR_SHAPE formalization and the M-MOTOKO-EXTENSION-INTEGRATION sprint; this doc supersedes it
- **Related strategy:** motoko_explore strategy doc §15 (threshold-impact framing) — `motoko_explore/design_docs/STRATEGY-AILANG-MOTOKO-NICHE.md` (commit 49fa213)
- **Related implemented:** M-AI-OPENROUTER (v0.14.x) — load-bearing dependency for motoko's cost-normalization path; M-MOTOKO-EXTENSION-INTEGRATION (v0.17.1, today) — published the 9 motoko-ext-* packages this benchmark exercises
- **Axiom reference:** [Design Axioms](/docs/references/axioms)

---

## Future Work

Features that build on this:
- **M-BENCH-MOTOKO-EXTENSIONS** (proposed v0.19) — add `motoko-with-context_mode`, `motoko-with-omnigraph`, etc. as separate `models.yml` entries to benchmark per-extension contribution to the harness lift
- **M-BENCH-MOTOKO-OPEN-WEIGHTS** (proposed v0.19) — expand the open-weight model row coverage (Qwen, Llama, Mistral) once GLM-5 and Gemma 4 prove the routing path
- **M-MOTOKO-DOGFOOD-CI** (proposed v0.19) — wire the motoko adapter into AILANG's own CI so AILANG-side regressions that affect motoko surface in our PR checks, not only in motoko's CI
- **M-MOTOKO-CLAUDE-CLOUD** (proposed v0.20) — re-evaluate binding `ANTHROPIC_API_KEY` to the motoko Cloud Run Job once we have a per-Job cost cap policy in place (currently blocked by EXECUTOR_SHAPE §8 cost-control rule)
- **Self-modifying eval** — motoko writes AILANG code that improves AILANG itself — strategic v0.20+ thought experiment from motoko_explore strategy §16
