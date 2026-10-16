### Changed — OpenRouter requests default to the fastest host

Every OpenRouter request now sends `provider.sort: "throughput"` unless the registry, the caller's routing policy or `AILANG_OPENROUTER_PROVIDER` chose a `sort` or an explicit `order`. OpenRouter's own default balances toward the cheapest host, and the cheapest hosts are the slow ones. In the 2026-10-16 GLM-5.3-Flash A/B, motoko was routed to open-inference and morph, and with those excluded, to Near AI at 12–20 tok/s: one 8k-token reasoning call took about 700 s against about 38 s on CoreWeave. The fast hosts cost up to about 2x per token on models with cheap hosts. To opt out per run, set `AILANG_OPENROUTER_PROVIDER='{"sort":"price"}'`.

### Fixed — OpenRouter `Generate` routes like `Step`

`Generate` built its `provider` block from the caller's policy alone, so it ignored the registry's per-model `openrouter_provider` and the `AILANG_OPENROUTER_PROVIDER` override. It now uses the same `routeFor` as `Step` and `StreamStep`.
