# Fleet mission — iteration 41, 2026-10-15

- Latest released binary: v0.54.11. dev: `7c10ce00d` (this iteration's last landing).
- LANDED this fire: the whole blocked stack.
  - #1824 → `514ca3bc3`: quorum verdict salvage (built iteration 36).
  - #1839 → `10cb2fae0`: repo-scoped quorum artifact dir (built iteration 38; replaces #1825).
  - #1840 → `7c10ce00d`: slot verdict UNCLASSIFIED and the Gate 1 stranded-record rule (built iteration 39; replaces #1828).
- Judge (sonnet, Agent tool, own worktrees): PASS 90 / 88 / 94 / 94. 4 tickets resolved. #941 closed.
- Fresh CI on the un-stacked #1825 found 3 defects that macOS-only verification missed: Linux bash 5, Windows `/bin/bash` and POSIX-fs, `check-home-isolation`. All fixed before the merge.
- Dev required `test` flakes on `internal/runner` PTY/signal lifecycle tests (4 of 9 runs this fire; green on rerun). V1's code; V1 is disabled.
- Next (banked): #16 `quorum:tier2-salvage-fold` (not live until a caller exists), then tickets by `slots_lost`; the newest are `worktree:native-background-terminal-marker-missing` and `messaging:pubsub-notify-wrong-project` (world, 10-15).
- Parked on Mark:
  - D-FLEET-16: skill source (A recommended / C default).
  - D-FLEET-18: `tools/eval` scope for #13 (A recommended / C default).
  - D-FLEET-17 is moot (both causes fixed outside the fleet); Mark may close it.
- `blocking=all` `agent-tool:sonnet-unavailable` stays PARKED-ON-LANE (codex-controller native spawn; claude controllers spawn sonnet fine).
- Cadence: fleet 6h, ticket-driven. Thread #1768; no new human directive. Nothing reloaded.
- Routing this fire: controller opus (glm probe → opus), executor haiku via `claude-sub`, evaluator sonnet. codex and openrouter over ration. Metered $0.
- Clauses: 1 unmeasured, 2 unmet, 3 met (48 open), 4 prior evidence, 5 preserved.
