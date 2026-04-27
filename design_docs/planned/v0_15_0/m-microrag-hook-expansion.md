# M-MICRORAG-HOOK-EXPANSION: μRAG hook coverage across harnesses + UserPromptSubmit

**Status**: Planned
**Target**: v0.15.0 (M1 + M2), v0.15.x (M3 follow-up)
**Priority**: P3 (Quality-of-life — μRAG is functional today; this expands signal coverage)
**Estimated**: 2 days for M1+M2 (frontend wiring + UserPromptSubmit). M3 is open-ended.

## Context

[ADR-002](../../decisions/ADR-002-pretooluse-microrag-disabled.md) disabled PreToolUse μRAG injection on `.ail` files because embedding-based retrieval over file content was too noisy: cosine averaged anti-pattern signals into the noise floor. The retrieval mechanism itself is sound; it just needs queries shaped for embedding similarity.

Two work streams complete the μRAG-on-agents story:

1. **UserPromptSubmit hook** — replaces the role PreToolUse used to play, but with a query type embeddings actually excel at: a dense, intent-bearing natural-language sentence ("how do I do string concat in AILANG?"). This was identified in ADR-002 as the killer fit.
2. **Frontend coverage** — opencode and Pi don't have μRAG hooks yet. PostToolUse builtin-lint (the targeted, working part of μRAG) should fire on those harnesses too so when AILANG benchmarks run via opencode/pi, they get the same builtin-first-use nudges that Claude Code / Codex / Gemini already get.

## Scope (M1–M3)

### M1: UserPromptSubmit μRAG hook (Claude Code first)

**Trigger**: every `UserPromptSubmit` event in Claude Code (and equivalents in Gemini CLI / Codex CLI / opencode / Pi if they have an analogous event).

**Behaviour**:
- Read the user's prompt from hook stdin (`prompt` field in Claude Code's `UserPromptSubmit` payload).
- Run it as the search query against `ailang-syntax` and `ailang-builtins` corpora (top-1 each).
- Inject the top hit if it clears the relevance floor.
- Same dedup ledger and session budget as today's PreToolUse path — so the same chunk doesn't fire twice in a session.

**Why this works where PreToolUse failed**: the user's prompt is a single dense sentence in the same vocabulary as the corpus chunks. No averaging over noisy file body, no single-character anti-patterns getting diluted.

**Files**:
- New `internal/microrag/userprompt.go` — `UserPromptRequest` + engine method that reuses search/dedup
- New `cmd/ailang/microrag.go` subcommand `ailang micro-rag user-prompt`
- New `.claude/skills/microrag/frontends/{claude-code,gemini,codex,opencode,pi}/microrag_userprompt.sh` (where the harness supports the event)

**Success criteria**:
- Test 1: prompt "how do I concat strings in AILANG?" → injects "String and List Concatenation" chunk
- Test 2: prompt "what's the syntax for pattern matching?" → injects "Pattern Matching" chunk
- Test 3: prompt unrelated to AILANG (e.g. "what's the weather?") → no injection (below relevance floor)

### M2: opencode + Pi μRAG frontends (PostToolUse builtin-lint)

**Goal**: when an agent runs an AILANG eval via opencode or Pi and emits a first-use builtin (e.g. `httpGet`, `jo`/`kv`/`js`), they get the same builtin-first-use nudge that Claude Code / Codex / Gemini agents already get.

**Why this matters for evals**: today the opencode agent_suite member (opencode-sonnet-4-6) and the Pi harness (pi-claude-sonnet-4-6) don't see PostToolUse μRAG nudges. That means the AILANG-vs-other-language gap evaluated through these harnesses isn't measured under the same μRAG conditions as the Claude Code / Codex / Gemini agents. Closing the gap makes harness comparisons fairer.

**Files**:
- `.claude/skills/microrag/frontends/opencode/microrag_lint.sh` — opencode plugin shim that calls `ailang micro-rag lint-builtin`. opencode plugins install via `opencode plugin install <module>`; the plugin is a small Node module that wraps the bash shim.
- `.claude/skills/microrag/frontends/pi/microrag_lint.sh` — Pi extension shim. Pi extensions install via `pi install <source>`; the extension is a small package wrapping the bash shim.
- Both shims read from harness-specific stdin/event format and adapt to `ailang micro-rag lint-builtin`'s flat CLI interface (already harness-agnostic).
- README updates in `.claude/skills/microrag/frontends/{opencode,pi}/README.md` covering install steps.

**Non-goal**: PreToolUse μRAG injection. Per ADR-002 it's disabled across harnesses. The opencode/Pi frontends do **not** include the PreToolUse `microrag_context.sh` shim.

**Success criteria**:
- AILANG eval run via opencode-sonnet-4-6 sees ≥1 builtin-first-use nudge fire during the v0.15.0 baseline.
- Same for pi-claude-sonnet-4-6.
- The cross-harness `harness_suite` numbers (claude vs opencode vs pi for Sonnet 4.6) reflect μRAG-on conditions consistently.

### M3 (open-ended): SessionStart minimal context

**Trigger**: SessionStart hook (where supported).

**Behaviour**: inject one fixed line at session start: `"Active AILANG prompt: vX.Y.Z. Use \`ailang prompt\` for syntax, \`ailang check\` for diagnostics."` Once-per-session only.

This is *partly* redundant with `CLAUDE.md` / `.claude/rules/ailang-syntax.md` already loading at session start. M3 is only worth doing if telemetry shows users benefit from a one-line μRAG-format nudge there. **Do not implement until M1 + M2 ship and we measure utility from telemetry.**

## Out of scope

- **Re-enabling PreToolUse μRAG on `.ail` files.** ADR-002 stands. If a future redesign justifies revisiting it (e.g. parser-error-driven queries), file an ADR amending 002.
- **Changing the brain corpus structure.** Current corpus chunks are fine; they retrieve correctly when given good queries. M1's UserPromptSubmit gets the queries right.
- **Cross-prompt-version migration.** UserPromptSubmit injection is keyed on the active prompt version (whatever `ailang prompt --version-active` reports). Old prompt versions aren't supported.

## Cost / risk

- **Low**: M1 reuses ~all existing engine code; the hook script + CLI subcommand are <100 LOC each. M2 is two ~50-line bash shims plus install docs.
- **Risk**: opencode/Pi plugin formats may change. Both are young projects. Mitigation: keep frontends thin (pure bash → `ailang micro-rag lint-builtin`) so plugin-API churn doesn't ripple.

## Verification (each milestone has its own success criteria; here's the integration test)

After M1 + M2:
```bash
# UserPromptSubmit fires correctly:
$ echo '{"prompt": "how do I concat strings in AILANG?"}' | \
    .claude/skills/microrag/frontends/claude-code/microrag_userprompt.sh
→ injects "String and List Concatenation" chunk

# opencode + pi PostToolUse builtin-lint fires:
$ ailang eval-suite --agent --models opencode-sonnet-4-6,pi-claude-sonnet-4-6 \
    --benchmarks api_call_json --langs ailang
# Inspect telemetry: confirm builtin-first-use nudges in the agent transcript
```
