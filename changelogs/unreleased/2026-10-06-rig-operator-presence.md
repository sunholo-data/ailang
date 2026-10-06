### Added — rig batch jobs step aside while someone is using the desktop

The rig's GPU also draws its screen: with a local model generating it sat at 98–100% and
window focus lagged ~5s. `dev.ailang.rig-operator-presence` (new LaunchAgent) keeps
`rig.operator` beside the rig lock in force while there has been keyboard/mouse input in
the last 10 minutes (`HIDIdleTime`; Screen Sharing counts, SSH does not).

- The OS rotation filler skips its cycle while the operator is present.
- Jobs that export `AILANG_RIG_YIELD_TO_OPERATOR=1` (filler, nightly-eval,
  nightly-lang-eval) lend the GPU at `riglock.Checkpoint` — between benchmarks — and
  resume once the operator is idle. Attended runs don't set it, so they never pause.
- Separate from the cooperative handoff, which stays first-come for short jobs such as
  Daneel's intake. The marker carries the watcher's pid and a 60s expiry, so a dead
  watcher cannot pause the rig.
- Tunables: `RIG_OPERATOR_IDLE_SEC` (600), `RIG_OPERATOR_POLL_SEC` (20).
