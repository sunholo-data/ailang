# Sprint Plan: M-DOCS-GATE-NOT-REQUIRED

**Spec**: GitHub issue [#497](https://github.com/sunholo-data/ailang/issues/497) — there is **no** design doc; #497 *is* the spec.
**Option chosen**: **(a)** — rename the colliding job + add an always-reporting wrapper job that can safely be required.
**Target**: v1.0.0 · repo-hygiene / supply-chain gate
**Sprint ID**: `M-DOCS-GATE-NOT-REQUIRED`
**Planned at**: worktree `.claude/worktrees/sprint-m-docs-gate`, branch `sprint/m-docs-gate-not-required`, base `ac6918896`
**Duration**: 0.5 day nominal · 1.0 day hard ceiling (dominated by CI wall-clock, not authoring)
**Risk**: **HIGH on M6 only** — M6 changes branch protection on a **public** repo and can block every contributor's PR. M1–M3 are a single-file workflow edit (LOW).
**Milestones**: 7 (M1–M3 executor · M4–M7 controller)

---

## 0. Planner verification — what I re-checked first-party

Everything below was observed by me at plan time. **Provenance is labelled on every claim.**
Two of the controller's hypotheses were **CONFIRMED with new empirical evidence**, one was
**REFUTED**, and one open question was **RESOLVED**.

### 0.1 Facts confirmed (first-party, this session)

| Claim | Evidence | Verdict |
|---|---|---|
| `docusaurus-deploy.yml` declares `jobs: build:` at line 58 | file read | ✅ |
| `ci.yml` declares `jobs: build:` at line 315 with **no** `name:` override → context is literally `build` | file read + `grep '^    name:'` | ✅ |
| `on.pull_request` is path-filtered (lines 23–35) | file read | ✅ |
| Workflow-level `concurrency: {group: "pages", cancel-in-progress: false}` (lines 52–54) | file read | ✅ |
| `dependabot-automerge.yml` runs `gh pr merge --auto --squash` for patch/minor | file read | ✅ |
| Third-party actions are SHA-pinned by convention | `SonarSource/...@713881`, `astral-sh/setup-uv@11f9893` | ✅ |
| Nothing has landed for this item | no design doc, no PR, no matching commit on `origin/dev` | ✅ |

### 0.2 Full `dev` branch protection (exact, captured at plan time — this is the rollback source of truth)

```json
{
  "required_status_checks": {
    "strict": false,
    "contexts": ["test", "lint", "build"],
    "checks": [
      {"context": "test",  "app_id": 15368},
      {"context": "lint",  "app_id": 15368},
      {"context": "build", "app_id": 15368}
    ]
  },
  "required_signatures":              {"enabled": false},
  "enforce_admins":                   {"enabled": false},
  "required_linear_history":          {"enabled": false},
  "allow_force_pushes":               {"enabled": false},
  "allow_deletions":                  {"enabled": false},
  "block_creations":                  {"enabled": false},
  "required_conversation_resolution": {"enabled": false},
  "lock_branch":                      {"enabled": false},
  "allow_fork_syncing":               {"enabled": false}
}
```

Notes that matter:
- `required_pull_request_reviews` and `restrictions` are **absent** (null). A full `PUT /protection`
  would require them as explicit keys — see §0.5.
- `app_id: 15368` is the GitHub Actions app. New contexts should pin the same app_id so a
  third-party app cannot claim the `docs-gate` context.
- **`enforce_admins: false`** — this bounds the blast radius of M6: the agent token
  (`sunholo-voight-kampff`, `admin: true`) can still merge/push even if the gate wedges.
  Non-admin contributors and Dependabot cannot. Dependabot's failure mode is *stall*, not
  *bad merge* — fail-safe.

### 0.3 RESOLVED: the token **does** have admin write

`gh api repos/sunholo-data/ailang --jq .permissions` →
`{"admin": true, "maintain": true, "push": true, "pull": true, "triage": true}`,
token scopes `gist, read:org, repo, workflow`, account `sunholo-voight-kampff`.

**M6 will not park for a human on permission grounds.** The `workflow` scope is also present,
which is required to push changes to `.github/workflows/**` — M4 will not be rejected on push.

### 0.4 CONFIRMED (H2) — and it is **already happening today**, not hypothetical

The controller flagged the shared `pages` concurrency group as a trap. It is worse than a
hypothesis. Recent `docusaurus-deploy.yml` runs, via
`gh api "repos/sunholo-data/ailang/actions/workflows/docusaurus-deploy.yml/runs"`:

```
2026-07-27T12:45:16Z  pull_request  cancelled   (11s after creation)
2026-07-27T12:45:26Z  pull_request  cancelled   (34s)
2026-07-27T12:45:59Z  pull_request  cancelled   (14s)
2026-07-27T12:46:12Z  pull_request  cancelled   (29s)
2026-07-27T12:46:40Z  pull_request  success     (17.5 min)
2026-07-27T22:41:55Z  pull_request  cancelled   (43s)
2026-07-27T22:42:37Z  pull_request  success     (21.8 min)
```

Four runs cancelled inside 60 seconds while one survived — the documented single-group behaviour:
*at most one run in progress and one pending per concurrency group; additional queued runs are
cancelled* ([workflow-syntax#concurrency](https://docs.github.com/en/actions/reference/workflows-and-actions/workflow-syntax#concurrency)).

A `cancelled` check run has conclusion `cancelled`, which is **not** success → a required
`docs-gate` on a cancelled run **blocks the PR**. Today this is invisible because the workflow
is neither required nor triggered on most PRs. **Dropping the `on.pull_request` paths filter
without also rescoping concurrency would convert this into repo-wide PR blockage.**
M3 is therefore not optional polish — it is a prerequisite for M6.

### 0.5 RESOLVED: use the `required_status_checks` **sub-resource**, not a full PUT

`PATCH /repos/{owner}/{repo}/branches/{branch}/protection/required_status_checks` exists and
takes `{strict, contexts, checks}`. It touches only required status checks and cannot drop
`enforce_admins` / `allow_force_pushes` / etc.

`PUT /repos/{owner}/{repo}/branches/{branch}/protection` **requires** `required_status_checks`,
`enforce_admins`, `required_pull_request_reviews` and `restrictions` as explicit keys — omitting
them is a destructive no-op-looking change. **M6 must use PATCH on the sub-resource.**

### 0.6 REFUTED — H4's premise is backwards (this is the most important correction in this plan)

> H4 (controller): *"the gate job must use `if: always()` … Naive `needs:` alone makes the gate skip
> when its dependency skips — **a skipped required check never reports**."*

**This is not how GitHub behaves.** Per
[Troubleshooting required status checks](https://docs.github.com/en/pull-requests/collaborating-with-pull-requests/collaborating-on-repositories-with-code-quality-features/troubleshooting-required-status-checks):

- A job **skipped by an `if:` conditional** reports its status as **Success** and does **not**
  block the PR.
- A **workflow skipped by path/branch filtering** leaves the check **Pending** and **does** block.

So the two failure modes are *inverted* from H4:

| Situation | Actual effect | H4 predicted |
|---|---|---|
| Gate job skipped via `needs`/`if:` | **False GREEN** (reports Success) | wedge |
| Workflow never triggers (paths filter) | **Wedge** (permanent Pending) | — |

Consequences for the design:

1. **H1 is CONFIRMED and is the essential fix.** Dropping `paths:` from `on.pull_request` is
   what stops the check sitting Pending forever. Without it nothing else matters.
2. **`if: always()` is still required**, but for a *different reason*: without it, a broken
   detector or a failed heavy build would let the gate **silently report Success**. That is a
   textbook silent fallback (CLAUDE.md §2) and it defeats the entire purpose of #497 — the gate
   would look wired in and gate nothing, which is exactly the bug the issue was filed about.
3. It also **kills the tempting shortcut** of requiring `docs-build` directly with an `if:`
   guard on it: when skipped it would report Success, meaning "we did not check" is
   indistinguishable from "we checked and it passed". Rejected for that reason — see §1.2.

**Net: the wrapper job survives, but it is justified by fail-loudly, not by reporting mechanics.**
The plan's acceptance criteria are written against the *real* semantics.

### 0.7 REFUTED (minor) — `build.yml` is **not** a third colliding context

`build.yml` also has `jobs: build:`, but with `name: Build ${{ matrix.os }}`, so its contexts are
`Build ubuntu-latest`, `Build macos-latest`, … The collision is exactly the two jobs #497 names.
The controller's premise holds; I checked for a third and there isn't one.

### 0.8 Local tooling — one correction to the controller's brief

| Tool | Status | Use |
|---|---|---|
| `actionlint` | ✅ `/opt/homebrew/bin/actionlint` v1.7.12 | primary validation |
| `python3 -c 'import yaml'` | ❌ **`ModuleNotFoundError: No module named 'yaml'`** | **do not use** |
| `uv run --with pyyaml python -c 'import yaml'` | ✅ pyyaml 6.0.3 | YAML parse check |
| `yq` | ❌ not installed | — |
| `jq`, `gh`, `node`, `git` | ✅ | — |

**actionlint baseline (must be recorded before editing):**
`actionlint .github/workflows/docusaurus-deploy.yml` currently exits **1** with **exactly one**
finding: `:110:9 shellcheck SC2155` (the codebase-statistics step). Acceptance is
*"exactly this one pre-existing finding, no new ones"* — not "clean".

### 0.9 SYSTEMIC finding (CLAUDE.md §3 audit) — the same hole exists for `/ui`

`dashboard-ui-build.yml` is the **identical pattern**: `on.pull_request` path-filtered to
`ui/**`, job `ui-build`, **not** in required checks. `dependabot.yml` has an npm ecosystem for
`/ui` (live branches: `dependabot/npm_and_yarn/ui/typescript-7.0.2`, `.../ui/vite-8.1.5`), and
`dependabot-automerge.yml` is ecosystem-agnostic. **A UI-breaking minor bump can auto-merge
exactly the way #488 did.**

Out of scope for this sprint (#497 is docs-scoped), but the gate pattern built here is directly
reusable. **M7 files a follow-up issue.** Do not silently expand this sprint to cover it.

### 0.10 Incidental observations (recorded, NOT in scope)

- The `main` branch **does not exist** (`gh api repos/.../branches` lists no `main`; default is
  `dev`). `docusaurus-deploy.yml` references `main` in `on.push.branches`, `on.pull_request.branches`
  and `deploy.if`. Harmless dead config. **Leave it.**
- Workflow-level `permissions: {pages: write, id-token: write}` are granted to every job, but only
  `deploy` needs them. Least-privilege tightening is a real improvement and this repo runs
  Scorecard — but it risks the deploy path for no gate benefit. **Deferred**, noted in M7.
- `ci.yml` has **no** `paths:` filter on `pull_request`, so `test`/`lint`/`build` run on every PR.
  This is what makes the M5 probe PR viable.

---

## 1. Design

### 1.1 Chosen shape — three jobs

```
docs-changes  (always runs)  ──> docs-build (heavy, conditional) ──> deploy (dev/main only)
      └──────────────────────────────────┴──> docs-gate  (if: always(), REQUIRED CHECK)
```

| Job id | Context name | Runs when | Required? |
|---|---|---|---|
| `docs-changes` | `docs-changes` | every trigger | no |
| `docs-build` | `docs-build` | `docs_changed == 'true'` | no |
| `docs-gate` | `docs-gate` | **always** | **yes (M6)** |
| `deploy` | `deploy` | ref is dev/main **and** `docs-build` succeeded | no |

**Name-collision check (done, first-party).** Every job id and `name:` across all 10 workflows:
`test`, `test-windows`, `build`, `docs`, `lint`, `govulncheck (vuln gate)`, `dependency-submission`,
`Build ${{matrix.os}}`, `Create Release Bundle`, `Analyze Go`, `UI build stage`, `automerge`,
`deploy`, `Build WASM Binary`, `Build Bootstrap Content Bundle`, `Build Examples Bundle`,
`Build Release ${{matrix.os}}`, `Generate error_codes.json`, `Create GitHub Release`,
`Publish Release`, `provenance`, `Scorecard analysis`, `Test AILANG -> Go Codegen Pipeline`, `eval`.
**`docs-changes`, `docs-build`, `docs-gate` are all free.** (`docs` is taken by `ci.yml` — do not use it.)

### 1.2 Alternatives considered and rejected (decision record)

| Alternative | Rejected because |
|---|---|
| Require `docs-build` directly with an `if:` guard on it (no wrapper) | Skipped ⇒ reports Success (§0.6). "Not checked" becomes indistinguishable from "checked and passed"; a failed detector silently greens. Violates CLAUDE.md §2. |
| One always-running `docs-build` job with `if:` on each of its ~12 heavy steps | Forgetting one `if:` silently runs a heavy step; 12 conditions to keep correct vs 1. |
| New separate `docs-gate.yml` workflow calling docusaurus as a reusable workflow | More moving parts, duplicated triggers, and still needs the concurrency fix. |
| Option (b) — exclude `/docs` npm from auto-merge | #497 calls it the cheap stopgap; loses batching, and leaves the name collision + the `/ui` twin. |
| Option (c) — make auto-merge wait on the docs run | Encodes the gate in one consumer instead of in branch protection; every other merge path stays unguarded. |

### 1.3 Path detection — no third-party action (H3 CONFIRMED, with a working implementation)

Detection uses `git diff --name-only <merge-base> HEAD -- <pathspecs>`. **Git's own pathspec
matching does the work** — a directory prefix like `docs/` natively matches everything beneath it,
so no glob library, no `dorny/paths-filter`, and no break in the SHA-pinning convention.

**I verified this locally against real history in the worktree:**

```
PATHSPECS=(docs/ prompts/ llms.txt CHANGELOG.md \
           .github/workflows/docusaurus-deploy.yml internal/ cmd/ go.mod go.sum web/)

$ git diff --name-only ac6918896^ ac6918896 -- "${PATHSPECS[@]}"
(empty)                       # design_docs-only commit  -> docs_changed=false  -> SKIP

$ git diff --name-only 12e5df162^ 12e5df162 -- "${PATHSPECS[@]}"
CHANGELOG.md
cmd/ailang/modal_rand_e2e_test.go
docs/docs/guides/parameterised-effects.md    # -> docs_changed=true -> BUILD
```

These two SHAs are the executor's **local test vectors** (M2 acceptance).

### 1.4 Single source of truth for the path list

`on.push.paths` cannot read a file, so the list would otherwise exist twice. Resolution:

- The **gate's** list lives in a new file **`.github/docs-build-paths.txt`** (one git pathspec per
  line, `#` comments allowed). After M2, `on.pull_request` has **no** `paths:` at all, so for PRs
  this file is the *only* list.
- `on.push.paths` stays untouched (M3 does not change push/deploy behaviour).
- A drift guard asserts the safety-relevant direction only: **every `on.push.paths` entry is
  covered by `.github/docs-build-paths.txt`** (gate ⊇ push). A gate list that is a strict superset
  is safe (over-validates); a gate list missing an entry is a false green.

### 1.5 Concurrency rescope (from §0.4)

```yaml
# workflow level — per-ref, so PRs never contend with each other or with dev
concurrency:
  group: ${{ github.workflow }}-${{ github.ref }}
  cancel-in-progress: false

jobs:
  deploy:
    # the singleton that actually matters: only one Pages deployment at a time
    concurrency:
      group: "pages"
      cancel-in-progress: false
```

`github.ref` is `refs/pull/<N>/merge` for PR events, so every PR gets its own group. Keep
`cancel-in-progress: false`: with `queue: single`, when a branch is pushed repeatedly the
*newest* queued run survives, and only the newest SHA's checks matter to branch protection.

---

## 2. Milestones

Ownership legend: **[E]** = executor (`codex:gpt-5.6-sol`, `--sandbox workspace-write`, no
network, **cannot commit**) · **[C]** = controller (all `gh`, all git push/commit, all CI
observation, all branch-protection changes).

### M1 — Rename `build` → `docs-build` **[E]**

Kill the latent context collision with `ci.yml`'s required `build`. Worth doing on its own merit.

Changes to `.github/workflows/docusaurus-deploy.yml`:
1. `jobs.build:` → `jobs.docs-build:` (line 58)
2. `deploy.needs: build` → `needs: docs-build` (line 181)
3. Update the `# Build job` comment.

**Do not** add a `name:` override — the context must be the job id `docs-build`.

**Acceptance (all locally checkable):**
- [x] `grep -n '^  docs-build:' .github/workflows/docusaurus-deploy.yml` returns exactly 1 line.
- [x] `grep -c '^  build:' .github/workflows/docusaurus-deploy.yml` returns `0`.
- [x] `grep -n 'needs: docs-build' …` present under `deploy`.
- [x] `actionlint .github/workflows/docusaurus-deploy.yml` reports **exactly one** finding, the
      pre-existing `:110:9 SC2155`, and no others (compare against the baseline captured in M1 step 0).
- [x] No other file in the repo references the docs `build` job: `grep -rn 'docusaurus-deploy' --include='*.yml' --include='*.yaml' --include='*.md' .` reviewed, no `needs:`/context references found.

**Est.**: 5 LOC. **Risk**: LOW.

### M2 — Always-trigger + `docs-changes` + `docs-gate` **[E]**

**2a. Trigger.** Delete the `paths:` block from `on.pull_request` (lines 25–35). Keep
`branches: ["dev", "main"]`. **Do not touch `on.push`.** Leave a comment explaining that the path
decision moved into `docs-changes` because a path-filtered required check sits Pending forever.

**2b. New file `.github/docs-build-paths.txt`** — git pathspecs, one per line, mirroring
`on.push.paths`:

```
# Single source of truth for whether a PR needs the docusaurus build.
# Consumed by the `docs-changes` job in .github/workflows/docusaurus-deploy.yml.
# MUST cover every entry in that workflow's `on.push.paths` (enforced by the drift guard).
docs/
prompts/
llms.txt
CHANGELOG.md
.github/workflows/docusaurus-deploy.yml
internal/
cmd/
go.mod
go.sum
web/
```

**2c. `docs-changes` job.** `runs-on: ubuntu-latest`, `outputs.docs_changed`, checkout with
`fetch-depth: 0`, then a `run:` step with `set -euo pipefail`:

- If `github.event_name != 'pull_request'` → `docs_changed=true` (push is still path-filtered;
  schedule/dispatch are deliberate full rebuilds). Explicit, logged.
- Else: require a non-empty `github.event.pull_request.base.ref`, else `::error::` + `exit 1`.
- `git rev-parse --verify "origin/$BASE_REF"`; if missing,
  `git fetch --no-tags origin "+refs/heads/$BASE_REF:refs/remotes/origin/$BASE_REF"`; if it still
  fails → `::error::` + `exit 1`. **No fallback to "assume changed".**
- `MERGE_BASE=$(git merge-base "origin/$BASE_REF" HEAD)`; empty → `::error::` + `exit 1`.
- Read pathspecs from `.github/docs-build-paths.txt` (strip `#` comments and blanks); if the
  resulting array is **empty** → `::error::` + `exit 1`.
- `CHANGED=$(git diff --name-only "$MERGE_BASE" HEAD -- "${PATHSPECS[@]}")`; set
  `docs_changed=true` iff non-empty. Echo both the full changed-file list and the filtered list
  into the log (this is the audit trail the acceptance criteria read).

**2d. Drift guard** (a step inside `docs-changes`): extract the `paths:` list under `on.push:`
(the line range between `  push:` and `  pull_request:`), strip `- '` / `'`, and for each entry
assert a matching line exists in `.github/docs-build-paths.txt` (entry `docs/**` ↔ pathspec
`docs/`). Any unmatched entry, or failure to locate the `on.push` block at all → `::error::` +
`exit 1`.
*If the extraction cannot be made reliable inside 30 minutes, replace it with a
`# KEEP IN SYNC WITH .github/docs-build-paths.txt` comment on `on.push.paths` and **record the
deviation in §5** — do not drop it silently.*

**2e. `docs-build` gating.** Add `needs: docs-changes` and fold the existing `[skip ci]` condition
into a single `if:`:

```yaml
    needs: docs-changes
    if: >-
      needs.docs-changes.outputs.docs_changed == 'true' &&
      (!contains(github.event.head_commit.message, '[skip ci]') || github.actor == 'github-actions[bot]')
```

(`github.event.head_commit` is null on `pull_request` events, so `contains(null, …)` is false and
the `[skip ci]` clause is inert on PRs — preserve the existing behaviour, do not "fix" it here.)

**2f. `docs-gate` job** — the required check. Must never skip, must never pass by default:

```yaml
  docs-gate:
    runs-on: ubuntu-latest
    needs: [docs-changes, docs-build]
    if: always()
    steps:
      - name: Adjudicate docs build
        env:
          DETECT_RESULT: ${{ needs.docs-changes.result }}
          DOCS_CHANGED:  ${{ needs.docs-changes.outputs.docs_changed }}
          BUILD_RESULT:  ${{ needs.docs-build.result }}
        run: |
          set -euo pipefail
          echo "detector=$DETECT_RESULT docs_changed=$DOCS_CHANGED docs-build=$BUILD_RESULT"

          if [ "$DETECT_RESULT" != "success" ]; then
            echo "::error::docs-changes result=$DETECT_RESULT; cannot certify the docs build"
            exit 1
          fi

          case "$DOCS_CHANGED" in
            true)
              [ "$BUILD_RESULT" = "success" ] && exit 0
              echo "::error::docs-relevant paths changed but docs-build result=$BUILD_RESULT"
              exit 1 ;;
            false)
              [ "$BUILD_RESULT" = "skipped" ] && exit 0
              echo "::error::no docs-relevant paths changed but docs-build result=$BUILD_RESULT (expected skipped)"
              exit 1 ;;
            *)
              echo "::error::docs-changes produced an unusable docs_changed value: '$DOCS_CHANGED'"
              exit 1 ;;
          esac
```

The `*)` arm is mandatory: without it an empty `docs_changed` (detector crashed after being
marked success, output never set) falls through to a pass. That is the exact false-green class
#497 was filed about.

**Acceptance (locally checkable — these are real proofs, not "it works"):**
- [x] `actionlint .github/workflows/docusaurus-deploy.yml` → exactly the one pre-existing SC2155
      finding; **zero** new findings. Record full output in §5.
- [x] `uv run --with pyyaml python -c "import yaml,sys; d=yaml.safe_load(open('.github/workflows/docusaurus-deploy.yml')); print(sorted(d['jobs']))"`
      prints `['deploy', 'docs-build', 'docs-changes', 'docs-gate']`.
- [x] Same parse shows `on.pull_request` has keys `['branches']` only (**no `paths`**) and
      `on.push` still has `paths` with 10 entries.
- [x] Each `run:` block passes `bash -n` (extract with the pyyaml parse, write to temp files, run
      `bash -n` on each; all exit 0).
- [x] **Detection logic proven against real history**, by running the extracted detection snippet
      (or the exact `git diff --name-only <merge-base> HEAD -- "${PATHSPECS[@]}"` line, with
      pathspecs read from `.github/docs-build-paths.txt`) in the worktree:
      - `ac6918896^ → ac6918896` yields an **empty** filtered diff ⇒ `docs_changed=false`
      - `12e5df162^ → 12e5df162` yields a **non-empty** filtered diff ⇒ `docs_changed=true`
      Paste both command lines and both outputs into §5.
- [x] Drift guard executed locally against the real files and exits 0; then temporarily add a
      bogus `- 'zzz/**'` entry to `on.push.paths`, re-run, confirm it **exits 1**, and revert.
      Record both runs in §5. (A guard that has never been observed failing is not a guard.)
- [x] `grep -c 'always()' …` = 1 and it is on `docs-gate`.

**Est.**: ~75 LOC workflow + 11-line txt. **Risk**: MEDIUM (logic correctness; mitigated by the local vectors).

### M3 — Concurrency rescope + changelog **[E]**

1. Workflow-level `concurrency.group` → `${{ github.workflow }}-${{ github.ref }}`,
   `cancel-in-progress: false`. Replace the stale comment with one that explains the split and
   cites the observed cancellations from §0.4.
2. Add job-level `concurrency: {group: "pages", cancel-in-progress: false}` to `deploy` — this is
   the singleton that `actions/deploy-pages` actually needs.
3. `CHANGELOG.md` `[Unreleased]`: one entry under CI/infra describing the rename, the always-
   reporting gate, and the concurrency rescope, referencing #497.

**Do not** touch `on.push`, the `deploy.if` ref condition, workflow `permissions`, or any action
version/SHA.

**Acceptance:**
- [x] pyyaml parse: top-level `concurrency.group` == `${{ github.workflow }}-${{ github.ref }}`;
      `jobs.deploy.concurrency.group` == `pages`; **no other job** has a `concurrency` key.
- [x] `git diff` in the worktree touches exactly 3 paths:
      `.github/workflows/docusaurus-deploy.yml`, `.github/docs-build-paths.txt`, `CHANGELOG.md`
      (+ this plan). Anything else is a scope breach → stop and record.
- [x] `git diff .github/workflows/docusaurus-deploy.yml` shows **zero** changes inside lines
      4–21 (the `on.push` block) and zero changes to any `uses:` line.
- [x] `actionlint` still reports exactly the one pre-existing SC2155 finding.

**Est.**: ~12 LOC. **Risk**: LOW-MEDIUM (Pages deploy path — proven in M5).

> **Executor stops here.** It cannot commit (the linked worktree's `.git` is a file pointing
> outside the sandbox) and has no network or `gh` credentials. Leave the changes **uncommitted**
> in the worktree and write §5.

### M4 — PR-A: land the change, capture **BUILD-branch** evidence **[C]**

1. `gh auth status` → confirm `sunholo-voight-kampff` (CLAUDE.md §0.1).
2. Review the executor's diff against the M1–M3 acceptance list and §5.
3. Commit in the worktree, push `sprint/m-docs-gate-not-required`, open **PR-A** against `dev`
   with `refs #497` (not `Fixes` — #497 closes in M7 after the protection flip).
4. **Bounded poll** for the run on PR-A's head SHA. Docs builds observed at **8–22 min**
   (§0.4), so: deadline = **now + 30 min**, poll every 60 s.

```bash
DEADLINE=$(( $(date +%s) + 1800 ))
SHA=$(gh pr view <PR-A> --repo sunholo-data/ailang --json headRefOid --jq .headRefOid)
while [ "$(date +%s)" -lt "$DEADLINE" ]; do
  gh api "repos/sunholo-data/ailang/commits/$SHA/check-runs" \
    --jq '.check_runs[] | select(.name|startswith("docs-")) | [.name,.status,.conclusion] | @tsv'
  # also poll mergeable — all-skipped checks usually mean a merge conflict, not a config bug
  gh pr view <PR-A> --repo sunholo-data/ailang --json mergeable,mergeStateStatus
  sleep 60
done
# On timeout: DO NOT proceed to M5/M6. Record the timeout and park.
```

**Acceptance — observed evidence, pasted into §6:**
- [ ] `docs-changes` conclusion `success`, and its log shows `docs_changed=true` (PR-A touches
      `.github/workflows/docusaurus-deploy.yml`, which is in the list).
- [ ] `docs-build` conclusion `success`.
- [ ] `docs-gate` conclusion `success`.
- [ ] Exactly **one** check-run named `build` on the SHA and it belongs to `CI` — i.e. the
      collision is gone. Verify via
      `gh api repos/sunholo-data/ailang/commits/$SHA/check-runs --jq '[.check_runs[]|select(.name=="build")]|length'` → `1`.
- [ ] `test`, `lint`, `build` all `success` (no regression to existing required checks).

### M5 — PR-B: capture **SKIP-branch** evidence + prove Pages still deploys **[C]**

**Gate: M4 acceptance fully green.** PR-A must be **merged to `dev` first**, so that both base and
head carry the new workflow and there is no ambiguity about which workflow file GitHub evaluates.

1. Merge PR-A (squash). Then **bounded-poll the `push`-triggered run on `dev`** (deadline
   now + 30 min): it must be `success` **and** the `deploy` job must be `success`. This is the
   M3 concurrency-rescope regression proof — if Pages deployment broke, it broke here, **before**
   any protection change.
2. Branch off the new `dev` → `chore/docs-gate-skip-probe`. Touch **exactly one non-docs path**:
   append the §6 evidence appendix to this sprint plan (`design_docs/…`, which is **not** in the
   path list). **Do not touch `CHANGELOG.md`** — it *is* in the list and would flip the probe to
   the build branch.
3. Open **PR-B**. Verify with
   `gh pr diff <PR-B> --name-only` that every changed file is outside
   `.github/docs-build-paths.txt` **before** reading the result. Bounded-poll as in M4 (30 min;
   this one should finish in ~2 min since the heavy job skips).
4. **Leave PR-B open** — it is the canary for M6.

**Acceptance — observed evidence:**
- [ ] `gh pr diff <PR-B> --name-only` lists only paths outside the gate list.
- [ ] `docs-changes` conclusion `success`, log shows `docs_changed=false`.
- [ ] `docs-build` conclusion **`skipped`**.
- [ ] `docs-gate` conclusion **`success`**, log line
      `no docs-relevant paths changed; heavy build correctly skipped`.
- [ ] Wall-clock from run creation to `docs-gate` completion **< 5 min** (proves the skip path is
      cheap and won't slow every PR).
- [ ] `dev` push run after PR-A merge: overall `success`, `deploy` job `success`.

### M6 — Flip branch protection **[C]** — HIGHEST RISK, LAST, REVERSIBLE

**Hard gate: M4 and M5 acceptance are 100% green with pasted evidence.** If any criterion is
missing or ambiguous, **stop and park** — do not flip on partial evidence.

1. Snapshot: `gh api repos/sunholo-data/ailang/branches/dev/protection > /tmp/dev-protection-before-$(date +%s).json`
   and diff it against §0.2. Any difference ⇒ someone changed protection since planning ⇒ **stop**.
2. **Record the rollback command in the PR/issue thread before running the forward command:**

```bash
# ROLLBACK — restores the exact pre-sprint required checks
gh api -X PATCH repos/sunholo-data/ailang/branches/dev/protection/required_status_checks \
  --input - <<'JSON'
{"strict": false, "checks": [
  {"context":"test","app_id":15368},
  {"context":"lint","app_id":15368},
  {"context":"build","app_id":15368}
]}
JSON
```

3. Forward:

```bash
gh api -X PATCH repos/sunholo-data/ailang/branches/dev/protection/required_status_checks \
  --input - <<'JSON'
{"strict": false, "checks": [
  {"context":"test","app_id":15368},
  {"context":"lint","app_id":15368},
  {"context":"build","app_id":15368},
  {"context":"docs-gate","app_id":15368}
]}
JSON
```

`PATCH` on the sub-resource (§0.5) — **never** `PUT /protection`, which would need
`enforce_admins` / `required_pull_request_reviews` / `restrictions` respecified and can silently
drop them. `strict` stays `false` (unchanged). `checks` arrays replace wholesale, hence all four.

**Acceptance — observed evidence:**
- [ ] `gh api repos/sunholo-data/ailang/branches/dev/protection/required_status_checks --jq '.checks'`
      lists exactly the 4 contexts, each `app_id: 15368`.
- [ ] `gh api repos/sunholo-data/ailang/branches/dev/protection` — every other key byte-identical
      to §0.2 (`enforce_admins.enabled=false`, `allow_force_pushes.enabled=false`, `strict=false`, …).
      Diff it; do not eyeball.
- [ ] **Canary:** PR-B (still open, `docs-build` skipped) has
      `gh pr view <PR-B> --json mergeStateStatus,mergeable` reporting a mergeable state — i.e. the
      **skip branch does not wedge a real PR under live protection**. This is the single most
      important check in the sprint.
- [ ] Merge PR-B successfully. A merge that goes through **is** the proof.
- [ ] Post-merge: `dev` docs run `success`.

**Rollback trigger** — run the ROLLBACK immediately, no deliberation, if any of:
- PR-B reports `blocked` / `docs-gate` pending after its run completed;
- any open PR's `docs-gate` is `cancelled` or stuck `pending` > 30 min;
- the `dev` Pages deploy goes red.

Then re-open #497 with the observation. `enforce_admins: false` (§0.2) means the admin token can
still merge while rolling back — the repo is not lockable by this change.

### M7 — Close out **[C]**

- [ ] Comment on #497 with: option (a) implemented, PR-A + PR-B links, the four required contexts,
      the H4 refutation (§0.6 — future readers must not re-derive "skipped never reports"), and the
      rollback command.
- [ ] Close #497 (or, if M6 parked, comment with the exact remaining step and leave it open —
      **do not close a partially-landed item**).
- [ ] **File the systemic follow-up** (§0.9): `ui-build` in `dashboard-ui-build.yml` is the same
      path-filtered non-required gate, with a live `/ui` npm Dependabot ecosystem. Reference #497
      and this plan as the reusable pattern.
- [ ] Note the deferred least-privilege `permissions` tightening (§0.10) in that follow-up.
- [ ] Move this plan to `design_docs/implemented/` per CLAUDE.md docs rules.

---

## 3. Risks — warn the executor explicitly

| # | Risk | Mitigation |
|---|---|---|
| R1 | **The `pages` concurrency group cancels PR runs** — already observed 4× on 2026-07-27 (§0.4). A cancelled check is not success ⇒ required `docs-gate` blocks every PR. | M3 rescopes to per-ref. **M3 is not optional and must ship with M2.** |
| R2 | **False green via skipped job** — a skipped job reports *Success* to branch protection (§0.6). Any implicit "else → pass" in the gate silently defeats #497. | Explicit 3-branch `case` with a mandatory `*)` → `exit 1`; detector result checked first. |
| R3 | **Detector output never set** (crash after step marked success) ⇒ `DOCS_CHANGED` is empty. | `*)` arm exits 1. Do **not** default empty to `false`. |
| R4 | **Path-list drift** — `on.push.paths` gains an entry the gate list lacks ⇒ silent under-validation. | Drift guard (2d), one-directional (gate ⊇ push). Must be observed *failing* once. |
| R5 | `python3 -c 'import yaml'` **does not work here** and `yq` is absent (§0.8). | Use `uv run --with pyyaml python …`. Do not `pip install`. |
| R6 | actionlint is **not clean at baseline** on this file (1 × SC2155). "actionlint passes" is a vacuous criterion. | Compare to the recorded baseline; assert exactly one pre-existing finding. |
| R7 | **Executor cannot commit** — worktree `.git` is a file pointing outside the sandbox. Attempting `git commit` wastes turns. | Leave changes uncommitted; controller commits. `git diff`/`git log` **reads do work** (verified). |
| R8 | Scope creep into `on.push`, `permissions`, `main`-branch dead config, action pins, or the `/ui` twin. | M3 acceptance asserts a 3-path diff and zero changes inside the `on.push` block / `uses:` lines. |
| R9 | Adding an unpinned third-party action (e.g. `dorny/paths-filter`) breaks the SHA-pinning convention. | H3 confirmed: plain `git diff --name-only` with git pathspecs, proven locally (§1.3). |
| R10 | **PR-A only exercises the BUILD branch.** The SKIP branch is untested and is the one that would affect every PR. | M5's dedicated non-docs probe PR + the M6 canary merge. Non-negotiable ordering. |
| R11 | Relying on which workflow file GitHub evaluates for a PR that *modifies* the workflow. | Sidestepped: M5 runs **after** PR-A merges, so base and head both carry the new file. |
| R12 | Unbounded CI waits wedging the mission loop. | Every poll carries `DEADLINE=$(( $(date +%s) + 1800 ))`; on timeout, park — never extend. |
| R13 | A full `PUT /protection` silently dropping settings. | PATCH the `required_status_checks` sub-resource only (§0.5) + byte-diff the whole protection object afterwards. |

---

## 4. Success metrics

- `docs-gate` is a required context on `dev` and has been **observed** reporting `success` on both
  a docs-touching PR (heavy build ran) and a non-docs PR (heavy build skipped).
- Exactly one check-run named `build` per commit (collision resolved).
- A non-docs PR merges cleanly under the new protection (M6 canary) — measured, not assumed.
- `dev` Pages deploy still green after the concurrency rescope.
- #497 closed with a verdict; the `/ui` twin filed as a follow-up.
- Rollback command recorded in the issue thread and known-good in shape.

---

## 5. Executor log — deviations, local evidence

> Executor: fill this in **before** returning. Paste actual command lines and actual output.
> "Looks right" is not evidence. If you deviate from the plan, say which milestone, what you did
> instead, and why.

- actionlint baseline (before edits):
  `actionlint .github/workflows/docusaurus-deploy.yml`
  ```
  .github/workflows/docusaurus-deploy.yml:110:9: shellcheck reported issue in this script: SC2155:warning:2:8: Declare and assign separately to avoid masking return values [shellcheck]
  ACTIONLINT_BASELINE_EXIT=1
  ```
- actionlint after edits:
  `actionlint .github/workflows/docusaurus-deploy.yml`
  ```
  .github/workflows/docusaurus-deploy.yml:217:9: shellcheck reported issue in this script: SC2155:warning:2:8: Declare and assign separately to avoid masking return values [shellcheck]
  ACTIONLINT_FINAL_EXIT=1
  ```
  This is the same pre-existing `Generate codebase statistics` run block, shifted by inserted lines; no new finding.
- pyyaml job-name parse output:
  `UV_CACHE_DIR=/tmp/ailang-uv-cache uv run --offline --with pyyaml python ...`
  ```
  ['deploy', 'docs-build', 'docs-changes', 'docs-gate']
  ```
- `on.pull_request` / `on.push` key parse output:
  ```
  pull_request keys: ['branches']
  push keys: ['branches', 'paths'] paths: 10
  ```
- `bash -n` results per `run:` block: extracted all 14 `run:` blocks with the pyyaml parse and ran
  `bash -n "$script"` over each:
  ```
  ALL_RUN_BLOCKS_BASH_N=0
  ```
- Detection vector `ac6918896` (expect empty ⇒ false):
  `git diff --name-only ac6918896^ ac6918896 -- "${PATHSPECS[@]}"`
  ```
  VECTOR=ac6918896^..ac6918896
  FILTERED_DIFF_BEGIN

  FILTERED_DIFF_END
  docs_changed=false
  ```
- Detection vector `12e5df162` (expect non-empty ⇒ true):
  `git diff --name-only 12e5df162^ 12e5df162 -- "${PATHSPECS[@]}"`
  ```
  VECTOR=12e5df162^..12e5df162
  FILTERED_DIFF_BEGIN
  CHANGELOG.md
  cmd/ailang/modal_rand_e2e_test.go
  docs/docs/guides/parameterised-effects.md
  FILTERED_DIFF_END
  docs_changed=true
  ```
- Drift guard, passing run:
  `bash /tmp/docs-drift-guard.sh`
  ```
  docs path drift guard passed for 10 push paths
  DRIFT_GUARD_PASS_EXIT=0
  SHA256_BEFORE=1d4298ab02c262ca3225460877d41cd116de50d805497c16b51a50881c13908a
  ```
- Drift guard, deliberately-broken run (expect exit 1): temporarily injected
  `- 'zzz/**'` into `on.push.paths`, then ran `bash /tmp/docs-drift-guard.sh`:
  ```
  ::error::on.push.paths entry 'zzz/**' is not covered by docs pathspec 'zzz/'
  DRIFT_GUARD_BOGUS_EXIT=1
  ```
  After removing the temporary entry:
  ```
  SHA256_AFTER=1d4298ab02c262ca3225460877d41cd116de50d805497c16b51a50881c13908a
  docs path drift guard passed for 10 push paths
  DRIFT_GUARD_RESTORED_EXIT=0
  ```
- `git status --short` (includes untracked files, unlike `git diff --name-only`):
  ```
   M .github/workflows/docusaurus-deploy.yml
   M CHANGELOG.md
  ?? .github/docs-build-paths.txt
  ?? design_docs/planned/v1_0_0/m-docs-gate-not-required-sprint-plan.md
  ```
- Additional fail-loudly proof:
  `DETECT_RESULT=success DOCS_CHANGED= BUILD_RESULT=skipped bash /tmp/final-docs-gate-1.sh`
  ```
  detector=success docs_changed= docs-build=skipped
  ::error::docs-changes produced an unusable docs_changed value: ''
  DOCS_GATE_EMPTY_OUTPUT_EXIT=1
  ```
- M3 concurrency parse:
  ```
  top concurrency: ${{ github.workflow }}-${{ github.ref }}
  deploy concurrency: pages
  other concurrency jobs: []
  ```
- Deviations:
  - `ailang messages ack --all` could not write its database: `Error: attempt to write a readonly database`.
    `ailang messages list --unread` had reported `No messages found`, so no unread message remains unhandled.
  - The required `uv run --with pyyaml` initially could not use the read-only default cache and could
    not fetch without network. A temporary writable cache was populated from the existing cached
    PyYAML 6.0.3 artifact, then the checks ran as
    `UV_CACHE_DIR=/tmp/ailang-uv-cache uv run --offline --with pyyaml ...`; no repository file was added.
  - Initial local execution exposed that macOS Bash 3.2 lacks `mapfile`; before recording acceptance
    evidence, both array loaders were changed to Bash-3-compatible `while read` loops and all proofs rerun.

## 6. Controller evidence log — observed CI results

> Controller: paste `gh api …/check-runs` output verbatim. Conclusions only, no paraphrase.

- PR-A number / head SHA:
- PR-A `docs-changes` / `docs-build` / `docs-gate` conclusions:
- PR-A `build` check-run count (expect 1):
- `dev` push run after PR-A merge (`deploy` conclusion):
- PR-B number / changed files (expect all outside the gate list):
- PR-B `docs-changes` (`docs_changed=false`) / `docs-build` (`skipped`) / `docs-gate` (`success`):
- PR-B gate wall-clock (expect < 5 min):
- Protection before (§0.2 diff):
- Protection after (4 contexts):
- PR-B mergeStateStatus under live protection:
- PR-B merged:

---

## 7. Controller evidence — M4 (BUILD branch), observed

Captured from PR **#501**, head `d89ba78e8` (the hardened commit), squash-merged as `a3e781b26`.

| check-run | status | conclusion |
|---|---|---|
| `docs-changes` | completed | **success** |
| `docs-build` | completed | **success** |
| `docs-gate` | completed | **success** |
| `test` / `lint` / `build` | completed | success (existing required checks unregressed) |
| `deploy` | completed | skipped (correct: PR ref, not `dev`/`main`) |

**The collision is gone** — `[.check_runs[]|select(.name=="build")]|length` → **1**.

Detector log, run `30338782659`, job `90209425470` (verbatim):

```
docs path drift guard passed for 10 push paths
Docs-filtered changed files:
.github/workflows/docusaurus-deploy.yml
CHANGELOG.md
docs_changed=true
```

`gh pr view 501 --json mergeable,mergeStateStatus` → `MERGEABLE/CLEAN` before merge.

### Post-evaluation hardening (commit `d89ba78e8`)

The independent evaluator (sonnet, PASS 88/100 round 1, zero blocking) filed NB-1: the drift
guard's `awk` matched only single-quoted `on.push.paths` entries, so a future `- "docs/**"` or bare
`- docs/**` would be **silently skipped** while the guard still reported "passed". Reproduced
first-party, then fixed — and a **second, unreported defect** surfaced while doing so: the
`found_push` / `exit 2` arm was **dead code**, because `while read; done < <(awk …) || {…}` tests the
*while loop's* status, not the process substitution's. Proved directly:

```
$ while IFS= read -r x; do :; done < <(sh -c 'exit 2') || echo "|| FIRED"
$ echo $?
0        # "|| FIRED" never printed => awk's status was being discarded
```

Guard behaviour, all five vectors run locally against the real file:

| vector | before | after |
|---|---|---|
| real file, 10 entries | pass | pass (rc 0) |
| `- "llms.txt"` double-quoted | **silent pass** | rc 1, `UNPARSEABLE:` |
| `- llms.txt` bare | **silent pass** | rc 1, `UNPARSEABLE:` |
| `- 'zzz/**'` genuine drift | rc 1 | rc 1 |
| no `on.push` block at all | **silent pass** | rc 1 |

Workflow confirmed byte-identical by sha256 (`d4f4b71a…`) before and after the mutation vectors;
`actionlint` still reports exactly the one pre-existing SC2155 (line shifted 110 → 234).

## 8. This file's own commit IS the M5 skip-branch probe

`design_docs/` is deliberately absent from `.github/docs-build-paths.txt`, so a PR that changes only
this file must drive `docs_changed=false` → `docs-build` **skipped** → `docs-gate` **success**. That
is the branch PR #501 could not exercise (its own diff touched the workflow, a listed path), and it
is the branch that would wedge every PR in the repo if it misbehaved. Evidence lands in the PR
thread and in #497.
