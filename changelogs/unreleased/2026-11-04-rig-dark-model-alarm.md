### Added — rig dark-model alarm

- `tools/rig-model-health.sh --vdir DIR --models CSV`: flags an active rotation model that banked
  ZERO valid rows on a version while a peer banked ≥3 (exit 3). It does not fail open: a broken
  check exits 1 and says so. Fixture test: `tools/launchd/test_rig_model_health.sh`.
- os-rotation filler step 10 runs it after every cycle and alerts the prod `user` inbox once per
  (version, dark set), quoting this cycle's failure line (usually the pre-flight canary); it
  re-alerts when the set changes and sends a recovery note when it clears.
- The weekly `tools/rig-value-report.sh` shows a DARK MODEL warning and lists active models with no
  rows instead of omitting them.

### Fixed — motoko absent from the v0.57.0 rotation

- From 2026-11-03 16:47 the rig ran v0.57.0 (#1958) against motoko checkout `835c738b`, whose test
  fixtures the new effect checker rejects; motoko's canary failed on every call and it banked no
  rows for ~21h, visible only in the filler log. The rig checkout is fast-forwarded to `083ef437`
  (the Docker source pin; 5 test-fixture annotation lines) — `check_core` 60/60, 9/9 extension
  boots on v0.57.0. MOTOKO.md §16.
