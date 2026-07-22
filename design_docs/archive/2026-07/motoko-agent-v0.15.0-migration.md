# M-MOTOKO-MIGRATE-V0.15.0: Migrate motoko_agent off the AILANG fork onto upstream v0.15.0

**Status**: Planned (draft PR shipped to arniwesth/motoko_agent for review)
**Target**: arniwesth/motoko_agent main branch
**Companion**: AILANG side complete — v0.15.0 shipped with M-AI-PROVIDER-CONFIG, M-AI-STREAMING-HELPER, M-AI-EFFECT-MODES, M-AI-OPENROUTER follow-ups
**Owner**: Mark + Claude
**Created**: 2026-05-05
**Phase**: 3 of [motoko-integration-sequence.md](./motoko-integration-sequence.md) — the actual cross-repo PR work

---

## Executive Summary

The motoko_agent project currently depends on a **fork of AILANG** (`github.com/sunholo-data/ailang` `motoko` branch) cloned at install-time by `scripts/install-prerequisites.sh`. The fork existed for three reasons that are all now obsolete in upstream v0.15.0:

1. **Custom OpenAI base-URL routing** → expressible as `[[ai_provider]]` block in `ailang.toml` (M-AI-PROVIDER-CONFIG)
2. **Token streaming** (`std/ai_motoko.callStreamResult`, 6 Go files) → `std/ai/streaming.callStream` (v0.15.1+, drop-in synchronous accumulator) or `openaiCompatStream` + event loop for per-delta UI updates
3. **OpenRouter prefix routing** → built-in `openrouter` provider, wired into `ailang run` (M-AI-OPENROUTER)

This doc captures the full migration plan: what changes, why each change is honest progress (not a sideways swap), and what we need from arni's team before code can land.

**Headline outcome**: motoko_agent runs on upstream `sunholo-data/ailang@v0.15.0` with no fork dependency. The `arniwesth/ailang@motoko` repo can be archived after merge.

---

## Is the new code "better" — or just different?

**Honest answer: better in fundamentals, more boilerplate at the call site.** The new API is a strict superset in terms of capabilities, but it asks the caller to manage an event loop instead of getting a synchronous accumulator.

### What v0.15.0 does that the fork couldn't

| Property | Fork's `callStreamResult` | v0.15.0's `openaiCompatStream` + `onEvent` |
|----------|---------------------------|---------------------------------------------|
| AI capability gating (`--caps AI`) | ❌ Bypassed (streaming op was outside the AI effect) | ✅ Required |
| Per-provider budget tracking | ❌ Bypassed | ✅ Routes through AI handler — same budget machinery as `_ai_call` |
| Trace span shape | ❌ Custom shape | ✅ Identical to non-streaming AI span (verified by snapshot test) |
| Provider configuration | ❌ Hardcoded URL/auth in Go fork | ✅ Declarative `[[ai_provider]]` block; secrets via env-var refs |
| Type-level routing semantics | ❌ Runtime `--allow-routing` flag only | ✅ `!{AI[mode=routeable]}` skips runtime gate; bare `!{AI}` desugars to `!{AI[mode=fixed]}` |
| Cross-platform | ❌ Required forking + maintaining a Go binary | ✅ Plain stdlib + `[[ai_provider]]` config in any package |

### What the new API asks of the caller in exchange

| Concern | Fork | v0.15.0 |
|---------|------|---------|
| Boilerplate at call site | `let r = callStreamResult(...)` (one line, blocks until done) | ~10 lines: `match openaiCompatStream(...) { Ok(conn) => { onEvent(conn, h); runEventLoop(conn); disconnect(conn) }, Err(e) => ... }` |
| Streaming abort | Stdin polling baked into runtime | Caller's `onEvent` handler returns `false` to stop the loop |
| Final-string accumulator | Built into `AIStreamResult.output` | Caller maintains a `mut accumulated: string` (or threads it via `runEventLoopFold` if we add one) |
| Messages serialization | Strongly typed `[Message]` arg | Pre-serialised JSON string (v1.1 will accept typed lists once `std/json` gains record-of-records encoding) |

### v0.15.1 helper: `callStream` closes the boilerplate gap

**Update (2026-05-05): the v1.1 helper anticipated below shipped as M-AI-CALL-STREAM-HELPER and is available from v0.15.1+.** It exposes exactly the synchronous accumulator the fork's `AIStreamResult` provided, with the same effect signature:

```ailang
import std/ai/streaming (callStream, AIError)

callStream(provider: string, model: string, messagesJson: string)
  -> Result[string, AIError] ! {AI, Stream, Net}
```

For motoko_agent's call sites this means each migration shrinks from ~20 lines of `match openaiCompatStream + onEvent + runEventLoop + accumulator` to a single `callStream(...)` call followed by the existing `match Ok(text) | Err(e)` pattern — almost the same diff size as the original `callStreamResult` substitution. **Targeting v0.15.1 means: use `callStream` directly; no need to write `runStreamCall` inline.**

The migration is therefore "swap one streaming call for another" with the v0.15.1 helper restoring the convenience the fork had.

---

## Files needing changes (8 total)

Audit captured 2026-05-05 by grepping motoko_agent main branch:

### Trivial (1 line each)

| File | Line | Change |
|------|------|--------|
| `scripts/install-prerequisites.sh` | 363 | `git clone --branch motoko https://github.com/sunholo-data/ailang` → `git clone --branch v0.15.0 https://github.com/sunholo-data/ailang` |
| `scripts/install-prerequisites.sh` | 10 | Comment update: "AILANG runtime (cloned from upstream sunholo-data/ailang at v0.15.0 tag)" |
| `ailang.toml` | (new field) | Add `ailang = ">=0.15.0"` to `[package]` |

### Real code restructuring (4 .ail files)

Each of these calls `callStreamResult(input, step, stream_id, model)` and expects `AIStreamResult { ok, output, chunks, streamed, ... }`. They need rewriting to use the event-loop pattern.

| File | Current calls | Estimated LOC delta |
|------|--------------|---------------------|
| `src/core/rpc.ail` | 1 call site | +20 LOC (event loop + accumulator) |
| `src/core/ext/compose/compose.ail` | TBD count | +20 per call |
| `src/core/ext/compose/claimcheck.ail` | TBD count | +20 per call |
| `src/core/ext/compose/author_loop.ail` | TBD count | +20 per call |

**Total estimated**: ~80–120 LOC of restructuring across the 4 files.

### TypeScript codegen (1 file)

| File | Line | Change |
|------|------|--------|
| `src/tui/src/env-server.ts` | 642 | The TS file emits AILANG source as a string template. Update the embedded `import std/ai_motoko (callStreamResult)` line to `import std/ai/streaming (openaiCompatStream, onEvent, runEventLoop, disconnect)` AND update the embedded call sites if the codegen produces full call expressions |

### Provider configuration (needs arni's input)

`motoko_agent/ailang.toml` needs `[[ai_provider]]` blocks for whatever endpoints the agent currently hits via the fork. **We don't know these without arni** — possibilities:
- OpenAI direct
- OpenRouter
- A local llama.cpp / vLLM instance
- A custom OpenAI-compat endpoint

Section [§ Questions for arni](#questions-for-arni) below lists what we need to know.

---

## API-shape adaptation pattern

The repeating migration pattern across the 4 `.ail` files. Documenting once here so each call-site rewrite is mechanical.

### Before (fork)

```ailang
import std/ai_motoko (callStreamResult, AIError)

let att_stream_id = stream_attempt_id(stream_id, attempt) in
let r = callStreamResult(input, step, att_stream_id, model) in
if r.ok then Ok(r.output)
else {
  let e = {
    message: r.error_message,
    provider: r.provider,
    statusCode: r.status_code,
    retryable: r.retryable,
    code: r.error_code
  } in
  Err(e)
}
```

The fork's `callStreamResult` accumulates internally and hands back the full string.

### After (v0.15.1+)

With `callStream` shipped in v0.15.1, each call site reduces to a single import + a single function call. Motoko_agent's `AIError` shape needs slight remapping (the upstream `AIError` doesn't carry `provider` / `statusCode` — those can be filled in from caller context) but the structural pattern is identical to the fork's `AIStreamResult.ok` switch:

```ailang
import std/ai/streaming (callStream, AIError)

let messagesJson = "[{\"role\":\"user\",\"content\":\"" ++ escape(input) ++ "\"}]" in
match callStream(providerName, model, messagesJson) {
  Ok(text) => Ok(text),
  Err(e) => Err({
    message: e.message,
    provider: providerName,           -- caller-supplied (not on upstream AIError)
    statusCode: 0,                    -- v0.15.1's AIError omits HTTP status; future enhancement
    retryable: e.retryable,
    code: e.code
  })
}
```

**Reasoning models**: in v0.15.1 `callStream` returns visible content only — `reasoning_content` / `thinking` deltas are read but not surfaced. v0.15.2's `callStreamWithReasoning` will return both. If motoko_agent needs the reasoning trace before then, drop down to `openaiCompatStream` + the manual event loop and inspect deltas yourself.

### Streaming-progress events (the fork's `chunks` array)

The fork's `AIStreamResult.chunks: [AIStreamChunk]` was useful for showing the TUI a token-by-token render. To preserve that in v0.15.0:

```ailang
-- Variant: emit each delta to a callback as it arrives, so the TUI can render incrementally.
-- This is what the fork's chunks array gave you, just without the "wait until done" buffering.
onEvent(conn, \event -> match event {
  SSEData(_, raw) => {
    let delta = extractContent(raw) in
    if delta != "" then onDeltaCallback(delta) else ();
    raw != "[DONE]"
  },
  _ => false
})
```

---

## Smoke test plan (before sending PR)

Before opening the migration PR for review, we verify locally that the migrated code actually runs against a real provider. Two scopes of test:

**Tier 1 (no real network)** — must pass on every commit:
1. `make test` in motoko_agent — all existing tests pass post-migration
2. `ailang check` on every `.ail` file — type-checks cleanly
3. `bun build` (or whatever the TS project uses) — TypeScript compiles

**Tier 2 (real provider, gated on env var)** — must pass before opening PR:
1. Set `OPENAI_API_KEY` (or whatever the chosen provider requires)
2. Run a representative agent loop (e.g. one of the benchmarks) against the new code path
3. Compare output shape to a pre-migration baseline
4. Verify `ailang chains list` shows AI events with the new "streamCall" op name and the provider config picked up

### What we need to be able to run Tier 2

- A test API key for the provider arni's team uses
- A list of model names known to work with their workflow
- An expected baseline output (last successful run on the fork)

These are part of [§ Questions for arni](#questions-for-arni).

---

## Risks and rollback

| Risk | Likelihood | Mitigation |
|------|-----------|------------|
| arni's team relies on `AIStreamResult.chunks` for TUI progress that's not visible to me from the open repo | Med | The "Streaming-progress events" variant pattern above covers this; flag in PR description; ask arni to point at the relevant code |
| Some provider config we don't know about (e.g. AWS Bedrock with SigV4) doesn't fit the v1 `[[ai_provider]]` schema | Low | Bedrock and similar custom-auth providers stay built-in or escape via `auth_headers`; arni confirms which providers they actually use |
| The TypeScript codegen at `env-server.ts:642` emits more than just an import line | Med | Read the full codegen template + emit equivalent v0.15.0 code; smoke-test by running the TUI |
| v0.15.0's `_ai_stream_call` in real workloads has latency / reliability gaps the unit tests didn't surface | Low-Med | Tier 2 smoke test gates the PR; if issues found, file v0.15.1 fix in upstream AILANG and update this branch |
| Rollback needed | Low | Migration is on a dedicated branch; if blocked, leave PR in draft and continue using the fork |

---

## Questions for arni

The PR will land as a draft with this design doc. Before any code lands, we need answers to:

1. **Which AI providers does motoko_agent currently hit?** (OpenAI, OpenRouter, self-hosted, multiple?)
2. **For each provider, what's the endpoint URL, request_shape, and auth method?** (so we can write the correct `[[ai_provider]]` blocks)
3. **What environment variables hold the secrets?** (`OPENAI_API_KEY`, `OPENROUTER_API_KEY`, custom?)
4. **Are there any per-call overrides** (e.g. dynamic model selection via `/model` command in the TUI) that need to flow through the new API?
5. **Is there a reference test prompt + expected output** we can use as a Tier 2 baseline?
6. **Does the team have bandwidth to help with the migration**, or should we deliver a complete diff for review?
7. **Any provider-specific behaviours** (e.g. reasoning_content from o1/DeepSeek-R1) that the fork was handling that we should preserve?

---

## Implementation plan (post-arni-ack)

Once questions are answered, execute in 4 milestones on the `ailang-v0.15.0-migration` branch. **Estimate revised down to ~2 hours after v0.15.1 shipped `callStream`** — the previously planned `runStreamCall` accumulator helper is now upstream, so M2 collapses from 1 hour of inline-helper authoring to ~20 minutes of a smoke check that the new built-in import resolves.

| M | Description | Estimated |
|---|------|-----|
| **M1** | Install script swap to upstream v0.15.1 (or later) + ailang.toml `ailang = ">=0.15.1"` constraint + `[[ai_provider]]` blocks | 30 min |
| **M2** | Verify `callStream` import resolves + Tier 1 smoke (one call site rewritten as a probe) | ~20 min |
| **M3** | Migrate the 4 `.ail` call sites + `env-server.ts` codegen + Tier 1 smoke after each — each call site is now a one-line `callStreamResult(...)` → `callStream(...)` swap plus the `AIError` shape mapping shown above | ~30 min |
| **M4** | Tier 2 smoke against real provider; if green, flip PR from draft to ready-for-review | 30 min + arni's review window |

Total: ~2 hours of focused work post-questions (down from 4-5 hours; the v0.15.1 helper recovers the convenience the fork's `callStreamResult` had).

---

## Related documents

- [motoko-integration-sequence.md](./motoko-integration-sequence.md) — master plan; this doc is its Phase 3 deliverable
- [m-ai-provider-config.md](./v0_15_0/m-ai-provider-config.md) — the upstream feature this migration consumes
- [m-ai-streaming-helper.md](./v0_17_0/m-ai-streaming-helper.md) — same; streaming side
- [docs/docs/guides/custom-ai-providers.md](../../docs/docs/guides/custom-ai-providers.md) — user-facing reference for `[[ai_provider]]` syntax
- [docs/docs/recipes/ai-token-streaming.md](../../docs/docs/recipes/ai-token-streaming.md) — user-facing reference for the new streaming API
- [arniwesth/ailang FORK.md](https://github.com/arniwesth/ailang/blob/motoko/FORK.md) — original fork manifesto (will be archived post-migration)
- motoko_agent local clone: `/Users/mark/dev/sunholo/motoko_agent` on branch `ailang-v0.15.0-migration`

---

**Document created**: 2026-05-05
**Last updated**: 2026-05-05
