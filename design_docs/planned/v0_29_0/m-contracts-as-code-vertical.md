# M-CONTRACTS-AS-CODE: The Verified Deontic Engine — AILANG's Flagship Vertical Example

**Status**: Planned
**Target**: v0.29.0 (docs + example; no language changes)
**Priority**: P1
**Estimated**: 3–4 days (example modules 1.5d, Z3 contract layer 0.5d, docs page + site wiring 1d, optional AI-extraction demo 1d)
**Dependencies**: None on language features (all verified live, see Verification Log). Resolves the m-fable-strategy-review R6 deferred decision "which orchestration benchmark becomes the flagship docs example — agent proposes, Mark picks": **this is the proposal.**

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | The showcase's whole pitch: same contract + same events = same settlement, replayable; the AI-extraction phase is quarantined behind `!{AI}` |
| A2: Replayability | +1 | Event-sourced design: the ledger is a pure fold over an event list — replay is trivial and demonstrated |
| A3: Effect Legibility | +1 | Pure reasoning core (`pure func` everywhere); IO only at the report boundary; AI extraction visibly `!{AI}` |
| A4: Explicit Authority | +1 | Demo runs with `--caps IO` only; the AI phase requires an explicit extra capability — the docs page shows the denial when it's absent |
| A5: Bounded Verification | +2 | The centerpiece: `requires`/`ensures` on settlement math, Z3-proved (`ailang verify`), shown failing on a seeded bug |
| A6: Safe Concurrency | 0 | No concurrency |
| A7: Machines First | +1 | The example doubles as agent teaching material; the eval benchmark twin (`legal_obligation_engine`) measures machine writability |
| A8: Minimal Syntax | +1 | Zero new syntax; showcases existing ADTs/records/contracts |
| A9: Cost Visibility | 0 | No change (AI phase inherits std/ai budgets) |
| A10: Composability | +1 | Deontic core is a reusable module (`examples/contracts/` package layout), composable with std/json for contract import |
| A11: Structured Failure | +1 | Invalid events/states are typed (`Result`-carrying transitions), not panics |
| A12: System Boundary | +1 | Prose→structured-contract crossing is explicit and validated (AI extraction returns typed data checked by the same engine) |

**Net Score: +11** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

## Problem Statement

The strategy review ([m-fable-strategy-review](../m-fable-strategy-review.md), R6) concluded that
AILANG's under-leveraged moats — Z3 contracts, capability-gated effects, deterministic replay —
are "not in the default benchmark rotation, not the marketing lead," and that
"general-purpose language that happens to be AI-friendly" loses to Python, while "the language
where AI orchestration is type-checked" has no competitor. R6 left the flagship example as a
deferred decision.

Meanwhile the M-EVAL-FRONTIER-TIER wave-5 work produced, as a benchmark reference, a working
**deontic contract engine** in 260 lines of AILANG
([benchmarks/frontier_refs/legal_obligation_engine.ail](../../../benchmarks/frontier_refs/legal_obligation_engine.ail)):
milestones, notice-and-cure windows, waiver forgiveness, force-majeure deadline extension,
amendment, termination cascade, integer-exact settlement. It runs byte-identically to its Python
twin — but the AILANG version can do things the Python twin cannot:

- **Prove the settlement math**: `ensures` clauses on cap/floor arithmetic, Z3-verified in
  milliseconds (verified live: `✓ VERIFIED addCap [31ms]`, Z3 4.15.4).
- **Make illegal states unrepresentable**: obligation statuses are a closed ADT; a missed case
  is a loud error, not a silently wrong settlement.
- **Replay deterministically**: the engine is a pure fold over events.
- **Gate authority**: the engine cannot touch the filesystem or network even if buggy.

Legal-tech is a domain where these properties aren't nice-to-haves — auditability, exact
arithmetic, and "show me why this clause fired" are the product. No mainstream scripting stack
offers proof-carrying settlement math. This is the R6 flagship wearing a costume a non-PL
audience immediately understands.

**Current State:** the engine exists only as an eval-reference artifact; no docs page, no
contracts, no package structure, no AI-extraction front end.

## Goals

**Primary Goal:** Ship a docs-site flagship example — "Contracts as Code: a verified deontic
engine" — that demonstrates every AILANG moat in one coherent, runnable artifact.

**Success Metrics:**
- One `examples/contracts/` package: deontic core + Z3-contracted settlement module + runnable demo, all passing `verify-examples` and `ailang verify`.
- Docs page walks the four moats with runnable snippets (each verified in CI like other examples).
- A seeded-bug variant where `ailang verify` produces a Z3 counterexample (the "show me it failing" moment).
- README/site positioning references it as the lead example (Design Freeze item).
- Optional stretch: `!{AI}` prose-extraction front end (clause text → typed contract data → same verified engine), demonstrating the full orchestration story.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| This becomes THE flagship docs example (closes R6 deferred decision) | Sets site/README positioning | human (Mark) | before docs work | low |
| Include the AI-extraction phase in v1 or defer to follow-up | Doubles scope but completes the orchestration story | human | design | low |
| Example lives in `examples/contracts/` (multi-module package) vs single file | Package layout showcases composability but complicates verify-examples wiring | agent proposes, human ratifies | sprint start | med |

### Design Freeze

- [ ] Mark confirms contracts-as-code as the R6 flagship (vs another orchestration candidate)
- [ ] Mark picks v1 scope: engine+Z3+docs only, or including the AI-extraction demo

## Routing (decided 2026-07-09)

**Extension, not core — and the package now exists.** Per PROGRAM.md's default
bias, the reusable engine shipped as
[`sunholo/deontic` 0.1.0](https://github.com/sunholo-data/ailang-packages/tree/main/packages/deontic)
(ailang-packages, sprint M-DEONTIC-PKG): pure event fold, 6/6 Z3-VERIFIED
settlement functions, ground-truth demo byte-identical to the wave-5
benchmark output. Core/stdlib gets NOTHING from this vertical — that absence
is the flagship's central claim ("v0.28.0 already does all of this").
Individual primitive gaps discovered later (decimal money, date types) route
as normal AILANG-fix lane items with their own docs. The docs-site flagship
(this doc's Solution Design) should now IMPORT the package rather than carry
its own engine copy — one source of truth, and the example doubles as
package documentation.

## Solution Design

### Overview

Four artifacts, in dependency order:

1. **`examples/contracts/deontic.ail`** — the engine, ported from the wave-5 reference with
   naming/comments upgraded from "benchmark ref" to "teaching example" quality: event ADT,
   obligation-state ADT, pure fold, typed transitions.
2. **`examples/contracts/settlement.ail`** — the arithmetic module with `requires`/`ensures`
   on every money function (penalty cap, floor-division interest, netting), each Z3-provable.
   Plus `settlement_buggy.ail` (not exported to verify-examples' run phase) with a seeded
   off-by-one whose `ailang verify` counterexample is shown in the docs.
3. **`examples/contracts/demo.ail`** — the wave-5 timeline as a runnable demo printing the
   settlement report (same 16 lines as the benchmark — already cross-validated).
4. **Docs page** `docs/docs/guides/contracts-as-code.md` — structured as the four moats:
   *unrepresentable states* (ADT exhaustiveness), *proved arithmetic* (verify transcript +
   counterexample), *replayable audit* (fold-over-events + trace), *least authority*
   (run with/without caps). Imports example code via raw-loader per docs rules — never inline.

Optional phase 2 (Design Freeze decision): **`examples/contracts/extract.ail`** using
`std/ai` `callJsonSimple` to turn three paragraphs of contract prose into the typed event/term
data, then feed the SAME verified engine — the "AI does the reading, the type system does the
trusting" demo. Runs only with `--caps IO,AI`; docs show both the run and the capability denial.

### Conflict Surface

Not applicable — no parser/typechecker/codegen/runtime changes. Docs + examples only.
The only integration risk is `verify-examples` treating multi-file example packages; if the
package layout fights the verifier, fall back to three flat top-level example files
(`contracts_deontic.ail`, `contracts_settlement.ail`, `contracts_demo.ail`) — decision granted
to the implementing agent (see Deferred Decisions).

### Files to Modify/Create

| File | Action | Est. LOC |
|---|---|---|
| `examples/contracts/deontic.ail` (or flat) | new — port + polish wave-5 ref | ~200 |
| `examples/contracts/settlement.ail` | new — contracted arithmetic | ~80 |
| `examples/contracts/settlement_buggy.ail` | new — seeded counterexample | ~40 |
| `examples/contracts/demo.ail` | new — runnable timeline | ~60 |
| `examples/contracts/extract.ail` | new (phase 2, optional) | ~80 |
| `docs/docs/guides/contracts-as-code.md` | new — flagship page | ~250 |
| `tools/verify_examples.sh` | touch — include package or flat files | ~10 |
| `README.md` / site landing | touch — positioning (after Design Freeze) | ~10 |

## Examples

The proved-arithmetic moat, exactly as it will appear on the docs page (verified live, v0.28.0):

```ailang
export pure func addCap(pen: int, cap: int) -> int
  requires { pen >= 0 && cap >= 0 }
  ensures { result <= cap }
{
  if pen < cap then pen else cap
}
```

```
$ ailang verify examples/contracts/settlement.ail
  Solver: Z3 version 4.15.4 - 64 bit
  ✓ VERIFIED addCap  [31ms]
```

And the seeded bug (`if pen <= cap then pen else cap` with an off-by-one cap check) fails with
a concrete counterexample — the moment no Python stack can reproduce.

## Success Criteria

- [ ] All example files pass `ailang check`, run under `verify-examples`, and the contracted module passes `ailang verify` in CI
- [ ] Buggy variant produces a Z3 counterexample (goldens captured in docs)
- [ ] Docs page live with raw-loader imports (no inline code drift)
- [ ] Demo output byte-identical to `legal_obligation_engine` expected_stdout (shared provenance)
- [ ] Design Freeze items resolved; README positioning updated if confirmed

## Testing Strategy

- `verify-examples` covers check+run for every new example file (existing CI gate).
- Add a CI step (or extend the examples gate) running `ailang verify` on `settlement.ail`
  asserting VERIFIED, and on `settlement_buggy.ail` asserting a counterexample — this doubles
  as regression coverage for the Z3 pipeline on real-world-shaped contracts.
- Docs snippets come from the example files via raw-loader, so they cannot rot independently.

## Deferred Decisions (agent latitude)

- Package layout (`examples/contracts/` package) vs flat top-level files — whichever
  `verify-examples` accepts cleanly; do not modify the verifier's contract to force the layout.
- Exact seeded bug in `settlement_buggy.ail` — any single-token change with a clean Z3 counterexample.
- Whether the docs page also embeds the eval-benchmark cross-link (`legal_obligation_engine`)
  as a "models write this in one shot" proof point.

## Non-Goals

- **No legal-domain completeness** — this is a teaching vertical, not a contract-management product; three milestone clauses and six rule families are the whole universe.
- **No new language features, no parser/stdlib changes** — the entire point is that v0.28.0 already does all of this.
- **No natural-language legal advice** — the AI phase (if included) extracts structure; the docs must state it is a demo, not legal tooling.
- **No new benchmark** — the eval twin (`legal_obligation_engine`) already shipped in wave 5.

## Timeline

| Day | Work |
|---|---|
| 1 | Port + polish engine, settlement module with contracts, verify green |
| 2 | Buggy variant + counterexample goldens; demo file; verify-examples wiring |
| 3 | Docs page (four-moat structure), raw-loader imports, CI verify step |
| 4 (optional) | AI-extraction phase + capability-denial demo |

## Verification Log (hard gate)

| Claim | Method | Result |
|---|---|---|
| `requires`/`ensures` clauses parse & type-check | `ailang check` on live snippet, v0.28.0-9-dirty | ✓ No errors found |
| Z3 proves `ensures` end-to-end | `ailang verify --relax-modules` on `addCap` | `✓ VERIFIED addCap [31ms]`, Z3 4.15.4 |
| The deontic engine itself is expressible | wave-5 ref runs byte-identical to Python twin | [legal_obligation_engine.ail](../../../benchmarks/frontier_refs/legal_obligation_engine.ail), 16-line expected output |
| Capability gating (no FS/Net without caps) | every wave-1..5 ref run with `--caps IO` only | routine across the campaign |
| `std/ai` typed calls exist (phase-2 claim) | teaching prompt import surface (`std/ai (call, callJson, callJsonSimple)`); NOT re-verified live | cite only — phase 2 must verify before implementation |

## Related Documents

- [m-fable-strategy-review](../m-fable-strategy-review.md) — R6: own the orchestration vertical; this doc closes its flagship-example deferred decision
- [m-eval-frontier-tier](m-eval-frontier-tier.md) — wave 5 produced the engine + the falsifiable language-delta benchmark twin
- [m-smt-cross-module-functions (implemented v0.15.x)](../../implemented/v0_15_x/m-smt-cross-module-functions.md) — Z3 verification of imported calls (neural 0.43): the machinery the settlement module leans on
- [m-contract-guided-eval (implemented v0.8.1)](../../implemented/v0_8_1/m-contract-guided-eval.md) — contract-guided eval harness (neural 0.42): grading-side use of the same contracts

---

**Document created**: 2026-07-09
**Last updated**: 2026-07-09
