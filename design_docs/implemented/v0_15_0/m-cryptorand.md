# M-CRYPTORAND: Cryptographic Randomness as a First-Class Effect

**Status**: Planned
**Target**: v0.13.0
**Priority**: P0 — High (security)
**Estimated**: ~30 hours (~1 sprint)
**Dependencies**: None at the type-system level; forward-compatibility constraint with M-EFFECT-REFINEMENT (v1.0.0).
**Commissioning memo**: [rand-determinism-sitrep.md](../rand-determinism-sitrep.md)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Separates deterministic PRNG from opaque CSPRNG; non-determinism is declared |
| A2: Replayability | +1 | Defines explicit replay contract (substitution) for opaque effects |
| A3: Effect Legibility | +1 | CryptoRand is a distinct effect visible in type signatures |
| A4: Explicit Authority | +1 | CryptoRand is capability-gated; security-sensitive code requires explicit grant |
| A5: Bounded Verification | +1 | Lint rule + type-level effect separation locally checkable |
| A6: Safe Concurrency | 0 | No concurrency changes; CSPRNG is thread-safe in `crypto/rand` |
| A7: Machines First | +1 | Type carries security contract — AI agents can reason about it |
| A8: Minimal Syntax | 0 | Adds one effect token; no new grammar constructs |
| A9: Cost Visibility | +1 | Keeps PRNG available for cheap RNG (10–100× cost delta matters) |
| A10: Composability | +1 | Forward-composes with M-EFFECT-REFINEMENT (`!{Rand[mode=crypto]}`) |
| A11: Structured Failure | +1 | Entropy-source failure is a typed, loud error (not silent fallback) |
| A12: System Boundary | +1 | OS entropy crossing is explicit in effect row |

**Net Score: +10** → **Decision: Proceed to implementation**

### Hard Violation Check

- [x] A1: No implicit nondeterminism — CryptoRand is declared
- [x] A3: No hidden side effects — entropy read is in the effect row
- [x] A4: No ambient access — capability-gated
- [x] A7: Type carries the security contract; not optimised for human convenience

## Problem Statement

The v0.10.14 incident (see [rand-determinism-sitrep.md §1](../rand-determinism-sitrep.md)) demonstrated that `!{Rand}` silently conflates three incompatible contracts: reproducible PRNG, OS-entropy PRNG, and cryptographic unpredictability. DocParse minted the same API key across cold starts because the PRNG default leaked into security-sensitive code with no type-level distinction.

The v0.10.14 fix (seed from `crypto/rand` in `init()`) closed the specific cold-start bug, but the structural problem remains: the type `! {Rand}` does not tell the caller whether the output is safe to use as a secret. An AI agent generating AILANG code reads `! {Rand}` as "random = random" and cannot distinguish cheap PRNG from cryptographic entropy.

**Current State:**
- `std/rand` provides `rand_int`, `rand_float`, `rand_bool`, `rand_seed`, `uuid4`. All share one effect row: `! {Rand}`.
- `uuid4()` uses `crypto/rand` internally, but this is invisible to the type.
- No lint or type-level guard against `rand_int` being used to mint keys, tokens, nonces, salts.
- v0.5.1 teaching prompt still says "Deterministic by default with seed 0" — actively wrong after v0.10.14.

**Impact:**
- Security-coded paths (API keys, session tokens, OTP nonces, salts, identity UUIDs) have no type-level protection. One misuse = one cross-tenant breach.
- Autonomous agents (docparse, executor) cannot reliably avoid the footgun — our own agent was the first victim.
- The conflation blocks M-ENTROPY envelopes from meaningfully scoping randomness.

## Goals

**Primary Goal:** Introduce a distinct `!{CryptoRand}` effect and `std/crypto/rand` stdlib module so that cryptographic entropy is a separately-typed, capability-gated, opaque-replay effect — with a migration path that does not break existing `std/rand` callers.

**Success Metrics:**
- `std/crypto/rand` module ships with `crypto_random_bytes`, `crypto_random_int`, `secure_token`, `secure_uuid4`.
- `!{CryptoRand}` effect defined; distinct capability in the runtime.
- Replay engine supports option (d) — deterministic substitution — with strict-refuse fallback.
- Lint rule flags `rand_*` usage in security-coded contexts (pattern-matched names + dataflow).
- Zero regressions on existing `std/rand` call sites; migration is opt-in.
- Forward-compatibility test: a draft `!{Rand[mode=crypto]}` syntax (from M-EFFECT-REFINEMENT prototype) accepts `!{CryptoRand}` as an alias without breaking callers.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| `!{CryptoRand}` as a separate effect token (not a mode on `!{Rand}`) | Pins the surface syntax pre-M-EFFECT-REFINEMENT; must forward-compose without breaking migration | human | design | high |
| Replay contract: substitution-default, refuse-opt-in (reject record-in-trace) | Record-in-trace leaks secrets; substitution matches RFC test-vector practice | human | design | high |
| Deterministic substitution source format (typed fixture vs injected `io.Reader`) | Shapes the replay harness API and test ergonomics | agent | compile | med |
| Lint policy: names *and* dataflow vs names only | Names-only is cheap but leaky; dataflow catches assignment into records/returns | agent | compile | med |
| Entropy-failure behaviour: panic vs typed error | Panic is loud but unrecoverable; typed error composes with `Result`. Current `rand.go` panics | human | design | med |
| Scope parameter syntax deferred to M-EFFECT-REFINEMENT (plain `!{CryptoRand}` in v0.13) | Avoids a second migration when parameterised effects land | human | design | low |
| `rand_seed()` stays in `std/rand` (Tier A — intentional determinism) | Removing it would break simulation/replay use cases that are legitimate | human | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] `!{CryptoRand}` syntax approved by design board (vs `!{Rand[mode=crypto]}` early)
- [ ] Replay contract (d) + refuse-opt-in ratified
- [ ] Entropy-failure: panic vs `Result<_, EntropyError>` decided
- [ ] Forward-compat constraint signed off — `!{CryptoRand}` must desugar to `!{Rand[mode=crypto, scope=_]}` under M-EFFECT-REFINEMENT

## Solution Design

### Overview

Three layers:

1. **Stdlib module `std/crypto/rand`** — typed functions backed by Go's `crypto/rand`. Effect row is `!{CryptoRand}`, not `!{Rand}`.
2. **Effect + capability** — new effect token and new capability (`crypto_rand`) in the runtime. `AILANG_SEED` ignored by contract.
3. **Replay + lint** — replay engine substitution harness; compiler lint rule over security-coded names; dataflow pass to catch leakage.

```
┌─ Tier A ──────────────┐  ┌─ Tier B ──────────────┐  ┌─ Tier C (NEW) ───────┐
│ std/rand + rand_seed   │  │ std/rand (no seed)    │  │ std/crypto/rand       │
│ !{Rand} deterministic  │  │ !{Rand} re-sampleable │  │ !{CryptoRand} opaque  │
│ Seed: AILANG_SEED     │  │ Seed: OS entropy      │  │ Seed: IGNORED         │
│ Replay: deterministic  │  │ Replay: resample      │  │ Replay: substitute    │
└────────────────────────┘  └────────────────────────┘  └───────────────────────┘
```

Today Tier A and Tier B share `!{Rand}` — distinguished at runtime by whether `rand_seed()` was called. Tier C is new and **typed-distinct**.

### Architecture

**Components:**

1. **`std/crypto/rand.ail`** — AILANG surface. Exports: `crypto_random_bytes`, `crypto_random_int`, `secure_token`, `secure_uuid4`. All return `! {CryptoRand}`.

2. **`internal/builtins/crypto_rand.go`** — Go-side builtin implementations. Backed by `crypto/rand.Reader`. Panics on entropy-source failure (matching v0.10.14 precedent for `cryptoSeed`).

3. **Effect registration** — `internal/effects/registry.go` — register `CryptoRand` as a distinct effect with its own capability.

4. **Capability `crypto_rand`** — `internal/capability/caps.go` — defaults to granted in production, denied in replay-without-harness.

5. **Replay harness** — `internal/replay/crypto_substitute.go` — new component. Accepts a fixture source (byte stream); replaces `CryptoRand` calls at runtime during trace replay.

6. **Lint rule `rand-in-crypto-context`** — `internal/lint/rand_security.go` — two passes:
   - Names: functions named `*_key`, `*_token`, `*_nonce`, `*_salt`, `*_secret`; types tagged `Secret<_>`.
   - Dataflow: `rand_*` result reaches `FS.write`, `Net.send`, or persisted datastore calls.
   Emits warning (v0.13), error (one release later, after telemetry).

7. **Telemetry** — `internal/telemetry/rand_usage.go` — record `std/rand` call sites with a synthesised "security-likelihood" tag; surface in dashboard before enforcement.

### Effect + Replay Contract Semantics

- `AILANG_SEED` has **no effect** on `!{CryptoRand}`. Document and test this.
- `ailang run --replay=trace.json` behaviour on `CryptoRand`:
  - No harness → fail loudly with error `ReplayMissingCryptoHarness` (option (a) fallback).
  - `--crypto-harness=file:fixture.bin` → substitute byte stream (option (d)).
  - `--replay-crypto=refuse` → refuse even with harness present (audit mode).
- Trace emits `crypto_rand.call{size=N}` span. Never emits the output bytes. Record-in-trace is **forbidden by contract**.

### Implementation Plan

**Phase 1: Effect + capability plumbing** (~6h)
- [ ] Add `CryptoRand` to effect token set (`internal/effects/`, `internal/types/effects.go`)
- [ ] Register `crypto_rand` capability
- [ ] Wire into effect-row unification
- [ ] Tests: effect row accepts `!{CryptoRand}`, rejects unification with bare `!{Rand}`

**Phase 2: Stdlib module + builtins** (~5h)
- [ ] Create `stdlib/std/crypto/rand.ail` with signatures
- [ ] Implement `internal/builtins/crypto_rand.go` backed by `crypto/rand.Reader`
- [ ] Functions: `crypto_random_bytes(n: int) -> [byte] ! {CryptoRand}`, `crypto_random_int(min, max) -> int ! {CryptoRand}`, `secure_token(n_bytes: int) -> string ! {CryptoRand}` (hex-encoded), `secure_uuid4() -> string ! {CryptoRand}`
- [ ] Entropy failure: panic loudly (follow v0.10.14 precedent); revisit if design board picks `Result` path

**Phase 3: Replay substitution** (~6h)
- [ ] `internal/replay/crypto_substitute.go` — harness interface: `Read(n int) []byte`
- [ ] CLI flags: `--crypto-harness=<source>`, `--replay-crypto=refuse`
- [ ] Trace span: `crypto_rand.call{size}` — never the output
- [ ] Tests: fixture stream replay produces deterministic output; missing harness fails with `ReplayMissingCryptoHarness`; refuse mode fails even with harness

**Phase 4: Lint + telemetry** (~6h)
- [ ] `internal/lint/rand_security.go` — names pass (regex on function/binding names)
- [ ] Dataflow pass: `rand_*` result reaches `FS.write`/`Net.send`/`db.*` — conservative, no false-positive enforcement
- [ ] Telemetry: record call-site category (security-likely vs general) to traces
- [ ] Warning level in v0.13; decision point for error level one release later

**Phase 5: Documentation + migration** (~4h)
- [ ] Update v0.5.1 teaching prompt (coordinate with `prompt-manager` skill)
- [ ] `docs/docs/guides/randomness.md` — tier model, when to use which
- [ ] `examples/crypto_rand.ail` — API key, session token, UUID generation
- [ ] Changelog entry; reference [rand-determinism-sitrep.md](../rand-determinism-sitrep.md)

**Phase 6: Forward-compat test** (~3h)
- [ ] Prototype parser hook accepting `!{Rand[mode=crypto]}` as alias for `!{CryptoRand}`
- [ ] Test: switching between forms preserves type-check, effect-row equivalence, runtime behaviour
- [ ] Document migration path in M-EFFECT-REFINEMENT

### Files to Modify/Create

**New files:**
- `stdlib/std/crypto/rand.ail` — AILANG module surface (~40 LOC)
- `internal/builtins/crypto_rand.go` — Go implementations (~200 LOC)
- `internal/replay/crypto_substitute.go` — replay harness (~180 LOC)
- `internal/lint/rand_security.go` — lint + dataflow (~260 LOC)
- `internal/telemetry/rand_usage.go` — call-site telemetry (~100 LOC)
- `examples/crypto_rand.ail` — usage example (~40 LOC)
- `docs/docs/guides/randomness.md` — tier model + migration guide (~200 LOC)

**Modified files:**
- `internal/effects/registry.go` — register `CryptoRand` (~20 LOC)
- `internal/types/effects.go` — effect token set (~15 LOC)
- `internal/capability/caps.go` — `crypto_rand` capability (~30 LOC)
- `cmd/ailang/main.go` — `--crypto-harness`, `--replay-crypto` flags (~40 LOC)
- `internal/builtins/rand.go` — add telemetry hook, link doc pointing to crypto/rand (~20 LOC)
- `prompts/v0.5.1.md` (or successor) — rewrite Rand section (~80 LOC delta)

**Grand Total: ~1,225 LOC (new: 1,020; modified: 205)**

## Examples

### Example 1: API key generation (the docparse case done right)

**Before (broken by v0.10.13 default; silently wrong):**
```ailang
import std/rand (rand_int)

export func mint_api_key() -> string ! {Rand} {
  -- Every cold start produces the same key. Tenant-fatal.
  let bytes = [rand_int(0, 255) | _ <- [1..16]]
  hex_encode(bytes)
}
```

**After:**
```ailang
import std/crypto/rand (secure_token)

export func mint_api_key() -> string ! {CryptoRand} {
  secure_token(16)  -- 16 bytes of OS entropy, hex-encoded
}
```

Effect row `!{CryptoRand}` makes the security contract type-visible. Linter rejects the "Before" version (`*_key` name + return-into-FS dataflow).

### Example 2: Simulation (Tier A — seeded PRNG, legitimate)

```ailang
import std/rand (rand_float, rand_seed)

export func monte_carlo(n: int, seed: int) -> float ! {Rand} {
  rand_seed(seed)
  let samples = [rand_float(0.0, 1.0) | _ <- [1..n]]
  mean(samples)
}
```

`!{Rand}`, seeded, reproducible. `AILANG_SEED` can pin globally. Lint passes (no security-coded name, no persist-to-FS dataflow). Untouched by this milestone.

### Example 3: Replay with fixture

```bash
# Record
$ AILANG_TRACE=standard ailang run mint_keys.ail
# Trace contains: crypto_rand.call{size=16} spans (no bytes)

# Replay without harness — fails loudly
$ ailang run --replay=trace.json mint_keys.ail
ReplayMissingCryptoHarness: trace contains 3 CryptoRand calls; no substitution source provided
  Hint: pass --crypto-harness=file:fixture.bin or --replay-crypto=refuse

# Replay with deterministic fixture
$ ailang run --replay=trace.json --crypto-harness=file:fixture.bin mint_keys.ail
# CryptoRand calls served from fixture.bin; same output every run
```

## Success Criteria

- [ ] `std/crypto/rand` module exports `crypto_random_bytes`, `crypto_random_int`, `secure_token`, `secure_uuid4`
- [ ] All four functions have effect row `! {CryptoRand}`, not `! {Rand}`
- [ ] `AILANG_SEED` verified to have no effect on `CryptoRand` (integration test)
- [ ] Replay with fixture produces deterministic output; without fixture fails loudly
- [ ] Trace never contains `CryptoRand` output bytes (only call metadata)
- [ ] Lint rule flags `rand_*` in `mint_*_key`, `generate_*_token`, `create_*_nonce` patterns
- [ ] Dataflow pass catches `rand_int` result persisted to `FS.write` / Firestore-like calls
- [ ] Telemetry records security-likelihood tag on `std/rand` call sites
- [ ] v0.5.1 teaching prompt rewritten (coordinated with `prompt-manager`)
- [ ] Forward-compat prototype: `!{Rand[mode=crypto]}` alias parses and type-checks equivalently to `!{CryptoRand}`
- [ ] Zero regressions on existing `std/rand` callers (verified via `make verify-examples`)
- [ ] Changelog + release notes reference the sit-rep

## Testing Strategy

**Unit tests:**
- `crypto_random_bytes` returns correct length; no two consecutive calls match (entropy smoke test)
- `secure_token` hex-encoding round-trip
- Entropy-failure handler (inject mock `io.Reader` returning `io.EOF`) panics loudly
- Effect row: `!{CryptoRand}` does not unify with `!{Rand}`
- Lint: every example in [examples/] that mints keys/tokens uses `crypto_*`; lint fails on synthetic counter-examples

**Integration tests:**
- `AILANG_SEED=42 ailang run mint_keys.ail` — two runs produce different outputs (CryptoRand ignored seed)
- Trace round-trip: record → replay with fixture → identical outputs; record → replay without fixture → error
- Replay refuse-mode: fails even with harness
- Forward-compat: `mod X = module X where use_crypto : () -> () ! {Rand[mode=crypto]} = ...` parses (behind feature flag) and typechecks identically to `!{CryptoRand}`

**Manual testing:**
- `ailang run examples/crypto_rand.ail` — generate key, token, UUID
- `grep -n 'rand_' stdlib/` — verify no security-coded callers of `rand_*` remain in stdlib itself
- Dashboard shows telemetry tag for security-likely call sites

## Deferred Decisions

- **Fixture source formats.** File, HTTP endpoint, embedded test fixture, mocked capability — agent may choose initial set; more can be added without breaking compat.
- **Dataflow-pass precision.** Conservative initially; tuning thresholds is an agent decision based on false-positive rate in telemetry.
- **Lint phrase-matching rules.** Which regexes exactly (`*_key` vs `*_api_key`, case sensitivity, language variants) — agent may extend; keep the list in a config file so security team can edit.
- **Telemetry retention + dashboarding.** Agent may wire into existing AILANG dashboard; human approves before privacy-sensitive usage patterns are logged.
- **Error type shape for `EntropyError`** (if design board picks `Result` over panic) — agent may resolve; match existing stdlib error conventions.

## Non-Goals

- **Parameterised effects syntax.** `!{Rand[mode=...]}` is deferred to [M-EFFECT-REFINEMENT](../v1_0_0/m-effect-refinement.md). This milestone uses the plain `!{CryptoRand}` form and only *proves* forward-compatibility via a prototype alias.
- **Capability scoping.** `!{CryptoRand[scope=identity|session|test-denied]}` is also deferred to M-EFFECT-REFINEMENT. This milestone introduces one unscoped `CryptoRand` capability.
- **Removing or deprecating `rand_seed()`.** It is the correct API for Tier A (seeded simulation, replay-critical RNG).
- **Audit sweep of other "convenience defaults"** in `Clock`, FS fixtures, etc. That's a separate sit-rep follow-on item.
- **Automated rewrite of user code.** Autofix is offered for `std/rand` → `std/crypto/rand` migrations under user control; we do not rewrite without consent.

## Timeline

**Week 1** (16h):
- Phase 1: effect + capability plumbing (6h)
- Phase 2: stdlib module + builtins (5h)
- Phase 3 start: replay harness scaffolding (5h)

**Week 2** (14h):
- Phase 3 finish: replay contract + CLI flags (1h)
- Phase 4: lint + telemetry (6h)
- Phase 5: docs + migration (4h)
- Phase 6: forward-compat test (3h)

**Total: ~30 hours across 2 weeks**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| `!{CryptoRand}` syntax painted into a corner before M-EFFECT-REFINEMENT | High | Phase 6 forward-compat prototype is a hard gate — design doc does not merge until alias test passes |
| Dataflow lint false-positives break legitimate code | Med | Warning-only in v0.13; promote to error only after one release of telemetry |
| Replay substitution fixture format forks into many incompatible flavours | Med | Single primary format (byte stream); others as documented extensions behind a plugin interface |
| Teaching prompt rewrite drifts from implementation | Med | `prompt-manager` skill owns the rewrite; acceptance criterion references it |
| CryptoRand panic on entropy failure is unrecoverable in server contexts | Med | Follows v0.10.14 precedent; revisit in `Result<_, EntropyError>` path if board prefers. Decision captured in Design Freeze |
| Parameterised-effects proponents want `!{CryptoRand}` rejected in favour of `!{Rand[mode=crypto]}` now | Low/High | The sit-rep's recommendation is plain `!{CryptoRand}` in v0.13 to avoid a double migration; Design Freeze item makes this explicit |

## Related Documents

**Commissioning memo:**
- [design_docs/planned/rand-determinism-sitrep.md](../rand-determinism-sitrep.md) — full analysis; this doc operationalises its §7, §9, §11, §12

**Follow-on:**
- [design_docs/planned/v1_0_0/m-effect-refinement.md](../v1_0_0/m-effect-refinement.md) — generalises parameterised effects + replay contracts across Rand, Clock, Net, FS; this milestone must forward-compose with it

**Integrates with:**
- [design_docs/planned/v1_0_0/m-entropy-budgets.md](../v1_0_0/m-entropy-budgets.md) — behavioural entropy axis, randomness source declaration

**Implemented (informs design):**
- [design_docs/implemented/v0_10_0/m-stdlib-crypto.md](../../implemented/v0_10_0/m-stdlib-crypto.md) — existing `std/crypto` hashing/HMAC surface; pattern for typed crypto modules
- [design_docs/implemented/v0_10_0/m-stdlib-crypto-jwt.md](../../implemented/v0_10_0/m-stdlib-crypto-jwt.md) — JWT signing; consumes secure randomness

**Superseded:**
- [design_docs/archive/v0_5_1_m-prompt-rand-effect.md](../../archive/v0_5_1_m-prompt-rand-effect.md) — teaching prompt with the now-incorrect "seed 0 by default" framing

## References

- [Design Axioms](/docs/references/axioms) — A1, A3, A4, A7 compliance
- [Philosophical Foundations](/docs/references/philosophical-foundations) — declared non-determinism
- [docparse incident report](msg_20260408_165307_ca9998b6) — agent message reporting the original breach
- [commit d02346d8](https://github.com/sunholo/ailang/commit/d02346d8) — v0.10.14 P0 fix
- NIST SP 800-90A — DRBG taxonomy (informs replay substitution design)
- RFC 6979 — deterministic signatures (precedent for substitution-based cryptographic determinism)

## Future Work

- **M-EFFECT-REFINEMENT** (v1.0.0) — generalises to `!{Rand[mode=...]}`, `!{Clock[mode=...]}`, `!{Net[mode=...]}`, `!{FS[mode=...]}`; adds scope parameters
- **Audit sweep** — other effect defaults that silently conflate contracts (sibling to this milestone, but separate design)
- **Lint error-level promotion** — one release after v0.13, convert `rand-in-crypto-context` warning to error
- **Crypto-typed `Secret<T>` wrapper** — future milestone; tagged types that the linter can follow even without name matching

---

**Document created**: 2026-04-15
**Last updated**: 2026-04-17
