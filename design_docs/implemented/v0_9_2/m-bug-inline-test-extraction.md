# M-BUG-INLINE-TEST-EXTRACTION: Fix ExtractFunctionBinding for Module-less Files and Absolute Paths

**Status**: Implemented
**Target**: v0.9.2
**Priority**: P1 (High — blocks DocParse project inline tests on 17 modules)
**Estimated**: 1 hour
**Dependencies**: None
**Triggered by**: DocParse agent correction (msg_20260313_080101_01142c8d)

---

## Axiom Compliance

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Bug fix, no semantics change |
| A2: Replayability | +1 | Fixes inline test execution — more traces from tests |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | +1 | Enables local inline test verification on more files |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Agents can write inline tests without workarounds |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | No cost impact |
| A10: Composability | 0 | No composition changes |
| A11: Structured Failure | 0 | No error model changes |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +3** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Improves machine (agent) test authoring

---

## Problem Statement

`ExtractFunctionBinding` in `internal/testing/executor.go` (lines 245-318) creates a `pipeline.Config` and `pipeline.Source` that fail in two common scenarios, preventing inline tests from running on real-world AILANG files.

**Bug 1: Files without `module` declaration fail**

The pipeline source sets `IsREPL: false` with a real filename. The pipeline dispatcher (`pipeline.go:156-160`) routes all non-REPL files to `runModuleWithContext`, which expects module infrastructure. Files without `module` declarations produce Core output where functions are not findable.

Error: `"function 'testLength' not found in Core program"`

**Bug 2: Absolute paths cause MOD010 module path mismatch**

The pipeline config doesn't set `RelaxModules: true`. When `ailang test` is invoked with an absolute path, the canonical module ID becomes the full filesystem path which doesn't match relative module declarations.

Error: `"MOD010: module declaration 'examples/test_simple' doesn't match canonical path 'Users/mark/dev/..."`

**Impact:**
- DocParse project (17 AILANG modules) cannot use inline tests
- Any module-less `.ail` file's inline tests silently fail
- Running `ailang test` with absolute paths fails

---

## Solution Design

**Single file change:** `internal/testing/executor.go`

**Actual fix (differs from initial proposal — see Implementation Notes):**

For module-less files, write a temp file with a synthetic `module _test/<filename>` declaration prepended. This ensures:
1. The module pipeline reads from disk (which it always does) and finds a valid module
2. `ElaborateFile` takes the function-processing path (not the statement-only path for `Module==nil`)
3. All pipeline passes run (including OpLowering for arithmetic operators)

```go
pipelineFilename := e.modulePath
if sourceFile.Module == nil {
    // Write temp file with synthetic module so the module pipeline can load it
    tmpDir, _ := os.MkdirTemp("", "ailang-test-*")
    defer os.RemoveAll(tmpDir)
    baseName := strings.TrimSuffix(filepath.Base(e.modulePath), ".ail")
    tmpFile := filepath.Join(tmpDir, filepath.Base(e.modulePath))
    syntheticSource := fmt.Sprintf("module _test/%s\n\n%s", baseName, strippedSource)
    os.WriteFile(tmpFile, []byte(syntheticSource), 0644)
    pipelineFilename = tmpFile
}
cfg := pipeline.Config{
    Mode:         pipeline.ModeEval,
    RelaxModules: true,
}
```

**Why the original IsREPL approach failed:**
- `IsREPL: true` → `runSingle` uses `p.Parse()` which wraps functions as statements, producing 0 Core decls
- `IsREPL: false` + empty filename → `runSingle` with `ParseFile()` works for elaboration but misses the OpLowering pass that `runModule` includes
- Temp file approach ensures module pipeline runs with all passes

`RelaxModules: true` converts MOD010 errors to warnings for absolute path support

---

## Files to Modify

| File | Change | LOC |
|------|--------|-----|
| `internal/testing/executor.go` | Add `RelaxModules`, conditional `IsREPL` | ~3 |

---

## Success Criteria

- [x] Inline tests pass on files WITHOUT `module` declaration
- [x] Inline tests pass when run with absolute paths
- [x] Inline tests still pass on files WITH `module` declaration (no regression)
- [x] `make test` all green

---

## Related Documents

- [design_docs/implemented/v0_7_4/m-dx25-inline-tests-import-resolution.md](../../implemented/v0_7_4/m-dx25-inline-tests-import-resolution.md) — Prior inline test import fix
- [design_docs/planned/v0_9_2/m-dx23-inline-tests-documentation.md](m-dx23-inline-tests-documentation.md) — Inline test documentation

---

**Document created**: 2026-03-13
**Last updated**: 2026-03-13
**Implemented**: 2026-03-13
