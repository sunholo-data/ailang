# Sprint M-WORLD-PNG-ALLOWANCE

**Design doc:** [m-world-release-png-allowance.md](m-world-release-png-allowance.md) (approved by Mark, attended, 2026-10-14; quorum r3 waived)
**Lands in:** `sunholo-data/ailang-world`, `dev`. This is a fleet-authored commit under D-WORLD-DRIVER-1. Nothing changes in the AILANG repo except this plan and the sprint JSON.
**Duration:** ~0.5 day attended · **Risk:** low · **Registry reuse:** none. These are bash/python CI scripts, not a package-like capability.

## Milestones

| ID | Milestone | LOC | Acceptance |
|---|---|---|---|
| M1 | Detector: extract gate → `scripts/tracked_binary_gate.sh`, whole-blob NUL scan | ~70 | Today's World tree prints `✓ 0 binary blobs among N tracked files` and exits 0. Non-vacuity FATAL on an empty repo. Fixtures 9, 10 and 12 reject. |
| M2 | PNG allowance + validator + caps; call site in `verify_go.sh` declares `--allow-png` | ~90 | Fixtures 1 and 13 pass; 2–8 each reject with their named reason. `grep -ci png scripts/verify_go.sh` ≥ 1. |
| M3 | `scripts/test_tracked_binary_gate.sh` (fixtures 1–13), run from `verify_go.sh`; mutation test | ~150 | Test green. Swapping the scanner back to `--numstat` makes fixtures 9, 10 and 12 fail the test. No `.github/` path in the diff. |
| M4 | Land + hand-off | ~15 | World CI green on the landed SHA. A `harness-resolved` reply in `mission-world` names the SHA and the contract. Ticket messages acked only after the reply. |

## Validation
`bash scripts/test_tracked_binary_gate.sh`; the gate run directly on the World checkout; World CI on the PR/push. The full `verify_go.sh` needs the pinned v0.41.0 `AILANG_BIN`, so CI is the authority for that run.
