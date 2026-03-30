# Sprint Plan: M-EVAL-XLANG — Cross-Language AI Code Generation Benchmark

## Summary

Measure AILANG's AI code generation performance against mainstream languages using two reproducible benchmark tracks: algorithmic problems (LeetCode-style) and agentic coding (mini-git). Prioritized by feasibility — algorithmic first (no missing builtins), mini-git second (needs `mkdir` builtin).

**Duration:** 6 days (~40 hours)
**Dependencies:** None (existing eval harness, existing builtins)
**Risk Level:** Medium (API costs, unknown AILANG pass rates)
**Design Doc:** [m-eval-cross-language-benchmark.md](m-eval-cross-language-benchmark.md)

## Current Status Analysis

### Completed Recently
- ✅ String builtins: split, join, replace, trim, startsWith, endsWith, chars, words, reverse, find, slice
- ✅ Crypto builtins: sha256hex, hmacsha256, constanttimeequal
- ✅ FS builtins: readFile, writeFile, appendFile, exists, listDir
- ✅ 53 existing benchmarks in eval harness
- ✅ Multi-model agent eval infrastructure (models.yml, agent_runner)

### Velocity
- Recent 14 days: ~237 commits, ~7684 insertions across 72 files
- Average: ~549 LOC/day
- Estimated capacity: ~2700 LOC for this 5-day sprint

### Remaining from Design Doc
- ⏳ Track B: Algorithmic benchmark (~1200 LOC)
- ⏳ Track A: Mini-git benchmark (~800 LOC, blocked on `mkdir`)
- ⏳ Missing FS builtins: mkdir, isDir, isFile (~150 LOC)
- ⏳ Analysis and reporting (~200 LOC scripts + prose)

### AILANG Capability Audit

| Capability | Status | Notes |
|------------|--------|-------|
| String manipulation | ✅ Complete | split, join, replace, trim, startsWith, etc. |
| SHA-256 hashing | ✅ Available | `_crypto_sha256hex` — sufficient for mini-git |
| File read/write | ✅ Available | readFile, writeFile, appendFile, exists |
| Directory listing | ✅ Available | listDir (sorted) |
| **Directory creation** | ❌ Missing | **No `mkdir` — blocks mini-git init** |
| **isDir/isFile** | ❌ Missing | Can't distinguish files from directories |
| Pattern matching | ✅ Available | match expressions, ADTs |
| List processing | ✅ Available | map, filter, foldl, length, head, reverse |
| JSON encode/decode | ✅ Available | For structured data |

---

## Proposed Milestones

### Milestone 1: Algorithmic Benchmark Setup (M1_ALGO_SETUP)
**Goal:** Create 20 AILANG-compatible algorithmic problems with test harnesses
**Estimated:** ~600 LOC problems + ~200 LOC harness = ~800 LOC
**Duration:** 1.5 days

**Tasks:**
- Day 1 AM: Audit AILANG capabilities → select 20 LeetCode-style problems that are expressible (no mutable graphs, no complex I/O)
- Day 1 AM: Create `benchmarks/cross-language/` directory structure
- Day 1 PM: Write first 10 problem files with test harnesses (easy/medium: two_sum, fibonacci, palindrome, binary_search, merge_sort, etc.)
- Day 2 AM: Write remaining 10 problem files (medium/hard: tree traversal, dynamic programming, string parsing, etc.)
- Day 2 AM: Create 4 prompt strategy templates (zero-shot, spec-prompted, few-shot, spec+few-shot)

**Problem Selection Criteria:**
- Must compile and run in current AILANG (no missing builtins)
- Must be algorithmically interesting (not trivial)
- Must have unambiguous expected output
- Good coverage: recursion, list ops, string ops, math, pattern matching

**Candidate Problems:**
1. Two Sum (list + nested loop/map)
2. Fibonacci (recursion/memoization)
3. Palindrome Check (string + recursion)
4. Binary Search (list + recursion)
5. Merge Sort (list + recursion + split/merge)
6. FizzBuzz (control flow — baseline)
7. Roman Numeral Converter (pattern matching)
8. Balanced Parentheses (string + stack-as-list)
9. Run-Length Encoding (string manipulation)
10. GCD/LCM (math + recursion)
11. Matrix Transpose (list of lists)
12. Caesar Cipher (string + chars + map)
13. Flatten Nested List (recursion + ADT)
14. Binary Tree Sum (ADT + recursion)
15. Reverse Polish Notation (list-as-stack + pattern matching)
16. Word Frequency Count (string + map/fold)
17. Longest Common Prefix (string + list)
18. Valid Anagram (string + sort)
19. Pascal's Triangle (recursion + list generation)
20. JSON-like Key-Value Parser (string manipulation)

**Acceptance Criteria:**
- [ ] 20 problem `.ail` files in `benchmarks/cross-language/algorithmic/problems/`
- [ ] Each problem has a test harness that prints "ALL TESTS PASSED" on correct solution
- [ ] Each reference solution compiles and passes (`ailang run` each file)
- [ ] 4 prompt templates in `benchmarks/cross-language/algorithmic/prompts/`
- [ ] README.md with problem list and difficulty ratings

**Risks:**
- Some problems may be inexpressible in AILANG → Mitigation: audit each before writing, have 5 backup problems

---

### Milestone 2: Algorithmic Benchmark Execution (M2_ALGO_RUN)
**Goal:** Run all 20 problems across 3 models × 4 prompt strategies and collect results
**Estimated:** ~300 LOC runner script + API costs
**Duration:** 1 day

**Tasks:**
- Day 2 PM: Write benchmark runner script (`benchmarks/cross-language/algorithmic/run.sh`)
  - For each (model, strategy, problem): send prompt → extract code → write to .ail → `ailang run` → check output
- Day 2 PM: Run pilot (3 problems × 1 model × 1 strategy) to validate runner
- Day 3 AM: Full run: 20 problems × 3 models × 4 strategies = 240 API calls
- Day 3 AM: Collect raw results to JSON

**Models to Test:**
- Claude Opus 4.6 (`claude-opus-4-6`)
- Gemini 2.5 Flash (`gemini-2.5-flash`)
- GPT-5 Mini (`gpt-5-mini`)

**Prompt Strategies:**
1. **Zero-shot**: "Write a function in AILANG that..." (no syntax help)
2. **Spec-prompted**: `ailang prompt` output as system prompt + problem
3. **Few-shot**: 3 solved AILANG examples + problem
4. **Spec + few-shot**: Both spec and examples + problem

**Acceptance Criteria:**
- [ ] Runner script executes all 240 combinations
- [ ] Raw results captured as JSON (model, strategy, problem, pass/fail, tokens, cost, code)
- [ ] Pilot run validates the pipeline end-to-end
- [ ] Total API cost tracked and reported

**Risks:**
- API rate limits → Mitigation: add retry with backoff, run sequentially per model
- High cost → Mitigation: pilot with 3 problems first, estimate total before full run

---

### Milestone 3: Add Missing FS Builtins + Mini-Git (M3_MINIGIT)
**Goal:** Add `mkdir`/`isDir`/`isFile` builtins, then create AILANG mini-git reference implementation for Track A
**Estimated:** ~150 LOC builtins + ~400 LOC mini-git + ~100 LOC tests = ~650 LOC
**Duration:** 1.5 days

**Tasks:**
- Day 3 PM: Add `_fs_mkdir`, `_fs_mkdirAll`, `_fs_isDir`, `_fs_isFile` to `internal/builtins/fs.go`
- Day 3 PM: Register new builtins, add tests, `make test`
- Day 4 AM: Fork/vendor the mini-git spec from `mame/ai-coding-lang-bench`
- Day 4 AM: Write AILANG reference implementation of mini-git (init, add, commit, log)
- Day 4 PM: Adapt test suite for AILANG execution
- Day 4 PM: Create AILANG spec file with `ailang prompt` context for agent

**New Builtins:**
```go
// internal/builtins/fs.go additions
_fs_mkdir(path: string) -> Result((), string)       // Create single directory
_fs_mkdirAll(path: string) -> Result((), string)    // Create directory tree (mkdir -p)
_fs_isDir(path: string) -> bool                     // Check if path is directory
_fs_isFile(path: string) -> bool                    // Check if path is regular file
```

**Acceptance Criteria:**
- [ ] New FS builtins pass unit tests
- [ ] AILANG mini-git reference implementation compiles and passes all tests
- [ ] Mini-git spec file includes `ailang prompt` context
- [ ] Test suite validates init, add, commit, log operations
- [ ] `make test` and `make lint` clean

**Risks:**
- Mini-git spec may require features AILANG lacks (e.g., date formatting) → Mitigation: simplify spec where needed, document deviations
- Reference implementation may be hard to write → Mitigation: start with v1 (init/add/commit/log), defer v2 (status/diff/checkout/reset)

---

### Milestone 4: Analysis, Gap Identification, and Design Docs (M4_ANALYSIS)
**Goal:** Analyze results, identify failure modes, write comparison report, and **create design docs for every gap identified**
**Estimated:** ~200 LOC analysis scripts + 3–5 design docs + prose
**Duration:** 1.5 days

**Tasks:**
- Day 5 AM: Write result aggregation script (pass rates, costs, token counts per model × strategy)
- Day 5 AM: Categorize failures: syntax error, type error, missing builtin, wrong algorithm, compilation timeout
- Day 5 PM: Create comparison tables: AILANG vs published results for Python, Ruby, JS, Go, Rust, Elixir
- Day 5 PM: Write `benchmarks/cross-language/analysis/comparison.md` with findings
- Day 5 PM: Write `benchmarks/cross-language/analysis/failure_modes.md` with top-3 failure patterns
- Day 6 AM: **For each identified gap, create a design doc** using `design-doc-creator` skill:
  - Missing builtins → design doc for stdlib additions
  - Syntax confusion → design doc for syntax improvements or better prompts
  - Type system issues → design doc for type system DX improvements
  - Error message confusion → design doc for better error messages
  - Pattern gaps → design doc for prompt/teaching improvements
- Day 6 AM: Update CHANGELOG.md, link gap design docs from benchmark analysis

**Key Analysis Questions:**
1. Where does AILANG's pass rate land relative to other languages? (vs Python ~87%, vs Elixir ~52%)
2. How much does the `ailang prompt` spec improve pass rates over zero-shot?
3. Which model generates the best AILANG code?
4. What are the most common failure modes? (syntax? types? missing builtins? wrong algorithms?)
5. What's the cost per successful solution in AILANG vs other languages?
6. **What specific AILANG changes would close the gap with Python?** (design doc per gap)

**Expected Gap Categories and Design Doc Outputs:**

| Gap Category | Example | Design Doc |
|-------------|---------|------------|
| Missing builtins | No `sort`, no `dict`/`map` literal | `m-stdlib-gaps-from-xlang-bench.md` |
| Syntax confusion | Models write `let x = ...` without `in` | `m-dx-syntax-ai-friendliness.md` |
| Type errors | Models forget effect annotations | `m-dx-effect-inference-hints.md` |
| Module boilerplate | Models forget `module` / `export func main()` | `m-dx-implicit-module.md` |
| Prompt gaps | `ailang prompt` missing critical info | `m-prompt-improvements-from-xlang.md` |

**Acceptance Criteria:**
- [ ] Pass rate comparison table: AILANG vs 5+ other languages
- [ ] Cost comparison table: AILANG vs published data
- [ ] Failure mode taxonomy with counts and examples
- [ ] **Design doc created for each major gap category identified** (minimum 2, expected 3–5)
- [ ] Each gap design doc includes concrete examples from benchmark failures
- [ ] Top-3 recommendations for improving AILANG's AI-friendliness
- [ ] CHANGELOG.md updated with results summary
- [ ] Original design doc updated with actual benchmark data

**Risks:**
- Results may be poor → Mitigation: this is expected and valuable — the gaps ARE the deliverable
- Too many gaps to address → Mitigation: prioritize by frequency, create design docs for top 5 only
- Small sample size (20 problems) → Mitigation: report confidence intervals, note limitations

---

## Success Metrics

- [ ] 20 algorithmic problems created and validated
- [ ] 240 API evaluations completed (20 × 3 models × 4 strategies)
- [ ] AILANG pass rates measured and compared against published baselines
- [ ] Failure modes identified and categorized
- [ ] Missing FS builtins added (`mkdir`, `isDir`, `isFile`)
- [ ] Mini-git reference implementation compiles and runs
- [ ] Comparison report published in `benchmarks/cross-language/analysis/`
- [ ] **3–5 gap design docs created** in `design_docs/planned/` for identified improvements
- [ ] Each gap design doc traceable to specific benchmark failure examples
- [ ] CHANGELOG.md updated
- [ ] All tests passing (`make test`)
- [ ] All linting clean (`make lint`)

## Dependencies

- API access to Claude, Gemini, and GPT-5 (existing keys)
- `ailang prompt` command working (verified)
- Existing eval harness infrastructure (verified)

## Open Questions

- **Budget cap for API costs?** 240 calls at ~$0.05–0.50 each = $12–120 total. Confirm acceptable range.
- **Mini-git scope**: Full v1+v2 (8 commands) or v1 only (4 commands: init, add, commit, log)?
- **Publish results publicly?** Blog post, GitHub discussion, or internal only?

## Notes

- Track B (algorithmic) is prioritized over Track A (mini-git) because it needs zero new builtins
- The 4 prompt strategies are the most interesting part — they measure how much AILANG's machine-readable spec helps models that have never seen AILANG
- If AILANG zero-shot pass rate is >0%, that's remarkable (models have never been trained on AILANG)
- Even "bad" results are valuable: they tell us exactly what to improve in prompts, stdlib, and syntax

---

**Sprint Plan created**: 2026-03-30
