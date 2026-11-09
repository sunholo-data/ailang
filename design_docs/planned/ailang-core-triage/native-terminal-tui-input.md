# Native terminal input for pure AILANG TUIs

- **Date**: 2026-10-10
- **Class**: feature
- **Recommend**: design-doc
- **Searched**: `TUI`, `terminal UI`, `readKey`, `terminalSize`, `raw mode`, `TTY`, `SIGWINCH`, `scoped cleanup`, `native IO`; GitHub issue titles/bodies across open and closed issues, and terminal PR titles.

Source: unread canonical message `inbox_1791568104057_d08245d1`, from `stapledons_godot` (2026-10-09), “DX request: native IO terminal keys, size and scoped cleanup for pure AILANG TUIs.” **Existing coverage is a scope decision, not an implemented feature.** `m-agent-step-cancellation.md` explicitly rules raw-mode single-keypress input outside core, and the maintained `docs/docs/reference/limitations.md` section “Interactive stdin / Keyboard Input” repeats that decision while allowing a future host/extension capability. The implemented `design_docs/implemented/v0_27_0/m-terminal-io.md` delivered terminal output/flush and explicitly excluded raw input. Current `std/io.ail`, builtin registration (`registerIO`), effect handlers (`ioReadLine`), and installed `ailang docs std/io` still offer line input, without public key, size, TTY-query, or scoped raw-mode operations. No separate matching TUI ticket or full input-API design was found. Related open [#231](https://github.com/sunholo-data/ailang/issues/231) concerns cancellation of an AI call and must not be treated as the TUI ticket; `design_docs/planned/v0_52_0/m-io-readline-eof-option.md` separately designs line EOF handling. Preserve this report as new consumer demand against the standing exclusion. If the operator chooses to revisit that decision, write a dedicated native terminal host-capability design covering TTY/size queries, typed key/resize/EOF events, scoped cleanup on error/exit/interrupt, recorded-input replay, redirected-input behavior, and platform/WASM boundaries; keep rendering/navigation policy in an AILANG package. That changes a public surface and requires design approval and a sprint, not a direct fix. No implementation, dispatch, ticket mutation, or message acknowledgement was performed.

**Follow-up, 2026-10-10:** The operator requested the design and clarified the existing-package motivation. Inspection of `sunholo-data/ailang-packages` at commit `145baa732e26e8a4cfe7464c55d9a258bc63dcab` found `packages/terminal-ui`, manifest `sunholo/terminal_ui@0.1.0`, with a pure renderer and IO-only `[bin]` demo. This was absent from the local sibling checkout and not found by registry keyword searches. It checks on v0.53.2 and its 12 named tests pass under evaluator and strict bytecode (zero VM fallback). The initial duplicate-of recommendation is superseded: the old docs do not design this package integration. [M-TERMINAL-UI-NATIVE-INPUT](../../implemented/v0_54_0/m-terminal-ui-native-input.md) proposes the enabling host API and evolution of the same package, explicitly revisiting the prior scope exclusion. Design creation is authorized; implementation is not yet approved.

**Delivery closure, 2026-10-10:** The operator subsequently approved sprint execution,
PR merges, publication and the supporting release. Core PR #1756 and package PR
#117 merged; AILANG v0.54.0 released and sunholo/terminal_ui@0.2.0 published.
Exact-head CI and fresh official registry consumer/native restoration tests pass.
Independent round2 PASS completes the native terminal sprint; design and companions
are archived together in implemented/v0_54_0. The separate follow-up worker leak
inbox_1791630246888_28589e92 remains unresolved and needs dedicated runtime work.
No message acknowledgement or unrelated coordinator approval was performed.
