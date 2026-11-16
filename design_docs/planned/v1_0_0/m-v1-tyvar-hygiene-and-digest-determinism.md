# M-V1-TYVAR-HYGIENE — closed export schemes, fresh record rows, canonical names at the module boundary

**Status**: Planned
**Target**: v0.58.x (the last breaking window — see the [release-readiness review](../v1-release-readiness-review-2026-11-09.md) §3.1c and §3.5)
**Priority**: P1. The new soundness hole (F3) makes it a 1.0 blocker; #2076 alone would not be one.
**Estimated**: 4–6 working days *(estimate, not a plan)*
**Dependencies**: none to start. Decision D-4 (lowercase type names) should be settled together with the §3.1d / #1485 doc.
**Planner-Lane**: opus-required (typechecker generalisation and instantiation)
**Issues**: #2157 (P1), #2076. New: F3 below (no issue yet).

## Summary of findings (all reproduced first-hand, binary built from `51f6636`)

| # | Finding | Severity |
|---|---|---|
| F1 | **#2157 is under-generalisation, not just a name collision.** `std/sem.is_newer` is exported with `type_vars: [] row_vars: []` and free variables `α74` and `r`. An importer instantiates nothing, so `α74` meets the importer's own `α74`, which comes from a per-module counter that restarts at zero. | P1, valid programs rejected |
| F2 | **#2076: allocating fresh variables depends on Go map order** at three sites. The raw `αN` names then reach the module digest. Adding a private helper above an export also changes the digest, even though the interface is the same. | A1 violation, cache churn |
| F3 | **New soundness hole.** If a function's declared closed-record return flows through an unannotated lambda parameter, it is exported as an open row, `{score: float, ...a}`. `best().bogus` then passes `check` and fails at run time with `record has no field: bogus`. This happens within a module as well as across modules. | **Soundness — §3.1-class blocker** |
| F4 | **The answer to the review's open question is no.** The nondeterminism does not reach `interface_hash` (v2), `HashProjection` or `.stdlib-golden/`, because they render through a renamer that works by first appearance. It does reach the compile-cache key. | Downgrades #2076 |
| F5 | **The v2 hash is deterministic but depends on how a function is implemented.** `[a]` and `list[a]` are two internal representations (`TList` and `TApp list`). Changing only the body of `ident(xs: [a]) -> [a]` from `xs` to `xs ++ []` changes `interface_hash` v2. That breaks the Stable sentence "`interface_hash` changes only when exports or effects change" (`stability.md:339`). | Stable-promise defect |
| F6 | **Lowercase type names are parsed as type variables** (`parser_type.go:77`). Every `f: sem_frame` annotation in `std/sem` is a type variable, so the declared record type is never enforced. This is what lets F1 occur in `std/sem`. | Trigger. Routed: D-4 |

**Shared root?** Partly. The proximate causes differ: F1 is incomplete generalisation, and F2 is fresh variables allocated in map order. Both reach users through the same missing invariant: **the raw type-variable names of one inference context cross the module boundary as identity.** No code makes an exported scheme *closed* (no free variables), and none gives it *canonical* names (`builder.go:512` and `builder.go:560` are literal `TODO`s for both). Fixing that boundary closes #2076 completely. It turns F1/F3-class bugs into loud export-time errors instead of rejections that depend on unrelated code. Each bug still needs its upstream fix: M2 for F1/F3, and M3 for in-module determinism.

## Problem Statement

### Reproductions

```bash
B=<scratch>/ailang            # built from 51f6636; installed b4e97cd behaves identically
S=<scratch>/dd-tyvar          # every run gets its own AILANG_CACHE_DIR, so every run is cold
```

**#2157 (3/3 on both binaries).** This is the issue's program, saved as `repro_is_newer_unify.ail`:

```
$ AILANG_CACHE_DIR=$S/c-1 $B check repro_is_newer_unify.ail
Error: type error in repro_is_newer_unify (decl 2): ... [function application of is_newer at
repro_is_newer_unify.ail:15:42]: failed to unify parameter 0: failed to unify field 'ver':
cannot unify record with unexpandable type constructor int
$ # encodeF32LE([0.12]) instead of the 4-element list:
✓ No errors found!
```

The exported scheme that the importer receives (`compile/modules/std__sem/iface.json`):

```
is_newer  type_vars: []  row_vars: []   free: α74, r
          ({ver: α74 | r}, {ver: α74 | r}) -> bool
```

A two-module repro needs neither `std/sem` nor a collision. The free variable is simply shared between import sites, so two legal uses at different field types conflict:

```
-- lib.ail
module lib
export pure func newer(a: t, b: t) -> bool = a.ver > b.ver
-- use2.ail: newer(p, q) with ver: int, then newer(s, t) with ver: string
$ $B run --caps IO use2.ail
Error: type error in use2 (decl 0): at literal at use2.ail:5:26: No instance for Num[string]
```

**Census of unclosed export schemes** (a script walks every cache `iface.json` and lists variables not in `type_vars ∪ row_vars`):

| Corpus | Exports | Unclosed | Where |
|---|---|---|---|
| `std/*.ail` (49 modules) | 495 | 10 | all in `std/sem`: 8 have only `r` free, plus `is_newer` (`α74`) and `has_embedding` (`α68`) |
| std + `examples/` (442 modules) | 1161 | 11 | the 10 above, plus `examples/runnable/stdlib_embedding.bestMatch` (`r`, which is F3) |

**F3, the soundness hole (minimal):**

```
module s5
export func best() -> {label: string, score: float} {
  let f = \acc. if acc.score > 1.0 then acc else acc
  in f({label: "none", score: 0.0})
}
export func main() -> () ! {IO} { println(best().bogus) }

$ $B check s5.ail   →  ✓ No errors found!
$ $B run --caps IO s5.ail  →  Error: execution failed: record has no field: bogus
```

The same shape through `foldl` exports `best (list[float])->{score: float, ...a}`. The declared return type `{label, score}` has been lost, and an importer's `best(xs).bogus` also passes `check`. Accessing a missing field on a let-bound or annotated record is correctly rejected ("record field 'bogus' not found"). Only the path through a lambda parameter leaks.

**#2076.** `std/sem` checked 20 times cold gave 6 distinct digests, which is all 3! permutations of `α87/α88/α89` in `make_frame`. A 12-run fixture gives 6 distinct digests as well:

```
module nd
export pure func mk(n: int) -> rec_t = {p: None, q: None, s: None, n: n}
export pure func mkAgain(n: int) -> rec_t = mk(n)
  → mk ['α1','α2','α3'] ×8 · ['α3','α1','α2'] ×2 · ['α2','α3','α1'] ×1 …   (6 digests / 12 runs)
```

The digest also moves when nothing in the interface changes. Adding `pure func helper(n: int) -> int = n + 1` above `mk(n: int) -> rec_t = {p: None, n: n}` changes `type_vars` from `['α1']` to `['α3']` and the digest from `400247b1` to `73989316`. `ailang iface` prints `(int)->{n: int, p: Option[a]}` for both.

**Other artifacts** (run 1 vs run 2, 7 modules): `iface.json` differs in 1 of 7, `core.gob` in 7 of 7, `coretypeinfo.gob` in 4 of 7, `artifacts.json` in 7 of 7, and `constructors.json` in 0 of 7. This matches V18 of [m-typecheck-env-scan-cost](../../implemented/v0_57_0/m-typecheck-env-scan-cost.md).

### Question 3: does the nondeterminism reach anything Stable?

| Surface | Nondeterministic? | Evidence |
|---|---|---|
| `interface_hash` v2 (`HashProjection`, `ifacev2`) | **No** | 20 cold `publish --dry-run` runs of a scratch package containing `std/sem`: 1 distinct v2 hash (`b6ba0197…`) and 1 distinct v1 hash, against 6 distinct cache digests in the same runs |
| `.stdlib-golden/*.json` (`ailang iface`) | **No** | 6 cold `ailang iface std/sem` runs were byte-identical, and their sha256 equals `.stdlib-golden/sem.sha256` (`e72a51e5…`) |
| v1 `interface_hash` | No | It hashes the manifest only (`internal/pkg/hasher.go:75`) |
| Module digest (`iface.Digest`) | **Yes** | It hashes `schemeToString` with raw names (`internal/iface/digest.go:61`) |
| Compile-cache key | **Yes** | `ModuleCacheKey` folds `cacheDepDigest = Digest + aliasDigest` (`internal/pipeline/cache_dep_closure.go:48`), so every importer of `std/sem` misses the cache whenever `std/sem` is recompiled cold. CI and per-task `AILANG_CACHE_DIR`s, such as motoko's, are always cold |
| Lockfile and published hashes | No | They store v1 and v2 only |

**Why the Stable surfaces are safe.** `ToNormalizedJSON`'s `canonicalizeType` (`internal/iface/json.go:191-210`) sorts record labels and renames variables to `a, b, c` in order of first appearance. α-permutations disappear. The cost is that the renderer cannot see **quantification or class constraints**: a free `b` and a quantified `b` print the same, and `Ord`/`Num` constraints are not printed. That is why the M2 fix is hash-neutral (good), and also why v2 cannot detect an export gaining a constraint (D-3).

**Severity verdict.** #2076 is an A1 defect in compile artifacts and causes cache churn. It is not a nondeterministic Stable hash. Fix it in v0.58.x because the fix is cheap and non-breaking, but it does not block 1.0. F3 and F5 are the 1.0-relevant items.

## Root Cause Analysis

| Bug | Site | What happens |
|---|---|---|
| F1, F3 | `internal/types/typechecker_defaulting.go:304-343` `collectFreeVarsWithVisited` | It has no `case *TRecordOpen` and skips `TRecord.Row`. Variables that occur only under an open-record shim are never collected, so `generalizeWithConstraints` (`typechecker_functions.go:485`) never quantifies them. The project rule is to use `traverse.Walk`, and `traverse` (`traverse.go:139`) does handle `TRecordOpen` |
| F1, F3 | `internal/types/typechecker_data.go:108` and `:156` | Every field access and record update uses **the literal row variable `"r"`**, even though `ctx.freshRecordRow()` (`inference.go:552`, which yields `ρN`) exists. All shims in a module share one row identity. `generalizeWithConstraints` quantifies only *effect* rows (`:520-537`), so `r` is never generalised either. *(F3's exact unification path is not traced. The hypothesis is that the shared `r` absorbs the closed return annotation. M2's mutation check confirms or refutes this.)* |
| F1 | `internal/types/types_v2.go:541-548` `InstantiateWithConstraints` | It freshens only quantified variables, so a free `α74` passes through verbatim. It also freshens every `RowVar` as `EffectRow` ("for now, assume effect rows"), so quantifying record rows needs a kind-aware fix here as well |
| F1 | `internal/types/inference.go:529-531` | `α%d` comes from a per-`InferenceContext` counter, and every module restarts it. Imported free names therefore share the importer's namespace. This is the collision |
| F1 | `internal/iface/builder.go:503-547` | `generalizeType` for a monotype: "`TODO: Implement proper free variable collection`", with `freeVars := []string{}`. `canonicalizeScheme`: "`TODO: Alpha-normalize the type`". Nothing at the boundary checks that a scheme is closed |
| F2 | `typechecker_data.go:17` | `for name, value := range rec.Fields` infers a record literal's fields in map order, so the field that gets `α1` changes from run to run |
| F2 | `typechecker_data.go:150`, `:172` | Record update: fresh field variables and value inference also run in map order |
| F2 | `typechecker_functions.go:503` | `generalizedTypeVars` is built by ranging over a map and is not sorted (row variables are, at `:537`). The order is fed to `Instantiate` (`types_v2.go:541`), so callers such as `make_frame` → `make_frame_at` receive permuted fresh names |
| F2 | `internal/iface/digest.go:61` and `builder.go:567` | The digest hashes `schemeToString`, which prints the raw `∀α87 α88 α89. …`. The structured scheme in `iface.json` (`types/json_scheme.go:32`) is raw as well |
| F5 | `internal/iface/json.go` `formatTypeCanonicalDepth` | `*types.TList` renders as `[a]` and `TApp{list}` as `list[a]`. Which one an export ends up with depends on inference: annotations parse to `TApp list` (`parser_type.go:56`), while `++` yields `TList`. `.stdlib-golden/` contains 150 `list[…]` and 42 `[…]` signatures. `list.concat` is `(list[a],list[a])->[a]` |
| F6 | `internal/parser/parser_type.go:77` | Any identifier that starts with a lowercase letter and is not a builtin becomes a `TypeVar`. The comment says "single letter"; the code tests only the first character. A `type sem_frame = {…}` declaration is accepted, and every later use of `sem_frame` becomes a fresh type variable. In the repo this affects only `std/sem` (5 declarations) |

## Goals

**Primary goal:** every scheme that crosses a module boundary is closed and canonically named. Compile output is then byte-identical across processes, and does not depend on unrelated declarations.

**Success metrics:**
- Unclosed export schemes: 11 → 0, enforced as an export-time error.
- Distinct `iface.json` digests over K cold runs: `std/sem` 6/20 → 1/20. All of `std/` + `examples/`: 1 module varies → 0.
- Adding a private declaration does not change an unchanged module's digest (the `nd` A/B fixture).
- #2157, `use2` and F3/`s5` give the correct result: accept, accept, reject with a location.
- `interface_hash` v2 is unchanged for every existing package by M1–M3. Only D-2 (F5) may change it, and only deliberately.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|---|---|---|---|---|
| D-1 Unclosed export scheme: hard error, or generalise silently at the boundary | Hard error is CLAUDE.md §2 (fail loudly) and catches the next F3. Silent generalisation hides checker bugs | human | design | med |
| D-2 F5 list rendering: normalise now as `ifacev3`, or freeze v2 rendering | Changes every published package's v2 identity, and PUB005 refuses skew between publisher and validator | human | design | high |
| D-3 Should the hash projection include quantifiers and constraints? | Without them v2 cannot detect `Ord a =>` being added. With them, a one-time identity change | human | design | high |
| D-4 Lowercase type names (F6): reject `type lowercase`, or resolve declared lowercase names as types | Breaks `std/sem`'s type names (Experimental tier, `stability.md:179`) | human | design | med |
| Canonical stored names use a reserved namespace (`'t0`, `'r0`, `'e0`) that neither the lexer nor the fresh supply can produce | Guarantees that no stored name ever collides with user or fresh names | agent | compile | low |

## Solution Design

### B1 — One canonicaliser at the serialisation boundary (fixes #2076; makes F1/F3 loud)

Add `types.CanonicalizeScheme(s *Scheme) (*Scheme, error)`, in a new file of about 150 lines in `internal/types/`:

1. Traverse with `traverse.Walk` in a **defined order**: params left to right, then return, then the effect row. Record labels and effect labels are sorted.
2. Rename every type, record-row and effect-row variable in order of first appearance, into the reserved spellings `'t0…`, `'r0…` and `'e0…`. Quantifier lists are rewritten in that same order rather than sorted.
3. **Closedness:** a variable that occurs free in the type but is not in `TypeVars ∪ RowVars` returns `ErrUnclosedScheme{Export, Var}`. Under D-1 (recommended), `iface.Builder.Build` turns this into an error that names the export and the variable, as an internal checker error that asks the user to file a report.
4. Constraints are renamed by the same map and sorted by `(class, type string)`.

Then:

- `builder.go` `canonicalizeScheme` calls it, and the two TODOs are deleted. The monotype branch of `generalizeType` also goes through it.
- `digest.go` hashes the canonical scheme. `json.go` `canonicalizeType` reuses **the same traversal order** and only maps `'t0` → `a` for display. One implementation, two spellings. The seam is where these bugs live.
- `iface.json` stores canonical schemes, so importers never see `αN` or `r`.

### B2 — Complete generalisation and fresh record rows (fixes F1, F3)

- Replace `collectFreeVars` *for generalisation only* with a `traverse`-based collector that also descends into `TRecordOpen` and record rows. Keep the defaulting caller as it is, because changing defaulting is a separate risk.
- Quantify free *record* row variables into `RowVars`, recording their kind. `InstantiateWithConstraints` freshens each row variable according to its kind (`freshRecordRow` / `freshEffectRow`).
- `inferRecordAccess` and `inferRecordUpdate` use `ctx.freshRecordRow()` instead of the literal `"r"`.
- Expected result: `is_newer` exports `∀'t0 'r0. Ord 't0 => ({ver:'t0|'r0}, {ver:'t0|'r0}) -> bool`, `use2` checks, and `s5` is rejected at `best().bogus`. If `s5` still passes after the fresh-row change, F3 has a second cause, and it gets its own milestone instead of being folded in here.

### B3 — Deterministic fresh allocation (in-module hygiene)

Iterate sorted keys at `typechecker_data.go:17/150/172`, and sort `generalizedTypeVars` at `typechecker_functions.go:503` (or keep first-appearance order). B1 already makes the *digest* deterministic. B3 makes in-module names, error messages and part of `core.gob` deterministic, and it is what makes the cross-process test meaningful below the boundary.

### Alternatives considered

| Alternative | Why not |
|---|---|
| B3 alone (sort the map ranges) | The next `range` over a map regresses silently. The digest would still move when a private helper is added (measured: `α1` → `α3`), so cache churn remains |
| Freshen *every* variable on import, free or not | Hides F1/F3 instead of fixing them. A free variable in an export is a checker bug, and silently quantifying it would make `is_newer` polymorphic by accident rather than by inference |
| Canonical names `a, b, c` in stored schemes | They collide with user-written type parameters (`getVer(a: x) -> y` stores `y` today). A reserved namespace removes the whole class |
| Per-process unique fresh prefix (`α<module-hash>_N`) | Fixes the collision but not determinism, and the names stay meaningless |

### Effect on hashes, caches, goldens and published packages

| Artifact | M1–M3 | D-2 / D-3 (if chosen) |
|---|---|---|
| `interface_hash` v2 | **Unchanged** (rendering is already canonical, and quantifiers are not hashed) | Changes → `ifacev3`. The validator must accept v2 or v3 for one window, or PUB005 refuses every publisher on an older binary |
| `.stdlib-golden/` | Unchanged: newly quantified variables and constraints are not rendered | Re-freeze once: 192 list signatures, plus `std/sem` if D-4 |
| Compile cache | Invalidated automatically, because the key includes the compiler identity (`cache_key.go:64`). No `cacheKeyVersion` bump needed | Same |
| Published packages | No republish needed | Validator dual-accept. Lockfiles store `interface_hash`, so `lock --check` must accept either version during the window |

## Conflict Surface (typechecker change)

- **Touched:** generalisation (`typechecker_functions.go`), instantiation (`types_v2.go`), record access and update (`typechecker_data.go`), iface build and digest (`internal/iface`).
- **Programs that MUST still work:** all of `make verify-examples`, `make test-imports`, and `std/` freeze (`make verify-stdlib`, v2 hashes byte-identical). The `#386` effect-row cases (`mapE`, `std/sem` `! {SharedMem}`, see [m-effect-row-show-interp](../../implemented/v0_31_0/m-effect-row-show-interp.md)) and nominal records ([m-alias-body-closure](../../implemented/v0_53_1/m-alias-body-closure.md)).
- **What deliberately changes:** programs like `s5` that access a missing field through a lambda-shim record now fail `check`. This is BREAKING, but only for programs that crash at run time today. Imported functions with open-record parameters become properly polymorphic, which accepts more programs (`use2`, #2157).
- **Risk spot:** a program may currently type-check *only because* every shim shares `"r"`. M2's gate is the full example and test corpus, and every newly failing program is triaged as either "was unsound" or "regression".

## Implementation Plan (sprint-planner sizing)

| M | Scope | Acceptance | Mutation check | Est. |
|---|---|---|---|---|
| M1 | Tests first, all red: T1 to T5 below; the census as a Go test; #2157, `use2` and `s5` fixtures under `tests/` | T1 fails on `std/sem` and `nd`; T5a lists exactly the 11 exports; #2157 is red | n/a: these are the tests | 0.5–1 d |
| M2 | B2: generalisation collector, kind-aware record rows, fresh `ρN` | T5a 11 → 0; #2157 and `use2` accept; `s5` rejected with `file:line`; full `make test` + `verify-examples` + `verify-stdlib` green; v2 hashes of `examples/*/ailang.toml` packages byte-identical before and after | Restore the literal `"r"` in a scratchpad copy (never `git checkout <file>`) → `s5`/T4 red. Remove the `TRecordOpen` case → #2157 red | 2–3 d |
| M3 | B1 canonicaliser plus the closed-scheme error; B3 sorted iteration | T1/T2: 1 digest per module over K runs; T3: the helper A/B gives the same digest; `ailang iface` output for all of std is byte-identical to the goldens | Make `CanonicalizeScheme` the identity → T1/T3 red. Revert one sorted range → T2 red at least once in 16 runs | 1–1.5 d |
| M4 | Only if Mark chooses D-2/D-3: `ifacev3` projection, validator dual-accept, golden re-freeze | v2 recomputation unchanged for old publishers; v3 is identical for `idA`/`idB` (F5) | Drop the list normaliser → the F5 test is red | 1–1.5 d |

## Testing Strategy

- **T1 — in-process determinism (Go, about 1 s).** `internal/iface/digest_determinism_test.go` builds the interfaces of `std/sem` and the `nd` fixture 32 times in one process, and asserts an identical `Digest` and identical serialised `iface.json` bytes. Go randomises each `range`, so a single process is enough to find map-order bugs. The observed rate (6 distinct digests in 20 runs) makes a false pass in 32 runs negligible.
- **T2 — cross-process determinism (CI script).** `tools/check-iface-determinism.sh` runs `ailang check` K times for each module in `std/*.ail`, each with a fresh `AILANG_CACHE_DIR` under `mktemp -d`, and asserts that `compile/modules/*/iface.json` is byte-identical. Measured cost: one cold pass over std takes about 1.5 s, so K=4 is about 6 s and fits in `make ci`. Over `examples/` a pass is about 16 s, so K=4 is about 65 s; run that nightly. `core.gob` and `artifacts.json` are reported but not asserted (non-goal).
- **T3 — padding invariance.** The `nd` A/B fixture: identical exports, one extra private declaration, identical digest.
- **T4 — #2157 regression and a hygiene property.** The issue repro and `use2` must check. **Property:** for every polymorphic export *e* in `std/` + `examples/`, generate importers with *k* ∈ {0, 1, 7, 50, 200} unrelated private declarations before a call of *e*. The check verdict must be independent of *k* ("rejected depending on unrelated code" is exactly this property). Also, instantiating any imported scheme twice gives disjoint variable sets that share no name with the stored scheme.
- **T5 — closed schemes.** (a) A census test: every export in the built std and example interfaces has no free variables. (b) A unit test: `CanonicalizeScheme` of a hand-built scheme with a free `α9` returns `ErrUnclosedScheme`. (c) `s5` and the `foldl` `bestMatch` shape: `check` rejects `.bogus`.

## Draft changelog fragment (`changelogs/unreleased/YYYY-MM-DD-tyvar-hygiene.md`)

```markdown
### Fixed — imported polymorphic functions no longer depend on unrelated code (#2157) — BREAKING

Exported functions with open-record parameters (e.g. `std/sem.is_newer`) were exported with free type and
row variables that collided with the importer's own fresh variables, so a valid program could be rejected
after an unrelated edit. Exports are now fully generalised, and every field access gets a fresh row
variable. BREAKING: a program that read a missing field through a lambda parameter's record — which
type-checked and then failed at run time with `record has no field` — is now rejected by `check`.

### Fixed — module interface digests are deterministic (#2076)

Type variables in stored interfaces are renamed canonically, and fresh variables are allocated in sorted
order, so `iface.json` is byte-identical across runs and does not change when a private declaration is
added. `interface_hash` and the stdlib goldens were already deterministic and are unchanged. Importers of
affected modules stop recompiling on cold caches.
```

## Axiom Compliance

| Axiom | Score | Justification |
|---|---|---|
| A1 Determinism | +1 | Removes map order from compile output |
| A2 Replayability | +1 | Byte-identical interfaces across processes |
| A3 Effect Legibility | 0 | Effect rows are only renamed |
| A4 Explicit Authority | 0 | — |
| A5 Bounded Verification | +1 | Closed schemes: an import is checked against its own interface only |
| A6 Safe Concurrency | 0 | — |
| A7 Machines First | +1 | An error no longer depends on code the agent did not touch |
| A8 Minimal Syntax | 0 | No syntax. D-4 might remove some |
| A9 Cost Visibility | +1 | No spurious recompiles |
| A10 Composability | +1 | Cross-module polymorphism works (`use2`) |
| A11 Structured Failure | +1 | Unclosed scheme becomes a named error; F3 is caught at `check` |
| A12 System Boundary | 0 | — |

Net +8. No −1 on A1, A3, A4 or A7.

## Non-Goals

- Determinism of `core.gob`, `coretypeinfo.gob` and `artifacts.json` (V18). T2 reports them; a follow-up owns them.
- Fixing F6 here. It is routed to the §3.1d / #1485 doc (undeclared type names), and this doc's milestones do not depend on it.
- Changing defaulting, or the `Num`-literal withholding (`literalWithhold`).
- Any change to v1 `interface_hash`.

## Decisions for Mark

1. **D-1: hard error on an unclosed export scheme?** **Recommend yes.** It costs nothing once M2 lands (the census reaches 0) and catches the next F3.
2. **D-2: F5 list rendering.** **Recommend normalising in v0.58.x as `ifacev3`, with validator dual-accept.** The alternative freezes a hash that changes when a function body is refactored, which contradicts `stability.md:339` for all of 1.x. This touches the sibling §4.2 "normalise golden rendering" work, so one doc should own the renderer.
3. **D-3: hash quantifiers and constraints?** **Recommend yes, folded into the same `ifacev3` bump.** It is free if D-2 happens anyway; otherwise defer.
4. **D-4: lowercase type names.** **Recommend rejecting `type lowercase = …` with a fix-it, and renaming `std/sem`'s five types** (Experimental tier). Today the declared record type of every `std/sem` function is unchecked. `make_frame_at` exports `ver: c`, which means any type. This belongs with #1485.
5. **F3 needs an issue filed.** It is a check-passes / run-time-crash hole, the same class as review §3.1a and §3.1b.

## Contradictions with the review

- §3.5 calls #2076 a potential "nondeterministic hash in a determinism-first language". **Measured: it is not.** v2 hashes and goldens are stable (20/20 and 6/6). It is a cache and artifact defect.
- §4.2 lists the `[a]`/`list[a]` golden rendering as a cosmetic normalisation. **It is a Stable-hash defect** (F5): the rendering, and therefore `interface_hash` v2, depends on how the body is written.
- §3.1c credits #2157 to a fresh-variable collision. That is the trigger. The defect is that export schemes are not closed, and it has a soundness sibling (F3) that the review did not find.

## Not verified

- F3's exact unification path. The mechanism is hypothesised and is confirmed or refuted by M2's mutation check.
- That the importer-side `α74` in #2157 is the colliding variable. This is inferred from the free `α74`, the dependence on the counter, and the "add a declaration and it passes" flip, not from a provenance trace.
- Windows and Linux runs, and registry-validator behaviour under D-2.

## Related Documents

- [v1 release-readiness review](../v1-release-readiness-review-2026-11-09.md) §3.1c, §3.5, §4.2 · [Stability Promise](../../../docs/docs/reference/stability.md) lines 339 and 350
- [m-registry-interface-hash-blind-to-signatures](../v0_35_0/m-registry-interface-hash-blind-to-signatures.md) (origin of `HashProjection`)
- [m-typecheck-env-scan-cost](../../implemented/v0_57_0/m-typecheck-env-scan-cost.md) (V18: artifact nondeterminism; #2076 found here)
- [m-effect-row-show-interp](../../implemented/v0_31_0/m-effect-row-show-interp.md) (#386: the same under-generalisation, fixed for *effect* rows only)
- [m-alias-body-closure](../../implemented/v0_53_1/m-alias-body-closure.md) · [docs/LIMITATIONS.md](../../../docs/LIMITATIONS.md)
