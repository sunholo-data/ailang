### Fixed — `func f(())` checks again (#2156); a parameter that is not a name is `PAR_PARAM_NAME`

`func f(()) -> int = 1` failed `ailang check` with `interface build error … IFC export
analysis: export f: missing IFC parameter name`, even though `f` was not exported. The cause was
in the parser, not the IFC pass: AILANG has no pattern parameters, and `parseParams` built a
parameter for *any* token, filling in a name and type only for an identifier. Every other token
became a nameless, untyped parameter — on v0.47 `func f(5) -> int = 1` checked clean and
`f("hello")` returned `1`.

- `()` in a parameter list is now the ignored unit parameter, the same node as `func f()` and
  `func f(_: ())`. `func f(())` checks and runs again.
- Any other non-name parameter is a parse error, in function declarations and `func(...)`
  literals alike:
  `PAR_PARAM_NAME at f.ail:2:8: parameter must be a name, got `5``, with the suggestions
  `name` / `name: Type`, `_: ()`, and "match on the argument in the body".
- An IFC summary error on a private function now says `function f`, not `export f`.

**Breaking:** a parameter written as a literal, a tuple pattern (`(a, b): (int, int)`) or any other
non-name token no longer parses. For a top-level function this already failed `check` since
v0.57.x (the IFC error above). A `func(...)` *literal* was not covered by that error: on v0.58.0
`let g = func(7) -> int { 1 }` checks and `g(3)` returns `1`; it is now `PAR_PARAM_NAME`. Measured:
no file in `examples/`, `std/`, `benchmarks/` or `tests/` changes its `check` result (725 files);
the downstream corpora are in the PR. Migration: name the parameter (`x: T`, `_: ()`), and
destructure in the body.
