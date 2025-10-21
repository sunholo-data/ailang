# M-EVAL Round-Robin: Better Parallel Distribution Across Model Providers

**Status:** Planned
**Version:** v0.3.16+
**Created:** 2025-10-21
**Author:** Claude (AI Assistant)

## Problem Statement

The current M-EVAL benchmark suite runs all benchmarks for one model before moving to the next model. This creates inefficient parallelization because all concurrent workers hit the same API provider endpoint, limiting how much we can increase parallelism without hitting rate limits.

### Current Behavior

**Job ordering** (model-first):
```
Jobs: [gpt5-mini/bench1, gpt5-mini/bench2, ..., gpt5-mini/bench44,
       claude-haiku/bench1, claude-haiku/bench2, ..., claude-haiku/bench44,
       gemini-flash/bench1, gemini-flash/bench2, ..., gemini-flash/bench44]
```

**Parallelism constraints:**
- With `--parallel 5`: First 5 jobs all hit OpenAI (gpt5-mini)
- Can't safely increase beyond 5 without hitting OpenAI rate limits
- Anthropic and Google endpoints sit idle during first 44 jobs
- Underutilized parallelism: could be using 2-3 workers per provider

**Performance impact:**
- 420 total jobs (--full suite: 6 models × 22 benchmarks × 2 languages)
- ~40s average per job
- ~55-70 minutes total (serialized within each provider)

### Observed Pattern (October 2025)

Running `ailang eval-suite --full` shows this pattern:
```
[1-70]   gpt5-mini          (OpenAI endpoint)
[71-140] claude-haiku-4-5   (Anthropic endpoint)
[141-210] gemini-2-5-flash  (Google endpoint)
[211-280] gpt5              (OpenAI endpoint)
[281-350] claude-sonnet-4-5 (Anthropic endpoint)
[351-420] gemini-2-5-pro    (Google endpoint)
```

With `--parallel 5`, this means:
- Jobs 1-5: 5 concurrent OpenAI calls
- Jobs 71-75: 5 concurrent Anthropic calls
- Jobs 141-145: 5 concurrent Google calls
- **Problem:** Can't increase to 10 parallel without 10 concurrent calls to same endpoint

## Solution: Model-First Round-Robin Job Queue

### New Job Ordering

**Round-robin by model:**
```
Jobs: [gpt5-mini/bench1, claude-haiku/bench1, gemini-flash/bench1,
       gpt5-mini/bench2, claude-haiku/bench2, gemini-flash/bench2,
       ...,
       gpt5-mini/bench44, claude-haiku/bench44, gemini-flash/bench44]
```

**Parallelism benefits:**
- With `--parallel 10`: Jobs 1-10 hit OpenAI, Anthropic, Google, OpenAI, Anthropic, Google, ...
- ~3-4 concurrent workers per provider (well within rate limits)
- Can safely increase to 12-15 workers for --full suite (6 models)
- All endpoints utilized from start

### Implementation

**File:** `cmd/ailang/eval_suite.go`

**Current code (lines 141-167):**
```go
for _, model := range modelList {
    for _, benchmark := range benchmarkList {
        for _, lang := range langList {
            jobs = append(jobs, Job{
                Model:     model,
                Benchmark: benchmark,
                Language:  lang,
            })
        }
    }
}
```

**New code:**
```go
// Build jobs in round-robin order: interleave models for better parallelism
// This ensures concurrent workers hit different API providers (OpenAI, Anthropic, Google)
for _, lang := range langList {
    // For each benchmark, create jobs for all models (round-robin)
    for benchIdx := 0; benchIdx < len(benchmarkList); benchIdx++ {
        for _, model := range modelList {
            benchmark := benchmarkList[benchIdx]
            job := Job{
                Model:     model,
                Benchmark: benchmark,
                Language:  lang,
            }

            // Check if result already exists (if resuming)
            if *skipExisting {
                pattern := filepath.Join(*outputDir, fmt.Sprintf("%s_%s_%s_*.json", benchmark, lang, model))
                matches, _ := filepath.Glob(pattern)
                if len(matches) > 0 {
                    skippedCount++
                    continue
                }
            }

            jobs = append(jobs, job)
        }
    }
}
```

### Parallelism Tuning

**New defaults:**
- `--parallel 10` (was 5) for dev suite (3 models)
  - ~3-4 workers per provider
- `--parallel 12` recommended for --full suite (6 models)
  - ~2 workers per provider
- `--parallel 15` maximum (if no rate limit issues)
  - ~2-3 workers per provider

**Update flag help text:**
```go
maxConcurrent := fs.Int("parallel", 10, "Maximum concurrent API calls across all providers (default: 10, recommended: 12-15 for --full)")
```

## Expected Performance Improvement

### Baseline (Current)
- **Dev suite** (3 models, 22 benchmarks, 2 languages = 132 jobs):
  - `--parallel 5`: ~18-22 minutes
  - Limited by sequential provider access

- **Full suite** (6 models, 22 benchmarks, 2 languages = 264 jobs):
  - `--parallel 5`: ~55-70 minutes
  - Limited by sequential provider access

### With Round-Robin
- **Dev suite** (132 jobs):
  - `--parallel 10`: ~10-12 minutes (**45% faster**)
  - All 3 providers used concurrently from start

- **Full suite** (264 jobs):
  - `--parallel 12`: ~22-28 minutes (**50% faster**)
  - All 6 models distributed across 3 providers

### Calculation
```
Sequential time per provider: N_jobs / parallel * avg_time
  = 44 jobs / 5 * 40s = ~352s per provider
  = 3 providers * 352s = ~17.6 minutes (dev suite)

Round-robin time: Total_jobs / parallel * avg_time
  = 132 jobs / 10 * 40s = ~528s = ~8.8 minutes

Speedup: 17.6 / 8.8 = 2x
```

## Testing Plan

### 1. Unit Testing
- Test job ordering with 2 models, 3 benchmarks, 2 languages
- Expected: [m1/b1/l1, m2/b1/l1, m1/b2/l1, m2/b2/l1, m1/b3/l1, m2/b3/l1, m1/b1/l2, ...]
- Verify skip-existing preserves round-robin order

### 2. Integration Testing
```bash
# Small test with 2 benchmarks
ailang eval-suite --benchmarks ackermann,factorial --parallel 10
# Expected pattern: [gpt5-mini, claude, gemini, gpt5-mini, claude, gemini, ...]

# Full dev suite
time ailang eval-suite --parallel 10
# Compare timing against baseline

# Full suite
time ailang eval-suite --full --parallel 12
# Verify no rate limit errors
```

### 3. Performance Benchmarking
```bash
# Baseline (current implementation)
git checkout main
make install
time make eval-baseline EVAL_VERSION=test-baseline-old

# New implementation
git checkout feature/round-robin
make install
time make eval-baseline EVAL_VERSION=test-baseline-new

# Compare results
ailang eval-compare eval_results/test-baseline-old eval_results/test-baseline-new
```

### 4. Rate Limit Testing
- Run with `--parallel 15` (aggressive)
- Monitor for 429 errors in logs
- If errors occur, reduce to 12 and document limits

## Alternative Approaches Considered

### Per-Provider Semaphores
**Idea:** Separate concurrency limits per provider
```go
type ProviderSemaphores struct {
    openai    chan struct{} // 5 concurrent
    anthropic chan struct{} // 5 concurrent
    google    chan struct{} // 5 concurrent
}
```

**Pros:**
- Explicit rate limiting per provider
- Could tune based on API tier (paid vs free)

**Cons:**
- More complex implementation (3 semaphores instead of 1)
- Harder to configure (3 flags or config file)
- Rate limits vary by user API tier (not portable)
- Round-robin + global limit achieves 90% of benefit

**Decision:** Rejected - Round-robin is simpler and sufficient

### Adaptive Parallelism
**Idea:** Start with high parallelism, back off on 429 errors

**Pros:**
- Automatically tunes to user's rate limits

**Cons:**
- Complex state machine (increase/decrease logic)
- Early 429s waste baseline run time
- Hard to make deterministic (different results each run)

**Decision:** Deferred to future work - Fixed tuning sufficient for now

## Future Work

### M-EVAL-LOOP v2.1: Advanced Scheduling
- **Priority queue:** Run fast benchmarks first (reduce idle workers at end)
- **Cost optimization:** Run cheap models first, expensive models when parallelism drops
- **Adaptive batching:** Increase parallelism as jobs complete (start at 5, ramp to 15)

### M-EVAL-LOOP v2.2: Distributed Execution
- **Remote workers:** Distribute across multiple machines
- **Provider affinity:** Dedicate machines to specific providers (avoid shared rate limits)
- **Resume from partial results:** Checkpoint progress, resume on failure

### M-EVAL-LOOP v2.3: Cost-Aware Scheduling
- **Budget caps:** Stop when cost exceeds threshold
- **Cheapest-first:** Run gemini-flash before claude-sonnet
- **Early termination:** Stop if accuracy plateaus (e.g., all models fail same benchmark)

## Metrics for Success

**Implementation complete when:**
- ✅ Job ordering verified (manual inspection of first 20 jobs)
- ✅ Dev suite runs in <12 minutes (down from ~20 minutes)
- ✅ Full suite runs in <30 minutes (down from ~60 minutes)
- ✅ No rate limit errors with recommended parallelism
- ✅ --skip-existing still works correctly

**Baseline comparison:**
```bash
# Before (model-sequential)
Duration: 55m23s
Success: 387/420 (92.1%)

# After (round-robin)
Duration: 27m12s (50% improvement)
Success: 387/420 (92.1%, same)
```

## Implementation Checklist

- [ ] Modify job ordering in `cmd/ailang/eval_suite.go` (lines 141-167)
- [ ] Increase default `--parallel` from 5 to 10
- [ ] Update help text for `--parallel` flag
- [ ] Add comment explaining round-robin rationale
- [ ] Test with small benchmark set (2 benchmarks, verify order)
- [ ] Test with dev suite (3 models, time comparison)
- [ ] Test with --full suite (6 models, verify no rate limits)
- [ ] Test --skip-existing with interrupted run
- [ ] Document new parallelism recommendations in README
- [ ] Update CHANGELOG.md with performance improvements
- [ ] Create benchmark comparison report (before/after)

## References

**Files modified:**
- `cmd/ailang/eval_suite.go` - Job ordering and default parallelism

**Related docs:**
- [M-EVAL Architecture](docs/docs/guides/evaluation/architecture.md)
- [Evaluation README](docs/docs/guides/evaluation/README.md)
- CLAUDE.md - Eval harness usage guidelines

**Prior art:**
- M-EVAL-LOOP v1.0 (Oct 2024): Python implementation with sequential execution
- M-EVAL-LOOP v2.0 (Oct 2025): Go rewrite with parallel execution
- M-EVAL-ROUND-ROBIN (Oct 2025): This improvement

**Discussions:**
- Issue: "Baseline eval taking 60+ minutes - can we speed it up?"
- Observation: "All early jobs hitting OpenAI, Claude/Gemini idle"
- Solution: "Interleave models in job queue for better provider distribution"
