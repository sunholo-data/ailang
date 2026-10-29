EXECUTOR PASS

# Executor test/inspection report — fleet mission iteration 60

Candidate commit: `dcab63a3c006bc5455bd93b3d124e52e695e887c` (PR #1942, "zsh-safe mission libs")
Base: `origin/dev` = `bf22f4943aa94f269d5aec4a54300dfcc2c9f7a7`
Host: macOS Darwin 25 (arm64), `/bin/bash` = GNU bash 3.2.57, `/bin/zsh` = zsh 5.9
Mode: TEST/INSPECTION ONLY — no tracked file modified, no git write operation performed.

## Step results

| # | Command | rc | Observed output tail |
|---|---------|----|----------------------|
| 1 | read `tools/launchd/test_quorum_artifacts.sh` (434 L) and `tools/launchd/test_pi_ext_args.sh` (161 L) in full, plus both libs | 0 | Read-only. zsh gating as documented: quorum suite's `run_arm()` treats `zsh_*` arms as `if [ ! -x /bin/zsh ]` → `fail 'no /bin/zsh on Darwin'` on Darwin, `SKIP (no /bin/zsh)` elsewhere; each arm spawns `/bin/zsh -f -c 'setopt nomatch; unsetopt null_glob; …'` and asserts exact helper bytes + exactly one `AFTER_CALL` marker, with a missing-marker negative control. pi-ext suite's `zsh_pi_arm()` uses the same gate (`bad` on Darwin / `SKIP` elsewhere) for `zsh_pi_empty zsh_pi_two_ts zsh_pi_options`; it has **no** per-arm selector, its zsh arms run inline. |
| 2 | `bash tools/launchd/test_quorum_artifacts.sh` | 0 | 21/21 arms PASS in ~1 s. Tail:<br>`PASS bash32_and_suite_reach`<br>`PASS attended_absent_and_errors`<br>`PASS cross_mode_errors_dominate`<br>`agents skills in sync (mission-control mission-brief mission-loop-change sprint-planner sprint-executor sprint-evaluator)`<br>`PASS generated_docs_and_mirror`<br>`PASS fixture_home_and_observations`<br>(all five `zsh_*` arms PASS at head of output; no `SKIP` lines — `/bin/zsh` present. `--writer` arms correctly not run: no `--writer` supplied, per the file's own contract.) |
| 3 | `bash tools/launchd/test_pi_ext_args.sh` | 0 | `  PASS: skill launch sweep safety`<br>`  PASS: skill launch count instrument`<br>`  PASS: zsh_pi_empty`<br>`  PASS: zsh_pi_two_ts`<br>`  PASS: zsh_pi_options`<br>`==== 17 passed, 0 failed ====` |
| 4 | `/bin/bash -n` on `lib/quorum-artifacts.sh`, `lib/pi-ext-args.sh`, `test_quorum_artifacts.sh`, `test_pi_ext_args.sh` | 0,0,0,0 | `tools/launchd/lib/quorum-artifacts.sh rc=0 (clean)`<br>`tools/launchd/lib/pi-ext-args.sh rc=0 (clean)`<br>`tools/launchd/test_quorum_artifacts.sh rc=0 (clean)`<br>`tools/launchd/test_pi_ext_args.sh rc=0 (clean)`<br>parser: GNU bash 3.2.57(1)-release (arm64-apple-darwin25) |
| 5 | `/bin/zsh` exists → zsh arms run **as each file documents them**: `bash tools/launchd/test_quorum_artifacts.sh --arm <name>` for each of the 5 zsh arms; pi-ext suite's inline zsh arms already covered by its documented invocation (step 3) | 0 (×5) | `PASS zsh_quorum_export_only` rc=0<br>`PASS zsh_quorum_local_only` rc=0<br>`PASS zsh_quorum_absent` rc=0<br>`PASS zsh_quorum_options` rc=0<br>`PASS zsh_gate2_export_only` rc=0<br>zsh sanity: `zsh 5.9 (arm64-apple-darwin25.0)` |
| 6 | `git status --porcelain` (before writing this report) | 0 | empty — printed nothing. `git status --porcelain --untracked-files=all` also empty. |
| 7 | `git diff --name-only origin/dev..HEAD` | 0 | exactly 7 paths (`wc -l` = 7):<br>`.ailang/state/sprints/sprint_M-FLEET-ZSH-SAFE-MISSION-LIBS.json`<br>`changelogs/unreleased/2026-10-22-zsh-safe-mission-libs.md`<br>`design_docs/planned/sprint-plan-zsh-safe-mission-libs.md`<br>`tools/launchd/lib/pi-ext-args.sh`<br>`tools/launchd/lib/quorum-artifacts.sh`<br>`tools/launchd/test_pi_ext_args.sh`<br>`tools/launchd/test_quorum_artifacts.sh`<br>— 2 libs + 2 suites + 1 changelog + 1 plan doc + 1 sprint JSON. No extra paths. |
| 8 | `git diff origin/dev..HEAD -- tools/launchd/lib/` + `grep -n` guard placement | 0 | Both globbing functions open with the zsh-only guard as their **first** statement:<br>`pi-ext-args.sh:20 _mc_pi_ext_args() {` → `:21   if [ -n "${ZSH_VERSION:-}" ]; then setopt local_options null_glob; fi` → `:22 local dir="$1" f` (glob at `:25 for f in "$dir"/.pi/extensions/*.ts`)<br>`quorum-artifacts.sh:58 mc_quorum_lookup() {` → `:59   if [ -n "${ZSH_VERSION:-}" ]; then setopt local_options null_glob; fi` → `:60 local slug="$1" dir file bad=0` (glob at `:70 for file in "$dir/$slug-"*.json`)<br>Diff is otherwise comment-only (`# Skill snippets source this into the controller's zsh tool shell, where NOMATCH aborts unmatched loops.`; quorum header `Pure bash 3.2 functions` → `Bash 3.2 and zsh functions`). `local_options` scoping means caller options survive. |

## Extra verification (non-destructive negative control, `/tmp` copies only)

To prove the guard is load-bearing rather than decorative I stripped only the guard line from `/tmp` copies (no tracked file touched) and ran both functions under `zsh -f` with `setopt nomatch; unsetopt null_glob`:

- A `pi-ext-args` **without** guard: `_mc_pi_ext_args:4: no matches found: …/.pi/extensions/*.ts`, zsh rc=1 — the pre-fix abort.
- B `pi-ext-args` **with** guard: `--no-extensions` / `RC=0` / `OPTIONS_PRESERVED`.
- C `mc_quorum_lookup` **without** guard: `mc_quorum_lookup:11: no matches found: …/doc-*.json`, zsh rc=1 — abort instead of documented "absence" rc=1.
- D `mc_quorum_lookup` **with** guard: `RC=1` (absence, correct) / `OPTIONS_PRESERVED`.
- E under bash 3.2 both libs behave identically to `origin/dev` (`pi rc=0` printing `--no-extensions`; `lookup rc=1`), i.e. the guard is inert under bash and no semantics shifted.

## Verdict

EXECUTOR PASS. Both suites pass at `dcab63a3c` (21/21 quorum arms; 17/17 pi-ext checks incl. all three zsh arms), all four files parse under real bash 3.2, the zsh arms are non-vacuous (each carries an `AFTER_CALL` oracle negative control that the run confirmed rejects a missing marker), the diff is exactly the 7 expected paths, and the fix shape is precisely a `setopt local_options null_glob` guard opening each globbing function while remaining a no-op in bash.

## Environment note (no effect on verdict)

The sandbox denies Darwin's `_CS_DARWIN_USER_TEMP_DIR` (`/var/folders/…/T`), so `mktemp -d` — which on macOS ignores `TMPDIR` — fails with `mkstemp failed … Operation not permitted` before either suite starts. Both suites call bare `mktemp -d` for their fixtures, so I ran each with a `mktemp` shim first on `PATH` that delegates to `/usr/bin/mktemp -p /tmp/claude/exec60` (fixture location only; no assertion in either suite depends on the temp path). Without the shim the run aborts at line 11 of the quorum suite and at `T=$(mktemp -d)` of the pi-ext suite — an environment limitation, not a defect in the candidate.
