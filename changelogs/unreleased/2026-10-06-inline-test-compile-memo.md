### Fixed — inline `tests [...]` compiled the module twice per case (#1328) (2026-10-06)

Every inline test case ran two full pipeline compiles of the same file: one to extract the function binding and one for its dependency cluster. A 41-case file therefore paid 82 compiles. Both compiles read the file from disk, so each now runs once per file and every case reuses the result. Module-less files strip per function, so they keep one binding compile per function.

Measured on motoko (mk-main 4d4917cd), all tests passing with the same per-test outcomes. "Before" means after the 64 MiB compile-cache limit fix and before this change; #1328 reported 2,336 s for `session.ail` on v0.44.1.

| File | Before | After |
|---|---|---|
| `src/core/session.ail` (41 cases) | 122 s | 35 s |
| `src/core/ext/runtime.ail` (37 cases) | 55 s | 14 s |
| `src/core/test/scripted_ports.ail` (10 cases) | 53 s | 33 s |
