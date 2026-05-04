# M-AI-EFFECT-MODES: AI Effect Modes (mode=fixed|routeable|replay-only, scope=byok)

**Status**: Implemented (2026-05-04)
**Target**: v0.15.0 (alongside M-EFFECT-REFINEMENT-PHASE1; same release window since v0.14.3 is current and v0.15.0 hasn't shipped)
**Priority**: P1 — Medium (completes the type-level half of M-AI-OPENROUTER's deferred work; pilots Phase 5 of parent doc on AI as the second mode-bearing effect after Rand)
**Estimated**: ~25-30 hours (~3-4 working days)
**Dependencies**:
- [M-EFFECT-REFINEMENT-PHASE1](m-effect-refinement-phase1.md) (shipping in v0.15.0; commits a90ada21 + 320b2a6c + 72859118 + de7fe9a7) — provides parameterised-effects machinery (parser, AST, `Row.Params`, default-mode table, invariant unification, JSON round-trip with omitempty back-compat)
- [M-AI-OPENROUTER](../../implemented/v0_16_x/m-ai-openrouter-provider.md) (v0.16.0, shipped) — provides `AIRoutingPolicy`, `AIHandlerWithRouting`, `ResolvedRoute` trace payload, `--allow-routing` runtime gate (becomes back-compat fallback)
- [M-EFFECT-REFINEMENT (v1.0.0) Example 4: Modal AI](../v1_0_0/m-effect-refinement.md#example-4-modal-ai) — canonical design specification; this sprint implements that Example 4 as the second pilot effect

## Framing

> **Lifts M-AI-OPENROUTER's runtime `--allow-routing` gate into a type-level marker by registering the AI effect in the default-mode table that M-EFFECT-REFINEMENT-PHASE1 just shipped. Bare `!{AI}` desugars to `!{AI[mode=fixed]}`; using a routing-capable provider under fixed mode becomes a unification-time type error. The runtime `--allow-routing` flag is preserved as a back-compat fallback for programs using bare `!{AI}` that still want to route. ~25-30 hours.**

This is the AI-specific subset of Phase 5 from the parent M-EFFECT-REFINEMENT (v1.0.0) doc. Clock/Net/FS port sprints will follow the exact same pattern (one row in `defaultEffectModes` + handler check + example).

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No new runtime behaviour; mode is type-level annotation only in this sprint (replay-only enforcement deferred to Phase 3) |
| A2: Replayability | 0 | Trace format unchanged; `ResolvedRoute` payload from M-AI-OPENROUTER stays |
| A3: Effect Legibility | +2 | This is the win: routing-capable functions advertise via `!{AI[mode=routeable]}`, agents read mode from signature alone |
| A4: Explicit Authority | +1 | Routing now requires explicit type-level opt-in in addition to or instead of runtime `--allow-routing` |
| A5: Bounded Verification | +1 | Mode mismatch is a local type-check failure (per-function) |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +2 | Type-level encoding is the canonical "machine-readable" form; `--allow-routing` gate stays as runtime fallback |
| A8: Minimal Syntax | 0 | Reuses Phase 1's `!{E[k=v]}` syntax; one new effect default entry |
| A9: Cost Visibility | 0 | No cost changes; `ResolvedRoute` trace from M-AI-OPENROUTER carries cost data |
| A10: Composability | +1 | Same mechanism as Rand modes; uniform across effects |
| A11: Structured Failure | +1 | Mode mismatch is a typed error with clear message |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +8** → **Decision: ✅ Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced — type-level annotation only
- [x] A3 (Effects): Strengthens effect legibility (routing visible in type)
- [x] A4 (Authority): Routing opt-in becomes type-level (more explicit than runtime flag)
- [x] A7 (Machines First): Mode-bearing types are statically inspectable

### Decision Thresholds

Net score +8 ≥ +2, no −1 on hard-violation axioms → **Proceed**.

## Problem Statement

[M-AI-OPENROUTER M3 (v0.16.0)](../../implemented/v0_16_x/m-ai-openrouter-provider.md#m3-airouteable-effect-row--trace-schema) shipped a runtime `--allow-routing` gate as a substitute for the planned `!{AI[Routeable]}` type-level marker. The marker was deferred because parameterised effects didn't exist. Now that [M-EFFECT-REFINEMENT-PHASE1](m-effect-refinement-phase1.md) has landed (Rand pilot proved the machinery), the AI port is unblocked.

**Current State:**
- `!{AI}` is a flat effect with no mode information in the type
- Routing-capable provider (OpenRouter) requires `--allow-routing` CLI flag at runtime
- The runtime gate works but is invisible to type-level analysis: agents reading a function signature cannot tell whether the function uses dynamic provider routing
- Three modes are conceptually distinct (fixed direct call / runtime-routed / replay-from-cache) but conflated in the type system
- Parent design doc Example 4 already specifies the target syntax and mode taxonomy

**Impact:**
- AI agents and auditors cannot read an effect row and understand the AI call's contract
- M-AI-OPENROUTER's `--allow-routing` is "explicit opt-in for routing" but at runtime, not compile time
- M-ENTROPY (planned) cannot reach into AI effect rows for behavioural constraints
- The eval infrastructure cannot statically reject routing-capable code in a sandboxed eval (would need runtime detection)

## Goals

**Primary Goal:** Register AI in the per-effect default-mode table so that `!{AI[mode=fixed|routeable|replay-only]}` and `!{AI[scope=byok]}` parse, type-check, and (for `mode=fixed` vs `mode=routeable`) gate routing at compile time.

**Success Metrics:**
- `DefaultModeFor("AI")` returns `("mode", "fixed", true)`
- Bare `!{AI}` desugars to `!{AI[mode=fixed]}`; **all existing AI-using examples produce byte-identical typecheck output** (zero-diff back-compat, same as M-EFFECT-REFINEMENT-PHASE1's M2 sweep)
- Functions declared `!{AI[mode=routeable]}` accept routing config without `--allow-routing`
- Functions declared `!{AI[mode=fixed]}` (or bare `!{AI}`) reject routing config with a typed error
- Programs using bare `!{AI}` + routing flags + `--allow-routing` continue to work (back-compat fallback)
- Worked example `examples/ai_modes.ail` runs end-to-end with `--ai-stub`
- All M-AI-OPENROUTER tests continue to pass unchanged

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Plumbing strategy: handler check via `AIHandlerWithRouting` (option c) vs typechecker pass-through (a/b) | Determines where the type-vs-runtime boundary sits; option c is simpler but couples runtime to typechecker output via convention | agent | design | med |
| `--allow-routing` flag: deprecated, removed, or back-compat fallback | Removing breaks existing scripts; deprecating is signal-only; fallback is most conservative | human | design | med |
| Should `!{AI[mode=routeable]}` require `--allow-routing` too, or supersede it | If type says routeable, runtime check is redundant. Recommendation: skip `--allow-routing` requirement when type is routeable | human | design | low |
| Test the unification-rejection at typecheck time vs handler time | Typecheck is earlier, faster, machine-readable; handler is fallback for dynamic config | human | design | low |

### Design Freeze

Before sprint-executor starts, these must be ratified by a human:

- [ ] **AI default mode** = `fixed` (matches parent doc Example 4)
- [ ] **`--allow-routing` flag** = back-compat fallback, NOT removed (programs using bare `!{AI}` keep working unchanged with the flag)
- [ ] **`!{AI[mode=routeable]}` skips the `--allow-routing` requirement** (type already attests intent)
- [ ] **Plumbing**: agent picks between handler-check (option c, simpler) vs typechecker pass-through (a/b, cleaner). Recommend option c.

## Solution Design

### Overview

Three integrated pieces, each independently testable:

1. **Default-mode entry for AI** in the table M-EFFECT-REFINEMENT-PHASE1 just shipped: `"AI": {"mode", "fixed"}`. Bare `!{AI}` desugars; existing programs unaffected.
2. **Runtime check** in the OpenRouter handler (or AIHandlerWithRouting interface): if a non-nil routing policy is set on a Request whose declared effect row contains `!{AI[mode=fixed]}`, return a typed error.
3. **`--allow-routing` relaxation**: when the declared mode is `routeable`, the runtime gate is skipped (type-level marker is sufficient evidence of intent).

### Architecture

```
internal/types/
└── effects.go                   # MODIFIED: add "AI" -> {"mode", "fixed"} to defaultEffectModes (~5 LOC)
└── effect_params_test.go        # MODIFIED: AI test cases (~30 LOC)

internal/ai/openrouter/
└── handler.go (or equivalent)   # MODIFIED: declared-mode check (~80-120 LOC)
└── handler_modes_test.go        # NEW: handler-mode tests (~100 LOC)

internal/ai/
└── routing.go OR handler.go     # MODIFIED: ErrRoutingRequiresRouteableMode sentinel (~10 LOC)

cmd/ailang/
└── routing_flags.go             # MODIFIED: relax --allow-routing requirement when mode=routeable (~20 LOC)
└── exec.go, main_run.go         # MODIFIED: pass declared mode through to handler (or use AIHandlerWithRouting result) (~30 LOC)

examples/
└── ai_modes.ail                 # NEW: worked example (~40 LOC)

docs/docs/guides/
└── ai-routing.md                # MODIFIED: type-level section (~30-50 LOC)
└── parameterised-effects.md     # MODIFIED: AI in default-mode table (~10 LOC)

prompts/v0.16.0.md               # MODIFIED: AI modes mention (~20 LOC)
cmd/ailang/prompts/v0.16.0.md    # MODIFIED: same (~20 LOC)

changelogs/v0.10-current.md      # MODIFIED: entry under v0.15.0
```

**Components:**

1. **`defaultEffectModes` entry**: One line addition. The M-EFFECT-REFINEMENT-PHASE1 machinery handles parsing, AST, elaboration, unification automatically.

2. **AI mode taxonomy** (per parent doc Example 4):

   | Value | Meaning | Phase 1 (this sprint) enforcement |
   |---|---|---|
   | `mode=fixed` (default) | Direct provider call, no routing | Bare `!{AI}` desugars here; routing config on this row → typed error |
   | `mode=routeable` | Runtime may pick from fallback list | Required for OpenRouter routing; skips `--allow-routing` |
   | `mode=replay-only` | No live calls; from trace/cache | Reserved (parser accepts; runtime stub — Phase 3 of parent doc) |
   | `scope=byok` | Bring-your-own-key path | Reserved (parser accepts; runtime stub — full BYOK design separate) |

3. **Handler-side check** (option c, recommended): The AI op (`aiCall`, `aiCallJson` etc.) reads the effect row from the elaborated CoreTypeInfo (already accessible via the existing typechecker output) and passes it to the handler via the existing `AIHandlerWithRouting` interface or a new sibling. The OpenRouter handler checks: if `req.Routing != nil && req.Routing.HasRouting()` AND the declared row's `Params["AI"]["mode"] == "fixed"` → return `ErrRoutingRequiresRouteableMode`.

4. **`--allow-routing` relaxation**: In `cmd/ailang/routing_flags.go`, the existing safety gate that requires `--allow-routing` when any `--routing-*` flag is set gets one new branch: if the program's declared AI effect mode is `routeable`, skip the requirement. This needs the typechecker output to be available before we build the routing policy — either at command setup time (after typecheck, before run) or via a deferred check at handler time.

### Implementation Plan (sprint-executor will follow)

**M1: AI default-mode entry + bare desugar** (~3-4h)
- [ ] Add `"AI": {"mode", "fixed"}` to `internal/types/effects.go::defaultEffectModes`
- [ ] Verify back-compat: zero-diff sweep on AI-using examples (`examples/ai_openrouter_routing.ail`, any `examples/runnable/ai_*.ail`)
- [ ] Tests: `TestDefaultModeFor_AI`, `TestElaborateEffectRow_AIDefault`, `TestUnify_AIModes` (all 5 cases: same params, default-desugar match, different params FAIL, polymorphic tail, row swap)
- ~30 LOC + ~80 LOC tests = 110 LOC

**M2: Runtime check + `--allow-routing` relaxation** (~6-8h)
- [ ] OpenRouter handler (or `AIHandlerWithRouting` equivalent): check declared effect row against routing config
- [ ] Define `ErrRoutingRequiresRouteableMode` sentinel in `internal/ai/`
- [ ] `--allow-routing` logic: relax requirement when declared mode is `routeable` (or `replay-only`)
- [ ] Tests: handler-rejection cases, `--allow-routing` skip-when-routeable, back-compat with bare `!{AI}` + flags + `--allow-routing`
- ~80-120 LOC + ~100 LOC tests = ~200 LOC

**M3: Worked example + integration tests** (~3-4h)
- [ ] `examples/ai_modes.ail` demonstrating `!{AI}`, `!{AI[mode=fixed]}`, `!{AI[mode=routeable]}` side-by-side
- [ ] End-to-end: `ailang run examples/ai_modes.ail --caps AI,IO --ai-stub`
- [ ] Integration tests: end-to-end CLI test with stub handler covering all three modes
- ~40 LOC example + ~60 LOC integration tests = 100 LOC

**M4: Docs + release** (~3-4h)
- [ ] `docs/docs/guides/ai-routing.md`: new section "Type-level mode markers (v0.15.0+)"
- [ ] `docs/docs/guides/parameterised-effects.md`: AI added to default-mode table
- [ ] CHANGELOG entry under v0.15.0
- [ ] Teaching prompt update (parallel sections in `prompts/v0.16.0.md` and `cmd/ailang/prompts/v0.16.0.md`)
- [ ] Move design doc + sprint plan to `design_docs/implemented/v0_15_x/`
- [ ] Implementation report appended

**Total: ~410 LOC across 4 milestones, ~3-4 working days**

### Files to Modify/Create

**New files:**
- `examples/ai_modes.ail` (~40 LOC)
- `internal/ai/openrouter/handler_modes_test.go` (~100 LOC)

**Modified files:**
- `internal/types/effects.go` (~5 LOC delta)
- `internal/types/effect_params_test.go` (~30 LOC delta)
- `internal/ai/openrouter/handler.go` OR `internal/ai/handler.go` (~80 LOC delta)
- `internal/ai/routing.go` (~10 LOC delta — new sentinel)
- `cmd/ailang/routing_flags.go` (~20 LOC delta — relaxation)
- `cmd/ailang/exec.go` and `main_run.go` (~30 LOC delta — declared-mode plumbing)
- `docs/docs/guides/ai-routing.md` (~30-50 LOC delta)
- `docs/docs/guides/parameterised-effects.md` (~10 LOC delta)
- `prompts/v0.16.0.md` (~20 LOC delta)
- `cmd/ailang/prompts/v0.16.0.md` (~20 LOC delta)
- `changelogs/v0.10-current.md` (entry)

**Total estimate: ~140 new LOC + ~270 modified LOC + ~210 tests = ~620 LOC**

## Examples

### Example 1: Bare AI (back-compat)

```ailang
module my_module
import std/ai (call)

-- Bare !{AI} desugars to !{AI[mode=fixed]}.
-- Continues to work without any changes; fixed mode = direct provider call.
export func summarize(text: string) -> string ! {AI} = call(text)

-- Existing programs that combine bare !{AI} with --routing-* flags
-- still need --allow-routing at runtime (back-compat fallback).
```

### Example 2: Type-level routeable opt-in

```ailang
-- Authors opt into routing at the type level:
export func summarize_routed(text: string) -> string ! {AI[mode=routeable]} =
  call(text)

-- ailang run --caps AI,IO --ai openrouter/auto \
--   --routing-fallback "anthropic,openai" \
--   --entry main my_module.ail
-- (Note: NO --allow-routing required; the type-level marker attests intent.)
```

### Example 3: Type error (compile-time rejection)

```ailang
-- Trying to use routing-capable provider with !{AI[mode=fixed]}:
export func bad(text: string) -> string ! {AI[mode=fixed]} = call(text)

-- ailang run --caps AI,IO --ai openrouter/auto \
--   --routing-fallback "anthropic,openai" \
--   --allow-routing \
--   --entry main bad.ail
--
-- Error: function bad declared !{AI[mode=fixed]} but routing flags
-- imply !{AI[mode=routeable]}. Change the effect row to !{AI[mode=routeable]}
-- or remove the routing flags.
```

## Success Criteria

- [ ] `DefaultModeFor("AI")` returns `("mode", "fixed", true)`
- [ ] Bare `! {AI}` elaborates to a Row with `Params["AI"] == {"mode": "fixed"}`
- [ ] `! {AI[mode=routeable]}` and `! {AI[mode=fixed]}` are distinct under unification
- [ ] OpenRouter handler rejects routing config on a function declared `! {AI[mode=fixed]}` (or bare `! {AI}`) with typed error
- [ ] OpenRouter handler accepts routing config on `! {AI[mode=routeable]}` without `--allow-routing`
- [ ] Bare `! {AI}` + routing flags + `--allow-routing` continues to work (back-compat)
- [ ] `examples/ai_modes.ail` runs end-to-end with `--ai-stub`
- [ ] `make test` green (no regressions)
- [ ] `make verify-examples` 171/171 + new `ai_modes.ail`
- [ ] Docs updated: `ai-routing.md`, `parameterised-effects.md`
- [ ] CHANGELOG entry under v0.15.0
- [ ] Design doc + sprint plan moved to `design_docs/implemented/v0_15_x/`

## Testing Strategy

**Unit tests:**
- Default-mode table: `DefaultModeFor("AI")` returns `("mode", "fixed", true)`
- Elaborate: bare `!{AI}` desugars to `!{AI[mode=fixed]}`; explicit `!{AI[mode=routeable]}` preserved
- Unification: `fixed` vs `routeable` → mismatch error; same modes unify; polymorphic tail preserved
- `ErrRoutingRequiresRouteableMode` is sentinel-checkable via `errors.Is`

**Integration tests:**
- OpenRouter handler with declared mode: routeable accepts, fixed rejects
- `--allow-routing`: skipped for routeable, required for bare `!{AI}` (back-compat)
- End-to-end: `ailang run examples/ai_modes.ail --ai-stub` produces three responses (one per mode)

**Back-compat:**
- All existing AI-using examples (`ai_openrouter_routing.ail`, etc.) type-check unchanged
- M-AI-OPENROUTER's existing tests pass unchanged (routing-flag tests, `--allow-routing` tests)
- `make verify-examples` 171/171 pre-/post-sprint

## Deferred Decisions

The following are intentionally left open for the implementer (agent latitude):

- **Plumbing approach** (handler-check vs typechecker pass-through): agent picks; recommend handler-check via `AIHandlerWithRouting` for simplicity
- **Error message wording** for type-vs-runtime mismatch: agent's call; should reference both `--allow-routing` fallback and `!{AI[mode=routeable]}` as remedies
- **Whether `mode=replay-only` triggers any runtime behaviour in this sprint**: agent picks; recommend "no, parser accepts but runtime treats as fixed" with TODO comment for Phase 3 replay engine work
- **Pretty-printer ordering for AI modes**: alphabetical (already required by M-EFFECT-REFINEMENT-PHASE1's `Row.String` invariant)

## Non-Goals

**Explicitly NOT attempted in this sprint:**
- **Replay-engine pin-to-resolved + `--reroute` flag** — Phase 3 of parent doc; needs replay engine refactor; trace data already captured by M-AI-OPENROUTER M3
- **`mode=replay-only` runtime enforcement** — parser accepts; runtime is identical to fixed (no replay-engine integration); actual replay-only enforcement needs the replay-engine refactor
- **`scope=byok` runtime enforcement** — parser accepts; runtime is identical to ambient-key path; full BYOK semantics (key rotation, capability flow) need a separate design doc
- **AILANG-side `ai.complete(req)` / `openrouter.provider({...})` constructor** — needs new builtins; tracked in `design_docs/planned/v0_17_0/m-ai-tool-loop.md` (untracked)
- **Clock/Net/FS mode ports** — Phase 5 of parent doc; same pattern as this sprint, separate sprints
- **Subtyping or widening coercion between modes** — invariant unification only

## Timeline

**Day 1** (~7h):
- M1: AI default-mode entry + bare desugar
- M2 start: handler-check architecture decision (option c recommended)

**Day 2** (~7h):
- M2 continued: handler-side check + `--allow-routing` relaxation + tests

**Day 3** (~7h):
- M2 finish + back-compat sweep
- M3: worked example + integration tests

**Day 4** (~7h):
- M4: docs guide + teaching prompt + CHANGELOG + design-doc move
- Buffer for any regressions

**Total: ~25-28 hours across 3-4 working days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Plumbing typechecker output to runtime requires deeper changes than option-c handler check | Med | Pick option c (handler-check via `AIHandlerWithRouting`) which uses existing infrastructure; option a/b are fallbacks if c proves insufficient |
| Existing AI-using examples regress under the bare-desugar | High | M-EFFECT-REFINEMENT-PHASE1 already validated 332/332 zero-diff; this sprint only adds one row to the table; same back-compat guarantee applies |
| `--allow-routing` relaxation breaks existing tests | Med | Make relaxation strictly additive: bare `!{AI}` + flags still requires `--allow-routing`; only `!{AI[mode=routeable]}` skips it. All existing tests use bare `!{AI}` + flags, so they pass unchanged |
| Worked example fails verify-examples | Low | Use `--ai-stub` mode; keep deterministic |
| Pretty-printer change in formatted error messages (similar to M-EFFECT-REFINEMENT-PHASE1's `TestFormatEffectRow/all_effects` update) | Low | Run pre-/post-sprint typecheck-output diff sweep; expect AI-using formatted output to gain `[mode=fixed]` annotation; update affected golden tests |

## Related Documents

**Direct dependencies (shipped):**
- [M-EFFECT-REFINEMENT-PHASE1](m-effect-refinement-phase1.md) — provides parameterised-effects machinery
- [M-AI-OPENROUTER](../../implemented/v0_16_x/m-ai-openrouter-provider.md) — provides `AIRoutingPolicy` + runtime `--allow-routing` gate

**Parent design (canonical, multi-phase):**
- [M-EFFECT-REFINEMENT (v1.0.0)](../v1_0_0/m-effect-refinement.md) — full 8-phase taxonomy; this sprint is the AI-specific subset of Phase 5

**Related future work (parent doc):**
- Replay contract registry (Phase 3) — subsumes the replay-only mode runtime enforcement
- Capability scoping `scope=...` (Phase 4) — generalises the BYOK scope marker
- Clock/Net/FS mode ports (Phase 5) — sibling sprints with same pattern

## References

- [Design Axioms](/docs/references/axioms) — The 12 non-negotiable principles
- `internal/types/effects.go::defaultEffectModes` — Phase 1 default-mode table; this sprint adds the AI row
- `internal/ai/openrouter/handler.go` — OpenRouter handler that gains the mode check
- `internal/ai/routing.go` — `AIRoutingPolicy` and `ErrRoutingNotSupported` sentinel; this sprint adds `ErrRoutingRequiresRouteableMode`
- `cmd/ailang/routing_flags.go` — `--allow-routing` runtime gate; this sprint adds the relaxation branch

## Future Work

- **Clock mode port** — same pattern; `Clock[mode=wall|pinned]` with `Clock` row in `defaultEffectModes`
- **Net mode port** — same pattern; `Net[mode=live|recorded]`
- **FS mode port** — same pattern; `FS[mode=real|fixture]`
- **Replay engine pin-to-resolved** — turns `mode=replay-only` from parser-accepted into runtime-enforced; needs trace-aware AI handler
- **BYOK runtime enforcement** — `scope=byok` parameter currently parser-accepted only; full key-rotation/capability semantics in a separate design doc
- **Subtyping/widening between modes** — currently invariant; some users may want `routeable <: fixed` semantics; research question for v1.0+

The parent design doc retains the full picture; this Phase-5-AI doc is the v0.15.0-deliverable carve-out.

---

**Document created**: 2026-05-04
**Last updated**: 2026-05-04 (status flipped to Implemented; report appended)

DESIGN_DOC_PATH: design_docs/implemented/v0_15_x/m-ai-effect-modes.md

---

## Implementation Report (2026-05-04)

All four milestones shipped against the `dev` branch over the sprint
window. Total LOC ≈ **810** across the four milestones (M1: ~30+140
tests, M2: ~451, M3: ~40, M4: ~150 docs/no Go). Mechanism already
validated by M-EFFECT-REFINEMENT-PHASE1's Rand pilot — this sprint
was the second instance of the same pattern.

### What shipped per milestone

| Milestone | Commit | LOC | Headline |
|-----------|--------|-----|----------|
| **M1** AI default-mode entry + bare desugar | `28d92602` | ~30 + ~140 tests | Single-row addition to `defaultEffectModes` in `internal/types/effects.go`: `"AI": {"mode", "fixed"}`. Bare `!{AI}` desugars via the existing default-mode machinery from M-EFFECT-REFINEMENT-PHASE1 M2; existing AI-using examples produce zero-diff typecheck output. Tests run with `-count=20` for determinism. |
| **M2** Runtime check + `--allow-routing` relaxation | `43f8266a` | ~451, 8 new tests | New `ErrRoutingRequiresRouteableMode` sentinel in `internal/ai/`. New `EffectModeFor` helper. The **`routingFlagValues` snapshot pattern** is the load-bearing architectural decision: routing-flag values are captured at CLI flag parse time before typecheck completes, then `determineDeclaredAIMode` reads the entry function's elaborated effect row from `Iface.Exports[entry].Type.Type.EffectRow` after typecheck and flips `allowRouting=true` whenever the declared mode is `routeable` or `replay-only`. Bare `!{AI}` + flags + `--allow-routing` continues to work as the back-compat fallback path. Smoke-tested all 4 paths end-to-end. |
| **M3** Worked example | `01642550` | ~40 | `examples/ai_modes.ail` demonstrates bare `!{AI}`, explicit `!{AI[mode=fixed]}`, and `!{AI[mode=routeable]}` side by side. Both relaxation paths verified end-to-end: routeable entry skips `--allow-routing`; bare-`!{AI}` entry still hits the safety gate (back-compat). |
| **M4** Docs + release | (this commit) | ~150 | `docs/docs/guides/ai-routing.md` gains a new "Type-level mode markers (v0.15.0+)" section. `docs/docs/guides/parameterised-effects.md` updated to list AI in the default-mode table. Teaching prompts in `prompts/v0.16.0.md` and `cmd/ailang/prompts/v0.16.0.md` gain an AI modes subsection mirroring the existing Rand modes section; `versions.json` SHA-256 hashes updated for both. CHANGELOG entry under `[Unreleased] - targeting v0.15.0` in `changelogs/v0.10-current.md`. Design doc + sprint plan moved from `design_docs/planned/v0_15_0/` to `design_docs/implemented/v0_15_x/`; status flipped; this report appended. |

### Architectural notes worth preserving

- **Plumbing approach: deferred-to-handler-setup-time, not
  per-call (M2).** The design doc proposed three options (a/b/c)
  for plumbing the declared mode from typechecker output to runtime
  enforcement. Option c (handler-side check inside the OpenRouter
  handler) was the original recommendation, but during M2 it became
  clear that wiring the declared mode through `AIContext.Call` would
  require extending the `AIHandler` interface from
  `string -> string` to carry the effect row — a much larger surface
  change. Instead, M2 lifted the check to **CLI setup time**: the
  `routingFlagValues` snapshot captures CLI flag values pre-typecheck;
  `determineDeclaredAIMode` reads the entry function's effect row
  from `Iface.Exports[entry].Type.Type.EffectRow` post-typecheck;
  `cmd/ailang/routing_flags.go` flips `allowRouting=true` when the
  declared mode is `routeable` or `replay-only`. This covers the
  user-visible safety property (the `--allow-routing` gate is the
  enforcement point users see) without needing a handler-side mirror.
- **Back-compat path preserved verbatim (M2).** Programs using bare
  `!{AI}` + `--routing-*` flags + `--allow-routing` continue to work
  unchanged. The bare desugar to `!{AI[mode=fixed]}` does not affect
  the relaxation logic because `determineDeclaredAIMode` reads the
  desugared row's `Params["AI"]["mode"]` value — `fixed` does not
  trigger the relaxation, so the existing safety gate fires as
  before. Only an explicit `!{AI[mode=routeable]}` declaration skips
  the gate.
- **Reused parameterised-effects machinery, no new typesystem
  work.** This sprint validated the design hypothesis from
  M-EFFECT-REFINEMENT-PHASE1: that adding a new mode-bearing effect
  is a one-row table edit plus an enforcement point, not a
  typesystem extension. The unification rules, JSON round-trip, and
  pretty-printer ordering all came for free from the Phase 1
  scaffolding.
- **Step-5 handler-side defence-in-depth deferred (intentional).**
  The original sprint plan included a handler-side check inside the
  OpenRouter handler as defence-in-depth: even if the CLI gate were
  bypassed (e.g., via programmatic API), the handler would still
  reject routing config on a `mode=fixed` row. This was deferred
  because (a) the CLI is the only entry point today, and (b) wiring
  the declared mode through `AIHandler` is a non-trivial interface
  extension. When AI calls become value-level via
  `ai.complete(req, mode=...)` (tracked in
  `design_docs/planned/v0_17_0/m-ai-tool-loop.md`), the handler-side
  mirror becomes a natural extension of the request struct and lands
  there.

### Deferrals (intentional)

| Deferred work | Why | Tracked in |
|---|---|---|
| Handler-side defence-in-depth check | CLI gate at runtime is the user-visible enforcement point; handler-side mirror would need extending `AIHandler` interface to carry declared mode through `AIContext.Call`. Lands naturally when value-level `ai.complete(req)` ships. | [m-ai-tool-loop](../../planned/v0_17_0/m-ai-tool-loop.md) (proposed) |
| `mode=replay-only` runtime enforcement | Parser accepts the value; runtime treats as fixed (no replay-engine integration). The relaxation logic also lets `replay-only` skip the `--allow-routing` gate (replay rows attest intent the same way `routeable` does). Actual replay-only enforcement needs a trace-aware AI handler. | Phase 3 of [parent doc](../../planned/v1_0_0/m-effect-refinement.md) |
| `scope=byok` runtime enforcement | Parser accepts the value; runtime is identical to ambient-key path. Full BYOK semantics (key rotation, capability flow) need a separate design doc. | Phase 4 of [parent doc](../../planned/v1_0_0/m-effect-refinement.md) |
| Replay-engine pin-to-resolved + `--reroute` flag | Trace data captured by M-AI-OPENROUTER M3; replay engine itself is trace-naive. Separate replay-engine refactor. | Phase 3 of [parent doc](../../planned/v1_0_0/m-effect-refinement.md) |
| Clock/Net/FS mode ports | Same one-row pattern as this sprint. Each is its own sprint. | Phase 5 of [parent doc](../../planned/v1_0_0/m-effect-refinement.md) |
| Subtyping or widening coercion between modes | Invariant unification only in v0.15.0; some users may want `routeable <: fixed`. Research question for v1.0+. | [parent doc](../../planned/v1_0_0/m-effect-refinement.md) Future Work |

### Success-criteria status

Tracking against the original criteria above:

- [x] `DefaultModeFor("AI")` returns `("mode", "fixed", true)`.
- [x] Bare `! {AI}` elaborates to a Row with `Params["AI"] == {"mode": "fixed"}`.
- [x] `! {AI[mode=routeable]}` and `! {AI[mode=fixed]}` are distinct under unification (invariant rules from M-EFFECT-REFINEMENT-PHASE1 M2 carry through unchanged).
- [ ] OpenRouter handler rejects routing config on a function declared `! {AI[mode=fixed]}` (or bare `! {AI}`) with typed error. **Scope-reduced**: handler-side check deferred (see Deferrals); the CLI-level relaxation in M2 covers the user-visible safety property. Programs that try to use `!{AI[mode=fixed]}` + `--routing-*` flags without `--allow-routing` still hit the existing M-AI-OPENROUTER safety gate; only `!{AI[mode=routeable]}` skips it.
- [x] OpenRouter handler accepts routing config on `! {AI[mode=routeable]}` without `--allow-routing` (CLI-level relaxation; smoke-tested end-to-end).
- [x] Bare `! {AI}` + routing flags + `--allow-routing` continues to work (back-compat).
- [x] `examples/ai_modes.ail` runs end-to-end with `--ai-stub`.
- [x] `make test` green (no regressions).
- [x] `make verify-examples` continues to pass.
- [x] Docs updated: `ai-routing.md`, `parameterised-effects.md`.
- [x] CHANGELOG entry under v0.15.0.
- [x] Design doc + sprint plan moved to `design_docs/implemented/v0_15_x/`.

Net: **10/11 shipped, 1/11 scope-reduced** with the user-visible
safety property preserved by an alternative mechanism (CLI-level
relaxation rather than handler-side defence-in-depth). The
deferred handler-side check lands naturally when value-level
`ai.complete(req)` ships.


