# M-OLLAMA-PER-MODEL-MAX-TOKENS — flow the registry's declared max_output_tokens to motoko

**Status**: Planned
**Target**: v0.25.0
**Priority**: P1 (principled completion of the landed truncation fix)
**Estimated**: ~1h (AILANG) + a small motoko_agent PR
**Dependencies**: builds on `fac848054` (the 16384 ollama floor)

## Problem Statement

The landed truncation fix (`resolveOllamaMaxTokens`, `fac848054`) floors the ollama `/v1`
`max_tokens` to 16384, which eliminated motoko's qwen3.6 truncation disengagement (21%→79% on the
disengaging benchmarks). But it's a **generic floor**, not the model's **declared** value:

- `models.yml` already declares `max_output_tokens: 32768` for motoko-local-qwen3.6 (the model's real
  strength), but that value **never reaches motoko** — motoko's `std/ai.step` uses the 4096 default, so
  it sent 4096; now the floor rescues it to 16384, still not the declared 32768.
- The floor also *overrides downward-declared* models (a model declaring 8192 gets floored to 16384) —
  a benign cap increase, but it doesn't honor the registry.

**Principle (user, 2026-06-19):** when we add a model it should default to the *model's* declared
strengths; overrides are the rare exception. The registry is the source of truth; the floor should be
a fallback, not the value.

## Goals
Make the registry's per-model `max_output_tokens` flow to motoko's ollama request, with the 16384 floor
demoted to a fallback (used only when no value is declared).

**Success metric:** motoko's wire request carries the model's declared `max_output_tokens` (32768 for
qwen3.6), verifiable via the HTTP-wire logger; AILANG-side unit tests pass.

## Solution Design
- **AILANG (dev):**
  1. `executor.Task`: add `MaxOutputTokens int` (per-request output budget; 0 = unset).
  2. `agent_runner_multi.go`: set `Task.MaxOutputTokens` from
     `GlobalModelsConfig.GetModel(modelName).MaxOutputTokens` (the registry value).
  3. `internal/executor/motoko/motoko.go`: when `task.MaxOutputTokens > 0`, append
     `AILANG_OLLAMA_MAX_TOKENS=<value>` to the motoko subprocess env. `resolveOllamaMaxTokens` already
     reads this env (override wins), so no change to the ollama path is needed.
- **motoko_agent (DRAFT PR):** add `AILANG_OLLAMA_MAX_TOKENS` to the `RuntimeProcess` `childEnv`
  allowlist (`src/tui/src/runtime-process.ts`) so the env reaches the ailang runtime — same pattern as
  the `MOTOKO_PERSIST_RETRIES` plumbing. Without this the env is scrubbed and the floor (16384) is used.

## Files to Modify
**AILANG:** `internal/executor/executor.go`, `internal/eval_harness/agent_runner_multi.go`,
`internal/executor/motoko/motoko.go` (+ a unit test).
**motoko_agent:** `src/tui/src/runtime-process.ts`.

## Success Criteria
- [ ] `executor.Task.MaxOutputTokens` set from the registry; motoko.go emits `AILANG_OLLAMA_MAX_TOKENS`.
- [ ] Unit test: motoko env includes `AILANG_OLLAMA_MAX_TOKENS=<registry value>` when declared.
- [ ] motoko PR allowlists the env (draft, verified-locally note).
- [ ] Analysis log + backlog updated.

## Non-Goals
- Changing the 16384 floor (kept as the fallback for any path that doesn't pass the env).
- The full-rotation re-measure (blocked on fresh post-fix rotation data).
