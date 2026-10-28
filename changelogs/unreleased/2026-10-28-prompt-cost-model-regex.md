### Changed — Teaching prompt v0.16.8: cost model, std/set, std/regex, `range` arity (2026-10-28)

`ailang prompt` now serves v0.16.8. v0.16.7 is frozen as legacy and is still served byte-for-byte for pinned runs. The new version was cut instead of amending v0.16.7 in place because a baseline eval was running on a binary that embeds v0.16.7. The changes come from the 2026-10-28 competitive-programming spike, in which models used `std/array` in 1 of 17 solutions and `std/map` in 0 of 17:

- **Cost model.** A "Cost Model" section replaces the short "Arrays (O(1) indexed access)" section.
  - It gives the cost of each list, `std/array`, `std/map` and `std/set` operation:
    - O(1): `nth`, `length`, matching, `member`.
    - A copy of the whole structure: `::` and `++`; `A.set`, `A.append`, `M.insert`, `M.remove`, `S.insert` and `S.remove` are O(n).
    - O(n + k): `fromList`, `updateMany` and `accumulate`.
  - It states that one insert, set or append per loop step is O(n²) overall, with the measured cost (20,000 `M.insert`s or `S.insert`s take about 5 s, about ×4 per doubling).
  - It then lists the alternatives that build the structure once. `A.accumulate` covers difference arrays and histograms. A visited-set bullet says `S.member` is O(1), that `S.insert` per state is fine for a few thousand states, and that beyond that you add one whole BFS level at a time with `S.union`; a deterministic walk can use Floyd cycle detection instead. It also points to the `visited_set_bfs`, `cycle_detection_visited_set`, `map_tuple_record_keys` and `difference_array_accumulate` examples.
- **std/map and std/set.**
  - `insert` copies the map, and `fromList` keeps the last duplicate.
  - Keys may be tuples, records and ADT constructors as well as int, string, bool, float and unit; a function or list key is a runtime error.
  - Int keys iterate in lexical order (`[10, 2, 9]`).
  - New import lines teach `import std/set as S`, plus `import std/set (Set)` to write the type, because `S.Set[...]` does not parse.
- **std/regex.** The module was missing from the prompt. It now has an import line and a short verified section, which notes that `"\d"` is a parse error and the literal must be written `"\\d"`.
- **`range(start, end)`.** `range(n)` recurred in the spike as a `TC_ARITY_001` failure. The two-argument form is now stated in the import block, the list-function list and the "What AILANG Does NOT Have" table.
- **`mapAccumL`**, the linear way to build a list, is now in the prompt.
- **Two wrong claims corrected.**
  - The prompt called `foldl(\acc. \x. acc + x, 0, xs)` an arity error. It works on both engines. The prompt now recommends `\a b. body` for callbacks as the simplest form and says the curried form also works.
  - The "mutual recursion also works" example, which used two local `letrec`s, failed `ailang check` with `undefined variable: isOdd`. It is replaced by top-level mutually recursive `func`s.
- **Cuts.** The hand-rolled recursive `map` example (it conses on every step and shadows `std/list.map`) and a duplicated `fromBase64URL` bullet are removed.
- **Examples.**
  - `examples/regex_basics.ail` moves to `examples/runnable/regex_basics.ail`. It now also covers `findAll`, capture groups, `replaceAll` with `$1`, the `"\\d"` escape and the qualified import.
  - The new `examples/runnable/bulk_build_cost_model.ail` contrasts per-step `set`/`insert` (quadratic) with building the structure once.
  - Both are in the manifest with expected output, both are on the website Examples page, and both are linked from the stdlib reference. The stdlib reference no longer says that lists are O(n) to index: `nth` is O(1).

Size: 99,175 → 104,343 bytes (+5,168). Every new snippet passes `ailang check`, and `ailang run` gives the same output on the evaluator and with `--strict-bytecode`.
