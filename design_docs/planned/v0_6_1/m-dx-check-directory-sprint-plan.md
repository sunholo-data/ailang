# Sprint Plan: M-DX Check Directory Support

**Sprint ID**: M-DX-CHECK-DIR
**Target Version**: v0.6.1
**Estimated Duration**: 2 hours (single session)
**Risk Level**: Low

## Summary

Add directory support to `ailang check` command so it can recursively check all `.ail` files in a directory, replacing complex `find ... -exec` workarounds in CI and manual workflows.

## Current Status Analysis

### Existing Implementation
- **File**: [cmd/ailang/check.go](cmd/ailang/check.go) (292 LOC)
- `checkFile()` function accepts a single filename
- Timeout support already exists (`--timeout` flag)
- Debug compile flag exists (`--debug-compile`)
- No directory handling - fails with "is a directory" error

### Velocity (Last 7 Days)
Based on recent commits:
- M-LETREC-SCOPING fix
- WASM REPL improvements
- Example verification updates
- Average: ~150-200 LOC/day for focused features

## Proposed Milestones

### M1: Directory Detection & Walking (~1 hour)
**Estimated LOC**: 80 (implementation) + 50 (tests) = 130 LOC

**Tasks**:
1. Add `os.Stat()` check at start of `checkFile` to detect directories
2. Create `checkDirectory()` function using `filepath.Walk()`
3. Filter for `.ail` files only (skip other extensions)
4. Call existing `checkFile` logic for each file found

**Acceptance Criteria**:
- [ ] `ailang check examples/` recursively finds all .ail files
- [ ] Non-.ail files are silently skipped
- [ ] Empty directories handled gracefully (no error, just "0 files")
- [ ] Maintains compatibility: single file path still works

**Files to Modify**:
- `cmd/ailang/check.go` - Add directory detection and walking

### M2: Result Aggregation & Summary (~30 min)
**Estimated LOC**: 40 (implementation) + 30 (tests) = 70 LOC

**Tasks**:
1. Track pass/fail counts during directory walk
2. Collect error messages from all files
3. Print summary at end: "N files checked, M errors"
4. Return proper exit code (0 if all pass, 1 if any fail)

**Acceptance Criteria**:
- [ ] Summary shows total files checked
- [ ] Summary shows number of errors (if any)
- [ ] Exit code 0 when all files pass
- [ ] Exit code 1 when ANY file has errors
- [ ] All errors displayed, not just first

**Files to Modify**:
- `cmd/ailang/check.go` - Add result aggregation

### M3: Integration with Existing Flags (~30 min)
**Estimated LOC**: 20 (implementation) + 20 (tests) = 40 LOC

**Tasks**:
1. Ensure `--timeout` applies to entire directory check (not per-file)
2. Ensure `--debug-compile` works with directory mode
3. Update usage message to show directory support
4. Update help text

**Acceptance Criteria**:
- [ ] `ailang check --timeout 60s examples/` works
- [ ] `ailang check --debug-compile examples/` shows timings for all files
- [ ] Help text updated

**Files to Modify**:
- `cmd/ailang/check.go` - Wire up existing flags
- `cmd/ailang/main.go` - Update usage message

## Implementation Approach

### Architecture Decision
Keep it simple - modify `checkFile` to detect directories and delegate:

```go
func checkFile(filename string, ...) {
    info, err := os.Stat(filename)
    if err != nil {
        // Handle error
    }

    if info.IsDir() {
        checkDirectory(filename, ...)
        return
    }

    // Existing single-file logic
}

func checkDirectory(dir string, ...) {
    var passed, failed int
    var errors []string

    filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
        if !strings.HasSuffix(path, ".ail") {
            return nil
        }
        // Check file, collect results
    })

    // Print summary
}
```

### Non-Goals (Deferred)
- `--recursive` flag (default true, no opt-out needed initially)
- `--parallel` flag (future optimization)
- `--exclude` patterns (can add later)
- Glob patterns (`examples/**/*.ail`) - shell already expands these

## Testing Strategy

### Unit Tests
```go
func TestCheckDirectory_Basic(t *testing.T) {
    // Create temp dir with .ail files
    // Run checkDirectory
    // Verify all files checked
}

func TestCheckDirectory_Empty(t *testing.T) {
    // Create empty temp dir
    // Run checkDirectory
    // Verify graceful handling
}

func TestCheckDirectory_MixedFiles(t *testing.T) {
    // Create dir with .ail and .txt files
    // Run checkDirectory
    // Verify only .ail files checked
}
```

### Integration Tests
- Run on `examples/runnable/` directory
- Verify with `--timeout` flag
- Test error aggregation on `examples/` (has some failing files)

## Success Metrics

- [ ] `ailang check examples/runnable/` completes successfully
- [ ] `ailang check examples/` shows summary with pass/fail counts
- [ ] CI workflows can simplify from `find -exec` to single command
- [ ] All existing single-file usage unchanged (no regression)
- [ ] New tests added and passing

## Total Estimated LOC

| Component | Implementation | Tests | Total |
|-----------|---------------|-------|-------|
| M1: Directory Walking | 80 | 50 | 130 |
| M2: Result Aggregation | 40 | 30 | 70 |
| M3: Flag Integration | 20 | 20 | 40 |
| **Total** | **140** | **100** | **240** |

## Timeline

Single session (~2 hours):
- M1: 1 hour
- M2: 30 min
- M3: 30 min

## Dependencies

None - this is a self-contained CLI enhancement.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Performance on large directories | Low | Existing `--timeout` flag |
| Breaking existing scripts | Low | Single file behavior unchanged |
| Exit code semantics unclear | Low | Follow standard: 0=success, 1=failure |

---

**Created**: 2025-12-18
**Status**: Ready for execution
