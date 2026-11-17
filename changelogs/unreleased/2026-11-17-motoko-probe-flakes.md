### Fixed — motoko connection probe self-test: two intermittent CI failures (2026-11-17)

`tools/eval/test_motoko_connection_probe.sh` failed in 8 of about 819 dev CI runs
(2026-10-08 to 2026-11-17) on the macOS "launchd drivers (bash 3.2)" job, with two signatures
(timer 5, refusing live path 3). Both were defects in the test, not the probe.

- **Timer arm** (`rc=0 expected_rc=73 … derived=1`): the suite's EXIT trap makes `/bin/bash`
  3.2 install a deferred SIGTERM handler that the timer child keeps until `execve`. A
  SIGTERM sent right after the fork was recorded and then discarded at exec, so the timer ran
  to completion (reproduced locally at about 1% of trials, 0% with SIGKILL). The fault is now injected with SIGKILL, and the arm runs with SIGTERM ignored,
  so reverting to SIGTERM fails every time instead of once in a while.
- **Refusing live path** (`unexpectedly succeeded`, or `treatment peer set contains OpenRouter
  endpoint: 203.0.113.8:443`): the `lsof` stub treated an empty lane-file read, which happens
  while the driver is creating or deleting the file, as the OpenRouter lane. This was reproduced only by forcing that read, never naturally, so the
  cause is argued from the code and the matching failure text. The stub now reports no
  peer for a lane it cannot name, and a new arm forces that read in both lanes.

Evidence: design_docs/fleet-mission-evidence/iteration117/plan.md.
