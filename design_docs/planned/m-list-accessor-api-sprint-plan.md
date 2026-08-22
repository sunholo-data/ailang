# Sprint plan — M-LIST-ACCESSOR-API (LC-2: the accessor seam + the `listrep` ratchet analyzer)

**Design doc (AUTHORITY):** [`design_docs/planned/m-list-accessor-api.md`](m-list-accessor-api.md) — 793 lines, quorum-cleared under the narrow-refinement carve-out at mission iteration 251, **not modified by this plan**.
The design doc WINS wherever it and this plan disagree. This plan adds only *how*, *in what order*, *under which gates*, and the first-party measurements taken to settle what the doc leaves unstated. **No rule, exemption, contract clause, or acceptance criterion is relaxed, paraphrased into a looser gate, or dropped here.** Where this plan believes the doc contains a defect, it says DEFECT and states the resolution it takes — it does not silently redesign.

**Programme position:** LC-2 of [`m-list-cons-cells-decomposition.md`](m-list-cons-cells-decomposition.md). LC-1 is COMPLETE (verdict GO). **D-22 = `C1`** — plain cons cells `{head Value, tail *cell, n int}` with a cached length. **`C2K32` is DECLINED; no chunk-aware anything appears in this plan.** LC-3a/b/c and LC-4 are all denominated in this piece's analyzer count, so a wrong number here mis-sizes ~6–8 further person-days.

**Created:** 2026-08-22 (V1 mission iteration 252, sprint-planner stage)
**Planned at HEAD:** `8e3928a08` (= `origin/dev`, 0 ahead / 0 behind). The design doc's own Verification Log declares base `684ebc23e` = `origin/dev~1`, one docs-only commit back; every load-bearing row was re-derived here rather than transcribed.
**Milestones:** 6 · **Estimated:** ~1,660 Go LOC + ~250 report/changelog lines · **4.5 days**
**Risk level:** MEDIUM · **Planner lane:** codex-ok (doc line 15) · **Target:** v0.35.0

> ⚠ **Estimate exceeds the doc's stated 4 days by +0.5.** Stated rather than compressed to fit. The three line items are named in §8 with their LOC. The roadmap band for LC-2 was 3–4 days, so this is a **band breach of 0.5 day** and the controller should treat it as such, not absorb it.

---

## 1. First-party verification log (this planning session)

Every row measured by the planner in the main checkout at `8e3928a08`. Rows the **controller** measured this iteration are marked `[C]` and were re-run anyway where they were load-bearing. Every negative claim carries a same-scope firing control in the same call, and every exit code was captured **without a pipe** (`cmd >log 2>&1; echo $?`), because `cmd | head; echo $?` reports `head`'s status — a trap this session hit once and corrected (see P1).

| # | Claim | Command | Observed |
|---|---|---|---|
| P1 | **`go build ./...` is rc=1 on pristine `dev`.** It MUST NOT appear as an acceptance gate anywhere | `go build ./... >log 2>&1; echo $?` (unpiped) | **rc=1**. `cmd/wasm` and `gen/main`: `function main is undeclared in the main package`. *Instrument note:* the same command **piped** into `head` reports rc=0 — the pipe returns `head`'s status. Every rc in this plan is captured unpiped |
| P2 | Buildable scopes on pristine `dev` | `go build ./internal/eval/... ; echo $?` · `go build -o /tmp/... ./cmd/ailang ; echo $?` | **0** and **0** |
| P3 | Gate baselines on pristine `dev` (each rc unpiped) | `go vet ./internal/eval/...` · `make check-file-sizes` · `make check-boundaries` · `make check-changelog` · `gofmt -l internal/eval cmd/ailang \| wc -l` | **0 · 0 · 0 · 0 · 0 lines** |
| P4 | **`golang.org/x/tools` is ABSENT from `go.mod` — not indirect, absent.** The doc's L9/HID wording "becomes a direct dependency" implies promoting an existing indirect requirement; there is none | `grep -c 'x/tools' go.mod` → **0**; control `grep -c 'golang.org/x' go.mod` → **8**; `grep -rl 'golang.org/x/tools' --include='*.go' .` → **0**; control `grep -rl 'github.com/fatih/color' --include='*.go' .` → **3** | Absent from `go.mod` and imported by **zero** Go files. `go.sum` holds **9** x/tools lines from the transitive graph, of which `v0.48.0 h1:…` is a full module hash |
| P5 | The x/tools version the sprint will pin is already in the local module cache, so `make test`'s **offline** leg (`GOPROXY=off`, poisoned `HTTP(S)_PROXY`, `make/test.mk:30`) will not break | `ls -d $(go env GOMODCACHE)/golang.org/x/tools*` | `v0.43.0`, `v0.44.0`, `v0.47.0`, **`v0.48.0`** all present, with `.ziphash` |
| P6 | `analysistest` and `singlechecker` exist in the pinned version | `test -d $(go env GOMODCACHE)/golang.org/x/tools@v0.48.0/go/analysis/analysistest` · same for `singlechecker` | **PRESENT · PRESENT** |
| P7 | `tools/linters/` does not exist — the analyzer is a new tree | `test -d tools/linters` → ABSENT; control `test -d tools/ci` → PRESENT | ABSENT with firing control |
| P8 | **`make ci` appears 0 times in `.github/workflows/ci.yml`** — a ci.mk-only edit gates nothing | `grep -c 'make ci' .github/workflows/ci.yml` → **0**; same-file control `grep -c 'make ' …` → **37** | Confirms doc L11. **AC-8 must gate on the ci.yml step, never on the make target** |
| P9 | **The `ci:` aggregate contains NO existing `check-*` gate** — so adding `check-listrep` to it diverges from the doc's own precedent (L12) | read `make/ci.mk:11` | `ci: deps fmt-check vet lint test test-nightly-classifier test-coverage-badge test-lowering verify-no-shim verify-examples verify-examples-toplevel verify-stdlib verify-mcp-tools verify-install-guide`. `check-file-sizes`, `check-boundaries`, `check-changelog`, `check-autoclose`, `check-skills` are **all** absent; every one of them is ci.yml-only |
| P10 | Gate precedent shape, as the doc's L12 claims | read `make/code-health.mk:139-151`; `.github/workflows/ci.yml:140-170` | `check-boundaries`/`check-changelog`/`check-autoclose`/`check-skills` are each `@bash scripts/…`; ci.yml pairs each gate with a `test-check-*` self-test step |
| P11 | **The vacuous-`-run` class, confirmed first-party.** `go test -run <selector>` exits **0** when the selector matches nothing | `go test ./internal/eval -run 'TestZzNoSuchTestIter252' -count=1 -v >log 2>&1; echo $?` | **rc=0**, **0** `=== RUN` lines, `ok … [no tests to run]` |
| P12 | Control for P11, and the exact regex that separates top-level tests from subtests | `go test ./internal/eval -run '^TestTaggedValue$' -count=1 -v` | **rc=0**, **5** `=== RUN` lines but **1** matching `^=== RUN   Test[^/]*$` (4 are subtests). **Exit codes for P11 and P12 are identical** |
| P13 | The enumeration floor is not merely theoretical: a 2-name selector where 1 name does not exist still exits 0 | `go test ./internal/eval -run '^(TestTaggedValue\|TestZzNope252)$' -count=1 -v` | **rc=0**, top-level count **1**, expected **2** → only a count assertion catches it |
| P14 | `*ListValue` has exactly 2 methods; the struct is a bare `Elements []Value` | `grep -n 'func (.*ListValue)' internal/eval/*.go`; control `grep -c 'func (' internal/eval/value.go`; read `value.go:83-89` | `value.go:88` `Type()`, `:89` `String()`; control **42**. `ArrayValue` (`:103`) and `TupleValue` declare the same field name |
| P15 | No symbol collision for `NewList`/`EmptyList`/`Cons` in `internal/eval` | `grep -rn 'func NewList\|func EmptyList\|func Cons(' --include='*.go' internal/eval/` | **1** hit and it is `builtin_errors.go:149 func EmptyListError` — a different symbol. Confirms doc L14 |
| P16 | Textual blast radius (contaminated upper bounds) reproduces exactly at `8e3928a08` | `grep -rn '\.Elements' --include='*.go' internal/ cmd/ \| wc -l`; `'ListValue{'`; controls `'ArrayValue{'`, `'ZzqxValue252{'` | **903** · **388** · control **14** · negative control **0** |
| P17 | **42% of `.Elements` and 75% of `ListValue{` live in `_test.go`** — the number the `Tests` flag decides (§2 DEFECT-1) | `grep -rn '\.Elements' --include='*_test.go' internal/ cmd/ \| wc -l`; same for `ListValue{` | **380** of 903 · **291** of 388 |
| P18 | Direct-syntax mutation at HEAD, **without** the `_test.go` filter | `grep -rnE '\.Elements(\[[^]]*\])? *(=\|\+=)[^=]' --include='*.go' internal/ cmd/`; control `echo 'x.Elements[0] = y' \| grep -cE <same>` → 1 | **2** hits, both `internal/parser/parser_literals.go` (`:248` on `*ast.ListLiteral`, `:282` on `*ast.ArrayLiteral`) — parser AST nodes Rule 2's type filter excludes. **0** in `_test.go`. Confirms doc L13/L28: Rule 2 goes in green |
| P19 | `copy`/`append` rooted at the field selector (doc's named unmeasured surface) | `grep -rn 'copy([a-zA-Z_.]*\.Elements' internal/ cmd/` → **0**, control `grep -rn 'copy(' …` → **83**; `grep -rnE 'append\([a-zA-Z_.]*\.Elements[,)]' …` → **2** | copy: zero with a firing control. append-first-arg: the same 2 parser-AST hits from P18. **Rule 2's "first full run may surprise" risk is now measured at zero for eval values** |
| P20 | The three known escapes exist and are `*eval.ListValue`-typed | read `internal/builtins/safe_cast.go:97`, `internal/embed/convert.go:346`, `internal/effects/testctx/mock_context.go:353-355` | all three are `return …(*eval.ListValue).Elements` shapes. ⚠ the doc abbreviates the third as `testctx/mock_context.go` — `internal/testctx` is **DIR-ABSENT** (control `internal/embed` DIR-PRESENT); the real path is `internal/effects/testctx/` |
| P21 | `internal/embed` (AC-4's hand-audit sample) is small enough to audit by hand | `grep -rn '\.Elements' --include='*.go' internal/embed \| wc -l` | **11** |
| P22 | `iter.Seq` has exactly 4 in-repo uses, all in the LC-1 spike | `grep -rn 'iter\.Seq' --include='*.go' internal/ cmd/ tools/` | **4**, all under `tools/internal/spike-listrep/`. `go.mod` declares `go 1.26.6` |
| P23 | `.claude/worktrees/` holds **7 full checkouts** of this repo, but neither instrument is contaminated | `find . -name mock_context.go` → 8 hits (1 real + 7 worktrees); `go list ./... \| grep -c worktrees` → **0**; `grep -rn '\.Elements' --include='*.go' . \| grep -c worktrees` → **0**, control total **910** | The Go tool skips dot-directories, and BSD `grep -r` did not descend here either. **910 = 903 (internal+cmd) + 7 (spike)**, so the whole-module textual set is fully accounted for |
| P24 | `make test`, `make vet`, `make fmt-check` all reach `tools/`; `make lint` does not; `check-file-sizes` walks only `internal cmd` | read `make/test.mk:27-30`, `make/code-health.mk:68-71`, `:122-130` | `make lint` is `golangci-lint run ./cmd/... ./internal/... ./testutil/...` — **`tools/linters/listrep` is not linted**. `check-file-sizes` `find internal cmd` — the ~500-LOC analyzer is not size-gated. Both are scope facts, not licences: §4 adds the gates explicitly |
| P25 | `.ailang/` is gitignored, but 56 sprint JSONs are tracked — a **new** one needs `git add -f` | `git ls-files .ailang/state/sprints/ \| wc -l` → **56**; `git check-ignore -v` on a fresh probe file → `.gitignore:82:.ailang/` | Controller must force-add this sprint's JSON |
| P26 | Velocity basis: non-zero Go-LOC commits, last 14 days | per-commit `git show --numstat -- '*.go'`, added+deleted, sorted | **72** non-zero commits, band **2 – 1,696**, median **≈ 330**. Every milestone below (250–430) sits inside the band |

### P27–P31 — the `go/packages` load contract, measured with a scratch probe

These five rows are the reason §2 contains three defects. They were measured with a **throwaway module at `/tmp/iter252scratch/pkgprobe`** (its own `go.mod` requiring `golang.org/x/tools v0.48.0`) whose `packages.Config.Dir` points at this repo. **Nothing was installed into `~/go/bin` and `go.mod`/`go.sum` were checksum-verified unchanged afterwards** (`shasum -c` → both `OK`; `git status --porcelain` shows only the 5 pre-existing known-dirty paths).

| # | Claim | Command | Observed |
|---|---|---|---|
| P27 | `packages.Load("./...")` with `Tests:false` — the **default** | probe, Mode `NeedName\|NeedFiles\|NeedSyntax\|NeedTypes\|NeedTypesInfo\|NeedDeps\|NeedImports\|NeedCompiledGoFiles` | **149 packages, 0 with errors, 1,236 syntax files, 1,236 distinct, 0 duplicated.** 6.21 s cold / **1.43 s warm** |
| P28 | Same load with `Tests:true` | probe | **386 packages, 0 with errors, 3,559 syntax files but only 2,377 distinct — 1,182 files are loaded MORE THAN ONCE.** **1.65 s warm** |
| P29 | **`go build ./...` rc=1 (P1) does NOT imply a dirty type-check.** Both failing packages load cleanly — the failure is at link time | probe `withErrors` field, both modes | **0** packages with errors in both modes. So "all packages load without errors" is a **valid, non-vacuous floor at base** |
| P30 | **Build-tag blindness, measured.** Under the host GOOS the analyzer sees 1 of `cmd/wasm`'s 5 files | probe `./cmd/wasm/...`, host env vs `Env: GOOS=js GOARCH=wasm` | host: **1 package, 1 syntax file** (`effects_helpers.go`), 0.24 s. GOOS=js: **1 package, 5 syntax files** (`effects.go`, `effects_cognition.go`, `effects_helpers.go`, `main.go`, `trace.go`), **0 errors**, 8.14 s cold |
| P31 | Those hidden files hold **3 type-confirmed `*eval.ListValue.Elements` sites** | `grep -c '\.Elements'` over `//go:build js && wasm` files; read the enclosing switch arms | `cmd/wasm/effects.go` **2** (`:438`, `:439`) and `cmd/wasm/main.go` **1** (`:508`), each inside a literal `case *eval.ListValue:` arm. Control: **10** js/wasm-tagged files exist repo-wide |
| P32 | **No PR-triggered workflow builds wasm**, so those 3 sites have no CI signal at all | `grep -rln 'wasm' .github/workflows/` → `release.yml`, `docusaurus-deploy.yml` **only** (control: `grep -rn 'wasm' .github/workflows/ \| wc -l` = 33, i.e. the instrument fires); `grep -c 'wasm' .github/workflows/ci.yml` → **0** with same-file control `grep -c 'ailang'` → **24**; `make/build.mk:103` = `GOOS=js GOARCH=wasm … ./cmd/wasm` | `make build-wasm` runs **only** in `release.yml` (trigger: `push` on tag `v*`) and `docusaurus-deploy.yml` (trigger: `push` to main/dev **with a `docs/**`-style paths filter**). ⚠ my own first pass here grepped only `ci.yml`, got 0, and would have been wrong about the whole `.github/` tree — the 33-hit control is what caught it |

---

## 2. Defects found in the design doc

Per the mission's standing plan/doc-mismatch rule these are **named, not silently resolved**. Each states the resolution this plan takes. In every case the stricter reading is taken and no threshold moves. **If the evaluator disagrees with a resolution, the doc wins and this plan is wrong** — that is the intended failure mode.

### DEFECT-1 (BLOCKING for M2) — the `packages.Config.Tests` flag is unspecified and it moves the headline census by ~380 sites

The whole programme is denominated in this analyzer's count. `packages.Config.Tests` **defaults to `false`** (x/tools v0.48.0 `go/packages/packages.go:217`), in which case `_test.go` files are never loaded. Measured (P17): **380 of 903** `.Elements` references and **291 of 388** `ListValue{` constructions are in `_test.go`. The doc never states the flag. Its round-2 quorum disposition (L28) *asserts* "the analyzer's `./...` scan includes them [test files]" — that is an unverified assumption about an unstated flag, and it is **false by default**.

**Resolution: `Tests: true`, mandatory, with a measured floor (§4 rule 6).** Rationale: LC-3's migration lanes must migrate test files too — LC-4 deletes the field, and 291 composite literals in tests would break the build regardless of whether a census counted them. A census that omits 42% of the sites is not "the TRUE migration count"; it is a differently-contaminated one.

### DEFECT-2 (consequence of DEFECT-1) — `Tests: true` double-loads 1,182 files; a naive count inflates the baseline

Measured (P28): `Tests:true` yields **386 packages / 3,559 syntax files / 2,377 distinct files / 1,182 files seen more than once.** `go list -test ./internal/eval` shows why:

```
github.com/…/internal/eval
github.com/…/internal/eval.test
github.com/…/internal/eval [github.com/…/internal/eval.test]      ← non-test files AGAIN
github.com/…/internal/eval_test [github.com/…/internal/eval.test]
```

The `pkg [pkg.test]` variant re-compiles every **non-test** file of the package. A per-`ast.File` walk therefore visits `internal/eval/value.go` twice, and the exact-match per-package baseline would be keyed on **three or four IDs per real package**.

**Resolution (§4 rule 7):** the analyzer (a) de-duplicates every reported site by absolute `token.Position` (file + offset), and (b) folds `pkg [pkg.test]`, `pkg_test [pkg.test]` and `pkg.test` back onto the base import path before aggregating, so `baseline.json` has **exactly one entry per real package**. Both properties get their own fixture and their own mutation (MUT-13, MUT-14).

### DEFECT-3 (a blind spot the doc's own scope argument forbids) — build-tag exclusion

The doc's scope ruling is argued from a correct principle: *"a scope restriction is invisible to a ratchet BY CONSTRUCTION."* That principle applies verbatim to **build-constraint** restriction, and the doc does not apply it. Measured (P30/P31/P32):

- Under the host GOOS, `go/packages` sees **1 of 5** files in `cmd/wasm`. Under `GOOS=js GOARCH=wasm` it sees all 5, cleanly (0 errors).
- The 4 hidden files hold **3 `*eval.ListValue.Elements` sites** (`effects.go:438,439`, `main.go:508`), each inside a literal `case *eval.ListValue:` arm — so they are type-confirmed, not textual candidates.
- **No PR-triggered workflow builds wasm.** `make build-wasm` runs only on a `v*` **tag push** (`release.yml`) and on a docs-path push (`docusaurus-deploy.yml`).

Consequence if unaddressed: the census under-reports by exactly 3 known sites; Rule 2 is permanently blind under `js/wasm`; the ratchet can never see a new `.Elements` added there; and **at LC-4 the field deletion breaks the build at tag-cutting time with zero prior signal** — the worst possible discovery point.

**Resolution (§4 rule 8, M2 scope):** the driver runs a **second `packages.Load` with `Env` extended by `GOOS=js GOARCH=wasm`, scoped to `./cmd/wasm/...`**, and merges its sites into the same per-package baseline entry. Measured cost 8.1 s cold, ~0.3 s host-side — negligible against the budget in §3. The baseline records the GOOS/GOARCH of each pass so a future drift is a visible diff. What this still cannot see is enumerated in §4 rule 9 and printed by `listrep -help`.

### DEFECT-4 — L9 / HID understate the `x/tools` change

L9 says x/tools "is NOT a direct module requirement" and the HID row says it "becomes a direct dependency" — phrasing that implies promoting an existing `// indirect` line. Measured (P4): `grep -c 'x/tools' go.mod` → **0** (control `golang.org/x` → **8**), and **zero** Go files import it. This is a **fresh module addition**: a new `require` entry, new `h1:` lines in `go.sum`, a new **govulncheck** surface (`ci.yml:509`, reads `.govulncheck-allow.yml`) and a new **dependabot** surface (`.github/dependabot.yml:4-5`, `gomod` at `/`). **Resolution:** M2's acceptance criteria gate on all four (§5 M2). Mitigating fact (P5): `v0.48.0` is already in the module cache, so `make test`'s offline leg is unaffected.

### DEFECT-5 — the `make/ci.mk` edit diverges from the doc's own precedent

The doc cites L12 ("gates ship with their own gate-self-test CI step") and then plans a `make/ci.mk` `ci:` edit. Measured (P9): **not one** existing `check-*` gate is in `ci:`. Adding `check-listrep` would make it the only one. **Resolution: the doc wins — the `ci.mk` edit is made as specified — but it is explicitly NOT an acceptance criterion.** AC-8 gates on the **ci.yml steps** (P8), per the doc's own L11 reasoning.

### DEFECT-6 — AC-8 is not verifiable inside the sprint

AC-8 requires "the PR's own CI run shows … steps executing (job log evidence)". That evidence cannot exist before the PR is opened and CI has run, so as written AC-8 can never be green at the sprint's last commit. **Resolution: split, no weakening.**
- **AC-8a (in-sprint, machine-checkable):** the two steps exist in `ci.yml` with exact `run:` strings, proven by a YAML parse (not a grep), **and** both targets are locally rc=0. Full command in §5 M5.
- **AC-8b (handoff obligation, post-PR):** the executor records the two job-log line references in the implemented report. Listed in §9 as a **release-blocking** item, not an in-sprint gate.

### DEFECT-7 — AC-2 pins a representation artifact, which is exactly what AC-3 was rewritten to stop doing

Round-2 objection 3 (`gpt5-6-sol`, CONFIRMED) rewrote AC-3 because pinning post-transfer caller mutation "makes a representation artifact part of the tested API and knowingly requires a behavioral contract change during the swap." **AC-2 does the same thing and survived:** it requires `&tail.Elements[0] == &src.Elements[k]`. After LC-4 there is no backing array and no `Elements` field, so that assertion must be **deleted**, not adapted.

**Resolution: the doc wins — AC-2 is implemented in full — but the assertions live in a dedicated, clearly-labelled `internal/eval/value_list_reprpin_test.go`** whose header states `DELETE AT LC-4: pins the slice representation, not the API`, and the file is named as an LC-4 obligation in the implemented report. This makes the future deletion a one-file operation instead of a hunt, and stops the pin from silently becoming an API promise. **No assertion is dropped or weakened.**

### DEFECT-8 — two off-by-one AC cross-references

Doc line **459** ("byte-identical output (AC-10)") and line **502** ("**Regression-surface**: AC-10's five fixtures") both point at AC-10 (*census handoff*) when they mean **AC-9** (*zero behaviour change*, line 545, which is what actually names the five fixtures). Line 595 uses AC-9 correctly. Cosmetic, but it is a doc/AC disagreement of the exact kind the mission asks planners to surface. **Resolution: this plan maps the five fixtures to AC-9 throughout (§7).**

### DEFECT-9 — the exact-match ratchet has no defined behaviour for a package that *vanishes* from the scan

Exact-match correctly reds when a count moves in either direction, and correctly reds if the whole scan collapses to 0 (`0 ≠ baseline`). But a package that **fails to load, is renamed, or is excluded by a config change** simply stops appearing. A comparison keyed on *scanned* packages would then skip it silently — a stale-baseline hole in a design whose whole point is that stale baselines hide violations. **Resolution (§4 rule 10):** the comparison is a **two-way set difference** with three distinct, separately-named failure classes — `NEW_SITES`, `STALE_BASELINE` (count decreased), `MISSING_PACKAGE` (in baseline, absent from scan) — plus `EXTRA_PACKAGE`. Each has its own mutation (MUT-15, MUT-16).

### DEFECT-10 — the self-test's fixture location is unspecified, and a missing fixture directory must not be a skip

The driver "runs the analyzer over its own `testdata/` fixture module" on every run. `analysistest.TestData()` resolves `os.Getwd()+"/testdata"`, which is correct in a test binary and **wrong** in a driver invoked from the repo root by `make`. If the path resolves to nothing, the natural failure is "0 fixtures, 0 expected diagnostics, all matched" — a **green self-test on an absent instrument**. **Resolution (§4 rule 5):** the driver resolves its fixture root from `go env GOMOD`'s directory, and a fixture root that is missing, unreadable, or yields **fewer than the pinned fixture-file count** is `exit 3, instrument failure, no verdict` — never a pass, never a skip. Mutation MUT-17.

### Non-defects checked and cleared

- **Rule 2 goes in green:** re-measured without the `_test.go` filter (P18) — 2 hits, both parser AST nodes, 0 in tests. `copy`-rooted: 0 with a firing control; `append`-first-arg-rooted: the same 2 (P19). The doc's L13/L28 stand.
- **`x/tools` availability offline:** P5/P6. No network dependency introduced into `make test`.
- **Symbol collisions:** P15. Clear.
- **Worktree contamination of the textual baselines:** P23. `910 = 903 + 7`, fully accounted for; the Go tool skips dot-dirs.
- **Analyzer CI cost:** the doc's Risks table budgets "~<2 min … vettool fallback". Measured (P27/P28): **1.4 s / 1.7 s warm, 6.2 s cold.** The risk is over-provisioned by roughly two orders of magnitude; §8 retires it and removes the vettool fallback from the critical path.

---

## 3. Execution environment and measurement posture

**Platform of record:** `go1.26.6 darwin/arm64`, main checkout `/Users/voightkampff/dev/sunholo-data/ailang`.

**GPU: not used. `rig.lock` is NOT taken and MUST NOT be taken.** This sprint is pure Go compile/type-check work — no model load, no ollama call. `rig.lock` is a GPU mutex only; holding it would block the eval consumers for nothing.

**Binary hygiene (hard constraint).** `make quick-install` and any write into `~/go/bin` are **forbidden** — that binary is shared with concurrent agents on this rig. Every binary this sprint builds goes to `./bin/` or a scratch dir and is invoked by path or via a locally-prepended `PATH`. The `listrep` make targets must invoke `go run ./tools/linters/listrep/cmd/listrep` or a `./bin/`-scoped build, never an installed name.

**Known-dirty paths to leave alone:** `docs/static/benchmarks/os/{history,latest}.json` (nightly-owned) and three untracked `tools/eval/*.sh` scripts. All path-disjoint from this sprint.

**Wall-clock budget.** The measured full-module type-check is 1.4–1.7 s warm / 6.2 s cold (P27/P28) plus ~8 s for the cold `GOOS=js` pass (P30). Per-PR CI cost of `check-listrep` is therefore expected in the **single-digit-seconds** range on a warm build cache. The executor records the **actual** wall-clock of the first full run in the implemented report; that number, not this estimate, is what LC-3 and LC-4 plan against.

---

## 4. Standing rules for every milestone

1. **One bisectable commit per milestone.** At every milestone boundary: `gofmt -l internal/eval cmd/ailang tools/linters` prints nothing; `go vet ./internal/eval/... ./tools/linters/...` rc=0; `go build ./internal/eval/... ./tools/linters/...` rc=0; the milestone's own test package is green with a non-zero enumerated test count (rule 3).
2. **`go build ./...` is NEVER an acceptance gate** (P1: rc=1 on pristine `dev`). Scoped builds only.
3. **Every `-run`-shaped criterion carries an enumeration floor, written into the criterion.** `go test -run` exits **0** when the selector matches nothing (P11) — identically to a real pass (P12) — so the exit code alone is green before a single test is written and stays green after a rename orphans the selector. The mandatory shape, with the expected count **stated as a literal**:

   ```bash
   go test <pkg> -run '<selector>' -count=1 -v > "$LOG" 2>&1; RC=$?
   N=$(grep -c '^=== RUN   Test[^/]*$' "$LOG")     # top-level only; excludes subtests (P12)
   if [ "$N" -eq 0 ]; then echo "INSTRUMENT FAILURE: selector matched NO tests"; exit 3; fi
   if [ "$N" -ne <EXPECTED_N> ]; then echo "INSTRUMENT FAILURE: expected <EXPECTED_N>, ran $N"; exit 3; fi
   if [ "$RC" -ne 0 ]; then echo "TEST FAILURE"; exit 1; fi
   ```
   `N == 0` is an **instrument failure (exit 3)**, never a pass. `<EXPECTED_N>` must equal the number of test names in the selector. V1 currently carries 54 floor-less `-run` invocations across 16 files in `design_docs/planned`; this design doc has **zero** and this plan introduces **zero**.
4. **`refs #676`** in every commit message; never `Fixes` — LC-2 does not fix the OOM. The final commit adds `refs #745`.
5. **The self-test is a gate, not a decoration.** Fixture root resolved from `go env GOMOD`; missing/unreadable/short fixture set ⇒ **exit 3, "instrument failure", no verdict**. Never skip, never pass (DEFECT-10).
6. **Load contract, part 1 — `Tests: true`, floored.** Every `packages.Load` sets `Tests: true` (DEFECT-1) and the driver asserts, before emitting any verdict: `packages ≥ 386`, `distinct compiled files ≥ 2377`, `packages with load errors == 0` — all three measured at base (P28/P29). Any breach ⇒ **exit 3**. An empty or shrunken scan set can never print a checkmark.
7. **Load contract, part 2 — dedup and fold.** Sites are de-duplicated by absolute `token.Position`; `pkg [pkg.test]`, `pkg_test [pkg.test]` and `pkg.test` are folded onto the base import path before aggregation, so `baseline.json` has exactly one entry per real package (DEFECT-2).
8. **Load contract, part 3 — the wasm pass.** A second load with `Env` extended by `GOOS=js GOARCH=wasm`, scoped to `./cmd/wasm/...`, merged into the same per-package entry. The baseline records each pass's GOOS/GOARCH (DEFECT-3).
9. **The enumerator's declared blind spots.** `listrep -help` and the baseline's `blind_spots` field state, verbatim, what the scan **cannot** see, so no future reader mistakes the census for exhaustiveness:
   - **Other build-constraint variants.** Only the host platform and `js/wasm` are scanned. `//go:build release` (`gen/main/debug_types_release.go`), `windows`, `linux`-only files and any future tag are unseen. Measured today: `.Elements` in `gen/` = **0** (control: 7 `package main` in `gen/`).
   - **Nested modules.** `examples/sim_stub/go.mod` makes 8 `.go` files invisible to `./...`. Measured `.Elements` in `examples/` = **0** (control: 18 `package ` declarations).
   - **Dot-directories.** The 7 checkouts under `.claude/worktrees/` are skipped by the Go tool (P23). This is correct, and it is recorded so nobody "fixes" it.
   - **Generated code is *in* scope, not out** — `gen/main` loads cleanly and is ratcheted like any other package.
   - **Case sensitivity / vendoring:** identifiers are compared via `types.Named` identity, never text, so case is a non-issue; there is no `vendor/` directory (control: `go list ./...` = 149 packages, none under `vendor`).
   - **Aliased writes** (`s := lv.Elements; s[0] = x`) — Rule 1 counts the alias, Rule 2 cannot see the write. Already named in the doc's Unverified table; repeated here because it belongs next to the other blind spots.
10. **Ratchet comparison is a two-way set difference** with four separately-named failure classes: `NEW_SITES`, `STALE_BASELINE`, `MISSING_PACKAGE`, `EXTRA_PACKAGE` (DEFECT-9).
11. **Mutation protocol (used by §6).** Neuter a branch with `if false && <original cond>` — **never** by deleting the block — so every identifier stays used and "the mutant does not build" can never masquerade as "the guard fired". For each mutation, in order, all captured unpiped:
    a. record `sha256` of the target file **before**; apply; record `sha256` **after**; **assert they differ** (mutation LANDED);
    b. `go build ./tools/linters/...` rc=0 (mutant BUILDS);
    c. only then run the named test and assert it is **RED**;
    d. restore from the pre-mutation copy and assert `sha256` matches the "before" value **and** the test is GREEN again.
    A mutation whose (a) or (b) fails is an **instrument failure**, not a passing mutation.
12. **No branch, no commit, no stash, no checkout in the main checkout by the planner.** The executor works on `sprint/iter252-list-accessor-api`.

---

## 5. Milestones

### M1 — The accessor seam (`internal/eval/value_list.go`)
**Closes:** AC-1, AC-2, AC-3 · **first checkpoint of** AC-9
**Estimated:** ~390 LOC (≈130 impl + ≈260 test) · **~0.75 day** · **Depends on:** nothing

Zero dependency on the analyzer or on `x/tools` — this milestone is committable and shippable on its own, and if the sprint is interrupted it leaves the tree strictly better off.

| File | Contents |
|---|---|
| `internal/eval/value_list.go` (new) | 6 methods + `Type`/`String` untouched in `value.go`; package-level `NewList`, `EmptyList`, `Cons`. Every doc comment carries **both** costs (slice today → C1 after LC-4), per A9 |
| `internal/eval/value_list_test.go` (new) | AC-1 and AC-3: shapes {0, 1, 2, 47, 1000}; `Len`, `At` in/out of bounds, `All` incl. early break, `ToSlice` deep-equal **and non-aliasing**, `Uncons`/`DropPrefix` value equality, `Cons` output equals `listConsImpl`'s |
| `internal/eval/value_list_reprpin_test.go` (new) | **AC-2 only.** Header: `DELETE AT LC-4 — pins the slice representation, not the API.` Holds every `&tail.Elements[0] == &src.Elements[k]` assertion (DEFECT-7) |

Surface, verbatim from the doc (no re-derivation): `Len() int` · `At(int) (Value, bool)` · `All() iter.Seq[Value]` · `ToSlice() []Value` · `Uncons() (Value, *ListValue, bool)` · `DropPrefix(int) (*ListValue, bool)` · `NewList([]Value) *ListValue` · `EmptyList() *ListValue` · `Cons(Value, *ListValue) *ListValue`. `DropPrefix` returns `(nil,false)` for `k<0 || k>Len()`. `EmptyList` returns a **fresh** value. `NewList` **consumes ownership**; the doc comment carries `gpt5-6-sol`'s verbatim contract text; **no test exercises post-transfer caller mutation** (AC-3, round-3 carve-out). This is the repo's first `internal/` use of `iter.Seq` (P22).

**Acceptance criteria** — every command, every rc unpiped:
- `go build ./internal/eval/...` → **rc=0**. *On pristine dev: rc=0 (P2)*, so this criterion is not red at base.
- `go vet ./internal/eval/...` → **rc=0**. *On pristine dev: rc=0 (P3).*
- `gofmt -l internal/eval` → **0 lines**. *On pristine dev: 0 (P3).*
- **AC-1 + AC-3, with enumeration floor:**
  ```bash
  go test ./internal/eval -run '^(TestListAccessor_Len|TestListAccessor_At|TestListAccessor_All|TestListAccessor_ToSlice|TestListAccessor_Uncons|TestListAccessor_DropPrefix|TestListAccessor_NewList|TestListAccessor_EmptyList|TestListAccessor_Cons|TestListAccessor_ConsMatchesBuiltin)$' -count=1 -v > "$L" 2>&1; RC=$?
  N=$(grep -c '^=== RUN   Test[^/]*$' "$L")
  [ "$N" -eq 0 ] && { echo "INSTRUMENT FAILURE: selector matched NO tests"; exit 3; }
  [ "$N" -eq 10 ] || { echo "INSTRUMENT FAILURE: expected 10 top-level tests, ran $N"; exit 3; }
  [ "$RC" -eq 0 ] || exit 1
  ```
  *On pristine dev this is **exit 3** (N=0) — correctly an instrument failure, not a pass (P11).*
- **AC-2 (aliasing pins), with enumeration floor:**
  ```bash
  go test ./internal/eval -run '^(TestListAccessor_ReprPin_UnconsAliases|TestListAccessor_ReprPin_DropPrefixAliases|TestListAccessor_ReprPin_ToSliceDoesNotAlias)$' -count=1 -v > "$L" 2>&1; RC=$?
  N=$(grep -c '^=== RUN   Test[^/]*$' "$L")
  [ "$N" -eq 0 ] && { echo "INSTRUMENT FAILURE: selector matched NO tests"; exit 3; }
  [ "$N" -eq 3 ] || { echo "INSTRUMENT FAILURE: expected 3 top-level tests, ran $N"; exit 3; }
  [ "$RC" -eq 0 ] || exit 1
  ```
- **AC-9 first checkpoint:** `git diff --name-only $(git merge-base origin/dev HEAD) -- internal/eval/` lists **only** the three new `value_list*.go` files — no existing file under `internal/eval` is edited. *On pristine dev: 0 lines (empty diff), so this criterion must be paired with the firing control* `git diff --name-only $(git merge-base origin/dev HEAD) | wc -l` *being non-zero, or it is vacuous.*
- `make test` rc=0 (whole suite; the seam is additive and has no production callers).

**Risks:** low. `iter.Seq` is proven to compile in this repo at `go1.26.6` (P22, and LC-1 built four of them).

---

### M2 — `x/tools` dependency, the load contract, and Rule 1
**Closes:** AC-7 · **establishes** the census instrument · **Depends on:** nothing (parallelisable with M1)
**Estimated:** ~430 LOC (≈250 analyzer + ≈180 fixtures/tests) · **~1.25 day**

This is where DEFECT-1, DEFECT-2, DEFECT-3 and DEFECT-4 are discharged. It is the largest milestone and the one most likely to overrun; it is deliberately placed before Rule 2 so a load-contract mistake is caught while only one rule depends on it.

Files: `tools/linters/listrep/listrep.go`, `listrep_test.go`, `testdata/src/fixture/…`, plus `go.mod`/`go.sum`.

Rule 1 reports three classes, tagged in the diagnostic: **(a)** `SelectorExpr` `x.Elements` where `x` type-checks as `…/internal/eval.ListValue` or `*ListValue` (aliasing reads included); **(b)** `CompositeLit` of type `eval.ListValue` — keyed, positional, or empty; **(c)** exemptions — the **function-identifier allowlist** (`eval.NewList`, `eval.EmptyList`, `eval.Cons`, and the `value_list.go` accessor bodies; package path + name, **not** file paths), which is **Rule 1's alone**, plus the **package-path exemption** for `…/tools/internal/spike-listrep`, which applies to **both** rules. Discrimination is by `types.Named` identity, never text.

**Acceptance criteria:**
- **Dependency (DEFECT-4), four gates:**
  `grep -c 'golang.org/x/tools' go.mod` → **≥1** *(pristine dev: **0**, control `golang.org/x` → 8 — P4, so this criterion is correctly red at base and green only after the addition)*; `go mod verify` rc=0; `git diff go.sum | grep -c '^+.*golang.org/x/tools.*h1:'` → **≥1**; and the new dep is declared to the two surfaces it enters — a `.govulncheck-allow.yml` review note and confirmation that `.github/dependabot.yml`'s `gomod` `/` entry (line 4-5) already covers it (**no dependabot edit needed** — verify, don't assume).
- `go build ./tools/linters/...` → **rc=0**; `go vet ./tools/linters/...` → **rc=0**; `gofmt -l tools/linters` → 0 lines. *All three are vacuously rc=0 on pristine dev because the tree is ABSENT (P7)* — so each is paired with `test -d tools/linters/listrep || exit 3` in the same criterion.
- **Load-contract floors (rule 6), asserted by a real test, not by inspection:**
  ```bash
  go test ./tools/linters/listrep -run '^(TestLoadContract_TestsTrue|TestLoadContract_MinPackages|TestLoadContract_ZeroLoadErrors|TestLoadContract_DedupByPosition|TestLoadContract_FoldTestVariants|TestLoadContract_WasmPass)$' -count=1 -v > "$L" 2>&1; RC=$?
  N=$(grep -c '^=== RUN   Test[^/]*$' "$L"); [ "$N" -eq 0 ] && { echo "INSTRUMENT FAILURE"; exit 3; }
  [ "$N" -eq 6 ] || { echo "INSTRUMENT FAILURE: expected 6, ran $N"; exit 3; }; [ "$RC" -eq 0 ] || exit 1
  ```
  `TestLoadContract_MinPackages` asserts `≥386` packages and `≥2377` distinct compiled files (P28); `TestLoadContract_ZeroLoadErrors` asserts 0 (P29); `TestLoadContract_FoldTestVariants` asserts `baseline` keys contain no `[` or `.test` and that `internal/eval` appears **exactly once**; `TestLoadContract_WasmPass` asserts the `GOOS=js` pass over `./cmd/wasm/...` yields **5** syntax files vs the host pass's **1** (P30) and finds **exactly 3** class-(a) sites (P31).
- **AC-7 (false-positive controls), with enumeration floor.** `analysistest` fixtures for `ArrayValue.Elements`, `TupleValue.Elements` and an `ast`-style `Elements` struct must produce **zero** diagnostics:
  ```bash
  go test ./tools/linters/listrep -run '^(TestRule1_Classes|TestRule1_Exemptions|TestRule1_Decoys)$' -count=1 -v > "$L" 2>&1; RC=$?
  N=$(grep -c '^=== RUN   Test[^/]*$' "$L")
  [ "$N" -eq 0 ] && { echo "INSTRUMENT FAILURE: selector matched NO tests"; exit 3; }
  [ "$N" -eq 3 ] || { echo "INSTRUMENT FAILURE: expected 3 top-level tests, ran $N"; exit 3; }
  [ "$RC" -eq 0 ] || exit 1
  ```
- **First full scan recorded** (not yet committed as a baseline): per-package counts printed with the host and wasm passes separated, and the total compared against the textual **903 / 388** upper bounds (P16) with the delta explained in the commit message.

**Risks:** MEDIUM. The `pkg [pkg.test]` fold (DEFECT-2) is fiddly and is the single most likely source of a wrong headline number. Mitigated by MUT-13/MUT-14 and by `TestLoadContract_FoldTestVariants` being a hard AC.

---

### M3 — Rule 2: structural write-context validation
**Closes:** AC-13 · **Depends on:** M2
**Estimated:** ~280 LOC (≈150 rule + ≈130 dual-config fixtures) · **~0.75 day**

Rule 2 is **permanent, zero-tolerance, never ratcheted**, config-driven by `(type, field)` pairs — at LC-2 exactly `(eval.ListValue, Elements)`. Its exemption set is **independent of Rule 1's**: no blanket accessor exemption, no whole-function constructor exemption. Reported write shapes: field assign; index/slice-into-field assign; compound assign; `IncDecStmt` on an element; address-take (`&lv.Elements`, `&lv.Elements[i]`); `copy(lv.Elements…, …)`; `append(x, …)` where the **first** argument is rooted at the field selector. `append(dst, lv.Elements...)` — the spread **read** — is not flagged.

**Composite-literal initializers are NOT a Rule-2 class at LC-2** (`gemini-3-1-pro`'s verbatim round-2 fix; 388 sites would break CI on day one — re-measured P16). Rule 1 tracks them; **LC-4 adds the class to Rule 2 when it retargets.** AC-13's dual-config fixtures already exercise it under a simulated LC-4 cell config, so LC-4 inherits a proven fixture.

The single accepted write context is **structural**: the write's base object is a value **freshly allocated in the same configured-constructor body** (`&ListValue{…}`, `new(ListValue)`, or a composite literal bound to a local). Writes rooted at parameters, receivers, globals, or any previously existing value are rejected **even inside a configured constructor**.

**Acceptance criteria:**
- **AC-13, both configs, with enumeration floor:**
  ```bash
  go test ./tools/linters/listrep -run '^(TestRule2_MutationInAccessorBody|TestRule2_ParamRootedInConstructor|TestRule2_CompositeLitNonConstructor|TestRule2_FreshAllocInConstructor|TestRule2_CellConfig_MutationInAccessorBody|TestRule2_CellConfig_ParamRootedInConstructor|TestRule2_CellConfig_CompositeLitNonConstructor|TestRule2_CellConfig_FreshAllocInConstructor)$' -count=1 -v > "$L" 2>&1; RC=$?
  N=$(grep -c '^=== RUN   Test[^/]*$' "$L"); [ "$N" -eq 0 ] && { echo "INSTRUMENT FAILURE"; exit 3; }
  [ "$N" -eq 8 ] || { echo "INSTRUMENT FAILURE: expected 8, ran $N"; exit 3; }; [ "$RC" -eq 0 ] || exit 1
  ```
  Semantics per the doc: (1) `l.Elements[0] = x` inside a fixture `At` → **MUST report**; (2) `tail.Elements[0] = head` inside a fixture `Cons` → **MUST report**; (3) `&ListValue{Elements: xs}` in a non-constructor → reported by **Rule 1** under the LC-2 config, by **Rule 2** under the cell config; (4) `c := &ListValue{Elements: xs}; return c` inside a configured constructor → **zero** Rule-2 diagnostics.
- **Write-shape coverage, with enumeration floor.** One fixture and one test per reported shape; `append(dst, lv.Elements...)` (the spread **read**) must NOT be flagged:
  ```bash
  go test ./tools/linters/listrep -run '^(TestRule2_Shape_Assign|TestRule2_Shape_IndexAssign|TestRule2_Shape_CompoundAssign|TestRule2_Shape_IncDec|TestRule2_Shape_AddrTakeField|TestRule2_Shape_AddrTakeElem|TestRule2_Shape_CopyRooted|TestRule2_Shape_AppendFirstArgRooted|TestRule2_Shape_SpreadReadNotFlagged)$' -count=1 -v > "$L" 2>&1; RC=$?
  N=$(grep -c '^=== RUN   Test[^/]*$' "$L")
  [ "$N" -eq 0 ] && { echo "INSTRUMENT FAILURE: selector matched NO tests"; exit 3; }
  [ "$N" -eq 9 ] || { echo "INSTRUMENT FAILURE: expected 9 top-level tests, ran $N"; exit 3; }
  [ "$RC" -eq 0 ] || exit 1
  ```
- **Rule 2 goes in green on the real tree:** a full scan reports **0** Rule-2 diagnostics. *Pre-measured as consistent at base: direct assign/index = 2 hits, both `*ast` nodes the type filter excludes; `copy`-rooted = 0 with a firing control; `append`-first-arg-rooted = the same 2 (P18/P19).* If the first full run finds anything else, it is triaged in-sprint per the doc: latent bug → its own commit; genuine constructor-shaped write → restructured to satisfy the check (never loosen the rule).

**Risks:** MEDIUM — this is the doc's own named risk (structural freshness without SSA). Mitigation is the doc's: constructors are ours, three functions in one file, kept in `allocate → initialize → return` shape; if a constructor shape defeats the checker, restructure the constructor.

---

### M4 — Driver, self-test, baseline, ratchet, census
**Closes:** AC-4, AC-6 · **Depends on:** M2, M3
**Estimated:** ~260 LOC · **~0.75 day**

`tools/linters/listrep/cmd/listrep/main.go`: (1) **self-test first, unconditionally, every run** — analyzer over its own fixture module, exact expected diagnostic set, any mismatch ⇒ **exit 3, "instrument failure", no verdict** (fixture root from `go env GOMOD`, missing/short ⇒ exit 3 — DEFECT-10); (2) **scan** host `./...` + `GOOS=js ./cmd/wasm/...`, dedup, fold, floors; (3) **compare** to `baseline.json` as a two-way set difference with four named failure classes (DEFECT-9); (4) `-write-baseline`, deterministic ordering, recording scope string, per-pass GOOS/GOARCH, exemption config, `blind_spots` (rule 9) and analyzer version.

**Acceptance criteria:**
- **AC-4 (true count + hand audit):** `tools/linters/listrep/baseline.json` committed from the first full run; every module package denominated; the spike listed as **exempt, not absent**; `internal/embed`'s count (11 textual references, P21) matches a by-hand audit recorded in the commit message; and **all three known escapes appear in analyzer output** — `internal/builtins/safe_cast.go:97`, `internal/embed/convert.go:346`, `internal/effects/testctx/mock_context.go:353-355` (⚠ note the corrected third path, P20). Any missing escape ⇒ instrument failure.
- **Non-vacuity of the baseline:** `jq '[.packages[].count] | add' baseline.json` > 0 **and** `jq '.packages["github.com/sunholo-data/ailang/internal/builtins"].count' baseline.json` > 0 — a zero where textual grep shows heavy eval-value usage is a red, per the doc's own AC-4 wording.
- **AC-6 (instrument failure is loud):** with the fixture expectation set emptied, `./bin/listrep -self-test-only` exits **exactly 3** (not 0, not 1), rc captured unpiped, and prints `instrument failure`. Paired with the positive control that the unmodified expectation set exits **0** in the same script run.
- **Ratchet direction tests (all four failure classes + the pass case), with enumeration floor:**
  ```bash
  go test ./tools/linters/listrep -run '^(TestRatchet_NewSitesFails|TestRatchet_DecreaseFails|TestRatchet_MissingPackageFails|TestRatchet_ExtraPackageFails|TestRatchet_ExactMatchPasses)$' -count=1 -v > "$L" 2>&1; RC=$?
  N=$(grep -c '^=== RUN   Test[^/]*$' "$L")
  [ "$N" -eq 0 ] && { echo "INSTRUMENT FAILURE: selector matched NO tests"; exit 3; }
  [ "$N" -eq 5 ] || { echo "INSTRUMENT FAILURE: expected 5 top-level tests, ran $N"; exit 3; }
  [ "$RC" -eq 0 ] || exit 1
  ```
- `go test ./tools/linters/...` rc=0 with a non-zero enumerated count.

**Risks:** LOW-MEDIUM. The census number is the programme's denominator; its correctness rests on M2's fold, which is why the hand audit is a hard AC.

---

### M5 — CI wiring, the gate-of-the-gate, and the mutation battery
**Closes:** AC-5, AC-8a, AC-12 · **Depends on:** M4
**Estimated:** ~300 LOC (≈110 gate script + ≈120 mutation harness + ≈70 make/ci) · **~0.75 day**

Edits: `make/code-health.mk` +3 targets (`check-listrep`, `test-listrep-gate`, `listrep-baseline`, each `@bash scripts/…` per P10's precedent); `make/ci.mk` `ci:` aggregate +1 (**made as the doc specifies, explicitly NOT an AC** — DEFECT-5); `.github/workflows/ci.yml` +2 steps following the `check-autoclose` pattern at lines 146-163.

`scripts/test_listrep_gate.sh` — five arms, each with its rc captured unpiped:
(i) seeded extra `.Elements` selector ⇒ non-zero, naming it; (ii) seeded non-constructor assignment ⇒ non-zero; (iii) unmodified fixtures **and real tree** ⇒ zero — the **seeded-ACCEPTED** arm, proving the analyzer is not simply rejecting everything; (iv) `-self-test-only` with emptied expectations ⇒ **exactly 3**; (v) **scope-coverage seed** — create `tools/listrep-gate-seed/` with one `*eval.ListValue.Elements` selector, run the **same `./...` invocation CI uses** ⇒ non-zero naming the seeded site, then delete the seed. Arm (v) is the arm that would have FAILED under the round-1 scope.

**Plan addition (DEFECT-3), arm (vi):** seed one `*eval.ListValue.Elements` selector into a **`//go:build js && wasm`** file under `cmd/wasm/` ⇒ the driver must red naming it, proving the wasm pass looks. Then delete the seed. Without this arm the wasm pass is a claim.

**Acceptance criteria:**
- **AC-5:** arms (i)–(iii) pass. **AC-12:** arm (v) passes. **New:** arm (vi) passes. Each arm asserts a **specific** expected rc and a **substring of the seeded site's path** in the output — an arm that merely checks "non-zero" would pass on a crash.
- **AC-8a (DEFECT-6), gating on the ci.yml step, not the make target:**
  ```bash
  python3 -c "import yaml,sys; d=yaml.safe_load(open('.github/workflows/ci.yml'));
  runs=[s.get('run','') for j in d['jobs'].values() for s in (j.get('steps') or [])];
  need=['make check-listrep','make test-listrep-gate'];
  missing=[n for n in need if not any(n in r for r in runs)];
  print('MISSING:',missing) or sys.exit(1) if missing else sys.exit(0)"
  ```
  *On pristine dev: exit 1 with `MISSING: ['make check-listrep', 'make test-listrep-gate']` — correctly red at base.* Same-run control: the identical parse finds `make check-changelog`, which exists today (P10), proving the parser sees the file. Plus `make check-listrep` rc=0 and `make test-listrep-gate` rc=0 locally.
- **The mutation battery of §6 passes in full** — every mutation LANDED (sha256 differs), BUILDING (`go build ./tools/linters/...` rc=0), and its named test RED; then restored and GREEN.

**Risks:** LOW. The one real hazard is arm (v)/(vi) leaving seed files behind on a failure path — the script must `trap` cleanup and the milestone's exit criterion includes `git status --porcelain` showing only the 5 known-dirty paths.

---

### M6 — Census report, escape classification, changelog, handoff
**Closes:** AC-9, AC-10, AC-11 · **Depends on:** M1, M5
**Estimated:** ~0 Go LOC + ~250 markdown lines · **~0.25 day**

**Acceptance criteria:**
- **AC-9 (zero behaviour change):** `make test` rc=0; `make verify-examples` rc=0; the five named fixtures (`examples/first_non_repeat.ail`, `inline_tests_recursive.ail`, `record_cons_pattern.ail`, `record_list_extraction.ail`, `pattern_matching_adt.ail`) byte-identical to merge-base output — captured by running each at merge-base and at HEAD and `diff`-ing, **with the firing control** that a deliberately altered expected file *does* produce a diff; and `git diff --name-only $(git merge-base origin/dev HEAD) -- internal/eval/` lists only the three new `value_list*.go` files. *(Note: the doc cross-references these fixtures as "AC-10" at lines 459 and 502 — DEFECT-8; they are AC-9's.)*
- **AC-10 (census handoff):** the implemented report contains the per-package analyzer table **and** the escape classification (every Rule-1 site whose selector value flows out of its function as a raw slice), enumerating **≥ the 3 known escapes** with site IDs, explicitly labelled as LC-3a/b/c's sizing input, and stating the wall-clock of the first full run. **No migration AC anywhere in it may be stated in grep counts.**
- **AC-11 (docs):** changelog entry under Unreleased in the current `changelogs/` file (root `CHANGELOG.md` stays an index — `make check-changelog` rc=0, *pristine dev: rc=0, P3*); this design doc moved to `design_docs/implemented/v0_35_0/` with the report.
- **Two LC-4 obligations recorded in the report, not as prose but as a checklist:** (1) add the configured-field composite-literal class to Rule 2 when retargeting (the doc's own carried-forward obligation); (2) **delete `internal/eval/value_list_reprpin_test.go`** — it pins the slice representation, not the API (DEFECT-7).
- **AC-8b recorded:** the two CI job-log line references from the PR's own run.

---

## 6. Mutation battery — one named mutation per refusal branch

The doc's headline verbs are "reject" and "refuse". A single "neuter the analyzer" mutation proves only that *something* fires. Each branch below gets its own. **Protocol: §4 rule 11 — LANDED (sha256 differs) and BUILDING (rc=0) are asserted BEFORE the test result is read.** Neuter with `if false && <cond>`, never by deletion, so every import stays used.

| ID | Refusal branch neutered | Edit | Test that MUST go RED |
|---|---|---|---|
| MUT-1 | Rule 1 class (a) — selector emit | `if false && isListValueSelector(x)` | `TestRule1_Classes` |
| MUT-2 | Rule 1 class (b) — composite-literal emit | `if false && isListValueCompositeLit(n)` | `TestRule1_Classes` |
| MUT-3 | Type-identity discrimination (`types.Named`) | `if false && sameNamed(t, listValueType)` | `TestRule1_Decoys` (decoys start reporting) |
| MUT-4 | Rule 1 function-identifier allowlist | `if false && inConstructorAllowlist(fn)` | `TestRule1_Exemptions` |
| MUT-5 | Spike package exemption (both rules) | `if false && isExemptPackage(pkg)` | `TestRule1_Exemptions` |
| MUT-6 | Rule 2 — plain field assign | `if false && isFieldAssign(lhs)` | `TestRule2_Shape_Assign` |
| MUT-7 | Rule 2 — index/slice-into-field assign | `if false && isIndexIntoField(lhs)` | `TestRule2_Shape_IndexAssign`, `TestRule2_MutationInAccessorBody` |
| MUT-8 | Rule 2 — `IncDecStmt` on an element | `if false && isIncDecOnElem(n)` | `TestRule2_Shape_IncDec` |
| MUT-9 | Rule 2 — address-take of field / element | `if false && isAddrTakeOfField(n)` | `TestRule2_Shape_AddrTakeField`, `..._AddrTakeElem` |
| MUT-10 | Rule 2 — `copy`/`append` first-arg rooted at field | `if false && isRootedAtField(call.Args[0])` | `TestRule2_Shape_CopyRooted`, `..._AppendFirstArgRooted` |
| MUT-11 | Rule 2 — **structural** reject of param/receiver/global-rooted writes inside a configured constructor | `if false && !isFreshlyAllocatedHere(base, fnBody)` | `TestRule2_ParamRootedInConstructor` **and** its cell-config twin |
| MUT-12 | Rule 2 — the one **accepted** branch (fresh alloc in constructor) inverted to reject | `if false && isFreshlyAllocatedHere(...)` on the *exempt* side | `TestRule2_FreshAllocInConstructor` (must red — proves the rule is not simply rejecting everything) |
| MUT-13 | Dedup by `token.Position` | `if false && !seen[pos]` | `TestLoadContract_DedupByPosition` |
| MUT-14 | Test-variant fold onto base import path | `if false && isTestVariant(pkg.ID)` | `TestLoadContract_FoldTestVariants` |
| MUT-15 | Ratchet `STALE_BASELINE` (decrease) branch | `if false && scanned < baseline` | `TestRatchet_DecreaseFails` |
| MUT-16 | Ratchet `MISSING_PACKAGE` branch | `if false && inBaselineNotInScan(p)` | `TestRatchet_MissingPackageFails` |
| MUT-17 | Self-test fixture-root floor | `if false && fixtureCount < minFixtures` | gate arm (iv) — driver must still exit 3; if it exits 0, MUT-17 has proved the floor was decorative |
| MUT-18 | Anti-vacuity scan floors (packages / files / load errors) | `if false && len(pkgs) < minPkgs` | `TestLoadContract_MinPackages` |
| MUT-19 | The `GOOS=js` wasm pass | `if false && wasmPassEnabled` | `TestLoadContract_WasmPass` **and** gate arm (vi) |

**19 mutations, 19 named tests.** Any mutation that does **not** turn its named test red is a defect in the guard, not in the mutation — the sprint does not proceed past M5 until every row is demonstrated.

---

## 7. Milestone → acceptance-criterion mapping

| Doc AC | Milestone | Plan note |
|---|---|---|
| AC-1 slice equivalence | M1 | enumeration floor, `EXPECTED_N=10` |
| AC-2 aliasing pins | M1 | isolated in `value_list_reprpin_test.go`; **LC-4 deletes it** (DEFECT-7) |
| AC-3 construction correctness | M1 | round-3 carve-out honoured: **no** post-transfer caller-mutation test |
| AC-4 true count + hand audit | M4 | third escape path corrected to `internal/effects/testctx/` (P20) |
| AC-5 ratchet fires both directions | M5 | arms (i)-(iii), each asserting a specific rc **and** a path substring |
| AC-6 instrument failure loud | M4 (+M5 arm iv) | exit code **exactly 3**, with a positive control in the same run |
| AC-7 false-positive controls | M2 | `EXPECTED_N=3` |
| AC-8 CI actually gates | **split** M5 / M6 | AC-8a YAML-parse + local rc (in-sprint); AC-8b job-log evidence (handoff) — DEFECT-6 |
| AC-9 zero behaviour change | M1 (checkpoint) + M6 | doc lines 459/502 misname this AC-10 — DEFECT-8 |
| AC-10 census handoff | M6 | must include first-run wall-clock and the escape classification |
| AC-11 docs | M6 | `make check-changelog` rc=0 (green at base, P3) |
| AC-12 gate looks outside internal/+cmd/ | M5 | arm (v) |
| AC-13 Rule-2 structural fixtures | M3 | both configs, `EXPECTED_N=8` |
| **(plan) build-tag coverage** | M2 + M5 | DEFECT-3: wasm pass + gate arm (vi) — **not in the doc** |
| **(plan) load contract** | M2 | DEFECT-1/2: `Tests:true`, dedup, fold, floors — **not in the doc** |
| **(plan) ratchet set-difference** | M4 | DEFECT-9: `MISSING_PACKAGE` / `EXTRA_PACKAGE` — **not in the doc** |

---

## 8. Risks, and the doc risks this plan re-prices

| Risk | Level | Basis / mitigation |
|---|---|---|
| **Analyzer CI cost** — the doc budgets "~<2 min, vettool fallback" | **RETIRED** | Measured 1.4 s (Tests=false) / 1.7 s (Tests=true) warm, 6.2 s cold, plus 8.1 s cold for the wasm pass (P27/P28/P30). Over-provisioned by ~2 orders of magnitude. **The vettool fallback is removed from the critical path**; if it is ever needed the analyzer is already `singlechecker`-shaped |
| **The `pkg [pkg.test]` fold produces a wrong headline number** | **MEDIUM — new, highest** | DEFECT-2. 1,182 files load twice. Mitigated by `TestLoadContract_FoldTestVariants` as a hard AC, MUT-13/MUT-14, and the `internal/embed` hand audit |
| Rule 2's structural freshness check without SSA | MEDIUM | The doc's own risk and its own mitigation: constructors kept in `allocate → initialize → return`; restructure the constructor, never loosen the rule. MUT-11/MUT-12 pin both directions |
| **Build-tag sites reach LC-4 uncounted** | MEDIUM → LOW once M2 lands | DEFECT-3. 3 measured sites, no PR-side CI signal (P32). Fixed by the wasm pass + gate arm (vi) |
| `baseline.json` merge conflicts across concurrent worktrees | MEDIUM | Exact-match means any count-touching PR regenerates. Deterministic key ordering + one-key-per-line JSON makes conflicts textual and small; `make listrep-baseline` regenerates in one command. **A conflict resolved by picking one side is a silent baseline forgery** — so the gate re-runs after any baseline merge and the exact-match comparison catches a mis-resolved file immediately (that is the property exact-match buys over "may only decrease") |
| **Can a stale baseline hide a new violation?** | Answered | **Rule 1: no.** Exact-match reds on any drift in either direction, including a package silently vanishing (DEFECT-9's `MISSING_PACKAGE`). **Rule 2: not applicable** — it is never ratcheted; any diagnostic fails unconditionally, so no baseline can hide it. **The residual is the enumerator, not the ratchet**: a site the scan cannot *see* (build tag, nested module, dot-dir) is not in the baseline and never will be — which is why rule 9 makes the blind spots a printed field rather than a footnote |
| First full Rule-2 run finds `copy`/`append`-rooted hits | **LOW — now measured** | P19: `copy`-rooted **0** (control 83 `copy(` calls fire), `append`-first-arg-rooted **2**, both parser AST nodes the type filter excludes. Triage path predeclared anyway |
| `x/tools` as a new direct dep breaks the offline `make test` leg | LOW | P5: `v0.48.0` already in the module cache with a `.ziphash`; `go.sum` already carries its `h1:` |
| **+0.5 day over the doc's 4-day estimate** | — | Three named line items: DEFECT-1/2 load contract + fold + its 2 mutations (**~0.25 d**), DEFECT-3 wasm pass + gate arm (vi) (**~0.15 d**), the 19-row mutation battery beyond the doc's single neuter (**~0.10 d**). The roadmap band was 3–4 days, so this is a **0.5-day band breach**, surfaced rather than absorbed |

---

## 9. Definition of done

- [ ] Six milestones committed, each independently bisectable, each with its own test package green and a **non-zero enumerated** test count at the boundary.
- [ ] AC-1 … AC-7, AC-9 … AC-13 green by the commands in §5 — every one of which states what it returns **on pristine `dev`**.
- [ ] AC-8a green locally (YAML parse + both make targets rc=0). **AC-8b recorded from the PR's own run — release-blocking, not sprint-blocking.**
- [ ] All **19** mutations in §6 demonstrated: LANDED (sha256), BUILDING (rc=0), named test RED, restored GREEN.
- [ ] `baseline.json` committed with per-package counts, both passes' GOOS/GOARCH, the exemption config, the `blind_spots` field, and the analyzer version.
- [ ] Census table + escape classification in the implemented report, with the first-run wall-clock. **≥3 escapes**, with the corrected `internal/effects/testctx/` path.
- [ ] `make test` rc=0 · `make verify-examples` rc=0 · `make check-file-sizes` rc=0 · `make check-boundaries` rc=0 · `make check-changelog` rc=0 · `gofmt -l internal/eval cmd/ailang tools/linters` = 0 lines. **`go build ./...` is NOT in this list and must never be added** (P1).
- [ ] `git status --porcelain` shows only the 5 pre-existing known-dirty paths — every seed directory from gate arms (v) and (vi) removed.
- [ ] Two LC-4 obligations recorded as a checklist in the report: add the composite-literal class to Rule 2 at retarget; **delete `value_list_reprpin_test.go`**.

---

**SPRINT_PLAN_PATH**: `design_docs/planned/m-list-accessor-api-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_M-LIST-ACCESSOR-API.json`
