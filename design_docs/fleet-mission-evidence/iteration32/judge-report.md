# Judge report: M-QUORUM-ZERO-SIGNAL (iteration 32), commit 736d58142

Score: 95/100. Verdict: PASS. Blocking issues: none.

## Acceptance criteria (M1)
| AC | Result | Evidence |
|---|---|---|
| Matrix passes; new controller-only tests red pre-fix; pre/post suite green pair | PASS | Matrix at quorum_test.go TestSynthesize_ExternalSignalMatrix (11 rows). MU1 turns it red, so the tests discriminate. The report's pre-fix RED output is plausible and consistent with my MU1 results. |
| Seated replacement works, all absences named | PASS | seating_test.go: poolRunner.calls must contain b and z; absent list must equal [a b z] in order. |
| Rejects keep vetoes; partial-present pass allowed | PASS | Matrix rows "one pass/reject", "one reject". |
| Existing serialization/accounting/escalation tests pass | PASS | Full suite green. |
| go test / -race / vet / gofmt | PASS | See commands below. |
| Contract comments and report describe the change | PASS | quorum.go RunQuorum and synthesize comments; report "Intentional behavior change". |
| Absent-pair / controller-pass accounting | PASS | costedAbsent has CostUSD 0.02, TokensIn 300, TokensOut 40. Totals are asserted against sums over absent outcomes, in input order. Controller entry is asserted. MU6 (drop absent cost) is killed. |
| Two-absent / controller-reject: both objections in order | PASS | wantObjections = [controllerVeto, zeroSignalObjection], compared with reflect.DeepEqual (ordered). |
| AbsentReviewers non-nil empty for all-present | PASS | Existing TestSynthesis_AbsentReviewersMarshalsAsEmptySlice is in the green suite. quorum.go still initialises the slice to []AbsentNote{}. |
| Seated regression | PASS | Primary a, reserve b, benched z, all absent, controller pass. Asserts blocked, the exact single objection, calls contain b and z, and absences [a b z]. |
| Tier-2 regression + positive control | PASS | tier2_test.go calls SynthesizeWithTier2 directly (not via escalation). Asserts blocked plus the zero-signal objection. Positive control (one pass + one absent + Tier-2 pass) proceeds. |
| Mutation evidence | PASS | MU1 reproduced independently; see table. |
| Scope | PASS | Changed files: .ailang/state/sprints/*.json, changelog fragment, iteration32 report (under design_docs), quorum.go, 3 test files. All within the allowed paths. |

## Commands (worktree at 736d58142, clean)
- `go test ./internal/mission/quorum -count=1`: `ok ... 0.494s`
- `go test -race ./internal/mission/quorum -count=1`: `ok ... 2.301s`
- `go vet ./internal/mission/quorum`: no output, rc=0
- `gofmt -l internal/mission/quorum`: no output
- `go build ./cmd/ailang/...`: rc=0

## Mutation drills
Each mutation was applied to a /tmp copy-backed file. After each, I restored from the backup and verified the sha256 equals ea9bb295...c66.

| Mutant | RED tests | Killed |
|---|---|---|
| MU1: `externalPresentCount++` as first stmt in `if controller != nil {` | Matrix, SeatedAllAbsentControllerPass, Tier2PassCannotRescue | yes |
| MU2: guard disabled (`if false && ...`) | existing AllAbsentRefusesToProceed, NothingAnswersBlocks, plus the 3 new tests | yes |
| MU3: `return s` after controller-reject append | Matrix only (the dual-objection rows) | yes |
| MU4: guard uses `len(outcomes) == 0` | AllAbsentRefuses, NothingAnswers, 3 new tests | yes |
| MU5: count every outcome including absent | same 5 as MU4 | yes |
| MU6: absent outcomes' cost dropped from totals | Matrix (accounting) | yes |

No survivors.

## Report fidelity
The report's claims match what I observed: the production diff, the test shape, the green runs, and MU1's failing set (Matrix, Seated, Tier2). I did not re-run the pre-fix RED run. I verified the equivalent via MU1, which restores the old behavior.

## Non-blocking follow-ups
- The report's pre-fix RED output is hand-condensed. The test names and failure messages differ slightly from raw `go test -v` output, and the report notes that the MU1 exit codes were not captured. It is a minor evidence-quality issue.
- The seated test would not catch an early-return mutant by itself (MU3 was caught only by the matrix). That is acceptable; the plan's early-return concern is covered.
- The wider repo gates (make test/ci) were not run; this is stated in the report.
