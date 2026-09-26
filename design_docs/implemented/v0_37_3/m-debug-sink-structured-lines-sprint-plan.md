# Sprint Plan: M-DEBUG-SINK-STRUCTURED-LINES

## Summary
One shared `effects.DebugSink` replaces the two divergent Debug flush loops so a JSON-object
`Debug.log` line reaches stderr verbatim in `run`, `run --batch` and `serve-api`; serve-api
assertion failures become `severity:ERROR` JSON; serve-api always has an effect context (no
silent drop). Design: [m-debug-sink-structured-lines.md](m-debug-sink-structured-lines.md).

**Duration:** 1 day (attended, single session)
**Dependencies:** None — all five decisions D1–D5 frozen agent-resolvable; V1–V13 all Confirmed
**Risk Level:** Low

## Current Status Analysis

### Completed Recently
- ✅ Design doc with reproduction transcripts on v0.37.2 (b7d9f1d7c)

### Velocity
- Recent 7-day pace is several hundred LOC/day of stdlib + CLI work; this sprint is ~350 LOC
  net (+~350 new incl. tests, −~85 deleted). Comfortably one session.

### Remaining from Design Doc
- ⏳ Phase 1 shared sink: ~120 impl + ~150 test
- ⏳ Phase 2 hosts: three call sites, −85 LOC
- ⏳ Phase 3 tests + docs + changelog: ~90 test + ~25 doc

## Proposed Milestones

### Milestone M1: Shared DebugSink
**Goal:** `internal/effects/debug_sink.go` — `DebugSink`, `IsStructuredLine`, `Severity`,
`SeverityLevel`, `Flush`; single-sourced level table.
**Estimated:** 120 impl + 150 tests = 270 LOC
**Duration:** 0.4 d

**Tasks:**
- Write table test first: `{structured, unstructured, `{`-prefixed invalid JSON, leading space, JSON array}` × `{Label "", "X"}` × `{MinLevel 0, 2}` × `{Structured true, false}`; assert exact bytes
- Implement; structured assertion line via `json.Marshal` (never hand-quoted)
- `Flush` calls `d.Reset()`; nil-safe on nil `*DebugContext`

**Acceptance Criteria:**
- [x] Structured line: verbatim, no label, written to `W` not `Logf`
- [x] Unstructured line: `Logf("%s%s", labelPrefix, msg)` — byte-identical to today's decoration
- [x] `{`-prefixed invalid JSON: decorated, never dropped
- [x] Filter suppresses below `MinLevel` for structured lines only (unstructured lines have no severity → always pass, as today)
- [x] `Structured:true` assertion failure → one `json.Valid` line with `severity:"ERROR"`
- [x] `go test ./internal/effects/` green; `golangci-lint` clean

### Milestone M2: Rewire the three hosts, delete the duplicates
**Goal:** `run_helpers.go`, `server.go`, `serve_api.go` use the sink; D3/D4/D5 applied.
**Estimated:** +20 / −85 LOC
**Duration:** 0.3 d

**Tasks:**
- `run_helpers.go`: `flushDebugOutput` becomes a wrapper building `DebugSink{W: os.Stderr, MinLevel: debugLogLevel, Label: label}`; delete `extractSeverity`/`severityLevel`; keep `parseLogLevel` (delegating to the moved table if it uses it)
- `server.go`: `flushDebugOutput` builds `DebugSink{W: os.Stderr, Logf: <log.Printf with "[Debug] ">, MinLevel: s.logLevel, Structured: true}`; delete `extractServerSeverity`/`serverSeverityLevel`
- `serve_api.go`: hoist `effCtx = effects.NewEffContext(nil)` out of the flag conditional; everything else stays gated
- `make build && make quick-install`; re-run the design-doc fixture against the new binary for all three hosts

**Acceptance Criteria:**
- [x] `grep -rn 'extractServerSeverity\|serverSeverityLevel\|func extractSeverity\|func severityLevel' cmd internal` empty
- [x] Fixture via `serve-api --caps FS,Env`: two `json.Valid` `ERROR` lines, no timestamp prefix
- [x] Fixture via `serve-api` (no caps): same two lines (D5)
- [x] Fixture via `run --batch … X`: JSON verbatim, no `[X] `; `[X] [ASSERT FAIL] …` still labelled
- [x] `go build ./... && go vet ./...` clean

### Milestone M3: Tests, docs, changelog
**Goal:** Pin the fixed behaviour where the old behaviour was pinned; document the contract.
**Estimated:** 90 test + 25 doc + changelog = ~120 LOC
**Duration:** 0.3 d

**Tasks:**
- `cmd/ailang/main_run_batch_debug_test.go`: retarget `TestBatchDebugOutput_SeverityFilterApplies` — ERROR line present AND exactly the JSON object on its own line (positive control kept), DEBUG line absent, `[ONLY] ` NOT on the JSON line
- New `internal/apiserver/debug_sink_test.go`: in-process server (existing `Config{EffCtx:…}` pattern), captured stderr, `@route` module logging JSON + failing check → two `json.Valid` `ERROR` lines
- New CLI test: `serve-api` without `--caps` still flushes (the D5 regression test) — or an in-process `serve_api` config test if spawning a server in `cmd/ailang` tests is impractical
- Mutation check: revert `server.go` only → apiserver test fails on the prefix; revert D5 only → no-caps test fails
- `docs/docs/guides/serve-api.md`: "Structured logging" section (one JSON object per line, `severity` lifted; `Debug.check` failures are `ERROR`)
- `CHANGELOG.md` Fixed entry with the exact before/after lines
- `make test`, `make verify-examples`

**Acceptance Criteria:**
- [x] Both mutation checks fail as predicted, then pass restored
- [x] `make test` green; `make verify-examples` green; `make lint` clean
- [x] serve-api guide + CHANGELOG updated
- [x] Design doc moved to `implemented/v0_37_3/` with implementation report

## Success Metrics
- Test coverage: sink 100% branch-covered by the table test
- Documentation: `docs/docs/guides/serve-api.md`, `CHANGELOG.md`
- All tests passing: ✅  All linting passing: ✅

## Dependencies
- None

## Open Questions
- None — D1–D5 frozen in the design doc.
