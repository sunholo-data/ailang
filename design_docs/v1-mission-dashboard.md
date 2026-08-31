# Mission Dashboard — V1

_Snapshot, overwritten every iteration. History lives in `v1-mission.md` STATUS + `v1-mission-log.md`._

**Last iteration:** 308 · 2026-08-31 · dev CI-red preemption ×3 (V1 owns `sunholo-data/ailang`)

## Now
- **Landed:** #969 → `c29ec1d00` (pi asset inventory pinned by exact filename list) · #977 →
  `9f267cf1f` (driver-notify lab never supplied `STATE_DIR`; 22 arms aborted under `set -u`; 5/22 →
  **27/0**, judge PASS 88/100).
- **Parked (draft):** #971 — process-tree discovery deadline decoupling. Structurally sound but does NOT
  deliver the determinism its message claims: identical commit `8a384e81b` ran **success then failure**.
- **Open findings:** **#975** — that probe arm now has **FOUR** failure modes in five days; root cause
  unconfirmed, not locally reproducible, and it reddened #977 (which touches no `tools/eval/` file), so it
  gates anything needing this required job. **#978** — episode-gating suppression + recovery-reset have
  zero coverage. **#981** — the Gate-0 watermark defect below.

## Next
1. Diagnose **#975** on a CI runner (gated `PS4`/`set -x` around `descendant_pids` + the stub) before
   any further calibration; consider an interim CI quarantine. Then #971 can be revisited.
2. **#978** coverage arms (needs an optional external state dir in `run()`).
3. Weekly-sweep orphans batched: #959, #960, #962, #963, #941 (triage-lite, normal ordering).
4. **D-50 is APPROVED** — `m-coordinator-child-env-opencode-retry-storm` is authorized to execute.
   #968 holds the recovered design + quorum artifacts and needs a rebase (CONFLICTING).
5. D-49 (a) and D-30 are answered and owed queue rows; **#981** — a fire that dies must not advance the
   Gate-0 watermark (Mark's explicit ask).

## Parked on Mark
- **NONE.** D-49 and D-50 are **RESOLVED** (`a121d8d8c`, Mark attended 2026-08-31): D-49 = **(a)
  repo-local wins**, duplicates suppressed; D-50 = **approve**, `execute sprint` authorized. Iteration
  308's own report re-asked both — Gate 0 read a watermark equal to Mark's comment instant and `--since`
  is exclusive, so the answers were unreachable. Retracted; defect filed as **#981**. Do not re-ask
  these, nor D-30 (answered 2026-08-26, also unconsumed for five days).

## Loop + routing
- Bookkeeping issue **#972** (rotated from #852 this iteration — Monday boundary + 83 comments).
- Roles: controller `claude-opus-5` · executor `pi:ollama/deepseek-v4-flash:0731-cloud` (verdict
  `wall_timeout` rc=13, FLAGGED; diff complete, so verified not discarded) · evaluator `sonnet` ×3.
  Designer/planner did not fire — a CI-red preemption produces no design doc or plan.
- **Codex lanes quota-exhausted** this fire (probe rc=1, resets ~09:36) — `PARKED-ON-LANE`, no decision needed.

## Cost
- metered **$0.00** (executor flat-rate on Ollama Cloud, 1.58M in / 33.8k out) · buckets: opus, sonnet ×3.
