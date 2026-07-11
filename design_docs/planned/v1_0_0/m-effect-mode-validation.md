# M-EFFECT-MODE-VALIDATION: Enforce the Closed Mode Set for Parameterised Effects

**Status**: Planned
**Target**: v1.0.0 (sprint-sized carve-out; ships on the normal v0.29.x road)
**Priority**: P1 — High within the effect-refinement track (public docs make a FALSE claim today)
**Estimated**: ~8 hours (~1 day)
**Dependencies**:
  - [M-EFFECT-REFINEMENT-PHASE1](../../implemented/v0_15_x/m-effect-refinement-phase1.md) (v0.15.0, shipped) — the `!{E[k=v]}` machinery this validates
  - [M-AI-EFFECT-MODES](../../implemented/v0_15_x/m-ai-effect-modes.md) (v0.15.0, shipped) — AI mode set to register
**Parent**: [M-EFFECT-REFINEMENT](m-effect-refinement.md) — decomposition sprint 1 of 4 (2026-07-11)

## Framing

> Phase 1 (v0.15.0) ratified a **closed mode set** as a design decision, and the public guide
> ([docs/docs/guides/parameterised-effects.md](../../../docs/docs/guides/parameterised-effects.md),
> "Mode set is closed") claims *"the typechecker rejects unknown values"*. **That claim is false
> today.** This sprint makes it true: a per-effect parameter schema (allowed keys + allowed
> values) enforced at effect-row elaboration, with structured errors. It is the prerequisite for
> the Clock/Net/FS ports (adding a mode row must mean the compiler knows the row's legal values)
> and for replay-contract dispatch (a registry keyed by (effect, mode) must not receive
> unvalidated strings).

**Live evidence (2026-07-11, `ailang` v0.28.0-148-g6c25f45e9, both binaries rebuilt):**

```
$ cat /tmp/modal_check2.ail
module tmp/modal_check2
export func f() -> int ! {Clock[mode=banana]} = 42
export func g() -> int ! {Rand[mode=banana]} = 42
$ ailang check /tmp/modal_check2.ail
✓ No errors found!          # ← should be two errors
```

`Rand[mode=crypto, scope=identity]` also passes (`scope` is parsed but has no schema either).
`grep -rn "unknown mode\|validModes\|knownModes" internal/` finds no validation code in
`internal/types/` or `internal/parser/` — the only "invalid mode" error in-tree is an unrelated
manifest check (`internal/manifest/manifest.go:234`).

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No runtime change |
| A2: Replayability | +1 | Guarantees the replay-contract registry (Phase 3) only ever sees registered modes |
| A3: Effect Legibility | +1 | Every mode value in a signature is now guaranteed to mean something |
| A4: Explicit Authority | 0 | No capability change |
| A5: Bounded Verification | +1 | Validation is a local table lookup at elaboration; no global analysis |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | AI agents get a structured error + the legal value list instead of silent acceptance |
| A8: Minimal Syntax | 0 | No syntax change; rejects strings the grammar already accepts |
| A9: Cost Visibility | 0 | No cost changes |
| A10: Composability | +1 | One schema table shared by defaults, validation, and future ports |
| A11: Structured Failure | +1 | Typo'd mode fails loudly at check time with a fix hint, not silently at replay time |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +6** → **Decision: Proceed** (no hard violations; A1/A3/A4/A7 all ≥ 0)

## Problem Statement

Phase 1 shipped the `!{E[k=v]}` grammar, invariant unification, and the default-mode table
(`internal/types/effects.go: defaultEffectModes` — `Rand→mode=os`, `AI→mode=fixed`). The
decision record and the published guide both say the mode set is closed and compiler-enforced.
The enforcement was never built:

1. **Unknown values pass**: `!{Rand[mode=banana]}` type-checks (transcript above).
2. **Unknown keys pass**: any identifier key is accepted by the grammar and never re-checked.
3. **Params on schema-less effects pass**: `!{Clock[mode=pinned]}` type-checks even though
   Clock has no registered modes at all (its port is Phase 5). The annotation silently means
   nothing.
4. **The public guide asserts the opposite** — a documented-behaviour falsehood of exactly the
   class the v1.0 bar's "LIMITATIONS accurate" clause exists to prevent.

Consequences: authors (human or AI) can ship meaningless or typo'd contract annotations that
unify only with their own typo (invariant unification makes `mode=seedd` a *distinct* mode);
the Phase-3 registry would need defensive handling for garbage strings; and the closed-set
rationale (auditable, compiler-gated rollouts) is void.

## Goals

**Primary Goal:** Every effect parameter that reaches a type-checked program is drawn from a
compiler-known schema: registered key, registered value, on an effect that has registered
parameters.

**Success Metrics:**
- `!{Rand[mode=banana]}`, `!{Rand[flavor=hot]}`, `!{Clock[mode=pinned]}` (pre-Phase-5) all
  produce structured errors naming the offending effect/key/value and listing legal options.
- `!{Rand}`, `!{Rand[mode=os|seeded|crypto]}`, `!{AI[mode=fixed|routeable|replay-only]}`,
  `!{AI[scope=byok]}` continue to type-check (the v0.15.0 surface is unchanged).
- `make verify-examples` green pre/post (no in-tree program uses an illegal param).
- The guide's "Mode set is closed" section becomes accurate; the interim truth-up note
  (added 2026-07-11) is removed.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Schema location: extend `defaultEffectModes` into a full per-effect param schema vs a parallel table | One table = one audit point (the guide already points readers at `effects.go`) | agent (propose in plan) | plan | low |
| Params on schema-less effects: hard error vs warn | Hard error is the closed-set semantics; makes Phase-5 ports *unlock* syntax rather than merely default it | this doc: **hard error** | design | med |
| Where validation runs: elaboration (types) vs parser | Elaboration sees the merged row incl. the `stringSliceToEffectRow` bridge; parser only sees literal syntax | this doc: **elaboration** | design | med |
| Error code: new effect-diagnostic code vs reuse | Machine-readable; AI repair loops key on codes | agent (follow existing conventions in `internal/pipeline/validate_effects.go`) | plan | low |

### Design Freeze

- [x] Closed-set enforcement is a **hard error**, not a warning (ratified by Phase 1's decision
  record; this doc implements it).
- [x] Validation at **elaboration**, covering both elaborator-built and bridge-built rows.
- [x] Schema registers **both** keys and values per effect: `Rand: {mode: os|seeded|crypto}`,
  `AI: {mode: fixed|routeable|replay-only, scope: byok}`. (Planner must verify AI's exact
  shipped surface against the M-AI-EFFECT-MODES outcome report before freezing the table.)
- [ ] Exact error code + message wording (planner; "boring errors" style).

## Solution Design

### Overview

Extend the per-effect table in `internal/types/effects.go` from
`map[string]struct{Key, Value string}` (defaults only) to a schema carrying the allowed value
set per key. At effect-row elaboration (the shared path also used by the
`effectiveParamsOf` normalisation), any explicit param is checked: effect has a schema → key
registered → value registered. Failures produce a structured diagnostic:

```
EFF_UNKNOWN_MODE: effect 'Rand' has no mode 'banana'
  Allowed modes: os, seeded, crypto
  (declared at foo.ail:3:34)
Fix: use one of the allowed modes, or drop the parameter for the default (mode=os)
```

For effects with no schema (Clock/Net/FS until their port lands): any explicit param errors
with "effect 'Clock' does not accept parameters yet (tracked in m-effect-clock-net-fs-modes)".

### Conflict Surface

This change touches `internal/types/` (elaboration/validation). It adds **no grammar**; it
narrows acceptance of already-parsed forms.

1. **Positions extended**: none syntactically. Semantically: effect-row elaboration gains a
   rejection path.
2. **Constructs in those positions today**: explicit params in-tree are confined to legal
   `Rand`/`AI` values — `grep -rn "mode=" std/ examples/` shows only `examples/modal_rand.ail`
   plus AI-mode docs/examples (sweep re-run required at plan time).
3. **Disambiguation**: n/a — no parse change.
4. **Programs that MUST still work** (regression fixtures):
   - `examples/modal_rand.ail` (all three Rand modes)
   - the M-AI-EFFECT-MODES worked example (`AI[mode=routeable]`)
   - any bare-effect program (`!{Rand}`, `!{Clock}` — bare forms have no params to validate)
   - iface-cache round-trip: pre-sprint `.ailang_iface` artefacts decode `Params` as empty →
     defaults applied → no explicit params → no validation triggered (same back-compat bridge
     Phase 1 shipped)
5. **Deliberate breakage**: programs using unregistered params (e.g. `Clock[mode=pinned]`
   written speculatively) stop type-checking. The
   [stability page](../../../docs/docs/reference/stability.md) classifies this surface
   **Experimental** — breaking it pre-1.0 is within the promise. In-tree sweep required;
   out-of-tree, the error message names the tracking doc.

## Examples

```ailang
-- OK (unchanged):
export func roll() -> int ! {Rand} = ...                      -- bare, desugars to mode=os
export func sim(seed: int) -> float ! {Rand[mode=seeded]} = ...

-- NEW ERRORS:
export func f() -> int ! {Rand[mode=banana]} = ...   -- EFF_UNKNOWN_MODE + allowed list
export func g() -> int ! {Rand[flavor=hot]} = ...    -- EFF_UNKNOWN_PARAM_KEY
export func h() -> int ! {Clock[mode=pinned]} = ...  -- EFF_PARAMS_NOT_SUPPORTED (until the port)
```

## Success Criteria

- [ ] Schema table registers Rand + AI surfaces exactly as shipped in v0.15.0
- [ ] Unknown value / unknown key / schema-less effect param → 3 distinct structured errors with fix hints
- [ ] All legal v0.15.0 forms type-check unchanged; `make verify-examples` green
- [ ] Iface-cache round-trip test (pre-sprint artefact loads clean)
- [ ] Guide "Mode set is closed" section verified accurate post-sprint; interim truth-up note removed
- [ ] Teaching prompt mentions the error codes (coordinate with `prompt-manager` skill)
- [ ] `make test && make lint` green

## Testing Strategy

- **Unit**: schema lookup; each error shape; legal-form matrix (every registered
  (effect, key, value) triple).
- **Integration**: `ailang check` fixtures for the three error classes (promote to the
  diagnostic-fixture CI harness from M-DIAG-FIXTURE-PROMOTION); modal_rand + AI examples
  unchanged.
- **Golden**: error message text (boring-errors style).

## Deferred Decisions

- **Error code naming** — agent picks, following existing effect-diagnostic conventions.
- **Whether `scope=byok` on AI validates as key+value or key-only** — agent verifies the
  M-AI-EFFECT-MODES shipped surface and matches it.
- **Migration tooling** (`ailang fix` for typo'd modes) — not required; the error hint suffices.

## Non-Goals

- No new modes, no new effects with params ([m-effect-clock-net-fs-modes](m-effect-clock-net-fs-modes.md)).
- No runtime behaviour change ([m-effect-replay-contracts](m-effect-replay-contracts.md)).
- No user-definable modes (parent doc: v1.0+ research question).

## Timeline

Single day: schema + validation (~4h), errors + fixtures (~2h), docs truth-up + prompt (~1h),
sweep + CI (~1h).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Out-of-tree code (motoko fork, packages) uses speculative params | Med | Surface is stability-page Experimental; error names the tracking doc; release-notes entry |
| Bridge path (`stringSliceToEffectRow`) bypasses validation | Med | Validate in the shared elaboration/normalisation path + a test through the bridge |
| Schema drifts from Phase-5 ports later | Low | Ports edit the SAME table; one audit point by construction |

## Related Documents

- [M-EFFECT-REFINEMENT](m-effect-refinement.md) — parent; decomposed 2026-07-11
- [M-EFFECT-REFINEMENT-PHASE1](../../implemented/v0_15_x/m-effect-refinement-phase1.md) — shipped machinery + the closed-set decision record
- [M-AI-EFFECT-MODES](../../implemented/v0_15_x/m-ai-effect-modes.md) — AI surface to register
- [m-effect-replay-contracts](m-effect-replay-contracts.md) — consumes validated modes (sprint 2)
- [m-effect-clock-net-fs-modes](m-effect-clock-net-fs-modes.md) — unlocked by this (sprint 3)

## References

- [Design Axioms](/docs/references/axioms)
- [Parameterised Effects guide](../../../docs/docs/guides/parameterised-effects.md) — the doc whose claim this sprint makes true

## Future Work

- User-extensible mode sets (parent doc, v1.0+ research question)
- `budget=`/`timeout=` parameter families (parent doc, v1.1+)

---

**Document created**: 2026-07-11 (decomposition of M-EFFECT-REFINEMENT, mission iteration 6)
