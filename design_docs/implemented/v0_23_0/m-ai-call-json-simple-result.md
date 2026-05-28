# M-AI-CALL-JSON-SIMPLE-RESULT: Result-returning variant of callJsonSimple

**Status**: Implemented (v0.23.0, 2026-05-28; commit 64f441cd)
**Target**: v0.23.0
**Priority**: P1 (Medium-High) — blocks docparse's prod AI-resilience work; transient provider 5xx on the highest-volume extraction path currently escapes as a bare HTTP 500
**Estimated**: 0.5 day (~60 LOC + tests; near-mechanical mirror of the existing `callJsonResult`)
**Dependencies**: None. Builds on the existing Result-variant AI surface (`callResult` / `callJsonResult`, v0.17.0).
**Bug Report**: msg_20260528_084937_623811a0 (from sunholo/docparse via `cli` inbox, 2026-05-28)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No new nondeterminism. Same AI call, just a Result wrapper around the existing `_ai_call_json_simple` path. |
| A2: Replayability | 0 | No trace-shape change beyond what `callResult`/`callJsonResult` already established. |
| A3: Effect Legibility | +1 | A transient provider 5xx on the no-schema JSON path currently escapes as an uncaught effect error (host crash → HTTP 500). This makes the failure a typed, catchable `Err(AIError)` — the effect's failure mode becomes legible at the call site instead of leaking out of the effect boundary. |
| A4: Explicit Authority | 0 | No authority change — same `{AI}` effect. |
| A5: Bounded Verification | 0 | No new analysis surface. |
| A6: Safe Concurrency | 0 | No concurrency change. |
| A7: Machines First | +1 | Agents that build extractors on `callJsonSimple` can now branch on `AIError.retryable` to retry-with-backoff vs surface-typed-error, rather than wrapping every call in host-level error handling that AILANG can't express. |
| A8: Minimal Syntax | +1 | No new syntax — one additional stdlib function + one builtin, mirroring an established pattern. |
| A9: Cost Visibility | 0 | No cost-model change. |
| A10: Composability | +1 | Brings the no-schema JSON path to parity with `callResult`/`callJsonResult`/`stepWithStream`. Closes the last gap where an AI surface function had no Result variant — completes the Result-returning family. |
| A11: Structured Failure | +1 | This is the entire point: converts an unstructured host crash into a structured `Err(AIError)` with a `retryable` flag. |
| A12: System Boundary | +1 | Provider 5xx is an external-system failure; surfacing it as a typed `AIError` at the AILANG boundary is exactly the explicit-boundary-crossing the axiom wants. |

**Net Score: +7** → **Decision: ✅ Proceed**

### Hard Violation Check
- [x] A1: No implicit nondeterminism
- [x] A3: No hidden side effects (the opposite — makes a hidden failure explicit)
- [x] A4: No ambient authority
- [x] A7: Optimises for machine-expressible error handling

## Problem Statement

The std/ai surface has Result-returning variants for two of its three non-streaming entry points:

| Legacy (crashes on provider failure) | Result variant (typed errors) | Builtin |
|---|---|---|
| `call(input) -> string` | `callResult(input) -> Result[string, AIError]` ✅ | `_ai_call_result` |
| `callJson(input, schema) -> string` | `callJsonResult(input, schema) -> Result[string, AIError]` ✅ | `_ai_call_json_result` |
| `callJsonSimple(input) -> string` | **MISSING** ❌ | `_ai_call_json_simple` (no Result variant) |

`callJsonSimple` is the no-schema JSON path. It exists separately from `callJson` because **`callJson` has a known large-response corruption bug** — so high-volume JSON extraction must use `callJsonSimple`. But `callJsonSimple` has no Result variant, which means:

- A transient provider 5xx (e.g. Gemini `500 Internal error encountered`) on a `callJsonSimple` call **escapes as an uncaught `{AI}` effect error**, crashing the host with `E_AI_CALL_ERROR`.
- Callers cannot retry-with-backoff (no `retryable` flag to branch on) and cannot surface a typed error (HTTP 502/503) — the failure becomes a bare HTTP 500.

### Concrete production impact (docparse)

`sunholo/docparse`'s `direct_ai_parser.ail`:
- `parsePdf → aiGetPageCount` used `call()` — **already mitigated** by switching to `callResult` (ships in ailang_parse 0.20.1).
- `parsePdfOnePage` uses `callJsonSimple()` — the **highest-volume** call (one per PDF page) — and **cannot be mitigated** because there's no `callJsonSimpleResult`.

Observed 2026-05-27 on prod `docparse.ailang.sunholo.com`: a Gemini 500 during page extraction → `E_AI_CALL_ERROR` → bare HTTP 500 on `/api/v1/parse`. The docparse team's resilience work (`design_docs/planned/v0_11_0/v0_11_0_ai_error_resilience.md` in their repo) is blocked on this one missing function.

## Goals

**Primary goal:** `std/ai.callJsonSimpleResult(input: string) -> Result[string, AIError] ! {AI}` — same raw-JSON-string semantics as `callJsonSimple`, but transient provider failures return `Err(AIError)` with a `retryable` flag instead of crashing the host.

**Success metrics:**
1. `callJsonSimpleResult("...")` returns `Ok(jsonString)` on success — byte-identical to what `callJsonSimple` would return.
2. A provider 5xx returns `Err(AIError{code, message, retryable})` — never crashes the host.
3. `AIError.retryable` is `true` for transient codes (RateLimit, Timeout, 502, 503) and `false` for caller-side codes — matching `callResult`'s existing classification.
4. docparse's `parsePdfOnePage` can switch from `callJsonSimple` to `callJsonSimpleResult` and add retry-with-backoff with no other changes.

## Solution Design

### Overview

Mirror the existing `callJsonResult` implementation exactly, minus the schema parameter. Three small additions:

1. **Builtin** `_ai_call_json_simple_result` in `internal/builtins/ai_step.go` (NumArgs: 1, Effect: AI) — mirror `registerAICallJsonResult()` but call the no-schema AI path.
2. **stdlib wrapper** `callJsonSimpleResult` in `std/ai.ail` — one export func.
3. **Impl** reuses the same AIError-mapping the other Result builtins use (`internal/ai/errors.go` already classifies provider errors → `AIError` with `retryable`).

### Implementation plan

**Builtin (`internal/builtins/ai_step.go`, ~35 LOC):**

```go
func registerAICallJsonSimpleResult() {
	err := RegisterEffectBuiltin(BuiltinSpec{
		Module:  "std/ai",
		Name:    "_ai_call_json_simple_result",
		NumArgs: 1, // input: string  (no schema, unlike _ai_call_json_result)
		Effect:  "AI",
		Type:    makeAICallSimpleResultType, // (string) -> Result[string, AIError] ! {AI}
		Impl:    aiCallJsonSimpleResultImpl,
		Metadata: &BuiltinMetadata{ /* mirror _ai_call_json_result; Since: v0.23.0 */ },
	})
	if err != nil { panic("...") }
}
```

The `Impl` (`aiCallJsonSimpleResultImpl`) is the no-schema sibling of `aiCallJsonResultImpl` — it calls the same underlying provider request as `_ai_call_json_simple` (no schema enforcement) but routes provider errors through the existing `internal/ai/errors.go` AIError mapper rather than panicking. The type-builder `makeAICallSimpleResultType` is `(string) -> Result[string, AIError] ! {AI}` (same shape as `makeAICallResultType`, which is also 1-arg — reuse it if it already produces that signature).

**stdlib (`std/ai.ail`, ~6 LOC):** insert right after `callJsonResult`:

```ailang
-- callJsonSimpleResult: Result-returning variant of callJsonSimple.
-- Same no-schema raw-JSON-string output, but typed errors via Err(AIError).
-- Prefer this over callJsonSimple on the high-volume extraction path so a
-- transient provider 5xx returns a catchable Err(AIError{retryable}) instead
-- of crashing the host with an uncaught {AI} effect error.
export func callJsonSimpleResult(input: string) -> Result[string, AIError] ! {AI} =
  _ai_call_json_simple_result(input)
```

**Wire the registration** into the same init path that registers `_ai_call_result` / `_ai_call_json_result`.

### Files to modify

| File | Change | LOC |
|---|---|---|
| `internal/builtins/ai_step.go` | `registerAICallJsonSimpleResult` + `aiCallJsonSimpleResultImpl` + type-builder + add to init | +40 |
| `std/ai.ail` | `callJsonSimpleResult` export | +6 |
| `internal/builtins/ai_step_test.go` (or sibling) | Success path + provider-5xx → Err(AIError) path + retryable-flag assertion | +30 |
| `examples/runnable/ai_call_json_simple_result.ail` | Worked example: call + match on Result + retry sketch | +20 |
| `examples/manifest.json` | Register the new example | +3 |
| `prompts/` active teaching prompt | Add `callJsonSimpleResult` to the AI-effect Result-variant list | +2 |
| `changelogs/v0.10-current.md` | Entry | +8 |

### Stretch goal (separate, do NOT block the primary on it)

The bug report's item #2: "have serve-api auto-map an uncaught `E_AI_CALL_ERROR` → HTTP 502 so passthrough services don't each reimplement this." This is a serve-api concern (`internal/apiserver/`), orthogonal to the stdlib function, and should be its own design doc if pursued. The primary `callJsonSimpleResult` unblocks docparse without it. **Recommendation: defer the stretch; ship the function.**

## Conflict Surface

Not applicable in the parser/typechecker sense — this adds a new builtin + stdlib function, touching no existing syntax or type rules. The only "surface" is the builtin-registration init path: adding `_ai_call_json_simple_result` must not collide with existing builtin names (verified: no such name exists) and must be added to the same registration sequence as its siblings so it's available in all execution paths (REPL, file, eval-harness). The `builtin_types.golden` test in `internal/pipeline/testdata/` will need regenerating to include the new builtin's type signature.

## Success Criteria

- [ ] `ailang run` of a program calling `callJsonSimpleResult` returns `Ok(...)` on a successful provider call
- [ ] A simulated provider 5xx returns `Err(AIError{retryable: true})` and does NOT crash the host
- [ ] `AIError.retryable` classification matches `callResult`'s for the same error codes (shared mapper)
- [ ] New builtin appears in `internal/pipeline/testdata/builtin_types.golden` with the correct signature `(string) -> Result[string, AIError] ! {AI}`
- [ ] Test runs deterministic at `-count=20` (the AIError-mapping path must not have map-iteration nondeterminism)
- [ ] `examples/runnable/ai_call_json_simple_result.ail` runs end-to-end under `ailang run --caps AI`
- [ ] CHANGELOG entry + teaching-prompt update
- [ ] Reply to msg_20260528_084937_623811a0 with the shipped function signature so docparse can unblock

## Out of Scope

- The serve-api `E_AI_CALL_ERROR → HTTP 502` auto-mapping (stretch item #2 — separate doc if pursued)
- Fixing the underlying `callJson` large-response corruption bug (a separate known issue; `callJsonSimple` exists precisely to route around it)
- Any change to the legacy `callJsonSimple` (stays as-is for back-compat per the established "legacy functions unchanged" convention)

## Related Documents

- **Established pattern**: `callResult` / `callJsonResult` (v0.17.0) — the Result-variant family this completes. Defined in [std/ai.ail:161-170](../../../std/ai.ail) + [internal/builtins/ai_step.go:144-184](../../../internal/builtins/ai_step.go).
- **AIError mapper**: [internal/ai/errors.go](../../../internal/ai/errors.go) — the provider-error → typed-AIError classification this reuses.
- **Downstream consumer**: `sunholo/docparse` `design_docs/planned/v0_11_0/v0_11_0_ai_error_resilience.md` (their repo) — the blocked resilience work.

---

**Document created**: 2026-05-28
**Last updated**: 2026-05-28
