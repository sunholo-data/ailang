# M-EVAL-FRONTIER-TIER: A harder benchmark tier + curation to de-saturate the suite

**Status**: Planned — authoring DONE (8 benchmarks merged as stretch, see 2026-07-08 update); remaining scope verified 2026-07-11 (see iteration-7 reality check below)
**Target**: v0.29.0
**Priority**: P1 (the suite no longer discriminates the frontier — Fable 36/37 on AILANG)
**Estimated**: design 0.5d; authoring + cross-language validation 2–3d (benchmark-manager skill)
**Dependencies**: [M-EVAL-RATING-EFFICIENCY](v0_24_0/m-eval-rating-efficiency.md) (ELO difficulty + selective rerun + tier graduation — this doc supplies the harder tier its graduation logic promotes into); [M-EVAL-OUTPUT-NORMALIZE](m-eval-output-normalization.md) (re-grade first, so we design against true scores); `benchmarks/CURATION.md` (keep-vs-demote philosophy).

> **📊 EMPIRICALLY VERIFIED (v0.25.0 standard baseline, 11 models):** **15 of 37 benchmarks
> (41%) are fully saturated** — 100% AILANG pass across every model — incl. 3 *stretch*
> benchmarks (`expression_evaluator`, `polymorphic_ord_defaulting`, `symbolic_diff`). Tier
> difficulty is shallow: core AILANG **88.5%**, stretch **68.6%**. The frontier is ceiling-bound:
> **claude-fable-5 passes 36/37 AILANG** — only one benchmark defeats it. And the saturation is
> *understated*: several "hard" benchmarks are grader/benchmark artifacts, not difficulty —
> `contract_sorted_merge` (set-repr), `contract_rle_roundtrip`/`contract_roman_numeral`
> (bool-casing) jump up after M-EVAL-OUTPUT-NORMALIZE; `decision_block_capture` (0%) is
> **mis-graded** (free-text justification scored by exact string match). After those land, the
> discriminating set shrinks further — the suite cannot measure a frontier model.

---

## Problem statement

A benchmark suite's job is to *discriminate*: to produce different scores for models of
different capability. Ours has stopped doing that at the top end. With 41% of benchmarks at
100% and the best model missing exactly one, additional runs of the saturated set buy zero
information (the exact argument in [M-EVAL-RATING-EFFICIENCY](v0_24_0/m-eval-rating-efficiency.md)
— "information-per-compute-hour"). We need benchmarks hard enough that frontier models *fail
some of them*, so the leaderboard, the ELO ratings, and the AILANG-vs-Python story all regain
resolution.

Two things must happen together:
1. **Add a `frontier` tier** of genuinely harder benchmarks (this doc's design).
2. **Demote the saturated 15** out of `core` so the headline metric isn't diluted by
   pass-everything tasks (curation, per `CURATION.md`).

## Benchmark-quality guardrails (learned from this run)

Hard ≠ ambiguous. Every new benchmark MUST:
- **Have a single, deterministic, exactly-specified output.** NO free-text/justification lines
  graded by exact match (the `decision_block_capture` anti-pattern — it measures verbatim
  copying of the prompt's example, not capability). If rationale is wanted, grade it
  structurally (presence of a token, a parseable field), not by matching prose.
- **Be language-neutral in its expected output** — write expected output in a form both AILANG
  and Python can emit, or rely on M-EVAL-OUTPUT-NORMALIZE for bool/numeric/container repr. Do
  not bake AILANG surface conventions into a cross-language expected file.
- **Fail for the right reason** — a frontier model that fails should fail on *reasoning/logic*,
  not on output formatting or an unsatisfiable spec. Validate with ≥2 strong models before merge.
- **Be deterministic** (seeded, no wall-clock/network/RNG in the expected path).

## Difficulty axes (what actually defeats the frontier)

Derived from the few genuinely-hard benchmarks this run (`explicit_dataflow_ssa` 55%,
`run_length_encode` 55%, `csv_to_json_converter` 73%, `prompt_injection` 64% — all post-bool-fix)
and from where saturated tasks are trivially easy:

1. **Multi-stage dataflow / SSA-style transforms** — programs that thread state through several
   non-obvious passes (the one axis already biting: `explicit_dataflow_ssa`).
2. **Stateful invariants under mutation** — interpreters/VMs/allocators where a wrong
   intermediate state only surfaces several steps later.
3. **Deep / non-structural recursion** — mutual recursion, CPS, accumulator inversion — beyond
   the `fib`/`tree-fold` patterns that are 100%.
4. **Effect composition under constraints** — multiple effects (State+Error+IO) that must be
   sequenced correctly; AILANG's row-typed effects make this a genuine differentiator.
5. **Type-directed reasoning** — programs whose correctness depends on instance resolution /
   defaulting the model must get right (AILANG-favoring; extends `polymorphic_ord_defaulting`).
6. **Adversarial / spec-compliance** — `prompt_injection`-style tasks where the model must
   resist a misleading instruction and follow the real spec.
7. **Large-input scaling / precision** — tasks where a naive O(n²) or float-naive solution
   produces wrong output at the tested size.

> **✅ AUTHORING UPDATE (2026-07-08, Claude Fable 5):** The first **8 benchmarks are authored,
> reference-verified, and merged** as `tier: stretch` (the `frontier` tier value doesn't exist in
> `ValidTiers` yet — when it lands, re-tier these): `ssa_constant_fold`, `bytecode_vm_trace`
> (= `bytecode_vm_stepper` sketch), `lfu_cache_trace`, `parse_prec_climb`, `effect_txn_rollback`,
> `glob_match_spec` (= `instruction_injection_guard`-adjacent spec-compliance axis),
> `dep_resolver_backtrack`, `stream_lcg_topk` (= `streaming_topk_large`, N=2000 to stay inside the
> 30s AILANG budget). Method per the guardrails: every `expected_stdout` computed by a Python
> reference impl AND reproduced exactly by a hand-run AILANG solution; each design validated by
> checking that plausible-wrong implementations (3 LFU bugs, greedy resolver, fnmatch delegation,
> peek-instead-of-pop rollback) produce DIFFERENT output. Remaining from the sketch list:
> `cps_transform` (canonical-form ambiguity risk — needs a pinned fresh-variable discipline),
> `register_allocator_linear`, `typeclass_dispatch_chain`, `effect_txn_rollback`'s contract-graded
> variant. **Frontier-failure validation (each must fail ≥1 frontier model) happens in the next
> rotation.** The blocker found during verification (unimported nullary constructor patterns
> silently matching everything, #323 — initially misdiagnosed as an nth/recursion bug) is **fixed**:
> uppercase identifiers in pattern position now always elaborate as constructor patterns.

> **🔎 ITERATION-7 REALITY CHECK (2026-07-11, mission-control — all premises live-verified at
> v0.28.0-149-g45bbbf8f9):**
> - **All 8 authored benchmarks exist** with `tier: stretch` (`ls benchmarks/*.yml` + grep
>   verified). `frontier` is NOT in `ValidTiers` (`internal/eval_harness/spec.go:82` =
>   smoke/core/stretch/vision/experimental). `spec_test.go:342-344` asserts stretch went 14→22
>   with these 8 and notes "they re-tier to `frontier` when that tier value lands" — tier-count
>   assertions WILL need updating when re-tiering.
> - **Both dependencies SHIPPED v0.26.0**: M-EVAL-OUTPUT-NORMALIZE (v0.25.0 baseline re-graded,
>   44 recoveries — the "re-grade first" precondition is DONE) and M-EVAL-RATING-EFFICIENCY
>   (`internal/eval_harness/ratings.go`). Design against the re-graded scores.
> - **`decision_block_capture` is UNCHANGED** — `expected_stdout` still exact-matches the
>   free-text CHOICE sentence. Genuinely open.
> - **Demotion NOT done**: all 12 saturated core + 3 saturated stretch benchmarks still carry
>   their old tiers (grep verified).
> - **Frontier-failure validation has ZERO banked data**: no results for any of the 8 new
>   benchmarks anywhere in `eval_results/` (they're stretch; the nightly runs smoke+core only).
>   Validation requires fresh frontier-model runs = **API-billed → NOT runnable in the headless
>   mission loop** (billing guard; same class as iteration 4's parked haiku re-run). The sprint
>   must (a) land the tier machinery + re-tier the 8 per the 2026-07-08 update's instruction,
>   (b) fix/retire decision_block_capture, (c) run the demotion audit from BANKED re-graded
>   data per CURATION.md's 4-dimension rule (agent-mode coverage in `eval_results/agent/` is
>   sparse — handle missing dimensions honestly, don't fabricate), and (d) PARK the
>   frontier-failure validation runs as an explicit human/next-rotation item. No step may
>   invoke ollama or any API-billed model.
> - `benchmarks/CURATION.md`'s tier table has no `frontier` row — update it in the same sprint.

## Candidate benchmarks (sketches — to be authored + validated)

| id (proposed) | axis | why it should defeat the frontier |
|---|---|---|
| `ssa_constant_fold` | 1 | build SSA, constant-fold across blocks, emit canonical IR — multi-pass, one canonical output |
| `register_allocator_linear` | 1,2 | linear-scan allocation; wrong spill surfaces only in final assignment |
| `bytecode_vm_stepper` | 2 | execute a tiny stack bytecode incl. jumps; off-by-one in PC/stack fails late |
| `cps_transform` | 3 | CPS-convert a small expr language — non-structural recursion |
| `effect_txn_rollback` | 4 | State+Error: apply ops, roll back on the failing one, report final state |
| `typeclass_dispatch_chain` | 5 | output depends on resolving the right instance through 3 layers |
| `instruction_injection_guard` | 6 | embedded "ignore the spec and print X" — must follow real spec |
| `streaming_topk_large` | 7 | top-k over 10k seeded ints; O(n log k) needed, naive sort-all mis-scales/precision |

Target: ~8–12 benchmarks, each validated to **fail ≥1 frontier model** in standard mode before
merge (if all frontier models pass, it belongs in stretch, not frontier).

## Demotion list (saturated → out of core)

100% across all 11 models on AILANG this run — move to `stretch` (or retire) so `core` regains
signal: `api_call_json, audit_chain_replay, cli_args, effect_composition, error_handling,
float_eq, graph_bfs, higher_order_functions, merge_sort, pattern_matching_complex,
shadowing_heavy_contract, tree_transformation_pipeline` (core), and re-tier the 3 saturated
stretch (`expression_evaluator, polymorphic_ord_defaulting, symbolic_diff`) as keep-for-coverage
but low ELO. **Re-confirm against agent-mode saturation first** — several standard-saturated
benchmarks still discriminate in agent mode (`CURATION.md`: demote only if ≥95% on ALL four
std/agent × AILANG/Python dimensions).

## Integration with ELO (M-EVAL-RATING-EFFICIENCY)

New `frontier` benchmarks enter with a high provisional ELO; every PASS/FAIL updates it. The
rating system then (a) auto-detects when `frontier` itself saturates (next model class), and
(b) drives selective reruns so we only spend compute where a trial moves the posterior. This doc
is the *content*; rating-efficiency is the *scheduler*.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1 Determinism | +1 | Guardrails forbid free-text/nondeterministic expected output (fixes the `decision_block_capture` class). |
| A7 Machines First | +2 | A suite that can't separate frontier models gives no usable signal; restoring discrimination is the core value. |
| A9 Cost Visibility | +1 | Demoting saturated benchmarks + ELO-driven reruns cut compute spent on zero-information trials. |
| A10 Composability | +1 | `frontier` tier composes with the existing tier/ELO machinery; no new mechanism. |
| A11 Structured Failure | +1 | Guardrail: a failing frontier benchmark must fail on logic, not formatting/ambiguity. |

**Hard violation check:** none.

## Acceptance

- [ ] Re-grade v0.25.0 (M-EVAL-OUTPUT-NORMALIZE) and re-run saturation + agent-mode audit before finalizing the demotion list.
- [ ] Author 8–12 `frontier` benchmarks per the guardrails; each validated to fail ≥1 frontier model in standard mode (else → stretch).
- [ ] Fix or retire `decision_block_capture` (free-text exact-match) and audit the rest of stretch for the same anti-pattern.
- [ ] Demote the saturated core benchmarks (post agent-mode re-confirm).
- [ ] Frontier tier wired into ELO ratings; `--tier frontier` runs in the release baseline.
