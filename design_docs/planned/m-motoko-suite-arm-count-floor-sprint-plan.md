# Sprint Plan: Motoko probe-suite arm-count gate

**Approved design:** `design_docs/planned/m-motoko-suite-arm-count-floor.md`

## Summary

Insert the approved host-scoped, exact-equality arm-count gate into
`tools/eval/test_motoko_connection_probe.sh`, then prove it with the complete four-mutant matrix.
This is a two-boundary sprint: M1 is the source change and pristine control; M2 is the independently
testable validation/evidence unit. The sandboxed executor must snapshot the cumulative tree after
each boundary into `.snap/M1/` and `.snap/M2/`; the controller, not the executor, owns commits.

**Expected duration:** 9-12 minutes elapsed on the Darwin rig, with a 15-minute working budget and
a hard ceiling below the mission's 30-minute cap.

**Risk level:** Medium. The code delta is small, but the gate is self-referential and a false count
would block the Darwin CI job.

**Dependencies:** Approved design at the path above; Darwin/bash-3.2 rig for authoritative runtime
evidence. Any run performed only in a socket/network-restricted sandbox is **UNINFORMATIVE UNDER
SANDBOX** and cannot satisfy a milestone boundary.

## Established facts — do not re-derive or substitute

Carry these values directly from the design's Verification Log:

- Darwin pristine before the gate: `PASS: 59 probe self-test arms ran`, rc=0, identical in 4/4 runs.
- Darwin with the gate: expected 60 (59 existing runtime arms plus the gate's own arm).
- Simulated non-Darwin pristine: `PASS: 55 probe self-test arms ran`, rc=0.
- Simulated non-Darwin with the gate: expected 56.

The non-Darwin observation is explicitly a PATH-shadowed-`uname` simulation, not evidence from a
real Linux host. This sprint does not repeat or upgrade that claim.

## Milestone boundaries

| Milestone | Independently committable/snapshotable unit | Boundary command | Expected observable |
|---|---|---|---|
| M1 — insert the gate | One source delta in `tools/eval/test_motoko_connection_probe.sh`; snapshot the cumulative tree to `.snap/M1/` only after the boundary passes | `bash tools/eval/test_motoko_connection_probe.sh` followed by `grep -c 'pass_arm ' tools/eval/test_motoko_connection_probe.sh` | On the Darwin rig: suite rc=0, terminal `PASS: 60 probe self-test arms ran`; grep prints `19`. A sandbox-only result is **UNINFORMATIVE UNDER SANDBOX**. |
| M2 — mutation-matrix validation | No additional production-source behavior. The independently committable validation unit is the completed four-row evidence plus restored pristine control and sprint-state completion; snapshot the restored cumulative tree and evidence to `.snap/M2/` | Run the five commands specified in the M2 matrix: four separate `bash tools/eval/test_motoko_connection_probe.sh` mutant runs and one fresh pristine run after final restoration | Each mutant has the exact rc/message specified below; every restoration matches the M1 SHA-256; final pristine run is rc=0 with `PASS: 60 probe self-test arms ran`, grep count 19, and the source tree contains only the intended M1 source delta plus planning/state artifacts. Sandbox-only results are **UNINFORMATIVE UNDER SANDBOX**. |

M1 and M2 must remain bisectable. Do not start M2 if M1's authoritative rig run is not green. Do
not retain any mutant in `.snap/M2/`; M2's source snapshot must be byte-identical to the M1 source
snapshot.

## M1 — insert the exact-equality gate

**Goal:** Add exactly the shell block in the approved design between the existing fatal
`arms == 0` guard and the final `echo "PASS: $arms probe self-test arms ran"`.

**Estimated delta:** approximately 20 shell lines; no new test script or production file.

### Tasks

1. In `tools/eval/test_motoko_connection_probe.sh`, insert the design's exact block without
   paraphrasing it. Preserve this load-bearing order:
   `arms == 0` guard → host-scoped `expected_arms` (60 Darwin / 56 otherwise) → the gate's own
   `pass_arm` → bare `if (( arms != expected_arms ))` → terminal PASS line.
2. Keep the gate last. Its own arm must execute before comparison, and no `+1` may appear in the
   comparison.
3. Run the M1 boundary command on the Darwin rig. Record the unpiped suite rc and terminal line;
   then record the static call-site count.
4. After the boundary passes, let the controller harness snapshot the cumulative tree to
   `.snap/M1/`. The executor performs no git write operation.

### M1 acceptance

- `bash tools/eval/test_motoko_connection_probe.sh` returns 0 and ends with
  `PASS: 60 probe self-test arms ran` on the Darwin rig.
- `grep -c 'pass_arm ' tools/eval/test_motoko_connection_probe.sh` prints `19`.
- The checked `$arms` and terminal `$arms` are the same fully observed value; comparison is exact
  equality expressed as `!=`, not a floor.
- The existing zero guard remains before the new gate, with no redundant zero guard added.
- M1 is snapshot independently in `.snap/M1/` after its boundary command.

## M2 — complete mutation matrix

**Goal:** Prove separately that the gate FIRES on removal, LOOKS for additions, observes its own
arm, and preserves the pre-existing zero anti-vacuity path. Removal and addition are not
interchangeable evidence: row R1 proves **FIRES**; only row R2 proves **LOOKS** because a floor gate
would survive an addition.

### Required protocol for every mutant

Use `target=tools/eval/test_motoko_connection_probe.sh`. Before the first mutant, copy the pristine
M1 file to a temporary backup outside the repo and capture
`base_sha=$(sha256sum "$target" | awk '{print $1}')`. For each row, in this order:

1. Apply only the exact edit named in the row.
2. Capture `mutant_sha` and require it to differ from `base_sha`; otherwise the row is an
   **INSTRUMENT FAILURE**, not a pass.
3. Run `bash "$target"` without a pipeline, capture its rc, and compare stderr/stdout with the
   row's observable. A sandbox-only run is **UNINFORMATIVE UNDER SANDBOX**.
4. Restore with `cp "$backup" "$target"`; require the restored SHA-256 to equal `base_sha`.
5. Before applying the next mutant, run `sha256sum "$target"` again and require `base_sha`.

Do not use git checkout/reset/stash/clean for restoration. Keep the backup until the final pristine
control passes.

### Four-mutant matrix

| Row / property | Exact edit | Command | Expected observable | Restore and byte-identity proof |
|---|---|---|---|---|
| R1 — removal proves the gate **FIRES** | Delete exactly `pass_arm "refusal-branch count still matches the set this suite covers ($actual_refusal_branches)"`. This arm is after the refusal comparison and matches none of that gate's three probe patterns. | `bash tools/eval/test_motoko_connection_probe.sh` | rc=1; output contains `not ok - arm-count drift: suite ran 59 arms, this suite is written for 60.` It must not reach terminal `PASS: 60`. | Copy the pristine backup over the target, then require `sha256sum` to equal `base_sha` before R2. |
| R2 — addition proves the gate **LOOKS** | Insert exactly `pass_arm "extra"` immediately before the new arm-count gate block, so it executes before the gate's own arm. | `bash tools/eval/test_motoko_connection_probe.sh` | rc=1; output contains `not ok - arm-count drift: suite ran 61 arms, this suite is written for 60.` This is the sole row that distinguishes exact equality from a floor. | Copy the pristine backup over the target, then require `sha256sum` to equal `base_sha` before R3. |
| R3 — self-reference regression | Delete exactly the gate's own line `pass_arm "arm-count still matches the set this suite covers ($((arms + 1)))"`; leave the subsequent bare comparison unchanged. | `bash tools/eval/test_motoko_connection_probe.sh` | rc=1; output contains `not ok - arm-count drift: suite ran 59 arms, this suite is written for 60.` This kills the defective first-draft `+1` assumption caught in quorum round 1. | Copy the pristine backup over the target, then require `sha256sum` to equal `base_sha` before R4. |
| R4 — zero anti-vacuity | Insert exactly `arms=0` immediately before the existing `if (( arms == 0 )); then` guard; do not alter either guard. | `bash tools/eval/test_motoko_connection_probe.sh` | rc=1; output contains `not ok - zero test arms ran`. The new gate's own `pass_arm` must not execute, and no arm-count-drift message is expected. | Copy the pristine backup over the target, then require `sha256sum` to equal `base_sha` before the control. |

### Fresh pristine control and M2 boundary

After R4 restoration:

1. Require `sha256sum "$target"` to equal `base_sha`.
2. Run `bash "$target"` once more on the Darwin rig, unpiped: expected rc=0 and terminal
   `PASS: 60 probe self-test arms ran`.
3. Run `grep -c 'pass_arm ' "$target"`: expected `19`.
4. Record all four mutant SHAs, rcs, named observables, each restored SHA match, and the pristine
   control in the M2 evidence captured by the controller's `.snap/M2/` snapshot. Mark M2 complete
   in the sprint JSON only after all rows and the control pass.
5. Snapshot `.snap/M2/`. Its copy of the target must hash identically to `.snap/M1/`'s copy.

### Wall-clock budget

One suite run takes about 50-70 seconds on this rig. M2 requires five runs (four mutants plus one
fresh pristine control): approximately 250-350 seconds, or 4m10s-5m50s of suite time. Allow 7-9
minutes for edits, rc capture, and SHA restoration checks. Including M1's boundary run, the sprint
uses six suite invocations: approximately 300-420 seconds (5-7 minutes) of suite time and a total
working budget of 9-12 minutes. Stop and report rather than silently exceeding 30 minutes.

## Conflict surface and unresolved limitation

The inserted text was measured against seven known direct grep patterns. The refusal-branch gate
reads `$probe`; the four wall-clock census greps read `$0`; the proposed lines matched none. Carry
the approved residual verbatim in substance:

> gpt6-astra's full-file audit of aliases, sourced files, and invoked helpers was NOT run. Four
> direct readers of `$0` were found; exhaustiveness is unverified.

Do not upgrade that statement to an exhaustiveness claim during execution. Discovering another
reader is a reason to stop and return to design review, not to expand this sprint silently.

## Out of scope

- Recounting or replacing the established 59/60 and simulated 55/56 values.
- Running a new real-Linux validation or treating the simulated non-Darwin result as real-Linux
  proof.
- Auditing aliases, sourced files, or invoked helpers to resolve the stated residual.
- Changing refusal-branch counts, wall-clock census logic, host-scoping policy, or using a `>=`
  floor.
- Any git write operation or implementation outside the single target shell file.

## Completion checklist

- [ ] M1 boundary command produces rc=0 / `PASS: 60...` and grep count 19 on the Darwin rig.
- [ ] `.snap/M1/` captured after M1.
- [ ] R1 removal proves FIRES with rc=1 / 59-vs-60 drift.
- [ ] R2 addition proves LOOKS with rc=1 / 61-vs-60 drift.
- [ ] R3 deletion of the gate's own arm proves observed self-reference with rc=1 / 59-vs-60 drift.
- [ ] R4 forced zero proves the old guard runs first with rc=1 / zero-test-arms message.
- [ ] Every mutant changes SHA and every restore returns to the same `base_sha`.
- [ ] Fresh pristine M2 control produces rc=0 / `PASS: 60...`; grep count is 19.
- [ ] `.snap/M2/` captured after restoration; target hash matches `.snap/M1/`.
- [ ] Any socket/process result earned only inside the sandbox is labeled **UNINFORMATIVE UNDER SANDBOX**.

