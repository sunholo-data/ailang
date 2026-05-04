# Sprint Plan: M-EFFECT-REFINEMENT-PHASE1

## Summary

Phase 1 of [M-EFFECT-REFINEMENT (v1.0.0)](../v1_0_0/m-effect-refinement.md) — ship the language-feature scaffolding for parameterised effects `!{E[k=v, ...]}`. Parser+AST accept the syntax, row algebra and unification rules accept parameters, and `type CryptoRand = Rand[mode=crypto]` validates zero-diff against existing v0.13.x M-CRYPTORAND programs. Rand is the pilot effect; Clock/Net/FS/AI ports + replay contract registry + capability scoping + M-ENTROPY integration stay deferred to follow-up sprints.

**Sprint ID:** M-EFFECT-REFINEMENT-PHASE1
**Target Version:** v0.15.0
**Design Doc:** [m-effect-refinement-phase1.md](m-effect-refinement-phase1.md) — Phase-1 carve-out
**Parent Design:** [m-effect-refinement.md](../v1_0_0/m-effect-refinement.md) — full 8-phase taxonomy
**Duration:** 5 working days (~37 hours)
**Dependencies:** M-CRYPTORAND (v0.13.0, ✅ shipped) — back-compat alias target / load-bearing validation; M-AI-OPENROUTER (v0.16.0, ✅ shipped) — informs the deferred AI port pattern
**Risk Level:** Medium-High (typechecker change; M2 unification rules are the hardest piece)

**Design-freeze decisions (await human approval BEFORE sprint-executor starts):**

- ⏳ Surface syntax `!{E[k=v, ...]}` ratified (per parent doc; recommended)
- ⏳ Mode set closed for Phase 1 (compiler-known per effect; no user extensibility yet)
- ⏳ Back-compat aliasing: bare `!{E}` desugars to `!{E[mode=default_for_E]}`
- ⏳ Unification rule: invariant on params (no subtyping in Phase 1)
- ⏳ Default mode table: Phase 1 ships only the `Rand → os` row; Clock/Net/FS/AI rows added by their port sprints

## Current Status Analysis

### Completed Recently (last 14 days, informs velocity)

- ✅ **M-AI-OPENROUTER** (4 milestones + gap-fill, ~5 days): thin OpenRouter adapter, routing policy IR, trace integration + `--allow-routing` safety gate, docs/release, then a follow-up wiring `ailang run` correctly. Total ~3,500 LOC.
- ✅ **M-EFFECT-REFINEMENT extension** for AI mode taxonomy (commit b683f9c5): added AI to the parent doc's collapsed-contracts table and authored Example 4: Modal AI as the canonical home for the deferred type-level work.
- ✅ **M-MAC-NOTIFY-DAEMON** (M4 ship): docs, smoke, Makefile, supersede hook
- ✅ **M-AGENT-MCP** sprint (7/8 milestones, in-flight): server-side filtering, per-stdlib-module JSON, MCP HTTP transport hardening
- ✅ **M-TAINT-TYPES** (v0.14.3 → v0.16.0): IFC labels, AST/parser/typechecker integration; precedent for this sprint's threading-new-AST-through-7-place-switch pattern

### Velocity

- **Recent average**: 150-540 LOC/day for milestone-style work; multi-day milestones land in 1-3 calendar days
- **M-AI-OPENROUTER average**: ~750 LOC/day with sub-agent parallelism, but contained to single packages (`internal/ai/`)
- **This sprint touches the typechecker** (`internal/types/`, `internal/parser/`, `internal/ast/`), which is higher-risk than provider-package work
- **Conservative estimate**: ~270 LOC/day for 5 days = 1,350 LOC budget — aligns with the design doc's ~1,365 LOC estimate

### Remaining from Design Doc (this sprint)

- ⏳ **M1: Phase 1A — Parser + AST** (~370 LOC, 1-1.5 days)
- ⏳ **M2: Phase 1B — Row algebra + unification** (~560 LOC, 1.5-2 days, **highest risk**)
- ⏳ **M3: Phase 1C — Rand pilot + CryptoRand alias** (~125 LOC, ~0.5 day)
- ⏳ **M4: Phase 1D — Docs + release** (~310 LOC, ~0.5 day)

**Total: ~1,365 LOC across 4 milestones, ~5 working days.**

## Proposed Milestones

### M1: Parser + AST (Phase 1A)

**Goal:** Accept `!{E[k=v, k2=v2]}` syntax in the parser. AST gains an effect-parameter representation. Pretty-printer round-trips parameters with deterministic ordering.

**Estimated:** ~250 LOC implementation + ~120 LOC tests = ~370 LOC
**Duration:** 1 day (~7 hours)
**Risk:** Medium — parser ambiguity between effect-row `[k=v]` brackets and existing list/array `[a]` brackets is the main hazard.

**Tasks:**

- Day 1 (morning): Lexer disambiguation. Choose: parser-level context disambiguation (already in effect-row context) vs distinct `LBRACKET_EFFECT` token. Document choice.
- Day 1 (morning): Parser grammar for `! {E[k=v, ...]}` — comma-separated `key=value` pairs, values are bare identifiers or strings. Emit structured errors for malformed forms (`{k=}`, `{=v}`, `{k v}`, `{k:v}`).
- Day 1 (afternoon): AST representation — recommend `Params []EffectParam` (slice of `{Key, Value}`) on existing effect AST node. Keeps ordering for tests; pretty-printer sorts on output.
- Day 1 (afternoon): Pretty-printer round-trips parameters alphabetically (deterministic for golden files; document the choice).
- Day 1 (afternoon): Tests — positive cases, four malformed-form structured errors at correct line/column, parser→pretty-printer→parser round-trip identical AST.

**Files to create:**
- `internal/parser/parser_effect_params_test.go` (~80 LOC)
- `internal/ast/effect_params_test.go` (~40 LOC)

**Files to modify:**
- `internal/lexer/token.go` (~15 LOC delta — disambiguation or new token)
- `internal/parser/parser.go` (~200 LOC delta — grammar)
- `internal/ast/ast.go` (~80 LOC delta — `EffectParam`)

**Acceptance Criteria:**

- [ ] `! {E[k=v, k2=v2]}` parses; AST exposes `[]EffectParam`
- [ ] Malformed forms (`{k=}`, `{=v}`, `{k v}`, `{k:v}`) produce structured parser errors at correct line/column
- [ ] Pretty-printer round-trip is identity (parse → print → reparse → identical AST)
- [ ] Pretty-printer ordering is deterministic (run with `-count=20` to catch map-iteration nondeterminism per coding standards)
- [ ] All existing parser tests pass unchanged (no regressions in non-parameterised effect rows)
- [ ] `make lint` clean

**Risks:**

- **Parser ambiguity** (high) — Mitigation: parser-level context check (effect rows already known to parser); fall back to distinct `LBRACKET_EFFECT` token if context-sensitivity proves fragile. Verify with golden parser tests on `! {Rand[mode=os]}` vs `[1, 2, 3]` vs `Array[int]` adjacencies.
- **Existing parser tests broken by lexer change** (med) — Mitigation: run `make test ./internal/parser/... ./internal/lexer/...` after each step; the parser-developer skill's rules apply.

### M2: Row algebra + unification (Phase 1B)

**Goal:** Extend effect-row representation with parameter map. Implement bare-effect desugar via per-effect default-mode table. Unification rules: invariant on params, polymorphic-tail-compatible. Verify zero-diff back-compat across the entire example corpus.

**Estimated:** ~440 LOC implementation + ~120 LOC tests = ~560 LOC
**Duration:** 1.5-2 days (~16 hours)
**Risk:** **HIGH** — Typechecker change. M2 is the load-bearing milestone of the sprint.

**Tasks:**

- Day 2 (morning): Extend effect-row representation in `internal/types/effects.go` with parameter map keyed by effect name. Decide: per-label `Params` field on `Row.Labels[name]` vs parallel `RowParams map[string][]EffectParam` (agent's call; document).
- Day 2 (morning): `DefaultModeFor(name string) (k, v string, ok bool)` lookup table — Phase 1 entry: `("Rand", "mode", "os")`. Other effects return `ok=false` (their bare forms continue to type-check as today).
- Day 2 (afternoon): Bare-effect desugar in elaboration: `!{E}` → `!{E[mode=default_for_E]}` when `DefaultModeFor` has an entry; otherwise leave bare.
- Day 3 (morning): Unification rules in `internal/types/unification.go` — two effects unify iff same name AND same parameter map (invariant). Polymorphic row tail still works.
- Day 3 (morning): Test matrix — 5+ cases per design-doc Example 2: same params, default-desugar match, different params (FAIL), polymorphic tail, row swap.
- Day 3 (afternoon): Back-compat sweep — `make verify-examples` 171/171, plus a typecheck-output diff harness that snapshots every example pre-/post-sprint and asserts byte-identical results.
- Day 4 (morning): Buffer for any back-compat regressions surfaced by the sweep. **Pause-and-reassess decision point**: if unification rules turn out harder than the parent doc's sketch, scope down to ship parser+AST only and defer the row algebra to a follow-up.

**Files to create:**
- `internal/types/effects_params_test.go` (~80 LOC — unification matrix)
- `tests/golden/typecheck/effect_params/...` (~40 LOC golden files for desugar)

**Files to modify:**
- `internal/types/effects.go` (~260 LOC delta — `Effect` representation, default-mode table, bare desugar)
- `internal/types/unification.go` (~180 LOC delta — invariant param unification)

**Acceptance Criteria:**

- [ ] Effect-row representation carries `Params []EffectParam` per effect label
- [ ] `DefaultModeFor("Rand")` returns `("mode", "os", true)`; other effects return `_, _, false` (Phase 1 ships only the Rand row)
- [ ] Bare `! {Rand}` elaborates to `! {Rand[mode=os]}`; pretty-printer reflects this
- [ ] `! {Rand[mode=os]}` unifies with `! {Rand}` (default-desugar match)
- [ ] `! {Rand[mode=os]}` does NOT unify with `! {Rand[mode=seeded]}` (invariant)
- [ ] `! {Rand[mode=os] | a}` unifies with `! {Rand[mode=os], FS | a}` (poly tail preserved)
- [ ] `! {Rand[mode=os], FS}` unifies with `! {FS, Rand[mode=os]}` (row swap)
- [ ] **Zero-diff back-compat**: every example in `examples/runnable/**/*.ail` produces byte-identical typecheck output pre-/post-sprint
- [ ] `make verify-examples` 171/171
- [ ] `go test ./internal/types/... -count=20` passes (catches map-iteration nondeterminism in canonicalisation)
- [ ] `make lint` clean

**Risks:**

- **Back-compat sweep miss** (HIGH) — Mitigation: byte-identical typecheck-output diff harness over the full example corpus; CI guard.
- **Unification rule subtlety in row-polymorphic code** (med) — Mitigation: test matrix covers 5+ cases; `DEBUG_TYPES=1` traces for edge cases; conservative invariant rule (no subtyping) keeps inference predictable.
- **Map-iteration nondeterminism** (med) — Mitigation: pretty-printer sorts; tests run `-count=20` per `internal/types/labels.go` precedent.
- **M2 takes longer than 2 days** (med) — Mitigation: explicit pause-and-reassess at end of Day 3; if unification proves harder than expected, ship M1 (parser+AST) only and defer M2 to a follow-up sprint that lands before any other-effect ports.

### M3: Rand pilot + CryptoRand alias (Phase 1C)

**Goal:** Ship `type CryptoRand = Rand[mode=crypto]` as the load-bearing validation. Existing M-CRYPTORAND programs compile and run zero-diff. Worked example demonstrates the three modes.

**Estimated:** ~85 LOC implementation + ~40 LOC tests = ~125 LOC
**Duration:** ~0.5 day (~4 hours)
**Risk:** Medium — CryptoRand alias is a release-gating zero-diff check.

**Tasks:**

- Day 4 (afternoon): `type CryptoRand = Rand[mode=crypto]` alias in `stdlib/std/crypto/rand.ail`.
- Day 4 (afternoon): Update `stdlib/std/rand.ail` if needed (modes, default preserved by desugar — likely no signature changes required).
- Day 4 (afternoon): Zero-diff regression test: snapshot `inbox_v2_lib.ail` + `inbox_v2_app.ail` compiled output pre-/post-sprint, assert identical (typecheck + runtime trace where applicable).
- Day 4 (afternoon): `examples/modal_rand.ail` (~60 LOC) demonstrating `!{Rand[mode=seeded]}`, `!{Rand[mode=os]}`, `!{Rand[mode=crypto]}` side-by-side. Must run with stub seeded mode.

**Files to create:**
- `examples/modal_rand.ail` (~60 LOC)
- `tests/golden/cryptorand_zero_diff_test.go` (~40 LOC)

**Files to modify:**
- `stdlib/std/crypto/rand.ail` (~5 LOC — alias)
- `stdlib/std/rand.ail` (~20 LOC delta if any)

**Acceptance Criteria:**

- [ ] `type CryptoRand = Rand[mode=crypto]` parses and elaborates correctly
- [ ] M-CRYPTORAND programs (`examples/runnable/contracts/inbox_v2_lib.ail`, `inbox_v2_app.ail`) produce **byte-identical** compiled output pre-/post-sprint
- [ ] M-CRYPTORAND programs produce **identical runtime behaviour** pre-/post-sprint (same trace events, same output)
- [ ] `examples/modal_rand.ail` runs with `--ai-stub` style seeded mode (deterministic)
- [ ] `make verify-examples` still 171/171

**Risks:**

- **CryptoRand alias breaks callers** (HIGH) — Mitigation: byte-identical snapshot test on the two M-CRYPTORAND demo programs; release-gating CI check.
- **`type` alias declaration not yet supported in stdlib for parameterised effect types** (med) — Mitigation: verify before M3 starts whether AILANG's existing type-alias machinery accepts parameterised effect types in the body; if not, scope-down to define `CryptoRand` as a synonym at the module level rather than a type alias.

### M4: Docs + release (Phase 1D)

**Goal:** User-facing documentation, teaching prompt update, CHANGELOG, design-doc move. Sprint complete.

**Estimated:** ~310 LOC docs (no Go code)
**Duration:** ~0.5 day (~3 hours)
**Risk:** Low.

**Tasks:**

- Day 5 (morning): `docs/docs/guides/parameterised-effects.md` (~250-400 lines): syntax, default-mode table, closed-mode-set rationale, back-compat alias mechanism, forward pointer to parent doc and follow-up sprints. Register in `docs/sidebars.js`.
- Day 5 (morning): Teaching prompt sections in `prompts/v0.16.0.md` and `cmd/ailang/prompts/v0.16.0.md` (~30 LOC each).
- Day 5 (afternoon): CHANGELOG entry under v0.15.0 in `changelogs/v0.10-current.md`. Match the established narrative-paragraph + per-milestone bullet style. Link parent design doc + Phase-1 doc + sprint plan paths (post-move).
- Day 5 (afternoon): `make ci` green check (build, test, lint, verify-examples, file-size).
- Day 5 (afternoon): Move design doc + sprint plan to `design_docs/implemented/v0_15_x/`. Append implementation report to design doc covering per-milestone LOC, coverage, deferred-from-M2-if-applicable, and architectural notes.

**Files to create:**
- `docs/docs/guides/parameterised-effects.md` (~250-400 lines)

**Files to modify:**
- `prompts/v0.16.0.md` (~30 LOC)
- `cmd/ailang/prompts/v0.16.0.md` (~30 LOC)
- `docs/sidebars.js` (sidebar registration)
- `changelogs/v0.10-current.md` (entry)
- Move `design_docs/planned/v0_15_0/m-effect-refinement-phase1*.md` → `design_docs/implemented/v0_15_x/`

**Acceptance Criteria:**

- [ ] `docs/docs/guides/parameterised-effects.md` published, registered in sidebar
- [ ] Teaching prompts updated with parameterised-effects section
- [ ] CHANGELOG entry references parent design doc + Phase-1 doc + sprint plan
- [ ] `make ci` green
- [ ] Design doc + sprint plan moved to `design_docs/implemented/v0_15_x/` with status updated and implementation report appended

**Risks:** None significant.

## Success Metrics

- **Test coverage**: ≥80% on `internal/types/effects.go` (extended), no regression elsewhere
- **Examples passing**: `examples/modal_rand.ail` runs; `make verify-examples` 171/171
- **Zero-diff guarantee**: M-CRYPTORAND programs (`inbox_v2_lib.ail`, `inbox_v2_app.ail`) byte-identical compiled output pre-/post-sprint
- **Documentation updated**:
  - `docs/docs/guides/parameterised-effects.md` (new)
  - `CHANGELOG.md` v0.15.0 entry
  - Teaching prompts updated
  - Design doc + sprint plan moved to `implemented/v0_15_x/`
- **All tests passing**: ✅ `make test` (with `-count=20` on type-system canonicalisation)
- **All linting passing**: ✅ `make lint`

## Dependencies

- **Internal**: M-CRYPTORAND (v0.13.0, ✅) — back-compat alias target / load-bearing validation
- **Coordination**: None required outside the codebase
- **Approval gate**: Five design-freeze items must be ratified by user BEFORE sprint-executor starts (see Summary)

## Open Questions

- Is parser-level context disambiguation for `[k=v]` cleaner than introducing `LBRACKET_EFFECT`? Sprint-executor decides during M1; document the choice.
- Does AILANG's existing type-alias machinery accept parameterised effect types in the body of `type CryptoRand = Rand[mode=crypto]`? Verify before M3; fall back to a module-level synonym if needed.
- Should `make verify-examples` produce the byte-identical typecheck-output diff in CI, or only as a one-shot pre-/post-sprint check? Recommend one-shot in this sprint; promote to CI guard in a follow-up if the back-compat surface grows.

## Notes

- The parent [M-EFFECT-REFINEMENT (v1.0.0)](../v1_0_0/m-effect-refinement.md) doc retains the full 8-phase picture. This Phase-1 doc + sprint plan are the v0.15.0-deliverable carve-out. Phases 3-6 stay tracked in the parent for follow-up sprints.
- The `--allow-routing` runtime gate from M-AI-OPENROUTER (v0.16.0) is the runtime substitute for the type-level `!{AI[mode=routeable]}` marker that lands in Phase 5 of the parent. Phase 1 unblocks that work but does not yet ship the AI port — Phase 5 of parent doc covers it.
- Pre-/post-sprint typecheck-output diff is the recommended back-compat verification mechanism. It catches subtle elaboration changes the way M-TAINT-TYPES M5's "20 contract examples produce identical pre-/post-sprint outcomes" check did.
- If M2's unification rules turn out harder than the parent doc's sketch, the explicit pause-and-reassess at end of Day 3 lets us ship M1 only and defer the row algebra to a follow-up sprint without violating the v0.15.0 release window.

---

**SPRINT_PLAN_PATH**: `design_docs/planned/v0_15_0/m-effect-refinement-phase1-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_M-EFFECT-REFINEMENT-PHASE1.json`
