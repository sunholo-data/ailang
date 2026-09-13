# M-IFC-AUTHORITY-SCOPING — Narrowing IFC Authority Primitives (Scoped Declassify + Call-Site Positive-Label Enforcement)

**Status**: Planned
**Target**: v0.39.0
**Priority**: P0 (High — security-sensitive semantics change to `internal/types`)
**Estimated**: 3 days
**Dependencies**: M-TAINT-TYPES label lattice (shipped); M-SECRET-EFFECT IFC walk (shipped); closure-laundering fix (1af9f5f30, shipped)

> **Semantics change to `internal/types`.** Per AGENTS.md, `internal/types` is a
> security- and semantics-sensitive package. This doc changes what a *correct*
> program is (previously-clean programs will now be rejected), touches the effect
> param schema and effect-row subtyping, and closes two enforcement holes in the
> IFC pass. It requires the full gate: design approval → sprint plan → execute.

## Problem Statement

Source: GitHub issue #752 — adopting IFC in a real package (`email-parse`, 9 AILANG
scripts with real MCP + CLI surface) surfaced two semantics gaps, both verified
against v0.38.5 (`ailang check` transcripts in the Verification Log).

**Gap 1 — `! {Declassify}` is whole-body authority that propagates to callers.**

The capability grants blanket relabel authority over the *entire function body*,
and the effect system spreads it up the call graph:

- A caller of a `! {Declassify}` function **must itself declare `! {Declassify}`**
  (effect checking rejects `caller` with "Missing effects: Declassify").
- `buildIFCSig` (`internal/types/ifc_check.go`) then sets `sig.declassify = true`
  for the caller, and `checkFunc` **skips Check B entirely** while
  `calleeResultLabel` relabels the caller's result to its declared return label.
- End-to-end (verified): a caller that calls `reveal(s)` can relabel an
  **unrelated** `getSecret()` result as `<clean>` with no error — blanket
  authority over everything in its body, inherited transitively. The declassifier
  itself can likewise return an unrelated secret as `<clean>`.

The workaround — extracting every relabel into a single-purpose `! {Declassify}`
wrapper and keeping all other logic out of any function that touches it — shapes
real code awkwardly and still leaves the wrapper itself with whole-body authority.

**Gap 2 — positive parameter labels are not enforced at call sites.**

`x: string<email>` on a parameter is enforced **only** through the callee's own
return refinement (Check B): `parse(m: string<email>) -> string<sanitized> ! {}`
is rejected for hiding `<email>`. At the call site, nothing checks the argument:

- An unlabelled string passed to a `string<email>` parameter: no error.
- A `<secret>`-labelled string passed to a `string<email>` parameter: no error
  (positive labels forbid nothing at the call site).

So a positive label cannot express "this parameter is a sink whose inputs must
carry / be compatible with label ℓ" — it is documentation plus return-refinement
input only. The dual ({not ℓ}) *is* enforced at call sites (Check A), making the
asymmetry a trap for AI-generated code.

## Goals

**Primary Goal:** Make IFC authority exact — a declassifier may relabel only the
labels it is authorized for, and a positive parameter label constrains argument
flow at the call site.

**Success Metrics:**
- A function declaring `! {Declassify[label=email]}` cannot relabel a `<secret>`
  value (compile error naming the unauthorized label); bare `! {Declassify}`
  keeps today's behavior.
- A caller of a scoped declassifier inherits only the callee's label scope, not
  blanket authority.
- Passing a value whose label is not covered by a positive parameter label
  (`<secret>` arg to `string<email>` param) is a compile error; unlabelled and
  matching-label arguments still pass.
- All existing IFC fixtures and examples compile unchanged (no forced migration).

## Proposed Solution

### 1. Label-scoped Declassify: `! {Declassify[label=…]}`

Ride the **existing** parameterised-effect syntax (`Rand[mode=crypto]`); no new
parser surface. `! {Declassify[label=email]]` — one `label` key, comma-separated
label list; bare `! {Declassify}` means **all labels** (⊤, today's semantics).

**Why label-scoped, not argument-scoped:** taint composes via `LabelJoin`, so
argument positions do not stay separable — in `f(s, other)` with body `s ++ other`
the result label mixes both, and a "declassify only `s`" authority has no sound
meaning. Labels are the lattice's unit of authority; scoping to them composes.

Changes:

- **`internal/types/effects.go`**: add a `Declassify` row to `effectSchema`. This
  is a deliberate, documented **exception to the closed-value-set rule**: the
  `label` key accepts any user-defined label name (labels are an open vocabulary,
  unlike Rand/AI modes). Validation still rejects other keys (e.g.
  `Declassify[mode=x]`), fail-loud, no fallback.
- **`internal/types/ifc_check.go`**: `ifcSig.declassify bool` becomes
  `authorizedLabels` (a set; bare form = ⊤). `checkFunc` **always runs Check B**;
  a leak is permitted iff every leaked constituent label is in the authorized set.
  `calleeResultLabel` is unchanged (a declassifier's result is its declared return
  label) — the scoped Check B is now what guarantees no unauthorized relabel.
- **Propagation rule** (the core change): effect-row subtyping still requires
  `Declassify` presence, but per-effect params for `Declassify` switch from exact
  invariance to a **covering rule** in `SubsumeEffectRows` / `effectParamsCompatible`:
  the caller's label set must be a superset of the callee's; bare covers any set;
  `label=email` does **not** cover `label=secret`. The relation is
  security-monotone: wider authority always subsumes narrower, and the exception
  is confined to `Declassify` (Rand/AI mode invariance is untouched). A caller
  therefore inherits exactly the label scope it must propagate — never more.

**Error shape**: the existing `DeclassifyRequiredError` message is extended to
name the authorized set ("hiding <secret>; this function authorizes only
Declassify[label=email]"). Suggestion: widen the declared return label or add the
label to the authorized set.

### 2. Check C: positive parameter labels enforced at call sites

New rule, dual to Check A, evaluated in `labelOfCall` for local callees (the
imported-callee half is the companion doc's, see Related):

- **(C)** A parameter declared `x: T<ℓ>` requires that the argument's label be
  **covered by** ℓ: every constituent label of the argument must satisfy
  `LabelSubsumes(LabelConst(ℓ), part)`. ⊥ passes (ordinary data, literals),
  `<ℓ>` passes, and any constituent ℓ does not subsume is a violation.

This closes the `<secret>`-to-`string<email>` hole while keeping plain data
passing freely. The callee-side seed is unchanged (params carry their declared
label — a sound over-approximation, since Check C proves actual ⊑ declared).
Result-label computation at the call site is unchanged (transparent join rule).

**Error shape**: new `TypeErrorKind` `ParamLabelCoverError` (kind string, not a
numbered code — same convention as `SinkRefinementError`), message mirrors Check A:
"value labelled <secret> reaches parameter \"m\" declared string<email>, which
does not cover it". Suggestion: relabel explicitly via a scoped declassifier, or
widen the parameter's declared label.

### Migration for existing `! {Declassify}` users

- **Bare `! {Declassify}` is unchanged** — `gated_secret.ail`, `leak_attempt.ail`,
  `secret_demo.ail`, and every IFC test fixture keep compiling byte-identically.
  No forced migration; the scoped form is opt-in narrowing.
- The blanket form remains a documented hazard (whole-body, propagating
  authority). A future lint suggesting narrowing is noted in Future Work, not
  gated here.
- New rejections come only from Check C on cross-label argument flows to
  positively-labelled parameters — that is the fix, not a regression: those flows
  were silently unchecked (the exact #752 report). The `email-parse` package is
  the intended first beneficiary.
- `sink_check.go`'s `CheckDeclassify` (whole-row bool, still zero callers) is
  superseded by the scoped rule; fold it into the scoped check or delete it in
  the same sprint — implementer's choice, note in the sprint report.

### Relation to the closure-laundering fix (1af9f5f30)

That fix made closures over-approximate (a closure carries its body's label) and
made `labelOfCall`'s fallback join the callee's own label. This design preserves
that conservative direction and **cannot reopen the hole**: relabelling anything
— including a closure value whose body carries `<secret>` — now requires the
leaked label to be in the authorized set. A scoped declassifier
(`Declassify[label=email]`) cannot launder a `<secret>`-returning closure or an
unrelated `secret()` result. Verified shapes in the Verification Log (V3, V4).

### Relation to M-IFC-CROSS-MODULE-LABELS (companion doc)

That doc adds an IFC summary to module ifaces so callers can enforce imported
callees' refinements. **Module-boundary propagation itself is out of scope here.**
Two coordination points so the docs do not conflict:

- The companion's summary field `declassify: bool` must be typed as a **label
  list** from day one (`declassify: ["email"]`; `[]` = none; bare = `["*"]`),
  and its per-param summary already carries `label`, which is exactly what
  Check C needs cross-module.
- If both land in v0.39.0, land this doc first (intra-module semantics), then
  the companion serializes the already-final shapes.

## Conflict Surface

**Syntactic positions touched**: effect-row atoms with param lists — a position
already occupied by `Rand[mode=…]` / `AI[mode=…, scope=…]`.

- **What else lives there**: every bare effect (`IO`, `FS`, `Net`, `Declassify`,
  …), row variables (`! {e}`), and budgets (`@limit`). The parser already accepts
  `Effect[params]` generally; rejection happens later in `validateEffectParams`.
- **Disambiguation**: none needed — no new tokens or grammar. The only new
  surface is a schema row plus an open-value-set exception for the `label` key.
- **The real conflicts are semantic, not syntactic**:
  1. `effectSchema` is a frozen closed-value-set table; `Declassify.label` values
     are user-defined labels. The exception must be surgical (key-level
     open set for `Declassify` only), or every closed-set invariant test
     (`TestEffectSchemaDefaultsConsistent` and friends) breaks.
  2. `effectParamsCompatible` / `SubsumeEffectRows` treat per-effect params as
     invariant. Declassify needs covering (⊇), Rand/AI must stay exact. The
     exception must not leak into `DiffEffectRows`' `ParamMismatches` for other
     effects.
  3. The IFC walk's `buildIFCSig` currently reads only `eff.Name`; it must read
     `eff.Params` for Declassify atoms, including duplicate-label atoms and
     `label=` with an empty/whitespace value (parse error, fail-loud).
- **Programs that MUST still work** (regression fixtures, all referenced by
  `TestSecretExamples_IFC` and the ifc tests):
  - `examples/runnable/secrets/gated_secret.ail` (0 violations),
  - `examples/runnable/secrets/leak_attempt.ail` (1 SinkRefinementError),
  - `examples/runnable/secrets/secret_demo.ail` (0 violations),
  - every fixture in `internal/types/ifc_check_test.go` and
    `ifc_closure_test.go` (blanket Declassify behavior unchanged).
- **What deliberately changes**: (a) cross-label arguments to positively-labelled
  params now error (claim V5 shape); (b) scoped declassifiers relabelling
  unauthorized labels now error; (c) `Declassify[label=…]` stops being rejected
  with `EFF_PARAMS_NOT_SUPPORTED`.

## Implementation Plan

**Phase 1: Scoped capability surface** (~1 day)
- [ ] Add `Declassify: {label: <open set>}` to `effectSchema` with the documented
      key-level open-value exception in `validateEffectParams` (+ tests).
- [ ] Covering rule for Declassify params in `SubsumeEffectRows` /
      `effectParamsCompatible` (+ tests: `label=email` ⊄ `label=secret`;
      bare ⊇ everything).

**Phase 2: IFC scoping + Check C** (~1.5 days)
- [ ] `ifcSig.authorizedLabels`; Check B always runs with scoped authorization;
      extended `DeclassifyRequiredError` message.
- [ ] Check C in `labelOfCall` + `ParamLabelCoverError` kind in `errors.go`.
- [ ] Tests: all Verification Log shapes become fixtures (V2–V6 and their
      clean-pass counterparts).

**Phase 3: Docs + examples** (~0.5 day)
- [ ] Update `examples/runnable/secrets/` README and `ailang prompt` only if the
      IFC section asserts the old blanket semantics (verify with
      `ailang prompt` before editing — prompt changes have their own gate).

**Files to modify**: `internal/types/effects.go` (~60 LOC),
`internal/types/ifc_check.go` (~80 LOC), `internal/types/errors.go` (~10 LOC),
tests in `internal/types/*_test.go` (~150 LOC). No new files.

## Success Criteria

- [ ] `! {Declassify[label=email]}` parses, type-checks, and authorizes only
      email relabels; relabelling `<secret>` under it is a `DeclassifyRequiredError`
      naming the authorized set.
- [ ] A caller of a scoped declassifier declaring `! {Declassify[label=email]}`
      cannot relabel `<secret>` in its own body.
- [ ] `<secret>` argument to a `string<email>` parameter: `ParamLabelCoverError`;
      unlabelled and `<email>` arguments still pass.
- [ ] All three `examples/runnable/secrets/` fixtures and all existing IFC tests
      pass unchanged.
- [ ] `make test`, `make lint`, `make check-boundaries` green.

## Non-Goals

- **Cross-module label propagation** — the companion doc
  (`m-ifc-cross-module-labels.md`) owns the iface summary and imported-callee
  enforcement. This doc only fixes the field shapes so they compose.
- Label-polymorphic inference (M-TAINT-TYPES Phase 3) and Z3-contract enforcement.
- Deprecating or linting the bare `! {Declassify}` form.
- Runtime IFC (tracing labels, #1132 second half).

## Risks & Trade-offs

| Risk | Impact | Mitigation |
|------|--------|------------|
| Open value set weakens the closed-schema invariant | Med | Key-level exception for `Declassify.label` only; tests assert other effects still reject unknown keys/values |
| Covering rule leaks into Rand/AI invariance | Med | Confine by effect name; dedicated subsumption tests for Rand/AI exact-match |
| Check C rejects previously-clean packages | Med | Intentional (the #752 fix); error message names both labels and the scoped-declassify remedy |
| Over-approximation rejects safe scoped programs (e.g. join labels crossing scopes) | Low | Same conservative direction as 1af9f5f30; widen the authorized set explicitly |

## Axiom Compliance

- A1 (Determinism): no runtime change; static analysis only. 0
- A3 (Explicit effects): scope is declared in the effect row, not inferred. +1
- A4 (Type safety): closes two unchecked flows; new compile-time errors. +1
- A7 (Machines first): error messages name labels and remedy; asymmetry trap removed. +1

Net: **+3**; no hard violations.

## Verification Log

All claims verified with live `ailang check` on v0.38.5 (temp-dir MOD010
auto-relaxation omitted from transcripts):

| # | Claim | Evidence |
|---|-------|----------|
| V1 | Caller of `! {Declassify}` must declare it too | `claim1_caller_effect.ail`: "Missing effects: Declassify" |
| V2 | Caller then relabels an unrelated `<secret>` as `<clean>` with no error (inherited blanket authority) | `claim1d_inherit.ail`: "✓ No errors found!" |
| V3 | Declassifier itself relabels an unrelated secret as `<clean>` (whole-body authority) | `claim1c_wholebody.ail`: "✓ No errors found!" |
| V4 | Closure-laundering fix is live (control: direct sink leak rejected; IFC runs in `ailang check`) | `control_ifc_runs.ail`: SinkRefinementError; `internal/pipeline/pipeline_module_compile.go:230-236` |
| V5 | `<secret>` arg to `string<email>` param: no error | `claim2b_cross_label.ail`: "✓ No errors found!" |
| V6 | Positive param label only feeds callee-side Check B (return refinement) | `claim2d_return_refinement.ail`: DeclassifyRequiredError on `-> string<sanitized>`; `claim2c_what_it_does.ail` (caller-side flow) clean |
| V7 | `Declassify[label=email]` is rejected today (`EFF_PARAMS_NOT_SUPPORTED`) | `claim3_scoped_declassify.ail` |
| V8 | Mechanism claims: `buildIFCSig`/`checkFunc`/`calleeResultLabel` blanket behavior; effect params invariant | Read from `internal/types/ifc_check.go`, `internal/types/effects.go` (`SubsumeEffectRows`, `effectParamsCompatible`), `effectSchema` comment ("mode set is CLOSED") |
| V9 | Fixtures exist | `ls examples/runnable/secrets/` → gated_secret.ail, leak_attempt.ail, secret_demo.ail |

## Related Documents

**Planned (companion):**
- [m-ifc-cross-module-labels.md](m-ifc-cross-module-labels.md) — iface IFC summary
  and imported-callee enforcement; owns the module-boundary half. Coordinate the
  `declassify` field shape (label list, not bool).

**Implemented (context):**
- M-SECRET-EFFECT / M-TAINT-TYPES (label lattice, IFC walk, closure-laundering
  fix 1af9f5f30) — the shipped base this doc narrows.

## References

- GitHub issue #752 (sunholo-data/ailang) — email-parse IFC adoption report.
- [Design Axioms](/docs/references/axioms)

## Future Work

- Lint suggesting narrowing bare `! {Declassify}` to a scoped form.
- Companion doc's cross-module enforcement consumes the scoped shapes.
- Runtime label-aware tracing (#1132 second half).

---

**Document created**: 2026-09-13
**Last updated**: 2026-09-13
