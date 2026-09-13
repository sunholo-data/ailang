# M-CLI-MSG-HANDLER: CLI Runtime Msg Effect Handler

**Status**: Planned
**Target**: v0.38.6
**Priority**: P1 (Medium) — unblocks std/cognition messaging from the primary CLI surface
**Estimated**: 2–3 days
**Dependencies**: None new — consumes the existing `MsgHandler` interface (internal/effects/msg.go, v0.21.x) and the existing message-plane store selection (`cmd/ailang/messages.go`)

> **Scope note**: This is a **runtime capability surface** design — it wires an *existing* effect interface to an *existing* message plane. It changes no language syntax, no effect-row algebra, and no message-store semantics.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Recv is a deterministic poll-once (oldest-unread-first), Subscribe is a drain-once — no poll-timing nondeterminism smuggled into batch runs |
| A2: Replayability | 0 | Traces unchanged; send/recv ops already emit effect spans |
| A3: Effect Legibility | +1 | Msg ops stay behind the declared `! {Msg}` row; nothing becomes ambient |
| A4: Explicit Authority | +1 | Handler binds only when the Msg capability is granted; an unreachable store fails loud at startup instead of degrading to an empty inbox |
| A5: Bounded Verification | 0 | No type-system change |
| A6: Safe Concurrency | 0 | Single-goroutine batch evaluator; store clients are concurrency-safe |
| A7: Machines First | +1 | Reproduces the exact error contract (`ErrNoMsgHandler`, typed Result codes) agents already see; one honest semantic table for all three transports |
| A8: Minimal Syntax | +1 | No new syntax; `--caps Msg` already parses |
| A9: Cost Visibility | +1 | `budget_remaining` in SendResult surfaces plane costs per send |
| A10: Composability | +1 | Third MsgHandler implementation behind the same interface (Stub / Wasm / Store) |
| A11: Structured Failure | +1 | Startup store failure = typed CLI error + non-zero exit; runtime failures = `{code, message}` via `sendMsgResult`/`recvMsgResult` |
| A12: System Boundary | +1 | Crossing into the shared message plane is an explicit, capability-gated boundary |

**Net Score: +10** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced (recv/subscribe are poll-once, not racy waits)
- [x] A3 (Effects): no hidden side effects (store writes happen only under `! {Msg}` ops)
- [x] A4 (Authority): no ambient access (handler exists only when `--caps Msg` / `auto` grants it)
- [x] A7 (Machines First): honest batch semantics documented once, not discovered per-run

## Problem Statement

GitHub issue #1130 (daneel, measured on v0.36.0): `ailang run --caps IO,Msg prog.ail` fails at the first `sendMsg`/`recvMsg` with:

```
no Msg handler configured — set one via cmd/wasm/effects.go (browser) or configure a StubMsgHandler for tests
```

`--caps auto` infers `Msg` from the entrypoint's effect row but hits the same wall.

**Verified mechanism** (not inferred): `grantCapabilities` (cmd/ailang/run_helpers.go:142) grants the `Msg` capability on both run paths — the single-shot path (cmd/ailang/main_run_exec.go:405-440) and the batch path (cmd/ailang/run_helpers.go:665) — but neither path constructs a handler. The handler-setup block wires `SharedMem`, `SharedIndex`, `Net`, `Stream`, `Process`, and `AI`; a grep for `SetMsgHandler|MsgHandler` across `cmd/ailang/` returns nothing. `effCtx.Msg` stays nil, so every `MsgContext.Send/Recv/Subscribe` nil-check fails with `ErrNoMsgHandler` (internal/effects/msg.go:34-40). This is exactly the gap the umbrella design left open: M-COG-RUNTIME's Phase 5 listed "`msg_native.go` — native handler routes to existing `internal/messaging/store.go`" and `NewMsgContext`'s own doc says the CLI wrapper sets it "when the CLI Msg API ships in a later sprint". That sprint never happened.

**Impact**: std/cognition messaging (`sendMsg`/`recvMsg`/`subscribeMsg`) is reachable only from the browser WASM build. Agent programs that reason about the shared message plane — the substrate the whole mission fleet runs on — cannot exercise it from the standard `ailang run` surface, and the error text tells them to fix a Go file.

## Goals

**Primary Goal:** Bind a store-backed `MsgHandler` in the CLI runtime so `Msg`-effect programs run under normal `ailang run`, capability-gated, against the same message plane `ailang messages` uses.

**Success Metrics:**
1. `ailang run --caps IO,Msg send.ail` produces a message visible in `ailang messages list` on the same store (local or gcp).
2. `ailang run --caps Msg recv.ail` pulls that message deterministically; empty inbox yields the typed `no_message` error, not a hang or a silent empty result.
3. Unreachable store (e.g. `AILANG_MESSAGES_STORE=gcp` without credentials) fails at startup with a loud, named error and non-zero exit — never an empty-inbox success.
4. `ailang messages` CLI behavior remains byte-identical (no edits under `internal/messaging/`).
5. All existing tests pass; new wiring covered by unit + integration tests.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| CLI binds the **store backend** (same plane as `ailang messages`), not a stub | Determines whether agent programs see the real inbox or a toy | human (this doc) | design | high |
| `recvMsg` is **poll-once, non-blocking** in batch runs | Defines the honest contract for a batch evaluator; blocking would hang every empty-inbox run | human (this doc) | design | med |
| `subscribeMsg` is **drain-once, no live streaming** in batch runs | Long-lived listeners cannot exist inside a batch-mode run; pretending otherwise breaks determinism | human (this doc) | design | med |
| Store failure fails **loud at startup** (before the program runs) | "No silent fallbacks" principle; an empty-inbox success would corrupt agent reasoning | human (this doc) | design | med |
| Handler binds **only when Msg capability is granted** (`--caps Msg` or `auto` inference) | `--caps` is the security boundary; a handler without the grant would be ambient access | compiler (existing semantics) | design | low |
| `clock` field = message `CreatedAt` Unix-millis (no Lamport clock on the store plane) | Envelope-shape honesty: store rows have no Lamport clock | agent | compile | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Store backend vs stub (resolved: store backend — this doc)
- [x] Batch recv/subscribe semantics (resolved: poll-once / drain-once — this doc, § Solution Design)
- [x] Loud startup failure on unreachable plane (resolved: yes — this doc)

## Solution Design

### Overview

Add a third `MsgHandler` implementation — `StoreMsgHandler` — that adapts the runtime `Msg` envelope shape (Mailbox / Message / SendResult) onto the `messaging.MessageStore` inbox surface. The CLI wires it in the existing handler-setup block, mirroring `setupSharedMemHandler`: construct only if the capability is granted, resolve the store through the **same** `messagesTarget()`/`openStore()` selection `ailang messages` uses (honoring `AILANG_MESSAGES_STORE` / `AILANG_MESSAGES_PROJECT`), and fail loud if the plane is unreachable.

The browser handler (`WasmMsgHandler`, cmd/wasm/effects_cognition.go:261-356) is the established pattern: a transport-specific adapter behind the uniform interface, host-selected at startup. This is the same pattern with the store as transport.

### Architecture

**Components:**

1. **`StoreMsgHandler`** (new, `cmd/ailang/msg_handler.go`, ~150 LOC) — implements `effects.MsgHandler`:
   - **Send(to, payload)** → `InsertInboxMessage` with `FromAgent: self`, `ToInbox: string(to)`, `MessageType: "msg"`, `Payload: string(payload)`. Returns `SendResult{MsgID: <store row ID>, Clock: <CreatedAt unix-millis>, BudgetRemaining: <from Msg budget, else 0>}`.
   - **Recv(mailbox)** → poll once, oldest unread first: `ListInboxMessages(InboxListOptions{Inbox: mailbox, UnreadOnly: true, Limit: 1})`; on hit, `MarkInboxMessageRead(id)` and map to `Message{ID, From: FromAgent, To: ToInbox, Payload, Clock: CreatedAt unix-millis}`. On empty → `ErrNoMsgAvailable` (the stub's established "does not block" contract, so the typed `no_message` code from `classifyMsgError` applies unchanged).
   - **Subscribe(mailbox, onMsg)** → drain-once: fire `onMsg` for each currently-unread message in oldest-first order, mark each read, then return a no-op cancel. No live listener is registered — a batch run has no event loop to serve one.
   - Field mapping is documented once at the top of the adapter: `Mailbox ≡ inbox name`, `NodeID ≡ from-agent string`, `clock ≡ CreatedAt unix-millis` (the store plane has no Lamport clock; the number is honest monotonic-ish ordering, and the doc comment says so).
2. **`setupMsgHandler(effCtx *effects.EffContext)`** (cmd/ailang/run_helpers.go, next to `setupSharedMemHandler`): if `effCtx.HasCap("Msg")` — resolve store via the `messagesTarget()`/`openStore()` logic (lift those two functions into a small shared helper or call them directly from `messages.go`); on any open error, return a loud error naming the offending env vars (`AILANG_MESSAGES_STORE`/`AILANG_MESSAGES_PROJECT`) and fail the run **before** the program executes. On success set `effCtx.Msg = effects.NewMsgContext(handler, self)` where `self` defaults to `cli_run` (stable string; a `--msg-self` flag is a Deferred Decision).
3. **Wiring** in both run paths: the handler-setup block of `main_run_exec.go` (~line 405) and the batch path of `run_helpers.go` (~line 665), inserted alongside the other `setup*Handler` calls. REPL parity (`ailang repl`) via the existing `REPL.SetMsgHandler` in the same phase.

**Capability gating:** the grant stays exactly where it is (`grantCapabilities` / `--caps auto` inference from the entrypoint's effect row). The handler is *constructed* only under the grant — so `--caps Msg` becomes sufficient (issue #1130 fixed), while a program calling Msg ops **without** the grant still fails at the capability check, unchanged. Inverse case — handler present but no grant — cannot occur, preserving `--caps` as the security boundary.

**Blocking/subscription honesty (the core semantic contract):** the evaluator is batch: one entrypoint invocation, then the process exits. A `Recv` that blocks would turn every empty inbox into a hung run; a live `Subscribe` has no goroutine lifetime to occupy. Therefore:
- `recvMsg` = single non-blocking poll (typed `no_message` when empty — agents branch on it with `recvMsgResult`).
- `subscribeMsg` = one synchronous drain of the mailbox's current unread messages, then done. Under gcp mode, live Pub/Sub pull (`messages watch --pubsub` machinery) is explicitly Future Work — it needs a bounded-timeout interaction model that batch runs don't have.
- These semantics are identical across local and gcp stores; only the store backend differs. `StubMsgHandler` already established "Recv does not block"; the CLI makes that the honest native contract too.

**Error surfaces (fail loud, never an empty inbox):**
| Failure | Surface |
|---|---|
| Store unreachable/misconfigured (bad mode, gcp without project/creds) | Startup error: `setupMsgHandler: <openStore error>` + exit ≠ 0, before program runs |
| Empty mailbox on `recvMsg` | `ErrNoMsgAvailable` → typed `{code: "no_message", ...}` from `recvMsgResult`; raw `recvMsg` returns the error (documented panic-path in std/cognition.ail unchanged) |
| Handler missing (only possible on surfaces that don't wire it, e.g. `serve` before parity) | Existing `ErrNoMsgHandler` text **updated** to name the CLI flag: "grant Msg via --caps Msg (CLI) or set a handler via cmd/wasm/effects.go (browser)" |

### Implementation Plan

**Phase 1: Handler + wiring** (~1 day)
- [ ] `cmd/ailang/msg_handler.go`: `StoreMsgHandler` + field-mapping doc comment + unit tests (in-memory SQLite store fixture)
- [ ] `setupMsgHandler` in `run_helpers.go`; wire into `main_run_exec.go` and batch path
- [ ] Update `ErrNoMsgHandler` text in `internal/effects/msg.go` (wording only; error identity preserved so `classifyMsgError` and existing tests hold)

**Phase 2: Parity + docs** (~1 day)
- [ ] REPL wiring (`ailang repl` → `SetMsgHandler(StoreMsgHandler, ...)`) and `serve_api.go` `--caps` path
- [ ] `std/cognition.ail` doc comments: batch-semantics note on `recvMsg`/`subscribeMsg`
- [ ] `cmd/ailang/help.go` + `--caps` help text: Msg row states "wired to the message plane selected by AILANG_MESSAGES_STORE"
- [ ] Example: `examples/` send/recv pair runnable against local store

**Phase 3: Tests + gates** (~0.5 day)
- [ ] Integration: send via `ailang run`, read via `ailang messages list`, recv via `ailang run` (round-trip on local store)
- [ ] Loud-failure tests: bad `AILANG_MESSAGES_STORE` mode, gcp without project → startup error, non-zero exit
- [ ] Regression: `ailang messages` byte-identical behavior snapshot (per M-COG-RUNTIME's equivalence gate)

### Files to Modify/Create

**New files:**
- `cmd/ailang/msg_handler.go` — StoreMsgHandler adapter, ~150 LOC
- `cmd/ailang/msg_handler_test.go` — unit tests, ~120 LOC
- `examples/msg_roundtrip.ail` (or pair) — runnable example, ~20 LOC

**Modified files:**
- `cmd/ailang/run_helpers.go` — `setupMsgHandler` + wiring in batch path, ~40 LOC
- `cmd/ailang/main_run_exec.go` — one wiring line in handler-setup block, ~5 LOC
- `internal/effects/msg.go` — `ErrNoMsgHandler` message text only, ~2 LOC
- `cmd/ailang/help.go` — Msg capability line, ~5 LOC
- `std/cognition.ail` — doc comments, ~10 LOC

## Examples

**Before (#1130, today):**
```
$ ailang run --caps IO,Msg send.ail
Error: no Msg handler configured — set one via cmd/wasm/effects.go (browser) or configure a StubMsgHandler for tests
```

**After:**
```
$ ailang run --caps IO,Msg send.ail        # AILANG_MESSAGES_STORE unset → local SQLite
{ msg_id: "a1b2c3", clock: 1765830000123, budget_remaining: 0 }
$ ailang messages list --unread
  ... (the message, same store)
$ ailang run --caps Msg recv.ail
{ msg_id: "a1b2c3", from: "cli_run", to: "verifier", payload: "...", clock: 1765830000123 }
$ AILANG_MESSAGES_STORE=bogus ailang run --caps Msg recv.ail
Error: setupMsgHandler: unknown message store mode "bogus" (valid: local, gcp, hybrid)
```

## Success Criteria

- [ ] `ailang run --caps IO,Msg` round-trips a message through the store (acceptance: Phase 3 integration test)
- [ ] `recvMsg` on empty inbox returns typed `no_message`, exits without hanging (acceptance: unit test with bounded test timeout)
- [ ] Unreachable plane fails at startup with named env vars, exit ≠ 0 (acceptance: loud-failure tests)
- [ ] `--caps` without Msg still denies Msg ops at the capability check (acceptance: existing capability-denial test extended to Msg)
- [ ] `ailang messages` CLI byte-identical (acceptance: regression snapshot)
- [ ] All tests passing (`make test`), docs/help updated

## Testing Strategy

**Unit tests:** field mapping (Mailbox↔inbox, Clock↔CreatedAt), poll-once ordering (oldest unread first), drain-once subscribe, read-marking side effects.

**Integration tests:** full round-trip via two `ailang run` invocations against a temp SQLite store; gcp-mode startup failure paths (mock/error injection on `NewClientForProject`).

**Manual testing:** local-store round-trip; gcp store with real project (read-only smoke: send to a scratch inbox, list, ack).

## Deferred Decisions

- **`--msg-self` flag for NodeID** (default `cli_run`) — agent may choose to add or omit; flag is additive.
- **`BudgetRemaining` wiring** to the Msg effect budget — agent; 0 is honest "no budget enforced" today.
- **Lifting `messagesTarget`/`openStore` into a shared helper** vs calling from `messages.go` — agent (they're already exported within package `main`).
- **Bounded blocking recv (`--msg-recv-timeout`)** — deferred until a consumer demonstrates need; poll-once is the shipped contract.

## Non-Goals

- **Live streaming subscriptions in batch runs** — impossible honestly in a batch evaluator; gcp Pub/Sub pull arrives via a future bounded-timeout interaction model.
- **Any change to `internal/messaging/` store semantics** — the plane is consumed, not modified (M-COG-RUNTIME's byte-identical CLI guarantee).
- **New Msg ops or envelope fields** — `sendMsg/recvMsg/subscribeMsg` + Result variants are frozen; a bytes-payload variant is M2-era scope.
- **Lamport clocks on the store plane** — the store has no clock; faking one is worse than documenting `CreatedAt`.

## Conflict Surface

This design touches `internal/effects/` (error-text only) and the `cmd/ailang` run paths — no parser/typechecker/codegen positions. Enumerating the runtime positions it *does* occupy:

| Position | Existing occupants | Interaction |
|---|---|---|
| Handler-setup block (main_run_exec.go:405+, run_helpers.go:665+) | SharedMem, SharedIndex, Net, Stream, Process, AI setups | New `setupMsgHandler` is additive; ordering irrelevant (independent contexts) |
| `grantCapabilities` (`--caps` / `auto`) | All 15 capabilities; unknown-name = error, nothing granted (#1116) | Unchanged — grant logic untouched; handler is grant-*conditional* |
| `EffContext.Msg` | nil (CLI), `WasmMsgHandler` (browser), `StubMsgHandler` (tests) | CLI joins as third binder; interface untouched |
| `ErrNoMsgHandler` consumers | `classifyMsgError` (`CogErrCodeNoHandler`), msg_test.go assertions | Error *identity* preserved; only the text changes — tests asserting on the exact string need a one-line text update (audited: assertions match on the sentinel error, not the string) |
| `messagesTarget`/`openStore` | `ailang messages` subcommands | Read-only reuse; no behavior change to the CLI commands |

**Programs that MUST still work:**
1. `ailang messages list/send/read/ack` — byte-identical (regression snapshot)
2. Browser WASM cognition programs (`WasmMsgHandler` path untouched)
3. Any program under `--caps` without Msg (capability denial unchanged)
4. `std/cognition.ail` consumers using `sendMsgResult`/`recvMsgResult` (typed codes unchanged)
5. `ailang run` with no capabilities at all (handler never constructed)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Recv-as-poll surprises agents expecting blocking | Med | Typed `no_message` code + std/cognition doc comments; the stub already set this precedent |
| Subscribe drain marks messages read as a side effect | Med | Documented in the adapter + cognition docs; consistent with "consume" semantics |
| Store open cost on every Msg-granted run (gcp latency) | Low | Startup only; local default is SQLite |
| `ErrNoMsgHandler` text change breaks string-matching tests | Low | Audited: tests assert on the sentinel, not the text (see Verification Log) |

## Related Documents

- [M-COG-RUNTIME](../implemented/v0_21_0/m-cog-runtime.md) — implemented umbrella; explicitly deferred the native CLI Msg handler (its Phase 5 `msg_native.go` line is this doc)
- [M-COG-RUNTIME-BROWSER](../implemented/v0_21_0/m-cog-runtime-browser.md) — browser handler pattern this design mirrors
- GitHub issue #1130 — source report

## Verification Log

| Claim | Method | Result |
|---|---|---|
| CLI run paths grant Msg but wire no handler | grep `SetMsgHandler\|MsgHandler` in `cmd/ailang/` (non-test) | empty — confirmed |
| Handler-setup block covers SharedMem/SharedIndex/Net/Stream/Process/AI only | read main_run_exec.go:405-440, run_helpers.go:640-690 | confirmed |
| `internal/effects/` has no `msg_native.go` | `ls internal/effects/` | only msg.go, dom.go (+tests) — confirmed |
| Message-store selection honors `AILANG_MESSAGES_STORE`/`AILANG_MESSAGES_PROJECT` | read messagesTarget/openStore, cmd/ailang/messages.go:120-185 | confirmed (mode: STORE > AILANG_STORAGE > local; project: MESSAGES_PROJECT > CLOUD_PROJECT; unknown mode = error) |
| `ErrNoMsgHandler` is a sentinel compared by `classifyMsgError` | read msg.go:40, 450-460 | confirmed (switch on error identity) |
| StubMsgHandler Recv is non-blocking by contract | read msg.go:260-274 | confirmed (`ErrNoMsgAvailable`, "stub does not block") |
| `subscribeMsg` requires `Msg + Cog` and enqueues via `ctx.Cog` | read std/cognition.ail:79-80, msg.go:331-355 | confirmed |
| Store plane has no Lamport clock | read InboxMessage struct (inbox.go:17-46) | confirmed (CreatedAt only) |
| No existing/planned doc covers a CLI Msg handler | `ailang docs search` (SimHash) + grep design_docs for MsgHandler/recvMsg | none — confirmed; umbrella explicitly deferred it |

## References

- [Design Axioms](/docs/references/axioms)
- [internal/effects/msg.go](../../../internal/effects/msg.go) — MsgHandler interface, StubMsgHandler, ops
- [cmd/wasm/effects_cognition.go](../../../cmd/wasm/effects_cognition.go) — WasmMsgHandler (pattern)
- [cmd/ailang/messages.go](../../../cmd/ailang/messages.go) — messagesTarget/openStore (store selection)
- [internal/messaging/inbox.go](../../../internal/messaging/inbox.go) — InboxMessage, InboxListOptions

## Future Work

- Live Pub/Sub-backed subscribe for gcp mode (bounded-timeout interaction model)
- Long-lived `ailang serve` Msg sessions with streaming callbacks
- Bytes-payload send variant (M2 envelope extension)

---

**Document created**: 2026-09-13
**Last updated**: 2026-09-13
