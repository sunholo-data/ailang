### Added — `long-frontier` benchmark tier (attended evals only)

- New tier `long-frontier` for benchmarks whose agent runs routinely exceed the 1h wall
  clock. `quine`, `gauntlet_10`, `legal_obligation_engine` (frontier) and
  `commonmark_emphasis` (stretch) move there: 38–83% of their local qwen3.8-27b agent
  runs timed out, ~48 rig-hours in one week, recorded as `duration_ms=0`.
- No scheduled job lists the tier, and `--benchmarks-by-confidence` now skips it, so it
  runs only on request: `ailang eval-suite --tier long-frontier` (standard) or an explicit
  `--benchmarks` list (agent). Tier counts: stretch 24, frontier 5, long-frontier 4.

### Changed — OS rotation filler: cross-language pass off by default

The python/js/go hand-off only ever looked off because AILANG coverage never completed.
With the long-running benchmarks out, coverage can complete, so the hand-off is now
opt-in (`OS_FILLER_CROSS_LANG=1`); a fully covered version leaves the rig idle.
