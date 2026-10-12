# Pure-row / iface-purity M1–M2 execution evidence

Refs #1443.

Branch: `coordinator/task-7861d01e`.
Base and remote dev: `7c43ece3ce7abac639aa04777c3abc56e4117e65`.
The landed row-unification validation and latent-function-value tests are present.
No model credential is needed; Z3 is available. Teaching prompt loaded: v0.16.7.
Cache base: v6 (alias-body closure); selected next unused version: v7.

## HOLD remeasurement before M2

Every std module checked on the base with canonical module paths and caching disabled.
50 modules, 484 function exports, 168 labelled outer rows, 17 open outer rows,
271 keyword-pure exports, and 27 rowless/unannotated-callback combinators.
The historical 26 grew by `std/list.mapAccumL`. Before M2 all exports carry the
stub `Purity: true`, so all 168 labelled functions are false positives.
Open and labelled counts overlap; they are not summed.

`std/list.flatMap`, `std/list.sortBy`, and `std/option.flatMap` have callback-shared
outer tails. The other 24 combinators have empty outer rows. All 27 should report
true. `std/list.mapE` has a declared shared `e` and should report false; all six
streaming exports have outer labels and should report false.

### Measured schemes

```text
std/ai/streaming.anthropicStream : (string, string, string) -> Result[StreamConn, StreamErrorKind] ! {AI[mode=fixed], Net, Stream}
std/ai/streaming.callStream : (string, string, string) -> Result[string, {code: string, message: string, retryable: bool}] ! {AI[mode=fixed], Net, Stream}
std/ai/streaming.disconnect : StreamConn -> () ! {Stream}
std/ai/streaming.onEvent : (StreamConn, StreamEvent -> bool ! {...e}) -> () ! {Stream, ...e}
std/ai/streaming.openaiCompatStream : (string, string, string) -> Result[StreamConn, StreamErrorKind] ! {AI[mode=fixed], Net, Stream}
std/ai/streaming.runEventLoop : StreamConn -> () ! {Stream}
std/list.any : (a -> bool, list[a]) -> bool
std/list.filter : (a -> bool, list[a]) -> list[a]
std/list.findIndex : (a -> bool, list[a]) -> Option[int]
std/list.flatMap : (a -> list[b] ! {...ρ2}, list[a]) -> list[b] ! {...ρ2}
std/list.foldl : ((α32, a) -> α32, α32, list[a]) -> α32
std/list.foldr : ((a, α38) -> α38, α38, list[a]) -> α38
std/list.map : (a -> b, list[a]) -> list[b]
std/list.mapAccumL : ((s, a) -> (b, s), s, list[a]) -> (list[b], s)
std/list.mapE : (a -> b ! {...e}, list[a]) -> [b] ! {...e}
std/list.sortBy : ((a, a) -> int ! {...ρ2}, list[a]) -> list[a] ! {...ρ2}
std/list.takeFlatMap : (int, a -> list[b], list[a]) -> list[b]
std/list.takeMap : (int, a -> b, list[a]) -> list[b]
std/list.zipWith : ((α128, α129) -> α130 ! {...ρ3}, list[α128], list[α129]) -> list[α130]
std/option.filter : (α53 -> bool ! {...ρ3}, Option[α53]) -> Option[α53]
std/option.flatMap : (α20 -> Option[b] ! {...ρ2}, Option[α20]) -> Option[b] ! {...ρ2}
std/option.map : (α9 -> α4 ! {...ρ3}, Option[α9]) -> Option[α4]
std/result.flatMap : (α58 -> Result[b, α59] ! {...ρ2}, Result[α58, α59]) -> Result[b, α59]
std/result.map : (α14 -> α6 ! {...ρ3}, Result[α14, α9]) -> Result[α6, α9]
std/result.mapErr : (α37 -> α31 ! {...ρ3}, Result[α36, α37]) -> Result[α36, α31]
std/string.foldChars : ((α47, string) -> α47, α47, string) -> α47
std/string.foldSlices : (string, string, α69, (α69, string) -> α69) -> α69
std/string.mapSlicesJoin : (string, string, string -> string) -> string
std/xml.flatMapChildren : (XmlNode, XmlNode -> list[a]) -> list[a]
std/xml.foldChildren : (XmlNode, α50, (α50, XmlNode) -> α50) -> α50
std/xml.foldChildrenStep : (XmlNode, α54, (α54, XmlNode) -> FoldStep[α54]) -> α54
std/xml.mapChildren : (XmlNode, XmlNode -> a) -> list[a]
std/xml.parseFold : (string, string, a, (a, XmlNode) -> a) -> Result[a, string]
std/xml.parseFoldStep : (string, string, a, (a, XmlNode) -> FoldStep[a]) -> Result[a, string]
```

## Scope

M1 and M2 only. M3–M4, #1718, SMT admission, corpus migrations and teaching
material migrations remain pending. This is a partial implementation for review;
the complete sprint is not ready for release.

## M1

The 16-case signature matrix established six failures against the old checker:
IO, Debug, Declassify, scoped Declassify, var-only and mixed open declarations.
The implemented check rejects all six with the function name and both repairs;
non-pure controls, absent rows and explicit empty rows remain accepted.

M1 checkpoint: focused `go test ./internal/pipeline/...` passes; `make lint`
reports zero issues; boundaries, file sizes and changelog checks pass.

## M2

The companion `internal/iface/purity.go` derives the signal after builder row
restoration. Function exports inspect only their outer row; callback-shared tails
are latent only with an available declaration and no declared outer variable.
Return-only and unrelated tails fail closed. Non-function exports use a deep,
cycle-safe walk: effect rows anywhere (including nested functions) must be empty
and closed. Record polymorphism itself does not imply an effect.

The unit matrix and public builder test failed against the always-true stub before
implementation. They now pass, including 20 repetitions and a cyclic type graph.
The cache test demonstrated that a v6 manifest retained stale always-true metadata;
v7 now evicts it. Actual checked empty, IO, declared-open and latent rows preserve
their derived values through stamped artifact serialization and deserialization.
Existing cache migration tests compare published stamps against the current
version rather than an obsolete hard-coded v6 expectation.

Census after M2: **311 pure / 173 impure** of 484 functions. All **168 labelled**
and **14 declared-open** outer rows are false; these sets overlap. All **271
keyword-pure** exports and **27 combinators** remain true. All six streaming
exports and list.mapE are false. Three combinator outer tails remain callback-shared
(list.flatMap, list.sortBy, option.flatMap); the other 24 are empty and closed.

Serve-api no longer overwrites iface purity while extracting annotations. A direct
regression preserves both possible iface values despite AST keyword spelling.
The in-memory MCP tools/list integration checks the declared-open control, pure
tag, and default read-only hint against the server export signal. Its obsolete
contradictory IO fixture was replaced by a valid declared-open fixture as part of
M2 consumer testing; no M3 source corpus or SMT admission changes were made.

## Validation and environment

- Focused pipeline cache/effect/purity tests and complete iface/apiserver packages pass.
- `make test-core` passes under a temporary Linux subreaper wrapper. Its direct
  run first exposed an obsolete v6 assertion in the alias-cache regression,
  updated to require the current version. Separately, the untouched effects
  lifetime test failed both directly and in isolation: its stopped fixture was a
  zombie owned by PID 1. A `/tmp`-only Python wrapper sets Linux
  `PR_SET_CHILD_SUBREAPER`, launches the unchanged command, and reaps orphaned
  children. The isolated lifetime test and entire core gate then pass. No effects
  implementation/test changes or skips were introduced. CI with a reaping init
  does not need this container accommodation.
- `make lint`: zero issues. `make check-file-sizes`: pass (builder.go 781 lines).
- `make check-boundaries`, `make check-changelog`, formatting and diff checks pass.
- Both new purity functions have 100% statement coverage in the focused unit run.
- Fresh CLI probes report streaming false, list.mapE false, and both flatMap
  controls true. The two M1 positive example modules check successfully.
- The CI stdlib freeze gate initially rejected the intended metadata change.
  `make freeze-stdlib` refreshes 27 modules (54 JSON/hash files): 166 true-to-false
  purity changes, with no type, effect, export or schema drift. The gate covers
  48 top-level modules; the census also checks both nested modules (50 total).
- No full `make test`, push, M3/M4 implementation, or #1718 work performed.

## Coordinator PR body

Refs #1443.

Implements only M1–M2: reject contradictory pure signatures, derive checked iface
purity, preserve callback-latent combinators, use the signal in serve-api/MCP,
and invalidate old metadata with cache v7. Re-freezes the intended stdlib JSON
purity change. M3–M4 remain pending; this partial sprint is not release-ready.

Validation: focused Go packages, core gate (container subreaper), lint, file sizes,
boundaries, changelog and stdlib freeze verification. Census: 168 labelled and
14 declared-open exports now false; 271 keyword-pure exports and 27 combinators
remain true. No push; coordinator opens the PR.


## Files created

- `changelogs/unreleased/2026-10-12-pure-row-iface-purity.md`
- `design_docs/planned/v0_53_0/m-pure-row-and-iface-purity-m1-m2-validation.md`
- `internal/apiserver/iface_purity_test.go`
- `internal/iface/purity.go`
- `internal/iface/purity_test.go`
- `internal/pipeline/cache_purity_test.go`
- `internal/pipeline/pure_signature_test.go`
- `internal/pipeline/purity_census_test.go`

## Files modified

- `.ailang/state/sprints/sprint_M-PURE-ROW-AND-IFACE-PURITY.json`
- `.stdlib-golden/ai.json`
- `.stdlib-golden/ai.sha256`
- `.stdlib-golden/clock.json`
- `.stdlib-golden/clock.sha256`
- `.stdlib-golden/cognition.json`
- `.stdlib-golden/cognition.sha256`
- `.stdlib-golden/debug.json`
- `.stdlib-golden/debug.sha256`
- `.stdlib-golden/dom.json`
- `.stdlib-golden/dom.sha256`
- `.stdlib-golden/env.json`
- `.stdlib-golden/env.sha256`
- `.stdlib-golden/extension.json`
- `.stdlib-golden/extension.sha256`
- `.stdlib-golden/fs.json`
- `.stdlib-golden/fs.sha256`
- `.stdlib-golden/game.json`
- `.stdlib-golden/game.sha256`
- `.stdlib-golden/gzip.json`
- `.stdlib-golden/gzip.sha256`
- `.stdlib-golden/io.json`
- `.stdlib-golden/io.sha256`
- `.stdlib-golden/list.json`
- `.stdlib-golden/list.sha256`
- `.stdlib-golden/net.json`
- `.stdlib-golden/net.sha256`
- `.stdlib-golden/package.json`
- `.stdlib-golden/package.sha256`
- `.stdlib-golden/process.json`
- `.stdlib-golden/process.sha256`
- `.stdlib-golden/rand.json`
- `.stdlib-golden/rand.sha256`
- `.stdlib-golden/secret.json`
- `.stdlib-golden/secret.sha256`
- `.stdlib-golden/sem.json`
- `.stdlib-golden/sem.sha256`
- `.stdlib-golden/sharedindex.json`
- `.stdlib-golden/sharedindex.sha256`
- `.stdlib-golden/sharedmem.json`
- `.stdlib-golden/sharedmem.sha256`
- `.stdlib-golden/smoke.json`
- `.stdlib-golden/smoke.sha256`
- `.stdlib-golden/stream.json`
- `.stdlib-golden/stream.sha256`
- `.stdlib-golden/tar.json`
- `.stdlib-golden/tar.sha256`
- `.stdlib-golden/terminal.json`
- `.stdlib-golden/terminal.sha256`
- `.stdlib-golden/trace.json`
- `.stdlib-golden/trace.sha256`
- `.stdlib-golden/web.json`
- `.stdlib-golden/web.sha256`
- `.stdlib-golden/zip.json`
- `.stdlib-golden/zip.sha256`
- `design_docs/planned/v0_53_0/m-pure-row-and-iface-purity-sprint-plan.md`
- `internal/apiserver/mcp_tool_hints_test.go`
- `internal/apiserver/routes.go`
- `internal/iface/builder.go`
- `internal/pipeline/alias_body_closure_runtime_test.go`
- `internal/pipeline/cache_invalidation_test.go`
- `internal/pipeline/cache_key.go`
- `internal/pipeline/validate_effects.go`
