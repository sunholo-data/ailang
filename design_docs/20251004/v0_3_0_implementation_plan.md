# v0.3.0 Implementation Plan

**Version**: v0.3.0
**Status**: DRAFT (Planning)
**Target Release**: October 17–21, 2025 (2 weeks)
**Author**: AILANG Development Team
**Created**: October 4, 2025

---

## Executive Summary

**What v0.2.0 Delivered**:
- Module execution runtime with cross-module imports
- Effect system (IO, FS) with capability-based security
- Pattern matching with guards and exhaustiveness checking
- Auto-entry fallback and user experience improvements
- **42/53 examples passing (79.2%)** - exceeded target of 35

**What v0.3.0 Will Deliver**:
- **Recursion support** - Enable factorial, fibonacci, quicksort patterns
- **Records & row polymorphism** - Proper record type unification
- **Extended effects** - Add Clock (time) and Net (networking) capabilities
- **Type system fixes** - Modulo operator (%) and Integral type class
- **Target**: **≥50 passing examples (83%)**, real-world program structures

**Focus**: Expressive power - enable common programming patterns (recursion, data modeling) while extending the effect system for practical applications.

---

## Status Snapshot

### Current State (v0.2.0)
- ✅ Module execution working
- ✅ Effect system (IO, FS) functional
- ✅ Pattern matching with exhaustiveness
- ✅ ADT constructors across modules
- ✅ Type classes with dictionary passing
- ✅ Auto-entry and capability detection
- ❌ **Recursive functions fail** (HIGH blocker)
- ⚠️ **Records partially working** (unification issues)
- ❌ **Modulo operator broken** (type class issue)
- ⚠️ **Only IO/FS effects** (need Clock, Net)

### Target State (v0.3.0)
- ✅ Recursive functions compile and run
- ✅ Record types unify with row polymorphism
- ✅ Clock and Net effects with capability security
- ✅ Modulo operator functional via Integral type class
- ✅ 50+ examples passing (83%+ of suite)

---

## Milestones

### M-R4: Recursion Support (HIGH PRIORITY)
**Effort**: ~600 LOC, 3-4 days
**Impact**: Unlocks 3-5 examples + enables common patterns

**Problem**:
Functions cannot call themselves recursively. Example fails:
```ailang
export func factorial(n: int) -> int {
  if n <= 1 then 1 else n * factorial(n - 1)  -- ❌ "undefined variable: factorial"
}
```

**Root Cause**:
- Function bindings not available in their own scope during evaluation
- LetRec handling incomplete in runtime evaluator
- Closure environment doesn't include self-reference

**Tasks**:
1. **Add LetRec support** (`internal/runtime/eval.go`, ~200 LOC)
   - Extend evaluateModule() to handle self-referential bindings
   - Add function to own environment before evaluating body
   - Support mutual recursion between functions

2. **Fix closure self-reference** (`internal/eval/value.go`, ~100 LOC)
   - ClosureValue needs backpointer to own name
   - Update closure creation to include self in environment

3. **Add recursive resolver** (`internal/runtime/resolver.go`, ~150 LOC)
   - Handle self-references in GlobalResolver
   - Distinguish between forward reference (error) and recursion (ok)

4. **Tests and examples** (~150 LOC)
   - Unit tests for factorial, fibonacci, quicksort
   - Add `examples/recursive_factorial.ail`
   - Add `examples/recursive_fib.ail`
   - Add `examples/recursive_quicksort.ail`

**Acceptance Criteria**:
- ✅ `factorial(5)` returns 120
- ✅ `fib(10)` returns 55
- ✅ `quicksort([3,1,4,1,5,9])` works
- ✅ Mutual recursion supported (isEven/isOdd pattern)
- ✅ No stack overflow on reasonable inputs (n < 1000)

**Examples Unblocked**: factorial, fibonacci, quicksort, tree traversal

**Files Modified**:
- `internal/runtime/eval.go` - LetRec handling
- `internal/eval/value.go` - Closure self-reference
- `internal/runtime/resolver.go` - Self-reference resolution
- `examples/recursive_*.ail` - New examples

---

### M-R5: Records & Row Polymorphism (HIGH PRIORITY)
**Effort**: ~500 LOC, 3-4 days
**Impact**: Unlocks 2-3 examples + enables data modeling

**Problem**:
Record examples fail with type errors:
```ailang
let person = {name: "Alice", age: 30} in
person.age  -- ❌ "unhandled type" or "field count mismatch"
```

**Root Cause**:
- TRecord unification incomplete (v0.2.0 added basic support)
- Field access doesn't properly unify record types
- Row variables not implemented for polymorphism
- No support for record extension/restriction

**Tasks**:
1. **Complete TRecord unification** (`internal/types/unification.go`, ~200 LOC)
   - Improve field-by-field unification
   - Add subsumption rules (more fields ⊆ fewer fields)
   - Handle missing fields gracefully with errors

2. **Add row variables** (`internal/types/types.go`, ~150 LOC)
   - Extend TRecord with row variable (ρ)
   - Support polymorphic record types: `{name: String | ρ}`
   - Implement row unification

3. **Fix record field access** (`internal/types/typechecker_core.go`, ~100 LOC)
   - RecordAccess should unify with record containing that field
   - Generate fresh row variable for polymorphism
   - Better error messages for missing fields

4. **Runtime support** (`internal/eval/eval_core.go`, ~50 LOC)
   - Ensure RecordValue handles all field operations
   - RecordAccess evaluation with proper error handling

**Acceptance Criteria**:
- ✅ `{name: "Alice", age: 30}.name` returns "Alice"
- ✅ `{a: 1, b: 2, c: 3}.b` returns 2
- ✅ Record with extra fields unifies with subset requirement
- ✅ Missing field gives clear error: "field 'x' not found in record"
- ✅ Nested records work: `{person: {name: "Bob"}}.person.name`

**Examples Unblocked**: records.ail, list_patterns.ail, lambda_expressions.ail (partial)

**Files Modified**:
- `internal/types/unification.go` - Complete TRecord unification
- `internal/types/types.go` - Add row variables
- `internal/types/typechecker_core.go` - Fix field access
- `internal/eval/eval_core.go` - Runtime support

---

### M-R6: Extended Effects (Clock & Net) (MEDIUM PRIORITY)
**Effort**: ~700 LOC, 3-4 days
**Impact**: Enables time-dependent and networked programs

**Problem**:
Only IO and FS effects exist. No support for time or networking.

**Use Cases**:
- Clock: timestamps, delays, timeouts, rate limiting
- Net: HTTP requests, API calls, webhooks

**Tasks**:
1. **Implement Clock effect** (`stdlib/std/clock.ail`, ~150 LOC)
   - `now() -> int` - Unix timestamp in milliseconds
   - `sleep(ms: int) -> ()` - Sleep for duration
   - `timeout(ms: int, f: () -> a) -> Option[a]` - Run with timeout

2. **Implement Net effect** (`stdlib/std/net.ail`, ~200 LOC)
   - `httpGet(url: string) -> string` - GET request
   - `httpPost(url: string, body: string) -> string` - POST request
   - `httpRequest(method: string, url: string, body: string) -> Response`

3. **Add runtime builtins** (`internal/runtime/builtins.go`, ~150 LOC)
   - `_clock_now()` - System time
   - `_clock_sleep(ms)` - Thread sleep
   - `_net_http_get(url)` - HTTP GET via net/http
   - `_net_http_post(url, body)` - HTTP POST

4. **Capability enforcement** (`internal/effects/context.go`, ~100 LOC)
   - Extend EffContext with Clock and Net capabilities
   - Check capabilities before allowing operations
   - Sandbox Net to http/https only (no file://, no localhost by default)

5. **CLI integration** (`cmd/ailang/main.go`, ~50 LOC)
   - Add Clock and Net to valid capability names
   - Update help text and error messages

6. **Examples** (~50 LOC)
   - `examples/micro_clock_now.ail` - Print current timestamp
   - `examples/micro_clock_sleep.ail` - Countdown timer
   - `examples/micro_net_fetch.ail` - Fetch JSON from API

**Security Considerations**:
- Clock: Prevent time manipulation (use monotonic clock)
- Net:
  - Deny by default (require `--caps Net`)
  - Restrict to http/https protocols
  - Optional: sandbox to specific domains with `--net-allow=domain.com`
  - Timeout all requests (default 30s)
  - No localhost access by default

**Acceptance Criteria**:
- ✅ `now()` returns Unix timestamp
- ✅ `sleep(1000)` pauses for 1 second
- ✅ `httpGet("https://api.example.com")` works with `--caps Net`
- ✅ Network operations fail without `--caps Net`
- ✅ file:// URLs are rejected
- ✅ Localhost is blocked by default

**Examples Added**: micro_clock_now, micro_clock_sleep, micro_net_fetch

**Files Modified**:
- `stdlib/std/clock.ail` - Clock effect API
- `stdlib/std/net.ail` - Net effect API
- `internal/runtime/builtins.go` - Clock/Net builtins
- `internal/effects/context.go` - Capability enforcement
- `cmd/ailang/main.go` - CLI integration
- `examples/micro_clock_*.ail` - Clock examples
- `examples/micro_net_*.ail` - Net examples

---

### M-R7: Modulo Operator Fix (MEDIUM PRIORITY)
**Effort**: ~200 LOC, 1-2 days
**Impact**: Fixes broken arithmetic operator

**Problem**:
The `%` operator fails with ambiguous type class constraints:
```ailang
export func main() -> int { 5 % 3 }
-- ❌ Error: ambiguous type variable α with classes [Num, Ord]
```

**Root Cause**:
- `%` requires both Num (for arithmetic) and Ord (for division semantics)
- Type system can't default constraints with multiple classes
- Num alone is insufficient (% undefined for Float)

**Solution**: Introduce Integral type class

**Tasks**:
1. **Define Integral type class** (`internal/types/typeclass.go`, ~50 LOC)
   ```ailang
   class Num a => Integral a {
     div: a -> a -> a,
     mod: a -> a -> a
   }
   ```

2. **Add Integral instances** (`stdlib/std/prelude.ail`, ~50 LOC)
   - `instance Integral[Int]`
   - No instance for Float (mod is integer-only)

3. **Update operator elaboration** (`internal/elaborate/elaborate.go`, ~50 LOC)
   - Map `%` to `mod` method call
   - Require Integral constraint instead of Num

4. **Update defaulting** (`internal/types/typechecker_core.go`, ~30 LOC)
   - Default Integral-constrained variables to Int
   - Better error message if Float used with %

5. **Tests** (~20 LOC)
   - `5 % 3 == 2`
   - `-5 % 3 == -2` (implementation-defined)
   - `5.0 % 3.0` → clear error

**Acceptance Criteria**:
- ✅ `5 % 3` returns 2
- ✅ `10 % 4` returns 2
- ✅ `-7 % 3` works (implementation-defined semantics)
- ✅ `5.0 % 3.0` gives error: "mod not defined for Float"
- ✅ Type inference defaults % to Int

**Examples Unblocked**: Any arithmetic using modulo

**Files Modified**:
- `internal/types/typeclass.go` - Integral type class
- `stdlib/std/prelude.ail` - Integral instances
- `internal/elaborate/elaborate.go` - Operator elaboration
- `internal/types/typechecker_core.go` - Defaulting

---

### M-UX2: User Experience Enhancements (LOW PRIORITY)
**Effort**: ~300 LOC, 2-3 days
**Impact**: Polish and convenience

**Tasks**:
1. **Better recursion errors** (~50 LOC)
   - When recursion fails, suggest: "Use --debug to enable recursive tracing"
   - Add `--max-recursion-depth=N` flag (default 1000)

2. **Audit script improvements** (`tools/audit-examples.sh`, ~50 LOC)
   - Auto-detect Clock effect (scan for `now()`, `sleep`)
   - Auto-detect Net effect (scan for `http`)
   - Add Clock,Net to capability string

3. **More micro examples** (~150 LOC)
   - `examples/micro_recursive_sum.ail` - Sum list via recursion
   - `examples/micro_record_person.ail` - Person record with fields
   - `examples/micro_clock_timer.ail` - Countdown timer
   - `examples/micro_net_json.ail` - Fetch and parse JSON

4. **Documentation updates** (~50 LOC)
   - Add recursion guide: `docs/guides/recursion.md`
   - Add records guide: `docs/guides/records.md`
   - Add effects guide update: Clock and Net sections

**Acceptance Criteria**:
- ✅ Recursion errors are helpful
- ✅ Audit script handles Clock/Net
- ✅ 4+ new micro examples
- ✅ Documentation covers new features

**Examples Added**: micro_recursive_sum, micro_record_person, micro_clock_timer, micro_net_json

**Files Modified**:
- `internal/runtime/eval.go` - Better recursion errors
- `tools/audit-examples.sh` - Clock/Net detection
- `examples/micro_*.ail` - New examples
- `docs/guides/*.md` - New guides

---

## Success Metrics

| Metric | v0.2.0 | v0.3.0 Target | v0.3.0 Goal |
|--------|--------|---------------|-------------|
| **Passing Examples** | 42/53 (79%) | ≥50 | 50/60 (83%) |
| **Test Coverage** | 27.3% | ≥30% | 35% |
| **Recursion** | ❌ Broken | ✅ Working | factorial, fib, quicksort |
| **Records** | ⚠️ Partial | ✅ Working | Unification + row polymorphism |
| **Modulo (%)** | ❌ Broken | ✅ Working | Via Integral type class |
| **Effects** | IO, FS | IO, FS, Clock, Net | 4 effects total |
| **LOC Delivered** | ~4,324 | ~2,000 | ~6,324 total |

**Key Performance Indicators**:
- Pass rate: 79% → 83% (+4%)
- New capabilities: 2 → 4 effects (+100%)
- Unblocked patterns: Recursion, records, networking

---

## Implementation Plan

### Week 1: Core Features (Days 1-7)

**Days 1-3: M-R4 Recursion**
- Day 1: Add LetRec support to runtime evaluator
- Day 2: Fix closure self-reference, add resolver logic
- Day 3: Write tests (factorial, fib, quicksort), validate examples

**Days 3-5: M-R5 Records**
- Day 4: Complete TRecord unification, add row variables
- Day 5: Fix field access type checking, runtime support
- Day 6: Test with records.ail, list_patterns.ail, validate

**Days 5-7: M-R6 Clock/Net Effects**
- Day 7: Implement stdlib/std/clock.ail and builtins
- Day 8: Implement stdlib/std/net.ail and builtins
- Day 9: Add capability enforcement, security sandbox

### Week 2: Polish & Release (Days 8-14)

**Days 8-9: M-R7 Modulo + M-UX2**
- Day 10: Add Integral type class, fix % operator
- Day 11: Better error messages, audit script improvements

**Days 10-11: Examples & Testing**
- Day 12: Create 4-6 new micro examples (recursive, record, clock, net)
- Day 13: Full audit run, fix failing examples

**Days 12-13: Documentation & Release**
- Day 14: Update README, CHANGELOG, implementation status
- Day 15: Create release notes, build binaries, publish v0.3.0

---

## Risks & Mitigations

| Risk | Severity | Impact | Mitigation |
|------|----------|--------|------------|
| **Recursive evaluation complexity** | HIGH | May need trampolining or stack management | Start with simple tail recursion, add regression suite, defer mutual recursion if needed |
| **Row polymorphism too complex** | MEDIUM | Record inference may be undecidable | Limit to closed records for v0.3.0, defer open/extensible records to v0.4.0 |
| **Network security** | HIGH | SSRF, data exfiltration risks | Deny by default, restrict to http/https, timeout all requests, optional domain whitelist |
| **Modulo operator regressions** | LOW | May break existing arithmetic | Isolate to Integral type class, keep Num untouched, extensive tests |
| **Time pressure (2 weeks)** | MEDIUM | Features may slip | Recursion is P0 (must have), Records/Clock/Net are P1 (nice to have), modulo is P2 |

**Mitigation Strategy**:
1. Recursion is **mandatory** - block release if broken
2. Records are **high priority** - defer row polymorphism if too complex
3. Clock/Net are **stretch** - ship without if security concerns arise
4. Modulo fix is **optional** - acceptable to defer to v0.3.1

---

## Testing Strategy

### Unit Tests
- Recursion: factorial(5), fib(10), quicksort([3,1,4,1,5,9])
- Records: field access, nested records, missing fields
- Clock: now() > 0, sleep(100) takes ~100ms
- Net: httpGet mocked, error handling
- Modulo: 5 % 3 == 2, type errors for Float

### Integration Tests
- Run all 53 existing examples, expect 50+ to pass
- New examples: micro_recursive_*, micro_record_*, micro_clock_*, micro_net_*
- Cross-module imports with new effects

### Security Tests
- Clock: cannot manipulate time
- Net: file:// blocked, localhost blocked, timeout enforced
- Capabilities: operations fail without --caps

### Regression Tests
- v0.2.0 examples must still pass
- No regressions in type classes, pattern matching, effects

---

## Definition of Done

### Code Complete
- ✅ All milestones (M-R4, M-R5, M-R6, M-R7, M-UX2) implemented
- ✅ Unit tests passing for all new features
- ✅ Integration tests passing (≥50/60 examples)
- ✅ No regressions in v0.2.0 functionality

### Documentation Complete
- ✅ README.md updated with v0.3.0 features and examples count
- ✅ CHANGELOG.md has v0.3.0 entry with full details
- ✅ docs/guides/recursion.md created
- ✅ docs/guides/records.md created
- ✅ docs/guides/effects.md updated (Clock, Net sections)

### Release Ready
- ✅ Version updated in all files (README, CHANGELOG)
- ✅ Git tag created: `v0.3.0`
- ✅ CI/CD passing (tests, linting, builds)
- ✅ Release binaries built for all platforms (macOS, Linux, Windows)
- ✅ Release published on GitHub

---

## Files Modified Summary

### Core Runtime (~800 LOC)
- `internal/runtime/eval.go` - LetRec, recursion support
- `internal/eval/value.go` - Closure self-reference
- `internal/runtime/resolver.go` - Self-reference resolution
- `internal/runtime/builtins.go` - Clock/Net builtins

### Type System (~500 LOC)
- `internal/types/unification.go` - TRecord completion, row variables
- `internal/types/types.go` - Row variable types
- `internal/types/typechecker_core.go` - Field access, Integral defaulting
- `internal/types/typeclass.go` - Integral type class

### Effects (~450 LOC)
- `stdlib/std/clock.ail` - Clock effect API
- `stdlib/std/net.ail` - Net effect API
- `internal/effects/context.go` - Clock/Net capabilities

### Elaboration (~100 LOC)
- `internal/elaborate/elaborate.go` - Modulo operator

### Examples & Tools (~450 LOC)
- `examples/recursive_*.ail` - Recursion examples (3 files)
- `examples/micro_clock_*.ail` - Clock examples (2 files)
- `examples/micro_net_*.ail` - Net examples (2 files)
- `examples/micro_record_*.ail` - Record examples (1 file)
- `tools/audit-examples.sh` - Clock/Net detection

### Documentation (~200 LOC)
- `docs/guides/recursion.md` - NEW
- `docs/guides/records.md` - NEW
- `docs/guides/effects.md` - UPDATED
- `README.md` - UPDATED
- `CHANGELOG.md` - UPDATED

**Total**: ~2,500 LOC (conservative estimate)

---

## Deferred to Later Versions

### v0.3.1 (Bugfixes)
- Performance optimizations for recursion (tail call elimination)
- Open/extensible records with row polymorphism
- More effects: Random, Env
- Network proxy support

### v0.4.0 (Safety Focus)
- Semantic annotations (`@intent`, `@requires`, `@ensures`)
- Resource budgets (computation limits)
- Info-flow labels (PII/Secret tracking)
- Linear/affine types (resource lifecycle)
- Policy DSL (org-level constraints)

### v0.5.0 (Concurrency)
- Structured concurrency (nurseries)
- Session types (protocol verification)
- Async effects (Promises/Futures)
- Channel-based communication

---

## Comparison: v0.2.0 vs v0.3.0 vs v1.0

| Feature | v0.2.0 | v0.3.0 | v1.0 Vision |
|---------|--------|--------|-------------|
| **Module Execution** | ✅ Basic | ✅ + Recursion | + Mutual imports |
| **Effects** | IO, FS | + Clock, Net | + Random, Env, DB |
| **Type System** | HM + classes | + Records, Integral | + Linear, Session |
| **Pattern Matching** | Guards, exhaustive | Same | + View patterns |
| **Recursion** | ❌ Broken | ✅ Working | + Tail call optimization |
| **Records** | ⚠️ Partial | ✅ Row polymorphism | + Extensible |
| **Operators** | Most work | + Modulo | All working |
| **Examples Passing** | 42/53 (79%) | 50/60 (83%) | 55/60 (92%) |
| **Security** | Basic caps | + Net sandbox | + Budgets, info-flow |

---

## Next Steps After v0.3.0

1. **Immediate (v0.3.1)**: Bugfixes, performance tuning, missing operator edge cases
2. **Short-term (v0.4.0)**: Safety features (budgets, annotations, linear types)
3. **Medium-term (v0.5.0)**: Concurrency (async, channels, session types)
4. **Long-term (v1.0)**: Feature-complete, production-ready

**Philosophy**: Each release adds one major capability cluster while maintaining stability.

---

## Appendix: Example Programs (v0.3.0)

### Recursion Example
```ailang
-- examples/recursive_factorial.ail
module examples/recursive_factorial

export func factorial(n: int) -> int {
  if n <= 1 then 1 else n * factorial(n - 1)
}

export func main() -> int {
  factorial(5)  -- Returns: 120
}
```

### Record Example
```ailang
-- examples/micro_record_person.ail
module examples/micro_record_person

type Person = {name: string, age: int}

export func main() -> string {
  let alice = {name: "Alice", age: 30} in
  alice.name  -- Returns: "Alice"
}
```

### Clock Example
```ailang
-- examples/micro_clock_now.ail
module examples/micro_clock_now
import std/clock (now)

export func main() -> int ! {Clock} {
  now()  -- Returns: Unix timestamp in ms
}
```

### Net Example
```ailang
-- examples/micro_net_fetch.ail
module examples/micro_net_fetch
import std/net (httpGet)
import std/io (println)

export func main() -> () ! {Net, IO} {
  let response = httpGet("https://api.github.com") in
  println(response)
}
```

---

**Document Version**: v1.0 - DRAFT
**Created**: October 4, 2025
**Last Updated**: October 4, 2025
**Author**: AILANG Development Team

**Changes from v0.0**:
- Initial draft based on v0.2.0 completion and v0.3.0 requirements
- Detailed milestones for recursion, records, Clock/Net effects, modulo fix
- Comprehensive risk assessment and mitigation strategies
- 2-week timeline with daily breakdown

---

## Conclusion

v0.3.0 will transform AILANG from a demonstration language to a practical tool for real programs by enabling:

1. **Recursion** - Unlocking fundamental programming patterns
2. **Records** - Enabling proper data modeling
3. **Clock/Net** - Connecting to the real world
4. **Type fixes** - Removing known blockers

**Ship v0.3.0** to prove AILANG can handle real-world program structures, then iterate toward v1.0 production readiness.

---

*Drafted by Claude Sonnet 4.5 with input from AILANG Development Team*
*October 4, 2025*
