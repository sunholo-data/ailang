# Mission Dashboard — V1

_Snapshot, overwritten every iteration. History lives in `v1-mission.md` STATUS + `v1-mission-log.md`._

**Last iteration:** 308 · 2026-08-31 · dev CI-red preemption ×3 (V1 owns `sunholo-data/ailang`)

## Now
- **Landed:** #969 → `c29ec1d00` (pi embedded-asset inventory pinned by exact filename list, closing a
  substring vacuity) · #977 → `9f267cf1f` (driver-notify lab never supplied `STATE_DIR`, which
  `149e47667`'s episode-gating reads inside the extracted blocks — all 22 arms aborted under `set -u`;
  harness-only, 5/22 → **27/0**, judge PASS 88/100).
- **Parked (draft):** #971 — process-tree discovery deadline decoupling. Structurally sound but does NOT
  deliver the determinism its message claims: identical commit `8a384e81b` ran **success then failure**.
- **Open findings:** **#975** — that probe arm now has **FOUR** measured failure modes in five days; root
  cause unconfirmed, not locally reproducible, and it reddened #977 despite that PR touching no
  `tools/eval/` file, so it gates anything needing this required job. **#978** — episode-gating
  suppression + recovery-reset have zero coverage (both guards deletable, suite still green).

## Next
1. Diagnose **#975** on a CI runner (gated `PS4`/`set -x` around `descendant_pids` + the
   `PROBE_TEST_PGREP_LOOP` stub) before any fourth calibration; consider an interim CI quarantine so a
   known-flaky instrument stops blocking correct work. Then #971 can be revisited.
2. **#978** coverage arms (needs an optional external state dir in `run()`).
3. Weekly-sweep orphans batched: #959, #960, #962, #963, #941 (triage-lite, normal ordering).
4. #968 (recovered retry-storm design + quorum artifacts) is CONFLICTING and waits on **D-50**.

## Parked on Mark
- **D-49** — Pi extension precedence: global `~/.pi/agent/extensions` vs repo-local `.pi/extensions`.
  (a) repo-local wins · (b) skip global for AILANG-repo sessions · (c) retire one channel.
- **D-50** — approve the narrowed `m-coordinator-child-env-opencode-retry-storm` recovery design and
  authorize `execute sprint`? Blocks #968.

## Loop + routing
- Bookkeeping issue **#972** (rotated from #852 this iteration — Monday boundary + 83 comments).
- Roles: controller `claude-opus-5` · executor `pi:ollama/deepseek-v4-flash:0731-cloud` (verdict
  `wall_timeout` rc=13, FLAGGED — diff was complete, so verified rather than discarded) · evaluator
  `sonnet` ×3 rounds. Designer/planner did not fire: a CI-red preemption produces no design doc or plan.
- **Codex lanes quota-exhausted** this fire (probe rc=1, resets ~09:36) — `PARKED-ON-LANE`, no decision needed.

## Cost
- metered **$0.00**. Executor ran flat-rate on Ollama Cloud (1,575,679 in / 33,785 out).
- Quota buckets: opus (controller), sonnet (evaluator ×3).
