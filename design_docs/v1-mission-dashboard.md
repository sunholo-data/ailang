# Mission Dashboard — V1
> Snapshot only; history lives in `v1-mission.md` (STATUS) and `v1-mission-log.md`.
> Namespaced at iteration 216: the shared `mission-dashboard.md` was one literal that every
> mission overwrote (frictions at iterations 212–215). Motoko's snapshot stays in its own file.

**Updated**: 2026-08-17 ~08:10 local (iteration 216)

## Now
- **v0.33.1** · `dev` @ `3a75ec7d2`; PR `#741` merged, 4/4 required contexts green.
- Iteration 216 **LANDED** a CI guard that the controller model pin is *exported*, not merely
  assigned (`#696`) — closing it after ghost-discipline showed it was REAL when filed and
  already FIXED by `de0e41099`.
- Four-iteration park streak (212–215) broken by the weekly sweep, not by a ledger decision.

## Next
1. **14 remaining sweep orphans** (queue row `m-sweep-orphans-2026-08-17`), mission-infra lane
   first: `#727` (untested `decide()` refusal branches), `#708` (quorum records no per-reviewer
   tokens), `#687` (stale-binary mtime heuristic mis-fires in every worktree).
2. Everything on the ordered frontier stays gated by the 10 OPEN ledger decisions below.

## Loop
- Controller `claude:claude-opus-5` (`probe ok`). Planner codex→`opus`, executor codex→`pi` by
  the ratified `#611` chain; neither heavy role fired this iteration.
- Billing CLEAN; metered **$0.00**; running skill byte-identical to origin at Gate 1.
- ⚠ **Codex quota dry until 2026-08-20 05:34** — V1 is on a single controller lane (Anthropic
  weekly bucket, resets Mon 07:00 local). 45 fires were refused over 2026-08-16 15:00→08-17
  07:19 with BOTH lanes exhausted; the driver refused loudly and never billed the API.
- ⚠ One died-mid-flight slot found (2026-08-16 15:00:31, no terminal log line). Its uncommitted
  park record was superseded and discarded; backup in `/tmp/iter216-orphan-backup`.

## Parked on Mark (issue rotates weekly — see `~/.ailang/state/mission-gh-issue`)
- **10 OPEN ledger rows**: `D-1`, `D-2`, `D-8`–`D-14`, `D-COV-1`.
  Generate the current list with `scripts/mission_decisions.sh --open` — never quote a range.
- **NEW this iteration**: the main checkout is **1 ahead / 15 behind** `origin/dev`, so the
  ratified `D-16` fast-forward authorization does **not** apply (any ahead-state triggers
  Critical Principle 0). Consequence: iteration 216's Gate-5 skill edit reaches `origin` but
  **not the running skill**, which resolves through that checkout's working tree.
