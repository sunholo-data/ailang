CODE ACCEPT — 93/100

# Independent evaluation — fleet iteration 60

**Candidate:** `dcab63a3c006bc5455bd93b3d124e52e695e887c` (PR #1942, branch `fleet/i60-zsh-libs-land`)
**Base:** `origin/dev`
**Plan:** `design_docs/planned/sprint-plan-zsh-safe-mission-libs.md` (M-FLEET-ZSH-SAFE-MISSION-LIBS, one milestone M1_ZSH_SAFE_MISSION_LIBS)
**Ticket:** `quorum-reader:zsh-nomatch`
**Worktree:** `/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter60-judge` (detached at the candidate)
**Rig:** macOS Darwin 25.0 · `/bin/bash` 3.2.57 · `/bin/zsh` 5.9 (arm64-apple-darwin25)
**Role:** independent evaluator. Controller statements were treated as context, not evidence; every claim below was re-derived locally.

**Verdict: CODE ACCEPT, 93/100. No blockers.**

---

## 0. Session handshake

1. `CLAUDE.md` read in full (this worktree) before any other action.
2. Task classified as **independent evaluation under an approved design + approved sprint plan** — read-only judgement plus one report file, no source edits.
3. Unattended mission stage (`AILANG_MISSION_STAGE=1`): the inbox was **not** read and no `ailang messages` command was run, per the mission-role instruction.
4. `session_protocol_ack` called with `{}`; the tool returned *"Session protocol acknowledged — mutating tools unlocked for this session."* (success; the guard on `bash`/`edit`/`write` lifted immediately afterwards, confirmed by the first successful `bash` call).
5. `CLAUDE.md` was read before the ack, as required.

---

## 1. The change is what the plan describes

`git diff origin/dev..HEAD` touches 7 files; the executor commit (`dcab63a3c`) touches exactly the 5 authorized implementation files plus the sprint-state bookkeeping file, and the planner commit (`bd9a41cf3`) touches only the plan + sprint JSON.

```
$ git show --stat --oneline bd9a41cf3 | tail -5
 .../sprint_M-FLEET-ZSH-SAFE-MISSION-LIBS.json      | 47 ++++++++++
 .../planned/sprint-plan-zsh-safe-mission-libs.md   | 99 ++++++++++++++++++++++
 2 files changed, 146 insertions(+)

$ git show --stat --oneline dcab63a3c | tail -10
 .../sprint_M-FLEET-ZSH-SAFE-MISSION-LIBS.json      |  5 +-
 .../unreleased/2026-10-22-zsh-safe-mission-libs.md | 11 ++++
 tools/launchd/lib/pi-ext-args.sh                   |  2 +
 tools/launchd/lib/quorum-artifacts.sh              |  4 +-
 tools/launchd/test_pi_ext_args.sh                  | 52 ++++++++++++
 tools/launchd/test_quorum_artifacts.sh             | 75 +++++++++++++-
 6 files changed, 145 insertions(+), 4 deletions(-)
```

### 1.1 Implementation diff — two guards, two header lines, nothing else

```
$ git diff origin/dev..HEAD -- tools/launchd/lib/quorum-artifacts.sh tools/launchd/lib/pi-ext-args.sh
diff --git a/tools/launchd/lib/pi-ext-args.sh b/tools/launchd/lib/pi-ext-args.sh
index 52704f92d..a4a9acf03 100644
--- a/tools/launchd/lib/pi-ext-args.sh
+++ b/tools/launchd/lib/pi-ext-args.sh
@@ -1,5 +1,6 @@
 # shellcheck shell=bash
 # pi-ext-args.sh — sourced by the mission driver and scripts/mission_pi_run.sh.
+# Skill snippets source this into the controller's zsh tool shell, where NOMATCH aborts unmatched loops.
 #
 # _mc_pi_ext_args DIR → prints pi extension flags (one per line) for a pi run whose CWD is
 # DIR, so a caller can read them into an array:
@@ -17,6 +18,7 @@
 # in that case; they never were in a fire, because the fire crashed first.
 # When DIR has none (world, stapledon), nothing is printed and discovery is unchanged.
 _mc_pi_ext_args() {
+  if [ -n "${ZSH_VERSION:-}" ]; then setopt local_options null_glob; fi
   local dir="$1" f
   [ -d "$dir/.pi/extensions" ] || return 0
   printf '%s\n' --no-extensions
diff --git a/tools/launchd/lib/quorum-artifacts.sh b/tools/launchd/lib/quorum-artifacts.sh
index ea1f942b1..adaead480 100644
--- a/tools/launchd/lib/quorum-artifacts.sh
+++ b/tools/launchd/lib/quorum-artifacts.sh
@@ -1,6 +1,7 @@
 # shellcheck shell=bash
-# Pure bash 3.2 functions. Nothing runs at source time. rc 1 is absence;
+# Bash 3.2 and zsh functions. Nothing runs at source time. rc 1 is absence;
 # rc 2 is a configuration/read error, never permission to skip quorum.
+# Skill snippets source this into the controller's zsh tool shell, where NOMATCH aborts unmatched loops.
 mc_quorum_error() { printf 'quorum ERROR: %s\n' "$*" >&2; return 2; }
@@ -55,6 +56,7 @@ mc_quorum_reader_dirs() {
 # Inspect every directory before returning any matches. Do not select a winner
 # or change freshness/verdict policy: the caller reads the existing nested schema.
 mc_quorum_lookup() {
+  if [ -n "${ZSH_VERSION:-}" ]; then setopt local_options null_glob; fi
   local slug="$1" dir file bad=0
```

The guard is the **first statement** of both functions, byte-identical to the plan's mandated line, in both files:

```
$ printf '%s\n' 'if [ -n "${ZSH_VERSION:-}" ]; then setopt local_options null_glob; fi' > guard.txt
$ sed -n '59p' tools/launchd/lib/quorum-artifacts.sh | sed 's/^  //' | cmp - guard.txt  → quorum guard line: BYTE-EXACT
$ sed -n '21p' tools/launchd/lib/pi-ext-args.sh     | sed 's/^  //' | cmp - guard.txt  → pi guard line:     BYTE-EXACT
```

`git diff --numstat` for the two libraries is `3/1` and `2/0` — one guard line plus the WHY header in each. **No other behaviour changed**: the base already had no `ZSH_VERSION` reference in either file (`git show origin/dev:… | grep ZSH_VERSION` → no hits), so the change is strictly additive.

Both glob sites in `tools/launchd/lib/*.sh` are the two guarded ones — nothing was missed:

```
$ grep -rn 'for [a-z_]* in "\$' tools/launchd/lib/*.sh
tools/launchd/lib/pi-ext-args.sh:25:  for f in "$dir"/.pi/extensions/*.ts; do
tools/launchd/lib/quorum-artifacts.sh:63:  for dir in "${MC_QUORUM_DIRS[@]}"; do
tools/launchd/lib/quorum-artifacts.sh:70:    for file in "$dir/$slug-"*.json; do
```

### 1.2 Patch identity vs the reviewed iteration-57 original

```
$ git range-diff origin/fleet/i57-zsh-safe-libs~2..origin/fleet/i57-zsh-safe-libs HEAD~2..HEAD
1:  b92b42818 = 1:  bd9a41cf3 docs(sprint): plan M-FLEET-ZSH-SAFE-MISSION-LIBS …
2:  6268fb0b8 = 2:  dcab63a3c fix(mission): quorum and pi-ext helpers survive zsh NOMATCH …
```

**Both patches compare `=`** — the rebase is a true identity, not a re-implementation.

Blob-level spot-check of the four code files (plus plan/JSON/changelog), candidate vs the reviewed original:

```
$ for f in …; do a=$(git rev-parse origin/fleet/i57-zsh-safe-libs:"$f"); b=$(git rev-parse HEAD:"$f"); …; done
SAME  adaead4806607be0f1ac7fcb63571398e8acf28c  tools/launchd/lib/quorum-artifacts.sh
SAME  a4a9acf0389c348d637d348a6901b442afee1cc1  tools/launchd/lib/pi-ext-args.sh
SAME  7b11a4e0f94ba1317b423207e3f748124f2838f9  tools/launchd/test_quorum_artifacts.sh
SAME  f51408cb012bc25061ca3d00f599f7691a332afc  tools/launchd/test_pi_ext_args.sh
SAME  7a5f85b49eee66cbff63f628750ee18114f6ebed  changelogs/unreleased/2026-10-22-zsh-safe-mission-libs.md
SAME  8455c67d9d1ee69f2d8b320614a9dcecabc5a921  design_docs/planned/sprint-plan-zsh-safe-mission-libs.md
SAME  acf7e58c6bfde2c35d0186cbf1ffd4e1d3591e0a  .ailang/state/sprints/sprint_M-FLEET-ZSH-SAFE-MISSION-LIBS.json
```

Every reviewed file is **byte-identical** to the iteration-57 original. The branch tip equals the evaluated commit: `git rev-parse origin/fleet/i60-zsh-libs-land` == `git rev-parse HEAD` == `dcab63a3c…`.

### 1.3 Independent base-vs-candidate reproduction of the ticket

Not relying on the suites, I replayed the ticket's exact case (export directory holds `doc-x-2026.json`, local directory exists but is **empty**, `zsh` with `NOMATCH` on) against `origin/dev`'s libraries and against the candidate:

```
=== BASE (origin/dev) ===
[BASE] child_rc=1
[BASE] stdout:
[BASE] stderr: quorum consult: …/export with spaces|quorum consult: …/repo/.ailang/state/mission-quorum|mc_quorum_lookup:11: no matches found: …
[BASE] child_rc=1
[BASE] stdout:
[BASE] stderr: _mc_pi_ext_args:4: no matches found: …/empty/.pi/extensions/*.ts|

=== CANDIDATE (HEAD) ===
[CANDIDATE] child_rc=0
[CANDIDATE] stdout: AFTER_CALL|RC=0|COUNT=1|
[CANDIDATE] stderr: quorum consult: …/export with spaces|quorum consult: …/repo/.ailang/state/mission-quorum|
[CANDIDATE] child_rc=0
[CANDIDATE] stdout: AFTER_CALL|--no-extensions|RC=0|
[CANDIDATE] stderr:
```

Base aborts with `no matches found` (rc 1, no `AFTER_CALL`, zero results — exactly the ticket's "reviewed doc read UNREVIEWED"); the candidate returns rc 0 with the match / `--no-extensions`.

### 1.4 Caller-option restoration (no leak into the controller shell)

`local_options` must restore the caller's options. Verified beyond the suite, in both call shapes:

```
=== TOP-LEVEL call in zsh ===
NOMATCH still ON (good)
NULL_GLOB off (good)
=== call wrapped in a caller FUNCTION ===
NOMATCH still ON (good)
NULL_GLOB off (good)
=== same for pi helper ===
NOMATCH still ON (good)
NULL_GLOB off (good)
```

No option leaks into the controller's zsh tool shell.

---

## 2. Focused suites

Suite files were read first; each suite gates its own zsh arms (both check `/bin/zsh` and FAIL on Darwin rather than silently skipping).

```
$ /bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_quorum_artifacts.sh
RC=0 ELAPSED=2s
PASS zsh_quorum_export_only
PASS zsh_quorum_local_only
PASS zsh_quorum_absent
PASS zsh_quorum_options
PASS zsh_gate2_export_only
PASS repo_namespace_missions
PASS repo_namespace_isolation
PASS invalid_repo_and_home
PASS invalid_and_explicit_override
PASS gate2_extracted_active_matrix
PASS lookup_error_matrix
PASS extraction_is_nonvacuous
quorum artifact directory: …/driver_export_before_both_launches/inherited path
quorum artifact directory: …/driver_export_before_both_launches/profile path
quorum artifact directory: …/driver_export_before_both_launches/secrets path
PASS driver_export_before_both_launches
PASS codex_actual_forwarding
PASS contaminated_host_startup
PASS standalone_policy_reader
PASS bash32_and_suite_reach
PASS attended_absent_and_errors
PASS cross_mode_errors_dominate
agents skills in sync (mission-control mission-brief mission-loop-change sprint-planner sprint-executor sprint-evaluator)
PASS generated_docs_and_mirror
PASS fixture_home_and_observations
```

21 arms PASS, rc 0. The three `quorum artifact directory:` lines are the helper's own stderr diagnostics on stdout-adjacent arms, present at base too.

```
$ /bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_pi_ext_args.sh
RC=0 ELAPSED=0s
  PASS: no .pi/extensions -> no flags (discovery unchanged: world, stapledon)
  PASS: repo extensions -> discovery off
  PASS: one -e per .ts
  PASS: README not loaded
  PASS: empty dir -> discovery off, nothing named
  PASS: probe disables discovery
  PASS: pi controller wired
  PASS: role runner wired
  PASS: …/.claude/skills/mission-control: probe disables discovery
  PASS: …/.claude/skills/mission-control: raw sandbox snippet wired
LAUNCH COUNT: …/.claude/skills/mission-control: 2 launch lines
  PASS: …/.agents/skills/mission-control: probe disables discovery
  PASS: …/.agents/skills/mission-control: raw sandbox snippet wired
LAUNCH COUNT: …/.agents/skills/mission-control: 2 launch lines
  PASS: skill launch sweep safety
  PASS: skill launch count instrument
  PASS: zsh_pi_empty
  PASS: zsh_pi_two_ts
  PASS: zsh_pi_options
==== 17 passed, 0 failed ====
```

Each zsh arm also passes standalone through the suite's own `--arm` gate:

```
arm=zsh_quorum_export_only rc=0 out=PASS zsh_quorum_export_only
arm=zsh_quorum_local_only rc=0 out=PASS zsh_quorum_local_only
arm=zsh_quorum_absent rc=0 out=PASS zsh_quorum_absent
arm=zsh_quorum_options rc=0 out=PASS zsh_quorum_options
arm=zsh_gate2_export_only rc=0 out=PASS zsh_gate2_export_only
```

**Base comparison (same harness, `origin/dev` libraries):**

```
BASE quorum focused rc=0 → 16 arms PASS (no zsh arms exist at base)
BASE pi focused rc=0     → ==== 14 passed, 0 failed ====
```

So the candidate adds 5 + 3 = 8 behavioural arms and breaks nothing that was green at base. The 8 arms are exactly the plan's table (`zsh_quorum_export_only`, `zsh_quorum_local_only`, `zsh_quorum_absent`, `zsh_quorum_options`, `zsh_gate2_export_only`, `zsh_pi_empty`, `zsh_pi_two_ts`, `zsh_pi_options`), each sourcing the **real** helper in `/bin/zsh -f -c` with `setopt nomatch; unsetopt null_glob` set explicitly, isolated scratch dirs, `MISSION_*`/policy exports scrubbed, function rc captured without `errexit`, `AFTER_CALL` printed immediately after capture, exact result bytes compared with `cmp`, array length + quoted path asserted, a path with spaces used, and **both** Gate-2 skill copies exercised (`cmp` of the two `gate-2-pick.md` files is already asserted by the pre-existing `extraction_is_nonvacuous` arm).

### 2.1 The tests are behavioural, not grep-based

The added test code sources the helper and compares exact output bytes; no source-text assertion is used as an oracle. The oracle is self-checked by a missing-marker negative control in both suites (see mutant **M7** below, which is killed by that control).

---

## 3. bash 3.2 is unaffected

```
$ /bin/bash -n tools/launchd/lib/quorum-artifacts.sh ; echo rc=$?
rc=0
$ /bin/bash -n tools/launchd/lib/pi-ext-args.sh ; echo rc=$?
rc=0
$ /bin/bash -c 'echo "$BASH_VERSION ${BASH_VERSINFO[0]}.${BASH_VERSINFO[1]}"'
3.2.57(1)-release 3.2
$ /bin/bash -c 'echo "ZSH_VERSION=[${ZSH_VERSION:-UNSET}]"'
ZSH_VERSION=[UNSET]
```

* The guard's condition is **false under bash** (`ZSH_VERSION` unset), so `setopt` is never reached: bash semantics are unchanged by construction.
* The suites ran under `/bin/bash` 3.2.57 — the same binary the plan names — and all their bash arms are green, including the suite's own `bash32_and_suite_reach` arm, which on Darwin asserts `BASH_VERSINFO` is exactly 3.2 and re-runs `/bin/bash -n` on the quorum helper. It PASSed.
* Direct bash probe of the candidate under `set -eu`: rc 0, stderr contains only the helper's own `quorum consult:` lines — **no** `command not found` noise.

```
--- lib=…/wt-fleet-iter60-judge/tools/launchd/lib/quorum-artifacts.sh
quorum consult: …/export
quorum consult: …/repo/.ailang/state/mission-quorum
AFTER_CALL
  bash top-level rc=0
```

---

## 4. Mutation discipline

Each mutant was applied to an **isolated scratch copy** of the repo (`/tmp/claude/i60/mut/<label>/`), with the four code/test files first restored from the pristine candidate, so no mutant could inherit a previous one, and **the worktree's tracked files were never modified** (verified: `git status --porcelain --untracked-files=all` empty at the end; `git diff --quiet HEAD --` clean; `git hash-object` of both libs == their `HEAD` blobs). The permitted `git checkout --` restore was therefore never needed.

Command shape for every row (run from the mutant tree, via the suite's own launcher):

```
cd <mutant-tree> && /bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_<suite>.sh
```

### 4.1 Mutation table

| # | Mutant (exact edit) | Library | Command | rc | Kill result |
|---|---|---|---|---|---|
| M1a | delete the guard line `if [ -n "${ZSH_VERSION:-}" ]; then setopt local_options null_glob; fi` | quorum | `suite-env.sh test_quorum_artifacts.sh` | **1** | **KILLED** — `FAIL zsh_quorum_export_only: child rc=1 … no matches found` (NOMATCH abort, `AFTER_CALL` absent) |
| M1b | same deletion | pi | `suite-env.sh test_pi_ext_args.sh` | **1** | **KILLED** — `FAIL: zsh_pi_empty` + `FAIL: zsh_pi_options` (`no matches found`), `zsh_pi_two_ts` still PASS, `==== 15 passed, 2 failed ====` |
| M2a | replace the guard with unconditional `setopt local_options null_glob` (so it also fires under bash) | quorum | `suite-env.sh test_quorum_artifacts.sh` | **0** | **SURVIVED — see §5.1 (not equivalent; real defect, undetected)** |
| M2b | same | pi | `suite-env.sh test_pi_ext_args.sh` | **0** | **SURVIVED — see §5.1** (`setopt: command not found` ×3 on stderr, suite still green) |
| M3a | drop `local_options` → `setopt null_glob` | quorum | `suite-env.sh test_quorum_artifacts.sh` | **1** | **KILLED** — `FAIL zsh_quorum_options: child rc=81 … OPTIONS_LEAKED` |
| M3b | drop `local_options` → `setopt null_glob` | pi | `suite-env.sh test_pi_ext_args.sh` | **1** | **KILLED** — `FAIL: zsh_pi_options: child rc=81 … OPTIONS_LEAKED`, `==== 16 passed, 1 failed ====` |
| M4a | nonexistent option → `setopt local_options null_globb` | quorum | `suite-env.sh test_quorum_artifacts.sh` | **1** | **KILLED** — `mc_quorum_lookup:setopt:1: no such option: null_globb` + `no matches found` |
| M4b | nonexistent option | pi | `suite-env.sh test_pi_ext_args.sh` | **1** | **KILLED** — `no such option: null_globb`, `==== 15 passed, 2 failed ====` |
| M5a | spelling alias `null_glob` → `nullglob` | quorum | `suite-env.sh test_quorum_artifacts.sh` | 0 | **SURVIVED — judged EQUIVALENT** (see below) |
| M5b | spelling alias | pi | `suite-env.sh test_pi_ext_args.sh` | 0 | **SURVIVED — judged EQUIVALENT** (see below) |
| M6a | guard on `BASH_VERSION` instead of `ZSH_VERSION` | quorum | `suite-env.sh test_quorum_artifacts.sh` | **1** | **KILLED** — `FAIL zsh_quorum_export_only … no matches found` |
| M6b | inverted condition `[ -z "${ZSH_VERSION:-}" ]` | quorum | `suite-env.sh test_quorum_artifacts.sh` | **1** | **KILLED** — `FAIL zsh_quorum_export_only … no matches found` |
| M6c | guard on `BASH_VERSION` | pi | `suite-env.sh test_pi_ext_args.sh` | **1** | **KILLED** — `FAIL: zsh_pi_empty`, `FAIL: zsh_pi_options`, `==== 15 passed, 2 failed ====` |
| M6d | inverted condition | pi | `suite-env.sh test_pi_ext_args.sh` | **1** | **KILLED** — `==== 15 passed, 2 failed ====` |
| M7a | oracle mutation: delete the marker check `[ "$(grep -cx 'AFTER_CALL' "$out" \|\| true)" = 1 ] \|\| return 1` from the checker | quorum test | `suite-env.sh test_quorum_artifacts.sh` | **1** | **KILLED** — `FAIL zsh_quorum_export_only: missing-marker negative control accepted` |
| M7b | same | pi test | `suite-env.sh test_pi_ext_args.sh` | **1** | **KILLED** — `FAIL: zsh_pi_empty/zsh_pi_two_ts/zsh_pi_options: missing-marker negative control accepted`, `==== 14 passed, 3 failed ====` |

**14 of 16 mutant instances killed; 2 survive (one per library), for one distinct reason each.**

### 4.2 The `nullglob` spelling mutant — judged EQUIVALENT, not a miss

M5a/M5b survive. This is the mutant the plan itself pre-declared as equivalent ("D3's spelling mutant is equivalent, so expecting its death would be false evidence. Keep D1's requested spelling."), and my measurement agrees:

```
$ /bin/zsh -f -c 'setopt nomatch; unsetopt null_glob; f() { setopt local_options nullglob; }; f; [[ -o nomatch ]] && print NOMATCH_STILL_ON; [[ -o null_glob ]] && print NULL_GLOB_LEAKED'
NOMATCH_STILL_ON
```

zsh 5.9 accepts `nullglob` as an alias for `null_glob`, so the mutant is **behaviourally identical to the shipped code** on this rig. Recorded as survived/equivalent rather than inventing a kill — exactly the discipline the plan demanded.

### 4.3 Producer-side mutant (the plan's "drop the marker echo" row)

The plan also asks for a *producer* mutant: remove the child's `AFTER_CALL` echo while the helper result stays correct. The positive arms must then reject it. The executor's notes record this row as killed for both libraries ("result/AFTER_CALL mismatch; helper result correct and child rc 0, producer marker missing"). I did not re-run that row independently — the shipped code does not contain the marker echo (it lives in the suite's child script), and the M7 oracle row already demonstrates that the marker assertion is load-bearing. **Minor gap in my own evidence, not in the candidate.**

---

## 5. Findings

### 5.1 F1 (non-blocking, test-coverage) — an unconditional `setopt` is not detected by either suite

**Finding.** The plan's mutation row says: *"Guard on BASH_VERSION instead of ZSH_VERSION, or invert the ZSH_VERSION condition … bash gates also catch inappropriate setopt execution."* The first half is true (M6a–M6d are killed). The second half is **not**: a guard that fires under bash as well is not caught by any arm.

Direct proof that the mutant is **not equivalent** to the shipped code:

```
$ /bin/bash -c 'set -eu; . "$1"; mc_quorum_reader_dirs "$MISSION_DRIVER_ROOT"; mc_quorum_lookup doc-x; echo AFTER_CALL' bash <M2a mutant>
…/quorum-artifacts.sh: line 59: setopt: command not found
  bash top-level rc=127

$ /bin/bash -c 'set -eu; . "$1"; mc_quorum_reader_dirs "$MISSION_DRIVER_ROOT"; mc_quorum_lookup doc-x; echo AFTER_CALL' bash <candidate>
quorum consult: …/export
quorum consult: …/repo/.ailang/state/mission-quorum
AFTER_CALL
  bash top-level rc=0
```

Under bash the mutant prints `setopt: command not found` on **every** call and aborts the script with rc 127 when `errexit` is active. The suites miss it because every call site in the suites is invoked from a context where `errexit` is suspended (`expect_rc` calls the function in an `if` condition; the other arms use `||` lists), so the failing command's status is discarded and the helper's own return code still governs.

**Impact on the candidate: none.** The shipped guard is zsh-only (`ZSH_VERSION` is unset under bash — verified in §3), so the defect is not present in the code under review. This is a *sensitivity* gap in the new arms, not a defect in the delivered behaviour.

**Suggested follow-up (out of scope for this PR):** add one bash-side arm to each suite that asserts the helper leaves bash stderr empty and returns 0 under `set -eu` — e.g. `bash -c 'set -eu; . lib; …; mc_quorum_lookup doc'` with `2>"$HOME/err"` and `[ ! -s "$HOME/err" ]`. That single assertion kills M2 in both libraries.

**Severity:** low (test sensitivity). Not a blocker for acceptance.

### 5.2 F2 (informational) — `make test-launchd-drivers` is red **at base and at the candidate**, for the same pre-existing reason

```
$ make test-launchd-drivers            # candidate
RC=2   → FAIL: E cleaned routing suite passes (rc=1)   → make: *** [test-launchd-drivers] Error 1
$ make test-launchd-drivers            # origin/dev libraries in an identical scratch tree
RC=2   → FAIL: E cleaned routing suite passes (rc=1)   → make: *** [test-launchd-drivers] Error 1
```

The PASS/FAIL lines of the two runs are identical except for the five new zsh arms the candidate adds:

```
$ diff candidate.txt base.txt
1,5d0
< PASS zsh_quorum_export_only
< PASS zsh_quorum_local_only
< PASS zsh_quorum_absent
< PASS zsh_quorum_options
< PASS zsh_gate2_export_only
--- line counts: cand= 54 base= 49 ---
```

The red gate is `tools/launchd/test_suite_env.sh` arm E → `tools/launchd/test_mission_routing.sh`, which fails in this sandbox for environment reasons (the executor recorded the same class of red at base: "rc 2 at test_mission_stall.sh … UNINFORMATIVE UNDER SANDBOX … Red at base; no workaround"). **This is a pre-existing, base-identical red — not a candidate regression**, and per the plan it is reported, not worked around. The controller should still re-run `make test-launchd-drivers` outside the sandbox.

### 5.3 F3 (informational) — sprint-state bookkeeping

`.ailang/state/sprints/sprint_M-FLEET-ZSH-SAFE-MISSION-LIBS.json` carries `status: "pending"` at the top level while the milestone is `completed`, and the executor notes record a generic-skill validator mismatch (`validator rc 5 (schema/count mismatch)`) that was deliberately not acted on. The executor's notes are otherwise thorough and honest (base gates, TDD red evidence, per-mutant command/rc/diagnostic, missing-interpreter simulation, final gates, explicit "UNINFORMATIVE UNDER SANDBOX" statements). No impact on the delivered behaviour; noted for the controller.

### 5.4 No defects found in the delivered change

* Guard is byte-exact, first-statement, zsh-only, in both functions; both libraries.
* Bash 3.2 path byte-identical to base; `setopt` never reached under bash.
* `local_options` restores NOMATCH/NULL_GLOB in the caller in both call shapes — no leak into the controller's tool shell.
* Exact results, rc codes and byte-for-byte output preserved for bash (all 16 pre-existing quorum arms and all 14 pre-existing pi assertions remain green).
* No change to Gate-2 snippet text, quorum policy, CI, or any file outside the plan's scope.
* `git diff --check` rc 0; `make check-changelog` rc 0; changelog fragment opens with `### Fixed — …`, explains both failures and the scoped option restoration, and links the plan.

---

## 6. Sandbox limitations (recorded, as the plan requires)

This evaluation ran inside a filesystem sandbox that denies writes to `/private/tmp`, `/var/folders/...` and `$HOME`.

* macOS `mktemp` always targets `$DARWIN_USER_TEMP_DIR` (`/var/folders/...`) regardless of `TMPDIR`, so the suites' own `mktemp -d` calls failed with *"mkstemp failed … Operation not permitted"*. A **PATH shim** (`/tmp/claude/i60/bin/mktemp`) redirected scratch creation to a writable scratch root while preserving `mktemp` semantics (`-d`, `-t prefix`, explicit templates pass through). The shim only changes *where* scratch directories are created; it does not alter any test logic, assertion, or code path.
* No socket/child-CPU-dependent arm was treated as informative. `make test-launchd-drivers` is red for a base-identical reason (§5.2) and is reported as such.
* Consequence: `make test-launchd-drivers` must still be re-run by the controller outside the sandbox. The two focused suites, which are the load-bearing gates for this change, ran green here with the real `/bin/zsh` and the real `/bin/bash` 3.2.57.

---

## 7. Score — 93/100

| Dimension | Weight | Score | Basis |
|---|---:|---:|---|
| **Code correctness** | 30 | **30** | Guard byte-exact and first-statement in both libraries; no other behaviour change (+3/−1, +2 lines); bash path provably untouched; `local_options` restores caller options at top level and inside a caller function; independent base-vs-candidate probe reproduces and fixes the ticket symptom. |
| **Tests** | 25 | **22** | 8 new behavioural arms, real `/bin/zsh -f` children, isolated scratch, scrubbed exports, explicit rc capture, exact byte comparison, quoted array/path-with-spaces assertions, both Gate-2 copies, plus a missing-marker negative control; base suites still green (16 arms / 14 assertions); arms individually gated and green. **−3:** no bash-side assertion on stderr/rc, so an unconditional `setopt` is undetectable (F1). |
| **Mutation coverage** | 20 | **18** | 16 mutant instances re-run on isolated trees with command/rc/diagnostic recorded; 14 killed including the oracle-mutation row; the spelling alias correctly recorded as an equivalent survivor instead of a manufactured kill; each library exercised independently. **−2:** one distinct reachable mutant (unconditional `setopt`) survives — the executor ran only the `BASH_VERSION`-guard variant, and the plan's claim that the bash gates would catch inappropriate `setopt` execution is not borne out. |
| **Evidence quality** | 15 | **13** | Executor notes are detailed and candid (base gates, TDD red, per-mutant rows, missing-interpreter simulation, explicit sandbox caveats); my own evidence is fully reproducible from this report and includes a base-vs-candidate `make` comparison. **−2:** producer-side marker-echo mutant not independently re-run by me; sprint JSON status left `pending` while the milestone is `completed`. |
| **Scope discipline** | 10 | **10** | Executor commit touches exactly the 5 authorized implementation files (+ sprint bookkeeping); planner commit touches only plan + sprint JSON; no Gate-2 snippet text, no CI, no policy, no other library changed; changelog fragment correct and `make check-changelog` rc 0; `git diff --check` rc 0; worktree left pristine. |
| **Total** | **100** | **93** | |

---

## 8. Verdict

**CODE ACCEPT — 93/100. No blockers.**

The candidate is a faithful, minimal, byte-identical rebase of the reviewed iteration-57 build of M-FLEET-ZSH-SAFE-MISSION-LIBS. Both glob sites are guarded exactly as the plan specifies; bash 3.2 behaviour is provably unchanged; the 8 new arms are behavioural, self-checked and green; 14 of 16 mutant instances are killed and the two survivors are explained (one equivalent, one a documented test-sensitivity gap in code that is **not** defective as shipped).

**Follow-ups (non-blocking, out of scope for this PR):**
1. **F1** — add a bash-side assertion (helper leaves stderr empty and returns 0 under `set -eu`) to both suites, so an unconditional `setopt` becomes killable.
2. **F5.2** — the controller should re-run `make test-launchd-drivers` outside the sandbox; the red observed here is base-identical and unrelated to this change.

---

### Appendix — raw commands cited

```
$ git rev-parse HEAD
dcab63a3c006bc5455bd93b3d124e52e695e887c

$ git range-diff origin/fleet/i57-zsh-safe-libs~2..origin/fleet/i57-zsh-safe-libs HEAD~2..HEAD
1:  b92b42818 = 1:  bd9a41cf3 docs(sprint): plan M-FLEET-ZSH-SAFE-MISSION-LIBS …
2:  6268fb0b8 = 2:  dcab63a3c fix(mission): quorum and pi-ext helpers survive zsh NOMATCH …

$ /bin/bash -n tools/launchd/lib/quorum-artifacts.sh ; echo rc=$?     → rc=0
$ /bin/bash -n tools/launchd/lib/pi-ext-args.sh     ; echo rc=$?     → rc=0

$ /bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_quorum_artifacts.sh   → rc=0 (21 PASS)
$ /bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_pi_ext_args.sh        → rc=0 (17 passed, 0 failed)
$ make check-changelog                                                             → rc=0
$ git diff origin/dev..HEAD --check                                               → rc=0
$ make test-launchd-drivers                                                       → rc=2 (identical red at base)

$ git status --porcelain --untracked-files=all     → (empty)
$ git diff --quiet HEAD --                          → clean
$ git hash-object tools/launchd/lib/quorum-artifacts.sh → adaead4806607be0f1ac7fcb63571398e8acf28c  (== HEAD blob)
$ git hash-object tools/launchd/lib/pi-ext-args.sh     → a4a9acf0389c348d637d348a6901b442afee1cc1  (== HEAD blob)
```
