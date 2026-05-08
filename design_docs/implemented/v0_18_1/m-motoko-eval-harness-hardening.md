# M-MOTOKO-EVAL-HARNESS-HARDENING: Close all 10 integration gaps surfaced by live smoke

**Status**: Implemented (2026-05-08)
**Target**: v0.18.1 (patch release on top of v0.18.0's M-MOTOKO-EXECUTOR-ADAPTER)
**Priority**: P0 (High — blocks M5 threshold-measurement; without this, every motoko eval number is suspect)
**Estimated**: 2–3 days (~300 LOC across both repos; cognitive load > LOC count due to cross-repo investigation)
**Actual**: ~6 hours wall-clock, ~330 LOC + 11 new tests, single session

**Result**: 5 of 7 acceptance-gate conditions met. Two (CostUSD>0 + end-to-end smoke success) blocked on a separate Bedrock validation issue (extension tool names with `/` fail Anthropic's `^[a-zA-Z0-9_-]{1,128}$` pattern) — pricing env-var plumbing verified by unit tests; live smoke needs the extension fix downstream.

**Commits (cross-repo):**
- AILANG `dev`: `20f97b00` (M2c HealthCheck), `06c28546` (M3a-c parser fallback + repo discovery), `a5d33677` (M5a-c pricing env-var passthrough)
- motoko_agent `motoko-bisect-gap1` (PR target): `7997c8f` (M1c JSONL drain), `faada61` (M2a profile mirror), `7d595a4` (M2b/M2c CLI flags), `1055976` (M4a-c session_id unification), `77e7de0` (M5b cost env override)
**Dependencies**:
- ✅ M-MOTOKO-EXECUTOR-ADAPTER (v0.18.0) — Pillar 1 + in-repo Pillar 2 shipped (commits `bc15bb8e` + `74ebf181` on AILANG dev)
- ✅ M-MOTOKO-EVAL-INSTRUMENTATION (motoko commits `0c006be` + `84fa449`) — schema v1 JSONL contract
- ✅ Today's progressive smoke fixes: `dc1f4eea` (HealthCheck + MOTOKO_REPO fallback), `83fb6cf` (motoko MOTOKO_HEADLESS), `cc5bc1f` (run_summary-before-done reorder). These are PARTIAL — this design completes them.

**Author**: Claude Sonnet 4.6 + Mark
**Created**: 2026-05-08

**Source event**: Live smoke testing on 2026-05-08 surfaced 10 interconnected integration gaps. User feedback: "we need it all I think. lets get to the bottom of the gaps - I think a design doc process will help."

---

## Problem Statement

The v0.18.0 adapter ships structurally correct code that passes 22+ tests against fixtures. The first live run against real motoko revealed that fixtures and reality diverge in 10 ways — all interconnected, all blocking trustworthy benchmark numbers.

**Today's smoke run** (after partial fixes) produces:

```
Success:      false        ← gap #2 (gated on missing run_summary)
Error:        motoko terminated without emitting run_summary (likely crash)
SessionID:    session_f00db4cf-dfe9-4810-a335-f4251744df08  ← gap #4 (3-way mismatch)
Turns:        2            ← real
Tool calls:   1            ← real (BashExec ran successfully)
Input tok:    2502         ← real
Output tok:   93           ← real
Cost USD:     $0.000000    ← gap #3 (no cost_rates)
```

The numbers that flow through ARE accurate. The numbers that don't are missing because of upstream architectural issues in motoko_agent + the AILANG-side adapter making fragile assumptions.

**Why this is P0**: M5 of M-MOTOKO-EXECUTOR-ADAPTER (the threshold-measurement experiment that motivates the whole sprint family — does motoko's harness lift cheap models?) cannot produce trustworthy numbers without these fixes. We'd report `motoko-claude-haiku-4-5: 0% pass rate, $0 cost` and that's wrong on both axes.

### The 10 Gaps (Inventory)

| # | Gap | Surface | Severity |
|---|---|---|---|
| 1 | `run_summary` doesn't reach disk on success path | motoko AILANG | **Critical** (blocks 2,3) |
| 2 | `Result.Success` over-gated on run_summary presence | AILANG adapter | High |
| 3 | `Result.CostUSD` always $0 (missing cost_rates) | motoko config | High |
| 4 | Three competing session_id schemes (adapter / TS / AILANG) | both | Medium |
| 5 | `MOTOKO_REPO` env var must be set manually for adapter to find JSONL | AILANG adapter | Medium |
| 6 | `loaded_extensions: []` reported even when ailang.toml lists 7 | motoko AILANG | Medium |
| 7 | No `--headless` CLI flag (only env var) | motoko TS wrapper | Low |
| 8 | No version querying (no `--version` mode) | motoko TS wrapper | Low |
| 9 | Cost rates configuration burden (per-profile per-model) | motoko config | Medium |
| 10 | TS-side `process.exit(0)` on `done` forces fragile emission ordering | motoko TS | High (architectural) |

These are interconnected — fixing #1 unblocks #2; fixing #10 makes #1 robust; fixing #9 makes #3 trivial; fixing #4 simplifies #5; etc.

---

## Goals

**Primary Goal:** All 10 gaps closed in a coordinated way so the next live smoke produces a Result with every field populated correctly: `Success=true, CostUSD>0, NumTurns>0, InputTokens>0, OutputTokens>0, ProviderData[motoko_finish_reason]="stop", ProviderData[motoko_commit]=<sha>, all 3 session_ids match`.

**Success Metrics:**
- A single end-to-end smoke (`ailang eval-suite --models motoko-claude-haiku-4-5 --benchmarks <small-tier>`) produces `Success=true` AND `CostUSD>0` AND non-zero token counts
- Re-run M5 paired comparison `motoko-claude-haiku-4-5 vs claude-haiku-4-5`, both rows produce real numbers, delta computable in pass-rate / cost-USD / token-count axes
- Adapter HealthCheck reports motoko version (e.g. "motoko 0.6.x at /Users/.../motoko, commit 84fa449") in <100ms
- `MOTOKO_REPO` env var is OPTIONAL — adapter discovers it automatically by reading the wrapper script
- Full `Result` struct asserted by a new integration test (`TestEndToEnd_FullResultPopulation`) — not just "no errors", but every field has the expected shape
- `loaded_extensions` field accurately reflects what's actually loaded (matches the registry_generated.ail extension list)

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|---|---|---|---|---|
| Single canonical session_id flowing adapter → TS wrapper → AILANG | Three different IDs in one run is operational chaos; observability tools need ONE id to correlate spans, JSONL, eval results, cost data | human | design | high (touches all three layers) |
| `models.yml.pricing` is the single source of truth for cost_rates; motoko profile inherits at runtime | Eliminates the per-profile-per-model config burden; one place to add a new model | human | design | med (motoko profile loader change) |
| Success criteria: `run_summary.finish_reason == "stop"` PREFERRED, last `thinking.finish_reason == "stop"` FALLBACK | run_summary may legitimately be missing (crash mid-flush); should not falsely report success-as-failure | human | design | low |
| TS wrapper waits for runtime stdout EOF to exit (NOT `process.exit(0)` on done) | Eliminates the emission-ordering fragility (gap #10); means future emitters don't need to remember to write run_summary BEFORE done | human | design | high (changes process lifecycle semantics) |
| motoko `--version` mode added at the WRAPPER level, not the agent loop | Wrapper can short-circuit before invoking the AILANG runtime; matches what every other CLI does (claude / pi / opencode) | agent | implementation | low |
| MOTOKO_REPO discovery via parsing the wrapper script's installed location | More robust than env var (works for any user / any install path); matches how `which X` + parsing-config patterns work for other CLIs | agent | implementation | low |
| Allow per-environment cost_rates override (eval = ailang models.yml; interactive = motoko profile) | Eval needs strict cost-tracking; interactive users may want to override for testing | agent | implementation | low |
| Investigation-first ordering for gap #1 — debug instrumentation BEFORE writing the fix | gap #1's root cause is unknown; writing a fix without knowing the cause is gambling | human | design | low (but blocks all other phases until known) |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Adapter is the canonical session_id source (it spawns motoko; it knows the task ID)
- [x] models.yml.pricing is the canonical cost_rates source for eval-harness invocations
- [x] TS wrapper exits on runtime EOF, not on done event (lifecycle change)
- [x] gap #1 investigation precedes any code change in that area (don't gamble)
- [ ] Whether motoko's `--version` should report just the wrapper version OR also the AILANG runtime + extension versions — pick before M3 (recommended: full triplet for diagnostic clarity)
- [ ] Whether session_id from the adapter should overwrite motoko's filename (i.e. wrapper writes to `${MOTOKO_SESSION_ID}.jsonl`) OR adapter renames after the run — pick before M5 (recommended: wrapper honors env var; cleaner than post-hoc rename)

---

## Deferred Decisions

The following are intentionally left open for the implementer:

- The exact debug instrumentation approach for gap #1 (binary-search via emit_event markers vs OTEL trace inspection vs pre-existing AILANG_TRACE=deep instrumentation) — agent may choose; recommend emit_event markers for the simplest reproducer
- The format of `--version` output (`motoko 0.6.0\nailang v0.17.x\nmotoko-ext-* (7 packages)`) vs JSON — agent may choose; plain-text is simpler, JSON is mechanically parseable
- Whether to remove the deprecated motoko-side cost_rates profile config OR leave it for back-compat — agent may choose; recommend leaving with deprecation comment
- The exact name for the test that asserts full Result correctness (`TestEndToEnd_FullResultPopulation` vs `TestMotokoAdapter_LiveSmoke` etc.) — agent may choose
- Whether to backfill cost_rates for the 4 already-published v0.18.0 motoko models (`motoko-claude-haiku-4-5` etc.) into models.yml in this sprint — recommend yes (these need to work for M5)

---

## Solution Design

### Overview

Three-layer cleanup, sequenced as investigation → motoko-side fixes → AILANG-side fixes → cross-cutting unification → config layer → end-to-end validation. The investigation phase (Phase 1) is non-negotiable for gap #1 because writing a fix without knowing the cause is gambling.

The architectural posture: **eliminate fragile assumptions at every layer**. Today's adapter assumes:
- Wrapper preserves `MOTOKO_SESSION_ID` for the filename (it doesn't — gap #4)
- AILANG-side `derive_session_id()` produces the same value as adapter's env var (it doesn't — gap #4)
- The wrapper writes JSONL at `WORKDIR/.motoko/logfile/` (it doesn't — gap #5)
- `run_summary` always reaches disk (it doesn't — gaps #1 + #10)
- Profile cost_rates are configured (they aren't — gap #3)
- `loaded_extensions` JSONL field is accurate (it isn't — gap #6)

After this hardening, none of those assumptions are needed; each is replaced with an explicit, observable contract.

### Architecture

**Component changes by gap:**

#### Gap #1 (motoko AILANG — investigation-first)

Add temporary debug emit_event markers in `agent_loop_v2.ail`'s success path:
```ailang
let _ = emit_event(session_id, "debug:checkpoint", [kv("phase", js("pre_intercept"))]);
match dispatch_response_intercept(rt, ctx, result.message.content) {
  ...
  NoIntercept => {
    let _ = emit_event(session_id, "debug:checkpoint", [kv("phase", js("post_intercept"))]);
    if result.finish_reason != "tool_calls" then {
      let _ = emit_event(session_id, "debug:checkpoint", [kv("phase", js("pre_solver"))]);
      ...
    }
  }
}
```

Run the smoke and bisect. The last debug:checkpoint in the JSONL identifies the hang point. Suspected: dispatch_response_intercept blocking on an unloaded extension's hook lookup (gap #6 might be related — empty extensions should default to NoIntercept but maybe the lookup itself fails).

Once root-caused, the fix is targeted (~5-20 LOC). Remove the debug markers in the same commit.

#### Gap #2 (AILANG adapter — success criteria)

`internal/executor/motoko/parser.go::parseSessionJSONL`:
```go
// CURRENT: gates on run_summary
if !gotRunSummary {
    res.Success = false
    res.Error = "..."
}
// NEW: prefer run_summary, fall back to last thinking event finish_reason
switch {
case gotRunSummary:
    // existing path — authoritative
case lastThinkingFinishReason == "stop":
    res.Success = true
    res.Output = lastDoneOutput // may be empty if done not emitted; tolerate
case lastThinkingFinishReason == "tool_calls":
    // loop exited mid-step; this IS a failure (incomplete run)
    res.Success = false
    res.Error = "motoko exited after tool_calls finish_reason without resolving"
default:
    res.Success = false
    res.Error = "..."
}
```

#### Gap #3 + #9 (cost_rates source-of-truth)

models.yml already has `pricing` blocks for every motoko-* model. Make motoko's profile loader read them at runtime via env var passed by the adapter:

```go
// Adapter side (motoko.go ExecuteStreaming):
env = append(env,
    "MOTOKO_COST_INPUT_PER_1K="+fmt.Sprintf("%.6f", costModel.InputTokenCost),
    "MOTOKO_COST_OUTPUT_PER_1K="+fmt.Sprintf("%.6f", costModel.OutputTokenCost),
    "MOTOKO_COST_CACHE_READ_PER_1K="+fmt.Sprintf("%.6f", costModel.CacheReadCost),
)
```

```ailang
-- motoko AILANG side (config loader):
-- Override profile cost_rates with env-supplied values when present.
-- This makes models.yml the canonical source for eval invocations
-- without touching motoko's profile config files.
```

Result: adding a new model to models.yml automatically gives it correct cost tracking through motoko. No per-profile config update needed.

#### Gap #4 (session_id unification)

**Adapter is canonical**: AILANG adapter generates a session_id (UUID-based today, e.g. `session_<task_id>`), passes it via `MOTOKO_SESSION_ID` env var. Both motoko TS wrapper AND AILANG runtime honor it.

- TS wrapper change: when constructing the session JSONL filename, use `${MOTOKO_SESSION_ID}.jsonl` if set, else fall back to today's ISO-timestamp scheme.
- AILANG-side `derive_session_id()` already reads `MOTOKO_SESSION_ID` (we wrote that today). Confirm it propagates to all event emissions — verify `session_id` field on every event matches the env var.

After: filename, JSONL `session_id` field, adapter's tracking ID — all three identical.

#### Gap #5 (MOTOKO_REPO discovery)

`internal/executor/motoko/motoko.go`: add `discoverMotokoRepo()` helper that:
1. Resolves the motoko binary path via `exec.LookPath`
2. Reads the file (it's a bash wrapper)
3. Parses the line `MOTOKO_REPO="${MOTOKO_REPO:-/path}"` to extract the default
4. Caches the result on the executor struct

Used by `findSessionJSONL` instead of relying on `os.Getenv("MOTOKO_REPO")`. Falls back to the env var if parse fails (back-compat with manual override).

#### Gap #6 (extension loading visibility)

Investigate motoko's session_start emission — does `loaded_extensions` come from runtime introspection (real) or a hardcoded `[]`? Looking at the JSONL output:
```json
{"type":"session_start","loaded_extensions":[],...}
```

This is from `agent_loop_v2.ail::run_v2_with_conversation` (or wherever session_start fires). It needs to read `rt.registry.hooks` and report each hook's `id` field. Currently looks like it's hardcoded.

Fix: emit the actual hook list from `rt.registry`:
```ailang
let extension_ids = collect_hook_ids(rt.registry.hooks);
let _ = emit_event(session_id, "session_start", [
    ...
    kv("loaded_extensions", ja(extension_ids))
]);
```

Investigation needed: confirm whether the `hooks` list is empty (extensions not loading — bigger bug) or populated but not surfaced (emission bug). Run `ailang test src/core/ext/runtime.ail` to verify the registry parser works.

#### Gap #7 (--headless flag)

In motoko's wrapper script:
```bash
HEADLESS=""
if [[ "$1" == "--headless" ]]; then
    HEADLESS="1"
    shift
fi
export MOTOKO_HEADLESS="${MOTOKO_HEADLESS:-$HEADLESS}"
```

AILANG side already reads `MOTOKO_HEADLESS` env var (no change needed).

#### Gap #8 (--version mode)

Wrapper-level short-circuit BEFORE invoking AILANG runtime:
```bash
if [[ "$1" == "--version" ]]; then
    echo "motoko ${MOTOKO_VERSION:-dev}"
    echo "ailang $(ailang --version 2>/dev/null | head -1)"
    echo "extensions $(grep -c '^  "sunholo' "$MOTOKO_REPO/ailang.toml")"
    exit 0
fi
```

Adapter `HealthCheck` updated to call `motoko --version` — captures version into `Result.ProviderData["motoko_version"]` for pinning verification.

#### Gap #10 (process lifecycle — TS waits for EOF)

`src/tui/src/index.ts` PlainLogger / JsonlLogger:
```typescript
// REMOVE: process.exit(0) on done
// ADD: track that done was seen; let the runtime EOF callback handle exit
let receivedDone = false;
case "done":
    process.stdout.write(...);
    receivedDone = true;
    // Don't exit — wait for runtime to close stdout naturally after run_summary
    break;
```

`runtime-process.ts::child.on("close")` already fires on runtime EOF and triggers TUI exit. The change is removing the premature exit hooks in the loggers.

Once this is in, gap #1's run_summary emission happens in a window where the runtime knows the TS layer is still listening — no more truncation race.

### Implementation Plan

**Phase 1: Investigation (~3 hours)** — gap #1 root cause
- [ ] M1a: Add debug:checkpoint emit_event markers around dispatch_response_intercept, dispatch_solver_candidate, dp7_gate, emit_run_summary, emit_event("done"). Commit on a branch named `motoko-bisect-gap1`.
- [ ] M1b: Run smoke 3x; bisect the last-emitted checkpoint to identify the hang phase. Document findings inline in commit message.
- [ ] M1c: Once root-caused, write the targeted fix (~5-20 LOC). Remove all debug markers. Commit as `fix(motoko): gap #1 — <root cause summary>`.

**Phase 2: motoko-side fixes (~6 hours)** — gaps #1 (fix) + #6 + #7 + #8 + #10
- [ ] M2a: gap #6 — investigate loaded_extensions reporting; fix emission to read from `rt.registry.hooks`
- [ ] M2b: gap #7 — wrapper `--headless` flag with env-var fallback
- [ ] M2c: gap #8 — wrapper `--version` mode (printed triplet: motoko / ailang / extension-count)
- [ ] M2d: gap #10 — TS PlainLogger + JsonlLogger remove `process.exit(0)` on done/error; rely on runtime EOF
- [ ] M2e: Verify reorder commit `cc5bc1f` is now safe to revert (run_summary can come AFTER done if TS doesn't exit) — leave the safer ordering in place but note it's belt-and-braces

**Phase 3: AILANG-side fixes (~4 hours)** — gaps #2 + #5
- [ ] M3a: gap #2 — parseSessionJSONL success-criteria refactor with thinking.finish_reason fallback
- [ ] M3b: gap #5 — discoverMotokoRepo() reads wrapper script; remove env-var fallback's primacy
- [ ] M3c: Update parser tests for the new success-criteria fallback (TestParseSessionJSONL_SuccessFromThinkingFallback)

**Phase 4: Cross-cutting (~3 hours)** — gap #4
- [ ] M4a: TS wrapper change — `${MOTOKO_SESSION_ID}.jsonl` filename when env var is set
- [ ] M4b: Verify AILANG-side `derive_session_id()` still honors `MOTOKO_SESSION_ID` (already done in M-MOTOKO-EVAL-INSTRUMENTATION; this is regression coverage)
- [ ] M4c: New test: `TestSessionIDUnification` — adapter sets MOTOKO_SESSION_ID, verifies all 3 places (filename, JSONL field, adapter result.SessionID) match

**Phase 5: Config layer (~3 hours)** — gaps #3 + #9
- [ ] M5a: Adapter passes `MOTOKO_COST_INPUT_PER_1K` / `MOTOKO_COST_OUTPUT_PER_1K` / `MOTOKO_COST_CACHE_READ_PER_1K` env vars derived from the model's pricing block
- [ ] M5b: motoko config loader reads those env vars; overrides profile-config when present
- [ ] M5c: Backfill cost_rates for 4 motoko-* model entries in models.yml's pricing blocks (already there from yesterday but verify)
- [ ] M5d: New test: `TestCostRatesFromModelsYML` — load each motoko-* model, verify pricing → env var conversion

**Phase 6: Validation (~3 hours)** — end-to-end
- [ ] M6a: New integration test `TestEndToEnd_FullResultPopulation` — runs motoko via adapter against a real OPENROUTER_API_KEY (gated by `AILANG_MOTOKO_LIVE=1`), asserts every field of Result is populated correctly
- [ ] M6b: Re-run M5 paired-comparison `motoko-claude-haiku-4-5 vs claude-haiku-4-5` — capture real numbers
- [ ] M6c: Update CHANGELOG with concrete paired-comparison results
- [ ] M6d: Update `internal/executor/motoko/README.md` + `docs/internal/EXECUTOR_SHAPE.md` with the cleaned-up architecture
- [ ] M6e: Move design doc + sprint plan to `design_docs/implemented/v0_18_1/` with implementation report

### Files to Modify/Create

**Modified files (this repo):**
- `internal/executor/motoko/motoko.go` (+~80 LOC) — discoverMotokoRepo; cost_rates env-var threading; --version probe in HealthCheck
- `internal/executor/motoko/parser.go` (+~30 LOC) — success-criteria fallback (thinking.finish_reason)
- `internal/executor/motoko/parser_test.go` (+~50 LOC) — fallback path test
- `internal/executor/motoko/execute_test.go` (+~80 LOC) — TestEndToEnd_FullResultPopulation, TestSessionIDUnification, TestCostRatesFromModelsYML
- `internal/executor/motoko/README.md` (+~50 lines) — updated architecture section
- `docs/internal/EXECUTOR_SHAPE.md` (+~10 lines) — note motoko's wrapper-discovery + version-querying patterns
- `cmd/smoke-motoko/main.go` (-~10 LOC) — remove the manual MOTOKO_REPO setting (no longer needed)
- `changelogs/v0.10-current.md` (+~50 lines) — `[Unreleased]` entry citing M6 measurement

**Modified files (motoko_agent repo, separate PR on `motoko-dx-compaction-pending`):**
- `src/core/agent_loop_v2.ail` (+~10 LOC, -~10 LOC) — gap #1 root-cause fix; loaded_extensions emission fix
- `src/tui/src/index.ts` (+~15 LOC, -~6 LOC) — PlainLogger + JsonlLogger no exit-on-done; --version short-circuit
- `src/tui/src/runtime-process.ts` (no change — already sets MOTOKO_HEADLESS in M2 of v0.18.0)
- `scripts/run-agent.sh` (+~15 LOC) — --headless flag + --version + ${MOTOKO_SESSION_ID}.jsonl filename
- `/Users/mark/go/bin/motoko` (wrapper) (+~5 LOC) — same flags propagated

---

## Examples

### Example 1: Full Result correctness after this sprint

**Before (today's smoke):**
```
Success:      false
Error:        motoko terminated without emitting run_summary (likely crash)
SessionID:    session_f00db4cf-...  (no match with JSONL filename)
Turns:        2
Tool calls:   1
Input tok:    2502
Output tok:   93
Cost USD:     $0.000000
ProviderData[motoko_finish_reason]: missing
ProviderData[motoko_commit]: missing
ProviderData[loaded_extensions]: []
```

**After (target):**
```
Success:      true
SessionID:    session_f00db4cf-...  (matches JSONL filename + JSONL session_id field)
Turns:        2
Tool calls:   1
Input tok:    2502
Output tok:   93
Cache read:   128
Cost USD:     $0.001242  (computed from models.yml.pricing × token counts)
ProviderData[motoko_finish_reason]: "stop"
ProviderData[motoko_commit]: "84fa449"
ProviderData[motoko_version]: "0.6.0"
ProviderData[loaded_extensions]: ["test_dummy", "compose", ...]
```

### Example 2: One-line cost_rates source-of-truth

**Before:** Adding model `motoko-deepseek-v4` requires:
1. Add to models.yml with pricing block
2. Add to motoko's `dogfood/config.json` cost_rates
3. Add to motoko's `default/config.json` cost_rates
4. Add to motoko's `openrouter/config.json` cost_rates

**After:** Adding model `motoko-deepseek-v4` requires:
1. Add to models.yml with pricing block. Done.

The adapter passes the pricing through env vars; motoko's config loader uses them when present.

### Example 3: Discoverable MOTOKO_REPO

**Before:** Smoke runner does `os.Setenv("MOTOKO_REPO", "/Users/mark/dev/sunholo/motoko_agent")` — hardcoded path.

**After:**
```go
repo, err := discoverMotokoRepo("motoko")  // reads /Users/mark/go/bin/motoko, parses MOTOKO_REPO=...
```

Works for any user with motoko installed via the wrapper, no env-var configuration needed.

---

## Conflict Surface

This sprint touches:
- AILANG: `internal/executor/motoko/` (adapter package; no other AILANG code paths affected)
- motoko: `src/core/agent_loop_v2.ail` (already modified in v0.18.0 family) + `src/tui/src/{index,runtime-process}.ts` + `scripts/run-agent.sh` + the wrapper

**Programs that MUST still work** (regression test fixtures in M6a):
1. **v0.18.0 motoko adapter behavior with `MOTOKO_REPO` env var manually set** — backward compat; fallback path remains
2. **motoko TUI invocation from a real terminal** (interactive use) — TS layer's exit-on-EOF logic must not break the TUI's normal lifecycle
3. **`motoko "task"` CLI invocation without `--headless` flag** — env-var-based detection (`process.stdin.isTTY=false → MOTOKO_HEADLESS=1`) must remain
4. **All v0.18.0 fixture-based parser tests** — TestParseSessionJSONL_Success / _CostExhausted / _DP7Rejected / _NoSummaryCrash continue to pass with the new success-criteria fallback in place
5. **Existing AILANG eval-suite invocations against pi/codex/opencode/claude/gemini** — none of those touch motoko code paths, but verify by running `make test` whole-tree

### What deliberately changes

- **Adapter no longer relies on env-var `MOTOKO_REPO`** as primary source — wrapper-script-discovery is canonical. Fallback retained.
- **Wrapper's session JSONL filename** changes from ISO-timestamp to `${MOTOKO_SESSION_ID}.jsonl` when env var is set. Backward-compat: when env var unset, ISO-timestamp scheme remains.
- **TS layer's `process.exit(0)` on done is removed** — TUI exits on runtime EOF instead. Behavioral change: any external observer of motoko CLI exit codes will see a slightly delayed exit (waiting for runtime EOF). Acceptable.
- **`Result.Success` for missing run_summary**: was always `false`; now `true` when last thinking.finish_reason="stop". Some currently-failing eval results will flip to passing — this is INTENTIONAL (they actually succeeded; the gating was wrong).

---

## Testing Strategy

**Unit tests:**
- `TestParseSessionJSONL_SuccessFromThinkingFallback` — JSONL with thinking finish_reason=stop but NO run_summary → Success=true (gap #2)
- `TestDiscoverMotokoRepo_FromWrapper` — given a stub wrapper script with `MOTOKO_REPO=...`, returns the parsed value (gap #5)
- `TestDiscoverMotokoRepo_FromWrapperFallbackToEnv` — wrapper unparseable → falls back to env var (gap #5 backward compat)
- `TestCostEnvVarsFromPricing` — pricing block → env var conversion (gap #3)
- `TestSessionIDUnification` — adapter MOTOKO_SESSION_ID flows to JSONL filename + JSONL session_id field (gap #4)

**Integration tests:**
- `TestEndToEnd_FullResultPopulation` (gated `AILANG_MOTOKO_LIVE=1`) — actual motoko subprocess + real OPENROUTER_API_KEY; asserts every Result field is populated as expected (THE acceptance test for this sprint)
- `TestCostRatesFromModelsYML` — every motoko-* model entry has a pricing block; pricing → env var produces non-zero cost_rates

**Regression tests:**
- All existing `TestParseSessionJSONL_*` continue to pass (no fixture changes)
- `TestRegistration_Motoko`, `TestRegister_Idempotent`, etc. — registration contract unchanged
- `make test` whole-tree green; no other executor adapter affected

**Coverage target:** ≥80% on `motoko.go` and `parser.go` (current 86.2% — should stay at or above)

**Manual smoke verification (M6b):**
```bash
ailang eval-suite \
  --models motoko-claude-haiku-4-5,claude-haiku-4-5 \
  --benchmarks <small-tier>
```
Expected: TWO comparable rows with non-zero pass-rate / non-zero cost / non-zero token counts. Real numbers, not all-zero placeholders.

---

## Non-Goals

**Not in this feature:**
- M-MOTOKO-EXT-PER-TASK (queued v0.19.0) — per-invocation extension specification; this hardening is a prerequisite, not a substitute
- Cloud Job deployment (queued M4 cross-repo PRs from v0.18.0) — those land separately on the multivac repo; orthogonal to this hardening
- Performance optimization (~3-4s per smoke is acceptable for v0.1; sub-second startup is M-MOTOKO-PERF territory)
- Streaming JSONL parser improvements (current post-completion read is fine for the eval-harness use case)
- New extension hooks or motoko-ext-* packages — this sprint touches only the kernel + adapter integration
- Replacing motoko's session JSONL with OTEL spans — separate v0.20+ thought experiment

---

## Timeline

**Day 1** (~6 hours):
- Phase 1 (M1a, M1b, M1c): gap #1 investigation + targeted fix — ~3h
- Phase 2 start (M2a, M2b): gap #6 + gap #7 — ~3h

**Day 2** (~6 hours):
- Phase 2 finish (M2c, M2d, M2e): gap #8 + gap #10 — ~2h
- Phase 3 (M3a, M3b, M3c): gap #2 + gap #5 — ~3h
- Phase 4 start (M4a): gap #4 wrapper change — ~1h

**Day 3** (~6 hours):
- Phase 4 finish (M4b, M4c) + Phase 5 (M5a-d): gap #4 verification + gap #3+#9 — ~3h
- Phase 6 (M6a-e): end-to-end validation + finalize — ~3h

**Total: ~18 hours across 3 working days** (with buffer; investigation phase #1 may compress if the bisect is fast).

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Gap #1's root cause is something I can't fix from outside motoko (e.g. AILANG stdlib bug in dispatch_response_intercept) | High (blocks the whole sprint) | Investigation phase produces a clear "this is the cause" output; if the cause is in AILANG core, escalate as a separate AILANG sprint with explicit dependency tracking |
| TS-side process lifecycle change (gap #10) breaks the TUI's interactive use | Medium | Comprehensive manual TUI smoke before merge; keep the existing exit-on-EOF code as a fallback if the new path misbehaves |
| Cost rates env-var passthrough conflicts with motoko's existing profile cost_rates | Low | env-var presence ALWAYS overrides; document the precedence in the AILANG-side adapter README + motoko's profile loader docs |
| MOTOKO_REPO discovery from wrapper script breaks if the wrapper is symlinked or aliased | Low | Use `realpath` / `EvalSymlinks` before reading; document that aliases bypass the discovery path |
| Session_id unification (gap #4) breaks tools that grep filenames for ISO timestamps | Low | Wrapper still produces ISO-timestamp filenames when env var is unset; only adapter-driven invocations use the new naming |
| Live integration test (M6a) is flaky due to OPENROUTER_API_KEY rate limits | Medium | Gate on `AILANG_MOTOKO_LIVE=1` so CI doesn't spawn real LLM calls; document expected $$ cost per run; cache the fixture and re-use for repeated debug runs |
| Backfilling cost_rates surfaces missing pricing for models we forgot | Low | M5d test catches this — every motoko-* model entry must have non-zero pricing; CI fails if not |
| Investigation phase #1 takes >1 day | Medium | Strict time-box: if no root cause after 4 hours of bisecting, escalate by adding emit_event markers in dispatch_solver_candidate's iteration loop and revisit |

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Scoring

| Axiom | Score | Justification |
|---|---|---|
| A1: Determinism | +1 | Single canonical session_id eliminates the 3-way race; cost_rates source-of-truth eliminates per-environment drift |
| A2: Replayability | +2 | Full Result-correctness assertion + matching session_ids + version-pinning means a captured run is fully reproducible |
| A3: Effect Legibility | +1 | Cost_rates env-var passthrough makes the cost-tracking effect visible at the adapter boundary; was implicit-via-profile-config |
| A4: Explicit Authority | 0 | No new ambient authority; just plumbing existing knowledge through |
| A5: Bounded Verification | +2 | Per-gap fix is locally verifiable; investigation phase #1 produces a bounded reproducer |
| A6: Safe Concurrency | 0 | No concurrency surface change |
| A7: Machines First | +2 | The whole point — Result fields are mechanically reliable; eval-suite output is trustworthy without human interpretation |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | +2 | Eliminates the $0 cost reporting gap; cost_rates source-of-truth means every benchmark row has accurate cost data |
| A10: Composability | +1 | --version + --headless flags compose with any caller (CI, eval-harness, MCP wrapper); session_id unification composes with span tracing |
| A11: Structured Failure | +1 | Success-criteria refactor distinguishes "real failure" from "missing run_summary" — typed Error messages identify which |
| A12: System Boundary | +1 | MOTOKO_REPO discovery makes the adapter↔motoko boundary self-describing; was implicit |

**Net Score: +13** ✅ Proceed to implementation

### Hard Violation Check

- [x] A1 (Determinism): No new nondeterminism — eliminating session_id race is determinism win
- [x] A3 (Effects): Cost_rates env-var passthrough is explicit; motoko's profile config inheritance is documented
- [x] A4 (Authority): No new ambient permissions; --version and --headless are explicit caller decisions
- [x] A7 (Machines First): The whole sprint's purpose

---

## References

- **Predecessor sprint** (the one whose smoke testing surfaced these issues): [`design_docs/planned/v0_18_0/m-motoko-executor-adapter.md`](../v0_18_0/m-motoko-executor-adapter.md)
- **Schema-v1 contract** (motoko's JSONL): [`m-motoko-eval-instrumentation.md`](https://github.com/sunholo-data/motoko_agent/blob/motoko-dx-compaction-pending/design_docs/implemented/motoko_agent/m-motoko-eval-instrumentation.md)
- **Downstream consumer** (depends on this hardening landing): [`design_docs/planned/v0_19_0/m-motoko-ext-per-task.md`](../v0_19_0/m-motoko-ext-per-task.md)
- **Today's progressive fixes** (the partial work this sprint completes):
  - AILANG `dc1f4eea` — HealthCheck + MOTOKO_REPO fallback
  - motoko `83fb6cf` — MOTOKO_HEADLESS env var
  - motoko `cc5bc1f` — run_summary-before-done reorder
- **Live smoke evidence**:
  - `ailang msg_20260508_101107_6d080f19` — original blocker report from sprint-executor
  - `ailang msg_20260508_102147_4f9aa71d` — motoko-explore's preferred fix-shape signal
  - `ailang msg_20260508_102940_4901dc8e` — fix-landed update
- **EXECUTOR_SHAPE.md** (the contract this sprint's adapter changes must respect): [`docs/internal/EXECUTOR_SHAPE.md`](../../../docs/internal/EXECUTOR_SHAPE.md)
- **Axiom reference**: [Design Axioms](/docs/references/axioms)

---

## Future Work

Features that build on this:

- **M5 of v0.18.0 (threshold-measurement run)** — becomes trustworthy after this lands; produces real numbers for the cost-arbitrage thesis
- **M-MOTOKO-EXT-PER-TASK (v0.19.0)** — per-invocation extension config; depends on gap #4 (session_id unification) and gap #6 (extension visibility) being clean
- **M-BENCH-MOTOKO-EXTENSIONS (queued v0.19+)** — per-extension contribution analysis; depends on cost_rates being authoritative (gap #3) so per-extension cost deltas are meaningful
- **M-MOTOKO-PERF (proposed v0.20+)** — startup latency optimization (currently ~3-4s per smoke); orthogonal to this sprint
- **M-MOTOKO-OTEL (proposed v0.20+)** — replace JSONL with OTEL spans; orthogonal long-term refactor that subsumes session_id unification
- **M-EVAL-HARNESS-CROSS-EXECUTOR-ASSERTIONS** (proposed v0.20+) — apply this sprint's "full Result correctness" pattern to claude/gemini/codex/opencode/pi adapters; surface their gaps the same way
