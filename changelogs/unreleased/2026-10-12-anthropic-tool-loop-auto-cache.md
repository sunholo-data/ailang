### Changed — Anthropic tool-loop requests use automatic prompt caching

- `std/ai` step calls on direct Anthropic that carry tools now send the request-level
  `cache_control: {type: "ephemeral"}` marker, so the cache point moves to the latest message and
  each turn reads its prior history from cache. Before, only a declared system breakpoint was
  cached: motoko on Haiku 5.5 billed 3.6M uncached input tokens on one real task where pi, which
  marks the latest message, billed 66. Requests without tools are unchanged on the wire.
