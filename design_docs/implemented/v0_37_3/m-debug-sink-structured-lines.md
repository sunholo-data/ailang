# M-DEBUG-SINK-STRUCTURED-LINES: Structured Debug.log lines survive every host sink

**Status:** Implemented (2026-09-11)
**Target:** v0.37.3
**Priority:** P1 (production observability — `severity>=ERROR` returns nothing on Cloud Run)
**Estimated:** 1 day
**Dependencies:** None
**Created:** 2026-09-11
**Quorum trigger:** #4 fires nominally (premise about Cloud Run's log parser, an external system). The premise is not theorised — it is the reporter's *observed* production behaviour (`textPayload`, severity `DEFAULT`) and Google's documented contract (see References). Recommended: one `ailang design-review --reviewer gpt5-6-sol` pass, not a full quorum.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Host-side sink formatting only; AILANG evaluation order and values unchanged |
| A2: Replayability | +1 | A Debug line that is a JSON object is reproduced byte-for-byte on stderr — the trace a program emitted is the trace the log store gets |
| A3: Effect Legibility | +1 | Removes a host-side silent drop (serve-api without `--caps` discards every Debug line today); the effect's output becomes legible in every host |
| A4: Explicit Authority | 0 | Debug remains a ghost effect; no capability changes |
| A5: Bounded Verification | 0 | No type-system impact |
| A6: Safe Concurrency | 0 | Sink runs on the request goroutine after `Call` returns, as today |
| A7: Machines First | +1 | Log stores (Cloud Logging, jq pipelines) parse the line; today the host mangles a machine-readable line into free text for a human's benefit |
| A8: Minimal Syntax | 0 | No syntax |
| A9: Cost Visibility | 0 | No cost surface |
| A10: Composability | +1 | One `DebugSink` shared by `run`, `run --batch` and `serve-api` replaces two divergent copies of the same severity filter |
| A11: Structured Failure | +1 | Assertion failures become `severity:ERROR` JSON in the server sink instead of an untyped `[Debug ASSERT FAIL]` string that no severity filter can see |
| A12: System Boundary | 0 | Same boundary (process stderr) |

**Net Score: +6** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects — one is *removed*
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): The whole point

### Decision Thresholds

| Net Score | Decision |
|-----------|----------|
| ≥ +2 | ✅ Proceed to implementation |
| 0 to +1 | ⚠️ Needs stronger justification |
| < 0 | ❌ Reject or redesign |
| Any −1 on A1/A3/A4/A7 | ❌ Automatic rejection |

## Problem Statement

A user's AILANG service on Cloud Run emits structured logs through `Debug.log`:

```ailang
Debug.log("{\"severity\":\"ERROR\",\"message\":\"…\"}")
```

That is the correct thing to write: Cloud Run parses any stdout/stderr line that is a bare JSON
object, lifts `severity` into the log entry, and stores the rest as `jsonPayload`. But the line
that actually reaches stderr is

```
2026/09/09 13:32:14 [Debug] {"severity":"ERROR","message":"…"}
```

so Cloud Logging stores it as `textPayload` at severity `DEFAULT`. A `severity>=ERROR` filter
returns nothing. Alerting on errors is impossible. The user correctly diagnosed this as "upstream
in AILANG's Debug sink, not fixable here" — and it is not: there is no flag, env var or config
that removes the prefix.

**Reproduced 2026-09-11 on v0.37.2** (fixture: a `@route` module that calls `Debug.log` with the
JSON above and `Debug.check(false, "repro-assert")`, served with `ailang serve-api --caps FS,Env`,
one `POST /api/repro/api/ping`). stderr, verbatim:

```
2026/09/11 16:48:05 [Debug] {"severity":"ERROR","message":"repro-error-line"}
2026/09/11 16:48:05 [Debug ASSERT FAIL] repro-assert at unknown
```

### This is a pattern, not a one-off

The audit (required by CLAUDE.md §3) found **four** faults in the same small surface — two host
sinks that each format Debug output their own way:

| # | Site | Fault | Effect on a JSON line |
|---|------|-------|-----------------------|
| 1 | `internal/apiserver/server.go:322` `log.Printf("[Debug] %s", …)` | Go's default logger (`LstdFlags`) prepends `YYYY/MM/DD HH:MM:SS `, then the sink adds `[Debug] ` | **Reported bug.** Unparseable → `textPayload`, `DEFAULT` |
| 2 | `internal/apiserver/server.go:326` `log.Printf("[Debug ASSERT FAIL] %s at %s", …)` | A failed `Debug.check` is never structured | Invisible to `severity>=ERROR` even after #1 is fixed |
| 3 | `cmd/ailang/run_helpers.go:849` `fmt.Fprintf(os.Stderr, "%s%s\n", prefix, …)` with `prefix = "[" + label + "] "` | `ailang run --batch` prefixes every line with `[<input>] ` | Same class as #1: `[X] {"severity":…}` is not JSON. Reproduced: `[X] {"severity":"ERROR","message":"repro-error-line"}` |
| 4 | `cmd/ailang/serve_api.go:96` — `effCtx` is only constructed when `--caps`, `--ai-model`, `--ai-stub` or `--verify-contracts` is set | Without those flags `apiserver.Config.EffCtx` is nil, `flushDebugOutput` returns at its nil-guard, and `debugLog` writes into a throwaway context the evaluator creates on demand (`debug.go:176`) | **Every Debug line is silently dropped.** Reproduced: same fixture, `serve-api` without `--caps` → `POST` returns `"pong"`, stderr contains zero Debug lines. Violates CLAUDE.md §2 (no silent fallbacks) |

Plus duplicated machinery: `extractSeverity`/`severityLevel` (`run_helpers.go:798-827`) and
`extractServerSeverity`/`serverSeverityLevel` (`server.go:332-359`) are copy-pasted. Both sinks
already **parse** the message as JSON to filter on `severity` — and then #1 and #3 destroy the
very structure they just read. The knowledge "this line is a JSON object" exists at the exact
point where the line is mangled.

`git log -S extractServerSeverity` → `b76036d92 Add --log-level flag for Debug ghost effect
output filtering` — the filter was added once, forked into two sinks, and the two have drifted
(the CLI has a `label` parameter the server does not; the server has a timestamp the CLI does
not). Patching #1 alone would be the fifth divergence.

## Goals

**Primary goal:** A Debug.log message that is a JSON object reaches the host's stderr as exactly
that JSON object, on its own line, in every host (`run`, `run --batch`, `serve-api`) — so any
JSON-line log consumer (Cloud Logging, `jq`, Loki) can lift `severity` from it.

**Success metrics:**
1. The reporter's service, unchanged, redeployed on v0.37.3: `severity>=ERROR` returns the ERROR lines.
2. `Debug.check` failures in `serve-api` appear under `severity>=ERROR`.
3. `serve-api` with no `--caps` emits Debug output (no silent drop).
4. One severity parser, one level table, one sink type; the two copies are deleted.
5. Every existing Debug-output test still passes, except the one that pins the batch-label-on-JSON behaviour, which is deleted and replaced (see Testing Strategy).

## High-Impact Decisions

| # | Decision | Options | Recommendation | Who decides | Change cost later |
|---|----------|---------|----------------|-------------|-------------------|
| D1 | What is a "structured line"? | (a) first non-space byte is `{` and `json.Valid` (b) any line the severity filter can parse (c) opt-in flag `--structured-logs` | **(a).** It is the rule Cloud Logging itself applies, needs no flag, and is exactly what both sinks already test to run the filter. (c) is a flag nobody will know to set — the reporter had to read our source to find the prefix | Agent | Low — a pure predicate in one place |
| D2 | What happens to the batch `[label] ` prefix on a structured line? | (a) omit — never modify a structured line (b) inject `"input":label` into the object (c) keep prefix | **(a).** The sink must not rewrite user JSON (b would collide with a user's own `input` key and change bytes the program chose); (c) is the bug. Programs that need per-input attribution under `--batch` put it in their own JSON — they control the object. Unstructured lines keep the label exactly as today | Agent | Low |
| D3 | Assertion-failure format in the server sink | (a) JSON `{"severity":"ERROR","message":"assertion failed: <msg>","location":"<loc>","source":"Debug.check"}` (b) keep text | **(a) in serve-api; text in the CLI.** A service's failed check *is* an error a `severity>=ERROR` filter should catch; the CLI is read by a human at a terminal and the current `[ASSERT FAIL] msg at loc` line stays | Agent | Low — one `Structured bool` on the sink |
| D4 | Timestamp on unstructured server lines | (a) keep `log.Printf` (timestamped) for non-JSON lines (b) raw stderr for all | **(a).** Cloud Run stamps its own `timestamp`; but local `serve-api` operators read stderr interleaved with the other `log.Printf` lines the server emits and lose ordering without it. Only the structured-line path bypasses the logger | Agent | Low |
| D5 | Fix #4 (nil effCtx) by always constructing the effect context in `serve-api` | (a) always `effects.NewEffContext(nil)` — grants only the ghost `Debug` cap (b) construct a Debug-only context (c) leave | **(a).** `NewEffContext` grants nothing but `Debug` (`context.go:274`); `grantCapabilities(effCtx, "")` grants nothing on empty input (Verification Log V8) | Agent | Low |

### Design Freeze

Every decision above is agent-resolvable with the stated recommendation. No human ratification
required; sprint-executor may start on this doc as written.

- [x] D1 structured-line predicate
- [x] D2 batch label omitted on structured lines
- [x] D3 assertion failures structured in serve-api only
- [x] D4 timestamp retained for unstructured server lines
- [x] D5 serve-api always has an effect context

## Solution Design

### Overview

Replace the two hand-rolled flush loops with one `effects.DebugSink` that owns the three things
both sinks currently do separately — severity filtering, formatting, and writing — and applies
one rule: **a structured line is written verbatim, unstructured lines get the host's decoration**.
Both hosts construct a sink with their own options (`Label` for batch, `Structured`+`Logger` for
the server) and call `Flush`.

### Architecture

```go
// internal/effects/debug_sink.go  (~120 LOC)

// DebugSink writes collected Debug output to a host.
type DebugSink struct {
    W        io.Writer            // raw writer for structured lines (os.Stderr)
    Logf     func(string, ...any) // decorated writer for unstructured lines; nil = fmt.Fprintf(W, …)
    MinLevel int                  // 0=DEBUG … 4=NONE; same table as --log-level
    Label    string               // batch attribution for UNSTRUCTURED lines only
    Structured bool               // emit assertion failures as JSON (serve-api)
}

// IsStructuredLine reports whether msg is a bare JSON object — the contract
// Cloud Logging, jq and friends apply. This is the single source of truth
// used by both the filter and the writer.
func IsStructuredLine(msg string) bool

// Severity returns the "severity" field of a structured line, or "".
func Severity(msg string) string
func SeverityLevel(sev string) int   // moved from cmd/ailang/run_helpers.go

// Flush collects, filters, writes and resets. Idempotent on an empty context.
func (s DebugSink) Flush(d *DebugContext)
```

Writer rule inside `Flush`:

```
for each log entry:
    if MinLevel > 0 && sev := Severity(msg); sev != "" && SeverityLevel(sev) < MinLevel: skip
    if IsStructuredLine(msg):  fmt.Fprintln(W, msg)              // verbatim, no label, no timestamp
    else:                      Logf("%s%s", labelPrefix, msg)    // exactly today's decoration
for each failed assertion:
    if Structured: fmt.Fprintln(W, `{"severity":"ERROR","message":"assertion failed: …","location":"…","source":"Debug.check"}`)  (json.Marshal, never hand-quoted)
    else:          Logf("%s[ASSERT FAIL] %s at %s", labelPrefix, msg, loc)
d.Reset()
```

Call sites become:

```go
// cmd/ailang/run_helpers.go
sink := effects.DebugSink{W: os.Stderr, MinLevel: debugLogLevel, Label: label}
sink.Flush(effCtx.Debug)

// internal/apiserver/server.go
sink := effects.DebugSink{W: os.Stderr, Logf: log.Printf, MinLevel: s.logLevel, Structured: true}
// Logf wraps the "[Debug] " prefix so unstructured lines are byte-identical to today
```

`server.go`'s `Logf` keeps the `[Debug] ` and `[Debug ASSERT FAIL]` decoration for unstructured
lines only; the CLI keeps `[ASSERT FAIL]`. Nothing a human sees today changes except that JSON
lines lose the prefix they should never have had.

### Implementation Plan

**Phase 1 — shared sink (0.4 d)**
- [ ] Create `internal/effects/debug_sink.go` with `DebugSink`, `IsStructuredLine`, `Severity`, `SeverityLevel`
- [ ] Table tests: structured/unstructured × label × MinLevel × Structured, incl. leading whitespace, `[` arrays (not objects → unstructured), invalid JSON starting with `{` (→ unstructured, decorated — never dropped)
- [ ] Move `parseLogLevel` next to the level table (keep the CLI flag parsing where it is; only the table moves)

**Phase 2 — hosts (0.3 d)**
- [ ] `cmd/ailang/run_helpers.go`: delete `severityLevel`, `extractSeverity`, the body of `flushDebugOutput`; keep the function as a 3-line wrapper so `main_run_exec.go:594` and `run_helpers.go:652` are untouched
- [ ] `internal/apiserver/server.go`: delete `extractServerSeverity`, `serverSeverityLevel`; `flushDebugOutput` builds the sink (D3, D4)
- [ ] `cmd/ailang/serve_api.go`: always construct `effCtx` (D5); the capability/AI/contract/stream setup stays inside the existing conditionals — only `effects.NewEffContext(nil)` moves out

**Phase 3 — tests + docs (0.3 d)**
- [ ] Delete `TestBatchDebugOutput_SeverityFilterApplies`'s `[ONLY] ` assertion on a JSON line; replace with an assertion that the line is **exactly** the JSON object (structured lines are unlabelled — D2). Keep the known-positive-control shape the test's comment insists on: the ERROR line must be present *and* verbatim, the DEBUG line absent
- [ ] New `internal/apiserver` test: serve a `@route` module that logs a JSON line and fails a check; assert stderr has two lines, each `json.Valid`, each with `severity == "ERROR"`, neither starting with a digit
- [ ] New `cmd/ailang` test: `serve-api` with no `--caps` still flushes Debug output (pins D5 — this is the silent-drop regression test)
- [ ] `docs/docs/guides/serve-api.md`: a "Structured logging" section — write a JSON object with `severity`, it reaches Cloud Logging as `jsonPayload`; `Debug.check` failures are `ERROR`
- [ ] CHANGELOG entry under Fixed

### Files to Modify/Create

- `internal/effects/debug_sink.go` — NEW (~120 LOC): `DebugSink`, `IsStructuredLine`, `Severity`, `SeverityLevel`
- `internal/effects/debug_sink_test.go` — NEW (~150 LOC): table tests over the writer rule
- `cmd/ailang/run_helpers.go` — −50 LOC: severity helpers and flush body replaced by the sink
- `internal/apiserver/server.go` — −35 LOC: same
- `cmd/ailang/serve_api.go` — ~5 LOC: unconditional `NewEffContext`
- `cmd/ailang/main_run_batch_debug_test.go` — ~10 LOC: retarget the JSON-line assertion (D2)
- `internal/apiserver/debug_sink_test.go` — NEW (~80 LOC): end-to-end structured stderr
- `docs/docs/guides/serve-api.md` — +25 LOC: structured logging section
- `CHANGELOG.md` — Fixed entry

## Examples

### Example 1: The reporter's service (serve-api on Cloud Run)

```ailang
-- @route POST /order
export func order(id: string) -> string =
  let _ = Debug.log("{\"severity\":\"ERROR\",\"message\":\"payment declined\",\"order\":\"" ++ id ++ "\"}") in
  "declined"
```

Before (stderr → Cloud Logging `textPayload`, severity `DEFAULT`):
```
2026/09/09 13:32:14 [Debug] {"severity":"ERROR","message":"payment declined","order":"42"}
```

After (stderr → `jsonPayload`, severity `ERROR`; `severity>=ERROR` finds it):
```
{"severity":"ERROR","message":"payment declined","order":"42"}
```

### Example 2: Batch run with mixed lines

```
$ ailang run --batch --quiet --entry main prog.ail A B
[A] starting            ← unstructured: label kept, exactly as today
{"severity":"INFO","message":"done","input":"A"}   ← structured: verbatim; attribution is the program's job
[B] starting
{"severity":"INFO","message":"done","input":"B"}
```

### Example 3: Failed check in serve-api

Before: `2026/09/11 16:48:05 [Debug ASSERT FAIL] repro-assert at unknown`
After: `{"severity":"ERROR","message":"assertion failed: repro-assert","location":"unknown","source":"Debug.check"}`

## Success Criteria

- [ ] Fixture in Problem Statement, served on v0.37.3 with `--caps FS,Env`: both stderr lines are `json.Valid` with `severity:"ERROR"`
- [ ] Same fixture, served with **no** `--caps`: the two lines are still emitted (D5)
- [ ] `ailang run --batch` of a JSON line: emitted verbatim, no `[label] ` (D2); an unstructured line in the same batch still carries `[label] `
- [ ] `--log-level warn` still suppresses a `severity:"DEBUG"` structured line in all three hosts
- [ ] An invalid-JSON line beginning with `{` is emitted decorated, never dropped
- [ ] `grep -rn 'extractServerSeverity\|serverSeverityLevel\|func extractSeverity\|func severityLevel' cmd internal` is empty
- [ ] `make test` green; `make verify-examples` green
- [ ] serve-api guide + CHANGELOG updated

## Testing Strategy

- **Unit (sink):** table over `{structured, unstructured, invalid-JSON-with-brace, leading-space, array}` × `{Label "", "X"}` × `{MinLevel 0, 2}` × `{Structured true,false}`. Assert exact bytes.
- **Integration (serve-api):** in-process `apiserver.New` with a captured stderr writer — assert two `json.Valid` lines. Existing pattern: `internal/apiserver/*_test.go` construct servers with `Config{EffCtx: …}`.
- **Integration (CLI):** extend `cmd/ailang/main_run_batch_debug_test.go` under its existing `runDebugFixture` helper. The existing test's comment (lines 118-127) documents why an absence-only assertion is worthless; the retargeted assertion keeps the positive control.
- **Regression (silent drop):** the no-`--caps` serve-api test is the one that would have caught #4 at `b76036d92`.
- **Mutation check before commit:** revert the `server.go` change alone → the apiserver integration test must fail on the timestamp prefix. Revert D5 alone → the no-caps test must fail.

## Deferred Decisions

The executing agent has latitude on:
- Exact field names in the structured assertion object beyond `severity`/`message` (`location`, `source` are recommendations)
- Whether `Logf` is a func field or a small interface
- Whether `parseLogLevel` moves into `internal/effects` or stays in `cmd/ailang` (only the level *table* must be single-sourced)
- Test file placement inside `internal/apiserver`

## Non-Goals

- **Not** a logging library. AILANG has no `std/log`; users build the JSON string themselves (or via a package). This doc only guarantees the host does not damage it.
- **Not** touching the evaluator, `DebugContext`, or the `Debug` effect ops — `internal/effects/debug.go` is unchanged.
- **Not** fixing `location: "unknown"` (the compiler's location injection is absent in these fixtures — `debug.go:169` fallback). Observed in the repro; separate issue.
- **Not** re-routing the server's other `log.Printf` lines (startup banner, `[apiserver] zero-padded args …`). Those are human-facing and correctly `DEFAULT`.
- **Not** adding a `--structured-logs` flag (D1c rejected).

## Timeline

| Day | Work |
|-----|------|
| 0.0–0.4 | Phase 1: sink + table tests |
| 0.4–0.7 | Phase 2: rewire three call sites, delete duplicates |
| 0.7–1.0 | Phase 3: integration tests, mutation check, docs, changelog |

## Risks & Mitigations

| Risk | Likelihood | Mitigation |
|------|------------|------------|
| A downstream consumer greps for `[Debug] {` | Low — the reporter is the only known serve-api structured-log user and is asking for the change | CHANGELOG entry names the exact before/after lines |
| D5 changes capability semantics for no-caps serve-api | Low — `NewEffContext` grants only `Debug` (`context.go:274`); `grantCapabilities(ctx, "")` grants nothing (V8) | The AI/contract/stream setup stays inside its existing conditionals |
| `IsStructuredLine` on every log line costs a `json.Valid` | Negligible — both sinks already `json.Unmarshal` every line for the filter; this *removes* one parse when the filter is off | — |
| A program emits multi-line pretty-printed JSON | Not structured by D1 (Cloud Logging would not parse it either); emitted decorated, unchanged from today | Document: one object per line |

## Related Documents

Neural search (2026-09-11) returned nothing above 0.44 in either `implemented/` or `planned/` —
no prior doc on the Debug sink or host log formatting. Nearest:
- [m-diagnostic-coverage](../../implemented/v0_29_0/m-diagnostic-coverage.md) (0.42) — compiler diagnostics, not runtime logs; distinct
- [m-prompt-footguns-to-diagnostics](../../implemented/v0_30_0/m-prompt-footguns-to-diagnostics.md) (0.43) — distinct
- `b76036d92 Add --log-level flag for Debug ghost effect output filtering` — the commit that forked the two sinks; no design doc

## Verification Log

| # | Claim | Method | Result |
|---|-------|--------|--------|
| V1 | Server prefixes Debug lines with timestamp + `[Debug] ` | `serve-api --caps FS,Env` on v0.37.2, POST the fixture route, read stderr | **Confirmed** — `2026/09/11 16:48:05 [Debug] {"severity":"ERROR",…}` |
| V2 | The timestamp comes from Go's default logger, not our code | `grep -rn 'log.SetFlags' cmd/ailang internal/apiserver` | **Confirmed** — empty; `log.Printf` uses `LstdFlags` (`YYYY/MM/DD HH:MM:SS `) |
| V3 | Only two Debug sinks exist | `grep -rn 'Debug.Collect()' internal cmd` | **Confirmed** — `server.go:314`, `run_helpers.go:840`; only two callers |
| V4 | Batch CLI prefixes structured lines | `ailang run --relax-modules --quiet --entry ping --batch api.ail X` | **Confirmed** — `[X] {"severity":"ERROR","message":"repro-error-line"}` |
| V5 | Single-file CLI emits structured lines verbatim (so `run` is already correct) | same fixture without `--batch` | **Confirmed** — `{"severity":"ERROR","message":"repro-error-line"}` on its own line |
| V6 | serve-api without `--caps` silently drops Debug output | same fixture, `serve-api --port … .` (no caps), POST → `"pong"`, `grep Debug server.log` | **Confirmed** — zero Debug lines. Mechanism read at `serve_api.go:96` (effCtx nil unless flags), `server.go:311` (nil-guard returns), `debug.go:176` (on-demand throwaway context) |
| V7 | The severity filter is duplicated | read `run_helpers.go:798-827` and `server.go:332-359` | **Confirmed** — identical tables; `git log -S extractServerSeverity` → one commit `b76036d92` |
| V8 | `grantCapabilities(ctx, "")` grants nothing (D5 safety) | read `cmd/ailang/run_helpers.go:147-171` | **Confirmed** — empty names are `continue`d, so no `Grant`; the only side effect is `attachCloudSecretApprover(effCtx)`, which the doc's comment says is a no-op for local runs. In cloud mode it attaches an approver that only matters if `secret()` is called — impossible without the `Secret` cap, which no-caps does not grant |
| V9 | `NewEffContext` grants only `Debug` | read `internal/effects/context.go:265-277` | **Confirmed** — sole `Grant` call is `NewCapability("Debug")` |
| V10 | No existing `DebugSink` / shared writer helper (negative existence) | `grep -rn 'DebugSink\|WriteDebug\|debugSink' internal cmd` at authoring | **Confirmed** — empty |
| V11 | Only one test pins current Debug output bytes | `grep -rn 'flushDebugOutput\|extractSeverity\|extractServerSeverity\|severityLevel' --include='*_test.go'` | **Confirmed** — only `cmd/ailang/main_run_batch_debug_test.go`; read: `TestBatchDebugOutput_SeverityFilterApplies` asserts `[ONLY] ` on a JSON line (must change under D2); `…FailingItemFlushesAndBatchContinues` asserts `[FIRST_FAIL] DEBUG-FIRST_FAIL` on an *unstructured* line (unchanged) |
| V12 | The fixture is valid AILANG | `ailang check repro/api.ail` | **Confirmed** — `✓ No errors found!` |
| V13 | Cloud Run parses bare JSON-object lines and lifts `severity` | External — Google Cloud Logging structured-logging contract (References) + the reporter's observation that the *prefixed* form lands as `textPayload`/`DEFAULT` | **Confirmed by observation on the failing side**; the passing side is vendor-documented, not measured here (quorum trigger #4) |

## References

- Google Cloud, *Structured logging* — "If your log entry is a serialized JSON object, Logging parses it as `jsonPayload` and uses the `severity` field": https://cloud.google.com/logging/docs/structured-logging
- Go `log` package default flags: `LstdFlags = Ldate | Ltime`
- Existing Debug effect design: [docs/docs/guides/go-interop.md](../../../docs/docs/guides/go-interop.md) §"Debug ghost effect" (write-only from AILANG; host collects)
- Fixture used for every row in the Verification Log:
  ```ailang
  module repro/api
  import std/debug as Debug
  -- @route GET /ping
  export func ping() -> string =
    let _ = Debug.log("{\"severity\":\"ERROR\",\"message\":\"repro-error-line\"}") in
    let _ = Debug.check(false, "repro-assert") in
    "pong"
  ```

## Future Work

- `location: "unknown"` — compiler location injection for `Debug.log`/`Debug.check` is absent in these fixtures; once fixed, the structured assertion line carries a real `file.ail:NN`.
- A `std/log` package that builds the severity object so users stop hand-escaping JSON strings (the escaping in Example 1 is the kind of thing a model gets wrong under eval).
- If a second structured-line consumer appears (Loki, Datadog), consider a `--log-format` flag; not warranted by one contract that all of them already share.

---

## Implementation Report (2026-09-11)

**Shipped in:** v0.37.3 (unreleased at time of writing) — commits `eb8674487` (M1), `ec4497df7` (M2+M3).

### What was built
Exactly the design: `internal/effects/debug_sink.go` (`DebugSink`, `IsStructuredLine`, `Severity`,
`SeverityLevel`, `Flush`) and all three hosts routed through it. D1–D5 applied as recommended.

### Deviations from plan
- **None in behaviour.** One addition the table test forced: a structured line with **no**
  `severity` field must still pass every `--log-level` — the old filter had that property
  (`sev != "" && …`) and the first sink draft ranked it INFO. Caught by the table, fixed before commit.
- `run_helpers.go` was already over the 800-line gate at HEAD (856); the CLI log-level block moved
  to `cmd/ailang/run_debug_output.go` (46 lines) rather than shaving lines.
- Assertion-line field order is fixed by a struct (`severity, message, location, source`) so the
  bytes are stable across Go versions.

### Code locations
- NEW `internal/effects/debug_sink.go` (131) · `internal/effects/debug_sink_test.go` (168)
- NEW `internal/apiserver/debug_sink_test.go` (127) · `cmd/ailang/serve_api_debug_sink_test.go` (115)
- NEW `cmd/ailang/run_debug_output.go` (46) · `internal/embed/testdata/debug_structured.ail`
- MOD `cmd/ailang/run_helpers.go` (−82) · `internal/apiserver/server.go` (−35/+17) ·
  `cmd/ailang/serve_api.go` (+5/−2) · `cmd/ailang/main_run_batch_debug_test.go` (retargeted)
- DOC `docs/docs/guides/serve-api.md` §Structured Logging · `changelogs/v0.32-current.md`

### Verification
| Check | Result |
|---|---|
| Fixture, `serve-api --caps FS,Env` | two bare `json.Valid` `ERROR` lines, no timestamp ✓ |
| Fixture, `serve-api` no caps | same two lines (was: zero lines) ✓ |
| Fixture, `run --batch … X` | JSON verbatim; `[X] [ASSERT FAIL] …` still labelled ✓ |
| Mutation: old `server.go` sink | `TestServeAPI_StructuredDebugLinesReachStderrVerbatim` FAILS ✓ |
| Mutation: old flag-gated effCtx | `TestServeAPI_NoCapsStillFlushesDebugOutput` FAILS ✓ |
| Duplicate-parser grep | empty ✓ |
| `make lint` · `make verify-examples` · `make check-file-sizes` | green ✓ |
| `make test` | green except `internal/cihygiene.TestWiredGatesAreCanonical` — **pre-existing** from `868b55898` (docs-only CI lane `if:` guards), untouched by this sprint |
| V8 (`grantCapabilities(ctx, "")`) | Confirmed at authoring; behaviour unchanged |

### Known limitations
- `location` is `"unknown"` in every fixture — the compiler's location injection for
  `Debug.log`/`Debug.check` is not landing (pre-existing, listed under Future Work).
- Multi-line pretty-printed JSON is text, by design (D1); documented in the guide.
