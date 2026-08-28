# Mission Dashboard — V1

_Snapshot after iteration 296 (2026-08-28 02:50 CEST). Overwritten each iteration; history lives in the charter STATUS block and `v1-mission-log.md`._

## Latest
- **Release**: v0.34.0 · `dev` @ `d5305fa79`
- **dev CI**: three named workflows `success`; SHA-addressed set `checks=16`
- **`SonarCloud`**: `failure` — **inherited**, five consecutive analysed commits, non-required. Named, not the pick.

## In flight / next
- **PARKED on `D-47`** — `m-openrouter-session-chain-registration`. PR [#945](https://github.com/sunholo-data/ailang/pull/945) → `d86399f0a`, **DRAFT, deliberately not merged**. The registration works at chain level; at stage level the mechanism **misattributes** spans, because the wire `session_id` is chain-grained while `sessions.session_id` is a PRIMARY KEY, so N benchmarks in one `eval-suite` run collapse onto one row. Evaluator FAIL 58/100, both blocking findings reproduced first-party.
- **NEXT**: `m-prompt-freeze-mirror-all-versions` — decision answered (EXTEND), design already written by iteration 293's planner.
- Then: `m-git-binary-resolution-sweep` (doc written iter-294, **quorum not yet run**).

## Loop cadence + routing
- Controller `opus` · executor `codex:gpt-5.6-sol` (probe rc=0) · evaluator `sonnet`, own worktree · designer rotation `claude:claude-fable-5` → `pi:ollama/kimi-k3:cloud`.
- **Fable diet UNSPENT** this iteration (no doc authored); rotation pointer unchanged.
- generator≠judge held on both axes: codex + opus generated, sonnet judged and refuted both.
- metered **$0.00** of $5.

## Parked on Mark — 5 open decisions
- **`D-47` (NEW)** — OpenRouter session registration: **(a)** chain-only (correct, small, no misattribution, strictly better than today's zero resolution) · **(b)** per-request unique correlation id on the wire (full fix, changes eval-path wire bytes) · **(c)** redesign the join. Blocks the row above.
- **`D-46`** — who reconciles the `M-MISSION-LOOP-UNIFIED-TELEMETRY` sprint JSON: `mine` or `loop`?
- **`D-44`** — may `ai_check.go`'s verify denominator be corrected, given it moves a KPI with a banked baseline?
- **`D-43`** — should `std/string.charAt` itself become total, or does the new `charAtOpt`/`charAt_or` pair close it?
- **`D-42`** — standing authorization to reconcile this checkout to `origin/dev` unattended? (Asked by motoko four times too.)

## Health notes
- Main checkout is **behind** origin (`0f9abeee1` vs `d5305fa79`) with 3 dirty files, incl. Mark's uncommitted sprint-JSON reset. `D-42` applies; every write this iteration went to a worktree branched from `origin/dev`.
- PR [#942](https://github.com/sunholo-data/ailang/pull/942) is **not this mission's** (0 charter/log mentions vs control 3/4 for `#939`) — a concurrent attended session's. Left alone.
- Quota posture: no Anthropic or codex limit hit this iteration.
