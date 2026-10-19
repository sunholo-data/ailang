# PROGRAM — The Self-Specializing Harness

**Status:** Living document (the north star above all design docs). Draft 2026-06-28.
**Scope:** AILANG (language + eval harness) × motoko (the agentic core, mk-ast) × the local rig.

This document is *above* a design doc. Design docs, extensions, and AILANG bug fixes are **spawned by**
it and trace **back to** it. It changes as the registries below change; the principles do not.

---

## 1. North Star (the end-state)

A self-improving agentic coding system with three layers, each with an *opposite* trajectory:

- **A stable, minimal motoko core** — the agent loop. Frozen. Touched only to fix a floor defect (a
  crash/overflow), then re-frozen. Change-rate → **0**.
- **AILANG** — a solid language substrate. Gaps (errors, builtins, dialect traps) get fixed as the loop
  finds them. Fix-rate → **declining toward 0** as the language matures.
- **A growing catalog of specialized motoko extensions** — the durable product. Each is a tailored,
  composable, independently-verified capability the operator picks per task. Count + quality → **grows**.

The agent's long-term job is **authoring and curating extensions**, not editing the core.

## 2. Invariants (non-negotiable)

1. **Minimal frozen core.** The core does the irreducible loop and never overflows/crashes. It is not
   where features or tuning live. A core change requires a floor-defect justification + re-freeze.
2. **Improvements land as extensions or AILANG — never core.** (The routing rule, §4.)
3. **Data before conclusions.** No fix without measurement: decompose with `tools/analyze_stuck.py`,
   aggregate N-run friction (`tools/aggregate_run_friction.py`), reproduce before claiming a gap.
4. **Verified fixes only.** Every fix ships with a regression test that captures the *exact* failure,
   and — for anything stochastic — a long-context run proving it before "done." (The compaction
   oscillation regressed 4× because this was skipped; it now can't.)

## 3. The Loop (the engine)

Codified in [m-agent-ergonomics.md](planned/m-agent-ergonomics.md): **measure** (N-run on the rig) →
**aggregate friction** (signal vs noise) → **deep-analyze stuck** (per-error dossier: full error +
per-attempt invariant) → **route** (§4) → **implement** (design doc → execute → deterministic verify) →
**re-measure** (the targeted entry vanishes from the stuck set). Every iteration runs this.

## 4. The Routing Rule (the heart)

When the loop surfaces friction, classify it into exactly one lane:

| Signal | Lane | Artifact | Trajectory |
|---|---|---|---|
| Bad/unfixable error, missing builtin, dialect trap, type-system gap | **AILANG fix** | design doc in `ailang/design_docs/` + regression test | → 0 |
| Better strategy: compaction, context orientation, dialect coaching, tool shaping, retrieval | **Motoko extension** | a package under `mk-ast/packages/motoko-ext-*` via the hooks (`on_pre_step`, …) | → catalog |
| The core floor itself is wrong (crash / overflow / data loss) | **Core fix** (rare) | minimal core change + trajectory test, then re-freeze | → 0 |

Default bias: **if it can be an extension, it is an extension.** The core floor only guarantees "never
crash, never overflow"; everything smart is above it.

## 5. Registries (living)

### 5a. Benchmark ladder (rising difficulty — each rung a friction engine)
- **Rung 1: `docx_reimplement`** — single-file reimplementation against a typed ADT + XML parsing.
  *Current.* Gated by context-overflow (now fixed, verifying) + the type-error/dialect walls.
- *Future rungs:* multi-file refactor · larger reimplementation · cross-module reasoning · long-horizon
  task with retrieval. (Add a rung only once the prior one passes reliably.)

### 5b. Extension catalog (the growing product)
| Extension | Task class it specializes | Status |
|---|---|---|
| compaction-strategy (`on_pre_step`) | long-context tasks: smart elision, keep-current-file, semantic UPDATE-merge, AI-summary | **planned (first)** — core floor done (M-COMPACT-CALIBRATED), strategy is the extension |
| context-orientation | tasks with many files: filepath headers, iface enrichment surfacing | candidate |
| dialect-coach | AILANG authoring: pre-check / auto-fix common dialect slips before the model's check | candidate |
| decision-gate (`sunholo/decisions`, TypeSafe Jev) | typed, calibrated decisions at loop choice points: friction→lane routing, done-gate second opinion, `api_error` sub-classification | **measured, not acting** — shadow pilot 2026-09-18: 14/20 vs declared lane, = a cheap LLM at 1/30 the latency; 4 misses are label defects (docs routed AILANG-substrate work as "extension"). See [m-ai-decide-system-one-shadow-report.md](implemented/v0_40_1/m-ai-decide-system-one-shadow-report.md) |

### 5c. AILANG fix backlog (shrinking)
- **Shipped:** PAR999 panic guard (M1), IMP010 auto-import/wrong-module hints (M2), iface record-fields,
  the `analyze_stuck` dossier + codified loop.
- **Open (data-named):** the **type-error wall** (unification/arity/undefined-var ergonomics) and the
  **dialect cluster** (generic `PAR_NO_PREFIX`/`PAR_UNEXPECTED` — 7/10 stuck sessions; structural, not
  sharper-message). See [project memory: docx stuck = dialect confusion].
- **Open (from the motoko queue grooming, 2026-10-01):** the eval harness's AILANG teaching prompt
  (`ailang prompt`) is **96,908 chars** and rides as the system prefix on **every agent step in every
  harness** (motoko measured ~99k chars ≈ 26k tokens/step); on the local rig that prefill dominates
  step time. Measure what each section earns before trimming. And `ParseChatStepResponse`
  (`internal/ai/openai/step.go`) reads `usage.prompt_tokens_details.cached_tokens` with no test
  pinning it (`cache_usage_test.go` covers `Generate`, a different path) — add the arm and a mutant.

## 6. Health metrics (is the flywheel turning?)
- **Core change-rate** → 0 (commits to `mk-ast/src/core/` excluding the frozen floor).
- **AILANG fix-rate** → declining (fixes per benchmark rung).
- **Extension count + reuse** → growing (catalog entries; how often each is picked).
- **Benchmark difficulty** → rising, **with pass-rate maintained**.

## 7. Eval strategy — how the signal is generated

The thesis: **can we build a language that AI models prefer and use better than existing ones — and can a
weak on-device model in AILANG match or beat a frontier cloud model?** Every benchmark run exists to
answer that, and to feed §3's loop. Three stages, each holding something different constant:

| Stage | Who runs | Langs | Holds constant | Measures |
|---|---|---|---|---|
| **Cloud standard** (post-release) | latest/frontier cloud models (0-shot + repair) | ailang + python | mode = API | benchmark **difficulty** (ELO grading) · cost/tokens · **language preference** (ailang vs python) — *the thesis* |
| **Cloud agent** | cheaper/faster models (agent_suite) | ailang | model overlaps standard | **agent uplift** — what harnessing adds over 0-shot |
| **Local rig** (continuous, $0) | qwen3-6 × 3 harnesses (opencode · Pi · **motoko**) | ailang first, then python/js/go | model + benchmark chunk | **harness comparison** · **on-device-vs-cloud** — *"can weak local match cloud"* |

**Which comparison lives where** (each is only valid *same-identity, same-benchmarks*):
- *Do models prefer AILANG over Python?* → cloud standard, `byLang` ELO. (Not the rig — the rig is ailang-first.)
- *What does agent mode add?* → cloud agent (ailang) vs cloud standard (ailang), same model. Needs the agent set to **overlap** the standard set, else "uplift" conflates model + mode.
- *Can on-device match cloud?* → local rig (ailang) vs cloud agent (ailang), coverage-gated until local AILANG coverage matches.
- *Which harness wins on the same model?* → local rig, opencode vs Pi vs motoko on one qwen — the north-star signal, controlled by construction.

**Rig discipline (AILANG-first).** `tools/launchd/os-rotation-filler.sh` fills *every* core+stretch+frontier
AILANG benchmark for the current version first; once that coverage is "in" it auto-hands-off to the
cross-language pass. A new release resets coverage. This is what makes local ELO comparable to cloud.

**Validity is enforced in the display, not the reader** — coverage is first-class, sparse cohorts are
provisional (never a bogus #1), deltas are intersection-only. The standing discipline is
[m-eval-validity-discipline.md](planned/m-eval-validity-discipline.md); every run has surfaced a new face
of the same bug (comparing things measured differently), and that doc is the fix that stops the cycle.

## 8. Where we are now (2026-10-19)

**Fleet switch rule (Mark, 2026-10-02):** motoko replaces codex/pi as a fleet executor only once it
writes *better* AILANG than codex, judged on real tasks over time (design-doc sprints), not benchmarks.

**Evidence so far, all benchmark or bench-of-real-tasks — none from the fleet yet:**

| Date | Set | Model | Result |
|---|---|---|---|
| 10-02 | 8 frontier benches × 1 | Sol 6.1 (subscription) | motoko 7/8, codex 7/8: tie, motoko ~1.5× input tokens |
| 10-12 | 5 valid real AILANG tasks (acceptance-checked) | Haiku 5.5 | pi 5/5, motoko 5/5; $0.26 vs $0.28; motoko ~+100 s cold start |
| 10-16 | 9 hard benches × 3 | GLM-5.3-Flash | pi 26/27, motoko 20/27 at a 32k cap; lifted cap on motoko's 5 fails: 12/15 vs pi 14/15, ~15× slower |

The October gaps were all harness plumbing, now fixed (system role on direct Anthropic/Gemini,
repetition guard, cache-token pricing, compaction cache-busting, OpenRouter host pool and output cap).
The open GLM gap is behavioural: motoko reasons ~9× more per call than pi and verifies in thought
rather than with tools.

**Real-task evidence is zero.** `ailang-only-executor` and `motoko` are registered on the cloud plane
but no task has been dispatched to either (listing back to 09-14). The instrument that would produce it,
[M-AILANG-LANE-EXECUTOR-SHADOW](planned/v0_55_0/m-ailang-lane-executor-shadow.md) (ratified 10-13),
has not started: its cloud sprint task `task-60e90d30` was killed at 42 min on 10-13
(`codex task failed: signal: killed`) and pushed nothing. M7 also needs the capped shadow key.

**Upstream:** Arni merged our #258 (direct-Anthropic docs), #260 (repetition guard) and #262
(byte-stable compaction summary); #234 (path guard) and #266 (TUI ollama flag) are open, as are issues
#259 (cost meter ignores cache tokens) and #263 (extension `ai_step` is sent the full tool list).
Upstream main has moved 93 commits past our pin base, mostly ADR 034 (tool faults disclose whether their
effect may have happened) and herdr delegation fixes. Our 20 commits are not yet rebased onto it.

**Prior art:** Harness² (Google Research) automates harness editing per task with an LLM judge. We do
not adopt it; the verifier-judged offline variant is noted in
[m-harness-propose-probe-offline.md](planned/m-harness-propose-probe-offline.md), gated on shadow data.

---
*This is a draft to react to — the vision is the operator's; this captures it so it can spawn the work.*
