# Fleet iter 67 evaluation, round 1 (M-SWEEP-ACCOUNT-HOME)

Score: 88/100. PASS (threshold 70).

Rubric: correctness and root-cause fit 28/30; acceptance criteria and tests 24/25; mutation and test quality 14/15; scope, docs and charter hygiene 13/15; residual-risk handling and robustness 9/15.

## Diagnosis check: confirmed
- The evidence log shows the real pins `~/.ailang-driver-pin/{stapledon,triage,world}` removed with "(on origin/dev)".
- `scripts/worktree_sweep.sh` line 53 builds PROTECT from `$HOME`, so a synthetic HOME leaves the real pins unprotected.
- The hook (`scripts/hooks/git_health.sh`) launched the sweep with `--apply` and gave it no account-home protection.

## Drills (all run on disposable copies, or dry-run only)
1. Dry-run from the judge worktree under a temp HOME, no protect: the sweep prints REMOVE for real `.wt-fleet-iter66`, `-iter66-judge` and `-iter66-record`. The bug is reproduced.
2. Same dry-run with the protect string the fixed hook builds: every verdict is `KEEP ... protected path`, including a worktree that was REMOVE in drill 1. The fix closes the hole.
3. Hook self-test: `/bin/bash scripts/hooks/test_git_health.sh` gives 12 passed, 0 failed, rc 0. `bash -n` is clean, and shellcheck -S warning gives rc 0.
4. Mutation HOME_PREFIXES (use `$HOME` instead of the account home): arm A FAILs, and arm B FAILs as a side effect (10 passed, 2 failed).
5. Mutation DROP_CALLER_PROTECT (drop the `AILANG_SWEEP_PROTECT` suffix): arm B FAILs (11 passed, 1 failed).
6. Mutation LOOKUP_FAIL_OPEN (launch even if the lookup fails): arm C FAILs (got `0:3`, want `0:2`).
7. Prefix handling in `protected()`: it ignores empty fields and strips a trailing slash. It compares a nonexistent prefix as-is, and a prefix that exists is canonicalised with `pwd -P`. The account home is resolved with `getpwuid`, and `[ -d ]` makes the path physical, so these all hold.
8. Environment variables: none are new. The change reuses `AILANG_SWEEP_PROTECT`, and `AILANG_WORKTREE_SWEEP` already existed. The code is plain POSIX, so it runs under bash 3.2.
9. Charter diff: the `design_docs/fleet-mission.md` change is confined to the Guardrails dry-run bullet. Both `env -i` recipes carry `AILANG_WORKTREE_SWEEP=0`, with the incident note.
10. No `scripts/worktree_sweep.sh --apply` was run, and nothing was written under the real home.

## Findings
1. Non-blocking: the root cause stays in `worktree_sweep.sh`. Its defaults still derive from `$HOME`. The fix protects only the SessionStart hook caller. Any other `--apply` caller under a synthetic HOME is still exposed. I found none today, since the other references are tests and CI.
2. Non-blocking: the synthetic-HOME dry-run recipe now depends on `AILANG_WORKTREE_SWEEP=0` surviving `env -i` and being inherited by the claude lanes. I did not run the driver, so that is unverified. The hook-side protection is the real guard anyway.
3. Non-blocking: with a synthetic HOME the stamp and log stay HOME-relative. Removals in other repos or outside the four prefixes are still logged only to the synthetic home, as the changelog states.
4. Non-blocking: the self-test arm A also fails under the HOME mutation through arm B, so A's isolation is slightly weaker. This is minor.
5. Non-blocking: a lookup failure skips the sweep silently. There is no log line, which is acceptable for a hook with a 5s budget.

## Strongest objection
The fix hardens one caller. The sweep script itself should resolve the account home, because that closes the hole for every caller. Today any new `--apply` caller that runs under a synthetic HOME would repeat the 2026-10-28 incident.

EVALUATION_SCORE: 88/100
EVALUATION_VERDICT: PASS score=88
