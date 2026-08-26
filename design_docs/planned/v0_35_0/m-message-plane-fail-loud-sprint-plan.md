# Sprint Plan: M-MESSAGE-PLANE-FAIL-LOUD

**Design doc**: [m-message-plane-fail-loud.md](./m-message-plane-fail-loud.md)
**Created**: 2026-08-26
**Duration**: 4 milestones, ~670 LOC
**Risk**: Medium — M3 touches the shared agent registry that 39 cloud agents read
**Decisions**: all three ratified by Mark 2026-08-26 (D1 fatal exit · D2 explicit `triage_only` · D3 explicit `execution_lane`)

---

## Pre-flight

| Check | Result |
|---|---|
| Design doc decisions frozen | ✅ D1/D2/D3 ratified |
| `triage_only` / `execution_lane` exist? | ✅ Neither exists — both new (grep: 0 hits) |
| dev CI green? | ❌ **RED, inherited** — `cmd/ailang/eval_suite.go` 826 lines > 800. Failing since `2a5d89b66`, i.e. before this sprint's first commit. Not ours; a mission loop has flagged it (`4ce562b5d`). **Consequence: CI green is not an available success signal for this sprint — each milestone must be verified by targeted `go test` on the packages it touches.** |
| Discord visibility for `public-feedback` | ✅ Verified delivering (daemon PID 50131, `--env prod`) |

---

## Milestones

### M1 — Fatal exit + worktree construction validation (D1)

**~180 LOC** (110 impl + 70 test)

The daemon must never be reachable-but-inert. Two changes:

1. `initTaskProcessing` failure → log the reason and **exit non-zero**. launchd's
   `KeepAlive.SuccessfulExit=false` restarts it; `ThrottleInterval: 10` bounds the loop. A
   misconfigured daemon crash-loops visibly instead of idling silently for three months.
2. `NewWorktreeManager` validates a **supplied** `repoDir` (today it only resolves when empty),
   so an unresolvable path fails at construction instead of on every later `CleanupOrphaned`.

**Acceptance criteria**
- Coordinator started with CWD outside a git repo exits non-zero with an actionable message
- `NewWorktreeManager("/does/not/exist", …)` returns an error
- `NewWorktreeManager` on a real repo still succeeds
- Existing daemon tests pass

**Files**: `internal/coordinator/daemon.go`, `internal/coordinator/worktree.go`, tests

---

### M2 — `triage_only` inboxes, and no third state (D2)

**~120 LOC** (60 impl + 60 test)

`public-feedback` has no agent **on purpose** — anonymous input is never handed to something that
acts on it, and Discord is its routing (verified). But today "no agent" is indistinguishable from
"agent forgotten", which is exactly what sent this session chasing a phantom routing gap.

Add `triage_only: bool` to the agent/inbox config and assert at startup that every known inbox is
either agent-served or explicitly `triage_only`. "Neither" becomes unrepresentable.

**Acceptance criteria**
- `triage_only: true` parses and round-trips
- A `triage_only` inbox is never dispatched (existing `resolveInboxAgent` refusal still applies)
- An inbox with neither an agent nor the marker is a startup error naming the inbox
- `public-feedback` marked `triage_only` in config

**Files**: `internal/coordinator/agent_registry.go`, `internal/coordinator/agent_config.go`, tests

---

### M3 — Explicit `execution_lane` (D3)

**~150 LOC** (90 impl + 60 test)

`workspace` currently means both "local worktree base" and "GitHub repo coordinate", so satisfying
one consumer breaks the other. Add `execution_lane: local|cloud` and a separate `repo` coordinate.

**Back-compat is load-bearing**: all 39 cloud agents carry a bare `org/repo` workspace (verified,
V14). Inference must default those to `cloud`, or this silently moves the fleet to a lane that
does not exist on Cloud Run.

**Acceptance criteria**
- `execution_lane` parses; absent → inferred (`org/repo` → cloud, absolute path → local)
- All 39 existing agents infer `cloud`; a regression test pins this
- `eval-rig` is `local` and never yields a cloud dispatch
- Deprecation warning when a bare `org/repo` workspace is used as a repo coordinate

**Files**: `internal/coordinator/agent_registry.go`, `internal/coordinator/daemon_tasks_exec.go`, tests

---

### M4 — Config compare-and-swap (Phase 2)

**~220 LOC** (150 impl + 70 test)

There is no `if-generation-match` anywhere in the repo (verified, V5), and a lost update was
demonstrated live: a correct edit at 14:24:33Z was clobbered at 14:37:10Z by a copy fetched before
it, with no error on either side.

`ailang coordinator config get|set|diff` with a generation precondition, YAML validation, and
refuse-on-mismatch naming the live generation.

**Acceptance criteria**
- `get` records the generation it fetched
- `set` against a stale generation is refused, naming both generations
- `set` validates YAML and agent invariants before writing
- `--force` retained but must name the generation it overwrites

**Files**: `cmd/ailang/coordinator_config.go` (new), tests

---

## Success Metrics

- [ ] M1–M4 acceptance criteria met
- [ ] `go test ./internal/coordinator/... ./cmd/ailang/...` green (CI green unavailable — see pre-flight)
- [ ] No file exceeds 800 lines (do not worsen the inherited red)
- [ ] CHANGELOG updated
- [ ] Design doc moved to `implemented/v0_35_0/` on completion

## Order and dependencies

M1 → M2 → M3 → M4. M1 and M2 are independent of each other in principle but M2's startup assertion
rides on M1's fail-loud path, so M1 first. M4 is standalone and could be parallelised.

## Risks

| Risk | Mitigation |
|---|---|
| M1 turns a misconfig into a restart loop | `ThrottleInterval: 10` already set; message must name the fix |
| M3 reclassifies the fleet | Default-to-cloud inference + a test that pins all 39 |
| Inherited red masks a real break | Per-package `go test` after each milestone |
