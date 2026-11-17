M7 is committed as `ab97f52b4`, not pushed. The done-gate is not fully green: `make test-changed` exits 2 from two failures outside this change.

- **Go gates:** gofmt empty, `go vet` rc 0, `go test -run 'Mission|Doctor|Ruling'` rc 0.
- **`make test-launchd-drivers`:** rc 0, all 58 suites passed.
- **Focused gates:** `check-file-sizes` rc 0, `check-changelog` rc 0, `test-changed` rc 2. It failed twice, on different tests. `TestPkgConfinementMatrix` passes alone and looks like a parallel-load flake. `TestConfinedGit_HooksPathDisablesPlantedHook` also fails at the pre-change commit `2841b61a6`, because this machine sets a command-line `core.hooksPath`.
- **Dry-runs (world, `MISSION_DRY_RUN=1`):** healthy rc 0 `lanes=ok`; degraded rc 0 `lanes=DEGRADED(3)`.
- **Doctor layer:** healthy rc 1 on both installed and scratch binaries. Degraded copy gave exactly one `rulings-malformed` and zero contradictions.
- **Live run, rc 1 on both binaries (same as baseline).** Counts: 1 `rulings-ledger-absent`, 0 `ruling-contradiction`. Predicted 2 `ruling-link-missing`, 0 contradiction. The default registry resolves the ledger to the `dev` checkout, which has no `fleet-rulings.md` until this branch merges. I corrected the doc's success criterion to the measurement and added V34 with the full output.
- **Supplementary:** with this worktree's registry (ledger present), counts match the prediction: 2 `ruling-link-missing` (world, stapledon), 0 contradictions.

Deviations:
- The world loop was mid-iteration (pid 55920), so the dry-runs yielded at the overlap guard. I used the synthetic-HOME recipe from `fleet-mission.md`, not an idle sibling. The real `~/.ailang/state` was not written.
- I set the design doc to "Implemented" even though `test-changed` is red. That is an explicit call to make: the status line states the red gate, and you can revert it if you want the doc left in `planned/` until `test-changed` is green.
- Left as-is: a historical comment in `internal/mission/rulings_parse_test.go:9` (outside M7's file list), and `design_docs/fleet-mission-evidence/` references, which quote the old artifact's SHA.
