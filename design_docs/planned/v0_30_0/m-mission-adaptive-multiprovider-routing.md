# M-MISSION-ADAPTIVE-MULTIPROVIDER-ROUTING: the heterogeneous model fleet — quota-aware selection, design quorums, cross-provider executors, local-GPU lane

**Status**: Planned (SCOPE EXPANDED 2026-07-14 per Mark: from availability-fallback to the
mission's standing model-fleet architecture — quorum design review, OpenAI Sol + Gemini in the
loop, local bare-metal GPU lane, and evidence-based (provider, model, task-class) assignment)
**Target**: v0.30.x (mission infrastructure — not a v1.0 release gate; the loop keeps running while this is built)
**Priority**: P1 → elevated (Anthropic quota is now the mission's binding constraint — Mark
2026-07-14 "my quota is draining faster now"; every phase here reduces per-iteration Anthropic
burn or adds a non-Anthropic lane)
**Estimated**: Phase A ~0.5d · Phase B ~1–2d · Phase C ~2–3d · Phase D ~2–3d · Phase E ~3–4d
(phase-gated; A+B are the committed near-term slice)
**Dependencies**: `tools/launchd/mission-control.sh` (the driver), the mission model routing policy in [../../v1-mission.md](../../v1-mission.md); leverages existing `internal/coordinator/provider_executor.go` (multi-CLI executors: claude, codex, opencode, motoko, pi), `internal/ai/{anthropic,openai,gemini,ollama,openrouter}` (multi-provider text generation), `internal/eval_harness/models.yml` (gpt-5.6-sol registered 2026-07-12), and the rig's local-model executors (the eval rotation's motoko/opencode/pi on qwen3.6 — the "local bare-metal GPU agent" already exists)
**Author**: Fable/Opus mission session, requested by Mark 2026-07-11; fleet expansion 2026-07-14

---

## Problem Statement

The mission outer loop runs on a single hardcoded model at a time. When that model's quota is
exhausted, the loop either stalls (rate-limit errors → failed iterations) or requires a human to
manually switch models. On 2026-07-11 the weekly **Fable** bucket hit 98% while weekly-all-models
was only 67% and the 5-hour window 39% — so Opus was fully usable, but the switch was manual and
we papered over it with a **time-boxed override that hardcodes the Monday reset** (a date, not a
signal). Three weaknesses:

1. **No self-monitoring.** The driver cannot read the subscription quota gauges (verified: no
   `claude` CLI usage command; `~/.claude/policy-limits.json` is policy not usage;
   `stats-cache.json` is stale local activity; the real gauges are server-side, app-only). It only
   learns of exhaustion *reactively*, by an iteration failing.
2. **Single provider.** Everything runs through `claude -p` (Anthropic-only). If the shared
   Anthropic weekly bucket is exhausted, the whole loop stops — there is no cross-provider fallback
   even though OpenAI and Gemini credits may be sitting unused.
3. **Homogeneous evaluation.** With controller+executor on the same model (the current Opus state),
   the evaluator judges its own model family — the generator≠judge *model* diversity is lost (see
   the caveat in v1-mission.md's routing table).

## Goals

**Primary**: the loop selects the best *available* model each iteration from a preference-ordered,
multi-provider list, falls back automatically on quota exhaustion, and recovers automatically when
the preferred model is usable again — with no hardcoded dates and no human in the path.

**Success metrics**:
- Zero mission stalls attributable to single-provider quota exhaustion over a 2-week window.
- The Monday-hardcoded override is deleted; recovery is signal-driven (a probe), not date-driven.
- Cross-provider evaluation available: executor and evaluator can be different *families*
  (e.g. Claude executes, GPT/Gemini judges) — genuine independence, stronger than Fable-vs-Opus.

## Design

### The core mechanism — preflight probe + preference list (replaces the time-box)

A per-iteration, ordered preference list with a cheap liveness probe. Pseudocode for the driver:

```
PREFERENCES=( "claude:claude-fable-5" "claude:claude-opus-4-8" "codex:gpt-5-x" "gemini:gemini-3-x" )
for entry in "${PREFERENCES[@]}"; do
  harness=${entry%%:*}; model=${entry#*:}
  if probe "$harness" "$model"; then   # 1-token call; success = usable & authed & under quota
    CONTROLLER_HARNESS=$harness; CONTROLLER_MODEL=$model; break
  fi
done
```

- **Probe** = a 1-token request. A usage-limit error (or auth failure) means "not available now" →
  fall through. Success means "usable" → select. This answers the only question that matters —
  *can I use this model right now?* — without reading any gauge.
- **Auto-recovery** is free: once the preferred model's probe succeeds again (quota reset, early or
  late), the next iteration selects it. No date to maintain.
- **Cost**: one tiny probe per candidate per iteration until the first success — negligible, and it
  spends a sliver of the preferred provider only when that provider is actually available.

### Provider/harness abstraction

The controller is not just a model string — it is a **(harness, model)** pair, because different
providers need different agentic CLIs:

| Provider | Controller harness | Status today |
|---|---|---|
| Anthropic | `claude -p` (Claude Code) | live (current mission driver) |
| OpenAI | `codex` | **already an integrated executor** (`internal/coordinator/provider_executor.go`) — not yet used as the *controller* |
| Gemini | managed_agents / `gemini` | Gemini CLI retired v0.22.0 (M-MANAGED-AGENTS); managed_agents path exists for gemini agent-mode |

The inner-loop **executors** are already multi-provider via `provider_executor.go` (claude, codex,
opencode, motoko, pi) — so Phase 2 is mostly wiring, not new capability.

### Phasing (A+B committed near-term; C–E opt-in as evidence accrues)

- **Phase A — adaptive Claude-family fallback (~0.5d).** Preference list limited to Anthropic
  models (Fable → Opus). Probe-based selection replaces the hardcoded expiry override. Delivers
  self-monitoring + auto-recovery immediately, zero new provider surface.
- **Phase B — design-doc QUORUM review (~1–2d; Mark's headline ask, and the cheapest
  non-Anthropic win).** Design docs (and optionally sprint plans) get N independent frontier
  reviews before execution: **gpt-5.6-sol + gemini (latest pro) + the Claude controller**, each
  scoring against the design-doc-creator hard gates (premise verification, Conflict Surface,
  axiom compliance) + a free-form "what would you reject". This is **pure text-in/text-out** —
  it rides `internal/ai/{openai,gemini}` providers directly, NO executor/CLI work — which is why
  it's days not weeks. Controller synthesizes: unanimous-pass → proceed; any-reject → the
  objection goes back through the doc author before planning. Quorum verdicts recorded in the
  log's routing-evidence rows (they're also the seed data for Phase E's assignment table).
  Cost note: 2 extra frontier calls per design doc ≈ cents; saves whole Opus sprints when a bad
  premise is caught pre-execution (iterations 22/27 both had planners correct doc mechanisms —
  quorum catches these earlier and off-quota).
- **Phase C — cross-provider executors (~2–3d).** Inner-loop planner/executor can run on Codex
  (GPT) or a Gemini managed-agent when Anthropic is constrained — or by assignment, not just
  fallback — via the existing executor registry. Sprint quality per (provider, model,
  task-class) recorded in the routing-evidence rows; the same evidence rule that governed
  Opus-vs-Sonnet extends to providers.
- **Phase D — the local-GPU lane (~2–3d).** Route long-running, low-urgency task classes to the
  rig's OWN local-model executors (motoko/opencode on qwen3.6) — **zero marginal cost, slow,
  always-on**. Candidate classes: corpus soak/fuzz sweeps, doc reality-check batches (the ghost
  audits), example-coverage generation, retry-until-green mechanical fixes. Constraints: GPU
  steps take `rig_lock_acquire nowait` + yield to the eval rotation (two-tier rule); output
  always lands behind the SAME evaluator gate as cloud work (local models get no quality
  discount). This is wiring, not new capability — the eval rotation already drives these
  executors daily.
- **Phase E — full assignment + cross-family evaluation (~3–4d).** The endgame Mark described:
  a standing **(provider, model) × task-class assignment table** in the charter, updated by the
  evidence rule (≥3 rows per cell to change routing), covering controller, design, plan,
  execute, evaluate, mechanical, and long-running lanes — including coordinator **cloud-run
  jobs** via the existing cloud dispatcher. Evaluation goes cross-FAMILY by default (a GPT/Gemini
  judge on Claude-written code and vice versa) — stronger generator≠judge independence than any
  same-vendor pairing, and it converts the quorum from a design-time gate into a fleet-wide
  quality property.

### What this builds on (verified, 2026-07-11)

- `internal/coordinator/provider_executor.go` — unified multi-CLI executor (claude, codex,
  opencode, motoko, pi); "adding a new executor only requires creating the [adapter]".
- `internal/ai/{anthropic,openai,gemini,ollama,openrouter,configdriven}` — six text-gen providers.
- `internal/eval_harness/models.yml` — the model registry the eval harness already routes across.

## Routing (per PROGRAM.md §4)

Harness/mission-infrastructure lane — **not** a motoko-core change, **not** an AILANG language
feature (the AILANG `std/ai` multi-provider angle is a *separate, optional* enabler, cross-linked
below, not required by this doc). Phase 1 is a `tools/launchd/` + driver change; Phases 2–3 extend
the coordinator's existing executor registry.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|---|---|---|
| Probe spends quota when the preferred model is scarce | Low | 1-token probe; only runs until first success; skip-probe within N minutes of a known-good selection (cache last-good) |
| A weaker fallback provider lands lower-quality sprints | Med | Evidence rule: record (provider, round-1 score, corrections) per sprint; demote a provider that underperforms — same mechanism as the model-routing table |
| Cross-provider controller behaves differently in the mission-control skill (tool-call formats, etc.) | Med | Phase-gated: Phase 1 stays Anthropic; Phase 3 validates a provider on supervised iterations before it joins the controller preference list |
| Probe false-negatives (transient 5xx read as "unavailable") vs the real signal (429/quota) | Med | Match the *quota/limit* error signature specifically; transient errors retry the same model, don't fall through |
| Non-Anthropic phases spend API credits (OpenAI/Gemini keys), unlike subscription-billed Claude | Med | Per-phase budget caps + the cost-per-item recorded in routing evidence; Phase B is cents/doc; Phase C sprints get an explicit per-sprint dollar cap in the driver env |
| Quorum reviewers rubber-stamp (LGTM bias) | Med | Prompt reviewers to REJECT-by-default with a required "strongest objection" field; track per-reviewer catch-rate in evidence rows; drop a reviewer whose objections never land |
| Local-GPU lane produces low-quality output cheaply (volume ≠ value) | Med | Same evaluator gate as cloud work, no exceptions; lane limited to task classes with deterministic verification |

## Non-Goals

- Reading the subscription quota gauges directly — verified not exposed to the driver; the probe is
  the deliberate substitute.
- Provider *load-balancing* for cost optimization — this is availability fallback, not a cost
  router (cost work is the separate m-cost-per-success-kpi, v1.0 clause 5).
- Changing the inner-loop skills' contracts — only which (harness, model) executes them.

## Verification Log

| Claim | Method | Result |
|---|---|---|
| No CLI quota command | `claude --help` subcommand scan | Confirmed absent |
| Quota gauges not in local files | read `~/.claude/policy-limits.json` (policy), `stats-cache.json` (stale activity) | Confirmed server-side only |
| Codex is an integrated executor | `internal/coordinator/provider_executor.go` grep | Confirmed (`_ ".../executor/codex"`, executorName list) |
| Six text providers exist | `ls internal/ai/*/` | Confirmed anthropic/openai/gemini/ollama/openrouter/configdriven |
| Gemini CLI retired, managed_agents successor | provider_executor.go comment | Confirmed (retired v0.22.0 M-MANAGED-AGENTS) |

## Open Questions (for the design/sprint phase)

- Probe cadence: every iteration vs. cache last-good for N minutes? (cost vs. freshness)
- Where does the preference list live — driver env, a `~/.ailang/state/mission-model-prefs`
  file, or the charter's routing table as the single source of truth?
- Does Phase 3's cross-family evaluation become the *default* even when Anthropic quota is fine
  (independence as a feature, not just a fallback)?

## Related Documents

- [../../v1-mission.md](../../v1-mission.md) — the mission whose driver + routing table this changes; the 2026-07-11 quota event + evaluation-independence caveat motivate it
- [m-eval-local-ollama.md](../m-eval-local-ollama.md) — the local-Ollama provider path (another non-Anthropic model source the preference list could include for cheap iterations)
- PROGRAM.md §4 — routing rule (harness/mission-infra lane)
- *(optional enabler, not required)* an AILANG `std/ai` provider-routing feature — if the loop ever runs an AILANG program to pick models, `std/ai`'s multi-provider `call` could back it; out of scope here

---

**Document created**: 2026-07-11
