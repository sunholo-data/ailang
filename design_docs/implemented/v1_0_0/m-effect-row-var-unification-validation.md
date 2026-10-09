# M-EFFECT-ROW-VAR-UNIFICATION validation

Refs #616. Execution completed 2026-10-09 on `coordinator/task-c86840f2`.
Independent sprint-evaluator review is the next stage; no push or merge performed.

## Implementation and provenance

App constraints preserve the instantiated explicitly open callee row. Existing
implicit/concrete no-join behavior is retained. One per-App publication owns the
pre-solve latent mask and post-solve call/callback rows, including scoped ownership
and immutable snapshots. Publication is sealed at declaration boundaries so later
solver-name reuse cannot alter earlier calls. Source-owned row variables remain
owned; return-only quantified rows instantiate minimally at pure call sites.
Type-only scope bindings and the implicit-contract origin flag preserve existing
inferred/curried lambda behavior and #1091 closure-before-generalization.

Validation charges callee calls from that publication and argument evaluation
separately. Tail-preserving union checks distinct-tail conflicts; subsumption
rejects unmatched tails without treating a generic tail as permission for IO.
No new callback traversal for nested list/tuple/ADT arguments was introduced.

Base: actual merge-base with origin/dev
`acaf73390eac8d8d8bb7ec71d7a3ea12bc74a886`, detached matching source/stdlib at
`/workspace/effect-base-source`. Fixed implementation staged tree:
`75b6edea3c5598512279423d14c7d931a036fb16` (before bookkeeping/report artifacts).
The JSON retains per-source SHA256 and both binary hashes. Embedded fixed version
metadata reports base-dirty; it does not identify the implementation commit.
Go toolchain: `go1.26.9 linux/amd64`.

Both CLI builds used `CGO_ENABLED=0 GOFLAGS=-p=1 GOMAXPROCS=2 GOGC=40
GOMEMLIMIT=600MiB go build -o <binary> ./cmd/ailang` in matching roots.
Executables: `/workspace/ailang-effect-base`, `/workspace/ailang-effect-fixed`.

## Complete corpus comparison

`python3 tools/audit-effect-rows.py ROOT BINARY OUTPUT_JSON` inventories
`rg --files --hidden --no-ignore examples std -g '*.ail'`, then executes each
file with `AILANG_NO_CACHE=1`, matching `AILANG_STDLIB_PATH=ROOT/std`, cwd ROOT,
`BINARY check --timeout 30s FILE`. All std files additionally receive
`check --timeout 30s --relax-modules FILE`. Four workers, outer timeout 40 seconds.
All original raw commands/status/stdout/stderr and normalized diagnostics are
retained in the [JSON evidence](../../../.ailang/state/sprints/validation_M-EFFECT-ROW-VAR-UNIFICATION.json).

| Corpus | Files | Examples | Std | Checks | Pass | Reject | Infrastructure errors |
|---|---:|---:|---:|---:|---:|---:|---:|
| Base | 496 | 447 | 49 | 545 | 462 | 83 | 0 |
| Fixed | 498 | 449 | 49 | 547 | 464 | 83 | 0 |

**Zero existing status flips.** New files are
`examples/runnable/effect_row_var_pure_caller.ail` and
`examples/runnable/effect_row_var_noisy_twice.ail`. Existing failures: 31 type,
42 module, 7 parse and 3 effect errors on each side. They remain visible per file;
shared example resolution failures do not substitute for std coverage. All 49 std
modules pass both direct and relaxed checks. No std resolution import fallback
was needed. The dev/v0.53.0 stdlib version warning occurs on both sides.

During development ten valid examples initially regressed. Each was repaired
before the final comparison, including concrete annotations, curried lambdas,
recursive closures and implicit callback rows; none is reclassified as unsound.
No corpus path needs migration. The fragment separately documents migration for
the synthetic formerly accepted pure repeated-call helper.

Streaming consumer inventory: std/stream, std/stream/bridge, std/ai/streaming;
examples/serveapi_ws_bridge; runnable ai_call_stream, ai_stream_openai,
stream_multi_source, stream_process_source, stream_sse, stream_websocket.
Every direct consumer retains its status. Tests import each onEvent API, accept
two calls sharing `{Stream, e}` and accept concrete `{Stream, IO}`; a Stream-only
caller rejects naming missing IO. Checked union tests reject distinct tails
explicitly and preserve sole/same tails. No streaming network access was required.

## Regression and mutation matrix

| Boundary | Base measurement / fixed assertion |
|---|---|
| Pure runIt(quiet) | Base rejects with empty Suggested fix; fixed accepts |
| Pure repeated callback helper | Base accepts and prints CALLBACK_MARKER twice with IO; fixed rejects runTwice before printing |
| Corrected helper | Generic helper and IO caller print exactly twice; missing capability still rejects |
| Independent instantiations | Pure/IO calls in both orders and later pure caller pass |
| Return-only row | Pure calls with zero args and non-callback int arg pass |
| Wrong FS / generic tail | Missing IO named; declared `{e}` cannot absorb concrete IO |
| Publication | Missing, nil, wrong kind and unowned tail fail at App; pure record differs from absent; snapshots and sealed rows resist mutation |
| Row algebra | Same/sole tails preserved; distinct conflict; matching ownership subsumes; empty diff becomes invariant |
| Imports / historical controls | Declared/inferred HOF importer, recursive declared-pure closure, concrete contamination, #386 and #1708 suites pass |

#1091 synthetic recursive helper checks as `check main.ail` and `check .` (two
explicit source files). Extra positional CLI filenames are not a supported
multiple-file mechanism. The external extraction patch is unavailable, so this
is bounded coverage. Existing inferred combinator/interface controls also pass.

Mutations were applied separately and restored: detaching App row fails
TestApplicationSharesCalleeRow; erasing union tail fails
TestValidationUnionPreservesTail; disabling published call-row consumption fails
TestEffectRowVariableDischarge/pure with unresolved e. All exit 1; raw outputs and
commands are in the evidence. Final focused suites run after restoration.

## Demonstrations and required gates

AILANG prompt version loaded: v0.16.6 from the rebuilt base before `.ail` edits.
Pure demo checks/runs to 42 and has a passing inline test. Noisy demo checks and
runs with `--caps IO --entry main`, output `callback` exactly twice. Contracts are
omitted because these are signature regressions; the IO demo uses runtime stdout
assertions instead of an inline test. Manifest and effects/limitations docs updated.

Required gate results and captured logs appear in the evidence. Full `make test`
was deliberately not run, following the approved RAM-backed /tmp restriction.
Core tests use CGO with a Zig C compiler for sqlite; other checks use bounded Go
parallelism/memory. Initial missing tools, cgo-disabled sqlite failure and OOM
attempts were resolved; the final required commands passed without reducing scope.
`typechecker_core.go` remains exactly 800 lines; functions companion 789 lines.

Remaining limitations: #1718 nested callback arguments and concrete annotation
upper bounds. Runtime capability enforcement remains a backstop, while this repair
rejects the explicit open-row pure helper statically.

Evaluator handoff: direct inbox rejected because no evaluator agent is registered.
Coordinator inbox delivery failed because the installed messaging CLI lacks cgo
SQLite; two bounded cgo CLI rebuilds were killed by the memory limit. No message
was delivered. Coordinator completion markers provide the artifact manifest for
independent evaluator routing. These extra messaging builds do not change the
successful compiler/test/core/lint gate results.
