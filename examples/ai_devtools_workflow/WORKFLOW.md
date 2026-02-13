# AI Developer Workflow with AILANG Devtools

This directory demonstrates how an AI agent uses AILANG's v0.8.0 devtools
to write, verify, trace, and iterate on code.

## The Module

`discount_calculator.ail` — A loyalty discount calculator with:
- ADT type (`Tier = Bronze | Silver | Gold | Platinum`)
- 3 contract-annotated functions (`requires`/`ensures`)
- Cross-function verification (Z3 inlines `discountPercent` into `applyDiscount`)
- IO effects for output

## Step-by-Step Workflow

### 1. Get Syntax Reference

```bash
ailang prompt > /tmp/syntax_ref.md
# 1,596 lines (~49KB) — full language syntax for AI context injection
```

### 2. Get Devtools Reference

```bash
ailang devtools-prompt > /tmp/devtools_ref.md
# 743 lines (~27KB) — all CLI commands documented
```

### 3. Type-Check

```bash
ailang check examples/ai_devtools_workflow/discount_calculator.ail
# Output: "✓ No errors found!" (exit 0)
# On error: human-readable error with PAR_/TC_ codes (exit 1)
```

**Note**: No `--json` flag available. AI must parse text output.

### 4. Verify Contracts with Z3

```bash
# Human-readable summary
ailang verify examples/ai_devtools_workflow/discount_calculator.ail
# Output: 3 verified

# Machine-readable JSON (recommended for AI)
ailang verify --json examples/ai_devtools_workflow/discount_calculator.ail
# Output: {"verified": 3, "counterexample": 0, ...}

# Show SMT-LIB (for debugging)
ailang verify --verbose examples/ai_devtools_workflow/discount_calculator.ail
```

### 5. Run with Execution Traces

```bash
# Normal run
ailang run --caps IO --entry main examples/ai_devtools_workflow/discount_calculator.ail

# Collect JSONL trace (stdout=trace, stderr=program output)
ailang run --emit-trace jsonl --caps IO --entry main \
  examples/ai_devtools_workflow/discount_calculator.ail > trace.jsonl

# With budget report
ailang run --budget-report json --caps IO --entry main \
  examples/ai_devtools_workflow/discount_calculator.ail
```

### 6. Verify Determinism via Replay

```bash
# Compare re-execution against baseline
ailang replay trace.jsonl
# Output: "✓ REPLAY MATCHES (28/28 events identical)"

# JSON for programmatic check
ailang replay --json trace.jsonl
# Output: {"match": true, "baseline_events": 28, "replay_events": 28}
```

### 7. Score Trace Quality for Training

```bash
# Human summary
ailang export-training --score trace.jsonl
# Score: 0.66 (completion: 1.00, complexity: 0.57, contracts: 0.50, ...)

# JSON for programmatic use
ailang export-training --score --json trace.jsonl
# Full breakdown with function counts and effect stats
```

### 8. Fix Bugs, Re-Verify

When `verify --json` returns a counterexample:

```json
{
  "function": "brokenDiscount",
  "status": "counterexample",
  "model": [{"name": "price", "sort": "Int", "value": "1"}]
}
```

The AI knows: `price=1` violates the postcondition. It reads the model,
understands the function arithmetic, and fixes the bug.

## The Full Loop

```
                    ┌──────────────────────┐
                    │  1. Write AILANG code │
                    └──────────┬───────────┘
                               │
                    ┌──────────▼───────────┐
                    │  2. ailang check      │──── Parse error? Fix syntax
                    └──────────┬───────────┘
                               │ ✓
                    ┌──────────▼───────────┐
                    │  3. ailang verify     │──── Counterexample? Fix logic
                    │     --json            │
                    └──────────┬───────────┘
                               │ ✓ verified
                    ┌──────────▼───────────┐
                    │  4. ailang run        │──── Runtime error? Fix effects
                    │     --emit-trace jsonl│
                    └──────────┬───────────┘
                               │ ✓
                    ┌──────────▼───────────┐
                    │  5. ailang replay     │──── Non-determinism? Add --seed
                    │     --json            │
                    └──────────┬───────────┘
                               │ ✓ match
                    ┌──────────▼───────────┐
                    │  6. export-training   │──── Low score? Add contracts,
                    │     --score --json    │     effects, complexity
                    └──────────┬───────────┘
                               │ score >= 0.7
                               ▼
                          ✅ Ship it
```

## Files

| File | Purpose |
|------|---------|
| `discount_calculator.ail` | Toy module with contracts and effects |
| `WORKFLOW.md` | This file — step-by-step commands |
| `learnings.md` | Friction points and improvement suggestions |
