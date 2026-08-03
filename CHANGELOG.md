# AILANG Changelog

For the latest version, see [changelogs/v0.18-current.md](changelogs/v0.18-current.md).

## v0.32.0 (Unreleased)

- Added experimental `std/ai.stepWithStreamRecorded`, originally authored by
  [@arniwesth](https://github.com/arniwesth). It preserves immediate stream
  callbacks while returning the exact ordered adapter-emitted chunk log and
  typed terminal outcome, including an explicit incomplete prefix on an
  unencodable chunk. See [#546](https://github.com/sunholo-data/ailang/issues/546)
  and [arniwesth/ailang#2](https://github.com/arniwesth/ailang/pull/2).

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
