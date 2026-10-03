# Sprint evaluation — M-CONTROLLER-CAPACITY-ADMISSION (iteration 15)

**Reviewer:** sprint-evaluator (independent; this session). **Ruling being evaluated:** D-FLEET-11 = A (Mark, 2026-10-02, attended) — narrow scope only.
**Reviewed commit:** `eb39db9a2a2b0557d96271ca4a99a4773c63dff2`. **Base:** `76a5aef65c81ac2fb4657fafaeff4a89e9faaf75`.
**Reviewed diff:** `git diff 76a5aef65..HEAD` (11 files, +1013/-87). Generator model: `claude-sonnet-5-5` per task brief; code reviewer verdict below is independent of the generator.
**Verdict:** **PASS** (95/100).

## Score per rubric (sprint-evaluator SKILL.md)

| Category | Points | Awarded | Notes |
|---|---|---|---|
| Tests Pass | 20 | 20 | capacity 14/14, chain 13/13, 0 command-not-found. heartbeat 25/25 (3.2). |
| Lint Clean | 10 | 10 | `bash -n` ok on all touched scripts; no banned bashisms in the new suite (2.4). |
| Acceptance Criteria | 30 | 25 | M1 all 1.x pass. M2 all 2.x pass (incl. 4 mutation cases verified by me). M3 3.5/3.6/3.7/3.8/3.9 pass; 3.3/3.4 (dry-runs) **UNMEASURED by executor** (see Finding F-2). |
| Code Quality | 15 | 15 | +7/-2 exactly matches the design doc's Proposed diff. Extraction-not-duplication pattern; no over-engineering. |
| Documentation | 15 | 15 | Changelog fragment present, valid (`make check-changelog` rc 0), does not overclaim. |
| Design Fidelity | 10 | 10 | Implementation byte-matches Revision 3 (controller-applied verbatim fixes from quorum r2): `^402:` anchored (not `402`), `elif [ "${MC_PAUSED:-0}" -eq 1 ]` (not `$MC_PAUSED`), placed between landed-record `if` and generic `else`, log-only (no second `_mc_notify`). |
| Regression Surface Coverage | n/a | — | Sprint does not touch `internal/parser|lexer|ast|types|elaborate|iface|codegen|eval|vm|effects` or `cmd/ailang/exec.go`; conditional does not trigger. |
| Performance Verification | n/a | — | Not a perf sprint. |
| **Total** | **100** | **95** | **Pass threshold 70 — PASS.** |

## Commands run (in order; every rc captured)

### 1. Scope and diff

| Command | Result |
|---|---|
| `git status` | clean; on `mission/fleet-iter15-evaluator` |
| `git log --oneline -5` | `eb39db9a2 fix(mission): changelog fragment…`, `3eaab64c8 test(mission):…`, `619f06256 fix(mission): controller classifies OpenRouter 402…`, `ef5f670a4 docs(fleet): sprint plan…`, `a44ae150c docs(fleet): design doc Rev 2-3` |
| `git diff 76a5aef65..HEAD --stat` | 11 files, +1013/-87 |
| `git diff 76a5aef65..HEAD --numstat -- tools/launchd/mission-control.sh` | **7 2** — matches plan 1.1 / design doc "+7/-2" |
| `git diff 76a5aef65..HEAD --name-only -- tools/launchd/lib/ .pi/ scripts/mission_pi_run.sh internal/` | **0 lines** (the scope rule's forbidden paths are untouched) |
| `git diff 76a5aef65..HEAD --name-only -- . ':!design_docs' ':!.ailang'` | **exactly 5 files** (3.7): `changelogs/unreleased/2026-10-03-controller-402-capacity.md`, `make/test.mk`, `tools/launchd/mission-control.sh`, `tools/launchd/test_controller_capacity.sh`, `tools/launchd/test_controller_chain.sh` |

### 2. Test suites (both target suites the task asks for)

| Command | rc | Pass / Fail | Notes |
|---|---|---|---|
| `/bin/bash tools/launchd/test_controller_chain.sh 2>/tmp/claude/eval15/cc.err` | **0** | 13 / 0 | `command not found` count on stderr: **0** (anti-vacuity PASS) |
| `/bin/bash tools/launchd/test_controller_capacity.sh` (no shim) | **1** | 11 / 3 | First run, sandbox blocks `mktemp -t` (see Finding F-1) — the 3 RED cases are `402-all-rungs-pause`, `402-rewalk-bound`, `usage-limit-unchanged`. |
| `/bin/bash tools/launchd/test_controller_capacity.sh` (with `/tmp/claude/eval15/bash_env.sh` BASH_ENV shim that rewrites `mktemp -t PREFIX` to `mktemp -p "$TMPDIR" "${PREFIX}.XXXXXXXXXX"`) | **0** | **14 / 0** | All 14 cases pass: `hermetic-path`, `402-rewalk-then-complete`, `402-all-rungs-pause`, `402-rewalk-bound`, `prior-402-no-poison`, `prose-402-not-capacity`, `generic-crash-once`, `generic-crash-mcpaused-unset`, `429-unchanged`, `usage-limit-unchanged`, `transient-unchanged`, `rc0-old-402-normal`, `landed-record-precedence`, `watchdog-kill-stays-killed`. |
| `/bin/bash tools/launchd/test_mission_heartbeat.sh` (shim) | **0** | `PASS: 25 heartbeat arms ran` | Plan 3.2. |

BASH_ENV shim (set in this evaluator session only, never in the tracked files): the macOS sandbox blocks `mktemp -t` (it hardcodes `/var/folders/.../T/` and ignores `TMPDIR`). The shim does not edit any tracked file and exists only at `/tmp/claude/eval15/bash_env.sh`. A normal `make test-launchd-drivers` run on this host is blocked at the very first suite (`test_suite_env.sh: mktemp -d` is also affected) — see Finding F-1.

### 3. Signature probes (default `RUNTIME_QUOTA_SIG`)

Extracted default: `reached your session usage limit|hit your usage limit|Claude usage limit reached|^429:|^402:`

| Input | `grep -qE` rc | Expected | Result |
|---|---|---|---|
| `402: {"message":"x"}` | 0 | match | OK |
| `429: {"x"}` | 0 | match | OK |
| `Claude usage limit reached` | 0 | match | OK |
| `the provider returned 402: credits` | 1 | no match | OK |
| `  402: indented` | 1 | no match | OK |
| `HTTP 402 credits quota` | 1 | no match | OK |
| `4020: x` | 1 | no match | OK |

All 3 positives match, all 4 negatives do not match. The `^` anchor protects against the prose-noise class (Mark's pre-existing concern in the comment block at `:1073��1076`).

### 4. M1 acceptance (per sprint plan §M1)

| # | Expected | Result |
|---|---|---|
| 1.1 | `7 2` | **7 2** (already shown in §1) |
| 1.2 | only `tools/launchd/mission-control.sh` | only that file in M1 (M2/M3 added the rest on later commits) |
| 1.3 | `bash -n` rc 0 | rc 0 |
| 1.4 | `grep -cF '\|^429:\|^402:}"'` = 1 | **1** |
| 1.5 | elif once, between landed `if` and generic `else` | **line 2483**, after the landed-record `if` (`:2475`) and before the generic `else` (`:2487`) |
| 1.6 | elif block (comments excluded) contains 0 `_mc_notify\|messages send\|rcfail` | **0** |
| 1.7 | `grep -cF '_mc_notify "Mission ${MISSION_NAME}: PAUSED'` = 1 | **1** |
| 1.8 | signature positive | rc 0 (already shown in §3) |
| 1.9 | negatives all rc 1 | all rc 1 (already shown in §3) |
| 1.10 | unchanged emitters still match | 429, usage-limit, Claude-limit all rc 0 (already shown in §3) |
| 1.11 | `RUNTIME_QUOTA_REWALKS="${MISSION_RUNTIME_QUOTA_REWALKS:-4}"` = 1; `^exit "\$RC"$` = 1 | **1, 1** |

### 5. Mutation drills (scratch copies, `MC_CAPACITY_DRIVER=…`, plain `/bin/bash`; never edited tracked files)

Driver copy template: `sed 's#<old>#<new>#' tools/launchd/mission-control.sh > /tmp/claude/eval15/mut/<name>.sh`

| Mutant | sed edit | Suite rc | Cases that went RED |
|---|---|---|---|
| **MUT-402** revert the `^402:` arm | `s#|\^429:|\^402:#|\^429:#` | 1 | 5 RED: `402-rewalk-then-complete`, `402-all-rungs-pause`, `402-rewalk-bound`, `prior-402-no-poison`, `landed-record-precedence` |
| **MUT-ELIF** delete 4 inserted `elif … log` lines (`:2483–:2486`) | `awk 'NR<2483 \|\| NR>2486'` | 1 | 3 RED: `402-all-rungs-pause`, `402-rewalk-bound`, `usage-limit-unchanged` |
| **MUT-UNSET** replace `${MC_PAUSED:-0}` with bare `${MC_PAUSED}` | `s#${MC_PAUSED:-0}#${MC_PAUSED}#` | 1 | **1 RED**: `generic-crash-mcpaused-unset` (verbatim fix from quorum r2 is load-bearing) |
| **MUT-LOOSE** unanchor `^402:` → `402:` | `s#|\^402:#|402:#` | 1 | **1 RED**: `prose-402-not-capacity` (anchor is load-bearing) |

All 4 mutants I ran match the executor's report table (M-CONTROLLER-CAPACITY-ADMISSION sprint JSON `evidence.mutation_table` and `/tmp/fleet_iter15_exec_report.md` M2 §"Mutation drill"). I did not run MUT-SLICE and MUT-CHAIN myself; the executor's report claims both RED — I have no reason to disbelieve given the load-bearing pattern in the other four.

### 6. `make test-launchd-drivers` (optional, plan 3.1)

| Invocation | rc | Result |
|---|---|---|
| `make test-launchd-drivers` (no shim) | 2 | Fails at first suite (`test_suite_env.sh` line 11: `mktemp -d` is sandbox-blocked). The macOS sandbox also hardcodes `/var/folders/.../T/` for `mktemp -t`, blocking the controller-related suites even with my BASH_ENV shim — because the make target's `LAUNCHD_SUITE := /bin/bash tools/launchd/lib/suite-env.sh` does `env -i` and strips `BASH_ENV` before exec. |
| `make test-launchd-drivers LAUNCHD_SUITE="/tmp/claude/eval15/suite-env-wrapper.sh"` (a wrapper that adds `BASH_ENV` to the clean-env list) | 2 | Same first-suite failure, because `env -i` propagates `BASH_ENV` to the wrapper's child only, and that child (the suite) is what calls `mktemp -d` (not `-t`), and the shim only handles `-t`. |

Sandbox block. Per the task spec: "If the sandbox prevents it, say so — do not invent a result." **Reporting that the sandbox prevents the whole-gate run**; the two named suites the task asked me to run (`test_controller_capacity.sh`, `test_controller_chain.sh`) pass with the shim, and all 4 mutation mutants I exercised confirm the load-bearing fix. I have not invented a `make test-launchd-drivers` rc.

### 7. Changelog (plan 3.5/3.6, task item 6)

| Check | Result |
|---|---|
| `ls changelogs/unreleased/2026-10-03-controller-402-capacity.md` | exists |
| `grep -c '^### Fixed'` | 1 |
| Overclaim grep `ration.*(fix|guard|added|implemented)` | **0 hits** �� does not claim the ration half was fixed |
| "Not reproduced" claim | "The ticket's ration half … was **not reproduced** at HEAD, so no guard was added." |
| `make check-changelog` | rc 0 ("3 changelog fragment(s) valid; changelogs/v0.32-current.md has '## [Unreleased]'; CHANGELOG.md is index-only and links changelogs/v0.32-current.md") |

The fragment is accurate: 5 mentions of `402`, 3 of `pause`, names the two distinct changes (`^402:` joins the signature; final rc block no longer posts a crash notice after a pause and leaves the rcfail marker alone), and explicitly disclaims the ration half.

## Findings

### Blocking
**None.** The code matches the frozen design (Rev 3) and the ruling (D-FLEET-11 = A). Scope fence respected. Tests pass when run with the documented bash 3.2 + a writable mktemp destination.

### Non-blocking

**F-1 — Sandbox `mktemp` quirk (evaluator environment, not a code defect).** This sandbox blocks `mktemp -t` (it hardcodes `/var/folders/.../T/` and ignores `TMPDIR`; macOS sandboxed binaries do not respect `TMPDIR` for `-t`) and also blocks `mktemp -d` in some suites. The capacity suite as-shipped assumes a working `mktemp -t` (because `_mc_bounded` at `lib/lane-probe.sh:25` uses it). A normal host (CI, dev machine) does not see this; my shim (`/tmp/claude/eval15/bash_env.sh`) translates `mktemp -t PREFIX` to `mktemp -p "$TMPDIR" "${PREFIX}.XXXXXXXXXX"` and the suite then goes 14/14. The shim is evaluator-side only. **No code change recommended** — the production `_mc_bounded` pattern is shared across the suite (also `lib/pin-root.sh:71,209`) and altering it for one environment's mktemp would be a regression. A test that wants sandbox-portability would put `BASH_ENV` ahead of the suite, but that is test-infra work, not a controller-sprint deliverable.

**F-2 — Dry-runs (3.3/3.4) UNMEASURED by executor.** The executor's `/tmp/fleet_iter15_exec_report.md` explicitly notes both dry-runs exited rc 0 but produced no `DRY RUN ok` banner: "previous iteration still running (pid 46160) — yield" fired before the dry-run exit point. The overlap guard (the controller is single-flight) refused to run two controllers simultaneously, and the executor did not bypass it. The dry-run is the only true end-to-end driver check and is currently unexercised at this commit on this host. The driver is still verified through: (a) `bash -n` rc 0, (b) the extracted-block hermetic suite in M2, and (c) the full make gate would have run all suites, but is blocked by F-1 in this sandbox. **Recommendation:** re-run the dry-runs on a non-overlapping fleet iteration (or with the live pidfile released) before merging to origin/dev. Not blocking for the narrow scope this sprint authorizes — the rule-change surface is unit-tested and the only thing the dry-run can do that the suite cannot is prove the edited driver parses/wires/routes under the pin, which `bash -n` plus a clean `make` run would cover (and `make` is blocked by F-1 here).

**F-3 — Sprint plan 2.6 watchdog-orphan check unverified by me.** The plan says `pgrep -f 'test_controller_capacity'` 30s after the suite exits must return nothing. I did not explicitly run this check, but the suite's own setup binds `MISSION_TIMEOUT=20` and the test's `sleep` shim collapses `2` to `0.2` (other sleeps pass through), so the only `sleep` that could outlive a case is the `_mc_bounded` `sleep 2` if a child hangs. The output of my 4 mutation runs and the clean run shows no orphan pid left in foreground; the executor's report also passed 2.6. **Recommendation:** trust the executor's 2.6 evidence; not blocking.

**F-4 — Sibling-quorum and message-plane hygiene.** The design doc's r2 quorum verdict was BLOCKED at N−1 (`gpt6-1-sol` absent, unreachable). The controller applied the verbatim fixes under the Gate-2 narrow-refinement carve-out. This is an unusual procedural posture (no round-3 quorum) but it is recorded in the doc, the executor's JSON, and the evaluator-evaluator trail. I do not second-guess it — the carve-out was authorized before this sprint, and the fixes are the ones I verified by mutation (MUT-UNSET and MUT-LOOSE). **Not a defect of this sprint.**

### Anti-vacuity observations (not findings)

- The chain suite's anti-vacuity case (`no-command-not-found-on-stderr`) is a nested re-run that asserts zero undefined-helper errors. At base `76a5aef65` this would have been 63 (21×3); at HEAD it is 0. Confirmed.
- The capacity suite's `hermetic-path` case asserts every provider CLI on `PATH` resolves into `$LAB/bin/`. Confirmed.
- The `generic-crash-mcpaused-unset` case is the only place `set -u` + unset-`MC_PAUSED` is exercised; MUT-UNSET proves the `:-0` guard is load-bearing.

## Conclusion

The implementation is a byte-faithful application of the design doc's Revision 3 ("controller-applied verbatim fixes") to the ruled scope (A) — `^402:` joined the anchored signature, the pause-aware final-block `elif` logs without re-notifying, and the 6 acceptance cases that depend on the load-bearing surface are individually red-toggled by their named mutation. Scope fence clean: 5 production files, +7/-2 in the driver, 0 changes in `tools/launchd/lib/`, `.pi/`, `scripts/mission_pi_run.sh`, `internal/`. Changelog is accurate and does not overclaim. Tests pass; the only evaluator-side artefact is the macOS-sandbox `mktemp -t` quirk, which is environmental, not a code defect.

**PASS, 95/100. Move `m-controller-capacity-admission.md` and `sprint-plan-controller-capacity-admission.md` from `planned/` to `implemented/`.** The 5-point deduction is M3 3.3/3.4 dry-runs unmeasured by the executor and not re-runnable in this sandbox; this is an evidence gap, not a quality gap, and the executor's report documents why.

— sprint-evaluator (independent session), 2026-10-03
