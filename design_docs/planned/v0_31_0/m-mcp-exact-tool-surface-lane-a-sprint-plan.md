# Sprint Plan: M-MCP-EXACT-TOOL-SURFACE — Lane A

**GitHub issue**: [sunholo-data/ailang#498](https://github.com/sunholo-data/ailang/issues/498)
**Design doc**: none — mission-classified as a small settled bug fix, same shape as
[m-nightly-run-validity-gate](../v1_0_0/m-nightly-run-validity-gate-sprint-plan.md) (iteration 119),
which also ran doc-less. Lane B (callback-driven embedder API) is design-doc-sized and OUT of scope.
**Sprint ID**: `M-MCP-EXACT-TOOL-SURFACE-A`
**Branch**: `sprint/m-mcp-exact-tool-surface`
**Worktree**: `/Users/voightkampff/dev/sunholo-data/ailang/.claude/worktrees/iter120-mcp-surface`
**Base**: `origin/dev` @ `0f0896c70`
**Planner model**: `claude-opus-4-8`
**Planned**: 2026-07-29 (mission-control iteration 120)
**Executor**: `codex:gpt-5.6-sol`, `workspace-write` sandbox, **loopback socket binds denied**
**Estimate**: **~1.0 working day (6.5h bottom-up + 20% = ~7.8h)** — the top of the mission's 0.5–1.0d box, not the bottom. Justified in §6.
**Risk**: medium-low. Small production diff (~43 LOC). The real risk is a **default-behaviour choice
that would break the live public MCP server** — see §3 D1, which is why this sprint does NOT touch
`--routes-only` semantics.

---

## 0. Premise re-verification (first-party, this worktree, at `0f0896c70`)

The controller labelled every handed-down fact and asked me to re-check. I did. **One is refuted**
(the `--a2a` leg), **one is materially narrower than stated** (which changes nothing about the fix
but everything about what to test), and **while refuting it I found a second, independent defect
in the same "exact surface" family that nobody had filed** (§0.3, M3).

### CONFIRMED

| # | Premise | How I verified it |
|---|---|---|
| C1 | `ms.registerFeedbackTool()` is called unconditionally at `internal/apiserver/mcp.go:43`, after `registerTools()` | Read `mcp.go:29-45`. The call sits outside every predicate. |
| C2 | Defined at `internal/apiserver/feedback_tool.go:52`; `mcp.go:43` is its only call site | `grep -n 'registerFeedbackTool' internal/apiserver/*.go` → exactly 2 hits: the definition and the call. |
| C3 | The live matrix (`--routes-only` filters the user's own function while the built-in survives) | **Reproduced in-process, socket-free, with a positive control.** Scratch test using `mcp.NewInMemoryTransports()` + a real `initialize`/`tools/list` client session against a module with one `@route` export (`status`) and one plain export (`addOne`): unfiltered → `[addOne status submit_feedback]`; `RoutesOnly:true` → `[status submit_feedback]`. `status` is the positive control — it proves the instrument sees tools; `addOne` correctly disappears; `submit_feedback` does not. Scratch files deleted; the worktree is clean. |
| C4 | No off-switch exists | `grep -rn 'AILANG_FEEDBACK\|DISABLE_FEEDBACK\|no-feedback\|nofeedback' internal/ cmd/` → only `internal/feedbackgate/*` + `internal/coordinator/feedback_gate_wiring.go`, which is the coordinator triage gate, a different subsystem. Positive control present (those hits came back), so the negative is credible. |
| C5 | Egress is gated on `AILANG_STORAGE=gcp` **and** `AILANG_CLOUD_PROJECT` | `internal/feedback/publisher.go:121-129` in `newPublisher`. Both are hard errors; `Get` memoises the failure via `sync.Once`. The controller's "advertises an egress tool it cannot perform" framing is exactly right — **not** a default-config exfiltration. |
| C6 | `@nomcp` (`ee04f13d0`) is the naming/test precedent | `internal/apiserver/routes.go:246` (`isExposed`), `mcp.go:85-90` (`IsNoMCP` skip), `internal/apiserver/nomcp_test.go:206-263`. Confirmed it **cannot** be reused directly: `@nomcp` is a per-export annotation extracted from user `.ail` source (`extractNoMCPAnnotations`), and `submit_feedback` has no `.ail` export — the `.ail` mirror `mcp_tools/feedback.ail` is `@noexpose`'d and is documentation only (`feedback_tool.go:40-50`). The reusable part is the **test harness and the vocabulary**, not the mechanism. |

### REFUTED

> **R-A. `--a2a` does NOT advertise `submit_feedback`. The `--a2a` leg of the defect, as stated in
> the issue title, is false.** This is a success of the loop: a milestone that "fixed A2A discovery"
> would have been a **no-op that shipped green**.

`buildAgentCard` (`internal/apiserver/a2a.go:30-57`) builds `skills[]` by iterating `s.modules` and
applying `s.isExposed(export)`. It never touches `ms.mcpServer` and has no access to Go-side tools.
I probed the real endpoint in-process (`httptest.NewRequest` + `mux.ServeHTTP` — no socket, `srv.a2aEnabled = true`, `RoutesOnly: true`):

```
A2A card contains submit_feedback: false
A2A card contains status:          true    <- positive control
A2A card contains addOne:          false   <- routes-only filter working
```

The issue reporter independently saw the same thing without realising it — #498 says the card had
"the same 26 **non-feedback** entries". So the title over-reaches. **Consequence for scope**: there
is no A2A discovery work in this sprint, and the executor must not invent any.

**This does not make the sprint vacuous for the sibling.** The sibling consumes MCP-HTTP, and
MCP-HTTP *is* covered: `server.go:509` (`StartMCP` → stdio) and `server.go:588` (`buildRoutes` →
`/mcp/`) **both** call `NewMCPServer(s)`. One fix at `mcp.go:43` closes `--mcp` and `--mcp-http`
together. Verified by reading both call sites.

### MATERIALLY NARROWED

**R-B. The `std/io` tool leak reported in #498 did NOT reproduce here.** #498 reports that
unfiltered `tools/list` on a 5-module directory returned 27 tools including 8 embedded
`std/io` exports (`exit`, `writeBytes`, ...) and that the A2A card carried
`<embedded>.std.io.writeBytes`. I built a module that imports `std/io (println)` and served it
unfiltered: `tools: [shout submit_feedback]` — **no `std/io` entries**. Either it is
configuration-dependent (directory-vs-file serving, package load path) or it was fixed after
v0.30.0. I am **not** scoping it: it is a separate, larger, and genuinely security-relevant leak
(`std.io.exit` as an agent-callable tool) that needs its own first-party repro before anyone prices
it. **Recommend the controller file it separately.** Do not let the executor chase it.

### NEW DEFECT FOUND WHILE VERIFYING (this is M3)

**R-C. A2A `tasks/send` dispatch has no `isExposed` gate: `--routes-only` and `@noexpose` are
card-only on A2A — functions are hidden from discovery but remain fully callable.**

Every other surface gates invocation: HTTP `handler.go:142`, MCP `mcp.go:85`, OpenAPI
`openapi.go:187`, A2A **card** `a2a.go:57`. `handleA2ATaskSend` (`a2a.go:161-193`) checks only that
the module is loaded and `e.Name == funcName`; it then calls `s.engine.CallPreserveFloats(...)`.
I probed it under `RoutesOnly:true` with `skill_id=test.api.keys.addOne` (a non-`@route` export
that is **absent from the card**):

```
POST /a2a/ -> 200 {"result":{"status":{"message":"expected float arguments","state":"failed"}}}
```

That message comes from the *engine argument coercion path*, i.e. dispatch was **allowed through**.
Had the gate existed the response would have been `-32602 function %q not found in module %q`.

This is the exact mirror image of the `--caps` discovery/execution split the sprint is chartered to
document, on the same protocol family, and CLAUDE.md §3 ("systemic fixes — audit before patching")
says fix the pattern rather than the reported instance. It is 4 lines of production code. **M3 is
CUTTABLE** (§7) if the budget bites — but if cut it must be filed, not forgotten.

---

## 1. Goal

Give an embedding host an **exact** MCP tool surface from the CLI, and stop lying to models about a
capability the server usually cannot perform.

After this sprint:

| invocation | `tools/list` |
|---|---|
| `serve-api --mcp DIR` | user exports **+ `submit_feedback`** (unchanged — see D1) |
| `serve-api --mcp --no-feedback-tool DIR` | user exports, **exactly** |
| `serve-api --mcp --routes-only --no-feedback-tool DIR` | `@route` exports, **exactly** |
| `serve-api --mcp-http --no-feedback-tool DIR` | same, over `/mcp/` |

Plus: `--caps` is documented as gating **execution, not discovery**; `--routes-only` is documented
as filtering **user exports**, with the built-in controlled separately; and (M3) A2A can no longer
invoke what it refuses to advertise.

**Non-goals** (Lane B, do not drift): caller-supplied tool descriptors, per-session/per-principal
capability resolution, mountable `http.Handler` export, callback-driven invocation. Also non-goals:
the `std/io` leak (R-B), any change to the feedback publisher, any change to the coordinator's
`internal/feedbackgate`.

---

## 2. Impact statement (use this wording; do not inflate it)

> The **discovery** defect is unconditional and real: every model that connects to any
> `ailang serve-api --mcp`/`--mcp-http` process is told a public-feedback egress tool exists, and
> there is no way to withdraw that claim. **Egress itself is not open by default** —
> `internal/feedback/publisher.go:121-129` requires *both* `AILANG_STORAGE=gcp` and
> `AILANG_CLOUD_PROJECT`; without them the publisher fails to construct and no client is opened. So
> a default local server **advertises an egress tool it cannot perform** — a false capability claim
> to every connected model, plus a live egress path in any environment that does carry those two
> vars. This is **not** a default-config exfiltration, and must not be described as one.

---

## 3. Design decisions (settled with evidence)

### D1 — Mechanism: a new opt-out CLI flag `--no-feedback-tool`. Default behaviour UNCHANGED.

**Options weighed:**

| option | verdict |
|---|---|
| Make `--routes-only` suppress the built-in | **REJECTED — it breaks production.** |
| Suppress by default, add `--feedback-tool` to opt in | **REJECTED** — same breakage, worse skew. |
| Per-export annotation (`@nomcp`-style) | **IMPOSSIBLE** — see C6; the built-in has no `.ail` export. |
| Env var only (`AILANG_MCP_NO_FEEDBACK_TOOL`) | **REJECTED** — see below. |
| **New flag `--no-feedback-tool`, default off** | **CHOSEN.** |

**The evidence that kills the `--routes-only` option** — and it is concrete, not hypothetical.
`docs/docs/guides/agent-mcp.md:127-129` documents the *live* public AILANG MCP server:

```
- Cloud Run service runs `ailang serve-api --mcp-http --routes-only --caps FS,Env`
- `--routes-only` means only `@mcp_name`/`@route` exports register as tools — no helper leakage
```

and lines 22/100/106 of the same file document `submit_feedback` as that server's **one write tool**
and the intended public triage channel. So the flagship deployment runs `--routes-only` **and
depends on the built-in surviving it**. Changing `--routes-only` semantics would:

1. silently delete the public feedback channel the moment a new binary rolls out;
2. require a matching config change in the **separate Terraform deploy project** (per project
   memory: ailang-multivac deploy is Terraform-only, in a different repo) — a cross-repo change this
   sprint cannot make atomically, guaranteeing a version-skew window.

The controller's framing — "`--routes-only` arguably already promises suppression" — is *right about
the words* and wrong about the cost. The honest resolution is: keep the behaviour, **fix the
promise in the docs** (M4), and give callers an explicit lever.

**Why not env-var-only**: `AllowDropsEnvVar` (`server.go:~228`) sets this repo's precedent — and its
own comment says env-var-only exists to force operators to make a **bypass** explicit in a
manifest. `--no-feedback-tool` is a *tightening*, not a bypass; the CLI is the discoverable place
for it, and adding both a flag and an env var would create two sources of truth (CLAUDE.md §2).
Recorded as a deliberate non-goal so nobody re-derives it.

**Back-compat cost of the chosen option, stated plainly**: none for existing users, and the
downstream (#498) must add one flag to its invocation. The residual is that an operator who passes
`--routes-only` alone still gets a tool they did not ask for. **CLAUDE.md §2 forbids that being
*silent***, so M1.4 emits a one-line stderr notice naming the flag whenever the built-in is
registered under `--routes-only`. `log` writes to stderr by default and nothing in the serve path
calls `log.SetOutput` (verified: the only hits are a test helper and an unrelated `cmd/`), so this
is safe in stdio MCP mode where stdout is the protocol channel.

### D2 — Coverage: one fix covers `--mcp` and `--mcp-http`. `--a2a` needs no discovery fix.

Both transports construct through `NewMCPServer(s)` (`server.go:509`, `server.go:588`); the
suppression predicate therefore lives on the `Server`, flows `Config → New() → Server` exactly like
`routesOnly` (`server.go:165,218`), and is read once inside `NewMCPServer`. A2A is refuted (R-A).
The A2A work in this sprint (M3) is about **invocation**, not discovery, and is a different bug.

### D3 — Test shape: in-memory MCP transport, positive control inside every assertion.

`internal/apiserver/nomcp_test.go:206-263` already establishes the pattern: `mcp.NewInMemoryTransports()`
+ `ms.mcpServer.Connect` + `client.Connect` + `ListTools`. **No sockets.** I ran the whole package:
`ok github.com/sunholo-data/ailang/internal/apiserver 0.711s`, and
`grep -rn 'httptest.NewServer|net.Listen|ListenAndServe' internal/apiserver/*_test.go` returns
**nothing** — the package is entirely socket-free and safe under the executor's sandbox.
(`httptest.NewRequest` + `httptest.NewRecorder` + `mux.ServeHTTP` do **not** bind; only
`httptest.NewServer` would.)

Every new assertion must name a real user tool that IS present in the **same** `tools/list`
response that asserts the built-in is absent. Rationale is on the record: the controller's first
probe returned an empty list for every row with `rc=1`/`server is closing: EOF` — a broken
instrument indistinguishable from "no tools found". An assertion of the form "`submit_feedback` not
in tools" passes trivially against an empty list.

---

## 4. Milestones

Each is independently committable with its own verification command and its own named mutation.

Run every command from the worktree root:
`/Users/voightkampff/dev/sunholo-data/ailang/.claude/worktrees/iter120-mcp-surface`.

---

### M1 — Suppress the built-in inside `NewMCPServer` (~2.0h, ~25 prod + ~120 test LOC)

**Files**: `internal/apiserver/server.go`, `internal/apiserver/mcp.go`,
`internal/apiserver/feedback_tool_surface_test.go` (new).

**M1.1** Add `NoFeedbackTool bool` to `Config` (`server.go:150-166`, next to `RoutesOnly`, with a
comment: *"suppress the built-in submit_feedback MCP tool; user exports unaffected"*). Add
`noFeedbackTool bool` to the `Server` struct and wire `noFeedbackTool: cfg.NoFeedbackTool` in `New()`
(`server.go:~218`, next to `routesOnly:`).

**M1.2** Guard the call at `mcp.go:43`:

```go
if !srv.noFeedbackTool {
    ms.registerFeedbackTool()
}
```

Update the `NewMCPServer` doc comment (`mcp.go:23-28`) to say the built-in is registered *unless*
suppressed. Do **not** touch `registerFeedbackTool` itself, and do **not** delete
`feedback_tool_test.go` — those tests build `&MCPServer{feedbackRL: ...}` by hand and call
`handleSubmitFeedback` directly, so they are unaffected by registration and must keep passing.

**M1.3** New test file with three tests, all on the in-memory transport:

- `TestFeedbackTool_PresentByDefault` — default `Config`: `submit_feedback` present **and** the
  `@route` export `status` present. (This is the over-suppression guard: it fails if someone just
  deletes the call.)
- `TestFeedbackTool_SuppressedWhenConfigured` — `Config{NoFeedbackTool: true}`: `submit_feedback`
  **absent** and `status` **present in the same response** (positive control). Assert on the same
  `res.Tools` slice; on failure print the full `toolNames(res.Tools)`.
- `TestFeedbackTool_SuppressedIsAlsoUncallable` — with suppression on, `CallTool{Name:"submit_feedback"}`
  returns a non-nil error (unregistered), mirroring `nomcp_test.go:255-262`.
- Add a `RoutesOnly:true, NoFeedbackTool:true` sub-case asserting the surface is **exactly**
  `["status"]` (a set equality, not a subset check) — this is the sibling's actual ask.

Reuse `toolNames` (already in `nomcp_test.go:268`). Write a local server builder modelled on
`nomcpTestServer` (`nomcp_test.go:122-166`) — including its `AILANG_STDLIB_PATH` walk-up, which is
required or `LoadModules` fails.

**M1.4 (cuttable, ~20min)** In `registerFeedbackTool`'s call site, when the built-in **is** registered
and `srv.routesOnly` is true, `log.Printf` one line to stderr: name the tool and name
`--no-feedback-tool`. Capture-and-assert it with the `log.SetOutput(&buf)` pattern from
`cold_start_test.go:121-129`. This is the CLAUDE.md §2 "not silent" mitigation for keeping the
default.

**Verify**:
```bash
go test ./internal/apiserver/ -run 'TestFeedbackTool|TestNoMCP|TestHandleSubmitFeedback' -v
go test ./internal/apiserver/
```

**Mutation proofs (all three required; each must be applied, observed red, and reverted):**

| # | Mutation | Must fail |
|---|---|---|
| MUT-1a | Revert `mcp.go` to the unconditional `ms.registerFeedbackTool()` | `TestFeedbackTool_SuppressedWhenConfigured` |
| MUT-1b | Delete the `ms.registerFeedbackTool()` call entirely | `TestFeedbackTool_PresentByDefault` |
| MUT-1c | Drop `noFeedbackTool: cfg.NoFeedbackTool` from `New()` (leave the field zero) | `TestFeedbackTool_SuppressedWhenConfigured` — this proves the `Config`→`Server` wiring, which MUT-1a does not |

---

### M2 — CLI flag `--no-feedback-tool` + end-to-end stdio proof (~2.0h, ~10 prod + ~130 test LOC)

**Files**: `cmd/ailang/serve_api.go`, `cmd/ailang/serve_api_mcp_surface_test.go` (new).

**M2.1** Add the flag next to `routesOnlyFlag` (`serve_api.go:33`):

```go
noFeedbackToolFlag := fs.Bool("no-feedback-tool", false,
    "Suppress the built-in submit_feedback MCP tool (exact tool surface)")
```

and `NoFeedbackTool: *noFeedbackToolFlag` in the `apiserver.Config` literal (`serve_api.go:~128-141`).

**M2.2** Add a line to `printServeAPIHelp()` (`serve_api.go:~258`, after `--routes-only`).

**M2.3 — the end-to-end proof.** `--mcp` is **pure stdio: it binds no socket**, so it is fully
runnable under the executor's sandbox. Build the binary into `t.TempDir()` with `exec.Command("go","build",...)`,
then drive two probes with a real `initialize` → `notifications/initialized` → `tools/list` handshake
against a one-`@route`-plus-one-plain-export module:

- default → asserts `submit_feedback` present, user tool present;
- `--no-feedback-tool` → asserts `submit_feedback` absent, user tool present **in the same response**.

**Framing requirements — these are the difference between a test and a broken instrument.** The
controller's first probe of this exact handshake returned empty lists with `rc=1` /
`server is closing: EOF`. Therefore:

- **Do not** close stdin and then read. Keep stdin open; read stdout line-by-line with
  `bufio.Scanner` until the JSON-RPC object with `"id":2` arrives; only then close.
- **Do not** use fixed `sleep`s for sequencing. Wait for the `initialize` **response** (`id:1`)
  before sending `notifications/initialized`.
- Wrap the whole probe in a 30s `context.WithTimeout`; on timeout, fail with the captured **stderr**
  attached (that is where module-load errors go).
- Guard with `if testing.Short() { t.Skip(...) }` so `go test -short` stays fast.
- If a probe returns an **empty** tool list, the test must `t.Fatal` with "instrument failure —
  positive control missing", never pass.

**Verify**:
```bash
go test ./cmd/ailang/ -run TestServeAPI_MCPToolSurface -v
go build ./... && go vet ./internal/apiserver/ ./cmd/ailang/
```

> **Sandbox note for the executor**: do **not** run the whole `cmd/ailang` package. Six test files
> there (`configdriven_*`, `messages_send_test.go`) call `httptest.NewServer` / `net.Listen` and will
> fail with `bind: operation not permitted` — a sandbox artefact, **not** a regression. Always use
> `-run`. If you ever do run the package, label any bind failure
> **UNINFORMATIVE UNDER SANDBOX** in your report so the controller re-runs it outside.

**Mutation proof**: delete `NoFeedbackTool: *noFeedbackToolFlag` from the `Config` literal →
`TestServeAPI_MCPToolSurface` must fail on the suppressed row. (This is the *only* guard that covers
the CLI wiring line; M1's tests cannot see it.)

**Cut-down if M2.3 exceeds 2h** (see §7): keep M2.1/M2.2, drop the in-test `go build`, and instead
record the manual command + its output verbatim in the milestone commit message, clearly labelled
`MANUALLY VERIFIED, NOT CI-GUARDED`. Do not silently ship an unguarded flag.

---

### M3 — CUTTABLE: gate A2A `tasks/send` on `isExposed` (~1.25h, ~8 prod + ~90 test LOC)

**Files**: `internal/apiserver/a2a.go`, `internal/apiserver/a2a_dispatch_gate_test.go` (new).

**M3.1** In `handleA2ATaskSend` (`a2a.go:~183-193`), the existence loop currently sets `found` on a
name match alone. Require exposure too:

```go
for _, e := range modInfo.Exports {
    if e.Name == funcName {
        found = s.isExposed(e)   // hidden exports are indistinguishable from absent
        break
    }
}
```

**Error shape is a deliberate design point, not an oversight**: reuse the existing
`-32602 function %q not found in module %q` message unchanged. `docs/docs/guides/serve-api.md:1312`
states the project convention explicitly — `FUNCTION_NOT_FOUND` is *"intentionally indistinguishable
so `@noexpose` reveals nothing to external callers."* Do **not** invent a "function is hidden" error.

**M3.2** Test with positive control in the same test:

- `RoutesOnly:true` + `tasks/send` on the non-`@route` export → JSON-RPC error `-32602`
  (**not** a 200 task result, and not an engine-level failure message);
- same server, `tasks/send` on the `@route` export → a completed task result (positive control:
  proves dispatch still works and the test is not just erroring on everything);
- `@noexpose` export with `RoutesOnly:false` → `-32602`.

Socket-free: `srv.a2aEnabled = true; mux := srv.buildRoutes()` + `httptest.NewRequest`/`NewRecorder`.
(Confirmed working — this is exactly how I found the bug.)

**Verify**: `go test ./internal/apiserver/ -run 'TestA2A' -v`

**Mutation proof**: revert `found = s.isExposed(e)` to `found = true` →
`TestA2ADispatch_RespectsRoutesOnly` must fail with the hidden export reaching the engine.

**If cut**: file a GitHub issue titled *"A2A tasks/send bypasses --routes-only / @noexpose (card-only
filtering)"* with the probe output from §0 R-C, and say so in the sprint report. Do not drop it silently.

---

### M4 — Docs: the `--caps` split, the `--routes-only` promise, the new flag (~1.25h, ~60 doc lines)

**Files**: `docs/docs/guides/serve-api.md`, `docs/docs/guides/agent-mcp.md`, `CHANGELOG.md`.

**M4.1 `serve-api.md`, flag list (~line 525)** — add `--no-feedback-tool` after `--routes-only`.
(Note: this block has already drifted from the CLI — it is missing `--a2a`, `--log-level`,
`--max-memory`. Adding those is **optional**; if it takes more than 5 minutes, skip it and say so.)

**M4.2 `serve-api.md`, MCP "Filtering" bullet (~line 399)** — it currently reads
*"`--routes-only` and `@noexpose` are respected in MCP `tools/list`, consistent with HTTP and
OpenAPI."* That is now the misleading sentence. Amend it to state that these filters apply to
**module exports**, that the server registers one built-in Go-side tool (`submit_feedback`) which is
**not** an export and is therefore not affected by them, and that `--no-feedback-tool` removes it.

**M4.3 `serve-api.md`, new subsection after `--routes-only` (~line 1090)** —
`### --no-feedback-tool — Exact MCP tool surface`: what it does, the `--routes-only --no-feedback-tool`
recipe for an exact `@route` surface, that it covers `--mcp` and `--mcp-http` identically, and that
A2A skills never included the built-in in the first place.

**M4.4 `serve-api.md`, the `--caps` discovery/execution note** — near the `--caps` docs (~line 716)
and in the MCP section: **`--caps` gates effect *execution*, not tool *discovery*.** A tool whose
effects are not granted is still advertised in `tools/list`; the failure surfaces at call time. Give
the measured example: `--caps ''` leaves the tool list unchanged. State plainly that `--caps` is
**not** a discovery filter and that `--routes-only` / `@noexpose` / `@nomcp` / `--no-feedback-tool`
are the discovery controls.

**M4.5 `agent-mcp.md` (~line 127)** — one clarifying sentence: the public server's
`--mcp-http --routes-only` invocation **intentionally** keeps `submit_feedback` (it is that server's
purpose), and self-hosted operators who want an exact surface add `--no-feedback-tool`. This is the
line that stops a future reader from "fixing" `--routes-only` and breaking production.

**M4.6 `CHANGELOG.md`** — under `## [Unreleased] / ### Added`, one entry for the flag naming
`refs #498`; under `### Fixed`, M3 if it shipped. Say explicitly that the default is unchanged.

**Verify**:
```bash
make check-file-sizes
gofmt -l ./internal/apiserver ./cmd/ailang   # must print nothing
make lint
go test ./internal/apiserver/
```
Plus a manual grep that the three claims added to the docs are true of the code as committed.

---

## 5. Sequencing and commits

| order | commit | depends on |
|---|---|---|
| 1 | `fix(apiserver): --no-feedback-tool suppresses the built-in MCP tool (refs #498)` (M1) | — |
| 2 | `feat(cli): serve-api --no-feedback-tool + stdio tool-surface E2E (refs #498)` (M2) | M1 |
| 3 | `fix(apiserver): A2A tasks/send now honours --routes-only/@noexpose` (M3) | independent of 1–2 |
| 4 | `docs(serve-api): --caps gates execution not discovery; document --no-feedback-tool (refs #498)` (M4) | 1–3 |

Use `refs #498` throughout. **Do not use `Fixes #498`** — #498's stated resolution is the Lane B
embedder API; Lane A is explicitly the *interim* unblock and must not auto-close the issue. Post a
comment on #498 instead, saying which of the seven requested behaviours (only #6) Lane A delivers.

---

## 6. Estimate, bottom-up (not top-down)

| item | h |
|---|---|
| M1.1 Config/Server/New wiring | 0.2 |
| M1.2 guard + doc comment | 0.1 |
| M1.3 four tests + server builder | 1.0 |
| M1.4 stderr notice + log-capture test | 0.35 |
| M1 mutation proofs (3 × build/run/revert) | 0.35 |
| M2.1/M2.2 flag + help | 0.25 |
| M2.3 stdio E2E (build-in-test, framing, timeout, 2 probes) | 1.5 |
| M2 mutation proof | 0.25 |
| M3.1 gate | 0.3 |
| M3.2 tests (3 cases) | 0.75 |
| M3 mutation proof | 0.2 |
| M4 docs (5 edits) + CHANGELOG | 1.0 |
| M4 lint/fmt/file-size/full-package run | 0.25 |
| **subtotal** | **6.5** |
| +20% unknowns | 1.3 |
| **total** | **~7.8h ≈ 1.0 day** |

**This is the top of the 0.5–1.0d box, not the bottom, and I am saying so rather than absorbing it.**
The production diff really is ~43 lines; the cost is entirely in *proving* it — three mutation
proofs, a hand-rolled JSON-RPC stdio handshake that has already broken once for the controller, and
five doc edits that each make a factual claim about behaviour. Without M2.3 and M3 the sprint is
~4.5h; both are defensible cuts (§7) and both have a named consequence.

**Velocity check**: comparable recent single-concern fixes in this repo — `a089a6fe7` (nightly
validity gate), `9253ec8a8` (Z3 hard timeout), `5998f4039` (SMT ADT declarations) — each landed as
one PR inside a day. This is in family.

---

## 7. Cut order (if the budget bites, cut in this order and SAY SO)

1. **M1.4** (stderr notice) — costs the §2 "not silent" mitigation. Cheapest cut.
2. **M2.3** (stdio E2E) → M2-lite with a recorded manual verification. Costs CI coverage of the CLI
   wiring line; the mutation becomes undetectable in CI. Must be labelled in the commit.
3. **M3** (A2A gate) → file the issue with the §0 R-C probe output. Costs a real security-relevant
   fix, but it is genuinely a different bug from #498.

**Never cut**: M1 (the fix), M2.1/M2.2 (the flag is useless without a CLI surface), M4.4 (the
`--caps` note is half of the chartered Lane A deliverable).

---

## 8. Risks the executor must actively guard

| # | risk | guard |
|---|---|---|
| R1 | **Changing `--routes-only` semantics.** The tempting "honest" fix silently kills the live public MCP server (D1). | `--routes-only` behaviour must be **byte-identical** after this sprint. `TestIsExposed_RoutesOnly*` (`filtering_test.go:78-113`) and `mcp_schema_test.go:107-265` must pass **unmodified**. If you find yourself editing either, stop. |
| R2 | **Empty-list false pass.** "`submit_feedback` not in tools" passes against a broken instrument. | Every absence assertion carries a presence assertion on the **same** response object. Any empty tool list is a `t.Fatal`, never a pass. |
| R3 | **Sandbox socket denial misread as a regression.** | `internal/apiserver` is socket-free (verified: no `httptest.NewServer`/`net.Listen`; package runs in 0.71s). `cmd/ailang` is **not** — always use `-run`. Label any `bind: operation not permitted` as **UNINFORMATIVE UNDER SANDBOX**. |
| R4 | **Lane B drift.** #498's headline ask is a callback-driven embedder API. | No new exported types on `apiserver` beyond one `Config` field. No `http.Handler` export. No per-request/per-principal anything. If a change needs a new public interface, it is Lane B — stop and report. |
| R5 | **Chasing the `std/io` leak** (R-B) — it did not reproduce and is a much bigger fix. | Out of scope. Report it; do not scope it. |
| R6 | **Deleting `feedback_tool_test.go`.** Those tests hand-build `&MCPServer{}` and look "orphaned" once registration is conditional. CLAUDE.md coding-standards: never delete on a linter's say-so. | All 5 tests in `feedback_tool_test.go` must still pass, untouched. |
| R7 | **Negative-boolean confusion.** `NoFeedbackTool` inverts twice (`if !srv.noFeedbackTool`). An inverted wiring bug passes a "flag exists" test. | MUT-1c exists precisely for this; the default-present test (M1.3 #1) catches the other polarity. |
| R8 | **M3 leaking information via a new error message.** | Reuse the existing `-32602` text verbatim (serve-api.md:1312 convention). |

---

## 9. Acceptance criteria

- [x] `serve-api --mcp --no-feedback-tool DIR` → `tools/list` has **no** `submit_feedback` and **does** have the user's exports.
- [x] `serve-api --mcp --routes-only --no-feedback-tool DIR` → `tools/list` is **exactly** the `@route` set.
- [x] `serve-api --mcp DIR` (no flag) → `submit_feedback` still present. **Default is unchanged.**
- [x] `--mcp-http` behaves identically to `--mcp` (same `NewMCPServer` path; asserted in M1, no separate transport test needed).
- [x] `tools/call submit_feedback` errors when suppressed.
- [x] Every new test carries a positive control; no absence assertion can pass on an empty list.
- [x] All named mutations (MUT-1a/1b/1c, M2, M3) observed **red**, then reverted — with the failing test names quoted in the sprint report.
- [x] `--routes-only` semantics unchanged; `filtering_test.go` and `mcp_schema_test.go` pass unmodified.
- [x] Docs state: `--caps` gates execution not discovery; `--routes-only` filters exports not built-ins; `--no-feedback-tool` exists.
- [x] `go test ./internal/apiserver/` green; `make lint`, `gofmt -l`, `make check-file-sizes` clean.
- [x] CHANGELOG `[Unreleased]` updated; `refs #498` on every commit; **no** `Fixes #498`.
- [x] Sprint report states explicitly which milestones were cut, if any, and what was filed instead.

---

## 10. Handoff notes for the executor

- Work **only** in the worktree. The main checkout is dirty with a sibling agent's work — do not
  touch it, do not `git checkout`, do not `git stash`.
- The controller's `--a2a` premise is **refuted** (§0 R-A). If you find yourself writing a test that
  asserts `submit_feedback` is gone from the A2A card, you are writing a test that already passes.
  Stop and re-read §0.
- The reproduction is cheap and socket-free; reproduce it yourself before you fix anything, and
  paste the before/after tool lists into your report.
- If you refute anything in **this** plan, say so loudly. Two of the last three iterations shipped a
  false "verified" fact to the next role; both were caught by the next role re-checking. That is the
  loop working.
