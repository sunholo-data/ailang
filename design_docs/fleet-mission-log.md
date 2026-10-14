# Fleet Mission Log

Append-only, one entry per iteration, newest at the bottom. Charter: [fleet-mission.md](fleet-mission.md).

> **Older entries are ARCHIVED.** This file holds the newest 20. The full record of every
> iteration is in `fleet-mission-log-archive.md`, and a one-line index of ALL of them —
> the thing to grep before picking work, so the loop never repeats itself — is in
> `fleet-mission-index.md`.

## 17 — 2026-10-03 — heartbeat fix pushed as #1578 and re-judged PASS 95; merge blocked by a weekend-only red in TestOllamaQuota* → PARKED-ON-CLOCK [HARNESS]

**Pick**: P0 #1 `skill:heartbeat-relative-path-absent-in-world` (8 slots lost, World), resuming iteration 16. Gate 0: kill switch armed. gh `sunholo-voight-kampff`. Billing CLEAN. 0 directives on #1380 since `2026-10-02T08:51:41Z`. Gate 1: pin = origin/dev `2a1f3f295`. SKILL.md and all 12 resources match origin, and the resolved symlink equals the pin copy. Dev CI `success`; the only non-green check of 17 is SonarCloud, inherited. Resume predicate run as a command: the pinned hook's `_scope_is_harness` has the `.agents/skills/mission-*` arm, so it is met. No new design, plan or code was needed, so the designer, planner and executor were not spawned. This is a resume of a judged build.
**Did**:
- Rebased `fleet/i16-heartbeat-rev5` onto `2a1f3f295` with no conflicts. Per-file blob check against the judged head `14b87879b`: 17 of 18 SAME. The DIFF is the design doc's one-line status header. The `pre-push` arm and the charter Authority line had already landed via #1575.
- Pushed. The pinned guard accepted the push. Opened PR #1578.
- Independent evaluator: **PASS 95/100**. Applied its two non-blocking doc nits in `ab0c539a1`: the stale M1 SHA `41e0005e9` became `bb573f466`, and the changelog section about the guard, which now ships in #1575's fragment, was removed.
- Gate 3b: the required `test` check and Build ubuntu/macos/windows are RED, every leg on the same two tests. `TestOllamaQuotaVerifiedLimits` (`ollama_quota_test.go:42`) and `TestOllamaQuotaHTTPAndCredentialBinding` (`:82`) build `limits` from `time.Now()`, with `WeeklyResetsAt = now+6d`. So the window starts yesterday, and `WeekdayPacePercent` counts only Mon–Fri hours (`quota_ledger.go:69`). On a weekend the allowance collapses to about 2.5% in the Fri→Sat window and 0% for Sat→Sun. #1524's 3pp ollama start margin (`quota_margin.go:33`) then marks even 0 usage `over`. It reproduces locally at `origin/dev` with no PR code, also under `env -i` with a temp HOME. Dev's own CI at 14:43Z was green because that window still held Friday hours. By arithmetic it goes green again around Mon 2026-10-05 08:00Z: test 1 needs about 6.6% allowance. `gh pr view 1578` reports `mergeStateStatus=BLOCKED`, and the required contexts are `test, lint, build, docs-gate`.
**Progress**: goal unmoved (0 tickets resolved). Clause 1 UNMEASURED. Clause 2 UNMET (the ticket has been open since 2026-09-26). Clause 3 MET (41 open signatures). Clause 4 prior evidence only. Clause 5 preserved (nothing merged).
**Routing evidence**: base-gate1=2a1f3f2952574356da196147b359acd2c11e4f65@2026-10-03T20:44:33Z; base=2a1f3f2952574356da196147b359acd2c11e4f65@2026-10-03T20:59:01Z. `MISSION_ROUTING_NOTE`: codex over daily ration → planner opus, executor sonnet-5-5 (not exercised).
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer / planner / executor**: NOT SPAWNED. The item was a resume whose build was judged in iteration 16. There was nothing to design, plan or author beyond a rebase, a push and two doc-nit edits by the controller.
- **Evaluator**: resolver `reroute pi:openrouter/minimax/minimax-m3 generator-equals-judge`. OpenRouter is in `MISSION_OVER_RATION` (backlog (a) from iteration 16, still open). `claude:claude-sonnet-4-6` was skipped as the same family as the iteration-16 executor `claude:claude-sonnet-5-5`. Ran `opus` through the Agent tool, foreground, 76,510 tok, 25 tool calls, 9 min. Judge ≠ generator (opus vs sonnet-5-5). FLAG: same model as the controller, in a fresh context.
- **Metered**: $0.00. Chain `eefe0a7e` posted.
- **Ration**: codex, ollama and openrouter over; Anthropic subscription OK.
**Verification** (judge, isolated worktree `fleet-i17-evaluator`, since removed):
- 18 guarded sites (9 per copy), 0 bare matches (rc=1), `sync-agents-skills.sh --check` rc0.
- Drill from `/tmp` with a temporary `AILANG_STATE_DIR`, bash 3.2.57 and zsh: rc0 with an absolute root; rc1 with "must be absolute" for unset, relative and empty roots; 0 rc127. Control: the old bare form gives rc127.
- `make test-launchd-drivers` MAKE_RC=0 (6m25s, 167 ok). One mutant (bare form restored): grep 1 match, guarded count 17, sync check rc1.
- Controller: `go test ./internal/mission/...`. Only the two `TestOllamaQuota*` tests fail. Every other mission package is ok.
**Done-gate**: not claimed. The dry-runs are owed after the merge, and both armed siblings (world, stapledon) were mid-iteration this fire, so the overlap guard would yield before the dry-run exit.
**Ruled out**:
- Fixing the quota tests from the loop. D-FLEET-12 reserves the quota-margin work (`quota:admission-has-no-headroom-margin-and-flaps`, still an open ticket) for attended sessions. Also no ticket, no self-sourcing.
- Admin-merging past a required check (standing rule 2).
- Treating the red as caused by the PR. The PR has no Go code, and the red reproduces on `origin/dev`.
- Re-asking D-FLEET-13.
- Reading or acking `mission-fleet`.
- Touching the 3 pending coordinator approvals.
**Retro lane**: none new for the skill. The "scope-guard widening ships alone" guardrail worked as written: the push went through first try. New observation for Mark, not ticketable by the fleet: weekday pacing plus start margins make any `time.Now()`-anchored quota test calendar-dependent. Every PR to dev is blocked until about Monday 08:00Z, for every mission and every attended session. Suggested fix: pin `now` to a fixed weekday in both tests, e.g. `time.Date(2026, 9, 30, 12, 0, 0, 0, time.UTC)`, since `pace_test.go` already pins `paceLocation = time.UTC`.
**Next**: once `gh pr checks 1578` is all-green on the required checks, merge → Gate 3b SHA-pinned on the merge → done-gate dry-runs under an idle armed sibling → `ailang mission ticket resolve skill:heartbeat-relative-path-absent-in-world --sha <merge>`. Then the D-FLEET-9 rotate-log pair, then pi-runner pre-dirty. The record PR for this entry is blocked by the same red.

## 18 — 2026-10-04 — rotate-log pair built + judged PASS 97 as PR #1580; parked on the same weekend TestOllamaQuota* red; #1578 re-probed still red [HARNESS]

**Picked**: `mission:rotate-log-registry-cwd` + `rotate-log:status-flag-mutates-and-world-resolves-to-status-archive` (charter P0 #2, `[UNPARKED · D-FLEET-9 = A]`; iteration 17 banked it as next). P0 #1 (heartbeat) re-probed at Gate 2: `gh pr checks 1578` still red on `test`/`Build*` (03:05Z) — the weekend `TestOllamaQuota*` window, a clock park, so the queue moved to the next unblocked item. #1380: 0 new allowlisted directives since 2026-10-02T08:51:41Z (22 comments; watermark advanced to 2026-10-03T21:01:06Z after triage). D-FLEET-13 and D-FLEET-14 (filed iteration 17, on the i17-record branch) remain OPEN awaiting Mark. Kill switch armed, gh account `sunholo-voight-kampff`, billing tripwire CLEAN. Gate 1: pin = origin/dev = `2a1f3f295` (drift 11 on the main checkout's `dev`, pin authoritative); running skill (pin + resolved symlink, 13 files each) byte-identical to origin/dev; dev CI green at base with the standing SonarCloud red (V1-owned, recorded). Weekly external-issue sweep NOT due this fire (not the first fire after a Monday-07:00 rotation; due on the first fire after 2026-10-05 07:00Z).
**Reality check**: all ticket premises re-verified at HEAD `2a1f3f295` — `loadMissionRegistry` CWD/ancestor discovery confirmed (loader + 6 production calls + iteration default); `--status` mutates the status archive (flag loop → RotateLog); the driver exports `AILANG_MISSION_REGISTRY` ONLY in the binary branch (`mission-control.sh:1976`); prior design doc `m-mission-rotate-log-safe.md` (iteration 8) exists with rounds 1–2 blocked and the D-FLEET-9 = A ruling pending execution. The pin's untracked sprint JSON/plan were iteration-16 leftovers already inside PR #1578 (ignored, not adopted). CLAIM sent to controlplane (sibling telemetry dirty in the main checkout).
**Shipped**:
- **Design `m-mission-rotate-log-safe.md` Rev 3–5** (worktree `fleet-iter18-rotate-log`, commit `c2bf5bef0`): Rev 3 = complete shared-loader caller audit (C1–C11) + `--status` rejection specified before load + premises re-verified at HEAD; Rev 4 = C7 widened (normalize maps shared targets through the loaded registry's origin, loud no-registry failure preserved) + `m.root` formula pinned; Rev 5 = narrow-refinement carve-out, controller-applied reviewer-verbatim fixes (`repoRootFor` deletion specified; whole-repo `--status` caller enumeration with positive controls; V26/V27).
- **Sprint plan + JSON** (`3391da544`, planner kimi-k3): 4 milestones, 345 LOC, every acceptance command baselined at `c2bf5bef0` (B1–B12, incl. B10 RED→GREEN feature assertion and B11 UNINFORMATIVE-UNDER-SANDBOX labeling).
- **Implementation** (`db2a5b0b3` M1+M2, `ef6b9fe93` M3, `60f2b7716` M4, executor deepseek-v4.1-flash): additive `Mission.Root()`; strict `--stream log|status`; ruled `--status` rejection BEFORE registry load; rotate-log AND normalize shared targets via `m.Root()` with loud empty-root failure; `repoRootFor` deleted (0 references incl. tests); loader errors name tried locations; `_mc_export_mission_registry` (bash 3.2) in both driver branches with missing-root validation; 4-arm fixture wired into `make test-launchd-drivers`; 9 pinned synthetic tests (159 LOC, no t.Parallel, t.TempDir fixtures only); changelog fragment.
- **PR #1580** pushed through the pinned guard (which now carries #1575's `.agents` arm) — no refusal.
**Progress**: goal unmoved (0 tickets resolved; the pair resolves when #1580 merges). Clause 2 UNMET: the rotate-log ticket has been open since 2026-09-26. Clause 3 MET (41 open signatures). Clause 1 UNMEASURED. Clause 4 prior evidence only. Clause 5 preserved: nothing unjudged merged; every check the PR can reach is green and the sole red is the inherited weekend signature.
**Routing evidence**: base-gate1=2a1f3f2952574356da196147b359acd2c11e4f65@2026-10-04T03:02:54Z; worktree provenance base=2a1f3f295 (MATCH at snap). Resolver: planner `codex:gpt-6.1-sol anthropic-fallback:fail-closed:planner-lane-field-invalid` (doc's Planner-Lane field) — codex bucket over ration → walked the declared chain (ollama/kimi-k3:cloud over ration) → `pi:openrouter/moonshotai/kimi-k3`. The operator's standing request to spawn designer/planner/executor/evaluator via the Agent tool: this harness (pi controller) has no Agent/Task tool — every role ran as a model-pinned sub-agent via the skill's own cross-provider `scripts/mission_pi_run.sh` recipe (typed verdicts, sandboxed), which is what the routing table specifies for `recipe` lanes this fire; NO role ran inline on the controller's model, and the judge is a different model family from designer, planner, executor, and controller.
- **Controller**: `pi:openrouter/z-ai/glm-5.3` (session; tok: not reported by the harness).
- **Designer**: `pi:openrouter/z-ai/glm-5.3` (rotation resolved by the driver after anthropic/codex/ollama rungs refused, rc75) — probe rc0 (`--no-extensions`); r3 run rc0 234s, 40 turns, 1,762,806 tok gross (76,362 in / 26,415 out / 1,629,440 cache-read), $0.6468; r4 revision run rc0 213s, 32 turns, $0.6124.
- **Quorum** (author benched, Z-AI sits out): r3 BLOCKED $0.1235 — gemini reject (C7), kimi reject (path formula), sonnet ABSENT (quota), gpt6-1-sol ABSENT (auth); r5 on Rev 4 BLOCKED $0.1172 — gemini reject (orphaned `repoRootFor`), kimi reject (directory-scoped `--status` audit), same two absent. Both rounds' objections controller-verified first-party; round 5 applied under the carve-out.
- **Planner**: `pi:openrouter/moonshotai/kimi-k3` — probe rc0; run rc0 437s, 44 turns, 1,885,419 tok gross, $1.1465.
- **Executor**: `pi:openrouter/deepseek/deepseek-v4.1-flash` — probe rc0; run rc0 325s, 85 turns, 6,488,804 tok gross, $0.0505.
- **Evaluator**: `pi:openrouter/minimax/minimax-m3` — probe rc0; run rc0 376s, 53 turns, 1,932,479 tok gross, $0.1441. Judge ≠ generator (minimax vs glm/kimi/deepseek). Session handshake completed (protocol ack verified in its own session).
- **Metered**: $2.8410 total ($1.2592 designer + $0.2407 quorum + $1.1465 planner + $0.0505 executor + $0.1441 evaluator) against the $5 ceiling.
- **Ration**: codex, ollama, anthropic over daily ration (Sunday); openrouter charged metered.
**Verification**:
- **Controller** (out-of-sandbox): 9/9 new tests `-v` first-party; `go test ./cmd/ailang/... -run 'Mission' -count=1` ok; `make test-launchd-drivers` rc0 (59 arms + `mission registry env` suite); both dry-run arms `DRY RUN ok` (world profile, `AILANG_DRIVER_PINNED=2bf95391f`); diff scanned for out-of-worktree writes — none (fixtures t.TempDir only); main checkout status unchanged by the executor (sibling telemetry only).
- **Judge** (isolated worktree `fleet-iter18-evaluator` at `60f2b7716`, own branch): **PASS 97/100**, 0 blocking. Re-ran the full gate table (including a broader `repoRootFor` sweep: zero hits in cmd/internal/tools/scripts); re-ran the mutation matrix — 3/3 mutations RED on the named tests (wrong-repo rotation caught by the absolute-path pin; 0 headings rewritten; both `--status` tests red) and GREEN on restore (md5-verified). Adjudicated the executor's AC5 finding (grep 2 vs plan 1: the rejection arm's own error text contains `--status`; PASS). Non-blocking: a robustness note on `--stream` value parsing shape.
- **Gate 3b (PR #1580, checks read direct)**: `launchd drivers (bash 3.2)` SUCCESS (the new suite ran on CI), `lint`/`govulncheck`/`CodeQL`/`changes` SUCCESS; required `test` FAILURE at 04:09Z on exactly `TestOllamaQuotaVerifiedLimits` + `TestOllamaQuotaHTTPAndCredentialBinding` (job log read with `--allow-escape-sequences`, ANSI stripped, `--- FAIL` lines quoted; control: `=== RUN`/`PASS` lines present in the same log) — the D-FLEET-14 weekend signature, inherited from the base, outside this diff → **PARKED-ON-CLOCK**, same predicate class as #1578/#1579. No wait burned: the signature was confirmed from the run itself within one poll window.
- **Done-gate**: surface edited ✓; reach = PR #1580 pushed ✓; dry-runs healthy+degraded ✓ (world profile; `lanes=DEGRADED(14)` is the genuine Sunday all-buckets-over-ration state, not a code defect — each lane probed and handed off in chain order correctly); `make test-launchd-drivers` ✓; nothing reloaded mid-iteration ✓. Ticket resolution remains gated on the merge (Monday), per the queue row predicate.
**Ruled out**:
- fixing the `TestOllamaQuota*` tests from the fleet (D-FLEET-12: quota-margin work is attended; D-FLEET-14 pending Mark — default: wait for Monday)
- merging on an inherited red or arming auto-merge on it (auto-merge never clears a base-inherited red pinned to the PR head)
- basing the record on `dev` and conflicting with the unmerged i17 record (record 18 stacks on `fleet/i17-record` instead)
- re-quoruming after round 4 (carve-out conditions met: both objections reviewer-verbatim, concrete, direction-preserving; recorded V26/V27)
- spawning any role inline on the controller model, or letting the judge share a model with any generator
**Retro lane**: none new for the skill this fire. Two process observations: (a) the plan-predicted grep count (AC5) was wrong while the executor followed the plan exactly — the plan's baseline arithmetic was controller-planner-checked but not re-derived; a plan AC that counts its own output text needs the count re-derived after the text is fixed. (b) The first `mission-worktree.sh add` call with an abbreviated 9-char SHA created a worktree at a SHA-named path inside the pin — the helper accepts the short form silently; worth a one-line guard at an attended moment. Neither is ticketable by the fleet; both are recorded here for Mark.
**Next**: when `gh pr checks 1580` (and #1578/#1579) are green after Mon 2026-10-05 ~08:00Z: merge #1578 → done-gate dry-runs → resolve the heartbeat signature; merge #1580 → Gate 3b SHA-pinned on the merge → `ailang mission ticket resolve mission:rotate-log-registry-cwd --sha <merge>` + `... rotate-log:status-flag-mutates...`; merge the record PRs in order. Then pi-runner pre-dirty (P1 #5), then the Phase 3a skill-resolution directive. D-FLEET-13/14 await Mark.

## 19 — 2026-10-05 — both P0 heads LANDED: heartbeat #1578 and rotate-log pair #1580 (after a Windows walk-up hang fix); 3 tickets resolved [HARNESS]

**Pick**: resume both P0 rows, which were PARKED-ON-CLOCK on the weekend-only `TestOllamaQuota*` red. Gate 0: kill switch armed (`mission-fleet.disabled` absent). gh `sunholo-voight-kampff`. Billing CLEAN. 0 directives on #1380 since `2026-10-03T21:01:06Z` (25 comments). Gate 1: pin = origin/dev `9eac33b7b`. The running skill (resolved symlink, 13 files) is byte-identical to origin/dev. Dev CI `success`; the only non-green of 21 checks is SonarCloud, inherited (also `failure` on `2a1f3f295`, `adab9b7d9`, `74d5a3b55`). Records 17 and 18 had never reached dev (#1579 and #1581 were open), so this fire read them from their branches. Clock predicate run as a command: `go test ./internal/mission/ -run TestOllamaQuota -count=1` → `ok` on Monday 10:36Z.
**Did**:
- **#1578.** Re-ran the failed CI. Required `test`/`lint`/`build`/`docs-gate` all SUCCESS. The one non-required red, `Build macos-latest` (`TestMemprobeDebugLogDoesNotAccumulate`, a memory test; the PR has no Go code), passed on rerun attempt 3, so it was a flake. Merged with `--squash --match-head-commit ab0c539a1` → **`c55ca4398`**. Gate 3b: dev CI success; full check set 17, only the inherited Sonar red.
- **#1580: a real Windows defect, hidden all weekend behind the quota red.** `TestMissionNormalizeNoRegistryStillFailsLoudly` hung to the 10-min package timeout on both `test-windows` and `Build windows-latest`. Job logs were read with `--allow-escape-sequences`, sized at 4.1 MB and 131 KB, and the `running tests:` block names the test. Dev's last 5 commits are green on Windows. Cause: `loadMissionRegistry`'s walk `for d := wd; d != "/" && d != "."; d = filepath.Dir(d)` never terminates on Windows (`filepath.Dir(C:\) == C:\`). It pre-dates the PR (`origin/dev` `mission_cmd.go:178`); #1580's new test was the first to run it from a directory with no registry. The second dev site (`:392`, `repoRootFor`) had already been deleted by #1580.
  - Fix `cffc0447a` (executor). The helper `registryWalkCandidates` stops when `filepath.Dir(d) == d`, with the same Unix candidate list (root excluded before and after). It adds `TestRegistryWalkCandidatesTerminatesAtVolumeRoot` and the changelog fragment `2026-10-05-mission-registry-walk-windows.md`.
  - Pushed as a fast-forward onto the PR branch, through the pinned guard. Both Windows legs and all required checks then went green.
  - Merged with `--match-head-commit cffc0447a` → **`e7628b05e`**. Gate 3b: dev CI success; 21 checks, only the inherited Sonar red.
- **Resolved 3 tickets** (open 41 → 38): `skill:heartbeat-relative-path-absent-in-world` (`c55ca4398`, replied to world), `mission:rotate-log-registry-cwd` (`e7628b05e`, replied to stapledon and v1), `rotate-log:status-flag-mutates-and-world-resolves-to-status-archive` (`e7628b05e`, replied to world).
- **Records.** #1579 merged (`845bbbbdf`). #1581 (iteration 18) conflicted after that squash, so its commit was cherry-picked cleanly onto dev in this record branch, which supersedes it.
- **Weekly external-issue sweep** (first fire after the Monday rotation boundary): per-issue table in the charter queue. 0 new queue items. `#1306` got a half-fixed verdict comment (comment count 0 → 1, still OPEN).
**Progress**: 3 tickets resolved, the first since iteration 15. Clause 1 UNMEASURED. Clause 2 UNMET: the three resolutions took about 8–9 days each from filing, against a ≤48h median target. Clause 3 MET (38 open signatures). Clause 4 prior evidence only. Clause 5 preserved: every merge was judged and the required CI on each merge SHA was green.
**Routing evidence**: base-gate1=9eac33b7b62058c9718930237cbbd696ceae7040@2026-10-05T10:36:08Z. `MISSION_ROUTING_NOTE`: controller pi:openrouter/z-ai/glm-5.3 → claude:claude-opus-5-5 (probe ok); codex over daily ration → planner opus, executor claude:claude-sonnet-5-5.
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer / planner**: NOT SPAWNED. Both P0 items were judged builds, and the Windows fix is a one-function termination guard whose defect, cause and fix were measured from the CI log. A design doc or plan would have added a gate and nothing to judge.
- **Executor**: resolver `recipe claude:claude-sonnet-5-5`. Ran via `claude-sub` in worktree `~/.ailang-worktrees/fleet-i19-win-walk`, billing tripwire CLEAN. Probe rc0, run rc0 (tok: not reported, text mode). Wrote `cffc0447a`.
- **Evaluator** (×2): resolver `reroute pi:openrouter/minimax/minimax-m3 generator-equals-judge`. OpenRouter is in `MISSION_OVER_RATION`, so minimax was skipped, and `claude:claude-sonnet-4-6` was skipped as the executor's family. Ran `opus` via the Agent tool, foreground, `MISSION-ROLE: evaluator`.
  - Judge 1 (78,553 tok, 24 tools, 24 min): #1578 **PASS 96**, #1580 **PASS 96**.
  - Judge 2 (52,981 tok, 13 tools, 10 min): `cffc0447a` **PASS 92**.
  - Judge ≠ generator (opus vs sonnet-5-5 / deepseek / glm / kimi). FLAG: same model as the controller, in a fresh context.
  - The first spawn was DENIED `fail-closed:role-missing` because the role token shared the first line with prose. Re-spawned with the token alone on line 1.
- **Metered**: $0.00. **Ration**: codex, ollama and openrouter over; Anthropic subscription OK.
**Verification**:
- **Judge 1, #1578**:
  - 9+9 guarded sites, 0 bare; sync check rc0; `test_agents_skills_sync.sh` rc0.
  - 36-run drill from `/tmp` under `env -i` with bash 3.2.57 and a temp state dir: absolute → rc0 ×9; unset, empty or relative → rc1 with "must be absolute"; control old form → rc127.
- **Judge 1, #1580**:
  - Delta `60f2b7716..2bf95391f` is docs only.
  - `go test ./cmd/ailang/... -run Mission` ok; `go test ./internal/mission/...` ok; `make test-launchd-drivers` rc0 (442 s).
  - Own mutants M1 (CWD target) and M2 (`--status` through) both red; restore green.
- **Judge 2, `cffc0447a`**:
  - Old-vs-new equivalence program on 9 Unix path shapes: all equal.
  - Windows `C:\`, UNC and drive-relative cases reasoned from Go 1.26 `filepathlite.Dir`.
  - `GOOS=windows go vet` rc0; `changelog_fold.sh --check` rc0.
  - Mutant B (guard dropped) red: "did not terminate". Mutant A (old guard) green on Unix: only Windows CI distinguishes it (non-blocking).
- **Controller**:
  - Guarded stamp from merged dev run from `/tmp` in this live driver-spawned shell → rc0 (this wrote the gate-4 heartbeat early, at 11:31Z).
  - `make test-launchd-drivers` MAKE_RC=0 at `e7628b05e` (504 s).
  - Driver diff `2bf95391f..e7628b05e` = 0 lines.
**Done-gate**:
- Surface ✓.
- Reach ✓ (both merges on origin/dev).
- `make test-launchd-drivers` ✓.
- Nothing reloaded ✓.
- Dry-run healthy/degraded **not re-run this fire**: no armed sibling was idle (world pid 18901 and stapledon pid 19130 ran all fire; v1, motoko and docs are disabled and exit at the kill switch). Carried instead:
  - #1578 changes 0 driver bytes.
  - #1580's driver bytes equal `2bf95391f`, which iteration 18 dry-ran healthy and degraded (world profile).
**Ruled out**:
- Admin-merging past the Windows red (standing rule 2), or treating it as a flake (it reproduced in two jobs, named one test, and has a deterministic cause).
- Fixing the walk in a separate PR after merging #1580 (that would merge a known hang into every Windows CI run).
- Spawning a designer or planner for a one-function guard.
- Touching `TestOllamaQuota*` (D-FLEET-12 / D-FLEET-14).
- Re-asking D-FLEET-13/14.
- Reading or acking `mission-fleet`.
- Touching the 3 pending coordinator approvals.
- Merging #1581 over its conflict.
**Retro lane**: none for the skill (one instance each). Two observations for Mark:
  - (a) A clock park can hide a real red. While every PR is red on the weekend signature, other reds on the same PR are never read. #1580's Windows hang sat behind the quota red for two fires. Resume predicates should re-read the whole failing set once the clock clears, as this fire did.
  - (b) D-FLEET-14 recurs every weekend. Next Saturday it blocks every PR to dev again, fleet and attended alike.
**Next**: P1 #5 `pi-runner:verdict-blind-to-commits-and-predirty` (pre-dirty half), then the Phase 3a skill-resolution directive (after P1). The D-FLEET-12 pair is planner-ready. D-FLEET-13/14 await Mark.

## 20 — 2026-10-05 — pi-runner pre-dirty fix built + judged PASS 86→85 as PR #1593; landing blocked by a GitHub Actions major outage → PARKED-ON-CLOCK [HARNESS]

**Pick**: P1 #5 `pi-runner:verdict-blind-to-commits-and-predirty`, the pre-dirty half (charter `[NEXT]`; Mark's 2026-09-29 triage order).
- Gate 0: kill switch armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing CLEAN; 0 directives on #1584 since `2026-10-05T12:33:09Z` (1 comment, public). 38 open tickets.
- Gate 1:
  - Pin = origin/dev `a12a319b5`. Running skill: `SKILL.md` + 5 resources match origin. 7 gate resources DIFFER: the resolved main checkout is 22 commits behind origin. The delta is the absolute `MISSION_DRIVER_ROOT` heartbeat-stamp form + the pi rc-19 line (#1578). This fire followed the resolved copy but used the absolute stamp form.
  - Dev CI: last push run `d273ea1cd` failure = `test-windows`, `cmd/ailang` at 401 s / 96% of its timeout budget, with hang-guard FAILs (not logic). The 2 prior runs were green; V1 owns this lane, recorded only.
  - HEAD `a12a319b5` (6 dependabot merges) had **no** push CI. Dispatched `CI` → `test` success, `lint` success; `changes` cancelled (unassigned 15 min), so Windows is unverified at HEAD.
- Gate 2: the pick still reproduces at HEAD. `mission_pi_run.sh:346` gates `ok` on porcelain non-empty, and `:347` says a pre-dirty tree reads `ok` vacuously. No orphan iteration-20 PR or worktree. #1583 (`fix/ollama-quota-test-clock`, attended) addresses D-FLEET-14's subject; not fleet's PR, untouched.

**Did**:
- **Plan** (opus Agent) `6f60e0d53`, `design_docs/planned/sprint-plan-pi-runner-predirty.md`. Two load-bearing findings, both verified by the controller:
  - TEST 1/5 started dirty with no-op stubs, so they passed only because of the bug.
  - `scripts/test_mission_pi_run.sh` ran in no gate (`make/test.mk` listed only the three `tools/launchd/test_mission_pi_run_*` suites).
- **Executor r1** `d3974c34f`:
  - A worktree fingerprint is taken right after `BASE_HEAD` and again after the run: porcelain v1 -z + `git diff --binary HEAD` + untracked hashes → `git hash-object --stdin`.
  - `ok` = fingerprint changed OR commits made.
  - New JSON fields `predirty_files` and `worktree_changed_since_start`.
  - Header comment fixed; the suite wired into `make test-launchd-drivers`; changelog fragment added.
- **Judge r1 PASS 86**, 0 blocking. Finding 1 was reproduced first-party: the batched `xargs git hash-object` aborts at a dangling symlink (`fatal: could not open 'aaa'`, rc=1), so a later untracked edit reads `empty_worktree` (a live lane reported dead).
- **Executor r2** `2e49cc9ac`: untracked entries are hashed one at a time, with stand-ins (symlink target; nested repo HEAD + porcelain). Arms 8.8–8.11 added.
- **Judge r2 PASS 85**, 0 blocking.
- Pushed through the pinned guard (`core.hooksPath` = pin); PR **#1593**.
- **Gate 3b**:
  - `lint` and `docs-gate` green (docs-gate after one re-run). `test` still running at record time.
  - `changes` sat unassigned 15 min and was cancelled, so `launchd drivers` and `test-windows` were skipped.
  - githubstatus: **Actions: major_outage**, "Incident with Actions" investigating since 21:09Z. The same symptom hit the 19:17Z dev dispatch.
  - Not merged; ticket NOT resolved.

**Progress**: 0 tickets resolved (38 open).
- Clause 1 UNMEASURED.
- Clause 2 UNMET.
- Clause 3 MET.
- Clause 4 prior evidence only.
- Clause 5 preserved: nothing merged, and the judged head waits for a real `pull_request` CI suite.

**Routing evidence**: base-gate1=a12a319b5eb9b208f4d8f5df21b6a892033b08e7@2026-10-05T19:14:56Z. `MISSION_ROUTING_NOTE`: codex over daily ration → planner opus, executor claude:claude-sonnet-5-5.
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer**: NOT SPAWNED. The resolver said `recipe claude:claude-opus-5-5`. The defect, cause and fix shape were measured at HEAD in one script, so the charter's "design only when the fix warrants one" applies, as in iteration 19.
- **Planner**: resolver `agent-tool opus fail-closed:env-pin` → opus via the Agent tool, foreground (87,227 tok, 17 tools, 13 min).
- **Executor**: resolver `recipe claude:claude-sonnet-5-5` → `claude-sub` in `~/.ailang-worktrees/fleet-i20-pi-predirty`. Billing CLEAN; probe rc0; r1 rc0, r2 rc0 (tok: not reported, text mode).
- **Evaluator**: resolver `reroute pi:openrouter/minimax/minimax-m3 generator-equals-judge`.
  - OpenRouter is in `MISSION_OVER_RATION`, so minimax was skipped. `claude:claude-sonnet-4-6` was skipped as the executor's family.
  - Ran `opus` via the Agent tool, foreground for r1 (92,747 tok, 20 tools, 22 min), resumed by SendMessage for r2 (105,477 tok cumulative, 16 min).
  - Judge ≠ generator (opus vs sonnet-5-5). FLAG: same model as the controller and the planner, in a fresh context.
- **Metered**: $0.00. **Ration**: codex, ollama and openrouter over; Anthropic subscription OK.

**Verification** (judge, first-party re-runs):
- `scripts/test_mission_pi_run.sh` 36/36.
- commits 9, sandbox 13, provider_quota 29, all rc0.
- `make test-launchd-drivers` rc0, now running this suite.
- `bash -n` rc0.
- shellcheck identical to base.
- `check-changelog` rc0.
- Mutations red-on-mutate, each restored byte-identical (cmp + `git diff --quiet`):
  - `ok` reverted to porcelain;
  - diff-only fingerprint;
  - porcelain-only fingerprint;
  - untracked hashes dropped;
  - batched `xargs` restored;
  - time-varying fingerprint;
  - fingerprint diffed against `BASE_HEAD`;
  - post-run fingerprint failure counted as changed.
- One judge mutation SURVIVED: the stand-in block replaced with `|| true` → 36/36, so no arm pins the stand-in (non-blocking follow-up).
- 2,000-untracked-file timing: 18.06 s per-entry vs 0.098 s batched, same hash. It runs outside the wall clock.

**Done-gate** (pending landing):
- Surface ✓ (`scripts/mission_pi_run.sh` is what every pi lane runs).
- Reach ✗ (not on origin/dev).
- `make test-launchd-drivers` ✓ on the branch.
- Nothing reloaded ✓.
- No dry-run needed: 0 driver bytes changed.

**Ruled out**:
- Merging past the skipped `launchd drivers`/`test-windows` legs, or treating an outage green as a verdict (Gate 3b).
- A third executor round for the judge-r2 non-blocking findings (one revision, bounded; queued as follow-ups instead).
- Spawning a designer for a one-script fix.
- Touching `TestOllamaQuota*` (D-FLEET-12/14) or #1583.
- Reading or acking `mission-fleet`.
- Re-asking D-FLEET-13/14.

**Retro lane**: none for the skill. One observation: the GitHub `changes` path-filter job is a single point of failure. When it cannot get a runner, every path-gated leg is SKIPPED, which renders as neutral rather than red. A rollup that counts only failures reads that as clean.
**Next**: land #1593 when Actions recovers (resume predicate in the queue row), resolve the ticket and reply to world. Then the Phase 3a skill-resolution directive. The D-FLEET-12 pair is planner-ready; D-FLEET-13/14 await Mark.

## 21 — 2026-10-06 — pi-runner pre-dirty fix LANDED: #1593 `c2bf04af3`, re-judged PASS 100 after the Actions incident cleared; ticket resolved [HARNESS]

**Pick**: resume of P1 #5 `pi-runner:verdict-blind-to-commits-and-predirty` (pre-dirty half). It was iteration 20's PARKED-ON-CLOCK item, and its predicate was met at Gate 2.
- Gate 0: kill switch armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing CLEAN; 0 directives on #1584 since `2026-10-05T12:33:09Z` (2 comments, both public). Rotation-week catch: #1380 also had 0 directives since the watermark. 39 open tickets.
- Gate 1:
  - Pin = origin/dev `a12a319b5`.
  - Running skill: `SKILL.md` + 5 resources match origin; 7 gate resources DIFFER. The resolved main checkout is 22 commits behind (the same #1578 stamp-form delta as iteration 20). This fire followed the resolved copy and used the absolute stamp form. This is open ticket P1 #6, measured live.
  - Dev `CI` at `a12a319b5` was `failure`: iteration 20's `workflow_dispatch`, where `changes` was cancelled with no runner during the incident. Not a code verdict.
  - Full check set: 14 checks; Sonar red (inherited); `changes` cancelled.
- Gate 2: predicate re-run as commands, not transcribed.
  - githubstatus: Actions `operational`; "Incident with Actions" resolved 2026-10-05T22:49:42Z.
  - #1593 head `2e49cc9ac`: required `test`/`lint`/`build`/`docs-gate` green, plus `launchd drivers` and `test-windows` (CI attempt 2).
  - `UI build gate` red: 0 steps, empty runner, cancelled at 15 min. An outage casualty; re-run → success (attempt 2, post-incident).
  - Base still `a12a319b5`, so the judged bytes = the PR head.

**Did**:
- **Re-judged before merge.** Every PR green had run during the incident, which licenses a code inference only (Gate 3b).
  - Resolver: `reroute pi:openrouter/minimax/minimax-m3 generator-equals-judge`. openrouter was not over ration this fire.
  - The judge ran via `scripts/mission_pi_run.sh` in the isolated worktree `fleet-iter21-evaluator` at `2e49cc9ac`; session handshake acked (5× `acked:true` in its NDJSON).
  - **PASS 100**, 0 blocking.
- Squash-merged #1593 `--match-head-commit 2e49cc9ac` → `c2bf04af3`.
- Merged iteration 20's record #1594 → `eb2850427`.
- Resolved the ticket with sha `c2bf04af3`; world replied.
- Handed two dev reds to V1 (owner), message `inbox_1791263210185`; body verified via list.

**Gate 3b**:
- `CI` on `c2bf04af3` = **success** (run 37414632276, 04:38–05:05Z, after the incident). All 10 jobs green, including `launchd drivers (bash 3.2)` and `test-windows`.
- Full check set (17), not-green on `c2bf04af3`:
  - **Sonar**: inherited.
  - **`ailang-core-dev` Cloud Build failure**: step 7 `build-agent-base`, `node v22.23.3 != pinned v22.23.2`. NodeSource's `setup_22.x` moved; the pin is in `docker/Dockerfile.agent-base`, outside fleet Authority. It fails on `eb2850427` too; the last success was `58dd06d` (10-05 13:43Z).
  - **`Build macos-latest`**: `TestMemprobeDebugLogDoesNotAccumulate`, the known flake (iteration 19). It passed on #1593's PR head.
  - **2 `cancelled`**: fail-fast siblings.
- `CI` on `eb2850427` (docs-only) = failure: `test-windows` `cmd/ailang` hang guard (TestStdNumeric*, TestTailCallTrace*: "this PACKAGE has outgrown its go test -timeout budget"). It passed on the parent `c2bf04af3`. Second instance after `d273ea1cd`. V1's lane: recorded and handed over.

**Progress**: 1 ticket resolved (39 → 38 open).
- Clause 1 UNMEASURED.
- Clause 2 UNMET: this ticket took ≈9.6 days from filing (2026-09-26) to resolve.
- Clause 3 MET.
- Clause 4 prior evidence only.
- Clause 5 preserved: judged twice, and dev CI green on the merge SHA after the incident.

**Routing evidence**: base-gate1=a12a319b5eb9b208f4d8f5df21b6a892033b08e7@2026-10-06T03:50:59Z; base=eb28504275ce09ec57a62b21e9a964cdc30e9422@2026-10-06T04:48:03Z (gate4). `MISSION_ROUTING_NOTE`: codex over daily ration → planner opus, executor `claude:claude-sonnet-5-5` (neither spawned).
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer / Planner / Executor**: NOT SPAWNED. This was a resume of work iteration 20 had already planned, built and judged; there was nothing to design, plan or execute. The operator's standing request to use the Agent tool for roles was honoured where a role ran.
- **Evaluator**: the resolver said `reroute pi:openrouter/minimax/minimax-m3`, a `recipe` lane, so the Agent tool is not the specified path. Ran via `mission_pi_run.sh`: pi_rc 0, 2094 s, 112 tool executions, 113 turns, 7,415,860 tok gross, $0.5452.
  - The runner verdict was rc 10 `empty_worktree`. Cause: the CONTROLLER's directive put the report under `.ailang/state/` (gitignored, `.gitignore:105`), so the porcelain check could not see it.
  - The report existed and was fresh: mtime 06:37 local, after the 06:02 start.
  - So this was an instrument artifact of the directive, not a lane failure. The verdict was taken from the report.
  - Judge ≠ generator: minimax vs sonnet-5-5 (executor) and opus (controller/planner).
- **Metered**: $0.5452. **Ration**: codex and ollama over; openrouter and Anthropic OK.

**Verification** (judge, first-party, in its worktree):
- `scripts/test_mission_pi_run.sh` 36/36; pi_run suites commits 9/9, sandbox 13/13, provider_quota 29/29; `bash -n` rc0 on both scripts; `check-changelog` rc0; shellcheck: no new findings.
- Mutations, each restored by `cp` + `cmp` rc0:
  - M-a (`ok` = porcelain only): 4 red, including 8.1;
  - M-b (untracked hashes dropped): 4 red, including 8.6;
  - M-c (post fp = pre fp): 12 red.
- Hand repro in a scratch repo:
  - A: mutated (old) code, pre-dirty, no-op pi → `ok` (the bug);
  - B: fixed code, same input → rc 10;
  - C: fixed code with a further edit → `ok`.
- Its `make test-launchd-drivers` gave rc 2 inside the pi sandbox, with the SAME 68 failures at the base `a12a319b5`. It is a sandbox artifact (`mktemp` denied under `DARWIN_USER_TEMP_DIR`, so the judge shimmed `mktemp`). The CI `launchd drivers (bash 3.2)` leg is green on both the PR head and `c2bf04af3`.

**Done-gate**:
- Surface ✓ (`scripts/mission_pi_run.sh`).
- Reach ✓ (`c2bf04af3` is an ancestor of origin/dev).
- `make test-launchd-drivers` ✓ (CI leg green on the merge SHA).
- 0 driver bytes, so no dry-run.
- Nothing reloaded ✓.

**Ruled out**:
- Merging on the incident-time greens alone (re-judged, and the merge SHA got its own post-incident CI).
- Fixing the node pin or the Windows test budget (outside Authority; handed to V1).
- Ticketing the judges' non-blocking follow-ups (no product loop has hit them; ticket-driven only).
- Reading or acking `mission-fleet`.
- Re-asking D-FLEET-13/14.

**Retro lane**: none for the skill. Two observations, one occurrence each:
- (1) A directive that sends a pi evaluator's report to a gitignored path makes `mission_pi_run.sh` read `empty_worktree` even when the judge did its work. The fix is the directive's path (a tracked scratch path, or check `check-ignore` before choosing one). Not yet ≥2 frictions.
- (2) The minimax judge ran `git stash` / `git stash pop` around a baseline run. The shared stash stack was empty (verified before the pop), so nothing was lost. A non-empty stack would have popped another session's entry.

**Next**: P1 #6 `skill-surface:main-checkout-not-synced-to-dev`. Measured live this fire: 22 behind, 7 gate resources differ. It feeds the Phase 3a skill-resolution directive. The D-FLEET-12 pair is pre-authorized; D-FLEET-13/14 await Mark.

## 22 — 2026-10-06 — Gate-0 self-notice read built + judged PASS 87→91→93 as PR #1604; merge blocked by an inherited dev red (07e1a89bc) → PARKED-ON-CLOCK; P1 #6 parked as D-FLEET-15 [HARNESS]

**Pick**: P1 #7 `gate0:driver-crash-notices-invisible` (ailang#1160). Mark's queue head, P1 #6 `skill-surface:main-checkout-not-synced-to-dev`, was skipped for a measured reason. Its fix is either (i) the driver auto-fast-forwarding the shared main checkout, whose standing authorisation `gate-1-observe.md` reserves to Mark, or (ii) Phase 3b per-fire skill pinning, which the 2026-09-26 directive gates on the 3a spike and parks for Mark. Neither is a mechanical fix the fleet can land alone, so it is filed as **D-FLEET-15** (rec A, default B). Measured this fire: the main checkout is `1 0` (ahead/behind origin/dev), so the symptom was not live.
- Gate 0: kill switch armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing CLEAN; 0 directives on #1584 since `2026-10-05T21:22:17Z` (3 comments, all public). Not a sweep week (iteration 19 swept).
- Gate 1:
  - Pin = origin/dev `e68a264fb`, drift 0.
  - Running skill: `SKILL.md` + all 12 resources byte-identical to origin, for both the resolved symlink and the pin copy.
  - Dev `CI`, `Build and Release` and `Docs-Deploy` all success on `e68a264fb`. Full check set: 21 checks; Sonar red (inherited), `ailang-core-dev` Cloud Build red (node pin, handed to V1 at iteration 21).
- Gate 2: premise re-verified at HEAD. `gate-0-preflight.md` had no self-notice read (`grep -n 'slot-verdict\|rc=143\|crash'` hit only unrelated prose). The driver literal is at `mission-control.sh:2514`. World's implementation is `ailang-world` `50103fd` (#137).

**Did**:
- Planner opus (Agent, foreground): plan `6934d2a0b`. Premises it corrected:
  - The shared namespace has no non-rotating watermark, so the rule passes both issue-scoped files.
  - World's script holds the prefix twice (bash and a `\u`-escaped python copy), and its `--driver-src` check covered only the bash copy. The port keeps a single literal.
  - `--control 107:4` is World-only. Measured controls: `ailang#852` = 10, `world#107` = 4, `stapledons-godot#4` = 5.
  - World has no `scripts/` copy, so the root comes from the driver.
  - A second death notice, `Mission slot verdict`, is out of scope (follow-up).
- Executor `claude:claude-sonnet-5-5` (`claude-sub`, probe rc 0):
  - Run 1 died after 227 s with `API Error: Server error mid-response`, M1 committed (`a3cbffe1e`). The remaining chain was ollama/openrouter, both over ration, so the same lane was retried with a resume note.
  - Run 2 re-verified M1 and finished M2–M4: `0336d0e30`, `7b7b39afc`, `40fb95a7f`.
  - Round 2 `949d5952b` and round 3 `2e0f92672` fixed the judge's findings.
- Deliverables:
  - `scripts/mission_gate0_self_notices.sh`: exit codes 0/1/2, floors F0–F10, bounded `gh`, never prints a body.
  - `scripts/test_mission_gate0_self_notices.sh`: 108 arms, wired into `make test-launchd-drivers`. World's four fixtures are `cmp`-identical, plus a real #1380 capture under `scripts/mission_gate0_self_notices_testdata/`. The `driver-literal` arm turns red if the driver's prefix drifts.
  - Gate 0 step 6a in both skill copies (`cmp`-identical): a second read with no authority. rc 1 sends Gate 2 to its died-mid-flight traces and gives a log credit once per `url=`. rc 2 is an instrument or precondition failure, not a verdict.
  - Changelog fragment.
- Judge opus (Agent, own worktree `.wt-fleet-iter22-judge`), 3 rounds:
  - **R1 PASS 87**, 0 blocking. Mutants m1, m2, m5–m9 and `drv` were killed. m3 (prev-issue read dropped) SURVIVED, because the prev issue always doubled as the control. F3: the 6a block failed as written in a fresh shell (`$ISSUE` unset; `${…:?}` inside `$(…)` gave rc 127).
  - **R2 PASS 91**: a new arm kills m3, and the block is self-contained. New finding R2-1: a precondition failure printed `rc=1`, the "fire died" token.
  - **R3 PASS 93**: preconditions now print `rc=2 (… not a verdict)`. Cases B–E verified rc=2, and A printed rc=1 for real data.
- PR #1604 opened with no closing keyword (`Refs #1160`).
- Handed dev's new red to V1: `inbox_1791290588115_58891a9a`.

**Gate 3b / blocked**: #1604's `test` and `test-windows` fail `TestValidateModulePath_SingleFileInsidePackage` (`package_layout_test.go:107`). The negative control is dev `07e1a89bc` (attended `fix(pipeline)…MOD010…`, 12:07Z), which fails the SAME test on `test` and `test-windows` (`CI=failure`, `Build and Release=failure`). `8cddc7e2a` sits on top of it. The diff has no Go, `internal/pipeline` is outside fleet Authority, and V1 owns the repo's reds. Not merged; ticket stays open.

**Resume predicate**: dev `CI` success on a SHA containing a fix for that test → `gh run rerun --failed` on #1604 (head `2e0f92672`) → required `test`/`lint`/`build`/`docs-gate` + `launchd drivers` + `test-windows` green → merge `--match-head-commit 2e0f92672` → Gate 3b on the merge SHA → `ailang mission ticket resolve gate0:driver-crash-notices-invisible --sha <merge>` → comment the verdict on #1160 (`--body-file`), then close it.

**Done-gate (pre-merge)**:
- Surface ✓: `scripts/mission_gate0_self_notices.sh` + both `gate-0-preflight.md` copies, the files every mission's Gate 0 reads.
- Reach: pending merge.
- `make test-launchd-drivers` rc 0 (executor and judge, unpiped).
- 0 driver bytes, so no dry-run.
- Nothing reloaded.

**Live finding**: the instrument's first fleet reading (rc 1) is the real `rc=143` KILLED-at-gate-5 slot of iteration 18 (2026-10-04T04:28:01Z; `/tmp/ailang-mission-fleet.log`: `STALL … killing early`, `slot-verdict: KILLED at=gate-5 rc=143`). Iteration 18's Gate-4 record had already landed, so nothing was orphaned. Credited here once (url on #1380).

**Progress**: goal unmoved (0 tickets resolved; 38 open).
- Clause 1 UNMEASURED.
- Clause 2 UNMET.
- Clause 3 MET.
- Clause 4 prior evidence only.
- Clause 5 preserved: nothing merged.

**Routing evidence**: base-gate1=e68a264fbea787896a963f0fd7ae35cb321bf64b@2026-10-06T11:41:04Z; worktree base=e68a264fb (MATCH at snap); base-gate4=8cddc7e2ac8120429dc8904ad8cbeda12a8f6c91@2026-10-06T12:45:24Z (dev advanced by 4 attended commits, record branched from there). `MISSION_ROUTING_NOTE`: codex over daily ration → planner opus, executor `claude:claude-sonnet-5-5`.
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer**: NOT SPAWNED. This is a port of World's implemented, judged design (`w-gate0-blind-to-its-own-crash-notices.md`); the plan cites it.
- **Planner**: opus via Agent (resolver `agent-tool opus fail-closed:env-pin`), 119,537 tok.
- **Executor**: `claude:claude-sonnet-5-5` via `claude-sub` recipe (`declared:provider-pin`), 4 runs (227 s died / ~13 min / round 2 / round 3), tok not reported (`claude -p` text output).
- **Evaluator**:
  - The resolver said `reroute pi:openrouter/minimax/minimax-m3 generator-equals-judge`, but openrouter was **over ration** (`ailang mission quota`: $3.09 against $2.33/day).
  - The `reroute` arm of `resolve-role-spawn.sh` never calls `_rs_over`, unlike `emit_recipe`.
  - Recorded dead via `mission-lane-dead.sh`: minimax (over ration), `claude:claude-sonnet-4-6` and `sonnet` (family sonnet = executor).
  - Spawned the declared bare fallback **opus** via Agent: 114,415 + 128,471 + 135,141 tok over 3 rounds.
  - Judge ≠ generator: opus vs sonnet-5-5.
- **Metered**: $0. **Ration**: codex, ollama and openrouter over; Anthropic OK (5h 33%, week 24%/25.7%).

**Ruled out**:
- Working P1 #6 autonomously (needs Mark: D-FLEET-15).
- Fixing `TestValidateModulePath_SingleFileInsidePackage` (language core, outside Authority; V1).
- Merging over the inherited red.
- Re-asking D-FLEET-13/14.
- Reading or acking `mission-fleet`.
- Resolving `weekly-report:unknown-mission`: attended `08ce76f30` appears to fix it, but it is unverified this fire. Next fire: verify, then resolve with that sha.

**Retro lane**: no skill edit (no gap has reached ≥2 frictions this fire). Observations, one occurrence each:
- (1) `resolve-role-spawn.sh`'s `reroute` arm skips the ration gate, so it pointed the judge at an over-ration bucket.
- (2) `ailang mission rotate-log fleet --stream status` resolves to the registry root (the PIN), not the CWD. It rotated the pin's `fleet-mission-status-archive.md` (18 → 3 entries plus `-old`/`-index` files) instead of moving the charter's stamps, so a record written in a separate worktree cannot use it. The controller backed up and restored the pin byte-for-byte from HEAD (status clean), then moved the stamp by hand with line-count and archive assertions.
- (3) A claude-sub executor can die on a transient API 5xx with an rc-0-looking partial transcript. The rc file (rc=1) and the commit log were the real signal.

**Next**: when dev `CI` is green again: rerun #1604, merge, resolve P1 #7. Then P1 #8 `quorum:zero-signal-guard-vacuous-with-controller-verdict` (premise re-read this fire: `internal/mission/quorum/quorum.go:164` increments `presentCount` for the controller before the zero-signal guard at :176; inside Authority). Then P1 #9 and the Phase 3a spike. D-FLEET-13/14/15 await Mark.

## 23 — 2026-10-06 — P1 #8 verified live at HEAD, but every role lane is over ration (ollama hard-capped mid-fire by this controller's own session) → PARKED-ON-LANE; zero roles spawned, nothing landed; both iteration-22 PRs still blocked [HARNESS]

**Pick**: P1 #8 `quorum:zero-signal-guard-vacuous-with-controller-verdict` (ailang#651). P1 #6 is parked on D-FLEET-15 (iteration 22); P1 #7 is built but its landing is blocked on V1's inherited dev red; P1 #8 was the top routable row of Mark's triage order.
- Gate 0: kill switch armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing tripwire CLEAN; 0 directives on #1584 since `2026-10-06T05:38:39Z` (4 comments, all public or this loop's own); 39 open tickets. Weekly rotation not due: #1584 was created `2026-10-05T12:32:44Z`, after the Monday-07:00-local boundary, and holds 4 comments (<80).
- Gate 1: pin = origin/dev `04dc2b7c8`; local `dev`, the pin and the resolved main checkout are byte-identical to origin on `SKILL.md` + all 12 resources (full drift sweep over both readable copies, per-file verdicts printed). P1 #6's symptom is therefore measured ABSENT this fire — the main checkout authored HEAD itself (attended session committed `8cddc7e2a`→`3557cd22c`→`04dc2b7c8` today); the defect (nothing fast-forwards it) stands, parked on D-FLEET-15. Dev HEAD `04dc2b7c8`: `test` FAILURE on `TestValidateModulePath_SingleFileInsidePackage` (V1's attended lane; handed to V1 at iteration 22), `test-windows` FAILURE, `ailang-core-dev` Cloud Build node-pin failure (known since iteration 21), `docs-build` in flight; full check set `checks=18` (control fired).
- Gate 2: the pick's premise re-verified first-party at `04dc2b7c8` (`quorum.go:146-183`: the controller verdict increments `presentCount` at `:170`, ahead of the zero-signal guard at `:176`). Died-mid-flight sweep: iteration 22's PR #1604 (judged PASS 93, head `2e0f92672`) and record PR #1605 are open and unmerged, blocked by the same dev red — CREDITED here and carried by this record, not redone. No iteration-23 PR/worktree existed before this fire. Grep-the-index: no prior iteration worked P1 #8.

**Did**:
- Gate 3 routing attempt, all four roles measured before any spawn:
  - Resolver: designer `recipe pi:ollama/glm-5.3:cloud declared:provider-pin`; planner `recipe pi:ollama/kimi-k3:cloud over-ration-reroute:codex`; executor `recipe pi:ollama/deepseek-v4.1-flash:cloud declared:provider-pin`; evaluator `refuse over-ration:anthropic` (reason token recorded verbatim).
  - `mission-lane-check.sh fleet`: every rung of every role `skip:over ration` — anthropic ENFORCED (headroom 0.6pp < 1pp start margin; long window 31.0/31.5%), codex (0.4pp < 2pp), openrouter ($3.59 of $2.33/day), ollama (0.8pp < 3pp at check time).
  - This session exposes no Agent/Task tool, so the `agent-tool` spawn path was unavailable as well; the pi lanes require `scripts/mission_pi_run.sh`, which the same ration blocks.
  - Mid-fire the ollama bucket crossed its hard cap: 6.1pp → 10.1pp of the 10pp/day ration, consumed by the controller's own pi-rung session (fractional session gauge 23.7%) reading its rulebook — the only ollama consumer after stapledon's fire ended 18:47:54Z. A pi-rung controller structurally starves the role lanes sharing its bucket.
  - Generator ≠ judge: the REQUIRED evaluator cannot run on any lane, so nothing may land. P1 #8 parked **PARKED-ON-LANE** (standing rule 8b): role = evaluator (and designer/planner/executor), refusals above, resume predicate = "`mission-lane-check.sh fleet` rungs ok + a judge lane ≠ the executor model". It never entered DECISIONS (rule 8c): its resume is a clock, not an ask.
- Landing predicates re-run as commands: dev `test` still red at `04dc2b7c8` → #1604 and #1605 stay parked; no re-run attempted (the red is outside fleet Authority and nothing in either diff changed).
- Bookkeeping only, otherwise: this record carries iteration 22's log entry and STATUS stamp (its PR #1605 is superseded by this PR and closed with a comment), the dashboard overwritten, the index extended, the iteration chain posted (controller stage only, `quota_tokens` not reported — same treatment as iteration 21's controller row; the ollama ration reads the provider's own gauge, so it is not undercounted by that omission).

**Gate 3b**: nothing pushed for landing. The record PR inherits dev's required-`test` red, so it waits with #1604. Resume predicate for both: dev's required `test` green at HEAD (V1's `TestValidateModulePath_SingleFileInsidePackage`). **Late re-measure at end of fire:** dev advanced mid-flight to `c57d4f00e` (attended M-TEST-RUNNER Phase 2 `70e4f77c8` + release v0.52.2), so the predicate was re-run as a command against the newest head, bounded: CI run 37515762996 completed **failure** at 19:18Z, `test` and `test-windows` both failing the SAME `TestValidateModulePath_SingleFileInsidePackage` (known-positive control: 157 `ok`/`--- PASS` lines in the same log). The park therefore stands on that head; #1605 was closed as superseded by #1611 during this fire. A THIRD dev head landed while the routing row was being stamped — `261505a86` = "fix(test): MOD010 single-file layout test fixture outside /tmp", the failing test's own fixture file — and its CI run (37516731812) was still in flight at the bounded wait's deadline (19:36:40Z, ~28 min elapsed), so it is recorded UNMEASURED, neither green nor red; the next fire re-runs the predicate as a command at whatever head it finds.

**Progress**: 0 tickets resolved (39 open).
- Clause 1 UNMEASURED.
- Clause 2 UNMET (0 resolved this fire; oldest open tickets date to 09-26).
- Clause 3 MET (39 open signatures, all in `mission-fleet`).
- Clause 4 prior evidence only (39 open tickets, so the fire proceeded under the driver's exit-at-0 rule).
- Clause 5 preserved trivially: nothing merged, nothing judged, nothing landed unjudged.
**Progress** (digest line): goal unmoved — a capacity park; P1 #8 is verified-routable the moment a lane reopens.

**Routing evidence**: base-gate1=04dc2b7c875f5d165fca2af352d0b0d93c4f09ed@2026-10-06T18:49:20Z. base=261505a86f7951acea7e40a9eb36e8a877daf401@2026-10-06T19:19:00Z (gate4; the record worktree is branched from `8b6354778` = origin/fleet/iter22-record, carrying iteration 22's entry; origin/dev had moved twice mid-fire — `04dc2b7c8` → `c57d4f00e` → `261505a86` = "fix(test): MOD010 single-file layout test fixture outside /tmp", the failing test's own file — by the time this row was stamped). `MISSION_ROUTING_NOTE`: codex over daily ration → planner opus, executor `claude:claude-sonnet-5-5` (both lanes were over ration this fire; the note is the driver's plan, not what ran).
- **Controller**: `pi:ollama/glm-5.3:cloud` (pi fallback rung; tok: not reported; this session took the ollama bucket 6.1pp → 10.1pp/day, session gauge 23.7%).
- **Designer**: NOT SPAWNED — resolver `recipe pi:ollama/glm-5.3:cloud declared:provider-pin`; the ollama bucket is over its hard cap and this session exposes no Agent tool. No fallback link was runnable (declared chain all over ration).
- **Planner**: NOT SPAWNED — resolver `recipe pi:ollama/kimi-k3:cloud over-ration-reroute:codex`; same capacity block.
- **Executor**: NOT SPAWNED — resolver `recipe pi:ollama/deepseek-v4.1-flash:cloud declared:provider-pin`; same capacity block.
- **Evaluator**: NOT SPAWNED — resolver `refuse over-ration:anthropic`, recorded verbatim per the spawn-pattern rule; the declared chain (`pi:openrouter/minimax/minimax-m3` → `claude:claude-sonnet-4-6` → `opus`) is fully over ration. The operator's standing request for this run makes the independent judge REQUIRED, so rather than land anything on the controller's own verdict the iteration parked. No judge verdict exists this fire, by design — that absence is the recorded outcome, not an omission.
- **Metered**: $0.00. **Ration**: ALL FOUR buckets over — anthropic ENFORCED (0.6pp headroom), codex (0.4pp; 2 reset credits in reserve, attended only, next expires 2026-10-22), openrouter ($3.59/$2.33), ollama (10.1pp of 10pp/day, hard-capped).

**Ruled out**:
- Spawning any role on an over-ration lane, or re-prompting in place (the recipe rule: a non-zero verdict is a lane failure → fall back; every fallback link is over ration too).
- Landing #1604/#1605 across the inherited red, or re-running their checks while dev's `test` fails for a reason outside fleet Authority (V1's lane).
- Working P1 #6 without D-FLEET-15's ruling (default B holds: wait for Phase 3b skill pinning).
- Controller-authored implementation of P1 #8 (generator ≠ judge).
- Reading or acking `mission-fleet` (unread = open).
- Re-asking D-FLEET-13/14 (pending, unchanged).

**Retro lane**: none for the skill (nothing reached the ≥2 bar). Observations, one instance each:
- (1) A pi-rung controller consumes the same ollama daily ration its designer/planner/executor lanes need; this fire the controller's own gate-reading took the bucket to its hard cap. If it recurs it is a routing-policy question for Mark (controller rung vs role lanes), with the evidence bar at ≥3 rows.
- (2) This session's read tool elided large outputs to ~2KB (a "quality-monitor" truncation), so the prescribed "read the resource file NOW" had to go through `sed`/`cut` extraction for gate-3's long lines. A harness-instrument friction, recorded here; the fleet cannot file a ticket for its own slot.
- (3) Iteration 22's two unfiled harness notes (resolver reroute ignores the daily quota check; `rotate-log` writes to the pin worktree regardless of CWD) are now in the log via the carried entry; neither bit this fire — the STATUS rotation was done by hand with the line-count assertions, and the archive was verified to hold the moved stamp.

**Next**: when capacity returns — (a) re-probe with `mission-lane-check.sh fleet`; (b) rungs ok → route P1 #8 through designer → planner → executor → evaluator (#651 rules the fix direction; a small design doc plus a one-milestone plan suits it); (c) when dev's `test` is green → land #1604, resolve `gate0:driver-crash-notices-invisible`, and merge the record PRs. Then the Phase 3a skill-resolution spike (after P1, before P2). D-FLEET-13/14/15 await Mark.

## 24 — 2026-10-07 — native role pins absent; planner/evaluator Agent pins rejected → PARKED-ON-LANE, no acceptance [ADMIN]

**Pick**: banked P1 #8 `quorum:zero-signal-guard-vacuous-with-controller-verdict`; this fire cannot route it because the newly open `blocking=all` ticket `agent-tool:mission-role-pins-unavailable` applies first-party. P1 #6 is parked on D-FLEET-15 and P1 #7 is built in #1604, awaiting dev health. Pending record #1611 contains iterations 22–23 and was read before assigning iteration 24.

**Reality check**: verified by controller at origin/dev `261505a86f7951acea7e40a9eb36e8a877daf401`: `internal/mission/quorum/quorum.go` increments `presentCount` for a controller verdict before checking zero signal. No fix made. Both readable skill directories, pin and resolved user symlink, match origin for SKILL.md and each of 12 resource files (26 explicit MATCH readings). Historical main-checkout drift is not a live mismatch this fire; D-FLEET-15 is preserved rather than self-resolved.

**Preflight**: armed, clean pin tree, correct GitHub account, billing guard CLEAN. `mission_directives.sh --issue 1584 --since 2026-10-06T19:02:43Z` returned 0 new allowlisted directives of 6 comments. No messages read or acked in mission-fleet. Three GCP approvals task-d80f4d05/task-98730ece/task-50acfbe6 remain with their operator, untouched. Injected docs sprint is unrelated and was not advanced.

**Outcome**: PARKED-ON-LANE. Zero designs, plans, implementation milestones, judgements or merges this fire. The operator requires native Agent transport and a required independent evaluator. No declared usable independent judge exists here, so controller-only acceptance is forbidden.

**Routing evidence**: base-gate1=261505a86f7951acea7e40a9eb36e8a877daf401@2026-10-07T01:39:40Z; base=261505a86f7951acea7e40a9eb36e8a877daf401@2026-10-07T01:41:09Z (gate4). Record worktree is branched from origin/dev; five mission bookkeeping files from b5d516b85 (#1611) were carried forward after confirming origin changed none of those files since their merge base. Controller native Codex (model/token total not exposed). Resolver commands each returned shell rc 0: routing refusal is encoded in their output, not a nonzero exit. Native spawn failures were tool errors, with no shell rc. `MISSION_ROUTING_NOTE`: absent; not represented as configured. Driver per-role model pins, fallback chains and ration data are absent; quota status UNMEASURED. Metered external role calls $0.00.

| Role | Resolver / Agent result | Fallback actually used | Tokens |
|---|---|---|---|
| Designer | `refuse fail-closed:designer-model-missing` | None. No declared pin, native spawn cannot be selected without inventing a lane. | 0 role-run tokens |
| Planner | derive `opus fail-closed:env-pin`; resolver `agent-tool opus fail-closed:env-pin`; native spawn `model=opus` → `Unknown model opus` | None; required pin unsupported. | 0 role-run tokens |
| Executor | `refuse fail-closed:executor-model-missing` | None. No approved plan or declared executor pin; default session inheritance refused. | 0 role-run tokens |
| Evaluator | `refuse fail-closed:evaluator-model-missing`; native capability probe `model=sonnet` → `Unknown model sonnet` | None. No judgement, independent or controller-authored. | 0 role-run tokens |

Available native enum reported by both failed spawns: gpt-6.1-sol, gpt-6-astra, gpt-6-sol, gpt-6-luna, gpt-5.6-sol. Retired models and dormant Astra were not used. A same-model diagnostic agent is not an authorized independent model judge. No role ran or ended on a background wait.

**Resume predicate**: re-run the four `resolve-role-spawn.sh <role>` commands with the driver-exported environment; require valid declared lanes and a supported native planner plus independently routed evaluator that actually returns a judgement. Alternatively an attended routing ruling may supply supported native pins. The required native Agent transport is retained. No reset time is known. This is a lane/capability park, not a new human decision; never file it as needs-human-review. No fleet self-ticket filed (fleet may not file its own slot). The signature is already open from world/stapledon.

**Gate 3b**: no push of implementation, no merge, no LANDED verdict. Re-measured dev CI run 37516731812 on full SHA 261505a86: failure. Complete check set: 21; Sonar failure, test-windows failure, Build windows-latest failure, Build macos-latest cancelled; all remaining checks success/skipped. Linux test is green. Failed Windows log is 248206 bytes; named FAIL lines and PASS lines validate the reader. Three failures: TestTestCommandBytecodeFlags (different temporary paths in evaluator/bytecode errors), TestNamedBatch_RuntimeErrorPositionsMapToSource (error retains temporary-source positions), TestForall_IllTypedPropertyFailsAlone (compile error retains temporary-source position). These are pre-existing relative to this docs-only run, not attributed to a fleet change. V1 owns cmd/ailang/test* and internal/testing; handed over through mission-v1. #1604 remains parked; no inherited implementation or pending record was merged.

**Progress**: goal unmoved — 0 tickets resolved, 41 open signatures. Clause 1 UNMEASURED; clause 2 UNMET (no turnaround improvement); clause 3 MET (canonical ticket queue); clause 4 prior evidence only; clause 5 no new landing or acceptance credit.

**Verification**: PATH CLI observed newly built at 261505a86 after quick-install; make build returned rc 0 for bin/ailang; no source change. Record checks: ledger valid, 3 STATUS stamps, exact previous stamp archived, old log retained byte-for-byte as a prefix, index includes 22–24, git diff --check. No runtime tests claimed for this ADMIN record. Record is draft only, because the required evaluator cannot be spawned.

**Done-gate**: no harness bytes edited; no dry-run, launchd reload or ticket resolution. No done-gate credit claimed.

**Ruled out**: blindly trusting main's iteration-21 state while #1611 banks 22–23; silently inheriting Codex for all roles; CLI substitution against the operator's native Agent instruction; taking a controller-only verdict as an evaluator; resolving D-FLEET-15 from a current match; working the injected docs sprint; resolving pending GCP approvals.

**Retro lane**: backlog / routing-policy signal. World 239 and Stapledon 18 already reported native pins unavailable; fleet measured the same missing environment and unsupported opus/sonnet pins. No routing or skill changes made; a policy change cannot be self-approved. This is capacity, so it adds no DECISIONS row. Last-20 fleet harness share: 16/20 (80%), rows 21,20,19,18,17,16,15,14,12,11,10,9,8,7,6,5; the other four index rows carry their recorded classes. It exceeds one third under the fleet charter's explicit harness mandate, not a product-loop violation. This is reported separately from product clause 1; the fleet's harness mandate makes a high share expected, not a product-loop authority breach. Last three landings (iteration 19 heartbeat and rotate-log, iteration 21 pi predirty) address unresolved harness tickets / turnaround work, though the turnaround bar remains UNMET; no all-none drift verdict.

**Next**: restore admitted role routing → P1 #8; dev Windows green → re-run #1604 at its pinned head, require remote CI plus independent judgement before landing. Then P1 #9 exit-path notices; Phase 3a measurement after P1. D-FLEET-13/14/15 remain OPEN, unchanged. Draft record retains prior pending history; #1611 is not closed until the consolidated record is reviewed.

## 25 — 2026-10-07 — `blocking=all` codex-controller role-env ticket LANDED: #1635 `0ceb1db01`, judged PASS 93; ticket resolved [HARNESS]

**Pick**: `agent-tool:mission-role-pins-unavailable` (blocking=all, World 239/240 + fleet 24). The charter ranks new tickets below Mark's list unless `blocking=all`, so it outranks P1 #6 (parked on D-FLEET-15), P1 #7 (#1604, built) and P1 #8 (parked by iteration 23). Pending record PRs #1611 (22–23) and #1612 (24) were read before numbering this iteration; this record branches from #1612's head and supersedes both.

**Reality check**: controller, first-party. Rig `~/.codex/config.toml` has `shell_environment_policy.inherit = "core"` (mtime 2026-10-06 13:02). `codex sandbox -- /usr/bin/env` (names only) shows `HOME`/`PATH` but no `MISSION_*` and no test var; `-c shell_environment_policy.inherit=all` shows both. The World driver log at 2026-10-07 01:21:49 shows the driver DID resolve all four roles, so the defect is delivery, not resolution. Option measurements: `include_only` with `AILANG_*` admits `AILANG_REGISTRY_API_KEY` and strips the rig's `.set` vars; `inherit=core` + `include_only` adds nothing.

**Preflight**: armed (`mission-fleet.disabled` absent), gh `sunholo-voight-kampff`, billing CLEAN, `mission_directives.sh --issue 1584 --since 2026-10-06T19:02:43Z` → 0 directives of 9 comments. All 13 running-skill files `cmp`-identical to origin/dev (resolved symlink). Local pin == origin/dev `fcce2394c`. Dev check set at `fcce2394c`: `test-windows`/`Build windows-latest` failure (V1's lane, same tests as iteration 24), Linux `test` pending then green.

**Outcome**: LANDED. Helper `tools/launchd/lib/codex-env-args.sh` forwards each exported var on a name allowlist (`MISSION_*`, `AILANG_DRIVER_*`, `AILANG_STORAGE_MESSAGING`, `AILANG_MESSAGES_PROJECT`, `AILANG_MISSION_REGISTRY`, `AILANG_STATE_DIR`, `CONTROLLER_*`) minus a secret-name denylist, plus only the `core.hooksPath` git-config entry renumbered to 0, as `-c shell_environment_policy.set.NAME="<TOML basic string>"`. The codex branch of `_mc_run_once` expands it with the bash-3.2 `set -u`-safe `${arr[@]+…}` form; a names-only summary line goes to the driver log. Ticket resolved with `--sha 0ceb1db01`.

**Routing evidence**: base-gate1=fcce2394c2cb2eaec64081ef89de1994a5502a7e@2026-10-07T19:47:10Z; base-gate3b=0ceb1db01df661c3608f0c3520a6eb28bcd6bab2@2026-10-07T20:49:24Z (drift = this iteration's own merge, 1 commit). Controller `claude:claude-opus-5-5` (driver: codex over daily ration, probe ok → opus) (tok: not reported) · designer: not spawned (no doc; one call site, trade-off measured by controller) · planner `opus` Agent (resolver `agent-tool opus fail-closed:env-pin`; 93,399 tok) · executor `claude:claude-sonnet-5-5` via `claude-sub`, probe rc 0 (tok: not reported, `claude -p` text mode) · evaluator: resolver `reroute pi:openrouter/minimax/minimax-m3 generator-equals-judge`; openrouter in `MISSION_OVER_RATION`, so the next declared rung `claude:claude-sonnet-4-6` via `claude-sub`, probe rc 0, own detached worktree (tok: not reported). **FLAG**: judge is the same vendor and family as the executor, a different model (iteration 22 chose opus over this rung for that reason). No role ended on a background wait; every wait was a bounded `date +%s` poll on an rc/marker file.

**Gate 3b**: PR #1635 required `test`/`lint`/`build`/`docs-gate`/`UI build gate` all SUCCESS on `36f386a61`. `launchd drivers (bash 3.2)` pass, govulncheck pass. Linux `test` first attempt: `go: … proxy.golang.org … stream error: stream ID 1125; INTERNAL_ERROR` at module download (log 90,107 B, 0 `--- PASS` lines, i.e. no tests ran), so it was re-run → SUCCESS. `test-windows`: `TestTestCommandBytecodeFlags`, `TestMCPFile_ListedGolden`, `TestMCPUI_Golden`, `TestNamedBatch_RuntimeErrorPositionsMapToSource`, `TestForall_IllTypedPropertyFailsAlone`, the same signature as dev HEAD (non-required, V1's lane). `Build macos-latest` ended `The operation was canceled` (fail-fast casualty). Merged `--match-head-commit 36f386a61…` → `0ceb1db01`. Dev CI run 37684928778 on `0ceb1db01`: run 37684928778 concluded `failure` ONLY on the non-required `test-windows`/`Build windows-latest`, with the identical five-test set as parent `fcce2394c` (inherited, V1); required Linux `test`, `lint`, `launchd drivers (bash 3.2)`, govulncheck and the `ailang-core-dev` Cloud Build are all success, so this counts as the infrastructure green.

**Progress**: 1 ticket resolved (filed 2026-10-06T23:23Z → resolved 2026-10-07 ≈21h, inside clause 2's 48h). Clause 1 UNMEASURED; clause 2 UNMET (median still over 48h); clause 3 MET; clause 4 prior evidence only; clause 5 preserved.

**Verification**: executor suite 18/18 + `make test-launchd-drivers` MAKE_RC=0 + shellcheck (helper/test 0; driver 28→29, one SC1091 for the new `.` line, same class as the existing `lane-probe.sh` source) + `check-changelog` rc 0; executor mutations (append dropped 6 red, call-site expansion removed 4 red, `*TOKEN*` dropped 2 red, hooksPath forward off 1 red). Judge: re-ran suite and `make test-launchd-drivers`, own mutations J1 denylist removed → `secret-names-denied`, J2 control-char guard removed → `toml-encode-unit`, J3 scope-guard block removed → `argv-scope-guard-forwarded`. Controller live: helper forwarded=49 in this mission shell, and `codex sandbox <args> -- resolve-role-spawn.sh` matched the native shell's four lines exactly; the control reproduced `refuse fail-closed:{designer,executor,evaluator}-model-missing`.

**Done-gate**: surface ✓ (only codex-controller call site); reach ✓ (origin/dev); `make test-launchd-drivers` ✓ (executor, judge, CI); dry-run not run, because `MISSION_DRY_RUN=1` exits at driver :1926, before role-env export (:2180) and the call site (:2274), as the plan measured; the e2e arms and the live check cover it; nothing reloaded ✓. First live proof will be the next codex-controller fire's `codex-env: forwarded=N` log line.

**Ruled out**: `inherit=all` (exposes every key); `include_only` (leaks `AILANG_REGISTRY_API_KEY` via `AILANG_*`, strips rig `.set`); editing `~/.codex/config.toml` (rig config, outside Authority, and anything else that rewrites it would bring the defect back); a design doc (a single call site).

**Retro lane**: backlog. Two non-blocking judge notes: `mc_toml_basic_string $'\x00'` returns `""` rc 0 (unreachable, since env cannot hold NUL); forwarded values are visible in `ps` (denylist keeps secrets out). Follow-up candidate for an attended ticket or decision: under a codex controller, a bare-alias evaluator pin (`sonnet`/`opus`) still resolves to `agent-tool <alias>`, which codex `spawn_agent` rejects (planner measured 1 of 26 codex-controller fires). That is the other half of World's evidence and may be routing policy, so it was not built. No skill edit (no second friction on one gap).

**Next**: resume P1 #7, #1604 (dev Linux `test` green now): re-run its failed jobs on `2e0f92672`, require the required checks green, merge `--match-head-commit`, resolve `gate0:driver-crash-notices-invisible`. Then P1 #8 if the lanes admit it. D-FLEET-13/14/15 remain OPEN.

## 26 — 2026-10-08 — gate0 self-notice read LANDED: #1604 `59c3e6a55`, re-judged PASS 96 on the merged head; ticket resolved; record #1636 landed [HARNESS]

**Pick**: P1 #7 `gate0:driver-crash-notices-invisible`, iteration 25's named resume. No `blocking=all` ticket open (43 open; `ailang mission ticket open --json`). Its predicate was re-measured as a command, not transcribed: dev required `test` green at `0ceb1db01`.

**Preflight**: armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing CLEAN; `mission_directives.sh --issue 1584 --since 2026-10-06T19:02:43Z` → 0 directives of 10 comments. Not a sweep week (iteration 19 swept), and no rotation: #1584 was created after Monday 07:00 local. Running skill: `SKILL.md` and all 12 resources `cmp`-identical to origin/dev at the resolved symlink. Pin == origin/dev `0ceb1db01`.

**Observe**: record PR #1636 (iterations 22–25) open and UNSTABLE: required checks green, Windows red inherited. Dev check set at `0ceb1db01`: 17 checks; NOT-GREEN = Sonar, `test-windows`, `Build windows-latest` (5 tests), and `Build macos-latest` cancelled. `mission-lane-check.sh fleet`: READY, every role has a usable rung (ollama rungs skipped as over ration; openrouter $0.10 of $2.33).

**Did**:
- Landed #1636 → `752ee765a` (closing-keyword scan empty, `--match-head-commit 3a94d3df8`), and closed superseded #1611 and #1612 with a comment.
- #1604: its `test` red on `2e0f92672` = inherited `TestValidateModulePath_SingleFileInsidePackage` (job 112275244977; 132,286 B log; one `--- FAIL`). 46 dev commits since its base `e68a264fb`. The only overlap with PR files is `make/test.mk`, where each side added a different line; `git merge-tree` clean. `gh pr update-branch` → `f4de90745`.
- Re-judged, because the bytes under review changed (a merge). Evaluator below. PASS 96, 0 blocking. Report banked at `design_docs/fleet-mission-evidence/iteration26/judge-report.md`.
- Controller reproduction, first-party, outside the sandbox, in the judge's worktree: suite `108 passed, 0 failed` rc 0. Mutant `: emit_for "$PREV_ISSUE"` → `103 passed, 5 failed` rc 1. Restore `cmp`-identical to `HEAD`; worktree porcelain = the report only.
- PR CI on `f4de90745`: `test` SUCCESS (the job that was red), `lint`, `docs-gate`, `UI build gate`, `launchd drivers (bash 3.2)`, govulncheck, CodeQL green. `test-windows` failed tests `diff`-identical to dev `0ceb1db01`'s (5; logs 527,124 B vs 528,078 B). Squash-merged `--match-head-commit f4de90745` → `59c3e6a55`.
- Resolved `gate0:driver-crash-notices-invisible` (`--sha 59c3e6a55`, "replied to v1"). Commented the verdict on #1160 via `--body-file` (comments 1 → 2, verified), then closed it.

**Gate 3b**: run 37724543642 on `59c3e6a55` (full SHA, via `mission-base.sh record gate3b`; drift from Gate 1 = #1636 + #1604, this iteration's own merges). `test` success, `lint` success, `build` success, `launchd drivers (bash 3.2)` success, govulncheck and float determinism success. `test-windows` failures `diff`-identical to the parent's 5 (528,733 B log). Sonar failure, also on parent `0ceb1db01`. Required set green → LANDED.

**Done-gate**: surface ✓ (`scripts/mission_gate0_self_notices.sh`, which every mission's Gate 0 runs via `MISSION_DRIVER_ROOT`, and both `gate-0-preflight.md` copies). Reach ✓ on origin/dev. **Running-skill reach PENDING**: `~/.claude/skills/mission-control` resolves to `~/dev/sunholo-data/ailang` (branch `dev`, `0 36` ahead/behind), where `grep -c "6a. SECOND, NO-AUTHORITY READ"` = 0, against 1 on origin. So no loop reads step 6a until that checkout is fast-forwarded (D-FLEET-15; not reconciled here, because Gate 1 reserves the standing authorisation to Mark). `make test-launchd-drivers` ✓ (CI leg on the merge SHA; the judge's in-sandbox rc 2 is the `mktemp` denial in the untouched `test_mission_lane_check.sh`). 0 driver bytes, so no dry-run. Nothing reloaded.

**Progress**: 1 ticket resolved (filed 2026-09-26 → resolved 2026-10-08, ≈12 d). Clause 1 UNMEASURED; clause 2 UNMET; clause 3 MET (42 open); clause 4 prior evidence only; clause 5 preserved.

**Routing evidence**: base-gate1=0ceb1db01df661c3608f0c3520a6eb28bcd6bab2@2026-10-08T03:20:14Z; base-gate3b=59c3e6a55bc58bb7202c0e9741fe46fc52547c61@2026-10-08T03:49:39Z; base=59c3e6a55bc58bb7202c0e9741fe46fc52547c61@2026-10-08T04:17:41Z (gate4). `MISSION_ROUTING_NOTE`: empty (as configured).
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer / Planner / Executor**: NOT SPAWNED. This was a resume of work iteration 22 had already planned, built (executor `claude:claude-sonnet-5-5`) and judged. The operator's standing request to use the Agent tool for roles applies where a role runs; the evaluator's lane is a `recipe`, so the Agent tool was not its specified path.
- **Evaluator**: resolver `agent-tool sonnet declared:alias-pin`. `sonnet` is the executor's family (`claude-sonnet-5-5`), so the generator ≠ judge guard re-routed to the first declared fallback `pi:openrouter/minimax/minimax-m3`. Probe rc 0 (`ok`); openrouter within ration. Ran via `scripts/mission_pi_run.sh --max-seconds 3600` in `fleet-iter26-evaluator` (detached at `f4de90745`). Handshake acked (4× `"acked":true`). Verdict `ok` rc 0, 1190 s, 93 tool executions, 94 turns. 4,912,033 tok (407,371 in / 41,686 out / 4,462,976 cache-read), $0.4400 (provider-reported `usage.cost`). Judge ≠ generator: minimax vs sonnet-5-5 (executor) and opus (controller).
- **Metered**: $0.44. **Ration**: ollama over; codex 3.0% of 7.3%; Anthropic 5h 9%, week 51% of 58.7%; openrouter $0.10 of $2.33 before the judge.

**Ruled out**:
- Re-running the failed jobs on the old head `2e0f92672`, as iteration 22's predicate wrote. Its base predates dev's `test` fix, so it would have re-run the inherited red; updating the branch was the cheaper, correct form.
- Merging on iteration 22's PASS 93 alone: the merged bytes are new, and the operator requires a judge each landing.
- Fast-forwarding the main checkout to make step 6a live: that is D-FLEET-15, still OPEN.
- Fixing the Windows five or Sonar: V1's lane, outside Authority.

**Retro lane**: no skill edit (no gap reached ≥2 frictions this fire). Observations, one each:
- (1) A resume predicate written as "re-run failed jobs on head X" goes stale when dev moves under an unmerged PR. The robust form is "update the branch, then require green on the new head".
- (2) The landed rule text is not live for any loop until the shared main checkout syncs. This is the first measured instance of a fleet fix landing on origin and not reaching the running skill; it is evidence for D-FLEET-15, not a new ask.

**Next**: P1 #8 `quorum:zero-signal-guard-vacuous-with-controller-verdict` (ailang#651). Its PARKED-ON-LANE predicate held at this fire's Gate 3 (`mission-lane-check.sh fleet` READY; minimax judge lane ≠ executor), so re-measure it and take it. Then P1 #9 `driver:exit-path-notices-unbounded`. D-FLEET-13/14/15 remain OPEN.

## 27 — 2026-10-08 — main-checkout ff-only auto-sync LANDED: #1647 `15d47b5dc` (D-FLEET-15 = A), judged PASS 91; ticket resolved [HARNESS]

**Pick**: P1 #6 `skill-surface:main-checkout-not-synced-to-dev`. Mark's attended ruling D-FLEET-15 = A (2026-10-08) unparked it ("treat it as HD-2a-authorised; plan it as the next P1"), so it ranks above P1 #8. No `blocking=all` ticket was open (42 open).

**Preflight**: armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing CLEAN; `mission_directives.sh --issue 1584 --since 2026-10-06T19:02:43Z` → 0 directives of 11 comments. Ledger valid (15 rows, 0 OPEN). Self-notices (step 6a) rc 1: the newest crash is the 2026-10-04 rc=143 kill, which iteration 22 already attributed to iteration 18's Gate-5 stall. Died-mid-flight traces: no open fleet PR, and the old `fleet-i1x` worktrees are all landed iterations. Running skill: `SKILL.md` and all 12 resources `cmp`-identical to origin/dev at the resolved symlink. Not a sweep or rotation week (#1584 was created after Monday 07:00 local).

**Observe**: origin/dev `c92739681`. Required `test` RED on `TestModels_CloudHeadroomEqualised` (job 113225065559): the motoko GLM-5.3-Flash rows were capped at 32000 by `69ddedd29`, in modelreg. Parent `f34214657` was green. Fleet does not own this red (Gate 1), so it went to `mission-v1` and `mission-motoko` with the repro. `382a8445d` (#1645) fixed it 25 minutes later. `test-windows` red as before (V1's lane).

**Did**:
- The planner (codex) wrote `design_docs/planned/sprint-plan-main-checkout-ff-sync.md` and the sprint JSON (`64a1bcab5`). Its premises section quotes the driver's file:line for every seam.
- The executor (codex) delivered M1 `d852c8969`: `tools/launchd/lib/skill-sync.sh` (`mc_skill_sync apply|report`) and `tools/launchd/test_skill_sync.sh`.
  - The helper uses bounded git calls through `_pin_bounded`, with a 5 s cap per call.
  - It merges only when the checkout is on `dev`, 0 ahead, has no op markers and no `index.lock`, and no dirty path intersects the incoming range (NUL-safe, both rename sides, directory prefixes).
  - It re-checks just before the one `merge --ff-only` and verifies HEAD and the count afterwards.
  - It always returns 0.
- The executor delivered M2 `4fe2da1f8`: a 10-line driver seam after the kill switch (:1625) and pidfile yield (:1908). Dry-run calls `report` and appends `skill-sync=` to the DRY RUN line; a real fire calls `apply` once before the boot stagger. Also: a `make/test.mk` line and a changelog fragment.
- The controller rebuilt both commits from the executor's `.snap/M1` and `.snap/M2`. `shasum -c` matched its final tree, and the suite was green at both boundaries.
- PR #1647. Every required check green on `4fe2da1f8`; the run included #1645, so `test` was green. Squash-merged `--match-head-commit` → `15d47b5dc`. Ticket resolved, with a reply to the attended filer.

**Gate 3b**: CI on `15d47b5dc` (full SHA; `mission-base.sh record gate3b` → `c114a1299`, i.e. dev moved past it with others' merges): `test`, `lint`, `build`, `launchd drivers (bash 3.2)` and govulncheck all success, out of 17 checks. NOT-GREEN = `test-windows` and `Build windows-latest` (inherited, as on parent `c92739681` and the PR head) plus a fail-fast `Build macos-latest` cancel.

**Done-gate**:
- Surface ✓.
- Reach ✓ on origin/dev.
- Dry-run ✓ healthy (`lanes=ok | pin=pinned | skill-sync=synced:4`) and ✓ degraded (`MISSION_EXECUTOR_MODEL=codex:bogus` → `lanes=DEGRADED(1)`). Both ran with `AILANG_DRIVER_PINNED=4fe2da1f8`, `MISSION_PROFILE=fleet`, `HOME=/tmp/fleet_it27_dryhome`: real config symlinked in, `.ailang/state` empty. That is how a fleet fire dry-runs its own profile past its own overlap guard. World and stapledon were mid-fire, and v1, docs and motoko were disabled, so no sibling was available.
- The real main checkout's HEAD and `.git/index` sha were identical before and after both dry-runs and both suite runs.
- `make test-launchd-drivers` ✓ (rc 0 unpiped; also the CI leg).
- Nothing reloaded ✓.

**Progress**: 1 ticket resolved (filed 2026-09-29 → 2026-10-08, ≈9 d, about 2 d of it parked on D-FLEET-15). Clause 1 UNMEASURED; clause 2 UNMET; clause 3 MET (41 open); clause 4 prior evidence only; clause 5 preserved.

**Routing evidence**: base-gate1=c92739681f50d33cf575a4e3a81b76aef360ea71@2026-10-08T08:55:31Z; base-gate3b=c114a1299bde5bbbad6d5f7c972afa3ecb6401b0@2026-10-08T11:01:39Z (drift: #1645, #1639 and this iteration's #1647). `MISSION_ROUTING_NOTE`: empty (as configured).
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer**: not spawned. Resolver said `recipe claude:claude-opus-5-5 declared:provider-pin`, but there was no doc to write: one helper plus a seam inside an attended ruling's exact scope (the same basis as iteration 25).
- **Planner**: resolver `agent-tool opus fail-closed:no-doc`, while `MISSION_PLANNER_MODEL=codex:gpt-6.1-sol`. Per role-spawn-routing rule (a), the pin was followed under the codex recipe (probe rc 0), in a detached worktree `fleet-iter27-planner`. (93,606 tok)
- **Executor**: `codex:gpt-6.1-sol` (`recipe … declared:provider-pin`) in `fleet-iter27-exec`, rc 0 in 17 min. (126,669 tok)
- **Evaluator**: `agent-tool sonnet declared:alias-pin` → Agent tool `model=sonnet`, foreground, own detached worktree at `4fe2da1f8`. judge-independence: cross-vendor (Anthropic judge, OpenAI generator). PASS 91. Report banked at `design_docs/fleet-mission-evidence/iteration27/judge-report.md`. (57,043 tok)
- The operator's standing request to use the Agent tool was honoured for the one role whose resolved path is the Agent tool (the evaluator). Planner and executor resolve to `recipe codex:…`, a path the routing table says is NOT the Agent tool; the spawn-pin hook would deny an alias there. No role failed to spawn.
- **Metered**: $0 this fire (codex and Anthropic subscription buckets; no openrouter or pi role).

**Ruled out**:
- Opus planner via the Agent tool, the resolver's literal answer: the role carries a provider pin, so the hook would deny it (rule (a)/(c)).
- Waiting for an idle sibling to dry-run: none was idle, and the synthetic-HOME dry-run of the fleet's own profile exercises the same patched driver.
- Fixing `TestModels_CloudHeadroomEqualised`: modelreg is outside fleet scope; it was handed off, and the owner fixed it in #1645.
- Raising the 5 s merge cap now: measured 0.66 s for an 804-file ff and 0.05 s for status on the real tree. Kept as a judge follow-up.

**Retro lane**: process fix, charter Guardrails: the done-gate now names the synthetic-HOME recipe for when no sibling is idle. Frictions: iterations 9 and 15 recorded "the fleet cannot dry-run itself", and this iteration measured a way around it. No skill edit. Observations, one each:
- (1) `tools/launchd/mission-lane-check.sh fleet` did not return within 10 min and was killed by the harness cap. The resolver lines in the same command had already printed, so nothing was lost; this is the first measured hang. Not ticketed: fleet does not self-source.
- (2) The controller's own zsh loops twice failed to word-split `$g`/`$FILES`, producing rc 127. These were instrument failures that were re-run under `/bin/bash`, not verdicts.
- (3) Judge follow-ups for a later ticket: an arm pinning `GIT_OPTIONAL_LOCKS=0`; the dry-run field prints `synced:N` for a would-sync verdict.

**Next**: P1 #8 `quorum:zero-signal-guard-vacuous-with-controller-verdict` (ailang#651); re-measure its premise at HEAD. Then P1 #9 `driver:exit-path-notices-unbounded`. Also check the first post-merge fire's driver log for `skill-sync=synced:` (running-skill reach).

## 28 — 2026-10-08 — P1 #8 candidate designed; independent evaluator unavailable → PARKED-ON-LANE, no implementation [HARNESS]

**Pick**: P1 #8 `quorum:zero-signal-guard-vacuous-with-controller-verdict`, charter head after iteration 27. 42 open tickets, none `blocking=all`. Index checked: iteration 23 capacity park, no prior implementation. Ambient docs sprint is unrelated.

**Preflight**: armed; gh `sunholo-voight-kampff`; billing CLEAN. 0 directives on #1584 since 2026-10-08T11:03:31Z. Ledger valid, 15 RESOLVED, 0 OPEN. No rotation/sweep due (#1584 created 2026-10-05T12:32:44Z, 12 comments). Main checkout's seven dirty paths untouched. Self-notices rc1 identified the October 4 rc143 death already credited by iteration 22. Died-mid-flight PR/worktree/uncommitted-state traces found no unmerged fleet implementation for this item; coordinator/product PRs left alone.

**Observe**: origin/dev `3e032cf4f13682a455129274d99097842c56e5ae`; CI run 37810475758 success, all 21 SHA-addressed checks success. All 13 mission-control files match origin in both resolved symlink and pin copies. Shared main checkout already at current origin; no driver-sync inference from transcript quotes.

**Did**: native pinned designer created `design_docs/planned/m-quorum-external-zero-signal-guard.md`, explicitly candidate only. Controller independently read source: reviewer increment at quorum.go:156, controller increment at :165, guard at :176. Candidate preserves vetoes, partial-present degradation, seating and Tier-2 blocking. Designer baseline `go test ./internal/mission/quorum` passed (0.767s); inherited baseline evidence, not new regression or judge verdict. No production/test implementation changed.

**Gate 3 / capacity**: operator requires native Agent roles. Designer succeeded as gpt-6.1-sol. Preferred evaluator native `sonnet` returned `Unknown model sonnet`; last declared fallback native `opus` returned `Unknown model opus`. Tool advertises OpenAI models only. No undeclared/same-model substitution. Existing probes outside a role sandbox, using fresh scratch CLI and `tools/launchd/lib/lane-probe.sh`: `_mc_probe_pi openrouter/minimax/minimax-m3` rc75; `_mc_probe claude-sonnet-4-6` rc75; `_mc_probe claude-opus-5-5` rc75. Admission refused before inference: `anthropic openrouter` blocked; OpenRouter $13.16/day vs $2.33 ration. No fallback judge admitted. Quorum/approval pending; planner and executor not dispatched through unmet gates.

**Bookkeeping verification**: ledger valid (15 rows); fresh-PATH canonical-heading/repo-local mission tests rc0 (0.354s); diff whitespace clean; last20 index 16/20 HARNESS; rotated iteration8 retained beside iteration7 control; full iteration25 STATUS archived; queue and decision markers retained. Pin clean after restoring only this invocation added notice.

**Gate 3b / done-gate**: ticket not LANDED: no implementation pushed, no judge verdict, no full launchd suite/dry-run/reload/ticket resolution. Bookkeeping checks are separate and cannot accept the candidate.

**Progress**: clauses 1 unmeasured, 2 unmet, 3 met (42 open), 4 prior evidence only, 5 preserved; 0 tickets resolved, goal unmoved. Last three ticket landings (25/26/27) drained clause-2 backlog; no regroom. Last-20 fleet harness share reported in digest, expected under inverted admissibility.

**Routing evidence**: base-gate1=3e032cf4f13682a455129274d99097842c56e5ae@2026-10-08T17:30:24Z; base=3e032cf4f13682a455129274d99097842c56e5ae@2026-10-08T17:33:42Z (gate4). Controller `codex:gpt-6.1-sol` (tok: not reported). Designer native `gpt-6.1-sol`, configured `codex:gpt-6.1-sol`, fresh context (tok: not reported): user explicitly requested native Agent, overriding provider-recipe transport. Planner/executor NOT SPAWNED: approval prerequisites unmet, not spawn failure (tok: not reported). Evaluator native sonnet/opus failed as above (tok: not reported); recipe fallback probes rc75, no verdict or score. Scratch CLI built with ldflags at exact base; shared PATH CLI stale `c92739681`, not overwritten. Metered role cost $0. Driver routing note is included verbatim in the two-channel digest.

**Ruled out**: controller acceptance; same-model fresh-context judge without authority; dormant Astra/retired OpenAI models outside D-FLEET-12; spending known over-ration buckets; planner/executor before quorum/approval; ticket resolution on source confirmation; repurposing ambient docs sprint; capacity filed as needs-human-review.

**Retro lane**: backlog observation, no policy/skill change. Native alias refusal recurs (iterations 11/24/28); World tickets corroborate unsupported Sonnet. Routing-policy signal, not permission to widen the chain. The initial STATUS matcher included the rotation heading and aborted before writing. The first rotate-log invocation followed exported MISSION_WORKDIR to the pin and added only a duplicate archive notice; those exact added bytes were restored (pin clean), then worktree overrides were used. Existing STATUS stamps have paragraph bodies, so rotation used bounded structural sections and byte-preserved the complete iteration25 block in the archive.

**Next**: re-probe declared evaluator chain; rc0 on a judge distinct from gpt-6.1-sol enables fresh design quorum with `--author codex:gpt-6.1-sol`, then approved-plan/execute/evaluate. P1 #9 exit notices follows; Phase 3a spike after P1. Human decisions: none.

## 29 — 2026-10-09 — native evaluator capability confirmed absent → PARKED-ON-LANE (external native capability); no runtime fix [HARNESS]

**Pick**: `agent-tool:sonnet-unavailable`, the only `blocking=all` ticket, ahead of P1 #8. 47 open signatures; 8 occurrence records. The latest occurrence is `blocking=none` with a successful workaround; the aggregate retains earlier blocking occurrences. Eight records do not establish eight terminal slot losses.

**Preflight**: armed; gh `sunholo-voight-kampff`; billing CLEAN; 0 directives of 14 comments since 2026-10-08T11:03:31Z. Decision ledger 15 RESOLVED, 0 OPEN. Full lane-check returned READY with every role ok. Self-notices rc1 identified the October4 rc143 death already credited by iteration22; no new orphan implementation. Died-mid-flight PR/worktree/dirty-state traces recovered the unmerged iteration28 record, not implementation.

**Observe**: origin/dev `48f4b5ef929532cc6bf89d32034435ec616f80e7`; CI workflow success, 17 checks with only `ailang-core-dev` deployment failure (outside fleet scope), not all-green. SKILL.md and all 12 resources match origin in both pin and resolved symlink. Worktree starts from this base plus cherry-picked iteration28 record `0932ac757` from clean PR #1662 head `4f173d9206d4d3f635f315ec76c594b6f609cdb3`. The original PR/worktree remain unchanged; inherited iteration28 is preserved verbatim, including its explicitly unreviewed candidate.

**Did**: native Sol designer and planner supplied read-only assessment. The repository cannot add missing models or tools to the hosting runtime. Native evaluator request `minimax/minimax-m3` was rejected `Unknown model`; the host advertised only `gpt-6.1-sol`, `gpt-6-astra`, `gpt-6-sol`, `gpt-6-luna`, and `gpt-5.6-sol`. No runtime, policy, skill or product change is justified. Ticket remains OPEN/PARKED-ON-LANE (external native capability).

**Gate 3 / capacity**: structural native capability failure, distinguished from exhausted quota. Current role transport recovered through the admitted recipe; the open host-capability ticket park does not block this fire’s independent judge or bookkeeping completion. User authorizes documented fallback after role spawn failure. Existing gate3 lines243–254 also permit fresh-context Sol fallback when no cross-vendor quota, FLAGGED; no new human ruling needed. Declared Minimax probe `_mc_probe_pi openrouter/minimax/minimax-m3` rc0 this fire. Independent evaluator uses documented pi runner cross-vendor fallback after native rejection; independent Minimax bookkeeping review PASS95/100, 0 blocking; report at `design_docs/fleet-mission-evidence/iteration29/judge-report.md` (verbatim), controller corrections in adjacent `controller-addendum.md`. No verdict or score fabricated.

**Bookkeeping verification**: controller fresh-PATH `go test ./internal/mission -run 'Canonical|RepoLocal' -count=1` rc0 (0.353s), ledger valid15rows, whitespace clean, and iteration28 body byte-identical. Independent Minimax bookkeeping review PASS95/100, 0 blocking; report at `design_docs/fleet-mission-evidence/iteration29/judge-report.md` (verbatim), controller corrections in adjacent `controller-addendum.md`. STATUS rotation preserves complete iteration26 in the archive; iteration27/28 retained live. Queue and decision ledger retained. Last20 index share 16/20 HARNESS, expected under fleet inverted admissibility, not a product-loop cap breach. Last three ticket landings25/26/27 moved clause2.

**Gate 3b / done-gate**: not LANDED; no runtime diff, implementation, product landing, ticket resolution, reload, suite or dry-run claimed. Record review is separate from accepting P1 #8's candidate.

**Progress**: clause1 UNMEASURED; clause2 UNMET; clause3 MET (47 open); clause4 prior evidence only; clause5 preserved (no fix shipped). 0 tickets resolved; goal unmoved.

**Routing evidence**: base-gate1=48f4b5ef929532cc6bf89d32034435ec616f80e7@2026-10-09T05:44:26Z; base=48f4b5ef929532cc6bf89d32034435ec616f80e7@2026-10-09T05:52:47Z (gate4). Controller native `gpt-6.1-sol` (tok: not reported); designer native `gpt-6.1-sol` (tok: not reported); planner native `gpt-6.1-sol` (tok: not reported); executor native `gpt-6.1-sol`, resolver `recipe codex:gpt-6.1-sol declared:provider-pin` (tok: not reported). User overrides transport only; declared Sol model pin retained. Evaluator native `minimax/minimax-m3` rejected (tok: not reported), then declared `pi:openrouter/minimax/minimax-m3` (2,547,205 provider tokens incl. cache; evaluator metered $0.206700). Known evaluator metered=$0.206700; model probes cost not reported separately; subscription role costs do not measure quota. Configured routing unchanged.

**Resolver evidence**: designer `recipe codex:gpt-6.1-sol declared:provider-pin`; planner `recipe codex:gpt-6.1-sol anthropic-fallback:fail-closed:no-doc`; executor `recipe codex:gpt-6.1-sol declared:provider-pin`; evaluator `recipe pi:openrouter/minimax/minimax-m3 declared:provider-pin`. Designer/planner/executor native transport is the operator override; evaluator native rejection is recorded above and actual recipe transport is recovered. `MISSION_ROUTING_NOTE`: controller pi:ollama/glm-5.3:cloud → codex:gpt-6.1-sol (probe ok); lanes degraded: ; `designer`: **anthropic** lane `claude-opus-5-5` unusable (probe rc=`75` — over daily ration) → handed to `codex:gpt-6.1-sol`; `evaluator`: **anthropic** lane `sonnet` unusable (probe rc=`75` — over daily ration) → handed to `pi:openrouter/minimax/minimax-m3`

**Ruled out**: treating native host model absence as quota; adding models/tools from repository code; undeclared native model substitution; controller acceptance; closing the aggregate ticket on one successful workaround; counting occurrences as terminal losses; adopting or approving iteration28's design by banking its orphan record; self-sourced fleet ticket or policy repair.

**Retro lane**: observation only, no mandatory skill edit or routing-policy change. The pi judge's live read-only ticket query stalled for >2min; controller terminated only that verified child PID35584, so the judge used the canonical ticket snapshot. Runner ended ok rc0 in448s,86tools, served model confirmed `openrouter/minimax/minimax-m3`. Its report contains incorrect SHA-equivalence/banner-attribution sentences and a mental-only drill; controller re-measured and documented those limitations separately without rewriting the verdict. Iteration28 conflated native transport capability with recipe quota capacity; this fire records them separately and uses the existing authorized fallback. No new decision needed; ledger unchanged.

**Next**: resume selected ticket only when native capability is proven on the affected controller; workaround remains existing declared CLI recipes. P1 #8 candidate next, then P1 #9 exit notices. Candidate requires independent review, quorum/approval and approved-plan execution gates; no approval claimed. Human decisions: none.

## 30 — 2026-10-09 — ORPHAN (no record written): P1 #8 quorum rounds 1–2 BLOCKED, revisions 2–3 under the carve-out; slot PAUSED-NO-CAPACITY at gate-2 [HARNESS]

Credited by iteration 32 from died-mid-flight traces. The slot ran 2026-10-09T12:41Z on the `pi:ollama/glm-5.3:cloud` controller and ended `PAUSED-NO-CAPACITY at=gate-2` after 2266 s. It ran quorum round 1 (`.ailang/state/mission-quorum/m-quorum-external-zero-signal-guard-2026-10-09T12-15-52Z.json`: BLOCKED; oc-glm-5-3 and oc-kimi-k3 rejected, gemini-3-1-pro passed, claude-sonnet-5 absent for quota). It wrote revision 2, then ran round 2 (`…T12-30-09Z.json`: BLOCKED; kimi and gemini rejected, glm passed, sonnet absent for quota). Revision 3 applied the reviewers' verbatim `proposed_fix` text under the narrow-refinement carve-out: both blocking rejects carried a concrete fix, and neither disputed the direction. The branches `fleet/i30-quorum-guard-revision` (`21c8073b1`) and `fleet/i30-quorum-signal-plan` were never pushed.

**Next**: re-apply revisions 2–3 onto a fresh base, then plan.

## 31 — 2026-10-12 — ORPHAN (no record written): P1 #8 revisions replayed onto `0a2202ad6`, plan left untracked; slot PAUSED-NO-CAPACITY at gate-3 [HARNESS]

Credited by iteration 32 from died-mid-flight traces. The slot ran 2026-10-12T01:08Z on the glm controller and ended `PAUSED-NO-CAPACITY at=gate-3` after 1325 s. It replayed revisions 2–3 as `c4c0f5515` and `8d0d3de0e` on `fleet/i31-quorum-signal-plan` (not pushed). It also left an untracked `design_docs/planned/sprint-plan-quorum-zero-signal-guard.md` in `fleet-i31-planner`, and an unpushed commit `5a0035113` on `fleet/i29-record-land` (archive retirement of iteration 9). Iteration 32 adopted the plan through its planner and did not adopt `5a0035113`.

**Next**: verify the plan, execute, evaluate.

## 32 — 2026-10-12 — P1 #8 quorum zero-signal guard BUILT and judged PASS 95 (sonnet), PR #1767; orphan iterations 30/31 credited [HARNESS]

**Orphans**: iterations 30 and 31 (entries above) carried this item with no record; credited here once.

**Pick**: charter P1 #8 `quorum:zero-signal-guard-vacuous-with-controller-verdict` (ailang#651), resumed from iteration 28's PARKED-ON-LANE. The resume predicate is an admitted judge distinct from the generator. This fire: `MISSION_ANTHROPIC_AVAILABLE=1`, and the evaluator resolved to `agent-tool sonnet declared:alias-pin`, which spawned. The only `blocking=all` ticket, `agent-tool:sonnet-unavailable` (8 slots, world/stapledon), stays parked as iteration 29 assessed it: the affected controllers' native hosts lack the model. A Claude controller spawning sonnet does not prove capability on those hosts.

**Preflight**: armed; gh `sunholo-voight-kampff`; billing CLEAN. 0 directives on #1584 since 2026-10-11T18:46:20Z (27 comments, all self). Self-notices rc 1: the 2026-10-04 rc=143 kill, already credited by iteration 22. All 13 skill files `cmp`-identical to origin in both the resolved symlink target and the pin. dev CI `success` on `0a2202ad6`, 21 checks, none not-green.

**Did**:
- **Planner** (opus, Agent tool, foreground) adopted the orphan plan with 10 corrections, committed `0c118d46c`. Two of them would otherwise have produced vacuous tests: (1) the Tier-2 regression must call `SynthesizeWithTier2` directly, because `RunQuorumWithEscalation` never escalates on an all-absent Tier 1 (`escalate.go:76`); (2) the accounting assertion needs non-zero cost/tokens on absent outcomes. The seated test was also moved to `poolRunner`/`seat`, which record fallback calls.
- **Executor** (`claude:claude-haiku-5-5` via `claude-sub`, probe rc 0, rc 0 in about 8 min) produced `7d34e97ae` (fix + matrix/seated/Tier-2 tests + changelog fragment) and `736d58142` (implementation report + sprint JSON). It rejected nothing and ran a pre-fix red run: the controller-pass/all-absent, empty-slice, seated and Tier-2 cases all failed before the fix. Production diff: `presentCount` → `externalPresentCount`, the controller increment removed, guard text unchanged, contract comments updated.
- **Evaluator** (sonnet, Agent tool, foreground, its own detached worktree `fleet-i32-judge` at `736d58142`): **PASS 95**, 0 blocking. Every M1 AC PASS. 6 of 6 mutants killed (MU1 controller increment, guard disabled, early return, `len(outcomes)`, absent counted, absent cost dropped). Report banked at `design_docs/fleet-mission-evidence/iteration32/judge-report.md` (`c55bc9a39`).
- **Controller check**: re-ran MU1 independently in the judge worktree. `TestSynthesize_ExternalSignalMatrix`, `TestRunSeatedQuorum_AllAbsentControllerPassBlocks` and `TestSynthesizeWithTier2_PassCannotRescueZeroSignalTier1` went RED. The restored file is `cmp`-identical, and the package is green.
- **Records**: pending record PR #1710 (iterations 28–29) had a red `test` on `7883f1c65`: `TestArtifactsRejectBoundaryViolations` in `internal/mission/iteration`, a TempDir `RemoveAll … .git/objects: directory not empty` cleanup race, on a docs-only diff. Re-ran the failed job → green; squash-merged `--match-head-commit 7883f1c65` → `6d0e5fd76`.

**Gate 3b**: PR #1767 head `c55bc9a39`: 26 checks, all complete, 0 failures (required `test`, `lint`, `build`, `docs-gate`, `UI build gate` green; `test-windows`/`Build windows-latest` green this time). Squash-merged `--match-head-commit c55bc9a39` → `a5f9af982`. Dev CI on `a5f9af982`: `test`, `lint`, `launchd drivers (bash 3.2)`, govulncheck, `test-windows` and all `Build` legs success (21 checks; release `build` still running at record time).

**Done-gate**:
- Surface ✓: `internal/mission/quorum/quorum.go`, the one function every `design-quorum` run synthesizes through.
- Reach ✓ on origin/dev, so every pinned mission builds it. **Caveat:** loops call the `ailang` binary on PATH, which only picks up the fix on its next rebuild or release. `~/go/bin/ailang` was not rebuilt by this loop, because it is shared with live evals.
- Dry-run: not applicable, because the driver and scripts were not changed (iteration 26's precedent).
- `make test-launchd-drivers` at `a5f9af982`: rc 0 unpiped, 558 s.
- Nothing reloaded ✓.
- Ticket resolved with `--sha a5f9af982`: rc 0, replied to v1; ailang#651 closed with a verdict (comments 0→1).

**Progress**: 1 ticket resolved (filed 2026-09-26 → 2026-10-12, ≈16 d, most of it parked on lane capacity). Clause 1 UNMEASURED; clause 2 UNMET; clause 3 MET; clause 4: the six PAUSED-NO-CAPACITY fires on 2026-10-10/11 each spent 37–66 s and no controller tokens; clause 5 preserved (judged; required CI green on the PR head and on the merge).

**Routing evidence**: base-gate1=0a2202ad6c663ef0f922d6a4e27b3bf1fcc8cb65@2026-10-12T07:09:48Z; dev moved to `6d0e5fd76` (#1710) and `a5f9af982` (#1767) during the fire. `MISSION_ROUTING_NOTE`: "controller pi:ollama/glm-5.3:cloud → claude:claude-opus-5-5 (probe ok); planner and executor codex lane gpt-6.1-sol over daily ration (rc 75) → opus / claude:claude-haiku-5-5".
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer**: not spawned. Revision 3 was already complete (iteration 30, carve-out), and the design rotation was not advanced.
- **Planner**: resolver `agent-tool opus fail-closed:env-pin`, pin `opus` (alias, so there was no hook conflict). Spawned via the Agent tool `model=opus`, foreground (91,465 tok).
- **Executor**: resolver `recipe claude:claude-haiku-5-5 declared:provider-pin` (codex over ration). Ran via `claude-sub -p --model claude-haiku-5-5`, probe rc 0, in `~/.ailang-worktrees/fleet-i32` (tok: not reported by `claude -p` text output).
- **Evaluator**: resolver `agent-tool sonnet declared:alias-pin`. Spawned via the Agent tool `model=sonnet`, foreground, in its own detached worktree. Judge independence: different model from the generator (haiku) and the planner (opus), same vendor. PASS 95 (64,197 tok).
- The operator's standing request to use the Agent tool was honoured for every role whose resolved path is the Agent tool (planner, evaluator). The executor's resolved path is a provider recipe, which the routing table says is not the Agent tool. No role failed to spawn.
- **Metered**: $0 (Anthropic subscription only; no quorum round this fire).

**Ruled out**:
- Re-running quorum on revision 3. The carve-out routes straight to the planner, and iteration 30 recorded the verbatim fixes.
- Adopting iteration 31's unpushed `5a0035113` (an archive rewrite on `fleet/i29-record-land`). It was not part of #1710, it is unreviewed, and it is not this item.
- Taking the `blocking=all` ticket `agent-tool:sonnet-unavailable` ahead of #8. Its resume predicate is native capability on the affected (non-Claude) controllers, which this fire cannot prove.
- Rebuilding `~/go/bin/ailang` for reach. It is a shared live binary, so it is left for release or an attended quick-install.

**Retro lane**: **skill fix** (one edit, two recorded frictions: iterations 28 and 32). `gate-4-record.md` in both the `.claude` and `.agents` copies now says that `rotate-log` writes into the checkout named by `AILANG_MISSION_REGISTRY`. The driver sets that to the pin, so a record worktree must pass `AILANG_MISSION_REGISTRY="$PWD/missions"`. Measured this fire: two runs wrote to the pin, each adding a duplicate archive notice; those bytes were reverted with `git apply -R`, and the pin is clean. `MISSION_WORKDIR` alone did not redirect it. Observations, one each, not ticketed (the fleet does not self-source):
- (1) Every `rotate-log` run appends another "Older entries are ARCHIVED" notice. This is the third instance: iteration 28, iteration 31's `5a0035113`, and this fire, which collapsed the duplicate by hand.
- (2) The controller's zsh again failed to word-split a file list, so the sweep's positive control read 0. That was an instrument failure; the sweep was re-run under `/bin/bash`, as at iteration 27.
- (3) Two consecutive capacity-paused slots (30, 31) did real work and wrote no record. `PAUSED-NO-CAPACITY` at gate-2/3 strands work exactly like a crash, so the died-mid-flight traces are what recovered it.

**Next**: P1 #9 `driver:exit-path-notices-unbounded`, then the Phase 3a skill-resolution spike (directive "after P1, before P2"). Follow-ups from the judge, not ticketed: the report's pre-fix RED output is condensed, MU1 rc not captured, and the seated test alone does not kill the early-return mutant (the matrix does).

## 33 — 2026-10-12 — P1 #9 exit-path notices bounded and store-scoped, judged PASS 90 (sonnet), PR #1791 → `e7dd6c127` [HARNESS]

**Pick**: charter P1 #9 `driver:exit-path-notices-unbounded` (filed by v1 iteration 315, 2026-09-26). It is the queue head after iteration 32. 53 open tickets. The only `blocking=all` ticket, `agent-tool:sonnet-unavailable`, stays parked as iterations 29 and 32 assessed it. Index grep: no prior attempt on this item.

**Preflight**: armed; gh `sunholo-voight-kampff`; billing CLEAN. Directives: 0 on #1768 (3 comments, all self; the `mission-1768-last-seen` watermark was absent) and 0 on #1584 since 2026-10-11T18:46:20Z (28 comments, all self). Self-notices rc 0: no crash notices; control `852:10` ok, one of two watermarks readable. Ledger unchanged (15 RESOLVED, 0 OPEN). No rotation or sweep due: #1768 was created after this Monday's boundary, and the sweep ran at iteration 32.

**Observe**: origin/dev == pin == `06d524c59`. `SKILL.md` and all 12 resources are `cmp`-identical to origin in both the resolved symlink target and the pin. Dev CI on `06d524c59` was in progress at Gate 1 (16 checks, 8 pending, 0 failed).

**Reality check at HEAD**: three unbounded pairs remained, at `mission-control.sh:2038/2041` (refusal), `:2529/2532` (late-kill "work landed") and `:2550/2553` (rc-fail). The charter's line numbers had drifted. None of the three scoped the message store, so even when they did send, they wrote to the rig's local SQLite inbox. `tools/launchd/lib/*.sh` holds no sends.

**Did**:
- **Planner** (opus, Agent tool, foreground) wrote `sprint-plan-exit-path-notices-bounded.md` (`468604ba4`). It adds a one-shot helper, `_mc_exit_notice`, that is bounded per channel and always returns 0. The helper scopes the store with the same per-command `env AILANG_STORAGE_MESSAGING=gcp …` form `_mc_notify` uses, and the slot-verdict send gets the same scoping. The plan rejected `_mc_notify` for these sites because it uses one body for both channels, adds date-stamped titles (a text change), and its retry/spool could stretch the exit path to about 2 min. Test arms: E1–E3 for the hang path, E4–E9 for preservation, and source guards G1–G3. G1 checks every send is bounded, G2 that every send is store-scoped, G3 that the F8 literal is present.
- **Executor** (`claude:claude-haiku-5-5` via `claude-sub`; probe rc 0, run rc 0, about 30 min) committed `1fe1e3911` (fix, tests and changelog fragment) and `5c8ac0daf` (report plus RED/GREEN logs).
  - Before the fix, `test_driver_notify.sh` failed 11 arms (104/11). After it, the suite passed 115/0. `test_controller_capacity`, `test_mission_heartbeat` and the gate0 self-notices suite are green; `make test-launchd-drivers` rc 0; `check-changelog` rc 0.
  - It ran 9 mutation drills, each rc 1 and each caught by its targeted arm.
  - Its anchor tightening of E5 came from drill d2, which survived the original arm.
- **Evaluator** (sonnet, Agent tool, foreground, in its own detached worktree `fleet-i33-judge` at `5c8ac0daf`): **PASS 90**, 0 blocking.
  - Titles, bodies, exit codes and the F8 literal are byte-identical; the F8 control still fires.
  - `_mc_notice_suppressed` reads this send's own `MC_BOUNDED_OUT`. The helper is `set -u`-safe and bash-3.2-safe. Worst case with both channels hung is about 36 s per call.
  - Mutation drills: M1–M5 killed (bare ailang send, dropped store prefix, F8 body, bare gh call, late-kill title). M6 was UNMEASURED: the drill was cut off by the tool limit.
  - It reported `test_driver_notify.sh` at 113/2, with the same 2 arms red on pristine base.
- **Controller check**: re-ran the suite in the judge tree under `env -i`: rc 0, 115/0. The judge's 2 failures came from the controller session's exported `MISSION_ROUTING_NOTE`, not from the change. Recorded in `fleet-mission-evidence/iteration33/controller-addendum.md`, beside the verbatim `judge-report.md` (`986def325`).

**Gate 3b**: PR #1791 head `986def325`. 25 checks complete, 0 failures (`test`, `lint`, `build`, `launchd drivers (bash 3.2)`, `test-windows`, all Build legs, SonarCloud, CodeQL, govulncheck). The poll was SHA-pinned with a per-invocation log. Squash-merged `--match-head-commit 986def325` → `e7dd6c127`.

**Done-gate**:
- Surface ✓: `tools/launchd/mission-control.sh`, the driver every mission runs.
- Reach ✓: on origin/dev, so every pinned mission picks it up on its next fire. This is a driver change, so no binary rebuild is needed.
- Dry-run ✓, healthy and degraded, under the idle armed **stapledon** profile (`env -i`, `AILANG_DRIVER_PINNED=e7dd6c127…`, from a detached worktree at the merge). Both rc 0 and `DRY RUN ok`. The degraded arm used `MISSION_EXECUTOR_MODEL=pi:bogus/no-such-model`, which was handed to opus. No stapledon pidfile was written.
- `make test-launchd-drivers` at `e7dd6c127` under `env -i`: rc 0 unpiped, 590 s ✓.
- Nothing reloaded ✓.
- Ticket resolved with `--sha e7dd6c127`: rc 0, replied to v1; 53 → 52 open.

**Progress**: 1 ticket resolved; it was filed 2026-09-26 and resolved 2026-10-12, ≈16.5 d. Clause 1 UNMEASURED. Clause 2 UNMET. Clause 3 MET (53 open). Clause 4: prior evidence only. Clause 5 preserved: judged, with required CI green on the PR head.

**Routing evidence**: base-gate1=06d524c59ebd10d249ad8cae97b3d57231e9c862@2026-10-12T20:56:48Z; base=e7dd6c127b87b5568b9f1dfa74e9546cb47879c6@2026-10-12T22:48:40Z (gate4). Dev moved only by this PR's own merge. `MISSION_ROUTING_NOTE`: "controller pi:ollama/glm-5.3:cloud → claude:claude-opus-5-5 (probe ok); lanes degraded: ; `planner`: **codex** lane `gpt-6.1-sol` unusable (probe rc=`75` — over daily ration) → handed to `opus`; `executor`: **codex** lane `gpt-6.1-sol` unusable (probe rc=`75` — over daily ration) → handed to `claude:claude-haiku-5-5`".
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer**: not spawned. The ticket is mechanical, and the charter says to design only when the fix warrants it. The rotation was not advanced.
- **Planner**: resolver `agent-tool opus fail-closed:env-pin`, spawned via the Agent tool `model=opus`, foreground (127,638 tok). The first spawn was refused by the spawn-pin hook (`fail-closed:role-missing`) because the role token was not alone on line 1. It was re-spawned with `MISSION-ROLE: planner` as the whole first line.
- **Executor**: resolver `recipe claude:claude-haiku-5-5 declared:provider-pin`, run via `claude-sub -p --model claude-haiku-5-5 --permission-mode bypassPermissions` in `~/.ailang-worktrees/fleet-i33` (tok: not reported by `claude -p` text output).
- **Evaluator**: resolver `agent-tool sonnet declared:alias-pin`, spawned via the Agent tool `model=sonnet`, foreground, in its own worktree. PASS 90 (73,027 tok). Judge independence: a different model from the generator (haiku) and the planner (opus); same vendor.
- The operator's standing request to use the Agent tool was honoured for every role whose resolved path is the Agent tool (planner, evaluator). The executor's resolved path is a provider recipe. No role failed to spawn.
- **Metered**: $0 (Anthropic subscription only).

**Ruled out**:
- Reusing `_mc_notify` at these sites. It would change the titles and could triple the exit-path wall clock.
- Date-stamping the titles. That changes the text and is filed as a follow-up.
- Bounding `nightly-eval.sh` and the other rig scripts. They are outside this ticket, and the guardrail is ticket-driven only.
- Taking `agent-tool:sonnet-unavailable` ahead of #9. Its resume predicate is native capability on non-Claude controllers.

**Retro lane**: observations only; there is no second friction for a skill edit.
- (1) The spawn-pin hook requires `MISSION-ROLE: <role>` to be the ENTIRE first line. A trailing parenthetical is refused as `role-missing`, while the skill's wording ("as the FIRST line") reads as a prefix. Instance 1; it cost one refused spawn.
- (2) Agent-tool roles inherit the controller's exported `MISSION_*` environment. That makes `test_driver_notify.sh`'s routing-note arms fail inside any sub-agent, and a judge could read that as a regression; this judge correctly attributed it to base. Instance 1; there is a workaround (`env -i`). Not ticketed: the fleet does not self-source.
- (3) `rotate-log` added a duplicate "Older entries are ARCHIVED" notice again (instance 4, after iterations 28, 31 and 32), collapsed by hand. The `AILANG_MISSION_REGISTRY` override kept the pin clean, as the iteration-32 skill fix prescribes.
- (4) Planner follow-ups, not ticketed: `driver:exit-notice-titles-undated` (an undated title is refused as a duplicate after the first send) and `nightly-eval:unbounded-sends`.

**Next**: the Phase 3a skill-resolution spike. P1 is now exhausted, and the directive says "after P1, before P2"; it is measurement only. Then P2 #10 `skill:gate0-ledger-provenance-S`.

## 34 — 2026-10-13 — Phase 3a skill-resolution spike measured (P1–P3), judged PASS 88 (sonnet); Phase 3b mechanism parked as D-FLEET-16 [HARNESS]

**Pick**: Mark's standing directive "after P1, before P2", the Phase 3a skill-resolution spike (M-HARNESS-MISSION-LOOP, premises P1–P3). P1 was exhausted at iteration 33. Measurement only. 52 open tickets. The only `blocking=all` ticket, `agent-tool:sonnet-unavailable`, stays PARKED-ON-LANE as iterations 29, 32 and 33 assessed it. The index grep found no prior attempt, and the orphan traces were clean: no open fleet PR for the item, and no worktree or uncommitted work for it.

**Preflight**: armed; gh `sunholo-voight-kampff`; billing CLEAN. Directives: 0 on #1768 since `2026-10-13T00:00:23Z` (4 comments, all self). Self-notices rc 0: no crash notices, control `852:10` ok, 2 of 2 watermarks readable. Ledger: 15 RESOLVED, 0 OPEN at pick. No rotation or sweep was due: #1768 opened this week, and the sweep ran at iteration 32.

**Observe**: origin/dev == pin == `67228f1ca` at Gate 1. `SKILL.md` and all 12 resources are `cmp`-identical to origin at the resolved symlink target. Dev CI on `67228f1ca`: `CI` success, 17 checks, none failing.

**Did**:
- **Planner** (`codex:gpt-6.1-sol`, recipe; probe rc 0, run rc 0, ~12 min) wrote `sprint-plan-phase3a-skill-resolution-spike.md` (242 lines) and `sprint_M-HARNESS-PHASE3A-SPIKE.json` in a detached planner worktree. The protocol uses per-run random tokens, pre/post explicit positive and negative controls, an ownership ledger with sha-audited parents, a write fence, `claude-sub`, and three CLIs × named arms. Committed `9b80f9ba2`.
- **Executor** (`codex:gpt-6.1-sol`, recipe; rc 0, ~15 min) built `skill_resolution_spike.sh` (400 lines). Sandbox gates: `bash -n` rc 0; shellcheck rc 0; a test-root dry run with 207 arms and 16/16 parents unchanged; a cleanup-mutant drill giving SAFETY_FAILURE, rc 1.
- **Controller runs outside the sandbox**: a real `--dry-run --reps 1` (161 arms, parents sha-identical, rc 0), then live runs per CLI. Eleven instrument defects that only live runs could expose were repaired in place, then each run executed a frozen copy. The defects include: `CLAUDE_CONFIG_DIR=~/.claude` is not the unset default; the markdown-fenced JSON answer; `K:H` pointed at `~/.codex` while the mission `CODEX_HOME` is `~/.codex-missions`; pi's synthetic agent dir had no `models.json`; variadic `--add-dir` swallowed the prompt; the auth `init` event; pi's trailing `agent_settled`; codex reads `CLAUDE.md` per `AGENTS.md`; and the fence blocks codex's app-server. The full list is in `findings.md`. Two by-hand probes of pi `/skill:` invocation have a negative control.
- **Results**:
  - claude, run `1791874306` (51/51 controls): project discovered; user level wins every collision (pin, v1 and `--add-dir`); `--add-dir` loads skills; `--setting-sources ''` disables discovery; a fresh `CLAUDE_CONFIG_DIR` authenticates under the env token, and its `skills/` wins.
  - codex, run `1791874447`: discovers `.agents/skills` and `$CODEX_HOME/skills` (CODEX_HOME wins), never `.claude/skills`; a fresh `CODEX_HOME` returns 401.
  - pi, run `1791874311` + probe: no catalog discovery; `--skill` loads a skill reachable by `/skill:<name>`.
- **Evaluator** (sonnet, Agent tool, foreground, in its own detached worktree `fleet-i34-judge` at `3ec678757`): **PASS 88**, 0 blocking. It re-derived at least 8 verdicts from `tokens.tsv` and raw stdout, and replicated C1 (claude user-wins, 3/3), C2 (codex `.claude` ignored, `.agents` read) and C3 (pi `--skill` + `/skill:`) with its own method and negative controls.
  - Mutation drill on repair 8: the audit was too permissive (N2).
  - Non-blocking: N1 wording, N3 small-n scope, N4 the repetition run's summary, N5 decision wording.
- **Controller on the judge's findings**: confirmed N2 by code reading, then fixed it: a removed non-spike entry in a non-world parent is again SAFETY_FAILURE. Mutation drill: an injected foreign deletion gave SAFETY_FAILURE rc 1, and an injected foreign addition gave CONCURRENT_CHANGE rc 0. N1 and N3 wording adopted. N5: D-FLEET-16 is filed by this record, recommending A with default C (the no-change default is deliberate).
- **Evidence hygiene**: raw claude stream-json init events list personal addresses, so the raw streams stay on the rig at `~/.ailang/state/fleet-i34-spike-raw/`. Only distilled per-call artifacts are tracked. `make check-no-personal-email` rc 0. The judge report's one regex hit is the pattern it quoted.
- **Safety**: after all runs, `find` over every real parent (`~/.claude/skills`, `~/.agents/skills`, `~/.codex-missions/skills`, `~/.codex/skills`, the main checkout's `.claude/skills`, `/private/tmp`, pin `.claude`/`.agents`) found 0 `zz-fleet-spike-*` entries, with the positive control `mission-control` = 1.

**Gate 3b**: PR #1793, head `d112f7b26`. A SHA-pinned poll with a per-head log: 26 checks complete, 0 failures. `MERGEABLE`/`CLEAN`. Squash-merged with `--match-head-commit d112f7b26` → `3a972e34a`. The PR body and commit messages had no closing keyword (scan control fired). This line was written by the follow-up record PR.

**Done-gate**: not applicable as harness change. No driver, skill or Go code shipped; the only executable is an evidence script under `design_docs/`. The design doc's Phase 3a acceptance box is checked; Phase 3b is not started.

**Progress**: directive item done; 0 tickets resolved. Clause 1 UNMEASURED. Clause 2 UNMET. Clause 3 MET (52 open). Clause 4: prior evidence only. Clause 5 preserved. The goal moved only through the directive: Phase 3b (`harness-stable`) is unblocked pending D-FLEET-16.

**Routing evidence**: base-gate1=67228f1ca07697af904a01673b3e03601fe17655@2026-10-13T06:02:44Z; base=87f8952128c71aa84cd6cbc7adb92705c9e51d50@2026-10-13T07:29:36Z (gate4). Dev moved one unrelated commit (`87f895212`, openrouter host exclusions). `MISSION_ROUTING_NOTE`: empty. Over-ration: none.
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer**: not spawned; the design doc exists and the spike is measurement. The rotation was not advanced.
- **Planner**: resolver `recipe codex:gpt-6.1-sol declared:planner-lane-default-pin` (derive-planner-lane agrees), run via `codex exec --sandbox workspace-write` in `.planner-wt-iter34` (92,785 tok). The first probe from `/tmp` returned rc=1 (`Not inside a trusted directory`), my instrument; re-probed from the repo, rc 0.
- **Executor**: resolver `recipe codex:gpt-6.1-sol declared:provider-pin`, run in `fleet-i34` (201,377 tok).
- **Evaluator**: resolver `agent-tool sonnet declared:alias-pin`, spawned via the Agent tool `model=sonnet`, foreground, in its own worktree. PASS 88 (111,732 tok). Judge independence: cross-vendor against the generators (codex) and a different model from the controller (opus).
- The operator's standing request to use the Agent tool was honoured for the one role whose resolved path is the Agent tool (evaluator). The planner and executor resolved to provider recipes. No role failed to spawn.
- **Spike calls**: about 200 haiku calls (subscription), about 75 `gpt-6.1-sol` codex calls (ChatGPT bucket), about 60 `kimi-k3:cloud` pi calls (flat-rate). **Metered**: $0.

**Ruled out**:
- Reading pi's `ABSENT` as "`--skill` does not work". The hand probe shows it loads a skill reachable only by `/skill:`.
- Reading the codex capacity failures as discovery evidence.
- Trusting the in-sandbox executor gates for live behaviour; they were re-run outside.
- A full `--reps 3` run: at 161 calls with `--reps 1`, the judge's independent replication was the better use of quota.
- Committing raw streams (personal addresses).

**Retro lane**: two frictions this iteration, both instance 1, so no skill edit:
- (1) **Editing a bash script while instances of it run corrupts them**: bash reads incrementally. Three live runs were in flight when the scorer was patched; two finished coherently, and later runs used frozen copies. The general rule is to execute a per-run copy of any script you might edit mid-run.
- (2) **A grep for personal addresses over a judge report matches the pattern the judge quoted**: a self-describing artifact poisons a known-absent check. This is the "a control you record is a control you spend" class.
- The planner's protocol was rigorous, but its live-only assumptions (fence vs codex app-server, pi catalog) needed 11 repairs. A spike protocol should budget a controller "instrument shake-down" round before the counted runs.

**Next**: P2 #10 `skill:gate0-ledger-provenance-S`. D-FLEET-16 is with Mark.

## 35 — 2026-10-13 — P2 #10 Gate 0 ledger provenance instrument, judged PASS 92 (sonnet, round 2), PR #1815 → `130486515` [HARNESS]

**Pick**: charter P2 #10 `skill:gate0-ledger-provenance-S` (filed by stapledon iteration 1, 2026-09-28), the queue head after iteration 34. 53 open tickets at pick. The only `blocking=all` ticket, `agent-tool:sonnet-unavailable`, stays PARKED-ON-LANE as iterations 29 and 32–34 assessed it. Index grep: no prior attempt. Orphan traces: no open fleet PR, worktree or uncommitted work for the item.

**Preflight**: armed; gh `sunholo-voight-kampff`; billing CLEAN. Directives: 0 on #1768 since `2026-10-13T00:00:23Z` (5 comments, all self). Self-notices rc 0: 0 crash notices, control `852:10` ok, watermarks 2/2. Ledger: 15 RESOLVED, 1 OPEN (D-FLEET-16). No rotation or sweep due.

**Observe**: origin/dev == pin == `b283b2d44` at Gate 1. `SKILL.md` and all 12 resources are `cmp`-identical to origin at the resolved symlink target. Dev CI on `b283b2d44`: `CI` success. `Deploy Documentation` was RED (`docs-build`, `docs-gate`): first red at `45f483efd`, last green at `092c7adcb`. Cause: `docs/docs/reference/env-vars.md` fails MDX/acorn parsing after `50fa27357` regenerated it. The generator lives in `internal/config`, outside fleet authority, and V1 owns repo reds, so it was handed to V1 (Gate 5 message).

**Reality check**: reproduced at HEAD on this charter. `-S` returns the creation commit for D-FLEET-13/15/10/1, while the first `-G … RESOLVED` hit returns the flip. A second defect, not in the ticket: every ledger flip in this repo is a GitHub squash merge authored by the bot, so even the corrected command reads Mark's attended rulings as `fleet`. `scripts/mission_answer.sh`'s own header already concedes the author is not a real control.

**Did**:
- **Planner** (`codex:gpt-6.1-sol`, recipe; probe rc 0, run rc 0, ~11 min) wrote `sprint-plan-gate0-ledger-provenance.md` with 2 milestones, 13 arms and a mutation column. It corrected my measurement: D-FLEET-1 was created `RULED`, not `RESOLVED`, so it is not a reword case. It confirmed that `make/test.mk` enumerates suites by hand and that CI's macOS bash-3.2 job runs `test-launchd-drivers`.
- **Executor** run A (`codex:gpt-6.1-sol`): built M1. It was reaped mid-mutation-drill at the harness's **10-minute background default**, because I launched the 30-minute wrapper without `timeout`; codex died with it. M1 verified outside the sandbox (112/0; real-history verdicts as planned) and committed `f46815d65`.
- **Executor** run B (resumed with a directive): 14 remaining mutation drills, so 20 rows in all, every target arm red. Plus M2. The sandbox refused the `.agents` mirror write; the controller copied it byte-for-byte (`cmp` silent). Committed `248c2611b`.
- **Evaluator** round 1 (sonnet, Agent tool, foreground, own worktree): **PASS 88**, 0 blocking. It ran 9 of its own mutants, and 3 dead guards survived (N5).
  - **N1**: Human resolves, Human reopens, then fleet re-resolves; the result read `attended` from the first flip.
  - **N3**: rule 6(b)'s opening sentence overstated what the check can do.
- **Controller** reproduced N1 in a fixture (`verdict=attended`). The fix selects the **latest** non-RESOLVED→RESOLVED transition. New arm A14 covers it, and the A3 alias loop now expects the latest flip. The earliest-selection mutant turns A14 and A3 red; 115/0. N3 is adopted as one clause in 6(b). Committed `8350e1c6c`.
- **Evaluator** round 2 (resumed by `SendMessage`): **PASS 92**, 0 blocking. N1 is fixed. Its new mutant (drop the `! before resolved` guard) is now killed by A5, so it is no longer a dead guard. A fleet RULED→RESOLVED flap over an attended ruling reads `fleet`, a conservative false positive.
- Real history at the merge: D-FLEET-1/13/14/15 give `undetermined:squash-merge` (flips `e6990da9a`, `996c411a4`), D-FLEET-16 gives `never-resolved`, and an unknown ID gives rc 2.

**Gate 3b**: PR #1815, head `8350e1c6c`. SHA-pinned polls with per-head logs.
- Round 1: `Build macos-latest` failed on `TestRuntimeNestedStopUsesRemainingDeadline` in `internal/effects`; the other legs were fail-fast cancellations.
- After rerun: `test` failed on `TestWorkerCLIProcessExitLifecycle/async/eval/sigint` in `internal/runner`. Both are signal/timing tests in code this PR does not touch. Negative control: `test` was also red on dev at `a71193a2f`.
- One rerun each. Final: 26/26 complete, 0 failures, including `launchd drivers (bash 3.2)` and all five required checks.
- The second poller hit its 30-min cap in the same minute that the last check completed. Re-read directly twice: 26/0/0.
- `MERGEABLE`/`CLEAN`; squash-merged with `--match-head-commit` → `130486515`. No closing keyword (scan control fired).

**Done-gate**: surface edited = both Gate 0 copies + the script. Reach = pushed to origin/dev, which every loop pins, and skill-sync fast-forwards the main checkout on the next fire. `make test-launchd-drivers` at `8350e1c6c` under `env -i`: rc 0. Dry-runs under the idle armed **world** profile from a detached worktree at `130486515` (`AILANG_DRIVER_PINNED` set):
- healthy: rc 0, `DRY RUN ok`, `lanes=ok`.
- degraded (`MISSION_EXECUTOR_MODEL=pi:bogus/no-such-model`): rc 0, `lanes=DEGRADED(1)`, handed to opus.
- No world pidfile was written, and nothing was reloaded.

Ticket resolved with `--sha 130486515` (rc 0, replied to stapledon); 53 → 52 open.

**Progress**: 1 ticket resolved, filed 2026-09-28 and resolved 2026-10-13 (≈15.5 d). Clause 1 UNMEASURED. Clause 2 UNMET. Clause 3 MET (52 open). Clause 4: prior evidence only. Clause 5 preserved: judged, done-gate passed, required CI green.

**Routing evidence**: base-gate1=b283b2d4406da7978a442064641366b19a2912f3@2026-10-13T14:31:33Z; base=1304865155e78b36165a50f51ee9001de83331db@2026-10-13T16:19:20Z (gate4). Dev moved by this PR and one unrelated merge (`ea9142b7a`, #1816). `MISSION_ROUTING_NOTE`: empty. Over-ration: openrouter.
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer**: not spawned. The fix is mechanical and the charter says design only when warranted. The rotation was not advanced.
- **Planner**: resolver `agent-tool opus fail-closed:no-doc`; derive-planner-lane `opus fail-closed:no-doc`. The role carries the provider pin `codex:gpt-6.1-sol`, so per role-spawn-routing §2 it was routed straight to the pin, recipe `codex exec --sandbox workspace-write` in `.planner-wt-iter35` (96,042 tok).
- **Executor**: resolver `recipe codex:gpt-6.1-sol declared:provider-pin`. Run A tokens not reported (reaped); run B 72,030 tok.
- **Evaluator**: resolver `agent-tool sonnet declared:alias-pin`, spawned via the Agent tool `model=sonnet`, foreground, in its own worktree. Round 1: 70,366 tok. Round 2 (resumed): 77,538 tok. Judge independence: cross-vendor against the generators (codex) and a different model from the controller (opus).
- The operator's standing request to use the Agent tool was honoured for the one role whose resolved path is the Agent tool (the evaluator, both rounds). The planner and executor resolved to provider recipes. No role failed to spawn.
- **Metered**: $0 (ChatGPT subscription bucket + Anthropic subscription).

**Ruled out**:
- Re-opening on a squash-merge author. That would re-open every attended ruling in this repo.
- Trusting `git log -1 -G` (the latest hit), which picks up later rewords and ledger reflows.
- Earliest-flip selection, which hides a fleet re-resolve (judge N1, reproduced).
- Attributing either CI red to this PR: a different package each time, dev red on `a71193a2f`, and green on rerun.
- Fixing the docs red: outside fleet authority, and V1 owns repo reds.

**Retro lane**:
- (1) **A `run_in_background` Bash with no `timeout` is reaped at 10 minutes, and it takes a 30-minute wrapper's child with it.** The codex recipe says to use `run_in_background: true` and bounds the work with its own 30-minute `date +%s` deadline. It does not say the harness's own default ceiling is 10 minutes, so the inner deadline is unreachable. Instance 1 for this loop; it cost one executor run (M1 survived on disk). Fix candidate for the recipe: "pass `timeout` ≥ the wrapper's cap + margin".
- (2) The judge's own mutation drill found the defect the plan's design choice introduced: the planner chose earliest, as did my directive. Generator ≠ judge earned its slot this iteration.
- (3) The sandbox again refused a `.agents/**` write (instance ≥2 across iterations); the controller copy is routine.
- (4) `rotate-log` added a duplicate "Older entries are ARCHIVED" notice again (instance 5, after iterations 28, 31, 32 and 33), collapsed by hand. This is a ticket-shaped tooling defect (`rotate-log:*`), but the fleet does not self-source, and the ticket `rotate-log:index-regen-drops-orphan-rows` is a different defect.

**Next**: P2 #11 `weekly-report:unknown-mission`. D-FLEET-16 is with Mark.

## 36 — 2026-10-14 — P2 #11 ghost-resolved (`08ce76f30`); P2 #12 quorum salvage designed, quorum'd, built and judged PASS 88 (sonnet), PR #1824 BLOCKED on inherited dev red [HARNESS]

**Pick**: P2 #11 `weekly-report:unknown-mission`, the queue head after iteration 35. 52 open tickets at pick. Gate-2 reality check: the ticket's defect was **already fixed on origin/dev** by `08ce76f30` (2026-10-06, attended), after the last occurrence (2026-10-05). That made it a bookkeeping pick, so standing rule 1 allowed a second: P2 #12 `quorum:invalid-absent-on-quoted-literals`, which also covers the attended ticket `quorum:reviewer-invalid-json-discarded` (#941). The only `blocking=all` ticket, `agent-tool:sonnet-unavailable`, stays PARKED-ON-LANE as iterations 29 and 32–35 assessed it. Index grep found no prior attempt on either item. Orphan traces: no open fleet PR, no iter-36 worktree.

**Preflight**: armed; gh `sunholo-voight-kampff`; billing CLEAN. Directives: 0 on #1768 since `2026-10-13T00:00:23Z` (6 comments, all self). Self-notices rc 0: 0 crash notices, control `852:10` ok, watermarks 2/2. Ledger valid, 16 rows: 15 RESOLVED, 1 OPEN (D-FLEET-16). No rotation or sweep due.

**Observe**: origin/dev == pin == `42652c19e` at Gate 1. `SKILL.md` and all 12 resources `cmp`-identical to origin at the resolved symlink target. Dev CI on `42652c19e`: `CI`, `Build and Release` and `Deploy Documentation` all **failure**. NOT-GREEN checks: `test`, `docs-gate`, `docs-build`, `Build windows-latest`. Red causes, both outside fleet authority, owned by V1:
- `test`: `make check-git-exec` reports `git exec site absent from baseline: internal/executor/managed_agents/devapi_workspace.go (2)`, introduced by `ffef3402a` (gemini_agents, 2026-10-13).
- Windows: `TestDevAPI_ExecuteRoundTrip` and `TestPlanWorkspace_LocalDeltaCoversEverythingAndSparesIndex` in the same package.
- `docs-build`: MDX compile of `docs/docs/reference/env-vars.md` (the iteration-35 red, still open).

**Reality check**:
- **#11**: all 6 registry missions rc 0 (stapledon, docs and fleet included); `bogusxyz` rc 1, naming the registry list.
- **#12**: `reviewer.go:135` still `%.200q`, no salvage. The #941 refresh had refuted the doc's quoted-literal premise, so the doc needed a redesign before planning.

**Did**:
- **#11 resolved.** The independent judge (sonnet, Agent tool) scored **PASS**, 0 blocking. It confirmed:
  - ancestry and no later edits;
  - all 6 missions, under both `python3` and `/usr/bin/python3` 3.9;
  - the main-checkout copy byte-identical;
  - the negative control.
  `ailang mission ticket resolve weekly-report:unknown-mission --sha 08ce76f30` rc 0, replied to docs and stapledon; 52 → 51 open.
- **Designer** (`claude:claude-opus-5-5`, resolver `recipe … declared:provider-pin`; Agent-tool spawn DENIED by the spawn-pin hook, `deny:provider-pin`; routed to the `claude-sub` recipe, probe rc 0, run rc 0). It re-grounded `m-quorum-salvage-retry.md` on #941 with a 17-row Verification Log:
  - 7/7 `invalid` absences hold a reject, against 0/90 in the control;
  - the #1767 guard covers the all-absent shape but **not N-1** (one pass plus a hidden reject is still `proceed` at HEAD, in `synthesize` AND `RunSeatedQuorum`);
  - retry dropped.
  The controller measured V14 (the attended ticket exists in the store).
- **Quorum round 1** (`--author claude:claude-opus-5-5`; seats glm-5-3, kimi-k3, gpt6-1-sol; reserve gemini-4-argon): **BLOCKED**. glm wanted the truncation twig verified; kimi wanted Tier-2 verified; gemini passed with a depth-tracking refinement. gpt6-1-sol was absent both rounds (`openai error (429): You have no credits remaining`); the reserve filled the seat.
- **Designer revision 2** (one allowed revision): every objection was measured. The truncation twig sits inside the same invalid block, so one hook covers it. Tier-2 was scoped out (zero non-test callers of `RunQuorumWithEscalation`). Depth tracking adopted.
- **Quorum round 2**: BLOCKED by glm alone (kimi PASS, gemini PASS). The remaining objection disputed no direction: its own fix keeps retry out of scope.
- **Narrow-refinement carve-out**: the controller applied the reviewers' fixes verbatim (revision 3, `4c5070c50`):
  - **V21**, controller-run: a depth-tracked strict scan over the 7 stored raw windows recovers `reject` from **7/7** (4 unexpected-end, 3 syntax), each window opening `{` with no break inside it.
  - the Retry bullet rewritten in glm's words;
  - **T6c** (gate-violation path);
  - **M5b** key/value case;
  - salvaged-PASS rendering.
  glm's fix (4), "file `quorum:tier2-salvage-fold` before landing", was adapted: `ailang mission ticket file` refuses the fleet ("the fleet mission works tickets; it does not file them"), so it became charter queue row 16.
- **Planner** (`codex:gpt-6.1-sol`, resolver `recipe … declared:planner-lane-default-pin`; probe rc 0, run rc 0; 132,725 tok): 3 milestones. Every HEAD citation was re-verified and none was wrong. It confirmed that T6c needs no extra branch: `ValidateReviewResult`'s error returns as `perr`.
- **Executor** (`codex:gpt-6.1-sol`, one run, rc 0, 157,456 tok):
  - M1 `salvage.go` (scanner + `describeJSONError`);
  - M2 `foldAbsent` + additive artifact fields;
  - M3 skill remedy + changelog.
  7 mutants (M1, M4b, M5b, M6, M8, M8b, M8c) were red, then green after restore. The sandbox refused the `.agents` mirror sync; the controller synced outside it (`--check` rc 0, `cmp` identical).
- **Controller**: rebuilt one commit per milestone from `.snap/M<k>` (`23467c484`, `22bad5519`, `5a5e7cc1c`), with quorum tests green at each boundary. The final tree is byte-identical to the executor's (11/11 files `cmp`).
- **Outside-sandbox gates** at `5a5e7cc1c`, all rc 0:
  - `go test -race ./internal/mission/quorum/`
  - `go vet`, `gofmt`
  - `golangci-lint` (quorum + `cmd/ailang`)
  - `go test ./cmd/ailang/`
  - `sync-agents-skills.sh --check`
  - `make test-launchd-drivers` under `env -i`
- **Evaluator** (sonnet, Agent tool, foreground, own worktree `fleet-i36-judge`): **PASS 88**, 0 blocking. Its 10 own mutants were all killed (M7, M10, M10b, M12a/b, duplicate-key inversion, M3, M8-all, UTF-8 snap, M6b). Adversarial probes found no input that turns a reject into `proceed`, and no path by which a salvaged pass satisfies the guard. Findings 1 (charter row 16 missing) and 4 (an ambiguous skill sentence) were fixed in `6189802a0`.

**Gate 3b**: PR #1824 at head `6189802a0`: 25 checks, 21 green, including `launchd drivers (bash 3.2)` and `test-windows`. The 4 failures are exactly dev HEAD's set: `test`, `docs-gate`, `docs-build`, `Build windows-latest`.
- Negative control: the PR's `test` log fails at the same `check-git-exec` line as dev's run `38459585113`, naming `devapi_workspace.go`, which this PR does not touch.
- The Windows failures are in `internal/executor/managed_agents`.
- `test` and `docs-gate` are **required** (`branches/dev/protection` contexts: test, lint, build, docs-gate, UI build gate), so the PR is `BLOCKED`. No PR can merge to dev until dev is fixed.
- **NOT LANDED.** Both #12 tickets stay OPEN.
- Resume predicate: `gh api repos/sunholo-data/ailang/commits/$(git rev-parse origin/dev)/check-runs` shows `test` and `docs-gate` green; then update the branch, re-poll, squash-merge with `--match-head-commit`, run the done-gate at the merge, and resolve both tickets with that SHA.

**Done-gate** (partial, at the PR head): surface edited = quorum package + both `gate-2-pick.md` copies. `make test-launchd-drivers` rc 0. Dry-runs under the idle armed **world** profile from a detached worktree at `6189802a0` (`AILANG_DRIVER_PINNED` set):
- healthy: rc 0, `DRY RUN ok`, `lanes=ok`.
- degraded (`MISSION_EXECUTOR_MODEL=pi:bogus/no-such-model`): rc 0, `lanes=DEGRADED(1)`, handed to opus.
- No world pidfile written; nothing reloaded.

Reach is NOT confirmed (not on origin/dev).

**Progress**: 1 ticket resolved: #11, filed 2026-09-28, resolved 2026-10-14 (≈16 d; the fix itself landed 2026-10-06, ≈8 d). #12 built and judged but blocked on merge.
- Clause 1 UNMEASURED.
- Clause 2 UNMET.
- Clause 3 MET (51 open, plus charter row 16, which the fleet cannot file as a ticket).
- Clause 4: prior evidence only.
- Clause 5 preserved: nothing unjudged shipped.

**Routing evidence**: base-gate1=`42652c19eaf975860f4f91a12b191d24738def79`@2026-10-13T22:53:36Z; dev did not move during the iteration. `MISSION_ROUTING_NOTE`: empty. Over-ration: openrouter.
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer**: `claude:claude-opus-5-5` via the `claude-sub` recipe, 2 runs: create + the one protocol revision (tok: not reported by `claude -p` text output). Rotation pointer read `claude:claude-opus-5-5`; the driver resolved the same; not advanced by hand. Agent-tool attempt DENIED by the hook (`deny:provider-pin`), which is information, not a lane failure. Quorum: R1 blocked → R2 blocked (1 reject) → carve-out. Metered quorum cost $0.267 + $0.233.
- **Planner**: `codex:gpt-6.1-sol` (132,725 tok), resolver `declared:planner-lane-default-pin`.
- **Executor**: `codex:gpt-6.1-sol` (157,456 tok), resolver `declared:provider-pin`.
- **Evaluator**: `sonnet`, Agent tool, 2 spawns: the #11 verify (39,425 tok) and the #12 judge (70,645 tok). Judge independence: cross-vendor against the codex generators, and a different model from the opus designer and controller.
- The operator's standing request to use the Agent tool was honoured for every role whose resolved path is the Agent tool (the evaluator, both spawns). The designer was attempted via the Agent tool and refused by the hook. Planner and executor resolved to provider recipes. No role failed to spawn, and no role was skipped.
- **Metered**: ≈$0.50 (quorum reviewers). Every role ran on a subscription bucket.

**Ruled out**:
- Re-working #11: its fix was already on origin/dev.
- The doc's original "quoted literals" premise: refuted by #941 and re-measured.
- A same-model re-ask (retry): measured-zero benefit (V21 confirms all 7).
- Counting a salvaged pass: it could satisfy the zero-signal guard.
- Wiring Tier-2 now: zero production callers; row 16.
- Admin-merging over the required reds: a guardrail.
- Fixing the dev reds: `internal/executor/**`, the git-exec baseline and the env-vars generator are all outside fleet authority, and V1 owns repo reds.
- Attributing any PR red to this diff: the failing set is identical to dev's, with the same failing line.

**Retro lane**:
- (1) **[SKILL FIX `c9f92fe6b`, with iteration 35's retro (1) as the second friction]** **A bounded poll loop's deadline must leave room for one more sleep inside the tool timeout.** A 580 s deadline plus a 30 s sleep overran the 600 s Bash cap, and the call was moved to the background. Harmless here (caught by the notification), but it is the rule-7 trap in miniature. Use `deadline ≤ timeout − sleep − margin`.
- (2) **A required red on dev blocks every fleet landing, and the owner is disabled.** V1's kill switch is set, so "V1 owns repo reds" currently routes to nobody. Instance 1 of a whole-iteration landing loss for this loop. Surfaced to Mark as an FYI, not a decision.
- (3) The spawn-pin hook denied an Agent-tool designer whose resolver path was `recipe`. Expected per role-spawn-routing §2; no cost beyond one denied call.
- (5) `rotate-log` added the duplicate "Older entries are ARCHIVED" notice again (instance 6, after iterations 28 and 31–33, 35), collapsed by hand. Still ticket-shaped (`rotate-log:*`); the fleet does not self-source.
- (4) Quorum's OpenAI seat has been absent for lack of API credit since at least 2026-10-13 (both rounds). The reserve covers it, but the pool is effectively 4.

**Next**: when dev's required checks are green, land PR #1824 and resolve both #12 tickets. Then P2 #13 `ci:launchd-suite-flakes-openrouter-peer-and-gh-hang`. D-FLEET-16 is with Mark.

## 37 — 2026-10-14 — No landing possible: #12 (PR #1824) still BLOCKED on dev's required reds; #13 reproduction measured, live flakes outside fleet scope; judged PASS 88 (sonnet); D-FLEET-17/18 filed [HARNESS]

**Pick**: iteration 36's **Next**: land PR #1824 once dev's required checks are green, then P2 #13. 51 open tickets at pick. The only `blocking=all` ticket, `agent-tool:sonnet-unavailable`, stays PARKED-ON-LANE as iterations 29 and 32–36 assessed it. Orphan traces: iteration 36's worktree `fleet-i36-salvage` and PR #1824 are both live and both recorded in iteration 36's record. No other open fleet PR exists. The record for iteration 36 is on the PR branch, not on dev, so the charter on dev read one iteration stale. I caught that at Gate 2 from the existing worktree.

**Preflight**: armed; gh `sunholo-voight-kampff`; billing CLEAN. Directives: 0 on #1768 since `2026-10-13T00:00:23Z` (7 comments, all self). Self-notices rc 0: 0 crash notices, control `852:10` ok, watermarks 2/2. Ledger valid, 16 rows, 1 OPEN (D-FLEET-16). No rotation or sweep due.

**Observe**: origin/dev == pin == `42652c19e`, unchanged since iteration 36. `SKILL.md` and all 12 resources `cmp`-identical to origin at the resolved symlink target. `CI`, `Build and Release` and `Deploy Documentation` are all **failure**; NOT-GREEN checks are `test`, `docs-gate`, `docs-build` and `Build windows-latest`, the same set as iteration 36.

**Reality check (#12 resume predicate)**: FALSE. `test` and `docs-gate` are still red on origin/dev, and no open PR fixes either. V1, which owns repo reds, has its kill switch set (since 2026-09-14). PR #1824 is `BLOCKED`.
- `test` fails `check-git-exec` on `devapi_workspace.go:237,259` (`ffef3402a`).
- `docs-gate` fails on the MDX literal `'{"sort":"throughput"}'` at `env-vars.md:252`, generated from `internal/config/ai.go:43` (judge).

**Did**:
- **#13 reproduction (its mandated first step; measurement only)**: enumerated CI runs since 2026-09-28 and the `launchd drivers (bash 3.2)` job of every failed run. The controller covered 19 failures from 17:01Z (search cap); the judge's uncapped listing covered 21 from 13:08Z. Each log was grepped with a known-present control.
  - **B** (gh hang, `tools/launchd/test_driver_notify.sh`): last FAIL 09-28 13:15Z, before the ticket was filed; 0 since.
  - **A** (OpenRouter peer, `tools/eval/motoko_connection_probe.sh:79`): 09-28 13:44Z and 10-02; none since.
  - **Untracked** `bound measurement refuses an unsuccessfully terminated timer` (`tools/eval/test_motoko_connection_probe.sh:1136`): 4× on dev (10-07, 10-10, 10-12 ×2). This is the live flake in the fleet's own done-gate suite.
  - Others: 12 deterministic stale-mirror failures on 10-02 (since fixed) and 2 codex-oauth branch failures.
  - `tools/eval/**` is refused for the fleet by `_scope_verdict`.
- **Evaluator** (sonnet, Agent tool, foreground, `model=sonnet`, 60,226 tok): independently re-measured C1–C3 with its own `gh api` calls, its own enumeration, 21 logs and 6 jobs from successful runs. **PASS 88**. It confirmed the dispositions and made three corrections, all adopted:
  1. `docs/` is in fleet scope; only the source (`ai.go`) is not, and the page is generated.
  2. `Build windows-latest` is also red (not required).
  3. The window caveat hid the 13:08–17:01Z slice, which held B's last FAIL and an A.
- **Ledger**: filed D-FLEET-17 (who fixes dev's required reds while V1 is disabled; recommend A, an attended fix now; default C) and D-FLEET-18 (allowlist the two `tools/eval/*motoko_connection_probe*` files; recommend A; default C).
- **Queue**: rows 12 and 13 updated. #13 kept OPEN: one signature is gone, but the live ones are out of scope.
- Evidence: `fleet-mission-evidence/iteration37/findings.md`.

**Gate 3b**: not applicable. Nothing was pushed for merge. This record is appended to PR #1824's branch as docs-only commits, beside iteration 36's record.

**Progress**: 0 tickets resolved. Clause 1 UNMEASURED. Clause 2 UNMET; no landing is possible until dev is green. Clause 3 MET (51 open). Clause 4: prior evidence only. Clause 5 preserved; nothing shipped.

**Routing evidence**: base-gate1=42652c19eaf975860f4f91a12b191d24738def79@2026-10-14T06:11:32Z; base=42652c19eaf975860f4f91a12b191d24738def79@2026-10-14T06:21:38Z (gate4). Dev did not move. `MISSION_ROUTING_NOTE`: empty. Over-ration: openrouter.
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer, planner, executor**: NOT SPAWNED. This was deliberate; no role failed. No buildable item could land this fire:
  - #12 is built and judged, waiting on dev.
  - #13's live signatures are outside fleet scope (D-FLEET-18), and its in-scope half is not reproducing.
  - Pre-building #14 behind the same red would only add a second blocked PR with record conflicts against #1824, and resolutions (clause 2) need a landing either way.
  - Resolver answers, for the record: designer `recipe claude:claude-opus-5-5 declared:provider-pin`; planner `recipe codex:gpt-6.1-sol declared:planner-lane-default-pin`; executor `recipe codex:gpt-6.1-sol declared:provider-pin`.
- **Evaluator**: `sonnet` via the Agent tool (resolver `agent-tool sonnet declared:alias-pin`), 60,226 tok. Judge independence: a different model from the controller (opus).
- The operator's standing request to use the Agent tool was honoured for the one role that ran, whose resolved path is the Agent tool. No role failed to spawn.
- **Metered**: $0.

**Ruled out**:
- Admin-merging #1824 over required reds: a guardrail.
- Hand-editing the generated `env-vars.md`: it edits the mirror, not the source, and the next regeneration reverts it.
- Adding a `git_exec_baseline.txt` row: outside scope, and it loosens a ratchet, which is the owner's call.
- Resolving #13 as stale: A recurred after filing, and the timer flake is live.
- Working the timer flake: it is untracked, so it would be self-sourced, and it is out of scope.

**Retro lane**:
- (1) **A record that lands only with its PR makes the charter on dev stale for every later fire.** Iteration 36 put its record on the blocked PR branch, so iteration 37's Gate 1 read iteration 35 as the latest and the queue head as #11. I caught it only because a `fleet-i36-salvage` worktree existed. Instance 1. While a landing PR is blocked, the next fire must read the record from the PR branch (`gh pr list --author @me --state open` at Gate 1). It is not yet a skill rule, because one instance is below the bar.
- (2) Instance 2 of whole-iteration landing loss from a required dev red with its owner disabled (instance 1 is iteration 36). It is escalated as D-FLEET-17 instead of a second FYI.
- (3) The judge again caught controller scope over-statement: "docs/ outside scope" was wrong, since only the source is. This is the rule-3b scope class: a true fact quoted for a wider sentence.

**Next**: when D-FLEET-17's predicate holds (`test` and `docs-gate` green on origin/dev), update and land #1824, run the done-gate at the merge, and resolve both #12 tickets. Then P2 #14, or #13 if D-FLEET-18 = A. D-FLEET-16/17/18 are with Mark.

## 38 — 2026-10-14 — P2 #14 quorum artifact dir designed, quorum'd, built and judged PASS 90 (sonnet); stacked PR on #1824, BLOCKED on dev's required reds [HARNESS]

**Pick**: charter P2 #14 `quorum:artifact-dir-cwd-relative` (filed by v1, 2026-09-26). Iteration 37's **Next** was to land #1824 once dev is green. That predicate is still FALSE at `d15217d24`: `make check-git-exec` still fails on `devapi_workspace.go`, and `env-vars.md` still holds the MDX literal. So I took the next row and stacked it on #1824's branch, because both touch `artifact.go`. 51 open tickets. The only `blocking=all` ticket, `agent-tool:sonnet-unavailable`, stays PARKED-ON-LANE as iterations 29 and 32–37 assessed it.
- Index grep: no prior attempt.
- Orphan traces: iterations 36 and 37 left PR #1824 and worktree `fleet-i36-salvage`, and both records live on that branch only. The charter on dev still read iteration 35 and queue head #11, which iteration 36 had ghost-resolved. I caught it at Gate 2 from `gh pr list --author` plus `git worktree list`. This is instance 2 of iteration 37's retro (1).

**Preflight**: armed; gh `sunholo-voight-kampff`; billing CLEAN. Directives: 0 on #1768 since `2026-10-13T00:00:23Z` (8 comments, all self). Self-notices rc 0: control `852:10` ok, watermarks 2/2. Ledger valid (16 rows on dev, 18 on the branch). No rotation or sweep due.

**Observe**: origin/dev == pin == `bf46845d7` at Gate 1. `SKILL.md` and all 12 resources are `cmp`-identical to origin at the resolved symlink target; the CWD copy is the same. Dev CI has been red since `c43b6c0d9`. NOT-GREEN checks are `test` (`check-git-exec`), `docs-gate`/`docs-build` and `Build windows-latest`, the same set as iterations 36 and 37.

**Reality check (#14)**: reproduced.
- The pin's `.ailang/state/mission-quorum` holds 5 artifacts, the main checkout 164, the hand-used `~/.ailang/state/mission-quorum` 15, and `-world` 2.
- `.gitignore:105` ignores the path.
- Gate 2's `ls .ailang/state/mission-quorum/<doc-id>-*.json` is relative.
- A constraint the ticket does not mention: `internal/policytool/cli_ops.go:60` gates the ailang_only sandbox's `design-quorum` write to the same relative path. Moving the default would let a sandboxed write escape, so the fix needed a design.

**Did**:
- **Designer** (rotation turn `codex:gpt-6.1-sol`, recipe, probe rc 0) wrote `m-quorum-artifact-dir-stable.md`: opt-in registered env, repo-scoped namespace, and policy-marker suppression.
- **Quorum r1** (`--author codex`, sol benched; gemini-4-argon, oc-glm-5-3, oc-kimi-k3): **BLOCKED**.
  - glm: marker status of the real lanes is unmeasured.
  - kimi: the attended reader path is unspecified.
- **Designer revision**: setter audit (85 refs / 36 files, 0 in launchd/scripts/mission env/plists), reader decision table, pre-flight limits; importer deferred.
- **Quorum r2**: gemini pass, glm pass, kimi reject. kimi disputed cross-mode visibility with a concrete `proposed_fix`, not the design's direction. The controller applied its second alternative verbatim under the narrow-refinement carve-out: a fixed-order two-directory reader, AC11, and A1 downgraded to 0 (rev 3, `58decf4f2`).
- **Planner** (`codex:gpt-6.1-sol`, resolver `declared:planner-lane-default-pin`) wrote a 2-milestone plan with a 9-row premise check. Among its corrections: `job.go` `AgentPolicy()` trims, so it must not be reused for confinement; the controller-launch paths are `:2048` and `:2337`/`:2381`/`:2383`; and `mc_codex_env_args` already forwards `MISSION_*`.
- **Executor** (`codex:gpt-6.1-sol`, one 90-min-cap run, ~45 min):
  - M1 complete: 16/16 mutants red then green.
  - M2 built, but the sandbox refused the `.agents` mirror sync, which left 7 mutation rows and G1/G5/G13 UNINFORMATIVE.
- **Controller** ran the sync outside the sandbox; only `gate-2-pick.md`'s mirror changed. Gates outside the sandbox: G1/G12 went red under the controller env and green under `env -i`. Bisected to `MISSION_NAME`, which the suite's own unset list missed. Fix: scrub every inherited `MISSION_*` (`compgen -e`). Its mutant (drop the loop) is red with `MISSION_NAME=fleet`. After that, all 23 M2 shell mutants went red then green, with no uninformative rows left. Commits were rebuilt per milestone from `.snap` via the index; the final tree differs from `.snap/M2` only by that fix and the mirror.
- **Evaluator** (sonnet, Agent tool, foreground, own worktree): **PASS 90**, 0 blocking. It ran 9 mutants of its own (marker inversion, marker trim, explicit precedence, validation drop, suppression bypass, and three reader-order/fallback/error-dominance variants), all red. It confirmed that `design_quorum`'s policytool schema cannot pass `--artifact-dir`, and that every live `mission-*.env` sets a valid `MISSION_REPO` and no marker.
  - N1: a failed init exits 2 before `_mc_notify` is defined, so it logs but does not notify.
  - N2: legacy artifacts in other checkouts are not read (the importer is deferred).
- Evidence: `fleet-mission-evidence/iteration38/` (executor report, mutation tables before and after the sync, both quorum artifacts, dry-run logs).

**Gate 3b**: not applicable; nothing merged. The branch `fleet/i38-quorum-artifact-dir` is pushed at `b151064db`, and its PR is stacked on #1824.

**Done-gate** (at `b151064db`, ahead of the merge): the surface edited is the driver wiring, the helper, the quorum writer and both Gate 2 copies. `make test-launchd-drivers` under `env -i`: rc 0. Dry-runs under the idle, armed **world** profile from the worktree, with `AILANG_DRIVER_PINNED` set:
- healthy: rc 0, `lanes=ok`.
- degraded (`MISSION_EXECUTOR_MODEL=pi:bogus/no-such-model`): rc 0, `lanes=DEGRADED(1)`.
- Both log `quorum artifact directory: ~/.ailang/state/mission-quorum-world`.
- No world pidfile was written. Real quorum-directory manifests were unchanged before and after all gates.

Reach is NOT confirmed (unmerged). The done-gate must be re-run at the merge.

**Progress**: 0 tickets resolved. Clause 1 UNMEASURED. Clause 2 UNMET; #12 and #14 are both built and judged, waiting on dev. Clause 3 MET (51 open). Clause 4: prior evidence only. Clause 5 preserved; nothing shipped.

**Routing evidence**: base-gate1=bf46845d758849c59aa38cddd0f8ff97f7ac0e1d@2026-10-14T12:25:56Z; base=d15217d24ac93cccbb43aaff2ce708c7aa019445@2026-10-14T13:46:47Z (gate4). Dev moved by 7 unrelated commits (executor/docs/modelreg). `MISSION_ROUTING_NOTE`: empty. Over-ration: openrouter.
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer**: `codex:gpt-6.1-sol` (rotation turn after `claude:claude-opus-5-5`; pointer advanced). Create run: 172,938 tok. Revision: 107,683 tok. Rev 3 was the controller applying the carve-out. Quorum: r1 BLOCKED, r2 2/3 plus the verbatim fix; costs $0.1787 + $0.2288. Resolver: `recipe claude:claude-opus-5-5 declared:provider-pin` names the seed; the rotation chose codex.
- **Planner**: resolver `recipe codex:gpt-6.1-sol declared:planner-lane-default-pin` (138,928 tok). It ran in the iteration worktree, not a separate detached planner worktree (the doc was committed there, and the artifacts were committed in place).
- **Executor**: `recipe codex:gpt-6.1-sol declared:provider-pin` (177,078 tok).
- **Evaluator**: resolver `agent-tool sonnet declared:alias-pin`, spawned via the Agent tool with `model=sonnet`, foreground, in its own worktree (67,642 tok). Judge independence: cross-vendor against the codex generators, and a different model from the controller.
- The operator's standing request was honoured: every role was spawned at its resolved path, and the evaluator ran via the Agent tool. No role failed to spawn.
- **Metered**: $0.41 (quorum OpenRouter/Vertex reviewers). Everything else ran on the ChatGPT and Anthropic subscriptions.

**Ruled out**:
- Resolving the default against git's common dir: it moves policytool's sandboxed write outside the sandbox.
- Resolving it against `$HOME` unconditionally: same escape.
- A reader-only fix: it cannot recover writes into deleted trees.
- Parking after quorum r2: the one remaining reject carried a concrete fix that did not dispute direction (carve-out).
- Pre-building the legacy importer: deferred by the designer and kept out.
- Hand-editing the `.agents` mirror: the sync tool was used instead.

**Retro lane**:
- (1) **Records stranded on a blocked PR branch**, instance 2 (iteration 37 was instance 1). The charter on dev said iteration 35 and queue head #11, while the real state was iteration 37 on #1824's branch. Two instances meet the skill-fix bar. The fix candidate: Gate 1 reads open own-account fleet PR branches for newer records (`gh pr list --author @me --state open`, then `git show origin/<branch>:<charter>`). Not edited this iteration; see Gate 5.
- (2) The codex sandbox again refused a `.agents/**` write (instance ≥3). The controller sync is routine now.
- (3) The controller env leaked into a new shell suite (`MISSION_NAME`). The executor's sandbox env had no `MISSION_*`, so its green could not see it. This is the iteration-33 class again (instance 2): a suite that unsets a named list instead of a prefix.
- (4) A new `MISSION_*` env with `${VAR:-}` checks in bash is fine. The driver's early exit 2 is not notified (judge N1). It is a follow-up, not ticketed.

**Next**: when D-FLEET-17's predicate holds (`test` + `docs-gate` green on origin/dev), land #1824, then retarget and land this PR. Run the done-gate at each merge and resolve `quorum:invalid-absent-on-quoted-literals`, `quorum:reviewer-invalid-json-discarded` and `quorum:artifact-dir-cwd-relative`. Then P2 #15 `driver:unclassified-state-unreachable`. D-FLEET-16/17/18 are with Mark.
