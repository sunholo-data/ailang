# Checked list consumption and construction benchmarks

From the repository root: `bench/vm_tail_walk/run.sh ./bin/ailang 3`.
Requires Bash, Python 3 and an AILANG binary; binary paths are resolved before
changing directories. The second argument is a positive repetition count.

Walk and foldl consume the same `range(0, n)` and count values greater than zero;
the exact oracle is `max(n-1, 0)`. Build reverses that range with repeated `::`
and prints length `n`: it is the still-quadratic D-19 construction contrast.
All modules declare `main(n: int) -> () ! {IO}`; traversal helpers are pure.
Contracts and inline tests are omitted because the runner checks the closed-form
oracle at 0/1/2 and on every timed repetition, on both engines.

The runner checks modules, exits, exact stdout and engine equality before
reporting minima. It also runs strict-alone boundary smokes. Wall seconds use
Python's monotonic timer and include process startup, compilation and input
construction. Walk/foldl sizes are 40k/80k/160k; build is bounded to 10k/20k/40k.
There is no CI timing threshold. Existing `internal/vm/list_tail_share_test.go`
identity/capacity/isolation tests mechanically protect Phase 1 sharing.

Historical pre-fix V6/V28 numbers in the design doc used different binaries and
machines. Compare current engine timings within one run; do not interpret
cross-machine differences as Phase 1 speedups. Execution results and binary
provenance are recorded in the residual sprint report.
