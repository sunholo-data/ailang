# M-AI-OPENAI-LOCAL-ENDPOINT-RELAX — Allow empty OPENAI_API_KEY when OPENAI_BASE_URL is custom

**Status**: Planned
**Target**: v0.17.0 (or v0.16.x patch — small focused fix)
**Priority**: P2 (blocks local-DGX / self-hosted-LLM use cases that don't authenticate)
**Estimated**: ~30 minutes
**Dependencies**: None
**Surfaced by**: motoko_agent fork (`.agent/learnings/2026-05-03-local-openai-endpoint-key-relaxation.md`); the workaround was forked in their AILANG copy and **lost during the v0.16.x fork-retirement** because it didn't ride upstream.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No semantic change; same code path runs whether `OPENAI_BASE_URL` is set or not |
| A2: Replayability | 0 | Replay harness already records the resolved endpoint; no new state |
| A3: Effect Legibility | 0 | No effect change |
| A4: Explicit Authority | **+1** | Local-endpoint use case becomes correctly representable: AI cap is still required, only the auth precondition relaxes when an explicit `OPENAI_BASE_URL` is provided. Authority remains explicit; the fix removes a bogus extra gate that wasn't enforcing real authority |
| A5: Bounded Verification | 0 | None |
| A6: Safe Concurrency | 0 | None |
| A7: Machines First | **+1** | The error message gains the fix-hint about `OPENAI_BASE_URL` — agent-readable, not just human-readable |
| A8: Minimal Syntax | 0 | No syntax change |
| A9: Cost Visibility | 0 | None — cost tracking still applies (zero-cost for local endpoints) |
| A10: Composability | **+1** | Built-in OpenAI provider now composes with the entire local-LLM ecosystem (vLLM, TGI, llama.cpp, LM Studio, Ollama-compat) without forcing a config-driven provider workaround |
| A11: Structured Failure | **+1** | Error message becomes more actionable when it does fire (cloud endpoint without key) — explicitly suggests the fix |
| A12: System Boundary | 0 | None — boundary is unchanged; only the auth precondition narrows |

**Net Score: +4** → **Decision: ✅ Proceed to implementation**

### Hard Violation Check

- [x] A1 (Determinism): No
- [x] A3 (Effects): No
- [x] A4 (Authority): No — improves, doesn't relax
- [x] A7 (Machines First): No — improves

## Conflict Surface

**N/A** — this milestone touches only `cmd/ailang/ai_handlers.go` and `cmd/ailang/exec.go` (CLI handler setup). It does not extend any syntactic or semantic position the parser, typechecker, codegen, or stdlib resolves. Changes are localized to two `if apiKey == ""` checks inside two case branches.

## Problem

`cmd/ailang/ai_handlers.go` line 198 and `cmd/ailang/exec.go` line 352 both unconditionally require `OPENAI_API_KEY` for any model with `provider: openai`:

```go
case ai.ProviderOpenAI:
    apiKey := os.Getenv("OPENAI_API_KEY")
    if apiKey == "" {
        return fmt.Errorf("OPENAI_API_KEY environment variable required")
    }
    client := openai.NewClient(apiKey)
    handler = client.NewHandler(modelName, opts...)
```

This breaks the legitimate use case where `OPENAI_BASE_URL` points at a **local/self-hosted endpoint that doesn't authenticate** — e.g. a vLLM / TGI / Ollama-compat server on a developer's local network or DGX. The motoko_agent fork carried a workaround that allowed the empty key when `OPENAI_BASE_URL` was set non-empty:

```go
customBaseURL := strings.TrimSpace(os.Getenv("OPENAI_BASE_URL"))
if apiKey == "" && customBaseURL == "" {
    return fmt.Errorf("OPENAI_API_KEY environment variable required (or set OPENAI_BASE_URL for a custom unauthenticated endpoint)")
}
```

When motoko_agent migrated off the fork at v0.16.x, this fix didn't ride upstream — it was a fork-only patch. Any user who wants to point AILANG at a local LLM endpoint hits the original error.

## Scope

Apply the same conditional relaxation upstream:

- `cmd/ailang/ai_handlers.go#setupAIHandlerFromConfig` line 198: same gate.
- `cmd/ailang/exec.go` line 352: same gate.
- One unit test per call site verifying:
  - `OPENAI_API_KEY=set, OPENAI_BASE_URL=unset` → handler creates (existing behaviour)
  - `OPENAI_API_KEY=unset, OPENAI_BASE_URL=unset` → returns the error (existing behaviour)
  - `OPENAI_API_KEY=unset, OPENAI_BASE_URL=http://localhost:8000/v1` → handler creates with empty key (new behaviour)
  - `OPENAI_API_KEY=set, OPENAI_BASE_URL=http://localhost:8000/v1` → handler creates (no change — explicit key wins)

## Acceptance criteria

- [ ] Both code paths gate as documented above
- [ ] Error message includes the new fix-hint: `"OPENAI_API_KEY environment variable required (or set OPENAI_BASE_URL for a custom unauthenticated endpoint)"`
- [ ] 4-case test table per code site, all green
- [ ] CHANGELOG entry under the target release's "Fixed" section, naming motoko_agent's `local-openai-endpoint-key-relaxation.md` as the source learning
- [ ] No behaviour change for the default cloud endpoint (`https://api.openai.com/v1`) — it still requires a key

## Why this is upstream-relevant, not motoko-only

The OpenAI-compat HTTP shape is the de facto standard for local-LLM hosting (vLLM, TGI, llama.cpp's server, Ollama's `/v1/chat/completions`, LM Studio, etc.). Forcing `OPENAI_API_KEY=anything` on every AILANG user with a local model is friction-without-purpose — the key isn't actually checked by the local endpoint; it's just a syntactic gate AILANG adds. Removing it (when `OPENAI_BASE_URL` is custom) makes AILANG work out-of-the-box with the entire local-LLM ecosystem.

## Out of scope

- Wider auth-shape changes (OpenRouter, Anthropic, Gemini already have their own auth paths and don't need this).
- Adding a config-driven provider for "OpenAI-compat with no auth" — that's a bigger ergonomic question and config-driven providers (M-AI-PROVIDER-CONFIG) already cover it via the `auth = none` pattern. This doc is about making the *built-in* OpenAI provider work with `OPENAI_BASE_URL` overrides, not adding new ones.

## Related Documents

Discovered via `ailang docs search --neural`:

- **[M-AI-OLLAMA — Unified Ollama Provider for Local Models](../../implemented/v0_7_0/m-eval-ollama-local-models.md)** (similarity 0.44). The Ollama provider already supports unauthenticated local endpoints natively; this doc is the OpenAI-compat parallel for vLLM / TGI / llama.cpp / LM Studio.
- **[M-EVAL-HTTP — Fix HTTP/JSON Syntax Errors](../../implemented/v0_3_22/M-EVAL-HTTP-FIX-sprint-plan.md)** (0.43). Historical context for the HTTP transport layer this fix sits in.
- **[M-EXEC — Multi-Executor Support](../../implemented/v0_6_1/m-exec-multi-executor-support.md)** (0.42). Adjacent: the executor layer already abstracts over multiple LLM endpoints; this doc removes one of the last asymmetries between cloud + local in the AI-effect path.

## Cross-references

- Source learning: [motoko_agent/.agent/learnings/2026-05-03-local-openai-endpoint-key-relaxation.md](https://github.com/sunholo-data/motoko_agent/blob/dev/.agent/learnings/2026-05-03-local-openai-endpoint-key-relaxation.md)
- Related: [M-EXTERNAL-CONSUMER-DX](m-external-consumer-dx.md) (motoko-surfaced DX gaps; this is a sibling external-consumer fix that the design-doc just missed when it was scoped)
- Related: [M-AI-PROVIDER-CONFIG](../../implemented/v0_15_0/m-ai-provider-config.md) (the `auth = none` shape exists for config-driven providers; this doc closes the same hole for the built-in provider)
