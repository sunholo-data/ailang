M4 is committed as `2f0becbd3` on `fleet/i118-rulings-ledger`. Not pushed.

**Files**
- `internal/mission/doctor.go`: +1 line at :295, `rep.Findings = append(CheckRulings(reg.Missions), rep.Findings...)`, between the loop close and the row sort.
- `internal/mission/rulings_test.go`: +223 lines, six tests (the plan's six: LinkMissing, PerMissionScope, RootDerivationEndToEnd, CharterLocationEndToEnd, DoctorRulingFindingsPrecede, DoctorRulingCheckRunsForLoadedMission).
- `design_docs/planned/m-fleet-rulings-ledger.md`: §3 call-site paragraph now states the prepend form and the P8b reason.

**Gates**
- `gofmt -l internal/mission`: no output, rc 0.
- `go vet ./internal/mission/`: rc 0.
- `golangci-lint run ./internal/mission/...`: `0 issues.`, rc 0.
- `go test ./internal/mission/ -run 'Ruling' -count=1`: ok.
- `go test ./internal/mission/ -count=1`: ok (27.8s).
- `-run 'TestRuling|TestDoctor_Ruling' -v` PASS count: 40 at M3, 46 now (+6).
- `git diff HEAD~1 -- internal/mission/doctor.go | grep -c '^+[^+]'`: 1.
- `doctor_test.go` untouched.

**Mutants (scratchpad copy, restored, stat matched pre-mutation each time)**
1. Append instead of prepend → FAIL `TestDoctor_RulingFindingsPrecedeMissionFindings` (rulings_test.go:432).
2. Wiring line deleted → FAIL `TestDoctor_RulingCheckRunsForLoadedMission` (rulings_test.go:462).
3. Charters scope via `Load(root/missions)` → FAIL `TestRulings_PerMissionScope` (rulings_test.go:317).
4. `charterPath` branches swapped → FAIL `TestRulings_CharterLocationEndToEnd` (rulings_test.go:375).

All four killed, none survived. The restored tree passes the full gate set above.

**Deviations**
- The plan asked me to replace the quoted `append(rep.Findings, CheckRulings…)` in design §3. That form was not in the doc at HEAD; line 154 already had the prepend form. I only added the plan's sentence, so the edit is additive.
- The plan's "same Ruling count +6" is 40→46, which matches.
- I used `one-tab`-indented edits by hand for the mutants; no `sed -i` was applied to the tree (a BSD `sed -i` attempt was a no-op and I confirmed the tree was unchanged).
