# M-DX-LONG-SESSION — long-run context resilience: compaction-loop guard, sticky evidence store, late-session tool re-injection

**Status**: Planned
**Target**: v0.35.0
**Priority**: P2
**Estimated**: ~1 session (L1 ~0.5d, L2 ~0.5d, L3 ~0.25d)
**Dependencies**: M-DX-PI-HARNESS (shipped — suite, State Management pattern) · M-EVAL-SEMANTIC-CONTEXT (planned — shares its telemetry instrument; this doc consumes its measurements) · M-MODEL-REGISTRY-SINGLE-SOURCE (planned — contextWindow rider)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Developer-experience tooling (pi extension), no language surface change.

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No language/runtime impact |
| A2: Replayability | 0 | No trace impact (telemetry entries are append-only, auditable in session branch) |
| A3: Effect Legibility | 0 | No effect system change |
| A4: Explicit Authority | +1 | Cancelling pi's compaction is a high-authority action: guarded by explicit thresholds, always surfaced as a persistent notice + session entry; re-arming conditions are mechanical, not heuristic |
| A5: Bounded Verification | +1 | Evidence store capped (entry cap + 1KB/snippet cap); injection is bounded per turn; guard conditions are numeric thresholds, never fuzzy |
| A6: Safe Concurrency | 0 | No concurrency surface |
| A7: Machines First | +1 | Long autonomous runs stop dying to (a) compaction loops, (b) post-compaction amnesia of critical state, (c) tool-schema drift — all three are machine failure modes silently burning rig time today |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | +1 | Compaction statistics surfaced as session entries + `ctx.ui.notify` — the "compaction is invisible" gap (M-EVAL-SEMANTIC-CONTEXT) closes at the pi layer too |
| A10: Composability | +1 | Wraps pi's documented compaction events; composes with quality-monitor (shared appendEntry state pattern) and microrag-context (same injection channel, D1) |
| A11: Structured Failure | +1 | A doomed second compaction becomes a structured PAUSED-notice instead of an unrecoverable session |
| A12: System Boundary | +1 | Separates pi's context accounting from the provider's server-side window (the rig's real overflow surface) |

**Net Score: +5** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 / A3 / A4 / A7 — no violations (A4/A7/A9/A11 strengthened)

## Verification Log

| # | Claim | Method | Result |
|---|-------|--------|--------|
| V1 | pi auto-compacts in-run when `contextTokens > contextWindow − reserveTokens` (default 16k, `keepRecentTokens` 20k), checked after tool batches and before new prompts | pi 0.84.3 `docs/compaction.md` §When It Triggers (L27–37) | Confirmed from docs |
| V2 | `session_before_compact` can cancel compaction (`return {cancel: true}`) or supply a custom summary; the event carries `{reason: "manual"\|"threshold"\|"overflow", willRetry, preparation}`; `session_compact`/`session_compact_failed` report the outcome | pi docs §session_before_compact (L452–490) | Confirmed from docs |
| V3 | Repeated compaction preserves `firstKeptEntryId` boundaries and recalculates `tokensBefore` — repeated under-effective compaction is a modeled-in pi failure mode | pi docs compaction.md (L81) | Confirmed from docs |
| V4 | **A compaction that frees too little retries into a doomed loop with `Nothing to compact`** — externally measured on the same pi, which then pauses auto-compaction instead of retrying | little-coder v1.11.0 changelog + issue #68 (studied 2026-08-30); pi's doc quote at L81 documents the *boundary* behavior this guard protects | Confirmed (external measurement, same harness version family) |
| V5 | **Compaction never fires for the rig's qwen3.6 model**: `context_limit_for("ollama/qwen3.6:…")=0` → `usage_percent`→0 → threshold check is a no-op; long runs risk provider-side overflow; the repo's elision-thrash hypothesis was refuted on the strength of this measurement | `m-ailang-semantic-context.md` §Progress (measured 2026-06-20, telemetry instrument A1–A3) | Confirmed — repo-documented measurement |
| V6 | Extensions can persist session state that survives restarts via `pi.appendEntry` (custom types), and can render TUI-only details separately (`registerEntryRenderer`) | pi docs L15, L1416–1418 | Confirmed from docs |
| V7 | `before_agent_start` exposes `event.systemPromptOptions.selectedTools` + `.toolSnippets` — the exact tool inventory pi loaded, without re-deriving it | pi docs §before_agent_start (L530–565) | Confirmed from docs |
| V8 | Tool descriptions are re-materializable from `systemPromptOptions` without re-parsing provider payloads; layering the reminder as an injected `message` keeps the system prompt unchanged (prefix-cache safe per m-dx-microrag-context D1, same pi mechanics) | pi docs L530–565 + D1 cross-reference | Confirmed from docs + design consistency |
| V9 | Per-run compaction telemetry fields (`compaction_count`, `first_compaction_step`, `compaction_level_max`) already flow in harness result JSON — this doc's rig measurements have a home | `m-ailang-semantic-context.md` §Observability instrument A2 | Confirmed — repo-documented |
| V10 | **No existing extension touches compaction state, evidence stores, or tool re-injection** | `grep -il "compact\|quality\|hallucin" .pi/extensions/*.ts` → empty; suite README inventory (8 extensions) | Confirmed (negative-existence) |
| V11 | Long autonomous sessions exist here concretely: mission-control iterations and rig benchmark runs are the drivers; two `mission-v1` messages today (2026-08-30) evidence long-iteration operational state (controlplane parking message) | `ailang messages list --unread` this session | Confirmed by live observation |
| V12 | Late-session tool-schema forgetting is a measured external failure mode on long small-model runs: TB 2.0 submission fixed by *re-adding tool descriptions + a concision guideline* mid-session | little-coder v0.1.24 changelog entry (studied 2026-08-30) | Confirmed (external measurement) |

## Problem Statement

Three measured failure modes make long-running sessions on this repo unreliable — all invisible in the metrics today:

1. **Compaction death spiral**: when a compaction frees too little (small `contextWindow − keepRecentTokens` budget, or a summary that barely shrinks), pi-side retry semantics can hit a state where the next compaction has nothing to keep and the next threshold check fails — externally measured as `Nothing to compact` brute-limbo (#68). Sub-variant that matters more to us on the rig: **when `contextWindow` is 0/unknown** (V5), threshold compaction *never* fires at all — runs push toward provider-side OOM/overflow with zero warning from pi. Both directions are the same blind spot: **pi doesn't know what the server knows**.
2. **Post-compaction amnesia**: pi's compaction summarizes conversation, but small models re-inject nothing structural — nothing structural is re-injected after it — state that an outer loop depends on (current sprint id, mission iteration state, a mission-relevant test command, "these 3 files are parked, uncommitted") lives only in messages that may summarize out. Mission-control's parked/recovered-iteration record today (V11) is exactly the kind of state a 200-turn iteration can lose.
3. **Late-session tool drift**: on long small-model runs, tool schemas drift out of the effective context (V12 external measurement). Our suite registers 9 custom tools (e.g., `session_protocol_ack`, `ailang_check`, `freshness_report`, `microrag_search` — once that doc lands); nothing re-surfaces them late in a run, which pushes models back to raw-bash guessing exactly when their context discipline is worst.

**Rig-side rider (not extension code):** V5's zero-window cause is a model-registration gap owned by M-MODEL-REGISTRY-SINGLE-SOURCE + the local-ollama-eval rig runbook. This doc's guard is what makes the *failure mode* survivable and *visible* either way; the registration fix removes the trigger. Both are needed: registration for correctness, guard for latency-window safety (e.g., transient wrong registry on a fresh image — which is exactly the machine-drift failure class Tier 0 exists for).

## Goals

**Primary goal:** long pi sessions (mission iterations, rig rotations) degenerate gracefully instead of unrecoverably: compaction that can't converge → PAUSED with a notice; critical session state survives every compaction; tool schemas re-surface late in long runs.

**Success metrics:**
1. Guard e2e: a compaction that fails to reduce usage below (threshold − 5%) → second compaction attempt cancelled with a persistent notice + session entry; re-arms after usage drops below threshold (or a `/clear`, `/compact`, or a smaller turn).
2. Evidence e2e: entry written pre-compaction → re-injected post-compaction (in the `session_compact` handler path and on next `before_agent_start`).
3. Re-injection e2e: after turn T_hi with usage > U%, one bounded tool-reminder message injected; byte-identical system prompt across the same turn (prefix-cache safety).
4. Rig rotation with telemetry on: zero sessions unrecoverable-by-compaction; evidence persists across ≥1 compaction; re-injection visible in transcripts and banked run fields.

## High-Impact Decisions

| # | Decision | Rationale |
|---|----------|-----------|
| D1 | **Guard cancels only `threshold`-reason compactions that would immediately re-fire; `overflow` always passes through** — and a cancel *always* writes `pi.appendEntry` + notifies | pi's own `willRetry` semantics (V2) mean overflow recovery is load-bearing — cancelling it would break pi's recovery. The doomed case specifically is repeated threshold compaction that frees nothing (V4); that is where pausing is safe and is what #68 measured. |
| D2 | **PAUSED state is a first-class, visible, re-arming condition** — never a silent fallthrough, never an error | Follows repo fail-loudly doctrine (CLAUDE.md no-silent-fallbacks): pausing auto-compaction is an authority action (A4), surfaced as a notice + entry, re-armed only by mechanical conditions. |
| D3 | **Evidence = extension-scoped key-value store with a 1KB-per-snippet / 10-snippet budget, re-injected as a single trailing message** | Same budget discipline as little-coder's per-session evidence cap (their Phase-2 KB findings); the store is for *pointers and decisions* (sprint id, parked branches, canonical commands), never for conversation or code — that's what compaction resumes and microrag retrieves. |
| D4 | **Evidence is written *only* by extension commands** (`/evidence <key> <text>`, `/evidence-list`), never parsed from conversation | Parsing would duplicate little-coder's hallucinated-evidence class; an explicit command is auditable in the transcripts we already bank. |
| D5 | **Tool re-injection keyed on `turnIndex` + measured usage, cap 1 message per 10 turns** | V12's fix (re-add tool descriptions) was made permanent by their scaffold; ours measures before injecting (cheap, deterministic trigger). Uses `systemPromptOptions.toolSnippets` (V7) so the reminder is *pi's own summaries*, zero inventory drift. |
| D6 | **All state via `pi.appendEntry` custom entries** (`session-scope keys: evidence, guard-state, last-re injected`) | V6 + existing session-gate State Management + m-dx-quality-monitor D6 — one shared pattern across the suite. |

## Solution Design

One extension, `long-session.ts`, in the existing suite. Event-only; no subprocess calls; no CLI.

### L1 — Compaction-loop guard

```
session_before_compact (reason="threshold") ──►
  last = entry("quality:compaction:last")   // appendEntry store (D6)
  if last && last.freed_tokens < 2000 &&
     last.started_tokens - this.preparation.tokensBefore < 5000 :
       notify("⏸ compaction paused: last compaction freed <2k tokens;
              enlarge /clear or raise contextWindow (see rig runbook)")
       appendEntry("compaction:paused", {reason, tokensBefore, ts})
       return {cancel: true}
  else → remember-at-sink: session_compact handler records freed_tokens =
         tokensBefore - tokensAfter; re-arm if tokensBefore < last.paused_at - 10%
```
- `overflow` reason: never cancel (D1).
- Custom-summary *support* is deliberately **not** used here: pi's default summary is the audited path; a custom summarizer in the same extension would entangle LLM calls with the guard (A-9). If a repo-native summary format is wanted later, it is its own extension (see Future Work → microrag summary).

### L2 — Sticky evidence store

- Store: `Map<key, {text ≤1KB, ts}>` capped at 10 entries LRU; persisted via `appendEntry("evidence", …)`; reconstructed at `session_start` (reason "resume"/"fork") — the reconstruction pattern the session gate already validated for ack state.
- Write surface: `/evidence <key> <text>` command (e.g. `/evidence parked "iter302 work preserved in commit 3bee6b6df"`), `/evidence-list`, plus `evidence_set` registered tool so the model can record its own conclusions (bounded: same caps, last-wins per key).
- Re-injection: `session_compact` handler returns after pi's compaction → next `before_agent_start` prepends `Message` with all entries joined (≤10KB). Post-compaction only — not every turn (per-turn re-mentioning is what `m-dx-microrag-context`'s dedup exists to avoid; post-compaction-only is the one moment the content has actually been lost to summarization, which is the case re-injection is for).

### L3 — Late-session tool re-injection

```
before_agent_start ──►
  if this.turnIndex > 25 && usagePercent > 60% && !injectedSince(10 turns):
      const {selectedTools, toolSnippets} = event.systemPromptOptions
      return { message: { customType: "long-session:tools",
               content: "Tools currently loaded: " + toolSnippets.join("; "),
               display: false } }
```
`display: false` (TUI noise, LLM still sees it — `pi.sendMessage` docs distinguish these). Mirror of the external v0.1.24 fix, but triggered *measured*, not every N turns (V12 rebuilt on V7).

### Rig-rider (documentation only, no code here)

`M-MODEL-REGISTRY-SINGLE-SOURCE` + `local-ollama-eval` runbook get a rider: "no ollama/qwen model may ship with `contextWindow` unset — Tier-1 fail on `freshness_report`-style lint". V5's measurement makes this the single highest-leverage overflow fix; it just shouldn't be implemented in an extension. Telemetry hook: when L1 enters PAUSED for a model, the notice inline-references V5's zero-window diagnosis to teach the runbook fix.

## Examples

**Guards in action (mission iteration):**

```
[t141] context at 92% of window — pi pre-compaction
       session_before_compact(threshold): last compaction freed 800 tokens
       → PAUSED: notify + appendEntry
       → mission session continues (L2 evidence intact: sprint id, parked-commit refs)
[t149] /clear of old turns drops usage to 41% → re-arm notice
```

**Evidence flow:**

```
/evidence parked "uncommitted iter302 files qwer,1234 — do not stash/checkout"
… 40 turns later, compaction fires …
[next before_agent_start] (display) "📌 session evidence (2): parked=…, active-sprint=M-DX-LONG-SESSION…"
```

## Success Criteria

- [ ] L1: simulated no-progress compaction → cancelled exactly once, notice shown, entry written; a `overflow`-reason compaction is never cancelled; re-arm works via smaller turn
- [ ] L2: `/evidence` + `evidence_set` write; store survives compaction (re-injected message observed in next turn's message list via `before_provider_request`-snapshot or transcript capture); 10-entry cap enforced
- [ ] L3: reminder injected once per 10-turn window past T_hi at >60% usage; system prompt byte-identical (asserted); `display:false` verified in TUI
- [ ] Interaction: L1 + m-dx-quality-monitor + m-dx-microrag-context all active → no double-injection at the same turn (one e2e asserting combined extensions)
- [ ] Result JSON / session entries expose `compaction_*` + `evidence_count` for the retro loop (V9 fields remain harness-side; this is the pi-side parity)
- [ ] README table + Tier-0 embedded copy updated; rig runbook cross-link added (V5 rider)

## Testing Strategy

1. **Extension e2e:** the session-gate pattern (headless `pi -p`, scripted turns); L1 needs a harness-level scripted compaction (stub provider returning a small `contextWindow` to force threshold behavior — pi honors provider-reported windows).
2. **State tests:** `/resume` reconstruction; branch isolation after `/fork` (evidence is per-branch — document that fork does not duplicate by design, matching pi's branch semantics).
3. **Guard partition test:** enumerate {threshold × no-progress, threshold × progress, overflow, manual `/compact`} × {guard on, off} — table-driven, no surprises.
4. **Rig rotation (field):** one overnight mission-control iteration with the extension on; verify via session JSONL that PAUSED/PAUSED_ARMED + evidence re-injection appear at the right turns (telemetry is per V9 the leading indicator).

## Deferred Decisions

- **Microrag-backed custom compaction summary**: replace pi's summary with a RAG-grounded one (`session_before_compact` custom summary, V2) — a *separate* change with its own LLM-call and A/B; do not entangle with the guard (explicitly split to keep L1's authority surface mechanical-only).
- **Evidence auto-capture hooks** (auto-record "task failed" on eval-error events): needs a model-agnostic event classification that isn't designed yet; keep the store opt-in until eviction policy (FIFO vs. priority pinning) is measured — defer with m-dx-microrag-context's route telemetry.
- **ContextWindow auto-detect extension** (probe provider `/props` like little-coder does at startup for llama.cpp): belongs in the model-registry workstream where the data-plane lives (D5 rider); revisit if the registry lands without a machine-readable window.

## Non-Goals

- No changes to pi's compaction internals (summarization, cut-point, reserveTokens) — guard and evidence are extension-layer; pi's own `reserveTokens` config remains the primary knob.
- No automatic contextWindow writes (rig-rider is a registry/runbook concern).
- No prompt-level "do compaction"-style re-injection of *conversation* content (the microrag extension owns knowledge retrieval; this doc owns state persistence — deliberately different failure domains).
- Never bypasses the session gate; no subprocess calls.

## Timeline

| Phase | Work | Est. |
|---|---|---|
| 1 | L1 guard + partition e2e | 0.5d |
| 2 | L2 store + commands + tool + resume e2e | 0.5d |
| 3 | L3 measured re-injection + combined-extensions e2e | 0.25d |
| 4 | Rig-rider doc updates + rotation observation window | 0.25d |

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Guard cancels a compaction that would have succeeded | Only fires on measured no-progress (freed <2k tokens immediately after a prior compaction — V4's mechanism); `overflow` never cancelled (D1); explicit manual paths (`/compact`, `/clear`) always re-arm |
| Pausing auto-compaction on a *full* window drifts the session into unrecoverable state | PAUSED is paired with a persistent notice + a rig-runbook pointer (registry fix); the PAUSED state is itself recorded as evidence via L2 (dogfooding) |
| Evidence store becomes a junk drawer | `/evidence` + `evidence_set` are model-visible commands with caps (10 × 1KB), last-wins; retro sees `evidence_count` in run JSON so bloat is measurable, never invisible |
| Re-injected tool reminders grow context on small windows | Measured trigger (turn >25 AND >60%) + 1-per-10-turns cap; message is one line built from pi's own snippets (V7) |
| Interplay with the two sibling extensions' `before_agent_start` handlers | Combined-extensions e2e (Success Criteria); each extension limits itself to: microrag = content-bearing injection, quality = steer-on-quality-fault, this = state persistence + guard. One message each at most in a given turn; observer notes ordering determinism (pi chains handlers in load order, docs L842–849) |
| Local model with 0-context already at OOM before any compaction fires | Mitigation is outside the extension (registry rider + runbook); L1's pause notice educates pointers to it instead of pretending to override provider OOM — silent-fallthrough would be the failure mode, and D2 forbids it |

## Related Documents

- [M-EVAL-SEMANTIC-CONTEXT](../v0_29_0/m-ailang-semantic-context.md) — compaction telemetry (A1–A3 land in the harness; the rig zero-window measurement (V5) is cited here; its Branch-B tool-result truncation is generalized per-mechanism by m-dx-quality-monitor Q3 — coordinate, do not double-sprint)
- [m-dx-quality-monitor](m-dx-quality-monitor.md) + [m-dx-microrag-context](m-dx-microrag-context.md) — sibling extensions sharing the appendEntry state pattern; combined e2e required
- [M-MODEL-REGISTRY-SINGLE-SOURCE](m-model-registry-single-source.md) — registration rider for the V5 trigger
- [M-DX-PI-HARNESS](m-dx-pi-harness.md) — suite, distribution, State Management pattern
- [M-MISSION-CONTROL](../../.agents/skills/mission-control/SKILL.md) — long-iteration controller this protects sessions for; parked-state example today (V11)
- External: little-coder compaction watchdog + `evidence`/`evidence-compact` design — precedent measurements (#59, #68, #73), not a code dependency

## References

- pi 0.84.3 `docs/compaction.md` (thresholds, cut-point rules, `firstKeptEntryId` chains) and `docs/extensions.md` §session_before_compact (L452–490), §before_agent_start (L530–565), `pi.appendEntry` (L15)
- `m-ailang-semantic-context.md` Progress (2026-06-20) — V5 measurement; `internal/eval_harness/` compaction telemetry capture (V9)
- Storage `.ailang/state/` session JSONL — banked-run instrumentation path for acceptance metrics

## Future Work

- Compaction-observability parity for non-rig (cloud) sessions: parse L1's session entries into the same `compaction_rate` report the semantic-context doc specs.
- `/evidence pin` for priority entries that also survive `/clear` warnings (currently out-scope: L2 re-arm semantics only cover compaction).
- Cross-session evidence templates (mission-control could pre-seed per-iteration stores — needs a separate design since mission sessions are driven by skills today).