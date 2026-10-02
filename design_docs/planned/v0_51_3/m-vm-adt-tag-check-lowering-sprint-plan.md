# Sprint Plan: M-VM-ADT-TAG-CHECK-LOWERING

## Summary
Fix constructor tag checks in mixed matches for strict bytecode and Go v2, using the approved [design](m-vm-adt-tag-check-lowering.md). Ratify `stmt.ADTTagEq{Value Expr, TypeName string, Tag string}`. No VM, builtin, parser or type-system changes.

**Duration:** 2 days, approximately 8 focused hours plus scheduling buffer.
**Estimated total:** 540 LOC (190 implementation, 330 tests/fixtures, 20 documentation).
**Risk:** Medium: shared lowering, expression visitors and two backends.
**Status:** Ready for coordinator plan review; implementation not started. Design approval is supplied by the handoff; sprint execution follows coordinator approval/merge.

## Current status and estimation evidence
The working tree was clean on `coordinator/task-9fb101b4`; std/VERSION is v0.51.0. Source inspection confirms both erroneous tag accesses in match.go, existing GET_TAG switch compilation, and the old containsTagCheck assertion. The design already audits related shapes, record-name collisions and both backends.

The seven-day velocity script found only the design commit 07846355 and no usable implementation LOC metrics. Its changelog scan includes historical foundation entries, which are not recent velocity evidence. A measured LOC/day cannot be inferred from this shallow history. Use the localized design's eight-hour budget and a conservative two-day window; 270 LOC/day is a planning target, not measured throughput. No coverage baseline was run: this planning host lacks Go. All implementation verification remains mandatory on an equipped executor host.

Additional source discovery: `rewriteExpr` and `walkExpr` in internal/gen/lower/program.go and the free-variable visitor in internal/bytecode/compiler/lambda.go must traverse the new node's Value. Include closure and cross-module regressions to prevent missing captures or rewriting.

## Registry reuse audit
`ailang pkg search bytecode` returned no packages (installed binary warned it may be stale). These milestones modify compiler-private Go IR and backends, not a package-like user capability. No candidate warranted pkg info/docs. Each milestone chooses **none**: registry dependencies cannot supply internal lowering, opcode compilation or emitter visitors. Record this decision in JSON for all three milestones.

## Milestones

### M1: ADT tag IR and bytecode compilation (~160 LOC)

**Dependencies:** none

**Budget:** 2.5 hours; 75 implementation + 85 tests

**Files:** `internal/gen/stmt/stmt.go`, `internal/gen/stmt/validate.go`, `internal/bytecode/compiler/expr.go`, `internal/bytecode/compiler/switch.go`, `internal/bytecode/compiler/lambda.go`, `internal/gen/lower/program.go`, `internal/bytecode/compiler/adt_tag_test.go (new)`.

**Tasks and acceptance criteria:**

- [ ] ADTTagEq validates its Value and compiles to GET_TAG, ordinal LOAD_CONST and EQ without new opcodes or builtins.
- [ ] Explicit types, declaration-order inference, shared-tag determinism and unknown type/tag errors have unit coverage.
- [ ] Switch resolution retains its existing behavior; rewriting, expression walking and closure free-variable discovery traverse ADTTagEq.Value.

**Example coverage:** Use direct IR tests here; M3 delivers the runnable cons_head_ctor.ail fixture.

**Risk and mitigation:** Preserve switch inference across the entire tag set; infer by declaration order only, and test unknown tags and closure traversal.

### M2: If-chain lowering and unsupported-pattern boundary (~170 LOC)

**Dependencies:** M1

**Budget:** 2.5 hours; 55 implementation + 115 tests

**Files:** `internal/gen/lower/match.go`, `internal/gen/lower/lower_match_test.go`.

**Tasks and acceptance criteria:**

- [ ] Both lowering sites emit ADTTagEq; no FieldAccess with Field Tag remains in internal/gen/lower.
- [ ] Type information propagates to tag checks and ADT positional bindings; Var and Wildcard arguments work.
- [ ] Literal and nested constructor arguments produce precise EvalOnly reasons; strict execution fails loudly and non-strict agrees with evaluator.
- [ ] Existing lowering regressions pass with containsTagCheck updated to the new node.

**Example coverage:** Exercise reported Tok/Opt shapes in lowering tests; M3 delivers executable examples.

**Risk and mitigation:** Tag repair can expose previously ignored argument tests; enforce the loud boundary before claiming parity. Preserve length guards and binding scope.

### M3: Go emission and engine parity verification (~210 LOC)

**Dependencies:** M1, M2

**Budget:** 3 hours; 60 implementation + 130 tests/fixtures + 20 docs

**Files:** `internal/gen/emitgo/funcs.go`, `internal/gen/emitgo/adt_tag_test.go (new)`, `tests/golden/bytecode/cons_head_ctor.ail (new)`, `cmd/ailang/run_bytecode_test.go`, `changelogs/v0.32-current.md`, `docs/docs/reference/implementation-status.md`.

**Tasks and acceptance criteria:**

- [ ] Go v2 emits Kind enum comparisons and ValueN bindings; generated repro Go builds and computes the evaluator result.
- [ ] Cons-head goldens cover both constructors, nullary heads, wildcard args, empty lists, mismatches and head-separated equivalence with exact evaluator/strict-VM outputs.
- [ ] Genuine record fields named Tag and _0 retain record semantics; unresolved ADT emission errors are loud.
- [ ] Focused tests, make test, make fmt, make lint and make check-boundaries pass on a Go-equipped execution host.
- [ ] Baseline-relative std/examples/golden sweep has no output changes or new EvalOnly functions on previously supported inputs; changelog and reference note are updated.

**Example coverage:** Use the new cons_head_ctor.ail fixture as the runnable example and Go compile/build/run input.

**Risk and mitigation:** List extraction is interface-valued in generated Go; verify necessary typed ADT conversion and positional field emission by building and running, not string assertions alone.

## Day-by-day execution
Day 1: baseline focused regressions, implement M1 with failing unit tests first, then M2 and loud unsupported-pattern tests. Read `ailang prompt` before writing any .ail fixture. Preserve existing match guard/binding regressions.

Day 2: M3 emitter and end-to-end tests; use existing golden harness and make targets rather than new sweep scripts. Compare evaluator, strict VM and generated Go; run required checks and update docs. Reserve the remaining window for failures or emitter typing issues.

## Validation and success metrics
All milestone criteria must pass. Focused Go tests cover internal/gen/stmt, internal/gen/lower, internal/bytecode/compiler, internal/gen/emitgo and cmd/ailang before full checks. Pin exact output, not merely successful process exit. Verify verbose non-strict execution uses VM without fallback for supported examples. Unsupported literal/nested arguments intentionally remain EvalOnly and must produce the evaluator's answer in non-strict mode.

Use existing test infrastructure to establish a baseline for runnable std consumers, examples and bytecode goldens. Many std modules have no entrypoint and some examples already require unavailable capabilities or unsupported VM features: record those baseline outcomes rather than promise every module runs strictly. Require zero regressions in the pre-existing supported set and no new EvalOnly classifications there. Record new unsupported-boundary classifications separately. Review coverage for all new branches; no unsupported repository-wide percentage target is asserted.

## Dependencies, scope and handoff
A Go-equipped executor host is required for compilation, generated-Go validation and make targets; absence here does not prevent planning. Use the current branch and preserve other work. This sprint precedes sibling nested-pattern and var-default-arm lowering work; coordinate overlapping match.go changes when they land. Recursive patterns and guard-routing changes remain outside this design's scope.

The latest commit title references #1511, but the supplied artifact does not explicitly identify its originating issue; leave github_issues empty pending coordinator confirmation, rather than infer an issue from unrelated numeric/disassembly references.

Machine progress lives at `.ailang/state/sprints/sprint_M-VM-ADT-TAG-CHECK-LOWERING.json`, with all passes null. Coordinator output markers provide the handoff; approval/merge triggers sprint-executor as documented in the skill's coordinator resource. Do not dispatch duplicate execution or mark milestones complete during planning.

## Planning artifact validation
Python validation passed for JSON syntax, milestone count, dependency IDs, LOC totals and registry decisions. The standard validator could not run successfully because jq is absent (its generic message reports invalid syntax); independent JSON parsing confirms valid syntax. Re-run the standard validator on the executor host.
