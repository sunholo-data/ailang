# M-MOTOKO-PARALLEL-EXECUTION-ISOLATION: Per-task isolation so `--agent-parallel ≥ 2` works

**Status**: Planned
**Target**: v0.18.2 (patch release on top of v0.18.1)
**Priority**: P0 (High — blocks reliable cross-harness comparisons in the eval harness; serial-only is a real-world cap on dev velocity)
**Estimated**: 2 days (~10–14 hours, ~250 LOC across both repos, with strict investigation-first phase 1 reproducing the dur=0 crash before any code change)
**Dependencies**:
- ✅ M-MOTOKO-EXECUTOR-ADAPTER (v0.18.0) — adapter exists
- ✅ M-MOTOKO-EVAL-HARNESS-HARDENING (v0.18.1, shipped 2026-05-08) — serial path is sound; this sprint extends to parallel
- ✅ M-MOTOKO-EVAL-INSTRUMENTATION (motoko commits `0c006be` + `84fa449`) — schema v1 JSONL contract

**Author**: Claude Sonnet 4.6 + Mark
**Created**: 2026-05-08

**Source event**: First live `ailang eval-suite` paired comparison (3 harnesses × 15 smoke benchmarks × `--agent-parallel 2`) on 2026-05-08 hit 5/15 motoko failures — all `dur_s=0` with "motoko terminated without emitting run_summary". Adding an `EADDRINUSE` retry/yield handler did not fix it (and arguably regressed it: 40/45 → 37/45). User feedback: "ok but we need to fix the parallel runs" and "lets make a design doc and get to the bottom of it". Captured eval results: `eval_results/v0_18_1_3harness_smoke/` (v1, 40/45) and `eval_results/v0_18_1_3harness_smoke_v2/` (v2, 37/45).

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +2 | **Fixes critical violation**: parallel motoko runs today are nondeterministic (which session "wins" the shared state changes per-run). Per-task isolation restores determinism. |
| A2: Replayability | +1 | A given `(task, model, harness)` triple now produces a reproducible result regardless of how many siblings are running |
| A3: Effect Legibility | 0 | No effect-row changes |
| A4: Explicit Authority | +1 | Per-task workdir + per-task env-server eliminates accidental cross-task FS access ("session A's env-server runs cmd from session B's workdir") |
| A5: Bounded Verification | 0 | No type-system changes |
| A6: Safe Concurrency | +2 | **Core fix**: eliminates 3 race conditions (cache writes, env-server bind, shared state in MOTOKO_REPO). Mirrors the M-SERVE-API-CONCURRENCY (v0.9.4) per-request-isolation pattern |
| A7: Machines First | +1 | Eval harness can now drive motoko at the same parallelism level as Pi/Claude/Codex/opencode — unblocks honest cross-harness comparisons |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | No change |
| A10: Composability | 0 | No change |
| A11: Structured Failure | +1 | Cache-collision crashes today produce empty JSONL ("likely crash") — post-fix, errors are typed events on disk |
| A12: System Boundary | +1 | Per-task workdir is a clearer boundary than "cd into shared repo and hope" |

**Net Score: +9** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): **Fixes** nondeterministic parallel-execution behavior
- [x] A3 (Effects): No new effects; no hidden side effects
- [x] A4 (Authority): **Fixes** accidental cross-task FS authority via shared env-server
- [x] A7 (Machines First): Removes the 1-at-a-time human-velocity cap on eval-suite

---

## Systemic Audit

**Is this a one-off or part of a larger pattern?**

**Pattern.** This is exactly the same architectural class as M-SERVE-API-CONCURRENCY (v0.9.4): a system designed for single-active-task execution being asked to handle concurrent tasks, with shared mutable state causing race conditions.

**Cross-executor comparison** (audit done before designing):

| Executor | Workdir model | Env-server | Cache | Parallel-safe? |
|---|---|---|---|---|
| `claude` | `cmd.Dir = task.Workspace` | None | None | ✅ Yes (each invocation is self-contained) |
| `gemini` | `cmd.Dir = task.Workspace` | None | None | ✅ Yes |
| `codex` | `cmd.Dir = task.Workspace` | None | None | ✅ Yes |
| `opencode` | `cmd.Dir = task.Workspace` | None | None | ✅ Yes |
| `pi` | `cmd.Dir = task.Workspace` | None | None | ✅ Yes |
| `motoko` | wrapper does `cd $MOTOKO_REPO` (shared) | 2 per task (inline + auto_start) | shared `.ailang/cache/` in MOTOKO_REPO | ❌ No (this sprint) |

**Motoko is the outlier.** Every other CLI-subprocess executor follows the "per-task isolation" pattern: `cmd.Dir = task.Workspace`, no shared filesystem state, no shared services. Motoko inherited a different design (long-lived TUI with embedded env-server) and the adapter wraps it without re-isolating.

**Anti-pattern this sprint avoids**: incremental special-casing of each parallel-mode failure (port race → backend-share race → cache race → ...). The unified fix is "make motoko look like Pi from the adapter's perspective."

---

## Problem Statement

The v0.18.1 sprint (yesterday) closed all serial-execution gaps for motoko. The first parallel test today (3 harnesses × 15 benchmarks × `--agent-parallel 2`) revealed that parallel-mode introduces a NEW class of failures the v0.18.1 fixes don't cover.

### Observed Symptoms

**v1 parallel run** (2026-05-08, before EADDRINUSE handler) → 40/45 (88.9%):
- 5 motoko failures, 4 of which are `dur_s=0` + "motoko terminated without emitting run_summary"
- Failed benchmarks: `adt_option`, `balanced_parens`, `binary_tree_sum`, `canonical_normalization` (alphabetically first 4 = first parallel batch), `numeric_modulo`

**v2 parallel run** (2026-05-08, after EADDRINUSE retry/yield handler) → 37/45 (82.2%):
- Same alphabetically-early failures, plus 3 additional sporadic ones
- Worsened by my "fix" — the EADDRINUSE handler causes losing siblings to yield to a winning sibling that's bound to a DIFFERENT workdir, so the AILANG runtime tries to /exec on the wrong filesystem

**Serial run** (`--agent-parallel 1`, 2026-05-08) → 42/45 (93.3%):
- 3 motoko failures, all benchmark-correctness misses (model produced AILANG that didn't match expected stdout) — NOT infrastructure failures

The delta (3 extra parallel failures, all `dur_s=0`) is the parallel-execution surface area we need to close.

### The dur=0 Signature

`dur_s=0` + 0-byte JSONL means motoko crashed BEFORE the TS SessionLogger constructor ran — i.e., before any AILANG code executed in the runtime. The crash is at the wrapper layer or the bun startup layer, NOT in the agent loop.

Three architectural hypotheses for this crash (this sprint will bisect and confirm which):

**H1: Cache-write race.** When the motoko branch `motoko-bisect-gap1` has uncommitted source changes (or a fresh clone), `ailang run src/core/supervisor.ail` recompiles motoko's `.ail` modules and writes to `MOTOKO_REPO/src/core/.ailang/cache/compile/modules/.../core.gob`. Two parallel `ailang run` invocations both detect fresh sources, both compile, both `os.WriteFile()` the same .gob — last writer wins, but partial writes can corrupt the file. Subsequent reads (by either sibling, or by a third sibling) hit gob deserialization errors and crash before the runtime initializes.

**H2: Per-task env-server isolation gap.** Currently `index.ts` calls `startEnvServer(envPort, workdir)` INLINE, AND `supervisor.ail`'s `auto_start` spawns `env-server-main.ts` as a SEPARATE process. With my v0.18.1 fixes both end up on the kernel-picked `boundPort`. The EADDRINUSE handler I added makes losing siblings yield, but yielding means the AILANG runtime then connects to the WINNING sibling's env-server — which is bound to the WINNING sibling's workdir, not the loser's. All FS operations from the loser's agent loop now silently target the winner's tmp dir.

**H3: Shared registry state.** Some startup-time write to `MOTOKO_REPO/src/core/ext/registry_generated.ail` or `MOTOKO_REPO/.motoko/store/` — both parallel sessions race on the same file. (Less likely than H1/H2 because the registry is checked-in source, but worth ruling out via instrumentation.)

### Impact

- **Tactical**: `ailang eval-suite --agent-parallel 2` is the standard release-baseline command. Motoko-on-parallel = unreliable means we either (a) run motoko serially while everything else parallelizes (bottleneck — motoko is the slowest harness) or (b) accept noisy comparison data.
- **Strategic**: M5 of M-MOTOKO-EXECUTOR-ADAPTER is the threshold-measurement experiment ("does motoko's harness lift cheap models?"). Each motoko-* row in that experiment needs many runs for statistical confidence. Serial-only = days of wall-clock to get 30 runs per (model, benchmark); parallel = hours.
- **Process**: this is the second sprint in 24 hours triggered by motoko's parallel-execution model diverging from the rest of the executor fleet. A unified fix now prevents a steady drip of parallel-only bugs going forward.

---

## Goals

**Primary Goal:** Per-task isolation that makes motoko safe at `--agent-parallel ≥ 4` — each motoko invocation owns its filesystem state (cache + env-server + workdir) without sharing anything mutable with siblings.

**Success Metrics:**
- 5 consecutive runs of the 15-benchmark smoke tier × motoko-claude-haiku-4-5 at `--agent-parallel 4` see ≥95% success rate (i.e. ≤3 failures per 60 runs total, all benchmark-correctness misses NOT infrastructure failures)
- Adding motoko to the standard release-baseline `ailang eval-suite ... --agent-parallel 2` produces results indistinguishable in failure-rate from the other 5 executors (claude, gemini, codex, opencode, pi)
- Wall-clock reduction: 15-benchmark smoke at `--agent-parallel 4` runs in ≤8 minutes (vs ~22 minutes serial)
- Zero `dur_s=0` "motoko terminated without emitting run_summary" failures across 60+ runs
- Phase 1 instrumentation pinpoints which of H1/H2/H3 is the actual cause (or surfaces a 4th)

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|---|---|---|---|---|
| Per-task `MOTOKO_HOME` directory (clone or hardlink-mirror of MOTOKO_REPO into a per-task tmp dir) | Eliminates ALL shared filesystem state between parallel sessions; matches Pi/claude pattern. Alternative: per-task `.ailang/cache/` overlay only — cheaper but doesn't cover registry / store races | human | design | high (changes adapter spawn from "cd shared, run" to "isolate, then run") |
| Single env-server per motoko process (drop the inline-server-in-bun OR drop the auto_start) | Today's two-server architecture (inline in bun + auto_start spawn) is the proximate cause of the EADDRINUSE-retry-creates-cross-task-routing bug. ONE env-server per session (kernel-picked port, owned by that session's workdir) is simpler and race-free | human | design | med (changes either index.ts startEnvServer call OR backend.ail auto_start branch) |
| Cache-warming as part of HealthCheck (single-threaded compile pass before parallel sessions launch) | Even with per-task isolation, if H1 is real (cache-write race), the FIRST run must complete a clean cache build — subsequent runs read-only from a hot cache. HealthCheck is the natural pre-flight hook | agent | implementation | low |
| Investigation-first ordering for the dur=0 crash — debug instrumentation BEFORE writing the fix | We have 3 hypotheses (H1/H2/H3) and don't know which is actually firing. Picking the wrong one = wasted sprint. Mirrors the v0.18.1 gap #1 phase-1 pattern (which paid off) | human | design | low (but blocks phases 2-3 until known) |
| Hardlink-mirror vs full-copy for MOTOKO_HOME | Hardlink saves disk + setup time but doesn't isolate writes (both link targets point to same inode). Full copy isolates but is slow (~5s for the AILANG cache alone). Pick after Phase 1 nails down which paths are actually written-to | agent | implementation | med (the answer informs the per-task setup speed, which affects HealthCheck latency) |
| MOTOKO_REPO discovery semantics: stays as the "source-of-truth template repo", with MOTOKO_HOME being the per-task working copy | Keeps the existing v0.18.1 MOTOKO_REPO discovery code intact; adds a layer beneath rather than replacing | agent | implementation | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **Per-task isolation strategy**: full clone vs hardlink-mirror vs cache-only-overlay. Phase 1 must inform this — DON'T pick before instrumentation tells us which paths are written.
- [ ] **Single-env-server architecture**: drop inline OR drop auto_start. Recommend dropping auto_start (the inline one in bun is tied to bun's lifetime, which already matches the per-task lifetime correctly).
- [ ] **HealthCheck cache-warming**: opt-in (HealthCheck builds cache only when `--healthcheck-warm-cache` is set) vs always-on (HealthCheck builds cache unconditionally). Recommend opt-in to keep HealthCheck fast for non-eval use.
- [ ] **MOTOKO_HOME naming + lifecycle**: `${TMPDIR}/motoko-task-<uuid>/` cleaned up by deferred `os.RemoveAll` in adapter. Confirm before sprint-executor runs.

---

## Deferred Decisions

The following are intentionally left open for the implementer:

- The exact debug instrumentation approach for Phase 1 (strace vs lsof loop vs adding println to the wrapper script vs OTEL trace inspection) — agent may choose; recommend `lsof | grep MOTOKO_REPO` snapshots taken at 100ms intervals during a parallel run to surface filesystem contention
- The hardlink-mirror implementation (cp -al on Linux/macOS vs Go's filepath.Walk + os.Link) — agent may choose if Phase 1 picks this strategy
- Whether to upstream the per-task isolation as a generic `executor.PerTaskIsolation` trait (so future executors that need similar isolation don't reinvent it) — agent may choose; recommend yes if motoko is the only consumer for now (extract later when N=2)
- Per-task env-server lifecycle: kill on adapter cleanup vs let it die when bun exits (Node automatically kills child sockets) — agent may choose
- Whether to remove the wrapper's `cd $MOTOKO_REPO` step entirely once MOTOKO_HOME exists, OR keep both — agent may choose; recommend keeping for back-compat with interactive `motoko "task"` use

---

## Conflict Surface

This sprint touches both `internal/executor/motoko/` (AILANG side) and `motoko_agent/src/core/backend.ail` + `src/tui/src/index.ts` (motoko side). Per the project rules, conflict surface analysis is required.

### What positions does this change extend?

1. **`Task.Workspace` semantic**: today, `task.Workspace` is the directory the agent EDITS files in (the "user repo" being worked on). After this sprint, the adapter ALSO needs a SEPARATE per-task dir for motoko's own state (cache, store, registry). This is a NEW semantic distinction — operator must understand the difference.

2. **`MOTOKO_REPO` env var contract**: today (v0.18.1) it points to the SHARED motoko_agent fork. After this sprint, MOTOKO_REPO stays the source-of-truth template and MOTOKO_HOME is added as the per-task working copy. Backwards-compat: if MOTOKO_HOME is unset, fall back to MOTOKO_REPO behavior (single-task mode).

3. **Wrapper `cd` semantic**: `cd $MOTOKO_REPO` vs `cd $MOTOKO_HOME` — for parallel-safety, must `cd $MOTOKO_HOME`. Affects the wrapper script (`/Users/mark/go/bin/motoko`, dev-installed) — which Claude Code can't edit without permission.

### What constructs already live in this position?

- The wrapper's `cd $MOTOKO_REPO` is also relied upon by the v0.18.1 MOTOKO_REPO-discovery fallback in `findSessionJSONL` (parser.go) and `mirrorProfileFromRepo` (TS). Those code paths assume cd happens; per-task isolation must preserve `pwd` semantics for them.
- The TS `index.ts` `await startEnvServer(envPort, workdir)` is called UNCONDITIONALLY — even when `dogfood` profile says `backend.mode = "external_http"`. Today this means there are always TWO env-servers per session (inline + spawned). Removing one is a behavior change for any operator who relied on the inline server's lifecycle (they'd get a different shutdown order).
- `start_or_connect_backend` health-check (added in v0.18.1) explicitly tries to SHARE env-servers across sessions. This sprint REVERSES that decision (per-task isolation) — must update or remove the health-check shortcut.

### Disambiguation strategy

- Per-task: introduce `MOTOKO_HOME` as the new explicit boundary; `MOTOKO_REPO` keeps its current meaning (source-of-truth template); operators using the v0.18.1 single-task setup see no change (MOTOKO_HOME defaults to MOTOKO_REPO when unset).
- Two-server collapse: if Phase 1 picks "drop auto_start", `start_or_connect_backend` becomes the connect-only branch; if Phase 1 picks "drop inline", we keep auto_start but skip the inline `startEnvServer` call in `index.ts`. The decision is informed by which side already owns the workdir-aware filesystem boundary (auto_start gets workdir from supervisor's `cli.workdir`; inline gets it from `process.env.WORKDIR`).

### Programs that MUST still work post-change (3-5 fixtures)

1. **Interactive `motoko "task"` (single-user, no MOTOKO_HOME)**: `cd /tmp/foo && motoko "Print 42"` must work without setup
2. **AILANG eval-suite serial (`--agent-parallel 1`)**: existing v0.18.1 behavior preserved exactly
3. **AILANG eval-suite parallel (`--agent-parallel 4`)**: 95% success rate sustained over 5 consecutive runs (the new acceptance gate)
4. **Cloud Run dispatch (`agent-motoko` Cloud Run Job)**: parallel execution in cloud (Job's `parallelism: 4` setting) works without serialization workaround
5. **`go run ./cmd/smoke-motoko`**: standalone smoke runner (which today bypasses some adapter setup) still produces expected `Success=true, CostUSD>0` Result

### Intentional incompatibilities

- **None** for the MOTOKO_HOME approach (additive — new env var, fallback to old behavior)
- If we choose "drop inline server" path, the bun process no longer answers /health on `localhost:${ENV_PORT}` directly — only via the spawned env-server. Operators who curl'd the bun process for status: behavior change, document in CHANGELOG.

---

## Solution Design

### Overview

Three coordinated changes that together eliminate ALL shared mutable state between parallel motoko sessions:

1. **Per-task `MOTOKO_HOME`** (AILANG adapter): adapter creates `${TMPDIR}/motoko-task-<uuid>/` per-task as a hardlink-mirror (or copy, TBD by Phase 1) of MOTOKO_REPO. Sets `MOTOKO_HOME=<path>` in spawn env. Wrapper script (or — if we can't edit it — the AILANG adapter's spawn invocation) `cd`'s into MOTOKO_HOME instead of MOTOKO_REPO.

2. **Single env-server per session** (motoko TS): drop one of the two env-server starts. Likely candidate: drop `auto_start` in `backend.ail` (keeps the inline `startEnvServer` in `index.ts`), since the inline one is already tied to bun's lifetime and gets the per-task workdir from `process.env.WORKDIR`.

3. **Cache pre-warming opt-in** (AILANG adapter HealthCheck): HealthCheck has a new `--warm-cache` mode that runs `cd $MOTOKO_HOME && ailang check src/core/supervisor.ail` once SERIALLY before the eval harness launches the parallel batch. After this, every per-task MOTOKO_HOME inherits a hot cache, parallel sessions don't write to .ailang/cache/.

### Architecture

```
TODAY (broken at parallel ≥ 2):
  AILANG eval-suite
    ├─ Task A: cmd("motoko", task_a_args)
    │    └─ wrapper: cd $MOTOKO_REPO ──┐
    │         └─ bun                   │
    │              ├─ inline env-server (port X)  │ shared
    │              └─ ailang run                  │ MOTOKO_REPO
    │                   └─ env-server-main (port X, EADDRINUSE!)  │
    │                                                              │
    └─ Task B: cmd("motoko", task_b_args)                          │
         └─ wrapper: cd $MOTOKO_REPO ──────────────────────────────┘ ← race
              └─ bun
                   ├─ inline env-server (port Y)
                   └─ ailang run
                        └─ env-server-main (port Y, EADDRINUSE!)

POST-FIX (this sprint):
  AILANG eval-suite
    ├─ HealthCheck: warm cache in MOTOKO_REPO (serial, once)
    ├─ Task A: setup MOTOKO_HOME_A (hardlink-mirror) → cmd("motoko", env: MOTOKO_HOME=A)
    │    └─ wrapper: cd $MOTOKO_HOME_A
    │         └─ bun (workdir = task_a.Workspace)
    │              └─ inline env-server only (port X, owned by A)
    │                   └─ ailang run (uses inline, no auto_start spawn)
    │
    └─ Task B: setup MOTOKO_HOME_B → cmd("motoko", env: MOTOKO_HOME=B)
         └─ wrapper: cd $MOTOKO_HOME_B
              └─ bun (workdir = task_b.Workspace)
                   └─ inline env-server (port Y, owned by B)
                        └─ ailang run

  → No shared filesystem state, no shared env-server, no port races possible
```

**Components:**
1. **`internal/executor/motoko/isolation.go`** (new): `setupMotokoHome(repoPath) → (homePath, cleanup)` — creates per-task hardlink-mirror, returns path + deferred cleanup
2. **`internal/executor/motoko/motoko.go` Execute path**: call `setupMotokoHome` before spawn; set `MOTOKO_HOME=<path>` in env; `defer cleanup()`
3. **`internal/executor/motoko/motoko.go` HealthCheck**: optional `--warm-cache` mode that runs `ailang check src/core/supervisor.ail` against MOTOKO_REPO (the template) so subsequent per-task copies inherit a hot cache
4. **`motoko_agent/src/core/backend.ail`**: `start_or_connect_backend` reverts the v0.18.1 health-check-share — auto_start branch now ALWAYS spawns (no sharing). OR the inline `startEnvServer` is dropped from `index.ts` (decided by Phase 1 + design freeze)
5. **Wrapper script** (`/Users/mark/go/bin/motoko`, outside repo): `cd "${MOTOKO_HOME:-$MOTOKO_REPO}"` instead of `cd $MOTOKO_REPO`. If Claude Code can't edit, AILANG adapter's spawn invocation can `cmd.Dir = motokoHome` and the wrapper inherits.

### Implementation Plan

**Phase 1: Investigate (~2 hours)**
- [ ] M1a: Add `lsof | head -1000` snapshots taken at 200ms intervals during a 4-parallel motoko run, capture which files are being read/written by each motoko PID
- [ ] M1b: Add a `strace -e openat,write,unlink` (or macOS `dtruss`) wrapper to ONE of the parallel motoko sessions, capture which writes to MOTOKO_REPO/.ailang/cache or .motoko/store occur
- [ ] M1c: Cross-check observed writes against the 3 hypotheses (H1 cache, H2 env-server, H3 registry) — narrow to one
- [ ] M1d: Acceptance gate: definitive answer on which of H1/H2/H3 (or H4) caused the dur_s=0 crashes, captured in this design doc as a follow-up entry

**Phase 2: Per-task MOTOKO_HOME (~3 hours)**
- [ ] M2a: New `internal/executor/motoko/isolation.go` with `setupMotokoHome(repoPath) → (homePath, cleanup, error)` — defaults to hardlink-mirror, falls back to copy on failure
- [ ] M2b: AILANG adapter `Execute` calls `setupMotokoHome` before spawn, sets `MOTOKO_HOME` env var, `defer cleanup()`
- [ ] M2c: Wrapper script update (`cd "${MOTOKO_HOME:-$MOTOKO_REPO}"`) — Mark to apply manually, OR adapter sets `cmd.Dir = motokoHome` (tested first)
- [ ] M2d: Test: `TestExecute_PerTaskMotokoHome` asserts that two sequential Execute calls with the same task ID get DIFFERENT MOTOKO_HOME paths

**Phase 3: Single env-server (~2 hours)**
- [ ] M3a: Pick: drop inline-in-index.ts OR drop auto_start in backend.ail (informed by Phase 1)
- [ ] M3b: Apply the chosen change in motoko_agent (TS or .ail)
- [ ] M3c: Update v0.18.1 health-check-share logic to be a no-op for per-task isolation
- [ ] M3d: Test: `TestExecute_SingleEnvServer` asserts only one env-server process per spawn (via `lsof -i` count from a side-channel)

**Phase 4: Cache pre-warming (~1.5 hours)**
- [ ] M4a: Add `--warm-cache` flag to motoko adapter HealthCheck (off by default, on when set)
- [ ] M4b: HealthCheck warm-cache runs `ailang check src/core/supervisor.ail` against MOTOKO_REPO (NOT MOTOKO_HOME — this builds the template's hot cache once)
- [ ] M4c: Adapter setupMotokoHome's hardlink mirror INCLUDES the freshly-warmed `.ailang/cache/` dir
- [ ] M4d: Test: `TestHealthCheck_WarmCache_Idempotent` asserts repeated warm-cache calls don't error and produce a populated `.ailang/cache/` in MOTOKO_REPO

**Phase 5: Validation (~2 hours)**
- [ ] M5a: Run 5 consecutive `ailang eval-suite --agent --models motoko-claude-haiku-4-5 --tier smoke --agent-parallel 4` — record pass rate per run
- [ ] M5b: Acceptance gate: ≥95% success rate across 60 runs total (5 × 15 benchmarks) — i.e. ≤3 failures, all benchmark-correctness misses NOT infrastructure failures
- [ ] M5c: Wall-clock comparison: smoke tier × 4-parallel ≤8 minutes (vs 22 min serial baseline)
- [ ] M5d: CHANGELOG + design doc move to implemented/v0_18_2/

### Files to Modify/Create

**New files (AILANG):**
- `internal/executor/motoko/isolation.go` — per-task MOTOKO_HOME setup + cleanup, ~80 LOC

**Modified files (AILANG):**
- `internal/executor/motoko/motoko.go` — call setupMotokoHome in Execute, add --warm-cache to HealthCheck, ~30 LOC delta
- `internal/executor/motoko/execute_test.go` — TestExecute_PerTaskMotokoHome, TestHealthCheck_WarmCache_Idempotent, ~80 LOC

**Modified files (motoko):**
- `src/core/backend.ail` — revert v0.18.1 health-check-share OR drop auto_start (Phase 1 informs), ~20 LOC delta
- `src/tui/src/index.ts` — possibly drop inline startEnvServer call (Phase 1 informs), ~10 LOC delta
- `src/tui/src/env-server-main.ts` — remove the EADDRINUSE retry/yield handler from v0.18.1 follow-up commit `6580adf` (no longer needed once per-task isolation is in place), ~30 LOC delta

**Wrapper script (outside repo, requires user action OR adapter cmd.Dir override):**
- `/Users/mark/go/bin/motoko` — `cd "${MOTOKO_HOME:-$MOTOKO_REPO}"`, ~1 LOC

---

## Examples

### Example 1: Adapter Execute with per-task isolation

**Before** (v0.18.1):
```go
cmd := exec.CommandContext(ctx, e.motokoPath, directive)
if task.Workspace != "" {
    cmd.Dir = task.Workspace
}
env = append(env,
    "MODEL=" + e.getModel(task),
    "MOTOKO_CONFIG=" + e.profile,
    "MOTOKO_SESSION_ID=" + sessionID,
    "ENV_PORT=0",
)
cmd.Env = env
cmd.Run()
```

**After** (this sprint):
```go
motokoHome, cleanup, err := setupMotokoHome(e.motokoRepo)  // hardlink-mirror per-task copy
if err != nil { return nil, err }
defer cleanup()

cmd := exec.CommandContext(ctx, e.motokoPath, directive)
if task.Workspace != "" {
    cmd.Dir = task.Workspace
}
env = append(env,
    "MODEL=" + e.getModel(task),
    "MOTOKO_CONFIG=" + e.profile,
    "MOTOKO_SESSION_ID=" + sessionID,
    "ENV_PORT=0",
    "MOTOKO_HOME=" + motokoHome,  // NEW: per-task source dir
)
cmd.Env = env
cmd.Run()
```

### Example 2: Single env-server (assuming Phase 1 picks "drop auto_start")

**Before** (v0.18.1):
```ailang
-- src/core/backend.ail
} else if cfg.auto_start then {
    if health_ok(cfg.url) then {
        -- share existing env-server (THE BUG: shares across siblings)
        ...
    } else {
        let handle = spawnProcess(cfg.command, append_backend_args(cfg.args, cfg.port, workdir));
        ...
    }
}
```

**After**:
```ailang
} else if cfg.auto_start then {
    -- Per-task isolation (M-MOTOKO-PARALLEL-EXECUTION-ISOLATION):
    -- Inline env-server in bun already serves on cfg.url. AILANG runtime
    -- inherits as a child of bun and uses cfg.url directly. No spawn.
    {
        mode: cfg.mode,
        url: cfg.url,
        process_id: "inline_in_bun",
        process: None
    }
}
```

---

## Success Criteria

- [ ] Phase 1 produces a written conclusion identifying H1/H2/H3 (or new H4) as the dur=0 root cause
- [ ] Per-task MOTOKO_HOME implementation: 5+ parallel motoko sessions never share filesystem state (verified by `lsof | grep MOTOKO_HOME` snapshots showing distinct paths)
- [ ] Single env-server: `lsof -i -P | grep $MOTOKO_PID` shows exactly one listening socket per session
- [ ] HealthCheck `--warm-cache` builds `.ailang/cache/compile/modules/` in <30s, and subsequent per-task spawns do NOT trigger recompilation (verified via cache mtime checks)
- [ ] `ailang eval-suite --agent --models motoko-claude-haiku-4-5 --tier smoke --agent-parallel 4` produces ≥95% success rate over 5 consecutive runs
- [ ] Wall-clock: 15-benchmark smoke at parallel-4 ≤8 minutes
- [ ] All v0.18.1 acceptance gates still pass (no regression)
- [ ] CHANGELOG entry covering the parallel-execution fix
- [ ] Design doc moved to `implemented/v0_18_2/` with actuals filled in

---

## Testing Strategy

**Unit tests:**
- `TestExecute_PerTaskMotokoHome`: two sequential Execute calls get distinct MOTOKO_HOME paths
- `TestSetupMotokoHome_HardlinkFallsBackToCopy`: hardlink failure (e.g. cross-filesystem) falls back to copy without erroring
- `TestSetupMotokoHome_CleanupRemovesDir`: defer cleanup actually removes the dir
- `TestHealthCheck_WarmCache_Idempotent`: 3x consecutive warm-cache calls don't error, cache dir present after each

**Integration tests:**
- `TestExecute_ParallelExecution_NoSharedState`: spawn 4 motoko mock-binaries simultaneously via real Execute, assert each gets distinct MOTOKO_HOME and no inode-sharing
- `TestEvalSuite_SmokeTier_Parallel4_5xRun`: end-to-end parallel test (gated behind `AILANG_MOTOKO_LIVE=1` env, Anthropic key required) — runs 5 iterations of the smoke tier × parallel-4

**Manual testing:**
- 5 consecutive `ailang eval-suite ... --agent-parallel 4` runs, observed pass rate per run logged in this doc as actuals
- A2A-style cross-harness comparison: motoko vs claude-code on smoke tier × parallel-4 × 3 iterations, check motoko's failure-rate is comparable to claude's (i.e. no infrastructure-failure penalty)

---

## Non-Goals

**Not attempted in this feature:**
- Cloud Run parallelism (the Job already supports `parallelism: N` — this sprint validates locally; cloud-side parallel verification is separate sprint if it surfaces issues)
- Per-task isolation for OTHER executors (claude/gemini/codex/opencode/pi) — they already have it implicitly via `cmd.Dir = task.Workspace` + no shared state; no change needed
- Removing the v0.18.1 EADDRINUSE retry/yield handler from `env-server-main.ts` — keep it as defense-in-depth (it's harmless once per-task isolation prevents the conflict in the first place); could be removed in a future cleanup sprint
- Sharing the `.ailang/cache/compile/` across MOTOKO_HOME instances via a network mount or read-only bind-mount (overkill — hardlink-mirror is a one-time per-task cost, ~100ms typically)

---

## Timeline

**Day 1 (~6 hours):**
- Phase 1 (investigate + bisect): 2h
- Phase 2 (per-task MOTOKO_HOME): 3h
- Phase 3 partial (env-server change picked): 1h

**Day 2 (~6 hours):**
- Phase 3 finish (env-server tested): 1h
- Phase 4 (cache pre-warming): 1.5h
- Phase 5 (validation, 5 parallel runs): 2.5h
- CHANGELOG + design doc finalize: 1h

**Total: ~12 hours across 2 days**

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Phase 1 is inconclusive (instrumentation doesn't surface clear root cause) | High (blocks fix design) | Time-box Phase 1 at 4 hours; if no clear signal, default to "fix all 3 hypotheses simultaneously" — slightly wasteful but unblocks Phases 2-4 |
| Hardlink-mirror crosses filesystems (TMPDIR on different volume from MOTOKO_REPO) | Med | `setupMotokoHome` falls back to full `cp -r` on link failure, with a warning log |
| Per-task MOTOKO_HOME breaks the v0.18.1 MOTOKO_REPO discovery / mirrorProfileFromRepo paths | Med | Test fixture: run v0.18.1's smoke (`go run ./cmd/smoke-motoko`) post-fix and verify Success=true + CostUSD>0 unchanged |
| Cache pre-warming is racy in itself (HealthCheck warm-cache called by 2+ AILANG processes) | Low | Document: HealthCheck is called by the eval harness ONCE per `ailang eval-suite` invocation — already serial. If multiple eval-suite invocations run concurrently against the same MOTOKO_REPO, that's outside scope (use separate MOTOKO_REPO clones) |
| User can't edit `/Users/mark/go/bin/motoko` wrapper | Med | Adapter sets `cmd.Dir = motokoHome` directly — wrapper's `cd` becomes a no-op (since pwd is already MOTOKO_HOME). Tested before relying on wrapper edit |

---

## Related Documents

**Implemented (informed this design):**
- [design_docs/implemented/v0_18_1/m-motoko-eval-harness-hardening.md](../implemented/v0_18_1/m-motoko-eval-harness-hardening.md) — direct precursor; this sprint extends serial-mode hardening to parallel-mode
- [design_docs/implemented/v0_9_4/m-serve-api-concurrency.md](../implemented/v0_9_4/m-serve-api-concurrency.md) — same architectural class (per-request isolation for shared mutable state); reuses the playbook
- [design_docs/planned/v0_18_0/m-motoko-executor-adapter.md](../planned/v0_18_0/m-motoko-executor-adapter.md) — the original adapter spec; explicitly mirrored Pi pattern but didn't formalize per-task isolation as a contract

**Planned (check for overlap):**
- [design_docs/planned/v0_19_0/m-motoko-ext-per-task.md](../planned/v0_19_0/m-motoko-ext-per-task.md) — per-task EXTENSION config (orthogonal to per-task SOURCE/CACHE here, but design surfaces should converge on a unified MOTOKO_HOME concept)
- [design_docs/planned/v0_19_0/m-fs-sandbox-diagnostics.md](../planned/v0_19_0/m-fs-sandbox-diagnostics.md) — surfaces silent-false sandbox rejections; tangentially related (same workdir-confusion class of bugs)

## References

- [Design Axioms](/docs/references/axioms) — A6 (Safe Concurrency) is the load-bearing axiom for this sprint
- v0.18.1 sprint commits: AILANG `abee8522`, motoko `d1c2783`, `6580adf` (the EADDRINUSE handler this sprint will eventually retire)
- Captured eval results: `eval_results/v0_18_1_3harness_smoke/` (v1 baseline, 40/45) and `v0_18_1_3harness_smoke_v2/` (v2 with EADDRINUSE handler, 37/45 — the regression evidence)
- Pi adapter as reference design: `internal/executor/pi/pi.go` lines 113-114 (`cmd.Dir = task.Workspace` — the entire isolation it needs)

## Future Work

- Generic `executor.PerTaskIsolation` trait if a SECOND executor needs the same MOTOKO_HOME-style isolation in the future (rule of three: extract on N=3, not N=2)
- HTTP-level env-server pooling (one shared env-server with per-request workdir isolation) — would let motoko run at higher parallelism without N processes; not needed for v0.18.2 but a possible v0.19.0+ optimization
- Cloud Run side: if local parallel works, validate `parallelism: 4` setting on `agent-motoko` Cloud Run Job behaves the same way

---

**Document created**: 2026-05-08
**Last updated**: 2026-05-08
