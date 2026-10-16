### Fixed — `--bytecode` re-ran a program after an evaluator-only function had performed IO (#1576, B4) (2026-10-16)

When the bytecode VM failed partway through a run, non-strict `ailang run --bytecode` restarted the
entry on the evaluator unless the VM had itself dispatched a native effect. IO performed by an
evaluator-only callee reached through the bridge (a `@limit` budget frame, a `Rand[mode=seeded]`
function, a stream builtin) was invisible to that guard, so the restart duplicated committed output
or ran against already-consumed stdin — the silent-truncation shape stapledons-godot reported.
`EffContext` now counts every capability check (`EffectAttempts`, shared across budget views), and
the runner refuses the fallback when the count moved during the VM run: the run fails loudly with the
VM error instead. A fallback is still allowed when nothing effectful happened first.

Behaviour change: `examples/runnable/stream_sse.ail` and `stream_websocket.ail` now fail under
non-strict `--bytecode` (the bridge cannot convert the stream builtins' untyped `Result`/`StreamConn`
values, M-BYTECODE-2E scope). They previously appeared to pass only because the fallback connected a
second time. The evaluator path is unchanged.
