# Mission Dashboard — the 30-second control context

> **Contract**: ≤40 lines, refreshed by mission-control Gate 4 every iteration (overwrite, never
> append — history lives in the charter/log). A fresh session reads THIS + MEMORY.md and has full
> steering context. Humans steer via comments on the bookkeeping issue, never a long-lived thread.

**Updated**: 2026-08-06 ~19:10 local (iteration 153)

## Now
- **Latest release**: v0.33.0 (2026-08-04)
- **⚠ dev CI is RED and it is NOT our code — GitHub Actions MAJOR OUTAGE** (incident 15:22Z,
  unresolved). The red jobs never acquired a runner (`steps=0` / died in `Set up job`, zero repo
  commands run). Controls: same jobs **green on the parent commit** an hour earlier, and across
  re-runs `docs` went cancelled→success on an **identical tree**. Re-runs fired; Build-and-Release
  already recovered green. **No revert warranted — re-run when the incident closes.**
- **`#603` FIXED (PR #606)** — CodeQL `go/reflected-xss` on the embedded-MCP replay. Taint trace is
  CORRECT (2 of 12 hostile shapes reflect a literal `<script>`), but NOT exploitable: Content-Type
  present on all 12, both reflecting paths carry `nosniff`, json escapes. Fixed rather than
  dismissed because all three guards were **inherited from the SDK**, asserted nowhere locally.
- **Next picks**: `#602` (the findings batch's other half) · `#545` (orphan-PR rebase) ·
  `m-property-generator-coverage` Lane B1 · `#604`
- **Loops**: v1 90min · world 4h · both armed
- **Routing**: controller opus-5 · **executor `pi:openrouter/deepseek/deepseek-v4-flash-0731`
  (NEW, Mark 2026-08-06)** · evaluator sonnet · planner via derive-lane
- **Designer rotation**: last-used `codex:gpt-5.6-sol` → next `claude:claude-fable-5` (not fired since iter-150)

## Parked on Mark
- **D-6 (one word, unanswered since iter-150)**: `m-net-effect-proxy-boundary` completeness gate —
  **(A)** grep gate now + AST analyzer as follow-up (3d) or **(B)** AST analyzer in-sprint (4d).
  All five constructions (B) would catch are **zero at HEAD**, so (A) suffices for present correctness.
  (Explained to Mark 2026-08-06 attended, recommendation A given; answer still owed.)
- Low-stakes tail: D-1 SSRF ratification · pure-prng split scope · persisted cost_status · ?-op briefing

## Recently settled (don't re-ask)
- **RELEASE ASK: WAIT** (Mark, attended 2026-08-06): "ailang world doesn't need a release yet
  I'm told so we can wait." Re-raise only when World actually demands a pinned release.
- **D-7: KEEP WEEKLY** (Mark, attended 2026-08-06): CodeQL stays on the weekly `dev` cadence;
  no push-on-dev/nightly change.
- Standing fast-forward authorization **YES** (0 ahead + clean → `merge --ff-only`, no ask)
- The 08-03 dev reconcile has held **13 iterations**; skill byte-identical to origin
- **codex quota dry until ~2026-08-08** — the pi lane now covers executor work ($0.0057 this iteration)

## Open findings worth knowing
- **`#604`**: named test bodies check only the **LAST** expression — `{ false; true }` reports
  `All tests passed!`. Vacuous-pass class.
- **`#602`**: `TestSolve_HardTimeout_FakeSolverIgnoringT` reds `go test ./...` on clean dev under
  load — but does **not** red on CI runners, so CI is not the instrument for it.
- `go build ./...` is **already rc=1 at base** (`cmd/wasm`, `gen/main`) — never a usable sprint gate.
- **Run the CI job's OWN gate list**, not a hand-picked subset (iter-152's `make check-changelog`).
- **A guard whose removal reds nothing is not a guard** — iter-153's Content-Type default was
  unreachable through the public path until a test injected a transport to reach it.
