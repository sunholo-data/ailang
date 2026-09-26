# Sprint Plan: M-V1-SIMPLIFY-S1 — Measure, fix, delete, and fence the core

**Design doc**: [m-v1-simplification-program.md](m-v1-simplification-program.md) (Phase 0, Phase 1.1–1.2, plus the weekly audit skill)
**Sprint ID**: M-V1-SIMPLIFY-S1
**Created**: 2026-09-14

## Summary

First sprint of the six-phase v1.0.0 simplification program. It lands the instruments that make every later sprint safe (a banked metrics baseline, a language-core closure test, a fast core test target, a weekly audit skill), closes the two live bugs the audit found, and deletes the code that is verified to have zero callers. Nothing in this sprint changes language semantics, banked eval data, or any public CLI name.

**Duration:** 3 days (7 milestones, ~1,900 LOC net of which ~3,000 LOC is deletion)
**Dependencies:** none of the seven Design Freeze decisions block this sprint. D6 (untracking `eval_results/`) was checked and is **not** executed here: `.gitignore` explicitly re-includes `eval_results/baselines/`, and the docs BenchmarkDashboard plus `eval-weekly.yml` read it. That stays with the owner.
**Risk Level:** Low (every deletion has a zero-caller grep in the Verification Log; every behaviour change has a regression test)

## Current Status Analysis

### Completed Recently
- ✅ Design doc M-V1-SIMPLIFICATION-PROGRAM committed (6b8b236e6), six audits consolidated
- ✅ Baseline `go test ./internal/... ./cmd/ailang/...` captured before any change (scratchpad `baseline_test.txt`)

### Velocity
- Last 7 days on dev: coordinator handoff fixes, registry scoped keys, riglock — roughly 400–600 LOC/day attended
- This sprint is deletion-heavy; the constraint is verification time, not typing

### Remaining from Design Doc after this sprint
- ⏳ Phase 1.3–1.6 (registration seams, compiler logic out of cmd, ARCHITECTURE regen) — next sprint
- ⏳ Phase 2 (one of each, 14 items) — needs D2 + D3 rulings
- ⏳ Phase 3 (CLI dispatch table) — needs D1 + D7
- ⏳ Phase 4 (navigation) — needs D4 + D5
- ⏳ Phase 5 (release gate)

## Proposed Milestones

### Milestone 1: Simplicity metrics + `make test-core`
**Goal:** `make simplicity-metrics` emits the design doc's Goals table as JSON + markdown and banks it under `.ailang/state/simplicity/`; `make test-core` runs the language packages in under 15 s.
**Estimated:** 220 LOC script + 40 LOC make = 260
**Duration:** 0.5 day

**Tasks:**
- `tools/simplicity_metrics.sh`: closure size (`go list -deps` over the language roots), third-party leak roots, top-level command count (from `ailang --help`), `--help` exit-0 rate, `os.Getenv` sites outside `internal/config`, env vars documented %, packages without a package comment, skill trees, always-on instruction bytes, tracked file count, `test-core` wall time
- `make/code-health.mk`: `simplicity-metrics` target writing `.ailang/state/simplicity/<date>.json` and printing the table
- `make/test.mk`: `test-core` over lexer, parser, ast, types, elaborate, eval, effects, pipeline, link, loader, module, format, iface, core, runtime

**Acceptance Criteria:**
- [ ] `make simplicity-metrics` runs clean from the repo root and banks a JSON file
- [ ] Every row of the design doc's Goals table has a value in the JSON
- [ ] `make test-core` passes in < 15 s wall on this machine
- [ ] Baseline snapshot committed

### Milestone 2: opencode executor adopts `proctree`
**Goal:** Close the live orphan bug. `internal/executor/opencode/opencode.go` uses `proctree.Configure` + `proctree.Kill` like pi and codex do; the seven bare `cmd.Process.Kill()` are gone.
**Estimated:** 30 LOC change + 60 LOC test = 90
**Duration:** 0.25 day

**Tasks:**
- Replace each `cmd.Process.Kill()` with `proctree.Kill(cmd)`; call `proctree.Configure(cmd)` after `exec.CommandContext`
- Regression test (unix build tag): spawn `sh -c 'sleep 30 & wait'` through the same helper path, kill, assert the grandchild `sleep` is gone

**Acceptance Criteria:**
- [ ] `grep -c 'Process.Kill' internal/executor/opencode/opencode.go` = 0
- [ ] Grandchild-orphan test passes; existing opencode stream fixtures pass
- [ ] All tests passing, lint clean

### Milestone 3: Dashboard approve goes through `ProcessApprovalRequest`
**Goal:** Close the live handoff bug. `internal/server/handlers_coordinator.go` approve/reject route calls `coordinator.ProcessApprovalRequest` (the path `handlers_approvals.go:377` already uses) instead of `ResolveApprovalRequest` alone.
**Estimated:** 40 LOC change + 80 LOC test = 120
**Duration:** 0.5 day

**Tasks:**
- Read `handlers_approvals.go:377` and reuse its wiring (store, handoff sender, workspace) — do not build a second one
- Route `/api/coordinator/approve/` and `/reject/` through it; keep the response shape
- Test: a pending approval with a `trigger_on_complete` edge, approved via the dashboard handler, produces a handoff message (the assertion the 2026-09-07 root cause lacked)

**Acceptance Criteria:**
- [ ] `ResolveApprovalRequest` is no longer called directly from `handlers_coordinator.go`
- [ ] Handoff-fires-from-dashboard test passes
- [ ] All tests passing, lint clean

### Milestone 4: Delete verified-dead code
**Goal:** Remove the six clusters the audit proved have zero non-test callers, with the `git log -S` note for each in the commit body.
**Estimated:** −2,956 LOC (four observatory backends, daemon approval family, streaming runner) − ~600 more (headless runner body, cognition, root testutil, benchmark/, output/) + 0 new
**Duration:** 0.75 day

**Tasks:**
- `internal/coordinator/daemon_approval.go`: HandleApproval/HandleRejection/requestHandoffApproval/processHandoffApproval + `sendHandoffMessage` (dupe of `approval_handoff.go`); keep anything else in the file that has callers
- `internal/eval_harness/agent_runner.go`: `runHeadlessSession`, `checkClaudeCLI`, `determineSuccess`, `getErrorMessage`, `getSolutionFilename`, `ClaudeHeadlessResult`; delete `agent_runner_streaming.go`; delete the matching tests in `agent_runner_test.go` (out-of-date tests are removed, per coding-standards). Keep `AgentBenchmarkConfig`, `DefaultAgentConfig`, `ResolvedVerifyTimeout`, `ValidationResult` (54 live uses)
- `internal/observatory/backend_jaeger.go`, `backend_gcp.go`, `backend_gcp_chains.go`, `backend_composite.go` + the `AILANG_ENABLE_GCP_TRACE` branch in `internal/server/server.go:328-353` (nothing sets the flag: repo, multivac repo, plists all grepped)
- `internal/cognition/` (0 importers), root `testutil/` (0 importers), `benchmark/` (2025-11 "temporary"), `output/` (3 tracked txt files)
- After each deletion: `go build ./internal/... ./cmd/ailang/...` and the package's tests

**Acceptance Criteria:**
- [ ] Each deleted symbol's zero-caller grep is quoted in the commit message
- [ ] `go vet ./...` and `make test` green
- [ ] No `AILANG_ENABLE_GCP_TRACE` read remains

### Milestone 5: Root housekeeping
**Goal:** The repo root stops carrying pipeline droppings; `git status` is quiet on a clean checkout.
**Estimated:** file moves + 20 LOC .gitignore
**Duration:** 0.25 day

**Tasks:**
- `git mv` the 8 tracked `sprint_*.json` / `sprint-m-*.json` at root to `.ailang/state/sprints/` (skip any whose twin already exists there — keep the newer, note it) and the 3 `sprint*-plan.md` to `design_docs/planned/`
- Delete the untracked eval droppings (`app_config.json app.log output.txt users.csv users.json widget_price.txt`) and confirm the current harness sets the generated program's cwd to the workspace (if it does not, file it as a Phase 2.6 item rather than fixing here)
- `.gitignore`: `gen/`, `.eval_workspace/`; verify `examples/**/.ailang/` cache dirs are ignored (they show as untracked today)

**Acceptance Criteria:**
- [ ] No `sprint_*` files at repo root
- [ ] `git status --short | grep -v '^??' ` is empty after the sprint on a clean tree
- [ ] Nothing referenced by Makefile/tools/skills was moved without its reference being updated

### Milestone 6: Core closure test + hot path
**Goal:** The language-core boundary is a test; `ailang version` and `ailang fmt` open no database and log nothing.
**Estimated:** 140 LOC test + 40 LOC expected-violations list + 30 LOC main.go = 210
**Duration:** 0.5 day

**Tasks:**
- `internal/diag/closure_test.go` (pattern: `internal/modelreg/leaf_test.go` with its positive control): compute `go list -deps` over the language roots; assert no package from the platform deny-list and no third-party root from {sqlite3, otel, grpc, cloud.google.com, ollama} — **except** those in `internal/diag/closure_expected_violations.txt`. The test also fails if a listed violation is no longer present (the list must be trimmed as leaks are fixed), so it can only shrink
- `cmd/ailang/main.go`: `observatory.CheckHealth` and `checkStaleBinary` run only when the command is not a language command (`run check fmt test watch repl iface verify prompt docs lsp examples compile disasm debug builtins init ai-check version`); Phase 3's dispatch table replaces the list
- Bank the closure size in the metrics JSON

**Acceptance Criteria:**
- [ ] Closure test passes on dev with the nine known leaks listed; removing a line makes it fail (positive control)
- [ ] `ailang version 2>&1 | wc -l` = 1 and `strace`-free proof: `AILANG_STATE_DIR=/nonexistent ailang fmt --check examples/hello.ail` succeeds without creating the dir
- [ ] All tests passing

### Milestone 7: Weekly `simplicity-audit` skill
**Goal:** A skill Mark can run weekly (or schedule) that re-runs the metrics, diffs against the last banked snapshot, and reports regressions in the classes this program fixes: new duplicate helper names across packages, new `os.Getenv` outside config, new commands without `--help`, new packages without a package comment, closure growth, tracked-file growth, instruction-surface growth, root clutter, dangling `ailang <cmd>` citations in skills.
**Estimated:** SKILL.md ~150 lines + `scripts/audit.sh` ~200 LOC + `scripts/dup_symbols.sh` ~60 LOC
**Duration:** 0.5 day

**Tasks:**
- `.claude/skills/simplicity-audit/SKILL.md` with a description that triggers on "simplicity audit", "duplicate implementations", "codebase health", "weekly audit"; explicit non-overlap note with `codebase-organizer` (file sizes) and `sonarcloud-triage`
- `scripts/audit.sh`: runs `tools/simplicity_metrics.sh`, diffs against the previous snapshot, prints a delta table and a ranked "new since last audit" list; exits 2 on any regression so it can gate
- `scripts/dup_symbols.sh`: the repeated-utility-name census from the audit (`truncate`, `fileExists`, `writeJSON`, …) with a per-name package list
- `make simplicity-audit` target; a note in CLAUDE.md's instrument table ("Is the codebase getting more complex?")
- Scheduling: document the `/schedule` or launchd one-liner; do **not** install a launchd job in this sprint (rig launchd changes go through mission-loop-change)

**Acceptance Criteria:**
- [ ] `make simplicity-audit` against the M1 baseline reports zero regressions on the sprint's final tree
- [ ] Introducing a deliberate `os.Getenv` in a test file and re-running reports it (positive control), then revert
- [ ] Skill description passes the trigger check in skill-builder's guidance (one sentence of what, one of when)

## Success Metrics
- Test coverage: unchanged or up (deletions remove untested code)
- `make test` green before and after; `make test-core` < 15 s
- Two regression tests added (grandchild kill, dashboard handoff)
- ~3,500 LOC deleted, ~700 added
- Documentation: CLAUDE.md instrument row, CHANGELOG entry under Unreleased

## Dependencies
- None blocking. The baseline test run must finish before M4 deletions start (it is running).

## Open Questions (for Mark, not blocking)
- D6: keep `eval_results/baselines/` tracked? It is deliberately re-included in `.gitignore` and consumed by the dashboard component and the weekly workflow. Left untouched.
- Sprint JSON tracking policy (78 tracked / 52 untracked under `.ailang/state/sprints/`): left untouched; the root files are moved there, not re-policied.
