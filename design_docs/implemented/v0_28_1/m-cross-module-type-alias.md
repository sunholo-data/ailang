## M-XMOD-ALIAS: Transparent type aliases to non-record types across module boundaries

**Status**: IMPLEMENTED (on `dev`, 2026-07-02; first release: v0.28.1 tbd)
**Target**: v0.28.1 (v0.28.0 already shipped; this is a patch-level bug fix)
**Priority**: P2 (Correctness gap — silently breaks any published package that aliases a non-record type across modules; surfaced under toolchain skew and hard to diagnose)
**Estimated**: ~0.5 day (one `*ast.TypeAlias` case in the interface builder + tests; mirrors code that already exists in the elaborator)
**Dependencies**: None. The elaborator and type checker already support this; only the interface-export path is missing.
**Related / prior art** (this is the third pass at cross-module aliases — the first two were **record-only**):
- [M-TYPE-ALIAS — Cross-Package **Record** Type Alias Unification](../implemented/v0_10_0/m-type-cross-package-alias-unification.md) (v0.9.11) — added `TCon → TRecord` expansion in the unifier + record-alias propagation. This is the `M-FIX-RECORD-UPDATE` code referenced throughout `iface/builder.go`.
- [M-TRANSITIVE-ALIAS-ENV-IMPORT](../implemented/v0_22_0/m-transitive-alias-env-import.md) (v0.22.0) — propagated transitively-imported **record** aliases into the unifier's `aliasEnv`.
- [M-TYPE-ALIAS-UNIFICATION](../implemented/v0_5_8/m-type-alias-unification.md) (v0.5.8), [M-CROSS-MODULE-RECORD-UNIFICATION](../implemented/v0_5_10/m-cross-module-record-unification.md) (v0.5.10) — earlier steps in the same lineage.

**Reported from**: `sunholo/duckdb@0.1.0` (registry package) broke the `eparse` CLI under AILANG v0.27.0. The package was locked against v0.14.2 — classic toolchain skew. Fixed at the package level in `sunholo/duckdb@0.1.1` (dropped the alias); this doc routes the underlying core fix.
**Verified against**: v0.27.0 — `internal/iface/builder.go` has no `*ast.TypeAlias` case; `internal/elaborate/file_funcs.go:220` does.

---

## Verdict: VALID core-floor fix — restore a consistency the language already promises within a module

`type X = Y` in AILANG is a **transparent alias**, not a nominal newtype. (Nominal wrapping is spelled
`type Gen = Gen(int)` — a single-constructor ADT.) Transparent aliases already work correctly **within
a single module**: the elaborator registers `Row = Json` and the type checker expands it during
unification. The bug is that this transparency is **dropped at the module boundary** for any alias whose
target is not a record type. Two modules in the same package therefore disagree about whether `Row`
equals `Json`. That is an internal inconsistency in the toolchain, not a design choice — the fix makes
the cross-module behaviour match the already-correct within-module behaviour.

This is not a new feature or an extension; it is closing a hole in an existing mechanism
(`M-FIX-RECORD-UPDATE`, which added cross-module alias transparency but only for record types).

---

## Prior art — this is the third pass, and the first two were record-only *by construction*

Cross-module type aliases have been fixed twice, both times for **record** aliases only:

1. **M-TYPE-ALIAS (v0.9.11)** — triggered by DocParse billing's `type Usage = { ... }`. It taught the
   unifier to expand `TCon("Usage") → TRecord{...}` and wired record-alias propagation. The interface
   builder's alias registration (`AddTypeAlias`) was added here — **gated on `*ast.RecordType`**.
2. **M-TRANSITIVE-ALIAS-ENV-IMPORT (v0.22.0)** — triggered by `motoko_agent`, a record alias declared in
   module A, referenced by B's signature, called from C. It widened the `aliasEnv` to include
   transitively-imported aliases. Still `TCon → TRecord`, still record-only.

Neither pass ever handled an alias whose target is **not a record** — `type Row = Json`,
`type UserId = int`, `type Handler = (Req) -> Resp`. That case is `TCon → TCon` (or `TCon → TApp` / tuple
/ function), a different expansion the record machinery never needed. The gate that made the earlier
fixes record-only (`if _, ok := def.(*ast.RecordType)`) is exactly the gate this doc removes by adding
the sibling `*ast.TypeAlias` case. So "we already fixed cross-module aliases" is true — but only for the
record half of the space. This is the non-record half, and it fails the same way, silently, under
toolchain skew (v0.22.0's own risk note called out "silent breakage of every package that splits
**record-type** aliases across multiple modules" — this is the non-record analog of that same hazard).

---

## Symptom

```
type error in pkg/sunholo/duckdb/query (decl 1): type unification failed
at [return type annotation at .../query.ail:58:1]:
  failed to unify record field 'rows': cannot unify type constructors: Json vs Row
```

Minimal repro (two modules in one package):

```ailang
-- types.ail
module demo/types
import std/json (Json)
export type Row = Json                       -- transparent alias to a NON-record type
export type QR  = { columns: [string], rows: [Row], rowCount: int }
```

```ailang
-- query.ail
module demo/query
import std/json (Json, asArray)
import ./types (QR)
export func parse(j: Json) -> QR =
  match asArray(j) {                          -- asArray gives Option[[Json]]
    Some(items) => { columns: [], rows: items, rowCount: 0 },  -- [Json] into [Row] field → FAIL
    None        => { columns: [], rows: [], rowCount: 0 }
  }
```

The same code with `Row` and `QR` **inlined into `query.ail`** type-checks fine. Only the cross-module
import fails. `type Row = { ... }` (a record alias) also works cross-module today — it is specifically
aliases to **non-record** targets (`Json`, `int`, `Result[a,b]`, tuples, functions, lists) that break.

## Root cause

Type aliases reach the type checker through two channels, and non-record aliases are present in one but
missing from the other:

| Channel | Populated by | Handles `type Row = Json`? |
|---------|--------------|-----------------------------|
| **Local** — `elaborator.GetTypeAliases()` → `typeChecker.RegisterTypeAlias` ([pipeline_module_compile.go:116](../../internal/pipeline/pipeline_module_compile.go#L116)) | `internal/elaborate/file_funcs.go:220` — a `*ast.TypeAlias` case that registers **any** target | ✅ Yes → within-module works |
| **Cross-module** — `depIface.TypeAliases` → `imports.ImportedTypeAliases` → `RegisterTypeAlias` ([pipeline_module_imports.go:111](../../internal/pipeline/pipeline_module_imports.go#L111), [pipeline_module_compile.go:123](../../internal/pipeline/pipeline_module_compile.go#L123)) | `internal/iface/builder.go:382-434` — only `*ast.RecordType` and single-ctor-record `*ast.AlgebraicType` call `AddTypeAlias` | ❌ No → cross-module breaks |

In [internal/iface/builder.go:376-435](../../internal/iface/builder.go#L376-L435), an exported
`type Row = Json` calls `iface.AddType("Row", 0)` (registering `Row` as a **nominal** `TCon`) but there
is **no `*ast.TypeAlias` case**, so `AddTypeAlias` is never called. The imported interface carries no
expansion for `Row`, so the importing module's type checker treats `Row` and `Json` as distinct
constructors and unification fails.

The elaborator already does the right thing — [file_funcs.go:220-227](../../internal/elaborate/file_funcs.go#L220-L227):

```go
case *ast.TypeAlias:
    targetType := e.astTypeToInternalType(def.Target)
    if targetType != nil {
        e.RegisterTypeAlias(typeName, targetType)   // registers ANY target, record or not
    }
    return nil, nil
```

The interface builder simply never mirrored this case.

## The fix

Add the missing `*ast.TypeAlias` case to the interface builder, mirroring the elaborator. In
[internal/iface/builder.go](../../internal/iface/builder.go), inside the exported-type loop (alongside
the existing `*ast.RecordType` and `*ast.AlgebraicType` handling ~L384):

```go
// Register transparent aliases to NON-record types (e.g. `type Row = Json`,
// `type UserId = int`, `type Handler = (Request) -> Response`) so they expand
// across module boundaries — mirrors the elaborator's *ast.TypeAlias handling
// in elaborate/file_funcs.go. Record aliases are already covered above via
// *ast.RecordType.
if aliasDef, ok := typeDecl.Definition.(*ast.TypeAlias); ok {
    if target := astTypeToInternalType(aliasDef.Target); target != nil {
        iface.AddTypeAlias(typeDecl.Name, target)
    }
}
```

That is the whole change. `AddType(name, arity)` is still called (the nominal name remains a valid
export for docs/rendering), but now an expansion accompanies it, so importers can unify through it —
exactly as the defining module already does.

## Semantics & risk

- **No new semantics.** `type X = Y` is already transparent within a module. This makes cross-module
  behaviour identical, removing an inconsistency. There is no nominal distinction to lose — code that
  wanted a nominal type would have used a single-constructor ADT.
- **Digest-neutral, low blast radius.** `computeDigest` ([builder.go:593](../../internal/iface/builder.go#L593))
  hashes only `Exports` + `Constructors`, **not** `TypeAliases`. Adding an alias expansion does not
  change the interface digest, so this will **not** trigger spurious dependent-package cascade rebuilds.
  (Function signatures that *use* the alias are unaffected — they were already stored expanded or
  nominal in `Exports`; verify no signature-string drift in the test plan below.)
- **Parameterized aliases are out of scope for v1.** `type Pair[a] = (a, a)` has free type variables in
  its target; `AddTypeAlias` stores a monotype, and the existing record-alias path does not substitute
  type params either. Non-parameterized aliases (the overwhelming majority, and 100% of the reported
  cases) are covered. Parameterized-alias expansion is tracked as a follow-up (§ Out of scope).
- **Alias-to-unexported-target.** `type Row = Json` where `Json` is itself imported is fine — the
  expansion is a `TCon{Name:"Json"}` and the importer already imports `Json`. An alias to a target that
  is neither exported nor stdlib-visible from the importer is a pre-existing concern, unchanged by this.

## Conflict Surface

*(Required — this change touches `internal/iface/` and interacts with `internal/elaborate/` and `internal/types/`.)*

**1. What position does this change extend?**
The set of exported type names that carry a *transparent structural expansion* in an imported
module's interface (`iface.TypeAliases`), consumed by the unifier's `expandAlias` via
`ImportedTypeAliases`. Today: record aliases + single-ctor-record ADTs. After: also `*ast.TypeAlias`
declarations (aliases whose target is a named/applied/tuple/function/list type).

**2. What OTHER valid constructs live in that position (exported type declarations)?**
| Declaration form | `TypeDecl.Definition` node | Registered as | After this change |
|---|---|---|---|
| `type Usage = { … }` (record) | `*ast.RecordType` | alias (transparent) | unchanged — handled by existing branch |
| `type Item = Item({ … })` (single-ctor record ADT) | `*ast.AlgebraicType` | alias (transparent, for field access) | unchanged — existing branch |
| `type Color = Red \| Green` (sum ADT) | `*ast.AlgebraicType` | **nominal** (`AddType`+ctors) | **unchanged — stays nominal** |
| `type Gen = Gen(int)` (nominal newtype) | `*ast.AlgebraicType` | **nominal** | **unchanged — stays nominal** |
| `type Row = Json` / `type UserId = int` | `*ast.TypeAlias` | **nominal only (the bug)** | **now also transparent (the fix)** |

**3. How is it disambiguated?**
Purely by the concrete `TypeDecl.Definition` AST node type, and the three node types are produced by
*disjoint* parser branches — verified in [parser_type_decl.go](../../internal/parser/parser_type_decl.go):
`*ast.TypeAlias` at [:169](../../internal/parser/parser_type_decl.go#L169) (no constructors, no leading/peek
`|`, not a `{…}` literal); `*ast.AlgebraicType` at [:208](../../internal/parser/parser_type_decl.go#L208)/[:215](../../internal/parser/parser_type_decl.go#L215)
(any `|` or `Ctor(...)` form — this is where **both** sum types and nominal newtypes like `Gen(int)` go);
`*ast.RecordType` at [:333](../../internal/parser/parser_type_decl.go#L333) (`{…}` literal). Because the new
case matches **only** `*ast.TypeAlias`, it is impossible for it to fire on a sum type, a nominal newtype,
or a record — so it **cannot silently make a nominal type structural**. This is the single most important
safety property of the change.

**4. Which existing programs MUST still work post-change? (regression-surface fixtures)**
1. **Nominal sum ADT cross-module** — `type Color = Red | Green` in A; `match` in B. Must stay nominal; must not unify with anything structural.
2. **Nominal newtype cross-module** — `type Gen = Gen(int)` in A; must **not** unify with bare `int` in B (would be a real regression if it did).
3. **Record alias cross-module** — the M-TYPE-ALIAS `type Usage = {…}` case; must still unify structurally.
4. **Single-ctor-record field access cross-module** — M-STREAM-DX `type Item = Item({name})` then `item.name` in another module.
5. **Chained alias** — `type A = Json` and `type B = A` in module X, `B` used in module Y. **Known pre-existing limitation** (confirmed during implementation): `expandAlias` does not chain through an intermediate alias TCon, so this fails identically *within a single module* too. Not introduced or fixed here; locked by a characterization test and tracked as the `M-XMOD-ALIAS-CHAIN` follow-up. The fixture guards that cross-module behavior stays *at parity with* single-module.

**5. What deliberately changes (intentional incompatibility)?**
Exactly one direction: a cross-module use of a non-record alias that previously **failed** to unify with
its target now **succeeds**. No program that previously type-checked should now fail — the change only
*adds* entries to the imported alias env (same "additive, only widens, local-wins precedence preserved"
risk profile that M-TRANSITIVE-ALIAS-ENV-IMPORT established for the record case). Fixture (5) plus the
digest-stability test (below) guard that additivity.

## Test plan

1. **Regression (the bug):** two-module fixture package where `types.ail` declares
   `type Row = Json` + `type QR = { rows: [Row], ... }` and `query.ail` builds a `QR` from `[Json]`.
   Must type-check. (Add under `internal/iface/` or `tests/` as a cross-module fixture.)
2. **Non-record target coverage:** aliases to `int`, `Result[int, string]`, a tuple `(int, string)`,
   a function `(int) -> bool`, and a list `[string]` — each imported and used cross-module.
3. **Record aliases unchanged:** existing `M-FIX-RECORD-UPDATE` / `M-STREAM-DX` tests still pass
   (`internal/iface/compact_adt_fields_test.go`, cross-module record-update tests).
4. **Digest stability:** assert the interface digest of a module that adds a non-record alias is
   unchanged vs. before the fix (guards the no-cascade claim).
5. **End-to-end:** `sunholo/duckdb` with the original `type Row = Json` restored type-checks and `eparse`
   runs — i.e. the package-level workaround in 0.1.1 becomes unnecessary (though we keep it).
6. `make test` + `make verify-examples` clean.

## Alternatives considered

- **Package-level workaround (done for duckdb):** drop the alias, use the underlying type in exported
  signatures. Correct and shipped as `0.1.1`, but it is per-package whack-a-mole — every package that
  aliases a non-record type across modules hits this, and the failure only appears under toolchain skew,
  long after publish. Does not scale.
- **Make `type X = Y` nominal cross-module (treat as newtype):** would make the language *consistently
  wrong* relative to its within-module behaviour, and break the idiomatic `type Row = Json` /
  `type UserId = int` documentation pattern. Rejected — AILANG already chose transparent `type` +
  nominal single-ctor ADT.
- **Expand aliases eagerly in signatures at export (never store the alias name):** loses the nominal
  name for hover/docs/error messages and is a larger change to `schemeToString`. The alias-map approach
  is smaller and preserves naming.

## Out of scope (follow-ups)

- **Chained non-record aliases** (`type B = A`, `A = Json`) — `expandAlias` doesn't chain through an
  intermediate alias TCon; this fails within a single module too, so it is a pre-existing unifier gap,
  not part of this fix. Tracked as `M-XMOD-ALIAS-CHAIN`. Single-level aliases (the reported case) are covered.
- **Tuple- and function-type aliases** (`type Pair = (int, string)`, `type Pred = (int) -> bool`) — these do
  not **parse** in alias position even within a single module (`PAR_TYPE_BODY_EXPECTED` / `PAR_NO_PREFIX_PARSE`).
  A separate parser-level gap; file as `M-PARSER-ALIAS-TARGETS` if needed. TCon and TList targets (the common
  cases, incl. `type Row = Json`) are covered.
- Parameterized non-record aliases (`type Pair[a] = (a, a)`) — needs type-param substitution at
  expansion; file as `M-XMOD-ALIAS-POLY` if a real case appears.
- REPL cross-module alias parity (`internal/repl/module_registry_load.go` already threads
  `TypeAliases`; confirm it benefits automatically once the builder populates them).

---

## Implementation Report (2026-07-02, on `dev`)

**What was built:** a single `*ast.TypeAlias` case in the interface builder, exactly as designed.

**Code (implementation, ~12 LOC incl. comment):**
- [internal/iface/builder.go](../../internal/iface/builder.go) — added, after the `*ast.RecordType` alias
  block, an `if aliasType, ok := typeDecl.Definition.(*ast.TypeAlias); ok` branch calling
  `iface.AddTypeAlias(typeDecl.Name, astTypeToInternalType(aliasType.Target))`. Mirrors
  `elaborate/file_funcs.go`.

**Tests (~200 LOC):**
- `internal/pipeline/cross_module_nonrecord_alias_test.go` — 6 fixtures via the `Run(ModeCheck)` harness:
  `NonRecordAliasInField` (the core positive, TDD fail-before/pass-after — pre-fix `int vs Id`),
  `NonRecordTargetKinds` (TCon + TList), `NominalNewtypeStaysNominal` + `NominalSumADTStaysNominal`
  (negative non-regression guards), `RecordAliasUnchanged`, and `ChainedAlias_KnownLimitation`
  (characterization). Deterministic across `-count=20`.
- `internal/iface/xmod_alias_digest_test.go` — `computeDigest` is identical with/without a non-record
  alias (locks the no-cascade claim).

**Verification:**
- `make test` green (full suite). `verify-examples`: 182 pass, 5 fail — all 5 confirmed **pre-existing**
  by building a baseline binary with the fix reverted and reproducing identical failures (effect-row /
  Option issues in those examples, unrelated to aliases). `gofmt`/`go vet` clean.
- **Real-world:** reconstructed the exact `type Row = Json` + `QueryResult.rows: [Row]` pattern (built from
  `asArray`'s `[Json]`) that broke `eparse` → `✓ No errors found` under the fixed binary. The
  `sunholo/duckdb@0.1.1` package-level workaround is now subsumed by the core fix (not republished).

**Deviations from plan:**
- Test home is `internal/pipeline/` (real cross-module type-check via `Run`), not `internal/iface/` as the
  plan tentatively said — the pipeline harness exercises the actual import path.
- **Two design fixtures reclassified out-of-scope** after discovering they fail *within a single module*
  too, so they are pre-existing limitations, not part of this fix: **chained aliases** (`type B = A`,
  `A = Json` — `expandAlias` doesn't chain; kept as a characterization test → `M-XMOD-ALIAS-CHAIN`) and
  **tuple/function-type aliases** (don't parse in alias position → `M-PARSER-ALIAS-TARGETS`).
- **No new example file** — this is a cross-module type-checking fix, not a surface feature; the
  regression fixtures are the executable evidence (deliberate deviation from the examples-per-feature policy).
