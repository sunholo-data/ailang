# M-QUOTA-RATIONING-ROUTING: route by budget position, not just availability

**Status**: **PROPOSAL — not actioned, for Mark's review.** Nothing here is implemented.
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

## The three buckets are not commensurable, and that is the crux

| bucket | monthly | how it is capped | refills |
|---|---|---|---|
| Claude | $200 | usage-limit units, **~5-hour rolling window** | ~4–5×/day |
| Codex | $100 | usage-limit units, **5-hour window AND a longer cap** | short: ~4–5×/day; long: measured resets on Aug 20, Sep 12 |
| OpenRouter | $200 | **metered dollars**, monthly | not at all until the month rolls |

**Two are subscriptions and one is metered.** A "cost" in Claude or codex is a fraction of a
*window*; a cost in OpenRouter is money. They cannot be added, and any design that puts them
in one currency is inventing a conversion. The honest common unit is
**fraction-of-this-bucket's-period-capacity consumed**.

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

Codex needs **two** rations because it has two windows, and the binding one is the tighter:
`over_ration = over(5h window) OR over(long cap)`. A single-window model would have called
codex healthy all through 09-06, because its 5-hour window kept refilling while the long cap
stayed spent until the 12th.

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

## Open questions for Mark (this doc cannot choose these)

- **D-1 — the daily ration.** 100/7 = 14.3%, or 100/6 = 16.7% with a slack day? A weekly
  bucket lasting exactly a week has no margin for a bad day.
- **D-2 — capacity estimation.** Infer capacity from observed exhaustion, or find a provider
  endpoint? Inference is available today and self-corrects; an endpoint is exact if it exists.
- **D-3 — ration scope.** Per mission, or fleet-wide? Fleet-wide is what the bucket actually
  is; per-mission is what stops one loop starving three others.
- **D-4 — behaviour when EVERY rung is over ration.** Take the least-over rung and log it, or
  stop the fire? Stopping is honest but idles the fleet; proceeding spends a bucket you said
  you would not.
- **D-5 — does the ration gate the CONTROLLER too?** It is the role that commits to a whole
  iteration up front, so it is the one where "affordable now" and "affordable in four hours"
  differ most — and the one where a mid-run exhaustion crashes at gate-5, as it did.

## Non-goals

- Changing which model is *best* for a role. That is ELO-routing's question; this doc only
  answers whether the chosen lane can be afforded right now.
- Reducing iteration cost. That is the context/rotation work already landed.
- A single cross-provider currency. See the crux above — it would be invented, not measured.
