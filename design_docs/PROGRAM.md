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

### 5c. AILANG fix backlog (shrinking)
- **Shipped:** PAR999 panic guard (M1), IMP010 auto-import/wrong-module hints (M2), iface record-fields,
  the `analyze_stuck` dossier + codified loop.
- **Open (data-named):** the **type-error wall** (unification/arity/undefined-var ergonomics) and the
  **dialect cluster** (generic `PAR_NO_PREFIX`/`PAR_UNEXPECTED` — 7/10 stuck sessions; structural, not
  sharper-message). See [project memory: docx stuck = dialect confusion].

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

## 8. Where we are now (2026-06-28)
On the cusp of the extension era. The core floor is being frozen (PR #75 compaction fix verifying via
docx N=5). Once green: declare the core minimal+frozen, settle #73, and stand up the **first extension**
(compaction strategy). From there the loop runs on the benchmark ladder and routes by §4.

---
*This is a draft to react to — the vision is the operator's; this captures it so it can spawn the work.*
