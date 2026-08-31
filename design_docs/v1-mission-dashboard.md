# Mission Dashboard — V1

*Snapshot, overwritten every iteration. History lives in the charter STATUS block and the mission log.*

**Last iteration**: 310 · 2026-08-31 · [PRODUCT] · design LANDED, sprint plan routed
**Latest release**: v0.34.0 (`git describe` at HEAD: v0.34.0-262-gbd17d9643)
**dev CI**: 14 checks at `bd17d9643`, only `test` in flight; standing non-required SonarCloud red is inherited

## In flight
- `m-registry-interface-hash-blind-to-signatures` — design `66add3b48`, quorum-cleared under the
  narrow-refinement carve-out after **2 BLOCKED rounds, 3/3 reviewers present both times**. A BREAKING
  API change is currently classified `patch` to every downstream consumer. Sprint plan being written.

## Next up (ready, no blockers)
1. `m-coordinator-config-route-preflight` **(a) only** — move validation above the `identical` early
   return in `config diff`. Standalone; the row's `(b) config check` half is unroutable until
   `ExecutionRoute` lands on dev (it exists only on `mission/iter309-route-authority-parity`).
2. `m-openrouter-session-chain-registration` — **un-parked iter-310**; `D-47` was answered
   2026-08-28 and the row was never updated. ~2 LOC, fully specified by the answer.
3. `m-registry-validator-unbounded-compile` — new iter-310. A public HTTP server compiles untrusted
   uploads with `exec.Command` (no timeout, no cancellation) at `validate.go:76/95/116`.
4. `m-weekly-sweep-orphans-2026-08-31` — triage-lite 5 zero-mention open issues.

## Loop health
- **Cadence**: unattended, launchd. Iterations 306 and 307 died without records (recovered by 308);
  308, 309, 310 all recorded.
- **Routing**: designer rotation now `pi:ollama/deepseek-v4-flash:0731-cloud` (3/3 runs verdict `ok`
  this iteration, flat-rate); planner `opus` via `derive-planner-lane.sh`; evaluator `sonnet`.
- **Cost**: metered **$0.1664** this iteration (quorum only), ceiling $5. Designer lane was $0.
- **Known loop defects, filed not fixed**: the skill tells controllers to read a quorum key
  (`absent_reviewers`) that does not exist — **instance 2 recorded this iteration, bar met**; and
  nothing reconciles a queue row's PARK tag against the decision ledger (2 of 2 parked rows were
  stale).

## Parked on Mark
**None.** `scripts/mission_decisions.sh --open` returns zero rows; the ledger holds 50 rows, all
RESOLVED. Note that a zero here does *not* imply the queue is clean — see the stale-park defect above.

## Charter goal
**None defined.** The V1 queue is a prioritized backlog with no countable finish line, so the
Gate-5 "Progress" line cannot be computed. Adding a goal block with a countable unit is a standing
process-fix trigger.
