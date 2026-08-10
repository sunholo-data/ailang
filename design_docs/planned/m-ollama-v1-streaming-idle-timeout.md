# M-OLLAMA-V1-STREAMING-IDLE-TIMEOUT: Stream the ollama /v1 tool-calling path with a read-level idle timeout

**Tracking**: ailang#618 (this) · ailang#619 (the publisher-validity fallout)
**Status**: REVIVED 2026-08-07 (was archived 2026-07-29 as charter-unreferenced — see Field Evidence below; the diagnosis was never wrong, it just wasn't on the charter) · REVISED 2026-08-10 after quorum round 2: both objections measured, not argued — the publisher claim is now V18, and the streaming feasibility premise is CONFIRMED on the wire (V21/V22), retiring the M3 capture task
**Target**: v0.34.0
**Priority**: **P0** (raised from P1 — measured 43-day production impact: 80 motoko runs lost, ~74.6 GPU-hours burned, and ACCELERATING)
**Estimated**: 1–2 days (down from 2–3: the repo already ships a tested OpenAI-compatible SSE client with per-index tool-call reassembly — `internal/ai/openai/streamstep.go` — so the streaming/reassembly work this doc originally budgeted does not exist; see Verification Log V8–V11)
**Dependencies**: None. Complements (does not block) motoko_agent#65 (raise total timeout) and P2 context-compression.

---

## Field Evidence (2026-08-07) — why this was revived

Measured by scanning all **2058** motoko session JSONLs in `mk-ast/.motoko/logfile` plus the live
ollama access log (`/private/tmp/ollama-serve-launchd.log` — NOT `~/.ollama/logs/server.log`,
which froze at the 2026-08-03 server restart).

| Metric | Value |
|---|---|
| Sessions carrying `context deadline exceeded` | **182 / 2058 (8.8%)** |
| — died outright (no `run_summary`) | **80** |
| — finished anyway after retrying (silently inflated latency/cost) | **102** |
| Total retry events | **895** |
| Lower-bound GPU time burned | **~74.6 h** (895 × 300s; excludes the 8–22s reload tax per retry and the final in-flight request killed at the wall clock) |
| First affected session → today | **2026-06-26 → 2026-08-07 = 43 days continuous** |

**It is accelerating, and not because of a model change** (`qwen3.6:35b-a3b-mxfp8` in every
affected session): retries/day **16.7 (Jun) → 13.5 (Jul) → 51.4 (Aug)**; retries per affected
session **2.7 → 4.6 → 7.5**. The step counter increments per failure and **step 0 itself times
out**, so "the context grew too long" is REFUTED — the cap is simply below the median turn.

**Why 300s is below a turn:** passing motoko-local runs measure **7–62 output tok/s** (slower at
long context), with whole sessions running **20–37 min**. At ~15 tok/s the 300s budget buys only
~4,500 tokens — right at the median thinking-mode turn, so easy benchmarks squeak under and
long-reasoning ones lose nearly every turn. Evidence it is real generation and not a hang:
durations spread 48s → 4m59 rather than always pinning the cap. On 2026-08-07 alone, **47 requests
ended at exactly `took=4m59.97x`**.

**Blast radius beyond lost runs:** the 30 resulting `api_error` rows all carried
`validity.valid=false, reason="harness_error"`, but `cmd/ailang/eval_publish.go` never reads
`validity` (V18) — so motoko-local's published v0.33.0 **frontier score was exactly 3/22 = 13.6%** where
**17 of those 22 were harness timeouts**. True figure ≈ 60% (n=5). That publisher defect is a
distinct bug tracked as W8 in
[m-eval-validity-discipline](m-eval-validity-discipline.md) / ailang#619; the corrupted rows were deleted
2026-08-07.

**Stopgap in place (REMOVE when this lands):** `AILANG_OLLAMA_HTTP_TIMEOUT_SEC=1800` set via
`launchctl setenv` and pinned in `tools/launchd/dev.ailang.os-rotation-filler.plist` +
`dev.ailang.nightly-eval.plist` (V19). This is precisely the band-aid the Problem Statement below warns
about — it enlarges the guess and over-tolerates genuine hangs (the cap exists because of a
**1h54m** and a **~7h** rig-wedging hang; see 63fc63e09 / 772704cbb — V20). It buys time, not a fix.

*Provenance note:* every claim in this section about the **codebase** is verified in the log below
(V18–V20). The session-scan metrics themselves (the 2058-JSONL sweep, retry counts, GPU-hour
lower bound) are the 2026-08-07 measurement inherited from ailang#619's own triage — external log
data, attributed to that scan, not re-runnable as a repo command.

## Verification Log (all commands re-run at HEAD aeab32c70, 2026-08-10)

One row per codebase claim in this doc. Negative/empty results carry a known-positive control
run in the same call. Glob-shaped flag values are quoted (`--include='*.go'`) — this rig runs
zsh, where an unquoted glob aborts the command before it executes. One exception to
"re-runnable at HEAD": V21 is the controller's 2026-08-10 rig probe against the live ollama
endpoint (GPU lock since released), whose durable artifact is the committed fixture — V22
re-derives everything derivable from that artifact locally.

| # | Claim | Command | Observed |
|---|---|---|---|
| V1 | `internal/ai/ollama/` is unchanged since the doc's authoring SHA, so code claims have not rotted (line numbers below are corrected against HEAD) | `git diff --name-only 79caa15b3e1d..origin/dev -- internal/ai/ollama` | Empty. Control (same call): `git diff --name-only 79caa15b3e1d..origin/dev \| wc -l` → **125** files changed repo-wide, so the instrument sees positives. |
| V2 | File sizes | `wc -l internal/ai/ollama/step.go internal/ai/openai/streamstep.go` | `447` / `344` |
| V3 | The `/v1` path knows it is non-streaming: "the /v1 path does not [stream], so we MUST cap it ourselves" | `grep -n "MUST" internal/ai/ollama/step.go` | `:62` (unrelated), `:291` (the claim) |
| V4 | Total-timeout default is 300s; env override exists, `"0"` disables | `grep -n "defaultOllamaV1TimeoutSec\|AILANG_OLLAMA_HTTP_TIMEOUT_SEC" internal/ai/ollama/step.go` | const `= 300` at `:24`; env read at `:30`; "0 disables the cap" comment at `:23` |
| V5 | The `/v1` client is built with `http.Client{Timeout: ollamaV1Timeout()}` and calls the non-streaming `Step` | `sed -n '288,315p' internal/ai/ollama/step.go` | `openai.NewClient("ollama", …WithHTTPClient(&http.Client{Timeout: ollamaV1Timeout()}))` at `:293-296`; `resp, err := v1.Step(ctx, &r2)` at `:309` |
| V6 | The native `/api/chat` path's cross-chunk tool-call comment (the reuse target the OLD architecture named) | `grep -n "Tool calls arrive" internal/ai/ollama/step.go` | `:411` |
| V7 | The "don't swallow a cut stream as EOF" guard is on the **native** path only (after the `/v1` early return at `:309-313`) | `sed -n '420,428p' internal/ai/ollama/step.go` | comment + `ctx.Err()` surfacing at `:423-428` |
| V8 | The repo already has an OpenAI-compatible SSE streaming client satisfying `ai.StreamingProvider` | `grep -n "func (c \*Client) StreamStep\|func ParseChatStepSSEStream\|bufio.NewScanner" internal/ai/openai/streamstep.go` + `grep -n "StreamingProvider" internal/ai/provider.go` | `StreamStep` at `streamstep.go:42` (sets `stream:true` + `stream_options.include_usage:true` at `:74-75`); `ParseChatStepSSEStream(body io.Reader, …)` at `:217`, scanner at `:218-219`; interface at `provider.go:444-445`; compile-time assert at `streamstep.go:21` |
| V9 | Tool-call deltas ARE accumulated per index into `Response.ToolCalls` (fragment loop) and this is tested | `grep -n "TestStreamStep_ParsesToolCallFragments" internal/ai/openai/streamstep_test.go` | `:109` (doc: "fragmented tool_calls"), test func at `:112`; accumulation loop at `streamstep.go:286-301`, assembly at `:315-341` |
| V10 | **`onChunk` NEVER fires for tool-call deltas** — a callback-driven idle timer starves on tool-call-only turns | `grep -n 'onChunk(' internal/ai/openai/streamstep.go` | Exactly 4 sites: `:255` (`StreamUsage`, final chunk only), `:269` (`StreamContentDelta`), `:281`/`:284` (`StreamThinkingDelta`). The 4 hits are the positive control for this negative claim; corroborated by the doc-comment at `:35-38`: "Tool deltas are NOT streamed individually in Phase 1". |
| V11 | `StreamStep` has exactly ONE non-test caller today — wiring ollama makes a second consumer | `grep -rn "StreamStep(" --include='*.go' internal/ cmd/ \| grep -v "_test.go"` | One call site: `internal/ai/handler.go:378`. (Other hits are the interface decl and the gemini/anthropic/openrouter *definitions*, not callers.) Control (same call): `grep -rn "\.Step(" --include='*.go' internal/ cmd/ \| grep -v "_test.go"` → **6** non-test call sites. |
| V12 | `http.Client.Timeout` covers the entire body read, so it cannot coexist with a long idle-governed stream | `go doc net/http.Client.Timeout` | "The timeout includes connection time, any redirects, **and reading the response body**. The timer remains running after Get, Head, Post, or Do return and will interrupt reading of the Response.Body. A Timeout of zero means no timeout." |
| V13 | A max-token value is applied to the `/v1` request (floor = 16384). **Re-scoped 2026-08-10:** an earlier revision cited this row as proof that a slow-drip-forever stream is bounded — it is not; max-tokens bounds what the *model* generates, not transport duration or keep-alive traffic (see V15/V16). That claim is deleted and replaced by the mandatory hard deadline (Design Freeze #3). | `grep -n "func resolveOllamaMaxTokens\|defaultOllamaMaxTokens = " internal/ai/ollama/step.go` | func at `:98`, `const defaultOllamaMaxTokens = 16384` at `:89`; applied to the `/v1` request at `:306` |
| V14 | `onChunk` is nil-safe, so ollama can pass `nil` (no per-chunk consumer) | `grep -n "onChunk != nil" internal/ai/openai/streamstep.go` | `:254`, `:268`, `:280`, `:283` — every fire site is guarded |
| V15 | `ParseChatStepSSEStream` contains **no bound of any kind** — not duration, not chunk count, not byte count. An endpoint emitting keep-alive bytes forever runs the loop forever absent an external bound. | `grep -nE "time\.\|Deadline\|deadline\|maxChunks\|limit\|budget" internal/ai/openai/streamstep.go` | **Zero hits** (exit 1). Known-positive control in the same call: `grep -c "scanner"` → **5**, so the instrument fires and the zero is a measurement. |
| V16 | The parser loop's ONLY exit is the scanner ending: non-`data:` lines (SSE `: keep-alive` comments) are `continue`d, and `[DONE]` is **`continue`d, not `break`ed** — a server that holds the connection open after `[DONE]` keeps the loop alive too | `sed -n '226,234p' internal/ai/openai/streamstep.go` | `for scanner.Scan()` at `:226`; non-`data:` `continue` at `:228-230`; empty/`[DONE]` `continue` at `:232-234` |
| V17 | The existing `StreamStep` consumer runs UNBOUNDED today: `handler.go` calls it with `context.Background()` (no deadline) and V15 shows the parser adds none | `sed -n '378p' internal/ai/handler.go` | `resp, err := streamingProvider.StreamStep(context.Background(), &Request{` |
| V18 | The publisher never reads `validity` under any casing — the Field Evidence blast-radius claim (quorum round-2 objection, gemini-3-1-pro) | `grep -c "validity\|Validity" cmd/ailang/eval_publish.go` | **0**. Controls: same file, `grep -c "PassRate\|pass_rate"` → **6** (the instrument fires on this file); widened once per the empty-search rule, `grep -niE "valid" cmd/ailang/eval_publish.go` → **0**; repo-level `grep -rniE "validity" --include='*.go' cmd/ \| wc -l` → **12** (the pattern CAN match under `cmd/`). The zero is a measurement, not a broken pattern. |
| V19 | The 1800s stopgap is pinned in exactly the two plists this doc names | `grep -l "AILANG_OLLAMA_HTTP_TIMEOUT_SEC" tools/launchd/dev.ailang.os-rotation-filler.plist tools/launchd/dev.ailang.nightly-eval.plist` | Both paths print; the pinned value is `<string>1800</string>` |
| V20 | The rig-wedging hangs that motivated a cap at all are real commits with matching messages | `git log --oneline --no-walk 63fc63e09 772704cbb` | `772704cbb fix(ai/ollama): bound native /api/chat with a context deadline (no more 7h hangs)` · `63fc63e09 fix(ai/ollama): bound /v1 tool-calling with an HTTP timeout (no more infinite hangs)` |
| V21 | **Feasibility premise (quorum round-2 objection, gpt5-6-sol): ollama `/v1` DOES stream tool calls in OpenAI-compatible shape with `stream:true`** — measured end-to-end on the exact target model, no longer deferred to M3 | Controller rig probe, 2026-08-10, under the rig GPU lock (since released — this row is NOT locally re-runnable; its artifact is the committed fixture, which V22 re-derives from): `POST http://127.0.0.1:11434/v1/chat/completions` with `stream:true`, `stream_options.include_usage:true`, `max_tokens:256`, one `get_weather` function tool, model `qwen3.6:35b-a3b-mxfp8` (already resident per `/api/ps` — no cold-load perturbation), ollama server **0.32.1**. Wire bytes committed verbatim: `internal/ai/ollama/testdata/ollama_v1_stream_toolcall.sse` | `curl` rc=0, **13,077 bytes**, **52 `data:` events** for one short tool-calling turn. Tool-call delta arrives as `choices[0].delta.tool_calls[0]` with `id`, `"index":0`, `"type":"function"`, `function.name`, and complete `function.arguments` (`{"city":"Paris"}`) — the whole call, arguments included, in **ONE** chunk. A following chunk carries `finish_reason":"tool_calls"`; a usage-only chunk carries `prompt_tokens:294, completion_tokens:80, total_tokens:374`; exactly **1** `[DONE]`. Endpoint family note: `[::1]` did not answer, only `127.0.0.1` — the historical split-brain was not present on probe day. |
| V22 | Chunk-type census of the committed fixture — re-derived locally from the repo file at HEAD, independent of the probe | `grep -c` per pattern on `internal/ai/ollama/testdata/ollama_v1_stream_toolcall.sse`: `'"reasoning"'`, `'tool_calls'`, `'"finish_reason":"tool_calls"'`, `'prompt_tokens'`, `'\[DONE\]'`, `'^:'`, `'^$'`; `wc -l` | **48** `delta.reasoning` chunks (fire `onChunk` at `streamstep.go:284`) · **1** `delta.tool_calls` (fires NOTHING — V10) · **1** empty-delta `finish_reason:"tool_calls"` (fires nothing) · **1** usage-only (fires at `:255`) · **1** `[DONE]`. **49 of 51 parsed chunks would have fired `onChunk`** on this turn — the V-C3 narrowing in the Design Constraint. File shape: **104 lines = 52 `data:` + 52 empty SSE event separators**; `grep -c '^:'` → **0** (control: `grep -c '^$'` → 52, same call) — ollama emitted **no** `:`-comment keep-alives on this probe. |
| V23 | The empty-content guard is why a `content:""` chunk fires nothing — both the tool-call chunk and the finish chunk in the fixture carry `content:""` and skip the fire site | `grep -n 'if choice.Delta.Content != ""' internal/ai/openai/streamstep.go` | Guard at `:266`; the `StreamContentDelta` fire site it protects is `:269`. (Corrects the controller directive, which placed the guard itself at `:269`.) |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Rationale |
|---|---|---|
| A1: Determinism | 0 | Timeouts are inherently wall-clock; result determinism unchanged. Fewer *spurious* failures on big prompts. |
| A2: Replayability | +1 | Fewer false timeout aborts → fewer truncated/aborted traces to replay. |
| A3: Effect Legibility | 0 | No effect-signature changes (Net effect still {Net}). |
| A4: Explicit Authority | 0 | No new ambient authority. |
| A5: Bounded Verification | 0 | No type-system impact. |
| A6: Safe Concurrency | 0 | Single SSE connection read; the idle watchdog is one timer goroutine per request, cancelled on completion. |
| A7: Machines First | +1 | Long-running local-LLM calls stop failing mid-task — directly serves AI-agent reliability. |
| A8: Minimal Syntax | 0 | No syntax. |
| A9: Cost Visibility | +1 | Idle-timeout + time-to-first-token are explicit, tunable env knobs; a genuine hang still surfaces fast — 2.5× faster than today (120s idle vs the 300s total cap). |
| A10: Composability | +1 | Raised from 0: the design reuses the existing, tested `openai.StreamStep`/`ParseChatStepSSEStream` (V8–V9) instead of building parallel reassembly in the ollama package — one SSE parser serves both consumers. |
| A11: Structured Failure | +1 | Distinguishes "idle/hung" from "long but progressing"; typed `idle-timeout` vs `ttft-timeout` errors instead of a swallowed cut stream. |
| A12: System Boundary | 0 | HTTP/ollama boundary unchanged. |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check
- [x] A1 (Determinism): no implicit nondeterminism introduced (timeout semantics only).
- [x] A3 (Effects): no hidden side effects.
- [x] A4 (Authority): no ambient access granted.
- [x] A7 (Machines First): optimizes machine reliability, not human convenience.

## Problem Statement

The ollama `/v1` tool-calling path in `internal/ai/ollama/step.go` is **non-streaming**: it delegates
to the non-streaming `openai.Client.Step` (V5), so it can only be bounded by a **TOTAL**
`http.Client.Timeout` (default 300s via `AILANG_OLLAMA_HTTP_TIMEOUT_SEC`, V4). The whole exchange —
connection + prefill (processing the entire prompt) + generating the *complete* response — must finish
within that single budget.

**Current State:**
- On large-context agentic tasks, the accumulated prompt makes a single qwen3.6 request exceed the total
  budget, and the AILANG runtime aborts the **entire run** with
  `Post "http://localhost:11434/v1/chat/completions": context deadline exceeded`.
- Observed concretely on the `docx_reimplement` P0 large-context instrument: motoko died at step 14 (300s)
  and again at step 27 after writing a full 526-line reimplementation — the run was killed mid-task, not
  because the model failed but because one request outran the total clock.
- The code already knows this is a limitation — `step.go:291` (V3): *"the /v1 path does not [stream], so
  we MUST cap it ourselves."* The native `/api/chat` path **does** stream chunk-by-chunk and "surfaces a
  dropped stream quickly," but the tool-calling path doesn't get that benefit.
- Band-aid in flight: motoko_agent#65 raises the total to 1800s. That just enlarges the guess; it doesn't
  remove the fundamental "must size a total budget" problem and over-tolerates genuine hangs.

**Impact:**
- Any harness (motoko, pi, the eval harness) running multi-step agentic AILANG synthesis on local ollama
  hits this on long/large-context tasks — the regime where AILANG-native harnesses are supposed to win.
- It silently caps the achievable difficulty of the eval suite (large-context tasks can't complete), so it
  blocks the P0 differentiator measurement entirely.

## Goals

**Primary Goal:** Make long-running local-LLM tool-calling calls robust to legitimately-long generation
without having to guess a total time budget.

**Success Metrics:**
- A `/v1` request whose *total* time exceeds 300s but which is *continuously producing bytes* completes
  successfully (no spurious abort) — **including turns that emit ONLY tool-call deltas**. That shape is
  a supported configuration this design must survive (non-thinking models, thinking mode disabled), not
  the observed qwen3.6 default — see the V-C3 narrowing in the Design Constraint below.
- A genuinely hung/stalled stream (no bytes for the idle window) still fails fast (≤ idle window + slack) —
  faster than today's 300s.
- `docx_reimplement` completes a full motoko run end-to-end (produces an X/17 grade) rather than dying on a
  timeout mid-task.
- No regression on normal small calls (latency + correctness unchanged), and the flag-off path is
  code-identical to today.

## Design Constraint (found in verification — this reshaped the architecture)

An earlier draft of this design armed a TTFT timer and then **reset the idle deadline from the
per-chunk callback** ("reset on each subsequent chunk"). That design is wrong for exactly the
traffic it must protect:

- `onChunk` fires at only four sites (V10): usage (final chunk only), content deltas, and thinking
  deltas. **There is no `onChunk` fire for tool-call deltas** — they accumulate silently into
  `Response.ToolCalls` (V9). A `content:""` chunk also fires nothing, because of the guard at
  `streamstep.go:266` (V23) — and the fixture's tool-call and finish chunks both carry `content:""`.
- A turn that emits only tool calls and no content/reasoning text therefore **starves a
  callback-driven idle timer**, which false-trips on a stream that is progressing perfectly.

**V-C3 narrowing (2026-08-10, from the rig probe — V21/V22).** An earlier revision of this doc
claimed tool-call-only turns are "motoko's dominant turn shape". **The wire refutes the strong
form of that claim.** On the probed qwen3.6 thinking-mode turn, 48 of 51 parsed chunks were
`delta.reasoning` and **49 of 51 would have fired `onChunk`** — qwen3.6 in thinking mode emits a
long reasoning stream *before* the tool call, so on that turn a callback-driven idle timer would
in fact have been fed continuously. The accurate position, on which the design now rests:

- **(a) The mechanism defect is real and unchanged:** `onChunk` has no tool-call fire site (V10),
  so callback-driven progress is not a sound signal *by construction* — regardless of how often
  the starvation shape occurs in practice.
- **(b) What the probe changes is the frequency argument:** for qwen3.6 thinking-mode turns the
  callback happens to be fed by `reasoning` deltas. The starvation case is a turn emitting tool
  calls with **no content and no reasoning** — a non-thinking model, or thinking mode disabled —
  a configuration this repo supports but which the probe did NOT exercise.

**Therefore the idle deadline lives at the READ level, not the callback level — as a robustness
argument, not a fix for an observed outage:** wrap the response body in an idle-timeout
`io.ReadCloser` whose deadline resets on every `Read` returning n>0, and hand that reader to the
SSE scanner. Read-level placement is correct **independent of model chattiness**: it measures
bytes on the wire, so it cannot be voided by a model, a mode, or a Phase-2 change to which deltas
get callbacks. Bytes-on-the-wire progress is agnostic to chunk type — tool-call-only turns,
content turns, reasoning turns, and SSE keep-alive comments alike. That last case cuts both ways:
keep-alive bytes resetting the idle timer is exactly why the idle timer alone is not a bound — a
server emitting `: keep-alive` comments forever feeds the reader bytes that the parser
`continue`s without ever producing a chunk (V15/V16). Note this keep-alive traffic is
**hypothetical for ollama** — the probe observed zero `:`-comment lines (V22); the mandatory hard
deadline (Design Freeze #3) that bounds it is defensive, kept for malformed/hostile servers, not
a response to observed ollama behaviour. The idle timer is only the fast hang detector.
`ParseChatStepSSEStream` already takes an `io.Reader` (V8), and the ollama package controls the
`http.Client` it hands to the openai client (V5), so the wrapper injects via a custom
`http.RoundTripper` **without modifying `internal/ai/openai` at all**.

## High-Impact Decisions

| Decision | Resolution | Why | Change cost |
|---|---|---|---|
| Idle deadline placement: read-level vs callback-level | **Read-level (RESOLVED — forced by V10)** | Callback-driven is unsound by construction (no tool-call fire site); read-level is correct independent of model chattiness. NOT because starvation is the observed default — the probed thinking-mode turn would have fed the callback (V-C3 narrowing, V22) | high if wrong |
| Stream the `/v1` path | **Switch `v1.Step` → `v1.StreamStep` with `onChunk: nil` (RESOLVED — enabled by V8/V9/V14; feasibility MEASURED on the wire, V21)** | Reassembly already exists and is tested; ollama 0.32.1 streams OpenAI-shaped tool calls with `stream:true` (V21); the ollama-side "reuse the native `/api/chat` reassembly" plan from the earlier draft is unnecessary | med |
| TTFT allowance separate from inter-chunk idle | **Yes — first-Read deadline = TTFT window, subsequent = idle window (RESOLVED)** | Prefill on a huge prompt is byte-less; a single idle window would false-trip during prefill | med |
| Hard ceiling as backstop | **MANDATORY finite hard deadline, default 3600s, `0` rejected (RESOLVED — reviewer-required, see Design Freeze #3)** | V15/V16: the SSE parser has no bound of any kind and keep-alive bytes reset a read-level idle timer forever; max-tokens bounds generation, NOT transport duration (V13 re-scope). `http.Client.Timeout` still can't be the mechanism (V12) — the deadline is applied via `context.WithTimeout` | low |

### Design Freeze (RESOLVED 2026-08-10 — grounded in Field Evidence; a human ruling is only needed if a reviewer rejects the grounding)

- [x] **Idle-window default: `AILANG_OLLAMA_IDLE_TIMEOUT_SEC=120`.**
  Grounding: measured output rates are 7–62 tok/s (Field Evidence), so the expected inter-chunk gap
  even at the slowest measured rate is well under 1s; the largest *legitimate* mid-stream stall we
  have measured is the 8–22s runner-reload tax. 120s is >5× that worst measured stall, yet detects
  the real hang class (the 1h54m and ~7h hangs behind 63fc63e09/772704cbb) 2.5× faster than
  today's 300s total cap. **Would change if:** a progressing run is ever observed with a
  mid-stream byte gap >120s (the streamed path makes inter-chunk gaps measurable for the first
  time — log the max observed gap per request at debug level so this is checkable from the field).
- [x] **TTFT-window default: `AILANG_OLLAMA_TTFT_TIMEOUT_SEC=600`.**
  Grounding: step-0 requests outran the entire 300s total budget (Field Evidence: "step 0 itself
  times out"), and that budget covered prefill + full generation; 600s gives prefill alone 2× the
  old *total* budget, on top of the 8–22s reload tax. **Would change if:** the TTFT distribution —
  observable for the first time once streaming lands; log it per request — shows p99 approaching
  600s on passing runs.
- [x] **Mandatory finite hard deadline (REPLACED 2026-08-10 — quorum objection, gpt5-6-sol):
  "Streaming mode always applies a configurable hard request deadline, with a conservative finite
  default; setting it to 0 is rejected rather than disabling the bound. The 120s byte-idle timer
  remains the fast hang detector, while the hard deadline bounds indefinitely progressing
  keep-alive or malformed streams."** The knob stays `AILANG_OLLAMA_HTTP_TIMEOUT_SEC`; in the
  streaming branch it is applied via `context.WithTimeout` (never `Client.Timeout` — V12), with
  **default 3600s**. On expiry the surfaced error is a typed `deadline-exceeded`, distinct from
  `idle-timeout`/`ttft-timeout`. **Framing (2026-08-10):** this backstop is **defensive**, not a
  response to observed ollama behaviour — the probe saw zero keep-alive comment lines from ollama
  (V22). It stays mandatory because the parser itself has no bound of any kind (V15/V16), which is
  the correct posture against malformed/hostile servers — and because the same parser's other
  consumer (`handler.go:378`, V17) talks to real OpenAI/OpenRouter endpoints, where this doc
  cannot vouch for server behaviour.
  Grounding for 3600s, from this doc's own Field Evidence: the worst *legitimate* single request
  is bounded by TTFT window (600s) + generation of the max-tokens floor (16384 tokens, V13) at
  the slowest measured rate (7 tok/s) = 600 + 2341 ≈ **2941s (~49 min)**; 3600s gives ~22%
  headroom over that computed worst case and sits above the longest observed whole *session*
  (37 min), while still catching both rig-wedging hangs that motivated a cap at all (1h54m and
  ~7h — 63fc63e09/772704cbb) faster than the current 1800s-stopgap-plus-retries reality.
  **`0` (or negative) is REJECTED, not "disabled":** the streaming branch returns a loud typed
  configuration error at client construction (Critical Principle 2 — no silent fallbacks where
  the value affects behaviour). This deliberately DIVERGES from today's `ollamaV1Timeout()`
  (`step.go:29-39`), where `<= 0` means "no timeout"; the flag-off path keeps that legacy
  semantics untouched, and the divergence is a documented behaviour change of the opt-in branch,
  not an accident. An explicit value ≥ 1 is honored as given.
  **Would change if:** the per-request duration logs (M2/S7) show a legitimate completing request
  exceeding 80% of the deadline, or `AILANG_OLLAMA_MAX_TOKENS` is raised above the 16384 floor —
  the generation term scales linearly with the floor, so the default must be re-derived from the
  same formula.
  *Superseded reasoning, kept for the record:* the previous freeze defaulted the ceiling OFF,
  arguing the max-tokens floor bounds slow-drip. The quorum objection is measurably right and
  worse than stated: max-tokens bounds what the model generates, but V15/V16 show the parser
  itself has no bound of any kind, `continue`s keep-alive comment lines and `[DONE]` alike, and
  only exits when the scanner ends — so keep-alive bytes reset the idle timer forever and
  max-tokens never enters the picture. The safeguard cannot be deferred until such a hang is
  observed; it is the bound.

## Solution Design

**Overview:** Behind an opt-in flag (`AILANG_OLLAMA_V1_STREAM=1`), switch the `/v1` tool-calling
call from the buffered `v1.Step` under a total deadline to the existing `v1.StreamStep`
(OpenAI-compatible SSE, tool-call reassembly already tested — V8/V9), governed by a **read-level**
idle deadline that resets on every received byte, a separate, more generous TTFT window to
absorb prefill silence, and a **mandatory finite hard deadline** that bounds streams which make
byte-progress forever without ever completing (Design Freeze #3).

**Architecture:**
1. **Idle-timeout reader** (new, `internal/ai/ollama`): an **`io.ReadCloser`** wrapper + watchdog
   timer. A blocked `Read` cannot interrupt itself, so the watchdog cancels the request's context
   on expiry — the transport then fails the pending `Read` (V12 semantics, but driven by our
   timer, not `Client.Timeout`). First-`Read` deadline = TTFT window; every `Read` returning n>0
   resets the deadline to the idle window. On expiry the wrapper records *which* window fired so
   the surfaced error is a typed `ttft-timeout` or `idle-timeout`, not a generic
   `context canceled`. **`Close()` stops the watchdog timer/goroutine and then closes the
   underlying body** — a fully-consumed happy-path body must not leak a timer or goroutine
   (gemini-3-1-pro's catch; the leak test is S9). It must be a ReadCloser, not a bare Reader,
   because the `http.Client` machinery and any downstream `body.Close()` call the wrapper's
   `Close`, and a bare Reader would either lose the underlying Close or leave the watchdog armed.
2. **Injection via `http.RoundTripper`**: the ollama `/v1` branch already constructs its own
   `http.Client` (V5). In streaming mode it supplies `Timeout: 0` (V12), a custom transport whose
   `RoundTrip` wraps `resp.Body` in the idle reader, and `Transport.ResponseHeaderTimeout` = TTFT
   window (defense for a server that stalls before even sending headers). The **mandatory hard
   deadline** (Design Freeze #3) is applied as `context.WithTimeout` around the whole request in
   the same branch — it is not optional and not skippable when the env knob is unset. On expiry
   the error maps to the typed `deadline-exceeded`. `internal/ai/openai` is not modified.
   **Timeout composition, stated exactly:** `ResponseHeaderTimeout` (600s) and the first-`Read`
   TTFT deadline (600s) compose **additively** — headers can arrive at 599s and the body then
   stay silent another 600s — so the true worst-case pre-first-byte wait is **~1200s (20 min)**,
   itself bounded by the 3600s hard deadline. Accepted: both phases are byte-less prefill-shaped
   waits, and collapsing them into one shared clock would complicate the wrapper for a bound the
   hard deadline already enforces.
3. **Switch the call**: `v1.Step(ctx, &r2)` (`step.go:309`) becomes `v1.StreamStep(ctx, &r2, nil)`
   in the flag-on branch — `onChunk` is nil-safe (V14) and we have no per-chunk consumer.
   `StreamStep` returns the same `*ai.Response` shape as `Step`, with tool calls assembled
   (V8/V9), so `logOllamaResponse` and everything downstream is unchanged.
4. **Cut-stream error surfacing**: the native path needs an explicit guard because the ollama
   client library can swallow a cancelled stream as EOF (`step.go:423-428`, V7).
   `ParseChatStepSSEStream` does not have that defect — a mid-body cancellation surfaces through
   `scanner.Err()` → `ai.ClassifyError` (`streamstep.go:308-310`). The ollama branch maps that
   error, when our watchdog caused it, to the typed idle/TTFT timeout error.
5. **Flag-off path**: the existing code at `step.go:293-313` is kept verbatim inside the
   `else` branch — the default path's diff is indentation-only, provable by inspection of the PR
   hunk, and Success Criterion S5 asserts its behavior (no `"stream":true` on the wire).

**Blast radius (must be watched, not feared):** wiring ollama to `StreamStep` makes it the
**second** non-test consumer — today `internal/ai/handler.go:378` is the only one (V11, control:
6 non-test `.Step(` sites). Any future behavior change inside `StreamStep`/`ParseChatStepSSEStream`
now affects both the OpenAI streaming path and the ollama tool-calling path; conversely, this
design deliberately adds **no** behavior change there (fixture tests may be added to
`streamstep_test.go` for ollama-shaped streams, which only widen coverage).

**Files to modify (re-derived under V8/V9 — the earlier draft budgeted 80–150 LOC to build
reassembly that already exists):**
- `internal/ai/ollama/idlereader.go` (**new**, ~100 LOC): idle/TTFT reader + watchdog + the
  wrapping `RoundTripper` + typed timeout errors.
- `internal/ai/ollama/idlereader_test.go` (**new**): pure-`io.Pipe` tests, no HTTP (M1 below).
- `internal/ai/ollama/step.go` (~40–60 LOC): the flag-gated streaming branch (client construction
  with `Timeout: 0` + transport, `StreamStep` call, error mapping) + two env knobs alongside
  `ollamaV1Timeout()`.
- `internal/ai/ollama/step_test.go`: fake-SSE-server tests (M2 below), including the
  tool-call-only starvation gate.
- `internal/ai/openai/streamstep_test.go` (M3): replay the **already-committed** ollama `/v1`
  fixture (`internal/ai/ollama/testdata/ollama_v1_stream_toolcall.sse` — real wire bytes from the
  2026-08-10 rig probe, V21, not a hand-written mock) through `ParseChatStepSSEStream` — coverage
  widening only, no source change. The capture itself is DONE; only the replay test remains.

**Out of scope / non-goals:** changing the native `/api/chat` path (already streams); any change to
`internal/ai/openai/streamstep.go` source; per-chunk consumer plumbing (Phase 2 `ToolCallDelta` per
the streamstep doc-comment); context compression (P2 — this fix handles "long generation", P2
handles "huge prefill"); any motoko-side change beyond already-merged env forwarding
(motoko_agent#65).

## Risks / Conflict Surface

This change is confined to `internal/ai/ollama` (NOT parser/typechecker/codegen), so there is no
grammar/type conflict surface. Runtime risks:

- **Tool-call delta reassembly correctness — DOWNGRADED AGAIN (2026-08-10): the shape is now
  MEASURED, not hypothesized.** The reassembly is existing, tested code
  (`TestStreamStep_ParsesToolCallFragments`, "tool_calls fragmented across 3 fragments" —
  `streamstep_test.go:109`, V9), and the probe shows ollama 0.32.1 emits the whole call —
  complete JSON `arguments` included — in **ONE** chunk with `id`/`index`/`type`/`name` in
  OpenAI-compatible positions (V21). The per-index accumulator handles that as the degenerate
  1-fragment case; the harder 3-fragment form is what the existing test already covers. The
  earlier draft's feared shapes (arguments split oddly, missing usage block) did not appear: the
  `stream_options` usage chunk arrived and `finish_reason:"tool_calls"` was set. **Residual
  risk:** a future ollama version or a different model could change the emission shape.
  Mitigation: the committed fixture replayed through `ParseChatStepSSEStream` (M3) pins today's
  shape as a regression test; the opt-in flag holds the old path as default for one release.
- **Second `StreamStep` consumer** (V11): behavior changes in the shared parser now have a
  two-consumer blast radius (OpenAI streaming + ollama tool-calling). Mitigation: this design
  changes no `internal/ai/openai` source; new fixtures only add coverage.
- **Prefill silence false-trip** — handled by the separate TTFT window (600s = 2× the old total
  budget); `Transport.ResponseHeaderTimeout` covers the pre-header phase.
- **Byte-progress without token-progress** (slow-drip/keep-alive runaway): the idle window only
  proves bytes are flowing, and V15/V16 prove the parser itself never terminates on such traffic —
  max-tokens does NOT bound this (V13 re-scope). Bounded solely by the mandatory hard deadline
  (Design Freeze #3); S8 is the test that proves the bound exists. **Defensive:** the probe
  observed no keep-alive comment traffic from ollama (V22) — this bullet guards against
  malformed/hostile servers and the parser's other, real-endpoint consumer, not observed ollama
  behaviour.
- **Server holds the connection open after `[DONE]`**: the parser `continue`s `[DONE]` rather
  than `break`ing (V16), so the loop only exits when the body ends. This design deliberately does
  NOT change that (no `internal/ai/openai` source changes — two-consumer blast radius): a silent
  post-`[DONE]` hold-open is caught by the 120s idle timer, and a keep-alive-emitting one by the
  hard deadline. Explicitly left to the two timers, not fixed in the parser. Also defensive: on
  the probe the stream ended normally after `[DONE]` (curl rc=0, 13,077 bytes — V21).
- **Pre-existing unbounded consumer**: `internal/ai/handler.go:378` already calls `StreamStep`
  today with `context.Background()` and no deadline of any kind (V17) — the hazard this freeze
  bounds for ollama has been live on that path all along. The backstop added here lives in the
  ollama package's transport, so it deliberately does NOT cover `handler.go`; that path's bound
  is out of scope for this doc and should be tracked as its own issue when this lands.
- **ollama SSE compatibility — RESOLVED (2026-08-10), no longer an open risk.** Measured on the
  live endpoint against the exact target model (V21): `stream:true` with tools streams 52 events,
  OpenAI-shaped tool-call delta, correct `finish_reason`, working `include_usage`, exactly one
  `[DONE]`. The wire bytes are committed as the fixture. What remains is version drift (covered
  by the fixture-replay regression test) — not feasibility.

## Rollout (opt-in for one release)

- v0.34.0 ships with `AILANG_OLLAMA_V1_STREAM` **default off**. Off means the `/v1` branch runs
  the existing code verbatim (moved into an `else` — indentation-only diff) with the existing
  300s-default total cap: byte-identical requests on the wire, proven by S5 below and by the
  unchanged existing `step_test.go` suite.
- The rig opts in via the launchd plists that today pin the `AILANG_OLLAMA_HTTP_TIMEOUT_SEC=1800`
  stopgap (`dev.ailang.os-rotation-filler.plist`, `dev.ailang.nightly-eval.plist`) — replace the
  stopgap pin with `AILANG_OLLAMA_V1_STREAM=1` in the same edit, so the band-aid is removed by
  the same change that supersedes it. Removing the pin is **mandatory, not tidy-up**: in
  streaming mode the same env var becomes the hard deadline (freeze #3), and a leftover 1800s
  pin sits BELOW the computed worst-case legitimate request (~2941s) — it would re-create the
  4m59.97 failure at a larger scale. Note the semantics split: flag-off, `0` still means "no
  timeout" (legacy `ollamaV1Timeout()`, `step.go:29-39`); flag-on, `0` is a loud construction
  error.
- Default flips on in v0.35.0 only if the M3 rig validation held for the release cycle (no
  idle/TTFT false-trips on passing runs — checkable from the per-request max-gap/TTFT debug logs
  added in M2).

## Success Criteria (each names the mutation that turns it red)

- [ ] **S1 — Tool-call-only starvation gate (the V10 regression test).** A fake SSE server emits
  ONLY `tool_calls` fragments (no content, no reasoning), one fragment per few seconds, for a
  total duration exceeding the configured idle window; the streamed call must COMPLETE with
  correctly assembled `ToolCalls`. **Rationale, stated honestly (V-C3 narrowing):** this tests a
  hypothetical, not the observed default — the probed qwen3.6 thinking-mode turn fed `onChunk`
  via 48 reasoning deltas and would NOT have starved a callback timer (V22). The starvation shape
  is what a non-thinking model, or thinking mode disabled, emits — configurations this repo
  supports that the probe did not exercise. The criterion is deliberately defensive: it pins the
  property that makes read-level placement correct by construction (V10: no tool-call fire site;
  V23: `content:""` fires nothing), rather than claiming to reproduce production traffic. **Red
  under:** moving the idle deadline to the `onChunk` callback (no callback ever fires → false
  trip), or any future streamstep change that stops tool-call bytes flowing through the wrapped
  reader.
- [ ] **S2 — Hang detection.** Fake server sends 3 content chunks then goes silent forever: the
  call fails with the typed `idle-timeout` error within idle window + slack (test-configured
  windows, sub-second). **Red under:** deleting the watchdog, or setting `Client.Timeout: 0`
  without the idle reader (test would hang to the Go test timeout).
- [ ] **S3 — Prefill absorption.** Fake server delays the first body byte for longer than the idle
  window but within the TTFT window, then streams normally: must COMPLETE. **Red under:** a
  single-window design (idle armed from t=0).
- [ ] **S4 — Long-but-progressing beats the old cap.** With test-scaled windows, a
  continuously-producing stream whose total time exceeds the old-total-cap-equivalent (but stays
  under the test-configured hard deadline) completes. **Red under:** leaving
  `Timeout: ollamaV1Timeout()` on the streaming branch's `http.Client` (the V12 trap — the
  client timer interrupts the body read).
- [ ] **S5 — Flag-off is byte-identical.** With `AILANG_OLLAMA_V1_STREAM` unset, a fake server
  asserts the request body contains no `"stream":true` and the non-streaming path answers; the
  entire pre-existing `step_test.go` suite passes unmodified. **Red under:** flipping the default,
  or any drive-by edit to the non-flag branch.
- [ ] **S6 — Streamed ≡ buffered.** The same response content served once buffered (via `Step`)
  and once as SSE (via the streaming branch) yields identical `Response.Text`,
  `Response.ToolCalls` (order, IDs, assembled arguments), and `FinishReason`. **Red under:**
  fragment-ordering or concatenation regressions in assembly.
- [ ] **S7 — Field validation (M3, on the rig).** With the flag on, `docx_reimplement` produces an
  end-to-end X/17 grade, and the per-request debug logs show max inter-chunk gap, TTFT, and total
  request duration for every request (the falsifiers for Design Freeze #1/#2/#3). **Red under:**
  any false idle/TTFT trip on a progressing run, or a run lost to `context deadline exceeded` at
  a step boundary.
- [ ] **S8 — Keep-alive-forever is bounded (MANDATORY — the quorum-objection test; this row is
  what proves the backstop exists).** A fake SSE server emits `: keep-alive` comment lines
  forever (bytes flowing, never a parseable chunk — the V15/V16 traffic shape), at an emission
  interval well under the test-scaled idle window (so the idle timer is genuinely being reset,
  never firing) and with hard deadline > idle window (so the terminating error provably comes
  from the deadline, not the idle timer). The
  call must TERMINATE with the typed `deadline-exceeded` error within hard deadline + slack.
  This traffic shape is synthetic-by-design: ollama emitted zero comment keep-alives on the probe
  (V22) — the test proves the defensive bound against malformed/hostile servers, not an observed
  ollama behaviour.
  **Red under:** removing the `context.WithTimeout` hard deadline — the keep-alive bytes reset
  the idle timer indefinitely and the test hangs to the Go test timeout. A variant asserting
  that `AILANG_OLLAMA_HTTP_TIMEOUT_SEC=0` fails client construction with the typed configuration
  error (not a silently-unbounded client) goes red under re-introducing "0 disables".
- [ ] **S9 — Close() leaks nothing.** After a fully-consumed happy-path stream, `Close()` on the
  idle reader stops the watchdog: assert no watchdog goroutine/timer survives (via
  `go.uber.org/goleak` — already in `go.sum` at v1.3.0 — or a watchdog done-channel probe), and
  that the underlying body's `Close` was called exactly once. **Red under:** dropping the
  `timer.Stop()`/goroutine-shutdown from `Close()`, or wrapping as a bare `io.Reader` so the
  underlying body's `Close` is lost.

(The earlier draft's "All tests passing" and "Documentation updated" rows are deleted — they pass
identically whether or not the feature works, so they gate nothing.)

## Implementation Plan (milestone-shaped; each independently landable and testable)

**M1 — Idle/TTFT reader (no HTTP, no flag, no behavior change).**
`internal/ai/ollama/idlereader.go` + `idlereader_test.go`: the deadline-resetting
**`io.ReadCloser`**, the watchdog that cancels a supplied `context.CancelFunc`, the typed
`ttft-timeout`/`idle-timeout`/`deadline-exceeded` errors, and the body-wrapping `RoundTripper`.
`Close()` stops the watchdog and closes the underlying body. Tests drive it with `io.Pipe` at
millisecond windows: slow-drip completes; pre-first-byte silence trips TTFT; mid-stream silence
trips idle; which-window-fired is preserved in the error; Close leaks no timer/goroutine (S9).
*Acceptance: S2/S3/S9 mechanics green at the reader level; `make lint` + package tests green;
zero call sites yet, so zero runtime change.*

**M2 — Flag-gated streaming branch.**
`internal/ai/ollama/step.go`: `AILANG_OLLAMA_V1_STREAM=1` branch building the zero-`Timeout`
client + transport, the **mandatory `context.WithTimeout` hard deadline** (freeze #3: default
3600s from `AILANG_OLLAMA_HTTP_TIMEOUT_SEC`; `<= 0` → typed configuration error at
construction), the two window knobs (freeze #1/#2 defaults), the `v1.StreamStep(ctx, &r2, nil)`
call, error mapping, and per-request max-gap/TTFT/total-duration debug logging. Fake-SSE-server
tests in `step_test.go` covering S1–S6 and S8 — S1 (tool-call-only starvation) and S8
(keep-alive-forever bounded) are the non-negotiable gates. *Acceptance: S1–S6 + S8 green;
existing tests untouched and green; flag-off diff is indentation-only.*

**M3 — Fixture replay + rig validation + docs (RE-SCOPED 2026-08-10: the capture task is DONE).**
The original M3 task "capture a real streamed response in a test fixture" was completed **before
the sprint** by the controller's 2026-08-10 rig probe: real wire bytes — not a hand-written mock
— are committed at `internal/ai/ollama/testdata/ollama_v1_stream_toolcall.sse` (13,077 B,
52 events; provenance in V21, contents re-derived locally in V22). What remains in M3: write the
replay test through `ParseChatStepSSEStream` (added to `streamstep_test.go` — coverage only, no
source change); flip the rig plists to `AILANG_OLLAMA_V1_STREAM=1` (removing the 1800s stopgap
pin in the same edit — V19); run `docx_reimplement` end-to-end; record the observed TTFT /
max-gap distributions against the freeze defaults. Add the three env knobs to the debug-flags
table in `.claude/rules/dev-workflow.md` + `docs/docs/guides/debugging.md`. *Acceptance: S7
green; fixture-replay test green; freeze-default falsifier data recorded in the PR description.*

## Timeline (realistic; ~1–2 days total under V8/V9)
- Day 1 AM: M1. Day 1 PM: M2 (the branch is ~40–60 LOC; the tests are the bulk).
- Day 2: M3 (rig time dominates; the docx run is wall-clock-bound — the fixture capture is
  already done, which shortens M3's code-bound portion to the replay test + plist flip).

## Related Documents
- [m-eval-stream-health-retry](v0_24_0/m-eval-stream-health-retry.md) (neural 0.48) — **distinct**: that covers eval-harness-level stream health + retry policy; this is the ollama client's per-call timeout semantics underneath it. Complementary.
- [m-ai-streaming-helper](v0_17_0/m-ai-streaming-helper.md) (neural 0.37) — **distinct**: std/ai user-facing streaming wrapper; this is the internal tool-calling transport. (M-AI-STEP-STREAMING v0.18.7 is what shipped `StreamStep` — the machinery this design now reuses.)
- motoko_agent#65 — the env-forwarding band-aid (raise total timeout) this design supersedes as the robust fix.
- P2 (context_mode / on_tool_handle compression) — attacks prefill *size*; orthogonal and complementary.
