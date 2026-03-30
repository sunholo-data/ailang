# Sprint Plan: M-EVAL-XLANG — Cross-Language AI Code Generation Benchmark (Revised)

## Summary

Add AILANG to existing third-party benchmark suites to get independent, credible cross-language comparisons. Two tracks: (A) fork `ai-coding-lang-bench` for agentic mini-git, (B) adapt LeetCode problems from `leetgptsolver` dataset for algorithmic benchmarking using our eval harness.

**Duration:** 6 days (~40 hours)
**Dependencies:** Bitwise XOR builtins (for Track A MiniHash)
**Risk Level:** Medium (unknown AILANG pass rates, API costs)
**Design Doc:** [m-eval-cross-language-benchmark.md](m-eval-cross-language-benchmark.md)

## Current Status Analysis

### Completed
- ✅ FS builtins: mkdir, mkdirAll, isDir, isFile, removeFile (added this sprint)
- ✅ 51 existing benchmarks in eval harness
- ✅ 611+ eval results in v0.9.1.1 baseline across 6 models (Python + AILANG)
- ✅ Research on all 3 benchmark suites complete (integration requirements documented)
- ✅ Design doc updated with third-party integration approach

### Research Findings

| Benchmark | Integration Effort | Blockers |
|-----------|-------------------|----------|
| ai-coding-lang-bench (Track A) | 1 config line + AILANG install | **No bitwise XOR** (MiniHash), no shebang/compile-to-binary |
| leetgptsolver (Track B) | Can't use LeetCode judge — use their problems + our eval harness | Need problem adapter, local test runner |
| AutoCodeBench (Track C) | Custom Docker image or shell wrapper | Deferred — highest effort |

### AILANG Capability Gaps for Track A

| Capability | Status | Impact |
|------------|--------|--------|
| Bitwise XOR (`^`) | ❌ Missing | **Blocks MiniHash** — FNV-1a hash requires XOR |
| Unsigned 64-bit ints | ❌ Missing | MiniHash needs mod 2^64 overflow |
| Executable output | ❌ Missing | `./minigit` must be executable (shebang or compile) |
| Bitwise shift (`<<`, `>>`) | ❌ Missing | Useful for hash functions |
| All string/FS ops | ✅ Available | split, join, replace, readFile, writeFile, mkdir, etc. |
| SHA-256 | ✅ Available | `_crypto_sha256hex` — not used by MiniHash but available |

---

## Proposed Milestones

### Milestone 1: Add Bitwise Builtins + Fork Track A (M1_BITWISE_FORK)
**Goal:** Add bitwise operator builtins needed for MiniHash, then fork ai-coding-lang-bench and add AILANG config
**Estimated:** ~200 LOC builtins + ~100 LOC config = ~300 LOC
**Duration:** 1.5 days

**Tasks:**
- Add bitwise builtins: `xor`, `bitAnd`, `bitOr`, `bitNot`, `shiftLeft`, `shiftRight` to `internal/builtins/math.go`
- Register builtins with Type Builder DSL, write hermetic tests
- Add `std/bits` or `std/math` stdlib wrappers
- Fork `mame/ai-coding-lang-bench` to `sunholo/ai-coding-lang-bench`
- Add AILANG entry to `LANGUAGES` hash in `benchmark.rb`
- Create `extra_prompt` with `ailang prompt` output
- Test AILANG detection (version command, file extension)
- Verify `ailang run` can be invoked as a shebang or via wrapper script

**Acceptance Criteria:**
- [ ] Bitwise XOR, AND, OR, shift builtins pass unit tests
- [ ] `make test` and `make lint` clean
- [ ] Fork exists with AILANG language entry
- [ ] `extra_prompt` contains current `ailang prompt` output
- [ ] Wrapper script makes `./minigit` executable with AILANG backend

---

### Milestone 2: Run Track A — Mini-Git Trials (M2_TRACK_A_RUN)
**Goal:** Run 20 mini-git trials with AILANG via the forked benchmark, collect results
**Estimated:** ~100 LOC analysis scripts
**Duration:** 1 day

**Tasks:**
- Install AILANG on benchmark machine (or run locally with PATH configured)
- Run warmup trial to verify pipeline works
- Run 20 v1 trials (init, add, commit, log — 11 tests each)
- Collect per-trial metrics: time, cost, LOC, pass/fail per test
- If v1 results are promising, run v2 trials (status, diff, checkout, reset — 30 tests)
- Compare results against published data for 15 existing languages

**Acceptance Criteria:**
- [ ] 20 v1 trials completed with metrics
- [ ] Results directly comparable to published Ruby/Python/JS/Go/Rust/Haskell data
- [ ] Raw data saved for reproducibility
- [ ] Summary table of AILANG vs top-5 languages

**Risks:**
- AILANG may pass 0/20 trials → Still valuable — failure analysis is the primary deliverable
- Claude may not understand AILANG despite `extra_prompt` → Compare with/without prompt to measure information value

---

### Milestone 3: Adapt Track B — LeetCode Problems via Our Eval Harness (M3_TRACK_B)
**Goal:** Convert leetgptsolver's 100 LeetCode problems to our eval harness format and run with AILANG
**Estimated:** ~400 LOC adapter + runner
**Duration:** 1.5 days

**Tasks:**
- Download problem dataset from HuggingFace (`whiskwhite/leetcode-complete`)
- Write adapter: LeetCode problem JSON → AILANG benchmark YAML (`benchmarks/*.yml`)
- Filter problems to those expressible in functional AILANG (document exclusions)
- Generate AILANG test harnesses with expected stdout matching
- Run filtered problems across 3 models with 4 prompt strategies using `ailang eval-suite`
- Collect pass/fail, cost, tokens per (model, strategy, problem)

**Prompt Strategies:**
1. **Zero-shot**: "Write in AILANG" (no syntax help)
2. **Spec-prompted**: `ailang prompt` as system prompt
3. **Few-shot**: 3 solved examples from existing eval suite
4. **Spec + few-shot**: Both

**Models to Test:**
- Claude Opus 4.6 (`claude-opus-4-6`)
- Gemini 3.1 Pro (`gemini-3-1-pro`)
- GPT-5-4 (`gpt5-4`)

**Acceptance Criteria:**
- [ ] Problem adapter converts LeetCode problems to AILANG eval format
- [ ] Exclusion list documented with reasons (e.g., "requires mutable hash map")
- [ ] At least 40 problems pass filtering (expressible in AILANG)
- [ ] All 4 prompt strategies tested across 3 models
- [ ] Results comparable to published Python/Java/Rust/Elixir pass rates

---

### Milestone 4: Analysis + Gap Design Docs (M4_ANALYSIS)
**Goal:** Analyze cross-track results, identify failure modes, create design docs for every identified gap
**Estimated:** ~200 LOC scripts + 3–5 design docs
**Duration:** 1.5 days

**Tasks:**
- Aggregate results across both tracks
- Categorize failures: syntax error, type error, missing builtin, wrong algorithm, effect annotation error, module boilerplate error
- Create comparison tables: AILANG vs published results for Python, Ruby, JS, Go, Rust, Elixir, Haskell
- For each major gap category, create a design doc using `design-doc-creator` skill:
  - Missing builtins → `m-stdlib-gaps-from-xlang-bench.md`
  - Syntax confusion → `m-dx-syntax-ai-friendliness.md`
  - Type/effect errors → `m-dx-effect-inference-hints.md`
  - Module boilerplate → `m-dx-implicit-module.md`
  - Prompt gaps → `m-prompt-improvements-from-xlang.md`
- Write comparison report with recommendations
- Update CHANGELOG.md
- Submit upstream PR to `mame/ai-coding-lang-bench` (stretch goal)

**Acceptance Criteria:**
- [ ] Pass rate comparison table: AILANG vs 5+ other languages
- [ ] Cost comparison table: AILANG vs published data
- [ ] Failure mode taxonomy with counts and examples
- [ ] Design doc created for each major gap category (minimum 2, expected 3–5)
- [ ] Each gap design doc includes concrete examples from benchmark failures
- [ ] Top-3 recommendations for improving AILANG AI-friendliness
- [ ] CHANGELOG.md updated with results summary

---

## Success Metrics

- [ ] Bitwise builtins added and tested
- [ ] AILANG added to ai-coding-lang-bench fork, 20 trials completed
- [ ] 40+ LeetCode problems adapted and run across 3 models × 4 strategies
- [ ] AILANG pass rates measured and compared against published baselines
- [ ] Failure modes identified and categorized
- [ ] 3–5 gap design docs created with concrete benchmark evidence
- [ ] All tests passing (`make test`), all linting clean (`make lint`)

## Open Questions

- **Budget cap for API costs?** Track A: ~$7-15 for 20 trials. Track B: ~$50-200 for full matrix. Confirm acceptable range.
- **MiniHash alternative**: If bitwise builtins are too costly, could we substitute `sha256hex` and note the deviation?
- **Publish results publicly?** Blog post, upstream PR, or internal only?
- **Shebang support**: Does AILANG support `#!/usr/bin/env ailang run`? Or do we need a wrapper script?

## Notes

- Track A is now the **primary** track (highest credibility — uses their exact infrastructure and scoring)
- Track B is adapted (their problems, our runner) but still independently sourced
- The bitwise builtins are useful beyond this benchmark (hashing, binary protocols, etc.)
- Even 0% pass rate is a valid result — the failure analysis drives the gap design docs
- The prompt strategy comparison (zero-shot vs spec-prompted) is the most novel finding

---

**Sprint Plan created**: 2026-03-30
**Sprint Plan revised**: 2026-03-30 (pivot from "create own problems" to "integrate with third-party benchmarks")
