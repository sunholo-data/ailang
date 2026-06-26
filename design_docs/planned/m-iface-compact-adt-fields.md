# M-IFACE-COMPACT-ADT-FIELDS: Compact Interface Carries ADT Constructor Fields

**Status**: Planned
**Target**: v0.25.1
**Priority**: P1 (defeats a shipped feature — AST iface-compaction — for ADT-heavy agent tasks)
**Estimated**: 1 day
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Rendering only; output already normalized/sorted |
| A2: Replayability | 0 | No trace impact |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | 0 | No authority changes |
| A5: Bounded Verification | +1 | A complete iface lets a consumer construct the ADT locally without reading full source |
| A6: Safe Concurrency | 0 | None |
| A7: Machines First | +1 | The compact iface exists FOR machines (token-efficient context); dropping ctor fields makes it unusable for ADT construction — this restores the feature's purpose |
| A8: Minimal Syntax | 0 | No new language syntax; renders existing record-type surface form |
| A9: Cost Visibility | 0 | None |
| A10: Composability | 0 | None |
| A11: Structured Failure | 0 | None |
| A12: System Boundary | 0 | None |

**Net Score: +2** → **Decision: Move forward**

### Hard Violation Check
- [x] A1 (Determinism): output stays sorted/canonical
- [x] A3 (Effects): none
- [x] A4 (Authority): none
- [x] A7 (Machines First): improves machine legibility (the whole point)

## Problem Statement

`ailang iface --compact` produces the compact interface that motoko's `MOTOKO_AST_AUTOREAD`
serves when an agent reads a *dependency* `.ail` module — the token-efficient substitute for
the full source (`internal/iface` → `cmd/ailang/check.go:compactInterface`). It has two defects
that make it useless for ADT-heavy tasks:

1. **Sum-type constructors are rendered as bare names** — fields dropped.
2. **Record types leak `<*types.TRecord>`** — a Go internal type name, not valid AILANG.

**Current State (verified `ailang iface --compact` on a minimal ADT):**
```
type Shape = Circle | Rect                  # fields {radius: float} / {w,h: float} dropped
simpleCell : (string)-><*types.TRecord>     # (from the real docparse/types/document.ail iface)
```
Underlying JSON: `"ctors": ["Circle", "Rect"]` — names only (verified).

**Root cause (all in AILANG core):**
1. `internal/iface/json.go` `formatTypeCanonical` has no `case *types.TRecord`; the `default`
   returns `fmt.Sprintf("<%T>", t)` → `<*types.TRecord>`.
2. The iface `ConstructorInfo` (`internal/iface/builder.go:248`) carries only `TypeName/CtorName/Arity`
   — it **drops** `FieldTypes`, which the *elaborate* `ConstructorInfo` (`internal/elaborate/core.go:56`)
   already captures (`FieldTypes []types.Type` — "Actual field types from AST").
3. `internal/iface/json.go` `TypeJSON.Ctors` is `[]string` (names only).
4. `cmd/ailang/check.go:650` `compactInterface` joins the bare ctor names with `" | "`.

**Impact:**
- A model implementing a parser whose job is to **construct ADT variants** (e.g. the DOCX/Block
  parser) cannot learn the constructor shapes from the iface. It falls back to `cat`-ing the full
  source. **Observed: 11 `cat` fallbacks in a single docx eval run** (the iface-compaction feature
  net *added* work: it served an incomplete view AND the model read full source anyway).
- This is the 7th instance in this mission of a wall being a harness/tooling defect, not the model.

## Goals

**Primary Goal:** The compact interface carries enough type information to *construct* an ADT —
constructor field signatures and properly-rendered record types — so an agent never needs the
full source for the public shape.

**Success Metrics:**
- `ailang iface --compact` on the minimal ADT renders `type Shape = Circle({radius: float}) | Rect({w: float, h: float})`.
- No `<*types.TRecord>` (or any `<*...>`) appears in any `ailang iface` output.
- A re-run docx eval shows the `cat`-fallback count drop materially (target: <3, was 11).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Enrich the normalized iface JSON with ctor field types (vs compact-output-only) | Changes the interface **hash/digest** for every ADT/record-bearing module (one-time churn) | compiler | design | high |
| Render record types in `formatTypeCanonical` (shared by ALL func signatures) | Every exported record-typed function signature changes from `<*types.TRecord>` to `{...}` | compiler | design | med |
| Compact ctor syntax form: `Ctor({label: type})` mirroring surface syntax | Affects what the consuming model parses | agent | compile | low |

### Design Freeze

- [x] Enrich the JSON (Decision 1): **chosen**. Constructor field types ARE part of the public
  interface — two modules differing only in a ctor's fields are genuinely different interfaces, so
  the digest *should* reflect them. Today's digest omitting them is a latent correctness gap; fixing
  it is correct and the churn is one-time. (If churn proves unacceptable, fall back to compact-only.)
- [x] Render TRecord in the shared formatter (Decision 2): **chosen** — it's the single root fix that
  resolves both the leak and ctor-field rendering.

## Solution Design

### Overview
One shared root fix (render `TRecord`) plus threading the already-captured `FieldTypes` from the
elaborator into the iface and out through the JSON + compact renderer.

### Architecture

**Components:**
1. **`formatTypeCanonical` gains `case *types.TRecord`** (`internal/iface/json.go`): render
   `{label: type, ...}` with sorted labels, **cycle-safe** (records can be recursive — pass a
   `visited`/depth guard per the type-system traversal rule). Fixes the `<*types.TRecord>` leak for
   all signatures and provides the renderer for ctor fields.
2. **Thread `FieldTypes`** (`internal/iface/builder.go`): add `FieldTypes []types.Type` to the iface
   `ConstructorInfo`; populate it from the elaborate `ConstructorInfo.FieldTypes` when building.
3. **JSON + compact rendering** (`internal/iface/json.go` + `cmd/ailang/check.go`): `TypeJSON` ctors
   become `{name, fields}` (fields = rendered type string); `compactInterface` renders
   `Ctor({...})` / `Ctor(t1, t2)` per arity.

### Conflict Surface

**This change touches `internal/iface/`, `internal/types/` (rendering), `internal/elaborate/` (read FieldTypes), `cmd/ailang/`.**

1. **Positions extended:**
   - `formatTypeCanonical` — the type renderer shared by **every** exported function signature in **every** iface (not just constructors).
   - `TypeJSON` schema (the normalized JSON) and the `--compact` text format.
2. **Other constructs already in those positions:**
   - **Every record-returning/-taking exported function** currently renders `<*types.TRecord>`; after the fix it renders `{...}`. This is a behavior change for *all* such signatures, not only ADT ctors (an improvement, but a change to existing output).
   - **The normalized JSON feeds the interface hash/digest** (`internal/iface` digest; tested in `internal/iface/constructor_test.go` — `TestDigestWithConstructors`, `TestDigestDifferentConstructors`). Enriching ctors changes those digests.
3. **Disambiguation:** N/A — not a parser/lexer change; no new ambiguous syntactic position.
4. **Existing programs/tests that MUST still work (fixtures):**
   - `internal/iface/constructor_test.go` digest tests — **golden hashes will change** (must be regenerated; the *invariants* — same ctors ⇒ same digest, different ctors ⇒ different digest — must still hold).
   - Package resolution comparing interface hashes — a **one-time** "interface changed" for ADT/record-bearing modules on upgrade.
   - `ailang iface` JSON consumers (the compact renderer, `mcp__ailang-parse` if it parses iface JSON) — additive field, must remain backward-readable (keep `ctors` semantics; add fields without removing the name).
5. **Deliberate (intentional) changes:** interface hashes for ADT/record modules **will** change once — this is correct (ctor field types are part of the public interface; the old hash under-counted interface identity).

### Implementation Plan

**Phase 1: Render records in the shared formatter** (~2h)
- [ ] Add `case *types.TRecord` to `formatTypeCanonical` → `{label: t, ...}`, sorted labels, cycle-safe (visited set or depth cap; do NOT call `.String()`).
- [ ] Handle open/row-polymorphic records (render known labels; mark the open tail).
- [ ] Unit test: a record-returning function no longer leaks `<*...>`.

**Phase 2: Thread FieldTypes into the iface** (~3h)
- [ ] Add `FieldTypes []types.Type` to iface `ConstructorInfo` (`builder.go`).
- [ ] Populate from elaborate `ConstructorInfo.FieldTypes` where the iface is built.
- [ ] Extend `TypeJSON` ctor entries to `{name, fields}` (rendered field-type string), preserving determinism (sorted).

**Phase 3: Compact rendering + tests + goldens** (~3h)
- [ ] `compactInterface` (`check.go`): render `Ctor({...})` / `Ctor(t)` instead of bare name.
- [ ] Update/extend `internal/iface/constructor_test.go` (new field-rendering tests; regenerate digest goldens; keep the same-vs-different invariants).
- [ ] `make quick-install`; verify on the minimal ADT and on `docparse/types/document.ail` (the real case).

### Files to Modify/Create

**Modified:**
- `internal/iface/json.go` — `formatTypeCanonical` TRecord case + `TypeJSON` ctor schema + `ToNormalizedJSON` ctor population (~45 LOC)
- `internal/iface/builder.go` — `ConstructorInfo.FieldTypes` + population (~20 LOC)
- `cmd/ailang/check.go` — `compactInterface` ctor rendering (~12 LOC)
- `internal/iface/constructor_test.go` — new tests + regenerated digest goldens (~50 LOC)

## Examples

### Example 1: Sum type with record constructors

**Before** (verified):
```
type Shape = Circle | Rect
```

**After:**
```
type Shape = Circle({radius: float}) | Rect({w: float, h: float})
```

### Example 2: Record-returning function (the `<*types.TRecord>` leak)

**Before** (from `docparse/types/document.ail` iface):
```
simpleCell : (string)-><*types.TRecord>
```

**After:**
```
simpleCell : (string)->{value: string, colspan: int, rowspan: int}
```

## Success Criteria

- [ ] `ailang iface --compact` on the minimal ADT shows ctor field signatures (acceptance: exact string match).
- [ ] No `<*types.TRecord>` / `<*...>` in any `ailang iface` output (grep across stdlib ifaces).
- [ ] `docparse/types/document.ail` iface shows `Block` ctor fields and proper `TableCell`/metadata records.
- [ ] `internal/iface` tests pass with regenerated digest goldens; same-ctors/different-ctors invariants hold.
- [ ] All tests passing (`make test`).
- [ ] Documentation updated (CHANGELOG; iface reference if present).
- [ ] (Validation) docx eval re-run: `cat`-fallback count < 3 (was 11).

## Testing Strategy

**Unit tests:**
- `formatTypeCanonical` renders closed records, nested records, and recursive records without hang or `<*...>`.
- Constructor field rendering for record-style (`Ctor({...})`) and positional (`Ctor(t1, t2)`) ctors.
- Digest: identical ctor fields ⇒ identical digest; differing field *types* ⇒ differing digest (new invariant the fix enables).

**Integration tests:**
- `ailang iface --compact` on `internal/iface` testdata + a stdlib module containing an ADT.

**Manual / eval:**
- Re-run the docx eval with the rebuilt binary; compare `cat`-fallback count and step-to-converge vs the 11-fallback baseline.

## Deferred Decisions

- Exact compact syntax for **positional** constructors (`Ctor(int, string)`) vs record constructors (`Ctor({...})`) — agent may choose, matching AILANG surface syntax.
- Whether to canonicalize type-parameter names inside ctor field types (e.g. `Some(a)` for `Option[a]`) — agent may choose; must stay deterministic.

## Non-Goals

- Changing motoko's `MOTOKO_AST_AUTOREAD` gate or `MOTOKO_AST_READ_FULL` policy (harness side) — out of scope.
- Structural / AI compaction behavior — unrelated.
- Adding the full constructor *bodies* to the iface — the iface stays a signature, not an implementation.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Interface hash churn for ADT/record modules | Med | One-time + semantically correct; regenerate goldens; note in CHANGELOG; ships in a version bump |
| `formatTypeCanonical` hangs on recursive records | Med | Cycle-safe traversal (visited set/depth cap) per type-system rule; explicit recursive-record test |
| Row-polymorphic (open) records render ambiguously | Low | Render known labels + explicit open-row marker; test |
| Downstream JSON consumers break on schema change | Low | Additive change (keep `ctors` name semantics); MCP/parse consumers read the name field unchanged |

## Related Documents

**Implemented (may inform design):**
- [m-dx29-option-nested-adt-type](design_docs/implemented/v0_6_0/m-dx29-option-nested-adt-type.md) (0.35) — nested ADT type handling; distinct (this is iface rendering, not type inference)
- [m-dx12-typed-adt-slices](design_docs/implemented/v0_5_3/m-dx12-typed-adt-slices.md) (0.35) — ADT codegen; distinct surface
- [m-dx18-recordaccess-typed-structs](design_docs/implemented/v0_5_4/m-dx18-recordaccess-typed-structs.md) (0.34) — record typing; related (record rendering) but different layer

**Planned:** none overlapping (top match 0.34, below warn threshold).

## References

- [Design Axioms](/docs/references/axioms)
- `internal/iface/json.go`, `internal/iface/builder.go`, `internal/elaborate/core.go:56`, `cmd/ailang/check.go:631`
- Motoko AST-autoread: `mk-ast/src/core/tool_runtime.ail:483` (`exec("ailang", ["iface", "--compact", resolved])`)

## Future Work

- Consider serving the compact iface for the *target* file too (currently `MOTOKO_AST_READ_FULL`), once it's lossless enough.

---

**Document created**: 2026-06-26
**Last updated**: 2026-06-26

DESIGN_DOC_PATH: design_docs/planned/m-iface-compact-adt-fields.md
