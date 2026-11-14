# pi local (ollama-rig) lanes run with no output cap on the wire

- **Date**: 2026-11-05
- **Class**: bug
- **Recommend**: design-doc
- **Searched**: `max_output_tokens` in design_docs/, `PI_WIRE_CLAMP`, `943717`, `gemma4`, `finish_reason` in design_docs/planned/, `per-turn output budget`; code: `internal/executor/pi/`, `internal/eval_harness/pi_models_config_test.go`, `scripts/check_pi_wire_budget.sh`
- **Duplicate coverage**: the finish_reason half of the ask is already ruled on by `design_docs/planned/m-eval-reasoning-model-fairness.md` (per-turn `finish_reason` capture; "truncation hides in summary finish_reasons"). The uncapped-wire half is **not** covered anywhere — none found.

## Reason

The report's framing ("make pi honour the lane's max_output_tokens") is half already-settled and half a real, verified gap:

**Settled:** pi DOES honour the lane cap — `buildBaseOptions` in pi-ai (`dist/providers/simple-options.js`) sends `min(model.maxTokens, 32000)`, root-caused on the wire 2026-08-26 and documented in the canonical `PI_WIRE_CLAMP` note on the `pi-or-deepseek-v4-flash` row of `internal/modelreg/models.yml` (~:4766, "THIS NUMBER DOES NOT REACH THE WIRE BY ITSELF"). The 943717 the report remembers was the OpenRouter catalogue-fallback case, fixed by the config preflight.

**The gap:** the cap only exists if the model has an entry in pi's `models.json`. `tools/pi-extensions/models.mission.json` (canonical copy) contains **no `gemma4:12b-mxfp8` entry** — only `gemma4:26b` (8192) and the qwen3.5/3.6 rows. With no entry, `buildBaseOptions` leaves `maxTokens` `undefined` → **no cap is sent at all**, and Ollama's OpenAI-compat endpoint defaults `num_predict` to unbounded. That is exactly what the measurements show: one request running 2,106 s (~97k output tokens at 46 tok/s) is impossible under any 32000 clamp; it is what an uncapped local lane looks like. The 218k/243k per-benchmark totals are multi-turn sums on top of that.

**Why every guard stayed green — all three are OpenRouter-scoped:**
1. `TestPiModelsConfigMatchesRegistry` (`internal/eval_harness/pi_models_config_test.go`) iterates pi-config models and looks each up in the registry; a registry row **missing from pi's config** is never visited (`continue` — "Registered in pi but not in models.yml"). The check runs in one direction only, so the missing entry passes silently.
2. `checkInstalledModelConfig` (`internal/executor/pi/config_preflight.go`, `checkInstalledModelConfig`) returns nil for every `provider != "openrouter"` — ollama-rig is unchecked, so installed-copy drift (the 2026-10-16 precedent) is invisible here too.
3. `scripts/check_pi_wire_budget.sh` declares its own scope: OpenRouter only (Broadcast traces; local lanes produce no trace).

There is also more than one acceptable remedy, which is what forces the design doc rather than a direct fix: (a) add the missing entry + extend the two config guards to ollama-rig (cheap, but cannot see installed drift on the rig), (b) enforce a per-turn budget harness-side from pi's NDJSON stream, and/or (c) a local-lane analogue of the wire check (pi --mode json emits usage per event; Ollama logs the request). The finish_reason banking half should be folded into m-eval-reasoning-model-fairness rather than designed twice — cite that doc, don't duplicate it.

Design doc should name: `tools/pi-extensions/models.mission.json`, `internal/eval_harness/pi_models_config_test.go` (TestPiModelsConfigMatchesRegistry — add the reverse direction: every pi-lane registry row must exist in pi's config), `internal/executor/pi/config_preflight.go` (drop the openrouter-only gate or add an ollama-rig arm), and the models.yml `PI_WIRE_CLAMP` note as the mechanism record.
