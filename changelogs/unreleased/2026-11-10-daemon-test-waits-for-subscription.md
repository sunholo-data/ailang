### Fixed — `internal/daemon` tests wait for the subscriber instead of sleeping 20 ms (2026-11-10)

Nine daemon tests started the daemon in a goroutine, slept a fixed 20 ms and then delivered a
message, assuming the subscriber had registered. On a loaded CI runner it sometimes had not, and the
test died with "no handler registered". It failed the macOS build twice in one day on pull requests
that do not touch the package. The test double's `deliver` now waits for the subscription itself
(bounded at 10 s), which removes the race at all nine call sites. A regression test subscribes 150 ms
late and must still be delivered to.
