# Sprint plan: reap descendants when a fleet slot is killed

**Ticket:** `driver:slot-kill-leaves-orphan-descendants` (P0)  
**Scope:** mechanical launchd driver fix  
**Milestone:** M1, about half a day

## Goal and evidence

When either watchdog kills a controller attempt, terminate its whole process tree, including tools and harnesses started by the controller. Mission-world observed a stall kill at 04:45 on 2026-09-26 (rc 143), yet the planner's 46-row mutation harness ran until 05:11. The current driver starts `claude`, `codex`, or a `pi` subshell in the background and records `$!` as `CONTROLLER_PID` (`tools/launchd/mission-control.sh:2251-2299`). Its HARD_TIMEOUT and stall branches signal only that PID (`:2303-2306`, `:2319-2324`). In the `pi` branch `$!` names the subshell, so even `pi` can outlive it. The existing `_mc_descendants` recursively walks `pgrep -P` (`:333-338`), but loses a child once its parent exits and launchd reparents it.

## Files to change during execution

- `tools/launchd/mission-control.sh`: add one `_mc_kill_tree PID GRACE` helper near `_mc_descendants`; call it from both watchdogs with their existing grace values (60 seconds hard, 30 seconds stall). Ensure `_mc_run_once` lets an already-triggered watchdog complete its reap before it cancels watchdog jobs and returns the controller's existing RC. Do not make the non-triggered watchdog delay normal completion.
- `tools/launchd/test_mission_kill_tree.sh` (new): extract real driver functions with `awk '/^fn\(\) \{/,/^\}$/'`, as `test_mission_stall.sh` does. Run real small shell process trees and the structural driver assertions below. Every fixture file and log goes in one `mktemp -d` directory; no `$HOME` or `.ailang/state` writes.
- `make/test.mk`: invoke the new test with `/bin/bash` in `test-launchd-drivers`.

All execution edits stay under the fleet charter's allowed paths. No separate design or release-note edit is needed for this mechanical correction.

## M1 — tree reap and regression gate

1. Snapshot `_mc_descendants "$root"` **before any TERM**. TERM every captured live PID, preferably leaves before the root so the driver can still observe the attempt. Retain the captured PIDs after reparenting. Keep PID input numeric and avoid signalling the driver's own PID or unrelated processes.
2. Wait the caller's **existing** grace (60 or 30 seconds). Before KILL, re-walk from every surviving captured PID to collect descendants born during grace. KILL the remaining live members of that union. Suppress expected `kill`/`pgrep` failures for already-exited processes, while returning a useful result if the root was invalid. The helper uses Bash 3.2 syntax only.
3. Call the helper at both watchdog sites without changing watchdog thresholds, sample counts, grace values, log messages, retry rules, or RC semantics. A triggered watchdog must finish the helper after `wait "$CONTROLLER_PID"` returns; use a per-attempt trigger marker written **before** the first TERM and wait for that watchdog before canceling it. Keep the marker in the driver's existing per-attempt slot-state area. Handle both watchdogs triggering close together without killing an active reaper. The marker is synchronization only; do not let it alter the watchdog decision predicate.
4. Add the real-process test. Generate parent and child `sh` scripts inside `mktemp -d`; parent backgrounds child, child backgrounds grandchild, and each stays alive with a `sleep` loop. Save all PIDs in that directory and use a `date +%s` deadline for fixture startup, helper completion, death polling, and cleanup. Include a child that ignores/traps TERM and a TERM-triggered child that starts a late grandchild during grace. Assert each tree PID is gone (or reaped zombie) by the deadline, while an unrelated sibling process remains alive. Check the extracted helper and `_mc_descendants` are nonempty, and assert that **both** watchdog kill sites in `_mc_run_once` call `_mc_kill_tree` with the original 60/30 values. The assertion must red if either site reverts to direct `kill`.

### Acceptance criteria (runnable)

From the repository root after execution:

```sh
/bin/bash -n tools/launchd/mission-control.sh
/bin/bash -n tools/launchd/test_mission_kill_tree.sh
/bin/bash tools/launchd/test_mission_kill_tree.sh
make test-launchd-drivers
git diff --check
git diff --name-only
```

The new test reports a positive fixture count for the root, child, grandchild, late descendant, and sibling; it fails on a missed PID or a killed sibling. `git diff --name-only` must contain only the three execution paths above (plus this approved plan and sprint JSON if the executor starts in this planning worktree). Test cleanup also obeys a bounded deadline; a failed assertion must not leave fixture processes running. The suite's shell is `/bin/bash` 3.2.57.

### Mutation table

| Test arm | Mutation it must kill |
| --- | --- |
| Parent → child → grandchild all dead | Helper signals only the root PID; descendants survive reparented. |
| TERM-resistant child and grandchild dead after grace | No KILL escalation, or KILL only the root. |
| Grandchild created by a surviving child during grace dead | No re-walk from surviving snapshot PIDs before KILL. |
| Unrelated sibling alive | Broad `pkill`, wrong process group, or signalling beyond the captured tree. |
| Both watchdog call sites use helper with 60/30 | Revert either watchdog to direct one-PID `kill`, or change a grace period. |
| Triggered watchdog completes after controller exits | Parent cancels an active reaper immediately after `wait "$CONTROLLER_PID"`, leaving a TERM-resistant descendant alive. |

Run the mutation arms by temporary source copies under `mktemp -d`, restoring the original source without git write operations; each mutation must be shown to apply, make its named arm red, and leave the clean test green again.

## Decisions, exclusions, and risks

Use the PID snapshot helper, **not a new process group**. Noninteractive Bash does not provide a dedicated group for each background job by default; enabling `set -m` for this driver changes job control and wait/signal behavior across the loop. A captured tree is smaller, fits the existing `_mc_descendants` mechanism, and is directly testable on the rig.

Normal controller exit with lingering children is **out of scope**: the ticket concerns descendants left by a watchdog slot kill. Reaping after every successful exit would change controller lifecycle policy and could interrupt intentional detached work. HARD_TIMEOUT, all `STALL_*` thresholds and sample counts, and the 60/30-second grace periods remain unchanged; policy changes stay parked for the human.

Risks: shell children can exit/reparent while the snapshot is taken; retaining the first snapshot and re-walking survivors narrows this race but a child created just before its parent exits can evade `pgrep -P`. PID reuse during a long grace could point at an unrelated process; before KILL, compare process identity/start time where available or otherwise fail closed rather than signal an unverified recycled PID. The regression test must prove the normal descendant and grace-born cases, plus sibling isolation. A sandbox that prevents spawning or signalling the real fixture makes that process test **UNINFORMATIVE UNDER SANDBOX**; do not report it as a pass or fail, and rerun on the launchd rig.
