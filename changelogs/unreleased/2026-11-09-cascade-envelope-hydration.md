### Fixed — a cascade bump request can no longer reach a package agent with a blank envelope (2026-11-09)

Every route that delivered a dependency-bump request to a `pkg:*` agent without its cascade
envelope produced the same paid run: the cascade template rendered blank, and the agent filed a
`[bug] cascade routing: EMPTY dispatch` report and stopped. That happened 627 times on the test plane
from 2026-08-31 to 2026-09-02, when the backstop sweep re-enqueued unread docparse rows. The burst
ended when the inbox was drained and the sweep went to `report` mode, but every code route was
still open, and a planned flip back to `dispatch` would have brought it back. Prod shows 0 such
reports in its last 5000 messages.

The coordinator now refuses instead of reconstructing. A stored row carries no `Source`, and its
payload's change class is not the A/B/C taxonomy the wrapper reads, so guessing would be worse than
refusing:

- **Pub/Sub adapter**: a cascade-topic delivery with no envelope is acked and logged, never buffered.
- **Backstop sweep**: a cascade-shaped row is logged as `WITHHOLDING` and left unread for triage,
  in both modes.
- **Cloud dispatch gate**: a cascade-shaped task for a package agent that lacks `source=cascade`,
  `root_package` or `to_version` fails with "dispatch refused" before any job is created. Package
  agents' ordinary bug and feature work is not gated.
- **Envelope copy**: both task-creation sites now use one helper. The local site used to drop the
  envelope.
- **SQLite task store**: the eleven envelope columns are now read back. It used to write them and
  never read them.

Design: `design_docs/implemented/v0_57_5/m-cascade-envelope-hydration.md`. That doc also records
the stale assumptions behind the changes and the before/after measurement.
