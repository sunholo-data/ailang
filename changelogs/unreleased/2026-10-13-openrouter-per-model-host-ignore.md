### Added — per-model OpenRouter host exclusions (`openrouter_ignore`)

- `models.yml` rows can list OpenRouter upstream provider slugs never to route that `api_name`
  to. `ailang run` sends them as `provider.ignore` on every OpenRouter request for that model,
  merged with any routing policy the caller sets. Rows sharing an `api_name` are unioned, so one
  entry covers the pi, motoko and opencode rows.
- GLM-5.3-Flash now ignores `inference-net`. On 2026-10-12, with the same task at the same
  time, `inference-net/fp4` cached 8% of input against 91–97% on Friendli, and its tool calls
  contradicted the model's own reasoning for 25+ steps. 102 of motoko's 141 calls landed there,
  and none of pi's did.
