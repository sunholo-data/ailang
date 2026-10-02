# Sprint Plan: M-INTERPRETER-TAIL-CALL-ELIMINATION

**Status:** Planned; awaiting sprint approval through the coordinator
**Design:** [Approved design](m-interpreter-tail-call-elimination.md)
**Target:** v0.51.1
**Issue:** #1488 (related regression surface: #1317)
**Duration:** 3 days, 18 engineering hours
**Risk:** High — production evaluator application semantics

## Summary

Add a tail-aware function-body evaluator and trampoline to CoreEvaluator so ordinary exact-arity tail calls use constant Go stack. Deliver interpreter/VM parity for the reported stdin loop while preserving non-tail recursion limits, effect obligations, and trace event order.

The approved design is the scope authority. Planning does not authorize implementation; coordinator approval of this plan starts the executor. No implementation was changed during planning.

## Current Status and Velocity

The working tree was clean on coordinator/task-b115fc4d. std/VERSION is v0.51.0. The available git history contains only f1e22cbe, the design-doc commit for #1488; the seven-day velocity script cannot establish implementation LOC/day or completion rate. Its historical changelog samples are not a current velocity baseline. Use the design's 9-hour base estimate doubled to 18 hours, allocated over three six-hour days. Estimated output is 780 LOC (290 implementation, 420 tests, 70 example/docs), or 260 LOC/day of planned capacity, not measured velocity.

Code inspection confirms evalCoreApp still evaluates FunctionValue bodies through evalCore and increments recursionDepth per application. Existing application logic includes arity/curry handling, trace hooks, budget boundaries, Rand modes, contracts, and resolver restoration. The design audits the systemic gap across if, let, letrec, linear match, mutual recursion, and effectful loops; all three implementation phases remain outstanding. Existing coverage was not measured: the coverage-badge target runs the full suite. Capture evaluator coverage before implementation and compare after M3.

## Registry Reuse Audit

`ailang pkg search tail-call` returned no packages (binary also warned it may be stale). No candidates required info/docs inspection. This is internal Go evaluator machinery, not a package-like capability; AILANG packages cannot replace the application path or trace/depth accounting. Decision **none** for M1, M2, and M3; reuse existing evaluator helpers, trace interfaces, and test harnesses. Registry results are supplementary to this architectural reason.

## Milestones

### M1: Tail evaluator and trampoline (~500 LOC)

**Estimate:** 260 implementation + 240 tests; 6 hours, Day 1.
**Dependencies:** None.
**Files:** internal/eval/eval_tail.go (new), eval_operations.go, eval_expressions.go, eval_patterns.go as needed for shared binding/match helpers, and eval_tail_test.go (new).
**Example:** Specify examples/runnable/tail_read_loop.ail for M3; this milestone uses direct Core AST tests without adding .ail code.

Implement the private marker, five-field obligation predicate, and tail evaluation through App/If/Let/LetRec/linear Match. Share LetRec binding machinery where practical. Separate callee/argument evaluation from application so a fallback never evaluates an effectful callee or argument twice. Preserve left-to-right order, IndirectValue forcing, exact arity, and curry/builtin/constructor behavior.

Introduce a trampoline only for eligible functions; obligated source functions and obligated callees keep the legacy recursive behavior. Keep the recursion guard once per trampoline entry. Retain per-iteration environment/resolver restoration and effect boundary lifetimes. Allocate pending trace names only when function-call recording is enabled; untraced runs must retain no growing call list. Enter events occur per invocation and exits unwind in reverse order with the final value; errors emit no pending exits.

- [ ] If, let+if, letrec-body and linear match loops complete at least 1,000 iterations at depth 100; mutual recursion completes 20,000 iterations.
- [ ] Non-tail operands, conditions, bindings, scrutinees and guards still count nested calls; RT_REC_003 fires at the configured limit.
- [ ] Exact-arity/IndirectValue calls work; partial/over-application, builtin and constructor fallbacks preserve results and evaluate effects exactly once.
- [ ] Each of Preconditions, Postconditions, EffectBudgets, EffectMinBudgets and EffectRandMode independently excludes TCO; test obligated callers as well as callees.
- [ ] Marker cannot escape into returned values, show or equality; DTree match takes the documented ordinary path.
- [ ] Environment, resolver and depth/segment counters restore after success and error; focused evaluator tests pass.

**Risk:** Shared application restructuring can change dynamic scopes. Mitigate with mixed eligible/obligated call chains, cross-module resolver fixtures and existing stack-hop tests. If preserving an obligation requires changing the approved semantics, return to design review.

### M2: Behavior pins, trace compatibility and diagnostics (~140 LOC)

**Estimate:** 30 implementation + 110 tests; 6 hours, Day 2.
**Dependencies:** M1.
**Files:** internal/eval/recursion_test.go, rt_rec_003_message_test.go, recursion_limit_error.go, eval_tail_test.go and existing typed trace tests as needed.
**Examples:** Reuse examples/inline_tests_recursive.ail and examples/runnable/recursion_mutual.ail.

Before changing behavior, bank a deterministic short-call trace and shallow fib benchmark baseline from the original evaluator. Replace the infinite-tail TestStackOverflow fixture with low-limit non-tail recursion to prevent hanging the suite. Add a finite 10,000+ iteration tail pin at limit 100. Propose RT_REC_003 advice: “rewrite so the recursive call is in tail position”; retain existing working remedies and max-depth flag guidance. Exact final diagnostic wording remains subject to review, as the design requires.

- [ ] TestStackOverflow terminates promptly and proves non-tail RT_REC_003; the finite tail replacement succeeds at limit 100.
- [ ] Diagnostic advice is exercised by rewriting a failing sum to an accumulator tail loop; existing flag-remedy tests still pass.
- [ ] Short self/mutual/mixed-call traces match banked event order, names, arguments and results; trace-ring invariants pass.
- [ ] Error traces omit pending exits and restore evaluator state; disabled tracing retains no pending-name list.
- [ ] Existing contracts, budgets, Rand replay and #1317 stack-hop tests pass with unchanged expectations.

**Risk:** Trace order and budget unwind are observable. Compare full events rather than counts alone; include error paths and nested non-tail calls inside tail iterations.

### M3: CLI parity, scale validation and documentation (~140 LOC)

**Estimate:** 70 tests + 70 example/docs; 6 hours, Day 3.
**Dependencies:** M1, M2.
**Files:** cmd/ailang/tail_recursion_test.go (new; reuse CLI test harness), examples/runnable/tail_read_loop.ail (new), docs/docs/reference/limitations.md, docs/LIMITATIONS.md, docs/docs/reference/implementation-status.md, CHANGELOG.md/changelogs/v0.32-current.md following repository convention. Existing cmd/ailang/deep_recursion_test.go expectations remain unchanged.

Run `ailang prompt` before writing the fixture, then `ailang check`. Feed 20,000 lines and 300,000 lines through both engines at default depth, asserting exact stdout counts. Use subprocess deadlines so an accidental infinite loop fails rather than blocking CI. Measure peak RSS for untraced 20k/300k runs using available platform tooling, recording platform, elapsed time and values; investigate growth attributable to retained calls rather than input/output buffers. Trace-enabled runs intentionally retain pending names and are not claimed constant-memory.

- [ ] The runnable stdin fixture prints 20000 under interpreter and VM with identical stdout, default settings and IO capability.
- [ ] Both engines complete 300,000 lines with output 300000; interpreter stack remains bounded and untraced memory measurements show no retained per-call chain.
- [ ] Run design Conflict Surface fixtures through both supported engines and compare outputs; inline recursive tests and bounded_take_parity_test pass.
- [ ] make test-core, make test, make fmt, make lint and make check-boundaries pass; new tail evaluator branches are covered and evaluator coverage does not regress from baseline.
- [ ] Repeated shallow fib measurements show no reproducible slowdown beyond baseline variability; investigate any consistent regression before completion.
- [ ] Mutation checks independently demonstrate that removing marker construction, disabling obligation exclusions, and counting eliminated calls break their respective tests; restore each mutation immediately.
- [ ] Docs distinguish eligible tail calls from non-tail limits, list obligation/DTree/SimpleEvaluator exclusions, and explain that infinite tail loops no longer hit RT_REC_003.

**Risk:** Full-suite/platform checks may exceed six hours. Keep scale tests bounded and log unavailable downstream or darwin-arm64 validation explicitly; do not claim the external sim/ship.ail session was run without that checkout.

## Day-by-day Schedule

| Day | Work | Hours | Exit condition |
|---|---|---:|---|
| 1 | Capture baselines; implement M1 and focused tests | 6 | Tail shapes flat, fallback and state tests green |
| 2 | Complete obligation/trace cases, M2 pins and diagnostic | 6 | Trace equality and legacy regressions green |
| 3 | M3 CLI/scale sweep, mutations, docs and full checks | 6 | All acceptance evidence recorded |

## Success Metrics and Handoff

Total estimate: **780 LOC**, **18 hours**, **3 days**. Every milestone has executable criteria; prioritize semantic branch coverage over a fabricated global percentage. No VM, bytecode compiler, core AST, parser or type-system changes, new flags, default-limit changes, SimpleEvaluator extension, DTree TCO or fuel budget are planned.

Executor must record commands/results, memory and timing measurements, baseline comparisons, and any unavailable platform/downstream validation. Sprint-evaluator assesses the implementation against the approved design and these criteria after execution. Link development commits with `refs #1488`; #1317 is related regression coverage, not an issue this sprint closes. The coordinator receives the plan and JSON paths for approval and subsequent executor handoff; this planning stage does not start implementation.
