# Cascade routing: empty dispatch, docparse (task-b3af211c)

- **Date**: 2026-10-09
- **Class**: bug
- **Recommend**: duplicate-of design_docs/planned/m-pkg-cascade-legacy-inbox-deprecation.md
- **Searched**: `design_docs/planned/ailang-core-triage/cascade-directive-unpopulated.md`
  (the family triage row: same mechanism, four prior parallel diagnoses —
  task-e891c221, task-ec1494ab, task-e705df2e, task-dc0edc0a);
  `design_docs/planned/m-pkg-cascade-legacy-inbox-deprecation.md` (Proposed design
  doc raised from the task-a28b6d8b instance of this same family); source check of
  `internal/coordinator/pubsub_adapter.go`, `internal/coordinator/backstop_sweep.go`,
  `internal/coordinator/daemon_tasks_polling.go`, `internal/coordinator/stage_execution.go`,
  `internal/pubsub/publisher.go`, `cmd/ailang/pkg_publish.go` — mechanism re-verified
  unchanged at HEAD 87ece315 (v0.54.3).
- **Estimate**: n/a — fix already specified in the duplicated design doc.

The report is a fifth instance of the measured cascade-empty-dispatch family: a
`pkg:*`-routed task (docparse, pinning `sunholo/ailang_parse = 0.39.4`) arrived with
`Source` empty and every cascade envelope field blank, so the `pkg-update` template
rendered an unpopulated directive and the agent correctly refused, filing a bug
instead of fabricating a bump. The design doc's root cause explains every symptom
verbatim: the legacy `EmitUpgradeAvailable` Firestore row (dual-write path 2 of
`emitDependentNotifications`) carries no Pub/Sub attributes and no cascade envelope;
however it reaches the drain (direct or via the backstop sweep's `Enqueue`, which
preserves `Kind`/`CreatedAt`/`Inputs` but not cascade fields), the task record
inherits `Source: ""` and empty envelope fields, and `BuildDirectiveFromConfig`
renders blanks. No code change is recommended here beyond the already-proposed doc;
what follows are two corrections to the field report's own diagnosis, because both
change what a human retry should do:

1. **The dispatch to docparse was probably NOT misrouted.** The reporter inferred
   "expected sunholo-data/ailang-packages" from the empty cascade context — but the
   context was empty precisely because the message never carried it. docparse depends
   on `sunholo/ailang_parse` (pinned in its `ailang.toml`), which makes docparse a
   legitimate cascade *dependent* of an `ailang_parse` release; a populated cascade
   for that root would route exactly there. Blank `Source` proves "legacy inbox
   copy", not "wrong repo" (see the design doc's "Why `Source` blank is diagnostic").
2. **"Needs human retry with populated cascade context" is the wrong remediation for
   this instance.** There is no populated context to retry with: the pin already
   matched the released version, so the correct outcome was a class-A deterministic
   bump or nothing. Re-dispatching the same message with hand-filled fields would
   fabricate cascade provenance the event never had. The remediation is the design
   doc's fix: stop the legacy dual-write (send side) and refuse envelope-less package
   messages at the drain (receive side).

The report's operational behavior was correct throughout: no commit, no success
claim, defense-in-depth refusal on empty `Source`. The cost it incurred is the
duplicate-report alarm-fatigue tax the design doc's "Cost of doing nothing" section
predicts — this family now has at least five independently-filed instances
(task-a28b6d8b, task-dc0edc0a plus its three cross-referenced diagnoses, and this
one), which raises the priority of approving the Proposed doc rather than triaging
each future copy.
