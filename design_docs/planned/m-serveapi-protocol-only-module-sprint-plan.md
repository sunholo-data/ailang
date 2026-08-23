# Sprint Plan — M-SERVEAPI-PROTOCOL-ONLY (#764)

**Design doc**: [`m-serveapi-protocol-only-module.md`](m-serveapi-protocol-only-module.md)
**Issue**: [#764](https://github.com/sunholo-data/ailang/issues/764) (`cross-mission`, filed by the ailang-world loop)
**Sprint ID**: `M-SERVEAPI-PROTOCOL-ONLY`
**Target**: v0.34.0 · **Duration**: 3.0 days · **Risk**: medium
**Planned**: 2026-08-23, mission iteration 261

---

## 0. Provenance contract (read this before trusting any number below)

Every codebase claim in this plan is labelled exactly one of:

- **`VERIFIED BY ME`** — re-run by the planner in this worktree during planning, with the command
  shown. Nothing was laundered from the design doc without re-running it.
- **`UNVERIFIED, inherited from <source>`** — carried forward, **re-check before relying**.

**Measurement environment for every `VERIFIED BY ME` row:**
worktree `/Users/voightkampff/dev/sunholo-data/.wt-iter261`, branch `docs/mission-iter261-record`,
HEAD `9ce91ce50`, tree clean, **darwin/arm64**, `go version go1.26.6 darwin/arm64`,
golangci-lint 2.11.4, interactive shell `/bin/zsh` (note: `$PIPESTATUS` is a bashism and is empty
here — rc was captured with `cmd > file 2>&1; echo $?`).

**Platform narrowing — stated once, applies to every gate in this plan:**
all planner measurements are **green on darwin/arm64 only**. The **windows-latest and
ubuntu-latest CI legs were not run locally** and are unrun for this diff. `go list -deps` closures
are GOOS/GOARCH-sensitive (build-tagged files), and the MCP SDK subtree measured below
demonstrably contains per-arch packages (`github.com/segmentio/asm/cpu/{arm,arm64,x86}`), so a
package **count** is not portable and a **module-root set** very likely is. The gate is designed
accordingly (§4.3): the tight assertion is on module roots, the count assertion carries headroom.

**Code drift check.** HEAD `9ce91ce50` is two commits past the design doc's measurement SHA
`a201237ca`. `git diff --stat a201237ca HEAD -- serveapi internal/apiserver cmd/ailang make
.github scripts` → **empty**. Every source file this sprint touches is byte-identical to what the
doc measured. **VERIFIED BY ME** (`git diff --stat a201237ca HEAD -- <paths>`).

---

## 1. Design-doc reproduction: what I re-ran, and what I found wrong

### 1.1 Reproduced exactly

| Doc row | Claim | My result | Command |
|---|---|---|---|
| V1 | closures 479 / 478 / 12 / 14 / 453 | **identical** | `for p in ./serveapi ./internal/apiserver ./cmd/wasm ./cmd/astdump ./cmd/registry-validator; do go list -deps $p \| awk -F/ '$1 ~ /\./ {n++} END{print n+0}'; done` |
| V22 | 325 cloud/telemetry-pattern packages **in the `./serveapi` closure** | **325** | the doc's `grep -Ec` pattern over the non-stdlib lines |
| V21 | 67 module roots **in the `./serveapi` closure** (66 external) | **67** | `go list -deps -f '{{if not .Standard}}{{with .Module}}{{.Path}}{{end}}{{end}}' ./serveapi \| sort -u \| wc -l` |
| V2 | `serveapi/serveapi.go` = 201 lines, one non-stdlib import at line 15 | **identical** | `wc -l`; `grep -n` |
| V3 | 7 `apiserver.*` symbols referenced by `serveapi.go` | **identical** (incl. the `EmbeddedA`/`NewEmbeddedA` truncation the `[A-Za-z]*` regex causes on the `2` in `EmbeddedA2A`) | `grep -o "apiserver\.[A-Za-z]*" \| sort \| uniq -c` |
| V4 | the 4 candidate files' direct imports are stdlib-only except `go-sdk/mcp` in `embedded_mcp.go` | **identical**; the non-stdlib filter fires on exactly one line | `sed -n '/^import (/,/^)/p'` on all four + dot-rule filter |
| V8 | **load-bearing**: re-scoped subset compiles standalone with **zero shims**, closure = itself only | **REPRODUCED FROM THE APPENDIX RECIPE, BYTE-FOR-BYTE**: `BUILD_OK` with no shim file; non-stdlib lines = exactly `…/internal/protoprobe`; count `1`; total enumeration `188`; module roots = `github.com/sunholo-data/ailang` only; `rm -rf` + `go build ./serveapi ./internal/apiserver` → `REPO_STILL_BUILDS`, `git status --short` empty | the appendix's five `sed` extractions verbatim; all line anchors (`authorized_surface.go:28-41`, `mcp_schema.go:11-14`+`103-108`, `serveapi.go:23-35;46-75`, `a2a.go:274-322`, `callbacks.go:10-12`, `embedded_mcp.go:177-236`) still land on the intended code |
| V9 | MCP SDK subtree = 29 external non-stdlib packages, 9 external module roots | **identical**: 29 / 9. Roots: `google/jsonschema-go`, `modelcontextprotocol/go-sdk`, `segmentio/asm`, `segmentio/encoding`, `yosida95/uritemplate/v3`, `golang.org/x/{oauth2,sync,sys,time}` | `go list -deps [-f module] github.com/modelcontextprotocol/go-sdk/mcp` |
| V13/V14/V15 | `serveapi/protocol`, `check-protocol-closure`, ci.yml step all absent at base | **identical**, each with its in-scope control firing (see §3 baseline table) | see §3 |
| V16 | base suite green | **identical** — `go test -count=1 ./serveapi/... ./internal/apiserver/...` rc=0 | see §3 |
| V17 | exactly 2 importing packages / 3 files, module-wide | **identical**: `cmd/ailang`, `serveapi`; files `cmd/ailang/serve_api.go`, `serveapi/serveapi.go`, `serveapi/serveapi_external_test.go`. Scope re-asserted: `pkg`/`test`/`testdata`/`gen` **absent**; `tools`/`examples`/`scripts` **exist** and are inside both instruments' scope | the doc's `go list` + repo-wide `grep -rln` + `test -d` loop |
| V19 | `apiserver.ToolDescriptor\|AuthorizedSurface` referenced outside the package **only** by `serveapi/serveapi.go` (6 hits) | **identical** | repo-wide `grep -rn … \| uniq -c` |
| V20 | stay-behind usage map | **identical once scope is written down**: `writeJSON` in **8** files **in `internal/apiserver`** of which 7 stay (the 8th is `embedded_a2a.go`, which moves); `loadedExportMember` in **7** files of which 6 are non-test (the 7th is `embedded_a2a_test.go` — see §1.2 D1) | `grep -rln <sym> internal/apiserver/*.go` |
| V23 | oracle test is `package serveapi`, 2 `apiserver.` refs (lines 68, 81) | **identical** | `head`; `grep -n` |
| V25 | `RunCallback`/`CallbackRunner` used by nothing outside the embedded path | **identical** (4 non-test files, all named in the doc) | repo-wide `grep -rln` |
| V26 | envelope helpers used by no non-test file outside the two embedded files | **identical**: `2 embedded_a2a.go`, `13 embedded_mcp.go`. Including tests adds only `2 embedded_mcp_replay_test.go` | repo-wide `grep -rn … \| uniq -c` |

All **VERIFIED BY ME**.

### 1.2 What I found WRONG or under-specified in the design doc

Five findings. **D2 is a mechanical defect that would have inverted the sprint's headline
refusal gate** — it is the reason this section exists.

---

**D1 — `embedded_a2a_test.go` CANNOT move to `serveapi` wholesale. The doc says it can.**

The doc's "New homes" list routes `embedded_a2a_test.go` → `serveapi`. But that file contains
`TestLoadedExportMembershipAndNoMCPProjection` (lines **323–344**), which calls the **unexported,
explicitly-stay-behind** `loadedExportMember` and constructs 4 `ExportInfo` literals — both of
which the same design doc keeps in `internal/apiserver` (V19/V20). Go forbids referencing an
unexported identifier from another package, so the move as written **does not compile**.

- **VERIFIED BY ME**: `grep -n 'ExportInfo\|loadedExportMember' internal/apiserver/embedded_a2a_test.go`
  → hits at lines 327, 330, 331, 332, 333, 337, all inside the func beginning at line 323
  (`grep -n '^func Test'` puts the preceding func at 299 and the following at 351).
- **Control, same scope**: the same grep over the other three moving test files
  (`embedded_mcp_test.go`, `embedded_mcp_replay_test.go`, `callbacks_test.go`) returns **empty** —
  and the known-positive control `internal/apiserver/mcp_schema_test.go` (a file that stays)
  returns 13 `ExportInfo` + 13 `isExposed`, so the instrument is not blind.
- **Plan disposition**: M2 **splits** `embedded_a2a_test.go`. `TestLoadedExportMembershipAndNoMCPProjection`
  stays in `internal/apiserver` (new home: append to `mcp_schema_test.go`, which already owns the
  `isExposed`/`ExportInfo` assertions, or a new `authorized_surface_test.go` stub). The other 9
  test funcs + 4 helpers move. Task M2.4 pins this.

---

**D2 — the CI self-test's intruder arm, as specified, DOES NOT FIRE. The instrument cannot see
the mutation it injects.**

The design doc adopted the round-2 reviewer's fix **verbatim**, and the verbatim text names a
**`_test.go`** file:

> `echo 'package protocol; import _ "github.com/google/uuid"' > serveapi/protocol/zz_intruder_test.go`

`go list -deps <pkg>` — the gate's own instrument in arm 1 — enumerates the **build** closure and
**does not include test-only imports**. So injecting a `_test.go` intruder leaves the enumeration
unchanged: the gate exits **0**, the count does not move, and the self-test's own
"expect exit 1 + count movement" assertion fails *for the wrong reason* — the exact failure mode
the reviewer's objection was raised to prevent.

- **VERIFIED BY ME** (scratch package `internal/probetest`, created and removed inside one call;
  `git status --short` empty afterwards):
  - stdlib-only `p.go` + `zz_intruder_test.go` importing `github.com/google/uuid`:
    `go list -deps ./internal/probetest | awk -F/ '$1 ~ /\./'` → **only** `…/internal/probetest`, **count 1**. uuid absent.
  - **Known-positive control in the same call**: `go list -deps -test ./internal/probetest` → **count 4**, `github.com/google/uuid` present. The intruder file *is* real and *is* visible to an instrument that looks for it — so "count 1" is a measurement of `-deps` scope, not a broken probe.
  - Re-run with the intruder as a **non-test** `zz_intruder.go`: `go list -deps` → `github.com/google/uuid` + `…/internal/probetest`, **count 1 → 2**, `go build` still rc=0.
- **Note the doc contradicts itself here**, which is how this was caught: Mutation-table row 1 says
  "add `_ "github.com/google/uuid"` to a `serveapi/protocol` **file**" (correct); the self-test
  spec says `zz_intruder_test.go` (incorrect); and the doc then asserts "the self-test and
  Mutation 1 are now the same experiment". They are not — one fires and one does not.
- **Plan disposition**: the self-test injects **`serveapi/protocol/zz_intruder.go`** (no `_test`
  suffix). The reviewer's *intent* — inject into the **real** package directory so arm 1's
  anti-vacuity floor is satisfied — is preserved exactly; only the filename changes, because a
  filename is what decides whether the gate's instrument can see it. Task M3.4.
- **And the blind spot is then pinned deliberately**: a fourth self-test arm injects the
  `_test.go` form and asserts the gate stays **green**, with the rationale in the message. Without
  that arm, a future reader re-deriving this defect cannot tell "deliberate scope" from "hole".
  Task M3.5. (Scope is correct: the gate models what a *consumer links*, and a consumer never
  links our test-only imports.)

---

**D3 — moving the machinery into `serveapi` would newly PUBLISH 7 symbols on a published seam.
The doc never decides their export status.**

`CallbackRunner`, `NewCallbackRunner`, `RunCallback`, `EmbeddedMCPConfig`,
`NewEmbeddedMCPHandler`, `EmbeddedA2AConfig`, `NewEmbeddedA2AHandler` are exported **inside
`internal/apiserver`**, i.e. invisible to every external consumer. Moving them verbatim into
`serveapi` — which shipped in v0.33.0/v0.33.1 — makes all seven **public API** for the first time.
The issue does not ask for any of them; the doc's own Non-Goals call `protocol` symbols
"cheap to add later and expensive to remove", and the identical asymmetry applies to `serveapi`.

- **VERIFIED BY ME** that nothing forces export:
  - `serveapi/serveapi_external_test.go` is `package serveapi` (in-package test → may use
    unexported identifiers). `head -12` → `package serveapi`.
  - The only other importer of `internal/apiserver` is `cmd/ailang/serve_api.go` (V17), which does
    not import `serveapi` at all.
  - Zero top-level name collisions between the 5 moving test files' decls and
    `serveapi_external_test.go`'s decls (`fixtureHost`, `validConfig`, `writeFixtureFile`,
    `externalFixtureSource`, `deniedFixtureSource`, `host`) — compared by `grep -n '^func \|^type \|^var \|^const '` on both sides.
- **Plan disposition (planner ruling, recorded)**: the moved machinery is **unexported** in
  `serveapi`: `callbackRunner`, `newCallbackRunner`, `runCallback`, `embeddedMCPConfig`,
  `newEmbeddedMCPHandler`, `embeddedA2AConfig`, `newEmbeddedA2AHandler`. **Net new public symbols
  in `serveapi`: zero.** This does not touch any Design-Freeze box (the freeze covers the split
  line, the contract-vs-machinery line, back-compat-by-alias, and D-A/module boundary — not export
  status), it preserves the doc's AC6 oracle exactly (`apiserver.RunCallback` → `runCallback` is
  still *two call-site renames + one import deletion* in `serveapi_external_test.go`), and it keeps
  the doc's stated follow-up cheap: if World later wants the ready-made A2A handler, exporting is
  additive. Task M2.2.
- **Resolves a Deferred Decision as measured fact, not choice**: the doc defers "whether
  `RunCallback` remains a free generic function or gains a method form". It has no choice —
  `RunCallback[T any]` is generic (`internal/apiserver/callbacks.go:39`) and **Go has no generic
  methods**. It stays a free function. **VERIFIED BY ME** (read the signature).

---

**D4 — `make lint` does not cover `serveapi`, so the new public package would ship unlinted.**

`make/code-health.mk:71` runs `golangci-lint run ./cmd/... ./internal/... ./testutil/...`.
`./serveapi/...` is not in that list, and `serveapi/protocol` would not be either.

- **VERIFIED BY ME**: `grep -n -A4 '^lint:' make/code-health.mk` shows the three-pattern
  invocation with no `serveapi`; and `golangci-lint run ./serveapi/... ./internal/apiserver/...`
  run directly (same repo `.golangci.yml`) → **rc=0, "0 issues"**, so adding the pattern is safe
  at base and cannot import a pre-existing red.
- **Plan disposition**: M4 adds `./serveapi/...` to the `make lint` pattern list. Baselined: `make
  lint` is rc=0 today and the added scope is rc=0 today, so the change cannot flip the gate on its
  own. Task M4.3.

---

**D5 — smaller inaccuracies, none blocking.**

- The doc says `authorized_surface_test.go`'s "stay-behind-gateway assertions
  (`loadedExportMember`, `isExposed`) remain in `apiserver`". That file contains **neither**: it
  holds only `ToolDescriptor` (12 refs) and `callerSurface` (3 refs), so it moves to
  `serveapi/protocol` **whole**. The `isExposed` assertions live in `mcp_schema_test.go` (13 refs),
  which stays untouched. **VERIFIED BY ME** (`grep -o … | sort | uniq -c` per file).
  This makes M1 *simpler* than the doc implies.
- The doc's `protocol_compat.go` is budgeted "~40 LOC". At the **M1 boundary** it must be roughly
  **2.5x** that, because `embedded_*.go` are still in `internal/apiserver` and now reference
  moved-away `ToolDescriptor`/`AuthorizedSurface`/`callerSurface`/`ErrCallbackCapacity`/the
  envelope helpers. ~55 of those LOC are **transitional and must be deleted in M2**. §5 M2.6 makes
  that deletion a checklist item rather than a hope. (Sizing is my estimate, not a measurement.)
- `ARCHITECTURE.md` currently contains **zero** occurrences of `serveapi`
  (**VERIFIED BY ME**: `grep -n 'serveapi' ARCHITECTURE.md` → empty, rc=1; control:
  `grep -c 'serveapi' docs/docs/guides/serve-api.md` → 4, rc=0). The doc's "ARCHITECTURE.md
  one-liner" is therefore the file's *first* mention of the published seam, not an edit to an
  existing entry.
- Issue #764's acceptance sentence is *"the closure passes `TestDaemonDependencyAllowlist`
  **unchanged**"*, while the design doc says World adds one allowlist line. Literal "unchanged" is
  unachievable by **any** new upstream package (its import path is non-stdlib and is not one of
  World's 11 roots), so the doc's reading — *their test code unchanged, one data line added* — is
  the only coherent one. **UNVERIFIED, inherited from the design doc (V10, read at ailang-world
  `48ef27518`) — I did not read World's checkout.** The #764 reply (M4.5) must state the one-line
  edit explicitly so World's loop is not surprised.

---

**D7 — `grep -c 'internal/apiserver' serveapi/*.go` is the WRONG instrument for "the facade no
longer imports the runtime". It would force deleting a load-bearing test.**

`serveapi/serveapi_external_test.go` mentions `internal/apiserver` **three** times, and only one of
them is the import:

- **line 17** — the real import. **Deleted by M2.**
- **line 120** — `want := "use of internal package github.com/sunholo-data/ailang/internal/apiserver not allowed"`, the expected compiler error inside `TestExternalModuleCanImportFacadeButNotInternal`. **Must stay.**
- **line 215** — inside the `deniedFixtureSource` const (a `//go:build denied` fixture whose whole purpose is to prove an external module *cannot* import the internal package). **Must stay.**

A source-text grep asserted to be `0` therefore demands deleting the very test that guards the
internal boundary. **VERIFIED BY ME** (`grep -n 'internal/apiserver' serveapi/serveapi_external_test.go`
→ lines 17, 120, 215; `sed -n '213,216p'` shows the fixture).

**Plan disposition**: the M2 boundary gate uses the **toolchain's own import list**, which is
immune to strings-in-fixtures: `go list -f '{{join .Imports "\n"}}' ./serveapi | grep -c 'internal/apiserver'`
must be **0** (rc=1), with the same command against `./cmd/ailang` as the same-call control
(**2**, rc=0). Base measured at B18: **1** for `./serveapi`. This also confirms the AC6 oracle
arithmetic — `serveapi_external_test.go`'s diff really is ≤ 3 lines, because lines 120 and 215 are
untouched.

---

## 2. Scope — frozen, and what is NOT in it

**D-A / ledger `D-35` is RESOLVED**: `serveapi/protocol` is a **plain package owned by the main
module**, shipped with **no promise of transparent later conversion** to a nested module.
**UNVERIFIED, inherited from the design doc and the iteration-261 briefing** (directive `D-35 A`
on `#745`, 2026-08-23T19:01:24Z, author `MarkEdmondson1234`) — I did not read the directive
first-hand.

Consequently **out of scope, and an executor producing any of it has failed the sprint**:

- no `serveapi/protocol/go.mod`
- no `serveapi/protocol/vX.Y.Z` tag scheme
- no `require`/`replace` directive changes in the main `go.mod`
- no release-workflow or release-manager change
- no CI matrix entry for a nested module
- no external-consumer resolution verification row (that is owed only by a future option-(b)
  project, which does not exist)

Also out of scope, from the doc's Non-Goals: reimplementing MCP transport in stdlib-only code;
shipping **anything executable** in `protocol`; shrinking `internal/apiserver`'s own closure;
editing ailang-world.

**Published-API-seam rule applied throughout (this is the tie-breaker the plan uses):** adding a
symbol later is cheap, removing one is not. Where the design doc and issue #764 could be read
differently, the issue's wording governs and the smaller surface wins. This produced D3
(machinery unexported in `serveapi`). It did **not** override the doc's one flagged maximal call —
the wire-error taxonomy (`ErrCallbackCapacity`, `CallbackMessage`, `AuthorizationStatus`) stays in
`protocol` — because the alternative is structurally worse, not merely smaller: `CallbackMessage`
switches on `ErrCallbackCapacity`, so the three cannot be separated, and moving all three to
`serveapi` would leave a downstream handler-writer able to *write* an envelope but unable to emit
the **frozen** #592/#603 wire strings — which is precisely the "envelope framing" the issue asked
for. Recorded as a planner ruling, still strikeable at zero design cost if a reviewer disagrees.

---

## 3. BASELINE TABLE — every acceptance command, run on the pristine tree FIRST

Nothing becomes an acceptance criterion in §5 unless it appears here with a base result. A gate
that is already red on untouched `dev` measures the repo, not the sprint.

**All rows: green/observed on darwin/arm64, go1.26.6, HEAD `9ce91ce50`, clean tree.
Windows and ubuntu CI legs unrun locally.**

| # | Command | Base result | Verdict |
|---|---|---|---|
| B1 | `go build ./serveapi/... ./internal/apiserver/...` | **rc=0**, no output | usable gate ✅ |
| B2 | `go build ./...` | **rc=1** — sole failure `# github.com/sunholo-data/ailang/cmd/wasm: runtime.main_main·f: function main is undeclared in the main package` | **RED AT BASE — MUST NOT be an acceptance criterion.** Use B1. |
| B3 | `go test -count=1 ./serveapi/... ./internal/apiserver/...` | **rc=0** — `ok serveapi 3.2s`, `ok internal/apiserver 1.5s`, `ok internal/apiserver/schema 0.8s`, `? internal/apiserver/templates [no test files]` | usable gate ✅ |
| B4 | `go list ./serveapi/protocol` | **rc=1**, `directory not found`. Control **in the same call**: `go list ./serveapi` → rc=0, prints the import path. Existence assertion: `test -d serveapi` → EXISTS, `test -d serveapi/protocol` → ABSENT | correctly red at base ✅ |
| B5 | `go list -deps ./serveapi \| awk -F/ '$1 ~ /\./ {n++} END{print n+0}'` | **479** non-stdlib packages **in the `./serveapi` closure** | ✅ |
| B6 | the V22 cloud-pattern `grep -Ec` over B5's non-stdlib lines | **325** packages **in the `./serveapi` closure**, grep rc=0 | ✅ |
| B7 | `go list -deps -f '{{if not .Standard}}{{with .Module}}{{.Path}}{{end}}{{end}}' ./serveapi \| sort -u \| wc -l` | **67** module roots **in the `./serveapi` closure** (66 external + the ailang module) | ✅ |
| B8 | `make check-protocol-closure` | **`make: *** No rule to make target 'check-protocol-closure'. Stop.`** Control **in the same file**: `grep -n '^check-boundaries:' make/code-health.mk` → line **139**, rc=0; `make -n check-boundaries` → `bash scripts/check_boundaries.sh` | correctly red at base ✅ |
| B9 | `grep -c check-protocol-closure .github/workflows/ci.yml` | **0**, grep **rc=1** (= "no match", not "no such file"). Existence assertion: `test -f .github/workflows/ci.yml` → EXISTS. Control **in the same file**: `grep -c check-boundaries .github/workflows/ci.yml` → **1**, rc=0 | correctly red at base ✅ |
| B10 | `make test` | **rc=0** — **113** `ok` package lines, **0** `--- FAIL`, 0 `FAIL` lines. Repo-wide; slowest gate in the plan | usable gate ✅ (M4 only) |
| B11 | `make lint` | **rc=0** — "✓ Lint complete (no bugs detected)"; reports 2 non-fatal `unused` findings, neither in `serveapi`/`apiserver` | usable gate ✅ |
| B12 | `golangci-lint run ./serveapi/... ./internal/apiserver/...` | **rc=0**, "0 issues" | usable gate ✅ (and proves D4's fix is safe) |
| B13 | `make check-boundaries` | **rc=0** — "OK: no architecture boundary violations." | usable gate ✅ |
| B14 | `make check-file-sizes` | **rc=0** — "✓ All files within 800 line limit" | usable gate ✅ |
| B15 | V8 probe rebuilt from the design doc's appendix recipe | `BUILD_OK` (no shim file), non-stdlib = **1** (itself), total enumeration **188**, module roots = `github.com/sunholo-data/ailang` only; cleanup `REPO_STILL_BUILDS`, `git status --short` empty | the sprint's premise holds ✅ |
| B16 | `go list -deps github.com/modelcontextprotocol/go-sdk/mcp` | **29** external non-stdlib packages, **9** external module roots | feeds the §4.3 allowlist ✅ |
| B17 | intruder-visibility probe (see D2) | `_test.go` intruder → count **stays 1**, uuid **absent**; `-test` control → count **4**, uuid present; non-test intruder → count **1→2**, uuid named | the self-test's mechanism is now measured, not assumed ✅ |
| B18 | `go list -f '{{join .Imports "\n"}}' ./serveapi \| grep -c 'internal/apiserver'` | **1**, rc=0. With `TestImports` appended: **2**. Control **in the same call**: same command on `./cmd/ailang` → **2**, rc=0 | ✅ — see D7 for why the naive `grep` over the source files is the WRONG instrument |
| B19 | `go doc ./serveapi \| grep -c 'Runner\|Embedded'` | **0**, grep rc=1. Control **in the same call**: `go doc ./serveapi \| grep -c 'func New'` → **1**, rc=0 | ✅ non-regression guard for AC8 |

**Post-change prediction, derived from verified measurements (arithmetic, not a guess):**
`./serveapi` non-stdlib closure = 29 (SDK subtree, B16) + `serveapi` itself + `serveapi/protocol`
= **31**. The doc's "~31" is exactly this sum. AC2's ceiling is set at **40** to absorb
cross-platform variance in the SDK's per-arch packages.

---

## 4. Technical approach

### 4.1 Package topology (unchanged from the frozen doc)

```
serveapi ──────────────► serveapi/protocol      (stdlib-only contract)
serveapi ──────────────► modelcontextprotocol/go-sdk/mcp
internal/apiserver ────► serveapi/protocol      (aliases + forwards; wire-type single definition)
cmd/ailang/serve_api.go ► internal/apiserver    (untouched)
```

`protocol` imports **only** stdlib. It must never import `serveapi` or `internal/apiserver`;
§4.3's gate makes that structural rather than a convention.

### 4.2 What lands where

`serveapi/protocol` (package `protocol`), four files, all **moves** with export renames:

| file | contents | source |
|---|---|---|
| `descriptor.go` | `ToolDescriptor`, `AuthorizedSurface` (+ `Lookup`, `All`), `CallerSurface` (was `callerSurface`), `validateToolDescriptor`, `validateHeaderAnnotations`, `validHTTPFieldName`, `cloneToolDescriptor`, `ValidateMCPName` + `mcpToolNameRegex` | `internal/apiserver/authorized_surface.go` **minus lines 28–41** (`loadedExportMember`, `isExposed` — those stay) + `mcp_schema.go` lines 11–14, 103–108 |
| `interfaces.go` | `Session`, `SessionResolver`, `ToolSource`, `Invoker`, `Invocation`, `InvocationResult`, `AgentInfo`, `AuthorizationError` | `serveapi/serveapi.go` lines 23–35, 46–75 (aliases left behind) |
| `a2a_wire.go` | `A2ARequest`, `A2ATaskSendParams`, `A2AMessage`, `A2AContent`, `A2AError`, `A2AResult` | `internal/apiserver/a2a.go` lines 274–322 (exported; full server re-points via aliases) |
| `envelope.go` | `WriteMCPEnvelope`, `RequestID`, `CallbackMessage`, `AuthorizationStatus`, `ErrCallbackCapacity` | `internal/apiserver/embedded_mcp.go` lines 177–236 + `callbacks.go` lines 10–12 |

Nothing in `protocol` runs: no `http.Handler`, no goroutine, no channel.

`serveapi` gains (all **unexported**, per D3):
`callbacks.go` (runner, returns `protocol.ErrCallbackCapacity`), `a2a_handler.go` (+ a private
7-line `writeJSON` copy; the original stays in `apiserver/handler.go` where 7 other files use it),
`mcp_handler.go` (+ a private 5-line `mcpError` copy — the original relocates into
`apiserver/mcp.go`, its remaining consumer alongside `protocol_test.go`).

One mechanical adjustment, **VERIFIED BY ME** as the only private-field access across the new
boundary: `internal/apiserver/embedded_a2a.go:65` reads `len(surface.tools)`; cross-package this
becomes `len(surface.All())`. `AuthorizedSurface.All()` exists today
(`authorized_surface.go:154`). Every other `surface.*` use in the movers is already
`All()`/`Lookup()` (`grep -n 'surface\.' internal/apiserver/embedded_{a2a,mcp}.go`).

`internal/apiserver` after M2 needs **permanent** compat for only three things — because
`ToolDescriptor`, `AuthorizedSurface`, `callerSurface`, `validateToolDescriptor`,
`cloneToolDescriptor`, `validateHeaderAnnotations`, `validHTTPFieldName` and `ErrCallbackCapacity`
are used by **no full-server file at all** (**VERIFIED BY ME**, per-symbol
`grep -rn <sym> internal/apiserver/*.go | uniq -c`: every hit is in `authorized_surface.go`
itself, the three embedded files, or their tests):

1. `a2a_*` wire-type aliases for `a2a.go`
2. `validateMCPName` forwarding to `protocol.ValidateMCPName` (consumers: `mcp.go`,
   `mcp_schema.go`, `mcp_schema_test.go`)
3. `mcpError` relocated into `mcp.go`

The envelope helpers need **no** alias at all: `writeMCPEnvelope`/`requestID`/`callbackMessage`/
`authorizationStatus` have zero consumers outside the two moving files (V26, reproduced).

### 4.3 The closure gate — `scripts/check_protocol_closure.sh`

bash-3.2-safe (no `declare -A`, no `${v,,}`), following the repo's existing
`scripts/check_changelog.sh` + `scripts/test_check_changelog.sh` pattern
(**VERIFIED BY ME**: both exist, 103 and 193 lines; targets at `make/code-health.mk:142` and
`make/test.mk:58`, the latter also running `/bin/bash -n` on the gate script).

**Arm 1 — `protocol`.** Enumerate `go list -deps ./serveapi/protocol`.
Anti-vacuity floor, all four conditions, each its own refusal branch:
`go list` rc must be 0 · enumeration must be non-empty · it must contain the literal
`github.com/sunholo-data/ailang/serveapi/protocol` (known positive) · it must contain ≥1 stdlib
package. Any failure → **exit 2, "vacuous enumeration", never a checkmark.**
Then: every non-stdlib line must be prefixed `github.com/sunholo-data/ailang/serveapi/protocol`.
Violators printed **by name** → exit 1.

**Arm 2 — `serveapi`.** Same floor against `./serveapi` (known positive:
`github.com/sunholo-data/ailang/serveapi`). Then its **external module roots** must be a subset of
a **10-entry** literal allowlist — the ailang module plus the 9 SDK-subtree roots
**VERIFIED BY ME** at B16:

```
github.com/sunholo-data/ailang
github.com/google/jsonschema-go
github.com/modelcontextprotocol/go-sdk
github.com/segmentio/asm
github.com/segmentio/encoding
github.com/yosida95/uritemplate/v3
golang.org/x/oauth2
golang.org/x/sync
golang.org/x/sys
golang.org/x/time
```

Roots, not package counts, are the tight assertion — B16 shows the SDK carries per-arch packages
(`segmentio/asm/cpu/{arm,arm64,x86}`), so a count is platform-sensitive while a root set is not.
Violators printed **by name** → exit 1. (Allowlist lives inline in the script: the doc's
recommended option, and edits stay reviewable diffs.)

**Deliberate scope, written into the script's header comment:** the gate measures the **build**
closure, i.e. what a consumer *links*. Test-only imports are out of scope by design — and arm (iv)
of the self-test pins that so it reads as a decision, not an oversight (D2).

---

## 5. Milestones

Four milestones. M3+M4 together are the design doc's M3; splitting them keeps M3 purely about the
refusal gate. **Every milestone is independently committable and bisectable**: the "green at
boundary" row is the gate the executor must observe **before committing**, and each is a command
from §3 with a known base result.

---

### M1 — extract `serveapi/protocol` (0.75 day, ~190 new/modified LOC + ~375 moved)

**Completed 2026-08-23 (iteration 262).** TDD structural compilation failed first on the absent
package surface, then passed after extraction. Boundary measurements: 188 total dependency
packages; exactly one non-stdlib package (`github.com/sunholo-data/ailang/serveapi/protocol`);
exactly one module root (`github.com/sunholo-data/ailang`). All seven M1 gates passed. The moved
descriptor tests contain no filesystem paths, external binaries, or golden/native line-ending
comparisons, so the Windows-portability scan found no risk. Transitional shim: 16 marked entries.

`serveapi/` is **not touched** in M1. Only `internal/apiserver` is rewired, behind a shim.

**Tasks**
- **M1.1** Create `serveapi/protocol/{descriptor,interfaces,a2a_wire,envelope}.go` per §4.2 by
  *moving* code (git-visible moves; export renames only, no logic edits).
- **M1.2** Package godoc on `descriptor.go`: must state the stdlib-only guarantee **and name
  `make check-protocol-closure`** as the thing that enforces it.
- **M1.3** Shrink `internal/apiserver/authorized_surface.go` to lines 28–41 only
  (`loadedExportMember`, `isExposed`) or fold them into `server.go`; delete `mcp_schema.go`'s
  regex + `validateMCPName` body.
- **M1.4** `internal/apiserver/protocol_compat.go` — **transitional** shim so the still-resident
  `embedded_*.go` compile: aliases/forwards for `ToolDescriptor`, `AuthorizedSurface`,
  `callerSurface`, `ErrCallbackCapacity`, the 5 a2a wire types, `validateMCPName`, and the 4
  envelope helpers. Annotate every transitional entry `// TRANSITIONAL — deleted in M2` so M2.6
  can be checked mechanically.
- **M1.5** Move `internal/apiserver/authorized_surface_test.go` → `serveapi/protocol/descriptor_test.go`
  **whole** (D5: it has no stay-behind refs), re-pointing `callerSurface` → `CallerSurface`.
  Assertions byte-preserved.

**Green at boundary (all must pass before the M1 commit)**
| gate | base (§3) | required at M1 |
|---|---|---|
| `go build ./serveapi/... ./internal/apiserver/...` | rc=0 (B1) | rc=0 |
| `go test -count=1 ./serveapi/... ./internal/apiserver/...` | rc=0 (B3) | rc=0 |
| `go vet ./serveapi/... ./internal/apiserver/...` | rc=0 | rc=0 |
| `go list -deps ./serveapi/protocol \| awk -F/ '$1 ~ /\./'` | rc=1, dir not found (B4) | prints **exactly one** line: `github.com/sunholo-data/ailang/serveapi/protocol` |
| `go list -deps ./serveapi/protocol \| wc -l` | n/a | **≥ 2** (anti-vacuity; B15 saw 188 for the probe) |
| `go list -deps -f '{{if not .Standard}}{{with .Module}}{{.Path}}{{end}}{{end}}' ./serveapi/protocol \| sort -u` | n/a | exactly `github.com/sunholo-data/ailang` |
| `golangci-lint run ./serveapi/... ./internal/apiserver/...` | rc=0 (B12) | rc=0 |

**Stop-and-re-measure rule (from the doc, kept):** if M1 surfaces a leaked symbol the B15 probe
did not (e.g. a build-tagged file outside darwin/arm64), **stop and re-measure** — do not widen
`protocol` silently. Any symbol added to `protocol` beyond §4.2 is a scope change requiring a
recorded decision.

---

### M2 — rewire `serveapi`; evict machinery from `internal/apiserver` (1.0 day, ~200 new/modified LOC + ~1,170 moved test LOC)

**Tasks**
- **M2.1** Move `callbacks.go` → `serveapi/callbacks.go`; `embedded_a2a.go` →
  `serveapi/a2a_handler.go` (+ private `writeJSON` copy, + `len(surface.All())` at the old
  line 65); `embedded_mcp.go` → `serveapi/mcp_handler.go` (+ private `mcpError` copy), importing
  `protocol` + `go-sdk/mcp`.
- **M2.2** **Unexport** all moved machinery in `serveapi` (D3): `callbackRunner`,
  `newCallbackRunner`, `runCallback`, `embeddedMCPConfig`, `newEmbeddedMCPHandler`,
  `embeddedA2AConfig`, `newEmbeddedA2AHandler`. `runCallback` stays a **free generic function**
  (Go has no generic methods).
- **M2.3** `serveapi/serveapi.go`: moved types become aliases
  (`type ToolDescriptor = protocol.ToolDescriptor`, …); `New()` builds runner + both handlers from
  the local constructors; **both descriptor-conversion loops deleted** (aliasing makes them
  identity); **delete the `internal/apiserver` import**.
- **M2.4** Move the machinery tests: `embedded_mcp_test.go`, `embedded_mcp_replay_test.go`,
  `callbacks_test.go` move **whole**. **`embedded_a2a_test.go` is SPLIT (D1)**:
  `TestLoadedExportMembershipAndNoMCPProjection` (lines 323–344) **stays in
  `internal/apiserver`**; the other 9 test funcs + 4 helpers move. Note the three embedded test
  files share `embeddedTestHost` (defined in `embedded_mcp_test.go`) — **they move in the same
  commit or none does**.
- **M2.5** Mechanical re-points only, in moved tests: package line, imports, and
  `writeMCPEnvelope` → `protocol.WriteMCPEnvelope` at `embedded_mcp_replay_test.go:207`.
  **Every assertion byte-preserved** — the frozen #592/#603 envelope tests are the wire oracle.
- **M2.6** Delete every `// TRANSITIONAL` entry from `protocol_compat.go`. A surviving
  transitional alias is a defect: `grep -c 'TRANSITIONAL' internal/apiserver/protocol_compat.go`
  must be **0** (control, same call: `grep -c 'protocol\.' internal/apiserver/protocol_compat.go`
  must be **> 0**, proving the file still exists and still forwards).
- **M2.7** `serveapi/serveapi_external_test.go`: exactly two call-site renames
  (`apiserver.RunCallback` → `runCallback`, lines 68 and 81) + delete the `internal/apiserver`
  import. **No other edit.**

**Green at boundary**
| gate | base (§3) | required at M2 |
|---|---|---|
| `go build ./serveapi/... ./internal/apiserver/...` | rc=0 (B1) | rc=0 |
| `go test -count=1 ./serveapi/... ./internal/apiserver/...` | rc=0 (B3) | rc=0 |
| `go list -deps ./serveapi \| awk -F/ '$1 ~ /\./ {n++} END{print n+0}'` | **479** (B5) | **≤ 40** (predicted **31**) — record the exact number in the report |
| V22 cloud-pattern `grep -Ec` over that list | **325** (B6) | **0** |
| `go list -deps -f '…{{.Module.Path}}…' ./serveapi \| sort -u` | **67** roots (B7) | exactly the **10** roots of §4.3 |
| `go list -f '{{join .Imports "\n"}}' ./serveapi \| grep -c 'internal/apiserver'` | **1**, rc=0 (B18) | **0**, rc=1; control **in the same call**: same command on `./cmd/ailang` → **2**, rc=0 |
| `git diff --stat HEAD~ -- serveapi/serveapi_external_test.go` | n/a | **≤ 3 changed lines** (AC6 oracle) |
| `golangci-lint run ./serveapi/... ./internal/apiserver/...` | rc=0 (B12) | rc=0 |

---

### M3 — the refusal gate (0.9 day, ~365 LOC)

**This milestone's deliverable is a REFUSAL. It is not done until something goes red when each
refusal branch is removed, and until an ADDED member is detected — not merely a removed one.**

**Refusal branches (8), one neutering mutation each.** A branch with no mutation is not a gate.

| B | branch | exit | neutering mutation | must go red as |
|---|---|---|---|---|
| R1 | arm 1: `go list` rc ≠ 0 | 2 | drop the rc check | vacuity arm (ii) exits 0/1 instead of 2 ⇒ self-test fails |
| R2 | arm 1: enumeration empty | 2 | drop the empty check | vacuity arm (ii) reports a checkmark ⇒ self-test fails |
| R3 | arm 1: self-literal `…/serveapi/protocol` absent | 2 | drop the known-positive check | point the gate at `./cmd/astdump`: it must exit 2; with the check dropped it exits 0 ⇒ self-test fails |
| R4 | arm 1: no stdlib package present | 2 | drop the stdlib-presence check | as R3 |
| R5 | arm 1: non-stdlib line lacking the protocol prefix | 1 | make the prefix test always true | intruder arm (i) exits 0 ⇒ self-test fails |
| R6 | arm 2: `go list` rc ≠ 0 / empty for `./serveapi` | 2 | drop the arm-2 floor | vacuity arm run against arm 2 exits 0 ⇒ self-test fails |
| R7 | arm 2: self-literal `…/serveapi` absent | 2 | drop it | as R3, arm 2 |
| R8 | arm 2: external module root outside the 10-entry allowlist | 1 | make the subset test always true | serveapi-intruder arm (v) exits 0 ⇒ self-test fails |
| R9 | the stdlib classifier itself (dot-rule) | — | neuter it (treat everything as stdlib) | intruder arm (i) no longer names uuid ⇒ self-test fails |

**Tasks**
- **M3.1** `scripts/check_protocol_closure.sh` — both arms, all 9 branches above, violators
  printed by name, bash-3.2-safe. Accepts a package path argument so the self-test can point it
  at a nonexistent path.
- **M3.2** `make/code-health.mk`: `check-protocol-closure` (next to `check-boundaries`, line 139).
- **M3.3** `make/test.mk`: `test-check-protocol-closure` running
  `/bin/bash scripts/test_check_protocol_closure.sh` **and** `/bin/bash -n scripts/check_protocol_closure.sh`
  — the exact shape of `test-check-changelog` (`make/test.mk:58`).
- **M3.4** `scripts/test_check_protocol_closure.sh`, **five arms**:
  - **(i) intruder / ADDITION arm.** Write `serveapi/protocol/zz_intruder.go` — **a non-test
    file; a `_test.go` file is invisible to `go list -deps` and would silently not fire (D2, B17)**
    — containing `package protocol` + `import _ "github.com/google/uuid"`. Require **exit 1**,
    output **naming `github.com/google/uuid`**, and the reported non-stdlib **count to have moved**
    (1 → 2) versus the clean run. A verdict flip alone does not pass. Remove the file; assert the
    tree is byte-identical to before (`git status --porcelain` empty).
  - **(ii) vacuity arm.** Point the gate at a nonexistent package path. Require **exit 2** and its
    vacuous-enumeration message; require **no checkmark** in the output.
  - **(iii) restoration arm.** Re-run the clean gate; require **exit 0**. A self-test that leaves
    debris fails loudly instead of poisoning the next run.
  - **(iv) scope arm (D2's blind spot, pinned deliberately).** Write
    `serveapi/protocol/zz_intruder_test.go` with the same uuid import; require the gate to stay
    **exit 0**, and print the reason: the gate models a *consumer's link closure*, and test-only
    imports are never linked by a consumer. Clean up; assert byte-identical tree.
  - **(v) serveapi-arm addition.** Inject a disallowed-root import into a temporary
    `serveapi/zz_intruder.go`; require **exit 1 naming the root**; clean up; assert byte-identical
    tree.
- **M3.5** `.github/workflows/ci.yml`: two steps immediately after
  `- name: Check architecture boundaries` (line 132–133) — `make check-protocol-closure` and
  `make test-check-protocol-closure` — mirroring the changelog gate + self-test pair at lines
  135–141.
- **M3.6** Run the full mutation table (R1–R9 above **plus** the doc's Mutation 2 "add
  `internal/apiserver` import to a protocol file" and Mutation 3 "re-add `internal/apiserver` to
  `serveapi`") and **record each observed red verbatim** in the implementation report. Revert every
  mutation; `git status --porcelain` empty at the end.

**Green at boundary**
| gate | base (§3) | required at M3 |
|---|---|---|
| `make check-protocol-closure` | "No rule to make target" (B8) | **rc=0** |
| `make test-check-protocol-closure` | target absent | **rc=0**, and arm (i)'s output names uuid and shows the count moving 1→2 |
| `grep -c check-protocol-closure .github/workflows/ci.yml` | **0**, rc=1 (B9) | **≥ 2**, rc=0 |
| `/bin/bash -n scripts/check_protocol_closure.sh` | n/a | rc=0 (bash-3.2 parse; the rig runs 3.2.57) |
| `git status --porcelain` after M3.6 | empty | empty |
| `go test -count=1 ./serveapi/... ./internal/apiserver/...` | rc=0 (B3) | rc=0 |

---

### M4 — docs, lint scope, CHANGELOG, #764 reply (0.35 day, ~120 LOC)

**Tasks**
- **M4.1** `docs/docs/guides/serve-api.md` — embedding section (from line 1348) gains
  `serveapi/protocol`: what it carries, the stdlib-only guarantee, and that
  `make check-protocol-closure` enforces it.
- **M4.2** `.claude/rules/api-server.md` — path references at lines **34**, **48**, **55** now
  point at `serveapi/protocol/descriptor.go` for `CallerSurface`/`AuthorizedSurface` and
  `ValidateMCPName`; `mcpToolName()` generation stays in `internal/apiserver/mcp_schema.go`.
  The one-gateway invariant is restated: `loadedExportMember` stays the loaded-export gateway in
  `apiserver`; `CallerSurface` is the embedded-callback membership path, now from `protocol`.
- **M4.3** `make/code-health.mk:71` — add `./serveapi/...` to the `make lint` pattern list (D4).
  Safe at base: B11 rc=0 and B12 rc=0.
- **M4.4** `ARCHITECTURE.md` — first-ever `serveapi` entry (D5): the two public packages and the
  enforced import direction `serveapi → serveapi/protocol`, never the reverse.
- **M4.5** `changelogs/v0.18-current.md` `[Unreleased]` — the new public package, the closure
  numbers **as measured at the merge SHA** (not the planned ones), the alias-based back-compat,
  and the one observable change: `reflect.Type.PkgPath()` on the moved types now reports
  `…/serveapi/protocol`.
- **M4.6** Reply on **#764** with: package path, measured closure (re-run the B15 shape at the
  merge SHA), gate location, D-A status = **(a) plain package, no promise of transparent later
  conversion** — and, **explicitly**, the trade the design doc requires be stated: World gets the
  contract (descriptor types + validation, `CallerSurface`/`AuthorizedSurface`, host interfaces,
  A2A wire types + writers, MCP envelope writer + `RequestID`, wire-error taxonomy) and must still
  **write its own HTTP handlers and callback-bounding machinery**, or import `serveapi` and accept
  the 9-root SDK subtree its gate rejects today. Also state that World's edit is **one allowlist
  line, their test code unchanged** (D5 — #764's literal "unchanged" is unachievable for any new
  upstream package).
  Use `refs #764`, **not** a closing keyword, until merge.

**Green at boundary — the only deliberately repo-wide gates in this plan**
| gate | base (§3) | required at M4 |
|---|---|---|
| `make test` | **rc=0**, 113 ok, 0 FAIL (B10) | rc=0, 0 FAIL |
| `make lint` | rc=0 (B11) | rc=0, with `./serveapi/...` now in scope |
| `make check-boundaries` | rc=0 (B13) | rc=0 |
| `make check-file-sizes` | rc=0 (B14) | rc=0 |
| `make check-protocol-closure` + `make test-check-protocol-closure` | (M3) | rc=0 |

**`go build ./...` is NOT a gate at any milestone (B2: rc=1 at base, `cmd/wasm`).**

---

## 6. Success criteria (AC1–AC8), each with its base result

- **AC1 — protocol closure.** `go list -deps ./serveapi/protocol | awk -F/ '$1 ~ /\./'` prints
  **exactly one** line, `github.com/sunholo-data/ailang/serveapi/protocol`, and
  `go list -deps ./serveapi/protocol | wc -l` ≥ 2. **Base: rc=1 "directory not found" (B4) — red.**
- **AC2 — facade closure.** `./serveapi` non-stdlib count ≤ **40** (expect **31**) AND the V22
  cloud-pattern grep over the same list = **0** AND the external module-root set = exactly the 10
  of §4.3. **Base: 479 / 325 / 67 roots (B5, B6, B7) — red.** Record the exact numbers.
- **AC3 — gate exists and is green.** `make check-protocol-closure` rc=0.
  **Base: "No rule to make target", control `check-boundaries` at `make/code-health.mk:139` (B8) — red.**
- **AC4 — gate can refuse, and can LOOK.** `make test-check-protocol-closure` rc=0; arm (i)'s
  output names `github.com/google/uuid` and shows the non-stdlib count moving 1→2; arm (ii) shows
  exit 2 with no checkmark; arm (v) names a disallowed module root. **Base: target absent — red.**
- **AC5 — CI wiring.** `grep -c check-protocol-closure .github/workflows/ci.yml` ≥ 2.
  **Base: 0 (rc=1), in-file control `check-boundaries` = 1 (rc=0) (B9) — red.**
- **AC6 — behavior preserved.** `go test -count=1 ./serveapi/... ./internal/apiserver/...` rc=0,
  with moved tests byte-preserved modulo package/import/qualified-name lines, and
  `git diff --stat` on `serveapi/serveapi_external_test.go` ≤ 3 changed lines.
  **Base: rc=0 (B3) — green at base, so the informative content is "still green after the move
  with a ≤3-line oracle diff".**
- **AC7 — no collateral damage.** `make test`, `make lint`, `make check-boundaries`,
  `make check-file-sizes` all rc=0. **Base: all rc=0 (B10–B14) — green at base, deliberately.**
- **AC8 — public surface did not grow.** `go doc ./serveapi | grep -c 'Runner\|Embedded'` = 0
  (control, same call: `go doc ./serveapi | grep -c 'func New'` ≥ 1). D3: net new exported symbols
  in `serveapi` = **0**. **Base: 0 (grep rc=1) and control 1 (grep rc=0) — VERIFIED BY ME,
  green at base by construction; this criterion guards against the sprint *adding* surface.**

---

## 7. Velocity & sizing

**VERIFIED BY ME.** Last 7 days: **147** commits, `git diff --stat` = 261 files, +51,829/−4,516 —
but that window is dominated by mission-record docs, so it is not a code-velocity signal. The
usable signal is the recent sprint JSONs in `.ailang/state/sprints/`: M-SYNTAX-AI-FORGIVING 315
LOC / 3d, M-TELEMETRY-REMOTE-READ-FASTFOLLOW 575 / 1.5d, M-TAKE-FLATMAP-PEAK-MEMORY 470 / 4d,
M-STRIP-DECL-AWARE 415 / 1.75d, M-STD-YAML 410 / 1d → roughly **105–380 new LOC/day**.

This sprint: ~**875 new/modified LOC** over 3.0 days ≈ **292 LOC/day** — inside range, and
conservative because roughly **1,540 additional LOC are verbatim moves** (mechanical, not
authored). Estimates in §5 are **my estimates, not measurements**.

| milestone | new/modified | moved | days |
|---|---:|---:|---:|
| M1 extract `protocol` | 190 | 375 | 0.75 |
| M2 rewire `serveapi` | 200 | 1,170 | 1.00 |
| M3 refusal gate | 365 | 0 | 0.90 |
| M4 docs + lint scope + reply | 120 | 0 | 0.35 |
| **total** | **875** | **1,545** | **3.00** |

---

## 8. Risks

| risk | impact | mitigation |
|---|---|---|
| A build-tagged file outside darwin/arm64 leaks a symbol the B15 probe never compiled | med | The stop-and-re-measure rule at M1; the CI gate runs on linux-amd64 and windows; M1 must not widen `protocol` silently |
| CI legs unrun locally (windows-latest, ubuntu-latest) | med | AC2's count ceiling is 40 against a predicted 31; the tight arm-2 assertion is on **module roots**, which B16 suggests are arch-stable while package counts are not |
| The three embedded test files share `embeddedTestHost`; a partial move breaks `internal/apiserver` | med | M2.4 requires them in one commit; the M2 boundary gate is `go test` on **both** package trees |
| D1's split of `embedded_a2a_test.go` drops the membership assertion instead of relocating it | med | M2.4 names the exact func and lines; AC6's byte-preservation instruction covers it; `grep -rn 'TestLoadedExportMembershipAndNoMCPProjection' internal/apiserver/` must return exactly 1 hit after M2 |
| D2 recurs: someone "restores" the reviewer's verbatim `_test.go` filename | high | The filename decision is written into M3.4 with the measurement (B17) inline, and arm (iv) makes the `_test.go` form's green result an *asserted* behaviour with a printed reason |
| Alias identity change breaks a reflection-based consumer | low | No known consumer (V17 reproduced; World is pre-consumption); called out in the CHANGELOG at M4.5 |
| World's gate drifts before delivery (allowlist read at `48ef27518`, **not re-read by me**) | low | The design depends only on "stdlib-only passes any prefix-admitting allowlist"; the #764 reply carries re-derived numbers |
| Someone plans nested-module work because the doc still *contains* the option-(b) spec | med | §2 lists the six artifacts that are out of scope; option (b) text in the doc is the bar for a *hypothetical future project*, not this sprint |

---

## 9. Deferred decisions left to the implementer

- Exact file names/splits inside `serveapi/protocol` — free, as long as the package set and the
  AC1/AC2 closures hold.
- Whether the arm-2 allowlist is a script literal or a checked-in data file (script literal
  recommended: reviewable diffs).
- Whether `authorized_surface.go`'s two stay-behind funcs get their own file or fold into
  `server.go`.
- Godoc wording — but it **must** state the stdlib-only guarantee and name the enforcing gate.

**Resolved by measurement, not deferred:** `runCallback` stays a free generic function (Go has no
generic methods — `internal/apiserver/callbacks.go:39`).

---

**SPRINT_PLAN_PATH**: `design_docs/planned/m-serveapi-protocol-only-module-sprint-plan.md`
**SPRINT_JSON_PATH**: `.ailang/state/sprints/sprint_M-SERVEAPI-PROTOCOL-ONLY.json`
