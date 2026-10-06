### Changed — OS rotation filler: heavy tier for chronic timeouts; cross-language pass off by default

- `quine`, `gauntlet_10`, `commonmark_emphasis` and `legal_obligation_engine` leave the
  regular local rotation and its coverage set (28-day timeout rates 83/43/41/38% on the
  qwen3.8-27b trio, ~48 one-hour timeouts in one week). They run as their own lap: one
  benchmark per cycle, 1 trial, at most once every 7 days (`OS_FILLER_HEAVY`,
  `OS_FILLER_HEAVY_DAYS`; `OS_FILLER_HEAVY=""` restores the old behaviour).
- The cross-language (python/js/go) hand-off is now off unless `OS_FILLER_CROSS_LANG=1`.
  It had only looked off because AILANG coverage never completed; with the heavy tasks
  out, coverage can complete, and the rig now goes idle instead of starting other languages.
