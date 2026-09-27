# Sprint plan: stall watchdog descendant progress

## Ticket

`stall-watchdog:kills-controller-on-long-drill` — mechanical half of Mark's attended triage. Sprint ID: `M-FLEET-STALL-DESCENDANT-PROGRESS`. **Policy park:** no implementation is authorized by this plan. The controller must obtain Mark's decision on the proposed numeric discriminator before execution.

## Evidence

- At 2026-09-26 04:45:43 the World watchdog reported `flat prog=1908272+w212409 hb=209 cpu=1`, then killed the Claude controller at gate 3 (`rc=143`, elapsed 4602 s). Its pi planner had begun a 46-row mutation harness at 04:40; that harness later reported `46/46 KILLED` at 05:11. Each row edited a `/tmp` scratch copy and ran a test. The transcript, held-open write-file byte sum, and heartbeat were flat while useful work continued.
- `_mc_stalled` in `tools/launchd/mission-control.sh` checks a long-lived descendant, transcript/driver-log bytes plus held-open write-file bytes, heartbeat bytes, and instantaneous tree CPU. `tools/launchd/test_mission_stall.sh` extracts the real functions and is included in `make test-launchd-drivers`. Commit `4a86ea17b` added the pi transcript and held-open write-file arms; every change in either counter counts as progress.
- W1: iteration 13's `until COND; do sleep 30; done` starts a short command about every 30 seconds, with roughly 0.1 s CPU per attempt. W2: the 2026-09-22 blocked `find` over `$HOME` read flat, including CPU 0. Both must remain killable.
- The sampling interval is 120 s. The rig uses Bash 3.2 and macOS BSD tools. `/tmp` is shared by missions; unrelated changes there cannot establish progress for this process tree.

## Arm analysis

“Yes” under 04:45 means the observed signal is established by the supplied evidence, rather than merely plausible. “Flat” means the arm would not reset the wedge count. Cost refers to each sample, without scans of unbounded directory trees.

| Candidate | 04:45 live? | W1 flat? | W2 flat? | No new threshold? | Bounded cost? | Residual false kill / false live |
| --- | --- | --- | --- | --- | --- | --- |
| Change in descendant PID set | Likely, but row boundaries can fall between samples | **No**: each failed condition starts a fresh process | Yes for a blocked `find` | Yes | Yes, proportional to tree size | False kill when work occurs between snapshots; false live on W1's repeated commands. |
| Any increase in cumulative tree CPU (`ps -S -o time`, counting reaped children on the persistent parent) | Likely, but no CPU-time series was recorded | **No**: repeated 0.1 s commands accumulate CPU | Yes while `find` remains blocked | Yes | Yes, proportional to tree size | False kill for I/O-bound useful work or sub-resolution deltas; false live on W1 and CPU-spinning wedges. Summing `-S` for parent and children can double-count active descendants. |
| Changed mtimes of files under descendant cwd paths from `lsof -a -d cwd -p <pids>` | **Unproved**: scratch files may be outside their cwd; edits to file contents do not advance the cwd directory mtime | Usually, unless the condition or another actor writes there | Usually, unless another actor writes there | Yes | **No** for a complete recursive scan; a bounded, shallow scan misses nested edits | False kill on nested or out-of-cwd scratch edits; false live when a different mission changes a shared cwd. Scanning all `/tmp` would both violate the cost bound and count unrelated work. |
| Files currently held open for write, with byte growth | **No**: the measured `w212409` was flat | Yes | Yes | Yes | Yes, already implemented | False kill on short-lived open/write/close cycles, exactly this case. |
| Process-owned bytes written via kernel I/O counters | Unproved; no verified Bash/BSD-`ps` interface for an entire descendant tree plus reaped children | **No** if failed conditions write logs/cache | Yes for a blocked read | Yes | Potentially | False live on repeated failed writes; false kill when attribution is unavailable or work only reads. |

No candidate satisfies all five properties. In particular, CPU movement and child churn detect activity, not useful progress; cwd movement is neither reliably attributable nor reliably visible for the measured harness.

## Chosen arm

**NONE — policy park.** The smallest plausible addition is cumulative CPU time on the persistent descendant shell using BSD `ps -S -o time`, compared between samples. It needs a **new numeric threshold**; any-change semantics leaves W1 alive indefinitely. A provisional discriminator for Mark to evaluate is **at least 2 CPU seconds per 120-second sample**. W1's example four 0.1-second attempts per interval should remain below it; W2's blocked `find` should remain at zero. A 46-row compile/test harness plausibly exceeds it, but the 04:45 log contains only instantaneous `%cpu`, not cumulative CPU time, so its success is **not established**. CPU accounting resolution, different `COND` costs, and harness phases can reverse these expectations. Do not implement this arm until a controller-run replay or direct measurement of all three shapes supports a threshold and Mark approves it.

## Milestones

These are conditional execution milestones after the policy decision. The controller owns the decision and the commit. No other implementation files are in scope.

### M1 — Measure and decide the discriminator

- Files: `tools/launchd/test_mission_stall.sh` (synthetic, `mktemp -d` fixture only); no production edits.
- Observe cumulative `ps -S -o time` deltas for a controlled 46-row scratch mutation/test reproduction, W1's repeated short condition, and W2's blocked-read shape on the unsandboxed rig. Record the two sample endpoints, PID identity and birth time, elapsed interval, and delta for each. A directory or file fixture must be rooted in `mktemp -d`; the test must never seed a real home or shared log.
- Exact acceptance command: `/bin/bash tools/launchd/test_mission_stall.sh`. Expected result: all base arms green (16/16 at base), with named 04:45-shape, W1, and W2 arms added and green only if the measured threshold separates them. An unsandboxed controller must inspect the raw measurement; a sandboxed `ps`, socket, or outside-worktree path result is **uninformative under sandbox**, neither pass nor fail.
- If the measured distributions overlap, stop here and return the decision to Mark; no watchdog change follows.

### M2 — Implement only after Mark approves a measured threshold

- Files: `tools/launchd/mission-control.sh`, `tools/launchd/test_mission_stall.sh`, `changelogs/v0.32-current.md` (the current changelog touched by `4a86ea17b`). Add a cumulative CPU-time arm for the persistent descendant, with process identity protection against PID reuse; fold it into existing progress bookkeeping only after the approved discriminator is met. Keep reads proportional to the process tree and use Bash 3.2 and BSD syntax.
- Exact acceptance command: `bash -n tools/launchd/mission-control.sh`. Expected result: exit 0.
- Exact acceptance command: `/bin/bash tools/launchd/test_mission_stall.sh`. Expected result: zero failed arms, including the 04:45-shape positive, W1 negative, W2 negative, and a red-on-mutation check for the new arm.
- Exact acceptance command: `env -i HOME=$HOME PATH=$PATH make test-launchd-drivers` (**controller-run outside the sandbox**). Expected result: exit 0 with the stall suite and all other launchd suites green. Runs inside this sandbox that touch sockets or paths outside the worktree are **uninformative under sandbox**, neither pass nor fail.

## Test arms

- **04:45 positive:** a long-lived descendant runs a mutation/test fixture rooted at `mktemp -d`; controller transcript, held-open write-file bytes, heartbeat, and instantaneous CPU remain flat at their measured shape. The approved cumulative CPU delta alone makes the second sample live. Use the recorded real delta in addition to a deterministic stub; an invented stub alone cannot establish the original false kill is fixed.
- **W1 negative:** a persistent shell repeatedly runs a short failed command every 30 seconds, with new child PIDs and a nonzero but sub-threshold cumulative CPU delta. With other arms flat, the second sample is stalled.
- **W2 negative:** a long-lived blocked `find` shape has no writes and zero cumulative CPU delta. With other arms flat, the second sample is stalled.
- **Mutation for the one proposed new arm:** replace its threshold-satisfied branch with `return 1` on one line. The named “04:45 positive” test must turn red. Also check the opposite one-line mutation, treating any positive CPU delta as progress: the named “W1 negative” test must turn red.
- Preserve the baseline transcript, heartbeat, held-open write-file, and no-instrument arms. Each test fixture creates and removes its own `mktemp -d` root; no acceptance criterion requires writing to a real path outside the worktree.

## Out of scope

Changing existing watchdog thresholds, sample counts, or their semantics; adding the proposed numeric threshold before Mark's policy decision; changing other providers' transcript arms; scanning `$HOME` or shared `/tmp`; and implementation outside the three files named in M2.

## Risks

- The 04:45 CPU-time delta is unmeasured. A provisional threshold cannot be called a fix until the three shapes are measured on the rig.
- A more expensive W1 condition could cross a fixed CPU threshold and become falsely live. A mostly I/O-bound useful drill could stay below it and be falsely killed.
- BSD `ps -S` includes reaped child CPU on a parent; attribution must use a stable parent identity, avoid double counting, and account for display precision and PID reuse.
- A sampling instant can miss short lived processes; no amount of synthetic stubbing proves that live rig accounting captures every harness row.
