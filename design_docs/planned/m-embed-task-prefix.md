# M-EMBED-TASK-PREFIX: Model- and Role-Aware Embedding Task Prefixes

**Status**: Planned
**Target**: v0.25.x
**Priority**: P1 — μRAG retrieval quality is silently degraded; this is the likely real unlock for whether μRAG helps eval pass rates
**Estimated**: ~150–250 LOC + tests; 0.5–1 day
**Owner**: sunholo (eval rig)

## Problem

The Ollama embedder sends **raw text** to `/api/embed` with no task-instruction
prefix, for both corpus documents and query text. But modern instruction-tuned
embedders **require** task prefixes prepended to every input:

- **EmbeddingGemma** (now the configured embedder, `ollama:embeddinggemma`,
  768-dim) — REQUIRED (per Google's model card):
  - Query (retrieval): `task: search result | query: {text}`
  - Query (code):      `task: code retrieval | query: {text}`
  - Document:          `title: none | text: {text}`
- **nomic-embed-text** — benefits from `search_query: {text}` / `search_document: {text}`.

Without the prefixes, retrieval relevance drops sharply. Observed after switching
to embeddinggemma: `ailang cache search "roman numeral"` → *time-and-dates* (cosine
~0.47), where a correctly-prefixed query should return *number conversions*. This
sits on top of (and was masked by) the now-fixed 0-embeddings bug
([[microrag-needs-nomic-embed]] — nomic-embed-text was never pulled, so the corpus
had 0% embeddings). With embeddings present but un-prefixed, retrieval is mediocre.

**Consequence:** every microRAG A/B to date tested degraded retrieval, so the
"μRAG doesn't help / −3pp" results are not a verdict on μRAG — they are a verdict
on a misconfigured embedder.

## Root cause

`internal/messaging/embedder.go` → `OllamaEmbedder.Embed(text)` / `embedSingle(text)`
embeds the bare string. The `Embedder` interface
(`Embed(text string) ([]float32, error)`) carries no notion of **query vs document**
role, and nothing is model-aware about prefixing. So the same raw call is used for
corpus documents (index time) and queries (retrieval time).

## Approach (no interface change)

Adding a role to the `Embedder` interface would ripple to every implementer
(`OllamaEmbedder`, `OpenAIEmbedder`, `GeminiEmbedder`) **and** every mock in tests.
Avoid that. Instead add a **free-function wrapper** + a pure **prefix table**:

```go
// internal/messaging/embed_prefix.go (new)
type EmbedRole int
const ( RoleDocument EmbedRole = iota; RoleQuery; RoleCodeQuery; RoleNone )

// applyTaskPrefix returns text with the model+role-appropriate instruction
// prefix prepended. Unknown models → text unchanged (safe no-op).
func applyTaskPrefix(model string, role EmbedRole, text string) string { … }

// EmbedWithRole wraps any Embedder, prefixing before delegating to Embed().
func EmbedWithRole(e Embedder, role EmbedRole, text string) ([]float32, error) {
    return e.Embed(applyTaskPrefix(e.ModelName(), role, text))
}
```

Prefix table (model substring match, case-insensitive):

| model | RoleDocument | RoleQuery | RoleCodeQuery |
|---|---|---|---|
| `embeddinggemma` | `title: none \| text: {t}` | `task: search result \| query: {t}` | `task: code retrieval \| query: {t}` |
| `nomic-embed-text` | `search_document: {t}` | `search_query: {t}` | `search_query: {t}` |
| other | `{t}` (no-op) | `{t}` | `{t}` |

### Wire ONLY the μRAG paths (leave messaging envelopes alone)

- **Corpus documents (index time):** the `BrainStore` embed path that calls
  `embedder.Embed(f.Content)` (`internal/effects/sharedmem_sqlite.go:334`,
  reached via `ailang cache embed` → `cmd/ailang/cache_ops.go:runCacheEmbed`).
  → `RoleDocument`.
- **Eval-time query (retrieval):** `internal/microrag/engine.go` query-embed
  (the buildQuery → embedding-search path). → `RoleQuery` (or `RoleCodeQuery`
  when the benchmark is code-synthesis; default `RoleQuery`).
- **CLI query:** `cmd/ailang/cache_ops.go:63,79` (`ailang cache search`).
  → `RoleQuery`.
- **DO NOT touch** `internal/messaging/envelope_builder.go` (message-routing intent
  vectors) or `search_neural.go` (message search) — a separate subsystem; leaving
  them as `Embed()` / `RoleNone` keeps their behaviour identical.

Because the effects-package `Embedder` interface (sharedmem_sqlite.go:21) is a
duplicate of the messaging one, the wrapper must be reachable from both — put the
pure `applyTaskPrefix` where both can import it (or duplicate the tiny pure table),
and apply `RoleDocument` inside the BrainStore embed call.

## Acceptance criteria

1. `applyTaskPrefix` unit-tested for embeddinggemma + nomic × {document, query,
   code-query} and the unknown-model no-op.
2. After implementing + `make brain-index-syntax-reset`:
   `ailang cache stats` → `With embeddings: 100% (ollama:embeddinggemma)`.
3. Retrieval relevance restored:
   `ailang cache search --namespace ailang-syntax "roman numeral"` returns
   number/math content (not time-and-dates); scores recover toward ~0.6+.
4. Corpus documents and queries use the SAME model with matching roles (no
   vector-space mismatch).
5. messaging envelope/search behaviour byte-for-byte unchanged (existing tests pass).
6. Re-run the microRAG A/B (opencode-qwen3-5, core, on vs off) on the prefixed
   corpus and record the delta — the first valid μRAG measurement.

## Axiom compliance

| Axiom | Score | Justification |
|---|---|---|
| A7 Machines First | +1 | Correct retrieval is core to AI-context tooling |
| A9 Cost Visibility | 0 | Local embedder, $0 |
| A11 Structured Failure | +1 | Pairs with task_d9924014 (fail loudly on 0 embeddings) |
| others | 0 | No language/effect/syntax change |

**Net +2 → Proceed.**

## Out of scope / follow-ups

- Making embeddinggemma the **shipped default** (repo default is still nomic at
  `embedder.go:60`; rig overrides via `~/.ailang/config.yaml`) — decide after the
  prefixed A/B shows which embedder wins.
- Loud-failure on missing embedder/0-embeddings: **task_d9924014**.
- Matryoshka dimension truncation (768→256) for speed — not needed now.

## References

- Google EmbeddingGemma model card: https://ai.google.dev/gemma/docs/embeddinggemma/model_card
- Predecessor finding: [[microrag-needs-nomic-embed]]
- Eval-rig embedder config: `~/.ailang/config.yaml` (embeddings.ollama.model)
