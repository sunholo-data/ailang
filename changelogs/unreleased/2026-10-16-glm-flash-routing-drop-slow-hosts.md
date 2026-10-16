### Fixed — GLM-5.3-Flash routing no longer lands on the two slowest hosts

The registry's `openrouter_provider.only` for `z-ai/glm-5.3-flash` listed 13 hosts meant to match pi's pool. Inside it OpenRouter's price weighting picked the two cheapest, open-inference (~10 tok/s) and morph (~5 tok/s): 72 of 72 motoko calls in the 2026-10-16 A/B went there (16.4 s median per call), while pi reached neither in 100 calls (2.0 s median). Both harnesses produced the same ~155 output tokens per call, so motoko's 9-minute median run against pi's 30 s came from the host. Both are dropped; 11 hosts remain.
