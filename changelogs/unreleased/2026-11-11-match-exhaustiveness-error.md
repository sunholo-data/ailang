### Changed — BREAKING: a `match` that misses a case is now a compile error (TC_MATCH_002) (2026-11-11)

`ailang check` and every engine path (`run`, `run --bytecode`, `run --strict-bytecode`,
`compile --emit-go`, `test`) reject a `match` whose unguarded arms do not cover every value of
the scrutinee's type — ADTs (every constructor of the type's own declaration, not the ones a
module imports; identified by defining module, so two modules' `T` or `C` never mix), `Option`/
`Result`, lists, tuples, records, bool, and int/float/string literals without a catch-all. The
error names concrete missing patterns (`Blue`, `Some(Err(_))`, `[]`, `(true, false)`,
`_ (any int other than 0, 1)`) with the match's file:line:col in human, `--json` and
`--format agent` output; `check --json` lists every such error of the module, not only the first.
Arms with `if` guards do not count toward coverage. Fix: add the named arm, end with `_ => …`, or
write `C => unreachable("why")` for a case that cannot happen. Previously these programs passed
`check` and failed at run time with RT002.
`check` still does not type-check named `test` blocks (#2232); `ailang test` reports the error.

A constructor matched by name — neither it nor its type imported — against a value whose type
nothing fixes (e.g. in a let-bound lambda) is `TC_MATCH_002` "cannot be checked": such a pattern
constrains nothing, so other values reach the match. Fix: import the constructor's type.

Also: a constructor pattern with the wrong number of arguments, `C(n)` for `C(int, int)`, is
`TC_MATCH_006` in every checked match, a catch-all arm or not (the engines disagreed on it,
#2243); so is a constructor matched by name against an untyped value when two loaded types declare
it with different arities (fix: import the type you mean); a match too large to check is `TC_MATCH_004` (fix: a final `_` arm). The elaborator's old literal-only warning (ELB004) and its checker are
gone. The teaching prompt line "`ailang check` does NOT catch it" is removed. Compile-cache key
v13: modules cached by an earlier compiler must be re-checked. Design:
design_docs/planned/v1_0_0/m-v1-match-exhaustiveness.md (#2146).

### Added — `unreachable(msg)` and warning TC_MATCH_003 (2026-11-11)

`unreachable(msg: string) -> a` is a pure, always-available builtin that never returns: evaluating
it stops the program with `RT010: unreachable code reached: <msg>`, identically on the evaluator,
the bytecode VM and generated Go. It is the body for a match case that cannot happen, so a broken
invariant fails loudly instead of returning a dummy value. `std/result.unwrap` now uses it on
`Err` (it divided by zero before, RT001).

An arm that no value can reach, because the unguarded arms before it match everything it matches,
is warning `TC_MATCH_003`, in `check --json` `warnings` and as a `warn:` line in
`--format agent`. It is stored with the module in the compile cache, so a warm `check` reports it
too.

### Changed — BREAKING: a module-level definition or import named `unreachable` is refused (TC_BUILTIN_SHADOW_001) (2026-11-11)

A builtin wins over a module-level name, so a program that defined its own `unreachable`, or
imported something under that name (`import M (x as unreachable)`), would now silently call the
new builtin instead. `check` refuses both with `TC_BUILTIN_SHADOW_001` (rename it), also when the
module is served from the compile cache; a local `let unreachable = …` keeps ordinary lexical
scope. No definition of that name exists in the 10,401 eval programs or the downstream corpora.
