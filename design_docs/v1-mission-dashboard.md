# Mission Dashboard — V1

*Snapshot, overwritten every iteration. History lives in `v1-mission.md` (STATUS) and `v1-mission-log.md`.*

**Last iteration:** 287 · 2026-08-26 · controller `opus`

## Release
- **v0.34.0** shipped 2026-08-26. Next planned bucket: `v0_35_0`.

## In flight / next
1. **`m-fmt-printer-line-width-limit` M0+M1+M1b — LANDED `0c7f58351` (PR #918), evaluator PASS 85/100.**
   `writer.effectiveCol()` returns the pending indentation at `atBOL`, so the width predicate no longer
   reads a column the indentation has not reached. **M2 (continuation layout) + M3 (corpus reformat)
   remain unbuilt** — AC3/AC8 unmet by design, doc stays in `planned/`, residual ~116 lines >120 runes.
2. **NEXT: M2 then M3** — M3 before M2 would bank lines M2 would have wrapped.
3. **NEW `m-fmt-measurement-att-isolation-unpinned`** (evaluator) — `newMeasurementPrinter`'s `att: nil`
   is documented load-bearing (V21/V23) with **zero** coverage; the mutant reinstating attachment
   inheritance survives the whole suite. **NEW `m-feedback-dispatch-workspace-path`** — feedback still
   does not route: `#900` closed 14:27:39Z on the `AILANG_AGENT_ID` cause, the next dispatch failed
   19:18:40Z on a different one, `chdir /workspace/task-9d538100/packages/ailang-parse: no such file`.
4. `m-fmt-measurementerr-propagation-no-killer` · `m-motoko-lane-enumerator-field-order-blind` ·
   `m-verify-stdlib-wrapper-exit-propagation-unpinned` · `m-skills-parity-no-ci-gate` ·
   `m-eval-suite-agent-tempdir-unguarded` · `m-weekly-sweep-orphans-2026-08-26`.

## Loop health
- **ITERATION 286 DIED MID-FLIGHT, RECORDING NOTHING** — stall watchdog killed it 19:49 (`rc=143`,
  *"idle with a descendant alive ≥2400s"*) after it had built M0+M1+M1b, pushed to #918 and made an
  evaluator worktree. **0** charter rows, **0** log entries (control: 285 → 3). Standing-rule-7 shape;
  the watchdog failed LOUDLY, so it was recoverable. 287 verified and landed it rather than redoing it.
- **The #918 red was NOT our code**: required `test` failed at step 6 *"Download all Go modules"* —
  `sum.golang.org` HTTP/2 `INTERNAL_ERROR`, before any repo command. Re-run on a byte-identical tree:
  **failure → success**. `bytedance/sonic` 0× in diff, 0× in `go.mod` (controls 79, 2).
- **Driver ran UNPINNED** again; the sibling motoko mission landed the cause in `#923` mid-iteration.
- **Routing:** controller `opus` · designer/planner/executor **not re-spawned** (Gate 2: verify-and-land
  inherited work, don't redo) · evaluator `sonnet`, own worktree. generator≠judge held: codex wrote it,
  sonnet judged it. **Running skill == origin** (`cmp` rc=0 via resolved `readlink`; inode `51683298`).

## Parked on Mark
- **`D-41` (the ONLY open row; 41 total, 40 resolved)** — may an ACTIVE prompt version be edited in place,
  or must a content change bump the version? Bears on eval-baseline reproducibility.
- **SonarCloud has no token on this rig**; standing red, 3 consecutive analysed commits.

## Quota / spend
- Iteration 287 metered **$0.00** of $5. Buckets: opus (controller), sonnet (evaluator). Fable **UNSPENT**.
