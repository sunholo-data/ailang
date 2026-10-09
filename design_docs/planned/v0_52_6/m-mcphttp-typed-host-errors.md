# M-MCPHTTP-TYPED-HOST-ERRORS: a host can return a typed JSON-RPC error from Invoke (#1602)

**Status**: Planned
**Target**: v0.52.6
**Priority**: P2 — the only upstream item blocking ailang-world row 136
(`w-mcp-effects-unrecorded-is-untyped`, World PR #219) from telling an MCP client "your effect
ran, but the commit failed"
**Estimated**: 1–1.5 days (M1 3h, M2 2h, M3 1h, plus buffer)
**Dependencies**: None. Builds on `serveapi/protocol/mcphttp` (ailang #885,
`m-serveapi-sdk-free-mcp-dispatch`, shipped v0.48.0) and the frozen-callback-envelope contract
that package inherited.
**Issue**: Refs #1602 (sunholo-data/ailang#1602, `area:serveapi` `priority:P2`, open, filed by
the ailang-world mission). Do not open a duplicate; this doc is the issue's design deliverable.
Triage 2026-10-08 verified the defect live on origin/dev `658ff76a3`.

---

## Measurement provenance

Every code claim below was re-read in this worktree at **`1dfd5615` (branch
`coordinator/task-f8d32bed`, 2026-10-08)** — the error-path functions are byte-for-byte the ones
the issue measured at v0.47.2, v0.52.1 and origin/dev `b7028a7cb`, and re-read at the
2026-10-08 triage pin `658ff76a3`. Each claim has a row in the Verification Log with the exact
grep/read that produced it.

This doc makes no claim about AILANG language semantics — it covers a Go seam (`serveapi`)
and the MCP/JSON-RPC wire — so the `ailang check` hard gate does not apply. Every
negative-existence claim (the "no hook exists today" family) carries its own Verification Log
row per the design-doc-creator rules.

World-side facts come from the #1602 issue body and World PR #219 (ailang-world, row 136
"option C"), read from the GitHub API on 2026-10-08. The issue has **no comments** — only two
label events (`area:serveapi` 2026-10-06, `priority:P2` 2026-10-08) and three cross-references
(World PR #219, World PR #223, World issue #202). The issue body is the sole and complete
statement of the ask.

## Problem Statement

A host serving MCP through `serveapi/protocol/mcphttp` cannot return a typed JSON-RPC error
from a tool call. Every non-nil `Invoker.Invoke` error becomes `-32603 "host callback failed"`,
and there is no `isError` result. World needs to tell an MCP client "your effect ran, but the
commit failed; here are the effect-record refs" (`EffectsUnrecorded`). It can already do this
over A2A (the A2A handler passes the host's result JSON through verbatim, so World encodes the
typed error in the artifact). Over MCP the client sees only the generic error, so it may
retry blindly and run the effect **twice**.

**Current State (all verified at `1dfd5615`, see Verification Log):**

- `serveapi/protocol/interfaces.go:16–23`: `Invoker.Invoke` returns
  `(InvocationResult{Value json.RawMessage}, error)`. The result struct has **no** error,
  status or `isError` field — it is a one-line struct literal.
- `serveapi/protocol/envelope.go:21–32`: `CallbackMessage` maps exactly three sentinel errors
  (`ErrCallbackCapacity`, `context.DeadlineExceeded`, `context.Canceled`) to fixed strings;
  **every other error becomes `"host callback failed"`**.
- `serveapi/protocol/envelope.go:33–51`: `WriteMCPEnvelope` hard-codes `-32603` as the code.
  There is no hook, parameter or wrapper that can change it.
- `serveapi/protocol/mcphttp/handler.go:204–209` (`serveMessages`): a non-nil error from
  `dispatch` answers the **whole POST** — including every other message in a batch — with
  `WriteMCPEnvelope(w, id, CallbackMessage(hostErr))` (`:208`).
- `serveapi/protocol/mcphttp/methods.go:25–27` (`callResultJSON`): the tools/call result has
  only `content` and `structuredContent`; no `isError` is ever emitted on this surface
  (`isError` appears in `serveapi/` only inside `BearerGate.ChallengeResult`,
  `bearer_gate.go:147–175`, a different code path that answers before dispatch).
- `serveapi/embedded_mcp_test.go:283` `TestEmbeddedMCPFrozenCallbackEnvelopes` freezes this
  envelope shape by test.

**Impact:** every embedder of `serveapi` (in-repo: the facade's test hosts; externally:
ailang-world, which consumes **pinned releases only**) loses host error semantics at the MCP
boundary. The failure is not cosmetic: World's MCP clients cannot distinguish "retryable
transport failure" from "effect committed, bookkeeping failed", so the safe behavior for them
is to retry — which is the unsafe behavior for an effectful tool. World is blocked on this
hook (row 136, World PR #219 M3 documents the operator-line workaround and the gated follow-up
row that will switch to the typed mapping once a release carries the hook).

## Goals

**Primary Goal:** An `Invoker.Invoke` error that opts in can carry a JSON-RPC error code and
message to the MCP client as a **per-message** JSON-RPC error, while every untyped error keeps
today's frozen whole-POST `-32603` envelope byte-for-byte.

**Success Metrics:**

1. A host error implementing the hook is answered with the host's exact code and message at
   HTTP 200, as a JSON-RPC error object for that message's id (golden-tested).
2. In a batch, the other messages still receive their results; only the failed message
   becomes an error object.
3. `TestEmbeddedMCPFrozenCallbackEnvelopes` passes **unchanged** (timeout/capacity paths and
   the frozen fallback are untouched).
4. Zero facade changes: the typed error survives `serveapi.New`'s adapter closures and
   `hostcall.Runner` unwrapped (V8/V10), so a host that implements the hook needs no other
   change and no re-compilation of anything but its own error type.
5. `make check-protocol-closure` stays green: `serveapi/protocol` gains no non-stdlib import
   (the interface is declared in `interfaces.go`; `mcphttp` adds only stdlib `errors`).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Hook shape: optional Go interface on the **error** returned by `Invoke` (issue's preferred option (a)), not an `InvocationResult.IsError` field (option (b)) | Decides the whole seam; option (b) would add a second error channel to the same wire and World's charter case needs schema-free transport of the refs | human (World's stated preference in #1602; frozen here) | design | low |
| Typed errors answer **per-message** at HTTP 200 via the existing `errorResponse`/`writeSSE` path — the same path "unknown tool" already uses — not via the whole-POST envelope | Changes batch semantics and media type for the typed case only; untyped keeps the frozen contract | human (issue asks exactly this) | design | med |
| The frozen `-32603` envelope remains the fallback for every error that does not implement the hook | A frozen, test-pinned public contract (`TestEmbeddedMCPFrozenCallbackEnvelopes`); silent change would break the pin and any client keyed to it | human | design | high |
| Hook validation rule: the typed path fires only when the hook yields `code != 0 && message != ""`; otherwise the error is treated as untyped (frozen fallback) | A half-implemented hook must not leak `code: 0` to clients (no silent fallbacks: a malformed hook degrades to the documented frozen behavior, and the rule is documented + tested) | agent | compile | low |
| Scope: `tools/call` (the `Invoker` seam) only. `ResolveSession`/`Tools` callback errors, the A2A handler, and BearerGate refusals keep current behavior | `Invoke` is the only seam the ask names; A2A already gives World typed errors via result pass-through | human (issue scope) | design | low |
| Interface lives in `serveapi/protocol` (exported, named), consumed via `errors.As` in `mcphttp` | `protocol` is the published contract package hosts already implement (`protocol.Invoker`); an unexported interface in `mcphttp` would be unreachable for hosts | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Hook shape: error-implemented optional interface `protocol.JSONRPCError` (option (a)),
      per World's stated preference in #1602.
- [x] Typed errors are per-message; untyped errors keep the frozen whole-POST envelope.
- [x] Frozen fallback, `CallbackMessage` sentinel mapping, and `WriteMCPEnvelope` are
      unchanged.
- [x] Scope is `tools/call`/`Invoker` only (not resolve/tools callbacks, not A2A, not the gate).
- [x] `code != 0 && message != ""` guard (a zero-valued or empty hook result is treated as
      untyped).

## Solution Design

### Overview

Add one small exported interface to `serveapi/protocol` and one check in
`mcphttp.callTool`. A host that wants typed MCP errors returns, from `Invoke`, an error that
also implements:

```go
// JSONRPCError may be implemented by an error returned from Invoker.Invoke.
// When it is, the MCP surface answers the calling message — not the whole
// POST — with a JSON-RPC error object carrying exactly this code and message
// (ailang#1602). Errors that do not implement it keep the frozen
// -32603 "host callback failed" envelope, byte for byte.
//
// code must be non-zero and message non-empty for the hook to fire; anything
// else is treated as an untyped error. Hosts should use the JSON-RPC 2.0
// server-error range -32000..-32099 or application-defined codes; the
// protocol-reserved range -32700..-32600 is passed through verbatim at the
// host's own risk.
type JSONRPCError interface {
	error
	JSONRPCError() (code int, message string)
}
```

Structural typing means a host needs no import change to satisfy it (though importing
`protocol` for the name is natural — hosts already do, for `Invoker`). The check follows the
house pattern of `protocol.AuthorizationStatus` (`envelope.go:52`), the existing optional
interface on this same error path.

### Architecture

**Components:**

1. **`protocol.JSONRPCError`** (`serveapi/protocol/interfaces.go`, next to `Invoker`): the
   contract. No new imports; the package's stdlib-only closure guarantee is untouched.
2. **`mcphttp.callTool`** (`serveapi/protocol/mcphttp/methods.go`): after `hostcall.Run`
   returns an Invoke error, try `errors.As(err, &coded)`. On a hit with
   `code != 0 && message != ""`, return `errorResponse(msg.ID, code, "%s", message), nil` —
   i.e. convert to a per-message response and report **no** host-level error to
   `serveMessages`. On a miss, return `response{}, err` exactly as today, so
   `serveMessages` writes the frozen envelope for the whole POST.
3. **`serveMessages`** (`handler.go:185`): **unchanged.** Its existing loop already collects
   `response` values (results and error objects alike) into the batch array / single SSE
   event; the typed error simply enters as one more error-shaped response, the same way
   `errorResponse(codeInvalidParams, "unknown tool %q")` does today (`methods.go:83`).

```go
// methods.go, in callTool, replacing `if err != nil { return response{}, err }`:
	if err != nil {
		var coded protocol.JSONRPCError
		if errors.As(err, &coded) {
			if code, message := coded.JSONRPCError(); code != 0 && message != "" {
				return errorResponse(msg.ID, code, "%s", message), nil
			}
		}
		return response{}, err
	}
```

Why this path is safe and already proven: dispatch-level error responses already ship at
HTTP 200 through `writeSSE` (single message: one SSE `event: message`; batch: one SSE event
whose data is the JSON array). MCP clients must Accept both media types before dispatch
rejects the POST (`acceptsBoth`, `wire.go:106`), so the media-type change
from the frozen `application/json` envelope to `text/event-stream` for typed errors is within
the contract every accepted client already handles — it is the same media type a successful
tools/call gets. A second improvement falls out: the frozen envelope echoes the POST-level
`id` (the first request id in the body) for every failure, so in a batch a host error could
answer the wrong id; the typed path answers with the **message's own** `msg.ID`.

**Timeouts cannot be hijacked by accident.** `hostcall.Run` returns the callback's error
**unwrapped** (`runner.go:63`) and, on its own deadline, returns `callCtx.Err()` — the
context package's error, which implements no hook. So `DeadlineExceeded`, `Canceled` and
`ErrCallbackCapacity` keep their frozen mapping unless a host *deliberately* wraps them with
a hook implementation, which is then the host's explicit, documented choice.

**Facade pass-through (zero changes).** `serveapi.New` adapts `cfg.Invoker` into the embedded
handlers with closures that return the error unchanged (`serveapi.go:93–95`, `:106–109` →
`mcp_handler.go:57–60`). No `fmt.Errorf` wrap, no mutation: a typed error reaches `mcphttp`
intact, so M2's facade-level test needs no production change to pass.

### Implementation Plan

**Phase 1 — the hook** (~3 hours)
- [ ] Declare `JSONRPCError` in `serveapi/protocol/interfaces.go` with the doc comment above.
- [ ] Add the `errors.As` branch in `mcphttp.callTool` (`methods.go`).
- [ ] Unit tests in `serveapi/protocol/mcphttp/` (new file `typed_error_test.go` or extend
      `handler_test.go`): single POST → typed error; code/message pass-through verbatim;
      `code: 0` and `message: ""` fall back to frozen; plain `errors.New` stays frozen;
      host error wrapping a typed error via `fmt.Errorf("%w")` still resolves (errors.As).
- [ ] `make test serveapi/...` and `make check-protocol-closure` green.

**Phase 2 — batch, wire goldens, facade** (~2 hours)
- [ ] Golden batch test: `[tools/call ok, tools/call typed-error, ping]` → 200 SSE array with
      one result, one host-coded error object (id-matched), one result.
- [ ] Regression: `TestEmbeddedMCPFrozenCallbackEnvelopes` (embedded facade) passes unchanged;
      add one facade-level case there: a typed error through `serveapi.New` surfaces per-message.
- [ ] Confirm the `parity_test.go` differential still passes. Note: it cannot and should not
      grow a parity row for error mapping — its SDK reference **discards** the Invoke error
      (`result, _ := host.Invoke(...)`, `parity_test.go:38`), so the SDK has no comparable
      behavior; the wire shape is defined by JSON-RPC 2.0 §5.1 and pinned by our goldens (V11).

**Phase 3 — docs and changelog** (~1 hour)
- [ ] `docs/docs/guides/serve-api.md` §"Embedding MCP and A2A in a Go host": document the
      hook (host returns an error implementing `JSONRPCError`), the `code != 0 && message != ""`
      guard, the per-message semantics, and the unchanged frozen fallback. While in that
      section: fix the stale sentence "its MCP handler brings the MCP SDK dependency subtree"
      (false since #885 — V12), which is in the same paragraph the new text extends.
- [ ] CHANGELOG entry under `[Unreleased]`/`v0.52.6` ("Added — serve-api: typed host errors
      over MCP").

### Files to Modify/Create

**New files:**
- `serveapi/protocol/mcphttp/typed_error_test.go` (~100 LOC) — hook unit tests + wire goldens
  (or equivalent tests folded into `handler_test.go`; agent's choice, see Deferred).

**Modified files:**
- `serveapi/protocol/interfaces.go` (+~15 LOC) — the `JSONRPCError` interface and doc comment.
- `serveapi/protocol/mcphttp/methods.go` (+~10 LOC) — the `errors.As` branch (plus stdlib
  `errors` import).
- `serveapi/embedded_mcp_test.go` (+~30 LOC) — one facade-level typed-error case next to the
  frozen-envelope test.
- `docs/docs/guides/serve-api.md` (+~15 / −1 LOC) — hook documentation; stale SDK sentence fix.
- `changelogs/v0.32-current.md` (+~6 LOC) — changelog entry.

**Unchanged on purpose:** `envelope.go` (`CallbackMessage`, `WriteMCPEnvelope`), `handler.go`
(`serveMessages`), `serveapi.go` / `mcp_handler.go` adapters, `bearer_gate.go`, A2A path.

## Examples

### Example 1: World's EffectsUnrecorded (the charter case)

**Before** — host returns a typed error; client sees nothing of it:

```json
POST /mcp/  {"jsonrpc":"2.0","id":41,"method":"tools/call","params":{"name":"commit","arguments":{}}}
→ HTTP 200, Content-Type: application/json
  {"jsonrpc":"2.0","id":41,"error":{"code":-32603,"message":"host callback failed"}}
```

The client cannot tell "transport hiccup" from "effect ran, commit failed" → it retries →
the effect runs twice. Today World logs the effect-record refs on its operator line only.

**After** — the host's error type:

```go
type effectsUnrecorded struct{ refs []string }

func (e effectsUnrecorded) Error() string { return "effects unrecorded: commit failed" }
func (e effectsUnrecorded) JSONRPCError() (int, string) {
	return -32002, fmt.Sprintf("effects unrecorded; commit failed; refs=%v", e.refs)
}
```

```json
→ HTTP 200, Content-Type: text/event-stream
  event: message
  data: {"jsonrpc":"2.0","id":41,"error":{"code":-32002,"message":"effects unrecorded; commit failed; refs=[er-1 er-2]"}}
```

The client machine-branches on `-32002` (World's follow-up row maps `dispatchError` codes onto
this hook once a release carries it) and does **not** retry.

### Example 2: a batch keeps its survivors

```json
POST /mcp/  (2025-03-26)
  [{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"echo"}},
   {"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"commit"}},
   {"jsonrpc":"2.0","id":3,"method":"ping"}]
```

**Before:** the whole POST answers `{"jsonrpc":"2.0","id":1,"error":{"code":-32603,"message":"host callback failed"}}` — message 1's successful result is thrown away and the failure answers with the wrong id.

**After:** HTTP 200, one SSE event whose data is the JSON array:

```json
[{"jsonrpc":"2.0","id":1,"result":{"content":[...],"structuredContent":{...}}},
 {"jsonrpc":"2.0","id":2,"error":{"code":-32002,"message":"effects unrecorded; ..."}},
 {"jsonrpc":"2.0","id":3,"result":{}}]
```

### Example 3: an untyped error is untouched

A host that returns plain `errors.New("db down")` (no hook), and every timeout/capacity/cancel
error, still gets exactly today's frozen envelope: `-32603 "host callback failed"` /
`"host callback timed out"` / `"host callback capacity exceeded"`, whole POST,
`TestEmbeddedMCPFrozenCallbackEnvelopes` unmodified.

## Success Criteria

- [ ] A host error implementing `protocol.JSONRPCError` answers the calling message with that
      exact code and message, HTTP 200 (golden test).
- [ ] In a batch, non-failing sibling messages keep their results; the typed error is
      id-matched to its own message (golden test).
- [ ] `code: 0`, empty message, and non-hook errors all keep the frozen `-32603` envelope;
      timeout/capacity/cancel mappings unchanged (unit tests + existing frozen tests).
- [ ] Facade proof: the typed error passes through `serveapi.New` unchanged
      (one case in `embedded_mcp_test.go`).
- [ ] `make check-protocol-closure` green; `serveapi/protocol` gains no non-stdlib import.
- [ ] All tests passing (`make test`), `make lint` clean.
- [ ] Documentation updated (`docs/docs/guides/serve-api.md` §Embedding) and changelog entry
      present.
- [ ] Issue #1602's ask satisfied per its own text: "An Invoker error that implements an
      optional interface … is answered as a per-message JSON-RPC error with that code and
      message. Other messages in a batch keep their results. CallbackMessage keeps its frozen
      fallback for every other error." (Delivery to World is the **release tag** — World pins
      releases only, so the changelog/version bump is part of done.)

## Testing Strategy

**Unit tests** (`serveapi/protocol/mcphttp/typed_error_test.go` or `handler_test.go`):
- code+message pass-through verbatim; id-matching for single and batch POSTs
- guard matrix: hook with `code: 0`; hook with empty message; plain error; `fmt.Errorf("%w")`-
  wrapped typed error (resolves); wrapped *untyped* error (stays frozen)
- typed error under every supported protocol version (the hook is version-independent)

**Integration/regression tests:**
- `serveapi/embedded_mcp_test.go`: facade-level typed error case;
  `TestEmbeddedMCPFrozenCallbackEnvelopes` untouched and green
- `serveapi/protocol/mcphttp/parity_test.go`: full differential still green (no new rows —
  see V11 for why the SDK reference cannot express error mapping)
- wire-table rows in `handler_test.go` style pinning the two new response shapes

**Manual testing:**
- Not required beyond the goldens; optionally drive one curl against a scratch host
  implementing the hook to eyeball the SSE frame.

## Verification Log

Re-derived in this worktree at `1dfd5615` (2026-10-08); the issue's own measurements at
v0.47.2 / v0.52.1 / `b7028a7cb` and the triage re-verification at `658ff76a3` agree on every
mechanism claim.

| # | Claim (incl. negative-existence claims) | Evidence at `1dfd5615` |
|---|---|---|
| V1 | `InvocationResult` has no error/status/isError field | `interfaces.go:23` — `type InvocationResult struct{ Value json.RawMessage }` (read) |
| V2 | `isError` appears in `serveapi/` only in `bearer_gate.go` (+ its test) | `grep -rn "isError" serveapi/` → `bearer_gate.go:147,158,169`, `bearer_gate_call_test.go:30` only |
| V3 | No typed-error hook exists anywhere in the surface today | `grep -rn "JSONRPCError\|JSONRPCCode" --include=*.go serveapi/ internal/ cmd/` → empty |
| V4 | The frozen envelope code is fixed at `-32603` | `envelope.go:50` — literal `{Code: -32603, Message: message}`; no parameter reaches it |
| V5 | `CallbackMessage` maps only the three sentinels; everything else is `"host callback failed"` | `envelope.go:21–32` (read) |
| V6 | A dispatch host error answers the whole POST, batch included | `handler.go:204–209` — `protocol.WriteMCPEnvelope(w, id, protocol.CallbackMessage(hostErr)); return` inside the `for _, raw := range raws` loop (`:187`) |
| V7 | The frozen envelope is pinned by test | `embedded_mcp_test.go:283–323` `TestEmbeddedMCPFrozenCallbackEnvelopes` asserts code `-32603`, message `"host callback timed out"`, `Content-Type: application/json`, HTTP 200 for resolve/tools/invoke stages |
| V8 | `hostcall.Run` returns the callback's error unwrapped; its own timeout returns `callCtx.Err()` | `runner.go:63` (`return completed.value, completed.err`), `:65` (`return zero, callCtx.Err()`) — no `fmt.Errorf` wrap on either path, so `errors.As` sees host types and can never misfire on runner timeouts |
| V9 | Per-message error responses already ship at HTTP 200 via `writeSSE` | `methods.go:83` returns `errorResponse(..., codeInvalidParams, "unknown tool %q")` with nil error; `handler.go:210–216` collects it and `writeSSE` answers — the mechanism this design reuses |
| V10 | The serveapi facade passes `Invoke` errors through unchanged | `serveapi.go:93–95` / `:106–109` return `result.Value, err`; `mcp_handler.go:57–60` returns `InvocationResult{Value: value}, err` — no wrap, no facade change needed |
| V11 | The parity differential cannot cover host-error mapping | `parity_test.go:38` — the SDK reference calls `host.Invoke` as `result, _ := host.Invoke(...)`, discarding the error; no SDK behavior exists to mirror. Wire shape is instead defined by JSON-RPC 2.0 §5.1 and pinned by in-repo goldens |
| V12 | `docs/docs/guides/serve-api.md`'s "its MCP handler brings the MCP SDK dependency subtree" is stale | `grep -rn "modelcontextprotocol" serveapi/ --include=*.go \| grep -v _test` → empty; #885 (`m-serveapi-sdk-free-mcp-dispatch`, shipped v0.48.0) removed the SDK from the build closure, measured by `make check-protocol-closure` |
| V13 | Issue #1602 has no comments; body is the complete ask | GitHub API `issues/1602` → `comments: 0`; timeline → 2 `labeled` + 3 `cross-referenced` events only |
| V14 | World prefers the error-hook option and forbids local workarounds | #1602 body: "(preferred) An Invoker error that implements an optional interface…" and "World will not work around this locally… its charter forbids vendored forks"; World PR #219: "a gated follow-up row covers that [switch to typed mapping once a release carries the hook]" |

## Wire behavior compatibility (regression surface)

Not a parser/typechecker change, so no syntactic Conflict Surface applies. The behavioral
surface this extends, and the fixtures that must still pass:

- Existing valid answers that MUST still work: `handler_test.go` wire table (incl. W6 batch
  rows), `gate_test.go` (BearerGate refusals answer **before** dispatch and are untouched),
  `parity_test.go` differential, `tool_hints_test.go`, `embedded_mcp_test.go` (frozen
  envelopes + facade), `bearer_gate_call_test.go` (ChallengeResult's `isError` body is a
  pre-dispatch HTTP 401 body, not a dispatch response — unaffected).
- What deliberately changes: **only** an Invoke error that opts in via the hook. For every
  other input the response bytes are identical to today (same code, same message, same media
  type, same whole-POST scoping). No previously-valid wire response changes shape.
- Medium-type note: for a typed error the answer moves from the frozen
  `application/json` envelope to `text/event-stream` (the media type of every successful
  tools/call today). This is within the contract the handler already enforces: it rejects any
  POST that does not Accept both (`acceptsBoth`, `wire.go:108`).

## Deferred Decisions

The following are intentionally left open for the implementer:

- Test file placement (`typed_error_test.go` vs extending `handler_test.go`) — agent may choose.
- Whether to log (stderr) when a hook yields `code: 0` / empty message before falling back —
  agent may choose; if added, it must not be a silent behavioral change on the wire.
- Interface doc-comment wording beyond the normative rules frozen above — agent may choose.
- Whether the changelog entry lands under `[Unreleased]` or a `v0.52.6` heading, per the
  release state when the sprint runs — agent may choose.

## Non-Goals

**Not attempted in this feature:**

- **`InvocationResult.IsError` (issue option (b))** — not built: World's stated preference is
  (a), (a) alone unblocks row 136, and a second error channel on the same wire doubles the
  contract for every embedder with no second consumer. If a future consumer needs
  "tool ran, result is a failure" semantics, that is a separate doc.
- **JSON-RPC `error.data`** — the `rpcError` struct has no `data` field; the ask is code+message.
  Extending the hook to carry `data` is additive later if asked.
- **Applying the hook to `ResolveSession`/`Tools` callback errors** — those answer before
  dispatch with the frozen envelope; the issue asks only for `Invoke`.
- **Applying the hook on the A2A surface** — World already projects typed errors over A2A via
  result pass-through; changing `writeCallbackError` would alter A2A wire behavior with no
  consumer asking.
- **`_meta` / idempotency keys reaching the host (World row 155)** — explicitly excluded by the
  issue ("It is not part of this ask").
- **Any change to `CallbackMessage`, `WriteMCPEnvelope`, or the frozen fallback** — pinned
  contract.

## Timeline

**Week 1** (~6 hours + buffer):
- Phase 1: hook + unit tests (3h)
- Phase 2: batch goldens + facade case + parity run (2h)
- Phase 3: docs + changelog (1h)

**Total: ~6 hours (estimate ×2 buffer already applied: 1–1.5 days wall-clock).**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| A mis-implemented hook (code 0 / empty message) leaks a broken error object to clients | Med | The `code != 0 && message != ""` guard routes malformed hooks to the frozen fallback; guard is unit-tested and documented |
| A host wraps `DeadlineExceeded` in a hooked error, hiding the timeout mapping | Low | `hostcall.Run`'s own timeout returns the context error (V8), so this requires a deliberate host choice; documented as the host's responsibility |
| Clients keyed to "every tools/call failure is -32603" mis-handle host codes | Low | Opt-in only: behavior is identical until a host returns a hooked error; World is the only known consumer and its follow-up row is written against this hook's shape |
| SSE media type for typed errors surprises a hand-rolled client | Low | `acceptsBoth` (`wire.go:106`) already rejects clients that don't Accept SSE before dispatch; same media type as successful calls |
| Doc drift hides the new hook from embedders | Med | serve-api.md §Embedding update is a Phase 3 deliverable, and the stale SDK sentence in the same paragraph is fixed with it (V12) |

## Related Documents

<!-- Auto-populated by search on "mcphttp typed host errors" (2026-10-08): SimHash and neural
     (fallback-simhash; Ollama embeddings unavailable) returned no semantic matches — top
     hits were token-overlap only (dx-17 TList unification, m-dx27 docs-search) and are
     unrelated. Related docs below were found by reading the ancestry of the touched code. -->

**Implemented (may inform design):**
- `design_docs/planned/v0_48_0/m-serveapi-sdk-free-mcp-dispatch.md` — created this handler
  (#885); its wire-contract, golden-test and closure-gate patterns are followed here
- `design_docs/planned/v0_51_0/m-serveapi-directory-ready.md` — BearerGate, the other `isError`
  user on this surface, and the pre-dispatch refusal path this design must not touch

**Planned (check for overlap):**
- `design_docs/planned/ailang-core-triage/serveapi-protocol-mcp-dispatch.md` — the #885 triage
  row; superseded by the shipped handler, kept as provenance

**External:**
- ailang-world `design_docs/planned/w-mcp-effects-unrecorded-is-untyped.md` (row 136) and
  World PR #219 — the consumer charter, constraints and the gated follow-up row

## References

- Issue #1602 — the ask this doc answers (Refs #1602)
- [JSON-RPC 2.0 specification, §5.1 Error object](https://www.jsonrpc.org/specification#error_object)
- [MCP specification — tools/call](https://modelcontextprotocol.io/specification/2025-06-18/server/tools)
- [OpenAI Apps SDK — Authentication](https://developers.openai.com/apps-sdk/build/auth) — the
  `isError` + `_meta` precedent on this surface (`bearer_gate.go:147–175`)
- [Design Axioms](/docs/references/axioms) — scoring below

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | The reply is a pure function of (request bytes, host callbacks); the typed code/message are host values pinned by golden fixtures, and the frozen fallback path is unchanged and still test-pinned |
| A2: Replayability | 0 | Stateless POST semantics unchanged |
| A3: Effect Legibility | 0 | No AILANG effect surface touched; host-side effect semantics are World's |
| A4: Explicit Authority | 0 | No capability or ambient-access change; the hook is host-opt-in |
| A5: Bounded Verification | +1 | New wire shapes are golden-tested in-repo; `make check-protocol-closure` keeps the stdlib-only guarantee |
| A6: Safe Concurrency | 0 | No new shared state; runs inside the existing bounded `hostcall.Runner` |
| A7: Machines First | +1 | The client can machine-branch on a typed code instead of parsing `"host callback failed"`; removes the blind-retry path that double-runs effects |
| A8: Minimal Syntax | 0 | No language change; one small Go interface |
| A9: Cost Visibility | 0 | No cost surface |
| A10: Composability | +1 | The interface lives at the `protocol` seam hosts already implement (`Invoker`), transport-neutral, so A2A or other transports can adopt it later without redesign |
| A11: Structured Failure | +1 | The boundary currently destroys host error structure (everything collapses to -32603); this restores typed, structured failure at the system boundary |
| A12: System Boundary | +1 | Host intent now crosses the MCP boundary explicitly, via a documented contract, instead of being silently erased |

**Net Score: +5** → **Proceed.** Hard-violation check: no −1 anywhere; A1/A3/A4/A7 all ≥ 0.

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced (typed values are host-decided and golden-pinned)
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): the change exists *for* machine clients (typed codes over prose)

## Future Work

- **JSON-RPC `error.data`** on the hook (e.g. `JSONRPCErrorData() json.RawMessage`), if a
  consumer needs structured error payloads.
- **Applying the hook to the A2A surface** (`writeCallbackError`) if an A2A embedder ever
  asks for code parity across transports.
- **`isError` tool results via `InvocationResult`** (issue option (b)) if a consumer needs
  MCP-spec "tool ran but failed" semantics rather than a protocol error.
- **World row 155** (`_meta` / idempotency keys reaching the host) — separate upstream ask,
  already excluded by #1602.

---

**Document created**: 2026-10-08
**Last updated**: 2026-10-08

## Maintainer rulings (Ruled 2026-10-08 by Mark)

The decision marked `human` is ratified as written: the frozen `-32603` envelope remains the fallback for every error that does not implement the hook.
