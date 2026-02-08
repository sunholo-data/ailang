# M-EFFECTFUL-LIST-COMBINATORS: Effectful map/filter/fold for std/list

**Status:** Planned
**Version:** v0.7.3
**Priority:** P1 — most-requested DX gap from DocParse (54 hand-rolled recursive functions)
**Estimated effort:** 8-12 hours
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

## Proposed API

Add effectful variants to `std/list` with `E` suffix (Haskell uses `M` for monadic, but AILANG uses effects not monads):

```ailang
-- Effectful map: apply an effectful function to each element
export func mapE[a, b, e](f: (a) -> b ! {e}, xs: [a]) -> [b] ! {e} {
  match xs {
    [] => [],
    [x, ...rest] => {
      let y = f(x);
      [y] ++ mapE(f, rest)
    }
  }
}

-- Effectful filter: effectful predicate per element
export func filterE[a, e](p: (a) -> bool ! {e}, xs: [a]) -> [a] ! {e} {
  match xs {
    [] => [],
    [x, ...rest] => {
      let keep = p(x);
      if keep then [x] ++ filterE(p, rest) else filterE(p, rest)
    }
  }
}

-- Effectful left fold
export func foldlE[a, b, e](f: (b, a) -> b ! {e}, acc: b, xs: [a]) -> b ! {e} {
  match xs {
    [] => acc,
    [x, ...rest] => {
      let newAcc = f(acc, x);
      foldlE(f, newAcc, rest)
    }
  }
}

-- Effectful flatMap (concatMap): map then flatten
export func flatMapE[a, b, e](f: (a) -> [b] ! {e}, xs: [a]) -> [b] ! {e} {
  match xs {
    [] => [],
    [x, ...rest] => {
      let ys = f(x);
      ys ++ flatMapE(f, rest)
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

## Design Decisions

### Naming: `mapE` vs `mapM` vs `traverse`

| Option | Precedent | Pros | Cons |
|--------|-----------|------|------|
| `mapE` | AILANG convention | Clear "effectful map" | Non-standard |
| `mapM` | Haskell | Familiar to FP devs | AILANG doesn't have monads |
| `traverse` | Haskell/Scala | Precise semantics | Unfamiliar to most users |
| `map!` | Ruby-ish | Visual bang for effects | `!` is effect syntax in AILANG |

**Recommendation:** `mapE` / `filterE` / `foldlE` / `flatMapE`. The `E` suffix is consistent with AILANG's effect system terminology and won't confuse users into thinking about monads.

### Effect polymorphism

The key question: can `e` in `(a) -> b ! {e}` be polymorphic?

AILANG already supports effect polymorphism via row polymorphism. The signatures above should work because:
- Effect rows are structurally typed
- `! {AI}` is a concrete instance of `! {e}`
- `! {IO, FS}` composes naturally

**Risk:** Type inference for effect-polymorphic higher-order functions may need testing. The type checker must unify `e` with the actual effects of the passed lambda.

### Where to put them

Two options:

1. **In `std/list` itself** — alongside pure variants. Users import from one place.
2. **In `std/list/effect`** — separate module to keep pure module pure.

**Recommendation:** Option 1, in `std/list`. The module header already says "Functional list operations" — effectful operations are functional too. Separating would force users to remember two import paths.

## Implementation Plan

### Phase 1: Core combinators (4-6 hours)
1. Add `mapE`, `filterE`, `foldlE`, `flatMapE` to `std/list.ail`
2. Add pure `flatMap` (currently missing, useful on its own)
3. Register exports in module manifest
4. Write inline tests (may hit test harness bug — use `ailang run` integration tests if needed)
5. Test with `! {IO}` effect (println per element)

### Phase 2: Validation (2-3 hours)
6. Test effect polymorphism: pass `! {AI}`, `! {IO, FS}`, `! {IO}` lambdas
7. Test with ADT lambdas (the DocParse use case)
8. Test with multi-effect lambdas
9. Verify type inference works without explicit annotations

### Phase 3: Documentation (2-3 hours)
10. Update teaching prompt (prompts/v0.7.3.md) with effectful list section
11. Add example file `examples/runnable/effectful_list.ail`
12. Update `ailang docs std/list` output
13. Update CHANGELOG.md

## Risks

1. **Effect polymorphism in HOFs** — type checker may struggle with `[e]` row variable in lambda arguments. Mitigation: test early, fall back to concrete effect signatures if needed.
2. **Test harness bug** — inline tests with stdlib imports still broken. Mitigation: use `ailang run` integration tests.
3. **Monomorphization** — effectful polymorphic functions may hit specialization limits. Mitigation: test with `--debug-compile`.

## Acceptance Criteria

- [ ] `mapE(\x. { println(show(x)); x * 2 }, [1,2,3])` works with `! {IO}`
- [ ] `filterE(\x. { let s = call("Is " ++ show(x) ++ " prime?"); s == "yes" }, [1..10])` works with `! {AI}`
- [ ] `foldlE` accumulates with effects
- [ ] `flatMapE` maps and flattens with effects
- [ ] Pure `flatMap` works without effects
- [ ] Type inference works without explicit effect annotations on lambdas
- [ ] Teaching prompt updated
- [ ] Example file created and verified
- [ ] CHANGELOG.md updated

## References

- DocParse DX Feedback (Feb 2026) — 54 recursive functions
- Haskell `Control.Monad` — `mapM`, `filterM`, `foldM`
- Effect handler literature — algebraic effects compose naturally with traversals
