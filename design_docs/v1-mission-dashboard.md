# Mission Dashboard — V1

_Snapshot, overwritten every iteration. History lives in `v1-mission.md` STATUS + `v1-mission-log.md`._

**Last iteration:** 308 · 2026-08-31 · dev CI-red preemption (V1 owns `sunholo-data/ailang`)

## Now
- **dev is GREEN** at `8bb982220`: `pending=0`, `launchd drivers` success. Only not-green is the
  standing **non-required** `SonarCloud Code Analysis`, inherited (also red on `84eb49237`, `3fc7be9b8`).
- **Landed this iteration:** #969 → `c29ec1d00` — pi embedded-asset inventory pinned by exact filename
  list (was two magic counts); closes the substring vacuity in the idempotence assertion.
- **Parked (draft):** #971 — process-tree discovery deadline decoupling. Structurally sound, but does
  NOT deliver the determinism its message claims: identical commit `8a384e81b` ran **success then
  failure** on `launchd drivers`. Not merged.
- **Open finding:** #975 — the `descendant discovery refuses on the real wall-clock deadline` arm has
  **three** measured failure modes in five days; root cause unconfirmed, not locally reproducible.

## Next
1. Diagnose #975 on a CI runner (gated `PS4`/`set -x` around `descendant_pids` + the
   `PROBE_TEST_PGREP_LOOP` stub) **before** any fourth calibration attempt. Then #971 can be revisited.
2. Weekly-sweep orphans batched: #959, #960, #962, #963, #941 (triage-lite, normal ordering).
3. #968 (recovered retry-storm design + quorum artifacts) is CONFLICTING and waits on **D-50**.

## Parked on Mark
- **D-49** — Pi extension precedence: global `~/.pi/agent/extensions` vs repo-local `.pi/extensions`.
  (a) repo-local wins · (b) skip global for AILANG-repo sessions · (c) retire one channel.
- **D-50** — approve the narrowed `m-coordinator-child-env-opencode-retry-storm` recovery design and
  authorize `execute sprint`? Blocks #968.

## Loop + routing
- Bookkeeping issue **#972** (rotated from #852 this iteration — Monday boundary + 83 comments).
- Roles this fire: controller `claude-opus-5` · designer `claude:claude-fable-5` (did not fire — no new
  doc) · planner `pi:ollama/kimi-k3:cloud` (did not fire) · executor `pi:ollama/deepseek-v4-flash:0731-cloud`
  · evaluator `sonnet` (2 rounds).
- **Codex lane quota-exhausted** this fire (usage limit, resets ~09:36) — planner/executor auto-routed
  to the pi/ollama lanes by the driver. `PARKED-ON-LANE`-class, no human decision needed.

## Cost
- metered **$0.00** this iteration. Executor ran flat-rate on Ollama Cloud (1,575,679 in / 33,785 out).
- Quota buckets touched: opus (controller), sonnet (evaluator ×2).
