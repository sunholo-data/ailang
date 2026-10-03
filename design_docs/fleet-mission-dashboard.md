# Fleet mission — iteration 15, 2026-10-03

- Release v0.52.0 (CLI on PATH). Pin = origin/dev; the skill and all 12 resources match origin.
- LANDED #1549 `27dab5bb4`: controller HTTP 402 → capacity (demote/re-walk → PAUSED-NO-CAPACITY), pause no longer posts a crash notice.
- Ticket `driver:controller-fallback-…-402-reads-as-crash` resolved (≈24h filed→resolved). First resolution since iteration 10.
- Evaluator pi:openrouter/minimax-m3 PASS 95/100. Done-gate green, with dry-runs under the world profile.
- NEW finding: an Anthropic "hit your session limit" is ALSO recorded CRASHED (killed iteration 15 attempt 1). Outside D-FLEET-11's scope, and the fleet cannot file a ticket for it. Needs Mark (D-FLEET-13).
- Next READY: heartbeat Revision 5 + fresh quorum (D-FLEET-10 = A) → rotate-log pair (D-FLEET-9 = A) → pi-runner pre-dirty.
- D-FLEET-12 pair (retired codex models; opencode quota pool) is pre-authorized, no ruling needed.
- Ledger: D-FLEET-10/11/12 normalized to RESOLVED; 1 OPEN (D-FLEET-13).
- Quota: codex and ollama over ration; OpenRouter admitted; Anthropic opus/sonnet subscription OK this fire.
- Metered this iteration $0.61 (quorum $0.28 in attempt 1, evaluator $0.33).
- Routing: planner via Agent opus. Executor sonnet-5-5 via claude-sub (the Agent alias was denied by the provider pin). Evaluator re-routed to pi minimax (generator≠judge).
- Retro: the charter done-gate now names the idle-sibling dry-run recipe. The pi judge sandbox blocks `mktemp -t` (backlog).
- 10 coordinator approvals pending, untouched (operator-owned).
- Bookkeeping #1380; log fleet-mission-log.md; design planned/m-controller-capacity-admission.md.
