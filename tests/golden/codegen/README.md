# Golden program corpus (evaluator and VM)

Small `.ail` programs, one language feature each. The directory name is historical: these were
once compiled to Go by the Go emitter (`ailang compile --emit-go`), which has been removed. They
stay here because other tests load them by this path. `tests/golden/bytecode` runs them on the
bytecode VM (golden, probe and benchmark tests), so moving them would break that suite.

| File | Features |
|------|----------|
| `literals.ail` | int, float, bool, string, unit literals |
| `arithmetic.ail` | binary ops (+, *, >, &&), int/float arithmetic |
| `functions.ail` | identity, higher-order, recursion, let-chain |
| `fib.ail` | recursion (VM benchmark) |
| `adt_simple.ail` | nullary constructors, single-arg ADT, match |
| `adt_multiarg.ail` | multi-arg constructors, match with field extraction |
| `records.ail` | record type, literal, field access, record update |
| `lists.ail` | empty list, cons (::), list pattern match |
| `if_else.ail` | if/else, chained if-else, multi-arg functions |
| `match_patterns.ail` | recursive ADT, nested match, literal patterns |
| `let_bindings.ail` | nested lets, variable shadowing |
| `string_ops.ail` | string concatenation (++), string equality |
| `tuples.ail` | tuple creation, tuple pattern matching, wildcard |
| `show_differential*.ail` | `show` output; `golden_test.go` requires the evaluator and the VM to print the same bytes |

The Go tests in this directory:

- `golden_test.go`: builds `ailang` from this checkout and runs the `show_differential*.ail`
  fixtures on the evaluator, `--bytecode` and `--strict-bytecode`. Output must be identical.
- `byname_ctor_test.go` and `byname_closure_test.go`: constructor patterns matched by name.
  Programs that leave the type unfixed are rejected on every path. Programs that fix it run
  identically on the evaluator and both VMs.
