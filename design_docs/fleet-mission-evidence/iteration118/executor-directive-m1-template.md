MISSION-ROLE: executor

You are the EXECUTOR for fleet mission iteration 118 (AILANG repo; unattended, no human present).
Work ONLY in `/Users/voightkampff/.ailang-driver-pin/.fleet-i118-rulings` (branch `fleet/i118-rulings-ledger`).

Your job: implement **milestone M1 only** of the sprint plan
`design_docs/planned/m-fleet-rulings-ledger-sprint-plan.md`. Read, in this order: the plan's
"Conventions for every milestone" section, then the whole "### M1:" section, then (for context only) the
design doc `design_docs/planned/m-fleet-rulings-ledger.md` §1 and §4. Implement exactly the constants,
types, function signatures and parsing rules the plan specifies, with the tests it names.

Definition of done for M1 (all must hold; report each with the command and its output):
1. `gofmt -l internal/mission` prints nothing; `go vet ./internal/mission/` rc 0;
   `golangci-lint run ./internal/mission/...` rc 0 (if golangci-lint is not installed, say so — do not skip silently).
2. `go test ./internal/mission/ -run 'Rulings' -count=1` passes, and `go test ./internal/mission/ -count=1` passes.
3. Every mutation the plan lists for M1, run by the plan's scratchpad-copy procedure (never `git checkout <file>`):
   the named test FAILS with the mutant applied and PASSES after restoring. Record each mutant + its FAIL line.
   Afterwards `git diff` must show no leftover mutation.
4. One commit on the branch: message `feat(mission): rulings ledger parser (M-FLEET-RULINGS-LEDGER M1)`, body
   listing the mutants and their FAIL lines, ending with the line
   `Co-Authored-By: Claude Haiku 5.5 <noreply@anthropic.com>`. Do not push. Do not touch files outside
   `internal/mission/rulings_parse.go` and `internal/mission/rulings_parse_test.go`.

If the plan is ambiguous or wrong, do the most conservative thing consistent with the design doc and SAY SO in
your final report (a deviation is fine if stated and justified; a silent one is not).
Rules: run everything in the foreground; never run `make test` or anything touching the rig GPU lock; never
`cp` over `~/go/bin/ailang`. Nothing will re-invoke you after you stop, so do not end your turn announcing a
wait. Final message (≤ 15 lines): files and line counts, the commit SHA, gate outputs, mutants killed/survived,
deviations.
