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
