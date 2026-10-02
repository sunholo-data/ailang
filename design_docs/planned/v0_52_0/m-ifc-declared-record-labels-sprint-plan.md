# Sprint Plan: M-IFC-DECLARED-RECORD-LABELS

**Design doc**: `design_docs/planned/v0_52_0/m-ifc-declared-record-labels.md`
**Issues**: #1523, #1527 (dup)
**Duration**: 1 day (single session), 4 milestones
**Registry reuse**: none — compiler-internal analysis, no package-like capability (all milestones).

## M1 — Failing tests (TDD)

`internal/types/ifc_declared_test.go`, one module source per shape, run through `types.CheckModuleIFC`.

| AC | Test |
|---|---|
| p1–p13, r1–r4 each rejected with a `SinkRefinementError` | `TestIFCDeclared_Rejected/<shape>` (17 subtests) |
| inline-record behaviour unchanged (still rejected) | `TestIFCDeclared_InlineRecordStillRejected` |
| sibling unlabelled field stays clean for param / call / let / annotated let / getter / getter-of-call | `TestIFCDeclared_FieldPrecision/<shape>` (6 subtests) |
| annotation cannot lower a label (V12 swap) | `TestIFCDeclared_AnnotationCannotLower` |
| mcp_oauth idioms keep working: classifier raises, plain-string return propagates, Declassify digest lowers | `TestIFCDeclared_McpOauthIdioms` |

Exit: rejection tests FAIL, precision/idiom tests PASS on the unfixed tree.

## M2 — Implementation

- `internal/types/ifc_static_type.go` (new, ~200 LOC): type-decl/constructor index, `deepLabel`, `fieldType`, `staticType`, `sameType`.
- `internal/types/ifc_check.go`: env `{label, typ}`, `labelOf = flowOf ⊔ deep(staticType)`, `handOff`, param-type seeding in `checkFunc` and `effectiveBodyLabel`, lambda/func-literal param annotations.

| AC | Command |
|---|---|
| all M1 tests pass | `go test ./internal/types/ -run 'TestIFC'` |
| scoped packages green | `go test ./internal/types/... ./internal/elaborate/... ./internal/iface/... ./internal/pipeline/...` |
| fast language loop green | `make test-core` |
| all 17 probe shapes rejected end to end, positive probe clean | `ailang check fx/probe.ail fx/probe2.ail` (each file separately), `ailang check fx/pos.ail` |

## M3 — Mutation test + regression surface

| AC | Command |
|---|---|
| mutant (RecordAccess declared-label join removed, scratch copy) COMPILES and ≥1 `TestIFCDeclared_Rejected` subtest fails | `go test ./internal/types/ -run TestIFCDeclared` on the mutant |
| `inbox_injection_v2.ail` verify output identical before/after | `ailang verify examples/runnable/contracts/inbox_injection_v2.ail` (diff old vs new binary) |
| examples green | `make verify-examples` (once) |
| mcp_oauth flow.ail compiles and IFC leak tests pass | `cd packages/mcp-oauth && bash tests/ifc_leaks.sh` (ailang-packages `sprint/mcp-oauth`, new binary first on PATH) |
| docparse IFC module still checks | `ailang check docparse_api/services/api_keys.ail` |

## M4 — Gates, docs, ship

| AC | Command |
|---|---|
| gates | `make check-file-sizes check-boundaries check-architecture-closure`, `gofmt -l internal/types`, `golangci-lint run ./internal/types/...` |
| docs | `docs/docs/guides/ifc-labels.mdx` section on labels inside types; changelog fragment `changelogs/unreleased/2026-10-02-ifc-declared-record-labels.md` |
| shipped | push to dev; `gh run list -R sunholo-data/ailang --branch dev --workflow ci.yml` green; #1523 commented + closed, #1527 closed |
