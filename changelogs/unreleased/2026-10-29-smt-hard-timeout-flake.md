### Fixed — `TestSolve_HardTimeout_FakeSolverIgnoringT` flake under suite load (#602)

- The fake solver in `internal/smt/solver_timeout_test.go` is now run once before the timed attempt. The test races Solve's 3s hard timeout against the fake solver writing its child PID. Most of the fake solver's startup time was macOS assessing a freshly written executable, which a full `go test ./...` floods with new test binaries.
- During `go test -count=1 ./internal/...` (60 trials per arm), spawn-to-pidfile took p95 2.6s and max 4.46s for a fresh script, against max 11ms once warmed.
- The earlier retry-on-inconclusive fix (#608) assumed each attempt was independent. In practice load delays every attempt together, so all three lost together (reproduced 2026-10-29, 3.0s each).
- With the fix, 12/12 runs passed with 0 inconclusive attempts at load 15–18.
