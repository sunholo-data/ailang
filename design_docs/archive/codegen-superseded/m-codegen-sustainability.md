# M-CODEGEN-SUSTAINABILITY: Codegen Validator Skill + Stdlib-as-Go-Module

**Status**: Planned
**Target**: v0.10.0
**Priority**: P1 (Strategic — prevents recurring multi-day debugging sessions)
**Estimated**: 2-3 weeks
**Dependencies**: M-CODEGEN-MULTIMODULE-BUGS (v0.9.2, done), M-CODEGEN-STDLIB-BUILTINS (v0.9.3, done)
**Created**: 2026-03-17

---

## Problem Statement

The Go codegen worked for stapledon's_voyage (1 module, Dec 2025) but broke catastrophically for DocParse (22 modules, March 2026). Fixing it required 8 commits, ~3,000 LOC, and a full day of iterative debugging. Every fix revealed another bug. The root causes are structural:

### 1. No CI Test for Multi-Module Compilation

The codegen has unit tests for individual expressions but **no integration test that compiles multiple modules and runs `go build`**. stapledon's_voyage worked because it was 1 module. DocParse's 22 modules hit 3 bugs that would have been caught by a single CI test.

### 2. Duplicated Logic Without Shared Abstraction

`isUserDefinedType` exists in **two separate files** (`adt.go` and `compile_types.go`). We fixed the `[]string` bug in one file but not the other. This pattern repeats across the codegen:

| Logic | Location 1 | Location 2 | Bugs from mismatch |
|-------|-----------|-----------|-------------------|
| isUserDefinedType | `adt.go:441` | `compile_types.go:203` | `[]*[]string` instead of `[][]string` |
| Type mapping (AST → Go) | `adt.go:mapASTType` | `compile_types.go:ailangTypeToGo` | Inconsistent pointer/value decisions |
| Function namespacing | `codegen_decl.go:generateFuncFromLambda` | `codegen_decl.go:generateImplFunc` + `generateTypedWrapper` | 3 places to update for any prefix change |

### 3. Hardcoded Stdlib Mappings Scale Linearly

Every stdlib function requires a manual entry in `mapStdlibBuiltin()` + a Go runtime helper implementation. Current count: **~80 entries** across math, string, list, JSON, XML, option, result, io, env, zip. Adding regex, date/time, or compression means +20 entries each.

```
v0.5.9:  36 entries (math only)
v0.9.3:  80+ entries (all stdlib modules)
v0.10.0: 100+ (projected, with new modules)
v0.11.0: 150+ (projected)
```

### 4. No Regression Detection Between Versions

When new language features are added (builtins, type system changes, Core lowering optimizations), there's no automated check that the Go codegen still works. Every user project becomes an integration test.

---

## Goals

**Primary Goal:** Make the Go codegen self-maintaining — new stdlib functions and language features don't require manual codegen updates.

**Success Metrics:**
1. CI catches codegen regressions before they reach users
2. New stdlib functions automatically available in Go codegen without manual mapping
3. `isUserDefinedType` logic exists in ONE place
4. DocParse-scale projects (20+ modules) compile to Go without manual intervention

---

## Solution Design

### Component 1: `codegen-validator` Skill

A new Claude Code skill that validates the Go codegen pipeline end-to-end.

**Trigger:** After any change to `internal/gen/golang/`, `cmd/ailang/compile*.go`, `std/*.ail`, or `internal/builtins/`

**What it does:**
1. Compile a reference multi-module project to Go
2. Run `go build` on the output
3. Report any new `undefined`, `duplicate case`, `declared and not used`, or type mismatch errors
4. Optionally run `go test` if test harness exists

**Reference project:** Either DocParse (real-world) or a synthetic test harness that exercises all stdlib modules, multi-module imports, ADT patterns, and effect handlers.

**CI Integration:** Add `.github/workflows/test-codegen-multimodule.yml` that runs on every PR touching codegen.

**Estimated:** 2-3 days

### Component 2: Shared Type Mapping Abstraction

Eliminate duplicated `isUserDefinedType` and `ailangTypeToGo` logic.

**Current state:**
- `adt.go:isUserDefinedType` — used by ADT type generator
- `compile_types.go:isUserDefinedGoType` — used by record type registration
- `adt.go:mapASTType` — used by ADT struct generation
- `compile_types.go:ailangTypeToGo` — used by function signature registration

**Fix:** Extract a single `TypeMapper` that all consumers use:

```go
// internal/gen/golang/type_registry.go
type TypeRegistry struct {
    valueRecords map[string]bool
    adtTypes     map[string]bool
}

func (r *TypeRegistry) IsUserDefined(goType string) bool { ... } // ONE implementation
func (r *TypeRegistry) MapASTType(t ast.Type) string { ... }     // ONE implementation
```

**Estimated:** 1 week (refactoring + updating all call sites + tests)

### Component 3: Stdlib-as-Go-Module (Long-term)

Instead of re-implementing every stdlib function in codegen runtime helpers, generate Go code that imports a published AILANG runtime Go module.

**Current approach (unsustainable):**
```go
// Generated runtime.go — 1000+ LOC of reimplemented stdlib
func Trim(s interface{}) interface{} { return strings.TrimSpace(s.(string)) }
func Map(f, xs interface{}) interface{} { ... }
func Filter(p, xs interface{}) interface{} { ... }
// ... 60+ more functions
```

**Proposed approach:**
```go
// Generated code imports shared module
import "github.com/sunholo/ailang-go-stdlib"

// Codegen just generates function calls — no reimplementation
func process_impl(text interface{}) interface{} {
    return stdlib.Split(stdlib.Trim(text), ",")
}
```

**What the `ailang-go-stdlib` module contains:**
- All stdlib function implementations (Trim, Map, Filter, etc.)
- ADT runtime helpers (CallFunc, Option/Result constructors)
- Type conversion utilities (toSlice, toInt64, etc.)
- Effect handler interfaces

**Advantages:**
- Stdlib functions defined once, tested once
- No `mapStdlibBuiltin` table needed — function names match directly
- New stdlib functions automatically available after module update
- Runtime helpers shared across all compiled projects

**Estimated:** 2-3 weeks (extract runtime to module, update codegen, publish module)

### Component 4: Pipeline Function Emission Fix (xlsx_parser)

Investigate and fix the Core IR pipeline issue where effectful functions compile to LetRec with Lambda bindings but produce zero Go output.

**Symptoms:** xlsx_parser has 26 LetRec declarations in Core IR. The codegen processes them without error but emits nothing. The functions are pure (`extractSharedStrings`, `parseSheetXml`) yet don't generate Go code.

**Investigation needed:** Add instrumentation to `generateFuncFromLambda` to trace what happens for xlsx_parser functions. Likely the Lambda body contains an expression type the codegen doesn't handle, causing it to produce empty output without erroring.

**Estimated:** 1-2 days

---

## Implementation Plan

**Phase 1: codegen-validator skill + CI** (3 days)
- [ ] Create synthetic multi-module test project in `tests/codegen/`
- [ ] Write `codegen-validator` skill that compiles and builds it
- [ ] Add CI workflow `.github/workflows/test-codegen-multimodule.yml`
- [ ] Document in skill README

**Phase 2: Shared type mapping** (1 week)
- [ ] Create `internal/gen/golang/type_registry.go`
- [ ] Migrate `isUserDefinedType` to single implementation
- [ ] Migrate `mapASTType` / `ailangTypeToGo` to unified function
- [ ] Update all call sites in adt.go, compile_types.go, types.go
- [ ] Run codegen-validator to verify no regressions

**Phase 3: stdlib-as-Go-module** (2-3 weeks)
- [ ] Extract runtime helpers to `github.com/sunholo/ailang-go-stdlib`
- [ ] Publish module with all current stdlib functions
- [ ] Update codegen to generate `import ailang-go-stdlib` instead of runtime helpers
- [ ] Remove `codegen_runtime_stdlib.go` (1000+ LOC eliminated)
- [ ] Run codegen-validator to verify parity

**Phase 4: Pipeline fix for effectful functions** (1-2 days)
- [ ] Instrument `generateFuncFromLambda` for xlsx_parser
- [ ] Identify the expression type that produces empty output
- [ ] Fix the codegen case or Core lowering
- [ ] Add regression test

---

## Evidence: Today's Debugging Session

This design doc is motivated by the 2026-03-17 session where 8 commits were needed:

| Commit | What broke | Root cause | Would CI catch it? |
|--------|-----------|------------|-------------------|
| b5a961fc | Constant redeclaration | No module prefix for non-lambda lets | **Yes** |
| 6da6f285 | XmlNode undefined | Type only in Go, not .ail | **Yes** |
| a77b0f79 | Trim/Split/Map undefined | No stdlib runtime helpers | **Yes** |
| bd49878e | Duplicate case OptionKindSome | ADT match with nested literals | **Yes** |
| b91de67d | Unused vars, missing stubs | Flat body suppression, effect stubs | **Yes** |
| 61a068e7 | []*[]string type mismatch | **Duplicated isUserDefinedType** | **Yes** |
| (pending) | xlsx_parser empty | Pipeline issue | **Yes** |

**Every single issue** would have been caught by a multi-module CI test that runs `go build` on DocParse or a similar reference project.

---

## Success Criteria

- [ ] CI test that compiles multi-module project and runs `go build`
- [ ] `isUserDefinedType` exists in ONE file, used by all consumers
- [ ] New stdlib function requires 0 codegen changes (stdlib-as-module)
- [ ] DocParse compiles with `go build` in CI
- [ ] No more "whack-a-mole" debugging sessions for Go codegen

---

## Deep Analysis: Architectural Options

The fundamental problem: AILANG has **two execution backends** (interpreter + Go codegen) that must both handle 57 registered builtins, 283 stdlib exports, ADT operations, pattern matching, and effects. Today these are maintained independently. Every new builtin or stdlib function requires updates in both.

### The Execution Chain

```
AILANG source
    ↓
Parser → AST → Elaboration → Core IR
    ↓                              ↓
Interpreter                    Go Codegen
    ↓                              ↓
eval.Value                     Go source
(builtins in Go)               (runtime helpers in Go)
```

Both paths need Go implementations of the same functions. The interpreter has them in `internal/builtins/*.go`. The codegen re-implements them in `codegen_runtime_stdlib.go`. This duplication is the root cause of all scaling problems.

### Option A: Stdlib-as-Go-Module (External Import)

Generated code imports a published `ailang-go-stdlib` module.

```go
import stdlib "github.com/sunholo/ailang-go-stdlib"
result := stdlib.Trim(text)
```

| Pros | Cons |
|------|------|
| Functions defined once | Generated code needs external dependency |
| Versioned updates | Module version must match codegen version |
| -1000 LOC from codegen | Still need `interface{}` wrappers |
| No `mapStdlibBuiltin` table | Need to publish and maintain separate repo |

**Verdict:** Solves the maintenance problem but adds deployment complexity. Every compiled project depends on an external module, making standalone binaries harder.

### Option B: Compile stdlib .ail to Go (Self-Hosting)

Instead of hardcoding mappings, compile `std/string.ail`, `std/list.ail` etc. through the same Go codegen pipeline as user code. The stdlib becomes part of the generated output.

The chain for `trim`:
```
std/string.ail: export pure func trim(s: string) -> string { _str_trim(s) }
    ↓ compile to Core IR
Let("trim", Lambda(["s"], App(VarGlobal("_str_trim"), [Var("s")])))
    ↓ Go codegen
func Trim(s interface{}) interface{} { return _str_trim(s) }
```

This **almost works** today — except `_str_trim` is a Go builtin that the codegen doesn't know how to emit. If we provide Go implementations for the ~57 low-level builtins (`_str_trim`, `_str_split`, `_list_map`, etc.), the stdlib .ail files would compile through the normal codegen pipeline.

| Pros | Cons |
|------|------|
| Truly self-hosting — stdlib compiles like user code | Still need ~57 builtin primitives in Go |
| No mapping table at all | Stdlib functions in generated output (larger binary) |
| Automatic — add .ail function, it just works | Two-phase compile (stdlib first, then user code) |
| Tests stdlib through codegen pipeline | May hit codegen bugs in stdlib itself |

**Verdict:** Elegant but requires solving the "last mile" — the ~57 primitives. And stdlib functions that use AILANG features the codegen doesn't support yet would fail.

### Option C: Colocated Builtin Registry (Recommended)

Extend `BuiltinMeta` with Go codegen specifications. When a builtin is registered for the interpreter, it simultaneously registers its codegen equivalent. **Zero separate tables.**

```go
// internal/builtins/registry.go — EXTENDED
type BuiltinMeta struct {
    Name    string
    NumArgs int
    IsPure  bool

    // Go codegen support — when set, codegen can emit this builtin
    GoCodegen *GoCodegenSpec  // NEW
}

type GoCodegenSpec struct {
    // For simple mappings: inline Go expression template
    // {{arg0}}, {{arg1}} etc. are replaced with argument expressions
    Inline string        // e.g., "strings.TrimSpace({{arg0}}.(string))"

    // For complex mappings: runtime helper function to emit
    Helper *GoHelperSpec  // e.g., Map, Filter, Foldl

    // Go imports needed
    Imports []string       // e.g., ["strings"]
}

type GoHelperSpec struct {
    FuncName  string  // Go function name, e.g., "Map"
    Signature string  // e.g., "func Map(f, xs interface{}) interface{}"
    Body      string  // Go function body
}
```

**Registration example:**
```go
// Before (two separate systems):
// 1. internal/builtins/string.go  — interpreter impl
// 2. codegen_runtime_stdlib.go    — re-implemented Trim()
// 3. codegen_expr_simple.go       — mapStdlibBuiltin("trim" → "Trim")

// After (single registration):
Registry["_str_trim"] = &BuiltinMeta{
    Name: "_str_trim", NumArgs: 1, IsPure: true,
    GoCodegen: &GoCodegenSpec{
        Inline:  "strings.TrimSpace({{arg0}}.(string))",
        Imports: []string{"strings"},
    },
}
```

**For higher-order functions:**
```go
Registry["_list_map"] = &BuiltinMeta{
    Name: "_list_map", NumArgs: 2, IsPure: true,
    GoCodegen: &GoCodegenSpec{
        Helper: &GoHelperSpec{
            FuncName:  "Map",
            Signature: "func Map(f, xs interface{}) interface{}",
            Body: `list := toSlice(xs)
result := make([]interface{}, len(list))
for i, x := range list {
    result[i] = CallFunc(f, x)
}
return result`,
        },
    },
}
```

**How the codegen uses it:**

1. When encountering `VarGlobal("_str_trim")`, query `Registry["_str_trim"].GoCodegen`
2. If `Inline` is set, substitute args and emit inline expression
3. If `Helper` is set, emit the helper function in `runtime.go` (only if not already emitted)
4. Track required imports from `GoCodegen.Imports`

**What this eliminates:**
- `mapStdlibBuiltin()` — **deleted** (replaced by registry lookup)
- `mapPureMathBuiltin()` — **deleted** (absorbed into registry)
- `mapPureListBuiltin()` — **deleted** (absorbed into registry)
- `codegen_runtime_stdlib.go` — **deleted** (~1000 LOC) (helpers emitted from registry specs)

**What this enables:**
- Adding a new builtin automatically makes it available in Go codegen
- The `builtin-developer` skill can generate both interpreter AND codegen implementations
- `ailang doctor builtins` can verify all builtins have codegen specs
- CI can validate: "every builtin with IsPure=true has GoCodegen set"

| Pros | Cons |
|------|------|
| Zero maintenance for new builtins | Need to annotate all ~57 existing builtins |
| Single source of truth | Registry struct grows |
| Existing skill (`builtin-developer`) can generate both | Initial migration effort |
| CI-verifiable: every builtin has codegen | Go body strings are less IDE-friendly |
| -1000 LOC codegen, -80 mapping entries | Runtime helper bodies as strings |

**Verdict: This is the right answer.** It follows the GHC primop pattern where each primitive has both an evaluator and a code generator, defined together. The migration is ~200 LOC of annotations across existing builtins.

### Option D: Hybrid — Registry + Compiled Stdlib

Combine Option C (registry for primitives) with Option B (compile stdlib .ail).

The ~57 low-level builtins (`_str_trim`, `_list_map`, etc.) get Go codegen specs via the registry. Then `std/string.ail`, `std/list.ail` etc. compile through the normal codegen pipeline — they call the primitives which now have Go implementations.

```
Layer 1: Primitives (_str_trim, _list_map, etc.)
         → GoCodegen specs in BuiltinMeta registry
         → Emitted as inline expressions or runtime helpers

Layer 2: Stdlib (trim, map, filter, etc.)
         → Compiled from .ail source via codegen pipeline
         → Calls Layer 1 primitives
         → No manual mapping needed

Layer 3: User code
         → Compiled from .ail source via codegen pipeline
         → Calls Layer 2 stdlib functions
```

This is the most sustainable architecture: **57 annotated primitives** support **283 stdlib functions** which support **unlimited user code**. The 283 stdlib functions require zero codegen maintenance because they compile through the same pipeline as user code.

| Pros | Cons |
|------|------|
| Only 57 primitives to annotate (not 283) | Two-phase compilation (stdlib → user code) |
| Stdlib changes propagate automatically | Stdlib must compile cleanly through codegen |
| Tests stdlib through the codegen itself | Initial effort to get stdlib compiling |
| Catches codegen bugs in stdlib before users hit them | May need stdlib-specific codegen fixes |

**Estimated effort:**
- Phase 1: Registry annotations for 57 primitives — 3 days
- Phase 2: Codegen changes to query registry — 2 days
- Phase 3: Compile stdlib .ail to Go — 1 week (may hit new codegen issues)
- Phase 4: CI integration — 1 day

### Recommendation

**Short-term (v0.10.0):** Option C — Colocated builtin registry. This is the smallest change that eliminates the mapping tables and makes new builtins automatic. ~1 week effort.

**Medium-term (v0.11.0):** Option D — Add compiled stdlib on top of registry. This eliminates ALL manual stdlib mapping. The 283 stdlib functions compile through the same pipeline. ~2 weeks additional effort.

**Long-term:** The compiled stdlib approach naturally leads to self-hosting — AILANG's Go backend compiles its own standard library. This is the path compilers like GHC, Rust, and Go itself took.

---

## Codegen Validator Skill: Detailed Design

The skill should be more than "run `go build`". It should be a comprehensive regression gate:

### Reference Test Project

Create `tests/codegen-harness/` with AILANG modules that exercise every codegen feature:

```
tests/codegen-harness/
├── types/types.ail           # ADTs, records, nested types, [[string]]
├── services/string_ops.ail    # All std/string imports
├── services/list_ops.ail      # All std/list imports (HOFs, pattern match)
├── services/json_ops.ail      # JSON construction, accessors, decode
├── services/xml_ops.ail       # XML parse, findAll, getText
├── services/math_ops.ail      # Math builtins
├── services/cross_ref.ail     # Cross-module function references
├── services/effects.ail       # IO, FS, Env, AI effect stubs
├── main.ail                   # Multi-module imports, match patterns
└── expected_errors.txt        # Known issues (xlsx_parser pipeline)
```

**Key patterns to test:**
- Multi-module with same-named functions (the DocParse parseDocxComments case)
- ADT match with nested literal patterns (Some("heading"), Some("text"))
- Forward references (function passed as value before declaration)
- Nested list types ([[string]], [Option[int]])
- Effectful function stubs
- Higher-order functions (Map, Filter with lambdas and named functions)
- Cross-module type imports

### Skill Trigger

```yaml
# .claude/skills/codegen-validator/SKILL.md
trigger: after changes to internal/gen/golang/, cmd/ailang/compile*, std/*.ail, internal/builtins/
```

### Validation Steps

1. `ailang compile --emit-go --out /tmp/codegen-test --package-name harness tests/codegen-harness/*.ail`
2. `cd /tmp/codegen-test && go mod init harness && go build ./harness/`
3. `go vet ./harness/`
4. Diff generated output against golden files (detect unexpected changes)
5. Report: pass/fail, new errors, removed errors

### CI Workflow

```yaml
# .github/workflows/test-codegen-multimodule.yml
name: Go Codegen Integration
on:
  pull_request:
    paths: ['internal/gen/golang/**', 'cmd/ailang/compile*', 'std/**', 'internal/builtins/**']
jobs:
  codegen-build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
      - run: make build
      - run: make quick-install
      - run: |
          ailang compile --emit-go --out /tmp/harness --package-name harness \
            tests/codegen-harness/*.ail
          cd /tmp/harness && go mod init harness && go build ./harness/
          go vet ./harness/
```

## Related Documents

- [m-codegen-multimodule-bugs](../v0_9_2/m-codegen-multimodule-bugs.md) — The bugs this session fixed
- [m-codegen-stdlib-builtins](../v0_9_3/m-codegen-stdlib-builtins.md) — The stdlib mapping problem
- [m-codegen-ir-strategy](m-codegen-ir-strategy.md) — Future IR refactoring (complements this)
- [m-codegen-api-server](m-codegen-api-server.md) — Compiled API server (depends on stable codegen)

---

**Document created**: 2026-03-17
**Last updated**: 2026-03-17

- [m-codegen-multimodule-bugs](../v0_9_2/m-codegen-multimodule-bugs.md) — The bugs this session fixed
- [m-codegen-stdlib-builtins](../v0_9_3/m-codegen-stdlib-builtins.md) — The stdlib mapping problem
- [m-codegen-ir-strategy](m-codegen-ir-strategy.md) — Future IR refactoring (complements this)
- [m-codegen-api-server](m-codegen-api-server.md) — Compiled API server (depends on stable codegen)

---

**Document created**: 2026-03-17
**Last updated**: 2026-03-17
