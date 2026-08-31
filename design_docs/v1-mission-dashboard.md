# Mission Dashboard — V1

*Snapshot, overwritten every iteration. History lives in the charter STATUS block and the mission log.*

**Last iteration**: 311 · 2026-08-31 · [PRODUCT] · M1 LANDED (Sprint 1, 1 of 5 milestones)
**Latest release**: v0.34.0 (origin/dev `3ad55d53b`)
**dev CI**: 16 checks at `3ad55d53b`; the one non-green is SonarCloud — non-required, **inherited** (same `failure` on parent `bd17d9643`).

## In flight
- `m-registry-interface-hash-blind-to-signatures` — **Sprint 1 M1 LANDED**: `internal/iface/
  hash_projection.go`, alias-excluded `HashProjection` + injective `SignatureSet`. Unwired dead
  code by design; nothing changes behaviour until M5. **M2–M5 remain (~3.5d)**: `internal-dump-iface`,
  the subprocess wrapper + `PublishLimits`, `InterfaceHashV2`, the `U`-class classifier.
  Sprint 2 (M6–M9) stays DEFERRED — its blast-radius precondition needs the live registry.

## Next up (ready, no blockers)
1. Same row, **M2** — `iface.BuildCanonicalJSON` + the hidden `internal-dump-iface` subcommand.
2. `m-coordinator-config-route-preflight` **(a) only** — validation above the `identical` early return in `config diff`; `(b)` stays unroutable until `ExecutionRoute` lands on dev.
3. `m-openrouter-session-chain-registration` — un-parked by `D-47`; ~2 LOC, fully specified.
4. `m-registry-validator-unbounded-compile` — untrusted uploads compiled with no timeout.
5. `m-weekly-sweep-orphans-2026-08-31` — triage-lite 5 zero-mention open issues.

## Loop health
- **Cadence**: 308–311 all recorded. Of 296–310, **6** (299, 300, 302, 303, 306, 307) were never written — reaped slots, ~40%.
- **Routing**: executor `codex:gpt-5.6-sol` (probe rc=0); evaluator `sonnet` (Agent tool),
  **PASS 92/100, zero blocking**. No designer/planner needed — both landed at iteration 310.
- **Cost**: metered **$0.00** — no quorum round, no metered lane. Ceiling $5.
- **STATUS rotation audited HEALTHY**: 270 archive stamps; every recorded iteration 296–310 resolves to charter or archive; the 6 gaps are reaped slots, not rotation loss (`git log -S`).
- **Loop defect FIXED**: the skill named the wrong path for the quorum absence key. Corrected live.
- **Still open**: nothing reconciles a queue row's PARK tag against the decision ledger.

## Parked on Mark
**None.** `mission_decisions.sh --check` → 50 rows valid; `--open` → zero. A zero here does *not*
imply the queue is clean — see the stale-park defect above.

## Charter goal (PROVISIONAL — Mark to ratify or replace)
**Unit: open queue rows**, `grep -cE '^\*\*\[<TAG>'` at iteration 311: **68** `[NEXT]` · **1**
`[PARKED]` · **2** `[IN-SPRINT]` · **29** `[LANDED]`. Distance = `[NEXT]`+`[PARKED]` → 0.
Iteration 311 moved it by **0**: milestone progress inside a row does not change the row count,
which is the weakness the goal block itself names.
