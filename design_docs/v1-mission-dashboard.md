# V1 Mission Dashboard

> 30-second control context. Snapshot, NOT a record — history lives in the charter STATUS block,
> `v1-mission-log.md` and `v1-mission-status-archive.md`. Overwritten every iteration.
> **Namespaced path**: `design_docs/v1-mission-dashboard.md`. Never write the bare
> `mission-dashboard.md` — that literal is shared by every mission on the rig.

**Updated**: 2026-08-21 (iteration 242) · **Release**: v0.33.1 · **Bookkeeping**: [#745](https://github.com/sunholo-data/ailang/issues/745)

## Just landed

- **`m-stdlib-reverse-delegates-to-builtin`** — PR [#814](https://github.com/sunholo-data/ailang/pull/814) →
  [`728ca8f3e`](https://github.com/sunholo-data/ailang/commit/728ca8f3e). `std/list.reverse` now
  delegates to `_list_reverse`: O(n²) + depth-O(n) → O(n) + O(1) stack. Evaluator sonnet
  **91/100 PASS**, zero blocking. Gate 3b green twice (PR head 21 checks, merge 20, zero not-green).
- **The regression arm is a DEPTH discriminator**, not a timing one — a 20,000-element reverse
  errors `RT_REC_003` on the old form and returns `7` delegated. Cannot flake on a loaded runner.
- **Principle-3 half**: **31** `_list_*` builtins registered, only **12** called from `std/` —
  **19 had zero callers**. The other 18 now carry an explicit reason enforced by
  `TestEveryListBuiltinIsDelegatedOrExplained`.
- **The judge falsified two of that gate's own claims by ADDING code, not removing it**: a builtin
  named by a Go constant was invisible to the AST scan, and a comment laundered a real revert.
  Fixed before merge by reading the live registries (neither complete alone: 18 and 26, union 31).

## Next picks

1. **`m-stdlib-list-delegation-sweep`** — the 18 remaining zero-caller builtins, 13 of them genuine
   delegation candidates (`take` is quadratic). Already enumerated by the gate; each needs its own
   semantic-equivalence check. Ungated by `D-22`.
2. **LC-2 `m-list-accessor-seam`** — still **blocked on `D-22`**.
3. **`m-ailang-test-builtin-resolution`** — `ailang test` cannot resolve `$builtin.*` for a
   `let`-bound delegated call; gets worse as `std/list` delegates more. 4. `m-sweep-orphans-2026-08-17`.

## Parked on Mark

- **`D-22` (OPEN, asked at 239, re-asked 240, 241, 242)** — which representation do LC-2…LC-5 build
  for: **`C1`** (plain cons cells, what the 15.5–21.5 person-day decomposition was scoped around)
  or **`C2K32`** (chunked, which the doc's tie-break selects on memory)? All three candidates passed
  all five clauses, so nothing separates them on correctness. **One word.**
- Ledger is 22 rows; this is the only OPEN one.

## Loop

Cadence nightly · controller opus · executor `codex:gpt-5.6-sol` · evaluator sonnet · designer
rotation unchanged (next = `codex`; no designer ran — no new doc needed) · metered **$0.00** of $5
· no GPU, no `rig.lock`.
