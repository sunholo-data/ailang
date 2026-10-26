### Fixed

- Prefetch the eval harness's Python 3.12 runtime in CI setup before tests disable
  network access, so a cold `uv` cache does not turn CLI-argument and stdin tests
  into interpreter-download failures. Offline test guards remain enabled.
