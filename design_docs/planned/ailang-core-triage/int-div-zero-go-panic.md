# Integer division by zero is a raw Go panic, not a structured error (#1449, dup #1528)

- **Date**: 2026-10-02
- **Class**: bug
- **Recommend**: design-doc
- **Searched**: `division by zero`, `div.?zero`, `DIV_ZERO`, `RT_DIV0`, `RT001` over `design_docs/` and `internal/`

Reproduced on a dev build: `(10 / n)` with `n == 0` reaches the `Num[int].div`
dictionary method in `internal/types/dictionaries.go` (`registerNumInt`), which
calls `panic("division by zero")`; `internal/eval/eval_patterns.go`
(`wrapDictionaryMethod`) has no error path for it, so `ailang run` dies with
`panic: ... [recovered, repanicked]` and the REPL exits. Integer `%` takes a
different route (`mod_Int` builtin) and already returns `[RT_DIV0] Modulo by
zero` — no position — while the VM says `division by zero` / `_mod_Int:
division by zero`, and the registered code for this concept (`RT001`, "Division
by zero", `internal/errors/codes.go`) is emitted nowhere. So the fix spans
`internal/types`, `internal/eval`, `internal/builtins`, `internal/vm`,
`internal/repl` and `internal/errors` (rubric row 5), needs a code choice
between the unregistered `RT_DIV0` and the registered-but-unused `RT001` (row
3), and needs a position-attachment mechanism. No existing design doc covers
it: the only hits are archived eval analyses and a 2025 float-equality note.
