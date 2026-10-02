### Added — `ailang lock --check`: verify a committed lock without rewriting it (2026-10-02)

`ailang lock --check` re-resolves `ailang.toml` and compares with the committed `ailang.lock`
(ignoring timestamp/generator/ailang_version), listing each drifted entry and exiting non-zero.
Because path dependencies are relative, the verdict is identical in every checkout of the same
tree, so it is safe as a CI gate. New `pkg.CheckLock` / `pkg.LockDrift` / `pkg.ResolveLock`
back it; `ailang lock` and `EnsureLock` share `ResolveLock`. (#1498)

### Fixed — path dependencies are portable across checkouts end to end (2026-10-02)

`{ path = "../sim" }` dependencies are recorded in `ailang.lock` relative to the depending
package's `ailang.toml`, with forward slashes on every OS (the writer landed in v0.32; the
read side and migration were incomplete). Now: the loader converts slashes before joining; the
content hash names files slash-separated so a lock written on Windows validates on Linux; and a
lock from an older ailang that names a path dep by absolute directory still loads while that
directory exists, is reported as drift by `lock --check`, is rewritten relative by the next
`ailang lock`, and — when the directory is gone — fails with an error that names the stale
absolute path and says to run `ailang lock` (previously a bare "package directory not found").
A two-package repo (`sim/`, `ai/`) locked in one checkout now type-checks and runs from a copy
at another absolute path with the original deleted. Docs updated
(`build-a-motoko-extension.md`, `extension-packages.md`, `packages.md`,
`package-publishing.md`, ailang-packages skill). Fixes #1498.
