---
sprint_id: M-MOTOKO-SUITE-BOUND-DERIVATION
iteration: 37
round: 1
evaluator: codex:gpt-6-astra
executor: codex:gpt-5.6-sol
same_provider_fallback: true
result: pass
score: 84/100
evaluated_at: 2026-09-06T16:00:09Z
worktree: /Users/voightkampff/dev/sunholo-data/.wt-motoko-iter37-eval
head_sha: a2525c6af5aeb7c1f7df976de5dfcfd2de03680d
residual_base: 0b7f3e3afbdaee91ead44fe8a555661f95719c04
design_doc: design_docs/planned/m-motoko-suite-bound-derivation.md
sprint_plan: design_docs/planned/m-motoko-suite-bound-derivation-sprint-plan.md
---

# Independent evaluation: iteration 37, residual M2 and M3

**PASS — 84/100**, for the scoped implementation. The ordinary and forced-200 suites independently passed all 59 arms. An M2-specific literal mutation and three M3-specific mutations failed for their intended reasons. The required delayless-control/factor-one pair discriminated the ceiling, but the delayless control **hung on its first attempt** and passed on its second. This is evidence that the checks discriminate on this host, not evidence that delayless execution is reliable or that the existing discovery hang is fixed.

**SAME-PROVIDER FALLBACK EXCEPTION: YES.** This final evaluator was independently spawned and declared/pinned to `codex:gpt-6-astra`; the executor was `codex:gpt-5.6-sol`. Model and agent-context separation hold. Provider separation does not: both are Codex/OpenAI lanes. This declared exception is preserved, not presented as a cross-provider evaluation.

## Fallback history and independent adjudication

The following runner artifacts were read directly, and NDJSON event types were independently counted with `jq -r '.type' ... | sort | uniq -c`:

| Lane | Typed result | Observed completion evidence | Treatment here |
|---|---|---|---|
| Primary `pi:ollama/minimax-m3:cloud` | `wall_timeout`, rc 13; pi rc 143; 1,806 seconds | 233 completed tool executions, **0 `agent_end`** | No verdict. It cannot count as an evaluation pass. |
| Next `pi:openrouter/minimax/minimax-m3` | `empty_worktree`, rc 10; pi rc 0; 730 seconds | 66 completed tool executions, **1 `agent_end`**; substantive PASS 90/100 report | Independent supporting evidence, **not an accepted typed lane verdict**. |
| Final `codex:gpt-6-astra` | This report | New independent commands and temporary-copy mutations below | Final declared same-provider fallback evaluation. |

Typed artifacts: `/tmp/motoko_iter37_eval_round1.verdict.json` and `/tmp/motoko_iter37_eval_round1_fallback.verdict.json`, with their corresponding `.ndjson` files. The second runner saw zero changed files because the controller required the report under ignored `.ailang/`. `git check-ignore -v .ailang/state/evaluations/eval_M_MOTOKO_SUITE_BOUND_DERIVATION_iter37_round_1.md` independently identified `.gitignore:82:.ailang/`. The report was read in full; its substantive findings were not rubber-stamped or silently discarded.

Corrections to that report matter: final ShellCheck has **11** informational findings versus **9** at the M1 baseline, including two additional SC2031 sites on the new proxy calls; the two implementation commit diffs each touch only the suite, not two files; the raw final node-reference count is **5**, not 4; and the tracked sprint artifacts remain incomplete. This evaluation also observed the delayless-control hang that its report did not observe. Its PASS 90/100 is retained as historical evidence and is not this report's score.

## Scope, diff review, and immutable dependency

Read `CLAUDE.md`, `AGENTS.md`, the complete sprint-evaluator skill and scoring resources, the complete design and sprint plan, the tracked root sprint JSON, and both exhaustive diffs:

- M2: `git diff 2600b0103..9b9ebd4c6` — suite only, 52 insertions / 13 deletions.
- M3: `git diff 9b9ebd4c6..a2525c6af` — suite only, 19 insertions / 8 deletions.
- `git diff --name-status 0b7f3e3afbdaee91ead44fe8a555661f95719c04..HEAD` — exactly `design_docs/planned/m-motoko-suite-bound-derivation-sprint-plan.md` and `tools/eval/test_motoko_connection_probe.sh`. The plan refresh is commit `2600b0103`; M2 and M3 are separate subsequent commits.

The production probe's SHA-256 is exactly:

```text
f0b5e02493369099f123c42107850fe062bf60d56ccabb2a7e4690d654aabc99
```

`git diff --quiet` for the production probe returned 0 against both the design base `087fbea631a0b80556baa034b499fbdae33e76d2` and the residual base. The same residual-base check on the suite returned 1, the nonempty-diff positive control. Final suite SHA-256 is `1cd3b4a0c641bb8b8b1fb6d31af1119e5d49a36e802c44f5458760f5191257ac`. Both hashes and HEAD were rechecked after the mutation runs. `git diff --check` on the residual delta returned 0.

The evaluator made no Git writes, did not enter the sprint/main worktrees, and did not modify tracked implementation. All mutants and run logs are under `/tmp/motoko-iter37-astra.gFoMA9/`. The sole new worktree artifact is this Git-visible, non-ignored report; it remains unstaged for controller handling, as required by the no-Git-writes instruction.

## Independent command evidence

Host: Darwin arm64; `/bin/bash` is GNU Bash 3.2.57(1)-release. Initial load averages were 2.91 / 2.03 / 1.91. Each full-suite invocation used `/bin/bash /tmp/motoko-iter37-astra.gFoMA9/run.sh LABEL 300 ...`, implementing the plan's completion-artifact/deadline runner. Derivation-only commands had 30-second outer bounds. Full-suite cases ran serially; temporary copies used `PROBE_UNDER_TEST="$PWD/tools/eval/motoko_connection_probe.sh"`. No deadline was extended and no test was retried more than twice.

| Check / log directory | Command after runner prefix | Result |
|---|---|---|
| Syntax | `/bin/bash -n` on suite, production probe, and each mutant | All rc 0. |
| `ordinary` | `/bin/bash tools/eval/test_motoko_connection_probe.sh` | rc 0, **59 ok / 0 not ok**, 68 seconds, exactly one `PASS: 59 probe self-test arms ran`. |
| `forced200` | `env PROBE_SELFTEST_FORK_RATE=200 /bin/bash tools/eval/test_motoko_connection_probe.sh` | rc 0, **59 ok / 0 not ok**, 66 seconds, exact PASS59. |
| `floor99` | `env PROBE_SELFTEST_FORK_RATE=99 PROBE_SELFTEST_DERIVATION_ONLY=1 ...` | rc 1; exactly one floor refusal; **0 derived lines / 0 disabled-floor lines**. |
| `floor100` | Same, rate 100 | rc 0; exactly one diagnostic, scale 4 / cap 480 / ceiling 1600 / enforced. |
| `floor-disabled99` | Rate 99 plus `PROBE_SELFTEST_BOUND_FLOOR_ENFORCED=0`, derivation only | rc 0; exactly one loud disabled-floor line and one diagnostic; scale 4 / ceiling 1584 / DISABLED. |
| `proxy-high` | Rates 800 and 100, derivation only | rc 1; exactly one `instrument failure, not a verdict: observed proxy spread 8.00 exceeds 4.70`; **0 derived lines**. |
| `proxy-low` | Rates 400 and 100, derivation only | rc 0; exactly one diagnostic with `p_obs=4.00`, scale 1, cap 120, ceiling 6400, enforced. |
| `proxy-equal` | Rates 470 and 100, derivation only | rc 0; exactly one diagnostic with `p_obs=4.70`; equality is accepted. |
| `suite-scope` | `env PROBE_MAX_TREE_NODES=50000 /bin/bash tools/eval/test_motoko_connection_probe.sh` | rc 1, 57 ok / 1 not ok, 66 seconds; exact suite-scope refusal below. |

Ordinary diagnostic and bookend, each exactly once:

```text
# bound derivation: r=638/s r_real=614/s p_obs=1.04 reference=400/s scale=1 arm_cap=120s node_ceiling=10208 floor=enforced
# bound drift: end-of-suite fork rate 685/s scale_end=1 scale_used=1 drift=none
```

Forced-200 diagnostic and bookend, each exactly once:

```text
# bound derivation: r=200/s r_real=200/s p_obs=1.00 reference=400/s scale=2 arm_cap=240s node_ceiling=3200 floor=enforced
# bound drift: end-of-suite fork rate 200/s scale_end=2 scale_used=2 drift=none
```

Exact floor and scope refusals:

```text
not ok - bound derivation: fork rate 99/s needs scale 5 > 4 (floor 100/s); host too slow to hold the ratio inside the CI budget; instrument failure, not a verdict
not ok - PROBE_MAX_TREE_NODES is set at suite scope; the ceiling override must stay on arm env lines
```

The ordinary and forced suites both pass arm 26's literal `exceeded 1s sampling deadline` expectation and the production run_lane fixture's `exceeded 2s sampling deadline` expectation. Code review confirms the five timeout literals remain `0, 2, 1, 1, 60`; the node=3 must-fire pin is unchanged. Those refusal texts are checked inside the suite's captured arm logs; an empty outer stderr is not used as a vacuous proof that the messages occurred.

### M2-specific non-vacuity and census

Raw final counts, independently obtained with `rg -c`, are timeout literals **5**, `bound_secs ` matches **12**, node literals **1**, node references **5**. The 12 matches include the census search itself. The **11 actual capacity consumers** are the two cleanup deadlines (lines 61/71), run_bounded grace (123), arm cap (352), four lane caps (535/583/596/609), elapsed ceiling (718), readiness cap (773), and outer margin (774).

The floor flip and first consumers are in the same M2 commit. The lane-order gate compares the computed lane deadline with the arm cap at lines 690–694. At forced rate 100 the independently measured derivation is cap 480; the unchanged expression `ARM_CAP_SECS + 30` yields lane deadline 510. The k=4 ordering is code/arithmetic verification here, not a claim of an additional independently executed full k=4 suite.

**M2 mutation red set:** `/tmp/motoko-iter37-astra.gFoMA9/m2-literal.sh` changes only the first `run_live` capacity assignment from `PROBE_TIMEOUT_SECS="$(bound_secs 4)"` to `PROBE_TIMEOUT_SECS=4`. Bash syntax remains valid. Forced rate 200 produced rc 1, 58 ok / 1 not ok, in 69 seconds. Its only suite failure was:

```text
not ok - wall-clock literal census drift (timeout_literals=6 expected=5 bound_secs=11 minimum=8 node_literals=1 expected=1 node_references=5 minimum=3)
```

The unchanged forced-200 PASS59 is its green control. The positive `bound_secs=11` and node-reference count show that the mutation was rejected for literal drift while the corpus remained populated. This is M2-specific evidence, not borrowed from M1's helper-failure mutations.

### M3-specific non-vacuity and complete retry record

The delayless copy changes only `PROBE_TEST_PGREP_LOOP_DELAY=1` to `0` on the discovery arm. The factor-one copy differs from that control by exactly one line, `NODE_CEILING_FACTOR=16` to `1`, independently verified with `git diff --no-index`. Both use forced fork and real-op rates 200.

| Variant / log directory | Outcome | Interpretation |
|---|---|---|
| `delayless`, attempt 1 | **rc 1, 32 ok / 1 not ok, 287 seconds**; `not ok - descendant discovery refuses on the real wall-clock deadline exceeded its 240s arm cap` | Failed green control. During the stall, the marker file had 257 lines and no discovery-refusal text had appeared. Cause is unestablished. This result is retained. |
| `delayless-attempt2`, same file and env | **rc 0, 59 ok / 0 not ok, 67 seconds**, exact PASS59; `# discovery walk marker_count=230 node_ceiling=3200` | Successful wall-clock control; 230 is in [1,799]. Stop after two attempts; no third was run. |
| `factor-one` | **rc 1, 32 ok / 1 not ok, 35 seconds** | Exact node refusal below, not an arm-cap failure or later floor-diagnostic mismatch. |
| `m3-node-literal` | **rc 1, 58 ok / 1 not ok, 68 seconds** | Restore only arm-scoped `PROBE_MAX_TREE_NODES=50000`; numeric node census moves 1 to 2. |
| `m3-census-zero` | **rc 1, 58 ok / 1 not ok, 63 seconds** | Corrupt only numeric-node grep token to `PROBE_MAX_TREE_NODES_ZZZ=`; zero is refused beside positive other counters. |

**M3 mutation red sets**, preserved separately:

```text
# factor-one: one exact node-refusal line, plus the suite's expected-message failure
not ok - descendant discovery refuses on the real wall-clock deadline lacked expected message: process-tree discovery deadline expired (wall clock)
process-tree discovery exceeded 200 nodes

# m3-node-literal
not ok - wall-clock literal census drift (timeout_literals=5 expected=5 bound_secs=12 minimum=8 node_literals=2 expected=1 node_references=5 minimum=3)

# m3-census-zero
not ok - wall-clock literal census matched nothing (timeout_literals=5 bound_secs=12 node_literals=0 node_references=4); instrument failure, not a verdict
```

The passing delayless control and exact factor-one red establish the required local M3 discrimination. The failed first control prevents any claim of stable delayless operation. The production delay remains 1, and the design explicitly leaves the pre-existing discovery hang outside this sprint.

## Loaded comparison, artifacts, and limitations

The executor's load experiment was **not rerun** by this evaluator. Its existing artifacts were independently inspected at `/var/folders/kr/h0wr2sj94vd6ljtmsxv8jkt00000gn/T/motoko-M2-load.FJmx9g`: summary/done and all 20 output files. They show 10 completed pairs, 20 PASS59 lines, derived reds 0, control reds 0, proxy refusals 0, completion rc 0, and recorded spinner survivors 0. Derived startup rates range from 140 to 205/s (scale 2 or 3); each control forces 800/s (scale 1). Recorded load averages reached roughly 72.5. This is **UNINFORMATIVE** about the claimed advantage because no control red occurred. Inspection validates what the artifacts contain; it does not turn executor-generated runs or historical spinner cleanup into new evaluator executions.

The controller reports verifying executable, distinct M1/M2/M3 snapshots against their commits. Those snapshots reside only in the sprint worktree, which this evaluator was instructed not to enter; they are declared controller provenance, not independently rechecked snapshot evidence. M1's already-accepted CI observation is recorded in the plan (`r=318`, `r_real=251`, `p_obs=1.27`, scale 2, floor DISABLED). This evaluator does not claim to have rerun or fetched that remote job, or the final branch's remote CI.

Additional limits:

- Scope is shell-test instrumentation. No compiler, runtime, production probe, or AILANG semantics changed. Broad Go `make test`, `make lint`, and coverage were not run. The generic evaluator script was attempted and returned rc 1 before tests because `.ailang/state/sprints/sprint_M-MOTOKO-SUITE-BOUND-DERIVATION.json` is absent; this repository instead has the tracked root `sprint_m-motoko-suite-bound-derivation.json`. The missing adapter input is not reported as a test execution or a green global check.
- Bash syntax and scope-specific suites replace those generic test commands in the rubric below. `shellcheck --severity=warning` returns 0. Full informational ShellCheck output is not clean: final 11 findings versus baseline 9, all SC2016/SC2030/SC2031. The two extra SC2031 findings at lines 1076/1082 arise on new proxy calls from the existing subshell-local `ARM_CAP_SECS` pattern; neither is a runtime warning/error. No new TODO/HACK/FIXME markers were found.
- Live loopback observations remain explicitly `UNINFORMATIVE UNDER SANDBOX`; fixture-backed results are the evidence.
- The plan correctly narrows `p_obs` to a contemporaneous symmetric throughput ratio for two Bash executable classes. It does **not** measure the two-load-condition degradation-factor ratio `P_PROXY`, or heterogeneity of real OS `pgrep`/`date`/`lsof`. This semantic limitation remains.
- The new proxy arm is placed before the old drift/helper-failure arms (lines 1075–1091), despite the plan's explicit instruction to place new residual arms after M1's measurement-failure arms (which end at 1136). All pre-existing timing-sensitive work still precedes it, so the placement purpose is preserved; literal plan conformance is not complete.
- The tracked sprint JSON lacks `passes`, completion timestamps, notes, overall completed status, and velocity. The plan has no completed milestone markings; the design still says `PLANNED (design, not yet started)`. No artifact statuses were modified by this evaluator.

## Full 100-point rubric

This is an explicitly scope-adapted application of the sprint-evaluator rubric to the approved one-file Bash-test sprint. Changelog and AILANG example requirements are inapplicable to this internal test-only change; their documentation allocations are retained for scope completeness. No global Go-test or remote-CI success is inferred.

| Category | Maximum | Score | Evidence and deductions |
|---|---:|---:|---|
| Tests Pass | 20 | 20 | Shipped suite ordinary and forced-200 each rc 0 / PASS59; syntax clean. Required mutation discriminators observed. The first temporary delayless-control hang remains an explicit limitation, not a silently retried shipped-suite failure. |
| Lint Clean | 10 | 10 | Bash 3.2 syntax, residual `git diff --check`, and ShellCheck warning/error severity checks all rc 0. Informational findings and their two-site increase are disclosed. |
| Acceptance Criteria | 30 | 30 | M2 floor/default, scaled consumers/fixed pins, census, ordering code, exact forced diagnostic, two-sided proxy; M3 derived arm ceiling, successful delayless control versus exact factor-one red, scope guard, node census, unchanged probe/refusal inventory. Loaded evidence is correctly classified UNINFORMATIVE as the plan requires when control reds are zero. Evidence modality and non-reexecuted checks are explicit above. |
| Code Quality | 15 | 5 | -5: suite is 1,212 lines, above the generic 800-line cap (baseline already 1,162). -3: existing sprint JSON completion fields absent. -2: plan milestones not marked complete. Large pre-existing functions were not refactored; no new long function or TODO introduced. |
| Documentation | 15 | 10 | Changelog/examples are not required for this shell-only internal scope (5+5). -5: design status still says not started; completion bookkeeping remains for controller handling. Plan usefully records corrections and semantic limits. |
| Design Fidelity | 10 | 9 | Implementation follows the corrected design, scoped ceilings, floor flip, and immutable probe. -1 for the explicit residual-arm placement mismatch described above. |
| **Total** | **100** | **84** | **PASS**, threshold 70. |

Conditional regression-surface category does not trigger: no shared compilation infrastructure changed. Conditional performance-profile category does not trigger: this sprint derives/refuses test bounds and makes no production optimization claim. No hard failure is present in the shipped suite; more than half of the acceptance contract is met; M2/M3 implementation commits exist; required per-milestone non-vacuity was established independently. The known delayless control instability and unmet cross-provider separation remain visible qualifications to this scoped pass.

The evaluator stops here: no fixes, status rewrites, design movement, commit, push, or controller action.

EVALUATION_RESULT: pass
EVALUATION_SCORE: 84/100
EVALUATION_ROUND: 1
EVALUATION_REPORT_PATH: docs/sprint-retros/motoko-iter37-evaluation-round1.md
FEEDBACK_SUMMARY: PASS with independent M2/M3 non-vacuity; delayless control failed once then passed, loaded A/B remains UNINFORMATIVE, completion artifacts are stale, and same-provider fallback is explicitly declared.
