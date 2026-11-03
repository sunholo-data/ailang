### Fixed — local-rig rotation banked five days of runs under a stale binary's version (2026-11-03)

`tools/launchd/os-rotation-filler.sh` only reinstalled `ailang` when origin/dev's `std/VERSION`
moved past the checkout's. Releases cut in the rig's own checkout move both together, so the
pickup never fired: from 2026-10-29 to 11-03 the rotation ran a `v0.55.0-dirty` binary while
`std/VERSION` read v0.56.1 and then v0.57.0. eval-suite banks by the binary's version and the
coverage gate counts by `std/VERSION`, so 220 runs landed under v0.55.0, coverage read `0/56`
forever, and v0.56.1/v0.57.0 were never measured on the rig. A new step 3c compares
`ailang --version` with `std/VERSION` every cycle, reinstalls on mismatch, snapshots the old
binary's version, and exits non-zero without banking if the mismatch survives the reinstall.
