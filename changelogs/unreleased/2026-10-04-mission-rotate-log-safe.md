### Changed — `mission rotate-log` resolves shared targets from the loaded registry, and `--status` is retired (2026-10-04)

`--status` rotated the STATUS archive and was never a report; it is now a hard error (D-FLEET-9,
ruled 2026-10-01) with no deprecation period — use `--stream status`. `rotate-log` and
`mission normalize` now resolve a shared-repo mission's design_docs from the registry they were
LOADED from (the explicit `AILANG_MISSION_REGISTRY` override wins over a CWD walk), instead of a
second walk that could rotate a stale clone. The legacy controller branch now exports
`AILANG_MISSION_REGISTRY` from the pinned driver root, as the binary branch already did.
