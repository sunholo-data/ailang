# #2007 independent evaluation — PASS 90/100

Integration confirmed: PR #2007 merged normally at `2bfd61bf4446c7e7e7dcc925d34af4bdc793f4fc` after required CI passed. Pending-gate wording below records the earlier evaluation snapshot; one hosted Mac build remained queued at merge.
Sprint M-IFC-FUNCTION-VALUE-SINKS; reviewed head `8771ceae521f58cc5e0b0d65f7bb02f37df09731`, integrated dev `61716789ae8f5ebcea534b1a993ca00d1533ec04`. Reviewer did not implement this sprint or change its source. Independent source acceptance is PASS90/100 using the documented local gate. Full native tests, lint and sizes passed; normal integration still waits the required exact-head CI retry.

## Source and design acceptance

Reviewed the ratified F1–F4 design, six-milestone approved plan, completed sprint JSON, complete production diff, named matrix/CLI/REPL/limit tests, corpus records and mutation logs. Original committed source from 3dd7b89fd was recovered with provenance while dirty historical work remained parked. Historical silent probe/replay/round cutoffs were reproduced failing, then replaced by repeated-state replay and summary-equality convergence. Defensive limits fail closed as ordinary ifc_analysis_limit diagnostics; unrelated panics propagate.

The callable value domain remains independent of HM/CoreTI/iface/codegen. A union of alternatives preserves all demands; sorted forbid sets and positive-cover conjunction implement the ruled strict join. Value lookup checks lexical bindings before module functions, retaining shadowing controls. Values carry demands through the approved binding/container/pattern/result forms. Checks A/C apply to the callee value, and deferred applications replay symbolic actual labels/callables for HOF and wrapper inference together. The concrete label/Check B walk preserves the prior environment. The conservative requirement scan skips symbolic summaries for ordinary unannotated modules without skipping the concrete check.

An independent preflight found and executed a real bypass: capitalized imported Apply was guessed to be a constructor, while identical lowercase apply was rejected. Cache-disabled upper caller checked and returned TAINT. The repaired source removes capitalization guessing, treats only proven local constructors specially, and applies the ratified F2 fallback to all unknown imports. Unit plus two-module CLI/evaluator/VM regressions now reject both names before output; imported Some(gate) storage remains accepted. Initial evidence remains in 1719-independent-preflight-review.md and the reduced source/logs. No inferred imported sink guarantee is claimed.

## Acceptance map

| Milestone | Inspected evidence |
| --- | --- |
| M1 callable algebra | Join associativity/commutativity/idempotence and strictness, sorted multi-forbids, absent versus explicit bottom covers, written alias depth40 |
| M2 value/application | Every R0–R9/R14/C0–C1 carrier, lexical shadowing N4, storage N6, record-update/array/ADT/block-letrec controls; direct diagnostic kind and positions pinned |
| M3 applied summaries | R2/R10/C2 reject; clean N2/N3 pass; actual-label substitution and HOF intrinsic taint; removal mutation kills the intended subset |
| M4 wrapper inference | R11–R13 plus nonredundant R11b, X1 email/secret/clean, recursion/source-order, no labelVar escape; depth3/4/12/40 result chains, depth64 replay and reachable-sink cycle; no partial shipping |
| M5 runtime/regressions | All24 named rows plus X1/R11b in fresh-file CLI and unit tests; REPL and evaluator/VM reject before the unsafe print; three independent overlay mutations fail for missing IFC errors |
| M6 corpus/docs | Initial2625 unchanged snapshots then combined2628 snapshots across11roots,23 pre-existing parse failures, only3 actual new fixture leak diagnostics;18 CLI consumer comparisons unchanged including gmail and leak_lab; true manifest stats and new immutable prompt10 |

Corpus source indices include per-file hashes and repository/dirty-state provenance. Each compiler arm analyzed identical frozen bytes. Existing HM errors cannot hide IFC changes because the corpus harness calls CheckModuleIFC directly; full CLI consumer comparisons are separate. Strict leak_lab module-name mismatches are preserved, with explicit existing module-relax mode used on both arms to reach IFC. No consumer was edited.

Prompt v0.16.10 adds a single same-module teaching sentence and a version header to frozen v0.16.9; old .9 bytes remain SHA6197fe5f4bf1c112737c550e08d557eea5c92db9b85839efaac8d46accd28be1. New mirrors share SHAa9435c79d78002c3c0b51e36bff06b5f918f7e3a90198dd2301e493c0ff9e001. The guide keeps the cross-module limitation and conservative imported fallback explicit. #1771 remains later M1–M2 work with sorted Forbidden[]string/applied reservation in v2; neither #1134 nor #1132 is closed by this sprint.

## Provisional score and gates

Final base90/100: tests20 (complete native full suite passed); lint10 (clean); criteria30; code quality10; documentation10; design fidelity10. Long-function deduction reaches the rubric cap−5 (flowVal, moduleHasIFCRequirements and lambdaVal exceed50lines); design status/archival remains pending−5. No new TODO/HACK is found. Regression surface coverage10/10 is reported separately: filled Conflict Surface, named fixture tests, explicit actual consumers and source-identical corpus differential. This is a security soundness sprint with no performance goal, so profiling category is inapplicable.

Native evaluation uses the documented EVAL_LOCAL_GATES=1 path on a clean exact-head worktree outside TMPDIR, with GOFLAGS=-p=1 package scheduling from the outset. Log1719-native-evaluation.log/session19035. Exact-head PR CI is independent additional evidence. No required check is bypassed; pending checks remain pending. Root owns a separate post-PASS design/companion-plan/report archive, not source changes or self-rescoring.


## Final native gate and integration hold

Full native evaluation completed successfully at2026-10-29T09:26:41Z on clean exact8771ceae5. Parsed machine output confirms tests_pass/lint_clean/file_sizes_ok=true, numeric TODO0, and no hard fails. The separate JSON artifact1719-native-gate.json retains exact command/head.

Initial remote Linux test and Build ubuntu failures both contain only the existing #1926 empty-PID-file race (strict/sigterm and strict/exit). Logs are preserved; one exact-head retry for each workflow is running. These are not reported as passed. Root will use normal merge only after required checks pass. Hosted Mac checks remain independently pending.

EVALUATION_SCORE: 90/100
EVALUATION_ROUND: 1
EVALUATION_REPORT_PATH: .ailang/state/evaluations/eval_M-IFC-FUNCTION-VALUE-SINKS_round_1.json
EVALUATION_VERDICT: PASS score=90
