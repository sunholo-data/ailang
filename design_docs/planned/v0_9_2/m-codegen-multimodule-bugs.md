# M-CODEGEN-MULTIMODULE-BUGS: Fix 3 Go Codegen Bugs for Multi-Module Compilation

**Status**: Planned
**Target**: v0.9.2
**Priority**: P0 (Blocking — prevents Go binary compilation of DocParse)
**Estimated**: 1 day (4h implementation + 2h testing + 1h docs)
**Dependencies**: None
**Reporter**: docparse agent (message 476f8495)
**Created**: 2026-03-17

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is a pure bug fix — no language semantics change. All axioms are neutral or strengthened.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No semantic change — codegen output becomes correct |
| A2: Replayability | 0 | No impact on traces |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No authority changes |
| A5: Bounded Verification | +1 | Generated Go code compiles, enabling `go vet` verification |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Correct Go output enables machine compilation of AILANG programs |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | No cost changes |
| A10: Composability | +1 | Multi-module compilation composes correctly |
| A11: Structured Failure | 0 | No error handling changes |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +3** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Strengthens machine compilation

---

## Problem Statement

DocParse compiled all 19 AILANG modules (406 declarations, 16K lines Go) with `ailang compile --emit-go`. Compilation succeeds but `go build` fails with 3 distinct codegen bugs.

**Current State:**
- 19/19 modules emit Go code successfully
- `go build` fails due to name collisions and syntax errors
- Workaround: none (must fix codegen)

**Impact:**
- Blocks DocParse from shipping as a native Go binary
- Affects any multi-module `--emit-go` project (stapledon's_voyage also hit Bug 1)

---

## Bug Analysis

### Bug 1: Non-Lambda Let Bindings Missing Module Prefix (HIGH)

**Symptom:** `go build` fails with "redeclared in this block" for constants like `evalMaxRawWords`.

**Root cause:** `generateTopLevelLet` at [codegen_decl.go:36-52](internal/gen/golang/codegen_decl.go#L36-L52) applies module prefixing only for Lambda let bindings (line 41 → `generateFuncFromLambda`). Non-lambda let bindings (line 44-51) emit bare `var VarName = <expr>` without any module prefix:

```go
// Current code (codegen_decl.go:44-51)
varName := ToGoFuncName(let.Name, exported)  // ← NO module prefix!
g.writef("var %s = ", varName)
```

When the same constant name appears in multiple modules, the Go compiler sees duplicate declarations in the same package.

**Additionally:** The constant may be emitted once per *reference* rather than once per *declaration* if the Core lowering duplicates let bindings that are referenced from multiple call sites. Need to verify whether the Core program contains duplicate `Let` nodes or if there's a separate dedup issue.

**Fix:** Apply the same `moduleName__` prefixing pattern used in `generateFuncFromLambda` (line 117-120):

```go
// Fixed code
varName := ToGoFuncName(let.Name, exported)
if !exported && g.moduleName != "" {
    varName = g.moduleName + "__" + varName
}
g.writef("var %s = ", varName)
```

Also add a `seen` set to `Generator` to prevent duplicate emissions of the same binding within a single module's generation pass.

### Bug 2: Cross-Module Function Name Collision (HIGH)

**Symptom:** `go build` fails with "redeclared in this block" for functions like `parseDocxComments` that exist in multiple modules.

**Root cause:** The M-DX18 module prefixing (line 117-120) only applies to *non-exported* functions. If two modules both have a non-exported function with the same name AND the same module short name (e.g., two files in `docparse/services/` both having module path `docparse/...` with last component colliding), or if functions are incorrectly marked as exported, the prefix won't help.

**Investigation needed:** Check whether:
1. The docparse modules have distinct last-path-component names (likely — `docx_parser` vs `docparse_browser`)
2. The `isExported` check is correct for these functions
3. The `topLevelFuncs` map from a previous module's generation leaks into the next module (the `Generator` is reused across modules in the compile loop at [compile.go:429-470](cmd/ailang/compile.go#L429-L470))

**Most likely root cause:** The `Generator` is reused across all modules (line 429 reuses `codeGen`). Maps like `topLevelFuncs`, `topLevelImplFuncs`, `funcParamTypes`, and `funcReturnTypes` accumulate entries from ALL modules. When module B has a function with the same AILANG name as module A, the map entry is overwritten — but the Go code was already emitted for module A. The collision happens because both `.go` files are in the same package.

**Fix options:**
1. **Per-module Generator** — create a fresh `Generator` per module in the compile loop (cleanest, prevents all cross-contamination)
2. **Always prefix** — prefix ALL functions (exported and non-exported) with module name, then generate thin exported wrappers without prefix that call the prefixed version
3. **Dedup check** — track emitted Go function names globally and error on collision

Option 1 is recommended — it's the smallest change and eliminates an entire class of bugs.

### Bug 3: Bracket Syntax Error in markdown_parser.go (MEDIUM)

**Symptom:** `go build` reports `66:200: expected operand, found ']'` in generated `markdown_parser.go`.

**Root cause:** A complex pattern match or list literal in the markdown_parser module generates invalid Go syntax. Likely a list pattern `[x, y, z]` in a match arm or a list literal with nested expressions where the codegen emits a bare `]` without proper surrounding syntax.

**Investigation needed:** Generate the markdown_parser.go file and inspect line 66, column 200 to identify the exact AILANG construct that triggers the bad output.

**Fix:** Will be determined after inspection. Likely a missing case in `codegen_match.go` or `codegen_ops.go` for a specific list/pattern construct.

---

## Goals

**Primary Goal:** Make `go build` succeed for all 19 DocParse modules compiled with `ailang compile --emit-go`.

**Success Metrics:**
- All 19 DocParse modules compile to Go without `go build` errors
- Zero "redeclared in this block" errors across any multi-module project
- Zero syntax errors in generated Go code
- Existing single-module codegen tests still pass

---

## Solution Design

### Overview

Three targeted fixes to the Go code generator, all in `internal/gen/golang/`:

1. Add module prefixing to non-lambda `generateTopLevelLet`
2. Create fresh `Generator` per module (or reset state between modules)
3. Fix the bracket syntax error for the specific list construct

### Implementation Plan

**Phase 1: Fix constant redeclaration** (~1.5h)

- [ ] Add module prefix to `generateTopLevelLet` for non-exported var bindings
- [ ] Add `emittedVars map[string]bool` to `Generator` to deduplicate within a module
- [ ] Add unit test: two modules with same constant name → distinct Go var names
- [ ] Run `make test`

**Phase 2: Fix cross-module function collision** (~1.5h)

- [ ] Option A: Create fresh `Generator` per module in compile loop, or
- [ ] Option B: Add `ResetPerModuleState()` method to clear accumulated maps
- [ ] Ensure shared state (ADT constructors, record types, runtime helpers) is preserved
- [ ] Add unit test: two modules with same function name → distinct Go func names
- [ ] Run `make test`

**Phase 3: Fix markdown_parser bracket error** (~1h)

- [ ] Reproduce: compile markdown_parser.ail to Go, inspect line 66
- [ ] Identify the AILANG construct that triggers bad output
- [ ] Fix the codegen case (likely in `codegen_match.go` or `codegen_ops.go`)
- [ ] Add regression test
- [ ] Run `make test`

**Phase 4: Integration verification** (~1h)

- [ ] Run full DocParse compilation: `ailang compile --emit-go --out /tmp/docparse-go --package-name docparse docparse/main.ail docparse/services/*.ail docparse/types/*.ail`
- [ ] Run `go build` in output directory
- [ ] Run `make verify-examples`
- [ ] Update CHANGELOG.md

### Files to Modify

**Modified files:**
- `internal/gen/golang/codegen_decl.go` — Add module prefix to `generateTopLevelLet`, ~5 LOC
- `internal/gen/golang/codegen.go` — Add `emittedVars` map to Generator, add `ResetPerModuleState()` or make compile loop create fresh generators, ~20 LOC
- `cmd/ailang/compile.go` — Create fresh Generator per module (if Option A), ~15 LOC
- `internal/gen/golang/codegen_match.go` or `codegen_ops.go` — Fix bracket syntax for list construct, ~TBD LOC

**New files:**
- `internal/gen/golang/codegen_multimodule_test.go` — Regression tests for all 3 bugs, ~100 LOC

---

## Examples

### Example 1: Constant Redeclaration (Bug 1)

**Before (broken):** Two modules both emit:
```go
var evalMaxRawWords = 5000  // from eval.ail
var evalMaxRawWords = 5000  // from eval.ail (duplicate!)
```

**After (fixed):**
```go
var eval__evalMaxRawWords = 5000       // from eval.ail (module-prefixed)
var docxParser__evalMaxRawWords = 5000  // if also in docx_parser.ail
```

### Example 2: Function Name Collision (Bug 2)

**Before (broken):** Both modules emit to same package:
```go
func parseDocxComments_impl(...) interface{} { ... }  // from docx_parser.go
func parseDocxComments_impl(...) interface{} { ... }  // from docparse_browser.go — COLLISION!
```

**After (fixed):** Fresh generator per module, or always-prefixed:
```go
func docxParser__parseDocxComments_impl(...) interface{} { ... }
func docparseBrowser__parseDocxComments_impl(...) interface{} { ... }
```

---

## Success Criteria

- [ ] `go build` succeeds for DocParse 19-module compilation
- [ ] No "redeclared in this block" errors for constants or functions
- [ ] No syntax errors in generated Go files
- [ ] All existing codegen tests pass (`make test`)
- [ ] New regression tests for all 3 bugs
- [ ] `make verify-examples` passes
- [ ] CHANGELOG.md updated

## Testing Strategy

**Unit tests:**
- Multi-module generator: same constant in 2 modules → unique Go names
- Multi-module generator: same function in 2 modules → unique Go names
- List literal/pattern in match arm → valid Go syntax

**Integration tests:**
- Full DocParse compilation + `go build`
- `make verify-examples` (ensures single-module codegen not regressed)

**Manual testing:**
- Inspect generated Go files for correct prefixing
- Run DocParse Go binary (stretch goal — requires harness)

## Non-Goals

**Not in this feature:**
- `--with-default-handlers` flag — DocParse will write handlers manually (proven pattern from stapledon's_voyage)
- Multi-backend IR refactoring — tracked in [m-codegen-ir-strategy](../v0_10_0/m-codegen-ir-strategy.md) for v0.10.0
- Binding hoisting optimization — tracked in [m-codegen-v3-binding-hoisting](../v0_10_0/m-codegen-v3-binding-hoisting.md)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Fresh Generator per module breaks shared state (ADT types, record types) | High | Register shared types BEFORE per-module generation; test with DocParse |
| Bug 3 is a deeper codegen gap (new pattern not supported) | Medium | Inspect generated code first; may need follow-up design doc if complex |
| Module prefix changes break existing single-module codegen | Low | Non-exported prefix only triggers when moduleName is set; single-module doesn't set it |

## Related Documents

**Implemented (informs design):**
- [m-dx17-codegen-concatlist-closure-scoping](../../implemented/v0_6_1/m-dx17-codegen-concatlist-closure-scoping.md) — Prior codegen bug fix (same pattern: agent report → targeted fix)
- [m-dx25-typed-let-bindings](../../implemented/v0_5_5/m-dx25-typed-let-bindings.md) — Typed let binding generation

**Planned (check for overlap):**
- [m-wasm-dictionary-dispatch](../../planned/m-wasm-dictionary-dispatch.md) — Related WASM codegen bug (different backend)
- [m-codegen-ir-strategy](../v0_10_0/m-codegen-ir-strategy.md) — Future IR refactoring (not blocking)
- [m-codegen-v3-binding-hoisting](../v0_10_0/m-codegen-v3-binding-hoisting.md) — Future binding optimization

## References

- [Design Axioms](/docs/references/axioms)
- DocParse agent message: `ailang messages read 476f8495-4d66-4c4b-8e2c-2252ea8074cd`
- M-DX18 namespacing in `codegen_decl.go:115-129`

## Future Work

- **Go harness generation** (`--with-default-handlers`): Auto-generate effect handler implementations for common patterns (FS, Env, AI, IO)
- **Collision detection**: Add a global emitted-name registry that errors at generation time rather than at `go build` time
- **Multi-backend IR**: The v0.10.0 IR refactoring would eliminate this class of bugs structurally

---

**Document created**: 2026-03-17
**Last updated**: 2026-03-17
