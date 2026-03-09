# M-DX-AGENT-EVAL-GAPS: Agent DX Improvements from v0.9.0 Eval Analysis

**Status**: Planned
**Target**: v0.9.1
**Priority**: High
**Estimated Effort**: 1-2 days

## Problem Statement

Analysis of the v0.9.0 eval baseline (552 standard runs, 132 agent stages) reveals recurring patterns where AI agents fail to generate correct AILANG code. These failures fall into 4 categories, all fixable with targeted DX improvements.

### Eval Data Summary

| Eval Type | AILANG | Python | Gap |
|-----------|--------|--------|-----|
| Standard (0-shot) | 84.0% | 75.7% | +8.3pp |
| Agent (multi-turn) | 68.2% | 81.8% | -13.6pp |

The agent gap is misleading — 93% of AILANG agent failures are Gemini API errors (timeouts, crashes), not language DX issues. Claude Sonnet achieves **92% on AILANG** in agent mode.

The standard eval gap favors AILANG, but 24 Python failures stem from a too-minimal system prompt (separate fix, already applied to `ai_provider.go`).

## Gap Analysis

### Gap 1: Markdown Fence Stripping (24 standard failures)

**Symptom**: `PAR_NO_PREFIX_PARSE at line 1:1 ILLEGAL`

**Root cause**: Models (especially Gemini) wrap output in ` ```ailang ... ``` ` fences. The `extractCodeFromMarkdown()` function in `ai_provider.go` handles basic fences, but fails on:
- Language-tagged fences: ` ```ailang ` (note the language tag)
- Double-fenced output: outer markdown + inner code fence
- Non-standard fence variants

**Evidence**: 24 of 41 AILANG compile errors are PAR_NO_PREFIX_PARSE, and manual inspection confirms markdown fences in the generated code.

**Affected models**: Primarily Gemini (gemini-3-pro, gemini-3-1-pro), occasionally Claude Sonnet.

**Fix**: Improve `extractCodeFromMarkdown()` to handle language-tagged fences (` ```ailang`, ` ```python`, ` ```typescript`, etc.) and strip multiple layers of fencing.

**Files**: `internal/eval_harness/ai_provider.go`

### Gap 2: List Concatenation Operator `++` (6+ failures, 100% failure on type_unify)

**Symptom**: `type unification failed: cannot unify list[(string, Type)] with string`

**Root cause**: Models universally assume `++` works on lists (like Haskell, Erlang, Elixir). In AILANG, `++` is string-only concatenation. There is no infix list concatenation operator — users must use `std/list.append(xs, ys)`.

**Evidence**: ALL 6 models fail `type_unify` benchmark. In every case, the model writes `s1 ++ s2` where `s1` and `s2` are `[(string, Type)]` lists. This is not a model intelligence failure — it's a reasonable assumption that AILANG's type system rejects.

**Impact**: This also affects `config_file_parser`, `graph_bfs`, and other benchmarks where models attempt list operations with `++`.

**Options**:

| Option | Pros | Cons |
|--------|------|------|
| A. Add `++` as polymorphic list concat | Natural, matches model expectations, zero prompt change | Operator overloading complexity, potential ambiguity with string `++` |
| B. Add `+++` as list concat (separate op) | No ambiguity, simple implementation | Models won't know to use it without prompt update |
| C. Prompt-only fix: add warning | Zero language change | Models may still default to `++` habit |

**Recommendation**: Option A (polymorphic `++`) is ideal but may conflict with the string-typed `++` implementation. If not feasible, Option C (prompt warning) is the minimum viable fix with a new `concat` or `append` function prominently documented.

**Files (Option A)**: `internal/types/typechecker.go`, `internal/eval/eval_evaluator.go`, `internal/elaborate/elaborate.go`
**Files (Option C)**: `prompts/v0.9.0.md` (or next version)

### Gap 3: `Result[T]` vs `Result[T, E]` Arity (4 failures)

**Symptom**: `type unification failed: arity mismatch: Result expects 2 args, got 1`

**Root cause**: Models write `Result[Config]` assuming Result has a default error type (like Rust's `anyhow::Result<T>`). AILANG's Result ADT requires both type parameters: `Result[OkType, ErrType]`.

**Evidence**: 4 failures on `config_file_parser` across claude-opus, claude-sonnet, gemini-3-pro, and gpt5-4. All write `Result[Config]` instead of `Result[Config, string]`.

**Options**:

| Option | Pros | Cons |
|--------|------|------|
| A. Allow `Result[T]` as sugar for `Result[T, string]` | Matches model expectations | Implicit default is un-AILANG-like (explicit > implicit) |
| B. Prompt fix: add `Result[T, E]` example | No language change, easy | Models may still omit the error type |

**Recommendation**: Option B. Add to teaching prompt:
```
-- Result ALWAYS takes 2 type parameters:
-- Result[OkType, ErrType]
-- Example: Result[int, string], Result[Config, string]
-- ❌ WRONG: Result[Config] (missing error type)
```

**Files**: `prompts/v0.9.0.md` (or next version)

### Gap 4: Python Standard Eval System Prompt (24 failures)

**Symptom**: Models respond conversationally ("I'm ready to help you write Python code") or generate empty stubs (`def main(): pass`).

**Root cause**: The system prompt was a single generic sentence: `"You are a programming assistant. Generate ONLY code..."`. Too vague for models to understand they should write a complete, specific program.

**Fix**: Already applied — updated system prompt in `ai_provider.go` to be explicit about generating complete, runnable programs with no conversation or stubs.

**Status**: ✅ Fixed (pending commit)

### Gap 5: Gemini Agent Timeout (25 API errors)

**Symptom**: `timeout after 1m0s (hard ceiling)` or `exit status 1` from Gemini CLI.

**Root cause**: 1-minute hard timeout is too short for complex AILANG benchmarks where Gemini needs multiple tool-use cycles. Gemini also occasionally crashes with non-zero exit without useful error output.

**Recommendation**: Increase Gemini agent timeout from 60s to 120s for eval harness runs. Consider adding retry logic for `exit status 1` failures.

**Files**: `internal/eval_harness/agent_runner_streaming.go` or eval harness config

## Implementation Plan

### Phase 1: Low-effort fixes (est. 2-4 hours)

1. **Improve markdown fence stripping** — handle ` ```ailang ` and multi-layer fences
2. **Update teaching prompt** — add `Result[T, E]` arity reminder, list append guidance
3. **Increase Gemini timeout** — 60s → 120s in eval harness

### Phase 2: Language improvement (est. 1 day, needs design decision)

4. **List `++` operator** — decide on Option A vs C, implement if Option A chosen

### Validation

Re-run `ailang eval-suite --full` after Phase 1 to measure improvement. Target:
- Standard eval: 80% → 87%+ (fixing ~40 of the 111 failures)
- Agent eval: re-run with increased timeout to get clean Gemini data

## Appendix: Full Error Distribution (v0.9.0 Standard Eval)

### AILANG Compile Errors by Benchmark

| Benchmark | Failures | Error Pattern |
|-----------|----------|---------------|
| type_unify | 6/6 | `++` on lists (4), PAR_NO_PREFIX_PARSE (2) |
| config_file_parser | 4/6 | Result arity (3), PAR_NO_PREFIX_PARSE (1) |
| log_file_analyzer | 4/6 | PAR_NO_PREFIX_PARSE (2), PAR_UNEXPECTED_TOKEN (2) |
| csv_to_json_converter | 3/6 | PAR_NO_PREFIX_PARSE (2), PAR_UNEXPECTED_TOKEN (1) |
| graph_bfs | 3/6 | PAR_NO_PREFIX_PARSE (1), PAR_UNEXPECTED_TOKEN (1), type error (1) |
| red_black_tree | 3/6 | PAR_NO_PREFIX_PARSE (2), PAR_UNEXPECTED_TOKEN (1) |

### Agent Eval Turn Distribution (AILANG successes)

| Turns | Count | Example Benchmarks |
|-------|-------|--------------------|
| 3 | 21 | Simple benchmarks (adt_option, fold_reduce, fizzbuzz) |
| 4 | 33 | Most benchmarks (typical write-check-fix-pass cycle) |
| 5-6 | 4 | type_unify, higher_order_functions, float_eq |
| 7-8 | 3 | binary_tree_sum, balanced_parens, canonical_normalization |
| 10+ | 2 | csv_to_json_converter (10), effect_composition (18 w/ haiku) |

Median turns for AILANG success: **4** (one write + one check cycle).
High-turn outliers indicate DX friction on effect annotations and complex ADT manipulation.
