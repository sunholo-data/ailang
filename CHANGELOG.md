# AILANG Changelog

For the latest version, see [changelogs/v0.18-current.md](changelogs/v0.18-current.md).

## [Unreleased]

### Fixed

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

- CI/infra: renamed the documentation build check to `docs-build`, added an
  always-reporting `docs-gate`, and scoped workflow concurrency per ref while
  retaining singleton Pages deployments (#497).
- Added end-to-end seeded and crypto Rand examples, backed by asymmetric
  validation-only subsumption from explicit `seeded`/`crypto` declarations to
  bare/os requirements.

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
