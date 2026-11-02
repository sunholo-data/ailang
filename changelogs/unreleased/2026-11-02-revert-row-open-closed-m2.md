### Fixed — valid callbacks rejected with "carries labels … which the closed row {} does not permit"

v0.56.0 was tagged but never published: its "open rows cannot lose their labels" check (#1958 M2)
rejected valid programs and is reverted here. Inference closes a lambda's effect row to `{}` when
its body combines two effectful calls (a `let` of an effectful argument, a sequence, an `if`, a
`match`, a list or tuple), and the new check then refused that lambda for a concrete callback slot:

```ailang
func withDir(f: string -> int ! {IO}) -> int ! {IO} = f("d")
export func a() -> int ! {IO} = withDir(\d. ex(p(), 1))   -- rejected by v0.56.0
```

Behaviour for this path is back to v0.55.0: an annotated IO callback passed to a closed pure slot
such as `std/list.map` is again accepted (the soundness gap #1958 tracks). The explicit-empty
callback contract (`! {}` requires a pure value) is unaffected. M2 returns once inference keeps
those rows open; the regression cases are in `effect_row_open_closed_test.go`.
