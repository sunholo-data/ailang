Implemented fleet iteration 140 in M1 → M2 order. `rotate-log --workdir <dir>` now targets the specified checkout for rotation and index regeneration. A missing source returns its absolute path before any write. All eight test-table cases are covered, and existing tests are unchanged.

The changes are uncommitted. Verified the cumulative snapshots byte for byte in `.snap/M1/` and `.snap/M2/`. No deviations from the approved plan.

### Files changed

- [mission_rotate.go](/Users/voightkampff/.ailang-driver-pin/fleet-i140-wt/cmd/ailang/mission_rotate.go)
- [mission_cmd.go](/Users/voightkampff/.ailang-driver-pin/fleet-i140-wt/cmd/ailang/mission_cmd.go) — help text only
- [mission_rotate_test.go](/Users/voightkampff/.ailang-driver-pin/fleet-i140-wt/cmd/ailang/mission_rotate_test.go)
- [gate-4-record.md](/Users/voightkampff/.ailang-driver-pin/fleet-i140-wt/.claude/skills/mission-control/resources/gate-4-record.md)
- [2026-11-25-rotate-log-workdir.md](/Users/voightkampff/.ailang-driver-pin/fleet-i140-wt/changelogs/unreleased/2026-11-25-rotate-log-workdir.md)

### Gates

| Command | Result |
|---|---|
| `go build ./cmd/ailang` | Passed. Removed the generated `ailang` binary. |
| `go test ./cmd/ailang -run Rotate -count=1` | Passed |
| `go test ./internal/mission -count=1` | **UNINFORMATIVE UNDER SANDBOX** — `TestObserveAnthropicQuota_OverRationBlocksByDefault` hit `listen tcp6 [::1]:0: bind: operation not permitted`. |
| `make check-context-docs` | Passed |

### Mutation checks

| Test-table row | Mutation | Test that failed | After restoring |
|---|---|---|---|
| Shared-repo mission | Let `m.Root()` overwrite the explicit directory | `TestMissionRotateWorkdirOverridesExternalAndShared/shared` | Passed |
| Index-only | Dispatch to `RotateLog` | `TestMissionRotateWorkdirIndexOnly` | Passed |
| Missing override source | Skip the explicit existence check | `TestMissionRotateWorkdirMissingSource` | Passed |
| Relative `--workdir` | Drop `filepath.Abs` | `TestMissionRotateWorkdirRelative` | Passed |