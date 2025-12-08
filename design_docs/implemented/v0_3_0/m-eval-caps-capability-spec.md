# M-EVAL-CAPS: Capability Specification in Benchmark YAML

**Status**: ✅ IMPLEMENTED (v0.3.0 or earlier)
**Completed**: 2025-11-02 (documentation and final benchmark updates)
**Original Target**: v0.4.2
**Actual Implementation**: v0.3.0 or earlier
**Priority**: P1 (Medium-High)

---

## ✅ Implementation Note (2025-11-02)

**This feature was ALREADY FULLY IMPLEMENTED in v0.3.0 or earlier!**

**What we found:**
- Schema support: ✅ `BenchmarkSpec.Caps []string` existed since v0.2.0-v0.3.0
- Runner integration: ✅ Passes `--caps` flag correctly (runner.go:170-172)
- Agent integration: ✅ Agent runner also uses caps (agent_runner.go:441)
- Benchmark coverage: ✅ 39 of 41 benchmarks already had caps specified
- Real-world validation: ✅ Benchmarks work correctly in v0.3.16 baseline (zero CAP_001 errors except test-specific case)

**What we completed (2025-11-02):**
- Added `caps: ["IO"]` to `float_eq.yml` (previously missing)
- Added `caps: ["IO"]` to `numeric_modulo.yml` (previously missing)
- Moved design doc from `planned/v0_4_1/` to `implemented/v0_3_0/`
- **Result**: 41/41 benchmarks now have capability specifications ✅

**Evidence from v0.3.16 baseline:**
- `api_call_json` with `caps: ["Net", "IO"]` - ✅ Works perfectly (12/12 model runs successful)
- `simple_print` with `caps: ["IO"]` - ✅ Works perfectly (12/12 model runs successful)
- Only CAP_001 errors: 3 occurrences in `targeted_repair_test` (test-specific, not system issue)

**Files changed (completion):**
- `benchmarks/float_eq.yml`: Added `caps` and `entrypoint`
- `benchmarks/numeric_modulo.yml`: Added `caps` and `entrypoint`

---

## Original Design Doc (Historical Reference)

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | No language change |
| Preserve Semantic Clarity | + | +1 | Makes capability requirements explicit and machine-readable |
| Increase Determinism | + | +1 | Benchmarks declare what they need, eval harness grants exactly that |
| Lower Token Cost | 0 | 0 | No impact on code generation |
| **Net Score** | | **+2** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

Benchmarks currently fail with CAP_001 errors because the eval harness doesn't grant the capabilities that solutions require. This creates false negatives in eval results.

**Current State:**
- Benchmarks like `api_call_json` require Net capability, but harness doesn't grant it
- Models correctly use `getEnv()` and `httpRequest()`, but get CAP_001 runtime errors
- No way to specify required capabilities in benchmark YAML
- Eval harness uses hardcoded default caps: `IO,FS` (from `internal/eval_harness/harness.go:119`)
- Results show compilation success but runtime failure (misleading metrics)

**Impact:**
- **False negatives**: Correct AILANG code marked as failed
- **Metrics pollution**: Success rates appear lower than reality
- **Developer confusion**: Why does my code fail when it compiles?
- **v0.4.1 regression**: New Env effect usage causes failures in benchmarks that previously passed

**Evidence from v0.4.1 test:**
```
api_call_json (gpt5-mini):
- Generated code: ✅ CORRECT (uses httpRequest, getEnvOr, decode)
- Compilation: ✅ SUCCESS
- Runtime: ❌ FAILED (CAP_001: effect 'Env' requires capability, but none provided)
```

## Goals

**Primary Goal:** Enable benchmarks to declare required capabilities, and have eval harness automatically grant them

**Success Metrics:**
- Zero CAP_001 errors in benchmarks that correctly specify capabilities
- 100% of benchmarks with effects have capabilities field in YAML
- Eval results accurately reflect code correctness (not capability mismatches)
- Backwards compatible: benchmarks without `capabilities` use default `IO,FS`

## Solution Design

### Overview

Add optional `capabilities` field to benchmark YAML schema. Eval harness reads this field and grants specified capabilities when executing AILANG code.

### Architecture

**Components:**
1. **Benchmark Schema Extension**: Add `capabilities: []string` field to YAML
2. **Harness Integration**: Read capabilities from benchmark spec, pass to compiler
3. **Default Behavior**: If no capabilities specified, use current default `IO,FS`
4. **Validation**: Warn if benchmark uses effects not in capabilities (best-effort)

**Data Flow:**
```
benchmark.yaml
  ↓
  capabilities: [IO, Net, Env]
  ↓
harness.go (loadBenchmark)
  ↓
ailang run --caps IO,Net,Env solution.ail
  ↓
Runtime grants exactly those capabilities
```

### Implementation Plan

**Phase 1: Schema & Parsing** (~2 hours)
- [ ] Add `Capabilities []string` to `Benchmark` struct in `internal/eval_harness/types.go`
- [ ] Update YAML unmarshaling to read `capabilities` field
- [ ] Add validation: warn if empty/nil (use defaults), error if invalid capability name
- [ ] Unit test: parse benchmark with/without capabilities field

**Phase 2: Harness Integration** (~1.5 hours)
- [ ] Modify `runAILANG()` in `internal/eval_harness/harness.go` to use `benchmark.Capabilities`
- [ ] If `Capabilities` is nil/empty, use default `[]string{"IO", "FS"}`
- [ ] Pass capabilities to `ailang run --caps` command
- [ ] Unit test: verify correct caps passed to compiler

**Phase 3: Benchmark Updates** (~1 hour)
- [ ] Add `capabilities` field to benchmarks using Net effect:
  - `api_call_json.yaml`: `capabilities: [IO, Net]`
  - `http_get_simple.yaml`: `capabilities: [IO, Net]`
  - `http_post_json.yaml`: `capabilities: [IO, Net]`
- [ ] Add `capabilities` field to benchmarks using Env effect:
  - `env_variable_access.yaml`: `capabilities: [IO, Env]`
  - `config_from_env.yaml`: `capabilities: [IO, Env, FS]`
- [ ] Verify existing benchmarks work with default (no `capabilities` field)

**Phase 4: Testing & Documentation** (~1.5 hours)
- [ ] Integration test: run benchmark with Net capability, verify no CAP_001
- [ ] Integration test: run benchmark WITHOUT required capability, verify CAP_001 (safety check)
- [ ] Update `benchmarks/README.md` with `capabilities` field documentation
- [ ] Update `docs/guides/evaluation/architecture.md` with capability workflow

### Files to Modify/Create

**New files:**
- None

**Modified files:**
- `internal/eval_harness/types.go` - Add `Capabilities []string` field (~10 LOC)
- `internal/eval_harness/harness.go` - Use capabilities in `runAILANG()` (~20 LOC)
- `benchmarks/api_call_json.yaml` - Add `capabilities: [IO, Net]` (~1 LOC)
- `benchmarks/http_*.yaml` - Add `capabilities: [IO, Net]` (~5 benchmarks × 1 LOC = 5 LOC)
- `benchmarks/env_*.yaml` - Add `capabilities: [IO, Env]` or `[IO, Env, FS]` (~3 benchmarks × 1 LOC = 3 LOC)
- `benchmarks/README.md` - Document `capabilities` field (~30 LOC)
- `internal/eval_harness/types_test.go` - Add parsing tests (~50 LOC)
- `internal/eval_harness/harness_test.go` - Add integration tests (~80 LOC)

**Total new code: ~200 LOC**

## Examples

### Example 1: API Call with Network Access

**Before (v0.4.1):**
```yaml
# benchmarks/api_call_json.yaml
id: api_call_json
name: "HTTP API Call with JSON Response"
description: "Make an HTTP request and parse JSON response"
languages:
  - ailang
  - python
expected_stdout: "200\n"
# ❌ No capability specification - harness grants only IO,FS
```

**Result:** CAP_001 error (Net capability not granted)

**After (v0.4.2):**
```yaml
# benchmarks/api_call_json.yaml
id: api_call_json
name: "HTTP API Call with JSON Response"
description: "Make an HTTP request and parse JSON response"
languages:
  - ailang
  - python
capabilities:  # ✅ Explicit capability declaration
  - IO
  - Net
expected_stdout: "200\n"
```

**Result:** ✅ Success (harness runs: `ailang run --caps IO,Net solution.ail`)

### Example 2: Environment Variable Access

**Before:**
```yaml
# benchmarks/config_from_env.yaml
id: config_from_env
name: "Read Configuration from Environment"
description: "Load API key and config path from env variables"
# ❌ No capability specification
```

**Result:** CAP_001 error (Env capability not granted)

**After:**
```yaml
# benchmarks/config_from_env.yaml
id: config_from_env
name: "Read Configuration from Environment"
description: "Load API key and config path from env variables"
capabilities:  # ✅ Explicit capability declaration
  - IO
  - Env
  - FS  # Also needs FS to read config file
```

**Result:** ✅ Success

### Example 3: Backwards Compatibility

**Current benchmark (no changes needed):**
```yaml
# benchmarks/fizzbuzz.yaml
id: fizzbuzz
name: "FizzBuzz"
description: "Classic FizzBuzz algorithm"
# No capabilities field
```

**Behavior:** Harness uses default `[IO, FS]` (backwards compatible)

## Success Criteria

- [ ] Benchmark YAML can specify `capabilities: [IO, Net, FS, Env]`
- [ ] Eval harness reads capabilities and passes to `ailang run --caps`
- [ ] Default capabilities `[IO, FS]` used if no field present
- [ ] CAP_001 errors eliminated for benchmarks with correct capability specs
- [ ] Validation warns if capabilities field is empty (should use defaults or be explicit)
- [ ] All tests passing
- [ ] Documentation updated (benchmarks/README.md, docs/guides/evaluation/)
- [ ] At least 8 benchmarks updated with explicit capabilities

## Testing Strategy

**Unit tests:**
- Parse benchmark YAML with `capabilities` field
- Parse benchmark YAML without `capabilities` (use defaults)
- Validate capability names (reject invalid like "INVALID_CAP")
- Verify default behavior (nil capabilities → `[IO, FS]`)

**Integration tests:**
- Run benchmark with Net capability, verify httpRequest succeeds
- Run benchmark with Env capability, verify getEnv succeeds
- Run benchmark WITHOUT required capability, verify CAP_001 error (safety)
- Run benchmark with no capabilities field, verify IO/FS work

**Manual testing:**
- Run full eval suite with updated benchmarks
- Verify api_call_json no longer fails with CAP_001
- Check that Python benchmarks still work (capabilities only affect AILANG)

## Non-Goals

**Not in this feature:**
- **Auto-inference of capabilities from code** - Too complex, requires static analysis (deferred to v0.5.0+)
- **Per-language capabilities** - All languages in benchmark get same capabilities (AILANG-specific behavior)
- **Capability validation at parse time** - Runtime validation is sufficient for now
- **Capability composition (supersets)** - Just list exactly what you need

## Timeline

**Day 1** (8 hours):
- Morning (4h): Phase 1 & 2 implementation (schema, parsing, harness integration)
- Afternoon (2h): Phase 3 (update 8-10 benchmark YAMLs)
- Evening (2h): Phase 4 (testing, documentation)

**Total: ~8 hours across 1 day**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Benchmarks forget to add capabilities | Medium | Good defaults (IO, FS cover most cases) |
| Invalid capability names cause errors | Low | Validation warns/errors on unknown capability |
| Python benchmarks affected | Low | Capabilities only passed to AILANG runtime |
| Breaking change for external tools | Low | Backwards compatible (no field = defaults) |

## References

- Current harness code: `internal/eval_harness/harness.go:119` (default caps hardcoded)
- Benchmark schema: `internal/eval_harness/types.go` (Benchmark struct)
- AILANG capability system: `internal/effects/` (runtime capability checking)
- V0.4.1 test results: `eval_results/test_v0.4.1_phase2/` (shows CAP_001 failures)
- Root cause analysis: `V0.4.0_FAILURE_ROOT_CAUSE_ANALYSIS.md` (mentions capability mismatches)

## Future Work

- **Auto-inference** (v0.5.0): Analyze generated code, detect required effects, auto-add capabilities
- **Capability budgets** (v0.6.0): Limit resources (e.g., max 5 HTTP requests, 10 file reads)
- **Effect-to-capability mapping** (v0.5.0): Declare which effects require which capabilities in schema
- **Validation at parse time** (v0.5.0): Check benchmark YAML for consistency (e.g., description mentions "HTTP" but no Net capability)

---

**Document created**: 2025-11-01
**Last updated**: 2025-11-01
