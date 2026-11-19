### Fixed — Gate 0 self-notices read is no longer blind when no watermark file exists (2026-11-19)

`scripts/mission_gate0_self_notices.sh` exited 2 ("watermark channel unavailable") whenever every
`--watermark-file` was absent. Since #981, `mission_directives.sh` creates
`mission-<issue>-last-seen` only on `--mark`, so a mission with no routed human directive never has
one. As a result the Gate 0 step 6a read was an instrument floor on every fire for the fleet (#2341/#2172) and
stapledon (#450/#410), where it hid a real died fire (#410, 2026-11-13, rc=1). All-absent is now
a DEGRADED read of the whole of each issue since `1970-01-01T00:00:00Z`, which matches how
`mission_directives.sh` reads an absent file. A `watermark: DEGRADED` line and a
`watermarks 0/N, all absent: whole-issue read` verdict suffix make it visible, and it exits 0/1 by
hits. A file that exists but is unreadable, empty or invalid is still a floor (rc 2). The
mission-control skill's step 6a (both copies) explains it. Ticket
`gate0:self-notices-rc2-no-watermark-file`.
