# Sprint Plan: M-MOTOKO-PARALLEL-EXECUTION-ISOLATION

**Status**: Planned
**Target**: v0.18.2 (patch release on top of v0.18.1)
**Estimated**: 2 working days (~12 hours, ~250 LOC across both repos)
**Source-of-truth design**: [m-motoko-parallel-execution-isolation.md](m-motoko-parallel-execution-isolation.md)

> This plan drives execution against the design doc. All architectural decisions, axiom scoring, risks, and rationale live there. This file is the milestone-by-milestone schedule.

> **⚠️ INVESTIGATION-FIRST**: Phase 1 (dur_s=0 root-cause via lsof/strace) MUST complete and be human-reviewed before any code change in Phases 2–4. The v0.18.1 EADDRINUSE handler regressed the success rate (40/45 → 37/45) precisely because it was written without bisecting first; this sprint will not repeat that mistake.

---

## Why this sprint, why now

- **v0.18.1 closed serial-mode but not parallel.** First 3-harness eval comparison today (2026-05-08, smoke tier × `--agent-parallel 2`) hit 5/15 motoko failures, 4 of which are `dur_s=0` "no run_summary" crashes — a different failure class from anything v0.18.1 addressed.
- **EADDRINUSE patch made it worse.** Adding the v0.18.1 follow-up `6580adf` (port-yield handler) reduced success rate from 88.9% (v1) to 82.2% (v2). Confirmed regression — the handler causes losing siblings to share a winner's env-server bound to a different workdir, breaking downstream tool calls.
- **Cross-executor audit shows motoko is the outlier.** Every other CLI executor (claude/gemini/codex/opencode/pi) uses `cmd.Dir = task.Workspace` + no shared filesystem state + no embedded services. Motoko inherited a TUI architecture (long-lived bun + embedded env-server + cd-into-shared-MOTOKO_REPO) and the v0.18.0 adapter wraps it without re-isolating.
- **M5 paired-comparison is blocked at scale.** Serial-only motoko means the threshold-measurement experiment ("does motoko's harness lift cheap models?") takes days instead of hours. Each `motoko-*` row needs many runs for statistical confidence.
- **User-confirmed direction**: "ok but we need to fix the parallel runs" + "lets make a design doc and get to the bottom of it" — sequence the architectural fix properly rather than continue patching symptoms.

---

## Velocity calibration

Reference points (most recent cross-repo motoko sprints):

| Sprint | Total LOC | Sprint length | Notes |
|---|---:|---:|---|
| M-MOTOKO-EXECUTOR-ADAPTER (v0.18.0) | ~1,700 | 1.5 days | Built from scratch |
| M-MOTOKO-EVAL-HARNESS-HARDENING (v0.18.1, yesterday) | ~330 + 11 tests | ~6 hours wall-clock vs 3-day estimate | Single session, parallel sub-agents on independent fixes |
| Today's EADDRINUSE follow-up (regression) | ~50 | ~30 min | Counter-example: written without bisecting → made things worse |

**Planning target**: ~250 LOC, 2 days. Investigation-first phase 1 (~2h time-boxed) gates everything; if Phase 1 informs a clean fix, total compresses to ~10 hours like v0.18.1 did. If hypotheses are wrong, expect 2 full days.

The v0.18.1 acceleration came from (a) clear bisection produced unambiguous fixes, (b) Phases 2-5 ran mostly in parallel after Phase 1. Same pattern applies here.

---

## Milestone breakdown

Five phases, 18 milestones. Investigation phase is non-negotiably first AND human-gated.

| # | ID | Title | Est. LOC | Phase | Depends on |
|---|---|---|---:|---|---|
| M1a | M1a_LSOF_INSTRUMENTATION | Add lsof-snapshot wrapper script around motoko spawn for parallel runs | ~30 (bash) | Phase 1 | — |
| M1b | M1b_PARALLEL_REPRO | Run smoke × parallel-2 with M1a instrumentation; capture FS contention timeline | — | Phase 1 | M1a |
| M1c | M1c_HYPOTHESIS_PICK | Cross-check observations against H1/H2/H3; document conclusion in design doc | ~20 (markdown) | Phase 1 | M1b |
| M1d | M1d_PHASE1_GATE | **HUMAN REVIEW** — confirm hypothesis pick before Phase 2 begins | — | Phase 1 | M1c |
| M2a | M2a_ISOLATION_HELPER | New `internal/executor/motoko/isolation.go`: `setupMotokoHome(repoPath) → (path, cleanup, err)` with hardlink-mirror + copy fallback | ~80 | Phase 2 | M1d |
| M2b | M2b_ADAPTER_INTEGRATE | Wire `setupMotokoHome` into `Execute`: set `MOTOKO_HOME` env + `cmd.Dir = motokoHome`; defer cleanup | ~20 | Phase 2 | M2a |
| M2c | M2c_ISOLATION_TESTS | `TestExecute_PerTaskMotokoHome`, `TestSetupMotokoHome_HardlinkFallsBackToCopy`, `TestSetupMotokoHome_CleanupRemovesDir` | ~100 | Phase 2 | M2b |
| M2d | M2d_SERIAL_REGRESSION | Re-run v0.18.1's smoke (`go run ./cmd/smoke-motoko`) — must produce same Success=true, CostUSD>0 | — | Phase 2 | M2b |
| M3a | M3a_ENV_SERVER_PICK | Phase 1 informs: drop inline (in `index.ts`) OR drop auto_start (in `backend.ail`); apply chosen change | ~10–30 | Phase 3 | M1d |
| M3b | M3b_REVERT_HEALTHCHECK | Update v0.18.1 health-check-share in `backend.ail` to be a no-op (per-task isolation removes the need to share) | ~15 | Phase 3 | M3a |
| M3c | M3c_RETIRE_EADDRINUSE | Comment-out (don't delete — defense-in-depth) the EADDRINUSE retry/yield handler from v0.18.1 commit `6580adf` | ~10 | Phase 3 | M3a |
| M3d | M3d_SINGLE_SERVER_TEST | `TestExecute_SingleEnvServer`: assert exactly one listening socket per spawn (via `lsof -i` snapshot from a side-channel) | ~50 | Phase 3 | M3a, M3b |
| M4a | M4a_WARMCACHE_FLAG | Add `--warm-cache` flag to motoko adapter HealthCheck (off by default) | ~30 | Phase 4 | M2a |
| M4b | M4b_WARMCACHE_LOGIC | HealthCheck warm-cache runs `ailang check src/core/supervisor.ail` against MOTOKO_REPO (template), populating `.ailang/cache/` | ~20 | Phase 4 | M4a |
| M4c | M4c_WARMCACHE_TEST | `TestHealthCheck_WarmCache_Idempotent`: 3x consecutive warm-cache calls no-error, cache dir present after each | ~50 | Phase 4 | M4b |
| M5a | M5a_PARALLEL_4X5_RUN | Run smoke × `--agent-parallel 4` × 5 consecutive iterations; capture pass rate per run | — | Phase 5 | M2d, M3d, M4c |
| M5b | M5b_ACCEPTANCE_GATE | Verify ≥95% over 60 runs (≤3 infrastructure failures); WALL-CLOCK ≤8 minutes per iteration | — | Phase 5 | M5a |
| M5c | M5c_CHANGELOG_FINALIZE | CHANGELOG entry with M5a numbers + design doc move to `implemented/v0_18_2/` | ~80 | Phase 5 | M5b |

**Total**: ~515 LOC across 18 milestones × 2 days. Tests (M2c, M3d, M4c) account for ~200 LOC.

---

## Day-by-day

### Day 1 — Investigation + per-task isolation (~6 hours)

**Morning (3h):** Phase 1 — investigation-first
- M1a (1h): Write `tools/snapshot-fs-contention.sh` — wraps `lsof | grep MOTOKO` snapshots taken at 200ms intervals during a parallel motoko run, dumps to a timestamped log. Also captures `dtruss -e openat,write,unlink -p <PID>` for ONE motoko PID (macOS strace equivalent).
- M1b (1.5h): Run `ailang eval-suite --agent --models motoko-claude-haiku-4-5 --benchmarks adt_option,balanced_parens,binary_tree_sum --langs ailang --agent-parallel 2` with M1a instrumentation. Capture which files are written by each motoko PID. Look specifically for: (a) double-writes to `MOTOKO_REPO/src/core/.ailang/cache/compile/.../core.gob`, (b) port-bind attempts on the same port, (c) writes to `MOTOKO_REPO/src/core/ext/registry_generated.ail` or `MOTOKO_REPO/.motoko/store/`.
- M1c (30min): Cross-check observations against H1 (cache race) / H2 (env-server cross-routing) / H3 (registry race). Document conclusion in design doc as a "Phase 1 Findings" section.

**Acceptance gate (Day 1 morning) — HUMAN REVIEW REQUIRED:**
- Phase 1 Findings section in design doc names ONE of H1/H2/H3 (or new H4) as confirmed root cause
- Design Freeze checkboxes for "per-task isolation strategy", "single-env-server architecture", and "MOTOKO_HOME naming + lifecycle" are checked off based on findings
- **sprint-executor PAUSES here. Do not start Phase 2 until user confirms findings.**

**Afternoon (3h):** Phase 2 — per-task MOTOKO_HOME
- M2a (2h): `internal/executor/motoko/isolation.go` — `setupMotokoHome(repoPath)` creates `${TMPDIR}/motoko-task-<uuid>/`, attempts hardlink-mirror via `cp -al` (Linux/macOS) OR Go `filepath.Walk` + `os.Link`, falls back to full copy on cross-filesystem `EXDEV`. Returns `(homePath string, cleanup func(), err error)`.
- M2b (30min): Wire into `Execute`: append `MOTOKO_HOME=<homePath>` to env, set `cmd.Dir = motokoHome`, `defer cleanup()`. Note: `cmd.Dir` overrides the wrapper's `cd $MOTOKO_REPO` step, so wrapper edit is NOT required.
- M2c (1h): Tests — `TestExecute_PerTaskMotokoHome` (two sequential calls get distinct paths), `TestSetupMotokoHome_HardlinkFallsBackToCopy` (force EXDEV via mock), `TestSetupMotokoHome_CleanupRemovesDir`.

**Acceptance gate (Day 1 afternoon):**
- All M2 unit tests pass
- M2d: re-run `go run ./cmd/smoke-motoko -task "Print just 42"` — Success=true, CostUSD>0 (regression check)
- Coverage on `internal/executor/motoko/` stays ≥80%

---

### Day 2 — Single env-server + warm-cache + validation (~6 hours)

**Morning (2.5h):** Phase 3 — single env-server
- M3a (1h): Apply Phase-1-informed change. If "drop auto_start": edit `motoko_agent/src/core/backend.ail` `auto_start` branch to skip `spawnProcess` (return BackendHandle with `process_id="inline_in_bun"`, `process: None`). If "drop inline": edit `motoko_agent/src/tui/src/index.ts` to skip the `await startEnvServer(envPort, workdir)` call — the AILANG runtime spawns its own.
- M3b (30min): Update v0.18.1 health-check-share logic — with per-task isolation each session has its own env-server, so `health_ok(cfg.url)` for SHARING purposes is no longer relevant. Either remove the share-branch entirely or document why it's still there.
- M3c (15min): Comment-out the EADDRINUSE retry/yield handler in `motoko_agent/src/tui/src/env-server-main.ts` (commit `6580adf`). Don't delete — defense-in-depth in case per-task isolation has an edge case.
- M3d (45min): `TestExecute_SingleEnvServer` — spawn motoko via mock binary that writes `lsof -i -P -p $$ | wc -l` to a side-channel file; assert count matches expected (1 listening socket).

**Acceptance gate (Day 2 morning):**
- All M3 tests pass
- A single motoko run shows exactly 1 env-server process via `lsof -i | grep $MOTOKO_PID`

**Late morning (1.5h):** Phase 4 — cache pre-warming
- M4a (45min): Add `--warm-cache` flag to motoko adapter HealthCheck. Default off; when set, runs `ailang check $MOTOKO_REPO/src/core/supervisor.ail` (uses MOTOKO_REPO, the template) before returning.
- M4b (30min): Wire warm-cache to populate `MOTOKO_REPO/src/core/.ailang/cache/`. Per-task `setupMotokoHome` (M2a) hardlink-mirrors from MOTOKO_REPO INCLUDING the freshly-warmed cache.
- M4c (15min): `TestHealthCheck_WarmCache_Idempotent` — 3x consecutive `HealthCheck(--warm-cache)` calls no-error, `MOTOKO_REPO/src/core/.ailang/cache/compile/modules/` exists after each.

**Afternoon (2h):** Phase 5 — validation + finalize
- M5a (1h): Run `ailang eval-suite --agent --models motoko-claude-haiku-4-5 --tier smoke --agent-parallel 4` × 5 consecutive iterations. Output to `eval_results/v0_18_2_parallel4_iter{1..5}/`. Capture pass rate per iteration.
- M5b (30min): Compute aggregate: 60 runs total (5 × 15 benchmarks × 1 model × 1 lang). Acceptance gate: ≥95% (≤3 failures), AND each iteration's wall-clock ≤8 minutes. If gate fails, identify which type of failure (infra vs benchmark-correctness), iterate.
- M5c (30min): CHANGELOG entry covering all 5 phases + `git mv` design doc + sprint plan to `design_docs/implemented/v0_18_2/`. Update both docs' status headers to "Implemented (date)" with actuals.

**Acceptance gate (Day 2 afternoon):**
- 5/5 iterations of parallel-4 smoke produce ≥95% success rate (≤3 infra failures across 60 runs)
- Wall-clock per iteration ≤8 min
- CHANGELOG + design doc + sprint plan committed to `dev`

---

## Dependency graph

```
M1a (lsof script)
  └── M1b (parallel repro)
        └── M1c (hypothesis pick)
              └── M1d (PHASE 1 GATE — HUMAN REVIEW) ────────┐
                                                             │
M2a (isolation.go) ──── M2b (adapter wire-in)                │
                          ├── M2c (unit tests) ──── M2d (regression check) ──┐
                          └────────────────────────────────────────────────────┤
                                                                              │
M3a (env-server pick) ─── M3b (revert healthcheck) ──┐                        │
                          M3c (retire EADDRINUSE)    │                        │
                          └─── M3d (single-server test) ──────────────────────┤
                                                                              │
M4a (warmcache flag) ─── M4b (warmcache logic) ──── M4c (warmcache test) ────┤
                                                                              │
                                                                              ▼
                                                              M5a (parallel-4 × 5 runs)
                                                                └── M5b (acceptance gate)
                                                                      └── M5c (CHANGELOG + finalize)
```

Phase 1 (M1a-d) gates everything. Phases 2/3/4 can run mostly in parallel after Phase 1; Phase 5 gates on M2d + M3d + M4c.

---

## External dependencies

| Dependency | Status | Owner | Notes |
|---|---|---|---|
| **dur=0 root cause** | ⚠️ **unknown — Phase 1 produces this** | this sprint | Time-box: 3 hours; if no clear signal, default to "fix all 3 hypotheses simultaneously" — slightly wasteful but unblocks Phase 2-4 |
| AILANG dev branch is clean | ✅ green | mark | All v0.18.1 work pushed (`abee8522`) + design doc (`361cd6ad`) |
| motoko_agent `motoko-bisect-gap1` branch | ✅ green | sunholo-data/motoko_agent | All v0.18.1 + EADDRINUSE follow-up pushed (`6580adf`) |
| OPENROUTER_API_KEY available | ✅ verified | mark | Required for Phase 5 live runs |
| ANTHROPIC_API_KEY available | ✅ verified | mark | Required for claude-haiku-4-5 baseline (cross-harness comparison context) |
| Wrapper script editable | ⚠️ blocked | mark (manual) | `/Users/mark/go/bin/motoko` outside repo. Workaround: M2b sets `cmd.Dir = motokoHome` directly, makes wrapper's `cd` a no-op. Wrapper edit becomes optional polish. |
| Eval harness `ailang eval-suite --agent-parallel 4` working | ✅ assumed | this repo | Used by Phase 5 acceptance gate |

---

## Sprint JSON (template for sprint-executor)

```json
{
  "sprint_id": "M-MOTOKO-PARALLEL-EXECUTION-ISOLATION",
  "design_doc_path": "design_docs/planned/v0_18_2/m-motoko-parallel-execution-isolation.md",
  "sprint_plan_path": "design_docs/planned/v0_18_2/m-motoko-parallel-execution-isolation-sprint-plan.md",
  "target_version": "v0.18.2",
  "status": "not_started",
  "milestones": [
    {"id": "M1a_LSOF_INSTRUMENTATION", "passes": null, "completed": null, "notes": ""},
    {"id": "M1b_PARALLEL_REPRO", "passes": null, "completed": null, "notes": ""},
    {"id": "M1c_HYPOTHESIS_PICK", "passes": null, "completed": null, "notes": ""},
    {"id": "M1d_PHASE1_GATE", "passes": null, "completed": null, "notes": "PAUSE for human review"},
    {"id": "M2a_ISOLATION_HELPER", "passes": null, "completed": null, "notes": ""},
    {"id": "M2b_ADAPTER_INTEGRATE", "passes": null, "completed": null, "notes": ""},
    {"id": "M2c_ISOLATION_TESTS", "passes": null, "completed": null, "notes": ""},
    {"id": "M2d_SERIAL_REGRESSION", "passes": null, "completed": null, "notes": ""},
    {"id": "M3a_ENV_SERVER_PICK", "passes": null, "completed": null, "notes": ""},
    {"id": "M3b_REVERT_HEALTHCHECK", "passes": null, "completed": null, "notes": ""},
    {"id": "M3c_RETIRE_EADDRINUSE", "passes": null, "completed": null, "notes": ""},
    {"id": "M3d_SINGLE_SERVER_TEST", "passes": null, "completed": null, "notes": ""},
    {"id": "M4a_WARMCACHE_FLAG", "passes": null, "completed": null, "notes": ""},
    {"id": "M4b_WARMCACHE_LOGIC", "passes": null, "completed": null, "notes": ""},
    {"id": "M4c_WARMCACHE_TEST", "passes": null, "completed": null, "notes": ""},
    {"id": "M5a_PARALLEL_4X5_RUN", "passes": null, "completed": null, "notes": ""},
    {"id": "M5b_ACCEPTANCE_GATE", "passes": null, "completed": null, "notes": ""},
    {"id": "M5c_CHANGELOG_FINALIZE", "passes": null, "completed": null, "notes": ""}
  ],
  "velocity": {
    "target_loc_per_day": 250,
    "estimated_total_loc": 515,
    "estimated_days": 2,
    "investigation_phase_hours": 3
  }
}
```

---

## Pause points for human review

1. **End of M1c** (Phase 1 Findings written) — sprint-executor MUST pause; Mark confirms hypothesis pick before any code change in Phase 2-4.
2. **End of M2d** (serial regression check) — quick sanity-check that v0.18.1 functionality is preserved before Phase 3 begins.
3. **End of M5b** (acceptance gate) — Mark reviews the 5-iteration aggregate and decides whether to ship v0.18.2 or extend the sprint with another fix iteration.

The other gates (Day 1 afternoon, Day 2 morning, Day 2 late morning) are programmatic — they auto-advance if tests pass, only pause if tests fail.

---

## Risks & mitigations

(Inherited from design doc; called out here for sprint-execution awareness.)

| Risk | Phase | Mitigation |
|---|---|---|
| Phase 1 inconclusive | 1 | Time-box at 3h; default to "fix all 3 hypotheses" if no clear signal — slightly wasteful but unblocks Phase 2-4 |
| Hardlink crosses filesystems | 2 | `setupMotokoHome` falls back to full `cp -r` on `EXDEV`, with warning log |
| Per-task MOTOKO_HOME breaks v0.18.1 paths | 2 | M2d serial-regression check is a hard gate before Phase 3 |
| Cache pre-warming itself races | 4 | HealthCheck called serially by eval harness — already single-threaded by construction |
| Wrapper script can't be edited | all | M2b uses `cmd.Dir = motokoHome` directly, sidesteps the wrapper |
| Phase 5 acceptance gate fails (parallel-4 < 95%) | 5 | Iterate: capture which failure mode dominates (infra vs benchmark-correctness), apply targeted fix; OR ship v0.18.2 with `--agent-parallel 2` as the supported max and document |

---

## What ships in v0.18.2

- Per-task `MOTOKO_HOME` isolation (additive — opt-in via env var, fallback to v0.18.1 behavior)
- Single env-server per motoko process (architectural simplification)
- HealthCheck `--warm-cache` flag (opt-in)
- All v0.18.1 functionality preserved (no regressions)
- ≥95% success rate on smoke-tier × `--agent-parallel 4`
- CHANGELOG + design doc + sprint plan all in `design_docs/implemented/v0_18_2/`

## What does NOT ship in v0.18.2

(See Non-Goals in design doc for full list.)
- Cloud Run parallelism validation (separate sprint if it surfaces issues)
- Per-task isolation extracted as a generic `executor.PerTaskIsolation` trait (defer to N=3 rule)
- Removal of v0.18.1's EADDRINUSE handler (kept as defense-in-depth, comment-out only)
- HTTP-level env-server pooling (deferred to v0.19.0+ if needed)

---

**Document created**: 2026-05-08
**Last updated**: 2026-05-08
