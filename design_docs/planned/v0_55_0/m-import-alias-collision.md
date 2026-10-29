# M-IMPORT-ALIAS-COLLISION: two selective imports of the same type name are a loud coded error, not last-import-wins

**Status**: Planned (design only — triage iteration 16 for issue #1980)
**Target**: v0.55.x (Phase 1); Phase 2 is a follow-up sprint with its own doc
**Priority**: P1 (order-dependent silent miscompile of imported type aliases; live downstream workaround in stapledons-godot)
**Estimated**: Phase 1: 2–3 days incl. corpus sweep. Phase 2 (policy sketched, deferred): 1 week, own doc.
**Dependencies**: None blocking. Lands on the shipped M-TYPE-NAME-SHADOW (M1/M3), M-ALIAS-BODY-CLOSURE and M-ELABORATOR-LEXICAL-SCOPE foundations; see "Neighboring designs" for how each bounds this one.
**Reporter**: stapledons_godot via ailang messages (issue #1980, v0.54.18 darwin arm64, reproduced by triage on origin/dev `51b00832`, binary `a6b4c2c` v0.54.19, stdlib from the same SHA).

## Problem Statement

Issue #1980: `import ./b (Leg)` and `import ./a (Leg)` in one module are both accepted; the
type-alias binding for the bare name `Leg` is whichever import comes **last**, silently. The
program then type-checks against an alias body the author may not have intended, and the only
diagnostic is a field-count mismatch in an unrelated function, or no diagnostic at all.

### Reproduction, verified live (Verification Log row 1)

Package `repro/dup` (a: `type Leg = {x,y,z,w: float}` 4 fields; b: `type Leg = {id, from, port,
t0, legT, flying}` 6 fields; main imports both, `export pure func mk() -> Leg = <literal>`).
All four cells run with `AILANG_NO_CACHE=1 ailang check --package .` on the binary named above:

| # | Import order | Literal shape | Result |
|---|--------------|---------------|--------|
| 1 | `import ./b (Leg)` then `import ./a (Leg)` | b's (6 fields) | ✗ `record field mismatch: expected 6 fields, got 4` — **the issue's repro, verbatim** |
| 2 | `import ./a (Leg)` then `import ./b (Leg)` | b's (6 fields) | ✓ passes — **the issue's "other order passes"** |
| 3 | `import ./a (Leg)` then `import ./b (Leg)` | a's (4 fields) | ✗ `expected 4 fields, got 6` |
| 4 | `import ./b (Leg)` then `import ./a (Leg)` | a's (4 fields) | ✓ passes |

Mechanism (row 2): the return annotation `-> Leg` is a bare `TCon` expanded through the
module's flat alias environment. In `resolveSelectiveImports`
(`internal/pipeline/pipeline_module_imports.go:244-245`):

```go
if alias, hasAlias := depIface.GetTypeAlias(sym); hasAlias {
    imports.ImportedTypeAliases[bindName] = alias      // unconditional overwrite
    imports.ImportedAliasOrigin[bindName] = depIface.Module  // origin overwritten too
```

each selective type import writes `ImportedTypeAliases[bindName]` unconditionally, so the
**last** import wins — and `ImportedAliasOrigin[bindName]` is overwritten with it, so even the
diagnostic origin data of the shadowed import is lost. Both "✓ passes" cells pass only because
the literal happens to match the **last** import's body; the **first** import declaration is
silently dead text. Cells 1/3 fail with a misleading distant error naming neither import.

Downstream hit (issue #1980): stapledons-godot `sim/core.ail` — a new `import ./earth_link (Leg)`
was shadowed by the existing `import ./navigation (Leg)`; the only symptom was a field-count
mismatch in an unrelated function. Live workaround: the reporter renamed their type to
`LinkLeg`. Issue stays open.

### Stale documentation of the boundary

`internal/errors/codes.go:112-117` (comment) and `:295` (registry) both describe MOD015 as
covering a name "that … two imports bind to different exports". The MOD015 implementation
(`internal/elaborate/import_collision.go:37-52`) explicitly does **not**: "Two imports binding
the same bare name to different exports are NOT checked here", per the #1467 ruling note
(m-elaborator-lexical-scope, implementation notes, point 5). The registry text over-claims
today; this design allocates the missing diagnostic and corrects the stale text (Phase 1 task).

## Neighboring designs (compared before proposing scope)

| Design | What it shipped | Bearing on this doc |
|--------|-----------------|--------------------|
| [m-elaborator-lexical-scope](../../implemented/v0_52_0/m-elaborator-lexical-scope.md) (implemented v0.52.0, #1467) | MOD015: import-vs-**local** func/let/ctor is a hard error at binding time, in `ElaborateFile`, the one place every front door shares. Ruling point 3: types keep their own rule (M-TYPE-NAME-SHADOW). **Point 5 explicitly deferred import-vs-import**, warning that an import-time error is "too blunt" **for values** because `import std/list as List (length)` + `import std/string as Str (length)` (used qualified) must keep compiling (motoko `phase_vocab.ail`, `digests.ail`), naming bare-use (Haskell-style) as the natural candidate. | This doc delivers the deferred **type** half of that ruling. The value half stays deferred with the same rationale (verified live: the qualified value idiom compiles, row 8). |
| [m-type-name-shadow-and-cache](../v0_44_0/m-type-name-shadow-and-cache.md) (M1/M3 implemented) | Local type declarations win over imported ones; captured aliases withheld with `TC_TYPE_SHADOW_001` at use; transitive pull sorted + reach-limited. **Its audit table item 3 is exactly this defect** (`pipeline_module_imports.go` selective type import, order-dependent write), deferred as "import ambiguity diagnostics". M4 (defining-module-qualified alias bodies) unscheduled. | This doc is that deferred item for the selective path. M4 remains out of scope (Non-Goals). |
| [m-alias-body-closure](../v0_52_4/m-alias-body-closure.md) (implemented v0.52.4) | Alias bodies **and annotation schemes** are closed over their defining module at interface build time; foreign `TCon` mentions in imported interfaces no longer consult the importer's alias env. | After closure, the residual ambiguity surface is exactly the importing module's **own written** type names (`-> Leg`) plus the bulk auto-import pick — i.e. this doc's S1/S4. T1–T3 of that doc are already fixed; this doc does not reopen them. |
| [m-transitive-alias-env-import](../../implemented/v0_22_0/m-transitive-alias-env-import.md) (implemented v0.22.0 + M1/M3 guards) | Transitive alias pull is a **gap-fill only** (first-wins), sorted, limited to the module's import closure; direct imports dominate. | The transitive pull cannot create a binding conflict with a direct import (guard `!exists`), so it stays out of the new check's blast radius (S6 unchanged). |

**Scope conclusion.** The deferral chain points at one deliverable: a binding-site diagnostic
for two **explicit selective type imports** of the same bare name from different defining
modules (the #1980 shape), plus a recorded policy for the sibling surfaces (values, ctors, bulk
auto-imports) that this doc rules on but does not implement.

## Policy: import/import ambiguity and shadowing

Principle: **a bare name bound by two explicit selective imports from different defining
modules is ambiguous. AILANG rejects the ambiguity at the binding site when the name has no
qualified fallback, and defers to the use site when it does.**

| # | Surface | Site today | Current behavior | Ruling |
|---|---------|-----------|------------------|--------|
| S1 | Selective **type** imports (`import M (T)`, `import M (T as U)`, list of `import M as L (T)`) | `pipeline_module_imports.go:244-245` unconditional write | last import wins silently | **Phase 1: hard error MOD016 at binding time.** Types have no qualified reference form (`A.Leg` in type position is PAR015, verified row 9), so an ambiguous type binding is unusable, not merely ambiguous. Precedent: MOD015 errors at binding time regardless of use. |
| S2 | Selective **value** imports | `GlobalRefs[bindName]` write in the same loop | last import wins silently for bare uses | **Deferred (Phase 2), use-site error.** Qualified fallback exists (`import M as L (x)` binds `L.x`; verified row 8) and the shipped qualified idiom must keep compiling — exactly the #1467 point-5 rationale. Import-time error is ruled out. |
| S3 | Selective **ctor** imports | `resolveConstructorImport` (`GlobalRefs`, `ImportedCtorTypes`, `ImportedCtorInfos`) | last wins silently | **Deferred with S2** (ctors are values in expression/pattern position; `M-CTOR-AUTO` bulk path already first-wins). |
| S4 | Bulk **type-alias** auto-import (M-TYPE-ALIAS: every alias of every direct import, selected or not) | `:278` guarded write | first import in declaration order wins silently | **Deferred (Phase 2), use-site capture.** Implicit binding, no user intent; first-wins must stay for legitimate cross-package unification (m-transitive-alias-env-import). Natural mechanism: extend the M-TYPE-NAME-SHADOW capture (`CapturedAliases`) to withhold bulk-collided names and error on expansion. Own doc + corpus sweep. |
| S5 | Bulk ctor auto-import (M-CTOR-AUTO) | guarded write | first-wins | **Deferred with S4.** |
| S6 | Transitive alias pull | `:175` guarded gap-fill, sorted, reach-limited | first-wins, direct imports dominate | **Unchanged.** Deterministic; cannot conflict with direct imports by construction. |
| — | Same module imported twice binding the same name | — | identical binding | **No error** (verified row 8: same-module duplicate compiles today and stays legal). |
| — | Type name vs value/ctor of the same bare name from two modules | — | separate namespaces | **No error** (MOD015 ruling point 3 precedent). |
| — | Local type vs imported type | M-TYPE-NAME-SHADOW M1 | local wins, capture guard | **Unchanged.** |

Phase 1 implements S1 only. S2/S3/S4/S5 are policy-recorded here, implementation deferred to
a Phase 2 design doc (the "own ruling" that m-elaborator-lexical-scope point 5 called for —
this section is that ruling's type-side half plus the recorded direction for the rest).

### Why binding-time for types when the value case defers to use-time

1. **No qualified fallback.** A bare type name is the only way to reference a type
   (`A.Leg` → PAR015, row 9). An S1 collision therefore makes one import declaration dead on
   arrival — the exact "silently dead text" MOD015 was ruled against.
2. **The alias env is not per-use.** `ImportedTypeAliases` is a flat name→body map consulted by
   unification wherever `TCon("Leg")` appears — including inside types the module imports from
   third modules — so a use-site check cannot cleanly attribute which import a given expansion
   "came from"; the binding site is the only unambiguous site.
3. **Precedent.** MOD015 (binding-time), MOD007 (binding-time duplicate let/func), Rust's
   split of E0255 (import vs local) from E0252 (import vs import): AILANG's existing convention
   is to reject ambiguous module-scope bindings at the binding, and to use a **separate code**
   for the import-vs-import kind.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | The meaning of a bare type name no longer depends on import declaration order; the last-write-wins pick (and its lost-origin twin at `:245`) is replaced by a deterministic error naming every binding site in source order. |
| A2: Replayability | 0 | No trace-format change. Verdict changes ride the compiler-identity part of the cache key (M-COMPILE-CACHE-DIRTY-BUILD-KEY), so no stale replay across binaries. |
| A3: Effect Legibility | 0 | Effects are not involved. |
| A4: Explicit Authority | 0 | No capability change. |
| A5: Bounded Verification | +1 | A module's type meaning no longer depends on a sibling import's position; reasoning about one import requires only that import. |
| A6: Safe Concurrency | 0 | — |
| A7: Machines First | +1 | The reported failure mode is a machine-written pair of plausible imports producing a field-count error in an unrelated function (or a silent wrong check); the fix yields a coded error naming both imports with a paste-able alias fix. |
| A8: Minimal Syntax | 0 | No new syntax; the existing `as` alias form becomes the required fix. |
| A9: Cost Visibility | 0 | The check is O(selective imports) in the elaborator pass that already loads each imported module. |
| A10: Composability | +1 | Two packages exporting the same type name compose by forced explicit aliasing instead of a silent order-dependent pick. |
| A11: Structured Failure | +1 | Adds coded MOD016 (structured report, `check --json`, LSP range) where today there is either a misleading TC error or silence. |
| A12: System Boundary | +1 | Import declarations become honest module boundaries: an accepted import is one that is actually bound. |

**Net Score: +6** → **Decision: Move forward.**

### Hard Violation Check

- [x] A1 (Determinism): the check iterates binds in source order and sorts groups; no map-order input.
- [x] A3 (Effects): no hidden side effects.
- [x] A4 (Authority): no ambient access granted.
- [x] A7 (Machines First): the error exists to make an invisible miscompile machine-decidable.

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

## Goals

**Primary Goal:** Two selective imports binding the same bare type name from different defining
modules become a hard, coded, both-sites-named compile error (MOD016) at the binding site —
mirroring the MOD015 ruling for import-vs-local — with a paste-able fix line.

**Success Metrics:**
- [ ] All four cells of the #1980 truth table produce MOD016 naming both imports (the two
      "passes" today become errors naming the silently dead first import).
- [ ] `ailang check --json` carries code/phase/span/data{name,imports[]}; the LSP shows a
      positioned diagnostic at the first (dead) import.
- [ ] Fix line pastes and compiles (fixture: `import ./a (Leg)` + `import ./b (Leg as bLeg)`).
- [ ] Zero verdict changes in the in-repo corpus sweep (std/, examples/, testdata — the
      2026-10-02 729-file sweep idiom) beyond new fixtures.
- [ ] MOD015's stale registry/comment text no longer claims import-vs-import coverage.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| S1 is a **binding-time hard error** (not warning, not use-time) | Decides the failure mode for every two-imports-same-type program; warning would leave the miscompile reachable | human (this doc records it; ratify before sprint) | design | med |
| **Separate code MOD016** rather than extending MOD015 | Codes are machine-keyed behavior; MOD015's span/fix shape (local definition) does not fit import-vs-import | agent (per MOD007/MOD015 precedent) | design | low |
| Values/ctors (S2/S3) stay **legal at binding time**, deferred to a use-site ruling | Import-time error would break the shipped qualified idiom (motoko corpus) | human (recorded here from #1467 point 5; Phase 2 doc needs its own ratification) | design | low |
| Bulk auto-import (S4/S5) stays first-wins; use-time capture is Phase 2 | Cross-package unification depends on the bulk pull | agent (follows m-transitive-alias-env-import) | design | low |
| MOD015 registry/comment text corrected to stop over-claiming | Docs-truth fix rides the same change; leaving it stale re-creates this triage | agent | design | low |
| Same-module duplicate imports stay legal; type-vs-value same name stays legal | Defines the non-cases to keep compiling | agent (MOD015 ruling point 3 precedent) | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] S1 = binding-time hard error MOD016 (this doc; human ratification at sprint-planner)
- [x] S2/S3/S4/S5 deferral and Phase 2 direction (this doc)
- [ ] Human approval of this design doc (per AGENTS.md feature gate) — **pending; this request is design-only**
- [ ] **Quorum: PENDING re-run.** Attempted twice 2026-10-28 (artifacts
      `.ailang/state/mission-quorum/m-import-alias-collision-2026-10-28T08-47-44Z.json` and
      `...-49Z.json`); both runs recorded every independent reviewer seat ABSENT
      (`oc-glm-5-3`, `oc-kimi-k3`: unreachable; `gpt6-1-sol`: auth; `gemini-4-argon`:
      unreachable; same-vendor recall `claude-sonnet-5@claude-p`: quota) — synthesis correctly
      refused to pass on zero signal ($0 spent). Re-run `ailang design-quorum` once reviewer
      access is restored, before sprint-planner consumes this doc. The blocking condition is
      outside this session (API access), so the doc is handed over with the gap labelled rather
      than re-ground.

## Solution Design

### Overview

Add an import-vs-import **type-name** collision check beside the existing MOD015 import-vs-local
check, in the elaborator — the same `ElaborateFile` choke point every front door shares — and
allocate MOD016 for it. The check classifies each selective import symbol by the namespaces it
binds (value / ctor / type) using the already-loaded `LoadedModule`, then rejects two type-kind
bindings of the same bare name from different defining modules. No change to
`resolveModuleImports` write semantics in Phase 1: the error fires before line 244 could matter.

### Architecture

**Components:**

1. **`importKind` gains `importType`** (`internal/elaborate/import_collision.go`). The binds
   loop in `ElaborateFile` (`internal/elaborate/file.go:216-243`) currently classifies each
   selective symbol as value (via `moduleLoader.GetExport`) or ctor (`importIsConstructor`) and
   `continue`s past types. Extend it: `GetExport` nil and not a ctor and
   `moduleLoader.Load(imp.Path).Types[sym]` exists → append a bind with `kind: importType`.
   `LoadedModule.Types` exists (`internal/loader/loader.go:37`, row 5) — no new loader
   machinery. Classification order mirrors the pipeline's found-flag chain (value → type →
   ctor), so a dual-exported name binds its value kind as today; the type half of a dual export
   is out of MOD016's scope (separate namespaces, same as MOD015 point 3).
2. **`checkImportTypeCollisions(file, binds)`** in `import_collision.go`, called from the same
   place `checkImportCollisions` runs (after binds are collected, before any binding reaches
   `symbols`/`imports`). Groups `importType` binds by `bindName`; a group with binds from two or
   more **different `path`** values → MOD016 error. Iteration in source order of import
   declarations (binds are appended that way); message names every colliding site
   (`import ./b (Leg) at main.ail:2:1`, `import ./a (Leg) at main.ail:3:1`) plus each defining
   module. Same-`path` groups and cross-kind (type vs value) groups are not collisions.
3. **MOD016 error path** mirroring `mod015`: `errors.Report` with `Code: MOD016`, `Phase:
   "elaborate"`, span at the **first** (silently dead) import declaration, `Data:
   {"name", "imports": [...]}`, and a `Fix` suggestion reusing the existing
   `aliasSuggestion`/`collisionFix` machinery (`import ./a (Leg as aLeg)` — prefix from the
   path's last segment, the MOD015 test already asserts this form pastes). Human text:
   `Error MOD016: 'Leg' is imported as a type from both ./b (at main.ail:2:1) and ./a (at
   main.ail:3:1) — two imports may not bind the same bare type name. Fix: alias one import:
   import ./a (Leg as aLeg)`.
4. **Code allocation** (`internal/errors/codes.go`): const MOD016 + registry entry
   `{MOD016, "module", "namespace", "Two imports bind the same bare type name"}` + the
   codes_test list; correct the stale MOD015 comment (`:112-115`) and registry text (`:295`)
   to describe only import-vs-local. MOD016 is verified unallocated (row 3).
5. **Docs**: `docs/docs/reference/errors/mod016.md` (same shape as `mod015.md`), index entry,
   and a section beside the MOD015 section in `docs/docs/reference/modules.md` stating the rule
   and the value/ctor deferral.

**Deliberately not changed:** `resolveSelectiveImports` write semantics (S4/S5 guards,
S6 transitive pull), `TC_TYPE_SHADOW_001`, MOD015 behavior and span policy, the cache key
(verdict changes ride compiler identity).

### Implementation Plan

**Phase 1: MOD016 selective type-import collision (~2 days)**
- [ ] `importType` classification in the binds loop (`file.go`), `importType` constant.
- [ ] `checkImportTypeCollisions` + `mod016` helper; wire beside `checkImportCollisions`.
- [ ] MOD016 const/registry/tests in `internal/errors`; MOD015 stale-text correction.
- [ ] Elaborator unit tests + fixture package (truth table, fix-pastes, non-cases).
- [ ] LSP positioned-diagnostic test; `check --json` shape test.
- [ ] Docs pages (`errors/mod016.md`, index, `modules.md`).
- [ ] Corpus sweep + motoko sibling static scan; changelog entry.

**Phase 2 (follow-up sprint, policy above, own doc): use-site ambiguity (~1 week)**
- [ ] S2/S3: bare-use ambiguity error for values/ctors (multi-binding table consulted at
      identifier resolution; qualified uses unaffected); new code; corpus sweep incl. motoko.
- [ ] S4/S5: bulk-collision capture-at-use via the `CapturedAliases` mechanism; sweep.

**Phase 3 (not scheduled): M4** — defining-module-qualified alias bodies / nominal identity,
per m-type-name-shadow-and-cache.

### Files to Modify/Create

**New files:**
- `docs/docs/reference/errors/mod016.md` - error reference page, ~40 LOC (mirror of mod015.md)

**Modified files:**
- `internal/elaborate/import_collision.go` - `importType` kind, `checkImportTypeCollisions`, `mod016` helper, ~90 LOC
- `internal/elaborate/file.go` - classify type symbols into binds, ~15 LOC
- `internal/errors/codes.go` - MOD016 const + registry entry; MOD015 stale-text fix, ~15 LOC
- `internal/errors/codes_test.go` - extend the codes list test, ~3 LOC
- `internal/elaborate/import_collision_test.go` - MOD016 unit tests (truth table cells, non-cases, fix-pastes), ~120 LOC
- `internal/lsp/import_collision_test.go` - positioned MOD016 diagnostic test, ~25 LOC
- `docs/docs/reference/errors/index.md`, `docs/docs/reference/modules.md` - rule + deferral note, ~25 LOC
- `changelogs/` (active) - entry, ~10 LOC

## Conflict Surface

1. **Positions extended:** entries of a selective import's symbol list, classified as binding a
   bare **type** name (`import M (T)`, `import M (T as U)` → binds `U`; the selective list of
   `import M as L (T)` still binds bare `T` — module aliases give no type-qualified form, row 9).
2. **Other constructs already in those positions:** value imports (S2 — must keep compiling,
  deferred), ctor imports (S3 — must keep compiling, deferred), a value and a type sharing one
  name across the two colliding modules (separate namespaces — no error), a module exporting
  both a func and a type under one name (binds its value kind today; type half out of scope).
3. **Disambiguation:** `bindName` = alias if present else symbol (existing rule); kind by
   `GetExport` → `Types` → `Constructors` in the pipeline's found-flag order; collision = same
   `bindName`, same kind `importType`, different `imp.Path`.
4. **Programs that MUST still work (all verified compiling today, rows 6–8):**
   - `import ./a as A (len2)` + `import ./b as B (len2)` (qualified value idiom; motoko parity).
   - `import ./a (Leg)` + `import ./b (Leg as bLeg)` (the fix form itself).
   - `import ./a (Leg)` twice from the same module (identical binding).
   - Type from a + value from c under one bare name (separate namespaces).
   - Every MOD015 fixture (`internal/elaborate/import_collision_test.go`), every
     M-TYPE-NAME-SHADOW capture fixture (`internal/pipeline/type_name_shadow.go` tests).
   - Full corpus: `ailang check` per file over `std/`, `examples/`, `cmd/ailang/testdata/`
     (729-file precedent, 2026-10-02) — expect zero verdict changes beyond new fixtures.
5. **Intentional incompatibilities:** truth-table cells 2 and 4 (today's "passes") become
   MOD016 errors — those programs had a silently dead import declaration; that is the defect.
   The reporter's exact repro in the "other order" stops compiling with both imports named.
   The shipped downstream workaround (`LinkLeg`) is unaffected.

## Examples

### Example 1: the issue's repro, after

**Before** (cell 2 — today passes with a dead first import):
```ailang
module repro/dup/main
import ./a (Leg)
import ./b (Leg)
export pure func mk() -> Leg = { id: 1, from: 0.0, port: 1.0, t0: 0.0, legT: 2.0, flying: true }
```

**After:**
```
Error MOD016: 'Leg' is imported as a type from both ./a (at main.ail:2:1) and ./b (at main.ail:3:1)
— two imports may not bind the same bare type name.
  Fix: alias one import: import ./b (Leg as bLeg)
```
and `import ./b (Leg as bLeg)` + `mk() -> Leg` (a's) type-checks as written.

### Example 2: non-cases that stay legal

```ailang
import ./a (Leg)              -- type
import ./c (Leg)              -- value (func) — separate namespace: legal
import ./a as A (len2)        -- + import ./b as B (len2): qualified idiom, legal (bare
                              --   len2 uses are S2's Phase 2 question, unchanged today)
```

## Success Criteria

- [ ] All four truth-table cells yield MOD016 naming both import sites and both modules (test).
- [ ] Non-cases (qualified idiom, aliased fix form, same-module dup, type-vs-value) still compile (tests).
- [ ] `check --json` report: code MOD016, phase elaborate, span at first import, data name+imports (test).
- [ ] LSP diagnostic positioned at the first import decl (test, mirroring `lsp/import_collision_test.go`).
- [ ] Fix line pastes and compiles (test, mirroring `TestMOD015AliasSuggestion`).
- [ ] MOD015 text no longer over-claims (grep `codes.go`).
- [ ] Corpus sweep: no verdict changes outside new fixtures; motoko sibling scan clean.
- [ ] All tests passing; documentation updated; examples added.

## Testing Strategy

**Unit tests (`internal/elaborate/import_collision_test.go`):**
- Each truth-table cell as a package fixture (pipeline-level compile through `ElaborateFile`).
- Both-sites-named message content; deterministic across repeated runs (binds in source order).
- Non-cases: same-module duplicate, type-vs-value, cross-kind, module-aliased type import.
- Fix line pastes: apply `import ./b (Leg as bLeg)` and assert compile success.

**Integration tests:**
- `cmd/ailang` package-level check on the fixture package (the `--package .` flow of the issue).
- `check --json` and LSP shape (span at the first import, not the second).

**Corpus sweep (gate):**
- `ailang check` per file over std/, examples/, cmd/ailang/testdata before/after — only new
  fixtures may change; plus the static selective-import × module-binding scan of the motoko
  sibling (phase_vocab.ail, digests.ail must stay compiling — the #1467 method).

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact MOD016 wording (template above is binding in content: both sites, both modules, fix) — agent may polish.
- Whether the elaborator loads `imp.Path` once per import decl instead of per symbol (perf-only) — agent may choose.
- Phase 2 code allocation and mechanism details (use-site value/ctor ambiguity) — **needs its own design doc + human ratification**, not agent latitude.

## Non-Goals

**Not attempted in this feature:**
- Value/ctor import-vs-import enforcement (S2/S3) — deliberately deferred to Phase 2 (use-site); import-time error is ruled out by the qualified idiom.
- Bulk auto-import ambiguity (S4/S5) — first-wins stays; capture-at-use is Phase 2.
- M4 defining-module-qualified alias bodies / nominal ADT identity — changes the type representation; unscheduled (m-type-name-shadow-and-cache).
- REPL top-level alias merge (audit row 10) — deferred there, unchanged here.
- Wildcard imports — do not exist (IMP012), per the MOD015 ruling; nothing to design.
- Issue #1980 stays OPEN until Phase 1 ships and the reporter's workaround can be un-renamed at their discretion.

## Timeline

**Week 1** (~3 days): Phase 1 implementation + unit/LSP/JSON tests.
**Week 1 end** (~1 day): corpus sweep + docs + changelog.
**Later sprint**: Phase 2 use-site doc + implementation (not counted here).

**Total: ~3 days for Phase 1.**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Corpus breakage: in-repo or sibling code legitimately double-imports a type name | Med | The 2026-10-02 sweep found exactly one import-vs-local value collision repo-wide; a type case would surface in the sweep gate before merge. Intentional incompatibility is documented (truth-table passes). |
| Stale cache replays old verdicts after the change | Low | Compiler identity is part of the cache key (M-COMPILE-CACHE-DIRTY-BUILD-KEY); new binary → fresh keys. |
| MOD015 message/format coupling in LSP or downstream consumers | Low | MOD016 is a new code; MOD015 bytes unchanged. |
| Dual-exported name (func + type same name) misses the type bind in the elaborator | Low | Same as today (classification by GetExport first); documented in Conflict Surface; Phase 2 revisit. |

## Verification Log

Hard-gate premise checks (skill: design-doc-creator), all run 2026-10-28 on repo HEAD `51b00832`, binary `a6b4c2c` v0.54.19:

1. **Repro verified live.** Four-cell truth table (above) run with `AILANG_NO_CACHE=1 ailang check --package .` on a properly-formed package (manifest exports must be package-prefixed module paths). Cell 1 reproduced the issue error verbatim: `expected 6 fields, got 4` at the return type annotation; cell 2 (order swapped) passes. Transcripts in this session.
2. **Root-cause mechanism read.** `internal/pipeline/pipeline_module_imports.go:244-245`: unconditional `ImportedTypeAliases[bindName]`/`ImportedAliasOrigin[bindName]` writes in `resolveSelectiveImports` (selective path); bulk loop `:278` and transitive pull `:175` are `!exists`-guarded (first-wins) — confirmed by reading the whole file.
3. **MOD016 unallocated.** `grep -rn "MOD016" internal/ cmd/ std/ docs/ examples/ dist/ llms.txt` → no hits; `dist/error_codes.json` lists no MOD016. Next free MOD code.
4. **MOD015 over-claim confirmed.** `internal/errors/codes.go:112-117` (comment) and `:295` (registry) claim two-import coverage; `internal/elaborate/import_collision.go:37-52` explicitly excludes it ("NOT checked here").
5. **Elaborator can classify types.** `internal/loader/loader.go:37`: `LoadedModule.Types map[string]*ast.TypeDecl` exists; `importIsConstructor` already loads modules this way; binds loop at `internal/elaborate/file.go:216-243` `continue`s past types today (verified read).
6. **Front-door coverage.** `ElaborateFile` call sites: `pipeline/pipeline_module_phases.go:362`, `pipeline/pipeline_single.go:181`, `repl/module_registry_load.go:131`, `runtime/runtime.go:174` — the same four doors MOD015 claims.
7. **Deferred-item lineage.** m-type-name-shadow audit item 3 (this defect, deferred) and m-elaborator-lexical-scope ruling point 5 (values deferred, "needs its own ruling") — both read in full this session.
8. **Non-cases verified compiling today** (live probes): same-module duplicate import; `import ./a (Leg)` + `import ./b (Leg as bLeg)`; `import ./a as A (len2)` + `import ./b as B (len2)` qualified value idiom; type-vs-value same bare name.
9. **No qualified type reference form.** `export pure func mk() -> A.Leg = ...` → PAR015 ("bare assignment not supported") — types are bare-name only; load-bearing for the binding-time ruling.
10. **Negative-existence sweep of this doc:** "no wildcard imports" and "no type-qualified form" — MOD015 ruling text + row 9 probe; "no other write path registers selective type aliases" — full file read (row 2); "codes_test list needs extension" — `internal/errors/codes_test.go:114` read.

## Related Documents

**Implemented (may inform design):**
- [m-elaborator-lexical-scope](../../implemented/v0_52_0/m-elaborator-lexical-scope.md) — MOD015 (#1467), whose ruling point 5 deferred exactly this; its corpus-sweep method is this doc's gate.
- [m-alias-body-closure](../v0_52_4/m-alias-body-closure.md) — closed alias bodies/schemes; bounds this doc's residual surface.
- [m-transitive-alias-env-import](../../implemented/v0_22_0/m-transitive-alias-env-import.md) — the transitive pull this doc leaves unchanged (S6).

**Planned (check for overlap):**
- [m-type-name-shadow-and-cache](../v0_44_0/m-type-name-shadow-and-cache.md) — its audit item 3 is this defect; M1/M3 shipped; M4 is this doc's Non-Goal.
- Auto-search (SimHash + neural) found no overlapping implemented/planned doc for import-vs-import collisions; the manual lineage above is complete (verified by reading each).

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- Issue #1980 (this design's driver; stays open until Phase 1 ships)
- [m-elaborator-lexical-scope](../../implemented/v0_52_0/m-elaborator-lexical-scope.md) implementation notes, ruling point 5
- [m-type-name-shadow-and-cache](../v0_44_0/m-type-name-shadow-and-cache.md), audit table item 3
- `internal/elaborate/import_collision.go` (MOD015), `internal/pipeline/pipeline_module_imports.go:244-245` (defect site)

## Future Work

- Phase 2: use-site ambiguity for values/ctors (S2/S3) and bulk-collision capture (S4/S5) — own design doc.
- M4: defining-module-qualified alias bodies and nominal ADT identity.
- REPL top-level alias merge (m-type-name-shadow audit row 10).

---

**Document created**: 2026-10-28
**Last updated**: 2026-10-28