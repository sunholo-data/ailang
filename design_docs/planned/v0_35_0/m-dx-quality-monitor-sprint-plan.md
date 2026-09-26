# Sprint Plan: M-DX-QUALITY-MONITOR — empty/looping-output detection + bounded excerpts

## Summary
Ship the `quality-monitor.ts` extension (suite member #10, event-only, zero subprocess
calls): Q3 bounded-excerpt rewrite of oversized tool results, Q2 identical-call loop
detection with block-then-inform, Q1 empty-content detection with capped corrective
steering, Q4 opt-in thinking-budget fallback. Pure pi-layer state via `pi.appendEntry`.

**Design doc:** m-dx-quality-monitor.md (ratified 2026-08-30)
**Duration:** 1 session (~350 LOC + tests)
**Risk Level:** Low — read-only observation + bounded steering; no subprocess, no engine, no harness surgery

## Scope line (explicit)

- pi-layer only. Go harness `quality_*` result-field capture and `error_categorizer`
  changes are OUT (design doc Deferred Decisions: data-led after first rig rotation).
  The channel already exists: the pi executor banks every raw event under
  `provider_data.pi_events` (`internal/executor/pi/pi.go` `piProviderData`), so
  empty-content/oversized-result evidence flows into archived run JSON with zero Go
  changes; analysis-side derivation is a follow-up, not this sprint.
- D1 attribution: where provider spans are not reachable from the extension, records
  say "cause unknown" honestly — never auto-blame the model.
- API correction vs design doc: steering uses `pi.sendUserMessage(text,
  {deliverAs: "steer"})` — `ctx.sendUserMessage` only exists on replacement-session
  contexts (pi docs L1262); D2's semantics (cap 1/class/run, silent no-op when
  settled via `ctx.isIdle()`) are implemented as specified. Q4 uses
  `pi.setThinkingLevel("off")` (clamped to model capabilities) instead of raw payload
  patching — provider-agnostic and never retroactive against mid-stream requests.

## Proposed Milestones

### Q3: Bounded-excerpt rewrite (~0.5h) — biggest win, lowest risk
**Estimated:** ~80 LOC
- `tool_result` handler: text content >16KB → head 1.5KB + separator + tail 0.5KB
  (errors live in tails — the 2026-08-26 lesson) + narrowing directive;
  `details.excerpted = {original_bytes, kept_bytes}`; non-text parts preserved.
- Pure function `excerptContent()` table-driven tested (thresholds, multibyte safety,
  details merge, images untouched).

**Acceptance criteria:**
- [ ] 90KB result → ≤2.2KB content + directive + `details.excerpted`; run continues
- [ ] ≤16KB results byte-identical (handler returns undefined)
- [ ] e2e: local-model session with a >16KB bash output shows the excerpt in transcript

### Q2: Loop detector (~0.5h)
**Estimated:** ~70 LOC
- `tool_call` handler: rolling window (8) of `hash(toolName + canonical input JSON)`;
  block at 3rd *consecutive* identical call (D3; window clears on any distinct call);
  reason names the repeat + directive, never a bare refusal. Cap 3 blocks/run then
  notify-only.

**Acceptance criteria:**
- [ ] identical×3 → block with reason; distinct-args calls never blocked
- [ ] `grep A | grep A | grep B | grep A` pattern restart clears (consecutive semantics)
- [ ] 3-block cap per run; after cap, notify-only
- [ ] pure `LoopWindow` unit tests (strikes, window eviction, reset)

### Q1: Empty-content detection + capped steer (~1h)
**Estimated:** ~90 LOC
- `turn_end` handler: zero non-empty text, zero tool calls, stopReason `stop|length`
  (strip thinking blocks) → `appendEntry("quality:empty_content", …)` + one steer
  (skipped when `ctx.isIdle()`; steer failure = logged entry, never silent); counter
  resets on any non-empty turn; per-run counters reset on `agent_start`.
- Session-state reconstruction from entries on `session_start` (resume parity).

**Acceptance criteria:**
- [ ] empty turn → exactly ONE corrective message per run cap; second empty turn → recorded, not steered
- [ ] non-empty turn resets the cap
- [ ] settled session → no steer (entry still recorded)
- [ ] stopReason `error`/`aborted`/`toolUse` never flagged (pi owns retry/toolloop surface)
- [ ] unit tests on `isEmptyAssistantTurn` incl. reasoning-only and whitespace-only arms

### Q4: Thinking-budget fallback, opt-in (~0.5h)
**Estimated:** ~40 LOC
- `PI_QUALITY_THINKING_FALLBACK=1` (default off) + ≥2 empty streams for the same
  model in-session → `pi.setThinkingLevel("off")` once + `notify` + entry.

**Acceptance criteria:**
- [ ] default-off verified; two fires → one fallback; already-off → no-op
- [ ] passive observation: empty_content entries carry the model id (D1 correlation hook)

### Ship: tests + suite integration (~0.5h)
- Table-driven unit suite `.pi/extensions/.quality-monitor.test.ts`
  (dot-prefixed: pi skips discovery; `node --experimental-strip-types --test`).
- `.pi/extensions/README.md` table row (also fixes measured README drift: add
  `ail-fmt-autolint.ts`, landed f57603cd7 but unlisted); counts eight→ten.
- `make pi-assets` + `make verify-pi-assets` (Tier-0 embedded copy).
- Design doc status flip; commit on `dev` with `Co-Authored-By: pi (anthropic/claude-opus-4-6)`.

**Acceptance criteria:**
- [ ] full unit suite green; verify-pi-assets clean
- [ ] kill switch `PI_QUALITY_MONITOR=0` → fully inert

## Out of scope (deferred, per design doc)
- Rig A/B paired run (phase 5, needs the next rig rotation window — this sprint's
  extension ships behind the same kill switches, so rotating is config-only)
- `error_categorizer.go` taxonomy changes (data-led)
- `thinkingLevelMap`-aware per-provider patch precision (setThinkingLevel clamps)
- Near-duplicate (Jaccard) loop detection
