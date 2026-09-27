# Sprint plan: M-STDLIB-ROOT-RESOLUTION

**Design doc**: [m-stdlib-root-resolution.md](m-stdlib-root-resolution.md)
**Progress file**: `.ailang/state/sprints/sprint_M-STDLIB-ROOT-RESOLUTION.json`
**Estimate**: about 1.5 agent-days, 3 milestones, ~900 LOC changed (about half tests, and a net deletion because `internal/module` goes).
**Risk**: medium. The loader precedence changes (two approved behaviour changes: explicit overrides beat `./std`; a partial `./std` errors).

## Premises re-verified against origin/dev (2026-09-26)

| Doc claim | Live check |
|-----------|-----------|
| `docs` has a private resolver with no embedded copy | `cmd/ailang/docs.go` `findStdlibDir` (AILANG_STDLIB_PATH, `./std`, `../std`), called before the `prelude` branch |
| Loader falls back to `std.FS` per module, in two places | `loader.go` `Load` and `resolvePath` |
| `--stdlib-path` is an `os.Setenv`, tier 4 below `./std` | `runner/run.go`, `_ = opts.TraceLoader`, `_ = opts.StrictVersion` |
| `stdlibindex` reads env then literal `std` | `internal/stdlibindex/index.go` `stdlibDir()` |
| `internal/module` has no importer | `go list` shows none; it is the only reader of `AILANG_STDLIB` **and** `AILANG_PATH` |
| embed engine sets `AILANG_STDLIB_PATH` to the project root | `internal/embed/embed.go` `New` |

**Additional findings the doc did not list**

- The run flag is spelled `--strict` (help: "Fail on stdlib version mismatch"), not `--strict-version`; the Options field is `StrictVersion`. Both it and `--trace-loader` get wired.
- There are three `loader.NewModuleLoader` construction sites (pipeline, runtime, elaborate), so threading a per-loader `ConfigureStdlibResolver` from the runner would touch all of them. **Decision:** the runner configures the process once through `stdlibroot.Configure(Options{Override, Trace, StrictVersion})`; every loader resolves through it. `ConfigureStdlibResolver` (which never had a caller) is deleted rather than given one.
- Memoisation is keyed on the inputs (override, `AILANG_STDLIB_PATH`, cwd). In a real process those are fixed, so the root is resolved exactly once; tests that change env or cwd get a fresh resolution instead of a stale one.
- Many tests set `AILANG_STDLIB_PATH` to the **repo root** (copying the embed engine's R5 behaviour). Under the explicit-override rule those become errors, so they are pointed at `<root>/std` or cleared.
- `AILANG_PATH` is, like `AILANG_STDLIB`, read only by the dead `internal/module`; both registry rows go.

## M1: `internal/stdlibroot` and the loader on top of it (0.5 day)

- New leaf `internal/stdlibroot` (imports `config`, `std`): `Root{FS, Dir, Source, Tried}`, `Resolve(override)`, `Configure`, `DisplayPath`, the moved `userDataDir`. Order: flag, env list, `./std`, `<bin>/../std`, user data dir, system dirs, embedded. Marker `io.ail`. An explicit flag or env that resolves nowhere is an error naming every tried path.
- Loader: `StdlibResolver` resolves against the one root; module missing from it is an error naming the root (no per-module fallback). The two `std.FS` fallbacks collapse. Version check applies to on-disk roots; `--strict` makes it fatal. Trace prints the root source and tried list.
- Runner: `stdlibroot.Configure` replaces `os.Setenv`; `TraceLoader`/`StrictVersion` wired.
- R5: drop the `Setenv` in `internal/embed`. R6: executor and eval harness pass a root only when `<dir>/io.ail` exists.
- R4: delete `internal/module`; drop `AILANG_STDLIB` and `AILANG_PATH` rows; `make docs-env`.
- **Acceptance**: table tests pin the order; shadow `./std/list.ail` + `--stdlib-path` gives the real `map`; a partial `./std` errors naming the root; bogus flag errors listing `Tried`.

## M2: `ailang docs` and import hints (0.5 day)

- Docs readers take `(fs.FS, name)`; `findStdlibDir`/`isStdlibDir` deleted; resolution after the `prelude` branch; error only for a bad explicit override.
- `stdlibindex.build` scans `Root.FS`; `stdlibDir()` deleted.
- **Acceptance**: from a temp dir, `docs std/stream`, `--list`, `--all-functions`, `prelude` exit 0 with real content; `Modules("length")` non-empty.

## M3: Out-of-repo tests, binary smoke, docs (0.5 day)

- Tests run under `t.Chdir(t.TempDir())` with `AILANG_STDLIB_PATH=""`.
- Binary smoke: a Go test that builds nothing new but runs the test binary's `docsCommand` path is not enough — add a CI step after `make build` that runs `cd "$(mktemp -d)" && ailang docs std/stream`.
- Mutation-check: revert the M2 docs change and the stdlibindex change and watch the tests fail.
- CHANGELOG under `[Unreleased]` in `changelogs/v0.32-current.md`; precedence section in `docs/docs/reference/cli.md`.
- Archive `design_docs/planned/m-docs-stdlib-resolution.md` and its sprint plan to `design_docs/archive/` with a superseded note.
- **Acceptance**: `make test`, `make lint`, `make verify-examples`, `make verify-stdlib`, file sizes green.

## Out of scope

LSP go-to-definition into the embedded stdlib (R7); shipping `std/` in the tarball.

## Execution status (2026-09-26)

- [x] **M1** `internal/stdlibroot` (7-tier order, `io.ail` marker, input-keyed memo, `NotStdlibError`), loader on one root, runner `Configure`, `--trace-loader`/`--strict` wired, embed `Setenv` removed, executor + eval harness check `<dir>/std/io.ail`, `internal/module` + `AILANG_STDLIB`/`AILANG_PATH` rows deleted.
- [x] **M2** `ailang docs` reads through `fs.FS`, `prelude` answered before resolution, `findStdlibDir`/`isStdlibDir` deleted; `stdlibindex` scans `Root.FS`.
- [x] **M3** out-of-repo tests (`t.Chdir(t.TempDir())`, `AILANG_STDLIB_PATH=""`), CI binary smoke after `make install`, six mutations all killed, CHANGELOG, archive of the superseded doc.

**Deviations from the design doc**

- `ConfigureStdlibResolver` was deleted instead of given a caller: with three `NewModuleLoader` sites (pipeline, runtime, elaborate) the runner configures the process once via `stdlibroot.Configure`.
- The precedence section went into `docs/docs/reference/stdlib.md`, not `cli.md`: `cli.md` is generated from the dispatch table (`make docs-cli`, gated by `check-cli-docs`).
- `AILANG_PATH` was dropped along with `AILANG_STDLIB`: the dead `internal/module` was its only reader too.
- Test fixtures that built a partial `std/` overlay (pipeline foreign-constructor tests) now carry an `io.ail` marker and every module they import — deliberate change 5(b) in action.
- `make simplicity-audit-fast` reports `closure_nonleaf_packages` +1: `internal/stdlibroot` imports `config` (env access must go through the Registry), so it cannot be a leaf.
