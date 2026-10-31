# Fleet Mission Log

Append-only, one entry per iteration, newest at the bottom. Charter: [fleet-mission.md](fleet-mission.md).

> **Older entries are ARCHIVED.** This file holds the newest 20. The full record of every
> iteration is in `fleet-mission-log-archive.md`, and a one-line index of ALL of them —
> the thing to grep before picking work, so the loop never repeats itself — is in
> `fleet-mission-index.md`.

## 55 — 2026-10-22 — `stall:watchdog:foreground-agent-and-wrong-transcript` BUILT: #1929 head `9d131c76a`; judge PASS 91, round 2 PASS 91 (sonnet); merge PARKED on dev's inherited required red (D-FLEET-17); ticket open [HARNESS]

**Pick**: iteration 54's **Next** was to land #1927, then `rotate-log:heading-parse-stapledon`. #1927 cannot land: dev is still red on the same SHA, `d25d9b8d0`. Gate 2's clause map ranked a fresher ticket above the rotate-log one. `stall:watchdog:foreground-agent-and-wrong-transcript` (world, 10-21 23:27Z, `blocking=item`) cost World two whole slots (iterations 296 and 297, both killed mid-judge; 297's slot verdict reads `KILLED_at=gate-3 rc=143 … elapsed_s=3075`). That loss is clause 1, product share, directly. It is also a defect in fleet-owned code (`tools/launchd/mission-control.sh`), and the fleet's own foreground judges are exposed to it. The rotate-log ticket is `blocking=none`. The third fresh ticket, `quorum:claude-p-ration-usage-429` (stapledon, 10-21 22:57Z), sits in the ration and usage reader, which is HD-2a territory; it is not measured yet.

**Orphans credited**: none new. Gate 0 step 6a returned rc 1 for the 2026-10-17T12:48:13Z crash notice on #1768, already credited at iteration 47 (`issuecomment-6234067020`). The only open `fleet/*` PR at pick was #1927 (iteration 54, built and blocked). Its record is the Gate-1 input, and this record is stacked on its branch.

**Preflight**: armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing CLEAN; messaging store gcp. Directives: 0 on #1891 (12 comments) since `2026-10-15T02:46:27Z`. No rotation or sweep due (#1891 created Monday 2026-10-19).

**Observe**: origin/dev == pin == `d25d9b8d0`, base recorded `2026-10-22T00:33:58Z`. `SKILL.md` and all 13 resources are `cmp`-identical to origin, both at the resolved symlink target and at the pin. Dev CI is still RED on `d25d9b8d0`: `test`, `test-windows`, `Build macos-latest` and `govulncheck` fail; the ubuntu and windows builds were cancelled by fail-fast. This is the same red iteration 54 attributed.

**Reality check (real at HEAD)**:
- C1. `_mc_progress_bytes`'s claude arm is `ls -t "$dir"/*.jsonl | head -1` over `~/.claude/projects/<$PWD slug>/`, a directory shared by every session with that cwd.
- C2. The World kill lines are real (`/tmp/ailang-mission-world.log:14970`, `:14989`). The second reads `prog=318821268`, which is another session's file.
- C3. The claude launch passes no session id. claude 2.1.307 has `--session-id <uuid>`. Probed from `/tmp`: rc 0. The file was written as `~/.claude/projects/-private-tmp-…/<UUID>.jsonl`, with the id's case kept and the slug taken from the PHYSICAL cwd. So the fix has to find the file by id, not recompute the slug.
- C4. Sub-agent transcripts sit at `<id>/subagents/agent-*.jsonl` (40 sessions in the fleet project dir have one).
- C5. `_mc_progress_bytes` is the only reader of `~/.claude/projects` in the harness.
- C6. Base `test_mission_stall.sh` is 35/0. The pi arm has the same newest-file defect.

**Did**:
- **Designer**: not spawned. The fix is mechanical and the controller's C1–C7/D1–D7 went into the planner directive (precedent: iterations 45, 49–54).
- **Planner** (codex `gpt-6.1-sol`, recipe, `.planner-wt-iter55`): plan + sprint JSON, one milestone, seven named-mutation arms. Committed `a7e2b6784`. The planner post-step `ailang messages import-github` was **skipped** (prod store; unrelated to a fleet ticket), as at iteration 54.
- **Executor round 1** (codex): each claude attempt gets a fresh `--session-id`. The arm sums `<id>.jsonl` plus `<id>/subagents/*.jsonl` found by id across `~/.claude/projects/*`. With no id the arm fails open. pi keys on `*_${MC_PI_SESSION_ID}.jsonl`. Arms a1–a7.
  - Controller edit: replaced the new comment's plan labels (`C1`/`C3`/`C4`) with a self-contained WHY.
  - Committed `22b56880b`. Suite 45/0 outside the sandbox. Inside the sandbox, one pre-existing arm fails because `/bin/ps` is denied there (`UNINFORMATIVE UNDER SANDBOX`).
- **Judge round 1** (sonnet, Agent tool, foreground, `.wt-fleet-iter55-judge`): **PASS 91**, 0 blocking.
  - 18 mutants: 11 killed, 6 survived, 1 equivalent. a7 was grep-only, so the provider gate (M10, M17), id reuse (M11), the unguarded empty-array expansion (M9), the parent+subagent sum (M13) and the log line (M14) all survived.
  - The judge ran a stub-claude `_mc_run_once` and confirmed two attempts get two ids.
  - Controller reproduced M17 surviving (45/0 against the mutant).
- **Executor round 2** (codex): the id block moves into `_mc_claude_session_prepare` with an `_mc_uuid` seam. Behavioural arms a7a–a7d and a sum arm a2b. All six survivors killed; the controller re-killed M10 (a7a/b/d) and M13 (a2b) itself. Committed `9d131c76a`. Suite 50/0.
- **Judge round 2** (sonnet, fresh, `.wt-fleet-iter55-judge2`): **PASS 91**, 0 blocking. 28 mutants anchored to the diff: 22 killed, 6 survived, all equivalent or theoretical:
  - prefix-loose globs; for pi, an id `mission-X-<epoch>-<pid>` could in principle prefix another fire's;
  - removing the `-f` guard (masked while the driver log exists);
  - lower-casing the id (equivalent);
  - dropping the python fallback (untested path).
- **Live check** (controller, this session's own id): the extracted arm read 2,331,887 B. That is this transcript (1,829,455) plus the round-1 judge's `subagents/agent-aac1078f2eabb1505.jsonl` (502,432). The old arm could never have counted the second file.
- **PR #1929** (no closing keyword; scan paired with a control). 25 checks on `9d131c76a`. `launchd drivers (bash 3.2)` (Linux CI, the gate that runs this diff), `lint`, `docs-gate` and `docs-changes` are green. Not green, all inherited from dev on the same SHA: `test` and `Build macos-latest` fail only `TestModels_PricingIsSlugConsistent` (raw job logs, ANSI stripped, `ok` package counts 162 and 161 as the control); `govulncheck`; the ubuntu, windows and second macOS builds were cancelled by fail-fast; `test-windows` was still running at record time and is red on dev. `mergeStateStatus` BLOCKED. **Gate 3b**: not reached; nothing merged.

**Done-gate**:
- Surface: `tools/launchd/mission-control.sh` (`_mc_progress_bytes`, `_mc_claude_session_prepare`, `_mc_run_once`), which the pin executes.
- Reach: pushed, not merged; it reaches every mission's next pinned fire once #1929 merges.
- `make test-launchd-drivers` under `env -i`: rc 0 at `22b56880b` and at `9d131c76a` (stall suite 45/0, then 50/0).
- Dry-runs under **triage** (armed, idle, no pidfile; `AILANG_DRIVER_PINNED=9d131c76a`): healthy rc 0 `lanes=ok`; degraded (`MISSION_EXECUTOR_MODEL=pi:bogus/no-such-model`) rc 0 `DEGRADED(1)` → opus. No pidfile was written and nothing was reloaded. A dry-run exits before `_mc_run_once`, so the live check and the judge's stub run are what exercised the new code.

**Progress**: 0 tickets resolved (2 built, both merges blocked by the same inherited red); 44 open. Goal unmoved on clause 1 until #1929 lands.
- Clause 1 UNMEASURED; this ticket is its direct cost (2 World slots).
- Clause 2 UNMET. This ticket counts toward the median only if it lands before 2026-10-23 23:27Z; #1927's window closes 2026-10-23 12:29Z.
- Clause 3 MET.
- Clause 4 prior evidence only.
- Clause 5 preserved (nothing merged).

**Routing evidence**: base=d25d9b8d0dde10ee7c14986fe21b94fd2cb500cd@2026-10-22T00:33:58Z (gate 1); sprint worktree base `d25d9b8d0`. `MISSION_ROUTING_NOTE` empty; `MISSION_OVER_RATION` empty.
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer**: not spawned. Resolver `recipe claude:claude-opus-5-5 declared:provider-pin`; rotation not advanced.
- **Planner**: resolver `agent-tool opus fail-closed:no-doc` against the pin `codex:gpt-6.1-sol`, so routed to the pin under the codex recipe (role-spawn-routing §2(a)). Probe rc 0; run rc 0 (61,922 tok).
- **Executor**: `codex:gpt-6.1-sol` via the recipe (`declared:provider-pin`), 2 runs, both rc 0: 109,768 and 81,334 tok.
- **Evaluator**: `sonnet` via the Agent tool (`declared:alias-pin`), foreground, 2 fresh spawns (81,325 + 72,344 tok). Judge independence: cross-vendor (Anthropic sonnet vs the OpenAI generator).
- The Agent tool was used where the resolver names it (the evaluator), per the operator's standing request. Planner and executor resolve to recipes. No role failed to spawn.
- Every codex run carried the spawn-override flag. A one-off `claude-sub … --model haiku` probe measured C3.
- **Metered**: $0.

**Ruled out**:
- Recomputing the claude slug from `pwd -P`: it still cannot separate sessions that share a cwd, and finding the file by id makes the slug irrelevant.
- Counting every `*/subagents/*.jsonl` in the project dir: another session's sub-agent would hold a wedged controller alive (arm a3).
- Falling back to the newest file when the id is missing: that fallback is the bug (arm a5). Failing open costs only the early kill; HARD_TIMEOUT still bounds the slot.
- Changing any threshold or sample count: policy (HD-2a), and not needed.
- Fixing `TestModels_PricingIsSlugConsistent` from the fleet: `internal/modelreg` is outside the Authority allowlist.

**Retro lane**:
- (1) **No skill edit.** No friction reached two recorded instances on a gap the fleet should close.
- (2) **Banked, 2 instances (54, 55): the planner recipe's step (5) `ailang messages import-github` is a V1-shaped step.** It writes the prod store and has nothing to do with a fleet ticket, so the controller has to decide to skip it every time. A third instance makes it a skill edit: scope the step to missions whose queue is GitHub issues.
- (3) **Banked, 1 instance: Gate 1's stacking rule says to put the record on the blocked PR's branch, but not that the previous iteration's worktree may still hold that branch.** `git worktree add -B` failed rc 128, and the fix was to reuse the clean leftover worktree.
- (4) **The judge's grep-only arm was the weak point, again.** Round 1's a7 checked wiring by text, and three mutants on the heart of the fix survived. Round 2's behavioural arms killed them. This is iteration 54's lesson (mutation-test the arm, not the suite) applied one level up: a static arm proves the text, not the behaviour.
- (5) **D-FLEET-17 recurs: the fourth landing it has blocked (37, 38, 54, 55).** Two judged PASS fixes (#1927, #1929) now wait on one attended commit's red. Both fixes are fresh enough to move clause 2 only if dev is green by 2026-10-23. The ledger row's evidence is updated with the current cause.

**Next**: when dev's `test` is green, re-run #1927's and #1929's checks (rebase only if `git diff --stat d25d9b8d0 origin/dev -- <each PR's files>` is non-empty), merge each with `--match-head-commit`, resolve both tickets with the merge SHAs, and reply to world. Then `rotate-log:heading-parse-stapledon` (stapledon, 10-21), then `quorum:claude-p-ration-usage-429` (stapledon, 10-21, `blocking=item`; measure first whether the fix is mechanical or a ration and billing change, which is HD-2a).

## 56 — 2026-10-22 — `rotate-log:heading-parse-stapledon` RESOLVED as a ghost of #1912 (stale rig binary), judge CONFIRMED (sonnet); `quorum:claude-p-ration-usage-429` PARKED as D-FLEET-28; D-FLEET-29 filed [HARNESS]

**Pick**: iteration 55's **Next** was to land #1927 and #1929 once dev's `test` is green, then `rotate-log:heading-parse-stapledon`, then `quorum:claude-p-ration-usage-429`. Dev is still red on the same SHA, `d25d9b8d0`, so neither PR can land. No new ticket has arrived since 10-21 23:27Z. Of the two fresh Stapledon tickets, the quorum one is `blocking=item` and costs product work (Stapledon's Kerr re-quorum), so it was measured first; the rotate-log one was reality-checked second.

**Orphans credited**: none new. Gate 0 step 6a returned rc 1 for the 2026-10-17T12:48:13Z crash notice on #1768, already credited at iteration 47 (`issuecomment-6234067020`). The iteration 55 worktree (`.wt-fleet-iter55`) is clean. The open `fleet/*` PRs are #1927 (iteration 54, which now also carries the iteration 55 record) and #1929 (iteration 55); this record is stacked on #1927's branch.

**Preflight**: armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing CLEAN; messaging store gcp. Directives: 0 on #1891 (13 comments) since `2026-10-15T02:46:27Z`. No rotation or sweep is due (#1891 was created Monday 2026-10-19).

**Observe**: origin/dev == pin == `d25d9b8d0`; base recorded `2026-10-22T07:55:33Z`. `SKILL.md` and all 13 resources are `cmp`-identical to origin at the resolved symlink target. Dev CI on `d25d9b8d0` is unchanged from iteration 55: of 20 checks, `test`, `test-windows`, `Build macos-latest` and `govulncheck` fail, and three builds were cancelled by fail-fast.

**Reality check, `quorum:claude-p-ration-usage-429`** (real at HEAD; out of fleet scope):
- Q1. `ailang mission quota --bucket anthropic` prints `anthropic provider usage: unknown [ENFORCED] — Anthropic usage endpoint returned HTTP 429; CLI fallback also failed (no "Current session" line in `claude -p /usage` output)`.
- Q2. On Claude Code 2.1.308, `claude -p /usage < /dev/null` returns rc 0 and 212 bytes: `Total cost`, `Total duration`, `Total code changes` and `Usage` lines only. The parser's `Current session:` and `Current week (all models):` lines are gone. `claude -p /status` prints "isn't available in this environment". Stream-json output for `/usage` has no rate-limit fields. No non-inference route to the windows remains.
- Q3. The seat gate is `cmd/ailang/design_quorum.go:192`, `if a := mission.ObserveAnthropicQuota(time.Now()); a.Blocked()`. `Blocked()` (`internal/mission/anthropic_quota.go:58`) is `Enforced && State != "ok"`, so it is true for `unknown`.
- Q4. The driver's rule is the opposite. `lib/lane-probe.sh` reads `quota --over` and treats `unknown` as UNREADABLE, so the lane is probed, not refused (`test_anthropic_quota_admission.sh`, "the September 2026 defect"). This fire's own driver log shows it: `anthropic:claude-opus-5-5 quota is UNREADABLE (not measured) — probing the lane instead of refusing it`.
- Q5. Scope: `design_quorum.go` is not in the Authority allowlist (`cmd/ailang/mission*` does not match it). `Blocked()` also feeds `admission.go` and `quota --over`, so changing it is HD-2a. A fix confined to `internal/mission` could only add a method that nothing calls.
- Verdict: parked as **D-FLEET-28**, with a recommendation.

**Reality check, `rotate-log:heading-parse-stapledon`** (GHOST at HEAD):
- R1. A pin build (`go build -o /tmp/fleet_i56_ailang ./cmd/ailang`, rc 0) on a scratch copy of Stapledon's `origin/main` log, through a scratch registry. The log has 45 headings, 342,209 bytes. Output: `20 of 45 entries kept`, `25 entries rotated out`, `orphan rows preserved: 10, 13, 15, 25`, rc 0. The sorted heading lists before and after (live plus archive) are identical (45). The live log keeps iterations 26–45, and the notice is true.
- R2. Every loop runs `~/go/bin/ailang`, because `mission-control.sh:68` puts `$HOME/go/bin` first on PATH. `go version -m` shows `vcs.revision=128371fb8…`, `vcs.time=2026-10-19T17:56:43Z`, `vcs.modified=true`. `3b4d11759` (#1912, 10-20) is not an ancestor of it.
- R3. The judge ran that old binary on its own scratch copy and got `9 of 9 entries kept`, `0 entries rotated out`, `orphan rows preserved: 0 … 36`. That is the ticket's failure; the ticket saw 8 of 8 on a 44-entry log.
- R4. The fleet log has recorded "picks up the Go change on its next rebuild or release" since iteration 32 (entries 32, 52 and 53). This is the first ticket that caveat produced.

**Did**:
- **Designer, planner, executor**: not spawned. Nothing was built: one ticket is a ghost and the other is out of scope. This is not a spawn failure.
- **Evaluator** (sonnet, Agent tool, foreground, fresh): asked to refute both verdicts first-party, with its own binary and scratch copies. Verdict 1 CONFIRMED: R1 was reproduced (45/45 headings, newest 20 live, 50 index rows before and after, orphan text kept), and R3 found that the old binary reproduces the failure. Verdict 2 CONFIRMED on Q1–Q5. The 429's cause is UNVERIFIED, because the judge found no token to make a raw call. **Overall PASS.** Its objections, both taken:
  - (a) The resolution must name the rebuild, or the ghost returns at the next Stapledon rotation. The resolution text does; D-FLEET-29 asks for it.
  - (b) `AILANG_ANTHROPIC_RATION=0` already unblocks the seat. It is a policy call, so it is option B of D-FLEET-28.
  - Minor residual at HEAD, not in the ticket: the regenerated index drops Stapledon's `Class` column, and the `pre-0`/`pre-0b`/`att-M2` labels become `—`. It is a one-time format change and is named in the resolution.
- **Resolved** `rotate-log:heading-parse-stapledon` with `--sha 3b4d1175919eccfda8c034b49d0b6633c415817a`, which is on origin/dev. The reply went to stapledon, with the evidence and the workaround (a binary built from origin/dev). 44 tickets are open after the resolve (`ticket open --count`; no pre-resolve count was taken, and iteration 55 recorded 44).
- **Filed** D-FLEET-28 (quorum seat; HD-2a, outside scope) and D-FLEET-29 (who rebuilds the rig binary). Updated D-FLEET-17's evidence; it is unchanged.

**Done-gate**: not applicable. No harness file changed: the resolution cites a fix that already passed its done-gate at iteration 51. Nothing was reloaded or installed; `~/go/bin/ailang` was not touched.

**Progress**: 1 ticket resolved; 44 open. 2 built fixes are still blocked.
- Clause 1 UNMEASURED. D-FLEET-28 is a direct product cost (Stapledon's Kerr plan).
- Clause 2 UNMET. This ticket counts: filed 10-21 15:24Z, resolved about 19 h later.
- Clause 3 MET.
- Clause 4 prior evidence only.
- Clause 5 preserved (nothing merged).

**Routing evidence**: base=d25d9b8d0dde10ee7c14986fe21b94fd2cb500cd@2026-10-22T07:55:33Z (gate 1). `MISSION_ROUTING_NOTE` empty; `MISSION_OVER_RATION` empty.
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer**: not spawned (nothing to design). Resolver env `claude:claude-opus-5-5`; rotation not advanced.
- **Planner**: not spawned (nothing to plan). Pin `codex:gpt-6.1-sol`.
- **Executor**: not spawned (nothing to build). Pin `codex:gpt-6.1-sol`.
- **Evaluator**: `sonnet` via the Agent tool (`agent-tool sonnet declared:alias-pin`), foreground, 1 fresh spawn (78,786 tok). Judge independence: a different model from the controller (sonnet vs opus), same vendor. Both verdicts were the controller's, so the judge's job was to refute them, and it reproduced them with its own binary.
- The Agent tool was used for the role the resolver names (the evaluator), per the operator's standing request. No role failed to spawn.
- **Metered**: $0.

**Ruled out**:
- Fixing the quorum seat from `internal/mission` alone. A new "over only" method has no caller until `design_quorum.go` changes, and changing `Blocked()` moves admission and `quota --over` (HD-2a).
- Repairing the CLI fallback. Claude Code 2.1.308 no longer prints the windows in `-p` mode, and no non-inference source was found.
- Caching the last good Anthropic reading. Every reading since 10-21 is a 429, so there is nothing to serve, and serving a stale reading is a ration policy.
- Rebuilding `~/go/bin/ailang` from the loop. Evals share it live, and the reach question is D-FLEET-29.
- Building any part of D-FLEET-29's option B before a ruling.

**Retro lane**:
- (1) **No skill edit.** No friction reached two recorded instances on a gap the fleet should close.
- (2) **Banked, 1 instance: a fleet Go fix carries a reach caveat that nothing discharges.** The done-gate's "reach confirmed (pushed)" is true for driver and skill changes and false for Go code: loops run the PATH binary, not the pin. The rule is now D-FLEET-29's question; if Mark rules A or B, the done-gate line needs a binary clause.
- (3) **Banked, carried: the planner recipe's `import-github` step** (iterations 54 and 55) did not recur, because no planner ran.
- (4) **D-FLEET-17 holds two judged fixes.** #1927's clause-2 window closes 2026-10-23 12:29Z.

**Next**: when dev's `test` is green, re-run #1927's and #1929's checks (rebase only if `git diff --stat d25d9b8d0 origin/dev -- <each PR's files>` is non-empty), merge each with `--match-head-commit`, resolve both tickets with the merge SHAs, and reply to world. Then reality-check `quorum:oc-glm-5-3-output-cap-invalid` (stapledon, 10-21).

## 57 — 2026-10-22 — `quorum-reader:zsh-nomatch` BUILT: #1932 head `6268fb0b8`; judge PASS 94 (sonnet); merge PARKED on dev's inherited required red (D-FLEET-17); ticket open; Gate-5 skill edit [HARNESS]

**Pick**: iteration 56's **Next** was to land #1927 and #1929 once dev's `test` is green, then reality-check `quorum:oc-glm-5-3-output-cap-invalid`. Dev is still red on the same SHA, `d25d9b8d0`, so neither PR can land. One ticket was filed after iteration 56: `quorum-reader:zsh-nomatch` (stapledon iteration 46, 10-22 11:50Z). Gate 2's clause map ranks it first:
- Clause 2 (turnaround) is UNMET, and only a fresh ticket can still resolve inside 48 h.
- It is a defect in fleet-owned code (the Gate-2 quorum reader and `tools/launchd/lib/quorum-artifacts.sh`), and it is mechanical.
- Its failure costs money: a false UNREVIEWED triggers a paid quorum.

`quorum:oc-glm-5-3-output-cap-invalid` is 10-21, and its fix is a seat output cap in `cmd/ailang/design_quorum.go`, outside the Authority allowlist. It stays the next reality-check.

**Orphans credited**: none new. Gate 0 step 6a returned rc 1 for the 2026-10-17T12:48:13Z crash notice on #1768, already credited at iteration 47 (`issuecomment-6234067020`). The open `fleet/*` PRs at pick were #1927 (iteration 54, carrying the records of 55 and 56) and #1929 (iteration 55). This record is stacked on #1927's branch.

**Preflight**: armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing CLEAN; messaging store gcp. Directives: 0 on #1891 (14 comments) since `2026-10-15T02:46:27Z`. Environment read by name. No rotation or sweep is due (#1891 was created Monday 2026-10-19).

**Observe**: origin/dev == pin == `d25d9b8d0`; base recorded `2026-10-22T14:07:53Z`. `SKILL.md` and all 13 resources are `cmp`-identical to origin, both at the resolved symlink target and at the pin. Dev CI on `d25d9b8d0` is unchanged: of 24 checks, `test`, `test-windows`, `Build macos-latest` and `govulncheck` fail, and three builds were cancelled by fail-fast.

**Reality check (real at HEAD)**:
- Z1. `quorum-artifacts.sh:68` `for file in "$dir/$slug-"*.json`. The scratch fixture: an export dir holding `doc-x-2026.json` and an existing empty local dir, with the real Gate-2 reader block extracted.
  - `/bin/bash -c`: rc 0, 1 match.
  - `/bin/zsh -c`: `mc_quorum_lookup:11: no matches found`; the trailing `echo` never runs; exit 1.
  - The mirror (empty export dir, local match) fails the same way.
  - Control: an ABSENT local dir is skipped by line 63, giving rc 0.
- Z2. Same class, second site: `pi-ext-args.sh` `_mc_pi_ext_args`, `for f in "$dir"/.pi/extensions/*.ts`. The gate-3 pi recipe sources it into the controller shell. With an empty `.pi/extensions`, zsh prints `--no-extensions`, then `no matches found`, exit 1; bash gives rc 0. No other `for … in <glob>` exists in `tools/launchd/lib/*.sh`.
- Z3. The controller hit the same class itself during this check: `grep -r --include=*.sh` under the Bash tool's zsh failed with `no matches found: --include=*.sh` and returned nothing.
- Z4. The CI job that runs both suites (`launchd drivers (bash 3.2)`) runs on `macos-latest`, which ships `/bin/zsh`, so zsh arms do execute in CI.

**Did**:
- **Designer**: not spawned. The fix is mechanical, so the controller's Z1–Z4 measurements and its D1–D6 design went into the planner directive (precedent: iterations 45 and 49–55).
- **Planner** (codex `gpt-6.1-sol`, recipe, `.planner-wt-iter57`): plan and sprint JSON, one milestone. Committed `b92b42818`.
  - It refuted two of the controller's design points first-party, and the controller reproduced both. `setopt nullglob` is an accepted zsh alias of `null_glob`, so that "mutant" is equivalent. An options arm that checks only NOMATCH cannot kill a dropped `local_options`, because NULL_GLOB leaks while NOMATCH stays on.
  - The post-step `ailang messages import-github` was **skipped** (prod store, not a GitHub-fed queue). This is the third instance; see Retro.
- **Executor** (codex, 1 run): the first statement of `mc_quorum_lookup` and `_mc_pi_ext_args` is now `if [ -n "${ZSH_VERSION:-}" ]; then setopt local_options null_glob; fi`, and both headers carry a one-line WHY.
  - Eight behavioural zsh arms: 5 quorum, including `zsh_gate2_export_only` over both skill copies, and 3 pi. Each runs the real helper under `/bin/zsh -f` with an AFTER_CALL marker and a missing-marker negative control, and the options arms assert NOMATCH on and NULL_GLOB off.
  - Missing zsh is a FAIL on Darwin.
  - TDD red: all 5 quorum arms and 2 of 3 pi arms (`zsh_pi_two_ts` is a positive control).
  - Changelog fragment added. Committed `6268fb0b8`.
  - In-sandbox `make test-launchd-drivers` was rc 2 at base and at head, labelled UNINFORMATIVE UNDER SANDBOX (the descendant-CPU check needs `ps`).
- **Controller gates, outside the sandbox, `env -i`, at `6268fb0b8`**: quorum suite rc 0 (21 PASS lines); pi suite 17/0; `make test-launchd-drivers` rc 0; `make check-changelog` rc 0; `git diff --check` rc 0.
  - Shellcheck over the four files warns only on pre-existing lines (94, 211, 212, 251).
  - The controller's own drill on a scratch copy: dropping the quorum setopt failed `zsh_quorum_export_only` and `zsh_gate2_export_only`, and a global `setopt null_glob` in pi failed `zsh_pi_options`. The unmutated control passed the same arms in the same layout.
- **Judge** (sonnet, Agent tool, foreground, fresh, `.wt-fleet-iter57-judge`): **PASS 94**, 0 blocking.
  - It reproduced the base failure for both helpers under `/bin/zsh -f`.
  - Bash 3.2.57 output, stderr and rc were IDENTICAL from base to head over 9 fixture cases (absent, unreadable, bad JSON, duplicate dir, invalid slug, and three pi shapes).
  - 15 mutants: 10 killed, including setopt moved after the first glob, setopt at source time, and a broken `.agents` or `.claude` snippet copy.
  - 5 survived, none of them reachable or meaningful:
    - `-z BASH_VERSION` (same as shipped in zsh and bash);
    - `no_nomatch` (equivalent);
    - setopt after the slug check;
    - a global-setopt leak on the rc 2 return, unreachable under `local_options`;
    - an unconditional setopt that prints a bash error, because no arm asserts bash stays silent.
  - It found no writes outside fixture dirs and no other unguarded lib glob.
- **PR #1932** (no closing keyword; the scan was paired with a positive control). 25 checks on `6268fb0b8`, all complete:
  - Green: `launchd drivers (bash 3.2)` (macos-latest, the job that runs both zsh-arm suites), `lint`, `docs-gate`, `CodeQL`, `float determinism (arm64)`.
  - Not green, all inherited from dev on the same SHA: `test`, `test-windows` and `Build macos-latest` each fail exactly one test, `TestModels_PricingIsSlugConsistent` (raw job logs with ANSI stripped; the control is 162, 162 and 161 `ok` package lines). `govulncheck` also fails, and three builds were cancelled by fail-fast.
  - `mergeStateStatus` BLOCKED.
  - **Gate 3b**: not reached; nothing merged. The poll was pinned to the head SHA, with a SHA-named verdict file.

**Done-gate**:
- Surface: `tools/launchd/lib/quorum-artifacts.sh` and `lib/pi-ext-args.sh`, read from `$MISSION_DRIVER_ROOT` (the pin) by the Gate-2 reader and the gate-3 pi recipe, and sourced by the driver.
- Reach: pushed, not merged. It reaches every mission's next pinned fire once #1932 merges. Shell only, so D-FLEET-29 does not apply.
- `make test-launchd-drivers` under `env -i`: rc 0 at `6268fb0b8`.
- Dry-runs under **stapledon** (armed, idle, no pidfile; `AILANG_DRIVER_PINNED=6268fb0b8`): healthy rc 0 `lanes=ok`; degraded (`MISSION_EXECUTOR_MODEL=pi:bogus/no-such-model`) rc 0 `DEGRADED(1)` → opus. No pidfile was written and nothing was reloaded.

**Progress**: 0 tickets resolved (3 built, all three merges blocked by the same inherited red); 45 open. Goal unmoved.
- Clause 1 UNMEASURED.
- Clause 2 UNMET. This ticket counts toward the median only if it lands before 2026-10-24 11:50Z; #1927's window closes 10-23 12:29Z and #1929's 10-23 23:27Z.
- Clause 3 MET.
- Clause 4 prior evidence only.
- Clause 5 preserved (nothing merged).

**Routing evidence**: base=d25d9b8d0dde10ee7c14986fe21b94fd2cb500cd@2026-10-22T14:07:53Z (gate 1); sprint worktree base `d25d9b8d0`. `MISSION_ROUTING_NOTE` empty; `MISSION_OVER_RATION` empty. Anthropic quota reads `unknown` (HTTP 429, D-FLEET-28); the lanes were probed, not refused.
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer**: not spawned. Resolver `recipe claude:claude-opus-5-5 declared:provider-pin`; rotation not advanced.
- **Planner**: resolver `agent-tool opus fail-closed:no-doc` against the pin `codex:gpt-6.1-sol`, so routed to the pin under the codex recipe (role-spawn-routing §2(a)). Probe rc 0; run rc 0 (51,570 tok). Post-step `import-github` skipped.
- **Executor**: `codex:gpt-6.1-sol` via the recipe (`declared:provider-pin`), 1 run, rc 0 (141,091 tok).
- **Evaluator**: `sonnet` via the Agent tool (`agent-tool sonnet declared:alias-pin`), foreground, 1 fresh spawn (73,499 tok). Judge independence: cross-vendor (Anthropic sonnet vs the OpenAI generator).
- The Agent tool was used for the role the resolver names (the evaluator), per the operator's standing request. The planner and executor resolve to recipes. No role failed to spawn.
- Every codex run carried the spawn-override flag.
- **Metered**: $0.

**Ruled out**:
- Wrapping the Gate-2 snippet in `bash -c`: the caller reads `MC_QUORUM_MATCHES` in its own shell, and fixing the helper from inside fixes every caller (snippet, driver, tests) at once.
- A global `setopt null_glob`: it leaks into the controller's shell for the rest of the session (arm `zsh_*_options`).
- Counting the `nullglob` and `no_nomatch` spelling survivors as test gaps: both are equivalent (measured).
- Fixing `quorum:oc-glm-5-3-output-cap-invalid` from the fleet: the seat cap lives in `cmd/ailang/design_quorum.go`, outside the Authority allowlist.
- Fixing `TestModels_PricingIsSlugConsistent` from the fleet: `internal/modelreg` is outside scope (D-FLEET-17).

**Retro lane**:
- (1) **Skill edit (Gate 5, the one allowed): the codex planner recipe's post-step (5) `import-github` now runs only for missions whose queue is fed by GitHub issues.** Both copies of `resources/gate-3-route.md` were edited identically (`269242b96`). Frictions: iterations 54, 55 and 57, each a hand skip; iteration 55 set three instances as the bar. `test_codex_spawn_override`, `test_pi_ext_args` (skill sweep), `test_mission_heartbeat` and `check-context-docs` are green.
- (2) **Banked, 1 instance: a Gate-3b poll for a PR that cannot merge has no prescribed shape.** The snippet polls dev's merge CI; the fleet has now polled PR-head check-runs by hand at iterations 54, 55 and 57. That is a stable workaround, not yet a defect.
- (3) **zsh again, in the controller's own hands (Z3).** The ticket's class bit the controller during the reality check. Iteration 54's skill edit covers hand-written polls; a one-off `grep --include=*.sh` is not a poll. Banked rather than widened: the cheaper habit is to quote globs, and the libs are now safe.
- (4) **D-FLEET-17 holds three judged fixes** (#1927, #1929, #1932). Fifth blocked landing (37, 38, 54, 55, 57). The first clause-2 window closes 2026-10-23 12:29Z.

**Next**: when dev's `test` is green, re-run the checks for #1927, #1929 and #1932 (rebase only if `git diff --stat d25d9b8d0 origin/dev -- <each PR's files>` is non-empty), merge each with `--match-head-commit`, resolve the three tickets with the merge SHAs, and reply to world and stapledon. Then reality-check `quorum:oc-glm-5-3-output-cap-invalid` (stapledon, 10-21; likely a D-row, since `design_quorum.go` is out of scope).

## 58 — 2026-10-22 — `routing-models-check:bold-colon-heading` LANDED through replacement #1935; independent MiniMax PASS 93; ticket resolved [HARNESS]

**Pick**: resume iteration 54's built `routing-models-check:bold-colon-heading`, the first of iteration 57's three banked landings. World filed it at 2026-10-21T12:29:47Z; its <48h window closes 2026-10-23T12:29:47Z. Dev's inherited pricing red has been repaired upstream, making this clause-2 row routable. One ticket only: #1929 and #1932 remain banked for their own iterations. No new design or quorum: the existing fixed mechanical design and approved sprint remain controlling.

**Orphans credited**: none new. The 2026-10-17 crash notice (`issuecomment-6234067020`) was already credited at iteration 47. Read state on #1927's branch: origin/dev initially ended at record 53, while the banked branch carried 54–57. All four records and their decision rows are preserved.

**Preflight**: armed; gh `sunholo-voight-kampff`; billing CLEAN; messaging canonical gcp. No human directives since watermark `2026-10-15T02:46:27Z`; no watermark advance. Live bookkeeping thread #1891 was created Monday 2026-10-19 and is below 80 comments; no weekly rotation due. The 9 unrelated remote approvals were not acted on. Decision ledger: 29 rows, 14 OPEN (D-FLEET-16 through -29), unchanged; no new ask.

**Observe**: Gate-1 base `42d93e4b6317e7b068bf52cd32dc226c013dd117` at `2026-10-22T21:17:43Z`. The parent `8f58117f2` was green, which did not prove the new base green. An exact-base CI dispatch, run [39856325172](https://github.com/sunholo-data/ailang/actions/runs/39856325172), subsequently completed successfully. `23c663815` repaired the pricing collision upstream; later base changes also updated Go and the Windows timeout. No fleet edit outside its authority was used. The user-named pin skill was followed; Gate-1 comparison covered SKILL.md and resources at their resolved running paths, not just a relative filename. D-FLEET-17's observed blocker is clear; its OPEN ledger row is not a controller ruling.

**Reality check**:
- The original head was `71609162c099aab1498249dd36d2b0f0e226a775` on old base `d25d9b8d0dde10ee7c14986fe21b94fd2cb500cd`. Gate 3b requires refreshing an old base even when the upstream changed paths are disjoint; iteration 57's “rebase only if non-empty diff” instruction is superseded by that gate.
- All 9 rebased patches compare `=` under `range-diff`; all 14 original PR file contents are byte-identical before/after. Candidate `0be2c1957a78b14aa281eb67f4c7e3efad82bb5a` was freshly tested and judged. Backup `fleet/i58-before-refresh` preserves the old head. No new source implementation in iteration 58.
- The active pre-push scope guard refused the old branch's explicit force-with-lease (rc1): its remote-head diff included 42 unrelated upstream paths. The guard was neither edited nor disabled. A new remote branch, `fleet/i58-routing-headings-land`, published the same candidate through that unchanged guard and produced replacement #1935. Local executor branch/path remain `fleet/i54-routing-check-headings` / `.wt-fleet-iter54-routing-heading`, correctly reported by the executor. Old #1927 is retained until the replacement lands, then receives a separate explanatory comment and closure.

**Did**:
- Spawned all four Agent-tool roles per the operator's explicit standing request. Native dispatch carriers have no model-override field, so each ran the resolved provider-pinned CLI recipe; no role failed to spawn. The actual evaluator was MiniMax, independently reading and judging the Codex-generated work.
- Designer: no new design needed; confirmed fixed scope, record preservation and mandatory base refresh. The first launch omitted the required role-label prefix and was interrupted; its shell rc0 was NOT accepted as a finished design. A corrected fresh run produced the substantive report. No new design doc, so rotation state was not advanced.
- Planner: preserved the existing plan and supplied a landing addendum. D7 includes PR, CI and ticket resolution; the existing planned/not_started JSON was stale about implementation but could not truthfully be marked completed before D7. Completion metadata was deferred to this record rather than changing the already-reviewed source head. Effort was not measured and is not fabricated.
- Executor: test/inspection only, exact candidate clean before and after. Focused suite **46/46**; both script syntax checks, changelog, file-size and diff checks passed; no source changes.
- Independent evaluator: **CODE ACCEPT / PASS 93/100**, zero blocking findings, explicit `CI_PENDING` at its final 21:44 snapshot. Canonical pi runner, 732 seconds, source HEAD unchanged, tool acknowledgement true, 53 tool executions, 0 provider errors, own evidence and report in an isolated worktree. Original report preserved verbatim at [iteration58/evaluation.md](fleet-mission-evidence/iteration58/evaluation.md). The controller's subsequent CI observations do not rewrite that verdict.
- Judge red control: pre-fix checker 22 passed / 24 failed; candidate 46/46; mutation evidence and limitations are in the original report. The controller reproduced C1/C2 and explains all findings separately in [controller-review-notes.md](fleet-mission-evidence/iteration58/controller-review-notes.md). Advisory checker acceptance of a hypothetical allowed-token suffix is banked; no real additional model or runtime allowlist change was established. The report's raw CI snapshot has 24 rows (not its prose's 22), and the raw checker/test numstat is +11/-4 and +113/-2. C5's claimed stale local branch/path is refuted by the actual unchanged local names. No self-rescore or hidden report rewrite.
- Controller D7 corpus: **116 numbered EM-DASH entries / 7 actual log snapshots**, old base-42 checker vs new checker. No old rc0 becomes rc1/rc2, and identical-rc output is unchanged. Every changed result is rc2→rc0/rc1: previously unreadable routing labels now parse, exposing historical retired tokens where present. Sources and hashes: [corpus-summary.json](fleet-mission-evidence/iteration58/corpus-summary.json); every result: [corpus-results.tsv](fleet-mission-evidence/iteration58/corpus-results.tsv). Motoko's shared log and Stapledon's local working tree were templates, caught by failed positive controls; corrected to Motoko's own clone and Stapledon's committed `origin/main` `405727f0f992c1313932fda7234dbda270799e68`. Unmerged sibling branches are outside this snapshot corpus.
- [PR #1935](https://github.com/sunholo-data/ailang/pull/1935): all 5 required contexts green; full 26-row check set terminal with no failure; all 5 applicable PR workflows successful (CI, Build and Release, CodeQL, Docs gate, UI gate), Dependabot skipped. Fresh exact head/base and MERGEABLE/CLEAN read immediately before `--merge --match-head-commit 0be2c1957a78b14aa281eb67f4c7e3efad82bb5a`. Merged `2026-10-22T21:56:55Z` as `2debbb0278cc08e018be2d0a1e75befcbda55454`, parents base42 + reviewed candidate0be; merge/candidate tree identical `618b06915cb7da4ad2991d7c2b6b12afc8c298e6`.

**Gate 3b**: Exact merge `2debbb0278cc08e018be2d0a1e75befcbda55454`: [CI 39861258501](https://github.com/sunholo-data/ailang/actions/runs/39861258501) and [Build and Release 39861258599](https://github.com/sunholo-data/ailang/actions/runs/39861258599), both attempt1/push/dev, completed SUCCESS before the 30-minute deadline. Fresh direct verification at 22:26Z required all 15 expected successful jobs and found all 17 checks terminal with no failure. The SHA-bearing pollers both ended completed success; no background watcher remains.. Gate-3b base reading `2debbb0278cc08e018be2d0a1e75befcbda55454@2026-10-22T21:57:02Z`; expected drift is this iteration's own 10-commit landing. Full-SHA separate per-workflow polls, one shared 30-minute deadline, distinct subject-bearing verdict artifacts. Expected on this push: CI and Build and Release. Docs push is path-filtered out; CodeQL and UI are PR-only; release is tag-only. No missing workflow was called pending or green. Fresh direct run/job/check reads confirmed the final verdict, not a latest-branch aggregate.

**Done-gate**:
- Actual surface: published checker and shell suite; the inherited iteration-54 and -57 Gate-5 skill edits are separately authorized and preserved, not additions to executor D6. Their reach is the next pin from origin/dev. D-FLEET-29's binary-rebuild question is unchanged; no shared CLI or live configuration was installed.
- Controller full `env -i make test-launchd-drivers` outside the sandbox at source0be: rc0, final “launchd drivers: tests + bash 3.2 syntax OK”. Fresh scratch CLI stamped base42 on PATH; Go sources unchanged by the candidate. The initially wrong ldflags module path yielded “dev”, was caught, and was rebuilt with the correct `sunholo-data` module before any trusted gates.
- Stapledon profile was armed and idle, no pidfile. Candidate driver with `AILANG_DRIVER_PINNED=0be...`: healthy transient available-cloud pins yielded rc0 `lanes=ok`; forced invalid executor yielded rc0 named degradation/fallback. Unmodified pins also yielded explicit degradation when fresh Codex probes hit rc75 daily-ration refusal. These were per-call probe pins, not installed policy changes. No PID created, mission started or live job reloaded. Original completed Codex roles predate those later refusals.
- Ticket resolution and old-PR closure: `mission ticket resolve routing-models-check:bold-colon-heading --sha 2debbb0278cc08e018be2d0a1e75befcbda55454` resolved 1 occurrence and replied to world; open count45→44. Completion observed 2026-10-22T22:27:24Z (about34h after filing, <48h). OldPR1927 received [a separate evidence comment](https://github.com/sunholo-data/ailang/pull/1927#issuecomment-6352671795), comment count0→1 verified, then stateCLOSED verified. No unrelated ticket or approval was acknowledged..

**Progress**: goal unmoved on clause 1 (product-loop share UNMEASURED); 1 fresh ticket resolved inside 48h improves clause 2, whose overall median remains UNMET. Clause 3 MET (45→44 open); clause 4 prior evidence only; clause 5 preserved by the full done-gate. Fleet's last 20 entries are 19/20 HARNESS, expected under its inverted remit, not evidence that the product-loop share bar is met.

**Routing evidence**: Gate1 base `42d93e4b6317e7b068bf52cd32dc226c013dd117@2026-10-22T21:17:43Z`; Gate3b base `2debbb0278cc08e018be2d0a1e75befcbda55454@2026-10-22T21:57:02Z`; Gate4 base 2debbb0278cc08e018be2d0a1e75befcbda55454@2026-10-22T22:26:45Z. User-authorized Agent spawns: designer, planner, executor, evaluator, all succeeded. Recipes follow the resolver's provider pins; no unavailable role silently substituted with the controller. Literal driver note:
controller claude:claude-opus-5-5 → codex:gpt-6.1-sol (probe ok); lanes degraded: ; `designer`: **anthropic** lane `claude-opus-5-5` unusable (probe rc=`1` — quota-limited) → handed to `codex:gpt-6.1-sol`; `evaluator`: **anthropic** lane `sonnet` unusable (probe rc=`1` — quota-limited) → handed to `pi:openrouter/minimax/minimax-m3`
- **Controller** `codex:gpt-6.1-sol` (tok: not reported); native Agent dispatch-carrier tokens also not reported.
- **Designer** `codex:gpt-6.1-sol`, provider recipe via Agent carrier: successful probe 16,183 tok + corrected run 118,036 tok; prior interrupted wrong-prefix attempt tok not reported, not counted as a completed result.
- **Planner** `codex:gpt-6.1-sol`, provider recipe via Agent carrier: probe 23,351 + run 48,483 tok; lane `declared:planner-lane-default-pin`, absolute existing plan. GitHub-fed-queue import step inapplicable to gcp ticket queue.
- **Executor** `codex:gpt-6.1-sol`, provider recipe via Agent carrier: probe 22,515 + run 36,610 tok; test/inspection only, rc0.
- **Evaluator** `pi:openrouter/minimax/minimax-m3`, independent cross-vendor recipe via Agent carrier; canonical fenced runner rc0. Usage: input 76,876; cache-read 1,987,683; output 31,405 (reasoning 12,414 included by provider); total 2,095,964. Main $0.18000978 + probe $0.00018210 = **$0.18019188 recorded metered**. Small live driver probes are not individually itemized by the driver, so this is not claimed as an exhaustive invoice. Known Codex probe+run total **265,178 quota tokens**, plus unreported interrupted attempt/controller/carrier consumption. Every Codex launch used the spawn-override=false flag. No role changed model mid-run; no quota threshold was changed.

**Ruled out**:
- Reading a green parent as exact-base evidence, or reusing the old pre-rebase PR CI for candidate0be.
- Landing on the controller's verdict: generator and judge were distinct providers/models, with the judge's original independent report preserved.
- Disabling the scope guard or expanding fleet authority to force-push an upstream-containing diff. New branch, same candidate, unchanged guard.
- Resolving all three banked tickets in one iteration, closing D-FLEET-17 without Mark, or treating the 14 unchanged policy asks as new decisions.
- Installing/reloading a shared binary or driver mid-fire; broadening the advisory suffix checker beyond the approved heading fix; inventing effort metrics to make stale sprint state look measured.

**Retro lane**:
- **Backlog / no new skill edit.** The first scope-guard false positive on a rebased existing remote branch is recorded with its safe publication workaround. The wrong role prefix, template-log corpus sources and wrong ldflags path were caught by existing role/positive-control/provenance gates, so they do not justify duplicating those rules.
- Judge arithmetic and stale-label claims were checked against raw artifacts; corrections are separate from the immutable independent report. Existing verification discipline covers them.
- D6 plan scope is explicitly clarified for inherited, separately authorized Gate-5 changes and mandatory records; no implementation expansion. The sprint is completed only after D7, marked implemented on dev/unreleased at its stable path; no release/version is invented.
- The first-party final share check refuted the draft 20/20 estimate: iteration40 is ADMIN, so the actual newest20 share is19/20 HARNESS. Joined newest index identifiers to full source headings because truncated index text can omit class tags; see `harness-share.json`. No new skill edit on one observed instrument instance.
- No new `docs/sprint-retros/` material since the preceding record (latest path commit 2026-10-16). A one-off unquoted zsh search glob repeated iteration57’s known class; corrected with `rg --files`, without broadening this ticket.
- Last 3 code landings: iteration52 quota-bucket repair→no proven movement of an UNMET clause (19.4d turnaround; clause5 already MET); iteration53 runtime-model allowlist→no proven movement of an UNMET clause (about19d; clause5 already MET); iteration58 heading repair→clause2 qualifying <48h closure. The overall median was not recomputed or declared MET. No all-none drift alarm. Clause1 remains UNMEASURED. No routing-policy change and no new human ask.

**Next**: refresh and independently validate banked #1929 (`stall:watchdog:foreground-agent-and-wrong-transcript`) on current dev, then #1932 (`quorum-reader:zsh-nomatch`), in separate iterations. Rebase old-base candidates under Gate3b even when changed paths are disjoint; fresh exact-head CI and exact-merge CI, then resolve only the landed ticket. Then reality-check `quorum:oc-glm-5-3-output-cap-invalid` (likely outside-scope decision, not a second item this iteration). Keep D-FLEET-16 through -29 OPEN pending Mark.

## 59 — 2026-10-23 — watchdog fix #1937 recovered after the post-merge capacity stop; independent MiniMax PASS94/100; ticket resolved [HARNESS]

**Pick**: recover `stall:watchdog:foreground-agent-and-wrong-transcript`, the next banked landing after iteration58. No new source implementation and no second ticket. Gate2's orphan traces found that the morning fire had already refreshed and merged the fix, but stopped before its ticket, sprint state and record were closed. The controlling design/plan remains `design_docs/planned/sprint-plan-stall-own-session-transcript.md`; bookkeeping recovery needs no new design or quorum.

**Orphans credited**: iteration59's original fire stopped `PAUSED-NO-CAPACITY_at=gate-3` at2026-10-23T06:33:41Z, 35seconds after PR1937 merged. Its bounded merge-CI poll nevertheless completed at06:59:07Z. The noon fire stopped on capacity before work. These are recovered evidence, not new implementation. Gate0's 2026-10-17 crash notice was already credited in iteration47 and is not credited twice.

**Preflight / observe**: armed, correct GitHub identity, billing CLEAN, canonical messaging; no new human directives since2026-10-22T23:29:08Z. No watermark advance. The source clone's unrelated dirty files and active shadow-sprint banner were left alone. Pin tree was clean. Both running and pin copies of SKILL.md plus all resources matched origin; no drift. Latest dev13092528c had CI39983989594 successful and21 successful checks. Own open PRs1929/1932 and related worktrees were inspected before picking. The charter's29 decision rows,14 OPEN, are unchanged. Nine unrelated approvals were not acted on.

**Reality check**: original candidate `da3726052d3a6eb34e6245baffdb99ac49eb6810` and merge `6c16b4ce8c961e2c3afc9cb4ba766d09cde4496f` are recorded by PR1937, merged2026-10-23T06:33:06Z. Merge parents are base763d336f1 and reviewed candidate; candidate/merge trees match. All3 refreshed patches compare `=` to iteration55; all5 original path blobs match. Driver, tests, plan and sprint JSON remain blob-identical at recovery base13092528c. The fragment's deletion is release329c9808f consolidation into the v0.54.18 changelog, not lost documentation. Explicit blob evidence: [source-identity.json](fleet-mission-evidence/iteration59/source-identity.json).

**Did**:
- Spawned all four Agent-tool roles under the operator's explicit request; each carrier dispatched the resolver's pinned provider recipe. No role was silently replaced by the controller. Designer and planner confirmed the existing contract and defined recovery; executor produced a conditional metadata patch without touching repository source. Stable plan paths preserve JSON and released-changelog links. Original implementation churn is272 driver/test additions+deletions (44+15+202+11); original full PR391+/26−. Historical active start, effort and velocity remain unmeasured.
- Preserved the original independent MiniMax report verbatim, SHA256 `2ecd4c2d1a293805bd65b7558a4718ff995f217139b22f2b26039bf5549174ab`, at [original-evaluation.md](fleet-mission-evidence/iteration59/original-evaluation.md). Its stated CODE_ACCEPT90 and CI_PENDING snapshot are unchanged. The rubric rows total100, its narrated deduction implies95, and neither is its stated90; the controller does not rescore it. Its failed-CI attribution was wrong: raw job lines show watchdog50/0 and the unrelated motoko timer refusal failing. Its blanket file-size and survivor/fallback claims are also narrowed in [controller-corrections.md](fleet-mission-evidence/iteration59/controller-corrections.md).
- Fresh independent evaluator `pi:openrouter/minimax/minimax-m3`: CODE_ACCEPT,84/100 at the pre-completion snapshot, then independent RECOVERY_ACCEPT94/100 after verifying C1–C7; neither score was assigned by the controller. Full original report, fresh report, role proposals, observed checks, dry-runs and usage are banked in [iteration59](fleet-mission-evidence/iteration59/). Review limits remain visible; no controller verdict stands in for the judge. The a7 helper-behaviour/call-before-launch implementation is an explicit reviewed deviation from the old direct-assignment-grep wording.
- Controller fresh outside-sandbox darwin/arm64 stall50/0; syntaxboth rc0; full `env -i make test-launchd-drivers` rc0, final “launchd drivers: tests + bash 3.2 syntax OK” (complete log banked). Scratch CLI built from recovery13092528c with explicit version/commit ldflags, prepended to PATH; shared `~/go/bin` untouched. Canonical headings test rc0. Full unrelated compiler test/lint is supplied by exact-source CI, not redundantly rerun locally.
- Historical files called healthy/degraded both said DEGRADED4. Fresh per-command cloud pins under idle, armed stapledon and `AILANG_DRIVER_PINNED=13092528c...` produced healthy `lanes=ok`; deliberately invalid executor produced named `DEGRADED(1)`. Both rc0. No PID created, policy deployed, binary installed or mission reloaded. The initial fresh Codex-pin dry-run also degraded on ration and is not claimed healthy.
- Completion bookkeeping applied2026-10-23T18:59:04Z after acceptance/gates, independently confirmed before ticket resolution. Ticket resolve against6c16b4ce replied to world; exactly one signature removed,44→43 open. Superseded PR1929 received a separately verified comment (count0→1, [receipt](https://github.com/sunholo-data/ailang/pull/1929#issuecomment-6372593292)), then CLOSED verified; branch retained.

**Gate3b**: direct exact-merge observations, not the newest branch run: CI[39912181755](https://github.com/sunholo-data/ailang/actions/runs/39912181755) and Build/Release[39912181610](https://github.com/sunholo-data/ailang/actions/runs/39912181610), push/dev/attempt1, both completed SUCCESS. All17 expected merge checks successful. Exact candidate's final26 checks also terminal success/skipped; its earlier launchd red was the motoko timer arm, then green on rerun. Docs push is path-filtered out for the implementation's five paths; CodeQL/UI are PR-only and release is tag-only. The source fix already passes Gate3b; record publication is separately observable by its commit/PR.

**Progress**: goal unmoved on clause1 (product-loop rolling14-day share UNMEASURED). Clause2 UNMET overall; this non-policy ticket's 43.60h closure is a qualifying <48h turnaround, without claiming the aggregate median. Clause3 MET, 44→43 open tickets. Clause4 prior evidence only. Clause5 preserved by current done-gate. Fleet's newest20 entries are19/20 HARNESS (iteration40 ADMIN); this inverted remit is not a product-loop share measurement.

**Routing evidence**: Gate1 base `13092528c8ae406c78054f58846b00572edceeae@2026-10-23T18:35:52Z`; Gate4 base `13092528c8ae406c78054f58846b00572edceeae@2026-10-23T19:01:16Z`. Four native Agent carriers accepted, token counts not reported. Literal driver note:
controller pi:ollama/glm-5.3:cloud → codex:gpt-6.1-sol (probe ok); lanes degraded: ; `designer`: **anthropic** lane `claude-opus-5-5` unusable (probe rc=`1` — quota-limited) → handed to `codex:gpt-6.1-sol`; `evaluator`: **anthropic** lane `sonnet` unusable (probe rc=`1` — quota-limited) → handed to `pi:openrouter/minimax/minimax-m3`
- Controller `codex:gpt-6.1-sol` (tok:not reported).
- Designer `codex:gpt-6.1-sol`: probe23,351 +run122,835 tokens, rc0/0; no new design, rotation not advanced.
- Planner `codex:gpt-6.1-sol`: probe23,351 +run92,713 tokens, rc0/0; `declared:planner-lane-default-pin`. GitHub issue import inapplicable to fleet ticket queue.
- Executor `codex:gpt-6.1-sol`: probe15,716 +run139,069 tokens, rc0/0; conditional bookkeeping proposal, no source edits.
- Evaluator `pi:openrouter/minimax/minimax-m3`: main review2,656,033 total tokens ($0.24928362) +confirmation1,251,035 ($0.10473120); probes$0.00122292 combined; canonical fenced runner rc0 for both, own protocol ack true; 356s/57 tools and122s/44 tools respectively. Source HEAD unchanged; original review and recovery review remain independent sessions of the same MiniMax model, cross-vendor from OpenAI generator/proposal writer.
- Recorded recovery Codex quota417,035 tokens are derived in usage.json; controller/carrier consumption is not fabricated. Original interrupted-fire cloud designer/planner/executor had runner rc10 (read-only empty-worktree), not accepted implementation verdicts; original judge attempt1 failed to deliver a report, attempt2 supplied the preserved CODE_ACCEPT90. Original metered assistant usage is$1.27709784; current metered $0.35523774 (known original+recovery$1.63233558), within$5 total. Probe coverage is not an exhaustive invoice. All roles used actual provider pins and the OpenAI spawn-model-override=false flag; no lane thresholds changed.

**Ruled out**: redoing a merged fix; treating either historical degraded run as healthy; deriving a fresh verdict from the old report's exit code; repairing its prose by rewriting the independent report; calling the motoko timer CI red a watchdog regression; inventing historical effort or backdating recovery; resolving an operator decision or unrelated approval; taking zsh-safe helper ticket as a second item.

**Retro lane**: backlog observation only, no new skill edit or self-sourced ticket. Existing orphan/gate rules caught the post-merge capacity stop and incomplete records. The pi empty-worktree validator is unsuitable evidence for a read-only role; fresh provider reports avoid laundering those rc10s into success. The independent-report arithmetic/CI attribution issues are corrected separately. Latest new sprint retrospective (SMT parametric ADTs) reinforces inspecting actual emitted evidence and separate independent review; no fleet change follows from it. Last3 code landings: iteration53 none of the UNMET clauses; iteration58 clause2 <48h; iteration59 clause2 <48h. No all-none drift alarm. No routing-policy change and no new human ask.

**Next**: refresh banked PR1932 `quorum-reader:zsh-nomatch` against current dev, independent review, exact-head/exact-merge CI and done-gate, then resolve it. Afterward reality-check existing ticket `quorum:oc-glm-5-3-output-cap-invalid`; any outside-authority or policy fix parks for Mark. Keep ledger OPEN decisions pending the operator.

## 61 — 2026-10-26 — `quorum-reader:zsh-nomatch` LANDED through replacement #1944 → `b7dd07599`; independent sonnet CODE ACCEPT 94; ticket resolved; orphan iteration 60 credited [HARNESS]

**Pick**: land the banked `quorum-reader:zsh-nomatch` (charter queue #37; iteration 59's **Next**). Gate 2's died-mid-flight traces showed iteration 60 had started it: PR #1942, five `.wt-fleet-iter60-*` worktrees and an uncommitted record. That changed the pick from "refresh" to "verify and land". The routable UNMET clause is 2; no other routable row moves it faster.

**Orphans credited**: **iteration 60** (fire 2026-10-26 03:13→03:57 local, slot verdict `PAUSED-NO-CAPACITY at=gate-3`).
- It refreshed the iteration-57 build onto `bf22f4943` as #1942 (`dcab63a3c`; range-diff `=`, blobs identical).
- Its roles ran on pi recipes: designer glm-5.3 CONFIRMED, planner kimi-k3 ADDENDUM OK, executor deepseek-v4.1-flash PASS, evaluator MiniMax CODE ACCEPT 93.
- Done-gate: green.
- It parked on #1942's required `test` red. Four eval_harness Python tests could not download uv's Python 3.12 through CI's poison proxy on the new ubuntu-26.04 image.
- It died before committing. Its STATUS stamp, D-FLEET-30 (that CI red) and D-FLEET-31 (`quorum:oc-glm-5-3-output-cap-invalid`, reality-checked REAL) exist only in its uncommitted worktree. They are now banked verbatim in [iteration60](fleet-mission-evidence/iteration60/), including `uncommitted-status-stamp.diff`.
- Its CI diagnosis was right when it was made. #1940 (`d8e47684c`, 07:56Z) fixed the cause upstream with a `uv python install 3.12` prefetch, so D-FLEET-30 is moot and was not filed.
- D-FLEET-31 is an unverified orphan claim with no evidence file. It is carried to **Next** rather than filed on its author's word.
- The 2026-10-17 crash notice was already credited at iteration 47.

**Preflight / observe**:
- Armed; gh `sunholo-voight-kampff`; billing CLEAN; messaging store gcp.
- Directives: 0 on #1891 since `2026-10-22T23:29:08Z` (28 comments, none allowlisted). No watermark advance.
- Self-notices rc 1: the single 10-17 notice, already credited.
- Running skill and all 13 resource files `cmp`-identical to origin/dev; the pin and the resolved copy are the same file.
- Gate-1 base `c8e348d08736e6b273efb73175ae706ce82eef62@2026-10-26T08:58:55Z`.
- Dev CI: the last `CI` run, at `d8e47684c`, was red only on non-required `test-windows` (`TestShadowGradeHangIsUngradedTimeout`). HEAD `c8e348d08` had **0** runs: it was a Dependabot auto-merge by `github-actions`, and pushes made with that token trigger no workflows. Gate 1's true-zero rule applied: dispatch [40296349998](https://github.com/sunholo-data/ailang/actions/runs/40296349998) came back all green, including `test-windows`, so that red reads as a flake. This fleet does not own the repo; it is recorded, not acted on.
- Ledger: 29 rows, 14 OPEN, unchanged. Nine unrelated approvals untouched.

**Reality check**:
- Iteration 60's park premise was re-measured. Dev's `test` job 121459024865 at `d8e47684c` ran on `Image: ubuntu-26.04`. It shows `Installed Python 3.12.15` and PASS for `TestPythonRunner` and all three `TestRunPythonSolution_*` tests. So the red that parked #1942 is gone, by #1940's prefetch, not by a flake. Branch protection requires `test, lint, build, docs-gate, UI build gate`, with strict=false.
- No file overlap: zero files are shared between dev's two new commits and the 7 branch paths (planner and designer each measured it, with a positive control).
- Trial merge: 0 conflicts.

**Did** (all four roles spawned; generator ≠ judge):
- Designer `claude:claude-opus-5-5` via the `claude-sub` recipe (resolver `recipe … declared:provider-pin`). The Agent tool is not valid for a provider pin, and the spawn hook denies the alias. Verdict CONFIRMED: no new design, and the contract holds at `c8e348d08`. Rotation pointer not advanced, since no doc was authored.
- Planner, Agent tool `opus` (resolver `agent-tool opus fail-closed:env-pin`). Verdict ADDENDUM OK, with an ordered landing plan. It independently re-verified #1940's prefetch in `ci.yml` (`test`, `test-windows`) and in `build.yml`.
- Executor `claude:claude-haiku-5-5` via `claude-sub`. Cherry-picked `bd9a41cf3` and `dcab63a3c` onto `c8e348d08` as `fleet/i61-zsh-libs-land` (`cd717c328`, `b2c9ff5b0`).
  - range-diff `=`/`=`; all 7 blobs IDENTICAL to `dcab63a3c`.
  - `test_quorum_artifacts.sh` rc 0, with all 5 zsh arms passing.
  - `test_pi_ext_args.sh` 17/0.
  - `check-changelog` and `diff --check` rc 0. Pushed by the controller, not the executor.
- Evaluator, Agent tool `sonnet`: **CODE ACCEPT 94/100**, 0 blocking findings.
  - 4/4 mutants killed: setopt removed (quorum, pi) and `local_options` dropped (quorum, pi).
  - A direct `/bin/zsh -f` check: rc 1 with no `no matches found`, and caller options preserved.
  - The return was suspiciously fast (32 s, 6 tool calls), so it was checked against disk before acceptance. The mutant outputs exist, show the real kills (`mc_quorum_lookup:11: no matches found`, `OPTIONS_LEAKED` rc 81), and the judge tree is clean.
- Controller done-gate at `b2c9ff5b0`:
  - Outside-sandbox `env -i … make test-launchd-drivers`: rc 0, 1063 PASS lines, 0 FAIL, final `launchd drivers: tests + bash 3.2 syntax OK`.
  - Dry-runs under idle, armed stapledon with `AILANG_DRIVER_PINNED=b2c9ff5b0` (no pidfile before or after). The first "healthy" arm read `DEGRADED(2)` only because the codex bucket is over its daily ration (the same fact as this fire's own routing note), so it is not claimed healthy. With planner and executor pinned off codex the healthy arm read `lanes=ok`, and the degraded arm (`codex:bogus`) read a named `DEGRADED` with in-order fallback to haiku.
- Opened [#1944](https://github.com/sunholo-data/ailang/pull/1944), which replaces #1942 and #1932. The closing-keyword scan of its title and body found 0 (control fired). Merged with `--merge --match-head-commit b2c9ff5b0` → `b7dd0759969ff47b5776524a31487a91c5dc8bd8`.
- Weekly external-issue sweep (rotation Monday): 61 issues enumerated, matching the search total. 57 have zero fleet mentions; positive control `#1306`=11; negative control fired. 25 are new since 10-05, all outside Authority. 0 queue items. Table: [issue-sweep.tsv](fleet-mission-evidence/iteration61/issue-sweep.tsv). The first pass was an instrument failure (zsh did not word-split the file list, so every count read 0); the positive control caught it, and the sweep was redone under `/bin/bash`.

- Weekly thread rotation (Monday, after 07:00 local; #1891 was created 10-19): opened [#1945](https://github.com/sunholo-data/ailang/issues/1945) and posted the weekly report (comment count 0→1). Posted the pointer on #1891 (28→29) and CLOSED it. `mission-fleet-gh-issue`=1945, `-prev`=1891.

**Gate 3b**:
- Exact PR head `b2c9ff5b0`: 26/26 checks terminal, 0 non-green, all 5 required green; MERGEABLE/CLEAN at merge time.
- Exact merge `b7dd07599`: CI [40302410292](https://github.com/sunholo-data/ailang/actions/runs/40302410292) and Build and Release [40302410031](https://github.com/sunholo-data/ailang/actions/runs/40302410031) — both push/dev attempt 1, completed SUCCESS. The 17-row check set is terminal. The one non-green row is `SonarCloud Code Analysis`, which is not required and is inherited: it is red on `d8e47684c` (#1940) and green on `bf22f4943` and on this PR head. Ticket resolved after this, replying to stapledon (43→42 open). #1942 and #1932 each received a comment (0→1, verified) and were then CLOSED; their branches are retained.

**Progress**: goal unmoved on clause 1 (UNMEASURED). Clause 2 UNMET overall. This ticket, filed `2026-10-22T11:50:43Z`, closed in about 94h: outside 48h, because two landing attempts were blocked by required reds outside fleet authority. Clause 3 MET, 43→42 open. Clause 4 prior evidence only. Clause 5 preserved: judged, done-gate green, exact-merge CI green.

**Routing evidence**: Gate-1 base `c8e348d08736e6b273efb73175ae706ce82eef62@2026-10-26T08:58:55Z`; Gate-4 base `b7dd0759969ff47b5776524a31487a91c5dc8bd8@2026-10-26T10:12:49Z`. Literal driver note:
controller pi:ollama/glm-5.3:cloud → claude:claude-opus-5-5 (probe ok); lanes degraded: ; `planner`: **codex** lane `gpt-6.1-sol` unusable (probe rc=`75` — over daily ration) → handed to `opus`; `executor`: **codex** lane `gpt-6.1-sol` unusable (probe rc=`75` — over daily ration) → handed to `claude:claude-haiku-5-5`
- Controller `claude:claude-opus-5-5` (tok: not reported).
- Designer `claude:claude-opus-5-5`, `claude-sub` recipe: probe rc 0, then a run of 6 turns / 70 s. Tokens: in 12, cache-create 60,763, cache-read 311,997, out 6,041. Subscription; metered $0; the CLI's reported $0.67 is list-price imputation, not billing.
- Planner `opus`, Agent tool: 90,287 subagent tokens, 25 tool uses, 197 s.
- Executor `claude:claude-haiku-5-5`, `claude-sub` recipe: probe rc 0, then a run of 8 turns / 30 s. Tokens: in 16, cache-create 45,670, cache-read 392,594, out 4,528. Subscription; metered $0.
- Evaluator `sonnet`, Agent tool: 51,721 subagent tokens, 6 tool uses, 33 s.
- Metered spend this iteration: $0. No lane thresholds changed; no lane recorded dead.

**Ruled out**:
- Rerunning #1942's failed jobs: a rerun reuses the old merge ref, which predates #1940.
- Force-pushing the old branch: iteration 58's scope-guard precedent.
- Re-judging on iteration 60's MiniMax verdict alone: the head changed, so a fresh judge was required.
- Filing D-FLEET-30: moot. Filing D-FLEET-31 on the orphan's word: unverified, carried to Next.
- Claiming the codex-rationed dry-run as healthy.
- Acting on the dev `test-windows` red: not owned, not required, and green on dispatch.

**Retro lane**: backlog observation only; no skill edit, because no gap has two recorded frictions this iteration. Observations, not tickets (the fleet does not self-source):
- (a) The driver's dry-run reports `lanes=ok` for a bogus pi executor model (`pi:ollama/bogus-nonexistent-model:cloud`), so the degraded arm can only be shown through a codex lane. The driver is unchanged by this patch.
- (b) A bare Dependabot auto-merge leaves dev HEAD with no CI run by construction (`github-actions` token). Gate 1's dispatch rule handled it.
- (c) This controller's zsh tool shell broke a hand-written loop twice: an unsplit `$F`, and an unsplit env-prefix variable (rc 127). Both were caught by controls; standing memory already says to run hand-written loops under `/bin/bash`.
Last 3 code landings: iteration 58 clause 2 (<48h); iteration 59 clause 2 (<48h); iteration 61 clause 2 (outside 48h). No all-none drift alarm.

**Next**:
1. Re-verify iteration 60's `quorum:oc-glm-5-3-output-cap-invalid` reality check first-party (its claim: both stapledon artifacts show the `oc-glm-5-3` seat at exactly 65,536 output tokens with no verdict, and the fix is a `models.yml` quota/routing change outside fleet authority). If it is REAL, file it as a decision row with a recommendation.
2. Then take the next ticket by `slots_lost`.

## 62 — 2026-10-27 — `quorum:oc-glm-5-3-output-cap-invalid`: diagnosis half LANDED via #1966 → `0802749cb`, independent sonnet PASS 93; policy half filed as D-FLEET-31; ticket stays open [HARNESS]

**Pick**: iteration 61's **Next** 1 was to re-verify iteration 60's unrecorded reality check of `quorum:oc-glm-5-3-output-cap-invalid` and file its decision row if REAL. Clause 2 is the routable UNMET clause. No open ticket was filed in the last 48h (the newest is 10-22), so no pick can close inside 48h; clause 2 does not change the pick.

**Orphans**: none. Slot verdicts show iteration 61 as the last fire (`COMPLETED` at 2026-10-26T11:09:54Z). No `iter62` worktree, PR or uncommitted record existed. Self-notices rc 0 (control 852:10).

**Preflight / observe**:
- Armed (`mission-fleet.disabled` absent). gh `sunholo-voight-kampff`. Billing CLEAN. Messaging store gcp.
- Directives: 0 on #1945 since `2026-10-22T23:29:08Z` (2 comments, none allowlisted). The watermark was not advanced.
- Running skill and all 13 resource files are `cmp`-identical to origin/dev.
- Gate-1 base `e07e2863c75d567f2a9d0388ae27f92332b43d27@2026-10-27T08:02:10Z`.
- Dev CI at HEAD `e07e2863c` failed only in the non-required `test-windows` (`TestCompileCache_DirtyBuildsDoNotShareVerdicts/AILANG_NO_CACHE_is_honored_by_check`, 180 s; job 121667637668, log 157 KB). `SonarCloud` is also red there; its check set has 17 rows. The fleet does not own the repo, so this is recorded only.
- Ledger: 29 rows, 14 OPEN, unchanged before this iteration. 42 tickets open.

**Reality check** (first-party, before any routing):
- Both stapledon artifacts show `oc-glm-5-3` `present=false` with `tokens_out` exactly 65536, an empty body, and the error `unexpected end of JSON input at byte 0 of 0`.
- Across all banked quorum artifacts: 2 of 23 `oc-glm-5-3` seatings hit the cap, both on kerr-black-holes on 10-21. A third run on the same doc succeeded at 56,487 tokens. With this iteration's two rounds the count is 25; table in [oc-glm-5-3-seatings.tsv](fleet-mission-evidence/iteration62/oc-glm-5-3-seatings.tsv).
- **New fact that iteration 60 missed: part of the fix is inside fleet scope.** `runReviewerWith` already explained a truncation, but only on `FinishReason == "length"`. The ollama legacy `Generate` (`internal/ai/ollama/client.go:120`, the path `handlerCaller.CallJSON` uses) never sets `FinishReason`. So for ollama seats that branch was dead.
- The waste-reduction half (cap, reasoning budget, re-seating) is policy, and the cap lives in `models.yml`, outside fleet scope. It became D-FLEET-31.

**Did** (all four roles spawned; generator ≠ judge):
- **Designer**, `claude:claude-opus-5-5` via `claude-sub` (resolver `recipe … declared:provider-pin`). Probe rc 0. It wrote [m-quorum-reviewer-cap-hit-loud.md](planned/m-quorum-reviewer-cap-hit-loud.md).
- **Quorum round 1** (author's vendor sat out; `gpt6-1-sol` absent, OpenAI 429) was BLOCKED. kimi and glm both rejected: the premise that `FinishReason == ""` reaches the reviewer had not been traced.
  - The controller traced it independently: `call.go:43` → `factory` → `ollama.Client.Generate`, with no `FinishReason`, and `OutputTokens` from `Metrics.EvalCount`.
  - One designer revision added a seven-command premise trace and fixed gemini's note about "consumed" wording.
- **Quorum round 2** was BLOCKED by one reject. kimi argued that `out.Model` might be the provider's `req.Model`. glm passed, with the same catch and a `capHit` tail fix.
  - The trace refuted kimi's premise: `out` is the `*ReviewerOutcome` built at `run.go:112` from the models.yml id.
  - The controller made a bounded second revision under the narrow-refinement carve-out. It applied kimi's trace request and glm's two fixes verbatim. It declined kimi's replacement sentence, whose premise the trace refutes. This is recorded in the doc's Quorum verification log.
- **Planner**, `codex:gpt-6.1-sol`. With no doc, the resolver said `agent-tool opus fail-closed:no-doc`; with the doc it said `recipe codex:gpt-6.1-sol declared:planner-lane-default-pin`, and that pin was followed. Probe rc 0.
  - It used a detached sibling worktree `.planner-wt-iter62`, since removed. It produced the [sprint plan](planned/sprint-plan-quorum-reviewer-cap-hit-loud.md) and sprint JSON (`jq -e` ok, no placeholders).
  - It ran every acceptance command on the base; all green, re-run by the controller outside the sandbox. Skipped `messages import-github` (ticket-queue mission).
- **Executor**, `codex:gpt-6.1-sol`. It added `capHit` and `capLimitClause` in `run.go`, plus `cap_hit_test.go` (9 rows) and a changelog fragment. It killed 10/10 plan mutants via scratch-copy restore (`cmp` 0) and made no git writes. The controller committed `56e1a2055`.
- **Evaluator**, Agent tool `sonnet`: **PASS 93/100**, 0 blocking, 9/9 of its own mutants killed (7 not on the plan's list). Report in [evaluation.md](fleet-mission-evidence/iteration62/evaluation.md).
  - It returned in 107 s with 5 tool uses, so the controller checked it on disk. Its `/tmp/run.go.orig` is byte-identical to the judge tree. Its mutant M1 (`>=` to `>`), re-applied by the controller, reddens `CapHit/a`. The judge tree is clean.
- **Done-gate**, outside the sandbox at `56e1a2055`: `env -i … make test-launchd-drivers` rc 0, 1063 PASS, 0 FAIL.
  - Dry-runs used the idle, armed `triage` profile, with no pidfile before or after: healthy `lanes=ok`; degraded (`codex:bogus-model-i62`) `lanes=DEGRADED(1)` with a named fallback to haiku. Details in [done-gate.md](fleet-mission-evidence/iteration62/done-gate.md).
- **PR** [#1966](https://github.com/sunholo-data/ailang/pull/1966). The closing-keyword scan found 0; the control fired.

**Gate 3b**:
- Exact PR head `56e1a2055`: 26/26 checks terminal, 0 non-green, MERGEABLE/CLEAN. Merged with `--merge --match-head-commit` → `0802749cba31b5394bcfa8a80ecf48e2ca46b0f8`.
- Exact merge: CI [40480255296](https://github.com/sunholo-data/ailang/actions/runs/40480255296), Build and Release [40480255142](https://github.com/sunholo-data/ailang/actions/runs/40480255142) and Docs Deploy [40480255239](https://github.com/sunholo-data/ailang/actions/runs/40480255239), all push/dev attempt 1, completed SUCCESS. The 30-min poll cap expired first, with only CI `build` pending (it started 09:51Z, after `test`; the D-FLEET-21 class). A direct read after the expiry showed it `completed success` (8/8 steps), so this is OBSERVED green, not inferred. The 21-row check set is terminal. Its one non-green row is `SonarCloud Code Analysis`, which is not required and is inherited: red on base `e07e2863c`, green on this PR head.

**Decision filed**: D-FLEET-31, recommendation A (no further change). The ticket stays OPEN: its ask (stop the zero-signal spend) is policy and is not answered by the diagnosis fix, so resolving it would overstate the landing. Reach: this is a Go fix, so it reaches the loops only when `~/go/bin/ailang` is rebuilt (D-FLEET-29).

**Progress**: goal unmoved on clause 1 (UNMEASURED). Clause 2 UNMET; nothing closed. Clause 3 MET (42 open, unchanged). Clause 4: prior evidence only. Clause 5 preserved: judged, done-gate green, exact-merge CI as recorded above.

**Routing evidence**: Gate-1 base `e07e2863c75d567f2a9d0388ae27f92332b43d27@2026-10-27T08:02:10Z`; Gate-4 base `0802749cba31b5394bcfa8a80ecf48e2ca46b0f8@2026-10-27T09:22:42Z`. Driver note: empty (`MISSION_ROUTING_NOTE=`); `MISSION_OVER_RATION` empty.
- Controller `claude:claude-opus-5-5` (tok: not reported).
- Designer `claude:claude-opus-5-5`, `claude-sub`:
  - Run 1: 22 turns / 373 s; in 44, cache-create 76,474, cache-read 1,482,675, out 13,954.
  - Revision: 16 turns / 130 s; cache-create 62,685, cache-read 706,263, out 13,265.
  - Subscription; metered $0. Rotation pointer not advanced: the resolver pinned the provider and did not rotate.
- Planner `codex:gpt-6.1-sol`: 72,035 tok.
- Executor `codex:gpt-6.1-sol`: 87,089 tok.
- Evaluator `sonnet`, Agent tool: 56,815 subagent tokens, 5 tool uses, 107 s.
- Quorum: round 1 $0.1456, round 2 $0.1893. Of that, the gemini seats (Vertex, metered) were $0.0250. The kimi and glm seats ($0.3099) are Ollama Cloud, flat-rate and imputed.
- Metered spend this iteration: about $0.03. No lane thresholds changed; no lane recorded dead.

**Ruled out**:
- Filing the decision on iteration 60's word alone: re-measured first.
- Treating the whole ticket as outside scope, as iteration 60's draft did: the diagnosis half is in `internal/mission/**`.
- Lowering the cap or adding a reasoning budget: restricting thinking is reserved to evals.
- Fixing `FinishReason` in `internal/ai/ollama`: outside scope, and a reviewer-side check also covers the next silent provider.
- Adding a new `AbsentReason`: it would fall through the gate-2 remedy table and lose salvage.
- Resolving the ticket.

**Retro lane**: backlog observation only; no skill edit (no gap has two recorded frictions this iteration). Observations, not tickets (the fleet does not self-source):
- (a) The codex 1-token probe from `/tmp` fails rc 1 on `Not inside a trusted directory`, not on capacity. Probe from the worktree, as the recipe's real run does.
- (b) Both quorum rounds blocked on premise-trace completeness, and the second blocker misread a variable. The carve-out handled it in one bounded edit.

Last 3 code landings: iteration 59, clause 2 (<48h); iteration 61, clause 2 (outside 48h); iteration 62, none (diagnosis only; the ticket stays open). No drift alarm: no routable UNMET-clause row is idle, because no ticket can close inside 48h.

**Next**:
1. Take the next open ticket by `slots_lost`, checking first whether its rulings are still pending. `agent-tool:sonnet-unavailable` (15) is D-FLEET-24, and `reporting:daneel-reports-not-declared-triage` (8) is D-FLEET-22, both OPEN. So pick the top ticket with no open decision, at HEAD.
2. If Mark answers D-FLEET-31: A means resolve the ticket citing `0802749cb`; B means build the seating rule.

## 63 — 2026-10-27 — dev's required-check red repaired: the sprint-evaluator `.agents` mirror re-synced via #1972 → `6094d2b33`, independent sonnet PASS 94; no ticket moved [HARNESS]

**Pick**: Gate 1 found dev's REQUIRED `test`, `launchd drivers (bash 3.2)` and `Build ubuntu-latest` red at HEAD `5f56181a1` and its parent `efcd7fe80`, and green at `3c4af5190`. A red outranks the queue only for the owning mission. V1 owns repo reds and has been disabled since 09-14, but this red sits in fleet territory: the `.agents` mission/sprint mirror, its sync script and its test are fleet scope (#1398). It also reddened the fleet's own done-gate (`make test-launchd-drivers`) and blocked every landing, including the clause-2 candidate below. That is the measured reason the pick is not the routable UNMET-clause-2 ticket `gate5:controlplane-assert-marks-read` (stapledon, filed 10-27 10:10Z).

**Orphans**: none. Slot verdicts show iteration 62 `COMPLETED` at 2026-10-27T10:21:59Z, and the log and charter on `origin/dev` both end at 62. Self-notices rc 0 (control 852:10).

**Preflight / observe**:
- Armed (`mission-fleet.disabled` absent). gh `sunholo-voight-kampff`. Billing CLEAN. Messaging store gcp.
- Directives: 0 on #1945 since `2026-10-22T23:29:08Z` (3 comments, none allowlisted). Watermark not advanced.
- The running skill and all 13 resource files are `cmp`-identical to origin/dev and to the pin copy.
- Gate-1 base `5f56181a1e0ffe97e9d72e3b2ebf5a1e95b058a7@2026-10-27T16:22:52Z`.
- Ledger: 30 rows, 15 OPEN, unchanged. 43 tickets open (new since 62: `gate5:controlplane-assert-marks-read`).

**Reality check** (first-party, before routing; detail in [ci-red-diagnosis.md](fleet-mission-evidence/iteration63/ci-red-diagnosis.md)):
- Job logs (ANSI stripped, sized): `launchd drivers` 163,991 B, 2 failing suites; `test` 95,521 B; `Build ubuntu-latest` 4.4 MB, 1 `--- FAIL` against 21,173 `--- PASS`. Every failure prints `STALE: .agents/skills/sprint-evaluator differs from .claude/skills/sprint-evaluator`.
- Cause: `efcd7fe80` (attended) moved the canonical evaluator skill to a final `EVALUATION_VERDICT:` line without running `tools/launchd/sync-agents-skills.sh`. Only `SKILL.md` differed; the other five mirrored skills were identical. The `.agents` copy still told codex and pi evaluators to emit `EVALUATION_RESULT`, which the coordinator parser does not read.
- Reproduced at base outside the sandbox: four acceptance commands rc 1 on the STALE line; `make check-changelog` rc 0. No open PR touched the file.

**Did** (designer skipped; planner, executor and evaluator spawned; generator ≠ judge):
- **Designer**: not spawned. The fix is a deterministic regeneration with no design surface (charter CURRENT GOAL: "design only when the fix warrants one").
- **Planner**: the resolver said `agent-tool opus fail-closed:no-doc`, and the spawn-pin hook DENIED the opus Agent spawn (`deny:provider-pin — planner is pinned to codex:gpt-6.1-sol`), as `gate-3-route.md` §1b warns. It was re-routed to the pin under the codex recipe. Probe rc 0. Detached sibling worktree `.planner-wt-iter63`, since removed. It wrote [the sprint plan](planned/sprint-plan-agents-mirror-sync-red.md): all six mirrors diffed, four acceptance commands baselined red, one mutation. Skipped `messages import-github` (ticket-queue mission).
- **Executor**, `codex:gpt-6.1-sol`. Run 1: codex-cli 0.164.0 `workspace-write` treats `<wt>/.agents/**` as read-only (`rsync … Operation not permitted`). The executor labelled it UNINFORMATIVE UNDER SANDBOX and stopped without a workaround, which was correct. Run 2 (same lane, plus `--add-dir <wt>/.agents`) synced the one file, wrote the changelog fragment, and killed the mutation (`--check` rc 1, suite rc 1; `cp` restore, `cmp` 0). No git writes. The controller committed `bfd312a8a`.
- **Evaluator**, Agent tool `sonnet`: **PASS 94/100**, 0 blocking ([evaluation.md](fleet-mission-evidence/iteration63/evaluation.md)). It read the three CI logs itself, confirmed the diff is byte-identical to the canonical file with no `.agents`-only content clobbered, and killed 2 mutants of its own (mirror reverted to base; a different mirrored skill drifted). It returned in 138 s with 7 tool uses, so the controller verified on disk: the report exists, the judge tree is clean, and mutant 1 replayed reddens `test_agents_skills_sync.sh` (rc 1, restored, `cmp` 0).
- **Done-gate**, outside the sandbox at `bfd312a8a` ([done-gate.md](fleet-mission-evidence/iteration63/done-gate.md)): `env -i … make test-launchd-drivers` rc 0, 1113 PASS lines, 0 FAIL, 49 suites. Dry-runs under the idle, armed `triage` profile (no pidfile before or after): healthy `lanes=ok`; degraded (`codex:bogus-model-i63`) `lanes=DEGRADED(1)`, with a named fallback to haiku.
- **PR** [#1972](https://github.com/sunholo-data/ailang/pull/1972). The closing-keyword scan found 0; the control fired.

**Gate 3b**:
- Exact PR head `bfd312a8a`: 27/27 checks terminal, 0 non-green; MERGEABLE/CLEAN; dev unmoved since Gate 1. Merged with `--merge --match-head-commit` → `6094d2b3318fa6a4cfdac1c5fd10eec2ca383f27`.
- Exact merge: Build and Release [40545301470](https://github.com/sunholo-data/ailang/actions/runs/40545301470) completed SUCCESS (attempt 1). CI [40545301442](https://github.com/sunholo-data/ailang/actions/runs/40545301442) attempt 1 failed only in the non-required `test-windows` job (`TestShadowGradeHangIsUngradedTimeout`, `verdict="ungraded" reason="primary: clone_failed"`, 22.35 s; job 122334401696, log 157 KB; `cmd/ailang` shadow-grade, the active `M-AILANG-LANE-EXECUTOR-SHADOW` sprint's code, not this diff). Negative control: the merge tree `b8cad3ab9` is identical to PR head `bfd312a8a`'s, whose `test-windows` passed, as did `5f56181a1`, `efcd7fe80` and `3c4af5190`. Re-running only the failed job: attempt 2 (job 122342514449) completed SUCCESS. The 18-row check set reads 0 non-green; required contexts (`test, lint, build, docs-gate, UI build gate, checks`) green. Recorded as a one-off Windows timing flake, not filed (outside fleet scope; the fleet does not self-source).

**Progress**: goal unmoved on clause 1 (UNMEASURED). Clause 2 UNMET; nothing closed (`gate5:controlplane-assert-marks-read`, filed 10-27 10:10Z, is the open in-window candidate). Clause 3 MET (43 open). Clause 4: prior evidence only. Clause 5 preserved and restored: the done-gate suite was red on dev before this landing.

**Routing evidence**: Gate-1 base `5f56181a1e0ffe97e9d72e3b2ebf5a1e95b058a7@2026-10-27T16:22:52Z`; Gate-3b base `6094d2b3318fa6a4cfdac1c5fd10eec2ca383f27@2026-10-27T16:53:29Z`. Driver note: empty (`MISSION_ROUTING_NOTE=`); `MISSION_OVER_RATION` empty.
- Controller `claude:claude-opus-5-5` (tok: not reported).
- Designer: not spawned (mechanical; no doc).
- Planner `codex:gpt-6.1-sol`: 46,122 tok. The resolver said `agent-tool opus fail-closed:no-doc` and the hook said `deny:provider-pin`; the pin was followed.
- Executor `codex:gpt-6.1-sol`: run 1 44,972 tok (sandbox-refused), run 2 42,105 tok.
- Evaluator `sonnet`, Agent tool: 50,879 subagent tokens, 7 tool uses, 138 s. Judge independence: cross-vendor.
- Quorum: none (no design doc). Metered spend this iteration: $0. No lane thresholds changed; no lane recorded dead (the run-1 refusal is a path capability limit, not a dead lane).

**Ruled out**:
- Handing the red to V1: V1 is disabled, and the surface is fleet scope.
- A hand edit of the `.agents` file: the sync script is the only sanctioned writer.
- Falling to the pi executor rung after run 1: codex worked and refused only one path, so one bounded retry with `--add-dir` measured whether the protection is overridable. It is.
- A prevention hook (pre-commit sync): self-sourced; listed below as an observation.
- Resolving any ticket: none was filed for this red.

**Retro lane**: backlog observation only; no skill edit (no gap has two recorded frictions this iteration). Observations, not tickets (the fleet does not self-source):
- (a) codex-cli 0.164.0 `workspace-write` protects `<wt>/.agents/**`, so any codex executor directive whose diff includes an `.agents` mirror needs `--add-dir <wt>/.agents`. Not yet measured: whether the protection also covers `.codex`.
- (b) An attended `.claude/skills/{mission,sprint}-*` edit without the sync reddens three required checks. Nothing local catches it before push.

Last 3 code landings: iteration 61, clause 2 (outside 48h); iteration 62, none (diagnosis only); iteration 63, none (a dev red, not a ticket; it restored clause 5's done-gate). No drift alarm: the in-window clause-2 row was blocked by this red until the merge.

**Next**:
1. `gate5:controlplane-assert-marks-read` (stapledon, filed 2026-10-27T10:10Z, 1 slot): its 48h window closes 2026-10-29T10:10Z. The fix is in `gate-5-retro.md` rule 3(c), both copies (a non-mutating `list --json` read, or `unack` after the read). This iteration's own Gate 5 used a non-mutating read.
2. Otherwise, the top ticket by `slots_lost` that no OPEN decision gates.

## 64 — 2026-10-28 — `gate5:controlplane-assert-marks-read` LANDED via #1975 → `8f8294c4e`; independent sonnet PASS 91 (round 2); ticket resolved in ~14h [HARNESS]

**Pick**: the queue head named by iteration 63's **Next**. `gate5:controlplane-assert-marks-read` (stapledon iteration 52, filed 2026-10-27T10:10:25Z, 1 slot, blocking none) was the only routable ticket inside clause 2's 48h window, and no OPEN decision gates it. Mechanical (a wrong command in a skill rule), so not policy class.

**Orphans**: none. No `fleet/i64*` branch, worktree or PR existed. Self-notices rc 0 (control 852:10). The log and charter on `origin/dev` both ended at 63.

**Preflight / observe**:
- Armed (`mission-fleet.disabled` absent). gh `sunholo-voight-kampff`. Billing CLEAN. Messaging store gcp.
- Directives: 0 on #1945 since `2026-10-22T23:29:08Z` (4 comments, none allowlisted). Watermark not advanced.
- Skill drift: the running skill and all 13 resource files `cmp`-identical to origin/dev in both the resolved symlink and the pin copy.
- Gate-1 base `9646caebd0f7f43d1e3d6198b897b795ce6ae1ee@2026-10-27T23:37:40Z`. Dev CI and Build and Release green; check set 18, 0 non-green.
- Ledger: 30 rows, 15 OPEN, unchanged. 43 tickets open.

**Reality check** (first-party, before routing; [done-gate.md](fleet-mission-evidence/iteration64/done-gate.md)):
- `runMessagesRead` (`cmd/ailang/messages_crud.go`) already has `--peek`, which guards `MarkInboxMessageRead` on every path. No CLI change was needed.
- On the prod store, `ailang messages read --peek --json <id>` on an unread controlplane message left it unread (re-listed under `--unread`).
- `read --peek <id> --json` printed human output: the trailing flag was ignored, because Go `flag` stops at the first positional. So `read <id> --peek` marks the message read anyway.

**Did** (designer skipped; planner, executor and evaluator spawned; generator ≠ judge):
- **Designer**: not spawned. A one-clause skill fix has no design surface (charter CURRENT GOAL: "design only when the fix warrants one").
- **Planner**: the resolver said `agent-tool opus fail-closed:no-doc`, and the role is pinned to `codex:gpt-6.1-sol`. Per `role-spawn-routing.md` §2(a) it went straight to the pin under the codex recipe, with no alias spawn burned on a guaranteed denial. Probe rc 0. Detached sibling worktree `.planner-wt-iter64`, since removed. It wrote [the plan](planned/sprint-plan-controlplane-assert-peek.md): one milestone, an exact-command observation arm, and two mutations. Skipped `messages import-github` (ticket-queue mission).
- **Executor**, `codex:gpt-6.1-sol` (with `--add-dir <wt>/.agents`, per iteration 63's observation (a)). It rewrote rule 3(c) and regenerated the mirror with `sync-agents-skills.sh`. It added the awk arm to `test_agents_skills_sync.sh` and wrote the changelog fragment. 5 gates rc 0; mutations (no `--peek`, and `--peek` after the id) both rc 1, each restored with `cp`+`cmp`. No git writes. The controller committed `92a5420e2`.
- **Evaluator**, Agent tool `sonnet`, round 1: **PASS 86**, 0 blocking ([evaluation.md](fleet-mission-evidence/iteration64/evaluation.md)). It confirmed both claims from code plus a standalone `flag.NewFlagSet` program, and ran 8 mutants of its own. Finding 1: BSD awk 20200816 `split()` also breaks on newline, so a second inline `read` on the clause's wrapped line survived. The controller reproduced it (`split("a`b\nc`d",…)` returns 4). It fixed the arm in `3f320eac4` (`gsub(/\n/," ",clause)`) and ran a negative control: the mutant passes the pre-fix guard (rc 0) and fails the fixed one (rc 1). Round 2, by the same judge resumed by name in a fresh judge worktree: **PASS 91**. All survivors were killed, no regression, and wrapping inside the backticks still fails loudly rather than passing silently. The controller checked the on-disk report (Round 2 section present) and that both judge worktrees were removed.
- **Done-gate**, out of sandbox: `make test-launchd-drivers` rc 0 at `92a5420e2`, and again at `3f320eac4` (1114 PASS, 0 FAIL, 49 suites). Dry-runs under `triage` were rc 0 but exited at triage's idle short-circuit BEFORE the lane probe, so they are not counted. Fleet dry-runs under a synthetic HOME, `AILANG_DRIVER_PINNED=3f320eac4`: healthy `lanes=ok`; degraded (`codex:bogus-model-i64`) `lanes=DEGRADED(1)`, with a named fallback to `claude:claude-haiku-5-5`. The real fleet pidfile was unchanged.
- **PR** [#1975](https://github.com/sunholo-data/ailang/pull/1975). The closing-keyword scan over the commits and the body found 0; the control fired.

**Gate 3b**:
- Exact PR head `3f320eac4`: 27 checks terminal, 26 green, 1 red: the non-required `govulncheck (vuln gate)`. MERGEABLE/UNSTABLE; required contexts (`test, lint, build, docs-gate, UI build gate, checks`) green; dev unmoved since Gate 1. Merged with `--merge --match-head-commit` → `8f8294c4eb91fcd9c11dbb322a4227ff661246dd`. Gate-3b base `8f8294c4e…@2026-10-28T00:14:20Z` (drift from Gate 1 = this merge).
- Exact merge: Build and Release 40607442394 SUCCESS. CI 40607442090 failed only in `govulncheck (vuln gate)` (job 122550452797, log 22.8 KB): `1 unallowlisted finding: GO-2026-6860`.
- Attribution: GO-2026-6860 ("quadratic complexity when coalescing text nodes in html", `golang.org/x/net` fixed 0.62.0; `go.mod` pins v0.61.0) was published 2026-10-27T21:16:18Z, after dev HEAD `9646caebd`'s green govulncheck at 17:37Z. Negative control: `govulncheck ./...` locally on the untouched base `9646caebd` (go 1.26.10) gives rc 3, with Vulnerability #1 GO-2026-6860. A first attempt was an instrument failure (go 1.26.4 under `GOTOOLCHAIN=local`) and is not counted.
- Verdict: an inherited, time-based, non-required red, not this diff. LANDED. The fix (`go.mod` bump, or an allowlist entry with an expiry) is outside fleet authority and V1 (the repo owner) is disabled, so it is handed to Mark in the report. No open PR or issue names it.

**Resolved**: `ailang mission ticket resolve gate5:controlplane-assert-marks-read --sha 8f8294c4e…`. It replied to stapledon; the open count went 43→42 and the signature is no longer listed. About 14h from filing.

**Progress**: goal unmoved on clause 1 (UNMEASURED). Clause 2 UNMET overall; this is a qualifying closure under 48h. Clause 3 MET (42 open). Clause 4: prior evidence only. Clause 5 preserved.

**Routing evidence**: Gate-1 base `9646caebd0f7f43d1e3d6198b897b795ce6ae1ee@2026-10-27T23:37:40Z`; Gate-3b base `8f8294c4eb91fcd9c11dbb322a4227ff661246dd@2026-10-28T00:14:20Z`. Driver note: empty (`MISSION_ROUTING_NOTE=`); `MISSION_OVER_RATION` empty.
- Controller `claude:claude-opus-5-5` (tok: not reported).
- Designer: not spawned (mechanical; no doc); rotation pointer not advanced.
- Planner `codex:gpt-6.1-sol`: 52,335 tok. The resolver said `agent-tool opus fail-closed:no-doc`; the pin was followed per §2(a).
- Executor `codex:gpt-6.1-sol`: 68,970 tok.
- Evaluator `sonnet`, Agent tool: round 1 65,181 subagent tokens / 14 tool uses / 133 s; round 2 70,220 / 3 / 27 s. Judge independence: cross-vendor.
- Quorum: none (no design doc). Metered spend: $0. No lane recorded dead.

**Ruled out**:
- A CLI change (making `read` non-mutating, or rejecting trailing flags): `--peek` already exists, and changing `cmd/ailang/messages*` is outside the fleet's allowlist.
- Widening to `agent-inbox` and `sprint-planner` (both self-sourced).
- Spending the triage dry-run as the done-gate: it never reached the lane probe.
- Treating the govulncheck red as this diff's: the base control is red too.

**Retro lane**: backlog observations only; no skill edit (no gap has two recorded frictions this iteration). Observations, not tickets (the fleet does not self-source):
- (a) `.claude/skills/sprint-planner/scripts/create_sprint_json.sh:276` runs `ailang messages read "$msg_id" --json`. The trailing `--json` is ignored by the same flag-order rule, so `jq` gets human text and `ISSUE_NUM` stays empty, and the call marks the message read as a side effect. It also greps the legacy `msg_YYYYMMDD_…` id shape. Likely dead or broken; not measured end-to-end.
- (b) The `triage` profile's idle short-circuit exits before the lane probe, so the charter's preferred done-gate dry-run (an idle armed sibling) proves nothing about lanes while triage's queue is unchanged. Iteration 63's triage dry-run reached `lanes=ok`, presumably because its queue had changed then. The synthetic-HOME fleet form is the one that measures.

Last 3 code landings: iteration 62, none (diagnosis only); iteration 63, none (a dev red, not a ticket); iteration 64, clause 2 (under 48h). No drift alarm.

**Next**:
1. The top ticket by `slots_lost` that no OPEN decision gates. `agent-tool:sonnet-unavailable` (15) and `reporting:daneel-reports-not-declared-triage` (9) lead; check each against D-FLEET-16..-31 before picking.
2. Watch dev's govulncheck: if Mark or a Dependabot batch bumps `golang.org/x/net` to ≥ 0.62.0, it greens. It is not the fleet's to fix.

## 65 — 2026-10-28 — planner-lane fence parsing LANDED via #1977 → `7c5cc5d33`, independent sonnet PASS 90 (round 2), two duplicate tickets resolved; `quota:admission-has-no-headroom-margin-and-flaps` verified a 3/4 ghost and parked on D-FLEET-32 [HARNESS]

**Pick**: the top `slots_lost` tickets are all gated by OPEN decisions (D-FLEET-24, -22, -20, -19, -18, -25, -26, -28, -31), so the pick fell to the 1-slot tier, oldest first, following iteration 53's **Next**.
- (1) **Bookkeeping**: ghost-check `quota:admission-has-no-headroom-margin-and-flaps` (attended, 10-02). D-FLEET-12 had reserved it for attended work, and attended #1524 says "Resolves" it.
- (2) **Standing rule 1's second item**: `planner-lane:files-section-parses-fenced-bullets` (attended, 10-03; #581 item 1) together with its duplicate `planner-lane:fenced-code-in-files-section` (attended, 10-08).
- Skipped on measured grounds:
  - `pi:unowned-dirty-absolute-vs-relative-paths` (10-02): real at HEAD (`.pi/extensions/unowned-dirty.ts:42` adds absolute paths, while `unownedDirty` compares repo-relative ones). It is D-FLEET-3 park class (`.pi/extensions/**`: diff plus eval impact go to Mark), so it is a separate bookkeeping pick, now Next.
  - `agent-tool:workspace-routing-discovery-timeout` (10-02): a codex-native Agent transport error with no harness surface named.
  - `skill:design-doc-creator-grep-portability` (10-03): outside the guard's scope.
  - `gate0:watermark-advanced-by-dying-fire` (10-03): has an unbuilt planned design (`m-gate0-watermark-crash-safety.md`), so it is a larger item.

**Orphans**: none. No `fleet/i65*` branch, worktree or PR existed. Self-notices rc 0 (control 852:10). The log and charter on `origin/dev` both ended at 64.

**Preflight / observe**:
- Armed. gh `sunholo-voight-kampff`. Billing CLEAN. Directives: 0 on #1945 since `2026-10-22T23:29:08Z` (5 comments, none allowlisted).
- Skill drift: SKILL.md and all 13 resource files are `cmp`-identical to origin/dev in both the resolved symlink and the pin copy.
- Gate-1 base `58a41bad456aa7d0d45a19998058c3775a561cf4@2026-10-28T07:02:36Z`. Dev CI is red only in the non-required `govulncheck (vuln gate)` (GO-2026-6860, inherited, iteration 64). Check set 18, 1 non-green.
- Ledger: 30 rows, 15 OPEN. 42 tickets open.

**Reality check** (first-party, before routing):
- Planner lane, at `58a41bad4`, codex pin, default allowlist:
  - a fenced example bullet denied an infra doc (the ticketed fail-safe direction);
  - **a fenced shell `# comment` matched the section terminator `^#{1,4}[[:space:]]`, so the scan stopped and a later `internal/parser/parser.go` bullet routed to `codex declared:codex-ok` (fail-OPEN). Both tickets call the defect fail-safe.** The control without the fence gave `opus fail-closed:path-not-in-codex-allowlist`;
  - a fenced `## Files` counted as a second section;
  - `derive-planner-lane.sh` is the only copy of the parser.
- Quota: `71eec62b6` (#1524) ends "Resolves harness ticket quota:admission-…". `internal/mission/quota_margin.go` holds per-bucket start margins (codex 2pp, anthropic 1pp, ollama 3pp, openrouter $0.60), and every bucket evaluator applies them.

**Did** (designer skipped; planner, executor and two evaluator runs spawned; generator ≠ judge):
- **Designer**: not spawned. The fix is a parser correctness fix with no design surface; the rotation pointer was not advanced.
- **Planner**: the resolver said `agent-tool opus fail-closed:no-doc` and the role is pinned to `codex:gpt-6.1-sol`, so per `role-spawn-routing.md` §2(a) it ran on the pin (probe rc 0) in detached sibling worktree `.planner-wt-iter65`, since removed. It wrote [the plan](planned/sprint-plan-planner-lane-fences.md): M1 (both awk scans, 7 fixtures, 3 mutation drills) and M2 (AC9 text from `cd76499c5`, changelog). Skipped `messages import-github` (ticket-queue mission).
- **Executor**, `codex:gpt-6.1-sol`, in the sprint worktree: built M1 and M2 and ran the gates (115/0), the three drills (all killed, all restored by `cp`+`cmp`) and x2 (14 fixtures byte-identical). It then exited **rc 1 on `ERROR: Selected model is at capacity`** while writing its final message, so the report was lost. The controller recovered every result from `codex_out.log` and re-ran all gates out of sandbox. This is the D-FLEET-26 string's second occurrence ever, and the first to cost a role's report.
- **Evaluator (quota ghost)**, Agent tool `sonnet`: **PASS 86** ([ghost-quota-evaluation.md](fleet-mission-evidence/iteration65/ghost-quota-evaluation.md)). Ask 1 was built differently (fixed per-bucket margins per D-FLEET-12 instead of a rolling per-mission p75). Ask 2 holds by construction. Ask 4 holds: every rung, probe and spawn reads one `ailang mission quota --over` verdict (`tools/launchd/lib/lane-probe.sh:192`). **Ask 3, the in-run cap, is NOT built.** Its strongest objection was that "Resolves" is not a ruling that ask 3 is waived. So the ticket stays open on new **D-FLEET-32** (recommend B: waive and resolve).
- **Evaluator (fix)**, Agent tool `sonnet`, in the fresh judge worktree `fleet-i65-judge`, round 1 at `c230c7555`: **PASS 74**, 1 BLOCKING: a CRLF closer never closed (`[ \t]*$` refused the `\r`), so a CRLF doc with any fence hid its later bullets and got codex, a new fail-open. NON-BLOCKING: an unterminated fence after a valid bullet routed to codex (fixture v enshrined it), and 4 of its 5 mutants survived (trailing-text closer, indent limit, backtick info string, minimum run length).
- **Controller fixes**:
  - `7f4a38240`: one shared awk function `fence_skip` for both scans; CR stripped before fence matching; END emits an unparsable entry for a fence open at EOF inside the Files section (v now expects opus); fixtures x, y, z, za, zb, zc.
  - `c50fe663b`: `.gitattributes`' `* text=auto` had stored the CRLF fixtures as LF, so the arms would have passed vacuously on a fresh checkout. They now build CRLF copies under mktemp and assert the CR bytes.
  - Nine controller mutants were all killed; B, D, E, F, G and I each have a sole killer.
- **Round 2**, the same judge resumed by name: **PASS 90**. All of its round-1 survivors were killed. The only survivor is M9 (END without `in_files`), which is equivalent because a fence opened before the heading already yields `no-files-section`. It found 0 diffs against base over 600 tracked design docs. Report: [evaluation.md](fleet-mission-evidence/iteration65/evaluation.md).
- **Done-gate** ([done-gate.md](fleet-mission-evidence/iteration65/done-gate.md)), out of sandbox:
  - `make test-launchd-drivers` rc 0 at `c230c7555` and at `c50fe663b` (49 suites).
  - Fleet dry-runs under a synthetic HOME, `AILANG_DRIVER_PINNED=c50fe66`: healthy attempt 1 read `DEGRADED(2)`, because the codex probe hit `Selected model is at capacity` (D-FLEET-26's third sighting today); attempt 2 read `lanes=ok`. Degraded (`codex:bogus-model-i65`) read `DEGRADED(1)` with a named haiku fallback.
  - The real pidfile was unchanged.
- **PR** [#1977](https://github.com/sunholo-data/ailang/pull/1977). The closing-keyword scan over the commits, title and body found 0 matches (control fired). #581 stays open: item 3 is out of scope.

**Gate 3b**: Exact PR head `c50fe663b`: 27 checks terminal, 26 green, 1 red: the non-required `govulncheck (vuln gate)`. MERGEABLE/UNSTABLE; required contexts (`test, lint, build, docs-gate, UI build gate, checks`) green; dev unmoved since Gate 1. Merged with `--merge --match-head-commit` → `7c5cc5d33f78fa266d4afaa44b0ad3bafd5c1b86`; #581 re-read OPEN afterwards. Exact merge: Build and Release 40652473219 SUCCESS; CI 40652473207 failed only in `govulncheck (vuln gate)` (job 122695684845, log 23.4 KB): `1 unallowlisted finding: GO-2026-6860`, the same advisory that reds the untouched base `58a41bad4` (iteration 64). Check set 18, 1 non-green. Verdict: an inherited, time-based, non-required red. LANDED.

**Resolved**: `ailang mission ticket resolve` for both signatures with `--sha 7c5cc5d33…`; each replied to attended. Open count 42→40; neither signature is listed any more. Turnaround about 25 d (10-03) and 20 d (10-08). `quota:admission-has-no-headroom-margin-and-flaps` stays OPEN on D-FLEET-32.

**Progress**: goal unmoved on clause 1 (UNMEASURED). Clause 2 UNMET overall; these closures took about 25 d and 20 d, so they do not qualify. Clause 3 MET (40 open). Clause 4: prior evidence only. Clause 5 preserved.

**Routing evidence**: Gate-1 base `58a41bad456aa7d0d45a19998058c3775a561cf4@2026-10-28T07:02:36Z`; Gate-3b base `7c5cc5d33f78fa266d4afaa44b0ad3bafd5c1b86@2026-10-28T07:48:52Z` (drift from Gate 1 = the 6 commits of this merge). Driver note: empty (`MISSION_ROUTING_NOTE=`); `MISSION_OVER_RATION` empty.
- Controller `claude:claude-opus-5-5` (tok: not reported).
- Designer: not spawned (mechanical; no doc); rotation pointer not advanced.
- Planner `codex:gpt-6.1-sol` (56,756 tok). The resolver said `agent-tool opus fail-closed:no-doc`; the pin was followed per §2(a).
- Executor `codex:gpt-6.1-sol` (66,637 tok). The run ended rc 1 on `Selected model is at capacity` after finishing the work; results were recovered from its log.
- Evaluator `sonnet`, Agent tool: quota ghost 58,962 subagent tokens / 10 tool uses / 79 s; fix round 1 67,359 / 9 / 125 s; fix round 2 78,134 / 5 / 100 s. Judge independence: cross-vendor.
- Quorum: none (no design doc). Metered spend: $0. No lane recorded dead (the capacity errors were transient; the pin answered again minutes later).

**Ruled out**:
- Reading the tickets' "fails safe" at face value: the fenced-comment shape is fail-open.
- Committing CRLF fixtures: `* text=auto` normalises them, and `.gitattributes` is outside fleet scope.
- Resolving the quota ticket on the attended commit's "Resolves" line alone: one of four asks is unbuilt, and waiving it is HD-2a.
- Treating the `DEGRADED(2)` healthy dry-run as a regression: the cause was a provider capacity message, and an identical re-run read `lanes=ok`.

**Retro lane**: backlog observations only; no skill edit (no gap has two recorded frictions this iteration). Observations, not tickets:
- (a) The codex capacity string (D-FLEET-26) fired twice in one fire. It cost the executor's final report (the work itself survived on disk) and turned a healthy dry-run degraded. That is new evidence for D-FLEET-26; the row text is unchanged.
- (b) A fixture-shaped test of line endings is vacuous under `* text=auto` unless the test builds the bytes itself. Any future CRLF arm anywhere in the repo has the same trap.

Last 3 code landings: iteration 63, none (a dev red, not a ticket); iteration 64, clause 2 (under 48h); iteration 65, none (old tickets, about 25 d). No drift alarm: no routable ticket moves an UNMET clause, because none was filed in the last 48h.

**Next**:
1. `pi:unowned-dirty-absolute-vs-relative-paths` (10-02): measure first-party, then park it per D-FLEET-3 with the proposed diff and its expected pi-eval impact as a decision row. A bookkeeping pick, so a second item may follow.
2. Then the 1-slot tier, oldest first: `gate0:watermark-advanced-by-dying-fire` (#981; its design `m-gate0-watermark-crash-safety.md` is planned and unbuilt), `skills:showcase-feature-discoverability`, `gate5:controlplane-send-refused-local-store`.
3. Watch dev's govulncheck (GO-2026-6860): it is not the fleet's to fix.

## 66 — 2026-10-28 — `gate5:controlplane-send-refused-local-store` LANDED via #1994 → `d95ead6af`, independent sonnet PASS 91 (round 2); `pi:unowned-dirty-absolute-vs-relative-paths` measured and parked on D-FLEET-33 [HARNESS]

**Pick**: iteration 65's **Next**. The top `slots_lost` tickets are still all gated by OPEN decisions, so the pick took the 1-slot tier, oldest first.
- (1) **Bookkeeping**: `pi:unowned-dirty-absolute-vs-relative-paths` (interactive, 10-02). This was a D-FLEET-3 park: measure it, then file the diff with its eval impact.
- (2) **Standing rule 1's second item**: `gate5:controlplane-send-refused-local-store` (world, 10-05).
- Skipped on measured grounds:
  - `gate0:watermark-advanced-by-dying-fire` (10-03): open PR #1962 (`p0/gate0-watermark-crash-safety`, created 10-26, `Closes #981`) builds it. The branch is not a fleet branch, so it is not attributable to this loop and was left alone.
  - `skills:agents-copies-stale-non-mission` (10-03): non-mission skills are outside the guard's scope.
  - `skills:showcase-feature-discoverability` (10-03): three asks (docs MCP in bootstrap, a mandatory syntax step in sprint-executor, a CI feature-usage gate). The third is a CI workflow (refused by the guard), and together they need a design, so it is a larger item. It is now Next.
  - The quota, D-FLEET-24, -25, -26 and -18 tickets stay gated by their OPEN rows.

**Orphans**: none. No `fleet/i66*` branch, worktree or PR existed. Self-notices rc 0 (control 852:10). The log and charter on `origin/dev` both ended at 65.

**Preflight / observe**:
- Armed. gh `sunholo-voight-kampff`. Billing CLEAN. Directives: 0 on #1945 since `2026-10-22T23:29:08Z` (6 comments, none allowlisted).
- Skill drift: SKILL.md and all 13 resource files are `cmp`-identical to origin/dev in the resolved symlink.
- Gate-1 base `c0458fdfd4f950eaeceae23e80ab371b6764a415@2026-10-28T14:21:31Z`. Dev CI was in progress; the check set (18) showed 1 completed non-green, the non-required `govulncheck (vuln gate)` (GO-2026-6860, inherited, iteration 64).
- Ledger: 31 rows, 16 OPEN. 40 tickets open; none filed in the last 48h.
- Driver note: codex over its daily ration (`probe rc=75`) for both planner and executor, handed to opus and `claude:claude-haiku-5-5`.

**Reality check** (first-party, before routing):
- unowned-dirty, at `c0458fdfd`: `.pi/extensions/unowned-dirty.ts:42` and `:78` add `${ctx.cwd}/${p}`; `unownedDirty` compares with porcelain's relative paths. A node repro of the HEAD wiring gave `["mine.ts","theirs.go"]` for own `mine.ts`, and `["theirs.go"]` once relativized. The pi eval path: `internal/executor/pi/pi.go:180` and `scripts/mission_pi_run.sh:302` both run `pi --mode json`; pi 0.85.1 `print-mode.js:53` binds extensions with no UI context, so `notify` is `runner.js:92`'s `() => { }`. `cmd/ailang/pi_assets/unowned-dirty.ts` `cmp`-matches the source (a `make pi-assets` copy, outside scope).
- controlplane: `ailang messages inboxes` reads `TRIAGE controlplane` on the gcp store and `! NOTHING controlplane … accepted and never dispatched` with `AILANG_STORAGE_MESSAGING=local`, so `checkSendInbox` refuses there. `~/.zshenv:38-39` exports the gcp pair, which is the only reason controller sends have worked. The driver scopes its own sends per call (`mission-control.sh:248`, `:307`, `:361`, `:2641`). `ailang mission ticket` is unaffected (it pins `config.MissionMessagePlaneProject`). An audit of `.claude/skills/mission-control` found 4 unscoped prescribed commands and 3 already scoped by `gate-3-triage.md`'s `$MSG`.

**Did** (designer skipped; planner, executor and two evaluator rounds spawned; generator ≠ judge):
- **Designer**: not spawned. Mechanical fix with no design surface; the rotation pointer was not advanced.
- **Planner**: the resolver said `agent-tool opus fail-closed:env-pin` and the pin is `opus`, so it ran on the Agent tool, foreground. It wrote [the plan](planned/sprint-plan-controlplane-store-scope.md): M1 (scope the 4 lines with the literal pair, not `$MSG`, because a shell variable does not survive between Bash-tool calls; regenerate the mirror; changelog) and M2 (the guard arm, plus the existing 3(c) arm stripping the prefix; 4 drills). It rejected a global driver export, because that would also move the controller's reads.
- **Executor**, `claude:claude-haiku-5-5` through `claude-sub` (probe rc 0, keys stripped), in the sprint worktree: built M1 and M2 as one commit (`b3bdd1050`, rebased to `60f60367c`). It ran D1–D4 (all red, each restored with `cp` + `cmp`) and the done-gate. Its first full suite run read rc 2 on three unrelated suites, each green alone and on re-run.
- **Unowned-dirty park**: the controller wrote the proposed diff against a copy and ran it under node: 4/4, and a prefix-strip mutant was killed. After the judge's objection, the controller added `realOrSelf` (realpath on both sides) and a symlink test: 5/5, with an identity mutant killed. The diff applies to HEAD's files with `patch -p1`.
- **Evaluator round 1**, Agent tool `sonnet`, in the fresh detached judge worktree `.wt-fleet-iter66-judge` at `60f60367c`: **PASS 84**, 0 blocking. It ran 8 drills. D8, the pair in a trailing comment after an unscoped command, PASSED the arm. D9, `send` at end of line, was caught only by the positive control. Part B (the park): PARTIAL; claims (i)–(iv) CONFIRMED. Its strongest objection was that `--show-toplevel` is physical while cwd may be a symlink, which would silently drop every own file.
- **Controller fix** `ffde0469a`: the arm strips scoped occurrences and fails on any remaining `ailang messages send|read|ack|unack( |$)`. D8 and an end-of-line D9b each pass the pre-fix arm and fail the new one; D1 and a wrong-project mutant (D13) fail. The failure message no longer prints empty fields.
- **Round 2**, the same judge resumed by name at `ffde0469a`: **PASS 91**. D8, D9b and its own D14–D17 are all red, with no new false reds. Residual: a double space or tab between `messages` and `send` is not matched. Report: [evaluation.md](fleet-mission-evidence/iteration66/evaluation.md).
- **Done-gate** ([done-gate.md](fleet-mission-evidence/iteration66/done-gate.md)): `make test-launchd-drivers` rc 0 at `60f60367c` and at `ffde0469a` (49 suites); `sync-agents-skills.sh --check` rc 0. Fleet dry-runs under a synthetic HOME, `AILANG_DRIVER_PINNED=ffde046`: healthy and degraded both read `DRY RUN ok`, `lanes=DEGRADED(2)`; codex is over its ration, and each arm named its fallback. The real pidfile was unchanged.
- **PR** [#1994](https://github.com/sunholo-data/ailang/pull/1994). The closing-keyword scan over the commits and body found 0 matches (control fired).

**Gate 3b**: PR head `ffde0469a`: 27 checks terminal, 26 green, 1 red: the non-required `govulncheck (vuln gate)`. Its log (24.7 KB) shows `1 unallowlisted finding: GO-2026-6860`; dev's own HEAD `af8d75e37` is red on the same job. MERGEABLE/UNSTABLE; required contexts (`test, lint, build, docs-gate, UI build gate, checks`) green. Merged with `--merge --match-head-commit` → `d95ead6afec0e608b94aeabc29d7c7478f4493f2`. Exact merge `d95ead6af`: Build and Release 40713515460 SUCCESS; CI 40713516083 failed only in `govulncheck (vuln gate)` (job 122905086538, log 23.4 KB): `1 unallowlisted finding: GO-2026-6860`, the advisory that reds the parent. `SonarCloud Code Analysis` is red on the merge and on the parent `af8d75e37`, so it is inherited too. Check set 18, 2 non-green, both inherited and non-required. Verdict: LANDED. (Instrument note: the poller's `red=` field word-split check names containing spaces, so the verdict was read from the check-run API directly.)

**Resolved**: `ailang mission ticket resolve gate5:controlplane-send-refused-local-store --sha d95ead6af…`, replied to world. Open count 40→39, then 40 again: `pin-root:worktree-removed-mid-fire` (stapledon) was filed at 15:27Z during this fire. Turnaround about 23 d (10-05 16:48Z → 10-28). `pi:unowned-dirty-absolute-vs-relative-paths` stays OPEN on D-FLEET-33.

**Progress**: goal unmoved on clause 1 (UNMEASURED). Clause 2 UNMET overall; this closure took about 23 d, so it does not qualify. Clause 3 MET (40 open, one of them filed during this fire). Clause 4: prior evidence only. Clause 5 preserved.

**Routing evidence**: Gate-1 base `c0458fdfd4f950eaeceae23e80ab371b6764a415@2026-10-28T14:21:31Z`; merge `d95ead6af`. Driver note: `lanes degraded: planner and executor codex gpt-6.1-sol unusable (probe rc=75, over daily ration) → opus / claude:claude-haiku-5-5`; `MISSION_OVER_RATION=codex`.
- Controller `claude:claude-opus-5-5` (tok: not reported).
- Designer: not spawned (mechanical; no doc); rotation pointer not advanced.
- Planner `opus`, Agent tool (62,210 subagent tokens / 7 tool uses / 116 s). Resolver `agent-tool opus fail-closed:env-pin`; the pin is the bare alias `opus`, so the hook and the resolver agreed.
- Executor `claude:claude-haiku-5-5` via `claude-sub -p --permission-mode bypassPermissions` (tok: not reported, text output), about 16 min, rc 0.
- Evaluator `sonnet`, Agent tool: round 1 72,121 subagent tokens / 16 tool uses / 113 s; round 2 79,166 / 3 / 53 s. Judge independence: different model from planner (opus) and executor (haiku); same vendor, so vendor-level independence is not claimed.
- Quorum: none (no design doc). Metered spend: $0. No lane recorded dead (codex is over ration, not dead).

**Ruled out**:
- A global `AILANG_STORAGE_MESSAGING` export from the driver: it would move the controller's reads too, and the operator rule is the prefix on sends only.
- `$MSG` at the new sites: a shell variable defined in one Bash-tool call is empty in the next, so the command would run unscoped with no error.
- Reading the ticket as already fixed because sends work today: they work only through a user dotfile, and the local store still refuses `controlplane` at HEAD.
- Reading "evals load the extension" as eval impact: they do load it (globally installed), but in json mode `notify` goes nowhere.

**Retro lane**: backlog observations only; no skill edit (no gap has two recorded frictions this iteration). Observations, not tickets:
- (a) Codex was over ration for the whole fire, so both dry-run arms read `DEGRADED(2)` and the "healthy" arm could not be measured healthy. The done-gate's healthy arm needs a lane state the loop cannot set; a healthy reading under a live ration block is not available.
- (b) A text-output `claude -p` executor reports no token count. The Gate-4 per-role token rule then reads "not reported" for every claude-recipe role unless the recipe uses `--output-format json`.

Last 3 code landings: iteration 64, clause 2 (under 48h); iteration 65, none (old tickets, about 25 d); iteration 66, none (about 23 d). No drift alarm: no routable ticket moves an UNMET clause, because none was filed in the last 48h.

**New ticket during this fire**: `pin-root:worktree-removed-mid-fire` (stapledon iteration 55, filed 15:27Z, `blocking=none`). Stapledon's live pin worktree `~/.ailang-driver-pin/stapledon` disappeared between about 12:00Z and 15:00Z while its fire ran. The ticket notes the pin root's mtime was 15:52 local (14:52Z), which is when this iteration's fleet dry-runs ran, so attribution was checked. The dry-runs ran under a synthetic HOME with `AILANG_DRIVER_PINNED` set; `pin-root.sh:260` derives the pin dir from `$HOME`, and both 17-line dry-run logs show no pin or worktree action. `~/.ailang/state/worktree-sweep.log` has no stapledon entry (its last run, 13:46Z, deleted one unrelated branch). Stapledon's pin is a worktree of a different repo's clone, which this repo's `git worktree prune` cannot touch. Not attributable to this iteration on those instruments; the cause is unmeasured.

**Next**:
1. `pin-root:worktree-removed-mid-fire` (stapledon, filed 10-28 15:27Z): the only ticket inside the 48h window (it closes 10-30 15:27Z), so it moves clause 2 and outranks the 1-slot tier. Measure first: what removes a live `~/.ailang-driver-pin/<mission>` (pin-root's own re-pin of another mission, a sweep, an attended cleanup), using the dir's entries and the git worktree metadata of stapledon's clone.
2. `skills:showcase-feature-discoverability` (10-03; #476): measure the three asks at HEAD. The CI-gate ask is outside scope (park it or route it); the mission-bootstrap and sprint-executor asks may need a design doc.
3. Then the 1-slot tier, oldest first: `pi:openrouter:uncapped-max-tokens-402`, `pi:provider-error:no-retry`, `stall:watchdog:slow-on-hung-tool` (all 10-07). Check whether the pi ones are `.pi/extensions/**` (D-FLEET-3) or quota/billing policy (HD-2a) before routing.
4. Watch dev's govulncheck (GO-2026-6860): it is not the fleet's to fix.

## 67 — 2026-10-28 — `pin-root:worktree-removed-mid-fire` LANDED via #1999 → `17ec65d68`, independent sonnet PASS 88; root cause was fleet iteration 66's own synthetic-HOME dry-run [HARNESS]

**Pick**: iteration 66's **Next**, `pin-root:worktree-removed-mid-fire` (stapledon iteration 55, filed 10-28 15:27Z, `blocking=none`). It was the oldest ticket inside the 48h window. A second ticket, `ci:distinct-build-test-failures` (triage, 21:18Z), arrived later and is in the window too.

**Orphans**: none. No `fleet/i67*` branch, worktree or PR existed. Self-notices rc 0 (control 852:10). The log and charter on `origin/dev` ended at 66.

**Preflight / observe**:
- Armed. gh `sunholo-voight-kampff`. Billing CLEAN. Directives: 0 on #1945 since `2026-10-22T23:29:08Z` (7 comments, none allowlisted).
- Skill drift: SKILL.md and all 13 resource files `cmp`-identical to origin/dev at the resolved symlink target.
- Gate-1 base `4c3dc018261fd263cba18cce8f001a2bb61c6377@2026-10-28T22:05:03Z`.
- **Dev CI red on `4c3dc0182`, outside fleet scope**: required `test` failed in `internal/runner` `TestWorkerTerminalPTYLifecycle/strict/sigint` (a Python `int('')` in the PTY harness). It failed on 2 of the last 8 dev CI runs and passed on the other 6, so it is an intermittent flake. V1 owns the repo and is disabled. Triage PR #1926 and `design_docs/planned/v0_54_17/m-worker-pty-pidfile-ack.md` already track it. Also red: the non-required `govulncheck` (GO-2026-6860, inherited since iteration 64).
- Ledger valid, 32 rows, 17 OPEN. 41 tickets open.
- Driver: codex was in ration this fire; `MISSION_OVER_RATION` was empty.

**Reality check** (first-party, before routing):
- `pin-root.sh` never removes a worktree, and `test_pin_root.sh` isolates both `HOME` and `AILANG_DRIVER_PIN_DIR`.
- The real `~/.ailang/state/worktree-sweep.log` had no stapledon line.
- A scan of every claude transcript (plus the codex and pi sessions) between 11:30 and 15:30Z found no command that removed a pin by name.
- The `scripts/worktree_sweep.sh` protect list is `$HOME`-derived. The SessionStart hook `scripts/hooks/git_health.sh` gates the sweep on a `$HOME` stamp and exits early only on `AILANG_MISSION_STAGE`/`AILANG_TASK_ID`, which `env -i` strips.
- Iteration 66's transcript shows a fleet dry-run under `HOME=/tmp/fleet_i66_home.3S7K` at 14:50:33Z. That directory still existed, and its own `.ailang/state/worktree-sweep.log` reads `removed worktree …/.ailang-driver-pin/stapledon` at 15:52:00 CET (14:52Z), then `triage` (15:52:08) and `world` (15:52:24), plus a landed branch `fix/agent-prompts-off-argv` (`cf73e7d82`).
- The pin directory's own mtime, 15:52 CET, matches. Stapledon's controller worked in `stapledons-godot`, so `lsof` saw nothing using the pin.
- **So iteration 66's done-gate caused the ticket it then filed as unattributed.** Its record said "no pin action logged" because it looked at the real log, while the sweep had written to the synthetic HOME.
- `perl -e 'print((getpwuid($<))[7])'` returns the account home under `HOME=/tmp/x`.

**Did** (designer skipped; planner, executor and one evaluator round spawned; generator ≠ judge):
- **Designer**: not spawned. Mechanical fix with no design surface; the rotation pointer was not advanced.
- **Planner**, `codex:gpt-6.1-sol`. The resolver said `agent-tool opus fail-closed:no-doc` against the codex pin, so the planner went straight to the pin, as at iteration 64. Probe rc 0; run rc 0. It wrote [the plan](planned/sprint-plan-sweep-account-home.md):
  - M1: the hook and three test arms.
  - M2: both dry-run recipes get `AILANG_WORKTREE_SWEEP=0`, plus a changelog fragment.
  - Rejected: editing the sweep (outside fleet scope); skipping the sweep under any non-account HOME, and resetting HOME for it (either would make the fake-HOME self-test write the real stamp).
- **Executor**, `codex:gpt-6.1-sol`, rc 0. It built M1 and M2 with snapshots, ran drills A/B/C on disposable copies (each red), and reported every gate green. The controller re-ran the gates outside the sandbox. It rebuilt one commit per milestone from the snapshots: `6e36e9d5f` (M1) and `d144c85bd` (M2).
  - Controller slip: a zsh-unsplit `$FILES` made the manifest step fail, and the controller then reset two M2 files in the tree. They were restored from the executor's cumulative `.snap/M2`. The M2 diff stat matched the executor's original, and every gate was re-run on the final tree.
- **Evaluator**, Agent tool `sonnet`, fresh judge worktree at `d144c85bd`: **PASS 88**, 0 blocking.
  - A sweep dry-run (no `--apply`) under a synthetic HOME printed REMOVE for real `.wt-fleet-iter66*` worktrees. With the hook's protect string, every one read KEEP protected.
  - It reproduced the three drills.
  - Non-blocking findings: (1) `worktree_sweep.sh` still trusts `$HOME` for any other `--apply` caller (none today); (2) it did not verify that the opt-out reaches the lane probes; (3) removals outside the protected prefixes still log to a synthetic HOME; (4) arm A also fails under drill B; (5) a lookup failure skips the sweep silently.
  - Strongest objection: fix the sweep, not the caller. That file is outside fleet authority, so it is recorded as a residual. Report: [evaluation.md](fleet-mission-evidence/iteration67/evaluation.md).
- **Done-gate** ([done-gate.md](fleet-mission-evidence/iteration67/done-gate.md)):
  - `make test-launchd-drivers` rc 0 at base `4c3dc0182` and at head `d144c85bd` (49 suites); hook self-test 9 → 12.
  - World dry-runs (idle and armed; real HOME; `AILANG_WORKTREE_SWEEP=0`; `AILANG_DRIVER_PINNED=d144c85bd`): healthy `DRY RUN ok`, `lanes=ok`; degraded `lanes=DEGRADED(1)`, executor handed to `claude:claude-haiku-5-5`.
  - The real sweep log (sha prefix `882378d5df7299e8`, 171 lines), the sweep stamp and the pin listing were unchanged across every gate and dry-run.
- **PR** [#1999](https://github.com/sunholo-data/ailang/pull/1999). The closing-keyword scan found 0 matches; the control fired.

**Gate 3b**:
- PR head `d144c85bd`: 27 checks terminal, 26 green. The one red is the non-required `govulncheck (vuln gate)`; its log (23.3 KB) shows `1 unallowlisted finding(s): GO-2026-6860`, inherited.
- The required contexts (`test, lint, build, docs-gate, UI build gate, checks`) were green and the PR read MERGEABLE/UNSTABLE. Merged with `--merge --match-head-commit` → `17ec65d68d79945ca3a0078275892956e8ef1edc`.
- Exact merge `17ec65d68`: Build and Release 40781817700 SUCCESS; CI 40781817640 failed only in `govulncheck (vuln gate)` (job 123145248380: `1 unallowlisted finding(s): GO-2026-6860`, inherited). The required `test` passed on the merge. `SonarCloud Code Analysis` is red on the merge with `74.8% Coverage on New Code (required ≥ 80%)`, a branch-level new-code gate that was already red on `07ce4a8dc` and `d95ead6af`; Sonar does not measure shell, so it is not from this change.

**Resolved**: `ailang mission ticket resolve pin-root:worktree-removed-mid-fire --sha 17ec65d68…`, which replied to stapledon. 41 → 40 open. Turnaround about 7.5h (10-28 15:27Z → about 22:45Z).

**Progress**: goal unmoved on clause 1 (UNMEASURED). Clause 2 is UNMET overall; this closure qualifies at under 48h. Clause 3 MET (40 open). Clause 4: prior evidence only. Clause 5 preserved.

**Routing evidence**: Gate-1 base `4c3dc018261fd263cba18cce8f001a2bb61c6377@2026-10-28T22:05:03Z`; merge `17ec65d68`. Routing note: as configured (codex in ration).
- Controller `claude:claude-opus-5-5` (tok: not reported).
- Designer: not spawned (mechanical; no doc).
- Planner `codex:gpt-6.1-sol` (43,741 tok). Resolver `agent-tool opus fail-closed:no-doc`; routed to the codex pin.
- Executor `codex:gpt-6.1-sol` (84,701 tok), rc 0.
- Evaluator `sonnet`, Agent tool (62,349 subagent tokens / 8 tool uses / 119 s). Judge independence: cross-vendor (Anthropic vs OpenAI).
- Quorum: none (no design doc). Metered spend: $0. No lane recorded dead.

**Ruled out**:
- `pin-root.sh` removing its own pin: it never removes, it refuses.
- A controller or attended `rm`/`worktree remove` naming the pin: the transcript scan was empty, and the 15:06Z attended cleanup touched `stapledons-godot-*` only.
- Iteration 66's "the instruments do not attribute it to this iteration's dry-runs": refuted by the synthetic HOME's own log.
- Skipping the sweep whenever HOME differs from the account home: the self-test could then only exercise a launch by writing the real stamp.

**Retro lane**: process fix (charter). Both dry-run recipes now carry `AILANG_WORKTREE_SWEEP=0`, with the why, so a dry-run starts no side jobs. No skill edit: this is the first recorded friction of its class. Observations, not tickets:
- (a) An instrument whose log path is `$HOME`-relative is blind under a synthetic HOME. Iteration 66 checked the real log, saw nothing, and wrote "not attributable". Any "no X logged" claim made during or after a synthetic-HOME run must also read the synthetic HOME's `.ailang/state/` before it counts.
- (b) The triage and world pins were removed too, but no fire was mid-flight in them, so nothing filed a ticket; they were re-created on their next fires. A destructive side effect is ticketed only when it happens to hit a live fire.
- (c) The `worktree_sweep.sh` residual stays outside fleet authority (`scripts/` beyond `scripts/hooks/**` and `scripts/mission_*`). The hook is its only `--apply` caller today.

Last 3 code landings: iteration 65, none (about 25 d); iteration 66, none (about 23 d); iteration 67, clause 2 (under 48h). No drift alarm.

**Next**: `ci:distinct-build-test-failures` (triage, 10-28 21:18Z; the 48h window closes 10-30 21:18Z). It asks for a diagnosis of `internal/runner` and daemon test failures, so the reality check must first decide whether the fix lies inside fleet scope. Then `skills:showcase-feature-discoverability` and the 1-slot tier.

## 68 — 2026-10-29 — `sprint-planner:template-make-test-changelog-filesize` LANDED via #2001 → `e29a42482`, independent sonnet PASS ~88–90; `quorum:zero-signal-guard-vacuous-with-controller` resolved as a ghost; `ci:distinct-build-test-failures` measured out of scope [HARNESS]

**Pick**: iteration 67's **Next** was `ci:distinct-build-test-failures` (triage, 10-28 21:18Z), the only ticket inside the 48h window. Its scope check failed: all three failing tests live outside fleet authority (`internal/runner/worker_terminal_pty_test.go`, `internal/daemon/daemon_test.go`, `cmd/ailang/run_policy_owned_test.go`), so it is parked on the D-FLEET-17 class (re-measured below). The pick moved to the oldest in-scope, ungated ticket, `sprint-planner:template-make-test-changelog-filesize` (attended, 10-09). The bookkeeping second item was the ghost `quorum:zero-signal-guard-vacuous-with-controller` (attended, 10-08).

**Orphans**: none. No `fleet/i68*` branch, worktree or PR existed. Self-notices rc 0 (control 852:10). The log and charter on `origin/dev` ended at 67.

**Preflight / observe**:
- Armed. gh `sunholo-voight-kampff`. Billing CLEAN. Directives: 0 on #1945 since `2026-10-22T23:29:08Z` (8 comments, none allowlisted).
- Skill drift: SKILL.md and all 13 resource files `cmp`-identical to origin/dev at the resolved symlink target.
- Gate-1 base `4e6ea7b25091e2a50c8de64c68257c024e54a331@2026-10-29T05:14:02Z`.
- Dev CI on `4e6ea7b25`: Build and Release red in `Build ubuntu-latest`, `internal/runner` `TestWorkerTerminalPTYLifecycle/eval/normal`. The required `test` passed. `govulncheck` (GO-2026-6860, inherited) and SonarCloud are red as on the parents.
- Ledger valid, 32 rows, 17 OPEN. 40 tickets open. Codex in ration (`MISSION_OVER_RATION` empty).

**Reality check** (first-party, before routing):
- `ci:distinct-build-test-failures`: the last 10 dev Build and Release runs failed 3 times, each on a different test: `07ce4a8dc` `TestRunPolicy_OwnedCancellation/managed/sigterm`, `6643f7ca2` `TestDaemon_TaskDedupSuppressesRepeats`, and `4e6ea7b25` `TestWorkerTerminalPTYLifecycle/eval/normal`. The third is new since the ticket was filed. The first two are the ticket's own pair. All three files are outside the fleet allowlist, and instrumenting them is the ticket's ask. Triage PR #1926 and `planned/v0_54_17/m-worker-pty-pidfile-ack.md` already cover the PTY test. The ticket stays open.
- `sprint-planner:template-make-test-changelog-filesize`: asks 1–3 were already landed by attended #1770 (`dbe4b26a3`, 10-12), and ask 6 by `b2d21a070` (jq in `docker/Dockerfile.agent-base`, 10-09).
  - Ask 4 was still open. The skill had no rule against putting a plan in `design_docs/planned/ailang-core-triage/`. Two plans had landed there before (`17254535d`, `a65e6c12b`), and `internal/coordinator/triage_rubric_test.go:108` checks every file in that directory.
  - Ask 5 was still open too: the skill had no rule on how a plan may stop for a human decision.
  - Constraint found at pick time: `scripts/context_docs_baseline.txt:24` caps the planner SKILL.md at 597 lines (596 at base). That file is outside fleet scope.
- `quorum:zero-signal-guard-vacuous-with-controller`: `quorum.go` at HEAD counts present external reviewers only (`a5f9af982`, #1767). The queue's P1 #8 had resolved the `...-controller-verdict` signature, so this re-filed spelling was left open. #651 is CLOSED.

**Did** (designer skipped; planner, executor and one evaluator round spawned; generator ≠ judge):
- **Designer**: not spawned. Mechanical skill fix; the rotation pointer was not advanced.
- **Planner**, `codex:gpt-6.1-sol`. The resolver said `agent-tool opus fail-closed:no-doc` against the codex pin, so the planner went straight to the pin (rule (a), as at iterations 64 and 67). Probe rc 0; run rc 0. It wrote [the plan](planned/sprint-plan-planner-placement-stop-gates.md):
  - M1: the two rules in one SKILL.md line, in `resources/coordinator.md`, and in `resources/sprint_plan_template.md`.
  - M2: a guard in `create_sprint_json.sh` plus `test_create_sprint_json.sh`, wired into `test-launchd-drivers`.
  - M3: a changelog fragment.
  - Rejected: exempting `*-sprint-plan.md` in the rubric test (`internal/` is out of scope, and an exemption would let a misplaced row escape the rubric); a prose-only fix.
- **Executor**, `codex:gpt-6.1-sol`, rc 0.
  - It built M1–M3 with snapshots, and all 10 of its mutants went red.
  - Inside the sandbox it reported `make test-launchd-drivers` `4 of 50 suites FAILED` (kill-tree, stall, stage-isolation, codex-controller-env). All four pass outside the sandbox, so they were sandbox denials.
  - The controller rebuilt one commit per milestone from the snapshots (`29b945e55`, `0ea4dd233`, `249ee0d94`), running the new test at each boundary. `shasum -c` matched 11 of 13 files. The 2 that differ are the SKILL.md copies, where the controller moved one blank line so the new bullet sits inside the checklist. Then `298faf903` set M2 `passes: true`.
- **Evaluator**, Agent tool `sonnet`, fresh judge worktree at `249ee0d94`: **PASS ~88–90**, 0 blocking.
  - Its own guard drills all behaved as designed. Refused: `./`, `..`, absolute, `../../` from a subdirectory, symlinked dir, `//`. Accepted: the sibling `ailang-core-triage-extra/`, and a triage design doc as the third argument. With python3 missing the script exits 127 before any side effect.
  - Test mutations: deleting the guard fails 30 of 49 arms; moving it below the import fails 27.
  - It confirmed C1, C2 and C7. C7 included a mutation of the zero-signal guard, which reddened 4 quorum tests.
  - Non-blocking: on case-insensitive APFS, `AILANG-core-triage/` passes the guard; python3 is an undocumented prerequisite; "decision-ledger row ID" is not defined in SKILL.md.
  - Strongest objection: the guard is a case-sensitive backstop, and the prose rule is the real control. Report: [evaluation.md](fleet-mission-evidence/iteration68/evaluation.md).
- **Done-gate** ([done-gate.md](fleet-mission-evidence/iteration68/done-gate.md)):
  - `make test-launchd-drivers` rc 0 at base (49 suites) and head (50).
  - `check-context-docs`, `check-skills`, `check-changelog`, `check-file-sizes` and shellcheck all rc 0.
  - World dry-runs (idle and armed; real HOME; `AILANG_WORKTREE_SWEEP=0`; `AILANG_DRIVER_PINNED=249ee0d94`): healthy `lanes=ok`; degraded `lanes=DEGRADED(1)`, executor handed to `claude:claude-haiku-5-5`.
  - The sweep log (`87fe4fc2a0c40067`, 173 lines), the pin listing and the world pidfile were unchanged.
- **PR** [#2001](https://github.com/sunholo-data/ailang/pull/2001). The closing-keyword scan found 0 matches; the control fired.

**Gate 3b**:
- PR head `298faf903`: 27 checks terminal, 26 green. The one red is `govulncheck (vuln gate)`, with `1 unallowlisted finding(s): GO-2026-6860` (inherited). MERGEABLE/UNSTABLE.
- Merged with `--merge --match-head-commit` → `e29a42482a3e39293b64403b1e4c0036334af7cf`.
- Exact merge: Build and Release 40827874474 SUCCESS. CI 40827874480 failed only in `govulncheck`. SonarCloud `Quality Gate failed`, as on the parent `4e6ea7b25`.
- The same govulncheck log lists `8 expired allowlist entr(ies)` (GO-2025-3557/3558/3559/3582 …), all dated `expired 2026-10-29`. This is outside fleet scope (`.govulncheck-allow.yml`).

**Resolved**:
- `sprint-planner:template-make-test-changelog-filesize` `--sha e29a42482…`. Turnaround about 20 d (10-09 06:22Z → 10-29 about 07:35Z).
- `quorum:zero-signal-guard-vacuous-with-controller` `--sha a5f9af982`, as a ghost. Open for about 21 d; it had been fixed since 10-12.
- Both replies went to attended. 40 → 38 open.

**Progress**: goal unmoved on clause 1 (UNMEASURED). Clause 2 UNMET: both closures took about 20 d, and the only in-window ticket is outside fleet authority. Clause 3 MET (38 open). Clause 4: prior evidence only. Clause 5 preserved.

**Routing evidence**: Gate-1 base `4e6ea7b25091e2a50c8de64c68257c024e54a331@2026-10-29T05:14:02Z`; merge `e29a42482`. Routing note: as configured (codex in ration).
- Controller `claude:claude-opus-5-5` (tok: not reported).
- Designer: not spawned (mechanical; no doc).
- Planner `codex:gpt-6.1-sol` (60,214 tok). Resolver `agent-tool opus fail-closed:no-doc`; routed to the codex pin.
- Executor `codex:gpt-6.1-sol` (131,595 tok), rc 0.
- Evaluator `sonnet`, Agent tool (76,193 subagent tokens / 10 tool uses / 170 s). Judge independence: cross-vendor (Anthropic vs OpenAI).
- Quorum: none (no design doc). Metered spend: $0. No lane recorded dead.

**Ruled out**:
- Working `ci:distinct-build-test-failures` in fleet scope: all three test files are outside the allowlist.
- Editing `scripts/context_docs_baseline.txt` so SKILL.md could grow: outside scope; the detail went to `resources/` instead.
- Exempting plan files in the triage rubric test: `internal/` is out of scope, and an exemption weakens the rubric.

**Retro lane**: none filed. No skill edit; nothing in this iteration's friction recurs twice. Observations, not tickets:
- (a) A codex executor's in-sandbox `make test-launchd-drivers` reads 4 suites red that pass outside. This is false-green (3) as documented, and the out-of-sandbox re-run settled it.
- (b) A ticket that keeps the same defect under two signatures (`...-controller` vs `...-controller-verdict`) stayed open about 16 d after its fix landed. Resolving one spelling does not close the other.
- (c) The govulncheck allowlist carries 8 entries that expire today. Whether that changes the gate is V1's or Mark's to read.

Last 3 code landings: iteration 66, none (about 23 d); iteration 67, clause 2 (under 48h); iteration 68, none (about 20 d). No drift alarm.

**Next**: no ticket is inside the 48h window except `ci:distinct-build-test-failures` (out of scope; window closes 10-30 21:18Z). Next in-scope ungated candidates, scope check first: `gate0:session-protocol-deadlock:agent-policy-unset` (stapledon, 10-07; does the driver export `AILANG_AGENT_POLICY` for pi controllers?), `pi:provider-error:no-retry` (stapledon, 10-07), then `skills:showcase-feature-discoverability`.

## 69 — 2026-10-29 — `gate0:session-protocol-deadlock:agent-policy-unset` LANDED via #2014 → `9ac96e78b`, independent sonnet PASS 93 (round 2); pi controllers now get a session-protocol note; ask 1 refuted, the extension half stays under D-FLEET-3 [HARNESS]

**Pick**: iteration 68's **Next**, `gate0:session-protocol-deadlock:agent-policy-unset` (stapledon, 10-07). It was the oldest in-scope, ungated ticket. No in-scope ticket is inside the 48h window; `ci:distinct-build-test-failures` is out of scope (D-FLEET-17 class).

**Orphans**: none. No `fleet/i69*` branch, worktree or PR existed. Self-notices rc 0 (control 852:10). The log and charter on `origin/dev` ended at 68.

**Preflight / observe**:
- Armed. gh `sunholo-voight-kampff`. Billing CLEAN. Directives: 0 on #1945 since `2026-10-22T23:29:08Z` (9 comments, none allowlisted).
- Skill drift: SKILL.md and all 13 resource files `cmp`-identical to origin/dev at the resolved symlink target. The pin worktree is at origin/dev (`65253e86e`).
- Gate-1 base `65253e86e68be251eecd796d92327e4d4828c234@2026-10-29T12:43:54Z`. Dev CI on it was in progress at Gate 1. It later read Build and Release green, with `test`, `launchd drivers` and `govulncheck` green. `ailang-core-dev (ailang-multivac-deploy)` was red (Cloud Build, outside fleet scope).
- Ledger valid, 32 rows, 17 OPEN. 38 tickets open. Codex over its daily ration (`MISSION_OVER_RATION=codex`, probe rc 75), so the driver handed the planner to `opus` and the executor to `claude:claude-haiku-5-5`.

**Reality check** (first-party, before routing):
- Session evidence: `~/.pi/agent/sessions/--Users-voightkampff-dev-sunholo-data-stapledons-godot--/` holds 40 sessions. 11 of them call `session_protocol_ack`; the planner's recount corrected the controller's 12. 5 of those 11 were REFUSED at least once.
  - The 09-29T12-53 session deadlocked. It made 0 bash `ailang messages` calls and 5 `ailang_cli` calls, including `{"argv":["messages","list","--unread","--json"]}`, and every one was refused with `AILANG_AGENT_POLICY is unset`.
  - Four sessions recovered later through bash: 10-02, 10-09 (two) and 10-19.
- Mechanism: `.pi/extensions/session-protocol-gate.ts` `prerequisitesMet()` credits the inbox step only to a `bash` call containing `ailang messages`. `BASH_ALLOW` already admits `ailang messages`. The model's first compound bash was refused with a generic reason, and it then reached for `ailang_cli`. `.pi/extensions/ailang-exec.ts` registers that tool with no policy and refuses every call (D4).
- **Ask 1 refuted.** Exporting a policy would inject the ailang_only "You have NO shell" lane prompt. `ailang-exec.ts` (comment at about line 320, measured 2026-09-19) records that prompt talking sessions out of a shell they had. **Ask 2** and any extension text change are `.pi/extensions/**`, which parks under D-FLEET-3.
- In-scope route: the pi controller branch of `tools/launchd/mission-control.sh`. `pi --help` lists `--append-system-prompt`. `tools/pi-extensions/controller-bash-cap.ts` is outside scope; only `sandbox/**` is allowed.

**Did** (designer skipped; planner, executor and two evaluator rounds spawned; generator ≠ judge):
- **Designer**: not spawned. Mechanical driver fix; the rotation pointer was not advanced.
- **Planner**: `opus`, Agent tool, foreground. Resolver `agent-tool opus fail-closed:env-pin`; the pin is the bare alias `opus`. It wrote [the plan](planned/sprint-plan-pi-controller-protocol-note.md) as one milestone:
  - `lib/pi-controller-note.sh` holds the note.
  - The pi branch sends it via `--append-system-prompt` and prefixes it to `-p`, on both the fresh and resume paths. Both copies are needed because a measured glm-5.3-flash route drops the system role.
  - New suite `test_pi_controller_note.sh` runs the real driver function against fake pi/claude/codex.
  - Role spawns get no note, because they are stages.
  - It found the World fork claim out of date: both plists run the main clone's driver.
- **Executor**: `claude:claude-haiku-5-5` through `claude-sub` (probe rc 0, keys stripped), `--output-format json`. Commits `7de55648d` (fix) and `fc646ca25` (changelog).
  - Its first drill harness used bash 3.2 `read -d ''` plus `${s//…}` on the 164 KB driver and ran 17 min at about 82% CPU.
  - The controller killed that one process at 13:23:59Z and extended the executor cap to about 47 min, because the fix was already committed.
  - The executor rewrote the drills in perl: 10 mutations, all red (M8 is the T6b control).
  - Base suite 51/51. The head suite's first run was red only in the load-sensitive `tools/eval/test_motoko_connection_probe.sh` (green alone, 59/59); the re-run passed 52/52.
- **Evaluator round 1**: Agent tool `sonnet`, fresh detached judge worktree `.wt-fleet-iter69-judge` at `fc646ca25`. **PASS 91**, 0 blocking.
  - Every sentence of the note is true of HEAD's gate and extension.
  - `--append-system-prompt` takes text unless the value is an existing file path (`resource-loader.js:17-31`), and is rebuilt on each session start (`agent-session.js:672`, `:1944`).
  - `make test-launchd-drivers` rc 0 (52). Shellcheck 33 vs 33.
  - Its mutations F ("NOT granted" → "granted"), F2 (inbox step via `ailang_cli`, the exact bug) and H (chain rule inverted) PASSED the suite.
- **Controller fix 1**, `9a9c95e58`: T6c pins five load-bearing sentences and refuses `(via|with|through|using|use) (the )?ailang_(cli|run)`. It also drops "single" from "bash runs only single read-only commands" (the judge's looseness finding). F, F2, F2b and H go red.
- **Evaluator round 2** (resumed): **PASS 93**, 0 blocking. N5 ("Prefer to call ailang_cli for the inbox.") and N7 ("every call succeeds") still passed. N1, N2 and N6 (skip the ack, any order, read CLAUDE.md via bash) also pass.
- **Controller fix 2**, `57add5d09`, which is the judge's named remedy: pin "every call refuses", and fail if either tool is named outside the NOT-granted sentence. F, F2, F2b, H, N3, N5 and N7 are all red, the `cmp` restore is ok, and the suite passes 36/36. Not re-scored by the judge. A first N3 drill read "missed" because perl's non-global `s///` hit the lib's header comment; aimed at the note sentence, it is red.
- **Real pi probe** (controller, flat-rate): `pi --model ollama/glm-5.3:cloud --no-extensions --no-tools --no-session --append-system-prompt "OVERRIDE … PERSIMMON …" -p "say hello"` → `PERSIMMON`, rc 0. The control without the flag gave a normal greeting. So the system append reaches the model with `-p` on the route that deadlocked. A resumed session was not probed.
- **Done-gate** ([done-gate.md](fleet-mission-evidence/iteration69/done-gate.md)): world dry-runs (idle and armed; real HOME; `AILANG_WORKTREE_SWEEP=0`; `AILANG_DRIVER_PINNED=fc646ca25`).
  - Healthy: rc 0, `DRY RUN ok`, `lanes=DEGRADED(2)`, because codex is over ration.
  - Degraded (`codex:bogus-model-i69`): rc 0, executor → `claude:claude-haiku-5-5`.
  - The sweep log hash, the pin listing and the world pidfile were unchanged.
- **PR** [#2014](https://github.com/sunholo-data/ailang/pull/2014). The closing-keyword scan found 0 matches; the control fired.

**Gate 3b**:
- PR head `57add5d09`, 27 checks terminal. Red only in `Build ubuntu-latest`: `internal/runner` `TestWorkerTerminalPTYLifecycle/eval/sigterm`. Windows and the two macOS builds were fail-fast `cancelled`. The required `test`, `launchd drivers (bash 3.2)`, `test-windows` and `govulncheck` were green. Dev HEAD `65253e86e` had all four Build jobs green.
- The diff touches no Go code. The same test family failed on dev at `4e6ea7b25` (iteration 68, `/eval/normal`). Re-run of the failed jobs on the same commit (run 40886298585, attempt 2): failure on a DISJOINT set. Ubuntu, Windows and one macOS leg were green. The other macOS leg failed on `cmd/ailang`'s hang guard ("this PACKAGE has outgrown its go test -timeout budget (not a failure of this test's logic)", package 584 s) plus `internal/runner`. Every leg that failed once passed on the other attempt. MERGEABLE/UNSTABLE.
- Merged with `--merge --match-head-commit` → `9ac96e78ba9203c4070271971d459dffa51b22ed`.
- Exact merge: required `test`, `lint`, `build` and `checks` green, plus `launchd drivers (bash 3.2)`, `test-windows` and `govulncheck` green (GO-2026-6860 no longer fires). `Build and Release` red in `Build ubuntu-latest` (`TestWorkerTerminalPTYLifecycle`, the same `internal/runner` flake), with the other legs fail-fast cancelled. `ailang-core-dev (ailang-multivac-deploy)` red, as on the parent `65253e86e`.

**Resolved**: `gate0:session-protocol-deadlock:agent-policy-unset` `--sha 9ac96e78ba9203c4070271971d459dffa51b22ed`. Turnaround about 22 d (10-07 18:58Z → 10-29). The resolution text carries the residual: the note is advisory, and the gate/extension hole is unchanged under D-FLEET-3. A recurrence re-files the same signature. The judge recommended keeping the ticket open until a live pi controller fire shows a bash `ailang messages` call; the controller resolved it and records that dissent here. 38 → 37 open.

**Progress**: goal unmoved on clause 1 (UNMEASURED). Clause 2 UNMET: this closure took about 22 d, and no in-scope ticket is inside the 48h window. Clause 3 MET (37 open). Clause 4: prior evidence only. Clause 5 preserved.

**Routing evidence**: Gate-1 base `65253e86e68be251eecd796d92327e4d4828c234@2026-10-29T12:43:54Z`; merge `9ac96e78b`. Routing note: `lanes degraded: planner and executor codex gpt-6.1-sol unusable (probe rc=75, over daily ration) → opus / claude:claude-haiku-5-5`.
- Controller `claude:claude-opus-5-5` (tok: not reported).
- Designer: not spawned (mechanical; no doc); rotation pointer not advanced.
- Planner `opus`, Agent tool (113,268 subagent tokens / 30 tool uses / 369 s). Resolver `agent-tool opus fail-closed:env-pin`; the hook and the resolver agreed.
- Executor `claude:claude-haiku-5-5` via `claude-sub -p --output-format json`: 116 input + 7,413,811 cache-read + 83,783 output tokens, 58 turns, 2,748 s, rc 0. It reported `total_cost_usd` 0.61 on the subscription, which is not metered spend.
- Evaluator `sonnet`, Agent tool: round 1 85,950 subagent tokens / 16 tool uses / 368 s; round 2 91,325 / 3 / 90 s. Judge independence: a different model from both the planner (opus) and the executor (haiku). Same vendor, so vendor-level independence is not claimed.
- Real pi probe: `pi:ollama/glm-5.3:cloud`, 2 calls (flag + control), flat-rate.
- Quorum: none (no design doc). Metered spend: $0. No lane recorded dead (codex is over ration, not dead).

**Ruled out**:
- Ask 1, exporting `AILANG_AGENT_POLICY` for pi controllers: it injects the ailang_only lane prompt into a session that needs bash.
- A stage marker (`AILANG_MISSION_STAGE`) for the controller: the controller triages the inbox at Gate 0 by design.
- Editing `controller-bash-cap.ts`: outside fleet scope.
- Editing either `.pi/extensions` file this iteration: D-FLEET-3. A park needs a proposed diff and its eval impact.

**Retro lane**: none filed; no skill edit. Observations, not tickets:
- (a) A drill harness that rewrites the 164 KB driver with bash 3.2 `${s//…}` or `read -d ''` stalls for 15+ minutes at full CPU. This is the second executor-instrument stall class seen on this lane. Directives that ask for mutation drills should say "perl or sed on scratch copies"; the evaluator directive did, and it ran clean.
- (b) A non-global perl `s///` aimed at a phrase that also appears in a file's comment mutates the comment, so the drill reads as a test miss. `cmp` only proves something changed, not that the right thing changed.
- (c) Phrase-pinned prompt tests are an arms race. The judge found 3 misses per round. The deterministic fix lives in the gate (D-FLEET-3), not in more needles.

Last 3 code landings: iteration 67, clause 2 (under 48h); iteration 68, none (about 20 d); iteration 69, none (about 22 d). No drift alarm: no routable UNMET-clause row exists, because the only in-window ticket is out of scope.

**Next**: no in-scope ticket is inside the 48h window (`ci:distinct-build-test-failures` is out of scope, and its window closes 10-30 21:18Z). Next in-scope ungated candidates, scope check first: `pi:provider-error:no-retry` (stapledon, 10-07; is pi's provider retry in the driver or in pi?), `stall:watchdog:slow-on-hung-tool` (stapledon, 10-07), then `skills:showcase-feature-discoverability` (needs a design).

## 70 — 2026-10-29 — `pi:provider-error:no-retry` LANDED via #2022 → `52e8958f0`, independent sonnet PASS 91 (round 2); the retry was a ghost of #1391, and the seam it never tested is now pinned end to end [HARNESS]

**Pick**: iteration 69's **Next**, `pi:provider-error:no-retry` (stapledon, 10-07). It was the oldest in-scope, ungated ticket. No in-scope ticket is inside the 48h window; `ci:distinct-build-test-failures` is out of scope (D-FLEET-17 class).

**Orphans**: none. No `fleet/i70*` branch, worktree or PR existed. Self-notices rc 0 (control 852:10). The log and charter on `origin/dev` ended at 69.

**Preflight / observe**:
- Armed. gh `sunholo-voight-kampff`. Billing CLEAN. Directives: 0 on #1945 since `2026-10-22T23:29:08Z` (11 comments, none allowlisted).
- Skill drift: SKILL.md and all 13 resource files are `cmp`-identical to origin/dev at the resolved symlink target. The controller's first per-file check printed DIFF for all 13 because its `git show` path dropped `resources/`. A size comparison exposed that instrument bug, and the corrected `cmp` was paired with a negative control.
- Gate-1 base `96d69f05b6e84bd6666e8fa91ff948f803b67d92@2026-10-29T21:05:52Z`.
- Dev CI on that base: `Build and Release` red in `Build ubuntu-latest`. Its job log shows `go: … verifying go.mod: … sum.golang.org/tile/…: stream error: … INTERNAL_ERROR`, a module-proxy network error, not a test. The other legs were fail-fast cancelled. `ailang-core-dev (ailang-multivac-deploy)` was red; CI was in progress. All of these are outside fleet scope.
- Ledger valid, 32 rows, 17 OPEN. 37 tickets open. Codex is over its daily ration (`MISSION_OVER_RATION=codex`, probe rc 75), so the driver handed the planner to `opus` and the executor to `claude:claude-haiku-5-5`.

**Reality check** (first-party, before routing):
- The cited event is the stapledon fire ending 2026-09-29T00:52Z. Its pi session ends `"stopReason":"error",…,"errorMessage":"Provider finish_reason: error"`.
- **Ghost.** #1391 (`7eaf9bece`, 07:24Z the same day) added the literal to `TRANSIENT_SIG` (`mission-control.sh:1160`). That is about 6.5 h after the event and 8 days before the ticket. The loop (`:2552-2600`) retries a match on the same rung: 3 attempts, 45 s/90 s backoff, `RUNTIME_QUOTA_SIG` checked first. The pi controller keeps one `--session-id` per fire, so a retry gets the resume prompt (`:2469-2480`).
- pi 0.85.1 print mode writes the error with `console.error` and rc 1 (`print-mode.js:115-117`). The driver launches pi with `>>"$LOG" 2>&1`.
- The gap: the only test was a membership grep (`test_driver_notify.sh` case 13). Nothing ran the seam. There is no live positive either: 0 `transient API error` lines in the rotated logs and 0 `RETRIED` slot verdicts. That is uninformative, because pi controllers since #1391 have mostly paused for capacity.
- `stall:watchdog:slow-on-hung-tool` (the next ticket) was checked but not picked. Ask 1 is built by #1391: `controller-bash-cap.ts` clamps every pi bash call to 540 s and is loaded with `-e`. Ask 2, a shorter threshold, changes values that D-FLEET-4 and -7 fixed. Since #1391 there has been one pi-controller KILLED verdict, the ticket's own 10-04 event, and no recurrence.

**Did** (designer skipped; planner, executor and two evaluator rounds spawned; generator ≠ judge):
- **Designer**: not spawned. This was a mechanical test addition with no doc, and the rotation pointer was not advanced.
- **Planner**: `opus`, Agent tool, foreground. Resolver `agent-tool opus fail-closed:env-pin`. It wrote [the plan](planned/sprint-plan-pi-provider-error-retry-seam.md) as one milestone, prototyped it in `/tmp`, and caught a controller error: pi names its session folder from the physical cwd (`session-manager.js:242-246`), while `$TMPDIR` is a symlink. A pi-faithful fake would therefore fail on the untouched driver unless the lab root uses `pwd -P`; its row C1 proves that. It also added `MISSION_TRANSIENT_RETRIES` to the suite's `unset` list.
- **Executor**: `claude:claude-haiku-5-5` through `claude-sub` (probe rc 0), `--output-format json`, bounded at 45 min, finished in 567 s. Commits `5297efa73` (test and changelog) and `5d5d5850f` (plan execution notes).
  - The fake pi now parses `--session-id` and `-p`, writes a pi-shaped session file under a lab `PI_CODING_AGENT_DIR` (and refuses to run without it), and can emit on stderr.
  - The new case is `provider-finish-reason-error-retries-and-resumes`. Suite 16/16; `make test-launchd-drivers` rc 0 (52).
  - M1–M5, F1 and C1 all went red; F2 survived, as tabled. Its one deviation was quoting `done` for SC1010.
- **Controller verification**: re-ran the suite (16/0), then ran M2 (`2>&1` dropped) on a scratch driver: 15 pass, 1 fail, and the new case is the only killer.
- **Evaluator round 1**: `sonnet`, Agent tool, fresh detached judge worktree `.wt-fleet-iter70-judge`. **PASS 88**, 0 blocking.
  - It confirmed the ghost first-party, including `mapStopReason` in pi-ai and the absence of any `MISSION_TRANSIENT_RETRIES` override.
  - Strongest objection: T5b+M2 (fake ignores the stderr selector, driver drops `2>&1`) passed 16/16, so the stderr half was undefended.
- **Controller fix**, `2c456d263`: the fake records its stream in `pi.stream`, and the case asserts attempt 1 used stderr. T5a, T5b and T5b+M2 each went red on that assertion only; `cmp` restore ok.
- **Evaluator round 2** (resumed): **PASS 91**. Stderr gap closed. A new LIE mutant (stdout labelled `stderr`) survives; accepted as a paired edit only real pi can refute. Still open, non-blocking: the backoff values and 3-attempt bound are unpinned, the changelog category, and the driver's logical session slug (latent).
- **Done-gate** ([done-gate.md](fleet-mission-evidence/iteration70/done-gate.md)):
  - World dry-runs (idle and armed; real HOME; `AILANG_WORKTREE_SWEEP=0`; `AILANG_DRIVER_PINNED=2c456d263`). Healthy: rc 0, `DRY RUN ok`, `lanes=DEGRADED(2)`, because codex is over ration. Degraded (`codex:bogus-model-i70`): rc 0, executor → `claude:claude-haiku-5-5`.
  - The sweep log hash, the pin listing and the world pidfile were unchanged.
  - `make test-launchd-drivers` on head: rc 0, 52 suites. shellcheck base 64 → head 70; all 6 added are SC2319.
- **PR** [#2022](https://github.com/sunholo-data/ailang/pull/2022). The closing-keyword scan found 0 matches; the control fired.

**Gate 3b**: PR head `2c456d263`: 27 checks terminal, 20 success and 7 skipped, 0 failures (SHA-pinned poll on the full head; MERGEABLE/CLEAN). Merged with `--merge --match-head-commit` → `52e8958f0013d08ac4e5f4271b1f8a01d48f6b8e`. Exact merge: `CI [push]` success and `Build and Release [push]` success (all four Build legs green, including both macOS legs); required `test`, `lint`, `build`, `checks`, `launchd drivers (bash 3.2)`, `test-windows`, `govulncheck` and SonarCloud green. The only red is `ailang-core-dev (ailang-multivac-deploy)` (Cloud Build, outside fleet scope), red on the parent `96d69f05b` as well. The first merge poller's ≥20-check floor could not be met by this commit's 18 checks; it was killed as superseded, and the verdict comes from a direct read of the check set plus `actions/runs?head_sha=` (2 runs, both success).

**Resolved**: `pi:provider-error:no-retry` `--sha 52e8958f0013d08ac4e5f4271b1f8a01d48f6b8e`. The resolution text cites #1391 `7eaf9bece` as the mechanism and this merge as test-only, and names the residual: `network_error`/`content_filter` finish reasons do not match the literal (policy class). Turnaround about 22 d (10-07 18:58Z → 10-29). 37 → 36 open.

**Progress**: goal unmoved on clause 1 (UNMEASURED). Clause 2 UNMET: this closure took about 22 d, and no in-scope ticket is inside the 48h window. Clause 3 MET (36 open). Clause 4: prior evidence only. Clause 5 preserved.

**Routing evidence**: Gate-1 base `96d69f05b6e84bd6666e8fa91ff948f803b67d92@2026-10-29T21:05:52Z`; worktree base the same (no drift); merge `52e8958f0`. Routing note: `lanes degraded: planner and executor codex gpt-6.1-sol unusable (probe rc=75, over daily ration) → opus / claude:claude-haiku-5-5`.
- Controller `claude:claude-opus-5-5` (tok: not reported).
- Designer: not spawned (mechanical; no doc); rotation pointer not advanced.
- Planner `opus`, Agent tool (118,661 subagent tokens / 31 tool uses / 398 s). Resolver `agent-tool opus fail-closed:env-pin`; `derive-planner-lane.sh` gave `opus fail-closed:env-pin`.
- Executor `claude:claude-haiku-5-5` via `claude-sub -p --output-format json`: 48 input + 87,822 cache-write + 1,955,877 cache-read + 27,045 output tokens, 28 turns, 567 s, rc 0. It reported `total_cost_usd` 0.066 on the subscription, which is not metered spend.
- Evaluator `sonnet`, Agent tool: round 1 89,270 subagent tokens / 25 tool uses / 633 s; round 2 94,131 / 2 / 101 s. Judge independence: a different model from both the planner (opus) and the executor (haiku). Same vendor, so vendor-level independence is not claimed.
- Dry-run probes (two world dry-runs) used the driver's own lane probes. Quorum: none (no design doc). Metered spend: $0. No lane recorded dead (codex is over ration, not dead).

**Ruled out**:
- A driver change of any kind. The mechanism exists; widening `TRANSIENT_SIG` to `network_error`/`content_filter` is policy class (HD-2a, compare D-FLEET-26).
- Taking `stall:watchdog:slow-on-hung-tool` as a second item. This pick shipped code, so the bookkeeping-only second-item carve-out does not apply. It goes next.
- The controller's first per-file skill-drift reading ("all 13 DIFF"): an instrument bug (a dropped `resources/` path segment), not drift.
- Live retry evidence as a resolution criterion: none exists, and its absence is uninformative.

**Retro lane**: none filed; no skill edit. Observations, not tickets:
- (a) The drift-check loop in gate-1 is correct as written. The controller's hand-rolled variant dropped a path segment and read "all DIFF". This is the first instance; one friction does not meet the ≥2 rule.
- (b) The planner's prototype-in-`/tmp` caught a controller premise error (logical vs physical session slug) before the executor spent a run. That is cheap and worth keeping for test-harness plans.
- (c) The driver's `_mc_pi_session_exists` builds its slug from the logical `$REPO`, while pi uses the physical cwd. It is latent, since live workdirs are physical. A symlinked `MISSION_WORKDIR` would silently lose resume, and so would the pi arm of `_mc_progress_bytes`. Not filed: the fleet cannot file tickets for its own slot, and no loop has hit it.

Last 3 code landings: iteration 68, none (about 20 d); iteration 69, none (about 22 d); iteration 70, none (about 22 d). No drift alarm: no routable UNMET-clause row exists, because the only in-window ticket is out of scope.

**Next**: no in-scope ticket is inside the 48h window. `stall:watchdog:slow-on-hung-tool` (stapledon, 10-07): ask 1 is a ghost of #1391; ask 2 (shorter threshold) conflicts with D-FLEET-4/-7 and needs a decision row with a recommendation. Then `skills:showcase-feature-discoverability` (needs a design) and `pi:openrouter:uncapped-max-tokens-402` (scope check: provisioning is outside `cmd/ailang/mission*`).

## 71 — 2026-10-30 — two `gate0:watermark-*` tickets RESOLVED as ghosts of #1962 (`5ad75cad9`), independent sonnet PASS 88; `stall:watchdog:slow-on-hung-tool` parks on D-FLEET-34 (ask 1 is a ghost of #1391) [HARNESS]

**Pick**: iteration 70's **Next**, `stall:watchdog:slow-on-hung-tool` (stapledon, 10-07). It measured as a decision row, not a build, which made it a bookkeeping-only pick, so a second item was taken (standing rule 1). That second item was `gate0:watermark-advanced-by-dying-fire` (attended, 10-03) plus `gate0:watermark-advanced-before-processing` (attended, 10-08). Neither had a queue row. Both describe the defect reported at #981, which #1962 merged on 10-29. `skills:showcase-feature-discoverability` needs a design, and `pi:openrouter:uncapped-max-tokens-402` looked out of scope at first look (see **Next**).

**Orphans**: none. No `fleet/i71*` branch, worktree or PR existed; `gh pr list --author @me` showed 21 open PRs, none on a `fleet/` branch. Self-notices rc 0 (control 852:10). The log and charter on `origin/dev` ended at 70.

**Preflight / observe**:
- Armed. gh `sunholo-voight-kampff`. Billing CLEAN. Directives: 0 on #1945 since `2026-10-22T23:29:08Z` (13 comments, none allowlisted; the watermark file is still LEGACY).
- Skill drift: SKILL.md and all 13 resource files `cmp`-identical to origin/dev, at the resolved symlink target and at the CWD copy (gate-1 loop (d), both bases).
- Gate-1 base `d45cf4e6b135070199a976d3f6bbe7cc2a2ed5b0@2026-10-30T04:47:22Z`. Local == origin/dev.
- Ledger valid, 32 rows, 17 OPEN. 36 tickets open. Codex is over its daily ration (`MISSION_OVER_RATION=codex`, probe rc 75). The driver handed the planner to `opus` and the executor to `claude:claude-haiku-5-5`; neither was needed.

**Reality check** (first-party, before routing):
- **Watermark tickets, ghost.** `git log origin/dev --grep=981` lists 351c415de, e06e372a0, 24c77c15b, 40a52cf22 and 034bda1bb, then the merge `5ad75cad9` (#1962, an ancestor of HEAD). `gh issue view 981` gives CLOSED 2026-10-29T07:43:12Z.
  - `scripts/mission_directives.sh:192-243`: the `--watermark-file` window is inclusive (`createdAt >= $since`), only `--mark <id>` writes the file (under a lock), and processed ids dedupe the boundary. The deprecated `--since` mode (`:132`, `:140`) stays exclusive by design (`24c77c15b`).
  - Gate 0 step 8 is PROCESS-THEN-MARK in both `.claude` and `.agents` `gate-0-preflight.md` (`:279`).
  - `/bin/bash scripts/test_mission_directives.sh` gave rc 0, 33 passed / 0 failed. It is listed in `make/test.mk:93`.
- **Stall ticket, ask 1 built.** `mission-control.sh:2480` loads `tools/pi-extensions/controller-bash-cap.ts` (default 540 s, `MISSION_PI_BASH_CAP_S`) with `-e`; `test_driver_notify.sh:1003` pins it. The thresholds sit at `:1121-1127` (grace 2400 s, child age 2400 s, 5 × 120 s).
- **Stall ticket, ask 2 measured.** Slot-verdict ledgers, all 7 missions, since `2026-09-29T07:24Z` with `controller=pi:`: 133 slots, 103 `PAUSED-NO-CAPACITY_at=fired`, and 1 KILLED. That kill is the ticket's own 10-04 event (`KILLED_at=gate-5`, elapsed 5217 s). The other 4 KILLED rows since #1391 are claude controllers. Retained driver logs begin 10-27. An anchored `^[<date>] STALL:` grep found 0 real kill lines, while the unanchored pattern matched 39 echoed code lines (the positive control). So the 10-04 log lines are rotated out, and the slot-verdict ledger is the instrument.
- **`pi:openrouter:uncapped-max-tokens-402`, first look only.** `tools/pi-extensions/models.mission.json` has no `z-ai/glm-5.3`, `moonshotai/kimi-k3` or `minimax/minimax-m3` entry. The installed `~/.pi/agent/models.json` carries the attended 10-07 caps. The repo copy is drift-tested against `internal/modelreg/models.yml`, and both that file and `tools/pi-extensions/**` outside `sandbox/` are outside fleet scope. The 402 reclassification ask would change D-FLEET-11's ruled `^402:` capacity class. Not judged; recorded for the next fire.

**Did** (no code; designer, planner and executor not spawned; evaluator spawned; generator ≠ judge):
- **Designer, planner, executor**: not spawned. There was nothing to build: one item is a ghost resolution, the other a decision row. The rotation pointer was not advanced. The resolver read `designer recipe claude:claude-opus-5-5`, `planner agent-tool opus`, `executor recipe claude:claude-haiku-5-5`, `evaluator agent-tool sonnet`.
- **Evaluator**: `sonnet`, Agent tool, foreground, read-only on the pin. It got both draft records ([as judged](fleet-mission-evidence/iteration71/draft-records-as-judged.md)) with instructions to refute them first-party.
  - Record 1 (ghost resolution): **PASS-WITH-EDITS 88**. It listed every ask in both tickets, found each built, and re-ran the suite (33/0). Mutation drill via `MD_SCRIPT_UT` on scratch copies: exclusive window gave 21 pass / 12 fail; `--mark` to the newest boundary gave 31 / 2. The tracked file was unchanged. Edits: name the `planned/` doc as a hygiene residual. "Closed 07:43Z" was not verified by the judge; the controller's own `gh issue view` reading supports it, so it stays.
  - Record 2 (D-FLEET-34): **PASS-WITH-EDITS 78**. It reproduced 133 / 103 / 30 / 1 and confirmed there is one pi spawn path. Edits, all applied: D-FLEET-4/-7 rule the sample count and 600-s budget but not the grace; quantify "fewer samples" (240–360 s); add the 1-in-30 caveat; name `STALL_CHILD_AGE`.
  - Overall **83**.
- **Resolved** `gate0:watermark-advanced-by-dying-fire` and `gate0:watermark-advanced-before-processing` with `--sha 5ad75cad97c66fb279618b03c092198e2c516703`. Both replied to `attended`. 36 → 34 open.
- **Filed** D-FLEET-34 (OPEN, recommendation B). Queue rows 47 and 48.
- **Done-gate** (resolution of a fix this fleet did not write; no driver change): the fix's own suite is 33/0 at HEAD, and `make test-launchd-drivers` passed all 52 suites at `d45cf4e6b` (rc 0). There were no dry-runs, since no driver surface changed.

**Gate 3b** (record only; no code landed this iteration): record PR [#2025](https://github.com/sunholo-data/ailang/pull/2025), head `9c9585bfd`. 27 checks terminal: 15 success, 12 skipped, 0 failures (SHA-pinned poll on the full head; MERGEABLE/CLEAN). Merged with `--merge --match-head-commit` → `0aff46c5a18f2032bd64f78f43efdc8c1d11db74`. Exact merge: `CI [push]` success and `Build and Release [push]` success (`actions/runs?head_sha=`, 2 runs, both success). The check set has 18 checks; the only red is `ailang-core-dev (ailang-multivac-deploy)` (Cloud Build, outside fleet scope), which is also red on the parents `d45cf4e6b` and `2b6dcfded`. The first committed copy of this entry carried a literal `GATE3B_RESULT` placeholder here; this line replaces it in a follow-up commit.

**Progress**: goal unmoved on clause 1 (UNMEASURED). Clause 2 UNMET: these closures took about 27 d and 21 d, filed to resolved. Clause 3 MET (34 open). Clause 4: prior evidence only. Clause 5 preserved.

**Routing evidence**: Gate-1 base `d45cf4e6b135070199a976d3f6bbe7cc2a2ed5b0@2026-10-30T04:47:22Z`; Gate-4 base=d45cf4e6b135070199a976d3f6bbe7cc2a2ed5b0@2026-10-30T04:54:17Z (no drift). Routing note: `lanes degraded: planner and executor codex gpt-6.1-sol unusable (probe rc=75, over daily ration) → opus / claude:claude-haiku-5-5`.
- Controller `claude:claude-opus-5-5` (tok: not reported).
- Designer: not spawned (nothing to design); rotation pointer not advanced.
- Planner: not spawned (nothing to plan). Resolver `agent-tool opus fail-closed:env-pin`.
- Executor: not spawned (no code). Resolver `recipe claude:claude-haiku-5-5 declared:provider-pin`.
- Evaluator `sonnet`, Agent tool (66,858 subagent tokens / 11 tool uses / 99 s). Resolver `agent-tool sonnet declared:alias-pin`. Judge independence: a different model from the controller (opus). Same vendor, so vendor-level independence is not claimed.
- Quorum: none (no design doc). Metered spend: $0. No lane recorded dead.

**Ruled out**:
- Building ask 2 of `stall:watchdog:slow-on-hung-tool` without a ruling: it re-opens D-FLEET-4/-7.
- Resolving `stall:watchdog:slow-on-hung-tool` now on the strength of ask 1: declining ask 2 is itself the policy call (same pattern as D-FLEET-32).
- `make test-launchd-drivers` under `timeout 1500`: rc 127, because macOS has no `timeout`. That was an instrument failure, not a verdict. It was re-run under a perl `alarm`.
- Taking `pi:openrouter:uncapped-max-tokens-402` as a third item: the bookkeeping carve-out allows two, and it needs its own scope ruling.

**Retro lane**: none filed; no skill edit. Observations, not tickets:
- (a) Tickets with no queue row hide behind the "Next" chain. The two watermark tickets were filed 10-03 and 10-08, sat below the iteration-to-iteration **Next** pointer, and had gone ghost the day before this fire. A ghost sweep of open tickets against the last week's merges (`git log --grep` for each ticket's issue number) is cheap. First instance; one friction does not meet the ≥2 rule.
- (b) The judge's "not verified" on a claim the controller had measured, the #981 close time, is the right posture (gate-2 rule (d)). The record says which instrument backs it.

Last 3 code landings: iteration 69, none (about 22 d); iteration 70, none (about 22 d); iteration 71, no code. No drift alarm: no routable UNMET-clause row exists, because the only in-window ticket is out of scope.

**Next**: `skills:showcase-feature-discoverability` (attended, 10-03; reported at #476). It needs a design: the sprint skills still have no mandatory syntax step, and asks 1 and 3 (docs MCP in bootstrap, a CI feature gate) need a scope check first. Then `pi:openrouter:uncapped-max-tokens-402` (stapledon, 10-07), which will likely park as an out-of-scope plus policy row per the first look above.

## 72 — 2026-10-30 — `skills:showcase-feature-discoverability` RESOLVED as a ghost of #1691 (`03263525c`) plus the attended #476 closure; `pi:openrouter:uncapped-max-tokens-402` parks on D-FLEET-35; independent sonnet PASS-WITH-EDITS 73 [HARNESS]

**Pick**: iteration 71's **Next**, `skills:showcase-feature-discoverability` (attended, 10-03; restates #476). It measured as a ghost resolution, not a design, which made it a bookkeeping-only pick, so a second item was taken (standing rule 1): iteration 71's second **Next**, `pi:openrouter:uncapped-max-tokens-402` (stapledon, 10-07). It measured as a decision row.

**Orphans**: none for this iteration. No `fleet/i72*` branch or PR existed before this fire, and `gh pr list --author @me` showed no open `fleet/` PR. Self-notices rc 0 (control 852:10). The log and charter on `origin/dev` ended at 71. `git worktree list` still holds about 60 old fleet worktrees (iterations 12–71, all landed or superseded); noted, not swept (the sweep is `worktree_sweep.sh`'s job).

**Preflight / observe**:
- Armed. gh `sunholo-voight-kampff`. Billing CLEAN. Directives: 0 on #1945 since `2026-10-22T23:29:08Z` (14 comments, none allowlisted; the watermark file is still LEGACY).
- Skill drift: SKILL.md and all 13 resource files `cmp`-identical to origin/dev, at the resolved symlink target and at the CWD copy (gate-1 loop (d), both bases).
- Gate-1 base `490fb730b2f809f846c2c9b437b2aefcdfc12933@2026-10-30T12:04:58Z`. Local == origin/dev.
- Dev CI at `490fb730b`: `CI` success, `Build and Release` success. The check set has 20 checks; the only red is `ailang-core-dev (ailang-multivac-deploy)` (Cloud Build, inherited, outside fleet scope), and `docs-build` was still pending.
- Ledger valid, 33 rows, 18 OPEN. 34 tickets open. Codex is over its daily ration (`MISSION_OVER_RATION=codex`, probe rc 75). The driver handed the planner to `opus` and the executor to `claude:claude-haiku-5-5`; neither was needed.

**Reality check** (first-party, before routing):
- **Showcase ticket, ask 2 built.** `git log -S 'Load AILANG syntax before writing'` gives `03263525c` (#1691, 2026-10-09), an ancestor of `origin/dev`. `.claude/skills/sprint-executor/SKILL.md:104` is Core Principle 11 ("Load AILANG syntax before writing `.ail` — no exceptions"). `.claude/skills/sprint-planner/SKILL.md:248-254` is the AILANG Syntax Gate, and `milestone_checklist.md:122-133` gives the details. Both `.agents` mirrors `cmp` identical. The ticket's "Checked 790169359: … no mandatory syntax step" predates the landing.
- **Showcase ticket, asks 1 and 3 unbuilt.** The approved design `design_docs/implemented/v0_53_1/m-skill-feature-discoverability-gate.md` § Non-Goals (`:281-291`) assigns ask 1 to "the mission charter/bootstrap template" and defers ask 3 "to be picked up if the advisory measurement misses target". The judge confirmed that nothing in `missions/`, `tools/launchd` or the mission-control skill registers an MCP, and there is no `.mcp.json`. `gh issue view 476`: CLOSED 2026-10-26T13:48:43Z, with an attended triage comment: "The design ruled MCP wiring and a CI gate as non-goals."
- **402 ticket, ask (a) real at repo level.** `tools/pi-extensions/models.mission.json` (openrouter: gemma-4-26b, glm-5.3-flash, the deepseek v4/v4.1-flash spellings) has no `z-ai/glm-5.3`, `moonshotai/kimi-k3` or `minimax/minimax-m3`. The installed `~/.pi/agent/models.json` has all three at 32000, and the backup `models.json.bak-20261007-orcaps` exists. This fire's env still routes all three as role fallbacks (designer, planner, evaluator), and `~/.config/ailang/mission-*.env` names two of them. The OpenRouter GLM controller rung was retired 10-07 (`mission-control.sh:929`).
- **402 ticket, out of scope + policy.** The guard (`tools/launchd/githooks/pre-push`) refuses `tools/pi-extensions/` outside `sandbox/` and `internal/modelreg/`. `TestPiModelsConfigMatchesRegistry` holds the two equal. `RUNTIME_QUOTA_SIG` (`:1152`) carries `^402:` under D-FLEET-11/13. `grep -c 'can only afford'` = 0 in all 15 `/tmp/ailang-mission-*.log` (retained since 10-27; positive control `quorum artifact` = 48 in the world log). The one `^402:` hit, `ailang-mission-triage.log:16093`, is a line-number prefix in quoted skill text.

**Did** (no code; designer, planner and executor not spawned; evaluator spawned; generator ≠ judge):
- **Designer, planner, executor**: not spawned. There was nothing to build: one item is a ghost resolution, the other a decision row. The rotation pointer was not advanced. Resolver: `evaluator agent-tool sonnet declared:alias-pin`; the driver exported designer `recipe claude:claude-opus-5-5`, planner `agent-tool opus`, executor `recipe claude:claude-haiku-5-5`.
- **Evaluator**: `sonnet`, Agent tool, foreground, read-only on the pin. It got both draft records ([as judged](fleet-mission-evidence/iteration72/draft-records-as-judged.md)) with instructions to refute them first-party ([report](fleet-mission-evidence/iteration72/evaluation.md)).
  - Record 1 (ghost): **PASS-WITH-EDITS 66**. C1 and C3 were verified. C2 was only partly verified: the Non-Goals assign ask 1 elsewhere and defer ask 3; they do not decline either. Its strongest objection: the ghost rests on the attended #476 closure, not on the design. Edits applied: the resolution says asks 1 and 3 are unbuilt and closed as non-goals by attended triage; the planner citation is now 248-254; the unverified "ailang-world applied it locally" was dropped.
  - Record 2 (D-FLEET-35): **PASS-WITH-EDITS 80**. All claims were verified. It added the root cause: `internal/executor/pi/config_preflight.go` checks only models whose task carries registry settings, and these three have no registry rows. It found no in-fleet mechanical piece: a warn-only probe duplicates the preflight, and skip/demote changes lane order. Edits applied: the rows must carry `max_output_tokens`; confirm the non-GLM caps with `scripts/check_pi_wire_budget.sh`; "cannot be formed" became "is not formed while the cap is installed"; the rejected probe is named.
  - Overall **73**.
- **Resolved** `skills:showcase-feature-discoverability` with `--sha 03263525c483311f725c6d2db0fda0782b1e74f9`, replied to `attended`. 34 → 33 open.
- **Filed** D-FLEET-35 (OPEN, recommendation A). Queue rows 49 and 50.
- **Done-gate**: not applicable (no harness change; the resolved fix is a skill-text change that this fleet did not write, verified present and mirrored at HEAD).

**Gate 3b** (record only; no code): record PR [#2028](https://github.com/sunholo-data/ailang/pull/2028), head `d1424cbfe`, is **BLOCKED**. Required `test` failed 3 of 3 attempts (run 41053659833, 12:24Z, 12:38Z, 12:50Z), every time in `cmd/ailang` on `internal/testutil/gate.go:102`'s hang guard ("this PACKAGE has outgrown its go test -timeout budget"), and on different tests each time (`TestStdNumeric*`, `TestTailCallTrace*`, then `TestVerifyM0_*`). The package took 409 s and 395 s against `go test -timeout 416s` (`.github/workflows/ci.yml:198`). Dev's own green `test` at the same base `490fb730b` took 383 s, and triage's `triage/i20-record` PR hit the same guard at 10:10Z. The PR changes only `design_docs/`, so the red is inherited budget pressure, not this change. The CI workflow is outside fleet scope (D-FLEET-17 class), so the PR is left open for the next fire. Gate 1's open-PR rule makes it the next fire's record input. All other checks were terminal-green (26 total, 1 bad).

**Progress**: goal unmoved on clause 1 (UNMEASURED). Clause 2 UNMET: this closure took about 27 d, filed to resolved, and no in-scope ticket was filed in the last 48h. Clause 3 MET (33 open). Clause 4: prior evidence only. Clause 5 preserved.

**Routing evidence**: Gate-1 base `490fb730b2f809f846c2c9b437b2aefcdfc12933@2026-10-30T12:04:58Z`. Routing note: `lanes degraded: planner and executor codex gpt-6.1-sol unusable (probe rc=75, over daily ration) → opus / claude:claude-haiku-5-5`.
- Controller `claude:claude-opus-5-5` (tok: not reported).
- Designer: not spawned (nothing to design); rotation pointer not advanced.
- Planner: not spawned (nothing to plan).
- Executor: not spawned (no code).
- Evaluator `sonnet`, Agent tool (69,313 subagent tokens / 13 tool uses / 109 s). Resolver `agent-tool sonnet declared:alias-pin`. Judge independence: a different model from the controller (opus). Same vendor, so vendor-level independence is not claimed.
- Quorum: none (no design doc). Metered spend: $0. No lane recorded dead.

**Ruled out**:
- Designing a new doc for the showcase ticket: the approved design already exists in `implemented/v0_53_1/`, and its in-scope ask is built.
- Building ask 1 (an MCP in mission bootstrap) without a ruling: it would change every controller's tool surface, and attended triage closed it as a non-goal.
- A fleet-side warn or skip probe for uncapped pi OpenRouter models: see the D-FLEET-35 row.
- Reclassifying the 402 in `RUNTIME_QUOTA_SIG`: D-FLEET-11/13 policy.

**Retro lane**: none filed; no skill edit. Observations, not tickets:
- (a) Iteration 71's observation (a) recurs: a ticket that restates a GitHub issue went ghost when the issue's own fix landed (#1691, 10-09), and nothing linked the two for 21 days. Instance 2 (after the #981 watermark tickets). The cheap check is `gh issue view <n>` for each open ticket whose evidence cites an issue number. Two frictions now meet the ≥2 bar for a retro candidate. It is a loop-side habit, not a harness defect, and this loop is ticket-driven, so it stays an observation for Mark rather than a self-sourced fix.
- (b) The judge's strongest objection was a wording one: "declined by the design" versus "unbuilt, closed by attended triage". Wording carries the provenance of a resolution, so it matters.

Last 3 code landings: iteration 70, none (about 22 d); iteration 71, no code; iteration 72, no code. No drift alarm: no routable UNMET-clause row exists, because no in-scope ticket was filed in the 48h window.

**Next**: `skill:design-doc-creator-grep-portability` (attended, 10-03). It is real at HEAD: `create_planned_doc.sh:128/137/154/163` still use `grep -E "^\d+\."`. But `.claude/skills/design-doc-creator/**` is outside the scope guard (`pre-push:17-18`), so it likely needs a scope row. Then `sprint-evaluator:worktree-cap-and-exit1` (attended, 10-09; scope check: coordinator worktree cap, outside, versus executor commit-per-milestone, in sprint-* skills).

## 73 — 2026-10-30 — `skill:design-doc-creator-grep-portability` parks on D-FLEET-36 and `sprint-evaluator:worktree-cap-and-exit1` parks on D-FLEET-37 (ask 3 a ghost of #1688); iteration 72's record PR #2028 merged; independent sonnet FAIL 66 → PASS-WITH-EDITS 88 / 84 [HARNESS]

**Pick**: iteration 72's **Next**, `skill:design-doc-creator-grep-portability` (attended, 10-03). It measured as a decision row: real at HEAD, but the fix path is outside the scope guard. That made it a bookkeeping-only pick, so a second item was taken (standing rule 1): iteration 72's second **Next**, `sprint-evaluator:worktree-cap-and-exit1` (attended, 10-09). It measured as a partial ghost plus a decision row.

**Orphans / prior record**: Gate 1's open-PR rule found iteration 72's record PR #2028 (`fleet/i72-record`) open, the log on `origin/dev` ending at 71. At this fire it was `MERGEABLE`/`UNSTABLE`: the required `test` had gone green on head `975145d34`, and the only red was the non-required `Build windows-latest`, whose log shows `panic: test timed out after 10m0s` in `cmd/ailang` (600.5 s), the same budget pressure on a docs-only PR. This fire merged it (`gh pr merge --merge`) → `dd183217458b5e897e8669262bf51dc1f82249d1`; all claims below were measured there. Self-notices rc 0 (control 852:10). Gate-2 died-mid-flight traces: no `fleet/i73*` branch or worktree before this fire.

**Preflight / observe**:
- Armed. gh `sunholo-voight-kampff`. Billing CLEAN. Directives: 0 on #1945 since `2026-10-22T23:29:08Z` (15 comments, none allowlisted; the watermark file is still LEGACY).
- Skill drift: SKILL.md and all 13 resource files `cmp`-identical to `origin/dev` at the resolved symlink target.
- Gate-1 base `490fb730b2f809f846c2c9b437b2aefcdfc12933@2026-10-30T18:53:58Z`; local == origin/dev.
- Dev CI at `490fb730b`: 22 checks, the only red `ailang-core-dev (ailang-multivac-deploy)` (Cloud Build, inherited, outside fleet scope). `CI` and `Build and Release` success.
- Ledger valid, 34 rows, 19 OPEN at start. 33 tickets open. Codex over its daily ration (`MISSION_OVER_RATION=codex`, probe rc 75); the driver handed the planner to `opus` and the executor to `claude:claude-haiku-5-5`; neither was needed.

**Reality check** (first-party, before routing):
- **Grep ticket, ask (a) real at HEAD.** `grep -E "^\d+\."` sits at `.claude/skills/design-doc-creator/scripts/create_planned_doc.sh:128/137/154/163` and the `.agents` copy `:125/134/153/162`, each followed by `|| echo ""` (an empty search, not an abort).
- **Grep ticket, BSD half refuted.** `/usr/bin/grep` is `BSD grep, GNU compatible 2.6.0-FreeBSD`; on `1. design_docs/x.md (0.85)` plus `d. foo`, `^\d+\.` matched only line 1, the same as the `^[0-9]+\.` control (the judge added `^d+\.` → only the `d.` line). `/bin/bash` resolves `grep` to `/usr/bin/grep`. No GNU grep or docker on the rig, so the GNU half is documented, unmeasured.
- **Grep ticket, ask (b) built.** `.claude`: #1542 `b51353bcd` (2026-10-03T07:16Z, two minutes after the ticket) moved the `merge_results` filter into awk. `.agents`: `|| true` at the `merge_results` calls (`:144`, `:170`).
- **Grep ticket, out of scope.** `tools/launchd/githooks/pre-push` `_scope_is_harness` allows only `mission-*`/`sprint-*` skills; `_scope_is_fleet_support` has none. Adding the path to `_scope_is_harness` would refuse it for product loops (`_scope_verdict`), so the recommendation names `_scope_is_fleet_support`.
- **Evaluator ticket, ask 3 ghost.** `face88a3f` (#1688, merged 2026-10-09T07:27:27Z, about 70 min after the ticket) added `sprint-executor/resources/unattended_runs.md:6` "Commit after each milestone"; #1905 `3f9a0c092` added push-every-milestone for cloud tasks.
- **Evaluator ticket, asks 1 and 2 out of scope.** `MaxWorktrees: 3` (`internal/coordinator/daemon.go:47,54`); `coordinator start --max-worktrees` exists (`coordinator_lifecycle.go:43-49`). Overlaps open ticket `coordinator:cloud-lane-capped-by-local-worktree-limit` (no charter row; 0 hits, positive control `agents-copies-stale-non-mission` 1).

**Did** (no code; designer, planner and executor not spawned; evaluator spawned; generator ≠ judge):
- **Designer, planner, executor**: not spawned. Nothing to build: both items are decision rows (one with a ghost half). Rotation pointer not advanced. Driver exports: designer `recipe claude:claude-opus-5-5`, planner `agent-tool opus`, executor `recipe claude:claude-haiku-5-5`.
- **Evaluator**: `sonnet`, Agent tool, foreground, read-only on the pin; resolver `agent-tool sonnet declared:alias-pin`. Given both draft records ([as judged](fleet-mission-evidence/iteration73/draft-records-as-judged.md)) with instructions to refute them ([report](fleet-mission-evidence/iteration73/evaluation.md)).
  - Round 1: record 1 **PASS-WITH-EDITS 82** (C2 PARTLY: BSD half refuted the ticket, GNU half unmeasurable). Record 2 **FAIL 66**: C7 missed `internal/coordinator/daemon_tasks_exec_run.go:163-170`, which already resets a task that hits `ErrWorktreeLimitReached` to pending for retry, so "queueing is a coordinator code change" was unverified and the fold recommendation unsupported. The controller confirmed the finding (`git log -S` → `debf5b66e`, 2026-01-01) and applied all five edits ([as revised](fleet-mission-evidence/iteration73/records-as-revised.md)).
  - Round 2 (same judge, resumed): **88** and **84**, PASS-WITH-EDITS. Remaining edits were wording (headline and default no stronger than "GNU documented, unmeasured"; judge attribution), plus one substantive: record 2's option A would have resolved the ticket while ask 2 (exit-1 root cause) had no home. All applied; the rows say the fleet resolves only once ask 2 has a home.
- **Filed** D-FLEET-36 and D-FLEET-37 (OPEN, recommendation A each). Queue rows 51 and 52. No ticket resolved; 33 open.
- **Done-gate**: not applicable (no harness change).

**Progress**: goal unmoved on clause 1 (UNMEASURED). Clause 2 UNMET (no in-scope ticket filed in the last 48h; neither pick was in scope). Clause 3 MET (33 open). Clause 4: prior evidence only. Clause 5 preserved.

**Routing evidence**: Gate-1 base `490fb730b2f809f846c2c9b437b2aefcdfc12933@2026-10-30T18:53:58Z`. Routing note: `lanes degraded: planner and executor codex gpt-6.1-sol unusable (probe rc=75, over daily ration) → opus / claude:claude-haiku-5-5`.
- Controller `claude:claude-opus-5-5` (tok: not reported).
- Designer: not spawned (nothing to design); rotation pointer not advanced.
- Planner: not spawned (nothing to plan).
- Executor: not spawned (no code).
- Evaluator `sonnet`, Agent tool: round 1 70,513 subagent tokens / 10 tool uses / 252 s; round 2 (resumed via SendMessage) 77,122 subagent tokens / 3 tool uses / 37 s. Judge independence: a different model from the controller (opus); same vendor, so vendor-level independence is not claimed.
- Quorum: none (no design doc). Metered spend: $0. No lane recorded dead.

**Ruled out**:
- Fixing the 8 grep sites in-iteration: the guard refuses `.claude/skills/design-doc-creator/**` for the fleet; widening it is a ruling (D-FLEET-5/6 precedent).
- Resolving `sprint-evaluator:worktree-cap-and-exit1` as ghost + duplicate: ask 2 has no other home (judge round 2).
- Treating ask 1 as "add queueing": reset-to-pending exists since January; the open question is why the 10-08 handoffs surfaced as failed.

**Retro lane**: none filed; no skill edit. Observations, not tickets:
- (a) Fourth consecutive fire whose picks were ghosts or out-of-scope decision rows (iterations 70 partly, 71, 72, 73). The ordered open tickets in fleet scope are thinning; most remaining tickets sit in coordinator, CI, pi-extensions or other skills. D-FLEET-36's option A is the first scope ask in a while that would turn a parked ticket into a landable one.
- (b) The judge caught a missed in-scope fact (existing retry code) that the controller's grep did not look for: the controller searched for the cap and the flag, not for what happens when the cap is hit. Draft records that say "X is unbuilt" should grep the error's handler, not just its definition.

Last 3 code landings: iteration 70 (`52e8958f0`), then none at 71, 72, 73. No drift alarm: no routable in-scope UNMET-clause row exists.

**Next**: `skills:agents-copies-stale-non-mission` (attended): scope check first. It names design-doc-creator and model-manager, which are outside the guard, so it may fold into D-FLEET-36's ruling. Then the coordinator tickets (`coordinator:cloud-lane-capped-by-local-worktree-limit`, `coordinator:job-retry-success-dropped-after-failed-completion`), which likely route to D-FLEET-37's owner.

## 74 — 2026-10-31 — `skills:agents-copies-stale-non-mission` parks on D-FLEET-38 and the two codex-credential tickets park on D-FLEET-39 (`79ede6ee9` live, lane frees during a Cloud Run retry); independent sonnet FAIL 48/52 → PASS-WITH-EDITS 84 / 80 [HARNESS]

**Pick**: iteration 73's **Next**, `skills:agents-copies-stale-non-mission` (attended, 10-03). It measured as a decision row, because every fix path is outside the scope guard, so this was a bookkeeping pick and a second item was taken (standing rule 1). Iteration 73 named the coordinator tickets next. Of those, `executor:codex-credential-collision-drops-automatic-planner-handoffs` (attended, 10-12) had a candidate ghost fix on `origin/dev`, so it and its sibling `executor:codex-credential-lease-collision-fails-concurrent-dispatch` (attended, 10-12) were taken together.

**Orphans / prior record**: no open `fleet/*` PR (iteration 73's #2031 merged → `3ca6d7e87`). No `fleet/i74*` branch or worktree before this fire. Self-notices rc 0 (control 852:10).

**Preflight / observe**:
- Armed. gh `sunholo-voight-kampff`. Billing CLEAN. Directives: 0 on #1945 since `2026-10-22T23:29:08Z` (16 comments, none allowlisted; watermark still LEGACY).
- Skill drift: SKILL.md and all 13 resource files `cmp`-identical to `origin/dev` at the resolved symlink target.
- Gate-1 base `3ca6d7e87acaf613f8c8f6659ebfe54944288d67@2026-10-31T01:24:04Z`; local == origin/dev.
- Dev CI at `3ca6d7e87`: `CI` success; `Build and Release` failure (`Build macos-latest` failure, windows and a second macos leg cancelled). There is also the inherited `ailang-core-dev (ailang-multivac-deploy)` Cloud Build red. None is required, all are outside fleet scope, and V1 owns the repo's reds, so they are recorded, not picked.
- Ledger valid, 36 rows, 21 OPEN at start. 33 tickets open. Codex over its daily ration (`MISSION_OVER_RATION=codex`, probe rc 75). The driver handed the planner to `opus` and the executor to `claude:claude-haiku-5-5`; neither was needed.

**Reality check** (first-party, before routing):
- **Skills ticket, real at HEAD.** 27 of 40 paired skills differ (`cmp`); none is `mission-*`/`sprint-*`. `sync-agents-skills.sh --check` rc 0 for its 6 (control). `_scope_verdict fleet` refuses `.agents/skills/{design-doc-creator,release-manager,motoko-analyzer}/SKILL.md` and `.claude/skills/design-doc-creator/SKILL.md`. Controls: allow for `.agents/skills/mission-control/SKILL.md` and `tools/launchd/sync-agents-skills.sh`, core-refuse for `internal/parser/x.go`. The first attempt sourced the functions through process substitution, got `command not found` and printed ALLOW for every path; it was discarded as an instrument failure, and the re-run asserted `type _scope_verdict` first.
- **Skills ticket, shape.** After lossy normalisation (`norm.sh`), 16 are rename-only and 11 carry two-way content edits (`norm.txt`).
- **Credential tickets, fix exists and is live.** `79ede6ee9` (10-15, v0.54.10) adds a provider-keyed lane (`internal/coordinator/credential_lane.go`; call site `daemon_tasks_exec.go:188`). Prod Cloud Logging shows 32 hold lines, all on 10-16.
- **Instrument note.** The coordinator task store cannot see this error: the 6 ticket-named tasks are `failed` with an empty `error` field, so a positive control read 0/6. Cloud Logging can: 10 hits on 10-12. Of 17 `credential already owned` lines in 25 days, 4 fall on 10-16 after the gate went live, and none after 08:53:51Z.

**Did** (no code; designer, planner and executor not spawned; evaluator spawned; generator ≠ judge):
- **Designer, planner, executor**: not spawned. Nothing to build: all three tickets are decision rows. Rotation pointer not advanced. Driver exports: designer `recipe claude:claude-opus-5-5`, planner `agent-tool opus`, executor `recipe claude:claude-haiku-5-5`.
- **Evaluator**: `sonnet`, Agent tool, foreground, read-only on the pin; resolver `agent-tool sonnet declared:alias-pin`. It was given the draft records ([as judged](fleet-mission-evidence/iteration74/draft-records-as-judged.md)) and told to refute them ([report](fleet-mission-evidence/iteration74/evaluation.md)).
  - Round 1: record 1 **PASS-WITH-EDITS 76**. Record 2 (resolve the handoffs ticket as a ghost) **FAIL 48** and record 3 **FAIL 52**. The draft had read the 10-16 failures as a dead owner's stuck lease. The judge extended the log window and found the real sequence. Attempt 0 of task-d61964fa was killed by its first missed heartbeat and published `failed`. Cloud Run's retry re-acquired the lease at 08:44:53Z and went on to open PR #1849. The lane counts only queued and running tasks, so the coordinator dispatched task-6dac947e and task-65c96a92 into the live retry. The controller re-ran the query and confirmed every timestamp ([log](fleet-mission-evidence/iteration74/prod-logs-2026-10-16-codex-go-0843-0912.txt)). Also confirmed: `.agents` design-doc-creator does have a quorum section (the draft said it lacked one); 21 non-mission `.agents` files point at a nonexistent `.Codex/skills`; and the post-10-16 zero is lack of exposure. The handoffs ticket was not resolved.
  - Round 2 (same judge, resumed): **84** and **80**, PASS-WITH-EDITS. Its checks: all 16 rename-only skills are safe to overwrite (no `.agents`-only files); D-FLEET-36 and -38 are one allowlist change; `max_retries=1` comes from a code comment, not inspected config; "two tickets and one adjacent", not three. It added option A2 (the lane also counts a recently failed codex task) and a must-check-first gate on the unread lease-release path. All edits are applied to the ledger rows.
- **Filed** D-FLEET-38 and D-FLEET-39 (OPEN, recommendation A each). Queue rows 53 and 54. No ticket resolved; 33 open.
- **Done-gate**: not applicable (no harness change).

**Progress**: goal unmoved on clause 1 (UNMEASURED). Clause 2 UNMET (no in-scope ticket filed in the last 48h; no pick was in scope). Clause 3 MET (33 open). Clause 4: prior evidence only. Clause 5 preserved.

**Routing evidence**: Gate-1 base `3ca6d7e87acaf613f8c8f6659ebfe54944288d67@2026-10-31T01:24:04Z`. Routing note: `lanes degraded: planner and executor codex gpt-6.1-sol unusable (probe rc=75, over daily ration) → opus / claude:claude-haiku-5-5`.
- Controller `claude:claude-opus-5-5` (tok: not reported).
- Designer: not spawned (nothing to design); rotation pointer not advanced.
- Planner: not spawned (nothing to plan).
- Executor: not spawned (no code).
- Evaluator `sonnet`, Agent tool: round 1 87,963 subagent tokens / 16 tool uses / 92 s; round 2 (resumed via SendMessage) 99,874 subagent tokens / 5 tool uses / 40 s. Judge independence: a different model from the controller (opus); same vendor, so vendor-level independence is not claimed.
- Quorum: none (no design doc). Metered spend: $0. No lane recorded dead.

**Ruled out**:
- Resolving `executor:codex-credential-collision-drops-automatic-planner-handoffs` as a ghost of `79ede6ee9`: its symptom reproduced on 10-16 via the retry path, after the fix shipped (judge round 1).
- "Dead-owner lease" as the 10-16 cause: the owner was a live Cloud Run retry.
- A verbatim sync of all 27 skills: 11 carry `.agents`-only content that `rsync --delete` would destroy.
- Putting `.agents/skills/*` in `_scope_is_harness`: that would refuse those paths for every product loop.

**Retro lane**: none filed; no skill edit. Observations, not tickets:
- (a) This is the fifth consecutive fire whose picks were ghosts or out-of-scope decision rows (iterations 70 partly, then 71–74). D-FLEET-36 together with D-FLEET-38 is the one ruling that would turn parked tickets into landable fleet work.
- (b) The judge again caught what the controller's evidence window cut off. The controller's log query stopped at the last failure line; the judge read past it to see what the "owner" did next. A causal claim about a lease or lock should read the owner's own log through the incident, not only the victims'. This is iteration 73's (b) in a new form: grep what happens after the event, not just the event.

Last 3 code landings: iteration 70 (`52e8958f0`), then none at 71–74. No drift alarm: no routable in-scope UNMET-clause row exists.

**Next**: `coordinator:job-retry-success-dropped-after-failed-completion` (attended, 10-12). It shares D-FLEET-39's cause (a task marked `failed` on attempt 0 while its Cloud Run retry succeeds); scope check first, and it likely folds into D-FLEET-39. Then `executor:3h-timeout-from-queue-and-unsliced-sprints` with its addendum `executor:timeout-loses-work-and-may-never-start`.

## 75 — 2026-10-31 — `coordinator:job-retry-success-dropped-after-failed-completion` folds into D-FLEET-39 (9 dropped retry successes, 7 on pi) and the two executor-timeout tickets park on D-FLEET-40 (SIGBUS, not queue wait); independent sonnet PASS-WITH-EDITS 78 / 72 → 80 / 82 / 76 [HARNESS]

**Pick**: iteration 74's **Next**, `coordinator:job-retry-success-dropped-after-failed-completion` (attended, 10-12). Its fix is coordinator code, so it measured as a decision-row amendment. That made it a bookkeeping pick, and a second item was taken (standing rule 1): `executor:3h-timeout-from-queue-and-unsliced-sprints` with its addendum `executor:timeout-loses-work-and-may-never-start` (both attended, 10-12), the named follow-on.

**Orphans / prior record**: no open `fleet/*` PR (iteration 74's #2032 merged → `ddf4ec6a1`). No `fleet/i75*` branch or worktree before this fire. Self-notices rc 0 (control 852:10).

**Preflight / observe**:
- Armed. gh `sunholo-voight-kampff`. Billing CLEAN.
- Directives: 0 on #1945 since `2026-10-22T23:29:08Z` (17 comments, none allowlisted; the watermark is still LEGACY).
- Skill drift: SKILL.md and all 13 resource files are `cmp`-identical to `origin/dev` at the resolved symlink target. The pin is at `origin/dev`.
- Gate-1 base `ddf4ec6a19c5e53a4e1585db3bc32a4e8a6feda8@2026-10-31T08:14:07Z`.
- Dev CI at `ddf4ec6a1`: `CI` and `Build and Release` success. 18 checks, one not green: the inherited `ailang-core-dev (ailang-multivac-deploy)` Cloud Build red. It is not required, it is outside fleet scope, and V1 owns the repo's reds, so it is recorded, not picked.
- Ledger valid, 38 rows, 23 OPEN at start. 33 tickets open.
- Codex is over its daily ration (`MISSION_OVER_RATION=codex`). The driver handed the planner to `opus` and the executor to `claude:claude-haiku-5-5`; neither was needed.

**Reality check** (first-party, before routing):
- **Job-retry ticket, real at HEAD.** `pubsub_completion_handler.go:117-121` skips any completion on a terminal task, and `completion_binding.go:64-67` names "a retried job reporting after the first attempt" as an intended duplicate. `git log origin/dev --since=2026-10-12` has no fix. `pubsub.TaskCompletion` has no attempt field.
- **Timeout tickets, prod Cloud Logging** (`ailang-multivac`):
  - All 6 named tasks went from `Cloud dispatch` to `execute-job: starting task` in 14–55 s. The stale detector's own `age` places `queued_at` on the dispatch line.
  - Each ran two attempts (Cloud Run retry 12–31 min later). All 12 attempts ended `Container terminated on signal 7.`, and each execution logged `has failed to complete` 24–58 min after dispatch.
  - The coordinator marked each task failed only at 3h0m–3h1m44s, i.e. 2h02m–2h37m of dead time.
  - Since 10-01, SIGBUS appears 18 times, all codex-go, all on 10-03..10-09; there are 0 in the 20 codex-go sprint-executor attempt-starts since 10-10.
  - The stale detector fired 8 times since 10-01.
- **Timeout tickets, ghosts.**
  - (d) focused gates: #1770 `dbe4b26a3` (10-12, 46 min after the ticket; `sprint-planner/SKILL.md:336`, mirror identical).
  - (b) planner split rule: same line.
  - (c) executor half: #1688 `face88a3f` and #1905 `3f9a0c092`.
  - Ask (a) is true in code (the cloud clock is the claim time; `stale_task_detector.go:232-247`, `store.go:64-69`) but not causal.
- **Also measured, not worked: `executor-image:no-make-no-cc-cgo-tests`.** The skill half is a ghost of #1905 (`unattended_runs.md:58-64`). `docker/Dockerfile.agent-codex-go` installs make and gcc, and `agent-go` / `agent-pi-go` do not. `docker/**` is outside scope.

**Did** (no code; designer, planner and executor not spawned; evaluator spawned; generator ≠ judge):
- **Designer, planner, executor**: not spawned, because nothing was in fleet scope to build. This is not a spawn failure. Resolver output: designer `recipe claude:claude-opus-5-5 declared:provider-pin`, planner `agent-tool opus fail-closed:env-pin`, executor `recipe claude:claude-haiku-5-5 declared:provider-pin`. Rotation pointer not advanced.
- **Evaluator**: `sonnet`, Agent tool, foreground, read-only on the pin; resolver `agent-tool sonnet declared:alias-pin`. It was given the draft ([as judged](fleet-mission-evidence/iteration75/draft-records-as-judged.md)) and told to refute it ([report](fleet-mission-evidence/iteration75/evaluation.md)).
  - Round 1: record 1 **PASS-WITH-EDITS 78**, record 2 **PASS-WITH-EDITS 72**. Its four BLOCKING-IF-LEFT findings:
    - The drop is recurring and mostly on pi: 9 tasks since 10-01, not the 2 the draft cited.
    - The minutes were wrong (24–58, not 27–63).
    - The SIGBUS exposure must be codex-go sprint-executor attempt-starts (20), not all jobs (114).
    - There are two never-started tasks, not one. Both are coordinator shutdown strands: `Received signal: terminated` came microseconds before `Failed to publish task … context canceled`, and the re-queue at `daemon_tasks_exec.go:206` runs `ResetTaskToPending(d.ctx, …)` on the cancelled context.
  - It also read `maxRetries: 1` from the job config, and said a row recommending "fold into D-FLEET-23" is circular.
  - The controller re-ran the drop recount (9 distinct tasks, which corrects the judge's own "in addition to" wording), the job config on both jobs, both shutdown lines and line 206.
  - Round 2 (same judge, resumed): **80 / 82 / 76**, PASS-WITH-EDITS. Its edits, all applied:
    - "only the image changed" became "image digest changed, other fields not diffed".
    - The D-FLEET-40 title says one ask is built and two half-built.
    - Every residual ask has a recorded home, including the answer to the cloud-lane ticket's ask (c), whose text the controller confirmed.
    - The stale D-FLEET-39 phrases were fixed.
- **Recorded**: D-FLEET-39 and D-FLEET-23 got re-measure appends; D-FLEET-40 is filed (OPEN, recommendation A). Queue rows 55 and 56. No ticket resolved; 33 open.
- **Done-gate**: not applicable (no harness change).

**Progress**: goal unmoved on clause 1 (UNMEASURED). Clause 2 UNMET (no in-scope ticket filed in the last 48h; no pick was in scope). Clause 3 MET (33 open). Clause 4: prior evidence only. Clause 5 preserved.

**Routing evidence**: Gate-1 base `ddf4ec6a19c5e53a4e1585db3bc32a4e8a6feda8@2026-10-31T08:14:07Z`. Routing note (driver): `lanes degraded: planner and executor codex gpt-6.1-sol unusable (probe rc=75, over daily ration) → opus / claude:claude-haiku-5-5`.
- Controller `claude:claude-opus-5-5` (tok: not reported).
- Designer: not spawned (nothing to design); rotation pointer not advanced.
- Planner: not spawned (nothing to plan).
- Executor: not spawned (no code).
- Evaluator `sonnet`, Agent tool: round 1 122,475 subagent tokens / 38 tool uses / 374 s; round 2 (resumed via SendMessage) 134,780 subagent tokens / 2 tool uses / 44 s. Judge independence: a different model from the controller (opus); same vendor, so vendor-level independence is not claimed.
- Quorum: none (no design doc). Metered spend: $0. No lane recorded dead.

**Ruled out**:
- Queue wait as the cause of the 6 timeouts: 0 of 6 waited more than 55 s.
- "Jobs that never started" (addendum) for the 6 named tasks: all 6 started.
- Attributing the post-10-09 SIGBUS drop to #1770 or #1905: the last SIGBUS predates both.
- `max_retries=0` as the job-retry fix: it would have lost all 9 retry successes.
- Filing a new row for the job-retry ticket: it is D-FLEET-39's event seen from the other side.

**Retro lane**: none filed; no skill edit. Observations, not tickets:
- (a) This is the sixth consecutive fire whose picks were ghosts or out-of-scope decision rows (71–75). All five rows filed since iteration 73 (D-FLEET-36 to -40) are outside fleet scope. D-FLEET-36 together with D-FLEET-38 is still the one ruling that would turn parked tickets into landable fleet work.
- (b) The judge again found what the controller's query window cut off. This time the controller counted the ticket's own named instances (2 codex) and never asked whether the class recurred elsewhere (7 pi). It is iteration 74's (b) in a third form. Iteration 73: read past the victim's last line. Iteration 74: read the owner's log. Here: count the class, not the cited instances. Not yet a skill edit: Gate 5 requires ≥2 frictions on one gap, and these are three adjacent gaps that one rule might cover. Flagged for the next retro that sees a fourth.

Last 3 code landings: iteration 70 (`52e8958f0`), then none at 71–75. No drift alarm: no routable in-scope UNMET-clause row exists.

**Next**: `executor-image:no-make-no-cc-cgo-tests` (attended, 10-08). Its skill half is a ghost of #1905; check which image task-69be551b ran before parking the `docker/**` half. Then `eval:motoko-connection-probe-orphaned` (motoko, 10-01): retire or own a probe in `tools/eval/`, which is outside scope (D-FLEET-18).
