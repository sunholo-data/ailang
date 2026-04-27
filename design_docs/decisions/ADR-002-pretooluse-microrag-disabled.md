# ADR-002: PreToolUse μRAG injection on `.ail` files is disabled

**Status**: Accepted
**Date**: 2026-04-27
**Related**: [m-brain-microrag.md](../planned/v0_15_0/m-brain-microrag.md), [m-eval-gap-fixes.md](../planned/v0_15_0/m-eval-gap-fixes.md)

## Context

μRAG was wired into Claude Code / Gemini CLI / opencode as a `PreToolUse` hook on `Edit | Write | Read | MultiEdit`. The hook calls `ailang micro-rag context`, which:

1. Routes the file path to a knowledge-base (`ailang-syntax` for `**/*.ail`).
2. Builds a query from filepath + first ~2KB of file content.
3. Runs an embedding-based search against the brain corpus.
4. Injects the top-1 chunk if it clears the route's relevance floor.

In practice — across the v0.14.2 baseline (1,178 runs) and direct user-side testing — this proved to be **noise more often than signal**:

| File pattern | What was retrieved | What was needed |
|---|---|---|
| `module mcp-tools/lib` (hyphen) | "Standard Library" chunk | "Multi-Module Projects" chunk |
| `"a" ++ "b"` (string operator misuse) | "Effects" chunk | "String and List Concatenation" chunk |
| Clean code with no issue | Random ("Testing", etc.) | (no injection at all would be fine) |

The root cause is fundamental, not a config tuning issue: **embedding cosine similarity averages over all tokens in the query**. A 1-character anti-pattern (single `-`, single `++`) contributes ~1% of the vector, while generic terms like `import`, `! {IO}`, `module` contribute ~5–10% each. Generic-topic chunks always win.

## Alternatives considered

1. **Hardcoded anti-pattern registry** (regex → hint phrases prepended to the query). Tried in commit `83e08a74`, reverted in `5432c65e`. Rejected as "training to the test" — every new quirk needs a code change, and the prompt is already the right home for canonical anti-patterns.
2. **Structural truncation** (top 12 lines / 512 chars instead of 2KB). Tighter but doesn't solve the averaging problem — generic terms still dominate the cosine.
3. **Parser-driven retrieval** (run `ailang check`, use error msg as query). Genuinely more focused, but redundant: the agent runs `ailang check` themselves when they're ready, and the parser's error messages are already a precise diagnostic — bouncing them through embedding similarity adds latency without clearer signal.
4. **Disable PreToolUse, keep PostToolUse builtin-lint.** PostToolUse `micro-rag lint-builtin` (a different code path) targets a specific question — "is this a first-use of a builtin we should remind the model about?" — and produces focused nudges. That stays on.

## Decision

**Disable PreToolUse μRAG injection on `**/*.ail` files** by setting the route's `kb` to `skip` in `~/.ailang/microrag.yaml`. The hook still runs (telemetry stays intact, dryrun mode keeps working) but never injects. PostToolUse builtin-lint stays on.

The single source of truth for AILANG syntax is **`ailang prompt`** — same one that's loaded into the teaching prompt at session start.

## Consequences

- Sessions stop accumulating ~150-token injections on every `.ail` file edit.
- The agent gets cleaner context; prompt-driven syntax knowledge is uncontested by retrieval noise.
- If a future use case for the PreToolUse hook emerges (e.g. once-per-session "active version" reminder), the route can be re-enabled — the engine still runs.
- This decision is **scoped to `.ail` files**. The `**/CHANGELOG.md` route to `ailang-breaking-changes` remains active because version-bump-aware injection on changelog edits is well-targeted.

## Future μRAG hook candidates

The retrieval mechanism itself is sound; it just needs queries shaped for embedding similarity, not lossy code-content averages. Better-fit hooks for future exploration:

- **`UserPromptSubmit`** — when the user asks "how do I do X in AILANG?", the prompt text *is* a natural-language embedding query. This is the killer fit: dense, intent-bearing, single-question vector. Could replace the PreToolUse hook entirely.
- **`SessionStart`** — once-per-session injection of active prompt version + canonical reference. Currently handled by CLAUDE.md / `.claude/rules/ailang-syntax.md`, but a tiny μRAG-loaded snippet could supplement.
- **`Stop`** — too late in the loop to be actionable; the assistant has already finished.

If we revisit this, prefer `UserPromptSubmit` over re-enabling PreToolUse on file content.
