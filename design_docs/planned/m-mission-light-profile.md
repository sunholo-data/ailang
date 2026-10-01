# M-MISSION-LIGHT-PROFILE: Verified Lanes Before Arming, and a Mission Profile Sized for Docs Work

**Status**: Planned — **HD-1..HD-4 RATIFIED on the recommended options (Mark, attended 2026-10-01).**
Quorum: round 1 and round 2 blocked; every objection fixed in-doc; round-2 fixes unreviewed (see
Quorum History).
**Target**: v0.50.0
**Priority**: P1 (blocks re-arming the docs mission; the fleet's fallback lanes share the defect)
**Estimated**: 3 days (Phase 1: 1d, Phase 2: 1.5d, Phase 3: 0.5d)
**Dependencies**: None. Builds beside the shell driver (live) and composes with the existing
`ailang mission doctor` (configuration drift, M-MISSION-LOOP-WORKBENCH). No Go changes. Does not
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
- `tools/launchd/mission-arm.sh <name> --check` exercises every role's real launcher, fallbacks
  included, in under 2 minutes, and refuses to report ready on any dead lane.
- Arming a mission (removing `mission-<name>.disabled`) goes through the doctor; a failed doctor
  leaves it paused with the reason.
- A weekly doctor run per armed mission files a harness ticket for each dead lane — fallbacks get
  exercised whether or not a fire needed them.
- A light-profile fire reaches its first docs edit in ≤ 5 minutes (iteration 17: never).
- A record-only commit costs no CI wait (iteration 17: ~27 min).

## High-Impact Decisions

| ID | Decision | Options | Recommendation | Who | Change cost |
|---|---|---|---|---|---|
| HD-1 | Where the light profile lives | (a) the shell driver + skill, now; (b) fold into the binary `mission iterate --work-item` path | **(a).** The binary path stalled on 2026-09-16 with M4 criterion 2 half-open (V13) and its docs canary failed its live trial (V20); waiting on it re-pauses docs indefinitely. The doctor (Phase 1) is runtime-agnostic and serves either. | Mark | Medium — profile knobs are env vars the binary path can read later |
| HD-2 | Gate 3b wait for record-only commits | (a) Gate 3b does not wait when the commit touches only the mission's own record files (charter, log, dashboard, index, archive); (b) wait only for the docs deploy workflow; (c) unchanged | **(a)** for record-only, **(b)** for docs product commits. Skill-side only — CI itself is unchanged (pushes run the full matrix by design, V15). Safe because no test or workflow reads these files (V16) and the one check that does apply, the ledger validator, runs locally before the push. | Mark | Low |
| HD-3 | Judge for a codex-led controller | (a) doctor requires an evaluator lane the controller can launch (pi/codex exec), never the Agent tool; (b) keep a Claude controller for docs | **(a).** It keeps the codex-led routing Mark chose and makes the coupling explicit and checked. | Mark | Low |
| HD-4 | Arming gate strictness | (a) doctor must pass before `.disabled` is removed; (b) advisory only | **(a)**, with `--force` + a recorded reason. | Mark | Low |

### Design Freeze

- [x] HD-1 decided — recommended option (Mark, attended 2026-10-01: "agree with recommendations")
- [x] HD-2 decided — recommended option (Mark, attended 2026-10-01: "agree with recommendations")
- [x] HD-3 decided — recommended option (Mark, attended 2026-10-01: "agree with recommendations")
- [x] HD-4 decided — recommended option (Mark, attended 2026-10-01: "agree with recommendations")

## Solution Design

### Overview

Three parts, smallest first. Phase 1 closes iteration 17's two fire-killing failures — the dead
judge lane (lane check) and the half-built worktree (completion guard, all profiles). The third
cost, the CI wait, is Phase 2's.

**Phase 1 — `tools/launchd/mission-arm.sh <name>` (check, arm, weekly).** A shell script beside
the driver — not Go: every check here is about the shell harness (its launchers, its clones, its
untracked dependencies), and the frozen core must not know about them (route-to-extension).
It is the ONE arming surface and runs two halves:

1. **Configuration** — `ailang mission doctor <name>`, unchanged (registry vs installed env and
   plist; M-MISSION-LOOP-WORKBENCH). Today it reports "no drift" and nothing else (V14).
2. **Runtime readiness** — `tools/launchd/mission-lane-check.sh <name>`, new, which reports each
   check as a typed row (ok / dead + reason + fix):

| Check | How |
|---|---|
| Clone freshness | workdir's `HEAD` vs `origin/<base>`; fail over a threshold, offer the fast-forward when the tree is clean |
| Untracked dependencies | `tools/pi-extensions/sandbox/node_modules/@anthropic-ai/sandbox-runtime` present and matching the lockfile |
| Each role's launcher, every rung | the SAME launchers the fire uses — `scripts/mission_pi_run.sh` for pi roles; for codex/claude the driver's probe functions, which Phase 1 first EXTRACTS into `tools/launchd/lib/lane-probe.sh` (see below) — with a one-line directive, under the ration gate; rc and verdict per rung |
| Judge reachability | the evaluator chain contains a rung the configured controller can launch (HD-3) |
| Extension load | pi in the workdir loads each extension once (the 2026-10-01 double-load class) |

**Probe extraction (prerequisite).** `_mc_probe` (claude), `_mc_probe_codex` and `_mc_probe_pi`
are defined at the top level of `mission-control.sh` (V18), so sourcing the driver to reach them
would run the driver. Phase 1 moves them, with the helpers they call (`_mc_bounded`, the ration
gate's `_mc_load_ration`/`_mc_is_over_ration`, `PROBE_TIMEOUT`), verbatim into
`tools/launchd/lib/lane-probe.sh` — the pattern `lib/pi-ext-args.sh` already uses — and both the
driver and the lane check source it. A test asserts the driver sources the lib and defines none
of those functions itself, so there is one copy.

**Worktree completion guard (all profiles).** Gate 1's `git worktree add` runs to completion
under a bound (`MISSION_WORKTREE_TIMEOUT`, default 900 s) and Gate 1 then asserts
`git status --porcelain` is empty before anything reads the tree. On timeout or a non-empty
status the iteration parks with `worktree_incomplete` and removes the partial worktree — never
proceeds. This is Phase 1, not Phase 2, so re-arming on `full` does not repeat V2.

`mission-arm.sh <name> --check` runs both and exits non-zero on any dead row; `mission-arm.sh
<name>` additionally removes `mission-<name>.disabled` only when both pass (HD-4; `--force
--reason TEXT` records an override in the disabled-file history). The weekly job (Phase 3) runs
`--check`. Because the lane check calls the fire's own launchers, it cannot drift from what a fire
runs — the seam the risk table names.

Cost: one tiny request per distinct lane; rungs over ration are reported, not called.

**Phase 2 — the `light` work profile** (`MISSION_WORK_PROFILE=light`, default `full`):

- **Sparse worktree.** `git worktree add --no-checkout` then `git sparse-checkout set` to the
  profile's cone (`docs/`, `examples/`, `tools/`, `.claude/`, the mission's charter files), under
  Phase 1's completion guard.
- **Gate 3b by commit class (HD-2).** Record-only commits (only the mission's own charter, log,
  dashboard, index and archive files): no wait, after `scripts/mission_decisions.sh --check`
  passes locally. Docs product commits: wait for `Deploy Documentation to GitHub Pages` only.
  Anything else: unchanged.
- **Right-sized gates.** No designer or quorum unless the item says so; planner skipped for
  items touching ≤ 3 files (the controller writes the executor brief, as the fleet loop did on
  2026-09-27, V19).
  The evaluator stays mandatory — it caught real defects in docs iterations 8 and 15 that the
  controller and executor missed (V17); iteration 16 passed clean (V17).

**Phase 3 — fallback exercise.** A weekly launchd job runs the doctor for each armed mission and
files one harness ticket per dead lane (`ailang mission ticket file`, signature
`lane-dead:<mission>:<role>:<rung>`), so the fleet loop sees rot before a fire needs the lane.

### Existing Machinery — reuse, extend, or build new

| Need | Existing | Decision |
|---|---|---|
| Configuration check | `ailang mission doctor <name>` (registry vs installed artifacts) | reuse unchanged — `mission-arm.sh` calls it |
| Runtime readiness | none — `mission doctor` does not look at lanes, clones or dependencies (V14) | build, in shell beside the driver (not Go: harness-specific) |
| Role launchers | `scripts/mission_pi_run.sh`, driver probe helpers | reuse — the doctor calls them, never a re-implementation |
| Ration gate | `ailang mission quota --over` | reuse |
| Ticket filing | `ailang mission ticket file` | reuse |
| Registry of missions | workbench `missions/*.toml` (M-MISSION-LOOP-WORKBENCH) | reuse for the mission list |
| Sparse worktree | none (verified V5) | build, in the skill's Gate 1 + `mission-base.sh` |

### Files to Modify/Create

- `tools/launchd/lib/lane-probe.sh` — the three probes + helpers, moved verbatim out of the driver (~120 LOC moved, not new)
- `.claude/skills/mission-control/resources/gate-1-observe.md` — bounded worktree completion guard, all profiles (~20 lines)
- `tools/launchd/mission-arm.sh` — the arming surface: config doctor + lane check, arm/--check/--force (~80 LOC)
- `tools/launchd/mission-lane-check.sh` — runtime-readiness rows, sourcing the driver's probe functions (~180 LOC)
- `tools/launchd/test_mission_lane_check.sh` — fixture tests, wired into `make/test.mk` (~150 LOC)
- `tools/launchd/mission-control.sh` — read `MISSION_WORK_PROFILE`; export it to the skill (~30 LOC)
- `.claude/skills/mission-control/resources/gate-1-observe.md` — sparse worktree for `light` (~25 lines)
- `.claude/skills/mission-control/resources/gate-3b-ci-green.md` — commit-class wait rule (~30 lines)
- `tools/launchd/dev.ailang.mission-doctor.plist` + `mission-doctor-weekly.sh` — Phase 3 (~60 LOC)
- `tools/launchd/mission-env/mission-docs.env` — `MISSION_WORK_PROFILE=light`

## Examples

### Example 1: re-arming docs

```
$ tools/launchd/mission-arm.sh docs --check
docs  config       ok    ailang mission doctor: no drift
docs  clone        ok    0 behind origin/dev
docs  deps         ok    sandbox-runtime 0.0.71 (lockfile match)
docs  extensions   ok    14 loaded once
docs  controller   ok    codex:gpt-6.1-sol
docs  evaluator    ok    sonnet (over ration → skipped) → pi:ollama/minimax-m3 (over ration → skipped) → pi:openrouter/minimax/minimax-m3 rc=0
docs  judge        ok    evaluator reachable from a codex controller via pi
READY (7/7)
```

On iteration 17's state the same command would have printed `deps dead: sandbox-runtime missing
(fix: npm ci in tools/pi-extensions/sandbox)`, `extensions dead: 8 duplicate tools`, `clone dead:
1,352 behind`, and refused `READY`.

### Example 2: a light fire on docs-14

Gate 1 checks out ~1,459 files instead of 25,798; the controller briefs the Luna executor directly
(two files); the Sonnet-or-fallback judge reviews; the product commit waits only for the Pages
deploy; the log commit waits for nothing.

## Success Criteria

- [ ] `mission-arm.sh <name> --check` reports every check above, exits non-zero on any dead row,
      and its launcher checks call the fire's own launchers (asserted by test)
- [ ] Run against a fixture reproducing iteration 17 (missing deps, double-loaded extensions,
      stale clone), it reports all three dead — mutation-tested
- [ ] Arming path refuses on a failed doctor; `--force` records the reason
- [ ] The driver sources `lib/lane-probe.sh` and defines no probe itself (test)
- [ ] Gate 1 never reads a worktree whose add timed out or whose status is non-empty; it parks
      `worktree_incomplete` and removes the partial tree (all profiles)
- [ ] `light` worktree is sparse
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
| A5: Bounded Verification | +1 | Lane readiness becomes a bounded pre-fire check; the worktree wait gets an explicit bound (900 s) where today it has none |
| A6: Safe Concurrency | 0 | — |
| A7: Machines First | +1 | Typed doctor rows a loop can act on (file a ticket, refuse to arm) |
| A8: Minimal Syntax | 0 | — |
| A9: Cost Visibility | +1 | Measured fire time and CI wait become per-profile figures; dead lanes stop silently pushing spend to dearer rungs (docs judge → opus) |
| A10: Composability | +1 | Composes the existing `mission doctor` with launchers, ration gate and tickets; no core change |
| A11: Structured Failure | +1 | A dead lane is a named row with a fix, not a parked iteration |
| A12: System Boundary | 0 | — |

**Net Score: +5** → **Decision: Move forward.** No −1 on A1/A3/A4/A7.

## Quorum History

Attended doc; trigger #1 (design-freeze items). **Round 1 (2026-10-01T10:34Z): blocked 3/3,
`gpt6-1-sol` absent (OpenAI API org out of credit).** Every objection accepted:
- `oc-kimi-k3`: a second doctor surface beside the workbench's `mission doctor`, relationship
  unverified → one arming surface (`mission-arm.sh`) that calls the existing config doctor
  unchanged plus a new lane check; V8, V14.
- `gemini-3-1-pro`: shell-harness specifics in Go (frozen core, route-to-extension) → Go files
  removed; the lane check is shell beside the driver, sourcing its launchers.
- `oc-glm-5-3`: HD-2's "cannot break the build" and the evaluator claim unverified → HD-2
  re-premised (CI unchanged; skill-side wait only; nothing reads the files; local ledger check);
  V15, V16, V17.

**Round 2 (2026-10-01): blocked 3/3, `gpt6-1-sol` absent.** All three objections were narrow and
verification-class; none disputed the design. Fixed in-doc, and — per the re-quorum-once rule —
**NOT re-submitted; the round-2 fixes are unreviewed, which is the stated gap:**
- `oc-kimi-k3`: "sources the driver's probe functions" unverified, and sourcing the driver runs it
  → probes extracted into `lib/lane-probe.sh` first, with a one-copy test; V18.
- `gemini-3-1-pro`: three historical claims lacked log rows → V19, V20, and V17 for iteration 16.
- `oc-glm-5-3`: the overclaim ("Phase 1 catches every failure") and the worktree guard living in
  Phase 2 only, unbounded → guard moved to Phase 1 for all profiles with a 900 s bound; Overview
  corrected.

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
| V8 | `ailang mission doctor` exists (config drift) | `cmd/ailang/mission_cmd.go:48`, help text: "does what is installed match what was reviewed?" |
| V9 | Dry-run probes models, not launchers | `mission-control.sh` probe: `pi --mode json --no-session --no-tools --no-extensions --model "$m" -p 'reply with exactly: ok'` |
| V10 | pi lanes dead in fleet/docs clones since 2026-09-30 | 12 `conflicts with` lines in /tmp/ailang-mission-fleet.log (first 09-30 03:05); fixed `6d2124339` |
| V11 | Sandbox runtime absent in docs and fleet clones | `tools/pi-extensions/sandbox/node_modules` missing in both on 2026-10-01; installed by `npm ci` that day |
| V12 | A codex controller cannot reach the Agent-tool Sonnet judge | iteration 17's own exposure record: `agent-tool:sonnet-unavailable` (/tmp/ailang-mission-docs.log ~23670) |
| V14 | `mission doctor` checks configuration only | `ailang mission doctor docs` → "1 mission(s): no drift" on 2026-10-01, the same day the docs lanes were dead; it has no lane, clone or dependency check |
| V15 | Pushes to dev always run the full CI matrix | `.github/workflows/ci.yml` `changes` job: non-`pull_request` events emit `code=true` "full matrix"; the docs-only lane applies to PRs only. Record commit `06d8f480a` (7 `design_docs/*.md` files) ran `test` 10:01:45–10:23:01 |
| V16 | No test or workflow reads the docs mission's record files | `git grep -ln docs-mission -- tools/ scripts/ make/ .github/ internal/ cmd/` → only the plist, env, driver comment, inbox router and a `testdata/` fixture; the ledger check in `test_mission_routing.sh:319` reads `v1-mission.md` only |
| V17 | The evaluator caught real defects in docs iterations 8 and 15 | archived STATUS headlines: iter 8 "an independent evaluator caught one real defect neither the controller nor the executor saw"; iter 15 "independent evaluator caught one blocking + two non-blocking defects"; iter 16 "independent evaluator PASS 98/100 zero blocking" |
| V18 | The probes are top-level functions in the driver | `mission-control.sh:904` `_mc_probe()`, `:974` `_mc_probe_codex()`, `:989` `_mc_probe_pi()`; the file runs on load (`set -uo pipefail` at `:38`, no main guard) |
| V19 | The fleet loop skipped the planner for a small item | /tmp/ailang-mission-fleet.log, 2026-09-27 fire: "For P0 #3 I skipped the planner and wrote the executor's instructions directly, since the fix was about 25 lines" |
| V20 | The binary path's docs canary failed its live trial | `design_docs/verification/mission-iteration-reliability/live-trial.md:28`: "execution_failed, token guard (finish reason thrash_aborted)"; the 2026-09-08 pause note cites it |
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
