### Added — per-model OpenRouter routing (`openrouter_provider`) and `AILANG_OPENROUTER_PROVIDER`

- A `models.yml` row can now carry `openrouter_provider`: OpenRouter `provider` routing options
  (`only`, `order`, `ignore`, `sort`, `preferred_min_throughput`, …). They are sent on every
  OpenRouter request for that `api_name`, from whichever harness. Precedence: registry defaults,
  then the caller's routing policy, then `AILANG_OPENROUTER_PROVIDER`, a JSON object for
  experiments such as `'{"sort":"throughput"}'`. `ignore` lists are unioned; a malformed override
  fails the request. One block per `api_name`, enforced by registry validation.
- The settings are installed on both AI setup paths. `ailang run` (motoko's runtime) uses
  `prepareRunAIBinding`, not `setupAIHandler`, so an earlier version of this change never reached
  motoko.
- GLM-5.3-Flash routes over pi's effective host pool: the 13 hosts that accept pi's 943,717-token
  output request. With its 32,000-token request, AILANG drew from all 32 hosts, so motoko
  compared against pi on different hardware. That pool included `inference-net/fp4` (19% cache
  hits, tool calls contradicting the model's own reasoning). Measured on 2026-10-13, per call: Crusoe
  1.2 s and 115 tok/s; OpenInference 25.7 s and 10 tok/s.
