# Integer division by zero is a raw Go panic, not a typed runtime error (#1449)

- **Date**: 2026-10-02
- **Class**: bug
- **Recommend**: design-doc
- **Searched**: `division by zero`, `DIV_ZERO`, `panic` across `design_docs/` (all hits are archives/completion reports, none ruling on typed runtime errors); `none found` in `design_docs/planned/` for a typed-runtime-error or RT_* error-code design doc.
- **Status**: triage/record only — the directive states this is being FIXED in an attended session; this row records the mechanism for that session, it does not queue new work.
- **Upstream**: https://github.com/sunholo-data/ailang/issues/1449

## Mechanism (reproduced on v0.51.0)

`let n = 0; 10 / n` under `ailang run` dies with `panic: division by zero [recovered, repanicked]` and a full Go stack. The panic originates in the numeric dictionary:

- `internal/types/dictionaries.go:129` (`registerNumInt`, the `div` method) — literal `panic("division by zero")` on the dictionary-passing eval path, reached via `internal/eval/eval_patterns.go` (`wrapDictionaryMethod` → `evalDictApp`), then re-panicked at `internal/runner/run.go:632` (otel span recover/re-panic).

Handling is inconsistent across the other integer-division call sites:

- `internal/eval/eval_operations.go:408,480`, `eval_simple.go:440,447` — plain `fmt.Errorf("division by zero")`, no code, no position.
- `internal/eval/eval_typed.go:642,649` — same error but appends `at %s` location as a bare string.
- `internal/vm/vm.go:627`, `internal/vm/builtins_math.go:90` (`_mod_Int`) — plain string errors, no code, no position.
- `internal/repl/repl_eval.go:311` — panics exactly like the dictionary path.

Under serve-api the panic reaches request goroutines; `serveapi/protocol/bearer_gate.go:93` recovers and fails closed, but nothing guarantees that for other call sites — which is why a typed error is wanted rather than more `recover()` sprinkling.

## Why design-doc

An `RT_*` convention already exists (`RT_REC_001/002/003`, documented in `docs/docs/reference/limitations.md`), so `RT_DIV_ZERO` fits — but "with position in eval and VM" means threading position through dictionary method dispatch and VM error propagation, changing the message format/exit behavior surfaced to users, and reconciling 4+ divergent call sites (dictionaries, repl_eval, eval_*, vm). That is a public-surface/semantics change spanning many files (rubric rows 4–6), not a 2-line/1-file fix. There is also more than one acceptable shape for the typed error (string-code error vs structured error carrying position vs an effect-typed failure), which is row 3 territory on its own.

The attending fix session should still route through a short design doc to pin: the error type, where position is attached, whether `_mod_Int`/other arithmetic paths get the same treatment, and how serve-api surfaces it.