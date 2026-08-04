# Sprint Plan — M-MCP-EXACT-TOOL-SURFACE-LANE-B

**Input design doc**: [`design_docs/planned/v1_0_0/m-mcp-exact-tool-surface-lane-b.md`](m-mcp-exact-tool-surface-lane-b.md) (785 lines)
**Milestone ID**: `M-MCP-EXACT-TOOL-SURFACE-LANE-B`
**Planned at HEAD**: `6bd8cda4231233f14fcfff2b9ce401b494b2b5ca` (branch `dev`, local == origin/dev)
**Planned at version**: v0.32.0 → target v1.0.0
**Planner**: claude-opus-5 (mission iteration 138)
**Executor**: `codex:gpt-5.6-sol` under `codex exec --sandbox workspace-write`
**Sprint worktree (LITERAL — do not improvise)**: `/Users/voightkampff/dev/sunholo-data/.wt-iter139-mcp-lane-b`
**Sprint branch**: `sprint/m-mcp-exact-tool-surface-lane-b`
**Design doc landed as**: PR #582, squash `2629ad8fa` (quorum-cleared, 2 rounds, $0.1910)

---

## 0. Planner corrections to the design doc — READ FIRST

The design doc is unusually well verified (28-row Verification Log). These are the premises I
re-measured that came back **different**, plus the hazards the doc does not cover. Each is
load-bearing on at least one milestone's acceptance criteria. Commands and observed output are in
[§8 Evidence](#8-evidence).

### 0.1 REFUTED — `make check-file-sizes` cannot see `serveapi/`. Its M3 AC is vacuous for the new package.

The gate is `for file in $(find internal cmd -name "*.go")` (`make/code-health.mk:127-131`). It never
walks a top-level directory. Positive control: that `find` returns **45** files under
`internal/apiserver/` and **0** under `runtime/`+`std/` even though **6** `.go` files exist there
[E1]. So `serveapi/serveapi.go` and `serveapi/serveapi_external_test.go` are **outside the 800-line
gate**, and the doc's Deferred Decision "every production file remains at or below 800 lines" is a
self-imposed standard the gate cannot enforce.

**Plan effect**: M3's `check-file-sizes` AC is retained but re-scoped to the `internal/apiserver/*`
files, and a separate explicit `wc -l serveapi/*.go` assertion is added. Claiming "size gate green"
as evidence about `serveapi` would be a fifth vacuous AC.

### 0.2 REFUTED — `make check-boundaries` cannot see `serveapi/` either. Two doc ACs are vacuous.

`scripts/check_boundaries.sh` iterates only `internal/<pkg>` for `pkg` in
`CORE_PKGS=(parser types eval core elaborate effects builtins lexer ast pipeline runtime link iface)`
and `DASHBOARD_PKGS=(server coordinator observatory messaging)`. `apiserver` appears in neither set
(grep count **0**; positive control `parser` = **1**) and no top-level directory is ever scanned [E2].

The gate therefore **passes identically whether or not `serveapi` imports the compiler core.** The
doc's M3 AC (`make check-boundaries ... pass`) and its Conflict Surface row "Public package layering
→ External compile fixture and `make check-boundaries`" are **not discriminating**.

**Plan effect**: M3 adds a real, two-sided import-direction assertion via `go list` (assert the
forbidden set is absent AND assert `internal/apiserver` IS present, in the same check). Running
`make check-boundaries` is kept only as a "did I break the existing gate" regression, with the
narrowing recorded.

### 0.3 REFUTED (scope-qualified) — `go test ./cmd/ailang` PASSES here; the doc's V19 sandbox result is real but does not travel.

Doc V19 reports `./cmd/ailang` panicking with `bind: operation not permitted` and labels it
UNINFORMATIVE UNDER SANDBOX. In my session it is `ok ... 19.282s`, rc=0, with **zero** occurrences of
`bind: operation not permitted`, and a positive control bound a real loopback listener at
`http://127.0.0.1:62979` [E3].

Both results are correct — different environments. The designer measured the codex sandbox; I measured
a Claude-agent session. **The consequence is the useful part: the controller CAN re-run every
loopback-dependent suite cheaply, so "controller re-runs outside sandbox" is a real, ~20s step, not a
deferred hope.**

### 0.4 NEW — `internal/apiserver` binds ZERO sockets today. Almost the whole sprint is informative inside the sandbox.

`httptest.NewServer` appears **13** times across exactly **6** files, **all** in `cmd/ailang`; there
are **0** in `internal/apiserver`, which instead uses `httptest.NewRecorder` in **12** files [E4].
And I proved the MCP SDK's stateless streamable handler works end-to-end through a bare
`httptest.NewRecorder` — GET → `405` + `Allow: POST` + `Method Not Allowed`; POST `initialize` →
`200`, `Content-Type: text/event-stream`, correctly SSE-framed `event: message\ndata: {...}` [E5].

**Plan effect**: every new test in this sprint MUST use `httptest.NewRecorder()`. Under that rule the
executor's own verdicts on M1/M2/M3 tests are **informative**, not sandbox-blind. Only `go test ./...`
breadth and `./cmd/ailang` are controller-re-run items.

### 0.5 NEW HAZARD (highest severity) — `mcp.Server.AddTool` **PANICS** on host-supplied schemas. It runs per request.

`AddTool` panics — not errors — in five cases (`mcp/server.go:273-314`). I reproduced two of them:

- nil `InputSchema` → `panic: AddTool "bad2": missing input schema`
- scalar `InputSchema` → `panic: AddTool "bad": can't marshal input schema to a JSON object: json: cannot unmarshal "\"scalar\"" into Go value of type map[string]interface {}` [E6]

Also panics on `type != "object"`, on an unmarshalable `OutputSchema`, and on invalid param-header
annotations. Meanwhile `validateToolName` only **logs** (`s.opts.Logger.Error`), so a bad *name*
registers silently.

On the embedded path these values come from **arbitrary host code**, and registration happens **inside
an HTTP handler goroutine**. A host that omits `InputSchema` — which the doc's `ToolDescriptor` permits,
since it is an ordinary optional field and only the doc's `submit_feedback` example bothers to set one —
crashes the handler. `net/http` recovers per-connection and aborts, so the client sees a truncated
response, not the doc's "explicit constructor/request error".

**Plan effect**: the M1 gateway MUST reject-or-normalize **before** any `AddTool`:
nil `InputSchema` → error (fail loudly; do NOT silently default to `{"type":"object"}` — a silent
default is exactly the class CLAUDE.md §2 forbids for data-integrity paths); non-object → error;
name → `validateMCPName` (AILANG's stricter `^[a-zA-Z0-9_-]{1,64}$`, not the SDK's 128-char logger).
The doc's M1 AC only tests "a scalar input schema"; **a nil-schema case is added**, and M2 gains a
panic-safety AC.

### 0.6 NEW — the doc contradicts itself on WHERE resolution happens, and only one of the two options can emit the frozen envelope.

Two statements cannot both hold:

- Solution Design/Overview: *"The embedded factory resolves the request, obtains descriptors, and
  returns a fresh SDK server"* — i.e. inside `getServer(req)`.
- Bounded Host Callbacks: *"The MCP timeout envelope is emitted before handing control to the SDK when
  resolution/surface construction fails"* — i.e. in an outer wrapper.

The second is the only implementable one. I measured that a `getServer` returning `nil` yields
**HTTP 400, `Content-Type: text/plain; charset=utf-8`, body `no server available`** [E5] — the SDK's
only failure channel from that hook. It is structurally incapable of producing the doc's frozen
`-32603` / `"host callback timed out"` / **HTTP 200** / `application/json` envelope, and it cannot echo
the JSON-RPC request id.

**Plan effect — this is the single biggest executor-rework risk, so the plan pins the shape**:
`MCPHandler()` returns a wrapper `http.Handler` that (1) bounded-reads and restores the body,
(2) best-effort decodes `id`, (3) runs `ResolveSession` → `Tools` → gateway, (4) on any failure writes
the frozen envelope **itself** and never touches the SDK, (5) on success stores the
`AuthorizedSurface`+session in the request context and delegates to a **pre-built** SDK
`StreamableHTTPHandler` whose `getServer(req)` only reads them back out. Note the wrapper's body read
happens **before** the SDK installs `http.MaxBytesReader` (`streamable.go:344-345`), so the wrapper
must impose its own cap — `mcp.DefaultMaxRequestBodyBytes` is `4 << 20`; reuse that constant.

Consequence to accept knowingly: the timeout envelope's `application/json` differs from the SDK
success path's `text/event-stream`. That is what the doc froze; it is recorded as a risk, not changed.

### 0.7 NEW — `@nomcp` is already a second, MCP-only filter. The invariant M3 is told to restate is false as written.

`.claude/rules/api-server.md:31` says `isExposed()` "is the single filtering point... No additional
filtering logic should be added in individual consumers." But `internal/apiserver/mcp.go:94-95` does
`if export.IsNoMCP { continue }` — a genuinely independent, MCP-only filter, deliberately so
(`server.go:146`: "hide from the MCP tool surface only (HTTP/OpenAPI/A2A unaffected)") [E7].

If the executor takes M3's restated invariant literally and folds `@nomcp` into the protocol-neutral
gateway, `@nomcp` exports vanish from A2A too — a silent behavior regression the doc explicitly says
is out of scope.

**Plan effect**: the M3 rule rewrite must name `@nomcp` as the one sanctioned **protocol-scoped,
annotation-declared** narrowing, distinguished from **authority** filtering (which must stay in the
gateway). Plus: the rule file's frontmatter `paths:` is `internal/apiserver/**` and
`cmd/ailang/serve_api.go` — it must gain `serveapi/**`, or the newly-written invariant never loads for
an agent editing the new package.

### 0.8 CONFIRMED (spot-checks of my brief and of the doc)

| Premise | Result |
|---|---|
| `internal/apiserver/server.go` = 764 lines; gate at 800 | CONFIRMED, 36 lines headroom [E1] |
| `serveapi/` does not exist (control: `runtime/` does) | CONFIRMED [E8] |
| module path `github.com/sunholo-data/ailang`; public import is `.../serveapi` | CONFIRMED [E8] |
| `Config` has 15 fields (doc's refutation of "16") | CONFIRMED [E9] |
| `mcp.go:303-306` discards the per-request `*http.Request`; `Stateless: true` already set | CONFIRMED [E10] |
| `mcp.Tool` has both `InputSchema any` and `OutputSchema any`, both accept `json.RawMessage` | CONFIRMED — the doc's `OutputSchema` projection is feasible [E6] |
| A2A error wire format = HTTP 200 + JSON-RPC (doc V27) | CONFIRMED: `a2a.go:304` `w.WriteHeader(http.StatusOK) // JSON-RPC always returns 200.`, codes `-32600/-32601/-32602`, no `StatusInternalServerError` on the task path [E11] |
| `-32603` unused in `a2a.go` (doc V28) | CONFIRMED: count 0, control `-32602` = 6 [E11] |
| `make check-boundaries` / `make check-file-sizes` exist (1 each) | CONFIRMED [E1][E2] |
| `.claude/rules/api-server.md` exists, 46 lines | CONFIRMED [E12] |
| `docs/docs/guides/serve-api.md` exists, **1384** lines (doc plans +70) | CONFIRMED [E12] |
| `TestIsTempPath` (internal/loader), `TestSolve_HardTimeout_FakeSolverIgnoringT` (internal/smt) exist → `/tmp` worktrees fail for LOCATION | CONFIRMED [E13] |
| 6 production `isExposed` call sites: `handler.go:142`, `server.go:686`, `a2a.go:57`, `a2a.go:185`, `openapi.go:187`, `mcp.go:91` | CONFIRMED (28 total refs incl. tests) [E7] |
| `internal/apiserver` does NOT import `serveapi` (control: imports `internal/embed` = 1) | CONFIRMED — no cycle today; keep it that way [E14] |

### 0.9 NEW — the external-module compile fixture (M1's flagship AC) has TWO offline gotchas. Recipe proven.

The fixture is the riskiest AC under a no-network sandbox. I built it and it works, but only with two
non-obvious steps:

1. The fixture's `go.mod` **must declare `go 1.26.5`**, matching the replace target. With `go 1.24` it
   fails `go: module ... requires go >= 1.26.5 (running go 1.26.4)` — an error that looks nothing like
   an import problem and would burn executor time [E15].
2. It **must `cp <worktree>/go.sum .`**, or transitive external deps (MCP SDK, jsonschema-go, …) have
   no sums and `GOPROXY=off` cannot invent them [E16].

Verified recipe, offline, rc=0, and the denial half returns the exact string the doc's AC demands:
`use of internal package github.com/sunholo-data/ailang/internal/apiserver not allowed` [E15][E16].
The `replace` must point at the **worktree**, not the main checkout.

---

## 1. Executor environment contract

The executor is `codex:gpt-5.6-sol` under `codex exec --sandbox workspace-write`. The plan is written
for that executor, not for a Claude agent.

**No git writes.** A linked worktree's `.git` is a *file* pointing under the main checkout's
`.git/worktrees/…`, which the sandbox excludes. The executor therefore performs **no** `git add`,
`git commit`, `git branch`, or `git push`. Instead:

> **Snapshot protocol.** On finishing milestone *k*, the executor copies **every file it has created
> or modified so far in the sprint** — cumulative, full post-milestone content, not a diff — into
> `.snap/M<k>/` preserving relative paths. The **controller** reconstructs one commit per milestone
> from `.snap/M1/`, `.snap/M2/`, `.snap/M3/` in the main checkout. `.snap/` is scratch and is never
> committed.

**No sockets, no network.** Loopback binds and outbound network are denied. Therefore:

| Check | Executor verdict | Why |
|---|---|---|
| all new `internal/apiserver` + `serveapi` tests (recorder-only) | **AUTHORITATIVE** | §0.4: recorder path proven to work in-sandbox |
| the external-module compile fixture | **AUTHORITATIVE** | §0.9: offline recipe proven |
| `go test ./internal/apiserver` | **AUTHORITATIVE** | zero socket binds in that package |
| `go test ./cmd/ailang`, `go test ./...` | **UNINFORMATIVE — controller re-runs** | 13 `httptest.NewServer` sites in `cmd/ailang` |
| anything using `httptest.NewServer` | **FORBIDDEN in new tests** | would be uninformative by construction |
| `make check-boundaries`, `make check-file-sizes`, `gofmt`, `go vet` | AUTHORITATIVE (no sockets) — but see §0.1/§0.2 for what they do **not** prove | |

**Worktree**: `/Users/voightkampff/dev/sunholo-data/.wt-iter139-mcp-lane-b` — a sibling of the repo,
**never** under `/tmp` (§0.8: `/tmp`-rooted checkouts fail `TestIsTempPath` and
`TestSolve_HardTimeout_FakeSolverIgnoringT` for the *location*, producing a red CI can never
reproduce). The path is free at plan time; a `/private/tmp/wt-iter130-qwen36` worktree exists as a
live example of the anti-pattern.

**Every wait bounded.** Any poll carries a `date +%s` deadline; cap ≤ 30 min.

---

## 2. Milestone ordering — CHANGED from the design doc

### 2.1 Why the doc's ordering is not bisectable

Milestones must be independently committable, and the relevant test package must be green at every
boundary. The doc's M1 fails that test **by its own acceptance criteria**:

- M1 AC2 ("`Tools` receives pointer sentinel A for request A and sentinel B for request B") needs a
  working HTTP handler.
- M1 AC5 (the 20 ms/250 ms timeout table) says explicitly: *"The same table covers MCP POST, A2A task
  POST, and (for resolver/tools) A2A card GET."*
- M1 AC6 (bounded concurrency) asserts the overload envelope *"HTTP 200 on MCP/A2A POST; HTTP 503 on
  A2A card GET"*.

The MCP handler arrives in M2 and the A2A handler in M3. **Three of M1's six ACs are unsatisfiable at
the M1 boundary.** This is the exact class that invalidated iteration 135's patch evidence.

### 2.2 The reordering

Two changes. Implementation sequence keeps the M1→M2→M3 names; what moves is AC assignment and one
block of work.

**(a) AC re-assignment — HTTP-level ACs move to the milestone that first provides that transport.**

| Doc AC | Doc home | New home | Reason |
|---|---|---|---|
| external compile fixture + internal-denied | M1 | **M1** | unchanged |
| descriptor sort / duplicate / scalar schema errors | M1 | **M1** (+ nil-schema case, §0.5) | unchanged, strengthened |
| existing `@noexpose`/`--routes-only`/`@nomcp` tests unchanged | M1 | **M3** | moves with the refactor, see (b) |
| pointer-sentinel session identity | M1 | **M1 at the runner level** (no HTTP) **and** M2/M3 at the wire level | runner-level is *more* discriminating: it isolates the mechanism from transport |
| 20 ms/250 ms timeout table | M1 | **M1 runner-level** + **M2** (MCP POST) + **M3** (A2A POST/GET) | split by transport availability |
| bounded-concurrency / overload envelope | M1 | **M1 runner-level (starts capped, token-held-to-exit, goroutine count)** + **M2/M3** (wire envelopes) | the goroutine-accounting half needs no HTTP at all |

Net effect: nothing is dropped and nothing is weakened. The callback runner becomes a
protocol-neutral, directly unit-testable unit, which is a *stronger* test of the semaphore semantics
than driving it through two protocol stacks.

**(b) The `isExposed` → gateway generalization moves M1 → M3.**

The doc puts it in M1 bullet 4. Move it to M3, because:

1. **The embedded path never calls it.** Embedded authorization is `callerSurface([]ToolDescriptor)`.
   M1 and M2 have zero dependency on the loaded-export branch.
2. **It is the only change in the sprint that can regress standalone behavior** — 6 production call
   sites across 5 files [E7], covering HTTP dispatch, OpenAPI, A2A card, A2A task, MCP list, and the
   startup banner. Landing it last keeps the bisect boundary sharp: if standalone tests go red,
   exactly one commit is suspect.
3. Its verification (existing filter tests unchanged) sits naturally beside M3's other standalone
   compatibility ACs, so one milestone owns "did we break the CLI".

M1 still defines the `AuthorizedSurface` type and the `callerSurface(...)` constructor — that is what
M2 consumes. Only the `loadedExportSurface(...)` branch and the rewiring of the 6 existing call sites
move to M3.

### 2.3 Resulting milestones

| # | ID | Focus | LOC | Hours |
|---|---|---|---|---|
| M1 | `M1_PUBLIC_CONTRACT_AUTHORIZED_SURFACE_AND_BOUNDED_CALLBACK_RUNNER` | public `serveapi` package; `callerSurface` gateway; bounded-callback runner; offline external compile fixture | 430 | 6.5 |
| M2 | `M2_MCP_REQUEST_SCOPED_ADAPTER_AND_FROZEN_WIRE_ENVELOPES` | outer wrapper handler; per-request SDK server; panic-safe registration; frozen MCP envelopes; A/B exact surface | 480 | 8.5 |
| M3 | `M3_A2A_PROJECTION_MOUNT_EXPOSURE_GENERALIZATION_DOCS_AND_GATES` | A2A card+task adapter; `Mount`; `isExposed` generalization; rules/docs; real boundary + size assertions; race | 540 | 10.0 |
| | | **Total** | **~1450** | **~25** |

---

## 3. Milestone detail

### M1 — Public contract, authorized surface, bounded callback runner (6.5 h, ~430 LOC)

**Files**

| Path | Now | After | Note |
|---|---|---|---|
| `serveapi/serveapi.go` | — | ~200 | NEW. Public types only, stdlib types only. **Not covered by `check-file-sizes`** (§0.1) |
| `serveapi/serveapi_external_test.go` | — | ~130 | NEW. Offline fixture per §0.9 |
| `internal/apiserver/authorized_surface.go` | — | ~190 | NEW. copy/validate/sort/`Lookup`/`All` |
| `internal/apiserver/callbacks.go` | — | ~150 | NEW (doc did not enumerate this file). Deadline + semaphore + buffered-select runner |
| `internal/apiserver/authorized_surface_test.go` | — | ~180 | NEW |
| `internal/apiserver/callbacks_test.go` | — | ~200 | NEW |

`serveapi` declares `MCPHandler()`, `A2AHandler()`, `Mount()` at M1 so the compile fixture is real;
their wire behavior completes in M2/M3.

**Acceptance criteria** (→ traces to doc AC)

1. **(→ doc M1 AC1) External-module fixture, offline.** In a temp dir outside the worktree: `go.mod`
   declaring **`go 1.26.5`**, `require github.com/sunholo-data/ailang v0.0.0`,
   `replace github.com/sunholo-data/ailang => /Users/voightkampff/dev/sunholo-data/.wt-iter139-mcp-lane-b`,
   plus `cp <worktree>/go.sum .`. Then `GOFLAGS=-mod=mod GOPROXY=off go build ./...` rc=0 for a file
   that imports `github.com/sunholo-data/ailang/serveapi` and calls `serveapi.New` with sentinels,
   `MCPHandler()`, `A2AHandler()`, and `Mount(mux)`. In the **same** fixture, a `//go:build denied`
   file importing `internal/apiserver` must fail with exactly
   `use of internal package github.com/sunholo-data/ailang/internal/apiserver not allowed`.
   *Would still pass if the claim were false?* No: the two halves are opposite-signed, and the
   positive half is what proves the toolchain/go.sum setup is not silently broken. Recipe proven
   offline [E15][E16].
2. **(NEW, → §0.2) Import direction, two-sided.**
   `go list -f '{{join .Imports "\n"}}' ./serveapi` contains **zero**
   `internal/(parser|types|eval|core|elaborate|pipeline)` **and at least one**
   `internal/apiserver`. Emitting the counts side by side; if the `internal/apiserver` count is 0 the
   test `t.Fatal`s as **instrument failure** rather than passing. Symmetric check:
   `go list -f '{{join .Imports "\n"}}' ./internal/apiserver` contains zero `ailang/serveapi`
   (control: `internal/embed` present = 1) [E14]. *This replaces `make check-boundaries` as the
   layering evidence, because that gate cannot see either package.*
3. **(→ doc M1 AC3, strengthened per §0.5) Descriptor gateway.** Table over: input `[zeta, alpha]` →
   surface order `[alpha, zeta]`; duplicate `alpha` → error; scalar `InputSchema` → error; **`nil`
   `InputSchema` → error** (new; the SDK would otherwise panic `missing input schema`); `OutputSchema`
   that fails to marshal → error; name failing `^[a-zA-Z0-9_-]{1,64}$` → error. Every error is
   returned, never a panic, never a silent default. Deep-copy proof: mutate the caller's slice, the
   nested `json.RawMessage`, and `Tags`/`Examples` **after** `Tools` returns, then assert the served
   surface is byte-identical to the pre-mutation snapshot. *Would still pass if the claim were false?*
   The mutation half fails on a shallow copy; the nil/scalar halves fail if validation is skipped
   (the SDK's panic is deferred to M2, so a skipped check here shows up as a *missing error*, not a
   crash).
4. **(→ doc M1 AC5, runner half) Timeout selection and bound.** Direct unit tests on the runner, no
   HTTP: `CallbackTimeout: 20*time.Millisecond` + a callback that never returns → the runner returns
   an error matching `context.DeadlineExceeded` after ≥ 20 ms and before a 250 ms outer deadline, and
   the *next* callback in the chain is never entered (counter == 0). `CallbackTimeout: 0` → effective
   deadline is `DefaultCallbackTimeout`, asserted by reading the deadline off the context handed to
   the callback and requiring it in `(now+4s, now+6s)` — **not** by asserting `5s` equals
   `DefaultCallbackTimeout`, which would be vacuous. Negative → `New` returns an error. A fast
   callback with `CallbackTimeout: 0` succeeds. *Would still pass if the claim were false?* No: the
   observed-deadline window fails if zero is treated as "no deadline" (no deadline at all) or if the
   configured value is ignored.
5. **(→ doc M1 AC6, accounting half) Bounded concurrency.** `MaxConcurrentCallbacks: 4` + a callback
   that blocks permanently. Launch 40 runner calls. Assert (a) callback **starts** == 4 exactly —
   call 5 never enters; (b) calls 5..40 each return the overload sentinel error without entering the
   callback; (c) `runtime.NumGoroutine()` sampled after the burst does not grow with call count
   (compare the 40-call and 80-call figures; difference ≤ a small constant). Then release the blocked
   callbacks and assert capacity recovers. Control: 40 **fast** calls at `MaxConcurrentCallbacks: 4`
   produce **zero** overload errors. *Would still pass if the claim were false?* No, in three
   directions: no semaphore → starts grow; token released when the handler stops waiting instead of
   when the goroutine exits → capacity recovers and starts exceed 4; cap rejecting everything → the
   fast-call control fails.
6. **`gofmt -l` empty over changed files** (with `wc -l` beside it), `go vet ./serveapi ./internal/apiserver`
   rc=0, `go test ./internal/apiserver ./serveapi` rc=0. Narrowing recorded: this is **not**
   `go test ./...`.

**Snapshot**: `.snap/M1/` — cumulative full content of all six files above.

---

### M2 — MCP request-scoped adapter and frozen wire envelopes (8.5 h, ~480 LOC)

**Files**

| Path | Now | After | Note |
|---|---|---|---|
| `internal/apiserver/embedded_mcp.go` | — | ~260 | NEW. Wrapper handler + per-request SDK server + projection |
| `internal/apiserver/embedded_mcp_test.go` | — | ~330 | NEW (doc's single `embedded_test.go` is split at the source; see §5) |
| `internal/apiserver/mcp.go` | 386 | ~370 | Extract `mcpError` reuse + name validation; standalone semantics unchanged |
| `serveapi/serveapi.go` | ~200 | ~215 | Wire `MCPHandler()` to the adapter |

**Architecture pinned by this plan** (§0.6 — do not re-litigate): `MCPHandler()` returns a wrapper
`http.Handler`. Order: bounded-read body (cap `mcp.DefaultMaxRequestBodyBytes` = `4<<20`) → restore
via `io.NopCloser(bytes.NewReader(buf))` → best-effort decode JSON-RPC `id` → `ResolveSession` →
`Tools` → gateway → **on failure, write the frozen envelope directly and return** → on success stash
surface+session in request context, delegate to a **handler built once at `New`** whose
`getServer(req)` only reads context and builds the per-request `*mcp.Server`.
`StreamableHTTPOptions{Stateless: true}` is mandatory and frozen.

**Acceptance criteria**

1. **(→ doc M2 AC1) Exact A/B surfaces.** `tools/list` as A is exactly `[alpha_only, shared]`; as B
   exactly `[beta_only, shared]`. Assert set equality both ways (not `Contains`) and explicitly assert
   the *foreign* sentinel is absent. Empty result set → `t.Fatal("instrument failure")`.
2. **(→ doc M2 AC2) Dispatch authorization + session identity.** `alpha_only` as A: invocation counter
   == 1, `Invoke` received the `sessionA` **pointer** (`==` on the sentinel pointer, plus a distinct
   `sessionB` pointer in the same test so a stuck-value bug is caught), arguments byte-equal
   `{"nonce":"A-137"}`. `alpha_only` as B: MCP unknown-tool, counter **stays 0**.
3. **(→ doc M2 AC3) `submit_feedback` is not ambient.** Empty descriptor set → `tools/list` returns
   **0** tools and does not contain `submit_feedback`. Same test then supplies a `submit_feedback`
   descriptor and asserts it appears and dispatches through `Invoker`. The zero-result half carries
   the non-empty half as its known-positive control in the same test function.
4. **(→ doc M2 AC5) Stateless transport.** Recorder GET to `/mcp/` → **405** with `Allow: POST` and
   body `Method Not Allowed`; correctly headed POST (`Content-Type: application/json`,
   `Accept: application/json, text/event-stream`) reaches the request-local server and its sentinel
   tool. Additionally assert `Content-Type: text/event-stream` and an `event: message` line on the
   success path. All via `httptest.NewRecorder` — verified achievable [E5]. *Would still pass if
   stateful?* No: stateful mode routes GET into `serveStateful`, which does not return 405.
5. **(NEW, → §0.5) Panic safety at per-request registration.** A host returning a descriptor whose
   `InputSchema` is nil, and one whose `InputSchema` is `"scalar"`, must each produce the frozen
   protocol error envelope with the invocation counter at 0 — **and the test must assert the process
   did not panic** (the handler returns a decodable body, and a subsequent good request on the same
   handler still succeeds, proving no poisoned state). Control: the same handler with a valid
   descriptor returns 200. *Would still pass if validation were removed?* No: `AddTool` panics
   (reproduced, [E6]), `net/http` aborts the connection, the recorder body is not a decodable
   envelope, and the follow-up request assertion catches poisoned state.
6. **(→ doc M1 AC5, MCP half) Frozen MCP timeout envelope.** `CallbackTimeout: 20*time.Millisecond`;
   three cases (non-returning `ResolveSession`, `Tools`, `Invoke`). Each: HTTP **200**,
   `Content-Type: application/json`, JSON-RPC `error.code == -32603`,
   `error.message == "host callback timed out"`, `id` echoed when decodable and `null` when not
   (two sub-cases), elapsed ≥ 20 ms and < 250 ms, next-callback counter == 0. Plus
   `context.Canceled` → `"host callback canceled"`. Plus resolver authorization error → 401/403 per
   the typed contract. Explicitly assert the response is **not** 500 and **not** 504.
7. **(→ doc M1 AC6, MCP half) Overload envelope.** `MaxConcurrentCallbacks: 4`, permanently blocking
   callback, 40 POSTs: excess requests return HTTP **200**, `-32603`,
   `"host callback capacity exceeded"`. Control: 40 fast POSTs at the same `N` → zero overload
   envelopes.
8. **(→ doc M2 AC4) Standalone MCP regressions unchanged.** `go test ./internal/apiserver` rc=0 with
   `protocol_test.go` (14 tests), `mcp_schema_test.go` (9), `feedback_tool_surface_test.go` (4),
   `nomcp_test.go` (5), `filtering_test.go` (8), `a2a_dispatch_gate_test.go` (2) all still executing —
   proven by `-run` name listing, not by an aggregate `ok`. No hand-written SSE encoder is introduced:
   `grep -c 'event: message' internal/apiserver/*.go` (non-test) == 0, control: the SDK's own
   streamable.go > 0.

**Snapshot**: `.snap/M2/` — cumulative (M1 files + M2 files).

---

### M3 — A2A projection, Mount, exposure generalization, docs, gates (10.0 h, ~540 LOC)

**Files**

| Path | Now | After | Note |
|---|---|---|---|
| `internal/apiserver/embedded_a2a.go` | — | ~240 | NEW |
| `internal/apiserver/embedded_a2a_test.go` | — | ~290 | NEW |
| `internal/apiserver/authorized_surface.go` | ~190 | ~250 | + `loadedExportSurface(...)` branch |
| `internal/apiserver/routes.go` | 390 | ~410 | `isExposed` → gateway delegation |
| `internal/apiserver/a2a.go` | 318 | ~300 | Extract card/task codecs |
| `internal/apiserver/handler.go` | 380 | ~382 | call-site rewire |
| `internal/apiserver/openapi.go` | 331 | ~333 | call-site rewire |
| `internal/apiserver/server.go` | **764** | **764** | **DO NOT TOUCH** except the one `isExposed` call at :686 (net 0 lines). 36 lines of headroom |
| `serveapi/serveapi.go` | ~215 | ~235 | `A2AHandler()`, `Mount()` |
| `.claude/rules/api-server.md` | 46 | ~62 | invariant rewrite **+ `serveapi/**` in frontmatter `paths:`** (§0.7) |
| `docs/docs/guides/serve-api.md` | 1384 | ~1454 | embedding example + ownership boundary |
| `CHANGELOG.md` | — | +~15 | required by coding-standards |

**Acceptance criteria**

1. **(→ doc M3 AC1) A2A card exact sets.** Same A/B descriptors as M2. Card as A exposes exactly
   `[alpha_only, shared]`; as B exactly `[beta_only, shared]`; set equality both ways plus explicit
   foreign-sentinel absence. Descriptions, `tags`, and `examples` byte-equal the source descriptors,
   using **non-default sentinel values** for each (empty tags would make the assertion vacuous).
   `id` and `name` both equal the descriptor name.
2. **(→ doc M3 AC2) A2A dispatch authorization.** `beta_only` as B → counter 1 with the `sessionB`
   pointer; the same send as A → JSON-RPC `-32602` invalid-params, counter **0**.
3. **(→ doc M3 AC3) `Mount`, recorder only.** One `http.ServeMux`, `api.Mount(mux)`, then via
   `httptest.NewRecorder`: MCP `initialize` at `/mcp/` returns 200 + `text/event-stream`; a card at
   `/.well-known/agent.json`; A2A JSON-RPC at `/a2a/`. **No socket is bound** — assert by construction
   (the test file must contain zero `httptest.NewServer`; `grep -c` == 0 with the SDK file as the
   known-positive control that the pattern matches at all). Also assert `/mcp` (no trailing slash)
   and `/mcp/deep/path` behave per `StripPrefix`.
4. **(→ doc M1 AC5/AC6, A2A half) Frozen A2A envelopes.** Task POST timeout → HTTP **200**, `-32603`,
   `"host callback timed out"`. Card **GET** timeout → HTTP **504** with the same message in a JSON
   body. Card GET overload → HTTP **503**, `"host callback capacity exceeded"`. Task POST overload →
   HTTP 200 + `-32603`. Canceled → `"host callback canceled"`, POST 200 / card GET **500**. Each case
   asserts the code **and** that the *other* protocol's mapping was not used (e.g. card GET timeout is
   504, explicitly not 200 and not 503).
5. **(→ doc M1 AC4, moved here) Exposure generalization is not vacuous.** After rewiring all 6
   production `isExposed` call sites [E7]: `filtering_test.go` (8 tests), `nomcp_test.go` (5),
   `mcp_schema_test.go` (9) pass **unchanged** — no edits to those files; prove it with
   `git diff --name-only` (via `.snap` comparison) showing those three paths absent. Then a
   *discriminating* addition: a table asserting `@noexpose` hides, `--routes-only` hides a
   non-`@route` export, and `@nomcp` hides from MCP **while remaining present in the A2A card and
   OpenAPI** — the last clause is what catches the §0.7 mistake of folding `@nomcp` into the neutral
   gateway. *Would still pass if the refactor made the gate always-true?* No: the hide-cases fail.
6. **(NEW, → §0.7) Rule file is actually reachable.** `.claude/rules/api-server.md` frontmatter
   `paths:` includes `serveapi/**` (assert by grep, with `internal/apiserver/**` as the
   known-positive control in the same command), and the Endpoint Filtering section names `@nomcp` as
   the single sanctioned protocol-scoped narrowing, distinct from authority filtering.
7. **(→ doc M3 AC4) Standalone CLI compatibility.** Default server MCP surface includes **both**
   `status` and `submit_feedback`; with `NoFeedbackTool: true` it includes `status` and excludes
   `submit_feedback`. Both directions in one test (`status` is the known-positive control that keeps
   the exclusion half from passing on an empty surface).
8. **(→ doc Testing Strategy) Race + interleave.** `go test -race -run 'Embedded' ./internal/apiserver`
   rc=0 with 100 interleaved A/B discovery requests; assert no foreign sentinel ever appears and that
   the observed request count == 100 (a loop that silently ran 0 times must fail).
9. **(→ doc M3 AC5, re-scoped per §0.1/§0.2) Gates, with narrowings recorded.**
   - `make check-file-sizes` rc=0 **and** an explicit `wc -l internal/apiserver/*.go serveapi/*.go`
     table showing every file ≤ 800, because the gate does not walk `serveapi/` [E1].
   - `make check-boundaries` rc=0 **as a regression only**; the real layering evidence is M1 AC2.
   - `gofmt -l` empty over changed files; `go vet` rc=0; `go test ./internal/apiserver ./serveapi` rc=0.
   - `go test ./...` and `go test ./cmd/ailang`: **executor must NOT report a verdict** — flagged for
     controller re-run outside the sandbox (§0.3 shows this costs the controller ~20 s).
10. Design doc moved to `design_docs/implemented/v1_0_0/`; `CHANGELOG.md` entry added.

**Snapshot**: `.snap/M3/` — cumulative (M1 + M2 + M3 files).

---

## 4. Day-by-day

| Day | Hours | Work |
|---|---|---|
| **1 AM** | 3.5 | M1: `serveapi` public types + `New` validation; `authorized_surface.go` `callerSurface` with the §0.5 validation set |
| **1 PM** | 3.0 | M1: `callbacks.go` runner (deadline + semaphore + buffered select); runner unit tests (AC4, AC5); offline external fixture (AC1, AC2) → **`.snap/M1/`** |
| **2 AM** | 4.0 | M2: wrapper handler + body tee + id decode + frozen envelope writer; per-request SDK server behind `getServer`; panic-safe registration |
| **2 PM** | 4.5 | M2: A/B exact-surface, dispatch-authorization, feedback-absence, stateless-405, panic-safety, timeout and overload envelope tests → **`.snap/M2/`** |
| **3 AM** | 4.0 | M3: A2A card + task adapter; skill projection; `Mount`; A2A envelope tests |
| **3 PM** | 3.5 | M3: `isExposed` → gateway generalization across 6 call sites; the `@nomcp`-survives-in-A2A test; race/interleave |
| **4 AM** | 2.5 | M3: rules file (incl. frontmatter `paths:`), `serve-api.md` embedding guide, CHANGELOG, gate sweep, doc move → **`.snap/M3/`** |

**~25 h over 3.5 working days** of executor time, excluding controller commit reconstruction and the
out-of-sandbox `go test ./...` re-runs.

---

## 5. Verdict on the design doc's ~17 h / 3-milestone budget

**The 17 h estimate is wrong — low by roughly 1.5×. I put it at ~25 h.** Reasons, in order of weight:

1. **The doc's own AC set is the cost driver, not the LOC.** Enumerated: an offline cross-module
   compile fixture; a 3×3 timeout table; a goroutine-accounting concurrency test; frozen wire
   envelopes across 5 error classes × 3 entry points (that is 15 assertions with explicit
   *negative* clauses); a 100-request interleaved race test; and two-sided exact-set assertions in
   both protocols. That is a test-heavy sprint where test LOC ≈ 70 % of implementation LOC, not the
   usual 30-50 %.
2. **The §0.6 architecture ambiguity is a real fork.** An executor that follows the Overview
   ("resolve inside the factory") writes M2, discovers HTTP 400 `no server available` cannot carry
   `-32603`, and rewrites the handler. That is a 2-4 h detour this plan removes by pinning the shape,
   but the pinning itself was not in the 17 h budget.
3. **Calibration against the same subsystem.** Lane A (`aa02f0d9f`) landed **1023 insertions**, of
   which ~490 was its sprint plan — so ~530 LOC of code+tests+docs for a *far* smaller scope (one
   flag, one conditional, one dispatch check). Lane B is ~1450 LOC across a new public package, a new
   protocol-neutral authorization type, a concurrency-bounded callback runtime, and two protocol
   adapters. A ~2.7× LOC scope at 3× the conceptual novelty does not fit the same 3-day box.
4. **Recent comparable sprints in this repo**: `M-RECORDED-STREAM-API-S1` 730 LOC / 3.75 d;
   `M-EVAL-STANDARD-CONFIDENCE-GATING` 460 LOC / 4 d. At those rates 1450 LOC is 4+ days.

**Recommendation: accept ~25 h / 3.5 days and keep all three milestones**, rather than de-scoping.
The doc's Non-Goals are already tight and each milestone is independently useful. If the mission needs
a hard 3-day box, the only clean cut is **defer M3's `isExposed` generalization to a follow-up
sprint** — it is the one block with no embedded-path dependency (§2.2b), worth ~3.5 h, and its absence
leaves the rule file temporarily saying something the code does not do. I do **not** recommend cutting
any AC.

---

## 6. Risks

| Risk | Sev | Mitigation in this plan |
|---|---|---|
| `AddTool` panics on host schemas inside a handler goroutine (§0.5) | **High** | M1 AC3 nil/scalar/name validation; M2 AC5 panic-safety with a follow-up-request poisoned-state check |
| Executor implements resolution inside `getServer` and must rewrite (§0.6) | **High** | Wrapper architecture pinned in M2 preamble with the measured `no server available` / HTTP 400 evidence |
| `@nomcp` folded into the neutral gateway, silently hiding it from A2A (§0.7) | **High** | M3 AC5 asserts `@nomcp` hides in MCP **and survives in A2A + OpenAPI** |
| `check-boundaries`/`check-file-sizes` credited as `serveapi` evidence (§0.1, §0.2) | **Med** | Replaced by M1 AC2 `go list` two-sided check and an explicit `wc -l` table |
| Offline external fixture fails on toolchain or go.sum, looks like an import bug (§0.9) | **Med** | Exact recipe in M1 AC1, both gotchas proven and written down |
| `internal/apiserver/server.go` at 764/800 crosses the gate | **Med** | M3 file table forbids growth; only the one `isExposed` call at :686 changes (net 0) |
| Goroutine-count assertion is flaky | **Med** | M1 AC5 compares *growth between two burst sizes*, not an absolute number, and releases blocked callbacks to prove recovery |
| Timeout envelope's `application/json` diverges from the SDK success path's `text/event-stream` | **Low** | Accepted knowingly — the doc froze it; recorded here so a client-side surprise is not read as a bug |
| Wrapper body read precedes the SDK's `MaxBytesReader` → unbounded read | **Med** | M2 preamble mandates reusing `mcp.DefaultMaxRequestBodyBytes` (`4<<20`) in the wrapper |
| Executor reports a green `go test ./...` from inside the sandbox | **Med** | §1 table marks it FORBIDDEN as an executor verdict; controller re-runs (~20 s, §0.3) |
| `/tmp` worktree produces a red CI cannot reproduce | **Med** | Literal sibling path in §1; `TestIsTempPath` + `TestSolve_HardTimeout_FakeSolverIgnoringT` confirmed present [E13] |
| Sprint plan's `embedded_test.go` split | Low | Doc's single ~350 LOC file is split at source into `embedded_mcp_test.go` (~330) + `embedded_a2a_test.go` (~290); both well under 800 |

---

## 7. Open questions for the human

1. Accept ~25 h / 3.5 days, or take the §5 cut (defer the `isExposed` generalization)?
2. `serveapi/` is outside both CI gates (§0.1, §0.2). Extend `find internal cmd` and
   `check_boundaries.sh` to cover top-level library packages in this sprint, or file a follow-up? This
   plan assumes **follow-up** and compensates with in-test assertions.
3. Doc `Success Metrics` promise "an external-module compile test". Should that fixture be committed
   as a permanent test (needs a `replace` to a relative path and offline-safe flags in CI) or remain a
   run-once M1 gate? This plan treats it as a committed test using a relative `replace`; CI offline
   behavior is unverified (§9).

---

## 8. Evidence

Every codebase claim above traces to a row here. All commands run by the planner in this session at
HEAD `6bd8cda42`. Narrowed scopes are stated. Negative results carry a positive control.

| ID | Claim | Command | Observed |
|---|---|---|---|
| E1 | File-size gate is `find internal cmd`, so it cannot see top-level packages; `server.go` = 764; gate currently green | `sed -n '121,140p' make/code-health.mk`; `wc -l internal/apiserver/*.go \| sort -rn`; `find internal cmd -name '*.go' \| grep -c '^internal/apiserver/'`; `find internal cmd -name '*.go' \| grep -cE '^(runtime\|std)/'`; `ls runtime/*.go std/*.go \| wc -l`; `make -pn \| grep -c '^check-file-sizes:'` | gate body `for file in $$(find internal cmd -name "*.go"); do SIZE=$$(wc -l < "$$file"); if [ $$SIZE -gt 800 ]`; `764 internal/apiserver/server.go` (max of 42 files, 11203 total); apiserver files seen = **45**; runtime/std files seen = **0**; runtime/std `.go` files existing = **6**; target count = **1** |
| E2 | Boundary gate scans only `internal/<pkg>` for two fixed sets; `apiserver` in neither; top-level dirs never scanned; gate green | `grep -n 'CORE_PKGS=\|DASHBOARD_PKGS=' scripts/check_boundaries.sh`; `grep -cE '^(CORE_PKGS\|DASHBOARD_PKGS\|CORE_SURFACE_PKGS)=.*\bapiserver\b' scripts/check_boundaries.sh`; `grep -cE '^(CORE_PKGS)=.*\bparser\b' ...`; `make check-boundaries > /tmp/cb.out 2>&1; echo "rc=$?"` | `CORE_PKGS=(parser types eval core elaborate effects builtins lexer ast pipeline runtime link iface)`, `DASHBOARD_PKGS=(server coordinator observatory messaging)`; apiserver = **0**; control parser = **1**; `rc=0`, `OK: no architecture boundary violations.` |
| E3 | `go test ./cmd/ailang` PASSES in a Claude-agent session; loopback bind works here | `go test ./cmd/ailang > /tmp/cmdail.out 2>&1; echo "cmd rc=$?"; grep -c 'bind: operation not permitted' /tmp/cmdail.out`; separate temp module `httptest.NewServer` probe | `cmd rc=0`; `ok github.com/sunholo-data/ailang/cmd/ailang 19.282s`; bind-error count = **0**; probe: `bound OK at http://127.0.0.1:62979` |
| E4 | `internal/apiserver` binds no sockets; all 13 `httptest.NewServer` sites are in `cmd/ailang` | `grep -rn 'httptest.NewServer' --include='*_test.go' cmd/ailang internal/apiserver \| wc -l`; `grep -rln ... `; `grep -rl 'httptest.NewRecorder' --include='*_test.go' internal/apiserver \| wc -l` | **13** total, files: `configdriven_streaming_span_snapshot_test.go`, `configdriven_dispatch_test.go`, `configdriven_callstream_test.go`, `messages_send_test.go`, `configdriven_streaming_test.go`, `configdriven_harvest_test.go` — **all under `cmd/ailang`**; recorder files in apiserver = **12** |
| E5 | SDK stateless handler is fully exercisable via `httptest.NewRecorder`; `getServer`→nil yields HTTP 400 text/plain | temp module (`go.mod`/`go.sum` copied from ailang), `GOFLAGS=-mod=mod GOPROXY=off go test -run TestProbe -v` | `GET status=405 allow="POST" body="Method Not Allowed"`; `POST status=200 ct="text/event-stream"` body `event: message\ndata: {"jsonrpc":"2.0","id":1,"result":{...}}`; `nil-getServer status=400 ct="text/plain; charset=utf-8" body="no server available"` |
| E6 | `AddTool` PANICS on nil and on scalar `InputSchema`; `mcp.Tool` has `InputSchema any` + `OutputSchema any` | same probe module, two `recover()` cases; `grep -n '^type Tool struct' -A 45 <sdk>/mcp/protocol.go`; `grep -rn 'func (s \*Server) AddTool' -A 60 <sdk>/mcp/server.go` | `recovered=AddTool "bad2": missing input schema`; `recovered=AddTool "bad": can't marshal input schema to a JSON object: json: cannot unmarshal "\"scalar\"" into Go value of type map[string]interface {}`; struct has `InputSchema any` (:1924) and `OutputSchema any` (:1942), both documented to accept `json.RawMessage`; `AddTool` has 5 `panic(` sites (:282, :286, :289, :294, :297, :303, :308, :313) and only `s.opts.Logger.Error` for a bad name (:275) |
| E7 | 6 production `isExposed` call sites; `@nomcp` is a separate MCP-only filter | `grep -rn 'isExposed' --include='*.go' internal/ cmd/` (28 lines, un-truncated); `grep -rn 'IsNoMCP\|nomcp' --include='*.go' internal/apiserver/ \| grep -v _test` | production: `handler.go:142`, `server.go:686`, `a2a.go:57`, `a2a.go:185`, `openapi.go:187`, `mcp.go:91`; def `routes.go:246`; `mcp.go:94: if export.IsNoMCP {` / `:95 continue // @nomcp: served over HTTP/OpenAPI/A2A but absent from MCP`; `server.go:146` comment "hide from the MCP tool surface only (HTTP/OpenAPI/A2A unaffected)" |
| E8 | `serveapi/` absent (control: `runtime/` present); module path | `ls -d serveapi; echo rc=$?; ls -d runtime; echo rc=$?; head -1 go.mod` | `ls: serveapi: No such file or directory` `rc=1`; `runtime` `rc=0`; `module github.com/sunholo-data/ailang` |
| E9 | `apiserver.Config` has 15 fields, all process-wide; `New` returns `*Server` (no error) | `sed -n '150,175p' internal/apiserver/server.go`; `grep -n '^func New' internal/apiserver/server.go` | fields `Port, CORS, FrontendPath, StaticPath, Watch, EffCtx, MCP, MCPOnly, A2A, MaxUploadSize, APIKeyHeader, APIKeyEnv, LogLevel, RoutesOnly, NoFeedbackTool` = **15**; `171:func New(basePath string, cfg Config) *Server {` |
| E10 | `mcp.go` discards the per-request `*http.Request`; `Stateless: true` already set today | `sed -n '290,320p' internal/apiserver/mcp.go`; control `grep -c '^func ' internal/apiserver/mcp.go` | `return mcp.NewStreamableHTTPHandler(` / `func(r *http.Request) *mcp.Server { return ms.mcpServer },` / `&mcp.StreamableHTTPOptions{Stateless: true},`; control = **9** funcs |
| E11 | A2A errors are HTTP 200 + JSON-RPC; `-32603` unused | `grep -n 'WriteHeader\|32603\|32600\|32601\|32602\|StatusInternalServerError\|StatusServiceUnavailable' internal/apiserver/a2a.go`; `grep -n '^func ' internal/apiserver/a2a.go` | `304: w.WriteHeader(http.StatusOK) // JSON-RPC always returns 200.`; codes `-32600` (:127), `-32601` (:137), `-32602` (:145,158,166,178,190,267) = **6**; **no** `-32603`, **no** `StatusInternalServerError`, **no** `StatusServiceUnavailable`; funcs: `handleA2AAgentCard:16`, `buildAgentCard(r *http.Request):31`, `handleA2ATask:108`, `handleA2ATaskSend(w, req *a2aRequest):142` (no `*http.Request`), `handleA2ATaskGet:266`, `a2aError:297`, `a2aResult:309` |
| E12 | Sizes of every file the doc plans to modify | per-file `wc -l` existence loop | `.claude/rules/api-server.md` **46**; `docs/docs/guides/serve-api.md` **1384**; `cmd/ailang/serve_api_mcp_surface_test.go` **143**; `cmd/ailang/serve_api.go` **319**; `scripts/check_boundaries.sh` **131**; `internal/apiserver/feedback_tool_surface_test.go` **158**; `internal/apiserver/a2a_dispatch_gate_test.go` **90**; rule frontmatter `paths:` = `internal/apiserver/**`, `cmd/ailang/serve_api.go` (**no** `serveapi/**`) |
| E13 | `/tmp`-location-sensitive tests exist | `grep -rn 'func TestIsTempPath' internal/loader/`; `grep -rn 'func TestSolve_HardTimeout_FakeSolverIgnoringT' internal/smt/`; control `grep -rc '^func Test' internal/smt/*_test.go` | `internal/loader/loader_test.go:11,96,106`; `internal/smt/solver_timeout_test.go:19`; control shows 5-16 Test funcs per smt file |
| E14 | `internal/apiserver` does not import `serveapi` (control: imports `internal/embed`) | `go list -f '{{join .Imports "\n"}}' ./internal/apiserver \| grep -c 'sunholo-data/ailang/serveapi'`; same with `internal/embed` | serveapi = **0**; control `internal/embed` = **1** |
| E15 | External fixture: `go 1.24` fails on toolchain; `go 1.26.5` builds; internal import denied with the exact string | temp module + `replace` → repo; `GOFLAGS=-mod=mod GOPROXY=off go build ./...`; then `-tags denied` | with `go 1.24`: `rc=1`, `go: module /Users/voightkampff/dev/sunholo-data/ailang requires go >= 1.26.5 (running go 1.26.4)`; with `go 1.26.5`: `rc=0`; denied half: `rc=1`, `bad.go:5:8: use of internal package github.com/sunholo-data/ailang/internal/apiserver not allowed` |
| E16 | Fixture with transitive external deps builds offline once ailang's `go.sum` is copied | temp module `extconsumer2`, `cp <repo>/go.sum .`, import both `ailang/runtime` and `modelcontextprotocol/go-sdk/mcp`, `GOFLAGS=-mod=mod GOPROXY=off go build ./...` | `rc=0`; go auto-added `github.com/modelcontextprotocol/go-sdk v1.7.0` plus 8 indirects (`jsonschema-go`, `segmentio/asm`, `segmentio/encoding`, `uritemplate/v3`, `oauth2`, `sync`, `sys`, `time`) |
| E17 | Existing regression test counts (so M2 AC8 / M3 AC5 can prove they still execute) | `grep -c '^func Test' internal/apiserver/{nomcp,filtering,feedback_tool_surface,a2a_dispatch_gate,protocol,mcp_schema}_test.go` | `nomcp_test.go` **5**, `filtering_test.go` **8**, `feedback_tool_surface_test.go` **4**, `a2a_dispatch_gate_test.go` **2**, `protocol_test.go` **14**, `mcp_schema_test.go` **9** |
| E18 | Baseline: `internal/apiserver` green at HEAD | `go test ./internal/apiserver > /tmp/apisrv.out 2>&1; echo "rc=$?"` | `rc=0`, `ok github.com/sunholo-data/ailang/internal/apiserver 0.794s` |
| E19 | Lane A calibration | `git show --stat --oneline --no-renames aa02f0d9f \| tail -15` | `11 files changed, 1023 insertions(+), 41 deletions(-)`, of which `m-mcp-exact-tool-surface-lane-a-sprint-plan.md` = 534 |
| E20 | Complete public (non-internal, non-main) package list — 28 non-internal packages total | `go list -f '{{.ImportPath}} {{.Name}}' ./... \| awk '!/\/internal\// && $2 != "main" {print}'` (un-truncated) | `runtime`, `std`, `tests/golden/bytecode`, `tests/golden/codegen`, `testutil` — 5 library packages; `go list ./... \| grep -vc '/internal/'` = **28** |
| E21 | Worktree path free; a `/tmp` worktree exists as the anti-pattern | `git worktree list`; `ls -d /Users/voightkampff/dev/sunholo-data/.wt-*` | 14 worktrees; `.wt-iter139-mcp-lane-b` **absent**; siblings present: `.wt-iter117`, `.wt-iter121`; `/private/tmp/wt-iter130-qwen36` present |
| E22 | SDK installs its body cap in `ServeHTTP` **before** stateless dispatch; default 4 MiB | `sed -n '320,360p' <sdk>/mcp/streamable.go`; `grep -n 'MaxRequestBodyBytes\|DefaultMaxRequestBodyBytes' <sdk>/mcp/streamable.go` | `:344 if req.Body != nil && h.opts.MaxRequestBodyBytes > 0 {` `:345 req.Body = http.MaxBytesReader(...)`; then `:356 if h.opts.Stateless { h.serveStateless(...)`; `:225 const DefaultMaxRequestBodyBytes = 4 << 20` |
| E23 | SDK DNS-rebinding 403 needs `http.LocalAddrContextKey`, which `httptest.NewRequest` does not set → recorder tests unaffected | `sed -n '324,336p' <sdk>/mcp/streamable.go`; corroborated by E5 (POST via `httptest.NewRequest` returned 200 with `host="example.com"`, `remoteAddr="192.0.2.1:1234"`) | `if localAddr, ok := req.Context().Value(http.LocalAddrContextKey).(net.Addr); ok && localAddr != nil { if util.IsLoopback(localAddr.String()) && !util.IsLoopback(req.Host) { ... StatusForbidden` — guard not taken under a recorder |

---

## 9. Could NOT verify — the executing iteration must check first

1. **Whether a committed external-module fixture works in CI.** I proved it works locally offline with
   a copied `go.sum` and an absolute `replace` [E15][E16]. A *committed* test needs a **relative**
   `replace` and must survive CI's module setup. **Unverified.** If M1 AC1's committed form fails in
   CI, fall back to a `t.Skip`-guarded fixture gated on a build tag and record that narrowing.
2. **Behavior of the `serveapi` external fixture under `codex exec --sandbox workspace-write`
   specifically.** My probes ran in a Claude-agent session where loopback binds work [E3]; codex's
   sandbox denies them. The fixture uses no sockets, so it *should* transfer — but `go build` writing
   into a temp dir outside the workspace may be denied by `workspace-write`. **Mitigation**: create
   the fixture *inside* the worktree (e.g. `<worktree>/.snap-scratch/extfixture/`), not in `/tmp`.
3. **Whether the mission wants `serveapi` added to the two CI gates now** (open question 2).
4. **Actual `Invoke` argument/result codec shape for A2A.** I read `handleA2ATaskSend` signature and
   error codes [E11] but did not read its full 120-line body; the embedded adapter is new code so
   reuse is optional, but the executor should confirm no shared helper is silently required.
5. **The installed `ailang` binary is stale**: `AILANG v0.32.0-7-g1f8e7d16e-dirty`, commit `1f8e7d1`,
   not HEAD `6bd8cda42`. Irrelevant to this sprint (no CLI behavior change) but do not use the
   installed binary to verify anything about this branch.
6. **Issue #498's verbatim body.** The doc's V20 records `gh issue view` failing on network. I did not
   re-attempt it; the doc treats the mission prompt's requested resolution as the contract, and this
   plan inherits that. If the executing iteration has network, re-read #498 and diff it against the
   doc's Goals before starting M1.
