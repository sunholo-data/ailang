# Fleet iteration 135 · Scope guard for rebased record pushes

Triage #2450 followed the one-record PR procedure in
`.agents/skills/mission-control/resources/gate-3-triage.md` §7b. The controller
reproduced the rejection in a bare repo. The old remote tip and the rebased tip
differed on Fleet docs inherited from `dev`, while the PR diff contained only
triage docs. Appending the record to the existing remote head with a regular
fast-forward push is the immediate workaround when it applies cleanly. The
hook fix supports the prescribed rebase when a lease-protected rewrite is needed.

## 1. Select the base in `_scope_main`

Keep the mission/path verdicts and delete handling. Resolve refs to commit
SHAs once before using them in comparisons. Use Bash 3.2 syntax throughout.

| Push shape | Comparison base and failure behavior |
| --- | --- |
| Existing ref, `git merge-base --is-ancestor "$remote_sha" "$local_sha"` exits 0 | Keep `remote_sha`. Diff that tip against `local_sha`, so previously published branch changes are outside the check. This arm needs no `origin/dev`. |
| Existing ref, ancestry check exits 1 | Verify the local `refs/remotes/origin/dev` against the push target's advertised `refs/heads/dev`, then use `git merge-base "$local_sha" "$dev_sha"`. Diff that base against `local_sha` and check the rewritten commits as described below. |
| New branch, remote SHA is zero | Keep the merge-base comparison against local `refs/remotes/origin/dev` when it resolves. |
| Required commit or merge-base unavailable, ancestry command errors, or path enumeration fails | Refuse with a diagnostic naming the ref and the failed check. For missing dev data, tell the caller to fetch and retry. Remove the single-tip `git show` fallback because it can omit earlier pushed commits. |

For an existing non-fast-forward ref, read the exact `refs/heads/dev` SHA
using `git ls-remote --exit-code "$remote_url" refs/heads/dev` with stdin
from `/dev/null` and terminal prompts disabled. Require it to match the resolved
local tracking SHA. Refuse missing, failed, or mismatched results. Keep the
verification read-only. This extra lookup applies only to the newly relaxed
rewrite path. A stale tracking ref can only cause refusal on that path. A bare
merge-base comparison can hide a path when the branch restores its stale value,
and a rewritten dev can leave the tracking ref holding commits no longer on dev.

For that rewrite path, also enumerate `git rev-list "$dev_sha..$local_sha"`
and collect the paths each listed commit changes against each parent
(`git diff-tree --root --no-commit-id --name-only -r -m --no-renames`). Feed
these paths and the merge-base diff through the existing verdict loop. This
catches branch-authored changes canceled by subsequent commits, checks both
sides of renames, and conservatively checks merge commits against each parent.
Dev-reachable commits are excluded from the commit scan. The rewrite exempts
only inherited history reachable from the verified dev snapshot, and any
restricted path changed by a branch-authored commit still refuses the push.
Collect `--name-only -z` output in a cleaned-up temporary file and check each
Git command's status before consuming it with `IFS= read -r -d ''`. This keeps
the path verdicts accurate for filenames Git would otherwise quote.

## 2. Add real-push coverage and verify

Reuse the temp bare repo harness in `tools/launchd/test_mission_scope_guard.sh`.
Give each new arm its own `mktemp` directory under `/tmp` and clean it with a
trap. Keep all clones, bare repos, and logs there and isolate Git config with
`GIT_CONFIG_GLOBAL=/dev/null` and `GIT_CONFIG_SYSTEM=/dev/null`. Scrub inherited
config injection as the existing harness does. For each push under test, use
the same env-only `GIT_CONFIG_COUNT`/`KEY_0`/`VALUE_0` `core.hooksPath`,
`MISSION_SCOPE_TEST_REMOTE`, and `MISSION_NAME=triage`. Check setup commands
and assert the bare ref SHA after each push as well as exit status and refusal
diagnostics, so a lease error or setup failure cannot pass a negative arm.

- **(a) Rebased record passes.** Publish base `D0` to `dev` and publish `rec`
  with only `design_docs/triage-mission-log.md`. Advance `dev` with
  `design_docs/fleet-mission-log.md`, fetch, and rebase `rec` onto `origin/dev`.
  Assert that the old remote record head is no longer an ancestor, its tip
  diff includes the Fleet log, and the three-dot diff lists only the triage
  log. Push with an explicit lease for `refs/heads/rec` at its saved old SHA.
  Expect success and the new tip.
- **(b) Rebased record with its own harness path refuses.** Repeat (a) in a
  fresh repo, then add `tools/launchd/x.sh` in a branch commit after the rebase.
  Push with the same lease shape. Expect the guard to name `tools/launchd/x.sh`
  and the bare ref to stay at the old SHA.
- **(c) Fast-forward checks the new delta.** Fetch `origin/dev` and seed a
  remote record branch from it with a pre-existing `tools/launchd/x.sh`. Append only a
  triage log change and push normally. Expect success. Append a change to
  `tools/launchd/x.sh` and push normally again. Expect a path refusal and the
  bare ref to stay at the first accepted tip.
- **(d) Stale dev fails closed.** Build (a), then advance the bare `dev` ref
  from a separate fixture clone without fetching in the pushing clone. Expect
  a stale-dev diagnostic, no ref movement, and success after fetch and retry.
  In a fresh copy of (a), remove the local tracking ref before the push and
  expect a fetch-and-retry refusal.
- **(e) A later revert cannot hide an authored path.** Build (b), then remove
  `tools/launchd/x.sh` in a second branch commit so the net PR diff is doc-only.
  Expect the guard to name `tools/launchd/x.sh` and leave the bare ref unchanged.

Run the focused suite with each mutant applied alone to a scratch copy that
preserves the harness's relative paths to the driver, hook and skill fixtures.
Report the command, exit status, failing assertion and diagnostic. Restore
the fixed hook and rerun the suite.

| Mutation | Expected killer |
| --- | --- |
| Restore old base selection for an existing remote ref | (a) rejects the inherited Fleet log instead of passing. |
| Select the dev merge-base even for a fast-forward | (c)'s first push rejects the pre-existing harness path instead of passing. |
| Trust the local tracking SHA without verifying it | (d) allows the stale-dev push. |
| Drop the rewritten-commit path scan | (e) allows the doc-only net diff. |

Executor gates

1. `bash tools/launchd/test_mission_scope_guard.sh`
2. `shellcheck tools/launchd/githooks/pre-push tools/launchd/test_mission_scope_guard.sh` if installed. Record whether it ran.
3. `make test-launchd-drivers`. It already includes the hook test. Mark sandbox-related results **UNINFORMATIVE UNDER SANDBOX** and let the controller rerun outside it.

## Implementation files

- `tools/launchd/githooks/pre-push`
- `tools/launchd/test_mission_scope_guard.sh`
- `changelogs/unreleased/2026-11-24-scope-guard-rebased-push.md`, opening with `### Fixed — Scope guard accepts rebased mission record pushes`

These three files fit the fleet write scope. The existing shell harness covers
both milestones and requires no registry package.
