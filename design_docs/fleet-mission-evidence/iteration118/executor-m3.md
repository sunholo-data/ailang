M3 is committed as `4cb0b32ef` on `fleet/i118-rulings`. Not pushed.

**Files:** `internal/mission/rulings.go` (181 lines, under the 300 target), `internal/mission/rulings_test.go` (244 lines).

**Gates:**
- `gofmt -l internal/mission`: no output, rc 0.
- `go vet ./internal/mission/`: rc 0.
- `golangci-lint run ./internal/mission/...`: "0 issues.", rc 0.
- `go test ./internal/mission/ -run 'Ruling' -count=1`: ok, rc 0. All 9 M3 tests and the M1/M2 tests pass.
- `go test ./internal/mission/ -count=1`: ok (27.9s), rc 0.

**Mutants (scratchpad copy, restored byte-identical each time; `git status` shows only the two new files, no leftover mutation):**
1. Link-missing as Drift: `TestRulings_EveryFindingIsNote` FAIL: `[DRIFT] a ruling-link-missing … is not a Note`.
2. Malformed ledger falls through: `TestRulings_MalformedLedgerRunsNothing` FAIL by panic (nil pointer dereference). It is killed by the panic, not the assertion, because the parser returns a nil ledger.
3. 20-hit cap removed: `TestRulings_HitCapTwentyThenSummary` FAIL: `23 hits: want 20 listed + 1 summary = 21, got 23`.
4. Iterate `ledger.Forbids`: `TestRulings_OpenAndSupersededAreQuiet` FAIL: `open_ruling_is_quiet: contradictions = 1, want 0`.
5. `charter=false` for charters scope: `TestRulings_CharterLedgerBlockExcluded` FAIL: literal inside the ledger block is reported.

All five mutants were killed.

**Deviations:**
- Commit subject is the one you specified, `feat(mission): CheckRulings — report-only ruling notes (…M3)`. The plan's text reads "report-only fleet ruling notes"; I used your wording.
- `ledger()` and `linkedCharter` test helpers live in `rulings_test.go`, not in the corpus test file. The plan allowed this.
- Summary finding for overflow hits uses Mission `rulings`, since a charters-scope detector can span several owners.
- The skills walk runs once per check, so a walk error produces one `ruling-file-unreadable`, not one per skills detector.
