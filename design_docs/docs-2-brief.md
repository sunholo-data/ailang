# docs-2 sprint brief — first real docs-sync sweep

**Not a design doc.** Per `docs-mission.md`'s Guardrails ("most items here need no design doc at
all — prefer a Gate-2 reality-check straight into a sprint"), this is a routing declaration only —
it exists so `tools/launchd/derive-planner-lane.sh` can route the planner to the mission's cheap
lane instead of failing closed to opus for a missing `Planner-Lane` field. It carries no design
claims and needs no quorum.

**Planner-Lane**: codex-ok

## Task

Run the `docs-sync` skill end to end (`audit_design_docs.sh`, `check_versions.sh`,
`check_examples.sh`, `derive_roadmap_versions.sh --full --check`, `generate_report.sh`) against the
current repo and turn the findings into a scored, clause-tagged (clauses 1-7 per `docs-mission.md`)
list of concrete drift items. Each finding should be small enough to be its own future queue row:
a one-line description, which clause it violates, and a rough severity (broken example / stale
version / missing page / orphaned page / etc).

Fix anything trivially in scope while sweeping (a stale version constant, an obviously broken
raw-loader import) — but the primary deliverable is the SCORED LIST, not a large content rewrite;
clause-5/6 taxonomy and benchmark work are separate, already-queued items (`docs-3`, `docs-4`).

Write the findings to `docs/docs-sync-findings.md` (a new, non-published-nav internal tracking
page is fine — it does not need a sidebar entry) so the mission controller can fold the highest-
severity rows into `docs-mission.md`'s Queue at Gate 4. Do not edit `design_docs/docs-mission.md`
or `design_docs/docs-mission-log.md` — that is the controller's job, outside this sprint's blast
radius.

Note on scope: the mission's `MISSION_PLANNER_ALLOWLIST` entry `docs/*` is a single-level glob
(not `docs/**`), so this brief only declares top-level `docs/` paths — do not plan edits to nested
files like `docs/src/constants/version.js` under this brief; a genuine version-constant fix found
during the sweep becomes its own queued item instead.

## Files

- `docs/docs-sync-findings.md`
