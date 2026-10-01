### Fixed — a failing Cloud Trace init killed the process instead of erroring (2026-09-30)

`cloudtrace.New` is declared `(*Exporter, error)` and returns `nil, err` on every failure path.
The wrapper returned that straight into the `sdktrace.SpanExporter` interface, which yields a
**non-nil interface holding a nil pointer** — so `exporter != nil` passed, the dispose path called
`(*Exporter).Shutdown`, and it dereferenced `e.traceExporter`. A SIGSEGV, not an error.

It fires whenever Cloud Trace cannot initialise: no Application Default Credentials reachable, or
an offline box. `ailang repl --help` exited 2 with a segfault instead of printing help, and it
would hit any command on a machine that cannot reach GCP. Found while tracing an unrelated fault;
`TestCLI_HelpExitsZeroEverywhere` and `TestGroups_GroupRouteIsByteIdenticalToTheTopLevelSpelling`
had been failing on it.

The constructor now normalises to an untyped nil, and `isNilExporter` (reflect-based) guards the
dispose site so no injected constructor can panic the process either. The regression test
reproduces the original SIGSEGV when the guard is reverted.
(`internal/platform/otel/sdk.go`, `sdk_nilexporter_test.go`)


### Fixed — `ailang storage status` named the wrong project for the messaging store (2026-09-30)

`AILANG_MESSAGES_PROJECT` pins the messaging store's project and wins over the generic resolver —
that is what `ailang messages` actually opens. `storage status` resolved one project for every GCP
store, so it reported the generic one for the messaging row. On the laptop it read
`project ailang-multivac-dev (AILANG_CLOUD_PROJECT)` — the stale `-dev` graveyard — while
`ailang messages list` printed `project ailang-multivac` in its own header and was reading prod.

CLAUDE.md's session-start check tells every machine to confirm the messaging store with this
command, and trap #1 in the message-plane topology doc sends you here specifically to defend
against that graveyard. The instrument lied in the alarming direction: chasing a wrong-project
misconfiguration that did not exist was the first wrong turn in the notify-daemon outage the same
day. The two commands now agree. (`cmd/ailang/storage.go`, tests in `storage_test.go`)

### Docs — corrected stale material that misattributed a healthy subscription as an outage (2026-09-30)

`ailang-tasks-executor` was reported as a dead consumer on the strength of comments describing an
architecture that was never built. Three places said it, and all three now say what shipped:

- `internal/pubsub/topics.go` and multivac's `terraform/pubsub.tf` claimed "Eventarc → Cloud Run
  Job". The coordinator calls the Cloud Run Jobs API directly; the Eventarc trigger is commented
  out in `eventarc.tf` with its reasoning. The subscription is an audit trail nothing consumes, so
  a backlog on it is the steady state — now stated, along with "do not alert on it".
- `design_docs/implemented/v0_9_0/m-cloud-infra.md` planned the Eventarc path and its tables still
  describe it. Marked superseded in that one respect rather than rewritten, since it is a
  historical design record.

Also documents two delivery-layer facts that had none, both of which had to be rediscovered from
Cloud Monitoring metrics:

- `docs/internal/message-plane-topology.md` gained a **delivery layer** section: one subscription
  per device (shared subscriptions work-steal), ordering keyed on inbox so one stuck message blocks
  that inbox's whole stream, how to read the backlog metrics (there is no
  `gcloud monitoring time-series` subcommand), which backlogs are expected, and that an undeclared
  subscription self-deletes after 31 days without a pull.
- `docs/docs/guides/notification-channels.md` now states that the Discord webhook's Keychain read
  requires the LaunchAgent to load into the **Aqua** session, gives the exit-36
  (`errSecInteractionNotAllowed`) signature, and shows how to verify the channel actually
  registered — registration is fail-closed and announces failure in a single log line.


### Fixed — a notification for a message that does not exist no longer blocks every real one (2026-09-30)

The laptop notify daemon acknowledged 10 messages and rejected 160,947 in six hours, and the Mac
stopped alerting. 360 notifications published 2026-09-24/25 named inbox documents that were never
written to the prod store (`task-*:handoff:*` ids from a run whose message store was ephemeral).
Resolving one is a hard 404, so the daemon nacked it — and a subscription drains oldest-first, so
the poison sat in front of the real traffic and starved it. Nothing ended the loop but Pub/Sub's
7-day retention.

Absence and failure were indistinguishable, and both were retried forever. The two backends did not
even agree on how to say "absent": SQLite's `GetInboxMessage` returns `(nil, nil)`, Firestore's
returned a bare `fmt.Errorf`. Every nack was also silent — no log line — which is why it ran 6.6
days unnoticed.

- `messaging.ErrMessageNotFound` is now the shared way to say absent; the Firestore backend wraps
  it, and `IsMessageNotFound` is how a caller tells "this will never resolve" from "retry me".
  (`internal/messaging/notfound.go`, `internal/storage/firestore/messaging_inbox.go`)
- The daemon retries an absence for 5 minutes — long enough for Firestore replication, which is
  the only real reason a notification can outrun its document — then **acks** it and logs
  `DROPPING message <id>: unresolvable`. A retryable failure (network, permission, deadline) is
  still nacked for as long as it keeps failing, so a Firestore outage cannot discard
  notifications. (`internal/daemon/unresolved.go`, `daemon.go`)
- Both retry and drop paths now log, so the next occurrence is visible in
  `ailang daemon status`.

### Fixed — three defects in the generated daemon plist (2026-09-30)

Found while tracing why Discord posts stopped. All three were silent.

- **`--messages-sub` could not be expressed.** `daemon install` always wrote `daemon run --env
  <env>`, so a second device could only get its own subscription by hand-editing the plist — which
  the next `--force` install reverted, putting both daemons back on one subscription to work-steal
  from. `ailang daemon install --messages-sub messages-rig` now writes it.
- **`AILANG_STORAGE=gcp` moved all three stores.** The daemon needs messaging in the cloud and
  nothing else; it was also pointing its coordinator and observatory reads at the cloud project
  (the `Observatory: 500MB` warning it logs was measuring the wrong store). Now
  `AILANG_STORAGE_MESSAGING=gcp` + `AILANG_MESSAGES_PROJECT`, per the one-plane-switch rule in
  CLAUDE.md.
- **The login Keychain was unreachable.** Channel registration is fail-closed and reads the Discord
  webhook from the Keychain, which only the Aqua session can reach. The rig's webhook item was
  present the whole time and read with exit 36 (`errSecInteractionNotAllowed`), so the daemon
  booted with `channels=[macos]` and one log line. The plist now sets
  `LimitLoadToSessionType=Aqua`.
