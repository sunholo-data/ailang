# Mission Dashboard — the 30-second control context

> **Contract**: ≤40 lines, refreshed by mission-control Gate 4 every iteration (overwrite, never
> append — history lives in the charter/log). A fresh session reads THIS + MEMORY.md and has full
> steering context. Humans steer via comments on the bookkeeping issue, never a long-lived thread.

**Updated**: 2026-08-07 ~06:50 local (iteration 157)

## Now
- **Latest release**: v0.33.0 · HEAD `74dd06bb6` (`v0.33.0-60`)
- **THE ACTIONS OUTAGE IS OVER** — githubstatus *All Systems Operational*, zero unresolved
  incidents. Closed on the provider's API, not on one of our own greens. First iteration since 152
  where a green is spendable.
- `dev` CI **GREEN at HEAD**: CI · Build-and-Release · Docs-Deploy all `success`.
- Nightly `9/24` — no regressions, no sustained failures, no flakes. Open `[nightly-eval]` alarms
  **0** (control: 30 in `--state all`).
- `dev == origin/dev`; running `SKILL.md` byte-identical to origin — 08-03 reconcile held **17×**.

## In flight
- **`#613`** proxy-boundary M1 — DRAFT *DO-NOT-MERGE*, implemented and held. Blocked on **`D-1`**.
- **`#604`** named-test vacuous pass — design doc written, **PARKED** on **`D-2`**.
- **`#614`** NEW — the same vacuous pass inside **nested blocks**, measured with discriminating
  controls, filed this iteration. Open, unrouted.
- **`#545`** orphan PR (eval cost provenance) — still needs rebase-or-recut.

## Next
`D-2` → resume `#604` (doc written, premises measured, resume cheap) · `D-1` → `#613` proceeds
as-written or narrowed · then `#545` · then `m-property-generator-coverage` Lane B1.

## Loop + routing
Controller **opus** · designer **rotation** (used `claude:claude-fable-5`; next
`codex:gpt-5.6-sol`) · planner **opus** (env pin) · executor **`pi:deepseek-v4-flash-0731`**
(codex bucket dry until ~Aug 8) · evaluator **sonnet**.
Metered this iteration **$0.213** of the $5 ceiling (quorum reviewers only).

## PARKED ON MARK — two asks, both one word
- **`D-1`** (open since iter-150): proxy-boundary knowingly drops target-IP SSRF pinning on
  **proxied** routes; iter-156 proved it reds 4 existing tests and found a third option (validate
  literal-IP targets only — zero DNS, no TOCTOU). **(A)** as-written · **(B)** narrow to literal-IPs
  · **(C)** rethink.
- **`D-2`** (new): `#604`'s fix closes the top-level vacuous pass but leaves the nested one
  (`#614`), and a reviewer blocks on pinning that silent false-green as expected behaviour. Bound
  measured — **27** expr node types, so an exhaustive switch with a loud `default:` makes "a walk
  silently misses a node" impossible. **(A)** ship top-level-only · **(B)** widen to close nested ·
  **(C)** make multi-expression blocks in test bodies an error.

Full record: charter `## STATUS … ITERATION 157` + `v1-mission-log.md`.
