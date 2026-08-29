# M-EQ-DERIVE-CONTAINERS — make `==` work for records, Options and lists when the parts already have Eq

**Status**: Planned
**Target**: v0.35.0
**Priority**: P1
**Estimated**: ~6 hours (single sprint, type-system surface)
**Dependencies**: None (builds on M-DX19's existing `deriving (Eq)` machinery)
**Tracking**: GitHub issue #960 (filed from the M-DX-PI-HARNESS dogfooding run)

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

## Solution Design

### Overview

Two coordinated changes, both inside the existing typeclass dictionary machinery:

1. **Record-shape instances** (elaborate): on a `*ast.RecordType` decl with `deriving (Eq)`,
   register a synthesized Eq keyed by the EXPANDED structural shape (field names + types),
   matching what unification produces — not the alias name.
2. **Container composition** (instances.Lookup): when `Eq[T]` is requested for a
   one-parameter container application (Option-shaped, list-shaped), synthesize from an
   `Eq[param]` result — recursively, depth-capped (8), memoized per canonicalKey. A missing
   param instance keeps the current loud failure, with the missing element type named.

The composition dict for containers is a fixed evaluator-level implementation (like M-DX19's
ADT dict): `eq_opt` and `eq_list` runtime helpers — two new dict implementations registered
beside the existing primitive `eq_*` ones.

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
**E2E (AILANG):** the probe matrix from Verification Log as live fixtures.
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
**Last updated**: 2026-08-28