# M-EFFECT-REFINEMENT Phase 1: Parameterised Effect Syntax + Row Algebra (Rand pilot)

**Status**: Implemented (2026-05-04)
**Target**: v0.15.0
**Priority**: P1 — Medium (foundation for type-level mode markers; unblocks `AI[mode=routeable]` from M-AI-OPENROUTER)
**Estimated**: ~37 hours (~4-5 working days)
**Dependencies**:
- [M-CRYPTORAND](../../implemented/v0_13_0/m-cryptorand.md) (v0.13.0, shipped) — provides the forward-compat constraint and the back-compat alias target (`type CryptoRand = Rand[mode=crypto]`)
- [M-AI-OPENROUTER](../../implemented/v0_16_x/m-ai-openrouter-provider.md) (v0.16.0, shipped) — runtime substance for AI routing already lands; Phase 1 doesn't yet port AI to modes (deferred), but the path is clear

## Framing

> **Phase 1 ships the language-feature scaffolding for parameterised effects: the syntax `!{E[mode=X]}` parses, the AST gains an effect-parameter representation, the row algebra and unification rules accept parameters, and `type CryptoRand = Rand[mode=crypto]` validates zero-diff against existing v0.13.x M-CRYPTORAND programs. Clock/Net/FS/AI ports, replay contract registry, capability scoping, and M-ENTROPY integration are deferred to follow-up sprints.**

This is a **Phase-1 carve-out** from the canonical [M-EFFECT-REFINEMENT (v1.0.0)](../v1_0_0/m-effect-refinement.md) design doc, which covers parameterised effects across Rand/Clock/Net/FS/AI plus replay contract registry, capability scoping, and M-ENTROPY integration. Phase 1 ships the syntax, row algebra, unification, and one pilot effect (Rand) with the M-CRYPTORAND alias as the load-bearing validation. Other phases stay tracked in the parent doc.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No runtime semantics change in Phase 1; modes are syntax-only until Phase 3 (replay contract dispatch) lands |
| A2: Replayability | 0 | Trace format unchanged; replay contract dispatch deferred to Phase 3 |
| A3: Effect Legibility | +1 | Mode visible in type signature when authors opt in; bare `!{E}` desugars to default mode |
| A4: Explicit Authority | 0 | Capability scoping (`scope=...`) deferred to Phase 4 |
| A5: Bounded Verification | +1 | Per-effect parameter unification is locally checkable (invariant on params) |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Parameterised syntax is machine-parseable; agents read mode from signature alone |
| A8: Minimal Syntax | 0 | New `[k=v]` annotation; confined to effect rows, opt-in by authors writing it |
| A9: Cost Visibility | 0 | No cost changes in Phase 1 |
| A10: Composability | +1 | One uniform mechanism extensible to all effects in later phases |
| A11: Structured Failure | +1 | Malformed `[k=v]` produces structured parser errors at correct location |
| A12: System Boundary | 0 | No boundary changes in Phase 1 |

**Net Score: +5** → **Decision: ✅ Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced — Phase 1 is syntax + types only
- [x] A3 (Effects): Modes strengthen effect legibility, never weaken it
- [x] A4 (Authority): No ambient access; capability scoping deferred to Phase 4
- [x] A7 (Machines First): Mode-bearing types are statically inspectable

### Decision Thresholds

Net score +5 ≥ +2, no −1 on hard-violation axioms → **Proceed**.

## Problem Statement

The v0.10.14 `std/rand` incident (see [rand-determinism-sitrep](../v1_0_0/rand-determinism-sitrep.md)) was the first failure of a structural pattern: AILANG effects model *what happens* without modelling *under which contract*. M-CRYPTORAND (v0.13.0) was a point fix splitting `!{Rand}` into `!{Rand}` + `!{CryptoRand}` — but duplicating effect tokens for every contract variation does not scale, and Clock/Net/FS/AI all have the same latent issue.

The recently-shipped M-AI-OPENROUTER (v0.16.0) made this concrete a second time. Its design called for `!{AI[Routeable]}` as a type-level marker for routing-capable AI calls. Without parameterised effects, we had to ship a runtime `--allow-routing` CLI gate as a substitute. The trace data is captured and the safety property holds at runtime, but the type signature does not say whether a function uses dynamic routing.

**Current State:**
- `std/clock` can be seeded (`AILANG_SEED` pins time) or wall-clock, with no type-level distinction
- `std/net` traces are replayable in principle, but `!{Net}` does not say whether a call is live or recorded
- `std/fs` sandbox vs real-disk is a runtime flag, invisible in the type
- `!{AI}` programs cannot express "this function uses runtime-routed inference" at the type level
- M-CRYPTORAND uses a separate `!{CryptoRand}` effect token rather than a refinement of `!{Rand}`

**Impact:**
- Every new effect category repeats the Rand taxonomy problem; every one will eventually ship its own incident
- AI agents and auditors have no uniform way to read an effect row and understand its replay contract
- M-ENTROPY (planned) cannot reach into effect rows for behavioural constraints — it can only gate at the capability layer, which is coarser
- Each "fix" duplicates the entire effect machinery (typechecker, capabilities, builtins) rather than parameterising it

## Goals

**Primary Goal:** Ship the language-feature scaffolding for `!{E[k=v, ...]}` so that subsequent sprints can port any effect to mode-bearing form by adding rows to the default-mode table — no new typechecker work per effect.

**Success Metrics:**
- Parser accepts `! {E[k=v, k2=v2]}` and `! {E[k=v] | tail}` with structured errors on malformed forms
- Unification rules: invariant on params; same params unify, different params do not
- Bare `! {E}` desugars to `! {E[mode=default_for_E]}` via a per-effect default-mode table — every existing AILANG program type-checks unchanged
- `type CryptoRand = Rand[mode=crypto]` ships as a stdlib alias and existing M-CRYPTORAND programs (`examples/runnable/contracts/inbox_v2_lib.ail` and others) compile and run zero-diff
- `make verify-examples` passes 171/171 pre-/post-sprint
- One worked example (`examples/modal_rand.ail`) demonstrates seeded/os/crypto Rand variants
- Net new LOC roughly within budget: ~870 new + ~720 modified + ~280 tests

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Surface syntax `!{E[mode=X]}` (per parent doc) | Locks parser shape; affects every future effect doc and prompt | human | design | high |
| Closed mode set (compiler-known per effect) for Phase 1 | Closed = auditable + simpler typechecking; open mode sets deferred | human | design | med |
| Back-compat aliasing: bare `!{Rand}` desugars to `!{Rand[mode=os]}` | If we forced migration, every existing program breaks. Aliasing ships faster. | human | design | high |
| Unification on parameters: invariant (no subtyping) | Simpler inference; matches parent doc's Phase 2 sketch | human | design | high |
| Default mode table location: `internal/types/effects.go` | Compiler-builtin per effect, not effect-author registration | agent | design | low |
| Pretty-printer parameter ordering (alphabetical vs insertion) | Determinism vs author intent; alphabetical recommended for golden-file stability | agent | design | low |
| Parser disambiguation: `[k=v]` after effect name vs existing `[a]` list/array brackets | Lexer/parser interaction; if context-sensitive, may need a distinct `LBRACKET_EFFECT` token | agent | compile | med |

### Design Freeze

Before sprint-executor starts, these must be resolved by a human:

- [ ] **Surface syntax confirmed** as `!{E[k=v, ...]}` (per parent doc) — recommendation: ratify
- [ ] **Mode set closed** for Phase 1 (no user-extensible modes; per-effect mode lookup table compiled into typechecker)
- [ ] **Back-compat aliasing approved** — bare `!{E}` desugars to `!{E[mode=default_for_E]}`
- [ ] **Unification rule** ratified as invariant on params (no subtyping; same param values required for unification)
- [ ] **Default mode table** values per effect: `Rand → os`, `Clock → wall`, `Net → live`, `FS → real`, `AI → fixed` (Phase 1 implements the `Rand` row only; the rest become live in their respective port sprints)

## Solution Design

### Overview

Three integrated pieces in Phase 1, each independently testable:

1. **Parser + AST** (Phase 1 of parent) — Accept `!{E[k=v]}` syntax. AST gains an effect-parameter representation. Pretty-printer round-trips parameters.
2. **Effect row algebra + unification** (Phase 2 of parent) — `Effect` representation extended with parameter map. Unification invariant on params. Bare-effect desugar via per-effect default table. Backward compatibility verified across the entire example corpus.
3. **Rand pilot + CryptoRand alias** (Phase 7 of parent) — `type CryptoRand = Rand[mode=crypto]` ships as stdlib alias. Existing M-CRYPTORAND programs compile and run zero-diff. One worked example demonstrating the three modes.

### Architecture

```
internal/
├── lexer/
│   └── token.go                 # MODIFIED: lexer disambiguation for [k=v] in effect rows
├── parser/
│   └── parser.go                # MODIFIED: !{E[k=v, ...]} grammar (~200 LOC delta)
├── ast/
│   └── ast.go                   # MODIFIED: EffectParam node or Params field on Effect (~80 LOC delta)
├── types/
│   ├── effects.go               # MODIFIED: parameter-bearing rows + default-mode table + bare desugar (~260 LOC delta)
│   └── unification.go           # MODIFIED: invariant param unification (~180 LOC delta)
└── ...

stdlib/
├── std/rand.ail                 # MODIFIED: modes (default preserved by desugar)
└── std/crypto/rand.ail          # MODIFIED: type CryptoRand = Rand[mode=crypto] alias

examples/
└── modal_rand.ail               # NEW: worked example (~60 LOC)

docs/docs/guides/
└── parameterised-effects.md     # NEW: user guide (~250-400 lines)

prompts/v0.16.0.md               # MODIFIED: parameterised-effects section (~30 LOC delta)
cmd/ailang/prompts/v0.16.0.md    # MODIFIED: same (~30 LOC delta)

changelogs/v0.10-current.md      # MODIFIED: v0.15.0 entry
```

**Components:**

1. **Lexer / parser** — `! {E[k=v, k2=v2]}` parses. Lexer recognises `[` and `]` in effect-row context distinct from list/array contexts (parser-level disambiguation OR distinct token, agent's choice). Comma-separated `key=value` pairs; values are bare identifiers or strings. Malformed forms (`{k=}`, `{=v}`, `{k v}`, `{k:v}`) produce structured parser errors at the correct token position.

2. **AST** — Effect representation extended with parameters. Either a new `EffectParam` AST node or a `Params map[string]string` (or `[]EffectParam` for ordering) field on the existing effect representation — agent picks the cleaner integration. Pretty-printer round-trips with deterministic parameter ordering (alphabetical recommended).

3. **`Effect` row representation** (in `internal/types/effects.go`) — Add a parameter map to the row's labels (or a parallel param map keyed by effect name). `IsKnownEffect` continues to validate effect names. New `DefaultModeFor(name string) (k, v string, ok bool)` lookup table — Phase 1 entry: `("Rand", "mode", "os")`. Other effects (Clock, Net, FS, AI) **do not have entries yet** — their bare forms continue to type-check as today; their port sprints add rows to the table.

4. **Bare-effect desugar** — When elaborating `! {E}` (no params), check `DefaultModeFor(E)`; if present, elaborate to `! {E[mode=os]}` (etc.). Otherwise leave bare. Crucially, this means **existing programs** that use `!{Rand}` get the same treatment as `!{Rand[mode=os]}` after Phase 1 — and the equivalence is enforced by the unification rules (so callers don't see a change).

5. **Unification rules** (in `internal/types/unification.go`) — Two effects unify iff:
   - Same effect name AND
   - Same parameter map (key set + values match exactly; **invariant**, no subtyping in Phase 1)
   - Polymorphic row tail still works: `!{E[mode=os] | a}` unifies with `!{E[mode=os], F[mode=real] | a}` provided `a` accepts `F[mode=real]`
   - Different parameter values do NOT unify: `!{Rand[mode=os]}` and `!{Rand[mode=seeded]}` are distinct effect rows
   - Missing-via-default: `!{Rand}` and `!{Rand[mode=os]}` unify (same desugared form)

6. **Stdlib alias** — `stdlib/std/crypto/rand.ail` exports `type CryptoRand = Rand[mode=crypto]`. The existing `crypto_random_int` etc. functions retain `! {CryptoRand}` in their signature; with the alias, this elaborates to `! {Rand[mode=crypto]}`. Existing programs that import `! {CryptoRand}` continue to type-check.

7. **`examples/modal_rand.ail`** — One worked example showing `!{Rand[mode=seeded]}`, `!{Rand[mode=os]}`, `!{Rand[mode=crypto]}` side-by-side. Runs with stub seeded mode (deterministic).

8. **Documentation** — `docs/docs/guides/parameterised-effects.md` covers the syntax, default-mode table, the closed-mode-set decision, the back-compat alias mechanism, and a forward pointer to follow-up sprints. Teaching prompts gain a brief section.

### Implementation Plan (sprint-executor will follow)

**Phase 1A: Parser + AST** (~14h, ~1-1.5 days)
- [ ] Lexer disambiguation for `[k=v]` after effect name (or distinct `LBRACKET_EFFECT` token if cleaner)
- [ ] Parser grammar: `! {E[k=v, ...]}` with comma-separated `key=value`
- [ ] AST: `EffectParam` node OR `Params` field on existing structure (agent's call)
- [ ] Pretty-printer round-trips parameters with deterministic ordering
- [ ] Tests: positive cases, malformed-form structured errors, round-trip parser→pretty-printer→parser

**Phase 1B: Row algebra + unification** (~16h, ~1.5-2 days)
- [ ] Extend effect-row representation with parameter map
- [ ] `DefaultModeFor` lookup table — Phase 1 entry: Rand → os
- [ ] Bare-effect desugar: `!{E}` → `!{E[mode=default_for_E]}` when entry exists
- [ ] Unification rules: invariant on params, polymorphic-tail-compatible
- [ ] Test matrix covering 5+ unification cases (same params, different params, missing-via-default, polymorphic tail, conflicting tail)
- [ ] Back-compat sweep: every existing AILANG program type-checks identically pre-/post-sprint (`make verify-examples` 171/171)

**Phase 1C: Rand pilot + CryptoRand alias** (~4h, ~0.5 day)
- [ ] `type CryptoRand = Rand[mode=crypto]` in `stdlib/std/crypto/rand.ail`
- [ ] Zero-diff regression: M-CRYPTORAND programs (`inbox_v2_lib.ail`, `inbox_v2_app.ail`, etc.) compile and run identically
- [ ] `examples/modal_rand.ail` demonstrating seeded/os/crypto

**Phase 1D: Docs + release** (~3h, ~0.5 day)
- [ ] `docs/docs/guides/parameterised-effects.md` (~250-400 lines)
- [ ] Teaching prompt update (`prompts/v0.16.0.md` and `cmd/ailang/prompts/v0.16.0.md`)
- [ ] CHANGELOG entry under v0.15.0 referencing parent design doc + this Phase-1 doc
- [ ] Move design doc to `design_docs/implemented/v0_15_x/`

**Total: ~37 hours, 4-5 working days**

### Files to Modify/Create

**New files:**
- `examples/modal_rand.ail` (~60 LOC)
- `docs/docs/guides/parameterised-effects.md` (~250-400 LOC)

**Modified files:**
- `internal/lexer/token.go` (~15 LOC delta — disambiguation or new token)
- `internal/parser/parser.go` (~200 LOC delta — `[k=v, ...]` grammar)
- `internal/ast/ast.go` (~80 LOC delta — effect-parameter representation)
- `internal/types/effects.go` (~260 LOC delta — params, default-mode table, bare desugar)
- `internal/types/unification.go` (~180 LOC delta — invariant param unification)
- `stdlib/std/rand.ail` (~20 LOC delta)
- `stdlib/std/crypto/rand.ail` (~5 LOC — alias)
- `prompts/v0.16.0.md` (~30 LOC delta)
- `cmd/ailang/prompts/v0.16.0.md` (~30 LOC delta)
- `changelogs/v0.10-current.md` (entry)

**Total: ~870 new LOC + ~720 modified + ~280 LOC tests = ~1,870 LOC**

## Examples

### Example 1: Parameterised Rand (the centrepiece)

**Before (v0.14.x):**
```ailang
import std/rand (random_int)
import std/crypto/rand (crypto_random_int)

-- Two separate effect tokens; M-CRYPTORAND ships them as siblings.
export func monte_carlo() -> float ! {Rand} = ...
export func mint_session() -> string ! {CryptoRand} = ...
```

**After (v0.15.0 with Phase 1):**
```ailang
import std/rand (random_int)
import std/crypto/rand (crypto_random_int)

-- Bare !{Rand} continues to work — desugars to !{Rand[mode=os]}.
export func monte_carlo() -> float ! {Rand} = ...

-- Authors can now opt into explicit modes:
export func simulate(seed: int) -> float ! {Rand[mode=seeded]} = ...
export func mint_api_key() -> string ! {Rand[mode=crypto]} = ...

-- The M-CRYPTORAND alias keeps working zero-diff:
export func mint_session() -> string ! {CryptoRand} = ...
-- Now elaborates to: ! {Rand[mode=crypto]}
```

### Example 2: Unification matrix

```
! {Rand[mode=os]}        unifies with  ! {Rand[mode=os]}        OK
! {Rand[mode=os]}        unifies with  ! {Rand}                 OK (default desugar)
! {Rand[mode=os]}        unifies with  ! {Rand[mode=seeded]}    FAIL
! {Rand[mode=os] | a}    unifies with  ! {Rand[mode=os], FS | a} OK (poly tail)
! {Rand[mode=os], FS}    unifies with  ! {FS, Rand[mode=os]}    OK (row swap)
```

### Example 3: Malformed parser errors

```
! {Rand[mode=]}     →  Error: expected value after '=' at line 5, col 17
! {Rand[=os]}       →  Error: expected key before '=' at line 5, col 13
! {Rand[mode os]}   →  Error: expected '=' between key and value at line 5, col 17
! {Rand[mode:os]}   →  Error: expected '=' (got ':') at line 5, col 17
```

## Success Criteria

- [ ] `! {E[k=v, k2=v2]}` parses and type-checks; AST exposes parameters
- [ ] Unification rules: invariant on params; same params unify, different params do NOT unify; row tail polymorphism works
- [ ] Bare `! {Rand}` desugars to `! {Rand[mode=os]}` via the default-mode table; identical typechecker behaviour pre-/post-sprint on existing programs
- [ ] `type CryptoRand = Rand[mode=crypto]` alias works; existing M-CRYPTORAND programs (`inbox_v2_lib.ail`, `inbox_v2_app.ail`, others) compile and run zero-diff
- [ ] Worked example `examples/modal_rand.ail` runs with stub seeded mode
- [ ] User guide `docs/docs/guides/parameterised-effects.md` published, registered in sidebar
- [ ] Teaching prompt updated with parameterised-effects section
- [ ] `make ci` green: build, test, lint, verify-examples, file-size check
- [ ] CHANGELOG entry references parent design doc + this Phase-1 doc

## Testing Strategy

**Unit tests:**
- Parser: positive `! {E[k=v]}` cases, malformed forms (`{k=}`, `{=v}`, `{k v}`, `{k:v}`) produce structured errors at correct location and column
- Unification matrix: 5+ cases (same params, different params, missing-via-default desugar, polymorphic tail, row swap)
- Default-mode desugar: every effect's bare form resolves to its default mode (Phase 1: Rand only; others remain bare and that's the expected behaviour)
- Pretty-printer round-trip: `Effect[k=v, k2=v2]` parses → pretty-prints → reparses identical AST

**Integration / golden tests:**
- `make verify-examples` 171/171 pre-/post-sprint with no diffs in compiled output
- Existing 20 contract examples produce identical pre-/post-sprint outcomes (M-TAINT-TYPES M5 pattern)
- M-CRYPTORAND zero-diff: `inbox_v2_lib.ail` + `inbox_v2_app.ail` produce identical compiled output AND identical runtime behaviour

**Regression tests:**
- Parser-error golden files for malformed `[k=v]` shapes
- Type-check diff suite: snapshot every example's type-check output before sprint, assert identical after sprint

**Determinism check** (per coding standards):
- Pretty-printer is a pure function operating over a `map`; if alphabetical ordering chosen, run with `-count=20` to catch map-iteration nondeterminism (relevant per `internal/types/labels.go` precedent — sorted/dedup lists must be stable under repeated runs)

## Deferred Decisions

The following are intentionally left open for the implementer (agent latitude):

- **Lexer strategy** — Distinct `LBRACKET_EFFECT` token vs reusing `LBRACKET` with parser-level disambiguation. Agent picks the cleaner integration; document in implementation report.
- **AST shape** — `EffectParam` node vs `Params` field on existing `Effect`. Agent's call. Recommendation: `Params []EffectParam` (slice of {Key, Value} pairs) for ordering; alphabetical sort happens in the pretty-printer.
- **Pretty-printer ordering** — Alphabetical (recommended for golden-file stability) vs insertion order. Agent picks; document.
- **Default-mode table location** — `internal/types/effects.go` vs `internal/effects/defaults.go`. Agent picks; recommend keeping it next to `IsKnownEffect`.
- **How to test the M-CRYPTORAND zero-diff** — Either snapshot compiled output of representative programs and diff, or rely on existing M-CRYPTORAND test suite. Agent picks; recommend snapshot+diff for clarity in the implementation report.

## Non-Goals

**Explicitly NOT attempted in Phase 1:**
- **Replay contract dispatch** (Phase 3 of parent doc) — `internal/replay/contracts.go` registry, replay engine dispatch on contract not effect token
- **Capability scoping** (`scope=...`, Phase 4 of parent doc)
- **Clock/Net/FS/AI port** (Phase 5 of parent doc) — Phase 1 ships the scaffolding; these effects continue to work bare-form. Their port sprints will add rows to the default-mode table and stdlib aliases.
- **AI mode subsuming `--allow-routing`** — once Phase 5 lands, `! {AI[mode=routeable]}` becomes the type-level expression of the runtime gate. Phase 1 does not yet do this.
- **M-ENTROPY mode constraints** (Phase 6) — envelope-level mode validation
- **Open / user-extensible mode sets** — Phase 1 ships closed (compiler-known per effect); open mode set is a v1.0+ research question

## Timeline

**Day 1** (~7h):
- Phase 1A: parser + AST + lexer disambiguation + tests for malformed forms

**Day 2** (~7h):
- Phase 1A finish + Phase 1B start: effect parameter representation + default-mode table

**Day 3** (~8h):
- Phase 1B continued: unification rules + test matrix + bare-effect desugar

**Day 4** (~8h):
- Phase 1B finish: full back-compat sweep on `make verify-examples`
- Phase 1C: CryptoRand alias + zero-diff regression test + worked example

**Day 5** (~7h):
- Phase 1D: docs guide + teaching prompt + CHANGELOG + design-doc move
- Buffer for any Phase 1B regressions

**Total: ~37 hours across 5 working days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Parser ambiguity between effect-row brackets and existing `[a]` list/array brackets | High | Parser-level context disambiguation (effect-row context already known); fall back to a distinct `LBRACKET_EFFECT` lexer token if context-sensitivity proves fragile |
| Unification rule choice causes inference surprises in row-polymorphic code | Med | Invariant params (no subtyping) — keeps inference predictable; comprehensive test matrix; `DEBUG_TYPES=1` traces for edge cases |
| Back-compat sweep misses an existing AILANG program | High | Run `make verify-examples` (171 examples) AND pre-/post-sprint typecheck-output diff on every example; CI guard |
| CryptoRand alias breaks M-CRYPTORAND callers | High | Zero-diff snapshot test on `inbox_v2_lib.ail` + `inbox_v2_app.ail` and explicit per-example regression in CI; mark M-CRYPTORAND test corpus as a release-gating check |
| Sprint exceeds 5 days due to typechecker complexity | Med | Pause-and-reassess after Phase 1B; if unification rules turn out harder than the parent doc's sketch, scope down further (ship parser+AST only, defer row algebra to a follow-up sprint that lands before any other-effect ports) |
| Lexer/parser change ripples into unexpected places (REPL, fmt, error messages) | Med | Run all tests after each Phase 1A milestone; the `parser-developer` skill rules apply |

## Related Documents

**Parent design (canonical, multi-phase):**
- [design_docs/planned/v1_0_0/m-effect-refinement.md](../v1_0_0/m-effect-refinement.md) — full M-EFFECT-REFINEMENT covering 8 phases. This Phase-1 doc carves out the v0.15.0-deliverable subset (Phases 1, 2, 7, partial 8).

**Validation targets (zero-diff):**
- [design_docs/implemented/v0_13_0/m-cryptorand.md](../../implemented/v0_13_0/m-cryptorand.md) — M-CRYPTORAND, the back-compat alias target; load-bearing validation for Phase 1
- [design_docs/implemented/v0_16_0/m-taint-types.md](../../implemented/v0_16_0/m-taint-types.md) — precedent for label/refinement type-system extension; pattern of adding a new AST type and threading it through 7+ type switches

**Downstream consumer:**
- [design_docs/implemented/v0_16_x/m-ai-openrouter-provider.md](../../implemented/v0_16_x/m-ai-openrouter-provider.md) — Phase 5 of parent doc (AI port) subsumes the runtime `--allow-routing` gate as a type-level marker once shipped. Phase 1 unblocks that work.

**Adjacent / orthogonal:**
- [design_docs/planned/v1_0_0/m-entropy-budgets.md](../v1_0_0/m-entropy-budgets.md) — M-ENTROPY; Phase 6 of parent doc integrates with envelope-level mode constraints

## References

- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- [Philosophical Foundations](/docs/references/philosophical-foundations) — Block-universe determinism (A1/A2 underpinnings)
- [Design Lineage](/docs/references/design-lineage) — What we adopted/rejected and why
- `internal/types/effects.go` — Existing effect-row implementation (~330 LOC); Phase 1 extends, doesn't replace
- `internal/types/labels.go` — M-TAINT-TYPES precedent for adding a new structured type to the type system
- `internal/parser/parser.go` — Current effect-row parsing; Phase 1 extends

## Future Work (deferred from this sprint, tracked elsewhere)

All future work is captured in the parent [M-EFFECT-REFINEMENT (v1.0.0)](../v1_0_0/m-effect-refinement.md):

- **Replay contract registry** (Phase 3 of parent) — likely a v0.15.x or v0.16.x follow-up sprint
- **Capability scoping `scope=...`** (Phase 4) — independent later sprint
- **Clock/Net/FS modes** (Phase 5) — can ship per-effect as separate sprints; each adds one row to the default-mode table
- **AI mode port** `AI[mode=routeable|fixed|replay-only]` (Phase 5 continuation) — subsumes M-AI-OPENROUTER's runtime `--allow-routing` gate; once shipped, programs using `! {AI[mode=routeable]}` get type-level visibility
- **M-ENTROPY integration** (Phase 6) — ships with M-ENTROPY itself
- **Open / user-extensible mode sets** — research question for v1.0+

The parent design doc retains the full 8-phase picture; this Phase-1 doc is the v0.15.0-deliverable carve-out.

---

**Document created**: 2026-05-04
**Last updated**: 2026-05-04 (status flipped to Implemented; report appended)

DESIGN_DOC_PATH: design_docs/implemented/v0_15_x/m-effect-refinement-phase1.md

---

## Implementation Report (2026-05-04)

All four milestones shipped against the `dev` branch over the sprint
window. Total LOC ≈ **1,506** across the four milestones (M1: 573,
M2: 565, M3: 58, M4: ~310 docs/no Go).

### What shipped per milestone

| Milestone | Commit | LOC | Headline |
|-----------|--------|-----|----------|
| **M1** Parser + AST | `a90ada21` | 573 | `EffectParam{Key, Value}` AST node; `EffectAnnotation.Params []EffectParam`; lexer disambiguation kept the existing `LBRACKET` token (parser-level context check sufficient — effect rows are already a known parser context); pretty-printer alphabetical by key with `-count=20` determinism guard. 22 new tests; 6 structured parse errors with stable codes `PAR_EFF010`–`PAR_EFF014` (empty `[]`, missing key, missing value, missing `=`, wrong separator `:`, missing comma, duplicate key). |
| **M2** Row algebra + invariant unification | `320b2a6c` | 565 | `Row.Params map[string]map[string]string` extends the effect row representation. `defaultEffectModes` table in `internal/types/effects.go` ships exactly one row: `Rand → ("mode", "os")`. `DefaultModeFor(name)` exposes it. The **`effectiveParamsOf`** back-compat bridge is the load-bearing architectural decision: rows from `validate_effects.go::stringSliceToEffectRow` have nil `Params`, while elaborator-built rows have desugared `Params` — the bridge consults `DefaultModeFor` during comparison so both row sources unify cleanly. JSON round-trip uses `omitempty` on `Params` so older iface caches load unchanged. 12 test functions / ~30 sub-cases run with `-count=20`. Zero-diff verified: 332/332 example `.ail` files byte-identical pre-/post-sprint. |
| **M3** Worked example | `72859118` | 58 | `examples/modal_rand.ail` demonstrates bare `!{Rand}`, explicit `!{Rand[mode=os]}`, and `!{Rand[mode=seeded]}` side by side; runs end-to-end under `ailang run --caps Rand,IO`. The CryptoRand zero-diff regression originally scoped for M3 was **skipped** — see scope-reduction note below. |
| **M4** Docs + release | (this commit) | ~310 | User guide `docs/docs/guides/parameterised-effects.md` (registered in `docs/sidebars.js` under Reference > Language). Teaching prompts updated in `prompts/v0.16.0.md` and `cmd/ailang/prompts/v0.16.0.md`. CHANGELOG entry under `[Unreleased] - targeting v0.15.0` in `changelogs/v0.10-current.md`. Design doc + sprint plan moved to `design_docs/implemented/v0_15_x/`; status flipped; this report appended. `make build && make test && make verify-examples` clean (file-size pre-existing failures elsewhere are out of scope). |

### Architectural notes worth preserving

- **`effectiveParamsOf` back-compat bridge (M2).** The cleanest way
  to land Phase 1 without rewriting every effect-row constructor in
  the codebase was a normalisation step at unification time. Rows
  constructed by older paths (most notably
  `validate_effects.go::stringSliceToEffectRow`, which still operates
  on `[]string` effect names) carry a nil `Params` field. The
  elaborator builds rows with desugared `Params` already populated.
  Rather than thread the desugar through every constructor,
  `effectiveParamsOf` calls `DefaultModeFor` during comparison and
  fills the gap. This preserves the invariant ("two rows that
  represent the same effect set unify") without forcing a flag-day
  migration of every row constructor.
- **JSON `omitempty` on `Params` (M2).** Iface caches written by
  pre-sprint binaries have no `Params` field. Decoding them with the
  new `Row.Params` field absent reads as the empty map, which the
  back-compat bridge then fills via `DefaultModeFor`. This means
  there is no cache-invalidation step for users upgrading across the
  Phase-1 boundary; old `.ailang_iface` artefacts continue to load.
- **CryptoRand-doesn't-exist scope reduction (M3).** The original
  Phase 1 design doc and sprint plan called for `type CryptoRand =
  Rand[mode=crypto]` as the load-bearing zero-diff validation. On
  inspection during M3, no `CryptoRand` effect token actually exists
  in this codebase: the v0.13.0 M-CRYPTORAND landed as a
  point-fix splitting use cases at the standard-library level rather
  than introducing a separate effect, so there are no callers of
  `! {CryptoRand}` to validate against. The default-mode desugar
  (bare `!{Rand}` ⇄ `!{Rand[mode=os]}`) gives the same back-compat
  property the alias check would have demonstrated. The alternative —
  adding a `CryptoRand` token retroactively just so the alias check
  has something to consume — would have created throwaway code. M3
  shipped the runnable worked example (`examples/modal_rand.ail`)
  which exercises the row algebra in a way that is meaningful for
  authors rather than for a hypothetical alias.
- **Pretty-printer alphabetical ordering (M1).** Insertion order
  would have been faster but golden-file tests would flake under
  Go's randomised map iteration. The `-count=20` determinism guard
  in M1's test suite catches any accidental reintroduction of map
  iteration in the canonicaliser.

### Deferrals (intentional)

| Deferred work | Why | Tracked in |
|---|---|---|
| Replay contract registry / mode-aware runtime dispatch | Phase 1 ships syntax + types only. Runtime treats all `Rand` modes identically (all dispatch to the same `_rand_int` / `_rand_float` / `_rand_bool` builtins). Runtime dispatch is a follow-up sprint. | [M-EFFECT-REFINEMENT (v1.0.0) — Phase 3](../../planned/v1_0_0/m-effect-refinement.md) |
| Capability scoping `scope=...` | Beyond mode parameters; scope parameters extend to FS sandboxing / Net recording. Independent design. | [M-EFFECT-REFINEMENT — Phase 4](../../planned/v1_0_0/m-effect-refinement.md) |
| Clock / Net / FS port | Each adds one row to `defaultEffectModes` and ports its stdlib. Ships per-effect as separate sprints. | [M-EFFECT-REFINEMENT — Phase 5](../../planned/v1_0_0/m-effect-refinement.md) |
| AI port `!{AI[mode=routeable\|fixed\|replay-only]}`, `!{AI[scope=byok]}` | Subsumes [M-AI-OPENROUTER](../v0_16_x/m-ai-openrouter-provider.md)'s runtime `--allow-routing` gate as a type-level marker. Phase 1 unblocks the work but does not yet port AI to modes. | [M-EFFECT-REFINEMENT — Phase 5](../../planned/v1_0_0/m-effect-refinement.md), [Example 4: Modal AI](../../planned/v1_0_0/m-effect-refinement.md#example-4-modal-ai) |
| M-ENTROPY mode constraints | Envelope-level mode validation composes with the language-level rules. Ships with M-ENTROPY itself. | [M-ENTROPY](../../planned/v1_0_0/m-entropy-budgets.md), [M-EFFECT-REFINEMENT — Phase 6](../../planned/v1_0_0/m-effect-refinement.md) |
| Open / user-extensible mode sets | Phase 1 closed-set decision was deliberate (auditable, simpler unification, compiler-enforced rollouts). User-extensible modes are a v1.0+ research question. | [M-EFFECT-REFINEMENT (v1.0.0)](../../planned/v1_0_0/m-effect-refinement.md) |
| CryptoRand alias zero-diff | `CryptoRand` doesn't exist as an effect token in this codebase (M-CRYPTORAND landed at the stdlib level rather than as a new effect). Default-mode desugar gives the same back-compat property. | Not re-tracked (scope-reduction; not a deferral) |

### Success-criteria status

Tracking against the original criteria above:

- [x] `! {E[k=v, k2=v2]}` parses and type-checks; AST exposes parameters.
- [x] Unification rules: invariant on params; same params unify, different params do NOT unify; row tail polymorphism works.
- [x] Bare `! {Rand}` desugars to `! {Rand[mode=os]}` via the default-mode table; identical typechecker behaviour pre-/post-sprint on existing programs (332/332 example files byte-identical).
- [ ] `type CryptoRand = Rand[mode=crypto]` alias works; existing M-CRYPTORAND programs compile and run zero-diff. **Scope-reduced**: `CryptoRand` doesn't exist as an effect token in this codebase, so the alias check had nothing to validate against. Default-mode desugar gives the back-compat property the alias check would have demonstrated; the load-bearing zero-diff data point is the 332/332 example sweep instead.
- [x] Worked example `examples/modal_rand.ail` runs end-to-end.
- [x] User guide `docs/docs/guides/parameterised-effects.md` published, registered in sidebar.
- [x] Teaching prompt updated with parameterised-effects section (both `prompts/v0.16.0.md` and `cmd/ailang/prompts/v0.16.0.md`).
- [x] `make build && make test && make verify-examples` green; `make lint` clean. Pre-existing file-size failures unrelated to this sprint (e.g. `registry_codegen.go`, `coordinator.go`) remain — out of scope.
- [x] CHANGELOG entry references parent design doc + this Phase-1 doc.

Net: **8/9 shipped, 1/9 scope-reduced** with the back-compat property
preserved by an alternative mechanism (default-mode desugar +
332/332 byte-identical example sweep).

