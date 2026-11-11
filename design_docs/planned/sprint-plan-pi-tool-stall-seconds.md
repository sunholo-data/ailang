# Sprint plan: `--tool-stall-seconds` for the pi runner (fleet iteration 105)

Ticket `tool-hang:pi-evaluator:stall-bound-below-single-test-run` (stapledon iter 86, 2026-11-11).
Base `cdf91c5a6`, branch `fleet/i105-tool-stall-seconds`. Two milestones, ~+40 LOC runner, ~+130 LOC test.

## Premise check (read at cdf91c5a6)

- TRUE: one `STALL_SECONDS` (`scripts/mission_pi_run.sh:89`, default 600) is the only no-progress
  bound (`:355`); progress = mtime of filtered `--out` (`:349`); `stream_dead` is relabelled `tool_hang`
  rc 18 post-loop when `tool_execution_start` count > `tool_execution_end` count (`:412-419`).
- FALSE: "validate the value like the others". `--max-seconds`, `--stall-seconds` and
  `MISSION_PI_POLL_SECONDS` are NOT validated anywhere; `abc` silently becomes 0 in `$(( ))` and an
  `[ -ge ]` error elsewhere. The new knob gets validation; the existing knobs stay as they are (out of
  scope, file as a follow-up ticket, because rejecting a value that is accepted today changes behaviour).
- `.pi/extensions/.session-protocol-gate.test.ts:38-57` (policy class, not editable) pins the text
  between `PI_EVALUATOR_SESSION_HANDSHAKE_START/END` in `gate-3-route.md`. Skill edits go OUTSIDE those
  markers.
- `.agents/skills/mission-control/resources/gate-3-route.md` is a byte-identical mirror; it is enforced
  by `tools/launchd/test_agents_skills_sync.sh` (runs `sync-agents-skills.sh --check`, in
  `LAUNCHD_SUITES`). The pre-push guard allows `.agents/skills/mission-*` (`tools/launchd/githooks/pre-push:18`).
- `make test-launchd-drivers` exists (`make/test.mk:134`). It syntax-checks every `tools/launchd/*.sh`
  automatically but RUNS a suite only if it is listed in `LAUNCHD_SUITES` (`make/test.mk:75-130`).
- **No default changes.** `STALL` 600, `MAX` 1800, `POLL` 10, `SNAP_EVERY` 50 are untouched; the new
  bound defaults to the effective stall bound, so a caller passing neither flag nor env gets the same
  kill time and verdict. The verdict JSON gains three additive keys (no consumer parses it strictly).
  Not policy class: no routing, lane order, quota, billing or `.pi/extensions/**` change.

## M1 — runner knob + test + make wiring

**Files:** `scripts/mission_pi_run.sh`, `tools/launchd/test_mission_pi_run_tool_stall.sh` (new),
`make/test.mk`.

Runner changes (keep the file ≤ ~520 lines):
1. `:91` add `TOOL_STALL_SECONDS="${MISSION_PI_TOOL_STALL_SECONDS:-}"` (empty = "not given").
2. Arg loop `:120`: `--tool-stall-seconds) TOOL_STALL_SECONDS="$2"; shift 2 ;;`.
3. AFTER `preflight_fail` is defined (`:142`, so the rc-14 verdict JSON lands at `$VERDICT`):
   `[ -n "$TOOL_STALL_SECONDS" ] || TOOL_STALL_SECONDS="$STALL_SECONDS"` then
   `case "$TOOL_STALL_SECONDS" in ''|*[!0-9]*|0*) preflight_fail 14 launch_failed "--tool-stall-seconds must be a positive integer: '$TOOL_STALL_SECONDS'";; esac`
   (`0*` also refuses leading zeros, which `$(( ))` reads as octal). The default MUST resolve after
   parsing, from the effective `STALL_SECONDS`, not from the env at init (arm 1 kills that mutant).
4. One helper, used by BOTH the loop and the post-loop relabel (one implementation of "a tool is open";
   replace the inline `_starts` count at `:413-414` with it):
   `tool_open() { local s e; s=$(grep -c '"type":"tool_execution_start"' "$OUT" 2>/dev/null); e=$(grep -c '"type":"tool_execution_end"' "$OUT" 2>/dev/null); [ "${s:-0}" -gt "${e:-0}" ]; }`
5. Poll clamp `:328`: clamp to a third of `MIN_BOUND = min(STALL_SECONDS, TOOL_STALL_SECONDS)`. With the
   default (equal bounds) it is the current clamp; a tool bound below the stall bound is allowed and
   must still be observed in time (arm 6).
6. Stall check `:355`: `IDLE=$((T-PROGRESS_AT))`; only when `IDLE -ge MIN_BOUND`, pick
   `BOUND=$TOOL_STALL_SECONDS` if `tool_open` else `$STALL_SECONDS` (grep runs only once a bound could
   fire, not every poll); if `IDLE -ge BOUND` record `STALL_BOUND_APPLIED=$BOUND` and keep the existing
   reasoning_stall / stream_dead split. `--max-seconds` check is unchanged and still bounds everything.
7. Verdict JSON: add `"stall_seconds"`, `"tool_stall_seconds"` (configured) and
   `"stall_bound_seconds"` (the bound that fired; `0` when the run did not end on a stall). The
   `tool_hang` stderr line adds `(no-progress bound ${STALL_BOUND_APPLIED}s)`.

Test `tools/launchd/test_mission_pi_run_tool_stall.sh`: `#!/bin/bash`, bash 3.2, copy the scaffold of
`test_mission_pi_run_commits.sh:1-47` (sandbox node_modules stub, `stub()`, `repo()`, `field()`),
`unset MISSION_PI_STALL_SECONDS MISSION_PI_TOOL_STALL_SECONDS` at top, `MISSION_PI_POLL_SECONDS=1`
except arm 6. Stubs write `$PI_SANDBOX_READY_FILE`; tool stubs emit
`{"type":"tool_execution_start","toolName":"bash","args":{"command":"make slow-test"}}`, `sleep N`,
write `f.txt`, emit `tool_execution_end` + `agent_end`. Each arm asserts rc, `verdict`, and
`stall_bound_seconds` where stated. `--max-seconds 30` unless noted.

| Arm | Flags / stub | Expect | Mutation only this arm kills |
|---|---|---|---|
| 1 default | `--stall-seconds 4`, no tool flag; tool sleeps 8 | rc 18 `tool_hang`, bound 4 | default resolved from env/600 at init instead of the effective `--stall-seconds` (mutant: rc 0 ok) |
| 2 raised | `--stall-seconds 4 --tool-stall-seconds 20`; tool sleeps 8 | rc 0 `ok` | flag parsed but loop still compares `STALL_SECONDS`; or `tool_open` always false (mutant: rc 18) |
| 3 model silence | `--stall-seconds 4 --tool-stall-seconds 20`; stub emits `{"type":"turn_start"}`, sleeps 12 silent, no tool | rc 12 `stream_dead`, bound 4 | loop uses `max(STALL,TOOL)` / TOOL unconditionally, i.e. the fix just widened the stall bound (mutant: rc 10 at 12 s) |
| 3b closed tool | as 3, but a completed start+end pair precedes the 12 s silence | rc 12 `stream_dead`, bound 4 | `tool_open` = "any start seen" instead of starts > ends (mutant: rc 10) |
| 4 hang past raised bound | `--stall-seconds 4 --tool-stall-seconds 6`; tool sleeps 15 | rc 18 `tool_hang`, bound 6, `hung_tool` names `make slow-test` | open tool exempt from any stall bound (mutant: rc 0 at 15 s); relabel dropped when TOOL bound fired (mutant: rc 12) |
| 5 bad value | `--tool-stall-seconds abc`, `0`, `-5`, `08`; and env `MISSION_PI_TOOL_STALL_SECONDS=x` | rc 14 `launch_failed`, verdict file exists | validation removed or placed before `VERDICT` is set (mutant: rc 0/18, or no verdict file) |
| 6 clamp | `MISSION_PI_POLL_SECONDS` unset (10), `--stall-seconds 60 --tool-stall-seconds 3 --max-seconds 60`; tool sleeps 30 | rc 18, `elapsed_seconds` ≤ 9 | clamp still on `STALL/3` only (mutant: first poll at 10 s, elapsed ~12) |

Arm 6 is the only timing assertion: correct run ≈ 3-4 s + 2 s kill sleep, mutant ≈ 12 s; both sides
keep ≥ 3 s margin. The executor runs the suite 3× under `make test-launchd-drivers`' parallel load
before claiming green. Each mutant is applied to a scratchpad copy of the runner (never
`git checkout <file>`), the suite is run against it via a `SUT` override, and the arm is shown red.

`make/test.mk`: add `tools/launchd/test_mission_pi_run_tool_stall.sh \` after
`tools/launchd/test_mission_pi_run_provider_quota.sh` (`:92`).

**Acceptance (from the worktree):**
```bash
/bin/bash -n scripts/mission_pi_run.sh && /bin/bash tools/launchd/test_mission_pi_run_tool_stall.sh
/bin/bash scripts/test_mission_pi_run.sh          # existing TEST 4b tool_hang pin stays green (default unchanged)
/bin/bash tools/launchd/test_mission_pi_run_commits.sh && /bin/bash tools/launchd/test_mission_pi_run_sandbox.sh \
  && /bin/bash tools/launchd/test_mission_pi_run_provider_quota.sh
make test-launchd-drivers                         # unpiped; runs the new suite via LAUNCHD_SUITES
wc -l scripts/mission_pi_run.sh                   # ≤ ~520
bash /tmp/fleet_i105/repro.sh                     # pre-fix shape still rc 18 with no new flag
```

## M2 — skill text + usage + changelog

**Files:** `scripts/mission_pi_run.sh` (usage only), `.claude/skills/mission-control/resources/gate-3-route.md`,
its mirror `.agents/skills/mission-control/resources/gate-3-route.md` (regenerated, never hand-edited),
`changelogs/unreleased/2026-11-11-pi-tool-stall-seconds.md` (new).

1. Usage (`:96`, `:104`): add `[--tool-stall-seconds N]` and
   `--tool-stall-seconds no-progress cap while a tool call is OPEN (default = --stall-seconds, env MISSION_PI_TOOL_STALL_SECONDS)`,
   plus one sentence: use it for a named long command; raising `--stall-seconds` also blinds the
   reasoning_stall/stream_dead detection.
2. `gate-3-route.md:431`: after "still dies in ten minutes." append: "A named long gate command is not
   a stall: give it `--tool-stall-seconds` per (d), never a raised `--stall-seconds`."
3. New rule **(d)** after (c) (ends ~`:437`), ≤ 6 lines: when a pi role (executor or evaluator) is
   directed to run a named long gate command (full test target, Godot run), pass
   `--tool-stall-seconds` = 2 × that command's longest duration MEASURED this iteration on this host
   (3× if measured at markedly lower host load than the run will see), never below 600, and size
   `--max-seconds` above it plus the role's other work. Cite verification rule 3m (derive the bound
   from the measured stimulus). Record the measurement and the derived value in the log. Leave
   `--stall-seconds` at its default. Evidence: stapledon iter 86, measured 396-573 s at load 12-25,
   killed at the 600 s bound at load 30-60.
4. `tool_hang` fallback paragraph (`:476-480`): add "Exception: if `hung_tool` is a gate command the
   directive told the role to run, the retry passes `--tool-stall-seconds` per (d) instead of forbidding
   the command; `stall_bound_seconds` in the verdict shows which bound fired."
5. One line AFTER `<!-- PI_EVALUATOR_SESSION_HANDSHAKE_END -->` (`:521`, outside the pinned block):
   "A pi evaluator that must run a long named gate gets `--tool-stall-seconds` per (d)."
6. `/bin/bash tools/launchd/sync-agents-skills.sh` to regenerate the mirror.
7. Changelog fragment, one section:
   `### Added — pi runner: --tool-stall-seconds bounds an open tool call separately (2026-11-11)`, body
   covering the ticket, the default-preserving behaviour, the new verdict keys and the skill rule.

**Acceptance:**
```bash
/bin/bash tools/launchd/sync-agents-skills.sh --check
/bin/bash tools/launchd/test_agents_skills_sync.sh
scripts/mission_pi_run.sh --help 2>&1 | grep -- --tool-stall-seconds
grep -c 'tool-stall-seconds' .claude/skills/mission-control/resources/gate-3-route.md   # ≥ 4
make check-changelog                              # unpiped
git diff --stat cdf91c5a6 -- . ':!scripts/mission_*' ':!tools/launchd' ':!.claude/skills/mission-*' \
  ':!.agents/skills/mission-*' ':!make/test.mk' ':!design_docs' ':!changelogs'          # must be empty
```
`.pi/extensions/.session-protocol-gate.test.ts` is not run in the bash-3.2 job; the executor confirms
by `git diff` that no line between the handshake markers changed.

| Mutation (M2) | Killed by |
|---|---|
| edit only `.claude`, skip the mirror | `test_agents_skills_sync.sh` |
| fragment misnamed or headed `##` | `make check-changelog` |
| rule text placed inside the handshake markers | marker-span `git diff` check above |

## Done-gate (controller)

`make test-launchd-drivers` green (unpiped), then the driver dry-runs healthy and degraded under an
idle armed sibling with `AILANG_DRIVER_PINNED=<sha>` (`MISSION_PROFILE=<m> MISSION_DRY_RUN=1`, and the
same with `MISSION_EXECUTOR_MODEL=codex:bogus`; `mission-control.sh:2065-2067`), per
`design_docs/fleet-mission.md` Done-gate. The driver does not call the runner, so the dry-runs show the
change is inert to the loop. Follow-up ticket: validate `--max-seconds` / `--stall-seconds` /
`MISSION_PI_POLL_SECONDS` (the premise the controller assumed exists).
