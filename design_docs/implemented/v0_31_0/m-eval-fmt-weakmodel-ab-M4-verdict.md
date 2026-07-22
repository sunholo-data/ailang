# M-EVAL-FMT-WEAKMODEL-AB — M4 Verdict (final)

**Milestone**: M4_VERDICT (write-up only; no GPU, no eval runs, no compiler changes).
**Date**: 2026-07-22 (mission iteration 77).
**Inputs**: the frozen preregistration ([`…-prereg.md`](m-eval-fmt-weakmodel-ab-prereg.md) §5–§6) and the
evaluated M3 analysis ([`…-M3-results.md`](m-eval-fmt-weakmodel-ab-M3-results.md), sprint-evaluator PASS
87/100, stats hand-verified). This milestone states the H1 verdict, distinguishes formatter benefit from
treatment-delivery failure, and records the adoption-policy stance. It introduces **no new data** — the
numbers below are exactly M3's, re-stated as the closing verdict.

---

## Verdict

> **NEUTRAL — true NULL, treatment proven delivered.**
>
> The LANDED `format_ail.sh` PostToolUse fmt hook, though fully and cleanly delivered (32/32 = 100%
> exit-0 formatting, 0% refusal), produced **no meaningful pass-rate or convergence advantage** for the
> weak model `claude-haiku-4-5` on the frozen easy→medium benchmark set. This is a genuine null result,
> **not** an "unevaluable" outcome: the treatment was demonstrably applied, so the absence of an effect
> is a real measurement, not a delivery failure.

Mapping to the M4 acceptance vocabulary (helps / neutral / harms / unevaluable):

| Candidate verdict | Supported? | Why |
|---|---|---|
| **helps** | ✗ | Overall delta ON−OFF = **+0.033**; Newcombe 95% CI **[−0.083, +0.167]** includes 0 and the point estimate is below the frozen **+0.10** support threshold. H1 not supported. |
| **harms** | ✗ | ON did not regress any control task (fizzbuzz/adt/gcd/hof/json all 5/5 both arms); ON 30/30 ≥ OFF 29/30. No evidence of harm. |
| **unevaluable** | ✗ | The prereg §6 "void" clause triggers only if `fmt` refused/no-opped enough to prevent treatment. It did **not**: delivery = 32/32 = 100% exit-0, 0% refusal (vs the ~8% fail-closed baseline). The void clause does not fire. |
| **neutral (NULL)** | ✓ | Delta CI includes 0 AND point < +0.10 (prereg §6 NULL rule), with treatment integrity proven. |

## Benefit vs delivery-failure (the M4 discrimination the prereg demanded)

The mission's treatment-integrity guardrail requires separating "the hook doesn't help" from "the hook
never ran". M3 settles this cleanly from the banked `fmt_hook_events`:

- **Treatment delivered** = 32/32 captured `.ail` edits formatted at exit 0 (Wilson 95% [0.893, 1.000]);
  **0** defers/refusals/errors — cleaner than the prereg's anticipated ~8% fail-closed rate.
- **Arm gating clean** = the OFF arm banked **0** hook events (no treatment leak).
- Therefore the null is attributable to the **formatter having no effect on this model/benchmark regime**,
  NOT to a treatment that failed to reach the code. This is the distinction M4 exists to make, and it lands
  on the "genuine null" side.

## Why the null (mechanistic reading, not a new claim)

`claude-haiku-4-5` is **near-ceiling** on this frozen easy→medium set: 58/60 pooled runs pass, ~1.07 `.ail`
edits/run, compile-stuck 0/30 (ON) vs 1/30 (OFF). The hook's hypothesized mechanism — canonical formatting
removing syntax drift that would otherwise drive compile-stuck spirals — has **almost no headroom** to act,
because the spirals it rescues essentially do not occur here. A hook that fixes drift cannot show a benefit
where there is no drift to fix.

## Statistical honesty (carried from M3)

N=30/arm at a near-ceiling proportion yields a wide pooled delta CI ([−0.083, +0.167]). This experiment can
confidently **rule out a large positive effect** (≥ +10 pp is not supported) but **cannot resolve a small
few-pp effect**. Green-stability was reported **NOT-COMPUTABLE** (no per-edit typecheck stream banked) rather
than fabricated; edits-to-first-green is a labeled proxy. None of these limits touch the PRIMARY verdict,
which rests entirely on fully-computed pass-rate + delivery data.

## Adoption-policy stance (explicit — prereg + mission guardrail)

**No adoption-policy change is made off this result.** Per the design doc's own out-of-scope clause and the
mission's "no single-benchmark/run policy change" rule:

- The fmt hook stays **opt-in**, exactly as LANDED. This null does **not** argue for making it mandatory, and
  (because there is no evidence of harm) it does **not** argue for removing or discouraging it either.
- A null-at-ceiling is **not** evidence the hook is useless in general — only that it is neither helpful nor
  harmful for a near-ceiling weak model on easy→medium tasks with demonstrably-delivered treatment.

## Follow-up (queued as evidence, not executed here)

Testing H1 where it could actually bite requires a regime with real syntax-drift spirals — i.e. **genuine
headroom**. Two independently-recorded findings point the same way:

- iter-75: `claude-haiku-4-5` on the harder frontier+stretch tiers scored 34/37 (92%) with **no spiraling** →
  even the harder cloud tiers give haiku little drift to correct.
- The near-1-edit convergence here (§3 of M3) confirms the easy set has ~zero convergence headroom.

⇒ A properly-powered re-test needs a **genuinely weaker model** (e.g. a small on-device Ollama model, which
would additionally require the GPU `rig.lock`) and/or a **harder benchmark tier** chosen to induce
compile-stuck spirals in the OFF arm. This is a scoping note for a future A/B, **not** a commitment made by
this sprint.

---

## Sprint closeout

All five milestones complete: M1 preregistration · M2a fmt-hook toggle build · M2b matched execution
(60 runs banked) · M3 analysis (evaluated 87/100) · **M4 verdict (this document)**. Evidence published across
positive/negative/null axes as required; the published axis is **NULL**. Language surface: **NONE**
(experiment + docs only). Metered spend across the whole sprint: **$0.00** (subscription/quota-bucket only).
