# Mission Dashboard — V1

_Snapshot, overwritten each iteration. History lives in `v1-mission.md` (STATUS) and `v1-mission-log.md`._

**Iteration 332 · 2026-09-05 · latest release v0.35.1**

## In flight
- **`m-compile-cache-unverified-artifacts`** — **M3/4 landed** (`d14bd42cc`, judge PASS 96/100).
  M1 iter-330, M2 iter-331, M3 iter-332. **Next: M4** (route integrity diagnostic + MCP
  regression, 0.75 d) — the last milestone; issue #1046 closes with it.

## Next up (banked)
1. **M4 of the compile-cache sprint** — finishes a CONFIRMED public correctness bug (#1046).
2. **`m-cachesrc-cognitive-complexity`** — SonarCloud new-code maintainability red inherited by
   M3/M4; non-required, so it gates nothing, but it is ours and dated.
3. **`ci-red-mission-loop-workbench`** — HANDED OVER to motoko (its PR #1055 covers all three
   remaining reds). Resume = #1055 merged, then close the row.

## Loop health
- dev CI: 3 non-required reds, all base-inherited and all motoko's (`test-windows`,
  `Build windows-latest`, `launchd drivers (bash 3.2)`). All four REQUIRED contexts green.
- V1 fixed the one required red this fire (`test` / `TestDriverCopiesDoNotMultiply`) because a
  required context blocks every PR in the repo.
- Two other sessions are active in this repo: motoko's loop and a concurrent attended session
  landing the mission workbench. `origin/dev` moved 4 commits mid-iteration.

## Routing / cost
- Controller opus · designer + planner NOT spawned (doc and plan already exist) · executor
  `codex:gpt-5.6-sol` · evaluator `sonnet`. Designer rotation pointer parked at `codex:gpt-6-astra`.
- `metered=$0.00` of the $5 ceiling. Fable budget unspent. No quorum, no GPU.

## Parked on Mark
- **`D-55`** — bound adversarial gob decode, or ship the correctness fix? Its pre-registered
  default (a) has now carried three milestones; the row stays OPEN because the loop may not
  resolve a ledger row on its own behalf. A (b)/(c) answer still supersedes the sprint's scope.

## Goal distance
**N = 13 design docs before v1.0.0 (±0 this iteration).** A doc leaves the count when it LANDS;
this one is three of four milestones in.
