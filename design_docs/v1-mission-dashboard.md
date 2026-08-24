# Mission Dashboard — V1

*Snapshot, overwritten every iteration. History lives in the charter STATUS block and the log.*

**Last iteration:** 268 · 2026-08-24 · **LANDED**

## Latest release
v0.33.1 (dev at `ba2eeb4b4`)

## In flight / next
- **`#764` protocol-only serveapi module** — M1 `d54672b85` · M2 `4a813b2c0` · **M3 LANDED `ba2eeb4b4`**.
  Next: **M4** — docs, lint scope, CHANGELOG, `#764` reply. Issue stays OPEN.
- M3 shipped `scripts/check_protocol_closure.sh`: 2 arms, 9 labelled refusal branches, 5-arm
  self-test. All 9 branches + the doc's Mutations 2/3 produced observed reds. Wired into `make`
  and `ci.yml`. **When M4 lands, `D-34` says cut v0.34.0.**

## Queue after #764
- **NEW** m-protocol-closure-arm2-floor — arm 2 lacks arm 1's stdlib-presence floor leg (judge-found)
- **NEW** m-protocol-closure-goos-scope — gate is single-OS in CI; platform-tagged files unseen
- m-sweep-orphans-2026-08-17 (3 of 15 remain)
- m-sweep-orphans-2026-08-24 (8 orphans)
- m-serveapi-moved-code-coverage (2 unpinned lines, pre-existing)
- clause-3 accessibility cluster (bulk of v1.0)

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
