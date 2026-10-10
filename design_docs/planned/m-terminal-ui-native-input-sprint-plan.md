# Sprint Plan: M-TERMINAL-UI-NATIVE-INPUT

**Status**: Approved for execution by operator, 2026-10-10
**Design**: [M-TERMINAL-UI-NATIVE-INPUT](m-terminal-ui-native-input.md)
**Goal**: Enable native terminal interaction and evolve the existing `sunholo/terminal_ui` package.
**Duration**: 10 engineering days / 60 hours, provisional
**Estimate**: 3,200 LOC including tests and docs; real implementation totals recorded at completion
**Branches**: `sprint/terminal-ui-native-input` in both core and package repositories
**Isolation**: `/private/tmp/ailang-terminal-sprint` and `/private/tmp/ailang-terminal-packages`; original dirty checkouts remain parked
**Authorization**: “great please sprint plan then execute” approves the proposed native IO/platform direction and execution. The follow-up “we can publish the package as part of our sprint” adds actual registry publication and fresh consumer installation to M5. A core release must first make the native API available to users and the release-gated validator; publication cannot bypass that gate.
**AILANG prompt version loaded**: v0.16.6 (`ailang prompt`, binary v0.53.2). Checkout base is core `939beb6fd`, packages `923a47b`.

## Calibration and reuse audit

The 7-day velocity script found concurrent merged sprints but no reliable per-controller LOC metric. Raw repository volume mixes many agents and is not a useful single-sprint speed estimate. Retain the design's 8–12-day range and plan 10 days rather than inventing velocity. Execution time in this attended session is recorded separately.

Registry searches `tui`/`terminal` returned no matches; `pkg info/docs sunholo/terminal_ui` returned not found. The source package at `packages/terminal-ui` is the verified reusable implementation. M3 contributes there directly; no new package or dependency alias is created. M0/M1/M2/M4/M5 are `none`: registry packages cannot replace native descriptor ownership, builtin dispatch or core checks. Each JSON audit row records this distinction. No cloud inbox handoff is needed in the local directly-invoked skill workflow.

## Dependency waves and ownership

M0 freezes the interface. Then M1 host and M2 builtin/backend work proceed in the same isolated core checkout with distinct ownership, while M3 pure package development proceeds in its own checkout. Integration M4 waits for all three; final evaluation M5 follows. The executor skill explicitly permits independent milestone sub-agents; the root integrates and owns runner wiring, plan/state/docs and end-to-end controls. Workers must not switch shared branches or revert other changes.

## M0 — Contract and deployment-independent evidence (150 LOC, 6h)

- [x] Freeze `std/terminal`: info, scoped withTerminal callback, readEvent; typed values and errors; IO only for host operations.
- [x] Confirm callback effect-row construction/VM wiring with actual registered-builtin checks, not just the earlier syntax stub.
- [x] Confirm package source, signal ownership, configured-output descriptors and buffered/async stdin exclusion.
- [x] Record exact `readLineOpt` dependency: implement its additive already-designed primitive in M2, with unchanged `readLine` semantics, because this package needs exact EOF.

Example: `examples/runnable/terminal_info.ail` (M2). Contracts: include pure size-validity helper; effects: `! {IO}` for main; inline tests: include boundary helper cases.

## M1 — Native host and decoder (900 LOC, 18h; depends M0)

- [x] POSIX info/size, device lease, validated session handles, bounded decoder, idle/EOF/resize behavior and typed unsupported-platform results.
- [x] Noncanonical/no-echo input retains ISIG. Scope restores termios, descriptor flags and trusted cursor/alternate-screen sequences on callback return, error, panic and exit sentinel.
- [x] Signals restore before injected CLI termination; embedding without lifecycle support refuses native activation.
- [x] Buffered reads, async reads, competing contexts and stale/forged handles cannot steal terminal input; scope and clone ownership are tested.
- [x] Fail-first host/decoder tests, bounded polling without stranded stdin goroutines; fake-host and real PTY controls.

Ownership: `internal/effects/terminal*.go`, context terminal state/reader ownership, IO and stream stdin guards. Root owns runner/CLI lifecycle wiring. Example: `examples/runnable/terminal_keys.ail` (M2); contracts: include pure event classification; effects: `! {IO}`; inline tests: include key/idle classification cases.

## M2 — Language surface, EOF and backends (650 LOC, 12h; depends M0)

- [x] Register terminal builtins and `std/terminal` types/signatures with callback effect propagation, capability/budget dispatch and complete metadata.
- [x] Add additive `std/io.readLineOpt` / `_io_readLineOpt` using the shared reader; preserve blank lines, final partial line and sticky EOF. Record nondeterministic trace parity.
- [x] Evaluator and strict VM execute terminal queries/native callbacks/events and EOF without fallback; cross-module types work.
- [x] Go codegen supports the new surface or rejects it clearly before execution; WASM/Windows build and return typed Unsupported for native calls.
- [x] Create/check examples and validate builtin health; new prompt v0.16.7 with existing frozen versions unchanged.

Ownership: std, builtins, VM/bytecode/gen integration, trace replay classification and examples; host EOF handler coordinated with M1 ownership. Example `examples/runnable/terminal_line_input.ail`: contracts include pure line classifier; effects `! {IO}`; inline tests include blank/nonblank inputs. Existing progress_bar/micro_io_echo/test_io_builtins remain regression fixtures.

## M3 — Existing package upgrade (850 LOC, 12h; depends M0; integration depends M1/M2)

- [x] Evolve `sunholo/terminal_ui` to v0.2.0; preserve ui exports/behavior and `[bin] terminal-ui-demo`.
- [x] Add pure events/widgets and versioned bounded transcript replay; selection, confirmation and paging are reusable.
- [x] Add explicit native/line/plain/auto adapter; activation errors are surfaced, not downgraded; exact EOF uses readLineOpt.
- [x] Real resize preserves state, small physical viewports use bounded compact output, never the old minimum clamp; large layout stays bounded.
- [x] Demo has all modes and immediate native navigation, with meaningful contracts/tests, IO-only library effects (demo adds Env for command arguments), docs, `_smoke.ail` and release description.

Ownership: package `packages/terminal-ui/**` only. Demo checklist: contracts include navigation/selection invariants; effects `! {IO, Env}` for CLI argument access; inline tests include first/last navigation and cancellation. Smoke checklist: contracts skip (effectful host driver has no pure result invariant); effects `! {IO}`; inline tests include pure smoke classifier if introduced, otherwise skip with reason and meaningful named test coverage.

## M4 — End-to-end regression and portability (450 LOC, 9h; depends M1/M2/M3)

- [ ] Installed `[bin]` runs from unrelated cwd; PTY arrow-key/resize tests work under evaluator and strict VM.
- [ ] PTY state/escape restoration is checked for normal/error/budget/exit/panic/SIGINT/SIGTERM; changed decoder/cleanup mutation fails controls.
- [ ] Plain output has no ESC; blank line survives; EOF exits; event replay is deterministic and altered event changes outcome.
- [ ] Full applicable test/lint/boundary/build checks pass; inherited baseline/environment failures are recorded separately and do not masquerade as regressions.
- [ ] Package lock/check/named/inline tests/strict quality/smoke/dry-run evidence includes actual totals and property skips.

Root ownership: runner/CLI wiring, end-to-end tests, package installation controls and portability validation. Existing example fixtures must be checked/run where applicable.

## M5 — Documentation, evaluation and delivery (200 LOC, 3h; depends M4)

- [ ] Changelog fragment, maintained limitations, builtin/package docs and usage examples reflect actual behavior/platform boundaries.
- [ ] Sprint plan and JSON record each acceptance result and all required evidence; both ride the core branch.
- [ ] Sprint-evaluator assesses tests/lint/acceptance/code/docs/fidelity plus conditional regression coverage; fix real findings and re-evaluate.
- [ ] Core/package changes are committed separately and reviewable; supporting core version and validator are verified, package is published, and a fresh consumer installs and runs the published binary.

## Schedule and release boundary

Days 1–2 M0 and fail-first tests; days 2–5 M1/M2; days 4–7 M3; days 7–9 M4; day 10 M5 and buffer. Parallel agent execution reduces attended elapsed time but does not turn the estimates into measured velocity.

The sprint validates against its built binary and package checkout. Package registry publication is required by the user’s scope expansion; the validator’s release-only rollout requires a supporting core release first; no package claims published availability or silently relaxes its runtime floor. If core release scheduling is pending, keep a truthful minimum version and record publication as pending rather than claim the sprint complete.
