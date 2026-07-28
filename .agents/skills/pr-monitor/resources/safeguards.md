# Safeguards: when to address, when to defer, when to escalate

CodeRabbit is a paid third-party service. Every iteration cycle we burn on its findings costs **our** tokens for value that partly accrues to **their** platform. After the first 1–2 substantive cycles, additional CR findings tend toward diminishing returns — polish, micro-style, alternate-but-equivalent patterns. We need explicit policy on when to keep iterating and when to stop.

## The problem

A single CR review wave on a non-trivial PR can produce 3–8 findings. Each finding cycle costs ~5–15K of our tokens (read finding, plan fix, apply, test, commit, reply). CR uses our pushed changes to refine its model, then posts a new wave with the next layer of suggestions.

Without a cap, three things go wrong:

1. **Token waste on polish** — Trivial/Nitpick findings (e.g., "use `dict[str, object]` instead of `dict`") consume real budget for marginal real-world value.
2. **Maintainer fatigue** — A PR with 6+ commits of CR-iteration smells process-heavy. Maintainers prefer 2–3 substantive iterations and a merge.
3. **Free training data** — We're providing CR with curated diffs that demonstrate "this is the kind of fix that closes the loop." That's their product moat, not ours.

## Severity ladder

CR explicitly tags findings with one of four severities. Treat them as our priority queue:

| Marker | What it means | Default response |
|---|---|---|
| 🔴 Critical | Real bug, security issue, or correctness break | **Always address** — even on wave 3+ |
| 🟠 Major | Significant improvement, API surface issue, or behavioral problem | **Address in wave 1–2**; defer to user on wave 3+ |
| 🟡 Minor | Polish, style, or non-functional refinement | **Address in wave 1**; batch for user review wave 2+ |
| 🔵 Trivial / 🧹 Nitpick | Wording, equivalent alternatives, no behavior change | **Defer to user** unless trivially cheap |

If a finding doesn't carry a severity marker (rare), treat it as 🟡 Minor.

## Iteration ladder

The principle: **first cycles are high-value, later cycles are diminishing returns.** Apply progressively stricter filters as wave count grows.

| Wave | Recommended policy |
|---|---|
| 1 | Address everything Major and above; address Minor unless costly; defer Trivial unless trivially cheap. |
| 2 | Address Critical + Major; surface Minor + Trivial to user with a one-line summary; let user choose. |
| 3+ | Address Critical only. Post a top-level comment explaining: "Wave 3 findings are diminishing-returns territory; deferring to maintainer's final review pass to make the call." |

After wave 3, the right move is usually to **wait for the maintainer** rather than continue. If the maintainer already approved earlier asks, additional CR findings rarely block merge.

## Decision tree

```
        new wave of findings landed
                 │
                 ▼
   ┌─ Is there a Critical (🔴)? ─── Yes ──► always address
   │
   ├─ Wave count? ──────────────── wave 1 ─► severity floor = Trivial (address all)
   │                                          (just be efficient)
   │                                wave 2 ─► severity floor = Minor
   │                                          (skip 🔵/🧹 with explanation to user)
   │                                wave 3+ ─► severity floor = Major
   │                                          (escalate to user; defer)
   │
   └─ Has maintainer engaged in last 24h? ─── No ─► escalate to user
                                                    ("CR posted wave N, no maintainer activity")
```

## Escalation patterns

When deferring, **always** post a top-level PR comment summarising what we're deferring and why. This avoids the "did we miss it?" trap.

### Template — deferring with reason

```markdown
@reviewer — quick note on the latest CR wave (3 findings):

- [x] Critical / Major: [F1] addressed in [SHA]
- [ ] Minor / Trivial: [F2] [F3] — deferring to your discretion. Both are style refinements
      that don't change behavior; happy to apply either if you'd prefer them in.

Standing by once you've had a chance to look.
```

### Template — escalating to user (in chat)

```
PR #N has 3 CR review waves now. Latest wave is 4 findings:
  🔴 Critical: 0
  🟠 Major:    1 ← worth addressing
  🟡 Minor:    2 ← borderline
  🔵 Trivial:  1 ← skip

Suggest: address the 1 Major, defer the rest with a comment.
Want me to proceed, or address everything anyway?
```

## Maintainer asks are different

The above is about **CR findings**. Maintainer (human reviewer) asks are different and always take priority:

- A maintainer ask with `CHANGES_REQUESTED` blocks merge. Address it.
- A maintainer comment that's not a formal review request is still high-priority — they have merge power.
- CR's findings are advisory; maintainer's are gating.

If maintainer + CR disagree, side with the maintainer and explicitly note the deviation in the reply.

## When CR is wrong

CR sometimes suggests diffs that contradict the project's actual behavior or conventions. Examples from our `aallan/vera-bench` work:

- Used capital `"True\nFalse"` (Python repr) where AILANG outputs lowercase
- Suggested adding `-> None` to one Click handler when project policy is to leave all four unannotated
- Suggested removing tests that "go beyond what was asked" when the extra coverage was load-bearing

When you spot this:
- Apply the **intent** if it's correct
- Deviate from the literal diff with explanation
- Reply explains the deviation cleanly

CR often **agrees** with the deviation and updates its learned heuristics, which is the only feedback loop where iteration with CR earns lasting value.

## Budget script

`scripts/pr_budget.sh <owner>/<repo> <pr-number>` reports:

- Total iteration cost (commits, review waves, our replies)
- Severity histogram of CR findings
- Whether we're past wave 2 (heuristic warning)
- Whether all threads are resolved (then "wait for maintainer" is the right next action)

Run it BEFORE responding to a new wave. If the verdict is "diminishing returns", check in with the user first.

## Hard limits to consider setting

These aren't enforced by tooling yet — they're policy norms:

- **Max 5 commits of pure review-iteration per PR.** Beyond that, the PR scope has expanded under cover of review fixes, or we're polishing past the point of value.
- **Max 24h of unresponded CR findings** before escalating to user with a status summary.
- **Stop applying CR Trivial/Nitpick findings entirely** on PRs with 2+ waves already.

The user can override any of these for a specific PR, but the default should be conservative.
