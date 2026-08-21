# Mission Dashboard — V1

_Snapshot, overwritten each iteration. History lives in `v1-mission.md` STATUS + `v1-mission-log.md`._

**Latest release:** v0.33.1 · **Loop:** iteration 246, 2026-08-21 · **dev CI:** green (16 checks, 0 not-green at `8040dfd41`)

## In flight
- **PR [#818](https://github.com/sunholo-data/ailang/pull/818)** — codegen resolved stdlib calls
  through a flat 161-name index, so a **compiled** program could run a *different module's*
  implementation while the interpreter was correct. `std/option.map` panicked; `std/list.length`
  became `utf8.RuneCountInString`. Now keyed on `(module, name)`, fail-closed, with a gate over all
  45 `std/*.ail`. Three commits, evaluator FAIL→re-review.

## Next picks
1. `m-codegen-claim-must-match-source` — the new gate checks claim IDENTITY, not SEMANTICS; a
   fabricated `std/list.zipWith` claim passes it while compiled output silently diverges. Needs a
   source-level delegation invariant. **Pairs with #2 — same instrument.**
2. `m-list-builtins-codegen-only` (dispatch half landed) — 13 `_list_*` builtins are codegen-only,
   so `std/list`'s recursive AILANG form and the Go helper are two implementations with no
   differential.
3. `m-show-diverges-between-run-and-compile` — `show` prints `[1, 2, 3]` interpreted, `[1 2 3]`
   compiled, and cannot render tuples at all interpreted. Blocks the naive byte-diff harness #2 wants.
4. `m-stdlib-take-recursion` · `m-math-abs-stdlib-name-mismatch` — both small, both first-party.

## Loop config
Controller opus (session) · designer rotation `codex:gpt-5.6-sol` (unspent 4 iterations) ·
planner/executor `codex:gpt-5.6-sol` · evaluator `sonnet`, own worktree · generator≠judge held.
Fires ~90 min; pinned worktree `~/.ailang-driver-pin/v1`; running skill = main checkout via symlink.

## Parked on Mark
- **`D-22`** (open since iteration 239, unchanged, re-asked verbatim each iteration): do LC-2…LC-5 build for
  **`C1`** (plain cons cells — what the 15.5–21.5d decomposition assumed) or **`C2K32`** (chunked,
  which the doc's tie-break selects)? Nothing separates them on correctness. **One word.**
- Nothing else. Ledger: 22 rows, 1 OPEN.

## Quota posture
metered **$0.00** of $5 this iteration — codex, opus and sonnet are all quota buckets. No GPU, no
`rig.lock`, no quorum. Fable unspent.
