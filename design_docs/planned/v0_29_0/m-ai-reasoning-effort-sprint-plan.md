# Sprint Plan — M-AI-REASONING-EFFORT

**Design doc:** `design_docs/planned/v0_29_0/m-ai-reasoning-effort.md` (Rev-2, both authorized R2 fixes green)
**Sprint JSON:** `.ailang/state/sprints/sprint_M-AI-REASONING-EFFORT.json`
**Target:** v0.31.0 · **Lane:** extension (`internal/ai` provider-layer) · **Risk:** medium
**Planner HEAD:** `v0.30.0-116-ge6d5f85c9` (all premises live-verified, binaries assumed fresh)
**Est. sprint effort:** ~13.5h / ~4 days (~1550 LOC incl. tests). Live-smoke PARKED (metered follow-up, not in total).

---

## Milestone breakdown

| M | Name | AC covered | LOC | Hrs |
|---|------|-----------|-----|-----|
| **M0** | Code-audit + resolver-hook test (MANDATORY, fork-a) | M0-audit, AC2 (partial) | 120 | 1.5 |
| **M1** | Shared field + 5 typed errors + resolver + capability table + resolver-unit tests | AC1, AC4, AC5, AC6, AC7 | 420 | 2.5 |
| **M2** | Gemini `thinkingConfig` + conflict/MaxTokens + golden | AC8, AC10, AC14 | 240 | 2.0 |
| **M3** | OpenAI Responses (preserve body) + Chat native field | AC3, AC12, AC14 | 300 | 2.0 |
| **M4** | Anthropic `thinking` block + header/model gate + strict budget | AC9, AC11, AC5, AC14 | 300 | 2.5 |
| **M5** | OpenRouter **replace** Effort-wins logic + extend | AC13, AC7, AC14 | 260 | 1.5 |
| **M6** | Full AC15 matrix + AC14 golden suite + CHANGELOG + notify + boundaries | AC2, AC14, AC15, AC16, AC17 | 230 | 1.5 |

All 17 design-doc ACs are mapped (see `acceptance_criteria_coverage` in the JSON). M0 gates M1; M2–M5 fan out in parallel from M1; M6 joins them.

---

## Premise drift found (report to Mark / executor)

1. **MATERIAL DRIFT — OpenRouter already has a `reasoning_effort` pass-through.** The doc's premise "OpenRouter supports only `reasoning.max_tokens`" (Problem table + Verification Log) is **STALE at HEAD**:
   - `openrouter/types.go:38-41` — `reasoningField` already has **both** `MaxTokens` AND `Effort`.
   - `openrouter/chat.go:64-68` — already reads `Options["reasoning_effort"]` with a silent **"Effort-wins"** precedence and **no validation**.
   - **Consequence:** M5 is a **replace**, not a pure extend. The executor must delete the untyped Effort-wins branch and route through the shared resolver, or two conflicting precedence rules coexist. Captured as a first-class M5 acceptance criterion.

2. **Line-ref drift (immaterial):** Anthropic `MaxTokens=4096` defaulting is at `client.go:161-164` (doc cited `160-168` — within range). All other line refs (`provider.go:46-47`, `openai/responses.go:46-53`, `openai/types.go:88-94/117-120/7-16`, `gemini/types.go:80-89`, `anthropic/client.go:71-80/225-227`, `openrouter/types.go:26-37`) **hold exactly**.

No premise **errors** that block planning — the OpenRouter drift actually strengthens the case for one shared resolver.

---

## M0 machinery inventory (Verification Log — the executor must re-verify + extend)

| Piece | Evidence (file:line) | Status |
|-------|---------------------|--------|
| `AIError{Code,Message,Retryable}` + `NewAIError` | `internal/ai/errors.go:52-71` | REUSE |
| `CodeSchemaValidation="SchemaValidation"`, non-retryable | `errors.go:33`, `errors.go:95` (in `IsRetryable` false-set) | REUSE |
| Proven schema-reject construction | `openrouter/step.go:73`, `openrouter/chat.go:76` — `ai.NewAIError(ai.CodeSchemaValidation, msg, false)` | REUSE (sentinels wrap via this) |
| Capability-gate precedent | `configdriven/step.go:26` (`CodeModelNoVision`); `openai/step.go:93`, `gemini/step.go:35` (`CodeCapabilityNotSupported`) | PATTERN for NEW table |
| Request constructors (12 in-scope paths) | `Generate` + `Step` + `StreamStep` on openai/anthropic/openrouter/gemini | HOOK POINTS |
| Existing option parsers | `openai/responses.go:46-53` (`reasoning_effort`), `openrouter/chat.go:59-68` (`reasoning_effort` + `reasoning_max_tokens`), `seed` | REPLACE the OpenRouter Effort-wins branch |
| `ProviderRegistry` | `registry.go:22-50` — config-driven only; built-ins NOT stored | NOT the hook (targets are built-ins) |
| No existing reasoning/thinking capability table | (grep negative) | BUILD FRESH |

(ollama is a 5th built-in with `Generate`/`Step` but is **out of scope** — only the 4 named providers.)

---

## Reuse-vs-Replace decision (planner recommendation — executor verifies in M0)

- **REUSE** the `AIError` machinery. The 5 sentinels (`ErrInvalidReasoningEffort`, `ErrUnsupportedReasoningEffort`, `ErrConflictingReasoningConfig`, `ErrInvalidThinkingBudget`, `ErrReasoningBudgetExceedsMaxTokens`) should be constructed via `NewAIError(CodeSchemaValidation, msg, false)` — matching the existing non-retryable schema-validation convention. **Do not invent a new error type or code.** (Open question for executor: whether to also expose them as sentinel `error` vars for `errors.Is`, while the wire/`errors.As` shape stays `AIError`/`SchemaValidation`.)
- **REPLACE** the untyped OpenRouter `Effort-wins` branch (`chat.go:64-68`) with a call to the one shared resolver.
- **BUILD FRESH** the provider/model capability table (no existing one). Follow the `ModelNoVision` gate shape. It must start **empty of live-unverified models** — every `NEEDS-LIVE-SMOKE` model stays unregistered ⇒ UNKNOWN ⇒ rejects explicit controls (fail-loud). Capability entries are only added after the parked metered smoke verifies behavior.
- **ONE extension point:** a new package-level `internal/ai` resolver (e.g. `ResolveReasoning(req, provider, model) (decision, error)`) invoked by each of the 12 constructors **immediately after req validation and before** JSON marshal / `http.NewRequest` / the MaxTokens defaulting. This ordering is load-bearing for Anthropic (`client.go:161-164` must NOT run first) and is exactly what the M0 acceptance test pins.

---

## Guardrails honored

- **M0 first (fork-a):** code-audit + a network-free acceptance test proving all 12 paths invoke the ONE resolver (invalid control ⇒ typed error, zero HTTP hits) precedes any resolver design.
- **Live-smoke PARKED:** all four `NEEDS-LIVE-SMOKE` rows moved to follow-up M7 (out of sprint). Capability table ships empty of unverified models; UNKNOWN ⇒ reject.
- **Minimal-Frozen-Core:** additive `internal/ai` provider-layer change only. No compiler/eval-core/parser/types touch. M6 runs `make check-boundaries`.
- **AC #14 golden:** byte-identical unset bodies are a per-milestone AC on M2–M5 and a consolidated suite in M6, plus OpenRouter `reasoning_max_tokens`-alone body preservation.

---

## Follow-ups (NOT in this sprint)

- **M7 (metered live-smoke):** verify Gemini older-model no-op, Anthropic thinking header/model IDs + strict `max_tokens>budget` boundary, OpenAI Chat native field + allowlist, OpenRouter effort/off upstream pass-through. Only then add capability-table entries.
- `std/ai` `callWithReasoning(prompt, effort)` helper.
- eval-harness requested-effort baseline column.

---

## Execution status (sprint-executor, 2026-07-22) — COMPLETE

All 7 milestones green. One commit per milestone on `sprint/m-ai-reasoning-effort`:

| M | Commit | Status | Tests (file) |
|---|--------|--------|--------------|
| M0 | `08e2aa935` | ✅ | `internal/ai/reasoning_hookpoints_test.go` (all 12 paths invoke resolver, zero HTTP) |
| M1 | `a172d6dd1` | ✅ | `internal/ai/reasoning_test.go` (resolver matrix, -count=20) |
| M2 | `c52aef5bd` | ✅ | `internal/ai/gemini/reasoning_test.go` |
| M3 | `5927d49a2` | ✅ | `internal/ai/openai/reasoning_test.go` |
| M4 | `4af97e8de` | ✅ | `internal/ai/anthropic/reasoning_test.go` |
| M5 | `298cd72e3` | ✅ | `internal/ai/openrouter/reasoning_test.go` |
| M6 | (this commit) | ✅ | CHANGELOG + sprint JSON `status:completed` + `make check-boundaries` green |

**AC coverage:** all 17 satisfied network-free EXCEPT the parked live-smoke rows
(capability table ships empty → unknown model rejects; entries added only post-M7).
AC17 (docparse notify) intentionally deferred to the controller at finalize.

**Note on commit granularity:** the resolver, 5 sentinels, capability table,
typed field, and all 12 provider hook wirings are interdependent (they must
compile together), so they land in the M0 foundation commit; M1–M5 commits add
the per-layer test suites. Each commit compiles and its tests pass.

SPRINT_PLAN_PATH: design_docs/planned/v0_29_0/m-ai-reasoning-effort-sprint-plan.md
SPRINT_JSON_PATH: .ailang/state/sprints/sprint_M-AI-REASONING-EFFORT.json
