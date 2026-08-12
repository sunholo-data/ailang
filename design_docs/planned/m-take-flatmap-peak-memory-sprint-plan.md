# Sprint Plan — M-TAKE-FLATMAP-PEAK-MEMORY

**Design doc**: [`design_docs/planned/m-take-flatmap-peak-memory.md`](m-take-flatmap-peak-memory.md) (878 lines, Status: Planned, quorum-resolved round 3)
**GitHub issue**: [#617](https://github.com/sunholo-data/ailang/issues/617)
**Target version**: v0.34.0
**Planner**: `claude-opus-5` (V1 mission iteration 182, SPRINT-PLANNER role)
**Planning base SHA**: `063442179` (worktree `.wt-iter182`, branch `iter182/take-flatmap-plan`)
**Doc's declared measurement base**: `eabab0611`
**Duration**: 4 days · **Risk**: medium · **Milestones**: 4 (one commit each, green suite at every boundary)

Every fact below is labelled `VERIFIED BY ME` (with the command, run at `063442179` unless
stated) or `UNVERIFIED (inherited from …)`. Nothing inherited is restated as established.

---

## 1. Freshness of the design doc

**VERIFIED BY ME** (`git -C . diff --stat eabab0611..063442179 -- ':!design_docs'`, relayed
from the controller and re-checked here by `git log --oneline eabab0611..063442179`): the only
non-`design_docs/` file changed between the doc's measurement base and the planning base is
`.claude/skills/mission-control/SKILL.md`. No source file cited by the Verification Log has
moved, so V1–V26 are fresh as *rows*. Where I re-measured a row, I say so; where a row is
still only the doc's word, it stays `UNVERIFIED (inherited from design doc V<n>)`.

Re-measured at `063442179` with a binary built in this worktree
(`./bin/ailang --version` → `AILANG v0.33.0-144-g063442179`):

| Row | Claim | My command | My observation | Verdict |
|---|---|---|---|---|
| V4 | No stdlib surface | `./bin/ailang check /tmp/it182/imp.ail` (imports `std/list (takeFlatMap)`) | `Error: IMP010: symbol 'takeFlatMap' not exported by 'std/list' — did you mean 'take'?` rc=1 | **CONFIRMED** |
| V5 | Annotated `int` cannot reach the builtin | `./bin/ailang check /tmp/it182/w.ail` (wrapper `n: int`) | `cannot unify type constructors: Int vs int` at the delegation site (`w.ail:6:20`) | **CONFIRMED** |
| V-BUILDER | Exactly two `TCon{Name: "Int"}` repair sites | `grep -n 'TCon{Name: "Int"}' internal/builtins/list_bounded.go` → lines **68, 150**; SAME-FILE control `grep -n 'TCon{'` → the same two lines | both `intT := &types.TCon{Name: "Int"}`, in `makeTakeMapType` (68) and `makeTakeFlatMapType` (150) | **CONFIRMED** |
| V16 | `std/list.ail` exports `flatMap` + `flatMapE`; no `takeFlatMap`/`takeMap` | `grep -n 'takeFlatMap\|takeMap' std/list.ail` → **rc=1**; SAME-FILE control `grep -n '^export' std/list.ail` → **38 rows**, incl. `flatMap` at :202, `flatMapE` at :250, `take` at :99, `map` at :56 | absent | **CONFIRMED** |
| V17 | Existing builtin tests green at base | `go test ./internal/builtins -run 'TakeFlatMap\|TakeMap' -count=1` | `ok … 0.380s` rc=0 | **CONFIRMED** (count corrected — see D-7) |
| V14 | LIMITATIONS silent on strictness | `grep -cin "strict\|materiali\|peak memory" docs/LIMITATIONS.md docs/docs/reference/limitations.md` → `0` and `0`; control `wc -l` → 92 and 345 | silent | **CONFIRMED** |
| V15 | No footgun row | `grep -n 'take.*flatMap\|flatMap.*take' internal/diag/footguns.md` → **rc=1**; SAME-FILE control `grep -c '| covered\|| inventoried'` → **12** | absent, table live | **CONFIRMED** |

**Baselines banked by iteration 181 and NOT re-run by me** — `UNVERIFIED (inherited from the
iteration-181 controller)`: `make test` rc=0 / 0 FAIL, `make lint` rc=0, `make verify-examples`
rc=0 (187 modules, pre-existing `1 missing-on-disk`), `internal/types`, `internal/effects`,
`internal/loader` rc=0. I did independently re-run `internal/builtins` (rc=0) and
`go test ./internal/pipeline -run 'BuiltinGolden|Golden' -count=1` (`ok … 0.410s`, rc=0).

---

## 2. Discrepancies found (read this before executing)

Seven. **D-1, D-2 and D-3 change what the sprint must edit**; skipping them ships the sprint's
teaching and its CI contract into files nothing reads — which is #617's own failure mode
(a fix that shipped and was never reachable) reproduced one layer up.

### D-1 — The teaching prompt target is a FROZEN, NON-ACTIVE version. **Blocking.**

The doc names `prompts/v0.16.2.md` in Phase 2 step 7, in "Files to Modify" (line 603) and in
**AC-5's grep**. Three independent things are wrong with that target.

1. **v0.16.2 is not active.** **VERIFIED BY ME** (`jq -r '.active' prompts/versions.json`) →
   `v0.16.5`. Same value in `cmd/ailang/prompts/versions.json`. Editing v0.16.2 changes
   nothing any model is served.
2. **Prompt versions are content-hashed and immutable.** **VERIFIED BY ME**
   (`jq -r '.versions["v0.16.2"]' prompts/versions.json`) — the entry carries
   `"hash": "1c16a8fc…"`. v0.16.5's own release note states the policy in as many words:
   *"v0.16.4 remains served byte-identical for pinned eval baselines."* An in-place edit to
   v0.16.2 desynchronises a recorded SHA-256 **and** mutates a version pinned by banked eval
   baselines.
3. **There are TWO prompt trees and the embedded one is `cmd/ailang/prompts/`.**
   **VERIFIED BY ME**: `//go:embed all:prompts` at `cmd/ailang/main.go:21`; `ls -ld` shows
   `prompts/` and `cmd/ailang/prompts/` are both **real directories** (66 entries each, not
   symlinks); `diff -q prompts/v0.16.5.md cmd/ailang/prompts/v0.16.5.md` → rc=0 (identical
   copies). **VERIFIED BY ME** that no sync automation exists: `grep -n "prompts" Makefile` →
   **0 hits**, SAME-FILE control `grep -c '' Makefile` → **342** lines. Editing only the root
   copy leaves the shipped binary unchanged.

**Correct action (M3).** Cut a **new** version `v0.16.6` = v0.16.5 + the Common-Mistakes block,
written to **both** `prompts/v0.16.6.md` and `cmd/ailang/prompts/v0.16.6.md`, with **both**
`versions.json` files gaining the entry (file, sha256 hash, description, tags, notes) and
`"active"` bumped to `v0.16.6`. This is not invented process — it is exactly the recipe of the
last prompt change. **VERIFIED BY ME** (`git show --stat 1677fcff9`), which touched:
`prompts/v0.16.5.md`, `cmd/ailang/prompts/v0.16.5.md`, `prompts/versions.json`,
`cmd/ailang/prompts/versions.json`, `changelogs/v0.18-current.md`, plus code.

**AC-5 is rewritten accordingly** (§5). As the doc writes it, AC-5 greps `prompts/v0.16.2.md`
— a file the executor would have had to corrupt to make green, and whose greening would still
leave the shipped prompt at **0** mentions. **VERIFIED BY ME**:
`grep -ci "takeFlatMap" prompts/v0.16.5.md cmd/ailang/prompts/v0.16.5.md prompts/v0.16.2.md`
→ `0`, `0`, `0`; SAME-FILE control `grep -ci "flatMap"` → `5`, `5`, `5`.

*Root cause, not the doc's fault:* `internal/diag/footguns.md`'s own header cites
`prompts/v0.16.2.md` as the prompt of record, and the doc inherited it. The footgun table's
"prompt lines to delete" column is therefore **also** stale — noted as a bounded cleanup in M3,
not adopted as this sprint's target.

### D-2 — The doc says there is no golden churn. There is, and it is REQUIRED. **Blocking.**

Conflict Surface, line 646: *"no registration-path change, so no golden builtin-count snapshot
churn beyond metadata."* **VERIFIED BY ME FALSE**
(`grep -n '_list_take' internal/pipeline/testdata/builtin_types.golden`):

```
137:_list_take        : (int, list[a]) -> list[a]
138:_list_takeFlatMap : (Int, a -> list[b], list[a]) -> list[b]
139:_list_takeMap     : (Int, a -> b, list[a]) -> list[b]
```

The golden records the *rendered type*, so the `Int`→`int` repair changes lines 138–139. Line
137 is the same-file known-positive control and the tightest possible evidence for V-BUILDER:
its sibling `_list_take` already renders lowercase `int`, so the deviation is exactly the two
builtins under repair.

**Two consequences the executor must not discover the hard way.** (a) The gate is
`TestBuiltinTypes_GoldenSnapshot` in `internal/pipeline`, updated with
`UPDATE_GOLDEN=1 go test -v ./internal/pipeline -run TestBuiltinTypes_GoldenSnapshot`
(**VERIFIED BY ME**, `internal/pipeline/builtin_golden_types_test.go:26-29,63`). (b) **`make
check-golden-drift` will NOT catch this** — **VERIFIED BY ME** (`make/test.mk:207-219`) it
diffs only `internal/parser/testdata/parser/`. So a stale golden shows up as a red
`internal/pipeline` package, not as the named golden gate.

### D-3 — The Phase-3 diagnostic site named in the doc is the wrong channel. **Resolved by measurement.**

The doc (line 605) defers the site to the planner with the candidate *"`pipeline_module.go`
warning channel used by MOD010"*. **The MOD010 "channel" is not a channel.** **VERIFIED BY ME**
(`grep -rn "MOD010" internal/ --include='*.go'`): `warnMOD010Relaxed`
(`pipeline_module.go:767-779`) is a bare `fmt.Fprintf(os.Stderr, …)`. It has no code, no
position, and never reaches `result.Warnings` — a note emitted there would be invisible to
`ailang check --json`, untestable through `pipeline.Run`, and unformattable.

The **real** structured, non-blocking channel is `result.Warnings []elaborate.Warning`
(`internal/pipeline/pipeline.go:125`). **VERIFIED BY ME** it is rendered and never fails the
check: `cmd/ailang/check.go:243` calls `printCheckWarnings(result.Warnings)` on the success
path, whose comment reads *"Warnings never fail the check"* and which writes
`yellow(w.String())` to stderr (`check.go:250-257`). That satisfies AC-6's "rc still 0"
requirement structurally rather than by convention.

**Decision — the site is a new file `internal/pipeline/warn_take_after_flatmap.go`, modelled
line-for-line on `internal/pipeline/strict_fallbacks.go`, wired into BOTH pipelines.**
`strict_fallbacks.go` is the exact precedent: a `StrictFallbackWarning` implementing
`elaborate.Warning` with `Code()`/`Position()`/`String()` (`:241-274`), a
`DetectStrictFallbacks(file, prog)` walking the elaborated Core (`:281`), and a
compile-time interface assertion (`:251`). **VERIFIED BY ME** the two wiring sites
(`grep -rn "DetectStrictFallbacks|DetectArgOrderWarnings" internal/ cmd/ --include='*.go'`):

- `internal/pipeline/pipeline_module.go:388` and `:395-396` — inside the sorted-`modID` loop.
- `internal/pipeline/pipeline_single.go:198` and `:206`.

**Both must be wired, and only the module one can ever fire.** The in-repo comments say why,
and they are load-bearing for the fixture design: *"In the single-file/REPL pipeline imports
are not resolved to VarGlobal, so this is effectively a no-op today; wired for consistency"*
(`pipeline_single.go:191-194`), versus *"Here imports ARE resolved to VarGlobal"*
(`pipeline_module.go:389-390`). The doc's Phase 3 requires resolved `std/list.take` /
`std/list.flatMap` globals, so the note lives or dies in the module pipeline. This is benign in
practice — any file calling `take`/`flatMap` must `import std/list`, and a file with an
`import` routes to the module pipeline — but it dictates the fixture shape in D-4.

Iterate `compiledUnits` in the existing sorted-`modID` order so warning output stays
deterministic (`pipeline_module.go:380-386` says so explicitly). A1/A2 depend on it.

### D-4 — AC-6 cannot use the fixture convention the doc points at. **Blocking for M4.**

The doc: *"CI contract via a `footgun_fixtures_test.go` fixture … per that file's existing
convention."* **VERIFIED BY ME** that convention is error-only
(`internal/diag/footgun_fixtures_test.go:207-224`):

```go
_, err := pipeline.Run(pipeline.Config{Mode: pipeline.ModeCheck, RelaxModules: true}, pipeline.Source{Code: fx.src})
if err == nil { t.Fatalf("%s (%s): expected a diagnostic, got nil error", fx.name, fx.status) }
msg := err.Error()
```

Our note is non-blocking, so `err == nil` and `pipeline.Run`'s **result** carries the warning.
Adding a row to the `footgunFixtures` table would fail with *"expected a diagnostic, got nil
error"* — for the right reason, in the wrong place.

**Correct action:** add a **separate test function** in the same file, following the shape the
file already uses for pipeline-specific footguns (`TestFootgunFixture_MOD014_ModuleLess:234`,
`TestFootgunFixture_MOD007_DuplicateModuleBinding:286`). Two constraints inherited from those
funcs and confirmed by D-3: it must write a **real file into `t.TempDir()`** (MOD014's comment:
*"only fires in the MODULE pipeline, which requires a real filename on disk"* — same for us),
and it must assert on `result.Warnings`, not on `err`. Also note `TestMain`
(`footgun_fixtures_test.go:38-43`) sets `AILANG_STDLIB_PATH=../../std` and
`types.ImportSuggester` before any fixture runs — our fixtures import `std/list`, so they
depend on that first step being intact.

### D-5 — AC-3(a) is ALREADY GREEN AT BASE. **Rewritten in §5.**

Round 2 accepted reviewer objection 3 and replaced AC-3 with *"a fast Go test using an
instrumented callback and a small fixed input … asserts that `_list_takeFlatMap` returns the
expected prefix and invokes `f` only through the first inputs needed to produce n outputs."*

**That test already exists, twice.** **VERIFIED BY ME**
(`internal/builtins/list_bounded_test.go`):

- `TestTakeFlatMapEarlyExit` (`:261`) — counting `expand` callback; asserts the prefix
  `[10,11,12,20,21]` **and** `callCount != 2 → t.Errorf("expected f to be called 2 times")`.
- `TestTakeMapEarlyExit` (`:103`) — counting `counter` callback; asserts `callCount != 3`.

And **VERIFIED BY ME** the package is green at base (`go test ./internal/builtins -run
'TakeFlatMap|TakeMap' -count=1` → `ok … 0.380s`). So AC-3a as written is satisfied by writing
no code: it measures the repo, not the change. Its own named mutation — *"re-implementing
`takeFlatMap` as sugar for `take(n, flatMap(f, xs))`"* — is **already killed at base** by
`TestTakeFlatMapEarlyExit`.

The reviewer's *reasoning* is right and the instrument is right; what is wrong is the **layer**.
The mutation that this sprint actually makes newly available is one layer up and invisible to
those tests: an executor who "fixes" the IMP010 by writing

```ailang
export pure func takeFlatMap[a, b](n: int, f: (a) -> [b], xs: [a]) -> [b] = take(n, flatMap(f, xs))
```

ships a byte-identical-output export that reintroduces the entire OOM, and **every existing Go
test still passes**, because they exercise `takeFlatMapImpl` directly and never look at
`std/list.ail`. AC-3a is retargeted at that, with an instrument I measured (D-6). Two small
genuine gaps at the Go layer are folded into M1: `TestTakeMapEarlyExit` asserts the count but
**not** the returned prefix (`_, err := takeMapImpl(...)` — **VERIFIED BY ME**, `:112`), and
neither test carries the explicit short timeout objection 3 asked for.

### D-6 — The replacement AC-3a instrument, measured end-to-end at HEAD

An invocation counter is unavailable through the stdlib export (the fused builtins are
pure-only, V6 — you cannot count in a pure `f`). A **refusal-divergence** instrument works and
is deterministic. **VERIFIED BY ME**, both arms, at `063442179`:

```ailang
pure func boom(x: int) -> [int] = if x > 2 then [100 / (x - x)] else [x, x]
```

| arm | command | observed | rc |
|---|---|---|---|
| fused (mechanism present) | `_list_takeFlatMap(4, boom, [1,2,3,4,5])` | prints `[1, 1, 2, 2]` | **0** |
| unfused (the mutation) | `take(4, flatMap(boom, [1,2,3,4,5]))` (imports `std/list (take, flatMap)`) | `panic: division by zero` | **2** |

`f` is never invoked past the second input under the fused form, so the poison branch never
evaluates; under `take∘flatMap` strictness forces every element and the run dies. The
observable (`rc` + stdout) is **downstream of the early-exit loop itself**, not stamped
alongside it, and the mutation named in AC-3a flips it.

One honesty note carried into the AC: the unfused arm's failure is an **unstructured Go panic**
raised from `internal/types/dictionaries.go:119` (`registerNumInt`), not a structured AILANG
error — an A11 gap that is **out of scope here** but that would change this test's expected
output if someone fixes it. The AC therefore asserts on the *fused* arm's rc=0 + exact stdout
(stable under any such fix) and treats the unfused arm as the documented kill-mutation, run by
the executor once and recorded, not pinned in CI.

### D-7 — Three small factual corrections (non-blocking)

- **V17's test count.** Doc: *"22 test funcs + 2 benchmarks"*. **VERIFIED BY ME**
  (`grep -c "^func Test" internal/builtins/list_bounded_test.go`) → **20** test funcs; plus the
  2 benchmarks (`BenchmarkTakeMap:437`, `BenchmarkTakeFlatMap:458`) that is 22 *functions*
  total. The doc double-counted the benchmarks.
- **The example path is not on the CI-gated surface.** Doc: `examples/bounded_take_flatmap.ail`.
  **VERIFIED BY ME**: `make verify-examples` (`make/examples.mk:18`) runs
  `scripts/verify_examples.go` + `scripts/validate_manifest.go` over **`examples/runnable/`**;
  `examples/manifest.json`'s first entry has `"path": "adt_option.ail"` and
  `ls examples/adt_option.ail` → **no such file** while `ls examples/runnable/adt_option.ail`
  → present. `examples/runnable/` holds 174 `.ail` files against 42 at `examples/` top level,
  and the manifest has 193 entries. **The example must be created at
  `examples/runnable/bounded_take_flatmap.ail`** and registered with `expected.stdout` +
  `expected.exit_code`, or it is not gated by anything.
- **V19's conflict scope is narrower than the file set.** V19 checked `list_bounded.go` and
  named only `m-parmap-effectful` for `std/list.ail`. **VERIFIED BY ME**
  (`grep -rln "std/list.ail" design_docs/planned/`) **six** planned docs mention it. I checked
  each by purpose (§3); the doc's conclusion survives, but one of the six is worth knowing
  about.

---

## 3. Conflict surface, re-verified by purpose (not quoted from the doc)

- **`m-parmap-effectful` — collision still LIVE, still low.** **VERIFIED BY ME**:
  `design_docs/planned/v0_30_0/m-parmap-effectful.md` is `**Status**: Planned`, and its line 191
  still reads *"`std/list.ail` or new `std/par.ail` (~+40 LOC) — `parMap`/`parMapN`/
  `parMapResult` surface"*, line 255 the same. Both changes are **additive exports at different
  sites**; textual conflict risk low, semantic none. It also cites the `std/list.ail:212-214`
  effectful-combinator contract comment that we cite — neither sprint edits that comment.
  Whichever lands second rebases. Unchanged from the doc's assessment.
- **`m-verify-stdlib-stale-path` (planned/v0_29_0) — NOT in the doc's Conflict Surface, and
  directly about this sprint's action.** **VERIFIED BY ME** by reading it: `make verify-stdlib`
  reads `STDLIB_DIR="stdlib/std"`, a directory that has not existed since v0.3.20, so the
  stdlib interface-stability gate *"hasn't actually verified anything"* and the
  `.stdlib-golden/` files are frozen against the old layout. Its own Problem section names our
  exact action as the thing it should have flagged: *"Adding the M-SNAKE-FEEDBACK exports …
  to `std/list` is exactly the kind of change this gate exists to flag — and it couldn't."*
  **Impact on us: none, and that is the point** — `verify-stdlib` is not in `make ci`
  (**VERIFIED BY ME**, `grep -rn "make " .github/workflows/*.yml` lists no `verify-stdlib`), so
  adding two `std/list` exports trips no interface gate. Do **not** fix it here (out of scope);
  do record in the changelog that the exports landed ungated by `verify-stdlib`.
- The other four `std/list.ail` mentions (`m-effect-replay-subsumption-sprint-plan`,
  `m-parser-reserved-keyword-diagnostics`, `m-effect-row-var-unification`, and this doc) are
  citations, not edits — **VERIFIED BY ME** by reading each hit in context; e.g. the parser doc's
  only hit is an illustrative `import std/array (length as lengthAlt) — [std/list.ail or similar]`.
- **`internal/pipeline/` and #616 (iteration 180, parked at `D-10`)** —
  `UNVERIFIED (inherited from design doc Conflict Surface)`. Our change adds a **new file** plus
  two one-line append statements; #616 is in `validate_effects.go` / `internal/types`. No shared
  lines expected. If #616 is authorized concurrently, coordinate merge order.
- **`internal/diag/footguns.md` + fixtures** — owned by the implemented `m-diagnostic-coverage`;
  adding a row is the mechanism it exists for. See D-4 for the one place its convention does not
  extend to us.

---

## 4. CI gates, derived not recalled

**VERIFIED BY ME** (`grep -rn "make " .github/workflows/ci.yml`) — the gates a change in this
sprint's file set can plausibly move:

`make test-parser` · `make check-file-sizes` · `make check-boundaries` · **`make check-changelog`**
· `make check-skills` · `make test-coverage-gate` · `make check-golden-drift` · `make fuzz-parser`
· `make test-lowering` · `make test-imports-success` · `make test-import-errors` ·
`make verify-no-shim` · `make doctor` · `make test-regression-guards` · `make test-coverage` ·
`make verify-examples` · `make fmt-check` · `make vet` · `make lint`

Three that specifically bite this sprint:

1. **`make check-changelog`** — **VERIFIED BY ME** it is real and runs
   `bash scripts/check_changelog.sh` (`make/code-health.mk:142-143`), described as *"Check root
   CHANGELOG.md stays an index, not a changelog"*. The doc's "Files to Modify" writes
   `CHANGELOG.md / changelogs/v0.18-current.md`. **The entry goes in
   `changelogs/v0.18-current.md`** — **VERIFIED BY ME** that this is where the comparable
   prompt+builtin change put it (`git show --stat 1677fcff9`). Anything narrative left in root
   `CHANGELOG.md` is dropped from the release *and* reds this gate.
2. **`make check-golden-drift` will NOT protect you** (D-2) — it watches
   `internal/parser/testdata/parser/` only. The pipeline golden is guarded by its own test.
3. **`make check-file-sizes`** — fails >800 lines. `internal/pipeline/pipeline_module.go` is
   already large; another reason D-3's decision is a **new file**, not an insertion.

`go build ./...` is **not** usable as a gate: it is known-red at base (`cmd/wasm` and `gen/main`
have no native `main`). Where a build check is needed (refusal-mutant compilability, §6), the
criterion is `go build ./internal/... ./cmd/ailang` with its base result recorded.

---

## 5. Acceptance criteria (as executed)

AC-1, AC-2, AC-4, AC-7 are adopted from the doc unchanged in substance. **AC-3 and AC-5 are
rewritten** (D-5, D-1) and **AC-6's mechanism is respecified** (D-4). Each row states its
baseline **measured on the pristine tree** and the mutation that kills it.

| # | Criterion | Baseline at `063442179` | Killed by |
|---|---|---|---|
| **AC-1** | Two fresh files, one importing `std/list (takeFlatMap)`, one `std/list (takeMap)`, both `ailang check` rc=0 | **VERIFIED BY ME**: rc=1, `IMP010: symbol 'takeFlatMap' not exported by 'std/list' — did you mean 'take'?`; `takeMap` likewise absent (`grep` rc=1, 38-export control) | dropping either export |
| **AC-2** | Both annotated wrappers (`n: int`) type-check and run; `takeFlatMap` half prints `[1, 2]`, `takeMap` half's prefix asserted | **VERIFIED BY ME**: `cannot unify type constructors: Int vs int` at the delegation site | reverting either `T.Int()` site (68 / 150); each is separately killed by the paired Go regression |
| **AC-3a** *(rewritten — D-5/D-6)* | `.ail` integration test: `takeFlatMap(4, boom, [1,2,3,4,5])` **imported from `std/list`** exits 0 and prints exactly `[1, 1, 2, 2]`, where `boom x = if x > 2 then [100/(x-x)] else [x,x]`; same shape for `takeMap`. Plus: `TestTakeMapEarlyExit` gains the missing returned-prefix assertion, and both early-exit tests gain explicit short timeouts | **VERIFIED BY ME**: inexpressible — AC-1's import fails. The *builtin-layer* instrument the doc specifies is **already green at base** (`TestTakeFlatMapEarlyExit:261` asserts prefix **and** `callCount==2`; `TestTakeMapEarlyExit:103` asserts `callCount==3`; package `ok 0.380s`) | writing the export body as `take(n, flatMap(f, xs))` — output-identical, but the run dies (**VERIFIED BY ME**: rc=2, `panic: division by zero`) |
| **AC-3b** | Release evidence only, non-CI: `/usr/bin/time -l` on both rewritten repros, recorded in `changelogs/v0.18-current.md` | `UNVERIFIED (inherited from design doc V1/V2, V25/V26)`: 425 MB/81.9 s → target <150 MB/<1 s; 559 MB/18.78 s → 101 MB/0.08 s | n/a — evidence, not a gate. Peak RSS never becomes a CI gate |
| **AC-4** | `.ail` parity: `takeFlatMap(n,f,xs) == take(n,flatMap(f,xs))` and `takeMap(n,f,xs) == take(n,map(f,xs))` for pure `f`, across n=0, n>total, empty inner lists, and the `dedup∘takeFlatMap` shape | **VERIFIED BY ME**: inexpressible (AC-1) | off-by-one in either early-exit loop — **VERIFIED BY ME** the site is `list_bounded.go:190-192` (`if len(result) >= n { break }`); `>=`→`>` yields n+1 |
| **AC-5** *(rewritten — D-1)* | `grep -ci "takeFlatMap"` ≥1 **and** `grep -ci "takeMap"` ≥1 in **each** of: `docs/docs/reference/limitations.md`, `docs/LIMITATIONS.md`, `internal/diag/footguns.md`, `prompts/v0.16.6.md`, `cmd/ailang/prompts/v0.16.6.md`. **And** `jq -r '.active' prompts/versions.json` = `jq -r '.active' cmd/ailang/prompts/versions.json` = `v0.16.6`. **And** the recorded hash matches the file: `shasum -a 256 prompts/v0.16.6.md` == `jq -r '.versions["v0.16.6"].hash' prompts/versions.json`. **And** `diff -q prompts/v0.16.6.md cmd/ailang/prompts/v0.16.6.md` rc=0 | **VERIFIED BY ME**: `takeFlatMap` = 0 in every one of the five files (control `flatMap` = 5 in each prompt, 12 footgun rows, 92/345 limitation lines); `.active` = `v0.16.5` in both trees | shipping code without teaching — the exact way M-EVAL-BOUNDED-PIPELINE stayed invisible for 5 months. `takeMap` is not a substring of `takeFlatMap`, so the greps are independent |
| **AC-6** *(mechanism respecified — D-4)* | New test func in `internal/diag/footgun_fixtures_test.go` (NOT a `footgunFixtures` row) writing real files to `t.TempDir()` and asserting on `result.Warnings`: (1) direct trap file → a warning whose `Code()` is `LIST_TAKE_AFTER_FLATMAP` and whose `String()` contains `takeFlatMap`, with `err == nil`; (2) fused file → no such warning; (3) `take(n, map(...))` file → no such warning; (4) a `sortBy`-using file → no such warning | **VERIFIED BY ME**: the trap file checks clean today (V1's `✓ No errors found!` shape; `TestFootgunFixtures:212` would reject a warning-only diagnostic with *"expected a diagnostic, got nil error"*) | removing the pass (1); over-matching (2–4) |
| **AC-7** | `go test ./internal/builtins -run 'TakeFlatMap\|TakeMap' -count=1`, `go test ./internal/pipeline -count=1`, `go test ./internal/diag -count=1`, `make verify-examples`, `make check-changelog`, `make lint`, `make fmt-check` all green | **VERIFIED BY ME**: builtins `ok 0.380s`; pipeline golden subset `ok 0.410s`. `UNVERIFIED (inherited from iteration 181)`: full `make test` rc=0, `make lint` rc=0, `make verify-examples` rc=0 at 187 modules with a **pre-existing** `1 missing-on-disk` that must not grow | anti-regression only — new-code reach is AC-1…AC-6, so "suite green" never stands in for them |

**Instrument hazard carried from iteration 180 and still live:** `ailang check` caches by
content. Every check-based AC runs on fresh content or states it is first-touch.

---

## 6. Mutation obligations (refusal branches)

This sprint adds no new user-facing refusal by design — the note is non-blocking and the type
repair *widens* what is accepted. But the obligation is on the **diff**, not on the plan, so:

Before declaring M4 done, run
`grep -cE 'return .*(fmt\.Errorf|errors\.New|status\.Error)\(' <each file in the M4 diff>` and
enumerate every refusal branch **from that output**, not from this document. Note the `%w`
wrapping form is blind to terminal refusals, hence the `\(`-anchored alternation above. For each
branch, add one `if false && <cond>` mutant, assert the mutant **builds**
(`go build ./internal/... ./cmd/ailang`, rc unchanged from its base value) *before* reading a
red, then confirm a test fails. For reference, `list_bounded.go` already carries four such
branches per builtin at base (non-Int `n`, non-List input, nil `FnCaller`, callback-must-return-a-List)
and all four are covered by existing tests — **VERIFIED BY ME**
(`TestTakeFlatMapNonListReturn:349`, `TestTakeFlatMapNilFnCaller:362`, `TestTakeMapNonIntN:146`,
`TestTakeMapNonListInput:157`). M1 must not weaken them: the `Int`→`int` change is to the
*declared type*, not to the runtime `*eval.IntValue` assertion.

**Executable-string obligation.** `examples/runnable/bounded_take_flatmap.ail` and the new
prompt block both show code a human or a model is told to run. The manifest entry's
`expected.stdout` must be taken from the example's **own** output, and `make verify-examples`
executes the file verbatim — that satisfies the obligation for the example. For the diagnostic,
AC-6 fixture (1) asserts the fix substring `takeFlatMap`, and AC-1 proves that exact identifier
is importable, so the string the note tells a model to write is one the compiler accepts.

---

## 7. Milestones

Four milestones, **one commit each**, full suite green at every boundary (bisectability). Order
is forced by dependency: nothing can be taught or linted before the name exists and works.

### M1 — REPAIR: `Int` → `int`, golden, metadata (Day 1, ~0.5 day, ~90 LOC)

**Goal:** make the fused builtins usable behind an annotated `int` and correct their metadata.

1. `internal/builtins/list_bounded.go:68` and `:150` — `&types.TCon{Name: "Int"}` → the
   builder's canonical `T.Int()`. Both sites; they are the *only* two (D-2's line-137 control).
2. `internal/builtins/list_bounded.go:49,131` — `Since: "v0.9.4"` → `"v0.10.0"`
   (`UNVERIFIED (inherited from design doc V12)`: `d41e43894`'s first tag is v0.10.0 — executor
   re-runs `git tag --contains d41e43894 | sort -V | head -1` and uses what it prints).
3. `internal/builtins/list_bounded.go:124` — `LongDesc`'s *"For effectful f, only evaluates f
   for as many input elements as needed"* is unreachable through the builtin's own pure type
   (V6). Rewrite to pure-only wording + "effectful variant deferred".
4. `internal/pipeline/testdata/builtin_types.golden` — regenerate via
   `UPDATE_GOLDEN=1 go test -v ./internal/pipeline -run TestBuiltinTypes_GoldenSnapshot`; the
   diff must be **exactly** lines 138–139, `Int` → `int`. Any larger diff means something else
   moved — stop and investigate.
5. `internal/builtins/list_bounded_test.go` — red-at-base regression exercising **both**
   repaired sites through an annotated `int`; plus D-5's two gap-fills (returned-prefix
   assertion in `TestTakeMapEarlyExit`, explicit short timeouts on both early-exit tests).

**Closes:** AC-2 (Go half). **Boundary gate:** `go test ./internal/builtins ./internal/pipeline
-count=1`, `make lint`, `make fmt-check`.

### M2 — EXPOSE: `std/list` surface, example, parity + delegation instrument (Day 2, ~1 day, ~140 LOC)

**Goal:** `import std/list (takeFlatMap)` / `(takeMap)` work, and the exports are pinned to the
fused mechanism rather than to output equality.

1. `std/list.ail` — two `export pure func`s delegating to `_list_takeFlatMap` / `_list_takeMap`,
   using the `map`→`_list_map` idiom at `std/list.ail:56`. Doc comments carry the **corrected**
   cost model (`peak = source residency + largest single f(x) + n retained`) and say plainly
   that neither bounds peak by `n`. Append after the pure-combinator block and **before** the
   effectful section at :212-214, to keep textual distance from `m-parmap-effectful`'s
   append point.
2. `examples/runnable/bounded_take_flatmap.ail` (**not** `examples/` — D-7) — unfused/fused
   pairs for both shapes with cost comments; register in `examples/manifest.json` with
   `expected.stdout` copied from the file's own run and `expected.exit_code: 0`.
3. Parity suite (AC-4) across n=0, n>total, empty inner lists, `dedup∘takeFlatMap`.
4. Delegation instrument (AC-3a, D-6): the `boom` divergence test through the **stdlib import**.

**Closes:** AC-1, AC-2 (`.ail` half), AC-3a, AC-4. **Boundary gate:** M1's gates +
`make verify-examples` (187 modules; the pre-existing `1 missing-on-disk` must not grow).

### M3 — TEACH: limitations, prompt **v0.16.6**, footgun row, changelog (Day 3, ~1 day, ~120 LOC)

**Goal:** the fix becomes discoverable — the step whose absence is the whole reason #617 exists.

1. `docs/docs/reference/limitations.md` — canonical entry *"Strict evaluation: `take(n,
   flatMap(f, xs))` bounds the result, not the peak"*, with the V1/V2 and V25/V26 tables, the
   corrected cost model, the `--max-recursion-depth` anti-pattern, the bound-it-inside-`f`
   guidance, and the allocating-`f` map half. Per M-V1-STABILITY-PROMISE: repro transcript +
   verified-at version.
2. `docs/LIMITATIONS.md` — summary row.
3. **New prompt version `v0.16.6`** per D-1: copy v0.16.5, add a ≤8-line Common-Mistakes block,
   write to **both** `prompts/` and `cmd/ailang/prompts/`, add the entry with its real sha256 to
   **both** `versions.json` files, bump `"active"` in both. Do **not** touch v0.16.2 or v0.16.5.
4. `internal/diag/footguns.md` — one row; blind-spot column records **both** known false
   negatives (value-level aliases, V10; allocating-`f` `take∘map`, V25/V26). Status
   `inventoried` now, flipped to `shipped-this-sprint` by M4. Bounded cleanup: the table header's
   `prompts/v0.16.2.md` reference is stale (D-1) — repoint the header and this row's
   "prompt lines to delete" column at v0.16.6; leave other rows' line numbers alone.
5. `changelogs/v0.18-current.md` — entry with the AC-3b `/usr/bin/time -l` before/after table.
   **Not** root `CHANGELOG.md` (§4).

**Closes:** AC-5. **Boundary gate:** M2's gates + `make check-changelog`, plus the AC-5 hash and
cross-tree `diff -q` checks.

### M4 — DIAGNOSE: the `LIST_TAKE_AFTER_FLATMAP` note (Day 4, ~0.75 day, ~120 LOC) — **CUT LINE**

**Goal:** error-time teaching, so the prompt line can later be deleted.

1. **New file** `internal/pipeline/warn_take_after_flatmap.go`, modelled on
   `strict_fallbacks.go`: a `TakeAfterFlatMapWarning` implementing `elaborate.Warning`
   (`Code() = "LIST_TAKE_AFTER_FLATMAP"`, `Position()`, `String()` carrying the fix), a
   `var _ elaborate.Warning = (*TakeAfterFlatMapWarning)(nil)` assertion, and a
   `DetectTakeAfterFlatMap(prog *core.Program) []elaborate.Warning` matching
   `App(std/list.take, [n, App(std/list.flatMap, [f, xs])])` on resolved `VarGlobal`s.
   New file, not an insertion — `check-file-sizes` (§4).
2. Wire into `internal/pipeline/pipeline_module.go` (inside the existing sorted-`modID` loop at
   :387-397, preserving deterministic order) **and** `internal/pipeline/pipeline_single.go`
   (:198-206), matching how both existing detectors are wired. Single-file is a
   consistency no-op (D-3).
3. `internal/diag/footgun_fixtures_test.go` — new warning-based test func per D-4, four
   fixtures (trap / fused / take∘map / `sortBy`), real files in `t.TempDir()`.
4. Flip the footguns.md row to `shipped-this-sprint`.

**Closes:** AC-6. **Boundary gate:** all prior gates + `go test ./internal/diag ./internal/pipeline
-count=1` + §6's mutation obligations.

**Cut policy (the doc's step 10, adopted):** if M4 runs over budget it is dropped as a
**recorded decision** — M1–M3 alone close #617's primary ask. The footgun row stays
`inventoried` with the target diagnostic recorded, and the changelog says so.

---

## 8. Risks

| Risk | Impact | Mitigation |
|---|---|---|
| Prompt shipped to the stale v0.16.2 (D-1) | **High** — the whole teaching half becomes invisible; #617's failure mode recreated | AC-5 asserts `.active == v0.16.6` in **both** trees, hash-matches the file, and `diff -q`s the two copies |
| Golden not regenerated (D-2) | Med — `internal/pipeline` reds and `check-golden-drift` won't say why | M1 step 4 makes it explicit and bounds the expected diff to two lines |
| Note wired only into the single-file pipeline (D-3) | Med — silently never fires; fixtures pass in the wrong direction | AC-6 fixtures write real files to `t.TempDir()`, forcing the module pipeline |
| AC-3a implemented at the already-green builtin layer (D-5) | Med — sprint scores green having pinned nothing new | AC-3a is retargeted at the stdlib export with a baseline of "inexpressible", not "green" |
| `std/list.ail` rebase against `m-parmap-effectful` | Low | Additive at different sites; append before the :212-214 effectful block |
| Unstructured `panic: division by zero` (D-6) changes if someone fixes A11 | Low | AC-3a pins only the **fused** arm's rc=0 + exact stdout; the unfused arm is a recorded one-off, not a CI assertion |
| `take(n, map(f, ...))` with an allocating `f` gets no note | Med — a stated lint false negative | Taught where the "if `f` allocates" conditional can be stated (AC-5); `takeMap` exported (AC-1); footgun row records the blind spot; AC-6 fixture 3 pins the *rule*, not a safety claim |
| Example lands outside `examples/runnable/` (D-7) | Low | M2 step 2 names the gated path and the manifest fields |

---

## 9. Out of scope

- `takeFilter`, `_list_take` exposure, further fused forms (evidence bar — Non-Goals).
- `takeFlatMapE` / `takeMapE`; general lazy evaluation / iterator protocol.
- Fixing `concat`/`take` recursion limits (V9); streaming JSON decode (M-JSON-STREAMING).
- Making the lint see value-level aliases (V10).
- **Fixing `make verify-stdlib`'s dead `stdlib/std` path** (`m-verify-stdlib-stale-path`, §3) —
  its own P3 doc owns it; record in the changelog that these exports landed ungated by it.
- **Fixing the unstructured `panic: division by zero`** at `internal/types/dictionaries.go:119`
  (D-6) — a real A11 gap, found here, routed out. Worth its own issue.
- Re-dating the other `prompts/v0.16.2.md` references in `footguns.md` rows beyond ours (D-1).

---

**SPRINT_PLAN_PATH**: `design_docs/planned/m-take-flatmap-peak-memory-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_M-TAKE-FLATMAP-PEAK-MEMORY.json`

**Created**: 2026-08-12 · V1 mission iteration 182 · planner `claude-opus-5`
