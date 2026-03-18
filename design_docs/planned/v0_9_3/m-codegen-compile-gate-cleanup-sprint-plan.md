# Sprint Plan: M-CODEGEN-COMPILE-GATE-CLEANUP

## Summary

| Field | Value |
|-------|-------|
| **Goal** | Fix remaining 35 compile gate failures, push pass rate from 73% to 90%+ |
| **Duration** | 1 day (4-5 hours) |
| **Risk Level** | Low — fixing existing bugs, no new architecture |
| **Total LOC** | ~250 |
| **Target** | v0.9.3 |

## Current Status: 94/129 passing (73%)

### Failure Categories

| Category | Examples | Error Pattern | Fix Approach |
|----------|----------|---------------|-------------|
| **Unused imports in module files** | directory_listing, json_array_extraction, json_helpers, effectful_list_t8, polymorphic_adt, string_parsing, math_trig, json_basic_decode (partial) | `"strings"/"strconv" imported and not used` | Conditional imports in Generate() like GenerateRuntime() |
| **ADT type redeclarations** | stream_multi_source, stream_process_source, stream_sse, stream_websocket, process_demo, process_stdin_write | `StreamConn redeclared in this block` | Types from stdlib imports (std/stream, std/process) duplicated in types.go |
| **Undefined: Map in runtime** | effectful_list (4 examples) | `undefined: Map` in runtime.go | Map helper not lazily registered when only used by other helpers (ForEachE body calls Map) |
| **Pattern match variable scoping** | list_pattern_cons, pattern_sugar, record_patterns, recursion_quicksort, cli_args_demo, batch_processing | `undefined: y`, `undefined: name`, `undefined: b` | Variables bound in match arms not in scope in generated code |
| **Missing builtins** | list_extremes, debug_effect, conway_grid, string_repeat, string_split | `undefined: MaximumString`, `undefined: Check` | Add remaining registry specs |
| **Type assertion issues** | array_basic, list_pattern_records, record_update, json_basic_decode | `cannot use ... as ... value` | Interface{} not asserted to concrete type |
| **Effect handler bugs** | effect_budgets_multi, effect_budgets_rand | `SetSeed (no value) used as value`, `too many arguments` | Handler codegen arg/return mismatch |
| **Declared and not used** | string_chars | `_elem0 declared and not used` | Pattern binding generates unused vars |

## Milestones

### M1: Conditional Imports in Module Files (~30 LOC)

Fix `Generate()` in codegen.go to only import strings/strconv/sort when the generated module code uses them — same pattern already applied to `GenerateRuntime()`.

**Examples fixed**: directory_listing, json_array_extraction, json_helpers, effectful_list_t8, polymorphic_adt, string_parsing, math_trig (~7 examples)

**Acceptance criteria**:
- No "imported and not used" errors in any example
- Generate() uses conditional import detection matching GenerateRuntime()

### M2: Map Helper Dependency Chain (~20 LOC)

When `ForEachE` or `MapE` helpers are registered, their bodies call `Map` — but `Map` isn't automatically registered. Fix: when registering a helper whose body references other helpers, also register those dependencies.

Simple approach: add `Map` to the eager-emit set (it's used by enough helpers that it should always be present).

**Examples fixed**: effectful_list, effectful_list_t1_mapE_basic, effectful_list_t7_chain_combinators, effectful_list_t8_string_list (4 examples)

**Acceptance criteria**:
- All effectful_list examples pass compile gate
- Map is always present in runtime.go when any list helper is emitted

### M3: ADT Type Deduplication (~40 LOC)

Types from stdlib imports (StreamConn, StreamSource, ProcessHandle) are generated as ADT structs in types.go AND as record structs — causing redeclaration. Fix: deduplicate type generation by tracking which types have already been emitted.

**Examples fixed**: stream_multi_source, stream_process_source, stream_sse, stream_websocket, process_demo, process_stdin_write (6 examples)

**Acceptance criteria**:
- No "redeclared in this block" errors
- Stdlib-imported types appear exactly once in types.go

### M4: Missing Builtin Specs (~30 LOC)

Add remaining specs: MaximumString, MinimumString, Check (debug effect), AbsInt (already added but not resolving for math_trig), string_repeat, string_split.

**Examples fixed**: list_extremes, debug_effect, math_trig, string_repeat, string_split, conway_grid (partial) (~5 examples)

**Acceptance criteria**:
- All listed symbols have registry specs
- list_extremes and math_trig pass compile gate

### M5: Verify and Report (~10 LOC)

Run `make compile-examples-go` and confirm 90%+ pass rate. Update the compile gate sprint JSON. Send results to docparse.

**Acceptance criteria**:
- Pass rate >= 116/129 (90%)
- Results documented

## Deferred (Not in This Sprint)

These require deeper investigation and are tracked separately:

- **Pattern match variable scoping** (6 examples) — variables bound in match arm conditions not visible in match arm bodies. This is a codegen_match.go logic bug needing careful analysis.
- **Type assertion issues** (4 examples) — interface{} values not asserted to concrete types in ADT constructor calls and record literals. Needs type analysis improvements.
- **Effect handler bugs** (2 examples) — handler codegen producing wrong signatures. Needs effects.go changes.
- **Unused pattern bindings** (1 example) — string_chars generates unused temp vars in tuple patterns.

These 13 examples would require ~2-3 hours of individual investigation each.
