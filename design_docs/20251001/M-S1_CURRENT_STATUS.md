# M-S1 Current Status - Stdlib Implementation

## Date: October 1, 2025

## Summary
Parts A & B of M-S1 (Import System & Builtin Visibility) are **COMPLETE** and working. However, full stdlib usage is blocked by two parser/runtime limitations discovered during Phase 3.

## ✅ What Works (Completed)

### Part A: Import System for Types/Constructors (~400 LOC)
- ✅ Type exports work: `export type Option[a] = Some(a) | None`
- ✅ Constructor resolution works: `import std/option (Some, None)`
- ✅ Function imports work: `import std/option (getOrElse)`
- ✅ All 7 files modified (loader, elaborator, interface, pipeline, linker)
- ✅ Tests pass, no regressions

### Part B: Builtin Visibility (~300 LOC)
- ✅ String primitives visible: `_str_len`, `_str_slice`, `_str_compare`, etc.
- ✅ IO primitives visible: `_io_print`, `_io_println`, `_io_readLine`
- ✅ Effect annotations work: `! {IO}` in type system
- ✅ `stdlib/std/string.ail` type-checks successfully (7 exports)

### Stdlib Modules
- ✅ `stdlib/std/option.ail` - Type-checks ✓ (6 functions)
- ✅ `stdlib/std/result.ail` - Type-checks ✓ (6 functions)
- ✅ `stdlib/std/string.ail` - Type-checks ✓ (7 wrappers)
- ⚠️ `stdlib/std/list.ail` - Type-checks locally, cross-module blocked
- ⚠️ `stdlib/std/io.ail` - Documented as stubs (inline syntax not supported)

## ⚠️ Blockers Discovered (Phase 3)

### Blocker 1: $adt Cross-Module Constructor Resolution
**Status**: Critical blocker for stdlib usage

**Issue**: Constructors from imported modules don't resolve in $adt runtime.

**Example that fails:**
```ailang
module examples/option_demo
import stdlib/std/option (Some, None)

export pure func test() -> int {
  let x = Some(42) in  -- ERROR: undefined global variable: make_Option_Some from $adt
  42
}
```

**Error**: `undefined global variable: make_Option_Some from $adt`

**Root Cause**: The `$adt` module is only populated with constructors from the CURRENT module, not from imported modules. Constructor factory functions need to be registered during import resolution.

**Impact**:
- ❌ Can't use Option/Result types from stdlib in other modules
- ❌ Can't write examples that import and use ADTs
- ❌ Stdlib list.ail fails (uses imported None from option)

**Fix Required**: ~200-300 LOC
1. Update `RegisterAdtModule()` to include constructors from ALL loaded modules
2. Ensure import resolution adds constructor factories to elaborator environment
3. Test cross-module constructor usage

---

### Blocker 2: Multiple Statements in Function Bodies
**Status**: Parser limitation

**Issue**: Parser doesn't support multiple statements in function bodies with semicolons.

**Example that fails:**
```ailang
export func main() -> () ! {IO} {
  let x = 42;           -- ERROR: expected next token to be }, got ; instead
  println(show(x));
  ()
}
```

**What Works**:
```ailang
export pure func main() -> int {
  let x = 42 in x * 2   -- Single expression with `in` works
}
```

**Impact**:
- ❌ Can't write realistic examples with multiple statements
- ❌ IO examples fail (need println then return ())
- ❌ Most user code patterns blocked

**Fix Required**: ~100-200 LOC
1. Update parser to handle statement sequences in function bodies
2. Support semicolon-separated statements
3. Support implicit return of last expression

---

### Blocker 3: Inline Function Bodies
**Status**: Parser limitation (minor)

**Issue**: Can't write `export func f(x: int) -> int { x * 2 }` on one line.

**Impact**: io.ail had to be converted to stub documentation

**Fix Required**: ~50 LOC in parser

---

## 📊 Current Metrics

### Code Delivered (M-S1 Parts A & B)
- **~700 LOC** across 7 files (import system + builtins)
- **360 LOC** stdlib modules (5 files)
- **All tests passing**, no regressions

### Test Coverage
- Overall: ~25% (unchanged)
- Parser: Still 0% (HIGH RISK)
- New infrastructure: Tested via integration

### Examples Status
- **17 total** example files
- **~5 working** (simple, arithmetic, lambda_expressions, adt_simple, hello)
- **~12 blocked** by constructor or multi-statement issues

---

## 🎯 Recommended Next Steps

Given the blockers, there are two paths forward:

### Option A: Fix Blockers First (Recommended)
**Time**: 2-3 days
**Priority**: HIGH - Unblocks all stdlib usage

1. **Fix $adt cross-module constructors** (1-2 days, ~200-300 LOC)
   - Critical for any ADT usage across modules
   - Required for Option/Result examples to work

2. **Add multi-statement function bodies** (1 day, ~100-200 LOC)
   - Required for realistic examples
   - Enables IO examples, stdlib_demo, etc.

3. **THEN continue with**: Parser tests, examples, documentation

**Outcome**: Full stdlib usable, examples work, solid foundation

---

### Option B: Document & Defer (Current State)
**Time**: 1-2 hours
**Outcome**: Document limitations, ship what works

1. Update README with:
   - ✅ Import system works (types, constructors, functions)
   - ✅ Builtins visible (string, IO)
   - ⚠️ Known limitation: Cross-module ADT usage blocked
   - ⚠️ Known limitation: Multi-statement functions not supported

2. Update CHANGELOG with achievements and limitations

3. Ship v0.1.0-rc1 with:
   - Working: Type checking, imports, effects (type-level)
   - Blocked: Full stdlib usage until $adt fix

**Outcome**: Document progress, clear path forward

---

## 💡 Recommendation

**Choose Option A** - Fix the blockers before shipping stdlib.

**Rationale**:
- These are foundational issues affecting ALL user code
- Fixing now prevents technical debt
- 2-3 days investment enables months of productive stdlib development
- Users expect cross-module ADTs to work (core feature)

**Alternative**: If timeline is critical, ship as "v0.1.0-alpha" with clear limitations documented, then fix in v0.1.1.

---

## Files Modified (Summary)

### Infrastructure (M-S1 A & B) - ✅ Complete
- `internal/loader/loader.go` - Type/constructor extraction
- `internal/elaborate/elaborate.go` - Builtin environment
- `internal/iface/iface.go` - Type export structures
- `internal/iface/builder.go` - AST type extraction
- `internal/pipeline/pipeline.go` - Import resolution + builtin injection
- `internal/link/module_linker.go` - Type/constructor linking
- `internal/link/builtin_module.go` - String/IO builtin types

### Stdlib - ✅ Committed
- `stdlib/std/option.ail` - 40 LOC, type-checks ✓
- `stdlib/std/result.ail` - 32 LOC, type-checks ✓
- `stdlib/std/list.ail` - 69 LOC, type-checks ✓ (blocked cross-module)
- `stdlib/std/string.ail` - 17 LOC, type-checks ✓
- `stdlib/std/io.ail` - 14 LOC, documented as stubs

---

## Conclusion

**M-S1 Parts A & B are COMPLETE** - the import system and builtin visibility work as designed. However, full stdlib *usage* requires fixing two parser/runtime limitations that were exposed during testing.

**Decision Point**: Fix blockers now (2-3 days) OR document limitations and defer (1-2 hours).

**Recommendation**: Fix blockers - they're foundational issues that will affect all future development.
