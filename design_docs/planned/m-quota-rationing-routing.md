# M-QUOTA-RATIONING-ROUTING: route by budget position, not just availability

**Status**: **PROPOSAL — D-1..D-5 RATIFIED by Mark (attended, 2026-09-06); not yet
implemented.** The design questions are settled; the build is not started and needs review
of the revised doc first.
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

## Non-goals

- Changing which model is *best* for a role. That is ELO-routing's question; this doc only
  answers whether the chosen lane can be afforded right now.
- Reducing iteration cost. That is the context/rotation work already landed.
- A single cross-provider currency. See the crux above — it would be invented, not measured.
