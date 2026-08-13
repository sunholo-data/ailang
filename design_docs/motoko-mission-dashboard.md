# Motoko Mission Dashboard — the 30-second control context
> **Contract**: ≤40 lines, OVERWRITTEN by mission-control Gate 4 each iteration (history lives in
> the charter STATUS + [log](motoko-mission-log.md)). Fresh session = THIS + MEMORY.md. Humans
> steer via the bookkeeping issue. **Namespaced** — `mission-dashboard.md` is V1's.

**Updated**: 2026-08-13 ~12:45 local (iteration 3)

## Where the mission is
- **Charter RATIFIED** (iter 0). Bar = 6 clauses; clause 6 (motoko graduates into the mission
  executor fleet) is the meta-goal. Epic = [DST migration](planned/m-motoko-dst-refactor-migration.md).
- **Iter 1**: item 3 landed — 51 fork commits dispositioned
  ([ledger](planned/m-motoko-fork-disposition.md)), 14 SUPERSEDED / 16 PORT / 14 DROP / **7 UNRESOLVED**.
- **Iter 2**: Phase 0 became a **bounded fail-closed gate**; **D1 resolved by Mark** (`d199df072`,
  attended) — Arni's ABI acknowledgement is predicate **G5**, so the gate cannot open on registry
  evidence alone. Item 4 **LANDED**.
- **Iter 3**: corrected the **output-headroom case** before filing it upstream — wrong both ways.
  The live ladder targets **70%** (not 95%), already leaving more headroom than our own 75k patch;
  the mitigation we credited has **zero production callers**; the real refusal is the phase core's
  seal, input-only at 95%. Rows **V27–V29**.

## Next
- **Item 5, evidence half DONE; one unit-level assertion left** — is the band between the ladder's
  70% target and the seal's 95% permission reachable? Both functions are `pure`, so a direct call,
  not a rig A/B (`R8`, re-sharpened iter 3). Then item 6 (`fmt` re-measurement instrument).

## Blocked / parked
- **Phase-0 gate REAL and unmet**: G1 `#154` **OPEN** (control: `#152` MERGED), G2 `packages/`
  absent from motoko_agent's `origin/main`, G3 registry has only `1.0.0–2.2.0`. Items 10/11/12
  parked; **G5 (Arni's word) is permanent human residual by design**. Items 9, 13, 14 need a green
  tree. An idle iteration here is correct.
- **Upstream PR #97: no reply since 2026-08-11** (0 `arniwesth` events across all 4 surfaces;
  control: 34 issues/PRs carry his comments). No longer open-ended — **expires 2026-08-27**, then
  we file standalone and carry locally.

## Open with Mark (see bookkeeping issue #663) — nothing new; nothing blocked on you
1. **Carried from iter 1, unanswered**: (a) routing item 3's *analysis* while its *design* was
   quorum-blocked — too loose? (b) keep the namespaced `motoko-mission-dashboard.md`?

## Loop posture
- Cadence **12h** (`43200s`) while quotas are watched. The 00:13 fire refused cleanly ("no usable
  model" — all three Anthropic probes timed out); 12:25 ran normally.
- Routing: controller opus · designer `claude:claude-fable-5` (rotation **unadvanced** — no doc
  created) · planner/executor `codex:gpt-5.6-sol` · evaluator sonnet. **No lane degradation.**
- **Metered iter 3: $0.00** of $5 — controller-only measurement, no sub-agent spawned.
