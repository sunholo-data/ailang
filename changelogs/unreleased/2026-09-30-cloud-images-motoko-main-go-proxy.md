### Changed — cloud motoko executor runs motoko main

`docker/Dockerfile.agent-motoko` now builds motoko main: commit `4d4917cd` on
branch `sunholo/main-dst` of sunholo-voight-kampff/motoko_agent. That is Arni's
upstream `main` (DST core, extension ABI 8.0) plus the cloud profile and
`motoko_ext_ailang_tools`, the same commit the rig runs. Until now the image ran
the ABI 2.2 fork at `84fa449`. bun is 1.3.14, which reads motoko main's text
`bun.lock`. The dependency step no longer swallows failures (`|| true`), and its
bare `ailang install`, which printed usage and exited 0, is gone. A real session
from the pinned commit is now a parser fixture.

### Fixed — cloud image builds survive proxy.golang.org stream resets

The v0.48.0 release build failed three times on `go mod download` ("stream
error: ... INTERNAL_ERROR; received from peer", a different module each time).
The four Dockerfiles that download modules (agent-base, dashboard, mcp,
registry-validator) now retry four times with backoff, and use
`GOPROXY=https://proxy.golang.org|direct`, which fetches a module from its origin
on any proxy error. With `,` Go falls back only on a 404/410. `go.sum` still
verifies every module.
