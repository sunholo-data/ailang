# Sprint Plan: M-PACKAGE-TEST-DISCOVERY

**Design doc:** [m-package-test-discovery.md](m-package-test-discovery.md)

**Sprint ID:** M-PACKAGE-TEST-DISCOVERY

**Issue:** #959

**Target:** next patch release after v0.42.0

**Duration:** 2 working days (about 12 hours)

**Risk:** Medium — discovery and exit semantics can create false-green test runs

## Goal

Make `ailang test --package <dir>` run the deterministic union of `*_test.ail` files and
ordinary modules that contain named, function-inline, or property tests. Preserve the existing
per-file runner as the only execution/seed-identity seam, and remove the duplicated result merge
loop shared by package and directory modes.

## Current State and Planning Basis

- At v0.42.0, `runPackageTests` in `cmd/ailang/test.go` walks every `.ail` file but runs only the
  `*_test.ail` partition. An inline-only package therefore exits 0 before invoking the runner.
- `runTestsV2` already passes every `.ail` file through `runTestFile`; files without collected tests
  return `nil`. Its discovery semantics do not need to change.
- Both command paths duplicate all `SuiteResult` merge fields. A shared helper can remove this
  drift without changing reporting or success policy.
- `RunTestsFromFileWithConfig` already resolves module identity and derives property seeds per file.
  Package discovery must continue to call through `runTestFile`, never recreate execution from a
  discovery-time `TestSuite`.
- The last seven days contain only the v0.42.0 release commit, so repository-wide LOC/day is not a
  useful velocity signal. The approved design estimates two days; the bottom-up estimate below is
  ~330 changed LOC (mostly subprocess tests), which fits that box with review buffer.

### Scope clarification

The design doc calls directory-mode alignment both a success metric and a non-goal. This sprint
interprets that consistently: directory **discovery** remains unchanged, while the duplicated
**aggregation** loop is factored into a helper used by both modes. No new directory-mode filtering,
manifest rule, or output semantics are introduced.

## Milestones

### M1 — Pin the false-green boundary with end-to-end tests

**Estimate:** 4 hours; ~150 test/fixture LOC

**Files:** create `cmd/ailang/test_package_discovery_test.go`; reuse helpers in
`cmd/ailang/main_test.go`

Tasks:

1. Add a temporary-package fixture with `ailang.toml` and an ordinary module containing inline
   tests/properties but no `*_test.ail` file.
2. Add mixed-convention and empty-package fixtures, plus stable names/order assertions in JSON.
3. Add a syntactically invalid ordinary module and assert package mode exits 1 with a synthetic
   `parse` failure rather than treating the suite as empty.
4. Add seed parity: for the same inline property and `--seed 42`, compare its reported seed under
   `--package` and direct-file invocation.

Acceptance criteria:

- [ ] The inline-only test fails on the v0.42.0 baseline by observing zero executed tests.
- [ ] Mixed discovery reports dedicated and inline-bearing files in lexical walk order.
- [ ] An empty union exits 0 and its warning names both `*_test.ail` and inline test blocks.
- [ ] A parse error in an ordinary source module exits 1 and appears as `parse` in JSON.
- [ ] Package and direct-file modes report the same derived seed for the same inline property.

### M2 — Implement union discovery and shared aggregation

**Estimate:** 5 hours; ~70 implementation LOC + ~30 focused unit-test LOC

**Files:** update `cmd/ailang/test.go` and `cmd/ailang/test_package_discovery_test.go`

Tasks:

1. Preserve one walk-ordered `.ail` slice. Dedicated test files always enter the run list; ordinary
   modules enter when collection finds tests/properties. Ordinary modules that fail parsing also
   enter the run list so `runTestFile` can synthesize the failing parse result.
2. Invoke every selected file through `runTestFile` → `RunTestsFromFileWithConfig`; do not execute a
   discovery-time AST or derive seeds in the CLI.
3. Move the package empty-union decision until after discovery. Update the package preamble/counts
   without contaminating JSON stdout.
4. Extract a small `mergeSuiteResult` helper and use it from both `runPackageTests` and `runTestsV2`.
   The helper must merge tests, properties, all pass/fail/skip/vacuity counters, and duration; seed
   metadata remains stamped once on the aggregate.

Acceptance criteria:

- [ ] `go test ./cmd/ailang -run 'TestPackageTestDiscovery|TestPackageAndFileAggregates'` passes.
- [ ] Removing ordinary-module admission makes the inline-only regression test fail.
- [ ] Removing parse-error admission makes the parse-error regression test fail.
- [ ] Existing JSON purity, `--allow-skips`, replay, and aggregate seed tests remain green.
- [ ] No production changes occur under `internal/testing`; the established per-file seam remains.

### M3 — User-facing contract and full validation

**Estimate:** 3 hours; ~20 help/changelog LOC + validation

**Files:** update `cmd/ailang/test.go`, relevant help assertions if present, and `CHANGELOG.md`

Tasks:

1. Update `ailang test --help` and examples to say package mode discovers dedicated files and inline
   test blocks through `ailang.toml`.
2. Add a changelog entry calling out the intentional behavior change and issue #959.
3. Run focused and package-wide gates, then formatting/lint appropriate to the touched Go files.

Acceptance criteria:

- [ ] `go test ./cmd/ailang/...` passes.
- [ ] `make fmt` leaves no diff beyond intended formatting.
- [ ] `make lint` passes, or any unrelated pre-existing failure is recorded verbatim.
- [ ] `make test` passes, or any unrelated pre-existing failure is recorded verbatim.
- [ ] `ailang test --help` describes both discovery conventions.

## Day Plan

| Day | Work |
|---|---|
| 1 | M1 red tests; M2 ordered union discovery and parse-error handling |
| 2 | M2 shared aggregation and compatibility suite; M3 help, changelog, full gates |

## Success Metrics

- Inline-only package: nonzero tests execute under `--package`.
- Mixed package: each test/property executes once in deterministic file/declaration order.
- Empty package: explicit warning and exit 0; invalid ordinary module: explicit failure and exit 1.
- Seed identity is mode-independent for the same property.
- No duplicated aggregate merge loop remains in `cmd/ailang/test.go`.

## Dependencies and Risks

- **Dependencies:** none; design is approved and issue #959 is linked.
- **Double execution risk:** discovery must collect only; execution stays exclusively in
  `runTestFile`.
- **False-green parse risk:** parse failures must be candidates even though no suite can be
  collected from their AST.
- **Ordering risk:** retain `filepath.Walk` order in one slice; do not append two partitions.
- **Performance risk:** ordinary inline-bearing modules may be parsed during discovery and again at
  execution. Accept for this sprint; optimize only with trace evidence.
- **Compatibility risk:** users who treated `--package` as “dedicated files only” will run more
  tests. This is intentional and must be stated in the changelog.

## Handoff

After human approval, invoke `sprint-executor` with
`.ailang/state/sprints/sprint_M-PACKAGE-TEST-DISCOVERY.json`. The executor should use TDD, preserve
the per-file seed seam, and update only milestone progress fields in the JSON.
