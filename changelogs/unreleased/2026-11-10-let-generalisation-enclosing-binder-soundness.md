### Fixed — `let` no longer generalises a variable that an enclosing parameter owns (soundness)

`ailang check` accepted programs that fail at run time when a `let` inside a function re-bound,
or built a value from, a parameter whose type is a record or a type application:

```ailang
export func f(p: {name: a}) -> int {
  let q = p;
  q.name + 1
}
```

`f({name: "s"})` type-checked, then failed with `execution failed: missing dictionary method:
prelude::Num::String::add`.

The same hole existed for `p: Option[a]`, and for an unannotated lambda parameter whose fields a
`let`-bound lambda used (`\p. let g = \u. {name: p.name} in …`). Hindley-Milner generalisation may
only quantify a variable that is not free in the environment. The check that the environment owns a
variable was a separate walk from the one that collects the variables of the type being generalised.
It had no case for records (closed or open), record rows or type applications, and it read the
environment without the current substitution. Both questions now use one traversal
(`internal/types/free_vars.go`), and the environment is read under the substitution.
The open-record forms (`{name: a, ...}`, `{name: a | r}`) were opened by an unreleased change
(#2209) and are closed by the same fix.

**What you will see.** A program that relied on the hole is now rejected where the parameter is
used at a second type, for example:

```
failed to unify parameter 0: failed to unify record field 'name': cannot unify type constructors: int vs string
failed to unify parameter 0: failed to unify type argument 0: cannot unify type constructors: int vs string
No instance for Num[string] in scope
```

**Migration.** Every such program was already wrong: it crashed when it reached the mis-typed use.
Use the parameter's field at one type, or take a type parameter for each type you need. Measured
blast radius: no newly rejected file in this repository (725 `.ail` files across `examples/`,
`std/`, `benchmarks/` and `tests/`) or in 1,195 files across nine downstream checkouts. `ailang iface`
output for `std/` and `examples/` is unchanged.
