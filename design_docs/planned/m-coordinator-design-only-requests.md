# M-COORDINATOR-DESIGN-ONLY-REQUESTS

**Status**: Implementation under verification
**Target**: next release after v0.37.0
**Priority**: P1
**Estimated**: one attended session
**Dependencies**: existing coordinator skill invocation and completion protocol

## Problem Statement

Daneel added six dedicated design inboxes with copied prompts. Mark requested reuse
of existing project inboxes, the repository's design skill, and a completion report.
The existing AILANG design agent invokes the right skill but its successor is
sprint-planner. Package inboxes select a maintenance prompt; website-builder pushes
main directly. A document request must not inherit those implementation behaviours.

## Goals

Reuse existing inbox and agent identities; invoke design-doc-creator; preserve
project/model/permission-tier configuration; finish on a task branch without a merge,
package publication or a successor task. Daneel observes the original request ID.

## High-Impact Decisions and Design Freeze

Mark authorized the existing-inbox/design-only architecture in the attended Daneel
session on 10 September 2026 and instructed its implementation.

- [x] Opt-in `design_requests: true` on existing registry entries. No new agents.
- [x] Transport stays `message_type=request`, with a top-level JSON
  `workflow=design-document-v1`. Request data cannot select repository or model.
- [x] Derive the reduced scope from persisted task Kind/Content at execution,
  completion and approval, including retries. Do not mutate the registry.
- [x] Invoke the repository's existing design-doc-creator skill; use installed
  shared skill when the project lacks one. Missing skill must be reported.
- [x] No automatic merge/direct push, package subdirectory automation, inherited
  session, or successor edges. Cap execution at twenty minutes without increasing
  a shorter configured timeout.
- [x] Successful document tasks are terminal, without a merge/handoff approval.
  Creating a document is the authorized result; approving a pipeline is a separate
  request. Dispatch still has SkipApproval=false so it uses a task branch.

## Solution Design and Implementation Plan

`AgentForTask` makes a shallow config copy, then replaces mutable workflow slices
and invoke settings; project, provider, model and WorkTier remain unchanged.
`BuildDirectiveFromConfig` uses the native skill directive builder plus fixed scope
instructions. Cloud dispatch uses that effective config for both prompt and push
parameters. Local execution applies it after package autonomy adjustment, before
execution. Clearing Subdirectory suppresses deterministic package bump/publication.
Finalization produces a terminal result; automatic and later approval handlers
remove successor edges for the persisted design scope. Ordinary requests retain
registry behaviour. Inbox discovery exposes the opt-in for clients to fail closed.

Daneel keeps an explicit repository/inbox/expected-agent allowlist, persists request
identity, and checks correlated completions. Existing package aliases route to their
actual package inboxes; an unqualified packages request must select a package.
Retired Daneel routes are read-only completion aliases for already-dispatched jobs.

## Examples

A request to `design-doc-creator` with the workflow above uses the AILANG repository
skill and ends with DESIGN_DOC_PATH. The same envelope at `pkg:sunholo/email` uses
that inbox's email-parse repository and cannot publish a package. An ordinary request
or feedback message still uses the configured maintenance route.

## Verification Log

| Claim | Evidence |
|---|---|
| Skill invocations are directives, with explicit markers | Read stage_execution.go buildSkillDirectiveWithConfig; existing TestBuildDirectiveFromConfig suite |
| Existing type routing selects template files, not per-request handoff policy | Read InvokeConfig.ResolveTemplateForType, task_finalize.go and approval_handoff.go |
| Direct push derives from SkipApproval and MergeBranch | Read daemon_tasks_exec.go DispatchParams construction |
| Local package publication derives from Subdirectory | Read daemon_tasks_exec_run.go success branch |
| Task Kind and Content are persisted | TaskRecord and existing SQLite/Firestore mappings retain both fields; no new task storage field |
| Bundled shared skill exists | ailang_bootstrap/skills/design-doc-creator/SKILL.md; cloud executor resolves /plugins/ailang_bootstrap |
| Existing inbox completion identity differs for packages | Canonical registry: inbox pkg:sunholo/email, agent pkg-sunholo-email |

## Testing Strategy and Success Criteria

- [x] Design config selects native skill, preserves route/model, suppresses direct
  push and successor/session/package automation, and leaves the registry unchanged.
- [x] Positive controls: ordinary requests retain maintenance prompts and automatic
  or approval-gated successor behaviour.
- [x] Completion, replay and later approval cannot release a design successor.
- [ ] Full coordinator and CLI regression suites; actual shared-route request.
- [ ] Verify only Markdown design artifacts and the skill read in execution evidence.
- [ ] Daneel reports the correlated result once, in its standard styled email.

## Axiom Compliance

| Axiom | Score | Reason |
|---|---:|---|
| A1 Determinism | +1 | Fixed workflow discriminator and configured destination |
| A2 Replayability | +1 | Scope derived from persisted task data |
| A3 Effect Legibility | +1 | Document work excludes implementation automation |
| A4 Explicit Authority | +1 | Registry opt-in; request only reduces authority |
| A5 Bounded Verification | +1 | Completion and ordinary controls tested locally |
| A6 Safe Concurrency | +1 | Per-task config copy; shared registry unchanged |
| A7 Machines First | +1 | Explicit capability and correlated completion |
| A8 Minimal Syntax | 0 | Existing JSON transport, no language syntax changes |
| A9 Cost Visibility | 0 | Existing task cost accounting retained |
| A10 Composability | +1 | Existing inboxes and skill pipeline reused |
| A11 Structured Failure | 0 | Existing coordinator failure reporting retained |
| A12 System Boundary | +1 | Explicit client/coordinator contract |

Net +9; no hard violations of A1/A3/A4/A7.

## Non-Goals, Deferred Decisions and Risks

No language/compiler changes, arbitrary skill execution, automatic implementation,
or new permissions. Executor file scope remains an instruction, not a filesystem
sandbox; Daneel checks reported paths and flags unexpected changes. Authentication
of a message sender is not strengthened here. Future promotion of a design into a
sprint must be a separate request. No implementation decision is deferred.

## Related Documents

The scaffold's search returned transcript unification, agent orchestration, provider
failover and microRAG context. Reading their problem statements shows different
workstreams (history, language-level agent effects, model routing and retrieval),
not this inbox policy. This work reuses the completion/handoff mechanisms rather
than redesigning those systems. The high displayed search ranks are not evidence
of feature coverage; source and behavioural tests above establish the actual gap.
