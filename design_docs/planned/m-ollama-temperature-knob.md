# M-OLLAMA-TEMPERATURE-KNOB: Configurable Temperature on the Agentic Ollama Path

**Status**: Planned (code landing; A/B is the GPU follow-up)
**Target**: v0.25.x
**Mission item**: #2 (request-param engagement) — the pi-faithful lever
**Estimated**: ~25 LOC + tests; <0.5 day (no GPU for the code part)

## Problem

Motoko's residual AILANG failures are qwen3.6 **non-deterministically emitting prose / 0
tool calls** instead of writing a solution (proven: WriteFile↔pass 100% correlation;
2026-06-17 analysis-log entries). pi runs the *same* model and engages reliably (88%) with
a **vanilla** `/v1` request — no loop magic, no forced `tool_choice`. The most likely
concrete differentiator: **sampling**. qwen3.6's ollama default is `temperature 1.0`
(+ top_p 0.95, presence_penalty 1.5) — high variance that fits the flaky engagement. Our
`/v1` delegation and native paths send no temperature, so ollama uses that 1.0 default.

Lowering temperature (~0.2–0.3) on the agentic path is a small, upstream-clean change that
should make tool-call emission more deterministic — and it's exactly the lever pi-style
reliability points at, rather than the reverted loop guard (M-MOTOKO-COMPEL-WRITE).

## Approach (this doc: the code knob; A/B is separate, needs GPU)

Add an **opt-in, off-by-default** temperature override to the ollama provider
(`internal/ai/ollama`):

- New env `AILANG_OLLAMA_TEMPERATURE` (float). When set to a value `> 0`, the ollama Step
  applies it on **both** paths:
  - `/v1` delegation: set `r2.Temperature` before delegating to the OpenAI provider.
  - native `/api/chat`: `options["temperature"]`.
- Precedence: an explicit `req.Temperature > 0` wins; otherwise the env; otherwise unset
  (ollama uses the model default — **today's behaviour, unchanged**).
- Helper `resolveOllamaTemperature(reqTemp float64) float64` keeps it one place + testable.

**Default behaviour is identical to today** (env unset → no temperature sent). So committing
this to `dev` is a no-op for the rotation until the A/B explicitly sets the env. That keeps
us from landing an unvalidated behaviour change (the lesson from the reverted guard).

## Acceptance criteria (code part — no GPU)
- [ ] `AILANG_OLLAMA_TEMPERATURE=0.2` ⇒ the `/v1` request body carries `"temperature":0.2`.
- [ ] Env unset ⇒ no temperature sent (byte-for-byte today's request).
- [ ] `req.Temperature > 0` takes precedence over the env.
- [ ] `internal/ai/...` builds; unit tests cover the resolver + the `/v1` wiring; existing
      ollama tests still pass.

## Validation (GPU follow-up, separate run)
- A/B on the 6 historically-flaky benchmarks ×N, lock-respecting: `AILANG_OLLAMA_TEMPERATURE`
  unset (control) vs `0.2`/`0.3` (treatment). Metric: WriteFile + pass rate. Wire the env in
  the motoko executor for the treatment arm. Only then consider making a low temperature the
  default for the agentic path.

## Out of scope
- Thinking-mode gating (`enable_thinking`) — the second lead; separate cycle if temperature
  doesn't fully close it.
- Changing the default temperature (stays unset until the A/B justifies it).

## References
- Diagnosis + pi reverse-engineering: [motoko-harness-analysis-log.md](../motoko-harness-analysis-log.md) (2026-06-17 entries)
- qwen3.6 ollama params: `temperature 1.0, top_p 0.95, presence_penalty 1.5` (api/show).
