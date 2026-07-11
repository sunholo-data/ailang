# M-EVAL-FRONTIER-TIER — Saturation Demotion Audit

**Date**: 2026-07-11
**Author**: sprint-executor (claude-opus-4-8), headless mission loop
**Data source**: banked re-graded baseline `eval_results/baselines/v0.25.0`
(standard: 814 result files / 11 cloud models; agent: 444 result files / 6 models).
**No new eval runs** — this audit is computed entirely from banked data
(re-grade precondition shipped in v0.26.0 via M-EVAL-OUTPUT-NORMALIZE).

## Method

A trial passes iff `compile_ok && runtime_ok && stdout_ok`. For each demotion
candidate the pass rate was computed for each of the four dimensions:

1. **std-ail** — standard mode, AILANG
2. **std-py**  — standard mode, Python
3. **ag-ail**  — agent mode, AILANG
4. **ag-py**   — agent mode, Python

Per `benchmarks/CURATION.md` §5 *Saturation demotion (the 4-dimension rule)*:
demote a **core** benchmark to **stretch** only if it is **≥ 95% on ALL FOUR
present dimensions**. Never demote on a missing dimension. Where standard and
agent modes **disagree** on saturation, **keep** (record the disagreement). A
saturated **stretch** benchmark is **kept-for-coverage** (low-ELO), not retired.

**Panel-asymmetry honesty note**: the standard panel is 11 models; the agent
panel is 6 (`claude-sonnet-4-6`, `gpt5-4-mini`, `opencode-or-{deepseek-v4-flash,
deepseek-v4-pro, glm-5-1, minimax-m3}`). A 100% agent verdict rests on fewer
models than a 100% standard verdict — weaker evidence, but present for all 15
candidates (no dimension was missing in this baseline, refuting the a-priori
"agent data is sparse" concern for these specific benchmarks).

## Results

Cell format: `passed/total = pass% (n models)`.

| Benchmark                     | std-ail       | std-py        | ag-ail       | ag-py        | Decision |
|-------------------------------|---------------|---------------|--------------|--------------|----------|
| api_call_json                 | 11/11=100%    | 10/11=91%     | 5/6=83%      | 6/6=100%     | **KEEP** — std/agent disagree (ag-ail 83%, std-py 91%) |
| audit_chain_replay            | 11/11=100%    | 11/11=100%    | 6/6=100%     | 6/6=100%     | **DEMOTE** core→stretch |
| cli_args                      | 11/11=100%    | 11/11=100%    | 4/6=67%      | 0/6=0%       | **KEEP** — strong agent-mode discriminator (ag-py 0%) |
| effect_composition            | 11/11=100%    | 11/11=100%    | 6/6=100%     | 6/6=100%     | **DEMOTE** core→stretch |
| error_handling                | 11/11=100%    | 10/11=91%     | 5/6=83%      | 6/6=100%     | **KEEP** — std/agent disagree |
| float_eq                      | 11/11=100%    | 11/11=100%    | 6/6=100%     | 6/6=100%     | **DEMOTE** core→stretch |
| graph_bfs                     | 11/11=100%    | 11/11=100%    | 6/6=100%     | 6/6=100%     | **DEMOTE** core→stretch |
| higher_order_functions        | 11/11=100%    | 10/11=91%     | 6/6=100%     | 6/6=100%     | **KEEP** — std-py 91% (not saturated on all 4) |
| merge_sort                    | 11/11=100%    | 11/11=100%    | 6/6=100%     | 6/6=100%     | **DEMOTE** core→stretch |
| pattern_matching_complex      | 11/11=100%    | 11/11=100%    | 6/6=100%     | 6/6=100%     | **DEMOTE** core→stretch |
| shadowing_heavy_contract      | 11/11=100%    | 11/11=100%    | 6/6=100%     | 6/6=100%     | **DEMOTE** core→stretch |
| tree_transformation_pipeline  | 11/11=100%    | 7/11=64%      | 6/6=100%     | 6/6=100%     | **KEEP** — std-py 64% (not saturated on all 4) |
| expression_evaluator (stretch)| 11/11=100%    | 8/11=73%      | 6/6=100%     | 6/6=100%     | **KEEP** — std-py 73% (not saturated) |
| polymorphic_ord_defaulting (stretch) | 11/11=100% | 8/11=73%   | 5/6=83%      | 6/6=100%     | **KEEP** — std/agent disagree + std-py 73% |
| symbolic_diff (stretch)       | 11/11=100%    | 11/11=100%    | 6/6=100%     | 6/6=100%     | **KEEP-FOR-COVERAGE** — saturated stretch, retained low-ELO (no tier move) |

## Verdict summary

- **DEMOTE (7, core → stretch)**: `audit_chain_replay`, `effect_composition`,
  `float_eq`, `graph_bfs`, `merge_sort`, `pattern_matching_complex`,
  `shadowing_heavy_contract`. All four dims ≥ 95%, no missing dims, no
  standard/agent disagreement.
- **KEEP-FOR-COVERAGE (1 stretch)**: `symbolic_diff` — saturated on all four but
  is already `stretch`; retained (comment added) rather than retired.
- **KEEP (7)**: `api_call_json`, `cli_args`, `error_handling`,
  `higher_order_functions`, `tree_transformation_pipeline`, `expression_evaluator`,
  `polymorphic_ord_defaulting`. Each fails the 4-dimension rule — the conservative
  rule protected real signal: `cli_args` is a strong **agent-mode** discriminator
  (agent-Python 0%), and several are not Python-saturated in standard mode.

**Note vs the design doc's 12+3 demotion list**: the design doc proposed demoting
all 12 saturated-in-standard core benchmarks. The 4-dimension rule (banked v0.25.0)
demotes only **7** of them — the other 5 (`api_call_json`, `cli_args`,
`error_handling`, `higher_order_functions`, `tree_transformation_pipeline`) still
discriminate on at least one dimension and are kept. This is the intended effect of
re-confirming "against agent-mode saturation first."

## Tier-count impact

core 26 → 19, stretch 22 → 29 (frontier 8, smoke 23, vision 9 unchanged). The
`spec_test.go` tier-distribution assertions were re-centered accordingly.

## Anti-pattern audit (M4 companion)

Scan of all stretch + frontier benchmarks for the free-text-exact-match
anti-pattern (`decision_block_capture` class — prose justification graded by exact
string match): **only `decision_block_capture` is a genuine instance** (fixed via
the `prefix_line` structural grader, M4). The other flagged files
(`contract_sorted_merge`, `docx_reimplement`, `legal_obligation_engine`,
`lfu_cache_trace`) are **false positives** — their expected output is fixed,
deterministic structured data (the prose in `docx_reimplement` /
`legal_obligation_engine` is fixed document content, not model-authored rationale).

## Parked

Frontier-FAILURE validation for the 8 re-tiered frontier benchmarks (each must
fail ≥ 1 frontier model in standard mode) is **PARKED** — it requires fresh
API-billed frontier-model runs, disallowed in the headless mission loop. Tracked
as a follow-up for a human / next paid rotation.
