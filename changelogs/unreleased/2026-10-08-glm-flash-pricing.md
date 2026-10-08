### Fixed — GLM-5.3-Flash OpenRouter prices were half the current rate

The five OpenRouter-routed GLM-5.3-Flash rows (`pi-or-`, `motoko-or-`, `motoko-lane-or-`,
`opencode-or-` and `or-glm-5-3-flash`) declared $0.075/M input and $0.25/M output, verified 2026-08-27.
OpenRouter now lists $0.15/M input, $0.50/M output and $0.03/M cache reads (checked 2026-10-08).
Every cost figure for these rows since the change was understated by about half, including the
2026-10-02 lane A/B ($6.86 vs $4.20). The rows now carry the current prices and the cache-read rate,
so cache hits are no longer priced as fresh input.
