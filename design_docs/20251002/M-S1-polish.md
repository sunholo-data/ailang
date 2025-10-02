# M-S1 Polish: Ship v0.1.0 in 72 Hours

## Executive Summary

**Goal**: Complete M-S1 stdlib implementation and polish AILANG for v0.1.0 release.

**Status**: M-S1 substantially complete (4/5 stdlib modules working). Two blockers remaining:
- `list.ail`: `++` operator type error
- `io.ail`: Stubbed (export let syntax not supported)

**Timeline**: 72 hours (2-3 days)
- **Today (4-5h)**: Close M-S1 completely + lock API surface
- **Tomorrow AM (2-3h)**: Create 3 ship-quality demos
- **Tomorrow PM (2-3h)**: Polish error messages + documentation

---

## Phase 1: Close M-S1 Completely (Today, 2-3h)

### A. Fix list.ail ++ operator (60-90 min)

**Problem**: Type unification fails for list concatenation
**Solution**: Add dedicated typing rule in type checker

**Typing Rule**: `xs : [α] ∧ ys : [α] ⇒ xs ++ ys : [α]`

**Implementation (pseudo-code)**:
```go
case OpListConcat:
  α := freshTVar()
  expect xs : List(α)
  expect ys : List(α)
  result = List(α)
```

**Error Message**:
```
LIST_CONCAT_MISMATCH: cannot concat lists with different element types:
  left is [${xsElem}], right is [${ysElem}]
Hint: ensure both sides have the same element type; try inserting an
      explicit 'map toString' or 'map fromString'
```

**Tests** (4 regression tests):
1. **Empty**: `[] ++ [] : [a]` (polymorphic empty)
2. **Homogeneous**: `[1] ++ [2,3] : [int]`
3. **Error**: `[1] ++ ["x"]` → type error
4. **Polymorphic**: `let append xs ys = xs ++ ys` infers `[a] -> [a] -> [a]`

**Additional generalization tests**:
- `let k = [] ++ []` infers `[a]` (polymorphic empty)
- Value restriction interaction

**Files to modify**:
- `internal/types/typechecker.go`: Add OpListConcat case
- `internal/types/errors.go`: Add LIST_CONCAT_MISMATCH error
- `internal/types/typechecker_test.go`: Add 4+ test cases

**Exit criteria**: All list concat tests pass, `stdlib/std/list.ail` type-checks

---

### B. Unblock io.ail (45-60 min)

**Strategy**: Choose **Option 2 first** (equation-form exports) for speed

**Rationale**:
- Faster to ship (minimal parser change)
- Simpler docs ("thin wrappers are just functions that call builtins")
- Save `extern` keyword for v0.2.0 when formalizing FFI semantics

**Option 2: Equation-form exports** (CHOSEN)
```ailang
export func println(s: string) -> () ! {IO} = _io_println(s)
export func print(s: string) -> () ! {IO} = _io_print(s)
export func readLine() -> string ! {IO} = _io_readLine()
export func debug(s: string) -> () ! {IO} = _io_debug(s)
```

**Parser change**:
- In `parseFuncDecl()`, allow `export func f(sig) = <expr>` in addition to block body
- Desugar to single-expression body node

**Implementation**:
```go
// In parseFuncDecl, after parsing signature:
if p.peekTokenIs(lexer.ASSIGN) {
    p.nextToken() // consume '='
    body := p.parseExpression(LOWEST)
    funcDecl.Body = &ast.Block{Exprs: []ast.Expr{body}}
}
```

**Option 1: extern keyword** (DEFERRED to Phase 5 stretch)
```ailang
extern func println(s: string) -> () ! {IO}
extern func print(s: string) -> () ! {IO}
```
- Lexer: Add EXTERN token
- Parser: Allow `extern func name(sig)` (no body required)
- Linker: Bind `extern` names to `_io_*` builtins

**Files to modify**:
- `internal/parser/parser.go`: Add equation-form support to parseFuncDecl
- `stdlib/std/io.ail`: Use equation-form syntax

**Exit criteria**: `stdlib/std/io.ail` type-checks with 4 exported functions

---

### C. Verify stdlib completion (15 min)

**Tasks**:
1. Run `ailang check stdlib/std/{option,result,string,list,io}.ail`
2. Verify all 5 modules type-check without errors
3. Run existing examples: `option_demo.ail`, `block_demo.ail`, `stdlib_demo.ail`
4. All 3 examples should execute successfully

**Exit criteria**:
- ✅ All 5 stdlib modules type-check
- ✅ 3/3 examples run successfully
- ✅ **M-S1 COMPLETE**

---

## Phase 2: Lock API Surface (Today, 1-2h)

### A. Stdlib interface freeze (60 min)

**Goal**: Prevent accidental API breakage with SHA256 golden files

**Interface JSON format** (normalized):
```json
{
  "module": "std/list",
  "types": [
    {"name": "Option", "params": ["a"], "ctors": ["Some", "None"]}
  ],
  "funcs": [
    {"name": "map", "type": "(a->b,[a])->[b]", "effects": []},
    {"name": "println", "type": "(string)->()", "effects": ["IO"]}
  ]
}
```

**Normalization rules**:
- Sort all JSON keys
- Sort exports by name
- Canonicalize type variables to a, b, c, ...
- Sort effect rows alphabetically
- Hash with `sha256sum`

**Makefile target**:
```make
STDLIBS := std/option std/result std/string std/list std/io

.PHONY: test-stdlib-freeze
test-stdlib-freeze:
	@for m in $(STDLIBS); do \
	  ailang iface $$m > .iface.json; \
	  jq -S . .iface.json > .iface.norm.json; \
	  sha256sum .iface.norm.json | awk '{print $$1}' > .hash; \
	  diff -q .hash goldens/stdlib/$$(echo $$m | tr '/' '__').sha256 || \
	    (echo "API drift in $$m"; exit 1); \
	done
```

**Files to create**:
- `Makefile`: Add `test-stdlib-freeze` target
- `goldens/stdlib/std__option.sha256`
- `goldens/stdlib/std__result.sha256`
- `goldens/stdlib/std__string.sha256`
- `goldens/stdlib/std__list.sha256`
- `goldens/stdlib/std__io.sha256`

**CLI command** (new):
```bash
ailang iface <module>  # Output normalized JSON interface
```

**Exit criteria**: `make test-stdlib-freeze` passes, CI enforces API stability

---

### B. Example golden files (45 min)

**Goal**: Verify examples produce consistent output with golden file comparison

**verify-examples.sh contract**:
- Runs each `*.ail` → captures stdout only
- Compares to `examples/<name>.golden`
- Emits unified diff on mismatch
- Honors `AILANG_SEED=42`, `TZ=UTC`, `LANG=C` for determinism

**Makefile target**:
```make
.PHONY: verify-examples
verify-examples:
	@TZ=UTC LANG=C AILANG_SEED=42 \
	./scripts/verify-examples.sh examples
```

**Script implementation** (`scripts/verify-examples.sh`):
```bash
#!/usr/bin/env bash
set -euo pipefail

EXAMPLES_DIR="${1:-examples}"
FAILED=0

for ail_file in "$EXAMPLES_DIR"/*.ail; do
    name=$(basename "$ail_file" .ail)
    golden="$EXAMPLES_DIR/$name.golden"

    if [[ ! -f "$golden" ]]; then
        echo "SKIP: $name (no golden file)"
        continue
    fi

    echo "Testing: $name"

    # Run with deterministic settings
    actual=$(ailang run "$ail_file" 2>&1 || true)

    if diff -u "$golden" <(echo "$actual"); then
        echo "  ✓ $name"
    else
        echo "  ✗ $name (output mismatch)"
        FAILED=$((FAILED + 1))
    fi
done

if [[ $FAILED -gt 0 ]]; then
    echo "Failed: $FAILED examples"
    exit 1
fi
```

**Golden files to create**:
- `examples/option_demo.golden`
- `examples/block_demo.golden`
- `examples/stdlib_demo.golden`
- Plus 2-3 new demos (from Phase 3)

**Exit criteria**: `make verify-examples` passes, CI enforces output stability

---

## Phase 3: Ship-Quality Demos (Tomorrow AM, 2-3h)

### Goal: Create 3 polished demos (≤30 LOC each) showing pure vs effect contrast

### 1. effects_pure.ail - Pure data transformation

**Theme**: No IO, pure functional programming
**Features**: list operations, option handling, pattern matching
**Output**: Integer result (e.g., "Sum: 42")

```ailang
-- Pure functional pipeline: map/filter/fold with no effects
import std/list (map, filter, fold)
import std/option (Option, Some, None, getOrElse)

-- Sum of even numbers doubled
let numbers = [1, 2, 3, 4, 5, 6]
let evens = filter(\x. x % 2 == 0, numbers)
let doubled = map(\x. x * 2, evens)
let sum = fold(\acc x. acc + x, 0, doubled)

sum  -- Output: 24 (no ! {} annotation)
```

**Golden file** (`examples/effects_pure.golden`):
```
24
```

---

### 2. io_hello.ail - IO effects demonstration

**Theme**: Interactive IO with explicit effect tracking
**Features**: readLine, println, effect rows in signatures
**Output**: Echoed input (e.g., "Hello, AILANG!")

```ailang
-- Interactive echo: demonstrates ! {IO} effect tracking
import std/io (readLine, println)

func greet() -> () ! {IO} {
  let name = readLine()
  let greeting = "Hello, " ++ name ++ "!"
  println(greeting)
}

greet()  -- Function signature shows ! {IO}
```

**Golden file** (`examples/io_hello.golden`):
```
Hello, AILANG!
```

**Input** (provided via STDIN in test):
```
AILANG
```

---

### 3. adt_pipeline.ail - Result plumbing with ADTs

**Theme**: Error handling with Result type
**Features**: ADT pattern matching, map/flatMap, error propagation
**Output**: Success or error message

```ailang
-- Result pipeline: demonstrates ADT + pattern matching + error handling
import std/result (Result, Ok, Err, map, flatMap)

type ParseError = InvalidFormat(string) | OutOfRange(int)

func parsePositive(s: string) -> Result[int, ParseError] {
  match parseInt(s) {
    Some(n) if n > 0 => Ok(n),
    Some(n) => Err(OutOfRange(n)),
    None => Err(InvalidFormat(s))
  }
}

func double(n: int) -> Result[int, ParseError] {
  Ok(n * 2)
}

let result = flatMap(double, parsePositive("21"))

match result {
  Ok(n) => "Success: " ++ toString(n),
  Err(InvalidFormat(s)) => "Parse error: " ++ s,
  Err(OutOfRange(n)) => "Must be positive, got: " ++ toString(n)
}
```

**Golden file** (`examples/adt_pipeline.golden`):
```
Success: 42
```

---

### Acceptance criteria for demos:

- ✅ Each demo ≤30 LOC
- ✅ Clear one-line README comment at top
- ✅ `.golden` file with expected output
- ✅ Demonstrates specific feature set:
  - `effects_pure.ail`: Pure functions, no effects
  - `io_hello.ail`: IO effects with `! {IO}` signatures
  - `adt_pipeline.ail`: ADT pattern matching + Result plumbing
- ✅ All pass `make verify-examples`

---

## Phase 4: Polish (Tomorrow PM, 2-3h)

### A. Error messages with actionable hints (60 min)

**1. LIST_CONCAT_MISMATCH** (enhanced):
```
Error: LIST_CONCAT_MISMATCH at examples/bad_concat.ail:5:10

cannot concat lists with different element types:
  left is [Int], right is [String]

Hint: ensure both sides have the same element type; try inserting an
      explicit 'map toString' or 'map fromString'

    let mixed = [1, 2] ++ ["a", "b"]
                       ^^
```

**2. IMPORT_CONSTRUCTOR_NOT_EXPORTED** (enhanced):
```
Error: IMPORT_CONSTRUCTOR_NOT_EXPORTED at examples/bad_import.ail:1:24

Constructor 'Some' not exported from module std/option

Available constructors: Some, None
Try: import std/option (Option, Some, None)

    import std/option (Option)
                       ^^^^^^^
```

**Files to modify**:
- `internal/types/errors.go`: Enhance error messages with hints
- `internal/loader/errors.go`: Add "Try:" suggestions to import errors

---

### B. Documentation updates (90 min)

**1. README.md updates**:

**"What's in v0.1.0" section**:
```markdown
## What's in v0.1.0

✅ **Type System**
- Hindley-Milner inference with let-polymorphism
- Type classes (Num, Eq, Ord, Show) with dictionary-passing
- Row-polymorphic records
- ADT support: `type Option[a] = Some(a) | None`

✅ **Effect System** (Type-level tracking)
- Effect annotations: `func f() -> int ! {IO, FS}`
- 8 canonical effects (IO, FS, Net, Clock, Rand, DB, Trace, Async)
- Effect propagation and inference

✅ **Pattern Matching**
- Literal, tuple, constructor, list patterns
- Works in functions and top-level expressions
- Guards (parsed, evaluation TODO)

✅ **Standard Library** (5 modules in AILANG)
- `std/option`: Option[a], map, flatMap, getOrElse, isSome, isNone, filter
- `std/result`: Result[a,e], map, mapErr, flatMap, isOk, isErr, unwrap
- `std/string`: length, substring, toUpper, toLower, trim, compare, find
- `std/list`: map, filter, fold, length, head, tail (with ++ operator)
- `std/io`: print, println, readLine, debug (all with ! {IO})

✅ **Module System**
- Path resolution (relative, stdlib, project)
- Type/constructor imports: `import std/option (Option, Some, None)`
- Cross-module pattern matching
```

**"Known Limitations" section**:
```markdown
## Known Limitations (v0.1.0)

⚠️ **Deferred to v0.2.0+**:
- `export let` syntax (currently use `export func`)
- `extern` keyword for FFI (currently use equation-form exports)
- Exhaustiveness checking for pattern matches
- Guard evaluation in patterns
- Runtime effect enforcement (type-level only in v0.1.0)
- Capability budgets
- Refinement types

⚠️ **Parser Edge Cases**:
- Some complex nested patterns may fail
- Test/property syntax not implemented

⚠️ **Coverage**:
- Parser: 75.1% (gap is unimplemented features)
- Overall: ~25% (focus on core functionality)
```

**2. Stdlib reference tables**:

**std/list**:
| Name   | Type                  | Effects |
|--------|-----------------------|---------|
| map    | `(a->b,[a])->[b]`    | []      |
| filter | `(a->bool,[a])->[a]` | []      |
| fold   | `(b->a->b,b,[a])->b` | []      |
| head   | `[a]->Option[a]`     | []      |
| tail   | `[a]->[a]`           | []      |
| length | `[a]->int`           | []      |
| ++     | `[a]->[a]->[a]`      | []      |

**std/io**:
| Name     | Type             | Effects |
|----------|------------------|---------|
| println  | `(string)->()`   | IO      |
| print    | `(string)->()`   | IO      |
| readLine | `()->string`     | IO      |
| debug    | `(string)->()`   | IO      |

**std/option**:
| Name      | Type                        | Effects |
|-----------|-----------------------------|---------|
| map       | `(a->b,Option[a])->Option[b]` | []   |
| flatMap   | `(a->Option[b],Option[a])->Option[b]` | [] |
| getOrElse | `(Option[a],a)->a`         | []      |
| isSome    | `Option[a]->bool`          | []      |
| isNone    | `Option[a]->bool`          | []      |
| filter    | `(a->bool,Option[a])->Option[a]` | [] |

**3. CHANGELOG.md entry**:
```markdown
## [v0.1.0] - 2025-10-03

### Added
- **Effect System** (type-level): Effect annotations `! {IO, FS}` with propagation
- **Standard Library** (5 modules in AILANG):
  - `std/option`, `std/result`, `std/string`, `std/list`, `std/io`
- **Pattern Matching**: Full support in functions and top-level
- **ADT Support**: Sum types, product types, recursive types
- **Module System**: Cross-module type/constructor imports
- **Golden File Testing**: API freeze + example verification

### Fixed
- List concatenation (`++`) typing rule with polymorphic support
- IO functions with equation-form export syntax
- Module + blocks normalization bug
- Cross-module constructor resolution

### Changed
- Parser coverage: 73.4% → 75.1% (+13 test cases)
- Type system fully migrated to TFunc2/TVar2

### Known Limitations
- Effect tracking is type-level only (runtime enforcement in v0.2.0)
- `export let` and `extern` syntax deferred to v0.2.0
- Exhaustiveness checking not implemented
- Test coverage at 25% (focused on core functionality)

### Metrics
- Total LOC: ~24,000
- Test coverage: ~25%
- Examples passing: 35+ (up from 22)
- Stdlib modules: 5 (all type-check)
```

---

## Phase 5: Stretch Goals (If time remains)

### A. extern keyword implementation (60 min)

**Goal**: Formalize FFI syntax for v0.2.0

**Implementation**:
1. **Lexer**: Add `EXTERN` token
2. **Parser**: Allow `extern func name(sig)` with no body
3. **Linker**: Bind `extern` names to builtin implementations
4. **Dual syntax**: Support BOTH extern and equation-form in std/io

**Example**:
```ailang
-- Option 1: extern (v0.2.0 preferred)
extern func println(s: string) -> () ! {IO}

-- Option 2: equation-form (v0.1.0 current)
export func println(s: string) -> () ! {IO} = _io_println(s)
```

**Files to modify**:
- `internal/lexer/token.go`: Add EXTERN token
- `internal/parser/parser.go`: Add parseExternDecl
- `internal/loader/linker.go`: Bind extern to builtins

---

### B. Parser coverage add-ons (30 min)

**Tests to add**:
1. Equation-form function parsing
2. List concat negative tests (type errors)
3. Import constructor error cases

**Target**: +1-2% coverage (75.1% → 76-77%)

---

### C. Additional examples (30 min)

**Nice-to-have demos**:
1. `string_transform.ail` - String operations with std/string
2. `list_operations.ail` - Comprehensive list usage
3. `error_handling.ail` - Result chain with multiple error types

---

## Final Checklist

### Phase 1: M-S1 Completion
- [ ] **TC-LIST-CONCAT**: Typing rule + 4 tests (incl. generalization) + tailored error with hint
- [ ] **IO-EQUATION**: Equation-form binding for print/println/readLine/debug
- [ ] **STDLIB-CHECK**: All 5 modules type-check; 3/3 examples run

### Phase 2: API Stability
- [ ] **FREEZE**: SHA256 goldens for std/* APIs (normalized JSON); CI gate
- [ ] **GOLDENS**: Example goldens + verify-examples.sh; CI gate

### Phase 3: Demos
- [ ] **DEMOS**: 3 polished demos (≤30 LOC) with comments + .golden files
  - [ ] effects_pure.ail (pure data pipeline)
  - [ ] io_hello.ail (IO effects)
  - [ ] adt_pipeline.ail (Result plumbing)

### Phase 4: Polish
- [ ] **ERRORS**: Enhanced error messages (LIST_CONCAT_MISMATCH, IMPORT_CONSTRUCTOR_NOT_EXPORTED)
- [ ] **DOCS**: v0.1.0 summary + known limits + stdlib tables
  - [ ] README.md updates
  - [ ] CHANGELOG.md entry
  - [ ] Stdlib reference tables

### Phase 5: Stretch (Optional)
- [ ] extern keyword + parser tests
- [ ] Parser coverage add-ons (+1-2%)
- [ ] Additional examples

---

## Success Metrics

**Technical**:
- ✅ All 5 stdlib modules type-check
- ✅ All examples pass golden file verification
- ✅ API stability enforced with SHA256 hashes
- ✅ Parser coverage: 75.1% (acceptable for v0.1.0)
- ✅ 3-5 polished demos ready

**Documentation**:
- ✅ README accurately reflects v0.1.0 capabilities
- ✅ Known limitations clearly documented
- ✅ Stdlib API reference complete
- ✅ CHANGELOG updated

**Timeline**:
- ✅ M-S1 complete within 72 hours
- ✅ Ship-ready v0.1.0 package

---

## Deliverables Checklist

### Code Artifacts
- [ ] `internal/types/typechecker.go` - List concat typing rule
- [ ] `internal/parser/parser.go` - Equation-form export support
- [ ] `stdlib/std/io.ail` - Equation-form exports
- [ ] `stdlib/std/list.ail` - Working with ++ operator
- [ ] `Makefile` - test-stdlib-freeze, verify-examples targets
- [ ] `scripts/verify-examples.sh` - Example verification script

### Test Artifacts
- [ ] `internal/types/typechecker_test.go` - List concat tests
- [ ] `goldens/stdlib/*.sha256` - API freeze hashes (5 files)
- [ ] `examples/*.golden` - Example output files (5+ files)

### Demo Artifacts
- [ ] `examples/effects_pure.ail` + `.golden`
- [ ] `examples/io_hello.ail` + `.golden`
- [ ] `examples/adt_pipeline.ail` + `.golden`

### Documentation Artifacts
- [ ] `README.md` - Updated features, limitations, stdlib tables
- [ ] `CHANGELOG.md` - v0.1.0 entry
- [ ] `design_docs/20251002/M-S1-polish.md` - This document

---

## Implementation Support

**Helper Scripts Available**:
1. `verify-examples.sh` - Example golden file verification
2. Normalized iface JSON schema
3. Parser diff for equation-form functions
4. List ++ typing rule in Go

**Ready to implement when approved.**

---

*Last updated: October 2, 2025*
*Target ship date: October 4-5, 2025*
