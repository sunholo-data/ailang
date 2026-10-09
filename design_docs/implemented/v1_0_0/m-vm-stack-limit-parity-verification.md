# M-VM-STACK-LIMIT-PARITY implementation verification

Refs #1576 — https://github.com/sunholo-data/ailang/issues/1576

## Scope and mechanism

The VM's 1,000-frame default caused overflow at evaluator-legal depths. Non-strict
entrypoint fallback restarted evaluation after stdin consumption or committed output.
The fix raises the VM default to 10,000 and wires positive `MaxRecursionDepth` values
at the production constructor. Zero and negative overrides retain the default.
Above-limit non-strict replay (parent design Lane B4) remains unresolved; this change
must not auto-close #1576 or claim safe fallback.

## Red-to-green evidence

The baseline unit test reported `default MaxStack = 1000, want 10000`.
The strict pure depth-200 fixture exited 0 with result 200 despite flag 50;
depth 9,000 and depth 11,001 with flag 12,000 incorrectly overflowed.
After wiring, the small cap fails with `stack overflow`, both legal cases succeed,
and default/zero/negative caps reject depth 11,001 on both backends.
The evaluator reports `RT_REC_003`; the strict VM reports `stack overflow`.

The service baseline printed `HELLO\nASK\n`, exit 0, with the verbose fallback warning.
After the fix, interpreter and VM print exactly
`HELLO\nASK\nHANDLED len=1170\nHANDLED-AFTER\n`, exit 0, without fallback.
The standalone baseline printed `START\nSTART\nDONE len=5000\n`; both fixed backends
print `START\nDONE len=5000\n` without fallback.
All three temporary fixtures are checked using `ailang check --relax-modules` in the
CLI tests. Pure fixtures use strict mode; IO fixtures use the existing evaluator bridge.

AILANG prompt version loaded: v0.16.6 (before fixture construction).
No showcase module was added: contracts and inline tests are skipped because these
are temporary CLI regressions asserting backend/depth/IO behavior, rather than public
language examples. Effects are explicit: `rep`/`sum` are pure; service and printing
functions declare `! {IO}`. Windows stdout line endings are normalized for IO assertions;
no new path assertions, external test binaries, or goldens were introduced.

## Full corpus reconciliation

Both legs ran `go run ./scripts/verify_bytecode_parity.go --json` after `make build`,
using `bin/ailang`, not the preinstalled CLI. Both used Go 1.26.9 and CGO disabled;
CGO was subsequently enabled to discharge existing SQLite-dependent test gates.
Source base: `3d4f49720cccdfc457ab9f8b4755a87bbf3cf566`.

| Leg | Binary SHA-256 | Elapsed | Files | MATCH | EVAL_SKIP | DIVERGE | NON_DET |
|---|---|---:|---:|---:|---:|---:|---:|
| Before | `040cb02f94105ab2f93ec0459c96ca983158cf3bd045f3237aa283011a59a43d` | 49.40 s | 200 | 178 | 17 | 3 | 2 |
| After | `a4626ab699b6ec5c49d471b0ac6204ab804fa51cd0de0515805a5dc9ac2eff79` | 77.79 s | 200 | 178 | 17 | 3 | 2 |

Compared every row by filename, status, evaluator exit, and VM exit: identical file
sets and **zero changed statuses or exits**. The three unchanged DIVERGE rows are
`examples/runnable/array_basic.ail`, `examples/runnable/process_demo.ail`, and
`examples/runnable/xml_walk_perf.ail`. Only `xml_walk_perf.ail` changed stdout between
legs, from its printed timing measurements; both still report count 20,000 and count
agreement. Before timing pairs (classic/fold): evaluator 667/618 ms, VM 773/699 ms;
after: evaluator 1479/1615 ms, VM 1407/2190 ms. This is an existing timing divergence.

Raw reports and command logs are banked outside the runnable corpus at
`/tmp/ailang-tools/parity-before.json` and `/tmp/ailang-tools/parity-after.json` during
execution. Harness MATCH does not prove VM-native execution, because fallback can
produce a match. The strict and verbose regression tests provide that separate evidence.

## Validation gates

- `go test ./internal/vm/... ./internal/runner/... ./internal/testing/...`: passed
  both before and after enabling CGO. CGO package times: VM 0.311 s, runner 0.403 s,
  named-test engine 10.126 s. Existing small-cap overflow, frame reuse, and TCO tests pass.
- `go test ./cmd/ailang/... -run 'StackLimit|Bytecode|TailCall|Fallback|Stdin' -count=1 -v`:
  passed with CGO enabled, 38.015 s. Pure-depth regressions, both IO regressions,
  existing tail-call parity, fallback warnings, and named-test flags pass.
- `make test-core`: passed with CGO enabled. The initial CGO-disabled run failed
  only existing SQLite tests; installing a C compiler fixed the environment.
- `make verify-examples -o build`: passed with the already-fresh binary, including
  manifest validation (212 modules checked, 0 drift). The option preserves corpus
  binary provenance by avoiding an unnecessary rebuild.
- `make fmt`, `make fmt-check`, `make check-file-sizes`, and `git diff --check`: passed.
  `vm.go` remains 791 lines (0 net growth); `vm_test.go` is untouched.
- `make lint`: passed with 0 issues using `GOMEMLIMIT=768MiB GOGC=30 GOMAXPROCS=2`.
  Its first run was OOM-killed (exit 137); no source finding was suppressed.
- Final `go test ./cmd/ailang/... -run StackLimit -count=1 -v`: passed, 15.262 s,
  including zero/negative under-limit and over-limit cases. An overlapping link was
  OOM-killed; the successful rerun followed lint with `GOMEMLIMIT=192MiB GOGC=30
  GOMAXPROCS=1`. Final CGO-enabled `golangci-lint run ./cmd/ailang/...` also passed
  with 0 issues. Service test: 0.59 s; print-once test: 1.99 s.

The full `make test` remains a CI gate as explicitly required by the approved plan;
no local full-suite result is claimed. The executor does not push or merge.

## Coordinator PR body

Refs #1576

Raise the bytecode VM's default frame cap from 1,000 to 10,000 and apply positive
`--max-recursion-depth` overrides in the production runner. This lets the reported
stdin service handler finish without overflowing into evaluator replay. Regression
tests pin complete service output, exactly-once printing, explicit depth limits, and
strict/interpreter behavior above and below the default ceiling.

The full 200-file parity corpus has no status or exit changes. Above-limit non-strict
fallback can still replay effects; the parent design's B4 policy remains separate.
See the validation gates in this report for commands and results.

## Size and handoff

Planned: 220 LOC including documentation. Actual: 3 net production lines,
146 new test lines, plus sprint/design/limitations/verification documentation.
Execution was sequential; most elapsed time went to fresh-toolchain compilation,
CGO setup, and memory-constrained gates. No production scope expansion was needed.
Implementation is ready for independent sprint-evaluator review and coordinator PR/CI.
