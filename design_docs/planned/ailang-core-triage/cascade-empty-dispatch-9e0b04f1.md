# Cascade routing: EMPTY dispatch recurrence (task-9e0b04f1)

- **Date**: 2026-10-22
- **Class**: bug (duplicate of an already-designed, unimplemented fix)
- **Recommend**: none — do NOT design again; land the existing design doc
- **Searched**: verified every mechanism claim of the existing design doc at HEAD
  `42d93e4b` (2026-10-22): `grep -rn "hydrateCascadeFields" internal/ cmd/` (empty —
  helper still does not exist); read `internal/coordinator/backstop_sweep.go`
  (the `Enqueue(&Message{…})` block still omits `Source` and all 11 cascade envelope
  fields); read the local task-creation site in `daemon_tasks_polling.go` (still
  copies `Source` only, no envelope fields); read
  `internal/coordinator/pubsub_adapter.go` (still silently proceeds with a nil
  envelope when cascade data decodes envelope-less).

This report is the 10th+ instance of the EMPTY-dispatch family. The family has a
complete, reviewed design doc awaiting implementation:
[`design_docs/planned/v0_44_0/m-cascade-envelope-hydration.md`](../v0_44_0/m-cascade-envelope-hydration.md)
(P0, net axiom score +7, design-freeze items resolved, ~1 day estimate, previously
independently diagnosed by seven parallel coordinator tasks and triaged to
`design-doc` by task-e1e2ce90). It is a *planned* doc — nothing from it has landed.

Task-9e0b04f1's own behavior was correct per the repo's work-routing guardrails and
the `pkg-update.md` defense-in-depth note: Source empty ⇒ misroute ⇒ file the bug
and stop without committing. Its evidence (branch at main, clean tree, dep pin
already at the released version, `ailang check` failures limited to registry
cache-not-found) is consistent with the doc's verified mechanism — a legacy-inbox
route or sweep recovery carrying no Pub/Sub attributes, not a real cascade.

**Routing note**: per AGENTS.md work routing, the fix is a multi-file
semantics/routing change behind an approved design doc. Implementing it from this
bug report without the `design-doc → approval → sprint-plan → execute` gate would
violate the gate. The correct next action for the human queue is to schedule
`m-cascade-envelope-hydration` for a sprint; this session logged the recurrence in
the doc's new Recurrence Log so the accumulating count is visible in the artifact
that needs approval.
