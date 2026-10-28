### Changed — Teaching prompt v0.16.8: cost model, std/regex, `range` arity (2026-10-28)

`ailang prompt` now serves v0.16.8. v0.16.7 is frozen as legacy and still served byte-for-byte for pinned runs. The new version was cut instead of amending v0.16.7 in place because a baseline eval was running on a binary that embeds v0.16.7. The changes come from the 2026-10-28 competitive-programming spike, in which models used `std/array` in 1 of 17 solutions and `std/map` in 0 of 17:

- **Cost model.** A "Cost Model" section replaces the short "Arrays (O(1) indexed access)" section. It gives the cost of each list, `std/array` and `std/map` operation: `nth`/`length`/matching are O(1); `::` and `++` copy; `A.set`, `A.append`, `M.insert` and `M.remove` copy, so each is O(n); `updateMany` and `fromList` are O(n + k). It states that one insert, set or append per loop step is O(n²), with the measured cost (20,000 `M.insert`s take 5.25 s, about ×4 per doubling). It then lists the alternatives that build the structure once, plus a one-line hint to use Floyd cycle detection.
- **std/map facts.** The prompt now states that `insert` copies the map and that `fromList` keeps the last duplicate. It states that keys must be int, string or bool, because a tuple key type-checks and then fails at runtime. It also states that int keys iterate in lexical order (`[10, 2, 9]`).
- **std/regex.** The module was missing from the prompt. It now has an import line and a short verified section, which notes that `"\d"` is a parse error and the literal must be written `"\\d"`.
- **`range(start, end)`.** `range(n)` recurred in the spike as a `TC_ARITY_001` failure. The two-argument form is now stated in the import block, the list-function list and the "What AILANG Does NOT Have" table.
- **`mapAccumL`**, the linear way to build a list, is now in the prompt.
- **Cuts.** The hand-rolled recursive `map` example (it conses on every step and shadows `std/list.map`) and a duplicated `fromBase64URL` bullet are removed.

Size: 99,175 → 103,093 bytes (+3,918). Every new snippet passes `ailang check` and `ailang run`.
