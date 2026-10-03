# Fleet iteration 15 — executor report (M-CONTROLLER-CAPACITY-ADMISSION, M1–M3)

Branch `fleet/i15-controller-402`, worktree `/Users/voightkampff/.ailang-driver-pin/fleet-i15`. Nothing pushed.

## M1 — driver (commit 619f06256) — DONE ✓
Applied the design doc's fenced diff with `git apply` (extracted via the plan's awk). `git diff --numstat` = `7 2`.
- 1.3 `bash -n` rc 0 · 1.4 `|^429:|^402:}"` count 1 · 1.5 elif at line 2483 (once, between landed-record `if` and `else`)
- 1.6 elif block, comments excluded: 0 `_mc_notify|messages send|rcfail` · 1.7 pause announcer count 1
- 1.8 `402: {"message":"x"}` → 0 (match). 1.9 negatives all 1: `the provider returned 402: credits`, `  402: indented`, `HTTP 402 credits quota`, `insufficient credits`, `4020: x`
- 1.10 unchanged emitters all 0: `429: {"x"}`, `You hit your usage limit`, `reached your session usage limit`, `Claude usage limit reached`
- 1.11 REWALKS=4 line count 1; `exit "$RC"` count 1

## M2 — suites + make wiring (commit 3eaab64c8) — DONE ✓
- New `tools/launchd/test_controller_capacity.sh`: 14 cases (the 13 in the plan plus `hermetic-path`), all PASS, ~5s wall.
  Extracts the real `_mc_run_once`, retry loop, SLOT VERDICT, SLOT NOTIFY and final block from `MC_CAPACITY_DRIVER`; fake pi/ailang/gh/claude/codex first on PATH; billing reader stubbed.
  `402-all-rungs-pause` runs twice (marker absent→absent and `1`→`1`).
- `test_controller_chain.sh`: now loads `_mc_canon_id/_mc_demote/_mc_is_demoted` (driver) and `_mc_rung_bucket/_mc_is_over_ration` (lane-probe); `_mc_load_ration` stubbed.
  Added `all-blocked-no-controller` (+ zero-probe assert), `openrouter-blocked-reaches-next` (+ no openrouter probe), `demoted-rung-skipped` (+ no probe), anti-vacuity `no-command-not-found-on-stderr` (nested re-run).
  Result: 13 PASS, rc 0, `command not found` count 0. At base 76a5aef65 (git archive, run from scratch dir) the old suite: rc 0 with **63** `command not found`.
- `make/test.mk`: line added directly after `test_controller_chain.sh`.
- 2.4 forbidden-bashism grep: no output (rc 1). 2.6 `pgrep -f test_controller_capacity` after exit: no output.
- Zero inference: 2.5 satisfied by the `hermetic-path` case (all five provider CLIs resolve to `$LAB/bin`).

### Mutation drill (scratch copies in a mktemp dir; no checkout/stash)
| Mutant | rc | Cases that went RED |
|---|---|---|
| MUT-402 revert `\|^402:` | 1 | 402-rewalk-then-complete, 402-all-rungs-pause, 402-rewalk-bound, prior-402-no-poison, landed-record-precedence |
| MUT-ELIF drop the 4 inserted lines | 1 | 402-all-rungs-pause, 402-rewalk-bound, usage-limit-unchanged |
| MUT-UNSET bare `$MC_PAUSED` | 1 | generic-crash-mcpaused-unset |
| MUT-SLICE `cat "$LOG"` on quota line | 1 | prior-402-no-poison |
| MUT-LOOSE `\|402` unanchored | 1 | prose-402-not-capacity |
| MUT-CHAIN :1010 ration check → `if false` (chain suite on scratch copy of tools/launchd) | 1 | all-blocked-no-controller, all-blocked-no-controller-zero-probes, openrouter-blocked-reaches-next, openrouter-blocked-reaches-next-no-probe |
All six RED; unmutated suite rc 0.
Caught on the way: my first drill run was all green because `suite-env.sh` does `env -i` and strips `MC_CAPACITY_DRIVER`; the drill must call the suite with plain `/bin/bash` (the plan's template does). Also fixed in-suite: `VAR=x check` is temporary in bash, so the probe-count asserts use plain assignments.

## M3 — done gate (commit eb39db9a2) — gates ✓, dry-runs UNMEASURED
- 3.1 `make test-launchd-drivers` (unpiped, /bin/bash 3.2.57): **rc=0**, 381s. Output includes `controller-capacity: 14 cases passed, fail=0`. Counts over the whole output: 46 `^PASS `, 415 `^  PASS:`, 167 `^ok `; 0 `not ok`, 0 `^FAIL `, 0 `  FAIL:`. Ends `launchd drivers: tests + bash 3.2 syntax OK`.
- 3.2 `test_mission_heartbeat.sh`: `PASS: 25 heartbeat arms ran`.
- 3.3 / 3.4 dry-runs (`AILANG_DRIVER_PINNED=eb39db9a2a2b0557d96271ca4a99a4773c63dff2 MISSION_PROFILE=fleet MISSION_DRY_RUN=1`, bounded 20 min): both exited rc 0 but **UNMEASURED** — no `DRY RUN ok` banner. Each log ends `previous iteration still running (pid 46160) — yield`; pid 46160 is the live `claude -p … mission-control iteration` that spawned this executor, so the overlap guard fired before the dry-run exit point. Healthy log: pin ok, `fleet: 35 open ticket signature(s)`, `ration gate: blocked buckets: codex ollama`. Degraded log: `codex:bogus` blocked by admission, fell back to `claude:claude-sonnet-5-5`, then the same yield. I did not bypass the guard (it would touch live pidfile state). Run these from outside an active fleet iteration. Logs: /tmp/i15_dry_healthy.txt, /tmp/i15_dry_degraded.txt.
- 3.5 changelog fragment exists, 1 `### Fixed`; says `^402:` joins the signature, a pause no longer posts a crash notice or writes the marker, ration half not reproduced. 3.6 `make check-changelog` rc 0.
- 3.7 file list vs 76a5aef65 (excl. design_docs, .ailang) = exactly the 5 planned files. 3.8 diff of lib/.pi/mission_pi_run.sh/internal = 0 lines.
- 3.9 mutation table above: 6/6 RED.
- Sprint JSON: M1–M3 `passes: true`, status `executed`.

## Pre-existing failures
None: the full driver gate passed.

## Not verified
- Dry-runs (above). The edited driver was only syntax-checked (`bash -n`), exercised via the extracted blocks in M2, and sourced by the full gate; not run end to end by the dry-run path.
- The 402 path against a real OpenRouter response (hermetic fake only).

## git log --oneline 76a5aef65..HEAD
eb39db9a2 fix(mission): changelog fragment for controller 402 capacity; sprint milestones M1-M3 done
3eaab64c8 test(mission): hermetic controller capacity suite + chain suite ration/demote seams
619f06256 fix(mission): controller classifies OpenRouter 402 as capacity; pause is not a crash
ef5f670a4 docs(fleet): sprint plan for controller 402 capacity classification (iteration 15)
a44ae150c docs(fleet): m-controller-capacity-admission Rev 2-3 (D-FLEET-11=A; quorum r1/r2 objections resolved)

## git diff --stat 76a5aef65..HEAD
 .../sprint_M-CONTROLLER-CAPACITY-ADMISSION.json    | 112 +++++++
 .../2026-10-03-controller-402-capacity.md          |  18 ++
 .../fleet-mission-evidence/iteration15/drill_u.sh  |  11 +
 ...er-capacity-admission-2026-10-02T21-41-23Z.json |  64 ++++
 ...er-capacity-admission-2026-10-02T21-49-44Z.json |  79 +++++
 .../planned/m-controller-capacity-admission.md     | 188 ++++++------
 .../sprint-plan-controller-capacity-admission.md   | 236 +++++++++++++++
 make/test.mk                                       |   1 +
 tools/launchd/mission-control.sh                   |   9 +-
 tools/launchd/test_controller_capacity.sh          | 330 +++++++++++++++++++++
 tools/launchd/test_controller_chain.sh             |  52 +++-
 11 files changed, 1013 insertions(+), 87 deletions(-)
