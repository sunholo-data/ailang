# Mission Dashboard — V1
> Snapshot only; history lives in `v1-mission.md` (STATUS) and `v1-mission-log.md`.
> Namespaced at iteration 216: the shared `mission-dashboard.md` was one literal that every
> mission overwrote (frictions at iterations 212–215). Motoko's snapshot stays in its own file.

**Updated**: 2026-08-18 ~06:10 local (iteration 220)

## Now
- **v0.33.1** · `dev` **GREEN** on the required set. `origin/dev` **`904cb9b0d`**.
- **`design-quorum` was discarding the token counts it had just read**, so a quorum stage could
  only ever post zeros into the Gate-3 chain ledger — the exact iteration-190 signature the
  mandate exists to prevent (`$0.0570`/`$0.0507` at **zero** tokens). Not unrecorded: `run.go`
  spent `resp.InputTokens/OutputTokens` on `estimateCost` and dropped them. Fixed in `#767` →
  **`904cb9b0d`** (21 checks, zero not-green, 4/4 required). Closes `#708`.
- **Both tiers dropped the same counts**, so it shipped as one sweep: text tier, agentic tier
  (through `AgenticRun` + the caller adapter), synthesis totals, and `TokenAccountingGaps()` —
  which names any reviewer billed with zero reported tokens, loudly, without ever blocking a quorum.
- **The drill found a bigger hole than the issue described.** Zeroing the token mapping in the
  production `coordinator.ExecuteResult` → `AgenticRun` adapter left the **whole package green**:
  it sat behind `NewExecutorProvider`, so every test stubbed the runner and none reached it. The
  one place the executor's real counts enter the quorum had no coverage. Now extracted and pinned.
- **The artifact arm is the load-bearing one.** A `json:"-"` tag leaves every struct-level arm
  green while the written file stays exactly as tokenless as before — and `jq` over that file is
  the actual consumer. 10 arms, 8 drills, every inverse `-skip` rc=0.
- **Sweep orphans: 3 of 15 dispositioned** — `#696` already-fixed, `#727` real, `#708` real.
- ⚠ **Standing non-required red**: `SonarCloud Code Analysis` = `failure` on six consecutive `dev`
  commits. **Fourth consecutive iteration naming it un-triaged.** It does not block; nobody has looked.

## Next
1. **12 remaining sweep orphans** — `#687` closes the mission-infra lane (`⚠ Binary may be stale`
   is an mtime heuristic that mis-fires in every fresh worktree, including this loop's own), then
   the language/stdlib group (`#688`, `#689`, `#662`, `#646`, `#644`).
2. **`[world-DEMAND]`** — `ailang#764`: `serveapi` is an API seam but not a *dependency* seam
   (486 non-stdlib packages in its closure). Blocks World's item 5. Needs a design doc + quorum.
3. Triage the SonarCloud red far enough to say whether it is real — four iterations of naming it
   without looking is itself the argument for picking it.

## Loop
- Controller `claude:claude-opus-5`, inline. No designer / planner / executor / evaluator / quorum /
  GPU lane fired — mechanical, well-specified code work. **metered $0.00**.
- Billing CLEAN; gh `sunholo-voight-kampff`; running skill byte-identical to origin (237334 B).
- ⚠ **Codex quota dry until 2026-08-20 05:34** — V1 remains on a single controller lane.

## Parked on Mark (issue rotates weekly — see `~/.ailang/state/mission-gh-issue`, now `#745`)
- **11 OPEN ledger rows**: `D-1`, `D-2`, `D-8`–`D-14`, `D-COV-1`, `D-18`.
  Generate the current list with `scripts/mission_decisions.sh --open` — never quote a range.
- **`D-18` is the one with a live cost**: two missions share this repo with no claim protocol, so a
  red blocking both gets fixed twice (`#758`/`#759`, 3m49s apart). Pick a mechanism (A: claim file
  under `~/.ailang/state/`; B: a `[claimed]` marker on the tracking issue; C: accept it as cheap).
- Nothing new parked this iteration.
