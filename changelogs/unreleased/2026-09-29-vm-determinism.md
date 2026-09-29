### Fixed — bytecode VM record operations are deterministic (2026-09-29)

Unresolved record field access and record updates no longer guess a layout by ranging over Go maps.
Field reads fall back to runtime name lookup, while the new `UPDATE_RECORD` bytecode operation copies
the runtime base shape and applies overrides by name. ADT fallback selection follows declaration order,
and compiler/CLI regression guards cover repeated compilation plus evaluator/VM parity. Fixes #1355 and
#1360.
