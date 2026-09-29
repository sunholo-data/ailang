# Sprint plan: stall watchdog descendant progress

## Ticket

`stall-watchdog:kills-controller-on-long-drill` — mechanical half of Mark's attended triage. Sprint ID: `M-FLEET-STALL-DESCENDANT-PROGRESS`. **M1 complete** (fleet iteration 4, 2026-09-27). **M2 authorized:** Mark ruled D-FLEET-7 on 2026-09-29: descendant-only rusage growth of at least 10 CPU-s per 120-s sample counts as progress. The measured heavy-poll W1 false-live risk is accepted; existing sample counts and 600-s budget stay unchanged.

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

## Chosen arm and ruling

Use `proc_pid_rusage` through the landed `tools/launchd/lib/proc_rusage.py` helper. Compare cumulative CPU for the controller's **descendants only**, excluding the root controller, across the existing 120-s samples. A delta of **at least 10 CPU-s** resets the stall hit count as progress. Preserve the existing instantaneous CPU, transcript, held-open write-file, heartbeat, long-child, sample-count, and 600-s budget behavior. A missing or invalid rusage reading is not fabricated as progress; preserve the watchdog's documented fail-open behavior where an instrument needed to distinguish live work is unavailable. D-FLEET-7 expressly approves this threshold and the heavy-poll W1 risk.

Historical pre-M1 proposal: `ps -S -o time` with a provisional 2 CPU-s threshold was parked pending measurement. M1 proved that BSD `ps -S` misses reaped-child CPU on this rig; neither that instrument nor that provisional threshold is an M2 instruction.

## Milestones

M1 is measured and landed (`d9e1211d0`); M2 is now executable under D-FLEET-7. The controller owns the commit. No other implementation files are in scope.

### M1 — Measure and decide the discriminator

- Files: `tools/launchd/test_mission_stall.sh` (synthetic, `mktemp -d` fixture only); no production edits.
- Observe cumulative `ps -S -o time` deltas for a controlled 46-row scratch mutation/test reproduction, W1's repeated short condition, and W2's blocked-read shape on the unsandboxed rig. Record the two sample endpoints, PID identity and birth time, elapsed interval, and delta for each. A directory or file fixture must be rooted in `mktemp -d`; the test must never seed a real home or shared log.
- Exact acceptance command: `/bin/bash tools/launchd/test_mission_stall.sh`. Expected result: all base arms green (16/16 at base), with named 04:45-shape, W1, and W2 arms added and green only if the measured threshold separates them. An unsandboxed controller must inspect the raw measurement; a sandboxed `ps`, socket, or outside-worktree path result is **uninformative under sandbox**, neither pass nor fail.
- If the measured distributions overlap, stop here and return the decision to Mark; no watchdog change follows.

### M2 — Implement the approved descendant rusage arm

- Files: `tools/launchd/mission-control.sh`, `tools/launchd/test_mission_stall.sh`, `changelogs/v0.32-current.md` (the current changelog touched by `4a86ea17b`). Reuse the landed `tools/launchd/lib/proc_rusage.py` helper; it emits own and reaped-child CPU centiseconds for each PID. Add an arm for cumulative descendant-only rusage CPU, excluding the controller root. Compare two samples of the same process identity, guarding PID reuse; count a window as progress at **≥1000 centiseconds per existing 120-s sample**. Sum or attribute active and reaped children without double counting. Retain existing progress arms, thresholds, sample counts, and the 600-s budget. Keep reads proportional to the process tree, use Bash 3.2-compatible shell, and surface unavailable or malformed rusage rather than treating it as a zero-cost success.
- Add a deterministic fixture for vanished non-root children before relying on its mutation check. Keep a live positive control for reaped-child accounting, including the reported but unreproduced reap-order concern. Confirm the new arm affects the Claude long-drill shape; #1391 separately caps pi controller commands at 540 s.
- Exact acceptance command: `bash -n tools/launchd/mission-control.sh`. Expected result: exit 0.
- Exact acceptance command: `/bin/bash tools/launchd/test_mission_stall.sh`. Expected result: zero failed arms, including the 04:45-shape positive, W1 negative, W2 negative, and a red-on-mutation check for the new arm.
- Exact acceptance command: `env -i HOME=$HOME PATH=$PATH make test-launchd-drivers` (**controller-run outside the sandbox**). Expected result: exit 0 with the stall suite and all other launchd suites green. Runs inside this sandbox that touch sockets or paths outside the worktree are **uninformative under sandbox**, neither pass nor fail.

## Test arms

- **04:45 positive:** a long-lived descendant runs a mutation/test fixture rooted at `mktemp -d`; controller transcript, held-open write-file bytes, heartbeat, and instantaneous CPU remain flat at their measured shape. The approved cumulative CPU delta alone makes the second sample live. Use the recorded real delta in addition to a deterministic stub; an invented stub alone cannot establish the original false kill is fixed.
- **W1 negative:** a persistent shell repeatedly runs a short failed command every 30 seconds, with new child PIDs and a nonzero but sub-threshold cumulative CPU delta. With other arms flat, the second sample is stalled.
- **W2 negative:** a long-lived blocked `find` shape has no writes and zero cumulative CPU delta. With other arms flat, the second sample is stalled.
- **Mutation for the approved new arm:** replace its threshold-satisfied branch with `return 1` on one line. The named “04:45 positive” test must turn red. Also check the opposite one-line mutation, treating any positive CPU delta as progress: the named “W1 negative” test must turn red. The vanished-child fixture must kill its corresponding mutation reliably.
- Preserve the baseline transcript, heartbeat, held-open write-file, and no-instrument arms. Each test fixture creates and removes its own `mktemp -d` root; no acceptance criterion requires writing to a real path outside the worktree.

## Out of scope

Changing existing watchdog thresholds other than the new approved rusage discriminator, sample counts, or their semantics; changing other providers' transcript arms; scanning `$HOME` or shared `/tmp`; changing the landed rusage helper without a demonstrated defect; and implementation outside the three files named in M2.

## Risks

- The 04:45 incident itself has no cumulative CPU series; M1's controlled 46-row reproduction measured 62.54–97.73 CPU-s against W1 ≤1.72 and W2 0. The live positive control must exercise rusage rather than rely solely on a stub.
- A more expensive W1 condition could cross 10 CPU-s and become falsely live; D-FLEET-7 accepts that risk. A mostly I/O-bound useful drill could stay below it and be falsely killed.
- `proc_pid_rusage` child accounting and PID reuse require stable identity, no double counting, and explicit handling of unavailable data. The evaluator's reap-order concern was unreproduced; retain a live positive control.
- A sampling instant can miss short lived processes; the deterministic vanished-child fixture must test that edge without relying on timing luck.

## M1 result (fleet iteration 4, 2026-09-27, measured outside the sandbox on the rig)

Instrument: `tools/launchd/measure_stall_cpu.sh` (default run: four shapes concurrently, 5 windows of
about 121 s). TSV and summary banked in the iteration-4 log entry.

**The plan's instrument does not work on the rig.** On macOS 26.6.2, `ps -S -o time` misses almost all of
a reaped child's CPU. A bash parent whose `perl` child burned about 4 CPU-s reads `0:00.00`, while a live
busy process reads correctly in the same run. The controller and the evaluator each reproduced this,
and other `ps` keywords behave the same. The working source is `proc_pid_rusage(pid, RUSAGE_INFO_V2)`:
`ri_child_user_time + ri_child_system_time` is the CPU of reaped children. It read 2.18–2.99 s for a
3-second busy child across six controller variants. The tool reads it through
`tools/launchd/lib/proc_rusage.py`.

Cumulative CPU per window, persistent parent (self plus reaped children):

| Shape | rusage min / median / max (CPU-s per ~121 s) | `ps -S` (CPU-s) |
|---|---|---|
| drill (04:45 shape: rewrite a constant, `go test -count=1`, 3 s sleep in the test) | 62.54 / 89.66 / 97.73 | 3.13–4.68 |
| w1-git (`until git status && false; do sleep 30`) | 0.73 / 1.63 / 1.72 | 0.00–0.01 |
| w1-gh (`until gh run list … && false; do sleep 30`) | 0.22 / 0.27 / 0.31 | 0.00–0.01 |
| w2 (blocked FIFO read) | 0.00 / 0.00 / 0.00 | 0.00 |

The whole-tree sum equals the parent's sum within 0.04 CPU-s in every window.

**Controller-process noise, measured separately:** 21 live `claude` processes each accrued
**0.57–1.60 CPU-s per 120 s** while mostly idle. An arm that included the ROOT controller would put
W1 (1.72 plus up to 1.60) above the provisional 2 CPU-s line. So the root must be excluded.

**Separation:** the drill's minimum window is 36x w1-git's maximum. The provisional ≥2 CPU-s line
separates the fixtures only if the root is excluded, and even then its margin over w1-git's maximum is
only 0.28 CPU-s. A threshold near the geometric midpoint, **10 CPU-s per 120-s window over the
descendants excluding the controller root**, leaves a margin of about 6x on both sides.

**Caveats for M2:** (1) The drill fixture is a small package, and the real World rows compiled larger
packages, so the real drill likely sits higher. That direction is safe. (2) A W1 whose condition is
itself heavy (a `go test` as the poll condition) could cross 10 CPU-s. That is a false-live risk and
belongs to the ruling. (3) Reading rusage needs `python3` (ctypes) or a compiled helper in the driver
path, where today the driver uses only `ps`. (4) The evaluator reported a reap-order blind spot in
`ri_child_*`: 0 after a trivial child is reaped first. The controller did **not** reproduce it in five
variants, including that exact ordering (2.18–2.99 s each), so it is recorded as unreproduced. M2
should keep a positive control on live data regardless. (5) The selftest's check for a vanished
non-root child is racy: the evaluator's mutation was caught in 1 of 10 runs. M2 needs a deterministic
fixture before it relies on this.

## M2 result (fleet iteration 7, 2026-09-29)

Implementation commits `983ae18dc` and `1f0c57882` add the approved descendant rusage arm. `bash -n tools/launchd/mission-control.sh` passed; the focused stall suite passed 35/35; `env -i HOME=$HOME PATH=$PATH make test-launchd-drivers` exited 0. Independent Sonnet evaluator round 1 found two surviving root-accounting mutations and a vanished-child gap (PASS 74); round 2 confirmed the corrections and passed 88/100 with no blocking findings. The remaining nonblocking finding is that the malformed non-NA fail-open branch lacks a direct mutation victim. This milestone is locally complete; Gate 3b still requires PR and merge-commit CI before the ticket can be resolved.
