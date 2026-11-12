### Added — Gemini 4 Hydrogen in the eval registry (opt-in)

`gemini-4-hydrogen` is callable on Vertex since 2026-11-12. It is registered at $1.50/$7.50 per
1M with cached input at $0.075 and no introductory schedule. That is the rate 3.7/3.8 Flash move to
on 2027-01-01. Its thinking is on by default and heavy; `thinkingBudget` and `MINIMAL` are
rejected, so the row sets no reasoning knobs.

It was placed in one run with 3.7 and 3.8 Flash, since neither had an anchored rating, across
81 benchmarks at `--trials 1`:

| Model | Passed | Anchored ELO |
|---|---|---|
| 3.8 Flash | 77/81 | 2251 |
| Hydrogen | 76/81 | 2179 |
| 3.7 Flash | 75/81 | 2127 |

All three ratings are banked. Four of Hydrogen's failures are `import std/map (Map)` and
`import std/array (Array)`: `std/set` and `std/option` export their type, but these two modules
do not. Hydrogen is in no suite yet.
