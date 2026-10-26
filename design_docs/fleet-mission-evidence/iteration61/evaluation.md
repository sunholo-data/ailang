CODE ACCEPT — 94/100

# Evaluation i61: quorum-reader:zsh-nomatch (candidate b2c9ff5b0)

## Rubric
| Area | Score | Basis |
|---|---|---|
| Correctness of fix | 38/40 | One-line guard first in each function; `local_options` restores NOMATCH/NULL_GLOB on return (verified in zsh) |
| Tests / mutation strength | 28/30 | 5 zsh quorum arms + 3 pi arms, oracle negative control (AFTER_CALL marker), options-leak check; all 4 mutants killed |
| Scope / regression safety | 14/15 | 7 files only; bash path gated by ZSH_VERSION so unchanged; callers (mission-control.sh, mission_pi_run.sh, gate-2/gate-3 skill snippets in .claude and .agents) all go through the fixed functions |
| Docs / changelog / plan | 14/15 | Changelog fragment, sprint plan, sprint JSON present |

## Commands and results
- `git range-diff bf22f4943..dcab63a3c origin/dev..HEAD`: both commits `=` (identical patches to PR #1942 commits).
- `git diff --name-only origin/dev...HEAD`: 7 files (sprint JSON, changelog, plan, 2 libs, 2 tests). `git diff --name-only bf22f4943 origin/dev -- <those 7>`: 0 files, so no dev drift on them.
- Full diff read. Guard is `if [ -n "${ZSH_VERSION:-}" ]; then setopt local_options null_glob; fi`; in bash it is a no-op branch (ZSH_VERSION unset), so the bash 3.2 path is behaviourally identical (the existing 15 bash arms still pass).
- grep callers: mission-control.sh:2472 and scripts/mission_pi_run.sh:288 (bash, source the lib), gate-3-route.md:355 and gate-2-pick.md:222 (.claude and .agents mirrors, zsh tool shell). All invoke the functions, so the in-function guard covers them. No sourcing path bypasses it.
- `/bin/bash tools/launchd/lib/suite-env.sh tools/launchd/test_quorum_artifacts.sh`: rc=0, all arms PASS including the 5 zsh_* arms.
- Same for `test_pi_ext_args.sh`: rc=0, "17 passed, 0 failed". No sandbox denials in either output.
- Direct `/bin/zsh -f` with nomatch on: `mc_quorum_lookup doc-x` against an empty artifact dir returned rc=1 count=0 (no "no matches found"); `_mc_pi_ext_args` on an empty .pi/extensions printed `--no-extensions`, rc=0. After both, `[[ -o nomatch && ! -o null_glob ]]` was true (OPTS_OK).

## Mutation table (backups in /tmp/fleet_i61/mut/, restored with cp)
| Mutant | Result | Killing arm |
|---|---|---|
| M1 remove setopt from quorum-artifacts.sh | KILLED | zsh_quorum_export_only |
| M2 remove setopt from pi-ext-args.sh | KILLED | zsh_pi_empty, zsh_pi_options ("no matches found") |
| M3 drop `local_options` in quorum lib (leak) | KILLED | zsh_quorum_options (rc=81 OPTIONS_LEAKED) |
| M4 drop `local_options` in pi lib (leak) | KILLED | zsh_pi_options (rc=81) |
4/4 killed.

## Findings
BLOCKING: none.
NON-BLOCKING:
- Only the `lookup` function and the pi helper are guarded; other quorum-artifacts.sh functions with globs (if any are later added) would need the same guard. None currently reachable from the cited callers.
- M1 reported only one failing arm (zsh_quorum_export_only) since later arms sharing the same failure were not all counted by the fail helper; the kill is still clear.
- Tests depend on /bin/zsh; they SKIP on non-Darwin without it (acceptable, Darwin fails loudly).

## Clean-tree check
`git status --porcelain` in /Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter61-judge: 0 lines; both libs byte-identical to backups.
