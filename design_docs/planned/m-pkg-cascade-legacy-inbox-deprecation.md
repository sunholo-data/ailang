# M-PKG-CASCADE-LEGACY-INBOX-DEPRECATION: Stop dispatching legacy `pkg:*` inbox copies as cascade tasks

**Status**: Proposed (triage of incident task-a28b6d8b — "cascade routing: EMPTY dispatch")
**Target**: unassigned — needs human approval (semantics change to the publish/cascade flow)
**Priority**: P1 — every dependent notification currently fires one wasted Cloud Run job + one duplicate bug report
**Origin**: bug report task-a28b6d8b, filed by the `sunholo/docparse` package agent on the test plane

## Incident summary

The docparse package maintainer agent received a "cascade dispatch" for a
`sunholo/ailang_parse` release in which **every cascade field was blank**:
root package, bump (`from → to`), change class, interface hashes, effect
widening, and — the decisive one — `Source`. The agent correctly refused to
fabricate a repair: its `ailang.toml` still pinned
`sunholo/ailang_parse = 0.39.4` (matching the current release commit), there
was no changed interface to adapt to, and `ailang check --package .` failures
were registry-cache-not-found only. No commits were made. **This refusal was
the correct behavior**, and it is what converted a silent misroute into an
evidenced bug report.

## Root cause

The `ailang publish` flow **dual-writes** dependent notifications
(`cmd/ailang/pkg_publish.go::emitDependentNotifications`):

1. **Authoritative path**: publish to the IAM-restricted `ailang-cascade`
   Pub/Sub topic via `PublishCascadeWithEnvelope`, which unconditionally
   stamps `source=cascade` and embeds the full envelope (root package, change
   class, version pair, interface/content hashes, effect ceilings) in the
   message data (`internal/pubsub/publisher.go`).
2. **Legacy path**: `EmitUpgradeAvailable` writes an upgrade-available message
   into the **dependent's Firestore `pkg:<vendor>/<name>` inbox**. This row
   carries no Pub/Sub attributes and no cascade envelope — those exist only on
   the wire of path 1.

The cloud coordinator drains `pkg:*` inboxes (direct drain, or the backstop
sweep via `PubSubInboxAdapter.Enqueue`, which by construction carries no
attributes). The hydrated `Message` therefore has `Source == ""` and every
cascade envelope field empty. Both task-creation sites copy those fields into
the `TaskRecord` verbatim (`internal/coordinator/daemon_tasks_polling.go`,
both call sites), the dispatcher propagates the blanks
(`internal/coordinator/daemon_tasks_exec.go`), and `BuildDirectiveFromConfig`
renders `pkg-update.md` with every `{{.RootPackage}}`-family placeholder blank
(`internal/coordinator/stage_execution.go`).

**Why `Source` blank is diagnostic**: any task created from a cascade-topic
delivery cannot show blank `Source` — the publisher forces the attribute, the
adapter reads it (`internal/coordinator/pubsub_adapter.go`), and both task
sites copy it. Blank `Source` ⇒ the message never travelled over Pub/Sub ⇒
the legacy Firestore inbox copy. There is no third route into a `pkg:*`
inbox.

**This was a known, deferred decision.** The v0.16 design doc
(`design_docs/implemented/v0_16_0/m-pkg-autonomous-cascade-safe.md`) chose
dual-write "for one release of observation" and explicitly recommended:
"Recommend dropping [the legacy `pkg:*` inbox write] after one release of
dual-write observation" (line 295) and "deprecate legacy in v0.17" (line 332;
sprint plan line 213). The deprecation never happened; we are many releases
past v0.17.

## Aggravating factor: the misroute runs with auto-merge autonomy

`AdjustAutonomyForChangeClass` (`internal/coordinator/autonomy_router.go`,
called from `daemon_tasks_exec_run.go`) re-reads the Firestore inbox message
by `task.MessageID` and adjusts autonomy from the package envelope. A legacy
`upgrade-available` classified `patch` maps to ChangeClassA →
`SkipApproval=true, AutoMerge=true`. So each misrouted dispatch is a paid
Cloud Run job with auto-merge autonomy, whose **only** safety stop is the
agent-side template guard. That guard worked this time — but "a template
instruction is the only thing between a misrouted auto-merge job and a push"
is not an acceptable steady state.

## Why the empty dispatch must not be "repaired"

There was no changed interface to adapt to: the pin matched the current
release, so a correct cascade would have been a class-A deterministic bump or
nothing. The empty directive was not a lossy cascade; it was a legacy notice
wearing a cascade template. Fabricating consumer changes from blank fields
would be a fallback value reaching business logic — the failure shape this
repo's second critical principle forbids outright.

## Proposed fix

Two changes, ordered; (1) is the fix, (2) is defense in depth.

### 1. Send side (primary): drop the legacy `pkg:*` inbox write for dependent notifications

In `emitDependentNotifications` (`cmd/ailang/pkg_publish.go`), remove the
`EmitUpgradeAvailable` call per dependent and keep only the
`PublishCascadeWithEnvelope` call. This executes the v0.16 design's own
deferred recommendation. The cascade topic is IAM-restricted and carries the
full envelope; the inbox copy adds noise and nothing else. Cascade-publish
failure is already logged loudly (`Cascade publish failed for <dep>: ...`);
if it fails, the correct outcome is a loud gap a human can see, not a
misrouted legacy task that silently no-ops.

**Open question for the approval gate**: the self-package coordination writes
in `emitPublishMessages` (upgrade-available / interface-change-notice /
effect-widening-warning into the package's *own* inbox) use the same legacy
route. They predate the cascade topic and are inbox-native by design, but they
are dispatched to package agents through the same blank-template path. Either
(a) they get the same receive-side guard treatment as (2), or (b) they are
also retired. Decide during design review; this doc proposes (a).

### 2. Receive side (defense in depth): refuse non-cascade package messages at the drain

In the cloud drain (`daemon_tasks_polling.go`) and the backstop sweep
recovery path, when a hydrated message routes to a `pkg:*` agent, is a
package-envelope kind (upgrade-available, interface-change-notice,
effect-widening-warning), and `Source != "cascade"`: do **not** create a task.
Log loudly, ack/mark-read, and emit a structured event naming the message ID
and the missing provenance. This closes the class regardless of who writes
legacy rows — older publish binaries still deployed, manual
`ailang messages send`, or queue replay.

Additionally: `AdjustAutonomyForChangeClass` must not grant
`SkipApproval`/`AutoMerge` from a message without cascade provenance (or the
(2) guard makes the case unreachable; either way, add a test asserting a
non-cascade upgrade-available can never produce an auto-merge-eligible task).

### Tests

- Extend `scripts/integration/test_cascade_e2e.sh`: one publish → exactly one
  task per dependent, `source=cascade`, envelope populated, no `pkg:*` inbox
  row for the dependent notification.
- Extend `scripts/integration/test_cascade_negative.sh`: legacy inbox row
  present + no cascade delivery → loud refusal event, no task created, agent
  never invoked.
- Unit: task-creation sites drop/flag envelope-less pkg messages;
  autonomy router never escalates autonomy on `Source != "cascade"`.

## Cost of doing nothing

Every dependent notification of every publish fires a wasted Cloud Run job
whose only possible outcomes are (a) guard fires → wasted spend + a duplicate
`[bug] cascade routing` report, or (b) guard ignored → an unauthorized bump
attempt with zero cascade context. The duplicate reports also train alarm
fatigue: the more agents file identical routing bugs, the more likely a real
misroute gets skimmed past one.

## Verification

- [ ] `make test` green; new unit tests for the drain guard and autonomy router
- [ ] e2e + negative integration scripts above pass against the test plane
- [ ] One real publish on the test plane produces exactly one cascade task per dependent