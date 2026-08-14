# Motoko Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, OVERWRITTEN by mission-control Gate 4 each iteration (history lives in
> the charter STATUS + [log](motoko-mission-log.md)). Fresh session = THIS + MEMORY.md. Humans
> steer via the bookkeeping issue. **Namespaced** — `mission-dashboard.md` is V1's.

**Updated**: 2026-08-14 ~23:00 local (iteration 5)

## Where the mission is
- **Charter RATIFIED** (iter 0). Bar = 6 clauses; clause 6 (motoko graduates into the mission
  executor fleet) is the meta-goal. Epic = [DST migration](planned/m-motoko-dst-refactor-migration.md).
- **Iter 1–3**: 51 fork commits dispositioned ([ledger](planned/m-motoko-fork-disposition.md));
  Phase 0 became a **bounded fail-closed gate** with **G5** (Arni's word) as a permanent human
  predicate; the output-headroom case was corrected before filing — it was wrong both ways.
- **Iter 4**: **R8 SETTLED → PORT**; ledger **14 SUPERSEDED / 17 PORT / 14 DROP / 6 UNRESOLVED**.
  The ladder has *no lever* (`elide_walk` only touches `role=="tool"`), so the upstream ask is
  **one argument at `session.ail:2561`**, with upstream's own `:2534` as precedent.
- **Iter 5**: **the loop's own 25%-of-fires refusal is FIXED, and it was ours.** `session_start.sh`
  backgrounded `embed-warmup --timeout 3m &` **without redirecting stdout**; a backgrounded child
  inherits the descriptor, so Claude Code's hook-stdout capture couldn't see EOF until the *child*
  exited — 180s against the driver's 120s probe cap. Hence `captured output: ''` on all three
  models: **never a model verdict, one hook stall seen six times.** PR **#721**.

## Next
- **Item 5b (next pick)** — `make test-launchd-drivers` is **green in CI, 1-passed/28-failed on the
  rig**. The one gate covering the driver scripts is blind where those scripts run.
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
2. **Iter 4's "fleet-wide/environmental" FYI is now CORRECTED**: the stall hit v1 and motoko
   because both are `sunholo-data/ailang` checkouts carrying the SessionStart hooks; world
   (**0 refusals / 89 fires**, no `.claude/settings.json`) was never affected. Not environmental.

## Loop posture
- Cadence **12h** (`43200s`); effective cadence should now recover — that was item 5a.
- Routing: controller opus · designer `claude:claude-fable-5` (rotation **unadvanced** — no doc
  created) · planner/executor `codex:gpt-5.6-sol` · evaluator sonnet. **No lane degradation.**
- **Metered iter 5: $0.00** of $5 — controller measurement + a one-line fix; no sub-agent spawned.
- `make quick-install` **skipped deliberately** (shared-write guardrail, V20).
