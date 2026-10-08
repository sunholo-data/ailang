### Changed — motoko cloud image pins `e3fa68ec`: compaction fires for fleet models

The pinned motoko commit adds `agent.context_limit: 262144` to the `cloud`, `ailang_only`,
`ollama_microrag`, `ollama` and `dogfood` profiles. motoko's model catalogue has none of the fleet's
models, so the context limit resolved Unknown and `compaction_ai` (which compacts at 75% of the
window) never ran. A real lane task reached 260k tokens of context over 157 steps, about $0.92 for
one bug fix. The profile value wins for any model, so compaction now starts at about 196k.
