### Added — executor shadowing, M4: shadow finalization and the `ShadowComparison` record (2026-10-19)

M-AILANG-LANE-EXECUTOR-SHADOW M4. A completed shadow ends `completed`, never `pending_approval` (the design
assumed `wantsApproval=false` was enough, but the status is set independently and would have put shadows in
the approval queue and the landed-card sweep). Shadows create no approval card and no handoff, post no
completion notice to the agent's inbox, and never touch stage metrics or chain aggregates. A local worker never
runs a shadow. Both sides' results (status, base/head commit, diff, changed files, cost, tokens, turns, tool
calls, duration, session, artifact path, error) are written under the primary's ID by a new ledgered
finalization effect `shadow_record` into `ShadowComparison`
(`internal/coordinator/shadow_comparison.go`; SQLite table `shadow_comparisons`, Firestore collection
`shadow_comparisons`, both a transactional JSON merge). `base_drift` is set when the two runs cloned different
commits. The Firestore store is covered by an opt-in live test, run against the dev project.
