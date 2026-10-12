### Fixed — system-role messages reach Anthropic and Gemini; lane `write` creates parent directories

- `std/ai` multi-turn calls on Anthropic and Gemini dropped every system-role message: those
  providers take one top-level system field, and the step builders kept only
  `Request.SystemPrompt`. motoko delivers its whole system prompt as a system-role message, so on
  direct Anthropic the model never saw it (measured 2026-10-12: Haiku 5.5 billed ~3.2k input
  tokens per call against a 102k-char system prefix). New `ai.Request.SystemText()` joins
  `SystemPrompt` and the system-role messages; Anthropic, Gemini and the ChatGPT provider all
  use it.
- `ailang policy-tool` `write` now creates missing parent directories inside the sandbox root.
  The ailang_only lane has no mkdir, so a first write to `.ailang-scratch/probe.ail` (where the
  lane prompt steers probes) failed with "no such file or directory".
