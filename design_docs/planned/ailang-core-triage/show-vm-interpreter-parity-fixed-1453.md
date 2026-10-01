# #1453 show() VM/interpreter parity — fixed on dev (resolution notice)

- **Date**: 2026-10-01
- **Class**: not-actionable
- **Recommend**: drop
- **Searched**: `show` × `parity`, `1453`, `elision` × `depth limit` across `design_docs/`; `design_docs/planned/ailang-core-triage/` listing; `git log --grep 1453`
- **Verified**: commit `7c640e83` ("fix(vm): one show renderer for both engines; VM ADTs carry constructor names") is present on this branch, cites `Fixes #1453`, and introduces `builtins.RenderShow` as the shared renderer plus a `TestShowParityVMvsInterpreter` regression test covering 13 entries (ADT constructor syntax, depth limit, 80-column elision).

The report is a follow-up announcing that the defect is already fixed, not a request for work: show() output and printed entry results are now byte-identical between the interpreter and `--bytecode`/`--strict-bytecode` because both engines route through one renderer, so the drift class is closed structurally rather than patched. No design doc claimed this item (none found — searched the terms above, an empty result), no backlog row names #1453, and there is nothing left to decide, estimate, or implement. The correct disposition is to drop the report and ack the message; the only residual action for a maintainer would be confirming #1453 is closed upstream.
