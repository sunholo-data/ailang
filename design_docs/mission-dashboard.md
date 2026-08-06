# Mission Dashboard — the 30-second control context

> **Contract**: ≤40 lines, refreshed by mission-control Gate 4 every iteration (overwrite, never
> append — history lives in the charter/log). A fresh session reads THIS + MEMORY.md and has full
> steering context. Humans steer via comments on the bookkeeping issue, never a long-lived thread.

**Updated**: 2026-08-06 ~16:45 local (iteration 152)

## Now
- **Latest release**: v0.33.0 (2026-08-04)
- **`#590` FIXED — `assert` had no working call site in the language.** `test "n" { assert c }`
  failed 100% of the time; `assert` is a reserved keyword parseable in exactly one construct, and
  that construct was broken. Lowered in the *fold* (not the printer) so it short-circuits. PR **#605**.
- **Sweep row closed**: `#590` fixed · `#589` NOT reproduced at HEAD, stays open (needs the
  reporter's 15-module closure; this repo's widest multi-test file has **1** import).
- **Next picks**: `#602`/`#603` findings batch · `#545` (orphan-PR rebase) · `m-property-generator-coverage` Lane B1
- **Loops**: v1 90min · world 4h · both armed
- **Routing**: controller opus-5 · executor codex `gpt-5.6-sol` · evaluator sonnet · planner via derive-lane
- **Designer rotation**: last-used `codex:gpt-5.6-sol` → next `claude:claude-fable-5` (not fired since iter-150)

## Parked on Mark
- **⚠ CODEX QUOTA EXHAUSTED until 2026-08-08 11:24** — the pinned executor/planner lane is DOWN.
  The 1-token probe still passes, so this is invisible until a real run. Opus fallback works but
  costs generator≠judge cross-provider independence (executor and evaluator are both Anthropic now).
- **RELEASE ASK (one word, from iter-151)**: tag a release now `#498` Lane B is complete? Carries
  `#510`/`#477`/Lane-A/Lane-B and unblocks World, which consumes pinned releases only.
- **D-6 (one word, unanswered since iter-150)**: `m-net-effect-proxy-boundary` completeness gate —
  **(A)** grep gate now + AST analyzer as follow-up (3d) or **(B)** AST analyzer in-sprint (4d).
  All five constructions (B) would catch are **zero at HEAD**, so (A) suffices for present correctness.
- Low-stakes tail: D-1 SSRF ratification · pure-prng split scope · persisted cost_status · ?-op briefing

## Recently settled (don't re-ask)
- Standing fast-forward authorization **YES** (0 ahead + clean → `merge --ff-only`, no ask)
- The 08-03 dev reconcile has held **12 iterations**; skill byte-identical to origin

## Open findings worth knowing
- **`#604`** (filed this iteration): named test bodies check only the **LAST** expression — earlier
  failing checks are discarded, so `{ false; true }` reports `All tests passed!`. Vacuous-pass class.
- **`#602`**: `TestSolve_HardTimeout_FakeSolverIgnoringT` reds `go test ./...` on clean dev under
  load — but it does **not** red on CI runners, so CI is not the instrument for it.
- `go build ./...` is **already rc=1 at base** (`cmd/wasm`, `gen/main`) — never a usable sprint gate.
- **Run the CI job's OWN gate list**, not a hand-picked subset: `make check-changelog` red-lighted
  a required context this iteration after seven local gates all passed.
