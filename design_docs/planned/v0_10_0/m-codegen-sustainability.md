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

## Related Documents

- [m-codegen-multimodule-bugs](../v0_9_2/m-codegen-multimodule-bugs.md) — The bugs this session fixed
- [m-codegen-stdlib-builtins](../v0_9_3/m-codegen-stdlib-builtins.md) — The stdlib mapping problem
- [m-codegen-ir-strategy](m-codegen-ir-strategy.md) — Future IR refactoring (complements this)
- [m-codegen-api-server](m-codegen-api-server.md) — Compiled API server (depends on stable codegen)

---

**Document created**: 2026-03-17
**Last updated**: 2026-03-17
