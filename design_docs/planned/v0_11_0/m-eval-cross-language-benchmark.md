# M-EVAL-XLANG: Cross-Language AI Code Generation Benchmark

**Status**: Planned
**Target**: v0.10.0
**Priority**: P1 (High — strategic positioning for AILANG)
**Estimated**: 2–3 weeks (phased)
**Dependencies**: Existing eval harness, `ailang eval-run` infrastructure

## Executive Summary

Recent research shows AI code generation performance varies significantly by programming language — up to 2x cost difference and 50+ percentage-point accuracy gaps between popular and niche languages. AILANG, as a language *designed for* AI synthesis, should outperform expectations for its training-data size. This benchmark suite will measure exactly where AILANG stands against mainstream and niche languages using established, reproducible methodologies.

**Core hypothesis**: AILANG's deterministic semantics, explicit effects, and machine-first design should yield AI code generation accuracy *disproportionate* to its training data volume — closer to Python/Ruby than to similarly-sized niche languages like Elixir or Racket.

---

## Research Foundation

### Source 1: AutoCodeBench (Tencent Hunyuan, 2025)

**Paper**: "AutoCodeBench: Large Language Models are Automatic Code Benchmark Generators"
**Website**: https://autocodebench.github.io/
**GitHub**: https://github.com/Tencent-Hunyuan/AutoCodeBenchmark
**HuggingFace**: https://huggingface.co/datasets/tencent/AutoCodeBenchmark
**Docker**: https://hub.docker.com/r/hunyuansandbox/multi-language-sandbox

- **Scale**: 3,920 problems across **20 languages** (Python, C++, Java, JS, Go, Shell, C#, Dart, Elixir, Julia, Kotlin, Perl, PHP, Racket, R, Ruby, Rust, Scala, Swift, TypeScript)
- **Metric**: pass@1 (code passes all test cases)
- **Key finding**: "Performance difference between models is small for popular languages, but large for low-resource languages"
- **Methodology**: Automated LLM-sandbox workflow generates problems in reverse (inputs → execute → outputs → problem statement)
- **Variants**: ACB-Full (3,920), ACB-Lite (1,586 solvable by ≥2 models), ACB-Complete (1,000 completion-style)

**Relevance to AILANG**: The Docker sandbox supports 30+ languages. Adding AILANG as a 21st benchmark language is feasible — requires adding the AILANG runtime to the sandbox container.

### Source 2: ai-coding-lang-bench (mame, 2025)

**Article**: "Which Programming Language Is Best for Claude Code?"
**GitHub**: https://github.com/mame/ai-coding-lang-bench

- **Task**: Build a simplified Git (init, add, commit, log, status, diff, checkout, reset) in each language
- **Agent**: Claude Code (Opus 4.6, high effort mode)
- **Languages**: 15 configs — Ruby, Python, JS, Go, Java, Rust, Perl, Python/mypy, OCaml, Lua, Scheme, TypeScript, C, Haskell, Ruby/Steep
- **Trials**: 20 per language (600 total runs)
- **Metrics**: Time, API cost, LOC, test pass rate

**Key results**:

| Language | Time | Cost | Pass Rate |
|----------|------|------|-----------|
| Ruby | 73.1s ± 4.2s | $0.36 | 40/40 |
| Python | 74.6s ± 4.5s | $0.38 | 40/40 |
| JavaScript | 81.1s ± 5.0s | $0.39 | 40/40 |
| Go | 101.6s ± 37.0s | $0.50 | 40/40 |
| Rust | 113.7s ± 54.8s | $0.54 | 38/40 |
| Haskell | 174.0s ± 44.2s | $0.74 | 39/40 |

**Key finding**: Dynamic languages are ~2x faster and cheaper than static languages. Type-checking overhead adds 60–220%.

**Relevance to AILANG**: This is the most directly reproducible benchmark. The mini-git task is well-scoped and the repo includes the full test suite. AILANG can be added as a 16th language.

### Source 3: Scaling Laws for Code (Yang et al., 2025)

**Paper**: "Scaling Laws for Code: Every Programming Language Matters"
**arXiv**: https://arxiv.org/abs/2512.13472

- **Scale**: 1,000+ experiments, model sizes 0.2B–14B, 1T tokens
- **Key finding**: Interpreted languages (Python) benefit more from scale than compiled languages (Rust)
- **Finding**: Multilingual pre-training creates synergistic cross-lingual transfer
- **Finding**: Optimal allocation should prioritize high-utility languages — but niche languages still contribute to overall performance

**Relevance to AILANG**: AILANG has essentially zero training data in any model. This paper's scaling laws predict AILANG should perform poorly — but AILANG's machine-first design may defy these predictions. Measuring this gap is the core experiment.

### Source 4: LeetCode Cross-Language Comparison (whisk, 2026)

**Article**: "Comparing LLMs' Coding Abilities Across Programming Languages"
**HackerNoon**: https://hackernoon.com/comparing-llms-coding-abilities-across-programming-languages
**Dataset**: https://huggingface.co/datasets/whiskwhite/leetcode-complete
**Tool**: https://github.com/whisk/leetgptsolver

- **Problems**: 100 LeetCode problems (Oct 2025–Feb 2026), likely unseen by models
- **Languages**: Python, Java, Rust, Elixir (+ MySQL, Oracle SQL)
- **Models**: Claude Sonnet 4.5, Gemini 2.5 Flash, Gemini 3 Flash, GPT-5 Mini, Grok

**Key results** (acceptance rate):

| Model | Python | Java | Rust | Elixir |
|-------|--------|------|------|--------|
| Gemini 3 Flash | 84% | 93% | 78% | 83% |
| GPT-5 Mini | 93% | 94% | 80% | 63% |
| Grok | 73% | 65% | 65% | 30% |

**Key finding**: Language choice matters as much as model choice. Elixir drops 30–43 percentage points vs Python on most models. But Gemini 3 Flash shows near-uniform performance — meaning model architecture matters too.

**Relevance to AILANG**: The LeetCode methodology with controlled problem sets is directly adaptable. AILANG can be positioned against Elixir (similar training data scarcity) to test whether language design compensates for data scarcity.

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Benchmark infrastructure, no language semantics changes |
| A2: Replayability | +1 | Benchmark results are reproducible artifacts |
| A3: Effect Legibility | 0 | No effect system changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | No type system changes |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +2 | Directly measures AILANG's AI-friendliness proposition |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | +1 | Measures literal $ cost of AI code generation per language |
| A10: Composability | 0 | No semantic changes |
| A11: Structured Failure | 0 | No error handling changes |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Directly validates machine-first design

---

## Problem Statement

**Current State:**
- AILANG has 47 internal benchmarks, all single-file, all measuring AILANG in isolation
- No cross-language comparison exists — we cannot answer "how does AILANG compare to Python for AI code generation?"
- Published research shows niche languages suffer 30–50% accuracy drops vs mainstream languages
- AILANG has zero training data in any model, placing it in the "niche language" category by data volume
- Without measurement, AILANG's core thesis (machine-first design > training data volume) is unproven

**Impact:**
- AILANG adoption depends on demonstrating that AI agents can write correct AILANG code
- Academic credibility requires published cross-language comparisons
- Users choosing between AILANG and Python need data, not assertions
- If AILANG performs poorly, we need to know *which* constructs cause failures so we can improve prompts/stdlib

---

## Goals

**Primary Goal**: Measure AILANG's AI code generation performance against 5+ mainstream languages using reproducible, published benchmark methodologies.

**Success Metrics:**
- Run at least 2 of the 4 benchmark suites with AILANG included
- Produce pass@1 / acceptance-rate numbers for AILANG alongside Python, Ruby, JS, Go, Rust
- Measure cost (tokens/dollars) for AILANG vs other languages
- Identify top-3 failure modes when models generate AILANG code
- Publish results (blog post / design doc update with data)

**Non-Goals (this phase):**
- Modifying AILANG's syntax or semantics based on results (separate design doc)
- Adding AILANG to AutoCodeBench's Docker sandbox upstream (future PR)
- Training or fine-tuning models on AILANG (out of scope entirely)

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Which benchmarks to run first | Determines what we can compare against | human | design | low |
| Which models to test | Results vary dramatically by model | human | design | low |
| How to provide AILANG context to models | Zero-shot vs prompt-with-spec vs few-shot | human | design | high |
| Whether to use ailang prompt or custom system prompt | Affects reproducibility and fairness | agent | compile | med |
| Problem selection for LeetCode-style bench | Must match AILANG's current capabilities | agent | compile | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] **Benchmark selection**: Which 2 benchmarks to run first (recommended: ai-coding-lang-bench + LeetCode-style)
- [ ] **Model selection**: Which models to test (recommended: Claude Opus 4.6, Gemini 2.5 Flash, GPT-5 Mini)
- [ ] **AILANG context strategy**: How models learn AILANG syntax — zero-shot, `ailang prompt` output, or few-shot with examples

---

## Solution Design

### Overview

Three benchmark tracks, ordered by feasibility and value:

1. **Track A: Mini-Git Benchmark** (adapt `ai-coding-lang-bench`) — measures agentic coding with tool use
2. **Track B: Algorithmic Benchmark** (adapt `leetgptsolver`) — measures pure code generation accuracy
3. **Track C: AutoCodeBench Integration** (future) — adds AILANG to the 20-language standardized suite

### Track A: Mini-Git Benchmark (Primary)

Fork `github.com/mame/ai-coding-lang-bench` and add AILANG as a language.

**What we need:**
1. AILANG implementation of the mini-git spec (reference implementation)
2. Test suite adapted for AILANG (the original uses language-specific test runners)
3. Benchmark runner configuration for AILANG
4. AILANG language spec file for the agent prompt

**Approach:**
- The benchmark gives Claude Code a spec file and says "implement this, make the tests pass"
- For AILANG, we provide `ailang prompt` output as additional context in the spec
- Run 20 trials, measure time, cost, LOC, pass rate
- Compare directly against the 15 existing language results

**Key challenge**: The mini-git task requires filesystem I/O, hashing, and string manipulation. AILANG needs:
- `fs` effect for file operations (available via stdlib)
- String builtins: `split`, `join`, `replace`, `trim`, `starts_with` (mostly available)
- Hash function (may need a simple builtin or use existing `crypto` stdlib if available)

### Track B: Algorithmic Benchmark

Adapt the LeetCode methodology for AILANG-compatible problems.

**What we need:**
1. Select 30–50 problems from the whisk dataset that are expressible in AILANG
2. Write AILANG test harnesses for each problem
3. Use the `leetgptsolver` tool pattern: prompt model → extract code → run tests → pass/fail
4. Test across 3+ models

**Problem selection criteria:**
- Must be expressible with AILANG's current type system and builtins
- Avoid problems requiring: mutable data structures (graphs with cycles), complex I/O, external libraries
- Good candidates: recursion, list processing, string manipulation, math, tree traversal, dynamic programming

**AILANG context strategy options:**

| Strategy | Description | Pros | Cons |
|----------|-------------|------|------|
| Zero-shot | No AILANG context, just "write in AILANG" | Fair baseline | Will fail — models don't know AILANG |
| Spec-prompted | Provide `ailang prompt` output as system prompt | Realistic for users | Large prompt, higher cost |
| Few-shot | Provide 3–5 solved examples | Best accuracy likely | Biases toward example patterns |
| Spec + few-shot | Both spec and examples | Most context | Highest token cost |

**Recommended**: Run all 4 strategies to measure the *information value* of each. This directly measures how much AILANG's machine-readable spec helps models.

### Track C: AutoCodeBench Integration (Future)

**Scope**: Add AILANG to the Tencent Docker sandbox as a 21st language.

**Requirements:**
- Install AILANG runtime in `hunyuansandbox/multi-language-sandbox` container
- Map AILANG to sandbox language identifier in `call_sandbox.py`
- Generate AILANG variants of benchmark problems (using AutoCodeGen pipeline)
- Submit results to leaderboard

**This is a significant effort** (custom Docker image, problem translation) and is deferred to after Tracks A and B produce initial results.

### Architecture

```
benchmarks/
  cross-language/
    README.md                    # Overview and reproduction instructions
    mini-git/                    # Track A
      spec/                      # Git spec files (from upstream)
      ailang/                    # AILANG-specific test suite and config
      results/                   # Raw results per language
      report.md                  # Generated comparison report
    algorithmic/                 # Track B
      problems/                  # Problem definitions with test cases
        001_two_sum.ail
        002_fibonacci.ail
        ...
      harness/                   # Test runner and result collector
      prompts/                   # System prompts for each strategy
      results/                   # Raw results per model × strategy
      report.md                  # Generated comparison report
    analysis/                    # Cross-track analysis
      comparison.md              # AILANG vs other languages summary
      failure_modes.md           # Top failure patterns
```

**Components:**
1. **Benchmark Runner** (`benchmarks/cross-language/run.sh`): Orchestrates trials, collects metrics
2. **Result Collector** (`benchmarks/cross-language/collect.py`): Aggregates raw results into comparison tables
3. **Failure Analyzer** (`benchmarks/cross-language/analyze_failures.py`): Categorizes why models fail on AILANG

### Implementation Plan

**Phase 1: Mini-Git Benchmark (Track A)** (~3 days)
- [ ] Fork `mame/ai-coding-lang-bench` or extract spec/test framework
- [ ] Write AILANG reference implementation of mini-git
- [ ] Adapt test suite for AILANG
- [ ] Create AILANG spec file with `ailang prompt` context
- [ ] Run 20 trials with Claude Code (Opus 4.6)
- [ ] Compare against published results for other 15 languages

**Phase 2: Algorithmic Benchmark (Track B)** (~5 days)
- [ ] Select 30–50 LeetCode-style problems compatible with AILANG
- [ ] Write AILANG test harnesses for each problem
- [ ] Implement benchmark runner (prompt model → extract code → evaluate)
- [ ] Run across 3 models × 4 prompt strategies × 30 problems
- [ ] Analyze results: pass rates, failure modes, token costs

**Phase 3: Analysis and Reporting** (~2 days)
- [ ] Cross-track comparison: where does AILANG land relative to other languages?
- [ ] Failure mode taxonomy: syntax errors, type errors, missing builtins, wrong algorithms
- [ ] Recommendations: what AILANG improvements would most help AI code generation?
- [ ] Write up results for blog/documentation

### Files to Modify/Create

**New files:**
- `benchmarks/cross-language/README.md` — Reproduction instructions (~50 LOC)
- `benchmarks/cross-language/mini-git/ailang/` — Reference impl + tests (~300 LOC)
- `benchmarks/cross-language/algorithmic/problems/*.ail` — 30–50 problem files (~1500 LOC)
- `benchmarks/cross-language/algorithmic/harness/run.sh` — Benchmark orchestrator (~100 LOC)
- `benchmarks/cross-language/algorithmic/prompts/` — System prompt variants (~200 LOC)

**Modified files:**
- `CHANGELOG.md` — Document benchmark results
- `docs/` — Add cross-language benchmark guide

---

## Examples

### Example 1: Mini-Git AILANG Trial

The agent receives this prompt:
```
Read the SPEC file and implement a simplified Git in AILANG.
Make all tests pass. You can also read the AILANG language reference
below for syntax guidance.

[contents of ailang prompt output]
```

Expected AILANG output (reference implementation sketch):
```ailang
module minigit

import fs from "std/fs"
import crypto from "std/crypto"

@effect(fs)
export func init(path: String) -> Result(String, String) =
  let git_dir = path ++ "/.git"
  let _ = fs.mkdir(git_dir)
  let _ = fs.mkdir(git_dir ++ "/objects")
  let _ = fs.mkdir(git_dir ++ "/refs")
  let _ = fs.write(git_dir ++ "/HEAD", "ref: refs/heads/main\n")
  Ok("Initialized empty repository")
```

### Example 2: Algorithmic Problem (Two Sum)

**Problem file** (`benchmarks/cross-language/algorithmic/problems/001_two_sum.ail`):
```ailang
module two_sum_test

// Problem: Given a list of integers and a target, return indices of
// two numbers that add up to the target.
// Constraint: Exactly one solution exists.

// MODEL GENERATES THIS FUNCTION:
// func twoSum(nums: [Int], target: Int) -> (Int, Int)

// Test harness:
export func main() -> String =
  let r1 = twoSum([2, 7, 11, 15], 9)
  let r2 = twoSum([3, 2, 4], 6)
  let r3 = twoSum([3, 3], 6)
  assert(r1 == (0, 1), "test1")
  assert(r2 == (1, 2), "test2")
  assert(r3 == (0, 1), "test3")
  "ALL TESTS PASSED"
```

### Example 3: Results Comparison Table (Expected Output)

| Language | Pass@1 (30 problems) | Avg Cost | Avg Time | vs Python Delta |
|----------|---------------------|----------|----------|-----------------|
| Python | 87% | $0.02 | 3.2s | baseline |
| Ruby | 83% | $0.02 | 3.0s | -4pp |
| JavaScript | 80% | $0.02 | 3.5s | -7pp |
| Go | 77% | $0.03 | 4.1s | -10pp |
| Rust | 70% | $0.04 | 5.8s | -17pp |
| **AILANG (spec-prompted)** | **???** | **???** | **???** | **???** |
| **AILANG (zero-shot)** | **???** | **???** | **???** | **???** |
| Elixir | 52% | $0.03 | 4.5s | -35pp |

---

## Success Criteria

- [ ] Mini-git benchmark runs 20 trials with AILANG, produces time/cost/pass-rate data
- [ ] Algorithmic benchmark runs 30+ problems across 3+ models with AILANG
- [ ] Results compared against at least 5 other languages from published data
- [ ] Failure mode analysis identifies top-3 reasons models fail on AILANG
- [ ] All benchmark code is reproducible (README with exact reproduction steps)
- [ ] CHANGELOG.md updated with results summary
- [ ] Results inform concrete recommendations for AILANG prompt/stdlib improvements

---

## Testing Strategy

**Benchmark validation:**
- AILANG reference implementations must pass all tests manually before benchmarking
- Verify test harnesses catch incorrect solutions (test with deliberately wrong code)
- Cross-validate against published results for existing languages (sanity check)

**Statistical rigor:**
- Minimum 20 trials per configuration for mini-git (matching upstream)
- Report mean ± std dev for all metrics
- Use two-proportion z-test (p=0.05) for pass-rate comparisons (matching Source 4)
- Track and report all failures, not just aggregate pass rates

**Reproducibility:**
- Pin model versions (e.g., `claude-opus-4-6`, not just "Claude")
- Record exact prompts used
- Store raw API logs for verification
- Docker-based execution where possible

---

## Deferred Decisions

The following are intentionally left open for the implementer:

- **Exact problem selection for Track B** — agent may choose based on AILANG capability audit
- **Result visualization format** — agent may choose (tables, charts, or both)
- **Whether to fork or vendor the upstream repos** — agent may choose based on licensing
- **Additional languages to test alongside AILANG** — agent may add if toolchains are available

---

## Non-Goals

- **Modifying AILANG syntax based on results** — that's a separate design doc triggered by findings
- **Fine-tuning models on AILANG** — we measure existing model capabilities
- **Upstream contributions to AutoCodeBench** — future work after initial results
- **Comprehensive 20-language comparison** — we compare against published data, not re-running all languages
- **Competing with HumanEval/MBPP** — those benchmarks are already saturated; we focus on newer methodologies

---

## Timeline

**Week 1** (~15 hours):
- Phase 1: Mini-git benchmark adaptation and 20-trial run
- Initial results comparison

**Week 2** (~20 hours):
- Phase 2: Algorithmic benchmark problem selection and harness
- Multi-model, multi-strategy runs

**Week 3** (~10 hours):
- Phase 3: Analysis, failure taxonomy, recommendations
- Documentation and blog draft

**Total: ~45 hours across 3 weeks**

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Models produce zero correct AILANG (no training data) | High | Use spec-prompted and few-shot strategies; measure improvement over zero-shot |
| Mini-git task exceeds AILANG's current stdlib | Med | Audit required builtins before starting; add missing builtins if trivial |
| API costs for 600+ model calls | Med | Start with 5 trials, scale to 20 only for promising configurations |
| Results are embarrassingly bad | Med | This is still valuable data — identifies exactly what to fix. Frame as "gap analysis" |
| Upstream benchmark repos change or disappear | Low | Fork/vendor the specific commits we use |
| AILANG compilation errors dominate failures (not generation quality) | Med | Separate "compiles but wrong" from "doesn't compile" in failure taxonomy |

---

## Related Documents

**Implemented (may inform design):**
- [M-EVAL-AGENT-QUEUE](../../implemented/v0_7_0/M-EVAL-AGENT-QUEUE.md) — eval infrastructure
- [M-EVAL-CHAINS-SOURCE-OF-TRUTH](../../implemented/v0_8_1/m-eval-chains-source-of-truth.md) — eval data model
- [M-EVAL-LOOP](../../implemented/v0_3_10/M-EVAL-LOOP_self_improving_feedback.md) — self-improving eval feedback

**Planned (check for overlap):**
- [M-LOCOBENCH](m-locobench-long-context-benchmark.md) — long-context benchmark (complementary, not overlapping)
- [M-CLOUD-EVAL-WORKERS](m-cloud-eval-workers.md) — cloud eval infrastructure (would help scale this)

---

## References

- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- [AutoCodeBench](https://autocodebench.github.io/) — Tencent, 3920 problems, 20 languages
- [ai-coding-lang-bench](https://github.com/mame/ai-coding-lang-bench) — Mini-git benchmark, 15 languages
- [Scaling Laws for Code](https://arxiv.org/abs/2512.13472) — Yang et al., cross-lingual transfer
- [LeetCode Cross-Language](https://hackernoon.com/comparing-llms-coding-abilities-across-programming-languages) — whisk, 5 models × 4 languages
- [leetgptsolver](https://github.com/whisk/leetgptsolver) — LeetCode evaluation tool
- [AutoCodeBenchmark dataset](https://huggingface.co/datasets/tencent/AutoCodeBenchmark) — HuggingFace
- [LeetCode Complete dataset](https://huggingface.co/datasets/whiskwhite/leetcode-complete) — HuggingFace

---

## Future Work

- **Track C**: Full AutoCodeBench integration (add AILANG to Docker sandbox, submit to leaderboard)
- **Prompt optimization**: Use failure analysis to improve `ailang prompt` output for better AI generation
- **Stdlib gap filling**: Add builtins identified as missing by the benchmark
- **Longitudinal tracking**: Re-run benchmarks after each AILANG release to track improvement
- **Model comparison**: Test new models as they release to find AILANG's best-performing model
- **Aider-style leaderboard**: Publish ongoing AILANG results in a public leaderboard format
- **Cross-pollination with LoCoBench**: Combine single-file accuracy (this doc) with multi-file coherence (M-LOCOBENCH)

---

**Document created**: 2026-03-30
**Last updated**: 2026-03-30
