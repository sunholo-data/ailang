### Fixed

- `std/package.assetPath` now uses the running program's locked dependency directory, including registry, path, and git sources. Runner, batch, embed, and served API calls retain the selected targets. Missing locked installs return an actionable error without reading another version's data.
- No-lock/per-package fallback now selects the compatibility-semver maximum (`0.10.0` over `0.9.0`), skips invalid version directories, and breaks numeric ties deterministically. This intentionally changes the previous lexical-order expectation. REPL and root-package asset lookup retain cache fallback; fallback still ignores `AILANG_PACKAGE_ROOT`.
