# M-ALIAS-BODY-CLOSURE implementation evidence

Implementation follows the approved [design](m-alias-body-closure.md) and
[sprint plan](m-alias-body-closure-sprint-plan.md). Work branch:
`coordinator/task-a660f5f6`; planner handoff: `task-ba132afd`; issue: #1614.
Independent sprint evaluation passed (95/100); no merge or release is performed.

## Delivered behavior

Interface construction now rebuilds alias bodies, export scheme types and
constructor fields/results in the defining module's post-shadow imported alias
environment overlaid with all local aliases, including private dependencies.
Closure runs before transitive embedding. The existing `SetDerivedEq` call then
recomputes the final digest; `internal/iface/builder.go` remains at 782 lines.

The walker covers every current concrete `types.Type`, verified by enumerating
actual `Substitute` receivers in the types package. It preserves record tags,
variables, scheme binders/constraints, row metadata and nominal constructor
results. It never mutates input types or shared schemes/constructor entries.
Only acyclic expansions are memoized: cycle-truncated expansions cannot be reused
under another root. Alias roots and record fields are traversed in sorted order.

Cache version v6 rejects v5 manifests. The regression fixture verifies cold
publication of every reachable module, zero fresh core encodes on the unchanged
warm run and cached/uncached runtime equivalence.

## Verification

The pipeline/types/iface baseline passed before implementation. The new regression
matrix failed first on the original implementation: valid T1/T2/T3 programs were
rejected, and `countItem(one())` incorrectly accepted incompatible record shapes.
After implementation all valid calls pass and incompatible records are rejected.

- `go test ./internal/pipeline/... ./internal/types/... ./internal/iface/... -cover`:
  passed. Coverage: pipeline 75.5%, types 52.7%, types/traverse 92.1%, iface 42.4%.
- Runtime matrix: T1/T2/T3 and both import orders return 1; cross-module record
  update returns 1. Constructor patterns, private alias dependencies, local
  precedence, parameterized-head controls and self/mutual cycles pass.
- Closure digest changes, deterministic recomputation and cache serialization
  round-trip pass. Positive local-alias capture regression passes; nominal capture
  retains `TC_TYPE_SHADOW_001`.
- Source binary: `ailang check --package examples/alias_body_closure` passes all
  three modules; `run --package-dir examples/alias_body_closure --entry run
  --args-json 1 examples/alias_body_closure/main.ail` returns 1. `countInput` with
  `--args-json '{"items":[{"x":1,"w":2}]}'` also returns 1.
- `ailang check --package examples/intra_package_imports` passes both modules;
  `greet` returns `Hello, closure!`.
- `make check-boundaries check-file-sizes` and `make fmt-check`: passed.
- `make lint`: passed with zero issues on the warm run. Initial cold run exceeded
  the configured five-minute timeout while reporting zero issues.
- `CC=/workspace/task-tools/cc CGO_ENABLED=1 make test-core`: passed, including
  the SQLite-backed brain-store tests. Source build also passed with CGO enabled.

No external Stapledons checkout is available. The mandatory automated four-module
navigation/sol/trappist regression passes without renaming Planet.

## Limits and environment

Applied parameterized alias heads are deliberately opaque while their arguments
close; recursive cycle references remain opaque. Nominal ADT identity remains
unqualified. Importer-written ambiguous names still use last-explicit/first-bulk
resolution. Import ambiguity diagnostics and REPL/WASM/SMT top-level name merging
remain deferred. These limits are documented in LIMITATIONS and the parent design.

Go was installed but absent from PATH. jq, make and golangci-lint were provisioned
outside the repository. The startup script was copied outside the repo to replace
its full-suite invocation with test-core as required by the approved plan; its
missing-make false positive was superseded by direct verification. Initial
`make test-core` encountered CGO-disabled SQLite brain-store tests because the
container lacked a C compiler. A local Zig C toolchain enables the corrected run.
Full `make test` is intentionally left to CI per the approved sprint plan.


## Independent handoff

The separate `/root/sprint_evaluator` agent received the design, plan, progress,
implementation diff, validation logs and residual limits. The local CLI handoff
was refused because no worker serves `sprint-evaluator` in this environment;
collaboration dispatch supplies the independent judge. Evaluation passed: 95/100, all 17 acceptance criteria met.


Independent evaluator report:
[eval_M-ALIAS-BODY-CLOSURE_round_1.json](../../../.ailang/state/evaluations/eval_M-ALIAS-BODY-CLOSURE_round_1.json).
The evaluator also ran `make verify-examples`: 232 passed, 0 failed, 9 skipped;
211 modules checked with zero manifest drift. No semantic defects or hard failures
were found. The five-point deduction concerns long test functions.

Measured change: 661 added Go/example lines versus the 720-line total sprint
estimate; total additions also include progress, documentation and evaluation evidence.
