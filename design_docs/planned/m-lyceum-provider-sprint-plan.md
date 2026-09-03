# Sprint Plan: M-LYCEUM-PROVIDER

## Summary

Add Lyceum (EU-hosted, OpenAI-compatible, per-token metered) as a built-in provider so
`provider: "lyceum"` rows in `models.yml` run through the standard eval harness, then seed
three rows and A/B the route against its OpenRouter twins.

**Design doc:** [m-lyceum-provider.md](m-lyceum-provider.md) (Phase 0 spike COMPLETE — V16–V22 verified 2026-09-03)
**Duration:** 1 day (~5–6 hours remaining)
**Dependencies:** `LYCEUM_API_KEY` in `~/.zshenv` (present, verified). D4 ratified (Mark, 2026-09-03).
**Risk Level:** Low (additive-only change; every premise live-verified in Phase 0)

## Current Status Analysis

### Completed
- ✅ Design doc created + quorum-gate duplicate check passed (max similarity 0.28)
- ✅ D4 design-freeze item ratified by Mark: seed rows = `lyceum-glm-5-3-flash`,
  `lyceum-qwen3-8-flash-next`, `lyceum-kimi-k3`
- ✅ Phase 0 spike (V16–V22): slugs verified (correction: `qwen/qwen3.8-flash-next`),
  streaming + reasoning fields + error shapes standard, step/stream usage structs lack
  `completion_tokens_details` (contingency task in M1)

### Velocity
- Recent pace: multiple milestone commits/day on comparable provider/harness work
- This sprint is small by design: ~240 LOC total, one session

### Remaining from Design Doc
- ⏳ M1 Provider plumbing: ~60 LOC impl + ~80 LOC tests
- ⏳ M2 Seed rows + smoke gate: ~100 LOC models.yml (3 rows with provenance notes)
- ⏳ M3 Route A/B + decision note: run + record (~0 LOC)

## Proposed Milestones

### Milestone 1: Provider plumbing (built-in `lyceum` provider)
**Goal:** `provider: "lyceum"` rows resolve to an openai-transport handler pointed at
`https://api.lyceum.technology/openai/v1`, authed by `LYCEUM_API_KEY`, classified dual-mode.
**Estimated:** ~60 impl + ~80 test = ~140 LOC
**Duration:** ~3 hours

**Tasks:**
1. `internal/ai/config.go`: `ProviderLyceum` const, `ProviderFromString` case,
   `EnvVarForProvider` → `LYCEUM_API_KEY`, `GetAPIKey` case (~15 LOC)
2. `cmd/ailang/ai_handlers.go`: dispatch cases in `setupAIHandlerFromConfig` +
   `setupAIHandlerDirect` — `openai.NewClient(key, openai.WithBaseURL(lyceumBaseURL))`,
   loud named error on missing key (~15 LOC)
3. `cmd/ailang/exec.go`: valid-provider sets (~135, ~139) + dispatch case near openrouter (~10 LOC)
4. `internal/ai/registry.go`: `"lyceum"` in `builtInProviderNames` (1 LOC)
5. `internal/modelreg/models.go:383`: add `"lyceum"` to the `SupportsStandardEval` list (1 LOC)
6. `cmd/ailang/help.go` + `docs/docs/reference/` env-var docs (~20 LOC)
7. Unit tests: enum round-trip; httptest stub proves base URL + Authorization header land
   on the wire; missing-key error; `builtInProviderNames` reservation; `OPENAI_BASE_URL`
   path untouched (~80 LOC)
8. CONTINGENCY (from V19): if smoke banks reason_tokens=0 on GLM, extend `ChatStepUsage`
   (step.go:336) + streamstep usage struct with `CompletionTokensDetails{ReasoningTokens}`
   (3 lines each, mirror types.go:58-66)

**Example files to create/update:**
- Modify: `internal/ai/config.go`, `internal/ai/registry.go`, `internal/modelreg/models.go`,
  `cmd/ailang/ai_handlers.go`, `cmd/ailang/exec.go`, `cmd/ailang/help.go`
- Modify: `internal/ai/config_test.go` (+ dispatch test file if the cmd/ailang pattern
  keeps those separate)
- Reference (existing patterns to mirror): `cmd/ailang/ai_handlers.go` openai/openrouter cases;
  `internal/ai/openai/client.go` `WithBaseURL`; `internal/ai/openai/client_test.go` httptest pattern

**Acceptance Criteria:**
- [ ] `ProviderFromString("lyceum") == ProviderLyceum`, env var `LYCEUM_API_KEY`
- [ ] Dispatch builds a handler whose wire target is the Lyceum base URL (stub-asserted)
- [ ] Missing key → error naming `LYCEUM_API_KEY` (no silent fallback)
- [ ] `SupportsStandardEval` returns true for a `lyceum` provider row
- [ ] `OPENAI_BASE_URL` global-override behaviour unchanged (existing tests still pass)
- [ ] All tests passing; `make lint` / `make fmt` clean

**Risks:**
- Missed hardcoded provider list somewhere — Mitigation: grep `"openrouter"` across
  cmd/ + internal/ (9 files, V8) and diff each against the lyceum additions

### Milestone 2: Seed rows + smoke gate
**Goal:** Three provenance-labelled rows; smoke tier passes through the new route.
**Estimated:** ~100 LOC (models.yml)
**Duration:** ~1 hour

**Tasks:**
1. Write 3 rows with V16-verified slugs, dashboard prices labelled
   `verified via dashboard 2026-09-03, wire-reconcile pending (V22: no billing API)`,
   `default_thinking: "always_on"` (GLM, V19-observed) / `"on"` (kimi, qwen — per twin rows),
   `max_output_tokens: 65536`, budgets scaled to Lyceum prices (flash 0.30, kimi 0.60)
2. Row-key names: `lyceum-glm-5-3-flash`, `lyceum-qwen3-8-flash-next`, `lyceum-kimi-k3`
   (provider-prefix convention, like `or-*`)
3. `modelreg` validation + `make build`
4. Smoke gate: `ailang eval-suite --models lyceum-glm-5-3-flash --tier smoke --langs ailang`
   (expect ≈ $0.06–0.15 at Lyceum prices)
5. Banked-row checks: provider=lyceum, real tokens, cost>0, provenance=metered,
   reason_tokens present (triggers M1 contingency if 0)

**Example files to create/update:**
- Modify: `internal/modelreg/models.yml` (3 rows, placed near the `or-*` cluster)

**Acceptance Criteria:**
- [ ] AC1: smoke tier completes in standard mode, zero changes to `internal/ai/openai/`
- [ ] AC2: banked rows show provider=lyceum, real tokens, cost>0, provenance=metered
- [ ] AC3: reason_tokens captured on GLM (or contingency applied, then captured)
- [ ] All tests passing

**Risks:**
- Upstream 429s — Mitigation: run smoke `--parallel 1` (R4), retry once before counting failures
- Price drift vs dashboard — Mitigation: rows carry dated provenance; Mark eyeballs the
  dashboard once before the gate (V22)

### Milestone 3: Route A/B + decision note
**Goal:** Banked same-weights/two-routes comparison; keep-or-drop decision recorded.
**Estimated:** 0 LOC (+ row-notes updates)
**Duration:** ~1–2 hours

**Tasks:**
1. `ailang eval-suite --models lyceum-glm-5-3-flash,or-glm-5-3-flash --tier smoke --langs ailang`
2. If smoke shows a faithful route (same failure-shape class, zero systematic truncation),
   run `--tier core` on the same pair (~$2.5 at blended prices; N=1 core precedent applies)
3. Decision note in the rows: seat = 429-fallback / EU-residency lane / opt-in only;
   roster changes require Mark ratification per house rules

**Acceptance Criteria:**
- [ ] AC8: A/B banked with a keep/drop decision recorded in row notes
- [ ] No suite displacement without core evidence + Mark ratification

## Success Metrics
- All tests passing: ✅
- `make lint` + `make fmt` + `make check-file-sizes` clean: ✅
- AC1–AC8 from the design doc: all checked
- Documentation: `cmd/ailang/help.go`, env-vars reference updated

## Dependencies
- `LYCEUM_API_KEY` (present in `~/.zshenv`, verified 2026-09-03)
- M1 → M2 → M3 strictly sequential

## Open Questions
- None blocking. (D4 ratified; slugs verified; prices banked with provenance labels.)

## Notes
- The openai transport is NOT modified in this sprint except via the M1 contingency
  (additive struct field only, mirroring an existing pattern).
- Lyceum rows start opt-in: not in any suite, not in the rotation. Suite entry is a
  Phase-3-decision + Mark-ratification matter, out of this sprint's scope.
- Phase 0 spend: <1,000 tokens (<$0.001). Smoke budget: ≤$0.20. Core (if run): ≤$3.00.