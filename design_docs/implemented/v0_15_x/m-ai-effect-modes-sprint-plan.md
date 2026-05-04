# Sprint Plan: M-AI-EFFECT-MODES

## Summary

Lifts M-AI-OPENROUTER's runtime `--allow-routing` gate into a type-level `!{AI[mode=routeable]}` marker, using the parameterised-effects machinery M-EFFECT-REFINEMENT-PHASE1 just shipped. Bare `!{AI}` desugars to `!{AI[mode=fixed]}` (zero-diff back-compat). One row added to `defaultEffectModes` plus one declared-mode check in the OpenRouter handler. Validates the Phase 1 mechanism on a second pilot effect (Rand was first).

**Sprint ID:** M-AI-EFFECT-MODES
**Target Version:** v0.15.0 (alongside M-EFFECT-REFINEMENT-PHASE1)
**Design Doc:** [m-ai-effect-modes.md](m-ai-effect-modes.md)
**Parent Design:** [M-EFFECT-REFINEMENT (v1.0.0)](../v1_0_0/m-effect-refinement.md) — Phase 5, AI subset
**Duration:** 3-4 working days (~25-30 hours)
**Dependencies:** M-EFFECT-REFINEMENT-PHASE1 (✅ shipped: a90ada21+320b2a6c+72859118+de7fe9a7), M-AI-OPENROUTER (✅ v0.16.0)
**Risk Level:** Low-Medium (mechanism already validated on Rand; AI is the second pilot)

**Design-freeze decisions (await human ratification BEFORE sprint-executor starts):**
- ⏳ AI default mode = `fixed`
- ⏳ `--allow-routing` flag preserved as back-compat fallback (NOT removed)
- ⏳ `!{AI[mode=routeable]}` skips `--allow-routing` requirement (type attests intent)
- ⏳ Plumbing: handler-check via `AIHandlerWithRouting` (option c, simpler) — agent picks during sprint

## Current Status Analysis

### Completed Recently (informs velocity)

- ✅ **M-EFFECT-REFINEMENT-PHASE1** (5 commits, ~1,490 LOC, 4 days): parser, AST, row algebra with invariant unification, Rand pilot, docs. Zero-diff back-compat across 332/332 examples. Passed sprint-evaluator at full scope.
- ✅ **M-AI-OPENROUTER + gap-fill** (5 commits + 1 follow-up, ~3,500 LOC + 670 LOC, 5 days): thin OpenRouter adapter, routing policy IR, trace integration, runtime `--allow-routing` gate, docs/release. Evaluator passed 92/100.

### Velocity

- **Recent average**: ~270-540 LOC/day for milestone-style work
- **M-EFFECT-REFINEMENT-PHASE1 average**: ~370 LOC/day on the typesystem; the M2 (HIGH RISK) milestone validated the unification machinery this sprint depends on
- **Conservative estimate for this sprint**: ~150-250 LOC/day (smaller surface area; mechanism already proven). 4 days × 150 LOC/day = 600 LOC budget — aligned with the design doc's ~410 LOC estimate plus 50% buffer

### Remaining from Design Doc

- ⏳ **M1: AI default-mode entry + bare desugar** (~110 LOC, 0.5 day)
- ⏳ **M2: Runtime check + `--allow-routing` relaxation** (~200 LOC, 1.5 days)
- ⏳ **M3: Worked example + integration tests** (~100 LOC, 0.5 day)
- ⏳ **M4: Docs + release** (~150 LOC docs, 0.5 day)

**Total: ~560 LOC across 4 milestones, ~3-4 working days.**

## Proposed Milestones

### M1: AI default-mode entry + bare desugar

**Goal:** Register AI in the default-mode table. Bare `!{AI}` desugars to `!{AI[mode=fixed]}`. Validate zero-diff back-compat across all AI-using examples.

**Estimated:** ~30 LOC implementation + ~80 LOC tests = ~110 LOC
**Duration:** 0.5 day (~3-4 hours)
**Risk:** Low — mechanism already validated by M-EFFECT-REFINEMENT-PHASE1's Rand pilot.

**Tasks:**
- Day 1 (morning): Add `"AI": {"mode", "fixed"}` to `internal/types/effects.go::defaultEffectModes`
- Day 1 (morning): Tests in `internal/types/effect_params_test.go`:
  - `TestDefaultModeFor_AI` — returns `("mode", "fixed", true)`
  - `TestElaborateEffectRow_AIDefault` — bare `!{AI}` → `Params["AI"] == {"mode": "fixed"}`
  - `TestUnify_AIModes` — 5 cases: same-mode unify, default-desugar match, different-modes mismatch, polymorphic tail, row swap
- Day 1 (morning): Pre-/post-sprint typecheck-output zero-diff sweep on AI-using examples (`examples/ai_openrouter_routing.ail`, `examples/ai_modes.ail` once M3 lands; for M1 verification use the M-AI-OPENROUTER demo)

**Files to modify:**
- `internal/types/effects.go` (~5 LOC delta)
- `internal/types/effect_params_test.go` (~80 LOC delta — AI test cases)

**Acceptance Criteria:**
- [ ] `DefaultModeFor("AI")` returns `("mode", "fixed", true)`
- [ ] Bare `!{AI}` elaborates to a Row with `Params["AI"] == {"mode": "fixed"}`
- [ ] `!{AI[mode=routeable]}` does NOT unify with `!{AI[mode=fixed]}` (invariant)
- [ ] All existing AI-using examples produce byte-identical typecheck output pre-/post-M1 (zero-diff sweep)
- [ ] `make verify-examples` 171/171
- [ ] `go test -count=20 ./internal/types/...` deterministic
- [ ] `make lint` clean

**Risks:**
- **Pretty-printer change in error messages** (low) — analogous to M-EFFECT-REFINEMENT-PHASE1's `TestFormatEffectRow/all_effects` update; expect AI-formatted output to gain `[mode=fixed]` annotation. Update any affected golden tests.

### M2: Runtime check + `--allow-routing` relaxation

**Goal:** OpenRouter handler rejects routing config on functions declared `!{AI[mode=fixed]}` with a typed error; accepts `!{AI[mode=routeable]}` without `--allow-routing`. Bare `!{AI}` + flags + `--allow-routing` continues to work (back-compat fallback).

**Estimated:** ~100 LOC implementation + ~100 LOC tests = ~200 LOC
**Duration:** 1.5 days (~10-12 hours)
**Risk:** Medium — plumbing the declared mode from typechecker output to handler is the architecture decision; option c (handler-check via `AIHandlerWithRouting`) is simpler.

**Tasks:**
- Day 1 (afternoon): Define `ErrRoutingRequiresRouteableMode` sentinel in `internal/ai/routing.go`; sibling to existing `ErrRoutingNotSupported`
- Day 2 (morning): Plumbing — pick option c (handler-check). The AI op (`aiCall` etc.) reads the elaborated effect row from CoreTypeInfo, passes the AI-mode value to the handler via either an extended `AIHandlerWithRouting` interface method or via the request directly.
- Day 2 (afternoon): OpenRouter handler check: if `req.Routing != nil && req.Routing.HasRouting()` AND declared `mode == "fixed"` → return `ErrRoutingRequiresRouteableMode` with a typed error message.
- Day 2 (afternoon): `--allow-routing` relaxation in `cmd/ailang/routing_flags.go`: when declared mode is `routeable` (or `replay-only`), skip the runtime gate.
- Day 3 (morning): Tests:
  - `TestOpenRouterHandler_RejectsFixedModeWithRouting` — typed error for `!{AI[mode=fixed]}` + routing
  - `TestOpenRouterHandler_AcceptsRouteableMode` — succeeds without `--allow-routing` for `!{AI[mode=routeable]}`
  - `TestAllowRoutingBackCompat` — bare `!{AI}` + flags + `--allow-routing` still works
  - All existing M-AI-OPENROUTER tests pass unchanged

**Files to create:**
- `internal/ai/openrouter/handler_modes_test.go` (~100 LOC)

**Files to modify:**
- `internal/ai/openrouter/handler.go` OR equivalent (~80 LOC delta — declared-mode check)
- `internal/ai/routing.go` (~10 LOC delta — `ErrRoutingRequiresRouteableMode` sentinel)
- `cmd/ailang/routing_flags.go` (~20 LOC delta — relaxation branch)
- `cmd/ailang/exec.go` and `main_run.go` (~30 LOC delta — declared-mode plumbing if needed)

**Acceptance Criteria:**
- [ ] `ErrRoutingRequiresRouteableMode` exported, sentinel-checkable via `errors.Is`
- [ ] OpenRouter handler rejects routing config on `!{AI[mode=fixed]}` (or bare `!{AI}` desugared) with typed error
- [ ] OpenRouter handler accepts routing config on `!{AI[mode=routeable]}` without `--allow-routing`
- [ ] Bare `!{AI}` + routing flags + `--allow-routing` continues to work
- [ ] Bare `!{AI}` + routing flags WITHOUT `--allow-routing` → existing safety-gate error (unchanged)
- [ ] All M-AI-OPENROUTER tests pass unchanged
- [ ] `make test` clean
- [ ] `make lint` clean

**Risks:**
- **Plumbing typechecker output to handler is harder than option c assumes** (med) — fallback: capture mode at command setup time (after typecheck, before run) rather than per-call.
- **`--allow-routing` relaxation breaks an existing test** (med) — make relaxation strictly additive: only `!{AI[mode=routeable]}` skips the requirement; bare `!{AI}` keeps existing behaviour (still requires `--allow-routing` when flags are set).

### M3: Worked example + integration tests

**Goal:** `examples/ai_modes.ail` demonstrates the three modes side-by-side. Runs end-to-end with stub handler.

**Estimated:** ~40 LOC example + ~60 LOC integration tests = ~100 LOC
**Duration:** 0.5 day (~3-4 hours)
**Risk:** Low.

**Tasks:**
- Day 3 (afternoon): Author `examples/ai_modes.ail` with three functions:
  - `summarize_default(text) -> string ! {AI}` — bare, desugars to fixed
  - `summarize_explicit_fixed(text) -> string ! {AI[mode=fixed]}` — explicit form
  - `summarize_routeable(text) -> string ! {AI[mode=routeable]}` — routing-capable
- Day 3 (afternoon): `main()` calls all three with stub handler
- Day 3 (afternoon): Document live-OpenRouter invocation in the file's header comment (similar to `examples/ai_openrouter_routing.ail`)
- Day 3 (afternoon): End-to-end CLI integration test in a new `cmd/ailang/exec_modes_integration_test.go` or extend an existing test file: spawns `ailang run` with `--ai-stub` and asserts all three calls produce stub responses

**Files to create:**
- `examples/ai_modes.ail` (~40 LOC)
- `cmd/ailang/exec_modes_integration_test.go` OR extend existing (~60 LOC)

**Acceptance Criteria:**
- [ ] `examples/ai_modes.ail` runs end-to-end with `--ai-stub`: `ailang run --caps AI,IO --ai-stub --entry main examples/ai_modes.ail`
- [ ] Three stub responses printed (one per function)
- [ ] `make verify-examples` still 171/171 (or 172/172 if `ai_modes.ail` is auto-discovered like `examples/runnable/*.ail`; if it's at top level next to `ai_openrouter_routing.ail`, count stays 171)
- [ ] Integration test passes

**Risks:** None significant.

### M4: Docs + release

**Goal:** User-facing documentation, teaching prompt update, CHANGELOG, design-doc move.

**Estimated:** ~150 LOC docs (no Go code)
**Duration:** 0.5 day (~3-4 hours)
**Risk:** Low.

**Tasks:**
- Day 4 (morning): `docs/docs/guides/ai-routing.md` — new section "Type-level mode markers (v0.15.0+)" pointing at `!{AI[mode=...]}` syntax and explaining the `--allow-routing` fallback
- Day 4 (morning): `docs/docs/guides/parameterised-effects.md` — add AI to the default-mode table example (was placeholder under "Future modes (parent doc)")
- Day 4 (morning): Teaching prompt sections in `prompts/v0.16.0.md` and `cmd/ailang/prompts/v0.16.0.md` — mention AI modes alongside Rand modes (~20 LOC each)
- Day 4 (afternoon): CHANGELOG entry under v0.15.0 in `changelogs/v0.10-current.md` (narrative-paragraph + per-milestone bullets style)
- Day 4 (afternoon): Move design doc + sprint plan to `design_docs/implemented/v0_15_x/`
- Day 4 (afternoon): Append implementation report to design doc

**Files to create:**
- (None new — only docs additions to existing files)

**Files to modify:**
- `docs/docs/guides/ai-routing.md` (~30-50 LOC delta)
- `docs/docs/guides/parameterised-effects.md` (~10 LOC delta)
- `prompts/v0.16.0.md` (~20 LOC delta)
- `cmd/ailang/prompts/v0.16.0.md` (~20 LOC delta)
- `prompts/versions.json` and `cmd/ailang/prompts/versions.json` (SHA-256 hash updates — required for prompt-loader integrity tests)
- `changelogs/v0.10-current.md` (entry)
- Move: `design_docs/planned/v0_15_0/m-ai-effect-modes*.md` → `design_docs/implemented/v0_15_x/`

**Acceptance Criteria:**
- [ ] `ai-routing.md` has type-level mode-markers section
- [ ] `parameterised-effects.md` lists AI in default-mode table
- [ ] Teaching prompts mention AI modes (parallel sections in both files)
- [ ] CHANGELOG entry references parent design doc + Phase-1 doc + this Phase-5-AI doc
- [ ] `make ci` green
- [ ] Design doc + sprint plan moved to `implemented/v0_15_x/` with status updated and implementation report appended

## Success Metrics

- **Test coverage**: ≥80% on new test files; no regression elsewhere
- **Examples passing**: `examples/ai_modes.ail` runs; `make verify-examples` still 171/171
- **Zero-diff guarantee**: existing AI-using examples (e.g. `ai_openrouter_routing.ail`) byte-identical typecheck output pre-/post-sprint
- **Documentation updated**: ai-routing.md, parameterised-effects.md, CHANGELOG, prompts
- **All tests passing**: ✅ `make test` and `make lint` (0 issues)
- **All M-AI-OPENROUTER tests pass unchanged**

## Dependencies

- **Internal (shipped)**: M-EFFECT-REFINEMENT-PHASE1, M-AI-OPENROUTER
- **Coordination**: None required outside the codebase
- **Approval gate**: Four design-freeze items must be ratified by user BEFORE sprint-executor starts

## Open Questions

- Plumbing: handler-check (option c) vs typechecker pass-through (a/b)? Sprint-executor decides during M2; recommend option c. Document the choice.
- Should `mode=replay-only` trigger any runtime behaviour in this sprint, or strictly parser-accepts? Recommend strict parser-accepts with TODO comment for Phase 3.
- Test the `!{AI[mode=routeable]}` skips `--allow-routing` behaviour: how to capture the declared mode when CLI flags are validated before typecheck completes? Two options: (a) defer the safety gate to handler time, (b) re-validate after typecheck. Agent picks during M2.

## Notes

- This sprint completes the type-level half of M-AI-OPENROUTER M3's deferred work. The runtime substance was shipped in v0.16.0; the type-level marker lands here in v0.15.0 (yes, before v0.16.0 in version order — both target the same release window since v0.15.0 hasn't shipped yet).
- The pattern (`one row in defaultEffectModes` + `handler check` + `example`) is identical to what Clock/Net/FS port sprints will follow. This sprint validates that pattern as the second instance after Rand.
- M-EFFECT-REFINEMENT-PHASE1's M2 zero-diff sweep already proved the back-compat mechanism is sound. This sprint inherits that guarantee.
- The `effectiveParamsOf` bridge from M-EFFECT-REFINEMENT-PHASE1 M2 (which normalises bare-effect rows via `DefaultModeFor` during comparison) handles the back-compat automatically — no new bridge logic needed here.

---

**SPRINT_PLAN_PATH**: `design_docs/planned/v0_15_0/m-ai-effect-modes-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_M-AI-EFFECT-MODES.json`
