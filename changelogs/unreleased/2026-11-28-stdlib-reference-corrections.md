### Fixed — stdlib reference: qualified `S.Set[a]` works, and fuller function lists

- `docs/docs/reference/stdlib.md` said "`S.Set[...]` does not parse". It does: `import std/set as S`
  with `s: S.Set[int]` type-checks and runs on v0.64.1, and so does `M.Map[k, v]`.
- The `std/list`, `std/option` and `std/result` rows now list the exported functions that
  were missing, among them `findIndex`, `any`, `takeWhile`, `dropWhile`, `sortBy`, `flatMap`,
  `zipWith`, `isSome`/`isNone` and `isOk`/`isErr`. The rows link `list_combinators.ail`.
