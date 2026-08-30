# M-DX-MICRORAG-CONTEXT — μRAG as a first-class pi extension: retrieval-triggered context injection + search tool

**Status**: Planned
**Target**: v0.35.0
**Priority**: P1
**Estimated**: ~1 session (M1+M2 ~0.5d, M3 ~0.25d, e2e + A/B ~0.5d)
**Dependencies**: M-BRAIN-MICRORAG (implemented v0.15.0 — the engine), M-MICRORAG-HOOK-EXPANSION (implemented v0.15.0 — establishes the trigger doctrine), M-DX-PI-HARNESS (shipped — extension suite + distribution it joins), ADR-002 (constraint)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Developer-experience tooling (pi extension), no language surface change.

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No language/runtime impact |
| A2: Replayability | 0 | No trace impact |
| A3: Effect Legibility | 0 | No effect system change |
| A4: Explicit Authority | +1 | Injections are explicit engine calls with a dedup ledger + session budget; every injected block names its query and corpus (traceable provenance) |
| A5: Bounded Verification | +1 | Injection is budget-capped and relevance-floored (engine's existing floors/budgets); subprocess calls run under the Subprocess Contract (timeout, 64KB cap) |
| A6: Safe Concurrency | 0 | No concurrency surface |
| A7: Machines First | +1 | Replaces "memorize the docs prompt" with structured retrieval; works identically in `-p`/RPC/headless eval runs |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | 0 | Engine calls are local (no metered API calls); injection content is visible in transcript (`display: true`) |
| A10: Composability | +1 | Wraps existing `ailang micro-rag` CLI — never re-implements retrieval (CLAUDE.md principle 1); composes with the session gate and `--microrag on/off` eval flag |
| A11: Structured Failure | +1 | Engine misses degrade to "no injection" (structured skip), never to fabricated context; error-triggered path turns opaque compiler errors into code-tagged fix guidance |
| A12: System Boundary | +1 | Makes the .ail↔docs boundary explicit per query; `microrag_search` returns structured envelope JSON |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 / A3 / A4 / A7 — no violations (A4/A7 strengthened)

## Verification Log

| # | Claim | Method | Result |
|---|-------|--------|--------|
| V1 | `ailang micro-rag` subcommands: `context`, `user-prompt`, `lint-builtin`, `init`, `bootstrap`; `--tool/--file/--content/--config` flags | Ran `ailang micro-rag --help` live this session | Confirmed |
| V2 | μRAG measurably lifts pass rate: paired A/B 2026-07-27, **on 65/84 = 77.4% vs off 54/84 = 64.3%** (+13.1pp) | Computed from `internal/eval_analysis/testdata/paired/microrag_20260727_{on,off}.json` (84 rows each) | Confirmed |
| V3 | **No pi frontend exists**: `.agents/skills/microrag/frontends/` contains only `claude-code`, `codex`, `gemini` (+README); the v0.15.0 hook-expansion doc lists "opencode and Pi don't have μRAG hooks yet" as its M3 follow-up | `ls` + read `m-microrag-hook-expansion.md` §Context | Confirmed (negative-existence) |
| V4 | pi `before_agent_start` can inject a persistent message (`return { message: { customType, content, display } }`) without touching the system prompt | pi 0.84.3 `docs/extensions.md` §before_agent_start (L530–560) | Confirmed from docs |
| V5 | pi extensions can register LLM-callable tools via `pi.registerTool()` | pi docs L10/L78; **demonstrated by shipped code**: `session-protocol-gate.ts` registers `session_protocol_ack`, exercised end-to-end | Confirmed by working code |
| V6 | File-content-triggered μRAG injection was **deliberately disabled** as embedding-noisy; dense NL queries (user prompt / error intent) are the proven query type | `design_docs/decisions/ADR-002-pretooluse-microrag-disabled.md`; `cmd/ailang/microrag.go` help text: "user-prompt … the right hook for embedding-driven retrieval — see ADR-002" | Confirmed by reading |
| V7 | Engine-side gates exist and are env-driven: `AILANG_MICRORAG_ENABLED/ROUTES/DRYRUN/SESSION/USERPROMPT_FLOOR` | `internal/microrag/engine.go` consts (read) | Confirmed |
| V8 | Search/embedding response caches exist (search TTL 240s, embedding TTL 1d) | `internal/microrag/engine.go` `searchCacheTTLSecs`, `embedCacheTTLDays` | Confirmed |
| V9 | Dedup ledger + per-session budget prevent repeat injection of the same chunk | `m-microrag-hook-expansion.md` §M1 behaviour ("same dedup ledger and session budget") | Confirmed by reading |
| V10 | The eval suite already exposes a controlled A/B switch: `--microrag on/off/auto` forces `AILANG_MICRORAG_ENABLED` in subprocesses | `internal/eval_harness/microrag_mode.go` | Confirmed |
| V11 | Error / hit envelopes are structured, not prompts: `SearchHit`/`SearchEnvelope` with tier/score/content fields | `internal/microrag/engine.go` | Confirmed |
| V12 | `ailang_check` gives structured diagnostics `{code, message, file, line, col, hint}` inside pi sessions | Shipped `ailang-lsp-lite.ts` extension (M-DX-PI-HARNESS Stream B), in active use | Confirmed by working code |

## Problem Statement

μRAG is the repo's proven just-in-time knowledge engine (+13.1pp paired A/B on agent-mode evals), with frontends for Claude Code, Gemini CLI, and Codex CLI — but **the harness this repo's daily work and rig evaluations actually run in, pi, has none.** Consequences today:

1. **pi sessions pay the residency tax**: agents that need AILANG knowledge either carry the full ~2.5k-token teaching prompt or work from memory. The delivery A/B (`m-eval-prompt-delivery`) already proved residency is the wrong axis — a compact prompt + front-loaded card ties the full prompt at ~1/7 the tokens. Retrieval is strictly better than residency when the retrieval trigger fires.
2. **Rig agent-mode evals can't flip the μRAG arm in pi sessions.** The eval harness supports `--microrag on/off` via env gates, but since pi sessions never retrieve, the flag is inert there — the +13.1pp measured on Claude Code hooks has never been replicated on the pi path we benchmark through.
3. **The error-triggered lane is entirely missing.** Our `ailang_check` tool returns structured `{code, ...}` diagnostics, then stops. The external reference implementation (little-coder, studied 2026-08-30) puts *error recovery* first in its injection selector (error-recovery > recency > intent) because that is where retrieval has the highest per-token value. μRAG + structured error codes is a natural fit — an error code plus one-line message is exactly the "dense, intent-bearing query" shape ADR-002 identified.

**Reference-note:** little-coder (built on the same pi) ships this pattern as `skill-inject`/`knowledge-inject` and measured it as a top-3 lever on small models. Its selector is a hand-rolled corpus guess; μRAG is the semantic, evaluated version. Their precedent is motivation, not requirement.

## Goals

**Primary goal:** pi sessions get μRAG retrieval on the two proven query types (user prompt intent, structured error codes) plus an on-demand search tool — engine untouched, so the +13.1pp evidence transfers through the same env-gated A/B switch.

**Success metrics:**
1. Known-prompt injection e2e: prompt shaped like the ADR-002 M1 tests ("how do I concat strings in AILANG?") → the right chunk injected exactly once per session.
2. Below-floor prompts ("what's the weather?") → zero injection.
3. Error-triggered e2e: `ailang_check` returning `IMP010` in turn N → relevant fix guidance injected at turn N+1.
4. `--microrag on/off` agent-mode eval pair on 2+ benchmarks: on ≥ off (no regression), with injection events visible in transcripts for audit.
5. Cache safety: system prompt is byte-identical across turns in all three modes (injection only ever adds messages).

## High-Impact Decisions

| # | Decision | Rationale |
|---|----------|-----------|
| D1 | **Inject as trailing `message`, never by mutating `systemPrompt`** — even though pi's `before_agent_start` allows both | The system prompt is the front of every request; mutating it invalidates the provider prefix cache (little-coder #73, measured on llama.cpp). On the Studio rig this would tax every rig turn. Trailing messages keep the prefix byte-stable. |
| D2 | **No file-content-triggered injection** (no PreToolUse-equivalent on `.ail` reads/edits) | ADR-002 disabled exactly this: file-body queries are embedding-noise floors. Do not re-litigate by default; if revisited, it must ship with its own A/B. |
| D3 | **Error-triggered path launches via `user-prompt` with a synthetic dense query** (template from `{code} {message}`); a dedicated `ailang micro-rag error --code` subcommand is the fallback if the synthetic query underperforms the floor | Zero engine change at first cut; the code+one-line-message template is still dense NL. The subcommand upgrade is small, gated on A2 evidence (see Deferred Decisions). |
| D4 | **Reuse engine budgets/dedup as-is** (`AILANG_MICRORAG_SESSION` session key; ledger + budget) | The engine already solved the two classic RAG failure modes (repeat-injection, unbounded context). Duplicating them extension-side would drift. |
| D5 | **Distribution rides the existing extension suite** (`.pi/extensions/`, Tier 0 `ailang pi install`) and the Subprocess Contract | No new install surface; freshness/staleness semantics inherited from the suite's doctrine. |

## Solution Design

One TypeScript extension, `microrag-context.ts`, joining the eight existing suite members. Every wrapper subprocess runs under the Subprocess Contract (per-command timeout, structured TIMEOUT failure, 64KB output cap). Gate interplay: read-only `micro-rag` calls are allowlist-safe beyond the session gate (no writes).

### M1 — Prompt-intent injection

Hook: `before_agent_start`. Only fires when `event.prompt` is non-trivial (> 8 words or contains an AILANG token — cheap pre-gate to skip pure commands like `/sprint-start`).

```
event.prompt ──► pre-gate ──► ailang micro-rag user-prompt (stdin: prompt JSON; env: AILANG_MICRORAG_SESSION=pi:<sessionId>)
                 │
                 ├─ hits above floor → return { message: { customType: "microrag-inject",
                 │        content: "<chunk provenance header>\n\n<content>", display: true } }
                 └─ below floor / disabled / engine error → return {}   (silent no-op; notify only on engine *failure*, not on floor-miss)
```

Envelope parsing matches the existing frontend scripts (same `ailang micro-rag context`/`user-prompt` output contract; identical stdin schema, so the existing engine dedup/budget apply unchanged).

### M2 — Error-triggered injection

The extension tracks the last structured check result via a one-entry buffer:

- `tool_result` hook: when `event.toolName === "ailang_check"` and `event.details`/content carries non-empty `codes` (V12 shape), store `{codes, firstMessage, turn}` in memory + `pi.appendEntry("microrag-last-error", …)` for /resume reconstruction (same State Management pattern the session gate uses for ack state).
- Next `before_agent_start` (i.e., the model's next attempt): if a buffered error exists and was not yet served, fire the D3 synthetic query → inject (dedup prevents re-injecting the same chunk). Buffer is clear-on-serve, max depth 1 — the newest error wins; no queue that could bloat context.
- Cross-check: injecting on the *same* turn as a `tool_result` modification would fight the parallel-tool contract (siblings interleave, L842–846); next-turn injection is order-safe and matches how the model consumes feedback anyway.

### M3 — On-demand search tool

`pi.registerTool` — name `microrag_search`, parameters `{ query: string, corpus?: "syntax" | "builtins" | "docs" }`, executes `ailang micro-rag user-prompt` (or `search`-equivalent route per corpus) and returns the envelope JSON `{ hits: [{tier, score, content, source}] }` back-flat (details = envelope, content = pretty excerpt ≤ 1KB/hit, cap 3 hits). Tool description includes one paraphrase example so small models use it before guessing APIs. Cost: one line in the system prompt's tool snippets — journal how much of the +13pp residency win survives the tool-snippet cost.

### Config & lanes

- Honors `AILANG_MICRORAG_ENABLED` from the environment first (so `--microrag off` in eval harness disables everything — arm parity with today's switch semantics).
- Extension-level kill switch `PI_MICRORAG_CONTEXT=0` for users who want search tool but no auto-injection (and vice versa).
- No model-lane gating needed: injection content is small and relevance-floored; keep uniform behavior for eval comparability (deliberate contrast with little-coder's cloud-model auto-off — we measure rather than assume).

## Examples

**Before (pi session, current):** model hits `TC010: cannot unify int with string`, flails, re-reads type docs from memory, 3 wasted turns.

**After (M2):**

```
[turn N]   ailang_check("...leted.ail") → {code:"TC010", message:"unification failed: int vs string", line:12}
[turn N+1] (before_agent_start serves buffer)
           ── μRAG · type・unify 〔micro-rag: syntax, score 0.71〕 ─────────
           TC010: unification failures — check the annotated literal types; numeric
           literals default to Num[a]. Use explicit annotation `x = 5 : int` or
           `fromInt`. (docs/reference/types#defaulting)
[turn N+1] model applies explicit annotation → passes on attempt 1
```

**A/B invocation (already wired in the harness):**

```bash
ailang eval-suite --microrag on  --suite agent-core --model ollama/qwen3.6:9b
ailang eval-suite --microrag off --suite agent-core --model ollama/qwen3.6:9b
ailang eval-paired eval_results/<microrag-on-…> eval_results/<microrag-off-…>
```

## Success Criteria

- [ ] M1: known-prompt → known-chunk e2e (ADR-002 M1's three test cases ported: concat / pattern match / below-floor no-op)
- [ ] M2: IMP010→fix-guidance e2e (injection appears on next turn; buffer cleared; second turn with same code does not re-inject)
- [ ] M3: `microrag_search` tool returns envelope JSON for a known query; below-floor query returns empty hits without error
- [ ] System prompt byte-identical across turns with extension active (test asserts via `before_provider_request` snapshot)
- [ ] `--microrag off` and unset env → extension inert (no subprocess spawned)
- [ ] Session gate unaffected (extension is read-only; no bash outside allowlist)
- [ ] A/B on ≥ 2 rig benchmarks with no pass-rate regression vs off; paired report archived under `eval_results/`
- [ ] README table in `.pi/extensions/README.md` updated; Tier-0 embedded copy bumps (`ailang pi install` serves the new file)

## Testing Strategy

1. **Unit/e2e (extension):** follow the session-gate extension test pattern (headless `pi -p` runs, captured transcripts) for M1/M2/M3 flows incl. below-floor and disabled arms.
2. **Contract test (engine parity):** assert the extension's stdin schema matches the claude-code shim (`microrag_userprompt.sh`) so frontends stay behaviorally identical — one test importing both stdin fixtures.
3. **A/B (rig or OpenRouter cheap tier):** paired eval run with the existing harness flag; success = no regression + documented injection-event count in transcripts.
4. **Cache discipline check:** provider-observatory spans (CH field) on a 10-turn session confirm no prefix invalidation vs. baseline.

## Deferred Decisions

- **D3 fallback** (`ailang micro-rag error --code` Go subcommand): only if the synthetic-query arm fails the floor in the M2 e2e; requires its own tiny Go change + test, split into a follow-up if triggered.
- **opencode frontend parity** (the other harness named in M3 of the v0.15.0 doc): same shim mechanics, different hook format — separate doc if benchmark demand appears; not this doc's scope.
- **File-content re-introduction** would need a cosine-shaping change at the engine level (ADR-002's actual diagnosis) — explicitly out of scope even if some other trigger wins the A/B.

## Non-Goals

- No changes to `internal/microrag` engine behavior, corpora, or scoring (D3 fallback excepted).
- No PreToolUse-style file reads (D2), no MCP bridge (the `cmd/ailang-microrag-mcp` binary exists; a subprocess CLI call is strictly simpler for a pi extension and doesn't add a protocol dependency — re-evaluate only if a second harness family adopts MCP).
- No prompt-template or teaching-prompt changes (that's `m-eval-prompt-delivery`'s lane; this doc makes retrieval available, retirement of residency is a separate decision with its own A/B).
- Never bypasses the session protocol gate; `micro-rag` reads remain allowlisted subprocess calls.

## Timeline

| Phase | Work | Est. |
|---|---|---|
| 1 | M1 prompt-intent injection + e2e (3 ADR-002 cases) | 0.5d |
| 2 | M2 error-triggered buffer + e2e | 0.5d |
| 3 | M3 search tool + README/Tier-0 install | 0.25d |
| 4 | Rig A/B + paired report + observatory CH check | 0.5d |

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Rig binary staleness breaks engine calls (recurring repo trap) | Subprocess Contract TIMEOUT envelope + equivalence with `freshness_report` doctrine: structured failure surfaced via `ctx.ui.notify`, never silent fallthrough |
| Ollama embedding latency on the rig (engine caches only soften cold queries) | Cached paths (V8) hit 240s/+1d TTLs; pre-gate skips pure commands; timeout converts to structured no-op injection |
| Injection bloats small context windows | Engine dedup + session budget (V9) + max-depth-1 error buffer (M2) + tool-snippet cost measured in M3 |
| M2's synthetic query shape misses the floor on real error messages | D3 fallback path pre-specified; failure is measurable in the e2e before any rig spend |
| A/B confound: both arms share scaffold | Runs are paired via existing `--microrag` env forcing (V10) — same pipeline, single delta |

## Related Documents

- [M-BRAIN-MICRORAG](../../implemented/v0_15_0/m-brain-microrag.md) — the engine (implemented v0.15.0); this doc adds a frontend, not engine work
- [M-MICRORAG-HOOK-EXPANSION](../../implemented/v0_15_0/m-microrag-hook-expansion.md) — trigger doctrine (UserPromptSubmit > PreToolUse), names pi as the missing harness (M3). Similarity note: topic-adjacent but distinct — that doc covered Claude/Gemini/Codex mechanics; this doc is the pi surface + error-triggered lane, which no frontend has anywhere.
- [ADR-002](../../decisions/ADR-002-pretooluse-microrag-disabled.md) — why file-content injection is excluded
- [M-DX-PI-HARNESS](m-dx-pi-harness.md) — extension suite, distribution Tiers 0–2, Subprocess Contract this rides
- [m-eval-prompt-delivery](../../implemented/v0_24_0/m-eval-prompt-delivery.md) — compact-prompt-beats-residency delivery evidence
- [M-EVAL-SEMANTIC-CONTEXT](../v0_29_0/m-ailang-semantic-context.md) — convergence-efficiency context (compaction/hygiene sibling workstream)
- [M-MODEL-REGISTRY-SINGLE-SOURCE](m-model-registry-single-source.md) — contextWindow registration rider (rig overflow guard) lives there
- External: little-coder (`skill-inject`, `knowledge-inject`; paper *Honey, I Shrunk the Coding Agent*) — precedent measurement, not a code dependency

## References

- `internal/microrag/engine.go`, `cmd/ailang/micro-rag` CLI (verified live)
- Paired A/B data: `internal/eval_analysis/testdata/paired/microrag_20260727_{on,off}.json`
- pi 0.84.3 `docs/extensions.md` §before_agent_start, §tool_result, registerTool API
- `.agents/skills/microrag/frontends/README.md` — stdin schema + `hookSpecificOutput.additionalContext` envelope

## Future Work

- Route telemetry: count injections per error code → feeds the AILANG-fix backlog with "what agents actually get wrong" signal (same instrument spirit as `ailang chains`).
- If M3's tool proves popular, a `microrag_search --k <n>` and corpus union search in the engine (small Go change, separately sprinted).
- Cross-harness injection ledger export to compare per-harness injection utility.