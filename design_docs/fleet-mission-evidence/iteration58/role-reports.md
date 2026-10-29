# Original role reports

These are historical role outputs. The final mission record controls observed landing facts; the controller’s metadata deferral and scope-guard publication adjustment are explicitly recorded there.

## Designer corrected final

**Assessment: resume the existing approved sprint through a controller-owned rebase and force-with-lease refresh. No new design or quorum is needed. This assessment does not clear the merge: live GitHub evidence remains unverified.**

I read `CLAUDE.md`, both requested skills, and all of `gate-3b-ci-green.md`. The resolved running mission-control `SKILL.md` and CI resource are byte-identical to the inspected `origin/dev` copies. No inbox, repo edits, shared-log writes, or git mutations were performed.

**Verified local evidence**

| Item | Inspected SHA |
|---|---|
| Detached HEAD and local `origin/dev` | `42d93e4b6317e7b068bf52cd32dc226c013dd117` |
| Local branch and tracking ref for `fleet/i54-routing-check-headings` | `71609162c099aab1498249dd36d2b0f0e226a775` |
| Common ancestor | `d25d9b8d0dde10ee7c14986fe21b94fd2cb500cd` |
| Dev’s parent | `8f58117f265471584176e2aeb45996c2c9822577` |

`git --no-optional-locks status --porcelain=v1 -b` reported only `## HEAD (no branch)`, both initially and finally: the worktree is clean.

The branch contains nine commits beyond the common ancestor; dev contains five. Comparing their changed-path sets produced **14 PR paths, 42 dev paths, zero overlap**. The gate’s prescribed `git diff --stat <base> origin/dev -- <PR paths>` returned empty output, exit 0. For these inspected trees, textual conflict risk is low; concurrent dev or branch advances require reassessment.

The 14 PR-side paths are:

```text
.agents/skills/mission-control/resources/gate-3-route.md
.agents/skills/mission-control/resources/gate-3b-ci-green.md
.ailang/state/sprints/sprint_M-FLEET-ROUTING-CHECK-HEADINGS.json
.claude/skills/mission-control/resources/gate-3-route.md
.claude/skills/mission-control/resources/gate-3b-ci-green.md
changelogs/unreleased/2026-10-21-routing-check-heading-shapes.md
design_docs/fleet-mission-dashboard.md
design_docs/fleet-mission-index.md
design_docs/fleet-mission-log.md
design_docs/fleet-mission-status-archive.md
design_docs/fleet-mission.md
design_docs/planned/sprint-plan-routing-check-heading-shapes.md
scripts/mission_routing_models_check.sh
scripts/test_mission_routing_models_check.sh
```

**Design and implementation remain applicable**

I read the plan and sprint JSON using `git show` from `71609162…`. The existing design artifact is `design_docs/planned/sprint-plan-routing-check-heading-shapes.md`: its summary explicitly says, “The ticket supplies the fixed design; no separate design doc.” The JSON’s `design_doc` and `markdown_plan` both reference that file. Creating another design would duplicate the established scope.

The implementation matches the relevant mechanisms:

- `scripts/mission_routing_models_check.sh:48–52`: broader Routing evidence heading recognition and form-dependent sibling boundaries.
- The same script, `canonical` and lines 62–67: allowlisted compound prefixes are accepted only when the remaining suffix contains no further `gpt`.
- `scripts/test_mission_routing_models_check.sh:run_arm`: arms `w/s/v/p/f/q` exercise those cases.
- `make/test.mk:test-launchd-drivers` already invokes the routing suite; its wiring is unchanged.

Both scripts passed Bash 3.2 syntax checks, and the PR-tree suite returned **46 passed, 0 failed, rc 0**. These checks ran in `/tmp/fleet-i58-designer-f6w9hdc4`, leaving the repo untouched. `git diff --check` also passed.

The branch additionally carries two documented Gate-5 changes: Bash execution guidance from iteration 54 and the GitHub-import restriction from iteration 57. Each resource remains identical across its `.agents` and `.claude` copies. Preserve these existing changes; the refresh adds no implementation scope.

The sprint JSON still says `planned`, with its milestone `not_started`. That is stale metadata, not evidence that implementation is absent. The branch’s iteration-54 record documents the implementation, controller corpus checks, and independent Sonnet **PASS 88**. I verified that recorded evidence; I did not independently recover the original evaluator transcript or rerun its corpus/mutation assessment.

**Rebase is required by the authoritative rule**

The inherited-base section of `.claude/skills/mission-control/resources/gate-3b-ci-green.md`, rule **(b)**, explicitly requires **“a rebase and force-push”** for a base-inherited PR red, after confirming disjointness. That confirmation succeeds locally for all 14 paths.

This supersedes iteration 57’s recorded “rebase only if … non-empty” next action. A merge-dev refresh could incorporate the base technically, but it does not follow the governing procedure. Rerunning the existing head leaves its source tree on the old base.

The dev-side changes include verified corrections in:

- `internal/eval_harness/models_pricing_consistency_test.go:pricingConsistencyKey`, introduced by `23c6638150f74a007ff243b535ffe58d3df9b120`.
- `.github/workflows/ci.yml:test-windows` and `internal/cihygiene/workflow_go_test_timeout_test.go:TestGoTestTimeoutIsDerived`, updated at `8f58117f…`.
- Dev head `42d93e4b…` itself changes only `docs/package-lock.json`.

Those source changes are verified. Their remote CI outcomes are not.

**Preserve records 54–57**

At the inspected PR head, `design_docs/fleet-mission-log.md` contains entries 54–57 at lines 1276, 1368, 1444, and 1510. The base log is an exact prefix of the PR log. STATUS 54 is preserved in `fleet-mission-status-archive.md:381`; STATUS 55–57 remain in the charter, and index rows 54–57 exist.

Rebase the complete branch history. Cherry-picking only the checker commits would lose these records and the later Gate-5 changes. Records mentioning #1929 and #1932 do not include those PRs’ implementations in this diff.

**Unresolved remote gates**

Read-only `gh pr view`, `gh pr checks --required`, commit check-run queries, and run/job queries for **39856325172** all failed with `error connecting to api.github.com`.

These results are **UNINFORMATIVE UNDER SANDBOX**. Consequently, I cannot verify:

- The live PR head, base, mergeability, or required checks.
- Whether run 39856325172 completed successfully on exact dev `42d93e4b…`.
- The supplied parent’s “40 checks” result or the characterization of its Dependabot failure.

The controller must verify these outside the sandbox. Any base red is a finding to report, without bypassing its gate.

**Controller steps and acceptance commands**

1. Re-read the live dev ref, PR identity, and reported run before mutation:

```bash
gh api repos/sunholo-data/ailang/git/ref/heads/dev --jq .object.sha
gh pr view 1927 --repo sunholo-data/ailang \
  --json headRefOid,baseRefOid,headRefName,mergeable,mergeStateStatus
gh api repos/sunholo-data/ailang/actions/runs/39856325172 \
  --jq '{head_sha,head_branch,event,path,status,conclusion,run_attempt}'
gh api --paginate \
  'repos/sunholo-data/ailang/actions/runs/39856325172/jobs?per_page=100'
```

The claimed dev verification requires the run’s full `head_sha` to equal `42d93e4b6317e7b068bf52cd32dc226c013dd117`, with completed successful CI and the expected jobs present. Parent green does not establish target green. A dev dispatch also does not satisfy the refreshed PR’s required checks.

2. After confirming unchanged refs and disjointness, rebase all nine branch commits onto the verified target in the controller-owned worktree. Publish using an explicit lease against the observed old head:

```bash
git push \
  --force-with-lease=refs/heads/fleet/i54-routing-check-headings:71609162c099aab1498249dd36d2b0f0e226a775 \
  origin HEAD:refs/heads/fleet/i54-routing-check-headings
```

3. Before adding iteration-58 records, verify the refreshed head and preservation:

```bash
target=42d93e4b6317e7b068bf52cd32dc226c013dd117
old=71609162c099aab1498249dd36d2b0f0e226a775
base=d25d9b8d0dde10ee7c14986fe21b94fd2cb500cd
new_head=$(gh pr view 1927 --repo sunholo-data/ailang \
  --json headRefOid --jq .headRefOid)

test "$new_head" = "$(git rev-parse HEAD)"
test "$new_head" != "$old"
git merge-base --is-ancestor "$target" "$new_head"
git range-diff "$base..$old" "$target..$new_head"
git diff --name-only -z "$base" "$old" |
  xargs -0 git diff --exit-code "$old" "$new_head" --
```

The final comparison must be empty for all 14 existing PR paths.

4. Run the plan’s shell/hygiene acceptance on the refreshed tree. Run `env -i … make test-launchd-drivers` outside the sandbox. Verify fresh exact-head PR checks, including required contexts:

```bash
gh pr checks 1927 --repo sunholo-data/ailang --required
gh api --paginate \
  "repos/sunholo-data/ailang/actions/runs?head_sha=$new_head&per_page=100"
gh api --paginate \
  "repos/sunholo-data/ailang/commits/$new_head/check-runs?per_page=100"
```

Expect PR workflows declared by `ci.yml`, `build.yml`, `docusaurus-deploy.yml`, `dashboard-ui-build.yml`, and `codeql.yml`; inspect completeness, events, and conclusions. Missing checks are not green.

Only after those gates pass should the controller merge with `--match-head-commit "$new_head"`. Then verify CI on the exact resulting merge SHA before recording LANDED or resolving the ticket.

DESIGN_DOC_PATH: design_docs/planned/sprint-plan-routing-check-heading-shapes.md

## Planner final

Implementation is complete and approved; landing remains incomplete. This refresh plans **0 production/test LOC**, no new design, JSON, package, dependency, or AILANG example. Registry reuse remains `none`. Preserve `M1_ROUTING_HEADING_SHAPES` and its prior evidence.

The JSON currently reads `planned/not_started` in both checkouts. Record that metadata discrepancy; **do not normalize, reset, or overwrite it**. The controller’s implementation-complete premise governs this refresh.

Allow roughly 15–25 minutes for preparation and review, plus remote CI: **30 minutes maximum per CI phase**. All commands below are proposed controller actions, not executed by this planner.

1. **Observe and preserve the existing landing state.**

   Run under `/bin/bash`, with fresh scratch storage. No branch switching:

   ```bash
   set -euo pipefail
   wt=/Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter54-routing-heading
   repo=sunholo-data/ailang
   branch=fleet/i54-routing-check-headings
   old=71609162c099aab1498249dd36d2b0f0e226a775
   base=42d93e4b6317e7b068bf52cd32dc226c013dd117
   evidence=$(mktemp -d /tmp/fleet-i58-planner-controller.XXXXXX)
   cd "$wt"

   gh_short() {
     /usr/bin/perl -e 'alarm 20; exec @ARGV; die "exec failed: $!";' gh "$@"
   }

   test -z "$(git status --porcelain=v1 --untracked-files=all)"
   test "$(git symbolic-ref --short HEAD)" = "$branch"
   test "$(git rev-parse HEAD)" = "$old"
   oldbase=$(git rev-parse 'd25d9b8d0^{commit}')

   gh_short pr view 1927 --repo "$repo" \
     --json state,isDraft,headRefName,headRefOid,baseRefName,baseRefOid,mergeable,mergeStateStatus \
     > "$evidence/pr-before.json"
   jq -e --arg h "$old" --arg b "$branch" \
     '.state=="OPEN" and .isDraft==false and .headRefOid==$h
      and .headRefName==$b and .baseRefName=="dev"' \
     "$evidence/pr-before.json"

   test "$(gh_short api "repos/$repo/git/ref/heads/dev" --jq .object.sha)" = "$base"
   gh_short run view 39856325172 --repo "$repo" \
     --json headSha,event,status,conclusion,jobs > "$evidence/dev-ci.json"
   jq -e --arg s "$base" '.headSha==$s' "$evidence/dev-ci.json"

   git diff --name-only "$oldbase" "$old" > "$evidence/pr-paths"
   git diff --name-only -z "$oldbase" "$old" > "$evidence/pr-paths.z"
   git ls-tree -r "$old" > "$evidence/tree-before"
   git log --format=fuller "$oldbase..$old" > "$evidence/history-before"
   shasum -a 256 \
     design_docs/planned/sprint-plan-routing-check-heading-shapes.md \
     .ailang/state/sprints/sprint_M-FLEET-ROUTING-CHECK-HEADINGS.json \
     > "$evidence/artifacts-before.sha256"
   ```

   Retain observation-only blob/hash snapshots for all iteration54–57 mission records and both copies of the current Gate5 edits. The whole-tree snapshot and PR-path inventory cover their committed content; explicitly identify those paths in the evidence. Never copy snapshots back into real state.

   Inspect exact-dev CI #39856325172, including expected jobs. Its dispatch, the green parent `8f58117`, and previous verdicts cannot establish refreshed-head green.

   **Acceptance:** clean correct branch/head; preserved artifacts/history; exact base recorded. Ref drift, dirty state, or contradictory observations parks the mutation step.

2. **Controller rebases now, preserving the original history.**

   The inherited `PricingIsSlugConsistent` red is addressed by dev fix `23c663815`. Gate3b requires a new head; iteration57’s conditional “rebase only if overlapping” advice is superseded. No pricing edits or cherry-pick.

   ```bash
   test -z "$(git status --porcelain=v1 --untracked-files=all)"
   test "$(git rev-parse HEAD)" = "$old"
   test "$(gh_short api "repos/$repo/git/ref/heads/dev" --jq .object.sha)" = "$base"

   # Preserve the original DAG locally; an existing archive name stops this step.
   git branch fleet/i58-routing-headings-pre-refresh "$old"
   git fetch --no-tags origin dev
   test "$(git rev-parse refs/remotes/origin/dev)" = "$base"

   git -c rebase.autoStash=false rebase --onto "$base" "$oldbase"
   fresh=$(git rev-parse HEAD)
   test "$fresh" != "$old"
   git merge-base --is-ancestor "$base" "$fresh"
   test -z "$(git status --porcelain=v1 --untracked-files=all)"

   git range-diff "$oldbase..$old" "$base..$fresh" > "$evidence/range-diff"
   git ls-tree -r "$fresh" > "$evidence/tree-after"
   git log --format=fuller "$base..$fresh" > "$evidence/history-after"

   while IFS= read -r -d '' path; do
     git diff --exit-code "$old" "$fresh" -- "$path"
   done < "$evidence/pr-paths.z"

   shasum -a 256 -c "$evidence/artifacts-before.sha256"
   git diff --check "$base...$fresh"
   ```

   **Acceptance:** identical content at every original PR path, all record/skill edits retained, and range-diff shows only the base refresh. Original commit history remains reachable through the archive branch. Conflict or unexpected patch change parks the refresh; do not improvise resolutions, reset, or clean.

3. **Executor verifies the exact refreshed head with focused gates.**

   Executor is inspection/testing only; controller owns every git mutation and final commit.

   ```bash
   test "$(git rev-parse HEAD)" = "$fresh"
   bash -n scripts/mission_routing_models_check.sh \
     scripts/test_mission_routing_models_check.sh
   /bin/bash -n scripts/mission_routing_models_check.sh \
     scripts/test_mission_routing_models_check.sh
   /bin/bash scripts/test_mission_routing_models_check.sh
   make check-changelog
   make check-file-sizes
   ```

   Require preserved base checks and green `w s v p f` arms. Retain prior mutation/corpus evidence; do not regenerate implementation work or run full Go suites.

   Controller additionally re-verifies outside the sandbox:

   ```bash
   /bin/bash scripts/test_mission_routing_models_check.sh
   env -i HOME="$HOME" PATH="$PATH" USER="$USER" \
     TMPDIR="${TMPDIR:-/tmp}" make test-launchd-drivers
   ```

   **Acceptance:** focused gates pass against full `$fresh`; worktree remains clean. Socket/network results under workspace-write are **UNINFORMATIVE UNDER SANDBOX**, requiring controller verification outside it.

4. **Push with the exact lease; inventory fresh CI completely.**

   Immediately before pushing:

   ```bash
   gh_short auth status --hostname github.com
   test "$(gh_short api user --jq .login)" = sunholo-voight-kampff
   test "$(git rev-parse HEAD)" = "$fresh"
   test -z "$(git status --porcelain=v1 --untracked-files=all)"
   test "$(gh_short api "repos/$repo/git/ref/heads/dev" --jq .object.sha)" = "$base"

   git push --force-with-lease="refs/heads/$branch:$old" \
     origin "HEAD:refs/heads/$branch"

   test "$(gh_short pr view 1927 --repo "$repo" \
     --json headRefOid --jq .headRefOid)" = "$fresh"
   gh_short pr checks 1927 --repo "$repo"
   gh_short api --paginate --slurp \
     "repos/$repo/actions/runs?head_sha=$fresh&per_page=100" \
     > "$evidence/pr-runs.json"
   ```

   Lease rejection parks; never weaken the lease.

   Derive an explicit expected workflow/job inventory from the **pinned-base and refreshed-head workflow definitions**, actual PR paths, event/branch/path filters, job conditions/matrices, and dev branch protection/rules:

   ```bash
   git ls-tree -r --name-only "$fresh" .github/workflows
   gh_short api "repos/$repo/branches/dev/protection/required_status_checks"
   gh_short api "repos/$repo/rules/branches/dev"
   ```

   Read the listed workflow definitions with `git show "$fresh:<path>"` and their base equivalents. Resolve inaccessible protection data through an authoritative controller inspection; an API error is not evidence of no requirements.

   For the inspected CI definition, these shell changes activate both `code` and `shell`. Expected runnable PR jobs include `changes`, `test`, `test-windows`, `build`, `lint`, `launchd-drivers`, `float-determinism-arm64`, and `govulncheck`. `docs` and `dependency-submission` have push-only conditions. Reconfirm this against the pinned definitions and enumerate other applicable workflows. Document legitimate skips/N/A individually.

   **Acceptance:** every expected workflow and job is present, correctly associated with `$fresh` and the PR event, and successful or explicitly justified by its configured skip condition. A dispatch green cannot replace missing PR-event evidence. Inspect mergeability alongside checks; absence of pending rows alone proves nothing.

5. **Bound CI observation and obtain an independent fresh-head evaluator.**

   Select and record each expected workflow’s numeric run ID and attempt from the exact-SHA inventory. Use this helper **once per selected run**, sharing one phase deadline:

   ```bash
   phase_deadline=$(( $(date +%s) + 1800 ))

   poll_one() {
     local target=$1 expected_event=$2 rid=$3 attempt=$4
     local row
     while [ "$(date +%s)" -lt "$phase_deadline" ]; do
       row=$(gh_short api "repos/$repo/actions/runs/$rid") || return 2
       printf '%s\n' "$row" | jq -e \
         --arg s "$target" --arg e "$expected_event" \
         --argjson id "$rid" --argjson a "$attempt" \
         '.id==$id and .head_sha==$s and .event==$e and .run_attempt==$a' \
         >/dev/null || return 2

       if printf '%s\n' "$row" | jq -e '.status=="completed"' >/dev/null; then
         printf '%s\n' "$row" | jq -e '.conclusion=="success"' >/dev/null
         return $?
       fi
       sleep 10
     done
     return 2
   }

   # Repeat for each recorded PR run, using its recorded attempt:
   # poll_one "$fresh" pull_request "$rid" "$attempt"
   # Then retrieve that attempt's complete job inventory:
   # gh_short api --paginate --slurp \
   #   "repos/$repo/actions/runs/$rid/attempts/$attempt/jobs?per_page=100"
   ```

   Reconcile job inventories with the expected manifest after polling. Each API call is capped at 20 seconds; controller tool reads stay bounded to at most 60 seconds. Keep the original deadline across reads. Missing data, parse/API errors, changed attempts, timeout, or incomplete inventories are **not green**. No unbounded watch.

   Independently evaluate full `$fresh`, its range-diff, preservation snapshots, focused results, and fresh remote evidence. Prefer cross-vendor evaluation; fresh-context same-model evaluation is allowed with an explicit independence `FLAG` if the preferred provider is unavailable. Historical PASS 80/88 is prior implementation evidence, never a fresh verdict.

   **Acceptance:** independent refreshed-head PASS; complete exact-dev and fresh-PR CI green. Any subsequent head change invalidates both fresh review and CI evidence.

6. **Controller merges only after all gates pass, then verifies exact merge CI.**

   Reserve the dev landing window against concurrent merges. Recheck the evaluator target, expected workflow/job completeness, required checks, mergeability, and unchanged base immediately before merging:

   ```bash
   test "$(gh_short api user --jq .login)" = sunholo-voight-kampff
   test "$(gh_short api "repos/$repo/git/ref/heads/dev" --jq .object.sha)" = "$base"
   test "$(gh_short pr view 1927 --repo "$repo" \
     --json headRefOid --jq .headRefOid)" = "$fresh"
   gh_short pr view 1927 --repo "$repo" \
     --json mergeable,mergeStateStatus,statusCheckRollup
   gh_short pr checks 1927 --repo "$repo"
   gh_short pr checks 1927 --repo "$repo" --required
   ```

   Only with every preceding acceptance criterion satisfied:

   ```bash
   gh_short pr merge 1927 --repo "$repo" --merge --match-head-commit "$fresh"

   gh_short pr view 1927 --repo "$repo" \
     --json state,mergedAt,mergeCommit > "$evidence/merged.json"
   jq -e '.state=="MERGED" and .mergeCommit.oid!=null' "$evidence/merged.json"
   merged=$(jq -er .mergeCommit.oid "$evidence/merged.json")
   gh_short api "repos/$repo/git/commits/$merged" > "$evidence/merge-commit.json"
   jq -e --arg b "$base" --arg h "$fresh" \
     '.parents[0].sha==$b and .parents[1].sha==$h' \
     "$evidence/merge-commit.json"
   ```

   No `--auto`, `--admin`, branch deletion, or merge while checks are pending/failing/missing. `--match-head-commit` guards the head; controller serialization and base rechecks guard base drift. Unexpected merge parents require explicit investigation.

   Derive expected **push-to-dev** workflows/jobs from the exact merge tree. CI’s push event activates the full matrix, including its push-only jobs. Discover runs using:

   ```bash
   gh_short api --paginate --slurp \
     "repos/$repo/actions/runs?head_sha=$merged&per_page=100" \
     > "$evidence/merge-runs.json"
   ```

   Record selected run IDs/attempts, initialize one new 30-minute deadline, and apply `poll_one "$merged" push "$rid" "$attempt"` per expected push run. Verify complete job inventories afterward. Keep the full merge SHA and selected IDs fixed across bounded reads; never substitute the latest dev run.

   **Landing complete:** exact merge SHA, expected push workflows/jobs present and green, and preserved records verified. If already merged but CI is incomplete/red, report **merged; landing verification parked**, not complete.

Stop/park on ref drift requiring a fresh base decision, dirty state, lease rejection, conflicts, unexpected content/history changes, fresh evaluator rejection, incomplete/error CI readings, or the deadline. Controller records the parked reason without rewriting iteration54–57 history. No duplicate JSON is warranted because this is a landing operation for the existing implemented sprint.

SPRINT_PLAN_PATH: design_docs/planned/sprint-plan-routing-check-heading-shapes.md
SPRINT_JSON_PATH: .ailang/state/sprints/sprint_M-FLEET-ROUTING-CHECK-HEADINGS.json

## Planner controlling addendum

# Iteration 58 landing-refresh controlling addendum

This addendum incorporates controller/designer facts delivered after the actual Sol planning launch. Read with `/tmp/fleet-i58-planner-run.deQ6pK.final.txt`; where they differ, this addendum controls. Existing approved design/plan remains `design_docs/planned/sprint-plan-routing-check-heading-shapes.md`; existing state remains `.ailang/state/sprints/sprint_M-FLEET-ROUTING-CHECK-HEADINGS.json`. No new design, duplicate JSON, package capability, examples, or production/test LOC. Registry reuse: none.

1. **Rebase/preservation stage already COMPLETE; do not repeat it.** Controller rebased all nine commits onto exact `42d93e4b6317e7b068bf52cd32dc226c013dd117`, yielding `0be2c1957a78b14aa281eb67f4c7e3efad82bb5a`. All nine range-diff rows were `=`; the explicit comparison of all 14 original PR paths returned rc0; tree was clean. Backup is `fleet/i58-before-refresh`, old `71609162c099aab1498249dd36d2b0f0e226a775`. This satisfies the original immediate-post-rebase preservation requirement. Preserve all iteration 54–57 records and both Gate 5 edits thereafter. The old PR remote head is still the explicit lease target, subject to live confirmation.

2. **Executor may annotate only the two EXISTING plan/state files, then run focused validation.** Designer independently confirmed source implementation and prior iteration 54 judge PASS 88. Literal D7 includes controller PR/CI/ticket resolution, so the milestone is integration-pending. Add precise implementation evidence and explicitly outstanding D7 to existing notes/progress; retain truthful noncompleted `passes`/status, no `completed` timestamp, no fabricated `actual_days`, LOC or throughput. Do not mark every acceptance box complete: distinguish established historic implementation evidence from fresh landing checks. This is deliberate metadata cleanup AFTER the verified 14-path byte-preservation point. No source, test, fragment, history, prior record or skill rewrite. Controller owns commits and all git writes.

   Focused executor gates on the refreshed tree:
   ```bash
   /bin/bash -n scripts/mission_routing_models_check.sh scripts/test_mission_routing_models_check.sh
   /bin/bash scripts/test_mission_routing_models_check.sh
   make check-changelog
   make check-file-sizes
   git diff --check
   jq -e . .ailang/state/sprints/sprint_M-FLEET-ROUTING-CHECK-HEADINGS.json
   ```
   Retain prior mutation/corpus evidence; no full local Go suite. Controller inspects diff side effects then runs the planned clean-environment `make test-launchd-drivers` outside sandbox. Sandbox GitHub/socket verdicts are UNINFORMATIVE UNDER SANDBOX. Require all 46 shell assertions green and unchanged scripts/Gate 5/history. Controller commits only listed metadata edits and derives final full candidate SHA again. Do not evaluate an earlier head and then silently add metadata.

3. **Fresh exact-head evaluator and remote PR checks.** Prefer cross-vendor evaluator; same-model fresh context is allowed FLAG when the preferred provider is unavailable. Target final candidate SHA, original approved plan, prior PASS 88 evidence, controller-preserved tree/range-diff evidence, metadata truthfulness, and fresh focused results. Evaluator must PASS before merge. Controller verifies active `sunholo-voight-kampff`, unchanged exact base and lease before publication:
   ```bash
   gh auth status --hostname github.com
   test "$(gh api user --jq .login)" = sunholo-voight-kampff
   test "$(gh api repos/sunholo-data/ailang/git/ref/heads/dev --jq .object.sha)" = 42d93e4b6317e7b068bf52cd32dc226c013dd117
   git push --force-with-lease=refs/heads/fleet/i54-routing-check-headings:71609162c099aab1498249dd36d2b0f0e226a775 origin HEAD:refs/heads/fleet/i54-routing-check-headings
   ```
   These are controller-owned commands, in the clean existing sprint worktree, under bounded execution. Lease rejection/ref drift is a finding, never permission for an unleased push. Dev CI 39856325172 must be terminal and evaluated at exact dev 42d93e4b6; prior parent green is insufficient. Derive expected PR workflow/job set from actual final-head workflow definitions, event/path/matrix filters and protection/rules; require every expected workflow/job present and green. Neither missing checks nor a dispatch green replaces PR-event checks. Preserve exact final SHA and run ID/attempt selection throughout. Model final contains full commands and bounded polling recipe; inspect parsed payloads, not absence of failures.

4. **Merge only green, then exact merge CI; completion remains pending until then.** Immediately re-read PR full head and dev full base, required/all checks, mergeability and evaluator target. With every acceptance criterion green:
   ```bash
   gh pr merge 1927 --repo sunholo-data/ailang --merge --match-head-commit "$final_head"
   gh pr view 1927 --repo sunholo-data/ailang --json state,mergedAt,mergeCommit
   gh api --paginate "repos/sunholo-data/ailang/actions/runs?head_sha=$merge_sha&per_page=100"
   ```
   Derive `$merge_sha` from that actual merge record, never latest dev. Assert expected merge parents/tree; select exact-SHA push workflow IDs/attempts; require full expected job inventory and terminal green. Use one 30-minute deadline per phase with <=60s controller reads, capped individual network calls, no unbounded watch. API/parse/incomplete/missing/changed-attempt readings are not green. If merged but CI red/missing, report merged; landing verification pending and park honestly. Final sprint completion metadata belongs Gate 4 only AFTER exact merge CI and ticket resolution, with observed evidence; no premature completion or invented effort.

Operational estimate: 15–25 minutes preparation/review plus up to 30 minutes per remote CI phase; no implementation estimate or invented velocity. Stop for unexpected diff/conflict, dirty state, changed base/head, rejected lease, failed fresh evaluator, red/incomplete checks or deadline. Report the specific remaining gate; never rerun unrelated broad tests to fill the gap.

Actual routing evidence: `codex:gpt-6.1-sol declared:planner-lane-default-pin`; bounded 120s probe rc0; bounded 600s actual run rc0. CLI totals printed `23.351` probe and `48.483` actual (23,351 and 48,483 tokens respectively; 71,834 combined). Actual log `/tmp/fleet-i58-planner-run.deQ6pK`; final `.final.txt`; rc `.rc`; directive `/tmp/fleet-i58-planner-pwk4vdws.directive.txt`. Worktree `.planner-wt-iter58` remained clean. Controller owns cleanup/removal.

SPRINT_PLAN_PATH: design_docs/planned/sprint-plan-routing-check-heading-shapes.md
SPRINT_JSON_PATH: .ailang/state/sprints/sprint_M-FLEET-ROUTING-CHECK-HEADINGS.json


## Executor final

# Fleet iteration58 executor scratch report

TEST/INSPECTION ONLY — no implementation code needed.
Worktree: /Users/voightkampff/.ailang-driver-pin/.wt-fleet-iter54-routing-heading

Before and after assertions: PASS
- Branch: fleet/i54-routing-check-headings
- Full HEAD: 0be2c1957a78b14aa281eb67f4c7e3efad82bb5a
- Full status (git --no-optional-locks status --porcelain=v1 --untracked-files=all): empty, clean including untracked.
- Final branch/status/HEAD commands each rc=0; exact-value assertions passed.

| Gate | Actual rc | Useful counts/results |
| --- | --- | --- |
| /bin/bash -n scripts/mission_routing_models_check.sh scripts/test_mission_routing_models_check.sh | 0 | 2 scripts; no syntax diagnostics |
| /bin/bash scripts/test_mission_routing_models_check.sh | 0 | 46 passed, 0 failed; base 20/20; w=4, s=4, v=2, p=2, f=2; q=12 |
| make check-changelog | 0 | 1 valid fragment; current Unreleased heading and index link checks pass |
| make check-file-sizes | 0 | All files within 800-line limit; no file total printed |
| git --no-optional-locks diff --check | 0 | No whitespace diagnostics |

All printed arms green: base a/b/c/d/e and argument, boundary, case, EOF checks; w world; s bulleted heading/sibling boundary; v continuation; p parenthetical/table; f bold-label bullets; q compound model tokens.
No sandbox/network/socket denials occurred. No extra test suite, launchd suite, mutation tests, network, inbox, gh, agents, model switches, startup scripts, or git writes performed.
Zero repository files changed. Sprint JSON remains planned/not_started and was not edited; no effort metrics recorded.
Historical PASS88 is prior implementation evidence, not a fresh evaluator verdict.
D7 REMAINS OPEN — controller PR/CI/ticket resolution outstanding; PR1927 remote evidence and preservation verification remain controller-owned.

## Actual focused shell-suite output

PASS a canonical allowed aliases rc=0
PASS b retired routing rc=1
PASS b original lines and every token
PASS c ignore prose outside routing rc=0
PASS d missing entry is instrument failure rc=2
PASS e last entry in file order rc=0
PASS e explicit non-last entry rc=1
PASS e file order over numeric order rc=0
PASS missing file rc=2
PASS missing block rc=2
PASS missing arguments rc=2
PASS unknown argument rc=2
PASS missing file value rc=2
PASS invalid entry rc=2
PASS missing entry value rc=2
PASS duplicate file rc=2
PASS case and punctuation rc=0
PASS entry boundary without next field rc=0
PASS inline token at EOF rc=1
PASS original token case
PASS w world allowed routing rc=0
PASS w clean findings
PASS w world retired routing rc=1
PASS w original line/token
PASS s indented retired routing rc=1
PASS s original line/token
PASS s sibling-only retired token excluded rc=0
PASS s clean sibling findings
PASS v continuation retired routing rc=1
PASS v original line/token
PASS p parenthetical heading table routing rc=1
PASS p original line/token
PASS f bold-label bullets retained rc=1
PASS f original line/token
PASS q allowed model in ticket compound rc=0
PASS q clean ticket findings
PASS q retired model in compound rc=1
PASS q original compound line/token
PASS q allowed dotted model in compound rc=0
PASS q clean dotted findings
PASS q reject gpt-6.1-sol-gpt-5 rc=1
PASS q original gpt-6.1-sol-gpt-5 line/token
PASS q reject gpt-6-luna-gpt-5.6-sol rc=1
PASS q original gpt-6-luna-gpt-5.6-sol line/token
PASS q reject gpt-6-lunatic rc=1
PASS q original gpt-6-lunatic line/token
46 passed, 0 failed
