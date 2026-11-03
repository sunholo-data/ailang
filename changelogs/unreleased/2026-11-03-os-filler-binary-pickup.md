### Fixed — local-rig rotation banked five days of runs under a stale binary's version (2026-11-03)

`tools/launchd/os-rotation-filler.sh` only reinstalled `ailang` when origin/dev's `std/VERSION`
moved past the checkout's. Releases cut in the rig's own checkout move both together, so the
pickup never fired: from 2026-10-29 to 11-03 the rotation ran a `v0.55.0-dirty` binary while
`std/VERSION` read v0.56.1 and then v0.57.0. eval-suite banks by the binary's version and the
coverage gate counts by `std/VERSION`, so 220 runs landed under v0.55.0, coverage read `0/56`
forever, and v0.56.1/v0.57.0 were never measured on the rig. A new step 3c compares
`ailang --version` with `std/VERSION` every cycle, reinstalls on mismatch, snapshots the old
binary's version, and exits non-zero without banking if the mismatch survives the reinstall.

### Changed — local-rig rotation parks benchmarks no harness passes, and reports its value weekly (2026-11-03)

`tools/eval-park-set.sh` names benchmarks with zero passes in at least 4 rows from at least 2
models over the last 3 version banks (today: `cp_60159`, `cp_60166`, almost all timeouts). The
filler takes them out of the coverage gate and runs each as one probe trial on
`pi-qwen3-8-27b` per release; a passing probe un-parks it. `OS_FILLER_PARK=0` restores the old
behaviour. `tools/rig-value-report.sh` reports, once a week to the prod `user` inbox: which
version the rig actually measured, runs and banked hours (timeouts bank `duration_ms=0`, so
they are listed separately), harness gaps, regressions against the previous version, and the
saturated and parked sets. `OS_FILLER_REPORT=0` turns it off.
