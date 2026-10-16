# M-COORDINATOR-AUTO-APPROVAL-POLICY: the coordinator approves its own cards, under a typed policy

**Status**: Planned (drafted attended 2026-10-15; Mark approved writing it the same day)
**Target**: v0_54_10 onward (five milestones; M5 is gated on two weeks of evidence, not on a date)
**Priority**: P1. Pending cards are stalled chains, and nothing drains them while V1 is paused.
**Estimated**: about 8 working days for M1–M4, then a 14-day or longer shadow window before M5
**Dependencies**: Part B of [M-TRIAGE-MISSION](m-triage-mission.md), the triage loop, which
writes the attestations P7 reads and the shadow ledger M5 is scored against. M1–M3 do not
depend on it.
**Lane**: AILANG-fix lane ([PROGRAM.md](../PROGRAM.md)): the policy is written in AILANG, and Go
is the shell (D-TRIAGE-8).
**Parent**: [m-triage-mission.md](m-triage-mission.md) Part A; charter
[triage-mission.md](../triage-mission.md) (D-TRIAGE-7, -8, -9).

## Summary

Every 10 minutes, on the cloud coordinator's existing sweep cadence, the coordinator evaluates
each pending card on an allow-listed agent. It gathers facts in Go (the card, the PR head, its
checks, mergeability, files, `dev` CI, the triage attestation, and the last 24h of policy
approvals) and calls `decide(facts, config)` in an AILANG module. The module returns
`Approve(approver)` or `Hold(clause, reason)`. An approval does not dispatch anything. It moves
the card to a new state, `approved_awaiting_merge`, and squash-merges the PR pinned to the
attested head SHA. The landed-card sweep then dispatches the handoffs once the merge exists.
The policy runs in shadow mode first. In shadow mode it records what it would have decided
and approves nothing.

Four facts found while verifying the parent design change how this has to be built. Each
is in the Verification Log.

1. **The coordinator image contains no `.ail` source** (`cloudbuild-dev.yaml:98`,
   `GOOGLE_CLEAR_SOURCE=true`), and `embed.ProjectRoot` resolves modules from the filesystem. The
   precedent pattern cannot run in the coordinator as written. The policy module must be
   embedded in the binary (M1).
2. **The existing authority check REFUSES outside a repo checkout, even with
   `AILANG_APPROVAL_CONTROLLER=1`.** Measured: `authority: false`, "no
   internal/dashboard_transforms/approval_authority.ail found … (set AILANG_PROJECT_ROOT)". If
   `Granted` were enforced as the parent design proposes, every attended CLI approval run from
   outside the repo would be locked out. That is the 16-day lockout again, from a different
   cause. M1's embedding fixes it before M3 enforces anything.
3. **"The author must be the bot account" proves nothing.** The coordinator's PRs are authored
   by `sunholo-voight-kampff` (all 22 open `coordinator/task-*` PRs), the executor job holds a
   `GITHUB_TOKEN` (`cmd/ailang/coordinator_cloud_task.go:43-48`), and the triage loop posts as
   the same account. An executor could write an "attestation" on its own PR. So the
   attestation is also HMAC-signed with a key that executor jobs cannot read (§5).
4. **A dashboard approval of a cloud task never merges its PR** (read in code; not yet observed
   live). The dashboard calls `ProcessApprovalRequest` with `SkipMerge: false`
   (`internal/server/handlers_approvals.go:329-343`). A cloud task has no worktree, so the
   processor completes the task without merging (`approval_processor.go:273-284`). Only the CLI
   calls `reconcileTaskPR` (`coordinator_approvals_remote.go:301`), and the landed-card sweep
   only looks at `pending_approval` tasks (`daemon_landed_cards.go:76`). H2 on the dashboard
   path is therefore not "if the merge is refused". It happens every time.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | The decision is a pure AILANG function of a recorded fact record. Shadow events store the inputs' identity (head SHA, policy version), so a decision can be re-derived |
| A2: Replayability | +1 | Every evaluation that changes outcome writes a task event (`auto_approval_eval`) with version, head SHA, decision and clause |
| A3: Effect Legibility | 0 | The policy is `! {}`. Effects (GitHub REST, Firestore) stay in the Go shell and are named per call |
| A4: Explicit Authority | +1 | Approving unattended becomes an explicit, versioned, testable rule whose approver string names the policy version and the judge. Today it is free text (`--by`) |
| A5: Bounded Verification | +1 | Inline `tests [...]` on every clause, run by `ailang test` without the Go suite. 25 already pass on the draft (V15) |
| A6: Safe Concurrency | 0 | Single daemon sweep. The merge is pinned to a SHA (`sha` on REST merge), so a concurrent push fails with 409 instead of merging unreviewed code |
| A7: Machines First | +1 | Hold reasons are a closed `Clause` ADT plus a string. `approvals --json` carries them for the triage loop |
| A8: Minimal Syntax | 0 | No language change |
| A9: Cost Visibility | +1 | The P6 cap bounds dispatched spend to `max_per_day × task_max_cost` (one target per card), stated in config |
| A10: Composability | +1 | Reuses `ProcessApprovalRequest`, the landed-card sweep, `DecidePR`'s scope check and the authority engine pattern. No parallel approval path |
| A11: Structured Failure | +1 | A policy that cannot be evaluated holds the card with the error as its reason. It never approves on error and never falls back to Go (same posture as `coordinator_approvals_engine.go:26-34`) |
| A12: System Boundary | +1 | The GitHub boundary is crossed with a SHA-pinned merge, and the triage→coordinator boundary with a signed, SHA-bound attestation |

**Net score: +9.** No −1 on A1/A3/A4/A7. Proceed.

## Problem Statement

Measured 2026-10-15 in the parent doc: 21 pending cards, 22 open `coordinator/task-*` PRs, and
green chains six days old (#1747, #1749, #1750). Approving is the only thing that advances a
chain, and only a human approves. The parent design lists eleven hazards (H1–H11) in what
approving and merging do today. Part A closes the four that live in the coordinator's code:

- **H1** handoffs fire before the merge (`approval_processor.go:223-257`, then
  `coordinator_approvals_remote.go:301`);
- **H2** a refused or failed merge leaves an orphan PR, and on the dashboard path the merge is
  never even attempted (finding 4);
- **H3** the authority gate is advisory (`resolveApprovalAuthority` labels; the remote path uses
  only `.Identity`, `coordinator_approvals_remote.go:231-240`);
- **H5** `ApprovedBy` is free text, and the squash body names the task but not the approver
  (`coordinator_prs.go:253-254`).

## Goals

**Primary goal:** a pending card whose PR is green, clean, in scope, attested by the triage loop
at its current head, and within the daily cap is approved, merged and handed off by the coordinator, with no
human click and no dispatch before the merge.

**Success metrics**
- During ≥14 days of shadow mode: zero would-approve decisions on a card that a human later
  rejected, whose PR was closed unmerged, or whose change was reverted. At least 10
  would-approve decisions in the window, or the window is extended.
- After M5: no auto-approved handoff is dispatched before its PR's merge commit exists on
  `dev` (asserted per event).
- Zero lockouts. The attended CLI path (`AILANG_APPROVAL_CONTROLLER=1`, any cwd) and the
  dashboard approve successfully in the M3 acceptance run, and the refused cases refuse.
- Every policy approval's merge commit body and `ApprovedBy` name the same
  `policy:<version>+attestation:<judge>` string.

## High-Impact Decisions

| Decision | Why high impact | Chosen by | Deadline | Change cost |
|---|---|---|---|---|
| D1 Where the policy lives | Rules typed and testable, or a Go if-chain | **human: RESOLVED D-TRIAGE-8**: an AILANG module, Go is the shell | design | high |
| D2 Attestation carrier | What the coordinator trusts | **human: RESOLVED D-TRIAGE-9**: a machine-readable PR comment | design | high |
| D3 Attestation authenticity: HMAC key vs a separate GitHub identity | Without one, any executor can attest its own PR (finding 3) | human | design | med |
| D4 Merge mechanism: REST `PUT …/merge` with `sha` pin, vs `--auto` | The coordinator image has no `gh` (`daemon_landed_cards.go:14-16`) | agent (recommend REST + SHA pin, §3) | design | med |
| D5 Do human approvals of cloud tasks also move to dispatch-after-merge? | H1/H2 apply to human approvals too; dashboard orphans (finding 4) | human | M2 | med |
| D6 H3 enforcement scope: approve only, or reject too | Remote reject is final (H6); enforcing it widens lockout risk | human | M3 | low |
| D7 BEHIND handling | `dev` protection is `strict: false`, so most agent PRs are BEHIND; P3 holds them all | human | M4 | med |

### Design Freeze

- [x] D1: AILANG module (D-TRIAGE-8, Mark 2026-10-15)
- [x] D2: PR comment (D-TRIAGE-9, Mark 2026-10-15)
- [ ] D3: HMAC key (recommended) or a second GitHub identity for the triage loop
- [ ] D5: move human cloud approvals to the same path in M2 (recommended), or policy-only
- [ ] D7: hold BEHIND (recommended for v1), or let the coordinator call `update-branch`, which forces
  a re-attestation

## Verification Log

Every load-bearing claim, checked against HEAD `d1e33bea6` on 2026-10-15.

| # | Claim | How verified | Result |
|---|---|---|---|
| V1 | Approve dispatches handoffs before any PR merge | Read `approval_processor.go:211-257` (dispatch at step 2.5) and `coordinator_approvals_remote.go:272-301` (`reconcileTaskPR` after `ProcessApprovalRequest` returns) | Confirmed (H1) |
| V2 | A failed handoff still reports success | `approval_processor.go:245-250` appends "HANDOFF FAILED" to the message, `Success` stays true | Confirmed. Parent cites `:247-250` under `approval_handoff.go`; the lines are in `approval_processor.go` |
| V3 | CLI merge = `gh pr merge --squash --delete-branch --auto`, no `--admin` | `messaging/github_pr.go:226-227` | Confirmed |
| V4 | The coordinator image has no `gh` and no source tree | `daemon_landed_cards.go:14-16` (no `gh`); `cloudbuild-dev.yaml:87-99` buildpacks with `GOOGLE_CLEAR_SOURCE=true`; `internal/feedbackgate/shadow_program.go:10-13` ("coordinator images copy /usr/local/bin/ailang, not the source tree") | Confirmed. New |
| V5 | `embed.ProjectRoot` resolves `.ail` from disk only (`AILANG_PROJECT_ROOT` or cwd ancestors) | `internal/embed/root.go:40-63`; the comment at `:17-19` says the dashboard image "contains the binary and no `.ail` files" | Confirmed. New |
| V6 | Authority REFUSES outside the repo even with the attended grant | Ran `AILANG_APPROVAL_CONTROLLER=1 ailang coordinator approvals --json --remote local` from the scratchpad: `"authority": false … no internal/dashboard_transforms/approval_authority.ail found`. Control run in the repo: `"authority": true, "identity": "attended controller session"` | Confirmed. New, and the main lockout risk |
| V7 | `Granted` is never checked on the remote path | `coordinator_approvals_remote.go:231-240` reads `.Identity` only; `grep -n Granted cmd/ailang/coordinator_approvals_remote.go` is empty | Confirmed (H3) |
| V8 | The authority module says it is not enforcement | `coordinator_approvals_authority.go:44-48` | Confirmed. Enforcing it in the CLI is a routing rule for a cooperating fleet, not a security boundary. §4 says so |
| V9 | The 09-07..09-23 lockout came from a guard that asked the wrong context (the laptop's registry), not from an authority check | `HANDOVER-message-plane-approvals-2026-09-23.md` §1; `coordinator_approvals_remote.go:242-266` | Confirmed. The parent says the lockout "came from exactly this kind of check". The precise lesson is a check evaluated in the caller's environment rather than the decision's, which V6 shows the authority check also has |
| V10 | The dashboard path never consults `resolveApprovalAuthority` | It lives in `cmd/ailang` (package `main`); `internal/server` cannot import it; `handlers_approvals.go:329-343` hard-codes `ApprovedBy: "dashboard-user"` | Confirmed |
| V11 | A dashboard approval of a cloud task does not merge its PR | `handlers_approvals.go:329-343` (no `reconcileTaskPR`); `approval_processor.go:273-284` (empty `WorktreePath` → complete without merge); worktree set only on the local lane (`daemon_tasks_exec_run.go:497`); `grep -rn reconcileTaskPR internal/` empty | Confirmed in code. **PENDING live**: M2 step 0 lists open PRs whose card shows `resolved_by=dashboard-user` |
| V12 | The landed-card sweep sees only `pending_approval` tasks with `pending` cards | `daemon_landed_cards.go:76`; `pr_landed.go:23-39` (`DecideLandedCard` requires both) | Confirmed. M2 must widen it for the new state |
| V13 | Boot recovery fires handoffs for any `approved` merge_handoff card not marked triggered | `daemon.go:599-640`; `store_sqlite_approvals.go:371-379` (`status = 'approved' AND handoffs_triggered = 0`); same in `internal/storage/firestore/coordinator_approvals.go` | Confirmed. **So "approve now, dispatch later" cannot reuse status `approved`**: boot recovery would dispatch before the merge. Hence the new status |
| V14 | Stranded-approval sweep keys on status `approved` | `daemon_stranded_approvals.go:24-38` | Confirmed. The new status is invisible to it, which is correct |
| V15 | The draft policy module type-checks and its inline tests pass | `ailang check` → "No errors found"; `ailang test` → 25 passed, 0 failed; `ailang run` demo produced `Approve(policy:v1+attestation:claude:sonnet)` and the expected `Hold(P1DevCi …)`, `Hold(P7Attestation … stale after a push)`, `Hold(P3Mergeable …BEHIND)`, `Hold(P6Rate …)`, `Hold(P5Paths …)`, `Hold(P8Tripped …)` | Confirmed (AILANG v0.54.9-14) |
| V16 | `std/list` has no `all` | `ailang check` with `import std/list (all)` → `IMP010: symbol 'all' not exported by 'std/list'` | Confirmed. The draft uses `any(not …) == false` |
| V17 | ADT results reach Go as tagged maps | `internal/embed/convert.go:168` (TaggedValue → `map[string]interface{}` with `__tag`) | Confirmed |
| V18 | Go→AILANG precedent: load ~65ms once, ~13µs per call; no Go fallback | `cmd/ailang/coordinator_approvals_engine.go:23-34, 51-60`; also `internal/riggate/policy.go:59` (`LoadPolicy`, a mutex-guarded decider) | Confirmed. No benchmark exists for the authority module itself. M1 adds one |
| V19 | `internal/coordinator` does not import `internal/embed` today; the import is sanctioned | `grep -rln 'internal/embed"' internal` (riggate, server, apiserver only); `.claude/rules/architecture.md` (embed is the bridge) | Confirmed. M1 runs `make check-boundaries` |
| V20 | PR authors and the executor token | `gh pr list`: all 22 open `coordinator/task-*` PRs are by `sunholo-voight-kampff`; `coordinator_cloud_task.go:43-48` configures git from `GITHUB_TOKEN`; `Dockerfile.agent-base:58-71` installs `gh`, which authenticates from it | Confirmed (finding 3) |
| V21 | `dev` protection: required `test, lint, build, docs-gate, UI build gate`, `strict: false`, `enforce_admins: false` | `gh api repos/sunholo-data/ailang/branches/dev/protection` | Confirmed. H9 holds (admin and bot bypass). BEHIND is allowed by protection, so P3's "not BEHIND" is our rule, not GitHub's |
| V22 | No generic KV in the coordinator store | `store.go:205-360` method list: tasks, approvals, events only | Confirmed. The daily counter and P8 are derived from approvals and task events plus config, with no new store |
| V23 | `auto_merge_code` is a completion-time, in-job merge with a non-author approver, not a card-time policy | `agent_automerge.go:9-35`; `internal/dispatch/cloudrun/automerge.go:12-22`; [m-auto-merge-code-prs.md](v0_52_6/m-auto-merge-code-prs.md) | Confirmed. Distinct mechanism. The policy skips agents with `auto_merge: true` (their PR lands on its own and the landed-card sweep handles it) |
| V24 | Parent line cites | `agent_registry.go:122-131` (handoff fields), `:157-162` (auto-merge fields); `task_max_cost` default `agent_config.go:203` ($25), `:209` ($15); `autonomy_router.go:22-47` | Confirmed. The parent's `agent_registry.go:123-131` is off by one; the auto-merge fields are at `:157-162`, not in that range |
| V25 | The approval authority module is CLI-only today | `grep -rn callDecide` → `cmd/ailang` only | Confirmed |
| V26 | Executor job service accounts cannot read a new Secret Manager secret | Read `ailang-multivac/terraform/iam.tf` (local checkout, branch `dev` @ `4976649`, 2026-10-09, not fetched). Agent lanes get **per-secret** `secretAccessor` only (`iam.tf:263-272`, M-SEC2, "replaces the project-wide secretAccessor"). The project-wide accessor is held only by the coordinator (`:57-61`), dashboard (`:345-349`) and website-builder (`:362-366`) SAs | Confirmed in terraform: a secret not added to `agent_lane_secret_pairs` is unreadable by executors. Residual: the dashboard and website-builder SAs can read it, and neither runs agent code. **M4 step 0** re-checks against live prod IAM (`gcloud secrets get-iam-policy` plus the project policy) because the local checkout may lag the prod branch |

### Corrections to the parent design (m-triage-mission.md)

1. "A failed handoff still reports … (`:247-250`)": the lines are `approval_processor.go:245-250` (V2).
2. "H2 … if the merge is refused": on the dashboard path the merge is never attempted (V11).
3. "Approving through the cloud path … `--auto`": the coordinator daemon has no `gh`, so
   the daemon cannot use `gh pr merge --auto`. It must use REST (V4, §3).
4. "The landed-card sweep dispatches the handoffs when the merge lands, which it already does": it does
   so only for cards still `pending`. An approved card is invisible to it (V12), and status
   `approved` would be dispatched early by boot recovery (V13). A new state is required.
5. "Enforce the authority gate (H3). `coordinatorResolveRemote` checks `Granted`": as written,
   this locks out every attended approval made outside a repo checkout (V6).
6. "Attestation … author must be the bot account": not sufficient (V20, finding 3).
7. "H5 … the squash commit names no approver": partly. It names the task and `DecidePR`'s reason
   (`coordinator_prs.go:253-254`) but not the approver.
8. `agent_registry.go:123-131`: the handoff fields are at `:122-131`, and the auto-merge fields cited
   alongside them are at `:157-162` (V24).

## Solution Design

### 1. The policy interface (AILANG)

File: `internal/dashboard_transforms/auto_approval_policy.ail`. Below are the type section and
`decide` exactly as checked (V15). The formatter hook printed them in canonical one-line form.
The full draft, with 25 inline tests over `checkPasses`, `checksGreen`, `mergeableClean`,
`denied`, `pathViolations` and `attestationProblem`, is the M1 starting point.

```ailang
module internal/dashboard_transforms/auto_approval_policy
import std/list (any, filter, length)
import std/string (startsWith, toUpper, join)

export type Check = { name: string, conclusion: string }

-- The SHELL verifies authorship and the HMAC; this module reads only the outcome, so a
-- forged comment arrives here as signatureOk: false, never as a verdict.
export type Attestation = { present: bool, signatureOk: bool, authorOk: bool, verdict: string, headSha: string, judge: string, verifier: string, ageHours: int }

export type DevCi = { headSha: string, conclusion: string }

export type Facts = { taskId: string, approvalType: string, diffAvailable: bool, scopeOk: bool, scopeReason: string, prHeadSha: string, checks: [Check], mergeable: string, mergeState: string, prFiles: [string], devCi: DevCi, attestation: Attestation, approvedLast24h: int, handoffTargets: int, evaluation: string, taskCostUsd: float, taskMaxCostUsd: float, tripped: bool, trippedReason: string }

export type Config = { version: string, maxPerDay: int, denyPrefixes: [string], goldenPrefix: string, attestationMaxAgeHours: int }

export type Clause = P0Scope | P1DevCi | P2Checks | P3Mergeable | P4Diff | P5Paths | P6Rate | P7Attestation | P8Tripped

export type Decision = Approve(string) | Hold(Clause, string)

-- Ordered cheapest-and-most-global first: a tripped switch or a red dev holds every card
-- before any per-card fact is read.
export func decide(f: Facts, c: Config) -> Decision ! {} {
  let bad = pathViolations(f.prFiles, c.denyPrefixes, c.goldenPrefix)
  let attest = attestationProblem(f.attestation, f.prHeadSha, c.attestationMaxAgeHours)
  if f.tripped then Hold(P8Tripped, f.trippedReason) else if toUpper(f.devCi.conclusion) != "SUCCESS" then Hold(P1DevCi, "dev CI on ${f.devCi.headSha} is ${f.devCi.conclusion}") else if f.scopeOk == false then Hold(P0Scope, f.scopeReason) else if f.diffAvailable == false then Hold(P4Diff, "no visible diff on the card") else if length(bad) > 0 then Hold(P5Paths, "denylisted paths: ${join(", ", bad)}") else if checksGreen(f.checks) == false then Hold(P2Checks, "not every check on ${f.prHeadSha} is SUCCESS or SKIPPED") else if mergeableClean(f.mergeable, f.mergeState) == false then Hold(P3Mergeable, "PR is ${f.mergeable}/${f.mergeState}") else if f.approvedLast24h >= c.maxPerDay then Hold(P6Rate, "daily cap reached") else if f.handoffTargets > 1 then Hold(P6Rate, "approving would dispatch ${show(f.handoffTargets)} stages; auto-approval dispatches at most one") else if f.taskCostUsd > f.taskMaxCostUsd then Hold(P6Rate, "task cost over task_max_cost") else if f.evaluation != "" && toUpper(f.evaluation) != "PASS" then Hold(P7Attestation, "evaluator verdict is ${f.evaluation}") else if attest != "" then Hold(P7Attestation, attest) else Approve("policy:${c.version}+attestation:${f.attestation.judge}")
}
```

**Clause → fact mapping** (P1–P8 from the parent, plus P0):

| Clause | Rule as implemented | Fact source (Go shell) |
|---|---|---|
| P0 Scope | The PR is on `coordinator/<task>`, every file is inside the agent's declared `artifact_patterns`, and the card's file list equals the branch's | One implementation: extract `CheckPRScope(agent, changedFiles, cardFiles)` from `DecidePR` (`pr_reconcile.go:83-126`). `DecidePR` calls it, and the shell passes `scopeOk/scopeReason`. Today these checks run after the decision (H2). Here they run before it |
| P1 `dev` CI | The `CI` workflow's run on `dev`'s **current** head concluded SUCCESS. In progress or missing → hold | REST `GET /repos/{r}/commits/dev`, then `GET /actions/workflows/ci.yml/runs?head_sha=…` |
| P2 Checks | **Every** check-run and commit status on the PR head is SUCCESS or SKIPPED, and there is at least one | REST `GET /commits/{head}/check-runs` plus `/status`, normalised to upper case |
| P3 Mergeable | `mergeable == MERGEABLE` and `mergeable_state == clean`. BEHIND, BLOCKED, DIRTY, UNSTABLE and UNKNOWN hold | REST `GET /pulls/{n}` (`mergeable`, `mergeable_state`). UNKNOWN on first read is normal while GitHub computes it, so it holds and the next tick retries |
| P4 Diff | `diff_available` | `context_json.diff_unavailable == ""` and `changed_files` non-empty, the same reading as `approvals --json` |
| P5 Paths | No file under `internal/coordinator/`, `cloudbuild`, `tools/launchd/`, `.github/workflows/`, `missions/`; `std/` only when `.stdlib-golden/` moved too | REST `GET /pulls/{n}/files`; prefixes from config |
| P6 Rate/cost | Under 4 policy approvals in a rolling 24h; at most one handoff target per card; the task's own cost within `task_max_cost` | Derived: count of cards with `resolved_by` prefix `policy:` and `resolved_at` in the last 24h, across `approved_awaiting_merge` and `approved` (`ListResolvedApprovals`); `len(approvalHandoffTargets(agent))`; task metrics |
| P7 Judgement | A valid attestation at the PR head (§5). If an evaluator verdict exists on the card, it is PASS | §5 parser; `ApprovalRequestRecord.Evaluation` |
| P8 Tripped | No policy-merged commit since `enabled_since` has a non-SUCCESS `CI` run on `dev`. An unreadable result, or no conclusion 6h after the merge, also trips it (fail closed) | Derived from settled `auto_approval_merge_ci` events. Only merges whose CI has not yet concluded are queried, at most `max_per_day` (§6.4) |

The worst-case daily spend from dispatch is `max_per_day × 1 target × task_max_cost` = 4 × $25
= $100 at the defaults. This matches D-TRIAGE-6's estimate, and P6 now enforces it.

### 2. The Go shell

New package-private files in `internal/coordinator`. They are in the coordinator because the
facts are coordinator records, and the bridge is `internal/embed` (V19).

- **`auto_approval_engine.go`** loads the module once (`sync.Once`) through a new
  `embed.NewForEmbeddedModule(fsys, modulePath)`. It never uses `NewForModule`, which needs a
  source tree (V4, V5). It calls `decide` and decodes the tagged map (V17). There is no Go
  fallback: a load or call error yields `Hold(error)`, logged loudly, and the card is
  untouched. This follows the posture of `coordinator_approvals_engine.go:26-34`.
- **`internal/dashboard_transforms/embed.go`** (new, the package's first non-test Go file):
  `//go:embed *.ail` exported as `dashboard_transforms.FS`. This one change serves the policy, the
  authority module (V6), `budget_checker` and `rig_admission`. The dashboard image comment at
  `root.go:17-19` describes the same gap. `NewForEmbeddedModule` writes the files into a
  content-addressed temp root (`os.TempDir()/ailang-embedded-<sha256>`), because
  `embed.New(basePath)` takes a directory and the stdlib already falls back to the copy built
  into the binary (`embed.go:59-61`). `AILANG_PROJECT_ROOT`, when set, still wins, so a
  developer's edited module is the one evaluated.
- **`auto_approval_facts.go`** gathers facts over REST with the token the landed-card sweep
  already uses (`config.GitHubToken()`, `daemon_landed_cards.go:66`). Each fact is fetched at
  most once per card per tick. A fact that cannot be read becomes a hold whose reason names it
  ("P2: check-runs unreadable: 502"). It is never defaulted.
  - **Rate budget.** Global facts (`dev` head and its CI run, the daily count, the P8 state) are
    read once per tick, not once per card. Per card the reads are the PR, its files, check-runs,
    statuses and issue comments: five calls.
  - At the measured queue (≤22 pending coordinator cards) that is about 110 calls per tick, so
    about 660 per hour at a 10-minute cadence. The authenticated limit is 5,000 per hour.
  - Cards whose `(head_sha, decision)` is unchanged since the last tick and whose hold is not
    time-dependent (P0, P4, P5, P7 on a stale head) skip the re-read. Only the PR head is checked.
  - A 403 or 429 rate-limit response holds every remaining card for this tick with that reason.
    It is never treated as a pass.
- **`daemon_auto_approval.go`** runs `sweepAutoApproval()` beside `sweepLandedCards()` at
  `daemon.go:565`: cloud only, same 10-minute throttle, the same SIGTERM-aware loop shape
  (`daemon_landed_cards.go:81-89`), agents restricted to `auto_approval.agents` and never an agent
  with `auto_merge: true` (V23).

Config, from the coordinator registry and never from a card or message:

```yaml
auto_approval:
  mode: shadow            # off | shadow | on   (default off)
  version: v1             # stamped into ApprovedBy; bump on any rule change
  agents: [design-doc-creator, sprint-planner]   # explicit allowlist
  max_per_day: 4
  deny_prefixes: ["internal/coordinator/", "cloudbuild", "tools/launchd/", ".github/workflows/", "missions/"]
  golden_prefix: ".stdlib-golden/"
  attestation_author: sunholo-voight-kampff
  attestation_max_age_hours: 72
  attestation_key_secret: triage-attestation-hmac   # Secret Manager name only
  enabled_since: 2026-11-01T00:00:00Z               # P8 window start; a human bumps it to re-arm
```

`AILANG_AUTO_APPROVAL=off` is the break-glass override, registered in the `internal/config`
Registry (the forbidigo rule). It wins over `mode` and needs only `gcloud run services update`,
not a config push. A config change reaches prod only on a multivac prod-branch push, plus a
revision roll for the registry to reload.

### 3. Dispatch after merge (H1, H2)

New approval status **`approved_awaiting_merge`**. The task stays `pending_approval`, because
that status is what the landed-card sweep lists.

```
pending ──policy Approve──▶ approved_awaiting_merge ──PR merged (sweep)──▶ approved  (+handoffs fired)
   ▲                              │
   │                              ├─PR closed unmerged──▶ rejected   (existing DecideClosedCard semantics)
   └──────merge refused / 409 / ──┘   (back to pending, Hold reason recorded, merge not retried this head)
          not merged in 24h
```

1. **Approve (policy).** A new store method, `MarkApprovalAwaitingMerge(taskID, approvedBy,
   headSHA)`, does a conditional write `pending → approved_awaiting_merge` in SQLite and
   Firestore. Nothing is dispatched. Boot recovery ignores the card (V13), and so does the
   stranded sweep (V14).
2. **Merge.** REST `PUT /repos/{r}/pulls/{n}/merge` with `merge_method: squash`,
   `sha: <attested head>`, `commit_title: <PR title>`, and `commit_message` naming the approver
   (§4). The `sha` pin is the TOCTOU guard: a push after the attestation fails with 409, and
   nothing unattested merges. Then `DELETE /git/refs/heads/coordinator/<task>`. This does not use `--auto`.
   P2 and P3 already require all checks green and `clean`, so there is nothing for auto-merge
   to wait for, and GitHub's `enablePullRequestAutoMerge` refuses a PR that is already clean.
   The image also has no `gh` (V4).
3. **Dispatch.** `DecideLandedCard` / `DecideClosedCard` (`pr_landed.go:23,49`) accept
   `approved_awaiting_merge` as well as `pending`. On MERGED, the landed path resolves the card to
   `approved` through `ProcessApprovalRequest`, keeping the policy approver
   (`policy:v1+attestation:… merged #N`) instead of `pr-merge #N (unknown)`, and fires the
   handoffs. `resolveApprovalByTask` (`store_sqlite_approvals.go:190`, Firestore `:266`) accepts
   that from-state for this transition only. After a successful synchronous merge, the policy
   sweep calls the same landed-card handler for that one task immediately, so handoff latency
   is one tick, not the sweep's up-to-an-hour gap at scale-to-zero (`daemon_landed_cards.go:20-23`).
   This is one implementation called from two places.
4. **Failure.** A merge refusal (405 or 409), a closed PR, or 24h without a merge returns the card
   to `pending` with the reason recorded in an `auto_approval_eval` event. The same head is not
   re-approved: a hold keyed on `(task, head)` remains until the head changes.
5. **Visibility.** `ListPendingApprovals` filters `status = 'pending'` (`store_sqlite_approvals.go:91`; Firestore `coordinator_approvals.go:96`), so an awaiting card leaves the human queue. That is intended, because it must not be approved twice, but it must not become invisible. A new `ListApprovalsAwaitingMerge` (both stores) feeds an "Awaiting merge" section in the dashboard's approvals view (`/api/approvals` gains an `awaiting_merge` array; `ui/` renders it read-only with PR link, approver and age) and in the `approvals` CLI. `approvals --json` gains an `awaiting_merge` list and, for each pending row, a
   `policy` object `{version, decision, clause, reason, head_sha, evaluated_at}`. The triage
   loop reads this, which is how the shadow ledger gets the policy's view (§6).

**D5, human approvals.** The same `RequestMerge` path, minus the policy, serves the CLI and the
dashboard for cloud tasks. They mark the card awaiting-merge and merge through REST in-process.
The dashboard then stops orphaning PRs (V11), and both paths stop dispatching before the merge.
The CLI's `reconcileTaskPR` hot path becomes a call to it. Recommended. If Mark prefers to keep
human paths untouched, M2 ships policy-only and V11 becomes its own bug.

### 4. Authority (H3) without a lockout, and audit (H5)

**Order matters.** M1 embeds the authority module, so `decide` evaluates from any cwd (V6).
M3 enforces only after an acceptance run shows the attended path granted outside the repo.

**What enforcement means here.** The authority rule is a routing rule for a cooperating fleet
(V8). Any session can export any variable. The control that actually matters for unattended
approvals is that **only the coordinator daemon approves unattended**, server-side, as
`policy:<v>`. The CLI check stops a mission role or a dry-out rung from approving by mistake;
it does not stop a hostile one.

M3 enforcement matrix (approve; reject per D6):

| Session | Today | After M3 |
|---|---|---|
| Spawned role (`MISSION_ROLE` = designer, planner, executor, evaluator) | approves | **refused** |
| Mission controller on an untrusted rung, or an unnamed controller | approves | **refused** |
| Coordinator task (`AILANG_TASK_ID` set) without a grant | approves | **refused** |
| Trusted controller (fable, astra, opus) | approves | approves (reviewable rows only, `coversRow`) |
| `AILANG_APPROVAL_CONTROLLER=1`, any cwd | approves | approves |
| Interactive TTY, no `--yes`, no identity (Mark at a terminal) | approves | approves (the y/N prompt is the human act) |
| Non-interactive or `--yes`, no identity, no grant | approves | **refused**, with the one-line remedy |
| `AILANG_APPROVAL_POLICY=always` | approves | approves (break-glass, unchanged) |
| Dashboard | approves | approves. **Not touched** (V10) |

Lockout guards: (a) M3 ships with `AILANG_APPROVAL_ENFORCE=warn` for 7 days. It prints and
records "WOULD REFUSE: <reason>" on the approval event and refuses nothing. (b) `ailang
coordinator approve <id> --dry-run` prints the verdict for this session without acting, so the
attended path can be checked before a real card. (c) The refusal names the exact remedy.
(d) The dashboard is never gated, so a working path always exists. (e) Acceptance (M3) tests
both directions on both paths. Before enforcement, check whether the Discord ping's payload
command is run from a TTY or by an agent, and add a matrix row if it is neither.

**Audit.**
- Policy: `ApprovedBy = policy:<version>+attestation:<judge>`. After the merge lands it gets
  ` merged #N` appended. The `auto_approval_eval` event records the attestation comment URL.
- Merge commit body, on every path: `Approved by: <ApprovedBy>` plus `Coordinator task:
  <id>` plus, for policy approvals, `Attestation: <comment URL> (head <sha>)`, plus `DecidePR`'s
  scope reason.
- The dashboard records the signed-in identity where the request carries one, and
  `dashboard-user` otherwise. M2 reads the request's auth header for this. If none is
  available, the merge body states "dashboard (identity not recorded)" rather than inventing one.

### 5. The attestation contract (the triage loop writes it, the coordinator reads it)

One PR comment (`POST /repos/{r}/issues/{n}/comments`) containing exactly one block:

```
<!-- ailang-triage-attestation v1
verdict: VERIFIED
repo: sunholo-data/ailang
pr: 1747
head_sha: 3f2c…(40 lowercase hex)
dev_sha: 9a1b…(40 hex; origin/dev the verifier read)
verifier: codex:gpt-6-luna
judge: claude:claude-sonnet-5-5
evidence: https://github.com/sunholo-data/ailang/issues/<bookkeeping>#issuecomment-<id>
issued_at: 2026-10-15T12:00:00Z
sig: hmac-sha256:<64 lowercase hex>
-->
Triage: VERIFIED at 3f2c… by codex:gpt-6-luna, judged by claude:claude-sonnet-5-5. [evidence](…)
```

Parser rules. Every failure is a P7 hold whose reason names the rule:

1. The fields appear in exactly this order, one per line, `key: value`, with no extra keys.
   `verdict` is one of the parent's verdict set. Only `VERIFIED` can pass P7.
2. `sig` = HMAC-SHA256(key, the bytes from `verdict:` through the `issued_at` line inclusive,
   lines joined by `\n`, no trailing newline), compared in constant time. The key comes from Secret
   Manager (`attestation_key_secret`). The rig reads the same secret. The executor job SAs must not
   (V26: confirmed in terraform; live re-check in M4 step 0).
3. The comment's `user.login` equals `attestation_author`. Necessary, not sufficient (V20).
4. `created_at == updated_at`: an edited comment is invalid, because anyone with the token can
   edit it.
5. `repo`/`pr` match the PR being evaluated, and `head_sha` equals the PR's current head. Any new
   push therefore invalidates it, as D-TRIAGE-9 requires.
6. Age is `now − created_at` from GitHub's timestamp, not `issued_at`. `issued_at` must be within
   one hour of `created_at`.
7. When several comments qualify, the newest valid one for the current head wins. A later
   `<!-- ailang-triage-revoke v1 … head_sha … sig -->` block, signed the same way, revokes it.
8. `verifier != judge`, enforced in AILANG. Vendor-disjointness is the triage loop's routing
   guarantee (charter "Routing policy"). The signature attests that the loop, and so its routing, produced it.

The format, the canonical signing bytes and a Go test vector go in
`docs/internal/triage-attestation-v1.md`, which both writers link to. The triage loop's
`gate-3-triage.md` writer uses the same vector.

### 6. Rollout: off → shadow → on, plus P8

1. **Shadow (M4).** `mode: shadow`: full fact gathering and `decide`, writing an
   `auto_approval_eval` task event only when `(head_sha, decision, clause)` changes, and filling the
   `policy` object in `approvals --json`. It approves, merges and dispatches nothing. The triage
   loop copies `policy` into its shadow ledger next to its own verdict and the item's eventual
   outcome (human approve, reject, close, revert).
2. **Scoring gate (M5 entry).** Scoring covers at least 14 consecutive days and at least 10
   would-approve decisions; with fewer, the window extends, because no data is not evidence.
   Pass requires zero would-approves that were later human-rejected, closed unmerged, or
   reverted, and zero would-approves contradicted by a triage verdict at the same head. The
   triage digest reports the counts, and Mark rules on the promotion. It is recorded in the
   charter ledger.
3. **On (M5).** `mode: on`, `enabled_since` set to the promotion time.
4. **P8 auto-off. Bounded, and it fails closed** (quorum round 0, gemini-4-argon).
   - Only **unsettled** merges are queried. Each `auto_approval_merge` event (merge SHA) is read
     until its `CI` run on `dev` concludes. The conclusion is then written once as an
     `auto_approval_merge_ci` task event, and that SHA is never queried again. Per tick this is at
     most the merges still awaiting CI, which is bounded by `max_per_day` (≤4) and in practice is
     0–1. It is not every merge since `enabled_since`. The trip state is then derived from the
     settled events, with no REST calls.
   - **Fail closed.** `tripped = true`, with the reason, when:
     - a settled conclusion is anything other than SUCCESS;
     - the CI status for an unsettled merge SHA cannot be read (a REST error, a 4xx/5xx, or an
       undecodable body);
     - an unsettled merge SHA has no concluded `CI` run 6h after its merge, because CI that never
       ran is not proof of green.
   - When tripped, P8 holds everything and sends one `ailang messages send user` notice (Discord)
     naming the PR, the SHA and the reason. The notice is deduplicated on (SHA, reason).
   - Re-arming is a human bumping `enabled_since` in config, a git-audited action. Re-arming does
     not erase the settled events: they stay as the audit record.
   - The read-failure trip is deliberately conservative. A GitHub outage switches auto-approval off
     until a human re-arms it. While GitHub is unreachable P1 and P2 cannot be read either, so
     nothing could be approved anyway.
5. **Kill switch.** `AILANG_AUTO_APPROVAL=off` on the service is immediate on the next revision.
   `mode: off` is the durable form.

## Conflict Surface (shared machinery this changes)

The parser and typechecker are not touched. The rows below override shared coordinator
behaviour, so quorum trigger 2 fires.

| Shared machinery | Decision | Other users who must keep working |
|---|---|---|
| Approval status vocabulary (`pending/approved/rejected`) | **Override**: add `approved_awaiting_merge` | `ListPendingApprovals` (dashboard queue, `approvals` CLI), boot recovery (V13), stranded sweep (V14), `ListResolvedApprovals` (history UI). M2 greps every `status` comparison in `internal/coordinator`, `internal/storage/firestore`, `internal/server` and `ui/` and records each in the sprint plan |
| `DecideLandedCard` / `DecideClosedCard` | **Override**: accept the new from-state | Hand-merged PRs (pending cards) unchanged; `prs --landed --apply` unchanged |
| `resolveApprovalByTask` WHERE `status='pending'` | **Override** for the landed transition only | Every other resolve keeps refusing non-pending cards |
| `DecidePR` | **Reuse**: extract `CheckPRScope`, behaviour identical (existing tests unchanged) | `coordinator prs` sweep, `reconcileTaskPR` |
| `embed` module resolution | **Extend**: embedded FS fallback after `AILANG_PROJECT_ROOT` and cwd | `ailang budget`, dashboard bridge, riggate. Their resolution order is unchanged when a source tree is present |
| Authority (`resolveApprovalAuthority`) | **Override**: the CLI enforces it (M3) | `approvals --json` labels unchanged; mission-control's unattended approve path (trusted controller) unchanged; dashboard untouched |
| Landed-card `ApprovedBy` | **Override** for policy cards only | `pr-merge #N (…)` kept for hand-merges |

## Milestones

**M1: Embed the decision modules and add the policy engine (2 days).**
- `internal/dashboard_transforms/embed.go`, `embed.NewForEmbeddedModule`; the authority engine
  switches to it.
- `auto_approval_policy.ail` with inline tests; `auto_approval_engine.go`; a Go benchmark of load
  and call for the policy and the authority module (V18 gap).
- Acceptance: `ailang test internal/dashboard_transforms/auto_approval_policy.ail` passes. From a
  directory outside any checkout, with `AILANG_PROJECT_ROOT` unset,
  `AILANG_APPROVAL_CONTROLLER=1 ailang coordinator approvals --json --remote local` reports
  `"authority": true` (V6 inverted; mutation: delete the embed fallback and watch it go false).
  The Go engine test decodes `Approve` and every `Hold` clause. `make check-boundaries` passes.

**M2: Dispatch after merge (2.5 days).**
- Step 0: measure V11 live, listing open PRs whose card is `resolved_by=dashboard-user`.
- `approved_awaiting_merge` in both stores; `MarkApprovalAwaitingMerge`; `RequestMerge`
  (REST, SHA pin, branch delete, approver in the body); widened `DecideLandedCard` and
  `DecideClosedCard`; immediate landed handling after a synchronous merge; 24h timeout back to pending.
- D5 wiring for the CLI and dashboard, if ruled.
- Acceptance: a fake GitHub (`githubAPIBase`, as in the existing sweep tests) shows that (a)
  after approve, zero handoff messages exist before the merge and exactly one after; (b) a
  `triggerMissedHandoffs` run while awaiting dispatches nothing (mutation: use status `approved`
  and watch boot recovery fire); (c) a 409 on SHA mismatch returns the card to pending with the
  reason; (d) a closed PR rejects; (e) the merge body contains `Approved by:`.

**M3: Authority enforcement, warn then enforce (1.5 days).**
- `AILANG_APPROVAL_ENFORCE=off|warn|on` (Registry), the matrix in §4, and `--dry-run`.
- Acceptance, in both directions on both paths: the attended CLI (`AILANG_APPROVAL_CONTROLLER=1`,
  run from the repo **and** from `/tmp`) approves a test card on the dev plane. A plain TTY approves
  via the prompt. `MISSION_ROLE=executor` is refused. `CONTROLLER_ID=pi:ollama/glm…` with
  `MISSION_CONTROL_ACTIVE=1` is refused. `--yes` with no identity is refused with the remedy.
  The dashboard approve on the dev plane succeeds with enforcement `on`. In `warn` mode, all the
  refused cases succeed and record "WOULD REFUSE". Seven days of `warn` show no WOULD-REFUSE on
  any human approval before `on`.

**M4: Facts, attestation parser, shadow mode (2 days).**
- Step 0: re-check V26 against live prod IAM, confirming no executor-lane SA can read the HMAC secret. If one can, stop and take D3's other option.
- `auto_approval_facts.go`, the §5 parser with the shared test vector,
  `docs/internal/triage-attestation-v1.md`, `sweepAutoApproval` in `shadow` mode, and the
  `policy` object in `approvals --json`.
- Acceptance: parser tests for each rule 1–8, including a valid block on an edited comment (rejected), a
  valid block by another login (rejected), a stale head (rejected), a bad signature (rejected),
  and the test vector accepted. A shadow run on the dev plane writes `auto_approval_eval` events
  and changes no card status (asserted by diffing card statuses before and after).

**M5: On (half a day, after the scoring gate).**
- `mode: on`, P8 trip and notice, runbook (`docs/internal/coordinator-auto-approval.md`).
- Acceptance: on the dev plane, an attested green PR is approved, merged, then handed off, in
  that order (event timestamps). A forced red CI on a policy merge SHA trips P8, the next card
  holds with `P8Tripped`, and one Discord notice is sent. Fail-closed: a fake GitHub returning 502 for an unsettled merge SHA trips P8. So does a merge SHA with no concluded CI 6h after the merge. A settled SHA is not re-queried on the next tick (call count asserted). `AILANG_AUTO_APPROVAL=off` holds
  everything on the next tick.

## Testing Strategy

The AILANG inline tests cover the clauses, and Go table tests cover the shell, fact
normalisation, the parser and the state machine. Fake-GitHub sweep tests follow the existing
`settledPRLookup` / `githubAPIBase` seams (`daemon_landed_cards.go:47-53`). Each new guard gets a
mutation test (revert the guard, watch its test fail), per the repo's practice. Both stores are
tested: SQLite in unit tests, plus a Firestore emulator test for the conditional
`pending → approved_awaiting_merge` write.

## Non-Goals

- Dependabot PRs (own workflow), `fleet/*` and other missions' PRs, non-coordinator branches.
- Reject. It stays human: remote reject is final (H6), and the policy never rejects. A `Hold` is
  not a reject.
- Prod promotion; anything tag-gated.
- Agents with `auto_merge` or `auto_merge_code` (V23); agents not on the allowlist.
- Rebasing or updating BEHIND branches (D7). For v1 they hold.
- Making the CLI authority check a security boundary (V8).

## Deferred Decisions (agent latitude)

- REST endpoint details (check-runs versus GraphQL `statusCheckRollup`), as long as P2 reads every
  check.
- Event payload field names, and the temp-root naming for embedded modules.
- Whether `auto_approval_eval` dedupe also keys on the policy version (recommended).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Forged attestation from an executor holding the bot token | High | HMAC (D3). V26 is confirmed in terraform and re-checked live at M4 step 0. The author check alone is never trusted |
| A new approval status missed by some reader, which then misreports or re-dispatches | High | M2's status-comparison audit; boot-recovery mutation test (M2 acceptance b) |
| Lockout of human approvals when M3 enforces | High | M1 first (V6); 7-day `warn`; `--dry-run`; dashboard untouched; break-glass `always` |
| Policy merges a change that turns `dev` red, and every merge redeploys dev (H8) | Med | P1 + P2 + P3 + SHA pin; P8 trips on the first red; cap of 4/day |
| P3 holds most PRs because they are BEHIND (`strict: false`) | Med | Measured in shadow. If the would-approve count is under 10 in 14 days because of P3, D7 comes back to Mark with the number |
| Scale-to-zero gaps delay evaluation (up to ~90 min) | Low | Acceptable. Nothing is lost, the next tick re-evaluates, and the immediate landed handling removes the second gap |
| Embedded module drifts from the repo copy | Low | It is the same file (`go:embed`), and `AILANG_PROJECT_ROOT` still overrides for development |

## Quorum

Triggers fired: **1** (design-freeze items D3, D5, D7) and **2** (overrides shared
machinery: approval status vocabulary, the landed-card sweep, authority enforcement).

`ailang design-quorum … --max-cost-usd 0.30 --author claude:claude-opus-5-5`, 2026-10-15.
Artifacts are in `.ailang/state/mission-quorum/m-coordinator-auto-approval-policy-*.json`, which
are local and not committed.

**Seats.** gemini-4-argon, oc-glm-5-3 and oc-kimi-k3 were seated, with gpt6-1-sol in reserve.
claude-sonnet sat out because it is the author's vendor. **Only gemini-4-argon was reachable in
either round**:
- oc-glm-5-3 and oc-kimi-k3 returned `429`, "Pro weekly limit" (Ollama Cloud);
- gpt6-1-sol returned `429`, "no credits remaining" (OpenAI API org).

Both verdicts below are therefore one off-Anthropic reviewer plus the in-session controller.
That is N−2, not a full quorum.

- **Round 0: BLOCKED** ($0.034). gemini-4-argon rejected. Its strongest objection: P8 queried CI for
  *every* policy merge since `enabled_since` on every tick, an unbounded REST amplifier with no
  fail-closed rule for an unreadable status. **Correct and accepted.** §6.4 now queries only
  merges whose CI is unsettled (≤ `max_per_day`) and records each conclusion once. It trips on an
  unreadable status or on no conclusion 6h after the merge. M5 acceptance asserts all three
  behaviours. A per-tick REST budget was added to §2.
- **Round 1: PROCEED** ($0.033). gemini-4-argon passed. Its strongest objection was that V26
  (executor SAs and the HMAC secret) was left PENDING; its catch was that awaiting cards would vanish from
  the dashboard queue. Both were addressed:
  - V26 is now confirmed from the multivac terraform (per-secret agent-lane bindings, M-SEC2), with
    a live re-check at M4 step 0;
  - §3.5 adds an "Awaiting merge" section (store method, `/api/approvals`, UI, CLI).
- **Recommendation.** Before M3 or M4 starts, re-run the quorum once the OpenAI credit or Ollama
  weekly quota is back, so that a second vendor reviews the authority matrix (§4) and the
  attestation contract (§5).

## Open questions for Mark

1. **D3**: an HMAC key in Secret Manager, readable by the coordinator and the rig only
   (recommended, one secret), or a second GitHub identity for the triage loop?
2. **D5**: move human cloud approvals (CLI and dashboard) to dispatch-after-merge in M2
   (recommended; fixes the dashboard orphan), or policy only?
3. **D6**: should M3 also gate `reject` (final and destructive), or approve only (recommended for v1)?
4. **D7**: hold BEHIND PRs for v1 (recommended), or let the coordinator `update-branch`, which forces a
   re-attestation every time `dev` moves?
5. Initial `agents` allowlist: `design-doc-creator` and `sprint-planner` (docs-only diffs), with
   executors added later on evidence?

## Related Documents

- [m-triage-mission.md](m-triage-mission.md): parent; hazards H1–H11, Part B
- [triage-mission.md](../triage-mission.md): charter and ledger (D-TRIAGE-7/8/9)
- [HANDOVER-message-plane-approvals-2026-09-23.md](HANDOVER-message-plane-approvals-2026-09-23.md): the 16-day lockout
- [v0_52_6/m-auto-merge-code-prs.md](v0_52_6/m-auto-merge-code-prs.md) (0.43): distinct. It is an in-job merge at completion with a non-author approver, and this policy skips those agents (V23)
- [ailang-core-triage/coordinator-design-pr-auto-merge.md](ailang-core-triage/coordinator-design-pr-auto-merge.md) (0.47): distinct. It is a per-agent `auto_merge` config proposal for docs PRs, with no card-time policy
- [m-rig-gpu-admission-gateway.md](m-rig-gpu-admission-gateway.md): the second Go→AILANG decision precedent (`internal/riggate/policy.go`)
- [implemented/v0_7_0/m-coord-human-loop.md](../implemented/v0_7_0/m-coord-human-loop.md) (0.43), [implemented/v0_6_2/m-coord-approvalwatcher-observability.md](../implemented/v0_6_2/m-coord-approvalwatcher-observability.md) (0.44): history of the approval surface

---

**Document created**: 2026-10-15
**Last updated**: 2026-10-15
