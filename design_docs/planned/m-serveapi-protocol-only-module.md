# M-SERVEAPI-PROTOCOL-ONLY: a protocol package with a stdlib-only closure (#764)

**Status**: Planned
**Target**: v0.34.0
**Priority**: P1 (ratified to queue head 2026-08-23, charter decision D-31; sole remaining blocker for ailang-world M4)
**Estimated**: 3 days
**Dependencies**: None (#498 Lane A + Lane B shipped in v0.33.0/v0.33.1 and are prerequisites already discharged)
**Issue**: [#764](https://github.com/sunholo-data/ailang/issues/764) (`cross-mission`, filed by the ailang-world mission loop)

---

## Measurement provenance

Every number in this document was re-derived by the doc author in this worktree —
**darwin/arm64, go1.26.6, clean worktree at `a201237ca` (= origin/dev), 2026-08-23** — with the
command shown in the Verification Log. Single-platform caveat: all measurements are
darwin/arm64-only; `go list -deps` closures can differ per GOOS/GOARCH (build-tagged files), so
the CI gate below runs on linux-amd64 in CI *and* darwin/arm64 locally, and the acceptance
criteria treat the CI run as authoritative. The downstream gate quoted here was read from the
ailang-world checkout on this machine at commit `48ef27518` (2026-08-23) — it may drift; nothing
in this design depends on its exact contents beyond what is called out in "Downstream acceptance".

This doc makes **no claims about AILANG language semantics** (it is Go packaging/tooling), so the
`ailang check` hard gate does not apply to any sentence here.

The extraction premise was measured with **two scratch probe packages**, both at
`internal/protoprobe/` and both deleted before delivery (`rm -rf internal/protoprobe` followed by
`go build ./serveapi ./internal/apiserver` → `REPO_STILL_BUILDS`):

- **Authoring-pass probe** (whole-file copies of the four candidate files + an 11-symbol shim):
  produced the leak analysis V5–V7 and the SDK-subtree measurement V9. Superseded as the closure
  evidence but still cited for what it measured.
- **Revision-pass probe** (sed extractions of exactly the re-scoped contents, **zero shim
  symbols**): produced the load-bearing closure result V8. Its full reconstruction recipe — every
  command, byte-exact — is in [Appendix: V8 probe reconstruction](#appendix-v8-probe-reconstruction-recipe),
  so the measurement is re-runnable from this document alone.

---

## Problem Statement

`serveapi` (shipped v0.33.0/v0.33.1, #498 Lane B) is exactly the public *API* seam ailang-world
asked for — but it is not a *dependency* seam. `serveapi/serveapi.go` is 201 lines whose only
non-stdlib import is `github.com/sunholo-data/ailang/internal/apiserver`, so importing the facade
links the entire server runtime:

| package | non-stdlib packages in transitive closure |
|---|---:|
| `./serveapi` | **479** |
| `./internal/apiserver` | **478** |
| *(control)* `./cmd/wasm` | 12 |
| *(control)* `./cmd/astdump` | 14 |

325 of the 479 match a cloud/telemetry/GCP/model-host pattern (V22). The issue's third control
(`cmd/registry-validator` → 6) does **not** reproduce — it measures **453** here and already did
at v0.33.1 (V1b) — but the headline numbers stand on the two controls that do fire.

Downstream, ailang-world enforces its local-first charter as a committed test,
`TestDaemonDependencyAllowlist` (`host/daemon/daemon_test.go`), which requires the full transitive
build graph of its daemon core to be **stdlib + its own repo + 11 enumerated module roots**.
Importing today's `serveapi` would inject hundreds of disallowed packages. World items 2/3/4 are
landed; item 5 (`w-mcp-projection`) is blocked on this issue and item 6 is parked behind it.

**The ask (issue #764, verbatim):** "a protocol-only module (or a build-tag-free subpackage)
carrying the MCP/A2A wire types, envelope framing and the caller-supplied-surface interfaces —
without linking internal/apiserver's runtime." Three named categories. The issue does **not** ask
for a callback runner or an executable HTTP handler, and this design's scope is held to the
issue's wording (see the iteration-260 quorum record at the end of this doc: the authoring pass
had both in `protocol`; the quorum blocked it as a minimal-frozen-core violation, and the re-scope
below is the disposition).

**Measured feasibility (this is the load-bearing result):** the seven `apiserver` symbols
`serveapi` uses live in four files whose direct imports are stdlib-only except for
`github.com/modelcontextprotocol/go-sdk/mcp` in `embedded_mcp.go`. Compiling copies of those four
files as a standalone package leaks exactly **9 undefined symbols** from the rest of
`internal/apiserver` (V5), every one of which is defined in stdlib-only code (V6/V7). Splitting
along the issue's boundary:

- the **re-scoped protocol subset** (descriptors + validation, caller-supplied-surface
  interfaces, A2A JSON-RPC wire types + writers, MCP envelope framing + wire-error taxonomy)
  compiles as a standalone package with **zero shim symbols** and a transitive closure of **zero
  external non-stdlib packages** — 188 packages enumerated, the only non-stdlib line is the
  package itself, the only module root is the ailang module (V8). The zero-shim result is itself
  a measurement: deleting the stay-behind gateway (`loadedExportMember`/`isExposed`) removed
  every reference to `Server`/`ExportInfo`, so descriptor validation is a property of the
  descriptor types, not of the server — no assertion needed;
- the **operational machinery** (`CallbackRunner`/`RunCallback`, the embedded A2A
  `http.Handler`) is used by nothing outside the embedded path (V25) and routes to `serveapi`,
  the already-published facade — not into the frozen core;
- the embedded **MCP handler** genuinely needs the SDK subtree: **29 external non-stdlib packages
  across 9 module roots** (V9), of which only `golang.org/x/sys` is in World's 11-root allowlist —
  the other 8 would be intruders (V10/V11). It stays in `serveapi` too.

So the extraction is real, small, and splits into **contract vs machinery**: a stdlib-only
`protocol` package carrying exactly what the issue named, and `serveapi` carrying every
executable piece (runner, A2A handler, MCP handler) on top of it.

## Goals

**Primary goal:** an importable `github.com/sunholo-data/ailang/serveapi/protocol` package whose
transitive non-stdlib closure is **empty** (itself only), carrying exactly the issue's three
categories — MCP/A2A wire types + envelope framing, and the caller-supplied-surface interfaces
with their type-owned descriptor validation — with `serveapi` rewired on top of it (runner + both
handlers move there) so the published facade keeps its exact surface.

**Success metrics:**
- `go list -deps ./serveapi/protocol` non-stdlib closure = only `…/serveapi/protocol`-prefixed packages (base: package does not exist).
- `./serveapi` non-stdlib closure drops 479 → ≤ 40 (measured prediction: ~31), with **zero** cloud/telemetry-pattern packages (base: 325).
- A CI-enforced allowlist gate (`make check-protocol-closure` + ci.yml step) with an anti-vacuity floor and its own refusal self-test.
- Existing public surface unbroken: `serveapi/serveapi_external_test.go` passes with only its two `apiserver.RunCallback` references renamed to bare `RunCallback` and the `internal/apiserver` import line deleted (V23).

## Non-Goals

- Reimplementing MCP transport machinery in stdlib-only code. The streamable-HTTP MCP handler
  genuinely uses the SDK (`mcp.NewStreamableHTTPHandler`, `mcp.NewServer`, `mcp.Tool`, … — V20c);
  hand-rolling it is a large, drift-prone project with no downstream demand: World asked for
  types/framing/interfaces, not our handler.
- Shipping **any executable machinery** in `protocol`. Everything in the new public core is a
  type, an interface, a validator over those types, or a wire writer; anything that runs (bounded
  concurrency, `http.Handler`s) lives in `serveapi`. This is the asymmetry the quorum named:
  every `protocol` symbol is public API that is cheap to add later and expensive to remove, so
  the package ships only what a consumer asked for and grows on demand.
- Shrinking `internal/apiserver`'s own 478-package closure (the full server legitimately links the
  runtime).
- Editing ailang-world. Its allowlist literal is its own reviewable event; this design only has to
  make that edit minimal (one line) and safe (stdlib-only closure).
- Deciding the module-boundary question (see Decision row D-A — **RESOLVED 2026-08-23 by Mark: option (a), plain package**; ledger `D-35` RESOLVED).

---

## Verification Log

Every claim about current repo state carries its command and observed output. Platform for all
rows: darwin/arm64, go1.26.6, worktree at `a201237ca`. The non-stdlib filter used throughout is
`awk -F/ '$1 ~ /\./ {n++} END{print n+0}'` (first path segment contains a dot ⇔ not stdlib — the
same classifier rule World's gate unit-tests, V10).

**Negative-result discipline (revision pass):** after the iteration-260 quorum caught V17
bounding its search space without justification, every row whose evidence is a negative or empty
result was swept for the same defect class, not just the flagged instance. The standard applied:
(a) a known-positive control in the same call and same scope, (b) a stated scope the row can
defend — either an unbounded instrument (the `go` toolchain's package graph, repo-wide grep) or a
scope that is correct by construction (Go unexported identifiers cannot be referenced outside
their package), and (c) a `test -d`-style existence assertion where an empty grep over a missing
directory would print a confident zero. The sweep changed V17 (replaced), V18 (scope assertion +
same-scope control added), and V19 (widened repo-wide for the exported symbols); V13/V14/V15
already met the standard; V20's package-bounded scope is defensible by construction because every
symbol in it is unexported (now stated in the row).

| # | Claim | Command | Observed |
|---|---|---|---|
| V1 | Closure counts: serveapi 479, apiserver 478, controls 12/14 | `for p in ./serveapi ./internal/apiserver ./cmd/wasm ./cmd/astdump ./cmd/registry-validator; do go list -deps $p \| <filter>; done` | `479 / 478 / 12 / 14 / 453` |
| V1b | The issue's third control (`registry-validator` → 6) is wrong; not drift | same command, plus `git show v0.33.1:cmd/registry-validator/main.go \| grep cloud.google` | 453 today; `cloud.google.com/go/storage` already imported at v0.33.1. Do not cite the "6". |
| V2 | `serveapi/serveapi.go` is 201 lines; sole non-stdlib import is `internal/apiserver` | `wc -l serveapi/serveapi.go; grep -n "sunholo-data/ailang" serveapi/serveapi.go` | `201`; one hit, line 15 |
| V3 | serveapi references exactly 7 apiserver symbols | `grep -o "apiserver\.[A-Za-z]*" serveapi/serveapi.go \| sort \| uniq -c` | ToolDescriptor(6), NewEmbeddedMCPHandler, EmbeddedMCPConfig, NewEmbeddedA2AHandler(+EmbeddedA2AConfig), NewCallbackRunner, CallbackRunner |
| V4 | The 4 defining files' direct imports are stdlib-only except `go-sdk/mcp` in embedded_mcp.go | `sed -n '/^import (/,/^)/p' internal/apiserver/{authorized_surface,embedded_a2a,callbacks,embedded_mcp}.go` | Only non-stdlib line anywhere: `github.com/modelcontextprotocol/go-sdk/mcp` (embedded_mcp.go) |
| V5 | Compiling copies of the 4 files standalone leaks exactly 9 distinct symbols | probe: `sed 's/^package apiserver$/package protoprobe/' …; go build -gcflags=-e ./internal/protoprobe/ \| grep undefined \| sort -u` | `ExportInfo, Server, validateMCPName, writeJSON, a2aError, a2aRequest, a2aMessage, a2aResult, a2aTaskSendParams` (9 distinct) |
| V6 | Each leaked symbol's definition site | `grep -rln "type <S>\|func <S>" internal/apiserver/*.go` per symbol | ExportInfo+Server → server.go; validateMCPName → mcp_schema.go; writeJSON → handler.go; a2a\* → a2a.go |
| V7 | Second-order leak: `embedded_a2a.go` uses `authorizationStatus` + `callbackMessage`, defined in `embedded_mcp.go`, both stdlib-only | remove embedded_mcp.go from probe → build; read definitions | `undefined: authorizationStatus` / `undefined: callbackMessage`; bodies use only `errors`, `context`, `net/http` |
| V8 | **Re-scoped** protocol subset (descriptors+validation, interfaces, A2A wire, envelope framing — no runner, no handlers) compiles standalone with **zero shims**; closure = **zero external non-stdlib packages**; validation is type-owned (no `Server`/`ExportInfo` reference survives) | revision-pass probe, built entirely by sed extraction — exact recipe in the appendix — then `go build -gcflags=-e ./internal/protoprobe/ && go list -deps ./internal/protoprobe \| awk -F/ '$1 ~ /\./'` + count + module roots | `BUILD_OK` (no shim file present); non-stdlib lines: exactly `github.com/sunholo-data/ailang/internal/protoprobe` (count 1); total packages enumerated 188 (non-vacuous); module roots: `github.com/sunholo-data/ailang` only |
| V9 | Full subset (with MCP handler) closure = 29 external packages, 9 external module roots | probe with embedded_mcp.go restored: `go list -deps \| <filter>`; `go list -deps -f '{{if not .Standard}}{{with .Module}}{{.Path}}{{end}}{{end}}' \| sort -u` | `nonstdlib_pkgs=30` (incl. itself); roots: `modelcontextprotocol/go-sdk, google/jsonschema-go, segmentio/asm, segmentio/encoding, yosida95/uritemplate/v3, golang.org/x/{oauth2,sync,sys,time}` |
| V10 | World's gate: 11 allowlisted roots, prefix matcher, dot-rule stdlib classifier, anti-vacuity (empty dep list = error), never-skips | read `~/dev/sunholo-data/ailang-world/host/daemon/daemon_test.go` lines 735–920 at commit `48ef27518` | `allowedDepModules` = ailang-world repo + modernc.org/{sqlite,libc,mathutil,memory} + golang.org/x/sys + dustin/go-humanize + google/uuid + mattn/go-isatty + ncruces/go-strftime + remyoudompheng/bigfft; match is `d == m \|\| strings.HasPrefix(d, m+"/")` |
| V11 | MCP-SDK subtree vs World allowlist: only `golang.org/x/sys` is allowed; **8 intruder roots** | set intersection of V9 roots × V10 list | 8 of 9 external roots absent from the allowlist |
| V12 | The issue's secondary concern (go directive floor) is already resolved at HEAD | `grep -n "^go " go.mod` | `go 1.26.6` (issue measured go1.26.6 fixes World's darwin/arm64 miscompile) |
| V13 | No `serveapi/protocol` package exists at base (negative + control) | `go list ./serveapi/protocol` / control `go build ./serveapi` | `directory not found`, RC=1 / control builds OK |
| V14 | No `check-protocol-closure` make target at base (negative + control) | `make check-protocol-closure` / control `grep -n check-boundaries make/code-health.mk` | `No rule to make target` / control hits line 139 |
| V15 | ci.yml has no protocol-closure step at base (negative + control, same file) | `grep -c check-protocol-closure .github/workflows/ci.yml; grep -c check-boundaries .github/workflows/ci.yml` | `0` / `1` |
| V16 | Base test suite is green (so "tests pass" ACs measure the change, not the repo) | `go test -count=1 ./serveapi/... ./internal/apiserver/...` | `ok serveapi 3.442s; ok internal/apiserver 1.811s; ok …/schema 0.614s` |
| V17 | `internal/apiserver` has exactly two importing packages, incl. test imports, module-wide (authoritative instrument = the toolchain's own package graph, which cannot miss an importer; grep corroborates; scope asserted) | primary: `go list -f '{{.ImportPath}} {{join .Imports " "}} {{join .TestImports " "}} {{join .XTestImports " "}}' ./... \| grep 'internal/apiserver' \| awk '{print $1}' \| grep -v 'internal/apiserver' \| sort -u` · corroboration: `grep -rln 'sunholo-data/ailang/internal/apiserver"' --include='*.go' . \| grep -v '^\./internal/apiserver/' \| sort` (whole repo, unbounded) · scope: `for d in pkg test testdata tools examples scripts; do test -d $d …` | packages: `…/cmd/ailang`, `…/serveapi` (2). Files: `cmd/ailang/serve_api.go`, `serveapi/serveapi.go`, `serveapi/serveapi_external_test.go` (3; serveapi.go is the in-call positive). Scope: `pkg`/`test`/`testdata` **absent** in this repo; `tools`/`examples`/`scripts` exist and are inside both instruments' scope. Reproduces the controller's independent run at the same SHA. |
| V18 | No planned/implemented design doc covers #764 other than this one (negative + scope assertion + **same-scope** control) | `test -d design_docs/planned && test -d design_docs/implemented`; `grep -rln "protocol-only\|#764" design_docs/planned/ design_docs/implemented/ --include='*.md' \| grep -v m-serveapi-protocol-only-module.md`; control in the same scope: `grep -rln "serveapi" design_docs/implemented/ --include='*.md'` | both dirs exist; negative → empty; control fires (`v0_20_0/m-serveapi-surface-drops.md`, …). (Original control greped `design_docs/` broadly — a different scope than the negative; replaced.) |
| V19 | The exported moved symbols (`ToolDescriptor`, `AuthorizedSurface`) are referenced outside `internal/apiserver` **only** by `serveapi/serveapi.go` (repo-wide, since exported symbols can travel); inside the package, only the 4 test files of the moved code + `authorized_surface_test.go` for unexported `callerSurface` | repo-wide: `grep -rn 'apiserver\.\(ToolDescriptor\|AuthorizedSurface\)' --include='*.go' . \| awk -F: '{print $1}' \| sort \| uniq -c` · in-package per-symbol `grep -l` sweep excluding the four files (ToolDescriptor firing = control) | repo-wide: `6 serveapi/serveapi.go` only (consistent with V3). In-package: ToolDescriptor → embedded_a2a_test.go, embedded_mcp_test.go, authorized_surface_test.go, embedded_mcp_replay_test.go; AuthorizedSurface → (none); callerSurface → authorized_surface_test.go. (Original sweep was package-bounded even for exported symbols; widened.) |
| V20 | Stay-behind symbol usage: `writeJSON` in 7 full-server files; `validateMCPName` in mcp.go+mcp_schema.go; a2a wire types defined & used in full-server a2a.go; `loadedExportMember` in 6 files; `mcpError` used by mcp.go + protocol_test.go | same grep sweep over `internal/apiserver/*.go` — package scope is **correct by construction** here: every symbol in this row is unexported, and Go forbids referencing unexported identifiers outside their package | as listed (drives the alias/forward plan below) |
| V20c | The MCP handler genuinely needs the SDK (not just wire structs) | `grep -o "mcp\.[A-Za-z]*" internal/apiserver/embedded_mcp.go \| sort \| uniq -c` | `NewStreamableHTTPHandler, NewServer, StreamableHTTPOptions, Tool, CallToolRequest/Result, Content, TextContent, Implementation, DefaultMaxRequestBodyBytes` |
| V21 | serveapi closure spans 67 module roots (66 external) — measured with the exact instrument | `go list -deps -f '{{if not .Standard}}{{with .Module}}{{.Path}}{{end}}{{end}}' ./serveapi \| sort -u \| wc -l` | `67` (the briefing's "~103" was a coarser approximation; 66/67 is the `.Module.Path` ground truth) |
| V22 | 325 of the 479 match the cloud/telemetry pattern | `go list -deps ./serveapi \| <nonstdlib> \| grep -Ec "^(cloud\.google\.com\|google\.golang\.org\|go\.opentelemetry\.io\|go\.opencensus\.io\|github\.com/googleapis\|github\.com/google/s2a-go\|github\.com/GoogleCloudPlatform\|github\.com/ollama\|github\.com/openai\|github\.com/anthropics\|github\.com/aws\|firebase)"` | `325` |
| V23 | The public-surface oracle test is `package serveapi` and touches apiserver **only** via 2 `RunCallback` calls | `head -20 serveapi/serveapi_external_test.go; grep -c "apiserver\." …` | `package serveapi`; `2` references (lines 68, 81) |
| V24 | No open PR touches these files | `gh pr list --state open` + `gh pr view 695 --json files` | open set = 3 dependabot + #695 (`cmd/ailang/chains*` only) |
| V25 | `CallbackRunner`/`RunCallback` (and the embedded A2A handler) are used by **nothing outside the embedded path** — no full-server file touches them, so routing them to `serveapi` needs no `apiserver`→`serveapi` import (positive: definitions + embedded uses fire in the same call) | `grep -rln 'RunCallback\|CallbackRunner' --include='*.go' . \| grep -v '_test.go' \| sort` (repo-wide, unbounded) | `internal/apiserver/callbacks.go` (defs), `internal/apiserver/embedded_a2a.go`, `internal/apiserver/embedded_mcp.go`, `serveapi/serveapi.go` — nothing else |
| V26 | The envelope-framing helpers (`writeMCPEnvelope`, `requestID`, `authorizationStatus`, `callbackMessage`) are used by **no non-test file outside the two embedded files**, so exporting them from `protocol` re-points exactly two consumers (positive control: hit counts inside those files in the same call) | `grep -rn 'writeMCPEnvelope\|requestID(\|authorizationStatus(\|callbackMessage(' --include='*.go' . \| grep -v '_test.go' \| awk -F: '{print $1}' \| sort \| uniq -c` (repo-wide) | `2 internal/apiserver/embedded_a2a.go`, `13 internal/apiserver/embedded_mcp.go` — nothing else |

---

## Axiom Compliance

This is build-graph/packaging work; most axioms are untouched. Scored honestly rather than
inflated:

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No language/runtime semantics change |
| A2: Replayability | 0 | No trace impact |
| A3: Effect Legibility | 0 | No effect changes |
| A4: Explicit Authority | +1 | A dependency allowlist makes build-graph authority explicit and reviewable, upstream and downstream |
| A5: Bounded Verification | +1 | The seam becomes mechanically checkable (`go list -deps` gate) instead of prose |
| A6: Safe Concurrency | 0 | CallbackRunner semantics move verbatim |
| A7: Machines First | +1 | Acceptance is a machine-checkable closure, designed for a downstream machine gate |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | 0 | — |
| A10: Composability | +1 | Wire contract becomes linkable without the runtime; the facade composes on top |
| A11: Structured Failure | 0 | Error taxonomy moves verbatim |
| A12: System Boundary | +1 | The embedding boundary (types/framing vs runtime) becomes a package boundary |

**Net Score: +5** → **Proceed.** Hard-violation check: A1/A3/A4/A7 all ≥ 0. ✅

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D-A: module boundary — plain package vs nested Go module vs separate repo | Determines downstream MVS/go.sum shape, tag scheme, release process, **and whether a main-module-only consumer keeps resolving the package at all** | **human (Mark)** — **RESOLVED 2026-08-23: (a) PLAIN PACKAGE** (directive `D-35 A` on `#745`); ledger `D-35` RESOLVED; see row below | resolved **before implementation began**, exactly as the round-2 quorum demanded | med |
| Split line: MCP handler excluded from `protocol`, kept in `serveapi` | 8 of the SDK's 9 module roots fail World's gate (V11); including it would make the package fail its own reason for existing | agent (this doc) — measured, settled | design | high to reverse |
| `CallbackRunner` + embedded A2A `http.Handler` routed to **`serveapi`**, not `protocol` (reverses the authoring pass) | They are operational machinery outside the issue's three named categories; everything in `protocol` is public API on a shipped seam — cheap to add later, expensive to remove | iteration-260 quorum (gpt5-6-sol objection) + issue #764's own wording; routing target chosen by agent, see below | design | low (both are stdlib-only, V8/V25 — they can move later without closure consequences) |
| Wire-error taxonomy (`ErrCallbackCapacity`, `CallbackMessage`, `AuthorizationStatus`) stays in `protocol`'s envelope framing | These map errors to the **frozen wire strings** the envelope tests assert; a downstream handler-writer (World) needs them to speak the same dialect. Deliberately the maximal point of the re-scope — a reviewer striking them to `serveapi` costs the design nothing | agent (this doc) — flagged for review | design | low |
| Back-compat mechanism: type aliases in `serveapi`, no deprecations | `serveapi` is published API (v0.33.0/v0.33.1); aliases keep source compatibility with zero consumer edits | agent (this doc) | design | low |
| Wire-type single-definition: full server aliases `protocol`'s a2a types rather than keeping copies | Duplicate wire structs drift (the #603 envelope-labelling incident is the cautionary case) | agent (this doc) | design | low |

### Decision row D-A — **RESOLVED 2026-08-23 (Mark): option (a), plain package**

**Question:** what module boundary should `serveapi/protocol` have?

- **(a) Plain package in the main module** *(this doc's default; the sprint proceeds under it)*.
  Cost: a consumer's `go.mod` requires `github.com/sunholo-data/ailang`, inheriting its `go 1.26.6`
  directive as a toolchain floor and its module graph as pruned go.sum entries (no cloud code is
  downloaded or built — module graph pruning, go ≥ 1.17, and the package's closure is stdlib-only).
  World's gate matcher is a **prefix** match (V10), so World can allowlist the narrow string
  `github.com/sunholo-data/ailang/serveapi/protocol` — one line, their review, their test code
  unchanged.
- **(b) Nested module** (`serveapi/protocol/go.mod`, zero `require` lines). Cost: a
  `serveapi/protocol/vX.Y.Z` tag scheme, release-manager changes, and require/replace churn in the
  main module. Benefit: consumers see an empty dependency graph — the cleanest possible seam.
  Convertible from (a) later **without changing any import path**.
- **(c) Separate repository.** Maximum isolation, maximum ops cost, drift risk against the server
  runtime. Not recommended.

**RESOLUTION — 2026-08-23, Mark: (a) PLAIN PACKAGE.** Directive `D-35 A` on bookkeeping issue
`#745` at `2026-08-23T19:01:24Z`, author `MarkEdmondson1234`, read first-party by
`scripts/mission_directives.sh` at iteration 261. `serveapi/protocol` ships as a **plain package
owned by the main module**, with **no promise of transparent later conversion**. The design-freeze
gate is satisfied and implementation may begin.

**DISPOSITION — AMENDED by the iteration-260 round-2 quorum (`gpt5-6-sol`), applied verbatim.
Retained below because it is the record of why this decision had to be surfaced at all; it was
BLOCKING until the ruling above.**

The paragraph that stood here claimed *"(a) → (b) is a later additive change with stable import
paths, so this decision does not gate the sprint"*. That claim was **unverified and is not safe**.
The reviewer's objection, sustained in full: *only the import path is stable.* Moving a directory
into a nested module changes **module ownership and version resolution** — it can require new
`require` directives and coordinated tags, and it can make previously valid main-module versions
**stop supplying the package** to a consumer that only requires
`github.com/sunholo-data/ailang`. The doc asserted a migration property it had never measured, and
then used that assertion to downgrade its own only human decision to non-blocking. Reviewer's fix,
adopted as written:

> *"D-A is a design-freeze gate. This mission will ship either (a) a package owned by the main
> module, with no promise of transparent later conversion, or (b) a nested module with its tag,
> main-module requirement, release workflow, and CI matrix specified now. Implementation does not
> begin until Mark selects one option."*

Consequently:

- **The claim is withdrawn, not repaired.** Nothing in this doc now asserts that (a) converts to
  (b) transparently. Option (a) ships **with no promise of transparent later conversion**; a future
  move to (b) is a breaking-change project with its own doc, not a footnote here.
- **RESOLVED — implementation may begin.** Mark selected **(a)** on 2026-08-23. Ledger **`D-35`**
  is **RESOLVED** and the queue row is **unparked**. Everything else in this doc was already
  quorum-answered, so nothing further gates the sprint plan.
- **(b) was NOT selected**, so none of the following is owed by this sprint. It is retained as the
  specification that any future (a)→(b) project must satisfy before it may proceed — that project
  must first gain: the `serveapi/protocol/vX.Y.Z` tag scheme, the
  main module's `require` directive on the nested module, the release-workflow change, the CI
  matrix entry, and the reviewer's demanded verification row — *a temporary external consumer
  running `go mod tidy`, `go list -deps` and `go test` before and after the conversion, recording
  the required directives, selected module versions, downloaded module roots, and whether
  resolution succeeds without a `replace`.* That row is **not** present today and must not be
  written from reasoning; it is a measurement.
- **Controller note on what does NOT settle this.** Mark's attended 2026-08-23 ruling (charter:
  *World consumes upstream via pinned releases only, so the tag is the delivery; cut v0.34.0 when
  `#764` lands*) is **consistent with (a)** and is the reason (a) is the recommendation — a
  main-module tag is exactly that delivery mechanism. But it was a ruling about *release delivery*,
  not about *module boundary*, and inferring a resolution from adjacent work is precisely what the
  decision-recording contract forbids. So it is recorded as a pointer, not as an answer.

### Design Freeze

- [x] Split line (MCP handler out of `protocol`) — settled by measurement V8/V9/V11
- [x] Contract-vs-machinery line (runner + A2A handler out of `protocol`, into `serveapi`) —
      settled by the iteration-260 quorum + issue wording; feasibility measured V8/V25
- [x] Back-compat via aliases — settled below
- [x] **D-A module boundary — RESOLVED 2026-08-23 (Mark): (a) PLAIN PACKAGE**, shipped with **no
      promise of transparent later conversion**. Ledger `D-35` RESOLVED (directive `D-35 A` on
      `#745`). The round-2 design-freeze gate is satisfied; implementation may begin. The
      unverified (a)→(b) migration claim stays **withdrawn** — a future move to (b) is a
      breaking-change project with its own doc and its own external-consumer verification row.

---

## Solution Design

### Overview

Split along **contract vs machinery**. The protocol contract — descriptors + their validation,
the caller-supplied-surface interfaces, A2A JSON-RPC wire types + writers, MCP envelope framing +
wire-error taxonomy — moves out of `internal/apiserver` into a new public, stdlib-only package
`serveapi/protocol`. Every executable piece — `CallbackRunner`, the embedded A2A handler, the
SDK-dependent embedded MCP handler — moves into `serveapi` itself. The full server
(`internal/apiserver`, imported only by `cmd/ailang` — V17) stays intact behind thin
aliases/forwarders. After this, `serveapi` no longer imports `internal/apiserver` at all.

Import directions (all valid, no cycles):

```
serveapi ──────────────► serveapi/protocol (stdlib-only contract)
serveapi ──────────────► modelcontextprotocol/go-sdk/mcp
internal/apiserver ────► serveapi/protocol   (aliases, wire-type single definition)
cmd/ailang/serve_api.go ► internal/apiserver (unchanged)
```

`protocol` imports nothing but stdlib — in particular it must never import `serveapi` or
`internal/apiserver`; the closure gate makes that structural.

### New package `serveapi/protocol` (package `protocol`) — closure: stdlib only

Contents (moves, with export renames where the symbol crosses a package boundary). Four files,
one per issue category plus the name validator's home:

| new file | contents | provenance |
|---|---|---|
| `descriptor.go` | `ToolDescriptor`, `AuthorizedSurface`, `CallerSurface` (exported `callerSurface`), `validateToolDescriptor`, `validateHeaderAnnotations`, `validHTTPFieldName`, `cloneToolDescriptor`, `ValidateMCPName` + `mcpToolNameRegex` | `internal/apiserver/authorized_surface.go` (minus `loadedExportMember`/`isExposed`, which stay — V19/V20) + `mcp_schema.go` (regex+func only). Type-ownership of the validation is measured, not asserted: the V8 probe compiles this file with the stay-behinds deleted and **no** `Server`/`ExportInfo` shim |
| `interfaces.go` | `Session`, `SessionResolver`, `ToolSource`, `Invoker`, `Invocation`, `InvocationResult`, `AgentInfo`, `AuthorizationError` | `serveapi/serveapi.go` (definitions move; aliases left behind) |
| `a2a_wire.go` | `A2ARequest`, `A2ATaskSendParams`, `A2AMessage`, `A2AContent`, `A2AError`, `A2AResult` (exported wire types + JSON-RPC writers) | `internal/apiserver/a2a.go` (types exported; full server re-points via aliases) |
| `envelope.go` | `WriteMCPEnvelope`, `RequestID`, `CallbackMessage`, `AuthorizationStatus`, `ErrCallbackCapacity` | `internal/apiserver/embedded_mcp.go` (its stdlib-only framing half — V7/V26) + the capacity sentinel from `callbacks.go` (the error value is wire taxonomy — `CallbackMessage` maps it to a frozen envelope string; the **runner** that returns it is machinery and lives in `serveapi`) |

This is the issue's ask and nothing else: MCP/A2A **wire types** (`a2a_wire.go`), **envelope
framing** (`envelope.go` — `WriteMCPEnvelope`/`RequestID` are the MCP JSON-RPC envelope
discipline including the #603 request-controlled-`id` labelling, plus the error→frozen-string
taxonomy any conforming handler must emit), and the **caller-supplied-surface interfaces**
(`interfaces.go`, `descriptor.go`). No `http.Handler`, no goroutines, no channels — nothing in
this package runs.

### Routing of the evicted machinery: `serveapi`, and why

`CallbackRunner`/`RunCallback` and the embedded A2A handler go to **`serveapi`** — not a new
sibling package. Considered and rejected: a `serveapi/a2ahandler`-style sibling would mint a
second new public package for two symbols with zero consumer demand (the only consumer of either
is `serveapi` itself — V25), multiplying exactly the public-API surface the quorum objected to.
`serveapi` already exists, already ships, is already where an embedder looking for ready-made
handlers looks (it carries the SDK-backed MCP handler after this change anyway), and its closure
already tolerates both (they are stdlib-only, so the facade's ~31 prediction is unchanged). The
resulting shape is legible: **`protocol` = the frozen contract, `serveapi` = the batteries**.
`protocol` does not depend on `serveapi` (the closure gate enforces the direction).

### `serveapi` (rewired; closure: `protocol` + MCP SDK subtree ≈ 31 non-stdlib)

- `serveapi.go`: every moved type becomes an alias (`type ToolDescriptor = protocol.ToolDescriptor`
  etc. — see Back-compat); `New()` builds the runner and both handlers from local constructors.
  The two descriptor-conversion loops disappear (aliasing makes them identity).
- `callbacks.go` (new home): `CallbackRunner`, `NewCallbackRunner`, `RunCallback` moved verbatim
  from `internal/apiserver/callbacks.go`, minus `ErrCallbackCapacity` (which lives in `protocol`;
  the runner returns `protocol.ErrCallbackCapacity`).
- `a2a_handler.go` (new home): `EmbeddedA2AConfig`, `NewEmbeddedA2AHandler` moved from
  `internal/apiserver/embedded_a2a.go`, re-pointed at `protocol`'s exported wire types/writers and
  taxonomy, plus a 7-line private `writeJSON` copy (the original stays in `apiserver`'s
  handler.go — V20). One mechanical adjustment: `writeCard`'s capacity hint reads the private
  `surface.tools` field; cross-package it becomes `len(surface.All())` (or an added
  `AuthorizedSurface.Len()` — implementer's choice, behavior identical).
- `mcp_handler.go` (new home): the embedded MCP handler moved from
  `internal/apiserver/embedded_mcp.go`, importing `protocol` + `go-sdk/mcp`; carries its own
  5-line `mcpError` copy (the original stays in `apiserver` for the full server — V20).

### `internal/apiserver` (thinned; full server untouched behaviorally)

- Delete `embedded_mcp.go`, `embedded_a2a.go`, `callbacks.go`; shrink `authorized_surface.go` to
  the two stay-behind functions (`loadedExportMember`, `isExposed`) or fold them into `server.go`.
- New `protocol_compat.go`: `type a2aRequest = protocol.A2ARequest` (and the other four wire
  types), `func a2aError(...) { protocol.A2AError(...) }`-style forwards,
  `func validateMCPName(name string) error { return protocol.ValidateMCPName(name) }`. Relocate
  `mcpError` into `mcp.go` (its full-server consumer).
- `writeJSON`, `loadedExportMember`, `ExportInfo`, `Server` and everything else stay exactly where
  they are (V20).
- Tests move with their code, assertions byte-preserved — the frozen-envelope assertions from
  #592/#603 are the wire-compat oracle. New homes: `embedded_mcp_test.go`,
  `embedded_a2a_test.go`, `embedded_mcp_replay_test.go`, `callbacks_test.go` → `serveapi` (their
  code is now the facade's machinery); `authorized_surface_test.go` → descriptor-validation
  assertions to `serveapi/protocol`, stay-behind-gateway assertions (`loadedExportMember`,
  `isExposed`) remain in `apiserver`. Per coding-standards, tests are *moved and re-pointed*, not
  rewritten.

### The CI-enforced closure gate (mandatory, not optional)

`scripts/check_protocol_closure.sh` (bash-3.2-safe: no `declare -A`, no `${v,,}`), two arms:

1. **protocol arm** — enumerate `go list -deps ./serveapi/protocol`; **anti-vacuity floor**: the
   enumeration must be non-empty AND contain the literal
   `github.com/sunholo-data/ailang/serveapi/protocol` (known positive) AND contain at least one
   stdlib package — otherwise **exit 2 with a "vacuous enumeration" message, never a checkmark**.
   Then: every non-stdlib line must have prefix `github.com/sunholo-data/ailang/serveapi/protocol`.
   Violators are printed **by name**; exit 1.
2. **serveapi arm** — same enumeration for `./serveapi`; its external module roots (via
   `go list -deps -f '{{if not .Standard}}{{with .Module}}{{.Path}}{{end}}{{end}}'`) must be a
   subset of an explicit 10-entry literal allowlist: the ailang module itself + the 9 SDK-subtree
   roots measured in V9. Same anti-vacuity floor. This pins the facade at ~31 and makes a future
   `import internal/apiserver` (or any new cloud dep) a named, loud failure.

Wiring — both named, per requirement:
- **make target**: `check-protocol-closure` in `make/code-health.mk` (pattern of
  `check-boundaries`, V14 control), plus `test-check-protocol-closure` — a refusal self-test in
  the mold of the existing `test-check-changelog` CI step ("proves it can still say no").

  **Self-test strategy — AMENDED by the iteration-260 round-2 quorum (`gemini-3-1-pro`), applied
  verbatim.** The original specification ran the script against a **separate temp package**, which
  *contradicts arm 1's own anti-vacuity floor*: a temp package does not contain the literal
  `github.com/sunholo-data/ailang/serveapi/protocol`, so the script would exit 2 (vacuous
  enumeration) **before** ever reaching the intruder-detection logic that returns exit 1. The
  self-test would therefore have failed for the wrong reason, and — worse — a reader would have
  read that failure as the gate refusing correctly. Reviewer's fix, adopted as written:

  > *"instead of running against a separate temp package, have the test inject a temporary file
  > directly into the real directory (e.g. `echo 'package protocol; import _
  > "github.com/google/uuid"' > serveapi/protocol/zz_intruder_test.go`), run the script to observe
  > the expected exit 1 and count movement, and then clean up the file."*

  So the self-test has three arms, and the intruder arm now matches Mutation 1 exactly (the
  mutation table and the self-test are the same experiment, which is the point):
  (i) **intruder arm** — inject `serveapi/protocol/zz_intruder_test.go` importing
  `github.com/google/uuid` into the REAL package directory, expect exit 1 **naming uuid**, assert
  the reported non-stdlib **count moved** vs the clean run (a count assertion, not just a verdict
  flip), then remove the file and assert the tree is byte-identical to before;
  (ii) **vacuity arm** — point the script at a nonexistent package path, expect the vacuity exit 2
  and its message, never a checkmark;
  (iii) **restoration arm** — re-run the clean gate after cleanup and require exit 0, so a
  self-test that leaves debris fails loudly instead of poisoning the next run.
- **CI job**: two steps in `.github/workflows/ci.yml` next to the existing
  `make check-boundaries` step (V15): `make check-protocol-closure` and
  `make test-check-protocol-closure`.

### Downstream acceptance (World's gate, read directly — V10)

- The `protocol` closure is stdlib + itself, which passes **any** allowlist that admits the
  package's own prefix. World's edit is one literal line; under D-A(a) the narrow prefix
  `github.com/sunholo-data/ailang/serveapi/protocol` works because their matcher is prefix-based,
  with their test code unchanged.
- The MCP-SDK-dependent handler **cannot** pass their gate today (8 intruder roots, V11), which is
  why it is excluded from `protocol` and kept in `serveapi`. Whether World later admits the SDK
  subtree, vendors a projection over `protocol`'s types, or builds its own transport is World's
  decision; every option is unblocked by this split.
- **What World gets vs what World must write — stated plainly, not assumed.** Under this boundary
  World's daemon can link, gate-clean: the descriptor types + validation, `CallerSurface`/
  `AuthorizedSurface`, the host-contract interfaces, the A2A JSON-RPC wire types and writers, the
  MCP envelope writer/`RequestID`, and the wire-error taxonomy. World must still **write its own
  HTTP handlers and its own callback-bounding machinery** (or import `serveapi` and accept the
  MCP SDK subtree, which its gate rejects today). That is a real consequence of the re-scope: the
  authoring pass would have handed World a working stdlib-only A2A endpoint; this version hands
  it the contract that endpoint speaks. We believe it is also the correct architecture — World
  runs its own daemon and its charter is local-first ownership of exactly this kind of machinery —
  but the issue asked for the contract, and if World wants the ready-made A2A handler too, that is
  a one-line follow-up issue and a cheap additive change (`serveapi`'s A2A handler is stdlib-only,
  V8/V25), not a redesign. The #764 reply must state this trade explicitly so World's loop decides
  with eyes open rather than discovering it at link time.
- On completion, reply on #764 with: the package path, the measured closure (V8 shape re-run at
  the merge SHA), the gate location, and the D-A status — so their iteration loop can consume it
  without re-deriving.

---

## Examples

### Example 1: what World's daemon can now link

```go
import "github.com/sunholo-data/ailang/serveapi/protocol"

// stdlib-only closure: passes TestDaemonDependencyAllowlist with a one-line
// allowlist addition; no cloud/telemetry/SDK packages enter the build graph.
surface, err := protocol.CallerSurface(descriptors) // validated, sorted, detached

// World's OWN handler speaks the shared dialect via the exported framing:
func (h *worldA2A) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    var req protocol.A2ARequest
    // … decode, dispatch against surface …
    protocol.A2AResult(w, req.ID, result) // same JSON-RPC envelope as ailang's server
}
```

### Example 2: existing serveapi consumers — no change

```go
// Compiles byte-identically before and after: every serveapi name still exists.
srv, err := serveapi.New(serveapi.Config{Resolver: r, Tools: t, Invoker: i,
    Agent: serveapi.AgentInfo{Name: "host", Version: "1"}})
srv.Mount(mux)
```

---

## Back-compat (published seam: chosen option and why)

**Chosen: serveapi keeps its entire current surface; moved types become aliases; nothing is
deprecated; no import path breaks.** `type ToolDescriptor = protocol.ToolDescriptor` (and the
rest) means existing consumer code — struct literals, interface implementations, conversions —
compiles unchanged, because an alias *is* the same type. The one observable change: reflection
(`reflect.Type.PkgPath()`) on these types now reports `…/serveapi/protocol` instead of
`…/serveapi`; no known consumer does this (World is pre-consumption; in-repo importers are V17).
Wire behavior is pinned by the moved frozen-envelope tests. The rejected alternative — leaving
`serveapi` on `internal/apiserver` and only *adding* `protocol` — would have left the published
facade at 479 packages and duplicated every wire definition (drift; the #603 class of bug).

The oracle: `serveapi/serveapi_external_test.go` (V23) passes with exactly two call-site renames
(`apiserver.RunCallback` → bare `RunCallback` — the runner now lives in `serveapi`, the test's own
package) and the `internal/apiserver` import line **deleted**; all assertions byte-identical. Any
further forced edit to that file is a back-compat smell the evaluator should treat as a failure.

## Conflict Surface

Not a parser/typechecker change; the surface is package topology and shared invariants:

- **Who imports what today (V17):** `internal/apiserver` has exactly two non-test importers —
  `cmd/ailang/serve_api.go` (full server; untouched) and `serveapi` (rewired here). No hidden
  consumers.
- **Files a concurrent change would collide with:** the four moved files + `a2a.go`, `mcp.go`,
  `mcp_schema.go`, `server.go` in `internal/apiserver`; all of `serveapi/`; `make/code-health.mk`;
  `.github/workflows/ci.yml`; `scripts/`. Recent churn here: #714 (2026-08-14, exit(0) handling in
  route/A2A/MCP handlers) is already in base; open PRs touch none of these paths (V24). The rig's
  concurrent missions (motoko ABI, eval lanes) do not touch `apiserver`.
- **Shared invariants that must survive the move** (`.claude/rules/api-server.md`): one
  authorized-surface gateway — `loadedExportMember` stays in `apiserver` for loaded exports;
  `CallerSurface`/`AuthorizedSurface` remain the single membership path for embedded callbacks,
  now from `protocol`; `@nomcp` stays a full-server MCP-projection concern (untouched — it lives
  in the export path, not the embedded path). The rule file's path references
  (`authorized_surface.go`, `mcp_schema.go::validateMCPName`) must be updated in M3.
- **`scripts/check_boundaries.sh`:** anchors on `internal/<pkg>` imports (CORE vs DASHBOARD);
  `serveapi/protocol` lives outside `internal/` and imports nothing internal, so the existing gate
  is unaffected; the new gate is a sibling, not a modification.
- **Release surface:** under (a) nothing about tagging changes and delivery is the ordinary main-module tag (the mechanism Mark's attended v0.34.0 ruling already assumes). Under (b) the release surface changes materially and must be specified before implementation, not after — see D-A / ledger `D-35`.

## Implementation Plan

**M1 — extract `serveapi/protocol` (day 1)**
- [ ] Create the four `protocol` files above by *moving* code (git-visible moves, export renames only)
- [ ] `internal/apiserver` compat shim (`protocol_compat.go`, `mcpError` relocation, shrink `authorized_surface.go`)
- [ ] Move + re-point the descriptor-validation tests; `go test ./internal/apiserver/... ./serveapi/protocol/...` green
- [ ] Measure: `go list -deps ./serveapi/protocol` non-stdlib = itself only (must reproduce V8's shape)

**M2 — rewire `serveapi` (day 2)**
- [ ] `callbacks.go`, `a2a_handler.go`, `mcp_handler.go` moved in (with the `writeCard` capacity-hint adjustment and the `writeJSON`/`mcpError` private copies); aliases in `serveapi.go`; delete the `internal/apiserver` import
- [ ] Move + re-point the four machinery test files; `serveapi_external_test.go`: the two-rename + import-delete edit only; green
- [ ] Measure and record the facade closure (predicted ~31; hard ceiling 40)

**M3 — gate + docs (day 3)**
- [ ] `scripts/check_protocol_closure.sh` (both arms, anti-vacuity floor, by-name violator output, bash-3.2-safe)
- [ ] `make check-protocol-closure` + `make test-check-protocol-closure` in `make/code-health.mk`; two ci.yml steps
- [ ] Run the mutation table below and record each observed red
- [ ] `CHANGELOG` (v0.18-current.md), `docs/docs/guides/serve-api.md` (embedding section), `.claude/rules/api-server.md` path updates, `ARCHITECTURE.md` one-liner for the new public package
- [ ] Reply on #764 with measured closure + gate location + D-A status

Scope check: 3 focused days; the extraction itself is measured-small (9 leaked symbols, all
stdlib). If M1 surfaces a leak the probe missed (e.g. a build-tagged file), stop and re-measure
rather than widening silently — the probe covered the darwin/arm64 file set only.

### Files to Modify/Create

**New files:**
- `serveapi/protocol/descriptor.go` — descriptors + type-owned validation (~160 LOC, moved)
- `serveapi/protocol/interfaces.go` — host contract interfaces (~60 LOC, moved)
- `serveapi/protocol/a2a_wire.go` — A2A JSON-RPC wire types + writers (~70 LOC, moved/exported)
- `serveapi/protocol/envelope.go` — MCP envelope framing + wire-error taxonomy (~80 LOC, moved)
- `serveapi/callbacks.go` — bounded callback runner (~60 LOC, moved; returns `protocol.ErrCallbackCapacity`)
- `serveapi/a2a_handler.go` — embedded A2A handler (~175 LOC, moved; private `writeJSON` copy)
- `serveapi/mcp_handler.go` — SDK-backed embedded MCP handler (~240 LOC, moved; private `mcpError` copy)
- `internal/apiserver/protocol_compat.go` — wire-type aliases + forwards (~40 LOC)
- `scripts/check_protocol_closure.sh` — closure gate (~120 LOC)
- moved test files under `serveapi/` and `serveapi/protocol/` (assertions byte-preserved)

**Modified files:**
- `serveapi/serveapi.go` — aliases + rewired `New()` (~-60/+40 LOC)
- `serveapi/serveapi_external_test.go` — two renames + import (±3 LOC)
- `internal/apiserver/authorized_surface.go` — shrinks to stay-behind gateway (~-120 LOC)
- `internal/apiserver/a2a.go` — wire types become aliases (~-40/+10 LOC)
- `internal/apiserver/mcp.go` — receives `mcpError` (~+8 LOC)
- `internal/apiserver/mcp_schema.go` — `validateMCPName` forwards (~-10/+4 LOC)
- `make/code-health.mk` — two targets (~+8 LOC)
- `.github/workflows/ci.yml` — two steps (~+6 LOC)
- `docs/docs/guides/serve-api.md`, `.claude/rules/api-server.md`, `ARCHITECTURE.md`, `changelogs/v0.18-current.md` — doc updates

Deleted: `internal/apiserver/{embedded_mcp,embedded_a2a,callbacks}.go` (+ their moved tests).

## Success Criteria

Each criterion was **run on pristine `origin/dev` (`a201237ca`) first**; the base result is part
of the criterion, so none is vacuously green.

- [ ] **AC1 — protocol closure.** `go list -deps ./serveapi/protocol | awk -F/ '$1 ~ /\./'`
  prints ≥ 1 line and every line has prefix `github.com/sunholo-data/ailang/serveapi/protocol`.
  **Base: `go list` fails, `directory not found` (V13) — red.**
- [ ] **AC2 — facade closure.** `go list -deps ./serveapi | <nonstdlib count>` ≤ 40 AND the V22
  cloud-pattern grep over the same list = **0**. **Base: 479 / 325 (V1, V22) — red.** Record the
  exact post-change number in the implementation report.
- [ ] **AC3 — gate exists and is green.** `make check-protocol-closure` exits 0.
  **Base: `No rule to make target` (V14) — red.**
- [ ] **AC4 — gate can refuse.** `make test-check-protocol-closure` exits 0, and its uuid arm's
  output shows the intruder **named** and the non-stdlib **count moved** vs the clean run.
  **Base: target absent — red.**
- [ ] **AC5 — CI wiring.** `grep -c check-protocol-closure .github/workflows/ci.yml` ≥ 2
  (gate + self-test). **Base: 0, with in-file control `check-boundaries` = 1 (V15) — red.**
- [ ] **AC6 — behavior preserved.** `go test -count=1 ./serveapi/... ./internal/apiserver/...`
  green (base green too — V16 — so the informative content is: still green *after* the move, with
  the moved frozen-envelope assertions byte-preserved and `serveapi_external_test.go` limited to
  the two-rename + import-delete diff, checked via `git diff --stat` on that file).
- [ ] **AC7 — full suite + boundaries.** `make test` and `make check-boundaries` green (base:
  green; guards against collateral damage — this is the only deliberately repo-wide criterion).
- [ ] Documentation updated (guide, rule file, ARCHITECTURE.md, CHANGELOG) and #764 replied to
  with measured numbers.

## Mutation Table

One row per gate/refusal branch; run in M3 and record each observed red. Removal mutants prove a
check FIRES; the addition mutant proves the enumerator LOOKS.

| # | Mutation | What must go red (observed output to record) |
|---|---|---|
| 1 | **ADDITION (enumerator looks):** add `_ "github.com/google/uuid"` to a `serveapi/protocol` file | protocol arm exits 1 **naming `github.com/google/uuid`**, and the reported non-stdlib count **moves** (1 → ≥2). A verdict flip alone does not pass this row. |
| 2 | Add `_ "github.com/sunholo-data/ailang/internal/apiserver"` to a protocol file | `go build` fails first (import cycle via apiserver→protocol); if built in isolation, protocol arm exits 1 naming the apiserver package and its cloud subtree |
| 3 | Re-add `internal/apiserver` import to `serveapi` | serveapi arm exits 1 naming disallowed roots (e.g. `cloud.google.com/go/...`); count jumps ~31 → ~480 |
| 4 | Point the gate at a nonexistent package path | anti-vacuity floor: exit 2 with the vacuous-enumeration message, **not** a green checkmark |
| 5 | Neuter the stdlib classifier in the script (treat everything as stdlib) | self-test uuid arm dies: intruder no longer named ⇒ `test-check-protocol-closure` exits non-zero |
| 6 | Delete the ci.yml gate step | AC5's grep count drops below 2 — caught at review/evaluation time (this row documents that the gate's *wiring* is only guarded by AC5, which is why AC5 is a criterion and not prose) |
| 7 | Weaken a moved frozen-envelope test (e.g. drop the #603 labelling assertion) | `git diff` on moved test files shows a non-mechanical change — evaluator instruction: moved tests must be byte-preserved modulo package/import lines; any assertion edit fails AC6 |

## Testing Strategy

- **Unit:** moved tests run in their new homes unchanged (envelope framing, callback capacity,
  descriptor validation, A2A dispatch, MCP replay labelling).
- **Gate:** `test-check-protocol-closure` self-test (uuid arm, vacuity arm, count-movement arm) —
  the gate's refusal branches are themselves CI-tested, mirroring `test-check-changelog`.
- **Integration:** `serveapi_external_test.go` exercises the public embedding path end-to-end
  (fixture host, Mount, both handlers).
- **Manual:** one `curl` smoke of `ailang serve-api` (full server) MCP + A2A endpoints to confirm
  the alias shim changed nothing — the full-server path has no closure gate and its behavior is
  covered by its existing tests, so this is belt-and-braces only.

## Deferred Decisions

- Exact file names / splits inside `serveapi/protocol` — implementer may reorganize as long as the
  package set and closure are as specified.
- Whether `RunCallback` remains a free generic function or gains a method form — implementer.
- Whether the serveapi-arm allowlist lives inline in the script or in a checked-in data file —
  implementer (script literal recommended: edits are reviewable diffs, matching World's pattern).
- Godoc wording for the new public package — implementer, but must state the stdlib-only guarantee
  and point at the gate that enforces it.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Probe missed a platform-tagged file (probe ran on darwin/arm64 only) | Med | CI gate runs on linux-amd64; M1 re-measures there before M2 proceeds; stated stop-and-remeasure rule |
| Hidden semantic coupling the compiler can't see (e.g. init-order, test doubles reaching into moved internals) | Low | V19/V20 sweeps found none outside the moved test set; moved tests byte-preserved as the oracle |
| World's gate drifts before delivery (allowlist read at `48ef27518`) | Low | Design depends only on "stdlib-only passes any prefix-admitting allowlist"; #764 reply includes re-derived numbers |
| Alias identity change breaks a reflection-based consumer | Low | No known consumer (V17, World pre-consumption); called out in CHANGELOG |
| D-A resolved to (b) after shipping (a) | Low — **decision frozen before implementation** | Mark selected **(a)** on 2026-08-23, which is precisely what the round-2 gate demanded. (a) ships with **no promise of transparent later conversion**; the unverified "import paths stable, only tagging changes" mitigation stays **withdrawn**, so a post-hoc (a)→(b) move is a breaking-change project with its own doc and the reviewer's demanded external-consumer verification row. |

## Timeline

**Day 1:** M1 extraction + moved tests green. **Day 2:** M2 facade rewire + closure measured.
**Day 3:** M3 gate + self-test + mutation table run + docs + #764 reply. Total ≈ 3 days, one
sprint, no decomposition needed.

## Appendix: V8 probe reconstruction recipe

The probe is deliberately absent from the worktree (scratch code is deleted before delivery), so
this recipe is the measurement's reproducibility. Run from the repo root at `a201237ca`
(darwin/arm64, go1.26.6). Line numbers are anchored to that SHA — if any source file has changed,
re-derive the ranges from the symbol names in comments before trusting the numbers.

```bash
mkdir -p internal/protoprobe

# (1) descriptors + validation: authorized_surface.go, package renamed, MINUS the
# stay-behind gateway lines 28-41 (loadedExportMember + isExposed). Deliberately NO
# shim for Server/ExportInfo: compiling this measures that validation is type-owned.
sed -e 's/^package apiserver$/package protoprobe/' -e '28,41d' \
  internal/apiserver/authorized_surface.go > internal/protoprobe/descriptor.go

# (2) MCP name validation: regex (mcp_schema.go:11-14) + validateMCPName (:103-108)
{ printf 'package protoprobe\n\nimport (\n\t"fmt"\n\t"regexp"\n)\n\n'
  sed -n '11,14p' internal/apiserver/mcp_schema.go
  sed -n '103,108p' internal/apiserver/mcp_schema.go; } > internal/protoprobe/mcpname.go

# (3) caller-supplied-surface interfaces: serveapi.go:23-35 (Session, SessionResolver,
# ToolSource, Invoker) + :46-75 (Invocation, InvocationResult, AgentInfo,
# AuthorizationError), skipping serveapi's ToolDescriptor duplicate at 37-44
{ printf 'package protoprobe\n\nimport (\n\t"context"\n\t"encoding/json"\n\t"net/http"\n)\n\n'
  sed -n '23,35p;46,75p' serveapi/serveapi.go; } > internal/protoprobe/interfaces.go

# (4) A2A JSON-RPC wire types + writers: a2a.go:274-322
{ printf 'package protoprobe\n\nimport (\n\t"encoding/json"\n\t"net/http"\n)\n\n'
  sed -n '274,322p' internal/apiserver/a2a.go; } > internal/protoprobe/a2a_wire.go

# (5) envelope framing + wire-error taxonomy: ErrCallbackCapacity (callbacks.go:10-12)
# + embedded_mcp.go:177-236 (requestID, writeMCPCallbackError, callbackMessage,
# writeMCPEnvelope, authorizationStatus — range stops BEFORE the SDK-typed mcpError)
{ printf 'package protoprobe\n\nimport (\n\t"context"\n\t"encoding/json"\n\t"errors"\n\t"net/http"\n)\n\n'
  sed -n '10,12p' internal/apiserver/callbacks.go
  printf '\n'
  sed -n '177,236p' internal/apiserver/embedded_mcp.go; } > internal/protoprobe/envelope.go

go build -gcflags=-e ./internal/protoprobe/ && echo BUILD_OK
go list -deps ./internal/protoprobe | awk -F/ '$1 ~ /\./'                      # non-stdlib lines
go list -deps ./internal/protoprobe | awk -F/ '$1 ~ /\./ {n++} END{print n+0}' # count
go list -deps ./internal/protoprobe | wc -l                                    # anti-vacuity
go list -deps -f '{{if not .Standard}}{{with .Module}}{{.Path}}{{end}}{{end}}' \
  ./internal/protoprobe | sort -u                                              # module roots

rm -rf internal/protoprobe && go build ./serveapi ./internal/apiserver         # cleanup + sanity
```

Observed on the recorded run (2026-08-23): `BUILD_OK` with no shim file present; non-stdlib lines
= exactly `github.com/sunholo-data/ailang/internal/protoprobe`; count `1`; total enumeration
`188`; module roots = `github.com/sunholo-data/ailang` only; after cleanup `REPO_STILL_BUILDS`.
Note the probe needs **no shim at all**, unlike the authoring-pass probe (V5's 9 leaked symbols):
5 of those 9 were the a2a wire types (now real extractions, step 4), `validateMCPName` is a real
extraction (step 2), `ExportInfo`/`Server` disappeared with the stay-behind deletion (step 1), and
`writeJSON` was only ever needed by the evicted A2A handler.

## Revision history & quorum record

**2026-08-23 — authoring pass (iteration 260).** Initial version: `protocol` scoped to the full
SDK-free subset, including `CallbackRunner` and the embedded A2A `http.Handler`.

**2026-08-23 — review quorum: BLOCKED (2/2 reviewers present, `absent_reviewers` empty).**

- **Objection 1 (gpt5-6-sol): the proposed core was not protocol-only** — it froze operational
  machinery (`CallbackRunner`, an executable A2A handler) into a new public core without
  justification, violating the minimal-frozen-core / route-to-extension axioms. **Upheld.** The
  controller's measurement settled it: issue #764's own wording asks for wire types, envelope
  framing, and caller-supplied-surface interfaces — neither evicted symbol is in the ask.
  **Disposition:** `protocol` re-scoped to the issue's three categories (+ measured type-owned
  validation); runner and A2A handler routed to `serveapi` with reasoning recorded above; closure
  of the re-scoped package re-measured from scratch (V8: zero shims, stdlib + itself only);
  World-facing consequence stated plainly in "Downstream acceptance".
- **Objection 2 (gemini-3-1-pro): V17's negative claim outran its instrument** — the importer
  search was bounded to `cmd/ internal/ serveapi/` with no stated justification, leaving 72 `.go`
  files in other directories unexamined. **Upheld on method** (the conclusion happened to
  survive: the controller's unbounded re-measurement found the same 2 importing packages / 3
  files). **Disposition:** V17 replaced — primary evidence is now the toolchain's own package
  graph (`go list` incl. test imports, whole module), corroborated by an unbounded repo-wide
  grep, with a directory-existence scope assertion (`pkg`/`test`/`testdata` do not exist here;
  saying so is part of the answer). The whole Verification Log was then swept for the same
  defect class (see "Negative-result discipline" above): V18 gained a scope assertion and a
  same-scope control, V19 was widened repo-wide for exported symbols, V20's package-bounded
  scope was shown defensible-by-construction. Class fix, not instance patch.
- **Controller addition (not a reviewer objection):** the load-bearing V8 measurement rested on a
  deleted scratch probe nobody could re-run. **Disposition:** full reconstruction recipe recorded
  in the appendix above; the re-scoped probe was rebuilt from that exact recipe, so the recipe is
  tested, not transcribed.

**2026-08-23 — re-quorum after revision: BLOCKED AGAIN (2/2 present, `absent_reviewers` empty,
$0.129129).** Both round-2 objections were on surfaces round 1 had not named and round 2's edits
had not touched — i.e. pre-existing holes, not regressions introduced by the revision. Both were
resolved by the controller under the mission's **narrow-refinement carve-out**: each carried a
concrete reviewer-authored `proposed_fix`, and neither disputed the design DIRECTION (no reviewer
has questioned that a protocol-only package should exist, across four reviews). The reviewers'
fixes were applied **verbatim**, not paraphrased and not overridden.

- **Objection 3 (gpt5-6-sol): D-A cannot be non-blocking, because the migration claim holding it
  open is unverified.** Sustained in full — see the amended D-A row. The doc had asserted that
  (a)→(b) is additive with stable import paths, and used that assertion to downgrade its own only
  human decision; in fact only the *import path* is stable, while module ownership and version
  resolution change, so a consumer requiring only the main module can stop resolving the package.
  **Disposition:** claim **withdrawn** rather than repaired; D-A promoted to a blocking
  design-freeze gate, filed as mission ledger `D-35`; implementation does not begin until Mark
  selects; the reviewer's demanded external-consumer verification row is named as required-if-(b)
  and explicitly marked *not yet measured*.
- **Objection 4 (gemini-3-1-pro): the CI self-test contradicted the gate's own anti-vacuity
  floor.** Sustained, and mechanically exact: a *separate temp package* cannot contain the literal
  `…/serveapi/protocol` that arm 1's floor requires, so the self-test would have exited 2 on
  vacuity before ever reaching the intruder logic it exists to exercise — a self-test that fails
  for the wrong reason, which reads exactly like a gate refusing correctly. **Disposition:** the
  reviewer's fix adopted verbatim — inject a temporary intruder file into the **real** package
  directory, observe exit 1 + count movement, clean up; plus a third restoration arm so a
  self-test that leaves debris fails loudly. The self-test and Mutation 1 are now the same
  experiment.

**Round-surface tracking (mission-control Gate 2, from round 3 on).** R1: protocol scope +
verification method. R2: module boundary + CI self-test mechanics. The objections are **spread
across four distinct surfaces**, not localising onto one consumer, and no reviewer has flipped to
pass — so the signal is an immature document being repaired, **not** a scoping error calling for
decomposition. Recorded so a later reader can tell the two apart.

**Status leaving iteration 260:** every reviewer objection is answered. The doc was **parked
`needs-human-review` on `D-35` alone** (module boundary). It was not parked on design direction, and
it was not `PARKED-ON-LANE` — nothing here unblocked on a clock.

**Status entering iteration 261 — UNPARKED AND FROZEN.** Mark answered `D-35` with **(a)** at
`2026-08-23T19:01:24Z` (directive on `#745`, author `MarkEdmondson1234`). The design-freeze gate the
round-2 quorum demanded is satisfied, ledger `D-35` is **RESOLVED**, and the doc routes to a sprint
plan. No reviewer objection remains open, and no reviewer disputed the design direction across four
reviews. Note for the planner: this doc had **no** sprint plan when the ruling landed, so rule
3b(vii)'s doc↔plan divergence hazard does not apply — the plan is authored against the frozen text.

No reviewer claim was found wrong. One nuance held rather than conceded: the wire-error taxonomy
(`ErrCallbackCapacity`, `CallbackMessage`, `AuthorizationStatus`) stays in `protocol` as part of
envelope framing — the frozen envelope strings are contract, and a downstream handler-writer
needs them to emit conforming envelopes — but this is flagged in the decision table as the
re-scope's maximal point, strikeable to `serveapi` at zero design cost if the next review
disagrees.

## Related Documents

- [m-mcp-exact-tool-surface-lane-b.md](../implemented/v1_0_0/m-mcp-exact-tool-surface-lane-b.md) — #498 Lane B: the embeddable contract this doc makes dependency-clean (its frozen-envelope tests are this doc's wire oracle)
- [m-serveapi-surface-drops.md](../implemented/v0_20_0/m-serveapi-surface-drops.md) — earlier serveapi surface discipline
- Neural-search near-matches reviewed and confirmed distinct (top score 0.45, below the 0.65
  warn-reject band): `m-dx-route-request-context` (route DX, v0.10.0), `m-serve-api-dx-sprint-plan`
  (v0.9.4), `m-arch-boundaries-eval-exclusion-tighten` (internal layering, different gate).

## References

- Issue [#764](https://github.com/sunholo-data/ailang/issues/764); prerequisites #498 (shipped), context #586, #603 (envelope labelling), #714 (exit(0) handling, in base)
- ailang-world `host/daemon/daemon_test.go` (allowlist gate, read at `48ef27518`)
- [ARCHITECTURE.md](../../ARCHITECTURE.md) — layer boundaries; `scripts/check_boundaries.sh` as the sibling-gate pattern

## Future Work

- D-A(b): promote `serveapi/protocol` to a nested zero-require module if World's review wants an
  empty consumer module graph. **Not a free later move** — round 2 struck the claim that it is
  (module ownership and version resolution change, so a main-module-only consumer can stop
  resolving the package). If it is wanted, it is wanted *now*, via `D-35`; after shipping (a) it
  becomes a breaking-change project with its own doc and its own consumer-resolution measurement.
- A public, SDK-backed `serveapi/mcpserve` package if a non-facade embedder ever wants the MCP
  handler without `serveapi` (no current demand).
- Extend the closure-gate pattern to other public seams (`cmd/wasm` is already small; nothing else
  is public today).

---

**Document created**: 2026-08-23
**Last updated**: 2026-08-23 (revision pass after iteration-260 quorum BLOCK — see Revision
history & quorum record)

**DESIGN_DOC_PATH**: `design_docs/planned/m-serveapi-protocol-only-module.md`
