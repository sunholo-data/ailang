Fleet mission iteration 135. Fixes ticket `push:scope-guard:rebased-record-inherited-sibling-paths` (filed by triage iteration 54 for #2450).

**Defect (reproduced first-party):** the pre-push mission scope guard diffed a non-fast-forward push against the old remote head. A record branch rebased onto a newer `dev` and force-pushed with a lease was therefore refused for every file the rebase inherited from `dev` (14 fleet docs in triage's case), while the PR's three-dot diff held only the branch's own triage docs.

**Fix (`tools/launchd/githooks/pre-push`):**
- Fast-forward push: unchanged; diff from the remote head.
- Rewrite (remote head not an ancestor): verify the local `origin/dev` equals the target's advertised `refs/heads/dev` (`git ls-remote`), then check the net diff from the dev merge-base **and** every branch-authored commit (`rev-list dev..local`, `diff-tree -m --no-renames`), so a later revert cannot hide a harness edit.
- New branch: merge-base with `origin/dev`, as before.
- Any failed lookup refuses the push with a fetch-and-retry diagnostic (fail closed). Paths are NUL-delimited.

**Tests:** `tools/launchd/test_mission_scope_guard.sh` gains five real-push arms: (a) rebased record passes, (b) rebased record with its own harness path refused, (c) fast-forward checks only the new delta, (d) stale/missing `origin/dev` refused and a fetch allows the retry, (e) a later revert cannot hide an authored harness path. Each asserts rc and the bare repo's ref SHA.

**Verification:**
- Planner and executor: `codex:gpt-6.2-sol`. Executor mutation drills: 4/4 killed (M1→a, M2→c, M3→d, M4→e).
- Controller, outside the sandbox: suite rc 0 (8 PASS); `shellcheck` rc 0 (base had 3 warnings, now 0); `make test-launchd-drivers` 60/60 suites; ticket repro now rc 0, own-harness-path control still refused; no temp files left; `git ls-remote` against the real GitHub remote matches `origin/dev`.
- Independent judge (sonnet): **PASS 92/100**, 0 blocking. It tried merge, rename, revert, multi-ref, newline-path and core-path bypasses; all refused.

🤖 Generated with [Claude Code](https://claude.com/claude-code)
