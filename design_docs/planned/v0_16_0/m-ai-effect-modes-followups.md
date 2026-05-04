# M-AI-EFFECT-MODES-FOLLOWUPS: Close the Loose Ends from v0.15.0 AI Modes

**Status**: Planned
**Target**: v0.16.0 (or v0.15.x point release, depending on which items ship first)
**Priority**: P2 — Low (the v0.15.0 work covers the load-bearing protections; these are belt-and-suspenders, runtime-stub-to-real conversions, and naturally-deferred items)
**Estimated**: ~20-30 hours total (~3-4 working days) if shipped as a single bundle; smaller pieces can ship independently
**Dependencies**:
- [M-AI-EFFECT-MODES](../../implemented/v0_15_x/m-ai-effect-modes.md) (v0.15.0, shipped) — provides the type-level marker, default-mode entry, and CLI safety-gate relaxation that this doc completes
- [M-AI-OPENROUTER](../../implemented/v0_16_x/m-ai-openrouter-provider.md) (v0.16.0, shipped) — provides `AIRoutingPolicy`, `AIHandlerWithRouting`, `--allow-routing` runtime gate
- [M-EFFECT-REFINEMENT-PHASE1](../../implemented/v0_15_x/m-effect-refinement-phase1.md) (v0.15.0, shipped) — provides parameterised-effects machinery
- [M-AI-TOOL-LOOP](../v0_17_0/m-ai-tool-loop.md) (planned, v0.17.0) — overlapping consumer; if it ships first the AIHandler interface extension is free

## Framing

> **Catalog of follow-up items from M-AI-EFFECT-MODES (v0.15.0). Each item is independently shippable; bundle as time/priority permits. Sprint-evaluator (round 1) flagged the OpenRouter handler-side defence-in-depth check as the only −5 fidelity gap; this doc tracks it plus three smaller stubs that the v0.15.0 doc explicitly deferred.**

The v0.15.0 sprint shipped:
- Type-level invariant unification (M1) — every program through the typechecker is protected
- CLI safety-gate relaxation (M2) — every program through `ailang run` is protected
- Worked example + docs (M3, M4)

This doc enumerates what's left:

| # | Item | Source | Estimated | Blocking? |
|---|------|--------|-----------|-----------|
| 1 | OpenRouter handler-side defence-in-depth check | M-AI-EFFECT-MODES M2 Step 5 (deferred) | ~6-10h | Optional defence layer |
| 2 | `mode=replay-only` runtime enforcement | M-AI-EFFECT-MODES Non-Goals (parser-accepted; runtime stub) | ~10-14h | Needs replay-engine refactor |
| 3 | `scope=byok` runtime semantics | M-AI-EFFECT-MODES Non-Goals (parser-accepted; runtime stub) | ~6-8h skeleton; full BYOK is a separate sprint | Needs BYOK design doc |
| 4 | Replay-engine pin-to-resolved + `--reroute` flag | M-AI-OPENROUTER M3 Future Work | ~10-14h | Independent; required for #2 |

**Total bundle: ~30-40 hours.** Items 1, 2, 3 can ship in any order; item 4 unlocks item 2.

## Axiom Compliance (delta from v0.15.0)

These follow-ups don't change the axiom scoring of M-AI-EFFECT-MODES; they restore the design-fidelity points the evaluator deducted (−5) for the deferred handler check, plus close a small replay-determinism gap.

| Axiom | Delta | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 (item 4) | Replay-engine pin-to-resolved makes routed AI calls replayable |
| A2: Replayability | +1 (items 2+4) | `mode=replay-only` enforcement + replay-engine pin |
| A3: Effect Legibility | 0 | Already +2 in v0.15.0 |
| A4: Explicit Authority | +1 (item 3) | Full BYOK-as-capability lands here |
| A11: Structured Failure | +1 (item 1) | `ErrRoutingRequiresRouteableMode` becomes runtime-active |

## Item 1: OpenRouter handler-side defence-in-depth check

### Context

M-AI-EFFECT-MODES M2 Step 5 was deferred. The sentinel `ErrRoutingRequiresRouteableMode` already exists in [internal/ai/routing.go](../../implemented/v0_15_x/m-ai-effect-modes.md) with a TODO comment. The runtime gap: a Go caller that constructs an `ai.Handler` directly (bypassing CLI parsing AND typecheck) can pass a non-zero `AIRoutingPolicy` to OpenRouter without a `routeable` declaration anywhere.

### Why it's worth closing

- All in-tree paths go through CLI or typecheck so this isn't actively exploited
- But a future package or external consumer could construct handlers directly (e.g. motoko_agent fork; tooling using the eval harness)
- Defence-in-depth is cheap to add once `AIHandler` carries the declared mode

### What to ship

Two architectural options:

**Option A: Carry declared mode on `ai.Request`**
- Extend `ai.Request` with `DeclaredAIMode string` field (omitempty)
- AI op (`aiCall` etc.) reads the calling function's effect row from CoreTypeInfo, copies AI mode value into request before calling handler
- OpenRouter `Generate`: if `req.DeclaredAIMode == "fixed"` AND `req.Routing != nil && req.Routing.HasRouting()` → `ErrRoutingRequiresRouteableMode`

**Option B: Wait for M-AI-TOOL-LOOP**
- That sprint extends `AIHandler` from `string -> string` to `AIHandler.Step(StepRequest) -> StepResponse`
- StepRequest can naturally carry declared mode
- Free integration point

**Recommendation**: prefer Option B if M-AI-TOOL-LOOP ships first; otherwise Option A is straightforward (~50-80 LOC across ai.Request, the AI op, and openrouter handler).

### Acceptance

- [ ] `errors.Is(err, ai.ErrRoutingRequiresRouteableMode)` returns true when a handler-direct caller passes routing config without a `routeable` declared mode
- [ ] All in-tree call paths continue to work unchanged (tested via existing M-AI-OPENROUTER + M-AI-EFFECT-MODES test corpus)
- [ ] One new test in `internal/ai/openrouter/handler_modes_test.go` constructs a Request manually and asserts the error

### Estimated: ~6-10h

## Item 2: `mode=replay-only` runtime enforcement

### Context

M-AI-EFFECT-MODES parses `!{AI[mode=replay-only]}` and treats it like `routeable` for the `--allow-routing` skip, but the runtime currently dispatches identically to `mode=fixed`. The parent design doc Phase 3 (replay contract registry) is the canonical home; this is the AI-specific runtime hook.

### What "replay-only" should mean at runtime

Functions declared `!{AI[mode=replay-only]}` must NOT make live network calls. Their AI responses must come from:
- A trace baseline being replayed via `ailang replay`
- A fixture cache (e.g. `examples/runnable/fixtures/ai_responses.jsonl`)
- Another deterministic source

If no fixture/trace is available at call time → typed error (e.g. `ai.ErrReplayOnlyNoFixture`).

### What to ship

- New `ai.ReplayOnlyHandler` wrapping the unified handler; consults trace/fixture before delegating
- Effect op (`aiCall` etc.) checks declared mode and routes to the replay-only handler when applicable
- New CLI flag or config: `--ai-fixture <path>` to point at a JSONL fixture file
- Sentinel: `ai.ErrReplayOnlyNoFixture` for missing-cache cases
- Worked example: `examples/ai_replay_only.ail` with companion fixture

### Dependencies

- Item 4 (replay-engine pin-to-resolved): if the trace is the source of truth, the replay engine needs to be trace-aware (see item 4)
- Or simpler: ship fixture-only enforcement first, defer trace-replay integration to item 4

### Acceptance

- [ ] `!{AI[mode=replay-only]}` programs hit `ErrReplayOnlyNoFixture` if no fixture/trace is available
- [ ] With a fixture provided, programs run deterministically with no network calls
- [ ] `mode=replay-only` cleanly handles the case where some calls are in the cache and some aren't (typed error per missing call)

### Estimated: ~10-14h (fixture-only); +6h if trace-replay integration in scope

## Item 3: `scope=byok` runtime semantics (skeleton)

### Context

M-AI-EFFECT-MODES parses `!{AI[scope=byok]}` but runtime is identical to ambient-key flow. Full BYOK semantics (per-call key rotation, capability-flow, key-allowlist) need their own design doc.

### What this item ships

A skeleton runtime check that distinguishes ambient-key vs scoped-key paths so the type-level annotation isn't pure decoration:

- Programs declared `!{AI[scope=byok]}` must explicitly receive a key via a parameter or capability handle
- Calling with ambient `OPENROUTER_API_KEY` env-var when the function declares `scope=byok` → typed error `ai.ErrBYOKKeyRequired`
- Mechanism: pass an opaque `AIKey` (or similar) value-type into the AI op via a new builtin or extend `call`

### What this does NOT ship

- Per-call key rotation (separate design)
- Multi-tenant key allowlists
- Audit logging of key usage
- Vault/secrets-manager integration

The full BYOK design lands in a separate doc; this item is the type-level-to-runtime bridge.

### Acceptance

- [ ] `!{AI[scope=byok]}` programs cannot use ambient env-var auth; must receive key via explicit parameter
- [ ] Ambient-auth fallback continues to work for `!{AI[scope=ambient]}` (the implicit default — needs explicit `scope=ambient` row in default-mode table OR a different pattern)
- [ ] Documented as Phase 1 of BYOK; Phase 2 in separate sprint

### Estimated: ~6-8h skeleton; full BYOK is multi-sprint

## Item 4: Replay-engine pin-to-resolved + `--reroute` flag

### Context

M-AI-OPENROUTER M3 captures `ResolvedRoute` in trace events, but the replay engine itself is trace-naive: it reruns the source file with a fresh AI handler that goes through OpenRouter's routing. To replay a routed call deterministically, replay needs to consult the trace's `resolved_model` and call that specific provider/model directly.

### What to ship

- Replay engine reads each AI effect event from baseline trace
- For events with non-empty `ResolvedRoute`, replay calls `resolved_model` directly (bypasses OpenRouter routing)
- New CLI flag `--reroute` opts back into live routing during replay (useful for "what would happen now" analysis; explicitly nondeterministic)
- New `ai.TraceReplayHandler` wraps the unified handler and consults the baseline trace JSONL

### Why this completes M-AI-OPENROUTER's design

Original design doc says: "AILANG AI effects support provider-routed inference with replayable model resolution." The trace data is captured (M-AI-OPENROUTER M3) but replay still goes through routing today, so a routed call captured at v0.16.0 won't replay deterministically against today's OpenRouter (which may have rebalanced providers).

### Acceptance

- [ ] Routed AI call in baseline trace replays against the captured `resolved_model`, no live routing
- [ ] `--reroute` flag opts back into live routing
- [ ] Replay reports a structured warning if `resolved_model` is no longer available at replay time
- [ ] Existing `ailang replay` tests pass unchanged (back-compat)

### Estimated: ~10-14h

## Bundle Sequencing Recommendation

If shipping all four:

1. **Item 4 first** (Replay-engine pin-to-resolved) — independent, completes M-AI-OPENROUTER design closure
2. **Item 1** (Handler defence-in-depth) — small, restores M-AI-EFFECT-MODES design-fidelity points
3. **Item 2** (mode=replay-only enforcement) — depends on item 4 if trace-replay integration is in scope
4. **Item 3** (scope=byok skeleton) — independent; can ship first or last

Alternative: ship items 1+2+3 as a single ~25h sprint; defer item 4 to a dedicated replay-engine refactor.

## Non-Goals

Out of scope for this doc and any sprint shipping its items:

- Full BYOK semantics (per-call key rotation, capability flow, audit logging) — separate design doc
- AILANG-side `ai.complete(req)` / `openrouter.provider({...})` constructor — already tracked in [v0_17_0/m-ai-tool-loop.md](../v0_17_0/m-ai-tool-loop.md)
- Streaming AI responses — Phase 1 of m-ai-tool-loop
- Clock/Net/FS mode ports — Phase 5 of [parent doc](../v1_0_0/m-effect-refinement.md); each is its own sprint
- M-ENTROPY integration for AI mode constraints — Phase 6 of parent doc
- Subtyping or widening between AI modes — research question for v1.0+

## Related Documents

**Direct parents (shipped):**
- [M-AI-EFFECT-MODES](../../implemented/v0_15_x/m-ai-effect-modes.md) — v0.15.0 sprint that produced the deferrals catalogued here
- [M-AI-OPENROUTER](../../implemented/v0_16_x/m-ai-openrouter-provider.md) — v0.16.0 sprint with item 4's deferral
- [M-EFFECT-REFINEMENT-PHASE1](../../implemented/v0_15_x/m-effect-refinement-phase1.md) — provides parameterised-effects machinery

**Adjacent / overlapping (planned):**
- [M-AI-TOOL-LOOP](../v0_17_0/m-ai-tool-loop.md) — bigger AI runtime work; if it ships first, item 1 becomes free via the AIHandler interface extension
- [M-EFFECT-REFINEMENT (v1.0.0)](../v1_0_0/m-effect-refinement.md) — parent doc with Phases 3-6

**Evaluator report:**
- [.ailang/state/evaluations/eval_M-AI-EFFECT-MODES_round_1.json](../../../.ailang/state/evaluations/eval_M-AI-EFFECT-MODES_round_1.json) — 91/100 pass; design-fidelity −5 for the deferred handler-side check (item 1 above)

## References

- `internal/ai/routing.go` — `ErrRoutingRequiresRouteableMode` sentinel reserved for item 1
- `internal/ai/openrouter/handler.go` — handler that gains the defence-in-depth check (item 1)
- `internal/types/effects.go::EffectModeFor` — already shipped; reads declared mode from elaborated row
- `cmd/ailang/replay.go` — replay engine (item 4 territory)

## Future Work

Items intentionally NOT in this doc but worth tracking:

- **Open / user-extensible mode sets** — research question for v1.0+
- **Subtyping/widening between modes** — `routeable <: fixed` semantics?
- **AI mode inference** — should `mode=routeable` be inferred from routing-flag presence rather than annotation?
- **Per-effect entropy budgets on modes** — `!{AI[mode=routeable] @cost-budget=$0.10}`?

---

**Document created**: 2026-05-04
**Last updated**: 2026-05-04

DESIGN_DOC_PATH: design_docs/planned/v0_16_0/m-ai-effect-modes-followups.md
