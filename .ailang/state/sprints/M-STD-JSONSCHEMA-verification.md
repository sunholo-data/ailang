# M-STD-JSONSCHEMA verification

Starting commit: `abd4dfd821e919654ddaa0fafd36dd27197126fb`.
Work branch: `coordinator/task-b48daa42`.

| Check | Result |
| --- | --- |
| Sprint JSON validation | PASS, 10 populated milestones |
| Schema-focused Go suite, `-count=20` | PASS, 6.478s final run |
| VM pure-builtin coverage and schema/list engine parity | PASS |
| Changed Go packages | PASS: builtins (20.455s), bytecode (0.013s), VM (8.809s) |
| `make test-stdlib-ail` | PASS: 7 suites and 4 run fixtures |
| `make verify-stdlib` | PASS: all 50 interfaces; intentionally accepted list/jsonschema changes |
| `make verify-examples` | PASS: 248 modules, 0 drift, 0 missing |
| New example | PASS: check, 3 inline tests, expected runnable stdout |
| `make fmt-check` | PASS |
| `make check-file-sizes` | PASS |
| `make check-boundaries` | PASS |
| `make check-changelog` | PASS |
| `go vet ./internal/builtins ./internal/bytecode ./internal/vm` | PASS |
| `make lint` | Unavailable: golangci-lint is not installed |
| `make test-changed` core portion | Completed: two baseline effects failures; new builtin golden drift fixed and focused recheck passes. All other core tests pass |

## Baseline failure evidence

`internal/effects/` has no changes in this sprint. A detached worktree at the
starting commit reproduced BOTH failing tests with this command:

```
go test ./internal/effects -run '^(TestRuntimeNestedStopUsesRemainingDeadline|TestProcessExec_TimeoutKillsGrandchildren)$' -count=1
```

```
--- FAIL: TestRuntimeNestedStopUsesRemainingDeadline (2.64s)
    process_runtime_lifetime_test.go:323: nested fixture 4329 survived bounded stop
--- FAIL: TestProcessExec_TimeoutKillsGrandchildren (2.32s)
    process_timeout_group_test.go:57: grandchild 4336 is still alive after exec returned Timeout
FAIL github.com/sunholo-data/ailang/internal/effects 4.967s
```

The baseline worktree was removed after checking its clean state. Tests were
not skipped and CGO/test semantics were not changed to hide these failures.

## Performance evidence

Linux amd64 AMD EPYC 7B13, 460 schema nodes, 1,000 document properties:

```
BenchmarkJSONSchemaCensusScale-3  100  24929864 ns/op  1766962 B/op  23816 allocs/op
BenchmarkJSONSchemaPublicAPI-3    98  38650266 ns/op  2550336 B/op  33795 allocs/op
```

Both engine-only and actual strict public API meet the 50 ms target under the
measured concurrent load. The initial engine-only run measured 8.848391 ms/op.
The exact-census fixture is synthetic, not the external client schema.

The core pipeline golden guard initially detected the three intentional new builtin
signatures. Regenerated with `UPDATE_GOLDEN=1 go test ./internal/pipeline -run
TestBuiltinTypes_GoldenSnapshot`, reviewed the three-line diff, then reran the
focused test without update mode: PASS. The initial full pipeline run otherwise
passed all tests; it was not repeated after this data-only golden update.
