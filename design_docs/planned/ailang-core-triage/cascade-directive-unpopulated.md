# Cascade routing: unpopulated cascade directive (task-dc0edc0a)

- **Date**: 2026-09-27
- **Class**: bug
- **Recommend**: design-doc
- **Searched**: `cascade directive`, `cascade envelope`, `hydrateCascadeFields`,
  `buildTemplateDirective`, `backstop sweep` across `internal/coordinator/` and
  `design_docs/`; browsed `design_docs/implemented/v0_16_0/m-pkg-cascade-deterministic-first.md`,
  `m-pkg-autonomous-cascade-safe.md`, `design_docs/planned/v0_29_0/m-cascade-observability.md`,
  and `design_docs/planned/v0_40_0/m-pkg-quality-ladder.md`. No design doc currently rules on the
  inbox→task envelope-loss route (none found for this mechanism; the v0.16.0 docs cover only the
  pub/sub embedded-envelope path — `m-pkg-cascade-deterministic-first.md` mentions hydration solely
  in the Firestore storage converter).

The report is one instance of a measured, recurring coordinator bug: cascade messages that reach a
task through anything other than the pub/sub embedded-envelope path dispatch with `Source` empty and
no cascade envelope fields, so `buildTemplateDirective` (`internal/coordinator/stage_execution.go`)
renders the `pkg-update` template with blank `{{.RootPackage}}`/`{{.FromInterfaceHash}}`/… — an
unpopulated cascade directive that tells the agent to perform a deterministic bump it cannot execute.
Two lossy routes are verified in source: the backstop sweep re-enqueues recovered messages without
`Source` or envelope fields (`internal/coordinator/backstop_sweep.go`, the `Enqueue(&Message{…})`
call — it deliberately preserves `Kind` and `CreatedAt` after the 2026-08-31 notice-flood incident
but not cascade fields), and `HandleNotification` (`internal/coordinator/pubsub_adapter.go`)
silently proceeds with a nil envelope when the data decodes as `CascadeMessageData` but the envelope
is absent, instead of falling through to payload-derived hydration. Downstream, the task record
inherits `Source: msg.Source` (`daemon_tasks_polling.go`, in both task-creation sites) and the empty
template variables render as described. This bug family has already been independently diagnosed by
at least three parallel coordinator tasks (task-e891c221, task-ec1494ab, task-e705df2e — their
summaries are in the coordinator inbox), converging on the same shape: a single
`hydrateCascadeFields(msg)` helper populated from the durable `PackageMessageEnvelope` payload, wired
into `HandleNotification` and the backstop sweep, plus a fail-loud render/dispatch-time guard against
dispatching a cascade template with an empty envelope. That is coordinator routing semantics with
multiple acceptable remedies (reconstruct-and-stamp vs withhold-for-triage at the sweep, and where
the guard belongs), spanning `pubsub_adapter.go`, `backstop_sweep.go`, `stage_execution.go`, and
tests — so it routes through design-doc-creator and the standard gate, not a direct fix. Attribution
caveat: task-dc0edc0a's own record could not be retrieved from this task's identity (prod store
`ailang-multivac` is PermissionDenied here; the test store has no matching message), but its symptom
matches the family above and the mechanism is verified in source regardless.
