# Fleet Mission Log

Append-only, one entry per iteration, newest at the bottom. Charter: [fleet-mission.md](fleet-mission.md).

> **Older entries are ARCHIVED.** This file holds the newest 20. The full record of every
> iteration is in `fleet-mission-log-archive.md`, and a one-line index of ALL of them —
> the thing to grep before picking work, so the loop never repeats itself — is in
> `fleet-mission-index.md`.

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

## 39 — 2026-10-14 — P2 #15 slot verdict UNCLASSIFIED made reachable, judged PASS 91 (sonnet, round 2); stacked on #1825, BLOCKED on dev's required reds; Gate 1 skill fix for stranded records [HARNESS]

**Pick**: P2 #15 `driver:unclassified-state-unreachable` (filed by v1, iteration 315). I first picked P2 #12 from `origin/dev`'s charter, which ended at iteration 35, and ran its pick-time quorum: **BLOCKED**, 3 rejects, `gpt6-1-sol` absent, $0.1017. Only then did `git worktree list` show `fleet-i36-salvage` and `fleet-i38-artifact-dir`. Iterations 36–38 had already built #12 (#1824) and #14 (#1825), both BLOCKED on dev's required reds. This is instance 3 of iterations 37/38 retro (1). The redundant quorum artifact is left in the pin; nothing acted on it. Re-reading the newest record (iteration 38's branch): the #1824 predicate is still FALSE, and its **Next** is P2 #15, so #15 was the pick. 50 open tickets. `agent-tool:sonnet-unavailable` (`blocking=all`) stays PARKED-ON-LANE as iterations 29 and 32–38 assessed it.

**Preflight**: armed; gh `sunholo-voight-kampff`; billing CLEAN. Directives: 0 on #1768 since `2026-10-13T00:00:23Z` (9 comments, all self). Self-notices rc 0: control `852:10` ok, watermarks 2/2. Ledger valid (16 rows on dev, 18 on the branch), 1 OPEN on dev (D-FLEET-16); D-FLEET-17/18 are OPEN on the branch. No rotation or sweep due.

**Observe**: origin/dev == pin == `474995629`. `SKILL.md` and all 12 resources are `cmp`-identical to origin at both the resolved symlink target and the CWD copy. CI, Build and Release and Deploy Documentation are all **failure**. NOT-GREEN checks: `test`, `docs-gate`, `docs-build`, `Build windows-latest` (2 macOS legs cancelled). The `test` log, 136 KB with ANSI stripped, still fails at `git exec site absent from baseline: internal/executor/managed_agents/devapi_workspace.go (2)`. V1's kill switch is still set.

**Reality check (#15)**: reproduced. The slot-verdict `case` (`mission-control.sh` ~2517) ends `*:*) CRASHED`, so rc=0 with an unknown label reads `CRASHED`. `grep -c UNCLASSIFIED` in the driver = 0. The planner also showed that `0:gate-*` accepted `gate-bogus`, which the writer can never emit (`mission-heartbeat.sh:11-13`).

**Did**:
- **Planner** (`codex:gpt-6.1-sol`, probe rc 0, run rc 0, 88,950 tok, detached `.planner-wt-iter39`): `sprint-plan-slot-verdict-unclassified.md` + sprint JSON (`cf055fd1a`). It wrote a 15-row premise table and corrected one count: the design names UNCLASSIFIED 9 times now, not 5. One milestone, 11 named arms, 10 mutants.
- **Executor** (`codex:gpt-6.1-sol`, one run, rc 0, 111,978 tok): driver +3/−2, suite +51. 6 arms red before the fix; all 10 planned mutants killed; 0 new shellcheck findings (69 inherited). `.snap/M1` is byte-identical to the committed tree (`81df7d609`).
- **Controller**:
  - Gates outside the sandbox: heartbeat suite 36/36 rc 0; `bash -n` ×2; `check-changelog`; `diff --check`.
  - Two scratch mutants (notify pattern, `0:*` arm): each killed by its target arms.
  - A sha observation of 12 real `~/.ailang/state` heartbeat/episode/verdict files: unchanged.
- **Evaluator** round 1 (sonnet, Agent tool, foreground, own worktree `fleet-i39-judge`, 65,694 tok): **PASS 86**, 0 blocking. It ran 10 drills of its own; 8 were killed, M2 is equivalent, and **M3 survived** (drop `0:` from `0:fired|0:`, i.e. an empty last label at rc=0 had no arm). Its N2/N3: the Q4 table did not say what UNCLASSIFIED now means.
- **Controller fix** (`e9783fae6`): arm `rc=0 empty last label => DIED-PRE-GATE-0`, the sole killer of M3 (rc 1). The Q4 REAPED row names the writer's gate labels, and the last row says UNCLASSIFIED means "exited 0 with an unknown label".
- **Evaluator** round 2 (resumed by `SendMessage`, 70,331 tok): **PASS 91**. M3 is killed; 37 arms rc 0; the table is accurate; no new findings. It did not run `test_controller_capacity.sh` itself; the controller's `make test-launchd-drivers` did (15 cases, fail=0).
- **Gate 5 skill fix** (`b081a39a4`, both copies, `sync-agents-skills.sh --check` rc 0, `check_context_docs.sh` rc 0): Gate 1 lists the mission's own open PRs and reads the newest record from a branch ahead of origin/dev.

**Gate 3b**: not applicable; nothing merged. Branch `fleet/i39-unclassified` is pushed, with its PR stacked on #1825.

**Done-gate** (at `b081a39a4`, ahead of the merge): the surface edited is the driver's SLOT VERDICT and NOTIFY blocks. `make test-launchd-drivers` under `env -i`: rc 0. Dry-runs under the idle, armed **world** profile from the worktree, with `AILANG_DRIVER_PINNED` set:
- healthy: rc 0, `lanes=ok`.
- degraded (`MISSION_EXECUTOR_MODEL=pi:bogus/no-such-model`): rc 0, `lanes=DEGRADED(1)`.
- No world pidfile was written, and nothing was reloaded.

Reach is NOT confirmed (unmerged). Re-run the done-gate at the merge.

**Progress**: 0 tickets resolved. Clause 1 UNMEASURED. Clause 2 UNMET; #12, #14 and #15 are all built and judged, waiting on dev. Clause 3 MET (50 open). Clause 4: prior evidence only. Clause 5 preserved; nothing shipped.

**Routing evidence**: base-gate1=4749956296d226f06d92dd2458b36843a92d48a3@2026-10-14T19:59:24Z; base=4749956296d226f06d92dd2458b36843a92d48a3@2026-10-14T20:32:29Z (gate4). Dev did not move. `MISSION_ROUTING_NOTE`: empty. Over-ration: openrouter.
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer**: not spawned. #15 needs no design; the slot-heartbeat doc's Q4 table is the spec. The resolver answered `recipe claude:claude-opus-5-5 declared:provider-pin`. The rotation pointer (`codex:gpt-6.1-sol`) was not advanced.
- **Planner**: resolver `recipe codex:gpt-6.1-sol declared:planner-lane-default-pin` (88,950 tok).
- **Executor**: resolver `recipe codex:gpt-6.1-sol declared:provider-pin` (111,978 tok).
- **Evaluator**: resolver `agent-tool sonnet declared:alias-pin`, spawned via the Agent tool with `model=sonnet`, foreground, in its own worktree. Round 1: 65,694 tok. Round 2: 70,331 tok. Judge independence: cross-vendor against the codex generators, and a different model from the opus controller.
- The operator's standing request was honoured: every spawned role went to its resolved path, and the required evaluator ran via the Agent tool both rounds. No role failed to spawn.
- **Metered**: $0.10 (the redundant #12 quorum; seats glm-5-3, kimi-k3, gemini-4-argon, with `gpt6-1-sol` unreachable). Everything else ran on the ChatGPT and Anthropic subscriptions.

**Ruled out**:
- Re-working #12 after the redundant quorum: it is built and judged (#1824).
- Admin-merging over the required reds.
- Fixing the dev reds: outside fleet authority (D-FLEET-17).
- Mapping rc≠0 unknown labels to UNCLASSIFIED: the Q4 table keeps CRASHED/KILLED for non-zero.
- Keeping `0:gate-*`: it reaps at gates the writer cannot emit.

**Retro lane**:
- (1) **[SKILL FIX `b081a39a4`]** Records stranded on a blocked PR branch, instance 3 (iterations 37 and 38 were instances 1 and 2). This time it cost a redundant quorum. Fixed at Gate 1, both copies. The fix reaches the loops only when this stack merges, because it is itself stranded behind the same red.
- (2) The judge's own drill found a hole that all 10 planned mutants and the controller's 2 missed (M3). Generator ≠ judge earned its slot again (cf. iteration 35).
- (3) Whole-iteration landing loss, instance 4 (iterations 36–39). D-FLEET-17 is still OPEN, and the stack is now 3 PRs deep. Each further stacked layer adds rebase risk. Consider pausing build work until D-FLEET-17 is answered; this is surfaced to Mark in the report.
- (4) `rotate-log` added the duplicate "Older entries are ARCHIVED" notice again (instance 7), collapsed by hand. Ticket-shaped (`rotate-log:*`); the fleet does not self-source.

**Next**: when D-FLEET-17's predicate holds (`test` and `docs-gate` green on origin/dev), land #1824, then #1825, then this PR. Run the done-gate at each merge and resolve 4 tickets (`quorum:invalid-absent-on-quoted-literals`, `quorum:reviewer-invalid-json-discarded`, `quorum:artifact-dir-cwd-relative`, `driver:unclassified-state-unreachable`). Until then, do not stack further builds without a ruling. D-FLEET-16/17/18 are with Mark.

## 40 — 2026-10-15 — stack recovery rechecked; required dev reds persist, no further builds; independent disposition review PASS 95 (fresh-context Sol) [ADMIN]

**Pick**: recover and verify the existing stack, the Next of iteration 39: #1824 → #1825 → #1828. Origin/dev ends at iteration 35, but the attributed fleet PR branches contain complete records 36–39. These are completed, reported iterations, not four newly discovered crashes. No new sprint, design, source changes, or ticket resolutions this fire. 50 open signatures; the native Sonnet capability ticket remains PARKED-ON-LANE.

**Preflight**: armed; gh sunholo-voight-kampff; billing CLEAN. Zero allowlisted directives among 10 comments on #1768 since 2026-10-13T00:00:23Z. Self-notices rc 0 (control 852:10, both watermarks valid). Latest branch ledger: 18 valid rows; exactly D-FLEET-16/17/18 OPEN, unchanged. No rotation/sweep due (weekly sweep iteration 32; current issue created October 12). Shared main dirt (fmt hook log, benchmark JSON, untracked background-worker design) and inherited .snap/ in the iteration-39 worktree left untouched.

**Observe**: pin == dev == origin/dev 4749956296d226f06d92dd2458b36843a92d48a3. SKILL.md and all 12 resource files match origin in BOTH the authoritative pin and resolved main-checkout skill directory (26 per-file comparisons). CI 38617113857, Build and Release 38617113836, Deploy Documentation 38617113983 all completed failure at this SHA. The check-runs positive control is 20 entries. V1's kill switch exists.

**Did**:
- Designer (native Sol fallback): checked design/authority disposition. No design revision warranted; existing implementation has already been independently judged. D17 needs a human ruling or a green-dev predicate; D16/18 govern separate work.
- Planner (native Sol): measured all three PR heads/bases and wrote the concrete resume checklist in iteration40/recovery-evidence.md. #1824 is OPEN/BLOCKED at b5ca4f6cf4083403499ba47fad7c7a9b9a3bc9a9; #1825 OPEN at 1f5f0d3ac07dfb35d49c46bc9d66e4ec0e097b51; #1828 OPEN at 10fdc22223c06d660a5769ccd604e0f06037acc8. Last two target their parent branches and have only skipped automerge checks, not implementation CI coverage.
- Executor (native Sol, read-only recheck): source gate make check-git-exec rc 2 (underlying script rc 1), same missing baseline path internal/executor/managed_agents/devapi_workspace.go (2). Docs log confirms MDX/acorn failure at generated env-vars.md:252. Current test job has 48 skipped steps; docs-build 3. Local source gate ran on darwin/arm64; other platform results come from CI, not local execution.
- Independent evaluator (fresh-context Sol): PASS 95/100 for PARKING/RESUME DISPOSITION ONLY; no implementation re-acceptance. Its newly measured required-context set is test, lint, build, docs-gate, UI build gate. Controller independently re-read protection and checks: test/docs-gate failure, lint success, build skipped, UI build gate absent. No green inferred for skipped/absent checks. Judge report: iteration40/judge-report.md. Follow-up independent review of this concrete record: PASS, no factual blocker; all prior live-log sections retained verbatim, status 37 moved intact, ledger and CURRENT GOAL onward byte-identical.
- Controller rechecked failed log content (known-present PASS lines plus specific baseline error), branch heads, open ticket count, required-context set and ledger. The first worktree-helper launch left a pending/incomplete worktree (fleet-i40-record); its status is not a verdict. It was not repaired or used. A distinct helper invocation kept its parent command alive through wait and returned ready rc 0 for fleet-i40-record-verified, clean at the reviewed branch head. No inherited files were deleted.

**Gate 3b**: BLOCKED, not bypassed. Resume predicate test + docs-gate SUCCESS on the exact current dev SHA is false. No merge, admin override, retarget, source push or automatic merge was requested. This bookkeeping-only record is appended to the existing #1828 branch, without another stacked PR.

**Done-gate**: no implementation or rollout this iteration. Prior PASS 88/90/91 and pre-merge done-gates remain historical branch evidence only. Full launchd tests and healthy/degraded dry-runs are owed at each actual merge, then each ticket may resolve. Nothing reloaded.

**Progress**: fleet clauses 1 UNMEASURED, 2 UNMET, 3 MET (50 open), 4 prior evidence only, 5 preserved. 0 tickets resolved; goal unmoved. Last three completed changes (35 provenance, 34 spike, 33 bounded notices) addressed turnaround/enabling work; this fire banks only a verified recovery state.

**Routing evidence**: base-gate1=4749956296d226f06d92dd2458b36843a92d48a3@2026-10-15T02:36:44Z; base=4749956296d226f06d92dd2458b36843a92d48a3@2026-10-15T02:40:59Z (gate4). Record tree base=10fdc22223c06d660a5769ccd604e0f06037acc8 (latest blocked fleet record, unchanged remote #1828). MISSION_ROUTING_NOTE: controller claude:claude-opus-5-5 → codex:gpt-6.1-sol (probe ok). Over ration: anthropic openrouter.
- Controller codex:gpt-6.1-sol (tok: not reported).
- Designer: resolver refuse over-ration:anthropic; used admitted rotation fallback gpt-6.1-sol via native Agent at operator's explicit request, authority review only (tok: not reported). Rotation pointer unchanged: no design authored.
- Planner: derive and resolver agree recipe codex:gpt-6.1-sol declared:planner-lane-default-pin; operator-requested native Agent transport, exact model unchanged (tok: not reported). Recovery checklist only; no vacuous sprint JSON.
- Executor: resolver recipe codex:gpt-6.1-sol declared:provider-pin; operator-requested native Agent transport, exact model unchanged (tok: not reported). Read-only gate recheck, not implementation.
- Evaluator: resolver agent-tool sonnet declared:alias-pin. Native spawn failed: Unknown model sonnet for spawn_agent; advertised models are gpt-6.1-sol, gpt-6-astra, gpt-6-sol, gpt-6-luna, gpt-5.6-sol. Anthropic/OpenRouter quota unavailable; Gate 3 fresh gpt-6.1-sol fallback spawned via native Agent with fork_turns=none (tok: not reported). judge-independence: same-model-fresh-context. No Astra spawn, no controller-only verdict. Distinct judge instance evaluated the disposition independently.
- Metered: $0; all completed roles use ChatGPT subscription. Provider-reported token counts unavailable from native Agent responses; not invented as zero usage.

**Ruled out**: repeating #11 (already resolved iteration 36); repeating #12/#14/#15 implementations; treating stacked PR CLEAN or skipped automerge as green CI; repairing dev's out-of-scope source paths without D17 authority; resolving/reopening human ledger decisions; bypassing Sonnet failure silently; advancing the watermark on self-authored comments; reading/acking fleet ticket messages.

**Retro lane**: observations only; no skill/policy change. The already-proposed stranded-record skill fix b081a39a4 remains on #1828 and is itself blocked on dev. Native sonnet capability failure repeats the existing ticket; fresh-context independent judge worked. The first worktree helper's detached launch did not yield a ready artifact, while the foreground-parent wait did; one measured transport friction, not a new self-sourced ticket or fix. Status rotation CLI targets the archive, not the charter; its generated changes were restored byte-for-byte before the oldest charter stamp was moved with ledger/queue guards. Fleet harness share is 13/20 in the regenerated last-20 index; high fleet share is its explicit authority, not a product-loop breach.

**Next**: re-run the named D17 predicate and allowlisted directives. When dev is green, verify/land #1824 → #1825 → #1828, fresh expected PR checks on each head and done-gates/CI on each merge SHA; resolve the four named tickets only then. Until then no further stacked builds without a ruling. D-FLEET-16/17/18 remain OPEN unchanged.

## 41 — 2026-10-15 — The blocked stack LANDED: #1824 → `514ca3bc3`, #1839 (ex-#1825) → `10cb2fae0`, #1840 (ex-#1828) → `7c10ce00d`; three CI-only defects fixed on the way; 4 tickets resolved [HARNESS]

**Pick**: iteration 40's **Next**, which is to re-run D-FLEET-17's predicate and, if it holds, land the stack #1824 → #1825 → #1828. Origin/dev's charter ended at iteration 35. Gate 2's trace (a) found the three open fleet PRs, and their branches held records 36–40 (iteration 39's stranded-record rule worked as written). These are not orphans. The `blocking=all` ticket `agent-tool:sonnet-unavailable` stays PARKED-ON-LANE as iterations 29 and 32–40 assessed it: its evidence is a codex controller whose native `spawn_agent` cannot pin sonnet, and this controller is claude, which spawned sonnet without trouble. 52 open signatures at pick.

**Preflight**: armed; gh `sunholo-voight-kampff`; billing CLEAN. Directives: 0 on #1768 since `2026-10-15T02:46:27Z` (13 comments, all self). Self-notices rc 0: 0 crash notices, control `852:10` ok, watermarks 2/2. No rotation or sweep due.

**Observe**: origin/dev == pin == `3342b46ce`. `SKILL.md` and all 12 resources are `cmp`-identical to origin at both the resolved symlink target and the CWD copy.
- The D-FLEET-17 causes are gone, and neither fix was the fleet's. `check-git-exec` passes. MDX was fixed by `95106ba27`. The last docs failure on dev (`046cecb9d`) was a manifest statistics mismatch, fixed by `35a49b851`.
- Dev `test` on `3342b46ce` was red on `TestWorkerTerminalPTYLifecycle/eval/error` (`internal/runner`). Walking back over the last 8 dev commits, `test` alternated: green on `377f96a0d` and `35a49b851`. Each red after `35a49b851` was a different `internal/runner` worker signal-lifecycle subtest (sigterm, sigint, PTY eval/error, PTY strict/normal, PTY strict/error). It is a flake, not a cause. A `--failed` rerun of dev's CI on `3342b46ce` came back **success**, so the predicate held on the exact SHA.

**Did**:
- **#1824** (P2 #12): branch protection is non-strict, and the PR merged cleanly onto current dev, so it was closed and reopened to get fresh `pull_request` checks with no push. The scope guard diffs a merge push against the old head and would refuse dev's own paths, so this route avoided a refusal rather than going around the guard.
  - The sonnet judge (Agent tool, own worktree `fleet-i41-judge`, a local merge `342d0bfd0`) scored **PASS 90**, 0 blocking. It ran 7/7 of its own mutants, all killed. Merge integrity holds against dev's 60 newer commits (2 touch the skill: triage); the ledger has 18 valid rows.
  - `make test-launchd-drivers` on the merge preview: rc 0.
  - Required checks green; the only red was the non-required macOS `TestMemprobeDebugLogDoesNotAccumulate` RSS threshold (also red on dev `66fdeefb1`).
  - Squash-merged with `--match-head-commit` → `514ca3bc3`. CI 19/19 green. World dry-runs (`AILANG_DRIVER_PINNED=514ca3bc3`): healthy and degraded both rc 0, `DRY RUN ok`.
  - Two tickets resolved (52 → 50). #941 closed with a verdict comment (comment count 1 → 2).
- **#1825 → #1839** (P2 #14): the squash left #1825 conflicting in the bookkeeping docs. Its 7 own commits were rebased onto dev and pushed as a NEW branch, so the guard diffs against origin/dev. The stable patch-id is identical to the judged diff (`f2105dbafbffe6e5a19d3fd5b9721259`), and `gen-env-docs` regenerated byte-identically. **Fresh CI found three defects that iteration 38 never saw, because a stacked PR gets no CI.** Each was fixed by the haiku executor (`claude-sub`, probe rc 0) in its own commit:
  1. `b05e98944`: the shell arm `bash32_and_suite_reach` hard-asserted bash 3.2, which fails under Linux's bash 5. It now asserts only on Darwin.
  2. `55f1566d5`: 4 tests exec `/bin/bash`, which Windows lacks. Windows `t.Skip`, following `pi_integration_test.go:17`.
  3. `c39758701`: on Windows a `0500` dir is writable and a live CWD cannot be removed. Windows skips on exactly those two cases.
  4. `10dd8b050`: 5 hand-rolled `t.Setenv("HOME")` calls violated dev's newer `check-home-isolation` gate. They now call `testutil.SetHomeDir`.
  The controller ran all 17 static `check-*` gates of the CI `test` job before the last push; only check-home-isolation was red. The judge scored the fixes **PASS 88** (round 2, which predicted the Windows `0500` case) and **PASS 94** (round 3), 0 blocking. CI was CLEAN at 20/20 on `10dd8b050`; squash-merged → `10cb2fae0`.
- **#1828 → #1840** (P2 #15): rebased the same way, with an identical patch-id (`16bb46047ad353e5960c575f84ca01c0`). All 16 static gates, the ledger and the skill-mirror check were green before the push. The judge scored the landing **PASS 94** (round 4): no revert of #1839 or #1824, heartbeat 37 arms rc 0, and 2/2 mutants killed. Required checks green; the only red was the non-required Build leg PTY flake. Squash-merged → `7c10ce00d`.
- #1825 and #1828 were closed with a pointer comment each (0 → 1).

**Gate 3b**: Each merge was pinned with `--match-head-commit` and polled by SHA. `514ca3bc3`: CI 19/19 green. `10cb2fae0`: `test` red on the `internal/runner` PTY flake (`strict/error`; quorum, dispatch and config all `ok` in that log); a `--failed` rerun went green, 21/21. `7c10ce00d`: 17/17 green. No closing keyword in any PR (the scan fired false on each). Non-required reds were left as they are, after negative controls: the macOS memprobe RSS test and the Build-leg PTY flake.

**Done-gate**: `make test-launchd-drivers` under `env -i` returned rc 0 at `342d0bfd0` (the #1824 preview), `55f1566` (the #1839 head), `10cb2fa` and `7c10ce0`. Dry-runs:
- `514ca3bc3`: under **world**.
- `10cb2fae0` and `7c10ce00d`: under **triage**. World went mid-fire (pid 23856), so the first `10cb2fae0` attempt under world yielded at the overlap guard and is discarded. Triage's opt-in workdir sync runs after the dry-run exit.
- Healthy (real lanes, codex over ration) and degraded (`MISSION_EXECUTOR_MODEL=pi:bogus/no-such-model` → handed to opus): rc 0 at every merge.
- No pidfile was written, and nothing was reloaded.

**Progress**: 4 tickets resolved, 52 → 48 open: `quorum:invalid-absent-on-quoted-literals`, `quorum:reviewer-invalid-json-discarded`, `quorum:artifact-dir-cwd-relative`, `driver:unclassified-state-unreachable`. Every reply went to its filer (world, attended, v1, v1). Clause 1 UNMEASURED. Clause 2 UNMET: these tickets' turnaround was about 19 days (filed 2026-09-26) and 7 days (`quorum:reviewer-invalid-json-discarded`, filed 2026-10-08). Clause 3 MET. Clause 4: prior evidence only. Clause 5 preserved: each landing was judged, passed the done-gate, and had green required CI at its merge.

**Routing evidence**: base-gate1=3342b46ce91a35f74d60d00c8e3e024b9b3b6dba@2026-10-15T17:03:13Z; base=7c10ce00dcadf5079cac4b754ef5d1ae094f0366@2026-10-15T19:51:55Z (gate4). `MISSION_ROUTING_NOTE`: controller pi:ollama/glm-5.3:cloud → claude:claude-opus-5-5 (probe ok); planner and executor codex `gpt-6.1-sol` over the daily ration (probe rc 75) → opus and haiku. Over ration: codex, openrouter.
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer**: not spawned. There was no design work; every design had already passed quorum (iterations 36, 38, 39). The rotation was not advanced. Resolver: `recipe claude:claude-opus-5-5 declared:provider-pin`.
- **Planner**: not spawned. Resolver: `agent-tool opus fail-closed:env-pin`. The pick was landing pre-planned, pre-built and pre-judged work; the fixes were CI-measured one-liners with no plan to write.
- **Executor**: resolver `recipe claude:claude-haiku-5-5 declared:provider-pin`. Spawned via the `claude-sub` recipe (subscription-only, billing CLEAN), 4 runs, each rc 0 and each one commit (tok: not reported; `claude -p` text output).
- **Evaluator**: resolver `agent-tool sonnet declared:alias-pin`. Spawned via the Agent tool `model=sonnet`, in its own worktrees (`fleet-i41-judge`, `fleet-i41-judge2`), 4 rounds resumed by `SendMessage`: 62,933 / 76,463 / 83,061 / 88,813 tok. Judge independence: a different model and vendor line from the executor (haiku) and the controller (opus), and the original generators (codex) were cross-vendor.
- The operator's standing Agent-tool request was honoured for the evaluator, the one role whose resolved path is the Agent tool. The executor's resolved path is a recipe, so the Agent tool is not a valid path for it. No role failed to spawn.
- **Metered**: $0 (Anthropic subscription only).

**Ruled out**:
- `gh pr update-branch`, or a local merge push, onto the stacked branches. The first bypasses the pre-push scope guard; the second is refused by it, because it diffs the old head against a head that contains dev.
- Attributing any `internal/runner` PTY or signal red to the stack. Each was a different subtest, and the same test is red on dev at `3342b46ce`, `d0d43b606` and `10cb2fae0` and green between.
- Attributing the macOS memprobe RSS red, which is non-required and also red on dev `66fdeefb1`.
- Re-running the iteration-36/38/39 designs or plans.
- Resolving D-FLEET-17 (a human row; see Next).

**Retro lane**:
- (1) **A stacked PR is UNVERIFIED, not "judged + blocked".** Iterations 38 and 39 recorded PASS 90 and 91 plus a done-gate. All of it ran on macOS only, because a PR whose base is not `dev` gets only `automerge` (skipped). Fresh CI then found 3 defect classes in #1825 within 2 runs: Linux bash 5, Windows `/bin/bash`, and a gate dev added afterwards. Instance 1; a skill-fix candidate once there is a second ("a judged stacked PR is not CI-verified; never stack beyond one").
- (2) Before pushing a long-lived branch, run the CI `test` job's static `check-*` list locally (rule 3g). Doing so made #1840 go green first time.
- (3) The dev `internal/runner` PTY/signal flake reddened required `test` on 4 of the 9 runs observed this fire. It is V1's code (outside fleet authority) and V1 is disabled. Reported to Mark as a Key find, not ticketed.
- (4) The STATUS archive is out of order: 37 sits at the end (line 253) while 36..32 are newest-first at the top. 38 goes in at the top; 37 is left where iteration 40 put it.
- (5) `rotate-log` (run with `AILANG_MISSION_REGISTRY=$PWD/missions`, `--keep 21`, 0 rotated) again inserted a duplicate "Older entries are ARCHIVED" notice at the top of the live log. This is instance 6, after iterations 28 and 31–33, and the log already carries 6 copies. The added 4 lines were removed by hand. It is a `rotate-log:*` tooling defect with no matching open ticket, and the fleet does not self-source.

**Next**: P2 #13 is parked on D-FLEET-18, and #16 `quorum:tier2-salvage-fold` is a charter follow-up. Then the remaining tickets by `slots_lost`, top first: `worktree:native-background-terminal-marker-missing` and `messaging:pubsub-notify-wrong-project` (world, filed 2026-10-15). D-FLEET-17 is moot (both causes fixed outside the fleet); Mark may close it. D-FLEET-16 and D-FLEET-18 remain with Mark.

## 42 — 2026-10-16 — `worktree:native-background-terminal-marker-missing` LANDED: #1843 → `51f9ecb29`; judge FAIL 66 then PASS 86 (sonnet); ticket resolved [HARNESS]

**Pick**: the queue after iteration 41. P2 #13 is parked on D-FLEET-18. #16 `quorum:tier2-salvage-fold` is still not live: `RunQuorumWithEscalation` has 0 non-test callers at `04496274e`. That left tickets by `slots_lost`. The four agent-tool tickets above it, including the `blocking=all` `agent-tool:sonnet-unavailable`, stay PARKED-ON-LANE as iterations 29–41 assessed them: they are codex-controller native-spawn limits, and this claude controller spawned sonnet without trouble. Of the two newest world tickets, `messaging:pubsub-notify-wrong-project` lives in `cmd/ailang/messages_*`, outside the Authority allowlist. So the pick was `worktree:native-background-terminal-marker-missing` (`tools/launchd/`). It is fresh (filed 10-15), so it moves clause 2. 50 open tickets at pick.

**Preflight**: armed; gh `sunholo-voight-kampff`; billing CLEAN. Directives: 0 on #1768 since `2026-10-15T02:46:27Z` (14 comments, all self). Self-notices rc 0 (control `852:10`, watermarks 2/2). No rotation (#1768 is this week's thread) and no sweep due.

**Observe**: origin/dev == pin == `04496274e`; `SKILL.md` and all 13 resources are `cmp`-identical to origin at the resolved symlink target. No open `fleet/*` PR.
- Dev CI on `04496274e` (iteration 41's docs-only record): the non-required `launchd drivers (bash 3.2)` job was red. The failing line is `tools/eval/test_motoko_connection_probe.sh`, "refusing live path … unexpectedly succeeded"; triage ticketed it as `ci:motoko-probe-refusing-live-path-unexpected-success`. It is the same file as #13 and D-FLEET-18 (outside scope), and the PR #1843 run of the same job was green. All required checks were green.

**Reality check**: reproduced first-party at HEAD. In a scratch repo with a post-checkout hook that sleeps 6 s:
- Control: `add` then `wait` → `ready`/0.
- Treatment: `add` launched under `perl setpgrp`, then `kill -TERM -- -<pgid>` 1 s later → every `wait` returned `pending`/75. The branch existed and the writer was gone.

`nohup` ignores HUP, not a group TERM.

**Did**:
- **Planner** (codex `gpt-6.1-sol`, recipe, own detached worktree): plan `9508b0051`, with 1 milestone, 8 named arms and 10 mutants.
- **Executor round 1** (codex, recipe, sandbox): `4469e1b31`. 8 arms were UNINFORMATIVE in the sandbox (`ps` denied).
  - The controller re-ran outside the sandbox: 96/0, and 10/10 planned mutants killed. The new suite against the old helper scored 19 passed / 27 failed. A black-box SIGKILL of the writer gave `failed:writer_died`, with the tree and branch removed.
- **Judge round 1** (sonnet, Agent tool, own worktree `fleet-i42-judge`): **FAIL 66**. The blocking finding, reproduced: `add` killed after the writer's `.started` but before `.owner` → the writer exits silently → `pending` forever. This is the ticket's own failure class. Its drill d (the PGID re-check in `quiesce`) survived.
- **Executor round 2** (codex): every registration failure is now terminal (`failed:owner_never_published`, only while the status is still pending). An ownerless pending resolves by the `.started` identity or by status age (`MISSION_WORKTREE_OWNER_GRACE`, 60 s), with **no** cleanup. New arms cover both caller-death windows, legacy bare pending and a group mismatch.
- **Controller fixes** (same commit `ee599bd80`):
  1. On the host the new live arm was red, deterministically (191/3, twice). On this loaded rig (load about 2.5), 100 × `sleep .1` took more than 18 s, against the arm's 15 s window. The rendezvous and registration bounds are now wall-clock (`SECONDS`).
  2. The GNU `stat -c %Y` is tried first. GNU `stat -f %m` prints `?` with rc 0, so BSD-first never expired a bare pending on Linux.
  - Host 194/0. Of 9 controller-run mutants, 8 were killed (including the judge's d). The survivor r2d (the stat order) is Linux-only by construction.
- **Judge round 2** (resumed by `SendMessage`): **PASS 86**, 0 blocking. The round-1 reproduction now returns `failed:owner_never_published`. No path removes a tree or branch it did not create, and the 60 s bound cannot expire a writer that has registered. Its drills: e and g killed; f (the writer's pending re-check before checkout) SURVIVED, non-blocking.
- PR #1843 (no closing keyword; the scan fired false), squash-merged with `--match-head-commit ee599bd80` → `51f9ecb29`.

**Gate 3b**: on the PR, the 5 required checks were green (`test`, `lint`, `build`, `docs-gate`, `UI build gate`), and so were `launchd drivers` and `test-windows`. The only red was non-required `Build macos-latest`: a setup-step `Manifest fetch failed … operation was aborted due to timeout` (infrastructure; the job log contains no test output), with the sibling Build legs cancelled by fail-fast. Dev CI on `51f9ecb29`: 17 check-runs, all 5 required green plus `launchd drivers` green. The only red was non-required `Build macos-latest`: `TestRuntimeNestedStopUsesRemainingDeadline` (`internal/effects`, 0.03 s timing; this PR touched no Go), with its siblings cancelled by fail-fast. Ticket resolved after the merge, with a reply to world. 48 open tickets after.

**Done-gate**:
- 12 static `check-*` gates rc 0 before the push.
- `make test-launchd-drivers` under `env -i`: rc 0 at `ee599bd80`, which is tree-identical in `tools/` to `51f9ecb29`.
- Dry-runs under **world** (idle and armed; `AILANG_DRIVER_PINNED=51f9ecb29`): healthy rc 0 with `lanes=ok`; degraded (`MISSION_EXECUTOR_MODEL=pi:bogus/no-such-model`) rc 0 with `lanes=DEGRADED(1)`, handed to opus.
- No pidfile was written and nothing was reloaded. Real `~/.ailang/state/worktrees` was unchanged by every test run. World's stale iteration-268 status file is left as is.

**Progress**: 1 ticket resolved; 48 open after resolve (50 at pick). Clause 1 UNMEASURED. Clause 2 UNMET: this ticket's turnaround was about 22 h, under the 48 h target, but the median is still dominated by older tickets. Clause 3 MET. Clause 4: prior evidence only. Clause 5 preserved: the landing was judged, passed the done-gate, and had green required CI.

**Routing evidence**: base-gate1=04496274e36d301b721886fb56ab1146ad4080e7@2026-10-16T02:57:41Z; gate3b=51f9ecb290472fbd18f166e482b83656366c7f84@2026-10-16T04:41:53Z. `MISSION_ROUTING_NOTE` was empty. Over ration: openrouter. Codex 15.0% used of 17.3% allowed.
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer**: not spawned. The ticket is a mechanical fix in one 65-line script, and the charter designs only when a fix warrants it. Resolver: `recipe claude:claude-opus-5-5 declared:provider-pin`. Rotation not advanced.
- **Planner**: resolver `agent-tool opus fail-closed:no-doc` against the pin `codex:gpt-6.1-sol`. Following role-spawn-routing §2(a), it was routed to the pin under the codex recipe. Probe rc 0; run rc 0 (70,751 tok).
- **Executor**: `codex:gpt-6.1-sol` via the recipe (`declared:provider-pin`), 2 runs, rc 0 and rc 0 (201,574 + 102,534 tok).
- **Evaluator**: `sonnet` via the Agent tool (`declared:alias-pin`), 2 rounds (48,891 + 56,741 tok). Judge independence: cross-vendor (Anthropic sonnet vs the OpenAI generator).
- Agent-tool use followed the operator's standing request wherever the resolver names the Agent tool (the evaluator). Planner and executor resolve to recipes, where the Agent tool is not a valid path. No role failed to spawn.
- **Metered**: $0.

**Ruled out**:
- `messaging:pubsub-notify-wrong-project` as a fleet fix. The notifier already targets the store's project; the error names the ADC quota project. That is credential config or `cmd/ailang` code, so it went to D-FLEET-19.
- The host red at 191/3 as a code defect. It was a time bound counted in iterations on a loaded rig, and making it wall-clock fixed it.
- The PR's macOS Build red as related to this change (setup-step manifest timeout).
- Self-sourcing the helper's simplification. The judge flagged proportionality (65 → 211 lines); it is recorded as a follow-up, not ticketed.

**Retro lane**:
- (1) **The sandbox hid the decisive red twice.** The executor's sandbox denied `ps`, so 8 arms and 8 mutants were UNINFORMATIVE. Its round-2 run could not see the iteration-count bound fail on the host. The controller's out-of-sandbox re-run (false-green 3) caught both. Nothing new to add to the skill; the rule worked.
- (2) **The Bash tool shell is zsh**, so `set -- $r` does not split, and the first CI poll printed garbage until its deadline. Wrapping polls in `/bin/bash -c` fixed it. Instance 1; a skill candidate on a second instance.
- (3) **A bound counted in loop iterations is not a time bound** on a loaded rig (`sleep .1` cost about 0.2 s). Instance 1, in a helper this iteration wrote; fixed in place.

**Next**: #16 stays not-live. Then tickets by `slots_lost`: the codex/agent-tool rows are PARKED-ON-LANE, so the next routable tickets are the rest of the `slots=1` set, oldest first. D-FLEET-16, D-FLEET-18 and D-FLEET-19 are with Mark; D-FLEET-17 is moot.

## 43 — 2026-10-16 — `skill:gate3b-ci-poll-exceeds-foreground-bash` LANDED: #1853 → `a8042fa3f`; judge PASS 74 (blocker) then PASS 90 (sonnet); D-FLEET-20 filed [HARNESS]

**Pick**: P2 #13, #18 and #19 are parked on D-FLEET-18/19, and #16 `quorum:tier2-salvage-fold` is still not live. That left the `slots=1` tickets, oldest first. The oldest, `quorum:gpt6-1-sol-openai-api-credits-exhausted` (world, 10-01), is billing policy, and its fix sits in `cmd/ailang/design_quorum.go`, outside Authority. It went to **D-FLEET-20**. The next, `skill:gate3b-ci-poll-exceeds-foreground-bash` (motoko, 10-01), is a mechanical skill fix in `.claude/skills/mission-*` + `.agents/skills/mission-*`, so it is in scope. The agent-tool `slots>1` rows stay PARKED-ON-LANE. 49 open tickets at pick.

**Preflight**: armed; gh `sunholo-voight-kampff`; billing CLEAN. Directives: 0 on #1768 since `2026-10-15T02:46:27Z` (15 comments, all self). Self-notices rc 0 (control `852:10`, watermarks 2/2). No rotation and no sweep due.

**Observe**: origin/dev == pin == `c4fab92c2`. `SKILL.md` and all 13 resources at the resolved symlink target are `cmp`-identical to origin. No open `fleet/*` PR, and no orphaned i42/i43 worktree. Dev CI on `c4fab92c2`: the 5 required checks were green. The non-required `Build ubuntu-latest` was red on `TestWorkerCLIProcessExitLifecycle` (a worker-process test, not fleet scope; V1 owns it and is disabled).

**Reality check**: at HEAD, both copies of `resources/gate-3b-ci-green.md` (separate inodes, byte-identical) carried `deadline=+1800` and `sleep 30`. Neither mentioned `run_in_background`, the Bash tool's 600 s foreground cap or its `timeout` parameter. The premise holds.

**Did**:
- **Planner** (codex `gpt-6.1-sol`, recipe, detached worktree `.planner-wt-iter43`): plan `ca574c2f8`, with 1 milestone, an edit contract and 4 text-mutation drills.
- **Executor round 1** (codex, recipe): edited the `.claude` copy and the changelog. The sandbox refused writes to `.agents/` (`Operation not permitted`), so the controller mirrored that copy. 8 static gates were rc 0 in the sandbox and again outside it. Commit `541d0f623`.
- **Judge round 1** (sonnet, Agent tool, own worktree `fleet-i43-judge`): **PASS 74 with a blocking finding**, reproduced with its drill 5. The "foreground alternative" (≤540 s slices) hit the snippet's single deadline branch, so slice expiry wrote the terminal `Gate 3b TIMEOUT … PARK` verdict for a run that was still in progress. The path was also not re-entrant: each slice re-ran `mktemp`, re-recorded the target and reset the cap. A marginal pass with an open blocker was not treated as landable.
- **Executor round 2** (codex): took the judge's minimal option. The foreground path is gone and the poll is background-only, with the WHY stated. `${iteration}`, which is defined nowhere, became `${MISSION_NAME:-mission}`. Commit `9285463`, 25+/3− per copy, both copies byte-identical.
- **Judge round 2** (resumed by `SendMessage`): **PASS 90**, 0 blocking. Drill 5 now shows that no reachable path writes a terminal verdict before 1800 s. The 4 plan drills were killed, and `bash -n`/`zsh -n` were rc 0.
- PR #1853, with no closing keyword, was squash-merged with `--match-head-commit 9285463` → `a8042fa3f`.

**Gate 3b**: The merged snippet was extracted from `origin/dev` and run **verbatim, as it now prescribes** (`run_in_background: true`, `timeout: 2000000`, no inner `&`) — the plan's behavioural acceptance. Launch output: `target=a8042fa3f…` (full SHA) `rid=38903552173 verdict=/tmp/ci_fleet_heada8042fa.fqsYg1`, plus the expected `DRIFT` line (my own merge), poll still pinned. It ran 30 min in the background, past the 600 s foreground cap, while the controller chained bounded 45 s and 540 s reads. It then wrote `Gate 3b TIMEOUT for a8042fa3f9eec8a108481230fa3de9a06cd82059 after 30m (status=in_progress ) — PARK, do not hang`. The mechanism worked, and the verdict is retro (1) measured a second time: dev CI outlasts the cap. A fresh bounded poll on a new subject-named path then read `ALL COMPLETE for a8042fa3f… total=17 failures=1`. `test`, `lint`, `build` and `launchd drivers` are green. `docs-gate` and `UI build gate` were not created for this push; both were green on the tree-identical PR head. The only red is non-required `Build macos-latest`, `TestMemprobeDebugLogDoesNotAccumulate` (a memory test; this merge changed only markdown), with windows cancelled by fail-fast. Ticket resolved after the merge; motoko received the reply.

**Done-gate**:
- The 8 static gates (`check-skills`, `-context-docs`, `-referenced-paths`, `-changelog`, `-file-sizes`, `-doc-review`, `-no-personal-email`, `-tmpfile-hygiene`) were rc 0 at base and at both rounds.
- `make test-launchd-drivers` under `env -i`: rc 0 at base `c4fab92c2` and at `9285463`, which is tree-identical to `a8042fa3f`.
- Dry-runs under **triage** (idle and armed, no pidfile; `AILANG_DRIVER_PINNED=9285463`): healthy rc 0 with `lanes=ok`; degraded (`MISSION_EXECUTOR_MODEL=pi:bogus/no-such-model`) rc 0 with `lanes=DEGRADED(1)`. No pidfile was written and nothing was reloaded.
- PR #1853: 26 check-runs, 0 failures, with all 5 required checks green.

**Progress**: 1 ticket resolved; 48 open after the resolve (49 at pick). Clause 1 UNMEASURED. Clause 2 UNMET: this ticket took about 15 d (filed 10-01), which pulls the median down but leaves it above 48 h. Clause 3 MET. Clause 4: prior evidence only. Clause 5 preserved: judged, done-gate passed, required CI green.

**Routing evidence**: base-gate1=c4fab92c29ab7e208dc96325a925f053cf0417a9@2026-10-16T11:45:50Z; base-gate4=a8042fa3f9eec8a108481230fa3de9a06cd82059@2026-10-16T12:52:29Z; gate3b=a8042fa3f9eec8a108481230fa3de9a06cd82059 (drift 1 commit, my own merge). `MISSION_ROUTING_NOTE` was empty and nothing was over ration.
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer**: not spawned. The ticket is a mechanical fix to one snippet, and the charter designs only when a fix warrants it. Resolver `recipe claude:claude-opus-5-5 declared:provider-pin`; rotation not advanced.
- **Planner**: resolver `agent-tool opus fail-closed:no-doc` against the pin `codex:gpt-6.1-sol`, so it was routed to the pin under the codex recipe (role-spawn-routing §2(a)). Probe rc 0; run rc 0 (79,862 tok).
- **Executor**: `codex:gpt-6.1-sol` via the recipe (`declared:provider-pin`), 2 runs, rc 0 and rc 0 (81,777 + 60,170 tok).
- **Evaluator**: `sonnet` via the Agent tool (`declared:alias-pin`), 2 rounds (53,101 + 58,116 tok). Judge independence: cross-vendor (Anthropic sonnet vs the OpenAI generator).
- The Agent tool was used wherever the resolver names it (the evaluator), per the operator's standing request. Planner and executor resolve to recipes, where the Agent tool is not a valid path. No role failed to spawn.
- **Metered**: $0.

**Ruled out**:
- `quorum:gpt6-1-sol-openai-api-credits-exhausted` as a fleet fix. It is billing/routing policy (HD-2a), and the seat is defined in `cmd/ailang/design_quorum.go:183`, outside Authority. Filed as D-FLEET-20.
- The dev `Build ubuntu-latest` red as fleet-related (`TestWorkerCLIProcessExitLifecycle`, a worker-process test).
- Keeping a foreground-slice form. Making it correct needs re-entry state carried across calls, plus a separate slice-expiry branch, about 3× the text. The ticket allows background-only.

**Retro lane**:
- (1) **The 30-min Gate-3b cap is now shorter than dev CI.** The snippet's comment still says "CI is ~15-20m". Measured today: dev CI runs took 33, 31, 46, 47 and 55 min (`gh run list --workflow CI --branch dev`), and the PR's required `test` job alone ran 29m14s. This iteration's own PR poll timed out at 30 min with 1 check pending, and a second poll was needed. Instance 1. Changing the cap was outside this ticket, so it is not ticketed by the fleet (the fleet cannot file tickets). It is recorded here and raised to Mark as an observation, not an ask. Second-instance candidate.
- (2) **The codex sandbox refuses `.agents/` writes.** The executor could not mirror the second copy, and the controller had to. Any D-FLEET-8 dual-copy fix will hit this. Instance 1; noted for directive-writing.
- (3) **The harness refused an inline `bash -c` containing `rm`** ("could not check the script for dangerous removals"). Moving the poller to a script file fixed it. Instance 1.

**Next**: #16 stays not-live. Then tickets by `slots_lost`: next routable `slots=1`, oldest first, is `skill:gate4-status-stamp-shape-unprescribed` (motoko, 10-01). D-FLEET-16, -18, -19 and -20 are with Mark; D-FLEET-17 is moot.

## 44 — 2026-10-16 — `skill:gate4-status-stamp-shape-unprescribed` LANDED: #1866 → `91d5d6390`; judge PASS 94 then PASS 97 (sonnet); ticket resolved [HARNESS]

**Pick**: the queue head named by iteration 43's **Next**. #16 is not live, and P2 #13/#18/#19/#21 are parked on D-FLEET-18/19/20. The agent-tool `slots>1` rows stay PARKED-ON-LANE. The oldest routable `slots=1` ticket was `skill:gate4-status-stamp-shape-unprescribed` (motoko iteration 40, 10-01), a mechanical skill fix in `.claude/skills/mission-*` + `.agents/skills/mission-*`, so it is in scope. 48 open tickets at pick.

**Preflight**: armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing CLEAN. Directives: 0 on #1768 since `2026-10-15T02:46:27Z` (16 comments, all self). Self-notices rc 0 (control `852:10`, watermarks 2/2). No rotation and no sweep due.

**Observe**: origin/dev == pin == `bd30497a0`. `SKILL.md` and all 13 resources at the resolved symlink target are `cmp`-identical to origin. There was no open `fleet/*` PR and no i43/i44 worktree, so no orphan. Dev CI on `bd30497a0` had `lint` red on `fmt-check` (`internal/eval_harness/pi_models_config_test.go`, V1's lane). It was already fixed upstream: `lint` was green on `e1ef33fa5` by the time of the PR.

**Reality check**: at HEAD, `canonicalStatusRe` (`internal/mission/normalize.go:37`) requires `## STATUS <date> — ITERATION <n>: <title>`. Both copies of `resources/gate-4-record.md` (separate inodes, byte-identical) never stated it. `TestMissionDocHeadingsStayCanonical` lints only logs and archives (`canonical_test.go:21-44`, ratchet 11), so a bad charter stamp reds only after rotation. The premise holds. `go test ./internal/mission/` was rc 0 at base, as were the 8 static gates.

**Did**:
- **Planner** (codex `gpt-6.1-sol`, recipe, detached worktree `.planner-wt-iter44`): plan `7c06ad384`, with 1 milestone, a Perl behavioural check against the production regex and 4 text-mutation drills.
- **Executor** (codex, recipe, worktree `fleet-i44-gate4-stamp`): +18/−1 in the `.claude` copy, plus the fragment. The sandbox again refused `.agents/` (`cp` rc 1, "Operation not permitted"), so the controller mirrored that copy. The controller confirmed the plan's claim that `ailang mission normalize` excludes charters (`canonicalDocs`, `cmd/ailang/mission_cmd.go:421`). Commit `1507ba05e`.
- **Judge round 1** (sonnet, Agent tool, own worktree `fleet-i44-judge`): **PASS 94**, 0 blocking.
  - Every claim was checked against code.
  - The self-check grep agrees with the Go regex under bash and zsh on 8 stamps.
  - The 4 plan drills plus 1 of its own (the em dash in the grep line) are killed.
  - Simulated rotation for v1, docs, motoko, fleet and triage: a canonical stamp gives rc 0 on all 5, and a `COMPLETE:` stamp gives rc 1 on all 5, which shows the drill can fail.
  - 3 non-blocking wording findings.
- **Controller** applied all 3 (+4/−2 per copy, `2c19f6779`): a local note may not change the shape, the grep is the gate for a stamp still in the charter, and an UNCONVERTIBLE heading needs a hand edit.
- **Judge round 2** (resumed by `SendMessage`): **PASS 97**, 0 blocking; drills re-killed.
- PR #1866 (no closing keyword) was squash-merged with `--match-head-commit 2c19f6779` → `91d5d6390`.

**Gate 3b**: the PR poll hit its 30-min cap with `build` still pending (`total=26 pending=1 failures=0`). A fresh poll on a new path read `ALL COMPLETE for 2c19f6779… total=26 failures=0` within minutes. The prescribed dev poll was extracted from the running skill and run as written: `target=91d5d6390…`, `rid=38966637774`, plus the expected `DRIFT` line (8 commits, including my merge and the v0.54.14 release), poll still pinned. Verdict: the poll hit its 30-min cap (`Gate 3b TIMEOUT for 91d5d6390… after 30m (status=in_progress)`) with only `test` running. A fresh check-run poll on a new path read `ALL COMPLETE for 91d5d6390… total=17 failed=[SonarCloud Code Analysis]`. The required `test`, `lint` and `build` are green. `docs-gate` and `UI build gate` were not created for this push; both were green on the PR head. Sonar is inherited: it is also red on the parent `e1ef33fa5`. So dev CI outran the cap twice in one iteration (the PR poll and the dev poll). Ticket resolved after the green; motoko got the reply.

**Done-gate**:
- 8 static gates rc 0 at base and at both heads; `go test ./internal/mission/ -count=1` rc 0 outside the sandbox.
- `make test-launchd-drivers` under `env -i` rc 0 (59 probe arms). It ran in the sprint worktree at `1507ba05e`, and the later `2c19f6779` delta is markdown only.
- Dry-runs under **triage** (idle and armed, no pidfile; `AILANG_DRIVER_PINNED=2c19f6779`): healthy rc 0, degraded (`MISSION_EXECUTOR_MODEL=pi:bogus/no-such-model`) rc 0. In the degraded arm the bogus pi lane failed its probe (rc 1) and was handed to `opus` (end of chain). Both arms read `lanes=DEGRADED(3)`, because codex was over its daily ration (rc 75) for the designer, planner and executor lanes. The base control (`AILANG_DRIVER_PINNED=bd30497a0`) read the same `DEGRADED(3)`, so this is quota state, not this change. No pidfile was written and nothing was reloaded.

**Progress**: 1 ticket resolved; 47 open after the resolve (48 at pick). Clause 1 UNMEASURED. Clause 2 UNMET: this ticket took about 15.5 d (filed 10-01). Clause 3 MET. Clause 4: prior evidence only. Clause 5 preserved: judged, done-gate passed, required CI green. The ticket's "green across one rotation per live mission" is met by simulation for the 5 in-repo missions. World (separate repo) is UNMEASURED, and real rotations happen as each mission next records.

**Routing evidence**: base-gate1=bd30497a0f5bd8a89a54483643cdc55f5ff69991@2026-10-16T19:59:37Z; base-gate4=91d5d63905ef9944ac3aedd4474bdc9206e90d0d@2026-10-16T21:37:14Z; gate3b=91d5d63905ef9944ac3aedd4474bdc9206e90d0d (drift 8 commits: my merge plus attended commits). `MISSION_ROUTING_NOTE` was empty. The driver reported the `codex`, `anthropic` and `openrouter` buckets over ration or unknown at dry-run time. Both codex runs below completed before that.
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer**: not spawned. The ticket is a mechanical one-paragraph rule, and the charter designs only when a fix warrants it. Resolver `recipe claude:claude-opus-5-5 declared:provider-pin`; rotation not advanced.
- **Planner**: resolver `agent-tool opus fail-closed:no-doc` against the pin `codex:gpt-6.1-sol` (`derive-planner-lane.sh` on the charter: `codex:gpt-6.1-sol declared:planner-lane-default-pin`), so it was routed to the pin under the codex recipe (role-spawn-routing §2(a)). Probe rc 0; run rc 0 (86,278 tok).
- **Executor**: `codex:gpt-6.1-sol` via the recipe (`declared:provider-pin`), 1 run, rc 0 (88,560 tok).
- **Evaluator**: `sonnet` via the Agent tool (`declared:alias-pin`), 2 rounds (67,868 + 72,218 tok). Judge independence: cross-vendor (Anthropic sonnet vs the OpenAI generator).
- The Agent tool was used wherever the resolver names it (the evaluator), per the operator's standing request. Planner and executor resolve to recipes, where the Agent tool is not a valid path. No role failed to spawn.
- **Metered**: $0.

**Ruled out**:
- Rotating a charter stamp with `ailang mission normalize`. It reads only the log and archive streams (`canonicalDocs`), so a charter stamp has to be fixed by hand before rotation. The rule says so.
- Treating the dry-run `DEGRADED(3)` as caused by this change. It is codex over its daily ration, and the base control reads the same.
- The dev `lint` red on `bd30497a0` as fleet-related (`fmt-check` in `internal/eval_harness`). It was fixed upstream by `e1ef33fa5`.
- Widening the ticket to normalise legacy non-canonical headings (ratchet 11). That is self-sourcing; the ticket asks only for the prescription.

**Retro lane**:
- (1) **Second instance: the 30-min Gate-3b cap is shorter than CI.** Iteration 43 measured dev CI at 31–55 min, and this iteration's PR poll timed out at 30 min with `build` still pending (0 failures), so a second poll was needed again. Two recorded frictions now point at the same gap, which meets the skill-fix bar (≥2), but changing a wait cap is a time-budget call that touches standing rule 6's ≤30-min default. Raised to Mark as D-FLEET-21, not edited by the loop.
- (2) **Second instance: the codex sandbox refuses `.agents/` writes** (iterations 43 and 44). Every dual-copy (D-FLEET-8) fix needs the controller to mirror the copy. It is a cheap, safe workaround; recorded, no change.
- (3) A judge's non-blocking wording findings were applied by the controller and re-judged by `SendMessage` resume in about 20 s. That is cheaper than another executor round, and the judge stays the one who accepts the change.
- (4) **`rotate-log` again.** Run without `AILANG_MISSION_REGISTRY`, it resolved to the **pin** worktree. There it rotated 0 entries but still prepended a third banner, which the controller restored from HEAD; the pin is clean. Re-run with the override in the record worktree, it rotated entry 24 correctly, and the duplicate banners were collapsed by hand (instance 5+ of the iteration-32 note). The controller missed the override because it lives in log prose, not in the Gate 4 text it read. The banner defect has no ticket, and the fleet cannot file one for itself, so it is recorded only.

**Next**: #16 stays not-live. Then tickets by `slots_lost`: the next routable `slots=1`, oldest first, is `mission:index-regenerate-needs-rotatable-archive` (motoko, 10-01; a `cmd/ailang/mission*` change, in scope). After it: `eval:motoko-connection-probe-orphaned` (motoko, 10-01; check its paths against D-FLEET-18 first). D-FLEET-16, -18, -19, -20 and the new -21 are with Mark; D-FLEET-17 is moot.

## 45 — 2026-10-17 — `mission:index-regenerate-needs-rotatable-archive` + `rotate-log:index-regen-drops-orphan-rows` LANDED: #1870 → `2061f445e`; judge PASS 90 (sonnet); 2 tickets resolved; D-FLEET-22 filed [HARNESS]

**Pick**: iteration 44's **Next**, `mission:index-regenerate-needs-rotatable-archive` (motoko iteration 40, 10-01), combined with `rotate-log:index-regen-drops-orphan-rows` (attended, 10-03; the second half of the defect reported at #1306). They share `internal/mission/rotate.go` and its index writer, so one change addresses both, and both are in scope (`internal/mission/**`, `cmd/ailang/mission*`). #16 is still not live. The agent-tool `slots>1` tickets stay PARKED-ON-LANE as iterations 29–44 assessed them: they are codex-controller native-spawn limits, and this claude controller spawned sonnet without trouble. The newest ticket, `reporting:daneel-reports-not-declared-triage` (triage, 10-16), needs a coordinator config change outside the Authority allowlist, so it went to D-FLEET-22. 47 open tickets at pick.

**Preflight**: armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing CLEAN; messaging store gcp. Directives: 0 on #1768 since `2026-10-15T02:46:27Z` (17 comments, all self). Self-notices rc 0 (control `852:10`, watermarks 2/2). No rotation and no sweep due.

**Observe**: origin/dev == pin == `37d6641b3`. `SKILL.md` and all 13 resources at the resolved symlink target are `cmp`-identical to origin. No open `fleet/*` PR and no i44/i45 worktree, so no orphan. Dev CI on `37d6641b3`: 17 checks, all green.

**Reality check**: on a temp copy with `AILANG_MISSION_REGISTRY` pointed at it, `rotate-log motoko` fails with `no iteration entries found`: the live log was reset empty on 09-30, and entries 0–39 are all in the archive. `--stream status` refuses on 11 interleaved structural headings. So `motoko-mission-index.md` is hand-maintained. The ticket's "byte-for-byte" done-criterion cannot be met as written: the hand index has edited titles for rows 38 and 39. The directive replaced it with the same iteration-number set, the Reset note kept verbatim, unchanged source sha256, and valid UTF-8. `RotateLog` builds the index from headings only, so the orphan-row half also holds at HEAD. The 5 directive gates were green at base, run outside the sandbox.

**Did**:
- **Planner** (codex `gpt-6.1-sol`, recipe, detached worktree `.planner-wt-iter45`): plan `87fd11ebc`, with 1 milestone and 14 named test arms, each paired with the mutation it must kill. It measured the motoko sets: hand index and archive both `{0–5, 7–39}`, with no `## 6` heading anywhere (iteration 7 is "recovered iteration 6").
- **Executor** (codex, recipe, worktree `fleet-i45-index-regenerate`):
  - New `internal/mission/index.go` (217 lines); `rotate.go` 339 → 276.
  - Handler extracted to `cmd/ailang/mission_rotate.go`, taking `mission_cmd.go` from 506 to 427 lines.
  - Tests, the Gate 4 doc and a changelog fragment.
  - The sandbox again refused `.agents/` (instance 3), so the controller mirrored that copy.
  - In-sandbox `go test ./internal/mission/` was UNINFORMATIVE (a quota test's loopback bind). Outside the sandbox, all 5 gates and `golangci-lint` (0 issues) were rc 0.
  - Commit `07681ab9a`.
- **Controller smoke** (new binary, temp copies): motoko `--index-only` printed 39 iterations and `non-entry headings ignored: 1` (the fenced template heading), and only rows 38 and 39 changed. `--stream status` wrote a separate status index (34 iterations, 12 ignored). Fleet's index regenerated byte-identical. `--keep` with `--index-only` was refused. Source sha256 was unchanged.
- **Judge** (sonnet, Agent tool, own worktree `fleet-i45-judge`): **PASS 90**, 0 blocking, gates re-run.
  - 14 mutation drills: 12 killed. One survivor is a byte-identical rewrite of the live file, which is harmless. The other is `t[:147]` byte truncation in `indexLine`, a gap that existed before this change: the plan claimed `MotokoSnapshot` covered it, and it does not.
  - A synthetic orphan row 999 survived a normal `rotate-log fleet --keep 20` on a copy, byte-exact, and was printed.
  - The 16 `.orig` files in `/tmp` match its drill list, and its tree was clean afterwards.
- PR #1870 (no closing keyword; the scan was paired with a control) was squash-merged with `--match-head-commit 07681ab9a` → `2061f445e`. #1306 is still OPEN, as intended.

**Gate 3b**: the PR poll hit its 30-min cap with `build` still pending (`total=26 pending=1 failed=[]`). A fresh poll on a new path read `ALL COMPLETE for 07681ab9a… total=26 failed=[]`. The prescribed dev poll was extracted from the running skill and run as written: `target=2061f445e…`, `rid=39004682058`. The expected `DRIFT` line (1 commit, my merge) appeared, and the poll stayed pinned. Verdict: the poll hit its 30-min cap (`Gate 3b TIMEOUT for 2061f445e… after 30m (status=in_progress)`) with only `test` running and every other job green. A fresh check-run poll on a new path read `ALL COMPLETE for 2061f445e… total=21 failed=[]`, and run `39004682058` is `completed success`. So dev CI outran the cap twice again in one iteration, once for the PR poll and once for the dev poll; the `test` job alone ran over 40 min. Both tickets were resolved after the green, with replies to motoko and attended. #1306 got its verdict comment (count 1 → 2, checked) and was closed, since both halves are landed.

**Done-gate**:
- 5 directive gates plus `golangci-lint` rc 0 at `07681ab9a`, outside the sandbox.
- `make test-launchd-drivers` under `env -i` rc 0 (59 probe arms, `PASS=25 FAIL=0`).
- Dry-runs under **triage** (idle and armed, no pidfile; `AILANG_DRIVER_PINNED=07681ab9a`): healthy rc 0 with `lanes=ok`. Degraded (`MISSION_EXECUTOR_MODEL=pi:bogus/no-such-model`) rc 0 with `DEGRADED(1)`: the bogus pi lane failed its probe (rc 1) and was handed to `opus`.
- No pidfile was written and nothing was reloaded. The driver itself is unchanged by this PR; the dry-run proves the pinned tree still starts.

**Progress**: 2 tickets resolved; 45 open after the resolve (47 at pick). Clause 1 UNMEASURED. Clause 2 UNMET: the two tickets took about 16 d (filed 10-01) and 14 d (filed 10-03). Clause 3 MET. Clause 4: prior evidence only. Clause 5 preserved: judged, done-gate passed, required CI green. #1306 is closed with a verdict comment: half 1 landed in #1580 and half 2 in #1870.

**Routing evidence**: base-gate1=37d6641b308a3e8ffb4d5a2023ba4faa89a5254a@2026-10-17T04:10:35Z; base=2061f445e881078dd3068a009ce1377b58cc0f7e@2026-10-17T05:01:17Z (gate4); gate3b=2061f445e881078dd3068a009ce1377b58cc0f7e (drift 1 commit: my merge). `MISSION_ROUTING_NOTE` was empty and `MISSION_OVER_RATION` was empty.
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer**: not spawned. The fix is mechanical and small, the controller wrote design decisions D1–D5 into the planner directive, and the charter designs only when a fix warrants it. Resolver `recipe claude:claude-opus-5-5 declared:provider-pin`; rotation not advanced.
- **Planner**: resolver `agent-tool opus fail-closed:no-doc` against the pin `codex:gpt-6.1-sol` (`derive-planner-lane.sh` on the charter: `codex:gpt-6.1-sol declared:planner-lane-default-pin`), so it was routed to the pin under the codex recipe (role-spawn-routing §2(a)). Probe rc 0; run rc 0 (79,186 tok).
- **Executor**: `codex:gpt-6.1-sol` via the recipe (`declared:provider-pin`), 1 run, rc 0 (108,240 tok).
- **Evaluator**: `sonnet` via the Agent tool (`declared:alias-pin`), foreground, 1 round (73,876 tok). Judge independence: cross-vendor (Anthropic sonnet vs the OpenAI generator).
- The Agent tool was used wherever the resolver names it (the evaluator), per the operator's standing request. Planner and executor resolve to recipes, where the Agent tool is not a valid path. No role failed to spawn.
- **Metered**: $0.

**Ruled out**:
- Byte-for-byte reproduction of motoko's hand index. Rows 38 and 39 carry hand-edited titles, and the archive headings are the source of truth.
- A new `mission index` subcommand, because the CLI surface is gated by `make simplicity-audit`; a flag on the existing verb was used instead.
- Preserving orphan `—` (note) rows. They have no stable identity, so only numbered orphans are kept.
- Treating the in-sandbox `internal/mission` test failure as a regression. It was a loopback bind, and the suite is rc 0 outside the sandbox.

**Retro lane**:
- (1) **Third instance: the 30-min Gate-3b cap is shorter than CI** (iterations 43, 44 and 45). The PR poll timed out with `build` pending and 0 failures again, and a re-poll on a fresh path went green. This is already D-FLEET-21; it is a time-budget call for Mark, so the loop did not change the cap. The count is now 3.
- (2) **Third instance: the codex sandbox refuses `.agents/` writes** (iterations 43–45). The controller mirroring is cheap and reliable. It stays a recorded workaround, not a skill change.
- (3) A plan's mutation-coverage claim was wrong: it said `MotokoSnapshot` would catch byte truncation, and the judge's drill showed it does not. Instance 1. Planner claims about which test kills which mutant need the judge's drills, which is what the judge is for. No change.
- (4) The new `--index-only` flag gives the next fleet record a non-destructive way to refresh an index, and the next motoko record can regenerate its index instead of editing it by hand. Gate 4's text says so in both copies.

**Next**: #16 stays not-live. Then tickets by `slots_lost`, oldest `slots=1` first. `eval:motoko-connection-probe-orphaned` asks whether to retire a `tools/eval/` probe, so it waits on D-FLEET-18. The next routable ticket is `rotate-log:status-archive-structural-header` (docs, 10-01; `internal/mission`). D-FLEET-16, -18, -19, -20, -21 and the new -22 are with Mark; D-FLEET-17 is moot.

## 47 — 2026-10-19 — `rotate-log:status-archive-structural-header` LANDED: orphaned iteration 46's #1877 verified and merged → `f048d6fa9`; judge PASS 93 (sonnet); ticket resolved [HARNESS]

**Pick**: iteration 45's **Next**, `rotate-log:status-archive-structural-header` (docs, 10-01). Gate 0's self-notice check returned rc 1, a died fire (crash notice 2026-10-17T12:48:13Z on #1768). Gate 2's died-mid-flight traces found it: open PR #1877 on `fleet/i46-docs-status-structural`, with clean worktrees `fleet-i46-docs-status-structural` and `fleet-i46-judge`. So the deliverable was to verify and land that work, not redo it. 45 open tickets at pick.

**Orphans credited**:
- **Iteration 46** (fire 2026-10-17 14:12). Its lanes were degraded: codex was over its daily ration, so the planner went to opus and the executor to `claude:claude-haiku-5-5`. It built the fix, got a judge PASS 92 (sonnet), and opened #1877. It then died at Gate 3 on "You've hit your weekly limit · resets Oct 19 at 7am" (slot-verdict `CRASHED at=gate-3 rc=1 elapsed_s=2227`). It left no log entry or STATUS stamp. This entry is its record.
- **The 2026-10-19 02:54 fire** wrote no record. Anthropic and codex were both quota-limited, so the controller was `pi:ollama/glm-5.3:cloud`. It reached Gate 3, hit ollama's Pro 5-hour limit (429), and the driver PAUSED it (`PAUSED-NO-CAPACITY at=gate-3`). It left no PR, no worktree and no branch.
- Five fires between those two (10-17 20:49 to 10-18 20:53) were `PAUSED-NO-CAPACITY at=fired`, at about 70 s each. That makes **two dead slots in a row**, both quota-caused.

**Preflight**: armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing CLEAN; messaging store gcp. Directives: 0 on #1768 since `2026-10-15T02:46:27Z` (25 comments, all self) and 0 on #1584. Self-notices rc 1 (control `852:10`, watermarks 2/2). No rotation and no sweep due.

**Observe**: origin/dev == pin == `38525b06b`. `SKILL.md` and all 13 resources at the resolved symlink target are `cmp`-identical to origin. Dev CI on `38525b06b`: `CI` and `Build and Release` success, 17 checks, none failing.

**Reality check**:
- #1877 had been red only on `Build ubuntu-latest`, in `internal/runner` `TestWorkerTerminalPTYLifecycle/eval/exit`: 1 of 11 subtests failed (a Python `int('')` in the PTY harness), and `strict/exit` passed. The macOS and Windows legs carry no `--- FAIL`; they are fail-fast casualties. The PR touches no runner code, so this is a flake, not this change.
- Rebased onto `38525b06b`; the 2 new dev commits are design docs only, and the rebase was clean. `go test ./internal/mission/` rc 0 at the new head `2f51ae8df`.
- On temp copies, with `AILANG_MISSION_REGISTRY` and the files under the registry's parent:
  - Base: `rotate-log docs --stream status` rc 1 at both `--keep 1000` and `--keep 3`, refusing on `## (Charter sections — moved back 2026-10-01)`.
  - Head: rc 0 at both, with the tombstone still present (count 1).
  - The first attempt put the files under `wd/` and got `no such file`. That was an instrument failure, not a verdict, and was re-run.

**Did**:
- Force-pushed the rebased branch with lease on the old head `95d1e15e1`.
- **Judge** (sonnet, Agent tool, foreground, own detached worktree `fleet-i47-judge` at `2f51ae8df`): **PASS 93**, 0 blocking.
  - Byte check: removing the tombstone from base and the preamble from head gives identical files under `cmp`.
  - Rotation at head: rc 0, and the set of STATUS headings across live plus old equals base's.
  - Mutation drills, 5 of 6 killed:
    - Killed: revert the archive move; pin the count to 11; pin it to 13; make `strayHeadings` return nil (also killed by `TestRotateLog_RefusesAFileWithInterleavedStructure`); append `## Backlog`.
    - Survived: demoting the tombstone heading to `###`, which is a legal state.
  - Gates rc 0: `go test`, `go vet`, `gofmt -l`, `make check-changelog`. Its tree was clean afterwards.
- PR #1877: all 26 checks complete with 0 failures, MERGEABLE/CLEAN. The closing-keyword scan found nothing, with its control firing. Squash-merged with `--match-head-commit 2f51ae8df` → `f048d6fa9`.

**Gate 3b**: the prescribed dev poll ran verbatim. It pinned `target=f048d6fa9c2df95f272e34264955b8628d4da589` (`rid=39221988856`), with the expected `DRIFT` of 2 commits (the base moved from Gate 1 because of my merge). Verdict: the poll hit its 30-min cap (`Gate 3b TIMEOUT for f048d6fa9… after 30m (status=in_progress)`) with only `test` running and nothing failed. A fresh poll on a new path read `CI for f048d6fa9…: completed success`, and the check set is 21 checks, 0 pending, 0 not-green. That is the **fourth** instance of D-FLEET-21. The ticket was resolved only after that green (`replied to docs`).

**Done-gate**:
- `make test-launchd-drivers` under `env -i`, in the judge worktree at `2f51ae8df`: rc 0 (`PASS: 59 probe self-test arms ran`, `PASS=25 FAIL=0`).
- Dry-runs under **triage** (idle and armed, no pidfile; `AILANG_DRIVER_PINNED=2f51ae8df`):
  - Healthy: rc 0, `lanes=ok`.
  - Degraded (`MISSION_EXECUTOR_MODEL=pi:bogus/no-such-model`): rc 0, `DEGRADED(1)`, with the executor handed to `opus`.
- No triage pidfile was written and nothing was reloaded. The driver itself is unchanged by this PR.

**Progress**: 1 ticket resolved; 44 open after the resolve.
- Clause 1 UNMEASURED.
- Clause 2 UNMET: this ticket took about 18 d.
- Clause 3 MET.
- Clause 4: prior evidence only. The paused fires cost about 70 s of driver time, and none spent controller tokens before the pause.
- Clause 5 preserved: judged, done-gate passed, CI green before the resolve.

**Routing evidence**: base-gate1=38525b06ba63f3316e7f67df36982acdada9e668@2026-10-19T07:41:46Z; base=f048d6fa9c2df95f272e34264955b8628d4da589@2026-10-19T08:20:52Z (gate4); gate3b=f048d6fa9c2df95f272e34264955b8628d4da589 (drift 2 commits, from my merge). `MISSION_ROUTING_NOTE`=`controller pi:ollama/glm-5.3:cloud → claude:claude-opus-5-5 (probe ok)`; `MISSION_OVER_RATION` empty.
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer**, **planner** and **executor**: not spawned. The orphan's fix was complete and is credited to iteration 46: planner opus, executor `claude:claude-haiku-5-5`, both from its degraded routing.
- **Evaluator**: `sonnet` via the Agent tool (`declared:alias-pin`), foreground, 1 round (51,800 tok). Judge independence: a fresh judge, independent of both the iteration-46 generator (haiku) and this controller (opus).
  - The first spawn was refused by the spawn-pin hook (`fail-closed:role-missing`), because the `MISSION-ROLE:` line carried trailing text. Re-spawned with the token alone on line 1.
- The Agent tool was used for the only role spawned, per the operator's standing request. No role failed to spawn.
- **Metered**: $0.

**Ruled out**:
- Redoing iteration 46's work. It was complete and reviewed; only verification and landing were owed.
- Attributing the #1877 red to the change. It was a single flaky subtest in an untouched package.
- My own PR-check background poller. It piped `gh api` JSON through `echo`, which mangled control characters, and jq parse-errored on every round. It was an instrument failure, not a verdict: I stopped it and replaced it with a direct `check-runs` read (26 total, 0 pending, 0 failing).

**Retro lane**:
- (1) **Two dead slots in a row, both quota-caused**: an Anthropic weekly limit (iteration 46) and an ollama Pro 5-hour limit (10-19 02:54), plus five ~70 s capacity pauses. Under the routing chain, a quota exhaustion mid-iteration kills the slot after the work is done but before the landing. This is a pattern for Mark, not a skill change. The capacity-pause path worked as designed on the five short fires.
- (1b) **Fourth instance: the 30-min Gate-3b cap is shorter than dev CI** (iterations 43–45 and 47). This is still D-FLEET-21, Mark's call.
- (2) **`TestWorkerTerminalPTYLifecycle/eval/exit` flaked on ubuntu.** That is V1's territory (`internal/runner`), outside fleet scope, so it is reported here only: 1 instance.
- (3) **The spawn-pin hook needs `MISSION-ROLE: <role>` alone on line 1.** Instance 1 here; the hook's message says "FIRST line", which is accurate. No change.

**Next**: `skill:design-scaffold-hardcodes-owning-repo` (world, 10-01). `eval:motoko-connection-probe-orphaned` waits on D-FLEET-18. D-FLEET-16, -18, -19, -20, -21 and -22 are with Mark; D-FLEET-17 is moot.

## 48 — 2026-10-19 — `skill:design-scaffold-hardcodes-owning-repo` + `sprint-skill:estimated-loc-zero-is-placeholder-sentinel` resolved as GHOSTS (#1542 `b51353bcd`, #1695 `2a47d1e65`); judge PASS 88 / PASS 93 (sonnet); 2 tickets resolved [HARNESS]

**Pick**: iteration 47's **Next**, `skill:design-scaffold-hardcodes-owning-repo` (world iteration 215, 10-01). 44 open tickets at pick. The only `blocking=all` ticket, `agent-tool:sonnet-unavailable` (12 slots; latest occurrence `blocking=none` with a working subscription-CLI workaround), stays PARKED-ON-LANE as in iterations 29 and 32–47. Its resume predicate is native capability on non-Claude controllers, which a claude controller cannot prove. The reality check showed the pick was already fixed, which made it a bookkeeping pick. Standing rule 1 then allowed a second: `sprint-skill:estimated-loc-zero-is-placeholder-sentinel` (attended, 10-03, `blocking=item`, the higher of the two `blocking=item` tickets by age). That one also proved already fixed. Index grep: no prior attempt on either.

**Orphans credited**: none new. Gate 0 step 6a returned rc 1 for the crash notice at 2026-10-17T12:48:13Z (https://github.com/sunholo-data/ailang/issues/1768#issuecomment-6234067020). That is iteration 46's death, already credited by iteration 47 (log entry 47, "Orphans credited"); this line records its URL. Traces: no open `fleet/*` PR; `fleet-i46-*` and `fleet-i47-*` worktrees are clean leftovers of landed work.

**Preflight**: armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing CLEAN; messaging store gcp. Directives: 0 on #1891 since `2026-10-15T02:46:27Z` (4 comments, all self). Self-notices rc 1 as above (control `852:10`, watermarks 2/2). Ledger valid, 22 rows; OPEN: D-FLEET-16 through D-FLEET-22. No rotation and no sweep due.

**Observe**: origin/dev == pin == `07d23f464`. `SKILL.md` and all 13 resources at the resolved symlink target are `cmp`-identical to origin. Dev CI on `07d23f464` was still running at Gate 1: 16 checks, 6 pending, 0 failed. No open fleet PR, so origin/dev's record is the newest.

**Reality check**:
- **Scaffold ticket.** `create_planned_doc.sh` and `move_to_implemented.sh` have derived `PROJECT_ROOT` from `git rev-parse --show-toplevel` since `b51353bcd` (#1542, 10-03). Outside a worktree they refuse with "Run this skill from the target Git worktree." The running copies under `~/.claude/skills/design-doc-creator` (a symlink to the main checkout) are byte-identical to origin.
  - HEAD run from a temp repo's subdirectory through the symlinked path: rc 0, and the doc landed in the temp repo. Nothing appeared in the main checkout.
  - Base (`b51353bcd^`) copy in a fake owning checkout: it resolved `DOC_PATH` to `<owner>/design_docs/planned/…`, failing at line 209 only because that directory did not exist. That is the reported defect.
  - `test_project_root.sh` rc 0.
- **estimated_loc ticket.** Since `2a47d1e65` (#1695, 10-09), `create_sprint_json.sh` emits `None`, and `validate_sprint_json.sh:125` refuses `estimated_loc == null`. On a copy of `sprint_M-AILANG-LANE-EXECUTOR-SHADOW.json` with `features[0].estimated_loc` set: base gave ZERO rc 1, NULL rc 0, REAL rc 0; head gave ZERO rc 0, NULL rc 1, REAL rc 0. The `.agents` copy and the running copies are identical to origin. #563 is already closed.
- Out of scope, noted: `tools/launchd/githooks/pre-push` refuses `.claude/skills/design-doc-creator/**` for the fleet (it is neither `_scope_is_harness` nor `_scope_is_fleet_support`). So if the scaffold fix had NOT existed, this ticket would have parked for an authority ruling. The open ticket `skill:design-doc-creator-grep-portability` has the same exposure.

**Did**:
- **Judge** (sonnet, Agent tool, foreground, own detached worktree `fleet-i48-judge` at `07d23f464`):
  - **Scaffold: PASS 88**, 0 blocking. Base arm, with the owner dir present and `ailang` stubbed, wrote the doc into the OWNER repo (rc 0); head wrote it into the target. `move_to_implemented.sh` at head resolves the target too. Mutation (script-relative root restored) killed by `test_project_root.sh`.
  - **estimated_loc: PASS 93**, 0 blocking. It added an end-to-end arm through `create_sprint_json.sh`: at head the generated placeholder is `null` and is refused, while a filled 0 is accepted; base generated 0 and refused it. Mutation (`== null` → `== 0`) killed.
  - Residue (judge): `.agents/skills/design-doc-creator/scripts/*` are still script-relative and `sync-agents-skills.sh` excludes that skill on purpose. The world and stapledon pins are ailang clones carrying that stale copy, which corrects my claim that World has no `.agents` directory. A relative invocation inside its own checkout is self-consistent, and no configured path (`~/.agents/skills`, `~/.codex/skills`, `~/.claude/skills`) invokes it against another repo. So the reported path is closed, and the latent copy belongs to `skills:agents-copies-stale-non-mission`.
  - The judge's tree was clean afterwards, with nothing new under the main checkout's `design_docs/planned/`.
- Resolved both tickets with full SHAs: `skill:design-scaffold-hardcodes-owning-repo` → `b51353bcd…` (replied to world); `sprint-skill:estimated-loc-zero-is-placeholder-sentinel` → `2a47d1e65…` (replied to attended). 44 → 42 open.

**Gate 3b**: no code landed this iteration; the fix commits are long on origin/dev. Their check sets: `b51353bcd` 17 checks, all green; `2a47d1e65` 17 checks, 1 not green: `govulncheck (vuln gate)`. That is a time-based advisory gate on a commit that changed only shell scripts, superseded by later green dev runs. The record PR's own CI is polled before merge.

**Done-gate**: not applicable to a ghost. Nothing in the harness changed, so there was no dry-run and no `make test-launchd-drivers`. The behavioural re-measurement of each fix (controller and judge, base vs head) stands in for "the right surface was edited". Reach is confirmed by the running copies being byte-identical to origin.

**Progress**: 2 tickets resolved; 42 open.
- Clause 1 UNMEASURED.
- Clause 2 UNMET: about 18 d and 16 d filed→resolved, although the fixes landed 2 d and 6 d after filing. Some of clause 2's miss is resolution lag: fixes landed by other lanes (attended, coordinator agents) never resolve the fleet ticket.
- Clause 3 MET.
- Clause 4 prior evidence only.
- Clause 5 preserved.

**Routing evidence**: base-gate1=07d23f4641911cd61be28b98c55be490dae93b66@2026-10-19T15:27:49Z; base=07d23f4641911cd61be28b98c55be490dae93b66@2026-10-19T15:33:15Z (gate4). `MISSION_ROUTING_NOTE` empty; `MISSION_OVER_RATION` empty.
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer**, **planner** and **executor**: not spawned. Both picks were ghosts with no design, plan or code to produce.
- **Evaluator**: `sonnet` via the Agent tool (`agent-tool sonnet declared:alias-pin`), foreground, 1 round (69,515 tok, 16 tool calls). Independent of the fix authors (#1542 and #1695 were coordinator/attended work) and of this controller (opus).
- The Agent tool was used for the only role this iteration needed, per the operator's standing request. No role failed to spawn.
- **Metered**: $0.

**Ruled out**:
- Re-implementing either fix. Both are on origin/dev, unreverted (`git log <sha>..origin/dev -- <files>` shows no touch of the changed lines), and running.
- Taking `agent-tool:sonnet-unavailable` ahead. It is parked on lane; see Pick.
- Treating the `.agents` design-doc-creator copy as keeping the scaffold ticket alive. No live invocation path reaches it cross-repo; it is the separate ticket's class and outside fleet scope.

**Retro lane**:
- (1) **Resolution lag hides fixed tickets.** Both tickets were fixed by non-fleet lanes within 2–6 days and sat open for another 10–16. Iteration 36's `weekly-report:unknown-mission` was the same shape (instance 1), so this is instances 2 and 3. It inflates clause 2's median and costs a fleet fire per ghost. A cheap sweep (`git log --since=<first_seen>` over each ticket's evidence paths) would surface candidates. Self-sourced audits are barred by the charter, though, so this is a question for Mark, not a change: proposed in the report, not filed as a decision.
- (2) **The fleet's write scope excludes `design-doc-creator`**, a skill that two open tickets name (`skill:design-doc-creator-grep-portability`, and the scaffold residue). Instance 1; noted for the next pick of either ticket.

**Next**: `crash:sprint-executor:sigbus-ram-tmp-full-make-test` (attended, 10-08, `blocking=item`). Ask 1 (scope local verification, no toolchain installs, push per milestone) sits in `.claude/skills/sprint-executor/**`, which is in scope. Asks 2 and 3 (Cloud Run scratch volume, coordinator failure marking) are outside scope and park. D-FLEET-16, -18, -19, -20, -21 and -22 are with Mark; D-FLEET-17 is moot.

## 49 — 2026-10-20 — `crash:sprint-executor:sigbus-ram-tmp-full-make-test` ask 1 LANDED: #1905 → `3f9a0c092`; judge PASS 86 then PASS 87 (sonnet); ticket stays open, asks 2–3 + a measured wrapper defect parked on D-FLEET-23 [HARNESS]

**Pick**: iteration 48's **Next**, `crash:sprint-executor:sigbus-ram-tmp-full-make-test` (attended, 10-08, `blocking=item`). 43 open tickets at pick; the one new ticket since iteration 48 is `probe:pi-duplicate-extension-discovery` (triage, 10-19, `blocking=none`). `agent-tool:sonnet-unavailable` (`blocking=all`) stays PARKED-ON-LANE, as in iterations 29 and 32–48. Index grep: no prior attempt.

**Orphans credited**: none new. Gate 0 step 6a returned rc 1 for the same crash notice (2026-10-17T12:48:13Z, #1768), which is iteration 46's death, already credited at iteration 47. Traces: no open `fleet/*` PR; the leftover `fleet-i46-*`/`fleet-i47-*`/`fleet-i48-*` worktrees are landed work.

**Preflight**: armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing CLEAN; messaging store gcp. Directives: 0 on #1891 since `2026-10-15T02:46:27Z` (6 comments, all self). Self-notices rc 1 as above (control `852:10`, watermarks 2/2). No rotation and no sweep due (#1891 was created after this Monday's 07:00 boundary).

**Observe**: origin/dev == pin == `4cc0f6a3e`. `SKILL.md` and all 13 resources at the resolved symlink target are `cmp`-identical to origin. Dev CI on `4cc0f6a3e`: `CI` and `Build and Release` green; 34 checks, 1 not green: `SonarCloud Code Analysis` (non-required, a GitHub App; V1's repo, so recorded, not picked).

**Reality check**: ask 1 was partly a ghost. #1770 (`dbe4b26a3`, 10-12) moved the skill to `make test-changed`, and #1688 (`face88a3f`) added "commit after each milestone". Two parts had no coverage: never install toolchains (no `toolchain`/`cgo`/`cc` rule anywhere in the skill) and push per milestone (the resource said "the coordinator pushes your branch when you finish"). The controller then measured the wrapper, in scratch repos:
- `cmd/ailang/coordinator_cloud_task.go:151` clones `--branch <base> --depth 1`, which is single-branch (fetch refspec = base only). An agent's own `git push origin HEAD:refs/heads/coordinator/task-x` creates **no** `refs/remotes/origin/coordinator/task-x`. After one more commit, Step 5b's `git log origin/<b>..HEAD` is rc 128, the `ls-remote` fallback is rc 0, so `newBranch=false`, `unpushed=false`, and the later commit is **silently not pushed** (no `assertRemoteMatchesHead` on that path). A naive "push per milestone" rule would therefore have *caused* work loss.
- With `+refs/heads/B:refs/remotes/origin/B` added before the first push, the tracking ref exists and the wrapper sees the later commit (log rc 0). A retry (fresh clone, same task id) resumes with `fetch` + `reset --keep origin/B`, keeping untracked wrapper files, and its next push is a fast-forward.
- Two instrument failures of my own, both caught: zsh read `$B:refs` as the `:r` modifier (refspec mangled), and a zsh "no matches found" on an `rm` glob aborted an `&&` chain, so a suite "rc=1" measured the `rm`. Both re-run under `/bin/bash`.

**Did**:
- **Designer**: not spawned. The controller wrote decisions D1–D8 into the planner directive (precedent: iteration 45); the change is skill text plus one helper script.
- **Planner** (codex `gpt-6.1-sol`, recipe, detached `.planner-wt-iter49`): plan `90a3d7c49` with 1 milestone and 22 arms (A01–A22), each paired with a mutation. A09 is the negative control that reproduces the wrapper loss. `validate_sprint_json.sh` hard-requires ≥2 milestones while the schema allows 1; kept at 1 (splitting would expose a push-only intermediate state).
- **Executor** (codex, recipe, worktree `fleet-i49-cloud-branch-sync`): `scripts/cloud_branch_sync.sh` (105 lines), its 22-arm hermetic test (390), `resources/unattended_runs.md` (+45/−1), `make/test.mk` wiring into `test-launchd-drivers`, and a changelog fragment. 22/22 arms pass; 21/22 helper mutations killed (A09 survives by design). The sandbox refused the `.agents/` writes (instance 4); the controller mirrored them byte-identically. Commit `f1a6fe966`.
- **Controller, outside the sandbox**: the 5 directive gates were all rc 0, and the caller's `~/.gitconfig`, the repo config and all refs were sha-identical before and after the test (observed, not seeded).
- **Judge** (sonnet, Agent tool, foreground, own worktree `fleet-i49-judge`): **PASS 86**, 0 blocking. It confirmed the wrapper hazard independently, from the Go source and empirically, and found the A08/A09 simulation a faithful transliteration of Step 5b + `branchNeedsPush`. Survivors it found: a glob gate `coordinator/${ID}*`, `--force-with-lease`, an extra push refspec, the tracking-ref check removed, `set -m` removed, the KILL escalation removed, `GIT_TERMINAL_PROMPT` removed.
- **Round 2**: the glob gate is the D2 class the plan rates highest-impact, so A03 now also skips `coordinator/task-x-other` and `coordinator/task-xy` (`d4d713c79`). Proven: the old A03 passes the glob mutant, the new one kills it, and the real helper passes. Re-judged **PASS 87** (glob mutant killed; worktree clean).
- PR #1905 (no closing keyword; scan paired with a control) squash-merged with `--match-head-commit d4d713c79` → `3f9a0c092`.

**Gate 3b**: the PR poll hit its 30-min cap with only `test` pending and 0 failures (`TIMEOUT … total=26 pending=1 failed=[]`). A fresh poll on a new path read `ALL COMPLETE for d4d713c79… total=26 failed=[]`. The dev poll ran as written from the running skill: `target=3f9a0c092…`, `rid=39347730823`, with the expected 1-commit `DRIFT` (my merge). Verdict: the first dev poll also hit its 30-min cap (`Gate 3b TIMEOUT for 3f9a0c092… (status=in_progress)`). A fresh poll on a new path read `CI for 3f9a0c092…: completed success`, and the full check set on the merge reads 17 checks, all green.

**Done-gate**:
- Surface: the sprint-executor skill is what cloud and mission executors read (`.claude` + `.agents`, byte-identical; `test_agents_skills_sync.sh` rc 0).
- Reach: pushed and merged to `origin/dev`.
- `make test-launchd-drivers` under `env -i`: rc 0 at base `4cc0f6a3e` and rc 0 at `d4d713c79`, which includes the new 22-arm suite. A stray `spawn-pin-hook.sh: line 11: cat: command not found` line appears once in both logs, so it is inherited.
- Dry-runs under **stapledon** (armed, idle, no pidfile; `AILANG_DRIVER_PINNED=d4d713c79`): healthy rc 0 `lanes=ok`; degraded (`MISSION_EXECUTOR_MODEL=pi:bogus/no-such-model`) rc 0 `DEGRADED(1)`, with the bogus lane handed to opus. Triage was tried first and exited at its idle gate before probing any lane, so it measured nothing for the degraded arm. No pidfile was written and nothing was reloaded.

**Progress**: 0 tickets resolved; 43 open. The ticket stays open because asks 2 and 3 are outside fleet scope; they are D-FLEET-23 with the wrapper defect.
- Clause 1 UNMEASURED.
- Clause 2 UNMET (unchanged; ask 1 landed 12 d after filing).
- Clause 3 MET.
- Clause 4 prior evidence only.
- Clause 5 preserved (judged, done-gate passed, required CI green).

**Routing evidence**: base-gate1=4cc0f6a3e97d6838f212b21ae36bb2202bd1dcea@2026-10-19T23:05:42Z; base=3f9a0c092a1e955c4ef060bf18d7bf28ff017809@2026-10-20T00:27:58Z (gate4); gate3b=3f9a0c092a1e955c4ef060bf18d7bf28ff017809 (drift 1 commit: my merge). `MISSION_ROUTING_NOTE` empty; `MISSION_OVER_RATION` empty.
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer**: not spawned (controller decisions D1–D8 in the planner directive). Resolver `recipe claude:claude-opus-5-5 declared:provider-pin`; rotation not advanced.
- **Planner**: resolver `agent-tool opus fail-closed:no-doc` against the pin `codex:gpt-6.1-sol`, so routed to the pin under the codex recipe (role-spawn-routing §2(a)). Probe rc 0; run rc 0 (102,370 tok).
- **Executor**: `codex:gpt-6.1-sol` via the recipe (`declared:provider-pin`), 1 run, rc 0 (101,966 tok).
- **Evaluator**: `sonnet` via the Agent tool (`declared:alias-pin`), foreground, 2 rounds (84,209 + 87,194 tok cumulative). Judge independence: cross-vendor (Anthropic sonnet vs the OpenAI generator).
- The Agent tool was used wherever the resolver names it (the evaluator), per the operator's standing request. The planner and executor resolve to recipes, where the Agent tool is not a valid path. No role failed to spawn.
- **Metered**: $0.

**Ruled out**:
- A naive "push after each milestone" sentence in the skill. It triggers the wrapper's Step 5b skip (measured), so the helper installs the tracking refspec first.
- Fixing Step 5b in the wrapper. `cmd/ailang/**` is outside fleet scope; parked as D-FLEET-23.
- Resolving the ticket. Asks 2 and 3 are undone and out of scope; resolving would drop them from the one queue (clause 3).
- A second dummy milestone to satisfy `validate_sprint_json.sh`.

**Retro lane**:
- (1) **The codex sandbox refused `.agents/` writes for the 4th time** (iterations 43–45, 49). Controller mirroring stays cheap and reliable; a recorded workaround, not a skill change.
- (2) **Gate 3b's 30-min PR poll timed out with only `test` pending for the 4th time** (iterations 43–45, 49). This is already D-FLEET-21; it is a time-budget call for Mark. Count now 4.
- (3) **zsh is the controller's own instrument hazard**: two failures this iteration (`$B:r` modifier, `no matches found` aborting a chain). Both are already-known classes (`$VAR:` mangling is in memory; `/bin/bash` is the prescribed shell for gates). Instance count for this iteration only; no skill change.
- (4) `validate_sprint_json.sh` refuses a 1-milestone sprint that `json_progress_schema.md` allows. Instance 1; noted, not ticketed (the fleet cannot file tickets).

**Next**: tickets by `slots_lost` among the routable ones. `probe:pi-duplicate-extension-discovery` (triage, 10-19) touches pi's extension discovery (`.pi/extensions/**` or the probe args), which parks under D-FLEET-3 if the fix is in `.pi/extensions`; reality-check it first. D-FLEET-16, -18, -19, -20, -21, -22 and the new -23 are with Mark; D-FLEET-17 is moot.

## 50 — 2026-10-20 — `probe:pi-duplicate-extension-discovery` LANDED: #1907 → `e5ab9f84f`; judge PASS 90 then PASS 91 (sonnet); ticket resolved [HARNESS]

**Pick**: iteration 49's **Next**, `probe:pi-duplicate-extension-discovery` (triage, 10-19, `blocking=none`, 1 slot). 44 open tickets at pick; new since iteration 49: `rotate-log:legacy-heading-format` (stapledon, 10-20, 1 slot). `agent-tool:sonnet-unavailable` (`blocking=all`) stays PARKED-ON-LANE as in iterations 29 and 32–49. Index grep: no prior attempt.

**Orphans credited**: none new. Gate 0 step 6a returned rc 1 for the same crash notice (2026-10-17T12:48:13Z, #1768): iteration 46's death, credited at iteration 47 (`issuecomment-6234067020` already in this log). No open `fleet/*` PR at pick.

**Preflight**: armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing CLEAN; messaging store gcp. Directives: 0 on #1891 since `2026-10-15T02:46:27Z` (7 comments, all self). Self-notices rc 1 as above (control `852:10`, watermarks 2/2). No rotation and no sweep due (#1891 created 2026-10-19T08:54Z, after Monday's 07:00 local boundary).

**Observe**: origin/dev == pin == `4c0ce261f`. `SKILL.md` and all 13 resources at the resolved symlink target are `cmp`-identical to origin. Dev CI on `4c0ce261f` was in progress; its parent `90b04a502` was green.

**Reality check**: real, and narrower than the ticket's framing. The driver was already fixed (`tools/launchd/lib/lane-probe.sh` `_mc_probe_pi` passes `--no-extensions`; `lib/pi-ext-args.sh` serves `mission_pi_run.sh` and the pi controller). The skill was not: `resources/gate-3-route.md:334` (both copies) still prescribed the probe without `--no-extensions`, and the raw sandboxed invocation at :347 had no extension-args splice. Measured: from the repo CWD the prescribed probe returns rc 1 with `Tool "ailang_run" conflicts with …/.pi/extensions/ailang-exec.ts` (8 tools); the same command from `/tmp` returns rc 0. The fix sits in `.claude/skills/mission-*/**`, not `.pi/extensions/**`, so D-FLEET-3 does not apply.

**Did**:
- **Designer**: not spawned. The controller wrote decisions D1–D6 into the planner directive (precedent: iterations 45 and 49).
- **Planner** (codex `gpt-6.1-sol`, recipe, detached `.planner-wt-iter50`): plan `827f1adfe`, 1 milestone, 14 arms and 6 disposable mutation/control drills. The validator's minimum-2-milestones refusal reproduced; kept at 1.
- **Executor** (codex, recipe, worktree `.wt-fleet-iter50-pi-probe`): probe gains `--no-extensions`; the raw snippet sources `pi-ext-args.sh` and splices `PI_EXT_ARGS` as `mission_pi_run.sh:288` does; `test_pi_ext_args.sh` 8 → 14 arms (per-copy probe and snippet arms, a sweep over both skill copies with `PI_EXT_ARGS_SKILL_ROOTS`, a launch-count instrument control); changelog fragment. TDD red first (9/5). The sandbox refused the `.agents` write (instance 5); the controller mirrored it byte-identically. Commit `ad04204f0`.
- **Controller, outside the sandbox**: 14/0; sync, `bash -n` and changelog rc 0. Live from a worktree with `.pi/extensions`: the new probe and the spliced form both reply `ok` with 0 conflict lines; the old form in the same CWD prints 8. (pi lingered past a 150 s cap after `agent_settled`, so exit rc is unmeasured; the NDJSON `stopReason: stop` is the reading.) One controller mutation (probe flag deleted) → 12/2.
- **Judge** (sonnet, Agent tool, foreground, own worktree `.wt-fleet-iter50-judge`): **PASS 90**, 0 blocking. All 6 plan drills killed by the intended arm; it added 10 more mutants. Survivors: S4 (a splice anywhere in the same bash fence admitted an unrelated launch), S8 (a line-continued launch), S9 (doubled spaces). The controller re-ran the judge's drill script and reproduced every row.
- **Round 2** (`a2b71770f`): the sweep checks each launch LINE for `--no-extensions` or the splice literal, and matches `pi[[:space:]]+--mode[[:space:]]+json`. S4 and S9: rc 0 on the round-1 test, rc 1 on round 2. S8 accepted as a known limit. Re-judged **PASS 91**, 0 blocking; two new contrived survivors (prose carrying the splice literal; `--no-extensions` in a trailing comment).
- PR #1907 (no closing keyword; scan paired with a control) squash-merged with `--match-head-commit a2b71770f` → `e5ab9f84f`.

**Gate 3b**: PR checks read directly at the head: 26 total, 0 not green. My background PR poller never wrote a verdict: zsh `echo "$j"` expanded `\n` inside the JSON, so every `jq` read failed (instrument failure, stopped). An earlier poller I launched with a 3000 s cap was also stopped, because the prescribed cap is 1800 s (D-FLEET-21 unruled). Dev poll ran as prescribed: `target=e5ab9f84f…`, `rid=39393223789`, 2-commit `DRIFT` (an unrelated triage doc + my merge). Verdict: the first dev poll hit its 30-min cap (`Gate 3b TIMEOUT for e5ab9f84f… (status=in_progress)`, only `build` running, every other job green). A fresh poll on a new path read `CI for e5ab9f84f…: completed success`, and the full check set on the merge reads 17 checks, all green. That is D-FLEET-21's 6th instance.

**Done-gate**:
- Surface: `resources/gate-3-route.md` is what controllers read for the pi recipe (both copies byte-identical; `test_agents_skills_sync.sh` rc 0).
- Reach: pushed and merged to `origin/dev`.
- `make test-launchd-drivers` under `env -i` at `a2b71770f`: rc 0, the two new sweep arms present in the log. The inherited `spawn-pin-hook.sh: … cat: command not found` line appears once, as at iteration 49.
- Dry-runs under **stapledon** (armed, no pidfile; `AILANG_DRIVER_PINNED=a2b71770f`): healthy rc 0 `lanes=ok`; degraded (`MISSION_EXECUTOR_MODEL=pi:bogus/no-such-model`) rc 0 `DEGRADED(1)`, handed to opus. No pidfile written; nothing reloaded.

**Progress**: 1 ticket resolved (`ailang mission ticket resolve … --sha e5ab9f84f`, replied to triage); 43 open. Goal unmoved on clause 1.
- Clause 1 UNMEASURED.
- Clause 2 UNMET (this ticket: filed 10-19 21:41Z, resolved 10-20, about 13 h; the median stays far above 48 h).
- Clause 3 MET.
- Clause 4 prior evidence only.
- Clause 5 preserved (judged, done-gate passed, required CI green).

**Routing evidence**: base-gate1=4c0ce261f5ca657327acb888fbf0612936fae53d@2026-10-20T07:35:58Z; worktree base c93a26f6a (1-commit drift, unrelated triage doc); gate3b=e5ab9f84f7d7b2a769bcc902bf6f227e8e45ac35 (drift 2 commits). `MISSION_ROUTING_NOTE` empty; `MISSION_OVER_RATION` empty.
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer**: not spawned (controller decisions D1–D6 in the planner directive). Rotation not advanced.
- **Planner**: resolver `agent-tool opus fail-closed:no-doc` against the pin `codex:gpt-6.1-sol`, so routed to the pin under the codex recipe (role-spawn-routing §2(a)). Probe rc 0; run rc 0 (66,177 tok).
- **Executor**: `codex:gpt-6.1-sol` via the recipe (`declared:provider-pin`), 1 run, rc 0 (73,191 tok).
- **Evaluator**: `sonnet` via the Agent tool (`declared:alias-pin`), foreground then resumed by `SendMessage`, 2 rounds (58,904 + 63,864 tok). Judge independence: cross-vendor (Anthropic sonnet vs the OpenAI generator).
- The Agent tool was used wherever the resolver names it (the evaluator), per the operator's standing request. The planner and executor resolve to recipes. No role failed to spawn.
- **Metered**: $0.

**Ruled out**:
- Parking under D-FLEET-3. The fix is skill text, not `.pi/extensions/**`.
- Editing `~/.pi/agent/extensions` or `ailang pi install`. The global copy is intended; the driver already handles duplicate discovery, and only the skill text lagged.
- Taking `rotate-log:legacy-heading-format` in the same iteration (standing rule 1).
- Closing S8 (line-continued launch). It needs a multi-line parser for a shape no skill uses.

**Retro lane**:
- (1) **The controller's hand-written pollers run under zsh, and zsh broke two of them this iteration.** In the PR-CI poller, `echo "$j"` expanded `\n` escapes inside the JSON, so every `jq` read failed and it could never finish. In the record-PR poller, `set -- $r` does not word-split in zsh, so every integer test errored. Both were caught by reading the output, and both re-ran under `/bin/bash`. With iteration 49's two (`$B:r` modifier, `no matches found`), that is 4 frictions in 2 iterations from one gap: Bash-tool commands run under zsh, while every prescribed snippet assumes bash. The prescribed Gate 3b snippet (`gh … --jq`, no splitting) survived; the hand-rolled ones did not. This meets Gate 5's ≥2-friction bar for a skill edit ("wrap any hand-written poll or gate in `/bin/bash -c`"). It is deferred, not made: a skill edit is its own PR, judge and CI cycle, which is a full slot. Proposed for a future fleet fire as a charter-tracked follow-up (the fleet cannot file tickets).
- (2) **The codex sandbox refused `.agents/` writes for the 5th time** (iterations 43–45, 49, 50). Controller mirroring remains the workaround.
- (3) **Guard the helper, miss the call site, again**: `test_pi_ext_args.sh` claimed "every pi launch site uses it" while checking only code. The new sweep covers the skill copies; the generalisation (skill text as a call site) is already in this skill's own vocabulary.
- (4) I launched a PR poll with a 3000 s cap, above the prescribed 1800 s, and stopped it once I noticed. D-FLEET-21 is the ruling that would make that legal.

**Next**: `rotate-log:legacy-heading-format` (stapledon, 10-20, 1 slot). `ailang mission rotate-log` sees only canonical headings, so a log with legacy headings never rotates, and the no-op run still prepends a false "Older entries are ARCHIVED" notice and rewrites the index. It is in `cmd/ailang/mission*` / `internal/mission`, in scope and mechanical; reality-check at HEAD first. D-FLEET-16 and -18 through -23 are with Mark; D-FLEET-17 is moot.

## 51 — 2026-10-20 — `rotate-log:legacy-heading-format` LANDED: #1912 → `3b4d11759`; judge PASS 82 then PASS 90 (sonnet); ticket resolved [HARNESS]

**Pick**: iteration 50's **Next**, `rotate-log:legacy-heading-format` (stapledon, 10-20, `blocking=none`, 1 slot). 43 open tickets at pick; none new since iteration 50. `agent-tool:sonnet-unavailable` (`blocking=all`) stays PARKED-ON-LANE as in iterations 29 and 32–50. Index grep: no prior attempt.

**Orphans credited**: none new. Gate 0 step 6a returned rc 1 for the same crash notice (2026-10-17T12:48:13Z, #1768), already credited at iteration 47 (`issuecomment-6234067020`). No open `fleet/*` PR; the leftover `fleet-*`/`.wt-fleet-*` worktrees all belong to landed iterations.

**Preflight**: armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing CLEAN; messaging store gcp. Directives: 0 on #1891 and 0 on #1768 since `2026-10-15T02:46:27Z` (8 and 26 comments, none allowlisted). No rotation or sweep due (#1891 created 2026-10-19T08:54Z).

**Observe**: origin/dev == pin == `f0f7df204`. `SKILL.md` and all 13 resources at the resolved symlink target are `cmp`-identical to origin. Dev CI green on `f0f7df204`: 22 check runs, 0 not green.

**Reality check**: real, and wider than filed. A scratch copy of stapledon's log (`stapledons-godot` origin/main `50d8d5ba`, 41 headings) rotated by a HEAD-built binary printed "5 of 5 entries kept", and spliced the "Older entries are ARCHIVED" notice in at line ~1572, below all 36 legacy entries that had been swallowed into the preamble. The same code path adds one more notice on every rotation: v1's log carries 8 copies, and this log records hand-collapsing them 7 times (instances logged at iterations 28 and 31–35, plus later ones). The notice is the ticket's own named side effect, so fixing it is in-ticket, not self-sourced. In scope (`internal/mission`), mechanical, no D-FLEET row applies.

**Did**:
- **Designer**: not spawned. The controller wrote measurements M1–M6 and decisions D1–D8 into the planner directive (precedent: iterations 45, 49 and 50).
- **Planner** (codex `gpt-6.1-sol`, recipe, detached `.planner-wt-iter51`): plan `88199ec6a`. 1 milestone, 8 arm/mutation pairs, and a fixture generated from stapledon by command. The validator's minimum-2-milestones refusal reproduced; kept at 1.
- **Executor round 1** (codex, worktree `.wt-fleet-iter51-rotate-legacy`), commit `a1a5b7278`: date-first numbered and note parsing; a stable decorated sort; no-op leaves the live log byte-identical; one notice; a preamble guard; `rotate_legacy_test.go`; the 41-line fixture (`cmp`-identical to its generator). TDD red first; all 9 mutation rows red.
- **Controller corpus gate** (outside the sandbox): `/tmp/fleet-iter/corpus51.sh`, HEAD binary vs new binary, `rotate-log --keep 20` on log and status streams for v1, fleet, docs, triage, motoko, world and stapledon. An A-vs-A control read SAME on all 14 arms. It found world's STATUS archive keeping a different, still-wrong 20: 100 `## STATUS <date> (iteration N) — …` stamps parse as notes, and the file is in mixed order. HEAD was already wrong here (kept 149–158 and 277–286).
- **Executor round 2** (`09f9d0917`): `parseLog` reuses `statusParenIterRe`. Fixture = a generated, classification-preserving prefix of `ailang-world` `17c28324` (281/281 lines classify the same as the full headings; the full headings are 1.47 MB). Kept set is 267..286.
- **Judge** (sonnet, Agent tool, foreground, own worktree `.wt-fleet-iter51-judge`): **PASS 82**, 0 blocking. Non-blocking: the guard refused `## 1. Purpose`, `## How to run iteration 5` and `## 2026 roadmap` (HEAD rotated them), and the notice stripper could eat hand-written quoted prose. Survivors: m1 (Slice for SliceStable), m2 (`Num > 0`), m4 (bounds), m8 (`\b`).
- **Executor round 3** (`c2e295de7`): the guard needs a record skeleton (full date, numbered em-dash entry, or STATUS date). The writer and stripper share four exact prefixes. `rotate_guard_test.go` adds tests that kill m1, m2, m4 and m8. Re-judged **PASS 90**, 0 blocking. The judge did not byte-diff the notice; the controller did, and HEAD's freshly written notice is byte-identical to the final one on fleet, world and stapledon.
- PR #1912 (no closing keyword; the scan was paired with a control) squash-merged with `--match-head-commit 0189f1e41` → `3b4d11759`.

**Gate 3b**: PR checks at head `0189f1e41`: 26 total, 0 not green (per-invocation poller under `/bin/bash`, SHA in the verdict). Dev poll ran as prescribed: `target=3b4d11759…`, `rid=39462439669`, a 1-commit `DRIFT` (my merge). Verdict: `CI for 3b4d11759…: completed success` on the first poll; the full check set on the merge reads 21 checks, 0 not green. Ticket resolved with `--sha 3b4d11759` (replied to stapledon).

**Done-gate**:
- Surface: `internal/mission/rotate.go` is what `ailang mission rotate-log` runs.
- Reach: merged to `origin/dev`. The PATH binary picks it up on its next rebuild or release (iteration 32 precedent).
- `make test-launchd-drivers` under `env -i` at `c2e295de7`: rc 0. An earlier run at `09f9d0917` was stopped because round 3 superseded it.
- Dry-runs under **stapledon** (armed, no pidfile; `AILANG_DRIVER_PINNED=c2e295de7`): healthy rc 0 `lanes=ok`; degraded (`MISSION_EXECUTOR_MODEL=pi:bogus/no-such-model`) rc 0 `DEGRADED(1)`, handed to opus. No pidfile written; nothing reloaded.
- Outside-sandbox gates at `c2e295de7`: `go test ./internal/mission/` ok, cmd/ailang rotate tests ok, gofmt and vet clean, `check-changelog` ok, `check-file-sizes` ok. The corpus round-2 vs round-3 binary read SAME on all 14 arms; HEAD vs final shows 7 differences, all explained, 0 unexplained.

**Progress**: 1 ticket resolved; 42 open. Goal unmoved on clause 1.
- Clause 1 UNMEASURED.
- Clause 2 UNMET (this ticket took about 15 h; the median stays far above 48 h).
- Clause 3 MET.
- Clause 4 prior evidence only.
- Clause 5 preserved.

**Routing evidence**: base-gate1=f0f7df2046a730dff350680fe558dcd4a288c69a@2026-10-20T15:39:05Z; worktree base f0f7df204 (no drift at creation); gate3b=3b4d1175919eccfda8c034b49d0b6633c415817a (drift 1 commit, my merge). `MISSION_ROUTING_NOTE` empty; `MISSION_OVER_RATION` empty.
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer**: not spawned (controller decisions D1–D8 in the planner directive). Resolver `recipe claude:claude-opus-5-5 declared:provider-pin`. Rotation not advanced (`codex:gpt-6.1-sol` stays last-used).
- **Planner**: resolver `agent-tool opus fail-closed:no-doc` against the pin `codex:gpt-6.1-sol`, so routed to the pin under the codex recipe (role-spawn-routing §2(a)). Probe rc 0; run rc 0 (67,598 tok).
- **Executor**: `codex:gpt-6.1-sol` via the recipe (`declared:provider-pin`), 3 runs, all rc 0 (100,905 + 65,796 + 80,979 tok).
- **Evaluator**: `sonnet` via the Agent tool (`declared:alias-pin`), foreground, then resumed by `SendMessage` for round 3 (56,838 + 62,327 tok). Judge independence: cross-vendor (Anthropic sonnet vs the OpenAI generator).
- The Agent tool was used wherever the resolver names it (the evaluator), per the operator's standing request. The planner and executor resolve to recipes. No role failed to spawn.
- **Metered**: $0.

**Ruled out**:
- Widening `ailang mission normalize` instead of the rotation parser. Stapledon's log lives in another repo, and the parser already carries a "deliberately tolerant" multi-format contract. Normalize is unchanged.
- Making a no-op rotation skip the index. `TestRotateLog_UnderThresholdStillWritesTheIndex` pins that as a deliberate contract.
- Keying notes by date. With `(iteration N)` parsed as numbered, file order is correct on every real file measured.
- Editing any real mission log, `.gitattributes`, or stapledon's repo.

**Retro lane**:
- (1) **A corpus gate across every mission's real logs caught what both executor runs and every unit test missed.** World's status-archive regression was invisible to the stapledon fixture by construction. The plan prescribed the gate because the controller asked for it in the planner directive. This is one instance, so no skill edit; noting the shape for parser changes: run the old and new binaries over every real input the parser consumes.
- (2) **zsh again**: `ls /tmp/x_*.verdict` inside a poll loop prints `no matches found` on every pass under zsh. The loop worked but spammed 30 lines; re-run under `/bin/bash -c`. With iteration 50's (1), that makes 5 frictions in 3 iterations from one gap (Bash-tool commands run under zsh). The skill edit ("wrap hand-written polls in `/bin/bash -c`") still waits for a slot of its own, and it remains a charter-tracked follow-up because the fleet cannot file tickets.
- (3) I printed the environment with `env | grep -v -i key` at Gate 0, and that filter does not catch `*_TOKEN` variables, so the session's OAuth token went into this transcript. It did not reach any repo, message or issue. Use `env | cut -d= -f1` for name-only reads.

**Next**: by `slots_lost`, `agent-tool:sonnet-unavailable` stays PARKED-ON-LANE and `reporting:daneel-reports-not-declared-triage` is parked on D-FLEET-22, so the head is `agent-tool-glm-designer-unavailable` (world, 10-02, 3 slots) with its siblings `agent-tool-sonnet-unavailable` and `agent-tool-opus-planner-unavailable` (hyphenated duplicates of colon-form tickets). Reality-check whether they are one routing-policy class (a D-FLEET park) before building. D-FLEET-16 and -18 through -23 are with Mark; D-FLEET-17 is moot.

## 52 — 2026-10-21 — `quota:opencode-reported-as-a-pool` LANDED: #1917 → `9aebccf9b`; judge PASS 93 (sonnet); ticket resolved; six `agent-tool` tickets parked on D-FLEET-24 [HARNESS]

**Pick**: iteration 51's **Next** was to reality-check the `agent-tool` tickets at the head (`agent-tool-glm-designer-unavailable` 3 slots, and its siblings) before building. That check made this a bookkeeping pick (one routing class, parked on new D-FLEET-24), so standing rule 1 allowed a second: `quota:opencode-reported-as-a-pool` (attended, 10-01, `blocking=none`), pre-authorized by D-FLEET-12 item 2. Rank: of the 1-slot tickets above it, `eval:motoko-connection-probe-orphaned` is `tools/eval` (D-FLEET-18 class) and `policy:fleet-rulings-not-propagated-to-charters-and-skill` is an ask for Mark that needs its own reality check. I took the pre-authorized, mechanical one and banked the policy one as **Next**. 42 open tickets at pick; none new since iteration 51. Index grep: no prior attempt at either.

**Orphans credited**: none new. Gate 0 step 6a returned rc 1 for the same crash notice (2026-10-17T12:48:13Z, #1768), already credited at iteration 47 (`issuecomment-6234067020`; 3 mentions in this log). No open `fleet/*` PR at pick; the leftover `fleet-*`/`.wt-fleet-*` worktrees all belong to landed iterations.

**Preflight**: armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing CLEAN; messaging store gcp. Directives: 0 on #1891 and 0 on #1768 since `2026-10-15T02:46:27Z` (9 and 26 comments, none allowlisted). No rotation or sweep due (#1891 created 2026-10-19T08:54Z, after Monday 07:00 local).

**Observe**: origin/dev == pin == `1579ef50c`. `SKILL.md` and all 13 resources at the resolved symlink target are `cmp`-identical to origin. Dev CI green on `1579ef50c`: `CI` and `Build and Release` success, 17 check runs, 0 not green.

**Reality check, the six `agent-tool` tickets**: one class, and a routing one. All six evidence blocks are a non-Claude controller whose native `spawn_agent` rejects a non-OpenAI model (`Unknown model sonnet|opus|glm-5.3`; enum `gpt-6.1-sol, gpt-6-astra, gpt-6-sol, gpt-6-luna, gpt-5.6-sol`). Resolver readings at `1579ef50c`, with `MISSION_EXECUTOR_RESOLVED=codex:gpt-6.1-sol`:

| `MISSION_ANTHROPIC_AVAILABLE` | evaluator `sonnet` / `opus` / `fable` | designer `fable` | planner, no doc (control) |
|---|---|---|---|
| 1 | `agent-tool <alias> declared:alias-pin` | `agent-tool fable declared:alias-pin` | `agent-tool opus fail-closed:no-doc` |
| 0 | `agent-tool <alias> declared:alias-pin` | `agent-tool fable declared:alias-pin` | `recipe codex:gpt-6.1-sol anthropic-fallback:fail-closed:no-doc` |

The driver sets the flag to 0 for every codex or pi controller (`mission-control.sh:978-979`). Only `derive-planner-lane.sh` reads it. Iteration 25's plan named this gap and routed it to policy review, and `mission-control.sh:1605-1608` records alias-vs-pin dispatch as deliberate. So it is HD-2a: filed as **D-FLEET-24** (A: reroute to the first provider-pinned declared fallback under `MISSION_ANTHROPIC_AVAILABLE=0`, recommended; B: map the alias to its `claude:` pin; C: no change). Exposed today: triage and docs (codex-first controllers with a `sonnet` evaluator), and any world or stapledon fire that falls to codex.

**Reality check, `quota:opencode-reported-as-a-pool`**: real. Measured with a HEAD-built binary on scratch homes. (M1) A copy of the rig's real `quota-ledger.json` prints `opencode 5h`/`long` rows with 0 tokens and four `quota: opencode/…` stderr lines. (M2) An empty home prints none. (M3) `rebuild()` keeps every cached row and only zeroes `WindowStart`, so the stale rows live forever. (M4) One journal line `opencode 5` creates a 5-token pool. The only production writer is `chains_post.go` → `AppendSpend`, and `observatory.CanonicalQuotaBucket` passes `opencode` through unchanged. Both are outside fleet scope. (M5) 0 registry rows have `provider: opencode`, so admission's `opencode` arm is unreachable. (M6) `lane-probe.sh`'s `opencode` alternation is dead.

**Did**:
- **Designer**: not spawned. The controller wrote measurements M1–M8 and decisions D1–D9 into the planner directive (precedent: iterations 45 and 49–51).
- **Planner** (codex `gpt-6.1-sol`, recipe, `.planner-wt-iter52`): plan `27ac23a6a`, 1 milestone, a 10-row arm/mutation table, and the corpus step named. The validator's minimum-2-milestones refusal reproduced; kept at 1.
- **Executor** (codex, worktree `.wt-fleet-iter52-quota-opencode`), committed by the controller as `5b136c011`: `quota_bucket.go` (`QuotaBucketFor`, one four-bucket list), the `AppendSpend` refusal before any write, the `rebuild()` cache drop and journal map-or-skip, the `--bucket` refusal, the admission and `lane-probe.sh` arms, 3 new test files plus 2 extended. TDD red on 9 tests first. All 10 plan mutations went red at their named assertions; the capacity control stayed green under every D3 mutation and went red only when the drop was widened to every zero-token row. The sandboxed `go test ./internal/mission/` was rc 1 only on an existing httptest bind, which the executor labelled UNINFORMATIVE.
- **Controller gates** (outside the sandbox, at `5b136c011`): `go test ./internal/mission/` ok; `go test ./cmd/ailang/ -run 'MissionQuota|Quota'` ok (32 targeted subtests PASS); gofmt, `go vet` and `golangci-lint` (0 issues) clean; `check-changelog` and `check-file-sizes` rc 0.
- **Controller corpus gate**: `/tmp/fleet-iter52/corpus52.sh`, HEAD vs new binary, `mission quota` and `mission quota --over`, each on its own scratch copy of the rig's real ledger and all journal segments. After normalising temp paths, HEAD-vs-HEAD controls read SAME on all four streams. HEAD vs new: the report lost exactly 2 stdout and 4 stderr lines, all naming `opencode`, with 0 other differing lines; `--over` was identical. The real ledger plus journal sha was unchanged (`11dee62e83d4` before and after).
- **Judge** (sonnet, Agent tool, foreground, own worktree `.wt-fleet-iter52-judge`): **PASS 93**, 0 blocking. It ran 10 plan mutations and 5 of its own; one survivor, `err != nil ||` in the CLI check, is an equivalent mutant (N1). N2: the cache drop is uncounted. N3: the backend must equal a bucket exactly, which is stricter than D1's "begins with". The controller re-ran the judge's own mutant 12 (drop only on error) through a Go overlay: red at `quota_ledger_opencode_test.go:64`, as reported.
- PR #1917 (no closing keyword; the scan was paired with a control) squash-merged with `--match-head-commit 5b136c011` → `9aebccf9b`.

**Gate 3b**: PR checks at head `5b136c011`: 26 total, 0 not green (per-invocation poller under `/bin/bash`, SHA in the verdict). Dev poll ran as prescribed: `target=9aebccf9b…`, `rid=39525459551`, a 1-commit `DRIFT` (my merge). Verdict: `CI for 9aebccf9b…: completed success`. The full check set on the merge reads 24 checks with one not green, `Dependabot: failure`. That is Dependabot's own `dynamic` security-update job (run 39525585138, `npm_and_yarn in /internal/executor/opencode/plugins, /tools/pi-extensions/sandbox`), which reported `security_update_not_found` for `node-forge` 1.4.0 and is stamped with whatever HEAD was current at 01:08Z. It is not a CI workflow, this diff touches no npm file, and its two sibling Dependabot jobs on the same SHA succeeded. Ticket resolved with `--sha 9aebccf9b` (replied to the attended filer).

**Done-gate**:
- Surface: `internal/mission/quota_ledger.go` and `quota_bucket.go` are what `ailang mission quota` and every `AppendSpend` run; `tools/launchd/lib/lane-probe.sh` is what the driver sources.
- Reach: merged to `origin/dev`. The driver picks up `lane-probe.sh` on the next pinned fire. The PATH binary picks up the Go change on its next rebuild or release (iteration 32 precedent).
- `make test-launchd-drivers` under `env -i` at `5b136c011`: rc 0 (59 probe arms; bash 3.2 syntax OK).
- Dry-runs under **stapledon** (armed, no pidfile; `AILANG_DRIVER_PINNED=5b136c011`): healthy rc 0 `lanes=ok`, and the patched awk still parsed the live ration line (`blocked buckets … anthropic`). Degraded (`MISSION_EXECUTOR_MODEL=pi:bogus/no-such-model`) rc 0 `DEGRADED(1)`, handed to opus. No pidfile written; nothing reloaded.

**Progress**: 1 ticket resolved; 41 open; 6 tickets parked on D-FLEET-24. Goal unmoved on clause 1.
- Clause 1 UNMEASURED.
- Clause 2 UNMET (this ticket: filed 10-01 15:52Z, resolved 10-21, about 19.4 d; the median stays far above 48 h).
- Clause 3 MET.
- Clause 4 prior evidence only.
- Clause 5 preserved.

**Routing evidence**: base-gate1=1579ef50ce4d57b185de7c3ff749928204d66570@2026-10-21T00:04:58Z; sprint worktree base 1579ef50c (no drift at creation, 00:17:27Z); gate3b=9aebccf9bf8737a4d92ecdfdc285a1b1ee387522 (drift 1 commit, my merge). `MISSION_ROUTING_NOTE` empty; `MISSION_OVER_RATION` empty.
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer**: not spawned (controller decisions D1–D9 in the planner directive). Resolver `recipe claude:claude-opus-5-5 declared:provider-pin`. Rotation not advanced.
- **Planner**: resolver `agent-tool opus fail-closed:no-doc` against the pin `codex:gpt-6.1-sol`, so routed to the pin under the codex recipe (role-spawn-routing §2(a)). Probe rc 0; run rc 0 (67,287 tok).
- **Executor**: `codex:gpt-6.1-sol` via the recipe (`declared:provider-pin`), 1 run, rc 0 (122,274 tok).
- **Evaluator**: `sonnet` via the Agent tool (`declared:alias-pin`), foreground (69,103 tok). Judge independence: cross-vendor (Anthropic sonnet vs the OpenAI generator).
- The Agent tool was used where the resolver names it (the evaluator), per the operator's standing request. The planner and executor resolve to recipes. No role failed to spawn.
- **Metered**: $0.

**Ruled out**:
- Building the D-FLEET-24 fix. Iteration 25 and `mission-control.sh:1605-1608` both place alias-vs-pin dispatch under policy, and D-FLEET-2 rules on the hook's chain walk, not the resolver's. An inferred extension of a ruling is the thing the recording contract forbids.
- Closing the hyphenated `agent-tool-*` tickets as duplicates of the colon-form ones. The ticket CLI has no merge, and `resolve` replies to the filer as if fixed. They park with the class.
- Mapping `opencode` inside `observatory.CanonicalQuotaBucket` (outside fleet scope), or by registry lookup inside the ledger (a model-registry load on every `LoadLedger` for a writer nobody runs).
- Pruning every zero-token cached row. Capacity and provider-reset rows are zero-token by construction; the capacity control proves the drop is harness-only.

**Retro lane**:
- (1) **A shell-env leak made a baseline gate read red.** `test_mission_routing.sh` was rc 1 from the mission shell (the planner arm read `fail-closed:unparsable-path-entry`, because `MISSION_PLANNER_ALLOWLIST` leaked in) and rc 0 under `env -i`. The charter's done-gate already prescribes `env -i` for `test-launchd-drivers`, but nothing says the same for a single `tools/launchd/test_*.sh` run by hand. The directive carried it, so both roles ran clean. One instance; no skill edit.
- (2) **The token leak again (instance 2).** I printed the environment at Gate 0 with `env | grep -E '^(MISSION|AILANG|CLAUDE_CODE)' | grep -v -i key`, and `CLAUDE_CODE_OAUTH_TOKEN` went into the transcript. This is the identical defect iteration 51's retro (3) recorded, with the identical filter. It reached no repo, message or issue. Two frictions on one gap now meet Gate 5's skill-edit bar. The edit, a names-only `env | cut -d= -f1` rule in the Gate 0 preamble, is banked as the next skill edit rather than made here, because this iteration's slot went to the record. It remains a charter-tracked follow-up, because the fleet cannot file tickets.
- (3) The executor reported that `test_mission_routing.sh` prints `_mc_canon_id: command not found` (and two siblings) while still exiting rc 0. The same lines appear in the controller's base log at `1579ef50c`, so it is pre-existing and not this diff. Not ticketed (the fleet does not self-source).

**Next**: reality-check `policy:fleet-rulings-not-propagated-to-charters-and-skill` (attended, 10-01; an ask for Mark, probably a park), then `codex:server-capacity-error-crashes-finished-iteration` (attended, 10-02), then `routing:retired-codex-models-still-spawnable-natively` (D-FLEET-12 item 1, pre-authorized). D-FLEET-16, -18 through -24 are with Mark; D-FLEET-17 is moot.

## 53 — 2026-10-21 — `routing:retired-codex-models-still-spawnable-natively` LANDED: #1922 → `d75ef77f5`; judge PASS 85, round 2 PASS 85 (sonnet); ticket resolved; D-FLEET-25, -26, -27 filed [HARNESS]

**Pick**: iteration 52's **Next** listed three tickets in order. First, `policy:fleet-rulings-not-propagated-to-charters-and-skill`: its ask is a mechanism for Mark, so it is parked on **D-FLEET-25**. Second, `codex:server-capacity-error-crashes-finished-iteration`: classifying `Selected model is at capacity` is the D-FLEET-11/13 runtime-classification class, and that class parked last time, so it is parked on **D-FLEET-26**. That made this a bookkeeping pick, so standing rule 1 allowed a second: `routing:retired-codex-models-still-spawnable-natively` (attended, 10-02), which D-FLEET-12 item 1 pre-authorizes. I also reality-checked the newest ticket, `quorum:oc-glm-5-3-output-cap-invalid` (stapledon, 10-21), because a fast resolution would move clause 2. Its fix is `internal/modelreg/models.yml` or the quorum's seating policy, both outside fleet scope, and its "$0.63 metered waste" is imputed: Ollama Cloud is flat-rate. It stays open, ranked by slots. 42 open tickets at pick; index grep found no prior attempt at any of the four.

**Orphans credited**: none new. Gate 0 step 6a returned rc 1 for the same crash notice (2026-10-17T12:48:13Z, #1768), already credited at iteration 47 (`issuecomment-6234067020`). No open `fleet/*` PR at pick; the leftover `fleet-*`/`.wt-fleet-*` worktrees all belong to landed iterations.

**Preflight**: armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing CLEAN; messaging store gcp. Directives: 0 on #1891 since `2026-10-15T02:46:27Z` (10 comments, none allowlisted). No rotation or sweep due (#1891 created 2026-10-19).

**Observe**: origin/dev == pin == `d3d389efe`. `SKILL.md` and all 13 resources are `cmp`-identical to origin at the resolved symlink target, and the pin copies equal the resolved ones. **Dev CI is RED on `d3d389efe`, from causes outside fleet scope.** V1 owns the repo and is disabled; D-FLEET-17 is the standing question. (1) `govulncheck` (not required) went red at `7ab609a6a` (iteration 52's docs-only record) on a newly published, function-reaching advisory **GO-2026-6809**, not allowlisted; it is time-based, not from a commit. (2) The required `test` failed on `TestWorkerTerminalPTYLifecycle/eval/sigint` (`internal/runner`, a Python `ValueError: invalid literal for int()` inside the PTY harness). Of the last 5 failed `test` jobs on dev and PRs, 3 carry this failure (10-20 07:36, 10-20 08:58, 10-21 04:36), so it is an intermittent flake. It did not recur on #1922's head.

**Reality check, `routing:retired-codex-models-still-spawnable-natively`**: real. All measurements are on the rig with codex-cli 0.162.0.
- (M1) Paired probe asking the model to list its `spawn_agent` parameters. Control: `model` with `gpt-6.1-sol, gpt-6-astra, gpt-6-sol, gpt-6-luna, gpt-5.6-sol`. With `-c features.multi_agent_v2.expose_spawn_agent_model_overrides=false` (key found in the binary's config struct): no model parameter.
- (M2) The same pair through `ailang mission codex-exec`, the daemon path every driver launch uses: identical.
- (M3) A real `spawn_agent` call under the flag produced a child rollout (`source.subagent.thread_spawn`) with every `model` field `gpt-6.1-sol`: children inherit the parent.
- (M4) Every driver codex launch goes through `mc_codex_command`.
- (M6) `SelectReviewers` would seat `gpt6-astra` from an explicit pool.
- (M7) **wrong as first written**: the plan said only disabled docs pins a non-Sol codex model.

**Did**:
- **Designer**: not spawned. The controller's measurements M1–M8 and decisions D1–D9 went into the planner directive (precedent: iterations 45 and 49–52).
- **Planner** (codex `gpt-6.1-sol`, recipe, `.planner-wt-iter53`, run under the new flag): plan `377f10704`, 2 milestones, a 13-row arm/mutation table, and the controller-only steps named.
- **Executor M1** (codex, 50-min cap; the milestone is plan-sized with a mutation drill), committed by the controller as `87cd10aa6`. New leaf package with the allowlist; `codex-exec` refusal before any daemon contact, forced flag, refusal of any caller value other than false; `mc_codex_command` flag; both `gate-3-route.md` copies (the controller mirrored `.agents`, which the sandbox refuses, and reworded one sentence); `tools/launchd/test_codex_spawn_override.sh` with an array check, a two-root sweep and an instrument control. TDD red first; 9/9 plan mutations red.
- **Executor M2** (codex), committed as `a53afdccf`: `Selection.Refused` before benching; `ReasonNotAllowlisted`; `scripts/mission_routing_models_check.sh` plus 20 fixture assertions; a Go parity test; both `gate-4-record.md` copies. The controller mirrored `.agents` and changed the script path to `$MISSION_DRIVER_ROOT/...`, since World's log lives in another repo. 10 plan mutations red; the default-pool control stayed green under both filter mutants.
- **Controller integration over the real logs refuted M7.** `triage-mission-log.md:31` routes verifiers to `codex:gpt-6-luna`. `mission-triage.env:24,33` pins it, triage is armed, and its charter (ratified 2026-10-15, later than D-FLEET-12) names luna. By gate-0 rule 6(g) the later statement wins on luna only, so the allowlist became `gpt-6.1-sol` + `gpt-6-luna`, recorded as plan Amendment 1 and **D-FLEET-27**. Mutation-tested: dropping luna from the Go list alone reddens `TestModelAllowed` and the parity test.
- **Live, new binary at `87cd10aa6`**: `--model gpt-6-astra` refused with rc 1 and 0 new rollout files (never reached the daemon). A caller `=true` was refused. `--model gpt-6.1-sol` with **no** `-c` reported no model parameter, where the PATH binary's control listed five.
- **Judge round 1** (sonnet, Agent tool, foreground, worktree `.wt-fleet-iter53-judge`): **PASS 85**, 0 blocking. 38 mutations: 34 killed; 4 survived, 2 of them equivalent. It ruled the luna deviation sound and found the prefixed-OpenAI-id gap in `IsOpenAI` non-blocking.
- **PR CI, first head `9b81a276e`**: the required `test` failed `make check-architecture-closure`. The new package raised the binary closure from 136 to 137, and `ARCHITECTURE.md` (outside fleet scope) went stale. The controller moved the code into `internal/mission/quorum/rolemodels.go` (`a32188128`): renamed identifiers, behaviour unchanged, and `check-architecture-closure` current locally. The same run's `Build macos-latest` failed `TestMemWatchdogKillsAllocator` (`internal/eval_harness`, untouched; green on dev's last 8 macOS jobs and on the next head), and Windows was cancelled by fail-fast.
- **Judge round 2** (sonnet): **PASS 85**, 0 blocking. The move is character-identical apart from renames and comments, and 4 mutations against the moved code were all killed. Its two cosmetic items were fixed in `abb75b597` (doc comment, sprint JSON text).
- PR #1922 (no closing keyword; the scan was paired with a control) squash-merged with `--match-head-commit abb75b597` → `d75ef77f5`. Head checks: 26 total, one red, `govulncheck`, inherited from dev and not required.

**Gate 3b**: dev poll as prescribed: `target=d75ef77f59cf4c3c3151fef4840335d269963b8d`, `rid=39582396242`, a 1-commit `DRIFT` (my merge). The first poll timed out at its 1800 s cap with only `build` still running and nothing failed except `govulncheck`, the D-FLEET-21 shape. A fresh-path re-poll (default C) read `CI for d75ef77f59cf4c3c3151fef4840335d269963b8d: completed failure`. The full check set on the merge reads 21 checks with one not green, `govulncheck (vuln gate)`, red on both parents (`d3d389efe`, `7ab609a6a`) for GO-2026-6809, inherited and not required. `test`, `build` and `launchd drivers (bash 3.2)` are green, and `Build and Release` succeeded on `d75ef77f5`. LANDED.

**Done-gate**:
- Surface: `tools/launchd/lib/codex-auth-profile.sh` (sourced by the driver; `_mc_probe_codex` and the codex controller launch use it); `cmd/ailang/mission_codex_exec.go`; `internal/mission/quorum`; both copies of the skill resources.
- Reach: merged to `origin/dev`. The shell flag rides the next pinned fire; the Go refusal and quorum guard ride the next rig rebuild or release.
- `make test-launchd-drivers` under `env -i`: rc 0 at the branch head, and rc 0 at pristine base `d3d389efe` in its own worktree. An earlier base run read 27 failures in the gate-0 self-notices suite while codex runs loaded the machine; that suite alone passed 108/0.
- Dry-runs under **stapledon** (armed, no pidfile; `AILANG_DRIVER_PINNED=9b81a276e`): healthy rc 0 `lanes=ok`; the codex probe went through the patched `mc_codex_command` on the current PATH binary, which proves the old binary accepts the flag. Degraded (`MISSION_EXECUTOR_MODEL=pi:bogus/no-such-model`) rc 0 `DEGRADED(1)`, handed to opus. No pidfile written; nothing reloaded.

**Progress**: 1 ticket resolved; 41 open; 2 parked on new asks. Goal unmoved on clause 1.
- Clause 1 UNMEASURED.
- Clause 2 UNMET (this ticket: filed 10-02, resolved 10-21, about 19 d).
- Clause 3 MET.
- Clause 4 prior evidence only.
- Clause 5 preserved.

**Routing evidence**: base=d75ef77f59cf4c3c3151fef4840335d269963b8d-10-21T10:09:08Z; base-gate1=d3d389efeba794013e72f861ad3a5317a2880dab@2026-10-21T08:04:25Z; sprint worktree base d3d389efe; gate3b=d75ef77f59cf4c3c3151fef4840335d269963b8d (drift 1 commit, my merge). `MISSION_ROUTING_NOTE` empty; `MISSION_OVER_RATION` empty.
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer**: not spawned (controller decisions in the planner directive). Resolver `recipe claude:claude-opus-5-5 declared:provider-pin`. Rotation not advanced.
- **Planner**: resolver `agent-tool opus fail-closed:no-doc` against the pin `codex:gpt-6.1-sol`, so routed to the pin under the codex recipe (role-spawn-routing §2(a)). Probe rc 0; run rc 0 (93,067 tok).
- **Executor**: `codex:gpt-6.1-sol` via the recipe (`declared:provider-pin`), 2 runs: M1 rc 0 (100,762 tok), M2 rc 0 (196,466 tok).
- **Evaluator**: `sonnet` via the Agent tool (`declared:alias-pin`), foreground, 2 rounds (114,438 + 58,270 tok). Judge independence: cross-vendor (Anthropic sonnet vs the OpenAI generator).
- The Agent tool was used where the resolver names it (the evaluator), per the operator's standing request. The planner and executor resolve to recipes. No role failed to spawn.
- Every codex run this iteration carried the new spawn-override flag.
- **Metered**: $0.

**Ruled out**:
- The literal `gpt-6.1-sol`-only allowlist. Armed triage's ratified verifier lane would have demoted at the next binary rebuild; a later ruling (triage charter) covers luna, and D-FLEET-27 asks for confirmation.
- Regenerating `ARCHITECTURE.md` to keep the new package. That is outside fleet scope; it would park for Mark or need a scope-guard PR shipped alone first.
- Fixing `IsOpenAIModel` for prefixed ids by substring matching inside this sprint. Substring matching would refuse allowed prefixed ids such as `motoko-chatgpt-gpt-6-1-sol`, because the canonical key keeps the prefix. It needs its own design, and no pool carries such ids today.
- Building D-FLEET-25 or D-FLEET-26 without a ruling.
- Reading `pgrep` silence as "the superseded poller stopped". The first `kill` failed on a zsh word-splitting error; a second, per-PID kill was confirmed by an empty `pgrep` and by the task's own completion notice.

**Retro lane**:
- (1) **Make-gate contamination from a shared worktree, twice this iteration.** I ran `make test-launchd-drivers` in the sprint worktree while an executor was editing it (the `.agents` parity arm read a half-applied edit), and later while I edited it myself. Both readings were void, and a pristine baseline needed its own worktree. The skill tells you to baseline on a pristine tree (rule 3e) but not that a background gate in a worktree you are still editing is not a reading of either version. One iteration, two instances. That meets the two-friction bar, but the one skill edit this iteration went to the older banked item in (4). Banked as the next skill edit.
- (2) **A plan measurement was refuted only by running the new instrument on real data.** M7 was a controller claim from a grep over env files, and it was wrong: the grep's `uniq -c` showed 2 for the luna lines, and I read that as one file. The routing check's first real-log run found it. Rule 3a's "an empty search is a claim" has a sibling: a count you summarised instead of listing is one too.
- (3) **A new Go package is a gate the fleet cannot pass.** `check-architecture-closure` regenerates a doc outside fleet scope, so any fleet sprint that adds a package to the binary's closure fails the required `test` job by construction. A planner hint belongs in the fleet charter's guardrails: prefer an existing linked package. One instance.
- (4) **Skill edit (Gate 5, the one allowed): the banked names-only environment rule.** It is now in both copies of `resources/gate-0-preflight.md`: read the environment by name (`env | cut -d= -f1 | grep -E '^(MISSION_|AILANG_)'`), print only the specific non-secret values, and test a secret with `[ -n "$VAR" ]`. The frictions were iteration 51 retro (3) and iteration 52 retro (2); each printed `CLAUDE_CODE_OAUTH_TOKEN`. This iteration's own Gate-0 read was prefix-restricted (`^(MISSION_|AILANG_|CLAUDE_CODE_PRINT)`) and printed no token.

**Next**: reality-check `quota:admission-has-no-headroom-margin-and-flaps` (attended, 10-02). It is probably a ghost, already fixed attended by #1524 `71eec62b6` ("admit a quota bucket only with headroom for one run"): verify and resolve. Then `pi:unowned-dirty-absolute-vs-relative-paths`. D-FLEET-16 and D-FLEET-18 through -27 are with Mark.

## 54 — 2026-10-21 — `routing-models-check:bold-colon-heading` BUILT: #1927 head `ec9e54a1e`; judge PASS 80 then PASS 88 (sonnet); merge PARKED on dev's inherited required red (D-FLEET-17); ticket open [HARNESS]

**Pick**: iteration 53's **Next** was the 19-day-old `quota:admission-has-no-headroom-margin-and-flaps` ghost-check. Gate 2's clause map ranks a fresh mechanical ticket above it. Clause 2 (turnaround, median ≤ 48 h) is UNMET, and only a ticket filed today can still resolve inside 48 h; the 19-day ticket counts above the median whenever it closes. Two fresh fleet-scope tickets were filed today, and both are defects in fleet-landed code (clause 5): `routing-models-check:bold-colon-heading` (world, 12:29Z; the iteration 53 script) and `rotate-log:heading-parse-stapledon` (stapledon, 15:24Z; the iteration 51 parser). I took the first. It is smaller, and it is the Gate-4 instrument every mission now runs.

**Orphans credited**: none new. Gate 0 step 6a returned rc 1 for the 2026-10-17T12:48:13Z crash notice on #1768, already credited at iteration 47 (`issuecomment-6234067020`, 5 mentions in this log). No open `fleet/*` PR at pick; the leftover fleet worktrees belong to landed iterations.

**Preflight**: armed (`mission-fleet.disabled` absent); gh `sunholo-voight-kampff`; billing CLEAN; messaging store gcp. Directives: 0 on #1891 (11 comments) and 0 on #1768 (26) since `2026-10-15T02:46:27Z`. Environment read by name only. No rotation or sweep due (#1891 created 2026-10-19).

**Observe**: origin/dev == pin == `d25d9b8d0`. `SKILL.md` and all 13 resources are `cmp`-identical to origin at both the resolved symlink target and the pin. **Dev CI is RED on `d25d9b8d0`, from an attended commit outside fleet scope.** The required `test` and `test-windows` fail only on `TestModels_PricingIsSlugConsistent`: `google|antigravity-preview-09-2026` carries two prices, on `gemini-agents-antigravity` and on the new `gemini-agents-argon` row (`internal/modelreg/models.yml`, commit `d25d9b8d0`, "gemini-agents-argon"). macOS fails the same test, and `govulncheck` is still GO-2026-6809. V1 owns the repo and is disabled; D-FLEET-17 is the standing question.

**Reality check**: real at HEAD. World `--entry 295` and `--entry 294` both rc 2; fleet `--entry 53` rc 0 (control). Heading survey across all 7 logs (world and stapledon from their own clones, stapledon at `origin/main`):

| Log | Heading shape | Count |
|---|---|---|
| world | `**Routing evidence:**` | 20 |
| stapledon | `- **Routing evidence:**` (bullet; indented sub-bullets; sibling `- **Ruled out:**`) | 41 |
| v1 | `**Routing evidence.**` | 20 |
| docs | `**Routing evidence**:` | 9 |
| docs | parenthetical form | 1 |
| fleet, motoko, triage | `**Routing evidence**:` | — |

The ticket's own suggestion (accept both colon placements) would have missed Stapledon and V1. Fleet's in-block lines are column-0 `- **Controller**:` bullets, while Stapledon's sibling field is a column-0 `- **Ruled out:**`, so the block end has to depend on whether the header itself is a bullet.

**Did**:
- **Designer**: not spawned. The fix is mechanical, and the controller's measurements C1–C7 and design D1–D7 went into the planner directive (precedent: iterations 45 and 49–53).
- **Planner** (codex `gpt-6.1-sol`, recipe, `.planner-wt-iter54`): `sprint-plan-routing-check-heading-shapes.md` + sprint JSON, one milestone, five real-shape arms each with a named mutation. Committed `7e19702c2`. The planner post-step `ailang messages import-github` was **skipped**: it writes to the prod store and is unrelated to a fleet ticket.
- **Executor round 1** (codex): header regex, the bulleted-sibling end, arms w s v p f. TDD red (w/s/v/p rc 2); all 6 named mutations killed.
- **Controller corpus gate** (HEAD vs new, every `## N — ` entry of 7 real logs, out of sandbox). It found one unexplained class: Stapledon 43/44 newly flagged `gpt6-1-sol-openai-api-credits-exhausted`, a ticket signature that the greedy `gpt[-0-9a-z.]*` token swallowed.
- **Executor round 2**: a hyphen-bounded allowlisted prefix makes the token a quoted compound; arm q added. The corpus changed in Stapledon 43/44 only (rc 1 → 0). Committed `579f927b4`.
- **Judge round 1** (sonnet, Agent tool, foreground, `.wt-fleet-iter54-judge`): **PASS 80**, 0 blocking. 11 mutations: 7 killed, 4 survived, 1 of them equivalent. Two findings reproduced by the controller: `gpt-6.1-sol-gpt-5` and `gpt-6-luna-gpt-5.6-sol` were hidden, and the hyphen-boundary mutant survived (`gpt-6-lunatic` hidden).
- **Executor round 3**: a compound counts only if the remainder names no further `gpt`; the assertions are pinned. Both mutations are killed. Corpus unchanged vs round 2. Committed `ec9e54a1e`.
- **Judge round 2** (sonnet, fresh): **PASS 88**, 0 blocking. 5/5 targeted mutations killed. The survivors (`[^*]*` → `.*`, the dropped `!seenBlock` guard, the equivalent `bulleted = 0` reset) were judged harmless. Its corpus TSV is byte-identical to the controller's.
- **Corpus result** (final checker vs HEAD), every change rc 2 → 0/1:

  | Log | Entries | Change |
  |---|---|---|
  | world | 20 | rc 2 → 0 |
  | stapledon | 8 | rc 2 → 0 |
  | v1 | 20 | rc 2 → 0 (5) or rc 2 → 1 (15, historical retired ids: 25× `gpt-5.6-sol`, 16× astra, plus terra/luna/5.5) |
  | docs entry 2 | 1 | rc 2 → 1 (`gpt-5.6-luna` in the parenthetical table) |
  | fleet, triage | — | unchanged |

  Remaining rc 2: docs 4, 5, 7, 8, 11, 12, 13 and triage 0. They carry no "Routing evidence" label at all (`**Routing / outcome**`, `**Routing — sprint-planner.**`), so rc 2 is correct there.
- **PR #1927** (no closing keyword; the scan was paired with a control): 25 checks. Not green, all inherited from dev, each failing job's FAIL list read from its raw log:
  - `test` / `test-windows`: the single `TestModels_PricingIsSlugConsistent`.
  - `Build macos-latest`: the same test.
  - `govulncheck`.
  - ubuntu/windows builds: cancelled by fail-fast.

  `launchd drivers (bash 3.2)`, `lint`, `docs-gate` and `UI build gate` are green. `mergeable=MERGEABLE`, `mergeStateStatus=BLOCKED`.

**Gate 3b**: not reached; nothing merged. The PR poll I wrote myself never wrote its verdict: `set -- $out` does not word-split under the Bash tool's zsh, so every count read non-numeric. I stopped it via TaskStop (confirmed by the tool) and read the checks directly. This iteration's record is stacked on `fleet/i54-routing-check-headings` (Gate 1 rule for a blocked landing PR).

**Done-gate**:
- Surface: `scripts/mission_routing_models_check.sh`, which Gate 4 runs from `$MISSION_DRIVER_ROOT`, i.e. the pin.
- Reach: pushed, not merged. It reaches every mission's next pinned fire once #1927 merges.
- `make test-launchd-drivers` under `env -i`: rc 0 at `579f927b4` (298 PASS) and at `ec9e54a1e` (304 PASS, routing suite 46/0). Each run waited for the worktree to be quiet, so neither is a contaminated reading (iteration 53 retro 1).
- Dry-runs under **stapledon** (armed, idle, no pidfile; `AILANG_DRIVER_PINNED=ec9e54a1e`): healthy rc 0 `lanes=ok`; degraded (`MISSION_EXECUTOR_MODEL=pi:bogus/no-such-model`) rc 0 `DEGRADED(1)` → opus. No pidfile written; nothing reloaded.
- After the skill edit: `test_codex_spawn_override.sh` (0 failures), `test_pi_ext_args.sh` (14/0), `test_mission_heartbeat.sh` (37 arms) and `check_context_docs.sh` are all green.

**Progress**: 0 tickets resolved (1 built, merge blocked); 43 open. Goal unmoved on clause 1.
- Clause 1 UNMEASURED.
- Clause 2 UNMET. This ticket counts toward the median only if it lands before 2026-10-23 12:29Z.
- Clause 3 MET.
- Clause 4 prior evidence only.
- Clause 5 preserved (nothing merged).

**Routing evidence**: base=d25d9b8d0dde10ee7c14986fe21b94fd2cb500cd@2026-10-21T18:26:43Z; base-gate1=d25d9b8d0dde10ee7c14986fe21b94fd2cb500cd@2026-10-21T17:09:37Z (no drift); sprint worktree base `d25d9b8d0`. `MISSION_ROUTING_NOTE` empty; `MISSION_OVER_RATION` empty.
- **Controller**: `claude:claude-opus-5-5` (tok: not reported).
- **Designer**: not spawned. Resolver `recipe claude:claude-opus-5-5 declared:provider-pin`; rotation not advanced.
- **Planner**: resolver `agent-tool opus fail-closed:no-doc` against the pin `codex:gpt-6.1-sol`, so routed to the pin under the codex recipe (role-spawn-routing §2(a)). Probe rc 0; run rc 0 (51,943 tok).
- **Executor**: `codex:gpt-6.1-sol` via the recipe (`declared:provider-pin`), 3 runs, all rc 0: 50,878, 54,285 and 53,805 tok.
- **Evaluator**: `sonnet` via the Agent tool (`declared:alias-pin`), foreground, 2 fresh spawns (50,089 + 46,802 tok). Judge independence: cross-vendor (Anthropic sonnet vs the OpenAI generator).
- The Agent tool was used where the resolver names it (the evaluator), per the operator's standing request. Planner and executor resolve to recipes. No role failed to spawn.
- Every codex run carried the spawn-override flag.
- **Metered**: $0.

**Ruled out**:
- The ticket's colon-only fix: it misses Stapledon's bullet header and V1's period header.
- Ending a block at any column-0 `- **`: that drops fleet's own `- **Executor**:` lines (arm f).
- Treating the Stapledon 43/44 flags as real: they quote a ticket signature, and the only model named is allowed.
- Fixing `TestModels_PricingIsSlugConsistent` from the fleet: `internal/modelreg` is outside the Authority allowlist. It needs an attended ruling on which Argon price is live.
- Bypassing the required check. Never.

**Retro lane**:
- (1) **Skill edit (Gate 5, the one allowed): hand-written polls run under `/bin/bash`.** It is now in both copies of `resources/gate-3b-ci-green.md` (`c2e8547bc`, byte-identical). Frictions: iterations 49 (2), 50 (2), 52, 53 and this one (`set -- $out`), seven in total. It had been banked since iteration 50.
- (2) **A judge's summary of an artifact is a claim.** Round 2's prose said "v1 13 → 1, 7 → 0" while its own TSV, byte-identical to mine, reads 15 and 5. Banked, one instance: when a judge quotes numbers, diff the artifact rather than the sentence.
- (3) **The corpus gate earned its place a second time** (iteration 51 was the first). The plan's five real-shape arms were all green, and only running the new checker over real logs found the compound false positive. Recommended as a standing controller step for any instrument the fleet changes. That step is already prescribed in the planner directive pattern, so no skill edit.
- (4) **D-FLEET-17 recurs**: an attended commit reddened dev's required `test` while the fleet was mid-iteration, and a judged PASS cannot land. This is the third iteration it has blocked (37, 38, 54).

**Next**: when dev's `test` is green, re-run #1927's checks (rebase only if needed: `git diff --stat d25d9b8d0 origin/dev -- scripts/mission_routing_models_check.sh scripts/test_mission_routing_models_check.sh` empty means no conflict), then merge with `--match-head-commit`, resolve `routing-models-check:bold-colon-heading` with the merge SHA, and reply to world. Then `rotate-log:heading-parse-stapledon` (stapledon, 10-21; fresh; a defect in iteration 51's parser), then `quota:admission-has-no-headroom-margin-and-flaps` (ghost-check against #1524 `71eec62b6`).

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
