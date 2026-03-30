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
- AILANG has 51 internal benchmarks and 611+ eval results across 6 models — but all measure AILANG in isolation
- No **independent, third-party** cross-language comparison exists
- Our internal benchmarks show AILANG outperforms Python on our own eval suite — but this is self-reported data
- Published research shows niche languages suffer 30–50% accuracy drops vs mainstream languages
- AILANG has zero training data in any model, placing it in the "niche language" category by data volume
- Without independent measurement, AILANG's core thesis (machine-first design > training data volume) is unproven

**Impact:**
- AILANG adoption depends on **independently verifiable** evidence that AI agents can write correct AILANG code
- Academic credibility requires comparison using established, third-party benchmark suites — not self-reported results
- Users choosing between AILANG and Python need apples-to-apples data from the same benchmark infrastructure
- If AILANG performs poorly on independent benchmarks, we need to know *which* constructs cause failures so we can improve prompts/stdlib
- Several high-quality benchmark suites already exist and accept new languages — we should join them, not build our own

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

### Overview: Add AILANG to Existing Third-Party Benchmarks

The approach is to **integrate AILANG into established, independent benchmark suites** — not to create our own problems. This gives credible, third-party-comparable results using their infrastructure, their problems, and their scoring methodology.

Three benchmark tracks, ordered by feasibility and value:

1. **Track A: ai-coding-lang-bench** (fork, add AILANG as 16th language) — agentic mini-git task
2. **Track B: leetgptsolver** (fork, add AILANG as 5th language) — 100 LeetCode problems
3. **Track C: AutoCodeBench** (add AILANG to Docker sandbox) — 3,920 problems, 20→21 languages

### Track A: ai-coding-lang-bench (Primary)

**Repo**: `github.com/mame/ai-coding-lang-bench`

Fork the repo and add AILANG as a 16th language configuration. The benchmark's infrastructure handles everything: it gives Claude Code a spec + language context, lets the agent write code, runs the test suite, and records time/cost/pass-rate.

**Integration details** (confirmed via repo analysis):

Adding AILANG requires just one entry in the `LANGUAGES` hash in `benchmark.rb`:
```ruby
'ailang' => { exts: %w[ail], version_cmd: 'ailang --version', extra_prompt: '<ailang prompt output>' }
```

The benchmark runner creates a fresh directory, copies spec + test files, and invokes Claude Code (`claude -p`) with a prompt like "Implement minigit using {Language}". Tests are **language-agnostic** — they just call `./minigit` and check output via bash scripts.

**What we contribute:**
1. One entry in `LANGUAGES` hash with `extra_prompt` containing `ailang prompt` output
2. AILANG must be installed on the benchmark machine (add to `extra_path` if needed)
3. Solution must produce a `./minigit` executable (shebang script or compiled binary)
4. Optionally: submit upstream PR to add AILANG to the official repo

**What their infra provides:**
- Mini-git spec (MiniHash = custom FNV-1a variant, v1: init/add/commit/log, v2: +status/diff/checkout/reset)
- Language-agnostic test suite (11 tests v1, 30 tests v2)
- Trial runner (20 trials per language, records time/cost/LOC/pass-fail)
- Metrics collection (wall-clock time, API cost, tokens, LOC)

**AILANG capability gaps identified:**
- **Bitwise XOR**: MiniHash requires 64-bit unsigned XOR and multiply. AILANG has NO bitwise operators (`^` is not supported). This is a **blocking gap** — needs a builtin or stdlib addition.
- **64-bit unsigned integers**: AILANG uses signed `int`. Need unsigned 64-bit arithmetic with mod 2^64 overflow behavior.
- **Executable output**: `./minigit` must be executable. Options: (a) shebang `#!/usr/bin/env ailang run --caps IO,FS --entry main` or (b) compile to Go binary via `ailang compile --emit-go`.
- **SHA-256 available**: `std/crypto` has `sha256hex` but MiniHash spec uses custom FNV-1a, not SHA-256.

**Result**: Directly comparable to their published 15-language results (Ruby, Python, JS, Go, Rust, Haskell, etc.)

### Track B: leetgptsolver (Secondary)

**Repo**: `github.com/whisk/leetgptsolver`

**Important finding**: leetgptsolver submits code to LeetCode's online judge, which does NOT support AILANG. We **cannot use their execution infrastructure directly**.

**Revised approach**: Use their **problem dataset** with our own eval infrastructure:
1. Download their 100 LeetCode problems from HuggingFace (`whiskwhite/leetcode-complete`)
2. Use their problem descriptions as prompts (same problems, same difficulty)
3. Ask models to generate AILANG solutions (with prompt strategy variants)
4. Evaluate locally using `ailang run` with expected-output matching (our existing eval harness)
5. Compare our AILANG pass rates against their published Python/Java/Rust/Elixir results

**What we reuse from their work:**
- Problem selection (100 problems, Oct 2025–Feb 2026, likely unseen by models)
- Problem descriptions and test cases
- Published baseline results for Python/Java/Rust/Elixir across 5 models

**What we build:**
1. Problem adapter: convert LeetCode problem JSON → AILANG eval harness format (our existing `benchmarks/*.yml`)
2. Test case extraction: map LeetCode expected outputs to `expected_stdout`
3. AILANG-specific system prompt with `ailang prompt` output

**Key challenge**: LeetCode problems assume mutable data structures and imperative patterns. Some problems may be inexpressible in pure functional AILANG:
- Filter to problems expressible with immutable lists/trees/recursion
- Report which problems were excluded and why (this itself is useful gap data)
- The exclusion list is itself a valuable finding — it quantifies AILANG's expressiveness gap

**AILANG context strategies** (run all to measure information value):

| Strategy | Description | Measures |
|----------|-------------|----------|
| Zero-shot | Just "write in AILANG" | Baseline — how much models already "know" |
| Spec-prompted | Provide `ailang prompt` as system context | Value of AILANG's machine-readable spec |
| Few-shot | Provide 3–5 solved examples from existing eval suite | Value of examples over documentation |
| Spec + few-shot | Both spec and examples | Maximum context, upper bound on performance |

**Result**: Directly comparable to their published Python/Java/Rust/Elixir results across 5 models.

### Track C: AutoCodeBench (Future)

**Repo**: `github.com/Tencent-Hunyuan/AutoCodeBenchmark`
**Docker**: `hunyuansandbox/multi-language-sandbox`

Add AILANG to the Tencent Docker sandbox as a 21st language.

**Important finding**: The sandbox is a **pre-built opaque Docker image** with no extension mechanism — no plugin system, no volume mounts for custom runtimes, no source code in the repo.

**Options to add AILANG:**
1. **Custom Docker image**: `FROM hunyuansandbox/multi-language-sandbox:v2`, install AILANG runtime, configure internal dispatcher
2. **Shell wrapper trick**: Use `"lang": "shell"` and have shell scripts invoke `ailang` — avoids image rebuild
3. **Build own sandbox**: Implement the same HTTP API (`POST /submit`) backed by `ailang run` — most control

The evaluation pipeline sends `func_code` (solution) and `main_code` (test harness) to the sandbox, which combines and runs them. Results include `exec_outcome: "PASSED"` or failure info. Translation templates in `AutoCodeGen/templates/translate_templates/` convert problems from source languages to target languages.

**Deferred** to after Tracks A and B produce initial results. The shell wrapper approach (option 2) is the fastest path.

### Architecture

```
benchmarks/
  cross-language/
    README.md                    # Overview and reproduction instructions
    forks/                       # Git submodules or fork references
      ai-coding-lang-bench/      # Track A fork (submodule)
      leetgptsolver/             # Track B fork (submodule)
    ailang-configs/              # AILANG-specific additions to each benchmark
      lang-bench/                # Language config, context file for Track A
      leetgptsolver/             # Language adapter, prompt config for Track B
    results/                     # Raw results from benchmark runs
      track-a/                   # Mini-git trial results
      track-b/                   # LeetCode eval results
    analysis/                    # Cross-track analysis
      comparison.md              # AILANG vs other languages summary
      failure_modes.md           # Top failure patterns
      gap_design_docs/           # Design docs for identified gaps
```

### Implementation Plan

**Phase 1: Fork & Integrate (Track A — ai-coding-lang-bench)** (~2 days)
- [ ] Fork `mame/ai-coding-lang-bench`
- [ ] Study existing language configs (Ruby, Python, etc.) to understand the pattern
- [ ] Add AILANG language config following the same pattern
- [ ] Create AILANG context file with `ailang prompt` output
- [ ] Verify AILANG test runner works with the mini-git test suite
- [ ] Audit missing builtins (SHA-1 hash, any others)

**Phase 2: Run Track A Trials** (~1 day)
- [ ] Run 20 trials with Claude Code (matching their methodology)
- [ ] Collect time, cost, LOC, pass/fail per trial
- [ ] Compare against published results for 15 existing languages

**Phase 3: Fork & Integrate (Track B — leetgptsolver)** (~2 days)
- [ ] Fork `whisk/leetgptsolver`
- [ ] Study language adapter pattern
- [ ] Add AILANG language support (compile/run commands, prompt template)
- [ ] Filter LeetCode problems to those expressible in AILANG (document exclusions)
- [ ] Run across available models with all 4 prompt strategies

**Phase 4: Analysis and Gap Design Docs** (~2 days)
- [ ] Cross-track comparison: where does AILANG land relative to other languages?
- [ ] Failure mode taxonomy: syntax errors, type errors, missing builtins, wrong patterns
- [ ] Create design doc for each major gap category (minimum 2, expected 3–5)
- [ ] Each gap design doc includes concrete examples from benchmark failures
- [ ] Top-3 recommendations for improving AILANG AI-friendliness
- [ ] CHANGELOG.md updated with results summary

### Files to Modify/Create

**New files:**
- `benchmarks/cross-language/README.md` — Reproduction instructions (~50 LOC)
- `benchmarks/cross-language/ailang-configs/lang-bench/` — AILANG config for Track A (~100 LOC)
- `benchmarks/cross-language/ailang-configs/leetgptsolver/` — AILANG adapter for Track B (~150 LOC)
- `benchmarks/cross-language/analysis/` — Results analysis and gap design docs

**Modified files:**
- `CHANGELOG.md` — Document benchmark results
- `docs/` — Add cross-language benchmark guide

**External (in forks):**
- Track A fork: Add AILANG language directory following upstream pattern
- Track B fork: Add AILANG language adapter following upstream pattern

---

## Examples

### Example 1: Track A — Adding AILANG to ai-coding-lang-bench

The upstream repo has per-language directories. We add an `ailang/` directory following the same pattern:

```
ai-coding-lang-bench/
  ruby/
    config.json        # Language-specific settings
    LANGUAGE.md        # Language context for Claude Code
  python/
    config.json
    LANGUAGE.md
  ailang/              # ← We add this
    config.json        # AILANG compile/run commands
    LANGUAGE.md        # Contains `ailang prompt` output + AILANG-specific hints
```

The AILANG context file (`LANGUAGE.md`) would include:
```markdown
# AILANG Language Reference
[output of `ailang prompt`]

## Running AILANG
ailang run --caps IO,FS --entry main solution.ail

## Key Differences from Python
- No loops — use recursion
- All side effects declared in function signatures
- Pattern matching instead of if/elif chains
```

Claude Code then implements mini-git in AILANG, just as it does for Ruby/Python/Go.

### Example 2: Track B — Adding AILANG to leetgptsolver

The upstream tool has language adapters. We add AILANG:

```python
# In leetgptsolver's language config
"ailang": {
    "extension": ".ail",
    "compile_cmd": "ailang check {file}",
    "run_cmd": "ailang run --caps IO --entry main {file}",
    "system_prompt": "You are writing AILANG code. {ailang_prompt_content}",
}
```

The tool then sends LeetCode problems to models, asking for AILANG solutions, and runs them against LeetCode test cases.

### Example 3: Expected Results Comparison Table

**Track A (Mini-Git) — AILANG vs published results:**

| Language | Time | Cost | Pass Rate | vs Python |
|----------|------|------|-----------|-----------|
| Ruby | 73.1s | $0.36 | 40/40 | baseline |
| Python | 74.6s | $0.38 | 40/40 | +2% cost |
| JavaScript | 81.1s | $0.39 | 40/40 | +8% cost |
| Go | 101.6s | $0.50 | 40/40 | +32% cost |
| Rust | 113.7s | $0.54 | 38/40 | +42% cost |
| **AILANG** | **???** | **???** | **???** | **???** |
| Haskell | 174.0s | $0.74 | 39/40 | +95% cost |

**Track B (LeetCode) — AILANG vs published results:**

| Model | Python | Java | Rust | Elixir | **AILANG** |
|-------|--------|------|------|--------|------------|
| Gemini 3 Flash | 84% | 93% | 78% | 83% | **???** |
| GPT-5 Mini | 93% | 94% | 80% | 63% | **???** |
| Grok | 73% | 65% | 65% | 30% | **???** |

---

## Success Criteria

- [ ] AILANG added to `ai-coding-lang-bench` fork, 20 mini-git trials completed
- [ ] AILANG added to `leetgptsolver` fork, LeetCode problems run across 3+ models
- [ ] Results directly comparable to published data for 5+ other languages (same infra, same scoring)
- [ ] Failure mode analysis identifies top-3 reasons models fail on AILANG
- [ ] Gap design doc created for each major failure category (minimum 2, expected 3–5)
- [ ] All work is in forks with READMEs — anyone can reproduce our results
- [ ] CHANGELOG.md updated with results summary
- [ ] Upstream PRs submitted to add AILANG to both benchmark repos (stretch goal)

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

- **Creating our own benchmark problems** — we use existing third-party suites for independent credibility
- **Re-running other languages ourselves** — we compare AILANG results against already-published data
- **Modifying AILANG syntax based on results** — that's a separate design doc triggered by findings
- **Fine-tuning models on AILANG** — we measure existing model capabilities
- **Competing with HumanEval/MBPP** — those benchmarks are already saturated; we focus on newer methodologies

---

## Timeline

**Week 1** (~15 hours):
- Fork repos, study existing language config patterns
- Add AILANG to ai-coding-lang-bench (Track A)
- Run 20 mini-git trials, collect initial results

**Week 2** (~15 hours):
- Add AILANG to leetgptsolver (Track B)
- Run LeetCode problems across models with prompt strategies
- Collect cross-language comparison data

**Week 3** (~10 hours):
- Analysis: failure taxonomy, gap identification
- Create design docs for each gap category
- Write up results, submit upstream PRs

**Total: ~40 hours across 3 weeks**

---

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Models produce zero correct AILANG (no training data) | High | Use spec-prompted and few-shot strategies; measure improvement over zero-shot |
| **Missing bitwise ops for Track A MiniHash** | **High** | **Add XOR/bitwise builtins before Track A, or substitute sha256hex** |
| **Track B can't use LeetCode judge** | **Med** | **Use their problems with our eval harness — same problems, local execution** |
| Track C Docker sandbox is opaque | Med | Use shell wrapper approach or defer to later phase |
| API costs for 600+ model calls | Med | Start with 5 trials, scale to 20 only for promising configurations |
| Results are embarrassingly bad | Med | This is still valuable data — identifies exactly what to fix. Frame as "gap analysis" |
| Upstream benchmark repos change or disappear | Low | Fork/vendor the specific commits we use |
| AILANG compilation errors dominate failures (not generation quality) | Med | Separate "compiles but wrong" from "doesn't compile" in failure taxonomy |
| LeetCode problems require mutable state / imperative patterns | Med | Document exclusions as expressiveness gap data; filter to functional-friendly subset |

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
**Last updated**: 2026-03-30 (revised: pivot from "create own problems" to "add AILANG to existing third-party benchmarks")
