# M-SMT-CROSS-MODULE-TYPES: Z3 Cross-Module Type Resolution

**Status**: Planned
**Target**: v0.10.0
**Priority**: P1 (Medium)
**Estimated**: 3-4 days (~24 hours implementation + testing)
**Dependencies**: M-SMT-FRAGMENT-EXPANSION (Phase A-E, complete), M-SMT-V2 (complete)
**Parent**: [design_docs/implemented/v0_8_0/m-smt-fragment-expansion.md](design_docs/implemented/v0_8_0/m-smt-fragment-expansion.md)
**Bug Report**: Regression from commit 8216408c (cross-module ADT discovery)
**Additional Reports**:
- ailang-parse msg `12839c7e` (2026-03-19) — field-name collision (Issue 6)
- ailang-parse msg `ce6e078e` (2026-04-14) — tex_parser.ail: 18/19 `ensures` clauses SKIPPED with "cross-module types not yet supported (Block ADT)" or "string operations not encodable (trim/toLower)". Confirms real-world DX impact: headline Z3 verification story does not apply to realistic multi-module parser code. Also surfaced a new gap — see **Issue 7** below.

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

### Issue 6: Field Name Collisions Across Record Types

**Severity**: Medium (blocks verification of functions using multiple record types with shared field names)

**Bug report**: docparse message 12839c7e (2026-03-19)

```smt
(declare-datatype CheckResult ((mk_CheckResult (applicable Bool) ...)))
(declare-datatype MetaAccum  ((mk_MetaAccum  (applicable Int) ...)))
;  Z3 error: unknown constant applicable (Int)
;  declared: (declare-fun applicable (CheckResult) Bool)
;  declared: (declare-fun applicable (MetaAccum) Int)
```

When two record types in the same module share a field name (e.g., `applicable` on both `CheckResult` and `MetaAccum`), Z3 emits two unqualified accessor functions with the same name but different signatures. Z3 cannot disambiguate which `applicable` is intended when the field is accessed in the function body.

This is perfectly valid AILANG — same-named fields on different record types is standard. The fix is to qualify field accessor names with the record type name: `CheckResult_applicable` and `MetaAccum_applicable`.

**Confirmed impact**: `evalComputeScore` in docparse `eval.ail` fails due to this. Workaround: rename one field (e.g., `MetaAccum.applicableCount`), but the encoder should handle this.

### Issue 7: String Builtins Not Encodable (`trim`, `toLower`, etc.)

**Severity**: Medium (blocks contracts on all text-processing code)

**Bug report**: ailang-parse msg `ce6e078e` (2026-04-14)

Functions whose `ensures` clauses reference `trim`, `toLower`, `toUpper`, `split`,
`charAt`, or similar `std/string` builtins are currently reported as SKIPPED with
"string operations not encodable" rather than verified or failed.

This is orthogonal to the cross-module ADT work but was surfaced by the same
real-world usage (tex_parser.ail). Text-processing code — which is a large fraction of
AILANG programs writing parsers/transformers — cannot currently benefit from Z3
verification.

**Scope clarification for this doc:** this issue is acknowledged here for visibility
but the fix belongs in a separate design doc (candidate: `m-smt-string-theory.md`
under v0.12.x). Z3's SMT-LIB theory of strings supports concat, substring, length,
indexing — enough to encode `charAt`, `length`, `substring`, and `++`. `trim`/`toLower`
require custom axioms or conservative over-approximation (`trim(s).length ≤ s.length`).

**Tracking:** Add a `@skip-reason` tag surfacing *which* builtin is unencodable, so
users can narrow contracts or refactor rather than guessing. This is a small,
independent improvement and SHOULD ship as part of this sprint's documentation
phase.

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

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Demand-driven type collection (per-function) vs emit-all-types | Determines whether cascade failures are structurally impossible or must be prevented by correct type declarations; demand-driven is more work but eliminates an entire failure class | human | design | high |
| Tarjan's SCC for mutual recursion detection | Commits to a specific cycle-detection algorithm; alternative (manual annotation of recursive groups) would be simpler but error-prone for users | agent | compile | med |
| `declare-datatypes` (plural) for recursive groups vs workaround (forward declarations) | Z3 has no forward declarations, so `declare-datatypes` is the only correct encoding; this is technically forced but has syntax complexity | compiler | compile | high |
| Parameter name prefix `$p_` to disambiguate from field accessors | Baked into all Z3 output; changing prefix later requires updating all Z3 assertion patterns and test expectations | agent | design | med |
| Type dependency graph built once per verify invocation vs per-function | Once-per-invocation is more efficient but requires careful invalidation if functions modify the graph; per-function is simpler but slower | agent | compile | low |
| Named record aliases stored in `RecordTypeAliases` map vs unified with ADT type registry | Separate map is simpler now but creates two parallel type registries; unifying later touches every Z3 codegen path | human | design | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] Demand-driven type collection is the approach (not fixing emit-all to handle all edge cases)
- [ ] `declare-datatypes` (plural, mutual recursion) is the encoding for recursive ADTs — no workaround
- [ ] Parameter prefix is `$p_` (not `param_` or other) — affects all Z3 variable references
- [ ] Record type aliases are a separate `RecordTypeAliases` map (not merged into the ADT registry)
- [ ] Topological sort for alias declaration ordering (not multi-pass with resolved-check or other strategy)
- [ ] Phase 1 (demand-driven) ships independently before Phase 2/3 — each phase must leave the system in a working state

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

**Phase 1: Demand-Driven Type Filtering** (~8 hours) — **DONE** (2026-03-19)
- [x] Walk function params, return type, and body to collect seed sorts
- [x] Compute transitive closure through ADT variant fields
- [x] Filter ADT types per-function in `verify.go` and `ai_check.go`
- [x] Test: pure int/string/bool functions no longer blocked by unrelated cross-module types
- [x] Confirmed: docparse 3→10 verified functions after fix
- Note: Implemented inline in `verify.go` (`filterADTTypesForFunction`) rather than as separate `type_deps.go`. Sufficient for current needs; can be extracted if reuse is needed.

**Phase 2: Named Record Aliases** (~6 hours)
- [ ] Resurrect stashed `collectNamedRecordAlias` and `RecordTypeAliases`
- [ ] Add topological ordering of alias declarations (multi-pass with resolved-check)
- [ ] Register field-set keys for body literal → named sort resolution
- [ ] Test: `type Point = {x: int, y: int}` → `(declare-datatype Point ((mk_Point (x Int) (y Int))))`
- [ ] Test: `type ParsedDocument = {metadata: DocMetadata, ...}` → declared after DocMetadata
- [ ] ~80 LOC new, ~30 LOC modified

**Phase 3: Mutual Recursion + Name Disambiguation** (~10 hours)
- [ ] Detect SCC groups in type dependency graph (Tarjan's algorithm)
- [ ] Implement `DeclareDatatypesMutual()` for `declare-datatypes` emission
- [ ] Handle Block + inline records as a mutual recursion group
- [ ] Prefix parameter names with `$p_` in `declare-const` (Issue 5)
- [ ] Update `EncodeExpr` variable lookup for prefixed names
- [ ] Qualify record field accessor names with record type: `RecordType_fieldName` (Issue 6)
- [ ] Update `encodeRecordAccess` to emit qualified accessor names matching declarations
- [ ] Test: Block with SectionBlock({blocks: [Block]}) compiles in Z3
- [ ] Test: `simpleCell(text: string)` → `(declare-const $p_text String)`, no ambiguity
- [ ] Test: `CheckResult.applicable` and `MetaAccum.applicable` → `CheckResult_applicable` / `MetaAccum_applicable`, no collision
- [ ] ~180 LOC new, ~50 LOC modified

### Files to Modify/Create

**Already modified (Phase 1 — DONE):**
- `cmd/ailang/verify.go` - `filterADTTypesForFunction()`, demand-driven per-function filtering (~180 LOC added)
- `cmd/ailang/ai_check.go` - Same filtering applied (~5 LOC)

**New files (Phases 2-3):**
- `internal/smt/codegen_mutual.go` - Mutual recursion `declare-datatypes` (~150 LOC)
- `internal/smt/codegen_mutual_test.go` - Tests (~100 LOC)

**Modified files (Phases 2-3):**
- `cmd/ailang/verify.go` - Record alias extraction (~50 LOC)
- `internal/smt/codegen.go` - RecordTypeAliases, param prefix `$p_`, qualified field accessors (~100 LOC)
- `internal/smt/codegen_records.go` - Skip anonymous sorts when named alias exists, qualified accessor names (~30 LOC)

**Remaining: ~250 LOC new code, ~180 LOC modifications**

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

### Example 5: Field Name Collision Across Record Types (Phase 3)

**Before (broken):**
```smt
(declare-datatype CheckResult ((mk_CheckResult (applicable Bool) ...)))
(declare-datatype MetaAccum  ((mk_MetaAccum  (applicable Int) ...)))
; ERROR: unknown constant applicable (Int)
; Z3 sees two 'applicable' functions with different signatures, can't disambiguate
```

**After:**
```smt
(declare-datatype CheckResult ((mk_CheckResult (CheckResult_applicable Bool) ...)))
(declare-datatype MetaAccum  ((mk_MetaAccum  (MetaAccum_applicable Int) ...)))
; OK: qualified accessor names, no ambiguity
```

Body references like `r.applicable` are emitted as `(CheckResult_applicable r)` when `r` is known to be a `CheckResult`.

## Success Criteria

- [x] Pure int/string/bool functions verify despite cross-module imports (Phase 1 — DONE)
- [x] docparse 3→10 verified functions after demand-driven filtering (Phase 1 — DONE)
- [ ] docparse `simpleCell`, `spanCell`, `mergedCell` VERIFIED (currently ERROR — needs Phase 2)
- [ ] docparse `emptyMetadata` VERIFIED (currently ERROR — needs Phase 2)
- [ ] docparse `countBlocks` VERIFIED (currently ERROR — needs Phase 3)
- [ ] docparse `headingLevelFromStyle` VERIFIED (currently ERROR — needs Phase 2)
- [ ] docparse `evalComputeScore` VERIFIED (currently ERROR — needs Phase 3, Issue 6)
- [ ] docparse `format_router.ail` maintains 3/3 VERIFIED (no regression)
- [ ] No cascade errors from unused type declarations
- [ ] No field accessor collisions across record types (qualified names)
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

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact seed-type extraction strategy for function bodies (AST walk vs type-annotation-only vs conservative over-approximation) — [agent may resolve]
- Whether the type dependency graph is exposed as a reusable API or kept internal to the verify command — [agent may resolve]
- How to handle XmlNode (builtin Go TaggedValue) in Z3 encoding — requires mapping Go types to Z3 datatypes; mechanism TBD — [human may resolve in future version]
- Whether SCC groups are cached across functions or recomputed per-function — [agent may resolve]
- Format for `--verbose` output of demand-driven type decisions (which types included/excluded per function) — [agent may resolve]
- Whether the stashed record alias prototype is resurrected as-is or rewritten to fit the new demand-driven architecture — [agent may resolve]

## Non-Goals

- **Full mutual recursion support for arbitrary depth** — Only handle the common case (ADT ↔ inline record). Deeply nested mutual recursion is rare in practice.
- **Z3 `declare-datatypes` for user-declared mutually recursive types** — AILANG doesn't yet support explicit mutual recursion declarations.
- **Automatic contract inference** — This doc focuses on fixing existing contract verification, not inferring new contracts.

## Timeline

**Phase 1: COMPLETE** (2026-03-19):
- Demand-driven ADT filtering in verify.go and ai_check.go
- Result: docparse 3→10 verified functions

**Day 1** (~6 hours):
- Phase 2: Named record alias extraction
- Expected result: simpleCell/spanCell/mergedCell/emptyMetadata/headingLevelFromStyle VERIFIED

**Day 2** (~10 hours):
- Phase 3: Mutual recursion groups + parameter disambiguation + field accessor qualification
- Expected result: countBlocks VERIFIED, evalComputeScore VERIFIED, all name collisions eliminated

**Day 3** (~2 hours):
- Integration testing with full docparse project
- Documentation and examples

**Remaining: ~18 hours across 2-3 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Tarjan's SCC complexity in deeply nested types | Low | AILANG types rarely exceed 3 levels of nesting |
| Parameter prefix breaking existing variable resolution | Medium | Comprehensive test coverage, rename only at declaration site |
| Z3 `declare-datatypes` syntax edge cases | Medium | Validate against Z3 4.15 specifically, add --verbose output |
| Demand analysis missing required types | High | Conservative: include type if in doubt, reject at Z3 level |
| Performance: building dependency graph per-function | Low | Graph built once, queries are O(types_used) |
| Qualified accessor names breaking body encoding | Medium | Must update all `encodeRecordAccess` paths to emit `RecordType_field` consistently |

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
**Last updated**: 2026-04-14 (added Issue 7: string builtins not encodable; referenced msg ce6e078e)

## Documentation Action (interim, before full fix)

Until Phases 2–3 ship, **users need clear visibility into what Z3 *can* and *cannot*
verify today**, because the headline verification story currently over-promises relative
to what works on realistic multi-module code.

**Required (can ship immediately, ~1 hour):**
- [ ] Update `docs/docs/guides/verification/` with a "Current Limitations" section citing:
  cross-module ADT types, recursive ADTs, string builtins (trim/toLower), field
  collisions — each with a one-line example and expected SKIP message
- [ ] `ailang verify --verbose` must print the exact reason for each SKIP (which type,
  which builtin) — not just "not yet supported"
- [ ] Teaching prompt / `ailang prompt` should mention the limitation when the user
  asks about contracts on parser/string-processing code
