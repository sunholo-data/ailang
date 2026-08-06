# Mission Dashboard — the 30-second control context

> **Contract**: ≤40 lines, refreshed by mission-control Gate 4 every iteration (overwrite, never
> append — history lives in the charter/log). A fresh session reads THIS + MEMORY.md and has full
> steering context. Humans steer via comments on the bookkeeping issue, never a long-lived thread.

**Updated**: 2026-08-06 ~21:35 local (iteration 154)

## Now
- **Latest release**: v0.33.0 (2026-08-04)
- **⚠ GitHub Actions MAJOR OUTAGE still live** (incident 15:22Z, `investigating`, last update
  19:43Z — 6h+). **Nothing can land.** `#608` has ZERO workflow runs created (GitHub never made
  them — nothing to re-run, no instrument to poll). On `#606` the re-runs partly drained: **docs
  went failure→success on a byte-identical tree** (the strongest control — only the environment can
  be the variable), Build-and-Release queued, CI+CodeQL `failure` whose every job is `cancelled` at
  **steps=0 with zero failed steps**. **No revert warranted — re-run when the incident closes.**
- **Two PRs open and BLOCKED ON CI ONLY** — both fully reviewed, neither LANDED:
  - **`#606`** (`#603`, CodeQL reflected-xss) — evaluator PASS 91/100
  - **`#608`** (`#602`, smt pidfile race) — evaluator PASS 88/100
- **Next picks**: merge `#606` + `#608` once CI returns · `#545` (orphan-PR rebase) ·
  `m-property-generator-coverage` Lane B1 · `#604` · `m-net-effect-proxy-boundary` (unparked)
- **Loops**: v1 90min · world 4h · both armed
- **Routing**: controller opus-5 · **executor `pi:openrouter/deepseek/deepseek-v4-flash-0731`
  (datapoint 3 of ≥3, $0.0063 this iteration)** · evaluator sonnet · planner via derive-lane
- **Designer rotation**: last-used `codex:gpt-5.6-sol` → next `claude:claude-fable-5` (not fired since iter-150)

## Parked on Mark
- Low-stakes tail: D-1 SSRF ratification · pure-prng split scope · persisted cost_status · ?-op briefing

## Recently settled (don't re-ask)
- **D-6 = (A)**, **RELEASE = WAIT**, **D-7 = KEEP WEEKLY** (all Mark, attended 2026-08-06)
- Standing fast-forward authorization **YES** (0 ahead + clean → `merge --ff-only`, no ask)
- The 08-03 dev reconcile has held **14 iterations**; skill byte-identical to origin
- **codex quota dry until ~2026-08-08** — the pi lane covers executor work

## Open findings worth knowing
- **An outage signature is a FAMILY, not `steps=0`.** One `#606` job ran **17 steps, all green**
  (incl. `Run tests`, `Complete job`) and still concluded `failure`. Iter-153's invariant read
  strictly would have called that a real regression. Ask "is the failure attributable to any
  **step**", not "did steps run".
- **`check-runs` EMPTIES during a re-run** — my first poll read `pending=0 failures=0` over a list
  of **one**. Gate 3b recommends it as drift-proof; that is true against drift and wrong during a
  re-run. Always assert the expected workflow set is PRESENT before any verdict counts.
- **`#602` measured**: budget is **3s** (not the issue's 1s); 1 spawn in 300 exceeds it under load
  (~0.7%/suite run). The worst trial hit at the **lower** load — a tail effect, not a load curve.
  Polling for the pidfile cannot fix it: on a lost race the file is never written at all.
- **`#604`**: named test bodies check only the **LAST** expression. Vacuous-pass class.
- `go build ./...` is **already rc=1 at base** (`cmd/wasm`, `gen/main`) — never a usable sprint gate.
- **A guard whose removal reds nothing is not a guard** (iter-153); **a mutation test needs proof
  the mutation LANDED** before its result means anything.
