---
name: eval-gap-finder
description: Find AILANG vs Python eval gaps and improve prompts/language. Use when user says 'find eval gaps', 'analyze benchmark failures', 'close Python-AILANG gap', or after running evals.
---

# Eval Gap Finder

Automates the process of finding and closing the gap between Python and AILANG benchmark success rates. Identifies language limitations, prompt gaps, and missing stdlib functions.

## Quick Start

**Most common usage:**
```bash
# User says: "Find eval gaps" or "Analyze benchmark failures"
# This skill will:
# 1. Run evals at the Core tier with dev models (resolved dynamically from
#    internal/eval_harness/models.yml — no hardcoded model lists)
# 2. Compare Python vs AILANG Core pass rates (the headline metric)
# 3. Identify benchmarks where Python passes but AILANG fails, and
#    AILANG-only wins (keep-candidates) vs saturated (demote-candidates)
# 4. Analyze error patterns and categorize them
# 5. Check if gaps are documented in prompt
# 6. Test proposed examples and add to prompt
# 7. Create design docs for language limitations
```

## When to Use This Skill

Invoke this skill when:
- User asks to "find eval gaps" or "close the Python-AILANG gap"
- User wants to analyze benchmark failures
- After running evals and seeing lower AILANG success
- User says "why is AILANG failing?" or "improve AILANG benchmarks"
- User wants to identify language limitations

## Available Scripts

### `scripts/run_gap_analysis.sh [eval_dir] [--tier <tier>]`
Run full gap analysis on eval results. Accepts `--tier` (default `core`); runs a fresh
eval via `dev_models_csv` when no eval_dir is supplied.

```bash
# Analyse an existing baseline
.claude/skills/eval-gap-finder/scripts/run_gap_analysis.sh eval_results/baselines/v0.14.0

# Or run a fresh core-tier eval against dev models resolved from models.yml
.claude/skills/eval-gap-finder/scripts/run_gap_analysis.sh --tier core
```

### `scripts/identify_python_only.sh <eval_dir>`
List benchmarks where Python passes but AILANG fails.

```bash
.claude/skills/eval-gap-finder/scripts/identify_python_only.sh eval_results/baselines/v0.14.0
```

### `scripts/categorize_errors.sh <eval_dir>`
Categorize AILANG failures by error type.

```bash
.claude/skills/eval-gap-finder/scripts/categorize_errors.sh eval_results/baselines/v0.14.0
```

### `scripts/test_example.sh <code>`
Test if an AILANG code example compiles and runs correctly.

```bash
.claude/skills/eval-gap-finder/scripts/test_example.sh /tmp/test.ail
```

## Workflow

### 1. Run Evals at the Core Tier With Dev Models

Dev models are resolved from `internal/eval_harness/models.yml` — never hardcode.

```bash
# Source the shared helper to resolve dev models dynamically
. .claude/skills/_shared/scripts/eval_lib.sh

# Core tier is the headline metric; run it first
ailang eval-suite --models "$(dev_models_csv)" --tier core --output eval_results/gap-analysis
```

Run cheap/fast dev models first. If Core passes there, larger models should too. Only move
to `--tier stretch` or `--tier vision` once Core is healthy — those tiers are expected to
have mixed results and shouldn't drive prompt changes.

### 2. Generate Summary and Identify Gaps

```bash
ailang eval-summary eval_results/gap-analysis
.claude/skills/eval-gap-finder/scripts/identify_python_only.sh eval_results/gap-analysis
```

Key Core-tier metrics (from `latest.json` aggregates or `ailang eval-matrix`):
- **AILANG Core pass rate** — the headline number (Python is usually ~95%; gap is what we close)
- **Python-only wins** — benchmarks we can likely fix with prompt or language work
- **AILANG-only wins** — keep-candidates; evidence of AILANG value
- **Saturated** (both ≥ 95%) — demote-candidates; not revealing anything

### 3. Analyze Error Patterns

For each Python-only pass, categorize the error:

| Category | Pattern | Fix Approach |
|----------|---------|--------------|
| WRONG_LANG | Model wrote Python syntax | Stronger "NOT Python" in prompt |
| PAR_001 | Parse errors (syntax) | Add more examples to prompt |
| Type errors | Type unification failures | May be language limitation |
| Logic errors | Compiles but wrong output | Better examples or algorithm |
| EOF errors | Incomplete code generation | Model limitation, not prompt |

### 4. Check Prompt Coverage

For each gap, check if the pattern is documented in the current teaching prompt
(find the active file with `ls prompts/ | grep -E 'v0\.[0-9]+.*\.md'`):

```bash
grep -n "pattern" prompts/v0.14.x.md
```

If not documented, add:
- Working example to Quick Reference section
- Entry in "What AILANG Does NOT Have" table (if limitation)
- New section if pattern is complex

### 5. Test Examples Before Adding

**CRITICAL**: Always test examples before adding to prompt!

```bash
cat > /tmp/test.ail << 'EOF'
module benchmark/solution
-- Your example code here
EOF
ailang run --caps IO --entry main /tmp/test.ail
```

If example fails, it reveals a language gap - create a design doc instead.

### 6. Create Design Docs for Language Gaps

If testing reveals a language limitation:

1. Create design doc: `design_docs/planned/vX_Y_Z/m-<feature>.md`
2. Document:
   - Minimal reproduction
   - Error message
   - Workaround (for prompt)
   - Proposed fix
3. Add workaround to prompt with note

### 7. Track Improvement

After updates, re-run evals at the same tier used for analysis. Dev models come from
`models.yml`, so the command stays stable as the roster evolves:

```bash
(. .claude/skills/_shared/scripts/eval_lib.sh && \
  ailang eval-suite --models "$(dev_models_csv)" --tier core --output eval_results/gap-analysis-v2)
```

Compare:
- Core pass-rate improvement (Python vs AILANG gap)
- Which previously Python-only benchmarks now pass in AILANG
- Any regressions (previously-passing AILANG benchmarks now failing)
- Whether saturated/AILANG-only sets shifted — a sprint that adds three new saturated
  benchmarks is less valuable than one that adds one AILANG-only win

## Tier-First Workflow (v0.14.0+)

The v0.14.0 tier system lets us focus gap closing on where it matters. Walk through this
loop every time, and skip any step that doesn't apply.

1. **Run at `--tier core` first.** Core is the headline number. If Core is clean, stretch
   failures are *expected* and usually not worth prompt changes.
2. **Use `DetectAILANGOnlyWins` to identify keepers.** Benchmarks where AILANG beats
   Python by ≥10pp are the skill's value evidence — protect them from regressions.
   ```bash
   ailang eval-matrix --ailang-wins
   ```
3. **Use `DetectSaturation` to identify demotion candidates.** Benchmarks where both
   AILANG and Python hit ≥95% tell us nothing. They're not regression guards — a broken
   change will hit many benchmarks. Demote them to free up run time for signal.
   ```bash
   ailang eval-matrix --show-saturated
   ```
4. **Close gaps in Core before touching Stretch.** A failing Stretch benchmark is
   *allowed*; a failing Core benchmark means we regressed on the headline metric.
5. **Check tag coverage.** Gaps clustered in one tag (e.g., 4/5 failures in
   `recursion`) probably signal a language or prompt problem, not per-benchmark bugs.
   ```bash
   ailang eval-matrix --by-tags
   ```

This matches the skill's rotation philosophy below: close Core gaps, demote the obvious,
protect the wins.

## Benchmark Rotation Philosophy

The eval suite is **curated, not accumulated**. The goal is high-signal benchmarks —
ones that tell us something about relative performance across AILANG, Python, and agent
harnesses. Time-series continuity is secondary; a small core of stable benchmarks is
retained for regression guarding, but the rest should rotate as signal changes.

Before adding, removing, or proposing fixes for a benchmark, answer:

- **Is it saturated?** Both Python ≥ 95% AND AILANG ≥ 95% → demote to `stretch` or remove.
- **Is it an AILANG-only win?** AILANG ≥ Python by ≥ 10pp → keep and cite as value evidence.
- **Does it fill an under-covered tag?** Check `ailang eval-matrix --by-tags`; a new
  benchmark in a thin tag is more valuable than one more in a saturated tag.

When prompt changes fix a benchmark, that's a success — but record whether the
benchmark also became *saturated*. If yes, it's done its job; rotate it out.

## Error Categories Reference

| Error | Meaning | Fix |
|-------|---------|-----|
| WRONG_LANG | Wrote Python instead | Prompt emphasis |
| PAR_001 | Parser error | Syntax examples |
| PAR_UNEXPECTED_TOKEN | Wrong token | Syntax examples |
| TC_* | Type check error | Type examples or design doc |
| "undefined variable" | Missing import/letrec | Document pattern |
| EOF errors | Incomplete code | Model limitation |
| logic_error | Wrong output | Algorithm examples |

## Resources

### Gap Analysis Template
See [`resources/gap_analysis_template.md`](resources/gap_analysis_template.md) for structured analysis format.

### Common Patterns
See [`resources/common_patterns.md`](resources/common_patterns.md) for frequently encountered gaps.

## Progressive Disclosure

This skill loads information progressively:

1. **Always loaded**: This SKILL.md file (workflow overview)
2. **Execute as needed**: Scripts in `scripts/` directory
3. **Load on demand**: Resources for templates and patterns

## Notes

- Always test examples before adding to prompt
- Prefer fixing language over prompt workarounds
- Track improvements with before/after eval runs at the **same tier**
- Create design docs for language limitations
- Update prompt hash in versions.json after changes
- Dev models come from `internal/eval_harness/models.yml` via
  `.claude/skills/_shared/scripts/eval_lib.sh` — never hardcode model IDs
- The eval suite is curated, not accumulated — see Benchmark Rotation Philosophy above
