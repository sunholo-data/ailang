# Sprint Plan: M-MOTOKO-DISCOVERY-ARM-DISCRIMINATING-REFUSAL

**Design doc**: [`design_docs/planned/m-motoko-discovery-arm-discriminating-refusal.md`](m-motoko-discovery-arm-discriminating-refusal.md)
**Sprint JSON**: `.ailang/state/sprints/sprint_m-motoko-discovery-arm-discriminating-refusal.json`
**Planned**: 2026-09-01 (mission iteration 32)
**Base commit**: `48817dcdd` (= `origin/dev`), worktree `/Users/voightkampff/dev/sunholo-data/.wt-motoko-iter32-sprint`
**Target**: v0.34.1 · **Risk**: low (code) / medium (schedule — see §Risks)
**Design status**: cleared to plan. Four quorum rounds; the ceiling surface was resolved by attended
ruling **D-MOTOKO-6N-1** (Mark Edmondson, 2026-09-01, option (B) = adopt §D4). Round-4's two
consistency defects were fixed verbatim under the narrow-refinement carve-out. **This plan does not
re-open the design.**

---

## 0. How to read this plan

Four milestones. **M1, M2, M3 each change code and each leaves the suite green**, so each is an
independently committable, bisectable point. **M4 changes no product code** — it produces the mutation
evidence and records it in the design doc.

Three standing rules for the executor, and they are not optional:

1. **The executor makes NO git writes.** No `git add`, no `git commit`, no `git checkout`, no
   `git stash`, no branch operations. The worktree is deliberately uncommitted; the **controller**
   commits, one commit per milestone, after the milestone's acceptance block is green. If a milestone's
   acceptance goes red, stop and report — do not proceed to the next milestone.
2. **Restore mutants from a `cp` backup, never `git checkout --`.** The sprint's own edits are
   uncommitted, so `git checkout -- <file>` would delete them. Every mutation is bracketed by
   `cp <file> <file>.bak.<tag>` before and `cp <file>.bak.<tag> <file>` after, and the restore is
   **verified by sha256 equality against the pre-mutation hash**, not assumed.
3. **Capture exit codes without a pipe**: `cmd > /tmp/out 2>&1; rc=$?`. The controller's shell is zsh
   and the suite is `/bin/bash`; `${PIPESTATUS[0]}` is not to be used anywhere.

All paths below are relative to the worktree root
`/Users/voightkampff/dev/sunholo-data/.wt-motoko-iter32-sprint`. Abbreviations used throughout:

- `P` = `tools/eval/motoko_connection_probe.sh`
- `T` = `tools/eval/test_motoko_connection_probe.sh`

**These are the only two files this sprint may modify**, plus the design doc (M4) and this plan's JSON.

---

## 1. Baseline — measured on the pristine tree, this iteration

Every acceptance command in this plan was run on the **unmodified** tree at `48817dcdd` before the plan
was written. The measured base result is recorded beside each criterion in the milestone sections, and
the whole set is summarised here. **No criterion in this plan was found red at base.**

Environment at measurement time:

| Fact | Measured |
|---|---|
| Worktree base | `48817dcdd`, clean except the (uncommitted) design doc |
| `sw_vers -productVersion` | `26.5.2` |
| `/bin/bash --version` | GNU bash 3.2.57(1)-release, arm64-apple-darwin25 |
| `command -v timeout gtimeout` | **neither exists** — see §Bounded waits |
| `uptime` load avg / `hw.ncpu` | 13.87–18.52 on **16** CPUs — heavily contended |
| sha256 `P` | `7a75c6984f743e1742316c6684dae1898ffc72b4db9266b7925e370c7faeef88` |
| sha256 `T` | `bd5302f596201a63bace988c9aae68bdb57bc9d3232f382e360cc4f1ca461ee6` |

**Base suite (the load-bearing baseline):**

```bash
/bin/bash tools/eval/test_motoko_connection_probe.sh > /tmp/base_suite.log 2>&1; rc=$?
```
→ **rc=0, 41 `ok`, 0 `not ok`, 50s**, last line `PASS: 41 probe self-test arms ran`.
Arm numbering confirmed in that log: **arm 33** = "descendant discovery refuses on the real wall-clock
deadline" (`T:449`), **arm 40** = "descendant discovery refuses on the node-count ceiling" (`T:698`),
**arm 41** = the refusal-branch gate, which at base prints `(24)`.

**Base static values** (each command run at base; `grep -c` counts *lines in the named file*):

| Command | Base |
|---|---|
| `bash -n $P` | rc=0 |
| `bash -n $T` | rc=0 |
| `grep -c 'echo "process-tree discovery' $P` | 3 |
| `grep -c 'instrument_failure "' $P` | 19 |
| `grep -cE '\|\| usage$' $P` | 5 |
| `grep -Fc 'echo "process-tree discovery deadline expired" >&2' $P` | **2** (the two branches that share a string — the defect) |
| `grep -Fc 'process-tree discovery deadline expired (test stub)' $P` | 0 |
| `grep -Fc 'process-tree discovery deadline expired (wall clock)' $P` | 0 |
| `grep -Fc 'MAX_TREE_NODES=${PROBE_MAX_TREE_NODES:-4096}' $P` | 1 (**`-F` mandatory**; the non-`-F` form returns a false **0** on this rig — re-confirmed this iteration) |
| `grep -c 'PROBE_MAX_TREE_NODES=50000' $T` | 0 |
| `grep -cE '^(export )?PROBE_MAX_TREE_NODES=' $T` | 0 (control `grep -c '^ARM_CAP_SECS=' $T` = 3) |
| `grep -Fc '"descendant discovery refuses on the real wall-clock deadline" "process-tree discovery failed"' $T` | 1 |
| `grep -Fc '"descendant discovery deadline refuses at the caller" "process-tree discovery failed"' $T` | 1 |
| `grep -Fc 'ARM_CAP_SECS=$(( ARM_CAP_SECS * 5 ))' $T` | 1 |
| `grep -Fc 'expected_refusal_branches=24' $T` / `...=27` | 1 / 0 |
| `grep -Fc 'actual_echo_refusals' $T` | 0 |
| `grep -Fc '[[ -f "$probe" ]]' $T` | 0 |
| `grep -Fc 'PROBE_MAX_TREE_NODES is set at suite scope' $T` | 0 |
| `PROBE_MAX_TREE_NODES=50000 /bin/bash $T` | **rc=0, 41 ok, 0 not ok, 57s** — i.e. an ambient global ceiling is **invisible today**. This is A3.7's base and it is exactly the hole the locality guard closes. |

**Edit positions re-verified at base** (the design doc's V17, all confirmed unchanged):
`P:182` and `P:187` = the two identical `echo` messages; `P:195` = the node-ceiling message;
`P:126` = `MAX_TREE_NODES=${PROBE_MAX_TREE_NODES:-4096}`; `P:217` = the caller collapse.
`T:447-452` = the arm-cap widening + arm :449 + restore; `T:450` = arm :449's `env` line;
`T:358` = the stub arm's drive; `T:698-701` = the node-ceiling arm; `T:707` =
`expected_refusal_branches=24`. `T:5` assigns `probe=${PROBE_UNDER_TEST:-$script_dir/motoko_connection_probe.sh}`
and `T:2` is `set -uo pipefail`.

**Stub rate re-measured at this base, this iteration** (the suite's own `pgrep` stub, extracted verbatim
from `T:255-261`, N=2000 per trial, timed with `date +%s`): **181 / 166 / 181 iter/s** at load average
13.87–18.52. The design doc's V23 recorded 181/200/200; iteration 31's quiet-machine figures were
474.9/652.7/648.6. **The rig is currently at or slightly below the slow end of the doc's own range** —
which is a schedule fact, not a design fact. See §Risks R1.

### Bounded waits

There is **no `timeout(1)` and no `gtimeout` on this machine**, so every bound comes from the harness,
not from a shell wrapper:

- Short commands (`grep`, `bash -n`, `sed`, `cp`, `shasum`): default tool timeout is ample.
- **Full suite runs (clean, ~50–60s)**: run with an explicit tool timeout of **300000 ms**.
- **T1 mutant run**: run with **`run_in_background: true`** and poll the log. Do **not** run it in the
  foreground: its arm :449 walks to the 50,000-node ceiling and, at the rates measured above, that arm
  alone is **~250–301s**, giving a whole-suite time of roughly **280–340s** — and under a load spike it
  can approach the 600s widened arm cap, which would exceed the harness's own 600s foreground ceiling.
  **A multi-minute T1 is EXPECTED, not a hang.** Iteration 31's 44s figure was measured at the *default*
  ceiling under the old design and **does not transfer**.
- **T2 mutant run**: arm 40's walk falls back to its own `PROBE_TIMEOUT_SECS=60` wall clock, so budget
  **~60s for that arm alone**; run with a tool timeout of **300000 ms**.

---

## 2. Milestones

### M1 — probe: three refusal branches, three distinct messages

**Design doc**: Implementation Plan Phase 1; decision (a).
**Files**: `P` only. **~2 lines changed.**

**Change** — in `descendant_pids`:

| line | from | to |
|---|---|---|
| `P:182` | `    echo "process-tree discovery deadline expired" >&2` | `    echo "process-tree discovery deadline expired (test stub)" >&2` |
| `P:187` | `      echo "process-tree discovery deadline expired" >&2` | `      echo "process-tree discovery deadline expired (wall clock)" >&2` |

Preserve the existing indentation exactly (4 spaces at 182, 6 at 187). `P:195` (the node-ceiling
message) and `P:217` (the caller collapse) are **not** touched.

**Why this is green on its own**: arm :449 and arm :357 both still assert the caller wrapper
`process-tree discovery failed`, which is unchanged, and the gate's `echo "process-tree discovery`
count is still 3. So M1 commits clean.

**Acceptance block** — run all of these; every one must hold:

| # | Command | **Base** | Required after M1 |
|---|---|---|---|
| A1.1 | `bash -n tools/eval/motoko_connection_probe.sh; rc=$?` | rc=0 | rc=0 |
| A1.2 | `grep -c 'echo "process-tree discovery' tools/eval/motoko_connection_probe.sh` | 3 | 3 (re-wording must not change the count) |
| A1.3 | `grep -Fc 'process-tree discovery deadline expired (test stub)' tools/eval/motoko_connection_probe.sh` | 0 | **1** |
| A1.4 | `grep -Fc 'process-tree discovery deadline expired (wall clock)' tools/eval/motoko_connection_probe.sh` | 0 | **1** |
| A1.5 | `grep -Fc 'echo "process-tree discovery deadline expired" >&2' tools/eval/motoko_connection_probe.sh` | 2 | **0** — no two branches share a string (Design Freeze) |
| A1.6 | `grep -Fc 'process-tree discovery exceeded $MAX_TREE_NODES nodes' tools/eval/motoko_connection_probe.sh` | 1 | 1 (node-ceiling message untouched) |
| A1.7 | `grep -Fc 'instrument_failure "process-tree discovery failed"' tools/eval/motoko_connection_probe.sh` | 1 | 1 (caller collapse untouched — Non-Goal: no production-semantics change) |
| A1.8 | `/bin/bash tools/eval/test_motoko_connection_probe.sh > /tmp/m1_suite.log 2>&1; rc=$?` | rc=0, 41 ok, 0 not ok, 50s | rc=0, 41 ok, 0 not ok |

**Commit (controller, not executor)**: `fix(probe): distinct refusal message per descendant_pids branch`.

---

### M2 — test: arm :449 asserts the wall-clock message and carries the scoped ceiling

**Design doc**: Implementation Plan Phase 2; decisions (b), (c); attended ruling D-MOTOKO-6N-1.
**Files**: `T` only. **~2 lines changed.** **Depends on M1** (the asserted string must exist first).

**Change** — the arm at `T:449-450` becomes:

```bash
expect_failure "descendant discovery refuses on the real wall-clock deadline" "process-tree discovery deadline expired (wall clock)" \
  env PATH="$live_bin" AILANG_BIN=ailang-stub PROBE_TIMEOUT_SECS=1 PROBE_MAX_TREE_NODES=50000 PROBE_TEST_PGREP_LOOP=1 \
    PROBE_STUB_STATE="$tmp_dir/lane-pgreploop" /bin/bash "$probe" treatment control "$tmp_dir/pgreploop.json"
```

Exactly two edits: the expectation string, and **one added token** `PROBE_MAX_TREE_NODES=50000`
inserted after `PROBE_TIMEOUT_SECS=1` on the `env` line. **Everything else on those lines is
byte-identical**, including `PROBE_STUB_STATE` and the continuation backslashes.

**Do not touch** `T:447` (`_arm_cap_saved=$ARM_CAP_SECS`), `T:448`
(`ARM_CAP_SECS=$(( ARM_CAP_SECS * 5 ))`) or `T:452` (the restore). The 5x widening is **KEPT** per the
Design Freeze — it is now also T1's outer bound.

**Do not** add an `export`, and **do not** add a `PROBE_MAX_TREE_NODES=` assignment anywhere at file
scope. A2.3 is the criterion that reds if that happens.

**Acceptance block**:

| # | Command | **Base** | Required after M2 |
|---|---|---|---|
| A2.1 | `bash -n tools/eval/test_motoko_connection_probe.sh; rc=$?` | rc=0 | rc=0 |
| A2.2 | `grep -c 'PROBE_MAX_TREE_NODES=50000' tools/eval/test_motoko_connection_probe.sh` | 0 | **exactly 1** — >1 means the override spread to another arm; 0 means D4 was not applied |
| A2.3 | `grep -cE '^(export )?PROBE_MAX_TREE_NODES=' tools/eval/test_motoko_connection_probe.sh` | 0 (control `grep -c '^ARM_CAP_SECS=' $T` = 3) | **0** — the static half of the locality claim |
| A2.4 | `grep -n -B1 'PROBE_MAX_TREE_NODES=50000' tools/eval/test_motoko_connection_probe.sh` | (no match) | the matched line is arm :449's `env` line, and the `-B1` line is that arm's own `expect_failure` header — **proving the token is on THIS arm, not merely present in the file** |
| A2.5 | `grep -Fc 'expect_failure "descendant discovery refuses on the real wall-clock deadline" "process-tree discovery deadline expired (wall clock)"' tools/eval/test_motoko_connection_probe.sh` | 0 | **1** |
| A2.6 | `grep -Fc '"descendant discovery refuses on the real wall-clock deadline" "process-tree discovery failed"' tools/eval/test_motoko_connection_probe.sh` | 1 | **0** — the vacuous assertion is gone |
| A2.7 | `grep -Fc '"descendant discovery deadline refuses at the caller" "process-tree discovery failed"' tools/eval/test_motoko_connection_probe.sh` | 1 | **1** — arm :357 still tests the caller collapse (decision (b)) |
| A2.8 | `grep -Fc 'ARM_CAP_SECS=$(( ARM_CAP_SECS * 5 ))' tools/eval/test_motoko_connection_probe.sh` | 1 | 1 — widening kept |
| A2.9 | `grep -Fc '"descendant discovery refuses on the node-count ceiling" "process-tree discovery exceeded 3 nodes"' tools/eval/test_motoko_connection_probe.sh` | 1 | 1 — arm 40 untouched |
| A2.10 | `grep -Fc 'MAX_TREE_NODES=${PROBE_MAX_TREE_NODES:-4096}' tools/eval/motoko_connection_probe.sh` (**`-F` mandatory**) | 1 | 1 — every other arm keeps the 4096 default |
| A2.11 | `/bin/bash tools/eval/test_motoko_connection_probe.sh > /tmp/m2_suite.log 2>&1; rc=$?` | rc=0, 41 ok, 0 not ok, 50s | rc=0, 41 ok, 0 not ok. **Also record the wall time** — the doc's iteration-31 evaluator measured the scoped override as free on the happy path (47.1s); a materially longer run would mean the ceiling is being reached on arm 33, which would be a finding worth reporting. |
| A2.12 | `grep -c '^ok 33 - descendant discovery refuses on the real wall-clock deadline' /tmp/m2_suite.log` | 1 (in the base log) | 1 — the arm is still arm 33 and still passes |

**Commit (controller)**: `test(probe): arm :449 asserts the wall-clock branch, ceiling scoped to its env line (D-MOTOKO-6N-1)`.

---

### M3 — test: refusal-branch gate counts the echo refusals, and enforces ceiling locality

**Design doc**: Implementation Plan Phase 3; decision (f); round-4 fix V26.
**Files**: `T` only. **~15 lines added, 1 changed.** Independent of M1/M2 in content; sequenced third so
each commit is green.

**Change** — replace the gate block that currently begins at `T:707` (`expected_refusal_branches=24`)
so the region reads:

```bash
# The D4 ceiling override belongs on arm :449's own env line and nowhere else. A per-command
# env assignment never persists into this shell, so this is invariantly quiet on a correct
# tree. It fires on exactly two leak shapes: an edit that promotes the override to a
# file-global assignment or export, and an ambient PROBE_MAX_TREE_NODES in the caller's
# environment — which would silently re-parameterise every arm that does not pin its own
# ceiling. Both un-hermeticize the suite.
if [[ -n "${PROBE_MAX_TREE_NODES:-}" ]]; then
  echo "not ok - PROBE_MAX_TREE_NODES is set at suite scope; the ceiling override must stay on arm env lines" >&2
  exit 1
fi

# Refusal-branch drift gate.  <-- KEEP the existing 4-line comment here verbatim
expected_refusal_branches=27
# Every counter below reads $probe. Assert it resolves to a file BEFORE any of them run, so
# that no grep in this gate can fall through to reading stdin.
[[ -f "$probe" ]] || { echo "not ok - refusal-branch gate: \$probe does not resolve to a file; instrument failure, not a verdict" >&2; exit 1; }
actual_instrument_failures=$(grep -c 'instrument_failure "' "$probe")
actual_usage_refusals=$(grep -cE '\|\| usage$' "$probe")
actual_echo_refusals=$(grep -c 'echo "process-tree discovery' "$probe")
# Anti-vacuity: a counter that returns zero is a broken instrument, not a clean result.
if (( actual_instrument_failures == 0 || actual_usage_refusals == 0 || actual_echo_refusals == 0 )); then
  echo "not ok - refusal-branch counter matched nothing; instrument failure, not a verdict" >&2
  exit 1
fi
actual_refusal_branches=$(( actual_instrument_failures + actual_usage_refusals + actual_echo_refusals ))
```

The drift-comparison `if` block and the final `pass_arm` line below it are **unchanged**.
Notes: the count moves 24 → 27 because the gate now **covers** three previously-unguarded branches, not
because this work adds any (`19 + 5 + 3 = 27`). `\$probe` inside the double-quoted `echo` is intentional —
the message must print the literal text `$probe`. No bash-4 constructs: `[[ -n ... ]]`, `[[ -f ... ]]`
and `(( a || b || c ))` are all bash-3.2-safe.

**Acceptance block**:

| # | Command | **Base** | Required after M3 |
|---|---|---|---|
| A3.1 | `bash -n tools/eval/test_motoko_connection_probe.sh; rc=$?` | rc=0 | rc=0 |
| A3.2 | `grep -Fc 'expected_refusal_branches=27' $T` / `grep -Fc 'expected_refusal_branches=24' $T` | 0 / 1 | **1 / 0** |
| A3.3 | `grep -Fc 'actual_echo_refusals' tools/eval/test_motoko_connection_probe.sh` | 0 | **≥ 3** (assignment, anti-vacuity test, sum) |
| A3.4 | `grep -Fc '[[ -f "$probe" ]]' tools/eval/test_motoko_connection_probe.sh` | 0 | **≥ 1** |
| A3.5 | `grep -Fc 'PROBE_MAX_TREE_NODES is set at suite scope' tools/eval/test_motoko_connection_probe.sh` | 0 | **1** |
| A3.6 | `/bin/bash tools/eval/test_motoko_connection_probe.sh > /tmp/m3_suite.log 2>&1; rc=$?` | rc=0, 41 ok, 0 not ok, 50s | rc=0, **41** ok, 0 not ok — the guard lives inside the gate flow, so the **arm count must NOT move** |
| A3.7 | `grep -c 'refusal-branch count still matches the set this suite covers (27)' /tmp/m3_suite.log` | 0 (base log says `(24)`) | **1** |
| A3.8 | `PROBE_MAX_TREE_NODES=50000 /bin/bash tools/eval/test_motoko_connection_probe.sh > /tmp/m3_ambient.log 2>&1; rc=$?` | **rc=0, 41 ok, 0 not ok, 57s** — an ambient global ceiling is invisible at base | **rc=1**, and `/tmp/m3_ambient.log` contains `not ok - PROBE_MAX_TREE_NODES is set at suite scope`, with **40** preceding `ok` lines. This is the runtime twin of A2.3 and the cheap, non-mutation half of T4's evidence. |
| A3.9 | `grep -cE '^(export )?PROBE_MAX_TREE_NODES=' tools/eval/test_motoko_connection_probe.sh` | 0 | 0 — M3 must not itself introduce a suite-scope assignment |

**Commit (controller)**: `test(probe): refusal-branch gate counts echo refusals (24→27) and refuses a leaked ceiling`.

---

### M4 — mutation validation (T1–T4), produced by running

**Design doc**: Implementation Plan Phase 4; Testing Strategy T1–T4.
**Files**: no change to `P` or `T` (each mutant is applied and then restored byte-identically). The
milestone's **product** is the recorded evidence, written into the design doc's Testing Strategy table
as an **"Observed (iteration 32)"** column. That doc edit is the committable artefact.

> **The red sets below are PREDICTIONS the executor must PRODUCE BY RUNNING.** Iteration-31's red sets
> were measured under the OLD design (arm :449 at the default 4096 ceiling) and **do not transfer**.
> Do not copy a predicted number into the record. If a produced result differs from the prediction,
> **record the produced result and report the divergence** — that is a finding, not a failure to fix.

**Mutation protocol — identical for all four, and non-negotiable:**

```bash
# 0. pre-hash and back up (cp, NOT git)
shasum -a 256 <file>                                  # record as SHA_PRE
cp <file> /tmp/it32.<tag>.bak

# 1. apply the mutant (sed / editor), then prove it LANDED
shasum -a 256 <file>                                  # must DIFFER from SHA_PRE

# 2. prove it PARSES
bash -n <file>; rc=$?                                 # must be rc=0

# 3. prove it had its INTENDED EFFECT — a query against the SYSTEM's own view,
#    not against the file's bytes (see each row for the specific query)

# 4. run the suite, capturing rc WITHOUT a pipe
/bin/bash tools/eval/test_motoko_connection_probe.sh > /tmp/it32.<tag>.log 2>&1; rc=$?

# 5. restore from the cp backup and PROVE the restore
cp /tmp/it32.<tag>.bak <file>
shasum -a 256 <file>                                  # must EQUAL SHA_PRE
```

Step 5 is mandatory **before** the next mutant is applied. Never run two mutants at once. Never use
`git checkout --` at step 5.

Note on all four red sets: `expect_failure` and the gate both `exit 1` on the first failure, so the
suite **aborts at the failing arm**. The expected shape is therefore *"N `ok` lines, then one
`not ok`, then nothing"* — **not** "41 minus one".

#### T1 — neuter the in-loop wall clock (the load-bearing acceptance)

- **File**: `P`. **Mutant**: at `P:186`, `if (( $(date +%s) > deadline )); then` →
  `if false && (( $(date +%s) > deadline )); then`.
- **Effect assertion (the system's own view, NOT the file's bytes)**: a grep for the injected
  `if false` text would only prove the edit exists, which step 1's sha256 already proves. The sound
  assertion is **the mutant run's own stderr**, read out of `/tmp/it32.t1.log`:
  it **must contain** `process-tree discovery exceeded 50000 nodes` — the ceiling's message, which can
  only appear if the wall clock did *not* fire — and it **must not contain**
  `process-tree discovery deadline expired (wall clock)`. Two greps over the log, both read from the
  run. (Run these *after* step 4; T1 is the one row whose effect assertion is necessarily
  post-execution.)
- **Run it with `run_in_background: true`** and poll `/tmp/it32.t1.log`. Budget **280–340s**;
  under a load spike, up to the 600s widened arm cap.
- **Predicted, to be produced**: rc=1; **32 `ok` lines then**
  `not ok - descendant discovery refuses on the real wall-clock deadline lacked expected message: process-tree discovery deadline expired (wall clock)`,
  followed by the captured stderr showing `process-tree discovery exceeded 50000 nodes`.
- **Cap-shaped fallback, explicitly allowed and explicitly recorded**: if the arm instead reports
  `exceeded its 600s arm cap`, that is **still a red and still satisfies T1's verdict** — only its
  diagnosticity degrades (design doc, decision (c): "It does not stop being a red"). Record it as
  **cap-shaped** and say so; do not present it as message-shaped. The optional confirmation run is
  `PROBE_SELFTEST_ARM_CAP_SECS=600 /bin/bash <suite>` (a `:-` default at `T:9`, so this is an
  environment override that touches no file and does **not** trip the ceiling-locality guard, which
  looks only at `PROBE_MAX_TREE_NODES`). If used, record **both** runs.
- **Records**: red set as run, wall time of the whole suite, wall time of arm 33 if separable, and the
  `uptime` load average at run time (the rate, and therefore the duration, is load-dependent).

#### T2 — neuter the node ceiling only (proves the two arms are separable)

- **File**: `P`. **Mutant**: at `P:194`, `if (( visited > MAX_TREE_NODES )); then` →
  `if false && (( visited > MAX_TREE_NODES )); then`.
- **Effect assertion**: the run's own output — arm 40 must fail for **lacking** the ceiling message,
  which is only possible if the ceiling branch is dead.
- **Predicted, to be produced**: rc=1; **arm 33 still `ok`** (this is the point of the row — the
  wall-clock arm no longer depends on the node ceiling), then **39 `ok` lines then**
  `not ok - descendant discovery refuses on the node-count ceiling lacked expected message: process-tree discovery exceeded 3 nodes`.
  Arm 40's walk now ends on its own `PROBE_TIMEOUT_SECS=60` wall clock, so **budget ~60s for that arm
  alone**; whole-suite ~110s. Iteration 31's E7 *shape* (39 ok, 1 not ok, only arm 40) is the
  prediction; its timings are not.
- **Run** in the foreground with a 300000 ms tool timeout.

#### T3 — addition-shaped mutant (proves the gate LOOKS, not merely FIRES)

- **File**: `P`. **Mutant**: insert immediately after `P:128` (the `PROBE_MAX_TREE_NODES` validation,
  which is after `instrument_failure` is defined at `P:4`) one dead-but-real refusal branch:
  ```bash
  [[ -z "${PROBE_TEST_T3_MUTANT_BRANCH:-}" ]] || instrument_failure "T3 mutation-validation branch"
  ```
  The guard is never satisfied in any arm, so **no arm's behaviour changes** — only the gate's count.
  A removal-shaped mutant would prove only that a check fires; this addition is what proves it looks.
- **Effect assertion (system's view)**: `grep -c 'instrument_failure "' "$P"` reports **20** where base
  reported 19 — read through the same expression the gate itself uses.
- **Predicted, to be produced**: rc=1; **40 `ok` lines then**
  `not ok - refusal-branch drift: probe has 28 refusal branches,` followed by the gate's two
  continuation lines naming 27. **Fast run** (~55s) — the added branch never executes.
- **Run** in the foreground with a 300000 ms tool timeout.

#### T4 — scoping-locality mutant (proves "scoped" is enforced, not decorative)

- **File**: `T`. **Mutant — the token survives; only its SCOPE moves.** Two coupled edits:
  1. **delete** ` PROBE_MAX_TREE_NODES=50000` from arm :449's `env` line;
  2. **add** `export PROBE_MAX_TREE_NODES=50000` near the top of the file (immediately after `T:9`,
     the `ARM_CAP_SECS` assignment).
- **Why this shape and not a removal**: on the happy path a global export is behaviourally
  **indistinguishable** from the scoped form — arm 40's own `env`-line `=3` still wins, and the wall
  clock still fires first on every arm — so a plain removal proves nothing on a green run. Only the
  moved-scope shape can distinguish "scoped to one arm" from "global", and A3.8's base measurement
  (`rc=0, 41 ok` with an ambient 50000 at base) is the direct evidence that the distinction is otherwise
  invisible.
- **Effect assertion (system's view + statics)**: `grep -cE '^(export )?PROBE_MAX_TREE_NODES=' "$T"`
  flips **0 → 1** (S7/A2.3 reds), and `grep -n -B1 'PROBE_MAX_TREE_NODES=50000' "$T"` no longer shows
  the arm :449 `env` line — i.e. A2.4's locality claim reds too.
- **Predicted, to be produced**: rc=1; **40 `ok` lines then a NAMED red** —
  `not ok - PROBE_MAX_TREE_NODES is set at suite scope; the ceiling override must stay on arm env lines`.
  ~55s. **This must be the guard's message, not a drift or arm failure**; any other red shape means the
  guard is in the wrong place in the gate flow.
- **Run** in the foreground with a 300000 ms tool timeout.

#### M4 acceptance block

| # | Criterion | **Base** | Required |
|---|---|---|---|
| A4.1 | For each of T1–T4: SHA_PRE recorded, post-mutation sha256 **differs**, `bash -n` **rc=0**, effect assertion holds, post-restore sha256 **equals** SHA_PRE | The protocol's own steps are all base-runnable (`shasum`, `cp`, `bash -n` all rc=0 at base) | all four complete, all four restored byte-identically |
| A4.2 | `shasum -a 256 tools/eval/motoko_connection_probe.sh tools/eval/test_motoko_connection_probe.sh` after **all** mutants are restored | `7a75c698…` / `bd5302f5…` at base | equals the **post-M3** hashes (recorded at M3 acceptance), **not** the base hashes — M1–M3 legitimately changed both files |
| A4.3 | Red set for each of T1–T4 recorded **as run** (rc, `ok` count, the exact `not ok` line, wall time, `uptime` at run time) | n/a — must be produced | four recorded red sets, each rc=1 |
| A4.4 | Final clean re-run after all restores: `/bin/bash tools/eval/test_motoko_connection_probe.sh > /tmp/m4_final.log 2>&1; rc=$?` | rc=0, 41 ok, 50s | rc=0, 41 ok, 0 not ok, gate line shows `(27)` |
| A4.5 | Design doc Testing Strategy table gains an **"Observed (iteration 32)"** column populated from A4.3, and any prediction/observation divergence is stated in prose | n/a | present |
| A4.6 | `git status --porcelain` lists **only** the two shell files, the design doc and this plan — **no `.bak` files, no stray logs in the tree** (keep all `.bak` and `.log` files under `/tmp`). The sprint JSON does **not** appear: `.ailang/` is gitignored (`.gitignore:82`), which is why it is not in the base line count either. | 1 line: ` M design_docs/planned/m-motoko-discovery-arm-discriminating-refusal.md` | as stated |

**Commit (controller)**: `docs(mission): record iteration-32 mutation validation (T1–T4) for row 6n`.

---

### Out of plan, but named by the design doc

Decision (e) says the #971 hand-over is **a message to the V1 mission**, not a rebase, and Non-Goals
says **motoko does not touch #971**. That message is **not** one of the doc's four implementation
phases, so it is not a milestone here. It is left to the controller as a post-sprint action, and the
executor must not attempt it. `PROBE_TREE_DISCOVERY_SECS` was re-confirmed **absent** at this HEAD
(0 occurrences in both files; control `PROBE_TIMEOUT_SECS` = 3 in `P`, 12 in `T`), so #971 has not
landed and no rebase question arises during this sprint.

---

## 3. Sequencing and dependencies

```
M1 (probe messages)  ──► M2 (arm :449)  ──► M3 (gate)  ──► M4 (mutation validation)
   green: 41 ok           green: 41 ok       green: 41 ok      green: 41 ok after restore
```

- **M2 depends on M1**: reversing the order reds arm 33 (it would assert a string the probe does not
  yet emit). M3 is content-independent of M1/M2 but is sequenced last among the code milestones so that
  every intermediate commit is green.
- **M4 depends on all three**: T1's predicted ceiling message (`exceeded 50000 nodes`) requires M2's
  scoped override, and T3/T4 exercise M3's gate.
- **Bisect property**: `git bisect` across M1→M2→M3 never lands on a commit where
  `/bin/bash tools/eval/test_motoko_connection_probe.sh` is red. This is asserted by A1.8, A2.11 and
  A3.6 respectively, each of which must be **run**, not assumed.

## 4. Effort

| Milestone | Edits | Run time | Wall estimate |
|---|---|---|---|
| M1 | 2 lines in `P` | 1 suite run (~50–60s) | ~15 min |
| M2 | 2 lines in `T` | 1 suite run (~50–60s) | ~20 min |
| M3 | ~15 lines in `T` | 2 suite runs (~120s) | ~35 min |
| M4 | 0 product lines; 4 mutants + restores; 1 doc table | 5 suite runs, one of them 280–340s | ~2.0–2.5 h |
| **Total** | **~20 lines across 2 files** | ~11–12 min of suite time | **~3.5 h** |

This matches the design doc's own estimate (~3–4 h). The dominant cost is T1's walk to the 50,000-node
ceiling, which is load-dependent.

## 5. Risks

**R1 — T1 runs long enough to be mistaken for a hang, or long enough to go cap-shaped.**
*Measured, not hypothetical.* At the stub rates measured on this rig **at plan time** (166–181 iter/s),
a 50,000-node walk is **276–301s**, i.e. only **2.0–2.2x** under the 600s widened cap. The design doc
states the range as **2.2x–5.7x**; the low end of my measurement sits **marginally below the doc's
stated floor**, purely because ambient load is higher now than when V23/V24 were taken. *Mitigation*:
run T1 in the background with polling; treat >105s as normal; accept and **label** a cap-shaped red
per decision (c); optionally confirm with `PROBE_SELFTEST_ARM_CAP_SECS=600` and record both runs.
*This is a schedule and diagnosticity risk, not a verdict risk — T1 reds either way.*

**R2 — a mutant is left applied.** *Mitigation*: A4.1's sha256-equality restore proof after **every**
mutant, plus A4.2's whole-file hash check and A4.4's clean re-run at the end. The `cp`-backup rule
(never `git checkout --`) is what keeps the restore from also deleting M1–M3.

**R3 — the ceiling override silently leaks to suite scope in a later edit.** *Mitigation*: this is
exactly what M3's guard plus A2.3/A3.9 (static) and A3.8/T4 (runtime) exist for, and A3.8 has a base
measurement proving the hole is real today.

**R4 — line numbers drift between milestones.** M3 adds lines at `T:707+`, i.e. **after** arm :449 at
`T:449-452`, so M2's positions are stable in the M1→M2→M3 order. If the executor deviates from that
order, re-derive positions with `grep -n` rather than trusting the numbers in this plan.

**R5 — a `grep` pattern silently misfires.** *Measured trap*: `grep -c 'MAX_TREE_NODES=${PROBE_MAX_TREE_NODES:-4096}'`
returns a false **0** on this rig (the `${...:-...}` text misparses as a regex); only the `-F` form
returns 1. **Every literal-string criterion in this plan uses `-F`.** Do not drop it.

**R6 — CI runner behaviour is unmeasurable locally.** Per decision (d), the settling measurement is the
"launchd drivers (bash 3.2)" leg of the first PR carrying this work: arm 33 must pass with the new
unique message. **Out of scope for the executor**; noted for the controller when the PR opens.

## 6. Definition of done

- M1, M2, M3 each applied, each acceptance block green **as run**, three controller commits.
- M4's four mutants each applied, asserted (landed / parses / effect), run, red set **recorded as
  produced**, and restored with sha256 proof.
- Final clean suite: rc=0, 41 `ok`, 0 `not ok`, gate arm prints `(27)`.
- `git status --porcelain` shows only `tools/eval/motoko_connection_probe.sh`,
  `tools/eval/test_motoko_connection_probe.sh`, the design doc and this plan. The sprint JSON lives
  under `.ailang/`, which is gitignored (`.gitignore:82`), so it is never listed.
- **Zero git writes by the executor.**
