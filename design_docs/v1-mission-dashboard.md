# Mission Dashboard — V1
> Snapshot only; history lives in `v1-mission.md` (STATUS) and `v1-mission-log.md`.
> Namespaced at iteration 216: the shared `mission-dashboard.md` was one literal that every
> mission overwrote (frictions at iterations 212–215). Motoko's snapshot stays in its own file.

**Updated**: 2026-08-18 ~00:45 local (iteration 218)

## Now
- **v0.33.1** · `dev` **GREEN** on the required set. `origin/dev` `c307db03b`.
- **The changelog gate certified a link to a file that did not exist.** Its link check was guarded
  by `[ -n "$ACTIVE" ]`, so an empty `ls changelogs/ | grep current` skipped it entirely: a missing
  `changelogs/` directory measured **rc=0**, printing `✓ CHANGELOG.md is index-only and links
  changelogs/` — blank filename in a success message. Same silent drop the gate exists to prevent,
  from the far end: it watched the index filling up, nothing watched whether the archive those
  entries move *into* still existed. Fixed in `#762` → **`c307db03b`** (21 checks, zero not-green).
- **The queue row under-stated its own work.** It framed all three `#758` deltas as missing test
  arms. Measured: `## [Unreleased]` and `## [v0.32.0]` already refuse (`rc=1`) — shape pins only —
  while the third was a missing *refusal branch*. Measuring before routing is what found it.
- Self-test **9 → 13 arms**, and the ubuntu `test` job's own log names all four new ones, so the
  pins are proven on CI rather than only on darwin/arm64.
- ⚠ **Standing non-required red**: `SonarCloud Code Analysis` has been `failure` on both recently
  analysed `dev` tips. Named, not attributed — it does not block, and nobody has triaged it.

## Next
1. **14 remaining sweep orphans** (`m-sweep-orphans-2026-08-17`) — mission-infra lane first
   (`#727`, `#708`, `#687`), each ghost-disciplined at HEAD before routing.
2. Triage the standing SonarCloud red far enough to say whether it is real.
3. Everything on the ordered frontier stays gated by the 12 OPEN ledger decisions below.

## Loop
- Controller `claude:claude-opus-5`, inline. No designer / planner / executor / evaluator / quorum /
  GPU call: the queue row itself specifies controller-inline mechanical work. **metered $0.00**.
- Billing CLEAN; gh `sunholo-voight-kampff`; running skill byte-identical to origin.
- ⚠ **Codex quota dry until 2026-08-20 05:34** — V1 remains on a single controller lane.
- GitHub's 2026-08-17 Partial System Outage has cleared; iteration 217's owed `Set up job` re-run is
  moot — `dev` has since gone green twice on its own runs.

## Parked on Mark (issue rotates weekly — see `~/.ailang/state/mission-gh-issue`, now `#745`)
- **12 OPEN ledger rows**: `D-1`, `D-2`, `D-8`–`D-14`, `D-COV-1`, `D-18`.
  Generate the current list with `scripts/mission_decisions.sh --open` — never quote a range.
- **`D-18` is the one with a live cost**: two missions share this repo with no claim protocol, so a
  red blocking both gets fixed twice (`#758`/`#759`, 3m49s apart). Pick a mechanism (A: claim file
  under `~/.ailang/state/`; B: a `[claimed]` marker on the tracking issue; C: accept it as cheap).
- **Resolved since 217**: the main checkout is no longer ahead of `origin/dev` (0 ahead / 2 behind,
  dirty only in the two rig-synced `docs/static/benchmarks/os/*.json`), and the running skill is
  byte-identical to origin. Ratified `D-16`'s conditions now hold; no reconcile was needed this
  iteration because the driver pin already equalled `origin/dev`.
