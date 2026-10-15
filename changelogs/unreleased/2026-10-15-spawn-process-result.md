### Added
- `std/process.spawnProcessResult` returns `Result[ProcessHandle, ProcessError]` so missing, denied, or failed worker startup can fall back without aborting the run. Existing `spawnProcess` retains its fail-fast behavior.
- Process documentation clarifies literal allowlist matching and `WorkerHandleInvalid` after automatic reaping.
