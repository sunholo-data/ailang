## M-XMOD-ALIAS-CHAIN: transitively expand chained type aliases (`type B = A`, `A = Json`)

**Status**: PLANNED
**Target**: tbd (patch-level; low priority)
**Priority**: P3 (Low — real correctness gap, but low frequency: aliasing an alias is uncommon. Has an obvious workalike: alias directly to the base type.)
**Estimated**: ~0.5 day (fixpoint loop + cycle guard in one function + tests)
**Dependencies**: None.
**Related**: follow-up carved out of [M-XMOD-ALIAS](implemented/v0_28_1/m-cross-module-type-alias.md) (which fixed single-level non-record aliases across modules). Same lineage as [M-TYPE-ALIAS](implemented/v0_10_0/m-type-cross-package-alias-unification.md).

**Discovered by**: M-XMOD-ALIAS implementation — `TestXModAlias_ChainedAlias_KnownLimitation` locks the current (failing) behavior.
**Verified against**: v0.28.0 — the chain `type Ref = Id; type Id = int` fails **within a single module** too (see repro), so this is not a cross-module bug; it is a general `expandAlias` gap.

---

## Problem

A type alias whose target is *another alias* does not expand transitively:

```ailang
module m
type Id  = int
type Ref = Id                         -- alias to an alias
type Box = { items: [Ref] }
func mk(xs: [int]) -> Box = { items: xs }   -- FAILS: cannot unify int vs Id
```

`ailang check` (single module, v0.28.0): `cannot unify type constructors: int vs Id`. The unifier
expands `Ref → Id` but stops there; it never expands `Id → int`, so `[int]` won't unify with `[Ref]`.

## Root cause

[`internal/types/unification_core.go:88`](../../internal/types/unification_core.go#L88) — `expandAlias`
does exactly **one** lookup:

```go
func (u *Unifier) expandAlias(t Type) Type {
    if con, ok := t.(*TCon); ok {
        if target, exists := u.aliasEnv[con.Name]; exists {
            // ... returns target directly (one level) ...
            return target
        }
    }
    return t
}
```

When `target` is itself a `TCon` present in `aliasEnv` (e.g. `Ref → TCon("Id")`), it is returned
un-expanded. There is no fixpoint iteration.

## Proposed fix

Iterate to a fixpoint with a cycle guard:

```go
func (u *Unifier) expandAlias(t Type) Type {
    if u.aliasEnv == nil { return t }
    seen := map[string]bool{}
    for {
        con, ok := t.(*TCon)
        if !ok { return t }
        if seen[con.Name] { return t }       // cycle guard: `type A = A`
        seen[con.Name] = true
        target, exists := u.aliasEnv[con.Name]
        if !exists { return t }
        if rec, ok := target.(*TRecord); ok && rec.TypeName == "" {
            return &TRecord{Fields: rec.Fields, Row: rec.Row, TypeName: con.Name}
        }
        t = target                            // keep expanding if target is another alias TCon
    }
}
```

## Conflict Surface

*(Touches `internal/types/` — required.)*

**Position extended:** alias expansion in the unifier — from single-level to fixpoint.

**Other constructs in that position / must still work:**
1. **Single-level non-record alias** (M-XMOD-ALIAS) — `type Row = Json`; first iteration returns `Json`, unchanged.
2. **Record alias** — `type Usage = {…}`; the `TRecord` branch still fires on the first (and only) hop, preserving `TypeName`. Chained record aliases (`type U2 = Usage`) now also resolve.
3. **Nominal TCon that is NOT an alias** — `int`, `Json`, ADT names: `exists` is false → returned as-is (no behavior change).
4. **Self/mutual cycle** — `type A = A`, or `type A = B; type B = A`: the `seen` guard returns the last TCon instead of looping forever. (Pre-fix this couldn't arise because expansion never chained; the guard makes the new loop safe.)

**Regression fixtures:** flip `TestXModAlias_ChainedAlias_KnownLimitation` to assert **success**; add a
single-module chained-record-alias test; add a cycle test (`type A = A`) asserting termination (no hang,
graceful mismatch). Existing M-XMOD-ALIAS + M-TYPE-ALIAS + record-update tests must stay green.

**Intentional change:** chained aliases that previously failed now unify. No previously-passing program regresses (the loop is a strict superset of the old single hop for non-cyclic input).

## Out of scope

- Parameterized alias chains (`type Pair[a] = (a, a)`) — blocked on parameterized-alias support generally.
