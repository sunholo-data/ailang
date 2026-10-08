# M-PURE-ROW-AND-IFACE-PURITY — `pure` must mean `! {}`: reject the keyword beside a declared effect row, and derive iface purity from the checked row

**Status**: Planned
**Target**: v0.53.0
**Priority**: P0 (High) — soundness: a `pure` signature can lie in `ailang check`, the lie is published as `"pure": true` in `ailang iface` (with the contradictory effects in the same JSON object), and it is the only thing admitting `! {Declassify}` functions into the SMT decidable fragment
**Estimated**: 3 days (Rule 1 ≈ 0.5 d, Rule 2 ≈ 1 d incl. the std blast-radius measurement, Rule 3 + serve-api cleanup + cache bump + corpus migration ≈ 1 d, tests/docs ≈ 0.5 d)
**Dependencies**: None. Independent of the parked [M-EFFECT-ROW-VAR-UNIFICATION](../v1_0_0/m-effect-row-var-unification.md) (#616) and of the planned [M-EFFECT-LATENT-FUNCTION-VALUES](../v0_48_0/m-effect-latent-function-values.md) (#1326/#573) — same soundness cluster, different mechanisms (see Related Documents).
**Source**: GitHub issue [#1443](https://github.com/sunholo-data/ailang/issues/1443) (open; this doc closes the design gap, `Closes #1443` belongs to the implementation PR only), part 1 of closed [#574](https://github.com/sunholo-data/ailang/issues/574) (`iface` pure-vs-effects contradiction on std/ai exports), and the triage doc [pure-keyword-vs-declared-row-and-iface-purity.md](../ailang-core-triage/pure-keyword-vs-declared-row-and-iface-purity.md) (2026-10-03, Recommend: design-doc). Scheduled by maintainer (Mark) 2026-10-08 from the P0 issue triage.
**Base measured**: worktree at `origin/dev` = `62ac2d09` (superset of the triage's `790169359` and the coordinator's `658ff76a3`), binary `AILANG v0.52.5 (Commit 7200786)`. Every claim below carries a Verification Log entry (V1–V16).

## Summary

Two halves of one broken contract — **`pure` does not mean `! {}`**:

1. **The keyword is not checked against the declared row.** `export pure func sneaky(x: string) -> unit ! {IO} { println(x) }` passes `ailang check` (V2). The parser sets `FuncDecl.IsPure` from the `pure` token (`internal/parser/parser_func.go:21-33`), the elaborator copies it into Core meta (`internal/elaborate/file.go:291,419`), but `ValidateEffects` (`internal/pipeline/validate_effects.go:103-122`) compares only the *declared row* against the *required* effects — `IsPure` never participates in any check.
2. **`iface` purity is a hard-coded `true`.** `determinePurity` (`internal/iface/builder.go:666-669`) returns `true` unconditionally (its own comment says "TODO: implement real analysis"). The value flows into `ExportInfo.Purity` (`builder.go:395-396`), the `ailang iface` JSON (`iface/json.go:118`), the compile cache (`pipeline/cache_store.go:227,306`), and serve-api's `extractModuleInfo` (`internal/apiserver/server.go:480`). Measured today: **155 of 468 std func exports (33%) report `"pure": true` alongside a non-empty effects list** (V8), including all 6 exports of `std/ai/streaming.ail` (V3) — the live remainder of #574's original report.

And one consequence the triage doc did not fully surface: **the lie is load-bearing.** The SMT decidable-fragment admission (`internal/smt/encodable.go:133-140`, `isPure(meta) → meta.IsPure`) rejects any function "with effects" — so the only way the corpus's `! {Declassify}` IFC functions get their `ensures` contracts verified today is by *also* being declared `pure`, i.e. by exploiting half 1. Dropping `pure` from the reference implementation of the prompt-injection benchmark makes `ailang verify` **skip** those functions (V5). Any fix that only rejects the contradiction breaks verification of declassification gates; the fix must also re-key SMT admission on the row (Rule 3 below).

The fix is one rule with three enforcement points: **`pure` ⇔ a closed, empty declared effect row** — rejected at the signature (Rule 1), reported honestly by `iface` from the checked row (Rule 2), and used as the SMT admission predicate *by row, not keyword* (Rule 3).

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No runtime or evaluation-order change; all three rules are compile-time |
| A2: Replayability | 0 | Traces unaffected |
| A3: Effect Legibility | +1 | Restores "the signature tells you the effects": today a `pure` function can perform IO and the iface says `"pure": true` next to `"effects": ["IO"]` (V2, V3) |
| A4: Explicit Authority | +1 | Declared rows feed capability reasoning (motoko's `declared_vs_performed` gate, package effect ceilings, serve-api MCP read-only hints); a `pure`-lying signature is ambient authority through the planning layer |
| A5: Bounded Verification | +1 | The SMT decidable fragment is admitted by a *checked row property* instead of an unchecked keyword; the fragment's soundness boundary becomes machine-decidable from the signature |
| A6: Safe Concurrency | 0 | No concurrency change |
| A7: Machines First | +1 | Agents and tool surfaces can trust `pure`/`Purity` instead of re-deriving it; serve-api's compensating AST heuristic and `@mcp_hints`' deliberate distrust both become redundant |
| A8: Minimal Syntax | 0 | No new syntax; the existing `pure` keyword starts meaning what the docs already say it means |
| A9: Cost Visibility | 0 | — |
| A10: Composability | +1 | Row-polymorphic combinators keep working unchanged at check time; their iface `Purity` becomes the honest `false`, which is the composable reading (the caller owns the callback's effects) |
| A11: Structured Failure | +1 | Rule 1 reuses the existing effect-check diagnostic shape (name, contradiction, suggested fix); no new error-code namespace |
| A12: System Boundary | +1 | `iface` JSON is a machine boundary (MCP tool descriptions, package registries, motoko planning); the published signal becomes derived from checked facts, not a stub |

**Net Score: +7** → **Decision: Move forward.**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced
- [x] A3 (Effects): this *removes* a hidden side effect (a `pure`-declared function performing declared-but-contradictory IO)
- [x] A4 (Authority): no ambient access granted; the opposite — a lying capability signal is repaired
- [x] A7 (Machines First): not optimizing for human convenience; the machine-readable signals (`pure`, `iface` purity) become trustworthy

## Problem Statement

### Current State

**Half 1 — the keyword is decorative at check time.**

```
$ ailang check sneaky.ail
→ Type checking sneaky.ail...
→ Effect checking...
✓ No errors found!
```

for

```ailang
module test/sneaky
import std/io (println)
export pure func sneaky(x: string) -> unit ! {IO} { println(x) }
```

(V2; exact transcript in the Verification Log.) The mechanics, all code-verified:

- The parser records the keyword (`parser_func.go:21-33`, `FuncDecl.IsPure`).
- The elaborator copies it to Core `DeclMeta.IsPure` (`elaborate/file.go:291`) and to the surface→Core bridge (`elaborate/file.go:419`).
- `ValidateEffects` builds `declaredEffects` purely from `funcDecl.Effects` and compares against required effects (`validate_effects.go:110-122`); `IsPure` is never consulted. A `pure` + `! {IO}` signature is accepted as long as the *row* covers the body.

The keyword is not cosmetic — it has three load-bearing consumers, which is why this is a soundness hole and not a lint:

1. **Typechecker generalisation** (`internal/types/typechecker.go:115`): `if d.IsPure || isValue(d.Body) { binding = ctx.generalize(fnType, EmptyEffectRow()) }` — the keyword selects the generalisation context (an empty ambient effect row). Note the exported scheme still carries the declared row: a caller of the cross-module `sneaky` **is** charged `IO` (V4), so the damage here is not row erasure but the keyword gating a premise (pure ⇒ empty row) that the checker never established.
2. **SMT admission** (`internal/smt/encodable.go:133-140`): `isPure(meta) → meta.IsPure` gates the decidable fragment; a `pure func … ! {IO}` is admitted as if pure. `verify.go:116-148` additionally *infers* `IsPure` from an explicit `! {}` — one-directional, so it never subtracts the lie.
3. **The corpus exploits half 1 to defeat 2.** The prompt-injection benchmark's reference implementation declares `export pure func sanitizeBody(rawBody: string<email>) -> string<sanitized> ! {Declassify}`; its `ensures { result == "[sanitized]" }` is verified today **only because** the `pure` keyword overrides the `Declassify` row at SMT admission. With `pure` dropped, `ailang verify` prints `⚠ SKIPPED sanitizeBody — Function "sanitizeBody" has effects` (V5).

**Half 2 — `iface` purity is a stub.**

`determinePurity` returns `true` for every export (`builder.go:666-669`, with its own TODO admitting it). Measured on the current binary across all 46 std modules:

- **155 of 468 func exports** report `"pure": true` with a non-empty `"effects"` list (V8) — the #574 class. #574 (closed) reported 16 on `std/ai` at filing; the mission log counted 12 on 2026-08-03; std/ai has since been trimmed to 6 exports, **all 6** of which still exhibit the contradiction today (V3).
- `std/list.mapE` (`export func mapE[a, b, e](f: a -> b ! {e}, xs: [a]) -> [b] ! {e}`, `std/list.ail:205`) reports `"pure": true` with an *invisible open row* — the row variable `e` does not appear in the JSON type string or effects list at all (V9). This is the #1091 class: an effect-polymorphic export is not pure, and today the iface cannot even represent that fact.

**Consumers of the stub value** (all verified): `ailang iface` JSON (`iface/json.go:118`), the compile cache (`pipeline/cache_store.go:227,306` — every cached entry carries `Purity: true`), serve-api's `extractModuleInfo` (`apiserver/server.go:480`, overwritten in the AST-loaded path by the compensating heuristic at `apiserver/routes.go:251`), the builder's second JSON emission (`builder.go:720`), and the linker's `ImportedSym.Purity` (`link/module_linker.go:151` — stored, and **never read** by any consumer; V10). serve-api was already patched to distrust it (commit `9305f1c19`, not present in this worktree's history but superseded by the current `routes.go:228-240` comment block, which documents the distrust), and `@mcp_hints` deliberately distrusts both signals.

**Impact:** every machine consumer of `pure` or `iface` purity — MCP tool descriptions (`[pure]` tag, read-only hints), OpenAPI `x-ailang-pure`, A2A tags, package quality gates, motoko's capability planning — is consuming either an unchecked keyword or a hard-coded `true`. The runtime capability layer still backstops execution (no `--caps IO` ⇒ refused), so this defeats *planning from signatures*, not runtime enforcement — the same bounded-impact classification as #616.

## Goals

**Primary Goal:** Make `pure` ⇔ a closed, empty declared effect row — enforced at the signature, reported truthfully by `iface`, and used as the SMT decidable-fragment admission predicate by row rather than keyword.

**Success Metrics:**

- The issue's `sneaky` repro fails `ailang check` with an actionable diagnostic naming the contradiction.
- `ailang iface` reports `"pure": false` for every export with a non-empty **or open** effect row, and `"pure": true` only for exports whose checked row is closed and empty (std re-measurement lands in M1: the 155 false-positives of V8 flip to `false`; the `map`/`mapE` family is the expected-and-correct headline case).
- `pure func … ! {}` (16 corpus sites), `pure func` with no row (762 corpus sites), and unannotated pure functions are unchanged (V11 sweep is the regression baseline).
- `ailang verify` on the migrated prompt-injection reference still reports **3 verified, 0 violations** (V5 becomes the acceptance test).
- Cached iface entries with stub purity are invalidated exactly once (cache-key bump).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **Option A: reject `pure` + non-empty/open declared row at the signature** (triage's recommended option; B "desugar" and C "advisory" rejected — see Solution Design) | Defines what `pure` means forever; the alternative (advisory) would orphan the keyword and the SMT fragment | human (this doc; ratified by triage) | design | med |
| **The signature rule covers open rows too**: `pure … ! {e}` is rejected, not just `pure … ! {IO}` | A pure function exporting an open row is #1091's defect; accepting it would leave the iface signal unsound for the var-only case the JSON cannot even display | human (this doc) | design | low |
| **SMT admission re-keyed to the row** (`row ⊆ smtTransparentEffects`), not the keyword | The only alternative — keeping keyword admission — breaks verification of every `! {Declassify}` IFC gate the moment Rule 1 lands (V5) | human (this doc) | design | med |
| **`smtTransparentEffects` starts as `{"Declassify"}`** (host-transparent per #1557, no runtime ops — V12), `Debug` deliberately excluded | Widening it later is cheap; narrowing after tools depend on it is not | agent (within the set documented here) | compile | low |
| **Derived iface purity includes "no quantified row vars"** (`scheme.RowVars` empty) | Determines whether `std/list.map` reports `false` (it must — an unannotated callback row makes it effect-polymorphic; the latent-effects analysis says callers own those effects) | human (this doc, aligned with #1091's principle) | design | low |
| **serve-api stops overwriting `Pure` from the AST heuristic** (`routes.go:251`) and reads the derived iface value | The heuristic is blind to open rows (`! {e}`-only rows read as pure today); keeping it would preserve a second, divergent purity definition | human (this doc) | design | low |
| **One cache-key bump** (`cacheKeyVersion` v5 → v6) | All existing cached ifaces carry `Purity: true`; without the bump the old lie survives the fix for every cache hit | compiler (mechanical consequence) | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Option A (reject the contradiction) over desugaring (B) or advisory (C) — decided in this doc, per the triage recommendation and the corpus sweep (V11: only 7 sites in 4 files use `pure` + non-empty row, all `! {Declassify}`).
- [x] The rule covers open rows (`pure … ! {e}` rejected), not just labelled rows.
- [x] SMT admission is row-based with `smtTransparentEffects = {"Declassify"}`; `Debug` stays out.
- [x] Derived purity requires a closed, empty outer row **and** no quantified row vars in the scheme.

## Solution Design

### Overview

One rule — **`pure` ⇔ closed, empty declared effect row** — enforced at three points that today disagree:

1. **Rule 1 (signature):** `ValidateEffects` rejects a `pure` declaration whose elaborated row has labels or a tail row variable. This is where rows are already elaborated and compared; `IsPure` is available on the same surface `FuncDecl`.
2. **Rule 2 (iface):** `determinePurity` is replaced by a derivation from the export's generalised scheme — the type-checked row, not the Core expression. The builder already holds the scheme (`builder.go:367-386`); the triage doc's "plumbing is the main cost" estimate is obsolete — `scheme.Type`'s `TFunc2.EffectRow` and `scheme.RowVars` are both in hand (V13), so this is a local change.
3. **Rule 3 (SMT):** decidable-fragment admission keys on the declared row (`fd.Effects ⊆ smtTransparentEffects`, where the empty row trivially qualifies) instead of `meta.IsPure`. The existing `verify.go:116-148` fixup loop is the natural site — it already walks `surfaceFuncs` and mutates `meta.IsPure`; it becomes the single place that computes admission row-basedly, and `encodable.go`'s `isPure` is unchanged (it reads a now-honest `meta.IsPure`).

### Architecture

**Components:**

1. **Signature check** (`internal/pipeline/validate_effects.go`): in the `declaredEffects` build loop (`:110-122`), after elaborating the row, if `funcDecl.IsPure` and the row has `len(Labels) > 0` or `Tail != nil`, return an error in the existing plain-text diagnostic shape (no error-code namespace — effect-check diagnostics are uncoded today, V14): name the function, the contradiction, and the two fixes ("drop `pure`" / "drop the row"), mirroring the "Missing effects" block's actionability.
2. **Derived purity** (`internal/iface/builder.go`): `determinePurity(coreExpr)` → `derivePurity(scheme)`:
   - If `scheme.Type` is `*types.TFunc2`: pure ⟺ `EffectRow == nil || (len(EffectRow.Labels) == 0 && EffectRow.Tail == nil)` **and** `len(scheme.RowVars) == 0`. Computed *after* the M-EFFECT-ROW-SHOW-INTERP `RowVars` recompute (`builder.go:386-394`) so a restored source row variable is honoured.
   - Non-function exports: pure ⟺ no effect labels and no row variables occur anywhere in the scheme type (a type walk; see Deferred Decisions for the nested-open-row depth question).
   - The TODO comment and the "(for now, assume pure unless marked otherwise)" comment at the call site (`builder.go:395`) are deleted with it.
3. **SMT admission** (`internal/smt/verify.go` + a small shared predicate): `smtTransparentEffects = map[string]bool{"Declassify": true}` — justified by #1557's "reaches nothing on the host" (the policy layer already states it, `internal/policy/resolve.go:48-51`) and by the absence of any `Declassify` implementation in `internal/effects/` (V12). The verify.go fixup sets `meta.IsPure = rowIsSmtAdmissible(fd.Effects)` (empty row or row ⊆ transparent set), replacing the current empty-row-only inference; same-module and imported-module loops both change identically (`verify.go:116-125,140-150`).
4. **serve-api cleanup** (`internal/apiserver/`): `extractModuleInfo` (`server.go:471-489`) now produces the true value; the AST-heuristic overwrite in `extractMCPToolMetaAnnotations` (`routes.go:251`, `len(fn.Effects) == 0`) is removed along with its explanatory comment block (`routes.go:228-240`), which documents the distrust this doc removes. This also fixes the heuristic's open-row blind spot (`! {e}`-only rows read as pure today, since a row variable is not an AST `Effect`).
5. **Cache invalidation** (`internal/pipeline/cache_key.go:32`): `cacheKeyVersion = "v5"` → `"v6"`. Cached entries are re-checked on the mismatch (`cache_store.go:425-427` already errors loudly on version mismatch — no silent fallback).

### Implementation Plan

**Phase 1: Rule 1 — the signature check** (~4 hours)
- [ ] Add the `IsPure`-vs-row contradiction check to `ValidateEffects`'s declared-effects loop; diagnostic in the existing plain-text shape.
- [ ] Unit tests: `pure + ! {IO}` rejected; `pure + ! {}` clean; `pure` with no row clean; `pure + ! {e}` (row var) rejected; the contradiction error names both fixes.
- [ ] Regression sweep: the 782-line `pure func` corpus (V11) checks clean except the 7 known `! {Declassify}` sites, which are migrated in Phase 3.

**Phase 2: Rule 2 — derived iface purity** (~8 hours)
- [ ] Replace `determinePurity` with `derivePurity(scheme)`; delete the TODO and the "assume pure" comment.
- [ ] Handle non-function exports with the type walk.
- [ ] `ailang iface` acceptance tests: `sneaky`-style module → `"pure": false` with `"effects": ["IO"]`; `std/ai/streaming.ail` → all 6 exports `false` (V3 flips); `mapE` → `false`; `std/list.map` → measured, expected `false` (open callback row — the honest #1091/#1326 answer); unannotated pure functions and `! {}` functions → `true`.
- [ ] Run the std re-measurement (the V8 script) and record the before/after flip count in the PR.

**Phase 3: Rule 3, serve-api, cache, corpus** (~8 hours)
- [ ] Row-based SMT admission in `verify.go` (both loops), `smtTransparentEffects = {"Declassify"}`.
- [ ] Remove the serve-api AST-heuristic overwrite; `extractModuleInfo` becomes authoritative.
- [ ] `cacheKeyVersion` v5 → v6.
- [ ] Migrate the 7 corpus sites (`examples/runnable/contracts/inbox_injection_v2.ail` ×2, `examples/runnable/contracts/inbox_v2_app.ail` ×2, `benchmarks/prompt_injection/expected_ailang_injected.ail` ×1, `benchmarks/prompt_injection/expected_ailang_safe.ail` ×2): drop `pure`, keep `! {Declassify}`; re-run `ailang fmt` on the touched files (the formatter prints `pure` from `IsPure`, `format/decl.go:57`).
- [ ] Acceptance: `ailang verify benchmarks/prompt_injection/expected_ailang_safe.ail` → 3 verified, 0 violations (V5 inverted); `expected_ailang_injected.ail` → 1 violation on `injectedForward`.

**Phase 4: docs + tests + release hygiene** (~4 hours)
- [ ] `docs/docs/reference/effects.md`: state the rule explicitly at the `pure` examples (`:405-407`) and the pure-usable column (`:478`): "`pure` declares a closed, empty effect row; combining it with a non-empty or open row is a check error."
- [ ] Full `make test`, `make fmt`, `make lint`, `make check-boundaries` (touches `internal/iface` and `internal/apiserver`).
- [ ] Changelog entry; note the iface JSON semantic change (`"pure"` now derived) for downstream consumers.

### Files to Modify/Create

**New files:** none (no new packages; the transparent-effects predicate lives beside its only consumer in `internal/smt/`, or in `internal/types/` if the implementer prefers — Deferred).

**Modified files:**
- `internal/pipeline/validate_effects.go` (+~20 LOC) — Rule 1: the contradiction check in the declared-effects loop.
- `internal/iface/builder.go` (+~30/−15 LOC) — Rule 2: `derivePurity(scheme)` replacing `determinePurity`; comment cleanup.
- `internal/smt/verify.go` (+~15/−10 LOC) — Rule 3: row-based admission in both fixup loops.
- `internal/smt/encodable.go` (comment-only) — `isPure`'s doc comment now states the carrier is computed from the row by verify.go.
- `internal/apiserver/routes.go` (−~25 LOC) — remove the heuristic overwrite + distrust comment.
- `internal/pipeline/cache_key.go` (1 LOC) — version bump.
- `examples/runnable/contracts/inbox_injection_v2.ail`, `examples/runnable/contracts/inbox_v2_app.ail`, `benchmarks/prompt_injection/expected_ailang_{safe,injected}.ail` (7 lines total) — drop `pure`.
- `docs/docs/reference/effects.md` (+~5 LOC) — document the rule.

## Examples

### Example 1: The defect (issue #1443's repro)

**Before (v0.52.5, V2):**
```
$ ailang check sneaky.ail
→ Type checking sneaky.ail...
→ Effect checking...
✓ No errors found!
```

**After:**
```
$ ailang check sneaky.ail
→ Type checking sneaky.ail...
→ Effect checking...
Error: effect checking failed in test/sneaky: Effect checking failed for function 'sneaky'
  `pure` declares no effects, but the signature declares ! {IO}

  Current signature: pure func sneaky(...) -> unit ! {IO}
  Suggested fix:     func sneaky(...) -> unit ! {IO}        (drop `pure`)
  Suggested fix:     pure func sneaky(...) -> unit         (drop the row)
```

(Exact wording is a Deferred Decision; the shape — function name, the contradiction, both fixes — is not.)

### Example 2: iface honesty on std/ai (#574 part 1)

**Before (V3):**
```json
{
  "name": "callStream",
  "type": "(string,string,string)->Result[string,{code: string, message: string, retryable: bool}]!{AI,Net,Stream}",
  "effects": ["AI", "Net", "Stream"],
  "pure": true
}
```

**After:** `"pure": false` for all six exports of `std/ai/streaming.ail`; `"pure": true` only where the checked row is closed and empty.

### Example 3: The load-bearing lie and its migration (Rule 3)

**Before (V5, current dev):** `export pure func sanitizeBody(rawBody: string<email>) -> string<sanitized> ! {Declassify}` verifies its `ensures` contract — admitted by the keyword despite the row.

**After:** the declaration drops `pure` (Rule 1 rejects the old spelling):
```ailang
export func sanitizeBody(rawBody: string<email>) -> string<sanitized> ! {Declassify}
ensures { result == "[sanitized]" }
{
  "[sanitized]"
}
```
and `ailang verify` still verifies it, because admission now reads the row: `{Declassify} ⊆ smtTransparentEffects` — Declassify is a compile-time IFC gate with no runtime ops (#1557, V12), so the function's Z3 encoding is faithful. Without Rule 3 this migration would print `⚠ SKIPPED sanitizeBody — Function "sanitizeBody" has effects` (V5 measured exactly this failure mode).

## Success Criteria

- [ ] `sneaky` repro (`pure func … ! {IO}`) fails `ailang check` with a diagnostic naming the contradiction and both fixes.
- [ ] `pure func … ! {e}` (open row) also fails.
- [ ] `pure func … ! {}` and `pure func` with no row are unchanged (16 + 762 corpus sites check clean — V11 baseline).
- [ ] `ailang iface` reports `"pure": false` for any export with a non-empty or open checked row; std re-measurement: the 155 false-positives of V8 flip to `false`; `std/ai/streaming.ail` all 6 exports `false`.
- [ ] `ailang verify benchmarks/prompt_injection/expected_ailang_safe.ail` → 3 verified, 0 violations; `expected_ailang_injected.ail` → 1 violation on `injectedForward` (both after dropping `pure` from the 5 benchmark sites).
- [ ] serve-api `ExportInfo.Pure` equals the iface-derived value on the e2e MCP tests (no heuristic overwrite).
- [ ] Compile cache invalidated once by the version bump; no stale `Purity: true` served after rebuild.
- [ ] All tests passing (`make test`), `make fmt`/`make lint` clean, `make check-boundaries` clean.
- [ ] Documentation updated (`docs/docs/reference/effects.md` states the rule).
- [ ] Changelog entry for the iface JSON semantic change.

## Conflict Surface

### Syntactic positions touched

**None — there is no grammar change.** The `pure` keyword prefix on function declarations already parses (`parser_func.go:21-33`); this design adds a *semantic* check on the co-occurrence of `FuncDecl.IsPure` with a declared row, changes the *value* written into `IfaceItem.Purity`, and changes the *predicate* used for SMT admission. The positions affected are therefore semantic, not syntactic:

- **(a)** `pure func` declaration + explicit `! {…}` row — currently accepted when the row covers the body; will be rejected when the elaborated row has labels or a tail.
- **(b)** The `"pure"` field of the iface JSON for every export — value becomes derived.
- **(c)** `ailang verify` admission for functions whose row is non-empty — value becomes row-based.

### What else lives here

| Position | Existing valid form | Effect of this change |
|----------|--------------------|-----------------------|
| `pure` + no row | 762 corpus sites (e.g. `std/list.ail:58` `pure func map`, `std/math.ail:104` `pure func isNaN`) | Unchanged — `pure` *is* the empty row; the body check against the empty row already enforces purity (V6: an unannotated effectful body is rejected with "Missing effects: IO") |
| `pure` + `! {}` | 16 corpus sites (e.g. `examples/runnable/contracts/cross_module_types.ail:25`) | Unchanged — closed empty row is exactly what `pure` means |
| `pure` + `! {Declassify}` | 7 sites in 4 files (V11) | **Deliberately rejected** — the intentional incompatibility; migration: drop `pure`, keep the row (Rule 3 keeps them verifiable) |
| `pure` + `! {e}` (row var, no labels) | 0 corpus sites (V11) | Rejected — #1091's principle: a pure function must not export an open row |
| unannotated `func` + non-empty row | e.g. all 6 `std/ai/streaming.ail` exports | Check unchanged; iface `pure` flips `true` → `false` |
| unannotated `func` + open row (`! {e}`) | e.g. `std/list.ail:205` `mapE` | Check unchanged; iface `pure` flips `true` → `false` (the row variable is invisible in today's JSON — V9) |
| `pure` prefix on a *lambda expression* (`parsePureLambda`, `parser_lambda.go:79-91`) | Half-supported: fails to parse in let-binding positions (V7) and its purity marking is an unmodified TODO ("Mark as pure somehow") | Out of scope (Non-Goals); Rule 1 is keyed on `FuncDecl.IsPure` and does not touch expression-position `pure` |
| Ghost effect `Debug` | `ghostEffects = {"Debug"}` (`validate_effects.go:25-27`), transparent to callers | Unchanged — `pure + ! {Debug}` is rejected like any labelled row (0 corpus sites, V11); `Debug` stays out of `smtTransparentEffects` (it has runtime ops) |

### Disambiguation strategy

No parser disambiguation is needed (no grammar change). The checker-side decision is a single conjunction on data the pass already holds: `funcDecl.IsPure` (set only by the `pure` token) **and** the elaborated declared row (`ElaborateEffectRowWithBudgets(funcDecl.Effects)`) having `len(Labels) > 0 || Tail != nil` ⇒ error. Row variables are not AST `Effect`s, which is exactly why the tail check must be explicit — the serve-api AST heuristic (`len(fn.Effects) == 0`) misses the var-only case, and this design replaces that heuristic rather than replicating its blind spot.

### Programs that MUST still work

Regression fixtures (all verified to exist, V15):

1. `std/list.ail:58` — `export pure func map[a, b](f: a -> b, xs: [a]) -> [b] = _list_map(f, xs)` (pure, no row; its iface purity is *expected* to flip to `false` — the callback row is open — while its check result must stay clean)
2. `std/math.ail:104` — `export pure func isNaN(x: float) -> bool = x != x`
3. `examples/runnable/contracts/cross_module_types.ail:25` — `export pure func makeCell(text: string, count: int) -> Cell ! {}` (pure + explicit empty row)
4. `examples/intra_package_imports/service.ail:19` — `export pure func greet(name: string) -> string = "${greeting()}, ${name}!"`
5. `std/list.ail:205` — `export func mapE[a, b, e](f: a -> b ! {e}, xs: [a]) -> [b] ! {e}` (row-var export; check result unchanged)

### What deliberately changes

1. **7 `pure … ! {Declassify}` declarations are rejected** (2 example files, 2 benchmark reference files). Migration: drop `pure`, keep the row. This is the entire user-visible breaking change; it requires a minor version bump (v0.53.0), consistent with the latent-values precedent of rejecting programs that compile today.
2. **The iface `"pure"` field changes meaning** from "always true" to "derived from the checked row". 155 std exports flip to `false` (V8), plus every open-row export. Downstream consumers that gated on `pure == true` will see honest `false`s — that is the point; the known consumers (serve-api MCP hints, `@mcp_hints`) already distrust the field.
3. **`ailang verify` admission predicate changes** from keyword to row. Net effect on the corpus: none (V5 acceptance); net effect on hand-written `pure + ! {IO}` functions: they no longer verify — they no longer *check*.

## Testing Strategy

**Unit tests:**
- `internal/pipeline`: the Rule 1 matrix (pure+IO / pure+{} / pure+no-row / pure+`! {e}` / non-pure+row / non-pure+no-row), asserting the diagnostic names both fixes.
- `internal/iface`: `derivePurity` over schemes — closed empty (true), labelled (false), open (false), row-var-quantified (false), non-function value export (true), non-function export containing a function type with a row (false).
- `internal/smt`: admission matrix — `! {}` admitted, `! {Declassify}` admitted, `! {IO}` refused, `! {Debug}` refused, `! {Stream, e}` refused.

**Integration tests:**
- The sneaky module end-to-end: `ailang check` fails; after dropping `pure`, `ailang iface` reports `"pure": false` with `"effects": ["IO"]`.
- `std/ai/streaming.ail`: all exports `false`; `std/list.ail` `map`/`mapE` measured and pinned (see M1).
- The two prompt-injection benchmark files: verify outcomes pinned (3 verified / 0 violations; 1 violation on `injectedForward`).
- serve-api e2e (`internal/apiserver/mcp_*_test.go`): `ExportInfo.Pure` equals the derived value; the `[pure]` MCP tag and read-only hints reflect it.

**Regression-surface tests** (per Conflict Surface):
- One pinned test per "Programs that MUST still work" fixture (1–5 above), asserting the check result and (for the flipped ones) the new iface value, so the flip is intentional and visible in the diff.

**Manual testing:**
- The V8 std re-measurement script run before/after; counts recorded in the PR.
- A stale-cache round trip: build with v5 cache present, run with v6 binary, confirm re-compile (no stale `Purity: true`).

## Deferred Decisions

- **Diagnostic wording for Rule 1** — agent may choose, within the shape (function name, contradiction, both fixes) and the existing plain-text effect-diagnostic style.
- **Home of `smtTransparentEffects`** (`internal/smt` beside its consumer vs `internal/types` for reuse) — agent may choose; if a second consumer appears, promote it then.
- **Depth of the type walk for non-function exports** (shallow "no labels anywhere" vs deep "no open rows anywhere") — agent may choose, with tests; the corpus has no deciding case today.
- **Whether to also assert `meta.IsPure` ⇔ empty declared row as a defense-in-depth invariant inside `internal/types/typechecker.go:115`** — agent may choose; Rule 1 already guarantees the premise pipeline-wide (both compile paths call `ValidateEffects`, V16).

## Non-Goals

- **`pure` on lambda expressions** — `parsePureLambda`'s purity marking is a TODO and the syntax fails to parse in common positions today (V7); a separate gap, not made worse or better by this doc.
- **M-EFFECT-LATENT-FUNCTION-VALUES (#1326/#573)** — charging latent effects of function-valued arguments. Same cluster, different mechanism (empty read sites in the effect pass); this doc's derived purity *reports* the honest `false` for effect-polymorphic combinators, which composes with that doc's caller-charging fix. Do not merge the sprints.
- **M-EFFECT-ROW-VAR-UNIFICATION (#616)** — parked in v1_0_0; row-var unification is orthogonal to the keyword-vs-row contradiction.
- **M-EFFECT-PURE-ROW-OVERGENERALIZATION (#1091)** — that planned doc stops a `pure func` from *exporting* an effect-polymorphic row via generalisation. Rule 1's tail check front-runs the declared-spelling half (`pure … ! {e}`), and derived purity reports those exports `false`, but #1091's inferred-row mechanism (recursive calls sharing the row var) is untouched and remains that doc's to fix.
- **Widening `smtTransparentEffects`** beyond `Declassify` (e.g. `Debug`) — evidence-gated future work; `Debug` performs runtime ops and is excluded on purpose.
- **Making `Declassify` a ghost effect** — rejected: ghost means "callers never need to declare it" (V6-measured the opposite: a caller of a `! {Declassify}` function must declare `Declassify`); IFC tracking must keep propagating.

## Timeline

**Week 1** (3 days):
- Phase 1: signature check + unit matrix (0.5 d)
- Phase 2: derived purity + std re-measurement (1 d)
- Phase 3: SMT admission, serve-api, cache, corpus migration (1 d)
- Phase 4: docs, full test/lint/boundaries, changelog (0.5 d)

**Total: ~3 days** (estimates doubled from first-pass guesses per skill guidance).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Unknown consumers of iface `"pure": true` outside this repo (motoko, package registry) see honest `false`s and change behaviour | Medium | The known in-repo consumers already distrust the field (serve-api heuristic, `@mcp_hints`); the changelog entry names the semantic change explicitly; the V8 flip list (155 std exports) is published in the PR for out-of-repo consumers to diff against |
| `std/list.map` flipping to `false` reads as a regression to users of "pure combinators" | Medium | It is the honest answer per #1091/#1326 (an unannotated callback row is effect-polymorphic); documented here and in the changelog; `mapE` already exists for the explicit-row spelling |
| Rule 1's tail check mis-fires on a row form the corpus uses but the sweep missed (e.g. budgeted rows `! {Rand[mode=os]}`) | Low | The check keys on `Labels`/`Tail` of the *elaborated* row, after budgets; the full-corpus `ailang check` sweep over std/ + examples/ + benchmarks/ is an M1 gate, and budgets are a declared-row property the sweep would catch |
| Cached ifaces serve stale `true` after upgrade | Low | Single `cacheKeyVersion` bump; the store already fails loudly on version mismatch (`cache_store.go:425-427`) |
| Dropping `pure` from the benchmark references changes `ailang verify` outcomes beyond the pinned pair | Medium | V5 measured the exact failure mode pre-migration; the pinned verify outcomes for both files are acceptance criteria |

## Related Documents

**Planned (check for overlap):**
- [m-effect-latent-function-values.md](../v0_48_0/m-effect-latent-function-values.md) — #1326/#573, the same soundness cluster: latent effects of function values read from silently-empty sources in the same `validate_effects.go` pass. Different mechanism (empty read sites vs unchecked keyword/stub); both must land for the `pure` signal to be honest end-to-end. The neural/simhash search surfaced no direct match for this doc's topic (V16 note: the create script's search ran with no results; the related set was assembled from the triage doc's own search, which named exactly these).
- [m-effect-pure-row-overgeneralization.md](../v0_35_3/m-effect-pure-row-overgeneralization.md) — #1091, the effect-polymorphic-row half; see Non-Goals for the division of labour.
- [m-effect-row-var-unification](../v1_0_0/m-effect-row-var-unification.md) — #616, parked; unrelated mechanism.
- [m-serveapi-directory-ready.md](../v0_51_0/m-serveapi-directory-ready.md) — noted #1443 as out-of-scope while landing M1 (`:362,459`); this doc is the follow-up it pointed at.

**Triage provenance:**
- [pure-keyword-vs-declared-row-and-iface-purity.md](../ailang-core-triage/pure-keyword-vs-declared-row-and-iface-purity.md) — the 2026-10-03 triage that recommended this design doc; its options A/B/C analysis is adopted (A) and its "plumbing is the main cost" estimate for the iface builder is corrected by V13.

## References

- **Issues**: [#1443](https://github.com/sunholo-data/ailang/issues/1443) (open — `Closes #1443` belongs to the implementation PR only, per the scheduling directive), [#574](https://github.com/sunholo-data/ailang/issues/574) (closed; part 1 covered here), [#1091](https://github.com/sunholo-data/ailang/issues/1091), [#1326](https://github.com/sunholo-data/ailang/issues/1326), [#573](https://github.com/sunholo-data/ailang/issues/573), [#1557](https://github.com/sunholo-data/ailang/issues/1557) (Declassify is compile-time, host-transparent). **Do not open a new issue** — the sprint plan and PR reference these with `Refs #N`.
- **Docs**: [effects reference](../../../docs/docs/reference/effects.md) `:405-407` (the `pure` examples this rule makes literal) and `:478` (the pure-usable column); `internal/apiserver/routes.go:228-240` (the distrust comment this doc retires).
- **Axioms**: [Design Axioms](/docs/references/axioms)
- **Prior art in-repo**: `internal/testing/source_strip.go:72` already reads purity disjunctively (`f.IsPure || empty row`) — the row-based reading Rule 3 generalises.

## Verification Log

Every load-bearing claim above, checked against the code or the live binary (`AILANG v0.52.5`, worktree `62ac2d09`):

| # | Claim | Method | Result |
|---|-------|--------|--------|
| V1 | Repro base | `git log --oneline -1` / `ailang --version` | worktree `62ac2d09`, binary v0.52.5 (superset of triage `790169359` and coordinator `658ff76a3`) |
| V2 | `pure func … ! {IO}` passes check | `ailang check` on the sneaky module | `✓ No errors found!`, exit 0 |
| V3 | std/ai reports pure alongside effects | `ailang iface std/ai/streaming.ail` | all 6 func exports `"pure": true` with non-empty `"effects"` (e.g. `callStream` `["AI","Net","Stream"]`) |
| V4 | The exported row still propagates (no erasure) | `ailang check` on a caller importing the sneaky module | caller **is** charged `Missing effects: IO` — bounds the typechecker generalisation claim to "keyword gates a premise", not "row is erased" |
| V5 | The lie is load-bearing for `ailang verify` | copied `expected_ailang_safe.ail`, dropped `pure` from the two `! {Declassify}` functions, `ailang verify` | `⚠ SKIPPED sanitizeBody — Function "sanitizeBody" has effects` / same for `safeForward`; original with `pure`: 3 verified, 0 violations |
| V6 | Unannotated effectful bodies are rejected; `Declassify` propagates to callers | two `ailang check` runs (no-row fn calling `println`; no-row fn calling a `! {Declassify}` fn) | both fail with `Missing effects: IO` / `Missing effects: Declassify` |
| V7 | `pure` on lambda expressions is half-supported | `ailang check` with `let f = pure func(x: string) -> unit ! {IO} { … }` and variants | parse errors in every tried position; `parsePureLambda` ends in a TODO (`parser_lambda.go:89-90`) |
| V8 | 155 of 468 std func exports report pure with non-empty effects | `ailang iface` over all 46 `std/*.ail`, JSON-parsed count | 155/468 (33%); 0 additional var-only-open rows are *visible* in the JSON (see V9) |
| V9 | Open rows are invisible in the iface JSON | `ailang iface std/list.ail`, `mapE` entry | type string `((a)->b,list[a])->[b]`, `effects: []`, `pure: true` — the declared `! {e}` appears nowhere |
| V10 | `ImportedSym.Purity` has no reader | `grep -rn "\.Purity" internal/ cmd/` minus assignment sites | only definitions/assignments (env.go:12, module_linker.go:151, builtin_module.go:51,185); no read site |
| V11 | Corpus sweep of `pure func` + non-empty row | `grep -rn "pure func" std/ examples/ benchmarks/` filtered by `! {…}` | **7 sites / 4 files, all `! {Declassify}`**: inbox_injection_v2.ail ×2, inbox_v2_app.ail ×2, expected_ailang_injected.ail ×1, expected_ailang_safe.ail ×2; std: 0; `pure` + `! {}`: 16; `pure` no row: 762; `pure` + `! {e}`: 0 |
| V12 | `Declassify` has no runtime ops | `grep -rn "Declassify" internal/effects/` | only `validate_effects.go` (no implementation file); `internal/policy/resolve.go:48-51` documents it as compile-time with no host reach (#1557) |
| V13 | The builder already holds the type-checked row (triage's "plumbing" estimate obsolete) | read `builder.go:367-401`, `types_v2.go:64-68` | `scheme.Type` is `*types.TFunc2` with `EffectRow` (`Labels`, `Tail`), `scheme.RowVars` recomputed at `:386-394`; `determinePurity`'s argument is simply unused today |
| V14 | Effect-check diagnostics carry no error codes (no allocation needed) | `grep` for coded errors in `validate_effects.go` / effect errors repo-wide | plain `fmt.Errorf` strings ("Missing effects: …"); `MOD`/`PAR`/`TC` codes unused by the effect pass; no `EFF` code space exists (only a test-file `EFF0` false positive) |
| V15 | The five regression fixtures exist | `sed -n` on each cited line | all five present at the cited lines and spellings |
| V16 | Both compile paths run `ValidateEffects`; the create-script search found no related docs | `grep -rn "ValidateEffects(" internal/ cmd/`; script run | `pipeline_module_compile.go:338` and `pipeline_single.go:399` (+ `repl_eval.go:125`); the skill's search script exited 1 on its no-matches path (known first-occurrence friction, reported) and returned no neural/simhash matches — the related-doc set comes from the triage doc's own prior search |

## Future Work

- `ailang design-quorum` on this doc (optional pre-sprint step) — cheap off-Anthropic reject-by-default review before an implementation sprint is spent.
- Widening `smtTransparentEffects` if a second host-transparent effect is ever added — the predicate's home should then be promoted out of `internal/smt`.
- `pure` on lambda expressions (`parsePureLambda`'s TODO) — decide whether to implement or remove the expression-position keyword.
- A lint/doctor pass that flags iface consumers still keying on the removed heuristic.

---

**Document created**: 2026-10-08
**Last updated**: 2026-10-08
