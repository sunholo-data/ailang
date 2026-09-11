# M-SCOPED-DESIGN-REQUEST-RECEIPTS: scoped multi-project design requests and completion receipts

**Status:** Planned — **integration verification document.** This doc records and tightens an
existing integration path (Daneel → coordinator → design-doc-creator → completion receipt);
it is deliberately concise and proposes no new subsystem.
**Priority:** P1
**Created:** 2026-09-10
**Related:** [m-message-plane-trust.md](m-message-plane-trust.md) (send-means-ran),
[m-coordinator-execution-trust.md](m-coordinator-execution-trust.md),
[message-plane-topology.md](../../docs/internal/message-plane-topology.md),
`internal/coordinator/pubsub_completion_handler.go`,
`internal/coordinator/agent_registry.go` (`OutputMarkers`, `DESIGN_DOC_PATH:`),
`internal/coordinator/daemon_tasks_budget.go`.

---

## 1. Problem

Mark requests design documents for projects already registered in the coordinator. Daneel (a
virtual employee) is the consumer: it must pick only institution-configured project routes,
dispatch a fixed document-only workflow, retain the request message ID, and report the artifact
to the requester once a *correlated* completion identifies it. Ordinary prose documents are
drafted locally and uploaded via Daneel's existing Drive path — that path is out of scope here.

Today each seam exists individually (registry, dispatch, `DESIGN_DOC_PATH:` marker extraction,
completion notification with `CorrelationID: task.MessageID`), but the *end-to-end contract* for
a scoped, document-only, non-implementing request is unwritten — and unwritten contracts are how
"a message was sent" stops meaning "a job ran and reported back" (M-MESSAGE-PLANE-TRUST).

## 2. Current behaviour (verified at HEAD)

- **Trusted mapping.** The coordinator routes by a configured agent registry
  (`internal/coordinator/agent_registry.go`); the `design-doc-creator` agent defaults to output
  marker `DESIGN_DOC_PATH:` (`agent_registry.go:565`). Inbox↔agent mapping is explicit
  (`InboxForAgent`, e.g. `pkg-sunholo-auth` → `pkg:sunholo/auth`).
- **Dispatch.** Tasks are claimed by a registered agent, executed on a task branch
  (`coordinator/task-<id>`), and finalized through one path (`FinalizeTaskCompletion`,
  M-COMPLETION-PATH-PARITY).
- **Completion receipt.** `CompletionHandler.handleCompletion`
  (`pubsub_completion_handler.go`) is idempotent (unknown task → ignore; terminal status → skip),
  posts an inbox `completion` message carrying `task_id`, `agent_id`, `status`, `branch_name`,
  `error_msg`, `changed_files`, and `CorrelationID: task.MessageID`. It explicitly does **not**
  reply to the original requester — consumers correlate home via `correlation_id`.
- **Changed files.** `TaskCompletion.ChangedFiles` is published and forwarded into the
  notification payload; marker parsing (`stage_execution.go`) tolerates both
  `DESIGN_DOC_PATH: path` and `**DESIGN_DOC_PATH**: \`path\`` forms.
- **Budgets.** `checkBudgetBeforeExecution` enforces configured per-provider daily and per-task
  max cost; absent config means *no* enforcement (a known posture, not a silent fallback).

## 3. Proposed design

A request/receipt contract with the following rules, all satisfiable with existing mechanisms:

1. **Trusted project/agent mapping.** Daneel resolves a requested project only through the
   institution-configured route table (coordinator registry). Unknown or unregistered project →
   reject locally, never guess a route. The route fixes the workflow: document-only.
2. **Request data ≠ authority.** The incoming request payload (topic, quoted instructions) is
   source material. The directive (institution-owned) fixes the bounds: Markdown under
   `design_docs/` only, on the task branch, no merge/deploy/publish/messages. The dispatched
   prompt must restate these bounds so the executor cannot inherit authority from payload text.
3. **No automatic implementation stage.** The design-doc pipeline stops at the document. Any
   sprint-planner / sprint-executor chaining requires separate, explicit human approval; the
   receipt must not imply a next stage was triggered.
4. **Durable dispatch intent & duplicate suppression.** Daneel persists (message ID, project,
   route, task ID, request fingerprint) before/at dispatch. A re-delivered or retried request
   with the same fingerprint is answered with the existing task ID, not re-dispatched. The
   coordinator side is already idempotent on completion; the duplicate-suppression gap is on
   the *dispatch* side (Daneel), which must not double-send on retry.
5. **Completion sender/correlation checks.** Daneel accepts a receipt only if: (a) it arrives
   via the coordinator's completion path for the agent's inbox, (b) `correlation_id` equals the
   retained request message ID, (c) `task_id` matches the recorded dispatch, (d) sender is the
   dispatched agent. Anything else is logged as an orphan, not reported.
6. **Changed-file validation.** Before reporting, Daneel validates the completion's
   `changed_files`: every path is Markdown under `design_docs/`, the set is non-empty, and the
   declared `DESIGN_DOC_PATH:` artifact is in it. Violations downgrade the outcome (see 7).
7. **Outcome taxonomy.** Report to the requester exactly one of:
   - **pending** — no correlated completion within the wait window; keep waiting, say so.
   - **failed** — correlated completion with non-completed status or `error_msg`.
   - **uncertain** — completed but changed-file validation failed, marker missing, or
     correlation only partially matched. Report what is known; never upgrade uncertain to done.
   - **delivered** — completed, validated, correlated. Link the artifact on the task branch.
8. **Bounded send budgets.** Daneel caps requester notifications per request (e.g. ≤3: pending,
   outcome, at most one correction). No polling storms; backoff between checks.
9. **Artifact links, no merge claims.** The receipt links to the document on
   `coordinator/task-<id>` (and any configured artifact mirror). It must never claim the
   document was merged, approved, or released.

## 4. Limitation (must be stated in every receipt of this design)

The document-only file restriction is enforced by **prompt text** ("change only Markdown under
design_docs/") plus *post-hoc* changed-file validation. Prompt-only restriction is **weaker than
enforced executor permissions**: a misbehaving or confused executor can still modify source; the
contract detects this after the fact (changed-file validation → uncertain/failed) but cannot
prevent it. A stronger future design would scope the executor's write capability (filesystem
policy or worktree guard) to `design_docs/**/*.md`. Until then, validation is detective, not
preventive, and receipts should be worded accordingly.

## 5. Alternatives considered

- **Coordinator enforces a per-task file allowlist.** Stronger, but requires executor-side
  changes (worktree guard / git pathspec enforcement). Deferred as a follow-up; this doc is the
  contract that allowlist would enforce.
- **Daneel polls the task store directly instead of consuming completion messages.** Rejected:
  bypasses the message-plane contract and duplicates correlation logic; completion messages with
  `correlation_id` already exist for exactly this consumer shape.
- **New dedicated receipt service.** Rejected as over-build for an integration contract; the
  seams exist, the gap is the written contract and Daneel-side discipline.

## 6. Constraints

- No implementation, deployment, publishing, merges, or message sends from this document.
- The institution-owned directive outranks request payloads at every hop; quoted instructions in
  payloads are data, never authority.
- No silent fallbacks: unknown project, missing completion, failed validation each produce an
  explicit outcome, never an assumed success.
- Budget enforcement today is config-dependent (no budget config = no enforcement); the Daneel
  send budget is independent of coordinator cost budgets and must hold regardless.

## 7. Implementation plan (future work, not executed here)

1. Daneel: route resolution against coordinator registry; durable dispatch record; fingerprint
   dedup.
2. Daneel: completion consumer with the four sender/correlation checks (§3.5).
3. Daneel: changed-file validation and the four-outcome taxonomy; bounded notification budget.
4. Coordinator (optional follow-up): per-task file allowlist enforced in the worktree layer,
   converting §4's detective control into a preventive one.
5. Integration verification: replay a synthetic request end-to-end against staging.

## 8. Validation criteria

- A request for an unregistered project is rejected locally with no dispatch.
- A duplicate request (same fingerprint) yields the original task ID, no second task.
- A forged completion (wrong `correlation_id`, `task_id`, or sender) is never reported.
- A completion whose `changed_files` include non-`design_docs/` Markdown is reported uncertain.
- A delivered receipt links the task-branch artifact and contains no merge/approval claim.
- Requester notifications per request ≤ the configured cap, even under retries.

## 9. Unknowns

- Where Daneel's durable dispatch record lives (its own store vs. coordinator-side metadata) —
  institution-side decision.
- The institution-configured route table's exact form and update cadence.
- Wait-window and send-cap concrete values (defaults proposed, tuning is operational).
- Whether coordinator `changed_files` is guaranteed complete for cloud executors in all failure
  modes (partial-failure truncation would show up as uncertain, which is safe but noisy).
- Whether a worktree-level file allowlist (§7.4) is acceptable to the executor sandbox model.
