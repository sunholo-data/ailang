# M-UX2: Dev Experience Polish

**Status**: 📋 DEFERRED from v0.3.0 to v0.4.0
**Priority**: P2 (NICE TO HAVE)
**Estimated**: 450 LOC (350 impl + 100 tests)
**Duration**: 2 days
**Dependencies**: None
**Deferred From**: v0.3.0 Implementation Plan
**Target**: v0.4.0
**Note**: Includes fixing existing `--quiet` flag wiring

## Problem Statement

**Current State (v0.3.0)**: Core functionality works but developer experience has rough edges.

### Problem 1: No Global Debug Control

```bash
# ❌ CURRENT: Debug output always enabled or always disabled (hardcoded)
# Want: Toggle debug output without recompiling

ailang run file.ail                 # No debug output
ailang run --debug file.ail         # ❌ Flag doesn't exist yet
AILANG_DEBUG=1 ailang run file.ail  # ❌ Env var not supported yet
```

**Root Cause**:
- Debug logging scattered across codebase with no global control
- No CLI flag for `--debug`
- No environment variable support
- Must edit source code and recompile to toggle debug output

**Impact**:
- Hard to diagnose issues in production
- Can't selectively enable debug for specific runs
- Debug output clutters normal usage

### Problem 1b: Quiet Flag Not Fully Wired Up

```bash
# ⚠️ PARTIALLY WORKING: --quiet flag defined but not used everywhere
ailang run --quiet file.ail         # ✅ Flag exists
                                    # ❌ Still shows some progress messages
```

**Root Cause** (from [cmd/ailang/main.go:37](cmd/ailang/main.go#L37)):
```go
// quietMode variable defined but set to placeholder
_ = false // quietMode placeholder for future use
quietFlag := flag.Bool("quiet", false, "Suppress progress messages (only show program output)")
// Later: _ = *quietFlag  (discards the value!)
```

The `--quiet` flag is parsed but its value is discarded with `_ = *quietFlag`. Progress messages like `→ Type checking...`, `→ Effect checking...`, `✓ Running...` are still shown.

**What works**:
- Flag parsing: `--quiet` accepted
- Some progress suppression: `if !quiet { ... }` in `runFile()`

**What doesn't work**:
- Progress messages still shown (type checking, effect checking, running)
- Global quietMode not set
- Other parts of codebase don't respect quiet flag

**Impact**:
- Can't get clean output for scripting/automation
- Progress messages clutter CI logs
- `--quiet` flag misleading (doesn't fully suppress output)

### Problem 2: Capability Auto-Detection Not Enhanced

```bash
# ❌ CURRENT: Manual --caps required for Clock/Net effects
ailang run --caps IO,Clock timer.ail    # Must specify manually
ailang run timer.ail                     # Fails with capability error

# ✅ WANT: Auto-detection suggests required capabilities
ailang run timer.ail
# Error: Missing capability: Clock
# Hint: Run with --caps Clock (detected usage: _clock_now at line 5)
```

**Root Cause**:
- Audit script (`tools/audit-examples.sh`) only detects IO and FS
- Clock and Net effects not scanned
- No capability suggestion in error messages

**Impact**:
- New users must manually figure out which capabilities to add
- Error messages not actionable
- Extra friction for Clock/Net adoption

### Problem 3: Limited Micro Examples

**Current (v0.3.0)**:
- 5 new micro examples added (blocks, clock, net, records)
- Covers basic use cases

**Want (v0.4.0)**:
- 10+ micro examples covering edge cases
- Examples for:
  - Clock: deterministic testing with `AILANG_SEED`
  - Net: domain allowlist, localhost blocking, redirect validation
  - Records: row polymorphism with `AILANG_RECORDS_V2=1`
  - Error handling patterns
  - Capability composition

### Problem 4: Missing Developer Guides

**Current (v0.3.0)**:
- CHANGELOG.md and README.md updated
- No dedicated guides for new features

**Want (v0.4.0)**:
- `docs/guides/recursion.md` - Recursion patterns, stack overflow handling
- `docs/guides/records.md` - Record subsumption, row polymorphism, field access
- `docs/guides/effects.md` - Updated with Clock, Net, capability security
- `docs/guides/debugging.md` - NEW: Using --debug, --trace, reading stack traces

## Goals

### Primary Goals (Must Achieve)
1. **Global debug control** - `--debug` flag and `AILANG_DEBUG` env var working
2. **Fix quiet flag wiring** - `--quiet` flag actually suppresses progress messages (currently broken)
3. **Enhanced audit script** - Clock/Net detection, capability suggestions
4. **10+ micro examples** - Comprehensive edge case coverage
5. **Developer guides** - 4 new/updated guides in docs/guides/

### Stretch Goals
1. Better recursion diagnostics (suggest `--max-recursion-depth=N`)
2. JSON output for structured logging
3. Performance profiling flag (`--profile`)

### Non-Goals (Out of Scope for v0.4.0)
- IDE integration / language server
- Syntax highlighting packages
- Web-based playground
- Package manager

## Implementation Plan

### Phase 1: Global Debug & Quiet Control (1 day, ~200 LOC)

**Files to modify**:
- `cmd/ailang/main.go` - Add `--debug` flag, fix `--quiet` wiring, add `AILANG_DEBUG` support
- `internal/runtime/config.go` - Add `Debug bool` and `Quiet bool` fields to Config
- `internal/eval/eval_core.go` - Wrap debug statements in `if config.Debug`
- `internal/types/typechecker_core.go` - Wrap debug/progress in `if config.Debug/!config.Quiet`
- `internal/elaborate/elaborate.go` - Wrap debug statements
- `internal/runtime/execute.go` - Respect quiet mode for progress messages

**Implementation**:
```go
// cmd/ailang/main.go
var debugFlag = flag.Bool("debug", false, "Enable debug logging")
var quietFlag = flag.Bool("quiet", false, "Suppress progress messages")

func main() {
    flag.Parse()

    // Support both CLI flag and env var for debug
    debug := *debugFlag || os.Getenv("AILANG_DEBUG") != ""
    quiet := *quietFlag  // FIX: Actually use the flag value!

    config := &runtime.Config{
        Debug: debug,
        Quiet: quiet,
        // ... other fields
    }
}

// internal/runtime/execute.go
func Execute(config *Config) error {
    if !config.Quiet {
        fmt.Println("→ Type checking...")
    }
    // ... existing code

    if !config.Quiet {
        fmt.Println("→ Effect checking...")
    }
    // ... existing code
}

// internal/eval/eval_core.go
func (e *Evaluator) eval(node ast.Node) (Value, error) {
    if e.config.Debug {
        fmt.Printf("DEBUG: Evaluating %T: %v\n", node, node)
    }
    // ... existing code
}
```

**Acceptance Criteria**:
- ✅ `ailang run --debug file.ail` shows debug output
- ✅ `AILANG_DEBUG=1 ailang run file.ail` shows debug output
- ✅ `ailang run --quiet file.ail` suppresses ALL progress messages (only shows program output)
- ✅ `ailang run --debug --quiet file.ail` shows debug but not progress (debug takes precedence)
- ✅ Without flags, normal progress messages shown, no debug
- ✅ `ailang --help` documents both `--debug` and `--quiet` flags

### Phase 2: Enhanced Audit Script (0.5 days, ~100 LOC)

**File to modify**:
- `tools/audit-examples.sh` - Add Clock/Net detection

**Implementation**:
```bash
# tools/audit-examples.sh
detect_capabilities() {
    local file=$1
    local caps=""

    # Existing: IO, FS detection
    if grep -q "println\|print\|readLine" "$file"; then
        caps="$caps,IO"
    fi
    if grep -q "readFile\|writeFile\|exists" "$file"; then
        caps="$caps,FS"
    fi

    # NEW: Clock detection
    if grep -q "_clock_now\|_clock_sleep\|std/clock" "$file"; then
        caps="$caps,Clock"
    fi

    # NEW: Net detection
    if grep -q "_net_httpGet\|_net_httpPost\|std/net" "$file"; then
        caps="$caps,Net"
    fi

    echo "$caps" | sed 's/^,//'
}

# Suggest capabilities in error messages
if [ $exit_code -ne 0 ]; then
    detected_caps=$(detect_capabilities "$file")
    if [ -n "$detected_caps" ]; then
        echo "Hint: Try running with --caps $detected_caps"
    fi
fi
```

**Acceptance Criteria**:
- ✅ Clock usage detected: `_clock_now`, `_clock_sleep`, `import std/clock`
- ✅ Net usage detected: `_net_httpGet`, `_net_httpPost`, `import std/net`
- ✅ Error messages suggest required capabilities
- ✅ `make verify-examples` uses enhanced detection

### Phase 3: Micro Examples (0.5 days, ~150 LOC)

**New files to create** (`examples/`):
1. `micro_clock_deterministic.ail` - Deterministic testing with `AILANG_SEED`
2. `micro_net_allowlist.ail` - Domain allowlist with wildcard
3. `micro_net_localhost_blocked.ail` - Localhost blocking demo
4. `micro_net_redirect.ail` - Redirect validation
5. `micro_record_row_poly.ail` - Row polymorphism with `AILANG_RECORDS_V2=1`
6. `micro_recursion_depth.ail` - Stack overflow handling
7. `micro_error_handling.ail` - Error pattern examples
8. `micro_capability_composition.ail` - Multiple effects in one function

**Example**:
```ailang
-- examples/micro_clock_deterministic.ail
-- Demonstrates deterministic time with AILANG_SEED
module examples/micro_clock_deterministic

import std/clock (now)
import std/io (println)

export func main() -> () ! {IO, Clock} {
  let t1 = now();
  let t2 = now();
  println("Time difference (should be 0 in deterministic mode):");
  println(show(t2 - t1))
}

-- Usage:
-- AILANG_SEED=42 ailang run --caps IO,Clock examples/micro_clock_deterministic.ail
-- Output: 0 (deterministic)
--
-- ailang run --caps IO,Clock examples/micro_clock_deterministic.ail
-- Output: 1-5 (real time, non-deterministic)
```

**Acceptance Criteria**:
- ✅ 8 new micro examples created
- ✅ All examples have comments explaining usage
- ✅ All examples pass verification
- ✅ Examples added to `examples/STATUS.md`

### Phase 4: Developer Guides (1 day, ~400 words each = ~1,600 words total)

**New files to create** (`docs/guides/`):

1. **`recursion.md`** (~400 words)
   - Self-recursion patterns (factorial, fibonacci)
   - Mutual recursion (isEven/isOdd)
   - Stack overflow handling
   - Tail call optimization (future roadmap)

2. **`records.md`** (~400 words)
   - Record literals and field access
   - Record subsumption (functions accepting supersets)
   - Row polymorphism with `AILANG_RECORDS_V2=1`
   - Nested records
   - Common patterns (user profiles, API responses)

3. **`effects.md` UPDATE** (~400 words new content)
   - Clock effect: now(), sleep(), deterministic mode
   - Net effect: httpGet(), httpPost(), security model
   - Capability security best practices
   - Effect composition patterns

4. **`debugging.md` NEW** (~400 words)
   - Using `--debug` flag for verbose output
   - Using `--trace` for execution tracing
   - Reading type errors and stack traces
   - Common error patterns and solutions
   - Environment variables (`AILANG_DEBUG`, `AILANG_SEED`, `AILANG_RECORDS_V2`)

**Acceptance Criteria**:
- ✅ 3 new guides + 1 updated guide created
- ✅ All guides follow existing docs/guides/ style
- ✅ Code examples in all guides tested and working
- ✅ Guides linked from README.md

## Testing Strategy

### Unit Tests (~50 LOC)
- `cmd/ailang/main_test.go` - Test `--debug` flag parsing
- `internal/runtime/config_test.go` - Test Config.Debug field

### Integration Tests
- Run all examples with `--debug` flag, verify debug output
- Run audit script on Clock/Net examples, verify detection
- Test all new micro examples pass verification

### Documentation Tests
- All code examples in guides must be runnable
- Verify all guides render correctly in markdown

## Risk Mitigation

| Risk | Severity | Mitigation |
|------|----------|------------|
| **Debug flag breaks existing behavior** | Low | Make debug opt-in (off by default) |
| **Audit script false positives** | Low | Test on all existing examples first |
| **Micro examples fail verification** | Medium | Test each example before committing |
| **Guides out of date quickly** | Medium | Use automated example testing |

## Success Metrics

| Metric | Target |
|--------|--------|
| **Debug control** | `--debug` and `AILANG_DEBUG` working |
| **Quiet flag fixed** | `--quiet` actually suppresses progress messages |
| **Audit detection** | Clock + Net detected correctly |
| **New examples** | ≥8 micro examples added |
| **Documentation** | 4 guides created/updated |
| **Test coverage** | All new code tested |

## Out of Scope (Deferred to Later)

- JSON structured logging (v0.5.0)
- Performance profiling (v0.5.0)
- LSP / IDE integration (v1.0+)
- Better recursion diagnostics (`--max-recursion-depth`) (v0.5.0)

## Definition of Done

- ✅ `--debug` flag implemented and tested
- ✅ `AILANG_DEBUG` env var supported
- ✅ `--quiet` flag wiring fixed (currently broken - value discarded)
- ✅ Audit script detects Clock/Net capabilities
- ✅ 8+ new micro examples created and passing
- ✅ 4 developer guides created/updated
- ✅ All tests passing
- ✅ Documentation updated (README, CHANGELOG)

## File Changes (DRI)

### CLI (~100 LOC)
- `cmd/ailang/main.go` - Add `--debug` flag, fix `--quiet` wiring, add `AILANG_DEBUG` support

### Runtime (~100 LOC)
- `internal/runtime/config.go` - Add `Debug bool` and `Quiet bool` fields
- `internal/runtime/execute.go` - Respect quiet mode for progress messages

### Evaluator (~50 LOC)
- `internal/eval/eval_core.go` - Wrap debug statements in `if config.Debug`

### Type Checker (~50 LOC)
- `internal/types/typechecker_core.go` - Wrap debug/progress in `if config.Debug/!config.Quiet`

### Tools (~100 LOC)
- `tools/audit-examples.sh` - Add Clock/Net detection

### Examples (~150 LOC)
- 8 new micro examples in `examples/`

### Documentation (~1,600 words)
- `docs/guides/recursion.md` - NEW
- `docs/guides/records.md` - NEW
- `docs/guides/debugging.md` - NEW
- `docs/guides/effects.md` - UPDATED
- `README.md` - Link to new guides
- `CHANGELOG.md` - Document M-UX2 completion

### Tests (~50 LOC)
- `cmd/ailang/main_test.go` - Test `--debug` and `--quiet` flags
- `internal/runtime/config_test.go` - Test Config.Debug and Config.Quiet

**Total Estimate**: ~450 LOC + 1,600 words documentation

## Priority Rationale

**Why P2 (NICE TO HAVE)?**
- Core functionality already works (P0 items shipped in v0.3.0)
- Debug control useful but not blocking
- Audit enhancement improves UX but not critical
- Examples and guides improve onboarding but language is usable without them

**When to ship?**
- If v0.4.0 timeline is comfortable: Ship M-UX2 alongside Net enhancements
- If v0.4.0 timeline is tight: Defer M-UX2 to v0.4.1 or v0.5.0

## Comparison with v0.3.0 Plan

**Original v0.3.0 Scope (Deferred)**:
- Debug flag - NOT shipped
- Audit script enhancement - NOT shipped
- Micro examples - PARTIALLY shipped (5 examples instead of 10+)
- Developer guides - NOT shipped

**Why deferred?**
- v0.3.0 shipped 13 days early with all P0 + P1 items
- M-UX2 not critical for release
- Better to ship v0.3.0 without M-UX2 than delay release

**What was shipped in v0.3.0?**
- 5 micro examples (blocks, clock, net, records, option)
- CHANGELOG.md and README.md updates
- Minimal but sufficient documentation

## Next Steps

1. **v0.4.0 Planning**: Decide if M-UX2 fits timeline
2. **If shipped**: Follow 4-phase implementation plan above
3. **If deferred**: Move to v0.4.1 or v0.5.0 backlog

---

**Document Version**: v1.0
**Created**: October 5, 2025
**Author**: AILANG Development Team
**Deferred From**: v0.3.0 Implementation Plan
**Target Release**: v0.4.0 (or later if timeline tight)
