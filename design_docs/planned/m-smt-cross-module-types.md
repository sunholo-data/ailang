# M-SMT-CROSS-MODULE-TYPES: Z3 Cross-Module Type Resolution

**Status**: Planned
**Target**: v0.10.0
**Priority**: P1 (Medium)
**Estimated**: 3-4 days (~24 hours implementation + testing)
**Dependencies**: M-SMT-FRAGMENT-EXPANSION (Phase A-E, complete), M-SMT-V2 (complete)
**Parent**: [design_docs/implemented/v0_8_0/m-smt-fragment-expansion.md](design_docs/implemented/v0_8_0/m-smt-fragment-expansion.md)
**Bug Report**: Regression from commit 8216408c (cross-module ADT discovery)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No change to language semantics |
| A2: Replayability | +1 | More functions verifiable = more reproducible guarantees |
| A3: Effect Legibility | 0 | No change to effect system |
| A4: Explicit Authority | 0 | No change to capabilities |
| A5: Bounded Verification | +1 | Expands the set of automatically verifiable functions |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Better toolability — AI agents can verify more contracts |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | 0 | No change to cost model |
| A10: Composability | +1 | Cross-module types compose correctly in verification |
| A11: Structured Failure | 0 | Errors remain structured |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Improves machine analysis coverage

## Problem Statement

When `ailang verify` processes a file that imports types from other modules, the Z3 SMT-LIB encoding fails for multiple interconnected reasons. This was exposed as a regression when cross-module ADT discovery was added (commit 8216408c) and tested against the docparse project.

**Current State (v0.9.1):**
- docparse project: 3/28 functions verified (down from 15/28 pre-regression)
- All functions in files that import cross-module types fail with cascading Z3 errors
- `format_router.ail` (no cross-module types) still verifies correctly (3/3)
- 25 ERROR, 5 SKIPPED across docparse modules

**Root Causes Identified (5 distinct issues):**

### Issue 1: Record Type Aliases Not Declared

**Severity**: Critical (blocks all dependent types)

```ailang
// types/document.ail
type TableCell = {text: string, colSpan: int, rowSpan: int, merged: bool}
```

`extractADTTypesWithRecords()` only handles `*ast.AlgebraicType` definitions. Record type aliases (`TypeDecl` where `Definition` is `*ast.RecordType`) are completely ignored. Z3 output references `TableCell` as a sort but never declares it.

**Partial fix exists** (stashed): Adds `collectNamedRecordAlias()` and `RecordTypeAliases` map. Works for simple cases but doesn't handle dependencies.

### Issue 2: Recursive ADT Circular Dependencies

**Severity**: Critical (fundamental Z3 encoding issue)

```ailang
type Block = TextBlock({text: string, style: string, level: int})
           | SectionBlock({kind: string, blocks: [Block]})  // ← recursive!
```

The current encoding uses individual `declare-datatype` for each type. Block and its inline record `Record_blocks_kind` create a circular dependency:
- `Record_blocks_kind` references `Block` (via `blocks: (Seq Block)`)
- `Block` references `Record_blocks_kind` (via `SectionBlock` constructor)

Z3 requires `declare-datatypes` (plural, mutual recursion) for this pattern.

### Issue 3: Preamble Pollution

**Severity**: High (causes cascade failures)

ALL ADT types from ALL imported modules are dumped into every function's Z3 preamble, even when the function doesn't use them. When any single type declaration fails (e.g., `Block` due to Issue 2), it cascades:

```
simpleCell(text: string) -> TableCell    // Only needs TableCell
  ← But gets Block, ParsedDocument, ExtractionResult, XmlNode, ...
  ← Block fails → ALL subsequent references to Block fail
```

### Issue 4: Declaration Ordering

**Severity**: Medium (solvable with topological sort)

Record type aliases may reference other aliases or ADTs:
```ailang
type DocMetadata = {title: string, ...}
type ParsedDocument = {metadata: DocMetadata, blocks: [Block]}  // depends on both
type ExtractionResult = {document: ParsedDocument, ...}         // depends on ParsedDocument
```

Current emission order is non-deterministic (Go map iteration). Need topological sort.

### Issue 5: Parameter-Accessor Name Collisions

**Severity**: Low (affects specific functions)

```smt
(declare-datatype TableCell ((mk_TableCell (text String) ...)))
(declare-const text String)  ; ← parameter name
;  Z3 error: ambiguous constant reference 'text'
```

When a parameter name matches a record field accessor name, Z3 can't disambiguate.

**Impact:**
- Any project using cross-module record type aliases and ADTs cannot verify contracts
- docparse (primary external user) blocked on Z3 verification
- Multi-module projects are the target audience for contract verification

## Goals

**Primary Goal:** Enable Z3 contract verification for functions that use cross-module record type aliases and ADTs, increasing docparse verified functions from 3 to 15+.

**Success Metrics:**
- docparse `types/document.ail`: simpleCell, spanCell, mergedCell, emptyMetadata, countBlocks all VERIFIED
- docparse `services/format_router.ail`: maintains 3/3 VERIFIED (no regression)
- docparse `services/docx_parser.ail`: headingLevelFromStyle, getCellGridSpan VERIFIED
- Total docparse verified: 15+ (from current 3)
- Zero cascade errors from unrelated type declarations

## Solution Design

### Overview

Three-phase approach, each independently shippable:

1. **Demand-driven type collection**: Only declare types actually used by the function being verified
2. **Named record alias support**: Proper extraction and declaration of `type X = {fields}`
3. **Mutual recursion support**: Use `declare-datatypes` for recursive ADTs + their inline records

### Architecture

**Current flow:**
```
verify.go                          codegen.go
  extractADTTypesWithRecords()       Step 0.5: ExtraDeclarations (all)
    → ALL ADTs from ALL modules      Step 1: ALL ADT declarations
    → ALL inline records             → Cascade failures
```

**Proposed flow:**
```
verify.go                          codegen.go
  extractTypesFromFile()             Step 0: Named record aliases (demand-driven)
    → ADTs + record aliases          Step 0.5: Inline records (demand-driven)
    → Per-function type sets         Step 1: ADT declarations (demand-driven)
                                     Step 1.1: Mutual recursive groups
  computeFunctionTypeDeps()          → Only declare what's needed
    → Transitive closure of
      types used by each function
```

**Components:**

1. **Type Dependency Graph** (`internal/smt/type_deps.go`, ~200 LOC)
   - Build graph of type → type dependencies from AST
   - Compute transitive closure for a given set of "seed" types (function params, return type)
   - Detect mutual recursion cycles
   - ~200 LOC

2. **Demand-Driven Declaration Filter** (`cmd/ailang/verify.go` changes, ~100 LOC)
   - For each function being verified, compute which types it needs
   - Pass only those types to `EncodeFunction`
   - Eliminates preamble pollution

3. **Named Record Alias Extraction** (`cmd/ailang/verify.go` changes, ~80 LOC)
   - Handle `TypeDecl` where `Definition` is `*ast.RecordType`
   - Pass as `RecordTypeAliases` map to encoder
   - Register field-set keys so body literals resolve to named sorts
   - Partial implementation exists (stashed)

4. **Mutual Recursion Groups** (`internal/smt/codegen_mutual.go`, ~150 LOC)
   - Detect cycles in type dependency graph (Tarjan's SCC)
   - For mutually recursive groups, emit `declare-datatypes` (plural)
   - Non-recursive types use existing `declare-datatype` (singular)

5. **Parameter Name Disambiguation** (`internal/smt/codegen.go` change, ~20 LOC)
   - Prefix parameter `declare-const` names with `$p_` to avoid accessor collisions
   - Update `EncodeExpr` variable resolution to use prefixed names

### Implementation Plan

**Phase 1: Demand-Driven Type Filtering** (~8 hours)
- [ ] Create `internal/smt/type_deps.go` with `TypeDependencyGraph`
- [ ] Implement `ComputeRequiredTypes(seedSorts []string) map[string]bool`
- [ ] Walk function params, return type, and body to collect seed sorts
- [ ] Filter ADT types and extra declarations per-function in `verify.go`
- [ ] Test: simpleCell should only declare TableCell (no Block in preamble)
- [ ] Test: format_router functions unaffected (no imported types)
- [ ] ~200 LOC new, ~50 LOC modified

**Phase 2: Named Record Aliases** (~6 hours)
- [ ] Resurrect stashed `collectNamedRecordAlias` and `RecordTypeAliases`
- [ ] Add topological ordering of alias declarations (multi-pass with resolved-check)
- [ ] Register field-set keys for body literal → named sort resolution
- [ ] Test: `type Point = {x: int, y: int}` → `(declare-datatype Point ((mk_Point (x Int) (y Int))))`
- [ ] Test: `type ParsedDocument = {metadata: DocMetadata, ...}` → declared after DocMetadata
- [ ] ~80 LOC new, ~30 LOC modified

**Phase 3: Mutual Recursion + Parameter Disambiguation** (~8 hours)
- [ ] Detect SCC groups in type dependency graph (Tarjan's algorithm)
- [ ] Implement `DeclareDatatypesMutual()` for `declare-datatypes` emission
- [ ] Handle Block + inline records as a mutual recursion group
- [ ] Prefix parameter names with `$p_` in `declare-const`
- [ ] Update `EncodeExpr` variable lookup for prefixed names
- [ ] Test: Block with SectionBlock({blocks: [Block]}) compiles in Z3
- [ ] Test: `simpleCell(text: string)` → `(declare-const $p_text String)`, no ambiguity
- [ ] ~150 LOC new, ~40 LOC modified

### Files to Modify/Create

**New files:**
- `internal/smt/type_deps.go` - Type dependency graph and demand analysis (~200 LOC)
- `internal/smt/type_deps_test.go` - Tests for dependency graph (~150 LOC)
- `internal/smt/codegen_mutual.go` - Mutual recursion `declare-datatypes` (~150 LOC)
- `internal/smt/codegen_mutual_test.go` - Tests (~100 LOC)

**Modified files:**
- `cmd/ailang/verify.go` - Record alias extraction, per-function type filtering (~130 LOC)
- `cmd/ailang/ai_check.go` - Same changes as verify.go (~40 LOC)
- `internal/smt/codegen.go` - RecordTypeAliases, demand-driven Step 0/1, param prefix (~80 LOC)
- `internal/smt/codegen_records.go` - Skip anonymous sorts when named alias exists (~15 LOC)

**Total: ~600 LOC new code, ~265 LOC modifications**

## Examples

### Example 1: Record Type Alias (Phase 2)

**Before (broken):**
```smt
; TableCell never declared — only AlgebraicType definitions extracted
(declare-datatype Block ((TableBlock (TableBlock_0 (Seq TableCell)))))
; ERROR: unknown sort 'TableCell'
```

**After:**
```smt
(declare-datatype TableCell ((mk_TableCell (colSpan Int) (merged Bool) (rowSpan Int) (text String))))
(declare-datatype Block ((TableBlock (TableBlock_0 (Seq TableCell)))))
; OK: TableCell declared before Block references it
```

### Example 2: Demand-Driven Filtering (Phase 1)

**Before (preamble pollution):**
```smt
; Verification of simpleCell (only needs TableCell!)
(declare-datatype ParsedDocument ...)  ; ← UNUSED, fails because references Block
(declare-datatype Block ...)           ; ← UNUSED, fails because of circular refs
(declare-datatype Record_blocks_kind ...)  ; ← UNUSED
(declare-const text String)
(define-const result TableCell ...)
```

**After:**
```smt
; Verification of simpleCell — only types it actually uses
(declare-datatype TableCell ((mk_TableCell (colSpan Int) (merged Bool) (rowSpan Int) (text String))))
(declare-const text String)
(define-const result TableCell (mk_TableCell 1 false 1 text))
```

### Example 3: Mutual Recursion (Phase 3)

**Before (circular dependency):**
```smt
(declare-datatype Record_blocks_kind ...)  ; ERROR: references Block (not yet declared)
(declare-datatype Block ...)               ; ERROR: Record_blocks_kind already failed
```

**After:**
```smt
(declare-datatypes (
  (Record_blocks_kind 0)
  (Block 0)
) (
  ((mk_Record_blocks_kind (blocks (Seq Block)) (kind String)))
  ((TextBlock (TextBlock_0 Record_level_style_text))
   (SectionBlock (SectionBlock_0 Record_blocks_kind))
   ...)
))
```

### Example 4: Parameter Disambiguation (Phase 3)

**Before:**
```smt
(declare-datatype TableCell ((mk_TableCell (text String) ...)))
(declare-const text String)  ; ← ambiguous with accessor
```

**After:**
```smt
(declare-datatype TableCell ((mk_TableCell (text String) ...)))
(declare-const $p_text String)  ; ← prefixed, no ambiguity
```

## Success Criteria

- [ ] docparse `simpleCell`, `spanCell`, `mergedCell` VERIFIED (currently ERROR)
- [ ] docparse `emptyMetadata` VERIFIED (currently ERROR)
- [ ] docparse `countBlocks` VERIFIED (currently ERROR)
- [ ] docparse `headingLevelFromStyle` VERIFIED (currently ERROR)
- [ ] docparse `format_router.ail` maintains 3/3 VERIFIED (no regression)
- [ ] No cascade errors from unused type declarations
- [ ] Recursive ADTs (Block with SectionBlock) encode correctly
- [ ] All existing AILANG tests pass (3400+)
- [ ] Linting clean (0 issues)
- [ ] Examples added: `examples/runnable/contracts_cross_module.ail`

## Testing Strategy

**Unit tests:**
- `type_deps_test.go`: dependency graph construction, transitive closure, SCC detection
- `codegen_mutual_test.go`: `declare-datatypes` output format for mutual recursion groups
- Record alias extraction: simple aliases, nested aliases, aliases with ADT fields

**Integration tests:**
- Synthetic test files mirroring docparse patterns (TableCell + Block + ParsedDocument)
- Cross-module imports with record type aliases
- Recursive ADTs with inline record constructors

**End-to-end testing:**
- docparse project `ailang verify` on all `.ail` files
- Regression check: `examples/runnable/contracts_*.ail` all still pass

## Non-Goals

- **Full mutual recursion support for arbitrary depth** — Only handle the common case (ADT ↔ inline record). Deeply nested mutual recursion is rare in practice.
- **Z3 `declare-datatypes` for user-declared mutually recursive types** — AILANG doesn't yet support explicit mutual recursion declarations.
- **XmlNode verification** — XmlNode is a builtin Go type (TaggedValue), not a user-defined ADT. Its Z3 encoding requires special handling (future work).
- **Automatic contract inference** — This doc focuses on fixing existing contract verification, not inferring new contracts.

## Timeline

**Day 1** (~8 hours):
- Phase 1: Type dependency graph + demand-driven filtering
- Expected result: simpleCell/spanCell/mergedCell VERIFIED

**Day 2** (~6 hours):
- Phase 2: Named record alias extraction
- Expected result: emptyMetadata VERIFIED

**Day 3** (~8 hours):
- Phase 3: Mutual recursion groups + parameter disambiguation
- Expected result: countBlocks VERIFIED, all cascade errors eliminated

**Day 4** (~2 hours):
- Integration testing with full docparse project
- Documentation and examples

**Total: ~24 hours across 3-4 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Tarjan's SCC complexity in deeply nested types | Low | AILANG types rarely exceed 3 levels of nesting |
| Parameter prefix breaking existing variable resolution | Medium | Comprehensive test coverage, rename only at declaration site |
| Z3 `declare-datatypes` syntax edge cases | Medium | Validate against Z3 4.15 specifically, add --verbose output |
| Demand analysis missing required types | High | Conservative: include type if in doubt, reject at Z3 level |
| Performance: building dependency graph per-function | Low | Graph built once, queries are O(types_used) |

## Related Documents

**Implemented (directly related):**
- [design_docs/implemented/v0_8_0/m-smt-fragment-expansion.md](design_docs/implemented/v0_8_0/m-smt-fragment-expansion.md) — Phase A-E (parent doc)
- [design_docs/implemented/v0_5_10/m-codegen-cross-module-impl.md](design_docs/implemented/v0_5_10/m-codegen-cross-module-impl.md) — Cross-module codegen patterns
- [design_docs/planned/m-smt-fragment-expansion-v2.md](design_docs/planned/m-smt-fragment-expansion-v2.md) — Phase 2 expansion (just completed)

**Planned (check for overlap):**
- [design_docs/planned/v0_10_0/m-dx-tapp-trecord-unification.md](design_docs/planned/v0_10_0/m-dx-tapp-trecord-unification.md) — TApp~TRecord unification bug (related but orthogonal)

## References

- [Design Axioms](/docs/references/axioms)
- [Z3 SMT-LIB2 Reference](https://smtlib.cs.uiowa.edu/papers/smt-lib-reference-v2.6-r2021-05-12.pdf) — `declare-datatypes` syntax (Section 3.9.1)
- Regression commit: `8216408c` (cross-module ADT discovery)
- Partial fix: `git stash` on dev branch (record alias extraction prototype)

## Future Work

- **Builtin ADT Z3 encoding** (XmlNode, Option with custom payloads) — requires mapping Go TaggedValue to Z3 datatypes
- **Automatic type pruning** — Remove unused type parameters from Z3 output to reduce solver burden
- **Incremental verification** — Cache Z3 results per-function, only re-verify when function or its deps change

---

**Document created**: 2026-03-12
**Last updated**: 2026-03-12
