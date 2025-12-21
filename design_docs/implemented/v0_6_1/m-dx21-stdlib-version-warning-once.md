# M-DX21: Show Stdlib Version Warning Once Per Session

**Status**: Implemented
**Target**: v0.6.1
**Priority**: P3 (Low - cosmetic improvement)
**Estimated**: 1 hour
**Actual**: 15 minutes
**Dependencies**: None
**Reporter**: stapledons_voyage (agent message)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No semantic changes |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | No verification changes |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Cleaner output for AI agents to parse |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | No cost changes |
| A10: Composability | 0 | No composability changes |
| A11: Structured Failure | 0 | No error handling changes |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +1** → **Decision: Move forward (minor improvement)**

### Hard Violation Check

- [x] A1 (Determinism): No changes to determinism
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Improves machine (AI agent) experience

## Problem Statement

The stdlib version mismatch warning appears on every compilation, creating noise in the output.

**Current State:**
```
$ ailang compile module1.ail
Warning: stdlib version mismatch: expected dev, found v0.6.0

$ ailang compile module2.ail
Warning: stdlib version mismatch: expected dev, found v0.6.0

$ ailang compile module3.ail
Warning: stdlib version mismatch: expected dev, found v0.6.0
```

**Issue:**
- On development laptops, stdlib may be from a release while CLI is dev
- Warning is shown every single compilation
- Obscures real warnings and errors
- Noisy CI/CD logs

## Goals

**Primary Goal:** Show stdlib version warning only once per session/process.

**Success Metrics:**
- [x] Warning shown once per CLI invocation
- [x] Subsequent stdlib loads don't repeat warning
- [x] Real warnings still visible
- [x] Optional: `AILANG_NO_VERSION_WARNINGS=1` suppresses entirely

---

## Implementation Report

### Solution Applied

Added a package-level boolean flag to track if warning has been shown:

```go
// internal/loader/stdlib_resolver.go

// stdlibVersionWarningShown tracks if we've already shown the version mismatch warning
// M-DX21: Show warning only once per process to reduce noise
var stdlibVersionWarningShown bool

// In ResolveStdlib(), where warning is printed:
if err := r.checkStdlibVersion(searchPath); err != nil {
    if r.strictMode {
        return "", err
    }
    // M-DX21: Non-strict: log warning only once per process
    // Also check AILANG_NO_VERSION_WARNINGS to suppress entirely
    if !stdlibVersionWarningShown && os.Getenv("AILANG_NO_VERSION_WARNINGS") == "" {
        fmt.Fprintf(os.Stderr, "Warning: %v\n", err)
        stdlibVersionWarningShown = true
    }
}
```

### Files Modified

| File | Changes | LOC |
|------|---------|-----|
| `internal/loader/stdlib_resolver.go` | Added `stdlibVersionWarningShown` flag and env var check | +10 |

**Total: 10 LOC**

### Features Implemented

1. **Warning shown once**: Package-level `stdlibVersionWarningShown` flag set after first warning
2. **Suppress entirely**: `AILANG_NO_VERSION_WARNINGS=1` environment variable skips all warnings

### Usage

```bash
# Default: warning shown once per process
$ ailang compile *.ail
Warning: stdlib version mismatch: expected dev, found v0.6.0
# (subsequent files don't repeat warning)

# Suppress entirely
$ AILANG_NO_VERSION_WARNINGS=1 ailang compile *.ail
# (no warning shown)
```

---

## Success Criteria

- [x] Warning shown only once per process
- [x] Multiple module compilations show warning once
- [x] Real warnings still visible
- [x] Optional: `AILANG_NO_VERSION_WARNINGS=1` suppresses entirely
- [x] All existing tests pass

## References

- stapledons_voyage DX Feedback (agent message)

---

**Document created**: 2025-12-20
**Last updated**: 2025-12-21
**Implemented**: 2025-12-21
