# M-EQ-DERIVE-CONTAINERS — make `==` work for records, Options and lists when the parts already have Eq

**Status**: Implemented 2026-09-22; evaluated 2026-09-23 (PASS 88/100, round-1 soundness finding fixed). Was: Planned — **RE-LAND**. First implementation `caa2d53a6` was reverted same-day in
`cd1c976fb` (2026-08-29); nothing has landed since. Read **Re-land revision (2026-09-22)** first:
it supersedes the original Solution Design, ABI, Implementation Plan, Files and Success Criteria
wherever they conflict.
**Target**: next minor (originally v0.35.0; missed — current line is v0.41)
**Priority**: P1
**Estimated**: ~4 hours (smaller than the original ~6h — see R-D1)
**Dependencies**: None (builds on M-DX19's existing `deriving (Eq)` machinery)
**Tracking**: GitHub issue #960 (filed from the M-DX-PI-HARNESS dogfooding run); #963 (list `==`)

## Re-land revision (2026-09-22)

### Why this doc is back

The design passed two quorum rounds on 2026-08-29 and was implemented the same day
(`caa2d53a6`), then reverted within hours (`cd1c976fb`). Post-hoc verification found its
"probes verified live" claim false. Records never type-checked. Lists crashed at runtime.
The one working surface, a locally declared Option-like ADT, already worked before it.
The depth cap was dead code: `if depth > 0 && false`. The failure map is in issue #960's
comment. It then sat for 24 days. The mission's weekly sweep listed #960 and #963 among
open issues with zero mentions, but no iteration picked them up.

It resurfaced on 2026-09-22 while calibrating three new hidden-input frontier benchmarks.
On `mlfq_scheduler_hidden`, **4 of 6 compile failures were `==` on a list**
(`claude-opus-5-5`, `gpt6-sol` ×2, `gpt6-luna`), e.g. `if st.q0 != [] then …`. The error
also told every model to "Import std/prelude", a module that does not exist. That hint was
fixed in `2122705e0`: missing-`Eq` errors now give advice that works for each type. The
language gap itself is what this doc closes.

**Frequency, measured honestly.** On the existing suite the gap is rare in *final* results:
banked baselines v0.32.0 and v0.30.0 hold 1 of 877 and 2 of 909 AILANG standard rows with a
missing-`Eq` error. That is a floor, not a rate. A banked row keeps only the final attempt's
stderr, so a run that hit the error and repaired it leaves no trace. It bites hardest on
stateful, collection-heavy programs (queues, sets of states, record snapshots), which the
old suite under-samples and the new frontier family targets. It is also the #960 DX case:
every test assertion on a record, Option or list needs a hand-written match helper.

### Verification Log — re-land rows (live, 2026-09-22, dev + `2122705e0`)

| # | Claim | Method | Result |
|---|-------|--------|--------|
| V12 | `[1,2] == [1,2]` fails at **check time** (was a runtime crash in the 08-29 matrix) | `ailang check` + `run` | `No instance for Eq[[int]]`, a check-time type error. The #963 crash no longer reproduces as a crash |
| V13 | `xs == []` on `xs: [int]` fails | probe | `No instance for Eq[list[int]]` |
| V14 | Built-in `Option[int]` has no Eq: `Some(1) == Some(1)` fails | probe | FAILS. The 08-29 matrix's ✅ Option row was for a *locally declared* Option-like ADT, not `std` Option |
| V15 | `Option[C]` with `C deriving (Eq)` fails | probe `Some(Red) == Some(Red)` | FAILS (container composition is the blocker, as V5 said) |
| V16 | Tuples have no Eq, **not in the original scope** | `(1, "a") == (1, "a")` | `No instance for Eq[(int, string)]` |
| V17 | Record alias with `deriving (Eq)` still fails; the constraint carries a residual row | `type P = {x:int, y:int} deriving (Eq)`; `mk(1) == mk(1)` | `No instance for Eq[{ x: int, y: int, ...{x: int, y: int} }]` (root cause 2 on #960, unchanged) |
| V18 | **Derived ADTs whose fields are lists, Options or tuples ALREADY compare correctly at runtime** | `B([1,2])==B([1,3])`, `B([1])==B([1,1])`, `O(Some 1)==O(None)`, `O(Some 1)==O(Some 2)`, `T((1,"a"))==T((1,"b"))`, plus positives | All five negatives `false`, positives `true`. Deep structural comparison of `ListValue`/`TupleValue`/nested `TaggedValue` works today |
| V19 | …but a derived ADT with a **record** field fails at check | `type R = R({a:int}) deriving (Eq)`; `R({a:1}) == R({a:1})` | `No instance for Eq[{ a: int }]`. The deriving check demands `Eq` of record fields but not of list fields: inconsistent field checking |
| V20 | The runtime comparator exists and is general: `valuesStructurallyEqual` handles Int/Float/String/Bool/Unit/Tagged/List/Tuple/Record recursively; unknown kinds (closures) → `false` | Read `internal/eval/eval_patterns.go:427–495` | Confirmed. The **only** runtime gap is the entry point: `makeADTEqualityFn` (`:385`) rejects any top-level value that is not a `*TaggedValue`. #960's root cause 1 ("needs a value-structural Eq") is already half-built |
| V21 | **No user-defined `Eq` instances exist** | `parseInstanceDeclaration` (`internal/parser/parser_func.go:534`) is `// TODO … return nil`; zero `instance Eq` in any `.ail` in the repo (`git grep`) | Confirmed. Every Eq is a builtin primitive or a derived structural one, so **structural comparison is exactly the semantics of composed dictionaries**. Child-dict threading (the round-1 "ABI" section) buys nothing today |
| V22 | A second runtime comparator exists in the bytecode VM and **diverges** | Read `internal/bytecode/value.go:259` `Value.Equal` | Two divergences: VM treats `NaN == NaN` as **true** (dedup semantics); VM compares record fields **positionally** (`a[i].Name == b[i].Name`), relying on canonical field order, while the evaluator compares by name |
| V23 | The VM is opt-in and hands dictionary-shaped code back to the evaluator | `cmd/ailang/main_run.go:121` `--bytecode` default `false`; `internal/runner/entrypoint.go:139–150` (EvalOnly functions trap to the evaluator in non-strict mode) | Confirmed. Default `ailang run` uses the evaluator's comparator. The VM divergence is a seam to pin, not the main path |
| V24 | Proposed error code `E_EQ_SYNTH_DEPTH` is unallocated | `git grep -n E_EQ_SYNTH_DEPTH -- internal cmd` | No hits outside this doc: free |
| V25 | **An `Eq` constraint whose type still contains a free type variable is not checked.** The value reaches the runtime structural comparator | `Ok(1)==Ok(1)` → `true`; `Ok(1)==Ok(2)` → `false`; `None==None` → `true`; `Ok([1])==Ok([1])` → `true`; `Err("a")==Err("b")` → `false`. Pin the type (`r: Result[int,string]`; `r == Ok(1)`) and it is **rejected**: `No instance for Eq[Result[int, string]]`. Yet `[] == []` is rejected (`Eq[[α4]]`) | Confirmed, and inconsistent: whether `==` type-checks depends on whether inference happened to leave a variable. This is how the 08-29 matrix recorded "Option works". It is a soundness gap: any value, including a function, that stays behind a residual variable would reach a comparator that returns `false` for closures instead of being rejected |

### What changed in the design (R-decisions)

**R-D1 — Runtime: generalize the entry point; add no new comparators.** V20 and V21 together
mean the evaluator needs no `eq_opt`/`eq_list` helpers and no child-dictionary threading.
Add one marker, `DerivedStructuralEquality`, that the evaluator maps to a function calling
`valuesStructurallyEqual(a, b)` on **any** two values. Container, tuple and record instances
synthesized by the type checker all resolve to it. `makeADTEqualityFn` keeps its ADT-only
contract for existing `DerivedADTEquality` dicts. This deliberately drops the original
"Container composition ABI" section; revisit it only when user-defined instances land (V21),
at which point a custom element `Eq` could differ from structural comparison.

**R-D2 — Type checker: synthesize instead of registering shapes.** `InstanceEnv.Lookup`
synthesizes `Eq[T]` when T is:
- `list[τ]` / `[τ]` (via `AsList`), `Option[τ]`, or `Result[τ, ε]`, when every parameter
  resolves `Eq`. `Result` "works" today only by the V25 leak, so it needs synthesis like the others;
- a tuple, when every component resolves `Eq`;
- a record, **only** when it is the expansion of an alias declared `deriving (Eq)`, and every
  field resolves `Eq`.

Recursion goes through `Lookup` itself, so nested shapes compose. Anonymous records stay a
loud failure, keeping the round-2 scoping decision (no action at a distance).

**R-D3 — Record key canonicalization comes first (#960 root cause 2).** Before any record
lookup, collapse the residual row `{ x: int, y: int, ...{x: int, y: int} }` to its closed
field set with sorted labels. Registration and use-site constraints then normalize to one key.
This also fixes the doubled row in the error text. Gate: the V17 probe must pass *through the
real type checker* before Phase 2 starts. That is precisely the step the reverted commit
skipped.

**R-D4 — The depth cap must be real, and tested by mutation.** `synthesizeEq(typ, depth)`
increments `depth` on every recursive call. Exceeding 8 returns `E_EQ_SYNTH_DEPTH` (V24), with
the message specified above. Acceptance includes a test that fails if the increment is removed.
The 08-29 cap was dead code that no test could have caught.

**R-D5 — Fix the deriving field check (V19).** `deriving (Eq)` on an ADT currently demands
`Eq` for record fields but not list fields. After R-D2 the rule is uniform: a derived ADT is
`Eq` iff every field type resolves `Eq`. List, Option and tuple fields then pass *because they
synthesize*, not because they are skipped. Functions stay non-`Eq`: a field of function type
makes `deriving (Eq)` fail loudly, naming the field.

**R-D6 — Pin the VM seam; don't unify it in this sprint.** V22 is pre-existing and reached
only through `--bytecode`. Two parity tests go in: records with differently ordered literals,
and a NaN field. Each asserts evaluator == VM, or asserts the documented difference. If the
record-order test fails on the VM, canonical record field order becomes a blocking prerequisite
for any **default** bytecode flip. File it; don't fix it here.

**R-D7 — Close the residual-type-variable leak (V25). ⚠ Design-freeze item: in or out of this
sprint?** Once synthesis exists, a residual `Eq[F(…α…)]` constraint must either be discharged
by synthesis (after instantiation) or fail loudly at generalization. It must never silently fall
through to runtime. Recommendation: **in scope**. Without it, the new rules and the old leak coexist,
and "does `==` type-check?" keeps depending on inference accidents, the same class of confusion
that sank the 08-29 verification. Risk: programs that type-check today *only* because of the leak
will start failing. The leak and the gap together make those rare (1 missing-`Eq` row in 877), but
the regression set must include a leak-dependent fixture that stays green because synthesis now
covers it (e.g. `None == None` becomes `Eq[Option[α]]`, which must default or generalize, not die).

### Re-land implementation plan (supersedes the original plan)

**Phase 0 — Evidence harness, before any code (~0.5h).** Check in `examples/eq_containers.ail`
exercising every positive in V12–V19, and `examples/eq_containers_negative/` (one file each:
element lacks Eq, a function field, an anonymous record, depth 9). Record current output:
all positives fail today. This is the probe file whose equality expressions *actually evaluate*;
the 08-29 probe never did. `make verify-examples-toplevel` runs the positive file in CI.

**Phase 1 — Record canonicalization (R-D3) (~1h).** Row collapse plus key normalization.
Gate: V17 passes `ailang check` and `run`, printing `true` and then `false` for unequal records.

**Phase 2 — Synthesis + marker (R-D1, R-D2, R-D4) (~1.5h).** `Lookup` synthesis for list,
Option, Result and tuple, plus derived records; `DerivedStructuralEquality` in the evaluator;
a real depth cap. Gate: every Phase-0 positive prints the right answer, including the negative
controls (unequal values → `false`).

**Phase 3 — Deriving uniformity + seam tests (R-D5, R-D6) (~1h).** Uniform field check;
evaluator/VM parity tests; mutation-test each new test (revert its fix, watch it fail).

### Re-land success criteria

- [x] Every V12–V19 positive compiles **and evaluates to the correct boolean**. Each has an
      unequal-value control that prints `false`, via `examples/eq_containers.ail` (26 checks),
      pinned by `TestEqContainersExample`
- [x] Element-lacks-`Eq` still fails at check. The hint now names the innermost part
      (`Equality on [int -> int] needs == on int -> int`)
- [x] Anonymous record `{x: 1} == {x: 1}` still fails (round-2 scoping)
- [x] R-D7 (in scope, Mark 2026-09-22): a residual-variable `Eq` is decomposed. `Err(f) == Err(f)`
      is rejected (it printed `false` before), and `None == None` / `[] == []` still type-check
- [x] Depth 9 fails with `E_EQ_SYNTH_DEPTH`. `TestEqSynthesisDepthCap` fails when the increment
      is removed (mutation run)
- [x] Evaluator/VM parity tests exist (`cmd/ailang/eq_parity_test.go`). Record field order and nested
      NaN agree. A top-level NaN is asserted as the documented divergence
- [x] The Eq hint covers only genuinely non-`Eq` types. `TestEqInstanceHintIsActionable` was updated deliberately
- [x] The teaching prompt (v0.16.6 head, amended in place) states which types support `==`. Every
      example was checked with `ailang check`/`run`
- [ ] **NOT DONE:** re-grading the banked `mlfq_scheduler_hidden` failures. The banked rows are
      not on this machine (searched the repo, `~/dev`, `~/.ailang` and `/tmp`), so the four
      failures are unaccounted for
- [x] `make test-core` and `verify-examples-toplevel` are green apart from `examples/ai_modes.ail`.
      For `make test`, see the implementation notes

### Implementation notes (2026-09-22)

Where the build departed from the plan, and why:

- **R-D3 needed no canonicalization.** A value of a record alias already carries the alias name
  (`TRecord.TypeName`; probed with a debug `Lookup`). Record `Eq` is therefore "`TypeName` names a
  derived-Eq type", with no shape registry and no action at a distance. The doubled row
  (`{ x: int, ...{x: int} }`) is only how the error text prints the type.
- **V19's cause was the alias, not a field check.** `type R = R({a:int})` registers `R` as an alias
  for its record (M-STREAM-DX/M4), so a use site sees `TRecord{TypeName:"R"}`. The nominal rule
  above fixes it. R-D5's field check was new: before it, no field of a derived type was checked.
  The field check (`pipeline/derived_eq.go`) also removes the duplicated registration block in
  `pipeline_single.go` / `pipeline_module_compile.go`.
- **Groundness is scoped to Eq.** `isGround` stops at lists, tuples and functions, so `Eq[[α]]`
  was "resolved" (and rejected) while `Eq[Option[α]]` was generalized. `constraintIsGround` looks
  through every structure for `Eq` only. Widening `isGround` for every class instead made
  `Num[int -> β]` generalize, so `42(1)` crashed at runtime (`internal/repl`
  TestLoadModuleTypeError caught it). `TestConstraintIsGroundIsScopedToEq` pins the split.
- **Op lowering:** a list `==` that stays polymorphic (`[] == []`, generic `xs == ys`) used to be
  lowered to a nonexistent `eq_List` (ELB_OP001). It now defers to the evaluator's structural `==`.
- **NaN:** `valuesStructurallyEqual` was IEEE while `Eq[Float]` is lawful, so the evaluator
  disagreed with itself (`nan == nan` true, `[nan] == [nan]` false). It is now lawful, which also
  makes nested NaN agree with the VM.
- **Round-1 evaluation finding, fixed (2026-09-23).** `func pick[a](x: a, y: a) { [x] == [y] }`
  let `pick(f, f)` compile and print `false`. ANF lifts `[x]` into a let-bound temporary, and a list
  literal is a syntactic value, so the temporary was generalized over the function's own `a`. The
  name-based withhold (`freeVars(env) \ baseEnvFreeVars`, M-TYPE-LIST-SOUND round 3) cannot
  withhold `a` when the base env already leaks a free `a`, which it does. Each use then got a fresh
  `α`, and the caller's Eq obligation was lost. Before this sprint, the shallow `isGround`
  accidentally rejected the resulting `Eq[[α]]`. Generalization now also withholds every variable
  free in the env frames the declaration pushed (`TypeEnv.FreeTypeVarsAbove`). As a side effect,
  `pick(3, 3)` now type-checks (it was rejected before). As defense in depth, the structural
  comparators return an error when they meet a function value instead of answering `false`
  (`rejectFunctionEquality`). Fixture `examples/eq_containers_negative/generic_list_fn.ail`.
  Mutation: dropping the frame withhold makes the fixture fail (with the runtime guard's error).
- **Mutation-tested:** removing the reduction breaks `residual_fn`, removing the field check
  breaks `fn_field`, reverting the NaN rule breaks the nested-NaN parity test, and removing the
  depth increment breaks the depth-cap test.
- **Dev hazard (unfixed, filed separately):** the module compile cache is keyed by build commit,
  so dirty rebuilds on one commit share entries. `ailang check` printed "No errors" from a stale
  entry during this sprint. Every probe here ran with `AILANG_NO_CACHE=1`. The same hazard is a
  plausible mechanism for the 08-29 attempt's false "verified live" claims.

### Conflict Surface — re-land additions

Beyond the original section: (1) `makeADTEqualityFn`'s ADT-only contract must stay intact.
Existing `DerivedADTEquality` dicts keep their path, and the new marker is a sibling, not a
widening. (2) The bytecode VM's `Value.Equal` (V22) is a second implementation of the same
concept, so parity tests are mandatory (R-D6). (3) `2122705e0`'s per-type Eq hint is exercised
by `TestEqInstanceHintIsActionable`; its list/Option/tuple cases change expectation once
synthesis succeeds, so update the test deliberately rather than deleting it.

### Quorum

Trigger 1 fires: R-D1 (dropping the child-dict ABI), R-D6 (VM seam deferred) and R-D7
(leak in or out of scope) are design-freeze items. Run `ailang design-quorum` before planning. It is cents per doc, but it is
spend, so it needs a human yes (Mark paused eval spend on 2026-09-22).


## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Instance synthesis is a pure function of the instance environment and types |
| A2: Replayability | 0 | No trace changes |
| A3: Effect Legibility | 0 | No effect surface change |
| A4: Explicit Authority | 0 | No capability change |
| A5: Bounded Verification | +1 | `ailang check` fully decides subset/synthesis locally; failure messages remain local |
| A6: Safe Concurrency | 0 | InstanceEnv is single-threaded per check |
| A7: Machines First | +1 | Every test an AI writes asserting spans/records stops needing hand-written match-helpers (#960 measured in dogfooding) |
| A8: Minimal Syntax | +1 | Zero new syntax — `deriving (Eq)` already parses on records; composition is implicit |
| A9: Cost Visibility | 0 | No metered surface |
| A10: Composability | +1 | Container instances compose: Eq[a] ⊢ Eq[Option[a]], Eq[a] ⊢ Eq[[a]] — standard dictionary composition |
| A11: Structured Failure | +1 | Missing instances still fail LOUDLY with the element type named (e.g. "Eq of the missing field type" attribution) |
| A12: System Boundary | 0 | No boundary change |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 / A3 / A4 / A7 — no violations

## Verification Log

All premises probed live 2026-08-28 with the dev binary (`check` + `run`):

| # | Claim | Method | Result |
|---|-------|--------|--------|
| V1 | Monomorphic ADT `deriving (Eq)` equality compiles | probe file with ADT deriving (Eq); `VInt(3) == VInt(3)` in check | Compiles ✓ |
| V2 | Record `deriving (Eq)` compiles but `record == record` FAILS with `No instance for Eq[{name,x,…}]` | probe `p1 == p2` on two identical records | FAILS — derive is tracked (`derivedEqTypes[typeName]`, M-DX19) but no instance is registered for the anonymous-record shape |
| V3 | `Option[int] == Option[int]` fails | probe | `No instance for Eq[Option[int]]` |
| V4 | `list[record] == list[record]` fails | probe | same container gap |
| V5 | `Option[ADT] == Option[ADT]` where the ADT itself derives Eq ALSO fails | probe `Some(VInt(3)) == Some(VInt(3))` | `No instance for Eq[Option[V]]` — container composition is the blocker even when the element derives |
| V6 | Root cause (records): `elaborateTypeDecl` handles `*ast.AlgebraicType` for derived-Eq registration; the `*ast.RecordType` arm registers an alias and returns WITHOUT registering an instance (file_funcs.go:171–232) | Read + probe V2 | Confirmed |
| V7 | Container synthesis machinery exists to build on: `InstanceEnv.Lookup/Add`, `canonicalKey`, `deriveEqFromOrd` (an Ord→Eq derivation precedent), and dict synthesis in `internal/types/dictionaries.go` | Read instances.go 28–135, dictionaries.go | Confirmed |
| V8 | Polymorphic derive is explicitly deferred ("deferred to v0.7+", file_funcs.go:168) | Read | Confirmed — container parameters must not silently re-open that door |
| V9 | Eq dictionary shape: instances carry `Dict{"eq","neq"}` impl names; operator `==` dispatches via the instance env | instances.go Eq Int/Float/String/Bool builtin rows | Confirmed |
| V10 | `DictionaryEntry.Impl` is `interface{}` — it can carry a nested marker/dict object, so the resolved child `Eq[τ]` IMPL can be captured inside a container marker | dictionaries.go:25 `Impl interface{} // The actual implementation` (read this session) | Confirmed — child-dictionary threading mechanism is type-safe as designed |
| V11 | The marker pattern already drives evaluator behavior: `DerivedADTEquality{TypeName}` markers are special-cased in the evaluator (structural TaggedValue comparison) rather than compiled to field-wise dict calls | dictionaries.go:12–19 marker type + comment | Confirmed — containers mirror this with the child dict captured, not a name string |

## Problem Statement

AILANG equality is dictionary-passed: `==` resolves through an `Eq[α]` instance. Instances
exist for primitives, monomorphic ADTs (`deriving (Eq)`, M-DX19), and nothing else. An AI
(or human) test author is stuck the moment the value is a **record**, an **Option** or a
**list** of anything — the exact three shapes real tests assert on (#960: my span test
needed a hand-written `spanIs(opt, want)` match-helper instead of `d.line == Some(6)`).

**Current State** (all verified V1–V6): derive works only for monomorphic ADTs; records
and containers fail and push the author toward match-helpers.

**Impact:** The write→assert loop — the most common AI testing loop — pays ceremony tax on
every span/record assertion. #960 documents the dogfooding instance.

## Goals

**Primary Goal:** `==`/`!=` compile for records (declared with `deriving (Eq)`), `Option[α]`
and `[α]` — by composing instances for the parts that already have them.

**Success Metrics:**
- The V2/V3/V4-style probes all compile at HEAD
- A test needing a hand-written span match-helper becomes a direct `==` assertion
- `ailang check` on a type WITHOUT an instance sub-part still fails, naming the missing
  element instance

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Composition happens at instance LOOKUP (synthesis from Eq[a]), not at derive-declaration time | Records/Options arrive as anonymous structural shapes; deriving is per-declaration | agent | design | med |
| Recursion depth cap on synthesis (8) to keep `ailang check` bounded (A5) | A deep TRecord chain could recurse; the cap keeps costs bounded and visible | human (confirm) | design | low |
| Only `Eq` derives through containers; `Ord` unchanged | Scope: this sprint fixes #960's shapes; Ord composition is a separate doc | human (confirm) | design | low |
| Records: instance keyed on the STRUCTURAL shape (expanded by the alias like M-FIX-RECORD-UPDATE), not the alias name | The probe failed because the alias NAME was tracked, not the shape == needs | agent | design | med |

### Design Freeze

- [ ] Confirm lookup-time synthesis + record-shape registration as the two-part scope
- [ ] Confirm depth cap (8) and the missing-element error attribution shape

## Conflict Surface

Touches `internal/types/` (instances.go, dictionaries.go) and `internal/elaborate/` — MANDATORY per the type-system gate. Positions:

1. `internal/types/instances.go` `Lookup` — synthesis for container TApps (Option/list) and composed record shapes
2. `internal/elaborate/file_funcs.go` `*ast.RecordType` arm — derive-aware registration of the expanded structural shape
3. `internal/types/dictionaries.go` — `DerivedContainerEquality` marker (mirrors `DerivedADTEquality`)

What else lives there: builtin Eq/Ord/Show rows (primitives), `deriveEqFromOrd` derivation, record-alias expansion (M-FIX-RECORD-UPDATE), `==` lowering to dict calls. Must-still-work fixtures: M-DX19 ADT derives (V1), ail_diag's 6 inline tests, email-parse packages/email primitives equality, the polymorphic-derive rejection (V8). Regression tests: the four probe matrix positives + element-lacks-Eq negatives + cap-exceed. Deliberately changes: nothing — previously-failing equalities remain failing only when a constituent genuinely lacks Eq.

## Solution Design

> **Original 2026-08-28 design, kept for provenance.** Where it conflicts with *Re-land revision
> (2026-09-22)* above (notably the container ABI, `eq_opt`/`eq_list` helpers, and shape-keyed record
> registration), the re-land revision wins.

### Overview

Two coordinated changes, both inside the existing typeclass dictionary machinery:

1. **Record derives via the declared ALIAS, module-scoped** (round-2 redesign — gemini's
   action-at-a-distance objection): records do NOT register global structural-shape instances.
   A record type with `deriving (Eq)` resolves `==` through its OWN alias name
   (`derivedEqTypes[aliasName]`, elaborated per module) — an unrelated anonymous
   `{x: int}` record gains nothing and stays a loud failure (A5 local reasoning preserved;
   two independent `type A/B = {x: int} deriving (Eq)` never collide).
   Runtime: a `DerivedRecordEquality` marker carrying per-field CHILD Eq dicts
   (`{fieldName → childEqImpl}`) — field-wise comparison through each field's resolved Eq.
2. **Container composition** (instances.Lookup): when `Eq[T]` is requested for a
   one-parameter container application (Option-shaped, list-shaped), synthesize from an
   `Eq[param]` result — recursively, depth-capped (8), memoized per canonicalKey. A missing
   param instance keeps the current loud failure, with the missing element type named.

The composition dict for containers is a fixed evaluator-level implementation (like M-DX19's
ADT dict): `eq_opt` and `eq_list` runtime helpers — two new dict implementations registered
beside the existing primitive `eq_*` ones. For records the runtime marker
(`DerivedRecordEquality{fields → child Eq impls}`, V10) compares FIELD-WISE through each
field's resolved Eq dictionary — so record equality honors custom element Eq (ADT elements
compose their own derived comparison) without any name-keyed global registry.
Cap-exceed (depth > 8) returns the distinct E_EQ_SYNTH_DEPTH error specified below — never
a silent "No instance"

### Container composition ABI (round-1 quorum premise, now verified)

The mechanism rides the existing M-DX19 marker pattern, which the evaluator already handles:
`DerivedADTEquality{TypeName}` is a **marker** the evaluator special-cases by structural
comparison of TaggedValues (dictionaries.go:12–19). For containers the marker becomes
`DerivedContainerEquality{Container: "option"|"list", Child Eq/LanguageImpl}` — the RESOLVED
count child `Eq[τ]` implementation is captured inside the marker's Dict and invoked by the
evaluator for Some/Some and element pairs (None/None → true; mismatched constructors → false).
This is precisely the child-dictionary threading gpt5-6-sol's round-1 objection demanded:
the child dict is captured in the marker's Impl field (DictionaryEntry.Impl is `interface{}`),
not a name-only reference. Frozen-core placement: Option and [α] are core types, and AILANG
has no user-space parameterized-instance declaration yet (instances are Go-registered, V7);
this marker machinery is the natural foundation for user-space parameterized instances when
they land — superseded by them, not duplicated by them.

### Depth-cap behavior (round-1 quorum gap, now specified)

Exactly the failure mode oc-glm-5-2 flagged: a silent `No instance` naming the type at depth
9 would be a misleading fallback. Instead, cap-exceed returns a DISTINCT error:

```
E_EQ_SYNTH_DEPTH: Eq synthesis depth cap (8) exceeded while resolving Eq[<type>] —
nesting is deeper than the cap supports. Fix: compare a shallower shape, or declare an
explicit Eq instance for the intermediate type.
```

No silent fallthrough to the ordinary missing-instance message; the error is unit-tested
(synthesis at depth 9 with all levels Eq-capable).

### Implementation Plan

**Phase 1: Record-shape derives** (~1.5h)
- [ ] `elaborateTypeDecl` RecordType arm: derive-aware registration of the expanded structural shape
- [ ] Instance lookup consults registered structural shapes (alias-expansion hook shared with M-FIX-RECORD-UPDATE)

**Phase 2: Container composition** (~2h)
- [ ] `eq_opt`/`eq_list` runtime dicts (evaluator, alongside primitive `eq_*`)
- [ ] `Lookup` synthesis: `Eq[Option[τ]]`/`Eq[[τ]]` from `Eq[τ]`, depth-capped
- [ ] Missing-element errors name the element instance

**Phase 3: Tests + fixtures** (~2h)
- [ ] The four Verification-Log probes as regression tests (compiling AND evaluating)
- [ ] Negative fixtures: element-lacks-Eq still fails with the element named
- [ ] sunholo/ail_diag regression: match-helper tests keep passing; a direct `==` variant passes
- [ ] `make test` green; CI drift-check clean

### Files to Modify/Create

**Modified files:**
- `internal/types/instances.go` — synthesis in `Lookup` (+ memo) (~60 LOC)
- `internal/types/dictionaries.go` — composed dict specs for containers (~30 LOC)
- `internal/eval/*` — `eq_opt`/`eq_list` runtime dict helpers (~60 LOC)
- `internal/elaborate/file_funcs.go` — RecordType derive registration (~25 LOC)

**New files:**
- `internal/types/instances_derive_test.go` — composition/depth/negative tests

## Examples

### Example 1: the dogfooding assertion, unblocked

```ailang
-- before (#960, needs a match-helper)
match d.line { Some(ln) => ln == 3, None => false }

-- after: direct
d.line == Some(3)
```

### Example 2: whole-struct assertions

```ailang
match nth(ds, 0) { Some(d) => d == want, None => false }   -- a Diag deriving (Eq)
```

## Success Criteria

- [ ] All four Verification-Log probes move from FAIL(compile) to PASS (acceptance: rerun the probe scripts)
- [ ] Element-lacks-Eq still fails loudly naming the element (acceptance: negative fixture)
- [ ] sunholo/ail_diag suite passes with the new `==`-based test variant added alongside old ones
- [ ] `ailang check` on regression fixture set (ail_diag, an email-parse sample) green
- [ ] All tests passing; docs updated (CHANGELOG)

## Testing Strategy

**Unit tests (Go):** Lookup synthesis depth/negative; record-shape registration; dict shapes.
**E2E (AILANG):** the probe matrix from Verification Log as live fixtures, plus a
cap-exceed fixture (a 9-deep Option chain) asserting the distinct E_EQ_SYNTH_DEPTH error.
**Manual:** `ailang repl` spot-checks for Option/list/record equality.

## Deferred Decisions

- Arbitrary user-defined parameterized containers (Map[κ,v]) — v2 doc
- Other classes (Ord, Show) compose — same machinery later, separate doc
- Deriving for polymorphic ADTs (already deferred in the v0.7+ error text)

## Non-Goals

- **Ord composition** — Eq only this sprint (separate doc if wanted)
- **Structural Eq for functions** — reject as before (element-lacks-Eq)
- **Changing the dictionary-passing ABI** — synthesis composes EXISTING dict shapes only

## Timeline

Single session (~6h), three phases with gates between.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Synthesis masks genuinely-missing instances (silent permissiveness) | Med | Depth cap + element-named failure when a leaf lacks Eq; negative fixtures |
| Record alias name-vs-shape double registration | Med | Key on the alias-expanded structural shape (M-FIX-RECORD-UPDATE hook) |
| Composed dict ABI mismatch with evaluator expectations | Med | Reuse the M-DX19 dictionary synthesis path verbatim; e2e fixtures before bank |

## Follow-up (2026-09-26): R-D5 broke working code

R-D5 made `deriving (Eq)` require `==` on every field. Before it, no field was checked. Two
kinds of field lost their only fix. First, a record alias reached through a container, such as
motoko_ext_abi's `PassThroughObserved(code: string, fields: [DiagnosticField])`. Second, stdlib
ADTs such as `Json`, which user code cannot annotate. Every motoko tree stopped booting on
v0.42+ (v0.41.0 was green).

Resolution, keeping "no action at a distance" at use sites:
- `CheckDerivedEqField` checks a record by its fields, whether written inline or named by an
  alias (resolved through local and imported aliases), and reaches through list, Option,
  Result and tuple. Recursive aliases are assumed Eq on re-entry. Function fields still fail.
  A plain alias still has no `==` of its own.
- Pure-data stdlib ADTs declare `deriving (Eq)`. Types that carry `bytes` are excluded because
  `bytes` has no `==` yet.

Found while running the long `let` chain in `examples/eq_containers.ail`: two added lines that
contain nested record literals took `ailang check` from about 1s to 42s on the *released*
v0.43.1. That is an inference-cost problem unrelated to Eq. It has not been fixed yet.

## Related Documents

<!-- Auto-populated by neural search on "eq derive containers"; duplicate gate passed (max 0.38) -->

**Planned (check for overlap):**
- [design_docs/planned/v0_35_0/m-dx-pi-harness.md](design_docs/planned/v0_35_0/m-dx-pi-harness.md) (0.38) — the dogfooding run that produced this doc

## References

- GitHub issue #960
- M-DX19 — the ADT Eq derive this doc composes
- `internal/types/instances.go` — InstanceEnv / deriveEqFromOrd precedent (V7)
- Probes V1–V9 (all live, 2026-08-28, dev binary) — see Verification Log

## Future Work

- Ord composition (comparable records/containers)
- Polymorphic `deriving` — with this sprint's synthesis as the foundation

---

**Document created**: 2026-08-28
**Last updated**: 2026-09-22 (re-land revision after the `cd1c976fb` revert)