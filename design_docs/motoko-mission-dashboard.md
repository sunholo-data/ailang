# Motoko Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, OVERWRITTEN by mission-control Gate 4 each iteration (history lives in
> the charter STATUS + [log](motoko-mission-log.md)). Fresh session = THIS + MEMORY.md. Humans
> steer via the bookkeeping issue. **Namespaced** — `mission-dashboard.md` is V1's.

**Updated**: 2026-08-14 ~10:30 local (iteration 4)

## Where the mission is
- **Charter RATIFIED** (iter 0). Bar = 6 clauses; clause 6 (motoko graduates into the mission
  executor fleet) is the meta-goal. Epic = [DST migration](planned/m-motoko-dst-refactor-migration.md).
- **Iter 1–3**: 51 fork commits dispositioned ([ledger](planned/m-motoko-fork-disposition.md));
  Phase 0 became a **bounded fail-closed gate** with **G5** (Arni's word) as a permanent human
  predicate; the output-headroom case was corrected before filing — it was wrong both ways.
- **Iter 4**: **R8 SETTLED → PORT.** Ledger now **14 SUPERSEDED / 17 PORT / 14 DROP / 6 UNRESOLVED**
  — first of the seven post-review UNRESOLVED rows to close. Measured, not re-read
  (`tools/motoko/r8_headroom_band.ail`): the ladder sends at **79%** and the seal returns **`Ok`**,
  leaving **54,905** headroom against a **65,536** output cap. Controls both fire.
  **The mechanism changes the ask**: the ladder has *no lever* — `elide_walk` only touches
  `role=="tool"`, so a large **user** message is invisible to all four tiers (~1% removed). So an
  extension-side reserve cannot fix it; the ask is **one argument at `session.ail:2561`**, with
  upstream's own `:2534` as precedent.
- **Iter 4 also landed the recovery job that made iter 4 exist** (`ceb2bb055`) — live but uncommitted.

## Next
- **Item 5a (NEW, next pick)** — the driver's model probe hangs with **empty output**: **6 refusals
  / 4 starts = 60% of fires**, each costing a half-day at the 12h cadence. Diagnose only.
- Then item 6 (`fmt` re-measurement instrument). **Item 5 needs no more work from us** — waiting
  only on the 2026-08-27 bound.

## Blocked / parked
- **Phase-0 gate REAL and unmet**: G1 `#154` **OPEN**, G2 `packages/` absent from motoko_agent's
  `origin/main`, G3 registry has only `1.0.0–2.2.0`. Items 10/11/12 parked; **G5 is permanent human
  residual by design**. Items 9, 13, 14 need a green tree. An idle iteration here is correct.
- **Upstream PR #97: still no reply** since 2026-08-11 — **expires 2026-08-27**, then we file
  standalone and carry locally. Nothing technical blocks filing now that R8 is settled.

## Open with Mark (see bookkeeping issue #663) — nothing blocked on you
1. **Carried from iter 1, unanswered**: (a) routing item 3's *analysis* while its *design* was
   quorum-blocked — too loose? (b) keep the namespaced `motoko-mission-dashboard.md`?
2. **New, FYI not a blocker**: the empty-output probe stall is **fleet-wide** (v1 hit the identical
   signature in the same window). Recovery now retries it, which also *hides* it — hence item 5a.

## Loop posture
- Cadence **12h** (`43200s`); effective cadence is worse — see item 5a.
- Routing: controller opus · designer `claude:claude-fable-5` (rotation **unadvanced** — no doc
  created) · planner/executor `codex:gpt-5.6-sol` · evaluator sonnet. **No lane degradation.**
- **Metered iter 4: $0.00** of $5 — controller-only measurement, no sub-agent spawned.
- `make quick-install` **skipped deliberately**: an eval-suite was live on the shared `ailang`.
