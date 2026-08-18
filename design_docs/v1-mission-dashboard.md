# Mission Dashboard — V1
> Snapshot only; history lives in `v1-mission.md` (STATUS) and `v1-mission-log.md`.
> Namespaced at iteration 216: the shared `mission-dashboard.md` was one literal that every
> mission overwrote (frictions at iterations 212–215). Motoko's snapshot stays in its own file.

**Updated**: 2026-08-18 ~03:30 local (iteration 219)

## Now
- **v0.33.1** · `dev` **GREEN** on the required set. `origin/dev` `6ff68eda9`.
- **`govulncheck-filter`'s exit code 2 was over-subscribed, and half its refusal surface was
  unpinned.** `decide()` has exactly four `return 2` branches and all four return the *same code*,
  so an arm asserting `wantCode: 2` alone passes for any of the other three — and reads as coverage.
  Two branches (duplicate allowlist `id:`, unparseable stdin) had no test at any level. Fixed in
  `#765` → **`6ff68eda9`** (21 checks, zero not-green, 4/4 required). Closes `#727`.
- Both new arms assert a **branch-unique message**, not the shared code, and are built so no earlier
  branch can fire first. `TestDecideExitCodes` **5 → 7 arms**.
- **The `-skip` arm is what proved the pins.** Each mutant (LANDED by sha256, BUILDS by `go build`)
  redded only its own arm *and* left `-skip TestDecideExitCodes` at rc=0 — the check that separates
  "my test killed it" from "a bystander redded".
- Ubuntu `test` log names both new arms 4× against a 4× control, so the pins are proven on CI.
- **Sweep orphans: 2 of 15 dispositioned, and they came out opposite ways** — `#696` already-fixed
  (iter-216), `#727` real (iter-219). That is the argument for per-issue ghost discipline over
  batch-closing.
- ⚠ **Standing non-required red**: `SonarCloud Code Analysis` = `failure` on all four analysed `dev`
  tips. **Third consecutive iteration naming it un-triaged.** It does not block; nobody has looked.

## Next
1. **13 remaining sweep orphans** — mission-infra lane finishes with `#708` (design-quorum records
   no per-reviewer tokens, so Gate 3's telemetry token mandate is unsatisfiable) and `#687`
   (`⚠ Binary may be stale` mis-fires in every fresh worktree, including this loop's own).
2. **NEW `[world-DEMAND]` row** — `ailang#764`: `serveapi` is an API seam but not a *dependency*
   seam. Confirmed first-party: its only non-stdlib import is `internal/apiserver`, closing over
   **486** non-stdlib packages. Blocks World's item 5. Needs a design doc, not a controller fix.
3. Triage the SonarCloud red far enough to say whether it is real — it is now earning a pick on
   its own count.

## Loop
- Controller `claude:claude-opus-5`, inline. No designer / planner / executor / evaluator / quorum /
  GPU lane fired: the queue row specifies per-issue triage-lite. **metered $0.00**.
- Billing CLEAN; gh `sunholo-voight-kampff`; running skill byte-identical to origin (237334 B).
- ⚠ **Codex quota dry until 2026-08-20 05:34** — V1 remains on a single controller lane.

## Parked on Mark (issue rotates weekly — see `~/.ailang/state/mission-gh-issue`, now `#745`)
- **11 OPEN ledger rows**: `D-1`, `D-2`, `D-8`–`D-14`, `D-COV-1`, `D-18`.
  Generate the current list with `scripts/mission_decisions.sh --open` — never quote a range.
- **`D-18` is the one with a live cost**: two missions share this repo with no claim protocol, so a
  red blocking both gets fixed twice (`#758`/`#759`, 3m49s apart). Pick a mechanism (A: claim file
  under `~/.ailang/state/`; B: a `[claimed]` marker on the tracking issue; C: accept it as cheap).
- Nothing new is parked this iteration; `#764` entered the queue on measurement, not as an ask.
