# M-MISSION-LIGHT-PROFILE: Verified Lanes Before Arming, and a Mission Profile Sized for Docs Work

**Status**: Planned — draft for Mark's review (attended 2026-10-01). Not yet quorum-reviewed.
**Target**: v0.50.0
**Priority**: P1 (blocks re-arming the docs mission; the fleet's fallback lanes share the defect)
**Estimated**: 3 days (Phase 1: 1d, Phase 2: 1.5d, Phase 3: 0.5d)
**Dependencies**: None. Extends `ailang doctor` (exists) and the shell driver (live). Does not
depend on the stalled binary iteration path — see HD-1.

## Problem Statement

The docs mission was re-armed on 2026-10-01 after a three-week pause. Its first fire (iteration
17) ran **51 minutes and changed nothing on the site**. None of the time went on docs work:

| Phase | Time | What happened |
|---|---|---|
| Worktree | ~5 min | `git worktree add` checked out all 25,798 tracked files; the controller's command timed out and it read a half-built tree (25,786 staged deletions) |
| Gate 3 | parked | Sonnet over ration → evaluator fell to pi → pi sandbox dependency never installed in this clone → no judge → item correctly parked |
| Gate 3b | ~27 min | the park's charter/log commit waited for the full CI run |

Each failure has the same shape: **a path the loop depends on was never exercised until a live
fire needed it.**

1. **Fallback lanes rot silently.** pi lanes in the fleet and docs clones had been dead since
   2026-09-30 (global and repo extensions loaded twice, rc=1; 12 fleet fires) and the sandbox
   runtime was never installed in either clone. Nobody noticed, because the primary lanes worked
   and the harness loop learns only from filed tickets — an unused fallback files none.
2. **Nothing verifies a mission before it is armed.** `MISSION_DRY_RUN=1` probes that each *model*
   answers (`pi --no-tools -p ok`), not that each role's *launcher* works. Per-mission clones
   carry untracked state (`node_modules`, staleness: docs was 1,352 commits behind) that nothing
   provisions or checks.
3. **The process is compiler-sized.** Every mission runs the same six gates, a full-repo worktree
   and a full-CI wait, whether the item is a parser change or deleting a duplicated page. For
   docs, 6% of the tree is relevant (≈1,459 files under `docs/`, `examples/`, `tools/`, `.claude/`
   and the mission's own charter).
4. **Controller/judge coupling is implicit.** The Sonnet judge is reached through Claude's Agent
   tool. A codex controller has no Agent tool, so on a codex-led mission the judge silently becomes
   whatever pi fallback works — which was none.

## Goals

**Primary goal:** a docs-class mission spends its fire on docs work, and never discovers a broken
lane at fire time.

**Success metrics:**
- `ailang doctor mission <name>` exercises every role's real launcher, fallbacks included, in
  under 2 minutes, and refuses to report ready on any dead lane.
- Arming a mission (removing `mission-<name>.disabled`) goes through the doctor; a failed doctor
  leaves it paused with the reason.
- A weekly doctor run per armed mission files a harness ticket for each dead lane — fallbacks get
  exercised whether or not a fire needed them.
- A light-profile fire reaches its first docs edit in ≤ 5 minutes (iteration 17: never).
- A record-only commit costs no CI wait (iteration 17: ~27 min).

## High-Impact Decisions

| ID | Decision | Options | Recommendation | Who | Change cost |
|---|---|---|---|---|---|
| HD-1 | Where the light profile lives | (a) the shell driver + skill, now; (b) fold into the binary `mission iterate --work-item` path | **(a).** The binary path stalled on 2026-09-16 with M4 criterion 2 half-open and its docs canary failed its live trial; waiting on it re-pauses docs indefinitely. The doctor (Phase 1) is runtime-agnostic and serves either. | Mark | Medium — profile knobs are env vars the binary path can read later |
| HD-2 | Record-only commits and CI | (a) no CI wait when the commit touches only the mission's own charter/log/dashboard files; (b) wait only for the docs deploy workflow; (c) unchanged | **(a)** for record-only, **(b)** for docs product commits. A record-only commit cannot break the build, and the next product commit's CI covers the tree anyway. | Mark | Low |
| HD-3 | Judge for a codex-led controller | (a) doctor requires an evaluator lane the controller can launch (pi/codex exec), never the Agent tool; (b) keep a Claude controller for docs | **(a).** It keeps the codex-led routing Mark chose and makes the coupling explicit and checked. | Mark | Low |
| HD-4 | Arming gate strictness | (a) doctor must pass before `.disabled` is removed; (b) advisory only | **(a)**, with `--force` + a recorded reason. | Mark | Low |

### Design Freeze

- [ ] HD-1 decided
- [ ] HD-2 decided
- [ ] HD-3 decided
- [ ] HD-4 decided

## Solution Design

### Overview

Three parts, smallest first. Phase 1 alone would have caught every failure in iteration 17.

**Phase 1 — `ailang doctor mission <name>`.** A new subject under the existing `ailang doctor`
(which already serves `builtins`, `memory`, `managed_agents`). For one mission it checks, and
reports each as a typed row (ok / dead + reason + fix):

| Check | How |
|---|---|
| Clone freshness | workdir's `HEAD` vs `origin/<base>`; fail over a threshold, offer the fast-forward when the tree is clean |
| Untracked dependencies | `tools/pi-extensions/sandbox/node_modules/@anthropic-ai/sandbox-runtime` present and matching the lockfile |
| Each role's launcher, every rung | the SAME command line the fire uses (`mission_pi_run.sh` for pi roles, `codex exec`, `claude -p`), with a one-line directive, under the ration gate; rc and verdict per rung |
| Judge reachability | the evaluator chain contains a rung the configured controller can launch (HD-3) |
| Extension load | pi in the workdir loads each extension once (the 2026-10-01 double-load class) |
| Env drift | installed `~/.config/ailang/mission-<name>.env` equals the reviewable repo copy |

Cost: one tiny request per distinct lane; rungs over ration are reported, not called.

**Phase 2 — the `light` work profile** (`MISSION_WORK_PROFILE=light`, default `full`):

- **Sparse worktree.** `git worktree add --no-checkout` then `git sparse-checkout set` to the
  profile's cone (`docs/`, `examples/`, `tools/`, `.claude/`, the mission's charter files). The
  worktree add waits for completion before Gate 1 reads it.
- **Gate 3b by commit class (HD-2).** Record-only commits: no wait. Docs product commits: wait for
  `Deploy Documentation to GitHub Pages` only.
- **Right-sized gates.** No designer or quorum unless the item says so; planner skipped for
  items touching ≤ 3 files (the controller writes the executor brief, as fleet iteration 4 did).
  The evaluator stays mandatory — independent judgement is the one gate that caught real defects
  in docs iterations 8 and 15 (iteration 16 passed clean).

**Phase 3 — fallback exercise.** A weekly launchd job runs the doctor for each armed mission and
files one harness ticket per dead lane (`ailang mission ticket file`, signature
`lane-dead:<mission>:<role>:<rung>`), so the fleet loop sees rot before a fire needs the lane.

### Existing Machinery — reuse, extend, or build new

| Need | Existing | Decision |
|---|---|---|
| Command surface | `ailang doctor <subject>` | extend (new subject) |
| Role launchers | `scripts/mission_pi_run.sh`, driver probe helpers | reuse — the doctor calls them, never a re-implementation |
| Ration gate | `ailang mission quota --over` | reuse |
| Ticket filing | `ailang mission ticket file` | reuse |
| Registry of missions | workbench `missions/*.toml` (M-MISSION-LOOP-WORKBENCH) | reuse for the mission list |
| Sparse worktree | none (verified V5) | build, in the skill's Gate 1 + `mission-base.sh` |

### Files to Modify/Create

- `cmd/ailang/doctor_mission.go` — new `doctor mission` subject (~250 LOC)
- `internal/mission/lanecheck.go` — per-rung launcher check, typed rows (~200 LOC + tests)
- `tools/launchd/mission-control.sh` — read `MISSION_WORK_PROFILE`; export it to the skill (~30 LOC)
- `.claude/skills/mission-control/resources/gate-1-observe.md` — sparse worktree for `light` (~25 lines)
- `.claude/skills/mission-control/resources/gate-3b-ci-green.md` — commit-class wait rule (~30 lines)
- `tools/launchd/dev.ailang.mission-doctor.plist` + `mission-doctor-weekly.sh` — Phase 3 (~60 LOC)
- `tools/launchd/mission-env/mission-docs.env` — `MISSION_WORK_PROFILE=light`

## Examples

### Example 1: re-arming docs

```
$ ailang doctor mission docs
docs  clone        ok    0 behind origin/dev
docs  deps         ok    sandbox-runtime 0.0.71 (lockfile match)
docs  extensions   ok    14 loaded once
docs  controller   ok    codex:gpt-6.1-sol
docs  evaluator    ok    sonnet (over ration → skipped) → pi:ollama/minimax-m3 (over ration → skipped) → pi:openrouter/minimax/minimax-m3 rc=0
docs  judge        ok    evaluator reachable from a codex controller via pi
READY (6/6)
```

On iteration 17's state the same command would have printed `deps dead: sandbox-runtime missing
(fix: npm ci in tools/pi-extensions/sandbox)`, `extensions dead: 8 duplicate tools`, `clone dead:
1,352 behind`, and refused `READY`.

### Example 2: a light fire on docs-14

Gate 1 checks out ~1,459 files instead of 25,798; the controller briefs the Luna executor directly
(two files); the Sonnet-or-fallback judge reviews; the product commit waits only for the Pages
deploy; the log commit waits for nothing.

## Success Criteria

- [ ] `ailang doctor mission <name>` reports every check above, exits non-zero on any dead row,
      and its launcher checks use the fire's own command lines (asserted by test)
- [ ] Run against a fixture reproducing iteration 17 (missing deps, double-loaded extensions,
      stale clone), it reports all three dead — mutation-tested
- [ ] Arming path refuses on a failed doctor; `--force` records the reason
- [ ] `light` worktree is sparse and fully checked out before Gate 1 reads it
- [ ] Record-only commits skip the CI wait; docs product commits wait for the deploy workflow only
- [ ] Weekly doctor files one ticket per dead lane, de-duplicated by signature
- [ ] A live docs fire reaches its first edit in ≤ 5 minutes
- [ ] `make test-launchd-drivers` and `make test` green; changelog fragment

## Testing Strategy

Unit tests for each check's classification on fixtures (missing dir, mismatched lockfile, duplicate
extension names, stale ref). Launcher checks are tested by stubbing the launchers on `PATH` the
way `test_driver_notify.sh` stubs `gh`/`ailang`, and asserting the argv matches the fire's. One
live attended doctor run per mission before Phase 1 is marked done.

## Deferred Decisions

- Exact staleness threshold for a clone (agent's call; start at 50 commits).
- Whether `light` also trims the skill text loaded per gate (measure token use after Phase 2).

## Non-Goals

- Finishing the binary iteration path (`mission iterate --work-item`). HD-1 keeps this doc
  independent of it.
- Changing model routing; the doctor reads the env, it does not choose models.
- A light profile for code missions — v1, fleet, motoko and world keep `full`.

## Timeline

Phase 1 (doctor) 1 day → re-arm docs on `full` behind it. Phase 2 (light profile) 1.5 days →
switch docs to `light`. Phase 3 (weekly exercise) 0.5 day.

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Doctor diverges from what the fire actually runs (the seam class) | it calls the fire's launchers, never a copy; a test asserts the argv |
| Doctor spends quota | one tiny request per distinct lane, over-ration rungs skipped |
| Sparse cone misses a file an item needs | the executor can `git sparse-checkout add`; the cone lives in the env, not code |
| Skipping CI on record-only commits hides a broken tree | the classifier is a path allowlist (charter/log/dashboard only); anything else waits |

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Harness only |
| A2: Replayability | 0 | No trace change |
| A3: Effect Legibility | 0 | No language effects |
| A4: Explicit Authority | 0 | Arming stays a human action; `--force` is recorded |
| A5: Bounded Verification | +1 | Lane readiness becomes a bounded, pre-fire check instead of a live-fire discovery |
| A6: Safe Concurrency | 0 | — |
| A7: Machines First | +1 | Typed doctor rows a loop can act on (file a ticket, refuse to arm) |
| A8: Minimal Syntax | 0 | — |
| A9: Cost Visibility | +1 | Measured fire time and CI wait become per-profile figures; dead lanes stop silently pushing spend to dearer rungs (docs judge → opus) |
| A10: Composability | +1 | Extends `ailang doctor` and reuses launchers, ration gate and tickets |
| A11: Structured Failure | +1 | A dead lane is a named row with a fix, not a parked iteration |
| A12: System Boundary | 0 | — |

**Net Score: +5** → **Decision: Move forward.** No −1 on A1/A3/A4/A7.

## Quorum

Attended doc. Triggers: #1 fires (four design-freeze items). Quorum to run before sprint
planning, with Mark's answers to HD-1..HD-4 applied first.

## Verification Log

| # | Claim | Evidence |
|---|---|---|
| V1 | Iteration 17 took 3,062 s and landed nothing | `slot-verdict: COMPLETED … elapsed_s=3062` (/tmp/ailang-mission-docs.log 12:29:52); docs-14 parked at Gate 3 |
| V2 | Worktree add ran ~5 min; the controller read it half-built | pid 45305 `git worktree add` etime 04:47 at 11:48:06; `status --porcelain` showed 25,786 `D ` entries at 11:48 |
| V3 | Tracked files 25,798; light cone ≈1,459 | `git ls-files \| wc -l`; `git ls-files docs examples tools .claude 'design_docs/docs-mission*' \| wc -l` (2026-10-01, 3c7fd2183) |
| V4 | Gate 3b CI wait ~27 min on a record commit | heartbeat `gate-3b` 10:02:03Z → `complete` 10:29:46Z |
| V5 | No sparse-checkout use in the skill or driver | `git grep -n -i sparse -- .claude/skills/mission-control tools/launchd` → empty |
| V6 | No record-only exemption in Gate 3b | `grep -i 'record-only\|docs-only\|skip.*wait' gate-3b-ci-green.md` → empty |
| V7 | No clone provisioning anywhere | `git grep 'npm ci\|npm install' -- tools/launchd scripts/mission_*` → only a comment in `mission_pi_run.sh:164` |
| V8 | `ailang doctor` exists with subjects; no `mission` subject | `cmd/ailang/commands_platform.go:33`; `ailang doctor --help` lists builtins, memory, managed_agents |
| V9 | Dry-run probes models, not launchers | `mission-control.sh` probe: `pi --mode json --no-session --no-tools --no-extensions --model "$m" -p 'reply with exactly: ok'` |
| V10 | pi lanes dead in fleet/docs clones since 2026-09-30 | 12 `conflicts with` lines in /tmp/ailang-mission-fleet.log (first 09-30 03:05); fixed `6d2124339` |
| V11 | Sandbox runtime absent in docs and fleet clones | `tools/pi-extensions/sandbox/node_modules` missing in both on 2026-10-01; installed by `npm ci` that day |
| V12 | A codex controller cannot reach the Agent-tool Sonnet judge | iteration 17's own exposure record: `agent-tool:sonnet-unavailable` (/tmp/ailang-mission-docs.log ~23670) |
| V13 | The binary iteration path is stalled | `design_docs/mission-runtime-handover-2026-09-16.md` §1: M4 criterion 2 half-open, one-role-table not done |

## Related Documents

- [m-mission-loop-workbench](v0_36_0/m-mission-loop-workbench.md) — registry, artifacts and
  reach (`mission doctor` there checks *configuration* drift; this doc's doctor checks *runtime
  readiness*. Same command family; the registry supplies the mission list.)
- [m-mission-runtime-contract](m-mission-runtime-contract.md) and
  [mission-runtime-handover-2026-09-16](../mission-runtime-handover-2026-09-16.md) — the binary
  path (HD-1).
- [docs-mission.md](../docs-mission.md) — the first consumer.
- [m-mission-portability](../implemented/v0_30_0/m-mission-portability.md) — per-mission profiles.

## Future Work

A `light` profile for other non-code missions (e.g. a benchmark-report or inbox-triage mission)
once docs proves it.
