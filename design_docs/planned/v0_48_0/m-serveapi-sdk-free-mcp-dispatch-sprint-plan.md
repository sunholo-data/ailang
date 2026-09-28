# Sprint plan: M-SERVEAPI-SDK-FREE-MCP (#885)

**Design doc:** [m-serveapi-sdk-free-mcp-dispatch.md](m-serveapi-sdk-free-mcp-dispatch.md). Design Freeze fully checked
(D-A (a′), D-B (i), D-F accept; Mark, 2026-09-28).
**Mode:** attended, executed in the same session ("sprint plan and execute now", Mark 2026-09-28).
**Branch/worktree:** `design/m-serveapi-mcp-dispatch-seam` in `.claude/worktrees/m-serveapi-mcp-dispatch`.
**Estimate:** ~1,000 LOC (≈ 450 implementation + 550 tests/fixtures/gate), 4 milestones.
**Risk:** medium. The code is small and self-contained. The risk sits in wire-contract details, which the differential test covers.

## Milestones

### ✅ M1 — `hostcall` runner move (≈ 80 impl + 0 new test LOC)
- Create `serveapi/protocol/hostcall/runner.go`, containing the body of `serveapi/callbacks.go` exported as
  `hostcall.Runner`, `hostcall.New(timeout, max)` and `hostcall.Run[T]`.
- `git mv serveapi/callbacks_test.go serveapi/protocol/hostcall/runner_test.go`. Package rename only.
- Switch `serveapi.go`, `a2a_handler.go` and `mcp_handler.go` (plus their tests) to `hostcall`. Delete `callbacks.go`.
- **AC:** `go test ./serveapi/...` is green with no behaviour edits. The three runner tests pass in their new home.

### ✅ M2 — `mcphttp` dispatcher + goldens (≈ 280 impl + 250 test LOC)
- `serveapi/protocol/mcphttp/handler.go`: exported `NewHandler(Config) (http.Handler, error)`. `Config` holds the
  resolver/tools/invoker funcs, agent info, and a **required** runner. Covers W1–W16 per the design doc's wire
  contract, including the version-dependent batch rule (W6).
- `serveapi/mcp_handler.go`: becomes a thin adapter building `mcphttp.Config`. The SDK import goes.
- Rewrite `TestEmbeddedMCPReplayDefaultsContentTypeWhenTransportOmitsIt` (C4).
- `serveapi/protocol/mcphttp/handler_test.go`: one table test per W-row, plus the SSE-frame assertions.
- **AC:** design doc AC2, AC3, AC5, AC6, AC8. MUT-SSE-FRAME is run and reds.

### ✅ M3 — SDK differential (≈ 220 test LOC)
- `serveapi/mcp_parity_test.go` (test-only SDK import): the reference handler is rebuilt from the pre-change
  implementation shape (`mcp.NewStreamableHTTPHandler` with `Stateless: true`, the same tools). It runs a corpus
  of 30 or more requests, with `parity` rows compared after normalisation and `intentional-diff` rows asserted explicitly.
- Interop: drive the new handler with the SDK's own client (`mcp.NewClient` + streamable transport):
  initialize, list tools, call a tool.
- **AC:** design doc AC4. MUT-PARITY is run and reds.

### ✅ M4 — closure gate + docs (≈ 90 script + 60 docs LOC)
- `scripts/check_protocol_closure.sh`: four exact per-package arms (facade, `mcphttp`, `hostcall`, `protocol`).
  `scripts/test_check_protocol_closure.sh`: MUT-SDK-BACK and MUT-INTERNAL-BACK refusal cases.
- Changelog entry: the wire-change list plus a migration note (D-F).
- **AC:** design doc AC1 and AC7; `make check-protocol-closure`, `make test-check-protocol-closure`, `make ci-quick` and
  `go test -race ./serveapi/...` are all green.

## Out of scope for this sprint
AC-W1..W3 (the tag, the World scratch-worktree proof, and closing #885) come **after release**. A tag is required
first. The #885 close stays open until then.

## Separately found, not in this sprint
Our own MCP client (`internal/mcp_client`, pinned to protocol `2024-11-05`) requires an `Mcp-Session-Id` header
that the stateless prod server no longer sends. `ailang mcp status` reports `reachable:false` against prod
(measured 2026-09-28). This is Phase 1 of the never-executed
[m-mcp-2026-07-28-adoption.md](../v0_31_0/m-mcp-2026-07-28-adoption.md), to be raised to Mark as its own item.

## Execution record (2026-09-28, same session)

All four milestones ✅. Commits: M1 runner move, M2 dispatcher, M3 differential, M4 gate + changelog.
Deviations from the plan, each deliberate:
- **Goldens are inline** in `mcphttp/handler_test.go` (`TestWireContract`, exact bytes per W-row), not in
  `testdata/` files. It is the same assertion with one fewer indirection.
- **Differential lives in `mcphttp/parity_test.go`**, not `serveapi/mcp_parity_test.go`. It exercises the
  dispatcher directly, and the SDK import stays test-only either way.
- **One regression-test edit beyond C4:** the #603 battery's anti-vacuity control counted only *literal*
  reflection, which the SDK produced in `text/plain` errors. The dispatcher JSON-escapes every body, so the
  control now counts escaped reflection and additionally **fails** on any literal reflection. That is
  stricter than before.
- The runner tests moved to `hostcall` with mechanical renames, because they read the runner's unexported
  slot channel.

AC-W1..W3 (tag, World scratch-worktree proof, #885 close) remain open until a release carries this.
