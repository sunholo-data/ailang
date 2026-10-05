# Fleet mission — iteration 19, 2026-10-05

- Release v0.52.1 (CLI on PATH). Pin = origin/dev `9eac33b7b` at Gate 1; dev is now `e7628b05e` plus this record. The skill and all 12 resources match origin.
- **LANDED: heartbeat fix, #1578 `c55ca4398`.** All 18 stamp calls (`.claude` + `.agents`) use an absolute `MISSION_DRIVER_ROOT` guard. World's rc127 is gone. Judged PASS 96/95/96.
- **LANDED: rotate-log pair, #1580 `e7628b05e`.** The registry comes from the driver's absolute path, shared targets come from the loaded registry, and `--status` is retired. A real Windows hang was found and fixed on the way: the registry walk-up never stopped at `C:\` (`cffc0447a`, judged PASS 92).
- Tickets: 3 resolved this fire, 41 → 38 open. Clause 2 turnaround is still UNMET (≈8–9 days each).
- **Next READY:** P1 #5 pi-runner pre-dirty half, then the Phase 3a skill-resolution directive (after P1). The D-FLEET-12 pair is pre-authorized.
- **Ledger, 2 OPEN:**
  - D-FLEET-13: Anthropic "session limit" counted as capacity (rec A).
  - D-FLEET-14: pin `now` in the `TestOllamaQuota*` tests (rec A). It recurs **every weekend** and blocks all PRs to dev.
- Quota: codex, ollama and openrouter over daily ration; Anthropic subscription OK. Metered this fire: $0.00.
- Routing:
  - Executor `claude:claude-sonnet-5-5` via `claude-sub`.
  - Judges: `opus` via the Agent tool (×2). The minimax lane is over ration and sonnet-4-6 is the same family as the executor.
  - No designer or planner (judged builds plus a one-function guard).
- Weekly external-issue sweep done: 59 enumerated, 0 new queue items. #1306 got a verdict comment (half fixed).
- 3 coordinator approvals pending, untouched (operator-owned).
- Bookkeeping thread rotated: **#1584** (from #1380). Log: fleet-mission-log.md.
