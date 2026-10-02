### Changed — motoko cloud executor image pins `sunholo/main-dst-20261002` (`de68fddf`)

`docker/Dockerfile.agent-motoko` moves from `4d4917cd` to `de68fddf`: upstream motoko main `4023bf08`
(strict extension loading, `motoko_ext_ailang_tools` with review fixes) plus our profiles with
`extensions.strict` on, the `ailang_only` lane extension, upstream PR #209, one schema per tool name
(duplicate tool schemas made strict OpenRouter providers return 400) and a portable path guard. The
dev plane builds it from `dev`; test and prod need a release and a promote. Supersedes the unmerged
`feat/motoko-image-pin-lane` branch (`780b9abd`).
