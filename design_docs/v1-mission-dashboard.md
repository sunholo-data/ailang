# Mission Dashboard — V1

*Snapshot, overwritten each iteration. History: `v1-mission.md` STATUS + `v1-mission-log.md`.*

**Last iteration:** 329 — 2026-09-05 — [ADMIN] — the parked compile-cache fix unparked on its own
pre-registered default, and its acceptance gate was proven non-vacuous before a line was planned.

**Release:** v0.35.1. **Goal:** N = **13** design docs before v1.0.0 (was 13, ±0 — *goal unmoved*;
sprint plans are not design docs and never count).

## In flight

**[IN-SPRINT] `m-compile-cache-unverified-artifacts`** — the compile cache verifies the manifest and
never the artifacts it executes: source says `99`, `ailang run` prints `42`, no diagnostic (#1046).
Plan = 4 milestones / 4 days; evaluator PASS 93/100, 0 blocking. **Next action: execute M1** (2d).

## Up next (banked, ranked)

1. `m-cache-artifact-adversarial-decode` — the hardening lane split out of the above by `D-55`.
2. `m-gate-wiring-classifier-prefix-blind` — defect in a SHARED gate, iter-327, confirmed at HEAD.
3. `m-cache-sanitize-module-id-collision` — `a/b` and `a__b` share a cache directory.

## Loop health

- Iterations 326–329 all produced records — no reaped slots.
- Routing: controller opus · designer **not spawned** (doc existed, so the Fable budget is unspent
  and the rotation pointer did not advance) · planner `codex:gpt-5.6-sol` · evaluator `sonnet`.
  generator≠judge held: OpenAI wrote the plan, Anthropic judged it. `metered=$0.00`.
- **Routing defect, instance 3:** `resolve-role-spawn.sh planner` returns
  `agent-tool opus fail-closed:planner-lane-field-missing` and the spawn-pin hook denies it — only
  2 docs in the repo carry a `planner_lane` field. The hook wins. The durable fix is in the TOOL
  and stays queued; this iteration's one skill edit went to the skill half.

## Parked on Mark

**`D-55` (OPEN)** — adversarial gob-decode scope for the compile-cache fix. **Its default (a) has
now been APPLIED as a controller routing call, so nothing is stalled.** Answer only to change
course: (b) or (c) supersedes the sprint's scope and the plan is revised first.

**Quota:** subscription/quota buckets only. Metered $0.00 of the $5 ceiling; no quorum round ran.
