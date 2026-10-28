### Changed — docs and CLI help say what the bytecode VM and Go codegen actually do

- `why-ailang`, `intro`, `vision` and `implementation-status` no longer claim AILANG "compiles to native Go performance". They describe the bytecode VM (`ailang run --bytecode`, about 40× faster than the evaluator) and the evaluator as the reference engine.
- The Go interop guide documents the child-process pattern (`ailang run --bytecode`, NDJSON over stdin/stdout, as stapledons-godot uses it). It states that `--emit-go` is not a speed path: values are boxed as `interface{}`, and there is no tail-call elimination, so a 10M-step tail loop overflows the Go stack. It also notes that `internal/embed` cannot be imported from another Go module.
- `ailang run --bytecode` help says it falls back to the evaluator silently. `ailang compile --help` lists `--emit-go-v2` and warns about tail calls.
- LIMITATIONS: new entries for "Go codegen has no tail-call elimination" and "`--bytecode` silently falls back; the VM is ~4× slower than CPython", with numbers measured 2026-10-28.
- The debugging guide's memory advice is corrected. Tail-recursive loops now free per-iteration bindings: 329 MB on the evaluator and 53 MB on the VM, against 870 MB before tail calls shipped.
- `ARCHITECTURE.md` describes the VM pipeline. M-PERF4 is marked superseded, and M-GAME-ENGINE-EFFECTS gets a note on the stapledons-godot architecture. New planned doc: `design_docs/planned/v0_55_0/m-vm-quick-wins.md`.
