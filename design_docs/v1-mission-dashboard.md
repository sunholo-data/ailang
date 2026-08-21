# V1 Mission Dashboard

> 30-second control context. Snapshot, NOT a record — history lives in the charter STATUS block,
> `v1-mission-log.md` and `v1-mission-status-archive.md`. Overwritten every iteration.
> **Namespaced path**: `design_docs/v1-mission-dashboard.md`. Never write the bare
> `mission-dashboard.md` — that literal is shared by every mission on the rig.

**Updated**: 2026-08-21 (iteration 240) · **Release**: v0.33.1 · **Bookkeeping**: [#745](https://github.com/sunholo-data/ailang/issues/745)

## Just landed

- **LC-0 `m-list-interim-communication`** — PR [#811](https://github.com/sunholo-data/ailang/pull/811)
  → [`7db3db2d9`](https://github.com/sunholo-data/ailang/commit/7db3db2d9). Gate 3b GREEN on the
  merge (3/3 workflows, 20 checks, zero not-green). The `::`-is-quadratic + 10,000-frame-recursion
  limitation is now documented in `docs/docs/reference/limitations.md` (canonical) with a summary
  row in `docs/LIMITATIONS.md`, workarounds verified against the Go builtins.

## ⚠ Two issues were auto-closed by accident — both reopened this iteration

A mission-record commit's body that merely *argues about* fixing an issue still auto-closes it:
GitHub's parser does not read English, and Gate 4 mandates exactly that kind of discursive prose.

- **[#676](https://github.com/sunholo-data/ailang/issues/676)** — a live user-reported OOM, triaged
  REAL at HEAD, closed by `dedf3b91f` (docs-only record, 7 files, zero code) over the sentence
  *"the arena fixes #676 completely"*. **REOPENED.**
- **[#612](https://github.com/sunholo-data/ailang/issues/612)** — the `go/packages` AST analyzer,
  closed by `7c7e5e58a`, which shipped one 636-line sprint plan. Measured: `go/packages` importers
  **0**, `x/tools` in `go.mod` **0** (controls fire at 2 and 99). **REOPENED.**
- Audit of all `docs(mission)` commits since June: **4** keyword hits, 2 harmless, these 2 wrong.
  Guard now in the skill (Gate 4); a durable CI gate is queued as `m-commit-autoclose-guard`.

## Next picks

1. **`m-stdlib-reverse-delegates-to-builtin`** — ungated, and this iteration's `++` finding *is*
   its defect: `acc ++ [x]` copies the left operand every step, so `reverse` stays quadratic even
   after cons cells. REQUIRED, not optional.
2. **LC-2 `m-list-accessor-seam`** — **blocked on `D-22`** (below).
3. `m-commit-autoclose-guard` (new, from this iteration's retro).

## Parked on Mark

- **`D-22` (OPEN, asked at iteration 239, re-asked 240)** — which representation do LC-2…LC-5 build
  for: **`C1`** (plain cons cells, what the 15.5–21.5 person-day decomposition was scoped around) or
  **`C2K32`** (chunked, which the doc's tie-break selects on memory: 1.070× vs C1's 1.952×)? All
  three candidates passed all five clauses, so nothing separates them on correctness. **One word.**
- Everything else: ledger is 22 rows, this is the only OPEN one.

## Loop

Cadence nightly · controller opus · planner/executor `codex:gpt-5.6-sol` · evaluator sonnet ·
designer rotation next = `codex` · metered **$0.00** of $5 this iteration (all quota buckets).
