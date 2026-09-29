# Fleet Mission Dashboard

Snapshot as of iteration 7 — 2026-09-29. History: [charter](fleet-mission.md) and [log](fleet-mission-log.md).

- **Latest release**: v0.48.0. The fleet does not release.
- **This iteration**: D-FLEET-7 stall-watchdog M2 merged in [#1399](https://github.com/sunholo-data/ailang/pull/1399), `ce1c0639f`; independent Sonnet PASS 88, zero blocking; merge-commit CI SUCCESS (CI run 36562795612; 20 check rows settled success/skipped).
- **Parked**: heartbeat relative-path ticket after two blocked quorums. D-FLEET-8 asks whether to update both tracked skill copies. No heartbeat implementation landed.
- **Next**: follow the charter's 2026-09-29 human-ranked queue; do not re-sort it by `slots_lost`.
- **Open tickets**: 16.
- **Loop**: `dev.ailang.mission-fleet`, every 6h, pinned to `origin/dev`; 0-ticket fire exits before agent spend.
- **Routing**: designer gpt-6-astra (Agent fallback), planner/executor gpt-6-sol (Agent), independent evaluator Sonnet 5.5 (`claude-sub` fallback after Agent model rejection).
- **Cost**: quorum about $0.24 metered; role token totals unavailable; Codex and Anthropic subscription buckets.
