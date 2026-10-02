### Fixed — float `<` `<=` `>` `>=` follow IEEE 754 on both engines (#1419)

- On the interpreter, `nan > 1.0`, `nan >= 1.0`, `1.0 < nan` and `1.0 <= nan` returned `true`. The VM
  returned `false` for all four. The cause was the interpreter's `Ord[Float]` dictionary, which used a
  total order with NaN as the greatest value. Both engines now return `false` for any ordered comparison
  with a NaN operand, through direct operators, monomorphic helpers, lambdas and generic functions alike.
- Effects you can see in code:
  - A range guard `if x > hi then hi else x` now lets a NaN through on the interpreter, as it already did
    on the VM. It used to clamp the NaN to `hi`.
  - `maximumFloat`/`minimumFloat` with NaN in the list, and `sortBy` with a comparator built from `<`/`>`,
    now give the same result on the interpreter as on the VM.
- One named rule, `types.FloatLt/FloatLte/FloatGt/FloatGte`, alongside `types.FloatEq`. It is used by the
  dictionary, both builtin registries, the evaluator's binop paths and the VM's `OpLt`/`OpLe`. The
  dictionary's `min`/`max` now propagate NaN. No surface syntax reaches them today.
- To detect NaN, use `std/math.isNaN` (or `x != x`). An ordered comparison never detects it.
- `examples/float_nan.ail` gains ordered-comparison checks. The teaching prompt (v0.16.6, amended in
  place) and `docs/LIMITATIONS.md` now state the rule.
