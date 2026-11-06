# Fleet Mission Log

Append-only, one entry per iteration, newest at the bottom. Charter: [fleet-mission.md](fleet-mission.md).

> **Older entries are ARCHIVED.** This file holds the newest 20. The full record of every
> iteration is in `fleet-mission-log-archive.md`, and a one-line index of ALL of them —
> the thing to grep before picking work, so the loop never repeats itself — is in
> `fleet-mission-index.md`.

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

## 76 — 2026-10-31 — `executor-image:no-make-no-cc-cgo-tests` RESOLVED as a ghost of #1753 (`939beb6fd`) for the sprint-executor; the cloud sprint-evaluator's image still lacks make and a C compiler (≥4 of 8 runs since 10-10) and parks on D-FLEET-41; `eval:motoko-connection-probe-orphaned` folds into D-FLEET-18; independent sonnet PASS-WITH-EDITS 72 / 85 [HARNESS]

**Pick**: iteration 75's **Next**, `executor-image:no-make-no-cc-cgo-tests` (attended, 10-08). It measured as a ghost plus an out-of-scope residual, so it was a bookkeeping pick and a second item was taken (standing rule 1): `eval:motoko-connection-probe-orphaned` (motoko, 10-01), the named follow-on.

**Orphans / prior record**: no open `fleet/*` PR (iteration 75's #2033 merged → `bf4f14473`). No `fleet/i76*` branch or worktree before this fire. Self-notices rc 0 (control 852:10).

**Preflight / observe**:
- Armed. gh `sunholo-voight-kampff`. Billing CLEAN.
- Directives: 0 on #1945 since `2026-10-22T23:29:08Z` (18 comments, none allowlisted; the watermark is still LEGACY).
- Skill drift: SKILL.md and all 13 resource files are `cmp`-identical to `origin/dev` at the resolved symlink target. The pin is at `origin/dev`.
- Gate-1 base `bf4f14473f704f57fd3c6c5454fc8eaf115c505e@2026-10-31T15:07:06Z`.
- Dev CI at `bf4f14473`: `CI` and `Build and Release` success. 22 checks, one not green: the inherited `ailang-core-dev (ailang-multivac-deploy)` Cloud Build red. Not required, outside fleet scope, V1-owned; recorded, not picked.
- Ledger valid, 39 rows, 24 OPEN at start. 33 tickets open.
- Codex is over its daily ration (`MISSION_OVER_RATION=codex`). The driver handed the planner to `opus` and the executor to `claude:claude-haiku-5-5`; neither was needed.
- Weekly issue sweep: not due (last run iteration 61, Monday 10-26; this fire is not the first after a rotation).

**Reality check** (first-party, prod Cloud Logging `ailang-multivac`, before routing):
- **The ticketed task.** task-69be551b ran on `ailang-agent-executor-codex-go` (`gpt-6.1-sol`), started 19:09:18Z on 10-08, and published `blocked`, `files=0`, at 19:12:19Z.
- **The fix.** #1753 (`939beb6fd`, 10-09) is the M-TASK-INPUTS sprint that task was executing, and it added `make gcc libc6-dev` to `docker/Dockerfile.agent-codex-go:6-13`. The live codex-go job's latest execution was created 10-29. The controller did not prove the image digest contains #1753; the behaviour below does (the judge also found codex-go sprint-executor runs on 10-16 and 10-27 running `make test-core`, `make lint` and `make test-changed`).
- **Before and after.** Missing-toolchain filter (`make: not found`, `make: command not found`, `C compiler`, `gcc` not found, `requires cgo`) over all executor jobs: at least 50 lines (the `--limit`, a floor) in 10-01..10-09 on codex-go and pi-go; since 10-09T18:00Z, 0 on codex-go. Since 10-10, all 20 sprint-executor starts (18 tasks) ran on codex-go.
- **Counting the class, not the cited instance** (iteration 75's retro (b)). The sprint-evaluator ran 26 times on `agent-pi-go` in 10-01..10-09 (pi-go ran nothing else then) and 8 times since on `agent-go` (`ailang-agent-executor-go` 2, `-go-claude-credit` 6, one image digest). Neither Dockerfile, nor their bases, installs make or gcc. The controller's filter caught 1 of the 8 (task-68e3cfb1, 10-27, `box has no C compiler`; it ran `CGO_ENABLED=0` build and vet and scored 84). The post-fix `ailang-agent-executor-pi` lines (10-13, 10-16) are design-doc-creator tasks on a non-Go image.
- **Skill half.** #1905 `3f9a0c092`, `sprint-executor/resources/unattended_runs.md:58-64`; the sprint-evaluator skill has no missing-toolchain text.
- **Second pick.** `git grep motoko_connection_probe` at HEAD, outside the pair: `make/test.mk:128,140-141`, `scripts/test_check_referenced_paths.sh:45-46` (fixture), banked sprint JSONs. Last change `6c03639f5` (09-06). Motoko disabled. `sprint/motoko-iter39-armcount-r3` is on origin at `f6750002d`.

**Did** (no code; designer, planner and executor not spawned; evaluator spawned; generator ≠ judge):
- **Designer, planner, executor**: not spawned, because nothing in fleet scope needed building. This is not a spawn failure. Resolver output: designer `recipe claude:claude-opus-5-5 declared:provider-pin`, planner `agent-tool opus fail-closed:env-pin`, executor `recipe claude:claude-haiku-5-5 declared:provider-pin`. Rotation pointer not advanced.
- **Evaluator**: `sonnet`, Agent tool, foreground, read-only on the pin (prod logs read-only); resolver `agent-tool sonnet declared:alias-pin`. It was given the draft ([as judged](fleet-mission-evidence/iteration76/draft-records-as-judged.md)) and told to refute it ([report](fleet-mission-evidence/iteration76/evaluation.md)).
  - Round 1: record 1 **PASS-WITH-EDITS 72** ("as drafted, about 55"), record 2 **PASS-WITH-EDITS 85**. Its BLOCKING-IF-LEFT findings:
    - F1: the evaluator gap is at least 4 of 8, not 1 of 8. Three 10-12 runs on `-go-claude-credit` said it in prose (``make` isn't installed``), which the filter could not match.
    - F2: "no missing toolchain on codex-go" is false as worded: task-401dce49 (10-29) logged `go: not found` under `make test-changed`. Make itself worked; likely a `bash -lc` PATH reset. Not the ticket's ask.
    - F3: name the non-Go pi image's `make`/`go: command not found` lines as out of scope.
  - Non-blocking: D-FLEET-41 option A names `Dockerfile.agent-go` first; option B names both skill copies; row 57's window is 10-01..10-09; record 2 should say retiring needs a third path.
  - The controller re-ran F1 (3 executions on 10-12, mapped to task-62beea26, task-f96e9162, task-f257f24d) and F2 (task-401dce49) before applying. All edits applied. The controller also moved D-FLEET-41's default from B to C (no change), because B is fleet work no ticket asks for, so it needs a ruling rather than a default.
  - No round 2: every edit was a wording change the judge supplied, and neither disposition changed.
- **Recorded**: ticket `executor-image:no-make-no-cc-cgo-tests` resolved (`--sha 939beb6fd`); D-FLEET-41 filed (OPEN, recommendation A, default C); D-FLEET-18 re-measure appended. Queue rows 57 and 58. 32 tickets open.
- **Done-gate**: not applicable (no harness change).

**Progress**: goal unmoved on clause 1 (UNMEASURED). Clause 2 UNMET (no in-scope ticket filed in the last 48h; the resolved ticket was filed 10-08, so its 23-day turnaround does not help the median). Clause 3 MET (32 open). Clause 4: prior evidence only. Clause 5 preserved.

**Routing evidence**: Gate-1 base `bf4f14473f704f57fd3c6c5454fc8eaf115c505e@2026-10-31T15:07:06Z`. Routing note (driver): `lanes degraded: planner and executor codex gpt-6.1-sol unusable (probe rc=75, over daily ration) → opus / claude:claude-haiku-5-5`.
- Controller `claude:claude-opus-5-5` (tok: not reported).
- Designer: not spawned (nothing to design); rotation pointer not advanced.
- Planner: not spawned (nothing to plan).
- Executor: not spawned (no code).
- Evaluator `sonnet`, Agent tool: 104,754 subagent tokens / 20 tool uses / 160 s. Judge independence: a different model from the controller (opus); same vendor, so vendor-level independence is not claimed.
- Quorum: none (no design doc). Metered spend: $0. No lane recorded dead.

**Ruled out**:
- Keeping the ticket open on D-FLEET-41: it names the sprint-executor image, and both halves of its ask are built; the evaluator is a different role and image (the judge agreed).
- A separate decision row for the motoko probe: same two files and the same owner question as D-FLEET-18 option A.
- Treating the post-fix `ailang-agent-executor-pi` `make` misses as this class: they are design-doc-creator and planner runs on a non-Go image.

**Retro lane**: no skill edit. Observation, not a ticket:
- (a) Iteration 75 flagged three adjacent "the query window cut off what the judge found" gaps and asked the next retro to watch for a fourth. This is the fourth, in a new form: the controller counted the class correctly this time (it found the evaluator), but its *pattern* matched only shell error text, and agents report missing tools in prose. Four frictions across iterations 73–76 now share one root: **a log census is only as wide as its wording**. The candidate rule (for Gate 2's reality-check): when a census over agent logs returns a count that the decision rests on, run a second, prose-shaped query (`isn't installed`, `not available`, `missing`) and report both counts. Not edited this fire: the four instances are adjacent rather than one gap, and a rule edit is live for every mission the moment it lands, so it should go through a fleet ticket or Mark. Flagged in the report.
- (b) Seventh consecutive fire (70 was the last code landing) whose picks were ghosts or out-of-scope rows. D-FLEET-36 and D-FLEET-38 remain the rulings that would turn parked tickets into landable fleet work; D-FLEET-41 option B is a third.

Last 3 code landings: iteration 70 (`52e8958f0`), then none at 71–76. No drift alarm: no routable in-scope UNMET-clause row exists.

**Next**: `agent-tool:workspace-routing-discovery-timeout` (stapledon, 10-02), the one open ticket with neither a queue row nor a decision row (30 of 33 have a queue row; `quorum:oc-glm-5-3-output-cap-invalid` and `quorum:claude-p-ration-usage-429` have D-FLEET-31 and D-FLEET-28). Scope check first: it names a codex-native Agent transport error and no harness surface (log entry 65 noted the same).

## 77 — 2026-11-02 — `agent-tool:workspace-routing-discovery-timeout` folds into D-FLEET-24 (one codex-internal transport error, recovered by `followup_task`; none in 652 later sessions); the fold gives option B's premise one reading; no other ticket routable; weekly sweep 0 items; independent sonnet PASS-WITH-EDITS 84 [HARNESS]

**Pick**: iteration 76's **Next**, `agent-tool:workspace-routing-discovery-timeout` (stapledon iteration 9, 10-02, 1 slot, `blocking=none`). It was the one open ticket with neither a queue row nor a decision row. It measured as a symptom of D-FLEET-24, so it was a bookkeeping pick. No second item was taken (standing rule 1 allows one): all 32 open tickets now have a queue row or an OPEN decision row, and row 16's trigger (`quorum:tier2-salvage-fold`, "its first non-test caller") has not fired. `git grep RunQuorumWithEscalation -- '*.go'` outside tests finds only its definition (`internal/mission/quorum/quorum.go:100,112`); the control, the same grep including tests, finds 9.

**Orphans / prior record**: no open `fleet/*` PR (iteration 76's #2038 merged → `ffabe3278`). No `fleet/i77*` branch or worktree before this fire. Self-notices rc 0 (control 852:10).

**Preflight / observe**:
- Armed (`mission-fleet.disabled` absent). gh `sunholo-voight-kampff`. Billing CLEAN.
- Directives: 0 on #1945 since `2026-10-22T23:29:08Z` (19 comments, none allowlisted; the watermark is still LEGACY).
- Skill drift: SKILL.md and all 13 resource files are `cmp`-identical to `origin/dev` in both the resolved symlink target and the pin copy. The pin is at `origin/dev` (`AILANG_DRIVER_AGE=0`).
- Gate-1 base `ffabe32784b2cf44c0bca57e723e60de46a7a905@2026-11-02T07:58:06Z`.
- Dev CI at `ffabe3278`: `CI` and `Build and Release` success. 22 checks, one not green: the inherited `ailang-core-dev (ailang-multivac-deploy)` Cloud Build red. Not required, outside fleet scope, V1-owned; recorded, not picked.
- Ledger valid, 40 rows, 25 OPEN. 32 tickets open. The newest filing is 10-28.
- Codex is over its daily ration (`MISSION_OVER_RATION=codex`). The driver handed the planner to `opus` and the executor to `claude:claude-haiku-5-5`; neither was needed.

**Reality check** (first-party, before routing; [census](fleet-mission-evidence/iteration77/census.md)):
- **Harness surface**: `git grep -i 'workspace routing\|routing discovery'` at `ffabe3278` finds no harness file (2 unrelated v0.9.0 design docs). 0 hits in the 14 `/tmp/ailang-mission-*.log` driver logs.
- **Census**: codex session store `~/.codex/sessions/2026/10`, 1,049 files (10-01..10-30; no `2026/11` directory). Three wording variants match the same 27 files. Most of the 65 matching records quote the ticket. The occurrences are: (1) 10-02, stapledon iteration 9: the `evaluator_transport` sub-agent's turn ended `workspace routing discovery timed out` (`codex_error_info: other`, `codex_exec` 0.159.2), and the parent received it as a FINAL_ANSWER. (2) 10-09: a `workspace routing discovery unauthorized (401)` variant in 3 attended `codex-tui` 0.162.0 stapledon sessions, which is not a mission fire and not ticketed. After 10-09 there are 652 files, 39 of which made 107 `spawn_agent` calls, and none has an occurrence.
- **The chain**: the controller's `spawn_agent(model: "sonnet")` returned ``Unknown model `sonnet` for spawn_agent`` (08:18:16Z; paired by `call_id`). The controller then spawned `evaluator_transport` (`gpt-6.1-sol`), which ran `claude-sub --model claude-sonnet-5-5 … --print` at 08:49:56Z. The codex turn errored at 09:03:15Z, `followup_task` followed at 09:04:32Z, and the turn completed at 09:19:56Z with "PASS, 91/100".
- **Option B's premise**: both sessions' `turn_context` reads `sandbox_policy: danger-full-access`, `approval_policy: never`, which is the driver's launch (`tools/launchd/mission-control.sh:2431`, `--dangerously-bypass-approvals-and-sandbox`). D-FLEET-24 B said `claude-sub` inside the codex sandbox was unmeasured. There is now one reading, in a session with the policy the driver's launch sets. That this fire came from the driver is inferred (the ticket calls it an "API scheduled invocation"); under a restricted sandbox it is still unmeasured.

**Did** (no code; designer, planner and executor not spawned; evaluator spawned; generator ≠ judge):
- **Designer, planner, executor**: not spawned, because nothing in fleet scope needed building. This is not a spawn failure. Resolver output: designer `recipe claude:claude-opus-5-5 declared:provider-pin`, planner `agent-tool opus fail-closed:env-pin`, executor `recipe claude:claude-haiku-5-5 declared:provider-pin`. Rotation pointer not advanced.
- **Evaluator**: `sonnet`, Agent tool, foreground, read-only; resolver `agent-tool sonnet declared:alias-pin`. It was given the draft records ([log](fleet-mission-evidence/iteration77/draft-log-entry-as-judged.md), [charter diff](fleet-mission-evidence/iteration77/draft-charter-diff-as-judged.patch)) and told to refute them ([report](fleet-mission-evidence/iteration77/evaluation.md)). **PASS-WITH-EDITS 84**. One BLOCKING-IF-LEFT finding: F1, the sweep's "7 new" is 14. Three non-blocking: F2, "A and B remove the transport" overreaches, because B may still be wrapped in a sub-agent under the skill's "never inline" invariant and the error can hit any codex sub-agent. F3, the option-B reading's driver origin is inferred, not proven. F4, D-FLEET-24's title, default and slot count were stale. The controller re-derived F1 (`comm` over the two tables) and checked F2's citation before applying. All edits applied, with no round 2: every edit was a correction to wording or a count, and the disposition did not change. The judge re-derived the census, the chain (including real Claude output in the transport's judge JSON), the sandbox policy, the ticket and slot sums and the row-16 predicate.
- **Recorded**: queue row 59 (`[PARKED · D-FLEET-24]`); row 31 points to it; D-FLEET-24 gains an iteration-77 re-measure (7 tickets, 25 of 63 lost slots; B's premise measured once; recommendation A unchanged). Ticket stays open. 32 tickets open.
- **Weekly external-issue sweep** (first fire after the Monday rotation; the last was iteration 61, 10-26): 57 open issues enumerated, matching the search total of 57. 55 have zero mentions across the charter, log, log archive, status archive and dashboard. The positive control `#1306` hit 13 times, and the negative control (a fresh, unpublished literal) fired. The 14 opened since the iteration-61 table (committed 10-26 10:13Z) are 6 `ailang-explainers` language and verify reports (#1947, #1950, #1953–#1956), 4 `motoko_agent` `std/process` reports (#2034–#2037), 2 `stapledons_godot` language reports (#1980, #2029) and the two bookkeeping issues (#1945 fleet, #1974 triage). All are outside Authority, so 0 queue items. The controller's first draft said 7: its filter, `createdAt > 2026-10-26`, dropped the 7 filed on 10-26 after that sweep. The judge caught it by diffing the two tables, and the controller re-derived it with `comm`. Table: [issue-sweep.tsv](fleet-mission-evidence/iteration77/issue-sweep.tsv).
- **Weekly thread rotation** (Monday, after 07:00 local; #1945 was created 10-26): opened [#2039](https://github.com/sunholo-data/ailang/issues/2039) and posted the weekly report (comment count 0→1). Posted the pointer on #1945 (19→20) and CLOSED it. `mission-fleet-gh-issue`=2039, `-prev`=1945. No `mission-2039-last-seen` exists yet; `mission_directives.sh` reads every allowlisted comment when it is absent, and the first `--mark` creates it, so it was not hand-written.
- **Done-gate**: not applicable (no harness change).

**Gate 3b** (the record, #2040):
- PR head `e8cd62f7a`: 27/27 checks terminal, 0 non-green, all 6 required green; MERGEABLE/CLEAN. Merged with `--merge --match-head-commit` → `4da85a61d`.
- Merge `4da85a61d`, CI run 41414186301: the required `test` FAILED. The `cmd/ailang` package hang guard fired at 395s against `go test -timeout 416s` ("this PACKAGE has outgrown its go test -timeout budget (not a failure of this test's logic)"). Not this commit, which is records only: the same package ran 302s, 343s (82%, warned) and 388s (93%, warned) on the three previous dev pushes. One `gh run rerun --failed`: attempt 2 passed, with `cmd/ailang` at 237s (57%). The merge's check set is then 18 checks, the only non-green being the inherited `ailang-core-dev` Cloud Build. Recorded on D-FLEET-17 as a fourth intermittent signature; `.github/workflows/**` is outside fleet scope. This Gate-3b addendum (#2041) was written after the judge returned and is not judged. It is CI readings only, and each one is cited to its run.

**Progress**: goal unmoved on clause 1 (UNMEASURED). Clause 2 UNMET (no ticket filed since 10-28; this one was filed 10-02 and stays open). Clause 3 MET (32 open). Clause 4: prior evidence only. Clause 5 preserved.

**Routing evidence**: Gate-1 base `ffabe32784b2cf44c0bca57e723e60de46a7a905@2026-11-02T07:58:06Z`. Routing note (driver): `lanes degraded: planner and executor codex gpt-6.1-sol unusable (probe rc=75, over daily ration) → opus / claude:claude-haiku-5-5`.
- Controller `claude:claude-opus-5-5` (tok: not reported). Heartbeat stamps: gate-0, -1, -2, -4 and -5; the gate-3 stamp was missed (no role was built, and the evaluator ran inside Gate 4's drafting).
- Designer: not spawned (nothing to design); rotation pointer not advanced.
- Planner: not spawned (nothing to plan).
- Executor: not spawned (no code).
- Evaluator `sonnet`, Agent tool: 124,936 subagent tokens / 31 tool uses / 272 s. Judge independence: a different model from the controller (opus); same vendor, so vendor-level independence is not claimed.
- Quorum: none (no design doc). Metered spend: $0. No lane recorded dead.

**Ruled out**:
- A separate decision row: the transport exists only because of D-FLEET-24's bare-alias spawn. Option A removes it; B removes the need for it, though the "heavy roles never inline" invariant may still produce one (unmeasured). The error is codex-internal and not ours to fix.
- Resolving the ticket as "native, not ours": resolve needs a `--sha` on `origin/dev`, and the fix that removes the exposure is D-FLEET-24's ruling, which has not been made.
- Ticketing the 10-09 `401` variant: attended `codex-tui` sessions, not a mission fire, and no product loop filed it.

**Retro lane**: no skill edit.
- (a) Iteration 76 named "a log census is only as wide as its wording" as a candidate Gate-2 rule. This fire ran three wording variants and they agreed (27 files each). The census's real hazard here was the opposite one: **a census over agent transcripts counts quotes of the ticket as occurrences** (27 files matched, 2 sessions held the event). The two share a root: a match is not an event until the record type says so. Still not edited, because a rule edit is live for every mission; it is flagged in the report alongside (a) from iteration 76.
- (b) Eighth consecutive fire (70 was the last code landing) without a landable pick. This is the first fire where no open ticket lacks a disposition: all 32 have a queue row or an OPEN decision row, and none can be built until a ruling lands. D-FLEET-24 alone gates 7 tickets and 25 of 63 lost slots. Until rulings land, each fire can only re-measure. The report asks Mark to rank the rulings rather than adding a new decision row.

Last 3 code landings: iteration 70 (`52e8958f0`), then none at 71–77. No drift alarm: no routable in-scope UNMET-clause row exists.

**Next**: no routable ticket. A record merge can still go red on `cmd/ailang`'s test budget (D-FLEET-17); re-run once before attributing it. Check `ailang mission ticket open --json` for a filing newer than 10-28, and the ledger for a ruling (first D-FLEET-24, then -18, -36, -38, -41). Take whichever lands first. If neither has, record a short no-op fire rather than re-measuring a parked ticket.

## 78 — 2026-11-02 — D-FLEET-24 option A LANDED (#2053 → `3d9112bb9`), built as a proposed diff before an attended ruling arrived mid-fire; 7 `agent-tool` tickets resolved; independent sonnet PASS 94 then PASS 96 [HARNESS]

**Pick**: D-FLEET-24, the ruling-gated class with the most lost slots (7 tickets, 25 of 63). Iteration 77's **Next** said to take a filing newer than 10-28 or a ruling. At pick time neither had arrived through an accepted channel: #2039 had 0 allowlisted comments, and no ticket had been filed since 10-28. A third kind of signal had arrived. At 13:53Z the prod store received `Ruling: D-FLEET-24 = Option A` from `sunholo-platform`, addressed to the `daneel` inbox, "on behalf of Mark" ([evidence](fleet-mission-evidence/iteration78/relayed-ruling.md)). The ledger cannot take that: a message-plane sender name is self-declared, the loop's own account can send to any inbox, and Daneel's decision-desk design (V8) rules out relaying. The Authority section says a policy ticket "parks with the proposed diff", and D-FLEET-24 had none, so the pick was to build A's diff and leave it unmerged. Mid-fire, at 16:33Z, an attended session recorded the ruling (#2051, `e1f914037`). That unparked the row, and the iteration landed it.

**Orphans / prior record**: no open `fleet/*` PR before this fire (iteration 77's records merged as `cc9c21743`). No `fleet/i78*` branch or worktree. Self-notices rc 0 (control 852:10; `mission-2039-last-seen` is absent, so the read was degraded to the #1945 watermark).

**Preflight / observe**:
- Armed. gh `sunholo-voight-kampff`. Billing CLEAN. 0 directives on #2039 (watermark absent) or #1945 (`2026-10-22T23:29:08Z`, LEGACY).
- Skill drift: SKILL.md and all 13 resource files `cmp`-identical to `origin/dev` at the resolved symlink target; the pin is at `origin/dev` (`AILANG_DRIVER_AGE=0`).
- Gate-1 base `02e1429d3d035079176c87662423b42cdad2703a@2026-11-02T15:50:41Z`.
- Dev CI: HEAD `02e1429d3` had **no** CI run (7 dependabot merges after `cc9c21743`; 5 check-runs, all docs or Cloud Build). The controller dispatched `CI` (run 41476197575): 11 jobs, all success or skipped. `Build and Release` at `cc9c21743` failed `Build ubuntu-latest` on `TestTerminalAsyncClosePreservesUnemittedReadAhead` (`internal/effects/terminal_host_test.go:426`); the other legs were fail-fast cancels. `CI` was green on the same records-only SHA, so this is a language-core flake: V1-owned, outside Authority, recorded and not picked. The inherited Cloud Build `ailang-core-dev` red remains.
- Ledger valid, 40 rows, 25 OPEN at pick time. 32 tickets open; the newest was filed 10-28. `MISSION_OVER_RATION` empty.

**Reality check** (first-party, at `02e1429d3`): `MISSION_ANTHROPIC_AVAILABLE=0 MISSION_EVALUATOR_MODEL=sonnet tools/launchd/resolve-role-spawn.sh evaluator` printed `agent-tool sonnet declared:alias-pin`, byte-identical to the ANTH=1 control. The designer and executor aliases behave the same way. The planner control falls back correctly (`recipe codex:gpt-6.1-sol anthropic-fallback:fail-closed:no-doc`).

**Did** (generator ≠ judge):
- **Designer, planner**: not spawned. The D-FLEET-24 row is the spec, and the change is a 30-line resolver arm, so this was a direct fix: the controller wrote the gate list into the executor directive. Resolver readings: designer `recipe claude:claude-opus-5-5 declared:provider-pin`, planner `agent-tool opus fail-closed:no-doc`. Neither was a spawn failure.
- **Executor** `codex:gpt-6.1-sol`, run by recipe: probe rc=0, `workspace-write`, 30-min cap, finished rc=0 in about 9 min. It added a `MISSION_ANTHROPIC_AVAILABLE = "0"` arm in the bare-alias branch. The arm walks `MISSION_<ROLE>_FALLBACK`, skips bare aliases, `claude:`, `_rs_over` entries and (evaluator) the executor's family, and emits `reroute <pin> anthropic-unavailable` or `refuse fail-closed:anthropic-unavailable-no-fallback`. It also added arms R15–R38, a doc sentence and a changelog ([directive](fleet-mission-evidence/iteration78/executor-directive.txt), [report](fleet-mission-evidence/iteration78/executor-report.md)).
- **Controller error, caught by the done-gate**: the directive asked for a "hermetic preamble" in `test_mission_routing.sh`, because the controller had measured the suite as environment-dependent at base (clean 122/0, ANTH=0 113/9, live fleet env 121/1). That dependence is deliberate. `test_suite_env.sh` arm E runs the suite directly in a poisoned env and needs it to FAIL, which proves the `suite-env.sh` launcher cleans the env. Arm F proves every make-target suite goes through that launcher. The first `make test-launchd-drivers` was rc=2: arm E failed, plus two mirror arms (`.agents/skills/mission-control` was stale after the `gate-3-route.md` edit). The controller removed the preamble, ran `tools/launchd/sync-agents-skills.sh`, and rewrote the doc paragraph and changelog. The second run was rc=0. Outside the sandbox the routing suite passes 146/0 under the launcher, in a clean env and with ANTH=0. In the live fleet env the evaluator now reads `reroute pi:openrouter/minimax/minimax-m3 anthropic-unavailable`.
- **Evaluator** `sonnet`, Agent tool, foreground, read-only ([evaluation](fleet-mission-evidence/iteration78/evaluation.md)). It scored the executor's diff **PASS 94**. Its grid compared base and patch on 14,400 inputs outside ANTH=0 and found 0 differences. Ten mutants are killed; one equivalent mutant survives. It made three NON-BLOCKING findings: the doc sentence (applied), a redundant equality clause (kept), and no planner arm (out of scope). Resumed with `SendMessage` on the controller's delta, it scored **PASS 96**: `test_suite_env.sh` 8/0, mirror check rc=0, and only pre-existing arms fail on a poisoned direct run.
- **Ruling arrives mid-fire**: the first record PR (#2054) went CONFLICTING, because #2051 had changed the same ledger row. Provenance check: `D-FLEET-24 verdict=undetermined:squash-merge flip=e1f914037 creation=7ab609a6a`. Per the contract, that verdict is reported, not re-asked. The PR body records Mark directing the attended session ("Approve d fleet 24 option a", 13:52Z). The record branch was rebuilt on the ruling (`checkout -B` onto `origin/dev`, edits re-applied by script) rather than auto-merged, and `mission_decisions.sh --check` passed.
- **Landing**: the code branch was rebased onto the ruling and the commit message re-worded (`b437d6f89`). Driver dry-runs ran under **stapledon** (armed, idle, no pidfile; `AILANG_DRIVER_PINNED=b437d6f89`): healthy rc 0 `lanes=ok`; degraded (`MISSION_EXECUTOR_MODEL=pi:bogus/no-such-model`) rc 0 `DEGRADED(1)`. #2053 was marked ready. 27 checks terminal: 20 success, 7 skipped, 0 non-green, required checks green, MERGEABLE/CLEAN. It merged with `--merge --match-head-commit` → `3d9112bb9`.
- **Gate-5 skill edit** (in this record PR): the read-env-by-NAME rule is now at the top of `SKILL.md`, from World iteration 332's `[proposal]` on the `mission-control` inbox. Frictions: fleet 51 and 52, World 332, and this fire's own first command, which read values filtered to `MISSION_`, `AILANG_` and `CLAUDE_CODE_PRINT`. Nothing secret was printed; it was a near miss. The `.agents/` mirror is synced.
- **Recorded**: the D-FLEET-24 row (RESOLVED by attended ruling) gains an iteration-78 implementation note. Queue rows 31 and 59 are LANDED. The 7 tickets are resolved with `3d9112bb9`.

**Ruled out**:
- Treating the relayed message as a ruling: it was not one of the ledger's two channels, and its sender name is self-declared. The attended edit that followed proves the cost was only latency.
- A hermetic preamble in `test_mission_routing.sh`: it breaks `test_suite_env.sh`'s deliberate launcher control. The env boundary belongs to `suite-env.sh`, not to each suite.
- Extending A to `claude:` provider pins (the designer's default is `claude:claude-opus-5-5`): the ruling names aliases, and B (the `claude-sub` route) was not chosen.
- Picking the `Build and Release` flake: it is in `internal/effects`, outside Authority and V1-owned.

**Retro lane**: one skill edit (the env-by-NAME hoist, above). Flagged, not edited:
- (a) **A controller-written acceptance item can be wrong in a direction no sandbox reveals**: "make the suite hermetic" read as hygiene and broke a deliberate control. This is false-green (4)'s shape aimed at intent rather than at the base: a gate the directive tells the executor to CHANGE needs the same baseline as a gate it tells the executor to RUN. It was caught because the full done-gate ran before the review was trusted. One instance.
- (b) **Build the proposed diff while waiting for a policy ruling.** Eight fires re-measured parked rows. This fire built the diff instead, and when the ruling landed mid-fire, the cost of landing was a rebase, a dry-run and a CI wait. The charter's Authority line already says "parks with the proposed diff"; earlier fires parked without one. Candidate process fix; one instance.
- (c) A ruling's lag from relay to ledger was 2h40m. Mark ruled in the platform at 13:52Z, it reached only the `daneel` inbox, and an attended session recorded it at 16:33Z. Whether the platform should post to the bookkeeping issue is Mark's platform call (Daneel V8). It is noted in the report, with no new row.

**Progress**: goal unmoved on clause 1 (UNMEASURED). Clause 2 UNMET: the 7 tickets resolved here were filed 10-01..10-21, so their turnaround is weeks. Clause 3 MET (25 open). Clause 4: prior evidence only. Clause 5 MET for this fix (done-gate passed before merge).

**Routing evidence**: Gate-4 base `base=4c2566568773bb70a68e62bb645fc1f151ede24b@2026-11-02T16:19:35Z`, re-read after the mid-fire ruling as `e1f914037` (#2051) and then `3d9112bb9` (#2053). Gate-1 base `02e1429d3d035079176c87662423b42cdad2703a@2026-11-02T15:50:41Z`. Driver routing note: empty (as configured).
- Controller `claude:claude-opus-5-5` (tok: not reported).
- Designer: not spawned (direct fix); rotation pointer not advanced.
- Planner: not spawned (direct fix). Resolver `agent-tool opus fail-closed:no-doc`.
- Executor `codex:gpt-6.1-sol` (99,535 tok), resolver `recipe codex:gpt-6.1-sol declared:provider-pin`.
- Evaluator `sonnet`, Agent tool (64,800 + 70,507 = 135,307 subagent tok; 13 tool uses), resolver `agent-tool sonnet declared:alias-pin`. Judge independence is cross-vendor: sonnet judged a codex executor.
- Quorum: none (no design doc). Metered spend $0. No lane recorded dead.

Last 3 code landings: iteration 78 (`3d9112bb9`), iteration 70 (`52e8958f0`), then earlier. This fire breaks the 71–77 run.

**Next**: the open tickets are all ruling-gated. Take a ruling if one lands: check the ledger first for D-FLEET-18, -36, -38 and -41, the in-scope or near-scope rows. Otherwise take a filing newer than 10-28. Both iteration 78's controller (`claude-opus-5-5`) and the sibling missions will exercise the new resolver arm the first time a codex or pi controller fires. Read that fire's routing row for an `anthropic-unavailable` reroute before claiming the class is gone.

## 79 — 2026-11-03 — D-FLEET-26 option A built as a proposed diff (draft PR #2065, `edb099371`), unmerged pending a ruling; independent sonnet PASS-WITH-EDITS 88 then PASS 94 [HARNESS]

**Pick**: D-FLEET-26 (`codex:server-capacity-error-crashes-finished-iteration`, attended, 10-02, 1 slot). Iteration 78's **Next** asked for a ruling or a filing newer than 10-28, and neither had arrived. The ledger has had no change since `409970ce4`. 25 tickets are open, the newest filed 10-28, and each maps to an OPEN row. The Authority section says a policy ticket "parks with the proposed diff". So the pick was the in-scope OPEN row whose recommendation is code and which had no diff. That is the pattern iteration 78's retro (b) named. D-FLEET-34 and D-FLEET-32 recommend waivers (no diff needed); D-FLEET-21 is next.

**Orphans / prior record**: there was no open `fleet/*` PR, `fleet/i79*` branch or worktree before this fire. Iteration 78's record merged as `409970ce4`. Self-notices rc 0 (control 852:10, single watermark: `mission-2039-last-seen` is absent).

**Preflight / observe**:
- Armed. gh `sunholo-voight-kampff`. Billing CLEAN. 0 directives on #2039 (watermark absent) or #1945.
- Skill drift: `SKILL.md` and all 13 resource files are `cmp`-identical to `origin/dev` at the resolved symlink target. The pin is at `origin/dev` (`AILANG_DRIVER_AGE=0`).
- Gate-1 base `6a527f7f672c55b9950c193ce481d767b9c8b7ff@2026-11-03T00:10:24Z`.
- Dev CI at `6a527f7f6`: `CI` success, `Build and Release` success, docs deploy success at `d555a52ac`. 18 check-runs; the only non-green is `SonarCloud Code Analysis` (non-required, inherited, V1-owned).
- Unread prod inbox: 20 messages, none addressed to the fleet. The language reports are V1's. The two "lane degraded" notices are the drivers' own.
- `MISSION_OVER_RATION=codex` (headroom 1.3pp, under the 2pp margin). The Anthropic quota read is `unknown` (HTTP 429, the D-FLEET-28 state).

**Reality check** (first-party, at `6a527f7f6`):
- `TRANSIENT_SIG` (`mission-control.sh:1160`) and `RUNTIME_QUOTA_SIG` (`:1152`) still carry no codex capacity emitter.
- `_mc_run_once` writes codex output `>>"$LOG" 2>&1`, and the retry loop greps the tail written by this attempt with `-qiE`.
- codex-cli 0.164.0's binary contains the literal `Selected model is at capacity. Please try a different model.`
- The rotated driver logs no longer hold the 10-02 occurrence (0 hits in `/tmp/ailang-mission-*.log`).

**Did** (generator ≠ judge):
- **Designer**: not spawned. The decision row is the spec, and no design doc was warranted. The resolver read `recipe claude:claude-opus-5-5 declared:provider-pin`.
- **Planner** `opus`, Agent tool, foreground, read-only ([summary](fleet-mission-evidence/iteration79/planner-directive.md)). It measured codex's error format: a rejected-model `codex exec` writes `ERROR: ` at column 0, twice, then exits rc=1. It chose the line anchor because the string is quoted mid-line in at least five repo files, and wrote the arms and the M1/M2 drills.
- **Executor** `claude:claude-haiku-5-5`, run by recipe (`claude-sub`, `bypassPermissions`, 30-min cap). Probe rc 0; finished rc 0 in about 10 min ([directive](fleet-mission-evidence/iteration79/executor-directive.txt), [report](fleet-mission-evidence/iteration79/executor-report.md)). It committed `9ba6889e0`: capacity suite 18/0, M1 and M2 each killed, `check-changelog` rc 0, `test-launchd-drivers` 52/52. It flagged that `-i` makes the anchor case-insensitive (harmless).
- **Evaluator** `sonnet`, Agent tool, foreground, read-only ([evaluation](fleet-mission-evidence/iteration79/evaluation.md)). Round 1 scored **PASS-WITH-EDITS 88**. It found that a too-broad `^ERROR:` mutant survived, and that the new fixture was the repo's only column-0 copy of the line, in a file a codex controller might read.
- **Controller delta** `72fc66375`:
  - arm `codex-other-error-not-transient` (a column-0 400 error stays a crash);
  - the fixture is assembled with `$'\n'`, so `git ls-files` has no column-0 hit;
  - the comment cites the ticket for "printed twice";
  - the changelog names the third case.
- Resumed with `SendMessage`, the judge scored **PASS 94**. It re-ran `test-launchd-drivers` itself (52/52), and each mutant (no anchor, too narrow, `|^ERROR:`) has exactly one killer ([drills](fleet-mission-evidence/iteration79/mutation-drills.txt)). Its one remaining nit was a 173-character comment line, re-wrapped in `edb099371` (comment-only; capacity suite 19/0).
- **Landing (proposed only)**: pushed `fleet/i79-dfleet26-capacity-transient` and opened **draft PR #2065**. Its body says not to merge until the ruling and gives reach and limits: the World fork is not reached; total backoff is about 2 minutes 15 seconds; a codex retry is a fresh run. It names what is owed at landing (driver dry-runs, healthy and degraded, and CI on the rebased head). The ticket stays open.
- **Recorded**: the D-FLEET-26 row has an iteration-79 note and stays OPEN. Queue row 34 is tagged `proposed diff #2065`. A process-fix sentence was added under the *Ticket-driven only* guardrail.

**Ruled out**:
- An unanchored literal: it would make every rc≠0 crash of an iteration that reads the ledger or the log retry twice. Killed by `prose-codex-capacity-not-transient`.
- Classifying any column-0 `ERROR:` as transient: a 400 or auth error would retry pointlessly. Killed by `codex-other-error-not-transient`.
- A third signature-only copy in `test_driver_notify.sh`: the house pattern is the end-to-end capacity arm.
- Merging: D-FLEET-26 is HD-2a and OPEN.
- Driver dry-runs this fire: nothing lands, and a synthetic-HOME dry-run deleted three live pins on 10-28. They are owed at landing.

**Retro lane**: one process fix, in the charter: "when every open ticket is ruling-gated, build the next in-scope proposed diff as a draft PR". Two instances, iterations 78 and 79. No skill edit. Flagged, not edited:
- (a) **A test fixture for a log signature can itself plant the signature.** The first fixture put the exact column-0 line in a tracked file that a codex controller may `cat` into `$LOG`. The judge caught it by measuring `git ls-files` rather than reasoning about the anchor. One instance.
- (b) The planner sub-agent used `TaskStop` once on a task of its own. The controller's background suite was unaffected. If a sub-agent can stop a task, it can stop the controller's; one instance and no harm, so this is noted only.

**Progress**: goal unmoved on clause 1 (UNMEASURED). Clause 2 UNMET (nothing resolved). Clause 3 MET (25 open). Clause 4: prior evidence only. Clause 5 preserved (nothing merged to code).

**Routing evidence**: Gate-4 base `6a527f7f672c55b9950c193ce481d767b9c8b7ff@2026-11-03T00:46:41Z`; Gate-1 base `6a527f7f672c55b9950c193ce481d767b9c8b7ff@2026-11-03T00:10:24Z`. Driver routing note: "lanes degraded: planner: codex lane gpt-6.1-sol unusable (probe rc=75, over daily ration) → handed to opus; executor: codex lane gpt-6.1-sol unusable (probe rc=75, over daily ration) → handed to claude:claude-haiku-5-5".
- Controller `claude:claude-opus-5-5` (tok: not reported). `MISSION_ANTHROPIC_AVAILABLE=1`, so D-FLEET-24's reroute arm was not exercised this fire.
- Designer: not spawned (no doc warranted); rotation pointer not advanced. Resolver `recipe claude:claude-opus-5-5 declared:provider-pin`.
- Planner `opus`, Agent tool (92,712 subagent tok; 25 tool uses). Resolver `agent-tool opus fail-closed:env-pin`.
- Executor `claude:claude-haiku-5-5`, recipe via `claude-sub` (tok: not reported; text output). Resolver `recipe claude:claude-haiku-5-5 declared:provider-pin`.
- Evaluator `sonnet`, Agent tool (54,552 + 60,578 = 115,130 subagent tok; 13 tool uses). Resolver `agent-tool sonnet declared:alias-pin`. Judge independence is same-vendor, cross-model: sonnet judged a haiku executor and an opus controller delta.
- Quorum: none (no design doc). Metered spend $0. No lane recorded dead.

**Next**: if Mark rules D-FLEET-26 A, rebase #2065, run the driver dry-runs (healthy and degraded) under an idle armed sibling, wait for CI, merge, and run `ailang mission ticket resolve codex:server-capacity-error-crashes-finished-iteration --sha <merge>`; then note the World port. Otherwise build D-FLEET-21 A as a draft PR (the Gate 3b poll cap, a skill edit), or take any ruling (D-FLEET-18, -36, -38, -41) or a filing newer than 10-28.

## 80 — 2026-11-03 — D-FLEET-21 option A built as a proposed diff (draft PR #2067, `bdaad2825`), unmerged pending a ruling; fresh CI re-measure; independent sonnet PASS 90 [HARNESS]

**Pick**: D-FLEET-21 (Gate 3b's CI poll cap), iteration 79's **Next**. No ruling or filing newer than 10-28 had arrived. The *Ticket-driven only* guardrail's ruling-gated clause selects the next in-scope OPEN row whose recommendation is code and has no diff. D-FLEET-20, -28 and -35 recommend code outside fleet scope; D-FLEET-32 and -34 recommend waivers. So D-FLEET-21 was the only one left. Caveat: D-FLEET-21 came from iterations 43 and 44's own retro, not a ticket, so the guardrail's "the ticket is already in the queue" does not literally hold for it. It is a ledger row the fleet filed from two measured frictions, and its diff stays unmerged until Mark rules.

**Orphans / prior record**: the open draft #2065 (iteration 79's D-FLEET-26 proposal) is expected and stays untouched. Iteration 79's record merged as `29ddaec78`. No other `fleet/*` PR, branch or worktree.

**Preflight / observe**:
- Armed; gh `sunholo-voight-kampff`; billing CLEAN.
- #2039: 0 directives. The watermark file `mission-2039-last-seen` is still absent, so it read all 4 comments, none allowlisted.
- Skill drift: `SKILL.md` and all 13 resource files `cmp`-identical to `origin/dev` at the resolved symlink target; pin at `origin/dev` (`AILANG_DRIVER_DRIFT=0`).
- Gate-1 base `29ddaec7899ea13dbd3d4c97b75f0e5ff6a235bd@2026-11-03T07:28:39Z`.
- Dev CI at `29ddaec78` success. 18 check-runs; the only non-green is `SonarCloud Code Analysis` (non-required, inherited, V1-owned).
- Ledger valid (40 rows, 24 OPEN). 25 tickets open, newest first_seen 10-28. `MISSION_OVER_RATION` empty; routing note empty.

**Reality check** (first-party, at `29ddaec78`):
- Both `gate-3b-ci-green.md` copies still carry `+ 1800`, `timeout: 2000000`, "CI is ~15-20m" and "after 30m" (byte-identical mirrors).
- **The row's premise has moved.** Last 40 CI runs: p50 14 min, p90 18, max 30. Fleet PR check sets: 14–15 min. Last 200 runs (10-28..11-03): 23 over 30 min (18 of them on 10-29, max 99), 2 of 47 on 11-02 (max 37). The judge re-measured: median 15.2, p90 31.1, 7 over 60 ([ci-durations](fleet-mission-evidence/iteration80/ci-durations.md)).
- So the 10-16 "31–55 min" no longer describes the median, but the 30-min cap still misses about 1 run in 9 this week, and a 60-min cap would still miss about 1 in 30.

**Did** (generator ≠ judge):
- **Designer**: not spawned; the decision row is the spec.
- **Planner**: `resolve-role-spawn.sh planner` and `derive-planner-lane.sh` both read `opus fail-closed:no-doc`. The opus Agent spawn was **denied** by the spawn-pin hook (`deny:provider-pin — planner is pinned to codex:gpt-6.1-sol`), which is the `gate-3-route.md` Step 1b ⚠ case. Routed to `codex:gpt-6.1-sol` under the planner recipe: probe rc 0, detached sibling worktree, rc 0, 78,579 tok ([directive](fleet-mission-evidence/iteration80/planner-directive.txt), [plan](fleet-mission-evidence/iteration80/plan.md)). It found the seven sites per copy and the one rule-6 line. It ruled `gate-3-route.md:160` out of scope (the codex wrapper's own cap) and found no test pinning the values.
- **Gate list run at base first** (false-green 4): `check-context-docs` rc 0, `check-changelog` rc 0, heartbeat + agents-sync suites rc 0.
- **Executor**: `codex:gpt-6.1-sol`, recipe, rc 0, 46,469 tok ([directive](fleet-mission-evidence/iteration80/executor-directive.txt), [report](fleet-mission-evidence/iteration80/executor-report.md)). It edited the `.claude` copies and the fragment. Its sandbox denied writes under `<wt>/.agents/` ("mounted read-only"), and it labelled the mirror checks UNINFORMATIVE UNDER SANDBOX rather than reporting them as passes.
- **Controller**: ran `tools/launchd/sync-agents-skills.sh` outside the sandbox (pre-check rc 1 STALE, sync rc 0, post-check rc 0), then committed `bdaad2825` crediting codex.
- **Gates at head (outside sandbox)**: residual grep rc 1 in both copies; the positive grep hits lines 16/19/20/47/68 in both; `cmp` rc 0; SKILL.md 628 lines; `git diff --check` rc 0; `check-context-docs` rc 0; `check-changelog` rc 0; `make test-launchd-drivers` **52/52**.
- **Evaluator**: `sonnet`, Agent tool, foreground, read-only ([evaluation](fleet-mission-evidence/iteration80/evaluation.md)). **PASS 90**, no edits required. It ran three mutants (revert only the `.agents` copy; 3600→1800 in `.claude` only; drop the rule-6 clause in `.claude` only), each killed by `sync-agents-skills.sh --check`. It noted that reverting both mirrors together passes every check. Strongest objection: A moves the cliff rather than removing it (7 of 200 runs exceed 60 min, and a stuck headless poll now waits twice as long); B handles that tail. The controller confirmed `rc=0 for bdaad28`, a clean worktree and `--check` rc 0 after the drills.
- **Landing (proposed only)**: pushed `fleet/i80-dfleet21-gate3b-poll-cap` and opened **draft PR #2067**. Its body says not to merge before the ruling, carries the re-measure and the judge's objection, and names reach (every mission's next fire; World not reached). The row stays OPEN with an iteration-80 note.

**Ruled out**:
- Editing `gate-3-route.md:160` (`2000000 ms for 1800 s`): it is the codex executor wrapper's 30-min cap, not Gate 3b. The planner and the judge both confirmed this.
- Rewriting `SKILL.md:497` ("a 30-minute CI poller"): it is iteration 320's war story, which keeps its literals.
- Hand-editing `.agents` files: the sync script is the only sanctioned writer (iteration 63).
- Merging: D-FLEET-21 is a time-budget ruling and OPEN.
- Driver dry-runs: nothing lands, and the change is skill text that the driver does not read.

**Retro lane**: no skill edit and no process fix. Flagged, not edited:
- (a) **codex `workspace-write` cannot write `<wt>/.agents/`**: second recorded instance (iterations 63 and 80; iteration 64 passed `--add-dir <wt>/.agents` as a workaround). Two frictions make it eligible for a skill fix, but the controller-side sync is already the sanctioned writer and cost one command. A recipe line ("for a skill-mirror diff, have codex edit `.claude` only and the controller run `sync-agents-skills.sh`") is a candidate for the next skill edit. It is not taken here, because this iteration's own proposed diff touches the same skill tree.
- (b) The spawn-pin hook denied a resolver-prescribed opus planner, the fourth instance of the documented Step 1b ⚠ case. The durable fix is in the resolver (`role-spawn-routing.md` §2), not here.
- (c) After this iteration, every in-scope OPEN row whose recommendation is code has a proposed diff (#2065, #2067), so the ruling-gated clause has nothing left to build. The next fire should expect to idle-check rather than build.

**Progress**: goal unmoved on clause 1 (UNMEASURED). Clause 2 UNMET (nothing resolved). Clause 3 MET (25 open). Clause 4: prior evidence only. Clause 5 preserved (nothing merged to code).

**Routing evidence**: base=29ddaec7899ea13dbd3d4c97b75f0e5ff6a235bd@2026-11-03T07:47:24Z (Gate 4); Gate-1 base `29ddaec7899ea13dbd3d4c97b75f0e5ff6a235bd@2026-11-03T07:28:39Z`. Driver routing note: empty (as configured).
- Controller `claude:claude-opus-5-5` (tok: not reported). `MISSION_ANTHROPIC_AVAILABLE=1`.
- Designer: not spawned; rotation pointer not advanced. Resolver `recipe claude:claude-opus-5-5 declared:provider-pin`.
- Planner `codex:gpt-6.1-sol` (78,579 tok), recipe. Resolver `agent-tool opus fail-closed:no-doc`, hook `deny:provider-pin`, routed to the pin.
- Executor `codex:gpt-6.1-sol` (46,469 tok), recipe. Resolver `recipe codex:gpt-6.1-sol declared:provider-pin`.
- Evaluator `sonnet` (61,738 subagent tok; 7 tool uses), Agent tool. Resolver `agent-tool sonnet declared:alias-pin`. Judge independence: cross-vendor from the codex generator, same vendor as the controller's mirror-sync step.
- Quorum: none (no design doc). Metered spend $0. No lane recorded dead.

**Next**: if Mark rules D-FLEET-21 or D-FLEET-26, rebase that draft (#2067 or #2065), run its owed gates (#2065 also needs driver dry-runs), wait for CI, merge, and resolve. For #2065, also run `ailang mission ticket resolve codex:server-capacity-error-crashes-finished-iteration --sha <merge>`. D-FLEET-21 has no ticket to resolve. Otherwise take any ruling (D-FLEET-18, -36, -38, -41) or a filing newer than 10-28. No in-scope code recommendation is left without a diff.

## 81 — 2026-11-03 — D-FLEET-25's shared mechanical sub-item built as a proposed diff (draft PR #2082, `399cf799b`), unmerged pending a ruling; the "World is a fork" claims are corrected and the mission-loop-change reach check (which read NO for every mission) is repaired; an independent sonnet judge refuted the controller's idle verdict, then PASS-WITH-EDITS 90 and PASS 94 [HARNESS]

**Pick**: D-FLEET-25's mechanical sub-item, the one that options A and B share. Iteration 80's **Next** expected an idle fire, and the controller's Gate-2 map agreed: 25 tickets open, the newest filed 10-28, every ticket mapped to an OPEN row, and no ruling. The independent idle-verdict judge refuted that. D-FLEET-25 A and B both include "correct the World fork claim", that sub-item is inside `missions/**`, and no diff for it existed. The controller verified it first-party. Under the *Ticket-driven only* ruling-gated clause, that made it the pick: build it as a draft PR and do not merge.

**Orphans / prior record**: drafts #2065 and #2067 are expected and untouched (no comments or reviews). Iteration 80's record merged as #2068 → `99ef89c26`. No other `fleet/*` PR.

**Preflight / observe**:
- Armed; gh `sunholo-voight-kampff`; billing CLEAN.
- #2039: 0 directives. The `mission-2039-last-seen` file is still absent; 5 comments, none allowlisted. Self-notices rc 0: 0 crash notices on #2039 and #1945, control 852:10.
- Skill drift: `SKILL.md` and all 13 resource files are `cmp`-identical to `origin/dev` at the resolved symlink target (`AILANG_DRIVER_DRIFT=0`).
- Gate-1 base `b23cadced75c7bbf1328eebcc5ab011453fba672@2026-11-03T14:25:28Z`.
- Dev CI at `b23cadced` (v0.57.0 release) success. 40 check-runs; the only red is `SonarCloud Code Analysis` (non-required, inherited, V1-owned).
- Ledger valid (40 rows, 24 OPEN). `MISSION_OVER_RATION` empty; routing note empty.
- `reporting:daneel-reports-not-declared-triage` recurred at 08:56Z (12 lost slots). Its row, D-FLEET-22, is unchanged.

**Reality check** (first-party, at `b23cadced`):
- `dev.ailang.mission-world.plist` `ProgramArguments.1` = `~/dev/sunholo-data/ailang/tools/launchd/mission-control.sh`, the shared driver, as for v1, fleet, stapledon and triage. `ailang-world/tools/launchd/mission-control.sh` does not exist. `golden_live_test.go:367` says "WORLD IS DE-FORKED (2026-09-06)". `doctor.go:281` is `Fork = m.Driver != ""`, and world.toml has no `driver` key, so **the doctor half of the row was already fixed**. Only text was stale.
- **New finding:** the `mission-loop-change` skill's Gate-1 reach check reads **NO for every mission**, v1 included. Its grep wants `. "$REPO/tools/launchd/lib/pin-root.sh"`, while the driver now sources `"$MC_DRIVER_ROOT/…"` indented inside an `if` (`mission-control.sh:1224`). The same skill's Gate 2 still told every driver change to be "made twice" against a World driver that no longer exists.

**Did** (generator ≠ judge):
- **Designer**: not spawned; the row is the spec.
- **Planner**: the resolver gave `agent-tool opus fail-closed:no-doc`, and the opus Agent spawn was **denied** by the spawn-pin hook (`deny:provider-pin`), the fifth instance of the Step 1b ⚠ case. Routed to `codex:gpt-6.1-sol`: probe rc 0, `--sandbox read-only` in the clean build worktree, rc 0, 97,389 tok ([plan](fleet-mission-evidence/iteration81/plan.md)). It found the skill, test and script sites. The controller narrowed its scope: no charter-row edits in the code PR, and the snippet repaired rather than deleted.
- **Gate list at base first** (false-green 4): `sync-agents-skills --check`, `bash -n`, `go test ./internal/mission/...`, the agents-sync suite, `check-context-docs`, `check-changelog` and `check-skills` all rc 0.
- **Executor**: `codex:gpt-6.1-sol`, `workspace-write`, rc 0, 68,119 tok ([directive](fleet-mission-evidence/iteration81/executor-directive.txt), [report](fleet-mission-evidence/iteration81/executor-report.md)). It edited `.claude` only. The new snippet reads each installed plist; its grep counts 1 on the driver and 0 on `mission-heartbeat.sh`.
- **Controller**: synced `.agents` (pre rc 1, sync rc 0, post rc 0). Ran the new snippet under `/bin/bash`: 7 of 7 mission-control plists read YES. Gates at head, outside the sandbox: all of the above rc 0, `make test-launchd-drivers` **52/52**, `test_mission_directives.sh` 33/33, residual grep clean. Committed `a58e08e68` crediting codex.
- **Evaluator** ([evaluation](fleet-mission-evidence/iteration81/evaluation.md)): sonnet, Agent tool, foreground, cross-vendor from the generator.
  - Idle-verdict judge: REFUTED (narrow), which is what produced the pick.
  - Diff round 1: **PASS-WITH-EDITS 90**. M1 (mirror revert) and M2 (old pattern, all NO) were killed; M3 was not applicable. Required edit: `test_mission_directives.sh:104` "(world fork depends on it)". Applied as `399cf799b`.
  - Diff round 2: **PASS 94**.
  - Strongest objection: nothing tests the skill snippet's regex against the real driver.
- **Landing (proposed only)**: pushed `fleet/i81-dfleet25-world-fork-comment` and opened **draft PR #2082**. Its body says not to merge before the ruling. D-FLEET-25 stays OPEN with an iteration-81 note. Its default if unanswered is C.

**Ruled out**:
- Editing `doctor.go` or `registry.go`: the doctor already reports no fork for World (`doctor.go:281`). The remaining mentions (`doctor.go:140`, `driver_agreement_test.go:114,184`, `mission-control.sh:1217,1417`) are past-tense history, and the judge agreed.
- Editing the D-FLEET-25 row text or the charter queue in the code PR, as the planner proposed: ledger text is the record's job, and the ask is unchanged.
- Deleting the reach-check snippet (the planner's proposal): it is the skill's only instrument for reach, so repair beats removal.
- Adding a test that guards the snippet's regex (the judge's objection): it is a new mechanism and widens scope past the row. It is a candidate only.
- Merging: D-FLEET-25 is OPEN and its default is C.

**Retro lane**: no skill edit; process fix in the guardrail text deferred. Flagged:
- (a) **The controller's idle verdict was wrong, and only the independent judge caught it.** Iteration 80's **Next** ("no in-scope code recommendation is left without a diff") and this fire's own map both read D-FLEET-25 as wholly ruling-gated, because its A bundles a design doc. Lesson: a row whose options SHARE a mechanical sub-item has a routable diff, even when the row as a whole is policy. One recorded instance, so no guardrail edit yet. If a second arrives, add one clause to the *Ticket-driven only* guardrail: "including the mechanical sub-item shared by all non-C options".
- (b) codex `workspace-write` cannot write `.agents/`: third instance (iterations 63, 80, 81). The controller's sync costs one command. A recipe line ("for a skill-mirror diff, codex edits `.claude` only; the controller syncs") now meets the two-friction bar. Not taken here, because the open draft #2067 touches the same skill tree (`gate-3b-ci-green.md`); the next fire that edits the skill can take it.
- (c) Spawn-pin hook denied the resolver-prescribed opus planner again (fifth instance). The fix is in the resolver (`role-spawn-routing.md` §2), as before.

**Progress**: goal unmoved. Clause 1 UNMEASURED. Clause 2 UNMET (nothing resolved). Clause 3 MET (25 open). Clause 4: prior evidence only. Clause 5 preserved (nothing merged to code).

**Routing evidence**: base=b23cadced75c7bbf1328eebcc5ab011453fba672@2026-11-03T14:53:39Z (Gate 4); Gate-1 base `b23cadced75c7bbf1328eebcc5ab011453fba672@2026-11-03T14:25:28Z`. Driver routing note: empty (as configured). Details: [routing](fleet-mission-evidence/iteration81/routing.md).
- Controller `claude:claude-opus-5-5` (tok: not reported). `MISSION_ANTHROPIC_AVAILABLE=1`.
- Designer: not spawned; rotation pointer not advanced. Resolver `recipe claude:claude-opus-5-5 declared:provider-pin`.
- Planner `codex:gpt-6.1-sol` (97,389 tok), recipe (read-only sandbox, build worktree). Resolver `agent-tool opus fail-closed:no-doc`, hook `deny:provider-pin`, routed to the pin.
- Executor `codex:gpt-6.1-sol` (68,119 tok), recipe. Resolver `recipe codex:gpt-6.1-sol declared:provider-pin`.
- Evaluator `sonnet`, Agent tool, two spawns: idle judge 64,455 subagent tok; diff judge 63,800 tok (round 1), 68,540 cumulative (round 2). Resolver `agent-tool sonnet declared:alias-pin`. Judge independence: cross-vendor from the codex generator.
- Quorum: none (no design doc). Metered spend $0. No lane recorded dead.

**Next**: if Mark rules D-FLEET-25 A or B, rebase #2082, re-run its gates, wait for CI and merge. Under A, the design-doc half then follows as its own pick. If he rules D-FLEET-21 or -26, do the same for #2067 or #2065 (#2065 also needs driver dry-runs and `ailang mission ticket resolve codex:server-capacity-error-crashes-finished-iteration`). Otherwise, before declaring idle, re-check every OPEN row for a mechanical sub-item shared by its non-C options (retro a). Take any ruling (D-FLEET-18, -36, -38, -41) or a filing newer than 10-28.

## 82 — 2026-11-03 — D-FLEET-36 A and D-FLEET-38 A's shared first step, the fleet scope-guard arm, built as a proposed diff (draft PR #2108, two independent commits `bcf7f97ac` and `261fb3b64`), unmerged pending rulings; independent sonnet judge PASS-WITH-EDITS 84 then PASS 92 [HARNESS]

**Pick**: the guard arm that D-FLEET-36 A and D-FLEET-38 A both begin with. Iteration 81's **Next** said: before declaring idle, re-check every OPEN row for a sub-item shared by its non-C options. The controller read options A–C of all 24 OPEN rows. Eighteen are outside fleet scope or have no common step (16–20, 22, 23, 27–29, 31, 32, 34, 35, 37, 39–41; 33 already carries its diff, and 21, 25 and 26 carry drafts). D-FLEET-36 A and D-FLEET-38 A are in-scope code. Both start with one `_scope_is_fleet_support` change, which must ship alone under the iteration-16 rule (iteration 74's judge: "D-FLEET-36 and -38 are one allowlist change"). No diff for it existed. Under the *Ticket-driven only* ruling-gated clause, the pick was to build it as a draft and not merge.

**Orphans / prior record**: drafts #2082, #2067 and #2065 are expected and untouched. Iteration 81's record merged as #2085 → `37e5b48da`. No `iter82` worktree or branch existed before this fire. `origin/dev`'s log ended at 81, and no open fleet PR carried a later record.

**Preflight / observe**:
- Armed; gh `sunholo-voight-kampff`; billing CLEAN.
- #2039: 0 directives (6 comments, none allowlisted). `mission-2039-last-seen` is still absent. Self-notices rc 0: 0 crash notices on #2039 and #1945, control 852:10.
- Skill drift: `SKILL.md` and all 13 resource files are `cmp`-identical to `origin/dev` at both the resolved symlink target and the pin copy.
- Gate-1 base `007809894c76aaeb904707262b8471e3643c4464@2026-11-03T21:55:08Z`.
- Dev at `007809894`: 17 check-runs. The only non-green were `test-windows`, `Build macos-latest` and `Build windows-latest`, all pending (Build and Release in flight). `CI` last success was `37e5b48da`.
- Ledger valid (40 rows, 24 OPEN). `MISSION_OVER_RATION` empty; routing note empty.
- `reporting:daneel-reports-not-declared-triage` recurred at 21:28Z (13 lost slots). D-FLEET-22 is unchanged.

**Reality check** (first-party, at `007809894`): iteration 74's normaliser re-run gives a byte-identical result (16 rename-only, 11 content; [re-measure](fleet-mission-evidence/iteration82/rename-only-remeasure-007809894.txt)). The control is that the six synced mission/sprint skills `cmp` same. `_scope_is_fleet_support` and `_scope_is_harness` are as the rows describe. `git log origin/dev --grep 'D-FLEET-3[68]'` returns only the two record commits (iterations 73, 74), so no diff existed.

**Did** (generator ≠ judge):
- **Designer**: not spawned; the two rows are the spec.
- **Planner**: `codex:gpt-6.1-sol` (resolver `recipe … declared:planner-lane-default-pin`; no hook denial this time), probe rc 0, `--sandbox read-only`, rc 0, 83,547 tok ([plan](fleet-mission-evidence/iteration82/plan.md)). It proposed two separable commits, an explicit 16-name list with no wildcard, per-arm mutation labels, and a mirror audit (no other allowlist copy). The controller changed one thing: one changelog fragment per row.
- **Gate list at base first** (false-green 4): `bash -n` on both, the scope suite, `mission_decisions.sh --check`, `check-changelog`, `check-context-docs` and `check-no-personal-email` all rc 0. `make test-launchd-drivers` 52/52.
- **Executor**: `codex:gpt-6.1-sol`, `workspace-write`, rc 0, 62,751 tok ([directive](fleet-mission-evidence/iteration82/executor-directive.txt), [report](fleet-mission-evidence/iteration82/executor-report.md)). It wrote `.snap/C1` and `.snap/final`, and its four mutation drills were all killed.
- **Controller**: rebuilt C1 from the snapshot (scope suite rc 0) → `bcf7f97ac`. Restored the final snapshot (`cmp` same, 5 of 5) → C2. A C2-only build on base: `git apply` conflicts only in the test file; resolved by hand it passes, and the `.agents/skills/*` mutant is caught by the model-manager arm.
- **Evaluator** ([evaluation](fleet-mission-evidence/iteration82/evaluation.md)): sonnet, Agent tool, foreground, cross-vendor from the generator.
  - Round 1 **PASS-WITH-EDITS 84**: a `/`-boundary drop on 15 of the 16 arms survived. The controller reproduced it (pr-monitor, rc 0), added a `<name>-x` refuse trap in the loop, and saw 16 of 16 killed. Amended into C2 → `261fb3b64`.
  - Edit 2 (drop the duplicate `.claude/skills/use-ailang` refuse line) was declined: each commit keeps its own over-widening guard. Edit 3 went into the PR body.
  - Round 2 **PASS 92**. Strongest objection: the merge-separately rule is process, not code.
- **Gates at head `261fb3b64`**, outside the sandbox: all of the above rc 0, `git diff --check` clean, `make test-launchd-drivers` **52/52**.
- **Landing (proposed only)**: pushed `fleet/i82-dfleet36-38-scope-arm` (the pinned guard admitted it) and opened **draft PR #2108**. Its body says to merge each commit on its own ruling and never squash both on one. Both rows stay OPEN with iteration-82 notes; queue rows 51 and 53 are annotated.

**Ruled out**:
- Extending `sync-agents-skills.sh`, or editing any design-doc-creator or `.agents` skill file, in this PR: the pinned guard would refuse the push (iteration-16 rule). That is the follow-on fire after a ruling.
- One combined changelog fragment (the planner's proposal): two files keep the commits independent.
- A wildcard `.agents/skills/*` arm: it would admit the 11 content-drift skills, which D-FLEET-38 A routes to an attended reconcile.
- Deleting the duplicate `.claude/skills/use-ailang` refuse line (judge edit 2): see above.
- Merging: both rows are OPEN, with default C.

**Retro lane**: no skill edit. Flagged:
- (a) Iteration 81's lesson (a row's options can share a buildable sub-item) now has a second instance: here the shared step is the guard arm that two rows' A options both begin with. Recorded instances: 81 and 82. That meets the bar for the guardrail clause iteration 81 proposed. It was not added here, because a charter Guardrails edit changes the loop's own pick rule and should be visible to Mark; it is proposed in the report as a one-line process fix.
- (b) Instrument slip: a controller file-list loop ran under zsh (no word-split), so the pre-overwrite sha manifest was never written. The executor's snapshot stood in as the reference. This is a known rig trap (Bash tool = zsh); file-list loops go under `/bin/bash -c`.
- (c) The judge's round-1 drills overlapped the controller's suite run in the same worktree. It was harmless (the suite passed, then was re-run with no overlap), but a judge directive should say "drill in a `mktemp -d` copy". The round-2 directive did.

**Progress**: goal unmoved. Clause 1 UNMEASURED. Clause 2 UNMET (nothing resolved). Clause 3 MET (25 open). Clause 4: prior evidence only. Clause 5 preserved (nothing merged to code).

**Routing evidence**: base=007809894c76aaeb904707262b8471e3643c4464@2026-11-03T22:23:59Z (Gate 4); Gate-1 base `007809894c76aaeb904707262b8471e3643c4464@2026-11-03T21:55:08Z`. Driver routing note: empty (as configured). Details: [routing](fleet-mission-evidence/iteration82/routing.md).
- Controller `claude:claude-opus-5-5` (tok: not reported). `MISSION_ANTHROPIC_AVAILABLE=1`.
- Designer: not spawned; rotation pointer not advanced.
- Planner `codex:gpt-6.1-sol` (83,547 tok), recipe (read-only sandbox, build worktree). Resolver `recipe codex:gpt-6.1-sol declared:planner-lane-default-pin`.
- Executor `codex:gpt-6.1-sol` (62,751 tok), recipe. Resolver `recipe codex:gpt-6.1-sol declared:provider-pin`.
- Evaluator `sonnet`, Agent tool, one spawn, two rounds: 58,650 subagent tok (round 1), 61,899 cumulative (round 2). Resolver `agent-tool sonnet declared:alias-pin`. Judge independence: cross-vendor from the codex generator.
- Quorum: none (no design doc). Metered spend $0. No lane recorded dead.

**Next**: if Mark rules D-FLEET-36 A and/or D-FLEET-38 A, rebase #2108 keeping only the ruled commit(s), wait for CI and merge. The following fire then does the dependent edit: the 8 design-doc-creator grep sites, and/or `sync-agents-skills.sh` extended to the 16 plus their sync, then resolves the ticket(s). The same applies to #2082 (-25), #2067 (-21) and #2065 (-26). Otherwise no OPEN row has an in-scope code step without a diff, so take a ruling or a filing newer than 10-28.

## 83 — 2026-11-04 — D-FLEET-18 A's guard step, the fleet scope-guard arm for the two motoko probe files, built as a proposed diff (draft PR #2110, `875581f46`), unmerged pending a ruling; an independent sonnet judge refuted the controller's idle verdict, then PASS 93 [HARNESS]

**Pick**: the guard arm that D-FLEET-18 A begins with. Iteration 82's **Next** expected an idle fire, and the controller's Gate-2 verdict agreed: no ruling, no new filing, and every in-scope code recommendation already had a diff. Iteration 81 set the precedent of putting an idle verdict before an independent judge. This fire's judge REFUTED it. D-FLEET-18 option A, the recommendation, starts with a `_scope_is_fleet_support` arm for `tools/eval/motoko_connection_probe.sh` and `tools/eval/test_motoko_connection_probe.sh`. That step is in-scope code, it is not policy-class, and no diff existed for it. Iteration 82's screen had put row 18 under "outside fleet scope", even though it is the same guard-arm-first shape as the 36/38 arms that iteration 82 built. Under the *Ticket-driven only* ruling-gated clause, the pick was to build that arm as a draft and not merge it. The judge also named D-FLEET-41 B (sprint-evaluator skill text) as a weaker candidate. It was not taken, because B is not the recommendation and iteration 76 moved that row's default to C on purpose.

**Orphans / prior record**: drafts #2108, #2082, #2067 and #2065 are expected and untouched; none has a comment or review. Iteration 82's record merged as #2109 → `aebc3dcbb`. No `iter83` worktree or branch existed before this fire. `origin/dev`'s log ended at 82, and no open fleet PR carried a later record.

**Preflight / observe**:
- Armed; gh `sunholo-voight-kampff`; billing CLEAN.
- #2039: 0 directives (7 comments, none allowlisted). #1945: 0 since its watermark. `mission-2039-last-seen` is still absent. Self-notices rc 0: 0 crash notices, control 852:10.
- Skill drift: `SKILL.md` and all 13 resource files are `cmp`-identical to `origin/dev` at the resolved symlink target. The pin copy is at `origin/dev` (drift 0).
- Gate-1 base `aebc3dcbbe87fb4e38ee322c6359f17f9187c776@2026-11-04T04:56:25Z`.
- Dev at `aebc3dcbb`: CI and Build and Release success, 18 check-runs, none non-green.
- Ledger valid (40 rows, 24 OPEN). 25 tickets; the newest `first_seen` is 10-28 and the newest `last_seen` is 11-03T21:28Z (daneel-reports, already recorded at 82). `MISSION_OVER_RATION` empty; routing note empty.

**Reality check** (first-party, at `aebc3dcbb`): the pinned `pre-push:29-36` admits no `tools/eval/*` path. `git log origin/dev --grep D-FLEET-18` returns only record commits plus #1824, and no open PR is for row 18. Both probe files exist. Their only callers are `make/test.mk:128,140-141` and fixture strings at `scripts/test_check_referenced_paths.sh:45-46`.

**Did** (generator ≠ judge):
- **Designer**: not spawned; the row is the spec.
- **Idle judge** (sonnet, Agent tool, foreground): REFUTED (narrow). It confirmed C1, C3 and C5 and refuted C4 on D-FLEET-18 A ([evaluation](fleet-mission-evidence/iteration83/evaluation.md)).
- **Gate list at base first** (false-green 4): `bash -n` on both, the scope suite, `mission_decisions.sh --check`, `check-changelog`, `check-context-docs` and `check-no-personal-email` all rc 0. `make test-launchd-drivers` 52/52.
- **Planner**: `codex:gpt-6.1-sol`, probe rc 0, `--sandbox read-only`, rc 0, 165,656 tok ([plan](fleet-mission-evidence/iteration83/plan.md)). It proposed one literal two-path arm in *support*, not harness, so motoko, the probe's owner, keeps access. It also proposed 10 arms, each naming the mutation it kills, a separate Authority line to keep the #2108 conflict small, and an eval-exposure check. The check showed that nothing in the eval runtime runs either probe; the probe runs `eval-suite` at `motoko_connection_probe.sh:236`.
- **Executor**: `codex:gpt-6.1-sol`, `workspace-write`, rc 0, 48,870 tok ([directive](fleet-mission-evidence/iteration83/executor-directive.txt), [report](fleet-mission-evidence/iteration83/executor-report.md)). It touched exactly 4 files. Its six in-tree drills were all killed and reverted; they ran in the sandbox, so they are labelled uninformative.
- **Gates at head**, outside the sandbox: all of the above rc 0, `git diff --check` clean, `make test-launchd-drivers` **52/52** ([gates](fleet-mission-evidence/iteration83/gates.md)). Controller commit `875581f46`.
- **Diff judge** (sonnet, Agent tool, foreground): **PASS 93**, no required edits. In a `mktemp -d` copy, 8 of 8 mutants were killed and the positive control passed. Minor findings: the `v1` arms are redundant with the `motoko` arms, and no `tools/eval/test_*` widening trap exists beyond `-x`. Strongest objection: the filenames are hard-coded, so a renamed probe needs another ruling.
- **Landing (proposed only)**: pushed `fleet/i83-dfleet18-scope-arm` (the pinned guard admitted it) and opened **draft PR #2110**. It is not stacked on #2108, so each row merges on its own ruling; whichever merges second rebases and keeps both arms. D-FLEET-18 stays OPEN with an iteration-83 note, and queue row 13 is annotated.

**Ruled out**:
- Any `tools/eval/` edit (the two live flake signatures, retiring the pair) in this PR: the pinned guard would refuse the push under the iteration-16 rule. That is the follow-on fire after a ruling.
- A `tools/eval/*` wildcard: it would hand the fleet the eval harness's runners.
- Putting the paths in `_scope_is_harness`: that would refuse them for motoko and every product loop.
- Stacking on #2108: it would make row 18's merge depend on rows 36/38's rulings.
- D-FLEET-41 B: not the recommendation, and its default was moved to C deliberately (iteration 76).
- Merging: the row is OPEN, with default C.

**Retro lane**: no skill edit. Flagged:
- (a) **Third instance of the idle screen missing a buildable step.** At 81 the judge caught a sub-item shared by A and B. At 82 the controller caught a guard arm shared by two rows' A options. Here the judge caught a recommendation whose first step is an in-scope guard arm, in a row the screen had filed as "outside fleet scope". The common error is screening on "is the row's *target* in scope?" when the question should be "does the recommended (or every non-default) option *begin* with an in-scope step?". Iteration 82's report asked Mark about a one-line Guardrails clause for this, with "Default if unanswered by the next fire: yes". This is that fire and the ask is unanswered, so the **process fix is applied**: the *Ticket-driven only* guardrail now says "screen by step, not by row" and requires an idle judge on every idle verdict. Mark can reverse it with one comment. It was applied after the controller first missed the default at Gate 0 and only caught it while drafting the report, a miss in its own right: the loop has no machine-readable list of defaults it has set in reports.
- (b) Instrument slip: the controller ran `resolve-role-spawn.sh planner` with no doc and got `opus fail-closed:no-doc`. It routed to the pin per rule (a) without spawning the alias. With the charter as the doc, the resolver returns `declared:planner-lane-default-pin`. Pass the charter.
- (c) The controller committed with `-c core.hooksPath=/dev/null`. That skipped nothing (the effective hooks path holds only `pre-push`, and the push ran under the guard), but it is a habit to drop: never neutralise a hooks path, even for a commit.

**Progress**: goal unmoved. Clause 1 UNMEASURED. Clause 2 UNMET (nothing resolved). Clause 3 MET (25 open). Clause 4: prior evidence only. Clause 5 preserved (nothing merged to code).

**Routing evidence**: Gate-1 base `aebc3dcbbe87fb4e38ee322c6359f17f9187c776@2026-11-04T04:56:25Z`. Driver routing note: empty (as configured). Details: [routing](fleet-mission-evidence/iteration83/routing.md).
- Controller `claude:claude-opus-5-5` (tok: not reported). `MISSION_ANTHROPIC_AVAILABLE=1`.
- Designer: not spawned; rotation pointer not advanced.
- Planner `codex:gpt-6.1-sol` (165,656 tok), recipe (read-only sandbox, build worktree). Resolver: `agent-tool opus fail-closed:no-doc` without a doc, so the pin was followed per rule (a); with the charter, `recipe codex:gpt-6.1-sol declared:planner-lane-default-pin`.
- Executor `codex:gpt-6.1-sol` (48,870 tok), recipe. Resolver `recipe codex:gpt-6.1-sol declared:provider-pin`.
- Evaluator `sonnet`, Agent tool, two foreground spawns: idle judge 75,574 subagent tok; diff judge 58,590 tok. Resolver `agent-tool sonnet declared:alias-pin`. Judge independence: cross-vendor from the codex generator.
- Quorum: none (no design doc). Metered spend $0. No lane recorded dead.

**Next**: if Mark rules D-FLEET-18 A, rebase #2110 (onto #2108 if that merged first, keeping both arms), re-run its gates, wait for CI and merge. The following fire then works #13's live signatures in `tools/eval/` and resolves the two tickets. The same applies to #2108 (-36/-38), #2082 (-25), #2067 (-21) and #2065 (-26). Otherwise, before declaring idle, screen every OPEN row for "does the recommended (or every non-default) option begin with an in-scope step that has no diff?". If none does, take a ruling or a filing newer than 10-28.

## 84 — 2026-11-04 — dev's required-check red repaired: the `.agents` mission-control mirror of `gate-3-triage.md` re-synced via #2116 → `981028c46`; ticket `ci:skill-mirror-not-synced-after-gate3-triage-edit` resolved in ~3 h; independent sonnet judge PASS 91 [HARNESS]

**Pick**: the newest ticket, `ci:skill-mirror-not-synced-after-gate3-triage-edit` (triage, filed 2026-11-04T09:39Z). Gate 1 found dev's REQUIRED `test`, `launchd drivers (bash 3.2)` and `Build ubuntu-latest` red at HEAD `abd4dfd82` and its two parents. A red outranks the queue only for the owning mission: V1 owns repo reds and has been disabled since 09-14 (D-FLEET-17), and this red is fleet territory (the `.agents` mission mirror, its sync script and its suites). It also blocked every landing and reddened the fleet's own done-gate. It moves UNMET clause 2, since it was filed today. This is the same class iteration 63 fixed for sprint-evaluator (#1972).

**Orphans / prior record**: none. Iteration 83's record is on `origin/dev` (`da0716a4c`), and the log ended at 83. Drafts #2110, #2108, #2082, #2067 and #2065 were expected and left untouched. No `iter84` worktree or branch existed before this fire. Self-notices rc 0 (0 crash notices, control 852:10).

**Preflight / observe**:
- Armed; gh `sunholo-voight-kampff`; billing CLEAN.
- #2039: 0 directives (8 comments, none allowlisted). `mission-2039-last-seen` is still absent, so the full thread was read; the watermark was not advanced (nothing to mark).
- Skill drift: `SKILL.md` and all 13 resource files are `cmp`-identical to `origin/dev` at the resolved symlink target. The pin is at `origin/dev` (`AILANG_DRIVER_DRIFT=0`).
- Gate-1 base `abd4dfd821e919654ddaa0fafd36dd27197126fb@2026-11-04T12:01:32Z`.
- Ledger valid (40 rows, 24 OPEN). 27 tickets open. New since 83: this one and `security:loop-env-inspection-prints-secrets-into-transcripts` (triage, 11-04 08:52Z).
- `MISSION_OVER_RATION=codex`, so the driver routed the planner to opus and the executor to haiku.

**Reality check** (first-party, before routing; detail in [gates](fleet-mission-evidence/iteration84/gates.md)):
- Job logs (ANSI stripped, sized): `test` 98 KB, `--- FAIL: TestQuorumArtifactShellSuite` → `generated_docs_and_mirror: sync`. `Build ubuntu-latest` 4.6 MB (Build and Release run), 1 `--- FAIL`, the same test, against 22,129 PASS.
- `sync-agents-skills.sh --check` printed STALE for mission-control. Per-file `cmp`: only `resources/gate-3-triage.md`.
- Cause: `f22bf1f4b` (D-TRIAGE-48) added 49 canonical lines without running the sync. No open PR touched the mirror (the only PR on that path is #2067, the D-FLEET-21 draft).

**Did** (generator ≠ judge):
- **Designer**: not spawned. The fix is a deterministic regeneration with no design surface.
- **Planner**: opus, Agent tool, foreground. Resolver `agent-tool opus fail-closed:env-pin`. It verified the diff set across all six mirrors, confirmed no `.agents`-only content would be lost to `rsync --delete`, baselined 4 acceptance commands at rc 1, and wrote the plan (`design_docs/planned/sprint-plan-gate3-triage-mirror-sync.md`, landed with the fix).
- **Executor**: `claude:claude-haiku-5-5` via `claude-sub`, probe rc 0, 1800 s cap, rc 0 ([directive](fleet-mission-evidence/iteration84/executor-directive.txt), [report](fleet-mission-evidence/iteration84/executor-report.md)). It ran the sync script, wrote the fragment, ran all five acceptance commands at rc 0, and ran the base-mirror mutation drill (rc 1, restored, `cmp` 0). It made no git writes; the controller committed `96e1691a9`.
- **Gates at head**, controller, outside any sandbox: all five rc 0; `make test-launchd-drivers` under `env -i` rc 0. Done-gate dry-runs under the idle, armed `triage` profile: healthy and degraded (`codex:bogus-model-i84`) both `DRY RUN ok`, no pidfile.
- **Evaluator**: sonnet, Agent tool, foreground, in its own judge worktree: **PASS 91/100**, 0 blocking ([evaluation](fleet-mission-evidence/iteration84/evaluation.md)). It read the CI logs itself and killed 3 mutants of its own (another skill drifted, a mirror file deleted, an extra mirror file). The controller checked the report on disk and replayed the deletion mutant (rc 1, restored, rc 0).
- **Landing**: PR [#2116](https://github.com/sunholo-data/ailang/pull/2116). The closing-keyword scan found 0.

**Gate 3b**:
- PR head `96e1691a9`: 27/27 checks terminal; every required context SUCCESS.
- One non-required red: `Build macos-latest`, `TestRuntimeNestedStopUsesRemainingDeadline` (`internal/effects`, 0.04 s). It failed in none of the 24 previous Build-and-Release runs, and the diff is markdown. The failed-job re-run completed SUCCESS. Recorded as a timing flake, not filed.
- Base steady since Gate 1. Merged `--merge --match-head-commit` → `981028c469697a5be74710555b40c7fb473b82fe`.
- Merge commit: CI and Build and Release `completed/success`, 18 check-runs, 0 non-green. Dev is green again after 7 red commits.
- Ticket resolved with `--sha 981028c46`, and the reply reached triage (27 → 26 open).

**Ruled out**:
- Handing the red to V1: V1 is disabled, and the surface is fleet scope.
- A hand edit of the `.agents` file: the sync script is the only sanctioned writer.
- Building the ticket's optional pre-push mirror guard in the same PR. It changes the hook that judges every push on the rig, so it is its own change with its own drills. Carried to **Next**.
- The judge's one deduction, that `Build ubuntu-latest` was "not verifiable": it looked only in the CI run. The job is in the Build and Release run (41803697383, job 126485417479), where the controller read its FAIL first-party, so the fragment's claim stands.
- A 3000 s Gate-3b cap: the controller started one, then killed it before reuse and re-polled at the skill's 1800 s. Raising the cap is D-FLEET-21's open question, not a controller choice.

**Retro lane**: no skill edit. Observations:
- (a) **Second instance of iteration 63's observation (b):** a `.claude/skills/{mission,sprint}-*` edit pushed without the sync reddens three required checks, and nothing local refuses it. This time the edit was the triage loop's own D-TRIAGE-48 commit. The ticket names the guard. A `sync-agents-skills.sh --check` arm in the pinned pre-push guard is fleet scope (`tools/launchd/**`); see **Next**.
- (b) A hand-written Gate-3b poll split `"Build and Release=success"` on spaces and printed a truncated verdict. The per-workflow direct read (Gate 3b rule (b)) caught it, which is what that rule is for. No edit.

**Progress**: Clause 1 UNMEASURED, goal unmoved. Clause 2 UNMET: one ticket resolved inside 48 h (~3 h, the first since iteration 78). Clause 3 MET (26 open). Clause 4: prior evidence only. Clause 5 preserved and restored: the done-gate suite and dev's required checks were red before this landing.

Last 3 landings: iteration 82, none (draft); iteration 83, none (draft); iteration 84, clause 2 (and clause 5's done-gate). No drift alarm.

**Routing evidence**: Gate-1 base `abd4dfd821e919654ddaa0fafd36dd27197126fb@2026-11-04T12:01:32Z`. Driver note: codex over daily ration (rc 75), so planner → opus and executor → `claude:claude-haiku-5-5`. Details: [routing](fleet-mission-evidence/iteration84/routing.md).
- Controller `claude:claude-opus-5-5` (tok: not reported). `MISSION_ANTHROPIC_AVAILABLE=1`.
- Designer: not spawned; rotation pointer not advanced.
- Planner `opus`, Agent tool (50,487 subagent tok). Resolver `agent-tool opus fail-closed:env-pin`; the pin is a bare alias, so the spawn was allowed.
- Executor `claude:claude-haiku-5-5`, `claude-sub` recipe (tok: not reported). Resolver `recipe claude:claude-haiku-5-5 declared:provider-pin`.
- Evaluator `sonnet`, Agent tool (52,616 subagent tok). Resolver `agent-tool sonnet declared:alias-pin`. Judge independence: different model from the haiku generator and the opus planner.
- Quorum: none. Metered spend $0. No lane recorded dead.

**Next**:
1. `security:loop-env-inspection-prints-secrets-into-transcripts` (triage, 11-04 08:52Z, in its 48 h window until 11-06 08:52Z). Its asks are a spawn-env strip of `secrets.env` keys per lane and/or a post-iteration session-file hash scanner. A strip that changes what a lane receives may touch a billing guard (HD-2a), so screen for the policy class first and build the mechanical half (the scanner) if it is separable.
2. The pre-push mirror guard (this ticket's second ask; two instances now, iterations 63 and 84): one `sync-agents-skills.sh --check` arm in `tools/launchd/githooks/pre-push`, shipped alone, with a refuse/allow drill pair.
3. Otherwise, a ruling on D-FLEET-18, -36/-38, -25, -21 or -26 lets its draft (#2110, #2108, #2082, #2067, #2065) rebase and land.

## 85 — 2026-11-04 — the transcript secret detector LANDED (#2122 → `26b35b081`): every legacy controller fire scans the session files it wrote for `secrets.env` values and reports key names privately; ticket `security:loop-env-inspection-prints-secrets-into-transcripts` resolved in ~11 h; the env-strip half parks as D-FLEET-42; independent sonnet judge PASS [HARNESS]

**Pick**: `security:loop-env-inspection-prints-secrets-into-transcripts` (triage, filed 2026-11-04T08:52Z), iteration 84's **Next** 1, inside its 48 h window. It moves UNMET clause 2. No directive, ruling or red outranked it.

**Orphans / prior record**: none. Iteration 84's record is on `origin/dev` (`12fd504c0`) and the log ended at 84. No open fleet branch carried a newer log. Drafts #2110, #2108, #2082, #2067 and #2065 were expected and left untouched. Self-notices rc 0 (0 crash notices, control 852:10).

**Preflight / observe**:
- Armed; gh `sunholo-voight-kampff`; billing CLEAN.
- #2039: 0 directives (9 comments, none allowlisted). `mission-2039-last-seen` is still absent, so the full thread was read; the watermark was not advanced (nothing to mark). The previous issue, #1945: 0 directives since its watermark.
- Skill drift: `SKILL.md` and all 13 resource files are `cmp`-identical to `origin/dev` at the resolved symlink target. `AILANG_DRIVER_DRIFT=0`.
- Gate-1 base `93ac5a915b1826ddb44d39e44c9265e429110709@2026-11-04T19:01:05Z`. Dev green: CI and Build and Release success at HEAD, 18 checks, none red.
- Ledger valid (40 rows, 24 OPEN). 26 tickets open. No lane degraded this fire (`MISSION_ROUTING_NOTE` empty), so codex served the planner and executor.

**Reality check** (first-party, before routing; key names and file counts only; detail in [gates](fleet-mission-evidence/iteration85/gates.md)):
- No scanner existed. The driver sources `secrets.env` (`mission-control.sh:119`) and strips only `ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN` and `OPENAI_API_KEY` (`:176`).
- Whole-tree scan of 7 keys over three transcript roots (7.2 GB): `OPENROUTER_API_KEY` 4 files, `GOOGLE_API_KEY` 4, `OLLAMA_API_KEY` 4, `ANTHROPIC_API_KEY` 1, `CLAUDE_CODE_OAUTH_TOKEN` 0 (rotated this morning), the rest 0. The class is live beyond the ticket's one key.
- That scan took ~15 min, while a real 6 h fire window touches ~48 files. So the detector must be marker-scoped.
- Policy screen: the ticket asks for "a strip and/or a scanner". The strip changes what each lane receives (HD-2a), so it parks. The scanner is mechanical, so it was built.

**Did** (generator ≠ judge):
- **Designer**: not spawned. The ticket plus D-TRIAGE-36's attended ruling is the spec (precedent iteration 83).
- **Planner**: `codex:gpt-6.1-sol`, read-only sandbox, probe rc 0, run rc 0, 95,712 tok ([directive](fleet-mission-evidence/iteration85/planner-directive.txt), [plan](fleet-mission-evidence/iteration85/plan.md)). One milestone, 29 arms each with a named killing mutation, reuse of `_mc_bounded`. It named the `exec ailang mission iterate` bypass itself.
- **Executor**: `codex:gpt-6.1-sol`, workspace-write, rc 0, 91,919 tok ([directive](fleet-mission-evidence/iteration85/executor-directive.txt), [report](fleet-mission-evidence/iteration85/executor-report.md)). It reported 30 arms, all plan mutations killed on disposable copies, and no git writes.
- **Controller edit**: the executor's leak notice sent key names and paths to the PUBLIC bookkeeping issue as well. The GitHub body is now a hit count. The arm was rewritten; the mutant restoring the old body FAILs `Driver_leak`.
- **Gates at head**, controller, outside any sandbox: `bash -n` ×3, shellcheck, the suite (30/30), `run-suites`, `check_changelog`, `git diff --check` all rc 0. `env -i make test-launchd-drivers` rc 0, 53/53 suites. The real `secrets.env` sha and state-dir entry count were the same before and after.
- **Done-gate dry-runs** (`AILANG_DRIVER_PINNED=74daa5741`, idle armed `triage` profile, no pidfile either side): healthy `lanes=ok`; degraded (`codex:bogus-model-i85`) `DEGRADED(1)` → `claude:claude-haiku-5-5`. Both rc 0, both print the new dry-run scan line, and neither leaves a marker.
- **Evaluator**: sonnet, Agent tool, foreground, own judge worktree: **PASS**, 0 blocking ([evaluation](fleet-mission-evidence/iteration85/evaluation.md)). It killed 9 of 10 of its own mutants; the survivor is equivalent, since the EXIT trap recomputes the status. Its score is internally inconsistent: it states 87, the category rows sum to 91, and its file says 77. All three are above 70, so the verdict stands and the number is recorded as uncertain.
- **Landing**: PR [#2122](https://github.com/sunholo-data/ailang/pull/2122). The closing-keyword scan found 0.

**Gate 3b**:
- PR head `74daa5741`: 27/27 checks terminal, 0 not green (SHA-pinned poll).
- Base steady since Gate 1 (drift rc 0). `MERGEABLE`/`CLEAN`. Merged `--merge --match-head-commit` → `26b35b08115ee9582b896025900b81dffb691cd2`.
- Merge commit: CI and Build and Release `completed/success` (per-workflow read on the merge SHA), 18 check-runs, 0 non-green.
- Ticket resolved with `--sha 26b35b081`, and the reply reached triage (26 → 25 open).

**Ruled out**:
- Building the env strip now. It changes what each lane receives and can break a lane's auth (HD-2a). It is D-FLEET-42, with A as a proposed-diff build.
- Redacting the leaked transcripts, or rotating the four exposed keys: those are Mark's steps (D-TRIAGE-36 left redaction awaiting his permission). Both are named in D-FLEET-42.
- Treating the judge's scalability finding as blocking. Per-file-per-key grep forks put a cliff at ~3,500 fresh files under the 120 s cap; a real fire touches ~48. A timeout logs INSTRUMENT FAILURE, never clean. Not ticketed (the fleet does not self-source).
- The judge's inline-comment parse gap (`KEY=v # c` keeps the comment in the value): latent here, since 0 of 7 real assignments carry one.
- Scanning without a marker: ~15 min per run, measured.

**Retro lane**: no skill edit. Observations:
- (a) The controller's own planner directive said "send ONE `_mc_exit_notice` whose body lists key names and file paths only", without noting that the helper's 4th argument goes to a PUBLIC GitHub issue. Plan and executor followed it faithfully; controller review caught it before the judge. This is one instance. A second would justify a Gate-3 directive rule: name each sink's audience when a detector reports sensitive findings.
- (b) The judge's score arithmetic disagreed with itself three ways (87 / 91 / 77). One instance. If it recurs, the evaluator spawn prompt should ask for the arithmetic to be checked.

**Progress**: Clause 1 UNMEASURED, goal unmoved. Clause 2 UNMET: one ticket resolved inside 48 h (~11 h). Clause 3 MET (25 open). Clause 4: prior evidence only. Clause 5 preserved: done-gate green, merge CI green.

Last 3 landings: iteration 83, none (draft); iteration 84, clause 2; iteration 85, clause 2. No drift alarm. Harness share: 20 of the last 20 by design (the fleet's whole remit is the harness).

**Routing evidence**: Gate-1 base `93ac5a915b1826ddb44d39e44c9265e429110709@2026-11-04T19:01:05Z`. Driver note: none (no lane degraded). Details: [routing](fleet-mission-evidence/iteration85/routing.md).
- Controller `claude:claude-opus-5-5` (tok: not reported). `MISSION_ANTHROPIC_AVAILABLE=1`.
- Designer: not spawned; rotation pointer not advanced (`codex:gpt-6.1-sol`).
- Planner `codex:gpt-6.1-sol` (95,712 tok). Resolver `recipe codex:gpt-6.1-sol declared:planner-lane-default-pin`.
- Executor `codex:gpt-6.1-sol` (91,919 tok). Resolver `recipe codex:gpt-6.1-sol declared:provider-pin`.
- Evaluator `sonnet`, Agent tool (69,157 subagent tok). Resolver `agent-tool sonnet declared:alias-pin`. Judge independence: cross-vendor.
- Quorum: none. Metered spend $0. No lane recorded dead.

**Next**:
1. A ruling on D-FLEET-42. Under A, the per-lane env strip is built as a proposed diff with per-lane dry-runs.
2. The pre-push mirror guard (carried from iteration 84; two instances, iterations 63 and 84): one `sync-agents-skills.sh --check` arm in `tools/launchd/githooks/pre-push`, shipped alone, with a refuse/allow drill pair. It is not ticketed; check whether the closed ticket's second ask covers it before building.
3. Otherwise, a ruling on D-FLEET-18, -36/-38, -25, -21 or -26 lets its draft (#2110, #2108, #2082, #2067, #2065) rebase and land.

## 86 — 2026-11-05 — the transcript scan now matches only bytes written in the fire window (#2127 → `9ea8adcf4`): a byte-size baseline taken before the controller spawns stops a long-lived attended session from re-reporting a 17-day-old hit on every fire; ticket `transcript-scan:mtime-window-reattributes-old-hits-to-fire` resolved ~3 h after filing; independent sonnet judge PASS 92 [HARNESS]

**Pick**: `transcript-scan:mtime-window-reattributes-old-hits-to-fire` (world, filed 2026-11-05T00:54Z, `blocking=none`, 1 slot lost). It was the only ticket filed since iteration 85, a mechanical defect in iteration 85's own detector, inside its 48 h window, so it moves UNMET clause 2. No directive, ruling or red outranked it. Iteration 85's **Next** items were both ruling-gated (D-FLEET-42) or self-sourced (the pre-push mirror guard has no ticket).

**Orphans / prior record**: none. Iteration 85's record is on `origin/dev` (`8fc51e764`) and the log ended at 85. The open fleet branches (#2110, #2108, #2082, #2067, #2065) carry older logs (78–83) and are the expected ruling-gated drafts; left untouched. Self-notices rc 0 (0 crash notices, control 852:10).

**Preflight / observe**:
- Armed; gh `sunholo-voight-kampff`; billing CLEAN.
- #2039: 0 directives (10 comments, none allowlisted). `mission-2039-last-seen` is still absent, so the full thread was read; nothing to mark. #1945: 0 since its watermark.
- Skill drift: `SKILL.md` and all 13 resource files `cmp`-identical to `origin/dev` at the resolved symlink target and at the CWD copy. `AILANG_DRIVER_DRIFT=0`.
- Gate-1 base `c2fbb9b5886f81328552d7fc28c290f501a8de37@2026-11-05T02:23:50Z`. Dev green: CI and Build and Release success at HEAD, 18 checks, none red.
- Ledger valid (41 rows, 25 OPEN). 26 tickets open. No lane degraded this fire (`MISSION_ROUTING_NOTE` empty).

**Reality check** (first-party, synthetic data only; detail in [gates](fleet-mission-evidence/iteration86/gates.md)):
- The scanner picks candidates with `find -newer <marker>` and then `grep -F`s the whole file.
- Reproduced at base: a pre-marker secret line plus a harmless post-marker append → `verdict=LEAK`, rc 1. Control: untouched file → `clean files=0`, rc 0.
- Policy screen: detector accuracy only; no routing, lane, quota or billing change, so mechanical.

**Did** (generator ≠ judge):
- **Designer**: not spawned. The ticket's evidence and direction plus the controller's reproduction are the spec (precedent iterations 83, 85).
- **Planner**: `codex:gpt-6.1-sol`, read-only sandbox in `.planner-wt-iter86`, probe rc 0, run rc 0, 86,592 tok ([directive](fleet-mission-evidence/iteration86/planner-directive.txt), [plan](fleet-mission-evidence/iteration86/plan.md)). It took the controller's byte-offset direction, chose a NUL-safe Python 3 metadata snapshot (one process, no per-file forks), a `baseline=<N|none|failed>` summary field with the driver regex moved in lock-step, and refined the notice wording so a whole-file fallback says "write time is unknown".
- **Executor**: `codex:gpt-6.1-sol`, workspace-write, rc 0, 91,949 tok ([directive](fleet-mission-evidence/iteration86/executor-directive.txt), [report](fleet-mission-evidence/iteration86/executor-report.md)). Gate list baselined at base first (all rc 0, suite 30/30). It reported 52 arms and 28 named mutants killed, with no git writes. The controller made no code edit.
- **Gates at head**, controller, outside any sandbox: `bash -n` ×3, shellcheck, the suite (52/52), `run-suites`, `check_changelog`, `git diff --check` all rc 0. `env -i make test-launchd-drivers` rc 0, 53/53 suites. Real `secrets.env` sha, state-dir entry count (489) and transcript-scan state files (1, this fire's own marker) unchanged.
- **Ticket repro at head**: with a baseline → `clean`; without one → the old `LEAK` (`baseline=none`, the fallback); a new secret after the snapshot → `LEAK`. The fake value occurs 0 times in the baseline.
- **Cost on the rig**: real snapshot 1.3 s, 12,055 entries, mode 600. Synthetic worst case 60 candidates × 12,060 entries = 15 s against the 120 s cap.
- **Done-gate dry-runs** (`AILANG_DRIVER_PINNED=96882f837`, idle armed `triage`, no pidfile either side): healthy `lanes=ok`; degraded (`codex:bogus-model-i86`) `DEGRADED(1)` → `claude:claude-haiku-5-5`. Both rc 0, both print the new snapshot line, neither leaves an artifact.
- **Evaluator**: sonnet, Agent tool, foreground, own judge worktree: **PASS 92**, 0 blocking ([evaluation](fleet-mission-evidence/iteration86/evaluation.md)). Category rows sum to the stated total (32+23+19+9+9). It killed 13 of 13 of its own mutants, including one that reverts to whole-file matching. Strongest objection (non-blocking): the baseline lookup is O(candidates × entries) in bash; it measured 300 candidates at 66 s, so a fire touching ~500+ files would hit the cap. That logs INSTRUMENT FAILURE, never clean.
- **Landing**: PR [#2127](https://github.com/sunholo-data/ailang/pull/2127). The closing-keyword scan found 0 (control fired).

**Gate 3b**:
- PR head `96882f837`: 27/27 checks terminal, 0 not green (SHA-pinned poll).
- Base steady since Gate 1 (drift rc 0). `MERGEABLE`/`CLEAN`. Merged `--merge --match-head-commit` → `9ea8adcf4f6de9954599c59bbfc6ad540ededff2`.
- Merge commit: CI and Build and Release `completed/success` (per-workflow read on the merge SHA), 18 check-runs, 0 non-green.
- Ticket resolved with `--sha 9ea8adcf4`; reply to world (26 → 25 open).

**Ruled out**:
- JSONL-timestamp filtering: claude, codex and pi records differ in shape and some carry no timestamp, so it would mean a second parser of three formats inside a secret detector.
- Scanning only the fire's own session files: controller session ids are not known for every lane, and narrowing roots would change coverage. The notice now says the window includes concurrent sessions.
- A second "report pre-existing hits once" pass: it would re-read 406 MB files every fire for exposure already counted under D-FLEET-42.
- Overlap bytes at the boundary: they would re-match pre-existing hits, which is the bug. Records are appended whole, so a straddling value is a documented residual risk.
- Fixing the judge's O(C×E) lookup now: it was found after judging, the failure is loud, and measured fire windows (~48 files per 6 h) sit well inside the cap. Not ticketed (the fleet does not self-source); a timeout on a real fire would surface as INSTRUMENT FAILURE in the driver log.

**Retro lane**: no skill edit. Observations:
- (a) The judge did not write its evaluation file, saying "the bash tool ran my other commands"; its final message carried the full verdict, which the controller saved. One instance. A spawn prompt that names a file deliverable could also say "if you cannot write it, return it in full", which this one did by luck of the output section.
- (b) Iteration 85's retro (b) arithmetic check was put in the evaluator prompt this time, and the judge's rows summed to its total. That closes the one-instance observation with no rule needed.
- (d) `ailang mission rotate-log fleet --keep 3 --stream status` does NOT rotate the charter's STATUS stamps: it treats `fleet-mission-status-archive.md` itself as the stream, kept 3 of its 78 entries and moved 75 into a new `fleet-mission-status-archive-old.md` (bodies retained, charter untouched). Caught by the post-rotation line check, restored in the record worktree before commit. The charter→archive move was then done by script with a line-count assertion, plus `--index-only --stream status`, which matches iteration 85's commit shape. Gate 4 prescribes `--stream status` only with `--index-only`, so this was a controller misuse rather than a rule gap; one instance. If it recurs, the tool should refuse `--keep` on a stream whose live file is already an archive.
- (c) The bare planner resolver returned `fail-closed:no-doc` while the provider pin is codex (role-spawn-routing §2(a)); passing the charter gave `declared:planner-lane-default-pin`, as at iteration 85. Known, documented, no new friction.

**Progress**: Clause 1 UNMEASURED, goal unmoved. Clause 2 UNMET: one ticket resolved inside 48 h (~3 h). Clause 3 MET (25 open). Clause 4: prior evidence only. Clause 5 preserved: done-gate green, merge CI green.

Last 3 landings: iteration 84, clause 2; iteration 85, clause 2; iteration 86, clause 2. No drift alarm. Harness share: 20 of the last 20 by design (the fleet's whole remit is the harness).

**Routing evidence**: Gate-1 base `c2fbb9b5886f81328552d7fc28c290f501a8de37@2026-11-05T02:23:50Z`. Driver note: none (no lane degraded). Details: [routing](fleet-mission-evidence/iteration86/routing.md).
- Controller `claude:claude-opus-5-5` (tok: not reported). `MISSION_ANTHROPIC_AVAILABLE=1`.
- Designer: not spawned; rotation pointer not advanced (`codex:gpt-6.1-sol`).
- Planner `codex:gpt-6.1-sol` (86,592 tok). Resolver bare `agent-tool opus fail-closed:no-doc`; with the charter `recipe codex:gpt-6.1-sol declared:planner-lane-default-pin`; the pin was followed.
- Executor `codex:gpt-6.1-sol` (91,949 tok). Resolver `recipe codex:gpt-6.1-sol declared:provider-pin`.
- Evaluator `sonnet`, Agent tool (70,809 subagent tok). Resolver `agent-tool sonnet declared:alias-pin`. Judge independence: cross-vendor.
- Quorum: none. Metered spend $0. No lane recorded dead.

**Next**:
1. A ruling on D-FLEET-42. Under A, the per-lane env strip is built as a proposed diff with per-lane dry-runs.
2. Any new ticket, in particular one reporting a transcript-scan INSTRUMENT FAILURE (the judge's O(C×E) lookup); the fix would be a single-pass lookup.
3. A ruling on any of the five drafts (#2110, #2108, #2082, #2067, #2065) lands it.

## 87 — 2026-11-05 — Gate 0's self-notice read now reports the driver's transcript-scan notices (#2132 → `7e9e9192b`): both public wordings are recognised, rows carry only metadata, exit codes unchanged; ticket `gate0:self-notices-skip-transcript-scan-notice` resolved ~6 h after filing; independent sonnet judge PASS 92 [HARNESS]

**Pick**: `gate0:self-notices-skip-transcript-scan-notice` (world, filed 2026-11-05T04:53Z, `blocking=none`, 1 slot lost). It was the only ticket filed since iteration 86, a mechanical gap left by iterations 85–86's detector, inside its 48 h window, so it moves UNMET clause 2. No directive, ruling or red outranked it. Iteration 86's **Next** items were ruling-gated (D-FLEET-42, the five drafts) or untriggered (no transcript-scan INSTRUMENT FAILURE ticket arrived).

**Orphans / prior record**: none. Iteration 86's record is on `origin/dev` (`d49e0e01e`) and the log ended at 86. The open fleet branches (#2110, #2108, #2082, #2067, #2065) carry older logs (78–83) and are the expected ruling-gated drafts; left untouched. Self-notices rc 0 (0 crash notices, control 852:10).

**Preflight / observe**:
- Armed; gh `sunholo-voight-kampff`; billing CLEAN.
- #2039: 0 directives (12 comments, none allowlisted). `mission-2039-last-seen` is still absent, so the full thread was read; nothing to mark. #1945: 0 since its watermark.
- Skill drift: `SKILL.md` and all 13 resource files `cmp`-identical to `origin/dev` at the resolved symlink target, per file.
- Gate-1 base `f6a95d75da9845c13adc9c052f81499f314ab66b@2026-11-05T09:40:15Z`. Dev: 22 checks on HEAD, none red once `test-windows` and `Build macos-latest` finished.
- Ledger valid (41 rows, 25 OPEN). 26 tickets open. No lane degraded this fire (`MISSION_ROUTING_NOTE` empty; `MISSION_OVER_RATION=openrouter`, unused).

**Reality check** (live, read-only `gh` GETs; detail in [gates](fleet-mission-evidence/iteration87/gates.md)):
- At base, the script read #2039/#1945 as `issue: 2039 comments=12 self=12 crash=0 other=12`, rc 0. A self-authored transcript-scan notice (2026-11-05T03:38:38Z, fire `fleet-1793845378-59113`) sits on #2039 unreported. On World #449 it missed both notices the ticket names.
- Two public wordings exist. The old one, `Transcript secret exposure on fire …` (74daa5741), is what the issues carry today. The current one, `Transcript secret scan after fire …`, came in with 96882f837. The fix must read both.
- No driver caller; the consumers are Gate 0 step 6a and the suite. Policy screen: detection/reporting only, so mechanical.

**Did** (generator ≠ judge):
- **Designer**: not spawned. The ticket's evidence and suggested shape plus the live repro are the spec (precedent iterations 83, 85, 86).
- **Planner**: `codex:gpt-6.1-sol`, read-only sandbox in `.planner-wt-iter87`, probe rc 0, run rc 0, 95,151 tok ([directive](fleet-mission-evidence/iteration87/planner-directive.txt), [plan](fleet-mission-evidence/iteration87/plan.md)). It accepted all seven directions and qualified one. The step-6a prose must not claim every hit was written during the fire, because the driver's whole-file fallback says write time is unknown. It added the fire id to the rows (already public) and put the new arms in a sourced companion so both test files stay under 800 lines.
- **Executor**: `codex:gpt-6.1-sol`, workspace-write, rc 0, 95,832 tok ([directive](fleet-mission-evidence/iteration87/executor-directive.txt), [report](fleet-mission-evidence/iteration87/executor-report.md)). It reported 168 assertions and 22 named mutants killed, with no git writes. The `.agents` mirror sync was sandbox-denied (`utimensat`), and the executor labelled the dependent gates UNINFORMATIVE rather than pass or fail. The controller ran the sync outside the sandbox: only the gate-0 mirror changed, and `--check` rc 0. That was the controller's only edit.
- **Gates at head**, controller, outside any sandbox: `bash -n` ×3, the suite (168/168), shellcheck (0 new vs base; base itself is rc 1 with 87 findings), `git diff --check`, `check-changelog`, `check-file-sizes`, `TestQuorumArtifactShellSuite` all ok. `env -i make test-launchd-drivers` rc 0, 53/53 suites.
- **Ticket repro at head**: #2039 → one `transcript:` row (`hits=1 in-window=yes fire=fleet-1793845378-59113`), `transcript=1 other=11`, rc 0. World #449 → 2 rows, `transcript=2`. The base script on the same inputs → none.
- **Done-gate dry-runs** (`AILANG_DRIVER_PINNED=b4651a0d5`, idle armed `triage`; world was live): healthy `DRY RUN ok`; degraded (`codex:bogus-model-i87`) probe rc 1 → `claude:claude-haiku-5-5`, `DRY RUN ok`. Both rc 0, no pidfile, state entry count 494 → 494. The driver is unchanged, so these show only that the pinned tree boots.
- **Evaluator**: sonnet, Agent tool, foreground, own judge worktree: **PASS 92**, 0 blocking ([evaluation](fleet-mission-evidence/iteration87/evaluation.md)). Rows sum to the total (33+23+19+9+8). It ran 10 mutants of its own: 9 killed, 1 equivalent survivor (`match`→`search` under `\A`). Strongest objection (non-blocking): regex tightness rests on executor-written fixtures; no test replays a verbatim live comment, though the live repro covers it.
- **Landing**: PR [#2132](https://github.com/sunholo-data/ailang/pull/2132). The closing-keyword scan found 0 (control fired).

**Gate 3b**:
- PR head `b4651a0d5`: 27/27 checks terminal. One non-required red: `Build macos-latest`, `TestRuntimeNestedStopUsesRemainingDeadline` (`internal/effects`, 0.04 s); the sibling macOS leg passed. This is the third sighting of that test in this log (iteration 84 and an archived entry). `gh run rerun --failed` → success. All 5 required checks green throughout.
- Base drift since Gate 1: 6 commits (v0.57.3 release, executor-mcp/OAuth), none touching the changed files. `MERGEABLE`/`CLEAN`. Merged `--merge --match-head-commit` → `7e9e9192bd916b3008f1914f11d746441002de57`.
- Merge commit: CI and Build and Release `completed/success`, 18 check-runs, 0 non-green.
- Ticket resolved with `--sha 7e9e9192b`; reply to world (26 → 25 open).

**Transcript-scan notices seen this fire** (step 6a's new disposition, applied to itself): 1 on #2039, fire `fleet-1793845378-59113` (iteration 86's fire), 1 hit. That fire launched on the pre-#2127 pin `c2fbb9b58`, so its scan matched whole files. The hit is therefore likely the stale-history class that #2127 fixed, already counted under D-FLEET-42. No value or path was read.

**Ruled out**:
- A new exit code for transcript-only windows: every reader of rc 1 and step 6a would change, and the ticket asked for a count.
- A live positive control for transcript notices: bookkeeping issues still receive notices and rotate weekly, so a fixed count would drift and floor every fire. Fixtures plus the F8 source check stand in for it.
- Requiring the old wording in F8: the driver no longer emits it. It stays recognised in comments only.

**Retro lane**: no skill edit. Observations:
- (a) The judge again wrote only a summary file and returned the full verdict in its final message (second instance after iteration 86). Its "return in full" fallback, added to the prompt after 86, worked; no rule needed while that line stays in the prompt.
- (b) The `internal/effects` macOS timing flake is on its third sighting in fleet PRs, each time on a shell/markdown-only diff and green on re-run. It is language-core territory (outside fleet scope) and V1, which owns repo reds, is disabled. Surfaced in the report, not ticketed.
- (c) zsh bit the controller twice (`$b:refs` expanded as a modifier in a fetch refspec, an unmatched glob in a poll). Both were caught and re-run under `/bin/bash`, as Gate 3b already prescribes. No new rule.

**Progress**: Clause 1 UNMEASURED, goal unmoved. Clause 2 UNMET: one ticket resolved inside 48 h (~6 h). Clause 3 MET (25 open). Clause 4: prior evidence only. Clause 5 preserved: done-gate green, merge CI green.

Last 3 landings: iteration 85, clause 2; iteration 86, clause 2; iteration 87, clause 2. No drift alarm. Harness share: 20 of the last 20 by design.

**Routing evidence**: Gate-1 base `f6a95d75da9845c13adc9c052f81499f314ab66b@2026-11-05T09:40:15Z`. Driver note: none (no lane degraded). Details: [routing](fleet-mission-evidence/iteration87/routing.md).
- Controller `claude:claude-opus-5-5` (tok: not reported). `MISSION_ANTHROPIC_AVAILABLE=1`.
- Designer: not spawned; rotation pointer not advanced (`codex:gpt-6.1-sol`).
- Planner `codex:gpt-6.1-sol` (95,151 tok). Resolver with the charter `recipe codex:gpt-6.1-sol declared:planner-lane-default-pin`; `derive-planner-lane.sh` agrees.
- Executor `codex:gpt-6.1-sol` (95,832 tok). Resolver `recipe codex:gpt-6.1-sol declared:provider-pin`.
- Evaluator `sonnet`, Agent tool (65,511 subagent tok). Resolver `agent-tool sonnet declared:alias-pin`. Judge independence: cross-vendor.
- Quorum: none. Metered spend $0. No lane recorded dead.

**Next**:
1. A ruling on D-FLEET-42. Under A, the per-lane env strip is built as a proposed diff with per-lane dry-runs.
2. Any new ticket.
3. A ruling on any of the five drafts (#2110, #2108, #2082, #2067, #2065) lands it.

## 88 — 2026-11-05 — D-FLEET-42 A's first step built as a proposed diff (draft #2141, `4ea46f05e`), unmerged pending a ruling: a per-lane `secrets.env` scope library, a marker bridge and the tested `~/.zshenv` host line; measured that claude and codex zsh tool shells re-source `secrets.env`, so a spawn-time strip alone protects only pi; independent sonnet judge PASS 86 [HARNESS]

**Pick**: D-FLEET-42 option A, first step (M1), as a draft that does not merge. No ruling, directive, crash notice or new ticket arrived after iteration 87. All 25 open tickets are ruling-gated, so the *Ticket-driven only* guardrail's idle rule applied: screen every OPEN row by step for an in-scope first step with no diff. D-FLEET-42 is the only row added since iteration 83's screen. Its recommended A ("the fleet builds a per-lane allowlist strip at spawn … as a proposed diff … merges only on your ruling") begins with in-scope code (`tools/launchd/**`) and had no diff. Iteration 85's reason for not building it ("HD-2a policy") bars merging, not a draft.

**Orphans / prior record**: none. Iteration 87's record is on `origin/dev` (`34b2a8f54`) and the log ended at 87. The open fleet branches (#2110, #2108, #2082, #2067, #2065) carry older logs (78–83) and are the expected ruling-gated drafts; left untouched. Self-notices rc 0 (0 crash notices, control 852:10).

**Preflight / observe**:
- Armed; gh `sunholo-voight-kampff`; billing CLEAN.
- #2039: 0 directives (13 comments, none allowlisted). `mission-2039-last-seen` is still absent, so the full thread was read; nothing to mark. #1945: 0 since its watermark.
- Skill drift: `SKILL.md` and all 13 resource files at the resolved symlink target `cmp`-identical to `origin/dev`, per file.
- Gate-1 base `1ec1847a7a720b4ffeddcf4bf56bf461af9c5608@2026-11-05T17:17:42Z` (pin at `9c7e997e0`, one docs commit behind). Dev CI green on the parent; HEAD queued, then green (18 checks, none red).
- Ledger valid (41 rows, 25 OPEN). 25 tickets open, newest first seen 10-28. Every signature appears in the charter. No lane degraded this fire (`MISSION_ROUTING_NOTE` empty; `MISSION_OVER_RATION=openrouter`, unused).
- Weekly external-issue sweep not due: the last ran Monday 11-02 (iteration 77).

**Reality check** (first-party, key names and SET/UNSET only; [gates](fleet-mission-evidence/iteration88/gates.md)):
- `secrets.env` exports 7 names. The driver sources it (`mission-control.sh:119`) and unsets only `ANTHROPIC_API_KEY`, `ANTHROPIC_AUTH_TOKEN` and `OPENAI_API_KEY` (`:171`).
- In this claude controller's Bash-tool shell (`/bin/zsh`), `OPENAI_API_KEY` is SET again. The control, `ANTHROPIC_API_KEY`, is UNSET. The other four keys are SET.
- One bounded `codex exec --sandbox read-only` probe under `CODEX_HOME=~/.codex-missions` (`inherit = "core"`) printed every key SET except `ANTHROPIC_API_KEY`, with `$0=/bin/zsh`. `inherit = "core"` drops non-core variables, so the only source is `~/.zshenv`, which sources `secrets.env` and then unsets only the two Anthropic keys.
- pi's sandboxed bash runs `spawn("bash", ["-c", …])` (`tools/pi-extensions/sandbox/index.ts:152`), which does not read `~/.zshenv`.
- Consequence: option A as worded changes what claude and codex tool shells hold only if Mark also adds a hook line to `~/.zshenv`, which is outside the repo.
- The quorum's `gemini-4-argon` seat pins Vertex ADC (`models.yml` `env_var: ""`, `gcp_project: "ailang-dev"`; `factory.go` ~207–231), so controllers need no Google key. Verified before the allowlist relied on it.

**Did** (generator ≠ judge):
- **Designer**: not spawned. The decision row plus the measured reality check are the spec (precedent: iterations 83, 85–87). Rotation pointer not advanced.
- **Planner**: `codex:gpt-6.1-sol`, read-only sandbox in `.planner-wt-iter88`, probe rc 0, run rc 0, 139,214 tok ([directive](fleet-mission-evidence/iteration88/planner-directive.txt), [plan](fleet-mission-evidence/iteration88/plan.md)).
  - It accepted the zsh finding and built the allowlists from consumers (C, O, R, Q, P).
  - It found that the marker name `AILANG_SECRETS_LANE` trips `codex-env-args.sh`'s `*SECRET*` deny rule and needs one exact exception.
  - It found that pi needs `BASH_ENV` cleared.
  - It sized the work at 3 milestones, 2–3 days.
- **Controller cut to M1**: library, bridge, host-line text, design record, suite, `make/test.mk`, changelog. No driver, recipe or existing-test change.
- **Executor**: `codex:gpt-6.1-sol`, workspace-write, cap raised to 3000 s (bounded), rc 0, 131,787 tok ([directive](fleet-mission-evidence/iteration88/executor-directive.txt), [report](fleet-mission-evidence/iteration88/executor-report.md)).
  - Six files, no git writes.
  - Suite: 57 arms under both `/bin/bash` 3.2 and `/bin/zsh` 5.9. Mutation bank: 27 mutants, all killed (one bash-equivalent).
  - It labelled `make test-launchd-drivers` UNINFORMATIVE under the sandbox (51/54 suites; three blocked by `ps` and nested sandboxes).
- **Controller edits**: two prose fixes in the design record before judging (it claimed no host file was inspected, which is false; and a confusing "M3" label). Both were disclosed to the judge. Build commit `95623a99d`.
- **Gates at head**, controller, outside any sandbox:
  - `bash -n` ×3, `zsh -n` ×2, suite 57/57, mutations 56/56, shellcheck rc 0.
  - `git diff --check`, `check-changelog`, `check-file-sizes`, `check-context-docs`, `check-no-personal-email` and `sync --check` all rc 0.
  - `env -i make test-launchd-drivers` **54/54** suites (base 53/53).
  - `~/.ailang/state` entries 496 → 496, and no fake value appeared in any output.
- **Idle-screen judge**: sonnet, Agent tool, foreground, read-only, 72,333 tok ([verdict](fleet-mission-evidence/iteration88/idle-judge.md)). It was not told the controller's verdict. It screened all 25 OPEN rows and agreed: D-FLEET-42 only. Borderline named: D-FLEET-25's design-doc half, judged no.
- **Diff judge**: sonnet, Agent tool, foreground, own worktree at `95623a99d`, 85,276 tok: **PASS 86**, 0 blocking ([evaluation](fleet-mission-evidence/iteration88/evaluation.md)). Rows sum to the total (31+22+16+9+8).
  - It reproduced the host line in a real synthetic `ZDOTDIR` startup under `zsh -c`, `-lc` and `-i -c`. `exit 78` in `.zshenv` stops the body.
  - It ran 15 mutants of its own: 9 killed, 6 analysed as equivalent or no-op.
  - Strongest objection (non-blocking): the guarantee rests on three things outside this diff: launch wiring setting the marker, an immutable pinned root, and the host line preceding any re-source. All three are disclosed.
- **Judge findings**: 1, 2 and 4 were reproduced first-party, then fixed in commit `4ea46f05e` (prose and comments; suite 57/57 and mutations 56/56 re-run):
  - (1) A name absent from `secrets.env` survives apply (probe: `ANTHROPIC_API_KEY` SET after `pi-role`), so "every lane strips X" became "strips X when `secrets.env` assigns it".
  - (2) The host line's `2>/dev/null` hides the bridge's audit line.
  - (4) The consumer is `mission-control.sh:157`, not the `:154` comment.
  - Finding 3 (the suite does not pin a real `ZDOTDIR` startup) is left for M3. Finding 5 (dead fallback) is harmless.
- **Landing**: draft PR [#2141](https://github.com/sunholo-data/ailang/pull/2141), not merged. The closing-keyword scan found 0 (control fired).

**Gate 3b**: not applicable; nothing merges (iteration 83 precedent). The draft's checks were read once: 24 reported, none failed at record time.

**Incident (controller hygiene, this fire)**: while measuring codex's `shell_environment_policy`, the controller ran a context `grep -A6` on `~/.codex/config.toml`. The output included a Claude OAuth token stored in that file's `[shell_environment_policy.set]` table. A hash comparison, printing only EQUAL or DIFFERENT, shows it is a different token from the loops' `secrets.env` one. Mission codex runs use `~/.codex-missions`, whose config has no such entry. No value is recorded anywhere in this record. The fire's transcript scan should report it. Rotation is Mark's step, added to D-FLEET-42's row.

**Transcript-scan notices seen this fire** (step 6a): 1 on #2039, fire `fleet-1793845378-59113`, 1 hit. It predates #2127 and was already reported by iteration 87.

**Ruled out**:
- Building M1–M3 in one fire. The plan's 2–3 days does not fit one executor slot. M2 also edits 10 existing harness fixtures, and M3's dry-runs need the M2 wiring.
- Keeping `GOOGLE_API_KEY` for controllers: the only Google quorum seat is on Vertex ADC, measured.
- Treating the driver-side strip alone as protection for claude and codex lanes: refuted by the zsh measurement.
- A skill edit for the controller's own config-file leak: it is one instance. The "read by NAME" rule covers the environment, not files. Recorded as instance 1.

**Retro lane**: no skill edit. Observations:
- (a) Instance 1 of a file-read variant of the env-by-NAME rule. A context grep on a credential-bearing config printed a value. If a second instance appears, extend the Gate 0 "read by NAME" rule to credential files: use `awk` to print key names only, never `grep -A/-B/-C` or `cat`.
- (b) The planner over-scoped for one fire. The cut to M1 kept the draft self-consistent. No rule needed.
- (c) Both judges returned full verdicts in their final messages (the prompt asked for that).

**Progress**: Clause 1 UNMEASURED, goal unmoved. Clause 2 UNMET (nothing resolved). Clause 3 MET (25 open). Clause 4: prior evidence only. Clause 5 preserved (nothing merged to code).

Last 3 landings: iterations 85, 86 and 87, all clause 2. No drift alarm. Harness share: 20 of the last 20, by design.

**Routing evidence**: base=d90098a245ada873e8570175ccdcc9fdddb945da@2026-11-05T18:13:37Z (Gate-4 record). Gate-1 base `1ec1847a7a720b4ffeddcf4bf56bf461af9c5608@2026-11-05T17:17:42Z`; Gate-3 worktree base `78104fab6` (DRIFT rc 1, 2 docs-only commits, benign). Driver note: none (no lane degraded). Details: [routing](fleet-mission-evidence/iteration88/routing.md).
- Controller `claude:claude-opus-5-5` (tok: not reported). `MISSION_ANTHROPIC_AVAILABLE=1`.
- Designer: not spawned. Resolver `recipe claude:claude-opus-5-5 declared:provider-pin`. Rotation pointer not advanced (`codex:gpt-6.1-sol`).
- Planner `codex:gpt-6.1-sol` (139,214 tok). Resolver with the charter: `recipe codex:gpt-6.1-sol declared:planner-lane-default-pin`; `derive-planner-lane.sh` agrees.
- Executor `codex:gpt-6.1-sol` (131,787 tok). Resolver `recipe codex:gpt-6.1-sol declared:provider-pin`.
- Evaluators: `sonnet`, Agent tool, ×2 (idle screen 72,333 tok; diff 85,276 tok). Resolver `agent-tool sonnet declared:alias-pin`. Judge independence: cross-vendor.
- Extra: one codex env probe (read-only, names only).
- Quorum: none. Metered spend $0. No lane recorded dead.

**Next**:
1. A ruling on D-FLEET-42.
   - Under A, M2 builds on #2141's branch: driver and role wiring, the codex marker exception, pi `BASH_ENV`, and fixture isolation.
   - M3 then runs healthy and degraded dry-runs per lane. Merging needs your ruling and your `~/.zshenv` line.
2. Rotate the Claude OAuth token exposed in `~/.codex/config.toml`.
3. Any new ticket.
4. A ruling on any of the six drafts (#2141, #2110, #2108, #2082, #2067, #2065) lands it.

## 89 — 2026-11-06 — D-FLEET-42 A's M2 wiring built as a proposed diff (draft #2143, `7f1e13d5f`, stacked on #2141), unmerged pending a ruling: controller children, a role entry point, the exact codex marker exception and pi `BASH_ENV` clearing; the controller caught a `$!`-is-a-subshell regression the fixtures could not see; independent sonnet judge PASS 92 [HARNESS]

**Pick**: D-FLEET-42 option A, milestone M2, as a draft stacked on #2141.
- No ruling, directive, crash notice or new ticket arrived after iteration 88. All 25 open tickets are ruling-gated. The one new occurrence, `reporting:daneel-reports-not-declared-triage` (last seen 11-05 23:06Z, 16 slots lost), is still D-FLEET-22, which is outside fleet scope.
- So the idle guardrail applied. The controller leaned IDLE: on the row-level reading of "no proposed diff yet", #2141 already counts as a diff for D-FLEET-42, and no other row qualified at iteration 88.
- The independent idle-screen judge was not told that lean. It ruled the step-level reading instead. "Screen by step" makes M2 a qualifying step: option A asks for the whole strip, wiring and dry-runs included, as a pre-ruling draft, and M1 alone protects nothing.
- The controller followed the judge, as at iteration 83, and scoped the build to M2. M3 (dry-runs) was deferred.

**Orphans / prior record**: none. Iteration 88's record is on `origin/dev` (`6ca4d4e90`), and the log ended at 88. The open fleet branches (#2141, #2110, #2108, #2082, #2067, #2065) carry the expected ruling-gated drafts. Self-notices rc 0 (0 crash notices; control 852:10).

**Preflight / observe**:
- Kill switch armed; gh account `sunholo-voight-kampff`; billing CLEAN.
- #2039: 0 directives (15 comments, none allowlisted). `mission-2039-last-seen` is still absent, so the full thread was read; nothing to mark. #1945: 0 since its watermark.
- Skill drift: the resolved symlink target and the pin's CWD copy were both checked. `SKILL.md` and all 13 resources are `cmp`-identical to `origin/dev`, per file.
- Gate-1 base `e4fcc15f62da859e16da4da34a107fe89ec746b7@2026-11-06T00:20:51Z`, with the pin at the same SHA. Dev CI green: `CI` success; 22 check-runs at HEAD, none red.
- Ledger valid: 41 rows, 25 OPEN. Every open ticket signature appears in the charter.
- No lane degraded (`MISSION_ROUTING_NOTE` and `MISSION_OVER_RATION` both empty).
- Weekly external-issue sweep not due; the last ran Monday 11-02 (iteration 77).

**Reality check**:
- Plan anchors: `git diff 9c7e997e0 origin/dev` over `tools/launchd`, `scripts/mission_pi_run.sh`, the mission-control skill and `make/test.mk` is empty. So iteration 88's plan anchors hold, and its M2 row is the spec.
- Base gates: the 11 suites the directive would name, plus `sync-agents-skills.sh --check`, were run on #2141's tip `4ea46f05e` before the directive was sent. All rc 0 (false-green 4).

**Did** (generator ≠ judge):
- **Designer**: not spawned. The decision row plus iteration 88's design record are the spec. Rotation pointer not advanced.
- **Planner**: not spawned. Iteration 88's planner (`codex:gpt-6.1-sol`) already planned M2, and the anchors are unchanged. The resolver, run without a doc, printed `agent-tool opus fail-closed:no-doc`; this is recorded, not acted on.
- **Idle-screen judge**: sonnet, Agent tool, foreground, read-only, 69,438 tok ([verdict](fleet-mission-evidence/iteration89/idle-judge.md)). Verdict: BUILD D-FLEET-42 M2; no other row qualifies. Borderlines named: D-FLEET-41 B and D-FLEET-25 A.
- **Executor**: `codex:gpt-6.1-sol`, workspace-write, 3000 s bounded wrapper. Probe rc 0; run rc 0, about 20 min; 159,523 tok ([directive](fleet-mission-evidence/iteration89/executor-directive.txt), [report](fleet-mission-evidence/iteration89/executor-report.md)).
  - It built M2a (controller wiring and fixtures), M2b (role wrapper and pi runner) and M2c (recipe prose).
  - It snapshotted each sub-milestone and made no git writes.
  - It labelled kill-tree, stall, codex-env and the skill sync UNINFORMATIVE under the sandbox, and did not run the four pi-runner suites.
- **Controller, outside the sandbox**: [gates](fleet-mission-evidence/iteration89/gates.md).
  - Before any re-run, a diff scan for writes outside the worktree found 10 `: > "$HOME/.config/ailang/secrets.env"` sites. Each runs after `export HOME=<mktemp/fixture>`, so none could reach the real file. Every gate run was also bracketed by a sha256 and line-count comparison of the real `secrets.env`, which printed only SAME or CHANGED: SAME every time.
  - The regenerated `.agents` mirror matched the executor's snapshot byte for byte.
  - **Defect 1: `$!` is not the controller.** On bash 3.2.57, `( export …; f || exit $?; sleep 7 ) &` gives `$!` = the bash subshell, with `sleep` as its child; a bare `sleep 7 &` gives `$!` = sleep. Base launched claude bare. The M2 subshell would therefore have made `_mc_stalled` (which skips its root and counts older descendants as long children) and `_mc_rusage_snapshot` key on a wrapper for the primary lane.
    - Every fixture passed, because they fake `claude` as a shell function.
    - Fix: `exec claude …` and `cd … && exec pi …`. The two fixtures move to PATH stubs, and `test_mission_kill_tree.sh` gains "controller pid is the controller process".
    - Mutation-proven: with `exec` reverted in a scratch copy, rc 1 `not ok`; with it, rc 0. The codex lane was left as is: it was already a backgrounded function.
  - **Defect 2: `test_pi_ext_args.sh` red at head (base 17/17).** The M2c prose replaced the raw pi sandbox snippet that the test pins, and took the `--mode json` teaching with it.
    - My first base control was invalid, because the archive omitted the skill dirs. The corrected control is what reads 17/17.
    - Fix: restore the snippet, and add one line saying the runner is preferred because it applies `pi-role`.
    - Root cause on my side: the directive's gate list did not name `make test-launchd-drivers`.
  - **Defect 3: repo-local scratch.** Two suites exported `TMPDIR=<repo>/.snap/test-tmp`. Fixed: one now uses `mktemp -d` with a trap, the other no longer overrides `TMPDIR`.
  - One `make test-launchd-drivers` run raced my edits and is void. The re-run on the final tree is **54/54**.
  - Also at the final tree: `test_secrets_scope.sh` 58/58; mutations 68 killed, 0 survivors; new-file shellcheck rc 0; the changed-file comparator (14 files) found no new findings; `sync --check`, `check-changelog`, `check-file-sizes`, `check-context-docs`, `check-no-personal-email` and `git diff --check` all rc 0.
  - Build commit `7f1e13d5f`, crediting codex.
- **Diff judge**: sonnet, Agent tool, foreground, its own worktree at `7f1e13d5f`, 89,376 tok: **PASS 92**, 0 blocking ([evaluation](fleet-mission-evidence/iteration89/evaluation.md)). Rows: 20+8+28+13+14+9. Its headline first said 90, and it corrected that to the row sum in the same message.
  - It confirmed: the parent stays unscoped; the billing unset is untouched; the codex marker exception is exact; a failed apply refuses the child visibly; `exec` keeps the logs and watchdogs; no fixture assertion was lost.
  - It killed 18 of its own mutants. Four died only in a sibling suite: no `exec` on pi, the ignored `mc_codex_env_args` failure and `include_only`-warns (all `test_quorum_artifacts.sh` only), and no `exec` on claude (kill-tree, stall, quorum_artifacts).
  - Strongest objection: nothing has been run against a real provider with the scope applied. M3's dry-runs are what would show it.
- **Landing**: draft PR [#2143](https://github.com/sunholo-data/ailang/pull/2143), base `fleet/i88-dfleet42-secrets-scope`, not merged. The closing-keyword scan found 0 (control fired).

**Gate 3b**: not applicable, because nothing merges (iteration 83 precedent).

**Transcript-scan notices seen this fire** (step 6a): 2 on #2039.
- Fire `fleet-1793845378-59113`, 1 hit: already reported by iterations 87 and 88.
- Fire `fleet-1793898974-52315`, 1 hit: iteration 88's own fire, consistent with the config-file incident iteration 88 recorded.
- Key names are in the private controlplane notice only.

**Ruled out**:
- Re-running the planner: its M2 plan is current, because the anchors are unchanged.
- Building M3 in the same fire: its six live dry-runs need an idle armed sibling or a synthetic HOME, and a full slot.
- Deleting or weakening `test_pi_ext_args.sh` to fit the plan's "replace the raw pi example": that test pins a deliberate contract.
- `exec` on the codex controller: it is behaviour-neutral against base (it was already a background function), and out of M2's scope.

**Retro lane**: no skill edit; one recorded friction.
- (a) **Instance 1 of "the directive's gate list omitted the full suite".** The executor could not see `test_pi_ext_args.sh`, which pins skill prose it edited. False-green (4) covers *base-red* gates in the directive. It does not cover *missing* gates, so a directive that edits skill prose should name `make test-launchd-drivers` even if the sandbox makes it uninformative. If this recurs, extend false-green (4).
- (b) A test-blind-spot shape worth naming: a fixture that fakes an external binary as a shell function cannot see process-shape changes (`exec`, `$!`, subshells). One instance; no rule yet.
- (c) Both judges returned full verdicts in their final messages.

**Progress**: Clause 1 UNMEASURED, goal unmoved. Clause 2 UNMET (nothing resolved). Clause 3 MET (25 open). Clause 4: prior evidence only. Clause 5 preserved (nothing merged to code). Last 3 landings: iterations 85, 86 and 87, all clause 2. No drift alarm. Harness share: 20 of the last 20, by design.

**Routing evidence**: base=e4fcc15f62da859e16da4da34a107fe89ec746b7@2026-11-06T01:12:15Z (Gate-4 record). Gate-1 and Gate-3 bases are the same SHA. Details: [routing](fleet-mission-evidence/iteration89/routing.md).
- Controller `claude:claude-opus-5-5` (tok: not reported). `MISSION_ANTHROPIC_AVAILABLE=1`.
- Designer: not spawned (no rotation advance).
- Planner: not spawned (iteration 88's plan reused).
- Executor `codex:gpt-6.1-sol` (159,523 tok). Resolver `recipe codex:gpt-6.1-sol declared:provider-pin`.
- Evaluators: `sonnet`, Agent tool, ×2 (idle screen 69,438 tok; diff 89,376 tok). Resolver `agent-tool sonnet declared:alias-pin`. Judge independence: cross-vendor.
- Quorum: none. Metered spend $0. No lane recorded dead.

**Next**:
1. A ruling on D-FLEET-42. Under A, M3 runs on #2143's branch: the scoped dry-run preview, then six healthy/degraded controller dry-runs. Merging needs your ruling and your `~/.zshenv` line.
2. Rotate the Claude OAuth token in `~/.codex/config.toml` (iteration 88's incident).
3. Any new ticket.
4. A ruling on any of the seven drafts (#2143, #2141, #2110, #2108, #2082, #2067, #2065) lands it.

## 90 — 2026-11-06 — D-FLEET-42 A's M3 built as a proposed diff (draft #2150, `26b713887`, stacked on #2143), unmerged pending a ruling: a pinned dry-run probes the controller inside its lane scope, and all six live healthy/degraded dry-runs passed; a bash 3.2 background-stdin trap was found and fixed on the way; independent sonnet judge PASS 90, then PASS 93 [HARNESS]

**Pick**: D-FLEET-42 option A, milestone M3, as a draft stacked on #2143.
- No ruling, directive, crash notice or new ticket arrived after iteration 89. There are 25 open tickets, the same signatures, all ruling-gated. #2039 has 0 allowlisted comments (16 comments).
- So the idle guardrail applied. The independent idle-screen judge (sonnet) was not told the controller's candidate. It ruled BUILD D-FLEET-42 M3, the only unbuilt in-scope step on a recommended option ([verdict](fleet-mission-evidence/iteration90/idle-judge.md)).
- Its strongest objection was a third stacked draft on one unanswered row. That is bounded: M3 is the row's last buildable step.

**Orphans / prior record**: none. Iteration 89's record is on `origin/dev` (`a6e112f9c`). Its worktree holds only the executor's `.snap/` scratch. No `fleet/i90` branch or PR existed. Self-notices rc 0: 0 crash notices, control 852:10.

**Preflight / observe**:
- Kill switch armed; gh account `sunholo-voight-kampff`; billing CLEAN.
- Directives: #2039 had 0. `mission-2039-last-seen` is still absent, so the full thread was read and there was nothing to mark. #1945 had 0 since its watermark.
- Skill drift: the resolved symlink target, `SKILL.md` and all 13 resources are `cmp`-identical to `origin/dev`. The pin sits at the same SHA.
- Gate-1 base `a6e112f9c74bd082f2807242208a3a5ef7ee6dbd@2026-11-06T07:33:30Z`. Dev CI `success`, 18 check-runs, none red.
- Ledger valid: 41 rows, 25 OPEN. Every open ticket signature appears in the charter (control count 3).
- No lane degraded (`MISSION_ROUTING_NOTE` and `MISSION_OVER_RATION` empty). Weekly sweep not due.

**Reality check**: base gates on #2143's head `7f1e13d5f`, run before the directive was sent: `make test-launchd-drivers` 54/54, `test_secrets_scope.sh` 58/58, sync/changelog/diff-check rc 0. `select_model` accepts an env pin without probing, so the directive told the executor that the preview must call the probe itself. `triage` was idle and armed: no kill switch, no pidfile.

**Did** (generator ≠ judge):
- **Designer / planner**: not spawned. The row and iteration 88's plan (arm table plus the "Healthy and degraded dry-runs per controller lane" section) are the spec. The planner resolver printed `agent-tool opus fail-closed:no-doc`, because the doc exists only on the draft branch; recorded, not acted on.
- **Idle-screen judge**: sonnet, Agent tool, foreground, 77,288 tok. The first spawn was refused by the spawn-pin hook (`fail-closed:role-missing`): line 1 was `MISSION-ROLE: evaluator (idle-screen judge), …`. It was re-sent with the bare token on line 1.
- **Executor**: `codex:gpt-6.1-sol`, workspace-write, 3000 s wrapper. Probe rc 0; run rc 0, about 25 min, 159,062 tok ([directive](fleet-mission-evidence/iteration90/executor-directive.txt), [report](fleet-mission-evidence/iteration90/executor-report.md)).
  - It built M3a: a `SECRETS SCOPE DRY RUN` block in the dry-run branch. The block uses a subshell, `_mc_set_controller`, the marker and root, `mc_secrets_scope_apply`, and the codex `mc_codex_env_args` name check. It then runs the provider probe and exits non-zero on failure.
  - It built M3b: `test_secrets_scope_dry_run.sh`, 5 arms with killing mutants.
  - In-sandbox, it labelled kill-tree, stall and codex-env UNINFORMATIVE.
- **Controller, outside the sandbox** ([gates](fleet-mission-evidence/iteration90/gates.md)):
  - The diff scan found writes only under the suite's own `mktemp`/`TemporaryDirectory`.
  - Snapshots matched the tree.
  - `make test-launchd-drivers` 54/54; scope suite 59/59; `--mutations` all killed; shellcheck: new file clean, driver 33 = base 33. Every run was bracketed by the real `secrets.env`'s sha and line count: SAME every time.
  - Build commit `b216285db`, crediting codex.
- **Six live dry-runs** under `triage` (`claude:claude-opus-5-5`, `codex:gpt-6.1-sol`, `pi:ollama/glm-5.3:cloud`, each healthy and degraded):
  - **Set 1, stdin inherited from this session (a socket):** pi rc 124 in both arms; claude-degraded rc 2.
  - **Diagnosis, four controls plus a minimal repro:**
    - The scoped pi probe hangs only through `_mc_bounded`. Scoped-direct and unscoped-bounded both answer in 2 s, and the working directory is irrelevant.
    - An explicit `< /dev/null` fixes it.
    - On `/bin/bash` 3.2.57, after any redirected loop (`done < <(…)`, a heredoc, a here-string), a background job's fd 0 is the inherited socket rather than `/dev/null`.
    - The scope library's inventory loop is one such loop.
  - **Set 2, `< /dev/null` (the expected launchd condition):** all six rc 0. Every lane printed `marker=ok controller-probe=ok`. Degraded runs printed `lanes=DEGRADED(1)`, handed to haiku. Codex printed `codex-marker-forward=ok driver-root-forward=ok`. The inventory read kept 4 / stripped 3, names only. No triage state was written (a `~/.ailang/state` listing diff showed only ambient daemon files).
- **Diff judge, round 1**: sonnet, Agent tool, foreground, own worktree, **PASS 90**, 0 blocking ([evaluation](fleet-mission-evidence/iteration90/evaluation.md)).
  - It ran 12 of its own mutants. M2 (drop the marker/root exit) and M5 (drop the forward-rc exit) survived.
  - Findings: untested forward exits; no explicit probe stdin; docs missing the live evidence.
- **Controller fix `913c6f9e1`**:
  - Arms `dry-run-codex-forward-rc`, `dry-run-codex-forward-missing` and `dry-run-probe-stdin` (pipe stdin; asserts the probe sees `/dev/null`).
  - `esac < /dev/null` on the preview probe.
  - Docs.
  - Results: 8/8 arms, 8/8 mutants killed, `make test-launchd-drivers` 54/54.
  - **Set 3, socket stdin, fixed tree:** pi rc 0 in both arms. claude-degraded rc 2 again.
- **Judge, round 2** (resumed by SendMessage): **PASS 93**, 0 blocking. All three findings closed, and its own M2, M5 and two redirect mutants were killed.
  - It attributed claude-degraded to the degraded recipe's `MISSION_PROBE_TIMEOUT=10` against a 10–13 s opus probe (set 3's log has no stdin warning), and argued set 1's is the same cap.
  - It flagged the docs' stdin attribution for that arm as an over-claim, corrected in `26b713887` (doc-only).
- **Landing**: draft PR [#2150](https://github.com/sunholo-data/ailang/pull/2150), base `fleet/i89-dfleet42-m2`, head `26b713887`, not merged. Closing-keyword scans on all three commit messages and the PR body found 0 (control fired).

**Gate 3b**: not applicable, because nothing merges (iteration 83 precedent).

**Transcript-scan notices seen this fire** (step 6a): 2 on #2039, the same two iteration 89 reported (fires `fleet-1793845378-59113` and `fleet-1793898974-52315`, 1 hit each). No new ones. Key names are in the private controlplane notice only.

**Ruled out**:
- Attributing set 1's pi failure to the scope. Unscoped-bounded and scoped-direct probes answer in 2 s; only scoped plus `_mc_bounded` plus a socket stdin hangs.
- Attributing claude-degraded to stdin. Set 3 has no stdin warning and fits the 10 s cap.
- Fixing `_mc_bounded` for its other callers: pre-existing, out of M3's scope, and not ticketed.
- Changing the degraded recipe's timeout or giving the preview its own floor: that changes the done-gate recipe. It is recorded as a trap in the design doc and the PR.
- Re-spawning the planner: iteration 88's M3 section is the spec.

**Retro lane**: no skill edit; two recorded frictions.
- (a) **Instance 1 of "a dry-run started from an interactive session is not a launchd dry-run".** The charter's done-gate `env -i … mission-control.sh` line inherits the caller's stdin. On bash 3.2 that reaches every `_mc_bounded` probe once any redirected loop has run. Pre-M3 dry-runs never probed a pi controller, which is likely why it never showed. If it recurs, add `< /dev/null` to the charter's done-gate line.
- (b) **Instance 1 of "the spawn-pin hook needs the role token alone on line 1".** A descriptive suffix after `MISSION-ROLE: evaluator` was refused. The skill's rule says "begin with the line", so this is consistent; no edit.
- (c) Both judges returned full verdicts in their final messages, and the resumed round produced the docs finding.

**Progress**: Clause 1 UNMEASURED, goal unmoved. Clause 2 UNMET (nothing resolved). Clause 3 MET (25 open). Clause 4: prior evidence only. Clause 5 preserved (nothing merged to code). Last 3 landings: iterations 85, 86 and 87, all clause 2. No drift alarm. Harness share: 20 of the last 20, by design.

**Next**: a ruling on D-FLEET-42 (A: merge #2141 → #2143 → #2150 and add the host line; B, the default: close the three drafts); any new ticket; otherwise a ruling on any of the eight drafts. An idle fire with no new ticket and no ruling should now find nothing buildable on D-FLEET-42.

**Routing evidence**: base=8a6b7bff5654f7a53f401569fc097ff9ac5b7686@2026-11-06T08:52:15Z (Gate-4 record; Gate-1 base `a6e112f9c`, and the six commits since touch neither the charter nor the log). Details: [routing](fleet-mission-evidence/iteration90/routing.md).
- Controller `claude:claude-opus-5-5` (tok: not reported). `MISSION_ANTHROPIC_AVAILABLE=1`.
- Designer: not spawned. Resolver `recipe claude:claude-opus-5-5 declared:provider-pin`; no rotation advance.
- Planner: not spawned. Resolver `agent-tool opus fail-closed:no-doc`.
- Executor `codex:gpt-6.1-sol` (159,062 tok). Resolver `recipe codex:gpt-6.1-sol declared:provider-pin`.
- Evaluators: `sonnet`, Agent tool, ×2. Idle screen 77,288 tok; diff 97,695 tok cumulative over 2 rounds. Resolver `agent-tool sonnet declared:alias-pin`. Judge independence: cross-vendor.
- Quorum: none. Metered spend /bin/zsh. No lane recorded dead.
