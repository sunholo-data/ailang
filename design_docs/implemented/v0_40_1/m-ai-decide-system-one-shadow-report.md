# M-AI-DECIDE-SYSTEM-ONE — shadow lane-router report

**Run**: `lane_shadow_2026-09-18.jsonl` · **docs**: 20 · **arms**: jev = `typesafe/jev-1.13-20260917` via OpenRouter `/api/alpha/decisions`; llm = `z-ai/glm-5.3-flash` via OpenRouter chat completions, strict `json_schema` from the same `Question` list, per-label probabilities requested. Same `std/net` transport and 30 s deadline for both. **Nothing acted on any answer.**

**Label set**: hand-read declared PROGRAM.md lanes (`tools/decisions/routed_docs.sh`, each row carries the quoted sentence). The design doc's "45 routed docs" were 45 docs with a *routing section*; **20** state their lane legibly. Treat every number below as a pilot on n=20, not a verdict.

## Headline

| Arm | Answered | Timeouts (30 s) | Other errors | Lane agreement with declared label | Mean / median latency | Total cost | Tokens in/out |
|---|---|---|---|---|---|---|---|
| jev | 20/20 | 0 | 0 | **14/20 = 70%** | 491 / 412 ms | $0.0012 | 28403/1660 |
| llm | 17/20 | 3 | 0 | **12/17 = 71%** | 16377 / 14562 ms | $0.0133 | 20169/16661 |

**Jev ↔ LLM lane agreement** (docs both answered): 15/17 = 88%

## By declared lane

| Declared lane | n | Jev correct | LLM correct (of answered) |
|---|---|---|---|
| ailang_fix | 6 | 5/6 | 4/5 |
| core_floor_fix | 1 | 0/1 | 0/1 |
| extension | 12 | 8/12 | 8/11 |
| other | 1 | 1/1 | 0/0 |

## Does confidence predict correctness?

Jev `confidence` is the vendor's calibrated statistic; the LLM column is `1 − normalised entropy` of its *self-reported* distribution (a proxy, not a calibration claim). Tertiles are within-arm.

| Arm | Band | n | Correct | Mean confidence |
|---|---|---|---|---|
| jev | low | 7 | 5/7 | 0.61 |
| jev | mid | 7 | 3/7 | 0.94 |
| jev | high | 6 | 6/6 | 0.98 |
| llm | low | 6 | 3/6 | 0.57 |
| llm | mid | 6 | 5/6 | 0.76 |
| llm | high | 5 | 4/5 | 0.83 |

## If a consumer had gated on Jev confidence

`gate(answer, t)` per the package: `Act` when confidence ≥ t, else `Escalate` to a frontier turn. Counts over the 20 Jev rows.

| Threshold | Acted | Acted & correct | Acted & wrong | Escalated | Escalated & would have been wrong |
|---|---|---|---|---|---|
| 0.50 | 17 | 12 | 5 | 3 | 1 |
| 0.70 | 16 | 11 | 5 | 4 | 1 |
| 0.90 | 13 | 9 | 4 | 7 | 2 |
| 0.95 | 10 | 9 | 1 | 10 | 5 |

## Per document

| Doc | Declared | Jev lane (conf) | Jev touches_core | Jev severity | LLM lane (top p) | Jev ms | LLM ms |
|---|---|---|---|---|---|---|---|
| m-eval-elo-priority-rotation | extension | ✓ extension (0.98) | 0.03 | 1.10 | ✓ extension (0.95) | 573 | 2777 |
| m-gemini-repo-mount | extension | ✗ ailang_fix (0.49) | 0.03 | 1.28 | ✓ extension (0.85) | 1107 | 13948 |
| m-mission-adaptive-multiprovider-routing | extension | ✓ extension (0.95) | 0.50 | 1.67 | ✓ extension (0.95) | 413 | 9117 |
| m-module-less-run-fail-loud | ailang_fix | ✓ ailang_fix (0.97) | 0.04 | 1.20 | ✓ ailang_fix (0.96) | 523 | 10640 |
| m-eval-standard-confidence-gating | extension | ✓ extension (0.96) | 0.03 | 1.00 | ✓ extension (0.88) | 384 | 5854 |
| m-recorded-stream-api | core_floor_fix | ✗ ailang_fix (0.95) | 0.10 | 1.99 | ✗ ailang_fix (0.94) | 756 | 17295 |
| m-v1-memory-footprint | ailang_fix | ✓ ailang_fix (0.98) | 0.03 | 1.80 | ✓ ailang_fix (0.94) | 400 | 12113 |
| 20260918_compile_error_ailang_compilation_failures | extension | ✗ ailang_fix (0.86) | 0.07 | 1.12 | ✗ ailang_fix (0.89) | 411 | 5274 |
| 20260918_non_agentic_ailang_non_agentic | ailang_fix | ✗ extension (0.91) | 0.18 | 1.28 | ✗ extension (0.85) | 303 | 18695 |
| 20260918_timeout_python_timeout | extension | ✓ extension (0.98) | 0.09 | 1.19 | ✓ extension (0.90) | 426 | 4178 |
| m-list-cons-quadratic | ailang_fix | ✓ ailang_fix (0.96) | 0.03 | 1.58 | ✓ ailang_fix (0.95) | 634 | 21943 |
| m-motoko-fmt-remeasurement-instrument | extension | ✓ extension (0.99) | 0.16 | 1.99 | ✓ extension (0.92) | 427 | 15175 |
| m-pkg-multi-namespace-auth | other | ✓ other (0.81) | 0.05 | 1.45 |   TIMEOUT | 375 | 30001 |
| m-contracts-as-code-vertical | extension | ✓ extension (0.80) | 0.04 | 1.21 | ✓ extension (0.93) | 367 | 21874 |
| m-diag-primitive-field-suggestions | extension | ✗ ailang_fix (0.92) | 0.12 | 0.88 | ✗ ailang_fix (0.95) | 354 | 11178 |
| m-decision-entropy-monitor | extension | ✓ extension (0.57) | 0.07 | 1.42 | ✓ extension (0.70) | 348 | 29078 |
| m-mem-budget-runtime | extension | ✗ ailang_fix (0.94) | 0.03 | 1.83 |   TIMEOUT | 897 | 30001 |
| m-partial-accessor-shape | ailang_fix | ✓ ailang_fix (0.99) | 0.04 | 1.19 | ✓ ailang_fix (0.94) | 329 | 8507 |
| m-pkg-quality-ladder | ailang_fix | ✓ ailang_fix (0.28) | 0.07 | 1.97 |   TIMEOUT | 361 | 30001 |
| m-game-engine-effects | extension | ✓ extension (0.49) | 0.17 | 1.02 | ✗ ailang_fix (0.60) | 434 | 29892 |

## Reading it

- Rows are banked in full (model, id, distributions, usage, latency) under `.ailang/state/decisions/`; this file is a pure aggregation and can be regenerated with `tools/decisions/lane_shadow_report.py`.
- Timeouts are the Net effect's 30 s deadline — the bounded wait the design doc committed to; they are counted, never retried.
- n=20 with a 12/6/1/1 label skew: an arm that always says `extension` scores 60%. Compare arms to each other and to that baseline, not to 100%.

## Interpretation (hand-written, 2026-09-18 — the numbers above are mechanical, this is not)

**What the pilot says.** On 20 self-declared lanes, Jev matched the label 14/20 (70%) and the GLM control arm 12/17 (71% of the 17 it answered within 30 s; 60% of 20 counting its three timeouts). The two oracles agreed with *each other* on 15/17 (88%). Jev did it at 491 ms mean and $0.0012 total; the LLM took 16.4 s mean, $0.013, and timed out three times. So: **same accuracy as a cheap reasoning LLM, ~33× faster, ~11× cheaper, zero timeouts** — on this small, skewed set (always-say-`extension` scores 60%).

**Where both oracles disagree with the label, read the doc, not the oracle.** Four of Jev's six misses are docs where *both* arms said `ailang_fix` (or `extension`) at ≥ 0.86 against the declared lane:

| Doc | Declared | Both arms said | What the doc actually changes |
|---|---|---|---|
| `m-recorded-stream-api` | core_floor_fix | ailang_fix (0.95 / 0.94) | `std/ai`, `internal/builtins`, `internal/effects` — the AILANG runtime; the doc used "core-floor" for what PROGRAM §4 calls the AILANG lane |
| `m-diag-primitive-field-suggestions` | extension | ailang_fix (0.92 / 0.95) | compiler diagnostics in `internal/` — §4's first row ("bad/unfixable error") is the AILANG lane |
| `m-mem-budget-runtime` | extension | ailang_fix (0.94 / timeout) | a runtime memory budget inside the AILANG runtime, declared "extension" citing the default bias |
| `20260918_compile_error_…` | extension | ailang_fix (0.86 / 0.89) | a "compat front-end" between source and the frozen parser — a compiler change by any reading |

These are not oracle errors; they are our docs applying "if it can be an extension, it is an extension" to work *inside the AILANG substrate*, where PROGRAM §4 gives AILANG its own lane. The decision model surfaced a real inconsistency in how the program's own routing rule has been applied. That is a finding about the label set and about PROGRAM.md's wording, and it is the most useful thing this pilot produced.

**The one clean Jev miss escalates.** `m-gemini-repo-mount` → `ailang_fix` at **0.49**. At any threshold ≥ 0.5 the package's `gate` returns `Escalate(0.49)` and a frontier turn decides. `20260918_non_agentic_…` (Jev `extension` 0.91 vs label `ailang_fix`) rests on the weakest label sentence in the set ("link from PROGRAM.md as AILANG fix lane"); the doc body is about harness flags.

**Calibration is not settled at n=20.** The high-confidence tertile was 6/6, the low tertile 5/7, the middle 3/7 — non-monotonic, and the middle band is where the label-disputed docs sit. A reliability curve needs an order of magnitude more rows; D6 banking makes that free as the primitive gets used.

**What this does and does not license.**
- It licenses **Phase 2's premise**: the typed-decision primitive is worth having in the language — same accuracy as an LLM classifier at ~1/30 the latency, with a confidence that *did* separate the clean miss from the hits.
- It does **not** license an acting lane-router yet: the label set is 20, skewed, and partly wrong; fix the labels (re-route the four docs above against §4 as written, or sharpen §4) before measuring again.
- Concrete follow-ups, in order: (1) a one-paragraph clarification in PROGRAM.md §4 that "extension" means *above the motoko core and outside the AILANG substrate*, with these four docs as the worked counter-examples; (2) re-run this shadow after the labels are corrected; (3) only then pick the first acting consumer.
