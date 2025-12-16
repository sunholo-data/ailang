# M-DX10: Complete S-CALL0 and Unit-Argument Model

**Status**: ✅ Implemented
**Target**: v0.4.6
**Completed**: v0.4.6
**Priority**: P0 - High (blocks CLI args feature)
**Estimated**: 2-3 days
**Dependencies**: S-CALL0 (partial implementation exists)

## Implementation Summary

**Implemented in v0.4.6** - All phases complete:

1. **Phase 1: Builtins aligned** - All "zero-arg" builtins updated to `NumArgs: 1` with unit parameter:
   - `_clock_now`: `internal/builtins/clock.go` - Type: `() -> int ! {Clock}`
   - `_env_getArgs`: `internal/builtins/env.go` - Type: `() -> [string] ! {Env}`
   - `_io_readLine`: `internal/builtins/io.go` - Type: `() -> string ! {IO}`
   - All implementations validate unit argument (defense against type system bugs)

2. **Phase 1.5: Entry invocation** - Runtime invokes `main()` with unit argument (`cmd/ailang/run_helpers.go:258`)

3. **Phase 2: S-CALL0 complete** - `f()` → `f(())` desugaring works in all contexts (expression, statement, lambda, match)

4. **Phase 3: Stdlib fixed** - All wrappers call builtins with `()`:
   - `std/clock.ail`: `now() = _clock_now()`
   - `std/env.ail`: `getArgs() = _env_getArgs()`
   - `std/io.ail`: `readLine() = _io_readLine()`

5. **Phase 4: Documentation** - Teaching prompts updated with unit-argument model explanation

**Verification**: All success criteria met. New builtins (e.g., `_sharedmem_keys` in v0.5.11) follow this pattern.

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | + | +1 | Enables clean `getArgs()` syntax without special cases |
| Preserve Semantic Clarity | + | +1 | Unifies all "zero-arg" functions under unit-argument model |
| Increase Determinism | + | +1 | Single, consistent desugaring rule: `f()` → `f(())` everywhere |
| Lower Token Cost | + | +1 | No arity-dependent rules for AI to learn; `f()` always works |
| **Net Score** | | **+4** | **Decision: Move forward (complete existing feature)** |

**Decision rule:** Net score > +1 → Move forward | ≤ 0 → Reject or redesign

**Reference:** See [AI-first DX philosophy](../v0_3_15/example-parity-vision-alignment.md#-design-principle-ai-first-dx)

## Problem Statement

**Discovered during M-LANG-CLI-ARGS implementation (v0.4.6):**

"Zero-argument" functions (`_env_getArgs`, `_clock_now`, `_io_readLine`) cannot be called correctly from AILANG code, blocking the CLI args feature and causing AI confusion.

**Current State:**
- `_env_getArgs` returns `<*eval.BuiltinFunction>` (function object) instead of `["arg1", "arg2"]`
- `_clock_now()` causes type error: "function arity mismatch: 0 vs 1"
- `_io_readLine()` has same issue
- Stdlib wrappers don't work: `export func getArgs() -> [string] ! {Env} = _env_getArgs`
- Only workaround: Call builtin implementation directly via `effects.Call()` (Go tests only)

**Impact:**
- **Blocks v0.4.6 CLI args feature** - cannot call `getArgs()` from AILANG code
- Affects existing builtins: `clock.now()`, `io.readLine()` (dormant bugs)
- AI models don't know whether to use `f`, `f()`, or `f ()`
- Inconsistent with AILANG's ML-style function application semantics

**Root Cause:**

AILANG **already chose the unit-argument model** with S-CALL0, but the implementation is incomplete:
1. **S-CALL0 sugar exists**: `f()` desugars to `f(())` in expression contexts
2. **Builtins are misaligned**: Registered as "zero-arg" in Go, expecting empty argument lists
3. **Stdlib wrappers are broken**: Try to call builtins without passing unit
4. **Teaching prompt is silent**: Doesn't explain the unit-argument model

This isn't a "missing feature" - it's an **incomplete feature rollout**.

## Semantic Model: No True Nullaries

**AILANG's Design Decision** (implicit in S-CALL0):

> **There are no semantically distinct nullary functions in AILANG.**
>
> All `func f() -> T` are surface syntax sugar for `func f(_ : ()) -> T`.
> All "zero-arg" calls are just applications to the unit value `()`.

**Why this model?**
- ✅ Consistent with ML tradition (`unit` as a proper type)
- ✅ Simplifies elaboration (no arity-dependent transformations)
- ✅ Preserves function application by juxtaposition (`f x`)
- ✅ Already implemented partially (S-CALL0)
- ✅ No special "nullary call" construct needed in core

**Type system:**
```
Surface:  func getArgs() -> [string] ! {Env}
Core:     () -> [string] ! {Env}          (single unit parameter)
```

**Call sites:**
```ailang
-- Surface syntax (S-CALL0 sugar)
let args = getArgs()

-- Desugars to
let args = getArgs (())

-- Both are valid; sugar is for ergonomics
```

## Goals

**Primary Goal:** Complete the unit-argument model by aligning builtins, stdlib, and documentation with S-CALL0

**Success Metrics:**
- All 3 affected builtins work: `_env_getArgs()`, `_clock_now()`, `_io_readLine()`
- Stdlib wrappers work correctly
- `cli_args` benchmark passes (currently 0%)
- No new language constructs (complete existing S-CALL0)
- AI models can reliably call "zero-arg" functions (measured via eval)
- Teaching prompt explicitly documents unit-argument model

## Solution Design

### Overview

**Complete the S-CALL0 feature by:**
1. Aligning builtin types to expect unit parameter
2. Ensuring S-CALL0 desugaring works everywhere
3. Fixing stdlib wrappers to pass unit
4. Documenting the unit-argument model

**No new semantics**. This is housekeeping.

### Architecture

**Components:**
1. **Builtins**: Update "zero-arg" builtins to have type `() -> T` (single unit param)
2. **S-CALL0**: Ensure `f()` → `f(())` transformation is complete
3. **Stdlib**: Update wrappers to call with `_builtin()`
4. **Teaching Prompt**: Document unit-argument model explicitly

**Invariant to enforce:**

> Every function that appears to take "no arguments" actually takes a single parameter of type `()` in core.

### Implementation Plan

**Phase 1: Align Builtins** (~1 day, 4 hours)

Audit all "zero-arg" builtins and update their Go implementations:

- [ ] **`_clock_now`** (`internal/builtins/clock.go`)
  - Current: Registered as `NumArgs: 0`
  - Fix: Change to `NumArgs: 1`, type `() -> int ! {Clock}`
  - Update `clockNowImpl` to expect 1 unit argument

- [ ] **`_env_getArgs`** (`internal/builtins/env.go`)
  - Current: Registered as `NumArgs: 0`
  - Fix: Change to `NumArgs: 1`, type `() -> [string] ! {Env}`
  - Update `envGetArgs` to expect 1 unit argument

- [ ] **`_io_readLine`** (`internal/builtins/io.go`)
  - Current: Registered as `NumArgs: 0` (assumed)
  - Fix: Change to `NumArgs: 1`, type `() -> string ! {IO}`
  - Update `readLineImpl` to expect 1 unit argument

- [ ] **Update golden snapshots**
  - Regenerate `internal/pipeline/testdata/builtin_types.golden`
  - Verify types show `() -> T` not `T`

**Phase 1.5: Entry Invocation** (~0.5 days, 2 hours)

Update runtime to invoke `main()` with unit argument:

- [ ] **Runtime invocation** (`cmd/ailang/main.go`, `internal/runtime/`)
  - Current: Assumes `main` has arity 0, calls with empty args
  - Fix: Call `main` with unit value: `main(())`
  - Consistent with `() -> () ! {E}` type

- [ ] **Entry detection**
  - Verify entry modules with `main()` still work
  - Test with various effect combinations: `! {IO}`, `! {IO, Env}`, etc.

- [ ] **Regression tests**
  - Entry modules with different effect signatures
  - Ensure no "arity mismatch" errors at runtime

**Phase 2: Complete S-CALL0** (~0.5 days, 2 hours)

Ensure `f()` → `f(())` desugaring works in **all** contexts:

**Where S-CALL0 must work:**
- [ ] **Expression contexts**:
  - `let x = f()` ✅ (already works)
  - `if cond then f() else g()`
  - `f() ++ g()` (binary operators)
  - Right-hand side of bindings

- [ ] **Statement contexts**:
  - Top of block: `{ f(); g(); }` ✅ (already works)
  - Sequence expressions
  - Effect statements

- [ ] **Lambda bodies**:
  - `\x. f()` (important for HOF)
  - Nested lambdas

- [ ] **Match arms** (expression position):
  - `match x { _ => f() }` (this bit us before!)
  - Guard expressions if implemented

- [ ] **Top-level in entry modules**:
  - If top-level `f()` is allowed (currently not, but future-proof)

**Where () must NOT appear:**
- ❌ **Patterns**: `match f() { ... }` (invalid - `f()` is expr, not pattern)
- ❌ **Type syntax**: `() -> T` is a type, not `f()`

- [ ] **Regression tests** (`internal/elaborate/elaborator_test.go`):
  - One test per context listed above
  - Verify AST transformation: `f()` → `App(f, UnitLit)`
  - Verify error messages for invalid positions

**Phase 3: Fix Stdlib Wrappers** (~0.5 days, 2 hours)

Update stdlib to use S-CALL0 sugar:

- [ ] **`std/clock.ail`**:
  ```ailang
  -- Before (broken)
  export func now() -> int ! {Clock} = _clock_now

  -- After (fixed)
  export func now() -> int ! {Clock} = _clock_now()
  ```

- [ ] **`std/env.ail`**:
  ```ailang
  -- Before (broken)
  export func getArgs() -> [string] ! {Env} = _env_getArgs

  -- After (fixed)
  export func getArgs() -> [string] ! {Env} = _env_getArgs()
  ```

- [ ] **`std/io.ail`**:
  ```ailang
  -- Before (broken)
  export func readLine() -> string ! {IO} = _io_readLine

  -- After (fixed)
  export func readLine() -> string ! {IO} = _io_readLine()
  ```

**Phase 4: Documentation & Testing** (~0.5 days, 2 hours)

- [ ] **Teaching prompt updates** (`prompts/v0.4.6.md`):
  - Add "Zero-Argument Functions" section
  - Explain unit-argument model
  - Show `f()` syntax and desugaring
  - Clarify that `()` is unit value, not "empty args"

- [ ] **Integration tests**:
  - Fix `tests/cli_args_test.ail` to use `getArgs()`
  - Test `clock.now()` in examples
  - Test `io.readLine()` if examples exist

- [ ] **Benchmark**:
  - Run `cli_args` benchmark
  - Target: 0% → 80%+ success rate

### Files to Modify/Create

**Modified files:**

**Phase 1: Builtins**
- `internal/builtins/clock.go` - Change `NumArgs: 0` → `1`, add unit validation (~15 LOC)
- `internal/builtins/env.go` - Change `NumArgs: 0` → `1`, add unit validation (~15 LOC)
- `internal/builtins/io.go` - Change `NumArgs: 0` → `1`, add unit validation (~15 LOC)
- `internal/pipeline/testdata/builtin_types.golden` - Update types (~3 LOC)

**Phase 1.5: Entry Invocation**
- `cmd/ailang/main.go` - Update `main()` invocation to pass unit (~10 LOC)
- `internal/runtime/runtime.go` - Ensure entry calls use unit argument (~10 LOC)
- `internal/runtime/runtime_test.go` - Test entry with various effect signatures (~30 LOC)

**Phase 2: S-CALL0**
- `internal/elaborate/elaborator_test.go` - Comprehensive S-CALL0 regression tests (~60 LOC)
- `internal/parser/parser_test.go` - Parser tests for `f()` contexts (~20 LOC)

**Phase 3: Stdlib**
- `std/clock.ail` - Add `()` to call: `_clock_now()` (~1 LOC)
- `std/env.ail` - Add `()` to call: `_env_getArgs()` (~1 LOC)
- `std/io.ail` - Add `()` to call: `_io_readLine()` (~1 LOC)
- `tests/cli_args_test.ail` - Use `getArgs()` instead of direct builtin (~2 LOC)

**Phase 4: Documentation**
- `prompts/v0.4.6.md` - Document unit-argument model with examples (~50 LOC)

**New files:**
- `design_docs/planned/v0_4_6/NULLARY-FUNCTIONS-SPEC.md` - Canonical spec (~100 LOC)

**Total: ~193 LOC (45 impl + 110 tests + 38 docs + stdlib)**

## Examples

### Example 1: CLI Arguments (The Blocker)

**Before (broken):**
```ailang
import std/env (getArgs)

export func main() -> () ! {IO, Env} {
  let args = getArgs;  -- ERROR: Returns function object, not list
  println(show(args))  -- Prints: <*eval.BuiltinFunction>
}
```

**After (fixed):**
```ailang
import std/env (getArgs)

export func main() -> () ! {IO, Env} {
  let args = getArgs();  -- ✓ Desugars to getArgs(()), returns ["arg1", "arg2"]
  println(show(args))    -- Prints: ["arg1", "arg2"]
}
```

### Example 2: Clock Operations

**Before (broken):**
```ailang
import std/clock (now)

export func main() -> () ! {Clock} {
  let ts = now();  -- ERROR: arity mismatch (builtin expects 0, got 1)
  ...
}
```

**After (fixed):**
```ailang
import std/clock (now)

export func main() -> () ! {Clock} {
  let ts = now();  -- ✓ Desugars to now(()), builtin expects unit, all good
  ...
}
```

### Example 3: Builtin Type Signature

**Before (broken):**
```
_env_getArgs : [string] ! {Env}  -- Wrong: looks like a value, not a function
```

**After (fixed):**
```
_env_getArgs : () -> [string] ! {Env}  -- Correct: single unit parameter
```

### Example 4: Higher-Order Functions (Still Work)

```ailang
-- Passing "zero-arg" function as value
let f = now  -- f has type (() -> int ! {Clock})

-- Calling through higher-order function
let callTwice[a](g: () -> a) -> (a, a) ! {Clock} = (g(), g())

let (t1, t2) = callTwice(now)  -- ✓ Both calls desugar to now(())
```

## Semantic Specification: "Nullary Functions" in AILANG

**Canonical rule:**

> AILANG does not have nullary functions at the semantic level.
>
> Surface syntax `func f() -> T` is sugar for `func f(_ : ()) -> T`.
> Surface syntax `f()` is sugar for `f(())` (application to unit value).

**Implications:**

1. **Type system**:
   - "Zero-arg function" types in core are always `() -> T`.
   - The parameter list contains exactly one element: the unit type.

2. **Elaboration**:
   - Parser recognizes `f()` as syntactic sugar.
   - Elaborator transforms to `App(f, UnitLit)`.
   - No arity-dependent logic; this is universal syntax sugar.

3. **Builtins**:
   - Every "zero-arg" builtin must be registered with `NumArgs: 1`.
   - Go implementation must expect one `eval.Value` (unit value).
   - Type builder must include unit in parameter list: `T.Func().Returns(...)`
   - **Validation invariant**: Builtin impl should validate the argument is unit (or safely ignore it).
     - Defense against type system bugs (e.g., annotation threading unsoundness).
     - If non-unit value is passed, panic with clear error: "internal invariant violation: expected unit".

4. **Stdlib**:
   - Wrapper functions use `f()` syntax (sugar).
   - Example: `export func now() -> int ! {Clock} = _clock_now()`

5. **Higher-order functions**:
   - "Zero-arg" functions can be passed as values: `let f = now`.
   - Type: `f : () -> int ! {Clock}`.
   - Calling: `f()` desugars to `f(())`.

6. **AI teaching**:
   - Teach: "Use `f()` to call functions with no parameters."
   - Don't teach: "Zero-arg functions are special."
   - Rationale: Reduces cognitive load; one rule covers all cases.
   - **Power-user note** (for teaching prompt):
     ```ailang
     -- These functions have type () -> T. You can pass them as values:
     let f = getArgs;      -- f : () -> [string] ! {Env}
     let xs = f();         -- calls getArgs(()), returns [string]

     -- Works with higher-order functions:
     let callTwice[a](g: () -> a) -> (a, a) = (g(), g())
     let (t1, t2) = callTwice(now)  -- both calls work
     ```

## Success Criteria

- [ ] `_env_getArgs()` returns `["arg1", "arg2"]` (not function object)
- [ ] `_clock_now()` returns Unix timestamp (no arity mismatch)
- [ ] `_io_readLine()` reads a line from stdin
- [ ] Stdlib wrappers work: `getArgs()`, `now()`, `readLine()`
- [ ] **Runtime invokes `main()` as `main(())` (or equivalent) and no entry module breaks**
- [ ] `cli_args` benchmark: 0% → 80%+ success rate
- [ ] All existing tests still pass (no regressions)
- [ ] Teaching prompt documents unit-argument model with examples
- [ ] Can pass "zero-arg" function as value: `let f = now` (type `() -> int ! {Clock}`)
- [ ] Golden snapshot shows `() -> T` types (not bare `T`)
- [ ] No new core AST nodes or arity-dependent elaboration
- [ ] S-CALL0 works in all listed contexts (expr, stmt, lambda, match)

## Testing Strategy

**Unit tests:**
- Builtins: Each "zero-arg" builtin accepts unit value, returns correct result
- S-CALL0: `f()` desugars to `f(())` in all contexts (expr, stmt, match, lambda)
- Type checker: Infers `() -> T` for "zero-arg" functions
- Higher-order: Can pass `now` as value, call via `g()`

**Integration tests:**
- `tests/cli_args_test.ail` - CLI arguments work end-to-end
- `examples/tests/micro_clock_measure.ail` - Clock operations work
- REPL: `now()` works interactively

**Manual testing:**
```bash
# Test 1: CLI args (the blocker)
ailang run --caps IO,Env --entry main tests/cli_args_test.ail arg1 arg2
# Expected: Prints "argc: 2", "arg: arg1", "arg: arg2"

# Test 2: Clock
ailang run --caps Clock examples/tests/micro_clock_measure.ail
# Expected: No errors, prints timestamps

# Test 3: REPL
ailang repl --caps Clock
> now()
# Expected: Returns int (Unix timestamp)

# Test 4: Stdlib wrappers
ailang repl --caps Env
> import std/env (getArgs)
> getArgs()
# Expected: Returns list of args
```

## Non-Goals

**Not in this feature:**
- ❌ True nullary functions (distinct semantic entity)
- ❌ New call syntax (e.g., `f!` operator)
- ❌ Auto-calling on reference (`f` → auto-call)
- ❌ Arity-dependent elaboration hacks
- ❌ Special "nullary call" core AST node

**Why not?**
- AILANG already chose the unit-argument model (S-CALL0)
- Adding true nullaries would require deep semantic surgery
- Current model is simpler, more consistent, and AI-friendly

## Timeline

**Day 1** (4 hours):
- Phase 1: Align builtins (change NumArgs, update impls)
- Update golden snapshots
- Unit tests for builtin changes

**Day 2** (4 hours):
- Phase 2: S-CALL0 regression tests
- Phase 3: Fix stdlib wrappers
- Integration tests

**Day 3** (2 hours):
- Phase 4: Teaching prompt updates
- Run benchmarks
- Verify no regressions
- Write canonical spec document

**Total: ~10 hours across 3 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Breaks existing code using "zero-arg" functions | High | Test all examples; expect breakage (dormant bugs) |
| **Entry runtime still assuming arity-0 `main`** | **High** | **Phase 1.5: Update runtime invocation explicitly** |
| S-CALL0 doesn't work in some contexts | Medium | Comprehensive regression tests; audit elaborator per context list |
| Golden snapshot changes cause test failures | Low | Regenerate with `UPDATE_GOLDEN=1` |
| AI models still confused by `f` vs `f()` | Medium | Clear teaching prompt; emphasize `f()` syntax |
| Type annotation threading interactions | Medium | Re-run concat inference & annotation tests after builtin changes |

## References

- **Triggered by**: M-LANG-CLI-ARGS implementation (v0.4.6)
- **Related features**:
  - S-CALL0 (partial implementation)
  - M-DX1 Builtin System (uses NumArgs)
- **Affected builtins**:
  - `internal/builtins/clock.go` - `_clock_now`
  - `internal/builtins/env.go` - `_env_getArgs`
  - `internal/builtins/io.go` - `_io_readLine`
- **Stdlib wrappers**:
  - `std/clock.ail` - `now()`
  - `std/env.ail` - `getArgs()`
  - `std/io.ail` - `readLine()`
- **Teaching materials**:
  - `prompts/v0.4.6.md` - Needs unit-argument model section
- **Type system**:
  - Hindley-Milner with row polymorphism
  - Unit type `()` as first-class value

## Cross-References to Other M-Docs

**Dependencies & Interactions:**

1. **M-BUG-CONCAT-INFERENCE** (String concatenation type inference)
   - **Impact**: Changing builtin types to `() -> T` adds more explicit function types
   - **Action**: Re-run concat inference tests after M-DX10 implementation
   - **Why**: Typechecker will see more `() -> T` patterns; verify no surprise interactions

2. **Type Annotation Threading** (Expected-type propagation)
   - **Impact**: Users will write explicit types like `() -> [string] ! {Env}` in code
   - **Action**: Ensure annotation threading doesn't silently ignore these types for builtins
   - **Why**: Runtime must not explode when annotations differ from actual types
   - **Critical**: This composes with M-DX10 - both touch typechecker paths

3. **M-LANG-CLI-ARGS** (CLI arguments feature)
   - **Blocks**: This feature is the primary blocker
   - **Unblocks**: After M-DX10, `getArgs()` will be usable from AILANG code
   - **Testing**: Use cli_args benchmark as success metric

**Suggested Implementation Order:**
1. M-DX10 (this doc) - Complete unit-argument model
2. M-LANG-CLI-ARGS - Unblocked by M-DX10
3. Type annotation threading - Builds on stable builtin types
4. M-BUG-CONCAT-INFERENCE - Can now assume `() -> T` is stable

**Note**: M-DX10 and annotation threading compose. Expect to touch both in same release.

## Future Work

**If true nullaries are ever needed** (defer to v0.5.0+):
- Introduce distinct nullary function kind at type level
- Add nullary call construct to core AST
- Define interaction with higher-order functions, partial application
- Update teaching prompt to explain both models

**For now**: The unit-argument model is sufficient, consistent, and AI-friendly.

---

## Appendix: Why the Unit-Argument Model?

**Advantages over true nullaries:**

1. **Simpler elaboration**: No arity-dependent transformations.
2. **Uniform function application**: `f x` covers all cases; `f ()` is just `f` applied to unit.
3. **ML tradition**: Follows OCaml/SML where `()` is the canonical "no meaningful value".
4. **Type system**: Unit is a proper type, not special syntax.
5. **Higher-order functions**: Can pass "zero-arg" functions as values without special cases.
6. **Already implemented**: S-CALL0 exists; this just completes it.

**Disadvantages:**
- Slightly verbose: `f()` instead of `f` (but sugar mitigates this).
- Not obvious from docs: Requires explicit teaching (this doc fixes that).

**Decision rationale**: AILANG prioritizes machine decidability and determinism. The unit-argument model is more compositional and requires less context-dependent reasoning.

---

**Document created**: 2025-11-18
**Last updated**: 2025-11-18 (revised with architectural feedback)
**Replaces**: Previous M-DX10 draft (semantic model changed from true nullaries to unit-argument)
**Discovered by**: M-LANG-CLI-ARGS sprint (Milestone 2)
**Blocks**: CLI arguments feature (v0.4.6)

**Key revisions:**
- Added Phase 1.5: Entry invocation handling (prevents repeat of S-CALL0 failure pattern)
- Expanded S-CALL0 contexts list with explicit inclusions/exclusions
- Added builtin validation invariant (defense against type system bugs)
- Enhanced teaching section with power-user HOF examples
- Added cross-references to type annotation threading and concat inference
- Updated risks to highlight entry runtime assumption
