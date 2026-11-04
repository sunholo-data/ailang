# Sprint Retrospective: M-STD-JSONSCHEMA

Implemented the approved ten-milestone sprint on `coordinator/task-b48daa42`.
Compiler/reference, validator/annotation, and list/census work ran concurrently
with public API and documentation integration. Shared-file ownership avoided
merge conflicts; follow-up commits address independently reproduced defects.

## Outcomes

- Strict schema audit, local resource references, dynamic anchors, scalar and
  container assertions, applicators, and successful-branch annotation propagation.
- Pure strict, lenient, and schema-only APIs with interpreter/VM parity.
- `std/list.all` uses the approved negated-any implementation and retains the
  existing iterative HOF's short circuit and large-list behavior.
- Exact synthetic census fixture; 100 reversed-key compiles pin deterministic
  error output. This does not reproduce the external client's envelope failures.
- Runnable envelope self-check and API docs describe RE2, annotation-only format,
  schema versus document roots, and float64 limitations.

## Measurements

Linux amd64, AMD EPYC 7B13, Go benchmark with 460 schema nodes and 1,000 document
properties. Initial engine compile+validate: 8.85 ms/op. Final concurrent-load
run: 24.93 ms/op, 1,766,962 B/op, 23,816 allocs/op. Public strict validate including
Json ADT conversion and output construction: 38.65 ms/op, 2,550,336 B/op,
33,795 allocs/op. Both final measurements meet the 50 ms target on this machine;
these are measurements rather than a universal latency guarantee.

## Friction and review

Go was installed outside PATH; checks use `/usr/local/go/bin`. The initial
session helper hid the missing-go failure through its output filter, so focused
gates were rerun directly. Cold CLI dependencies required downloading and
compiling cloud and SQLite packages. `golangci-lint` is unavailable; targeted
`go vet` and formatting checks supply partial verification, with lint left to CI.
No toolchain was installed or test semantics changed.

Independent review reproduced composite type names accepted by a substring test,
resource-local reference scope, dynamic anchors under unvisited definitions,
static pointer dynamic references, overly tolerant large-number multipleOf,
cycles hidden by predicates, and an overbroad reference-depth counter. Each
was corrected with regression coverage. The VM pure-builtin coverage ratchet
required native bridges for all three new APIs; no ratchet exception was added.

## Follow-up

Retain resource-scope and unevaluated-location tests when extending vocabulary.
The engine deliberately has no remote-reference support or schema cache. Future
conformance work may vendor the official suite; that remains outside this sprint.
