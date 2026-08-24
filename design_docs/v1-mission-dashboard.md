# Mission Dashboard — V1

*Snapshot, overwritten every iteration. History lives in the charter STATUS block and the log.*

**Last iteration:** 269 · 2026-08-24 · **LANDED — `#764` sprint COMPLETE (M1–M4)**

## Latest release
v0.33.1 (dev at `7e7bdffcb`) — **v0.34.0 is the outstanding ask, see Parked on Mark**

## In flight / next
- **`#764` protocol-only serveapi module — ALL FOUR MILESTONES LANDED.**
  M1 `d54672b85` · M2 `4a813b2c0` · M3 `ba2eeb4b4` · **M4 `7e7bdffcb`**.
- At the merge: `serveapi/protocol` = 188 packages, **exactly 1 non-stdlib (itself)**; `serveapi`
  = 224 / 31 non-stdlib / 9 external roots + ailang. Was 479 non-stdlib pre-extraction.
- **#764 deliberately kept OPEN**: World pins upstream by RELEASE, so the merge does not deliver
  it. The delivery is the **v0.34.0 tag** — `D-34` pre-authorises the ask, not the tag.
- M4 corrected a plan defect: `make lint` has TWO path lists; widening only the scan list left the
  gate unable to refuse anything in `serveapi/`. Both changed; 3-arm drill (both rc=2, either rc=0).

## Queue after #764
- **NEW** m-lint-unused-filter-vacuity — `grep -v "is unused"` strips findings before the verdict
  predicate, for `internal`/`cmd`/`testutil` too (judge-found; pre-existing since 2026-04-22)
- m-protocol-closure-arm2-floor · m-protocol-closure-goos-scope (both judge-found iter-268)
- m-sweep-orphans-2026-08-17 (3 of 15) · m-sweep-orphans-2026-08-24 (8 orphans)
- m-serveapi-moved-code-coverage (2 unpinned lines, pre-existing)
- clause-3 accessibility cluster (bulk of v1.0)

## Loop cadence + routing
- launchd `dev.ailang.mission-control`, pinned worktree `~/.ailang-driver-pin/v1`
- controller `opus` · designer ROTATION (seed fable) · planner/executor `codex:gpt-5.6-sol` ·
  evaluator `sonnet`; generator≠judge enforced, each role in its own worktree

## Parked on Mark (OPEN decisions)
- **D-34 IS NOW LIVE — cut v0.34.0?** `#764` is complete on dev; the tag is what reaches World.
- **D-30** harness↔`ai-check` version coupling before the `not_applicable` split
- **D-31** split the designer rotation into authoring vs review lanes (or widen it) — 4 instances
- **D-32** exempt `inconclusive` obligations from `cost_per_verified_success`?

## Quota posture
Anthropic available (controller opus, evaluator sonnet live). Codex lane live.
Metered spend this iteration: **$0.00** of $5 — every lane used was a quota bucket.
