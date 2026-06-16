# Sprint Plan: M-EMBED-TASK-PREFIX

**Design doc**: [m-embed-task-prefix.md](m-embed-task-prefix.md)
**Risk**: low-medium (vector-space consistency is the only real risk)
**Duration**: ~0.5 day, single milestone, executed inline
**Branch**: `feat/embed-task-prefix`

## Goal

Make μRAG embeddings model- and role-aware: prepend the correct task-instruction
prefix (EmbeddingGemma / nomic) for documents vs queries, so semantic retrieval
works. No `Embedder` interface change; messaging-envelope embeds untouched.

## Milestone M1 — Task-prefix helper + μRAG wiring (~200 LOC + tests)

### Tasks
1. **Prefix table + helper** (`internal/messaging/embed_prefix.go`, new, ~70 LOC)
   - `EmbedRole` enum (Document, Query, CodeQuery, None)
   - `ApplyTaskPrefix(model string, role EmbedRole, text string) string` — pure,
     model-substring match, unknown→no-op
   - `EmbedWithRole(e Embedder, role EmbedRole, text string) ([]float32, error)`
2. **Unit tests** (`embed_prefix_test.go`, ~80 LOC) — embeddinggemma + nomic ×
   {document, query, code-query} + unknown-model no-op + exact prefix strings
3. **Wire document embed** — BrainStore corpus embed (`internal/effects/
   sharedmem_sqlite.go:334`) → Document role. Resolve the cross-package helper
   (effects can't import messaging if it causes a cycle → put the pure
   `ApplyTaskPrefix` in a leaf package or duplicate the tiny table in effects).
4. **Wire query embed** — `internal/microrag/engine.go` (retrieval) and
   `cmd/ailang/cache_ops.go:63,79` (CLI search) → Query role.
5. **Leave untouched** — `envelope_builder.go`, `search_neural.go` (messaging).

### Acceptance criteria
- [ ] `go test ./internal/messaging/... ./internal/effects/... ./internal/microrag/...` green
- [ ] `make brain-index-syntax-reset` → `ailang cache stats` = 100% embeddings
- [ ] `ailang cache search --namespace ailang-syntax "roman numeral"` returns
      number/math content (not time-and-dates); score recovers toward ~0.6+
- [ ] messaging tests unchanged/green
- [ ] microRAG A/B (opencode-qwen3-5, core, on vs off) re-run on prefixed corpus;
      delta recorded

### Risks
- **Import cycle** (effects ↔ messaging): mitigate by placing pure `ApplyTaskPrefix`
  in a no-dep location, or duplicating the ~15-line table in effects.
- **Missed query site** → vector mismatch: the acceptance retrieval-probe catches it.

## Success metrics
- Retrieval relevance visibly restored (probe).
- First *valid* microRAG A/B number (prefixed corpus).

SPRINT_PLAN_PATH: design_docs/planned/m-embed-task-prefix-sprint-plan.md
SPRINT_JSON_PATH: .ailang/state/sprints/sprint_M-EMBED-PREFIX.json
