# i123 executor report — gate0:self-notices-rc2-no-watermark-file

**Commit:** 1ffb79aaec7fbdab05f8c9edc2479cf79f8a89bc (branch fleet/i123-self-notices-absent-watermark, base d2f2ec238; not pushed)

## git show --stat HEAD
 .agents/skills/mission-control/resources/gate-0-preflight.md       | 6 +++
 .claude/skills/mission-control/resources/gate-0-preflight.md       | 6 +++
 changelogs/unreleased/2026-11-19-gate0-self-notices-absent-watermark.md | 13 ++++++
 design_docs/implemented/v0_53_1/sprint-plan-gate0-self-notices.md  | 3 ++
 scripts/mission_gate0_self_notices.sh                              | 33 +++++---
 scripts/test_mission_gate0_self_notices.sh                         | 46 ++++----
 6 files changed, 90 insertions(+), 17 deletions(-)

## Steps
1. Pre-state: porcelain empty, branch fleet/i123-self-notices-absent-watermark. OK.
2. sha256 d5422b35...1314f matched. `git apply --check` rc=0, `git apply` rc=0.
3. Plan 4a step 6a edited (.claude copy), text as written.
4. `sync-agents-skills.sh --check` rc=1 (STALE: mission-control, expected, the .agents copy was not yet updated). Then sync rc=0; `--check` rc=0. `git status` showed only the .agents gate-0-preflight.md under .agents — no extra skills touched, nothing reverted.
5. Plan 4c amendment appended to the v0_53_1 design doc.
6. Changelog fragment created from plan §5 text (13 lines).

## Acceptance (plan §6)
| Command | rc | Key output |
|---|---|---|
| bash -n both scripts | 0 | — |
| --help \| grep -c '^  [012] ' | 0 | 3 |
| grep -c 'watermark channel unavailable' (script) | 1 (grep no-match) | 0 |
| grep -nE banned bash-3.2 constructs | 0 | only line 18 disclaimer |
| test_mission_gate0_self_notices.sh | 0 | 176 passed, 0 failed |
| --only watermark-all-absent | 0 | 8 passed, 0 failed |
| test_mission_directives.sh | 0 | 33 passed, 0 failed |
| shellcheck -S warning (both) | 1 (warnings present) | 9 SC2034 — same count as base d2f2ec238 |
| sync-agents-skills.sh --check | 0 | agents skills in sync (...) |
| test_agents_skills_sync.sh | 0 | 7 passed, 0 failed |
| make check-changelog | 0 | ✓ 5 changelog fragment(s) valid |
| git diff --check | 0 | no output |
| git diff --name-only | — | exactly the 5 expected tracked files + the new changelog |
| live stapledon 450/410 | 1 | watermark DEGRADED line; `crash: issue=410 at=2026-11-13T14:15:39Z rc=1 in-window=yes`; verdict `… watermarks 0/2, all absent: whole-issue read]` |
| live fleet 2341/2172 | 0 | DEGRADED line; `verdict: 0 crash notice(s) since 1970-01-01T00:00:00Z on #2341,#2172` |
| make test-launchd-drivers (background, ~6 min) | 0 | `run-suites: all 58 suites passed`; `ok scripts/test_mission_gate0_self_notices.sh`; 0 FAIL; log /tmp/mc-fleet/i123-exec-launchd.log |

Mutation drill (run on a scratch copy via G0_SCRIPT_UT, real script untouched):
- M1: `168 passed, 8 failed` (matches plan)
- M4: `171 passed, 5 failed` (matches plan)

## Deviations
1. **Commit trailer.** The plan and the executor brief specify `Co-Authored-By: Claude Opus 5.5`. I used `Co-Authored-By: Claude Haiku 5.5 <noreply@anthropic.com>` because the session's attribution instruction names Haiku 5.5 and I am Haiku 5.5. The commit message is otherwise exactly as briefed. If the Opus trailer is required, amend with `git commit --amend -F` on this branch before pushing.
2. Nothing else deviated. Every step ran as written.

## Surprises / notes
- The background launcher reported "completed" within seconds; that was only the wrapper subshell. The real make run continued and wrote the rc file about 6 min later. I confirmed it was still running via `pgrep -x make` before polling.
- `pgrep -f "make test-launchd-drivers"` matched the zsh tool wrapper, which echoes the command text. Use `pgrep -x make` instead.
- `grep -c` exits 1 when the count is 0, which is the intended result for the channel-unavailable check.
- Shellcheck is not rc 0 on baseline either (SC2034 warnings in both base and patched files); the count, not the exit code, is the gate.

## Not done (out of scope, per plan §7.5)
- World's fork (separate repo) may carry the same latent bug if its watermark files are also --mark-only. A follow-up ticket should be filed; I did not file it.
