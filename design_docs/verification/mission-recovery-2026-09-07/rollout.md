# Controlled rollout — 2026-09-07

## Candidate and upstream integration

Merged origin/dev at 1fd1c7a70 into the isolated recovery branch without conflicts. This includes
V1's #1074 event-handler extraction and the current mission records. Re-ran full make test,
lint, architecture and file-size checks, targeted race tests and vet: all passed, including
the formerly failing file-size check. Five receipt tests now explicitly skip Windows directory
sync, matching the existing fail-closed platform boundary; independent reviewer PASS.

Candidate code: dad6e8efb0d939a48f639b5f39907994a5f11792.

## Actual World driver preflight

Executed the committed candidate through the real pin helper and real mission-control driver,
using a new isolated pin worktree, World profile, existing World work repository, dry-run mode,
5-second model-probe setting and 180-second outer process deadline. No saved configuration edit.
The driver performed its ordinary small model probes; no full mission iteration ran.

Result: exit 0, PIN_STATUS=pinned, all role lanes usable, World workdir retained. The new pin
checkout's full SHA matches the candidate; World's origin is sunholo-data/ailang-world and its
charter exists. See [machine evidence](world-preflight.json). A preflight is not a successful
live mission iteration and does not justify calling the whole fleet migrated.

## Authorized publication and World deployment

The user explicitly authorized deployment after the initial publication approval block.
Published branch `sprint/mission-runtime-recovery` to `sunholo-data/ailang`, verified its
remote SHA, and opened draft PR https://github.com/sunholo-data/ailang/pull/1082.
The deployed immutable revision is `f26d646661c4e9ef7c7840d6a2ad355afd402945`
(the tested code above plus its rollout evidence).

At a verified idle World boundary, backed up the saved World environment and enabled pinning
at that revision in a fresh dedicated pin worktree. The exact saved configuration passed the
real driver preflight: all role lanes usable, pinned SHA correct, World repository retained.
See [deployment evidence](world-deployment.json) for configuration hashes, rollback backup
and actual driver output. Restore that backup to roll back this deployment.

The installed binary, other missions, cadence, weekly pointers and decision ledgers were not
changed. No inbox messages were acknowledged. This deploys the pin repair to World; the
new durable role-run protocol is not automatically adopted by the existing loop.

## Live canary and CI status

The immediate `launchctl kickstart gui/501/dev.ailang.mission-world` client exceeded its
15-second deadline. Subsequent inspection showed no World PID, last exit 0, and
`state = spawn scheduled`; there was no new full-iteration start in the log. The existing
four-hour throttle may explain the delay; this is not yet a successful full canary run.
Do not repeatedly kickstart or alter fleet cadence merely to accelerate verification.

PR checks inspected after publication: lint, CodeQL and vulnerability gate passed; build,
Go tests and docs were pending. The launchd job failed seven notification assertions
(pin notice, lane notice and source-drift notice), matching the previously recorded D-60
baseline failure set. These failures remain visible and are not waived by this deployment.

Next acceptance evidence: observe the next World fire's driver SHA, actual repository,
charter and stage outcome, then its completed iteration. Restore the saved backup on
wrong-repository or startup regression. Only after live proof consider default-ref adoption.
Artifact acceptance, durable role-state adoption and quota reservations remain separate work.

## Blocked decisions

See [the attended decision review](decision-review.md) for 12 proposed rulings. None is
recorded as approved until the user answers. Record accepted rulings in the charter each
loop actually reads, retaining pending-record decisions D-60 and D-WORLD-35.
