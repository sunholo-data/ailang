# Sprint Plan: M-EVAL-MEASUREMENT-CONTRACT

**Design doc**: [m-eval-measurement-contract.md](m-eval-measurement-contract.md)
**Target**: v0.31.0
**Milestones**: 5 (M1–M2 P0 · M3–M4 P1 · M5 P2)
**Estimate**: **~3.5 working days (28h)** — down from the doc's 4d, because M1's hook point turned out to be an existing filter pattern rather than new plumbing (see §0). M5 is explicitly droppable.
**Status**: COMPLETE (2026-07-29) · **Risk was**: Medium — one material design correction (§0), and M1's integration test needs a deliberately-broken subject.
**Branch**: `dev` (no compiler surface touched; `make check-boundaries` applies)

---

## 0. Premise re-verification (first-party, this session, at `48fe89b35`)

The design doc's verification log (V1–V11) was re-checked. **Ten of eleven confirmed.** One is materially wrong in a way that changes M1's implementation.

### CONFIRMED (carried forward, not re-litigated)

- V1/V2 — `HealthCheck` exists, is on the `Executor` interface (`internal/executor/executor.go:31`), implemented by claude/codex/managed_agents/motoko.
- V3 — `motoko --version` is handled in `parseMotokoFlags()` (`src/tui/src/index.ts:638`) at the TS argv level and exits before any AILANG module load. This is why it exited 0 while the core was dead.
- V4/V5/V6 — no validity concept, no paired analysis, `runtime_config_resolved` broadcast but never banked. All three negative-existence claims re-grepped and confirmed absent.
- V7–V11 — outage count (72), the `${PASS_ON:-0}` coercion bug, the n=84 power arithmetic, the haiku ceiling, and the zero fmt-arm runs.

### MATERIALLY CORRECTED

**C1 — `HealthCheck` is called PER BENCHMARK RUN, not per model.**

The doc says the canary should be "called once per model… next to the existing HealthCheck at `agent_runner_multi.go:107`". But that call site sits inside the per-`(spec, language)` function that then creates the isolated workspace — it runs **once per benchmark execution**. A 90s canary there would cost `90s × benchmarks × langs × trials`, i.e. hours per suite. Implementing the doc literally would make the gate unusable and it would be turned off.

**The correct hook already exists.** `cmd/ailang/eval_suite.go:408` filters the model list in agent mode:

```go
originalModels := modelList
modelList = eval_harness.GlobalModelsConfig.FilterAgentSupportedModels(modelList)
// ... then warns about each skipped model
```

This is per-model, runs exactly once, and already implements skip-with-warning — which is precisely decision **D2 (skip that model, continue others)**. M1 extends this filter rather than adding a second mechanism.

**Consequences for the plan:**
1. M1 lands in `eval_suite.go`'s filter, not `agent_runner_multi.go`.
2. Models failing the canary never enter the run matrix, so "bank zero rows" is structural rather than a thing we must remember to enforce.
3. `totalRuns` (line ~437) is computed from `len(modelList)` *after* filtering — so the progress accounting stays correct for free.
4. M1 shrinks (~1 day → ~0.6 day); the saved time is redistributed to M3, which is the milestone most likely to be underestimated.

### Estimate delta

Doc said 4d. This plan says 3.5d: M1 −0.4d (existing filter pattern), M3 +0.3d (McNemar edge cases + schema migration are fiddlier than "1 day"), M5 −0.4d (reduced to a warning + doc; no lint framework).

---

## 1. Milestones

### M1 — Per-model canary gate (~4.5h, ~165 LOC + ~140 test LOC) **P0**

The milestone that would have prevented the six-day outage.

**Tasks**
1. Add `CanaryCheck(ctx) error` to the `Executor` interface with a **default no-op** so claude/codex/managed_agents compile unchanged and opt in later (`internal/executor/executor.go`, ~25 LOC).
2. Implement `internal/executor/motoko/canary.go` (~140 LOC): fixed one-line task, temp workspace, hard 90s timeout. Asserts **all** of: exit 0, ≥1 step executed, ≥1 tool call dispatched, `run_summary` present. Anything else ⇒ typed error carrying the first stderr line.
3. Extend the agent-mode filter at `cmd/ailang/eval_suite.go:~408` to drop canary-failing models, reusing the existing skipped-model warning block.
4. `--no-canary` escape hatch for iteration.

**Acceptance criteria**
- [~] **DEVIATED** — a canary-failing subject is skipped and banks zero rows (`TestRunModelCanary_EndToEnd`), but via a *registered fake*, not a deliberately-broken mk-ast worktree. Reproducing the true outage needs mk-ast + bun + ollama, which CI does not have; a local-only test would go stale. The true-condition check remains a manual/rig-only follow-up.
- [x] A healthy motoko passes the canary and the suite proceeds normally.
- [x] claude/codex/managed_agents are unaffected (default no-op) — proven by their existing tests still passing untouched.
- [x] Canary runs exactly **once per model per suite**, asserted by a call-counter in the test.

**Risks** — canary flakiness would block good runs. Mitigation: retry once on ambiguous failure (timeout/empty output), and only a *clean* reproducible failure causes a skip.

### M2 — Universal validity flag (~5h, ~85 LOC + ~110 test LOC) **P0**

**Tasks**
1. `Validity{Valid bool; Reason string}` in `internal/eval_harness/metrics.go`, serialised on every banked result (~45 LOC).
2. Reason constants: `canary_failed`, `zero_files`, `zero_pass_all`, `config_mismatch`, `harness_error`.
3. `internal/eval_analysis` excludes `valid == false` from every aggregate by default; `--include-invalid` opts back in (~40 LOC).
4. Quarantine the 2026-07-20 row in `docs/static/benchmarks/microrag_ab.jsonl` **in place** (D4 — annotate, never delete).

**Acceptance criteria**
- [x] Round-trip test: `Validity` survives write→read with and without a reason.
- [x] An aggregate over a fixture containing one invalid row excludes it by default and includes it under `--include-invalid`.
- [x] The 2026-07-20 row carries `validity.valid=false, reason=zero_pass_all`; the microRAG trend then reads −3.1 / −4.8 / +13.1.
- [x] Back-compat: rows with **no** `validity` field are treated as valid (absent ≠ invalid), so historical data doesn't vanish.

### M3 — Paired A/B analysis + McNemar (~8h, ~200 LOC + ~170 test LOC) **P1**

Largest milestone; the one that makes future A/Bs able to answer anything.

**Tasks**
1. `internal/eval_analysis/paired.go`: join arms on `(benchmark, lang, trial_index)` → `(on_pass, off_pass)` pairs (~90 LOC).
2. McNemar over discordant counts `b`/`c`. **Exact binomial** when `b+c < 25`, χ² with continuity correction above (document the choice in the file header) (~70 LOC).
3. Extend the `*_ab.jsonl` schema with a `pairs` array so the test is recomputable from banked data without re-running (~40 LOC).
4. Emit `pairs` from `tools/launchd/nightly-eval.sh` for both weekly A/Bs.

**Acceptance criteria**
- [x] Replaying the banked 2026-07-27 microRAG data reproduces the historical aggregate `delta_pp = +13.1` **exactly** (regression against real data, not a fixture).
- [x] McNemar matches hand-computed values on at least 3 fixtures including the `b+c=0` degenerate case.
- [x] **No p-value is reported when `b+c < 10`** — output states the evidence base is too small (design-doc risk mitigation, enforced by test).
- [x] Aggregate delta is still emitted (additive change, nothing removed).

**Risks** — trial-index alignment across arms may not be stable if a benchmark errored in one arm only. Mitigation: unmatched benchmarks are reported as an explicit `unpaired` count rather than silently dropped.

### M4 — Bank the resolved config (~4h, ~95 LOC + ~80 test LOC) **P1**

**Tasks**
1. Capture motoko's step-0 `runtime_config_resolved` (currently consumed by `system_prompt_guard.go` and discarded) into the result row: `resolved_profile`, `resolved_extensions`, `resolved_verification`.
2. Assert against the `models.yml` claim (`motoko_profile`); mismatch ⇒ `validity: invalid(config_mismatch)`.

**Acceptance criteria**
- [~] **PARTIAL** — `resolved_profile` is recorded and asserted. **Extensions are NOT**: the design assumed the step-0 broadcast already carried them; it carries only model, step_budget, 7 env flags and system_md. Adding the profile was a one-line motoko change (mk-ast `f7bbe8d`); extensions need real plumbing (the runtime config is not in scope at the emit site). The profile alone catches the motivating bug. Follow-up filed in the guide's Known gaps.
- [x] A deliberate mismatch (claim `cloud`, run `ollama`) yields `config_mismatch` and is excluded from aggregates.
- [x] Executors that don't broadcast a resolved config are unaffected (absent ⇒ no assertion, not a failure).

### M5 — Subject-selection headroom rule (~2.5h, ~45 LOC + ~40 test LOC) **P2 — DROPPABLE**

**Tasks** — warn at A/B setup when the control arm's historical pass rate exceeds a configurable ceiling (default 90%), since a small effect cannot be resolved there. Document the rule in the eval guide.

**Acceptance criteria**
- [x] Configuring an A/B whose control has a ≥90% historical rate emits a loud warning naming the ceiling problem.
- [x] The haiku-fmt configuration (both arms ~96%) triggers it.
- [x] Warning only — never blocks a run.

**Drop condition**: if M1–M4 run past day 3, cut M5 and file it separately. It prevents a *future* mistake; M1–M4 fix *current* broken data.

---

## 2. Day-by-day

| Day | Work | Exit condition |
|---|---|---|
| 1 | M1 canary (interface, motoko impl, filter hook, `--no-canary`) + broken-subject integration test | Broken motoko skipped in <60s, 0 rows banked |
| 2 | M2 validity (struct, serialisation, analysis filter, back-compat) + quarantine the bad row | microRAG trend reads 3 points, not 4 |
| 3 | M3 paired + McNemar + schema + nightly-eval emission | 2026-07-27 replay reproduces +13.1 exactly |
| 3.5 | M4 resolved config; M5 if time; docs (MOTOKO.md §6, eval guide) | `make test` + `make check-boundaries` green |

## 3. Migration / back-compat

- **Absent `validity` ⇒ valid.** Historical rows have no such field; treating absent as invalid would erase every pre-v0.31.0 datapoint. Explicitly tested.
- **`pairs` is additive.** Existing `*_ab.jsonl` consumers keep working; the aggregate fields are untouched.
- **`CanaryCheck` defaults to no-op.** No executor is forced to implement it in this sprint.
- **Dashboard numbers will move** where invalid rows were previously included. This is the point of the sprint, but it must be called out in the release notes rather than landing silently.

## 4. Scope guard

Explicitly **not** in this sprint: re-running or back-filling historical evals; changing benchmarks/tiers/grading; raising trials or n (deferred until pairing reveals the true variance); extending `CanaryCheck` beyond motoko; any compiler/stdlib change.

## 5. Risks (most-likely-to-make-this-silently-wrong first)

| # | Risk | Mitigation |
|---|---|---|
| R1 | Canary passes while the subject is subtly broken (e.g. tools work, verification gate doesn't) | Canary asserts a **tool call** and `run_summary`, not just exit 0 — the outage's exact signature |
| R2 | M3 pairing silently drops unmatched benchmarks, biasing the result | Report `unpaired` count explicitly; test asserts it is surfaced |
| R3 | Treating absent `validity` as invalid erases history | Dedicated back-compat test in M2 |
| R4 | The canary's 90s timeout is too tight on a loaded rig and skips healthy models | Retry once; only clean reproducible failure skips; `--no-canary` escape hatch |
| R5 | M1's broken-subject test is hard to build reproducibly | Use a temp git worktree of mk-ast with one signature reverted — the exact, known outage condition |

## 6. Velocity basis

Recent comparable sprints: `m-nightly-flake-guard` (~1.4d, 4 milestones, same nightly-eval surface), `m-check-strict-fallbacks` (~2d, revised up from ~1d). 14-day window shows sustained multi-milestone delivery. This sprint is ~590 impl + ~540 test LOC across 5 milestones in a domain with existing test scaffolding — 3.5d is consistent, with M5 as the release valve.

## 7. Deliverables

- `internal/executor/motoko/canary.go` (new), `internal/eval_analysis/paired.go` (new)
- Modified: `executor.go`, `eval_suite.go`, `metrics.go`, `eval_analysis/*`, `tools/launchd/nightly-eval.sh`
- Quarantined 2026-07-20 row in `microrag_ab.jsonl`
- Docs: MOTOKO.md §6, eval guide
- Design doc moved to `design_docs/implemented/v0_31_0/` on completion
