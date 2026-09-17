# Which question, which instrument

Knowing an instrument exists is not the same as knowing which question it answers. Look
the question up here before writing a new script (CLAUDE.md §1).

| Question | Instrument | Notes |
|---|---|---|
| Did the runtime get slower or fatter since last month? | `make perf-sweep` | Diffs `.ailang/state/perf/<date>.json`; exit 2 on a regression on the same machine class |
| Is a workload over its p95 budget? | `make bench-workloads` + `benchmarks/budget_ledger.md` | The ledger is the 50/50 accounting (half of every saving tightens the target, half credits the pool). Re-baseline the ledger when the machine class changes; the seed rows are an M2 |
| Does this change copy a large value? | `testing.Benchmark(...).AllocedBytesPerOp()` traced-vs-untraced, in the package that owns the site | Deterministic on every platform. Example: `internal/effects/trace_render_alloc_test.go` (+8 KB with the fix, +8 MB without) |
| Is this peak live data or garbage? | `GODEBUG=gctrace=1 ailang run …` | Many cycles with low live = garbage (GOGC/limit territory); few cycles with high live = the program holds it (no limit will help) |
| What is the peak of one program? | `/usr/bin/time -l` (macOS, bytes) / `-v` (Linux, KB) with `GOGC=100 AILANG_NO_TRACE=1` | A ratio against the same program with the suspect feature off is robust to the ~50 MB floor; an absolute number on one run is not |
| Where does a program allocate? | `ailang run --memprofile mem.prof …` then `go tool pprof -top -sample_index=alloc_space <ailang binary> mem.prof` | The sweep prints the top 8 for `list_large` each run |
| Where does it spend CPU? | `ailang run --cpuprofile cpu.prof …` then `go tool pprof -top` | Evaluator hot paths: `evalCore`, `applyFunction`, `Environment.Get` |
| Which compile phase is slow? | `ailang check --debug-compile file.ail` | Parse / elaborate / typecheck / validate split |
| What memory controls does this host resolve? | `ailang doctor memory` | Soft limit and source, GOGC, cgroup files, the other caps |
| Is the evaluator's core loop regressing? | `go test ./internal/eval -bench 'BenchmarkEval_' -benchmem` | `Fib30` is 11 s/op; the sweep uses ListMapFilter, PatternMatch, StringPipeline |
| Is a builtin regressing? | `go test ./internal/builtins -bench . -benchmem` | `ListMap50K`, `ListFoldl50K`, `ParseElements_1K` are pinned by the sweep |
| Did tracing distort the number? | Re-run with `AILANG_NO_TRACE=1` | The default tier is `standard`; two 2026-09 reports attributed the tracer's cost to `concat` and to `cons` |

## What the sweep measures, and why each row exists

| Row | Shape | Why |
|---|---|---|
| `lat_*_p95_ms` | six canonical workloads | the release SLO (M-LAT-BUDGET) |
| `rss_floor_mb` | `println` hello | the process floor everything else sits on (~47 MB) |
| `rss_cons_6000_mb` | `n :: acc` recursion | LIVE data: every frame holds a copy (M-LIST-CONS-QUADRATIC). Stable because it is live |
| `rss_cons_deep_trace_ratio` | same, at `deep` | the tracer must not multiply live data (was 3.6×, now 1.08×) |
| `rss_effect_result_mb` | 20 × `readFileResult` of 20 MB via `foldlE` | FS double copy + effect-trace render; the `foldlE` step is what keeps it flat |
| `rss_debuglog_200k_mb` | 200k filtered `Debug.log` lines | must be dropped on arrival, not retained |
| `rss_string_build_20k_mb` | fold string accumulator + join | the quadratic idiom, 884 MB; a builder or fusion would flatten it |
| `eval_*_b_op`, `builtin_*_b_op` | Go benchmarks | copies show here deterministically; RSS of the same loops does not |
| `eval_mapinsert5k_b_op` | 5,000 `MapValue.Insert` | copy-on-write, 581 MB/op; its RSS swung +34% run to run, so it is B/op |

Rows that are garbage-dominated do not belong in the RSS section. If a new probe's RSS moves
more than its tolerance with no code change, move it to a Go benchmark.
