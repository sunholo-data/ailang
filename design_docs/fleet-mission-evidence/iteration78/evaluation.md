# Iteration 78 — independent evaluation (sonnet, Agent tool, foreground; generator = codex gpt-6.1-sol)

Two passes by the same evaluator instance (agent a5015731526dc8139), read-only on the worktree.

## Pass 1 — PASS 94/100 (on the executor's diff, including its test preamble)
- Suite 146/0 in a clean env and with `MISSION_ANTHROPIC_AVAILABLE=0`.
- Byte-identity grid, base vs patched resolver (stdout, stderr, rc) under /bin/bash 3.2.57:
  5 roles × 8 pins × 6 ANTH values (unset, 1, empty, 2, 00, true) × 5 fallback chains ×
  3 MISSION_OVER_RATION values × 4 executor values = **14,400 combinations, 0 differ**.
- Mutants: m1 delete block (18 fail), m2 drop claude: skip, m3 drop ration skip, m4 drop evaluator
  family skip, m5 `= "0"` → `!= "1"`, m6 drop trim, own2 drop family clause, own3 stop skipping bare
  aliases, own4 read the evaluator chain for every role, own5 remove final refuse — all killed.
  own1 (drop the exact-equality clause) survives as an EQUIVALENT mutant: `family` of a non-Claude
  value is the value itself.
- An evaluator instrument slip, self-caught: a GNU-only `sed 0,/re/` produced a false m4 "survivor"
  on BSD sed; redone by line number, killed.
- Consumers: `spawn-pin-hook.sh` never calls the resolver nor reads MISSION_ANTHROPIC_AVAILABLE; no
  Claude-controller path sets ANTH=0 (`mission-control.sh:988-991`).
- Findings (all NON-BLOCKING): F1 doc sentence accurate but incomplete (no evaluator family skip) and
  wedged into one long line (docs 6/10); F2 the exact-equality clause is redundant; F3 no planner arm
  under ANTH=0 (out of scope; grid shows planner unchanged).

## Controller finding between passes
`make test-launchd-drivers` rc=2 on three arms. (1) `test_suite_env.sh` arm E: it runs
`test_mission_routing.sh` DIRECTLY in a poisoned env and expects a FAIL, to prove the `suite-env.sh`
launcher's cleaning is non-vacuous. The executor's hermetic preamble, which the CONTROLLER's
directive asked for, made that control pass. The env-sensitivity the controller measured at base
(clean 122/0, ANTH=0 113/9, live fleet env 121/1) is by design: every make-target suite runs through
the launcher (arm F). Preamble removed. (2)+(3) `test_agents_skills_sync.sh` and
`test_quorum_artifacts.sh generated_docs_and_mirror`: the `.agents/` mirror was stale after the
gate-3-route.md edit; `tools/launchd/sync-agents-skills.sh` run. Doc sentence rewritten per F1;
changelog rewritten. Re-run: `make test-launchd-drivers` rc=0.

## Pass 2 — PASS 96/100 (on the final delta)
- Diff = R15+ arms only in the test file; launcher run 146/0; `env -i` 146/0.
- Poisoned direct run (MISSION_EXECUTOR_RESOLVED, MISSION_OVER_RATION="openrouter ollama codex",
  ANTH=0, all fallbacks `x:y`): 129/17, every failing arm pre-existing (planner, D-FLEET-1, R1–R6,
  arm12); no R15–R38 arm failed, so the new arms own their env.
- `test_suite_env.sh` 8/0 (arm E passes). Mirror `--check` rc=0. Doc paragraph matches the code.
- Findings (NON-BLOCKING): the following `refuse …` sentence runs on in the same line; F2 carried over.

## After evaluation
The diff judged at PASS 96 was rebased unchanged onto the attended ruling (#2051) and its commit
message re-worded (`b437d6f89`). The resolver, tests, docs and mirror are byte-identical to what was
judged. Merged as `3d9112bb9`.
