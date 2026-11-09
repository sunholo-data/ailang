### Fixed — every expression in a named test body is a check; a false non-final check no longer passes (#604) (2026-11-09)

`ailang test` graded only the LAST expression of a `test "name" { … }` body. Every earlier bare
expression was bound to a dead `_seq` and its value thrown away, so

```ailang
test "x" {
  add_one(1) == 99;
  add_one(1) == 2
}
```

reported `All tests passed!` and exited 0. Swapping the two lines failed it: the verdict depended
on position.

Now every top-level bare expression in a named test body is a check, numbered together with
`assert`s, left to right. The body passes only if every check is true; evaluation stops at the
first false one, and the failure names it with the source you wrote and its line:
`` check 1 failed: `(add_one(1) == 99)` (at x.ail:8:14) ``. It holds on the evaluator and under
`--bytecode`. Bodies with no bare expression before the last (one expression, or `let`s plus a
final expression) fold exactly as before. Every named test tracked in this repo reports the same
result as before the fix (36 files, 111 named test blocks, swept with both binaries).

**Breaking change.** A non-final bare expression must now be a `bool`. One that only computes a
value, such as `add_one(1);`, used to be silently discarded and is now a type error
(`cannot unify type constructors: int vs bool`). To migrate, bind it (`let _ignored = add_one(1);`)
or delete it. An effectful non-final expression (`println("hi");`) was already an error and still
is; its message is now a type error instead of an effect error.

**Not fixed: nested blocks ([#614](https://github.com/sunholo-data/ailang/issues/614)).** The rule
covers the top level of the body only. Inside a check, a block `{ a; b }` still evaluates `a` and
discards it, so `test "t" { if true then { add_one(1) == 99; add_one(1) == 2 } else false }` still
passes. `internal/testing/testdata/assert/nested_block_residual.ail` pins that behaviour until #614
lands.

Design: `design_docs/planned/v0_33_1/m-named-test-body-check-semantics.md`. Code:
`internal/testing/test_body_lowering.go` (`FoldTestBody`). Docs: "Several checks in one test" in
`docs/docs/guides/testing.md`.
