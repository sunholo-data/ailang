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

### Fixed — an explicit `eval-suite --agent-timeout` was silently ignored (2026-11-03)

Any benchmark with a `timeout:` replaced the flag, so callers passing `-agent-timeout 2400`
for slow local models (`run_smoke.sh`, `ollama_eval.sh`) got the spec's value, raised to the
model's `hard_timeout_secs`, without knowing. `eval_harness.EffectiveAgentTimeoutSecs` is now
the one resolution: the benchmark timeout (else the flag default), raised to
`hard_timeout_secs`, raised to an EXPLICIT `--agent-timeout` as a floor. Every agent row,
including failed ones, banks the budget it ran under as `agent_timeout_secs`. No existing
caller changes behaviour: their values are all below the limits they already got.

### Changed — rig parked probes get a 3h agent budget; the filler keeps its rig lock fresh (2026-11-03)

Every parked CP benchmark died at the rig's 1h local hard timeout, while pi solved `cp_60149`
in 53 min (34/34 hidden tests). The 1h wall, not the model, may be the limit, so the one probe
per release now runs with `--agent-timeout 10800` (`OS_FILLER_PROBE_TIMEOUT`). Separately,
`rig-lock.sh` steals a lock whose directory mtime is older than 6h, and nothing refreshed it,
so any filler cycle over 6h (11-02 ran 7.7h) could have a second GPU job start alongside it.
The filler now touches the lock every 10 min, but only while it is alive and is still the
recorded holder, so crash recovery still works.
