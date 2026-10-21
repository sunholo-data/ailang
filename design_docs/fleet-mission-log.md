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
