# Mission Dashboard — V1

*Snapshot, overwritten every iteration. History lives in the charter STATUS block and the log.*

**Last iteration:** 267 · 2026-08-24 · **LANDED**

## Latest release
v0.33.1 (dev at `4a813b2c0`)

## In flight / next
- **`#764` protocol-only serveapi module** — M1 LANDED `d54672b85` (iter 266), **M2 LANDED `4a813b2c0` (iter 267)**.
  Next: **M3** — the refusal gate (`scripts/check_protocol_closure.sh`, 9 refusal branches each with a
  neutering mutation, `make` + CI wiring, docs/CHANGELOG). Then M4. Issue stays OPEN.
- M2 measured: `./serveapi` non-stdlib deps **480 → 31** (target ≤40); module roots **67 → 10**;
  `internal/apiserver` import **1 → 0**; TRANSITIONAL shims **16 → 0**.

## Queue after #764
- m-sweep-orphans-2026-08-17 (3 of 15 remain)
- clause-3 accessibility cluster (bulk of v1.0)
- **NEW** m-sweep-orphans-2026-08-24 (8 orphans from this week's sweep)
- **NEW** m-serveapi-moved-code-coverage (2 unpinned lines the iter-267 judge found; pre-existing)

## Loop cadence + routing
- launchd `dev.ailang.mission-control`, pinned worktree `~/.ailang-driver-pin/v1`
- controller `opus` · designer ROTATION (seed fable) · planner/executor `codex:gpt-5.6-sol` ·
  evaluator `sonnet` (Anthropic)
- generator≠judge enforced: codex executor → sonnet judge, each in its own worktree

## Parked on Mark (OPEN decisions)
- **D-30** harness↔`ai-check` version coupling before the `not_applicable` split (schema / same-binary / accept)
- **D-31** split the designer rotation into authoring vs review lanes (or widen it) — 4 instances
- **D-32** exempt `inconclusive` verification obligations from `cost_per_verified_success`?

## Quota posture
Anthropic available (controller opus, evaluator sonnet both live). Codex lane live.
Metered spend this iteration: **$0.00** of $5 ceiling — every lane used was a quota bucket.
