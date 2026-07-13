# Sprint Plan: M-XMOD-ALIAS-POLY — Parameterized Type-Alias Substitution

**Status**: ✅ COMPLETED (mission iteration 26). All milestones landed; M4
(cross-module) folded into M3 as planned. Acceptance criteria all met — see the
per-milestone "Acceptance criteria" lists below, each verified by the tests
named in the design doc's Implementation notes. Full suite green,
`verify-examples` green (top-level gate; the 5 pre-existing `runnable/` failures
are unrelated effect-row/Option bugs, identical on the base commit).

**Design doc**: [m-xmod-alias-poly.md](m-xmod-alias-poly.md)
**Sprint ID**: M-XMOD-ALIAS-POLY
**Target version**: v0.30.0
**Duration**: 1.5–2 days
**Risk level**: Medium (touches the hot unification path + iface serialization; strong non-regression guardrails required)
**Verified against**: worktree binary built at `fedbee699` (origin/dev), behaviourally probed (NOT via `--version`, which lies for worktree builds).

---

## Goal

Make parameterized type aliases substitute their concrete arguments at use sites, so
`type Box[a] = { items: [a] }` + `func mk(xs: [int]) -> Box[int] = { items: xs }` type-checks
(and the same imported cross-module), while genuine parameterized ADTs (`Option[a]`,
`Result[a,b]`, user `type Tree[a] = …`) stay strictly nominal.

---

## Verified Root-Cause Summary

I built the worktree binary and reproduced every failure the doc predicts, then read the
current code to confirm each of the three root-cause claims. **All three claims CONFIRMED**
(line refs updated to current tree). Two useful CORRECTIONS/additions below.

### Failure surface (behaviourally reproduced with worktree `bin/ailang check`)

| Probe (single-module) | Result | Error text |
|---|---|---|
| `type Box[a] = { items: [a] }` used as `Box[int]` | FAILS | `cannot unify old record type with *types.TApp` |
| `type Ident[a] = a` used as `Ident[int]` | FAILS | `cannot unify Ident[int] with int` |
| `type Pair[a,b] = (a,b)` used as `Pair[int,string]` | FAILS | `cannot unify tuple type (int, string) with Pair[int, string]` |
| `type Box[a] = {items:[a]}` nested `Box[Box[int]]` | FAILS | `cannot unify old record type with *types.TApp` |
| `type Fn[a,b] = (a) -> b` used as `Fn[int,int]` | FAILS (extra shape) | `cannot unify type application Fn[int, int] with int -> α1 ! {...}` |
| **Non-regression** `type Tree[a] = Leaf(a) \| Node(...)` + `match` | **PASSES** ✓ | baseline nominal ADT works |
| **Non-regression** `import std/option` → `Some(5): Option[int]` | **PASSES** ✓ | baseline imported ADT works |

So the failure surface is: **records, bare-param, tuple, function, and nested** alias bodies all fail
identically (un-expanded applied alias reaches unification as `TApp{TCon(alias), args}`). ADTs are
untouched. The function-body form (`Fn[a,b] = (a)->b`) is an extra shape not enumerated in the doc's
test plan — the substitution approach covers it for free (TFunc2 has a `Substitute` method), so we
add it as a test.

### Root-cause claim verification

1. **`RegisterTypeAlias` stores body without params — CONFIRMED.**
   `internal/elaborate/core.go:173` — `RegisterTypeAlias(name string, target types.Type)` stores into
   `e.typeAliases map[string]types.Type`. No param list. The three call sites at
   `internal/elaborate/file_funcs.go:201,216,225` all have `decl.TypeParams []string` in scope (e.g.
   `["a"]`) but discard it. Alias param type-vars are converted to `*types.TVar2{Name:"a"}`
   (`file_funcs.go:333`), so they survive in the body as named vars — but with nothing to bind them to.

2. **`expandAlias` matches bare `*TCon` only — CONFIRMED.**
   `internal/types/unification_core.go:88-127`. The fixpoint loop does `con, ok := t.(*TCon)` (line 99)
   and returns `t` unchanged for any non-`TCon` (line 100-102). An applied alias is a `*TApp` whose
   *constructor* is the alias `TCon`, so it never enters the loop. Called from `Unify` at
   `unification_core.go:174-175`, **before** the `SafeEquals` check and the type-switch — so expansion
   correctly happens before record/tuple/func unification (relevant to the PR #380 note below).

3. **iface builder has no param binding — CONFIRMED.**
   `internal/iface/iface.go:106` — `AddTypeAlias(name string, target types.Type)` stores into
   `TypeAliases map[string]types.Type`. Builder call sites `internal/iface/builder.go:386,400,443` pass
   only `(name, internalType)`; `typeDecl.TypeParams` is in scope but dropped.

### CORRECTIONS / additions to the doc

- **CORRECTION (substitution helper):** the doc says "reuse the existing substitution helper in
  `internal/types`". The right primitive is the **`Substitute(subs map[string]Type) Type` method on the
  `Type` interface** (every variant implements it: `types.go`, `types_v2.go`, `label_helpers.go`).
  `*TVar2.Substitute` (types_v2.go:33) keys by `.Name`, so `body.Substitute({"a": int})` binds param `a`.
  This is NOT `ApplySubstitution` (that walks a unifier `Substitution` keyed by fresh inference vars).
  Plan uses the `.Substitute(map)` method — no new traversal code needed.

- **ADDITION (PR #380 / open-record ordering):** M-LAMBDA-OPEN-RECORD-PATTERN changed `unifyRecord`
  (`internal/types/unification_records.go`). Because `expandAlias` runs at `Unify` entry (line 174-175),
  **before** the type-switch dispatches to `unifyRecord`, the open-record logic already sees the expanded
  `{items:[int]}` — no reordering needed. We add a regression test proving an *open-record pattern over a
  parameterized alias body* still works, to lock this ordering.

### cacheKey / digest decision

- **Digest: NEUTRAL, no change.** `computeDigest` (`iface/builder.go:607`) serializes only
  Module/Schema/Exports/Constructors — **not** `TypeAliases` (locked by
  `iface/xmod_alias_digest_test.go`). Adding params to the alias representation does **not** change any
  module's digest → **no dependent-package cascade**. Preserve this; extend the digest test to assert a
  *parameterized* alias is still digest-neutral.

- **cacheKey: DEFENSIVE bump `v2 → v3` REQUIRED-IF-STRUCT-CHANGES.**
  `internal/pipeline/cache_key.go:18` — `cacheKeyVersion = "v2"`. Two facts:
  (a) `compilerVersion` (build commit) already invalidates the module cache on every rebuild, so a
  released binary is safe either way.
  (b) `CachedModule.Iface` is **JSON-encoded** on disk (`cache_store.go:131,136`), and JSON tolerates
  new/missing fields (missing → zero value), so a schema change there is not a hard round-trip hazard
  the way the v1→v2 **gob** `RecordPattern.Rest` change was.
  **Decision:** if we change the on-disk shape of `TypeAliases` (i.e. store a struct with `Params`
  instead of a bare `types.Type`), bump `cacheKeyVersion` to `v3` as a cheap correctness guard against
  same-version dev/worktree builds colliding keys and decoding an old bare-alias blob into the new
  struct. **If** we instead keep `TypeAliases map[string]types.Type` unchanged and store params in a
  **separate** map/field (see M1 design choice), the on-disk Iface shape still gains a field → still
  bump to `v3`. Net: **bump to v3** in this sprint (one-line, in M3).

---

## Technical Approach

Introduce an alias record carrying `(Params []string, Body Type)` and teach `expandAlias` to
instantiate an applied alias by `Body.Substitute(param→arg)`, keyed **strictly on alias-env
membership** so real ADTs are never touched.

**Representation choice (decided):** add a sibling map alongside the existing `TypeAliases`:
`AliasParams map[string][]string` (name → param names), in `iface.Iface`, in the elaborator, and in the
`CoreTypeChecker`/`Unifier` alias env. This is minimally invasive (existing `TypeAliases map[string]Type`
untouched → existing nullary-alias code and the record-`TypeName` path keep working verbatim), and a
missing entry naturally means "nullary alias" (arity 0). The `Unifier` gains an `aliasParams
map[string][]string` field next to `aliasEnv`.

Expansion rule in `expandAlias`, added as a new branch that runs when `t` is `*TApp`:
1. `head, args := decomposeApp(t)`; require `head` is `*TCon` and `head.Name ∈ aliasEnv`.
2. Look up `params := aliasParams[head.Name]`. If `len(args) != len(params)` → **arity diagnostic**
   (coded error, see M2). If the name is in `aliasEnv` but has **0** params (nullary) and got args, that
   is also an arity mismatch.
3. Build `subs := {params[i]: args[i]}`, compute `expanded := aliasEnv[head.Name].Substitute(subs)`,
   then **continue the existing fixpoint loop** on `expanded` (so `type A[x] = B[x]` chains resolve, and
   the record-`TypeName` terminal path still fires).
4. Non-membership (`head.Name ∉ aliasEnv`) → return `t` unchanged (ADTs stay nominal — the critical
   non-regression).

---

## Milestones

### M1 — Carry `(params, body)` through elaborate + iface + type-checker alias env (~120 LOC + ~40 test)

**Files:** `internal/elaborate/core.go`, `internal/elaborate/file_funcs.go`,
`internal/iface/iface.go`, `internal/iface/builder.go`, `internal/types/typechecker_core.go`,
`internal/types/unification_core.go`.

**Work:**
- Add `aliasParams map[string][]string` field + `RegisterTypeAliasParams` / getter in the elaborator;
  populate at the three `file_funcs.go` call sites (201, 216, 225) from `decl.TypeParams`.
- Add `AliasParams map[string][]string` to `iface.Iface` (+ init in `NewIface`, + `AddTypeAliasParams`);
  populate at the three `builder.go` call sites (386, 400, 443) from `typeDecl.TypeParams`.
- Thread params from iface/elaborator into `CoreTypeChecker.aliasParams` and into the `Unifier`
  (`NewUnifierWithAliases` gains a params arg or a companion `NewUnifierWithAliasesAndParams`; the
  construction site is `typechecker_core.go:418`).

**Acceptance criteria:**
- Elaborator stores `["a"]` for `type Box[a] = {items:[a]}` and `[]`/absent for `type Row = Json`.
- iface round-trips `AliasParams` (imported module's parameterized alias arrives with its params).
- `make build` green; no behavioural change yet (M1 is plumbing — applied aliases still error).

---

### M2 — Expand applied aliases in `expandAlias` + arity diagnostic (~90 LOC + ~60 test)

**Files:** `internal/types/unification_core.go`, `internal/types/errors.go`.

**Work:**
- Add the `*TApp` branch to `expandAlias` per the Expansion rule above, keyed strictly on
  `aliasEnv` membership; substitute via `Body.Substitute(subs)`; continue the fixpoint loop.
- Add a coded arity diagnostic `TC_ALIAS_ARITY_001`, styled on `TC_ARITY_001`
  (`errors.go:344-375`): coded prefix + inline `Suggestion:`, directional (`Box[a]` expects 1, got 2 →
  "supply 1 type argument"; nullary-alias-applied → "`Row` takes no type arguments").

**Acceptance criteria:**
- `Box[int]`, `Ident[int]`, `Pair[int,string]`, `Fn[int,int]`, and nested `Box[Box[int]]` all
  type-check (single-module).
- `Box[int,string]` (arity 2 on a 1-param alias) fails with `TC_ALIAS_ARITY_001` and a directional hint.
- Golden test asserts the error is coded and directional (mirrors `arity_diagnostic_test.go`).

---

### M3 — Non-regression lock + cacheKey bump + examples + docs (~40 LOC + ~80 test)

**Files:** `internal/types/*_test.go`, `internal/iface/xmod_alias_digest_test.go`,
`internal/pipeline/cache_key.go`, `examples/type_alias_poly.ail`, `CHANGELOG.md`,
move design doc `v0_29_0 → implemented/v0_30_0`.

**Work:**
- Non-regression tests: `Option[int]`, `Result[int,string]`, and a user `type Tree[a]` ADT with
  `match` stay nominal (assert expansion does NOT fire — key on alias-env membership).
- Open-record-over-alias test (PR #380 ordering lock): field access / open-record pattern against a
  parameterized-alias record body still unifies.
- Extend `xmod_alias_digest_test.go` to assert a **parameterized** alias is digest-neutral.
- Bump `cacheKeyVersion` `"v2" → "v3"` with a comment citing this milestone (on-disk Iface gained
  `AliasParams`).
- Example file `examples/type_alias_poly.ail` (record + bare-param + tuple aliases used with concrete
  args, runnable `main`). Register it in `make verify-examples` expectations.
- CHANGELOG entry; move design doc to `design_docs/implemented/v0_30_0/`.

**Acceptance criteria:**
- All non-regression tests green; `Option`/`Result`/`Tree` behaviour byte-identical to baseline.
- `examples/type_alias_poly.ail` runs and `make verify-examples` green.
- `make test` green; digest test proves parameterized alias is digest-neutral.

---

### M4 (optional, fold into M3 if time) — Cross-module end-to-end (~0 new LOC, ~40 test)

**Files:** `internal/link` or `internal/pipeline` module-test fixtures, `examples/`.

**Work:** two-module fixture: module `shapes` exports `type Box[a] = {items:[a]}`; module `main`
imports and uses `Box[int]`. Proves M1's iface `AliasParams` threading composes with M-XMOD-ALIAS.

**Acceptance criteria:** cross-module `Box[int]` type-checks; arity mismatch across modules still coded.

---

## Test Plan (consolidated)

- **Single-module (M2):** `Box[int]` (record), `Ident[int]` (bare param), `Pair[int,string]` (tuple),
  `Fn[int,int]` (function body), nested `Box[Box[int]]`, 2-param alias.
- **Cross-module (M4):** the same, imported and used.
- **Non-regression (M3):** `Option[int]`, `Result[int,string]`, user `type Tree[a]` ADT + `match` —
  all still nominal; open-record-over-alias still unifies (PR #380 lock).
- **Arity-mismatch diagnostic (M2):** `Box[int,string]` and nullary-applied → `TC_ALIAS_ARITY_001`,
  directional hint; golden test.
- **Digest neutrality (M3):** parameterized alias does not change module digest.
- **Gate:** `make test` + `make verify-examples` green.

## Example file to create

`examples/type_alias_poly.ail` — a single module declaring `Box[a]` (record body), `Ident[a]`
(bare param), and `Pair[a,b]` (tuple body), each used at a concrete instantiation inside a runnable
`main`, printing a small result. Per CLAUDE.md every language feature needs an example; wire into
`make verify-examples`.

## Risks

| Risk | Likelihood | Mitigation |
|---|---|---|
| **Accidentally instantiating a real ADT** (`Option[a]`) if keying is loose | Medium | Key STRICTLY on `aliasEnv` membership; ADT constructors are registered as constructors, never aliases. Non-regression tests M3 assert `Option`/`Result`/`Tree` unchanged. |
| Substitution captures/collides a free var in the body | Low | Alias bodies are closed over `params`; `.Substitute(map)` replaces named `TVar2` leaves only. Add nested `Box[Box[int]]` test to exercise recursion. |
| Fixpoint loop + new TApp branch introduces non-termination (`type A[x] = A[x]`) | Low | Reuse the existing `seen` cycle guard; extend it to key on the `TApp` head name too. |
| cacheKey/gob round-trip of a stale bare-alias blob into new struct | Low | Bump `cacheKeyVersion` to `v3` (M3); Iface is JSON (tolerant) but bump anyway. |
| PR #380 open-record path sees an un-expanded alias | Low | Expansion already runs at `Unify` entry (line 174-175) before the type-switch; add explicit lock test (M3). |

## Out of scope

- Higher-kinded / partially-applied aliases (`type F = Box` used as `F[int]`) — punt unless a real
  case appears (per design doc).

---

**Velocity note:** ~380 LOC total (≈230 impl + ≈150 test) across 3–4 milestones over 1.5–2 days,
consistent with the doc's ~1–2 day estimate and recent iteration velocity (M-LAMBDA-OPEN-RECORD-PATTERN
#380 was a comparable single-day unification-path change).
