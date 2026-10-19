### Added — executor shadowing, M1: the `shadow:` agent block and `ShadowOf` (2026-10-19)

M-AILANG-LANE-EXECUTOR-SHADOW M1. Agents gain an optional `shadow:` block (`internal/coordinator/shadow_config.go`:
`enabled`, `provider`, `executor_variant`, `model`, `sample_rate`, `max_cost_usd`, `offer_alternative`),
validated at config load by `ValidateAgentProviders`: only `ailang_only` agents, the shadow provider must
differ from the primary's, the variant must be able to run it (same rule as the agent's own declaration),
`sample_rate` in [0, 1]. A disabled block is validated too. Task records gain `ShadowOf` (the primary a
shadow child mirrors), persisted in SQLite (`shadow_of` column, additive migration) and Firestore; existing
records read as primaries. Nothing dispatches shadows yet (M2). Design:
`design_docs/planned/v0_55_0/m-ailang-lane-executor-shadow.md`.
