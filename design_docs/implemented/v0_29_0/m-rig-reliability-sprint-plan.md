# Sprint Plan: M-RIG-RELIABILITY — make the rig catch breaks + measure docx

**Source design doc:** [m-eval-rig-reliability.md](m-eval-rig-reliability.md)
**Goal:** Stop breaking changes from silently bricking the rig (catch at PR time), and make docx a
trustworthy *measured* instrument. Sprint scope = P0 + P1; P2 is stretch.
**Risk/estimates:** honest and rough — M2 depends on what the grade path is actually doing.

---

## Milestones (recommended order)

### M1 — CI breaking-change guardrail (P0-1) · ~0.5 day · **START HERE**
The root cause of the whole cascade: an AILANG change bricked motoko with zero CI signal.
- **Do:** add a CI step (new `.github/workflows/motoko-smoke.yml`, or a step in `ci.yml`) that on
  every push/PR: (a) `ailang check`s the motoko `.ail` core (`mk-ast/src/core/supervisor.ail` +
  deps); (b) runs ONE agent-mode benchmark end-to-end (`balanced_parens`, proven-good) and asserts
  **>0 session events** and a **non-`api_error` finish_reason**.
- **Acceptance:** the job FAILS on a synthetic `\uXXXX`-style break (verify by temporarily
  reverting the lexer fix locally); passes on green. A helper script `tools/ci/motoko_smoke.sh`.
- **Note:** CI may not have the motoko fork / a local model — so gate the *run* part behind
  availability and always run the *compile* part (which catches the lexer-class break).

### M2 — Fix docx result-recording (P1-1) · ~1 day · unblocks the mission
Real `max_steps` docx runs (463 events) are recorded as `api_error, 0ms`. Suspected cause:
the multi-file reimplement grade path doesn't populate `compileOk/runtimeOk/stdoutOk/duration`,
so `CategorizeError` (metrics.go:136) falls back to `api_error`.
- **Do:** trace the `docx_reimplement` (grade_entrypoint) result path; populate the real metrics
  (compile/runtime/stdout + duration/tokens) so a `max_steps` run records as `max_steps`, and a
  passing run as a pass.
- **Acceptance:** re-run one docx via the harness → result JSON shows the true `finish_reason`
  (not `api_error 0ms`) and non-zero duration; a Go test on the reimplement grading path.

### M3 — A/B session-capture + rig health alarm (P1-2 + P0-2) · ~1 day · hardening
- **Session-capture:** scope each A/B arm's session list to its OWN runs (benchmark + pid/time),
  not the naive before/after `comm` diff that caught a stray `binary_tree_sum`. (`tools/ab_*.sh`)
- **Health alarm:** the rig raises an alert (agent message / dashboard flag) when a run banks 0
  events, or a chunk's `api_error` rate exceeds a threshold — instead of silently banking garbage.
- **Acceptance:** an A/B arm's session list contains only that arm's benchmark; a forced 0-event
  run triggers the alarm.

## Stretch (P1-3 / P2)
- De-stale the metric: don't `--skip-existing` across a binary change; flag "stale (pre-vX)".
- Verify the watchdog kills a REAL wedge in the wild (only synthetic-tested so far).

## Success metrics
- CI guardrail merged + demonstrably fails on a lexer-class break.
- One docx run records its true outcome (not `api_error 0ms`), with a regression test.
- A/B session-capture clean; health alarm fires on a 0-event run.

## Out of scope
- Resuming the convergence-card A/B (that's *after* docx is measurable — the next sprint).
- Reworking the rotation architecture (P2, later).
