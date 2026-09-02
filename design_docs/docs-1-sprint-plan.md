# Sprint Plan: DOCS-1 Inbox-Routing Trigger

## Summary

Build `tools/messaging/docs_inbox_router.sh`, a standalone bounded poller for doc-related
traffic in the canonical `public-feedback` and `pkg:<vendor>/<name>` message inboxes plus
new `sunholo-data/ailang` GitHub issues. It forwards each matching source item to
`docs-mission`, persists per-item idempotency keys and GitHub watermark state, and exposes a
safe `--selftest` path. This is a planning-only handoff; the executor must create only the
script named above.

**Duration:** 1 iteration / 1 day (approximately 6 hours)
**Dependencies:** Existing `ailang messages send`, `ailang messages forward`, canonical
Firestore message store, and `gh issue list`; no Go or dispatch-path changes.
**Risk Level:** Medium (external CLI/cloud failure handling and idempotency state)

## Current Status Analysis

### Baseline on the pristine tree

The worktree was clean apart from the supplied untracked routing brief. Before this plan was
written, the following measurements were made:

- `test -e tools/messaging/docs_inbox_router.sh` and `test -x ...` both returned false:
  the trigger is absent, so routing, re-run idempotency, summary output, and self-test are not
  already satisfied.
- `AILANG_MESSAGES_STORE=not-a-real-store AILANG_MESSAGES_PROJECT=ailang-multivac ailang
  messages list --inbox docs-mission --unread --json` returned rc=1 with
  `unknown message store mode`; the scoped selector is therefore suitable for a loud
  configuration/control check. The command also emitted pre-existing readonly Observatory
  cleanup and stale-binary warnings; the router must not mistake those warnings for data.
- `AILANG_MESSAGES_STORE=gcp AILANG_MESSAGES_PROJECT=ailang-multivac ailang messages list
  --inbox docs-mission --unread --json` produced no JSON before the bounded probe ended and
  emitted only the same readonly/stale-binary warnings. This is a baseline of an unavailable
  canonical read in this environment, not evidence that the inbox is empty.
- `gh` is installed (`/opt/homebrew/bin/gh`), but no completed issue probe was obtained in
  the baseline command. The executor must run the explicit bounded GitHub probe below and
  record whether authentication/network access is available.
- Clause 7 was independently read and fixes the required command forms: `send` flags follow
  the body; `forward --to <inbox> --reason "<reason>" <message-id>` puts flags before the
  message ID; and canonical reads use `--inbox docs-mission --unread --json`. The cloud
  variables must be scoped to individual commands, never exported process-wide.

### Estimate

One small shell script plus its self-test logic is estimated at 150–220 lines including
comments and defensive checks. The one-day estimate is deliberately bounded: no launchd
wiring, Go changes, package installation, or repair of GitHub/message infrastructure is in
scope.

## Proposed Milestone

### M1: Bounded, idempotent docs inbox router

**Goal:** Implement and verify the new router at
`tools/messaging/docs_inbox_router.sh`.
**Estimated:** 170 LOC implementation + 50 LOC self-test/documentation = 220 LOC total
**Duration:** 1 day / approximately 6 hours
**Files:**

- `tools/messaging/docs_inbox_router.sh` (NEW; the only executor implementation file)

**Tasks:**

- First, inspect the existing messaging/launchd shell conventions and choose a state directory
  that is configurable for tests. Use an atomic state update and a stable source-qualified
  idempotency key (message ID plus source/inbox; GitHub issue number plus repository), not only
  a timestamp watermark.
- Implement bounded wrappers for every `ailang messages list`, `ailang messages forward`, and
  `gh issue list` call. Require command success, validate expected JSON/shape, and distinguish
  unreachable/error from a valid zero-result read. Do not export `AILANG_MESSAGES_STORE` or
  `AILANG_MESSAGES_PROJECT` globally; prefix only the commands that need the canonical store
  with `AILANG_MESSAGES_STORE=gcp AILANG_MESSAGES_PROJECT=ailang-multivac`.
- Poll `public-feedback`, enumerate or otherwise discover `pkg:<vendor>/<name>` inboxes,
  and query GitHub issues after a persisted watermark. Apply a documented, defensible
  case-insensitive heuristic for docs site, examples, guides, published pages and related
  terms/labels; explicitly state what the heuristic can miss and that it is not a precision
  classifier.
- For each match, invoke exactly the verified forward shape
  `ailang messages forward --to docs-mission --reason "<reason>" <message-id>` and persist
  the idempotency key only after a successful forward. Emit exactly one final summary line in
  normal mode: `checked=N forwarded=M`.
- Implement `--selftest` without depending on pre-existing live traffic. It may use a
  configurable fake command/dry-run harness or a uniquely tagged synthetic message, but must
  exercise matching, forward argument ordering, state recording, duplicate suppression and
  the non-vacuous known-positive/known-negative read controls. Keep any live synthetic data
  cleanup explicit and bounded.
- Verify the script with the acceptance commands below and record all external-access
  limitations in the executor handoff rather than converting them to passing empty results.

## Acceptance Criteria and Pristine Baselines

Each criterion below was measured before implementation. The baseline is intentionally the
current absence (`router_present=0`) where the new behavior does not exist; the executor must
demonstrate the stated transition to passing rather than inherit it silently.

1. **External routing (brief, verbatim):** “A message sent from outside docs-mission's own
   inbox (e.g. via `ailang messages send public-feedback "..."` or an equivalent test-visible
   channel) is observed, by the script, forwarded into the docs-mission inbox, and is readable
   there via the verified read command from clause 7's own verification log (`ailang messages
   list --inbox docs-mission` or equivalent — confirm the actual inbox name/command against
   that log rather than assuming it).”
   - Baseline: FAIL/absent; the router path does not exist (`test -e tools/messaging/docs_inbox_router.sh`
     returned false), and the canonical read probe yielded no result before the bounded probe
     ended.
   - Test: with scoped variables on each command, send a uniquely titled doc request using
     `ailang messages send public-feedback "docs-1 acceptance $(date -u +%Y%m%dT%H%M%SZ)"
     --title "docs-1 acceptance ..." --from "docs-1-test"`; run the router; then run
     `AILANG_MESSAGES_STORE=gcp AILANG_MESSAGES_PROJECT=ailang-multivac ailang messages list
     --inbox docs-mission --unread --json` and assert the source ID/body is present.

2. **Idempotency (brief, verbatim):** “Re-running the script against the same already-forwarded
   message does not forward it twice (idempotency check, demonstrated with two consecutive
   runs).”
   - Baseline: FAIL/absent; no script or forwarding ledger exists (`test -x ...` returned
     false).
   - Test: run the router twice with the same temporary state directory and fixture/live
     message; assert the first run reports one forward, the second reports zero forwards, and
     the docs-mission JSON contains exactly one matching forwarded item. The self-test must
     run the same two-pass assertion without live traffic.

3. **Loud store failure (brief, verbatim):** “The script exits non-zero and prints a clear
   error if it cannot reach the message store at all (never a silent `checked=0 forwarded=0`
   that looks identical to "nothing new").”
   - Baseline: FAIL/absent; invoking the intended script is impossible because the file is
     absent. The control `AILANG_MESSAGES_STORE=not-a-real-store AILANG_MESSAGES_PROJECT=ailang-multivac
     ailang messages list --inbox docs-mission --unread --json` did return rc=1 with a clear
     unknown-store error, while the canonical gcp probe produced no data before ending.
   - Test: inject a failing `ailang`/network fixture or point the router at an unreachable
     store; assert non-zero status, stderr names the failed source/store and operation, and
     stdout does not contain a successful-looking `checked=0 forwarded=0` summary.

4. **Scope and boundedness:** only the new `tools/messaging/docs_inbox_router.sh` is needed
   for implementation; no `internal/`, `cmd/`, Go source, ailang#900 dispatch code, or
   launchd job is changed. Every external call has a finite timeout and validates non-vacuous
   output; cloud variables are command-scoped.
   - Baseline: PASS for the repository’s current absence of implementation changes, but the
     required script behavior is absent; this criterion is a guard on the future diff, not a
     claim that the feature already works.
   - Test: `git diff --name-only -- tools internal cmd`; `shellcheck tools/messaging/docs_inbox_router.sh`
     if available; inspect each external invocation; run `env | rg '^AILANG_(MESSAGES_STORE|MESSAGES_PROJECT)='`
     before and after to show no process-wide export; exercise timeout fixtures.

5. **Self-test and summary:** `tools/messaging/docs_inbox_router.sh --selftest` completes
   without live external traffic and verifies forward argument ordering, state persistence,
   duplicate suppression, and known-positive/known-negative controls; normal runs print one
   line exactly matching `checked=N forwarded=M`.
   - Baseline: FAIL/absent; the script and `--selftest` do not exist (`router_present=0`).
   - Test: `tools/messaging/docs_inbox_router.sh --selftest`; assert rc=0 and its reported
     checks include a two-pass duplicate test; run a fixture-backed normal poll and assert the
     final line matches `^checked=[0-9]+ forwarded=[0-9]+$`.

## Success Metrics

- All five acceptance criteria pass with command output recorded by the executor.
- Two consecutive polling passes never duplicate a forwarded source item, including an item
  arriving out of order relative to the GitHub/message watermark.
- Store/network/auth failures are bounded and actionable; no empty-success fallbacks.
- Implementation diff is limited to `tools/messaging/docs_inbox_router.sh`; no launchd wiring.

## Dependencies and Risks

- Canonical message commands require per-command
  `AILANG_MESSAGES_STORE=gcp AILANG_MESSAGES_PROJECT=ailang-multivac`.
- Live acceptance requires access to prod Firestore and, for GitHub polling, authenticated
  `gh` access. If unavailable, self-test and injected failure fixtures remain mandatory, but
  live AC evidence must be marked blocked rather than passed.
- Message inbox enumeration and GitHub JSON formats may drift. Validate fields explicitly and
  fail loudly when the shape changes.
- Persisted state must be writable and protected from partial writes; support a configurable
  state path so CI does not share production state.

## Out of Scope

No launchd installation, no changes to push dispatch/ailang#900, no `internal/` or `cmd/`
changes, no package installation, and no git write operations. The controller commits the
planner’s artifacts and the executor’s implementation.
