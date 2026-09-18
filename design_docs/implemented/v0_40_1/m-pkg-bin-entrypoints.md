# M-PKG-BIN-ENTRYPOINTS: `[bin]` — install a CLI from an AILANG package

**Status**: Implemented
**Target**: v0.40.1 (designed against v0.40.0; the release cut underneath the PR)
**Priority**: P0
**Estimated**: 1.5 days
**Dependencies**: None (builds on M-PKG-PACKAGE-SYSTEM v0.9.5, the ailang#671 package-search fix, `--caps auto`)
**Source**: message `inbox_1789450729585_d0020028` (email-parse, 2026-09-15, "packages cannot ship a runnable CLI
entrypoint (no [bin]; installed files cannot resolve deps)") and
`ailang-parse/design_docs/planned/v0_40_0/v0_40_0_cli_install_without_clone.md` (docparse, 2026-09-04) — two
consumers, same shape, both currently reinventing a runtime-directory installer.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | A bin pins the exact package version AND a lock generated from that version's manifest; the shim is regenerated from those two inputs only |
| A2: Replayability | 0 | No trace change; a bin run is an ordinary `ailang run` |
| A3: Effect Legibility | +1 | The shim runs with `--caps auto` (inferred from the entrypoint's declared effects) or the caps the manifest declares — nothing ambient |
| A4: Explicit Authority | +1 | Caps stay explicit; `[bin]` cannot widen `[effects].max` |
| A5: Bounded Verification | +1 | Publish verifies each bin module resolves to a file that exports the entry — before the tarball exists |
| A6: Safe Concurrency | 0 | No change |
| A7: Machines First | +1 | An agent can `ailang install vendor/name` and have the command, instead of parsing a README that says "clone" |
| A8: Minimal Syntax | 0 | One TOML table; no language syntax |
| A9: Cost Visibility | 0 | No change |
| A10: Composability | +1 | Reuses `ResolveDependencies`/`NewLockFile`/`ResolveModuleToFile`/`PackageDir`; no new resolution path |
| A11: Structured Failure | +1 | Missing module, missing entry, bad name, and "not on PATH" each fail loudly at publish/install, not at first run |
| A12: System Boundary | 0 | No new boundary crossing |

**Net Score: +8** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

## Problem Statement

An AILANG package can ship a library but not a command. Two consumers have hit the same wall
independently and built the same workaround:

- **sunholo/email (eparse)** — 13 exported modules, 14 CLI entrypoints. Consumers who
  `ailang install sunholo/email` get the library and are told to clone the repo. Its installer
  now materialises `~/.local/share/eparse/runtime/` with a generated `ailang.toml`, a lock, and
  copies of the entrypoints, then runs everything with cwd set there.
- **sunholo/ailang_parse (docparse)** — same shape: `bin/docparse` walks up to find
  `docparse/main.ail`, and a planned `curl | sh` installer unpacks the registry tarball into a
  prefix and runs `ailang lock` there.

Both are reimplementing, per package, the thing the package manager should do once.

Two verified blockers (Verification Log V1–V4):

1. **No entrypoint concept in the manifest.** `ExportConfig` has only `Modules`. A package cannot
   say "this module is a command".
2. **A file inside an installed package cannot be run.** The cache holds `ailang.toml` and
   `*.ail` but no `ailang.lock` (excluded from the tarball by design). Package search is anchored at
   the source file's directory (ailang#671), so the cached package's own manifest is found, its lock
   is not, and every external import fails with `not found in ailang.lock; run 'ailang lock'` — in a
   directory the user is not meant to write to. Even with a lock present, `MOD010` rejects the
   canonical module path because `ailang run` never sets `PackageDir` (V4).

## Goals

**Primary**: `ailang install vendor/name` on a package that declares `[bin]` puts a working command
on the user's PATH. No clone, no wrapper script, no per-package installer.

**Success metrics**:
1. `ailang install sunholo/email` → `eparse --version` works from any cwd (once eparse publishes
   with `[bin]`).
2. The same manifest table works for a local checkout: `ailang install --path packages/email`
   shims the developer's working tree, so a bin can be tested before it is published.
3. Publishing a package whose `[bin]` names a module that does not exist, or whose entry is not
   exported, is refused before a tarball is built.
4. Zero new resolution logic: the shim is `ailang run --package-dir <pkg> <file> -- "$@"`.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: The lock for a bin is written INTO the cached package directory at install time | Makes the cached package a self-sufficient program root; touches the "cache is read-only" convention | agent (design below) | design | low |
| D2: Shims are generated shell/`.cmd` files that `exec` the installing `ailang` binary, not an `ailang bin run` indirection | One fewer lookup per invocation; a shim is regenerable from (package, version) | agent | design | low |
| D3: `ailang run` gains `--package-dir` (exposes the existing `pipeline.Config.PackageDir`) | Also unblocks email-parse's option (b) for anyone running files out of the cache by hand | agent | design | low |
| D4: Caps default to `auto` (inferred from the entrypoint); `[bin]` may pin them | `--caps auto` already exists and is the least-authority default | agent | design | low |
| D5: Shim location is `~/.ailang/bin/` (override `--bin-dir`); no new env var | Keeps the config registry unchanged; the dir is printed with a PATH hint when missing | agent | design | low |

### Design Freeze

No "high" change-cost decisions. Every row is agent-resolvable and resolved in this document.

**Quorum triggers** (design-doc-creator §Quorum): no freeze items; no shared machinery overridden
(every Conflict-Surface row is "reuse"); no cost/KPI/banked-schema surface; every premise verified
in-repo (V1–V12). **All four false → quorum skipped.**

## Solution Design

### Overview

A package declares commands in `ailang.toml`:

```toml
[bin]
eparse = "cli"                                   # module sunholo/email/cli, entry main, caps auto
docparse = { module = "docparse/main", entry = "main", caps = "IO,FS,Env,AI",
             run_flags = ["--max-recursion-depth", "50000", "--process-timeout", "5m"] }
```

`ailang install vendor/name` then:

1. downloads + verifies + extracts to `~/.ailang/cache/registry/<vendor>/<name>/<ver>/` (unchanged);
2. if the manifest has `[bin]`: resolves the package's own dependencies and writes `ailang.lock`
   **beside the cached manifest** (D1) — the package is now a program root;
3. verifies each bin's module resolves to a file that exports the entry;
4. writes one shim per bin to `~/.ailang/bin/<name>` (D5):

```sh
#!/bin/sh
# ailang-bin: sunholo/email@0.3.3 eparse
# regenerate: ailang install sunholo/email@0.3.3
exec "/Users/x/go/bin/ailang" run --quiet --package-dir "/Users/x/.ailang/cache/registry/sunholo/email/0.3.3" \
  --entry main --caps auto "/Users/x/.ailang/cache/registry/sunholo/email/0.3.3/cli.ail" -- "$@"
```

5. prints the shims written and, if `~/.ailang/bin` is not on `$PATH`, the one line to add.

`ailang install --path <dir>` does 2–5 against a local package directory (the developer loop).
`ailang bin list` reads the `# ailang-bin:` header of each shim; `ailang bin uninstall <name>`
removes one.

### Architecture

```
ailang.toml [bin] ──LoadManifest──▶ BinConfig{name → BinSpec{Module, Entry, Caps, RunFlags}}
                                        │ Validate(): name shell-safe, module non-empty
                                        ▼
ailang publish ──▶ pkg.VerifyBinEntrypoints(dir, manifest)     (module file resolves, entry exported)
ailang install ──▶ extract ─▶ ensurePackageLock(cacheDir) ─▶ VerifyBinEntrypoints ─▶ writeShims
                                        │                                      │
                          ResolveDependencies + NewLockFile.Save        ~/.ailang/bin/<name>
                                                                               │
                                              exec ailang run --package-dir <cacheDir> <file> -- "$@"
                                                                               │
                                              runner.Options.PackageDir → pipeline.Config.PackageDir
                                              (lock found beside manifest; MOD010 uses package layout)
```

**Why the lock lives in the cache (D1).** The lock is a pure function of the cached manifest's pinned
dependencies. `ContentHash` covers only `.ail` files (V7), so writing `ailang.lock` there changes no
hash any consumer validates. `ResolveDependencies` fetches missing transitive packages into the same
cache (V8), so this is exactly what `ailang lock` would do if the cache were writable — it is, by
`ailang install`, which already writes there. Nothing else reads a lock out of the cache, so a
library consumer is unaffected. The alternative — a per-bin prefix with copies — is what both
consumers built and is the duplication this design removes.

**Why a shim, not `ailang bin run` (D2).** An indirection would need a bin registry file and a
second resolution at every invocation. The shim carries everything in its own header; `list` and
`uninstall` parse that header, so there is one source of truth (the shim file) and no registry to
drift.

**Why `--package-dir` (D3).** `pipeline.Config.PackageDir` already exists for `ailang test`,
`verify --package`, `check --package` (V4): it anchors lock lookup and lets `MOD010` validate
against the package layout. `ailang run` is the one entry point that never sets it. Exposing it
also answers the email-parse report's option (b) for anyone running cached files by hand.

**The one behaviour change outside new code:** the runner prints `auto-granted capabilities: …`
to stderr on `--caps auto` regardless of `--quiet` (V9). A CLI's stderr must not carry this on
every invocation; the message is gated on `!opts.Quiet`. `--quiet` already suppresses the sibling
progress lines, so this is the flag doing what it says.

### Conflict Surface

Not a parser/typechecker change, but the design touches `cmd/ailang/main_run*.go` and the runner, so
the surface is enumerated anyway.

| Position | What already lives there | Decision |
|---|---|---|
| `pipeline.Config.PackageDir` | Set by named tests, `verify --package`, `check --package`; drives `loaderBaseDir`, lock search, MOD010 layout check | **reuse** — `run --package-dir` sets the same field; empty = today's behaviour |
| `packageSearchDir(explicit, base, entry)` | explicit wins → entry dir → base | **reuse** unchanged |
| `ailang run` positional args after the file | `programArgs`, leading `--` stripped once | **reuse** — the shim always inserts `--` so user args that look like flags reach `getArgs()` |
| `[exports].modules` validation | modules must start with package name or `module_prefix` | **reuse** — bin modules are validated by resolving with `ResolveModuleToFile`, which accepts the same prefixes |
| `CreateTarball` include set | `ailang.toml`, `*.ail`, `AGENT.md`, `CHANGELOG`, `assets/**`; lock excluded | **unchanged** — bin modules are `.ail` so they already ship; the lock is generated at install, not published |
| `~/.ailang/cache/registry/<v>/<n>/<ver>/` | written by `install`, `lock` (transitive fetch); read by the loader | **extend** — `install` may now also write `ailang.lock` here for `[bin]` packages |
| `ailang install` in a consumer project | appends the dep to `ailang.toml` | **unchanged** — shims are written in addition, never instead |
| stderr under `--quiet --caps auto` | `auto-granted capabilities:` line | **change** — suppressed under `--quiet` |

Programs that MUST still work post-change (regression fixtures, all exist — V10):
- `ailang run examples/hello_world.ail` (no package, no flag)
- `ailang run --caps auto <file> -- a b` still yields `getArgs() == [a, b]`
- `ailang test` named tests (already pass `PackageDir`)
- `ailang check --package` on `sunholo/deontic`-shaped flat tarballs (the MOD010 layout path)

### Implementation Plan

**Phase 1 — manifest + verification (core, ~0.5 day)**
1. `BinConfig map[string]BinSpec` on `PackageManifest` (`toml:"bin"`); `BinSpec.UnmarshalTOML` accepts
   a string (module) or a table (`module`, `entry`, `caps`, `run_flags`), mirroring `Dependency`.
2. `Validate()`: bin name matches `^[a-z0-9][a-z0-9._-]*$`; module non-empty; entry defaults `main`;
   `run_flags` must not contain `--caps`, `--entry`, `--package-dir` (those are the shim's).
3. `pkg.VerifyBinEntrypoints(dir, manifest) error`: for each bin, `ResolveModuleToFile` must find a
   file, and that file must contain `export func <entry>(`. Called by `publish` (next to
   `VerifyDeclaredAssets`) and by `install`.

**Phase 2 — `run --package-dir` + quiet (~0.25 day)**
4. `runner.Options.PackageDir` → `pipeline.Config.PackageDir`; `ailang run --package-dir` flag.
5. Gate `auto-granted capabilities:` on `!opts.Quiet`.

**Phase 3 — install/shims (~0.75 day)**
6. `internal/pkg/bin.go`: `EnsureLock(pkgDir)` (skip when a valid lock exists; else
   `ResolveDependencies` + `NewLockFile.Save`), `ShimPath(binDir, name)`, `WriteShim(binDir, ailangBin,
   pkgDir, manifest, name, spec)`, `ListShims(binDir)`, `RemoveShim(binDir, name)`. Shim is POSIX `sh`
   (mode 0755); on Windows a `.cmd` with `%*`.
7. `ailang install`: after extract, if `len(manifest.Bin) > 0` → EnsureLock → VerifyBinEntrypoints →
   WriteShim each → PATH hint. New flags: `--path <dir>` (local package, same steps against that dir),
   `--bin-dir <dir>` (default `~/.ailang/bin`), `--no-bin` (library only).
8. `ailang bin list | uninstall <name>` — one new top-level route.

### Files to Modify/Create

- `internal/pkg/manifest.go` — `BinConfig`/`BinSpec`, `UnmarshalTOML`, `Validate()` rules (+60)
- `internal/pkg/manifest_test.go` — string/table forms, bad names, reserved run_flags (+80)
- `internal/pkg/bin.go` — NEW: lock ensure, shim write/list/remove, entry verification (~180)
- `internal/pkg/bin_test.go` — NEW: shim round-trip, header parse, verify-entry positives/negatives (~150)
- `internal/runner/run.go` — `Options.PackageDir` → `cfg.PackageDir`; quiet gate on auto-caps (+4)
- `cmd/ailang/main_run.go` — `--package-dir` flag threaded to `runFile` (+4)
- `cmd/ailang/main_run_exec.go` — pass through to `runner.Options` (+2)
- `cmd/ailang/pkg_install.go` — `--path`, `--bin-dir`, `--no-bin`; bin steps after extract (+90)
- `cmd/ailang/pkg_install_test.go` — install --path on a fixture package writes a working shim (+80)
- `cmd/ailang/pkg_bin.go` — NEW: `ailang bin list|uninstall` (~90)
- `cmd/ailang/pkg_publish.go` — `VerifyBinEntrypoints` beside `VerifyDeclaredAssets` (+4)
- `cmd/ailang/main.go` — `case "bin"` (+6)
- `docs/docs/guides/packages.md` (or the packages reference) — `[bin]` section (+40)
- `changelogs/v0.32-current.md` — entry

## Examples

### Example 1: the package author (email-parse)

```toml
# packages/email/ailang.toml
[bin]
eparse = "cli"          # sunholo/email/cli exports main
```

```bash
$ ailang install --path packages/email        # developer loop, no publish
✓ Lock: packages/email/ailang.lock (1 package)
✓ bin: eparse → ~/.ailang/bin/eparse  (sunholo/email@0.3.3, module sunholo/email/cli)
$ eparse --version
eparse 1.0.1
```

### Example 2: the consumer (Daneel's Studio account)

```bash
$ ailang install sunholo/email
Resolved sunholo/email@latest → sunholo/email@0.3.4
✓ Downloaded sunholo/email@0.3.4 (188213 bytes)
✓ Lock: ~/.ailang/cache/registry/sunholo/email/0.3.4/ailang.lock (1 package)
✓ bin: eparse → ~/.ailang/bin/eparse
⚠ ~/.ailang/bin is not on your PATH. Add:  export PATH="$HOME/.ailang/bin:$PATH"
```

### Example 3: publish refuses a broken bin

```bash
$ ailang publish --dry-run
Error: bin "eparse": module sunholo/email/cli resolves to cli.ail but it does not export func main
```

## Success Criteria

- [x] `[bin] name = "module"` and the table form both load; `ailang.toml` without `[bin]` is unchanged
- [x] `ailang install --path <fixture with [bin]>` writes an executable shim; running it from `/`
      prints the fixture's output and receives its positional args
- [x] `ailang install <registry pkg with [bin]>` writes `ailang.lock` into the cache dir and a shim
      (exercised against a temp `AILANG_REGISTRY` fixture in tests, not the live bucket)
- [x] `ailang run --package-dir <cache dir> <cache dir>/_smoke.ail` runs sunholo/email's smoke without
      `AILANG_RELAX_MODULES` once a lock is beside it (today: V3/V4 failures)
- [x] `ailang publish --dry-run` refuses a `[bin]` whose module is missing or whose entry is not exported
- [x] `ailang bin list` shows package@version per shim; `ailang bin uninstall` removes exactly one
- [x] `--quiet --caps auto` prints nothing to stderr on success
- [x] `make test-core`, `go test ./internal/pkg/... ./internal/runner/... ./cmd/ailang/...` green;
      `make simplicity-audit` shows +1 command route (`bin`) and no new env var
- [x] Changelog + package docs updated

## Testing Strategy

- **Unit** (`internal/pkg`): manifest forms; name validation; `VerifyBinEntrypoints` positive and the
  two negatives; shim write → parse header → list → remove; `EnsureLock` idempotent when a lock exists.
- **Integration** (`cmd/ailang`): fixture package under `testdata/` with `[bin]`, `install --path` into a
  temp bin dir, then `exec` the shim with args and assert stdout. Mutation check: delete the fixture's
  `export func main` and assert install refuses (feedback: mutation-test your own tests).
- **Manual on dev**: `ailang install --path ~/dev/sunholo-data/email-parse/packages/email` after adding
  `[bin]` there; run `eparse` from `/`.

## Deferred Decisions

- Whether `ailang install` should write a lock into the cache for **every** package (not just `[bin]`
  ones) — agent may extend later if a second consumer needs to run cached files by hand; today the
  `--package-dir` flag plus a hand-run `ailang lock` in a copy covers it.
- Shim tracing defaults (docparse sets `AILANG_NO_TRACE=1` for speed) — left to `run_flags`/env; the
  shim inherits the caller's environment unchanged.
- `tar.Header.Mode` plumbing so `assets/bin/*` ships executable (docparse V7) — separate, tiny; not needed
  once the shim exists.

## Non-Goals

- **A single self-contained binary** — the shim needs `ailang` on the machine; bundling the runtime is
  a different project.
- **Version switching / multiple installed versions of one bin** — re-running `install` rewrites the
  shim; side-by-side is what the cache already gives, and `install vendor/name@x.y.z` picks one.
- **Non-`.ail` payloads** (docparse's Python adapter) — that is `assets/`, already shipped.
- **Windows shim testing in CI** — a `.cmd` is written; verified by inspection until a Windows runner
  exercises it.

## Timeline

**Day 1** (6h): Phase 1 + Phase 2 with tests.
**Day 2** (5h): Phase 3, integration test, docs, changelog, manual run against email-parse.

**Total: ~11 hours**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Writing a lock into the cache surprises a tool that assumes the cache holds only tarball contents | Low | Only `.ail` files are hashed (V7); the tarball is never rebuilt from the cache; grep showed no reader of a cache-resident lock |
| A package's `[bin]` module imports a dep with a **range** version, so the cache lock differs from a consumer's | Low | Manifest deps are exact today (install rejects ranges, V11); `EnsureLock` resolves exactly what `ailang lock` in the source dir would |
| Shim pins the absolute `ailang` path; user moves/reinstalls the binary elsewhere | Low | Header says how to regenerate; `ailang bin list` flags a shim whose interpreter path no longer exists |
| `--caps auto` under-grants for an entry that delegates effects through a callback | Low | `[bin]` `caps` pins them explicitly; the auto path is the same one `ailang run --caps auto` users get today |
| Name collision with an existing command on PATH (`eparse` already installed by the repo's launcher) | Med | Install prints the shim path and does not touch anything outside `bin-dir`; PATH order is the user's |

## Verification Log

All rows checked on 2026-09-17 against HEAD `62bf3993b` (AILANG v0.39.5-19).

| # | Claim | Evidence |
|---|---|---|
| V1 | No entrypoint concept in the manifest | `internal/pkg/manifest.go:142-144` — `ExportConfig{Modules []string}` only; `grep -rn '"bin"' cmd/ailang internal/pkg internal/config` → empty |
| V2 | The cached package has no lock, and it is excluded from the tarball | `ls ~/.ailang/cache/registry/sunholo/email/0.3.2/` → `AGENT.md ailang.toml *.ail`; `tarball.go:20-22` "Excludes … ailang.lock" |
| V3 | Running a cached file fails on its own deps even when the consumer's lock has them | scratch consumer: `ailang lock` → 2 packages incl. `sunholo/duckdb@0.1.3`; `ailang run --caps IO,Process,Env,FS --entry main ~/.ailang/cache/registry/sunholo/email/0.3.2/_smoke.ail` → `package "sunholo/duckdb" not found in ailang.lock` |
| V4 | With a lock beside the package, deps resolve but MOD010 rejects the canonical module because `run` never sets PackageDir | copy of the cache dir + `ailang lock`, then `ailang run … _smoke.ail` from `/` → `Error MOD010: module 'sunholo/email/_smoke' doesn't match file path …`; `pipeline_module.go:266` short-circuits MOD010 only when `cfg.PackageDir != ""`; `grep -n PackageDir internal/runner/*.go` → empty |
| V5 | Package search is anchored at the source file's dir, explicit `PackageDir` wins | `internal/pipeline/package_resolver.go:170-179` `packageSearchDir` |
| V6 | `ResolveModuleToFile` accepts flat root, `src/`, and canonical layouts | `internal/pkg/discover.go:139-165` |
| V7 | `ContentHash` covers only `.ail` files, so a cache-resident lock changes no hash | `internal/pkg/hasher.go:14-31` |
| V8 | `ResolveDependencies` fetches missing registry deps into the cache itself | `internal/pkg/resolver.go:262-277` `CachedPackagePath` → `FetchPackage` → `ExtractTarball` |
| V9 | `auto-granted capabilities:` is printed to stderr regardless of `--quiet` | `internal/runner/run.go:248-256`; `ailang run --quiet --caps auto argtest.ail` → line on stderr; positional args reach `getArgs()`: `-- --triage 5 "x y"` → `[--triage, 5, x y]` |
| V10 | Regression fixtures exist | `examples/hello_world.ail` present; named tests pass `PackageDir` (`package_resolver.go:164` comment, `verify_package.go:89`) |
| V11 | Manifest dependency versions are exact | `pkg_install.go` rejects `^~><=`; `appendDependencyToFile` writes the resolved exact version |
| V12 | `bin` is an unallocated top-level command; `~/.ailang/bin` is unused | `grep -n 'case "bin"' cmd/ailang/main.go` → empty; `grep -rn '\.ailang/bin' cmd internal` → empty |
| V13 | Publish has a pre-tarball hook for manifest-declared files | `cmd/ailang/pkg_publish.go:94-97` `pkg.VerifyDeclaredAssets` — `VerifyBinEntrypoints` goes beside it |
| V14 | No error code namespace needed | Failures are `install`/`publish` command errors (exit 1 with message), same as `VerifyDeclaredAssets`; no PUB/MOD code allocated |

## Related Documents

**Consumer-side (the two requests):**
- email-parse message `inbox_1789450729585_d0020028` (2026-09-15) — proposes `[bin] eparse = "cli"` (option a),
  CWD-lock resolution (b), lock in tarball (c). This design is (a) with `--package-dir` covering (b).
- `ailang-parse/design_docs/planned/v0_40_0/v0_40_0_cli_install_without_clone.md` — docparse's installer;
  Phases 2 and 4 there collapse to `[bin]` + `ailang install` once this ships (assets/ handling stays).

**Implemented (inform design):**
- [design_docs/implemented/v0_10_0/m-dx-package-check.md](../../implemented/v0_10_0/m-dx-package-check.md) (0.45) — `--package` anchoring
- [design_docs/implemented/v0_9_5/m-pkg-package-system-sprint-plan.md](../../implemented/v0_9_5/m-pkg-package-system-sprint-plan.md) (0.42)
- [design_docs/implemented/v0_10_0/m-pkg-metadata-urls.md](../../implemented/v0_10_0/m-pkg-metadata-urls.md) (0.42)

**Planned (distinct):**
- [design_docs/planned/m-package-test-discovery.md](../m-package-test-discovery.md) (0.40) — test discovery, not entrypoints
- [design_docs/planned/v1_1_0/m-pkg-inflight.md](../v1_1_0/m-pkg-inflight.md) (0.40) — publish status
- daneel's "dependency override (replace)" request (`inbox_1789449531924_c0b6df4d`) — orthogonal; `install --path` here is
  the developer loop for bins, not a replace mechanism

## Future Work

- `ailang install` writing a lock for every cached package (see Deferred).
- `tar.Header.Mode` for executable assets.
- Homebrew/`curl | sh` bootstrap that installs `ailang` then runs `ailang install vendor/name` — trivial once this exists.

---

**Document created**: 2026-09-17
**Last updated**: 2026-09-18

## Implementation Report

**Shipped on dev 2026-09-18:** PR #1256 (`1adc48285`, six commits incl. three rebases over
M-V1-SIMPLIFY-S5 M1–M4), then `68bb6d6e7` (field fix) and `b1e1099c3` (prompt + docs).

### What was built

Exactly the design, with two deviations forced by dev moving underneath:

- `bin` is a **`pkg` verb** (`ailang pkg bin list|uninstall`) with the bare `ailang bin` spelling
  generated from the same row (`pkgLegacyTopLevel`) — S5 M2 turned the registry verbs into one table
  while the PR was open. The design's `case "bin"` in `main.go` no longer exists as a shape.
- `editDistance` for did-you-mean became optimal-string-alignment: adding a three-letter route
  tied `chian` between `chains`, `bin` and `check` at 3 under plain Levenshtein.

### Field fix (email-parse, 2026-09-18, same day)

The shim passed MOD010 **only from `/`**: `CanonicalModuleID` strips the leading `/` of the absolute
entry path and `declaredModuleMatchesPackageLayout` then `filepath.Abs`'d the stripped id at the
caller's cwd. Two test gaps let it through — the manual E2E had been run from `cd /`, and the Go E2E's
package sat in `t.TempDir()` where the temp-path relaxation carried it silently. Fixed by comparing
`ast.File.Path` (what the loader read) with the reattached-slash form as fallback; pinned by
`TestDeclaredModuleMatchesPackageLayout_AbsoluteEntryFromAnyCwd` (mutation-checked) and an
empty-stderr assertion in `TestInstallPath_ShimRunsFromAnyCwd`. Verified end to end by the
reporter (email-parse) against `packages/email` from `/private/tmp`, `$HOME` and the repo.

### Code locations

- `internal/pkg/bin.go` (+320) / `bin_test.go` (+330): name validation, `ResolveBinFile`,
  `VerifyBinEntrypoints`, `EnsureLock`, `WriteShim`/`shimBody`/`shimPath` (goos-parameterised),
  `ReadShim`/`ListShims`/`RemoveShim`
- `internal/pkg/manifest.go` (+90): `BinSpec`, string/table `UnmarshalTOML`, `Validate()` rules
- `cmd/ailang/pkg_bin.go` (+230) / `pkg_bin_test.go` (+320): `install --path`, `installBinsTo`,
  `reportPathStatus`, `binCommand`; fake-registry install test; shim executed from another cwd
- `cmd/ailang/pkg_install.go`, `pkg_publish.go`, `commands_pkg.go`, `commands_groups.go`,
  `main_run.go`, `main_run_exec.go`, `repl.go`, `commands.go` (edit distance)
- `internal/runner/run.go`: `Options.PackageDir`; `--quiet` gates the auto-caps line
- `internal/pipeline/pipeline_module.go` (+30) / `package_layout_test.go` (+70): the field fix
- Docs: `docs/docs/guides/packages.md` §Commands, `docs/docs/packages/index.mdx`,
  `prompts/v0.16.6.md` §Shipping a command, `.claude/skills/ailang-packages/{SKILL.md,resources/manifest_reference.md}`

### Success criteria — all met

Every box in §Success Criteria is ticked; the two field-found gaps became the two tests above.

### Known limitations

- Windows `.cmd` shim is rendered and parsed in tests but has not been executed on a Windows runner.
- Sonar's new-coverage gate counts `internal/` only; `cmd/ailang/pkg_bin.go` coverage is real
  (~85%) but invisible to it.
- `ailang install` writes a cache-resident lock only for `[bin]` packages (deferred decision kept).

### Not done here (by design)

- eparse / docparse adding `[bin]` to their manifests and publishing — their repos.
- `tar.Header.Mode` for executable `assets/` (docparse V7).
- A release: consumers on released binaries (Daneel's account installs from tags) need v0.40.1.
