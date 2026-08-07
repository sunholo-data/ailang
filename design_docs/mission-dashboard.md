# Mission Dashboard — the 30-second control context

> **Contract**: ≤40 lines, refreshed by mission-control Gate 4 every iteration (overwrite, never
> append — history lives in the charter/log). A fresh session reads THIS + MEMORY.md and has full
> steering context. Humans steer via comments on the bookkeeping issue, never a long-lived thread.

**Updated**: 2026-08-07 ~10:40 local (iteration 158)

## Now
- **Latest release**: **v0.33.0** · HEAD `32583be57`. ⚠ the changelog carries a
  `## [v0.33.1] - 2026-08-06` section but **no `v0.33.1` tag exists** — written, not released.
- `dev` CI **GREEN at HEAD**: CI · Build-and-Release · Docs-Deploy all `success`. Outage stays
  closed (githubstatus *All Systems Operational*); `pull_request` webhooks fire normally again.
- Nightly `[nightly-eval]` open alarms **0** (control: 30 in `--state all`).
- `dev == origin/dev`; running `SKILL.md` byte-identical to origin — 08-03 reconcile held **18×**.
- ⚠ **SonarCloud red on 8 consecutive ANALYSED commits** (enumerated, not extrapolated) — and it is
  **no longer purely inherited**: `#545` moved new-code coverage **78.7% → 73.7%** (its head read
  63.4%). Non-required, so it did not gate. Gate 1 CAN now see it — that blind spot was this
  iteration's skill edit. Tracked in `#615`.

## In flight
- **`#613`** proxy-boundary M1 — DRAFT *DO-NOT-MERGE*, implemented and held. Blocked on **`D-1`**.
- **`#604`** named-test vacuous pass — design doc written, **PARKED** on **`D-2`**. `#614` (the
  nested-block twin) open and unrouted.

## Next
`D-2` → resume `#604` · `D-1` → `#613` · then `m-property-generator-coverage` Lane B1 (sequence
`#535` first — still open) · then the swept-issue batch · `#615` (test gap `#545` left behind).

## Loop + routing
Controller **opus** · designer **rotation** (next `codex:gpt-5.6-sol`) · planner **opus** (env pin)
· executor **`pi:deepseek-v4-flash-0731`** (codex bucket dry) · evaluator **sonnet**.
Metered this iteration **$0.00** — no metered lane fired (opus controller + sonnet judge, both quota).

## PARKED ON MARK — two asks, both one word (unchanged, no reply yet)
- **`D-1`** (open since iter-150): proxy-boundary drops target-IP SSRF pinning on **proxied**
  routes. **(A)** as-written · **(B)** narrow to literal-IPs · **(C)** rethink.
- **`D-2`** (iter-157): `#604`'s fix closes the top-level vacuous pass but leaves the nested one
  (`#614`). Bound measured — **27** expr node types, so an exhaustive switch with a loud
  `default:` makes "a walk silently misses a node" impossible. **(A)** ship top-level-only ·
  **(B)** widen to close nested · **(C)** make multi-expression test-body blocks an error.

Full record: charter `## STATUS … ITERATION 158` + `v1-mission-log.md`.
