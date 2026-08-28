# Mission Dashboard — V1

*Snapshot, overwritten each iteration. History lives in the charter STATUS block and the mission log.*

**As of**: 2026-08-28 06:50 CEST · iteration **297** · `origin/dev` = `5e5c77dee`

## Latest
- **v0.34.0** released. Last landing: **PR #949 → `5e5c77dee`** (iteration 297) — the prompt-freeze
  CI gate now checks **every** registry entry, not only frozen ones.
- Why it mattered: 59 registry entries, 58 frozen, **1 mutable — `v0.16.6`, which is also `active`**,
  i.e. the prompt every agent reads. Its `.md` could diverge, be deleted, or have its embedded mirror
  deleted, all at rc=0. Now rc=1 with a named violation. Merge-base immutability stays frozen-only.

## In flight / next
- **`m-openrouter-session-chain-registration`** — PARKED on **`D-47`**. PR #945 is a DRAFT; the
  mechanism misattributes spans rather than merely failing to resolve them. Do not re-route until D-47 lands.
- Next picks: `m-std-smt` (external feature request, needs a doc) · `sonarcloud-new-code-gate-red`
  (premise re-measured iter-294) · `m-git-binary-resolution-sweep` (doc written, quorum not yet run).

## Loop health
- Cadence nominal. Iterations 295/296/297 all completed end-to-end; no reaped slots.
- Routing this iteration: controller `opus` · designer `fable` (Agent-tool pin) · planner `opus`
  (lane derived verbatim) · executor `codex:gpt-5.6-sol` · evaluator `sonnet` (own worktree).
  generator≠judge held on both axes.
- **FLAGGED, instance 2 (after iter-294)**: the designer rotation's next entry is
  `pi:ollama/kimi-k3:cloud`, which the Agent tool cannot express, so the pointer did not advance.
  The rotation has one Agent-tool-expressible authoring lane. Routing-policy fix needs a human.
- `SonarCloud Code Analysis` has been `failure` on dev for 9+ consecutive commits. Non-required,
  named every iteration, never the pick. `sonarcloud-new-code-gate-red` is the queue row for it.

## Parked on Mark — 5 open decisions
| ID | One-line |
|----|----------|
| **D-42** | Standing authorization to reconcile this clone to `origin/dev` unattended? Local `dev` is now **21 behind**; a fleet-wide driver fix cannot reach a clone that never advances. |
| **D-43** | Should `std/string.charAt` itself become total, or does the new `charAtOpt`/`charAt_or` pair close it? |
| **D-44** | May `ai_check.go`'s verify denominator be corrected, given it moves a KPI with a banked baseline? |
| **D-46** | Who reconciles the `M-MISSION-LOOP-UNIFIED-TELEMETRY` sprint JSON — `mine` or `loop`? |
| **D-47** | OpenRouter session registration is chain-grained; the doc asks for stage-bound. Chain-only, per-request id, or redesign the join? |

## Quota / cost posture
- Iteration 297 metered spend: **$0.2358** of the $5 ceiling — two quorum rounds only.
- No managed_agents call, no pi lane, no GPU. codex lane probed rc=0 and was used for the executor.
