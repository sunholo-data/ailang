# Vision-Aligned Benchmarks

This document explains how benchmarks test AILANG's vision goals and what metrics track progress toward those goals.

## Vision Goals → Benchmark Mapping

### 1. Explicit State Management (vs Python's Implicit State)

**Vision Problem**: Python's global state makes AI reasoning difficult. Same input → different output.

**Benchmark**: `explicit_state_threading.yml`

**What it tests**:
- ✅ **Behavioral**: Functions return (newState, result) tuples
- ✅ **Observable**: Correct state threading produces correct outputs
- 🔮 **Structural** (future): No global variables used, all state is explicit

**Current eval**: Checks stdout matches expected state progression.

**Enhanced eval**: Could parse AST to verify no global state mutations.

---

### 2. Deterministic Generation (One Canonical Form)

**Vision Problem**: Python has 5+ ways to transform a list. AI picks inconsistently.

**Benchmark**: `deterministic_list_transform.yml`

**What it tests**:
- ✅ **Behavioral**: Correct transformation output
- 🔮 **Consistency** (future): Same prompt → same code structure across runs

**Current eval**: Checks stdout for correct transformed list.

**Enhanced eval**:
- Run same benchmark 10 times with same model/seed
- Measure: `hash(generated_code)` consistency
- Target: ≥95% identical code structure

---

### 3. Effect System Safety (!: IO, FS, Net)

**Benchmarks**:
- `effect_tracking_io_fs.yml` - Basic effect declarations
- `effect_pure_separation.yml` - Pure/effectful separation
- `effect_composition.yml` - Effect propagation

**What they test**:
- ✅ **Behavioral**: Correct execution with effects
- 🔮 **Type-level** (future): Effect signatures match actual behavior

**Current eval**: Checks stdout and file creation.

**Enhanced eval**:
```bash
# Parse generated code to verify:
ailang check-effects generated.ail --verify-signatures
# Returns:
# ✅ computeSum: pure (no effects)
# ✅ printSum: !: IO (correct)
# ✅ processUser: !: IO,FS (correct)
# ❌ readFile: declared pure but does FS (ERROR)
```

**Metrics**:
- **Effect Safety Rate**: % of functions with correct effect declarations
- Target: ≥94% (from vision doc)

---

### 4. Total Functions (Exhaustive Pattern Matching)

**Benchmarks**:
- `exhaustive_pattern_matching.yml`
- `no_runtime_crashes_option.yml`

**What they test**:
- ✅ **Behavioral**: All cases handled, no crashes
- 🔮 **Compile-time** (future): Exhaustiveness verified by type checker

**Current eval**: Checks stdout for all expected outputs.

**Enhanced eval**:
```bash
# Compile generated code, verify exhaustiveness warnings:
ailang compile generated.ail --check-exhaustiveness
# Should report: "Pattern match on Status is exhaustive"
```

**Metrics**:
- **Totality Rate**: % of pattern matches that are exhaustive
- **Crash Rate**: % of runs that crash (should be 0%)

---

### 5. Type Safety (vs Python's Runtime Errors)

**Benchmark**: `type_safe_record_access.yml`

**What it tests**:
- ✅ **Behavioral**: Correct nested field access
- 🔮 **Compile-time** (future): Invalid field access rejected by compiler

**Current eval**: Checks stdout for correct value.

**Enhanced eval**:
```bash
# Test with intentionally broken code:
# Replace: user.profile.city
# With:    user.profile.country (doesn't exist)
ailang compile broken.ail
# Should fail with: "Field 'country' not found in Profile record"
```

**Metrics**:
- **Type Correctness Rate**: % of generated code that type-checks
- Target: ≥98% (from vision doc)

---

### 6. Referential Transparency

**Benchmark**: `referential_transparency.yml`

**What it tests**:
- ✅ **Behavioral**: Same input → same output (3 calls produce identical results)
- 🔮 **Purity** (future): No side effects in pure functions

**Current eval**: Checks that all three calls return same value.

**Enhanced eval**:
```bash
# Run same code 100 times, verify:
# - Same inputs always produce same outputs
# - No randomness, no time dependencies
# - Hash of output is consistent
```

**Metrics**:
- **Determinism Rate**: % of runs with identical outputs
- Target: ≥95% (from vision doc)

---

### 7. Canonical Code Structure

**Benchmark**: `canonical_normalization.yml`

**What it tests**:
- ✅ **Behavioral**: Correct functional composition
- 🔮 **Structural** (future): Code normalizes to identical form

**Current eval**: Checks stdout for correct result.

**Enhanced eval** (v0.4+):
```bash
# Generate code 10 times, normalize all variants:
for i in {1..10}; do
  ailang eval --benchmark canonical_normalization --model gpt5
  ailang normalize generated_$i.ail > normalized_$i.ail
done

# Check: all normalized versions are identical
diff normalized_*.ail
# Should show: 0 differences
```

**Metrics**:
- **Normalizer Lift**: % improvement in consistency after normalization
- Target: +20pp (from vision doc)

---

### 8. Immutable Data Structures

**Benchmark**: `immutable_data_structures.yml`

**What it tests**:
- ✅ **Behavioral**: Updates create new records, originals unchanged
- 🔮 **Immutability** (future): No in-place mutations

**Current eval**: Verifies all three records print correctly.

**Enhanced eval**:
```python
# Static analysis to detect mutations:
detect_mutations(generated_code)
# Flags: record.field = value (mutation!)
# Allows: new_record = {record | field: value} (immutable update)
```

---

## Current Evaluation Capabilities

### ✅ What Works Today (v0.3.14)

**Output Validation**:
```bash
ailang eval-suite --benchmark explicit_state_threading
# ✅ Compares stdout to expected_stdout
# ✅ Measures tokens, duration, cost
# ✅ Tracks success/failure per model
```

**Metrics Available**:
- Final success rate (compile + correct output)
- Zero-shot success (first attempt)
- Repair success rate (after error feedback)
- Token efficiency (AILANG vs Python)

### 🔮 Future Enhancements (v0.4+)

**Structural Validation**:
```bash
ailang eval-suite --validate-structure
# Check:
# - Effect declarations match usage
# - Pattern matches are exhaustive
# - No global state mutations
# - Type safety (all field accesses valid)
```

**Determinism Tracking**:
```bash
ailang eval-suite --runs=10 --measure-determinism
# Run each benchmark 10x with same seed
# Report: code hash consistency, output consistency
```

**Normalization Testing**:
```bash
ailang eval-suite --test-normalization
# Generate code → normalize → compare
# Measure: % of semantically equivalent code that normalizes identically
```

---

## Recommended Eval Enhancements

### Phase 1: Static Analysis (v0.3.15)

Add validation hooks to check generated code properties:

```go
// internal/eval_harness/validators.go
type StructuralValidator interface {
    ValidateEffects(code string) (bool, []string)
    ValidateExhaustiveness(code string) (bool, []string)
    ValidateImmutability(code string) (bool, []string)
}
```

Usage:
```bash
ailang eval-suite --validate=effects,exhaustiveness,immutability
```

### Phase 2: Multi-Run Consistency (v0.3.16)

```bash
ailang eval-suite --determinism-runs=10
# Generates:
# - determinism_report.json (per-benchmark consistency scores)
# - code_variance.md (shows variations in generated code)
```

### Phase 3: Normalization Baseline (v0.4.0)

```bash
ailang eval-baseline --with-normalization
# For each benchmark:
# 1. Generate code (all models)
# 2. Normalize each variant
# 3. Measure semantic equivalence
# 4. Report: normalizer lift (+Xpp improvement)
```

---

## Metrics Tracking

### Current Dashboard (`docs/static/benchmarks/latest.json`)

```json
{
  "aggregates": {
    "finalSuccess": 0.64,
    "zeroShotSuccess": 0.59,
    "repairSuccessRate": 0.14
  }
}
```

### Enhanced Dashboard (Future)

```json
{
  "aggregates": {
    "finalSuccess": 0.64,
    "typeCorrectness": 0.89,      // NEW: % passing type check
    "effectSafety": 0.94,          // NEW: % with correct effects
    "determinismRate": 0.91,       // NEW: consistent outputs
    "normalizerLift": 0.20         // NEW: consistency improvement
  },
  "visionAlignment": {
    "explicitState": 0.85,         // explicit_state_threading success
    "canonicalForm": 0.78,         // deterministic_list_transform success
    "effectTracking": 0.72,        // effect_* benchmarks avg
    "totalFunctions": 0.88,        // exhaustive_* benchmarks avg
    "typeSafety": 0.89,            // type_safe_* benchmarks avg
    "referentialTransparency": 0.91 // referential_transparency success
  }
}
```

---

## Running Vision Benchmarks

### Test All Vision Goals

```bash
# Run all vision-aligned benchmarks:
ailang eval-suite \
  --benchmarks explicit_state_threading,deterministic_list_transform,effect_tracking_io_fs,effect_pure_separation,effect_composition,exhaustive_pattern_matching,type_safe_record_access,referential_transparency,canonical_normalization,no_runtime_crashes_option,immutable_data_structures \
  --models gpt5,claude-sonnet-4-5,gemini-2-5-pro

# Generate vision-specific report:
ailang eval-report eval_results/vision_baseline v0.3.14 --format=vision
```

### Compare AILANG vs Python on Vision Goals

```bash
# Run with both languages:
ailang eval-suite --benchmarks explicit_state_threading --langs python,ailang

# Expected results:
# Python: May use global state (non-idiomatic but works)
# AILANG: Forces explicit state threading (only way to do it)
```

---

## Success Criteria

Based on vision doc targets:

| Metric | Target | Benchmark | Status |
|--------|--------|-----------|--------|
| **Compile Success** | ≥67% | All benchmarks | ⏳ Measuring |
| **Type Correctness** | ≥98% | type_safe_* | 🔮 Need validation |
| **Effect Safety** | ≥94% | effect_* | 🔮 Need validation |
| **Determinism Rate** | ≥95% | referential_transparency | 🔮 Need multi-run |
| **Model Pass Rate** | ≥70% | All benchmarks | ⏳ Measuring |
| **Normalizer Lift** | +20pp | canonical_normalization | 🔮 Need v0.4 |

**Next Steps**:
1. ✅ Run current eval suite on new vision benchmarks
2. 🔮 Add structural validators (v0.3.15)
3. 🔮 Implement multi-run determinism testing (v0.3.16)
4. 🔮 Add normalization baseline (v0.4.0)

---

## Interpreting Results

### High Success = Vision Goal Achieved

If a vision benchmark has >90% success:
- ✅ AI models understand the concept
- ✅ AILANG syntax supports it well
- ✅ Teaching prompt is effective

### Low Success = Opportunity for Improvement

If a vision benchmark has <50% success:
- 🔧 May need syntax improvements
- 🔧 May need better teaching examples
- 🔧 May need tooling support (error messages, suggestions)
- 📊 Still valuable: shows gap between vision and current AI capability

### Python vs AILANG Comparison

**Metrics to watch**:
1. **Token Efficiency**: AILANG should use fewer tokens (more concise)
2. **First-Attempt Success**: AILANG may be lower initially (unfamiliar syntax)
3. **Type Safety**: AILANG should catch errors at compile time (Python: runtime)
4. **Determinism**: AILANG code should be more consistent across runs

---

## Adding New Vision Benchmarks

Template:
```yaml
id: vision_goal_name
description: "Brief explanation of vision goal being tested"
languages: ["python", "ailang"]
entrypoint: "main"
caps: ["IO"]  # or ["IO", "FS", "Net"] as needed
difficulty: "easy|medium|hard"
expected_gain: "low|medium|high|very_high"
task_prompt: |
  Clearly describe the task.

  Emphasize the vision property being tested:
  - Explicit state
  - Effect tracking
  - Exhaustiveness
  - Type safety
  - Referential transparency

  Output only the code, no explanations.
expected_stdout: |
  Expected output
```

**Guidelines**:
- Focus on observable behavior (stdout)
- Highlight properties that differentiate AILANG from Python
- Make success/failure unambiguous
- Include edge cases that test totality
