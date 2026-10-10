# M-TERMINAL-UI-NATIVE-INPUT — Enable the existing terminal UI package

**Status**: Implementation on sprint branch — final validation and release/publication in progress
**Target**: AILANG v0.54.0 (proposed); `sunholo/terminal_ui` v0.2.0
**Priority**: P1 — reusable interactive CLIs built on shipped `[bin]` support
**Estimated**: 8–12 engineering days across core and package repositories; sprint planning must calibrate this estimate
**Dependencies**: shipped M-PKG-BIN-ENTRYPOINTS and M-TERMINAL-IO; M-IO-READLINE-EOF-OPTION for the line adapter's exact EOF contract
**Source**: `stapledons_godot` message `inbox_1791568104057_d08245d1` (2026-10-09); attended operator request (2026-10-10)
**Package evidence**: [terminal-ui at commit 145baa732e26e8a4cfe7464c55d9a258bc63dcab](https://github.com/sunholo-data/ailang-packages/tree/145baa732e26e8a4cfe7464c55d9a258bc63dcab/packages/terminal-ui)

## Problem Statement

AILANG packages can already ship executable commands through `[bin]`. An actual consumer now needs the next layer: a reusable package for attractive, keyboard-driven terminal CLIs without shell adapters or a second application host.

That package **already exists** in `sunholo-data/ailang-packages/packages/terminal-ui`: its manifest names `sunholo/terminal_ui`, version `0.1.0`, and a `terminal-ui-demo` command with `caps = "IO"`. Its public `ui.ail` provides sanitized text, conservative cell measurement, wrapping, padding, rules, gauges, bounded dimensions and paged ANSI/plain screens. Preserve this work and evolve the same package; do not create a competing `sunholo/tui` library. Repository presence is verified; registry publication is not established (keyword searches returned no matches).

The present IO demo hardcodes `dimensions(60, 24)`, reads `readLine()`, and requires Enter after `n/v/h/b/a/0`. The package's AGENT.md explicitly excludes raw mode, terminal-size detection, automatic resize, cursor hiding and an alternate screen. The missing capabilities belong to the host: a pure package cannot turn its line reader into raw key input or restore an OS terminal mode.

The old raw-input exclusion in M-AGENT-STEP-CANCELLATION and the maintained limitations page is an explicit policy decision, not evidence this API shipped. This proposal asks to revise that exclusion for a small, effect-gated native terminal adapter while keeping UI policy in a package. [#231](https://github.com/sunholo-data/ailang/issues/231) concerns AI-call cancellation and is adjacent, not the TUI ticket. No distinct matching TUI ticket was found during this investigation.

## Goals

1. The installed `terminal-ui-demo` supports arrow keys, paging and immediate quit without Enter, using current physical terminal dimensions and reacting to resize.
2. A consumer reuses `sunholo/terminal_ui` for screens, selection and confirmations; application state transitions remain pure AILANG functions.
3. Terminal state is restored on normal completion, runtime errors, `std/io.exit`, recoverable panics, SIGINT and SIGTERM; restoration is tested against a real pseudo-terminal.
4. Plain/line and recorded-event adapters make CLIs usable in pipes, automated agents and deterministic tests. Unsupported host behavior and activation failures are explicit.
5. Existing `ui` exports and `[bin]` installation keep their behavior. Core gains terminal primitives, not widgets, themes or a game loop.

## High-Impact Decisions

The user approved this direction with “great please sprint plan then execute”, and then authorized package publication with “we can publish the package as part of our sprint”. Target core release scheduling remains subject to the release workflow; implementation uses the frozen contracts below.

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|---|---|---|---|---|
| D1: Evolve existing `sunholo/terminal_ui` | Avoids a duplicate public package and preserves consumer investment | human; established by this session's clarification | design | high |
| D2: Add `std/terminal` operations under existing `IO`, limited to configured input/output endpoints | Revises raw-input exclusion and defines authority boundary | human; proposed | design | high |
| D3: Native adapter owns the terminal lease, mode changes, signals and cleanup; pure layout/navigation stays in the package | Cleanup cannot depend on an AILANG success path | human; proposed | design | high |
| D4: First native release supports macOS/Linux; Windows and WASM return typed unsupported for native sessions | Controls portability scope without claiming emulated support | human; proposed | design | medium |
| D5: Native, line and plain modes have explicit selection; auto resolves once and reports its selected mode | Avoids silently changing interaction after a native failure | agent; specified below | design | medium |
| D6: Keep `ui.ail` v0.1 behavior; add an adapter that respects actual small viewports | Existing minimum dimension clamp cannot describe a tiny physical terminal | agent; specified below | design | medium |

### Design Freeze

- [x] Reuse existing package and its `[bin]` demo.
- [ ] Approve the limited native IO surface and replacement of the raw-input exclusion (D2/D3).
- [ ] Approve the first-release platform scope (D4).
- [ ] Confirm callback/effect-row and strict-VM integration in M0 before freezing signatures.

## Solution Design

### Ownership and architecture

```text
consumer [bin] command
  ├── application model + pure update(event, model)
  └── sunholo/terminal_ui
       ├── existing ui: sanitize / wrap / page / render
       ├── events + widgets: pure selection / confirmation / paging
       └── adapter: native | line | recorded, explicit mode + dimensions
             ├── std/io: print / flush / readLineOpt (line only)
             └── std/terminal: facts / scoped lease / key+resize+EOF events
                   └── native host endpoints and restoration
```

The package repository owns package features and consumer examples. The AILANG repository owns the minimal host support, language-facing types, builtin registration and backend integration. No package is permitted to contain Go code or a Process-based escape hatch. Core must not import the package repository. No messaging-store, coordinator approval or `ailang messages` menu redesign is included; `[bin]` is the distribution prerequisite, not a reason to replace the Go CLI's UI.

### Proposed core API

`std/terminal` is an additive module. Operations use `! {IO}` and the existing capability/budget/trace dispatch (`effects.Call`). They operate on the host's configured input and active program output, honoring overrides such as `--emit-trace`; they never open `/dev/tty` to acquire different authority.

| Export | Proposed contract |
|---|---|
| `info()` | `Result[TerminalInfo, TerminalError] ! {IO}`; reports input/output TTY status, native-mode availability, and `Option[TerminalSize]` for measured size |
| `withTerminal(options, body)` | `Result[a, TerminalError] ! {IO, e}` with callback `TerminalSession -> a ! {e}`; scoped acquisition/restoration |
| `readEvent(session, timeout_ms)` | `Result[TerminalEvent, TerminalError] ! {IO}`; reads only within a live session |

Proposed value types:

- `TerminalSize = {columns: int, rows: int}`: physical positive dimensions, never the package's clamped layout dimensions.
- `TerminalInfo = {input_tty: bool, output_tty: bool, raw_supported: bool, size: Option[TerminalSize]}`. Lack of a TTY is ordinary data; an actual query failure is `Err`.
- `TerminalOptions = {alternate_screen: bool, hide_cursor: bool}`. Raw input is intrinsic to a native session. Options govern only trusted host escape sequences and are restored symmetrically.
- `TerminalSession`: a validated host handle bound to context and active lease. A constructor-shaped representation is not an authority proof: fabricated, stale, escaped and cross-context handles must fail validation.
- `TerminalKey`: `Text(string)`, Up/Down/Left/Right, Enter, Escape, Backspace, Tab, Home/End, PageUp/PageDown and Delete.
- `TerminalEvent`: `Key(TerminalKey)`, `Resize(TerminalSize)`, `EndOfInput`, `Interrupted`, `Idle`.
- `TerminalError`: typed Unsupported, NotTTY, Busy, InvalidSession, InvalidTimeout, InputFailure, QueryFailure and CleanupFailure cases. Final payload structure is frozen in M0; these are ADT cases, not newly allocated diagnostic codes.

Timeout `0` polls; `1..60000` bounds the wait; `-1` blocks until an event or host interruption; other values return InvalidTimeout. `Idle` is never EOF or an empty key. Each physical read/query consumes its normal IO budget; timeout behavior is an explicit host input, not a pure clock observation.

The syntax of a generic, effect-polymorphic callback and the proposed event ADTs was checked in a local type-shape stub on v0.53.2. This verifies the surface syntax only. It does not establish builtin type construction, callback dispatch, effect charging or VM support; M0 must exercise those paths with the proposed registration before implementation is committed to a signature.

### Native session lifecycle

1. Check IO authority before querying or modifying a terminal. Resolve terminal descriptors from the configured host endpoints; wrappers must expose a verified terminal endpoint rather than guessing from `os.Stdout`. If descriptors are unsupported or belong to different terminal devices, refuse native activation.
2. Acquire an exclusive terminal lease. Only one reader may own that terminal device at a time, even across contexts. Nested/concurrent acquisition returns Busy. Reject raw acquisition when existing buffered line input remains unread; do not discard or steal those bytes.
3. Register restoration **before** the first mutation. Save the exact input termios state; initialize bounded decoder state; optionally enter alternate screen/hide cursor. Failure at any step rolls back all completed steps before returning.
4. Invoke the callback synchronously through the runtime's function-caller facility. Cleanup is a Go-owned scope, not an AILANG `let` following the callback. Budget exhaustion, runtime error and the `EvalExitCode` sentinel unwind the same scope. Preserve the original error/exit/panic after restoration; attach restoration failure as a secondary diagnostic rather than turning failure into success.
5. Restore mode, cursor and screen, flush trusted restoration output, revoke handles, stop watchers and release the lease exactly once. Cleanup is mandatory and is not denied by an exhausted program IO budget.

Budget scopes share a lease pointer; `EffContext.Clone` starts without an active session. The host device lock still prevents a cloned request competing for the same terminal. Access outside the lexical scope returns InvalidSession.

Signal ownership must be explicit. Use noncanonical/no-echo input with signal generation retained (`ISIG` on), so terminal Ctrl+C reaches the native CLI runner even while a callback is inside another blocking effect. Do not use unmodified `term.MakeRaw` settings, which disable signal generation. The native CLI runner registers SIGINT/SIGTERM handling only while a session is active, restores first, then applies the runner's termination behavior (conventional 130/143 unless an existing runner policy dictates otherwise). It must not swallow a signal or invoke an AILANG callback from a signal goroutine. An embedder receives lifecycle hooks and must provide its own termination/cancellation integration; no process-global handler is silently installed for an embedded evaluator. Native acquisition by an embedder without the required lifecycle integration returns Unsupported. This design does not promise to cancel arbitrary AI/network callbacks. Terminal restoration must not wait for such a callback to finish. SIGKILL, power loss and unrecoverable process death cannot have an in-process cleanup guarantee.

### Input decoding and resize

Use a bounded incremental decoder: UTF-8 text, common CSI/SS3 navigation keys, CR/LF Enter and DEL/BS Backspace. Escape alone must resolve after a documented 30 ms ambiguity deadline, independent of how many OS reads split an escape sequence. Bound an undecided control sequence to 64 bytes; malformed/unsupported controls produce InputFailure and never become executable rendering text. Invalid UTF-8 produces a typed error. Mouse reporting, modifier-complete keyboard protocols and bracketed-paste negotiation are future work; literal pasted text is read as Text events within the same bounds.

POSIX resize notification causes a fresh physical size query and a Resize event. Coalesce resize bursts to the latest dimensions, with a bounded queue (64 events) and documented ordering relative to already decoded keys. Never silently drop keys: queue overflow is InputFailure. EOF produces EndOfInput exactly once, and later reads report the closed session state. Native Ctrl+C is handled as interruption, not an ordinary Text event.

The worker that reads the descriptor must be cancellable/joinable on cleanup. A goroutine blocked permanently on stdin is not an acceptable implementation. Choose poll/nonblocking descriptor integration against existing `x/term`/`x/sys` dependencies; do not change flags without restoring them. Shared use with `readLine`, `asyncReadStdinLines` or a second event pump is refused while the native lease is active. M0 must map and prove this exclusion in the actual reader paths before freezing it.

### Package evolution: `sunholo/terminal_ui` v0.2.0

Preserve `ui` exports and sanitation. Add the following public modules:

| Module | Responsibility |
|---|---|
| `events` | Package-level input ADT, pure key-to-action mappings, versioned event transcript encode/decode |
| `widgets` | Bounded selection list, confirm dialog and paging state; pure update functions returning model plus optional accepted/cancelled result |
| `adapter` | Native/line driver, explicit mode resolution, measured viewport handling and effectful rendering |

Expose composable primitives before a generic application framework. The caller owns the event loop and business effects. Widget handlers must not execute shell commands, grant capabilities or call providers. A small standalone demo composes a selection list, a confirmation and the existing paged body; it proves the package can serve CLIs beyond the captain/crew application.

Keep `[effects].max = ["IO"]` for the first package release. Core's effect-polymorphic scope allows consumer code to declare additional effects without adding those effects to the package's own adapter contract. Add exports for the three modules, retain `ui`, and keep `[bin] terminal-ui-demo = {module = "demo", entry = "main", caps = "IO"}`. Set the package's minimum AILANG version to the first release containing both native primitives and exact line EOF; do not ship a manifest floor naming a version whose APIs are absent.

### Modes and physical viewport

Provide explicit `native`, `line`, `plain` and `auto` options in the demo/adapter. Native requires suitable terminal endpoints and fails on activation/query errors. Line accepts explicit dimensions, uses no raw mode and requires Enter. Plain emits no escape sequences and no full-screen clears. Auto queries capabilities once, reports the resolved mode, and chooses native only when supported; otherwise it chooses the documented line/plain behavior. A failed native activation is an error, not an automatic downgrade. Machine-output behavior belongs to the consuming CLI's separate `--json`/noninteractive path.

The adapter never replaces an unknown physical size with invented 80×24 data. Explicit dimensions remain supported for line and recorded modes. Native physical sizes below 40×16 must bypass `ui.dimensions`, whose existing minimum clamp would enlarge them: render a compact, sanitized message and quit/help controls within the measured width/height, preserve application state, and resume the regular screen after enlargement. For large screens cap layout at existing 160×60 limits. Resize repaginates and clamps the current page/selection; all actions remain reachable.

The existing demo treats `readLine() == ""` as EOF, conflating blank input with end of input. Its upgraded line driver must use the separately designed `readLineOpt`; a blank line is a no-op and EOF terminates. This dependency must land or the package release must remain blocked; do not add another heuristic.

### Replay and output discipline

Package transitions and rendering consume explicit viewport, theme and input values. Record a versioned initial configuration plus ordered normalized Key/Resize/Idle/EOF/Interrupted inputs. Replay supplies those values to pure transitions and rendering without querying a live terminal, changing modes or waiting on clocks. Compare final model and ordered frames; rendering remains sanitized, and machine-readable exports remain independent of display text.

Existing effect traces provide observational evidence but render/truncate values; `effects.Call` is not by itself a lossless event replay store. The package transcript is explicit caller-controlled data. The package need not gain FS just to record: return/encode transcript values and let a consumer decide whether to persist them. Document that captured input can contain user text; do not treat capture as automatic telemetry. Frame-size and transcript-length limits are caller configuration with typed failure, not silent truncation.

## Implementation Plan and Files

This is a design outline, not an approved sprint. Work spans two repositories; stage one must produce a core release usable by stage two.

| Phase | Deliverable | Estimate |
|---|---|---|
| M0 | Freeze public signatures, endpoint identity, reader ownership, signal integration, backend behavior and dependency status with focused probes | 1–2 days |
| M1 | Core info API, native lease, scoped callback, decoder, resize and cleanup; fake-host tests | 3–4 days |
| M2 | Real PTY regression suite, evaluator/strict-VM parity, unsupported-host behavior, docs and core release evidence | 2–3 days |
| M3 | Existing package's events/widgets/adapter/demo, line EOF integration, replay and installed-bin controls | 2–3 days |

Core files (proposed): `std/terminal.ail` (~100 LOC); `internal/builtins/terminal.go` (~250); `internal/effects/terminal*.go` (~650 plus ~550 tests, platform split); endpoint/lease fields and reader exclusion in `internal/effects/context.go`, IO and stream stdin paths (~120); runner signal/lifecycle and endpoint wiring (~100 plus tests). Inspect `internal/vm` builtin/evaluator interoperability and `internal/gen/golang` builtin routing in M0. Native evaluator and strict bytecode are release gates; Go codegen must either support these operations or reject them before execution with a clear diagnostic. WASM info/native calls must return typed Unsupported without Go signal/TTY assumptions. No parser syntax change is proposed.

Package files (in `sunholo-data/ailang-packages/packages/terminal-ui`): new `events.ail`, `widgets.ail`, `adapter.ail` (~500–800 LOC together), native tests and `_smoke.ail` (~300–500); update demo, manifest, AGENT.md and CHANGELOG.md. Keep existing `ui.ail` behavior and tests. Documentation: update the maintained limitations entry and cross-reference this proposal from the old exclusion when implementation lands; update builtin docs and package authoring examples. Estimates are provisional, not measured velocity.

## Conflict Surface

| Shared position / mechanism | Existing valid use | Required isolation |
|---|---|---|
| `IO` capability and operation dispatch | Line reads, writes, exit, overrides, budgets, traces | All new operations pass capability/budget checks; mandatory cleanup remains host-owned |
| Configured stdin and persistent `bufio.Reader` | `readLine` retains buffered data; async line streams read concurrently | Exclusive raw ownership; refuse leftover buffered data and competing readers |
| Callback invocation and effect rows | `std/stream` uses row-polymorphic handlers and `FnCaller` | Preserve `{IO,e}` charges, callback errors and strict-VM dispatch |
| `EffContext.WithBudget` / `Clone` | Shared same-run state vs independent request state | Share lease only across same-run budget scopes; revoke escaped/cross-context handles |
| Entry execution and exit sentinel | `internal/runner/run.go` recovers `EvalExitCode` before returning to CLI exit | Restore inside the terminal scope before outer recovery/termination |
| Program output override | Runner redirects IO output for `--emit-trace` | Query actual configured output; never corrupt trace stdout or open another device |
| Existing package's dimension contract | `dimensions` clamps to 40..160 × 16..60 | Keep API behavior; adapter handles smaller physical terminals separately |

No grammar position is extended, so parser ambiguity is unchanged. New module types and effect-polymorphic builtin registration still require cross-module/backend checks. Existing fixtures that must retain behavior: `examples/progress_bar.ail`, `examples/runnable/micro_io_echo.ail`, `examples/runnable/test_io_builtins.ail`, plus the pinned package's `ui_test.ail` and line demo. Their files were inspected during design. Deliberate changes: the maintained raw-input exclusion is revised only for the native host API; the demo's native mode gains immediate keys/resize and exact EOF. Existing line-reader semantics and `ui` exports remain unchanged.

## Examples

Current verified consumer flow: `ailang install --path packages/terminal-ui` installs the existing `[bin]` command; its guide says every navigation key needs Enter. Package source and tests were run directly during design; installation itself is an implementation acceptance check, not a completed measurement here.

Proposed demo flows (future behavior):

```text
terminal-ui-demo --mode native
  Up/Down choose an item; Enter confirms; Esc/q quits; resizing reflows the screen.
terminal-ui-demo --mode line --columns 60 --rows 24
  Existing n/v/h/b controls plus Enter; blank input stays; EOF quits.
terminal-ui-demo --mode plain
  No ANSI escapes; stable text for redirected input/output.
```

Consumer integration: depend on the exact published `sunholo/terminal_ui` version, import the pure widget and renderer modules, map typed events through a pure update function, and declare the consuming command in its existing `[bin]`. The application keeps journal/business data intact; sanitize only display projections. No CLI installs or registry publication were performed while authoring this design.

## Success Criteria and Testing Strategy

- [ ] Installed demo runs from an unrelated cwd with IO only; all native navigation actions work without Enter.
- [ ] Physical size is measured and emitted on resize; shrink to 20×8 never renders a fabricated 40×16 viewport; enlargement retains state and clamps paging.
- [ ] PTY tests compare terminal attributes before/after success, callback error, budget exhaustion, `exit(7)`, panic, SIGINT and SIGTERM. Verify cursor/alternate-screen restoration bytes and signal exit behavior, not just process completion.
- [ ] Missing IO authority mutates nothing; fake/stale/cross-context handles and nested sessions fail; line/async/native reader ownership is tested; all workers stop after teardown.
- [ ] Decoder handles split UTF-8/escape sequences, lone Escape, EOF, partial EOF, invalid bytes, timeout, oversized sequences and resize bursts. Mutation of key decoding or cleanup must fail the targeted control.
- [ ] No-terminal and unsupported-platform behavior returns typed outcomes; injected input/output endpoints are honored. Native activation errors are never hidden by auto downgrade.
- [ ] Same recorded inputs produce identical models and ordered frames, without native IO; a changed key/resize demonstrably changes the expected result.
- [ ] Existing package tests pass under evaluator and strict bytecode with zero VM fallback. Run source inline tests explicitly and report property skips separately; add valid-domain properties for the new bounded models.
- [ ] Line mode distinguishes blank input from EOF after M-IO-READLINE-EOF-OPTION lands; redirected/plain output contains no ESC bytes.
- [ ] `ailang pkg quality --strict .`, lock/check, native tests, explicit inline tests and smoke all pass with totals recorded. `publish --dry-run` is packaging evidence only. Registry publication and consuming package installation are verified separately when authorized.
- [ ] Core targeted tests, `make check-boundaries`, formatting/lint and affected backend checks pass; docs, AGENT.md and examples are updated. Go codegen/WASM behavior is explicit.

## Verification Log (2026-10-10)

| Claim | Evidence | Outcome / limit |
|---|---|---|
| Package already exists | GitHub contents and package commit history; manifest/ui/demo/AGENT/test files copied at `145baa732e26e8a4cfe7464c55d9a258bc63dcab` | Manifest v0.1.0, IO ceiling, ui export and `[bin]` verified; local sibling checkout predates this package |
| Source is viable on installed toolchain | `ailang check --package .` in temporary pinned copy; `ailang version` | Three files check; binary v0.53.2 `bebfe6818`; local core `std/VERSION` is v0.53.1, so distinguish binary and checkout versions |
| Existing named tests work | `ailang test --package . --json`; same with `--bytecode --strict-bytecode` | Both: 12 passed, 0 failed/skipped; strict VM: 12 named bodies, 0 fallback |
| Inline tests are not equivalent to named tests/proof | `ailang test ui.ail --json`; `ailang test demo.ail --json` | ui: 8 passed, 0 failed, 10 property skips; demo: 4 passed, 0 failed, 1 property skip. Generated precondition-domain shortages are not proof |
| Missing key API | `ailang docs std/io`; `ailang check terminal_missing.ail` importing `std/io(readKey)` | IMP010: readKey not exported; current registration and IO handlers confirm line-only public input |
| Proposed callback/event syntax | `ailang check terminal_protocol.ail` with ADTs and stub `withTerminal[a](..., body: TerminalSession -> a ! {e}) -> Result[a, TerminalError] ! {IO,e}` | Check passes; stub always returns Unsupported, so no runtime/builtin/cleanup claim follows |
| Small-size clamp and demo EOF behavior | Read pinned `dimensions`, `screen` and demo `loop` bodies | Minimum 40×16, explicit 60×24, `readLine()==""` ends demo; adapter must handle both issues |
| Terminal operations need host changes | Read `effects.Call`, context `GetIOReader/GetIOWriter/WithBudget/Clone`, runner output wiring and exit recovery | Existing dispatch/callback facilities can be reused; buffered IO and clone ownership are material integration points |
| Existing terminal dependencies and signal flags | `go.mod` pins `x/term v0.46.0` and `x/sys v0.48.0`; read cached `x/term@v0.46.0/term_unix.go` `makeRaw` | Dependencies exist; `makeRaw` clears `ISIG`, so retain signal generation explicitly for the promised Ctrl+C cleanup |
| Existing docs rule out raw input | Read M-TERMINAL-IO non-goals, M-AGENT-STEP-CANCELLATION, maintained limitations input section | Scope exclusion verified; this document proposes revision, not an already approved implementation |
| No separate TUI ticket located | Open/closed issue searches for tui/terminal/terminal_ui/native IO/scoped cleanup in core and package repos; recent issue title scan | No matching TUI ticket found; #231 is AI cancellation. Search absence is not proof no ticket exists elsewhere |
| Registry discovery | Successful network-enabled `ailang pkg search tui` and `... terminal` | No keyword matches; publication of the existing package remains unverified |
| Related-doc search instrument | Scaffold reported identical high-ranked SimHash/neural results for unrelated CLI args, AI streaming, tag routing, concurrency and billing DX docs; read titles/scopes, then targeted raw-input/bin/EOF search | Reported scores do not establish substantive coverage. Existing docs address prerequisites/exclusions, not this package integration; user explicitly authorized the follow-on design |

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Package request grows into a second UI framework in core | Limit core to facts/session/events; widgets and business policy stay in existing package |
| Deferred cleanup alone misses asynchronous termination | Real signal/PTY controls and host-owned restoration before runner termination |
| Raw reads steal buffered/async input | Device lease plus enforcement in all existing reader entry paths; M0 proves ownership mechanism |
| Terminal/session state leaks across requests | Context validation, clone reset, budget-scope sharing and stale-handle tests |
| Replay claims overstate trace fidelity | Explicit versioned package transcript; distinguish observational effect traces from full replay |
| Cross-repository drift or premature package release | Exact minimum toolchain floor, core release first, package integration against that released binary |
| Tests appear green because generation skips bounded inputs | Report skips and add explicit valid-domain properties; do not treat quality badges as proof |

## Axiom Compliance

| Axiom | Score | Justification |
|---|---|---|
| A1 Determinism | +1 | Pure transition/render behavior depends on explicit inputs; native observations remain IO |
| A2 Replayability | +1 | Versioned normalized events and viewport configuration enable provider-free UI replay |
| A3 Effect Legibility | +1 | Terminal operations and consumer effects remain declared and charged |
| A4 Explicit Authority | 0 | Existing IO grant only; configured endpoints, no implicit device/shell authority |
| A5 Bounded Verification | +1 | Bounded decoder/queue/layout models and focused PTY controls |
| A6 Safe Concurrency | 0 | Exclusive lease and reader ownership; no new language concurrency surface |
| A7 Machines First | +1 | Typed events/errors plus plain and replay adapters remain usable by agents |
| A8 Minimal Syntax | 0 | Existing modules, ADTs and effect rows; no grammar extension |
| A9 Cost Visibility | +1 | Explicit IO reads/timeouts and bounded frame/transcript settings |
| A10 Composability | +1 | Reuses `[bin]`, IO and existing package; pure widgets fit consumer loops |
| A11 Structured Failure | +1 | Typed unsupported/query/input/session failures; teardown preserves original failure |
| A12 System Boundary | +1 | OS terminal work confined to host API, UI policy confined to package |

**Net: +9.** A1/A3/A4/A7 have no proposed hard violation. This is author assessment, not independent quorum or permission to implement. Native input itself is nondeterministic IO; only transitions/replay with fixed recorded inputs are deterministic.

## Deferred Decisions and Non-Goals

The implementer may choose private helper names, test file organization, default colors and the internal poll abstraction within the fixed bounds. M0 freezes diagnostic payloads and backend registration details with evidence. Version scheduling is an operator/release decision; v0.54.0 is a proposal.

This first release excludes mouse support, rich keyboard-protocol negotiation, password-entry guarantees, background AI-step cancellation, a general task scheduler, Go CLI rewrites and a curses-style stdlib. Windows native input and browser-provided terminal sessions are future host adapters; pure rendering, line use and recorded events remain useful independently. Additional forms, tables, search boxes and themes can extend the same package after selection/confirmation/paging validates the model.

## Related Documents

- [M-PKG-BIN-ENTRYPOINTS](../implemented/v0_40_1/m-pkg-bin-entrypoints.md) — implemented command distribution; reused, not redesigned.
- [M-TERMINAL-IO](../implemented/v0_27_0/m-terminal-io.md) — implemented escapes/flush; explicitly excluded this input surface.
- [M-AGENT-STEP-CANCELLATION](v0_29_0/m-agent-step-cancellation.md) — separate #231 use case and older exclusion this proposal asks to revise.
- [M-IO-READLINE-EOF-OPTION](v0_52_0/m-io-readline-eof-option.md) — exact line EOF dependency, kept in its existing design.
- [Initial triage](ailang-core-triage/native-terminal-tui-input.md) — superseded routing context: existing-package evidence and operator request now warrant this design.
- [Maintained limitations](../../docs/docs/reference/limitations.md#interactive-stdin--keyboard-input) and [PROGRAM](../PROGRAM.md) — public boundary and routing principles.

**Created / last updated**: 2026-10-10. Implementation and package publication are authorized by the user messages recorded above. No message acknowledgement or unrelated coordinator approval is included.

### M4 integration addendum: recursive row alias resolution

The complete crew consumer gate exposed a pre-existing compiler defect also
reproduced on untouched dev: a pure recursive function with five or more nested
conditional self-calls could leave the final application's effect row unowned.
The row unifier applied only one substitution hop before comparing tails, then
rebound an intermediate alias and disconnected the final call from the declared
closed row. This is independent of native terminal dispatch.

The integration repair resolves the complete alias chain before row unification,
retaining row kind, labels, budgets, minimum budgets, refinement parameters and
provenance. A cycle guard keeps cyclic aliases open rather than silently treating
them as pure. ApplicationEffects ownership checks remain mandatory. Regression
controls cover four/five/eight recursive branches, real IO rejection in a pure
recursive function, effect/record multi-hop open and closed aliases, metadata and
cycle behavior. General structural substitution APIs are unchanged; this repair
is confined to the row-unification boundary.
