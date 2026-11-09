# Sprint Plan: M-DX-MICRORAG-CONTEXT — μRAG as a first-class pi extension

## Summary
Ship the `microrag-context.ts` extension (suite member #11): M1 prompt-intent
injection via `before_agent_start`, M2 error-triggered injection from a one-entry
`ailang_check` buffer, M3 on-demand `microrag_search` tool. Every engine call runs
under the Subprocess Contract; engine untouched (the +13.1pp evidence transfers
through the existing env-gated A/B switch).

**Design doc:** m-dx-microrag-context.md (approved by Mark 2026-08-31)
**Duration:** 1 session (~380 LOC extension + ~300 LOC tests, then e2e on local model)
**Risk Level:** Low — read-only engine calls, budget-capped injection, kill switches;
no Go changes, no engine changes, no harness surgery

## Scope line (explicit)

- pi-layer only; joins the suite via the standard paths (`.pi/extensions/` +
  Tier-0 `make pi-assets`). `internal/microrag` untouched (D3 fallback subcommand
  is deferred unless the M2 e2e measurably fails the floor — see Out of scope).
- D1 honored absolutely: `before_agent_start` returns `{ message: {…} }` only —
  never a `systemPrompt` field (pi docs L530–560: both are possible; we never
  touch the prompt so the KV-cache prefix stays byte-stable).
- D2 respected: no file-content-triggered injection anywhere; the only trigger
  surfaces are prompt intent (M1), buffered check errors (M2), explicit tool (M3).

## Resolved design points (verified live this session, pre-implementation)

These close ambiguities between the doc's sketches and the real contracts; the
design doc gets an implementation note on landing, same pattern as quality-monitor:

| # | Point | Resolution | Evidence |
|---|-------|------------|----------|
| R1 | Gate default when `AILANG_MICRORAG_ENABLED` is **unset** | Extension inert (no subprocess). Diverges from the engine's `EnabledFromEnv` (unset→true) and the shims' `${ENABLED:-1}` default-on — deliberately: pi is the rig's eval path; Auto-mode arms (`--microrag auto`, unset env) must not silently start injecting vs today's baseline. Explicit `=1` opts in; `=0`/false disables. | Design doc Success Criterion: "unset env → extension inert (no subprocess spawned)" |
| R2 | Kill switch | `PI_MICRORAG_CONTEXT=0` → whole extension inert (no subprocess, no tool registered at load)... tool registration skipped too: one switch, one meaning. | Doc §Config & lanes |
| R3 | Prompt transport to engine | `--prompt @<tmpfile>` (engine has no stdin for user-prompt; flag-text risks arg-length/quoting). Temp file in `os.tmpdir()`, cap 8KB, deleted after the call. `pi.exec` uses argv (no shell), so paths with spaces are safe. | `ailang micro-rag user-prompt --help` |
| R4 | Envelope contract | Hit: `{"injection":{"injection_text",…},"reason":"injected"}` (exit 0); miss: `{"reason":"below_floor"}` (exit 0). Inject iff `reason==="injected"` && `injection_text` non-empty. Provenance header already embedded by engine (`🧠 μRAG` marker). | Live probe this session (concat query → std/string concat hit, score 0.81; weather query → below_floor) |
| R5 | Session key | `AILANG_MICRORAG_SESSION=pi:<ctx.sessionManager.getSessionId()>` — gives the engine's dedup ledger a pi-stable key; ledger persists across `/resume` with the same id, so no re-injection after restart. | pi `session-manager.d.ts` `getSessionId()` (V-read); V9 ledger semantics |
| R6 | Timeout budget | 5000ms default (claude-code UserPromptSubmit hooks run the identical command at 4s; +1s headroom), override `PI_MICRORAG_TIMEOUT_MS`. Timeout → structured no-op (silent skip) + `ctx.ui.notify` once per run — engine *failure* notifies, floor-miss never does (A11). | `.claude/settings.json` UserPromptSubmit timeouts |
| R7 | Pre-gate | Skip pure commands: prompt starting with `/`, or ≤8 words without an AILANG token (`/\b(ailang|\.ail)\b/i`). Pure-function tested. | Doc §M1 |
| R8 | M2 buffer source | `tool_result` where `event.toolName === "ailang_check"` and `details.diagnostics` carries `severity==="error"` entries — the shipped V12 shape returns `details: { ok, diagnostics: [{code, severity, message, …}] }` (bash-run `ailang check` is unstructured and out of contract; noted). Cap: depth 1, clear-on-serve, newest wins. | `ailang-lsp-lite.ts` execute() details |
| R9 | M2 synthetic query (D3) | `AILANG error <CODE>: <message> — how to fix` (dense, code-bearing, ≤512 chars). Fires on the **next** `before_agent_start`; never same-turn (parallel-tool interleave, doc §M2 cross-check). | Doc D3 |
| R10 | M3 corpus map | `{syntax: "ailang-syntax", builtins: "ailang-builtins", docs: "ailang-docs"}` → `--namespaces CSV`; omitted → engine default (syntax+builtins). Missing namespace degrades to `below_floor` → empty hits, no error. | `cmd/ailang/microrag.go` bootstrap namespaces |

## Proposed Milestones

### M1: Prompt-intent injection (~1h) — the +13.1pp evidence path
**Estimated:** ~110 LOC
- `before_agent_start` handler: R7 pre-gate → R4/R3 engine call under the
  Subprocess Contract (R6 timeout) with R5 session key → on hit, return
  `{ message: { customType: "microrag-inject", content: injection_text, display: true } }`.
- System-prompt untouched, always (D1). All misses/errors/undersized returns → `{}`.

**Acceptance criteria:**
- [ ] Known prompt (ADR-002 M1 case: "how do I concat strings in AILANG?") → the
      std/string concat chunk injected, exactly once per session
- [ ] Below-floor prompt ("what's the weather?") → zero injection, zero notify
- [ ] `/`-command and short non-AILANG prompts → zero subprocess spawned
- [ ] `AILANG_MICRORAG_ENABLED` unset/0 or `PI_MICRORAG_CONTEXT=0` → inert, no subprocess
- [ ] Injected content carries engine provenance (`🧠 μRAG` marker) — grep-able in transcripts

### M2: Error-triggered injection (~1h) — the lane no frontend has anywhere
**Estimated:** ~90 LOC
- `tool_result` handler (R8): buffer `{codes, firstMessage, at}` on error diagnostics;
  persist `pi.appendEntry("microrag:last_error", …)`.
- `before_agent_start`: unserved buffer → R9 synthetic query through the same
  M1 injection path; clear-on-serve; newest wins. Session-state reconstruction
  from entries on `session_start` (/resume parity — session-gate pattern).

**Acceptance criteria:**
- [ ] `ailang_check` → IMP010 in turn N → relevant guidance injected turn N+1; buffer cleared
- [ ] Same code re-checked next turn → no re-injection (engine dedup ledger, V9)
- [ ] Warning-only diagnostics → no buffer, no injection
- [ ] /resume reconstruction: buffered-and-served state survives restart without re-serving

### M3: On-demand `microrag_search` tool (~45min)
**Estimated:** ~70 LOC
- `pi.registerTool` (session-protocol-gate pattern): `{query, corpus?}`, executes the
  engine's user-prompt route (R10), returns `{hits: [{tier, score, content, source}]}` —
  cap 3 hits, ≤1KB excerpt/hit, `details` = full envelope. Description carries one
  paraphrase example so small models use it before guessing APIs.

**Acceptance criteria:**
- [ ] Known query → envelope hits with content; below-floor query → `{hits: []}`, no error
- [ ] Envelope JSON round-trips into `details` for audit (A12)

### Ship: tests + suite integration + docs (~1h)
- Unit suite `.pi/extensions/.microrag-context.test.ts` (dot-prefixed, pi skips
  discovery): pure functions — pre-gate, envelope parser, synthetic-query builder,
  diagnostics extractor, gates matrix. Contract test: stdin/flag fixture parity
  against the claude-code shim (`microrag_userprompt.sh`) per design doc Testing §2.
- e2e on local model (headless `pi -p`, ollama local arm): M1 hit + below-floor +
  M2 IMP010 flow; transcripts captured under `.ailang/state/` (not committed).
- `.pi/extensions/README.md` table row (nine → ten members + this = eleven... recount at PR time);
  `make pi-assets` + `make verify-pi-assets` (Tier-0 embedded copy serves the new file).
- Cache-discipline spot check: system prompt byte-identical across turns — asserted
  structurally (extension returns no `systemPrompt` field; the only mutation vector)
  + transcript spot check.
- Design doc status flip → Implemented + implementation note (R1–R10 deltas);
  commit on `dev` (design doc + plan + extension together), `Co-Authored-By: pi`.

**Acceptance criteria:**
- [ ] Full unit suite green; `verify-pi-assets` clean; README table updated
- [ ] Kill switch + unset-env both fully inert (e2e arms)
- [ ] `--mode json` banked events show injection entries in the harness capture channel

## Out of scope (deferred, per design doc)
- **Rig A/B paired run** (doc phase 4): needs the next rig rotation window; the
  extension ships behind R1/R2 switches so rotating is config-only (`--microrag on/off`
  already wired via V10). Paired report archived under `eval_results/` at rotation.
- D3 fallback Go subcommand (`ailang micro-rag error --code`) — only if the M2 e2e
  shows the synthetic query missing the floor on real error messages; split follow-up.
- opencode frontend parity (the doc's own listed separate scope).
- File-content triggers (D2/ADR-002), MCP bridge, engine/scoring changes, teaching-prompt changes.

## Risks & Mitigations
| Risk | Mitigation |
|---|---|
| Engine cold-start latency blocks turn start (before_agent_start awaits) | 5s timeout → structured no-op; pre-gate skips most non-knowledge prompts; caches (240s/1d) make warm path ms-scale |
| Silent baseline drift on the rig (auto arms start injecting) | R1: unset-env → inert; only explicit `=1` arms inject — measured, not assumed |
| Double-injection with claude-code-style hooks if both active in one harness | Different harnesses read different hook configs; pi has only this extension — no hook file exists for pi (V3) |
| Buffer served stale after long idletime | Buffer carries turn context; clear-on-serve + clear on newer error; /resume reconstruction marks served buffers consumed |