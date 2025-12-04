# Sprint Plan: M-DX11 Relaxed Module Matching

## Summary
Implement relaxed module path validation (MOD010) with auto-relaxation for temp paths, environment variable and CLI flag support, and warning-based feedback instead of silent failures.

**Duration:** 1.5 days (~8 hours)
**Dependencies:** None
**Risk Level:** Low

## Current Status Analysis

### Completed Recently
- v0.5.1: Go codegen fixes (lambda calls, list patterns, pattern bindings) - ~200 LOC
- v0.5.0: AI effect, prompt versioning - ~500 LOC
- v0.4.8: Import aliasing - ~300 LOC

### Velocity
- Recent average: ~150-200 LOC/day for focused features
- Estimated capacity: ~120 LOC implementation for this sprint
- Note: This is a DX feature, not language feature - simpler implementation

### From Design Doc
- ⏳ Temp path auto-relaxation: ~25 LOC
- ⏳ Warning infrastructure: ~30 LOC
- ⏳ Environment variable support: ~15 LOC
- ⏳ CLI flag support: ~20 LOC
- ⏳ Improved error messaging: ~30 LOC
- **Total: ~120 LOC**

## Proposed Milestones

### Milestone 1: Temp Path Detection & Warning Infrastructure
**Goal:** Add `IsTempPath()` function and warning deduplication system
**Estimated:** 55 LOC implementation + 40 LOC tests = 95 LOC
**Duration:** 0.5 days (4 hours)

**Tasks:**
- Hour 1-2: Implement `IsTempPath()` in `internal/loader/loader.go`
  - Conservative detection: `os.TempDir()`, `/tmp/`, `/var/folders/`
  - Windows support: `%TEMP%` prefix
  - Unit tests for various path patterns
- Hour 3-4: Add warning infrastructure in `internal/pipeline/`
  - Add `hasWarnedForPath map[string]bool` to config
  - Create `warnMOD010Relaxed()` function
  - Emit warning once per file on mismatch

**Acceptance Criteria:**
- [ ] `IsTempPath("/tmp/foo.ail")` returns true
- [ ] `IsTempPath("/var/folders/xyz/foo.ail")` returns true (macOS)
- [ ] `IsTempPath("./src/foo.ail")` returns false
- [ ] Warning emitted only once per file path
- [ ] All tests passing
- [ ] Linting clean

**Files to Create/Modify:**
- `internal/loader/loader.go` - Add `IsTempPath()` (~25 LOC)
- `internal/loader/loader_test.go` - Unit tests (~40 LOC)
- `internal/pipeline/config.go` - Add warning tracker (~15 LOC)
- `internal/pipeline/pipeline_module.go` - Add `warnMOD010Relaxed()` (~15 LOC)

**Risks:**
- Platform-specific temp paths may vary - Mitigation: Use `os.TempDir()` as primary, known patterns as fallback

### Milestone 2: Relaxation Config & Integration
**Goal:** Add `RelaxModules` config option with env var and CLI flag support
**Estimated:** 35 LOC implementation + 20 LOC tests = 55 LOC
**Duration:** 0.5 days (3 hours)

**Tasks:**
- Hour 1: Add `RelaxModules bool` to pipeline Config
  - Read `AILANG_RELAX_MODULES` env var (values: 1, true, yes)
  - Wire into existing config parsing
- Hour 2: Add `--relax-modules` flag to CLI
  - Update `cmd/ailang/run.go`
  - Update `cmd/ailang/check.go`
  - Pass through to pipeline config
- Hour 3: Integration testing
  - Test env var enables relaxation
  - Test CLI flag enables relaxation
  - Verify strict mode unchanged

**Acceptance Criteria:**
- [ ] `AILANG_RELAX_MODULES=1 ailang run` enables relaxation
- [ ] `ailang run --relax-modules` enables relaxation
- [ ] Default behavior (strict) unchanged
- [ ] Warning emitted when mismatch occurs in relaxed mode
- [ ] All tests passing

**Files to Create/Modify:**
- `internal/pipeline/config.go` - Add `RelaxModules` field (~5 LOC)
- `cmd/ailang/run.go` - Add `--relax-modules` flag (~10 LOC)
- `cmd/ailang/check.go` - Add `--relax-modules` flag (~10 LOC)
- `internal/pipeline/pipeline_module.go` - Check relaxation before MOD010 (~10 LOC)

**Risks:**
- Flag parsing conflicts - Mitigation: Standard Go flag library, well-tested

### Milestone 3: Improved Error Messaging & Documentation
**Goal:** Update error messages to suggest relaxation options and document the feature
**Estimated:** 30 LOC implementation + documentation
**Duration:** 0.5 days (2 hours)

**Tasks:**
- Hour 1: Update error messages
  - Strict mode: suggest `--relax-modules` or `AILANG_RELAX_MODULES=1`
  - Relaxed mode: explain how to get strict again
  - Temp-path: explain auto-relaxation
- Hour 2: Documentation
  - Update CLAUDE.md with feature documentation
  - Update CLI help text
  - Add examples to design doc

**Acceptance Criteria:**
- [ ] Strict mode error suggests relaxation options
- [ ] Relaxed mode warning mentions how to get strict
- [ ] CLAUDE.md documents the feature
- [ ] CLI `--help` shows `--relax-modules` flag
- [ ] Design doc moved to implemented/

**Files to Create/Modify:**
- `internal/pipeline/pipeline_module.go` - Update error messages (~20 LOC)
- `CLAUDE.md` - Document feature (~30 lines)
- `cmd/ailang/run.go` - Update help text (~5 lines)
- `cmd/ailang/check.go` - Update help text (~5 lines)

**Risks:**
- None significant

## Success Metrics
- Test coverage: All new code has unit tests
- Integration tests: 3+ scenarios (temp, env var, CLI flag)
- Documentation: CLAUDE.md, CLI help updated
- All tests passing: `make test`
- All linting passing: `make lint`

## Example Verification

**Create example file to verify feature:**
```bash
# Create temp file with mismatched module
echo 'module test/hello
let main = print("Hello")' > /tmp/test_relax.ail

# Test 1: Should fail in strict mode (default)
ailang check /tmp/test_relax.ail 2>&1 | grep -q "MOD010"

# Test 2: Should warn but succeed with --relax-modules
ailang run --relax-modules --caps IO --entry main /tmp/test_relax.ail

# Test 3: Should warn but succeed with env var
AILANG_RELAX_MODULES=1 ailang run --caps IO --entry main /tmp/test_relax.ail

# Cleanup
rm /tmp/test_relax.ail
```

## Dependencies
- None - standalone feature

## Open Questions
- None - design doc is comprehensive and reviewed

## Notes
- This is a DX improvement, not a language feature
- Conservative approach: temp detection fails safe (doesn't auto-relax if uncertain)
- Warnings instead of silence ensures users know what's happening
- Future-compatible with `ailang build`/`ailang package` strict modes
