# M-GEMINI-EXEC-PROJECT-PLUMBING: Thread GCP Project/Location Env into `ailang exec` Task Construction

**Status**: Planned
**Target**: v0.30.0
**Priority**: P0 (fleet (c) unblocker — blocks live gemini reviewer/evaluator lane, fleet step c0)
**Estimated**: ≤1 day (mechanical CLI plumbing + one unit test)
**Dependencies**: None (managed_agents executor, Task.GCPProject/GCPLocation fields, and eval-harness plumbing all already exist)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is CLI-wiring only — no language, runtime, or type-system change. Scored for completeness:

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No language semantics touched; env→Task mapping is a pure read at CLI startup |
| A2: Replayability | 0 | No trace impact |
| A3: Effect Legibility | 0 | No effects added; the GCP call already exists behind the executor |
| A4: Explicit Authority | +1 | Project must be EXPLICITLY set via env; no ambient/default project is invented — misconfig still fails loud |
| A5: Bounded Verification | 0 | No verification impact |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Unblocks the agentic gemini lane for autonomous fleet roles (reviewer/evaluator) via the CLI |
| A8: Minimal Syntax | +1 | Zero new syntax, zero new flags — reuses the repo-wide env convention |
| A9: Cost Visibility | 0 | No cost model change |
| A10: Composability | +1 | Same Task fields the eval harness and sibling executors already thread; one convention, all paths |
| A11: Structured Failure | +1 | Preserves the existing loud `managed_agents: GCP project not set` error when env is absent |
| A12: System Boundary | 0 | Boundary (Vertex AI HTTP call) unchanged; only who fills in the address changed |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted — no silent default project
- [x] A7 (Machines First): Optimizes for machine (fleet) access to the gemini lane

## Problem Statement

`ailang exec gemini "<directive>"` fails with:

```
managed_agents: GCP project not set (Task.GCPProject or executor config)
```

Live-reproduced at HEAD `9e8504ccb`. The agentic gemini provider (managed_agents / Vertex AI
Managed Agents) is therefore unreachable via the `ailang exec` CLI outside the eval harness.

**Current State:**
- The eval harness works: it sets `Task.GCPProject`/`Task.GCPLocation` per-model from
  `models.yml` (`internal/eval_harness/models.go:28-29`, threaded at
  `internal/eval_harness/agent_runner_multi.go:252-253`).
- The CLI path does NOT: `cmd/ailang/exec.go:336` builds `executor.Task{}` with
  ID/Directive/SystemPrompt/Workspace/Timeout/Model but never sets `GCPProject`/`GCPLocation` —
  even when `AILANG_CLOUD_PROJECT`/`GOOGLE_CLOUD_PROJECT` are set in the environment.

**Impact:**
- Blocks fleet item (c)'s parked M0/M4 (live gemini reviewer/evaluator lane) and any
  gemini reviewer/evaluator role driven through `ailang exec`.
- This doc is the fleet step **c0** unblocker.

## Root Cause (verified in code at HEAD `9e8504ccb`)

| Fact | Evidence |
|------|----------|
| Task struct HAS the fields | `internal/executor/executor.go:88-89` (`GCPProject`, `GCPLocation`) |
| managed_agents CONSUMES them | `internal/executor/managed_agents/managed_agents.go:136-142` (`project := task.GCPProject`, falls back to executor default) |
| Executor default project is EMPTY by design | `managed_agents.go:41-45` (`project: ""` — "Filled per-task") |
| Empty project errors loud | `internal/executor/managed_agents/client.go:83` (`GCP project not set`) |
| Sibling executors already thread it | `internal/executor/codex/codex.go:160`, `internal/executor/claude/claude.go:247` (both `GCPProject: task.GCPProject`) |
| CLI never fills it | `cmd/ailang/exec.go:336` — `executor.Task{}` construction omits both fields |
| Repo env convention exists | `internal/coordinator/daemon_tasks_init.go:315-317`: `AILANG_CLOUD_PROJECT`, fallback `GOOGLE_CLOUD_PROJECT` (same in `cmd/ailang/coordinator_cloud.go:68-70`) |
| Location default is `"global"` | `internal/executor/managed_agents/client.go:32` (`defaultLocation = "global"`); README: "Region locked to `global`. No us-central1 / europe-west fallback." |

So the eval harness path works and the CLI path does not — a pure plumbing gap.

## Goals

**Primary Goal:** `ailang exec gemini` reaches the managed_agents backend when the standard
GCP project env vars are set, with zero behavior change for every other provider and for the
unset-env case.

**Success Metrics:**
- `AILANG_CLOUD_PROJECT=<proj> ailang exec gemini "reply with exactly: ok"` no longer errors
  with "GCP project not set" (given ADC present).
- Unset-env behavior byte-identical to today (same loud error).
- Zero diff in codex/claude/opus exec behavior.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Env precedence: `AILANG_CLOUD_PROJECT` → `GOOGLE_CLOUD_PROJECT` | Must match the existing repo convention (coordinator) or we fork the config surface | design doc (follows `daemon_tasks_init.go:315-317`) | design | low |
| Location: read `GOOGLE_CLOUD_LOCATION` if set, else leave `Task.GCPLocation` EMPTY | Empty falls through to the executor's own `defaultLocation = "global"` (`client.go:32`) — the CLI must NOT invent a second default that could drift from the executor's | design doc | design | low |
| NO silent default project | If neither env is set, keep the existing loud managed_agents error — a silent default hides misconfig (Critical Principle #2) | design doc | design | low |
| Set fields unconditionally on the shared Task (not gemini-gated) | Sibling executors already thread `task.GCPProject` identically (codex.go:160, claude.go:247) — env override reaching them is the same convention, not a regression | design doc | design | low |

### Design Freeze

All decisions above are low change-cost and resolved in this doc:

- [x] Env precedence = coordinator convention (`AILANG_CLOUD_PROJECT` → `GOOGLE_CLOUD_PROJECT`)
- [x] No CLI-side location default — empty string defers to executor's `"global"`
- [x] No silent project fallback

## Solution Design

### Overview

In `cmd/ailang/exec.go`, resolve project/location from env with the existing repo convention
and set them on the `executor.Task{}` at construction (line 336). ~10 LOC + a small helper.

### Proposed Change

```go
// resolveGCPProjectEnv returns the GCP project for exec tasks, using the same
// precedence as the coordinator (daemon_tasks_init.go): AILANG_CLOUD_PROJECT
// first, then GOOGLE_CLOUD_PROJECT. Empty when neither is set — the executor
// fails loud downstream (no silent default project).
func resolveGCPProjectEnv() string {
	if p := os.Getenv("AILANG_CLOUD_PROJECT"); p != "" {
		return p
	}
	return os.Getenv("GOOGLE_CLOUD_PROJECT")
}
```

And in `executeCLI` (`exec.go:336`):

```go
	task := &executor.Task{
		ID:           taskID,
		Directive:    directive,
		SystemPrompt: systemPrompt,
		Workspace:    workspace,
		Timeout:      timeout,
		Model:        model,
		GCPProject:   resolveGCPProjectEnv(),
		GCPLocation:  os.Getenv("GOOGLE_CLOUD_LOCATION"), // empty → executor default ("global")
	}
```

Notes:
- No new default location string on the CLI side — an empty `GCPLocation` falls through to
  `managed_agents`'s own `defaultLocation = "global"` (`client.go:32`), keeping ONE source of
  truth for the default.
- No project fallback beyond the two envs — unset stays unset and the existing
  `client.go:83` error fires (fail loud).
- Non-gemini executors: codex/claude already read `task.GCPProject` into their own
  subprocess env override (`environment.go:43-47`) with the same meaning; opus/other
  providers ignore the fields. No behavior change when env is unset (fields stay `""`,
  which is today's value).

### Implementation Plan

**Phase 1: Plumbing** (~1h)
- [x] Add `resolveGCPProjectEnv()` helper to `cmd/ailang/exec.go`
- [x] Set `GCPProject`/`GCPLocation` on the Task construction at `exec.go:336`

**Phase 2: Regression guard** (~1-2h)
- [x] Unit test in `cmd/ailang/exec_test.go` (file exists): env-injected via `t.Setenv`,
      asserts precedence (`AILANG_CLOUD_PROJECT` wins over `GOOGLE_CLOUD_PROJECT`),
      fallback, and empty-when-unset. No live GCP call — test the resolver and/or the
      constructed Task fields directly. (`TestResolveGCPProjectEnv` +
      `TestExecTaskGCPFieldsFromEnv`)

**Phase 3: Verification** (~1h)
- [x] `make test` green (`go test ./cmd/ailang/...` green; `go build ./cmd/ailang/` green)
- [ ] Live probe (needs ADC): `AILANG_CLOUD_PROJECT=<proj> ailang exec gemini "reply with exactly: ok"` — reaches backend *(deferred to controller: headless run has no ADC; command + expected output recorded in executor report)*
- [ ] Live probe: `env -u AILANG_CLOUD_PROJECT -u GOOGLE_CLOUD_PROJECT ailang exec gemini "x"` — existing loud error unchanged *(deferred to controller: recorded in executor report)*
- [x] CHANGELOG entry (`changelogs/v0.18-current.md`, `### Fixed`)

### Files to Modify/Create

**Modified files:**
- `cmd/ailang/exec.go` — resolver helper + 2 fields on Task construction (~15 LOC)
- `cmd/ailang/exec_test.go` — env-precedence regression test (~40 LOC)

No new files.

## Examples

### Example 1: Fleet gemini reviewer via CLI

**Before:**
```
$ AILANG_CLOUD_PROJECT=ailang-multivac-dev ailang exec gemini "reply with exactly: ok"
Error: managed_agents: GCP project not set (Task.GCPProject or executor config)
```

**After:**
```
$ AILANG_CLOUD_PROJECT=ailang-multivac-dev ailang exec gemini "reply with exactly: ok"
ok
```

### Example 2: Misconfig still fails loud (unchanged)

```
$ env -u AILANG_CLOUD_PROJECT -u GOOGLE_CLOUD_PROJECT ailang exec gemini "x"
Error: managed_agents: GCP project not set (Task.GCPProject or executor config)
```

## Acceptance Criteria

1. With `AILANG_CLOUD_PROJECT` (or `GOOGLE_CLOUD_PROJECT`) set + ADC present,
   `ailang exec gemini "reply with exactly: ok"` reaches the managed_agents backend
   (no "GCP project not set" error).
2. With NEITHER env set, the existing loud error still fires (no silent default project).
3. Non-gemini providers (codex/claude/opus) are UNAFFECTED — this only adds env-plumbing to
   the shared Task construction, which those executors already ignore or already thread
   identically (`codex.go:160`, `claude.go:247`).
4. Regression guard: a Go unit test asserting exec.go's Task construction picks up
   `AILANG_CLOUD_PROJECT`/`GOOGLE_CLOUD_PROJECT` with the correct precedence
   (env-injected via `t.Setenv` — no live GCP call).

## Conflict Surface

This change touches `cmd/ailang/exec.go` (listed as a conflict-surface path), so enumerated
explicitly:

1. **What positions does this extend?** None syntactic/semantic — no parser, types, codegen,
   elaborate, eval, or VM code is touched. The change is CLI wiring: two struct fields filled
   from env at Task construction.
2. **What else lives in this position?** The `executor.Task{}` at `exec.go:336` is shared by
   ALL agentic providers (claude, codex, gemini/managed_agents, …) via
   `resolveAgenticExecutorName` (`exec.go:317`). The new fields are already part of the Task
   contract: codex (`codex.go:160`) and claude (`claude.go:247`) thread `task.GCPProject` into
   their subprocess env override (`environment.go:43-47`); managed_agents uses it as the API
   target (`managed_agents.go:136`). Executors that don't read the fields see no change.
3. **Behavioral delta for non-gemini providers:** only when `AILANG_CLOUD_PROJECT`/
   `GOOGLE_CLOUD_PROJECT`/`GOOGLE_CLOUD_LOCATION` are set — then codex/claude subprocesses
   get an explicit env override carrying the SAME value the ambient env already had, which is
   a no-op in effect. When unset, fields are `""` — byte-identical to today.
4. **Programs that MUST still work:**
   - `ailang exec claude "..."` / `ailang exec codex "..."` with and without the env set
   - `ailang eval-suite` gemini models (eval harness sets Task.GCPProject itself from
     models.yml — per-model value at `agent_runner_multi.go:252-253` is set AFTER/independently
     of CLI construction; the eval path does not go through `executeCLI`, verified: the harness
     builds its own Task)
   - `env -u … ailang exec gemini` → existing loud error (AC2)
5. **Deliberate changes:** exactly one — gemini via CLI now works when project env is set.

**Overall: LOW conflict.** Touches ONLY `cmd/ailang/exec.go` + one test file. Does NOT touch
parser/types/codegen/elaborate.

## Testing Strategy

**Unit tests:**
- `resolveGCPProjectEnv` precedence: AILANG_CLOUD_PROJECT wins; GOOGLE_CLOUD_PROJECT
  fallback; empty when neither set (`t.Setenv`).
- Task construction picks up the resolved values (and `GOOGLE_CLOUD_LOCATION` when set).

**Integration tests:**
- None required (live GCP call needs ADC + billing; covered by manual probes below).

**Manual testing:**
- Live probe with env + ADC (AC1); live probe with env stripped (AC2); one
  `ailang exec claude` smoke to confirm AC3.

## Deferred Decisions

The following are intentionally left open for the implementer:

- Whether the env resolution lives inline in `executeCLI` or in a small helper (helper
  recommended above for testability) — agent may choose.
- Exact test structure (table-driven vs. separate cases) — agent may choose.

## Non-Goals

**Not attempted in this feature:**
- Live gemini reviewer/evaluator wiring — that is fleet (c) M0/M4/M5, a follow-up that this
  doc unblocks.
- Any managed_agents server-side behavior (agent config, sandbox, streaming).
- The CapRemoteSandbox read-only role-scope limit — already documented in fleet (b).
- New CLI flags (`--gcp-project`) — env convention is sufficient for the fleet use case;
  flags can be added later if a human use case appears.
- Changing the eval harness's per-model `models.yml` GCP plumbing (already correct).

## Timeline

**Single day** (~3-4 hours total): Phase 1 (1h) → Phase 2 (1-2h) → Phase 3 (1h).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Env override changes codex/claude subprocess env in surprising ways | Low | Value comes FROM the ambient env, so the override is a semantic no-op; AC3 smoke test confirms |
| CLI-side location default drifts from executor default | Low | Deliberately NO CLI default — empty string defers to `client.go:32` `"global"` |
| Test pollution from env vars | Low | `t.Setenv` auto-restores per test |

## Verification Log

All premises verified live at HEAD `9e8504ccb` (2026-07-16), by controller and re-confirmed
by this doc's author via grep/read:

- `cmd/ailang/exec.go:336` — `task := &executor.Task{` with no GCP fields ✓ (read)
- `internal/executor/executor.go:88-89` — `GCPProject`/`GCPLocation` fields exist ✓ (read)
- `internal/executor/managed_agents/managed_agents.go:136-142` — consumes `task.GCPProject`,
  falls back to executor default which is `""` ✓ (read)
- `internal/executor/managed_agents/client.go:83` — loud error on empty project ✓ (read)
- `internal/executor/managed_agents/client.go:32` — `defaultLocation = "global"` ✓ (grep);
  README.md:153 confirms region locked to `global` ✓
- `internal/coordinator/daemon_tasks_init.go:315-317` — env precedence convention ✓ (read)
- `internal/executor/codex/codex.go:160`, `internal/executor/claude/claude.go:247` — siblings
  thread `GCPProject: task.GCPProject` ✓ (grep)
- `internal/eval_harness/agent_runner_multi.go:252-253` — eval harness sets Task.GCPProject
  from models.yml (why eval works, CLI doesn't) ✓ (read)
- `cmd/ailang/exec_test.go` exists (test lands there, no new file) ✓ (ls)
- Live repro of the failure: controller-verified at HEAD (`ailang exec gemini` → "GCP project
  not set") ✓
- No AILANG-language claims are made in this doc (Go CLI change only) — no `ailang check`
  needed.

## Related Documents

<!-- Auto-populated by Ollama neural search on "gemini exec project plumbing" -->

All neural scores < 0.45 (duplicate gate passed); listed for context only:

**Implemented (may inform design):**
- [design_docs/implemented/v0_6_1/m-exec-gemini-sprint-plan.md](../../implemented/v0_6_1/m-exec-gemini-sprint-plan.md) (0.38) — the original non-agentic gemini exec provider; distinct: this doc fixes the AGENTIC managed_agents path's project plumbing
- [design_docs/implemented/v0_14_2/m-exec-pi-harness-sprint-plan.md](../../implemented/v0_14_2/m-exec-pi-harness-sprint-plan.md) (0.36)
- [design_docs/implemented/v0_5_6/m-msg-sprint-plan.md](../../implemented/v0_5_6/m-msg-sprint-plan.md) (0.35)

**Planned (checked for overlap — none duplicates this):**
- [design_docs/planned/v0_30_0/m-mission-fleet-ab-sprint-plan.md](m-mission-fleet-ab-sprint-plan.md) (0.37) — the fleet program this doc unblocks (step c0); complementary, not overlapping
- [design_docs/planned/v0_29_0/m-file-handling-improvements-sprint-plan.md](../v0_29_0/m-file-handling-improvements-sprint-plan.md) (0.37)
- [design_docs/planned/v1_1_0/m-executor-variants-sprint-plan.md](../v1_1_0/m-executor-variants-sprint-plan.md) (0.34)

## References

- [Design Axioms](/docs/references/axioms)
- `internal/executor/managed_agents/README.md` — managed_agents executor contract (region locked to `global`)
- Fleet item (c) quorum-agentic-verify (PR #400, `0e83a1b12`) — the parked M0/M4 gemini lane this unblocks

## Future Work

- Fleet (c) M0/M4/M5: live gemini reviewer/evaluator wiring on top of this plumbing.
- Optional `--gcp-project`/`--gcp-location` CLI flags if a human-interactive use case appears.

## Quorum Review

Run 2026-07-16T17:44:15Z via `ailang design-quorum` (artifact:
`.ailang/state/mission-quorum/m-gemini-exec-project-plumbing-2026-07-16T17-44-15Z.json`):

- **Synthesis: PROCEED** (total $0.0134)
- `gpt5-6-sol` → **ABSENT** (unreachable) — degraded to N−1, recorded by name, not a silent pass
- `gemini-3-1-pro` → **ABSENT** (invalid) — degraded to N−1, recorded by name, not a silent pass.
  Note the irony: the gemini reviewer lane is degraded for reasons adjacent to what this very
  doc unblocks (fleet (c) gemini lane plumbing).
- Controller (in-session) → **PASS**: every premise verified live at HEAD `9e8504ccb`
  (exec.go:336 Task construction, executor.go:88-89 fields, managed_agents.go:136-142
  consumption, client.go:32/83 default+error, coordinator env convention at
  daemon_tasks_init.go:315-317, sibling threading at codex.go:160/claude.go:247).

Effective quorum = controller-only (both external reviewers absent). Per mission policy this
is a degraded-but-valid PROCEED; premises were controller-verified in code before the run.

---

**Document created**: 2026-07-16
**Last updated**: 2026-07-16
