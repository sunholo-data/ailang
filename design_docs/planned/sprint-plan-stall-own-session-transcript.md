# Sprint Plan: M-FLEET-STALL-OWN-TRANSCRIPT

## Summary
Fix `stall:watchdog:foreground-agent-and-wrong-transcript`: measure the controller's own session and its foreground Agent-tool subagents, preventing false gate-3 STALL kills.
Fleet iteration 55; base `origin/dev` d25d9b8d0; ticket from world iteration 298 (2026-10-21), after iterations 296/297 lost slots.
**Status:** planned; planning only. This plan is also the design artifact; no separate design doc.
**Duration:** one short executor run, 60–90 minutes within one day. **Risk:** low, with Bash 3.2 and fail-open semantics requiring care.
**Dependencies:** existing watchdog helpers/test extraction only; controller owns commits and outside-sandbox acceptance runs.

## Current Status Analysis
Controller measurements C1–C7 are established outside the sandbox and must not be repeated by the planner.
- C1: newest-file discovery reads other sessions sharing the checkout; iter-297 reported 318 MB while its own transcript was about 1 MB. pi has the same defect.
- C2/C5: claude has no launch session id and retries start fresh; pi already has a per-fire session id and resumes it.
- C3: Claude 2.1.307 accepts `--session-id`; UUID filename case is preserved; physical cwd slug can differ from `$PWD` (`/tmp` → `/private/tmp`).
- C4: foreground Agent work grows `<sid>/subagents/agent-*.jsonl` while the parent transcript remains flat.
- C6: `_mc_progress_bytes` is the only relevant Claude-project reader; fix both provider arms there.
- C7: established base `/bin/bash tools/launchd/test_mission_stall.sh`: 35 passed, 0 failed; wired into `make test-launchd-drivers`.
**Capacity:** 120 LOC target (80–160 including tests): ~40 driver + ~75 tests + ~5 changelog. Estimate follows the bounded ticket, not an invented historical velocity.
**Remaining:** one milestone; no implementation performed during planning.

## Registry Reuse Audit
`M1_STALL_OWN_TRANSCRIPT`: **none**. Internal shell instrumentation using existing watchdog/test helpers; no package-like capability, external dependency or `.ail` code. Registry search and AILANG syntax/example gates are inapplicable.

## Proposed Milestone
### M1_STALL_OWN_TRANSCRIPT — Own-session progress and foreground subagents
**Goal:** apply fixed design D1–D7 without changing watchdog policy.
**Estimate:** 120 LOC total; one day maximum, one 60–90 minute run; no milestone dependencies.

**Files to update/create (exhaustive execution scope):**
- `tools/launchd/mission-control.sh`
- `tools/launchd/test_mission_stall.sh`
- `changelogs/unreleased/2026-10-22-stall-watchdog-own-transcript.md` (create; opens with `### Fixed`)
- The two planning artifacts: this file and `.ailang/state/sprints/sprint_M-FLEET-STALL-OWN-TRANSCRIPT.json`.
No changes under `internal/`, `cmd/`, `.pi/`, CI workflows, make wiring or other files. No new example module; isolated shell fixtures demonstrate the behavior.

**Day 1 tasks (sequential):**
1. Update the existing fake-HOME harness to set `MC_CLAUDE_SESSION_ID` and place its transcript at `$MC_CLAUDE_SESSION_ID.jsonl`; set `MC_PI_SESSION_ID` for existing pi coverage. Preserve all 35 existing arms and their intent; rename obsolete cwd-slug descriptions to own-session discovery.
2. Add a1–a7 below against extracted production functions, isolating fixture/provider/id state between arms. Establish deterministic newer-file ordering in decoy fixtures. Name the mutation killed by each arm.
3. D1: inside `_mc_run_once`, for `CONTROLLER_PROVIDER=claude` only, generate a fresh `MC_CLAUDE_SESSION_ID` on EVERY attempt, before launching claude and before watchdog subshells fork. Try `/usr/bin/uuidgen`, then `python3 -c 'import uuid; print(uuid.uuid4())'`; if both fail clear the id and continue. Keep generated case verbatim; never reuse a prior attempt's id. Log one line naming the id (or empty/unavailable state).
4. Pass `--session-id "$MC_CLAUDE_SESSION_ID"` only when non-empty. Use a Bash 3.2-safe array expansion or conditional; no unquoted empty word, and retain the existing launch arguments/redirection. pi's id remains per fire.
5. D2/D3: in `_mc_progress_bytes`'s claude arm, sum the single matching `$HOME/.claude/projects/*/"$MC_CLAUDE_SESSION_ID".jsonl` and every `$HOME/.claude/projects/*/"$MC_CLAUDE_SESSION_ID"/subagents/*.jsonl`. Guard unmatched globs/missing files; they contribute nothing, including before the first transcript write. Preserve UUID case. Do not derive a Claude directory from `$PWD`, use newest-file discovery, or include `.meta.json` files.
6. If Claude's id is unset OR empty, contribute no transcript bytes and return non-zero from `_mc_progress_bytes`, EVEN WHEN `LOG` exists. Use a claude-specific guard; the unchanged driver-log arm alone must never make this case return 0. `_mc_stalled` must take its existing `no-progress-instrument` fail-open path, warn once and disable early kill; HARD_TIMEOUT still bounds the slot.
7. D4: keep pi's current directory rule, but sum only `"$dir"/*_"$MC_PI_SESSION_ID".jsonl` with a non-empty id and file guards. Unset/empty pi id contributes no transcript bytes; preserve pi's current driver-log contribution and return-code semantics. Neither provider arm uses `ls -t`/newest discovery.
8. D5: update the comment block above `_mc_progress_bytes`, citing C1 shared-session contamination, C3 physical-cwd/case behavior, and C4 foreground subagents. Keep the driver-log arm, thresholds, sample counts, `STALL_*`, rusage, tree-write and heartbeat arms unchanged.
9. Add the scoped `### Fixed` changelog fragment. Controller runs the acceptance commands below outside the sandbox and records results.

**Required regression arms and mutation targets:**
| Arm | Observation/assertion with other progress signals flat | Mutation killed |
| --- | --- | --- |
| a1 wrong-transcript | Own transcript flat; different newer same-project `.jsonl` grows; `_mc_stalled` reports stalled. | Revert Claude to `ls -t \| head -1`. |
| a2 subagent progress | Own transcript flat; `<sid>/subagents/agent-x.jsonl` grows; reports live. | Drop the subagent fold. |
| a3 other session's subagent | Other session's `subagents/*.jsonl` grows; own flat; reports stalled. | Use an unanchored `*/subagents` glob. |
| a4 physical-path slug | Own transcript under a project slug differing from `$PWD` (e.g. `-private-…`) grows; reports live. | Re-derive Claude's directory from `$PWD`. |
| a5 unset/empty Claude id | Test both unset and empty; driver log exists; `_mc_progress_bytes` non-zero; `_mc_stalled` live with `no-progress-instrument`, warning once. | Newest-file fallback OR letting LOG alone return 0 for claude. |
| a6 pi wrong-transcript | Own `*_${MC_PI_SESSION_ID}.jsonl` flat; different newer pi file grows; transcript bytes unchanged. Also check unset/empty id adds nothing while LOG retains existing rc semantics. | Restore pi newest-file discovery. |
| a7 launch wiring | Extract `_mc_run_once` using `^_mc_run_once() {` through `^}$` awk boundaries; require non-empty extraction; grep within it for Claude launch `--session-id` wiring and `MC_CLAUDE_SESSION_ID` assignment. | Remove launch flag OR move/remove per-attempt assignment. |
Use fixtures only under the suite's own `$TMP`, including fake HOME and any `PI_CODING_AGENT_DIR`/`AILANG_STATE_DIR`; retain extraction of real driver functions rather than duplicating them. Static a7 must associate the flag with the Claude launch, not accept pi's existing flag.

**Acceptance Criteria:**
- [ ] D1–D5 behavior above is present; fresh per-attempt Claude id precedes launch/watchdog forks, optional flag is Bash 3.2-safe, generation failure clears the id and fails open.
- [ ] Only own parent/subagent JSONL bytes count for Claude, across project slugs and with verbatim UUID case; missing files add nothing. Claude unset/empty id returns non-zero despite an existing driver log; pi preserves its log/rc semantics.
- [ ] All 35 established arms remain meaningful; a1–a7 pass, named with mutation targets; total ≥42 passing arms, zero failures.
- [ ] Observation: all test-created files are under the suite's `$TMP`; no test reads or writes the real `~/.claude` or `~/.ailang/state`, and no seeded writes to real paths are used as evidence.
- [ ] Driver-log arm and all D5 policy/other progress arms unchanged; scoped changelog opens with `### Fixed`; only the five allowed paths changed.
- [ ] All four controller acceptance commands below pass outside the sandbox.

**Risks/mitigation:** Bash 3.2 empty-array/set-u behavior → guarded expansion/conditional; unmatched globs → per-file existence checks; fixture contamination → reset ids/provider/signals and isolate decoys. Duplicate Claude parent matches violate the established single-match expectation; do not broaden discovery to unrelated sessions.

## Validation and Success Metrics
Controller re-runs outside the sandbox; planning makes no new driver-gate claims:
```bash
/bin/bash tools/launchd/test_mission_stall.sh
bash -n tools/launchd/mission-control.sh
/bin/bash -n tools/launchd/test_mission_stall.sh
make test-launchd-drivers
```
Expected: ≥42 arms, zero failures; both syntax checks rc0; fleet done-gate `make test-launchd-drivers` rc0 (outside sandbox only). No full-suite requirement.
Any socket- or HOME-touching in-sandbox result is **UNINFORMATIVE UNDER SANDBOX**. C1–C7 remain controller evidence, not planner measurements.
Planning validation: `jq -e .` parses the JSON; its key sets match the reference sprint schema, status is `planned`, and exactly one milestone is initialized unstarted.

## Dependencies and Open Questions
No open design questions: D1–D7 are fixed. Controller handles artifact commits and subsequent execution; this planning stage performs no implementation, git writes or executor dispatch.
