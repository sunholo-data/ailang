# v0.3.0 Implementation Plan

**Version**: v0.3.0
**Status**: DRAFT (Planning)
**Target Release**: October 17–21, 2025 (2 weeks)
**Author**: AILANG Development Team
**Created**: October 4, 2025
**Last Updated**: October 3, 2025

---

## Background: v0.2.0 Post-Release Fixes (October 3, 2025)

After releasing v0.2.0-rc1, CI failures were discovered and fixed:

### Issues Fixed
1. **Non-module file execution** (v0.1.0 compatibility)
   - Files without `module` keyword were failing to execute
   - **Root cause**: All files went through module execution path, but non-module files have no exports
   - **Fix**: Detect module vs non-module files; use ModeEval for non-module files (proper resolvers for ADT constructors)
   - **Implementation**: Added module keyword detection in `cmd/ailang/main.go:248-266`

2. **Test lowering failures**
   - Non-module test files (`tests/binops_int.ail`) weren't printing results
   - **Root cause**: ModeCheck doesn't evaluate; results weren't printed
   - **Fix**: Non-module files use ModeEval, results printed from pipeline

3. **Flag ordering** (Go `flag` package behavior)
   - Makefile had flags BEFORE subcommand: `ailang --caps IO run file.ail` (broken)
   - **Fix**: Flags must come AFTER subcommand: `ailang run --caps IO file.ail`
   - **Updated**: `Makefile:189-196` and `scripts/verify_examples.go:110-119`

4. **Example verification script**
   - Used `go run cmd/ailang/main.go` (only compiles main.go, missing runEval)
   - **Fix**: Changed to `go run ./cmd/ailang` (compiles full package)

### Results
- **Before**: 0/53 examples passing (100% failure)
- **After**: 36/53 examples passing (68% success)
- **CI**: All checks now pass ✅
- **Commit**: `57bb6de` - "Fix CI failures: Non-module execution and flag ordering"

### Lessons Learned
- **Module detection**: Checking for `module` keyword determines execution path
- **Flag ordering**: Go's `flag` package stops parsing after first non-flag argument
- **Build commands**: Use `go run ./package` not `go run file.go` for multi-file packages

---

## Executive Summary

### What v0.2.0 Delivered
- Module execution runtime (cross-module imports, entrypoint execution, function calls)
- Effect system (IO, FS) with capability-based security
- Pattern matching w/ guards + exhaustiveness (decision trees available)
- Auto-entry fallback, capability auto-detection
- **36/53 passing examples (68%)** — post-CI-fix baseline

### What v0.3.0 Will Deliver
- **Recursion**: real LetRec and self/mutual recursion (factorial/fib/quicksort)
- **Records & row polymorphism**: usable record modeling + field access/type inference
- **Extended effects**: Clock (now/sleep) and Net (httpGet/httpPost) behind capabilities
- **Type system fix**: modulo % via a new Integral type class + sane defaulting
- **Target**: **≥50 passing examples (≥83%)**, covering common real-world program structures

**Theme**: Expressive power without compromising safety.

---

## Status Snapshot

### Current (v0.2.0)
- ✅ Modules, IO/FS effects, pattern guards/exhaustiveness, ADT constructors, type classes, auto-entry/caps
- ❌ **Recursion fails** (LetRec/closure env)
- ⚠️ **Records partial** (unification/rows)
- ❌ **Modulo broken** (ambiguous constraints)
- ⚠️ **Only IO/FS** (need Clock/Net)

### Target (v0.3.0)
- ✅ Recursion works (self + mutual)
- ✅ Records unify (row-polymorphic access)
- ✅ Clock/Net effects (caps-enforced, sandboxed)
- ✅ % fixed via Integral
- ✅ ≥50 examples passing

---

## Milestones

### M-R4: Recursion Support (P0)

**Effort**: ~600 LOC | **Owner**: runtime
**Why**: Fundamental building block; blocks several examples.

#### Root Cause (v0.2.0)
- LetRec not evaluating into self-referential closures.
- Environments bound after function creation; no self backpointer.

#### Tasks
1. **LetRec in runtime** (`internal/runtime/eval.go`)
   - Pre-bind names to thunked closures; fill bodies afterward.
   - Support mutual recursion (isEven/isOdd).

2. **Closure self-ref** (`internal/eval/value.go`)
   - Add self-name backpointer or indirection cell.

3. **Resolver tweaks** (`internal/runtime/resolver.go`)
   - Distinguish legal recursion vs invalid forward refs.

4. **Tests & examples**
   - `examples/recursive_factorial.ail`, `recursive_fib.ail`, `recursive_quicksort.ail`
   - Mutual recursion unit tests.

#### Acceptance Criteria
- ✅ `factorial(5)` returns 120
- ✅ `fib(10)` returns 55
- ✅ `quicksort([3,1,4,1,5,9])` works
- ✅ Mutual recursion passes
- ✅ No stack overflow for n<1000; friendly error if exceeded

**Examples Unblocked**: factorial, fibonacci, quicksort, tree traversal

---

### M-R5: Records & Row Polymorphism (P0)

**Effort**: ~500 LOC | **Owner**: types/eval
**Why**: Data modeling & ergonomics; unblocks current broken examples.

#### Tasks
1. **Complete TRecord unification** (`internal/types/unification.go`)
   - Field-by-field unification; subset/subsumption rules
   - Diagnostics for missing/mismatched fields

2. **Row variables** (`internal/types/types.go`)
   - Represent `{k:v | ρ}`; unify rows; scoped generalization

3. **Field access typing** (`internal/types/typechecker_core.go`)
   - Generate fresh row var; unify required field; better errors

4. **Runtime record ops** (`internal/eval/eval_core.go`)
   - Field access on RecordValue with clear failures

#### Acceptance Criteria
- ✅ `{name:"Alice",age:30}.name == "Alice"`
- ✅ Subset unification works; nested records ok
- ✅ Missing field → "field 'x' not found in record" (pointing span)

**Examples Unblocked**: records.ail, list_patterns.ail, lambda_expressions.ail (partial)

---

### M-R6: Extended Effects — Clock & Net (P1)

**Effort**: ~700 LOC | **Owner**: effects/runtime/CLI
**Why**: Practicality; enables time-based logic and HTTP interactions.

#### API
- **Clock** (`std/clock`): `now() -> int`, `sleep(ms:int) -> ()`
- **Net** (`std/net`): `httpGet(url) -> string`, `httpPost(url, body) -> string`

#### Runtime
- **Builtins**: `_clock_now`, `_clock_sleep`, `_net_http_get`, `_net_http_post`
- **Caps**: `--caps Clock,Net`
- **Security**:
  - Deny by default
  - Restrict Net to http(s); block file://; block localhost by default
  - Global timeout (default 30s); optional `--net-allow=domain.com[,..]`

#### Examples
- `micro_clock_now.ail`, `micro_clock_sleep.ail`, `micro_net_fetch.ail`

#### Acceptance Criteria
- ✅ Works with caps; fails without
- ✅ Blocks non-http(s) and localhost; respects timeouts

**Files Modified**:
- `stdlib/std/clock.ail`, `stdlib/std/net.ail`
- `internal/runtime/builtins.go` — Clock/Net builtins
- `internal/effects/context.go` — Capability enforcement
- `cmd/ailang/main.go` — CLI integration

---

### M-R7: Type System Fixes — Integral & Float Comparison (P1)

**Effort**: ~300 LOC | **Owner**: types/elab/stdlib/eval

#### Root Causes (documented)

**Issue 1: Modulo operator (`%`)**
- `%` had ambiguous constraints (Num ∧ Ord) and defaulting couldn't pick a concrete type.
- Example: `5 % 3` failed with "ambiguous type variable α with classes [Num, Ord]"

**Issue 2: Float comparison (`==` with float)**
- Float equality comparison incorrectly tries to use `eq_Int` instead of `eq_Float`
- Example: `b == 0.0` where `b: float` fails with "builtin eq_Int expects Int arguments"
- **Discovered in**: adt_option benchmark eval (October 3, 2025)
- **AI generated correct code**, but AILANG runtime failed on `if b == 0.0` comparison

#### Solution

**For modulo:**
- Add **Integral type class** (inherits Num; methods: div, mod)
- Map `%` to `mod`
- Default Integral to Int unless annotated

**For float comparison:**
- Fix type class instance resolution in `==` operator elaboration
- Ensure `eq_Float` is used when both operands are float
- May require fixes in dictionary elaboration or builtin resolution

#### Tasks
- **Type class** (`internal/types/typeclass.go`): Integral
- **Instances** (`stdlib/std/prelude.ail`): Integral[Int] only (no Float)
- **Elaboration** (`internal/elaborate/elaborate.go`): % → mod
- **Defaulting** (`internal/types/typechecker_core.go`): prefer Int
- **Float comparison fix** (`internal/elaborate/elaborate.go` or `internal/eval/builtins.go`):
  - Debug why `==` on float uses `eq_Int` instead of `eq_Float`
  - Ensure proper dictionary lookup for Eq[Float]
  - May need fixes in dictionary passing or builtin dispatch
- **Tests**:
  - `5%3==2`; `5.0%3.0` → error with hint
  - `0.0 == 0.0` → true (using eq_Float)
  - `let x = 5.0 in x == 0.0` → works correctly
  - ADT with float comparison (the adt_option benchmark case)

#### Acceptance Criteria
- ✅ `%` works on Int
- ✅ Errors clearly on Float with actionable hint for %
- ✅ Float comparison `==` uses `eq_Float` correctly
- ✅ adt_option benchmark passes with float division guard

**Note for CHANGELOG**:
- Fixed: % modulo operator previously failed due to ambiguous typing (Num ∧ Ord). v0.3.0 introduces an Integral type class and maps % to mod, defaulting to Int. Floating-point % is disallowed with a precise error message.
- Fixed: Float equality comparison (`==`) was incorrectly using `eq_Int` instead of `eq_Float`, causing runtime errors. Dictionary resolution now properly selects the correct type class instance for float comparisons.

---

### M-UX2: Dev Experience Polish (P2)

**Effort**: ~400 LOC | **Owner**: cli/tools/docs

#### Tasks
- **Debug flag** — Add `--debug` CLI flag to enable/disable debug logging globally
  - Wrap all DEBUG statements in conditional checks
  - Add `Debug bool` to Config struct
  - Support both CLI flag and `AILANG_DEBUG` environment variable
  - Clean output by default; verbose diagnostics when enabled
- **Better recursion diagnostics** — suggest `--debug`, `--max-recursion-depth=N`
- **Audit script** — detect Clock/Net usage and auto-add caps
- **Micro examples**: `micro_recursive_sum`, `micro_record_person`, `micro_clock_timer`, `micro_net_json`
- **Docs**: `docs/guides/recursion.md`, `records.md`, update `effects.md` (Clock/Net)

#### Acceptance Criteria
- ✅ `--debug` flag controls all debug output globally
- ✅ 4+ new micros; improved audit; docs added
- ✅ Clean output by default; verbose diagnostics with `--debug`

**Files Modified**:
- `cmd/ailang/main.go` — Add `--debug` flag and Config.Debug field
- `internal/pipeline/pipeline.go` — Conditional DEBUG statements based on Config.Debug
- `internal/iface/builder.go` — Conditional DEBUG statements
- `internal/link/topo.go` — Conditional DEBUG statements
- `internal/loader/loader.go` — Conditional DEBUG statements
- `internal/runtime/eval.go` — Better recursion errors + conditional DEBUG
- `tools/audit-examples.sh` — Clock/Net detection
- `examples/micro_*.ail` — New examples
- `docs/guides/*.md` — New guides

---

## Success Metrics

| Metric | v0.2.0 (post-fix) | v0.3.0 Target |
|--------|-------------------|---------------|
| **Passing examples** | 36/53 (68%) | **≥50 (≥94%)** |
| **Recursion** | Broken | Working |
| **Records** | Partial | Row-poly access working |
| **% operator** | Broken | Working (Integral) |
| **Float comparison** | Broken (uses eq_Int) | Working (uses eq_Float) |
| **Effects** | IO, FS | + Clock, Net |
| **Test coverage** | 27.1% | ≥30% (stretch 35%) |

### RC Gate
- ✅ **Recursion must work** (P0)
- ✅ **Records must be usable** for common shapes (closed rows acceptable; open rows can be partial)
- ✅ **% must be fixed** or documented w/ explicit deferral to 0.3.1
- ✅ **Security for Net** must meet deny-by-default + protocol/domain guard

---

## Implementation Plan (2 Weeks)

### Week 1
- **Days 1–3**: M-R4 Recursion
- **Days 3–5**: M-R5 Records
- **Day 5**: Sanity pass on examples unblocked by R4/R5

### Week 2
- **Days 6–7**: M-R6 Clock/Net
- **Day 8**: M-R7 Integral/%
- **Days 9–10**: M-UX2, micro examples, audit script
- **Day 10**: Full suite run + triage
- **Days 11–12**: Docs, CHANGELOG, release prep

### Scope Guardrails
- If **Clock/Net security hardening slips** → move Net to 0.3.1; ship Clock only
- If **rows complexity spikes** → ship closed records + partial rows; defer open/extensible records to 0.3.1

---

## Risks & Mitigations

| Risk | Severity | Mitigation |
|------|----------|------------|
| **Recursive evaluation pitfalls** (mutual recursion, cycles) | High | Pre-bind closures; regression tests; depth guard |
| **Row polymorphism complexity** | Medium | Start with closed records; partial row support; clear errors |
| **Networking security** (SSRF, localhost) | High | Deny by default; protocol/domain allowlist; timeouts |
| **% defaulting regressions** | Low | Confine change to Integral; robust tests |
| **Timeline pressure** | Medium | **P0**: Recursion; **P0**: usable Records; **P1**: Clock; **P1**: %; Net can slip |

---

## Testing Strategy

### Unit Tests
- **Recursion**: factorial/fib/quicksort; mutual recursion
- **Records**: field access, subset unification, missing field diagnostics
- **%**: int pass, float error with actionable hint
- **Effects**: Clock/Net stubs with caps/no-caps behavior

### Integration Tests
- Run all existing examples; expect ≥50 passing
- New micros for recursion/records/clock/net
- Net tests use deterministic endpoints or mocked runner

### Security Tests
- **Net**: block file://, localhost, private ranges by default
- **Caps**: operations fail without `--caps`; CLI help lists valid caps
- **Timeouts** enforced; configurable via env/flag (documented)

### Regression Tests
- All v0.2.0 examples must still pass
- No regressions in type classes, pattern matching, effects

---

## Definition of Done

### Code Complete
- ✅ M-R4/M-R5 complete; % fixed; Clock complete; Net complete or safely deferred
- ✅ No regressions in 0.2.0 examples
- ✅ ≥50 examples pass locally and in CI

### Documentation Complete
- ✅ README and CHANGELOG updated
- ✅ Guides: recursion, records, effects (Clock/Net)
- ✅ Note % root cause + Integral in CHANGELOG ("Type System: Breaking issue fixed")

### Release Ready
- ✅ CI green (tests, lint)
- ✅ Binaries for macOS (amd64/arm64), Linux, Windows
- ✅ Git tag v0.3.0, release notes published

---

## File Changes (DRI)

### Runtime (~600 LOC)
- `internal/runtime/eval.go` — LetRec/thunks; recursion diagnostics
- `internal/eval/value.go` — closure self-ref/indirection
- `internal/runtime/resolver.go` — recursion-safe name resolution
- `internal/runtime/builtins.go` — `_clock_*`, `_net_*`

### Type System (~500 LOC)
- `internal/types/unification.go` — record field/row unification
- `internal/types/types.go` — row variables
- `internal/types/typechecker_core.go` — field access; Integral defaulting
- `internal/types/typeclass.go` — Integral class

### Elaboration (~100 LOC)
- `internal/elaborate/elaborate.go` — % → mod

### Stdlib (~350 LOC)
- `stdlib/std/clock.ail`, `stdlib/std/net.ail`
- `stdlib/std/prelude.ail` — Integral[Int]

### Effects (~150 LOC)
- `internal/effects/context.go` — Clock/Net capabilities

### CLI (~100 LOC)
- `cmd/ailang/main.go` — `--debug` flag and Config.Debug field

### Examples/Tools (~400 LOC)
- `examples/recursive_*.ail`, `micro_record_person.ail`, `micro_clock_*.ail`, `micro_net_*.ail`
- `tools/audit-examples.sh` — Clock/Net detection and caps

### Documentation (~200 LOC)
- `docs/guides/recursion.md` — NEW
- `docs/guides/records.md` — NEW
- `docs/guides/effects.md` — UPDATED (Clock/Net sections)
- `CHANGELOG.md`, `README.md` — UPDATED

**Total**: ~2,400 LOC (conservative estimate)

---

## Out of Scope (v0.3.0)

### Deferred to v0.3.1
- Open/extensible records with full row calculus (ship partial; finish in 0.3.1/0.4.0)
- Advanced effect handlers/composition DSL
- Performance optimizations (tail call elimination)
- More effects: Random, Env

### Deferred to v0.4.0 (Safety Focus)
- Semantic annotations (`@intent`, `@requires`, `@ensures`)
- Resource budgets (computation limits)
- Info-flow labels (PII/Secret tracking)
- Linear/affine types (resource lifecycle)
- Policy DSL (org-level constraints)

### Deferred to v0.5.0 (Concurrency)
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

### Mutual Recursion Example
```ailang
-- examples/recursive_mutual.ail
module examples/recursive_mutual

export func isEven(n: int) -> bool {
  if n == 0 then true else isOdd(n - 1)
}

export func isOdd(n: int) -> bool {
  if n == 0 then false else isEven(n - 1)
}

export func main() -> bool {
  isEven(4)  -- Returns: true
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

### Modulo Example (Fixed)
```ailang
-- examples/arithmetic_modulo.ail
module examples/arithmetic_modulo

export func main() -> int {
  5 % 3  -- Returns: 2 (now works via Integral type class)
}
```

---

## Next Steps After v0.3.0

1. **Immediate (v0.3.1)**: Bugfixes, performance tuning (tail call elimination), missing operator edge cases
2. **Short-term (v0.4.0)**: Safety features (budgets, annotations, linear types)
3. **Medium-term (v0.5.0)**: Concurrency (async, channels, session types)
4. **Long-term (v1.0)**: Feature-complete, production-ready

**Philosophy**: Each release adds one major capability cluster while maintaining stability and security.

---

**Document Version**: v2.0 - TIGHTENED
**Created**: October 4, 2025
**Last Updated**: October 4, 2025
**Author**: AILANG Development Team

**Changes from v1.0**:
- Tightened acceptance criteria with crisp checkboxes
- Added RC gate requirements (must-haves for release)
- Clarified P0/P1 priorities with scope guardrails
- Documented % root cause explicitly for CHANGELOG
- Added mutual recursion examples
- Refined risk mitigations with specific strategies
- Added DRI (Directly Responsible Individual) roles per area
- Scope guardrails for slippage scenarios
- Clearer security requirements for Net effect

---

## Conclusion

v0.3.0 will transform AILANG from a demonstration language to a practical tool for real programs by enabling:

1. **Recursion** - Unlocking fundamental programming patterns (factorial, fibonacci, quicksort)
2. **Records** - Enabling proper data modeling with row polymorphism
3. **Clock/Net** - Connecting to the real world (time, HTTP)
4. **Type fixes** - Removing known blockers (modulo operator via Integral)

**Ship v0.3.0** with these four pillars to prove AILANG can handle real-world program structures, then iterate toward v1.0 production readiness with safety features (budgets, annotations, linear types) and concurrency (async, session types).

---

*Drafted by Claude Sonnet 4.5 with input from AILANG Development Team*
*Tightened and refined based on v0.2.0 learnings*
*October 4, 2025*
