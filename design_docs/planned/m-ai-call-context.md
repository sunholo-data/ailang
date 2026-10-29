# M-AI-CALL-CONTEXT: a cancellable, deadline-carrying context for every AI provider call

**Status**: Planned
**Target**: v0.55.0 (tentative)
**Priority**: P1. Every other oracle in the runtime is bounded; this one is not. Today a hung provider can park a run for good, and Ctrl+C kills the process without the program's own summary.
**Estimated**: ~4 days (1 plumbing, 1 interrupt and deadline, 1 idle watchdog and latch cancel, 1 tests and docs)
**Dependencies**: None. It builds on `EffContext.GoCtx` (already set by the runner) and on the Ollama idle watchdog (`internal/ai/ollama/idlereader.go`, already shipped).
**Planner-Lane**: opus-required. It changes an interface across `internal/ai`, `internal/effects`, `cmd/wasm` and `cmd/ailang`, and changes the CLI's signal semantics.
**Tracks**: [#1259](https://github.com/sunholo-data/ailang/issues/1259), [#578](https://github.com/sunholo-data/ailang/issues/578), [#231](https://github.com/sunholo-data/ailang/issues/231)
**Verified against**: origin/dev `772703831` (2026-10-26, `std/VERSION` v0.54.18)
**Lane** (PROGRAM.md §4): **AILANG fix (core-floor)**. A blocking oracle call that ignores cancellation and deadlines is the floor being wrong, and only the runtime can repair it. Policy (abort protocols, retry, summaries) stays in extensions. See [Relationship to other docs](#relationship-to-existing-docs).
**Quorum**: triggers 1 (design-freeze items) and 2 (it overrides the CLI signal handler, which is shared machinery) both fire. Trigger 3 fires weakly: an additive trace field. The quorum is **required before sprint planning**. It was skipped for this draft at the requester's instruction.

---

## Problem Statement

Three open issues report three symptoms with one root cause.

| Issue | Symptom | Reporter |
|---|---|---|
| #1259 (P1) | `std/ai` calls have no deadline. A stalled endpoint hangs the caller. `std/net` has `--net-timeout`; `std/ai` has nothing. Two quorum reviewers blocked M-AI-DECIDE-SYSTEM-ONE on exactly this, and that doc routed its LLM arm through `std/net` to get a bound (`implemented/v0_40_1/m-ai-decide-system-one.md` Key Fact 15). | sprint-executor |
| #578 (P2) | After a recorded stream latches a fail-loud error, AILANG must **drain** the provider's stream to its natural end. Work is bounded (256 chunks / 1 MiB), but wall-clock time is not. Live adjacent case: free-tier OpenRouter accepts a request and never streams, so the loop parks for good (@arniwesth, #578 comment). | M-RECORDED-STREAM-API S1 |
| #231 (P2) | Ctrl+C or motoko's `{"type":"abort"}` during `std/ai.step` cannot stop the call. The workaround is SIGKILL, which loses the `run_summary` event. | motoko_explore |

**Root cause:** no `context.Context` reaches the provider's HTTP request from the AI effect. The providers themselves accept a context and honour it. The layer above them drops it:

1. `effects.AIHandler` (`internal/effects/ai.go:27-88`) has **no context parameter** on any method. Neither do its capability extensions: `AIHandlerWithImageRefs` (`ai_image.go:22`) and `AISpeechHandler` (`ai_speech.go:19`).
2. The only production adapter, `ai.Handler`, therefore calls every provider with `context.Background()`. There are **6 sites**, and they serve all 15 handler-routed AI ops (audit below).
3. Nothing derives a deadline. There is no `--ai-timeout`, no env var and no `ctx.AI` bound. The hosted-provider HTTP clients are `http.DefaultClient` (`Timeout: 0`).
4. `ai.ClassifyError` maps `context.Canceled` to `Timeout` with `retryable=true` (`internal/ai/errors.go:285-291`). A user abort, if one could be delivered, would be reported as a transient timeout and invite a retry.

The plumbing already exists one layer up. `EffContext.GoCtx` (`internal/effects/context.go:98`) is set by the runner (`internal/runner/run.go:467`, `batch.go:48`), the REPL (`repl.go:100,210`) and the MCP gate (`apiserver/mcp_auth.go:209`). `std/net` and `std/stream` already honour it through `requestContext` (`net_authorize.go:365`, used at `net.go:88,181,425`, `stream_sse.go:75,328`, `stream_ndjson.go:85`). **The AI effect is the only network oracle that does not.**

### Provider-path audit (every path in `internal/ai` + the AI effect)

**Layer 1: provider methods (21 entry points, 8 packages). All 21 take `ctx`. All 20 that perform I/O forward it to the HTTP request.**

| Package | Method(s) | Takes ctx | ctx reaches request via | HTTP client (whole-call Timeout) |
|---|---|---|---|---|
| anthropic | `Generate` client.go:190, `Step` step.go:38, `StreamStep` streamstep.go:41 | yes ×3 | `DoJSON` client.go:302, step.go:72; `NewRequestWithContext` streamstep.go:74 | `http.DefaultClient` (0), client.go:66 |
| openai | `Generate` client.go:76, `Step` step.go:88, `StreamStep` streamstep.go:42 | yes ×3 | `DoJSON` chat.go:91, responses.go:83, step.go:130; streamstep.go:84 | `http.DefaultClient` (0), client.go:63 |
| openrouter | `Generate` client.go:171, `Step` step.go:34, `StreamStep` streamstep.go:37 | yes ×3 | `DoJSON` chat.go:126, step.go:129; streamstep.go:136 | `http.DefaultClient` (0), **client.go:162** |
| gemini | `Generate` client.go:127, `Step` step.go:30, `StreamStep` streamstep.go:41, `Speech` speech.go:65 | yes ×4 | `DoJSON` generate.go:87, step.go:71, speech.go:89; streamstep.go:72 | `http.DefaultClient` (0), client.go:73, :98 |
| chatgpt | `Generate`/`Step` → `StreamStep` client.go:76,81,87 | yes ×3 | `NewRequestWithContext` client.go:99 | own client, **10 min** (`DefaultTimeout`, client.go:22, :65) |
| ollama | `Generate` client.go:120, `Step` step.go:282, `StreamStep` streamstep_provider.go:25 | yes ×3 | `ollamaapi` Chat/Generate (step.go:454, client.go:195); `/v1` via openai client (step.go:351) | native: `http.DefaultClient` + rig lease (client.go:91) with ctx deadline 300 s (`ollamaCallContext`, step.go:64); `/v1` buffered **300 s** (step.go:337); `/v1` streaming `Timeout: 0` + TTFT 600 s / idle 120 s / hard 3600 s watchdog (streamstep.go:268, idlereader.go) |
| configdriven | `Generate` provider.go:78, `Step` step.go:20 | yes ×2 | `DoJSON` provider.go:175 (`Step` returns typed errors, no I/O) | own client, **60 s** (provider.go:58) |
| (shared) | `ai.DoJSON` httpjson.go:56 | yes | `NewRequestWithContext` httpjson.go:57 | nil → `http.DefaultClient`, **httpjson.go:71** |

Pre-request auth lookups that ignore any caller context: 3 sites, all gemini (`client.go:181` `gcpauth.AccessToken(context.Background())`, `client.go:195` `config.CloudProject(context.Background())`, `prepared.go:23`).

**Layer 2: the `ai.Handler` adapter. Nothing threads a caller context to the provider: 6 `context.Background()` sites, plus 2 context-taking methods with no callers.**

| Site | Serves AIHandler methods |
|---|---|
| `internal/ai/handler.go:101` `Call` | `call`, `callResult` |
| `internal/ai/handler.go:184` `CallJson` | `callJson`, `callJsonResult`, `callJsonSimple`, `callJsonSimpleResult` |
| `internal/ai/handler.go:298` `StepWithStream` (streaming providers) | `stepWithStream`, `stepWithStreamRecorded` |
| `internal/ai/handler.go:319` `StepWithCache` | `step` (delegates, :250), `stepWithCache`, and the `StepWithStream` fallback for non-streaming providers (:281) |
| `internal/ai/handler_image.go:75` `generateImage` | `callImage`, `callImageBase64`, `callImageWithRefs`, `callImageBase64WithRefs` |
| `internal/ai/speech.go:83` `CallSpeech` | `callSpeech` |
| `handler.go:203` `CallWithContext`, `:220` `GenerateWithDetails` | Take ctx, but no production caller (V6). `CallWithContext` also drops `Attribution`, a drift from `Call`. |

**Layer 3: AI effect ops. 17 registered; 15 are context-less, 2 are already cancellable.**

- **15 through `AIHandler`, context-less.** `ai.go:384-386` (3), `ai_step.go:49-55` (7), `ai_image.go:28-31` (4), `ai_speech.go:79` (1). They reach the handler through 14 `ctx.AI.*` call sites in `ai.go`, `ai_step.go`, `ai_stream_core.go`, `ai_image.go` and `ai_speech.go`.
- **2 through the Stream effect, already honouring `GoCtx`.** These are `streamCall` and `callStream` (`cmd/ailang/configdriven_streaming.go:25-26`), which use `effects.StreamSSEPost` with `requestContext` (`stream_sse.go:328`), plus the Stream defaults of idle 60 s and max duration 5 min (`stream_context.go:85-86`). **This path is the in-repo precedent: an AI call that is already cancellable and bounded.**

**Audit counts:** 21 provider entry points (21 take ctx; 20 do I/O, all forward it). 6 adapter sites drop it. 15 of 17 AI ops cannot be cancelled. HTTP client bounds come in 5 different shapes: unbounded `DefaultClient` ×7 sites, 60 s, 10 min, 300 s, and the 120/600/3600 s watchdog. They are spread over 5 files and none of them is visible in a trace.

### What already mitigates, and why it is not enough

- **Ctrl+C on `ailang run` already terminates.** `InstallWorkerSignalHandler` (`internal/effects/terminal_lifecycle.go:63-89`, installed unconditionally for CLI runs at `run.go:640-648`) restores terminals, closes owned workers, flushes IO, writes the trace and exits 130/143. It works from the signal goroutine without waiting for the evaluator (M-BACKGROUND-WORKER-LIFECYCLE, v0.54.4). This is the **partial mitigation**: the process dies cleanly, but the AILANG program never sees an outcome. A driver whose `run_summary` is AILANG code (motoko) still loses it. Both `implemented/v0_54_0/m-terminal-ui-native-input.md` ("does not promise to cancel arbitrary AI/network callbacks") and `implemented/v0_54_4/m-background-worker-lifecycle.md` ("in-process AI cancellation remains separate") exclude #231 explicitly. This doc is that separate piece.
- **Ollama bounds its own calls.** Those bounds are provider-private, have Ollama-specific semantics (`AILANG_OLLAMA_HTTP_TIMEOUT_SEC` changes meaning under streaming), and do not reach hosted providers.

**Impact:** motoko and any AILANG agent driver (no graceful abort; a hung free-tier stream hangs the run), package authors who want a bounded `std/ai` call (today they bypass `std/ai` for `std/net`), and the #578 latch (wall-clock bounded by the provider, not by AILANG).

---

## Goals

**Primary goal:** every AI effect op runs under one per-call `context.Context`. That context derives from the run's `GoCtx`, can be cancelled by the run's interrupt, carries an explicitly declared deadline when one is configured, and reaches the provider's HTTP request. Cancellation and deadline outcomes come back as typed `Err(AIError)` values.

**Success metrics:**
- `context.Background()` in the AI handler and effect layers: **6 → 0** (`grep -n "context.Background()" internal/ai/handler*.go internal/ai/speech.go internal/effects/ai*.go`). Gemini's auth sites are tracked separately (Deferred Decisions).
- AI ops that can be cancelled: **2/17 → 17/17**.
- SIGINT during a blocked `step`: today the process exits 130 and no AILANG code runs. After: `step` returns `Err({code: "Cancelled"})` in **< 1 s**, the program's own summary line is printed, and the exit status is still 130.
- A hung provider with `--ai-timeout 2s` returns `Err({code: "Timeout", retryable: true})` at 2 s ± 0.5 s. Today it never returns.
- **With no flag and no env set, the hosted-provider wire request is byte-identical** and no new deadline exists (negative control).

---

## High-Impact Decisions

| # | Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|---|---|---|---|---|---|
| D1 | **Interface shape:** a clean break that adds `ctx context.Context` as the first parameter of every `AIHandler` method and its capability interfaces (alternatives below) | It sets the contract for every handler, test double and the WASM host | human | design | high |
| D2 | **Interrupt semantics:** the first SIGINT/SIGTERM cancels an *AI-scope* context and gives the evaluator a bounded grace window, armed only when an AI call is in flight. A second signal, or grace expiry, takes today's hard path. The exit status stays 130/143. | It changes shared CLI machinery (`InstallWorkerSignalHandler`) that terminal and worker lifecycles also depend on | human | design | high |
| D3 | **No default deadline for hosted providers.** `--ai-timeout` and `--ai-idle-timeout` are opt-in. Existing provider-private bounds keep their values but become declared and traced. | It is the determinism and "only evals may restrict thinking" boundary. A wrong default silently truncates reasoning. | human | design | high |
| D4 | **Error vocabulary:** add `Cancelled` (`retryable=false`). A deadline stays `Timeout` (`retryable=true`). Classification uses `context.Cause`, not error strings. | It is public `AIError.code` vocabulary that consumers switch on | human | design | med |
| D5 | **Surface:** two flags plus two env vars registered in `internal/config` (`AreaAI`). An invalid value is a hard startup error. | It adds CLI and env routes counted by the simplicity audit | human | design | med |
| D6 | **The #578 latch cancels the in-flight call.** The latched `Internal` error still wins. The drain bound stays as defence in depth. | It closes #578's wall-clock residual without changing the recorded-stream contract | agent | compile | low |
| D7 | **Generalise the Ollama idle watchdog** into `internal/ai`, reused by all SSE providers, rather than writing a second watchdog | One concept, one implementation ("verify the seam") | agent | compile | med |

### Design Freeze

- [ ] **D1.** Clean-break `ctx` parameter. Recommended; alternatives in §Interface alternatives.
- [ ] **D2.** Grace window **2 s**, a constant with no flag. That matches the per-owner shutdown deadline in M-BACKGROUND-WORKER-LIFECYCLE. The window is armed only with an AI call in flight. Interrupt cancels the **AI scope**, not `GoCtx`, so Net calls and workers used during cleanup keep working inside the grace window.
- [ ] **D3.** No default total or idle deadline for hosted providers. The existing provider-private bounds (chatgpt 10 min, configdriven 60 s, the Ollama windows, Stream idle 60 s / 5 min) keep their values in this milestone. Ruling requested: should configdriven's 60 s whole-response cap be lifted? It was introduced in `51fa7c27f` with no recorded rationale, and it cuts long reasoning today.
- [ ] **D4.** The new code is spelled `Cancelled`, matching the existing PascalCase codes and the `m-agent-step-cancellation` text. `IncompleteStream` is **not** decided here (Non-Goals).

---

## Solution Design

### Overview

Every AI call follows one path:

```
run/batch/REPL/embedder ──▶ EffContext.GoCtx            (exists today)
                              │
                              ├─▶ aiScope = WithCancelCause(GoCtx)       (new; shared across Clone/WithBudget,
                              │                                         cancelled by interrupt)
AI op ──▶ AIContext.begin(op) ──▶ callCtx = aiScope
                              │      + WithTimeoutCause(total)          (if declared)
                              │      + per-call cancel (latch)          (stream ops)
                              │      + in-flight registration            (for D2's grace arming)
                              ▼
          AIHandler.<Method>(callCtx, …) ──▶ provider.<Method>(callCtx, …) ──▶ http.NewRequestWithContext
                              │                                   (+ idle watchdog transport on SSE paths)
                              ▼
          AIContext.end: if callCtx.Err() != nil → AIError from context.Cause(callCtx)
                         record bounds + cause on the AI trace event
```

### Interface change (D1) and alternatives

**Recommended: a clean break.**

```go
type AIHandler interface {
    Call(ctx context.Context, input string) (string, error)
    CallJson(ctx context.Context, input, schema string) (string, error)
    CallImage(ctx context.Context, prompt, outputPath, options string) (string, error)
    CallImageBase64(ctx context.Context, prompt, options string) (string, error)
    Step(ctx context.Context, model string, messages []ai.Message, tools []ai.ToolSchema) (*ai.Response, error)
    StepWithCache(ctx context.Context, model string, messages []ai.Message, tools []ai.ToolSchema, bps []ai.CacheBreakpoint) (*ai.Response, error)
    StepWithStream(ctx context.Context, model string, messages []ai.Message, tools []ai.ToolSchema, bps []ai.CacheBreakpoint, onChunk func(ai.StreamChunk)) (*ai.Response, error)
}
// AIHandlerWithImageRefs, AISpeechHandler: same treatment.
```

Blast radius (V7, V8): **4 production implementers.** These are `ai.Handler`, `WasmAIHandler` (`cmd/wasm/effects.go`), `StubAIHandler` and `FixtureAIHandler`, plus the `AIContext` wrapper and **7 test doubles**. **Zero out-of-repo implementers are possible**: the interface lives under `internal/` and Go's internal-package rule forbids importing it from outside the module. @arniwesth confirmed motoko implements none (#578 comment). `CallWithContext` and `GenerateWithDetails` become redundant. Delete `CallWithContext` only after re-running the zero-caller grep (V6), per coding-standards "unused" rule. `GenerateWithDetails` is used by a test and named in `internal/mission/quorum/call.go`, so keep it.

| Alternative | Verdict |
|---|---|
| **B. Optional `AIHandlerWithContext` capability** (the `AIHandlerWithRouting` pattern, `ai.go:101`), falling back to `Background()` | **Rejected.** A handler that does not implement it would silently ignore a declared deadline, which breaks "no silent fallbacks" (CLAUDE.md §2). It also keeps two code paths alive for good, and the motive for the pattern (out-of-repo implementers) does not exist (V8). |
| **C. Store the context on the handler** (`SetContext` before each call) | **Rejected.** It is hidden mutable state on a handler that `AIContext`'s lazy and per-call-model resolution (`ai_lazy.go`, `ai_providers.go`) may swap between calls. It also races the signal goroutine. #578 asked for the context *through* `StepWithStream`. |
| **D. One `AICall` request struct replacing the 7 methods** | **Deferred.** It is cleaner long-term but doubles the diff, and it is not needed to fix the root cause. |

### Who cancels: four causes on one context

| Cause | Mechanism | `AIError` | Issue |
|---|---|---|---|
| Run interrupt (SIGINT/SIGTERM, first signal) | `InstallWorkerSignalHandler` calls `effCtx.InterruptAI()`, which cancels `aiScope` with cause `ErrAIInterrupted` and returns whether a call is in flight. If one is, the hard path is deferred by the 2 s grace window. A second signal or grace expiry runs today's hard path unchanged. | `Cancelled`, `retryable=false` | #231 |
| Declared total deadline | `context.WithTimeoutCause(…, ErrAIDeadline)` per call | `Timeout`, `retryable=true` | #1259 |
| Declared idle gap (SSE paths only) | Shared watchdog transport cancels the per-call context with cause `ErrAIIdle` | `Timeout`, `retryable=true` (message names the window) | #578 hang case |
| Recorded-stream latch | `aiStreamCore` calls the per-call `cancel(errStreamLatched)` at the latch (`ai_stream_core.go:76-80`) | The latched `Internal` error stands, unchanged | #578 |
| Embedder context (MCP gate, future serve-api request) | Inherited through `GoCtx` | `Timeout` if deadline exceeded, `Cancelled` if cancelled | free |

After every call, `AIContext` checks `callCtx.Err()`. When it is set, `AIContext` classifies by `context.Cause(callCtx)`, so the result never depends on which wrapping a provider chose. `ai.ClassifyError` changes in one place: `errors.Is(err, context.Canceled)` maps to `Cancelled` instead of `Timeout` (`errors.go:286`). `CodeCancelled` joins `IsRetryable`'s non-retryable list.

**Why interrupt cancels an AI scope and not `GoCtx`.** `GoCtx` also parents owned workers (`worker_owner.go:125`) and every Net request. Cancelling it would kill exactly what a cleanup handler might use inside its grace window. An AI scope keeps the change to the oracle the issues are about (A10). After an interrupt, every later AI call in the run fails fast with `Cancelled`, so a program that loops cannot start a new provider call during grace.

**The stdin `{"type":"abort"}` protocol.** This is a correction to the Lane B text in the step-cancellation doc. AILANG evaluation is single-threaded. `spawn` is a reserved keyword but does not parse (V14). So no AILANG code can watch `asyncReadStdinLines` *while* `step` is blocked, because the `selectEvents` handler that would read the abort line runs only after `step` returns. Lane B therefore has two routes. (a) The host that writes `{"type":"abort"}` (motoko's frontend) also sends SIGINT to the child, which needs nothing beyond this doc. (b) The runtime-side `CancelToken` (step-cancellation A2) becomes one more cancel cause on the `aiScope` context built here: purely additive, and this doc does not choose it.

### Deadline surface (D3, D5)

| Flag | Env (registered in `internal/config/ai.go` `aiVars`, `AreaAI`) | Meaning | Default |
|---|---|---|---|
| `--ai-timeout <dur>` | `AILANG_AI_TIMEOUT` | Total wall-clock per AI call, from dispatch to final byte | **unset = no AILANG-imposed bound** |
| `--ai-idle-timeout <dur>` | `AILANG_AI_IDLE_TIMEOUT` | Max silence between response bytes on streaming paths (also before the first byte) | **unset** |

- **Parsing:** a positive Go duration. Anything else is a **hard startup error** naming the flag or env, mirroring `--net-timeout` (`internal/runner/handlers.go:186-189`). That deliberately differs from the Ollama vars' "malformed keeps the default".
- **Precedence:** flag over env over the provider-default table (below) over unbounded. An embedder `GoCtx` deadline always applies as well; context deadlines compose as a minimum.
- **Flag reach:** `ailang run` and `serve-api`, wired where `--ai` is wired (`cmd/ailang/main_run.go:69-73`). Both are stored on `AIContext` as `Bounds{Total, Idle, Source}`, the analogue of `ctx.Net.Timeout` that #1259 asked for.
- **Provider-default table.** The existing hard-coded whole-client timeouts move into **one declared table** in `internal/ai`: chatgpt 10 min, configdriven 60 s. Each becomes a context deadline that applies only when no flag or env is set. HTTP clients become `Timeout: 0` on one shared, explicitly constructed `ai.HTTPClient()` that replaces the 7 `http.DefaultClient` uses. The client gets connect bounds only; `http.DefaultTransport` already gives a 30 s dial and 10 s TLS handshake, which cannot cut a running response. The Ollama windows stay provider-internal and registry-declared (`internal/config/ollama.go`). The table documents them, and as context deadlines they compose as a minimum with `--ai-timeout`. The Stream-effect AI path keeps the Stream context's own bounds.

**Why no default (the evidence).** The memory rule is "ONLY evals may restrict thinking", and a default must not silently cut long reasoning. Measured data argues against any value we could pick today:
- M-AI-DECIDE-SYSTEM-ONE's shadow run: the `glm-5.3-flash` control arm hit **3 timeouts in 17 calls at 30 s**, Net's default (`implemented/v0_40_1/m-ai-decide-system-one.md:422`). So copying `std/net`'s 30 s would truncate ordinary calls.
- The Ollama streaming work found that a 300 s buffered cap killed every long turn "at ~4m59s" (`internal/ai/ollama/streamstep.go:52-58`), and needed a 600 s TTFT for cold loads (`internal/config/ollama.go:32`).
- Hosted reasoning models can stay silent for minutes before the first byte on non-streaming paths. No hosted-provider TTFT or idle distribution has been measured in-repo (PENDING P1).

So the defaults stay unbounded, which is today's behaviour and introduces nothing new. Evals that want bounds set the flag or env explicitly. A future default needs a measured distribution, for example from OpenRouter Broadcast spans (`output_tokens.reasoning`, wall time per host).

### Determinism and replay (A1, A2)

A deadline makes the outcome depend on wall-clock time. AILANG makes that explicit in four ways:
1. **Declared, never inferred.** A bound exists only when a flag or env var sets it, or when it is an entry in the one provider-default table. No heuristic adapts a timeout at runtime.
2. **Recorded.** Each AI trace event gains an optional machine-readable `bounds` object: `{total_ms, idle_ms, source: flag|env|provider-default|embedder|none}`. When the call ended by cancellation, it also gets `cancel_cause: deadline|idle|interrupt|latched|embedder`. `RecordAIEffect` (`internal/trace/collector.go:344`) already marks AI events `Deterministic=false`. The deadline adds no new class of non-determinism to an op that is already an oracle. It makes the time-dependent part legible.
3. **A value, not a crash.** Outcomes are ordinary `Err(AIError)` values in the op's existing result type. `AIError.code` is a `string` (`std/ai.ail:66`), so **no stdlib type or interface changes**. The only `std/` edit is the code list in the comment at `std/ai.ail:59-62`, and the freeze gate hashes `ailang iface` output, not comments (V15).
4. **Replayable.** `--ai-stub-fixtures` already replays error outcomes, so a recorded `Cancelled` or `Timeout` replays deterministically. This was verified today (V13).

### Implementation Plan

**Phase 1: plumbing (no behaviour change)** (~1 day)
- [ ] Add `ctx` to `AIHandler`, `AIHandlerWithImageRefs`, `AISpeechHandler` and all 4 production implementers, the `AIContext` wrapper and the 7 test doubles.
- [ ] `AIContext` derives `callCtx` from `requestContext(effCtx)`. Reuse the helper (move it next to `GoCtx` in `context.go`; do not copy it). `ai.Handler` forwards `ctx` at all 6 sites.
- [ ] `WasmAIHandler.awaitPromise` selects on `ctx.Done()`.
- [ ] Gate: the golden wire-request tests are byte-identical with no flags set. `make test` is green.

**Phase 2: interrupt, deadline and vocabulary** (~1 day)
- [ ] `aiScope` on `EffContext`: a pointer shared by `Clone` and `WithBudget`, like `fsRoot`. Add `InterruptAI()` and in-flight accounting.
- [ ] Change `InstallWorkerSignalHandler` and both install sites (`run.go:641`, `batch.go:92`) to the two-stage interrupt with a 2 s grace. The exit status stays 130/143 even when the program finishes inside the grace window.
- [ ] Register `AILANG_AI_TIMEOUT` and `AILANG_AI_IDLE_TIMEOUT` in `internal/config/ai.go`. Add the flags to `run` and `serve-api` plus help text, with hard-error parsing.
- [ ] Provider-default table. Add the shared `ai.HTTPClient()` and drop the `DefaultClient` and whole-client `Timeout` uses.
- [ ] `CodeCancelled`, the `IsRetryable` entry and cause-based classification in `AIContext`. Fix `ClassifyError`'s `context.Canceled` arm.
- [ ] Add the trace `bounds` and `cancel_cause` fields.

**Phase 3: idle watchdog and latch** (~1 day)
- [ ] Move `newIdleTimeoutTransport` and its sentinels from `internal/ai/ollama/idlereader.go` to `internal/ai`. Ollama keeps its config resolvers and behaviour. Wire the watchdog into the anthropic, openai, openrouter, gemini and chatgpt `StreamStep` transports when `Idle > 0`.
- [ ] `aiStreamCore`: the latch calls the per-call cancel. The drain bound stays in place.

**Phase 4: tests, example, docs** (~1 day): see Success Criteria and Testing Strategy.

### Files to Modify/Create

**New files:**
- `internal/ai/callctx.go` (~120 LOC): `Bounds`, cause sentinels, the provider-default table, `HTTPClient()`.
- `internal/ai/idlewatch.go` (~250 LOC, moved): the generalised watchdog transport, from `ollama/idlereader.go`.
- `internal/effects/ai_scope.go` (~120 LOC): `aiScope`, `InterruptAI`, in-flight accounting, per-call derivation and classification.
- `examples/ai_step_interrupt.ail` (~25 LOC): the program in Example 1 (already type-checked).

**Modified files:**
- `internal/effects/ai.go` (+40/−20): interface, `AIContext` methods, `StubAIHandler`.
- `internal/effects/ai_step.go`, `ai_stream_core.go`, `ai_image.go`, `ai_speech.go`, `ai_stub_fixtures.go`, `ai_lazy.go` (+60/−30): pass `callCtx`; latch cancel.
- `internal/ai/handler.go`, `handler_image.go`, `speech.go` (+30/−30): forward ctx; remove `CallWithContext` (after V6 re-check).
- `internal/ai/errors.go` (+10/−3): `CodeCancelled`, `IsRetryable`, the `Canceled` arm.
- `internal/ai/httpjson.go`, `anthropic/client.go`, `openai/client.go`, `openrouter/client.go`, `gemini/client.go`, `chatgpt/client.go`, `configdriven/provider.go`, `ollama/client.go` (+20/−15): use the shared client; table entries replace whole-client timeouts.
- `internal/ai/{anthropic,openai,openrouter,gemini}/streamstep.go`, `chatgpt/client.go` (+25): idle watchdog transport.
- `internal/ai/ollama/idlereader.go`, `streamstep.go` (−200/+20): use the shared watchdog.
- `internal/effects/terminal_lifecycle.go` (+40/−5): two-stage interrupt.
- `internal/effects/context.go` (+10): `aiScope` field; `requestContext` relocated.
- `internal/runner/run.go`, `batch.go`, `handlers.go` (+40): grace exit status; bounds wiring.
- `internal/config/ai.go` (+20): two registry vars and getters.
- `internal/trace/collector.go` and `internal/trace` event types (+25): optional `bounds` and `cancel_cause`.
- `cmd/ailang/main_run.go`, `serve_api.go`, `help.go`, `ai_handlers.go` (+40): flags.
- `cmd/wasm/effects.go` (+30/−10): ctx parameters; `awaitPromise` select.
- `std/ai.ail` (comment only, +1): the code list gains `Cancelled`.
- `docs/docs/guides/ai-effect.mdx`, `docs/docs/reference/effects.md`, `docs/docs/guides/debugging.md` (+60): flags, codes, interrupt semantics.
- `changelogs/unreleased/2026-10-XX-ai-call-context.md` (new fragment).

---

## Examples

### Example 1: a driver turns an interrupted step into its own summary

Checked on v0.54.18 with `ailang check`. Run with `--ai-stub` (prints `run_summary status=done finish=stop`), and with `--ai-stub-fixtures` replaying `Cancelled` and `Timeout` (prints `status=aborted` and `status=deadline retryable=false`). It needs **no new syntax and no stdlib change**. After this milestone, a real SIGINT produces the `Cancelled` branch.

```ailang
module examples/ai_step_interrupt

import std/ai (step, Message, StepResult, AIError)

-- One user turn; tool list empty.
pure func userMsg(text: string) -> Message =
  { role: "user", content: text, tool_calls: [], tool_call_id: "", images: [] }

-- Map a step outcome to the run-summary line the driver emits.
-- "Cancelled" and "Timeout" are AIError codes (strings), not new types:
-- the stdlib interface is unchanged.
pure func summarize(r: Result[StepResult, AIError]) -> string =
  match r {
    Ok(res) => "run_summary status=done finish=${res.finish_reason}",
    Err(e) =>
      if e.code == "Cancelled" then "run_summary status=aborted"
      else if e.code == "Timeout" then "run_summary status=deadline retryable=${show(e.retryable)}"
      else "run_summary status=error code=${e.code}"
  }

export func main() -> () ! {IO, AI} =
  println(summarize(step("", [userMsg("Plan the next edit.")], [])))
```

### Example 2: CLI

**Before:** `ailang run --caps IO,AI --ai openrouter/<free-model> agent.ail`. A provider that accepts the request and never streams hangs forever. Ctrl+C exits 130, and `agent.ail` never prints its summary.

**After:**
```
ailang run --caps IO,AI --ai openrouter/<free-model> --ai-idle-timeout 120s --ai-timeout 30m agent.ail
# hung stream → Err({code:"Timeout", …}) after 120 s of silence; trace bounds.source="flag", cancel_cause="idle"
# Ctrl+C mid-step → Err({code:"Cancelled"}) < 1 s; program prints its summary; exit 130
# second Ctrl+C (or 2 s grace expiry) → today's hard path
```

---

## Success Criteria

- [ ] **AC1, abort mid-step.** Setup: `--ai` points at a local `httptest` server through `OPENAI_BASE_URL` (precedent: `cmd/ailang/openai_local_endpoint_test.go`); the server accepts and never answers. Run `examples/ai_step_interrupt.ail` and send SIGINT once the server has seen the request. Expected: stdout contains `run_summary status=aborted` within **1 s** of the signal, the server observes the connection closed within 1 s, the exit status is 130 and the trace file is written. SIGTERM behaves the same, with status 143.
- [ ] **AC2, hung provider hits the deadline.** Same hanging server, `--ai-timeout 2s`, no signal. `step` returns `Err({code:"Timeout", retryable:true})` at 2 s ± 0.5 s. The trace event carries `bounds.total_ms=2000`, `source="flag"`, `cancel_cause="deadline"`. Repeat with `AILANG_AI_TIMEOUT=2s`: `source="env"`. Set both: the flag wins.
- [ ] **AC3, idle gap.** The SSE server sends one chunk, then stalls. With `--ai-idle-timeout 1s`, `stepWithStream` returns `Timeout` with `cancel_cause="idle"` within ~1.5 s.
- [ ] **AC4, #578 latch.** Extend `TestAIStreamCoreMatrix` case `05_unencodable_first_and_middle` (`internal/effects/ai_stream_core_test.go:76`): the handler's `ctx.Done()` fires within one chunk of the latch. The outcome is still `Internal` with the `unencodable stream chunk` prefix (the contract is unchanged).
- [ ] **AC5, typed vocabulary.** `ClassifyError(context.Canceled)` returns `Cancelled` with `retryable=false`. `IsRetryable("Cancelled")` is false. A `DeadlineExceeded` still returns `Timeout` with `retryable=true`.
- **Negative controls:**
  - [ ] **NC1, no default bound.** With no flag or env: the per-call context has no deadline (`ctx.Deadline()` reports `ok=false`) for anthropic, openai, openrouter and gemini, and their golden wire requests are byte-identical to pre-change.
  - [ ] **NC2, a progressing stream is not cut.** The SSE server sends a chunk every 500 ms for 6 s, with `--ai-idle-timeout 1s` and no total. Result: `Ok`, all chunks delivered.
  - [ ] **NC3, a non-AI program keeps today's signal behaviour.** A program blocked in `readLine` gets SIGINT: exit 130 with no grace delay (< 200 ms), the same as v0.54.18.
  - [ ] **NC4, a second signal is immediate.** During the grace window, a second SIGINT takes the hard path at once.
  - [ ] **NC5, invalid value fails loudly.** `--ai-timeout 0`, `--ai-timeout banana` and `AILANG_AI_TIMEOUT=-1s` each exit non-zero at startup, naming the flag or env.
  - [ ] **NC6, interrupt scope does not leak.** A worker spawned before the interrupt still accepts `writeProcessStdin` during the grace window. A Net call in the cleanup path succeeds.
  - [ ] **NC7, mutation tests.** Revert the `handler.go:298` forward and AC1 must fail. Remove the latch cancel and AC4 must fail.
- [ ] `grep -n "context.Background()" internal/ai/handler*.go internal/ai/speech.go internal/effects/ai*.go` returns nothing.
- [ ] `make test`, `make lint`, `make verify-examples`, `make simplicity-audit` (flag and env deltas are expected and recorded), `make check-file-sizes`.
- [ ] Docs updated: the ai-effect guide, the effects reference, the debugging guide. Changelog fragment added. `examples/ai_step_interrupt.ail` added.

## Testing Strategy

**Unit:** `ai.ClassifyError` and `IsRetryable` rows; cause-to-code mapping in `ai_scope.go`; the provider-default table precedence matrix (flag/env/table/none × embedder deadline); config parsing (`internal/config`) including the hard errors; the generalised watchdog keeps Ollama's existing `idlereader` tests passing unchanged after the move.

**Integration:** AC1–AC3 and NC2–NC6 use a hermetic `httptest` provider (no key, no spend). AC1/NC3/NC4 run the built binary under a real signal. AC4 runs in `internal/effects`.

**Regression surface:** existing `internal/effects/ai_*_test.go` (all 7 doubles updated mechanically); `cmd/ailang/*dispatch_test.go`; Ollama `TestOllamaV1Timeout` and the streaming watchdog tests (must pass unchanged); golden wire tests per provider.

**Manual:** one motoko run on the rig with SIGINT mid-step, read with `ailang chains chat` to confirm the driver's `run_summary` appears.

---

## Conflict Surface

The change touches `internal/effects/` and the CLI entry (`cmd/ailang`, `internal/runner`). It adds no parser, typechecker or syntax changes.

**Positions touched:** the `AIHandler` method signatures; the `InstallWorkerSignalHandler` signal path; the `AIError.code` value set; the `run`/`serve-api` flag set; the AI trace event schema (additive).

| Position | What else lives there | Resolution |
|---|---|---|
| SIGINT/SIGTERM handler (`terminal_lifecycle.go:63`) | Native terminal restore (M-TERMINAL-UI-NATIVE-INPUT); owned-worker shutdown (M-BACKGROUND-WORKER-LIFECYCLE); trace flush | On the first signal, terminals are **restored immediately** (unchanged order). Only the *exit* waits up to 2 s, and only with an AI call in flight. Worker close runs on whichever exit path fires. Those docs' tests must still pass. |
| `context.Canceled` → `Timeout` (`errors.go:286`) | Ollama watchdog cancels surface typed sentinels, not bare `context.Canceled` (`idlereader.go` header); the MCP gate's `GoCtx` does not reach AI today | Changing the arm affects only causes that are new in this design. Verify with a grep for `context.Canceled` producers in `internal/ai` (V16). |
| `AIError.code` consumers | `std/ai.runTools` emits `Internal`; motoko matches codes; `--ai-stub-fixtures` replays arbitrary codes | Adding `Cancelled` is additive. A consumer that matches exhaustively would see a new string, but the type is `string`, so it cannot break type-checking. |
| `AIHandler` in **generated Go** (`internal/gen/golang/ai.go:77`, `examples/sim_stub/gen/game/ai_types.go`) | A **separate**, string→string interface emitted for compiled Go programs | **Out of scope, unchanged.** It is a distinct type in generated packages and never implements `effects.AIHandler`. |
| `GoCtx` consumers | Workers (`worker_owner.go:125`), Net, Stream, secrets (`secret.go:93`), OTEL spans (`ops.go:104`) | Interrupt cancels a **child** AI scope, never `GoCtx`, so none of these observe a change (NC6). |

**Programs that must still work:** `examples/ai_step_interrupt.ail` (normal path); the existing AI examples under `make verify-examples`; motoko's driver (`std/ai.stepWithStream` / `stepWithStreamRecorded`); the M-BACKGROUND-WORKER-LIFECYCLE exit reproductions; the M-TERMINAL-UI native-input SIGINT restore test.

**What deliberately changes:**
1. Ctrl+C during an AI call gives the program up to 2 s to finish. It used to die immediately.
2. Every AI call dispatched after an interrupt fails fast with `Cancelled`.
3. Anything that once produced `Timeout` from a bare `context.Canceled` now yields `Cancelled`. No such producer exists at HEAD (V16).
4. chatgpt and configdriven bounds move from client `Timeout` to context deadlines. The values are unchanged, but the error now carries `cancel_cause=deadline`.

---

## Axiom Compliance

| Axiom | Score | Justification |
|---|---|---|
| A1: Determinism | +1 | Today's hidden, scattered timeouts (5 shapes, untraced) become declared and recorded. No new default bound, so behaviour with nothing set is unchanged. |
| A2: Replayability | +1 | `bounds` and `cancel_cause` on trace events. Fixtures replay `Cancelled`/`Timeout` (V13). |
| A3: Effect Legibility | +1 | Cancellation and deadline become values of the AI effect's existing result type. Nothing is hidden. |
| A4: Explicit Authority | 0 | Authority is unchanged. The interrupt comes from the host, the deadline from the operator. |
| A5: Bounded Verification | +1 | AI calls gain an enforceable bound that tests can assert hermetically. |
| A6: Safe Concurrency | +1 | The signal goroutine cancels a context; it never runs AILANG code (M-TERMINAL-UI's rule kept). One owner for in-flight accounting. |
| A7: Machines First | +1 | A typed `Cancelled` code distinguishes an abort from a transient timeout, so automated retry stops misfiring. |
| A8: Minimal Syntax | 0 | No syntax. Two flags. |
| A9: Cost Visibility | +1 | A cancelled stream stops billing tokens instead of draining to completion (#578). The bound source is visible in the trace. |
| A10: Composability | +1 | One context composes with embedder deadlines, a future `CancelToken` and the existing Net/Stream `GoCtx` path. |
| A11: Structured Failure | +1 | `Cancelled` and `Timeout` are typed `Err` values, not process death. |
| A12: System Boundary | +1 | The provider HTTP boundary finally honours the runtime's lifetime. |

**Net score: +10.** Proceed.

**Hard-violation check:** [x] A1: no implicit nondeterminism (opt-in, declared, traced) · [x] A3: no hidden effects · [x] A4: no ambient authority · [x] A7: machine-readable codes and trace fields.

---

## Verification Log

| # | Claim | Instrument | Result |
|---|---|---|---|
| V1 | `ai.Handler` uses `context.Background()` at 4 + 2 sites | `grep -rn "context.Background()" internal/ai` | handler.go:101,184,298,319; handler_image.go:75; speech.go:83 (plus 3 gemini auth sites) |
| V2 | All 21 provider entry points take ctx; the I/O ones forward it | `grep -rn "func (.*) \(Generate\|Step\|StreamStep\|Speech\)(ctx"`; `DoJSON(ctx`/`NewRequestWithContext(ctx` per file | 21/21 take ctx; configdriven `Step` does no I/O; the other 20 forward it |
| V3 | `http.DefaultClient` sites | `grep -rn "http.DefaultClient"` in `internal/ai` (non-test) | httpjson.go:71; anthropic:66; openai:63; openrouter:162; gemini:73,98; ollama:91 (7) |
| V4 | Existing whole-call bounds | read | configdriven 60 s (provider.go:58); chatgpt 10 min (client.go:22); ollama 300 s / 600 / 120 / 3600 (step.go, config/ollama.go:31-32, idlereader.go:71) |
| V5 | **Negative:** no `--ai-timeout`, `AILANG_AI_TIMEOUT` or `AITimeout` exists | `grep -rn "ai-timeout\|AI_TIMEOUT\|AITimeout"` in `internal cmd docs/docs` | empty |
| V6 | **Negative:** `CallWithContext` has no caller | `grep -rn "CallWithContext"` (all files) | only its definition. `GenerateWithDetails` has a test caller, so keep it |
| V7 | Implementers of `AIHandler` | `grep -rn "func (.*) StepWithStream("` | 4 production (ai.Handler, WasmAIHandler, StubAIHandler, FixtureAIHandler) + `AIContext` + 7 test doubles |
| V8 | **Negative:** no `effects.AIHandler` implementer outside the module | interface under `internal/`; `grep -rln AIHandler` outside internal/cmd → only generated-Go `examples/sim_stub` (separate interface) | confirmed. Matches @arniwesth's #578 comment |
| V9 | 17 AI ops; 15 via `AIHandler`, 2 via Stream SSE with `requestContext` | `grep -rn 'RegisterOp("AI"'`; read configdriven_streaming.go, stream_sse.go:328 | confirmed |
| V10 | `GoCtx` is set by the runner, batch, REPL and MCP gate; Net/Stream use it | `grep -rn GoCtx` | run.go:467; batch.go:48; repl.go:100,210; mcp_auth.go:209; requestContext net_authorize.go:365 |
| V11 | CLI SIGINT today exits 130/143 without running AILANG code | read terminal_lifecycle.go:47-89, run.go:640-648, main_run_exec.go:97 | confirmed |
| V12 | `context.Canceled` classifies as `Timeout`, retryable | read errors.go:285-291 | confirmed |
| V13 | Fixtures replay error codes, including a not-yet-existing `Cancelled` | `ailang run --ai-stub --ai-stub-fixtures fx_Cancelled.json examples/ai_step_interrupt.ail` | prints `run_summary status=aborted`. `Timeout` prints `status=deadline retryable=false` |
| V14 | **Negative:** no in-language concurrency to watch stdin during `step` | `ailang check` on `let t = spawn { … }` | parse error (`spawn` is reserved, not implemented) |
| V15 | The stdlib freeze gate ignores comments | read tools/stdlib-iface-lib.sh:44-52 | it hashes `ailang iface std/<m>` output (signatures only) |
| V16 | **Negative:** no current producer of a bare `context.Canceled` into the AI classifier | read ollama/idlereader.go header ("never a generic context canceled"); AI never received `GoCtx` (V1) | confirmed for HEAD; re-run in sprint |
| V17 | **Negative:** `Cancelled` is not an existing `AIError` code | `grep -rn '"Cancelled"\|CodeCancel'` | only an unrelated UI label (internal/display/status.go:93) |
| V18 | `timeout` cannot be an AILANG record field (relevant to any future per-call option) | `ailang check` on `{ timeout: 5 }` vs `{ deadline_ms: 5 }` | `timeout` is a parse error (reserved keyword). `deadline_ms` passes |
| V19 | The example program type-checks and runs | `ailang check` + `ailang run --ai-stub` | ✓ No errors; prints `run_summary status=done finish=stop` |
| P1 | **PENDING:** hosted-provider TTFT/idle distribution, and whether OpenRouter keep-alive comments (`: OPENROUTER PROCESSING`) would reset a byte-level idle timer during the free-tier hang | external; OpenRouter Broadcast spans + a captured hung stream | **unmeasured.** It is why D3 keeps idle opt-in. If keep-alives reset the timer, `--ai-timeout` is the bound that catches that hang |

---

## Deferred Decisions

- Whether idle activity counts raw bytes or only semantic SSE events (P1 decides): agent, after the measurement.
- Passing an `AbortSignal` to the WASM JS callback as an extra trailing argument, versus abandoning the promise: agent may choose (JS ignores extra arguments).
- Threading ctx into gemini's 3 auth lookups: agent may choose (they are short, pre-request calls).
- The exact trace field names and placement (`EffectEvent.Bounds`): agent, as long as they are machine-readable and optional.
- Collapsing the 7 methods into one request struct (alternative D): human, in a later doc.

## Non-Goals

- **A default deadline.** It needs measured evidence (D3, P1).
- **`IncompleteStream` as a first-class code.** That is the open S2 error-surface ruling on #546/#578; @arniwesth has answered "yes, a code". It is orthogonal: this doc neither blocks nor decides it.
- **A runtime `CancelToken` (step-cancellation A2) and the motoko abort orchestrator (Lane B).** Both stay in `m-agent-step-cancellation.md`. They depend on this doc.
- **serve-api per-request contexts** (client disconnect cancels an AI call). It falls out for free once `GoCtx` is set per request; follow-up.
- **The Go codegen `AIHandler`.** It is a separate interface (Conflict Surface).
- **Changing the Ollama knobs.** Their semantics and defaults stay as they are.
- **Raw-keyboard input.** Separate, and by design not in core.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| The grace window changes terminal/worker shutdown timing | Med | Armed only with an AI call in flight (NC3). Terminals are restored before the wait. A second signal is immediate (NC4). The existing lifecycle tests are the gate. |
| A provider ignores ctx mid-body (e.g. a buffered `io.ReadAll`) | Med | `NewRequestWithContext` aborts the body read on cancel in `net/http`; AC1/AC2 prove it per provider. |
| A user sets a tight `--ai-timeout` and truncates reasoning | Low | Opt-in. The trace says `source=flag`. Docs state the M-AI-DECIDE 30 s evidence. |
| Interface churn collides with in-flight AI work (e.g. `m-ai-cross-provider-per-call-routing`, which explicitly leaves `AIHandler` signatures alone) | Low | Phase 1 is mechanical and lands first, on its own. Rebase early. |
| WASM: the abandoned JS promise keeps fetching | Low | The bound is the browser's. An optional `AbortSignal` is deferred. |

## Timeline

**Days 1–2:** Phases 1–2. **Day 3:** Phase 3. **Day 4:** tests, example, docs, changelog. **Total: ~4 days** (the step-cancellation doc's Lane A estimate was ~1 day for the SIGINT case alone; this one covers all three issues).

---

## Relationship to existing docs

- **`planned/v0_29_0/m-agent-step-cancellation.md` (#231).** Its **Lane A** (A1: SIGINT cancels the in-flight AI context) is **generalised and owned here**. This doc is now the home of the core plumbing: interface, interrupt, deadline, vocabulary. Its **Lane B** (the motoko abort extension) and the optional **A2** `CancelToken` stay there and **depend on this doc**. Its Status/Dependencies lines were updated to say so. Correction for Lane B: the abort watcher cannot be AILANG code running beside a blocked `step` (V14), so it is either host-side signal translation or A2 (see §Who cancels).
- **`planned/ailang-core-triage/ai-cancellable-provider-context.md`.** This doc answers the triage note's four decisions: 1 → D1 (clean break), 2 → D2 plus the deadline surface (A1 + A3 first, A2 deferred), 3 → D4 (`Cancelled` yes; `IncompleteStream` deferred to S2), 4 → the WASM `ctx.Done()` select with no panic. The note recommended revising the step-cancellation doc instead of writing a new milestone. A separate doc is used because the scope now includes #1259's deadline surface, which that doc never covered, and because its remaining Lane B has a different owner and lane (extension).
- **`planned/ailang-core-triage/ai-deadline-and-named-test-roundtrip.md` (#1259).** Its Defect 1 is fixed here. Defect 2 (named-test round-trip) is already fixed on dev, per #1259's retitle comment.
- **`implemented/v0_54_0/m-terminal-ui-native-input.md`, `implemented/v0_54_4/m-background-worker-lifecycle.md`.** Both exclude AI-call cancellation. This doc reuses their signal subscription and keeps their restore-first and no-AILANG-on-signal-goroutine rules.
- **`implemented/v0_32_0/m-recorded-stream-api.md`.** It deferred the cancellable context (":790") and filed #578. D6 closes the wall-clock residual without changing its contract.
- **`implemented/v0_40_1/m-ai-decide-system-one.md`.** Key Fact 15 and Future Work (":417") filed the `ctx.AI.Timeout` request answered by the Deadline surface section.

## References

- [Design Axioms](/docs/references/axioms)
- Issues: #1259, #578 (and #546), #231
- `docs/docs/guides/debugging.md` "Ollama Streaming Timeouts": the watchdog semantics being generalised

---

**Document created**: 2026-10-26
**Last updated**: 2026-10-26
