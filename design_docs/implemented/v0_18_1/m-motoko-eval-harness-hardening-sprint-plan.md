# Sprint Plan: M-MOTOKO-EVAL-HARNESS-HARDENING

**Status**: Implemented (2026-05-08)
**Target**: v0.18.1 (patch release on top of v0.18.0)
**Estimated**: 3 working days (~18 hours, ~300 LOC across both repos)
**Actual**: ~6 hours wall-clock, ~330 LOC + 11 new tests, single session
**Source-of-truth design**: [m-motoko-eval-harness-hardening.md](m-motoko-eval-harness-hardening.md)

> This plan drives execution against the design doc. All architectural decisions, axiom scoring, risks, and rationale live there. This file is the milestone-by-milestone schedule.

> **⚠️ INVESTIGATION-FIRST**: Phase 1 (gap #1 root-cause via debug instrumentation) MUST complete before any other code change in `agent_loop_v2.ail`. Writing a fix without knowing the cause is gambling.

---

## Why this sprint, why now

- **v0.18.0's adapter is structurally sound but operationally wrong**. 22+ tests pass against fixtures; first live run reveals 10 interconnected gaps.
- **M5 (threshold-measurement) blocked**. Without these fixes, `motoko-claude-haiku-4-5` reports `0% pass, $0 cost` — wrong on both axes; the cost-arbitrage thesis cannot be tested.
- **M-MOTOKO-EXT-PER-TASK (v0.19.0) blocked**. That sprint depends on session_id unification (gap #4) and accurate extension visibility (gap #6).
- **User-confirmed direction**: "we need it all I think. lets get to the bottom of the gaps - I think a design doc process will help" — sequence the fixes properly rather than ad-hoc patching that keeps surfacing surprise failures.

---

## Velocity calibration

Reference points (recent cross-repo motoko sprints):

| Sprint | Total LOC | Sprint length | Notes |
|---|---:|---:|---|
| M-MOTOKO-EXECUTOR-ADAPTER (v0.18.0) | ~1,700 | 1.5 days | Built from scratch; comprehensive |
| M-MOTOKO-EVAL-INSTRUMENTATION (motoko side) | ~330 | 0.5 days | Targeted refactor, mostly mechanical |
| Today's progressive fixes (3 commits) | ~80 | 0.5 days | Partial; this sprint completes them |

**Planning target for this sprint**: ~300 LOC, but cognitive load > LOC count because of the cross-repo investigation. Investigation phase #1 alone may consume 3+ hours. **Estimate: 3 days** (with buffer; investigation may compress).

---

## Milestone breakdown

Six phases, 22 milestones. Investigation phase is non-negotiably first.

| # | ID | Title | Est. LOC | Phase | Depends on |
|---|---|---|---:|---|---|
| M1a | M1a_DEBUG_MARKERS | Add debug:checkpoint emit_event markers in agent_loop_v2.ail | ~15 | Phase 1 | — |
| M1b | M1b_BISECT_RUN | Run smoke 3x; identify hang phase from JSONL | — | Phase 1 | M1a |
| M1c | M1c_TARGETED_FIX | Write the fix for gap #1's root cause; remove debug markers | ~5–20 | Phase 1 | M1b |
| M2a | M2a_EXTENSIONS_VIS | gap #6 — fix loaded_extensions emission to read rt.registry.hooks | ~15 | Phase 2 | — |
| M2b | M2b_HEADLESS_FLAG | gap #7 — wrapper `--headless` flag with env-var fallback | ~10 | Phase 2 | — |
| M2c | M2c_VERSION_FLAG | gap #8 — wrapper `--version` mode (motoko/ailang/extensions triplet) | ~15 | Phase 2 | — |
| M2d | M2d_TS_NO_EXIT | gap #10 — TS PlainLogger + JsonlLogger no longer process.exit on done | ~10 | Phase 2 | — |
| M2e | M2e_REORDER_AUDIT | Audit cc5bc1f reorder is now belt-and-braces (TS no longer exits early) | — | Phase 2 | M2d |
| M3a | M3a_SUCCESS_FALLBACK | gap #2 — parseSessionJSONL success-criteria fallback to thinking.finish_reason | ~30 | Phase 3 | — |
| M3b | M3b_REPO_DISCOVERY | gap #5 — discoverMotokoRepo() reads wrapper script | ~50 | Phase 3 | — |
| M3c | M3c_PARSER_TESTS | TestParseSessionJSONL_SuccessFromThinkingFallback + tests for M3b | ~50 | Phase 3 | M3a, M3b |
| M4a | M4a_TS_FILENAME | gap #4 — TS wrapper writes ${MOTOKO_SESSION_ID}.jsonl when env set | ~10 | Phase 4 | — |
| M4b | M4b_AILANG_SID_VERIFY | Verify AILANG-side derive_session_id honors MOTOKO_SESSION_ID (regression cov) | — | Phase 4 | — |
| M4c | M4c_UNIFICATION_TEST | TestSessionIDUnification — all 3 places match | ~30 | Phase 4 | M4a, M4b |
| M5a | M5a_COST_ENV_PASS | Adapter passes MOTOKO_COST_*_PER_1K env vars from models.yml.pricing | ~30 | Phase 5 | — |
| M5b | M5b_MOTOKO_COST_OVERRIDE | motoko config loader honors MOTOKO_COST_*_PER_1K env-var override | ~25 | Phase 5 | M5a |
| M5c | M5c_PRICING_BACKFILL | Verify all 4 motoko-* models.yml entries have pricing blocks | — | Phase 5 | M5a |
| M5d | M5d_PRICING_TEST | TestCostRatesFromModelsYML — every motoko-* entry produces non-zero cost env vars | ~30 | Phase 5 | M5b |
| M6a | M6a_FULL_RESULT_TEST | TestEndToEnd_FullResultPopulation (live, gated on AILANG_MOTOKO_LIVE=1) | ~80 | Phase 6 | All above |
| M6b | M6b_PAIRED_RUN | Re-run M5 paired comparison motoko-claude-haiku-4-5 vs claude-haiku-4-5 | — | Phase 6 | M6a |
| M6c | M6c_CHANGELOG | CHANGELOG entry with M6b's concrete numbers | ~50 | Phase 6 | M6b |
| M6d | M6d_DOC_UPDATE | Update README + EXECUTOR_SHAPE.md with cleaned-up architecture | ~60 | Phase 6 | M6c |
| M6e | M6e_FINALIZE | Move design doc + sprint plan to implemented/v0_18_1/ + Implementation Report | — | Phase 6 | M6d |

**Total**: ~530 LOC across 22 milestones × 3 days. Heavier than the headline ~300 because tests (M3c, M4c, M5d, M6a) account for ~190 LOC.

---

## Day-by-day

### Day 1 — Investigation + motoko-side fixes (~7 hours)

**Morning (4h):** Phase 1 — gap #1 investigation
- M1a (1h): Add debug:checkpoint markers around dispatch_response_intercept, dispatch_solver_candidate, dp7_gate, emit_run_summary, emit_event(done). Commit on `motoko-bisect-gap1` branch.
- M1b (1h): Run smoke against the bisect branch 3x. Identify last-emitted checkpoint. Document findings.
- M1c (2h): Write targeted fix; remove debug markers; commit as `fix(motoko): gap #1 — <root cause>`.

**Acceptance gate (Day 1 morning):** smoke produces both `done` AND `run_summary` events on the no-tool-calls success path.

**Afternoon (3h):** Phase 2 — motoko-side fixes
- M2a (1h): Fix loaded_extensions emission
- M2b (30min): Wrapper --headless flag
- M2c (45min): Wrapper --version mode
- M2d (45min): TS PlainLogger + JsonlLogger no exit-on-done

**Acceptance gate (Day 1 afternoon):**
- `motoko --version` prints 3-line triplet
- `motoko --headless "task"` works the same as `MOTOKO_HEADLESS=1 motoko "task"`
- session_start event's loaded_extensions list matches rt.registry.hooks contents

---

### Day 2 — AILANG-side + cross-cutting (~6 hours)

**Morning (3h):** Phase 3 — AILANG adapter fixes
- M3a (1h): Success-criteria fallback in parseSessionJSONL
- M3b (1.5h): discoverMotokoRepo() helper
- M3c (30min): Parser tests for both

**Acceptance gate (Day 2 morning):** All M3 unit tests pass; coverage stays ≥80% on parser.go.

**Afternoon (3h):** Phase 4 — session_id unification
- M4a (1h): TS wrapper filename logic
- M4b (30min): AILANG-side regression coverage
- M4c (1h): TestSessionIDUnification end-to-end
- Phase 5 start (M5a, 30min): Cost env-var passthrough

**Acceptance gate (Day 2 afternoon):** TestSessionIDUnification passes — adapter's MOTOKO_SESSION_ID flows to JSONL filename + JSONL session_id field + Result.SessionID.

---

### Day 3 — Config + validation (~5 hours)

**Morning (2.5h):** Phase 5 — cost rates
- M5b (1h): motoko config loader honors env-var override
- M5c (15min): Verify pricing blocks
- M5d (45min): TestCostRatesFromModelsYML
- M2e (15min): Audit reorder commit; comment if redundant
- Phase 6 start (M6a setup, 30min): Smoke runner setup

**Acceptance gate (Day 3 morning):** Cost env-vars flow correctly; tests pass.

**Afternoon (2.5h):** Phase 6 — end-to-end validation + finalize
- M6a (1h): Run TestEndToEnd_FullResultPopulation against live motoko + capture full Result
- M6b (45min): Run paired comparison; capture numbers
- M6c (15min): CHANGELOG with concrete numbers
- M6d (15min): README + EXECUTOR_SHAPE.md updates
- M6e (15min): Move docs to implemented/

**Acceptance gate (Day 3 afternoon):**
- Live smoke produces a Result with Success=true, CostUSD>0, all session_ids matching, motoko_version populated
- Paired comparison produces 2 distinct rows with non-zero data
- Design doc + sprint plan moved to implemented/v0_18_1/

---

## Dependency graph

```
M1a (debug markers)
  └── M1b (bisect)
        └── M1c (targeted fix) ──┐
                                  │
M2a (extensions vis)              │
M2b (--headless)                  │
M2c (--version)                   │
M2d (TS no-exit) ──── M2e (audit) │
                                  │
M3a (success fallback) ── M3c (tests)
M3b (repo discovery) ──┘     │
                              │
M4a (TS filename)             │
M4b (AILANG regression cov)   │
  └── M4c (unification test) ─┤
                              │
M5a (cost env pass)           │
  └── M5b (motoko override)   │
        └── M5d (test) ───────┤
M5c (verify pricing) ─────────┤
                              │
                              ▼
                          M6a (full result test)
                            └── M6b (paired run)
                                  └── M6c (CHANGELOG)
                                        └── M6d (docs)
                                              └── M6e (finalize)
```

Phase 1 (M1a-c) gates everything in agent_loop_v2.ail (M2a, M2e). Phases 2-5 can run partly in parallel after Phase 1; Phase 6 gates on all preceding.

---

## External dependencies

| Dependency | Status | Owner | Notes |
|---|---|---|---|
| **gap #1 root cause** | ⚠️ **unknown — Phase 1 produces this** | this sprint | Time-box: 4 hours; if no progress, escalate (see Risks in design doc) |
| AILANG dev branch is clean | ✅ green | mark | All today's work pushed |
| motoko_agent PR #6 open | ✅ green | sunholo-data/motoko_agent | New work continues to land here |
| OPENROUTER_API_KEY available for live smoke | ✅ verified | mark | Set in shell env |
| AILANG eval-suite + claude executor working | ✅ assumed | this repo | Required for M6b paired comparison |

---

## Sprint JSON (template for sprint-executor)

```json
{
  "sprint_id": "M-MOTOKO-EVAL-HARNESS-HARDENING",
  "design_doc_path": "design_docs/planned/v0_18_1/m-motoko-eval-harness-hardening.md",
  "sprint_plan_path": "design_docs/planned/v0_18_1/m-motoko-eval-harness-hardening-sprint-plan.md",
  "target_version": "v0.18.1",
  "status": "not_started",
  "milestones": [
    {"id": "M1a_DEBUG_MARKERS", "passes": null, "completed": null, "notes": ""},
    {"id": "M1b_BISECT_RUN", "passes": null, "completed": null, "notes": ""},
    {"id": "M1c_TARGETED_FIX", "passes": null, "completed": null, "notes": ""},
    {"id": "M2a_EXTENSIONS_VIS", "passes": null, "completed": null, "notes": ""},
    {"id": "M2b_HEADLESS_FLAG", "passes": null, "completed": null, "notes": ""},
    {"id": "M2c_VERSION_FLAG", "passes": null, "completed": null, "notes": ""},
    {"id": "M2d_TS_NO_EXIT", "passes": null, "completed": null, "notes": ""},
    {"id": "M2e_REORDER_AUDIT", "passes": null, "completed": null, "notes": ""},
    {"id": "M3a_SUCCESS_FALLBACK", "passes": null, "completed": null, "notes": ""},
    {"id": "M3b_REPO_DISCOVERY", "passes": null, "completed": null, "notes": ""},
    {"id": "M3c_PARSER_TESTS", "passes": null, "completed": null, "notes": ""},
    {"id": "M4a_TS_FILENAME", "passes": null, "completed": null, "notes": ""},
    {"id": "M4b_AILANG_SID_VERIFY", "passes": null, "completed": null, "notes": ""},
    {"id": "M4c_UNIFICATION_TEST", "passes": null, "completed": null, "notes": ""},
    {"id": "M5a_COST_ENV_PASS", "passes": null, "completed": null, "notes": ""},
    {"id": "M5b_MOTOKO_COST_OVERRIDE", "passes": null, "completed": null, "notes": ""},
    {"id": "M5c_PRICING_BACKFILL", "passes": null, "completed": null, "notes": ""},
    {"id": "M5d_PRICING_TEST", "passes": null, "completed": null, "notes": ""},
    {"id": "M6a_FULL_RESULT_TEST", "passes": null, "completed": null, "notes": ""},
    {"id": "M6b_PAIRED_RUN", "passes": null, "completed": null, "notes": ""},
    {"id": "M6c_CHANGELOG", "passes": null, "completed": null, "notes": ""},
    {"id": "M6d_DOC_UPDATE", "passes": null, "completed": null, "notes": ""},
    {"id": "M6e_FINALIZE", "passes": null, "completed": null, "notes": ""}
  ],
  "estimated_loc": 530,
  "actual_loc": null,
  "estimated_days": 3,
  "actual_days": null,
  "github_issues": []
}
```

---

## Risks (sprint-execution-specific)

The design doc captures architectural risks. Sprint-execution risks specific to the schedule:

| Risk | Likelihood | Impact | Mitigation |
|---|---|---|---|
| Investigation phase #1 takes >4 hours without root cause | Medium | High (blocks Day 1) | Strict time-box; if exceeded, escalate by adding markers in dispatch_solver_candidate's iteration loop and try motoko-explore for a fresh pair of eyes |
| M2d (TS no-exit) breaks the TUI's interactive use | Low | Medium | Manual TUI smoke before merge; keep exit-on-EOF as the actual exit path so TS still terminates correctly |
| M5b (motoko cost-config override) requires deeper config-loader refactor than estimated | Medium | Low (Phase 5 can extend into Day 4 buffer) | If 1h estimate blows past 2h, ship an env-var-only override (no config-loader change) and add the proper inheritance in a follow-up |
| M6a (live integration test) is flaky on rate limits | Medium | Low | Gated on AILANG_MOTOKO_LIVE=1; cache fixture for repeat runs; document expected ~$0.01 cost per run |
| Cross-repo coordination — motoko-side commits need to land on PR #6 + AILANG-side on dev IN ORDER (motoko first for new env vars + flags) | Medium | Medium | Each phase commits to one repo at a time; Day 2 morning depends on Day 1 motoko commits being pushed |
| Backfilling pricing (M5c) reveals models.yml has no pricing block for some motoko-* entries | Low | Low | M5c is a verification step; if a block is missing, add it (5 min per entry) and continue |

---

## Done criteria (sprint-level)

- [ ] M1a–M6e all `passes: true` in sprint JSON
- [ ] `make test` whole-tree green; coverage on motoko adapter package ≥80%
- [ ] Live smoke (`go run ./cmd/smoke-motoko -task "<simple>"` against real OPENROUTER_API_KEY) produces:
  - `Success: true`
  - `CostUSD > 0`
  - `NumTurns > 0`, `InputTokens > 0`, `OutputTokens > 0`
  - `ProviderData["motoko_finish_reason"] == "stop"`
  - `ProviderData["motoko_commit"]` populated
  - `ProviderData["motoko_version"]` populated
  - `ProviderData["loaded_extensions"]` matches motoko's actual registry
  - All 3 session_ids match (filename, JSONL field, Result.SessionID)
- [ ] M6b paired comparison `motoko-claude-haiku-4-5 vs claude-haiku-4-5` produces 2 rows with real numbers in the eval-suite output
- [ ] CHANGELOG entry under `[Unreleased]` cites M6b's concrete delta (pass-rate, cost, tokens)
- [ ] README + EXECUTOR_SHAPE.md updated to reflect the cleaned-up architecture
- [ ] Design doc + sprint plan moved to `design_docs/implemented/v0_18_1/` with Implementation Report appended

---

## References

- **Design doc** (source of truth): [m-motoko-eval-harness-hardening.md](m-motoko-eval-harness-hardening.md)
- **Predecessor**: [`design_docs/planned/v0_18_0/m-motoko-executor-adapter.md`](../v0_18_0/m-motoko-executor-adapter.md)
- **Downstream consumer**: [`design_docs/planned/v0_19_0/m-motoko-ext-per-task.md`](../v0_19_0/m-motoko-ext-per-task.md)
- **Source proposal**: User feedback after live smoke testing (2026-05-08): "we need it all I think. lets get to the bottom of the gaps"
- **EXECUTOR_SHAPE.md**: [`docs/internal/EXECUTOR_SHAPE.md`](../../../docs/internal/EXECUTOR_SHAPE.md)
- **Today's progressive fixes** (the partial work this sprint completes):
  - AILANG `dc1f4eea` — HealthCheck fix + MOTOKO_REPO fallback
  - motoko `83fb6cf` — MOTOKO_HEADLESS env var
  - motoko `cc5bc1f` — run_summary-before-done reorder
