# Iteration 135 judge report (round 1)

Judge: sonnet, Agent tool, `MISSION-ROLE: evaluator`, foreground, read-only, own detached worktree `.fleet-i135-judge` at `dd9c8222a`. 62,818 tok, 13 tool uses, 187 s. The brief carried the ticket, the code location and the security constraint, not the controller's verdict.

VERDICT: PASS 92/100

**Blocking findings:** none.

**Non-blocking findings**
1. A new branch (no remote head) is still scanned only by its net diff from the merge-base with `origin/dev`. Harness add, then revert, then push as a new branch gives rc=0.
   - The old hook behaved the same way, so this is not a regression.
   - It is inconsistent with the rewrite path, which scans every commit.
   - Repro: in a throwaway repo, branch off dev, commit `tools/launchd/x.sh`, `git rm` it in a second commit, then `MISSION_NAME=triage git push origin b1` gives rc=0.
2. The `--force-with-lease` push shape was not run with the actual `--force-with-lease` flag. The judge used `--force`, which hands the hook the same remote and local SHAs. (The suite's arms (a), (b), (d) and (e) do use `--force-with-lease=refs/heads/rec:<old>`.)
3. Rewrite pushes now refuse fail-closed where the old hook fell back to a merge-base diff: remote commit not local, `ls-remote` failure, stale or missing `origin/dev`. Stricter, as intended, but it can block a legitimate push until the mission fetches.
4. A merge of dev into a rewritten branch diffs against the branch-side parent under `-m`, so inherited paths are refused. Spurious but conservative.

**Bypass attempts (throwaway repos, triage unless noted, remote = bare repo)**

| Shape | Result |
|---|---|
| Rebased record that inherits dev files | rc=0 (the ticket case works) |
| Rewrite with branch-authored harness add, then a `git rm` revert, then more commits | refused, rc=1 |
| Rewrite with the harness edit inside a merge commit that is later removed | refused, rc=1 |
| Rewrite that renames `design_docs/fleet-mission-1.md` to `notes.md` | refused, rc=1 (`--no-renames` exposes the source path) |
| New branch with a harness add | refused, rc=1 |
| Multiple refs in one push (benign `m1` plus harness `m2`) | refused, rc=1 |
| Fleet mission pushing a core path on a rewrite | refused, rc=1 |
| Branch delete | rc=0 (pushes no content, same as before) |
| Fast-forward push | checks only the delta from the remote head, as before |
| Path containing a newline | handled; paths are NUL-delimited |

**Mutation table** (scratch copy of the tree; the worktree was never edited)

| Mutant | Arm that goes red |
|---|---|
| Rewrite diffs from the old remote head | (a) rebased record accepts the inherited Fleet log |
| Per-commit scan disabled | (e) reverted harness change cannot hide |
| `ls-remote` check dropped | (d) stale dev |
| Missing `origin/dev` made fail-open | (d-missing) diagnostic |

All four scratch suites returned rc=1.

**Commands and results**
- `/bin/bash tools/launchd/test_mission_scope_guard.sh`: rc=0, 8 PASS, 0 FAIL.
- `shellcheck` on both files: rc=0.
- Bash 3.2: no `mapfile`, `${v,,}` or associative arrays.
- Temp files: 0 `mission-scope.*` left in `$TMPDIR` or `/tmp`.
- Scope: exactly three files (hook, test, `changelogs/unreleased/2026-11-24-scope-guard-rebased-push.md`).
