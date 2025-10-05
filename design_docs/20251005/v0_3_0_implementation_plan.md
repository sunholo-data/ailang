# v0.3.0 Implementation Plan

**Version**: v0.3.0
**Status**: ACTIVE (Ready for Implementation)
**Target Release**: October 18, 2025 (2 weeks from Oct 5)
**Author**: AILANG Development Team
**Created**: October 4, 2025
**Last Updated**: October 5, 2025 (Sprint Plan Analysis)

---

## Sprint Plan Analysis (October 5, 2025)

### Reality Check: Current Baseline

**Actual v0.2.0 Status** (as of Oct 5, 2025):
- Production code: ~31,864 LOC
- Test coverage: **27.1%** (not 31.3% as previously thought)
- Examples passing: **32/51 (62.7%)** (not 42/53 - that was a temporary spike, baseline is 32)
- Recent velocity: **400-600 LOC/day** for well-scoped milestones

**Recent Implementation Velocity** (from CHANGELOG analysis):
1. M-EVAL (Oct 2): ~600 LOC in 1 day
2. M-R3 Pattern Matching (Oct 2): ~700 LOC in 1 day
3. M-R2 Effects (Oct 1-2): ~1,550 LOC in 2-3 days
4. M-R1 Module Runtime (Sept 30-Oct 1): ~1,874 LOC in 4-5 days

**Average: 400-600 LOC/day** for features with clear scope

### Critical Discovery: Float Comparison Bug

**NEW BUG FOUND** (not in original plan):
- **Float comparison broken**: `==` on floats incorrectly uses `eq_Int` instead of `eq_Float`
- Discovered in adt_option benchmark (Oct 3)
- **Added to M-R7 scope**

### Revised Success Targets

| Metric | v0.2.0 Baseline | Original v0.3.0 Target | **REVISED v0.3.0 Target** |
|--------|-----------------|------------------------|---------------------------|
| **Passing examples** | 32/51 (62.7%) | 50/60 (83%) | **≥42/51 (82%)** |
| **Recursion** | Broken | Working | **Working** |
| **Records** | Partial | Row-poly access | **Usable (closed rows OK)** |
| **% operator** | Broken | Working (Integral) | **Working (Integral)** |
| **Float comparison** | Broken (uses eq_Int) | - | **Working (uses eq_Float)** |
| **Effects** | IO, FS | + Clock, Net | **+ Clock (Net stretch)** |
| **Test coverage** | 27.1% | ≥30% | **≥30%** |

### Scope Guardrails (Timeline Protection)

**Priority Levels**:
- **P0 (MUST SHIP)**: M-R4 Recursion, M-R7 Type Fixes, M-R5 Records → **1,400 LOC, ~7 days**
- **P1 (STRETCH)**: M-R6 Clock → **250 LOC, ~1 day**
- **P1 (OPTIONAL)**: M-R6 Net → **450 LOC, ~2 days** ← **DEFER if timeline tight**
- **P2 (NICE TO HAVE)**: M-UX2 Polish → **400 LOC, ~2 days**

**Decision Tree**:
```
Timeline OK? → Ship all (P0 + P1 + P2)
Timeline tight (Week 2)? → Ship P0 + Clock, defer Net to v0.3.1
Timeline very tight? → Ship P0 only, defer everything else
```

### Updated Implementation Plan

**Week 1: Core Blockers (Oct 5-11)**
- Day 1: ✅ **M-R4 Recursion** (P0, ~1,780 LOC) - COMPLETE (v0.3.0-alpha1)
- Day 2: ✅ **M-R8 Block Expressions** (P0, ~10 LOC fix) - COMPLETE (v0.3.0-alpha2) ✨
- Days 3-4: **M-R7 Type Fixes** (P0, 300 LOC - includes float comparison)
- Day 5: Buffer for unknowns
- Weekend: Buffer

**Week 2: Records & Polish (Oct 12-18)**
- Days 6-8: **M-R5 Records** (P0, 500 LOC)
- Day 9: **M-R6 Clock** (P1, 250 LOC) ← **Recommended: YES**
- Day 10: **M-R6 Net** (P1, 450 LOC) ← **Recommended: DEFER to v0.3.1**
- Days 11-12: **M-UX2 Polish** (P2, 400 LOC)
- Days 13-14: Testing, docs, release prep

### Code Estimate Breakdown

| Component | Implementation | Tests | Total | Priority | Days | Status |
|-----------|----------------|-------|-------|----------|------|--------|
| M-R4 Recursion | 400 LOC | 200 LOC | 600 LOC | P0 | 3 | ✅ COMPLETE |
| **M-R8 Block Expressions** | **~10 LOC** | **0 LOC** | **~10 LOC** | **P0** | **0.1** | ✅ **DONE** |
| M-R7 Type Fixes | 200 LOC | 100 LOC | 300 LOC | P0 | 2 | Planned |
| M-R5 Records | 350 LOC | 150 LOC | 500 LOC | P0 | 3 | Planned |
| M-R6 Clock | 200 LOC | 50 LOC | 250 LOC | P1 | 1 | Planned |
| M-R6 Net | 300 LOC | 150 LOC | 450 LOC | P1 | 2 | Planned |
| M-UX2 Polish | 300 LOC | 100 LOC | 400 LOC | P2 | 2 | Planned |
| **Total (all)** | **1,950 LOC** | **850 LOC** | **2,800 LOC** | - | **~13.5 days** | - |
| **Total (P0 only)** | **1,150 LOC** | **550 LOC** | **1,700 LOC** | **Critical** | **~7.5 days** | - |

**Velocity Check**:
- Recent average: 400-600 LOC/day
- P0 scope: 1,700 LOC = 3-4 days ideal (6-8 days with testing = **1.5 weeks realistic**)
- Full scope: 2,800 LOC = 5-7 days ideal (11-13 days with testing = **2 weeks TIGHT**)

**Conclusion**: 2-week timeline is achievable for P0 + Clock. Net and UX polish may slip.

**Impact of M-R8 Addition**: +300 LOC (+0.5 days) to P0 scope. Still comfortably fits in 2-week timeline and is **critical for AI compatibility** - unblocks Claude Sonnet 4.5's generated recursive code with blocks.

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
- **Long-term baseline**: 32/51 passing (62.7%) after cleanup
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
- **32/51 passing examples (62.7%)** — realistic baseline after post-release fixes

### What v0.3.0 Will Deliver

**P0 (MUST SHIP)**:
- **Recursion**: LetRec with self/mutual recursion (factorial/fib/quicksort)
- **Type fixes**: % modulo via Integral + float comparison via eq_Float
- **Records**: Closed records with field access (row polymorphism partial OK)

**P1 (STRETCH)**:
- **Clock effect**: now(), sleep() with capability enforcement
- **Net effect**: httpGet(), httpPost() with security sandbox ← **DEFER if timeline tight**

**P2 (NICE TO HAVE)**:
- **UX polish**: --debug flag, better errors, micro examples

**Target**: **≥42/51 examples passing (82%)**, +10 examples over baseline

**Theme**: Real-world programs without sacrificing safety.

---

## Status Snapshot

### Current (v0.2.0)
- ✅ Modules, IO/FS effects, pattern guards/exhaustiveness, ADT constructors, type classes, auto-entry/caps
- ❌ **Recursion fails** (LetRec/closure env)
- ⚠️ **Records partial** (unification/rows)
- ❌ **Modulo broken** (ambiguous constraints)
- ❌ **Float comparison broken** (uses eq_Int instead of eq_Float) ← **NEW BUG**
- ⚠️ **Only IO/FS** (need Clock/Net)

### Target (v0.3.0)
- ✅ Recursion works (self + mutual)
- ✅ Records unify (closed rows working, row polymorphism partial)
- ✅ % fixed via Integral type class
- ✅ Float comparison fixed (uses eq_Float)
- ✅ Clock effect (stretch: Net effect)
- ✅ ≥42 examples passing

---

## Milestones (Detailed Design Docs)

### M-R4: Recursion Support (P0 - MUST SHIP)

**Effort**: ~600 LOC | **Priority**: P0 | **Duration**: 3 days
**Design Doc**: [`design_docs/implemented/v0_3_0/M-R4_recursion.md`](../implemented/v0_3_0/M-R4_recursion.md)

**Why**: Fundamental building block; blocks factorial, fibonacci, quicksort, tree traversal.

#### Root Cause (v0.2.0)
- LetRec not evaluating into self-referential closures.
- Environments bound after function creation; no self backpointer.

#### Acceptance Criteria
- ✅ `factorial(5)` returns 120
- ✅ `fib(10)` returns 55
- ✅ `quicksort([3,1,4,1,5,9])` works
- ✅ Mutual recursion (isEven/isOdd) passes
- ✅ Stack overflow gives friendly error (not panic)

**Examples Unblocked**: +4 (factorial, fibonacci, quicksort, mutual)

**Status**: ✅ **COMPLETE** (v0.3.0-alpha1, commits df608e1 + 3cd4c33)

---

### M-R8: Block Expressions (P0 - MUST SHIP) ✅ **COMPLETE** (v0.3.0-alpha2)

**Actual Effort**: ~10 LOC (bug fix only!) | **Priority**: P0 | **Duration**: 2 hours
**Design Doc**: [`design_docs/implemented/v0_3_0/M-R8_block_expressions.md`](../implemented/v0_3_0/M-R8_block_expressions.md)
**Status**: ✅ SHIPPED in v0.3.0-alpha2

**Discovery**: Blocks were **already implemented**! Parser and elaboration both support `{ e1; e2; e3 }` syntax.

#### The Bug (Found & Fixed)
- ❌ **Root cause**: `findReferences()` in `scc.go` was missing a case for `*ast.Block`
- ❌ **Impact**: Recursive functions with blocks not detected as recursive
- ❌ **Symptom**: `if n <= 1 then { 1 } else { n * fact(n-1) }` → "undefined variable: fact"
- ✅ **Fix**: Added 5 lines to handle `*ast.Block` case in SCC analysis

#### The Fix (internal/elaborate/scc.go)
```go
case *ast.Block:
    // Blocks can contain function references in any expression
    for _, expr := range ex.Exprs {
        refs = append(refs, findReferences(expr)...)
    }
```

#### Results
- ✅ Self-recursion with blocks works
- ✅ Mutual recursion with blocks works
- ✅ All existing tests pass
- ✅ 3 new example files: `micro_block_seq.ail`, `micro_block_if.ail`, `block_recursion.ail`
- ✅ AI-generated code with blocks now works ✨

**Examples Unblocked**: All AI-generated code with blocks (critical for eval benchmarks)

**Impact**: 10 LOC fix with massive AI compatibility improvement!

---

### M-R7: Type System Fixes — Integral & Float Comparison (P0 - MUST SHIP)

**Effort**: ~300 LOC | **Priority**: P0 | **Duration**: 2 days
**Design Doc**: [`design_docs/20251005/M-R7_type_fixes.md`](../20251005/M-R7_type_fixes.md)

**Why**: Two critical type system bugs blocking arithmetic and comparisons.

#### Root Causes

**Issue 1: Modulo operator (`%`)**
- `%` had ambiguous constraints (Num ∧ Ord) and defaulting couldn't pick a concrete type.
- Example: `5 % 3` failed with "ambiguous type variable α with classes [Num, Ord]"

**Issue 2: Float comparison (`==` with float)** ← **NEW**
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
- Fix dictionary elaboration or builtin resolution

#### Acceptance Criteria
- ✅ `5 % 3` returns 2 (Int modulo works)
- ✅ `5.0 % 3.0` errors with "Float not Integral; use / for float division"
- ✅ `0.0 == 0.0` returns true (uses `eq_Float`)
- ✅ `let x = 5.0 in x == 0.0` works correctly
- ✅ adt_option benchmark passes with float comparison

**Examples Unblocked**: +2-3 (using % or float comparison)

---

### M-R5: Records & Row Polymorphism (P0 - MUST SHIP)

**Effort**: ~500 LOC | **Priority**: P0 | **Duration**: 3 days
**Design Doc**: [`design_docs/20251005/M-R5_records.md`](../20251005/M-R5_records.md)

**Why**: Data modeling & ergonomics; unblocks current broken examples.

#### Tasks
1. **Complete TRecord unification** (`internal/types/unification.go`)
   - Field-by-field unification; subset/subsumption rules
   - Diagnostics for missing/mismatched fields

2. **Row variables** (`internal/types/types.go`)
   - Represent `{k:v | ρ}`; unify rows; scoped generalization
   - **Partial implementation acceptable** for v0.3.0

3. **Field access typing** (`internal/types/typechecker_core.go`)
   - Generate fresh row var; unify required field; better errors

4. **Runtime record ops** (`internal/eval/eval_core.go`)
   - Field access on RecordValue with clear failures

#### Acceptance Criteria
- ✅ `{x:1, y:2}.x` returns 1
- ✅ Subset unification: `{x:1}` unifies with `{x:1, y:2}`
- ✅ Missing field error: "field 'z' not found in record {x:int, y:int}"
- ✅ Nested records: `{addr: {street: "Main"}}.addr.street` works

**Note**: Full row polymorphism can be partial. Closed records MUST work.

**Examples Unblocked**: +3-5 (records.ail, list_patterns.ail, lambda_expressions.ail partial)

---

### M-R6: Extended Effects — Clock & Net (P1 - STRETCH)

**Effort**: ~700 LOC (250 Clock + 450 Net) | **Priority**: P1 | **Duration**: 3 days
**Design Doc**: [`design_docs/20251005/M-R6_clock_net_effects.md`](../20251005/M-R6_clock_net_effects.md)

**Why**: Practicality; enables time-based logic and HTTP interactions.

#### API
- **Clock** (`std/clock`): `now() -> int`, `sleep(ms:int) -> ()`
- **Net** (`std/net`): `httpGet(url) -> string`, `httpPost(url, body) -> string`

#### Scope Decision
- **Clock (P1 - RECOMMENDED: YES)**: Simple, low-risk, 250 LOC, 1 day
- **Net (P1 - RECOMMENDED: DEFER)**: Security complexity, 450 LOC, 2 days

**IF Week 2 timeline is tight**: Ship Clock only, defer Net to v0.3.1

#### Acceptance Criteria (Clock)
- ✅ `now()` returns Unix timestamp (ms)
- ✅ `sleep(100)` blocks for 100ms
- ✅ Virtual time works in deterministic mode (`AILANG_SEED`)
- ✅ Requires `--caps Clock`, fails without

#### Acceptance Criteria (Net - if shipped)
- ✅ `httpGet(url)` fetches HTTP/HTTPS
- ✅ Localhost, private IPs blocked by default
- ✅ file://, ftp://, data:// rejected
- ✅ 30s timeout enforced
- ✅ `--net-allow` allowlist works

**Examples Unblocked**: +2 (Clock) or +4 (Clock + Net)

---

### M-UX2: Dev Experience Polish (P2 - NICE TO HAVE)

**Effort**: ~400 LOC | **Priority**: P2 | **Duration**: 2 days
**Recommended**: Defer to v0.3.1 if timeline tight

#### Tasks
- **Debug flag** — Add `--debug` CLI flag to enable/disable debug logging globally
  - Wrap all DEBUG statements in conditional checks
  - Add `Debug bool` to Config struct
  - Support both CLI flag and `AILANG_DEBUG` environment variable
- **Better recursion diagnostics** — suggest `--debug`, `--max-recursion-depth=N`
- **Audit script** — detect Clock/Net usage and auto-add caps
- **Micro examples**: 4+ new examples
- **Docs**: `docs/guides/recursion.md`, `records.md`, update `effects.md`

#### Acceptance Criteria
- ✅ `--debug` flag controls all debug output globally
- ✅ 4+ new micro examples
- ✅ Improved audit script
- ✅ Documentation updated

---

## Success Metrics (REVISED)

| Metric | v0.2.0 Baseline | v0.3.0 Target | Gate |
|--------|-----------------|---------------|------|
| **Passing examples** | 32/51 (62.7%) | **≥42/51 (82%)** | Must achieve |
| **Recursion** | Broken | Working | Must fix |
| **Records** | Partial | Usable (closed rows OK) | Must fix |
| **% operator** | Broken | Working (Integral) | Must fix |
| **Float comparison** | Broken (eq_Int) | Working (eq_Float) | Must fix |
| **Effects** | IO, FS | + Clock (Net stretch) | Clock required |
| **Test coverage** | 27.1% | ≥30% | Nice to have |

### RC Gate (Release Criteria)
- ✅ **Recursion must work** (P0 - factorial, fib, quicksort pass)
- ✅ **Records must be usable** for common shapes (closed rows acceptable; open rows can be partial)
- ✅ **% must be fixed** via Integral type class
- ✅ **Float comparison must work** using eq_Float
- ✅ **Clock effect** implemented (Net can slip to v0.3.1)
- ✅ **No regressions** in v0.2.0 examples
- ✅ **≥42 examples passing** (10+ improvement over baseline)

---

## Implementation Plan (2 Weeks - REVISED)

### Week 1: Core Blockers (Oct 5-11)
- **Days 1-3**: M-R4 Recursion (P0, 600 LOC)
  - LetRec with self-referential closures
  - Mutual recursion support
  - Stack overflow protection
- **Days 4-5**: M-R7 Type Fixes (P0, 300 LOC)
  - Integral type class for % modulo
  - Float comparison fix (eq_Float)
- **Weekend**: Buffer, code review, testing

### Week 2: Records & Polish (Oct 12-18)
- **Days 6-8**: M-R5 Records (P0, 500 LOC)
  - TRecord unification (closed records)
  - Row variables (partial implementation)
  - Field access type inference
- **Day 9**: M-R6 Clock (P1, 250 LOC) ← **Recommended: YES**
  - Clock effect implementation
  - Virtual time for deterministic mode
- **Day 10**: M-R6 Net (P1, 450 LOC) ← **Decision point: Ship or defer?**
  - If timeline OK: Net effect with security
  - If timeline tight: Defer to v0.3.1
- **Days 11-12**: M-UX2 Polish (P2, 400 LOC) or Testing
  - If time: Debug flag, examples, docs
  - Otherwise: Comprehensive testing, release prep
- **Days 13-14**: Final testing, documentation, CHANGELOG, release

### Scope Guardrails (Timeline Protection)
- **If Clock/Net security hardening slips** → Ship Clock only; defer Net to v0.3.1
- **If rows complexity spikes** → Ship closed records + partial rows; defer full row polymorphism to v0.3.1
- **If Week 2 timeline is very tight** → Cut M-UX2; ship P0 + Clock only
- **Emergency escape**: Ship P0 only (Recursion + Type Fixes + Records), defer everything else

---

## Risks & Mitigations

| Risk | Severity | Likelihood | Mitigation |
|------|----------|------------|------------|
| **Recursive evaluation pitfalls** | High | Medium | Pre-bind closures; extensive tests; depth guards |
| **Row polymorphism complexity** | Medium-High | Medium | Start with closed records; partial rows acceptable |
| **Net security vulnerabilities** | Critical | Medium | **Defer Net to v0.3.1 if security can't be hardened** |
| **Float comparison fix scope creep** | Low | Low | Confine to Eq[Float]; don't touch broader type system |
| **Timeline pressure (2 weeks tight)** | Medium | High | **P0 non-negotiable**; P1/P2 can slip to v0.3.1 |

**Critical Decision Points**:
1. **End of Week 1**: Are P0 items complete? If not, cut P1/P2 immediately.
2. **Day 10**: Net security ready? If not, defer to v0.3.1 and focus on testing.
3. **Day 12**: UX polish achievable? If not, ship docs-only update.

---

## Testing Strategy

### Unit Tests (~750 LOC)
- **Recursion**: factorial/fib/quicksort; mutual recursion; stack overflow
- **Records**: field access, subset unification, missing field diagnostics
- **Type fixes**: % on Int (pass), % on Float (error); float comparison (eq_Float)
- **Effects**: Clock (real + virtual time); Net (security, if shipped)

### Integration Tests
- Run all existing examples; expect ≥42 passing
- New examples for recursion/records/clock/net
- Net tests use deterministic endpoints or mocked runner (if shipped)

### Security Tests (Net only, if shipped)
- **Net**: block file://, localhost, private ranges by default
- **Caps**: operations fail without `--caps`; CLI help lists valid caps
- **Timeouts** enforced; configurable via env/flag (documented)

### Regression Tests
- All v0.2.0 examples (32 baseline) must still pass
- No regressions in type classes, pattern matching, effects

---

## Definition of Done

### Code Complete
- ✅ M-R4/M-R5/M-R7 complete (P0 items)
- ✅ M-R6 Clock complete (P1)
- ✅ M-R6 Net complete OR safely deferred to v0.3.1
- ✅ M-UX2 complete OR deferred to v0.3.1
- ✅ No regressions in v0.2.0 examples (32 baseline)
- ✅ ≥42 examples pass locally and in CI

### Documentation Complete
- ✅ CHANGELOG updated with:
  - Recursion support (LetRec, mutual recursion)
  - Type fixes (Integral, float comparison)
  - Records (closed rows, partial row polymorphism)
  - Clock effect (if shipped)
  - Net effect (if shipped) or deferred note
- ✅ README updated with v0.3.0 status
- ✅ New guides (if time): recursion.md, records.md, effects.md update
- ✅ Design docs linked from CHANGELOG

### Release Ready
- ✅ CI green (tests, lint, all checks)
- ✅ Binaries for macOS (amd64/arm64), Linux, Windows
- ✅ Git tag v0.3.0, release notes published
- ✅ All P0 acceptance criteria met
- ✅ P1 items shipped or explicitly deferred

---

## File Changes (DRI)

### Runtime (~600 LOC)
- `internal/eval/eval_core.go` — LetRec/thunks; recursion diagnostics
- `internal/eval/value.go` — closure self-ref/indirection
- `internal/runtime/resolver.go` — recursion-safe name resolution
- `internal/runtime/builtins.go` — `_clock_*`, `_net_*` (if Net shipped)

### Type System (~800 LOC)
- `internal/types/unification.go` — record field/row unification
- `internal/types/types.go` — row variables
- `internal/types/typechecker_core.go` — field access; Integral defaulting; float comparison fix
- `internal/types/typeclass.go` — Integral class

### Elaboration (~100 LOC)
- `internal/elaborate/elaborate.go` — % → mod; float comparison dictionary selection

### Stdlib (~120 LOC)
- `stdlib/std/clock.ail` — Clock effect
- `stdlib/std/net.ail` — Net effect (if shipped)
- `stdlib/std/prelude.ail` — Integral[Int]

### Effects (~400 LOC)
- `internal/effects/clock.go` — NEW (Clock effect)
- `internal/effects/net.go` — NEW (Net effect, if shipped)
- `internal/effects/net_security.go` — NEW (Net security, if shipped)
- `internal/effects/context.go` — Clock/Net capabilities

### CLI (~100 LOC)
- `cmd/ailang/main.go` — `--debug` flag (if M-UX2 shipped), Net flags (if Net shipped)

### Examples/Tools (~500 LOC)
- `examples/recursive_*.ail` — NEW (4 files)
- `examples/micro_record_*.ail` — NEW (2 files)
- `examples/micro_clock_*.ail` — NEW (2 files)
- `examples/micro_net_*.ail` — NEW (2 files, if Net shipped)
- `tools/audit-examples.sh` — Clock/Net detection (if M-UX2 shipped)

### Documentation (~300 LOC)
- `docs/guides/recursion.md` — NEW (if M-UX2 shipped)
- `docs/guides/records.md` — NEW (if M-UX2 shipped)
- `docs/guides/effects.md` — UPDATED (Clock/Net sections, if M-UX2 shipped)
- `CHANGELOG.md` — UPDATED (v0.3.0 section)
- `README.md` — UPDATED (v0.3.0 status)
- `design_docs/20251005/*.md` — Design docs for milestones

### Tests (~750 LOC)
- `internal/eval/recursion_test.go` — NEW
- `internal/types/record_test.go` — NEW
- `internal/types/typeclass_test.go` — UPDATED (Integral)
- `internal/elaborate/float_test.go` — NEW
- `internal/effects/clock_test.go` — NEW
- `internal/effects/net_test.go` — NEW (if Net shipped)

**Total Estimate**:
- **P0 only**: ~1,400 LOC (7 days realistic)
- **P0 + Clock**: ~1,650 LOC (8-9 days realistic)
- **All (P0 + P1 + P2)**: ~2,500 LOC (12-14 days TIGHT)

---

## Out of Scope (v0.3.0)

### Deferred to v0.3.1
- Net effect (if timeline slips)
- UX polish: --debug flag, micro examples, docs (if timeline slips)
- Open/extensible records with full row calculus (ship partial; finish in v0.3.1/v0.4.0)
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
| **Effects** | IO, FS | + Clock (Net stretch) | + Random, Env, DB |
| **Type System** | HM + classes | + Records, Integral | + Linear, Session |
| **Pattern Matching** | Guards, exhaustive | Same | + View patterns |
| **Recursion** | ❌ Broken | ✅ Working | + Tail call optimization |
| **Records** | ⚠️ Partial | ✅ Usable (closed rows) | + Full row polymorphism |
| **Operators** | Most work, % broken | + Modulo, float == | All working |
| **Examples Passing** | 32/51 (63%) | 42/51 (82%) | 48/51 (94%) |
| **Security** | Basic caps | + Clock, Net sandbox | + Budgets, info-flow |

---

## Appendix: Design Documents

### Implemented (v0.3.0-alpha1)
1. **[M-R4_recursion.md](../implemented/v0_3_0/M-R4_recursion.md)** - ✅ COMPLETE: Recursion support with self/mutual recursion

### Planned (v0.3.0)
2. **[M-R8_block_expressions.md](../20251005/M-R8_block_expressions.md)** - Block syntax as syntactic sugar (NEW)
3. **[M-R7_type_fixes.md](../20251005/M-R7_type_fixes.md)** - Integral type class and float comparison fix
4. **[M-R5_records.md](../20251005/M-R5_records.md)** - Records with row polymorphism
5. **[M-R6_clock_net_effects.md](../20251005/M-R6_clock_net_effects.md)** - Clock and Net effects

Each doc contains:
- Problem statement with root cause analysis
- Implementation plan (day-by-day breakdown)
- Acceptance criteria (testable)
- Risk mitigation strategies
- Test strategy
- Code examples

---

## Next Steps After v0.3.0

1. **Immediate (v0.3.1)**: Ship deferred items (Net effect, UX polish), bugfixes, performance tuning
2. **Short-term (v0.4.0)**: Safety features (budgets, annotations, linear types), full row polymorphism
3. **Medium-term (v0.5.0)**: Concurrency (async, channels, session types)
4. **Long-term (v1.0)**: Feature-complete, production-ready

**Philosophy**: Each release adds one major capability cluster while maintaining stability and security.

---

**Document Version**: v3.0 - SPRINT PLAN ANALYSIS
**Created**: October 4, 2025
**Last Updated**: October 5, 2025
**Author**: AILANG Development Team

**Changes from v2.0**:
- Added sprint plan analysis with revised baseline (32/51 examples, 62.7%)
- Added float comparison bug to M-R7 scope (discovered Oct 3)
- Revised success targets to be realistic (42/51 vs 50/60)
- Updated velocity estimates based on actual v0.2.0 data (400-600 LOC/day)
- Added clear priority levels (P0/P1/P2) with scope guardrails
- Revised timeline with decision tree for scope cuts
- Added links to detailed milestone design docs
- Added code estimate breakdown with realistic day counts
- Clarified Net effect as "defer if timeline tight"
- Updated RC gate with specific pass/fail criteria

---

## Conclusion

v0.3.0 will transform AILANG from a demonstration language to a practical tool for real programs by enabling:

1. **Recursion** (P0) - Unlocking fundamental patterns (factorial, fibonacci, quicksort)
2. **Type fixes** (P0) - Removing known blockers (% modulo, float comparison)
3. **Records** (P0) - Enabling proper data modeling with closed rows (partial row polymorphism)
4. **Clock effect** (P1) - Connecting to real-world time (now, sleep)
5. **Net effect** (P1, optional) - HTTP integration (defer if timeline tight)

**Ship v0.3.0** with P0 + Clock (minimum) to prove AILANG can handle real-world program structures, then iterate toward v1.0 production readiness with safety features and concurrency in future releases.

**Critical Success Factors**:
- Week 1 focus: P0 items (Recursion + Type Fixes) non-negotiable
- Week 2 decision: Ship Clock (recommended) + Records, defer Net if needed
- Quality gate: ≥42 examples passing, no regressions, all P0 criteria met

---

*Drafted by Claude Sonnet 4.5 with input from AILANG Development Team*
*Sprint plan analysis and revision: October 5, 2025*
*Timeline: 2 weeks (Oct 5-18, 2025)*
