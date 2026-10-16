# m-test-package-source-check — `test --package` type-checks every source module

**Status**: Implemented in v0.54.13 (commit 03e53668c, #1845) as `checkUntestedSources` in `cmd/ailang/test_package_check.go`: after the tests, every non-test source module is checked via `check.CheckPackageFiles`; compile/interref/read/timeout failures exit 1 on stderr. STRICT_FALLBACK stays a `check --package` gate.
**Target**: v0.54.13
**Priority**: P1 (external CI correctness + a measured 2.0–3.6 min/run saving on the stapledons-godot 2-core runner)
**Estimated**: 1–1.5 days (implementation ~3h, output contracts ~2h, tests + docs ~3h, buffer)
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Every feature must align with AILANG's 12 Design Axioms. Score each axiom and verify no hard violations.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | The pre-pass is the same deterministic per-file loop `check --package` runs (sorted path order, no timing dependence in outcomes). |
| A2: Replayability | 0 | Seed metadata and the replay command line are unchanged. |
| A3: Effect Legibility | 0 | No effect-row or capability changes. |
| A4: Explicit Authority | 0 | No capability changes; the pre-pass reads the same files the walk already reads. |
| A5: Bounded Verification | +1 | A green `test --package` now bounds verification at the *whole package* (every module-decl file compiles), not just at what tests happen to import — the gap that let a type-broken module ship behind passing tests. |
| A6: Safe Concurrency | 0 | No concurrency changes. |
| A7: Machines First | +1 | The exit code stops lying to CI agents: today `test --package` exits 0 on a package that does not compile, and even its banner ("N source modules") implies those modules were part of the run. Machine CI (stapledons-godot) had to compensate by running a second command. |
| A8: Minimal Syntax | 0 | No parser or grammar change. |
| A9: Cost Visibility | 0 | No resource-metering changes. |
| A10: Composability | +1 | Reuses `internal/check`'s exported package path (`DiscoverPackageSources` + `CheckPackageFiles`) verbatim instead of a second ad-hoc walk; composes with the existing compile cache and output-format family. |
| A11: Structured Failure | +1 | Failures move from "silently absent" to structured: per-file ✗ lines in human mode, a `source_check` block in the JSON report, exit 1 — same shape `check --package` already emits. |
| A12: System Boundary | 0 | No boundary changes. |

**Net Score: +4** → **Decision: Move forward**

### Hard Violation Check

**These axioms cannot have −1 scores (automatic rejection):**

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

## Problem Statement

`ailang test --package .` reports "N source modules, M test files" but only compiles the
`*_test.ail` files (and, through module resolution, whatever they import). A source module that
no test imports is never type-checked, so it can contain a type error and `test` still exits 0
with "All tests passed". `ailang check --package` catches the same error, so CI has to run both.

**Minimal repro (verified live, binary v0.54.11-dirty; same code at HEAD — see Verification Log V1–V3):**

```
ailang.toml:   [package] name="probe/p", version="0.1.0", edition="1"
               [exports] modules=["probe/p/good","probe/p/bad"]
good.ail:      module probe/p/good
                 export pure func one() -> int = 1
good_test.ail: module probe/p/good_test
                 import probe/p/good (one)
                 test "one is one" { one() == 1 }
bad.ail:       module probe/p/bad
                 export pure func oops() -> int = "not an int"

$ ailang test --package .    → "2 source modules, 1 test files" → 1 passed → exit 0
$ ailang check --package .   → ✗ bad.ail: cannot unify string vs int → exit 1
```

**Current State:**

- `runPackageTests` (cmd/ailang/test.go:200) walks the package dir, *splits* the walk into
  `testFiles`/`sourceFiles`, prints the source count in the banner (test.go:244), then runs only
  `testFiles`. `sourceFiles` is used for nothing but the count (V4).
- **Systemic audit — the same gap exists in dir mode:** plain `ailang test .` on the repro
  directory also exits 0 with "All tests passed" — a test-less file with a type error is skipped
  without even a warning, because the runner compiles per test body and zero tests means zero
  compiles (V3). File/dir mode has no manifest, so this sibling needs its own design; it is
  audited here and explicitly deferred (Non-Goals).
- **Measured cost of running both** (stapledons-godot CI, 2-core runner, as reported):
  `ailang check --package sim` over 108 files takes **2.0–3.6 min**, then `ailang test --package .`
  re-compiles much of the same code (~200 s of its ~13 min wall) — i.e. roughly a full extra
  check pass per CI run, purely to make the test step's exit code honest.

**Impact:**

- Every CI consumer of `test --package` (stapledons-godot today; `ailang pkg quality`, the
  coordinator cascade, and the motoko-extension CI template in-repo) either pays double or
  trusts a green that does not mean "the package compiles".
- Machine agents (A7): a "tests passed" signal that is false at the package level is exactly
  the kind of mis-verification an AI-driven loop builds on.

## Goals

**Primary Goal:** One command — `ailang test --package .` — proves what `check --package` proves
(every source module type-checks) *and* runs the tests, so CI drops the separate check step with
no loss of coverage.

**Success Metrics:**

- The repro package above: `test --package` exits 1 naming `bad.ail`'s type error (today: exit 0).
- A green package: identical outcomes/exit code as today (regression fixtures in Conflict Surface).
- CI step count for the reported consumer: 2 compile-bearing steps → 1 (the dropped `check` step).
- JSON report stays one valid document; existing machine parsers (`pkg quality`) unaffected
  (additive `source_check` key, verified tolerated — V9).
- No new type-checker or parser surface: the pre-pass reuses `check --package`'s loop verbatim.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|----------------|-----------|----------|-------------|
| Default-on subsumption (not an opt-in `--check-all`) | This is the reporter's stated preference and the ecosystem convention (`go test ./...`, `cargo test` compile the whole package). Opt-in keeps the lying default in place for every existing caller; default-on changes `test --package`'s exit code on broken packages (0→1), which external CI may depend on | human | design | high |
| Pre-pass = the *full* `check --package` pass (compile + InterFunctionRefs + StrictFallbacks), not a bare type-check | If CI drops `check --package`, anything the check step caught must now be caught by the test step, or the publish boundary (STRICT_FALLBACK_001, strict-fallbacks guide) silently weakens | agent | design | med |
| Fail-fast: on pre-pass failure, print errors and exit 1 **without running tests** | Matches `go test`/`cargo test` (no tests run when the package doesn't compile); saves the wall time the reporter is asking about; running tests anyway would give a green "1 passed" line above a failing package | agent | design | med |
| Add an opt-out `--no-source-check` flag | `pkg quality` already measures the whole compile in-process (same `CheckPackageFiles`, pkg_quality.go:105-109) and would otherwise pay it twice; third parties that pre-check need an escape hatch | agent | design | low |
| JSON: extend the suite report with a top-level `source_check` object (not a second stdout document, not stderr-only) | One JSON document per run keeps the `pkg quality` subprocess contract (`json.Unmarshal(stdout, …)`, V9) and the `TestTestCommand_JSONStdoutPackage` "stdout is pure JSON" contract intact; two documents or human text on stdout would break both | agent | design | med |
| No new cross-pass artifact sharing; rely on the existing on-disk compile cache for dedup, and *measure* the hit rate in-sprint | Cache identity (`compilerIdentity`: commit + dirty-fingerprint + release, V8) already matches across the check/test passes, so a custom sharing mechanism is probably unneeded — but claiming the dedup without measuring is exactly the kind of unverified assertion this program rejects | agent | compile | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Default-on; the reporter's "or add `--check-all`" alternative is rejected in favor of default-on + opt-out (rationale above; recorded for quorum).
- [x] Full `CheckPackageFiles` reuse (analyses included).
- [x] Fail-fast exit 1, no test execution on pre-pass failure.
- [x] Flag name `--no-source-check` (namespace verified free — V7).
- [x] JSON via additive `source_check` key on the existing suite document.
- [x] `pkg quality` passes `--no-source-check` to its subprocess.

## Solution Design

### Overview

`runPackageTests` gains a **source-check pre-pass** between the banner and the test loop: it
discovers package sources exactly as `check --package` does (`check.DiscoverPackageSources`) and
runs them through the same exported loop (`check.CheckPackageFiles`, internal/check/package.go:75)
with the same options `check --package` uses today. On any file failure it renders the failures in
`check --package`'s shape, prints the (zero-test) suite report, and exits 1 — tests do not run. On
a green pre-pass it proceeds to the existing test loop unchanged. Because the pre-pass and the
test-compile pass share the on-disk module compile cache (default-on, M-PERF6), modules the tests
import should largely be cache hits rather than recompiles.

### Architecture

**Components:**

1. **Source-check pre-pass** (`cmd/ailang/test.go`, `runPackageTests`): after manifest load and
   test-file discovery, call `check.DiscoverPackageSources(absDir)` then
   `check.CheckPackageFiles(absDir, manifest, sourceFiles, check.PackageOptions{OnFile: per-file
   progress line})`. `CheckPackageFiles` pins the compile config internally (DryLink, RelaxModules,
   PackageDir — internal/check/package.go:114-120), so the pass is byte-identical in coverage and
   semantics to `check --package`; `PackageOptions.Timeout` stays zero (unbounded), matching
   `check --package`'s default. Discovery is by module declaration (not by `_test.ail` suffix), so
   `*_test.ail` files are checkable modules and module-less orphans are excluded and reported,
   exactly as in `check --package`.
2. **Flag plumbing** (`cmd/ailang/commands_language.go`, `runTestCommand`): new
   `testFlags.Bool("no-source-check", …)` threaded into `runPackageTests`. `--package` with the
   flag keeps today's exact behavior.
3. **Output contracts**: human mode prints the per-file `✓/✗` lines (same glyphs/order as
   `check --package`) plus the error list before the suite summary; JSON mode adds one top-level
   `source_check` object (`{ "passed": int, "failed": int, "errors": [string], "warnings": [string] }`)
   to the existing suite document. `SuiteResult` (internal/testing/result.go:48) gains an optional
   `SourceCheck` field set only in package mode; `reportJSON` (internal/testing/reporter.go:48)
   emits it when non-nil.
4. **Consumer updates**: `pkg quality`'s subprocess (`cmd/ailang/pkg_quality.go:187`) passes
   `--no-source-check` (it measures the compile itself). The coordinator cascade
   (coordinator_cloud_cascade.go:110-120) may keep or drop its explicit check step — see
   Deferred Decisions.
5. **Docs**: CI pattern in `docs/docs/guides/packages.md:26-29` and package-mode section in
   `docs/docs/guides/testing.md:310`; the strict-fallbacks boundary table
   (`docs/docs/guides/strict-fallbacks.md:76`) gains the `test --package` row;
   `printTestHelp` (test.go:317) and the generated motoko-extension CI checklist
   (`init_motoko_extension_templates.go:319,378`) get the flag and the simplified workflow.

### Implementation Plan

**Phase 1: Core pre-pass** (~3 hours)
- [ ] Add the pre-pass to `runPackageTests` (discover → `CheckPackageFiles` → fail-fast exit 1).
- [ ] Plumb `--no-source-check` through `runTestCommand`/`runPackageTests`.
- [ ] Keep the no-test-files early return AFTER the pre-pass (a package with zero test files but a broken source now fails; green + zero tests keeps today's warn + exit 0).

**Phase 2: Output contracts** (~2 hours)
- [ ] Human rendering of pre-pass results (reuse `check --package`'s per-file/status rendering; prefer extracting the small shared helper over duplicating).
- [ ] `SuiteResult.SourceCheck` field + `source_check` key in `reportJSON`; errors included in both formats; `success:false` + exit 1 on failure.
- [ ] `pkg quality` subprocess passes `--no-source-check`.

**Phase 3: Tests + docs** (~3 hours)
- [ ] E2E tests driving the real binary (follow `test_multi_path_test.go` / `runAilangBin` pattern): repro fixture (exit 1, names `bad.ail`), `--no-source-check` (exit 0), green fixtures unchanged, JSON `source_check` shape, `TestTestCommand_JSONStdoutPackage` still passes.
- [ ] Cache hit-rate measurement on a ≥50-module package (`ailang cache stats` before/after; record numbers in the implementation report).
- [ ] Docs listed above; changelog entry.

### Files to Modify/Create

**New files:**
- `cmd/ailang/test_package_source_check_test.go` (~120 LOC) — e2e: repro exit 1, opt-out exit 0, green regression, JSON shape.

**Modified files:**
- `cmd/ailang/test.go` (+~45 LOC) — pre-pass in `runPackageTests`, `printTestHelp` flag line.
- `cmd/ailang/commands_language.go` (+~6 LOC) — flag parse + dispatch.
- `internal/testing/result.go` (+~10 LOC) — `SourceCheck` field + type.
- `internal/testing/reporter.go` (+~12 LOC) — emit `source_check` when set.
- `cmd/ailang/pkg_quality.go` (+1 LOC) — subprocess flag.
- `cmd/ailang/check_package.go` (±small) — only if the shared human renderer is extracted here.
- `docs/docs/guides/packages.md`, `docs/docs/guides/testing.md`, `docs/docs/guides/strict-fallbacks.md`, `cmd/ailang/init_motoko_extension_templates.go` — docs/templates.

## Examples

### Example 1: The repro, before and after

**Before (verified live):**
```
$ ailang test --package .
→ Package probe/p
  2 source modules, 1 test files
✓ All tests passed!          # 1 passed, exit 0  — bad.ail was never compiled
$ ailang check --package .   # CI must run this too: ✗ bad.ail … exit 1
```

**After:**
```
$ ailang test --package .
→ Package probe/p
  2 source modules (checked), 1 test files
  ✓ good.ail
  ✗ bad.ail
  ✗ 2 files checked: 1 passed, 1 failed            # check --package's rendering
Errors:
  • bad.ail: type error … cannot unify type constructors: string vs int
✗ package source check failed — 1 errors, 0 failed tests, exit 1   # tests NOT run
$ # CI drops the separate `ailang check --package .` step entirely
$ ailang test --package --no-source-check .        # escape hatch: old behavior
```

### Example 2: CI workflow (docs/docs/guides/packages.md)

**Before:**
```
ailang check --package .   # type-check every module
ailang test --package .    # run tests (compiles much of it again)
```

**After:**
```
ailang test --package .    # type-checks every module AND runs tests — one compile
```

## Success Criteria

- [ ] Repro package: `test --package` exits 1 and names `bad.ail`'s type error; with `--no-source-check` it exits 0 as today.
- [ ] Green package (`examples/intra_package_imports`): identical outcome/exit as today (V10 baseline).
- [ ] Pre-pass failure prints `check --package`-shaped per-file lines (human) / `source_check` block (JSON); tests are not executed on failure.
- [ ] JSON stdout remains a single valid JSON document with all existing keys; `pkg quality --json` output unchanged; `TestTestCommand_JSONStdoutPackage` passes.
- [ ] Cache hit-rate measured on a ≥50-module package; before/after wall time recorded in the implementation report (do not claim the dedup without the numbers).
- [ ] `make test`, `make lint`, `make check-boundaries` pass (cmd → internal/check is an existing legal import: check_package.go/pkg_quality.go already do it).
- [ ] Documentation updated (packages.md, testing.md, strict-fallbacks.md, `printTestHelp`, init templates) + changelog entry.

## Conflict Surface

This change touches a **compilation entry point** (`cmd/ailang/test.go` package mode) and its CLI
surface — no parser/lexer/typechecker/codegen change.

### Syntactic positions touched

None. No grammar production, token position, or AST shape changes. The surface is: the
`test` command's flag set, its package-mode control flow, its stdout/stderr contracts, and its
exit codes.

### What else lives here (CLI surface positions)

| Position | Existing valid forms | Impact |
|----------|--------------------|--------|
| `test` flag namespace | `--json`, `--format`, `--no-color`, `--package`, `--allow-skips`, `--seed`, `--random-seed`, `--max-recursion-depth`, `--bytecode`, `--strict-bytecode` (commands_language.go:234-244) | New `--no-source-check` joins the family; `--no-*` prefix follows `--no-color`; name verified unallocated (V7) |
| `--package` takes exactly ONE path (refusal exit 2, test_multi_path_test.go) | Unchanged; pre-pass runs on that one dir only |
| stdout JSON document (suite report; `TestTestCommand_JSONStdoutPackage` asserts pure JSON) | Additive `source_check` key only in package mode; no key removed/renamed; suite keys (`passed_tests` etc.) unchanged on failure paths |
| Exit codes: 0 success, 1 test/check failure, 2 usage error (`--seed`+`--random-seed`, multi-path) | Pre-pass failure → 1 (consistent with `check --package` failure and test failures); `--allow-skips` does NOT cover source-check failure (it is an all-skipped-suite allowance, not a compile allowance) — document this in `--allow-skips` help |
| stderr/stdream split in `--format json` (preamble → stderr) | Pre-pass per-file lines follow the same split as the current preamble (stderr in JSON mode) so stdout stays pure JSON |
| Seed/replay metadata (`cfg.ReplayTarget`) | Unchanged; replay of a failed pre-pass run is `ailang check --package`, not a re-run of tests |
| `--bytecode`/`--strict-bytecode` engine selection | Orthogonal: pre-pass is compile-only, engine choice applies to test bodies as today |
| Walk/discovery semantics | Two walks exist today (test.go:213 suffix-split, check/package.go:228 module-decl-split). The pre-pass must use the *check* discovery so coverage is identical to `check --package` (module-less orphans excluded, `*_test.ail` included, named-test debris skipped, hidden dirs skipped) |

### Disambiguation strategy

No syntactic disambiguation is needed (no parser change). Behavioral disambiguation:
`--no-source-check` is presence-flag only; it composes with every existing flag except that it
suppresses the pre-pass. `--package` + `--no-source-check` = today's exact behavior.

### Programs that MUST still work (regression fixtures)

1. `examples/intra_package_imports` — green package, 0 test files: today warn + exit 0 (V10 baseline); must stay warn + exit 0.
2. The repro package (good/good_test/bad) under `--no-source-check` — byte-comparable to today's output, exit 0.
3. `pkg quality` on any package — subprocess output parsed by `json.Unmarshal` into a 3-field struct (unknown keys ignored, V9); report content unchanged.
4. Coordinator cascade (`coordinator_cloud_cascade.go:110-120`) — runs `check --package` then `test --package`; both steps still pass on green workdirs (the pre-pass is a cache-warmed re-run).
5. `cmd/ailang/test_json_output_test.go` / `test_multi_path_test.go` / `check_package_strict_fallbacks_test.go` — existing contracts must pass unmodified (the strict-fallbacks test asserts `check --package` hard-error behavior; `test --package` now enforces the same, which is the intent).

### What deliberately changes

- `ailang test --package` exit code on packages whose sources don't compile: **0 → 1**, and tests do not run (the point of the change; go test/cargo test precedent).
- stdout JSON gains a `source_check` key (package mode only). Machine consumers that reject unknown keys would break; none exist in-repo (V9, V11) — external strict consumers are flagged in the changelog as intentional.
- `test --package` wall time grows by ≈ one check pass on cold cache; CI that ran both commands nets ≈ zero (and saves a process), CI that ran only tests pays the honesty tax.
- The banner wording may note the check (e.g. "2 source modules (checked)") — deferred.

## Testing Strategy

**Unit tests:**
- `SuiteResult.SourceCheck` marshal/unmarshal round-trip in `internal/testing`.
- Pre-pass wiring: `--no-source-check` suppresses the pre-pass; default runs it.

**Integration tests (real binary, `runAilangBin` pattern):**
- Repro fixture (good + good_test + bad): default → exit 1, error names `bad.ail`; opt-out → exit 0; `--format json` → single JSON document, `source_check.failed == 1`, `success == false`, exit 1.
- Green package unchanged (exit 0, no `source_check` surprises in parsed keys).
- Zero-test-file package with broken source: exit 1 (new), with green source: warn + exit 0 (unchanged).
- `pkg quality --json` before/after on a fixture package: identical report bytes.

**Regression-surface tests:**
- One test per fixture in "Programs that MUST still work" (the strict-fallbacks package fixture doubles as the publish-boundary case: an `Ok(jo([]))` default in a Result function must now fail `test --package` exactly as it fails `check --package`).

**Manual testing:**
- Cache hit rate on a ≥50-module package: `ailang cache stats` before/after the pre-pass; measure `test --package` wall time cold vs. warm; record in the implementation report (2-core-shaped runner if available).
- `ailang test --help` shows the new flag; docs render.

## Deferred Decisions

The following are intentionally left open for the implementer:

- Exact banner wording ("source modules (checked)" vs separate line) — agent may choose.
- Whether the shared human renderer is extracted from `check_package.go` or re-emitted in `test.go` — agent may choose (prefer extraction if the diff stays small).
- Whether the coordinator cascade drops its explicit `check --package` step (it becomes redundant; keeping it is harmless and attributes errors to the check step) — human at review.
- Whether the pre-pass honors a future `--timeout` spelling on `test` (today unbounded, matching `check --package` default) — agent; do not add a `--timeout` flag now (flag-family collision with `check --timeout`'s different meaning is documented at commands_language.go:381-388).
- Changelog wording for the intentional 0→1 exit-code change — agent, with human sign-off at release.

## Non-Goals

**Not attempted in this feature:**

- Dir/file mode (`ailang test .`, `ailang test a.ail`) source checking — different discovery model (no manifest, per-file semantics); audited here (V3), needs its own design if wanted.
- Any change to `ailang check` / `check --package` behavior — untouched.
- Removing the `check --package` command — it stays the standalone type-check entry point.
- New type-checker analyses, new diagnostics codes, or cross-process artifact sharing beyond the existing on-disk cache.
- Parallelizing the pre-pass across files (single-threaded, sorted, deterministic — as `check --package` is today).

## Timeline

**Week 1** (~8 hours):
- Phase 1: pre-pass + flag (~3h)
- Phase 2: output contracts (~2h)
- Phase 3: e2e tests + docs + changelog (~3h)

**Total: ~1–1.5 days** (with buffer).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Default-on changes exit codes for external CI that relied on the loose behavior | High | `--no-source-check` escape hatch; loud changelog entry; in-repo consumers audited (V9, V13: none break) |
| Wall-time regression on 2-core runners (pre-pass ≈ one check pass, cold cache) | Medium | Net-zero-or-better for CI that ran both (the reported consumer); cache reuse expected (V8) — measured, not assumed, in-sprint |
| Double compile inside `pkg quality` (in-process check + subprocess pre-pass) | Low | `pkg quality` passes `--no-source-check` |
| JSON strictness of external parsers vs the new key | Low | Additive key only; documented as intentional in changelog |
| The banner/exit semantics of the zero-test-files early return shift subtly | Low | Pinned by new e2e tests (broken source + no tests → exit 1; green + no tests → warn + exit 0) |

## Related Documents

**Planned (check for overlap):**
- `design_docs/planned/v0_54_12/examples-manifest-stat-sync.md` — same command family (package manifest stat reporting); check its banner-sync decisions for wording consistency. No overlap: it syncs example stats, this doc adds a source-check gate.
- `design_docs/implemented/v0_54_13/m-test-strip-brace-literals.md` — same `ailang test` area; orthogonal (named-test body stripping), no overlap with source-module discovery.

(Duplicate/coverage gate: SimHash + neural search over implemented/ and planned/ returned no match ≥ 0.45 for "test package source check" — V14. This doc's topic — package-mode source-check subsumption — is distinct from test-body compilation issues.)

## Verification Log

Every load-bearing claim above, with its evidence. Binary: `/usr/local/bin/ailang` v0.54.11-dirty
(commit 35a49b8; not in this shallow checkout's history — the source tree at HEAD v0.54.12 was read
directly for every code claim, so each V-row cites file:line at HEAD).

| # | Claim | Evidence |
|---|-------|----------|
| V1 | `test --package .` on the repro exits 0 (1 passed) while `bad.ail` has a type error | Live run: "2 source modules, 1 test files … All tests passed! exit 0" |
| V2 | `check --package .` on the same package fails: `✗ bad.ail: … cannot unify type constructors: string vs int`, exit 1 | Live run (same binary, same dir) |
| V3 | Dir mode `ailang test .` also exits 0 on the same dir, **no warning at all** for `bad.ail` | Live run: "1 tests: 1 passed … exit 0", stderr empty |
| V4 | `runPackageTests` counts `sourceFiles` in the banner but never compiles them | Read cmd/ailang/test.go:200-260 — the walk appends to `sourceFiles`; only `testFiles` reach `runTestFile` |
| V5 | `check --package` compiles every module-decl file (incl. `*_test.ail`) via the exported `DiscoverPackageSources` + `CheckPackageFiles` (with InterFunctionRefs + StrictFallbacks) | Read internal/check/package.go:75-228; cmd/ailang/check_package.go:74-115 |
| V6 | `pkg quality` already measures the compile in-process with the same `CheckPackageFiles` call | Read cmd/ailang/pkg_quality.go:103-109 |
| V7 | Flag names `--no-source-check` / `--check-all` are unallocated across cmd/, internal/, docs/ | `grep -rn "no-source-check\|check-all" cmd/ internal/ docs/` → empty |
| V8 | Compile cache is default-on and its identity (`compilerIdentity`) keys on commit + dirty-fingerprint + release — not pipeline Mode — so pre-pass entries are reusable by the test pass's import resolution | Read internal/pipeline/pipeline_module_phases.go:291-306 (`newPipelineModuleCache`, nil only when NoCache), internal/pipeline/pipeline_module_cache.go:77-88 |
| V9 | `pkg quality`'s subprocess parser tolerates new JSON keys | Read cmd/ailang/pkg_quality.go:181-200 — `json.Unmarshal(out, &summary)` into a 3-field anonymous struct; no `DisallowUnknownFields` |
| V10 | `examples/intra_package_imports` is a green 0-test package: "2 source modules, 0 test files" warn + exit 0 | Live run |
| V11 | Package-mode JSON stdout is asserted to be a single pure-JSON document; suite keys are fixed | Read cmd/ailang/test_json_output_test.go:131-160; internal/testing/reporter.go:48-74 |
| V12 | Empty suite ⇒ `success:false` ⇒ exit 1 (pre-pass failure path will render a failing suite, not a green one) | Read internal/testing/result.go:130-133 (`ran > 0 && FailedTests == 0 && VacuousSkips == 0`); live `ailang test good.ail` → exit 1 |
| V13 | No in-repo eval/test/script consumer subprocesses `test --package` besides `pkg quality` and the cascade | `grep -rn "test.*--package" eval_projects/ tests/ scripts/ bench/` → empty (hits only in cmd/ + docs) |
| V14 | No existing/planned design doc covers this topic | create_planned_doc.sh dual search (SimHash + neural) over implemented/ and planned/: no match ≥ 0.45 |
| V15 | `std/` is not a manifest package (no `ailang.toml`), so `test --package std` is not an affected surface | `ls std/ailang.toml` → not found |
| V16 | Coordinator cascade runs `check --package .` then `test --package .` as separate CI steps | Read cmd/ailang/coordinator_cloud_cascade.go:110-120 |

## References

- **Reporter**: stapledons-godot CI (task directive, AILANG v0.54.8): the 2.0–3.6 min check / ~200 s of 13 min test-wall numbers are *their* measurements, quoted as reported.
- **Ecosystem precedent**: `go test ./...` and `cargo test` compile every package unit before running tests; test failure to compile ⇒ no tests run, exit ≠ 0.
- **Prior art in-repo**: `check --package` semantics (docs/docs/guides/strict-fallbacks.md:76), M-PKG-INTERREF, M-CHECK-STRICT-FALLBACKS.
- **Cache**: M-PERF6 (compile cache), M-COMPILE-CACHE-DIRTY-BUILD-KEY (#1275, #1572).
- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- [Philosophical Foundations](/docs/references/philosophical-foundations) - Block-universe determinism
- [Design Lineage](/docs/references/design-lineage) - What we adopted/rejected and why

## Future Work

- Dir-mode `ailang test <dir>`: decide whether test-less files with compile errors should warn-and-skip (V3) — needs its own design; a `--check-sources` default there has no manifest to define "the package".
- Cross-pass artifact sharing (pre-pass → test-runner without disk round-trip) if the measured cache hit rate disappoints.
- Coordinator cascade simplification (drop its explicit check step once `test --package` carries the gate).

---

**Document created**: 2026-10-16
**Last updated**: 2026-10-16