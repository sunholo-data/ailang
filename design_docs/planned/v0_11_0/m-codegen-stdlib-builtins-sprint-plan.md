# Sprint Plan: M-CODEGEN-STDLIB-BUILTINS

## Summary

Implement Go codegen support for stdlib functions so DocParse (and any AILANG project using stdlib) compiles to a working Go binary. Extends the established `mapPureBuiltin` pattern and adds runtime helpers for higher-order functions.

**Duration:** 1 day (4 milestones, ~8h total)
**Dependencies:** M-CODEGEN-MULTIMODULE-BUGS (complete)
**Risk Level:** Low-Medium — established patterns, but HOF runtime helpers need `applyFunc` infrastructure

## Current Status Analysis

### Completed Recently
- ✅ M-CODEGEN-MULTIMODULE-BUGS: All 3 codegen bugs fixed + List type alias + XmlNode type decl
- ✅ DocParse compiles all 22 modules to Go with zero warnings
- ✅ `go build` gets past type/name issues, now only fails on 4 undefined stdlib functions

### Velocity
- Recent: ~200 LOC/day (codegen area, from today's sprint)
- This sprint: ~750 LOC estimated, aggressive but achievable in 1 day

### Remaining — DocParse `go build` errors
- `undefined: Trim` (3 references) — std/string
- `undefined: Substring` (3 references) — std/string
- `undefined: Split` (2 references) — std/string
- `undefined: Map` (2 references) — std/list

## Proposed Milestones

### M1: STRING_BUILTINS — Map std/string functions to Go stdlib

**Goal:** All `std/string` functions referenced by DocParse generate valid Go using `strings.*` package calls or runtime helpers.

**Estimated:** 100 LOC implementation + 40 LOC tests = 140 LOC
**Duration:** ~2h

**Tasks:**
1. Add `mapPureStringBuiltin()` function in `codegen_expr_simple.go` with mappings for: `trim` → `strings.TrimSpace`, `toUpper` → `strings.ToUpper`, `toLower` → `strings.ToLower`, `contains` → `strings.Contains`, `startsWith` → `strings.HasPrefix`, `endsWith` → `strings.HasSuffix`, `find` → runtime helper (returns int, not Option), `compare` → `strings.Compare`
2. Add `Substring`, `Split`, `Chars`, `Words`, `Repeat`, `Join` as runtime helpers in new `codegen_runtime_stdlib.go` (these need argument adaptation or aren't direct Go stdlib calls)
3. Track `needsStringsImport` flag in Generator, emit `"strings"` import when needed
4. Wire `mapPureStringBuiltin` into VarGlobal resolution chain (after `mapPureListBuiltin`)
5. Add `mapPureStringConvBuiltin` for `intToStr`, `floatToStr`, `stringToInt`, `stringToFloat`
6. Run `make test`

**Acceptance Criteria:**
- [ ] `Trim`, `Split`, `Substring` resolve to valid Go in generated DocParse code
- [ ] `strings` package imported when string builtins are used
- [ ] Unit test for each mapped string function
- [ ] `make test` passes
- [ ] `make lint` passes

### M2: LIST_HOF_BUILTINS — Runtime helpers for list higher-order functions

**Goal:** `Map`, `Filter`, `Foldl` and other HOF list functions work in generated Go via runtime helpers.

**Estimated:** 150 LOC implementation + 60 LOC tests = 210 LOC
**Duration:** ~2.5h

**Tasks:**
1. Add `applyFunc(f, args ...interface{}) interface{}` to runtime — dispatches `func(interface{}) interface{}` and similar signatures
2. Add `Map(f, xs interface{}) interface{}` runtime helper
3. Add `Filter(p, xs interface{}) interface{}` runtime helper
4. Add `Foldl(f, acc, xs interface{}) interface{}` runtime helper
5. Add `Reverse`, `Take`, `Drop`, `Any`, `SortBy`, `Zip`, `FlatMap` runtime helpers
6. Extend `mapPureListBuiltin` to map AILANG names to runtime helper names
7. Run `make test`

**Acceptance Criteria:**
- [ ] `Map(func, list)` works with named functions, lambdas, and partial applications
- [ ] `applyFunc` handles 1-arg and 2-arg function signatures
- [ ] All list HOF helpers have unit tests
- [ ] `make test` passes
- [ ] `make lint` passes

**Risks:**
- `applyFunc` type dispatch — Go functions from codegen are `func(interface{}) interface{}` but the runtime needs to handle them generically. Follow the pattern used by existing runtime helpers.

### M3: JSON_XML_OPTION_BUILTINS — Stdlib accessor helpers

**Goal:** JSON, XML, Option, and Result accessor functions generate valid Go.

**Estimated:** 200 LOC implementation + 60 LOC tests = 260 LOC
**Duration:** ~2.5h

**Tasks:**
1. Add JSON accessor helpers: `JsonDecode`, `JsonEncode`, `JsonGet`, `JsonHas`, `JsonAsString`, `JsonAsNumber`, `JsonAsBool`, `JsonAsArray`, `JsonKeys` — these operate on the generated `*Json` ADT struct
2. Add Option helpers: `OptionMap`, `OptionFlatMap`, `OptionGetOrElse`, `IsNone`, `IsSome`
3. Add Result helpers: `ResultMap`, `ResultFlatMap`, `ResultGetOrElse`, `IsOk`, `IsErr`
4. Add XML helpers: `XmlParse`, `XmlFindAll`, `XmlFindFirst`, `XmlGetText`, `XmlGetAttr`, `XmlGetChildren`, `XmlGetTag`
5. Map stdlib names to runtime helper names in the builtin resolution chain
6. Run `make test`

**Acceptance Criteria:**
- [ ] JSON accessor functions work on generated Json ADT structs
- [ ] Option/Result helpers work on generated ADT structs
- [ ] XML helpers work on generated XmlNode ADT structs
- [ ] Unit tests for each helper category
- [ ] `make test` passes
- [ ] `make lint` passes

**Risks:**
- JSON decode/encode may need `encoding/json` import in generated code
- XML parse needs `encoding/xml` — may be complex. Start with pure accessors, defer parse/serialize to effect handlers if needed.

### M4: DOCPARSE_BUILD — Full integration verification

**Goal:** DocParse 22-module compilation produces a Go binary that builds.

**Estimated:** 20 LOC (CHANGELOG) + verification
**Duration:** ~1h

**Tasks:**
1. Run `make test` and `make lint`
2. Rebuild: `make quick-install`
3. Compile DocParse: `ailang compile --emit-go --out /tmp/docparse-final --package-name docparse`
4. Init and build: `cd /tmp/docparse-final && go mod init docparse && go build ./docparse/`
5. If new undefined symbols appear, add mappings (iterate)
6. Update CHANGELOG
7. Update design doc status

**Acceptance Criteria:**
- [ ] `go build` succeeds with zero errors for DocParse
- [ ] `make test` passes
- [ ] `make lint` passes
- [ ] CHANGELOG updated
- [ ] Design doc updated

## Success Metrics
- DocParse `go build` succeeds: ✅
- All existing tests passing: ✅
- New runtime helper tests: 15+
- Zero undefined symbols in generated Go: ✅
- CHANGELOG updated: ✅

## Dependencies
- M-CODEGEN-MULTIMODULE-BUGS (complete)
- DocParse source files at `/Users/mark/dev/sunholo/docparse/docparse/`

## Open Questions
- **JSON decode/encode**: Should these be runtime helpers (import `encoding/json`) or effect handler methods? Decision: start as runtime helpers, move to effect handlers if complexity warrants it.
- **Scope creep**: DocParse currently only uses 4 undefined functions. M3 (JSON/XML/Option/Result) may find more. Budget extra time in M4 for iteration.
