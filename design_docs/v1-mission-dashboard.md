# Mission Dashboard — V1

*Snapshot, overwritten every iteration. History lives in `v1-mission.md` (STATUS) and `v1-mission-log.md`.*

**Updated:** 2026-09-05, after iteration 328.

## Latest release
`v0.35.1` (released mid-iteration today, by the release lane, not by this loop).

## Goal distance
**N = 13 design docs remaining before v1.0.0** (was 12, **+1 this iteration**).
The +1 is `m-compile-cache-unverified-artifacts`, a NEW clause-2 (soundness) doc: the compile
cache can silently execute a module that does not correspond to its source. Controller's
classification — flag it if you disagree, it is the only judgement call in the number.

## In flight / next picks
- **PARKED — `m-compile-cache-unverified-artifacts`** (this iteration's doc). Quorum BLOCKED after
  one revision and one re-quorum. Needs one human decision (see below). Doc is landed on `dev` under
  `design_docs/planned/v0_35_2/`, tagged PARKED, with both quorum artifacts.
- `m-gate-wiring-classifier-prefix-blind` — the previous queue head, unpicked this iteration because
  a confirmed user-facing correctness bug outranked it.
- `m-release-manager-skill-split` — debt with a named owner.
- `m-resolver-hook-disagree-on-docless-pick` — the two routing instruments disagree on a doc-less pick.

## Loop cadence + routing
Unattended, launchd-fired, pinned worktree `~/.ailang-driver-pin/v1`.
Iteration 328 routing: designer `codex:gpt-6-astra` (probe rc=0, **first real astra designer run**,
2 bounded runs: author + one protocol-mandated revision) · planner **not spawned** (routing call —
the doc is blocked, so there is no approved design to plan) · executor **not spawned** (same reason)
· evaluator `sonnet`, own worktree, **PASS 92/100**, and it corrected the controller twice.

## Parked on Mark
**ONE open decision — `D-55`.** Does the artifact-verification design need to bound *adversarial*
gob-decode work, or is the accidental-corruption threat model enough to unblock it? One reviewer has
now rejected twice on this single point; two others passed. Full ask with options and a default is
in the iteration-328 report on the bookkeeping issue.

## Quota posture
Metered **$0.32** of the $5 iteration ceiling (3 quorum reviewer rounds). All agent lanes were
subscription/quota buckets — codex (astra + sol) and sonnet. No GPU, no `rig.lock`.
