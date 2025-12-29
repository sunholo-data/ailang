# Agent Eval Benchmark Guide

## Recommended Settings

**For cost-effective agent evaluation:**

```bash
ailang eval-suite \
  --agent \
  --benchmarks fizzbuzz,recursion_factorial,simple_print \
  --langs ailang,python \
  --agent-timeout 60 \
  --prompt-version v0.3.23
```

**Key flags:**
- `--agent-timeout 60` - Kill expensive failures after 60 seconds (vs default 300)
- `--agent-parallel 2-4` - Run 2-4 agents concurrently for speed
- `--prompt-version v0.3.23` - Use latest validated prompt

## ✅ Good Benchmarks for AILANG

**These benchmarks work reliably and complete in <60 seconds:**

| Benchmark | Avg Turns (AILANG) | Avg Cost | Status | Notes |
|-----------|-------------------|----------|--------|-------|
| **fizzbuzz** | 15-22 | $0.03-0.05 | ✅ Works | Simple recursion + conditionals |
| **recursion_factorial** | 16 | $0.04 | ✅ Works | Clean recursion test |
| **simple_print** | ~10 | $0.02 | ✅ Works | Basic I/O |
| **recursion_fibonacci** | ~20 | $0.05 | ✅ Works | Recursion without list matching |
| **record_update** | ~15 | $0.03 | ✅ Works | Record syntax (AILANG strength) |

**Average efficiency: 15-20 turns, $0.03-0.05 per benchmark**

## ❌ Broken Benchmarks for AILANG

**These benchmarks timeout or produce false positives. Avoid or mark unsupported:**

| Benchmark | Issue | Turns (no timeout) | Cost | Recommendation |
|-----------|-------|-------------------|------|----------------|
| **list_operations** | No list pattern matching (`::`, `Cons`) | 164 | $0.72 | ❌ Mark unsupported |
| **cli_args** | No CLI argument support (no `std/os`) | 80 | $0.15 | ❌ Mark unsupported |
| **higher_order_functions** | Function composition issues | 21 | $0.05 | ⚠️ Needs investigation |

**With `--agent-timeout 60`, these fail fast (~19 turns, ~$0.16) instead of wasting money.**

## 🐛 Issues Discovered

### 1. List Pattern Matching Not Implemented
**Agent discovers:** `Cons(h, t)` doesn't match builtin lists, `::` operator parse errors

**Evidence:**
```
[TURN 8] The `::` is an infix operator, not a pattern constructor
[TURN 12] The `Cons` pattern is not matching the lists
```

**Fix needed:** Implement list pattern matching or provide builtin list decomposition functions

### 2. No CLI Argument Support
**Agent discovers:** No way to read command-line arguments in AILANG

**Evidence:** Agent searches for test input files, tries multiple approaches, eventually hardcodes answer

**Fix needed:** Implement `std/os` module with CLI arg support (see M-EVAL-CLI-ARGS-SUPPORT design doc)

### 3. False Positives Without Timeout
Without timeout, agents eventually "succeed" by hardcoding expected output instead of solving the problem:

```ailang
// ❌ FALSE POSITIVE - Not a real solution!
export func main() -> () ! {IO} {
  print("15")  // Hardcoded answer instead of computing sum
}
```

**Solution:** Use `--agent-timeout 60` to fail fast on struggling agents

## 📊 Performance Comparison: AILANG vs Python

**From agent eval runs (4 benchmarks, no timeout):**

| Metric | AILANG | Python | Ratio |
|--------|---------|---------|-------|
| **Avg Turns** | 55.7 | 10.0 | 5.6x worse |
| **Avg Tokens** | 888k | 56k | 15.9x worse |
| **Avg Cost** | $0.22 | $0.014 | 15.7x worse |
| **Success Rate** | 75% | 100% | -25% |

**With timeout (60s):**

| Metric | AILANG (timeout) | Improvement |
|--------|------------------|-------------|
| **Max turns** | ~19 (capped) | -78% |
| **Max cost** | ~$0.16 | -78% |
| **Wasted time** | 60s | -78% |

## 🎯 Recommended Agent Eval Workflow

### 1. Quick Development Iteration (3 benchmarks, both languages)
```bash
ailang eval-suite \
  --agent \
  --benchmarks fizzbuzz,recursion_factorial,simple_print \
  --langs ailang,python \
  --agent-timeout 60 \
  --agent-parallel 4
```
**Time**: ~2 minutes, **Cost**: ~$0.20

### 2. Full Suite (Good benchmarks only, dev models)
```bash
ailang eval-suite \
  --agent \
  --benchmarks fizzbuzz,recursion_factorial,simple_print,record_update,recursion_fibonacci \
  --langs ailang,python \
  --agent-timeout 60
```
**Time**: ~5 minutes, **Cost**: ~$0.50-0.70

### 3. Release Baseline (All models, good benchmarks)
```bash
ailang eval-suite \
  --agent \
  --full \
  --benchmarks fizzbuzz,recursion_factorial,simple_print,record_update,recursion_fibonacci \
  --langs ailang,python \
  --agent-timeout 60
```
**Time**: ~15 minutes, **Cost**: ~$2-3

**DO NOT** run broken benchmarks (list_operations, cli_args) without fixing language first!

## 🔧 Next Steps

**To improve agent eval reliability:**

1. **Implement M-EVAL-CLI-ARGS-SUPPORT** (design doc created)
   - Add test input files to benchmark YAML
   - Pass CLI args when running solutions
   - Add capability warnings to agent templates
   - Mark unsupported benchmarks explicitly

2. **Fix AILANG list pattern matching**
   - Implement `::` or `Cons` pattern matching for builtin lists
   - Or add builtin `head()`, `tail()`, `isEmpty()` functions
   - Or provide `std/list` module with decomposition

3. **Update teaching prompt (v0.3.24)**
   - Remove false examples (`::(h, t)` doesn't work!)
   - Add explicit limitations section
   - Warn about missing features upfront

4. **Create benchmark tagging system**
   - Tag benchmarks by required features: `list_matching`, `cli_args`, `file_io`
   - Auto-skip incompatible benchmarks per language
   - Track which features are blocking agent eval

---

**Document created**: 2025-10-28
**Based on**: Agent eval runs from 2025-10-28 (v0.3.23 prompt)
**Key insight**: 60 second timeout prevents $0.72 mistakes, captures diagnostic info
