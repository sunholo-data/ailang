### Fixed — the motoko image's runtime user can resolve its registry packages (2026-10-19)

Since v0.53.1 (M-PKG-REGISTRY-CONFINEMENT), package setup hashes every registry dependency from
`$HOME/.ailang/cache/registry` and stops when one is missing. `docker/Dockerfile.agent-motoko` ran `ailang lock`
as root, so motoko's registry packages (`sunholo/logging`) landed in root's cache, and the runtime user `ailang`
had none: every `ailang_only` motoko task failed at the lane check with "could not resolve motoko profile
"ailang_only" through motoko's own resolver". Found by the executor-shadow dev smoke on 2026-10-19. The image now
copies the fetched registry cache to the runtime user, and the build runs the lane's own profile check as that
user, so an image that cannot start the lane no longer ships.

### Fixed — the motoko image's runtime user can write motoko's run state (2026-10-19)

Upstream motoko now keeps a session journal under its own repo (`.motoko/sessions/<id>/`, ADR-003). The image's
repo is root-owned, so every motoko task died at startup with `mkdir … EACCES` in `SessionJournal`, first seen
when the image moved to the a94204aa base. The image now gives the runtime user `/opt/motoko_agent/.motoko`, and
the build proves that user can write there.
