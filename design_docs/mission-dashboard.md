# Mission Dashboard — the 30-second control context

> **Contract**: ≤40 lines, refreshed by mission-control Gate 4 every iteration (overwrite, never
> append — history lives in the charter/log). A fresh session reads THIS + MEMORY.md and has full
> steering context. Humans steer via comments on the bookkeeping issue, never a long-lived thread.

**Updated**: 2026-08-07 ~10:40 local (iteration 158)

## Now
- **Latest release**: **v0.33.0** · HEAD `c82aeeb40` (`v0.33.0-63`). ⚠ the changelog carries a
  `## [v0.33.1] - 2026-08-06` section but **no `v0.33.1` tag exists** — written, not released.
- `dev` CI **GREEN at HEAD**: CI · Build-and-Release · Docs-Deploy all `success`. Outage stays
  closed (githubstatus *All Systems Operational*); `pull_request` webhooks fire normally again.
- Nightly `[nightly-eval]` open alarms **0** (control: 30 in `--state all`).
- `dev == origin/dev`; running `SKILL.md` byte-identical to origin — 08-03 reconcile held **18×**.
- ⚠ **SonarCloud has been red for 6 consecutive analysed commits** (78.7% coverage on new code vs
  a required 80%). Non-required, inherited, not a regression — but Gate 1 still cannot see it.

## In flight
- **`#545`** orphan PR (eval cost provenance) — **UNBLOCKED THIS ITERATION**. dev merged in,
  3 conflicts resolved, `make test` rc=0 / 107 pkgs / 0 FAIL. `CONFLICTING → MERGEABLE`.
  Awaiting its own CI; then it is mergeable on sight.
- **`#613`** proxy-boundary M1 — DRAFT *DO-NOT-MERGE*, implemented and held. Blocked on **`D-1`**.
- **`#604`** named-test vacuous pass — design doc written, **PARKED** on **`D-2`**. `#614` (the
  nested-block twin) open and unrouted.

## Next
Land `#545` once green · `D-2` → resume `#604` · `D-1` → `#613` · then `m-property-generator-
coverage` Lane B1 (sequence `#535` first — still open) · then the swept-issue batch.

## Loop + routing
Controller **opus** · designer **rotation** (next `codex:gpt-5.6-sol`) · planner **opus** (env pin)
· executor **`pi:deepseek-v4-flash-0731`** (codex bucket dry) · evaluator **sonnet**.
Metered this iteration **$0.00** — no metered lane fired (controller-only measurement work).

## PARKED ON MARK — two asks, both one word (unchanged, no reply yet)
- **`D-1`** (open since iter-150): proxy-boundary drops target-IP SSRF pinning on **proxied**
  routes. **(A)** as-written · **(B)** narrow to literal-IPs · **(C)** rethink.
- **`D-2`** (iter-157): `#604`'s fix closes the top-level vacuous pass but leaves the nested one
  (`#614`). Bound measured — **27** expr node types, so an exhaustive switch with a loud
  `default:` makes "a walk silently misses a node" impossible. **(A)** ship top-level-only ·
  **(B)** widen to close nested · **(C)** make multi-expression test-body blocks an error.

Full record: charter `## STATUS … ITERATION 158` + `v1-mission-log.md`.
