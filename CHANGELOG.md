# AILANG Changelog

For the latest version, see [changelogs/v0.18-current.md](changelogs/v0.18-current.md).

## [Unreleased]

### Added

- `serve-api --no-feedback-tool` can suppress the built-in
  `submit_feedback` MCP tool for an exact caller-selected surface. Existing
  invocations keep the tool by default (refs #498).

- Managed Agents: live contract probe for the 2026-07 feature drop
  (`internal/executor/managed_agents/managed_agents_features_live_test.go`,
  ADC-gated behind `AILANG_LIVE_MA_FEATURES=1`, never run in CI). Records what
  the Gemini Developer API docs advertise vs what **Vertex** — the surface this
  executor uses — actually does. Headline: `agent_config` is now accepted on
  Vertex (the earlier "unknown field" was a missing `"type"` discriminator, not
  an absent feature), but `max_total_tokens` is **validated and then ignored**
  (a cap of 64 ran to completion at 216,843 tokens), `agent_config.model`
  accepts a nonexistent model id and is not echoed anywhere on the wire, hooks
  have no delivery path (`inline` sources still rejected; `/` not writable;
  a self-installed `/workspace/.agents/hooks.json` is never consulted), and the
  Environments API returns `Method not found`. Findings and their evidence are
  tabulated in `internal/executor/managed_agents/README.md`.

### Fixed

- A2A `tasks/send` now applies the same `--routes-only` / `@noexpose`
  projection as the agent card, so hidden functions are not callable and
  remain indistinguishable from absent functions (refs #528).
- SMT verification now closes declarations through named record fields into
  user ADTs, including `list[ADT]` and mutual record/ADT cycles.
- `ai-check` now uses the same per-function type-demand filtering as `verify`,
  so unrelated declarations no longer cascade otherwise verifiable functions
  into skips. This changes the verification KPI population (the measured corpus
  moved from 76→79 verified and 10→7 skipped) and must not be read as a model
  quality improvement.

### Changed

- **Breaking for out-of-repo shell callers:** `ai-check` now exits 1 when its
  emitted JSON reports `verify.errors > 0`; complete JSON is still written
  before the exit decision. Skips remain exit 0.

- Bounded both Z3 solver calls (`Solve` and the `Z3Version` header probe) by
  killing their process groups and reaping them after a hard deadline. `Solve`
  uses `max(configured timeout, effective -T seconds) + 2s grace` and preserves
  the existing `StatusUnknown` / `"solver timeout"` result shape (Fixes #510).

- Added `tools/nightly_classify.py`, a durable variance guard for nightly evals.
  It records history in `~/.ailang/state/nightly-eval-history.jsonl`, labels noisy
  flips `SUSPECTED-FLAKE` or `INSUFFICIENT-HISTORY`, and requires the explicit
  `--bootstrap` flag to seed missing history.
- Nightly classification now labels runs `INVALID` when the any-trial
  infrastructure-taint fraction reaches `--invalid-infra-fraction` (default
  0.30), suppressing verdicts without confusing infrastructure outages with
  compiler regressions. History rows may carry a `validity` field, with an
  absent field meaning valid; `--mark-invalid` backfills affected dates in
  `~/.ailang/state/nightly-eval-history.jsonl` without deleting evidence.

- CI/infra: renamed the documentation build check to `docs-build`, added an
  always-reporting `docs-gate`, and scoped workflow concurrency per ref while
  retaining singleton Pages deployments (#497).
- Added end-to-end seeded and crypto Rand examples, backed by asymmetric
  validation-only subsumption from explicit `seeded`/`crypto` declarations to
  bare/os requirements.
- Eval docs: de-staled the `post-release` skill and `models.yml` suite headers
  against the current roster. The skill still named `claude-opus-4-8`,
  `or-glm-5-1`/`opencode-or-glm-5-1` and "7 production models" for an
  `extended_suite` that is now 18, and claimed the motoko harness "currently
  hangs" — obsolete since it was re-added to `ollama_suite` 2026-06-15. Also
  corrected the tier breakdown (19 core + 21 stretch + 16 frontier, not
  19/29/8), the `harness_suite` count (8, not 6), consolidated the three
  contradictory cost tables into one flagged-stale table, and documented the
  Anthropic-quota coverage-hole failure mode that produced v0.30.0's 43 missing
  standard rows and zero Claude agent rows.

## Changelog Archives

The full changelog has been split into themed files for searchability and readability:

| File | Versions | Theme |
|------|----------|-------|
| [v0.18-current.md](changelogs/v0.18-current.md) | v0.18.0+ | Eval Harness, Extensions & Agent Loop |
| [v0.10-v0.17-bytecode-vm.md](changelogs/v0.10-v0.17-bytecode-vm.md) | v0.10.0–v0.17.0 | Bytecode VM & Runtime |
| [v0.9-cloud-pubsub.md](changelogs/v0.9-cloud-pubsub.md) | v0.9.0–v0.9.12 | Cloud Integration & Pub/Sub |
| [v0.8-cloud-features.md](changelogs/v0.8-cloud-features.md) | v0.8.0–v0.8.1.1 | Cloud Features & Advanced Coordinator |
| [v0.7-observatory.md](changelogs/v0.7-observatory.md) | v0.7.0–v0.7.3 | Observatory, Chains & Dashboard |
| [v0.6-coordinator.md](changelogs/v0.6-coordinator.md) | v0.6.0–v0.6.2 | Coordinator Daemon & Agents |
| [v0.5-ai-providers.md](changelogs/v0.5-ai-providers.md) | v0.5.0–v0.5.10 | AI Providers, Eval Harness & Search |
| [v0.4-monomorphization.md](changelogs/v0.4-monomorphization.md) | v0.4.0–v0.4.10 | Monomorphization & DX Improvements |
| [v0.3-core-language.md](changelogs/v0.3-core-language.md) | v0.3.0–v0.3.25 | Core Language Stabilization |
| [v0.0-v0.2-foundation.md](changelogs/v0.0-v0.2-foundation.md) | v0.0.1–v0.2.1 | Foundation (Initial Release through Modules) |

All archives are indexed by `ailang docs search` for full-text and neural search.
