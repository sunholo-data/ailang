# Sprint Plan: M-CODEGEN-SUSTAINABILITY (Option C — Colocated Builtin Registry)

## Summary

Refactor the Go codegen to use the existing BuiltinMeta registry for codegen specs instead of separate hardcoded mapping tables. Eliminates `mapStdlibBuiltin`, `mapPureMathBuiltin`, `mapPureListBuiltin`, and `codegen_runtime_stdlib.go`. Also adds CI test harness and unifies duplicated `isUserDefinedType`.

**Duration:** 3 days (5 milestones)
**Dependencies:** M-CODEGEN-STDLIB-BUILTINS (done)
**Risk Level:** Medium — large refactoring but all existing tests must pass throughout

## Proposed Milestones

### M1: EXTEND_REGISTRY — Add GoCodegenSpec to BuiltinMeta

**Goal:** Extend the builtin registry struct with codegen specifications.

**Estimated:** 80 LOC implementation + 20 LOC tests = 100 LOC
**Duration:** ~2h

**Tasks:**
1. Add `GoCodegenSpec` struct to `internal/builtins/registry.go` with `Inline`, `Helper`, `Imports` fields
2. Add `GoCodegen *GoCodegenSpec` field to `BuiltinMeta`
3. Add `GoHelperSpec` struct for runtime helper functions
4. Add `GetCodegenSpec(name string) *GoCodegenSpec` helper function
5. Unit test for registry query
6. `make test`

**Acceptance Criteria:**
- [ ] `BuiltinMeta` has `GoCodegen` field
- [ ] Registry compiles with new types
- [ ] No existing tests break
- [ ] `make test` passes

### M2: ANNOTATE_BUILTINS — Add GoCodegenSpec to all builtins

**Goal:** Annotate all ~87 internal builtins with their Go codegen equivalents.

**Estimated:** 250 LOC annotations + 30 LOC tests = 280 LOC
**Duration:** ~3h

**Tasks:**
1. Annotate arithmetic builtins (add_Int, sub_Int, etc.) — already handled by codegen, mark as `Inline`
2. Annotate string builtins (_str_trim, _str_split, etc.) — migrate from `mapStdlibBuiltin` entries
3. Annotate list builtins (_list_map, _list_filter, etc.) — migrate from `mapPureListBuiltin` + runtime helpers
4. Annotate math builtins (_math_sin, _math_cos, etc.) — migrate from `mapPureMathBuiltin`
5. Annotate JSON builtins — migrate from `mapStdlibBuiltin` + runtime helpers
6. Annotate XML builtins — migrate stubs
7. Annotate IO/FS/Env/AI effect builtins — migrate effect handler stubs
8. Test: verify every builtin with IsPure=true has GoCodegen set
9. `make test`

**Acceptance Criteria:**
- [ ] All builtins that appear in `mapStdlibBuiltin`/`mapPureMathBuiltin`/`mapPureListBuiltin` have GoCodegen specs
- [ ] CI-verifiable: `ailang doctor builtins --check-codegen` reports zero gaps
- [ ] `make test` passes

### M3: CODEGEN_QUERY_REGISTRY — Update codegen to use registry

**Goal:** Replace hardcoded map lookups with registry queries. Delete mapping tables.

**Estimated:** 150 LOC changes + 50 LOC tests = 200 LOC
**Duration:** ~3h

**Tasks:**
1. Import `builtins` package in codegen (add dependency)
2. Replace `mapPureMathBuiltin` calls with registry lookup
3. Replace `mapPureListBuiltin` calls with registry lookup
4. Replace `mapStdlibBuiltin` calls with registry lookup
5. For `Inline` specs: emit inline expression with arg substitution
6. For `Helper` specs: emit runtime helper function (dedup — only emit each once)
7. Track `Imports` and add to generated file header
8. Delete `mapPureMathBuiltin`, `mapPureListBuiltin`, `mapStdlibBuiltin` functions
9. Delete `codegen_runtime_stdlib.go` (move helper bodies to registry annotations)
10. Run DocParse compilation to verify parity
11. `make test`

**Acceptance Criteria:**
- [ ] Zero hardcoded mapping tables in codegen_expr_simple.go
- [ ] `codegen_runtime_stdlib.go` deleted (or reduced to shared utilities only)
- [ ] DocParse compiles to Go with same result as before
- [ ] All codegen tests pass
- [ ] `make test` passes

### M4: UNIFY_TYPE_MAPPING — Single isUserDefinedType

**Goal:** Eliminate duplicated `isUserDefinedType` / `isUserDefinedGoType`.

**Estimated:** 50 LOC refactoring + 20 LOC tests = 70 LOC
**Duration:** ~1h

**Tasks:**
1. Create unified `IsUserDefinedGoType` in shared location (registry or type_registry.go)
2. Update `adt.go` to call shared function
3. Update `compile_types.go` to call shared function
4. Delete duplicate implementations
5. `make test`

**Acceptance Criteria:**
- [ ] `isUserDefinedType` exists in ONE file
- [ ] Both adt.go and compile_types.go use the same function
- [ ] `make test` passes

### M5: CI_HARNESS — Multi-module codegen test + CHANGELOG

**Goal:** Add CI test that compiles a reference multi-module project and runs `go build`.

**Estimated:** 150 LOC test harness + 50 LOC CI = 200 LOC
**Duration:** ~2h

**Tasks:**
1. Create `tests/codegen-harness/` with multi-module .ail files exercising all stdlib modules
2. Add `make test-codegen` target that compiles + builds
3. Add `.github/workflows/test-codegen-multimodule.yml` CI workflow
4. Update CHANGELOG
5. `make test`

**Acceptance Criteria:**
- [ ] `make test-codegen` compiles harness and runs `go build`
- [ ] CI workflow triggers on codegen/stdlib/builtin changes
- [ ] CHANGELOG updated
- [ ] `make test` passes

## Success Metrics
- All existing tests passing: ✅
- `mapStdlibBuiltin` deleted: ✅
- `mapPureMathBuiltin` deleted: ✅
- `mapPureListBuiltin` deleted: ✅
- `codegen_runtime_stdlib.go` eliminated or minimized: ✅
- CI multi-module test: ✅
- `isUserDefinedType` in one file: ✅
- DocParse compiles identically: ✅

## Open Questions
- **Registry import cycle:** `internal/gen/golang/` importing `internal/builtins/` — verify no circular dependency
- **Helper dedup:** When multiple builtins share the same Go helper (e.g., `CallFunc`), ensure it's emitted once
- **Inline vs Helper threshold:** Simple builtins use Inline, complex ones use Helper. Where's the line? Proposal: if it's a single Go expression, use Inline; if it needs control flow, use Helper.
