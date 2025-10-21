# Example Parity & Vision Benchmark Alignment

**Status**: Planned
**Target**: v0.3.15
**Priority**: P0 (High) - Blocks AI teaching and vision benchmarks
**Estimated**: 2-3 days
**Dependencies**: None

## AI-First Alignment Check

**Score this feature against AILANG's core principles:**

| Principle | Impact | Score | Notes |
|-----------|--------|-------|-------|
| Reduce Syntactic Noise | ✅ Positive | **+1** | Removes `import std/io (println)` boilerplate (4 tokens) for entry modules |
| Preserve Semantic Clarity | ✅ Preserved | **0** | Effects remain explicit in type signatures (`! {IO}`), no hidden magic |
| Increase Determinism | ✅ Positive | **+1** | Prelude injection is deterministic per entry module (scan for `export func main`) |
| Lower Token Cost | ✅ Positive | **+1** | ~10 tokens saved per "hello world", enables more examples in teaching prompts |
| **Net Score** | | **+3** | **Decision: ✅ Move forward (strong alignment)** |

**Rationale:**
- Entry-module prelude is scoped (not global) → no namespace pollution
- Libraries remain explicit (must use `import std/io`) → preserves "import is intent"
- String-only `print` forces `show()` conversion → teaches canonical `print(show(x))` pattern
- First step in "minimize syntactic entropy" roadmap → clear future direction

**Reference:** See [AI-first DX philosophy](#-design-principle-ai-first-dx)

## Problem Statement

**45 out of 79 example files (57%) are currently failing**, preventing them from being used for:
- AI model training and teaching
- Vision benchmark demonstrations
- User documentation and tutorials
- Verification of language features

**Current State:**
- **Working**: 34 files (27 runnable + 7 snippets/tests)
- **Failing**: 45 files (57% failure rate)
  - 28 files: Module path mismatches (MOD010 errors)
  - 9 files: Missing `print` builtin
  - 3 files: Parse errors (syntax gaps)
  - 2 files: Type class syntax issues
  - 1 file: Effect context error
  - 4 files: Deprecated imports

**Impact:**
- **AI Model Training**: Cannot use simple examples in teaching prompts
- **Vision Benchmarks**: Blocked benchmarks for:
  - Referential Transparency (needs simple print examples)
  - Canonical Code Structure (needs working demonstrations)
  - Simple demonstrations for deterministic generation
- **User Experience**: "Hello World" doesn't work as a file (only in REPL)
- **Documentation**: 57% of examples can't be verified

## Goals

**Primary Goal:** Achieve 90%+ example pass rate (40+ of 45 failing examples fixed)

**Design Principle:** Follow AILANG's AI-first DX philosophy: **"Minimize syntactic entropy"** - teach the compiler to carry context so the AI doesn't have to. (See [Design Principle](#-design-principle-ai-first-dx) section)

**Success Metrics:**
- Pass rate: 34/79 (43%) → 74+/79 (93%+)
- Vision-aligned examples: 0% → 100% working
- Simple examples (hello.ail, etc.): 0% → 100% working
- Module path compliance: 64% → 100%
- Time to verify all examples: ~2 min → ~30 sec (batch script)
- Syntactic entropy: Reduced by removing unnecessary imports for entry modules

## Solution Design

### Overview

Fix failing examples in priority order based on vision benchmark impact:

**Priority 1 (Critical for Vision)**: Missing `print` builtin - blocks 9 files
**Priority 2 (Quick Wins)**: Module path fixes - blocks 28 files, 5-minute fix
**Priority 3 (Syntax Gaps)**: Parse errors - blocks 3 files, needs investigation
**Priority 4 (Future Work)**: Type class syntax - blocks 2 files, deferred to v0.4.0

### Architecture

**Key Design Decision: Entry-Module Prelude (Not Global)**

Instead of making `print` globally available (which would pollute library namespaces and weaken "import is intent"), we use an **entry-module prelude** that's auto-injected only for:
- Modules with an exported `main` function
- REPL environment
- Files run with `ailang run --entry main`

**Why this approach:**
- Zero friction for "hello world" and teaching examples
- Libraries remain explicit (must import `std/io` if they need I/O)
- Preserves AILANG's effect discipline
- No namespace pollution

**Components:**

1. **Entry-Module Prelude**: Auto-injected for entry modules only
   - Provides: `print : string -> () ! {IO}` (string-only, not polymorphic)
   - Implementation: Thin wrapper over `_io_println`
   - Scope: Only entry modules (main functions) and REPL
   - Libraries: Must use explicit `import std/io (println)` (correct!)

2. **Module Path Fixer**: Batch script to update module declarations
   - Pattern: Replace `examples/X` with `examples/SUBDIR/X`
   - Validation: Check all files match canonical paths

3. **Parse Error Fixes**: Update parser to handle edge cases
   - Match expressions as statements
   - Block expression trailing tokens

4. **Deprecated Import Fixer**: Update old `stdlib/std/*` to `std/*`

### Implementation Plan

**Phase 1: Quick Wins - Module Paths** (~1 hour)
- [x] Analyze all MOD010 errors (DONE - found 28 files)
- [ ] Create batch fix script for module declarations
  - [ ] Make idempotent (no change on already-correct files)
  - [ ] Print summary table: fixed/unchanged/skipped
  - [ ] Add `--dry-run` mode for safety
  - [ ] Precise regex: only touch module declarations matching error pattern
- [ ] Run script on all affected files
- [ ] Verify with `make verify-examples`
- [ ] Test: 28 additional files should pass

**Phase 2: Entry-Module Prelude ('print')** (~4 hours)
- [ ] Create `internal/pipeline/prelude.go` with `InjectPrelude(env)` function
- [ ] Register `print : string -> () ! {IO}` in builtin spec registry (uses `_io_println` with newline)
- [ ] Entry detection: Scan public env for exported `main` (arity 0), not flag-based
- [ ] REPL: Always inject prelude (REPL grants IO caps)
- [ ] Runner: Inject prelude before typecheck if entry module detected
- [ ] Libraries: Never inject (must use `import std/io`)
- [ ] Shadowing: Allow user definitions to shadow `print` (no warning, keep simple)
- [ ] Error message when `! {IO}` missing: "print performs IO. Add ! {IO} to main (e.g., export func main() -> () ! {IO}) or run with --caps IO."
- [ ] Debug logging: "prelude: injected symbols [print] for entry module <mod>"
- [ ] Update examples to use `print(show(...))` pattern
- [ ] Test: 9 additional files should pass
- [ ] Add prelude smoke tests (entry vs lib, effect discipline, REPL)

**Phase 3: Syntax Fixes - Parse Errors** (~3 hours)
- [ ] Investigate `func_expressions.ail` trailing `}`
- [ ] Investigate `list_patterns.ail` match-as-statement
- [ ] Investigate `test_m_r7_comprehensive.ail` nested syntax
- [ ] Fix parser edge cases OR update examples (minimal scope)
- [ ] Add 2 golden parser tests to cover these cases
- [ ] Test: 3 additional files should pass

**Phase 4: Housekeeping - Deprecated Imports** (~30 min)
- [ ] Create fixer script with precise regex: `^import\s+stdlib/std/`
- [ ] Add `--dry-run` mode
- [ ] Update 4 files: `stdlib/std/*` → `std/*`
- [ ] Test: 4 files should pass (may already pass after Phase 1)

**Phase 5: Documentation & CI Gating** (~2 hours)
- [ ] Add `make verify-examples` to CI with threshold gate (fail if pass rate < 90%)
- [ ] Print one-line summary: "Examples: 74/79 passed (93.7%)"
- [ ] Persist JSON artifact (counts + file names) for tracking deltas
- [ ] Update README.md with new pass rate
- [ ] Update teaching prompts (prompts/v0.3.15.md):
  - [ ] "How to print" page with 3 code blocks (entry module, REPL, library pattern)
  - [ ] Show error text for missing effect case
  - [ ] REPL example with `:type print`
- [ ] Update `benchmarks/simple_print.yml`:
  - [ ] Pure string literal case
  - [ ] Concatenation with `show` case
  - [ ] Failure case without IO effect (expect diagnostic)

### Files to Modify/Create

**New files:**
- `tools/fix_module_paths.sh` - Batch fix for MOD010 errors (~50 LOC)
- `tools/update_deprecated_imports.sh` - Fix stdlib imports (~30 LOC)
- `benchmarks/simple_print.yml` - Test print builtin works (~20 LOC)
- `internal/pipeline/prelude.go` - Entry-module prelude injection (~100 LOC)
- `internal/pipeline/prelude_test.go` - Prelude tests (~80 LOC)

**Modified files:**
- `internal/builtins/io.go` - Add `print : string -> () ! {IO}` (~30 LOC)
- `internal/builtins/spec.go` - Register `print` in spec registry (~15 LOC)
- `internal/repl/repl.go` - Use NewTypeEnvWithPrelude() (~10 LOC)
- `internal/runtime/runtime.go` - Entry module detection + InjectPrelude (~30 LOC)
- `internal/parser/expressions.go` - Fix match-as-statement (~10 LOC)
- `examples/snippets/*.ail` - Update 9 files to use `print(show(...))` (~5 LOC each)
- `examples/tests/*.ail` - Update 28 module declarations (~1 LOC each)
- `examples/snippets/*.ail` - Update 4 deprecated imports (~1 LOC each)
- `README.md` - Update pass rate statistics (~5 LOC)
- `prompts/v0.3.15.md` - Document `print` builtin and entry-module prelude (~80 LOC)

**Total new code: ~430 LOC**
**Total modifications: ~130 LOC across 40+ files**

## Examples

### Example 1: Simple Hello World (Entry Module)

**Before:**
```ailang
-- examples/snippets/hello.ail
-- Error: undefined variable: print

print("Hello, AILANG!")
```

**After:**
```ailang
-- examples/snippets/hello.ail
-- Works! Entry-module prelude provides print

module examples/snippets/hello

export func main() -> () ! {IO} {
  print("Hello, AILANG!")  -- Prelude provides this for entry modules
}
```

**Key points:**
- `print` is ONLY available in entry modules (modules with `export func main`)
- Type is `string -> () ! {IO}` (string-only, not polymorphic)
- Effect `! {IO}` must be explicit in main's signature
- Run with: `ailang run --caps IO --entry main hello.ail`

### Example 2: Module Path Fix (Currently Broken)

**Before:**
```ailang
-- examples/tests/test_effect_io.ail
-- Error: MOD010: module declaration 'examples/test_effect_io'
--        doesn't match canonical path 'examples/tests/test_effect_io'

module examples/test_effect_io
```

**After:**
```ailang
-- examples/tests/test_effect_io.ail
-- Fixed!

module examples/tests/test_effect_io
```

### Example 3: Print with Show (Recommended Pattern)

**Before (Workaround):**
```ailang
module examples/snippets/arithmetic
import std/io (println)

export func main() -> () ! {IO} {
  println("Result: " ++ show(2 + 2))
}
```

**After (Entry-Module Prelude):**
```ailang
module examples/snippets/arithmetic

export func main() -> () ! {IO} {
  print("Result: " ++ show(2 + 2))  -- print takes string, show converts
}
```

**Why string-only?**
- No type class constraints needed (keeps types honest)
- Forces explicit `show()` conversion (teaches the right pattern)
- Users (and AIs) learn: `print(show(value))` is the canonical form

### Example 4: Library Module (No Prelude!)

```ailang
-- mylibrary/utils.ail
-- This is a library, NOT an entry module (no main function)

module mylibrary/utils

export func debug(msg: string) -> () ! {IO} {
  print(msg)  -- ❌ ERROR: undefined variable: print
}

-- ✅ CORRECT: Libraries must be explicit about dependencies
import std/io (println)

export func debug(msg: string) -> () ! {IO} {
  println(msg)  -- Works! Library explicitly imports std/io
}
```

**This is intentional!** Libraries should declare their dependencies explicitly. Only entry modules get the prelude.

## Success Criteria

- [ ] **Pass rate ≥90%**: At least 71/79 examples working (currently 34/79 = 43%)
- [ ] **All vision examples working**: hello.ail, arithmetic.ail, showcase/* (9 files)
- [ ] **Module path compliance**: 0 MOD010 errors (currently 28)
- [ ] **Parse errors fixed**: func_expressions.ail, list_patterns.ail work
- [ ] **`print` builtin available**: Works in modules and REPL with IO effect
- [ ] **No deprecated imports**: All use `std/*` not `stdlib/std/*`
- [ ] **All tests passing**: `make test && make verify-examples`
- [ ] **Documentation updated**: Teaching prompt includes `print` usage
- [ ] **CI verification**: Automated check for example pass rate

## Testing Strategy

**Prelude smoke tests** (lock the behavior):
1. **Entry vs library:**
   - Entry module with `print("hi")` compiles and runs
   - Library using `print` → "undefined variable: print"
2. **Effect discipline:**
   - Entry main without `! {IO}` using `print` → deterministic error with guidance
3. **REPL behavior:**
   - `:type print` shows `string -> () ! {IO}`
   - `print("ok")` prints in REPL
4. **Shadowing:**
   - User can define own `print` (no error, no warning)

**Unit tests:**
- `internal/pipeline/prelude_test.go`: Test InjectPrelude logic
- `internal/builtins/io_test.go`: Test `print` builtin with mock EffContext
- `internal/parser/expressions_test.go`: Golden tests for match-as-statement, trailing tokens

**Integration tests:**
- Run all 79 examples with `make verify-examples`
- Check pass rate ≥90% (CI gate)
- Verify vision-aligned examples work
- JSON artifact tracking: counts + file names for delta analysis

**Manual testing:**
```bash
# Test print builtin (entry module)
echo 'module test export func main() -> () ! {IO} { print("Hello!") }' > /tmp/test.ail
ailang run --caps IO --entry main /tmp/test.ail
# Should output: Hello!

# Test effect error (missing ! {IO})
echo 'module test export func main() -> () { print("Hi") }' > /tmp/test.ail
ailang run --entry main /tmp/test.ail
# Should error: "print performs IO. Add ! {IO} to main..."

# Test in REPL
ailang repl
> :type print
string -> () ! {IO}
> print("Test")
Test
> print("Result: " ++ show(42))
Result: 42

# Test library (no prelude)
echo 'module lib func helper() -> () { print("x") }' > /tmp/lib.ail
ailang run /tmp/lib.ail
# Should error: "undefined variable: print"

# Verify all examples with summary
make verify-examples
# Should show: Examples: 74/79 passed (93.7%)
```

**Vision benchmark validation:**
```bash
# Run vision-aligned benchmarks
ailang eval-suite --benchmarks simple_print,referential_transparency,canonical_normalization --models gpt5-mini
# All should pass
```

## Non-Goals

**Not in this feature:**
- **Polymorphic print** (`forall a. a -> ()`) - Would require type class constraints we don't have
  - String-only is intentional: forces `print(show(x))` pattern
  - Teaches canonical form for AI models
  - Keeps types honest without runtime reflection
- **Global print in all modules** - Would pollute library namespaces
  - Entry-module prelude only (modules with `main`)
  - Libraries must use explicit `import std/io`
- **User-defined type classes** (show_Int syntax) - Deferred to v0.4.0 reflection milestone
- **Concurrency examples** (experimental/*.ail) - Requires v0.4.0 CSP channels
- **Auto-import of std/io** - Keep imports explicit for libraries
- **Breaking changes to effect system** - Use existing IO effect, no new effects

## Timeline

**Day 1** (6 hours):
- Phase 1: Module path fixes (1h)
- Phase 2: Global `print` builtin (4h)
- Verify: 37+ examples passing (34 → 71+)

**Day 2** (4 hours):
- Phase 3: Parse error fixes (3h)
- Phase 4: Deprecated imports (0.5h)
- Verify: 74+ examples passing

**Day 3** (2 hours):
- Phase 5: Documentation (2h)
- Final testing and CI setup
- Release prep

**Total: ~12 hours across 3 days**

## 🧭 Design Principle: AI-First DX

### Core Philosophy

**"AI-first DX = minimize syntactic entropy."**

Every syntactic token that isn't essential semantics should be optional, inferred, or injected deterministically.

**Why this matters:**
- **Lower token cost** - AIs use fewer tokens for the same program
- **Higher signal density** - More semantic content per token
- **Better determinism** - Less syntactic variation → more canonical forms

### Manifestations in AILANG

The entry-module prelude, auto-capability inference (`! {IO}`), and canonical imports are all manifestations of the same philosophy:

**Teach the compiler to carry context so the AI doesn't have to.**

| Feature | Syntactic Burden Removed | Semantic Clarity Preserved |
|---------|-------------------------|---------------------------|
| Entry-module prelude | No `import std/io` for hello world | `! {IO}` still explicit in type |
| Auto-cap inference (future) | No manual `! {IO}` annotation | Effects still tracked in types |
| Canonical imports | No `stdlib/std/*` vs `std/*` choice | Dependencies still explicit |
| `print(show(x))` pattern | No polymorphic constraints | Type conversion explicit |

### Next Logical Steps

**After `print`, extend the prelude with:**

1. **`assert : bool -> ()`** (pure)
   - Testing and invariant checking
   - No effect (crashes on false, doesn't return)

2. **`show : forall a. string`** (already pure, already global)
   - Already available, just ensure it's in prelude docs

3. **`panic : string -> never ! {IO}`** (optional)
   - Explicit failure with message
   - Alternative to assert for debugging

**Auto-capability inference (v0.3.16+):**
- Infer `! {IO}` from `print` usage
- See: [design_docs/planned/20251013_auto_caps_capability_inference.md](../../20251013_auto_caps_capability_inference.md)

### Why Entry-Module Prelude Fits This Philosophy

**Before (high syntactic entropy):**
```ailang
module examples/hello
import std/io (println)  -- 4 tokens of boilerplate

export func main() -> () ! {IO} {
  println("Hello")
}
```

**After (minimal syntactic entropy):**
```ailang
module examples/hello

export func main() -> () ! {IO} {  -- Effect still explicit
  print("Hello")  -- Prelude provides this
}
```

**Future (with auto-cap inference in v0.3.16+):**
```ailang
module examples/hello

export func main() {  -- Effect inferred from print usage
  print("Hello")
}
```

**Semantic content preserved, syntactic noise eliminated.**

---

## Design Rationale: Why Entry-Module Prelude?

### Problem: Tension Between Simplicity and Discipline

**Teaching/examples need:**
- Zero-friction "hello world"
- Simple demonstrations for AI training
- No boilerplate for first-touch experience

**AILANG's design principles require:**
- Explicit effects (`! {IO}` must be declared)
- Explicit imports ("import is intent")
- No hidden magic or silent effects

### Solutions Considered

| Approach | Pros | Cons | Verdict |
|----------|------|------|---------|
| **Global `print`** | Simplest for users | Pollutes all namespaces, weakens "import is intent" | ❌ Rejected |
| **Auto-import std/io** | Explicit in code | Still requires understanding modules for hello world | ❌ Too complex |
| **REPL-only print** | No file changes needed | Examples don't work as files, only interactive | ❌ Inconsistent |
| **Entry-module prelude** | Zero-friction for entry points, libraries stay explicit | Requires entry detection logic | ✅ **Chosen** |

### Why Entry-Module Prelude Wins

1. **Preserves Effect Discipline**
   - `print` requires `! {IO}` in main's signature (type error if missing)
   - No silent effects, all I/O is explicit in types

2. **Preserves Import Intent**
   - Libraries must use `import std/io (println)` (correct!)
   - Entry modules get prelude (teaching/examples/scripts)
   - Clear separation: "scripts" vs "libraries"

3. **Zero Friction for Teaching**
   - "Hello world" just works (with proper effect annotation)
   - AI models learn canonical `print(show(x))` pattern
   - Vision benchmarks unblocked

4. **String-Only Type**
   - Forces `show()` conversion (teaches good habits)
   - No type class constraints needed
   - Keeps types honest without runtime reflection

### Registry Wiring (Avoid Dual Registration)

**Critical:** Register `print` once in the spec registry, source everywhere from there.

**Implementation:**
- Register in `internal/builtins/io.go` using `RegisterEffectBuiltin()`
- Runner/REPL source from spec registry (not legacy path)
- If legacy eval still exists, make it delegate to registry for this symbol
- No dual registration = no drift between REPL and runner behavior

**Why this matters:**
- M-DX1 already established single-source-of-truth registry
- Avoids "print works in REPL but not in files" bugs
- Maintains consistency with all other builtins

### Entry Module Detection

**Entry module = module with `export func main` (arity 0)**

Detection logic:
1. Scan public environment after module load
2. Check for exported function named `main`
3. Verify arity is 0 (no parameters)
4. If match: inject prelude before typecheck
5. If no match: library module, no prelude

```ailang
-- Entry module (gets prelude)
module examples/hello
export func main() -> () ! {IO} {
  print("Hi")  -- ✅ Available
}

-- Library module (no prelude)
module mylib/utils
func helper() -> string {
  print("Debug")  -- ❌ ERROR: undefined variable
}
```

### Helpful Error Messages

**Case 1: Forgot effect annotation**
```ailang
export func main() -> () {  -- Missing ! {IO}
  print("Hello")
}
```
**Error:**
```
Type error: print performs IO. Add ! {IO} to main (e.g., export func main() -> () ! {IO}) or run with --caps IO.
```

**Case 2: Used print in library**
```ailang
module mylib/utils
func debug(msg: string) -> () {
  print(msg)  -- Not available in libraries
}
```
**Error:**
```
Error: undefined variable: print
  Note: print is only available in entry modules (modules with 'export func main')
  Suggestion: Import std/io for library I/O:
    import std/io (println)
    func debug(msg: string) -> () ! {IO} {
      println(msg)
    }
```

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `print` conflicts with user code | Low | Entry-module scope only, document as prelude |
| Entry detection logic complexity | Medium | Simple check: module has `export func main` |
| Confusion: why no print in libraries? | Medium | Clear error message + documentation |
| Parse fixes break other syntax | High | Extensive parser tests, regression suite |
| Example fixes introduce new errors | Medium | Run full test suite after each change |
| Module path regex too broad | Low | Manual review of all changes, git diff check |
| Vision benchmarks still fail | High | Test benchmarks after each phase, iterate |

## Vision Benchmark Alignment

**From VISION_BENCHMARKS.md analysis:**

### Blocked Benchmarks (Currently)

1. **Referential Transparency** (line 126)
   - Needs: Simple examples showing same input → same output
   - Blocked by: Missing `print` for demonstrations
   - **Fixed by**: Phase 2 (global `print` builtin)

2. **Canonical Code Structure** (line 149)
   - Needs: Working examples of functional composition
   - Blocked by: Parse errors, missing print
   - **Fixed by**: Phase 2 + Phase 3

3. **Simple Print Benchmark** (simple_print.yml)
   - Needs: Basic "hello world" to work
   - Blocked by: Missing `print` builtin
   - **Fixed by**: Phase 2

### Enabled After This Feature

- ✅ All 9 showcase examples work (teaching AI models)
- ✅ hello.ail works (first example users see)
- ✅ arithmetic.ail works (type inference demo)
- ✅ Vision benchmarks have working examples
- ✅ Can generate training data from examples

## References

- [VISION_BENCHMARKS.md](../../../benchmarks/VISION_BENCHMARKS.md) - Vision goals and metrics
- [Module System Design](../../../design_docs/implemented/v0_3_0/module_system.md) - Module path rules
- [Effect System](../../../design_docs/implemented/v0_2_0/effect_system.md) - IO effect semantics
- [M-DX1 Builtin Development](../../../design_docs/implemented/v0_3_10/M-DX1_developer_experience.md) - Builtin registry
- [Teaching Prompts](../../../prompts/v0.3.8.md) - Current AI teaching prompt

## Optional Enhancements (If Time Allows)

**Nice-to-have additions that don't block ship:**

1. **`printRaw : string -> () ! {IO}`** (no newline)
   - Complements `print` for cases where newline is unwanted
   - Document-only (mention in teaching prompt, don't prioritize)

2. **REPL `:print` special form**
   - Accepts any value, automatically calls `show()`
   - Example: `:print 42` → internally runs `print(show(42))`
   - Purely for REPL exploration convenience

3. **Enhanced `simple_print.yml` benchmark**
   - ✅ Already in plan: pure literal, show concat, effect error
   - Optional: Add shadowing case (user defines own `print`)

4. **Debug telemetry**
   - ✅ Already in plan: Log "prelude: injected symbols [print] for entry module"
   - Optional: Add to `ailang --version` output which builtins are in prelude

## Future Work

**Extending the Entry-Module Prelude (v0.3.16)**
Following the "minimize syntactic entropy" principle:

1. **`assert : bool -> ()`** - Pure invariant checking
   - Example: `assert(x > 0)`
   - Crashes on false, no return value
   - Useful for tests and preconditions

2. **`panic : string -> never ! {IO}`** - Explicit failure with message
   - Example: `panic("Unreachable code")`
   - Alternative to assert for debugging
   - Effect: `! {IO}` for error reporting

3. **Auto-capability inference** - Infer `! {IO}` from usage
   - See: [design_docs/planned/20251013_auto_caps_capability_inference.md](../../20251013_auto_caps_capability_inference.md)
   - Example: `export func main() { print("Hi") }` infers `! {IO}`
   - Reduces syntactic entropy while preserving type safety

**Type Class Examples (v0.4.0+)**
- Requires structural reflection milestone
- Will enable user-defined `show_Int` syntax
- 2 examples currently blocked: records.ail, typeclasses.ail

**Concurrency Examples (v0.4.0+)**
- Requires CSP channels and session types
- experimental/*.ail examples will work
- AI agent integration examples

---

**Document created**: 2025-10-21
**Last updated**: 2025-10-21 (Refinements incorporated)
