# Sprint Plan: M-BYTECODE-PATTERN-ARITY-FIX

**Design doc**: [m-bytecode-pattern-arity-fix.md](m-bytecode-pattern-arity-fix.md) (quorum-resolved 2026-08-12, committed `8a859d6a2`)
**Issue**: [#505](https://github.com/sunholo-data/ailang/issues/505)
**Base**: `origin/dev` @ `8ecebc0e1`; worktree branch `sprint/iter187-bytecode-pattern-arity`
**Planner verification**: all rows below measured live in this worktree on 2026-08-12 (see §7).

## Summary

One-branch change in `internal/gen/lower/match.go:lowerPatternCond`: a `core.ListPattern` with
`Tail == nil` must lower its length guard to `OpEq`, not `OpGte`. The root cause is already pinned
and mutation-proven by the design doc (V-C, V-E, V-I); this sprint does **not** re-derive it. The
work is almost entirely *test* work — building guards that have teeth, and proving they have teeth.

**Duration**: 1 day (3 milestones, ~205 LOC of which only ~6 are production)
**Dependencies**: none. Explicitly NOT blocked on the parent doc's parked A1/A2/B3/B4.
**Risk Level**: Low for the change itself; **Medium for test vacuity** — the two ACs most likely to
ship as decoration are AC3 (its stderr clause is already true on the broken binary) and AC2 (its
fixture can be silently bridged to the evaluator and pass for the wrong reason). Both are handled
by an explicit red-first gate in M1 and a mutation drill in M3.

## 1. The change

`internal/gen/lower/match.go`, `lowerPatternCond` (declared line 390), `case *core.ListPattern`
(line 410), length guard at lines 412-427.

Current (verified live; file sha256 `fd1d890e7d3e1ed58211c0fec4eab41640aad7d8274ca31b09d25b6aa87500a2`):

```go
var cond stmt.Expr
if len(p.Elements) == 0 && p.Tail == nil {
    cond = stmt.BinOp{Op: stmt.OpEq,  Left: lenExpr, Right: stmt.LitInt{Value: 0}}
} else {
    cond = stmt.BinOp{Op: stmt.OpGte, Left: lenExpr, Right: stmt.LitInt{Value: int64(len(p.Elements))}}
}
```

Target — collapse both branches into one operator selection driven by `p.Tail`:

```go
var cond stmt.Expr
lenOp := stmt.OpGte
if p.Tail == nil {
    lenOp = stmt.OpEq
}
cond = stmt.BinOp{
    Op:    lenOp,
    Left:  lenExpr,
    Right: stmt.LitInt{Value: int64(len(p.Elements))},
}
```

**Why the collapse is safe rather than merely equivalent-looking** — the four `(len(Elements), Tail)`
quadrants, exhaustively:

| case | before | after | verdict |
|---|---|---|---|
| `[]` — 0 elems, `Tail == nil` | `len == 0` | `len == 0` | identical |
| `[a,b]` — n elems, `Tail == nil` | `len >= n` **(the bug)** | `len == n` | **fixed** |
| `[a, ...rest]` — n elems, `Tail != nil` | `len >= n` | `len >= n` | unchanged |
| `[...rest]` — 0 elems, `Tail != nil` | `len >= 0` (always true) | `len >= 0` (always true) | identical |

So the empty-list special case at line 414 is *subsumed*, not deleted: it is exactly the
`Tail == nil, n = 0` instance of the general rule. The `Tail == nil ⟺ exactly-n` invariant is
established over the whole codebase, not just the struct, by V-I (exactly two construction sites,
both in `internal/elaborate/patterns.go`).

Expected diff: **~6 net LOC** in one file. Nothing in `internal/vm` or `internal/bytecode` is
touched (V-D: zero `Pattern` occurrences there).

## 2. Where the tests live — CORRECTED from the design doc

The design doc says: *"Commit a minimal failing `.ail` repro under `tests/golden/bytecode/`."*
I verified this before prescribing it, and it needs two corrections.

**Correction A — `tests/golden/bytecode/` is a Go test package, not a fixture directory.**
It contains exactly four files, all `.go`, all `package bytecode_golden_test`:
`golden_test.go`, `probe_test.go`, `bench_test.go`, `io_test.go`. Its `.ail` fixtures live one
directory over in `tests/golden/codegen/`, loaded **by name** from a hard-coded `goldenSpecs()`
list (`tryCompileAILFile` → `filepath.Join("..", "codegen", name)`). There is no `Glob`/`WalkDir`
over either directory, so dropping a `.ail` file in will not be auto-discovered by anything.

**Correction B — AC1 and AC3 are not implementable in that package at all.**
`tests/golden/bytecode` drives `pipeline.Run` → `lower.LowerProgram` → `compiler.Compile` →
`vm.NewVM` **in-process**. It never runs the CLI, so it cannot observe:
- the `falling back to evaluator` stderr line that AC3 asserts on (emitted at
  `cmd/ailang/run_helpers.go:376`, i.e. in the CLI run path, not in the VM);
- `--caps IO` capability grants;
- process exit code.

**Chosen homes:**

| artifact | path | rationale |
|---|---|---|
| the `.ail` repro | `tests/golden/bytecode/pattern_arity.ail` | honours the doc's stated location; verified inert w.r.t. the Go package (`go test ./tests/golden/bytecode/` stays `ok` with the file present) |
| AC1, AC2, AC3 tests | `cmd/ailang/run_bytecode_test.go` (append) | the only existing harness that shells out to the real CLI |

`cmd/ailang/run_bytecode_test.go` is exactly the right host: it is 164 LOC, has five passing
`TestCLI_RunBytecode_*` tests, drives the CLI through `runCLI` →
`testutil.RunBounded(t, projectRoot, 120s, "go", "run", "./cmd/ailang", …)` with cwd = repo root,
and **already contains the precise AC3 assertion** at line 63:

```go
if strings.Contains(stderr, "falling back to evaluator") {
```

**Wiring verdict: already gated, no new target or job needed.** This is the point the design doc's
"golden file that no target runs is decoration" worry is aimed at, and here it does not bite —
because these are Go tests, not `ailang test` `.ail` suites. `.github/workflows/ci.yml` runs
`go test -timeout 300s ./...`, which covers `./cmd/ailang`, `./tests/golden/bytecode` and
`./internal/gen/...`. Measured at base: `go test -count=1 -run 'TestCLI_RunBytecode' ./cmd/ailang`
→ 5/5 PASS in 6.35s; `go test ./tests/golden/bytecode/` → ok. Contrast with iteration 183's gap,
which was specific to `ailang test`-driven `.ail` suites having no runner — not the case here.

**Do NOT put the repro in `examples/runnable/`.** `scripts/verify_bytecode_parity.go:85` globs
`examples/runnable/*.ail`, so a fixture there would silently mutate the very corpus we use as the
blast-radius control, and would also enter the `make verify-examples` manifest.

## 3. The AC2 fixture (verified red at HEAD, verified green under the probe)

`tests/golden/bytecode/pattern_arity.ail`. Shaped after V-H — each fixed arity is tested in an
**isolated** function so no shorter arm can intercept and hide it (the arm-ordering artifact the
round-2 quorum correctly refused). Module decl matches the canonical path, so **no
`--relax-modules` is needed** (measured).

```ailang
module tests/golden/bytecode/pattern_arity

import std/io (println)

// n=1 fixed arm, isolated: only the tailed fallback can catch anything else.
func arity1(xs: [int]) -> string =
  match xs {
    [] => "other",
    [a] => "n1",
    [p, ...rest] => "other"
  }

// n=2 fixed arm, isolated.
func arity2(xs: [int]) -> string =
  match xs {
    [] => "other",
    [a, b] => "n2",
    [p, ...rest] => "other"
  }

// n=3 fixed arm, isolated.
func arity3(xs: [int]) -> string =
  match xs {
    [] => "other",
    [a, b, c] => "n3",
    [p, ...rest] => "other"
  }

// Tailed pattern MUST keep "at least 2" semantics.
func tail2(xs: [int]) -> string =
  match xs {
    [a, b, ...rest] => "tail2",
    _ => "other"
  }

// Cons form: per V-I always Tail != nil, so "at least 1" must survive.
func consForm(xs: [int]) -> string =
  match xs {
    ::(h, t) => "cons",
    _ => "other"
  }

export func main() -> () ! {IO} {
  println("arity1 []      = ${arity1([])}");
  println("arity1 [7]     = ${arity1([7])}");
  println("arity1 [7,8]   = ${arity1([7, 8])}");
  println("arity2 [7]     = ${arity2([7])}");
  println("arity2 [7,8]   = ${arity2([7, 8])}");
  println("arity2 [7,8,9] = ${arity2([7, 8, 9])}");
  println("arity3 [7,8]   = ${arity3([7, 8])}");
  println("arity3 [7,8,9] = ${arity3([7, 8, 9])}");
  println("arity3 [7,8,9,10] = ${arity3([7, 8, 9, 10])}");
  println("tail2  [7]     = ${tail2([7])}");
  println("tail2  [7,8]   = ${tail2([7, 8])}");
  println("tail2  [7,8,9] = ${tail2([7, 8, 9])}");
  println("cons   []      = ${consForm([])}");
  println("cons   [7]     = ${consForm([7])}");
  println("cons   [7,8]   = ${consForm([7, 8])}")
}
```

Measured behaviour (planner, live). `→`/`✓` status lines stripped:

| row | direction | evaluator (control) | `--bytecode` @ HEAD | `--bytecode` @ probe |
|---|---|---|---|---|
| `arity1 []` | exact n=0 | other | other | other |
| `arity1 [7]` | exact n=1 | n1 | n1 | n1 |
| **`arity1 [7,8]`** | **overflow n=1** | other | **n1 ✗** | other ✓ |
| `arity2 [7]` | **underflow n=2** | other | other | other |
| `arity2 [7,8]` | exact n=2 | n2 | n2 | n2 |
| **`arity2 [7,8,9]`** | **overflow n=2** | other | **n2 ✗** | other ✓ |
| `arity3 [7,8]` | **underflow n=3** | other | other | other |
| `arity3 [7,8,9]` | exact n=3 | n3 | n3 | n3 |
| **`arity3 [7,8,9,10]`** | **overflow n=3** | other | **n3 ✗** | other ✓ |
| `tail2 [7]` | tailed, below | other | other | other |
| `tail2 [7,8]` | tailed, at | tail2 | tail2 | tail2 |
| `tail2 [7,8,9]` | **tailed, above — must stay ≥** | tail2 | tail2 | tail2 |
| `cons []` | cons, empty | other | other | other |
| `cons [7]` | cons, at | cons | cons | cons |
| `cons [7,8]` | **cons, above — must stay ≥** | cons | cons | cons |

Three rows diverge at HEAD and only those three; the fix flips exactly those three and nothing
else. Crucially the divergence **also proves these functions really execute in the VM** rather
than being bridged back to the evaluator — a fixture that were silently bridged would show zero
divergence and pass at HEAD, i.e. be vacuous.

## 4. Milestones

### M1 — Red first: land the fixture and AC2 **against the unfixed lowering** (~0.25d)

**Goal**: prove the guard has teeth *before* the thing it guards exists. No production code changes
in this milestone.

**Deliverables**
- `tests/golden/bytecode/pattern_arity.ail` exactly as §3.
- `TestCLI_ListPatternArity_Bytecode` appended to `cmd/ailang/run_bytecode_test.go` (~70 LOC).
  Shape — a **two-clause** assertion, both clauses required:
  1. **cross-engine parity**: run the fixture under `ailang run --caps IO` and under
     `ailang run --bytecode --caps IO`; stdout must be byte-identical.
  2. **absolute golden**: bytecode stdout must equal the 15 expected lines verbatim (table §3,
     evaluator column). Parity alone is satisfiable by breaking *both* engines identically;
     the absolute golden is what stops that.
  Also assert `exitCode == 0` and `strings.Contains(vmStderr, "via bytecode VM")` — the latter is
  the anti-bridging sanity check that the run actually took the VM path.

**Acceptance (this milestone is DONE when the test is RED)**
```
go test -count=1 -run 'TestCLI_ListPatternArity_Bytecode' ./cmd/ailang
```
must exit **non-zero**, and the failure output must name **exactly three** wrong rows:
`arity1 [7,8]`, `arity2 [7,8,9]`, `arity3 [7,8,9,10]`. Fewer than three ⇒ the fixture is being
bridged or an arm is intercepting; more than three ⇒ the expected table is wrong. **Record the
failure output in the commit message.** If it is green at HEAD, STOP — the test is vacuous and
must be reworked, not proceeded past.

**Risk**: fixture gets silently bridged to the evaluator and passes at HEAD.
**Mitigation**: that is precisely what the red-first gate detects; it cannot be skipped.

---

### M2 — The fix, plus AC1 and AC3 (~0.25d)

**Goal**: land the one-branch change and the flagship-symptom goldens.

**Deliverables**
- `internal/gen/lower/match.go` — the §1 change (~6 net LOC).
- `TestCLI_RunBytecode_QuicksortArity` appended to `cmd/ailang/run_bytecode_test.go` (~45 LOC),
  covering **AC1 and AC3 in one test** (they are the same run; splitting them would double a
  ~4s `go run` for no signal):
  - **AC1**: `ailang run --bytecode --caps IO examples/runnable/recursion_quicksort.ail` stdout
    contains `Quicksort: [1, 1, 2, 3, 4, 5, 6, 9]` **and** `sortBy:    [1, 1, 2, 3, 4, 5, 6, 9]`
    (note: three spaces after `sortBy:` — copy the literal from the doc, it is column-aligned).
  - **AC3**: same run, `exitCode == 0`, `!strings.Contains(stderr, "falling back to evaluator")`,
    and `strings.Contains(stderr, "via bytecode VM")`.
  - Do **not** pass `--relax-modules`: the example resolves canonically (measured).

**Acceptance**
```
go test -count=1 -run 'TestCLI_RunBytecode_QuicksortArity|TestCLI_ListPatternArity_Bytecode' ./cmd/ailang
go test -count=1 ./internal/gen/...
```
first rc=0 (M1's test must now be **green** — the same binary, the same command that was red);
second rc=0 (base measured rc=0, 5 packages).

**⚠ AC3's stderr clause is decorative on its own — measured.** At HEAD the quicksort run prints the
*wrong* answer with `rc=0` and stderr containing only `✓ Running … via bytecode VM`; there is **no**
`falling back to evaluator` line. So `!Contains(stderr, "falling back")` is already true on the
broken binary — it is set alongside the mechanism, not by it. It earns its place only as a
conjunction with the stdout assertion (it rules out a "fix" that got the right answer by bailing
to the evaluator), and it needs its own mutation (D4, §5) to prove it is wired to anything at all.

**Risk**: over-correcting into the underflow direction, or breaking tailed patterns.
**Mitigation**: M1's `arity2 [7]` / `arity3 [7,8]` rows are the underflow arm; `tail2 [7,8,9]` and
`cons [7,8]` are the tailed arm. Both directions are already in the fixture and both were measured
unchanged under the probe.

---

### M3 — Prove the guards have teeth; blast radius; gates; changelog (~0.5d)

**Goal**: mutation drills, whole-corpus regression check, full local gate sweep, docs.

**Deliverables**
- Mutation drill table (§5) executed, with before/after sha256 + build rc recorded per row.
- Blast-radius results (§6) recorded.
- Changelog entry in **`changelogs/v0.18-current.md`** under `## [Unreleased]` → `### Fixed`.
  **NOT** root `CHANGELOG.md` — `make check-changelog` (`scripts/check_changelog.sh`) fails on any
  `### Fixed` heading there; root is an index only.
- `docs/LIMITATIONS.md`: if it carries a "fixed-length list patterns under `--bytecode`" caveat,
  remove it; if not, add nothing. (Check, don't assume.)

**Acceptance**: the §7 gate sweep, all rc=0.

## 5. Non-vacuity drill — one mutation per test-plan row

**Protocol for every row.** Prefer neutering over deletion so imports stay used.
1. `shasum -a 256 <file>` → record **before**.
2. Apply the mutation.
3. `shasum -a 256 <file>` → record **after**; assert it **differs**. A mutation that did not land
   makes the subsequent red meaningless.
4. Assert it **builds** — see the build-command correction immediately below.
5. Run the row's command; assert it goes **red**, and that the red names the expected rows.
6. Restore from backup; `shasum -a 256` must equal the **before** value.

**⚠ CORRECTION to the prescribed drill command — `go build ./...` is RED AT BASE.**
Measured in this worktree on the pristine tree:
```
$ go build ./...
# github.com/sunholo-data/ailang/cmd/wasm
runtime.main_main·f: function main is undeclared in the main package
```
`cmd/wasm` only builds under `GOOS=js GOARCH=wasm`, so `go build ./...` never returns rc=0 on
darwin/arm64 — it would report every mutation as "failed to build" and the drill would be
uninterpretable (rule 3e: a command already red at base measures the repo, not the change).
**Use instead** (measured rc=0 at base):
```
go build ./internal/... ./cmd/ailang
```

| # | row it kills | mutation | file | expected red |
|---|---|---|---|---|
| D1 | **AC2 overflow** (the fix itself) | revert §1: force `lenOp := stmt.OpGte` unconditionally, i.e. `if false && p.Tail == nil {` | `internal/gen/lower/match.go` | `TestCLI_ListPatternArity_Bytecode` fails on exactly `arity1 [7,8]`, `arity2 [7,8,9]`, `arity3 [7,8,9,10]` |
| D2 | **AC2 tailed arm** (over-correction guard) | invert the discriminator: `lenOp := stmt.OpEq` unconditionally (i.e. `OpEq` even when `Tail != nil`) | `internal/gen/lower/match.go` | same test fails on `tail2 [7,8,9]` and `cons [7,8]` (they become `other`). **This is the row that proves the tailed arm is not decoration** — under D1 those rows stay green, so only D2 exercises them. |
| D3 | **AC1** | in the fixture-independent path: change the AC1 expectation source — mutate `internal/gen/lower/match.go` as D1 | `internal/gen/lower/match.go` | `TestCLI_RunBytecode_QuicksortArity` fails with `Quicksort: [3]` |
| D4 | **AC3 stderr clause** (the decorative one) | force the top-level fallback: at `cmd/ailang/run_helpers.go:376`, neuter the guard so the fallback branch always runs (`if true || vmErr != nil {`) | `cmd/ailang/run_helpers.go` | `TestCLI_RunBytecode_QuicksortArity` fails **on the stderr assertion specifically** (`falling back to evaluator` present). If it fails only on stdout, the stderr clause is unwired — fix the test, not the mutation. |
| D5 | **AC2 absolute-golden clause** | in the test, corrupt one expected line (e.g. `n2` → `nX`) — proves the golden is compared, not merely computed | `cmd/ailang/run_bytecode_test.go` | test fails naming that row |
| D6 | **AC2 parity clause** | in the test, replace the evaluator-run stdout with the bytecode stdout (`evalStdout := vmStdout`) — proves the control is a real second measurement | `cmd/ailang/run_bytecode_test.go` | parity clause can no longer fail ⇒ re-run D1: if the test still reds it is because of the golden clause only. **Record which clause fired.** |

**Rows deliberately NOT drilled**: the blast-radius examples and the parity harness (§6). They are
regression *observations*, not gates — the doc explicitly declines to own the harness's bucket
definitions. Do not manufacture a mutation for them.

## 6. Blast radius (regression check, NOT a gate)

**Named must-still-work programs.** All four are MATCH at HEAD **and** MATCH under the probe, with
identical hashes — measured, so this is a genuine before/after control, not a restatement of V-F:

| example | eval sha256[0:12] | bytecode sha256[0:12] | HEAD | probe |
|---|---|---|---|---|
| `cons_expression` | `2230a604cb25` | `2230a604cb25` | MATCH | MATCH |
| `block_recursion` | `d4295f6e4c13` | `d4295f6e4c13` | MATCH | MATCH |
| `adt_list_fields` | `5430d1a50e8f` | `5430d1a50e8f` | MATCH | MATCH |
| `effectful_list` | `69fd90893805` | `69fd90893805` | MATCH | MATCH |

I added three the doc does not name but which are the most list-pattern-dense examples in the
corpus — all MATCH at HEAD and under the probe: `list_pattern_cons` (`3d9a0ed566bd`),
`list_patterns` (`ec1c27cb21d0`), `list_pattern_records` (`316ab9a842c6`). Include them.

Command (`md5` is absent on this machine — `shasum -a 256`):
```
for f in cons_expression block_recursion adt_list_fields effectful_list \
         list_pattern_cons list_patterns list_pattern_records; do
  e=$(./bin/ailang run --caps IO examples/runnable/$f.ail 2>/dev/null | tail -20 | shasum -a 256 | cut -c1-12)
  b=$(./bin/ailang run --bytecode --caps IO examples/runnable/$f.ail 2>/dev/null | tail -20 | shasum -a 256 | cut -c1-12)
  [ "$e" = "$b" ] && s=MATCH || s=DIVERGE
  echo "$f  eval=$e  bc=$b  $s"
done
```
All seven must read MATCH after the change. (`--caps IO` is mandatory: without it the run errors
with `effect 'IO' requires capability` and the bytecode path falls back to the evaluator, which
masks the bug entirely.)

**Whole-corpus parity harness.** `go run ./scripts/verify_bytecode_parity.go`, before/after.
Measured by the planner:

| bucket | BASE (HEAD) | PROBE | delta |
|---|---|---|---|
| MATCH | 151 (86.3%) | **152 (86.9%)** | **+1** |
| NON_DET | 2 | 2 | 0 |
| DIVERGE | 6 (3.4%) | **5 (2.9%)** | **−1** |
| EVAL_SKIP | 16 | 16 | 0 |
| total | 175 | 175 | — |

The delta is **exactly one file**: `recursion_quicksort.ail` leaves DIVERGE. Nothing else moves in
either direction. Residual DIVERGE after the fix, all out of scope:
- `array_basic.ail` — closure-dispatch family, parent doc B1/B3.
- `pattern_sugar.ail` — the parent doc's **A1**. Worth stating precisely because it looks adjacent:
  the divergence is `firstPair([("a",1),...])`, where the **evaluator** prints `<*eval.TupleValue>`
  and the **VM prints the correct `(a, 1)`**. The evaluator is the wrong one. This fix neither
  helps nor hurts it.
- `claude_haiku_call.ail` (network), `tar_gzip_reader.ail` (missing `/tmp/tar_smoke` fixture),
  `xml_walk_perf.ail` (wall-clock `time_ms` in output).

**The harness is not byte-stable across runs** (the last two rows embed timings/IO state), so
compare **bucket counts and the DIVERGE name set**, never raw output diffs. Treat any change other
than `recursion_quicksort.ail` leaving DIVERGE as a finding to report — not as a build failure.

## 7. Local gate sweep before pushing

Derived from `.github/workflows/ci.yml` (the `test` job's steps and the lint job), filtered to what
a diff touching `internal/gen/`, `cmd/ailang/` and `tests/` can plausibly break. Note CI has **no
push paths filter**, so the full suite runs regardless.

All of the following were measured rc=0 on the pristine tree, so a red is attributable to the
change (rule 3e):

```
go test -timeout 300s ./...
go test -count=1 ./internal/gen/...
go test -count=1 ./tests/golden/bytecode/
go test -count=1 -run 'TestCLI_RunBytecode|TestCLI_ListPatternArity' -v ./cmd/ailang
make test-lowering
make test-stdlib-ail
make check-file-sizes
make check-boundaries
make check-changelog
make check-golden-drift
make test-regression-guards
make verify-examples
make fmt-check
make vet
make lint
```

Notes:
- `make check-golden-drift` only diffs `internal/parser/testdata/parser/` — adding files under
  `tests/golden/` does **not** trip it (verified by reading the target).
- `make check-file-sizes` (800-line limit): `internal/gen/lower/match.go` is 566 lines and the
  change is net **−4**; `cmd/ailang/run_bytecode_test.go` is 164 and grows to ~280. Both clear.
- `make check-changelog` is the one that has bitten this loop before. Entry goes in
  `changelogs/v0.18-current.md`, never root `CHANGELOG.md`.
- `make verify-examples` runs the **evaluator** path, which this change does not touch; expect it
  unchanged. Run it anyway — it is the examples job's real gate.
- Do **not** add `go build ./...` — red at base (`cmd/wasm`, §5).

## 8. Success metrics

- [ ] AC1 green: quicksort under `--bytecode --caps IO` prints both correct lines.
- [ ] AC2 green: all 15 fixture rows match the evaluator **and** the absolute golden, for n=1,2,3,
      in the overflow, exact **and** underflow directions, plus the tailed and cons arms.
- [ ] AC3 green: rc=0, stderr free of `falling back to evaluator`, stderr contains `via bytecode VM`.
- [ ] M1's red-first failure output recorded in the commit message (exactly three wrong rows).
- [ ] Six mutation drills D1-D6 executed; each recorded with before/after sha256, build rc, and the
      observed red.
- [ ] All seven blast-radius examples MATCH.
- [ ] Parity harness delta is exactly `recursion_quicksort.ail` DIVERGE → MATCH.
- [ ] Full §7 gate sweep rc=0.
- [ ] Changelog entry in `changelogs/v0.18-current.md`.

## 9. Scope boundaries (do not cross)

Out of scope, per the design doc and Mark's option-C carve-out:
- Closure-dispatch family (`array_basic`, `array_grid`, `module_let_helpers`) — parent B1/B3.
- Parity-harness classification scheme (A1 eval tuple-show, A2 harness honesty) — parked.
- Unsafe-effect-replay fallback policy (B4) — parked with A2.
- M-BYTECODE-2E bridge gaps (Result.Ok/Err TaggedValue, MapValue, BytesValue).
- **Nested tail sub-patterns.** `lowerPatternCond` does **not** recurse into `p.Tail`, so a pattern
  like `::(x, [])` still lowers to `len >= 1` after this fix and does not enforce its inner
  fixed-length tail. This is a real residual, it is **not** a regression introduced here, and it is
  not in AC1-AC3. Flag it for a follow-up doc; do not fix it in this sprint.

## 10. Open questions / discrepancies found

| id | design-doc claim | finding | resolution |
|---|---|---|---|
| **D-A** | "Commit a minimal failing `.ail` repro under `tests/golden/bytecode/`" implies a fixture directory with a runner | That path is a **Go test package** (`package bytecode_golden_test`, 4 `.go` files); its `.ail` fixtures live in `tests/golden/codegen/` and are named in a hard-coded list. The package is in-process and **cannot** observe CLI stderr or `--caps IO`, so AC1/AC3 are unimplementable there. | `.ail` fixture at `tests/golden/bytecode/pattern_arity.ail` (honours the stated location, verified inert); **all three ACs** as CLI tests in `cmd/ailang/run_bytecode_test.go`. Already gated by `go test ./...` in CI — no new make target or CI job required. |
| **D-B** | AC3: "no fallback (stderr asserted free of `falling back to evaluator`)" reads as an independent observable | **Already true at HEAD on the broken binary** — quicksort prints `[3]`, rc=0, stderr has only `✓ Running … via bytecode VM`. The clause is set alongside the mechanism, not by it. | Keep it (it rules out a fix that bails to the evaluator), but only as a conjunction with the stdout assertion, and drill it separately (D4). |
| **D-C** | Conflict Surface names four must-still-work programs | Correct, and all four MATCH at HEAD as well as under the probe. But the three most list-pattern-dense examples (`list_pattern_cons`, `list_patterns`, `list_pattern_records`) are unnamed. | Added to the blast-radius set; all MATCH before and after. |
| **D-D** | (not addressed) `lowerPatternCond`'s handling of `p.Tail` sub-patterns | The function never recurses into `p.Tail`, so `::(x, [])` remains `len >= 1` after the fix. | Documented as an explicit residual in §9; follow-up doc, not this sprint. |
| **D-E** | (controller-prescribed drill command) `go build ./...` rc=0 | **Red at base** on darwin/arm64 — `cmd/wasm` needs `GOOS=js GOARCH=wasm`. | Drills use `go build ./internal/... ./cmd/ailang` (rc=0 at base). |
| **D-F** | Parity harness as before/after regression check | Sound, but the harness is **not byte-stable**: `xml_walk_perf.ail` embeds `time_ms`, `tar_gzip_reader.ail` depends on a `/tmp` fixture, `claude_haiku_call.ail` hits the network. | Compare bucket counts + DIVERGE name set only. Expected delta measured: `recursion_quicksort.ail` DIVERGE → MATCH, nothing else. |

## 11. Planner verification log

All measured in `/Users/voightkampff/dev/sunholo-data/.wt-iter187` on 2026-08-12, base `8a859d6a2`
(= design doc commit on top of `8ecebc0e1`). The probe was applied and restored; final
`shasum -a 256 internal/gen/lower/match.go` = `fd1d890e7d3e1ed58211c0fec4eab41640aad7d8274ca31b09d25b6aa87500a2`
(identical to the doc's V-E restore hash) and `git status --short` / `git diff --stat` are both empty.

| ID | check | result |
|---|---|---|
| P-1 | `sed -n '386,432p' internal/gen/lower/match.go`; `sed -n '365,380p' internal/core/core.go` | Confirms every controller-VERIFIED fact: `lowerPatternCond` at 390, `case *core.ListPattern` at 410, `OpEq` empty special-case at 414, `OpGte` at 421-427; `ListPattern{Elements, Tail}` at core.go:370. |
| P-2 | `ailang run --caps IO` vs `--bytecode --caps IO` on `recursion_quicksort.ail` | Evaluator (control): correct both lines. Bytecode: `Quicksort: [3]` / `sortBy:    [3]`, **rc=0**, stderr = `✓ Running … via bytecode VM` only, **no** `falling back`. Source of D-B. |
| P-3 | §3 fixture, both engines, at HEAD | Exactly three rows diverge: `arity1 [7,8]`, `arity2 [7,8,9]`, `arity3 [7,8,9,10]`. Underflow, tailed and cons rows correct on **both** engines. Reproduces V-H independently and proves the fixture is VM-executed, not bridged. |
| P-4 | probe applied (`lenOp` per §1); `shasum` before `fd1d890e…` → after `5142335d7e74…`; `go build ./internal/... ./cmd/ailang` rc=0; `make build` | Fixture byte-identical to evaluator on all 15 rows; quicksort prints both correct lines. |
| P-5 | `go test ./internal/gen/...` | rc=0 at base (5 pkgs) **and** under the probe. Confirms the doc's V-G: the existing suite pins nothing in either direction, so AC2 is load-bearing. |
| P-6 | 7 examples × 2 engines, `shasum -a 256` of last 20 lines, at HEAD and under probe | All 7 MATCH in both states; hash prefixes identical across states (see §6 table). |
| P-7 | `go run ./scripts/verify_bytecode_parity.go` at base and under probe | 151→152 MATCH, 6→5 DIVERGE; delta = exactly `recursion_quicksort.ail`. Residual DIVERGE set enumerated in §6. |
| P-8 | `ls tests/golden/bytecode/`; `head -3 tests/golden/bytecode/*.go`; `grep Glob\|WalkDir` | 4 `.go` files, all `package bytecode_golden_test`; no directory walk; fixtures loaded from `../codegen` by name. Source of D-A. |
| P-9 | wrote fixture to `tests/golden/bytecode/pattern_arity.ail`, ran without `--relax-modules`, ran `go test ./tests/golden/bytecode/`, deleted it | Module `tests/golden/bytecode/pattern_arity` resolves canonically (no MOD010, no `--relax-modules`); Go package unaffected (`ok`). Tree left clean. |
| P-10 | `go test -count=1 -run 'TestCLI_RunBytecode' -v ./cmd/ailang` | 5/5 `--- PASS` in 6.35s at base. Confirms the chosen host harness is green and fast. |
| P-11 | `go build ./...` at base | **rc≠0**: `cmd/wasm: runtime.main_main·f: function main is undeclared in the main package`. Source of D-E. |
| P-12 | `make check-file-sizes`, `make check-changelog`, `make vet`, `make fmt-check`, `go test ./tests/golden/bytecode/` at base | all rc=0. |
| P-13 | read `check-golden-drift` target in `make/test.mk` | diffs only `internal/parser/testdata/parser/` — unaffected by `tests/golden/` additions. |
| P-14 | `grep -rn "falling back to evaluator" --include=*.go` | Exactly two sites: emitter `cmd/ailang/run_helpers.go:376`; existing assertion `cmd/ailang/run_bytecode_test.go:63`. Basis for D4 and for the AC3 host choice. |

---

**Plan created**: 2026-08-12 · planner model `claude-opus-5` · read-only w.r.t. git (no add/commit/branch)
