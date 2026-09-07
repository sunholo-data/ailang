# M-MISSION-ITERATION — one durable work item through validated completion

**Status:** In progress; Mark authorized execution on 2026-09-07: “lets go! spritn execute”.
**Design:** [Runtime contract, next increment](m-mission-runtime-contract.md#next-increment-proposal--binary-owned-work-item-iteration-7-september-2026).
**Duration:** 8 engineering days including integration buffer; not a calendar promise.
**Estimate:** 4,000 implementation/test lines. **Priority:** P0. **Risk:** High at crash,
cancellation and acceptance boundaries. **Release:** assign after validation.
**Progress:** `.ailang/state/sprints/sprint_M-MISSION-ITERATION.json`.

## Goal and delivery boundary

One explicit work item advances through its remaining approved stages using binary dispatch,
quota checks, durable recovery and validated artifacts. The output is a candidate ready for the
existing landing workflow. Implementation completion includes a reviewed canary activation
packet; a live successful canary is a separately tracked adoption gate, not inferred from tests.

Retain the full-workflow policy. Imported design/planning prerequisites let the first canary
start at execution without regenerating already-approved documents. Autonomous selection,
general `mission add`, non-Git projects, distributed workers, reservations, automatic merges,
decision ingestion and communications/outbox changes stay outside this sprint.

## Baseline and estimate

Source reviewed through local `a9a08e60d` on 7 September; other attended work is advancing HEAD.
Preserve unrelated benchmark JSON edits and the untracked handover. Execute in an isolated
worktree/branch containing the approved documents, after checking current ownership and base.
Do not continue or commit someone else's coordinator edits.

Existing dispatch and recovery increments are implemented. Recent concrete sizes:
`0c1a3b9cc` 296 insertions; `f8af911a7` 827; `a56b8e73f` 486;
`f8918e7b3` 591 insertions/15 deletions. These include tests, docs and progress records.
They demonstrate comparable delivery, not 2,200 lines/day of individual sustained velocity:
fleet history mixes multiple agents and concurrent sessions. The supplied velocity script
exited 141 at its `git log | head` pipeline and its file-stat calculation reads only HEAD~1;
its output cannot establish a seven-day rate. No repair to that script is included here.

Plan capacity is approximately 500 implementation/test lines per engineering day, an estimate,
with days 7–8 reserved for crash fixtures, integration, review and repairs. Increase the estimate
if adapter cleanup or authority validation changes the contract; do not weaken checks to fit it.
This planning session inspected test bodies but did not rerun the implementation tests.

## Frozen interfaces and implementation choices

### WorkItem v1

Strict JSON, one object, unknown fields rejected, maximum 1 MiB. Required fields:

| Field | Meaning / validation |
|---|---|
| `version` | Integer 1 |
| `mission_id`, `work_item_id` | Registered mission name; stable ID using existing attempt ID rules |
| `repository`, `base_revision` | Expected normalized origin and full local Git commit ID |
| `brief`, `allowed_paths` | Nonempty brief; exact repo-relative paths or directory prefixes ending `/`; no globs, absolute paths or traversal |
| `workflow` | `full-v1`; ordered subset of designer/planner/executor/evaluator, with earlier prerequisites supplied |
| `stages` | Ordered objects: `id`, `role`, `instructions`, `required_artifacts`, `authority_refs`, `limits`; IDs unique |
| `prerequisites` | Prior stage: role, artifact commit/path/hash, review/approval references and author model registry identity |
| `verification` | Check ID, nonempty argv array, relative cwd, positive timeout; repeatable local checks only |
| `limits` | Positive `timeout_seconds`, `max_tokens`, `max_cost_usd`; finite values; cumulative across remaining stages |

Stage limits use existing role-request ranges (1–1800 seconds, positive tokens/cost); iteration
timeout is 1–7200 seconds. At most four stages, 32 checks and 256 allowed/required paths.
Do not silently truncate oversized data. Resource caps count attempts, including failed attempts.
Each verification check is at most 600 seconds and shares the remaining iteration deadline.

The machine-local binding file is `~/.config/ailang/mission-runtime.toml`, version 1,
containing `state_db` and `workspace_root` absolute paths. It holds runtime placement only;
`missions/*.toml` remains the registry, and `models.yml` remains role policy. Missing binding
fails with setup instructions. All iterate/resume/cancel/status commands resolve this binding;
no per-invocation DB override. Tests inject paths through Go options. A trusted operator changing
the binding can split local state; this is not a cross-machine or hostile-operator lock guarantee.

Resolve routes with `ModelsConfig.ResolveRole` using explicit stage-to-registry-role mapping,
then existing dispatch resolution/capability checks. Validate the mapping against the actual
registry at implementation; no fallback role name or newly invented model default. Persist the
input bytes, canonical digest, relevant registry snapshot/digest and resolved instruction bytes.
Resume uses the snapshot; changed model policy requires a reviewed successor work item.

### StageResult v1 and Acceptance v1

Read `stage-result.json` from the stage workspace only after provider completion; maximum 64 KiB,
strict JSON. Fields: `version`, `request_digest`, `input_revision`, `output_revision`,
`artifact_paths`, `outcome`, `criteria`, `blocking_findings`. Outcome is `produced`, `pass`,
`fail` or `needs_decision`, valid for the named role. Evaluator `criteria` maps every frozen
criterion ID to pass/fail and evidence; extra/missing IDs reject. No bare numerical PASS shortcut.
Pass criteria IDs are supplied with the frozen brief as `acceptance_criteria` (ID/text objects).
`stage-result.json` is protocol output, not a product artifact; permit this one reserved untracked
file, copy its exact bytes outside the workspace, and reject all other unexpected dirty files.
Its output commit must not include this reserved file. No-change author output is rejected in v1.

Acceptance fields: version, work/stage IDs, request/outcome digests, output commit/tree,
artifact path/blob hashes, actual author route provenance, check receipts, evaluator receipt
reference when applicable, authority references, and acceptance digest. The binary supplies these
from inspected objects and dispatch receipts, never from an author-written acceptance document.
Persist exact bytes and hashes; bound each check output to 1 MiB and mark truncation explicitly.
An exit code is preserved even when output is truncated. Evidence paths stay outside author roots.

Authority reference: revision, repo-relative document path, stable decision locator and SHA-256
of referenced content, plus the prerequisite artifact digest it authorizes. Resolve against an
operator-approved input snapshot. This is attended input authority, not cryptographic proof of
the historical human identity. Any new/missing/changed decision produces decision-blocked status;
the runtime neither ingests inbox approvals nor edits canonical decision documents. A reviewed
new input may supersede the blocked item; resume cannot mutate frozen approval references.

Verify candidates in fresh worktrees at exact commits; check parent ancestry, required files and
changed-path scope using Git objects, including deletions/renames and symlink blobs. Verification
and evaluator commands must not modify tracked candidate content. Changed verification worktrees
reject; generated ignored build outputs may remain local. Evaluator output uses the same reserved
protocol file and may not change the product tree. Recheck hashes immediately before acceptance.

### Store and recovery APIs

Coordinator owns additive schema and transactions; the iteration package depends on a narrow
store interface. Coordinator must not import iteration types. Use existing migrations and bound
SQL arguments. Keep current standalone attempt APIs and terminal state semantics compatible.

Add work-item records plus an explicit mission admission/lease row keyed by mission ID. Lease
renewal does not increment semantic cancellation version. Acceptance rows are immutable per stage.
Store operations: admit/reclaim work item, renew ownership, record wait, prepare/start child under
parent fence, accept-and-advance, finish, cancel-parent-and-child and inspect/reconcile.
All checks requiring both parent/child ownership happen in one SQLite transaction. Use DB time.

Keep the admission row assigned to an unfinished item even while quota/decision-blocked or
ambiguous. Release only when completed, failed without a live/ambiguous child, or safely cancelled.
This favors safety over picking another task during a blocker. A resume reclaims only expired
ownership; a fresh CLI invocation never steals a live parent.

The resume decision table in the design is mandatory. After reclaiming an expired parent, an
unstarted child can gain a new owner only once its old lease expires; a live/running child is
observed or cancelled, never replaced. Successful child outcomes can be validated without rerun.
Unknown post-dispatch outcomes require reconciliation. Record preflight blocks before child claim;
if quota changes after claim retain prepared state, release its lease safely and resume the same
immutable request with a new exclusive receipt generation. Preserve every old receipt.

Cancellation commits the parent/child fence before requesting process termination. Runtime status
retains reconciliation-required if an owned external process cannot be confirmed stopped; never
launch a replacement solely because cancellation returned. Unix process-group cleanup is required
for admitted adapters. First runtime support is macOS/Linux; Windows iterate fails before dispatch
until receipt sync and descendant cleanup guarantees are supported. Existing Windows APIs remain.

### CLI and compatibility

Commands match the parent design: iterate/dry-run, status/json, resume and versioned cancel.
JSON status carries `phase`, `reason_code`, `next_action`, lease/deadline, stage, route and evidence.
Exit codes: 0 completed or successful read-only command; 2 invalid input/config; 3 waiting;
4 reconciliation-required; 5 execution/verification failure; 130 cancellation. No status query
creates a DB or changes a record. Reconcile remains an explicit mutation through existing tools
extended to coordinate parent state. A completed iterate returns existing evidence without execution.

The compatibility branch uses an explicit `AILANG_MISSION_WORK_ITEM` input path, checked after
pin/project-root setup and existing kill-switch/memory/overlap checks, before any model probe.
Invoke the tested binary once and propagate its status; no controller prompt or retry loop.
The input is not deleted on success, so a repeated scheduler fire is an idempotence check rather
than a new task. Production env generation/pins change only in the reviewed canary activation.

## Milestones

### M1: Input contract and shared quota admission (~650 LOC)

**Dependencies:** None. **Duration:** day 1. **Estimate:** 350 implementation + 300 tests.
Own `internal/mission/iteration/spec*.go`, `internal/mission/admission*.go`, dispatch hook and
role CLI wiring. Avoid import cycles by injecting admission into dispatch, with policy supplied
by production callers. Missing policy errors, including standalone role-run; pure dry-run skips it.
Example: `cmd/ailang/testdata/mission-iteration/work-item.json`, rendered with temp-repo commits
by fixtures; companion malformed and quota-blocked cases.

- [x] Strict input/result limits and role ordering reject malformed, ambiguous or unsupported input.
- [x] Existing Codex/Ollama/ledger policy is reused before construction/health and before dispatch.
- [x] Unknown/over quota, wire aliases and explicit pins cannot leak a protected provider call.
- [x] Dry-run creates no DB/worktree/receipt and makes zero provider calls.
- [x] Focused spec/admission/dispatch tests and lint pass.

### M2: Parent ownership and atomic child fences (~900 LOC)

**Dependencies:** M1. **Duration:** days 2–3. **Estimate:** 450 implementation + 450 tests.
Own coordinator `mission_work_item*.go`, migration hook, parent-aware attempt APIs and acceptance
schema. Example fixtures: concurrent starts, parent loss, prepared quota wait, acceptance replay.

- [x] Two DB connections racing identical/different work IDs produce one active mission admission.
- [x] Parent cancellation and lease loss fence child start and accept-and-advance transactionally.
- [x] Quota-blocked prepared attempts resume same request without terminalizing or replacing work.
- [x] Accepted stages remain immutable; altered input/digest/outcome and stale owners fail.
- [x] Existing standalone attempt recovery/cancellation tests pass unchanged in meaning.

### M3: Git artifacts and stage acceptance (~850 LOC)

**Dependencies:** M1, M2. **Duration:** days 3–4. **Estimate:** 400 implementation + 450 tests.
Own iteration `artifacts*.go`, `verify*.go`, `authority*.go`; reuse bounded Git command facilities
after checking existing helpers. Example repositories: clean committed result, wrong-origin result,
dirty output, stale evaluator and tampered check policy. All generated under temporary roots.

- [ ] Clean committed work is recognized; ancestry/scope/required artifacts are verified from Git.
- [ ] Runtime accepts only the reserved protocol file as untracked result output.
- [ ] Frozen checks execute at the recorded commit with bounded output and deadlines.
- [ ] Wrong revision, changed authority, failed hard checks, mutated candidate or stale judge fails.
- [ ] Actual prior author receipts determine evaluator independence, including fallback routes.

### M4: Iteration execution and process recovery (~900 LOC)

**Dependencies:** M2, M3. **Duration:** days 5–6. **Estimate:** 400 implementation + 500 tests.
Own iteration service/state-machine and move `mission_role_state.go` orchestration behind it;
targeted cleanup in `internal/executor/{claude,codex,pi}` only where needed. Codex has inspected
process-group helpers; Pi/Claude inspected kill paths target the leader. Reuse a common helper
if the adapter audit supports it; no broad executor refactor. Examples: counted fake providers
driving successful execution/evaluation and each crash boundary in isolated subprocesses.

- [ ] Sequential remaining stages dispatch once, with immutable inputs and persisted acceptance.
- [ ] Crash tests cover before/after dispatch, DB/receipt disagreement and acceptance-before-next.
- [ ] No fallback/retry after execution starts; completed role is validated without another call.
- [ ] Parent cancellation stops owned descendants for each admitted adapter; unrelated processes live.
- [ ] Iteration/stage deadlines and remaining budgets survive restart and cannot reset on resume.
- [ ] Lost parent with live child waits; ambiguity is preserved and never auto-reclaimed.

### M5: CLI status and one-shot driver seam (~450 LOC)

**Dependencies:** M4. **Duration:** day 7. **Estimate:** 250 implementation + 200 tests.
Own mission iterate/status/resume/cancel commands, help.go and mission_cmd.go, local binding loader,
the narrow driver branch and its Bash fixtures. Read cli-doc-maintainer when implementation starts.
Example: `docs/docs/guides/mission-iteration.md` using a disposable repository and fake executor;
update `mission-role-dispatch.md` for mandatory quota admission and support limitations.

- [ ] Status distinguishes running, waiting reasons, reconciliation, failure and validated completion.
- [ ] Missing config/DB and changed inputs fail loudly without creating status-side state.
- [ ] Driver fixture shows zero legacy probes/controller calls/retries on the opted-in path.
- [ ] Foreign-repository fixture remains in its intended repository through actual driver entry.
- [ ] Repeated completed invocation returns the same evidence and zero extra provider calls.
- [ ] CLI help and documented fixtures agree; Bash 3.2 compatibility tests pass.

### M6: Integration evaluation and canary packet (~250 LOC)

**Dependencies:** M5. **Duration:** day 8. **Estimate:** 50 fixture glue + 200 integration assertions.
Own remaining cross-boundary fixtures and `design_docs/verification/mission-iteration/` evidence.
Load sprint-evaluator for independent assessment after implementation. Docs and review text are
excluded from the LOC estimate. Example: runnable hermetic end-to-end command plus durable report.

- [ ] Fault matrix and all preceding acceptance criteria pass with provider call counts recorded.
- [ ] Required tests/lint/build/boundary checks and relevant CI asset/shell checks pass.
- [ ] Independent evaluation passes with zero blocking findings; repairs are rechecked.
- [ ] Canary packet identifies an unowned approved Docs item, base, binary hash, routes, token caps,
      at most $5 metered spend/60 minutes, isolated paths, idle check and exact rollback commands.
- [ ] Canary remains unactivated until reviewed; implementation status and live adoption are separate.

## Test execution and tracking

Write failing boundary tests before code. Use temporary homes, DBs and repositories; tests never
read production DBs, acquire provider credentials, call inference or mutate mission scheduling.
Bound every subprocess. Test API refusal assertions alongside fake-provider invocation counts.
Crash fixtures must use abrupt process exit, not only mocked errors or orderly deferred cleanup.

Run focused package tests per milestone. At integration run full `make test`, `make lint`, build,
`make check-boundaries`, race tests for changed runtime/store packages and the relevant Bash 3.2
driver suite. Inspect included Make targets before invocation. If Pi embedded assets change,
run `make pi-assets`/`make verify-pi-assets`; do not change assets just to trigger those checks.
Record inherited failures separately; a green local subset is not a claim all CI gates passed.

Use the six real milestone IDs/dependencies in the progress JSON. Mark passes only with evidence;
do not change the acceptance contract to hide a failing test. Each milestone has a narrow commit
on the isolated branch; avoid staging unrelated work. No automatic issue-closing references are
inferred from the parent design's historical issue mentions.

## Live adoption gate (outside implementation completion)

Before activation refresh Docs status/ownership and select the actual task; it may change while
this sprint is being implemented. Do not spend provider quota to choose or prepare it. Suspend
the legacy Docs next fire only after its active iteration exits, preserving configuration backups.
Run the tested artifact once under the reviewed caps. Success requires a useful validated Docs
artifact and persisted binary dispatch/acceptance, then repeat invocation with no duplicate work.
Report actual costs/unknown usage, elapsed time and interventions without claiming one sample
proves productivity improvement. Restore scheduling using the packet; ambiguous work stays held.

No runtime deployment, process restart, outbound handoff message or canary activation is performed
by sprint planning. The next step after plan review is the repository's “execute sprint” gate.
