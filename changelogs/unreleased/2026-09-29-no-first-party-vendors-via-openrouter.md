### Changed — Anthropic, OpenAI and Google models are never reached through OpenRouter (2026-09-29)

This is a rule now, enforced by `TestModels_FirstPartyVendorsNeverViaOpenRouter`.
Open-weights releases (gemma, gpt-oss) are exempt. Three rows that reached Claude via
OpenRouter were retired: `motoko-claude-sonnet-4-6` (it leaves harness_suite, which drops to
7), `motoko-claude-haiku-4-5` and `motoko-or-sonnet-5`. motoko therefore has no Claude
comparison lane. The model-manager skill is updated to match. Its `.claude` and `.agents`
copies had diverged, and one had "Claude"→"Codex" rewrites in model names. They are merged
into one identical copy. The merge also fixes the persist command, documents where the
credentials live (`~/.config/ailang/secrets.env`), and adds an onboarding checklist for
Anthropic models.
