# M-EFFECT-SCOPE-PARAMS: Capability-Scoping Parameters on Effects

**Status**: Planned
**Target**: v1.0.0 (sprint-sized carve-out; LAST of the four — candidate for re-scoring to v1.1 at the release gate, see Framing)
**Priority**: P1 — Low within the effect-refinement track
**Estimated**: ~16 hours (~2.5 days)
**Dependencies**:
  - [m-effect-mode-validation](m-effect-mode-validation.md) (sprint 1) — `scope` keys validate through the same schema
  - [M-CAPABILITY-BUDGETS](../../implemented/v0_6_2/m-capability-budgets.md) (v0.6.2, shipped) — the capability surface scopes refine
  - [m-effect-replay-contracts](m-effect-replay-contracts.md) (sprint 2) — orthogonal, but the dispatch mechanism decision constrains how scopes reach grant checks
**Parent**: [M-EFFECT-REFINEMENT](m-effect-refinement.md) — decomposition sprint 4 of 4 (2026-07-11)

## Framing

> The parent doc's **Phase 4**: the `scope=` parameter becomes a typed *narrowing* of a
> capability grant — `Rand[mode=crypto, scope=identity]` demands a capability grant at least as
> narrow as `identity`; `scope=test-denied` marks operations that must fail under test harness
> grants. Today `scope=identity` parses and type-checks with **no semantics whatsoever**
> (live-verified 2026-07-11; after sprint 1 it will be *rejected* until this sprint registers
> the key).
>
> **Release-gate note**: of the four carve-outs this one has the weakest v1.0 forcing function —
> no public doc promises scope semantics (the guide lists it under "Future work"), and the
> stability page holds the whole surface Experimental. Queued per the ratified "effect
> refinement IN" decision, ordered last; Mark may re-score to v1.1 at the release gate without
> breaking any published promise.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No determinism change |
| A2: Replayability | 0 | Orthogonal to replay contracts |
| A3: Effect Legibility | +1 | Authority narrowing visible in the signature |
| A4: Explicit Authority | +1 | **Primary goal** — fine-grained, typed capability grants |
| A5: Bounded Verification | +1 | Scope-vs-grant check is local at the grant site |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Agents read authority requirements from the type |
| A8: Minimal Syntax | 0 | Grammar already exists (Phase 1) |
| A9: Cost Visibility | 0 | No cost change |
| A10: Composability | +1 | Scopes compose with modes in one param map |
| A11: Structured Failure | +1 | Scope violations are typed capability errors with fix hints |
| A12: System Boundary | 0 | No new boundary |

**Net Score: +6** → **Decision: Proceed**

## Problem Statement

Capabilities today are effect-granular: `--caps Rand` grants ALL Rand operations. A program
that mints identity material and also shuffles a list holds one undifferentiated Rand grant —
the security-sensitive operation is indistinguishable from the trivial one at the authority
layer. The parent doc's `scope=` parameter is the typed narrowing mechanism; Phase 1 shipped
its grammar only.

## Goals

**Primary Goal:** `scope=` parameters participate in capability-grant checking: a scoped
operation requires a grant that covers its scope; harness/test configurations can deny scopes
wholesale.

**Success Metrics:**
- Initial scope set registered (sprint-1 schema): `identity | session | test-denied | _`
  (wildcard) on the effects that declare scope support (Rand + AI initially; AI's `byok` scope
  reconciled with the shipped M-AI-EFFECT-MODES surface).
- Grant checking: scoped op under a grant that doesn't cover the scope → typed capability
  error naming both.
- `scope=test-denied` operations fail under the test-harness capability profile (integration
  test).
- Unscoped ops behave exactly as today (wildcard).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Scope semantics: subset lattice (`identity < _`) vs flat named grants | Lattice = expressive but is the subtyping the parent doc avoided for modes; flat = string-match simplicity | planner proposes; **recommendation: flat named grants + `_` wildcard** (matches invariant-params philosophy) | plan | high |
| Grant surface: extend `--caps` syntax (`--caps Rand[identity]`) vs config file only | CLI surface is public + stability-relevant | human-or-planner; conservative default: config first, CLI sugar optional | plan | med |
| Overlap with capability BUDGETS (numeric) | Parent risk table: scopes are typed narrowing, budgets stay numeric — must not merge | this doc: **keep disjoint** (ratifies parent mitigation) | design | med |
| Which effects declare scope support in v1.0 | Every effect = big sweep; Rand+AI = matches shipped reality | this doc: **Rand + AI only**; others opt in later by schema row | design | low |

### Design Freeze

- [x] Flat named scopes + wildcard; no scope lattice in v1.0 (revisit with evidence).
- [x] Rand + AI only; scope support is a per-effect schema opt-in.
- [x] Budgets and scopes stay disjoint mechanisms.
- [ ] Grant-surface shape (config vs CLI) — planner with a Conflict-Surface pass on `--caps`.
- [ ] Initial scope set ratification (`identity|session|test-denied|_`) — carried from parent
  Design Freeze; confirm with Mark at sprint review if convenient, else agent proceeds
  (parent doc grants additive-scope latitude).

## Solution Design

### Overview

1. Schema rows (sprint-1 table): `scope` key + value set for Rand, AI.
2. Capability-grant model gains per-effect scope sets; default grant = wildcard (today's
   behaviour).
3. Grant check at effect-op dispatch: declared scope ∈ granted scopes (or grant is wildcard).
4. Test-harness profile denies `test-denied` by construction.

### Conflict Surface

Touches `internal/types/` (schema), capability checking (grant sites in `internal/effects/`
context / `internal/eval`), CLI if grant sugar lands.

1. **Positions extended**: no syntax; `scope` key becomes legal on Rand/AI (post-sprint-1 it's
   rejected — strictly widening, ordering-safe).
2. **Existing constructs**: `--caps` parsing (bracket syntax would collide with shell globbing
   and any existing caps grammar — planner enumerates); capability-budget config (must remain
   untouched — disjointness is frozen above); `AI[scope=byok]` shipped semantics (M-AI-EFFECT-
   MODES runtime stub per [m-ai-effect-modes-followups](m-ai-effect-modes-followups.md) item 3
   — this sprint must NOT ship conflicting byok semantics; reconcile or exclude byok).
3. **Disambiguation**: n/a (no parse change).
4. **Programs that MUST still work**: every current program (all unscoped → wildcard grants);
   capability-budget tests; M-AI-EFFECT-MODES examples.
5. **Deliberate change**: none for existing programs; new rejections only for newly-written
   scoped code under narrow grants.

## Examples

```ailang
-- Identity-scoped crypto randomness: needs a grant covering scope=identity
export func mint_api_key() -> string ! {Rand[mode=crypto, scope=identity]} = ...

-- Must fail under the test-harness profile
export func irreversible_send() -> () ! {Net, Rand[scope=test-denied]} = ...

-- Unscoped: wildcard, unchanged
export func shuffle[a](xs: [a]) -> [a] ! {Rand} = ...
```

## Success Criteria

- [ ] Scope schema rows (Rand, AI) validated via sprint-1 machinery
- [ ] Grant check: covered / not-covered / wildcard paths tested
- [ ] `test-denied` integration test under harness profile
- [ ] Zero-diff for all unscoped programs (full sweep)
- [ ] byok reconciliation recorded (adopted or explicitly excluded with pointer to followups doc)
- [ ] Guide section + teaching prompt updated
- [ ] `make test && make verify-examples && make lint` green

## Testing Strategy

- **Unit**: schema; grant-set coverage logic; wildcard defaults.
- **Integration**: scoped op end-to-end under narrow/wide/absent grants; harness profile.
- **Golden**: capability error messages.

## Deferred Decisions

- **Scope lattice** — only with evidence of need (parent grants agent latitude on additive
  scopes, not on subtyping).
- **CLI grant sugar** — optional; config-only is a valid v1.0 ship.
- **Further scopes** — additive, agent latitude per parent doc.

## Non-Goals

- No mode work (sprints 1–3).
- No budget-system changes (disjoint by freeze).
- No BYOK implementation (followups doc item 3 owns it).
- No scope support beyond Rand + AI in v1.0.

## Timeline

Day 1: schema + grant model + coverage check. Day 2: dispatch-site enforcement + harness
profile + tests. Half-day 3: docs, prompt, sweep, CI.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Collision with capability budgets | Med | Frozen disjointness; budget tests in CI |
| byok double-implementation vs followups doc | Med | Explicit reconciliation gate in success criteria |
| `--caps` grammar conflicts | Med | Conflict-Surface pass required before any CLI sugar; config-first fallback |
| Scope creep into a general refinement system | Med | Flat named scopes frozen; lattice needs new evidence |

## Related Documents

- [M-EFFECT-REFINEMENT](m-effect-refinement.md) — parent (Phase 4); decomposed 2026-07-11
- [m-effect-mode-validation](m-effect-mode-validation.md) — sprint 1 (schema machinery)
- [M-CAPABILITY-BUDGETS](../../implemented/v0_6_2/m-capability-budgets.md) — the surface refined
- [m-ai-effect-modes-followups](m-ai-effect-modes-followups.md) — owns byok runtime semantics
- [m-agent-orchestration](../v1_1_0/m-agent-orchestration.md) — future consumer (permissions substrate)

## References

- [Design Axioms](/docs/references/axioms)

## Future Work

- Scope lattices / refinement subtyping (evidence-gated)
- Scopes on FS/Net (fixture-tree scoping, host allowlists as scopes)

---

**Document created**: 2026-07-11 (decomposition of M-EFFECT-REFINEMENT, mission iteration 6)
