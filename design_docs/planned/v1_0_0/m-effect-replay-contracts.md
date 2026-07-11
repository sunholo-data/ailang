# M-EFFECT-REPLAY-CONTRACTS: Replay Contract Registry + Mode-Aware Runtime Dispatch (Rand pilot)

**Status**: Planned
**Target**: v1.0.0 (sprint-sized carve-out; ships on the normal v0.29.x road)
**Priority**: P1 — Medium (turns mode annotations from documentation into behaviour)
**Estimated**: ~20 hours (~3 days)
**Dependencies**:
  - [m-effect-mode-validation](m-effect-mode-validation.md) (sprint 1) — registry keys must be validated modes
  - [M-EFFECT-REFINEMENT-PHASE1](../../implemented/v0_15_x/m-effect-refinement-phase1.md) (v0.15.0, shipped) — the type-level machinery
**Parent**: [M-EFFECT-REFINEMENT](m-effect-refinement.md) — decomposition sprint 2 of 4 (2026-07-11)

## Framing

> Phase 1 shipped mode syntax with **no runtime meaning**: the
> [parameterised-effects guide](../../../docs/docs/guides/parameterised-effects.md) is explicit
> that `!{Rand[mode=seeded]}` and `!{Rand[mode=os]}` "both dispatch to the same `_rand_int` /
> `_rand_float` / `_rand_bool` builtins". This sprint is the parent doc's **Phase 3**: a replay
> contract registry mapping each validated (effect, mode) pair to a contract label
> ({deterministic, re-sampleable, opaque}), and mode-aware dispatch for the **Rand pilot** so the
> three Rand modes actually differ at runtime. Clock/Net/FS rows register their contracts in
> their own port sprint (sprint 3); this sprint builds the registry they populate.

**Verified current state (2026-07-11, v0.28.0-148-g6c25f45e9):**
- `internal/replay/` does not exist; no contract registry anywhere in-tree.
- `internal/builtins/rand.go`: ONE global `math/rand` source, crypto-seeded at init,
  re-seedable via `_rand_seed`/`SetRandSeed`. All modes hit it identically.
- Runtime effect handlers (`internal/effects/`) have no access to the declared effect params —
  `grep -rn "Params" internal/effects/*.go` (non-test) is empty. Threading the declared mode
  from the elaborated signature to the dispatch site is the substantive work of this sprint.
- **M-CRYPTORAND never landed as specced**: `grep -rn "CryptoRand\|crypto_random"` across
  `internal/`, `std/`, `examples/` is empty; its doc sits in `implemented/v0_15_0/` with
  `Status: Planned` (swept there by the bulk relocation 645467e13, not by implementation).
  `Rand[mode=crypto]` runtime dispatch in this sprint **supersedes** M-CRYPTORAND's runtime
  intent (CSPRNG draws for security-sensitive code) via the mechanism the parent doc always
  intended.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | `mode=seeded` becomes actually deterministic at runtime, not aspirational |
| A2: Replayability | +1 | **Primary goal** — replay harnesses dispatch on contract label, not effect token |
| A3: Effect Legibility | +1 | The signature's mode now states what the runtime does |
| A4: Explicit Authority | 0 | No capability change (scope= is sprint 4) |
| A5: Bounded Verification | +1 | Contract lookup is per-(effect,mode), locally checkable |
| A6: Safe Concurrency | 0 | Seeded source is per-context, guarded like the existing global |
| A7: Machines First | +1 | Agents can choose deterministic-vs-live behaviour from the type alone |
| A8: Minimal Syntax | 0 | No syntax change |
| A9: Cost Visibility | +1 | crypto vs PRNG cost difference is visible in the mode |
| A10: Composability | +1 | Registry is the single extension point for sprint-3 ports + M-PROCESS-MODES (v1.1) |
| A11: Structured Failure | +1 | Entropy failure / missing seed are typed, loud errors |
| A12: System Boundary | +1 | OS-entropy crossing is explicit (`mode=os`/`crypto` vs `seeded`) |

**Net Score: +9** → **Decision: Proceed**

## Problem Statement

Mode annotations are type-checked documentation. Today:

1. `!{Rand[mode=seeded]}` does NOT give deterministic runtime behaviour — all draws come from
   the one global source, seeded from crypto/rand at process start. An author (or AI agent)
   reading the signature is entitled to expect determinism and gets none.
2. `!{Rand[mode=crypto]}` does NOT give CSPRNG draws — security-sensitive callers get
   `math/rand`. The init comment in `builtins/rand.go` even says Rand is "used for
   security-sensitive code (API keys, session tokens)" — that usage deserves crypto/rand,
   which was M-CRYPTORAND's whole point, and it never shipped.
3. Replay tooling has no contract taxonomy: nothing distinguishes "replay must pin this value"
   from "replay may redraw" from "replay must substitute from harness".

## Goals

**Primary Goal:** A populated replay-contract registry and mode-aware Rand dispatch, so the
three Rand modes have three observable behaviours and trace tooling can read the contract.

**Success Metrics:**
- `internal/replay/contracts.go` registry: (effect, mode) → {deterministic, re-sampleable,
  opaque}; populated for Rand (seeded/os/crypto) and AI (fixed/routeable/replay-only — labels
  per the parent doc's Example 4 table; AI *dispatch* already exists via M-AI-EFFECT-MODES).
- `Rand[mode=seeded]` + `AILANG_SEED` (or `rand_seed`) → identical sequences across runs
  (integration test).
- `Rand[mode=crypto]` draws from `crypto/rand`; entropy failure panics loudly (existing
  no-silent-fallback stance preserved).
- `Rand[mode=os]` behaviour unchanged (current global source) — bare `!{Rand}` programs are
  byte-identical in behaviour.
- Trace events for moded effect ops carry the contract label; `ailang trace` surfaces it.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| How the declared mode reaches the dispatch site: (a) effect-context metadata threaded from the elaborated signature; (b) elaboration lowers moded ops to distinct builtins (`_rand_int` vs `_rand_int_seeded` vs `_rand_int_crypto`); (c) scoped handler config | Defines the pattern every later port (Clock/Net/FS/Process) follows; wrong choice = re-plumb per effect | human-or-planner with rationale; **recommendation: (b) lowering** — it reuses the existing builtin registry, keeps the runtime ignorant of types, and is the smallest diff; (a) is more general but touches eval | plan | high |
| Per-mode sources: separate PRNG state for seeded vs os | Sharing one source makes seeded sequences perturbable by unrelated os-mode draws — breaks determinism | this doc: **separate sources** | design | med |
| Contract labels for AI modes: registry entries only vs also wiring AI replay dispatch | AI replay (`mode=replay-only` enforcement) is already scoped in [m-ai-effect-modes-followups](m-ai-effect-modes-followups.md) item 2 — do NOT duplicate | this doc: **registry entries only**; enforcement stays in the followups doc | design | low |
| Does `mode=seeded` without any seed error or fall back? | Silent fallback to a random seed would make "deterministic" a lie | this doc: **typed error** (`AILANG_SEED` unset AND no `rand_seed` call → loud failure at first draw) | design | med |

### Design Freeze

- [x] Contract taxonomy: {deterministic, re-sampleable, opaque} — ratified by the parent doc;
  closed for v1.0.
- [x] Rand pilot only for dispatch; registry rows for AI (labels), Clock/Net/FS rows land in
  sprint 3.
- [x] Separate randomness sources per mode; seeded-without-seed is a typed error.
- [ ] Mode→dispatch mechanism (lowering vs context metadata) — planner decides with a spike,
  records rationale.

## Solution Design

### Overview

1. **Registry** (`internal/replay/contracts.go`, new, ~150 LOC): static table + lookup API
   `ContractFor(effect, mode) (Contract, bool)`. Consumed by trace emission now; by replay
   harnesses later.
2. **Mode-aware Rand dispatch**: three behaviours —
   - `seeded`: dedicated `math/rand` source initialised from `AILANG_SEED` or the program's
     `rand_seed` call; no seed → typed error.
   - `os` (default): existing global source, unchanged.
   - `crypto`: direct `crypto/rand` draws (uniform, unbiased int range); failure panics
     (matches existing `cryptoSeed` stance).
3. **Trace integration**: effect events for moded ops carry `contract` (and `mode`) fields;
   schema change is additive.

### Conflict Surface

Touches `internal/types`/`internal/elaborate` (if lowering) or `internal/eval`/`internal/effects`
(if context metadata), plus `internal/builtins/rand.go`, `internal/trace/schema.go`.

1. **Positions extended**: no syntax. Elaboration of effect ops in moded contexts (if lowering:
   builtin selection becomes mode-dependent).
2. **Existing constructs in those positions**: `rand_seed` explicitly reseeds the global source
   — its interaction with per-mode sources MUST be specified: `rand_seed` seeds the *seeded*
   source; os/crypto sources are never author-seedable (that's their contract).
   `std/game` and any stdlib user of `_rand_*` (sweep at plan time) stay on bare `!{Rand}` →
   os mode → unchanged.
3. **Disambiguation**: n/a.
4. **Programs that MUST still work**: every bare-`!{Rand}` program byte-identical behaviour
   (os path untouched); `examples/modal_rand.ail` (runs today with identical behaviour across
   modes — post-sprint it demonstrates *different* behaviour: update its comments);
   `AILANG_SEED`-pinned eval-harness runs (Clock virtual time already keys on it — do not
   change Clock here).
5. **Deliberate change**: `mode=seeded` without a seed goes from "silently non-deterministic"
   to typed error — Experimental surface, intended.

## Examples

```ailang
-- Deterministic simulation: same AILANG_SEED → same output (NEW: actually true)
export func monte_carlo(n: int) -> float ! {Rand[mode=seeded]} = ...

-- Security token: CSPRNG draws (NEW: actually crypto/rand)
export func mint_api_key() -> string ! {Rand[mode=crypto]} = ...

-- General-purpose: unchanged global source
export func shuffle[a](xs: [a]) -> [a] ! {Rand} = ...   -- mode=os via default
```

## Success Criteria

- [ ] Registry exists, populated for Rand (3 rows) + AI (3 label rows); lookup API tested
- [ ] Seeded determinism integration test: two runs, same seed, identical sequences
- [ ] Crypto mode draws from crypto/rand (test via statistical smoke + source inspection hook)
- [ ] Seeded-without-seed → typed error with fix hint
- [ ] Bare-Rand behaviour byte-identical (golden test on a seeded harness run of os-mode program)
- [ ] Trace events carry contract label; additive schema verified against existing trace readers
- [ ] `examples/modal_rand.ail` comments updated (no longer "runtime treats all modes identically")
- [ ] Guide + teaching prompt updated; `m-cryptorand.md` header corrected to Superseded (points here)
- [ ] `make test && make verify-examples && make lint` green

## Testing Strategy

- **Unit**: registry lookups; per-mode source isolation (seeded sequence unperturbed by
  interleaved os draws); crypto range uniformity smoke.
- **Integration**: end-to-end determinism (`AILANG_SEED=42 ailang run` twice, diff empty);
  seeded-error path; trace label presence.
- **Golden**: os-mode behavioural identity pre/post.

## Deferred Decisions

- **Replay-harness dispatch implementation** (pin/redraw/substitute engines) — this sprint
  ships the registry + labels; harness consumption is incremental follow-on (trace tooling).
- **`uuid4` mode assignment** — currently draws from the global source; planner decides whether
  it stays os-only or gains crypto (likely crypto — it's identity material).
- **Error wording** — boring-errors style; agent resolves.

## Non-Goals

- Clock/Net/FS dispatch (sprint 3 registers their rows).
- `mode=replay-only` AI enforcement ([m-ai-effect-modes-followups](m-ai-effect-modes-followups.md) item 2 — not duplicated here).
- Scope parameters (sprint 4).
- M-ENTROPY envelopes (ships with M-ENTROPY itself; see parent doc routing note).

## Timeline

Day 1: registry + mode→dispatch mechanism spike + decision record. Day 2: Rand three-mode
dispatch + tests. Day 3: trace integration, docs/prompt, bookkeeping (m-cryptorand supersession),
sweep + CI.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Lowering approach leaks mode into iface/caches → stale-cache surprises | Med | Additive builtin names only; iface round-trip test (Phase-1 precedent) |
| Seeded source shared state across concurrent evaluators | Med | Per-context source (follow DEBUG_CONCURRENCY per-request evaluator pattern); mutex like existing `randMu` |
| Behavioural change to bare Rand sneaks in | High | Golden identity test is a hard gate |
| Trace schema change breaks existing readers | Med | Additive fields only; comparator test in `internal/trace` |

## Related Documents

- [M-EFFECT-REFINEMENT](m-effect-refinement.md) — parent (Phase 3); decomposed 2026-07-11
- [m-effect-mode-validation](m-effect-mode-validation.md) — sprint 1, prerequisite
- [m-effect-clock-net-fs-modes](m-effect-clock-net-fs-modes.md) — sprint 3, populates registry rows
- [m-cryptorand](../../implemented/v0_15_0/m-cryptorand.md) — runtime intent superseded by this sprint (doc header to be corrected here)
- [m-ai-effect-modes-followups](m-ai-effect-modes-followups.md) — owns AI replay enforcement; boundary kept explicit
- [m-process-modes](../v1_1_0/m-process-modes.md) — v1.1 consumer of the registry

## References

- [Design Axioms](/docs/references/axioms)
- Parent doc's replay-contract taxonomy table (the normative (effect, mode) → contract map)

## Future Work

- Replay-harness engines dispatching on contract (pin/redraw/substitute) — incremental after registry
- Process/AI-tool-loop contract rows (v1.1)

---

**Document created**: 2026-07-11 (decomposition of M-EFFECT-REFINEMENT, mission iteration 6)
