### Fixed — bytecode VM match lowering: nested patterns, constructor sub-patterns, guards and catch-all arms now match the evaluator (2026-10-02)

The Statement IR lowering of `match` (`internal/gen/lower`) handled only a flat
subset of patterns, while the evaluator matches every pattern kind at any depth.
Under `ailang run --bytecode --strict-bytecode` that showed up as five bug families,
all now fixed by one recursive lowering that mirrors `matchPattern`
([m-vm-match-lowering](../../design_docs/implemented/v0_51_1/m-vm-match-lowering.md)):

- **Nested patterns were evaluator-only** — `a :: b :: _`, `_ :: x :: _`, `[x :: y, _]`,
  `Some(Some(x))`, `{user: {email: e}}`, `(x :: _, z)` failed with
  `compiler: unbound variable` (#1420, #1505, #1517).
- **Literal and nested sub-patterns were silently wrong** — `[1.0, 2.0]`, `(1, y)`,
  `{name: "alice"}`, `[1, 2, ...r]`, `x :: y :: []`, `"a" :: rest` matched inputs they
  should not, returning the wrong arm with exit 0.
- **Constructors in list/cons/tuple positions crashed the VM** — `Some(x) :: rest`,
  `[Some(x)]`, `TText(t) :: r` failed with `_record_get: arg 0 must be record, got ADT`
  (#1503). Tag checks now use a dedicated `stmt.ADTTagEq` node compiled to the switch
  path's own `GET_TAG` + ordinal `EQ`; no VM opcode or builtin changed.
- **Guards ran before their arm's bindings existed** — `x :: rest if x > 5`,
  `[a, b] if a > b`, `{x: v} if v > 3`, `other if other == "b"` were evaluator-only.
  Bindings are now in scope for the guard, and a false guard falls through to the next arm.
- **Variable catch-all arms** — `other => describe(other)` in a constructor match was
  evaluator-only (unbound `other`); a catch-all before a constructor arm, or a guarded
  catch-all, silently returned a different arm's value; an all-variable match failed
  with `unknown ADT ""` (#1473).

A disassembly sweep of `examples/`, `std/` and `tests/golden/` drops from 2575 to 2487
evaluator-only prototypes with no new ones (e.g. `std/jwt.getClaimInt`,
`std/net.httpGet`, `pattern_sugar.sumThree`, `list_pattern_cons.secondElement`).
New parity gate: `TestCLI_RunBytecode_MatchLoweringParity` over
`tests/golden/bytecode/nested_patterns.ail` (42 shapes); new example
`examples/runnable/nested_patterns.ail`.
