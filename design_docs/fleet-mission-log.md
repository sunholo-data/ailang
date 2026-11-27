# Fleet Mission Log

Append-only, one entry per iteration, newest at the bottom. Charter: [fleet-mission.md](fleet-mission.md).

> **Older entries are ARCHIVED.** This file holds the newest 20. The full record of every
> iteration is in `fleet-mission-log-archive.md`, and a one-line index of ALL of them —
> the thing to grep before picking work, so the loop never repeats itself — is in
> `fleet-mission-index.md`.

## 120 — 2026-11-18 — D-FLEET-36 A + D-FLEET-38 A guard arm BUILT, not landed: rebuilt fresh as #2413 (supersedes draft #2108; judge sonnet PASS 93); merge waits on dev's inherited required `test` red [HARNESS]

**Pick**: queue rows 51/53 (`skill:design-doc-creator-grep-portability`, D-FLEET-36 A, and `skills:agents-copies-stale-non-mission`, D-FLEET-38 A; both ruled by Mark attended 2026-11-16). This is iteration 119's **Next**: its first branch (land #2411) was still blocked, since dev did not move and the attended pi_assets sync is not on the branch.
- 0 allowlisted directives on #2341 (7 comments).
- CLAIM sent before routing (`inbox_1794978645722_e8eeb66d`).
- Already-landed check: since `007809894` the only commit touching `pre-push` or the scope test is `97e792147` (#2382, D-FLEET-18). No open PR other than draft #2108 names these rows.
- Critical path: clause 2 is UNMET. Row 41 is attended-gated, so these are the next routable rows that move it.

**Preflight / observe**:
- Armed; gh `sunholo-voight-kampff`; billing CLEAN.
- Skill drift: resolved copy and pin copy both match origin/dev on every file (silent per-file loop over SKILL.md + resources).
- Gate 0 self-notice: watermark form rc 2 (twenty-fourth fire; neither watermark file exists). Not a verdict.
- Gate 1 base `022f68f1a8fd29f501df52a772db4710b9802cee@2026-11-18T05:09:20Z`, the same as iteration 119.
- Dev CI on it is unchanged: RED, 21 checks, 6 not green, all on `TestCloudAgents_RegistryMatchesTheDeletedRoutingTable` (attended `8e04a29c5`; handed to the `user` inbox by iteration 119). Not re-handed.
- Tickets: 19 open, the same list as iteration 119.
- Records 95–119 sit on the stacked record PRs (newest #2412). This record stacks on it.

**Did** (generator ≠ judge):
- **Reality check (controller)**: all 8 `grep -E "^\d+\."` sites are still at HEAD.
  - The rename-only set was re-measured with iteration 74's normaliser over all 41 paired skills: 17 identical, 14 rename-only, 10 content.
  - Of the ruling's 16, 14 are still rename-only. `collaboration-hub` and `trace-debugger` are now byte-identical (both copies edited by attended `a6868eecb`, 11-10). The arm keeps all 16, per "exactly those 16". An identical pair still needs the arm: once it is in the sync list, an edit to the `.claude` copy needs a re-sync push.
  - `headless-runner` (a content-class skill) has been deleted; `microrag` is a new content-class pair.
  - The cherry-pick of `bcf7f97ac` conflicts in the scope test and the charter, at #2382's lines.
- **Designer**: not spawned. The diff was pre-specified (#2108) and ruled. A routing outcome, not a spawn failure. Rotation pointer unchanged.
- **Planner**, opus (Agent tool, `MISSION-ROLE: planner`, foreground, 125,492 tok, 44 tool uses, 750 s): [plan](fleet-mission-evidence/iteration120/plan.md).
  - Conflict resolution: keep #2382's D-FLEET-18 lines first.
  - Five changes beyond the cherry-pick: changelogs re-dated and reworded "ruled A"; charter Authority lines after D-FLEET-18; pre-push comment "rename-only or identical"; one duplicate test label; a NEW loop asserting every other `.agents` skill stays refused, with a minimum count. #2108's own test let `agent-inbox`, `release-manager` and `codebase-organizer` through as a 17th arm.
  - Mutants: 17/17 killed. `make test-launchd-drivers`: 58/58 (386 s).
  - Corrected the controller's attribution of the identical pairs: the change is `a6868eecb`, not `f9ffb99c8`/`87f77e13e`.
- **Executor**, `claude:claude-haiku-5-5` via `claude-sub` (probe rc 0, 11 s; billing CLEAN; bash deadline watchdog, 1500 s cap). rc 0 in 73 s, 10 turns, 7,219 output tok.
  - Commit `8e78d4ad8`, applied with `git apply`.
  - G1 scope test, G2 `mission_decisions.sh --check`, G3 `sync-agents-skills.sh --check` and G4 `git diff --check`: all rc 0.
  - Mutants (a)/(b)/(c): 3/3 killed.
- **Controller checks**: the commit's changed lines equal the patch's (sorted `cmp`). G1 rc 0. Hand mutants: drop the `eval-gap-finder` arm → rc 1; drop the D-FLEET-36 line → rc 1.
- **Judge**, sonnet (Agent tool, `MISSION-ROLE: evaluator`, foreground, own detached worktree at `8e78d4ad8`, 62,578 tok, 458 s): **PASS 93**, 0 blocking. [Evaluation](fleet-mission-evidence/iteration120/judge-r1.md).
  - Fidelity holds: `_scope_is_harness` is unchanged and v1 keeps all new paths.
  - Charter Authority matches the guard (programmatic diff).
  - Premise re-derived with its own normaliser: 16/16 have zero residual.
  - origin/dev's pinned guard admits all 5 touched paths.
  - `changelog_fold.sh --check` rc 0; `make test-launchd-drivers` 58/58 (401 s).
  - 12/12 of its own mutants killed.
- **PR**: branch pushed through the pinned guard (rc 0), opened as **#2413**, head `8e78d4ad8`, 0 closing keywords. Draft #2108 got a superseded comment (count 0 → 1) and is left open until #2413 merges.

**Gate 3b**:
- #2413 head `8e78d4ad8`: 23/23 check-runs complete (poll pinned to the SHA, 30 min cap).
- Green: `launchd drivers (bash 3.2)` (the job that runs this change's tests), `checks`, `lint`, `build`, `docs-gate`, `govulncheck`, CodeQL, float determinism.
- Red, 6 runs: `test`, `test-windows`, `Build ubuntu-latest`, `Build windows-latest`, 2× `Build macos-latest`. Every one fails ONLY on `TestCloudAgents_RegistryMatchesTheDeletedRoutingTable`.
  - Logs were fetched ANSI-stripped and sized: 82 KB–5.8 MB.
  - Control: 169 `ok` package lines in each test log, and 23–24k `--- PASS` lines in each Build log. The only `FAIL` package is `internal/coordinator`.
  - This is the base red at `022f68f1a`. This diff has no Go.
- **NOT LANDED.** Tickets stay open. Park class: dev's red is attended (D-FLEET-17 A) and already handed. Not a new decision, so it does not go in DECISIONS.

**Ruled out**:
- Cherry-picking #2108 verbatim: it conflicts with #2382, and its test let a 17th skill through.
- Pushing the 8-site edit or the sync-list extension in the same PR: the iteration-16 rule (the pinned guard refuses those paths until this arm is on origin/dev).
- Trimming the arm to 14: the ruling says "exactly those 16".
- Working dev's `test` red: outside fleet scope (D-FLEET-17 A); already handed.
- Closing #2108 now: it stays open until #2413 merges.
- Re-handing the red: no new information.

**Observations (not ticketed; the fleet does not self-source)**:
- (a) `release-manager` is in `sync-agents-skills.sh`'s list (`95061c37f`), but the guard refuses `.agents/skills/release-manager` for the fleet. So a fleet re-sync of it cannot push. This came from the planner and is reported for Mark. No ruling admits it, so the new test keeps it refused.
- (b) Dev's required `test` red now blocks three fleet PRs (#2411, #2413 and the record stack), unchanged since `8e04a29c5`.
- (c) Self-notice watermark form rc 2, twenty-fourth consecutive fire.

**Retro lane**: no skill edit this fire. Friction: the record-stack-behind-a-red pattern (iterations 119 and 120) is iteration 39's rule working as written. It is not a new gap.

**Progress**: 19 open tickets, unchanged (built, not resolved). Clause 2 not moved this fire. Clause 1 UNMEASURED; 3 MET; 4 met by its letter; 5 preserved (the done-gate is owed at landing; `make test-launchd-drivers` already 58/58 ×2).

**Validation**: darwin/arm64, bash 3.2.57 (`/bin/bash`).
- `test_mission_scope_guard.sh` rc 0 (executor, controller, judge).
- `make test-launchd-drivers` 58/58 (planner, judge).
- Mutants: planner 17/17, executor 3/3, controller 2/2, judge 12/12.
- PR CI as above.
- Record checks: `mission_decisions.sh --check` (44 rows); STATUS rotation arithmetic 416 → 416 plus an archive grep (`ITERATION 117` → 1, control `ITERATION 116` → 1); `mission_routing_models_check.sh` on entry 120; `go test ./internal/mission/ -run Canonical`.

**Budget**: metered **$0** (subscription lanes).
- Planner opus 125,492 tok.
- Executor haiku 7,219 output tok (CLI self-report $0.017, not billed).
- Evaluator sonnet 62,578 tok.
- Controller `claude:claude-opus-5-5`: tok not reported.
- No quorum (no new doc).

**Next**: when dev's `test` is green, land #2413 (then retire draft #2108) and #2411 (after the attended pi_assets sync). After #2413 is on origin/dev, push the 8-site `[0-9]` edit and the sync-list extension (`sync-agents-skills.sh` SKILLS + `test_agents_skills_sync.sh`) on a fresh branch. Run the done-gate, then resolve both tickets. Otherwise D-42's fleet half (drafts #2141–#2150). Row 58 stays open (retire-or-own not ruled).

**Routing evidence**: base=022f68f1a8fd29f501df52a772db4710b9802cee@2026-11-18T05:52:15Z (Gate-4 read; equal to the Gate-1 read, drift rc 0).
- Controller `claude:claude-opus-5-5` (session; tok: not reported).
- Designer: resolver `recipe claude:claude-opus-5-5 declared:provider-pin`. **Not spawned**: the diff was pre-specified and ruled. Rotation pointer unchanged (`claude:claude-opus-5-5`).
- Planner: `derive-planner-lane.sh` → `opus fail-closed:env-pin`; resolver `agent-tool opus fail-closed:env-pin`. Agent tool `model=opus`, `MISSION-ROLE: planner`, foreground, 125,492 tok.
- Executor: resolver `recipe claude:claude-haiku-5-5 declared:provider-pin` → `claude-sub -p --model claude-haiku-5-5`, probe rc 0; run rc 0 (7,219 output tok).
- Evaluator: resolver `agent-tool sonnet declared:alias-pin` → Agent tool `model=sonnet`, `MISSION-ROLE: evaluator`, foreground, own worktree, 1 spawn (62,578 tok). Judge independence: cross-model (planner and controller opus, executor haiku, judge sonnet; same vendor).
- Driver routing note verbatim: lanes degraded: ; `planner`: **codex** lane `gpt-6.2-sol` unusable (probe rc=`75` — quota UNREADABLE (not measured — the lane may be fine;  see `ailang mission quota`)) → handed to `opus`; `executor`: **codex** lane `gpt-6.2-sol` unusable (probe rc=`75` — quota UNREADABLE (not measured — the lane may be fine;  see `ailang mission quota`)) → handed to `claude:claude-haiku-5-5`

## 121 — 2026-11-18 — D-FLEET-36 A + D-FLEET-38 A guard arm LANDED (#2413 → `55d336df7`, dev CI 18/18 green); follow-up 8-site `[0-9]` edit + 16-skill sync BUILT and judged (sonnet PASS 93) as `ae343722d`, pushes next fire under the iteration-16 rule [HARNESS]

**Pick**: queue rows 51/53 (`skill:design-doc-creator-grep-portability`, D-FLEET-36 A; `skills:agents-copies-stale-non-mission`, D-FLEET-38 A; both ruled by Mark attended 2026-11-16). This is iteration 120's **Next**, now routable: dev's required `test` red cleared at `55788b010` (gpt-6.1-sol → gpt-6.2-sol ratification row), and `7ec6edd25` is green on all 22 checks.
- 0 allowlisted directives on #2341 (8 comments).
- CLAIM sent before routing (`[fleet i121] CLAIM rows 51/53`, controlplane).
- Already-landed check: #2413 was OPEN/BLOCKED at Gate 2 on head `8e78d4ad8`, and its 6 reds were the inherited test. No other open PR names these rows. Draft #2108 was still open.
- Critical path: clause 2 is UNMET. Row 41 (#2411) is attended-gated (its `checks` is still red on `verify-pi-assets`; no attended sync commit yet), so rows 51/53 are the routable rows that move it.

**Preflight / observe**:
- Armed; gh `sunholo-voight-kampff`; billing CLEAN.
- Skill drift: the resolved symlink copy matches origin/dev on all 14 files (SKILL.md plus 13 resources), checked per file; the pin is at origin/dev.
- Gate 0 self-notice: watermark form rc 2 (twenty-fifth fire; neither watermark file exists). Not a verdict.
- Gate 1 base `7ec6edd25943d1fa14dd511bf925cba6721d6b33@2026-11-18T11:58:17Z`. Dev CI green (`CI`, `Build and Release`, Docs all success; 22 check-runs, 0 not green).
- Tickets: 20 open. One is new since iteration 120: `quorum:codex-gpt-6.2-sol-not-allowlisted` (stapledon, first seen 06:00Z today). Not picked; it arrived behind rows 51/53.
- Records 95–120 sit on the stacked draft record PRs (newest #2416). This record stacks on it. The stack's bottom is draft #2171 (a design parked for human review), so it is not landable by the loop now that dev is green.

**Did** (generator ≠ judge):
- **Land #2413**: `gh pr update-branch` merged dev into it (head `41f55ffdc`), re-running CI on a green base. 24/24 check-runs complete, 0 not green (SHA-pinned poll, 30 min cap). The PR diff against dev was checked byte-identical (`cmp`) to the diff the iteration-120 judge scored at `8e78d4ad8`. 0 closing keywords. Merged with `--match-head-commit 41f55ffdc` → **`55d336df7`**.
- **Retire draft #2108**: comment posted first with `--body-file` (count 1 → 2), then closed.
- **Designer**: not spawned. The follow-up diff is fully specified by the two rulings. A routing outcome, not a spawn failure. Rotation pointer unchanged.
- **Planner**, opus (Agent tool, `MISSION-ROLE: planner`, foreground, 120,625 tok, 41 tool uses, 610 s): [plan](fleet-mission-evidence/iteration121/plan.md), reviewed patch [followup-41f55ffdc.patch](fleet-mission-evidence/iteration121/followup-41f55ffdc.patch) (sha256 `32f4cb10…f2bff19b`, 19 files, +181/-124).
  - Re-measured at `41f55ffdc`: all 8 `\d` sites present; 14 rename-only + 2 identical; no `.agents`-only file and no mode difference in any of the 16.
  - Change: the 8 sites become `^[0-9]+\.` (BSD grep 2.6.0 selects identical lines before and after; `^d+\.` control matches only literal-d lines); SKILLS gains exactly the 16 names; `sync-agents-skills.sh` regenerated 14 `.agents/skills/*/SKILL.md`; `test_agents_skills_sync.sh` gains an exact-list check (23 names, read from the list the script iterates) and a `\d`-regression check over both design-doc-creator copies; one changelog fragment.
  - Gates: sync test 7/7, `--check` 23 skills, scope-guard suite, `changelog_fold.sh --check`, `make check-changelog`, `git diff --check`, `make test-launchd-drivers` 58/58 (332 s). Mutants 5/5 killed.
  - Guard: the #2413 guard admits all 19 paths for `fleet`. The pin's guard (`7ec6edd25`) refuses 16 of them.
- **Executor**, `claude:claude-haiku-5-5` via `claude-sub` (probe rc 0; billing CLEAN; bash deadline watchdog, 1500 s cap). rc 0, 8 turns, 5,831 output tok. Applied the patch with `git apply` → commit **`ae343722d`** on local branch `fleet/i121-dfleet36-38-followup`. G1–G5 rc 0. Mutants (a)/(b) 2/2 killed and restored by sha256.
  - It flagged that its first mutant attempt was a `sed` no-op (passing for the wrong reason), discarded it, and re-applied with `perl`.
- **Controller checks**: the commit's changed lines equal the patch's (sorted compare); G1 rc 0; hand mutant (17th name `model-manager` in SKILLS) → G1 rc 1, restored by sha256.
- **Judge**, sonnet (Agent tool, `MISSION-ROLE: evaluator`, foreground, own detached worktree at `ae343722d`, 74,004 tok, 20 tool uses, 370 s): **PASS 93**, 0 blocking. [Evaluation](fleet-mission-evidence/iteration121/judge-r1.md).
  - Fidelity: exactly 8 sites, exactly 16 names, no wildcard, design-doc-creator not in SKILLS.
  - Data loss: none. Own perl normaliser (`claude[- ]code|codex|claude` → X): every differing file is rename-class wording, residual 0.
  - Guard: `fleet` admits all 19 paths. `v1` admits 17 and refuses the two `tools/launchd` scripts, which is the existing harness rule, so product loops are not newly refused.
  - Gates all rc 0, including `make test-launchd-drivers`. 11/12 own mutants killed. Survivor M10 edits the test's threshold and the pattern together. Non-blocking: assert exactly 4 `[0-9]` sites per copy rather than ≥4.
- **NOT PUSHED**: the iteration-16 rule (charter Authority section): the guard that judges a push is the PINNED copy (`core.hooksPath` = the pin worktree at `7ec6edd25`), which does not carry #2413's arm. The next fire's pin will be at or after `55d336df7`.

**Gate 3b**:
- #2413 head `41f55ffdc`: 24/24 complete, 0 not green.
- Merge `55d336df7` on dev: 18/18 check-runs complete, 0 not green (poll pinned to the full SHA). The 4 not present relative to `7ec6edd25` (`deploy`, `docs-build`, `docs-changes`, `docs-gate`) are the path-filtered docs workflow. This diff touches no docs path, and `docs-gate` was green on the PR head.
- **#2413 LANDED.** Tickets stay open: their fix is two-part, and the done-gate runs after the follow-up is on origin/dev.

**Ruled out**:
- Pushing the follow-up this fire: refused by the pinned guard by construction (iteration-16 rule). `--no-verify` and re-pointing `core.hooksPath` are forbidden.
- Landing the record stack now that dev is green: its base is draft #2171 (a design parked for human review), so it is not a red-only block.
- Folding the planner's extra `\d` find (below) into this change: outside both rulings, and the fleet does not self-source.
- Tightening the regression check to exactly 4 sites in this fire: non-blocking, and changing the patch now would void the judged diff. Recorded for the push fire, at the planner's option.
- Working #2411: still needs the attended `make pi-assets` commit.

**Observations (not ticketed; the fleet does not self-source)**:
- (a) The planner found the same `\d` class in `sprint-planner/scripts/create_sprint_json.sh:204` (both copies, inside `grep -oE`). It is inside fleet scope (`sprint-*`) but no ticket names it. Reported for Mark.
- (b) Iteration 120's (a) still stands: `release-manager` is in `sync-agents-skills.sh` while the fleet guard refuses its `.agents` path.
- (c) The charter's list of two-way skills is out of date at HEAD: headless-runner is gone, release-manager was synced on 11-12, and microrag is newly different. Ten paired skills still differ after the follow-up (design-doc-creator plus 9). The fragment names them.
- (d) Self-notice watermark form rc 2, twenty-fifth consecutive fire.
- (e) The executor's commit trailer credits Opus (my directive's verbatim text). The planner and controller are opus and the executor haiku. Record the actual roles here rather than amend a judged commit.

**Retro lane**: no skill edit this fire. No friction gap with ≥2 recorded instances. The pin/guard ordering cost a fire, as designed (iteration 16).

**Progress**: 20 open tickets (one new from stapledon). Rows 51/53 moved from BUILT to guard-arm LANDED, follow-up BUILT; neither ticket is resolved yet. Clause 2 not moved this fire. Clause 1 UNMEASURED; 3 MET; 4 met by its letter; 5 preserved (`make test-launchd-drivers` 58/58 ×2 on the follow-up; the done-gate is owed at its landing).

**Validation**: darwin/arm64, bash 3.2.57 (`/bin/bash`).
- PR #2413 CI 24/24 green; dev `55d336df7` 18/18 green.
- Follow-up: sync test 7/7 (planner, executor, controller, judge); `make test-launchd-drivers` 58/58 (planner, judge).
- Mutants: planner 5/5, executor 2/2, controller 1/1, judge 11/12.
- Record checks: `mission_decisions.sh --check`; STATUS rotation arithmetic plus an archive grep; `mission_routing_models_check.sh` on entry 121; `go test ./internal/mission/ -run Canonical`.

**Budget**: metered **$0** (subscription lanes).
- Planner opus 120,625 tok.
- Executor haiku 5,831 output tok (CLI self-report $0.015, not billed).
- Evaluator sonnet 74,004 tok.
- Controller `claude:claude-opus-5-5`: tok not reported.
- No quorum (no new doc).

**Next**: push `fleet/i121-dfleet36-38-followup` (`ae343722d`; or re-apply `fleet-mission-evidence/iteration121/followup-41f55ffdc.patch` on fresh origin/dev if it no longer applies). First confirm the pinned guard now admits all 19 paths. Open the PR, Gate 3b, merge, then run the done-gate (reach, dry-run healthy and degraded, `make test-launchd-drivers`) and resolve both tickets with `ailang mission ticket resolve … --sha`. Then land #2411 once the attended pi_assets sync lands. Otherwise the new stapledon ticket or D-42's fleet half (drafts #2141–#2150).

**Routing evidence**: base=55d336df78ec31ae3b082e5b3d38b5921a67fe55@2026-11-18T12:39:25Z (Gate-4 read; drift from Gate 1 `7ec6edd25` is this iteration's own #2413 merge).
- Controller `claude:claude-opus-5-5` (session; tok: not reported).
- Designer: resolver `recipe claude:claude-opus-5-5 declared:provider-pin`. **Not spawned**: the diff is pre-specified by the rulings. Rotation pointer unchanged.
- Planner: `derive-planner-lane.sh` → `opus fail-closed:env-pin`; resolver `agent-tool opus fail-closed:env-pin`. Agent tool `model=opus`, `MISSION-ROLE: planner`, foreground, 120,625 tok.
- Executor: resolver `recipe claude:claude-haiku-5-5 declared:provider-pin` → `claude-sub -p --model claude-haiku-5-5`, probe rc 0; run rc 0 (5,831 output tok).
- Evaluator: resolver `agent-tool sonnet declared:alias-pin` → Agent tool `model=sonnet`, `MISSION-ROLE: evaluator`, foreground, own worktree, 1 spawn (74,004 tok). Judge independence: cross-model (planner and controller opus, executor haiku, judge sonnet; same vendor). The #2413 landing rests on iteration 120's sonnet PASS 93, with the merged diff checked byte-identical to the judged one.
- Driver routing note verbatim: lanes degraded: ; `planner`: **codex** lane `gpt-6.2-sol` unusable (probe rc=`1`) → handed to `opus`; `executor`: **codex** lane `gpt-6.2-sol` unusable (probe rc=`1`) → handed to `claude:claude-haiku-5-5`

## 122 — 2026-11-18 — D-FLEET-36 A + D-FLEET-38 A LANDED and RESOLVED: follow-up #2419 → `d2f2ec238` (fresh judge sonnet PASS 92; PR CI 24/24, dev CI 18/18 green); done-gate run; tickets 21 → 19 [HARNESS]

**Pick**: queue rows 51/53 (`skill:design-doc-creator-grep-portability`, D-FLEET-36 A; `skills:agents-copies-stale-non-mission`, D-FLEET-38 A). This was iteration 121's **Next**: push the judged follow-up `ae343722d` now that the pin carries #2413's guard arm.
- 0 allowlisted directives on #2341 (9 comments) or on the previous issue #2172 (31 comments).
- Already-landed check: no open or merged PR carried `fleet/i121-dfleet36-38-followup`, and the branch existed only locally (worktree `fleet-i121-wt`, clean, HEAD `ae343722d`).
- Critical path: clause 2 is UNMET, and rows 51/53 are the routable rows that move it. Row 41 (#2411) is still attended-gated.
- Two tickets have arrived since iteration 120 and neither was picked. Both rank below the attended list by `slots_lost`, and both are now queue rows 65/66 (untriaged).

**Preflight / observe**:
- Armed; gh `sunholo-voight-kampff`; billing CLEAN.
- Skill drift: the resolved symlink copy matches origin/dev on all 14 files (SKILL.md plus 13 resources), checked per file. The pin is at origin/dev.
- Gate 0 self-notice: watermark form rc 2 (twenty-sixth fire; neither watermark file exists). Not a verdict. Stapledon has now ticketed it (row 66).
- Gate 1 base `18e848db69a8d84cb4d50d9484ed0beb8e514224@2026-11-18T18:44:41Z`. Dev CI is green (`CI`, `Build and Release`, Docs all success; 18 check-runs, 0 not green).
- Records 95–121 sit on the stacked draft record PRs (newest #2417), and this record stacks on that one.

**Did** (generator ≠ judge):
- **Pinned guard re-measured before push**: `tools/launchd/githooks/pre-push` at the pin (`18e848db6`) admitted all 19 paths under `MISSION_NAME=fleet` (rc 0). The `MISSION_NAME=v1` control returned rc 1 on the two `tools/launchd` scripts, which is the existing harness rule.
- **Base check**: the branch's parent `41f55ffdc` is an ancestor of origin/dev, and none of origin/dev's 5 newer files intersect the 19 (the incoming list was the control). So the PR diff equals the judged diff, and nothing was re-applied.
- **Pushed** through the pinned guard (rc 0) and opened **#2419**, with 0 closing keywords.
- **Designer, planner, executor**: not spawned. The diff was planned (opus), executed (haiku) and judged (sonnet PASS 93) in iteration 121, and this fire only lands it. This is a routing outcome, not a spawn failure. The rotation pointer is unchanged.
- **Judge**: sonnet (Agent tool, `MISSION-ROLE: evaluator`, foreground, own detached worktree at `ae343722d`, 71,016 tok, 12 tool uses, 416 s) scored **PASS 92** with 0 blocking. [Evaluation](fleet-mission-evidence/iteration122/judge-r1.md).
  - Fidelity: exactly 8 sites and exactly 16 names, no wildcard, and design-doc-creator only in a comment.
  - Data loss: none. There is no `.agents`-only file in any of the 16 at the parent. Every removed token is Codex rename wording, and all 16 directories are identical to `.claude` afterwards.
  - Gates: sync test 7/7, `--check` 23 skills, `check-changelog`, `diff --check`, and `make test-launchd-drivers` 58/58 (5m31s), all rc 0.
  - Mutants: 10/11 killed, each restored by sha256. The survivor, M11, deletes the changelog fragment, which nothing requires; that is pre-existing.
  - Portability: BSD grep selects identical lines with `[0-9]` and `\d`, and the `^d+\.` control matches only literal-d lines. GNU grep remains unmeasured.
  - Non-blocking: the site check is `>= 4` per copy, the changelog's list of still-differing skills names microrag where the ruling names headless-runner, and codex/pi now read `.claude` wording, which ruling A intends.
- **Controller checks on the judge**: its worktree was clean afterwards at the same HEAD. A `^\d` grep found 0 lines in the `.claude` copy, and `[0-9]` counted 9 per copy (≥ 4).
- **Merge**: PR CI 24/24 complete with 0 not green (SHA-pinned poll, 30 min cap; direct re-read checks=24, MERGEABLE CLEAN). It merged with `--match-head-commit ae343722d…` → **`d2f2ec238`**.
- **Done-gate** at `d2f2ec238`, run in the judge worktree after moving it to the merge. [Gates](fleet-mission-evidence/iteration122/gates.md).
  - The surface is the skills and sync script every mission runs from the pinned origin/dev, and it reached origin/dev by merge.
  - Dry-runs: stapledon (fire started 2 min earlier) and world were mid-fire, triage idles out before lane resolution, and v1, docs and motoko are disabled. So the dry-runs used `MISSION_PROFILE=fleet` under a synthetic HOME (charter fallback, `AILANG_WORKTREE_SWEEP=0`, `AILANG_DRIVER_PINNED=d2f2ec238…`, `env -i`).
    - Healthy: rc 0 (87 s), `lanes=DEGRADED(2)` natively.
    - Degraded (`MISSION_EXECUTOR_MODEL=codex:bogus-model-i122`): rc 0 (93 s), executor → `claude:claude-haiku-5-5`.
    - Pins were 40 before and 40 after (diff empty), and no pidfile was left.
  - `make test-launchd-drivers`: rc 0, 330 s, 58/58.
- **Resolved** both tickets with `ailang mission ticket resolve … --sha d2f2ec238550dd02f9b3912fc46249d86b658726`, and each replied to its filer (attended). Open tickets went 21 → 19, re-listed.

**Gate 3b**:
- #2419 head `ae343722d`: 24/24 complete, 0 not green.
- Merge `d2f2ec238` on dev: 18/18 check-runs complete, 0 not green (poll pinned to the full SHA). `CI` and `Build and Release` both completed/success at `d2f2ec238`. The 4 docs-path checks are absent, as at `55d336df7`, because the diff touches no docs path.
- **#2419 LANDED; rows 51/53 RESOLVED.**

**Ruled out**:
- Re-planning or re-executing the follow-up: the judged commit applied unchanged on a base already in origin/dev.
- Folding in iteration 121's observations (the `sprint-planner/scripts/create_sprint_json.sh:204` `\d` site, and `release-manager` in the sync list while the guard refuses it): neither is ticketed, and the fleet does not self-source.
- Waiting for stapledon to idle for the dry-run: its fire had just started. The synthetic-HOME fallback is the charter's prescribed alternative, and pins were checked before and after.
- Working rows 65/66 this fire: one item per iteration.

**Observations (not ticketed; the fleet does not self-source)**:
- (a) **Every `codex:gpt-6.2-sol` probe fails `model_not_found`**: "The 'gpt-6.2-sol' model is not supported when using Codex with a ChatGPT account" (both dry-runs, verbatim in [gates.md](fleet-mission-evidence/iteration122/gates.md)). That is the cause of this fire's routing note (planner → opus, executor → haiku), and probably related to stapledon's row 65. Lane routing is HD-2a policy class, so it goes to Mark, not to a fleet fix.
- (b) The self-notice rc 2 has now run 26 consecutive fires. It is row 66.
- (c) Iteration 121's (a) and (b) still stand.
- (d) `rm -f /tmp/…verdict*` under the Bash tool's zsh errored `no matches found`, a harmless no-op because the path was fresh. It is the documented zsh trap, and it is one more reason to keep pollers under `/bin/bash`.

**Retro lane**: no skill edit this fire. No friction gap with ≥ 2 recorded instances that is not already ticketed.

**Progress**: 21 → 19 open tickets. Clause 2 moved (2 resolutions). Both tickets were filed 2026-10-03, about 46 days ago, so this closes long-standing items rather than improving the median. Clause 1 UNMEASURED; 3 MET; 4 met by its letter; 5 preserved (done-gate run).

**Validation**: darwin/arm64, bash 3.2.57 (`/bin/bash`).
- PR #2419 CI 24/24 green; dev `d2f2ec238` 18/18 green.
- Judge gates rc 0, including `make test-launchd-drivers` 58/58. The controller's done-gate run at the merge was 58/58.
- Mutants: judge 10/11.
- Record checks: `mission_decisions.sh --check`, STATUS rotation arithmetic (416 → 416) plus an archive grep (`ITERATION 119` = 1, control `ITERATION 118` = 1), `mission_routing_models_check.sh` on entry 122, and `go test ./internal/mission/ -run Canonical`.

**Budget**: metered **$0** (subscription lanes).
- Evaluator sonnet: 71,016 tok.
- Controller `claude:claude-opus-5-5`: tok not reported.
- No designer, planner or executor run, and no quorum (no new doc).

**Next**: rows 65/66 by `slots_lost` (both 1 slot, blocking none). Row 65 (`quorum:codex-gpt-6.2-sol-not-allowlisted`) needs re-measurement against observation (a), and if the fix is lane routing it parks for Mark under HD-2a. Row 66 (`gate0:self-notices-rc2-no-watermark-file`) is a mechanical candidate in harness scope (`scripts/mission_gate0_self_notices.sh`). Land #2411 once the attended pi_assets sync lands. Otherwise D-42's fleet half (drafts #2141–#2150).

**Routing evidence**: base=d2f2ec238550dd02f9b3912fc46249d86b658726@2026-11-18T19:05:09Z (Gate-3b read; drift from Gate 1 `18e848db6` is this iteration's own #2419 merge).
- Controller `claude:claude-opus-5-5` (session; tok: not reported).
- Designer: resolver `recipe claude:claude-opus-5-5 declared:provider-pin`. **Not spawned**: the diff was pre-specified by the rulings and already built. Rotation pointer unchanged.
- Planner: resolver `agent-tool opus fail-closed:env-pin`. **Not spawned**: the plan was carried from iteration 121 (opus, 120,625 tok).
- Executor: resolver `recipe claude:claude-haiku-5-5 declared:provider-pin`. **Not spawned**: the commit `ae343722d` was carried from iteration 121 (haiku, 5,831 output tok).
- Evaluator: resolver `agent-tool sonnet declared:alias-pin` → Agent tool `model=sonnet`, `MISSION-ROLE: evaluator`, foreground, own worktree, 1 spawn (71,016 tok). Judge independence: cross-model (controller opus, executor haiku, judge sonnet; same vendor). This is a fresh judge on the pushed head, on top of iteration 121's PASS 93.
- Driver routing note verbatim: lanes degraded: ; `planner`: **codex** lane `gpt-6.2-sol` unusable (probe rc=`1`) → handed to `opus`; `executor`: **codex** lane `gpt-6.2-sol` unusable (probe rc=`1`) → handed to `claude:claude-haiku-5-5`

## 123 — 2026-11-19 — `gate0:self-notices-rc2-no-watermark-file` LANDED and RESOLVED: #2421 → `cf540337e`; Gate 0 step 6a is a DEGRADED whole-issue read when no watermark file exists, instead of a floor; independent judge sonnet PASS 91; ticket resolved about 14 h after filing [HARNESS]

**Pick**: queue row 66 (`gate0:self-notices-rc2-no-watermark-file`, stapledon 99, 1 slot). This was iteration 122's **Next**.
- Row 65 (`quorum:codex-gpt-6.2-sol-not-allowlisted`) has the same `slots_lost`. Its fix lives in `cmd/ailang/design_quorum.go`, which is outside fleet scope, and the lane question is HD-2a. So row 66 was the routable row of the pair.
- Critical path: clause 2 is UNMET. Row 66 was filed 2026-11-18 12:48Z, and landing it now resolves it well inside 48 h.
- 0 allowlisted directives on #2341 (10 comments) or on the previous issue #2172 (31). No new tickets since iteration 122; 19 open.
- Already-landed check:
  - `git log origin/dev` has no commit for the signature.
  - No open or merged PR matched `self-notices` except #2132 (iteration 87, a different ticket).
  - No `i123` worktree existed.

**Preflight / observe**:
- Armed; gh `sunholo-voight-kampff`; billing CLEAN.
- Skill drift: the resolved symlink copy matches origin/dev on all 14 files, checked per file. The pin is at origin/dev `d2f2ec238`.
- Gate 1 base `d2f2ec238550dd02f9b3912fc46249d86b658726@2026-11-19T01:29:19Z`. Dev CI green (`CI` and `Build and Release` success; 18 check-runs, 0 not green).
- Records 95–122 sit on stacked draft PRs (newest #2420), and this record stacks on that one.
- Gate 0 self-notice: rc 2 for the twenty-seventh fire. Neither `mission-2341-last-seen` nor `mission-2172-last-seen` exists. That reproduces the ticket first-party.

**Root cause** (controller-measured, then re-derived by the planner):
- Since #981 (`351c415de`/`24c77c15b`, 2026-10-26), `mission_directives.sh` creates the watermark file only on `--mark`, that is, only when an allowlisted directive is processed. It reads an absent file from the epoch itself.
- The self-notices design (`implemented/v0_53_1/sprint-plan-gate0-self-notices.md` §2.1, 2026-10-03) assumed at least the `-prev` file exists. That held while Gate 0 wrote the watermark at read time.
- Planner measurement: the newest `*-last-seen` file on the rig is dated 2026-10-27, the day after #981. Stapledon's ticket note "fires 73–75 read rc 0 with the watermark absent" is that transition.

**Did** (generator ≠ judge):
- **Designer**: not spawned. The spec is the ticket plus the repro, a mechanical script fix, following iterations 87/105/109's precedent. This is a routing outcome, not a spawn failure. Rotation pointer unchanged.
- **Planner** opus (Agent tool, `MISSION-ROLE: planner`, foreground, 152,523 tok, 39 tool uses, 584 s).
  - It confirmed the root cause and the epoch's path through `validate_ts`, the BSD/GNU arms and the in-window string compare.
  - Consumer grep: no code consumer of the rc 2 text in `cmd internal tools .github`.
  - It prototyped the patch on a scratch copy and ran a 7-mutant drill. [Plan](fleet-mission-evidence/iteration123/plan.md).
  - Finding: with an epoch window, stapledon's `-prev` issue #410 carries an unseen crash notice (2026-11-13T14:15:39Z rc=1) that the floor had hidden. The fleet's own two issues have none.
- **Executor** haiku (Agent tool, `MISSION-ROLE: executor`, foreground, 90,778 tok, 23 tool uses, 501 s) applied the plan verbatim as `1ffb79aae` (6 files, +90/−17). [Report](fleet-mission-evidence/iteration123/executor-report.md).
  - The change: all-absent is now `watermark: DEGRADED — … reading the WHOLE of each issue since 1970-01-01T00:00:00Z` plus a `watermarks 0/N, all absent: whole-issue read` verdict tag, exit 0/1 by hits. Present-but-unreadable/empty/invalid files stay rc 2. The mixed path and `--since` are unchanged.
  - Tests: arm 21 rewritten as all-absent (rc 1 with a notice, rc 0 without), plus an absent-then-malformed rc 2 case. The suite went 168 → 176.
  - Docs: step 6a in both `gate-0-preflight.md` copies (synced by `sync-agents-skills.sh`), a dated amendment to the v0_53_1 plan, and a changelog fragment.
  - Gates: `make test-launchd-drivers` 58/58 at `1ffb79aae`. Live reads: fleet rc 0 DEGRADED, stapledon rc 1.
  - Deviation: commit trailer `Claude Haiku 5.5` instead of the briefed Opus line, because haiku wrote the commit. Accepted as accurate.
- **Judge** sonnet (Agent tool, `MISSION-ROLE: evaluator`, foreground, own detached worktree at `1ffb79aae`, 66,764 tok, 7 tool uses, 242 s): **PASS 91**, 0 blocking. [Evaluation](fleet-mission-evidence/iteration123/judge-r1.md).
  - "0 readable ⇒ all absent" holds on every path. A directory reads empty → rc 2; a broken symlink counts as absent.
  - Live before/after on the fleet's own args: base rc 2, new rc 0.
  - Mutants: 11/11 killed, restored by sha256.
  - Non-blocking: the per-file `ABSENT … single-watermark read` wording is misleading when all are absent; the `ABSENT -ne GIVEN` internal guard is unreachable; the judge did not grep consumers repo-wide.
- **Controller checks on the judge**:
  - The thin report (7 tool uses for 11 mutants) was re-measured. The controller ran the restore-`exit 2` mutant itself in the judge worktree: 168 passed / 8 failed. The judge said 7, the planner 8. That is the same kill; the count difference is the mutation's exact form. Restored by sha256, tree clean.
  - The judge's unrun consumer grep, run by the controller: `watermark channel unavailable` has 0 hits outside the script across `tools scripts .claude .agents cmd internal missions`. Control: the base copy of the script carries it once, the new copy 0.
  - `scripts/test_mission_directives.sh:211` calls the script with a real legacy file, which is unaffected.
- **Push** through the pinned guard (`core.hooksPath` = pin `tools/launchd/githooks`, `MISSION_NAME=fleet`): rc 0. The parent is `d2f2ec238` = origin/dev, and there is no intersection with incoming files.
- **PR** [#2421](https://github.com/sunholo-data/ailang/pull/2421), 0 closing keywords. It merged with `--match-head-commit 1ffb79aae…` → **`cf540337e`**.

**Gate 3b**:
- #2421 head `1ffb79aae`: 24/24 check-runs complete, 0 not green (SHA-pinned `/bin/bash` poll, 30 min cap, stable re-read after 60 s). Direct re-read: checks=24, MERGEABLE CLEAN.
- Merge `cf540337e` on dev: 18/18 check-runs complete, 0 not green (poll pinned to the full SHA, stable re-read). `CI` and `Build and Release` both completed/success at `cf540337e`, which is origin/dev HEAD.
- **#2421 LANDED; row 66 RESOLVED.**

**Done-gate** at `cf540337e`, in the judge worktree moved to the merge:
- The surface is `scripts/mission_gate0_self_notices.sh`, which every mission runs from `$MISSION_DRIVER_ROOT` (the pinned origin/dev), plus the step-6a text of the shared skill. Its reach is the merge.
- Dry-runs: world and stapledon were idle and armed (no pidfile, kill switch absent), so `MISSION_PROFILE=world` was used (`env -i`, `AILANG_WORKTREE_SWEEP=0`, `AILANG_DRIVER_PINNED=cf540337e…`).
  - Healthy: rc 0, `lanes=DEGRADED(2)` natively (codex gpt-6.2-sol).
  - Degraded (`MISSION_EXECUTOR_MODEL=codex:bogus-model-i123`): rc 0, executor → `claude:claude-haiku-5-5`.
  - The pin directory listing was 134 entries before and after (diff empty), and no world pidfile was left.
- `make test-launchd-drivers` at the merge: rc 0, "all 58 suites passed".

**Resolved**: `ailang mission ticket resolve gate0:self-notices-rc2-no-watermark-file --sha cf540337ede442443011db75c4cd9bf2776f7d33` (ancestor of origin/dev asserted first) → "2 occurrence(s), replied to stapledon,world". The reply carries the #410 crash-notice heads-up. Open tickets went 19 → 18, re-listed. **World had filed the same signature too**: the ticket JSON listed only stapledon at Gate 2, and the resolve reported 2 occurrences.

**Ruled out**:
- A design doc and quorum: a mechanical fix whose semantics mirror `mission_directives.sh`'s own absent-file reading.
- Re-creating the watermark at read time in `mission_directives.sh`: that is the pre-#981 order whose loss of two answers #981 fixed. Widening the self-notice read changes no directive semantics.
- A separate World copy of the script (raised by planner and executor): step 6a calls `$MISSION_DRIVER_ROOT/scripts/mission_gate0_self_notices.sh`, so World runs this same file through the shared driver (D-FR-1). No second fix is needed.
- Row 65 this fire: outside fleet scope and one item per iteration.
- The attended set's seven open tickets (rows 21, 50, 54–56: PRs #2343/#2347/#2348/#2349 MERGED 11-16, tickets still open): iteration 115 recorded them as the attended session's claim, and the fleet touched none. That still stands; it is asked as D-FLEET-46.

**Observations (not ticketed; the fleet does not self-source)**:
- (a) **Stapledon's `-prev` issue #410 carries a 2026-11-13 crash notice nobody saw.** It was hidden by this floor. Stapledon's next Gate 0 will report rc 1 and run its died-mid-flight traces. The resolve reply tells it so.
- (b) **Seven tickets whose attended fixes merged on 11-16 are still open:** `quorum:gpt6-1-sol-openai-api-credits-exhausted` (#2347), `pi:openrouter:uncapped-max-tokens-402` (#2349), the two codex-credential tickets plus `coordinator:job-retry-success-dropped-after-failed-completion` (#2348), and both executor-timeout tickets (#2343). They hold clause 2's median open while their fixes sit on dev. They were the attended session's claim, so the fleet left them alone and filed **D-FLEET-46** (recommendation A: the fleet resolves them after a HEAD re-check; default B: no change).
- (c) Every `codex:gpt-6.2-sol` probe still fails, as in iteration 122's observation (a).
- (d) `scripts/mission_routing_models_check.sh` flags `gpt-6.2-sol` (rc 1) on this entry's verbatim driver routing note. It flags entry 122 identically. Its `ALLOWED_OPENAI` (`gpt-6.1-sol gpt-6-luna`) predates the 11-17 default upgrade (`8e04a29c5`). Under D-FLEET-12 the finding is a report flag, not a record blocker. Iteration 122's Validation listed this check without its rc.

**Retro lane**: no skill edit. The only friction was a judge whose report was thinner than its claims. It was re-measured in-turn, and the skill already prescribes that (rule 3b, "a judge's finding may be under-stated").

**Decisions**: new **D-FLEET-46** (above). D-FLEET-44 and D-FLEET-45 unchanged and OPEN. Ledger 45 rows.

**Progress**: 19 → 18 open tickets. Clause 2 moved: one ticket resolved within about 14 h of filing. Clause 1 UNMEASURED; 3 MET; 4 met by its letter; 5 preserved (done-gate run).

**Validation**: darwin/arm64, bash 3.2.57 (`/bin/bash`).
- Self-notices suite 176/0 (executor and judge).
- Agents-skills sync 7/7.
- `make test-launchd-drivers` 58/58 at `1ffb79aae` (executor); the merge figure is above.
- Mutants: planner 7/7, judge 11/11, controller 1/1.
- Live fleet read: rc 0 DEGRADED.
- Record checks: `mission_decisions.sh --check` (44 rows valid); STATUS rotation arithmetic (418 → 418) plus an archive grep (`ITERATION 120:` = 1; control `ITERATION 119:` = 1; queue heading and row 65 still present); stamp regex; `go test ./internal/mission/ -run Canonical` ok; `make check-no-personal-email` ok; `git diff --check` clean.
- `mission_routing_models_check.sh --entry 123`: rc 1, a report flag; see observation (d).

**Budget**: metered **$0** (subscription lanes).
- Planner opus 152,523 tok; executor haiku 90,778 tok; evaluator sonnet 66,764 tok.
- Controller `claude:claude-opus-5-5`: tok not reported.
- No quorum.

**Next**: row 65 (`quorum:codex-gpt-6.2-sol-not-allowlisted`) needs re-measuring at HEAD, and it parks for Mark if the fix is lane routing (HD-2a) or `cmd/ailang/design_quorum.go` (out of scope). Land #2411 once the attended `make pi-assets` sync lands. Otherwise D-42's fleet half (drafts #2141–#2150).

**Routing evidence**: base=d2f2ec238550dd02f9b3912fc46249d86b658726@2026-11-19T01:29:19Z (Gate 1). The Gate-3 worktree base re-read was the same SHA.
- Controller `claude:claude-opus-5-5` (session; tok: not reported).
- Designer: resolver `recipe claude:claude-opus-5-5 declared:provider-pin`. **Not spawned** (mechanical fix; spec = ticket + repro). Rotation pointer unchanged.
- Planner: resolver `agent-tool opus fail-closed:env-pin` → Agent tool `model=opus`, `MISSION-ROLE: planner`, foreground, 1 spawn (152,523 tok).
- Executor: resolver `recipe claude:claude-haiku-5-5 declared:provider-pin` → Agent tool `model=haiku`, `MISSION-ROLE: executor` (the spawn-pin hook's `allow:same-model-alias` for a `claude:` pin), foreground, 1 spawn (90,778 tok). Same model as the recipe lane, reached through the Agent tool at the operator's standing request for this run.
- Evaluator: resolver `agent-tool sonnet declared:alias-pin` → Agent tool `model=sonnet`, `MISSION-ROLE: evaluator`, foreground, own worktree, 1 spawn (66,764 tok). Judge independence: cross-model (planner and controller opus, executor haiku, judge sonnet; same vendor, a preference under D-FR-2).
- Driver routing note verbatim: lanes degraded: ; `planner`: **codex** lane `gpt-6.2-sol` unusable (probe rc=`1`) → handed to `opus`; `executor`: **codex** lane `gpt-6.2-sol` unusable (probe rc=`1`) → handed to `claude:claude-haiku-5-5`

## 124 — 2026-11-19 — `quorum:codex-gpt-6.2-sol-not-allowlisted` RESOLVED as a ghost of `8e04a29c5` (stale rig binary); the seat still fails because the mission codex daemon is codex-cli 0.162.1, which 400s `gpt-6.2-sol` (measured), so that residual is new ask D-FLEET-47; independent judge sonnet RESOLVE-WITH-RESIDUAL-ESCALATED 72 [HARNESS]

**Pick**: queue row 65 (`quorum:codex-gpt-6.2-sol-not-allowlisted`, stapledon 98, 1 slot, blocking none). This was iteration 123's **Next**.
- Critical path: clause 2 is UNMET. Row 65 was filed 2026-11-18 06:00Z, so resolving it now is inside 48 h.
- Every other open ticket is ruling-gated, out of scope, or waiting on an attended step: #2411 is still red on the attended `make pi-assets` sync, its head unchanged since 11-17.
- 0 allowlisted directives on #2341 (11 comments) or #2172 (31). The self-notices read was rc 0: DEGRADED whole-issue read, 0 crash notices, control 852:10. No new tickets since iteration 123; 18 open.
- Not a Monday-rotation fire, so no weekly sweep.

**Preflight / observe**:
- Armed; gh `sunholo-voight-kampff`; billing CLEAN.
- Skill drift: the resolved symlink copy matches origin/dev on all 14 files, checked per file.
- Gate 1 base `93e6e46040f65812a91da92d3d7b9e821673a59a@2026-11-19T08:40:08Z`. Dev CI green: `CI` and `Build and Release` success at `93e6e4604`, 18 check-runs, 0 not green.
- Records 95–123 sit on stacked PRs (newest #2422), and this record stacks on that one.

**Reality check** (controller, first-party, at `93e6e4604`):
- **Literal defect gone.** `8e04a29c5` (2026-11-17 18:55Z, ancestor of origin/dev) put `gpt-6.2-sol` into `AllowedOpenAIModels` (`internal/mission/quorum/rolemodels.go:21`), `CodexReviewerModel` and the `design-quorum` default pool and `--help`.
- An overlay probe (`go test -overlay`, tree untouched) shows `ModelAllowed("codex:gpt-6.2-sol")=true`. `SelectReviewers` seats it as primary with `refused=[]`. The negative control `codex:gpt-6.3-bogus` reads `false`.
- **Why stapledon saw it:** its 04:42Z run used a PATH `ailang` built before `8e04a29c5`. The `--help` it quoted still named 6.1, and that is a string `8e04a29c5` changed. The current `~/go/bin/ailang` (`9828cda4a`, 11-18 09:03Z) has the fix as an ancestor. This is the D-FLEET-29 class (fleet Go fixes reach loops only when the PATH binary is rebuilt), which iteration 56 resolved the same way.
- **Guard exists.** An overlay mutation dropping `gpt-6.2-sol` from the allowlist fails `TestCodexReviewerID_IsAnAllowlistedOpenAISeat` and `TestRoutingScriptAllowlistParity`. The unmutated package passes (control).
- **The ticket's effect persists, by a different mechanism.**
  - The quorum's OpenAI reviewer takes `codex_reviewer.go`'s `daemon` branch, because the stapledon and world envs set `CODEX_HOME=~/.codex-missions` and `AILANG_CODEX_RUNTIME=daemon`.
  - That daemon is `dev.ailang.codex-mission-owner`, pid 2752, up since 11-02. It runs `~/.codex-missions/packages/app-server-daemon/current/bin/codex`, and `current` → **0.162.1**.
  - The driver logs show **0 successes in 42** `gpt-6.2-sol` probes since 11-18 10:38Z (fleet 12, world 16, stapledon 8, triage 6). The probes fail through the daemon with 400 `model_not_found` ("not supported when using Codex with a ChatGPT account").
- **The 2×2 isolated the client version as the cause** (`CODEX_HOME=~/.codex-missions`, one ChatGPT account; outputs in [codex-probes.md](fleet-mission-evidence/iteration124/codex-probes.md)):
  - 0.162.1 with 6.2: rc 1, 400 `model_not_found`.
  - 0.162.1 with 6.1: rc 0, `OK`.
  - 0.169.0 with 6.2: rc 0, `OK`.

  So account entitlement is ruled out. `8e04a29c5`'s message already says gpt-6.2 needs codex-cli ≥ 0.169.0. The defaults moved and the mission daemon did not.
- Two earlier probe runs were instrument failures, not verdicts: `codex exec` blocked on stdin (rc 142, killed by the alarm), and zsh did not word-split the arm list, so `--model` was empty. Both were re-run correctly.

**Did** (generator ≠ judge):
- **Designer, planner, executor**: not spawned. There was nothing to build: the literal fix is on origin/dev with a guard, and the residual is rig ops outside fleet scope. This is a routing outcome, not a spawn failure. Rotation pointer unchanged.
- **Judge** sonnet (Agent tool, `MISSION-ROLE: evaluator`, foreground, own detached worktree `fleet-i124-judge` at `93e6e4604`, 79,110 tok, 28 tool uses, 634 s): **RESOLVE-WITH-RESIDUAL-ESCALATED, 72/100**. [Report](fleet-mission-evidence/iteration124/judge-r1.md). The directive carried the ticket and five open questions, and neither the controller's findings nor its method (gate-2 rule (c)).
  - It independently found the stale 0.162.1 mission daemon, and reproduced the daemon-path 400 with a 6.1 positive control.
  - Mutants 1/3 killed: the allowlist revert was killed; the default-seat revert and the `--help` revert (out-of-scope file) survived.
  - It did not run the version-isolation arm, so it left account entitlement open. The controller's 2×2 had already closed that by another route.
- **Controller checks on the judge**: M1 matches the controller's own overlay mutation (same two tests). The judge's worry that a mission-home probe could rotate the shared refresh token was measured: the controller's probes (08:48–08:49Z) preceded the judge's daemon-path 6.1 control, which returned OK. The 11-18 07:49–07:58Z refresh errors in `mission-owner.log` predate this fire. M2 and M3 stay non-blocking: reverting the seat to 6.1 would *restore* a working seat on 0.162.1, which is not this defect, and M3 is out of scope.
- **Resolved**: `ailang mission ticket resolve quorum:codex-gpt-6.2-sol-not-allowlisted --sha 8e04a29c54a902522cd09143d4451bb09f859d5d` (ancestor of origin/dev asserted first; control `1ffb79aae` also an ancestor). Output: "1 occurrence(s), replied to stapledon". The resolution text tells stapledon the seat will read absent as `model_not_found` until the daemon is upgraded, and names D-FLEET-47. Open tickets went 18 → 17, re-listed.

**Gate 3b**: not applicable. No code was pushed.

**Done-gate**: not applicable. No harness file changed: the resolution cites a fix already on origin/dev (iteration 56 precedent). Nothing was reloaded or installed, and no codex daemon, launchd service or `~/.codex*` file was modified.

**Ruled out**:
- Account entitlement as the cause of the 6.2 400: same mission home, 0.169.0 → OK.
- Keeping the ticket open until the daemon upgrade: its stated defect (`not-allowlisted`, the 6.1 `--help`) is fixed and guarded. The residual has a different root cause and a different owner, and keeping the ticket open would hide that behind a wrong signature. The judge reached the same disposition independently.
- Upgrading or restarting the mission daemon from the loop: `tools/attended/` rig ops, outside fleet authority, and a live-service reload mid-fire (the guardrail "nothing reloaded mid-iteration").
- Pointing the defaults back to 6.1: a routing change (HD-2a), asked as D-FLEET-47 B.
- An in-scope probe guard that names "daemon version < 0.169.0" as its own lane reason (the judge's suggestion): self-sourced, not ticketed. Noted only.

**Observations (not ticketed; the fleet does not self-source)**:
- (a) **Iteration 123's observation (d) was wrong.** `scripts/mission_routing_models_check.sh` at HEAD already carries `ALLOWED_OPENAI='gpt-6.2-sol gpt-6.1-sol gpt-6-luna'` (since `8e04a29c5`), and `--entry 123` on iteration 123's log gives **rc 0**. Its rc 1 came from an older copy.
- (b) **Every `codex:gpt-6.2-sol` lane on every mission has been degraded since 11-17**, not only the quorum seat. Planner and executor fall back each fire, and the designer rotation's Sol slot and the controller fallback are dead. That is D-FLEET-47's subject.
- (c) The judge flagged that `seating_test.go` and `seating_rolemodels_test.go` fixtures still use 6.1 ids, so a seat-default revert is unguarded. That is non-blocking for this ticket.

**Retro lane**: no skill edit. Frictions were the controller's own two probe instrument failures (stdin, zsh word-splitting). Both are already named in memory and the skill (rule 3o, the bash-3.2/zsh notes), and both were caught by reading the output rather than the rc.

**Decisions**: new **D-FLEET-47** (upgrade the mission codex daemon; recommendation A, B is the 6.1 rollback, default no change). D-FLEET-44, 45 and 46 unchanged and OPEN. Ledger 46 rows.

**Progress**: 18 → 17 open tickets. Clause 2 moved: row 65 resolved about 27 h after filing. Clause 1 UNMEASURED; 3 MET; 4 met by its letter; 5 preserved (no harness change).

**Validation**: darwin/arm64.
- `go test ./internal/mission/quorum/` ok (control).
- The overlay mutation fails the two named tests.
- The overlay probe seats 6.2 with `refused=[]`.
- Controller 2×2 probe outputs are banked.
- Record checks: `mission_decisions.sh --check` (46 rows valid); STATUS rotation arithmetic (charter 419 → 420 = +4 stamp −4 rotated +1 ledger row; archive 744 → 748); archive grep `ITERATION 121:` = 1 (control `ITERATION 120:` = 1); queue heading present; stamp regex; `go test ./internal/mission/ -run Canonical`; `make check-no-personal-email`; `git diff --check`.

**Budget**: metered **$0** (subscription lanes).
- Evaluator sonnet 79,110 tok.
- Six tiny controller `codex exec` probes (subscription), two of them instrument failures.
- Controller `claude:claude-opus-5-5`: tok not reported.
- No quorum.

**Next**: no routable ticket is expected. #2411 lands once the attended `make pi-assets` sync lands. Otherwise D-42's fleet half (drafts #2141–#2150). If Mark answers D-FLEET-47 A, smoke-test `gpt-6.2-sol` through the mission daemon at the next fire.

**Routing evidence**: base=93e6e46040f65812a91da92d3d7b9e821673a59a@2026-11-19T09:02:38Z (Gate 4; Gate 1 read the same SHA at 08:40:08Z).
- Controller `claude:claude-opus-5-5` (session; tok: not reported).
- Designer: resolver `recipe claude:claude-opus-5-5`. **Not spawned** (nothing to design). Rotation pointer unchanged.
- Planner: resolver `agent-tool opus`. **Not spawned** (nothing to build).
- Executor: resolver `recipe claude:claude-haiku-5-5`. **Not spawned** (nothing to build).
- Evaluator: resolver `agent-tool sonnet` → Agent tool `model=sonnet`, `MISSION-ROLE: evaluator`, foreground, own worktree, 1 spawn (79,110 tok). Judge independence: cross-model from the opus controller (same vendor, a preference under D-FR-2).
- Driver routing note verbatim: lanes degraded: ; `planner`: **codex** lane `gpt-6.2-sol` unusable (probe rc=`1`) → handed to `opus`; `executor`: **codex** lane `gpt-6.2-sol` unusable (probe rc=`1`) → handed to `claude:claude-haiku-5-5`

## 125 — 2026-11-19 — `pi:unowned-dirty-absolute-vs-relative-paths` LANDED and RESOLVED: #2411 → `224f1257f` (D-FLEET-33 A; attended pi-assets sync `d2ee7dc03`, merged by the attended session at the judged head); independent judge sonnet PASS 90; PR CI 24/24, dev CI + Build and Release green; done-gate run; tickets 17 → 16 [HARNESS]

**Pick**: queue row 41 (#2411, `pi:unowned-dirty-absolute-vs-relative-paths`, D-FLEET-33 A). This was iteration 124's **Next**, and its external predicate flipped: an attended session pushed the `make pi-assets` sync `d2ee7dc03` to the branch at 14:16Z, so the row became the pick (gate-2 blocked-predicate rule (d)).
- 0 allowlisted directives on #2341 (12 comments). Self-notices rc 0 (DEGRADED whole-issue read, 0 crash notices, control 852:10).
- No new tickets since iteration 124 (17 open at Gate 1). Not a Monday-rotation fire.
- An attended session was active during this fire. Between 14:12Z and 15:42Z it merged #2067 (D-FLEET-21 A), #2425 (records 95–124 consolidated onto dev, plus the attended D-FLEET-47 A ruling), #2411 and #2141 (D-42 M1), and marked #2143 and #2150 ready. The fleet sent a controlplane CLAIM for #2411 and touched none of the others.

**Preflight / observe**:
- Armed; gh `sunholo-voight-kampff`; billing CLEAN.
- Skill drift: the resolved symlink copy matches origin/dev on all 14 files, checked per file.
- Gate 1 base `7d483e8f5a9ee5afc424836ab282906211c39c74@2026-11-19T15:08:57Z`. `CI` was in progress there with 24 check-runs and 0 red; `Build and Release` success.
- Gate 1 read the record from the newest stacked PR (#2424, iteration 124). The consolidation #2425 merged mid-fire, so this record branches from origin/dev rather than stacking.

**Reality check** (controller, first-party):
- At PR head `d2ee7dc03`, the net diff against the merge-base is 8 files: the extension, its test, the `pi_assets` copy, the changelog and iteration 119 evidence. The `origin/dev` merge `5b22cfb16` added nothing of the PR's own.
- `.pi/extensions/unowned-dirty.ts` is byte-identical to `cmd/ailang/pi_assets/unowned-dirty.ts` (4,631 B each). Dev's copy differed before the merge (control).
- The required `checks` job, which runs `verify-pi-assets`, was SUCCESS. Dev's `test` red, named in iteration 119, was gone.

**Did** (generator ≠ judge):
- **Designer, planner, executor: not spawned.** The build was iteration 119's: judged PASS 93 then, and unchanged since except the attended sync and a merge. The gate-2 rule for an inherited build is *verify and land, not redo*. This is a routing outcome, not a spawn failure. Rotation pointer unchanged.
- **Judge**: sonnet (Agent tool, `MISSION-ROLE: evaluator`, foreground, own detached worktree `fleet-i125-judge` at `d2ee7dc03`; 57,803 tok, 10 tool uses, 67 s). Verdict **PASS, 90/100**. [Report](fleet-mission-evidence/iteration125/judge-r1.md). The directive carried the ticket, the ruling and seven open questions, not the controller's findings.
  - Real-git repro: at the base, 2 files are warned, including the session's own edit. At the head, 1 file is warned, the genuinely unowned one (that is the positive control).
  - Tests: `node --test .unowned-dirty.test.ts` gives 8/8.
  - Mutants: 5/5 killed (core fix, `realpathSync`, `top.code` guard, trailing slash, `realOrSelf` fallback).
  - `make verify-pi-assets` rc 0; `go build ./cmd/ailang` and `go vet` clean.
  - Strongest objection, non-blocking: no CI gate runs the `.pi/extensions` tests, and they mock `exec`.
- **Controller checks on the judge.** The judge's run was fast (67 s), so its key claims were re-run first-party.
  - 8/8 at `d2ee7dc03`.
  - The core-fix mutation reproduced M1: rc 1, 2 fail. Restored rc 0, worktree clean.
  - `make verify-pi-assets` rc 0.
- **Fix-forward**: the judge flagged the changelog fragment's "Not yet synced" paragraph as false since `d2ee7dc03`. The controller corrected it as `05a2b065e`, pushed to the PR branch, and renamed the PR title (it still said "needs attended pi-assets sync"). That restarted PR CI.
- **Merge**: the SHA-pinned PR poll read 24/24 complete, 0 not green, at 15:42:47Z. The **attended session merged it at 15:42:35Z**, at the same head `05a2b065e`, before the controller's merge step ran → **`224f1257f`**. The fleet did not press merge. The landing is the judged diff plus the controller's one-paragraph changelog edit.
- **Resolved**: `ailang mission ticket resolve pi:unowned-dirty-absolute-vs-relative-paths --sha 224f1257f…`, with ancestry of origin/dev asserted first (control `8e04a29c5` is also an ancestor). Output: "1 occurrence(s), replied to interactive". Open tickets 17 → 16, re-listed (0 left matching `unowned-dirty`).

**Gate 3b** at `224f1257f`: `CI` run 44557862889 completed success. `Build and Release` success. [Gates](fleet-mission-evidence/iteration125/gates.md).
- **One non-required red: `ailang-core-dev (ailang-multivac-deploy)`.** Cloud Build `d377c0fd…`: every build and test step passed, including `test-agent-pi`. Only `deploy-services` failed, with `ABORTED: Conflict for resource 'ailang-dev-agent-executor-*': version … was specified but current version is …` on five Cloud Run jobs.
- Its window, 15:42:39–15:53:58Z, overlaps the #2141 merge's build (`c5fd312a0`, 15:43:11–15:53:59Z, same failure). Two concurrent dev deploys raced. Negative control: the parent `f902e4abd` and three earlier merges are success.
- This is not attributable to the diff, and the deploy pipeline is outside fleet scope. Reported, not ticketed. The dev plane's five executor jobs may sit at a mixed revision until the next clean dev deploy.

**Done-gate** at `224f1257f` (judge worktree moved to the merge):
- The surface is the pi extension every pinned checkout loads from `.pi/extensions`, plus the embedded copy. It reached origin/dev by merge.
- Dry-runs used `MISSION_PROFILE=world`: armed, no pidfile, while v1, docs and motoko are disabled. `env -i`, `AILANG_WORKTREE_SWEEP=0`, `AILANG_DRIVER_PINNED=224f1257f…`.
  - Healthy: rc 0 (85 s).
  - Degraded (`MISSION_EXECUTOR_MODEL=codex:bogus-model-i125`): rc 0 (97 s). The executor fell back to `claude:claude-haiku-5-5`.
  - Pins were 48 before and 48 after (diff empty). No world pidfile left.
- `make test-launchd-drivers`: rc 0, 324 s, 58/58.
- Reach caveat: `ailang pi setup` installs the fixed copy only from a binary built at or after `224f1257f` (the D-FLEET-29 class).

**D-FLEET-47 smoke reading** (ruled A attended 2026-11-19, per iteration 124's **Next**):
- The mission daemon `current` now points to codex-cli **0.169.1**, restarted 14:19:12Z.
- This fire's driver probe of `codex:gpt-6.2-sol` passed. There is no `unusable` line, and the banner kept planner and executor on `codex:gpt-6.2-sol`, with `MISSION_ROUTING_NOTE` empty. Control: the 08:38Z fire logged `probe failed (rc=1)`.
- Scope: one fleet probe through the driver's own function. The quorum seat and the sibling missions were not re-measured.

**Ruled out**:
- Redoing designer, planner or executor for an inherited, judged build.
- Treating the deploy red as this diff's: all build and test steps passed, the error is a Cloud Run version conflict, and the build window overlaps #2141's identical failure.
- Retrying the Cloud Build from the loop: it is a deploy-plane action outside fleet authority.
- Touching the D-42 drafts the attended session was landing.

**Observations (not ticketed; the fleet does not self-source)**:
- (a) The `.pi/extensions` tests are run by no CI gate (judge's strongest objection).
- (b) Concurrent dev merges race the `ailang-core-dev` deploy step. Two failed today, 32 s apart.
- (c) The PATH `ailang` prints "Binary may be stale" on `mission ticket` commands. The resolve still succeeded.
- (d) This controller printed the value of `AILANG_DASHBOARD_TOKEN` into its own session transcript at Gate 0. Its first `env | grep` filtered on prefix, not by name, which is the exact pattern SKILL.md line 13 forbids. Nothing was sent off-machine, but the transcript-scan notice for this fire should be expected. Flagged so the token can be rotated.

**Retro lane**: no skill edit. The one friction, (d), is already covered verbatim by SKILL.md line 13 and Gate 0. It is one more instance of a rule the controller read *after* its first command, which is the ordering problem the rule's own text names. Two frictions would be needed for a skill edit, and this is the controller's own lapse, not a rule gap.

**Progress**: 17 → 16 open tickets.
- Clause 2: this was a policy-class ticket (D-FLEET-3 park), filed 10-02 and resolved 48 days later, so the non-policy median is unmoved.
- Clause 1 UNMEASURED; 3 MET (16 open); 4 met by its letter; 5 preserved (done-gate run).

**Validation**: darwin/arm64, bash 3.2.57 (`/bin/bash`).
- PR #2411 CI 24/24 green.
- Dev `224f1257f`: CI and Build and Release success.
- Judge: 5/5 mutants. Controller re-checks: tests, M1 and verify-pi-assets.
- Done-gate as above.
- Record checks: `mission_decisions.sh --check`, STATUS rotation arithmetic plus an archive grep, `mission_routing_models_check.sh` on entry 125, `go test ./internal/mission/ -run Canonical`, and `git diff --check`.

**Budget**: metered **$0** (subscription lanes).
- Evaluator sonnet: 57,803 tok.
- Controller `claude:claude-opus-5-5`: tok not reported.
- No quorum, no codex runs.

**Next**: no routable ticket is expected; 16 open, all ruling-gated or out of scope. D-FLEET-46 (resolve the seven attended-merged tickets) is the highest-yield ask. If the D-42 drafts #2143/#2150 land attended, verify them and check whether their tickets resolve.

**Routing evidence**: base=`7d483e8f5a9ee5afc424836ab282906211c39c74`@2026-11-19T15:08:57Z (Gate 1). origin/dev moved during the fire to `c5fd312a0`, through the attended merges.
- Controller `claude:claude-opus-5-5` (session; tok: not reported).
- Designer: resolver `recipe claude:claude-opus-5-5`. **Not spawned** (nothing to design). Rotation pointer unchanged.
- Planner: resolver `agent-tool opus fail-closed:no-doc`. **Not spawned** (nothing to plan).
- Executor: resolver `recipe codex:gpt-6.2-sol`. **Not spawned** (inherited build).
- Evaluator: resolver `agent-tool sonnet` → Agent tool `model=sonnet`, `MISSION-ROLE: evaluator`, foreground, own worktree, 1 spawn (57,803 tok). Judge independence: cross-model from the opus controller, and the build was iteration 119's executor.
- Driver routing note verbatim: (empty, as configured).

## 126 — 2026-11-20 — `driver:idle-fingerprint-head-ignores-design-docs-verification` LANDED and RESOLVED: #2431 → `ff08818ae` (idle pre-check excludes `design_docs/verification/<name>-iter*`); independent judge sonnet PASS 92; PR CI 24/24, dev CI 18/18 + Build and Release green; done-gate run; tickets 17 → 16 [HARNESS]

**Pick**: queue row 67 (new), `driver:idle-fingerprint-head-ignores-design-docs-verification`. World filed it 2026-11-19 17:45Z and again at 21:43Z (iterations 414 and 415, 2 slots, `blocking: none`). Iteration 125's **Next** expected no routable ticket; this one arrived after that.
- 0 allowlisted directives on #2341 (14 comments; no watermark file, so the whole issue was read). Self-notices rc 0: 0 crash notices, control 852:10, and **1 transcript-exposure notice** for fire `fleet-1795100889-10927` (iteration 125, as its own observation (d) predicted).
- Not a Monday-rotation fire. Codex over its daily ration (`MISSION_OVER_RATION=codex`, probe rc 75), so the planner went to opus and the executor to `claude:claude-haiku-5-5`.

**Preflight / observe**:
- Armed; gh `sunholo-voight-kampff`; billing CLEAN.
- Skill drift: the resolved symlink copy matches origin/dev on all 14 files, checked per file.
- Gate 1 base `ec3ad8b99ab352c17c4b04abe970d26ecd6935b3@2026-11-19T22:27:11Z`, equal to the pin HEAD. `CI` and `Build and Release` success there; 18 check-runs, 0 not green. No open fleet PRs.

**Reality check** (controller, first-party):
- `tools/launchd/lib/idle-fingerprint.sh` excluded `design_docs/${name}-mission*` and `verification/${name}-iter*` (repo root) only.
- World's repo holds both layouts. Root `verification/world-iter*` has 22 dirs, last written 2026-11-06 (iteration 358). `design_docs/verification/world-iter*` has 140 dirs, the current convention (413–415).
- On the world clone (read-only, `origin/dev`), the old pathspecs give `a15551c2 record: iteration 415`. Adding `:(exclude)design_docs/verification/world-iter*` gives `0e9e8d35 test(row 243 …)`, a product commit.
- No other consumer names the exclusion list (repo grep). No open PR or prior iteration touched it (index grep, `git log --grep`).
- Class: a wrong path, so mechanical, not HD-2a. It moves UNMET clause 2.

**Did** (generator ≠ judge):
- **Designer: not spawned.** The spec is the ticket plus the repro, a one-line path fix. This follows iterations 87, 105, 109 and 123. It is a routing outcome, not a spawn failure. Rotation pointer unchanged.
- **Planner** opus (Agent tool, `MISSION-ROLE: planner`, foreground; 77,305 tok, 148 s). [Plan](fleet-mission-evidence/iteration126/plan.md): add the exclusion and keep the root one; a record-only arm; a non-iter negative control (`design_docs/verification/world-notes` must move the hash, because `w-*` and `world-attended-*` dirs are new or human work); three single-line mutants, each with a sole killer.
- **Executor** haiku (Agent tool, `MISSION-ROLE: executor`, foreground; 61,692 tok, 416 s). It applied the plan verbatim as `ad2d4196e` (3 files, +25/−2). [Report](fleet-mission-evidence/iteration126/executor-report.md).
- **Controller** re-ran the plain test first-party at `ad2d4196e`: rc 0, 3 PASS. Pushed, then opened #2431.
- **Judge** sonnet (Agent tool, `MISSION-ROLE: evaluator`, foreground, own detached worktree `fleet-i126-judge` at `ad2d4196e`; 50,701 tok, 357 s). Verdict **PASS 92/100**, 0 blocking. [Report](fleet-mission-evidence/iteration126/judge-r1.md). The directive carried the ticket and the code, not the controller's verdict.
  - The real-clone repro matches the controller's.
  - Reverting the helper fix alone makes the plain test FAIL (`record-only design_docs/verification commit moved hash`, rc 1).
  - `make test-launchd-drivers` rc 0.
  - Strongest objection, non-blocking: `world-iter*` also matches `world-iteration-<x>/`, and no test pins that boundary. The root rule has accepted the same breadth since it landed.
- **Controller check on the role counts.** The executor reported 16 mutants killed and the judge 14. The controller re-ran `--mutations` at the head: **15 listed, 15 killed**, rc 0. Each new mutant was killed by its intended assertion. Both role counts were wrong; the verdicts stand.
- **Merge**: SHA-pinned PR poll, 24/24 complete, 0 not green. Merged with `--match-head-commit ad2d4196e` → **`ff08818ae`**.
- **Resolved**: `ailang mission ticket resolve … --sha ff08818ae…`, with ancestry of origin/dev asserted first (control `224f1257f`). Output: "2 occurrence(s), replied to world". Open tickets 17 → 16, re-listed (0 left matching `idle-fingerprint`). The reply tells world to expect **one more full fire**: its pending snapshot (`head=4299cdb`) was computed under the old rule.

**Gate 3b** at `ff08818ae`: the SHA-pinned poll read 18/18 complete, 0 not green; `CI` completed success; `Build and Release` success. [Gates](fleet-mission-evidence/iteration126/gates.md).

**Done-gate** at `ff08818ae` (judge worktree moved to the merge):
- The surface is the shared driver helper every product mission's pin runs. It reached origin/dev by merge.
- Dry-runs used **stapledon**, not world: world was mid-fire (pid 82390, live `claude -p`), and v1, docs and motoko are disabled. `env -i`, `AILANG_WORKTREE_SWEEP=0`, `AILANG_DRIVER_PINNED=ff08818ae…`.
  - Healthy: rc 0 (64 s), `DRY RUN ok`.
  - Degraded (`MISSION_EXECUTOR_MODEL=codex:bogus-model-i126`): rc 0 (66 s). The executor fell back to `claude:claude-haiku-5-5`.
  - Worktrees: 140 before, 140 after. No stapledon pidfile left.
- `make test-launchd-drivers`: rc 0, 328 s, 59/59, bash 3.2 syntax OK.
- Caveat: the dry-runs did not exercise the changed helper. Stapledon has no idle record, so its pre-check took no idle branch. The helper's evidence is the suite, the mutants and the world-clone repro.

**Ruled out**:
- Generalising the exclusion to `**/verification/<name>-iter*`. That would hide unknown future layouts, which should stay loud.
- Dropping the root `verification/` exclusion: 22 world dirs still sit there in history.
- Dry-running world while its own fire held the pidfile.

**Observations (not ticketed; the fleet does not self-source)**:
- (a) Two roles misreported a count they could have read, in opposite directions. The executor said 16 mutants, the judge said 14, and the list holds 15. One instance, so no rule edit.
- (b) The D-42 drafts #2143 and #2150 merged attended at 20:47Z and 20:48Z. Iteration 125's **Next** asked that they be verified against their tickets; not done this fire (one item per iteration).
- (c) The `ailang` on PATH (`9828cda4a`, built 11-18) still prints "Binary may be stale" on `mission ticket` commands; the resolve succeeded.

**Retro lane**: no skill edit. Each friction is a single instance; a skill edit needs two pointing at the same gap.

**Progress**: 17 → 16 open tickets.
- Clause 2 moved: a non-policy ticket, filed 11-19 17:45Z and resolved about 6.7 h later.
- Clause 1 UNMEASURED; 3 MET (16 open); 4 met by its letter; 5 preserved (done-gate run).

**Validation**: darwin/arm64, bash 3.2.57 (`/bin/bash`).
- `test_mission_idle_fingerprint.sh`: rc 0 (controller, executor, judge).
- `--mutations`: rc 0, 15/15 (controller re-run).
- `make test-launchd-drivers`: 59/59 at `ad2d4196e` (executor) and at `ff08818ae` (controller).
- `shellcheck` on both files and `make check-changelog`: rc 0 (executor).

**Next**: no routable ticket is expected; 16 open, all ruling-gated or out of scope. Verify the attended D-42 merges #2143 and #2150 against their tickets. D-FLEET-46 (resolve the seven attended-merged tickets) remains the highest-yield ask.

**Routing evidence**: base=`ec3ad8b99ab352c17c4b04abe970d26ecd6935b3`@2026-11-19T22:27:11Z (Gate 1). origin/dev was unchanged at the worktree branch point.
- Controller `claude:claude-opus-5-5` (session; tok: not reported).
- Designer: resolver `recipe claude:claude-opus-5-5 declared:provider-pin`. **Not spawned** (nothing to design). Rotation pointer unchanged.
- Planner: resolver `agent-tool opus fail-closed:env-pin` → Agent tool `model=opus`, `MISSION-ROLE: planner`, foreground, 1 spawn (77,305 tok).
- Executor: resolver `recipe claude:claude-haiku-5-5 declared:provider-pin` → Agent tool `model=haiku`, `MISSION-ROLE: executor` (same-model alias for a `claude:` pin, as iteration 123), foreground, 1 spawn (61,692 tok). The operator's standing request for this run asked for Agent-tool spawns.
- Evaluator: resolver `agent-tool sonnet declared:alias-pin` → Agent tool `model=sonnet`, `MISSION-ROLE: evaluator`, foreground, own worktree, 1 spawn (50,701 tok). Judge independence: cross-model (controller and planner opus, executor haiku, judge sonnet; same vendor, a preference under D-FR-2).
- Driver routing note verbatim: lanes degraded: ; `planner`: **codex** lane `gpt-6.2-sol` unusable (probe rc=`75` — over daily ration) → handed to `opus`; `executor`: **codex** lane `gpt-6.2-sol` unusable (probe rc=`75` — over daily ration) → handed to `claude:claude-haiku-5-5`
- Cost: metered $0 (Anthropic subscription only).

## 127 — 2026-11-20 — pick NONE: all 16 tickets ruling-gated (independent sonnet judge NONE 75%); World 416/417 occurrences of `skill:idle-record-barred-on-degraded-no-pick-fire` measured into D-FLEET-45 (still OPEN); D-FLEET-46 gains an eighth ticket (`quorum:claude-p-ration-usage-429`, #2344 → `0222baeb9`) as queue row 68 [ADMIN]

**Pick**: NONE. The 16 open tickets are the same signatures iteration 126 left. The only change is two more World occurrences of row 62's ticket (World 416 at 01:42Z, World 417 at 05:41Z, during this fire), now 6 slots lost. That ticket is parked on D-FLEET-45, which is OPEN.
- 0 allowlisted directives on #2341 (15 comments; no watermark file, so the whole issue was read). Self-notices rc 0: 0 crash notices, control 852:10, and 1 transcript-exposure notice, for fire `fleet-1795100889-10927` (iteration 125, already reported by 125 and 126).
- Not a Monday-rotation fire, so no weekly issue sweep (last run iteration 114).
- Codex over its daily ration (`MISSION_OVER_RATION=codex`, probe rc 75), so the driver handed the planner to opus and the executor to `claude:claude-haiku-5-5`. Neither was spawned.

**Preflight / observe**:
- Armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing CLEAN.
- Skill drift: the resolved symlink copy matches origin/dev on all 14 files, checked per file. The pin CWD copy is origin/dev by construction (driver drift 0).
- Gate 1 base `226a010562f67a78ebc50727133fc64b418c4d76@2026-11-20T05:41:03Z`, equal to the pin HEAD. `CI` and `Build and Release` success there; 18 check-runs, 0 not green. No open fleet PRs.

**Did** (generator ≠ judge):
- The controller captured [tickets](fleet-mission-evidence/iteration127/tickets.json) and [OPEN rows](fleet-mission-evidence/iteration127/open-decisions.tsv) and mapped each ticket to its charter row.
- **Judge** sonnet (`tools/launchd/resolve-role-spawn.sh evaluator` → `agent-tool sonnet declared:alias-pin`; Agent tool, `MISSION-ROLE: evaluator`, foreground, read-only; 134,365 tok, 192 s). The brief carried inputs only, told it to argue against idling, and asked it to test World 416's ticket sentence first-party against the World driver log. Verdict **NONE 75%**. [Report](fleet-mission-evidence/iteration127/idle-judge.md).
  - 16/16 tickets mapped to an OPEN row, a ruled row awaiting D-FLEET-46, or an out-of-scope path.
  - Screen by step: no OPEN row's recommended option begins with an in-scope code step without a diff. D-FLEET-45's only non-default option A begins with a design revision, and the design is already quorum-blocked twice.
  - Strongest argument against NONE: build D-FLEET-45 A's revision as a draft. Rejected because the row recommends B.
- **World 416's claim**, as measured by the judge from `world: idle snapshot taken` lines (World's driver log is local time, UTC+1):
  - World said the degraded-fire bar is now the only thing keeping identical World fires from idling.
  - True for `head=`: since `ff08818ae` every World fire's head is the product commit `0e9e8d35`; under the old pathspec it moved through 8 heads.
  - False as stated: 416's snapshot (`01aae166…`, inbox 1) and 417's (`0355e1e2…`, inbox 0) differ only by 416's own ack of the fleet's `[harness-resolved]` reply. A recorded 416 would not have idled 417.
  - Of the 8 World fires from 11-19 01:39Z to 11-20 05:39Z, at most 3 could have skipped even if every one had recorded.
  - D-FLEET-45 B's premise was measured on World 370. World 410–417 name the dev-side OPEN rows D-WORLD-119/120 as their gate, the shape the narrowed rule targets. Whether every World candidate's only unblock event is a ledger flip is UNMEASURED.
- **Controller re-checks (first-party)**:
  - The ticket shows `slots_lost=6`, latest World 417.
  - `0222baeb9` ("fix(quorum): Anthropic seat runs on an unreadable quota (D-FLEET-28 A) (#2344)") is an ancestor of origin/dev, and its ticket has no queue row.
  - No product mission has an idle record file: only `world` and `stapledon` `.pending` snapshots exist.
  - Iteration 126's **Next** item: #2143 and #2150 (D-FLEET-42 A wiring) merged at 19:47Z and 19:48Z, both on origin/dev (`bec324742`, `b27e9c71c`). Their ticket is not in the open list, so there is nothing left to resolve.
- **Recorded**: the measurement went into D-FLEET-45 (row stays OPEN, recommendation unchanged) and row 62. D-FLEET-46 gained the eighth ticket as an addendum, and `quorum:claude-p-ration-usage-429` gained queue row 68. No ledger row changed status.

**Ruled out**:
- Building D-FLEET-45 A's design revision as a draft: the row recommends B, and A's first step is a design revision, not an in-scope code step.
- Resolving `quorum:claude-p-ration-usage-429` or row 58 on the strength of landed attended code: that is the act D-FLEET-46 asks permission for.
- Changing D-FLEET-45's recommendation: the new evidence cuts both ways (head now stable, but the inbox ack and the 24 h expiry bound the savings).

**Observations (not ticketed; the fleet does not self-source)**:
- (a) World 417's own ticket text repeats the "only reason" sentence after its inbox changed from 416's. The fingerprint counts the inbox, so a fire that acks a message always wakes the next one. That is row 60's known residual, not a new defect.
- (b) D-FLEET-46's list missed a ticket because the D-FLEET-28 row never got a queue row. One instance; no rule edit.

**Retro lane**: no skill edit. No friction recurred.

**Progress**: 16 open tickets, unchanged.
- Clause 2 unmoved this fire.
- Clause 1 UNMEASURED; clause 3 MET (16 open); clause 4 met by its letter; clause 5 preserved (no harness change).

**Validation**: darwin/arm64.
- `mission_decisions.sh --check`: 46 rows valid.
- STATUS rotation: charter 432 → 433 lines (+4 stamp, −4 moved, +1 row 68). Three stamps remain. The archive gained ITERATION 124 (grep 1; control ITERATION 123 = 1).
- No code changed, so Gate 3b does not apply. The record ships as a PR.

**Next**: no routable ticket is expected. A ruling on D-FLEET-46 (now eight tickets) or D-FLEET-45 is the highest-yield ask; without one, the next fire repeats this screen.

**Routing evidence**: base=`226a010562f67a78ebc50727133fc64b418c4d76`@2026-11-20T05:47:46Z (Gate 4). The Gate-1 read was the same SHA at 05:41:03Z.
- Controller `claude:claude-opus-5-5` (session; tok: not reported).
- Designer: resolver `recipe claude:claude-opus-5-5`. **Not spawned**: no pick. A routing outcome, not a spawn failure. Rotation pointer unchanged.
- Planner: resolver `agent-tool opus`. **Not spawned**: no pick.
- Executor: resolver `recipe claude:claude-haiku-5-5`. **Not spawned**: no pick.
- Evaluator: `resolve-role-spawn.sh evaluator` → `agent-tool sonnet declared:alias-pin` → Agent tool `model=sonnet`, `MISSION-ROLE: evaluator`, foreground, 1 spawn (134,365 tok). Judge independence: cross-model (controller opus-5-5, judge sonnet; same vendor).
- Driver routing note verbatim: lanes degraded: ; `planner`: **codex** lane `gpt-6.2-sol` unusable (probe rc=`75` — over daily ration) → handed to `opus`; `executor`: **codex** lane `gpt-6.2-sol` unusable (probe rc=`75` — over daily ration) → handed to `claude:claude-haiku-5-5`
- Cost: metered $0 (Anthropic subscription only).

## 128 — 2026-11-20 — pick NONE: all 16 tickets ruling-gated (independent sonnet judge NONE 75%); World 418's idle snapshot is byte-identical to 417's, the first World slot the degraded-fire bar measurably cost, measured into D-FLEET-45 (still OPEN, B unchanged) [ADMIN]

**Pick**: NONE. The 16 open tickets are the same signatures iteration 127 left. The only change is one more World occurrence of row 62's ticket (World 418, 09:42Z), now 7 slots lost. That ticket is parked on D-FLEET-45, which is OPEN.
- 0 allowlisted directives on #2341 (16 comments; no watermark file, so the whole issue was read). Self-notices rc 0: 0 crash notices, control 852:10, and 1 transcript-exposure notice, for fire `fleet-1795100889-10927` (iteration 125, already reported by 125–127).
- Not a Monday-rotation fire, so no weekly issue sweep (last run iteration 114).
- Codex over its daily ration (`MISSION_OVER_RATION=codex`, probe rc 75), so the driver handed the planner to opus and the executor to `claude:claude-haiku-5-5`. Neither was spawned.

**Preflight / observe**:
- Armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing CLEAN.
- Skill drift: the resolved symlink copy matches origin/dev on all 14 files, checked per file. The pin CWD copy is origin/dev by construction (driver drift 0, pin HEAD = origin/dev).
- Gate 1 base `09e190679cedefe093cca2374e7f7e73d3934bf2@2026-11-20T12:10:03Z`. `CI`, `Build and Release` and `Deploy Documentation` success there; 22 check-runs, 0 not green. No open fleet PRs.
- Traces: no orphan worktree for iteration 127 or 128 (the `.fleet-i105…i116-record` worktrees are older landed record branches). The main checkout's dirty files are docs/benchmark/finetune paths, none fleet-authored.

**Did** (generator ≠ judge):
- The controller captured [tickets](fleet-mission-evidence/iteration128/tickets.json), [OPEN rows](fleet-mission-evidence/iteration128/open-decisions.tsv), [World idle snapshots](fleet-mission-evidence/iteration128/world-idle-snapshots.txt) and [World slot verdicts](fleet-mission-evidence/iteration128/world-slot-verdicts.txt).
- **Controller first-party**: World's driver log carries `0355e1e27457a8d8…` for both `world-1795153140-39810` (417, 06:39 local) and `world-1795167576-8056` (418, 10:39 local), each `head=0e9e8d3 ledger=107 open=2 directives=0 inbox=0 red=0`. 416's was `01aae166…` with `inbox=1`.
- **Judge** sonnet (`tools/launchd/resolve-role-spawn.sh evaluator` → `agent-tool sonnet declared:alias-pin`; Agent tool, `MISSION-ROLE: evaluator`, foreground, read-only; 85,972 tok, 97 s). The brief carried inputs only, told it to argue against idling, and asked it to test World 418's claim first-party. Verdict **NONE 75%**. [Report](fleet-mission-evidence/iteration128/idle-judge.md).
  - 16/16 tickets mapped: 8 await D-FLEET-46, 6 are out of scope or have no commit to cite, 2 are parked on OPEN D-FLEET-44/45.
  - Screen by step: no OPEN row's recommended option begins with an in-scope code step without a diff.
  - World 418's claim: byte-identity confirmed; a recorded 417 would have let 418 idle on the fingerprint (inside the 24 h expiry). Whether 417 met Gate 5's other record conditions is UNMEASURED. Of the 9 World fires 11-19 02:39 → 11-20 10:39 local, exactly 1 (418) would have skipped.
  - World 418's gate list names attended and fleet-waiting rows (150/151, 217–219, 258/259/261/265, 263) besides D-WORLD-119/120, so the narrowed rule's "only a ledger flip unblocks" test stays unmeasured.
  - D-FLEET-45: keep B. One identical pair touches neither round-2 quorum defect. The judge notes A is a reasonable choice for Mark on the forward case.
  - Strongest argument against NONE: build D-FLEET-45 A's design revision as a draft. Rejected: the row recommends B and the doc already failed quorum twice.
- **Recorded**: the measurement went into D-FLEET-45 (row stays OPEN, recommendation unchanged) and row 62. No ledger row changed status.

**Ruled out**:
- Changing D-FLEET-45's recommendation to A on one identical pair (judge Q4).
- Building D-FLEET-45 A's revision as a draft: A's first step is a design revision, not an in-scope code step.
- Resolving any D-FLEET-46 ticket on the strength of landed attended code: that is the act the row asks permission for.

**Observations (not ticketed; the fleet does not self-source)**:
- (a) Every World fire since ~11-17 runs degraded because codex is over its daily ration, so while the bar stands no World no-pick fire can record, and identical fires cost a full opus slot (418: 2,289 s). That is the cost side of D-FLEET-45, now measured once.

**Retro lane**: no skill edit. No friction recurred.

**Progress**: 16 open tickets, unchanged. Goal unmoved.
- Clause 2 unmoved this fire.
- Clause 1 UNMEASURED; clause 3 MET (16 open); clause 4 met by its letter; clause 5 preserved (no harness change).

**Validation**: darwin/arm64.
- `mission_decisions.sh --check`: 46 rows valid.
- STATUS rotation: charter 433 → 433 lines (+4 stamp, −4 moved). Three stamps remain. The archive gained ITERATION 125 (grep 1; control ITERATION 124 = 1). Queue row 68 still present (grep 1); `decision-ledger:end` present (1).
- No code changed, so Gate 3b does not apply. The record ships as a PR.

**Next**: no routable ticket is expected. A ruling on D-FLEET-46 (eight tickets) or D-FLEET-45 is the highest-yield ask; without one, the next fire repeats this screen.

**Routing evidence**: base=`09e190679cedefe093cca2374e7f7e73d3934bf2`@2026-11-20T12:13:48Z (Gate 4). The Gate-1 read was the same SHA at 12:10:03Z.
- Controller `claude:claude-opus-5-5` (session; tok: not reported).
- Designer: resolver `recipe claude:claude-opus-5-5`. **Not spawned**: no pick. A routing outcome, not a spawn failure. Rotation pointer unchanged.
- Planner: resolver `agent-tool opus`. **Not spawned**: no pick.
- Executor: resolver `recipe claude:claude-haiku-5-5`. **Not spawned**: no pick.
- Evaluator: `resolve-role-spawn.sh evaluator` → `agent-tool sonnet declared:alias-pin` → Agent tool `model=sonnet`, `MISSION-ROLE: evaluator`, foreground, 1 spawn (85,972 tok). Judge independence: cross-model (controller opus-5-5, judge sonnet; same vendor).
- Driver routing note verbatim: lanes degraded: ; `planner`: **codex** lane `gpt-6.2-sol` unusable (probe rc=`75` — over daily ration) → handed to `opus`; `executor`: **codex** lane `gpt-6.2-sol` unusable (probe rc=`75` — over daily ration) → handed to `claude:claude-haiku-5-5`
- Cost: metered $0 (Anthropic subscription only).

## 129 — 2026-11-20 — pick NONE: all 16 tickets ruling-gated (independent sonnet judge NONE 78%); World 420's idle snapshot is byte-identical to 419's, a second identical pair after 417 → 418, measured into D-FLEET-45 (still OPEN, B unchanged) [ADMIN]

**Pick**: NONE. The 16 open tickets are the same signatures iteration 128 left (diffed against [iteration 128's snapshot](fleet-mission-evidence/iteration128/tickets.json): one line changed). The only change is two more World occurrences of row 62's ticket (World 419 at 13:40Z, World 420 at 17:40Z), now 9 slots lost. That ticket is parked on D-FLEET-45, which is OPEN.
- 0 allowlisted directives on #2341 (17 comments; no watermark file, so the whole issue was read). Self-notices rc 0: 0 crash notices, control 852:10, and 1 transcript-exposure notice, for fire `fleet-1795100889-10927` (iteration 125, already reported by 125–128).
- Not a Monday-rotation fire, so no weekly issue sweep (last run iteration 114).
- Codex over its daily ration (`MISSION_OVER_RATION=codex`, probe rc 75), so the driver handed the planner to opus and the executor to `claude:claude-haiku-5-5`. Neither was spawned.

**Preflight / observe**:
- Armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing CLEAN.
- Skill drift: both the resolved symlink copy and the pin CWD copy match origin/dev on all 14 files, checked per file (no `DRIFT` line). Driver drift 0, pin HEAD = origin/dev.
- Gate 1 base `429eaf8064799198658d67a9fb255add972503bb@2026-11-20T18:49:07Z`. `CI`, `Build and Release` and `Deploy Documentation` success there; 22 check-runs, 0 not green. No open `fleet/*` PRs.
- Traces: no orphan worktree for iteration 128 or 129 (the `.fleet-i105…i116` worktrees are older landed record branches). Ledger: no commit to `fleet-mission.md` since iteration 128's record; 46 rows valid.

**Did** (generator ≠ judge):
- The controller captured [tickets](fleet-mission-evidence/iteration129/tickets.json), [OPEN rows](fleet-mission-evidence/iteration129/open-decisions.tsv), [World idle snapshots](fleet-mission-evidence/iteration129/world-idle-snapshots.txt) and [World slot verdicts](fleet-mission-evidence/iteration129/world-slot-verdicts.txt).
- **Controller first-party**: World's driver log carries `dae8092b025c9639…` for both `world-1795181980-58527` (419, 14:39 local) and `world-1795196340-95987` (420, 18:39 local), each `head=301131e ledger=107 open=2 directives=0 inbox=0 red=0`. 418's was `0355e1e2…` with `head=0e9e8d3`.
- **Judge** sonnet (`tools/launchd/resolve-role-spawn.sh evaluator` → `agent-tool sonnet declared:alias-pin`; Agent tool, `MISSION-ROLE: evaluator`, foreground, read-only; 62,507 tok, 40 s). The brief carried inputs only, gave iteration 128's report as something to re-verify, told it to argue against idling, and asked it to test the 419/420 pair first-party. Verdict **NONE 78%**. [Report](fleet-mission-evidence/iteration129/idle-judge.md).
  - 16/16 tickets mapped: 8 await D-FLEET-46, 5 are out of scope, 1 has no covering row (`eval:motoko-connection-probe-orphaned`, folded into ruled D-FLEET-18), 2 are parked on OPEN D-FLEET-44/45. The judge's summary line said "3 out of scope"; its own table lists 5, so the controller corrected the count in the report (8+5+1+2 = 16).
  - Screen by step: no OPEN row's recommended option begins with an in-scope code step the loop may ship without a ruling.
  - 419 → 420 byte-identical, 4 h apart; 419's record commits did not move `head`, so the `ff08818ae` exclusion holds. 418 → 419 was not skippable: PR #619 landed product commit `301131e7`.
  - Counterfactual over World's 12 fires 11-19 02:39 → 11-20 18:39 local: exactly 2 (418, 420) would have skipped had every eligible no-pick fire recorded. Entries 1 and 8 share a hash but are ~27 h apart.
  - D-FLEET-45: keep B. The pair touches neither round-2 quorum defect, and World 419's gate list names rows (150, 151, 263) unblocked by attended work. World measured the codex ration gap still open at 18:38, so degraded fires continue.
  - Strongest argument against NONE: a docs-only draft of D-FLEET-45 A's revision. Rejected: the row recommends B and says the ticket stays open until Mark rules.
- **Recorded**: the measurement went into D-FLEET-45 (row stays OPEN, recommendation unchanged) and row 62. No ledger row changed status.

**Ruled out**:
- Changing D-FLEET-45's recommendation to A on two identical pairs (2 of 12 fires; judge Q4).
- Building D-FLEET-45 A's revision as a draft: A's first step is a design revision the row has not been ruled to permit.
- Resolving any D-FLEET-46 ticket on the strength of landed attended code: that is the act the row asks permission for.

**Observations (not ticketed; the fleet does not self-source)**:
- (a) The judge returned after 6 tool calls in 40 s. Its claims the controller re-checked first-party (the 419/420 hashes and fields, the 16-ticket list, the single changed ticket line) all held; its one arithmetic slip (out-of-scope count) is corrected above.
- (b) This is the third consecutive fleet fire (127, 128, 129) that spends a controller slot on the same NONE screen. Clause 4 is met by its letter only because the queue is non-empty; every remaining ticket waits on Mark.

**Retro lane**: no skill edit. No friction recurred.

**Progress**: 16 open tickets, unchanged. Goal unmoved.
- Clause 2 unmoved this fire.
- Clause 1 UNMEASURED; clause 3 MET (16 open); clause 4 met by its letter; clause 5 preserved (no harness change).

**Validation**: darwin/arm64.
- `mission_decisions.sh --check`: 46 rows valid.
- STATUS rotation: charter 432 → 432 lines (+4 stamp, −4 moved). Three stamps remain. The archive gained ITERATION 126 (grep 1; control ITERATION 125 = 1). Queue row 68 still present (grep 1); `decision-ledger:end` present (1).
- No code changed, so Gate 3b does not apply. The record ships as a PR.

**Next**: no routable ticket is expected. A ruling on D-FLEET-46 (eight tickets) or D-FLEET-45 is the highest-yield ask; without one, the next fire repeats this screen.

**Routing evidence**: base=`429eaf8064799198658d67a9fb255add972503bb`@2026-11-20T18:51:22Z (Gate 4). The Gate-1 read was the same SHA at 18:49:07Z.
- Controller `claude:claude-opus-5-5` (session; tok: not reported).
- Designer: resolver `recipe claude:claude-opus-5-5`. **Not spawned**: no pick. A routing outcome, not a spawn failure. Rotation pointer unchanged.
- Planner: resolver `agent-tool opus`. **Not spawned**: no pick.
- Executor: resolver `recipe claude:claude-haiku-5-5`. **Not spawned**: no pick.
- Evaluator: `resolve-role-spawn.sh evaluator` → `agent-tool sonnet declared:alias-pin` → Agent tool `model=sonnet`, `MISSION-ROLE: evaluator`, foreground, 1 spawn (62,507 tok). Judge independence: cross-model (controller opus-5-5, judge sonnet; same vendor).
- Driver routing note verbatim: lanes degraded: ; `planner`: **codex** lane `gpt-6.2-sol` unusable (probe rc=`75` — over daily ration) → handed to `opus`; `executor`: **codex** lane `gpt-6.2-sol` unusable (probe rc=`75` — over daily ration) → handed to `claude:claude-haiku-5-5`
- Cost: metered $0 (Anthropic subscription only).

## 130 — 2026-11-21 — pick NONE: all 16 tickets ruling-gated (independent sonnet judge NONE 82%); World 421 adds a 10th slot to D-FLEET-45's ticket but no identical pair (World PR #629 moved head); codex within ration this fire, which changes nothing [ADMIN]

**Pick**: NONE. The 16 open tickets are the same signatures iteration 129 left (diffed against [iteration 129's snapshot](fleet-mission-evidence/iteration129/tickets.json): one ticket changed). The change is one more World occurrence (World 421 at 21:41Z) of row 62's ticket, now 10 slots. That ticket is parked on D-FLEET-45, which is OPEN.
- 0 allowlisted directives on #2341 (18 comments; no watermark file, so the whole issue was read). Self-notices rc 0: 0 crash notices, control 852:10, and 1 transcript-exposure notice, for fire `fleet-1795100889-10927` (iteration 125, already reported by 125–129).
- Saturday fire, not the first after a Monday rotation, so no weekly issue sweep.
- `MISSION_OVER_RATION` empty this fire: the planner and executor resolved to `codex:gpt-6.2-sol` (127–129 were degraded to opus/haiku). Neither was spawned.

**Preflight / observe**:
- Armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing CLEAN.
- Skill drift: the resolved symlink copy matches origin/dev on SKILL.md and all 13 resource files, checked per file; the pin CWD copy is byte-identical to the resolved copy on all 13 resource files. Pin HEAD = origin/dev.
- Gate 1 base `5fd11a779517427f4baf9251b8f34f74dede8ef2@2026-11-21T01:14:43Z`. `CI` success there; 18 check-runs, 0 not green. No open `fleet/*` PRs.
- Traces: no orphan worktree or open PR for iteration 129 or 130. Ledger: the only commit to `fleet-mission.md` since `429eaf806` is iteration 129's own record (`39be1b3ad`); 46 rows valid.

**Did** (generator ≠ judge):
- The controller captured [tickets](fleet-mission-evidence/iteration130/tickets.json), [OPEN rows](fleet-mission-evidence/iteration130/open-decisions.tsv), [World idle snapshots](fleet-mission-evidence/iteration130/world-idle-snapshots.txt) and [World slot verdicts](fleet-mission-evidence/iteration130/world-slot-verdicts.txt).
- **Controller first-party**: World's driver log carries `c3c1a8b1458bb880…` for `world-1795210768-54059` (421, 22:39 local, `head=a51307d ledger=107 open=2 directives=0 inbox=0 red=0`) against 420's `dae8092b…` (`head=301131e`). World PR #629 (`854747c5`, Heads/Timeline UI under `host/`) landed between them.
- **Judge** sonnet (`tools/launchd/resolve-role-spawn.sh evaluator` → `agent-tool sonnet declared:alias-pin`; Agent tool, `MISSION-ROLE: evaluator`, foreground, read-only; 52,651 tok, 4 tool uses, 26 s). The brief carried inputs and four claims to re-verify, gave iteration 129's judge report as something to re-check rather than copy, and told it to argue against idling. Verdict **NONE 82%**. [Report](fleet-mission-evidence/iteration130/idle-judge.md).
  - Claims 1–3 confirmed first-party (one changed ticket; 420 → 421 not skippable; no ledger commit). Claim 4 (codex within ration) not re-read by the judge; the controller read it from the driver env and records it in the report.
  - 16/16 tickets mapped: 8 await D-FLEET-46, 5 out of scope, 1 with no covering row (`eval:motoko-connection-probe-orphaned`, folded into ruled D-FLEET-18), 2 parked on OPEN D-FLEET-44/45. Counts add to 16.
  - Lane availability does not matter: no ticket is routable and starved of a lane.
  - Strongest argument against NONE: resolve `eval:motoko-connection-probe-orphaned` as bookkeeping. Rejected: it is the act D-FLEET-46 asks permission for on eight others, its landing SHA was not verified, and it moves nothing.
  - D-FLEET-45: keep B. 421 adds no identical pair (still 2 in World's last 13 fires).
- **Recorded**: the measurement went into D-FLEET-45 (row stays OPEN, recommendation unchanged) and row 62. No ledger row changed status.

**Ruled out**:
- Resolving `eval:motoko-connection-probe-orphaned` alone ahead of a D-FLEET-46 ruling (judge Q2).
- Treating codex's return as a reason to pick: every ticket is gated by a ruling or by scope, not by a lane.
- Changing D-FLEET-45's recommendation: 421 is not an identical pair.

**Observations (not ticketed; the fleet does not self-source)**:
- (a) This is the fourth consecutive fleet fire (127–130) spending a controller slot on the same NONE screen. Clause 4 is met by its letter only because the queue is non-empty; every remaining ticket waits on Mark or is out of scope.
- (b) The judge returned after 4 tool calls in 26 s. The controller re-checked its load-bearing claims first-party (the ticket diff, the 420/421 hashes, the ledger log) and they held.

**Retro lane**: no skill edit. No friction recurred.

**Progress**: 16 open tickets, unchanged. Goal unmoved.
- Clause 2 unmoved this fire.
- Clause 1 UNMEASURED; clause 3 MET (16 open); clause 4 met by its letter; clause 5 preserved (no harness change).

**Validation**: darwin/arm64.
- `mission_decisions.sh --check`: 46 rows valid.
- STATUS rotation: charter 433 → 433 lines (+4 stamp, −4 moved). Three stamps remain. The archive gained ITERATION 127 (grep 1; control ITERATION 126 = 1). Queue row 68 still present (grep 1); `decision-ledger:end` present (1).
- No code changed, so Gate 3b does not apply. The record ships as a PR.

**Next**: no routable ticket is expected. A ruling on D-FLEET-46 (eight tickets) or D-FLEET-45 is the highest-yield ask; without one, the next fire repeats this screen.

**Routing evidence**: base=`5fd11a779517427f4baf9251b8f34f74dede8ef2`@2026-11-21T01:14:43Z (Gate 1).
- Controller `claude:claude-opus-5-5` (session; tok: not reported).
- Designer: resolver `recipe claude:claude-opus-5-5`. **Not spawned**: no pick. A routing outcome, not a spawn failure. Rotation pointer unchanged.
- Planner: resolver `recipe codex:gpt-6.2-sol`. **Not spawned**: no pick.
- Executor: resolver `recipe codex:gpt-6.2-sol`. **Not spawned**: no pick.
- Evaluator: `resolve-role-spawn.sh evaluator` → `agent-tool sonnet declared:alias-pin` → Agent tool `model=sonnet`, `MISSION-ROLE: evaluator`, foreground, 1 spawn (52,651 tok). Judge independence: cross-model (controller opus-5-5, judge sonnet; same vendor).
- Driver routing note verbatim: (empty — no lane degraded).
- Cost: metered $0 (Anthropic subscription only).

## 131 — 2026-11-21 — pick NONE: all 16 tickets ruling-gated, byte-identical to 130 (independent sonnet judge NONE 85%); World 422 recorded its idle fingerprint undegraded and World 423 idled at the pre-check, the first product-mission idle skip on record, measured into D-FLEET-45 (still OPEN, B unchanged); dev CI red on this mission's own record merge was a 416 s package-timeout flake, green on rerun [ADMIN]

**Pick**: NONE. The 16 open tickets match [iteration 130's snapshot](fleet-mission-evidence/iteration130/tickets.json) exactly: none new, gone or changed ([snapshot](fleet-mission-evidence/iteration131/tickets.json)). OPEN rows are still D-FLEET-44, 45 and 46 ([tsv](fleet-mission-evidence/iteration131/open-decisions.tsv)); no commit touched the charter since `8e28921dc`.
- 0 allowlisted directives on #2341 (19 comments; no watermark file, whole issue read). Self-notices: 0 crash notices, control 852:10, 1 transcript-exposure notice for fire `fleet-1795100889-10927` (iteration 125, already reported by 125–130).
- Saturday fire, not the first after a Monday rotation, so no weekly issue sweep. #2341 created 2026-11-16T07:02Z (08:02 CET, after the boundary), 19 comments: no rotation.
- `MISSION_OVER_RATION` empty; planner and executor resolved to `codex:gpt-6.2-sol`. Neither was spawned.

**Preflight / observe**:
- Armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing CLEAN.
- Skill drift: resolved symlink copy and pin CWD copy both match origin/dev on SKILL.md and all 13 resource files, checked per file. Pin HEAD = origin/dev = `0090b48a6`.
- Gate 1 base `0090b48a69027fc89279da785009fc91e8d9166d@2026-11-21T07:40:17Z`. No open `fleet/*` PRs; no orphan worktree for 130 or 131.
- **dev CI red at Gate 1**: `CI` run 44851261722 on `0090b48a6` (the merge of this mission's own record PR #2439; 9 `design_docs/` files) failed in job `test`, step 12 (`Run tests with timeout (M-DX11)`), 7 steps skipped behind it. The job log (job-logs API, 152,670 B, ANSI stripped) shows `panic: test timed out after 6m56s` and `FAIL github.com/sunholo-data/ailang/internal/pipeline 416.065s`; the test running at the alarm had run 0 s, so the package wall clock ran out and nothing hung. The same package on the four prior green dev runs: 403.149 s (44708657713), 357.607 s (44799939962), 289.633 s (44804128253), 269.275 s (44738679988). The controller re-ran the failed job on the byte-identical tree (ci-health control (c)): attempt 2 **success**, `internal/pipeline` **222.249 s**, 18 check-runs, 0 not green. Verdict: runner variance against a thin budget (222 → 416+ s on one tree, ≥1.87x), not the commit.

**Did** (generator ≠ judge):
- **Controller first-party**: World's driver log shows World 421 (22:39 local, degraded: planner=opus) and World 422 (02:39, undegraded: planner=codex) taking the same idle snapshot `c3c1a8b1…` (`head=a51307d ledger=107 open=2 directives=0 inbox=0 red=0`). 422's report says it ran `idle-fingerprint.sh --record`, and World 423 at 06:38:19 logged `nothing changed since the last no-pick fire (c3c1a8b1… ) — idle, nothing spent` ([excerpt](fleet-mission-evidence/iteration131/world-idle-snapshots.txt)).
- **Judge** sonnet (`tools/launchd/resolve-role-spawn.sh evaluator` → `agent-tool sonnet declared:alias-pin`; Agent tool, `MISSION-ROLE: evaluator`, foreground, read-only; 51,584 tok, 4 tool uses, 51 s). The brief asked it to refute NONE and to check five claims. Verdict **NONE 85%**. [Report](fleet-mission-evidence/iteration131/idle-judge.md).
  - Claims 1, 2, 3, 4c and 4d confirmed. On claim 3, "first" survives: the phrase appears once across every `/tmp/ailang-mission-*.log` on disk (triage's 11-17 digest skip is not a product mission). Caveat: `/tmp` logs can rotate.
  - 4b (fix surfaces outside Authority) was not re-read by the judge. The controller confirmed it: the charter refuses "CI workflows", and `internal/pipeline/**` is language core.
  - The judge marked the 290 s reading UNMEASURED because it grepped `gh run view --log`. The controller's reading came from the job-logs API on that run's `test` job.
  - Strongest argument against NONE: a red on the fleet's own merge commit is an obligation. Rejected: the commit did not cause it, both fix surfaces are out of scope, and V1 already queues the budget as `[NEXT] m-headroom-blocking-threshold-calibration` (`v1-mission.md:617`).
  - Screen by step: no OPEN row begins with an in-scope step without a diff (44 A asks for a design; 45 A is a design revision plus re-quorum; 46 A is an act needing the ruling). Same as 127–130.
- **Recorded**: the D-FLEET-45 row gains the 422 → 423 measurement (row stays OPEN, B unchanged; the opening premise "no product mission has ever promoted an idle fingerprint" is now out of date). Row 62 notes it. No ledger row changed status.

**Ruled out**:
- Fixing or filing the timeout: out of Authority; V1 owns the row; filing it would be self-sourcing and a duplicate.
- Reverting or re-recording iteration 130: the red was not the commit's (identical tree green on rerun).
- Changing D-FLEET-45's recommendation: 422 → 423 shows the existing rule engaging on an undegraded fire, which supports B.

**Observations (not ticketed; the fleet does not self-source)**:
- (a) The V1 loop's last slot verdict is 2026-09-14, so `m-headroom-blocking-threshold-calibration` has no live owner. Gate 1's owning-mission rule ("hand it to the owner") assumes the owner runs. One instance; no skill edit.
- (b) Fifth consecutive NONE fire (127–131).

**Retro lane**: no skill edit. One new friction (owner dormant, observation (a)); the ≥2-friction bar is not met.

**Progress**: 16 open tickets, unchanged. Goal unmoved.
- Clause 2 unmoved. Clause 1 UNMEASURED; clause 3 MET; clause 4 met by its letter; clause 5 preserved (no harness change).
- Harness share, last 20 log headlines: 13/20 (20 tagged).
- Last 3 landings (123, 125, 126) moved clause 2 (ticket resolutions).

**Validation**: darwin/arm64.
- `mission_decisions.sh --check`: 46 rows valid.
- STATUS rotation: charter 433 → 433 lines (+4 stamp, −4 moved). Three stamps remain. The archive gained ITERATION 128 (grep 1; control ITERATION 127 = 1). Queue row 68 present (1); `decision-ledger:end` present (1).
- No code changed, so Gate 3b does not apply. The record ships as a PR.

**Next**: no routable ticket is expected. A ruling on D-FLEET-46 (eight tickets) is the highest-yield ask. D-FLEET-45 now has evidence that World idles on its own when its codex lane is within ration.

**Routing evidence**: base=`0090b48a69027fc89279da785009fc91e8d9166d`@2026-11-21T07:53:22Z (Gate 4).
- Controller `claude:claude-opus-5-5` (session; tok: not reported).
- Designer: resolver `recipe claude:claude-opus-5-5`. **Not spawned**: no pick. A routing outcome, not a spawn failure. Rotation pointer unchanged.
- Planner: resolver `recipe codex:gpt-6.2-sol`. **Not spawned**: no pick.
- Executor: resolver `recipe codex:gpt-6.2-sol`. **Not spawned**: no pick.
- Evaluator: `resolve-role-spawn.sh evaluator` → `agent-tool sonnet declared:alias-pin` → Agent tool `model=sonnet`, `MISSION-ROLE: evaluator`, foreground, 1 spawn (51,584 tok). Judge independence: cross-model (controller opus-5-5, judge sonnet; same vendor).
- Driver routing note verbatim: (empty — no lane degraded).
- Cost: metered $0 (Anthropic subscription only).

## 132 — 2026-11-23 — pick NONE: all 16 tickets ruling-gated (independent sonnet judge NONE 88%); World 423/424 add two slots to D-FLEET-45's ticket (10 → 12) on an idle snapshot no fire could record; the 11-21 15:15 fire died on the Anthropic weekly limit and is credited here; weekly sweep 0 harness-scope orphans of 90; bookkeeping rotated #2341 → #2445 [ADMIN]

**Pick**: NONE. The 16 open tickets match [iteration 131's snapshot](fleet-mission-evidence/iteration131/tickets.json) except one: `skill:idle-record-barred-on-degraded-no-pick-fire` gained two occurrences (`inbox_1795404296951_40d30714`, `inbox_1795418443750_1004eee2`), and its count went from 10 to 12 slots ([snapshot](fleet-mission-evidence/iteration132/tickets.json)). OPEN rows are still D-FLEET-44, 45 and 46 ([tsv](fleet-mission-evidence/iteration132/open-decisions.tsv)). No commit touched the charter since `21a39fb90`.
- 0 allowlisted directives on #2341 (28 comments; no watermark file, so the whole issue was read). Self-notices returned rc 1. There is 1 crash notice (2026-11-21T14:19:15Z, url `…#issuecomment-7063993188`), credited below and not seen in any earlier log entry. The control is 852:10. There is 1 transcript-exposure notice, for fire `fleet-1795100889-10927` (iteration 125, already reported by 125–131).
- This is the first fire after the Monday 07:00 boundary: #2341 was created 2026-11-16T07:02Z, before 2026-11-23 07:00 CET. Both the weekly sweep and the thread rotation were due.
- `MISSION_OVER_RATION=codex`. The planner was handed to `opus` and the executor to `claude:claude-haiku-5-5`. Neither was spawned.

**Orphaned fire (credited, not adopted)**: iteration 132's first attempt was the 2026-11-21 15:15 fire (controller opus).
- It died after 301 s with "You've hit your weekly limit · resets Nov 23 at 6am". Slot verdict: `CRASHED at=gate-4 rc=1`.
- The 16:17 fire (pi controller `glm-5.3:cloud`) paused at gate-3 on an Ollama session 429 (`PAUSED-NO-CAPACITY`).
- They left the untracked `design_docs/fleet-mission-evidence/iteration132/` in the pin, plus the worktree `.wt-fleet-iter132-judge` (branch `fleet/i132-judge` at `0e522fb4b`, never pushed: `git ls-remote origin 'refs/heads/fleet/i132*'` is empty). That worktree holds an `idle-judge.md` reading NONE 90% ([excerpt](fleet-mission-evidence/iteration132/orphan-fire.txt)).
- Its verdict was not adopted: it judged a two-day-old queue.
- Between 11-21 22:25 and 11-23 04:36, all 6 fleet fires paused at `fired`: codex over ration, Ollama demoted, Anthropic quota-limited.

**Preflight / observe**:
- Armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing CLEAN.
- Skill drift: the resolved symlink copy and the pin's CWD copy both match origin/dev on SKILL.md and all 13 resource files, checked per file. Pin HEAD = origin/dev = `ca4d738e1`.
- Gate 1 base `ca4d738e1f654b5f9a9118603edbbdb602acdd98@2026-11-23T09:38:13Z`.
  - `CI` and `Build and Release` success at `ca4d738e1`; Docs-Deploy success at `0e522fb4b` (path-filtered).
  - 18 check-runs, 0 not green. No open `fleet/*` PRs.

**Weekly sweep** ([issue-sweep.md](fleet-mission-evidence/iteration132/issue-sweep.md)): 0 harness-scope orphans of 90 enumerated.
- `gh issue list` length and search total are both 90. The positive control (`#2341`) fired at 23; a fresh negative control fired at 0.
- 79 issues have zero mentions. The 16 filed since the 11-16 sweep are language core, type checker, REPL/playground, CLI, stdlib or packages (#2442, which already has the design doc `m-pkg-path-dep-precedence`), plus triage's own bookkeeping thread #2428. Iteration 114 triaged the older ones the same way.
- No queue row.

**Did** (generator ≠ judge):
- **Controller first-party**, from World's driver log ([excerpt](fleet-mission-evidence/iteration132/world-idle-snapshots.txt), [verdicts](fleet-mission-evidence/iteration132/world-slot-verdicts.txt)):
  - World's head moved to `95a8775` between its 14:38 IDLE exit and its 16:17 fire on 11-21.
  - All 11 World fires since then took the same snapshot `1ac80c01…` (`head=95a8775 ledger=107 open=2 directives=0 inbox=0 red=0`).
  - 10 of them paused before Gate 5 (pi controller). World 424 (08:16 11-23, opus) COMPLETED in 3258 s but was degraded (codex 39.0% vs 32.9%), so it could not record.
- **Judge** sonnet (`resolve-role-spawn.sh evaluator` → `agent-tool sonnet declared:alias-pin`; Agent tool, `MISSION-ROLE: evaluator`, foreground, read-only).
  - Spawn attempts 1 and 2 ended in `API Error: 529 Overloaded` with no output. Attempt 3, identical brief, returned 70,479 tok, 9 tool uses, 58 s.
  - The brief carried the inputs and asked it to refute NONE. Verdict **NONE 88%**. [Report](fleet-mission-evidence/iteration132/idle-judge.md).
  - Claims 1–5 confirmed. The mapping re-derives to 8 awaiting D-FLEET-46, 5 out of scope, 1 folded into ruled D-FLEET-18, and 2 parked on D-FLEET-44/45, which sums to 16.
  - Correction adopted: the first `1ac80c01` snapshot is 11-21 16:17, not 20:16 as briefed. The controller re-read it first-party.
  - New point: World 423 wrote a full no-pick record before its 429 (World 424 landed it via World PR #634). So D-FLEET-45 A's retained unfinished-gate bar would have barred it as well.
  - Strongest argument against NONE: D-FLEET-44 A could begin with an in-scope design revision. Rejected: its default is B, and the revision feeds an approval that is still required.
- **Recorded**: the D-FLEET-45 row and queue row 62 gain the 423/424 measurement. The row stays OPEN and B is unchanged. No ledger row changed status.
- **Rotated** the bookkeeping thread. Opened #2445 and posted the weekly report there. #2341 got "→ continues in #2445" (comment count 28 → 29) and was closed. State files now read 2445 (current) and 2341 (prev).

**Ruled out**:
- Adopting the orphan's NONE 90% verdict: it judged the queue as it stood on 11-21.
- Counting World 424 as a slot the degraded bar cost: no fire had recorded `1ac80c01`, so 424 could not have skipped.
- Spawning the evaluator on a fallback lane after two 529s. A 529 is a transient server overload, not a dead lane, and the third attempt on the pinned lane succeeded.

**Observations (not ticketed; the fleet does not self-source)**:
- (a) The rig lost about two days of fleet slots (11-21 15:15 → 11-23 10:37) to the three buckets being spent at the same time. The fleet's PAUSE did its job: no doomed runs.
- (b) Sixth consecutive NONE fire (127–132).

**Retro lane**: no skill edit. The 529 retry is a single friction, so the ≥2-friction bar is not met.

**Progress**: 16 open tickets, unchanged. Goal unmoved.
- Clause 2 unmoved. Clause 1 UNMEASURED; clause 3 MET; clause 4 met by its letter; clause 5 preserved (no harness change).
- Harness share, last 20 log headlines: 12/20 (20 tagged; the HARNESS rows are 115–126, the fleet's own ticket landings).
- Last 3 landings (123, 125, 126) moved clause 2 (ticket resolutions).

**Validation**: darwin/arm64.
- `mission_decisions.sh --check`: 46 rows valid.
- STATUS rotation: charter 432 → 432 lines (+4 stamp, −4 moved). Three stamps remain. The archive gained ITERATION 129 (grep 1; control ITERATION 128 = 1). Queue row 68 present (1); `decision-ledger:end` present (1).
- No code changed, so Gate 3b does not apply. The record ships as a PR.

**Next**: no routable ticket is expected. A ruling on D-FLEET-46 (eight tickets) is the highest-yield ask. If the next identical World fire runs in full while codex is over ration, count it as D-FLEET-45's 13th slot. That would be the first slot the degraded bar cost on the `1ac80c01` snapshot.

**Routing evidence**: base=`ca4d738e1f654b5f9a9118603edbbdb602acdd98`@2026-11-23T09:52:10Z (Gate 4).
- Controller `claude:claude-opus-5-5` (session; tok: not reported).
- Designer: resolver `recipe claude:claude-opus-5-5`. **Not spawned**: no pick. This is a routing outcome, not a spawn failure. Rotation pointer unchanged.
- Planner: `opus` (codex over ration → opus, per driver). **Not spawned**: no pick.
- Executor: `claude:claude-haiku-5-5` (codex over ration, per driver). **Not spawned**: no pick.
- Evaluator: `resolve-role-spawn.sh evaluator` → `agent-tool sonnet declared:alias-pin` → Agent tool `model=sonnet`, `MISSION-ROLE: evaluator`, foreground.
  - 3 spawn attempts: 2 × `529 Overloaded` (no output), then 1 success (70,479 tok).
  - No fallback lane used.
  - Judge independence: cross-model (controller opus-5-5, judge sonnet; same vendor).
- Driver routing note verbatim: controller pi:ollama/glm-5.3:cloud → claude:claude-opus-5-5 (probe ok); lanes degraded: ; `planner`: **codex** lane `gpt-6.2-sol` unusable (probe rc=`75` — over daily ration) → handed to `opus`; `executor`: **codex** lane `gpt-6.2-sol` unusable (probe rc=`75` — over daily ration) → handed to `claude:claude-haiku-5-5`
- Cost: metered $0 (Anthropic subscription only).

## 133 — 2026-11-23 — pick NONE: all 16 tickets ruling-gated (independent sonnet judge NONE 85%); World 425/426 ran two more full, codex-degraded fires on the unrecorded snapshot `1ac80c01…` (D-FLEET-45's ticket 12 → 14), but the judge showed the narrowed rule would probably not have let 424 record, so they are not proven slots the bar cost; dev CI red on Windows is V1-owned (`internal/executor/agy` test uses `syscall.Kill`) [ADMIN]

**Pick**: NONE. The 16 open tickets match [iteration 132's snapshot](fleet-mission-evidence/iteration132/tickets.json) except `skill:idle-record-barred-on-degraded-no-pick-fire`, which gained `inbox_1795432887694_08f512e1` (World 425) and `inbox_1795447105537_40ad8c3a` (World 426); slots 12 → 14 ([snapshot](fleet-mission-evidence/iteration133/tickets.json)). OPEN rows are still D-FLEET-44, 45 and 46 ([tsv](fleet-mission-evidence/iteration133/open-decisions.tsv)).
- 0 allowlisted directives on #2445 (2 comments) or #2341 (29 comments); no watermark files, so both issues were read whole.
- Self-notices rc 0 with a crash verdict: the single crash notice (`…#issuecomment-7063993188`, 11-21 14:19Z) and the transcript notice (fire `fleet-1795100889-10927`) were both already credited (log 132 and 125–132), so no new death. Control 852:10.
- `MISSION_OVER_RATION=codex google`. Planner → `opus`, executor → `claude:claude-haiku-5-5`. Neither was spawned.

**Preflight / observe**:
- Armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing CLEAN.
- Skill drift: the resolved symlink copy and the pin's CWD copy both match origin/dev on SKILL.md and every resource file, checked per file (no DRIFT line). Pin HEAD = origin/dev = `82917eca8`.
- Died-mid-flight traces: no open `fleet/*` PR; no new fleet worktree beyond `.wt-fleet-iter132-judge` (credited in 132); pin tree clean.
- **Dev CI red at `82917eca8`**: `CI` and `Build and Release` failure; 18 check-runs, 3 not green (`test-windows`, `Build windows-latest`, `SonarCloud Code Analysis`). The `test-windows` job log: `internal\executor\agy\agy_test.go:303:14` and `:308:14: undefined: syscall.Kill` → `FAIL internal/executor/agy [build failed]`. Inherited: `test-windows` is failure on every dev commit back to `be5779e2c` checked (HEAD~1..~3, ~4, ~7, ~8), and the test file was added by `1c5f48816` (agy lane) and edited by `c124fbeb9`, both 11-23. `internal/executor/**` is outside the fleet's Authority and V1 owns reds on this repo (Gate 1 owning-mission rule), so recorded and reported, not picked. SonarCloud not examined.

**Did** (generator ≠ judge):
- **Controller first-party**, from World's driver log ([excerpt](fleet-mission-evidence/iteration133/world-424-426.txt)): World 424 (08:16, 3258 s), 425 (12:17, 2171 s) and 426 (16:16, 1943 s) all reached `slot-verdict: COMPLETED`, all were degraded on codex only (rc 75, used 39.0% vs allowed 32.9 / 36.3 / 39.6% with a 2.0 pp start margin), and all took snapshot `1ac80c011e23…` (`head=95a8775 ledger=107 open=2 directives=0 inbox=0 red=0`), as did the paused 04:16 fire (423).
- **Judge** sonnet (`resolve-role-spawn.sh evaluator` → `agent-tool sonnet declared:alias-pin`; Agent tool, `MISSION-ROLE: evaluator`, foreground, read-only), first attempt, 65,712 tok, 8 tool uses, 55 s. Verdict **NONE 85%**. [Report](fleet-mission-evidence/iteration133/idle-judge.md).
  - Confirmed the World 424–426 readings, the ticket diff, and the 16/16 mapping (8 await D-FLEET-46, 5 out of scope, 1 folded into ruled D-FLEET-18, 2 parked on D-FLEET-44/45).
  - **Refuted the controller's framing** that 425 and 426 were "the first two slots the bar measurably cost": 424 was not a pure no-pick fire (it landed the stranded 423 record via World PR #634, filed World row 266 and spawned an evaluator), and the D-FLEET-45 design keeps the unfinished-gate bar and adds a "only unblock event is a dev-side OPEN ledger flip" test, which World rows 150 (attended A1) and 151 (waits on row 188) fail. Adopted: they are two further slots on a stable snapshot; whether option A would have recorded 424 is unmeasured and probably no. This corrects iteration 132's **Next**, which predicted the next identical fire would be a slot the bar cost.
  - Strongest argument against NONE: build D-FLEET-45 option A as the in-scope code row. Rejected: OPEN with recommendation and default B, an unresolved design defect, quorum-blocked twice.
- **Recorded**: D-FLEET-45 row and queue row 62 gain the 425/426 measurement. Status stays OPEN, B unchanged. No ledger row changed status.

**Ruled out**:
- Counting 425/426 as slots the degraded bar cost (judge refutation above).
- Picking the dev Windows red: outside Authority, V1-owned, inherited from before this fire.
- Building D-FLEET-45 A as a draft: default B, design defect unresolved.

**Observations (not ticketed; the fleet does not self-source)**:
- (a) Seventh consecutive NONE fire (127–133).
- (b) Dev has been red on `test-windows` since the agy lane landed on 11-23; V1's loop has no live owner (last V1 slot verdict 2026-09-14), so this reaches Mark only as an observation.

**Retro lane**: no skill edit; no friction this fire beyond the controller's own over-claim, which the judge caught (one instance; the ≥2-friction bar is not met).

**Progress**: 16 open tickets, unchanged. Goal unmoved.
- Clause 2 unmoved. Clause 1 UNMEASURED; clause 3 MET; clause 4 met by its letter; clause 5 preserved (no harness change).
- Harness share, last 20 log headlines (114–133): 12/20 (20 tagged; the HARNESS rows are 115–126, the fleet's own ticket landings).
- Last 3 landings (123, 125, 126) moved clause 2 (ticket resolutions).

**Validation**: darwin/arm64.
- `mission_decisions.sh --check`: 46 rows valid (re-run after edits).
- STATUS rotation: ITERATION 130 moved to the archive; charter +4/−4.
- No code changed, so Gate 3b does not apply. The record ships as a PR.

**Next**: no routable ticket is expected. A ruling on D-FLEET-46 (eight tickets) remains the highest-yield ask; D-FLEET-45 keeps accumulating World slots while World's codex lane is over ration and the snapshot is stable, but the judge's point stands that its option A may not engage on these fires at all.

**Routing evidence**: base=`82917eca8a04adb47876635146d56ba0223421df`@2026-11-23T16:36:20Z (Gate 4).
- Controller `claude:claude-opus-5-5` (session; tok: not reported).
- Designer: resolver `recipe claude:claude-opus-5-5`. **Not spawned**: no pick. This is a routing outcome, not a spawn failure. Rotation pointer unchanged.
- Planner: `opus` (codex over ration → opus, per driver). **Not spawned**: no pick.
- Executor: `claude:claude-haiku-5-5` (codex over ration, per driver). **Not spawned**: no pick.
- Evaluator: `resolve-role-spawn.sh evaluator` → `agent-tool sonnet declared:alias-pin` → Agent tool `model=sonnet`, `MISSION-ROLE: evaluator`, foreground (65,712 tok), first attempt. No fallback lane used. Judge independence: cross-model (controller opus-5-5, judge sonnet; same vendor).
- Driver routing note verbatim: lanes degraded: ; `planner`: **codex** lane `gpt-6.2-sol` unusable (probe rc=`75` — over daily ration) → handed to `opus`; `executor`: **codex** lane `gpt-6.2-sol` unusable (probe rc=`75` — over daily ration) → handed to `claude:claude-haiku-5-5`
- Cost: metered $0 (Anthropic subscription only).

## 134 — 2026-11-23 — pick NONE: all 16 tickets ruling-gated, byte-identical to 133 (independent sonnet judge NONE 90%); World 427 ran undegraded and recorded the `1ac80c01…` idle fingerprint that 423–426 could not, under the existing rule, measured into D-FLEET-45 (still OPEN, B unchanged); dev CI still red on Windows, V1-owned [ADMIN]

**Pick**: NONE. The 16 open tickets match [iteration 133's snapshot](fleet-mission-evidence/iteration133/tickets.json) exactly: `diff` of the sorted JSON is empty ([snapshot](fleet-mission-evidence/iteration134/tickets.json)). Row 62's ticket stays at 14 slots. OPEN rows are still D-FLEET-44, 45 and 46 ([tsv](fleet-mission-evidence/iteration134/open-decisions.tsv)).
- 0 allowlisted directives on #2445 (3 comments, all the loop's own); no watermark file, so the issue was read whole.
- Self-notices rc 1 with the same single crash notice (`…#issuecomment-7063993188`, 11-21 14:19Z, already credited in log 132) and the same transcript notice (fire `fleet-1795100889-10927`, credited 125–133). No new death. Control 852:10.
- `MISSION_OVER_RATION` empty this fire: codex was within ration, so the driver kept planner and executor on `codex:gpt-6.2-sol`. Neither was spawned.

**Preflight / observe**:
- Armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing CLEAN.
- Skill drift: the resolved symlink copy and the pin's CWD copy both match origin/dev on SKILL.md and all 13 resource files, checked per file (no DRIFT line). Pin HEAD = origin/dev = `6c4ec9c4b`.
- Died-mid-flight traces: no open `fleet/*` PR; no fleet worktree newer than `.fleet-i133-record` (clean, merged as #2451); pin tree clean.
- **Dev CI red at `6c4ec9c4b`**: `CI` and `Build and Release` failure; 23 check-runs, 3 not green (`test-windows`, `Build windows-latest`, `SonarCloud Code Analysis`). Same cause as 133: `internal/executor/agy/agy_test.go:303` and `:308` call `syscall.Kill`, which does not exist on Windows (re-read at HEAD). Outside fleet Authority and V1-owned; recorded only.

**Did** (generator ≠ judge):
- **Controller first-party**, from World's driver log ([excerpt](fleet-mission-evidence/iteration134/world-427.txt)): World 427 (20:16 local, `world-1795461408-73741`) took snapshot `1ac80c011e23…` (`head=95a8775 ledger=107 open=2 directives=0 inbox=0 red=0`), the same as 423–426. Its controller reported codex at 39.0% used vs 42.9% allowed, every role on its configured model, idle fingerprint recorded (rc 0), record landed via World PR #638 (merge `35f5dc6b`). Slot verdict COMPLETED, 3724 s. The World log's last line is 427's `iteration complete` at 21:18:11; no later World fire has started.
- **Judge** sonnet (`resolve-role-spawn.sh evaluator` → `agent-tool sonnet declared:alias-pin`; Agent tool, `MISSION-ROLE: evaluator`, foreground, read-only), first attempt, 59,357 tok, 5 tool uses, 29 s. The brief asked for an independent per-ticket mapping and did not supply the controller's verdict. Verdict **NONE 90%**. [Report](fleet-mission-evidence/iteration134/idle-judge.md).
  - Re-derived the 16/16 mapping: 8 await D-FLEET-46, 5 out of scope under RESOLVED rows, 1 folded into D-FLEET-18, 2 parked on D-FLEET-44/45. Matches iteration 133's judge.
  - On D-FLEET-45: 427 supports B (the bar does not bind while codex is within ration) and does not refute A (426 shows it did bind when degraded). Whether 428 idles is unmeasured.
  - Strongest argument against NONE: resolve the eight D-FLEET-46 tickets now as bookkeeping. Rejected: that is exactly what D-FLEET-46 asks Mark, default B.
  - Residual: the judge did not re-read the charter's step-screen rule (line 109). The controller did: no OPEN row's recommended or shared non-default option begins with an in-scope code step without a diff (44 A asks for a design; 45 A is a design revision plus re-quorum; 46 A is an act needing the ruling). Same as 127–133.
- **Recorded**: D-FLEET-45 row and queue row 62 gain the World 427 measurement. Status stays OPEN, B unchanged. No ledger row changed status.

**Ruled out**:
- Resolving D-FLEET-46's eight tickets without the ruling (judge's strongest objection, rejected above).
- Picking the dev Windows red: outside Authority, V1-owned, inherited.
- Claiming World 428 will idle: no fire has run yet.

**Observations (not ticketed; the fleet does not self-source)**:
- (a) Eighth consecutive NONE fire (127–134).
- (b) World 427 is the second time the existing idle-record rule worked on its own (after 422/423). Its record means the next unchanged World fire should exit at the idle pre-check; the next fleet fire can measure that directly from `/tmp/ailang-mission-world.log`.

**Retro lane**: no skill edit; no friction this fire (the ≥2-friction bar is not met).

**Progress**: 16 open tickets, unchanged. Goal unmoved.
- Clause 2 unmoved. Clause 1 UNMEASURED; clause 3 MET; clause 4 met by its letter; clause 5 preserved (no harness change).
- Harness share, last 20 log headlines (115–134): 12/20 (the HARNESS rows are 115–126).
- Last 3 landings (123, 125, 126) moved clause 2 (ticket resolutions).

**Validation**: darwin/arm64.
- `mission_decisions.sh --check`: 46 rows valid (after edits).
- STATUS rotation: ITERATION 131 moved to the archive (archive count 1, control ITERATION 130 count 1; charter 433 → 433 lines, +4/−4).
- No code changed, so Gate 3b does not apply. The record ships as a PR.

**Next**: no routable ticket is expected. A ruling on D-FLEET-46 (eight tickets) remains the highest-yield ask. Measure whether World 428 exits at the idle pre-check on `1ac80c01…`; if it does, that is direct evidence for D-FLEET-45 B.

**Routing evidence**: base=`6c4ec9c4b72358324e6f84502c17a5ee95b7a789`@2026-11-23T22:54:39Z (Gate 1).
- Controller `claude:claude-opus-5-5` (session; tok: not reported).
- Designer: resolver `recipe claude:claude-opus-5-5`. **Not spawned**: no pick. This is a routing outcome, not a spawn failure. Rotation pointer unchanged.
- Planner: `codex:gpt-6.2-sol` (recipe). **Not spawned**: no pick.
- Executor: `codex:gpt-6.2-sol` (recipe). **Not spawned**: no pick.
- Evaluator: `resolve-role-spawn.sh evaluator` → `agent-tool sonnet declared:alias-pin` → Agent tool `model=sonnet`, `MISSION-ROLE: evaluator`, foreground (59,357 tok), first attempt. No fallback lane used. Judge independence: cross-model (controller opus-5-5, judge sonnet; same vendor).
- Driver routing note verbatim: (empty; no lane degraded this fire).
- Cost: metered $0 (Anthropic subscription only).

## 135 — 2026-11-24 — `push:scope-guard:rebased-record-inherited-sibling-paths` LANDED + RESOLVED: #2456 → `18acb95b7` (independent sonnet judge PASS 92); the pre-push scope guard accepts a rebased mission record and still refuses every branch-authored harness path; the next two World fires exited at the idle pre-check on `1ac80c01…` (D-FLEET-45 still OPEN) [HARNESS]

**Pick**: `push:scope-guard:rebased-record-inherited-sibling-paths` (triage 54, filed 11-24 03:51Z, blocking none, 1 slot). It is the only ticket that changed: the other 16 match iteration 134's snapshot ([tickets](fleet-mission-evidence/iteration135/tickets.json), 17 open). It is mechanical (a wrong diff base in `tools/launchd/githooks/pre-push`), not policy class: no allowlist entry, routing, ration or billing guard moves. Harness path, inside fleet Authority.
- 0 allowlisted directives on #2445 (4 comments, all the loop's own); no watermark file, so the issue was read whole.
- Self-notices rc 1 with the same single crash notice (`…#issuecomment-7063993188`, credited in log 132) and the same transcript notice (fire `fleet-1795100889-10927`, credited 125–134). No new death. Control 852:10.

**Preflight / observe**:
- Armed; gh `sunholo-voight-kampff`; billing CLEAN. `MISSION_OVER_RATION` empty.
- Skill drift: the resolved symlink copy matches origin/dev on SKILL.md and all 13 resource files, per file. Pin HEAD = origin/dev = `f7d54a9c7` at Gate 1.
- Died-mid-flight traces: no open `fleet/*` PR; newest fleet worktree `.fleet-i134-record` (merged as #2452).
- **Dev CI red at `f7d54a9c7`**: 18 check-runs, 4 not green (`test-windows`, `Build windows-latest`, `govulncheck (vuln gate)`, SonarCloud). Windows is the same `agy_test.go` `syscall.Kill` build failure as 133/134; `govulncheck` is new since 134 and has a V1 design already in flight (`f7d54a9c7`, gorilla/websocket past GO-2026-6278). All outside fleet Authority and V1-owned; recorded only.

**Reality check (controller, first-party)**: a throwaway bare repo reproduces the ticket. A `rec` branch pushed with only `design_docs/triage-mission-log.md`, rebased onto a dev that gained `design_docs/fleet-mission-log.md`, then `--force-with-lease` under `MISSION_NAME=triage`: old hook rc 1, "may not push design_docs/fleet-mission-log.md". The three-dot diff lists only the triage log. [Gates and repro](fleet-mission-evidence/iteration135/gates.md).

**Did** (generator ≠ judge):
- **Designer: not spawned.** The spec is the ticket plus the repro (as iterations 87, 105, 109, 123, 126). Routing outcome, not a spawn failure.
- **Planner** `codex:gpt-6.2-sol` (recipe, detached worktree from local HEAD, 60,131 tok). [Plan](fleet-mission-evidence/iteration135/plan.md): keep the remote-head diff for fast-forwards; for a rewrite, verify `origin/dev` against the target's advertised dev with `git ls-remote`, then check the merge-base diff plus every branch-authored commit; fail closed on any lookup error; five real-push arms and four mutants.
- **Executor** `codex:gpt-6.2-sol` (recipe, worktree `.fleet-i135-wt`, 98,163 tok), uncommitted diff, 3 files (+195/−16 in the hook and test, plus a changelog fragment). It reported 4/4 mutants killed (M1→a, M2→c, M3→d, M4→e), shellcheck clean (base had 3 warnings), and `make test-launchd-drivers` UNINFORMATIVE UNDER SANDBOX. [Report](fleet-mission-evidence/iteration135/executor-report.md).
- **Controller**, outside the sandbox at `dd9c8222a`: suite rc 0 (8 PASS); shellcheck rc 0; `make test-launchd-drivers` all 60 suites passed; the ticket repro now rc 0 while a branch-authored `tools/launchd/x.sh` is still refused; 0 temp files left; `git ls-remote` against the real GitHub remote matches `origin/dev`. Committed `dd9c8222a` (co-authored codex), pushed, opened #2456.
- **Judge** sonnet (Agent tool, `MISSION-ROLE: evaluator`, foreground, own detached worktree `.fleet-i135-judge`; 62,818 tok, 187 s), first attempt. The brief carried the ticket and the security constraint, not the controller's verdict. Verdict **PASS 92/100**, 0 blocking. [Report](fleet-mission-evidence/iteration135/judge-r1.md).
  - Bypass attempts all refused: revert after a harness add, harness edit inside a removed merge, rename (`--no-renames` exposes the source path), multi-ref push, fleet core path on a rewrite, newline in a path.
  - Its own mutants: 4/4 killed.
  - Non-blocking: a NEW branch is still checked by net diff only, so add-then-revert passes there. The old hook did the same; not a regression, and not ticketed (the fleet does not self-source).
- **Merge**: SHA-pinned PR poll at `dd9c8222a`, 24/24 complete; required `test`, `lint`, `build`, `docs-gate`, `checks` all green; not green `test-windows`, `Build windows-latest`, `govulncheck` (the inherited dev reds; this diff has no Go). Merged with `--match-head-commit` → **`18acb95b7`** (tree identical to `dd9c8222a`).
- **Done-gate**: surface edited is the one that runs (every mission's guard is the pinned `origin/dev` copy, reached on its next fire); `make test-launchd-drivers` 60/60; dry-runs under the idle, armed stapledon profile with `AILANG_DRIVER_PINNED=18acb95b7`: healthy rc 0 (44 s), degraded (`MISSION_EXECUTOR_MODEL=codex:bogus-model-i135`) rc 0 (41 s, probe rc 1, fell back to `claude:claude-haiku-5-5`); no pidfile left; nothing reloaded.
- **Also measured** (iteration 134's **Next**): the two World fires after 427, at 00:15 and 04:16 on 11-24, each logged "nothing changed since the last no-pick fire (`1ac80c01…`) — idle, nothing spent". [Excerpt](fleet-mission-evidence/iteration135/world-idle-exits.txt). Added to D-FLEET-45 and row 62; status stays OPEN.

**Ruled out**:
- Treating the guard change as policy class: no allowlist entry, lane, ration or billing guard changed; the change narrows attribution to branch-authored commits and fails closed.
- Fixing the judge's new-branch add-then-revert note in the same PR: outside the ticket, pre-existing, and the fleet does not self-source.
- Attributing the PR's three reds to this change: the same three are red on dev's parent `f7d54a9c7`, and the diff touches no Go.

**Observations (not ticketed; the fleet does not self-source)**:
- (a) The base `make test-launchd-drivers` reading was void: it ran in the executor's worktree while the executor was editing it and read the scope-guard suite mid-edit. A base reading belongs in its own worktree.
- (b) Judge note 1 above (new-branch net-diff only) is the one known gap left in the guard.
- (c) Dev deploy is now red on a time-based Node pin drift (`docker/Dockerfile.agent-base` `NODE_VERSION=22.23.4` vs upstream 22.23.5); see Validation. Any dev build that misses the layer cache will fail the same way until the pin moves.

**Retro lane**: no skill edit. One friction (observation a), self-inflicted and below the ≥2-friction bar.

**Progress**: 17 open tickets → 16 after resolution. Goal moved on clause 2.
- Clause 2 moved: a non-policy ticket filed 03:51Z resolved the same morning. Clause 1 UNMEASURED; clause 3 MET; clause 4 met by its letter; clause 5 preserved (done-gate run).
- Harness share, last 20 log headlines (116–135): 12/20, all 20 tagged (the HARNESS rows are 116–126 and 135; the fleet's own work is harness by charter).
- Last 3 landings (125, 126, 135) moved clause 2 (ticket resolutions).

**Validation**: darwin/arm64.
- `mission_decisions.sh --check`: 46 rows valid (after edits).
- STATUS rotation: ITERATION 132 moved to the archive (archive count 1; charter 432 → 433 lines).
- Gate 3b on `18acb95b7` (SHA-pinned, polled to completion before the ticket was resolved): 18 check-runs; required `test`, `lint`, `build`, `checks` green. Not green: the four inherited reds (SonarCloud, `test-windows`, `Build windows-latest`, `govulncheck`) and one NEW red, `ailang-core-dev (ailang-multivac-deploy)`, green on every parent back to `82917eca8`. Attributed first-party from the Cloud Build log (build `efdd7ccb`): step 8 `build-agent-base` fails `node v22.23.5 != pinned v22.23.4 — bump NODE_VERSION deliberately` at `docker/Dockerfile.agent-base:82`. Nodesource shipped 22.23.5 upstream; the parent's 04:07Z build served that layer from cache, this one rebuilt it. Time-based, not from this shell-only diff; `docker/**` is outside fleet Authority and V1-owned, so recorded and reported to Mark.
- Ticket resolved: `ailang mission ticket resolve … --sha 18acb95b7` → "1 occurrence(s), replied to triage"; open list 17 → 16, signature absent.

**Next**: no routable ticket is expected; D-FLEET-46 (eight tickets) remains the highest-yield ask, and the World idle exits are new evidence for D-FLEET-45 B. If triage's next record push is a rebase, its fire is the live confirmation of this fix.

**Routing evidence**: base=`f7d54a9c7bb613bdedf329c403ed3b808e3a14e3`@2026-11-24T05:30:36Z (Gate 1).
- Controller `claude:claude-opus-5-5` (session; tok: not reported).
- Designer: resolver `recipe claude:claude-opus-5-5 declared:provider-pin`. **Not spawned**: nothing to design. Rotation pointer unchanged.
- Planner: resolver `agent-tool opus fail-closed:no-doc`; `derive-planner-lane.sh` → `codex:gpt-6.2-sol declared:planner-lane-default-pin`; role pinned `codex:gpt-6.2-sol`, so routed to the pin under the codex recipe (role-spawn-routing §2(a)). Probe rc 0; 1 run, rc 0, not capped (60,131 tok). Ticket-queue mission: `import-github` skipped.
- Executor: resolver `recipe codex:gpt-6.2-sol declared:provider-pin` → codex recipe, `workspace-write`; 1 run, rc 0, not capped (98,163 tok).
- Evaluator: resolver `agent-tool sonnet declared:alias-pin` → Agent tool `model=sonnet`, `MISSION-ROLE: evaluator`, foreground, own worktree, 1 spawn (62,818 tok). Judge independence: cross-vendor (planner and executor OpenAI codex, judge Anthropic sonnet).
- Driver routing note verbatim: (empty; no lane degraded this fire).
- Cost: metered $0 (Anthropic and codex subscriptions only).

## 136 — 2026-11-24 — `publish:evidence-pack:no-secret-value-guard` BUILT + JUDGED (independent sonnet judge PASS 84 → PASS 90 after two fix rounds): #2463 adds a value-based pre-publication secret scan and a Gate-4 Public publish rule; it cannot merge over dev's inherited required reds (V1 loop disabled), so D-FLEET-48 asks Mark; ticket stays open [HARNESS]

**Pick**: `publish:evidence-pack:no-secret-value-guard` (stapledon 113, filed 11-24 07:31Z, blocking none, 1 slot). It is the only new ticket: the other 16 match iteration 135's post-resolution set ([tickets](fleet-mission-evidence/iteration136/tickets.json), 17 open). It is not policy class: no routing, lane, ration or billing guard moves. The paths (`scripts/mission_*`, `scripts/test_mission_*`, `.claude/skills/mission-*`, `.agents/skills/{mission,sprint}-*`, `make/test.mk`, `changelogs/**`) are all inside fleet Authority.
- 0 allowlisted directives on #2445 (5 comments) or `-prev` #2341 (29 comments); no watermark files, so both issues were read whole.
- Self-notices rc 0: the same single crash notice (`…#issuecomment-7063993188`) and transcript notice (fire `fleet-1795100889-10927`) as 125–135, both already credited. Control 852:10. No new death.

**Preflight / observe**:
- Armed; gh `sunholo-voight-kampff`; billing CLEAN; `MISSION_OVER_RATION=openrouter google` (no role used those buckets).
- Skill drift: the resolved symlink copy and the pin copy match origin/dev on SKILL.md and all 13 resource files, per file. Pin HEAD = origin/dev = `16d168f19`.
- Died-mid-flight traces: no open `fleet/*` PR; no uncommitted fleet worktree. The main checkout holds an attended session's UNCOMMITTED `.agents/skills/sprint-*` sync (12:29), which `b49dc1c59` (12:32) committed without. See Did.
- **Dev CI red at `16d168f19`** (21 check-runs, 11 not green):
  - **Fleet territory (fixed in this PR):** `launchd drivers (bash 3.2)` runs `test_agents_skills_sync.sh` and `test_quorum_artifacts.sh`, and `test` includes `TestQuorumArtifactShellSuite`. All three are red because `b49dc1c59` left `.agents/skills/sprint-*` stale.
  - **V1-owned, not touched:**
    - `lint`: `internal/ai/gemini/generate.go:287` unused `vertexModelURL`.
    - `test`: cihygiene `verify-gap-map` not wired; `TestModels_PricingIsSlugConsistent`; pipeline `TestBuiltinTypes_GoldenSnapshot` and `TestDeprecatedASTConstructsClassified` (`ast.LetPattern`). The last one does not appear in dev's own job log, but it fails locally at base: `go test -run` rc 1.
    - `checks`: `internal/executor/agy/pin_test.go:36` hand-rolled HOME.
    - Windows, `govulncheck`, and the Cloud Build Node pin.
  - The V1 loop is disabled (`mission-control.disabled` present), so no loop owns them.

**Reality check (controller, first-party)**: the shared skill at HEAD carries no publish rule and no secret guard (grep of SKILL.md and all 13 resources for bucket/gs:///tarball/upload finds none). The public-bucket step lives in sibling charters (stapledon: `gs://…/refs/<set>/`). The ticket's premise that leaked values came from the launchd environment, not only `secrets.env`, is why the helper also reads credential-named env vars. [Gates](fleet-mission-evidence/iteration136/gates.md).

**Did** (generator ≠ judge):
- **Designer: not spawned.** The spec is the ticket plus the reality check (as iterations 87, 105, 109, 123, 126, 135). This is a routing outcome, not a spawn failure.
- **Planner** `codex:gpt-6.2-sol` (recipe, detached worktree from HEAD, 103,634 tok). [Plan](fleet-mission-evidence/iteration136/plan.md):
  - M1: helper for files and directories.
  - M2: tar/tgz/zip members, plus suite wiring.
  - M3: Gate-4 rule, a line-neutral SKILL.md pointer, mirror regeneration (controller-only) and a 15-row mutation table.
  - It named the two inherited base reds.
- **Executor** `codex:gpt-6.2-sol` (recipe, worktree `.fleet-i136-wt`, 143,983 tok): uncommitted diff with per-milestone snapshots, 18 arms, 26/26 mutants killed, shellcheck clean. The aggregate was marked UNINFORMATIVE UNDER SANDBOX. [Report](fleet-mission-evidence/iteration136/executor-report.md).
- **Controller**:
  - Reconstructed M1/M2/M3 commits from the snapshots, with the suite run at each boundary; `shasum -c` against the executor's final tree was OK.
  - Wrote the changelog fragment and ran `tools/launchd/sync-agents-skills.sh`. The regenerated sprint mirrors are byte-identical to the main checkout's uncommitted copies, so the attended session's eventual commit rebases to a no-op.
  - Base `make test-launchd-drivers` (own worktree): 2 of 61 suites red. Candidate: all 62 green, with live `secrets.env` unchanged.
- **Live defect → fix round 1** (executor, 46,311 tok): on macOS `/tmp` → `/private/tmp` and `/var` → `/private/var`, so the helper's per-component O_NOFOLLOW made every `/tmp` or `$TMPDIR` path rc 2. That would have blocked every legitimate publish. Now the parent prefix is followed fd-relative; the final component and in-tree links still fail closed. New arm, both mutants killed. Live re-check: `/tmp` and `$TMPDIR` clean rc 0; final symlink rc 2.
- **Live incident shape**: an `env` dump, plain and in a `.tgz`, returned LEAK with 14 hits from 18 loaded values; only key names appear in the output.
- **PR**: #2463. Its body was scanned clean with the new helper in a separate call before `gh pr create` (the rule, dogfooded).
- **Judge** sonnet (Agent tool, `MISSION-ROLE: evaluator`, foreground, own detached worktree `.fleet-i136-judge`, 67,263 tok), first attempt. The brief carried the ticket and the code, not the controller's verdict.
  - **PASS 84/100**, 0 blocking. [Report](fleet-mission-evidence/iteration136/judge-r1.md).
  - Non-blocking but silent: `.tar.bz2`, `.tar.xz`, `.bz2` and `.xz` holding a value returned **clean rc 0**, and a value in a file or directory-child name was missed.
  - Its own mutants: 7 of 8 killed; one equivalent.
- **Fix round 2** (executor, 68,799 tok): bzip2, xz, zstd and 7z signatures (any suffix, top level or as a member) now fail closed. Path-component names are matched as data. One sentence was added to the rule (literal values only; tar/tgz/zip only). Two new arms, both mutants killed. Controller live check: an `env` dump in `.tar.bz2` returns rc 2.
- **Judge round 2** (same agent resumed, cumulative 76,218 tok): **PASS 90/100**, 0 blocking. Findings 1 and 2 re-verified fixed; its own two mutants were killed. [Report](fleet-mission-evidence/iteration136/judge-r2.md).
- **Landing: BLOCKED.** PR CI at `66e0aae56`:
  - Green: required `build` and `docs-gate`, and `launchd drivers (bash 3.2)` (red on dev; this PR fixes it).
  - Red: required `lint`, `test` and `checks`, on exactly dev's V1-owned failures; `TestQuorumArtifactShellSuite` no longer fails.
  - No admin merge (never done by this loop). The record is stacked on the PR branch (Gate 1's blocked-PR rule). D-FLEET-48 filed.

**Ruled out**:
- Policy class: no routing, lane, ration or billing guard changes. The rule adds a publish precondition; the helper only reads.
- Treating the PR's red `test` as caused here: the diff has 0 Go files, and every failing test fails at base.
- Fixing dev's V1-owned reds: they are outside fleet Authority (`internal/ai`, `internal/executor`, `internal/cihygiene`, `internal/pipeline`, `internal/eval_harness`), and the scope guard refuses them.
- Decompressing bz2/xz rather than failing closed: fail closed is the safe minimum, and the rule now states the supported formats.

**Observations (not ticketed; the fleet does not self-source)**:
- (a) Dev's required checks have been red since at least `b49dc1c59` while the V1 loop is disabled. Every PR to dev is BLOCKED, including every sibling mission's record PR. This is the iteration's key finding for Mark.
- (b) The judge's non-blocking residue: base64, split-line and zlib values scan clean by design (now stated in the rule), and a text file starting `BZh` is rejected (fails safe).
- (c) The helper requires `python3` ≥ 3.9 (`removesuffix`); the rig has 3.9.6.

**Retro lane**: no skill edit beyond the deliverable. Friction 1: the live `/tmp` defect was caught by the controller's real-environment run, which no sandboxed gate could see. That is false-green (3) in a new shape, and it is the first instance, below the ≥2 bar.

**Progress**: 17 open tickets, 17 after this iteration (the ticket resolves only once #2463 is on `origin/dev`). Goal unmoved on clause 2; the fix is built and judged but not landed.
- Clause 1 UNMEASURED; clause 2 UNMET, not moved; clause 3 MET; clause 4 met by its letter; clause 5 preserved (nothing landed).
- Harness share, last 20 log headlines (117–136): 12/20, all 20 tagged (the fleet's own work is harness by charter).
- Last 3 landings (126, 135, and none this iteration) moved clause 2.

**Validation**: darwin/arm64.
- `mission_decisions.sh --check`: 47 rows valid (after edits).
- STATUS rotation: ITERATION 133 moved to the archive (archive count 1, control ITERATION 132 count 1). Charter +2 lines net: stamp +4, moved −4, queue row +1, ledger row +1.
- Evidence directory scanned with the new helper before commit: clean, 8 files, control ok.
- Ticket NOT resolved.

**Next**: if dev's required checks are green (re-measure first), rebase #2463, run a SHA-pinned poll, merge with `--match-head-commit`, run the done-gate dry-runs (healthy and degraded, idle armed sibling profile) and resolve the ticket. Otherwise, a ruling on D-FLEET-48.

**Routing evidence**: base=`16d168f1900d6d2ea05cd30f08924141ebb0da81`@2026-11-24T13:49:55Z (Gate 4; Gate 1 read the same SHA at 12:59:13Z).
- Controller `claude:claude-opus-5-5` (session; tok: not reported).
- Designer: resolver `recipe claude:claude-opus-5-5 declared:provider-pin`. **Not spawned**: nothing to design. Rotation pointer unchanged.
- Planner: resolver `agent-tool opus fail-closed:no-doc`; `derive-planner-lane.sh` → `codex:gpt-6.2-sol declared:planner-lane-default-pin`. The role is pinned `codex:gpt-6.2-sol`, so it was routed to the pin under the codex recipe (role-spawn-routing §2(a)). Probe rc 0; 1 run, rc 0, not capped (103,634 tok). Ticket-queue mission: `import-github` skipped.
- Executor: resolver `recipe codex:gpt-6.2-sol declared:provider-pin` → codex recipe, `workspace-write`; 3 runs (main + 2 fix rounds), all rc 0, none capped (143,983 + 46,311 + 68,799 = 259,093 tok).
- Evaluator: resolver `agent-tool sonnet declared:alias-pin` → Agent tool `model=sonnet`, `MISSION-ROLE: evaluator`, foreground, own worktree; 1 spawn plus 1 SendMessage resume (76,218 tok cumulative).
- Judge independence: cross-vendor (planner and executor are OpenAI codex; the judge is Anthropic sonnet).
- Driver routing note verbatim: (empty).
- Cost: metered $0 (Anthropic and codex subscriptions only).

## 137 — 2026-11-24 — pick NONE: 20 open tickets, none routable (independent sonnet judge NONE 90%); triage 55 filed three of dev's required reds as tickets, all outside fleet Authority and already named in D-FLEET-48; dev's required `lint`, `test`, `checks` still red at `ea6d2f28f`, so #2463 stays unmerged [ADMIN]

**Pick**: NONE. The queue went from 17 to 20 open tickets ([tickets](fleet-mission-evidence/iteration137/tickets.json), [diff](fleet-mission-evidence/iteration137/ticket-diff.txt)): triage iteration 55 filed `ci:verify-gap-map-workflow-wiring`, `ci:agy-pin-test-home-isolation` and `ci:modelreg-qwen-slug-pricing-mismatch` at 16:06Z (blocking none, 1 slot each). None left the queue. Each of the three is one of dev's required-check reds:
- The fixes land in `.github/workflows/**` or `internal/cihygiene` (gap-map wiring; the target is `make/code-health.mk:349`), `internal/executor/agy/pin_test.go:36`, and `internal/modelreg/models.yml`.
- All three paths are refused by the guard (`tools/launchd/githooks/pre-push`; `make/test.mk` is the only allowed make file).
- No fix begins with an in-scope step, so "screen by step" does not apply.
- The pricing one would also need owner evidence on vendor rates.
- D-FLEET-48 option A already names all three. They are queued as rows 71–73 under it; no new decision is filed.

The other 17 are unchanged: ruling-gated on D-FLEET-44/45/46, held by their queue rows, or built and waiting (#2463).
- 0 allowlisted directives on #2445 (6 comments) or `-prev` #2341 (29); no watermark files, so both were read whole.
- Self-notices rc 1 on the same single crash notice (`…#issuecomment-7063993188`, 11-21) and transcript notice (fire `fleet-1795100889-10927`) as 125–136, both already credited. Control 852:10. No new death.

**Preflight / observe**:
- Armed; gh `sunholo-voight-kampff`; billing CLEAN; `MISSION_OVER_RATION=openrouter google` (no role used those buckets).
- Skill drift: SKILL.md and all 13 resources match origin/dev, per file, in both the resolved symlink copy and the pin copy. Pin HEAD = origin/dev = `ea6d2f28f`.
- Blocked-PR rule: #2463's branch carries log entry 136, while origin/dev's log ends at 135. This record is stacked on that branch.
- Died-mid-flight traces:
  - The only open `fleet/*` PR is #2463 (known).
  - `.fleet-i136-wt` is clean at the PR head `48eae22ab`.
  - Older fleet worktrees hold only historical residue.
  - The main checkout's uncommitted `.agents/skills/sprint-*` sync is still present (noted in 136).
- **Dev CI red at `ea6d2f28f`** (26 check-runs, 11 not green; [excerpts](fleet-mission-evidence/iteration137/dev-ci.txt)). Required contexts, read from branch protection: `test`, `lint`, `build`, `docs-gate`, `checks`.
  - Red, same causes as at `16d168f19`:
    - `lint`: `internal/ai/gemini/generate.go:287` unused `vertexModelURL`.
    - `test`: `TestVerifyTargetsAreWiredIntoAWorkflow`, `TestModels_PricingIsSlugConsistent`, `TestQuorumArtifactShellSuite`, `TestBuiltinTypes_GoldenSnapshot`.
    - `checks`: hand-rolled HOME.
  - Green: `build` and `docs-gate`.
  - `launchd drivers (bash 3.2)` is still red on the stale `.agents` sprint mirrors, which #2463 fixes.
  - The V1 loop is still disabled (`mission-control.disabled` present).

**Did**: no design, plan, build or merge.
- Evidence directory `iteration137/`: tickets, ticket diff and dev CI excerpts.
- Queue rows 71–73 added. A measurement was appended to D-FLEET-48.
- Judge: sonnet (Agent tool, `MISSION-ROLE: evaluator`, foreground, read-only, 77,520 tok), first attempt. The brief carried the ticket queue, the charter and the CI evidence, not the controller's verdict.
  - Verdict **NONE, 90%**. [Report](fleet-mission-evidence/iteration137/judge.md).
  - It re-derived the ticket diff, the OPEN-row set, the guard allowlist and the four OPEN rows' defaults.
  - It read the required-context list from the API; the controller re-read the same list first-party.
  - It confirmed that #2463's reds are dev's, not the diff's: 0 `.go` files, the same lint and HOME annotations. The PR head is MERGEABLE.
  - Its cautions match this record: no admin merge, and no self-sourced fix for the untracked lint red.

**Ruled out**:
- Building any of the three new tickets: all are outside Authority.
- Splitting #2463's mirror-regeneration commit into its own PR: it would face the same three required reds.
- Admin-merging #2463: that is D-FLEET-48 option B, Mark's call.
- Filing a separate decision for the new tickets: D-FLEET-48 A already covers them.

**Retro lane**: none. No new friction this iteration. Iteration 136's sandbox-blind `/tmp` defect stays at one instance, below the ≥2 bar.

**Progress**: 20 open tickets (17 → 20). Goal unmoved.
- Clause 1 UNMEASURED; clause 2 UNMET, not moved (#2463 waits on dev; three new tickets need an owner outside fleet scope); clause 3 MET (every escalation is in `mission-fleet`); clause 4 met by its letter (tickets are open); clause 5 preserved (nothing landed).
- Harness share, last 20 log headlines (118–137): 11/20, all 20 tagged. This is the fleet's own work, which is harness by charter.
- Last 3 landings (126, 135, and none in 136 or 137): 135 moved clause 2.

**Validation**: darwin/arm64.
- `mission_decisions.sh --check`: 47 rows valid (after edits).
- STATUS rotation: ITERATION 134 moved to the archive (archive count 1; control ITERATION 133 ≥1). Charter +3 lines net: stamp +4, moved −4, queue rows +3; D-FLEET-48 edited in place. Queue-row control `62. [PARKED` still present.
- Live log 39 entries, so no rotation.

**Next**: re-measure dev's required checks. If green: rebase #2463, run a SHA-pinned poll, merge with `--match-head-commit`, run the done-gate dry-runs and resolve its ticket. Otherwise pick NONE again unless a ruling lands (D-FLEET-44/45/46/48) or a new in-scope ticket arrives.

**Routing evidence**: base=`ea6d2f28fe9d2bbe3456c9c3b167e2051eb6b9c7`@2026-11-24T20:07:16Z (Gate 4; Gate 1 read the same SHA at 20:02:58Z).
- Controller `claude:claude-opus-5-5` (session; tok: not reported).
- Designer, planner, executor: **not spawned**, because no ticket was routable (pick NONE, judged). This is a routing outcome, not a spawn failure. Rotation pointer unchanged. The operator's standing request to spawn roles was met for the one role a NONE pick needs: the judge.
- Evaluator: resolver `agent-tool sonnet` (`MISSION_EVALUATOR_MODEL=sonnet`, path `agent-tool`) → Agent tool `model=sonnet`, `MISSION-ROLE: evaluator`, foreground, 1 spawn (77,520 tok).
- Judge independence: the controller is opus and the judge is sonnet. They share a vendor, which is the configured evaluator for this mission. No generator role ran.
- Driver routing note verbatim: (empty).
- Cost: metered $0 (Anthropic subscription only).

## 138 — 2026-11-25 — `skill:mission-base-relative-path-records-driver-repo` BUILT + JUDGED (independent sonnet judge PASS 90): `mission-base.sh` refuses a CWD whose origin is not `$MISSION_REPO`, and every gate calls it through `$MISSION_DRIVER_ROOT`; stacked on #2463, it cannot merge while dev's required `test` and `checks` stay red (D-FLEET-48); ticket stays open [HARNESS]

**Pick**: `skill:mission-base-relative-path-records-driver-repo` (world 431, 01:47Z, blocking none, 1 slot), the one ticket new since iteration 137 ([tickets](fleet-mission-evidence/iteration138/tickets.json), [diff](fleet-mission-evidence/iteration138/ticket-diff.txt)). The other 20 are unchanged from 137: ruling-gated on D-FLEET-44/45/46, outside Authority under D-FLEET-48 A, or built and waiting (#2463).
- Reality check, first-party: `~/.ailang/state/mission-world-base` holds `base-gate1 … f447d431f…` (ailang) at 23:17:44Z and the corrected World SHA `cd06d950…` 6 s later. The World clone has no `tools/launchd/mission-base.sh`, and the helper reads refs from the CWD without checking which repo that is.
- Admissible: the fix is in `tools/launchd/**` and `.claude/skills/mission-*/**`. It is a wrong-path fix, not routing, quota or billing (not HD-2a).
- Critical path: it moves clause 2 (turnaround). D-FLEET-46's eight tickets need a ruling, so this is the only routable clause-2 row.
- 0 allowlisted directives on #2445 (7 comments) or `-prev` #2341 (29). Self-notices rc 1 on the same crash notice (`…#issuecomment-7063993188`) and transcript notice (fire `fleet-1795100889-10927`) as 125–137, already credited. Control 852:10.

**Preflight / observe**:
- Armed; gh `sunholo-voight-kampff`; billing CLEAN; `MISSION_OVER_RATION=google` (no role used it).
- Skill drift: SKILL.md and all 13 resources match origin/dev, per file, in the resolved symlink copy. Pin HEAD = origin/dev = `f447d431f`.
- Blocked-PR rule: #2463's branch carries log entries 136 and 137, while origin/dev's log ends at 135. This build and record are stacked on it (`gate-4-record.md` is in both diffs).
- Died-mid-flight traces: the only open `fleet/*` PR is #2463 (known). The main checkout's uncommitted `.agents/skills/sprint-*` edits are still present (not this loop's; noted since 136).
- **Dev CI red at `f447d431f`** ([excerpt](fleet-mission-evidence/iteration138/dev-ci.txt)): `lint` went green, `build` green. Required `test` is still red on the same four tests and `checks` on the agy hand-rolled HOME (`check-home-isolation`). The V1 loop is still disabled.

**Did**:
- Plan: codex `gpt-6.2-sol` ([plan](fleet-mission-evidence/iteration138/plan.md)), two milestones and a 9-row test table, each row tied to a mutation. The controller gave it its own measurements, labelled, plus an inherited-red note (SC2034 `oldn`).
- Build: codex `gpt-6.2-sol`, `workspace-write` ([report](fleet-mission-evidence/iteration138/executor-report.md)).
  - M1 `a56a2fdc7`: `verify_repo` in `mission-base.sh` parses the CWD's `remote.origin.url` (https, scp, `ssh://`, optional `.git` and trailing slash, case-insensitive). On a mismatch, a missing origin, or a CWD outside Git, `record`/`snap`/`drift` exit **3** before any ref read or write. `last` stays a state-only read. Unset or empty `MISSION_REPO` keeps the old behaviour. The test arms go from 13 to 37, and the unused `oldn` is removed.
  - M2 `8fc5c0e6b`: the ten relative calls in gate-1/3/3b/4 and ref-drift become `bash "$MISSION_DRIVER_ROOT/tools/launchd/mission-base.sh"`, with one ref-drift sentence saying to run it from the mission's repo. A changelog fragment was added.
  - The sandbox refused writes to `.agents/`, so the controller regenerated the mirror (`sync-agents-skills.sh`). Exactly the five counterparts changed, each byte-identical to the executor's `.snap/M2`.
- Controller gates, out of sandbox ([gates](fleet-mission-evidence/iteration138/gates.md)): `test_mission_base.sh` 37/37 (also with an inherited `MISSION_REPO`), shellcheck rc 0 (base rc 1), relative-call grep 0, agents-sync 7/7, context-docs rc 0, `check-changelog` rc 0, `make test-launchd-drivers` 62/62 on base and candidate.
- Incident replay against the real repos with a throwaway state dir: the OLD helper, run from the ailang pin with `MISSION_REPO=sunholo-data/ailang-world`, records `f447d431fb92` (the ticket). The NEW one exits 3 naming both slugs and writes no file. From the World clone it records World's origin/HEAD. Fleet and Stapledon placed correctly also pass.
- All 14 mission clones and pins have `https://github.com/<owner>/<name>.git` origins that match `missions/*.toml`, so the guard cannot refuse a correctly placed fire on this rig.
- Judge: sonnet (Agent tool, `MISSION-ROLE: evaluator`, foreground, own worktree `fleet-i138-judge`, 87,241 tok). The brief carried the ticket, plan and commit range, not the controller's verdict or replay.
  - **PASS 90**, 0 blocking ([report](fleet-mission-evidence/iteration138/judge.md)).
  - It reproduced the incident with old and new helpers, and ran `snap` against all five ailang pins and five clones (all rc 0).
  - It ran 22 mutations. 20 were killed; 2 survived (N2: `--local` dropped; N3: guard placed before argument validation).
  - The controller reproduced N2 first-party (mutant suite rc 0, 0 not ok). It is non-blocking, since no global `remote.origin.url` exists on the rig, and it is recorded as residue on row 74.
- Evidence secret scan (the #2463 helper, from this branch) over `iteration138/`: verdict clean, 18 values, 7 files, control ok.
- Charter: STATUS 138; row 74 added; iteration 138 measurement appended to D-FLEET-48.

**Ruled out**:
- Resolving the ticket: the done-gate needs the fix on `origin/dev`, and dev's required `test`/`checks` reds block every PR.
- Opening the PR against `fleet/i136-evidence-secret-scan`: CI's `pull_request` filter is `[main, dev]`, so that PR would get no CI. It targets dev and says it is stacked.
- A round-2 fix for N2/N3: both are non-blocking, and each would buy a second judge round for a case that does not occur on the rig.
- Folding this into #2463: that would mix two tickets in one PR.
- Admin merge: that is D-FLEET-48 option B, Mark's call.

**Retro lane**: one friction. The codex `workspace-write` sandbox refused writes to `.agents/skills/` in the worktree, so a planned mirror regeneration could only run outside it. This is instance 1 in fleet (136's mirror commit was also controller-side). It is below the ≥2 bar, so no skill edit.

**Progress**: 21 open tickets (20 → 21). Goal unmoved.
- Clause 1 UNMEASURED; clause 2 UNMET, not moved (two built fixes, #2463 and row 74, wait on dev reds outside fleet scope); clause 3 MET; clause 4 met by its letter; clause 5 preserved (nothing resolved).
- Last 3 landings: 126, 135, and none in 136–138.

**Validation**: darwin/arm64.
- `mission_decisions.sh --check`: 47 rows valid.
- STATUS rotation: ITERATION 135 moved to the archive (archive +4 lines; `ITERATION 135` in the archive ≥1). Charter 438 → 439: stamp +4, moved −4, row 74 +1; D-FLEET-48 edited in place. Three `## STATUS 2026` stamps remain.

**Next**: re-measure dev's required checks. If green: land #2463 first (rebase, SHA-pinned poll, `--match-head-commit` merge, done-gate, resolve), then rebase `fleet/i138-mission-base-repo-guard` onto dev and land it the same way. Otherwise pick NONE unless a ruling lands (D-FLEET-44/45/46/48) or a new in-scope ticket arrives.

**Routing evidence**: base=`f447d431fb92bfe051d729b8cb98367cab123ceb`@2026-11-25T02:44:27Z (Gate 4; Gate 1 read the same SHA at 02:12:40Z).
- Controller `claude:claude-opus-5-5` (session; tok: not reported).
- Designer: resolver `recipe claude:claude-opus-5-5 declared:provider-pin`. **Not spawned**: a wrong-path fix with nothing to design. Rotation pointer unchanged.
- Planner: resolver `agent-tool opus fail-closed:no-doc`; `derive-planner-lane.sh` → `opus fail-closed:no-doc`. The role is pinned `codex:gpt-6.2-sol`, so it was routed to the pin under the codex recipe (role-spawn-routing §2(a)). Probe rc 0. 1 run, rc 0, not capped (77,724 tok). Ticket-queue mission: `import-github` skipped.
- Executor: resolver `recipe codex:gpt-6.2-sol declared:provider-pin` → codex recipe, `workspace-write`. 1 run, rc 0, not capped (126,503 tok).
- Evaluator: resolver `agent-tool sonnet declared:alias-pin` → Agent tool `model=sonnet`, `MISSION-ROLE: evaluator`, foreground, own worktree, 1 spawn (87,241 tok). Judge independence: cross-vendor (planner and executor OpenAI codex, judge Anthropic sonnet).
- Driver routing note verbatim: (empty; no lane degraded this fire).
- Cost: metered $0 (Anthropic and codex subscriptions only).

## 139 — 2026-11-25 — #2463 and the i138 branch REBASED + RE-JUDGED (independent sonnet judge PASS 90), not merged: dev went fully green at `7a61839c7` once D-FLEET-48's three reds were fixed attended (`cb6bb75be`), then the v0.16.20 prompt release (`2da0c4592`) added a new required `test` red, `TestRealRegistry_PostMigrationSplitCounts`, which both PRs inherit; both tickets stay open [HARNESS]

**Pick**: land #2463 (`publish:evidence-pack:no-secret-value-guard`, row 70), then the i138 branch (`skill:mission-base-relative-path-records-driver-repo`, row 74), per iteration 138's **Next** and D-FLEET-48's default. The queue is byte-identical to iteration 138 ([tickets](fleet-mission-evidence/iteration139/tickets.json), [diff](fleet-mission-evidence/iteration139/ticket-diff.txt)): 21 open, no new ticket.
- 0 allowlisted directives on #2445 (8 comments) or `-prev` #2341 (29). Self-notices rc 1 on the same crash notice (`…#issuecomment-7063993188`) and transcript notice (fire `fleet-1795100889-10927`) as 125–138, already credited. Control 852:10.
- Died-mid-flight traces: the open `fleet/*` PRs are #2463 and #2466 (the i138 branch), both known. Worktrees `.fleet-i136-wt` and `fleet-i138-wt` were clean at their pushed heads. No iteration-139 trace.

**Preflight / observe**:
- Armed; gh `sunholo-voight-kampff`; billing CLEAN; `MISSION_OVER_RATION=google` (no role used it).
- Skill drift: SKILL.md and all 13 resources match origin/dev, per file, in the resolved symlink copy. Pin HEAD = origin/dev = `2da0c4592` at Gate 1.
- Blocked-PR rule: the i138 branch carries log entries 136–138 while origin/dev's ends at 135, so this record stacks on it.
- **Dev CI** ([excerpt](fleet-mission-evidence/iteration139/dev-ci.txt)), required contexts `test lint build docs-gate checks`:
  - `7a61839c7` (run started 08:30Z): **all green**, the first green dev commit since iteration 133. It was no longer dev's head when the result arrived: `df96971d4` was pushed at 08:34Z. The attended `cb6bb75be` ("unblock v0.61.1 release gate") fixed the agy HOME, qwen pricing, golden, `verify-gap-map` and skill-sync reds. `internal/cihygiene` and `internal/modelreg` are `ok` in that run.
  - `df96971d4`: `test` red, only from an `internal/pipeline` package timeout (6m56s, with a different test running each time; 258 s on the green run). The same timeout also hit `c8022abce` and `5315d160e`.
  - `2da0c4592` (`prompts(v0.16.20)`), and every head after it through `df3994262`: `test` red on `cmd/ailang` `TestRealRegistry_PostMigrationSplitCounts` (`prompt_freeze_test.go:167`, "split 19/54/2 v0.16.19"). The registry now has two mutable versions and `active` is v0.16.20; the test wants `m == 1` and `id == active`. `a5baf4e6d` edited v0.16.20's entry and did not change the split.

**Did**:
- Rebased #2463 onto `2da0c4592`: clean, 8 → 7 commits (git dropped `e4541e88b`, the `.agents` sprint mirror regeneration, as "patch contents already upstream"; `cb6bb75be` carries it). `scripts/mission_decisions.sh --check` 47 rows. Secret scan suite rc 0; `make test-launchd-drivers` 62/62. Force-pushed with lease, `3f5143370` → `cc9216a20`.
- Rebased the i138 branch `--onto cc9216a20 3f5143370`: clean, 3 commits. `test_mission_base.sh` 37/37; `make test-launchd-drivers` 62/62. Force-pushed with lease, `1e223e071` → `981c80183` (#2466).
- Judge: sonnet (Agent tool, `MISSION-ROLE: evaluator`, foreground, own worktree `.fleet-i139-judge`, removed afterwards, 58,734 tok). The brief carried both PRs, the old and new heads, and the dev reds as facts, not the controller's verdict.
  - **PASS 90**, 0 blocking.
  - Range-diff: #2463 7/8 `=`, 1 dropped; #2466 3/3 `=`. The controller re-ran both range-diffs first-party and got the same result. The dropped commit's six paths are all in the dev diff, and `test_agents_skills_sync.sh` passes 7/7.
  - Nine gates re-run, all rc 0: the secret scan suite, `test_mission_base.sh`, `make test-launchd-drivers`, `mission_decisions.sh --check`, `test_agents_skills_sync.sh`, `test_skill_sync.sh`, `check-context-docs`, `test-check-context-docs` and `check-skills`.
  - Five mutants: 4 killed (`verify_repo` return 0 ×2, the scan's `MIN_BYTES`, `CREDENTIAL_NAME`); 1 survived (`O_NOFOLLOW` dropped at `mission_evidence_secret_scan.sh:69`). The survivor is covered by the `follow_symlinks=False` stat at :338, except for a stat-to-open race. Non-blocking residue on row 70.
  - Dev-red attribution: neither PR touches Go, `std/` or prompts. By diff inspection, not measurement (the judge did not run the Go tests).
- Gate 3b: #2463's PR CI on `cc9216a20` (run 45584475837) has `test` red on the same `TestRealRegistry_PostMigrationSplitCounts`, inherited from dev. `lint`, `build`, `checks` and `docs-gate` are green; `test-windows` and the matrix `Build` jobs are red, both non-required. No merge.
- A bounded 45-minute poll of dev (`/tmp/fleet_i139_devpoll.sh`, 09:37–10:22Z) saw five new heads and no green `test`.
- Charter: STATUS 139; rows 70, 71–73 and 74 annotated; iteration 139 measurement appended to D-FLEET-48.

**Ruled out**:
- Admin merge over the new red: that is D-FLEET-48 B, Mark's call.
- Fixing the prompt-freeze red: `prompts/**` and `cmd/ailang/prompt_freeze_test.go` are outside fleet Authority (the guard refuses `cmd/ailang` outside `mission*`), and the red belongs to whoever is releasing prompts. It is V1-owned and the V1 loop is disabled.
- Resolving rows 71–73: their charter predicate fired at `cb6bb75be`, but resolving a ticket whose fix landed attended is exactly D-FLEET-46's open question (default B: leave it). Recorded as a measurement only.
- Waiting past 45 minutes: no dev commit in that window touched the split, so a longer wait is a bet on the attended session's next commit.
- Treating the `internal/pipeline` timeout as the blocker: it hit three of the last six runs and appeared alone only on `df96971d4`. It is a time-budget flake (258 s green vs the 416 s cap) and a second, independent risk to the next green head.

**Retro lane**: one friction. D-FLEET-48's default lands #2463 "once dev's required checks are green". `7a61839c7`'s green arrived at about 08:40Z, but `df96971d4` had become dev's head at 08:34Z and went red. So dev's head was never green when anyone could have observed it, and no fire cadence could have caught it. Instance 1, so no skill edit. If it recurs, the fix is an event trigger (land on a green dev push), not a faster cadence: cadence and scheduling are policy (HD-2a).

**Progress**: 21 open tickets (21 → 21). Goal unmoved.
- Clause 1 UNMEASURED; clause 2 UNMET, unmoved (two built, re-judged fixes wait on one new dev red outside fleet scope); clause 3 MET; clause 4 met by its letter; clause 5 preserved (nothing resolved).
- Last 3 landings: 126, 135, and none in 136–139.

**Validation**: darwin/arm64.
- `mission_decisions.sh --check`: 47 rows valid.
- STATUS rotation: ITERATION 136 moved to the archive (archive +4 lines; `ITERATION 136` in the archive ≥1). Three `## STATUS 2026` stamps remain.

**Next**: re-measure dev's required checks. When `test` is green on dev's head (the prompt-freeze test fixed), rebase #2463 onto it, poll PR CI pinned to the new head, merge with `--match-head-commit`, run the done-gate and resolve the ticket. Then rebase #2466 onto the merge and land it the same way. Otherwise pick NONE unless a ruling lands (D-FLEET-44/45/46/48) or a new in-scope ticket arrives.

**Routing evidence**: base=`df39942627033b50199ade798a33436329311bfb`@2026-11-25T10:18:31Z (Gate 4; Gate 1 read `2da0c4592e047c0b39f326ce0a087d11198afb9c` at 09:10:43Z; dev advanced five commits during the iteration).
- Controller `claude:claude-opus-5-5` (session; tok: not reported).
- Designer: resolver `recipe claude:claude-opus-5-5 declared:provider-pin`. **Not spawned**: landing already-designed work. Rotation pointer unchanged.
- Planner: `codex:gpt-6.2-sol` pin. **Not spawned**: nothing to plan; the work was a rebase of judged commits.
- Executor: `codex:gpt-6.2-sol` pin. **Not spawned**: the rebase was mechanical, clean, with no conflicts, and was done by the controller; no code was authored this iteration.
- Evaluator: resolver `agent-tool sonnet declared:alias-pin` → Agent tool `model=sonnet`, `MISSION-ROLE: evaluator`, foreground, own worktree, 1 spawn (58,734 tok). Judge independence: the code under review was authored by codex `gpt-6.2-sol` (iterations 136 and 138) and the rebase by the controller (opus); the judge is sonnet.
- Driver routing note verbatim: (empty; no lane degraded this fire).
- Cost: metered $0 (Anthropic subscription only).

## 140 — 2026-11-25 — `skill:rotate-log-index-only-targets-registry-workdir` BUILT + JUDGED (independent sonnet judge PASS 92): #2471, stacked on #2466, adds `ailang mission rotate-log --workdir <dir>`, and Gate 4 prescribes it with a fallback for older CLIs; it cannot merge while dev's required `test` stays red (prompt-freeze plus a new `TestCachePipeline_EmbeddedKeys` from `e3595b4f9`, both outside fleet scope); ticket stays open [HARNESS]

**Pick**: `skill:rotate-log-index-only-targets-registry-workdir` (World 433, 11-25 09:54Z, blocking none, 1 slot), the one new in-scope ticket. Iteration 139's **Next** allowed exactly this. The queue went 21 → 22 open ([tickets](fleet-mission-evidence/iteration140/tickets.json), [diff](fleet-mission-evidence/iteration140/ticket-diff.txt)). The other 21 are byte-identical to 139 (sorted JSON `cmp`). D-FLEET-44/45/46/48 are all OPEN. Index grep: rotate-log work at 8, 18, 19, 45, 47, 51 and 56, none on the target directory.
- 0 allowlisted directives on #2445 (9 comments; watermark absent, so a whole-issue read). Self-notices rc 1 on the same crash notice (`…#issuecomment-7063993188`) and transcript notice (fire `fleet-1795100889-10927`) as 125–139, already credited. Control 852:10.
- Died-mid-flight traces: the open `fleet/*` PRs are #2463 and #2466, both known; `fleet-i138-wt` is clean at `03612b56c`. No iteration-140 trace.
- Reality check, first-party from code: `missionRotateLog` sets `logDir := m.Workdir` and replaces it with `m.Root()` only for `sunholo-data/ailang` (`cmd/ailang/mission_rotate.go:72-79`). There is no override flag. `missions/world.toml` `workdir` is the shared main checkout of ailang-world, which has no `missions/` directory. So for World, `--keep` and `--index-only` both write into that main checkout. This is wider than the ticket says: it is not only `--index-only`.

**Preflight / observe**:
- Armed; gh `sunholo-voight-kampff`; billing CLEAN; `MISSION_OVER_RATION=google` (no role used it).
- Skill drift: SKILL.md and 12 of 13 resources match origin/dev in the resolved symlink copy. `gate-2-pick.md` DIFFERS: an uncommitted edit in the main checkout changes the quorum reviewer pool (`codex:gpt-6.2-astra`, `gemini-4-carbon`, `claude-fable-5-2@claude-p`, "2026-11-25"). The resolved copy was followed. No quorum ran this iteration, so it changed nothing here. The main checkout is dirty with an attended session's work, which was left alone.
- Pin HEAD = origin/dev = `e3595b4f9` at Gate 1 and at Gate 4.
- Blocked-PR rule: the i138 branch carries 136–139, so this build and record stack on it. The build shares `gate-4-record.md` with it.
- **Dev CI** ([excerpt](fleet-mission-evidence/iteration140/dev-ci.txt)) at `e3595b4f9`: required `test` red on two failures (log 249,722 B, 170 `ok` package lines as the control): `TestRealRegistry_PostMigrationSplitCounts` ("split 19/54/2", as in 139) and `TestCachePipeline_EmbeddedKeys` ("std/result unexpected dependencies: [std/option]"). The second is new and comes from `e3595b4f9` itself, which added `result.toOption`/`fromOption`. `lint`, `build`, `checks`, `docs-gate` green.

**Did**:
- Plan: codex `gpt-6.2-sol` ([plan](fleet-mission-evidence/iteration140/plan.md)), 2 milestones and an 8-row test table, each row tied to a mutation. The directive named the direction (`--workdir`, the ticket's first candidate) and the no-fallback constraint, and labelled the controller's code reading as verified. Gate list baselined green on `03612b56c` first, including the full `internal/mission`.
- Executor: codex `gpt-6.2-sol`, `workspace-write`, rc 0, not capped ([report](fleet-mission-evidence/iteration140/executor-report.md)). Four mutation drills, all killed. Its `internal/mission` run was correctly labelled UNINFORMATIVE UNDER SANDBOX (`listen tcp6 [::1]:0: bind: operation not permitted`).
- Controller: rebuilt M1 and M2 from the `.snap/` snapshots, with the tests green at each boundary.
  - **Instrument slip (disclosed):** a zsh-unsplit `$FILES` broke the sha256 manifest step. A later `;`-joined reset still ran, so the executor's final tree was reverted before it was manifested.
  - Recovered from the snapshot backup. Faithfulness was shown by the cumulative `--stat` matching the executor's final tree file-for-file (4 files, 290+/15−), not by sha256.
- M3, controller-authored: Gate 4 names the fallback when the CLI answers `unknown flag "--workdir"`. Measured: the rig's PATH `ailang` (v0.61.1-dirty, `cb6bb75`) rejects the flag. The skill reaches every mission on merge, but the flag reaches a mission only when its CLI is rebuilt, and M2 had deleted the earlier forms.
- `.agents` mirror regenerated outside the sandbox (iteration 138's friction, instance 2: the executor was told not to touch it).
- Out-of-sandbox gates, all rc 0:
  - `go build ./cmd/ailang`, `go test ./cmd/ailang -run Rotate`, full `go test ./internal/mission`;
  - `make check-context-docs`, `test_agents_skills_sync.sh` 7/7, `test_skill_sync.sh`;
  - gofmt, `go vet ./cmd/ailang`.
- Smoke on the ticket's exact case, with the built binary and the driver's `AILANG_MISSION_REGISTRY` (pin): a scratch record checkout holding World's origin/dev log and archive, `--index-only --workdir "$PWD"`. The INDEX was written under the record checkout (416 iterations). The World main checkout's porcelain and `world-mission-*` files were byte-identical before and after. A missing target errors with its absolute path, and the old binary gives `unknown flag`.
- Judge ([report](fleet-mission-evidence/iteration140/judge.md)): sonnet, Agent tool, `MISSION-ROLE: evaluator`, foreground, own worktree `.fleet-i140-judge`, 56,020 tok.
  - **PASS 92**, 0 blocking.
  - Five gates re-run green.
  - 11/11 mutations killed, four of them the judge's own (empty-string guard, stat only when not `--index-only`, stat the directory, stat the log regardless of stream).
  - Controller reproduced one kill (skip the stat check → `TestMissionRotateWorkdirMissingSource` FAIL; restored clean) and N1 (`--workdir --keep 5` → `no mission "5"`; loud, nothing written).
- Gate 3b, PR #2471 CI on `62f569387` (bounded poll, 18 min), required checks:
  - green: `lint`, `build`, `checks`, `docs-gate`, `launchd drivers (bash 3.2)`;
  - red: `test`, on the same two failures as dev `e3595b4f9` (PR CI tests the merge with current dev). The diff touches neither file.
  - No merge.
- Record stacked on the branch; `.fleet-i140-judge` removed.

**Ruled out**:
- The scratch-registry recipe alone, the ticket's second candidate: it needs a TOML copy every fire, and it leaves `--keep` hazardous by default. The flag fixes both directly. The recipe is kept only as the old-CLI fallback.
- Auto-detecting the record checkout from CWD: it would be a silent change of target. An explicit flag fails loudly.
- Admin merge over dev's reds: that is D-FLEET-48 B, Mark's call.
- Fixing either dev red: `prompts/**`, `cmd/ailang/prompt_freeze_test.go` and `internal/pipeline` are outside fleet Authority, and both reds come from attended release work.
- Resolving the ticket: the fix is not on `origin/dev`, so the done-gate cannot be met.

**Retro lane**: two frictions, both instance 1 or below the bar, so no skill edit.
- (a) The controller's own zsh slip during snapshot reconstruction. This is the same class as the Gate 3b rule "run every poll or gate loop YOU write under `/bin/bash`", applied to a one-off command. It is covered by the existing rule, so it needs no new text.
- (b) A skill edit that prescribes a new CLI flag goes live before the binary carries it. Instance 1. If it recurs, the fix is a Gate-3 reach check, not prose.

**Progress**: 22 open tickets (21 → 22). Goal unmoved.
- Clause 1 UNMEASURED; clause 2 UNMET, unmoved (three built, judged fixes, rows 70, 74 and 75, wait on dev reds outside fleet scope); clause 3 MET; clause 4 met by its letter; clause 5 preserved (nothing resolved).
- Last 3 landings: 126, 135, and none in 136–140.

**Validation**: darwin/arm64.
- `mission_decisions.sh --check`: 47 rows valid.
- STATUS rotation: ITERATION 137 moved to the archive (archive +4 lines; `ITERATION 137` in the archive ≥1, control `ITERATION 136` ≥1). Three `## STATUS 2026` stamps remain; queue rows 70–75 present.

**Next**: re-measure dev's required `test`. When both reds are gone from dev's head, land #2463, then #2466, then #2471, in that order: rebase onto dev, poll PR CI pinned to the new head, merge with `--match-head-commit`, run the done-gate, resolve the ticket. Otherwise pick NONE unless a ruling lands (D-FLEET-44/45/46/48) or a new in-scope ticket arrives.

**Routing evidence**: base=`e3595b4f953b22863c284dca490c6f89277c083b`@2026-11-25T17:20:03Z (Gate 4; Gate 1 read the same SHA at 16:28:06Z).
- Controller `claude:claude-opus-5-5` (session; tok: not reported).
- Designer: resolver `recipe claude:claude-opus-5-5 declared:provider-pin`. **Not spawned**: a one-flag fix with the direction given by the ticket; nothing to design. Rotation pointer unchanged.
- Planner: resolver `agent-tool opus fail-closed:no-doc`; `derive-planner-lane.sh` → `opus fail-closed:no-doc`. The role is pinned `codex:gpt-6.2-sol`, so it was routed to the pin under the codex recipe (role-spawn-routing §2(a)), not the Agent tool. Probe rc 0. 1 run, rc 0, not capped (60,221 tok). Ticket-queue mission: `import-github` skipped.
- Executor: resolver `recipe codex:gpt-6.2-sol declared:provider-pin` → codex recipe, `workspace-write`. 1 run, rc 0, not capped (93,478 tok).
- Evaluator: resolver `agent-tool sonnet declared:alias-pin` → Agent tool `model=sonnet`, `MISSION-ROLE: evaluator`, foreground, own worktree, 1 spawn (56,020 tok). Judge independence: cross-vendor. The planner and executor were OpenAI codex; M3 and the mirror were the controller (opus); the judge is Anthropic sonnet.
- Operator request for this fire ("use the Agent tool for every role as the routing table specifies"): the table's path for the two codex-pinned roles is the recipe, and the spawn-pin hook refuses an Agent-tool alias for them. They ran on codex as configured, and the evaluator ran on the Agent tool. No role was unspawnable.
- Driver routing note verbatim: (empty; no lane degraded this fire).
- Cost: metered $0 (Anthropic and codex subscriptions only).

## 141 — 2026-11-26 — #2463, #2466 and #2471 LANDED + RESOLVED: dev went green at `63b605cbc` (Release v0.62.1), the stack was rebased and re-judged (independent sonnet judge PASS 91 / 89 / 91) and squash-merged in order as `25cc5bc42`, `fcc3d3481` and `92bc214ba`; dev CI green on all three; tickets `publish:evidence-pack:no-secret-value-guard`, `skill:mission-base-relative-path-records-driver-repo` and `skill:rotate-log-index-only-targets-registry-workdir` resolved [HARNESS]

**Pick**: land the three built, judged PRs in order. Iteration 140's **Next** prescribed exactly this once dev's required `test` went green, and Gate 2's blocked-row rule (d) makes a flipped external predicate the pick regardless of position. D-FLEET-48's default says the same.
- Gate 1, dev `63b605cbc`: 42 check-runs, all success; `CI`, `Build and Release` and Docs-Deploy success on the same SHA. The two `test` reds 140 recorded (`TestRealRegistry_PostMigrationSplitCounts`, `TestCachePipeline_EmbeddedKeys`) were fixed attended before the release.
- Newest record was on the PR branch (#2471 carried 136–140; origin/dev ended at 135), per Gate 1's blocked-PR rule.
- Queue: 23 open (22 → 23). New: `idle-fingerprint:record-eval-json-moves-head` (stapledon 118, 11-25 19:35Z, blocking none, 1 slot). Left for the next fire: one item per iteration ([diff](fleet-mission-evidence/iteration141/ticket-diff.txt), [tickets](fleet-mission-evidence/iteration141/tickets.json)).
- 0 allowlisted directives on #2445 (10 comments; watermark absent, whole-issue read). Self-notices rc 1 on the same crash (`…#issuecomment-7063993188`) and transcript (fire `fleet-1795100889-10927`) notices as 125–140, already credited. Control 852:10.
- Died-mid-flight traces: the open `fleet/*` PRs were exactly #2463, #2466, #2471; their worktrees `.fleet-i136-wt`, `fleet-i138-wt`, `fleet-i140-wt` clean at their recorded heads. No iteration-141 trace.

**Preflight / observe**:
- Armed; gh `sunholo-voight-kampff`; billing CLEAN; `MISSION_OVER_RATION=google` (no role used it).
- Skill drift: SKILL.md and all 13 resources byte-identical to origin/dev in the resolved symlink copy, per file; the CWD (pin) copy also matched. Pin HEAD = origin/dev = `63b605cbc` at Gate 1.
- The main checkout is dirty with attended work (simplicity snapshot, design docs); left alone.

**Did**:
- Rebase: `fleet-i140-wt` onto `63b605cbc`. 0 of the stack's 59 files are among the 205 dev changed since base `2da0c4592`. `git range-diff`: 17 of 17 commits `=`. Force-pushed all three heads with `--force-with-lease` pinned to the old SHAs: #2463 `86daac1f2`, #2466 `d4ae955cd`, #2471 `2c9273918`.
- Judge (sonnet, Agent tool, foreground, own worktree `.fleet-i141-judge` at `2c9273918`, 68,138 tok) ([report](fleet-mission-evidence/iteration141/judge.md)): **PASS** #2463 91, #2466 89, #2471 91, **LAND**, 0 blocking.
  - Re-measured: range-diff 17/17; no dev commit since `2da0c4592` calls `mission-base.sh`, `rotate-log` or the scan helper, and every `mission-base.sh` call in the combined tree goes through `$MISSION_DRIVER_ROOT`.
  - Gates rc 0: `go build ./cmd/ailang`, `go test ./cmd/ailang -run Rotate`, `go test ./internal/mission/...`, `make test-launchd-drivers`, `make check-context-docs`, `go vet ./cmd/ailang`; gofmt clean.
  - Mutations: 6 of 7 killed (origin guard, ZIP members, ZIP fail-open, `--workdir` stat, `--workdir` ignored, one-mirror call-site revert). **Survivor (N1, non-blocking):** reverting a gate's `$MISSION_DRIVER_ROOT/…/mission-base.sh` call to the bare relative path in BOTH mirrors passes `make test-launchd-drivers`. Nothing pins the call-site form; the guard only fires at runtime.
- PR CI round 1 (SHA-pinned, all three in parallel): required five green on each; full set 24/24 each, `CLEAN`.
- Merge #2463 (`--squash --match-head-commit`) → `25cc5bc42`. Dev CI 18/18 green, including `launchd drivers (bash 3.2)`.
- #2466 then read `CONFLICTING`: the squash rewrote record lines that i138's record commits edit further. `diff 86daac1f2 origin/dev` was 0 lines, so `rebase --onto origin/dev 86daac1f2` replayed only i138+i140. Tree diff against the judged heads: 0 and 0, so the judge's verdict carries over unchanged. Force-pushed (#2466 `a055ea325`, #2471 `1601cd4cc`); #2466 full set 24/24 → merged → `fcc3d3481`.
- Same replay for #2471 (`fab014943`, tree diff vs judged head 0). Required five green; full set 23/24 with `Build macos-latest` red on `internal/riglock` `TestAcquireAs_QueuedWaiterCannotTakeTheGap` (non-required; stack touches no riglock file). Merged → `92bc214ba`. Dev tree at `92bc214ba` is byte-identical to the judged head `2c9273918`.
- Gate 3b, dev:
  - `fcc3d3481` attempt 1: `checks` red at step 11 "Install z3" (curl 403 on the z3 release download) and `test` red on an `internal/pipeline` package timeout (6m56s, inside `TestTyvarHygiene_StdExportSchemesAreClosed`; iteration 139's `df96971d4` class). Controls: the stack touches 0 files under `internal/pipeline`, `internal/riglock` or `install_z3` (control pattern 5); the identical z3 step was green on #2471's concurrent run and on `25cc5bc42`/`a055ea325`. Re-ran failed jobs → 18/18.
  - `92bc214ba`: 22/22 green, both `Build macos-latest` legs included, which is the second reading that settles the PR's riglock red as a flake.
- Done-gate, on `92bc214ba` (`AILANG_DRIVER_PINNED`, stapledon profile, idle, `env -i` so this fire's `MISSION_*` could not leak): healthy rc 0 (102 s, `DRY RUN ok`); degraded `MISSION_EXECUTOR_MODEL=codex:bogus-model-i141` rc 0 (104 s, probe rc 1, executor fell back to `claude:claude-haiku-5-5`). No pidfile left; nothing reloaded. `make test-launchd-drivers` green in CI on every merge commit ([gates](fleet-mission-evidence/iteration141/gates.md)).
- Tickets resolved with their landing SHAs: replies sent to stapledon (`publish:…`) and world (both `skill:…`). Open list 23 → 20; all three signatures absent.

**Ruled out**:
- Merge-commit or rebase-merge to avoid the per-PR replay: the fleet lands by squash, and replay plus a tree-identity proof costs only CI time.
- Landing all three through #2471 alone: it would leave #2463/#2466 to be closed by hand and give three tickets one SHA.
- Treating dev's `fcc3d3481` reds as caused by the stack: 0 stack files on either path, positive controls green, and the re-run went green.
- Picking the new stapledon ticket as well: one item per iteration; it is next.
- Resolving D-FLEET-48: the loop may not close its own decision row. The default has run and the ask is moot; Mark can close it.

**Retro lane**: two frictions.
- (a) Instrument slip (controller): a `pkill -f '<poller pattern>'` also matched the backgrounded parent shell, whose command line contained the same string, and ended the harness task holding two other pollers. They kept running as orphans and their verdict files were read instead, so no reading was lost. This is standing rule 7's `pgrep` amendment (a pattern over full argv matches more than you meant). It is instance 1 for `pkill`: kill by PID from `ps -eo pid,ppid,command`, never by pattern. Below the bar for a skill edit.
- (b) A Gate-4 rule that lands mid-fire cannot be followed from the pin: the Public publish rule names `$MISSION_DRIVER_ROOT/scripts/mission_evidence_secret_scan.sh`, and this fire's pin (`63b605cbc`) predates `25cc5bc42`, so the helper is absent there. The evidence was scanned with the origin/dev copy (`.fleet-i141-judge` at `92bc214ba`), and the deviation is disclosed. It resolves itself on the next fire. Instance 1, the same shape as 140's friction (b) (a skill edit outruns the artifact it prescribes); with 140 that makes two. If a third appears, the fix is a Gate-4 fallback line, not prose.

**Progress**: 20 open tickets (23 → 20). Clause 2 moved: three non-policy tickets resolved; turnaround 11-24 07:31Z → 11-26 (≈42 h), 11-25 01:47Z → 11-26 (≈23 h), 11-25 09:54Z → 11-26 (≈15 h).
- Clause 1 UNMEASURED; clause 2 moved; clause 3 MET; clause 4 met by its letter; clause 5 preserved (done-gate run before each resolve).
- Last 3 landings: 126, 135, 141.

**Validation**: darwin/arm64.
- `mission_decisions.sh --check`: 47 rows valid.
- STATUS rotation: ITERATION 138 moved to the archive (charter 441 → 441, archive +4; `ITERATION 138` in archive ≥1, control `ITERATION 137` ≥1); queue rows 64, 73 and the `## Queue` header present; three `## STATUS 2026` stamps.

**Next**: `idle-fingerprint:record-eval-json-moves-head` (stapledon 118): `idle-fingerprint.sh`'s head pathspec does not exclude `.ailang/state/evaluations/eval_MISSION-ITER-*_record.json`, so a record-only fire always moves the head and the idle pre-check can never skip. Reality-check the claim at HEAD first. Otherwise pick NONE unless a ruling lands (D-FLEET-44/45/46/48) or a new in-scope ticket arrives. Optional follow-up from judge N1: a grep gate in `test-launchd-drivers` that pins every gate's `mission-base.sh` call to `$MISSION_DRIVER_ROOT` (not a ticket; would need one to be in scope).

**Routing evidence**: base=`92bc214badb719081b9aca61c6510bc950e67bfb`@2026-11-26T01:02:26Z (Gate 4; Gate 1 read `63b605cbcdcdbbd763264997f37eae7dc3e733c1` at 2026-11-25T23:25:07Z, and the base moved only by this iteration's own three merges).
- Controller `claude:claude-opus-5-5` (session; tok: not reported).
- Designer: resolver `recipe claude:claude-opus-5-5 declared:provider-pin`. **Not spawned**: a landing pick has nothing to design. Rotation pointer unchanged.
- Planner: resolver `agent-tool opus fail-closed:no-doc`; pinned `codex:gpt-6.2-sol`. **Not spawned**: nothing to plan; the plans are iterations 136/138/140's. Ticket-queue mission: `import-github` skipped.
- Executor: resolver `recipe codex:gpt-6.2-sol declared:provider-pin`. **Not spawned**: no code was written; the rebase, pushes and merges were controller bookkeeping on already-built branches.
- Evaluator: resolver `agent-tool sonnet declared:alias-pin` → Agent tool `model=sonnet`, `MISSION-ROLE: evaluator`, foreground, own worktree, 1 spawn (68,138 tok). Judge independence: cross-vendor against the builders (codex `gpt-6.2-sol` planned and built all three) and distinct from the controller (opus). Generator ≠ judge holds.
- Operator request for this fire ("spawn designer, planner, executor and evaluator exactly as the routing table specifies; the evaluator is REQUIRED"): the evaluator ran as specified. The other three roles had no work in a landing-only pick, so none was spawned; this is a scope fact, not a spawn failure. No role was unspawnable, and no fallback was used.
- Driver routing note verbatim: (empty; no lane degraded this fire).
- Cost: metered $0 (Anthropic subscription only).

## 142 — 2026-11-26 — `idle-fingerprint:record-eval-json-moves-head` BUILT + JUDGED (independent sonnet judge PASS 88): #2473 excludes stapledon's record-only `eval_MISSION-ITER-*_record.json` verdicts from the idle fingerprint's `head=`; it cannot merge while dev `94026beac` keeps two required `test` reds from language-core commits outside fleet scope; ticket stays open [HARNESS]

**Pick**: `idle-fingerprint:record-eval-json-moves-head` (stapledon 118, 11-25 19:35Z, 1 slot, `blocking: none`), iteration 141's **Next**. The two higher-ranked tickets (`reporting:daneel-reports-not-declared-triage` 20 slots, `skill:idle-record-barred-on-degraded-no-pick-fire` 14) sit behind OPEN rulings D-FLEET-44 and D-FLEET-45. A wrong path, so mechanical, not HD-2a. It moves clause 2.
- Gate 1: pin `2bd2fe72b`, origin/dev `162f2dcd1` (Release v0.62.2, 1 ahead); base recorded there. Dev CI in flight at Gate 1, no red. No open `fleet/*` PR.
- 0 allowlisted directives on #2445 (12 comments; watermark absent, whole-issue read). Self-notices rc 1 on the same crash (`…#issuecomment-7063993188`) and transcript (fire `fleet-1795100889-10927`) notices as 125–141, already credited. Control 852:10.
- Died-mid-flight traces: no iteration-142 worktree or PR. Open tickets 20, unchanged from 141's list after its three resolutions.

**Preflight / observe**:
- Armed; gh `sunholo-voight-kampff`; billing CLEAN; `MISSION_OVER_RATION=google` (no role used it).
- Skill drift: SKILL.md and all 13 resources byte-identical to origin/dev in the resolved symlink copy, per file. My first loop dropped the `resources/` prefix and printed DIFF for all 13; the re-run with full paths printed SAME for all 13.
- The main checkout is dirty with attended work; left alone.

**Reality check** (controller, first-party):
- At HEAD the helper's `head=` pathspec excludes only `design_docs/<name>-mission*`, `verification/<name>-iter*` and `design_docs/verification/<name>-iter*`.
- stapledon `origin/main` `891c3bc0`: the old pathspec selects `a510c43b` (iteration 119 record); `git show --stat` restricted to the pathspec lists one file, `eval_MISSION-ITER-119_record.json` (same for `6fd5f76c`, iteration 118). With the new exclusion, `head=` is `beb17c4a`, a product commit.
- `eval_MISSION-ITER-*` files: stapledon 7, world 0, this repo 0. Ordinary verdicts are `eval_<sprint>_round_<n>.json`. Nothing in the skills names the record filename; it is stapledon's own convention.

**Did** (generator ≠ judge):
- **Designer: not spawned.** The spec is the ticket plus the repro, a one-line path fix (precedent 87, 105, 109, 123, 126). Rotation pointer unchanged.
- **Planner** codex `gpt-6.2-sol` (recipe, own detached worktree `.planner-wt-fleet-iter142`, probe rc 0; 91,201 tok). [Plan](fleet-mission-evidence/iteration142/plan.md): exclude exactly `eval_MISSION-ITER-*_record.json`; a record-only arm; an ordinary-verdict negative control (`eval_M-FOO_round_1.json` must move the hash); mutants `exclude-record-eval` and `exclude-evaluations-broad`, each checked against its exact expected FAIL line.
- **Executor** codex `gpt-6.2-sol` (recipe, `fleet-i142-wt`, 85,296 tok, rc 0). Applied the plan verbatim; the changelog's link line was dropped per the directive. [Report](fleet-mission-evidence/iteration142/executor-report.md).
- **Controller** re-ran every gate outside the sandbox: plain rc 0; `--mutations` 17/17; shellcheck rc 0; `make check-changelog` rc 0; `make test-launchd-drivers` rc 0 (388 s). The diff writes nothing outside the test's temp fixture. Committed `b79be251f`, secret scan clean.
- **Judge** sonnet (Agent tool, `MISSION-ROLE: evaluator`, foreground, own detached worktree `fleet-i142-judge` at `b79be251f`; 83,804 tok, 669 s). **PASS 88/100, LAND**, 0 blocking ([report](fleet-mission-evidence/iteration142/judge.md)).
  - Re-measured: 17/17 mutants; the helper reverted alone fails on the intended assertion; stapledon repro matches; `make test-launchd-drivers` 61 passed.
  - Strongest objection, non-blocking: the `_record.json` suffix is not pinned. Four widening mutants survive (`eval_MISSION-ITER-*`, `eval_*_record.json`, `eval_MISSION-ITER-*_*.json`, `*_record.json`). No real file is affected. A second negative control such as `eval_MISSION-ITER-9_round_1.json` would kill two of them.
- **PR** [#2473](https://github.com/sunholo-data/ailang/pull/2473).
  - Round 1, head `8bf73c131`: the merge ref was built on dev `8571e7449`, whose own `launchd drivers (bash 3.2)` is red on a stale `.agents/skills/sprint-planner` mirror (`e0f7d431f` synced it). Inherited. The poller was killed by PID.
  - Rebased onto `94026beac`: range-diff 2/2 `=`, 0 files overlap, dev touched nothing under `tools/launchd`. The judge's verdict carries over. Force-pushed with lease → `3c99211fa`.
  - Round 2: 23 checks, 17 green, including `launchd drivers (bash 3.2)`, `lint`, `build`, `checks`, `docs-gate` and govulncheck. 6 red (`test`, `test-windows`, `Build ubuntu-latest`, `Build macos-latest` ×2, `Build windows-latest`), every leg on the same two tests, `TestEveryListBuiltinIsDelegatedOrExplained` (`internal/builtins`) and `TestLetInIfElseBranchError` (`internal/elaborate`).
  - Dev `94026beac` has the same two failures in its own required `test` (contexts `test, lint, build, docs-gate, checks`). They come from language-core commits on dev (`4498d1c2d` parser, `a6e5304f5` std; M-AI-PREFERRED-V2), outside fleet Authority. A bounded 30-min watch saw no new dev commit.
- **Not merged; ticket stays open.** Same posture as D-FLEET-48's default: each fleet fire re-measures dev's required `test` and lands #2473 once it is green. No new decision row; this is an external predicate, not a judgment.
- Done-gate not run (nothing merged). [Gates](fleet-mission-evidence/iteration142/gates.md).

**Ruled out**:
- Excluding the whole `.ailang/state/evaluations/` directory: it would hide ordinary sprint verdicts (the `exclude-evaluations-broad` mutant pins this).
- A mission-namespaced pattern (`eval_${name}-iter*`): no mission writes that; stapledon's convention is the literal `MISSION-ITER`.
- Re-running PR CI on the round-1 head: a PR re-run reuses the same merge ref, so it would rebuild the stale mirror.
- Admin-merging over the inherited reds: branch protection stays unbypassed (D-FLEET-48 recommendation A).
- Strengthening the suffix control after the judge's PASS: a post-verdict edit needs a re-judge; it is a follow-up.

**Retro lane**: two frictions, no skill edit.
- (a) Instance 2 of 141(a): killing the superseded poller with an `awk` match over full argv also matched its wrapper shell, and `ps` output printed the harness's inbox env dump into the transcript (message bodies, no secret values). The poller was superseded, so nothing was lost. Two instances now point at one gap: standing rule 7's `pgrep` amendment covers "is it alive?" but not "kill it". The fix is a line there: kill a poller by the PID its launcher records, never by pattern. Not edited this fire (the edit would ride a PR that cannot merge while dev is red).
- (b) My Gate-1 drift loop used `$(cd "$R" && ls resources/)`, which yields bare basenames, so all 13 read DIFF. The re-run caught it; one instance, below the bar.
- (c) I ran `ailang mission rotate-log fleet --stream status --keep 3` expecting it to rotate the charter's STATUS block. It rotates the status ARCHIVE as a log: 131 of 134 entries moved into a new untracked `fleet-mission-status-archive-old.md`, rc 0, with a plausible summary. The charter was untouched (still 4 stamps), which is what gave it away. I restored the two files the command wrote in my own worktree, deleted its new file, and moved ITERATION 139 by script with both-end assertions (charter 445 → 441, archive 815 → 819; `ITERATION 139` in archive 1, control `ITERATION 138` 2). One instance, my misuse; Gate 4 never prescribes this form for the charter rotation.

**Progress**: 20 open tickets (unchanged). Clause 2 not yet moved: the fix is built and judged but not on origin/dev.
- Clause 1 UNMEASURED; clause 2 pending on #2473; clause 3 MET; clause 4 met by its letter; clause 5 preserved (no resolve without the done-gate).
- Last 3 landings: 126, 135, 141.

**Validation**: darwin/arm64, bash 3.2.57 (`/bin/bash`).
- `test_mission_idle_fingerprint.sh` rc 0 and `--mutations` 17/17 (controller, executor, judge).
- `make test-launchd-drivers` rc 0 (controller 388 s; judge 61 passed); CI `launchd drivers (bash 3.2)` green on `3c99211fa`.

**Next**: re-measure dev's required `test`. If green: rebase #2473, re-check the required set, squash-merge with `--match-head-commit`, run the done-gate on the merge, resolve `idle-fingerprint:record-eval-json-moves-head` (reply to stapledon: expect one more full fire, because its pending snapshot was computed under the old rule). Otherwise pick NONE unless a ruling lands (D-FLEET-44/45/46/48) or a new in-scope ticket arrives. Optional follow-ups (not tickets): the suffix negative control (judge, this iteration) and the `mission-base.sh` call-site grep gate (judge N1, iteration 141).

**Routing evidence**: base=`94026beac12ecffb9663220369292da1f0d46a2c`@2026-11-26T09:49:01Z (Gate 4; Gate 1 read `162f2dcd117f81d66385ef54b056e55b40e891d3`, drift 7 commits, all dev work outside fleet scope).
- Controller `claude:claude-opus-5-5` (session; tok: not reported).
- Designer: resolver `recipe claude:claude-opus-5-5 declared:provider-pin`. **Not spawned**: no doc warranted. Rotation pointer unchanged.
- Planner: resolver `agent-tool opus fail-closed:no-doc`; `derive-planner-lane.sh` with no doc `opus fail-closed:no-doc`. The role is pinned `codex:gpt-6.2-sol`, so per role-spawn-routing §2(a) it ran on the pin under the codex recipe (probe rc 0), 1 run (91,201 tok). `import-github` skipped (ticket-queue mission).
- Executor: resolver `recipe codex:gpt-6.2-sol declared:provider-pin` → codex recipe, 1 run (85,296 tok, rc 0).
- Evaluator: resolver `agent-tool sonnet declared:alias-pin` → Agent tool `model=sonnet`, `MISSION-ROLE: evaluator`, foreground, own worktree, 1 spawn (83,804 tok). Judge independence: cross-vendor against the builders (codex planned and built), distinct from the controller (opus). Generator ≠ judge holds.
- Operator request for this fire: spawn the roles as the routing table specifies, the evaluator required. Planner and executor ran on their codex pins (the table's lane, not the Agent tool); the evaluator ran on the Agent tool. The designer had no doc to write. No role was unspawnable, and no fallback was used.
- Driver routing note verbatim: (empty; no lane degraded this fire).
- Cost: metered $0 (codex and Anthropic subscriptions only).

## 143 — 2026-11-26 — `idle-fingerprint:record-eval-json-moves-head` LANDED + RESOLVED: #2473 → `216d6562b` after its test was hardened against the i142 judge's four surviving widening mutants (independent sonnet judge PASS 91); dev went fully green at `f43d12e3c` mid-fire, so the branch was rebased, merged and done-gated; ticket resolved, open 20 → 19 [HARNESS]

**Pick**: the open ticket `idle-fingerprint:record-eval-json-moves-head` (stapledon 118), iteration 142's **Next**. At Gate 1 its landing predicate was still false, so the fire first took the ticket's own judge objection (non-blocking at 142: the glob's edges were not pinned), then landed once dev turned green. The higher-ranked tickets sit behind OPEN rulings D-FLEET-44/45/46, the rest outside fleet Authority (unchanged from 142). Mechanical, not HD-2a. It moves clause 2.
- Gate 1: pin = origin/dev = `76549ad39`; base recorded there. Open fleet PR #2473 (`fleet/i142-…`) carried iteration 142's record, read from there.
- Dev `76549ad39` required `test` red: `TestEqEvaluatorVMParity/*` fail `IMP010: symbol 'member' not exported by 'std/list'` (`c0ff2b0a4` removed `member`, `cmd/ailang/eq_parity_test.go` still imported it). Language core, outside fleet Authority. `docs-gate` also red at `76549ad39` and `09b77efcf` (MDX error in `docs/docs/reference/security-model.md`).
- 0 allowlisted directives on #2445 (14 comments; watermark absent, whole-issue read). Self-notices rc 1 on the same crash (`…#issuecomment-7063993188`) and transcript (fire `fleet-1795100889-10927`) notices as 125–142, already credited. Control 852:10.
- Died-mid-flight traces: no iteration-143 worktree or PR. Open tickets 20, the same list as 142.

**Preflight / observe**:
- Armed; gh `sunholo-voight-kampff`; billing CLEAN; `MISSION_OVER_RATION=google` (no role used it).
- Skill drift: SKILL.md and all 13 resources byte-identical to origin/dev in the resolved symlink copy AND in the pin copy, per file.
- The main checkout is dirty with attended work; left alone.

**Reality check** (controller, first-party): on #2473's head `04f307466` each of the four widenings the i142 judge named (`eval_MISSION-ITER-*`, `eval_*_record.json`, `eval_MISSION-ITER-*_*.json`, `:(exclude)*_record.json`), applied by `sed` to a scratch helper (each verified to change line 80) and run through `IDLE_TEST_HELPER`, left the plain test at rc 0. All four survive.

**Did** (generator ≠ judge):
- **Designer: not spawned.** A test-only hardening of an existing fix; the spec is the judge report plus the repro. Rotation pointer unchanged (`claude:claude-opus-5-5`).
- **Planner** codex `gpt-6.2-sol` (recipe, own detached worktree at `04f307466`, probe rc 0; 59,004 tok). [Plan](fleet-mission-evidence/iteration143/plan.md): two negative controls (`eval_MISSION-ITER-9_round_1.json`, `eval_M-FOO_record.json` must move the hash), four mutants with exact expected first FAILs, and a per-target no-op guard (six driver mutants tagged `driver`, the rest must change the helper). I checked the six-driver classification against the test before routing.
- **Executor** codex `gpt-6.2-sol` (recipe, `fleet-i142-wt`, 62,716 tok, rc 0). Applied the plan verbatim, no deviations; helper byte-identical. [Report](fleet-mission-evidence/iteration143/executor-report.md).
- **Controller** re-ran the gates outside the sandbox: plain rc 0; `--mutations` 21/21; shellcheck rc 0; `make check-changelog` rc 0. `make test-launchd-drivers` rc 2 on three unrelated suites (`test_mission_workdir_sync` git-timeout, `test_skill_sync` watchdog, `test_motoko_connection_probe` 4 s sampling deadline). None references the idle fingerprint. Two passed when re-run alone. `workdir_sync` also failed on the pristine base at load 9.3 (control). Committed `4eb80c825`, secret scan clean.
- **Judge** sonnet (Agent tool, `MISSION-ROLE: evaluator`, foreground, own detached worktree `fleet-i143-judge` at `4eb80c825`; 54,248 tok, 26 min). **PASS 91/100, LAND**, 0 blocking ([report](fleet-mission-evidence/iteration143/judge.md)).
  - Re-measured: 21/21. Each of b/e/f/g killed by its intended FAIL. Deleting either new arm from a scratch test copy makes its mutants survive, so both are necessary. A deliberately broken helper sed now fails `sed did not apply`. Untagging `age` also fails.
  - Strongest objection, non-blocking: the edges are still not pinned. `eval_MISSION-ITER-[0-9]*_record.json`, `…-*_record*`, `…-*_record.jso*`, `…-?_record.json`, `*/eval_MISSION-ITER-*_record.json` and `.ailang/state/*` all pass. Only `?` could matter (it would wake on multi-digit iterations); that gap predates this commit.
- **Landing**: mid-fire, attended `9ed9f6d7d` fixed `test` and Release v0.64.1 (`f43d12e3c`) fixed `docs-gate`. A SHA-pinned poll read `test`, `lint`, `build`, `docs-gate`, `checks` all success at `f43d12e3c`.
  - Rebased onto it: range-diff 4/4 `=`, 0 files overlap, 0 dev files under `tools/launchd/`. Plain test rc 0. The judge's verdict carries over.
  - Push: the first `--force-with-lease` came back `[remote rejected] (failed)` with no message; the identical command then succeeded → `dde340400`.
  - PR CI: 24 checks, required 5/5 green. `test-windows` (non-required) red on `internal/effects` process timeouts, while the PR has 0 Go files and dev `f43d12e3c` is green on it, so it is a runner flake. `UNSTABLE`, not `BLOCKED`.
  - Merged: `gh pr merge 2473 --squash --match-head-commit dde340400…` → `216d6562b`. Squash patch-id = judged patch-id; `988772ad4` (docs/finetune) landed between rebase and merge. Dev CI on the merge: 18 checks, 0 not-green, incl. `launchd drivers (bash 3.2)`.
- **Done-gate** on `216d6562b` (stapledon profile, idle and armed, `env -i`, `AILANG_DRIVER_PINNED`): healthy rc 0 (121 s, `DRY RUN ok`, but `lanes=DEGRADED(2)` because codex is over its daily ration, probe rc 75); degraded `MISSION_EXECUTOR_MODEL=codex:bogus-model-i143` rc 0 (116 s, executor → `claude:claude-haiku-5-5`). No pidfile left. The dry-run skips the idle pre-check, so CI's `launchd drivers` is the coverage of the change itself. [Gates](fleet-mission-evidence/iteration143/gates.md).
- **Resolved** `idle-fingerprint:record-eval-json-moves-head` with `216d6562b`: rc 0, replied to stapledon (expect one more full fire there; its pending snapshot predates the rule). Open 20 → 19.

**Ruled out**:
- Picking NONE at Gate 1: the open ticket's judge objection was routable work on the ticket itself, not self-sourced.
- Changing the helper's glob: the judge found no real file the widenings or narrowings would misclassify; the fix pins the test, not the rule.
- Waiting for `test-windows` to go green before merging: it is non-required, the PR has no Go code, and dev is green on it.
- Fixing dev's `docs-gate` MDX red (`docs/**` is inside fleet Authority): dev reds are V1's, and Release v0.64.1 fixed it before any fleet action was needed.
- Reading the healthy dry-run's `DEGRADED(2)` as a regression: both entries are codex `rc=75 over daily ration`, the same on the pre-change pin, and the change does not touch lane resolution.

**Retro lane**: three frictions, no skill edit.
- (a) A `git push --force-with-lease` returned `[remote rejected] (failed)` with no reason, and the identical retry succeeded. My first read used `| tail -2; echo rc=$?`, which printed the pipe's rc 0 over a rejected push; the rejection line was visible only because tail kept it. Instance 1 here; the pipe-rc trap is already in memory and in the codex recipe.
- (b) Local `make test-launchd-drivers` is load-sensitive on this rig: 3/62 suites timed out at load ~6–10 while CI's `launchd drivers (bash 3.2)` was green. The base fails the same way, so a local rc 2 needs a base control before it means anything. Instance 1 in the fleet record. The known memory note covers one probe arm, not these three.
- (c) The done-gate says "dry-run healthy and degraded", but no healthy arm can read `lanes=ok` while a quota ration is spent. One instance; recorded, not edited.
- (d) My Gate-1 drift loop changed directory and then globbed `resources/*.md` relative to the wrong tree (zsh: no matches). The Gate-1 file carries a correct snippet (rule (d)); with 142(b) this is two hand-rolled loops that broke, but the fix is to use the snippet, not a new rule.

**Progress**: 19 open tickets (20 → 19). Clause 2 moved: one non-policy ticket resolved (filed 11-25 19:35Z, resolved 11-26 ≈18:40Z, ≈23 h).
- Clause 1 UNMEASURED; clause 2 moved; clause 3 MET; clause 4 met by its letter; clause 5 preserved (done-gate before resolve).
- Last 3 landings: 135, 141, 143.

**Validation**: darwin/arm64, bash 3.2.57 (`/bin/bash`).
- `test_mission_idle_fingerprint.sh` rc 0 and `--mutations` 21/21 (controller, executor, judge); plain rc 0 again after the rebase.
- CI `launchd drivers (bash 3.2)` green on `dde340400` and on the merge `216d6562b`.

**Next**: no routable ticket is expected. Pick NONE unless a ruling lands (D-FLEET-44/45/46; D-FLEET-48 is moot since 141) or a new in-scope ticket arrives. Optional follow-up (not a ticket): a two-digit record control (`eval_MISSION-ITER-19_record.json` must not move the hash) would kill the judge's `?` narrowing.

**Routing evidence**: base=`216d6562bc170153c3838e8ebbcca50dde9f2945`@2026-11-26T18:32:43Z (Gate 4; Gate 1 read `76549ad3983c397c9341885bf5109e298361fbe9`, drift 4 dev commits plus this fire's own merge).
- Controller `claude:claude-opus-5-5` (session; tok: not reported).
- Designer: resolver `recipe claude:claude-opus-5-5 declared:provider-pin`. **Not spawned**: no doc warranted. Rotation pointer unchanged.
- Planner: resolver `agent-tool opus fail-closed:no-doc`; `derive-planner-lane.sh` with no doc `opus fail-closed:no-doc`. The role is pinned `codex:gpt-6.2-sol`, so per role-spawn-routing §2(a) it ran on the pin under the codex recipe (probe rc 0), 1 run (59,004 tok). `import-github` skipped (ticket-queue mission).
- Executor: resolver `recipe codex:gpt-6.2-sol declared:provider-pin` → codex recipe, 1 run (62,716 tok, rc 0).
- Evaluator: resolver `agent-tool sonnet declared:alias-pin` → Agent tool `model=sonnet`, `MISSION-ROLE: evaluator`, foreground, own worktree, 1 spawn (54,248 tok). Judge independence: cross-vendor against the builders (codex planned and built), distinct from the controller (opus). Generator ≠ judge holds.
- Operator request for this fire: spawn the roles as the routing table specifies, the evaluator required. Planner and executor ran on their codex pins (the table's lane, not the Agent tool); the evaluator ran on the Agent tool. The designer had no doc to write. No role was unspawnable, and no fallback was used. After this fire's runs codex reads over its daily ration (done-gate probe rc 75), so the next fleet fire's codex roles will likely degrade.
- Driver routing note verbatim: (empty; no lane degraded this fire).
- Cost: metered $0 (codex and Anthropic subscriptions only).

## 144 — 2026-11-27 — pick NONE: 19 open tickets, none routable (independent sonnet judge NONE 88%); no directive, no new ticket since 143; dev `9b35c44d0` has the required `checks` red on two broken context-doc links left by the attended docs-overhaul merge, outside fleet Authority and V1-owned [ADMIN]

**Pick**: NONE. Iteration 143's **Next** said to pick NONE unless a ruling lands or a new in-scope ticket arrives, and neither happened.
- [Tickets](fleet-mission-evidence/iteration144/tickets.json): 19 open, the same set 143 left. Newest `first_seen` and `last_seen` are both 2026-11-24T16:06Z, so none was filed or re-seen after 143's fire ([diff](fleet-mission-evidence/iteration144/ticket-diff.txt)).
- OPEN rows D-FLEET-44, 45, 46, 48. No ledger row changed since 143 (charter diff `216d6562b..origin/dev` is 143's own record only). D-FLEET-48 stays moot.
- Step screen: 44 A asks for corrected design evidence (no diff); 45's recommendation is B (no change); 46 A is resolving tickets, which is the ruling itself. No row's recommended option begins with an in-scope code step.
- 0 allowlisted directives on #2445 (15 comments; no watermark file, whole issue read). Self-notices: the same crash (`…#issuecomment-7063993188`, 11-21) and transcript (fire `fleet-1795100889-10927`) notices as 125–143, both already credited; control 852:10. No new death.
- Died-mid-flight traces: no open `fleet/*` PR, no `i144` worktree before this fire's own.

**Preflight / observe**:
- Armed; gh `sunholo-voight-kampff`; billing CLEAN; `MISSION_OVER_RATION=codex ollama google` (codex over its daily ration, so the driver handed planner and executor to `opus` / `claude:claude-haiku-5-5`; neither ran).
- Skill drift: SKILL.md and all 13 resources byte-identical to origin/dev in the resolved symlink copy and in the pin copy, per file. Pin HEAD = origin/dev = `9b35c44d0`; driver drift 20.
- **Dev CI red at `9b35c44d0`** (22 check-runs, 2 not green; [excerpts](fleet-mission-evidence/iteration144/dev-ci.txt)). Required contexts: `test`, `lint`, `build`, `docs-gate`, `checks`.
  - `checks`: `make check-context-docs` fails on `.claude/rules/coordinator.md` → `docs/docs/guides/coordinator-workers.md` and `.claude/skills/local-ollama-eval/SKILL.md` → `docs/docs/guides/evaluation/local-ollama.md`. Both targets exist at `216d6562b` and were moved by `d03096c91` (docs M5 overhaul, attended). Walk: `e06d90554` (143's record) `checks=success`; `e5b18e2d7` and `9b35c44d0` `failure`.
  - `Build (1074573805007)` (google-cloud-build): "Couldn't read commit", a Cloud Build read failure, not a code defect.
  - Fix paths: repointing the links or baselining them in `scripts/context_docs_links_baseline.txt` are both outside fleet Authority. Dev reds are V1's, and the V1 loop is disabled.

**Did**: no design, plan, build or merge.
- Evidence directory `iteration144/`: tickets, ticket diff, dev CI excerpts.
- Judge: sonnet (Agent tool, `MISSION-ROLE: evaluator`, foreground, read-only, 86,950 tok), first attempt. The brief named the sources and the question, not the controller's verdict.
  - Verdict **NONE, 88%**. [Report](fleet-mission-evidence/iteration144/judge.md).
  - It re-derived the ticket count and dates, the OPEN-row set, each row's first step, the directive count and the `checks` cause. Its per-ticket table matches 137/143.
  - Strongest objection: recreating the two moved guides under `docs/**` (inside Authority) would turn `checks` green. Rejected: no ticket, it would undo the overhaul's deliberate move, and the real fix (repoint or baseline) is outside Authority.
  - Cautions, all followed: no edits to `.claude/rules/**`, `.claude/skills/local-ollama-eval/**` or the baseline; no resolve while D-FLEET-46 is OPEN (including tickets 17–19, whose fixes have landed); no self-sourced two-digit control test; no `messages read` on `mission-fleet`.

**Ruled out**:
- Recreating the moved `docs/docs/guides/` files to green `checks`: self-sourced, and it reverses attended work.
- Resolving the attended-fixed tickets (D-FLEET-46's set, rows 71–73, and the two config tickets): that is D-FLEET-46 A, Mark's call.
- The two-digit record control 143 floated: not a ticket.

**Retro lane**: one friction, no skill edit.
- (a) `cd /Users/voightkampff/.ailang-driver-pin && git -C fleet worktree add … fleet-i144-rec …` placed the worktree INSIDE the pin (`fleet/fleet-i144-rec`), because the `cd` outside the session's working directory did not hold and the relative path resolved against the pin. Fixed with `git worktree move`. Instance 1; give `git worktree add` an absolute path.

**Progress**: 19 open tickets (unchanged). Goal unmoved.
- Clause 1 UNMEASURED; clause 2 not moved (every remaining ticket waits on a ruling or an owner outside fleet scope); clause 3 MET; clause 4 met by its letter (tickets are open); clause 5 preserved (nothing landed).
- Last 3 landings: 135, 141, 143.

**Validation**: darwin/arm64. `mission_decisions.sh --check` (after edits); STATUS rotation: ITERATION 141 moved to the archive (archive ≥1, charter 0), charter 442 → 442 lines, queue-row control present; canonical heading test.

**Next**: pick NONE unless a ruling lands (D-FLEET-44/45/46) or a new in-scope ticket arrives. If this record PR is blocked by dev's `checks`, the next fire reads it from `fleet/i144-record`.

**Routing evidence**: base=`9b35c44d0749490d79f1ae1a8dd2e528400a23bb`@2026-11-27T01:17:38Z (Gate 4; Gate 1 read the same SHA at 01:10:40Z, drift 0).
- Controller `claude:claude-opus-5-5` (session; tok: not reported).
- Designer: resolver `recipe claude:claude-opus-5-5`. **Not spawned**: no pick, no doc. Rotation pointer unchanged.
- Planner: resolver `agent-tool opus` (codex `gpt-6.2-sol` probe rc 75, ration gate → `opus`). **Not spawned**: no pick.
- Executor: resolver `recipe claude:claude-haiku-5-5` (codex rc 75 → fallback). **Not spawned**: no pick.
- Evaluator: resolver `agent-tool sonnet` → Agent tool `model=sonnet`, `MISSION-ROLE: evaluator`, foreground, read-only, 1 spawn (86,950 tok). Judge independence: distinct from the controller (opus); no generator this fire.
- Operator request for this fire: spawn the roles as the routing table specifies, the evaluator required. The evaluator ran and judged the pick. The designer, planner and executor had nothing to do on a NONE pick. No role was unspawnable, and no fallback was used.
- Driver routing note verbatim: lanes degraded: ; `planner`: **codex** lane `gpt-6.2-sol` unusable (probe rc=`75` — blocked by the ration gate (no reason line captured)) → handed to `opus`; `executor`: **codex** lane `gpt-6.2-sol` unusable (probe rc=`75` — blocked by the ration gate (no reason line captured)) → handed to `claude:claude-haiku-5-5`
- Cost: metered $0 (Anthropic subscription only).
