---
paths:
  - "internal/eval_harness/**"
  - "internal/eval_analysis/**"
  - "benchmarks/**"
  - "eval_results/**"
  - "internal/modelreg/**"
---

# Evaluation Suite Rules

## M-EVAL: AI Evaluation

Use the `eval-analyzer` skill or `ailang eval-*` commands. Two modes: Standard (0-shot API) and Agent (agentic CLI).

**CRITICAL:** `ailang eval-suite` OVERWRITES the output directory. Run all models in ONE command:
```bash
ailang eval-suite --models gpt5,claude-sonnet-4-5,gemini-2-5-pro  # Correct
```

**Dashboard updates preserve history automatically:**
```bash
ailang eval-report eval_results/baselines/v0.3.10 v0.3.10 --format=json  # Correct
# DON'T redirect stdout — bypasses history preservation
```

**Full guide**: See `docs/docs/guides/evaluation/`

**NEVER route Anthropic, OpenAI or Google (Gemini) models through OpenRouter** (Mark, 2026-09-29).
We have direct lanes for all three — claude CLI subscription, codex subscription, Vertex — so an
OpenRouter route is metered spend on a model we already pay for. When a direct lane fails, fix the
lane; do not add an `or-` twin. Open-weights releases (google/gemma-*, openai/gpt-oss-*) are fine.
Enforced by `TestModels_FirstPartyVendorsNeverViaOpenRouter` (internal/modelreg).

**Claude CLI rows pin the full model id**, never `sonnet`/`opus`/`fable` — the aliases re-point
at the newest model. Enforced by `TestModels_ClaudeCLIRowsNeverUseAnAlias`.

## Adding Builtin Functions

Use the `builtin-developer` skill. Validation: `ailang doctor builtins`.
