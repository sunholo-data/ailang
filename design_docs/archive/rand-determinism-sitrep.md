---
status: discussion
revision: 2
type: design-board-memo
milestone: (pending)
supersedes: none
related:
  - design_docs/archive/v0_5_1_m-prompt-rand-effect.md
  - design_docs/implemented/v0_10_0/m-stdlib-crypto.md
  - design_docs/planned/v1_0_0/m-entropy-budgets.md
  - docs/docs/guides/consumer-contract-v0.5.md
spawns:
  - M-EFFECT-REFINEMENT (parameterized effects + replay contracts)
  - M-CRYPTORAND (std/crypto/rand + !{CryptoRand} effect)
incident_refs:
  - commit d02346d8 (v0.10.14 P0 fix)
  - msg_20260408_165307_ca9998b6 (docparse identical API key report)
date: 2026-04-15
revised: 2026-04-15
---

# Sit-Rep: Determinism vs. Unpredictability — The Rand Axiom Tension

**Audience**: AILANG Program Design Board
**Purpose**: Analyse the v0.10.14 `std/rand` incident against Principle 2 (Reproducibility), decide whether the axiom holds, and commission the follow-on work the incident has exposed.

> **v2 thesis**: The bug was not that randomness was deterministic. **The bug was that entropy had no type.** That is precisely the class of problem AILANG exists to eliminate — and the fix is not to patch `std/rand`, but to recognise that effects in AILANG currently model *what happens* without modelling *under which contract*. This memo proposes both a point fix and the general design work it demands.

---

## 1. What happened

**Incident** (reported by `docparse` agent, fixed in `d02346d8` / v0.10.14):

- `std/rand` in v0.10.13 and earlier seeded `math/rand` with `NewSource(0)` in its package `init()`.
- Every fresh process (Cloud Run cold start, container restart) produced the *identical* sequence from `rand_int`, `rand_float`, `rand_bool` unless the program explicitly called `_rand_seed()`.
- Real-world impact: DocParse minted the **same API key** (`dp_fc4af748e9e0a03496c86bdae37091f6`) across ≥4 cold starts in 2 GCP projects, overwriting legitimate user keys in Firestore.
- Fix: `init()` now calls `cryptoSeed()` — reads an `int64` from `crypto/rand` and panics on entropy failure. `SetRandSeed()` / `_rand_seed()` remain for explicit reproducibility (tests, game replays).
- `uuid4()` was always correct — it reads `crypto/rand` directly and ignores the PRNG seed.

**Location**: [internal/builtins/rand.go](internal/builtins/rand.go), [std/rand.ail](std/rand.ail), env-seed plumbing in [internal/effects/context.go:76-88](internal/effects/context.go#L76-L88).

---

## 2. The real root cause

The surface bug was "default seed = 0". The **real** bug is one level deeper:

> A single effect (`!{Rand}`) was overloaded across three incompatible contracts: reproducible PRNG, convenience randomness, and security entropy.

That is a violation of a deeper invariant than Principle 2:

> **Every effect must correspond to exactly one semantic contract.**

`Rand` today is not a contract — it is a union of contracts with a shared, silent default. The same shape appears (latently) elsewhere in the system:

| Effect | Collapsed contracts |
|---|---|
| `Rand` | seeded PRNG / OS-entropy / CSPRNG |
| `Clock` | deterministic pinned time / wall-clock |
| `Net` | replayable (recorded) / live |
| `FS` | sandboxed fixture / real filesystem |

The Rand incident is the first one to fire because crypto keys have the sharpest failure mode. The structural problem is general.

---

## 3. Three properties that got conflated

| Property | What it means | Who needs it |
|---|---|---|
| **Reproducibility** | Same declared inputs + same seed ⇒ same outputs | Auditors, testers, replayers |
| **Determinism** | No *hidden* non-determinism; every source of variation is named | Language designers, type system |
| **Unpredictability** | Output is computationally infeasible to guess | Security (keys, tokens, nonces) |

Principle 2 is about (1) and (2). It is **silent** on (3) — unpredictability is a security requirement, not an epistemic one. "Default seed = 0" silently promoted (1) above (3) in a context where (3) was load-bearing.

**The axiom is intact. The type discipline around it was not.**

---

## 4. Why a "documentation fix" is insufficient

The v0.5.1 prompt doc literally said *"Deterministic by default with seed 0 (can be overridden)"*. That didn't help:

1. The failure mode is silent (keys look random; collision only visible across processes).
2. The blast radius is cross-tenant (one user's key overwrites another's).
3. AI agents generating AILANG code will not read the warning — they will read the type `! {Rand}` and assume "random = random".
4. Our own agent, docparse, got burned. If our tooling can't avoid the footgun, external users certainly won't.

Documentation is necessary but not sufficient. The type must carry the contract.

---

## 5. Refined Principle 2

**Current wording** (paraphrased): *Reproducibility is the precondition of trust.*

**Proposed refinement**:

> **A computation is reproducible iff every source of variation is (a) declared as an effect and (b) bound to a replay contract.** Declaration is necessary but not sufficient; the replay contract fixes *how* the effect behaves when the computation is re-run.

Three replay contracts suffice for v1:

| Contract | Behaviour on replay | Example |
|---|---|---|
| **Deterministic** | Same inputs ⇒ same outputs. Seed is part of the input. | Seeded PRNG, pinned `Clock` |
| **Re-sampleable** | Re-draw from the same distribution; distribution is part of the contract | OS-entropy PRNG, live `Clock` |
| **Opaque** | Never persisted in the trace; replay requires injected substitute or refusal | CSPRNG, secret-bearing `Net` calls |

This closes the gap the v1 memo left open: it was too easy to read "declared = reproducible" as "declared = safe to persist". It isn't. A `CryptoRand` draw is declared **and** opaque; its trace says *that* it was called, never *what it produced*.

---

## 6. Does this break Principle 2?

No. Principle 2 (refined) explicitly welcomes opaque declared non-determinism. A `CryptoRand` effect is non-determinism that is:

- **Declared** in the type — visible to the compiler and auditor.
- **Capability-gated** — the runtime decides whether to grant it.
- **Opaque by replay contract** — traces record *that* it was invoked, not its output.

The recipe analogy holds, with the refinement: a recipe that says *"add a pinch of fresh yeast from today's batch"* is still a recipe — provided it says so. What violates the axiom is *hidden* entropy, or entropy whose replay contract is unstated.

---

## 7. The tier model (revised)

The v1 memo proposed three tiers (A: seeded PRNG, B: OS-entropy PRNG default, C: CSPRNG). Review objection: "Tier B is not a stable semantic category — not reproducible, not cryptographically guaranteed, not distinguishable at runtime."

**Partially accepted.** Tier B *was* underspecified. But collapsing B into C is wrong on cost and semantics:

- CSPRNG draws are ~10–100× more expensive than PRNG draws; forcing them for simulations, ML shuffles, property tests, and Monte Carlo is a real tax for zero benefit.
- A simulation that wants "a million cheap normal deviates, reproducible when I pass a seed" is not served by `CryptoRand`'s "seed ignored by design" contract.

**The correct move is to type B, not delete it.** Parameterised effects (§8) make B distinguishable at the type level, which was the only substantive objection.

### Revised tier model

| Tier | Effect (parameterised) | Replay contract | Use cases |
|---|---|---|---|
| **A** | `!{Rand[mode=seeded]}` | Deterministic | Tests, simulation, replay-critical RNG |
| **B** | `!{Rand[mode=os]}` | Re-sampleable | General application randomness (cheap, non-security) |
| **C** | `!{CryptoRand}` | Opaque | Keys, tokens, nonces, salts, identity UUIDs |

Key properties:
- A and B share the `Rand` effect (same capability surface, different mode); C is a **distinct effect** (different capability, different replay contract, distinct audit class).
- `AILANG_SEED` applies to `Rand[mode=seeded]` only. `Rand[mode=os]` ignores it. `CryptoRand` ignores it by contract.
- The compiler can require an explicit mode in security-sensitive scopes (enforced via M-ENTROPY envelope, §10).

---

## 8. The deeper generalisation: parameterised effects

The Rand incident exposes a pattern that reappears across the effect system. Today AILANG models:

- `!{Clock}` — but is this pinned (deterministic) or wall-clock (re-sampleable)?
- `!{Net}` — live or recorded/replayed?
- `!{FS}` — sandbox fixture or real disk?
- `!{Rand}` — which of three contracts?

All of these are effect *families* masquerading as single effects. The correct generalisation is:

```
!{Rand[mode=seeded]}       !{Rand[mode=os]}        !{CryptoRand}
!{Clock[mode=pinned]}      !{Clock[mode=wall]}
!{Net[mode=recorded]}      !{Net[mode=live]}
!{FS[mode=fixture]}        !{FS[mode=real]}
```

Each parameter is bound to a replay contract; the compiler and capability system use the parameter for policy. This is a significant language change and should not be bundled with the Rand point fix — see §11 for the commissioned follow-on.

**Capability scoping.** Where security is involved, the parameter can carry a scope:

```
!{CryptoRand[scope=identity]}    -- API keys, user IDs
!{CryptoRand[scope=session]}     -- session tokens
!{CryptoRand[scope=test-denied]} -- must not be reachable from test context
```

---

## 9. Replay semantics for `CryptoRand`

The v1 memo offered three options and picked "refuse". Review correctly points out all three are bad:

| Option | Problem |
|---|---|
| (a) refuse replay | breaks debugging workflows |
| (b) redraw fresh entropy | breaks causality / reproducibility of bug |
| (c) record output in trace | **leaks secrets** — fatal |

**Adopted: option (d) — deterministic substitution via replay harness.** On replay, `CryptoRand` calls are satisfied from an injected fixture stream (test vectors, KATs, or a harness-provided deterministic source). Not original values, not fresh entropy, not persisted output.

- **Default mode**: substitution required. If no harness is configured, replay fails loudly (merged (a) as the safety net).
- **Opt-in**: `--replay-crypto=refuse` for audit contexts that must never see synthetic crypto.
- **Rejected**: (b) and (c) entirely — (c) is a secrets-leak vulnerability, (b) breaks the property that replay is a function of the trace.

This matches how real cryptographic test suites operate (RFC-mandated test vectors, NIST KATs).

---

## 10. Relationship to M-ENTROPY

[M-ENTROPY](design_docs/planned/v1_0_0/m-entropy-budgets.md) distinguishes axes of entropy (semantic, behavioural, interpretive, authority, temporal). The revised envelope schema, normalised for composition:

```yaml
entropy:
  behavioral:
    randomness:
      sources: [seeded, os, crypto]
      default: crypto
      replay:
        seeded: deterministic
        os:     resampleable
        crypto: substitute
```

This aligns 1:1 with the effect model (§8), the replay contract taxonomy (§5), and the capability system. For a module that mints API keys, the envelope can declare `sources: [crypto]` and the compiler rejects any `Rand[mode=*]` call at design-validation time — the safety net we lacked.

---

## 11. Work to commission

### Immediate (point fix, already mostly shipped)
- ✅ v0.10.14: `cryptoSeed()` default (shipped).
- **New**: audit sweep for any other "seed = 0 by default" or similar convenience defaults in `Clock`, test harnesses, FS fixtures.

### Near-term: **M-CRYPTORAND** (proposed v0.11 or v0.12)
- Introduce `std/crypto/rand` and the `!{CryptoRand}` effect.
- Implement replay contract (d) substitution + strict-refuse mode.
- Audit of existing `std/rand` call sites; lint rule flagging `rand_*` in contexts matching key/token/nonce/salt patterns.
- Static analysis + autofix: `rand_int → crypto_rand_int` where the usage pattern is security-coded.
- Telemetry pass to observe real-world misuse before turning on enforcement.

### Medium-term: **M-EFFECT-REFINEMENT** (proposed v0.13+, commissioned by this memo)
- Parameterised effects (`!{E[mode=...]}`) in the type system and effect row algebra.
- Uniform replay contract taxonomy across `Rand`, `Clock`, `Net`, `FS`.
- Capability scoping on effect parameters.
- Integration with M-ENTROPY envelopes as the authoritative policy layer.

**M-CRYPTORAND is a point solution; M-EFFECT-REFINEMENT is the general solution. The point solution must be designed to forward-compose with the general one — i.e., `!{CryptoRand}` should be readable as `!{Rand[mode=crypto, scope=_]}` under M-EFFECT-REFINEMENT without a breaking migration.**

---

## 12. Migration strategy (revised)

The v1 memo asked whether to deprecate `rand_seed()`. That's the wrong level — it's an API question masking a semantic one. The right migration is:

1. **Lint rule (P0)** — forbid `rand_*` in functions whose names or return types match security patterns (`*_key`, `*_token`, `*_nonce`, `*_salt`, `*_secret`, types tagged `Secret`).
2. **Static analysis** — detect key/token/session generation patterns via dataflow (result reaches `FS.write`, `Net.send`, Firestore writes, etc.).
3. **Autofix** — rewrite `rand_int` → `crypto_rand_int` where the lint + static pass agrees.
4. **Telemetry** — instrument first, enforce second. We need to see the misuse before we break builds on it.

`rand_seed()` itself stays — it's not dangerous in Tier A, which is its intended home.

---

## 13. Open questions for the board

1. **Accept parameterised effects as a language direction?** (Yes → commission M-EFFECT-REFINEMENT.) Big scope; needs explicit ratification before M-CRYPTORAND pins any syntax.
2. **Scope syntax on `CryptoRand`.** `!{CryptoRand[scope=identity]}` in v0.11, or plain `!{CryptoRand}` with scopes deferred to M-EFFECT-REFINEMENT? Recommendation: plain in v0.11, add scopes when parameterised effects land, to avoid a second migration.
3. **Replay harness API.** What does `CryptoRand` substitution look like from the test/replay side — an injected `io.Reader`, a typed fixture map, or a capability?
4. **Telemetry before enforcement.** How long do we observe `rand_*` misuse before turning lint into error? Recommend one minor release.
5. **Prompt/teaching doc ownership.** Design board signs off on refined Principle 2, then `prompt-manager` skill rewrites the v0.5.1 teaching doc.
6. **Other parameterised effects first?** Is Rand the right pilot for M-EFFECT-REFINEMENT, or does `Clock` (simpler) give us a cleaner prototype? Recommend Rand — the motivation is concrete and auditable.

---

## 14. Recommendations to the board

1. **Ratify** refined Principle 2 (§5): declaration + replay contract.
2. **Ratify** that the Rand incident was an effect-taxonomy failure, not a determinism failure.
3. **Approve** the three-tier model with parameterised types (§7).
4. **Adopt** replay contract (d) substitution (§9) for opaque effects.
5. **Commission** M-CRYPTORAND (near-term) and M-EFFECT-REFINEMENT (medium-term) with the forward-compatibility constraint in §11.
6. **Integrate** the normalised entropy-source schema into M-ENTROPY before M-ENTROPY lands (§10).
7. **Run** the audit sweep for similar "convenience defaults" elsewhere in the effect system.

---

**Prepared for**: Program Design Board
**Prepared by**: Claude (sessions 2026-04-15, revised same day after board-level review)
**Requested action**: Discuss at next design review; assign decisions on §13 open questions; commission M-CRYPTORAND and M-EFFECT-REFINEMENT.

---

### Appendix A — One-line thesis

> **The bug was that entropy had no type.** Everything else follows.

### Appendix B — Changes from v1

- Thesis promoted to the top (was §10 aphorism).
- Principle 2 refined with explicit replay contracts (deterministic / re-sampleable / opaque).
- Tier model retained but reframed as parameterised effects, not three sibling APIs; Tier B defended on cost/semantics grounds but made typed.
- Replay semantics: adopted substitution (option d); rejected record-in-trace (option c) as a secrets-leak.
- `CryptoRand` capability scoping added (`scope=identity|session|test-denied`).
- Migration strategy reframed from API deprecation to lint + static analysis + autofix + telemetry.
- M-EFFECT-REFINEMENT commissioned as a distinct medium-term design doc; M-CRYPTORAND constrained to forward-compose with it.
- M-ENTROPY schema normalised to match effect/replay taxonomy.
