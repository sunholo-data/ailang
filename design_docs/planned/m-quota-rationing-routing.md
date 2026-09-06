# M-QUOTA-RATIONING-ROUTING: route by budget position, not just availability

**Status**: **RATIFIED — D-1..D-5 decided by Mark (attended, 2026-09-06). Approved to plan
and execute.**
**Target**: v0.36.0 · **Priority**: P1 — codex reached 50% of a bucket in a day
**Estimated**: 5 milestones, ~1,400 LOC, 3 days
**Author**: Claude Opus 5 (attended session, Mark), 2026-09-06
**Related**: [M-MISSION-ELO-ROUTING](m-mission-elo-routing.md) — this is the *scarcity term*
that doc's cost channel lacks. If both proceed, this supplies the budget signal and that
one supplies the quality signal; they meet at lane selection.

## The problem, stated from measurement

Routing today asks **"is this lane up?"** and never **"can it afford to be used?"**. A
1-token probe answers the first. Nothing answers the second, and on 2026-09-06 that cost a
codex bucket in a day: 50% consumed, with the fleet routing four codex roles per iteration
whenever Anthropic was unavailable.

The concrete failure: astra's controller probe returned rc=0 at 09:03; the iteration died at
gate-5 after 5,421s and 399,933 tokens on *"try again at Sep 12th"*. The probe answered the
**5-hour window's** question while the **longer cap** was already spent.

## The buckets are not commensurable, and that is the crux

**CORRECTED 2026-09-06 (Mark, attended): Claude has a weekly cap AND a 5-hour window, same
as codex.** The first draft of this doc said Claude was 5-hourly and codex 5-hourly-plus-
weekly, and used that asymmetry as a reason to prefer Anthropic. The asymmetry does not
exist, and the evidence was in our own mission log the whole time:

> *"the 16-hour silence from 2026-08-16 15:00 to 2026-08-17 07:19 was **45 fires refused
> before starting**, every Anthropic preference `quota-limited`"*

A 5-hour-only window refills three times in sixteen hours. That drought was a longer cap.

| bucket | monthly | how it is capped | refills |
|---|---|---|---|
| Claude | $200 | usage units — **5-hour window AND a weekly cap** | short ~4–5×/day; weekly |
| Codex | $100 | usage units — **5-hour window AND a longer cap** | short ~4–5×/day; measured resets Aug 20, Sep 12 |
| OpenRouter | $200 | **metered dollars**, monthly | not until the month rolls |

**The correction makes the design SIMPLER, not more complex.** Every subscription bucket has
two windows, so there is no special case: the rule is uniform — *ration per (bucket, window),
and the binding constraint is the tighter of them*. The first draft was about to encode codex
as the exception, which would have been wrong the moment Anthropic's weekly cap bound first.

**Two are subscriptions and one is metered.** A "cost" in Claude or codex is a fraction of a
*window*; a cost in OpenRouter is money. They cannot be added, and any design that puts them
in one currency is inventing a conversion. The honest common unit is
**fraction-of-this-bucket's-period-capacity consumed**.

**What DOES differ, and is the real basis for ordering:** Claude is $200/month against
codex's $100 — roughly twice the capacity for the same class of work. That, not a refill
rate, is why the controller ladder should exhaust Anthropic before crossing to codex. The
ladder reorder landed on 2026-09-06 with the wrong reason attached and has been corrected in
the driver.

That distinction is also why the fleet's existing KPI cannot help: `ailang chains stats`
measures metered dollars, so it reports v1 at **$11.34 all-time** while v1 burned half a
codex bucket in a day. Subscription usage is invisible to it by construction.

## Proposal

### 1. A ration, expressed per bucket and per period

For a bucket with capacity `C` over period `P`, the pro-rata allowance at time `t` is:

```
allowance(t) = C × (elapsed(t) / P)
over_ration   = consumed(t) > allowance(t) × tolerance
```

Mark's target was *"~17% a day (100/7?)"*. Worth settling the arithmetic before it is
encoded: **100/7 = 14.3%/day**, 100/6 = 16.7%. A 7-day ration is the one that makes a weekly
bucket last a week; 6 days builds in a slack day. **Open question D-1 below.**

**Every subscription bucket needs two rations**, because every one has two windows, and the
binding constraint is the tighter: `over_ration = over(5h window) OR over(weekly cap)`. This
applies to Claude exactly as it does to codex. A single-window model would have called codex
healthy all through 09-06 while its long cap stayed spent until the 12th — and would equally
have called Anthropic healthy through the 16-hour drought of 08-16.

### 2. Lane selection consults budget position, not just liveness

The pre-flight loops gain one question before the probe:

```
for each rung in the chain:
    if bucket(rung) is OVER RATION and a lower rung exists:  skip, log why
    if probe(rung) fails:                                    advance (unchanged)
    take it
```

Deliberately **skip, not refuse**: a rung over ration is deprioritised, never fatal, so an
over-ration fleet still runs — on the next rung down. Refusing would convert a budget
condition into an outage, which is the failure this exists to prevent.

### 3. OpenRouter becomes the declared overflow

Per Mark: *"OpenRouter models are fallbacks after Claude or codex are above ration."* That
makes it the **third rung by policy rather than by accident**, and it is the right shape —
it is metered and monthly, so it has no cliff, only a slope. It absorbs a spike that would
otherwise exhaust a window.

But it is real money, so it gets its own ration on the same formula (`$200 / 30 days ≈
$6.67/day`), and it is the one bucket where going over ration should eventually become a
**hard** stop rather than a skip, because nothing catches it downstream.

### 4. Coverage estimate

What $500/month buys, from measured iteration costs:

| | measured |
|---|---|
| controller session, codex sol (iter 338) | 999,376 tok |
| controller session, codex astra (mean of 4) | 394,254 tok |
| world post-change | 256,656 tok |

**These are controller sessions only** — planner and executor are separate `codex exec`
processes whose totals reach no log today. That gap is now closed going forward (Gate 4
records per-role tokens), but **a coverage estimate cannot be honestly completed until a few
iterations have run under it.** Presenting a number now would be arithmetic over one
unattributed sample, and this doc would rather say so than produce a figure that looks
authoritative.

What can be said: at ~1M tokens for a single codex-heavy iteration and 8–16 fires/day
available under the current cadence, **the fleet can outspend a codex bucket in under a
day** — which it did.

## What this needs that does not exist yet

1. **Consumption per bucket, live.** Gate 4 now records per-role tokens, but nothing
   aggregates them into "fraction of bucket consumed this window". Requires a small ledger
   in `~/.ailang/state/` keyed by bucket and window.
2. **Bucket capacity.** We do not know codex's window capacity in tokens — only that it
   ends. Two options: infer from observed exhaustion (cheap, imprecise, self-correcting) or
   read a provider usage endpoint if one exists. **Open question D-2.**
3. **Window start times.** Recoverable from the reset timestamps the providers already print
   in their errors ("resets 05:34", "try again at Sep 12th").

## Decisions (RATIFIED by Mark, attended 2026-09-06)

- **D-1 — daily ration: 14%** (100/7). A weekly bucket lasting exactly a week, no slack day.
- **D-2 — capacity: use the PROVIDER ENDPOINT**, not inference from observed exhaustion.
  Exact beats self-correcting, and it removes the need to guess a capacity we have never
  measured. *Implementation note: the endpoint must be found and verified first — the
  OpenRouter account key is inference-only and `/api/v1/activity` 403s, so a provider
  usage endpoint is an assumption until one is confirmed reachable for each of the three.*
- **D-3 — scope: FLEET-WIDE.** That is what the bucket physically is. Consequence to design
  for: one loop can starve three others, so per-mission fairness has to come from somewhere
  else (cadence, or a per-mission share of the fleet ration) rather than from the ration.
- **D-4 — every rung over ration: PAUSE.** Not "take the least-over rung". A paused fleet is
  a visible, recoverable state; a fleet quietly spending past its own ration is the thing
  this doc exists to stop.
- **D-5 — the controller gets RESERVED HEADROOM.** Mark: *"we need enough left over to always
  be available."* The controller is the role that commits to a whole iteration up front and
  crashes at gate-5 when a bucket empties mid-run, so the ration must hold back a controller
  reserve that other roles cannot draw on. This is the one place where "affordable now" and
  "affordable in four hours" genuinely differ, and it is the strongest argument in the doc
  for rationing at all.

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | A lane choice becomes a function of declared ration + measured consumption, not of whichever probe happened to answer first. |
| A2: Replayability | +1 | The ledger records consumption per (bucket, window), so a routing decision can be re-derived after the fact. |
| A3: Effect Legibility | +1 | Spending a subscription bucket is currently an undeclared side effect of picking a lane; it becomes a measured, logged one. |
| A4: Explicit Authority | +1 | The controller reserve (D-5) is an explicit grant: other roles may not draw on it. |
| A5: Bounded Verification | +1 | The ration check is arithmetic over a local ledger — no network call on the hot path once capacity is cached. |
| A6: Safe Concurrency | +1 | Fleet-wide scope (D-3) means one shared ledger; concurrent missions read/write it under `internal/riglock`, which already has stale/dead-holder semantics. |
| A7: Machines First | +2 | "Can this lane afford to be used?" has no machine-readable answer today. It becomes one. |
| A8: Minimal Syntax | 0 | No language surface. |
| A9: Cost Visibility | +2 | The core of the doc. `chains` currently reports v1 at $11.34 all-time while it burned half a codex bucket in a day, because subscription lanes bill $0 metered. |
| A10: Composability | +1 | Rides the existing `chains` stage store and the existing pre-flight loops rather than adding a parallel path. |
| A11: Structured Failure | +1 | "Over ration" becomes a typed, logged lane state alongside quota-limited and unusable. |
| A12: System Boundary | +1 | Provider usage endpoints become one declared crossing with a cached result, not an ambient assumption. |

**Net: +13** → proceed.

### Hard Violation Check
- [x] A1 — removes nondeterminism (probe-order dependence) rather than adding it.
- [x] A3 — the point is to make an existing hidden effect explicit.
- [x] A4 — the controller reserve narrows what other roles may spend; it grants nothing.
- [x] A7 — the doc exists to make a human-only question machine-answerable.

## Verification Log

| # | Claim | How verified | Result |
|---|---|---|---|
| V1 | The bucket label is parsed from FREE TEXT, so it is not canonical | Read `mission_rollup.go:161-164` — `parseQuotaBucket` scans `agent_id` for a `quota:` marker | Confirmed; comment says "parsed from the free-text agent_id" |
| V2 | That produces four spellings of one bucket | `ailang chains stats --by-source-prefix 'mission:v1/'` | `codex 70`, `codex-chatgpt 6`, `Codex-OAuth 1`, `codex-oauth 4`, plus `unlabeled 43`, `none 2` |
| V3 | The rollup counts stages, it does not sum tokens | Read `MissionRollup.QuotaByBucket map[string]int` | `int` count per bucket; no token field |
| V4 | Stages DO carry token counts | Read `mission_rollup.go:102` | `ChainStage{Cost, TokensIn, TokensOut, AgentID}` |
| V5 | `chains` cannot see subscription burn in DOLLARS | `ailang chains stats --by-mission` | v1 `$11.3380` metered all-time while burning ~50% of a codex bucket in one day; 4,972 quota stages `$0-by-design` |
| V6 | Anthropic's limit is ACCOUNT-WIDE, not per-model | 08-16 drought record; today's probe log | `claude-opus-5`/`opus-4-8`/`fable-5` all quota-limited, 45 each; opus-5 and fable-5-1 failed 24s apart |
| V7 | Both providers have a 5-hour AND a longer window | Provider error text in our logs | `resets 05:34`/`11:24` alongside `try again at Aug 20th`/`Sep 12th`; a 16-hour Anthropic drought a 5h window cannot produce |
| V8 | A provider usage endpoint is NOT yet proven reachable | CLAUDE.md; OpenRouter key is inference-only | `/api/v1/activity` 403s. **D-2 depends on Mark's new key; the other two providers are unverified.** |

## Conflict Surface

**1. `internal/observatory/mission_rollup.go` is shared.** `parseQuotaBucket` and
`QuotaByBucket` are consumed by `chains stats --by-mission`/`--by-source-prefix`. Changing
the map's value type from `int` to a struct breaks any caller reading it as a count.
Mitigation: add a parallel `QuotaTokensByBucket` rather than changing the existing field, so
existing output is untouched.

**2. Canonicalising labels changes historical output.** `codex-chatgpt` and `Codex-OAuth`
folding into `codex` makes past rollups read differently. That is the point — but it means a
number quoted from a previous run will not reproduce. Mitigation: canonicalise at READ time,
leaving stored `agent_id` untouched, so the raw record is still auditable.

**3. The pre-flight loops are the hot path.** A ration check that makes a network call per
rung per fire would add latency to every iteration. Mitigation: capacity is fetched on a
cadence and cached in the ledger; the hot path reads local state only.

**4. `internal/riglock` is reused for the fleet-wide ledger** (D-3). It already has
stale-lock and dead-holder semantics, so this adds a caller rather than a mechanism — but a
held lock now blocks a mission's pre-flight, so the acquire must be bounded and fail OPEN
(proceed unrationed, loudly) rather than wedge a fire.

**5. Gate 4's per-role token rule (landed 2026-09-06) is SUPERSEDED by M1.** Recording
tokens in prose in the routing-evidence row duplicates a structured store that already has
them (V4). M1 removes it.

## Milestones

**M1 — canonical buckets + token sums (~250 LOC). LANDED 2026-09-06.** Canonicalise the
bucket label at read time; add `QuotaTokensByBucket` beside the existing counts. (The
Gate-4 revert moves to M2 — see the M1 finding above.)
*AC: the four codex spellings report as one; tokens sum per bucket; existing count output
unchanged; `unlabeled`/`none` are reported, never silently folded.*

**M1 FINDING, and it redirects M2.** Quota lanes post `TokensIn/TokensOut = 0` **by
design**, not by omission. `iteration_post.go` states the contract: *"Quota lanes MUST post
0/0 so M1's tokens>0 estimation gate excludes them structurally (no schema marker)."*
`tokens > 0` IS the marker for "this stage is metered and can be priced" — so writing real
token counts into those fields would make the cost estimator price a subscription run as if
it were billed, corrupting the metered KPI to fix the quota one.

So the ledger cannot reuse that field. M2 adds a **separate `QuotaTokens`** that the
estimator ignores by construction, keeping the two accounting systems from contaminating
each other. This also settles the earlier question about Gate 4's prose rule: the structured
store is still the right home, but it needs a new field rather than the existing one, and
until that field exists the routing-evidence row is the only record of per-role quota spend
— so **Gate 4's rule stays until M2 lands**, then goes.

**M2 — the fleet-wide ledger (~350 LOC).** `~/.ailang/state/quota-ledger.json` keyed by
(bucket, window), fed from the chains store. Fleet-wide per D-3, guarded by `riglock` with a
bounded acquire that fails open.
*AC: consumption per bucket per window is queryable; concurrent missions do not corrupt it;
a held lock never wedges a fire.*

**M3 — capacity from the provider endpoint (~300 LOC, D-2).** One adapter per provider,
cached. **Gated on V8**: each endpoint must be proven reachable before its adapter is
trusted, and an unreachable provider degrades to "capacity unknown → do not ration this
bucket", loudly — never to a guessed capacity.
*AC: a reachable endpoint yields capacity; an unreachable one is reported and that bucket is
left unrationed rather than rationed on a guess.*

**M4 — the ration gate (~300 LOC).** In the pre-flight loops: skip a rung whose bucket is
over its 14%/day pro-rata allowance when a lower rung exists; PAUSE the fire when every rung
is over (D-4).
*AC: an over-ration rung is skipped and logged with its numbers; a fire pauses rather than
proceeding when all rungs are over; the pause is visible on the message plane.*

**M5 — controller reserve (~200 LOC, D-5).** A fraction of each bucket that only the
controller may draw on.
*AC: a non-controller role is refused a lane whose bucket is inside the reserve; the
controller is not; the reserve is reported in the ledger.*

## Non-goals

- Changing which model is *best* for a role. That is ELO-routing's question; this doc only
  answers whether the chosen lane can be afforded right now.
- Reducing iteration cost. That is the context/rotation work already landed.
- A single cross-provider currency. See the crux above — it would be invented, not measured.
