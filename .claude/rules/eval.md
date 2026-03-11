---
paths:
  - "internal/eval_harness/**"
  - "internal/eval_analysis/**"
  - "benchmarks/**"
  - "eval_results/**"
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

## Adding Builtin Functions

Use the `builtin-developer` skill. Validation: `ailang doctor builtins`.
