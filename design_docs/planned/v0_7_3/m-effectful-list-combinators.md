# M-EFFECTFUL-LIST-COMBINATORS: Effectful map/filter/fold for std/list

**Status:** Planned
**Version:** v0.7.3
**Priority:** P1 — most-requested DX gap from DocParse (54 hand-rolled recursive functions)
**Estimated effort:** 10-14 hours
**Origin:** DocParse DX Feedback (Feb 2026)

## Problem

`std/list` provides `map`, `filter`, `foldl`, `foldr` — but they are all `pure`. If a user needs to apply an effectful function over a list (e.g., call AI for each element, read a file per item), they must hand-write recursive traversals every time.

DocParse reported **54 hand-rolled recursive functions** that should have been stdlib calls:
- 12 mapping functions (effectful transform per element)
- 8 filtering functions (effectful predicate per element)
- 14 counting/aggregation functions (foldable)
- 9 concatenation functions (flatMap-like)

Example of what users write today:
```ailang
func describeImagesRec(blocks: [Block]) -> [Block] ! {AI} {
  match blocks {
    [] => [],
    [b, ...rest] => match b {
      ImageBlock(img) => {
        let desc = call("Describe this image: " ++ img.data);
        [ImageBlock({...img, description: desc})] ++ describeImagesRec(rest)
      },
      _ => [b] ++ describeImagesRec(rest)
    }
  }
}
```

What they should write:
```ailang
import std/list (mapE)

let described = mapE(\b. match b {
  ImageBlock(img) => {
    let desc = call("Describe: " ++ img.data);
    ImageBlock({...img, description: desc})
  },
  _ => b
}, blocks)
```

## Semantics

### Evaluation Order

**All effectful combinators evaluate elements left-to-right, sequentially.**

- `mapE(f, [a, b, c])` evaluates `f(a)`, then `f(b)`, then `f(c)` — in that order.
- No parallelism. No reordering.
- Determinism is preserved: same input + same effects = same observable behavior.

This is critical for AILANG's deterministic execution model. Effects + list traversal without an order guarantee would break replayability and trace reproducibility.

### Effect Row Transparency

The effect row of the result is exactly `{e}` — the combinator introduces no hidden effects. If the passed lambda has `! {AI}`, the combinator has `! {AI}`. If the lambda has `! {IO, FS}`, the combinator has `! {IO, FS}`.

### Budget Interaction

Effectful list combinators are **budget multipliers**: if `f` consumes `AI @limit=1` per call, then `mapE(f, xs)` needs `AI @limit=length(xs)`.

**How budgets work in AILANG (context for this design):**
- Budget tracking is dual: `used` (semantic, what's charged) and `physicalUsed` (actual calls)
- `CheckAndConsume()` in `internal/effects/budget.go` increments physical count per builtin call
- Budget scoping is per-function: callee's declared `@limit` is charged to caller on return
- On exhaustion: `BudgetExhaustedError` — hard fail, no partial results

**Guidance for users:**

1. **Size your budgets proportional to list length.** If processing N documents with AI, declare `@limit=N` (or `@limit=N+slack`).
2. **Use `take(n, xs)` to bound inputs** when budget is fixed: `mapE(f, take(10, docs))`.
3. **Budget exhaustion inside `mapE` is a hard error** — the traversal stops, no partial results returned. This matches AILANG's no-silent-fallback principle.

**Future (not v0.7.3):** A `mapEWithin(limit, f, xs) -> ([b], [a]) ! {e}` variant that returns processed results + remaining items on budget exhaustion. Requires `Result`-returning design — defer to v0.8.0+.

**Semantic vs physical counting:** `mapE` itself does not declare budgets — it threads the caller's budget scope. Each `f(x)` call inside `mapE` charges the **caller's** budget directly. There is no intermediate budget scope to reason about.

## Proposed API

Add effectful variants to `std/list` with `E` suffix, plus `forEachE` and pure `flatMap`:

```ailang
-- Effectful map: apply an effectful function to each element (left-to-right)
export func mapE[a, b, e](f: (a) -> b ! {e}, xs: [a]) -> [b] ! {e} {
  match xs {
    [] => [],
    [x, ...rest] => {
      let y = f(x);
      [y] ++ mapE(f, rest)
    }
  }
}

-- Effectful filter: effectful predicate per element (left-to-right)
-- Note: effectful predicates may be non-idempotent; ordering guarantee matters.
export func filterE[a, e](p: (a) -> bool ! {e}, xs: [a]) -> [a] ! {e} {
  match xs {
    [] => [],
    [x, ...rest] => {
      let keep = p(x);
      if keep then [x] ++ filterE(p, rest) else filterE(p, rest)
    }
  }
}

-- Effectful left fold (left-to-right)
export func foldlE[a, b, e](f: (b, a) -> b ! {e}, acc: b, xs: [a]) -> b ! {e} {
  match xs {
    [] => acc,
    [x, ...rest] => {
      let newAcc = f(acc, x);
      foldlE(f, newAcc, rest)
    }
  }
}

-- Effectful flatMap (concatMap): map then flatten (left-to-right)
export func flatMapE[a, b, e](f: (a) -> [b] ! {e}, xs: [a]) -> [b] ! {e} {
  match xs {
    [] => [],
    [x, ...rest] => {
      let ys = f(x);
      ys ++ flatMapE(f, rest)
    }
  }
}

-- Effectful forEach: execute effect per element, discard results
-- Prevents the anti-pattern: mapE(f, xs) just to force effects.
export func forEachE[a, e](f: (a) -> () ! {e}, xs: [a]) -> () ! {e} {
  match xs {
    [] => (),
    [x, ...rest] => {
      f(x);
      forEachE(f, rest)
    }
  }
}

-- Pure flatMap (also missing and useful)
export pure func flatMap[a, b](f: (a) -> [b], xs: [a]) -> [b] {
  match xs {
    [] => [],
    [x, ...rest] => f(x) ++ flatMap(f, rest)
  }
}
```

### Internal design note: `traverseE` as foundation

All effectful combinators are conceptually specializations of `traverseE`:
- `mapE` = `traverseE`
- `flatMapE` = `traverseE` + flatten
- `filterE` = `traverseE` with conditional accumulation
- `forEachE` = `traverseE` with unit return

We don't need to expose `traverseE` publicly, but the implementation can be structured around it internally to reduce future refactor surface if we need to optimize the traversal path.

## Design Decisions

### Naming: `mapE` vs `mapM` vs `traverse`

| Option | Precedent | Pros | Cons |
|--------|-----------|------|------|
| `mapE` | AILANG convention | Clear "effectful map" | Non-standard |
| `mapM` | Haskell | Familiar to FP devs | AILANG doesn't have monads |
| `traverse` | Haskell/Scala | Precise semantics | Unfamiliar to most users |
| `map!` | Ruby-ish | Visual bang for effects | `!` is effect syntax in AILANG |

**Recommendation:** `mapE` / `filterE` / `foldlE` / `flatMapE` / `forEachE`. The `E` suffix is consistent with AILANG's effect system terminology and won't confuse users into thinking about monads.

### Effect polymorphism

The key question: can `e` in `(a) -> b ! {e}` be polymorphic?

AILANG already supports effect polymorphism via row polymorphism. The signatures above should work because:
- Effect rows are structurally typed
- `! {AI}` is a concrete instance of `! {e}`
- `! {IO, FS}` composes naturally

**Risk:** Type inference for effect-polymorphic higher-order functions may need testing. The type checker must unify `e` with the actual effects of the passed lambda. Given earlier GAP-2/GAP-3 reports on lambda type inference, this is both a stdlib feature AND a typechecker regression test suite (see Phase 2 below).

### Where to put them

**Decision: In `std/list` itself**, alongside pure variants. Users import from one place.

`std/list` remains "functional" — effectful doesn't mean "impure", it just means the functions thread effect rows. Separating into `std/list/effect` would force users to remember two import paths for no meaningful architectural benefit.

### Stack Safety

**Current runtime facts:**
- No TCO in AILANG. Recursion guarded by depth counter (default: 10,000, configurable via `--max-recursion-depth`)
- Lists are Go slices (`[]Value`), not linked lists. Tail patterns (`[x, ...rest]`) create new `ListValue` wrappers.

**Implications:**
- `mapE` over a 10,000-element list hits the recursion limit. DocParse lists are likely small (hundreds of blocks at most), so this is acceptable for v0.7.3.
- `foldlE` is the most likely to hit large lists (aggregation over data). It is the priority candidate for future stack-safe implementation.

**v0.7.3 strategy:** Ship recursive definitions with documentation noting the depth limit. Add a comment in the stdlib source:

```ailang
-- NOTE: Recursive implementation. Stack-safe for lists up to ~10,000 elements.
-- For larger lists, increase --max-recursion-depth or use explicit recursion.
```

**Future (v0.8.0+):** Implement `foldlE` as a Go-backed builtin for true stack safety. Since lists are Go slices, the builtin can iterate with a `for` loop over `ListValue.Elements` — trivial to implement. Other combinators can then be defined in terms of `foldlE`.

## Implementation Plan

### Phase 1: Core combinators (4-6 hours)
1. Add `mapE`, `filterE`, `foldlE`, `flatMapE`, `forEachE` to `std/list.ail`
2. Add pure `flatMap` (currently missing, useful on its own)
3. Register exports in module manifest
4. Write inline tests (may hit test harness bug — use `ailang run` integration tests if needed)
5. Test with `! {IO}` effect (println per element)

### Phase 2: Type inference regression tests (3-4 hours)

This phase doubles as typechecker validation. Include 8 canonical tests:

6. `mapE` with `! {AI}` — single effect
7. `mapE` with `! {IO, FS}` — multi-effect row
8. `filterE` with `! {IO}` — effectful predicate
9. `foldlE` with lambda using `func(acc: T, x: U) -> T { ... }` syntax (reliable inference path)
10. `foldlE` with lambda using `\acc. \x. ...` syntax (known fragile path — document if broken)
11. Nested `mapE` inside `mapE` — effect row composition
12. `mapE` with ADT lambda (the DocParse use case: `Block -> Block ! {AI}`)
13. `forEachE` with unit-returning lambda

### Phase 3: Documentation (2-3 hours)
14. Update teaching prompt (prompts/v0.7.3.md) with effectful list section
15. Add example file `examples/runnable/effectful_list.ail`
16. Update `ailang docs std/list` output
17. Update CHANGELOG.md

## Risks

1. **Effect polymorphism in HOFs** — type checker may struggle with `{e}` row variable in lambda arguments. Mitigation: test early (Phase 2), fall back to concrete effect signatures if needed (e.g., separate `mapIO`, `mapAI` as temporary workaround).
2. **Test harness bug** — inline tests with stdlib imports still broken (M-DX25, M-DX-TEST-HARNESS-NIL). Mitigation: use `ailang run` integration tests.
3. **Monomorphization** — effectful polymorphic functions may hit specialization limits (16 per function, 512 per module). Mitigation: test with `--debug-compile`.
4. **Stack depth** — recursive impl limited to ~10,000 elements. Mitigation: document limit, plan Go builtin for v0.8.0+.
5. **Budget error UX** — users may not understand why `mapE` exhausts their budget. Mitigation: document budget multiplier behavior in teaching prompt and stdlib docstring.

## Acceptance Criteria

- [ ] `mapE(\x. { println(show(x)); x * 2 }, [1,2,3])` works with `! {IO}`
- [ ] `filterE(\x. { let s = call("Is " ++ show(x) ++ " prime?"); s == "yes" }, [1..10])` works with `! {AI}`
- [ ] `foldlE` accumulates with effects
- [ ] `flatMapE` maps and flattens with effects
- [ ] `forEachE(\x. println(show(x)), [1,2,3])` works (unit return, no result accumulation)
- [ ] Pure `flatMap` works without effects
- [ ] Type inference works without explicit effect annotations on lambdas
- [ ] Left-to-right evaluation order verified (e.g., `forEachE(println, ["a","b","c"])` prints a, b, c in order)
- [ ] Budget exhaustion mid-traversal produces `BudgetExhaustedError` (not silent truncation)
- [ ] 8 type inference regression tests pass (Phase 2)
- [ ] Teaching prompt updated with budget guidance
- [ ] Example file created and verified
- [ ] CHANGELOG.md updated

## Future Work (not v0.7.3)

- **`mapEWithin(limit, f, xs)`** — budget-bounded traversal returning `([b], [a])` (processed + remaining)
- **Go-backed `foldlE` builtin** — stack-safe for arbitrarily large lists
- **`mapAccumE`** — map while carrying state (stateful traversal)
- **Parallel variants** (`mapPar`) — when AILANG adds parallelism, effect-safe parallel map

## References

- DocParse DX Feedback (Feb 2026) — 54 recursive functions
- Haskell `Control.Monad` — `mapM`, `filterM`, `foldM`
- Effect handler literature — algebraic effects compose naturally with traversals
- AILANG budget system — `internal/effects/budget.go` (CheckAndConsume, scoped charging)
- AILANG recursion guard — `internal/eval/eval_operations.go` (RT_REC_003, default 10,000)
