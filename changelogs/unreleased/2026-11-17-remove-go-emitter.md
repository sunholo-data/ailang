### Removed — BREAKING: the Go emitter (`ailang compile --emit-go`, `--emit-go-v2`) (2026-11-17)

`ailang compile` translated AILANG to Go source. It is gone, together with everything that
existed only to serve it:

- `internal/gen/golang` (the `--emit-go` emitter), `internal/gen/emitgo` (the frozen
  `--emit-go-v2` Statement IR emitter) and `internal/gen/block` (a let-chain flattener only the
  emitter used).
- The Go codegen specs in `internal/builtins` (`GoCodegenSpec`, the `BuiltinMeta` registry,
  `StdlibIndex`, the `registry_codegen_*.go` tables).
- The public `github.com/sunholo-data/ailang/runtime` package, documented as the runtime for the
  Go backend. Nothing on this machine or in the published repos imported it.
- `tests/codegen-harness`, `make test-codegen`, `make test-sim-stub`,
  `make compile-examples-go`, `examples/sim_stub` and the "Test AILANG -> Go Codegen Pipeline"
  workflow. `DEBUG_CODEGEN` is no longer listed in help.

**Why.** v1.0.0 promises two engines that agree: the evaluator and the bytecode VM. The emitter
was a third engine in the dev tier. It boxed every value as `interface{}`, had no tail-call
elimination, and gave silently wrong answers that the other two engines did not (#2293, #2332,
#2369). Its output also varied from run to run (#2360). The only downstream user was
Stapledon's Voyage, which is deprecated.

**What to use instead.** `ailang run` (add `--bytecode` for the VM, or `--strict-bytecode` to
fail instead of falling back to the evaluator). A Go host runs the program as a child process
and exchanges JSON over stdin/stdout (see the Go Interop guide).

**What still answers.** `ailang compile` (and `ailang dev compile`) remains a hidden route that
exits 1 with a message naming `ailang run` and `--bytecode`, so an old script fails with that
message rather than "unknown command". `ailang compile --help` prints the same text and exits 0,
as `--help` does everywhere.

**Tests.** Tests that compared the evaluator, the VM and emitted Go keep their evaluator and VM
assertions and drop the Go leg: the by-name constructor tests, match exhaustiveness, rigid
names, the WAV corpus, the std renames transcript, and the `show` differential, which now
compares the evaluator with `--bytecode` and `--strict-bytecode`. `tests/golden/codegen` keeps
its `.ail` corpus because `tests/golden/bytecode` loads it by that path. The limitations pages
lose their seven Go-codegen-only rows.
