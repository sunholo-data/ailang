# Mission Dashboard — V1

_Snapshot; overwritten every iteration. History lives in `v1-mission.md` (STATUS) and `v1-mission-log.md`._

**Last iteration:** 247 · 2026-08-22 · LANDED · evaluator 83/100 PASS (3 rounds: 78 → 80 → 83)

## Latest

- **Release:** v0.33.1 · `dev` @ `058feebc3`, 22 checks, only SonarCloud not-green (non-required)
- **Just landed:** [#822](https://github.com/sunholo-data/ailang/pull/822) — `show` had two
  implementations that disagreed. A compiled binary printed **live heap addresses**, so its stdout
  differed on every run; the interpreter printed **Go type names** (`<*eval.TupleValue>`) for 7 of
  17 value types. Both fixed, and `tests/golden/codegen` now **runs** what it compiles.

## Next picks

1. `m-array-show-diverges-run-vs-compile` — **NEW, iteration-247 evaluator.** `#[1,2,3]` vs
   `[1,2,3]`. Structural: `codegen_ops.go:357` compiles Array and List to the same Go type by an
   explicit M-TYPE1 decision, so `Show` has no runtime tag to key on. Needs a wrapper type.
2. `m-codegen-claim-must-match-source` + `m-list-builtins-codegen-only` — now unblocked; the
   differential instrument they needed exists.
3. `m-sonar-differential-fixture-duplication` — NEW; Sonar reds at 35.7% duplication on new code.

## Loop

- Cadence: launchd, ~90 min · controller `claude:claude-opus-5` · designer rotation at `codex`
- Executor `codex:gpt-5.6-sol` (5 milestones, 3 rounds) · evaluator `sonnet`, own worktree
- **metered $0.00** of $5 — every lane is a quota bucket

## Parked on Mark

- **`D-22`** — `C1` (plain cons cells) or `C2K32` (chunked, K=32) for LC-2…LC-5? One word.
- **`D-23`** — may the controller fast-forward the main checkout when local `dev` is *ahead* but
  every ahead-commit is content-duplicated upstream? `yes`/`no`.

Nothing else is blocked on a human.
