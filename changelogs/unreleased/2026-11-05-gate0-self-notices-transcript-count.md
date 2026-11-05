### Fixed — Gate 0 reports transcript-secret scan notices (2026-11-05)

Gate 0 now extracts old and current self-authored transcript-scan notices alongside crash notices,
reporting notice counts, hit counts, fire IDs, and public URLs without comment bodies. Transcript-only
windows retain exit code 0; crash and instrument-failure exit codes remain 1 and 2. The driver-source
control requires the current transcript prefix as well as the crash signature. Related work: [#2127](https://github.com/sunholo-data/ailang/issues/2127).
