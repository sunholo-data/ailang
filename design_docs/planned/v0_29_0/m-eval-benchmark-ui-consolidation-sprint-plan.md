# Sprint Plan — M-EVAL-BENCHMARK-UI-CONSOLIDATION

**Goal:** Make the benchmark site coherent and correct — finish the standard/agent split, prune the accreted cloud dashboard, give OS/local (cross-language + harness + longitudinal) a real zero-cost home, and add a framing landing page.
**Design doc:** [m-eval-benchmark-ui-consolidation.md](../m-eval-benchmark-ui-consolidation.md)
**Mode:** overnight autonomous execution (2026-06-12 night) — committed per-phase on `dev` for morning review.
**Risk:** medium — frontend (React/MDX) auto-deploys via docusaurus-deploy; verify each push's build. Phase C's *data* needs a local-rig run (offline) → scaffold + placeholder if the rig isn't driven tonight.

## Execution order & milestones

### M-A — Standard/agent split correctness (Go + verify) · ~0.5d
- Audit `internal/eval_analysis/export_json*.go` + `BenchmarkDashboard/*` for any remaining mode-mixing (beyond today's models-map fix).
- Add a Go test on `ExportBenchmarkJSON`: `models` are ran-standard, `agentModels` ran-agent, no empty/zero entries from mixing.
- **Accept:** test green; regenerate latest.json; no zero rows from mode-mix.

### M-B — Prune cloud dashboard + delete orphans · ~1d
- Delete orphan components: `LanguageLeaderboard`, `HarnessComparisonTable`, `DimensionSelector`, `BenchmarkOverview` (keep `BenchmarkMini`).
- Trim the Model Leaderboard to headline + per-model table + one trend; demote secondary metrics behind a toggle.
- **Accept:** build green; page is scannable; orphans gone.

### M-C — OS/Local section data + component · ~2d (data may be rig-blocked)
- Define static OS rotation JSON schema (per benchmark×model×lang×harness, N-trial pass-rate + trend).
- Build `OSLocalLeaderboard` React component (reuse EloLeaderboard pattern); renders AILANG/Python/JS/Go × harness; placeholder when no data.
- Wire `eval-publish` to emit to `docs/static/benchmarks/os/`. (Real data = a local rotation; scaffold tonight, populate when rig runs.)
- **Accept:** component renders from a sample/real OS JSON; degrades cleanly when absent.

### M-D — Benchmarks Overview landing + grouped nav · ~0.5d
- New `benchmarks/index` overview page framing Cloud-frontier vs OS/Local; group sidebar accordingly; fold Explorer under OS/Local.
- **Accept:** coherent first-screen; build green.

## Overnight guardrails
- Commit each milestone separately; push; confirm docusaurus-deploy green before next.
- NEVER break the build for the morning: if a phase can't be verified (build red), revert that phase's commit and leave a note in the sprint JSON.
- Do NOT touch prod/MCP/release; this is docs + eval_analysis only.
- Phase C data generation requires the local rig — if not driven tonight, ship the component + schema + placeholder and mark C "data pending" in the JSON.

## Morning review surface
- `dev` commits M-A..M-D; `.ailang/state/sprints/sprint_M-EVAL-BENCH-UI.json` shows passes/notes per milestone; live site reflects each green deploy.
