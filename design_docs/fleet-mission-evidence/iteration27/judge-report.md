# Iteration 27 — evaluator report (verbatim summary)

Evaluator: `sonnet` via the Agent tool (resolver `agent-tool sonnet declared:alias-pin`). Generator: codex `gpt-6.1-sol`. Judge independence: cross-vendor.
Subject: `origin/fleet/i27-ff-sync` head `4fe2da1f8d24e40846ff0db7ad62f878618c210e`, base `c92739681`, own detached worktree `fleet-iter27-evaluator`.

**VERDICT: PASS, 91/100. 0 blocking.**

| Category | Score |
|---|---|
| Correctness / contract (every D-FLEET-15 guard present; nothing wider) | 28/30 |
| Driver seam (after kill switch :1625 and pidfile yield :1908; dry-run report only; apply once before the boot stagger; cannot stop a fire) | 15/15 |
| Tests (~38 arms, driver-seam arms, spy, watchdog, mutation harness) | 21/25 |
| bash 3.2 portability | 10/10 |
| Docs / changelog | 9/10 |
| Scope / safety | 8/10 |

Re-ran: suite and `--mutations` rc 0, all executor arms killed. Scope check: only `tools/launchd/**`, `make/test.mk`, `changelogs/**`, `design_docs/**`, `.ailang/state/sprints/**` changed.

Own mutation drills (on a backed-up copy, restored with `cp`, `cmp`-identical, `git diff --stat` empty):

| Mutation | Result |
|---|---|
| drop `--untracked-files=all` | GREEN — equivalent (the prefix check still matches the untracked directory) |
| prefix `[[ "$d" = "$incoming/"* … ]]` → `false` | RED |
| remove `GIT_OPTIONAL_LOCKS=0` | GREEN — **untested guard** |
| `merge --ff-only` → `merge` | RED |
| drop `-M -C` | GREEN — equivalent (a rename shows as a delete plus an add on the same paths) |
| remove the `ss_ahead != 0` check | RED |

Non-blocking findings:
1. No arm asserts `GIT_OPTIONAL_LOCKS=0` (`tools/launchd/lib/skill-sync.sh:14`). "Report never writes" rests on that one flag. Add an arm on the index file's mtime and sha, or a spy on the environment.
2. Two survivors are equivalent mutations; a comment saying so would help.
3. The final preflight is not atomic (disclosed in the header); Git's locks are the last line of defence.
