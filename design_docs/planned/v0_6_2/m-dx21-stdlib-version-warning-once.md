# M-DX21: Show Stdlib Version Warning Once Per Session

**Status**: Planned
**Target**: v0.6.2
**Priority**: P3 (Low - cosmetic improvement)
**Estimated**: 1 hour
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
- Warning shown once per CLI invocation
- Subsequent stdlib loads don't repeat warning
- Real warnings still visible
- Option to suppress entirely with flag or env var

## Solution Design

### Overview

Add a package-level flag to track if the warning has been shown, and skip subsequent warnings in the same process.

### Implementation

```go
var stdlibVersionWarningShown bool

func checkStdlibVersion(expected, found string) {
    if expected != found && !stdlibVersionWarningShown {
        fmt.Fprintf(os.Stderr, "Warning: stdlib version mismatch: expected %s, found %s\n", expected, found)
        stdlibVersionWarningShown = true
    }
}
```

### Optional Enhancement

Add environment variable to suppress entirely:

```go
if os.Getenv("AILANG_NO_VERSION_WARNINGS") != "" {
    return
}
```

### Files to Modify

**Modified files:**
- `internal/loader/stdlib.go` or equivalent (~10 LOC)

## Success Criteria

- [ ] Warning shown only once per process
- [ ] Multiple module compilations show warning once
- [ ] Real warnings still visible
- [ ] Optional: `AILANG_NO_VERSION_WARNINGS=1` suppresses entirely
- [ ] All existing tests pass

## Non-Goals

**Not in this feature:**
- Fixing the underlying version mismatch
- Config file for warning preferences

## References

- stapledons_voyage DX Feedback (agent message)

---

**Document created**: 2025-12-20
**Last updated**: 2025-12-20
