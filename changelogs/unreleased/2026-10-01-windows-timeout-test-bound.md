### Fixed — `TestCheckTimeoutAndIgnoredOutput/timeout` no longer flakes on Windows CI

On Windows `proctree` kills only the process leader, so the orphaned `sleep` held the output pipe until `WaitDelay` (1s timeout + 2s grace + slow runner git ≈ 6s) and overran the 5s bound (runs 36760993847, 36830798996). The bound is 10s on Windows, still well under the 30s sleep it proves was cut; Unix keeps 5s.
