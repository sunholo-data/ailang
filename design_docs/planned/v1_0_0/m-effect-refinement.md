# M-EFFECT-REFINEMENT: Parameterised Effects and Unified Replay Contracts

**Status**: Decomposed (2026-07-11) — umbrella doc; execution tracked in the four sprint docs below
**Target**: v1.0.0
**Priority**: P1 — Medium (strategic language feature)
**Estimated**: ~90 hours original; ~64h remaining after v0.15.0 landings, split into 4 sprint docs

## Decomposition (2026-07-11, mission iteration 6)

This doc is now the **umbrella/reference**: the taxonomy tables, unification rationale, and
worked examples below remain normative, but execution is tracked per sprint doc. Repo-verified
status of the original eight phases:

| Original phase | Status | Where |
|---|---|---|
| P1 parser + AST | ✅ shipped v0.15.0 | [M-EFFECT-REFINEMENT-PHASE1](../../implemented/v0_15_x/m-effect-refinement-phase1.md) |
| P2 effect row algebra | ✅ shipped v0.15.0 | same (invariant unification + default-mode table) |
| P3 replay contract registry | Planned | [m-effect-replay-contracts](m-effect-replay-contracts.md) (sprint 2, ~3d) |
| P4 capability scoping | Planned | [m-effect-scope-params](m-effect-scope-params.md) (sprint 4, ~2.5d; release-gate re-score candidate) |
| P5 Clock/Net/FS port | Planned | [m-effect-clock-net-fs-modes](m-effect-clock-net-fs-modes.md) (sprint 3, ~3d) |
| P5 AI port | ✅ shipped v0.15.0 | [M-AI-EFFECT-MODES](../../implemented/v0_15_x/m-ai-effect-modes.md); loose ends in [m-ai-effect-modes-followups](m-ai-effect-modes-followups.md) (P2) |
| P6 M-ENTROPY integration | Routed OUT of this track | ships with [M-ENTROPY](m-entropy-budgets.md) itself (not v1.0-required); envelope mode-validation composes with sprint-1 machinery |
| P7 CryptoRand alias | Scope-reduced away (v0.15.0) | no `CryptoRand` token exists; crypto-strength runtime intent superseded by `Rand[mode=crypto]` dispatch in sprint 2 |
| P8 docs + examples | Partially shipped v0.15.0 | guide + modal_rand shipped; remainder folded into sprints 2–3 |
| — NEW: closed-set enforcement | Planned (discovered 2026-07-11: guide's "typechecker rejects unknown values" is false — `Rand[mode=banana]` passes `ailang check`) | [m-effect-mode-validation](m-effect-mode-validation.md) (sprint 1, ~1d, FIRST) |

Sprint order (dependency-driven): **1 validation → 2 replay-contracts → 3 clock-net-fs → 4 scope**.
**Dependencies**:
  - [M-CRYPTORAND](../v0_13_0/m-cryptorand.md) (v0.13.0) — pilot milestone; provides the forward-compat constraint
  - [M-CAPABILITY-BUDGETS](../../implemented/v0_6_2/m-capability-budgets.md) (v0.6.2 ✅) — budget plumbing for scoped effects
  - [M-ENTROPY](m-entropy-budgets.md) — integrates as the authoritative policy layer
**Commissioning memo**: [rand-determinism-sitrep.md](../rand-determinism-sitrep.md)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Replay contracts make determinism properties explicit in the type |
| A2: Replayability | +1 | **Primary goal** — unified taxonomy of how effects behave on replay |
| A3: Effect Legibility | +1 | Effect mode is visible in the type signature, not hidden in runtime |
| A4: Explicit Authority | +1 | Scope parameters make capability grants fine-grained and typed |
| A5: Bounded Verification | +1 | Per-effect modes are locally checkable; composable validation |
| A6: Safe Concurrency | 0 | No direct concurrency changes |
| A7: Machines First | +1 | AI agents can reason about effect contracts from the type alone |
| A8: Minimal Syntax | 0 | Adds `[mode=...]` / `[scope=...]` annotation; confined to effect rows |
| A9: Cost Visibility | +1 | Mode distinction preserves cheap vs expensive effect variants |
| A10: Composability | +1 | Uniform framework across Rand, Clock, Net, FS; composes with M-ENTROPY |
| A11: Structured Failure | +1 | Replay-contract violations are typed errors with fix hints |
| A12: System Boundary | +1 | Mode parameter makes boundary-crossing explicit |

**Net Score: +10** → **Decision: Proceed to implementation**

### Hard Violation Check

- [x] A1: Replay contracts strengthen determinism, not weaken it
- [x] A3: Effect mode is part of the effect row — visible, not ambient
- [x] A4: Scope parameters are typed capabilities; no ambient authority
- [x] A7: Machine-parseable syntax; typed contract carries semantic information

## Problem Statement

The v0.10.14 `std/rand` incident (see [rand-determinism-sitrep.md §2](../rand-determinism-sitrep.md)) was not a one-off bug. It was the first failure of a structural pattern:

> Effects in AILANG currently model *what happens* without modelling *under which contract*.

The same shape — single effect token masking multiple incompatible contracts — is latent across the effect system:

| Effect | Collapsed contracts |
|---|---|
| `!{Rand}` | seeded PRNG / OS-entropy PRNG / CSPRNG |
| `!{Clock}` | pinned-deterministic / wall-clock |
| `!{Net}` | recorded/replayable / live |
| `!{FS}` | fixture-sandbox / real-disk |
| `!{AI}` | direct fixed-model / runtime-routed / replay-only / BYOK |

[M-CRYPTORAND](../v0_13_0/m-cryptorand.md) is a point fix: it splits Rand into `!{Rand}` and `!{CryptoRand}`. But duplicating effect tokens for every contract variation does not scale, and it leaves `Clock`, `Net`, `FS`, `AI` uncorrected.

The AI row was added to the taxonomy by [M-AI-OPENROUTER](../../implemented/v0_16_x/m-ai-openrouter-provider.md) (v0.16.0): the OpenRouter sprint shipped runtime-routed AI calls (`AIRoutingPolicy`) and captured routing decisions in the trace, but had to enforce explicit opt-in via a `--allow-routing` runtime flag rather than the planned `!{AI[mode=routeable]}` type-level marker — because parameterised effects didn't exist yet. M-EFFECT-REFINEMENT is the canonical home for that deferred work; see [Example 4: Modal AI](#example-4-modal-ai) below.

**Current State:**
- `std/clock` can be seeded (`AILANG_SEED` pins time) or wall-clock, with no type-level distinction.
- `std/net` traces are replayable in principle, but the type `!{Net}` does not say whether a call is live or recorded.
- `std/fs` sandbox vs real-disk is a runtime flag, invisible in the type.
- M-ENTROPY (planned) proposes envelope-level policy but has no compiler-visible hook to enforce "this module may only use pinned clocks".

**Impact:**
- Every new effect category repeats the Rand taxonomy problem; every one will eventually ship its own incident.
- AI agents and auditors have no uniform way to read an effect row and understand its replay contract.
- M-ENTROPY cannot reach into effect rows for behavioural constraints — it can only gate at the capability layer, which is coarser.

## Goals

**Primary Goal:** Unify effect contract variation under a single parameterisation mechanism — `!{E[mode=..., scope=...]}` — with a taxonomy of replay contracts (deterministic / re-sampleable / opaque) that applies uniformly to Rand, Clock, Net, FS.

**Success Metrics:**
- Parameterised effect syntax ships in type system and effect row algebra.
- Replay contract taxonomy defined and enforced at type-check time.
- `!{CryptoRand}` (from M-CRYPTORAND) becomes a typed alias for `!{Rand[mode=crypto, scope=_]}` — zero breakage to existing callers.
- Clock, Net, FS ported to parameterised form with back-compat aliases.
- M-ENTROPY envelopes can constrain mode choices per-module.
- At least three worked examples (Rand, Clock, Net) demonstrate mode-aware replay in trace tooling.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Surface syntax: `!{E[mode=X]}` vs `!{E.X}` vs typeclass-style refinement | Defines language-visible parser + AST shape; ~all future effect docs depend on this | human | design | high |
| Mode set is fixed vs open (user-definable modes) | Fixed = auditable; open = extensible but unbounded; affects whole type theory | human | design | high |
| Replay contract taxonomy: the three labels {deterministic, re-sampleable, opaque} as the total surface | Closes or opens future extensions; changing later is a type-system migration | human | design | high |
| Capability scoping on effect parameters (`scope=identity\|session\|test-denied`) | Couples effect system to capability system at the type level | human | design | high |
| Back-compat aliasing strategy — old `!{Rand}` remains as `!{Rand[mode=os]}` alias vs forced migration | Forced migration breaks every existing AILANG program; aliasing ships faster | human | design | high |
| Unification rules — does `!{Rand[mode=os]}` unify with `!{Rand[mode=seeded]}`? | Subtly affects type inference in polymorphic code; wrong choice causes inference surprises | compiler | compile | high |
| M-ENTROPY integration: entropy envelope can veto modes at compile time (vs warn only) | Makes M-ENTROPY normative for mode selection; larger change than v0.7 plan | human | design | med |
| Clock/Net/FS modes shipped alongside Rand, or Rand-only in v1.0 with others following | Staging vs big-bang; affects timeline and risk | human | design | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [ ] Effect parameter syntax — `[mode=X]` vs alternative — locked
- [ ] Mode set: closed or open? Decision recorded with rationale
- [ ] Three replay contracts {deterministic, re-sampleable, opaque} ratified as the complete surface for v1.0
- [ ] Scope parameter schema agreed (`identity|session|test-denied|_` as initial set)
- [ ] Unification rules for parameterised effects specified (tighten-only? invariant? subtyped?)
- [ ] Back-compat aliasing strategy approved — bare `!{Rand}` must continue to work
- [ ] Staging plan: Rand-first vs all-four-simultaneously

## Solution Design

### Overview

Three integrated pieces:

1. **Parameterised effect rows** — effect tokens carry a named-parameter map. `!{Rand[mode=seeded]}`, `!{Clock[mode=pinned]}`, `!{CryptoRand[scope=identity]}`. Empty-parameter forms (`!{Rand}`) desugar to documented defaults.
2. **Replay contract taxonomy** — every mode maps to one of {deterministic, re-sampleable, opaque}. The runtime's replay engine reads the contract label and dispatches to the right strategy (pin input, redraw, substitute).
3. **M-ENTROPY integration** — entropy envelopes can declare allowed modes per axis. Compiler rejects module-level mode usage outside the envelope.

### Architecture

**Components:**

1. **AST + parser** — new `EffectParam` node; `!{E[k=v, ...]}` grammar.
2. **Effect row algebra** — parameters participate in unification with explicit rules (likely invariant on parameters; mode-erasure only via alias).
3. **Replay contract registry** — `internal/replay/contracts.go` — each (effect, mode) pair registers its contract label and replay strategy.
4. **Capability scoping** — `scope=` parameter maps to a capability-refinement grammar.
5. **M-ENTROPY hook** — entropy envelope validator cross-checks source mode usage against envelope constraints.
6. **Back-compat shim** — bare `!{E}` tokens desugar to `!{E[mode=default]}` for each effect's documented default.

### Surface syntax (proposed, subject to Design Freeze)

```ailang
-- Current (stays valid via back-compat shim):
func get_random() -> int ! {Rand} = ...

-- Parameterised, modal:
func simulate(seed: int) -> float ! {Rand[mode=seeded]} = ...
func timestamp() -> int ! {Clock[mode=wall]} = ...
func pinned_time() -> int ! {Clock[mode=pinned]} = ...
func mint_key() -> string ! {Rand[mode=crypto, scope=identity]} = ...

-- Aliases (shipped for migration):
type CryptoRand = Rand[mode=crypto]
```

### Replay contract taxonomy

Each (effect, mode) pair declares its contract:

| Effect | Mode | Contract | Replay strategy |
|---|---|---|---|
| `Rand` | `seeded` | deterministic | Same seed, same sequence |
| `Rand` | `os` | re-sampleable | Redraw from OS entropy |
| `Rand` | `crypto` | opaque | Substitute from harness (see M-CRYPTORAND) |
| `Clock` | `pinned` | deterministic | `AILANG_SEED` pins time |
| `Clock` | `wall` | re-sampleable | Reread system clock |
| `Net` | `recorded` | deterministic | Replay from recorded trace body |
| `Net` | `live` | re-sampleable | Redispatch the call |
| `FS` | `fixture` | deterministic | Read from sandboxed fixture tree |
| `FS` | `real` | re-sampleable | Reread real disk (or opaque if file contains secrets) |

Contracts are a three-element poset: `deterministic < re-sampleable < opaque` in terms of information preserved. Replay harnesses dispatch on contract; callers reason about trace-persistence based on it.

### Unification (sketch — final rules are a Design Freeze item)

Proposed: parameters are **invariant under unification**. `!{Rand[mode=os]}` does not unify with `!{Rand[mode=seeded]}`. Modes are erased only via explicit aliasing or back-compat shim. This is stricter than subtyping; rationale is to avoid inference surprises in polymorphic code where the contract matters.

### M-ENTROPY integration

Envelope schema (normalised from [rand-determinism-sitrep.md §10](../rand-determinism-sitrep.md)):

```yaml
entropy:
  behavioral:
    effects:
      Rand:
        modes: [seeded, os, crypto]
        default: crypto
      Clock:
        modes: [pinned, wall]
        default: pinned
      Net:
        modes: [recorded]        # live forbidden in this module
      FS:
        modes: [fixture]         # real disk forbidden
    replay:
      deterministic: required
      opaque:        substitute
```

Compiler emits `EntropyViolation` when source uses a mode outside the envelope's declared set for that effect.

### Implementation Plan

**Phase 1: Parser + AST** (~12h)
- [ ] `[k=v, ...]` grammar extension on effect tokens
- [ ] `EffectParam` AST node
- [ ] Lexer tokens: `LBRACKET_EFFECT`, `EQ`, `RBRACKET_EFFECT`
- [ ] Pretty-printer preserves parameter ordering

**Phase 2: Effect row algebra** (~16h)
- [ ] Extend `Effect` type with parameter map
- [ ] Unification rules (invariant on params) with test matrix
- [ ] Row-polymorphism still works across parameter variations
- [ ] Back-compat: bare `!{E}` desugars to `!{E[mode=default_for_E]}`

**Phase 3: Replay contract registry** (~10h)
- [ ] `internal/replay/contracts.go` — (effect, mode) → contract label
- [ ] Register Rand, Clock, Net, FS mode pairs
- [ ] Replay engine dispatches on contract, not effect token

**Phase 4: Capability scoping** (~12h)
- [ ] `scope=` parameter parser
- [ ] Initial scope set: `identity|session|test-denied|_`
- [ ] Capability refinement: `crypto_rand[identity]` is a narrower capability than `crypto_rand[_]`
- [ ] Scope participates in capability-grant checking

**Phase 5: Clock / Net / FS porting** (~16h)
- [ ] `Clock` modes: `pinned`, `wall`
- [ ] `Net` modes: `recorded`, `live`
- [ ] `FS` modes: `fixture`, `real`
- [ ] Back-compat aliases for each
- [ ] Migration test: existing stdlib modules type-check unchanged

**Phase 6: M-ENTROPY integration** (~10h)
- [ ] Envelope schema extension for per-effect modes
- [ ] Layered validator cross-checks source mode against envelope
- [ ] `EntropyViolation` error type extended with (effect, mode) fields
- [ ] Tests: envelope forbids `wall` clock → source using `Clock[mode=wall]` errors

**Phase 7: CryptoRand alias test** (~4h)
- [ ] `type CryptoRand = Rand[mode=crypto]` ships as stdlib alias
- [ ] Verify existing M-CRYPTORAND code using `!{CryptoRand}` continues to work
- [ ] Zero-diff migration test on representative v0.13 program

**Phase 8: Documentation + examples** (~10h)
- [ ] User guide: `docs/docs/guides/parameterised-effects.md`
- [ ] Three worked examples: modal Rand, modal Clock, modal Net
- [ ] Migration guide for v0.13 → v1.0
- [ ] Update teaching prompt (coordinate with `prompt-manager` skill)

**Total: ~90 hours across ~3 sprints**

### Files to Modify/Create

**New files:**
- `internal/replay/contracts.go` — replay contract registry (~240 LOC)
- `internal/capability/scope.go` — scope parameter handling (~180 LOC)
- `docs/docs/guides/parameterised-effects.md` — user guide (~500 LOC)
- `examples/modal_rand.ail` — worked example (~60 LOC)
- `examples/modal_clock.ail` — worked example (~50 LOC)
- `examples/modal_net.ail` — worked example (~80 LOC)

**Modified files:**
- `internal/lexer/token.go` — effect-parameter tokens (~15 LOC)
- `internal/parser/parser.go` — `[k=v, ...]` grammar (~200 LOC)
- `internal/ast/ast.go` — `EffectParam` AST node (~80 LOC)
- `internal/types/effects.go` — parameter-bearing effect rows (~260 LOC)
- `internal/types/unify.go` — parameter unification rules (~180 LOC)
- `internal/entropy/schema.go` — mode constraints per effect (~120 LOC)
- `internal/entropy/layered.go` — mode vs envelope cross-check (~100 LOC)
- `internal/errors/errors.go` — extended `EntropyViolation` (~40 LOC)
- `stdlib/std/rand.ail`, `stdlib/std/clock.ail`, `stdlib/std/net.ail`, `stdlib/std/fs.ail` — aliases (~40 LOC each)

**Grand Total: ~2,385 LOC (new: 1,110; modified: 1,275)**

## Examples

### Example 1: Modal Rand (the M-CRYPTORAND sequel)

```ailang
import std/rand (random_int)
import std/crypto/rand (crypto_random_int)

-- Tier A: seeded for simulation
export func monte_carlo(seed: int) -> float ! {Rand[mode=seeded]} = ...

-- Tier B: OS entropy for general-purpose randomness
export func shuffle[a](xs: [a]) -> [a] ! {Rand[mode=os]} = ...

-- Tier C: cryptographic, opaque, scoped to identity
export func mint_api_key() -> string ! {Rand[mode=crypto, scope=identity]} = ...

-- Alias from M-CRYPTORAND continues to work:
export func mint_session() -> string ! {CryptoRand} = ...   -- same as Rand[mode=crypto]
```

### Example 2: Modal Clock

```ailang
import std/clock (now)

-- Wall clock — re-sampleable on replay
export func timestamp() -> int ! {Clock[mode=wall]} = now()

-- Pinned — deterministic under AILANG_SEED
export func deterministic_now() -> int ! {Clock[mode=pinned]} = now()
```

### Example 3: M-ENTROPY envelope enforcing security

```yaml
---
milestone: M-AUTH-SERVICE
entropy:
  behavioral:
    effects:
      Rand:
        modes: [crypto]         # ONLY crypto allowed
        default: crypto
      Clock:
        modes: [pinned, wall]   # either is fine
---
```

```ailang
-- This passes — uses Rand[mode=crypto]
export func new_user_id() -> string ! {Rand[mode=crypto]} = crypto_random_bytes(16) |> hex_encode

-- This fails at compile time:
export func weak_token() -> string ! {Rand[mode=os]} = random_bytes(16) |> hex_encode
-- Error: EntropyViolation: Rand[mode=os] not permitted in this module
--   Envelope allows: [crypto]
--   Source uses: os (line 17)
--   Fix: use Rand[mode=crypto] or add `os` to envelope modes
```

### Example 4: Modal AI

**Context:** [M-AI-OPENROUTER](../../implemented/v0_16_x/m-ai-openrouter-provider.md) (v0.16.0) shipped runtime-routed AI calls — OpenRouter can dynamically pick a provider per call, falling back through a configured order. The original sprint plan called for `!{AI[Routeable]}` as a type-level marker that gates which functions are allowed to use a routing-capable provider. Without parameterised effects, M-AI-OPENROUTER had to fall back to a runtime `--allow-routing` flag and a `ResolvedRoute` trace payload. M-EFFECT-REFINEMENT delivers the missing type-level dimension.

**Modes for AI:**

| Mode | Meaning | Replay contract |
|---|---|---|
| `mode=fixed` (default) | One configured provider+model. Equivalent to today's bare `!{AI}`. | Deterministic — request hash maps to recorded response. |
| `mode=routeable` | Runtime may pick from a fallback list (OpenRouter's `provider.order`). Resolved model captured in trace. | Pin-to-resolved by default (replay calls `resolved_model` directly); `--reroute` opts back into live routing. |
| `mode=replay-only` | No live calls permitted; response must come from cache/trace. | Hard fail if no cached response. |
| `scope=byok` | Bring-your-own-key path — provider keys flow as scoped capabilities, not ambient config. | Orthogonal to mode; combines, e.g., `AI[mode=routeable, scope=byok]`. |

```ailang
import std/ai (call)

-- Tier A: fixed model (today's default; no syntax change for existing programs)
export func summarize(text: string) -> string ! {AI[mode=fixed]} =
  call("Summarize: " ++ text)
-- Bare !{AI} elaborates to !{AI[mode=fixed]} — back-compat preserved.

-- Tier B: runtime-routed; opts into OpenRouter-style fallback at the type level.
-- Replaces M-AI-OPENROUTER's runtime --allow-routing flag.
export func summarize_routed(text: string) -> string ! {AI[mode=routeable]} =
  call("Summarize: " ++ text)

-- Tier C: deterministic-replay only — useful for offline agent eval.
-- Live calls are a typed error.
export func eval_against_fixture(prompt: string) -> string ! {AI[mode=replay-only]} =
  call(prompt)

-- BYOK: scope-parameterised key path. Combines with any mode.
export func with_byok_key() -> string ! {AI[mode=routeable, scope=byok]} =
  call("...")
```

**Compile-time check:** a function declared `!{AI[mode=fixed]}` cannot call a function whose signature is `!{AI[mode=routeable]}` without explicit widening — this is the type-level expression of the `--allow-routing` runtime gate. The error mirrors the routing rejection in `internal/ai/openai/anthropic/gemini/ollama` Generate methods that M-AI-OPENROUTER M2 already ships at the runtime layer.

**Trace integration:** the existing `EffectEvent.Route *ResolvedRoute` payload (shipped in M-AI-OPENROUTER M3) is the replay-contract data for `AI[mode=routeable]`. No trace-schema change needed when this milestone lands; only the type-level enforcement is new.

**Migration from v0.16.0 runtime gate:**
- Programs that today rely on `--allow-routing` keep working unchanged: bare `!{AI}` elaborates to `!{AI[mode=fixed]}`, which raises a type error if the host has routing flags set. The runtime gate becomes the elaborator's fallback when programs have not yet adopted modes.
- Programs that want to opt into routing add `mode=routeable` to the effect row and drop `--allow-routing` from their invocation.
- The `AIHandlerWithRouting` interface (M-AI-OPENROUTER M3) is reused as-is by the elaborated handlers.

## Success Criteria

- [ ] `!{E[mode=X]}` syntax parses and type-checks
- [ ] Unification rules specified, documented, tested for all effect pairs
- [ ] Back-compat: every existing AILANG program continues to type-check without modification
- [ ] `type CryptoRand = Rand[mode=crypto]` alias verified zero-diff with M-CRYPTORAND programs
- [ ] Clock, Net, FS, AI ported with modes + aliases (`AI[mode=fixed|routeable|replay-only]`, `AI[scope=byok]`)
- [ ] M-AI-OPENROUTER's `--allow-routing` runtime gate is subsumed by `AI[mode=routeable]` type checking; bare `!{AI}` continues to require the runtime flag for back-compat
- [ ] Replay contract registry populated for all (effect, mode) pairs shipped
- [ ] Trace tooling reads contract label and dispatches correctly
- [ ] M-ENTROPY envelope accepts per-effect mode constraints and enforces at compile time
- [ ] `EntropyViolation` error messages include fix hints with concrete mode suggestions
- [ ] Three worked examples + migration guide + updated teaching prompt

## Testing Strategy

**Unit tests:**
- Parser: `!{E[a=1, b=2]}` parses; malformed variations error at the right location
- Effect-row unification: matrix of every supported (E, mode) × (E, mode) pair
- Back-compat shim: bare `!{E}` equals `!{E[mode=default]}` in every context
- Scope parameter: capability-grant logic respects scope narrowing
- Replay dispatch: each (effect, mode) hits the right contract strategy

**Integration tests:**
- End-to-end: program uses `Rand[mode=seeded]` + `Clock[mode=pinned]` → `AILANG_SEED` produces identical outputs across runs
- M-ENTROPY envelope forbidding a mode → compiler rejects; fix it → compiler accepts
- CryptoRand alias: stdlib module exports `type CryptoRand = Rand[mode=crypto]`; existing M-CRYPTORAND examples compile without changes
- Trace: `Rand[mode=os]` replay redraws; `Rand[mode=crypto]` replay requires harness (from M-CRYPTORAND)

**Golden tests:**
- Representative stdlib modules unchanged pre/post migration (diff == no material change)
- Error messages for each `EntropyViolation` shape match expected format

## Deferred Decisions

- **Mode set extensibility.** Whether users can define their own modes (e.g., `Rand[mode=my_custom]`) — agent prototype may propose; human ratifies before v1.0 freeze.
- **Scope grammar extension.** Initial set `identity|session|test-denied|_`; further scopes added as needs emerge. Agent may extend without design-board review as long as they're additive.
- **Unification corner cases.** Polymorphic code with effect-variable row + parameterised effect. Agent may choose conservative default (invariant) with escape via explicit alias; revisit if inference surprises users.
- **Pretty-printing conventions.** Sorted parameters? Original-order? Canonicalisation for diff-friendly traces — agent may pick.
- **Error-message wording.** Must follow "boring errors" style guide per [M-ENTROPY](m-entropy-budgets.md#error-message-style-guide) — agent may resolve.
- **Migration tooling.** `ailang fix --modes` autoupgrade for `!{Rand}` → `!{Rand[mode=os]}` opt-in — agent may build; not required for v1.0 ship.

## Non-Goals

- **Runtime mode inference.** We do not attempt to infer mode from usage; programmer declares it.
- **Removing `!{CryptoRand}` syntax.** M-CRYPTORAND ships as a first-class effect; under this milestone it *additionally* becomes readable as `!{Rand[mode=crypto]}`. Neither form is removed.
- **Replacing M-ENTROPY.** This milestone *extends* M-ENTROPY's schema; it does not subsume semantic/interpretive axes.
- **Automatic migration of all existing AILANG programs.** Back-compat shim ensures no migration is required; explicit migration is a user choice.
- **Parameters beyond `mode` and `scope` in v1.0.** Other parameters (e.g., `budget=N`, `timeout=...`) considered for future work; not in scope here.
- **Cross-effect parameters.** E.g., `!{Net, Clock}[mode=recorded]` applied to multiple effects at once — rejected; each effect declares its own mode.

## Timeline

**Sprint 1** (30h):
- Phase 1: parser + AST (12h)
- Phase 2: effect row algebra (16h)
- Back-compat shim working end-to-end (2h)

**Sprint 2** (30h):
- Phase 3: replay contract registry (10h)
- Phase 4: capability scoping (12h)
- Phase 5 start: Clock porting (8h)

**Sprint 3** (30h):
- Phase 5 finish: Net + FS porting (8h)
- Phase 6: M-ENTROPY integration (10h)
- Phase 7: CryptoRand alias test (4h)
- Phase 8: documentation + examples (8h)

**Total: ~90 hours across 3 sprints**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Effect-row unification rules cause inference surprises in polymorphic stdlib code | High | Design Freeze locks invariant-on-params; stdlib migration test validates no regressions |
| Back-compat shim leaks mode information in error messages (ugly defaults) | Med | Pretty-printer suppresses default modes in diagnostics |
| M-ENTROPY schema change conflicts with in-flight entropy-budgets work | High | Co-design with M-ENTROPY owner; land envelope extension first, enforcement second |
| CryptoRand alias diverges from M-CRYPTORAND replay semantics | High | Phase 7 zero-diff test is a hard gate |
| Users build brittle code that depends on specific mode strings | Med | Teaching prompt warns; mode set is closed by default (extensions require design review) |
| Staging: trying to ship Rand + Clock + Net + FS simultaneously blows timeline | High | Rand-first ships as v0.14 or v0.15 milestone; Clock/Net/FS follow-on milestones if bandwidth requires |
| Scope parameters overlap with existing capability-budget system | Med | Explicit coordination with M-CAPABILITY-BUDGETS; scopes are typed narrowing, budgets remain numeric |

## Related Documents

**Commissioning memo:**
- [design_docs/planned/rand-determinism-sitrep.md](../rand-determinism-sitrep.md) — §8 parameterised-effect insight; this doc generalises that insight

**Pilot milestone:**
- [design_docs/planned/v0_13_0/m-cryptorand.md](../v0_13_0/m-cryptorand.md) — ships `!{CryptoRand}` as standalone; this milestone subsumes it via alias

**Integrates with:**
- [design_docs/planned/v1_0_0/m-entropy-budgets.md](m-entropy-budgets.md) — envelope schema extension; compile-time enforcement

**Builds on:**
- [design_docs/implemented/v0_6_2/m-capability-budgets.md](../../implemented/v0_6_2/m-capability-budgets.md) — capability surface that `scope=` parameters refine
- [design_docs/implemented/v0_6_0/semantic-caching-complete.md](../../implemented/v0_6_0/semantic-caching-complete.md) — deterministic caching patterns informing replay-contract design

## References

- [Design Axioms](/docs/references/axioms) — A1, A2, A3, A4 compliance
- [Philosophical Foundations](/docs/references/philosophical-foundations) — declared non-determinism, replay-as-function-of-trace
- Plotkin & Pretnar (2009), "Handlers of Algebraic Effects" — effect-handler theory that parameterised effects extend
- Leijen (2017), "Type Directed Compilation of Row-Typed Algebraic Effects" — row-polymorphism background
- Koka and Eff languages — parameterised effect prior art

## Future Work

- **v1.1+:** Additional effect parameters (`budget=`, `timeout=`, `retries=`) unified under the same grammar
- **v1.2+:** User-defined modes with registration via effect declaration
- **v1.3+:** Mode inference hints (proposals, not inference) — compiler suggests `mode=` when it can prove which one fits
- **Beyond:** Parameterised effects as the substrate for M-AGENT-ORCHESTRATION permissions; M-CSP session types integration

---

**Document created**: 2026-04-15
**Last updated**: 2026-04-17
