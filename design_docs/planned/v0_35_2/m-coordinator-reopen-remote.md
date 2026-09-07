# M-COORD-REOPEN-REMOTE: `coordinator reopen` --remote Support

**Status**: Planned
**Target**: v0.35.2
**Priority**: P1
**Estimated**: 0.5 day
**Dependencies**: None (pattern already shipped for approve/reject)

## Problem Statement

`ailang coordinator reopen <task-id>` is local-only. Unlike its siblings
`approve` and `reject` (cmd/ailang/coordinator_actions.go:26,140), it never
checks `remoteCoordinatorSelected(args)`, so with a remote plane selected it
silently resolves against the local SQLite file — exactly the failure mode the
approve conversion comment warns about ("reporting success either way"). Its
help text also fails to disclose the local-only limitation.

Secondary defect: help claims it reopens "rejected or cancelled" tasks, so a
task stuck in `pending_approval` has no CLI escape hatch even locally.

## Decision

**Option (a): gain `--remote` via the existing pattern.** Rejected option (b)
(fail loudly) because the defect report frames reopen as a sibling of
approve/reject that "was simply never converted" — a loud rejection leaves the
cloud plane with no reopen path at all, and the remote machinery already
exists: `coordinatorResolveRemote(args, action)` in
cmd/ailang/coordinator_approvals_remote.go:159 plus the remote store selected
by `remoteCoordinatorSelected` (cmd/ailang/coordinator_remote_store.go:108).

## Design

1. In `coordinatorReopen` (coordinator_actions.go:265), add the same guard as
   approve/reject, before flag parsing:
   `if remoteCoordinatorSelected(args) { return coordinatorResolveRemote(args, "reopen") }`.
2. Extend `coordinatorResolveRemote` to accept `"reopen"` alongside
   approve/reject, calling the remote store's ReopenTask equivalent.
3. Fix help text (and `coordinator.go:80` usage line): document `--remote`,
   and correct the status claim to match `ReopenTask`'s real semantics
   (rejected/cancelled → pending_approval), stating that pending_approval
   tasks need no reopen.

## Acceptance Criteria (all fail today)

1. `AILANG_STORAGE=gcp ailang coordinator reopen <task-id>` (or `--remote`)
   returns a non-nil error or silently touches only the local SQLite store —
   after the fix it must route to the cloud coordinator's Firestore store.
2. `ailang coordinator reopen --help` contains no mention of `--remote` or of
   being local-only; after the fix it must document both.
3. A cloud-plane task in `rejected` status cannot be reopened from the CLI at
   all; after the fix `reopen --remote <task-id>` returns it to
   `pending_approval` and it appears in `coordinator pending --remote`.

## Non-Goals

- Converting other unconverted subcommands (handled under the shared-store
  milestone m-coord-cli-shared-store.md).
- Changing ReopenTask status semantics in internal/coordinator.

## Related Documents

- design_docs/planned/v0_35_0/m-coord-cli-shared-store.md
