# M-EFFECT-EXPLICIT-EMPTY-CALLBACK-CONTRACT

**Status:** Design APPROVED — Mark, attended, 2026-10-29: “Approve option A design”. Sprint planning authorized; implementation still requires explicit execution.
**Target:** v0.56.0 (breaking minor; no version bump/deployment in this proposal).
**Priority:** P0 compatibility decision, blocking #1958 M4.
**Estimated:** Recommended option: 1–2 engineering days including corpus, cache, and consumer verification; not a sprint approval.
**Dependencies:** #1963 nested charging is merged at `87f846980c8b6fc601173be427b1364968559488`. #1958 remains unmerged. Its previously approved one-line motoko migration is already delivered.
**Scope:** A focused follow-up/amendment prompted by real consumer evidence. It does not reclassify #1963's ratified conservative rule as an implementation defect.

## Problem and verified boundary

The final #1958+#1963 consumer sweep checks 60 motoko core entries. Five now fail through `src/core/test/ext_fixture.ail:fixture_rt`, a pure factory calling `fixture_hooks(id, ov: FixtureOverrides)`. The helper stores callbacks in a registration record. `FixtureOverrides` mixes four implicit/open pure callback arrows with four explicitly effectful callback arrows. #1963's approved per-parameter boolean mask sees an open arrow; its per-argument row unions all nested arrows. Charging this mixed hand-off follows the frozen conservative algorithm.

This is distinct from the already resolved #1958 `play_chunks` width migration. Before #1963, the delivered motoko source checks 60/60 with #1958. After combining #1963, it checks 55/60 and reports all eleven callback effects on `fixture_rt`: AI, Clock, Env, FS, IO, Net, Process, Rand, SharedMem, Stream, Trace. The five importing entries are agent_loop_v2, dst_execution, rpc, session, supervisor.

The documented all-concrete storage migration is unavailable for a pure callback arrow today: writing `! {}` still elaborates as implicit/open. The attempted exact four-line consumer patch adding `! {}` to describe/prompt/budget/policy **fails unchanged**. That hypothesis is preserved, not delivered. V1–V6 below verify these claims.

## Four-arm evidence

All checks use `AILANG_NO_CACHE=1`, each binary's matching stdlib, and freshly archived exact source for the rebuilt arms.

| Arm | Source commit |
|---|---|
| Base | `c04a44780f32b6af98905a46191c104b1d5b32ce` |
| #1958 only | `752cebf3cf09202c01ec74d6ce1172a9c39b2294` |
| #1963 only | `87f846980c8b6fc601173be427b1364968559488` |
| Combined | `970a4669470566163cba09892a351e639b6de60c` |

The base is the earlier measured dev checkpoint; the two feature-only arms and combined arm contain later package-tooling changes. The reduction imports only std/io; the actual consumer comparison uses the same pinned migrated source in every arm. This table does not claim identical non-feature commit ancestry.

| Checked case | Base | Row only | Nested only | Combined |
|---|---|---|---|---|
| Actual `fixture_rt`/`fixture_hooks` | accept | accept | reject | reject |
| Actual fixture + four pure-arrow `! {}` annotations | accept | accept | reject | reject |
| Mixed record hand-off: implicit pure + concrete IO | accept | accept | reject | reject |
| Same mixed record; pure field explicitly `! {}` | accept | accept | reject | reject |
| Record with one concrete IO field | accept | accept | accept | accept |
| Record with two concrete IO fields | accept | accept | accept | accept |
| Direct record literal storage | accept | accept | accept | accept |
| Hand-off with all implicit arrows and an IO value | accept | accept | reject | reject |
| Explicit empty callback slot receives IO value from an IO caller | accept | accept | accept | accept |

The last row is a deliberate compatibility change under the recommended option, not an existing rejection.

### Exact minimal reproducer

```ailang
module mixed
import std/io (println)
pure func inc(x: int) -> int = x + 1
func noisy(x: int) -> int ! {IO} = { println("called"); x }
type Hooks = { describe: (int) -> int, provide: (int) -> int ! {IO} }
pure func keep(h: Hooks) -> Hooks = h
pure func factory() -> Hooks = keep({describe: inc, provide: noisy})
```

Change describe to `(int) -> int ! {}`: same observed rejection. Replace `keep({...})` by the literal `{...}`: accepted in all four arms. Give both fields nonempty concrete IO contracts and pass noisy to both: accepted in all four arms. These are measured distinctions, not proposed per-leaf precision rules.

The actual constructor simply stores `ov.describe`, `ov.prompt`, `ov.budget`, `ov.compact`, `ov.policy`, `ov.provide`, `ov.intercept`, and `ov.judge` in capability constructors. `fixture_rt` passes named callbacks and returns `{registry: {entries: [hooks]}, strict_mode: false, verification: v}`. Full exact sources remain at the pinned fork commit; no source beyond the approved `play_chunks` line was delivered.

## Goals and non-goals

Provide an explicit, sound pure callback contract so the already documented all-concrete storage pattern can be expressed, while retaining omitted-arrow polymorphism and #1963's intentional store-only hand-off charging.

Success means the four explicit pure annotations become a valid, measured consumer migration; original mixed/unannotated hand-offs remain conservative; an IO value cannot enter an explicit closed-empty slot even if the enclosing caller has IO. Final consumer acceptance must restore all 60 entries and 9 extension boots with the actual combined compiler before #1958 M4 passes.

This proposal does not authorize per-leaf charging, width subsumption, removal of #1958 guards, broad consumer effect widening, a global pure-keyword redesign, or changes to #1790's outer iface purity algorithm.

## High-impact decisions and options

| Option | Outcome and cost | Recommendation |
|---|---|---|
| A. Honor explicit empty function-type rows as closed, concrete pure contracts | Reuses existing syntax and AST presence. Distinguishes omission from `! {}`. Breaks formerly accepted IO-to-empty-slot programs; requires conversion/import/cache parity and negative tests. Then separately approve/deliver the exact four-line consumer annotation patch. | Recommended, subject to human approval and sprint gate. |
| B. Replace parameter-wide charging with callee-position ownership | Can accept mixed records without source annotations, but changes the ratified mask/union design. Needs callee-position ownership through aliases, records, tuples, containers and generic ADTs. Filtering actual argument metadata alone is unsound: an explicitly IO argument in an implicit callee slot still must charge. | Larger separate precision design; not an implementation shortcut here. |
| C. Retain current semantics and rewrite/widen consumer factories | Widening factory rows may cascade through many callers; literal inlining duplicates the test-owned constructor. Neither has a validated full-corpus patch. | Not review-ready; no authority to deliver it from this document. |
| D. Hold consumer enablement / #1958 M4 | Leaves the compatibility failure explicit while a choice is made. No compiler or new consumer changes. | Safe current state until approval. |

### Design freeze — RULED 2026-10-29 (Mark, attended)

- [x] **F1:** Option A approved. B/C are not authorized; hold applies until execution and proof.
- [x] **F2:** `(T) -> U ! {}` is **closed empty**; omitted arrows remain implicit/open; nonempty and row-polymorphic contracts retain their current rules.
- [x] **F3:** Approve the measured IO-to-explicit-empty-slot rejection as a v0.56.0 breaking change, with migration by removing an unintended annotation or declaring the actual callback row.
- [x] **F4:** Approve the exact four pure-field consumer annotations conditionally: deliver only after the implemented contract proves the actual 60-core and 9-extension gates. Further consumer edits require a new decision; no delivery before execution and proof.

All four design decisions are ruled. No implementation until the concrete sprint plan is approved and the user says to execute. Optional design quorum has not been run; parent/operator review is handling this decision.

## Recommended solution A

### Reuse the existing representation

The parser already preserves absence versus explicit emptiness: `parser_type.go` initializes `var effects` to nil when no `!` appears; `parseEffectAnnotation` initializes a non-nil empty slice. The canonical formatter's `formatEffectRow` already preserves this distinction. A new parser token, grammar production, or AST presence flag is unnecessary (V7).

The Core elaborator currently tests `len(typ.Effects)>0`; zero length creates an open compatibility tail and marks the arrow implicit, so `! {}` and omission collapse (V8). Add a distinct explicit-empty branch before the existing nonempty path:

| Source function type | Required contract under A |
|---|---|
| No annotation | Existing implicit/open row; unchanged |
| `! {}` | Closed empty row, ConcreteEffectContract=true, ImplicitEffectContract=false |
| Concrete nonempty `! {IO}` | Existing L1-open compatibility tail and concrete-ownership behavior; unchanged |
| `! {e}` / `! {IO,e}` | Existing declared polymorphic row sharing; unchanged |

Do **not** merely set ConcreteEffectContract on an open empty row. Closure is necessary: accepting IO by unifying it into an empty compatibility tail and later erasing that tail could hide IO. The IO-to-empty-slot negative is a required discriminator with an IO-capable enclosing caller and runtime caps, on both engines before output.

### Conversion, import and persistence audit

This is not a one-line-only change. Core conversion is in `internal/elaborate/file_funcs.go`; the legacy AST checker also converts FuncType in `internal/types/typechecker.go`; iface alias/constructor conversion is in `internal/iface/builder.go`. Those paths currently use length tests; the latter already publishes ownership flags. Audit and align explicit-empty handling in every used path while preserving each path's existing omitted/nonempty behavior. Do not use this work to convert every legacy omitted arrow to a new rule (V9).

`ast.FuncType.String()` uses `ast.FormatEffects`, which currently omits both nil and empty. Preserve explicit emptiness in this string path where used; avoid changing all function-declaration pretty printing accidentally. Canonical `internal/format` already handles empty rows and should retain round-trip behavior (V10).

TFunc2 substitution and JSON persistence already carry both ownership flags. The cache key includes compiler identity and dependency iface digests. Reuse these mechanisms; prove cold/warm/imported alias behavior, and invalidate through the existing compiler identity rather than inventing a second cache (V11). Do not claim an existing unknown AST clone mechanism needs replacement; the parser/formatter and known conversion/substitution paths are enumerated here.

### Candidate files and phases (estimate, not sprint authorization)

1. Failing-first conversion/matrix tests and explicit-empty contract implementation: elaborate, legacy types conversion, iface conversion; about 80 production and 180 test lines.
2. Function-type string/formatter, imported aliases and cold/warm cache preservation; about 30 production and 120 test lines. Existing format behavior is a control, not a rewrite.
3. Re-run both semantic fixes, rejection-before-output tests, all 49 interfaces, examples/manifest and the actual pinned consumer. If safe, submit the separately approved exact four-line fork patch, pin its full merged SHA, record 60/60 + 9/9 and independent evaluation. Allow 0.5–1 day for corpus and review.

## Conflict surface

| Position / existing behavior | Preserve or deliberate change | Required fixture |
|---|---|---|
| Omitted callback arrow in record/list/tuple/Option | Retain implicit hand-off charging | Existing `TestLatentNested_LeaksRejected`, `TestLatentNested_IntendedFlips`; read bodies, including keepAll controls |
| Explicit nonempty concrete nested arrows | Preserve all-concrete storage acceptance | Measured single_concrete/all_io reductions; existing store_only_concrete_annotation_ok case |
| Explicit empty callback arrow | Deliberately becomes closed pure, not inferred effect polymorphism | empty_slot_io negative and pure callback positive; direct parameter, alias, imported alias, tuple/list/Option/record positions |
| Row-polymorphic callbacks | Preserve caller/callback tail relationship | Existing `TestAIStreamCallbackEffectRows`, rowpoly controls in `TestLatentNested_Controls` |
| Direct literal storage and length over callback containers | Preserve acceptance | Existing ok_storage case and measured literal reduction |
| Pure declarations / #1790 outer purity | No new declaration rule here | Existing std pure combinators; separately check iface changes rather than copying older goldens |
| Parser/formatter representation | Keep nil versus non-nil empty slice and printed `! {}` | Existing formatter effect-row controls plus new FuncType string round trip |

Actual consumer callbacks use multi-argument arrows, imported types, return ADTs, and named effectful values; the final compatibility gate must cover those actual sources, not just the int reduction. Evaluator and bytecode acceptance/rejection must agree; generated Go is outside this amendment.

## Acceptance criteria

- [ ] A1: omission, explicit empty, concrete nonempty and row-polymorphic arrows remain distinct through conversion, alias expansion, substitution, iface serialization and cold/warm compilation.
- [ ] A2: both existing feature regression families pass; unannotated nested store hand-offs remain rejected, and direct literals/length/all-concrete controls remain accepted.
- [ ] A3: IO-to-explicit-empty-slot fails even from an IO caller before output on evaluator and bytecode; matching pure callback passes. Test direct and imported aliases plus nested container positions.
- [ ] A4: exact four pure-field annotations in the actual pinned consumer restore every previously passing core entry; all 60 core files and 9 extension boots pass. If they do not, stop for classification rather than widen rows silently.
- [ ] A5: zero unexplained corpus flips; old/new behavior of existing explicit-empty function types is inventoried. Every intentional rejection is reported, not counted as a pass or hidden by a cache.
- [ ] A6: focused lint, boundaries, sizes, examples/manifest and all interface gates pass; regeneration preserves intervening #1790 metadata if it lands. Final independent evaluator decides acceptance.

## Axiom compliance

| Axiom | Score | Reason |
|---|---|---|
| A1 Determinism | +1 | Explicit syntax fixes a stable contract across cold/warm/import paths |
| A2 Replayability | 0 | No runtime trace change |
| A3 Effect legibility | +1 | Empty callback contract cannot silently mean an effect-polymorphic slot |
| A4 Explicit authority | +1 | IO cannot enter an explicitly pure slot; no blanket consumer widening |
| A5 Bounded verification | 0 | Local conversion rule and existing row guard |
| A6 Safe concurrency | 0 | No scheduler/runtime change |
| A7 Machines first | +1 | Same syntax survives parse/print/iface/cache and has a testable meaning |
| A8 Minimal syntax | +1 | Reuses `! {}` and existing nil/non-nil representation |
| A9 Cost visibility | 0 | No cost-accounting change |
| A10 Composability | +1 | Concrete pure/effectful storage fields compose under existing mask rule |
| A11 Structured failure | 0 | Existing type/effect error paths |
| A12 System boundary | 0 | Source/compiler/fork delivery boundaries remain explicit |

Net +6; no negative hard-axiom score. These are projected scores for A, not implementation results.

## Verification log and reproducibility

| ID | Claim and verified evidence |
|---|---|
| V1 | Four-arm reduction: `1958-nested-mixed-matrix.json/log`; exact source strings in companion evidence JSON. Cases mixed/concrete/literal/implicit. |
| V2 | Actual fixture four-arm check: `1958-fixture-rt-four-arm.json/log`; PASS/PASS/FAIL/FAIL, identical missing effect set on nested/combined. |
| V3 | Failed exact four-line pure annotation hypothesis: `1958-nested-four-pure-matrix.json/log`, archived scratch `1958-nested-consumer-four-pure`; not delivered. |
| V4 | All-concrete nonempty controls: same matrix, single_concrete/all_io accept in every arm. |
| V5 | Explicit empty slot accepts IO today: `1958-empty-slot-io-matrix.log`, all four exit 0. Future rejection is explicitly a new decision. |
| V6 | Actual combined core: `1958-final-motoko-core-summary.json`, 55/60; before nested merge, delivery 60/60 recorded in #1958 validation. Same consumer source c3f4c85e, tree-identical to merge af1d5651. |
| V7 | Read parser_type.go's three FuncType sites and parser_effect.go's empty-slice construction; canonical formatter `formatEffectRow` explicitly preserves nil/non-nil. |
| V8 | Read elaborate/file_funcs.go FuncType branch: length gate, fresh ε_annot, Concrete=false/Implicit=true on zero length. Unchanged in the source comparison between base and nested-only (independent review corroborates). |
| V9 | Read types/typechecker.go conversion and its parameter-declaration caller; iface/builder.go conversion and alias/constructor call sites. Both use length predicates; do not infer they share one implementation. |
| V10 | Read ast/ast_type.go FuncType.String and ast/ast.go FormatEffects: zero length omits annotation. Read format/types.go's separately correct presence check. |
| V11 | Read types_v2.go TFunc2.Substitute, unification_substitution.go, types/json.go ownership fields, pipeline/cache_key.go compiler identity/dependency digests. Test persistence rather than assume it. |
| V12 | Read #1963 design214/223/333/334 and existing test bodies: parameter-wide charging and store-only flips were approved; AC4 literal storage remains accepted. |
| V13 | Related search ran through creator script and standalone neural command. Runtime reported model=fallback-simhash, zero embeddings; high unrelated scores are not neural duplicate proof. Direct reading identified the relevant docs below. |

All raw logs are under `/tmp/ailang-backlog-20261029/`; the companion evidence JSON commits source strings, exact compiler SHAs and observed outcomes so the premise survives scratch cleanup. Reproduce with `ailang check <case>.ail` under the matching source's stdlib and `AILANG_NO_CACHE=1`. No new diagnostic identifier is allocated by this proposal.

## Related documents and non-duplication

- [Nested callbacks](../../implemented/v0_56_0/m-effect-latent-nested-callbacks.md): owns conservative parameter-wide charging. This follow-up supplies the missing explicit-empty callback contract; it does not reopen that rule implicitly.
- [#1958 design](../v0_55_0/m-effect-row-open-closed-soundness.md) and its validation: owns label preservation and the delivered play_chunks migration; M4 remains pending this decision.
- [Pure rows / iface purity](../v0_53_0/m-pure-row-and-iface-purity.md): planned #1790 owns outer pure declarations and derived iface purity. Callback **function-type** empty presence is the distinct topic here; coordinate touched conversion/iface files and regenerate combined goldens.
- [Formatter round-trip](../../implemented/v0_30_0/m-fmt-properties-printer-roundtrip.md): already preserves nil-versus-empty formatting; reuse it, do not create a competing parser representation.
- [Effect-row show/interpolation](../../implemented/v0_31_0/m-effect-row-show-interp.md): protects callback-body effect checking; this proposal is about the callback type's contract at hand-off, including a caller that already declares IO.
