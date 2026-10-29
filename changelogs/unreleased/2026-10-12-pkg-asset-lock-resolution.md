### Fixed — Locked package assets survive embedded execution and budget scopes

- `std/package.assetPath` retains the compiled package snapshot in runner and batch execution, and an engine-scoped snapshot in embed and served API calls. Missing locked targets return an actionable error naming the pinned version, without reading another cached version's data.
- Budget scopes preserve both versioned asset targets and existing directory-only embedding contexts. Package identities and relative paths reject cross-platform traversal.
- No-lock fallback retains the existing semantic-version ordering and `AILANG_PACKAGE_ROOT` support; a cache containing only invalid version names now returns an explicit error. The root package still resolves assets directly from its source directory, including before a lockfile exists. The basic CLI lock-first fix shipped earlier in v0.54.13; this change completes its runtime propagation.
