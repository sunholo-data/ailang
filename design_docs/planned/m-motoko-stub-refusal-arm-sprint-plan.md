# Sprint Plan: M-MOTOKO-STUB-REFUSAL-ARM

**Design doc**: [`design_docs/planned/m-motoko-stub-refusal-arm.md`](m-motoko-stub-refusal-arm.md)
**Sprint JSON**: `.ailang/state/sprints/sprint_m-motoko-stub-refusal-arm.json` (`.ailang/` is gitignored — `.gitignore:82` — so the JSON is deliberately NOT part of the commit; same convention as the row 6n plan)
**Planned**: 2026-09-02 (motoko mission iteration 33)
**Base commit**: `373a12d17`, worktree `/Users/voightkampff/dev/sunholo-data/.wt-motoko-iter33-sprint`, branch `sprint/motoko-iter33-test-stub-arm`, clean
**Target**: v0.34.x · **Risk**: low (one inserted test arm; no production file is modified)
**Design status**: quorum-cleared under the narrow-refinement carve-out (design doc §"Quorum round 2"). **This plan does not re-open the design.**

---

## 0. How to read this plan

**Two milestones.**

- **M1** inserts exactly one `expect_failure` arm into the test file. It leaves the suite green at 43 arms, so it is an independently committable, bisectable point.
- **M2** changes no product code. It runs the mutation drill that proves the M1 arm is non-vacuous, and records the evidence in the design doc.

**Platform.** Every base reading in this plan is **darwin/arm64 (macOS 26.5.2), GNU bash 3.2.57(1)-release, arm64-apple-darwin25**, on this rig, with `/usr/sbin` on PATH (`command -p -v lsof` → `/usr/sbin/lsof`, rc=0). **The windows and ubuntu CI legs are unrun locally.** The `launchd drivers (bash 3.2)` CI leg is the only place the runner's behaviour is observable; nothing in this plan may be reported as "verified on CI" from a local green. A run showing far fewer than 42/43 arms is a PATH problem (no real `lsof`), not a code problem.

**Five standing rules for the executor, and they are not optional:**

1. **The executor makes NO git writes.** No `git add`, no `git commit`, no `git checkout`, no `git stash`, no `git clean`, no branch or worktree operation. The worktree is deliberately uncommitted. **The controller commits**, one commit per milestone, after that milestone's acceptance block is green. If an acceptance block goes red, STOP and report — do not proceed to the next milestone.
2. **Restore mutants from a `cp` backup, never `git checkout --`.** The sprint's own edits are uncommitted, so `git checkout -- <file>` would silently delete them. Every mutation is bracketed by `cp <file> <file>.bak.<tag>` before and `cp <file>.bak.<tag> <file>` after, and every restore is **verified by sha256 equality against the pre-mutation hash**, not assumed.
3. **Keep every mutant, backup and log under `/tmp`.** Nothing this sprint writes may land in the worktree except the two deliverable edits (`T` in M1, the design doc in M2). `git status --porcelain` is an acceptance criterion in both milestones.
4. **Capture exit codes without a pipe**: `cmd > /tmp/out 2>&1; rc=$?`. The controller's interactive shell is zsh and the suite is `/bin/bash`; `${PIPESTATUS[0]}` is not to be used anywhere.
5. **Cumulative snapshot per milestone.** Before running milestone *k*'s acceptance block, copy the current state of `P`, `T` and `DD` into **`.snap/M<k>/`** at the worktree root — cumulative, so `.snap/M2/` holds M1's edit plus M2's. `.snap/` is **not** gitignored (checked: `git check-ignore .snap/` exits 1, and `.gitignore` has no `snap` rule that covers it), so it appears as a single untracked `?? .snap/` line; both `git status` criteria below account for it explicitly. **`.snap/` must never be `git add`ed** — the controller commits only the deliverable files.

Abbreviations used throughout; all worktree-relative paths:

- `P` = `tools/eval/motoko_connection_probe.sh` — **READ-ONLY for this entire sprint.** The design doc's Non-Goal is explicit: no production-semantics change. The only writes to `P` are inside the M2 mutation drill, and they are to `/tmp` **copies**, never to the tree.
- `T` = `tools/eval/test_motoko_connection_probe.sh` — the **only** file M1 modifies.
- `DD` = `design_docs/planned/m-motoko-stub-refusal-arm.md` — the only file M2 modifies.

**Where the design doc and this plan disagree: nowhere.** Every claim in the design doc that this plan makes load-bearing was re-measured on this worktree at `373a12d17` and reproduced. Two points the plan *adds* beyond the doc are flagged inline as **NEW MEASUREMENT** (§1, rows B7 and B10-B12).

---

## 1. Baseline — measured on the UNMODIFIED tree at `373a12d17`, this session

Every acceptance command in this plan was run on the unmodified tree before the plan was written, and its base result is recorded beside the criterion. **No criterion in this plan was found red at base.**

Environment at measurement time:

| Fact | Measured |
|---|---|
| Worktree base | `373a12d17`, `git status --porcelain` → **0 lines (clean)** |
| `sw_vers -productVersion` | `26.5.2` |
| `/bin/bash --version` | GNU bash 3.2.57(1)-release, arm64-apple-darwin25 |
| `uname -sm` | `Darwin arm64` |
| `command -v timeout` / `gtimeout` | **neither exists** — see §Bounded waits |
| load avg / `hw.ncpu` | 39.15 / 39.38 / 39.44 on **16** CPUs — heavily contended during measurement |
| `command -p -v lsof` | `/usr/sbin/lsof`, rc=0 |
| sha256 `P` | `f0b5e02493369099f123c42107850fe062bf60d56ccabb2a7e4690d654aabc99` |
| sha256 `T` | `b0b445897b307d812dcecb0b3130f29f07fc556134fc355ab2cf9bf09ba43cd7` |
| mode `P` / `T` | `-rwxr-xr-x` / `-rwxr-xr-x` |

### Base measurements table

| # | Command (run at base) | Observed at base |
|---|---|---|
| B1 | `/bin/bash -n tools/eval/motoko_connection_probe.sh; echo rc=$?` | rc=0 |
| B2 | `/bin/bash -n tools/eval/test_motoko_connection_probe.sh; echo rc=$?` | rc=0 |
| B3 | `/bin/bash tools/eval/test_motoko_connection_probe.sh > /tmp/base.log 2>&1; rc=$?` (no `PROBE_UNDER_TEST`) | **rc=0, 42 `^ok `, 0 `^not ok`, 0 skip lines**, last line `PASS: 42 probe self-test arms ran` |
| B4 | The same run with `PROBE_UNDER_TEST=<abs path to the tree's own P>` | **byte-equivalent verdict: rc=0, 42 ok, 0 not ok** — so the swap mechanism itself is not what a mutant result measures |
| B5 | Refusal-branch gate (final arm of B3) | passes; `expected_refusal_branches=28` at `T:763`; counters `grep -c 'instrument_failure "' P` = **20**, `grep -cE '\|\| usage$' P` = **5**, `grep -c 'echo "process-tree discovery' P` = **3**; 20+5+3 = 28 |
| B6 | `grep -n 'descendant discovery deadline refuses at the caller' T` | `358:` — the existing caller-message arm; its `run_live` continuation is `T:359`. Insertion point is **immediately after `T:359`** |
| B7 | **M2 mutant** (strip ` (test stub)` from `P:184`, `/tmp` copy, `chmod 755`) run against the **pristine** `T` | **rc=0, 42 ok, 0 not ok** — **the mutant SURVIVES. This is the defect.** Provenance: sha `f0b5e024…` → `df45556c…`, `bash -n` rc=0, mode `-rwxr-xr-x`, `grep -Fc '…(test stub)'` **1 → 0** while `grep -Fc '…(wall clock)'` stays **1**, echo-refusal count stays 3 |
| B8 | **M3 known-positive control** (strip ` (wall clock)` from `P:189`) against pristine `T` | **rc=1, 32 ok, 1 not ok**; failing arm **by name**: `not ok - descendant discovery refuses on the real wall-clock deadline lacked expected message: process-tree discovery deadline expired (wall clock)`. Provenance: sha → `645fbfc0…`, `bash -n` rc=0, `(wall clock)` count 1 → 0 while `(test stub)` stays 1. **The harness DOES fire for this class of edit; B7's green is a measurement, not a broken instrument.** |
| B9 | **T3 mutant** (alter the suffix: `(test stub)` → `(stub)` at `P:184`) against pristine `T` | **NEW MEASUREMENT — rc=0, 42 ok, 0 not ok. This mutant ALSO survives.** So the gap is not specific to deletion: any reword of the stub message lands green today. Provenance: sha → `0e7f60cc…`, `bash -n` rc=0, `(test stub)` count 1 → 0, echo-refusal count still 3 (so the count gate cannot catch it either) |
| B10 | **NEW MEASUREMENT — prototype of the M1 arm** (the exact two lines of §M1 inserted after `T:359` in a `/tmp` copy of `T`) run with `PROBE_UNDER_TEST` = the **pristine** `P` | **rc=0, 43 ok, 0 not ok**, last line `PASS: 43 probe self-test arms ran`. **The proposed arm passes on correct code.** Prototype sha `98d7fe5a…`, `bash -n` rc=0 |
| B11 | **NEW MEASUREMENT — the same prototype `T` against the B7 (strip) mutant** | **rc=1, 25 ok, 1 not ok**; failing arm **by name**: `not ok - descendant discovery stub refusal carries its own message lacked expected message: process-tree discovery deadline expired (test stub)` |
| B12 | **NEW MEASUREMENT — the same prototype `T` against the B9 (reword) mutant** | **rc=1, 25 ok, 1 not ok**; same failing arm by name |
| B13 | `grep -Fq` semantics, re-measured with `E='process-tree discovery deadline expired (test stub)'` | rc=0 on a file containing `E` exactly; **rc=0 on a file containing `E X`** (append-shaped mutants are VACUOUS); rc=1 on `…expired (stub)`; rc=1 on `…expired`. Confirms design-doc V13/T3 |
| B14 | Tree integrity after the entire drill | sha256 of `P` and `T` **unchanged**; `git status --porcelain` → **0 lines**. The drill wrote nothing into the worktree |

**What B7+B9 vs B10+B11+B12 establish, together:** the suite as it stands is blind to *any* edit to the stub's message (deletion B7, reword B9), and the proposed arm converts both of those from green to a named red (B11, B12) while staying green on correct code (B10). That is the whole sprint, already proven at design time. M1 reproduces it in the tree; M2 re-proves it against the tree's own file rather than a `/tmp` prototype.

**Note on the 25-vs-42 arm counts in B11/B12.** The harness is **fail-fast**: `expect_failure` ends in `exit 1` on both failure paths (`T:161` `unexpectedly succeeded`, `T:166` `lacked expected message`). The first failing arm terminates the suite and every later arm is unreached. So a red run's `ok` count is a *position*, not a coverage number, and 25 < 42 is expected rather than alarming. This is why every mutation criterion below names the **failing arm by name** instead of asserting an exit code or an arm count.

### Bounded waits

There is **no `timeout(1)` and no `gtimeout` on this machine**, so every bound comes from the harness, not from a shell wrapper.

- Short commands (`grep`, `bash -n`, `sed`, `awk`, `cp`, `shasum`, `stat`): the default tool timeout is ample.
- **Full suite runs**: timed this session at load average ≈ 39 on 16 CPUs — base **53 s** and **52 s**, the B10 prototype (43 arms) **50 s**. Budget ~60 s and run each with an explicit tool timeout of **300000 ms**. The extra arm costs no measurable time (the stub short-circuits before the walk).
- **The M2 drill runs four suites back to back** (D1 strip-mutant, D2 reword-mutant, D3 wall-clock control, and the A2.1 pristine re-run) ≈ **3–4 minutes** at the timings above. Run the drill as a **single background script that appends to one artifact file**, then poll that artifact inside a bounded `date +%s` loop. Do not run five foreground suites in sequence.
- **A multi-minute M2 is EXPECTED, not a hang.** None of the arms in this sprint's path are timing-sensitive: the stub short-circuits before the process-tree walk, so the new arm is deterministic and fast.

---

## 2. Milestones

### M1 — test: give the stub refusal branch an arm that asserts its own message

**Design doc**: Goals (Primary); High-Impact Decision (A); Acceptance Criteria AC3.
**Files**: `T` only. **+2 lines, 0 lines changed, 0 lines deleted.** `P` is not touched.

**The change.** Insert these **exactly two lines** immediately after `T:359` (i.e. after the existing caller-message arm's `run_live` continuation line, and before `expect_failure "lane sampling deadline refuses" …`):

```bash
expect_failure "descendant discovery stub refusal carries its own message" "process-tree discovery deadline expired (test stub)" \
  run_live PROBE_TEST_DESCENDANT_FAILURE=1
```

Byte-exact requirements, all of which the B10 prototype satisfied:

- Line 1 starts at **column 0** (no leading whitespace), matching every other `expect_failure` in the `run_live` block.
- Line 2 is indented **two spaces**, matching `T:359`.
- The continuation backslash on line 1 is the **last** character of the line (no trailing space after it) — a trailing space breaks line continuation in bash and would be caught by AC1.1.
- The arm name is `descendant discovery stub refusal carries its own message`. It must be **distinct** from the existing arm name at `T:358` so a red is attributable; AC1.4 is what enforces that the two arms are separately identifiable in the log.
- The asserted string is `process-tree discovery deadline expired (test stub)` — character-for-character the string at `P:184`, including the single space before `(`.

**What must NOT change.** `T:358-359` (the existing caller-message arm) stays byte-identical — the design doc's decision (A) is that the new arm sits **BESIDE** it, not in place of it. `T:763` `expected_refusal_branches=28` stays **28**: this change adds a self-test *arm*, not a production refusal *branch*, so the gate's VALUE must not move (its arm NUMBER moves 42 → 43 as a consequence of inserting an arm ahead of it — that is expected and is what AC1.2 reads). `T:319-321` (`run_live`) is untouched. The wall-clock arm (`T:476`) and node-ceiling arm (`T:731`) are far from the insertion point and are untouched.

**Why this is green on its own.** With `PROBE_TEST_DESCENDANT_FAILURE=1` the probe's stderr carries **both** the stub message and the caller wrapper (design doc V7; reproduced end-to-end here by B10, where the new arm passed against the pristine probe). `expect_failure` requires only `rc != 0` plus a fixed-string substring hit anywhere in the captured stderr (`T:151-169`, B13), both of which hold.

**Blast radius, classified BEFORE choosing criteria.** M1 adds one arm and changes no behaviour of any existing arm. The only observable it can move is the suite's **arm count and final line** (42 → 43). It cannot move the refusal-branch gate's value, the probe's parse, or any other arm's verdict. Therefore the narrowest gate that can actually fail for this diff is *the suite's own arm accounting plus a clean green* — AC1.2 — and everything else in the block is a guard against collateral damage.

**Snapshot.** Before running the acceptance block: `mkdir -p .snap/M1 && cp tools/eval/motoko_connection_probe.sh tools/eval/test_motoko_connection_probe.sh design_docs/planned/m-motoko-stub-refusal-arm.md .snap/M1/`.

**Acceptance block** — run all of these; every one must hold. **No criterion here is a static grep for a string in a file** (that shape is the exact defect this sprint exists to remove); every one reads either a parse result, the suite's own runtime output, or the working tree's cleanliness.

| # | Command | **Base (measured)** | Required after M1 |
|---|---|---|---|
| A1.1 | `/bin/bash -n tools/eval/test_motoko_connection_probe.sh; echo rc=$?` | rc=0 (B2) | **rc=0** — catches a trailing space after the continuation backslash, an unbalanced quote, or a broken insertion |
| A1.2 | `/bin/bash tools/eval/test_motoko_connection_probe.sh > /tmp/m1_suite.log 2>&1; rc=$?` then `grep -c '^ok ' /tmp/m1_suite.log`, `grep -c '^not ok' /tmp/m1_suite.log`, `tail -1 /tmp/m1_suite.log` | rc=0, **42** ok, 0 not ok, `PASS: 42 probe self-test arms ran` (B3) | **rc=0, 43 ok, 0 not ok, `PASS: 43 probe self-test arms ran`** — behavioural: the suite RAN and PASSED one more arm. 42 means the arm was not added; 44+ means more than one arm was added |
| A1.3 | `grep -ci 'skip' /tmp/m1_suite.log` | 0 (B3) | **0** — no arm may be skipped into a false green |
| A1.4 | `grep -cE '^ok [0-9]+ - descendant discovery stub refusal carries its own message$' /tmp/m1_suite.log` **and** `grep -cE '^ok [0-9]+ - descendant discovery deadline refuses at the caller$' /tmp/m1_suite.log` | **0 and 1** | **1 and 1** — reads the suite's own emitted pass lines, so it proves the new arm EXECUTED and that the pre-existing caller arm is still executing beside it (design doc decision (A): beside, not instead of). Anti-vacuity: a base of 0 for the first and 1 for the second means both counters are live. **`pass_arm` (`T:75-78`) emits `ok $arms - $1`, i.e. the arm NUMBER is present — a pattern written as `^ok - …` matches nothing and would be a vacuous criterion.** In the B10 prototype these lines were `ok 25 - descendant discovery deadline refuses at the caller` and `ok 26 - descendant discovery stub refusal carries its own message` |
| A1.5 | The refusal-branch gate arm, read out of the same log: `grep -cF 'ok 43 - refusal-branch count still matches the set this suite covers (28)' /tmp/m1_suite.log` | 0 at base — at base the same line reads `ok 42 - … (28)` (B3, B5) | **1** — the gate still passes with the value **28** while its arm NUMBER has moved 42 → 43, which is exactly the design doc's M5 prediction. If it printed `(29)` or reported drift, a production branch was added, which this sprint forbids. Confirmed in the B10 prototype log: `ok 43 - refusal-branch count still matches the set this suite covers (28)` |
| A1.6 | `shasum -a 256 tools/eval/motoko_connection_probe.sh` | `f0b5e024…aabc99` (B-env) | **unchanged** — the production probe is read-only this sprint (design doc Non-Goal) |
| A1.7 | `git status --porcelain` | 0 lines (clean — the plan and design doc are committed by the controller before the executor starts) | **exactly 2 lines**, in any order: ` M tools/eval/test_motoko_connection_probe.sh` and `?? .snap/`. **No `.bak`, no `.log`, no mutant copies inside the tree.** Anything else in the list is collateral damage and reds this criterion |

**Commit (CONTROLLER, not executor)**: `test(probe): assert the stub refusal's own message (motoko iter-33, row 6r)`.

---

### M2 — mutation drill: prove the M1 arm is the sole killer, and record it

**Design doc**: Test Plan T1/T2/T3; Acceptance Criteria AC5.
**Files**: `DD` only (an appended evidence section). **No product code changes.** **Depends on M1.**

This is the load-bearing milestone. M1 without M2 is a plausible-looking arm with no proof it can fail.

**Standing rules for every mutation in this drill** (each is a step the executor must actually perform and record, not a caveat):

1. **Never mutate the tree.** `cp tools/eval/motoko_connection_probe.sh /tmp/iter33/probe.pristine.bak` once, then build every mutant as a **separate file under `/tmp`** and run the suite with `PROBE_UNDER_TEST=<mutant path> /bin/bash tools/eval/test_motoko_connection_probe.sh`. `T:5` is `probe=${PROBE_UNDER_TEST:-$script_dir/motoko_connection_probe.sh}`, so this swaps the probe under test without touching the tree. **`T` itself must NOT be copied out of the tree** — `script_dir` is only used to resolve `$probe`, but running a `/tmp` copy of `T` would resolve `$probe` to a non-existent `/tmp/motoko_connection_probe.sh` unless `PROBE_UNDER_TEST` is also set; keep `T` in place and vary only the probe.
2. **`chmod 755` every mutant, and prove it.** `sed > file` creates mode **644**, and the suite invokes `"$probe" --classify-fixture …` **directly** at `T:201`. An un-chmod'd mutant therefore reds at the **FIRST** arm (`not ok - classification fixture: missing loopback 127.0.0.1:11434`, ok=0) for its file **mode**, not for the mutation — a false kill. Record `stat -f '%Sp' <mutant>` = `-rwxr-xr-x` for each.
3. **Prove each mutant LANDED, PARSES, and had its INTENDED EFFECT** before believing its suite verdict:
   - LANDED: `shasum -a 256` of the mutant **differs** from the pristine hash `f0b5e024…aabc99`.
   - PARSES: `/bin/bash -n <mutant>; rc=0`.
   - EXECUTABLE: `stat -f '%Sp'` = `-rwxr-xr-x`.
   - INTENDED EFFECT, read against the system's own view: the count of the targeted string goes **1 → 0** while the **sibling suffix's count is unchanged at 1**. A mutant where both counts move is a botched `sed`, not an experiment.
4. **Add-shaped mutants are forbidden** (B13). Under `grep -Fq` a mutant that only *appends* around the asserted substring leaves the substring present and is vacuous. Every mutant below **removes or alters** the asserted substring.
5. **Read WHICH arm failed, never the exit code alone.** Rules 2 and 4 both produce non-zero exit codes for reasons that are not the mutation.
6. **Restore from the `cp` backup and prove byte-identity.** After the drill, `cp /tmp/iter33/probe.pristine.bak tools/eval/motoko_connection_probe.sh` **only if the tree hash ever changed** (it should not — the drill only writes to `/tmp`), and in all cases assert `shasum -a 256 tools/eval/motoko_connection_probe.sh` = `f0b5e024…aabc99`. **Never `git checkout -- <file>`**: M1's edit to `T` is uncommitted at drill time and `git checkout` would destroy it.

**The three mutants, with the assertion each must move and the expected blast radius classified in advance:**

| Mutant | Edit (against a `/tmp` copy of `P`) | Effect proof (system's own view) | Expected blast radius | Required verdict |
|---|---|---|---|---|
| **D1 = T1/M2 (the defect)** | strip ` (test stub)` — `P:184` becomes `process-tree discovery deadline expired` | `grep -Fc 'expired (test stub)'` **1 → 0**; `grep -Fc 'expired (wall clock)'` **stays 1**; `grep -c 'echo "process-tree discovery'` **stays 3** (so the count gate cannot be the catcher) | The M1 arm only. The caller-message arm at `T:358` asserts `process-tree discovery failed`, which is untouched, so it stays green and the fail-fast harness REACHES the new arm | **rc=1**, and the failing line is exactly `not ok - descendant discovery stub refusal carries its own message lacked expected message: process-tree discovery deadline expired (test stub)`. **Base (B7): this same mutant is rc=0, 42 ok, 0 not ok — it SURVIVES.** The 0→1 flip in that arm's kill status IS the sprint's result |
| **D2 = T3 (reword)** | alter the suffix — `P:184` becomes `… deadline expired (stub)` | `grep -Fc 'expired (test stub)'` **1 → 0**; `grep -c 'echo "process-tree discovery'` **stays 3** | Same as D1 | Same failing line as D1. **Base (B9): survives, rc=0, 42 ok.** Proves the arm pins the *exact* string, not merely the presence of a suffix |
| **D3 = T2 (known-positive control)** | strip ` (wall clock)` — `P:189` becomes `process-tree discovery deadline expired` | `grep -Fc 'expired (wall clock)'` **1 → 0**; `grep -Fc 'expired (test stub)'` **stays 1** | The **wall-clock** arm (`T:476`), which sits AFTER the new arm, so under fail-fast the new arm passes first and the wall-clock arm is the catcher | **rc=1**, failing line `not ok - descendant discovery refuses on the real wall-clock deadline lacked expected message: process-tree discovery deadline expired (wall clock)`. **Base (B8): identical verdict.** This is the control that shows the harness's ability to red for this class of edit is unchanged by M1 — a mutant that must NOT be attributed to the new arm |

**Mutants deliberately NOT run, and why** (design doc T4/T5, upheld verbatim from `gemini-3-1-pro` round 2): removing the stub branch entirely (T4) and rewording the caller wrapper (T5) both red the **caller-message arm at `T:358` first**, which `exit 1`s and aborts the suite **before the new arm executes**. The mutation is still caught, but the new arm is not the catcher, so these produce no information about M1 and would only invite a mis-attribution. `-skip`-style "everything else stayed green" inverse arms are **not expressible in this shell harness** at all — there is no way to run the remaining arms after a failure — which is why every row above names the expected FAILING ARM BY NAME rather than asserting an exit code or a residual arm count.

**Recording step.** Append a `## Mutation Evidence (executed, motoko iteration 33)` section to `DD` containing, per mutant: the sed expression, the pristine and mutant sha256, the `bash -n` rc, the mode, the 1→0 effect count with its unchanged sibling, the suite rc, and **the verbatim failing line**. Also record the pristine re-run (see A2.1) and the tree-integrity check. Do not restate predictions — record only what was observed.

**Snapshot.** Before running the acceptance block: `mkdir -p .snap/M2 && cp tools/eval/motoko_connection_probe.sh tools/eval/test_motoko_connection_probe.sh design_docs/planned/m-motoko-stub-refusal-arm.md .snap/M2/` (cumulative: `.snap/M2/`'s copy of `T` already contains M1's edit, and `.snap/M1/` is left in place).

**Acceptance block:**

| # | Command | **Base (measured)** | Required after M2 |
|---|---|---|---|
| A2.1 | Re-run the suite against the **pristine** probe **after** the whole drill: `PROBE_UNDER_TEST="$PWD/tools/eval/motoko_connection_probe.sh" /bin/bash tools/eval/test_motoko_connection_probe.sh > /tmp/m2_pristine.log 2>&1; rc=$?` | rc=0, 42 ok (B4 — the swap mechanism is itself neutral) | **rc=0, 43 ok, 0 not ok**, `PASS: 43 probe self-test arms ran`. The drill left nothing behind |
| A2.2 | D1 verdict: `grep -cF 'not ok - descendant discovery stub refusal carries its own message lacked expected message: process-tree discovery deadline expired (test stub)' /tmp/m2_d1.log` | **0** (B7: at base this mutant produced ZERO `not ok` lines at all — the defect). Note `expect_failure` emits `not ok - $name …` with **no** arm number, unlike `pass_arm` | **1**, and D1's suite rc=1 |
| A2.3 | D2 verdict: the same `grep -cF` against `/tmp/m2_d2.log` | **0** (B9: survives at base) | **1**, and D2's suite rc=1 |
| A2.4 | D3 control verdict: `grep -cF 'not ok - descendant discovery refuses on the real wall-clock deadline lacked expected message: process-tree discovery deadline expired (wall clock)' /tmp/m2_d3.log` **and** `grep -cE '^ok [0-9]+ - descendant discovery stub refusal carries its own message$' /tmp/m2_d3.log` | 1 and 0 (B8 — at base the second pattern cannot match, the arm does not exist yet) | **1 and 1** — the wall-clock arm is the `not ok`, and the new arm appears in D3's log as a PASS. This is the attribution check: D3 must NOT be killed by the new arm |
| A2.5 | Mutant hygiene, per mutant: sha256 ≠ pristine; `/bin/bash -n <mutant>` rc=0; `stat -f '%Sp'` = `-rwxr-xr-x`; targeted count 1→0 with the sibling count unchanged at 1 | n/a (drill-internal) | all hold for D1, D2, D3. **Any mutant failing this is discarded and rebuilt — its suite verdict is not evidence** |
| A2.6 | Tree integrity: `shasum -a 256 tools/eval/motoko_connection_probe.sh` | `f0b5e024…aabc99` | **`f0b5e024…aabc99` — byte-identical.** Restore, if ever needed, came from `/tmp/iter33/probe.pristine.bak`, never from `git checkout` |
| A2.7 | `git status --porcelain` | 0 lines | **exactly 3 lines**, in any order: ` M tools/eval/test_motoko_connection_probe.sh` (M1), ` M design_docs/planned/m-motoko-stub-refusal-arm.md` (M2's evidence section), and `?? .snap/`. **No `.bak`, no `.log`, no mutant copies in the tree** — the drill writes only to `/tmp` |

**Commit (CONTROLLER, not executor)**: `docs(motoko): record the stub-refusal arm's mutation evidence (row 6r)`.

---

## 3. Success metrics

| Metric | Base | Target |
|---|---|---|
| Suite arms passing | 42 | **43** |
| Suite `not ok` | 0 | **0** |
| `expected_refusal_branches` | 28 | **28 (unchanged)** |
| Mutants that survive an edit to the stub message | **2 of 2** (strip B7, reword B9) | **0 of 2** |
| Known-positive control still attributed correctly | wall-clock arm (B8) | wall-clock arm, **not** the new arm |
| Production files modified | — | **0** (`P` untouched) |
| LOC | — | **+2** in `T`, plus an evidence section in `DD` |

**No example file is required.** This is a test-harness change to a shell self-test; it adds no AILANG language feature, so the CLAUDE.md "every language feature needs `examples/feature_name.ail`" rule does not apply. No CHANGELOG or website change is warranted either — nothing user-facing moves.

---

## 4. Risks

| # | Risk | Likelihood | Mitigation |
|---|---|---|---|
| R1 | Executor inserts the arm at the wrong offset (e.g. after `T:358` instead of `T:359`), splitting the existing arm from its `run_live` continuation | low | A1.1 (`bash -n`) catches a severed continuation; A1.4 catches a missing or renamed caller arm; the exact insertion point and byte-exact text are given in §M1 |
| R2 | Executor "helpfully" replaces the caller-message arm instead of adding beside it | low | A1.4's second counter (`^ok - descendant discovery deadline refuses at the caller$` = 1) reds. The design doc's decision (A) is explicit that the arms are complementary |
| R3 | A mutation drill step reds for file MODE rather than for the mutation, and is misread as a kill | **medium — this has already happened once on this rig** | Rule 2 of §M2 plus A2.5: `chmod 755` and record `stat -f '%Sp'`; and A2.2/A2.3 match the failing line **by name**, so a mode failure (which reds at the classification-fixture arm) cannot satisfy them |
| R4 | An add-shaped mutant is used and produces a vacuous green | low | Rule 4 plus B13's measurement; all three mutants remove or alter the substring |
| R5 | Rig contention (load avg ≈ 39 on 16 CPUs at planning time) stretches the drill and a suite run is misread as hung | medium | §Bounded waits: ~40 s per run measured, 5 runs ≈ 3–5 min, run in background and poll the artifact in a bounded loop. There is no `timeout(1)` on this machine |
| R6 | Concurrent edit collides in the `run_live` arm block (`T:330-363`) | low | Design doc §Conflict Surface: the insertion point is inside that block; any concurrent add/remove/reorder there, or a change to `run_live` at `T:319-321`, conflicts. `P` is untouched, so it has no collision surface this sprint |
| R7 | Local green is mistaken for CI green | medium | §0 Platform: all readings are darwin/arm64 bash 3.2.57. Windows and ubuntu legs are unrun locally; the `launchd drivers (bash 3.2)` CI leg is the only observation of the runner |

---

## 5. Residuals — found, and deliberately left

- **Queue row 6o** — the SIGKILL-escalation group form and the `REAL_LSOF` PATH assertion. Out of scope; not touched.
- **Queue row 6p** — deriving the node ceiling from an in-test stimulus. Out of scope; not touched.
- **The wall-clock arm's accepted structural weakness** (it can die on the arm cap rather than on a message mismatch under a full de-race revert) is documented in `T`'s own comment and is not re-litigated. The new stub-message arm has no such weakness: the stub short-circuits before the walk.
- **The row 6n plan's static-grep acceptance criteria** (`m-motoko-discovery-arm-discriminating-refusal-sprint-plan.md`, AC A1.3/A1.4/A1.5 etc.) are exactly the shape this sprint is replacing. They are historical record and are **not** edited by this sprint; the point is that after M1 the `(test stub)` string is held by a behavioural arm rather than by that grep. Retro-fitting the row 6n plan's other static greps into behavioural arms is a separate, larger piece of work and is **not** attempted here.
- **`expect_failure` cannot express "the rest of the suite stayed green"** because it `exit 1`s on first failure. A harness change that collected failures instead of aborting would make inverse assertions expressible — genuinely useful, and genuinely out of scope for a two-line sprint. Recorded as a residual, not planned.

---

## 6. Handoff

**Executor**: follow §0's five standing rules literally. M1 then M2, in order. Stop and report on the first red acceptance criterion.
**Controller**: commits, one per milestone, after that milestone's acceptance block is green.

**SPRINT_PLAN_PATH**: `design_docs/planned/m-motoko-stub-refusal-arm-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_m-motoko-stub-refusal-arm.json`
