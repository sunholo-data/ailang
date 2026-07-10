# M-EVAL-RIG-RELIABILITY — post-mortem + remediation plan

**Status:** Planned (2026-07-02)
**Trigger:** "This was working, then we descended into bugs and fixes for bugs." A multi-day arc
trying to run ONE controlled A/B on the motoko eval rig, blocked by a cascade of failures. This
doc stops the reactive firefighting, names what actually went wrong, and lays out a plan so it
can't recur.

---

## 1. What happened (the cascade)

1. **v0.27.0 shipped a breaking lexer change** — M-TERMINAL-IO (`37b1de765`) made the 4-hex
   `\uXXXX` string escape a hard parse error (only `\u{HEX}` accepted). *[FIXED — M-LEXER-U4-COMPAT]*
2. **It silently bricked the motoko harness.** The motoko `.ail` core has a `—` (em-dash);
   it stopped parsing → `ailang run supervisor.ail` died at module-load → **every motoko run
   produced 0 events for ~35 hours.**
3. **The break was invisible and looked like "flaky infra".** It manifested as: a 9h-wedged
   rotation chunk (the filler stuck retrying dead runs), `api_error`s, stalled experiments,
   port-8080 zombies. The leaderboard stayed green ("motoko 93.4%") because `--skip-existing`
   froze pre-break passing results.
4. **The operator firefought symptoms for many turns** — :8080 zombies, rig-lock steal
   collisions, ollama model-pinning, the watchdog — before root-causing the lexer.
5. **After the lexer fix, latent bugs surfaced:**
   - The rig-watchdog never caught wedges — two bugs (`set -u` crash on a short/empty `ps etime`;
     `etime_secs` was passed the pid but parsed it as an etime string → returned the pid AS
     seconds). *[FIXED — M-RIG-WATCHDOG-WEDGE followup]*
   - **docx result-recording is broken** — a real `max_steps` run (463 events, 59 steps) is
     recorded as `api_error, 0ms`. docx-specific (balanced_parens records correctly). *[OPEN]*
   - The A/B session-capture contaminates each arm with concurrent-run sessions (a stray
     `binary_tree_sum` landed in the treatment set). *[OPEN]*
6. **Net:** we can now *run* docx (lexer fixed, verified: balanced_parens 1/1, docx 463 events),
   but we can't *measure* it — and the episode consumed days.

---

## 2. Why a working system descended into bugs (systemic root causes)

The rig is three layers with **no enforced integration contract**: **AILANG** (language) ×
**motoko** (harness, a fork under `mk-ast`) × the **rotation infra**. Every failure above traces
to one of these gaps:

- **RC1 — No breaking-change guardrail.** AILANG changes can silently break motoko because the
  motoko `.ail` core is NOT compiled or run in AILANG's CI. A one-line lexer change bricked the
  rig for 35h with **zero CI signal**. This is the primary root cause.
- **RC2 — No health signal; failures bank silently.** A run that produces 0 events, or an arm
  that is all-`api_error`, is banked as *data* rather than raising an alarm. A total outage is
  invisible until someone reads raw session logs.
- **RC3 — Stale data hides regressions.** `--skip-existing` + version-banking froze the
  leaderboard at pre-break values; the dashboard said "healthy" while every live run failed.
- **RC4 — Unreliable measurement.** docx grading records real runs as `api_error 0ms`, so even
  when runs work the pass-rate is garbage — the *frontier* benchmark can't be measured at all.
- **RC5 (process) — Symptom-down firefighting.** The operator chased downstream symptoms instead
  of first asking "does the harness's code still compile/run?" That turned a ~5-minute root-cause
  (`ailang check` the motoko core) into a multi-day slog. (See memory
  `feedback_dont_overreach_consolidate_restart`, `feedback_ground_conclusions_in_data`.)

---

## 3. Remediation plan (prioritized)

### P0 — Fail fast (prevention; do first, cheap, highest-value)
- **CI integration smoke test.** On every AILANG change: (a) `ailang check` the motoko `.ail`
  core (`mk-ast/src/core/supervisor.ail` + deps); (b) run ONE agent-mode benchmark end-to-end and
  assert **>0 session events** and a **non-`api_error` finish_reason**. This single gate would have
  failed the v0.27.0 PR instead of surfacing on the rig 35h later. *(The lesson from
  M-LEXER-U4-COMPAT.)*
- **Rig health alarm.** Emit an alert (agent message / dashboard flag) when a run banks 0 events,
  or when a chunk's `api_error` rate exceeds a threshold — instead of silently banking garbage.

### P1 — Make docx measurable (unblocks the mission)
- **Fix docx result-recording** (RC4). Find why `docx_reimplement` grading records `api_error
  0ms` for a completed `max_steps` run — the agent clearly ran (463 events), so it's the grade
  path, not the agent. Record the true outcome (pass / fail / max_steps + real duration/tokens).
- **Fix A/B session-capture.** Scope each arm's session list to the arm's own runs (benchmark +
  pid/time), not a naive before/after `comm` diff that catches concurrent sessions.
- **De-stale the metric.** Don't `--skip-existing` across a binary/lexer change — re-eval on the
  current binary; or make the leaderboard visibly flag "stale (pre-vX)". (Ties to
  `project_os_rolling_stale_eval_data`.)

### P2 — De-fragile the rig (reduce the failure surface)
- **Isolate experiments from the rotation.** A dedicated experiment run-path (or rely on the
  rig-lock now that the watchdog works) so an A/B never fights the continuous filler.
- **Verify the watchdog in the wild.** It's fixed but only synthetically tested — confirm it
  kills a real long-running wedge, not just a fake.

### Process discipline (non-code)
- **When the rig "goes flaky," first verify the harness still compiles + runs** (`ailang check`
  the motoko core + a 1-run smoke) BEFORE diagnosing infra. Root-cause up the stack, not
  symptom-down.

---

## 4. Status of the concrete bug backlog

| Bug | Root cause | Status |
|---|---|---|
| `\uXXXX` parse error (35h motoko outage) | v0.27.0 lexer tightening, no consumer test | **FIXED** (M-LEXER-U4-COMPAT) |
| Watchdog never kills wedges | `set -u` crash + pid-vs-etime in `etime_secs` | **FIXED** (M-RIG-WATCHDOG-WEDGE) |
| docx runs recorded as `api_error 0ms` | grade path (RC4) | **OPEN — P1** |
| A/B arm session contamination | naive before/after capture | **OPEN — P1** |
| Stale leaderboard (motoko "93.4%") | `--skip-existing` froze pre-break data (RC3) | **OPEN — P1** |
| No CI guardrail (the meta-cause) | motoko core not in AILANG CI (RC1) | **OPEN — P0** |

---

## 5. Decision needed

The mission (motoko convergence, measured on docx) is blocked on **P1** (docx must be measurable).
**P0** prevents recurrence. Suggested order:
1. **P0 CI smoke test** (cheap, stops the next silent break).
2. **P1 docx recording fix** (so the frontier is measurable at all).
3. Then resume the convergence-card A/B with clean measurement + larger N.

The rig is currently **stopped** (rotation filler disabled: re-enable with
`launchctl bootstrap gui/$(id -u) ~/Library/LaunchAgents/dev.ailang.os-rotation-filler.plist`).
