# Sprint plan — M-GATE0-WATERMARK-CRASH-SAFETY (#981)

- Status: **approved** (pre-approved by Mark for the attended P0 triage, 2026-10-26; no pause)
- Design: [`m-gate0-watermark-crash-safety.md`](m-gate0-watermark-crash-safety.md) (P0, no open freeze items)
- Issue: sunholo-data/ailang#981
- Base: `origin/dev` `8e634480f`, branch `p0/gate0-watermark-crash-safety`
- Size: ~0.5 day, ~520 LOC (≈170 script, ≈290 test, ≈60 skill/docs). Three serial milestones.
- JSON: `.ailang/state/sprints/sprint_M-GATE0-WATERMARK-CRASH-SAFETY.json`

## Relationship to the other Gate-0 sprint plans (not duplicates)

| Plan | Scope | Overlap with this sprint |
|---|---|---|
| `sprint-plan-gate0-self-notices.md` (#1160, landed `59c3e6a55`) | step **6a**: a second, no-authority read that surfaces the driver's own crash notices | Partial mitigation only: it tells the next fire *a fire died*; it does not stop the watermark skipping a directive. Its reader takes **line 1** of the watermark file — this sprint keeps line 1 a bare timestamp so 6a is untouched. |
| `sprint-plan-gate0-ledger-provenance.md` | rule 6(b): who flipped a ledger row (`mission_ledger_provenance.sh`) | None. Different instrument, same step. |

Neither is the plan for this design; this file is.

## Premises verified on origin/dev `8e634480f`

- `gate-0-preflight.md:264-267` (step 7's tail) still says write the newest `createdAt` "before routing".
- `scripts/mission_directives.sh:91,102` filter `.createdAt > $since` — **exclusive**; a comment at
  exactly the watermark is unreachable. #852 measurement: watermark `2026-08-30T08:11:12Z` equals the
  `createdAt` of Mark's D50/D49 comment `IC_kwDOP3Mz2c8AAAABReRotQ`.
- The script prints no comment identity, so nothing can deduplicate a re-read.
- The watermark file has one reader in code: `mission_gate0_self_notices.sh:245` reads line 1 only.
- `.agents/skills/mission-control/` is a **generated mirror** (`tools/launchd/sync-agents-skills.sh`,
  enforced by `test_agents_skills_sync.sh`); the source is `.claude/skills/mission-control/`.

## Deviations from the design (recorded, not silent)

1. Test path: the design names `tools/tests/test_gate0_watermark.sh`; `tools/tests/` does not exist and
   the repo convention (and the fleet pre-push scope guard) is `scripts/test_mission_*.sh`. The suite is
   `scripts/test_mission_directives.sh`, wired into `make test-launchd-drivers`.
2. Watermark advance rule made precise: `--mark` re-fetches the issue and sets line 1 to the oldest
   still-**unprocessed** allowlisted `createdAt` (or the newest processed one when none is pending). So an
   out-of-order mark, or a directive that lands after this fire's read, can never be jumped. This is
   stronger than "newest processed createdAt", which is the early-write bug in another shape.
3. The boundary is made **inclusive** (`>=`) and dedupe is by id. Ids are retained while their
   `createdAt >= watermark` (they are the only ones an inclusive read can return again).

## Registry reuse

`registry_reuse`: M1 none, M2 none, M3 none — repository-local bash instrumentation of the mission
loop, not a package-like AILANG capability. No `.ail` code; AILANG syntax gate not applicable.

## M1 — `mission_directives.sh`: identity, inclusive boundary, process-then-mark helper (~170 LOC)

- Emit `id=<GitHub comment node id>` in every directive header.
- `--since` comparison becomes `>=` (inclusive).
- New `--watermark-file <path>`: reads line 1 as `since` (absent → epoch), reads `processed <id> <createdAt>`
  lines; each re-read directive whose id is recorded is labelled `ALREADY PROCESSED — context only`.
  Loud stderr `BOUNDARY` warning when an allowlisted comment sits exactly at the watermark and is not
  recorded (the #981 signature) and a `LEGACY` note for a bare-timestamp file.
- New `--mark <id>` (requires `--watermark-file`): re-fetch, refuse an id that is not an allowlisted
  comment on the issue, record it, recompute the watermark per deviation 2, prune ids older than the new
  watermark, write atomically (tmp + `mv` in the same directory).
- Reading never writes the watermark. bash 3.2, BSD tools only; `jq` does all JSON work.

**Acceptance:** `/bin/bash -n` clean; shellcheck clean; default output unchanged except the `id=` token
and inclusive boundary; `--mark` of a non-allowlisted / unknown id exits 1 and leaves the file byte-identical.

## M2 — `scripts/test_mission_directives.sh` (~290 LOC), stub `gh` on PATH

Arms:
1. **Boundary** (#852 literal fixture): watermark = comment `createdAt` → the comment is returned.
2. **Kill-at-each-step**: a simulated fire (fetch → route → mark, per directive) killed after fetch, after
   routing directive 1, mid-mark (tmp written, no `mv`), after marking directive 1. For every kill point the
   next clean fire leaves each directive routed **exactly once** (ledger count) and nothing lost.
3. **Out-of-order / late arrival**: marking the newer of two directives first, or a directive created after
   the fire's read, must not move the watermark past the unprocessed one.
4. **Dedupe**: a re-read processed id is labelled and not routed again.
5. **Legacy file**: a bare-timestamp file is read, the boundary comment re-presented, and the first
   `--mark` upgrades the format with line 1 still a bare timestamp (6a-compatible).
6. **Multi-mission**: two watermark files and two allowlists through the one script, no cross-talk.
7. **Refusals**: `--mark` without `--watermark-file`, unknown id, non-allowlisted id.
8. **Control**: the OLD protocol (write newest `createdAt` before routing, exclusive read) driven through
   the same simulated fire loses the directive — proves the suite can see the failure.

**Acceptance:** all arms pass under `/bin/bash` 3.2; `make test-launchd-drivers` exit 0.

## M3 — Protocol text, mirror, mutation evidence, changelog (~60 LOC)

- `gate-0-preflight.md`: replace the "before routing" sentence with the process-then-mark ruling; step 6
  reads via `--watermark-file`, marks via `--mark <id>` after each directive's routing has landed; the
  hand-written watermark write is forbidden. Run `tools/launchd/sync-agents-skills.sh`.
- Mutation-test from a scratchpad copy (`MD_SCRIPT_UT=<copy>`): (a) early write on read, (b) watermark =
  newest marked, (c) exclusive `>`, (d) no `id=` emission. Each must fail a named arm.
- Changelog fragment `changelogs/unreleased/2026-10-26-gate0-watermark-crash-safety.md`.

**Acceptance:** `make test-changed`, `make check-changelog`, `make check-context-docs`,
`tools/launchd/sync-agents-skills.sh --check` all exit 0.

## Result (2026-10-26): M1 ✅ M2 ✅ M3 ✅

- Suite: 24/24 under `/bin/bash` 3.2; `make test-launchd-drivers`, `make test-changed`,
  `make check-changelog`, `make check-context-docs`, `sync-agents-skills.sh --check` all exit 0.
- Mutations (scratchpad copies via `MD_SCRIPT_UT`): early write on read → 2 arms fail;
  watermark = newest → 2; exclusive `>` → 9; no `id=` → 13. None survives.
- Live read-only check on #852 with a scratch legacy watermark `2026-08-30T08:11:12Z`: the lost
  D50/D49 comment `IC_kwDOP3Mz2c8AAAABReRotQ` is returned with BOUNDARY + LEGACY warnings.

## Rollout

The loops run Gate 0 from the pinned clones (`~/.ailang-driver-pin/*`), refreshed from `origin/dev`, so
nothing is live until merged. First fire after merge reads each mission's legacy bare-timestamp file
(boundary comment re-presented; the decision ledger prevents a re-ask) and its first `--mark` upgrades
the file. Close #981 after one observed fire per mission.
