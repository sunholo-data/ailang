## M-XMOD-ALIAS-POLY: parameterized type aliases substitute their arguments (`type Box[a] = { items: [a] }`)

**Status**: IMPLEMENTED (v0.30.0, mission iteration 26 — M-XMOD-ALIAS-POLY sprint)
**Target**: v0.30.0 (a real feature, not a quick fix — needs substitution machinery)
**Priority**: P3 (Medium-low — a genuine expressiveness gap, but with a clean workaround: inline the parameterized shape, or use a single-constructor ADT `type Box[a] = Box([a])`. Unblocks nothing outright.)
**Estimated**: ~1–2 days (alias env must store `(params, body)`; substitute on applied-alias expansion; wire through the interface export; tests).
**Dependencies**: None, but builds directly on [M-XMOD-ALIAS](implemented/v0_28_1/m-cross-module-type-alias.md) and [M-XMOD-ALIAS-CHAIN](implemented/v0_28_1/m-xmod-alias-chain.md).

**Name note**: the `M-XMOD-ALIAS-POLY` ID is inherited from where this was first referenced as a follow-up
(M-XMOD-ALIAS et al.). It is **not** a cross-module-specific bug — see below, it fails single-module too.
Kept for reference continuity.

**Discovered by**: M-XMOD-ALIAS / M-PARSER-ALIAS-TARGETS scope-out.
**Verified against**: v0.28.1-dev (with M-PARSER-ALIAS-TARGETS applied, so the tuple form now parses).

---

## Problem

Parameterized aliases **parse and type-check when merely declared**, but **fail the moment they are used
with a concrete type argument** — and they fail *within a single module*, so this is a general
substitution gap, not a cross-module one.

```ailang
module t
type Box[a] = { items: [a] }
func mk(xs: [int]) -> Box[int] = { items: xs }
-- Error: cannot unify old record type with *types.TApp
```

```ailang
type Ident[a] = a
func use(x: int) -> Ident[int] = x
-- Error: cannot unify Ident[int] with int
```

The applied form `Box[int]` reaches unification as an un-expanded `TApp{TCon("Box"), [int]}` — the alias is
never instantiated to `{ items: [int] }`, and the type parameter `a` is never bound to `int`.

## Root cause

Alias expansion is **nullary-only** at every layer:

1. **`RegisterTypeAlias`** ([elaborate/core.go](../../../internal/elaborate/core.go)) stores the alias body as a
   monotype with the type parameters left as *free* `TVar`s — there is no record of the parameter list, so
   there is nothing to bind arguments to.
2. **`expandAlias`** ([types/unification_core.go](../../../internal/types/unification_core.go)) only matches a
   bare `*TCon` (`aliasEnv[con.Name]`). An **applied** alias is a `*TApp` whose *constructor* is the alias
   `TCon` — that shape is never looked up or instantiated.
3. **The interface builder** ([iface/builder.go](../../../internal/iface/builder.go)) registers the alias
   target via `AddTypeAlias(name, astTypeToInternalType(target))` with no parameter binding, so even a
   correct expander couldn't recover `a`'s position cross-module.

## Proposed approach

Teach the alias machinery to carry and apply parameters:

1. **Store `(params []string, body Type)`** for each alias instead of a bare `body`. (New small struct in
   the alias env / iface `TypeAliases`.)
2. **Expand applied aliases** in `expandAlias`: when `t` is a `*TApp` whose constructor is an alias `TCon`
   with arity *n* and `len(args) == n`, instantiate — substitute `params[i] → args[i]` throughout `body`
   (fresh-var-safe substitution), then continue the existing fixpoint loop on the result. Reuse the
   existing substitution helper in `internal/types` rather than hand-rolling.
3. **Carry params through the iface** so the substitution is available to importers (compose with
   M-XMOD-ALIAS's cross-module path). Confirm this stays **digest-neutral** the way M-XMOD-ALIAS is
   (aliases are not part of `computeDigest`).
4. Arity mismatch (`Box[int, string]` for a 1-param alias) → a clear diagnostic, not a silent fall-through.

## Conflict Surface

*(Touches `internal/types/`, `internal/elaborate/`, `internal/iface/` — required.)*

- **Position extended:** alias expansion — from `TCon` only to also `TApp{aliasTCon, args}`.
- **Other constructs at that position / must still work:**
  1. **Nullary aliases** (`type Row = Json`, `type Ref = Id` chains) — still a `TCon`; the new `TApp` branch
     doesn't touch them.
  2. **Genuine parameterized ADTs** (`Option[a]`, `Result[a,b]`, user `type Tree[a] = …|…`) — these are
     `TApp` over a *constructor* `TCon` that is **NOT** in `aliasEnv` (ADTs register constructors, not
     aliases). Lookup misses → returned unchanged → stays nominal. **This is the critical non-regression:
     the fix must key strictly on alias-env membership so it never instantiates a real ADT.**
  3. **Records/tuples/functions as alias bodies** — substitution must recurse into `TRecord`/`TTuple`/
     `TFunc2` fields.
- **Programs that MUST still work (fixtures):** all existing alias tests (M-XMOD-ALIAS pack), `Option`/
  `Result` usage across modules, a user parameterized ADT with `match`, and nullary-alias chains.
- **Intentional change:** applied parameterized aliases that previously errored now instantiate. No
  previously-passing program should change meaning (ADTs are untouched by the alias-env keying).

## Test plan

- Single-module: `Box[int]` (record), `Ident[int]` (bare param), `Pair[int,string]` (tuple, needs
  M-PARSER-ALIAS-TARGETS), a 2-param alias, and nested `Box[Box[int]]`.
- Cross-module: the same imported and used (composes with M-XMOD-ALIAS).
- Non-regression: `Option[int]` / `Result[int,string]` and a user `type Tree[a]` ADT still nominal.
- Arity-mismatch diagnostic test.
- `make test` + `make verify-examples` green.

## Out of scope

- Higher-kinded / partially-applied aliases (`type F = Box` used as `F[int]`) — punt unless a real case appears.

## Implementation notes (as landed, v0.30.0)

Delivered as sprint M-XMOD-ALIAS-POLY across three milestones (see the sprint plan
alongside this doc). Summary of what landed vs the proposal:

- **Representation:** rather than replacing `TypeAliases map[string]Type` with a
  `(params, body)` struct, a **sibling** `AliasParams map[string][]string` was
  added at every layer (elaborator, `iface.Iface`, `CoreTypeChecker`, `Unifier`).
  A missing entry means "nullary alias". This kept the existing nullary/record-
  `TypeName` paths byte-identical (the M-XMOD-ALIAS pack is untouched).
- **Substitution helper:** the right primitive was the `Type.Substitute(map[string]Type)`
  method (every variant implements it; `TVar2.Substitute` keys by name), NOT
  `ApplySubstitution` (which walks a unifier `Substitution` of fresh inference
  vars). No new traversal code.
- **expandAlias:** a `*TApp` branch was added at the top of the fixpoint loop —
  decompose, require the head `TCon` be in `aliasEnv` (strict membership = the
  ADT-nominality guarantee), arity-check, `body.Substitute(param→arg)`, continue
  the loop. The `seen` cycle guard was extended to the `TApp` head name.
- **Arity diagnostic:** `TC_ALIAS_ARITY_001` (errors.go), coded + directional,
  styled on `TC_ARITY_001`. `expandAlias` returns `Type` (4 call sites) so the
  error is latched on the unifier and surfaced by `Unify` right after expansion.
- **cacheKey:** bumped `v2 → v3` defensively (on-disk `Iface` gained
  `AliasParams`; the blob is JSON so tolerant, but a same-version dev build could
  otherwise treat a parameterized alias as nullary). Digest stays neutral
  (`computeDigest` excludes both `TypeAliases` and `AliasParams`) — no cascade.
- **PR #380 ordering:** expansion runs at `Unify` entry, before the type-switch
  dispatches to `unifyRecord`, so open-record patterns over a parameterized-alias
  body already see the expanded record. Locked by a test.
- **Extra shape covered for free:** function-body aliases (`type Fn[a,b] = (a)->b`)
  work via `TFunc2.Substitute` — not in the original test plan, added as a test.

Tests: `internal/types/alias_poly_test.go` (unit: expansion, ADT nominality,
arity), `internal/pipeline/alias_poly_test.go` (E2E single- + cross-module, PR
#380 lock), extended `internal/iface/xmod_alias_digest_test.go` and
`internal/pipeline/cache_store_test.go`. Example: `examples/type_alias_poly.ail`.
