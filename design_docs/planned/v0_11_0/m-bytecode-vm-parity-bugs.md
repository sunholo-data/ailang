# M-BYTECODE-VM-PARITY-BUGS — Remaining VM/Eval Divergences

**Status**: Planned
**Target**: v0.11.0
**Priority**: P1 (blocks bytecode parity gate; not blocking M2/M3 of M-BYTECODE-MULTIMODULE)
**Estimated**: 1–2 days (investigation-heavy)
**Dependencies**: M-BYTECODE-MULTIMODULE M1 (surfaced these; M1 complete)

## Problem Statement

During M-BYTECODE-MULTIMODULE M1 verification, the parity harness
(`scripts/verify_bytecode_parity.go`) reveals **3 remaining examples** where
`ailang run --bytecode` produces output that disagrees with `ailang run`
(pure evaluator). All 3 were already divergent at the start of the M1 session
(measured baseline **126 MATCH / 7 DIVERGE**); M1's work on per-proto
validation and `builtinShow` completeness reduced this to **130 MATCH / 3
DIVERGE** (with the math_trig float-format fix rolled into M1's show work).

The 3 remaining failures are distinct root causes — this doc groups them
because they share a single investigation surface (VM→eval bridge + lowered
call dispatch), but each sub-bug gets its own Phase in the implementation
plan.

### Current Parity State (as of M1 completion, 2026-04-08)

```
MATCH      130  (92.2%)
NON_DET      2  (1.4%)
DIVERGE      3  (2.1%)
EVAL_SKIP    6  (4.3%)
```

### The 3 Diverging Examples

| # | File | Symptom | Likely Class |
|---|------|---------|--------------|
| 1 | `examples/runnable/pattern_sugar.ail` | `tail([1,2,3,4,5]) = <List>` in VM; `[2, 3, 4, 5]` in eval | show dispatch — list return from bridged stdlib call |
| 2 | `examples/runnable/recursion_quicksort.ail` | `Quicksort: <List>` in VM; `[1, 1, 2, 3, 4, 5, 6, 9]` in eval | same `<List>` pattern as #1 |
| 3 | `examples/runnable/string_parsing.ail` | Header lines duplicated; mojibake on non-ASCII chars (`\xe2\x9c…`) | separate: loop/UTF-8 bug, not show |

**Impact:**
- Blocks the sprint-acceptance parity criterion for M-BYTECODE-MULTIMODULE
  ("parity harness stays at X MATCH / 2 NON_DET / 6 EVAL_SKIP").
- Does **not** block M2/M3/M4/M5 of M-BYTECODE-MULTIMODULE — the same bugs
  would exist if those milestones landed today.
- All 3 were pre-existing bugs masked before M1 (because stdlib modules
  were never lowered to bytecode). M1's broadened lowering surface exposed
  them as the default path shifted from eval-bridge to in-VM dispatch.

## Goals

**Primary Goal:** Close the 3 remaining parity divergences so the
M-BYTECODE-MULTIMODULE acceptance gate is met and the bytecode VM produces
output byte-identical to the evaluator across the full runnable corpus (modulo
known NON_DET cases).

**Success Metrics:**
- Parity harness reaches **≥ 133 MATCH / 2 NON_DET / 0 DIVERGE / 6 EVAL_SKIP**
  (i.e., all 3 DIVERGE resolved; held-constant NON_DET and EVAL_SKIP sets).
- No regression in the 130 currently-matching examples.
- For each sub-bug: a minimal reproducer committed under
  `tests/golden/bytecode/` or `internal/vm/*_test.go` that would have caught
  it pre-fix.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Where the `<List>` wrap happens for bridged return values (VM side vs bridge converter vs lower pass) | Determines whether the fix is in `bytecode_bridge.go`, `vm/vm.go`, or `gen/lower/expr.go` | agent (after investigation) | design | med |
| Whether `string_parsing.ail`'s duplicated output is a lowering bug (double-emit) or a VM execution bug (jump off-by-one) | Changes the file being edited and the shape of the repro test | agent | design | low |
| Whether the fix lands in v0.11.0 or gets deferred to v0.11.1 if investigation exceeds 1 day | Affects release timing for M-BYTECODE-MULTIMODULE | human | runtime | low |

### Design Freeze

- [ ] Confirm with user: is this sprint blocking v0.11.0 release, or can M2–M5 proceed in parallel?
- [ ] Confirm scope: fix all 3 OR document remaining as known-limitations if investigation stalls on any single bug?

## Solution Design

### Overview

Three focused bug investigations. Each follows the same pattern:
1. **Reproduce** via minimal AILANG snippet
2. **Trace** via `DEBUG_STRICT=1`, `DEBUG_OPERATOR_LOWERING=1`, and/or
   `ailang disasm` + `ailang trace`
3. **Fix** at the correct layer (lower pass vs VM dispatch vs bridge)
4. **Regress-test** with a golden or unit test

### Sub-Bug 1 & 2: `<List>` from bridged stdlib returns

**Hypothesis:** `tail(...)` and `quicksort(...)` lower to VM calls but the
stdlib implementations remain EvalOnly. The bridge roundtrip should convert
`*eval.ListValue` → `bytecode.Value{Tag: TagList}` — and for most cases it
does (cons_expression, buildList, etc. now MATCH). For these two, the return
value is reaching the caller as some other tag (likely `TagADT` with ordinal
0, since `List` is represented as a cons-list ADT in the Core IR) and
`builtinShow`'s `TagADT` case delegates to `Value.String()` which prints
`<adt#N …>` — but neither of those substrings appears, so the actual path
must be elsewhere.

**Actual emission site** to confirm: only 3 places format `<TagName>`:
- `internal/vm/builtins.go:109` (`showValue` default) — primary suspect
- `internal/bytecode/disasm.go:217` — disassembler only, not runtime
- `internal/bytecode/value.go:412` — unknown-tag fallback

The emission as literal `<List>` rules out the unknown-tag fallback. It must
be `builtinShow`'s default case hitting a `v.Tag` whose `String()` returns
`"List"` — i.e., `TagList` itself. Which means the `case bytecode.TagList`
IS being bypassed. **Investigation needed**: under what conditions does a
`switch v.Tag { case bytecode.TagList: ... }` not match a value with
`Tag == TagList`? Possibilities:
- `_show` dispatched to a **different** handler (not `builtinShow`) via a
  type-class dictionary indirection
- `v` is wrapped in an ADT envelope (`TagADT` with 1 field that is a TagList)
  and `Tag.String()` for `TagList` happens somewhere in a dict call
- The lowered show call is going through `OpBuiltinTrap` → eval-bridge →
  eval's `showValue` — but eval's default case is `fmt.Sprintf("<%T>", val)`
  which would print `<*eval.ListValue>`, not `<List>`

**Most likely cause:** list-returning stdlib functions like `tail`
are lowered but their type parameters are not properly propagated to the
show call's dictionary lookup, so show resolves to a generic `showAny` that
prints the Tag name as a fallback. Fix would be in either the type-class
dictionary lowering (`internal/gen/lower/`) or the dispatch table binding.

### Sub-Bug 3: `string_parsing.ail` duplicated output + mojibake

**Hypothesis:** Separate class of bug. Two distinct failure modes in one file:

1. **Duplicated header** — "=== String Parsing Examples (M-DX10) ===" prints
   twice. Suggests either a main-function double-entry (entry-point
   resolution bug from M1's canonical naming?) or a loop-emission bug in
   the lower pass for sequential `println` statements.

2. **Mojibake (`\xe2\x9c`)** — `✓` (U+2713) is 0xE2 0x9C 0x93 in UTF-8. The
   bytes 0xE2 0x9C are the first 2 of the 3-byte sequence. This suggests
   the string is being truncated mid-UTF-8 or sliced by byte index somewhere
   in string lowering or the VM's OpConcat handling.

Likely independent fixes. Worth triaging whether the duplication is caused
by M1's canonical-name changes (new: check whether `examples/runnable/
string_parsing.main` AND `string_parsing/main` both end up in `FuncDecls`
after the disasm.go:compileBytecodeFromResult iteration).

### Implementation Plan

**Phase 1: Investigate `<List>` root cause** (~3 hours)
- [ ] Add `DEBUG_STRICT=1` tracing to `builtinShow` default case that logs
      `v.Tag` integer value + proto name of the caller
- [ ] Run `ailang run --bytecode` on a minimal repro: `show(tail([1,2,3]))`
- [ ] Inspect disasm of `examples/runnable/pattern_sugar.ail` to find how
      `tail` is lowered (OpBuiltinCall? OpBuiltinTrap? OpCall to stdlib
      proto?) — `ailang disasm` is the tool
- [ ] Write down the actual dispatch path; update design doc with findings

**Phase 2: Fix `<List>` at the correct layer** (~3 hours)
- [ ] Implement fix based on Phase 1 findings (likely in `internal/gen/
      lower/expr.go` or `internal/vm/vm.go`)
- [ ] Add golden test covering `show(tail([1,2,3,4,5]))` and
      `show(quicksort([3,1,4,1,5]))` in `tests/golden/bytecode/`
- [ ] Re-run parity harness — expect MATCH to increase by 2

**Phase 3: Investigate `string_parsing.ail` dual failures** (~3 hours)
- [ ] Minimal repro for duplication: strip file to just 2 println calls,
      check if duplication reproduces
- [ ] Minimal repro for mojibake: `println("✓")` alone, check if mojibake
      reproduces
- [ ] Disasm of repros; check for double-emission of entry-point body
- [ ] Trace OpConcat / string lowering paths for multi-byte UTF-8

**Phase 4: Fix `string_parsing.ail` bugs** (~3 hours)
- [ ] Fix duplication (likely `cmd/ailang/disasm.go:compileBytecodeFromResult`
      or `internal/gen/lower/file.go` entry emission)
- [ ] Fix mojibake (likely `internal/bytecode/value.go:AsString` or
      OpConcat string slicing)
- [ ] Add golden tests for both
- [ ] Re-run parity harness — expect MATCH = 133

### Files to Modify/Create

**Likely modified (actual list depends on investigation):**
- `internal/gen/lower/expr.go` — if dictionary lowering for show is at fault
- `internal/vm/vm.go` — if a dispatch opcode mishandles a value tag
- `cmd/ailang/bytecode_bridge.go` — if roundtrip conversion is wrong
- `cmd/ailang/disasm.go` — if `compileBytecodeFromResult` double-emits entry
- `internal/bytecode/value.go` — if `AsString` truncates UTF-8

**New test files:**
- `tests/golden/bytecode/show_bridged_list.ail` — minimal repro for sub-bugs 1 & 2
- `tests/golden/bytecode/unicode_println.ail` — minimal repro for sub-bug 3 mojibake
- `internal/vm/show_dispatch_test.go` — unit tests for `builtinShow`
  dispatch across all Tag shapes, including bridged returns

## Examples

### Sub-Bug 1: `tail` return

**Eval output (correct):**
```
head([1,2,3,4,5]) = 1
tail([1,2,3,4,5]) = [2, 3, 4, 5]
```

**VM output (buggy):**
```
head([1,2,3,4,5]) = 1
tail([1,2,3,4,5]) = <List>
```

Note `head` works — it returns an int. Only the list-returning call is broken.

### Sub-Bug 2: quicksort

**Eval output (correct):**
```
Quicksort: [1, 1, 2, 3, 4, 5, 6, 9]
sortBy:    [1, 1, 2, 3, 4, 5, 6, 9]
```

**VM output (buggy):**
```
Quicksort: <List>
sortBy:    <List>
```

### Sub-Bug 3: string_parsing

**Eval output (correct):**
```
=== String Parsing Examples (M-DX10) ===

Integer parsing:
✓ stringToInt("42") = Some(42)
```

**VM output (buggy — header doubled, UTF-8 truncated):**
```
=== String Parsing Examples (M-DX10) ===

Integer parsing:
=== String Parsing Examples (M-DX10) ===

Integer parsing:
<mojibake bytes>
```

## Success Criteria

- [ ] `go run ./scripts/verify_bytecode_parity.go` reports **≥ 133 MATCH /
      2 NON_DET / 0 DIVERGE / 6 EVAL_SKIP**
- [ ] Sub-Bug 1: `show(tail([1,2,3,4,5]))` via bytecode prints `[2, 3, 4, 5]`
- [ ] Sub-Bug 2: quicksort produces correct sorted list via bytecode
- [ ] Sub-Bug 3: UTF-8 string with `println` in bytecode path produces
      byte-identical output to evaluator
- [ ] Sub-Bug 3: A file with only 2 sequential `println` calls does not
      duplicate output in bytecode path
- [ ] Regression tests added for all 3 sub-bugs; they would fail on
      `f108cceb` (pre-fix state) and pass on the fix commit
- [ ] No regression in the 130 currently-matching parity examples

## Testing Strategy

**Unit tests:**
- `internal/vm/show_dispatch_test.go` — constructs every Value tag shape
  (Int/Float/Bool/String/Unit/List/Tuple/Record/ADT) and asserts
  `builtinShow` output matches evaluator `showValue`
- Roundtrip test: eval.ListValue → bytecode.Value → builtinShow must equal
  eval.showValue(original eval.ListValue)

**Integration tests (golden):**
- `tests/golden/bytecode/show_bridged_list.ail` — reproduces sub-bugs 1 & 2
- `tests/golden/bytecode/unicode_println.ail` — reproduces sub-bug 3 mojibake
- `tests/golden/bytecode/sequential_println.ail` — reproduces sub-bug 3
  duplication

**Manual testing:**
- Re-run full parity harness
- Re-run `ailang disasm /Users/mark/dev/sunholo/ailang-parse/docparse/main.ail`
  to confirm no regression in the large-program case (currently 1109 protos,
  435 EvalOnly)

## Deferred Decisions

- **ADT constructor name rendering** — Already explicitly deferred to
  M3_CROSS_MOD_ADT_RECORD of M-BYTECODE-MULTIMODULE. The VM's `builtinShow`
  currently falls through to `Value.String()` for ADT, printing
  `<adt#N …>`. Not in scope here.
- **NON_DET set (2 examples)** — Deliberately excluded from parity; out
  of scope.
- **EVAL_SKIP set (6 AI/exit-code examples)** — Evaluator itself fails on
  these (missing AI keys / intentional exit codes); out of scope.
- Whether a generic "VM dispatch table debugger" deserves a standalone
  helper — agent may decide during Phase 1 investigation.

## Non-Goals

- **Fixing unrelated bytecode bugs** — Only the 3 specific diverges in
  this doc.
- **Rewriting the show dispatch model** — If a targeted fix works, take it.
  A broader refactor of type-class dictionary lowering is v1.0.0 scope.
- **Fixing the evaluator's show default case** — `eval.showValue` prints
  `<*eval.TupleValue>` for tuples (missing case); out of scope here, would
  be a separate doc if it matters.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Sub-Bug 1 root cause is in type-class dictionary lowering and requires broader refactor | Medium | Time-box Phase 1 to 3 hours; if investigation stalls, document findings and defer to a dedicated sprint |
| string_parsing's 2 failures have distinct root causes and require 2 separate fixes | Low | Already planned as distinct Phase 3/4 steps |
| Fix introduces regression in the 130 currently-matching examples | High | Parity harness + full `make test` after every change; bisect with `--only` flag |

## Related Documents

<!-- Auto-populated by Ollama neural search on "bytecode vm parity bugs" -->

**Implemented (may inform design):**
- [design_docs/implemented/v0_11_0/m-bytecode-vm.md](../../implemented/v0_11_0/m-bytecode-vm.md) — M-BYTECODE-VM master design; §17.3 is the multimodule extension
- [design_docs/implemented/v0_5_10/m-string-conversion.md](../../implemented/v0_5_10/m-string-conversion.md) (0.47)

**Planned (check for overlap):**
- [design_docs/planned/v0_11_0/m-bytecode-multimodule-sprint-plan.md](m-bytecode-multimodule-sprint-plan.md) — parent sprint; these bugs block its parity gate
- [design_docs/planned/v1_0_0/m-perf4-bytecode-interpreter.md](../v1_0_0/m-perf4-bytecode-interpreter.md) (0.48)

## Axiom Compliance

Per-bug fixes; no semantic changes to the language. All axioms score 0 except:

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Bytecode path becomes deterministic wrt eval (currently divergent → non-reproducible across backends) |
| A7: Machines First | +1 | Agent-driven bytecode parity verification becomes reliable |
| A11: Structured Failure | 0 | No change to error surface |
| Others | 0 | No change |

**Net Score: +2** → **Decision: Proceed**

**Hard Violation Check:**
- [x] A1: Fix *improves* determinism
- [x] A3: No effect changes
- [x] A4: No authority changes
- [x] A7: Fix *improves* machine-first alignment

## References

- [M-BYTECODE-VM §17.3 Multimodule lowering](../../implemented/v0_11_0/m-bytecode-vm.md)
- [Sprint JSON for M-BYTECODE-MULTIMODULE](../../../.ailang/state/sprints/sprint_M-BYTECODE-MULTIMODULE.json) — M1 completion notes document the 3 remaining DIVERGE as carryover
- `scripts/verify_bytecode_parity.go` — the parity harness
- Commit `f108cceb` — `M-BYTECODE-BATCH: --batch honors --bytecode + per-fn lower recovery` (pre-M1 baseline)

## Future Work

- Unified ADT name rendering (waits on M3 of M-BYTECODE-MULTIMODULE)
- Eval `showValue` default case for TupleValue (separate bug, out of scope)
- A `make parity` target that runs `verify_bytecode_parity.go` and fails
  if MATCH drops below the current floor — prevents silent regressions

---

**Document created**: 2026-04-08
**Last updated**: 2026-04-08
