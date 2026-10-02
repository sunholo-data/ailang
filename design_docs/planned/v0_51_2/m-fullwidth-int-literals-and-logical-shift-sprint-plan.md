# Sprint Plan: M-FULLWIDTH-INT-LITERALS-AND-LOGICAL-SHIFT

## Summary
Implement the approved v0.51.2 design: write full-width 64-bit radix constants directly, reject invalid pattern literals explicitly, and provide logical right shift through a pure builtin and std/math export.

**Duration:** 2 days, 13 engineering hours (11 implementation/verification + 2 buffer).
**Estimated LOC:** 420 total: 70 Go/stdlib implementation, 245 tests, 105 examples/docs/prompt.
**Risk:** Medium: parser error propagation, MinInt64 formatting and SMT encoding.
**Design:** [Approved design](m-fullwidth-int-literals-and-logical-shift.md).
**Issue:** #1483; source handoff task-685c1642.
**Status:** Planned; implementation has not started. This artifact completes the requested planner stage; coordinator advances execution under its approval workflow.

## Current status and velocity
Source inspection confirms signed ParseInt for radix literals, discarded int/float pattern errors, no logical-shift registration, and overflow-prone SMT negation. Working tree was clean; current branch is coordinator/task-f7cef40e (the supplied design branch is provenance, not a branch-switch instruction). std/VERSION is v0.51.0.

The existing analyze_velocity.sh 7 script found only c77fae8d (2026-10-02, design-doc handoff); this is a shallow checkout. Its changelog matches include old milestones, so they are not current velocity evidence. Measured implementation LOC/day and completion rate are unavailable. Planning capacity is **210 LOC/day**, an estimate from scoped task effort, not a historical measurement. Coverage baseline is not measured during planning; capture touched-package coverage before implementation and require no regression plus coverage of every new boundary/error branch.

## Registry reuse audit
`ailang pkg search bitwise` and `ailang pkg search splitmix` both returned no packages (the binary also warned it may be stale). No candidate required pkg info/docs. All three milestones use **none**: packages cannot repair parser or SMT internals or add a runtime primitive; the mixer is a fixture, and std/prng remains outside this sprint. JSON records the decision per milestone.

## Milestones

### M1: Full-width radix literals and positioned diagnostics (~145 LOC)

**Files and effort:** internal/parser/parser_literals.go, internal/parser/parser_pattern.go; extend parser_literals_test.go and pattern tests. Update examples/integer_literals.ail in M3. Estimate: 35 implementation + 110 tests = 145 LOC; 4h. No dependencies.

**Acceptance criteria:**

- [ ] Hex, binary and octal patterns through MaxUint64 reinterpret as int64; SplitMix64/FNV constants, MinInt64 and all-ones boundaries parse correctly.
- [ ] Decimal overflow remains an error; 0123 stays 123; existing in-range radix tests remain unchanged.
- [ ] Expression and pattern integer overflow produce positioned PAR021; float pattern overflow produces PAR021 rather than +Inf.
- [ ] Top-bit literal patterns match the intended signed value only; errors cannot leak ParseUint clamp values.
- [ ] Diagnostic suggestions use computed values and a parseable replacement for MinInt64 (0x8000000000000000); tests pin position, code and suggestions.

### M2: Pure logical right shift builtin and std/math export (~105 LOC)

**Files and effort:** internal/builtins/math_bitwise.go, std/math.ail, math_bitwise_test.go, internal/vm/builtin_coverage_test.go; exercise examples/runnable/bitwise.ail and add SplitMix64 in M3. Estimate: 30 implementation + 75 tests = 105 LOC; 3h. No dependencies.

**Acceptance criteria:**

- [ ] shiftRightLogical_Int is pure with int/int -> int metadata and std/math exports shiftRightLogical.
- [ ] Both backends return 9223372036854775807 for (-1,1), 1 for (-1,63) and (MinInt64,63), and 4611686018427387904 for (MinInt64,1).
- [ ] Count 0 preserves bits; counts 64,65 and MaxInt64 return 0; negative counts raise RT_SHIFT on both backends.
- [ ] VM adapter coverage includes _shiftRightLogical_Int; existing arithmetic >> and frozen builtin interface remain unchanged.

### M3: MinInt64 SMT safety, examples and teaching documentation (~170 LOC)

**Files and effort:** internal/smt/codegen_expr.go and encoder tests; examples/integer_literals.ail, new examples/runnable/splitmix64.ail and example manifest; formatter/backend integration tests; docs/docs/reference/stdlib.md, docs/docs/reference/language-syntax.md, prompts source selected by prompts/versions.json, docs/docs/prompts/current.md. Use prompt-manager when editing prompt/version metadata; use ailang prompt before writing .ail. Estimate: 5 implementation + 60 tests + 105 examples/docs/prompt = 170 LOC; 4h plus 2h buffer. Depends on M1 and M2.

**Acceptance criteria:**

- [ ] SMT encodeLit emits (- 9223372036854775808) for MinInt64; focused encoder tests pass and a contract fixture reaches Z3 when available.
- [ ] Report repro with argument 1 yields -7046029254386353130 on interpreter and strict bytecode.
- [ ] SplitMix64 runnable example yields -2152535657050944081 for seed 0 and -4767286540954276203 for seed 42 on both backends.
- [ ] Formatter output for full-width literals is stable and reparses to the same value, including MinInt64; if MinInt64 cannot round-trip, fix within approved literal scope or surface a design blocker before completion.
- [ ] integer_literals, runnable/bitwise, tests/binops_int and existing logical-shift mask construction retain behavior.
- [ ] Prompt source/version metadata/current mirror and stdlib reference document radix bit patterns and shiftRightLogical; all new AILANG examples are checked.
- [ ] make test-core, focused SMT tests, make verify-examples, make verify-examples-toplevel, make fmt-check, make lint and make check-boundaries pass; example manifest is synchronized.

## Day-by-day execution

| Day | Time | Work |
|---|---|---|
| 1 | 4h | M1: extend conversion tables and pattern error tests; implement shared radix conversion and positioned PAR021; run parser tests. |
| 1 | 3h | M2: builtin, wrapper, runtime vectors, metadata and VM adapter tests; run builtins/VM tests. |
| 2 | 1h | M3: fix SMT unsigned magnitude conversion; unit-test MinInt64 and exercise contract through Z3. |
| 2 | 2h | M3: add checked SplitMix64 and literal fixtures; verify known answers on both backends, formatter round-trip and generated-Go negative constants. |
| 2 | 1h | M3: prompt/docs synchronization and focused/full required gates. |
| 2 | 2h | Buffer: gate failures, manifest synchronization, MinInt64 formatting and diagnostic refinements; evaluator handoff. |

## Technical constraints and risks

Use ParseUint only for explicit radix prefixes and int64 reinterpretation; decimal stays ParseInt base 10. Preserve conversion errors structurally from literalValue to parseBasePattern. Reserve PAR021 after rechecking allocation. Match existing builtin argument validation and RT_SHIFT behavior; unsigned shift is int64(uint64(a) >> count). Registry adaptation is automatic: do not add a core operator or freeze-table entry. No ushr alias this sprint.

MinInt64 has two integration hazards. SMT must format an unsigned magnitude rather than an overflowed signed negation. Formatter currently emits signed decimal, but the design says decimal magnitude 9223372036854775808 still fails to parse: explicitly test MinInt64 round-trip before declaring completion. Prefer a parseable radix representation for that boundary if required; do not silently broaden decimal overflow policy. Diagnostic text must suggest the parseable hex replacement and identify the signed value without promising the signed decimal form parses.

Z3 absence must be reported as an unexecuted integration check, not a successful proof. Unit encoder coverage remains mandatory. Verify generated-Go consumers remain negative-safe; avoid speculative changes to unrelated backends. All new examples must be registered in the existing manifest so verification gates observe them.

## Verification and completion

Run focused Go tests for parser, builtins, VM, SMT and formatter during each milestone; capture touched-package coverage baseline and final coverage with go test -cover. Run make test-core, make verify-examples, make verify-examples-toplevel, make fmt-check, make lint and make check-boundaries at integration. Keep examples/integer_literals.ail, examples/runnable/bitwise.ail and tests/binops_int.ail as regression fixtures. Verify the parked m-pure-prng mask construction in a temporary checked fixture without implementing that package.

Completion requires all milestone criteria, backend agreement, populated JSON progress, and sprint-evaluator review against the approved design. >>> syntax, unsigned types, decimal wrapping and std/prng remain deferred. No blocking product questions; test placement and prompt version mechanics follow existing skills. The MinInt64 formatter mismatch is a concrete verification risk, not a reason to omit the test.
