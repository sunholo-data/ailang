### Fixed — the agy executor ran an unpinned, self-updated binary

The executor and interactive `agy` shared `~/.local/bin/agy`. Interactive agy updates
the binary it runs from, so a session started 2026-11-24 08:52 replaced 1.5.15 with
1.5.16, and executor runs carried on unpinned: `ailang exec agy` (the mission lane
probe) never called `HealthCheck`, the only place the pin was checked.

- `tools/agy/install.sh` now installs the pinned binary at `<agy HOME root>/bin/agy`
  (`~/.ailang/state/agy/bin/agy`; `config.AgyBinFor`), a path no interactive session
  runs. `~/.local/bin/agy` is left to interactive use.
- The executor and the `google` quota observer prefer that path, then `PATH`, then
  `~/.local/bin`.
- Every agy run checks the pinned version before starting (cached per executor), so
  a wrong binary fails loudly on every path, not only where a caller remembered
  `HealthCheck`.

### Added — cloud-only agy sign-in

`tools/agy/signin.sh --profile cloud --upload <project> <secret>` signs a cloud holder
in under a throwaway HOME, uploads the token file as a new Secret Manager version
without printing it, and deletes the local copy (design_docs/planned/m-agy-cloud-executor.md,
Phase 0 P0a).
