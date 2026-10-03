# Cancellable provider context for the AI effect: one enabler for #231 (abort a step) and #578 (AIHandler v2)

- **Date**: 2026-10-03
- **Class**: feature
- **Recommend**: design-doc (revise `planned/v0_29_0/m-agent-step-cancellation.md` instead of writing a new milestone)
- **Searched**: `AIHandler v2`, `cancellable provider context`, `IncompleteStream`, `context.Context`, `StepWithStream`, `Cancelled`, `#231`, `#578` across `design_docs/`. Found: `planned/v0_29_0/m-agent-step-cancellation.md` (#231, Status PLANNED, never started), `implemented/v0_32_0/m-recorded-stream-api.md:538,790` (defers both the cancellable context and an `IncompleteStream` code to "S2"). No doc specifies the interface change.
- **Estimate**: omitted (design-doc)

**The two issues need the same missing piece.** #231 (motoko_explore, P2) wants Ctrl+C or a stdin `{"type":"abort"}` to stop an in-flight `std/ai.step`. #578 (from PR #577) wants the runtime to cancel the provider stream after a latched `unencodable stream chunk` failure, and @arniwesth's comment adds a live case: free-tier OpenRouter accepting a request and never streaming, which hangs the loop forever. All three need a `context.Context` that reaches the HTTP request.

**Where it stops today (origin/dev `2d7a9a174`).** The providers already take a context: `http.NewRequestWithContext(ctx, …)` in `internal/ai/{anthropic,openai,gemini}/streamstep.go` and `chatgpt/client.go`. The layer above them drops it. `ai.Handler.StepWithStream` (`internal/ai/handler.go:269`) calls `StreamStep(context.Background(), …)` (about line 296). The effect-side interface `AIHandler.StepWithStream` (`internal/effects/ai.go:87`) has no context parameter. `m-agent-step-cancellation.md` "Lane A" assumes this wiring exists, but it does not specify the interface change. #578's comment thread also corrects the blast radius: no out-of-repo `AIHandler` implementers (motoko consumes `std/ai` from `.ail`). Only in-repo implementers are affected: 4 production handlers plus test doubles (`StubAIHandler`, `FixtureAIHandler`, …).

**Decisions to rule on.**

1. **Interface shape.** Either add `ctx` to every `AIHandler` method (a clean break, all in-repo), or add an optional `AIHandlerWithContext` capability interface (the pattern `AIHandlerWithRouting` already uses) and fall back to `Background()`. Recommend the clean break, since nobody outside the repo implements it.
2. **What owns cancellation.** Options: (A1) a process SIGINT/SIGTERM handler cancels the in-flight step's context, with no language surface; (A2) `stepCancellable(..., cancel: CancelToken)` composed with `asyncReadStdinLines`/`selectEvents`; (A3) a per-call idle/first-byte timeout for the hung-stream case, which neither A1 nor A2 bounds by itself. Recommend A1 + A3 first (no new syntax, and they close the hung-stream and Ctrl+C cases), then A2 if motoko still needs the stdin protocol.
3. **Error vocabulary.** Add `Cancelled` and `IncompleteStream` as first-class `AIError` codes instead of message prefixes. The reporter's evidence (#578 comment, 2026-08-04) is that prefix-matching became a compatibility trap in their own code. Today `unencodable stream chunk` is a public prose prefix.
4. **WASM.** `js && wasm` ruled out panic-based aborts. The context path must work under `js.FuncOf` callbacks (`AbortController` on the fetch).

#578's own gate said "do not start without demand evidence". #231's live Ctrl+C case and the OpenRouter hang now provide it. Recommend updating `m-agent-step-cancellation.md` with decisions 1-4 and tracking both issues against it.

Issues: https://github.com/sunholo-data/ailang/issues/231, https://github.com/sunholo-data/ailang/issues/578
