# Sprint Plan: M-MICRORAG-EXPAND (μRAG hook coverage expansion)

**Status**: Ready for execution
**Target**: v0.15.0
**Estimated**: 1 day (~6h focused work)
**Priority**: P3 (quality-of-life; closes harness-fairness gap before next baseline)
**Risk**: Low

## Source-of-Truth Design Doc

[m-microrag-hook-expansion.md](m-microrag-hook-expansion.md) — full architectural rationale (post-ADR-002).

## Why This Sprint (and Why Now)

Two motivations stack:

1. **UserPromptSubmit is the right home** for embedding-based retrieval. The user's NL prompt is exactly the dense intent-bearing query embeddings excel at. ADR-002 closed the wrong-fit PreToolUse path; this sprint opens the right-fit one.
2. **Harness-fairness gap before v0.15.0 baseline**: opencode-sonnet-4-6 and pi-claude-sonnet-4-6 currently run AILANG benchmarks without μRAG PostToolUse builtin-lint nudges, while claude/codex/gemini agents get them. The next baseline run will produce cleaner cross-harness numbers if this is closed first.

## Velocity Calibration

Recent commits (last 7 days, dev branch):
- M-EXEC-PI sprint: 6 milestones in ~5 days (PostToolUse Dockerfile + cloudrun in 1d, executor wiring 1d, tests 0.5d, etc.)
- M-AGENT-MCP M1: tool skeleton in <1 day
- v0.14.2 post-release: dashboard work + ADR + design docs in 1 day

**Estimated velocity for this sprint**: 1 day end-to-end is realistic. M1 is ~150 LOC of new Go + 1 bash shim; M2 is two ~50-line bash shims + READMEs. Test coverage adds ~50 LOC.

## Milestones

### M1 — UserPromptSubmit μRAG hook (4h)

**Goal**: when the user submits a prompt mentioning AILANG, retrieve the most-relevant `ailang-syntax` chunk and inject it as additional context.

**Files**:
- `internal/microrag/userprompt.go` (new, ~80 LOC) — `UserPromptRequest` struct, `(*Engine).UserPrompt(req)` method that reuses search/dedup logic from existing `Context` method
- `internal/microrag/userprompt_test.go` (new, ~80 LOC) — unit tests covering: (a) prompt with AILANG syntax question retrieves matching chunk, (b) off-topic prompt below relevance floor returns no injection, (c) dedup prevents same chunk firing twice, (d) session budget honored
- `cmd/ailang/microrag.go` (extend, ~30 LOC) — add `runMicroragUserPrompt` subcommand parsing `--prompt` (or `--prompt @path`), wiring to engine
- `.claude/skills/microrag/frontends/claude-code/microrag_userprompt.sh` (new, ~40 lines bash) — reads UserPromptSubmit hook stdin, extracts `prompt`, shells out to `ailang micro-rag user-prompt --prompt "$PROMPT"`, emits `additionalContext` envelope
- `.claude/settings.json` (extend) — register the new hook alongside the existing `brain_on_prompt.sh` UserPromptSubmit hook

**Acceptance criteria**:
- `ailang micro-rag user-prompt --prompt "how do I concat strings in AILANG?"` injects "String and List Concatenation" chunk
- Same command with `--prompt "what's the weather?"` returns `state: "on", reason: "below_floor"` (no injection)
- Re-running the same prompt within the dedup window returns `reason: "dedup_suppressed"`
- All new tests pass: `go test ./internal/microrag/...`
- `make quick-install` succeeds; `ailang micro-rag user-prompt --help` lists the new flags

**Risk**: low — reuses existing Engine + Searcher + dedup. Only new path is the entry point.

### M2 — opencode + Pi PostToolUse builtin-lint frontends (2h)

**Goal**: when AILANG benchmarks run via opencode-sonnet-4-6 or pi-claude-sonnet-4-6, agents see the same builtin-first-use μRAG nudges that claude/codex/gemini agents already get.

**Files**:
- `.claude/skills/microrag/frontends/opencode/microrag_lint.sh` (new) — bash shim mirroring the gemini/codex `microrag_lint.sh` shape; reads opencode's PostToolUse event (we'll need to inspect opencode plugin API to confirm exact stdin schema), extracts file path + new content, calls `ailang micro-rag lint-builtin --file ... --code @...`
- `.claude/skills/microrag/frontends/opencode/README.md` (new) — install instructions: how to register the shim as an opencode plugin via `opencode plugin install <local path>` or wrap as a small npm module
- `.claude/skills/microrag/frontends/pi/microrag_lint.sh` (new) — same shape adapted to Pi's extension event format
- `.claude/skills/microrag/frontends/pi/README.md` (new) — install instructions: `pi install <source>`

**Acceptance criteria**:
- Each shim, run with simulated stdin matching the host harness's event format, produces the same `additionalContext` envelope as the gemini/codex shims for the same input.
- READMEs document the install command and any known caveats.
- The existing `ailang micro-rag lint-builtin` Go path is **unchanged** (zero risk of regression for already-wired harnesses).

**Risk**: low — pure bash shims. The unknown is each harness's exact PostToolUse event schema; if either harness's plugin API doesn't support a passive observer hook, we document the gap and skip that harness for now without blocking M1 ship.

### M3 — DEFERRED (open-ended, telemetry-gated)

SessionStart minimal context — described in the design doc but not in scope for this sprint. Re-evaluate after M1+M2 ship and we have telemetry on UserPromptSubmit usefulness.

## Day-by-day Plan

### Day 1 (single-day sprint)

| Slot | Work | Output |
|---|---|---|
| 0–1.5h | M1 engine layer: `userprompt.go` + tests | Code + 5 passing tests |
| 1.5–2h | M1 CLI subcommand + smoke test against real corpus | `ailang micro-rag user-prompt` works end-to-end |
| 2–2.5h | M1 Claude Code frontend shim + settings.json wiring | Hook fires on UserPromptSubmit |
| 2.5–3h | M1 verify: trigger via real prompt in this very session | Live injection observed |
| 3–4h | M2 opencode shim + README (inspect plugin API first; if blocked, document gap) | shim file + install instructions |
| 4–4.5h | M2 Pi shim + README (similar) | shim + instructions |
| 4.5–5h | M2 integration test: run benchmark via opencode-sonnet, observe μRAG fires | Telemetry confirms |
| 5–5.5h | Update changelog, commit, push | `[v0.15.0] M-MICRORAG-EXPAND` entry |
| 5.5–6h | Send completion handoff to sprint-evaluator | sprint-executor → sprint-evaluator pipeline |

## Acceptance Criteria (sprint level)

1. ✅ Live UserPromptSubmit μRAG injection visible in a real Claude Code session.
2. ✅ All new tests pass; `go test ./internal/microrag/...` green.
3. ✅ opencode-sonnet-4-6 and pi-claude-sonnet-4-6 frontends documented; install steps verified locally OR a tracked gap if the harness API blocks it.
4. ✅ ADR-002 unchanged — PreToolUse μRAG on `.ail` files stays disabled.
5. ✅ Changelog `[v0.15.0]` entry mentions both milestones.

## Dependencies

- `internal/microrag/engine.go` (already exists) — reused for search/dedup/ledger.
- `ailang cache search` (existing) — corpus retrieval backend.
- `.claude/settings.json` — extend the existing UserPromptSubmit hook list (don't replace `brain_on_prompt.sh`).

## Out of scope

- Re-enabling PreToolUse μRAG. ADR-002 stands.
- Brain corpus chunk authoring/restructuring. Existing chunks are sufficient.
- Cross-prompt-version migration. UserPromptSubmit retrieval keys on the active prompt version.
- Codex/Gemini frontend changes for UserPromptSubmit beyond what M1 needs to compile cleanly. (They can be added in a follow-up if telemetry shows value.)

## Risks & Mitigations

| Risk | Likelihood | Mitigation |
|---|---|---|
| opencode plugin API doesn't expose a passive PostToolUse hook | Medium | Document the gap, ship M1 + Pi-only M2; revisit when opencode adds support |
| Pi extension format isn't bash-friendly | Low-Medium | Same: document gap, ship M1; pi extension is well-documented |
| New UserPromptSubmit hook conflicts with existing `brain_on_prompt.sh` | Low | Both register as separate entries in `hooks.UserPromptSubmit[]`; Claude Code runs both, joining their `additionalContext` |
| User prompt embedding misses obvious queries | Medium | Manual smoke test of 5 representative AILANG questions before commit; relevance floor tunable |
