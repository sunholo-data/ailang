### Fixed — the motoko image builds on std/ai `Message.media`

v0.61.0 renamed `std/ai` `Message.images` to `Message.media` (#2458). The motoko commit that
`docker/Dockerfile.agent-motoko` pinned still wrote `images: []`, so `build-agent-motoko` failed
on every v0.61.x build. `MOTOKO_COMMIT` now points at fork PR #12 (`809921f0`), which migrates
motoko and keeps its journal format and chain digests byte-identical. See `MOTOKO.md` §19.
