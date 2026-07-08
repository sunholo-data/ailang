# M-THREE-CAMPS Re-run — Fair Cross-Language Benchmark on the Local Rig

**Status**: ✅ Implemented (v0.26.0) — eval re-run completed (Aver harness fixes 0fcae605)
**Target**: v0.25.1 (methodology / eval run — no language changes)
**Priority**: P1 (Medium) — corrects a public, unfair scoreboard; not blocking a release
**Estimated**: ~1 day of rig wall-clock + ~2h analysis/writeup (mostly unattended)
**Dependencies**: Aver harness fixes #240/#241 (commit `0fcae605`, merged to `dev`) — **already shipped**

> **Doc type**: This is an **evaluation-methodology** doc, not a language-feature design. It defines
> *how to re-run the three-camps cross-language benchmark correctly* on the local Ollama rig now that
> the Aver harness has been fixed. It introduces no new syntax, types, or runtime behaviour. The Axiom
> Compliance section is scored for completeness but most axioms are N/A (score 0) for a process doc.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Pins reproducible methodology: fixed binary SHA, fixed toolchain versions, N≥3 trials, recorded seeds/dirs |
| A2: Replayability | +1 | Every trial recorded to `eval_results/` + observatory.db; run is re-derivable from a commit SHA |
| A3: Effect Legibility | 0 | No language effect changes |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | No type-check changes |
| A6: Safe Concurrency | 0 | `-parallel 1` is an ops constraint, not a language one |
| A7: Machines First | 0 | No prompt/codegen change to AILANG itself |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | +1 | Records tokens/cost/wall-clock per trial; local models are $0 token cost |
| A10: Composability | 0 | N/A |
| A11: Structured Failure | +1 | Honest reporting: missing-feature failures (e.g. Aver no bitwise) reported, not pre-filtered |
| A12: System Boundary | 0 | N/A |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced (methodology *increases* determinism)
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

## Problem Statement

The public [Three Camps self-audit scoreboard](../../../docs/docs/guides/three-camps-self-audit.md) ranks four
languages on 49 smoke+core benchmarks (single model `claude-haiku-4-5`, run **2026-05-21**):

| Language | Pass rate |
|---|---|
| Python | 38/49 — 77.5% |
| AILANG | 36/49 — 73.4% |
| MoonBit | 29/49 — 59.1% |
| **Aver** | **15/49 — 30.6%** |

**Current State — the Aver number is not a fair measurement:**

- That run used the **old hand-rolled `prompts/aver.md`**, which had verified defects:
  it taught integer `/` (Aver's `/` is **Float-only**; integer division is `Int.div : Result<Int,String>`)
  and treated `Int.mod` as returning `Int` (it returns `Result<Int,String>`) — both produce *guaranteed*
  type errors — and it omitted features several benchmarks need (`verify fn trace`/`given`, independent
  products `(a,b)!`, `Tuple<A,B>`, `Bool.and/or/not`). Verified by diffing against upstream `averlang.dev/llms.txt`.
- The self-repair retry loop fed the LLM **`aver run`'s one-line stderr** instead of **`aver check`'s**
  structured diagnostics (named categories, `repair:` hints, source excerpts). Verified locally on `aver 0.21.0`:
  `aver run` → `error[2:7]: …` (one line); `aver check` → full `Check:` block with `repair:`.

Both were fixed in **#240 / #241** (commit `0fcae605`, merged to `dev` 2026-06-16). The scoreboard ran
~4 hours *before* the issues were even filed, so its Aver column reflects a harness bug, not the language.

**Impact:**
- We are publicly under-reporting a peer language (Aver author @jasisz raised #240/#241 in good faith; we
  told him on the issues we'd re-measure). Leaving 30.6% standing is the unfair outcome the fix was meant to avoid.
- The whole scoreboard is **one cheap model**; the "multi-model run" was explicitly deferred.
- No re-run has happened since the fix — the binary on the rig still embeds the old prompt until rebuilt.

## Goals

**Primary Goal:** Produce a *fair, reproducible, multi-trial* cross-language scoreboard on the local Ollama
rig using the **corrected** Aver harness, and quantify how much of Aver's prior 30.6% was harness-induced.

**Success Metrics:**
- The embedded Aver prompt on the rig is verified as the corrected one **before any trial runs** (hard gate).
- A 4-language (AILANG, Python, MoonBit, Aver) scoreboard re-run on ≥1 local model, smoke+core (49), **N≥3 trials**, median/best-of-N reported.
- An **A/B** on one model (old vs corrected Aver harness) isolates the prompt+diagnostics lift as a single number.
- The self-audit page is updated with the new numbers and an explicit footnote that the 30.6% used the pre-#240/#241 harness; loop closed back on #240/#241.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Model is **local Ollama** (gemma4:26b primary), not cloud `claude-haiku-4-5` | The old 30.6% was cloud-haiku; a local model is a *different* generator, so the re-run is **not** a like-for-like delta vs 30.6% unless we A/B on the same model | human | design | high |
| **A/B the Aver harness fix on one local model** (old prompt+`aver run` vs new prompt+`aver check`) | Only clean way to attribute the lift, since the generator model changed | human | design | med |
| Tier = **smoke (17) first, then smoke+core (49)** | Smoke verifies plumbing cheaply before committing ~hours of rig time on 49×N×4-lang | agent | runtime | low |
| **N≥3 trials**, report median/best-of-N | Local OS-model variance is real (5/17 benchmarks flipped between identical runs per M-EVAL-LOCAL-OLLAMA) | human | design | low |
| Peer toolchains (`aver`, `moon`, `python/uv`) installed + version-pinned on the rig | A missing toolchain = harness error miscounted as a language failure | agent | compile | med |

### Design Freeze

Before the rig run begins, these must be resolved:

- [ ] **Model choice confirmed**: gemma4:26b primary; list any second local model for cross-check (e.g. a Qwen3 coder). Owner: human.
- [ ] **A/B scope confirmed**: A/B old-vs-new Aver harness on gemma4:26b, Aver-only, smoke+core. Owner: human.
- [ ] **Comparison framing confirmed**: report "fair Aver on local model" + "A/B lift", and explicitly do **not** claim a delta against the cloud-haiku 30.6%. Owner: human.

## Solution Design

### Overview

Reuse the existing multi-language eval harness (it already supports `-langs ailang,aver,moonbit,python`)
and the `local-ollama-eval` operational workflow. The novelty here is **correctness discipline**, not new code:
(1) prove the rig's binary embeds the corrected Aver prompt, (2) prove the peer toolchains are present and
pinned, (3) A/B the fix to attribute the lift, (4) N≥3 for variance, (5) honest failure reporting.

### Architecture (run topology)

1. **Generator**: local Ollama model (gemma4:26b) via opencode subprocess — `$0` token cost, on the rig.
2. **Graders / runners** (per benchmark, per language):
   - AILANG → `ailang run`
   - Python → `uv run` (pinned interpreter)
   - MoonBit → `moon run`
   - Aver → `aver run` (authoritative for pass/fail) with **`aver check` enrichment on compile-class failure** (the #241 change)
3. **Observability**: OTLP → local `ailang serve` (port 1957) → `observatory.db`; live view via `ailang chains live`.
4. **Artifacts**: `eval_results/rotation/<date>/<time>_<model>_<tier>_<lang-set>/`.

### Implementation Plan

**Phase 0: Rig pre-flight (correctness gates)** (~30 min)
- [ ] On the rig, `git pull` `dev` to ≥ `0fcae605`, then `make install` (embeds `cmd/ailang/prompts/aver.md`).
- [ ] **Prompt gate (binary-level)**: the corrected Aver prompt is loaded internally by the harness
      (`langreg/aver.go` → `LoadPrompt("aver")`); there is no CLI dump for it, so verify the *installed binary*
      embeds it via its `go:embed`'d string:
      `strings "$(which ailang)" | grep -c 'averlang.dev/llms.txt'` MUST be ≥1, and
      `strings "$(which ailang)" | grep -c 'Authoritative sources for this prompt'` MUST be 0.
      (Source-level sanity: `head -6 cmd/ailang/prompts/aver.md` shows the `#240` header before `make install`.)
- [ ] **Toolchain gate**: `aver --version` (expect 0.21.x), `moon version`, `uv --version`, `ailang --version` — record all four.
- [ ] Run `.claude/skills/local-ollama-eval/scripts/verify_setup.sh` and `warmup_rig.sh`.

**Phase 1: Smoke plumbing check — Aver only** (~10–20 min)
- [ ] One Aver benchmark end-to-end on gemma4:26b to confirm the path runs and `aver check` enrichment appears on a failure.
- [ ] Confirm a *passing* Aver solution is graded PASS (no false fail from `aver check`'s stricter `module` requirement — the #241 design guards this; `TestAverRunner_Success` covers it).

**Phase 2: A/B the harness fix (Aver-only, one model)** (~2–3h)
- [ ] Run A: corrected harness (current `dev`), Aver, smoke+core, N≥3.
- [ ] Run B: old harness (checkout pre-`0fcae605` `prompts/aver.md` + revert the runner enrichment), Aver, smoke+core, N≥3.
- [ ] Report `ΔpassRate = A − B` as the isolated prompt+diagnostics lift.

**Phase 3: Full 4-language scoreboard** (~4–8h unattended)
- [ ] `-langs ailang,aver,moonbit,python`, smoke+core (49), N≥3, gemma4:26b.
- [ ] (Optional) repeat on a second local model for a cross-check.

**Phase 4: Report + close the loop** (~2h)
- [ ] Update `docs/docs/guides/three-camps-self-audit.md` with the new table + footnote dating the old 30.6% to the pre-#240/#241 harness.
- [ ] Post the corrected Aver number + A/B lift back on #240/#241 (and tag @jasisz).
- [ ] CHANGELOG entry under the existing Aver eval block.

### Files to Modify/Create

**New (result artifacts, generated):**
- `eval_results/rotation/<date>/…_gemma4-26b_smoke-core_4lang/` — scoreboard run output
- `eval_results/rotation/<date>/…_gemma4-26b_aver_ab-{old,new}/` — A/B runs

**Modified:**
- `docs/docs/guides/three-camps-self-audit.md` — new scoreboard + dated footnote (~30 LOC)
- `changelogs/v0.10-current.md` — re-run result entry (~5 LOC)
- `internal/eval_harness/models.yml` — only if a second local model is added (template per local-ollama-eval skill)

## Examples

### Example 1: The prompt correctness gate (the single most important check)

The Aver prompt is embedded into the binary via `go:embed` and loaded internally by the harness — it has
no CLI dump. Verify the *installed* binary, not just the repo file. Built from current `dev`, the corrected
marker appears (2×) and the old header is gone (0×):

**Before (rig binary still embeds the old prompt — DO NOT RUN):**
```
$ strings "$(which ailang)" | grep -c 'averlang.dev/llms.txt'          # → 0   (corrected prompt absent)
$ strings "$(which ailang)" | grep -c 'Authoritative sources for this prompt'  # → 1   (old prompt present)
```

**After (`make install` from dev ≥ 0fcae605):**
```
$ strings "$(which ailang)" | grep -c 'averlang.dev/llms.txt'          # → ≥1  (corrected prompt embedded)
$ strings "$(which ailang)" | grep -c 'Authoritative sources for this prompt'  # → 0   (old prompt gone)
```

### Example 2: Multi-language smoke+core run on the rig

**Before** (cloud, single model, old harness — the run we're replacing):
```
ailang eval-suite --models claude-haiku-4-5 --langs ailang,python,moonbit,aver --tier smoke,core
# → Aver 15/49 (30.6%), measured with buggy prompt + aver run diagnostics
```

**After** (local rig, corrected harness, N≥3, honest framing):
```
export OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:1957
make eval-core MODELS=opencode-gemma4-26b \
  EXTRA='-agent -langs ailang,aver,moonbit,python \
    -benchmarks <smoke+core 49> -parallel 1 -agent-timeout 2400 \
    -output eval_results/rotation/$(date +%F)/$(date +%H%M)_gemma4-26b_smoke-core_4lang'
# repeat ×3, report median/best-of-N; Aver number is now a fair measurement on this generator
```

## Success Criteria

- [ ] Prompt gate passes on the rig (corrected Aver prompt embedded) — recorded in the run log.
- [ ] Toolchain versions recorded (`aver`, `moon`, `uv`, `ailang`).
- [ ] Phase 1 smoke: an Aver benchmark runs end-to-end; a known-good module-less Aver solution still grades PASS.
- [ ] A/B lift (corrected − old) reported as a single ΔpassRate for Aver on gemma4:26b.
- [ ] 4-language scoreboard, 49 benchmarks, N≥3, median/best-of-N published.
- [ ] self-audit page updated with new numbers + footnote dating the 30.6% to the pre-fix harness.
- [ ] #240/#241 updated with the corrected number; @jasisz tagged.
- [ ] CHANGELOG updated.

## Testing Strategy

**Plumbing (Phase 1):** single Aver benchmark; assert the harness runs `aver run` + `aver check` and grades correctly.
Backed by existing Go tests `TestAverRunner_Success` (module-less program passes — no false fail) and
`TestAverRunner_CompileErrorEnrichedWithCheck` (rich diagnostics surface).

**Variance:** N≥3 per (model, language, benchmark); report median and best-of-N. Single-trial pass/fail is
not trustworthy for local OS models (M-EVAL-LOCAL-OLLAMA: 5/17 flipped between identical runs).

**Honesty:** where a language lacks a feature a benchmark needs (Aver has no native bitwise operators), the
failure is reported, not pre-filtered — matching the original M8 methodology.

## Deferred Decisions

- **Second local model for cross-check** — agent may pick a coder model already on the rig (e.g. a Qwen3 variant) if gemma4:26b alone looks noisy.
- **Whether to also re-run the cloud `claude-haiku-4-5` leg** for a true like-for-like vs the old 30.6% — human decides; out of scope for *this* (local-rig) doc but the natural companion.
- **Exact 49-benchmark list** — derived from the smoke+core tier at run time (agent resolves; exclude meta files like `events.yml`).

## Non-Goals

- **No new AILANG features.** Gaps surfaced (AILANG `PAR_001` on side-channel/multi-output patterns from M8) are filed separately, not fixed here.
- **No new benchmarks.** Same 49 smoke+core grid as the original scoreboard (apples-to-apples).
- **Not the cloud re-run.** Re-running the frontier/cloud models is a separate cloud-eval effort; this doc is the *local-rig* methodology that makes the Aver column fair at $0 token cost.
- **Not a defence of Aver's score.** If Aver still scores low *with* the correct harness (a niche language the local model has little training data for is a legitimate cause), that is an honest finding, not a bug to fix.

## Timeline

**Day 1** (mostly unattended rig wall-clock):
- Phase 0 pre-flight + Phase 1 smoke (~1h attended)
- Phase 2 A/B (~2–3h) + Phase 3 full scoreboard (~4–8h unattended, overnight-friendly)

**Day 1 (later) / Day 2** (~2h):
- Phase 4 report, self-audit page update, #240/#241 close-out, CHANGELOG

**Total: ~1 day rig wall-clock + ~3h attended analysis.**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Rig binary still embeds the **old** Aver prompt → whole run invalid | High | Phase 0 prompt gate (`ailang prompt --lang aver` must show the `#240` header) is a hard blocker |
| Local model ≠ cloud haiku → not comparable to old 30.6% | High | Don't claim a delta vs 30.6%; report fair-Aver-on-local + the same-model A/B lift instead |
| Peer toolchain missing/version-skew on rig → false language failures | Med | Phase 0 toolchain gate records `aver`/`moon`/`uv` versions before any trial |
| Local OS-model variance | Med | N≥3, median/best-of-N |
| `aver check` stricter than `aver run` falsely fails runnable solutions | Low | Already designed against in #241 (run is authoritative); `TestAverRunner_Success` guards; Phase 1 re-confirms live |
| Aver still scores low with correct harness | Low (expected-possible) | That's a valid finding (niche language, sparse training data), reported honestly per Non-Goals |
| Corrected prompt is longer (420 vs 337 lines) → more tokens | Low | Marginal next to opencode's ~100k framework prefill; local cost is $0 |

## Related Documents

**Implemented (inform design):**
- [m-eval-cross-language-benchmark.md](../../implemented/v0_11_0/m-eval-cross-language-benchmark.md) — original multi-language harness this run reuses
- [m-eval-lang-jsgo-sprint-plan.md](../../implemented/v0_15_0/m-eval-lang-jsgo-sprint-plan.md) — adding JS/Go runners (same runner pattern as Aver/MoonBit)

**Planned (related):**
- [m-eval-local-ollama.md](m-eval-local-ollama.md) — the local Ollama rig workflow this doc operationalizes (`-parallel 1`, warmup, variance)
- [m-three-camps-sprint-plan.md](v0_22_0/m-three-camps-sprint-plan.md) / [m-three-camps-language-survey.md](v0_22_0/m-three-camps-language-survey.md) — the survey + scoreboard this re-run corrects
- [m-eval-openrouter-baseline-rotation.md](v0_24_0/m-eval-openrouter-baseline-rotation.md) — cloud OS-model rotation (the companion cloud re-run, deferred here)

## References

- [Design Axioms](/docs/references/axioms)
- Aver harness fixes: [#240](https://github.com/sunholo-data/ailang/issues/240) (prompt), [#241](https://github.com/sunholo-data/ailang/issues/241) (`aver check` diagnostics), [#242](https://github.com/sunholo-data/ailang/issues/242) (agentlanguages.dev listing) — commit `0fcae605`
- [Three Camps self-audit page](../../../docs/docs/guides/three-camps-self-audit.md) — the scoreboard being re-run
- `local-ollama-eval` skill — operational checklist for the rig run

## Future Work

- A standing cross-language rotation (not just AILANG-only) on the rig, so peer-language regressions/fixes are caught continuously rather than via one-off scoreboards.
- The companion **cloud** re-run on frontier models (the deferred "multi-model run") for a like-for-like comparison against the original `claude-haiku-4-5` numbers.

---

**Document created**: 2026-06-16
**Last updated**: 2026-06-16

---
DESIGN_DOC_PATH: design_docs/planned/m-three-camps-rerun-local-rig.md
