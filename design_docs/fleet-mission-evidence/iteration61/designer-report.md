DESIGNER VERDICT: CONFIRMED

# Designer report — fleet iteration 61, ticket `quorum-reader:zsh-nomatch`

- Base: `origin/dev` = `c8e348d08` (HEAD of this read-only checkout, verified by `git rev-parse HEAD origin/dev`).
- Candidate: `origin/fleet/i60-zsh-libs-land` = `dcab63a3c` (PR #1942); `git merge-base origin/dev origin/fleet/i60-zsh-libs-land` → `bf22f4943…` (the iteration-60 base).
- Controlling plan: `design_docs/planned/sprint-plan-zsh-safe-mission-libs.md` on the branch (read via `git show`); iteration-60 designer report read (it was CONFIRMED).
- Verdict: the design contract still holds at `c8e348d08`. No new design is needed. Neither commit that landed on dev since bf22f4943 touches any branch file, any caller of the two helpers, or the test wiring the plan depends on.

## 1. The fix still matches the plan's mandated shape

`git diff origin/dev...origin/fleet/i60-zsh-libs-land -- tools/launchd/lib/quorum-artifacts.sh tools/launchd/lib/pi-ext-args.sh`
gives exactly the following:
- `pi-ext-args.sh`: one WHY header line, plus `if [ -n "${ZSH_VERSION:-}" ]; then setopt local_options null_glob; fi` as the first statement of `_mc_pi_ext_args()`.
- `quorum-artifacts.sh`: the header changes from "Pure bash 3.2" to "Bash 3.2 and zsh", one WHY line is added, and the same guard is the first statement of `mc_quorum_lookup()`.
Nothing else changed: no existence guards, return codes, lookup or verdict policy. This matches plan section "One milestone" (the exact statement, first in BOTH functions, header WHY lines) and is the same diff the iteration-60 report recorded.

## 2. Commits on dev since the iteration-60 base

`git log --oneline bf22f4943..origin/dev`
```
c8e348d08 build(deps): bump fast-uri from 3.1.8 to 3.1.9 in /docs (#1943)
d8e47684c Maintenance instrument: retained stages, native evidence and closed admission (#1940)
```
- `c8e348d08` changes only `docs/package-lock.json` (1 file, +3/-3).
- `d8e47684c` changes 82 files under `.ailang/state/{evaluations,sprints}` (MAINTENANCE-EVAL-ADAPTER), `.github/workflows/{build,ci,maintenance-instrument}.yml`, `Makefile`, `make/{ci,eval}.mk`, `cmd/ailang/*`, `design_docs/{planned,validation}/maintenance-*`, `docs/docs/guides/maintenance-benchmark.md`, `internal/eval_harness/maintenance_*`, `internal/executor/**`, and `tools/pi-extensions/maintenance-*`. None of these are under `tools/launchd/`.

### 2a. File overlap with the branch: none
`comm -12 <(git diff --name-only origin/dev...origin/fleet/i60-zsh-libs-land | sort) <(git diff --name-only bf22f4943..origin/dev | sort)`
→ (empty), `overlap_count=0`.
The branch's 7 paths are the sprint JSON, the changelog fragment, the plan, the 2 libs and the 2 test files. The dev side shares none of them, including the sprint JSON (dev added `sprint_M-MAINTENANCE-EVAL-ADAPTER.json`, which is a different file).

### 2b. Callers of `mc_quorum_lookup` / `_mc_pi_ext_args` at origin/dev
`git grep -n -E 'mc_quorum_lookup|_mc_pi_ext_args' origin/dev -- tools scripts .claude/skills .agents/skills` finds these call sites outside the libs and their tests:
```
.agents/skills/mission-control/resources/gate-2-pick.md:222:  mc_quorum_lookup "$1"
.claude/skills/mission-control/resources/gate-2-pick.md:222:  mc_quorum_lookup "$1"
.agents/skills/mission-control/resources/gate-3-route.md:355:  ... done < <(_mc_pi_ext_args "$WT")
.claude/skills/mission-control/resources/gate-3-route.md:355:  ... done < <(_mc_pi_ext_args "$WT")
scripts/mission_pi_run.sh:288:  ... done < <(_mc_pi_ext_args "$WORKDIR")
tools/launchd/mission-control.sh:2472:  ... done < <(_mc_pi_ext_args "$REPO")
```
The remaining hits are definitions and doc comments in the two libs, and test call sites in `tools/launchd/test_pi_ext_args.sh` and `tools/launchd/test_quorum_artifacts.sh`.

Is any caller touched by the new dev commits?
`git diff --stat bf22f4943..origin/dev -- tools/launchd scripts/mission_pi_run.sh .claude/skills .agents/skills .pi/extensions`
→ (empty), rc=0. Every caller file, both skill copies of the Gate-2/Gate-3 snippets (which `zsh_gate2_export_only` extracts), and the real `.pi/extensions` directory are byte-unchanged since bf22f4943.

Does the dev diff mention any helper, lib or test name?
`git diff bf22f4943..origin/dev | grep -n -E 'mc_quorum|_mc_pi_ext_args|quorum-artifacts|pi-ext-args|test-launchd-drivers|test_quorum_artifacts|test_pi_ext_args|suite-env'`
→ only lines 8841/8842: the `ci:` aggregate target in `make/ci.mk:11`, which contains `test-launchd-drivers` as a dependency. The only change on that line is a new `verify-maintenance-assets` dependency; `test-launchd-drivers` stays in the list.

### 2c. Test wiring the plan relies on is intact
- `make/test.mk:70` (`test-launchd-drivers:`) is not in the dev diff. `make/test.mk` is not in `git diff --name-only bf22f4943..origin/dev`.
- `.github/workflows/ci.yml:786` still runs `make test-launchd-drivers`. The `ci.yml`/`build.yml` diff adds only "Prefetch Python for offline eval-harness tests" (`uv python install 3.12`) steps. The plan's claim that existing make wiring reaches the zsh arms on macos-latest without CI edits therefore still holds.
- The `Makefile` diff touches only `pi-assets` and `verify-pi-assets`/`verify-maintenance-assets`. These copy and compare `tools/pi-extensions/maintenance-*` into `internal/executor/pi/profile_assets/`, and the `.pi/extensions` glob that `_mc_pi_ext_args` reads is not involved.

### 2d. Branch still merges cleanly onto the current base
`git merge-tree --write-tree origin/dev origin/fleet/i60-zsh-libs-land` → `24ebc09ed3b0f92b2ac3762f045bbc0997445835`, rc=0 (no conflicts).

## 3. Decision

**CONFIRMED.** The contract still holds at `c8e348d08`:
- The exact zsh-only, function-local `null_glob` guard is still the first statement of both globbing functions.
- bash 3.2 behavior is preserved.
- No routing, quota or quorum policy surface is touched.
- The callers and the suite/CI wiring that the plan's behavioral arms depend on are all unchanged.

The two new dev commits don't intersect the candidate: one is a docs lockfile bump, the other a maintenance-eval adapter that changes no file under `tools/launchd/`. No design revision is required. A rebase or merge onto `c8e348d08` should carry the twice-judged patch (PASS 94 sonnet / ACCEPT 93 MiniMax) unchanged.

## Not done here (outside designer scope)
- I did not rerun the plan's validation gates (both focused suites, `make test-launchd-drivers`, `make check-changelog`, `git diff --check`). As the plan says, in-sandbox results are UNINFORMATIVE UNDER SANDBOX, and the controller's run outside the sandbox on the merged result is still the landing gate.
- I edited no repository files and ran no `ailang messages` commands.
