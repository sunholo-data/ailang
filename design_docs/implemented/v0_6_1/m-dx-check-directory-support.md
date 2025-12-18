# M-DX: Add Directory Support to `ailang check`

**Status**: IMPLEMENTED
**Target**: v0.6.1
**Priority**: P2 (Low)
**Estimated**: 2 hours
**Dependencies**: None

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Removes need for shell loops in CI/scripts |
| Preserve Semantic Clarity | 0 | 0 | No change to semantics |
| Increase Determinism | + | +1 | Consistent ordering via filepath.Walk |
| Lower Token Cost | + | +1 | Single command vs multi-line find/xargs |
| **Net Score** | | **+3** | **Decision: Move forward** |

**Decision rule:** Net score > +1 -> Move forward | <= 0 -> Reject or redesign

## Problem Statement

The `ailang check` command currently only supports checking individual files. When passed a directory path, it fails with:

```
Error: cannot read file 'examples/': read examples/: is a directory
```

**Current State:**
- CI workflows must use workarounds: `find ... -exec ailang check {} \;`
- Manual checking of directories requires shell scripting
- Inconsistent with user expectations (most tools support directories)

**Impact:**
- CI workflows are more complex than necessary
- Users get confusing error messages
- Batch checking requires external scripting

## Goals

**Primary Goal:** Enable `ailang check` to recursively check all `.ail` files in a directory.

**Success Metrics:**
- `ailang check examples/` works and checks all .ail files
- `ailang check --timeout 60s examples/runnable/` works in CI
- Exit code reflects whether ANY file had errors (fail-fast or aggregate)

## Solution Design

### Overview

Add directory detection in the `check` command. When a directory is passed:
1. Walk the directory tree recursively
2. Filter for `.ail` files
3. Run check on each file
4. Aggregate results and report summary

### Architecture

**Components:**
1. **Directory Detection**: Check if path is file or directory using `os.Stat()`
2. **File Walker**: Use `filepath.Walk()` to find all `.ail` files
3. **Result Aggregator**: Collect pass/fail counts and report summary

### Implementation Plan

**Phase 1: Core Implementation** (~1.5 hours)
- [ ] Add `isDir` check in cmd/ailang/check.go
- [ ] Implement `walkAndCheck()` function
- [ ] Wire up file filtering (*.ail only)
- [ ] Add summary output (X files checked, Y errors)

**Phase 2: Polish** (~0.5 hours)
- [ ] Add `--recursive` flag (default: true for directories)
- [ ] Support glob patterns (e.g., `examples/**/*.ail`)
- [ ] Handle empty directories gracefully

### Files to Modify/Create

**Modified files:**
- `cmd/ailang/check.go` - Add directory handling (~50 LOC)

## Examples

### Example 1: Check Directory

**Before:**
```bash
# Fails with error
ailang check examples/
# Error: cannot read file 'examples/': read examples/: is a directory

# Workaround needed
find examples/runnable -name '*.ail' -type f -exec ailang check {} \;
```

**After:**
```bash
# Just works
ailang check examples/
# -> Type checking examples/runnable/hello.ail...
# -> Type checking examples/runnable/factorial.ail...
# ...
# Summary: 48 files checked, 0 errors
```

### Example 2: CI Usage

**Before:**
```yaml
- name: Check compilation
  run: |
    find examples/runnable -name '*.ail' -type f -exec ailang check --timeout 60s {} \; || true
```

**After:**
```yaml
- name: Check compilation
  run: ailang check --timeout 60s examples/runnable/
```

## Success Criteria

- [ ] `ailang check dir/` recursively checks all .ail files
- [ ] `ailang check file.ail` continues to work (no regression)
- [ ] Summary shows total files and errors
- [ ] Exit code 0 if all pass, non-zero if any fail
- [ ] Works with --timeout flag
- [ ] All tests passing
- [ ] CI workflow simplified

## Testing Strategy

**Unit tests:**
- Test directory detection
- Test file filtering (.ail only)
- Test empty directory handling

**Integration tests:**
- Test on examples/ directory
- Test with timeout flag
- Test error aggregation

**Manual testing:**
- Verify CI workflow passes
- Test on various directory structures

## Non-Goals

**Not in this feature:**
- Parallel checking - Would complicate implementation, defer to future
- Watch mode - Out of scope for this issue
- Exclude patterns - Can add later if needed

## Timeline

**Single session** (2 hours):
- Phase 1: Core implementation (1.5h)
- Phase 2: Polish and testing (0.5h)

**Total: ~2 hours**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Performance on large directories | Low | Use timeout flag, add progress output |
| Breaking existing scripts | Low | Single file path behavior unchanged |

## References

- CI failure: `ailang check --timeout 60s examples/` error
- Prior art: `go vet ./...`, `eslint dir/`, `ruff check dir/`

## Future Work

- `--parallel` flag for concurrent checking
- `--exclude` patterns for ignoring files
- `--json` output for programmatic consumption
- Watch mode (`--watch`)

---

**Document created**: 2025-12-17
**Last updated**: 2025-12-17
