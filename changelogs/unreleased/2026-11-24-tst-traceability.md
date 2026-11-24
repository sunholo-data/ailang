### Changed — design docs: Testing Strategy is a traceable `TST-<TAG>-NN` table

Adopted from the blueprint-spec format (only this piece). A design doc's Testing Strategy is now a
table of `TST-<TAG>-NN` rows (level + Given/When/Then, surprising rules stated outright). The test
that proves a row carries its ID in a comment (`// TST-<TAG>-NN` above the Go `func Test…`), and
sprint acceptance criteria name the IDs. Updated: the design-doc-creator template and structure
guide, the sprint-plan template and planner checklist, and the executor's failing-tests-first step.

### Added — sprint-evaluator runs the tests a criterion names

`check_acceptance_criteria.sh` treats `TST-` IDs as the strongest anchor. For each ID it finds the
carrying test outside the doc/state trees (including untracked files) and runs the enclosing Go
`Test*` with `go test -run`. A red test or an ID no test carries scores `missing`. Non-Go carriers,
or `ACCEPTANCE_RUN_TESTS=0`, fall back to the suite result. It also reports design-doc rows no
test carries yet (`tst_uncovered`, not scored). Criteria without IDs score exactly as before
(verified identical on two live sprints). Self-test: `test_check_acceptance_criteria.sh` in
`make test-launchd-drivers`; each of three seeded mutants turns it red.
