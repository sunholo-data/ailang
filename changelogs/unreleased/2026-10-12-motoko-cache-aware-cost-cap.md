### Fixed — motoko's in-run cost cap prices prompt-cache tokens

- The motoko executor now passes the budget's cache-read and cache-write rates to motoko
  (`MOTOKO_COST_CACHE_READ_PER_1M_MILLICENTS`, `MOTOKO_COST_CACHE_WRITE_PER_1M_MILLICENTS`), and
  the pinned motoko build prices cache tokens at those rates. Before this, on OpenRouter-style
  routes (which count cached tokens inside input), every cached token was charged at the full
  input rate, so GLM-5.3-Flash lane runs stopped at a self-reported $2.00 when OpenRouter had billed
  about $0.45. On direct Anthropic, cache tokens were not priced at all.
- Rate conversion rounds instead of truncating: $0.15/M used to reach motoko as 14,999 millicents.
- The forwarding moved from `motoko.go` into `costRateEnv` in `motoko/cost.go`.
- `docker/Dockerfile.agent-motoko` pins motoko_agent `84fde7b7`, which carries the cache-aware meter and forwards its rate variables to the runtime.
