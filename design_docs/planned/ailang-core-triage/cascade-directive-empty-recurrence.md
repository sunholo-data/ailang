# Cascade routing: unpopulated cascade directive (task-dbc6e559)

- **Date**: 2026-10-17
- **Class**: bug
- **Recommend**: duplicate-of `design_docs/planned/ailang-core-triage/cascade-directive-unpopulated.md`
- **Searched**: `cascade directive`, `hydrateCascadeFields`, `backstop`, `empty Source` across
  `design_docs/planned/` and `design_docs/implemented/`; read the prior triage
  `design_docs/planned/ailang-core-triage/cascade-directive-unpopulated.md`; re-verified the
  mechanism in `internal/coordinator/` source.

Another instance of the same recurring coordinator bug family already diagnosed and triaged in
`design_docs/planned/ailang-core-triage/cascade-directive-unpopulated.md` (task-dc0edc0a, since
independently reported by task-b3af211c and others): a cascade message dispatched with `Source`
empty and no cascade envelope fields, so the `pkg-update` template renders blank
`{{.RootPackage}}`/`{{.FromInterfaceHash}}`/… and the wrapper agent correctly stops without
committing (defense-in-depth rule: empty Source → misroute → file bug, do not commit). Two
refinements since that triage, both verified in source today: (1) the pub/sub route is now
defended — `HandleNotification` (`internal/coordinator/pubsub_adapter.go`, the hydration block
around :218–:260) fetches from the message store and *refuses to dispatch* when hydration yields
no usable content, so the silent nil-envelope fall-through the prior triage flagged appears fixed;
(2) the backstop sweep route survives unchanged — `sweepInbox`
(`internal/coordinator/backstop_sweep.go`, the `adapter.Enqueue(&Message{…})` call, comment block
at ~:194) deliberately preserves `Kind`, `CreatedAt`, and `Inputs` but still drops `Source` and the
cascade envelope fields, and `daemon_tasks_polling.go` inherits `Source: msg.Source` at both task
creation sites. So the actionable residue of the duplicate is narrower than the original triage
described: a single-field-class fix to the sweep's re-enqueue (or a fail-loud guard in
`buildTemplateDirective`, `internal/coordinator/stage_execution.go`, against rendering a cascade
template with an empty envelope). The prior triage's design-doc recommendation stands for that
residue; no design doc has ruled on it since (`backstop`/`empty Source` in `design_docs/planned/`
returns only this family's own files). No new investigation warranted — point the cascade-routing
design doc at this instance as further recurrence evidence.
