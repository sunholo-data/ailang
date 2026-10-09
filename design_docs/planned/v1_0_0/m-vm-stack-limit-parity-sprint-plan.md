# Sprint Plan: M-VM-STACK-LIMIT-PARITY

Refs #1576 — https://github.com/sunholo-data/ailang/issues/1576

**Design:** [m-vm-stack-limit-parity.md](m-vm-stack-limit-parity.md)
**Related:** [m-bytecode-vm-parity-bugs.md](m-bytecode-vm-parity-bugs.md), Lane B4.
**Status:** Planned; approved design handoff, implementation awaits coordinator sprint approval.
**Duration:** 1 day (approximately 5 hours including 25% validation buffer).
**Priority:** P1. **Risk:** Medium. **Estimated total:** 220 LOC (10 production, 190 tests, 20 documentation).

## Goal and scope

Restore evaluator-legal deep non-tail recursion under `ailang run --bytecode`: raise the VM default frame cap to 10,000 and apply positive `MaxRecursionDepth` values at the production VM construction site. Pin the stdin-consuming service regression and print-exactly-once behavior. Preserve strict-bytecode behavior and existing tail-call limits.

The approved design establishes stack overflow followed by top-level evaluator replay as the mechanism, rather than a lost continuation or faulty frame unwind. The parent design already records systemic B4 unsafe replay and the quiet-default problem. This sprint covers F1–F3 and AC1–AC7 only. B4 policy, bridge completeness, and the review note's optional unconditional fallback warning remain separate work; above-limit non-strict fallback can still replay effects silently. Do not claim this sprint fixes all silent fallback cases, and do not use an auto-closing keyword for #1576. Every resulting PR body must contain `Refs #1576`; open no new issue for this task.

## Current status and sizing evidence

Issue #1576 and its comments endpoint were read on 2026-10-08 (zero comments). The issue reports the stapledons-godot 2b54a08 service reproducer. The approved design contains first-party default-depth sweeps, verbose fallback observations, independent review confirmation, and audits of VM constructors, callback unwind, explicit test caps, and the existing named-test engine contract.

Current source still has `DefaultMaxStack = 1000` in `internal/vm/vm.go`; `internal/runner/vm.go` constructs its machine without a depth override. `internal/testing/bytecode_engine.go` already uses 10,000 and positive flag wiring. Existing helpers `buildAilang`, `writeFile`, and `runWithStdin` support CLI regressions without new infrastructure.

The seven-day velocity script found no measured LOC velocity. This checkout exposes only one commit (8f15a892, a planning change), so a historical LOC/day rate cannot be inferred. The design estimates half a day; allocate one day for test construction, full-corpus before/after runs, and reconciliation. Planning capacity is 220 LOC/day, not a measured throughput claim. No coverage percentage is invented; target every changed contract with meaningful regression assertions.

## Registry reuse audit

Neither milestone supplies a package-like capability: both modify compiler/runtime enforcement or its CLI regression coverage and documentation. Registry searches are not applicable; no registry package can configure the internal VM frame cap or replace production runner wiring. Each milestone records `none`, package null, and this core-layer rationale in JSON. No new dependency or helper script is needed.

## Milestones and day-by-day execution

### M1: Enforce the production depth contract (~60 LOC)

**Duration:** Day 1 morning, 1.5 hours. **Dependencies:** None.
**Estimate:** 10 production LOC + 50 tests.
**Files:** `internal/vm/vm.go`, new `internal/vm/vm_stack_limit_test.go`, new `cmd/ailang/stack_limit_parity_test.go`, `internal/runner/vm.go`.

**File-size budget:** `internal/vm/vm.go` is already 791 lines. The F1 change (constant plus its comment rewrite) stays at **+9 net lines or fewer** in vm.go. The new VM default-contract test goes in the new file `internal/vm/vm_stack_limit_test.go`, not in `vm_test.go`.
**Examples:** Pure non-tail depth fixture written into `t.TempDir()` by the new CLI test; no new language feature or public example required.

Before editing, build the current checkout (`make build`), capture a full `go run ./scripts/verify_bytecode_parity.go --json` baseline using the freshly built binary, and obtain `ailang prompt` before writing any AILANG fixture. Confirm fixtures type-check with `ailang check` and use `--relax-modules` for temporary module paths.

Add a VM default contract test and a pure CLI flag-wiring test; observe both fail on the baseline. Change `DefaultMaxStack` to 10000 with an accurate evaluator-default comment. Immediately after `vm.NewVM(img)` apply `params.MaxRecursionDepth` only when positive, mirroring the named-test engine. Preserve zero/negative default behavior and explicit small test overrides.

- [ ] New VM construction has MaxStack 10000; explicit smaller caps still overflow and tail calls stay constant-depth.
- [ ] Pure non-tail depth 200 with `--bytecode --strict-bytecode --max-recursion-depth 50` exits nonzero with `stack overflow`; demonstrate red before wiring and green after (design AC3).
- [ ] Default and positive override paths are covered; zero/negative values retain the default rather than disabling the cap.
- [ ] `go test ./internal/vm/...` and the focused new CLI contract tests pass; `git diff --numstat internal/vm/vm.go` shows at most +9 net lines and `vm_test.go` is untouched.

**Risk:** Entry frames shift precise boundaries. Use depths safely separated from the limit; avoid exact threshold assumptions.

### M2: Lock service parity and reconcile the corpus (~160 LOC)

**Duration:** Day 1 afternoon, 2.5 hours plus 1 hour buffer. **Dependencies:** M1.
**Estimate:** 140 tests + 20 documentation LOC.
**Files:** `cmd/ailang/stack_limit_parity_test.go`, `docs/docs/reference/limitations.md`, a changelog fragment `changelogs/unreleased/YYYY-MM-DD-vm-stack-limit-parity.md` (never `changelogs/v0.32-current.md`); reconcile results in this sprint plan or an adjacent markdown verification report.
**Examples:** Temporary service-loop, standalone deep recursion, and pure-depth fixtures; existing `examples/runnable/recursion_quicksort.ail`, `block_recursion.ail`, `cmd/ailang/testdata/tailcall/shapes.ail`, and `stdin_loop.ail` remain working.

Use one locally built CLI and the existing stdin helper. Record baseline failures for AC1, AC2, and the under-limit half of AC4 before the fix, then validate against the fixed binary. Service and println fixtures must use non-strict mode because IO stubs need the evaluator bridge; pure fixtures must use strict mode to rule out top-level fallback.

- [ ] Service handler with non-tail rep(1170), stdin `hello\nvoice\n`, and `--bytecode --verbose --caps IO --entry main` exits 0 with exactly four stdout lines: HELLO, ASK, HANDLED len=1170, HANDLED-AFTER; stderr has no `falling back to evaluator` (AC1). Compare the interpreter output too.
- [ ] Standalone rep(5000) prints exactly START once and DONE len=5000, exits 0, and has no fallback warning under `--bytecode --verbose` (AC2).
- [ ] Pure depth 9000 succeeds with identical interpreter/strict-VM output; depth 11001 fails loudly on both, with RT_REC_003 for the interpreter and stack overflow for the strict VM (AC4). IO-free strict fixtures return a computed value.
- [ ] Focused `go test ./internal/vm/... ./internal/runner/... ./internal/testing/...` plus `go test ./cmd/ailang/... -run 'StackLimit|Bytecode|TailCall|Fallback|Stdin' -count=1` pass, including named-test engine depth wiring, frame reuse, existing fallback messages, and tail-call parity (AC5); then `make test-core`. Do **not** run the full `make test` (or an unfiltered `./cmd/ailang/...`) locally: in the executor's RAM-backed /tmp it has crashed with SIGBUS. The full suite runs in CI on the PR.
- [ ] Limitations reference documents default 10,000 and the shared flag, while distinguishing VM frame counting from evaluator depth accounting (AC6).
- [ ] Changelog fragment `changelogs/unreleased/YYYY-MM-DD-vm-stack-limit-parity.md` records the **behaviour change**: `--max-recursion-depth` now bounds the bytecode VM as well as the evaluator, and the VM's default frame cap goes from 1000 to 10000. Refs #1576.
- [ ] Run the full parity harness after rebuilding; reconcile every changed row by filename and status against the baseline (AC7). Record counts, commands, binary provenance, and timings. Never accept totals alone or a fallback as proof of VM success.
- [ ] `make fmt`, `make lint`, and `git diff --check` pass; fixture type-checking and elapsed integration-test timings are recorded.

**Risk:** Newly VM-native deep examples may expose latent VM defects previously hidden by fallback. Record evidence by name; do not revert the cap or silently widen this sprint. Link existing reports where available; escalate any blocking newly exposed defect to the coordinator for scope routing. The issue prohibition means do not open new issues in this task.

## Completion and handoff

Both milestones must satisfy their criteria before sprint-evaluator evaluates the implementation against the approved design. Store pre/post harness reports outside the runnable corpus. Avoid adding a stdin-consuming public example that would hang the corpus runner. A fresh CLI build is required for both harness legs to prevent testing a stale preinstalled binary.

There are no external implementation dependencies or unresolved design choices within F1–F3. The original service fixture does not need to be fetched because the approved minimal reproducer captures the consumed-input mechanism. Deep tests and frame-pool retention at opt-in large limits are the principal validation risks; no memory redesign is authorized here.

The populated sprint JSON is the executor resumption artifact. Coordinator approval/merge routes this plan to sprint-executor; this planning stage does not start implementation or send a duplicate manual dispatch. Include `Refs #1576` in that PR body and subsequent implementation PR bodies.

Executor rules: re-run `.claude/skills/sprint-executor/scripts/validate_sprint_json.sh M-VM-STACK-LIMIT-PARITY` before starting and after every sprint JSON update. The executor cannot push or merge; it commits locally (messages carry `Refs #1576`) and the coordinator raises the PR. Keep B4 residual behavior explicit in final verification and issue updates.
