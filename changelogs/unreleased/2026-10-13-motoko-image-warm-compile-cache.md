### Changed — the motoko image ships a warm compile cache

- `docker/Dockerfile.agent-motoko` runs `ailang check src/core/supervisor.ail` at build time, so
  Cloud Run Jobs, which always start cold, no longer recompile motoko on every task. Measured
  2026-10-13 with #1788: a cold compile takes about 76 s; with the warm cache, `ailang run` of
  motoko's supervisor starts in about 1 s.
- The build fails if any compile-cache write fails or the `session.ail` cache is missing. The
  #1778 regression (`ARTIFACT_TOO_LARGE`, a silent recompile on every run) now fails the image
  build instead. Verified: the step passes with a build containing #1788 and fails with v0.54.5.
- The MCP image needs no warm-up: its modules compile cold in 0.4 s.
