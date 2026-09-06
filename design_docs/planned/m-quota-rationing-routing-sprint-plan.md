# Sprint Plan: M-QUOTA-RATIONING-ROUTING

**Design doc**: [m-quota-rationing-routing.md](./m-quota-rationing-routing.md)
**Created**: 2026-09-06
**Duration**: 5 milestones, ~1,400 LOC, 3 days
**Decisions**: D-1 (10%/day) · D-2 (provider endpoint) · D-3 (fleet-wide) · D-4 (pause) ·
D-5 (controller reserve) — all ratified by Mark, attended 2026-09-06.

## Ordering, and why

M1 and M2 are **measurement**; M4 and M5 are **enforcement**. Measurement lands first and
alone, because a ration gate built on labels that spell codex four different ways would
ration the wrong bucket — and would do it silently, since nothing downstream checks.

M3 is **gated on evidence that does not exist yet** (V8): no provider usage endpoint is
proven reachable. Mark is adding an OpenRouter key; Anthropic and codex are unverified. M3
starts with a reachability probe, and if a provider fails it that bucket stays UNRATIONED
rather than rationed on a guessed capacity.

## Milestones

### M1 — canonical buckets + token sums (~250 LOC)
Canonicalise the bucket label at READ time (`codex`, `codex-chatgpt`, `Codex-OAuth`,
`codex-oauth` → `codex`), leaving stored `agent_id` untouched so the raw record stays
auditable. Add `QuotaTokensByBucket` beside the existing `QuotaByBucket` counts rather than
changing that field's type, so existing callers are unaffected. Revert the Gate-4 prose rule
it supersedes.

- [ ] The four codex spellings report as one bucket
- [ ] Tokens sum per bucket; existing count output byte-unchanged
- [ ] `unlabeled` (43) and `none` (2) are REPORTED, never silently folded into a real bucket
- [ ] Stored `agent_id` is not rewritten
- [ ] Gate-4 per-role token rule removed, with the reason recorded

### M2 — fleet-wide ledger — **LANDED 2026-09-06**
`~/.ailang/state/quota-ledger.json`, keyed by (bucket, window), fed from the chains post path.
Fleet-wide (D-3).

- [x] Consumption per (bucket, window) is queryable — `ailang mission quota [--bucket B] [--json]`
- [x] Concurrent missions cannot corrupt it
- [x] A held lock NEVER wedges a fire
- [x] Window boundaries derive from provider reset timestamps, not wall-clock guesses —
      `BucketUsage.ProviderReset` snaps the window to the provider's own boundary; with none
      pinned, `BoundarySource()` reports `local` rather than implying authority it does not have

**Two things changed from the plan, both for a reason found while building it.**

*Not `internal/riglock`.* That is the single-GPU mutual-exclusion lock; borrowing it would have
serialised the quota ledger against eval jobs and vice versa, and its `Wait` mode is unbounded —
the opposite of the acceptance criterion. The ledger got its own lock, on riglock's atomic-mkdir
mechanism (macOS has no flock) with stale-lock stealing.

*Not a locked read-modify-write.* The write path is an **append journal**: a fire appends one
line with `O_APPEND` and takes NO LOCK, so "a held lock must never wedge a fire" holds by
construction rather than by a timeout that could still be wrong. The journal is the sole authority
for spend; `quota-ledger.json` is a cache of totals plus the capacities M3 will write, and any
reader folds an unconsolidated journal in memory. Consolidation is the only step that needs
exclusion, and it gives up instantly when contended because skipping it loses nothing.

**A separate `quota_tokens` field, end to end.** `tokens > 0` is the fleet's structural marker for
"metered and priceable" — the estimator has no schema flag — so the 4,979 quota stages that record
zero do so BY DESIGN, and reusing those fields would invoice subscription runs to fix the ration.
`quota_tokens` is therefore a new column (`chain_stages`, migration v20), a new field on
`IterationStage`/`ChainStage`, a new `UpdateStageQuotaTokens` on the Backend interface (all five
implementations, including the deployed Firestore one), and the field `mission_rollup` now sums for
`QuotaTokensByBucket`. `Validate` rejects a stage carrying `quota_tokens` without a `quota_bucket`,
because such a stage would be counted once by each accounting system as if it were two runs.

**Verification.** 21 mutants, 21 killed — including the original bug re-introduced as a mutant
(`canonTok += TokensIn+TokensOut`), the double-count on rebuild, lock-steal-always, retire-segments-
early, and quota tokens leaking into the chain total. Three mutants initially SURVIVED and each
was a real finding: two guards were redundant dead code (removed, not test-papered) and one
assertion could not reach the branch it claimed to test (the test was fixed to reach it).

**Still open before the Gate-4 prose token rule can go.** The ledger is fed by the mission-control
skill posting `quota_tokens` (Gate 3 now instructs it). Until loops have run a full window on the
new binary the ledger reads zero, so the prose row stays as the only record — remove it once
`ailang mission quota` shows non-zero across a window, not before.

### M3 — capacity from the provider endpoint (~300 LOC) — GATED ON V8
- [ ] A reachability probe per provider, run once and cached
- [ ] A reachable endpoint yields capacity
- [ ] An UNREACHABLE one leaves that bucket unrationed and says so — never a guessed capacity
- [ ] OpenRouter blocked until Mark's key lands; Anthropic and codex probed independently

### M4 — the ration gate (~300 LOC)
- [ ] An over-ration rung is SKIPPED when a lower rung exists, with its numbers logged
- [ ] Every rung over ration ⇒ the fire PAUSES (D-4), visibly on the message plane
- [ ] 10%/day pro-rata (D-1), computed per (bucket, window), binding = the tighter
- [ ] A bucket with unknown capacity is never treated as over ration

### M5 — controller reserve (~200 LOC)
- [ ] A non-controller role is refused a lane whose bucket is inside the reserve
- [ ] The controller is not refused
- [ ] The reserve is reported in the ledger so it can be seen, not inferred

## Risks

| Risk | Mitigation |
|---|---|
| Rationing on mis-labelled buckets | M1 lands first and alone; M4 cannot start until buckets sum correctly |
| A guessed capacity rations the wrong amount | M3 leaves an unverifiable bucket UNRATIONED and loud. Never guess a capacity |
| The ledger lock wedges the fleet | Bounded acquire, fail open, loud — the pattern the memory gate already uses |
| PAUSE idles the fleet on a bad threshold | The pause is visible on the message plane and the ration is env-overridable for a fast rollback |
| Fleet-wide scope lets one loop starve three | Named in the design as out of scope for the ration itself; fairness comes from cadence or a per-mission share |
