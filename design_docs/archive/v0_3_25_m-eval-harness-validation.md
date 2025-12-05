# M-EVAL: Enhanced Validation for Agent Benchmarks

**Status**: Planned
**Target**: v0.3.25 (Phase 1), v0.4.0+ (Phase 2)
**Priority**: P1 (Medium)
**Estimated**: 3 hours (Phase 1), 6 hours (Phase 2)
**Dependencies**: None

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | 0 | 0 | Infrastructure feature, no language syntax change |
| Preserve Semantic Clarity | 0 | 0 | No impact on language semantics |
| Increase Determinism | + | +1 | Enables benchmarks with deterministic file validation |
| Lower Token Cost | + | +1 | Enables more complex benchmarks that test real use cases, reducing need for workarounds |
| **Net Score** | | **+2** | **Decision: Move forward** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

**Rationale**: While this is infrastructure (not language design), it enables benchmarks that test deterministic real-world workflows (config generation, data transformation) which are core to AILANG's vision.

## Problem Statement

The current eval harness can only validate agent solutions via stdout comparison. This prevents testing benchmarks where the primary output is file generation (CSV converters, config generators, data exporters).

**Current State:**
- **Validation method**: Compare `stdout` to `expected_stdout` only
- **File location**: Single file at `benchmark/solution.ail` (or `solution.py`)
- **Multi-file support**: Agent can create additional files, but they're not validated
- **Limitation**: Cannot test benchmarks like:
  - CSV → JSON converter (validates output file exists and has correct structure)
  - Config file generator (validates config file schema)
  - Log analyzer (validates stats file produced)

**Impact:**
- **Who is affected**:
  - Benchmark authors trying to test realistic workflows
  - Agent evaluation (missing 8+ potential agent benchmarks)
  - v0.4 roadmap (normalization, import suggestions need file validation)

- **How significant**:
  - **Moderate**: Current workaround is to create files and print confirmation to stdout
  - **Blocking**: 3 new benchmarks need file validation to work properly
  - **Future**: Will block v0.4 normalization/determinism benchmarks

## Goals

**Primary Goal:** Enable agent benchmarks to validate generated files and complex outputs beyond stdout.

**Success Metrics:**
- ✅ File existence validation works (check file was created)
- ✅ JSON schema validation works (verify file structure)
- ✅ 3+ new benchmarks using file validation ship in v0.3.25
- ✅ Agent success rate improves on file-generation tasks
- ✅ Zero false positives (valid solutions don't fail validation)

## Solution Design

### Overview

Add optional `validation` section to benchmark YAML specs that runs after code execution. Two phases:

**Phase 1 (v0.3.25)**: File existence + basic JSON validation (~50 LOC)
**Phase 2 (v0.4.0+)**: Custom validation scripts for advanced checks (~150 LOC)

Both phases preserve existing stdout validation (backwards compatible).

### Architecture

**Phase 1: File Validation**

```yaml
# benchmarks/csv_to_json_converter.yml
id: csv_to_json_converter
# ... existing fields ...
validation:
  files_must_exist:
    - "users.json"
  json_schemas:
    users.json:
      type: "array"
      min_items: 1
      items:
        type: "object"
        required: ["name", "age", "email"]
```

**Execution Flow**:
1. Run solution code (existing)
2. Compare stdout (existing)
3. **NEW**: If `validation` section exists:
   - Check files in `files_must_exist` exist in workspace
   - Validate JSON files against schemas
4. Mark as success only if ALL checks pass

**Phase 2: Custom Validators**

```yaml
# benchmarks/recursive_directory_listing.yml
validation:
  script: "benchmarks/validators/check_directory_structure.sh"
  # OR inline:
  inline_script: |
    #!/bin/bash
    # Validate directory listing output
    [ -f file_list.txt ] || exit 1
    lines=$(wc -l < file_list.txt)
    [ "$lines" -eq 5 ] || exit 1
    exit 0
```

**Components:**

1. **ValidationConfig (Phase 1)**: ~30 LOC
   - YAML parsing for `validation` section
   - File existence checker
   - Basic JSON schema validator

2. **validateAgentSolution() (Phase 1)**: ~20 LOC
   - Called after running solution
   - Iterates through validation checks
   - Returns error with clear diagnostic

3. **Custom Script Runner (Phase 2)**: ~100 LOC
   - Execute bash scripts in workspace context
   - Security sandboxing (timeout, resource limits)
   - Capture output and exit code

4. **Validator Library (Phase 2)**: ~50 LOC
   - Reusable validator functions
   - JSON schema validation (use encoding/json)
   - File structure checking

### Implementation Plan

**Phase 1: File Validation** (~3 hours total)

**Milestone 1: Core Validation (~1.5 hours)**
- [ ] Add `ValidationConfig` struct to `BenchmarkSpec` (internal/eval_harness/spec.go)
- [ ] Implement `FilesMustExist` checker
- [ ] Implement basic JSON schema validation
- [ ] Add validation call to `RunAgentBenchmark()`

**Milestone 2: Testing (~1 hour)**
- [ ] Unit tests for validation logic
- [ ] Integration test with test benchmark
- [ ] Test error messages are clear

**Milestone 3: Documentation (~0.5 hours)**
- [ ] Update benchmarks/README.md with validation examples
- [ ] Add 3 example benchmarks using file validation
- [ ] Document YAML schema

**Phase 2: Custom Validators** (~6 hours total, deferred to v0.4.0+)

**Milestone 1: Script Execution (~2 hours)**
- [ ] Create `internal/eval_harness/validators.go`
- [ ] Implement `RunValidationScript()`
- [ ] Add timeout and resource limits

**Milestone 2: Security (~2 hours)**
- [ ] Sandboxing (chroot/containers if needed)
- [ ] Input validation (prevent code injection)
- [ ] Safe temp file handling

**Milestone 3: Validator Library (~1 hour)**
- [ ] Common validation functions
- [ ] Helper scripts in `benchmarks/validators/`

**Milestone 4: Testing & Docs (~1 hour)**
- [ ] Security tests (malicious scripts)
- [ ] Performance tests (timeouts work)
- [ ] Validator authoring guide

### Files to Modify/Create

**Phase 1 (~50 LOC total):**

**New files:**
- `internal/eval_harness/validation.go` - Validation logic (~30 LOC)
- `internal/eval_harness/validation_test.go` - Unit tests (~80 LOC)

**Modified files:**
- `internal/eval_harness/spec.go` - Add `ValidationConfig` field (~5 LOC)
- `internal/eval_harness/agent_runner.go` - Call validation after run (~15 LOC)

**Phase 2 (~150 LOC total):**

**New files:**
- `internal/eval_harness/validators.go` - Script runner (~100 LOC)
- `internal/eval_harness/validators_test.go` - Security tests (~120 LOC)
- `benchmarks/validators/README.md` - Validator authoring guide (~50 lines)
- `benchmarks/validators/example_validator.sh` - Example validator (~20 lines)

**Modified files:**
- `internal/eval_harness/validation.go` - Add script validation (~30 LOC)
- `benchmarks/README.md` - Document custom validators (~40 lines)

## Examples

### Example 1: CSV to JSON Converter (Phase 1)

**Benchmark YAML:**
```yaml
id: csv_to_json_converter
description: "Parse CSV and convert to JSON"
languages: ["ailang", "python"]
entrypoint: "main"
caps: ["IO", "FS"]
validation:
  files_must_exist:
    - "users.json"
  json_schemas:
    users.json:
      type: "array"
      min_items: 1
      items:
        type: "object"
        required: ["name", "age", "email"]
expected_stdout: |
  Converted 3 valid rows to users.json
```

**Agent Creates:**
- `benchmark/solution.ail` (main solution)
- `users.csv` (input file)
- `users.json` (output file) ← **Validated!**

**Validation Flow:**
1. ✅ Run solution → `ailang run --entry main --caps IO,FS benchmark/solution.ail`
2. ✅ Compare stdout → "Converted 3 valid rows to users.json" matches
3. ✅ **NEW**: Check `users.json` exists
4. ✅ **NEW**: Validate JSON structure (array of objects with required fields)
5. ✅ All checks pass → Success!

**Error Example** (if file missing):
```
Validation failed for benchmark csv_to_json_converter:
  Required file not found: users.json

Expected files:
  - users.json

Files found in workspace:
  - benchmark/solution.ail
  - users.csv

Suggestion: Check that writeFile("users.json", ...) is called
```

### Example 2: Config File Parser (Phase 1)

**Benchmark YAML:**
```yaml
id: config_file_parser
validation:
  files_must_exist:
    - "app_config.json"
  json_schemas:
    app_config.json:
      type: "object"
      required: ["app_name", "version", "port", "features"]
      properties:
        port:
          type: "integer"
          minimum: 1024
          maximum: 65535
expected_stdout: |
  Loaded MyApp v1.0.0 on port 8080 with 3 features
```

**Validates:**
- ✅ File `app_config.json` exists
- ✅ JSON has required fields
- ✅ Port is in valid range (1024-65535)

### Example 3: Recursive Directory Listing (Phase 2)

**Benchmark YAML:**
```yaml
id: recursive_directory_listing
validation:
  inline_script: |
    #!/bin/bash
    # Check file_list.txt was created
    [ -f file_list.txt ] || { echo "file_list.txt not found"; exit 1; }

    # Check it has 5 entries
    count=$(wc -l < file_list.txt)
    [ "$count" -eq 5 ] || { echo "Expected 5 files, got $count"; exit 1; }

    # Check all paths start with test_dir/
    grep -q "^test_dir/" file_list.txt || { echo "Paths don't start with test_dir/"; exit 1; }

    echo "Validation passed"
    exit 0
expected_stdout: |
  Found 5 files
```

**Custom validator checks:**
- ✅ Output file created
- ✅ Correct number of entries
- ✅ Path format is correct

## Success Criteria

**Phase 1:**
- [ ] `ValidationConfig` parses from YAML correctly
- [ ] File existence check works (pass when exists, fail when missing)
- [ ] JSON schema validation works (structural validation only)
- [ ] Clear error messages when validation fails
- [ ] 3 new benchmarks using file validation ship
- [ ] All existing benchmarks still pass (backwards compatible)
- [ ] Unit test coverage ≥90% on new code
- [ ] Documentation updated (benchmarks/README.md)

**Phase 2:**
- [ ] Custom validation scripts execute correctly
- [ ] Timeouts work (scripts can't hang eval suite)
- [ ] Security: malicious scripts don't escape workspace
- [ ] Inline scripts work (no need for separate file)
- [ ] Clear error messages from failed validators
- [ ] Example validators in benchmarks/validators/
- [ ] Validator authoring guide exists
- [ ] 2+ benchmarks using custom validators

## Testing Strategy

**Phase 1 Unit Tests:**
```go
// internal/eval_harness/validation_test.go

func TestFilesMustExist_AllPresent(t *testing.T) {
    // Create temp workspace with required files
    // Run validation → should pass
}

func TestFilesMustExist_Missing(t *testing.T) {
    // Create temp workspace WITHOUT file
    // Run validation → should fail with clear error
}

func TestJSONSchemaValidation_Valid(t *testing.T) {
    // Create valid JSON file
    // Run schema validation → should pass
}

func TestJSONSchemaValidation_Invalid(t *testing.T) {
    // Create JSON missing required field
    // Run validation → should fail with field name
}
```

**Phase 1 Integration Test:**
```bash
# Create test benchmark with file validation
# Run agent eval → should validate files correctly
ailang eval-suite --agent \
  --benchmark test_file_validation \
  --output test_results
```

**Phase 2 Security Tests:**
```go
func TestValidatorScript_Timeout(t *testing.T) {
    // Script with infinite loop
    // Should timeout after 10s
}

func TestValidatorScript_CantEscape(t *testing.T) {
    // Script trying to write to /tmp outside workspace
    // Should fail (sandboxed)
}

func TestValidatorScript_CantReadSecrets(t *testing.T) {
    // Script trying to read ~/.ssh/id_rsa
    // Should fail (sandboxed)
}
```

**Manual Testing:**
- Create each of the 3 new benchmarks
- Run in agent mode with claude-sonnet-4-5
- Verify validation catches intentional errors
- Verify validation passes on correct solutions

## Non-Goals

**Not in Phase 1:**
- Custom validation scripts - Deferred to Phase 2
- Content validation (regex on file contents) - Use Phase 2 validators
- Binary file validation - Out of scope (focus on text/JSON)
- Network request validation - Out of scope

**Not in Phase 2:**
- Complex schema validation (use JSON Schema spec fully) - Keep simple
- Interactive validators (user input) - Scripts must be non-interactive
- Validator marketplace/registry - Out of scope

## Timeline

**Phase 1 (v0.3.25)**

**Day 1** (2 hours):
- Milestone 1: Core validation implementation
- Add `ValidationConfig` to spec
- Implement file existence checker
- Implement basic JSON validation

**Day 2** (1 hour):
- Milestone 2: Testing
- Write unit tests
- Test with example benchmark
- Fix bugs

**Day 3** (0.5 hours):
- Milestone 3: Documentation
- Update README
- Create example benchmarks
- Commit and push

**Total Phase 1: ~3.5 hours across 3 days**

**Phase 2 (v0.4.0+)** - Deferred

**Week 1** (4 hours):
- Milestones 1-2: Script execution + security
- Implement script runner
- Add sandboxing

**Week 2** (2 hours):
- Milestones 3-4: Validator library + documentation
- Create helper validators
- Write authoring guide

**Total Phase 2: ~6 hours across 2 weeks**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| **JSON schema validation too complex** | High - May take >3 hours | Start with basic validation (type, required fields). Defer advanced features to Phase 2 |
| **False positives (valid solutions fail)** | High - Would break existing benchmarks | Extensive testing with real agent solutions. Make validation opt-in (no validation if section missing) |
| **Security issues in Phase 2 (script injection)** | High - Could compromise eval environment | Sandboxing (chroot or containers). Input validation. Resource limits. Security-focused testing |
| **Phase 2 scripts add too much complexity** | Medium - Hard to debug validators | Good error messages from validators. Example validators. Authoring guide |

## References

- **Current harness**: `internal/eval_harness/agent_runner.go` (lines 106-165)
- **Agent task templates**: `internal/eval_harness/templates/agent_task_ailang.txt`
- **Related analysis**: [AGENT_BENCHMARK_SOLUTIONS.md](../../AGENT_BENCHMARK_SOLUTIONS.md)
- **Benchmark audit**: [BENCHMARK_AUDIT_ANALYSIS.md](../../BENCHMARK_AUDIT_ANALYSIS.md)
- **New benchmarks**:
  - `benchmarks/csv_to_json_converter.yml`
  - `benchmarks/config_file_parser.yml`
  - `benchmarks/log_file_analyzer.yml`

## Future Work

**Post-v0.4.0 Enhancements:**
- **Semantic validation**: Check generated code compiles/runs (v0.4.1+)
- **Performance validation**: Check execution time/resource usage (v0.4.2+)
- **Comparative validation**: Compare output files across languages (v0.5.0+)
- **Snapshot testing**: Store expected file outputs, diff against them (v0.5.0+)

**Potential Extensions:**
- Visual diff for file outputs (for debugging)
- Validation result caching (skip validation if code unchanged)
- Parallel validation (run validators concurrently)

---

**Document created**: 2025-10-29
**Last updated**: 2025-10-29
