### Added

- `std/result`: added stack-safe, order-preserving, short-circuiting `mapResult` and `collect` combinators.
- `std/math`: added `minInt`, `maxInt`, and contract-annotated `clampInt` integer helpers.
- `ailang docs search`: added `--stdlib` flag to search embedded `std/*.ail` modules, types, and functions offline, plus automatic unconfined fallback when local design docs and GitHub search are unavailable.
- `ailang run`: added static pre-flight capability check (`E_CAPS_MISSING`) before executing user code so missing capabilities fail cleanly with zero partial side effects.
- `ailang install`: added `--grant-new-caps` flag for explicit opt-in when reinstalling a `[bin]` command whose pinned capabilities widen.

### Changed

- `ailang check --debug-compile --json`: routed compilation phase timing tables to `stderr` so `stdout` remains strictly valid JSON.
- `ailang run --caps auto`: now discloses granted capabilities to `stderr` unless `--quiet` is passed explicitly, and excludes `Process` (pass `--caps auto,Process` to opt in).
- `[bin]` shims: `ailang install` resolves `[bin].caps = "auto"` once at install time, records `# caps:` in the shim header, pins explicit `--caps` flags, and refuses silent capability widening unless `--grant-new-caps` is given.
- Go module public surface: moved vendored `third_party/goopus` to `internal/third_party/goopus`, added `Internal` stability notices in `serveapi/...` `doc.go` files, and documented Go package and Internal binary split boundaries in `docs/docs/reference/stability.md`.
