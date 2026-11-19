# Sprint plan: sprint-executor cloud push per milestone

**Sprint:** M-FLEET-CLOUD-PUSH-PER-MILESTONE  
**Fleet iteration:** 49 · planner artifact · 2026-10-20  
**Base:** sunholo-data/ailang `origin/dev` at `4cc0f6a3e` (detached planning checkout)  
**Ticket:** `crash:sprint-executor:sigbus-ram-tmp-full-make-test` (attended report, 2026-10-08, blocking=item)  
**Duration:** one implementation milestone, one executor run of approximately 60–90 minutes; reserve one working day including CI/review.  
**Estimate:** 1,190 changed/added LOC including byte-identical mirrors; approximately 600 independently authored LOC.  
**Risk:** medium; an incorrect branch gate could publish a live branch.

## Goal and scope

Preserve completed cloud sprint milestones across SIGBUS, timeout and job retry by publishing each milestone commit and resuming the task branch on retry. Prevent resource-heavy attempts to provision missing test toolchains. Two prod Cloud Run sprint-executor attempts on the same task died with SIGBUS while building a C toolchain/sysroot under RAM-backed `/tmp` with `CGO_ENABLED=1` and running full `make test` (2 CPU, 8 GiB, maxRetries=1). Nothing had been pushed, so neither attempt left durable work.

This sprint implements the remaining parts of Ask 1: no compiler/system-package installation, and a safe skill-side push/resume protocol. Ask 2 (scratch volume/memory), Ask 3 (late coordinator failure reporting), and changes to `cmd/ailang` are outside fleet scope. The wrapper defect remains; this protocol neutralises it for participating executor runs.

Already shipped: #1770 (`dbe4b26a3`) changed executor gates to `make test-changed`; #1688 (`face88a3f`) added `resources/unattended_runs.md` and commit-after-each-milestone guidance. Preserve both. Do not reimplement focused testing or broaden local verification to full `make test`; CI owns the full suite.

The controller supplied and settled the design below. Planning writes only this plan and `.ailang/state/sprints/sprint_M-FLEET-CLOUD-PUSH-PER-MILESTONE.json`; the controller commits them. Implementation is a later stage, with no dispatch or git writes from this planner.

## Evidence and current implementation

### Controller measurements M1–M5

These are first-party **controller measurements**, made in scratch repos with git 2.54 and `/bin/bash` 3.2. They are the design's empirical basis, not experiments rerun by this planner.

| ID | Measurement | Consequence for the skill |
| --- | --- | --- |
| M1 | `cmd/ailang/coordinator_cloud_task.go` clones with `git clone --branch <base> --depth 1` (line 151), implicitly single-branch with only `+refs/heads/<base>:refs/remotes/origin/<base>`. It creates `coordinator/<taskID>` (lines 196–199), runs the agent and commits leftovers. Step 5b uses `git log origin/<branch>..HEAD`; on error, `ls-remote --exit-code origin <branch>` finding an existing branch leaves `newBranch=false` and empty log output. The wrapper then opens the PR without pushing or asserting the remote matches HEAD. | A successful agent push alone does not make the wrapper's tracking-ref comparison reliable. |
| M2 | In that shallow clone, `git push origin HEAD:refs/heads/coordinator/task-x` succeeds but creates no `refs/remotes/origin/coordinator/task-x`. After a second local commit, log exits 128 while ls-remote exits 0; the wrapper incorrectly concludes there are no unpushed commits. | A naive push-per-milestone instruction can strand later work. Tests need both the broken control and the protected path. |
| M3 | Adding `git config --add remote.origin.fetch "+refs/heads/${B}:refs/remotes/origin/${B}"` **before** the first push makes that push create the tracking ref. A later commit is then listed by `git log origin/${B}..HEAD` with rc 0; the wrapper pushes and asserts. | Ensure this exact branch refspec idempotently in both helper subcommands before any push/fetch. |
| M4 | A retry re-clones the base and recreates the same task branch; a push without resuming is rejected non-fast-forward. After adding the refspec, `git ls-remote --exit-code origin refs/heads/${B}` rc 0 → `git fetch origin` rc 0 → `git reset --keep origin/${B}` rc 0 restores prior commits while keeping untracked wrapper files. A new commit then pushes fast-forward. | Run `start` once before choosing the first unfinished milestone; resume instead of overwriting remote history. |
| M5 | The child carries `AILANG_TASK_ID` (`internal/executor/environment.go:79`). The default `AILANG_CHILD_GIT_CREDENTIALS=repo` grants credentials for the task repo (`cmd/ailang/coordinator_cloud_executor.go:410–438`); shadow children have no push credential. `AILANG_PUSH_BRANCH` direct-push mode works on the live branch, such as main/dev. | Exact task-branch gating is essential. Permission/auth failures must be recorded and non-fatal; direct-push and shadow behavior do not justify widening the gate. |

Read-only inspection at this base confirms the clone/branch setup, Step 5b fallback, `branchNeedsPush(newBranch, logOutput)` returning `newBranch || trim(logOutput) != ""`, shadow credential exclusion, and the resource's current sentence “the coordinator pushes your branch when you finish.” The existing `SKILL.md:97` already links the resource, so no SKILL.md edit is needed.

### Reuse and sizing

**Registry reuse decision, M1_CLOUD_BRANCH_SYNC: `none`.** This is a Bash/Git skill workflow, with no package-like AILANG capability or `.ail` module. Registry searches and candidate `pkg info`/`pkg docs` calls are not applicable; no package is added or modified. Reuse the existing resource link, `$(LAUNCHD_SUITE)` integration and mirror-check workflow. Consult the portable bounded-command patterns in `tools/launchd/lib/pin-root.sh` and `lane-probe.sh`; avoid sourcing the whole mission driver or adding a timeout/toolchain dependency.

Recent history provides sizing comparisons, not a measured LOC/day throughput: #1770 changed 57 lines added / 33 removed across 18 files, mostly mirrored instructions; fleet #1843 (`51f9ecb29`) added 189 implementation and 414 test lines to its two shell files. This sprint budgets 150 helper lines and 400 hermetic test lines plus documentation/integration. Mirrors double most physical LOC without doubling authored behavior. The planner velocity script was attempted, but exited 141 in its `git log | head` pipeline under pipefail; its partial output does not establish a reliable velocity. The one-day reservation is an estimate with review buffer, not a historical throughput claim.

## Fixed design decisions D1–D8

1. **D1 — interface and portability.** Create `.claude/skills/sprint-executor/scripts/cloud_branch_sync.sh`, invoked with `/bin/bash`, accepting exactly one subcommand: `start` or `push`. Use Bash 3.2 and `set -u`; no associative arrays, `${v,,}`, `mapfile`, GNU-timeout prerequisite or tool installs. All Git operations must be bounded or naturally finite. Local branch/ref/config reads are finite; network Git operations require a hard wall-clock deadline, disabled interactive prompting, and bounded termination/reaping. Use existing portable patterns without importing unrelated mission behavior. Keep failures explicit rather than relying on `set -e` to exit prematurely.
2. **D2 — exact branch safety gate.** Act only when `AILANG_TASK_ID` is nonempty and `git branch --show-current` equals exactly `coordinator/$AILANG_TASK_ID`. Evaluate before any Git mutation or network operation. Attended sessions, mission worktrees, wrong task branches, detached HEAD, unset/empty task IDs and direct-push mode on live branches produce `cloud-branch-sync: skipped:<reason>`, rc 0. Read-only discovery may identify the subject; skipped paths must not change Git config, refs, index or working files, fetch or push. Never push any other branch; never force-push.
3. **D3 — tracking ref prerequisite.** Both commands first check `remote.origin.fetch` for the exact line `+refs/heads/${B}:refs/remotes/origin/${B}` and add it once if absent. Preserve existing entries; repeated calls add no duplicate. Use repository-local config only. The leading `+` permits fetch tracking updates; it is not permission to force-push. Config failure produces a failure verdict before any network mutation.
4. **D4 — start/resume.** After the gate and refspec check, query `git ls-remote --exit-code origin refs/heads/${B}`. rc 2 means absent: print `cloud-branch-sync: fresh` and leave HEAD, index and working tree unchanged. rc 0 means present: bounded `git fetch origin`, then `git reset --keep origin/${B}` and print `cloud-branch-sync: resumed <sha>`. Any other query result is `failed:<reason>`, not `fresh`. If fetch or reset refuses, report it without altering HEAD/index/working files; retain dirty and untracked work. Fetch/config may update Git metadata as specified. Do not substitute reset-hard, clean, checkout or stash. On successful resume, read the restored sprint JSON and continue at the first unfinished milestone.
5. **D5 — publish and verify.** After every milestone commit, `push` ensures the refspec, executes bounded `git push origin HEAD:refs/heads/${B}` without force, then uses exact-ref ls-remote to assert remote SHA equals HEAD. Print `cloud-branch-sync: pushed <sha>` only after equality; failure/rejection, timeout, missing ref or mismatch prints `cloud-branch-sync: failed:<reason>`. Do not amend, reset or discard local work on push failure. A push failure is non-fatal to the sprint: record it in sprint JSON notes and continue; the coordinator still attempts its final push. A non-fast-forward refusal never authorizes a force-push.
6. **D6 — verdict contract.** rc 0 for pushed/resumed/fresh/skipped **and** failed outcomes; rc 2 only for missing, unknown or extra command arguments. Every verdict names `branch=<subject>` and `sha=<subject-head>`; use explicit `<unknown>` / `<detached>` markers if unavailable. Success examples: `cloud-branch-sync: resumed <sha> branch=<B> sha=<sha>` and `cloud-branch-sync: pushed <sha> branch=<B> sha=<sha>`. Fresh/skipped/failed lines keep their specified prefix and append the same subject fields. Verification failures should also name the observed remote SHA when available. A stale line must identify its own branch/commit. Tests assert verdict content and exit status separately; rc 0 is not proof of publication.
7. **D7 — resource instructions and mirrors.** Update `resources/unattended_runs.md` with the two sections specified below, replacing the claim that publishing waits until the end. Leave SKILL.md unchanged because it already links this resource and has an enforced size baseline. Mirror the helper, its test and the resource byte-for-byte into `.agents/skills/sprint-executor/`; confirm `tools/launchd/test_agents_skills_sync.sh`. Do not rewrite unrelated skill files with a broad sync operation.
8. **D8 — hermetic test and Make integration.** Create `.claude/skills/sprint-executor/scripts/test_cloud_branch_sync.sh`, Bash 3.2, using only a temp fixture root, synthetic HOME and a local bare remote via `file://`. Reproduce M1 with `git clone --depth 1 --branch <base>` (not a normal local-path clone that ignores depth), then create the exact task branch. The test never changes the caller's HOME/config/repo. Wire it next to `.claude/skills/sprint-evaluator/scripts/test_ci_gate.sh` in `make/test.mk` using `@$(LAUNCHD_SUITE) .claude/skills/sprint-executor/scripts/test_cloud_branch_sync.sh`; add explicit `/bin/bash -n` recipe lines for the helper and test. Include the Step 5b simulation and broken control below.

### Required resource content

Add **“Cloud tasks: push every milestone”** to the existing unattended resource:

- Run `/bin/bash .claude/skills/sprint-executor/scripts/cloud_branch_sync.sh start` once at cloud session start, before selecting/executing the first milestone. Inspect its verdict, then read sprint JSON; after resume, continue from the first unfinished milestone. Do not rerun `start` after creating new local milestone commits.
- Keep commit-after-each-milestone behavior. Include the updated sprint JSON/checkpoint in each milestone commit, then run `/bin/bash .claude/skills/sprint-executor/scripts/cloud_branch_sync.sh push` immediately after that commit. There is no new end-of-sprint-only publication rule.
- Explain durability using the 2026-10-08 ticket identifier: SIGBUS/timeout before the wrapper's final push loses all unpublished work. Explain M2 in one sentence: shallow single-branch clones lack the task tracking ref, so the wrapper can miss later commits unless the exact fetch refspec is installed before the first agent push.
- Explain skipped scope, credential/shadow failures, and rc 0 versus verdict. Record `failed:*` with branch/SHA/reason in sprint JSON notes and continue independent work; never force, install credentials or expand the branch gate. For start failures, record the failure and preserve the tree; do not improvise destructive recovery. The wrapper remains the final publication attempt, not a guarantee that a refused push is durable.

Add **“Never install toolchains”** as a bullet: never install or build compilers, sysroots, C/CGO toolchains or system packages, including apt, brew, sudo, building cc under `/tmp`, and downloading Go or z3. If focused verification needs missing cc for cgo/sqlite, z3, make or another absent tool, run the focused gates that do work, record the exact skipped test and missing tool in sprint JSON notes, and leave that verification to CI. Preserve `make test-changed` guidance; never run full `make test` locally. Do not disable cgo or otherwise change test semantics just to claim a pass.

## Implementation files

| File | Change | Estimated LOC |
| --- | --- | ---: |
| `.claude/skills/sprint-executor/scripts/cloud_branch_sync.sh` | New gated start/push helper | 150 |
| `.claude/skills/sprint-executor/scripts/test_cloud_branch_sync.sh` | New hermetic test, 22 named arms | 400 |
| `.claude/skills/sprint-executor/resources/unattended_runs.md` | Push/resume and no-toolchain instructions | 40 |
| `.agents/skills/sprint-executor/scripts/cloud_branch_sync.sh` | Byte-identical helper mirror | 150 |
| `.agents/skills/sprint-executor/scripts/test_cloud_branch_sync.sh` | Byte-identical test mirror | 400 |
| `.agents/skills/sprint-executor/resources/unattended_runs.md` | Byte-identical resource mirror | 40 |
| `make/test.mk` | LAUNCHD_SUITE test + syntax recipes beside ci_gate | 4 |
| `changelogs/unreleased/2026-10-20-sprint-executor-cloud-push-per-milestone.md` | Fragment opening `### Changed — Sprint executor cloud milestone durability` | 6 |
| **Total** | **One cohesive milestone** | **1,190** |

SKILL.md and coordinator Go files are unchanged. No `.ail` code or example manifest changes; the runnable examples are the two resource command invocations and the test's local-remote retry scenario. No new package dependency, CLI flag, credential policy or cloud configuration.

## M1_CLOUD_BRANCH_SYNC — durable milestones without installing toolchains

**Dependencies:** none beyond the approved controller design and existing Bash/Git. **Estimated LOC:** 1,190 including mirrors. **Duration:** one 60–90-minute executor run, within a one-day review window.

One milestone keeps the branch guard, tracking ref, retry resume, publishing, regression proof and discoverable instructions together. Splitting would expose a push-only intermediate state with M2/M4 data-loss hazards. Tests and the helper are small shell changes; no full suite or cloud deployment is required.

**Day 1 sequence:**

1. First 20 minutes: build isolated shallow-clone fixtures and the wrapper simulation/control; add the skip, tracking and retry arms before the helper. Capture meaningful red failures for the behavior being added.
2. Next 35 minutes: implement D1–D6; complete refusal, dirty-resume, bounded-call and verification arms. Run the focused shell test and syntax checks. A failed helper verdict is an assertion target, not a test-run failure by itself.
3. Next 15 minutes: update the resource, mirror the three files, wire Make recipes, and add the changelog fragment.
4. Final 20 minutes: run the listed focused gates, confirm arm-to-mutation evidence and mirror equality, and record precise results/limitations in sprint JSON. Commit the completed milestone; use the gated push helper when executing on its eligible cloud task branch. A separate controller handles this planner's artifacts.

**Acceptance criteria:**

- [ ] Bash 3.2 helper implements both commands, all D1–D6 outcomes and exit codes, exact task-branch gating, and bounded/noninteractive Git operations; no force push or toolchain installation.
- [ ] Both commands ensure exactly one task fetch refspec before network work, preserving the base refspec; push creates both remote and remote-tracking refs equal to HEAD.
- [ ] Step 5b wrapper simulation sees a later commit after helper push; its no-refspec control reproduces M2 with log rc 128, ls-remote rc 0 and `branchNeedsPush=false` despite that unpublished commit.
- [ ] Retry from a new shallow clone restores prior remote milestone commits, keeps an untracked wrapper file, and publishes the next commit fast-forward. Failed fetch/reset leaves local work intact.
- [ ] All 22 arms below pass, including real remote-ref observations on skipped branches and non-force refusal on a diverged remote. Each arm is paired with its specified mutation; record whether the mutation is killed, not only aggregate assertion counts.
- [ ] Resource includes the exact start/commit/push sequence, resume-from-JSON instructions, ticket and one-sentence M2 explanation, no-toolchain bullet, and non-fatal failure/skip notes. Already-shipped focused-test and commit behavior remains.
- [ ] All three `.agents` files equal their `.claude` sources byte-for-byte; SKILL.md stays unchanged; Make invokes the test through LAUNCHD_SUITE and syntax-checks both new scripts; changelog fragment exists with the specified heading.
- [ ] Required local gates and the added Make integration gate have recorded results. No socket-dependent sandbox result is presented as a pass/fail; run those gates on the host/CI for an informative verdict. No full `make test` or provisioning of absent tools.

## Regression arms and mutations (22 arms)

All arms assert the subject-bearing verdict and exit code. For skip/refusal/fresh/conflict cases, compare HEAD, index/working-file contents and status before/after as applicable. Snapshot **all remote refs** (`git --git-dir=<bare> for-each-ref`) and Git config where “no push/no mutation” is required. Ensure skip fixtures contain a new local commit, so an accidental push changes an observable remote ref. The live-branch arm must observe the remote; a correct-looking skipped line alone cannot prove safety.

| Arm | Fixture and required observation | Mutation it must kill |
| --- | --- | --- |
| A01 Live/direct-push branch | On live `dev` (also exercise `main`), with nonempty task ID and matching `AILANG_PUSH_BRANCH`, run both commands with a local-only commit. rc 0 skipped; all remote refs, fetch config and local state unchanged; trace contains no fetch/push. | Remove D2 or allow the live branch because task ID/direct-push env is present; also print skipped while actually pushing. |
| A02 Mission branch | On a non-coordinator mission branch with task ID set and an unpublished commit, both commands skip and leave remote refs/config/local state unchanged. | Broaden D2 to any branch when a task ID exists. |
| A03 Wrong task branch | Current `coordinator/task-other`, env task ID `task-x`; both commands skip with remote refs/config/local state unchanged. | Match only the `coordinator/` prefix instead of exact branch equality. |
| A04 No task ID | Exercise unset and explicitly empty `AILANG_TASK_ID` on a coordinator branch with unpublished work; both commands skip without writes/network and all remote refs remain unchanged. | Infer task identity from branch or use an unsafe default under `set -u`. |
| A05 Detached HEAD | Nonempty task ID, detached commit containing unpublished work; both commands skip, remote refs/config unchanged and HEAD remains detached. | Infer a push destination from task ID without checking actual current branch. |
| A06 Refspec idempotence | Fresh eligible clones: run `start` twice; independently run `push` twice without start. Each path leaves exactly one exact task refspec plus unchanged base entry. | Add on every call, replace base entry, omit the prerequisite in either subcommand, or add it only after pushing. |
| A07 Push publishes and tracks | M1-style clone, eligible branch and milestone commit. Remote exact ref and `refs/remotes/origin/B` both equal HEAD; success line names that B/SHA. | Push the wrong source/destination, omit refspec setup or claim success without updating tracking state. |
| A08 WRAPPER-SIMULATION protected | Helper push milestone 1, make milestone 2 locally, then run the literal Step 5b decision below. log rc 0 contains milestone 2, `newBranch=false`, `unpushed=true`; remote still points to milestone 1 before wrapper publication. | Remove/prevent M3 refspec installation, leaving the wrapper unable to see later work. |
| A09 WRAPPER-SIMULATION control | Same fresh shallow fixture, **without helper/refspec**, naive push milestone 1 then local milestone 2; same simulation gives log rc 128, ls-remote rc 0, `newBranch=false`, `unpushed=false`; remote remains milestone 1 and tracking ref absent. This expected loss is a passing negative control. | Make the simulator always return true, treat every log error as newBranch, manually seed tracking refs, or use a full/all-branch clone that hides M2. |
| A10 Fresh start | Remote task branch absent (real ls-remote rc 2); start prints fresh, rc 0, no HEAD/index/work-file/untracked-file or remote-ref change; only the specified refspec metadata is added. | Treat absent as resume/failure, reset to base, or create/push a task branch in start. |
| A11 Crash/retry resume | Attempt 1 pushes a milestone; discard only that disposable fixture, then fresh `--depth 1 --branch` clone and same task branch from base. Add untracked wrapper file. Start restores exact remote SHA and milestone file/JSON contents; wrapper file bytes survive. | Skip fetching/resetting, reset to base, delete untracked files, or read stale base JSON before resuming. |
| A12 Fast-forward after resume | From A11's resumed history, add the next milestone and push. Remote advances to new HEAD; old remote SHA is its ancestor and tracking ref follows it. | Resume only files rather than branch history, so retry push is non-fast-forward, or push an unmoved task ref. |
| A13 Remote refusal | Bare remote pre-receive hook rejects push. Verdict is failed with B/HEAD and rc 0; remote refs and local HEAD/index/work-file contents stay unchanged. | Treat rejected push as success/fatal, or reset/drop local commits to recover. |
| A14 Usage | Missing command, unknown command and extra argument each return rc 2 and usage information, without changing local state/config or remote refs. | Accept unknown/extra arguments, or return normal non-fatal rc 0 for usage errors. |
| A15 Never force | A second disposable clone advances the remote task branch independently; eligible original clone has a different local commit. Push refuses, rc 0 failed; remote tip stays on the other commit and local HEAD stays intact. | Add `--force`, `--force-with-lease` or a force refspec, or overwrite divergent history through reset/retry. |
| A16 Fetch refusal | Start finds an existing branch, but fetch fails (narrow Git shim or unavailable local transport after lookup). Failure names fetch; HEAD/index/working files remain intact. | Ignore fetch failure and reset against stale/missing tracking state or report resumed. |
| A17 Keep refuses dirty overlap | Real tracked edit conflicts with fetched remote changes. `reset --keep` refuses; start reports failed, rc 0; HEAD, index, dirty-file bytes and wrapper file remain intact. | Use reset-hard/checkout/stash, or ignore reset error and report resumed. |
| A18 Probe failure is not fresh | Local-only missing/broken origin or Git shim yields ls-remote error other than absent rc 2. Failure, rc 0; no fetch/reset/push and no local-work change. | Interpret every ls-remote nonzero as absent and report fresh. |
| A19 Verify remote equality | After real successful push, narrowly shim verification ls-remote to report missing/wrong SHA. Failure, rc 0, with branch/HEAD and observed mismatch; never print pushed. | Trust push exit 0, skip exact-ref lookup, or accept any returned SHA. |
| A20 Bounded Git | Narrow Git shim stalls a network call; the helper reaches failed within its documented bound, rc 0, and reaps the injected process. Check noninteractive Git invocation. | Remove the hard bound/termination or allow an interactive credential prompt to hang the run. |
| A21 Config refusal | Narrow shim refuses adding the needed local refspec; failed, rc 0, no subsequent push/fetch/reset; remote refs and local work unchanged. | Ignore config errors and publish without the tracking prerequisite. |
| A22 Repeat successful push | Push twice at same HEAD, verifying exact remote/tracking SHA both times and a single refspec. Each success line names the current subject. | Treat “Everything up-to-date” as failure, duplicate config, or reuse a stale cached verdict/SHA instead of verifying. |

A real Bash/Git fixture drives the branch/ref/history arms. Narrow shims are only for hard-to-trigger faults/verification/deadline and log the commands they intercept; no production test-only bypass or external network is introduced. Isolate each arm so refspec or retry history from another cannot make it pass accidentally. For negative controls/mutation runs, alter only disposable copies in the temp root, never the checked-out helper or caller repository.

### Wrapper simulation must preserve the production decision

Implement the following shell transliteration **verbatim in decision order and Git arguments**, with an accompanying source comment pointing to Step 5b in `cmd/ailang/coordinator_cloud_task.go` and `branchNeedsPush` in `coordinator_cloud_branch.go`. The production body is Go; the shell arm must preserve its actual rc/output semantics rather than implementing a corrected interpretation. Use this same function unchanged in A08 and A09. Capture output and error separately so a failed log's diagnostic does not become nonempty `logOutput`.

```bash
logOutput=$(git log "origin/${B}..HEAD" --oneline 2>"${fixture}/log.err")
logRC=$?
newBranch=false
lsRC=not-run
if [ "$logRC" -ne 0 ]; then
    git ls-remote --exit-code origin "$B" >"${fixture}/ls.out" 2>"${fixture}/ls.err"
    lsRC=$?
    if [ "$lsRC" -ne 0 ]; then
        newBranch=true
    fi
fi
# branchNeedsPush: newBranch || len(strings.TrimSpace(logOutput)) > 0
unpushed=false
if [ "$newBranch" = true ] || [ -n "$(printf '%s' "$logOutput" | tr -d '[:space:]')" ]; then
    unpushed=true
fi
```

The wrapper's ls-remote fallback deliberately uses bare `$B`; the production helper's exact-ref checks use `refs/heads/$B`. Do not silently improve the simulation, hand-create a tracking ref, fetch after the second commit, or merge stderr into log output. Record log rc, ls rc, newBranch, unpushed, HEAD, tracking SHA/absence and remote SHA in both arms. No Go build or wrapper changes are needed to observe the hazard.

## Gates and execution evidence

Required executor gates, from repository root:

```bash
/bin/bash .claude/skills/sprint-executor/scripts/test_cloud_branch_sync.sh
/bin/bash -n .claude/skills/sprint-executor/scripts/cloud_branch_sync.sh
/bin/bash -n .claude/skills/sprint-executor/scripts/test_cloud_branch_sync.sh
/bin/bash tools/launchd/test_agents_skills_sync.sh
make check-changelog
```

`make check-changelog` exists at `make/code-health.mk:219`; it checks the root index, so separately inspect the new fragment's existence and `### Changed — …` heading. **Added gate: `make test-launchd-drivers`**, to prove the new LAUNCHD_SUITE recipe runs under the existing shell-only harness and that integration does not regress the controller's baseline. Run on a suitable host/CI; it is not permission to run the full Go suite. Syntax/mirror/fragment inspection directly cover the mirrored files without redundant implementation tests. Existing `make test-changed` guidance stays intact; no new Go test gate is needed for shell/resource-only changes.

The new test exports a synthetic HOME under its temp root, `GIT_CONFIG_GLOBAL=/dev/null`, `GIT_CONFIG_SYSTEM=/dev/null`, `GIT_CONFIG_NOSYSTEM=1`, and disables inherited command-scope Git config/credentials, hooks, signing, repository redirection, task IDs and branch overrides before setting fixture values. Set author/committer identity only within fixture scope. Use isolated config/home/working directories for every Git command, including the wrapper simulation and shims; resolve the real Git path before adding shims. Never run git writes against the caller repo. Temp cleanup is bounded to its own fixture root. No sockets or external credentials are needed by the `file://` test.

**Sandbox interpretation:** Any obtained gate result involving sockets is **UNINFORMATIVE UNDER SANDBOX**, regardless of its exit code; report the affected gate and require host/CI evidence. Do not label it pass or fail. If a required tool is missing, record the gate, tool and skip in JSON notes; never provision it. New-script functional gates are implementation-stage work, not planner claims.

### Planner validation and a measured tooling mismatch

The sprint JSON schema (`resources/json_progress_schema.md`, Validation) permits at least one feature. The current validator (`scripts/validate_sprint_json.sh:92–99`) instead hard-fails counts below two. Keep this controller-requested one-milestone sprint: adding a dummy milestone or editing the validator would violate scope. No validator change is part of this sprint.

**Measured planner result:** `/bin/bash .claude/skills/sprint-executor/scripts/validate_sprint_json.sh M-FLEET-CLOUD-PUSH-PER-MILESTONE` ran in this sandbox and exited 1 with exactly one error: `Only 1 milestone(s) defined (minimum: 2)`. Its syntax, placeholder, reuse-audit and custom-value checks passed. A separate `jq -e` check of required fields/types, unique feature ID, initial state, numeric estimate, reuse linkage, plan paths and estimate/duration totals exited 0. This is a tooling policy conflict, not an inability to run under the sandbox. Neither validation command used sockets. Implementation gates have not been run by this planner. Final read-only status showed only the two requested new artifacts; the plan contains 22 arm rows. No git write operation was performed.

## Risks and limits

- **Live-branch push from a D2 mutant:** highest-impact risk. A01 puts new commits on live fixture branches and compares all remote refs; verdict-only tests would miss a helper that prints skipped and still pushes. A02–A05 cover other bypass paths; gate precedes refspec writes as well as pushes.
- **Wrapper's missing tracking ref:** M2 survives an apparently successful push. A08/A09 are paired evidence that the real shallow-clone shape exposes the defect and the refspec removes it. Do not confuse the forced fetch mapping with a force push.
- **Retry and dirty work:** run start only once before milestone work; `reset --keep` is the narrowly authorized retry protocol, not a general recovery command. A11/A17 ensure untracked files and conflicting dirty work survive. Fetch/config metadata updates are intentional; “untouched” on refusal refers to HEAD/index/working files, not rolling back fetched refs.
- **Auth, hook, timeout and remote races:** credentials may be absent (especially shadows); rejection is non-fatal but unpublished commits remain vulnerable until a successful push. Verify remote SHA after push, record failures, continue eligible work and leave final publication to the wrapper. Do not promise durability from an rc 0 failed verdict.
- **Resource consumption:** focused gates can still lack tools. Skip with recorded missing-tool evidence instead of installing, building into RAM `/tmp`, or changing cgo semantics. This sprint does not fix RAM allocation, scratch volumes or late failure marking.
- **Portability and isolation:** `/bin/bash` is 3.2.57 and git is 2.54.0 on this planner host (read-only version checks); CI may use a newer Bash. Explicit Bash 3.2-compatible syntax, isolated Git config and a `file://` shallow fixture matter. No caller HOME, global config, credential helper or repository writes from the test.
- **Instruction reach and size:** update both skill trees and test equality. Keep details in resources and use the existing SKILL link; avoid growing its enforced baseline.
- **Planning validator policy:** one valid-schema feature is intentionally retained despite the older validator's two-feature policy. Report its result to the controller; do not hide it with a dummy feature or self-approve execution.

Completion requires durable-branch behavior plus the focused test evidence and documentation, not a production Cloud Run replay. A production replay, infrastructure changes and the wrapper code fix remain outside this milestone.
