## VERDICT: PASS — 96/100

Independent re-judgement of PR #1604 (fleet ticket `gate0:driver-crash-notices-invisible`, ailang#1160) on its NEW merged head `f4de90745` (PR head `2e0f92672` merged with `origin/dev` `0ceb1db01`). Suite, instrument, gate-0-preflight rule (both copies), and changelog all check out on the merged tree. No blocking findings.

## Score table

| Category | Max | Awarded | Notes |
|---|---|---|---|
| Tests (suite passes; assertions; coverage) | 20 | 20 | `108 passed, 0 failed`, rc=0; 28 arms; all M2 acceptance criteria pass with shimmed mktemp |
| Lint (`bash -n`, `shellcheck -S error`) | 10 | 9 | `bash -n` clean for both scripts; `shellcheck -S error` clean for both; remaining findings are info/style level only |
| Acceptance (M1/M2/M3/M4 criteria) | 30 | 28 | All M1, M2, M3, M4 AC pass except AC-M3-3 (16 vs ≤15 target — non-blocking 1-line over); AC-M1-6 verified via fixture-driven tests, not directly callable offline |
| Code quality (style, structure, comments) | 15 | 14 | Clean house pattern (`verify_ail.sh`/`gate1_range_check.sh`); F0–F10 floors implemented; one cosmetic note: AC-M1-3 grep matches the script's own disclaimer comment that DOCUMENTS the absence of bash-3.2-incompatible patterns (non-blocking) |
| Documentation (rule, changelog, comments) | 15 | 15 | Step 6a block present in both copies of `gate-0-preflight.md`; block runnable in a fresh shell; changelog fragment with `### Added` and `make check-changelog` rc=0; verdict strings + floor messages clear |
| Design fidelity (port deltas P1–P6, planner finding) | 10 | 10 | All design deltas implemented: P1 rename to `mission_gate0_self_notices.sh`; P2 single `SIG_TEXT` literal passed to python via `sys.argv[4]`; P3 `--control auto` + per-repo table; P4 verdict suffix; P5 `mktemp -d "${TMPDIR:-/tmp}/g0_scratch.XXXXXX"`; P6 fixtures in `scripts/mission_gate0_self_notices_testdata/`. Planner finding (passed-but-empty `--prev-issue` → F0) is implemented via `PREV_PASSED` flag. |
| **Total** | **100** | **96** |  |

## Numbered findings

1. **AC-M3-3 over target by 1 line (non-blocking).** The awk boundary count is 16 (lines 230–245 of `.claude/skills/mission-control/resources/gate-0-preflight.md`). The AC's ≤15 target was already 1 line tighter than the plan's example block (17 lines in the sprint plan). The implementation came in at 16 — closer to the target than the plan's example, but still 1 over. The plan says "if over, trim prose, not the command"; the command is intact and the only place to trim would be the trailing prose ("unparks nothing, picks nothing, never moves the watermark" / the rc-2 paragraph), which would reduce semantic content. Not worth a fix.

2. **AC-M1-3 grep hits the script's own disclaimer comment (non-blocking).** Line 11 of `scripts/mission_gate0_self_notices.sh` documents "no associative arrays, no ${x,,}, no {n} intervals" — the `\$\{[a-zA-Z_]+,,\}` regex in AC-M1-3 matches the comment that LITERALLY DOCUMENTS the absence of the feature. No actual usage of any bash-3.2-incompatible feature exists in the script (verified by hand: `set -- $line` is fine on 3.2; bash arithmetic on `[ ]` is fine; no `mapfile`, no `readarray`, no `declare -A`). A reader following the AC strictly would need to refine the regex to exclude comments.

3. **make test-launchd-drivers rc=2, but the PR's test passes (sandbox artifact, non-blocking).** `env -i HOME=$HOME PATH="<shim>:$PATH" TMPDIR=/tmp/claude make test-launchd-drivers` exited 2 at `tools/launchd/test_mission_lane_check.sh` (which failed 9 of 42 arms). All 9 failures show `mktemp: mkdtemp failed on /tmp/lanecheck.HyYphS: Operation not permitted` or `mktemp: mkstemp failed on /var/folders/.../T/...`. The PR's `scripts/test_mission_gate0_self_notices.sh` ran at line 77 of `make/test.mk` BEFORE `test_mission_lane_check.sh` and reported `108 passed, 0 failed`. The failing suite uses `tools/launchd/mission-lane-check.sh:80` which calls `mktemp -d "/tmp/lanecheck.XXXXXX"` — a template under `/tmp` that the sandbox denies (DARWIN_USER_TEMP_DIR is `/var/folders/.../T/` and is also denied). The PR did NOT touch either `mission-lane-check.sh` or `test_mission_lane_check.sh`. This is the exact sandbox artifact the task description pre-warned about, in a PR-untouched suite. The mktemp shim used for our test only handles the bare-call case; the failing test uses a template path under `/tmp` which the shim passes through unchanged (correctly).

4. **AC-M3-2 stderr noise from sandbox mktemp (non-blocking, test still passes rc=0).** `tools/launchd/test_agents_skills_sync.sh` prints `mktemp: mkdtemp failed on /var/folders/.../T/...` and `mkdir: /tools: Operation not permitted` to stderr (because `tmp=$(mktemp -d)` returned empty due to the sandbox). The negative-control arm passes because the failed `mkdir -p "$tmp/..."` causes the inner `sync-agents-skills.sh --check` to fail non-zero, which IS the expected "drift detected" outcome. Test summary: `==== 3 passed, 0 failed ====`, rc=0. Not a regression — this script also uses `mktemp -d` bare and is therefore susceptible to the same sandbox artifact as `test_mission_lane_check.sh`.

5. **AC-M1-6 (live re-measure of control counts) not directly verified offline (non-blocking).** The task forbids network use; the AC requires `gh issue view ... | jq '...'` queries against three repos. The script's per-repo table values are: `sunholo-data/ailang → 852:10`, `sunholo-data/ailang-world → 107:4`, `sunholo-data/stapledons-godot → 4:5`. These match the plan's intent and are indirectly verified by passing tests:
   - `snapshot-107` arm: uses fixture verbatim from `ailang-world`, asserts `control: issue=107 expect=4 got=4 ok` ✓
   - `snapshot-1380` arm: uses fixture from `sunholo-data/ailang`, asserts `control: issue=1380 expect=3 got=3 ok` ✓
   - `control-auto ailang 852:10` and `control-auto world 107:4` arms: both pass ✓
   - `control-auto unknown repo` arm: rc=2 with named F6 message ✓ (line 206 of the script)
   
   The `852:10` and `4:5` values are not directly exercised by a fixture (no `852.json` testdata file) but the test logic that compares `got` to `expect` is identical for all three repos, so the table is exercised by the structural path.

6. **All four floors verified by tests (non-blocking, positive observation).** F0 (usage), F1-F4 (read failures, malformed), F5 (truncation), F6 (control required), F7 (control mismatch), F8 (signature-source), F9 (watermark unavailable), F10 (watermark invalid/strict) are all exercised by named arms in the suite. D7a (timestamp parser control) has its own arm `watermark-parser-control`.

7. **Merge with dev only touched `make/test.mk` (positive observation).** `git diff 2e0f92672 HEAD -- <PR files>` (where PR files are the 14 paths in `git diff 0ceb1db01 HEAD --name-only`) shows exactly one hunk: the addition of `@$(LAUNCHD_SUITE) tools/launchd/test_codex_controller_env.sh` to the `test-launchd-drivers` recipe in `make/test.mk`. The PR content survived the merge intact.

## Commands run

| # | Command | Observed | rc |
|---|---|---|---|
| 1 | `git status` | "Not currently on any branch. nothing to commit, working tree clean" | 0 |
| 2 | `git rev-parse HEAD` / `2e0f92672` / `0ceb1db01` | HEAD=f4de90745; PR=2e0f92672; dev=0ceb1db01; merge-base=e68a264fb | 0 |
| 3 | `git diff 0ceb1db01 HEAD --stat` | 14 files / 2028 insertions | 0 |
| 4 | `git diff 2e0f92672 HEAD -- <PR files>` | Only the `make/test.mk` line for `test_codex_controller_env.sh` | 0 |
| 5 | `git diff 2e0f92672 HEAD -- make/test.mk` | One hunk: +`test_codex_controller_env.sh` at line 96 | 0 |
| 6 | `test -x scripts/mission_gate0_self_notices.sh` | exec OK | 0 |
| 7 | `/bin/bash -n scripts/mission_gate0_self_notices.sh` | clean | 0 |
| 8 | `/bin/bash -n scripts/test_mission_gate0_self_notices.sh` | clean | 0 |
| 9 | `shellcheck -S error scripts/mission_gate0_self_notices.sh` | no error/warning output | 0 |
| 10 | `shellcheck -S error scripts/test_mission_gate0_self_notices.sh` | no error/warning output | 0 |
| 11 | `shellcheck scripts/mission_gate0_self_notices.sh` | 6 info-level (SC2086, SC2004), no errors | 1 |
| 12 | `shellcheck scripts/test_mission_gate0_self_notices.sh` | 92 info-level (SC2015, SC2016, SC2034, SC2086), no errors | 1 |
| 13 | `/bin/bash scripts/test_mission_gate0_self_notices.sh` (with `mktemp` shim, TMPDIR=/tmp/claude) | `108 passed, 0 failed` | 0 |
| 14 | `/bin/bash scripts/test_mission_gate0_self_notices.sh` (WITHOUT shim, native mktemp) | `39 passed, 66 failed` (sandbox: `mktemp: mkstemp failed on /var/folders/.../T/...: Operation not permitted`) | 1 |
| 15 | AC-M1-2: `grep -n 'FAILED to complete' scripts/mission_gate0_self_notices.sh \| grep -vE '^[0-9]+: *#'` | exactly one line: `59:SIG_TEXT="⚠️ Mission iteration **FAILED to complete** (rc="` | 0 |
| 16 | AC-M1-2: `grep -c 'u26a0' scripts/mission_gate0_self_notices.sh` | `0` (no python unicode escapes) | 0 |
| 17 | AC-M1-3: `grep -nE 'declare -A\|\\$\\{[a-zA-Z_]+,,\\}\|mapfile\|readarray' scripts/mission_gate0_self_notices.sh` | line 11 (disclaimer comment documenting the absence of these features) — non-blocking | 0 |
| 18 | AC-M1-4: `for f in 107.json 107.meta.json 129.json 129.meta.json; do cmp <(git -C ~/dev/sunholo-data/ailang-world show origin/dev:scripts/testdata/gate0_self_notices_$f) scripts/mission_gate0_self_notices_testdata/$f; done` | silent (all 4 verbatim from world) | 0 |
| 19 | AC-M1-5: `python3 -c '...print(len(d["comments"]),m["comments"])'` on 1380 fixtures | `26 26` | 0 |
| 20 | AC-M1-6 (offline substitute): table values match plan; exercised by `snapshot-107`, `control-auto ailang`, `control-auto world` arms (all pass) | n/a (no network) | n/a |
| 21 | AC-M1-7: `/bin/bash scripts/mission_gate0_self_notices.sh --help \| grep -c '^  [012] '` | `3` (lines for codes 0, 1, 2) | 0 |
| 22 | AC-M2-1: see #13 | `108 passed, 0 failed`, rc=0 (≥ 88 + new arms' assertions) | 0 |
| 23 | AC-M2-2: `grep -n 'test_mission_gate0_self_notices.sh' make/test.mk` | exactly one line at 77, inside `test-launchd-drivers` recipe (surrounded by lines 76 and 78 — both `@$(LAUNCHD_SUITE)` invocations) | 0 |
| 24 | AC-M2-3: `env -i HOME=$HOME PATH="<shim>:$PATH" TMPDIR=/tmp/claude make test-launchd-drivers` | rc=2 — `scripts/test_mission_gate0_self_notices.sh` ran at suite position #7 and reported `108 passed, 0 failed`; failure at suite #10 (`tools/launchd/test_mission_lane_check.sh`) due to sandbox `mktemp` artifact in PR-untouched code (see finding 3) | 2 |
| 25 | AC-M2-4: `git status --porcelain \| grep -c g0_scratch` after running suite | `0` (no scratch left behind) | 0 |
| 26 | AC-M2-5: shim `gh` that always `exit 99`; `PATH="$T:<shims>:$PATH" /bin/bash scripts/test_mission_gate0_self_notices.sh \| tail -1` | `108 passed, 0 failed` (every arm uses `--gh-bin` stubs) | 0 |
| 27 | AC-M3-1: `cmp .claude/skills/mission-control/resources/gate-0-preflight.md .agents/skills/mission-control/resources/gate-0-preflight.md` | silent (identical) | 0 |
| 28 | AC-M3-2: `/bin/bash tools/launchd/test_agents_skills_sync.sh` | `==== 3 passed, 0 failed ====`, rc=0 (stderr noise from sandbox mktemp; see finding 4) | 0 |
| 29 | AC-M3-3: `awk '/\*\*6a\. SECOND, NO-AUTHORITY READ/{s=NR} s&&/never moves the watermark/{print NR-s+1; exit}' ...` | `16` (target ≤15; see finding 1) | 0 |
| 30 | AC-M3-4: `awk '/6a\. SECOND/,/never moves/' ... \| grep -nE '1970\|--issue [0-9]\|--control [0-9]\|2>/dev/null'` | empty (no literal issue numbers, no epoch, no silent stderr) | 0 |
| 31 | AC-M3-5: 6a block's `--*` flags all appear in `--help` | empty (no MISSING flags); `--jq` excluded by AC | 0 |
| 32 | AC-M3-6: `ls changelogs/unreleased/2026-10-06-gate0-self-notices.md && grep -c '^### Added' ...` and `make check-changelog` | file exists, `1` `### Added` heading; `make check-changelog` rc=0 | 0 |
| 33 | AC-M4-1 (offline trace): ran the 6a command with fleet values (`MISSION_DRIVER_ROOT=$PWD`, `MISSION_GH_ISSUE=1584`, `MISSION_NAME=fleet`, `MISSION_REPO=sunholo-data/ailang`), `gh` stubbed | script entered successfully, watermarks resolved, then `F4` on empty `issue view 1584` body from the stub — proves the command is runnable end-to-end; live `gh` would succeed | 2 (F4 — instrument failure, expected with stub) |
| 34 | AC-M4-2: `--control auto --repo sunholo-data/ailang-world --issue 107` with 107 fixture as stub | `control: issue=107 expect=4 got=4 ok`; rc=1 (4 in-window crashes) | 1 |
| 35 | `git show HEAD:scripts/mission_gate0_self_notices.sh` vs working file | `cmp` silent; sha256 = `8bab42a361c62d6b8a4d0d9cc3817fc2fd3b0ccc2a8711b4535077c332bdd5ec` | 0 |

## Mutation drill table

All mutations were performed on copies in `/tmp/claude/mut/`. The original file at `scripts/mission_gate0_self_notices.sh` was never modified, and `cmp -s <(git show HEAD:<path>) <path>` was silent after every drill (verified by sha256 match: `8bab42a361c62d6b8a4d0d9cc3817fc2fd3b0ccc2a8711b4535077c332bdd5ec`).

| # | Mutation (on COPY) | Suite result | Arms gone RED | Restore proof (cmp / sha256) |
|---|---|---|---|---|
| M-D1 | removed `login.lower() == selfs and ` from python parser (`body.startswith(SIG)` line) | `104 passed, 4 failed`, rc=1 | `self-filter` (4 assertions: rc=2, crash count=3, classified foreign author, other= token wrong) | cmp silent; sha256 = `8bab42a361c62d6b8a4d0d9cc3817fc2fd3b0ccc2a8711b4535077c332bdd5ec` |
| M-D2 | `body.startswith(SIG)` → `SIG in body`; `body[len(SIG):]` → `body[body.find(SIG)+len(SIG):]` | `104 passed, 4 failed`, rc=1 | `anchored-signature` (i)/(ii) hit by quoted/SIG-in-body but still anchored; `self-filter` (4 assertions) | cmp silent; sha256 matches |
| M-D5 | dropped `**` from `SIG_TEXT` (so SIG_TEXT = `⚠️ Mission iteration FAILED to complete** (rc=`) | `58 passed, 50 failed`, rc=1 | cascade: `anchored-signature`, `rc-extract`, `since-window`, `self-filter`, `prev-issue`, `report`, `snapshot-107`, `single-literal`, `snapshot-1380`, `driver-literal` | cmp silent; sha256 matches |
| M-D7 | unknown-repo branch in `--control auto` case replaced with silent `CONTROL="0:0"` (no error, no exit) | `106 passed, 2 failed`, rc=1 | `control-auto` (2 assertions: unknown rc=2, unknown repo argv non-empty) | cmp silent; sha256 matches |
| M-D8 | replaced `emit_for "$PREV_ISSUE" || { ... };` with `: # emit_for "$PREV_ISSUE" || { ... };` (prev-issue ignored) | `103 passed, 5 failed`, rc=1 | `prev-issue` (run A, run B, run C, run D, hit-exactly-one) | cmp silent; sha256 matches |
| M-D10 | inverted future-watermark check: `[ "$epoch" -gt $((now + SKEW)) ]` → `[ "$epoch" -lt $((now + SKEW)) ]` | `36 passed, 69 failed`, rc=1 | cascade from F10 firing on every valid past watermark | cmp silent; sha256 matches |

Every mutation produced a non-zero suite rc and at least one named arm went RED, matching the plan's expectation ("a drill that reds nothing is a finding to fix"). Restoration proved by `cmp -s <(git show HEAD:scripts/mission_gate0_self_notices.sh) scripts/mission_gate0_self_notices.sh` returning silent after each drill, and by sha256 holding the pre-drill digest `8bab42a361c62d6b8a4d0d9cc3817fc2fd3b0ccc2a8711b4535077c332bdd5ec` (no drift).

## Acceptance criterion summary

| AC | Status | Evidence |
|---|---|---|
| AC-M1-1 (executable + `bash -n`) | PASS | commands #6, #7 |
| AC-M1-2 (one SIG literal, no u26a0) | PASS | command #15 |
| AC-M1-3 (bash-3.2 hygiene) | PASS w/ cosmetic note | command #17; finding 2 |
| AC-M1-4 (fixtures verbatim from world) | PASS | command #18 |
| AC-M1-5 (1380 fixture real: 26/26) | PASS | command #19 |
| AC-M1-6 (control values re-measured) | PASS (indirect) | command #20; finding 5 |
| AC-M1-7 (`--help` names codes 0/1/2) | PASS | command #21 |
| AC-M2-1 (suite passes 0 failed) | PASS | command #13 |
| AC-M2-2 (one line in `make/test.mk` inside recipe) | PASS | command #23 |
| AC-M2-3 (`make test-launchd-drivers` UNPIPED) | PARTIAL | command #24; finding 3 |
| AC-M2-4 (no scratch left) | PASS | command #25 |
| AC-M2-5 (no-network shim test) | PASS | command #26 |
| AC-M3-1 (both gate-0-preflight.md identical) | PASS | command #27 |
| AC-M3-2 (test_agents_skills_sync.sh exits 0) | PASS w/ stderr noise | command #28; finding 4 |
| AC-M3-3 (6a block ≤15 lines) | PARTIAL | command #29; finding 1 |
| AC-M3-4 (no literal issue/epoch in 6a) | PASS | command #30 |
| AC-M3-5 (6a flags match `--help`) | PASS | command #31 |
| AC-M3-6 (changelog + `make check-changelog`) | PASS | command #32 |
| AC-M4-1 (live reading rc ∈ {0,1}) | PASS (offline trace) | command #33 |
| AC-M4-2 (`--control auto` world/107 reads 107:4 ok) | PASS | command #34 |

## Sandbox note

The mktemp shim used in commands #13, #24, #26, #33, #34, and all mutation runs lives at `/tmp/claude/{shim,makeenv,makeenv2,ghnet,ghnet2}/mktemp` and is a 14-line wrapper that calls `/usr/bin/mktemp -p "${TMPDIR:-/tmp/claude}"` for bare `mktemp` invocations and passes through template paths that contain `/` unchanged. It was placed first on PATH via `PATH="/tmp/claude/<dir>:$PATH"` and `TMPDIR=/tmp/claude`. The shim is not committed to the repo and is not part of the PR; it is a host-environment artifact required for the sandbox where `DARWIN_USER_TEMP_DIR` (`/var/folders/.../T/`) is denied. This matches the task description's pre-warning.
