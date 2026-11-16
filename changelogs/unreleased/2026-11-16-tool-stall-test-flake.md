### Fixed — mission_pi_run: a tool closing mid-poll no longer reads as stream_dead (2026-11-16)

`tools/launchd/test_mission_pi_run_tool_stall.sh` arm `2_raised` failed twice on the
macOS bash 3.2 CI job (2026-11-10 and 2026-11-13, run 43361761294) with
`rc=12 verdict=stream_dead bound=10`. The cause was in the runner, not the test.
`scripts/mission_pi_run.sh` read the output file's mtime first and grepped for an open
tool call afterwards. When a tool that had run longer than `--stall-seconds` closed between
those two reads, the runner combined the old clock (idle past the stall bound) with the
closed tool (so the stall bound applied) and killed a healthy run as `stream_dead`. The rig
hits the same window whenever a tool runs longer than the stall bound under a raised
`--tool-stall-seconds`, which is the case that flag exists for.

- The runner now reads the tool state before the clock. A write that closes the tool also
  moves the clock, so it counts as progress. The grep still runs only once idle time could
  reach the smaller bound.
- New arm `2b_close_mid_poll` reproduces the failure every run. A `grep` shim appends the
  `tool_execution_end` line at the moment the runner asks whether a tool is open. Against the
  old order it fails with the CI signature (`stream_dead`, bound 10, `tool_executions: 1`).
  With the fix it passes. A second check fails if the shim never fired.
