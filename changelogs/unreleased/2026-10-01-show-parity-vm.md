### Fixed — `show` and printed results are identical on the bytecode VM and in the interpreter (#1453)

- The VM printed ADTs as `<adt#0 6>` (`show(Some(6))`, or an entry returning `Some(6)`), whole floats inside
  them as `2` instead of `2.0`, and closures as `<closure>`. It also skipped the interpreter's depth limit
  (`[[[[...]]]]`) and its 80-column elision, so long lists and records rendered differently too.
- **One renderer.** `builtins.RenderShow` makes every formatting decision; the evaluator and the VM each only
  describe their values to it. The two hand-written copies cannot drift apart again.
- **VM ADTs carry their constructor name.** The compiler puts it in a pseudo-`LOAD_CONST` after `MAKE_ADT`,
  the encoding `UPDATE_RECORD` already uses for field names, and image validation requires it.
  `bytecode.NewADT` now takes the name, so every construction site supplies one. Equality and pattern
  dispatch still use the ordinal.
- Printing a VM entry's return value goes through `vm.BytecodeToEvalForDisplay`. The bridge still refuses to
  hand VM ADTs to the evaluator, since they carry no type.
- `TestShowParityVMvsInterpreter` runs 13 entries on both engines (strict VM). Against the previous binary,
  at least 8 of them differ (those 8 were checked).
- **REPL `show` too.** The REPL/simple-evaluator environment had a third `show` (and `toText`) that quoted
  strings and spelled floats with `%g`: `show({a: "x"})` was `{a: "x"}` in the REPL but `{a: x}` under
  `ailang run`. The renderer now lives in `internal/eval` (`eval.Show`) and every path uses it; the legacy
  copy and the tests that pinned its output are gone.
