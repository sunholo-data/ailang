### Fixed — a PR closed without merging now clears its approval card

The coordinator's landed-card sweep already resolves a card when its PR merges. A PR closed *without* merging left its card pending forever: on 2026-09-30, 15 of the 35 pending prod cards were for closed PRs, the oldest 21 days old.

- The sweep now asks GitHub for PRs in every state (`state=all`) and applies them in this order: a **merged** PR approves the card, as before. Any **open** PR from the branch leaves the card pending, which covers a PR closed and then reopened or replaced. Otherwise, a **closed** PR rejects the card (`pr-closed #N`), with no re-attempt by the agent and no handoff.
- When Cloud Run shuts an instance down mid-sweep, the sweep now stops with one `landed-card sweep interrupted` line. Previously it logged a `cannot read PRs … context canceled` warning for every remaining card, 234 in a week, which made a working sweep look broken.
