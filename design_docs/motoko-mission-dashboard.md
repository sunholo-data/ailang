# Mission Dashboard — Motoko

*Snapshot, overwritten every iteration. History lives in the charter STATUS block and
[motoko-mission-log.md](motoko-mission-log.md). 30-second read for a fresh session.*

**Last iteration**: **17** — 2026-08-21 · landed iteration 16's orphaned M3+M5.
**Last release**: v0.33.1 (repo-wide; this mission does not release).

## In flight / next

- **Queue item 6** (`m-motoko-fmt-remeasurement-instrument`): **M1 + M3 + M4 + M5 LANDED**.
  **Only M2 remains — and it NEEDS THE RIG** (`AC-D1-live`: one fmt-lane run reaching
  `localhost:11434` with zero `openrouter.ai` connections, asserted on the connection, paired with
  an OpenRouter-lane positive control). Requires `rig.lock`.
- **If no rig slot** → item **7** (profile restoration design), then item **8** (repin the stale
  OpenRouter motoko models).
- **Deployment precondition still owed** (doc §6, issue `#558`): merging to `dev` does **not** put
  D1/D1b/the smoke gate on the rig — the installed plist runs `nightly-eval.sh` in place from V1's
  checkout. Verify at the path in the plist's `ProgramArguments`, never a working-tree path.
- **Gated behind Phase 0** (G1–G5 conjunctive, all still FALSE): items 10, 11, 12.
  **Parked on a green tree**: items 9, 13, 14.

## Loop health

- Cadence: launchd `dev.ailang.mission-motoko`, `StartInterval=43200` (12h), staggered against
  V1 (90 min) and World (4h).
- **Iteration 16 was killed by the driver's stall watchdog** (`idle with a descendant alive
  ≥2400s`, `rc=143`) with its work finished and unlanded. The watchdog behaved correctly — early
  kill, non-zero exit, failure posted to `#743` within the hour. Cost was one landing step.
- Last iteration's dev CI: green (16 exact-SHA checks, 0 not-green, a run exists).
- Note: `~/.claude/skills/mission-control` resolves to **V1's** checkout, which is currently
  1 commit ahead of origin with an unpushed skill edit. This mission executes its **own** repo-local
  copy, verified `cmp` rc=0 against origin.

## Routing (as configured)

controller `claude:claude-opus-5` · designer rotation pointer `claude:claude-fable-5` ·
planner/executor `codex:gpt-5.6-sol` (executor fallback `pi:deepseek-v4-flash-0731`) ·
evaluator `sonnet`. Metered budget $5/iteration; iteration 17 spent **$0.00**.

## Parked on Mark

**Nothing.** Decision ledger: 3 rows, **0 OPEN** (`scripts/mission_decisions.sh --check`).

## Quota posture

Bookkeeping issue `#743` (15 comments; rotates weekly Monday 07:00 **local**, not due).
No GPU held; `rig.lock` untouched for 5 consecutive iterations.
