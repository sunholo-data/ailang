# M-EFFECT-REPLAY-CONTRACTS: Replay Contract Registry + Mode-Aware Runtime Dispatch (Rand pilot)

**Status**: **LANDED (PARTIAL) 2026-07-24, iter-99** — M0–M5 shipped (evaluator sonnet PASS 86/100 r1, generator≠judge vs opus executor). **Delivered + green**: `internal/replay/contracts.go` registry (Rand 3 + AI 3 label rows, cross-table drift-guard invariant), mode-aware Rand dispatch machinery (os/seeded/crypto via `EffContext` threading), seeded-source seed via `AILANG_SEED` + typed no-seed error, crypto via `crypto/rand` (rejection-sampled), additive trace `Mode`/`Contract` fields, bare-Rand byte-identical golden gate, `modal_rand.ail` migrated (now type-checks + runs). **BLOCKED follow-up** → new gating item **M-EFFECT-REPLAY-SUBSUMPTION**: seeded/crypto modes are unit-proven at the Go level but NOT reachable from a runnable `.ail` program — a `!{Rand[mode=seeded]}` function cannot call the bare-`!{Rand}` (os) `std/rand` wrappers because `SubsumeEffectRows` treats effect modes as INVARIANT (`internal/types/effects.go:625`, guarded by `TestSubsumeEffectRows_InvariantOnParams` which Phase-5 routeable→fixed depends on). This is a SHARED gate for the whole effect-mode-dispatch line — the parent doc's `Clock[mode=pinned]` examples have the identical structure, so sprints 3/4 (clock/net/fs) hit it too. Needs a Mark/type-system decision: does an explicit declared mode subsume an os/bare required effect? (The executor correctly declined to overturn a tested invariant out of ratified scope.)

<details><summary>Original pre-sprint status (READY FOR SPRINT-PLANNER)</summary>

Controller fold applied 2026-07-24, iter-99 — Mark's option-(b) decision folded into the normative body; baseline re-pinned to `v0.30.0-154-gb326c3fd3`; caller sweep executed at design time; open design question RESOLVED; no third text round per Mark.
</details>

**DECIDED by Mark 2026-07-24 — option (b), explicit seeding surfaces**: `rand_seed` keeps its existing os-source contract UNTOUCHED (bare-Rand determinism preserved, the round-1 fix stands); `mode=seeded` is seeded ONLY via its dedicated explicit path (`AILANG_SEED` / the seeded-source API) — never implicitly by mode-aware magic (explicit-over-implicit is the language's axiom). CONSEQUENCE: `examples/modal_rand.ail:44` — `deterministic_roll() ! {Rand[mode=seeded]}` seeding via `rand_seed(42)` — is the ONE real DOC/EXAMPLE BUG (fix it to the dedicated path in-sprint), plus a teaching diagnostic if seeded-mode is entered with no seed. **CORRECTION (planner, iter-99):** `examples/expected_fail/effect_budgets_multi.ail:62` is NOT a bug — its `rand_seed(42)` is inside `main() ! {IO, Rand, Clock}` = **bare/os mode** (no `mode=seeded` in the file); under the settled contract it is CORRECT and must stay byte-identical → it becomes the bare-Rand+`rand_seed` GOLDEN FIXTURE, not a migration. Route to sprint-planner (the round-2 premise nits — baseline re-pin + caller sweep — resolved by the controller fold below); no third text round.
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

**Verified current state (re-pinned 2026-07-24, `v0.30.0-154-gb326c3fd3`):**
- `internal/replay/` does not exist; no contract registry anywhere in-tree (controller `ls` at HEAD).
- `internal/builtins/rand.go`: ONE global `math/rand` source (`randSource`, `rand.go:28`),
  crypto-seeded at init (`cryptoSeed`, `rand.go:41`), re-seedable via `_rand_seed`/`SetRandSeed`
  (`rand.go:51`). All modes hit it identically.
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
- `Rand[mode=seeded]` + `AILANG_SEED` (or a dedicated seeded-mode seed path chosen at plan
  time — NOT `rand_seed`, which keeps its existing os-source contract) → identical sequences
  across runs (integration test).
- `Rand[mode=crypto]` draws from `crypto/rand`; entropy failure panics loudly (existing
  no-silent-fallback stance preserved).
- `Rand[mode=os]` behaviour unchanged (current global source, **including** `rand_seed`
  continuing to reseed it exactly as today) — bare `!{Rand}` programs, with or without
  `rand_seed` calls, are byte-identical in behaviour.
- Trace events for moded effect ops carry the contract label; `ailang trace` surfaces it.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| How the declared mode reaches the dispatch site: (a) effect-context metadata threaded from the elaborated signature; (b) elaboration lowers moded ops to distinct builtins (`_rand_int` vs `_rand_int_seeded` vs `_rand_int_crypto`); (c) scoped handler config | Defines the pattern every later port (Clock/Net/FS/Process) follows; wrong choice = re-plumb per effect | human-or-planner with rationale; **recommendation: (b) lowering** — it reuses the existing builtin registry, keeps the runtime ignorant of types, and is the smallest diff; (a) is more general but touches eval | plan | high |
| Per-mode sources: separate PRNG state for seeded vs os | Sharing one source makes seeded sequences perturbable by unrelated os-mode draws — breaks determinism | this doc: **separate sources** | design | med |
| Contract labels for AI modes: registry entries only vs also wiring AI replay dispatch | AI replay (`mode=replay-only` enforcement) is already scoped in [m-ai-effect-modes-followups](m-ai-effect-modes-followups.md) item 2 — do NOT duplicate | this doc: **registry entries only**; enforcement stays in the followups doc | design | low |
| Does `mode=seeded` without any seed error or fall back? | Silent fallback to a random seed would make "deterministic" a lie | this doc: **typed error** (`AILANG_SEED` unset AND no dedicated seeded-mode seed provided → loud failure at first draw; `rand_seed` does NOT count — it seeds the os source, its existing contract) | design | med |

### Design Freeze

- [x] Contract taxonomy: {deterministic, re-sampleable, opaque} — ratified by the parent doc;
  closed for v1.0.
- [x] Rand pilot only for dispatch; registry rows for AI (labels), Clock/Net/FS rows land in
  sprint 3.
- [x] Separate randomness sources per mode; seeded-without-seed is a typed error.
- [x] `rand_seed`/`_rand_seed` keeps its existing contract: it reseeds the os/global source
  exactly as today (bare-`!{Rand}` + `rand_seed` determinism is preserved). The seeded-mode
  source has its own seed path (`AILANG_SEED` or a dedicated mechanism); it is NOT wired to
  `rand_seed`.
- [x] Mode→dispatch mechanism: **(a) context-threading** (NOT the doc's recommended (b) lowering).
  Spike finding (M0, confirmed in-tree): `_rand_int`/`_rand_float`/`_rand_bool` are referenced ONLY
  inside `std/rand`'s wrappers, whose rows are bare `!{Rand}` (= os) — so a lowering pass keyed on the
  effect row at the builtin reference site would ALWAYS see `os`; the outer `seeded`/`crypto` mode
  never reaches the builtin. (b) would require per-caller-mode inlining of the stdlib wrappers
  (monomorphization-scale), NOT the smallest diff. Chosen: thread the resolved `Rand` mode onto
  `EffContext` at moded-lambda entry (`EffectModeFor(row,"Rand")` extracted at closure creation like
  `EffectBudgets`, pushed only when non-`os` so the innermost EXPLICIT mode wins and bare-`!{Rand}`
  stays byte-identical); `builtins/rand.go` reads the mode off the context and dispatches. This retires
  the "lowering leaks mode into iface/caches" risk. Registry-duplication finding (spike 2): NO
  duplication — `effectSchema` (types) is *validation* (which modes are legal), `internal/replay` is
  *taxonomy* ((effect,mode)→contract label); guarded by `TestReplayContractsAreLegalModes` +
  exported `types.IsLegalEffectMode` so `effectSchema` stays the single source of legal modes.

## Solution Design

### Overview

1. **Registry** (`internal/replay/contracts.go`, new, ~150 LOC): static table + lookup API
   `ContractFor(effect, mode) (Contract, bool)`. Consumed by trace emission now; by replay
   harnesses later.
2. **Mode-aware Rand dispatch**: three behaviours —
   - `seeded`: dedicated `math/rand` source initialised from `AILANG_SEED` (or a dedicated
     seeded-mode seed path chosen at plan time); it does NOT hijack `rand_seed`, which keeps
     its existing os-source semantics; no seed → typed error.
   - `os` (default): existing global source, unchanged — including `rand_seed`/`_rand_seed`
     continuing to reseed it exactly as today.
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
   — its interaction with per-mode sources MUST be specified: `rand_seed` CONTINUES to seed
   the *os/global* source exactly as today. Bare-`!{Rand}` programs that call `rand_seed` for
   deterministic testing get determinism today and keep it. The *seeded* source is separate,
   seeded from `AILANG_SEED` (or a dedicated seed path) — not from `rand_seed`. The *crypto*
   source is never seedable (that's its contract). Precise wording on os seedability: os mode
   IS author-seedable via `rand_seed` (its existing contract), but os sequences remain
   perturbable by interleaved draws from other os-mode callers — only `seeded` mode gives an
   isolated, unperturbable sequence.
   `std/game` and any stdlib user of `_rand_*` (sweep at plan time) stay on bare `!{Rand}` →
   os mode → unchanged.
3. **Disambiguation**: n/a.
4. **Programs that MUST still work**: every bare-`!{Rand}` program byte-identical behaviour
   (os path untouched, **including** programs that call `rand_seed` — their determinism-via-
   reseeding is preserved because `rand_seed` still targets the os/global source);
   `examples/modal_rand.ail` (runs today with identical behaviour across
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

- [x] Registry exists, populated for Rand (3 rows) + AI (3 label rows); lookup API tested
  (`internal/replay/contracts.go` + `TestContractFor_*`)
- [x] Seeded determinism unit test: same seed → identical sequences; different seed → different
  (`TestSeeded_Deterministic`, `TestRandIntImpl_SeededMode_Deterministic`). **NOTE:** proven at the
  Go/builtin level, not via an end-to-end `.ail` run — a seeded-mode `.ail` function cannot yet call
  the os-mode `std/rand` wrappers (invariant subsumption blocker, see Design-Freeze note below).
- [x] Crypto mode draws from crypto/rand (`TestRandIntImpl_CryptoMode_Range` + `effects.CryptoIntn`
  unbiased rejection sampling; entropy failure panics)
- [x] Seeded-without-seed → typed `*SeededModeError` with fix hint pointing to `AILANG_SEED` (NOT
  `rand_seed`) — `TestSeeded_NoSeed_TypedError`, `TestRandIntImpl_SeededMode_NoSeed_Error`
- [x] Bare-Rand behaviour byte-identical: golden gate `TestRandGolden_OsSeedByteIdentical` pins the
  `rand_seed(42)`→os-source draw sequence; also live-verified two `.ail` runs identical
- [x] Trace events carry mode+contract label; additive schema verified against existing readers
  (`internal/trace/moded_effect_test.go`, `TestEffectEvent_OldFormatParses`)
- [~] **`examples/modal_rand.ail` rewritten** — the `rand_seed(42)` misuse inside the seeded-mode
  `deterministic_roll` is REMOVED (it seeded the os source inside a seeded fn). The file now
  type-checks AND runs (it did neither before) demonstrating the os path end-to-end, and documents
  the seeded/crypto type-checker limitation + the `AILANG_SEED` seeding contract. `deterministic_roll`
  itself was dropped (not merely re-seeded) because a seeded-mode fn calling os-mode `rand_int` is
  ill-typed under the current invariant subsumption rule — see the blocker below.
- [x] **`examples/expected_fail/effect_budgets_multi.ail`** UNCHANGED (os-mode, correct; premise §0)
- [~] Guide updated; `m-cryptorand.md` header updated (crypto intent now realized). **Teaching prompt
  intentionally NOT updated**: seeded/crypto are not reachable from a runnable `.ail` program yet
  (subsumption blocker), so teaching agents to use them would produce non-checking code.
- [x] `make verify-examples && make lint` green; `make test` green modulo the pre-existing live-network
  flake `TestNetHttpPost/httpbin.org` (503, unrelated)

**BLOCKER surfaced for controller/Mark (NOT a Mark HARD constraint — a new type-system decision):**
A function declared `!{Rand[mode=seeded]}` or `!{Rand[mode=crypto]}` cannot CALL the `std/rand`
wrappers, because those are bare `!{Rand}` (= `mode=os`) and `SubsumeEffectRows` treats modes as
invariant (an os-mode body does not satisfy a seeded/crypto declaration — deliberately, per
`TestSubsumeEffectRows_InvariantOnParams`, which Phase 5 routeable→fixed depends on). The parent doc's
Example 1/2 (`deterministic_now() ! {Clock[mode=pinned]} = now()`) imply the intended rule is that an
explicit declared mode SHOULD subsume an os-mode required effect, but that overturns a tested
invariant and is out of this sprint's ratified scope. **The runtime dispatch machinery is complete and
unit-proven; making seeded/crypto reachable from `.ail` needs a subsumption decision** (does an
explicit declared mode subsume an os/bare required effect? — the `SubsumeEffectRows` validate path is
separate from the `effectParamsCompatible` unification path, so a narrow relaxation is feasible
without weakening function-value mode distinctness).

## Testing Strategy

- **Unit**: registry lookups; per-mode source isolation (seeded sequence unperturbed by
  interleaved os draws); crypto range uniformity smoke.
- **Integration**: end-to-end determinism (`AILANG_SEED=42 ailang run` twice, diff empty);
  seeded-error path; trace label presence.
- **Golden**: os-mode behavioural identity pre/post — including a bare-`!{Rand}` program that
  calls `rand_seed`, diffed against its pre-change output.

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

## Verification Log

| # | Fact / Requirement | Source | Status |
|---|--------------------|--------|--------|
| 1 | `std/rand._rand_seed(n)` → Go `SetRandSeed(int64)` at `internal/builtins/rand.go:51`, which reseeds the SINGLE global `math/rand` source (`randSource`, guarded by `randMu`, crypto-seeded at process init via `cryptoSeed`). Bare `!{Rand}` (os mode) draws from that same global source — so a bare-`!{Rand}` program calling `_rand_seed(42)` IS deterministic today. | Controller probe of `internal/builtins/rand.go` at `v0.30.0-154-gb326c3fd3` | Verified |
| 2 | **Caller sweep EXECUTED at design time** (`grep -rn "rand_seed\|SetRandSeed"` across `internal/`, `std/`, `examples/`, `benchmarks/`; planner-reverified at `v0.30.0-155-g541c1950f`): the ONLY stdlib Rand user is `std/rand` itself (`std/rand.ail:24` `rand_seed(seed) ! {Rand}` → `_rand_seed`; all 5 `std/rand` exports carry `Rand[mode=os]` in the iface — confirmed against `iface.json`). NO `std/game` or other stdlib caller exists. **Exactly ONE** example caller seeds a **seeded-mode** function via `rand_seed` → design-bug to fix in-sprint (row 4): `examples/modal_rand.ail:44` (`deterministic_roll() ! {Rand[mode=seeded]}` body calls `rand_seed(42)`). `examples/expected_fail/effect_budgets_multi.ail:62` seeds inside `main() ! {IO, Rand, Clock}` = **bare/os** (no `mode=seeded`) → CORRECT under the settled contract, NOT a bug; it becomes the bare-Rand golden fixture (row 3). Every remaining `rand_seed` usage is bare-`!{Rand}` (os) → unchanged. | Controller + planner design-time sweep (satisfies gemini round-2) | Verified |
| 3 | Before/after determinism test on a bare-`!{Rand}` program that calls `rand_seed`: capture its output at pre-change HEAD, assert byte-identical output post-change. **`effect_budgets_multi.ail` (bare/os + `rand_seed(42)`) is the fixture.** This is the hard gate backing Conflict Surface point 4. | Success Criteria golden test | Required before merge |
| 4 | The ONE seeded-mode-via-`rand_seed` example caller (`modal_rand.ail:44`) MUST be rewritten to the dedicated seeded-source seed path (`AILANG_SEED` / the seeded-source API), NOT `rand_seed` — per Mark's option-(b) decision. A new teaching diagnostic fires when a `mode=seeded` op draws with no seed provided (fix hint → the dedicated path). | Mark decision 2026-07-24 + Success Criteria | Required before merge |

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

## Quorum verification log

- **Round 1** (2026-07-24): controller PASS; quorum reviewers gpt5-6-sol + gemini-3-1-pro both
  REJECT on the same convergent objection — the Conflict Surface routed `rand_seed` to the new
  seeded source and declared the os source "never author-seedable", contradicting the
  bare-`!{Rand}` byte-identical hard gate (today `rand_seed` reseeds the single global os
  source, so bare-Rand + `rand_seed` programs are deterministic and would have silently lost
  that). **This revision resolves it** by preserving os-source seedability: `rand_seed` keeps
  its existing os/global-source contract; the seeded-mode source is a separate source with its
  own seed path (`AILANG_SEED` or a dedicated mechanism); the golden test now explicitly covers
  a `rand_seed`-calling bare-`!{Rand}` program against pre-change output.
- **Round 2** (2026-07-24): controller PASS; quorum reviewers gpt5-6-sol + gemini-3-1-pro both
  REJECT again — **but the round-1 objection is RESOLVED** (neither re-raised the `rand_seed`/os
  contradiction). The NEW objections are premise-verification completeness, and one is a genuine
  DESIGN gap, so the doc is **PARKED needs-human-review** (the one-revision/one-re-quorum budget is
  spent and the QUORUM narrow-refinement carve-out is **unratified** — its first use awaits Mark):
  1. **gpt5-6-sol** — contradictory baselines: the "Verified current state" header is pinned to
     `v0.28.0-148-g6c25f45e9` while the Verification Log probe (row 1) is `v0.30.0`; wants the whole
     doc pinned to ONE SHA with re-run probes, plus verification rows for the repo-wide
     contract-registry search, bare-Rand default-mode resolution, the elaboration→builtin Rand call
     path, AI mode machinery, `AILANG_SEED` consumers, trace schema/readers, and **whether this
     static `internal/replay` table DUPLICATES sprint-1's `m-effect-mode-validation` registry
     (drift-prevention)** — the last is a design question, not a mechanical fix.
  2. **gemini** — the `_rand_*`/`_rand_seed` caller sweep is deferred to "plan time" while the
     Conflict Surface already asserts std/game etc "stay on bare `!{Rand}`" unverified; wants the
     sweep executed at design time with each caller's effect signature.
  - **Controller reality-check sweep (satisfies gemini's ask):**
    all `std/rand` exports (`rand_int/float/bool`, `rand_seed`, `uuid4`) carry `Rand[mode=os]` in the
    iface; `std/rand.ail:24` `rand_seed(seed) ! {Rand}` → `_rand_seed`; the ONLY stdlib Rand user is
    `std/rand` itself. `examples/modal_rand.ail:44` defines
    `deterministic_roll() ! {Rand[mode=seeded]}` whose body calls `rand_seed(42)` — `rand_seed` is
    `mode=os`, so this is the ONE real seeded-mode-via-`rand_seed` design-bug (fix in-sprint).
    `examples/expected_fail/effect_budgets_multi.ail:62` is bare/os (inside `main`, no `mode=seeded`)
    → CORRECT, becomes the golden fixture (Verification Log rows 2, 3 & 4).
  - **RESOLVED by Mark 2026-07-24 — option (b), explicit seeding surfaces** (the open design question
    below is now closed): a program seeds the seeded-mode source at author level ONLY via the dedicated
    explicit path (`AILANG_SEED` / the seeded-source API) — NOT via `rand_seed` (which keeps its
    os-source contract) and never implicitly. CONSEQUENCE folded into scope: the two example callers
    are rewritten to the dedicated path in-sprint, and a teaching diagnostic fires on a seeded-mode
    draw with no seed. The reviewers' round-2 premise nits (baseline re-pin to ONE SHA + design-time
    caller sweep) are resolved by the controller fold (iter-99): baseline re-pinned to
    `v0.30.0-154-gb326c3fd3`; sweep executed and recorded as Verification Log row 2. The
    registry-duplication concern (gpt5-6-sol: does static `internal/replay` duplicate sprint-1's
    `m-effect-mode-validation` registry?) is a **plan-time spike task** — sprint-1's registry is
    effectSchema mode *validation* (which modes are legal), whereas `internal/replay` is the (effect,
    mode)→contract-label map (replay taxonomy); distinct concerns, but the planner confirms no drift
    (single source of legal modes) before implementation. Per Mark: route to sprint-planner, no third
    text round.

  **Superseded open question (kept for provenance; Mark chose (b) above):** how does a program seed
  the seeded-mode source at author level? (a) a mode-aware seed builtin; ~~(b) `AILANG_SEED`/dedicated-
  path only + rewrite `modal_rand.ail`~~ **← CHOSEN**; (c) rethink.

---

**Document created**: 2026-07-11 (decomposition of M-EFFECT-REFINEMENT, mission iteration 6)
