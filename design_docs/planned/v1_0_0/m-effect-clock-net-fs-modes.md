# M-EFFECT-CLOCK-NET-FS-MODES: Port Clock, Net, FS to Parameterised Modes

**Status**: Planned
**Target**: v1.0.0 (sprint-sized carve-out; ships on the normal v0.29.x road)
**Priority**: P1 — Medium (completes the mode taxonomy across the core replay-relevant effects)
**Estimated**: ~20 hours (~3 days)
**Dependencies**:
  - [m-effect-mode-validation](m-effect-mode-validation.md) (sprint 1) — schema table this sprint adds rows to
  - [m-effect-replay-contracts](m-effect-replay-contracts.md) (sprint 2) — registry this sprint populates; dispatch pattern this sprint reuses
**Parent**: [M-EFFECT-REFINEMENT](m-effect-refinement.md) — decomposition sprint 3 of 4 (2026-07-11)

## Framing

> The parent doc's **Phase 5**, minus AI (already ported by
> [M-AI-EFFECT-MODES](../../implemented/v0_15_x/m-ai-effect-modes.md), v0.15.0). Clock, Net and
> FS gain mode schemas, default-mode rows, replay-contract registry rows, and runtime wiring to
> the **switches that already exist** — this sprint is largely surfacing existing runtime
> behaviour at the type level, not building new runtime capability. Also absorbs the parent's
> Phase-8 remainder for these effects: `examples/modal_clock.ail`, `examples/modal_net.ail`,
> and the migration guide section.

**Verified current state (2026-07-11, v0.28.0-148-g6c25f45e9):**
- `defaultEffectModes` in `internal/types/effects.go` has Rand + AI only; Clock/Net/FS are an
  explicit "Future:" comment naming this port.
- **Clock**: `internal/effects/clock.go` ALREADY implements two behaviours — wall clock vs
  virtual time under `AILANG_SEED` ("Production mode" / "Deterministic mode" per its comments).
  The mode exists at runtime with no type-level marker.
- **FS**: sandbox machinery exists (`internal/effects/fs.go`, `fs_sandbox_debug_test.go`) —
  again runtime-only, invisible in types.
- **Net**: live dispatch in `internal/effects/net.go` (+ `net_security.go`). Recorded/replay
  is trace-level, not a Net-handler switch — the planner must verify what "recorded" can mean
  TODAY and scope Net's `recorded` mode to what is real (possibly registry-label-only in this
  sprint, like AI).
- Stdlib annotations: `std/clock.ail` (2 `!{Clock}` funcs), `std/net.ail` (4 `!{Net}`),
  `std/fs.ail` (18 `!{FS}`) — all bare; they STAY bare (bare = default mode by construction).

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Pinned clock / fixture FS become declarable, checkable properties |
| A2: Replayability | +1 | Registry rows complete the taxonomy for the core replay-relevant effects |
| A3: Effect Legibility | +1 | AILANG_SEED's clock pinning and FS sandboxing stop being invisible runtime flags |
| A4: Explicit Authority | 0 | Capability grants unchanged (scope= is sprint 4) |
| A5: Bounded Verification | +1 | Same local schema/table checks as Rand |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Agents can require `Clock[mode=pinned]` for deterministic tests from the type |
| A8: Minimal Syntax | 0 | No new grammar; new legal values for existing grammar |
| A9: Cost Visibility | 0 | No cost change |
| A10: Composability | +1 | Three more effects on the one mechanism; validates "add a row, not a subsystem" |
| A11: Structured Failure | +1 | Mode/runtime mismatches (e.g. pinned without AILANG_SEED) are typed errors |
| A12: System Boundary | +1 | Live-vs-recorded / real-vs-fixture boundary crossing explicit in types |

**Net Score: +8** → **Decision: Proceed**

## Problem Statement

Clock pinning (`AILANG_SEED`), FS sandboxing, and (partially) Net recording exist as runtime
behaviours with zero type-level visibility. A function that only works under pinned time, or a
test that must not touch real disk, cannot say so; reviewers and agents must know the env-var
folklore. This is exactly the "collapsed contracts" table from the parent doc, for the three
effects still uncorrected after the Rand pilot and AI port.

## Goals

**Primary Goal:** Clock/Net/FS each gain a validated mode schema, a default row, registry
contract rows, and wiring from declared mode to their existing runtime switches — proving the
parent's "porting an effect = adding rows, not building machinery" claim.

**Success Metrics:**
- Schemas: `Clock: {mode: wall|pinned}` (default `wall`), `FS: {mode: real|fixture}` (default
  `real`), `Net: {mode: live|recorded}` (default `live`) — value sets confirmed at plan time
  against runtime reality.
- Bare `!{Clock}`/`!{Net}`/`!{FS}` programs: zero behaviour or typecheck change
  (`make verify-examples` green; 22+ bare stdlib annotations untouched).
- `Clock[mode=pinned]` + no `AILANG_SEED` → typed error (no silent wall-clock fallback).
- `FS[mode=fixture]` routes through the existing sandbox; escape attempts fail loudly.
- Registry rows for all six (effect, mode) pairs with contract labels per the parent taxonomy.
- `examples/modal_clock.ail` + `examples/modal_net.ail` runnable; migration-guide section
  published.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Net `recorded` mode: real dispatch switch vs registry-label-only (like AI rows in sprint 2) | Net record/replay engine may not exist yet; overclaiming ships a lie, underclaiming wastes the port | planner verifies runtime reality, scopes accordingly | plan | med |
| `pinned`-without-seed: error at first clock op vs at startup | Startup check is friendlier but needs whole-program mode knowledge; op-site check matches Rand's seeded-error precedent | this doc: **op-site error** (consistent with sprint 2) | design | low |
| Default rows change bare-form desugar for 3 widely-used effects | Phase-1's `effectiveParamsOf` normalisation makes bare ≡ explicit-default by construction — but this is the first time defaults are ADDED to effects with heavy existing usage | validate via full example sweep + iface round-trip (mechanism proven at Phase 1: 332/332 byte-identical) | execute | high |
| FS fixture root configuration (which dir is the fixture tree) | Couples to existing sandbox config; must not invent a second sandbox mechanism | planner: reuse existing sandbox config surface as-is | plan | med |

### Design Freeze

- [x] Mode value sets as in the parent taxonomy table (wall/pinned, live/recorded, real/fixture)
  — subject only to the planner's Net reality check.
- [x] Bare forms keep today's behaviour as the default mode for every effect (wall, live, real).
- [x] No new runtime subsystems: wire to existing switches; where no switch exists (Net
  recorded?), ship the registry row + typed not-yet-supported error, not a stub behaviour.
- [ ] Exact error codes/wording (planner).

## Solution Design

### Overview

Per effect: (1) schema row in the validation table (sprint 1's table); (2) default row in
`defaultEffectModes`; (3) contract rows in the sprint-2 registry — Clock: pinned=deterministic,
wall=re-sampleable; Net: recorded=deterministic, live=re-sampleable; FS: fixture=deterministic,
real=re-sampleable (parent taxonomy); (4) dispatch wiring using the mechanism sprint 2 chose,
targeting the existing runtime switches (`clock.go` virtual-time path, `fs.go` sandbox path);
(5) worked examples + migration guide.

### Conflict Surface

Touches `internal/types/` (two tables), `internal/effects/clock.go`/`fs.go`/`net.go` (dispatch),
possibly `internal/elaborate` (if sprint 2 chose lowering).

1. **Positions extended**: no syntax. Three effects move from "params rejected" (post-sprint-1)
   to "these params legal".
2. **Existing constructs**: `AILANG_SEED` is ALSO Rand's seed and the eval-harness determinism
   key — Clock's pinned mode must not change AILANG_SEED semantics, only surface them. FS
   sandbox flags already exist; mode must alias, not fork, that config.
3. **Disambiguation**: n/a.
4. **Programs that MUST still work** (fixtures): all bare-effect stdlib modules
   (`std/clock.ail`, `std/net.ail`, `std/fs.ail` — byte-identical iface); the eval harness's
   seeded runs; `examples/` full sweep; existing FS-sandbox tests.
5. **Deliberate change**: explicit `Clock[mode=pinned]` etc. go from sprint-1 hard error
   ("does not accept parameters yet") to legal — strictly widening post-sprint-1, so ordering
   sprints 1 → 3 has no whiplash for in-tree code (nothing in-tree uses these params).

## Examples

```ailang
-- Deterministic test helper: type says it needs pinned time
export func deterministic_now() -> int ! {Clock[mode=pinned]} = now()

-- Wall clock, explicit (same as bare !{Clock})
export func timestamp() -> int ! {Clock[mode=wall]} = now()

-- Fixture-only file access for tests
export func load_fixture(p: string) -> string ! {FS[mode=fixture]} = readFile(p)
```

## Success Criteria

- [ ] Six (effect, mode) schema entries + three default rows + six registry rows
- [ ] Bare-form zero-diff: full example sweep + iface round-trip green
- [ ] `Clock[mode=pinned]` without seed → typed error; with seed → virtual time (integration test)
- [ ] `FS[mode=fixture]` → sandbox path (reuses existing tests as fixtures)
- [ ] Net scoped honestly per planner's reality check (dispatch or label-only, recorded in doc)
- [ ] `examples/modal_clock.ail`, `examples/modal_net.ail` runnable; migration guide section live
- [ ] Guide's default-mode table updated (drops "(none yet)" rows); teaching prompt updated
- [ ] `make test && make verify-examples && make lint` green

## Testing Strategy

- **Unit**: schema/default/registry rows; per-effect mode dispatch.
- **Integration**: pinned-clock determinism under `AILANG_SEED`; fixture FS containment;
  bare-form behavioural identity (golden).
- **Golden**: stdlib iface byte-identity pre/post; error messages.

## Deferred Decisions

- **Net recorded-mode dispatch depth** — planner scopes to runtime reality (label-only is a
  valid outcome; the row still completes the taxonomy).
- **Whether `sleep` is meaningful under pinned mode** (virtual-time advance vs no-op) — agent
  decides matching existing virtual-clock semantics in `clock.go`.
- **Error wording** — boring-errors style.

## Non-Goals

- AI modes (shipped v0.15.0) and AI followups ([m-ai-effect-modes-followups](m-ai-effect-modes-followups.md)).
- Scope parameters (sprint 4) — `FS[scope=...]` capability narrowing is NOT this sprint.
- Process effect modes ([m-process-modes](../v1_1_0/m-process-modes.md), v1.1).
- Building a Net record/replay engine (if absent, the mode is label-only until that engine exists).

## Timeline

Day 1: schemas + defaults + registry rows + bare-form sweep. Day 2: Clock + FS dispatch wiring
+ tests. Day 3: Net reality-check + scoped wiring, examples, migration guide, prompt, CI.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Adding defaults to 3 heavy-use effects perturbs unification somewhere unswept | High | Phase-1 mechanism (`effectiveParamsOf`) proven; full sweep + iface round-trip are hard gates |
| AILANG_SEED coupling: pinning clock via mode surprises Rand/eval-harness | Med | Mode only *reads* the existing switch; never sets/unsets the env var |
| Net `recorded` overclaims | Med | Planner reality-check is a plan-stage gate; label-only fallback is pre-authorized |
| FS mode forks the sandbox config | Med | Design freeze: alias existing config, no second mechanism |

## Related Documents

- [M-EFFECT-REFINEMENT](m-effect-refinement.md) — parent (Phase 5 minus AI + Phase 8 remainder)
- [m-effect-mode-validation](m-effect-mode-validation.md) — sprint 1 (schema table)
- [m-effect-replay-contracts](m-effect-replay-contracts.md) — sprint 2 (registry + dispatch pattern)
- [M-AI-EFFECT-MODES](../../implemented/v0_15_x/m-ai-effect-modes.md) — the AI port precedent
- [M-R6 clock/net effects](../../implemented/v0_3/M-R6_clock_net_effects.md) — original Clock/Net implementation (v0.3)
- [m-process-modes](../v1_1_0/m-process-modes.md) — v1.1 sibling following this pattern

## References

- [Design Axioms](/docs/references/axioms)
- Parent doc replay-contract taxonomy table

## Future Work

- Net record/replay engine (if scoped out here) as its own design doc
- Process modes (v1.1)

---

**Document created**: 2026-07-11 (decomposition of M-EFFECT-REFINEMENT, mission iteration 6)
