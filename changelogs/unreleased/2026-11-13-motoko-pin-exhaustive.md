### Fixed — agent-motoko image builds under the exhaustiveness check
- The v0.60.3 release build failed at `build-agent-motoko`. The pinned motoko commit (`597a6fbf`) had
  a `match` that the new `TC_MATCH_002` check (#2246) rejects, so v0.60.3 never reached test.
  `docker/Dockerfile.agent-motoko` now pins the fork's `2c2f9c8d`, where those matches are
  exhaustive. It type-checks clean on v0.60.3 (MOTOKO.md §18).
