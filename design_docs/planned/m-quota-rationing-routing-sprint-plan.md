# Sprint Plan: M-QUOTA-RATIONING-ROUTING

**Design doc**: [m-quota-rationing-routing.md](./m-quota-rationing-routing.md)
**Created**: 2026-09-06
**Duration**: 5 milestones, ~1,400 LOC, 3 days
**Decisions**: D-1 (14%/day) · D-2 (provider endpoint) · D-3 (fleet-wide) · D-4 (pause) ·
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

### M2 — fleet-wide ledger (~350 LOC)
`~/.ailang/state/quota-ledger.json`, keyed by (bucket, window), fed from the chains store.
Fleet-wide (D-3), guarded by `internal/riglock`.

- [ ] Consumption per (bucket, window) is queryable
- [ ] Concurrent missions cannot corrupt it
- [ ] A held lock NEVER wedges a fire: bounded acquire, fail OPEN and loud
- [ ] Window boundaries derive from provider reset timestamps, not wall-clock guesses

### M3 — capacity from the provider endpoint (~300 LOC) — GATED ON V8
- [ ] A reachability probe per provider, run once and cached
- [ ] A reachable endpoint yields capacity
- [ ] An UNREACHABLE one leaves that bucket unrationed and says so — never a guessed capacity
- [ ] OpenRouter blocked until Mark's key lands; Anthropic and codex probed independently

### M4 — the ration gate (~300 LOC)
- [ ] An over-ration rung is SKIPPED when a lower rung exists, with its numbers logged
- [ ] Every rung over ration ⇒ the fire PAUSES (D-4), visibly on the message plane
- [ ] 14%/day pro-rata (D-1), computed per (bucket, window), binding = the tighter
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
