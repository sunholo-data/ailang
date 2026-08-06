# Mission Dashboard — the 30-second control context

> **Contract**: ≤40 lines, refreshed by mission-control Gate 4 every iteration (overwrite, never
> append — history lives in the charter/log). A fresh session reads THIS + MEMORY.md and has full
> steering context. Humans steer via comments on the bookkeeping issue, never a long-lived thread.

**Updated**: 2026-08-06 ~12:45 local (iteration 151)

## Now
- **Latest release**: v0.33.0 (2026-08-04)
- **`#498` LANE B IS COMPLETE** — M3 landed (PR **#601**), the final milestone. `serveapi` now
  serves MCP *and* A2A from one caller-supplied descriptor set, with a single membership gateway
  and `@nomcp` kept as an MCP-only projection. Evaluator sonnet PASS 81/100 r1, zero blocking.
- **→ This unblocks Ailang World**, which consumes upstream via **pinned releases only**. A tag is
  now the only thing between them and `w-mcp-projection`. **See DECISIONS.**
- **Next picks**: `#545` (orphan-PR rebase) · `m-property-generator-coverage` Lane B1 · sweep `#589`/`#590`
- **Loops**: v1 90min · world 4h · both armed
- **Routing**: controller opus-5 · executor codex `gpt-5.6-sol` · evaluator sonnet · planner via derive-lane
- **Designer rotation**: last-used `codex:gpt-5.6-sol` → next `claude:claude-fable-5` (not fired since iter-150)

## Parked on Mark
- **RELEASE ASK (new, one word)**: tag a release now Lane B is complete? It would carry
  `#510`/`#477`/Lane-A/Lane-B and unblock World. Releases are Mark's sole decision.
- **D-6 (one word, still unanswered from iter-150)**: `m-net-effect-proxy-boundary` M4 completeness
  gate — **(A)** grep gate now + AST analyzer as follow-up, sprint stays 3d; **(B)** `go/packages`
  AST analyzer in-sprint, 3d→4d. Measured: all five constructions the analyzer would catch are
  **zero at HEAD**, so (A) suffices for *present* correctness; (B) buys durability.
- **Ratify alongside D-6**: D-1 trades target-IP SSRF pinning on **proxied** requests only
  (preserved on direct/`NO_PROXY`; the doc is explicit, never claims equivalence).
- Low-stakes tail: pure-prng split scope · persisted cost_status · pipe-operator · ?-op briefing

## Recently settled (don't re-ask)
- Standing fast-forward authorization **YES** (0 ahead + clean → `merge --ff-only`, no ask)
- recorded-stream S2 does **not** jump the queue; the 08-03 dev reconcile has held **11 iterations**

## Open findings worth knowing
- **`#602`** (filed this iteration): `TestSolve_HardTimeout_FakeSolverIgnoringT` reds `go test ./...`
  on **clean dev** — a load-sensitive 3s pidfile race that survived the CI-flake sprint.
- `go build ./...` is **already rc=1 at base** (`cmd/wasm`, `gen/main`) — never a usable sprint gate.
- Four straight milestones shipped a **non-discriminating test** caught only by mutation testing.
