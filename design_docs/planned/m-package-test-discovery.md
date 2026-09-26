# Unify Package Test Discovery: Inline Test Blocks Under `ailang test --package`

**Status**: Planned
**Type**: DX / test-runner change (CLI + orchestration only; no language semantics)
**Target**: v0.38.x
**Priority**: P1
**Estimated**: 2 days
**Source**: GitHub issue #959 (M-DX-PI-HARNESS dogfooding run; reproduced at HEAD 2026-09-13)
**Dependencies**: None

## Problem Statement

AILANG packages use two test conventions:

1. **Dedicated files** — `*_test.ail` alongside source modules.
2. **Inline blocks** — top-level `test "name" { … }` / `property "name" = …` declarations, plus
   function-attached `tests [...]` / `properties [...]` (collected by
   `internal/testing/collector.go`), inside ordinary exported modules.

`ailang test --package .` (implemented as `runPackageTests` in `cmd/ailang/test.go`) discovers
**only** `*_test.ail` files. A package whose tests live entirely in inline blocks reports
`⚠ No *_test.ail files found in package <name>` and exits 0 — the suite silently appears empty
while `ailang test mod.ail` on the same module runs the inline tests fine. This is exactly what
issue #959 hit during the M-DX-PI-HARNESS dogfooding run: the real-world package used inline
blocks, and package mode reported 0 test files.

**Impact:** Any agent or CI pipeline using `--package` (the natural "run this package's tests"
entry point) gets a false green on packages that only use inline blocks. The two conventions
have two disjoint discoverers; there should be one.

## Goals

**Primary Goal:** `ailang test --package .` runs the union of `*_test.ail` files and inline
test/property blocks from a package's modules, with one deterministic discovery order and one
aggregation path.

**Success Metrics:**
- Package with only inline blocks: `--package` finds and runs them (0 → N tests).
- Property seeds identical whether a module is run via `--package`, via `ailang test mod.ail`,
  or via `RunTestsFromFile` (seed regression tests prove it).
- One code path for file discovery + aggregation shared by package mode and directory mode
  (`runTestsV2` currently duplicates `runPackageTests`' aggregation loop verbatim).
- Exit code semantics unchanged and consistent across the union.

## Design

### Discovery semantics

Package mode's `filepath.Walk` already partitions `.ail` files into `testFiles` and
`sourceFiles`. Change: for each file in `sourceFiles`, run collection to detect inline tests,
and run those files too. Concretely:

- Parse each source module; use `NewCollector(path).Collect(file)` (the same collector the
  runner uses — one discoverer, two conventions). Files whose collected `TestSuite` has zero
  tests and zero properties contribute nothing and are not counted.
- The "no tests found" warning and exit-0 path move to the **union**: empty package mode result
  (0 test files AND 0 inline-bearing modules) is what triggers it.
- Keep `ailang.toml` as the package-mode gate (manifest still required) — discovery scope is
  the walk, not the manifest.

Non-goal: changing `runTestsV2` (bare directory mode) — it already walks all `.ail` files and
would incidentally cover inline blocks; aligning it with package mode is a follow-up, not part
of this change.

### Deterministic ordering

Per CLAUDE.md determinism principles, discovery output must be a stable total order:

1. `filepath.Walk` order (lexical per directory level, depth-first) is already deterministic —
   keep it, do not re-sort.
2. Within the union, run files in walk order; `*_test.ail` files and inline-bearing source
   modules are interleaved by walk position, not partitioned into two phases. This keeps the
   per-file order identical to `ailang test <dir>` semantics and makes the aggregate report
   reproducible byte-for-byte (same files, same order, same seeds).
3. Within a file, the `Collector` already appends in declaration order (`TestDecl`/`PropertyDecl`
   before per-`FuncDecl` inline cases, in AST order) — unchanged.

### Seed / identity handling

This is the subtle part. Property seeds derive via
`DeriveSeedV1(masterSeed, moduleIdentity, propertyName)` where identity is resolved per-file by
`ResolveModuleIdentity` inside `RunTestsFromFileWithConfig`. Because the seam is
**per-file invocation**, seeds are automatically stable across modes: an inline property in
`mod.ail` gets the same `moduleIdentity` whether the file was reached via `--package` or via
`ailang test mod.ail`. Requirements:

- **Do not** hoist collection above the per-file seam and re-implement running — always call
  `runTestFile` → `RunTestsFromFileWithConfig` per file, so identity resolution stays in one
  place (D4: one consumption point for `WorkspaceRoot`).
- Mixed conventions: a property named the same in a `*_test.ail` file and inline in a module
  still gets distinct seeds because identity differs per file path/module. No cross-file seed
  collision handling needed; document this invariant with a regression test
  (`--package` seed == single-file seed for the same inline property).
- `--seed N` / `--random-seed`: the master seed is already threaded once via `cfg` into the
  aggregate `SetSeedMetadata(cfg)` and per-file `RunTestsFromFileWithConfig` — unchanged; the
  union merely widens which files consume it.

### Failure aggregation and exit codes

- Reuse the existing aggregation (append per-file `Tests`/`Properties`, sum counters and
  durations) — factor the duplicated loop in `runPackageTests`/`runTestsV2` into a shared
  helper rather than adding a third copy.
- Parse errors in an inline-bearing source module must not be silently swallowed: `runTestFile`
  already synthesizes a failing `parse` test result — that flows into the union aggregate and
  forces exit 1. A package whose module fails to parse must not exit green.
- Exit codes unchanged: `Success()` (or `SuccessAllowingSkips()` with `--allow-skips`) over the
  union result. Empty union still exits 0 with the warning (surfaced clearly, mentioning both
  conventions: "no *_test.ail files and no inline test blocks found").

### Relation to `RunTestsFromFileWithConfig` (the seam)

`RunTestsFromFileWithConfig(filePath, file, cfg)` is the single entry point that validates
config, resolves module identity, collects, runs, and stamps seed metadata. The design keeps
it exactly as the seam: package mode changes **only what files get passed to it** (discovery),
never how results are produced. This guarantees mode-parity of seeds, names, and result
shapes, and confines the change to `cmd/ailang/test.go` discovery/aggregation plus tests.
No changes to `internal/testing/runner.go`, `collector.go`, or `config.go` are required.

## Implementation Notes (for the sprint plan)

1. `cmd/ailang/test.go`: in `runPackageTests`, run collection over `sourceFiles`; collect
   inline-bearing modules into the run list interleaved in walk order.
2. Factor the shared aggregation loop (`aggregateResults` merging) into a helper used by both
   package and directory modes.
3. Update the "no tests found" message to mention both conventions; update
   `printTestHelp` (`--package` description: "discover `*_test.ail` files **and inline test
   blocks** via ailang.toml").
4. Tests: e2e package fixture with (a) inline-only, (b) mixed, (c) empty packages; seed-parity
   test (`--package` vs single-file for the same inline property, same reported seed); parse
   error in module → exit 1.

## Alternatives Considered

- **Collect-only-in-test-files convention (status quo):** rejected — issue #959 shows real
  packages use inline blocks; forcing a convention contradicts the language's own examples.
- **Run inline blocks via a separate phase after `*_test.ail` files:** rejected — two phases
  means two orders and a seed/report ordering dependent on convention partitioning; interleave
  in walk order instead.

## Risks

- Larger per-file parse cost in package mode (every source module is parsed twice: once for
  discovery-collection, once inside the runner). Acceptable at package scale; note as a known
  cost with a follow-up to cache the AST between discovery and run if it shows in traces.
- Users relying on `--package` to only exercise `*_test.ail` get more tests run — behavior
  change is the point; flag in changelog.