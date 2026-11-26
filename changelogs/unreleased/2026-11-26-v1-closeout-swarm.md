### Breaking — Remove 16 pre-1.0 `@deprecated` stdlib aliases (2026-11-26)

Removed all 16 pre-1.0 `@deprecated` stdlib exports across `std/math`, `std/string`, `std/bytes`, `std/array`, `std/io`, `std/rand`, `std/list`, and `std/fs` (`abs_Float`, `abs_Int`, `charAt_or`, `slice`, `unsafeGet`, `printErr`, `rand_int`, `rand_float`, `rand_bool`, `rand_seed`, `nth_or`, `head_or`, `last_or`, `member`, `readFileBytes`, `glob`) and re-froze `.stdlib-golden/*.json` (ROADMAP Step 16, `design_docs/planned/v1_0_0/m-v1-closeout-swarm.md`).
- `internal/importhint/renames.go` is preserved so importing any removed name continues to emit structured `IMP010` errors naming the exact replacement (and argument-order note for `std/list.member` -> `std/list.contains`).
- `tools/migrate-stdlib-renames.sed` remains available and tested for migrating 0.x codebases.

### Changed — Harden release pipeline and Go semver for `v1.0.0-rc.*` pre-releases (2026-11-26)

Prepared the release automation, docs version scripts, and Go semver ordering for `v1.0.0-rc.1` (ROADMAP Step 15):
- `.github/workflows/release.yml` now accepts `v*.*.*-rc.*` tags, sets `make_latest: false` on pre-releases, skips `mcp-registry` publishing on pre-releases, and diffs release notes from the previous final tag (`--exclude='*-*'`).
- `docs/scripts/generate-version-constants.sh` and `.github/workflows/docusaurus-deploy.yml` filter out hyphenated pre-release tags so `v1.0.0-rc.*` never overwrites `STABLE_RELEASE` / `AILANG_VERSION` (verified by `tools/ci/test_rc_version_filter.sh`).
- `internal/effects/pkg.go` now implements SemVer 2.0.0 §11.4 numeric pre-release ordering (`1.0.0-rc.2 < 1.0.0-rc.10 < 1.0.0`), with expanded tests in `internal/effects/pkg_semver_test.go` and `internal/pkg/version_compat_test.go`.

### Docs — 1.0 reference sync, migration/security guides, and Bar `G5` close-out (2026-11-26)

- Synced `docs/docs/reference/language-syntax.md` and `docs/docs/reference/stdlib.md` with `v0.61`–`v0.63` features (`|>`, typed holes, unbraced multi-statement branches, `deriving (Json)`, `expr?`, lambda/record destructuring, stdlib combinators).
- Added `docs/docs/guides/upgrading-to-1-0.md`, `docs/docs/reference/security-model.md`, and `docs/docs/why-1-0.mdx`, wired into `docs/sidebars.js`, and updated `SECURITY.md` and `docs/docs/reference/stability.md` with the 1.x support window (`D-67`).
- Closed Bar `G5` (`N = 0` open v1.0.0 gate items) with refreshed post-merge simplicity metrics and ratified `D4` waivers in `design_docs/planned/v1_0_0/g5-simplicity-metrics-fragment.md`.

### Fixed — CI hygiene timeout assertion, REPL stdlib loader test, and Windows diagnostic path regex (2026-11-26)

- Updated `internal/cihygiene/workflow_go_test_timeout_test.go` to match the `660s` Linux `go test -timeout` budget in `.github/workflows/ci.yml`.
- Updated `TestLoadRealStdlibFromFiles` in `internal/repl/module_registry_stdlib_test.go` to preload `std/list` before `std/string` (required since `std/string.lines` imports `std/list (map)`).
- Updated `strConcatLocRe` (`internal/quickfix/string_concat.go`) and `embeddedLocRe` (`internal/lsp/diagnostics.go`) to accept Windows drive-letter prefixes (`C:\...`) in embedded diagnostic locations.
