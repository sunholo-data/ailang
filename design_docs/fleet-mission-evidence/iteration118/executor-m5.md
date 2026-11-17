M5 is committed as `a3881c2e7` on `fleet/i118-rulings-ledger`, not pushed. The worktree is clean. Both mutants were killed and restored.

**Files:** `cmd/ailang/mission_cmd.go` (+14/−5; new `printDoctorReport`, `io` import, help line), new `cmd/ailang/mission_doctor_print_test.go` (+53).

**Gates:**
- `gofmt -l internal/mission cmd/ailang`: no output, rc 0.
- `go vet ./internal/mission/` and `./cmd/ailang/`: rc 0.
- `golangci-lint run ./internal/mission/...`: "0 issues.", rc 0.
- `go test ./internal/mission/ -run Ruling -count=1`: ok.
- `go test ./internal/mission/ -count=1`: ok.
- `go test ./cmd/ailang/ -run 'Mission|Doctor|Ruling|CLIReference' -count=1`: ok.
- `TestMissionDoctorPrint_*` (3 tests): all PASS.
- `TestCLIReferenceMatchesTable` and `TestCLIReferenceCoversEveryRoute`: PASS.
- `tools/launchd/test_mission_lane_check.sh`: 42 passed, 0 failed, rc 0.

**Mutants:**
- Restore `if len(rep.Findings) == 0`: FAIL `TestMissionDoctorPrint_NotesThenSummary`.
- Drop the condition: FAIL `TestMissionDoctorPrint_DriftHasNoSummary` (`mission_doctor_print_test.go:48`, last line was `1 mission(s): no drift`).

**Deviations:**
- The commit subject follows your wording, "prints the no-drift summary", not the plan's "prints its summary".
- I did not run `make check-cli-docs`; the `TestCLIReference*` tests cover the help-line change.
