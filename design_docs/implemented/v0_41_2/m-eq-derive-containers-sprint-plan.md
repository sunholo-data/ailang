# M-EQ-DERIVE-CONTAINERS — Sprint Plan (re-land)

**Design doc**: [m-eq-derive-containers.md](m-eq-derive-containers.md), *Re-land revision (2026-09-22)*
**Issues**: #960, #963
**Estimated**: ~5h, one session (design said ~4h; +1h for R-D7, which is in scope)
**Risk**: medium. It touches instance resolution, which every `==` goes through.

## Decisions (Mark, 2026-09-22)

| Item | Ruling |
|------|--------|
| R-D7 residual-variable leak | **In scope** |
| R-D6 VM seam | **Parity tests only.** No unification this sprint |
| Design quorum | **Skipped** (spend paused). The live Phase-0 evidence file is the guard instead |

## Recon findings that change the plan (read before implementing)

1. **Nominal records already carry their alias name.** `TRecord.TypeName` is `"P"` for a value
   of `type P = {…} deriving (Eq)`, `"R"` for the single-constructor ADT `R({a:int})`, and `""`
   for an anonymous literal. This was probed live with a debug `Lookup`. Record Eq therefore
   becomes "if `TypeName` names a derived-Eq type, use its instance". No shape registry is
   needed, anonymous records stay rejected, and no action at a distance is possible.
   R-D3's "canonicalization" reduces to this. The doubled row only appears in error *text*.
2. **V19's cause is not a field check.** No field check exists at all. `type R = R({a:int})`
   registers `R` as an alias for the record, so the use-site type expands to
   `TRecord{TypeName:"R"}`, which finding 1 fixes. R-D5 adds the missing field check
   explicitly.
3. **Runtime plumbing.** `resolveGroundConstraints` stores `TCon{NormalizeTypeName(T)}` and the
   evaluator looks up `prelude::Eq::<name>::eq`. Synthesized instances record the sentinel
   type `StructuralEq`, and a single registry row maps it to the `DerivedStructuralEquality`
   marker, which calls `valuesStructurallyEqual`. The bytecode path lowers every `Eq.eq` to
   `OpEq` → `Value.Equal` (the V22 seam), so parity tests cover it.
4. **The leak (V25)** is at the two top-level `partitionConstraints` sites
   (`typechecker_core.go`). Non-ground constraints are returned or dropped unchecked. Fix:
   *reduce* each non-ground `Eq[T]` whose head is not a bare type variable via the same
   synthesis rules. Ground leaves must resolve, bare-variable leaves stay polymorphic, and
   anything else (arrows, polymorphic user ADTs) fails loudly.
   The runtime for surviving residual nodes stays the shim's structural `==`, which is now
   provably only reached on Eq-safe shapes.

## Milestones

### ✅ M0 — Evidence harness (0.5h, ~120 LOC .ail)
- `examples/eq_containers.ail`: every V12–V19 positive, each with an unequal control, printing
  `label=bool`.
- `examples/eq_containers_negative/`: element lacks Eq (function list), anonymous record, a
  function field under `deriving (Eq)`, depth 9, plus a leak-closing case (`Eq[α -> α]` behind
  a residual).
- Record the pre-change output (all positives fail at check).

### ✅ M1 — Synthesis + marker + depth cap (1.5h, ~200 LOC Go + tests)
- `instances.go`: `Lookup` → `lookupDepth(class, typ, depth)`. `Eq` synthesis for list,
  `Option`, `Result`, tuples and nominal records (`TypeName` → derived instance). Synthesized
  instances are flagged `Structural`. Depth > 8 → `*EqSynthDepthError` (`E_EQ_SYNTH_DEPTH`).
- `resolveGroundConstraints`: `Structural` → sentinel `StructuralEq` type.
- `dictionaries.go`: `DerivedStructuralEquality` marker plus a builtin registry row.
  `eval_patterns.go` handles the marker. `makeADTEqualityFn` stays unchanged.
- Gate: every M0 positive prints the right boolean.
- Tests: synthesis table, element-missing names the element, depth cap (mutation: remove the
  increment and the test fails).

### ✅ M2 — Deriving field check (R-D5) (0.5h, ~80 LOC)
- The elaborator records the field types of each derived type. A pipeline helper (deduplicating
  the two copies in `pipeline_single.go` / `pipeline_module_compile.go`) registers them all,
  then checks every field: anonymous record fields are checked field-wise, the rest go through
  `Lookup`. On failure it errors, naming the type, the field and the missing instance.
- Gate: a function field fails loudly. Every existing example with `deriving (Eq)` still passes.

### ✅ M3 — Close the residual leak (R-D7) (1h, ~80 LOC)
- `reduceEqConstraint` at the top-level partition. `None == None` and `Ok(1) == Ok(2)` still
  work. `Eq[α -> β]` and residual `Eq[Tree[α]]` are rejected.
- Gate: `make test-core`, then `make verify-examples-toplevel` (leak-dependent programs surface
  here).

### ⚠️ M4 — Seams, hints, docs (all but the offline re-grade) (1h)
- VM parity tests: record field order and NaN field (evaluator vs `--bytecode`). They assert
  agreement or the documented difference, and any divergence is filed.
- `eqInstanceHint` and `TestEqInstanceHintIsActionable` are updated deliberately.
- The teaching prompt states which types support `==`, verified with `ailang check`.
- CHANGELOG. Re-grade the banked `mlfq_scheduler_hidden` list-`==` failures offline (no spend).
- The design doc moves to `implemented/` only after the evaluator passes.

## Acceptance
The design doc's *Re-land success criteria*, verbatim.

## Outcome (2026-09-23)

Implemented in `6d096fe98`. Independent evaluation, round 1: **PASS 88/100**
(`.ailang/state/evaluations/eval_M-EQ-DERIVE-CONTAINERS_round_1.json`). Its high-severity
finding, a soundness hole via `[x] == [y]` in generic functions, was fixed before the design doc
moved (see the design doc's implementation notes). The offline `mlfq_scheduler_hidden` re-grade
remains undone: the data isn't on this machine.
