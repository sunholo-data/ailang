# M-DOCS-MISSION-WORK-ITEM-GUIDE

**Status:** Proposed canary task; awaiting Mark's scope approval. No execution authority recorded.
**Created:** 2026-09-07. **Target:** mission-iteration branch, release unassigned.
**Priority:** P1. **Estimate:** one bounded documentation edit, about 150–220 lines.
**Dependency:** M-MISSION-ITERATION implementation at `77dc7287e7bce225e64062452598f0694ac361fc`.
**Proposed work ID:** `docs-canary-work-item-guide-1`.

## Problem and goal

The new `docs/docs/guides/mission-iteration.md` describes the runtime and lists input
fields, but its commands refer to an unexplained `/absolute/reviewed-work-item.json`.
The complete JSON example is an internal test fixture containing dummy commits,
hashes, repository and model identities. A project operator needs to understand how
to replace those values with real, approved Git artifacts before the dry-run works.

Deliver one worked, source-accurate guide that helps an operator prepare that input.
This is useful project documentation and a bounded candidate for demonstrating the
new runtime. It extends the M5 guide; it does not redesign the runtime or reopen M5.

## Proposed scope

Only product path: `docs/docs/guides/mission-iteration.md` (exact path, no directory wildcard).
Expand its existing “Review and run a work item” section with:

1. One complete annotated-by-prose JSON example for imported designer/planner
   prerequisites followed by executor/evaluator. Use the existing test fixture as
   the structural reference and label every illustrative value that requires replacement.
2. A short sequence to obtain full Git revisions, normalized origin, artifact SHA-256,
   authority document hash and stable decision locator. Hash committed content using
   `git show REV:path`; explain that Git blob object IDs and SHA-256 content hashes differ.
3. Explain how authority binds each prerequisite digest; approval documents must be
   present unchanged at the chosen base. Recording a locator does not create approval.
4. Explain registry author identities and evaluator independence; a transport change
   alone does not make the same model vendor an independent judge.
5. Show dry-run, status, and the next action after invalid input, quota wait and
   reconciliation. Keep placeholder examples clearly distinct from an executable command.

Preserve the existing lifecycle, cancellation, exit-code and hermetic-example sections.
No compiler/runtime/CLI/config/scheduler changes, automatic registration, or publication.
Do not add a generator script, new dependency or copied machine credentials.

## Acceptance criteria to freeze in the eventual WorkItem

- `complete-example`: the complete JSON contains all four roles, explicit criteria,
  verification, scope, authority and positive cumulative/stage limits; its structure
  agrees with `internal/mission/iteration/spec.go` and the existing fixture.
- `real-artifact-inputs`: the guide explains replacement of every illustrative
  revision/hash/model/repository/path with actual committed, approved inputs.
- `honest-authority`: artifact content SHA-256, approval SHA-256, locator and bound
  digest are distinguished; instructions never manufacture human approval.
- `bounded-operation`: dry-run is described as local validation, with no inference;
  quota/reconciliation outcomes and the absence of automatic merge are accurate.
- `scope-preserved`: only the allowed Markdown file changes, existing sections remain,
  and an independent evaluator reviews the exact candidate commit against these IDs.

## Validation approach

Use the candidate binary and source at the frozen base. Read the JSON against the
strict spec validator and inspect authority behavior in `authority.go`. The independent
evaluator must check every example field and command against source/CLI help, rather
than accept a word-count or heading check as proof of correctness.

Freeze `git diff --check BASE` as a local hard check. Before activation, the sprint
plan must choose and verify a bounded structural check for the extracted JSON against
existing runtime validation, with real temporary Git artifacts if a full dry-run is
claimed. A Markdown syntax check alone cannot establish a valid runtime input.
Do not run `npm run build` as a supposedly read-only check: its `sync-all` prerequisite
can modify source documentation. Choose a direct docs renderer only after confirming
its local prerequisites and output behavior. No new runtime tests are proposed.

## Canary placement and resource proposal

Use the reviewed candidate binary by absolute path. Base the task on this sprint's
history after the approved design, plan and authority references have been committed;
freeze the resulting full base hash in the input. A dedicated clean source worktree
under `/private/tmp/ailang-docs-canary/` avoids changing the older live Docs checkout.
Use a reviewed canary copy of the Docs registry entry pointing to that source, retaining
mission ID `docs` and expected origin `sunholo-data/ailang`. Do not edit fleet registry.

Proposed executor: existing `claude-sonnet-5` registry entry via Claude CLI.
Proposed evaluator: existing `pi-or-deepseek-v4-flash` via Pi/OpenRouter.
No fallback candidates. The design author for this document is this attended Codex
session; freeze its actual registry identity when preparing prerequisite provenance.
Importing later planning must account for that planner's actual identity too.
The current registry values are source policy, not a live pricing verification.

Keep the existing maximum 60 minutes, 100,000 fresh tokens and $5 metered guard;
executor 30 minutes/70,000 tokens/$3, evaluator 20 minutes/30,000 tokens/$2.
Admission must pass at execution time. Unknown metered usage prevents acceptance.
Activate only after legacy Docs exits and ownership is rechecked, using the existing
canary packet's saved binding, disable-marker and rollback procedure.

## Verification log and related work

- Read the full guide, fixture, `spec.go` and `authority.go` at `ab6931696`:
  confirmed the guide lacks the complete JSON present in the internal fixture.
- Searched mission designs and guides for work-item examples: existing runtime design
  defines the schema; M5 delivered the overview. This task fills its operator walkthrough.
- Read `docs/package.json`: build invokes `sync-all` before rendering.
- Candidate binary SHA-256 verified unchanged:
  `1a062252d21d2e055a1b0552cfc836a9ffe634ddf27916484f81a5691c3fd2e8`.
- At 19:53–19:54 UTC, legacy Docs was running (launchd PID 11260, heartbeat gate-3),
  and its previous completion named docs-12 next. Treat that queue item as unavailable.
- Codex provider observation at 19:54 UTC: weekly usage 50%, allowance 10%, `over`.
  This supports proposing another explicit route; it does not authorize bypassing admission.
- No installed runtime binding file was found at inspection time. Recheck before activation.

Related: [iteration sprint](m-mission-iteration-sprint-plan.md),
[runtime contract](m-mission-runtime-contract.md),
[activation packet](../verification/mission-iteration/canary-activation.md).

## Axiom assessment

Documentation-only impact: A4 Explicit Authority +1 (explains content-bound approval),
A7 Machines First +1 (complete structured input), A9 Cost Visibility +1 (explicit limits).
A1 Determinism, A2 Replayability, A3 Effect Legibility, A5 Bounded Verification,
A6 Safe Concurrency, A8 Minimal Syntax, A10 Composability, A11 Structured Failure,
A12 System Boundary: 0 (no runtime changes). Net +3; no hard violations.

## Approval boundary

This document is a proposal, not an approved prerequisite or an execution request.
After Mark approves this scope, use sprint-planner to freeze the checks and remaining
stages, then record actual execution authorization before building the bound input.
M6 remains partial until an unowned approved task and validated activation packet exist.
