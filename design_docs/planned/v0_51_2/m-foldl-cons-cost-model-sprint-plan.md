# Sprint Plan: M-FOLDL-CONS-COST-MODEL

## Summary
Correct list-building cost guidance, add pure `std/list.mapAccumL` on both engines, and measure the consing-fold VM overhead before considering an allocation fix.

**Design:** [approved design](m-foldl-cons-cost-model.md)  
**Duration:** 5 working days (40 hours, including approximately 25% contingency over the design's four days).  
**Estimated total:** 800 LOC (approximately 330 implementation, 300 tests/examples/benchmarks, 170 documentation).  
**Risk:** Medium; conditional VM frame reuse is high risk.  
**Status:** Planning complete; implementation not started. Coordinator review/merge controls executor handoff.

## Current status and evidence

Source version is v0.51.0; current work branch is coordinator/task-1c987f6d. The approved design artifact is already present; no inherited uncommitted source changes were found. The design audits interpreter cons, VM OpCons, fold callbacks, array append, iterator capabilities and all teaching surfaces. The flat-list representation and D-19 remain outside this sprint.

The velocity script ran for seven days, but the shallow history exposes only commit 1d0601dc (a design document). Its changelog scan yielded no usable recent LOC metric. A measured LOC/day estimate cannot be inferred: the planning capacity is 160 LOC/day, an estimate rather than observed velocity. The 800 LOC budget includes the conditional VM fix; if profiles justify an account alone, actual code volume will be lower without reducing evidence requirements.

An installed AILANG v0.51.0 b99dd25 dirty binary is now available, unlike the design authoring session. Go, make and jq are absent. It is unsuitable as proof of current-source behavior: AC-0 requires a fresh build on an equipped executor runner. Planning does not claim tests, coverage, syntax checks or timings passed.

The requested pre-planning quorum was attempted on 2026-10-02. It exited 3 because every reviewer was absent (auth, unreachable or quota); no technical verdict was produced. Evidence: `.ailang/state/mission-quorum/m-foldl-cons-cost-model-2026-10-02T16-00-59Z.json`. Planning continues under the explicit handoff instruction that previous work is approved. This is not a quorum pass or an implementation authorization.

## Registry reuse audit

Executed `ailang pkg search mapAccumL` (no packages) and `ailang pkg search list` (a2ui, duckdb, decisions). Inspected `sunholo/a2ui` with `pkg info`: component-tree rendering, unrelated to list runtime primitives. `pkg docs` failed explicitly because the package has no AGENT.md; no documentation result was assumed. All four milestones choose **none**: core teaching surfaces, builtin execution and VM internals cannot acquire this primitive from these packages. Decisions are populated per milestone in the sprint JSON.

## Milestones

### M1: Baseline and cost-model honesty (~120 LOC)

**Duration:** 0.75 days. **Dependencies:** None.

**Files and examples:** Update `std/list.ail`, `internal/eval/recursion_limit_error.go`, `docs/docs/reference/no-loops.md`; create baseline `bench/list_accumulation/consrepro.ail` and measurement notes. Prepare the prompt-manager caveat in the sprint report.

**Acceptance criteria:**

- [ ] Fresh make build binary and N=40000 consrepro baseline recorded for both engines (AC-0); new fixtures type-check before use.
- [ ] std/list header and concat comment, RT_REC_003 and no-loops guide state foldl is O(n) only with an O(1) step; consing a growing accumulator makes the fold O(n²) (AC-1).
- [ ] Prompt-manager note proposes a new prompt version; frozen prompt unchanged; tail-call message advice retained (AC-6).

### M2: Pure mapAccumL on interpreter and VM (~360 LOC)

**Duration:** 1.75 days. **Dependencies:** M1.

**Files and examples:** Update `internal/builtins/list_iterative.go`, `internal/vm/builtins_hof.go`, `internal/bytecode/builtin_names.go`, `internal/vm/builtins.go`, `std/list.ail`; extend adjacent builtin/VM tests and coverage tests as required. Create `examples/runnable/mapAccumL_running_total.ail`.

**Acceptance criteria:**

- [ ] mapAccumL(f,s0,xs) invokes f left-to-right once per element with (state,element), consumes (output,newState), and returns (outputs,finalState); empty input returns ([],s0).
- [ ] Both engines build outputs with one pre-sized output allocation, preserve input and output order, and pass state-threading, polymorphism, error propagation and non-aliasing tests.
- [ ] HOF names and dispatch tables append at matching positions; existing indices preserved; TestPureBuiltinCoverage and HOF tests pass (AC-3).
- [ ] Stdlib and running-total example type-check and produce identical results on interpreter and VM.

### M3: Measure scaling and resolve VM overhead (~240 LOC)

**Duration:** 1.5 days. **Dependencies:** M2.

**Files and examples:** Create `bench/list_accumulation/map_accum_linear.ail`, `run.sh`, `README.md` and reproducible profile artifacts. Modify `internal/vm/vm.go` and `internal/vm/frame.go` only if measured frame allocation warrants reuse; otherwise update `docs/LIMITATIONS.md` with profile attribution.

**Acceptance criteria:**

- [ ] Consrepro and mapAccumL benchmarks run on both engines at 40000,80000,160000 with min-of-three results, machine/build provenance and expected outputs.
- [ ] mapAccumL O(1)-step benchmark at 160000 completes in under one second on both engines; scaling recorded; any failed threshold remains an explicit failed AC-2.
- [ ] CPU and allocation profiles at 80000 distinguish frame allocation, wider copied values, dispatch and residual costs.
- [ ] AC-4 satisfied either by profile-attributed fix with VM/interpreter ratio <=1.5 at 40000 or committed pprof artifact and LIMITATIONS note accounting for >=90% of gap.
- [ ] Any frame reuse passes nested HOF, callback error recovery and race tests, clears stale references and preserves live frames; no representation change.

### M4: Integration and evaluator handoff (~80 LOC)

**Duration:** 1.0 days. **Dependencies:** M1, M2, M3.

**Files and examples:** Update `changelogs/v0.32-current.md` under v0.51.2 and this plan implementation report; deliver prompt-manager note and evaluator evidence.

**Acceptance criteria:**

- [ ] make test, make verify-examples, make lint and make check-boundaries pass on a toolchain-equipped runner (AC-5).
- [ ] Existing no_loops_fold and effectful_list_t3_foldlE_acc files unchanged with identical outputs; reverse and all HOF dispatch regressions pass.
- [ ] Target-version changelog, prompt-manager note and implementation report map AC-0 through AC-6 to measured evidence; sprint-evaluator receives the completed artifacts.

## Day-by-day execution

1. Build current source, fetch `ailang prompt` before writing `.ail` files, check baseline fixture and run N=40000 on both engines. Correct cost guidance and reconcile RT_REC_003 with any landed tail-call changes. Begin builtin registration and tests.
2. Implement interpreter `mapAccumL`, its pure type/metadata and pre-sized output allocation. Test empty/state/order/polymorphic cases and callback failures.
3. Add native VM callback loop and append-only HOF table entries. Run coverage ratchet and engine-parity tests; check and run the running-total example. Final guidance names mapAccumL only once it is available.
4. Run scaling measurements and CPU/allocation profiles. Decide fix or account from evidence. If frame pooling is justified, implement a per-VM free-list with nesting and error safety and run targeted race tests.
5. Finish AC-4 attribution or remeasure fix, complete integration checks, changelog and prompt note, and prepare evaluator handoff. Use reserved time for profiling or integration failures.

## Measurement and validation contract

Build before measuring; record OS, architecture, hardware, commit and command. Use min-of-three wall times at each size with identical inputs and validate results. Running totals provide one output per element; stateful filtering is illustrated as mapAccumL followed by filter, since this primitive cannot skip outputs. O(n) total cost assumes an O(1) callback; callbacks that cons into state remain quadratic.

AC-0's baseline is the first execution gate; checks of new files recur when those files exist. Use the acceptance section's numbering: AC-2 is the <1s benchmark and AC-3 is coverage/table parity (the design's earlier goals paragraph uses different labels). Keep the absolute <1s criterion intact despite hardware variation; report failures and require an explicit criterion revision rather than silently substituting a scaling ratio. Measure incremental allocation growth as supporting evidence, without treating callback tuple allocations as part of the one-output-allocation guarantee.

Run focused builtin, HOF and recursion-message tests while editing; run the full required suites once after integration. Test new branches and contract cases rather than imposing an unsupported whole-repository coverage percentage. If a pool is implemented, race and nested callback tests are mandatory. Existing executable fixtures remain unchanged; record their outputs on both engines before/after. Inspect available make targets on the executor runner and report missing targets explicitly.

## Risks, dependencies and scope

- Go/make and a fresh source binary are execution prerequisites; current tooling cannot establish AC-0 or integration passes.
- Profile attribution must distinguish CPU versus allocation evidence and explain the wall-time gap quantitatively. Inability to account for 90% is a failed AC-4, not permission to claim a complete investigation.
- Preserve existing HOF indices by appending both tables together; verify native coverage and dispatch.
- Coordinate shared RT_REC_003 text with the tail-call sprint; preserve any implemented tail-call remedy and add the cost caveat.
- Frame reuse must not retain stale references or reuse a frame while nested callbacks still execute. No pool lands without profile evidence and safety tests.
- No changes to list representation, cons semantics, effectful mapAccumLE, parser or tail-call elimination. The representation decision D-19 remains parked.

## Handoff

The coordinator receives both paths through the final markers and reviews the plan before execution. No executor task is launched from this planning stage; merging the coordinator sprint-plan PR provides the skill's approval-controlled handoff. Issue #676 is related context only: use `refs #676`, never auto-close it because quadratic cons remains unresolved. No verified issue identifier for this report was supplied.

## Implementation report (executor fills after execution)

Record fresh-build baseline, exact guidance text, tests and examples, benchmark tables, profile attribution/fix decision, all AC-0–AC-6 outcomes, and prompt-manager caveat. Leave sprint milestones unpassed until evidence exists.
