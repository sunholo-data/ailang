DESIGNER CONFIRMED

# Designer report — fleet iteration 60, refreshed landing of M-FLEET-ZSH-SAFE-MISSION-LIBS

- Candidate: `dcab63a3c` (`dcab63a3c006bc5455bd93b3d124e52e695e887c`), branch `fleet/i60-zsh-libs-land`, PR #1942 — verified `origin/fleet/i60-zsh-libs-land = HEAD`.
- Base: `bf22f4943` (`bf22f4943aa94f269d5aec4a54300dfcc2c9f7a7`) = `origin/dev` — verified `HEAD~2 = origin/dev = git merge-base origin/dev HEAD`.
- Verdict: the existing design contract (the banked plan `design_docs/planned/sprint-plan-zsh-safe-mission-libs.md`, which declares itself its own design document) still holds for this landing. No new design decision is required; the refresh is a pure rebase with byte-identical content.

## Verification rows

### 1. Scope unchanged — exactly the 7 authorized paths

Command:
```sh
git diff --name-only origin/dev..HEAD
```
Observed:
```
.ailang/state/sprints/sprint_M-FLEET-ZSH-SAFE-MISSION-LIBS.json
changelogs/unreleased/2026-10-22-zsh-safe-mission-libs.md
design_docs/planned/sprint-plan-zsh-safe-mission-libs.md
tools/launchd/lib/pi-ext-args.sh
tools/launchd/lib/quorum-artifacts.sh
tools/launchd/test_pi_ext_args.sh
tools/launchd/test_quorum_artifacts.sh
```
(`git diff --name-only origin/dev..HEAD | wc -l` → `7`) — exactly the declared list, no more, no less.

### 2. The rebase onto bf22f4943 introduced no content change — no new design decision needed

Command:
```sh
git range-diff origin/fleet/i57-zsh-safe-libs~2..origin/fleet/i57-zsh-safe-libs HEAD~2..HEAD
```
Observed:
```
1:  b92b42818 = 1:  bd9a41cf3 docs(sprint): plan M-FLEET-ZSH-SAFE-MISSION-LIBS — mission libs survive zsh NOMATCH when sourced into the controller shell (fleet iteration 57)
2:  6268fb0b8 = 2:  dcab63a3c fix(mission): quorum and pi-ext helpers survive zsh NOMATCH when sourced into the controller shell (fleet iteration 57)
```
Both patches compare `=` — content-identical across the rebase.

Command (blob identity for each of the 7 paths, `origin/fleet/i57-zsh-safe-libs` vs `HEAD`):
```sh
for p in <the 7 paths>; do a=$(git rev-parse origin/fleet/i57-zsh-safe-libs:"$p"); b=$(git rev-parse HEAD:"$p"); [ "$a" = "$b" ] && echo "SAME $p" || echo "DIFF $p"; done
```
Observed:
```
SAME  tools/launchd/lib/quorum-artifacts.sh  adaead4806607be0f1ac7fcb63571398e8acf28c
SAME  tools/launchd/lib/pi-ext-args.sh  a4a9acf0389c348d637d348a6901b442afee1cc1
SAME  tools/launchd/test_quorum_artifacts.sh  7b11a4e0f94ba1317b423207e3f748124f2838f9
SAME  tools/launchd/test_pi_ext_args.sh  f51408cb012bc25061ca3d00f599f7691a332afc
SAME  changelogs/unreleased/2026-10-22-zsh-safe-mission-libs.md  7a5f85b49eee66cbff63f628750ee18114f6ebed
SAME  design_docs/planned/sprint-plan-zsh-safe-mission-libs.md  8455c67d9d1ee69f2d8b320614a9dcecabc5a921
SAME  .ailang/state/sprints/sprint_M-FLEET-ZSH-SAFE-MISSION-LIBS.json  acf7e58c6bfde2c35d0186cbf1ffd4e1d3591e0a
```
All 7 blobs identical between the iteration-57 branch head and the refreshed HEAD. Since both patches compare `=` and every changed blob is unchanged, the rebase onto bf22f4943 introduced no content change — the design decisions banked in the plan remain the controlling ones, and no new design decision is needed.

### 3. Inside the fleet's write authority; no policy-class change

Charter mapping (design_docs/fleet-mission.md, Authority, allowlist enforced by `tools/launchd/githooks/pre-push`): `tools/launchd/**` covers the two lib files and two test files; `changelogs/**` covers the changelog fragment; `design_docs/**` covers the plan; `.ailang/state/sprints/**` covers the sprint JSON. All 7 paths are inside "May change". None touch the refused core (`internal/{parser,lexer,ast,types,…}/**`, `std/**`, `examples/**`, `benchmarks/**`), and none touch `.pi/extensions/**` (the D-FLEET-3 parks-for-Mark class — the pi helper only *reads* that directory in its glob; nothing under it is modified).

Command:
```sh
git diff origin/dev..HEAD -- tools/launchd/lib/quorum-artifacts.sh tools/launchd/lib/pi-ext-args.sh
```
Observed — the entire lib diff is comments plus one guard line per function:
```
diff --git a/tools/launchd/lib/pi-ext-args.sh b/tools/launchd/lib/pi-ext-args.sh
@@ -1,5 +1,6 @@
 # shellcheck shell=bash
 # pi-ext-args.sh — sourced by the mission driver and scripts/mission_pi_run.sh.
+# Skill snippets source this into the controller's zsh tool shell, where NOMATCH aborts unmatched loops.
@@ -17,6 +18,7 @@ _mc_pi_ext_args() {
+  if [ -n "${ZSH_VERSION:-}" ]; then setopt local_options null_glob; fi
   local dir="$1" f
diff --git a/tools/launchd/lib/quorum-artifacts.sh b/tools/launchd/lib/quorum-artifacts.sh
@@ -1,6 +1,7 @@
-# Pure bash 3.2 functions. Nothing runs at source time. rc 1 is absence;
+# Bash 3.2 and zsh functions. Nothing runs at source time. rc 1 is absence;
 # rc 2 is a configuration/read error, never permission to skip quorum.
+# Skill snippets source this into the controller's zsh tool shell, where NOMATCH aborts unmatched loops.
@@ -55,6 +56,7 @@ mc_quorum_lookup() {
+  if [ -n "${ZSH_VERSION:-}" ]; then setopt local_options null_glob; fi
   local slug="$1" dir file bad=0
```
What the two lib files change, stated: header-comment lines (one WHY line each; the quorum header now says "Bash 3.2 and zsh" instead of "Pure bash 3.2") and a single zsh-guarded option line as the first statement of each globbing function. **No routing, lane order, quota, ration threshold or billing guard is altered** — the diff touches globbing mechanics only; model/role routing, lane scheduling and all quorum verdict/freshness policy lines are untouched.

### 4. Record preservation — original head intact

Command:
```sh
git rev-parse origin/fleet/i60-before-refresh
```
Observed:
```
6268fb0b86f59a6517b6b0a697e1f5e61dcb17f9
```
Equals the expected original head, and equals `origin/fleet/i57-zsh-safe-libs` (the iteration-57 branch head PR #1932 carried) — the pre-refresh head is preserved on a dedicated ref.

### 5. The design contract is sound — and the guard has the required shape

Codebase claims (both lib functions read at HEAD). Command:
```sh
grep -n 'setopt local_options null_glob\|^mc_quorum_lookup()\|^_mc_pi_ext_args()' tools/launchd/lib/quorum-artifacts.sh tools/launchd/lib/pi-ext-args.sh
```
Observed:
```
tools/launchd/lib/quorum-artifacts.sh:58:mc_quorum_lookup() {
tools/launchd/lib/quorum-artifacts.sh:59:  if [ -n "${ZSH_VERSION:-}" ]; then setopt local_options null_glob; fi
tools/launchd/lib/pi-ext-args.sh:20:_mc_pi_ext_args() {
tools/launchd/lib/pi-ext-args.sh:21:  if [ -n "${ZSH_VERSION:-}" ]; then setopt local_options null_glob; fi
```
The exact statement the plan mandates is the FIRST statement of both functions. Command (no other glob site exists):
```sh
grep -nE 'for [a-zA-Z_]+ in .*\*' tools/launchd/lib/*.sh
```
Observed:
```
tools/launchd/lib/pi-ext-args.sh:25:  for f in "$dir"/.pi/extensions/*.ts; do
tools/launchd/lib/quorum-artifacts.sh:70:    for file in "$dir/$slug-"*.json; do
```
Exactly two glob sites in `tools/launchd/lib/*.sh`, both inside the guarded functions — matching the plan's read-only inspection claim.

Guard shape, confirmed by reading and by probes below:
- **zsh-only**: guarded on `[ -n "${ZSH_VERSION:-}" ]`; false in bash (where `setopt` does not exist), so the guard never executes there.
- **local**: `setopt local_options` scopes the option change to the function.
- **restores caller options**: LOCAL_OPTIONS restores the caller's options on return — verified empirically (`[[ -o nomatch && ! -o null_glob ]]` holds in the same process after the call).

Empirical probes (zsh 5.9, read-only: sandbox blocks scratch creation, so the pre-fix libs were piped from `git show origin/dev:…` and the post-fix libs sourced from the worktree; real existing directories used):

**P1 — pre-fix quorum aborts the sourcing shell** (real `origin/dev` lib, existing readable dir with no `doc-x-*.json`):
```sh
{ git show origin/dev:tools/launchd/lib/quorum-artifacts.sh; printf 'MC_QUORUM_DIRS=("<pwd>/tools/launchd/lib")\nmc_quorum_lookup doc-x\necho "rc=$? AFTER_CALL"\n'; } | /bin/zsh -f -c 'setopt nomatch; unsetopt null_glob; source /dev/stdin'
```
Observed:
```
quorum consult: /Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter60-designer/tools/launchd/lib
mc_quorum_lookup:11: no matches found: /Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter60-designer/tools/launchd/lib/doc-x-*.json
exit=126
```
`AFTER_CALL` never printed — under zsh with NOMATCH set, the unguarded glob aborts the sourcing shell. Contract premise confirmed.

**P2 — post-fix quorum survives the same case** (HEAD lib, same dir, same zsh options):
```sh
/bin/zsh -f -c 'setopt nomatch; unsetopt null_glob; . tools/launchd/lib/quorum-artifacts.sh; MC_QUORUM_DIRS=("<pwd>/tools/launchd/lib"); mc_quorum_lookup doc-x; echo "rc=$? AFTER_CALL n=${#MC_QUORUM_MATCHES[@]}"; [[ -o nomatch && ! -o null_glob ]] && echo OPTIONS_RESTORED'
```
Observed:
```
quorum consult: <…>/tools/launchd/lib
rc=1 AFTER_CALL n=0
OPTIONS_RESTORED
exit=0
```
Absence is returned as rc 1 (the contract's "rc 1 is absence"), no abort, caller options intact.

**P2b — post-fix quorum matched path unchanged** (real artifacts in `~/.ailang/state/mission-quorum`, slug `w-software-engineering-domain`):
Observed: `rc=0 AFTER_CALL n=10` — matched lookup returns exactly the real artifacts.

**P3 — pre-fix pi aborts after printing `--no-extensions`** (real `origin/dev` lib, real `~/.pi/extensions` which exists and has no `.ts`):
```sh
{ git show origin/dev:tools/launchd/lib/pi-ext-args.sh; printf '_mc_pi_ext_args "$HOME"\necho "rc=$? AFTER_CALL"\n'; } | /bin/zsh -f -c 'setopt nomatch; unsetopt null_glob; source /dev/stdin'
```
Observed:
```
--no-extensions
_mc_pi_ext_args:4: no matches found: /Users/voightkampff/.pi/extensions/*.ts
exit=126
```
No `AFTER_CALL` — exactly the plan's measured failure shape ("print `--no-extensions` then abort").

**P4 — post-fix pi survives the same call** (HEAD lib, `$HOME`):
Observed:
```
--no-extensions
rc=0 AFTER_CALL
OPTIONS_RESTORED
exit=0
```

**P5 — post-fix pi matched path unchanged** (HEAD lib, worktree root whose `.pi/extensions` has `.ts` files):
Observed: `--no-extensions` followed by ordered `-e <path>` pairs for each `.ts` file, `rc=0 AFTER_CALL`.

**P6 — bash behavior byte-preserved** (GNU bash 3.2.57(1)-release; pre-fix lib via `git show origin/dev:… | /bin/bash` vs post-fix HEAD lib under `/bin/bash`, same input `$HOME`):
Observed: identical stdout (`--no-extensions`) and `rc=0` on both sides; no `setopt: command not found` — the ZSH_VERSION guard is false in bash, so `setopt` never executes and bash 3.2 semantics are untouched.

Supporting landing facts (the other three implementation paths):
- Test files implement the plan's behavioral arms, not source-text grep oracles. Command: `grep -n 'zsh_quorum_export_only\|zsh_quorum_local_only\|zsh_quorum_absent\|zsh_quorum_options\|zsh_gate2_export_only' tools/launchd/test_quorum_artifacts.sh` — observed: all five arms defined (lines 405–413) and registered in the default `run_arm` loop (line 431). Command: `grep -n 'zsh_pi_empty\|zsh_pi_two_ts\|zsh_pi_options' tools/launchd/test_pi_ext_args.sh` — observed: all three arms, run via `for arm in zsh_pi_empty zsh_pi_two_ts zsh_pi_options; do zsh_pi_arm "$arm"; done` (line 158). Command: `grep -c 'local_options null_glob' tools/launchd/test_*.sh` — observed: `0` in both test files (assertions are behavioral; they do not grep the guard string).
- Command: `head -8 changelogs/unreleased/2026-10-22-zsh-safe-mission-libs.md` — observed: opens with `### Fixed — Mission helpers handle empty glob matches in zsh (2026-10-22)`, describes both aborts and the scoped option restoration, per the plan's changelog requirement.

## In-sandbox limitation (recorded, as the plan itself requires)

The plan states in-sandbox suite results are **UNINFORMATIVE UNDER SANDBOX** and the controller reruns everything outside the sandbox. Observed here: `/bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_pi_ext_args.sh` → `==== 11 passed, 6 failed ====`, where every failure is scratch-creation refused by this sandbox (`mkdir: … Operation not permitted` → e.g. `FAIL: zsh_pi_two_ts: child rc=1, result/AFTER_CALL mismatch`); `/bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_quorum_artifacts.sh` cannot run at all in-sandbox (`mktemp: mkdtemp failed on …: Operation not permitted`). These are sandbox FS restrictions, not code findings; the P1–P6 probes above avoided scratch entirely by using the real libs against real existing directories, so the design-contract confirmation does not depend on in-sandbox suite results. The controller's outside-sandbox gate (both focused suites + `make test-launchd-drivers`) remains required before landing, per the plan.

## Scope concerns

None. The candidate diff is confined to the 7 authorized paths; the lib change is the exact two-line guard the plan mandates plus comments; behavior under bash 3.2 and the matched-glob happy paths are preserved; caller options are restored; no policy-class surface (routing, lane order, quota, ration, billing) is touched; the original head and the iteration-57 branch are preserved on their refs; and the refreshed patch is content-identical to the banked iteration-57 patch, so the approved design contract carries over unchanged.