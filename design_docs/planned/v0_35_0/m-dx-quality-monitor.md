# M-DX-QUALITY-MONITOR — empty/looping-output detection with corrective steering + bounded tool-result excerpts

**Status**: Implemented (pi-layer Q1–Q4 landed 2026-08-30; rig A/B pending next rotation window)
**Target**: v0.35.0
**Priority**: P1
**Estimated**: ~1 session (Q1+Q2 ~0.5d, Q3+Q4 ~0.5d, telemetry e2e ~0.25d)
**Sprint plan**: [m-dx-quality-monitor-sprint-plan.md](m-dx-quality-monitor-sprint-plan.md) — executed same day
**Implementation note (2026-08-30)**: shipped as `.pi/extensions/quality-monitor.ts` (suite member #9, event-only, zero subprocess calls). API deltas vs this doc, resolved by pi docs + live e2e: steering = `pi.sendUserMessage(text, {deliverAs: "steer"})` (`ctx.sendUserMessage` exists only on replacement-session contexts); Q4 = `pi.setThinkingLevel("off")` (clamped to model capabilities) instead of payload patching. Live e2e on `ollama/gemma4:e4b` (headless `--mode json`): 3rd identical call blocked with directive; 90KB tool result delivered as 2,211-byte head+tail excerpt with `details.excerpted` provenance; an *organic* reasoning-only empty turn was detected and steered once, after which the model answered with content; kill switch `PI_QUALITY_MONITOR=0` fully inert; `--mode json` banked `pi_events` + session `quality:*` entries are the harness channel (no Go change needed for capture). Harness `quality_*` result fields + `error_categorizer` taxonomy remain deferred per below. 22 unit tests green: `node --experimental-strip-types --test .pi/extensions/.quality-monitor.test.ts`
**Dependencies**: M-DX-PI-HARNESS (shipped — suite, Subprocess Contract, distribution) · observatory span access (existing `curl` instrument, CLAUDE.md)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Developer-experience tooling (pi extension), no language surface change.

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No language/runtime impact |
| A2: Replayability | 0 | No trace impact (it *consumes* traces, adds none) |
| A3: Effect Legibility | 0 | No effect system change |
| A4: Explicit Authority | +1 | Steering injections name the quality rule violated and are capped (max 1 corrective message per failure class per run); blocks on loop are explicit, with the detected repeat shown |
| A5: Bounded Verification | +1 | Detection is a bounded window (last K tool calls, ring buffer); corrective retries are capped at N per run; excerpt rewrite has explicit byte cap |
| A6: Safe Concurrency | 0 | No concurrency surface (parallel-tool contract respected: detection uses per-call events, never sibling results) |
| A7: Machines First | +1 | Converts provider-level silent failure into structured, categorized telemetry the harness can act on (`error_category` quality, not `api_error` guesswork) |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | +1 | Every corrective retry is an at-need LLM call that is logged and budget-capped — cheaper than today's silent full-run re-queues |
| A10: Composability | +1 | Wraps pi lifecycle events only; composes with session gate (read-only detection + notify), eval harness (new `error_category` value), compaction extensions |
| A11: Structured Failure | +1 | The whole point: empty/looping output becomes a structured, actionable event instead of a null-content mystery |
| A12: System Boundary | +1 | Provider stream (reasoning-only) vs. harness (size ceiling) vs. model (loop) are distinguished surfaces — the extension attributes failures to the right one |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 / A3 / A4 / A7 — no violations (A4/A7/A9/A11 strengthened)

## Verification Log

| # | Claim | Method | Result |
|---|-------|--------|--------|
| V1 | Zero-content stream failures are real and recurring here: five pi:deepseek failures rendered as "zero bytes" were **one mechanism** — the model streamed only reasoning tokens and never emitted content — and two prompt-level fixes were aimed at a model problem that did not exist; the harness's size ceiling killed them | CLAUDE.md measured incident 2026-08-26 (settled via OpenRouter observatory Broadcast spans, the provider-side instrument) | Confirmed — repo-documented measured incident |
| V2 | `error_category` = `api_error` is the documented catch-all meaning "cause unknown, not model failed" — empty-content runs today land in a category that explicitly does *not* say what happened | CLAUDE.md instrument table; `internal/eval_harness/error_categorizer.go` (exists; api_error catch-all with mis-recording history noted in its own comments) | Confirmed |
| V3 | pi fires `turn_end` with `{turnIndex, message, toolResults}` — the assistant text and tool results of each completed turn are inspectable | pi 0.84.3 `docs/extensions.md` §turn_end (L601–612) | Confirmed from docs |
| V4 | pi `tool_call` handlers can **block** a call with `{block: true, reason}` and see mutating annotations from earlier handlers; a rolling window of last tool inputs is buildable from `ctx.sessionManager`/event stream | pi docs §tool_call (L778–835) | Confirmed from docs |
| V5 | pi `tool_result` handlers can **modify** the result (`content`, `isError`, …) and are `ctx.signal`-aware | pi docs §tool_result (L842–880) | Confirmed from docs |
| V6 | An extension can send a follow-up user message mid-session: `ctx.sendUserMessage()` (sources include `"extension"`), and `pi.sendMessage()` injects messages that participate in LLM context | pi docs L926, L1158, L1224, L1416–1424 | Confirmed from docs |
| V7 | pi has native auto-retry on provider errors, and `agent_end` vs `agent_settled` distinguish "may continue/retry" from "done" | pi docs §agent_start/end/settled (L567–574) | Confirmed from docs |
| V8 | Provider payloads can be inspected/patched pre-flight: `before_provider_request` can replace the payload (thinking config lives on it); model-level `thinkingLevelMap` (off…max) exists for models that support mapping | pi docs §before_provider_request (L705–731); models.md L205, L261 | Confirmed from docs |
| V9 | **No existing extension in this repo monitors output quality, detects tool-call loops, or rewrites oversized results** | `grep -il "compact\|quality\|hallucin" .pi/extensions/*.ts` → empty; suite inventory (README table: gate, freshness, sprints, dirty-warn, builtin-finish, quota, lsp-lite, prepush) | Confirmed (negative-existence) |
| V10 | Bounded-excerpt-with-paging beats blunt hard ceilings on long streams: the concrete repo incident was a size ceiling measuring pi's quadratic event replay killing valid runs; external reference (little-coder `ShellStart`/`ShellLog`) ships excerpt+page as the designed alternative | CLAUDE.md 2026-08-26 incident; little-coder README §Background jobs (studied 2026-08-30) | Confirmed — both precedents documented |
| V11 | The eval harness preserves per-run JSON fields (pattern exists: `agent_turns`, `finish_reason`) so new quality columns land in result JSON without harness surgery | `internal/eval_harness/` result-capture pattern (Cited by design precedent in m-ailang-semantic-context §Observability) | Confirmed by pattern |

## Problem Statement

The harness today has **no visibility into *quality* failure modes** — output that arrives but is empty (or reasoning-only), tool calls that repeat forever, or results that are verbatim full-stream logs. All three were first-hand-measured in this repo in one month:

1. **Empty/zero-content streams** (measured ×5, 2026-08-26): a model streams only reasoning tokens; pi sees null content; every downstream analyzer reads `api_error` ("cause unknown"). Two prompt fixes and a blunt size ceiling were aimed at the wrong layer before provider-side observatory spans settled it. The *detection* that finally found it — "did the model emit any content at all, per stream?" — is not captured anywhere in our own telemetry.
2. **Tool-call repetition** on local small models: rig failures show turn inflation (15–50 turns vs pi's 5) (`m-ailang-semantic-context`), part of which is re-reading/re-running the same call. Nothing intervenes at call K of an identical call K−1.
3. **Unbounded tool output**: our 64KB Subprocess Contract cap turns oversized output into an abrupt failure — the same class the provider-incident called out. A "bounded excerpt + the exit code + a paging directive" shape preserves the run *and* the context budget.
4. **Reasoning-token burn**: small local models (and providers like the deepseek incident) can spend the entire budget in reasoning content on trivial turns, or emit reasoning tokens alone with no answer. pi offers per-request thinking controls but nothing caps or steps them down *reactively*.

**Reference-note:** little-coder ships exactly these three as extensions (`quality-monitor`, `thinking-budget`, bounded excerpts in `ShellStart`) and credits them among its top levers for 9.7B/35B local models on Terminal-Bench (TB 2.0 scaffold). External precedent confirms the failure class is generic, not a repo quirk.

## Goals

**Primary goal:** make empty-content, looping, and oversized tool output **structured, attributed, and actionable** at the pi layer — detection and bounded correction land; reshaping eval `error_category` follows from the same telemetry.

**Success metrics:**
1. A reasoning-only empty stream → detected at `turn_end`, one corrective re-prompt steered (`sendUserMessage`), capped at 1; if steering is unavailable in that mode (headless eval), the run fails into a NEW structured category `empty_content` (not `api_error`), visible in result JSON.
2. Repeated identical tool call (same tool + same input hash ×3 in window 8) → 4th call blocked with a reason naming the repeat; measured as blocked-count per run.
3. Oversized tool results → excerpt rewrite ≤2KB with excerpt-length summary preserved in `details`.
4. Rig A/B: benchmarks where loops/empty streams occurred at ≥1 trial show no pass-rate regression and either fewer wasted turns (loops) or same-pass-with-fewer `api_error` unknowns (empty streams).

## High-Impact Decisions

| # | Decision | Rationale |
|---|----------|-----------|
| D1 | **Attribute, don't auto-blame the model**: every detection writes a structured record (`platform_raw` span id or span lookup hint when available) before any steer/block | The 2026-08-26 incident's root lesson: our own logs misattributed a provider/harness issue to the model. Detection without attribution would add a *second* wrong layer. Where spans are reachable (spinner: dashboards observatory curl — provider host keyed by time), the record goes in `details`; absent spans, log "cause unknown" honestly. |
| D2 | **Steer via `ctx.sendUserMessage`, cap 1 corrective message per failure class per run** | Documented extension-to-stream API (V6); a cap prevents correction-storms (little-coder ships a similar correction-follow-up cap). Silent no-op if the session is settled. |
| D3 | **Loop detection: content-hash window, block-then-inform** | Rolling window of input hashes over the last 8 tool calls, both observable in pi's tool events (V4). Block only at the 3rd identical consecutive call, with an explicit reason (runaway-loop risk otherwise). Blocking must never fire on *distinct* args — hash includes the full input. |
| D4 | **Excerpt rewrite, not rejection, for oversized results** | V10: our ceiling incident cost 5 runs × full re-queues. A `tool_result` patch (V5) keeps the run alive with a 2KB head+tail excerpt + `[N chars elided]` + a "page deeper on demand" directive in the content. |
| D5 | **Thinking budget is a fallback lane, not a primary** | `thinkingLevelMap`/payload rewrite (V8) lets us drop to `off` on retry *only* for providers advertising the key (deepseek-class). It is per-provider opt-in config, since off-thinking changes answer quality on some models — never a global switch. |
| D6 | **All counters live in session state via `pi.appendEntry`, reconstructed on `/resume`** | Matches the session-gate State Management pattern; survives resume the way ack state does. |

## Solution Design

One extension, `quality-monitor.ts`, in the existing suite. Note: pure observation + bounded steering; no subprocess contract needed (no CLI calls).

### Q1 — Empty-content detection + corrective steer

```
turn_end ──► did the assistant message have non-empty text content?
             (strip reasoning blocks; empty && toolResults empty = candidate)
   ├─ empty → appendEntry("quality:empty_content", {turn, model})
   │          if steerBudget remaining → ctx.sendUserMessage(
   │             "Your previous reply contained no content (possibly reasoning-only
   │              output). Answer in plain text now; keep tool use out of the reply.")
   │          notify (TUI) + structured record for eval capture
   └─ non-empty → reset per-run empty counter
```
Headless parity: in `-p`/RPC the steer still works via the message stream (source "extension"); if the run then ends empty again, the *result JSON* carries `quality_empty_content=true` so the eval harness can categorize it.

### Q2 — Tool-call loop detection

Rolling window (size 8) of `hash(toolName + canonicalInput)`:
```
tool_call ──► hash in window ≥ 3 consecutive? → return {block: true,
              reason: "identical call repeated 3× at turns t-2..t; do X differently —
              see last result before retrying"}
              else → append to window
```
Content-hash (canonicalized JSON) not event-order: guards concatenation like `grep A | grep A | grep B | grep A` (pattern restart, window clears on any distinct call). Cap per run: 3 blocks, then notify-only (a *reason* to loop may be legitimate — long-polling by design — so blocks are informative, never deterministically fatal).

### Q3 — Bounded-excerpt rewrite

```
tool_result (can modify) ──► size(content) > 16KB?
   ├─ yes → content = head(1.5KB) + "\n…\n" + tail(0.5KB of stderr/summary)
   │        + "\n[result truncated: N chars; re-run with narrower flags/grep to fetch more]"
   │        details.excerpted = {original_bytes, kept_bytes}
   └─ no  → passthrough
```
16KB is well above every legitimate inline result we've seen in rig transcripts and well below the 64KB fail-cap; the excerpt *directs the narrowing* instead of ending the run. stderr side-channels keep their own excerpt tail (errors live at the end of logs — that's the 2026-08-26 incident's actual lesson).

### Q4 — Reasoning-budget fallback

- Passive pass first: in `before_provider_request` observe, never patch — after Q1 flags an empty-content stream for a model, log the payload's thinking config (observatory correlation, D1).
- Active fallback only when Q1 has fired ≥ 2 times for the same model in a session: patch `thinking` → `off` (or lowest mapped level via the model's `thinkingLevelMap`) on the *next* request, notify once. Never retroactive against mid-stream requests. Config off by default (`PI_QUALITY_THINKING_FALLBACK=1` to enable) until an A/B justifies the default everywhere.

### Eval-harness surface

Result JSON gains optional `quality_*` fields (Q1/Q2 signals) — capture-only; renaming error categories is a separate, data-led decision made after the first rig rotations surface category counts (avoid litigating `error_categorizer.go` semantics in the same sprint that introduces the signals).

## Examples

**Q2 in action (rig session):**

```
[t12] read(csv_inserter.ail)            ✓
[t14] read(csv_inserter.ail)            ✓  (window: 2×)
[t16] read(csv_inserter.ail)  → BLOCKED
      "read call repeated 3× with identical args; the file has not changed since
       t12 — re-reads cannot resolve your type error. See the TC010 diagnostic
       you already have, or grep the file for 'Result'."
[t17] model edits the file instead  (2 turns saved; transcript shows the block)
```

**Q3 vs. today:** a 90KB test-log `bash` result is today killed by the 64KB cap (`TIMEOUT`-style structured failure, run dies); with Q3 it arrives as a 2KB head+tail excerpt ending in "…re-run with narrower flags", and the model proceeds.

## Success Criteria

- [ ] Q1: scripted empty-content stream (mock provider response) → corrective message sent once, no second; headless JSON carries `quality_empty_content`
- [ ] Q2: identical-call loop → block at 3rd with reason; distinct-arg calls never blocked; block cap at 3
- [ ] Q3: >16KB result → ≤2.2KB content, `details.excerpted` set, narrowing directive present, run continues
- [ ] Q4: with fallback enabled and 2 empty streams → next request payload shows thinking disabled; disabled-by-default verified
- [ ] Zero subprocess calls in the extension (pure event logic); no session-gate interplay
- [ ] `/resume` reconstruction: state entries reload; counters survive fork/clone branches correctly (per-branch isolation)
- [ ] Rig A/B on the known loop-prone benchmark set: no pass-rate regression; ≥1 previously-`api_error` run either completes or re-categorizes to `empty_content`
- [ ] README table + Tier-0 embedded copy updated

## Testing Strategy

1. **Unit:** pure detectors (hash window, empty detector, excerpter) with fixture transcripts.
2. **Extension e2e:** headless `pi -p` with scripted provider events per the session-gate test pattern; assert steer/block/excerpt observables in transcript JSON.
3. **A/B:** paired rig run before/after; metrics = wasted turns, `api_error` count, pass rate.
4. **Observatory correlation (optional, cheap):** one rig session with a deliberately-empty stream; verify the event correlates with the provider span (`rawRequest` reasoning config) — closes the D1 attribution loop.

## Deferred Decisions

- **Default-on thinking fallback** (D5): needs its own paired A/B (deepseek-class model, empty-content-prone benchmark) before being default anywhere.
- **`error_category` rename/addition** in `internal/eval_harness/error_categorizer.go`: after a rig rotation with `quality_*` fields banked, pick the catch-all taxonomy from data.
- **Cross-session loop ledger** (remembering looped benchmarks across sessions): evaluationbanking domain (`ailang eval-*`), not pi-layer state; only if per-run capping proves insufficient.
- **pi auto-retry interplay**: if pi's native retry already resolves most empty streams upstream (unknown as of pi 0.84.3 — verify at implement time via `agent_end` vs `agent_settled` semantics, V7), Q1's steer must de-dup against it rather than stack a second correction.

## Non-Goals

- No changes to pi's provider stack, size limit, or retry implementation (extension-side observation and bounded steering only).
- No "quality scoring" of substantive output (no judge-LLM calls, no cost-adds — A9).
- No behavioral difference between cloud and local models at first cut (D4's thinking fallback is the only lane-sensitive piece and is opt-in).
- Never bypasses the session gate: steering messages are extensions' own documented channel; no bash, no writes.

## Timeline

| Phase | Work | Est. |
|---|---|---|
| 1 | Q3 bounded excerpt + tool_result e2e (biggest single win, lowest risk) | 0.25d |
| 2 | Q2 loop detector + block/inform e2e | 0.25d |
| 3 | Q1 empty detector + steer + harness JSON capture | 0.5d |
| 4 | Q4 thinking fallback (opt-in) + unit tests | 0.25d |
| 5 | Rig paired A/B + observatory correlation note | 0.5d |

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Correction storms on pathological models (steer→empty→steer) | Hard cap: 1 steer per failure class per run (D2); counter resets only on a non-empty turn |
| Loop-blocker fires on legitimate repetitions (poll watches, retry-backoff commands) | Content-hash window only catches *identical* args; 3-block cap then notify-only; blocks always carry an alternative directive, never a bare refusal |
| Excerpt rewrite hides the portion the model needed | Head+**tail** shape (errors live in log tails — the 2026-08-26 incident's actual mechanism); `details.excerpted` records original size so truncation is never invisible |
| µRAG-context extension (m-dx-microrag-context) also fires on `tool_result` → ordering interplay | That extension returns `{}` when no error-codes path triggers; detectors are independent, and M2's cross-turn buffer was designed order-safe (parallel-tool contract) — assert the composed ordering in one e2e |
| Q1 mis-attributes a provider outage to a model | D1: observatory correlation optional but attribution stays "cause unknown" in every record that lacks span evidence |
| Extra turns cost tokens on the rig | Caps (1 steer, 3 blocks) bound the cost; A/B measures net turn delta — negative delta is the goal |

## Related Documents

- [M-DX-PI-HARNESS](m-dx-pi-harness.md) — suite + Subprocess Contract this joins (event-only here: no subprocess)
- [m-dx-microrag-context](m-dx-microrag-context.md) — sibling extension; interplay covered in Risks
- [M-EVAL-SEMANTIC-CONTEXT](../v0_29_0/m-ailang-semantic-context.md) — sibling context-hygiene workstream (its Branch B: tool-result truncation is generalized by Q3; cite coordination point)
- CLAUDE.md 2026-08-26 incident log — provider-trace root cause; the two prompt fixes aimed at the wrong layer
- External: little-coder `quality-monitor` / `thinking-budget` / `ShellStart` excerpt design — precedent measurements (TB2.0 24.6% run), not a code dependency

## References

- pi 0.84.3 `docs/extensions.md`: §turn_end (L601), §tool_call (L778), §tool_result (L842), §before_provider_request (L705), §agent_settled (L567), `ctx.sendUserMessage` (L926/L1416), `pi.appendEntry` (L15)
- `internal/eval_harness/error_categorizer.go` — `api_error` catch-all semantics
- OpenRouter observatory Broadcast spans — the provider-side instrument that settled the 2026-08-26 incident

## Future Work

- Feed quality signals into eval-elo as covariates (difficulty vs. quality-failure correlation).
- A `--quality-report` summarizing per-run detection counts for the retro loop (rides existing `eval_analysis` tooling).
- If loop detection proves the top lever on rig models, extend to near-duplicate detection (token Jaccard > 0.9) — separately A/B'd.