# Fleet mission — iteration 25, 2026-10-07

- origin/dev `fcce2394c` at Gate 1; all 13 running mission-control files match origin (resolved symlink).
- **LANDED:** `blocking=all` ticket `agent-tool:mission-role-pins-unavailable`, #1635 → `0ceb1db01`, judged PASS 93 (0 blocking). Ticket resolved.
- Cause: the rig's codex config `shell_environment_policy.inherit = "core"` (since 2026-10-06 13:02) stripped every `MISSION_*` from codex-controller shells, so every role resolved `fail-closed`. It also dropped the pre-push scope guard's `core.hooksPath`, so the guard was **off for codex controllers** until this fix.
- Fix: per-variable `-c shell_environment_policy.set.NAME=…`, name allowlist minus secret denylist, plus hooksPath only. Rig config untouched.
- First live proof will be the next codex-controller fire's `codex-env: forwarded=N` driver log line.
- FLAG: the judge `claude-sonnet-4-6` is the same family as executor `claude-sonnet-5-5` (minimax/openrouter over ration).
- Follow-up candidate (possible routing policy, not built): a bare-alias evaluator pin still resolves to `agent-tool sonnet`/`opus` under a codex controller.
- Next: resume #1604 (gate0 self-notices); dev Linux `test` is green now. Then P1 #8 if the lanes admit it.
- OPEN decisions unchanged: D-FLEET-13 session-limit signature; D-FLEET-14 test clock; D-FLEET-15 main auto-sync.
- Dev Windows red (`TestTestCommandBytecodeFlags` et al.) persists: V1's lane.
- Record PR supersedes #1611 and #1612 (iterations 22–24 carried).
