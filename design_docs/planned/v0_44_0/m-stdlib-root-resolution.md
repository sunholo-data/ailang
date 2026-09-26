# M-STDLIB-ROOT-RESOLUTION: one stdlib root for every command (`ailang docs std/<module>` fails outside a project root)

**Status**: Planned
**Target**: v0.44.0
**Priority**: P1 (Medium). The fourth external report of the same gap (GitHub #1131 was the third), and the tool agents are told to use first.
**Estimated**: about 1.5 agent-days. M1 takes 0.5, M2 takes 0.5 and M3 takes 0.5.
**Dependencies**: None. No design-freeze items: every decision below can be resolved by an agent, and the open questions have defaults.
**Planner-Lane**: codex-ok
**Requested by**: Daneel, bug ticket "Bug: ailang docs std/<module> fails outside a project root (stdlib directory not found)" (2026-09-25). Also closes GitHub #1131.
**Supersedes**: [m-docs-stdlib-resolution.md](../m-docs-stdlib-resolution.md) and its sprint plan (planned for v0.38.6, not implemented). See "Relation to the earlier doc" below.

---

## Problem Statement

```
$ cd /tmp && ailang docs std/stream
Error: stdlib directory not found

Tip: Set AILANG_STDLIB_PATH or run from project root
```

Reproduced on 2026-09-25 from a scratch directory outside any repo, using both the installed
`ailang` (v0.43.1-6-g62608138d-dirty) and a binary built from this worktree (`go build ./cmd/ailang`
at origin/dev). `std/clock`, `docs --list`, `docs --all-functions` and **`docs prelude`** all fail
the same way. `docs prelude` does not read the stdlib at all. From the same directory,
`ailang run` and `ailang check` of a program that imports `std/list` both succeed.

Daneel's CLAUDE.md tells every agent to run `ailang docs std/<module>` before writing helpers.
Agents usually work in a scratch directory, so they get the error and fall back to guessing
signatures or reading `std/*.ail` from a git tag. This is the fourth time this gap has been
reported (GitHub #1131 was the third).

**Done when (the ticket's criterion):** `cd /tmp && ailang docs std/stream` prints the module's
exports on a release binary.

## Root cause

The stdlib is compiled into the binary (`std/embed.go:10`, `//go:embed *.ail` gives `std.FS`,
added in 272483a50 in March 2026). The module loader falls back to that copy
(`internal/loader/loader.go:196-213` for `Load` and `:381-393` for `resolvePath`), which is
why `run` and `check` work anywhere. **`ailang docs` has its own private resolver that never
consults `std.FS`**:

- `cmd/ailang/docs.go:116-141` (`findStdlibDir`) tries only `AILANG_STDLIB_PATH`, `./std`
  and `../std`, then returns `stdlib directory not found`.
- `cmd/ailang/docs.go:73-78` calls it *before* the `prelude` special case at `:102`. So
  `docs prelude` fails too, although it is rendered from live mechanisms and needs no files.
- Every docs reader is path-based: `parseModuleFile` at `docs.go:208-209`,
  `parseExportSignatures` at `docs_signatures.go:33-34`, `discoverModules` (`os.ReadDir`) and
  `showModuleDocs` (`os.Stat`) at `docs.go:398`. None of them can read an `fs.FS`.

## Audit: every place that resolves the stdlib directory (CLAUDE.md principle 3)

There are five independent resolvers with four different search orders. Only one of them
knows about the embedded copy.

| # | Site | Order it searches | Outside a repo | Symptom |
|---|------|-------------------|----------------|---------|
| R1 | `internal/loader/stdlib_resolver.go:222-283` (`StdlibResolver`), used by `run`, `check`, `iface`, `test`, the REPL, the API server and `internal/embed` | `--stdlib-path` override (**never set**, see R1a), then **`./std`**, then `<bin>/../std`, then `AILANG_STDLIB_PATH`, then the user data dir, then `/usr/{local/,}share/ailang/std`. The resolution is **per module**, followed by the embedded copy (in `loader.go`) | Works through the embedded copy | Correct. The search order is still wrong: see R1a and R1b |
| R1a | `cmd/ailang/main_run.go:41` `--stdlib-path` becomes `internal/runner/run.go:122-123` `os.Setenv(AILANG_STDLIB_PATH)`. `ConfigureStdlibResolver` (`loader.go:102`) has **no non-test caller**. `--trace-loader` and `--strict-version` are accepted and discarded (`runner/run.go:125-128`) | The flag becomes tier 4, which is **below `./std`** | – | **Verified**: from a directory containing a shadow `std/list.ail`, `ailang run --stdlib-path <real std> t.ail` printed `[999]` (the shadow `map`). The flag help says it "overrides AILANG_STDLIB_PATH", but in practice it overrides nothing that is in the cwd |
| R1b | The same per-module search | Each module is resolved independently, so one process can load modules from **two roots** | – | The same run printed `stdlib version mismatch … at <real std>` while also using the shadow `list`. The modules came from mixed roots |
| R2 | `cmd/ailang/docs.go:116-141` (`findStdlibDir`), used by `docs <module>`, `--list`, `--all-functions` and `prelude` | `AILANG_STDLIB_PATH` (a single path, with no list splitting), then `./std`, then `../std`. **No embedded copy** | **Fails** | This ticket |
| R3 | `internal/stdlibindex/index.go:33-46`, used for the import suggestions on "undefined variable" (`cmd/ailang/diagnostics_wiring.go:13-23`, `types.ImportSuggester` and `importhint.*`) | `AILANG_STDLIB_PATH` (the first entry that exists), then the literal `"std"`. **No embedded copy** | **Silently empty** | **Verified**: `ailang check` of `println(show(length([1,2])))` inside the repo appends "`length` is exported by std/array, std/bytes, std/list, std/string; add the matching import". From the scratch dir it prints only `undefined variable: length`. This is the same bug as R2, but silent, and it hits every agent that works outside the repo |
| R4 | `internal/module/loader.go:89-105` and `internal/module/resolver.go:274-303` (they read `AILANG_STDLIB`, `../stdlib` and `./stdlib`) | – | – | **Dead code.** `go list -f '{{.Imports}}' ./...` shows no importer of `internal/module`. Its only live effect is the `AILANG_STDLIB` registry row (`internal/config/paths.go:8,25`), which documents a variable nothing reads |
| R5 | `internal/embed/embed.go:66-70` sets `AILANG_STDLIB_PATH` to the **project root** (not `<root>/std`). Used by `cmd/ailang/budget.go`, `coordinator_approvals_engine.go` and `internal/server/ailang_bridge.go` | – | – | This is a no-op today, because R1 skips an entry that holds no `<module>.ail`. It would become an error under M1's rule for explicit overrides, so M1 removes it |
| R6 | `internal/executor/environment.go:80-88` exports `AILANG_STDLIB_PATH=<cwd>/std` **without checking that it exists**. `internal/eval_harness/runner.go:276` and `agent_validation.go:243` pass `--stdlib-path <cwd>/std` | – | – | These are callers that choose a root, not resolvers. Neither checks that the path exists, which is harmless today because R1 skips missing entries. Under M1's rule for explicit overrides, a missing path becomes an error, so M1 guards both sites with a stat on `<dir>/io.ail` |
| R7 | `internal/lsp/definition.go:212` (go-to-definition into std) uses R1's `ResolveStdlib` and then `os.ReadFile` | – | Silently finds nothing | There is no file URI for an embedded module. **Out of scope** here (see Non-Goals) |

Commands that are *not* affected: `docs search` and `docs embed-warmup`, which index design
docs and not the stdlib; `ailang prompt`; and `ailang check std/io.ail` / `iface std/io`,
which go through R1.

## Goals

**Primary goal:** one function decides where the stdlib lives, it is resolved once per
process, and it always ends at the embedded copy. After this, no command's behaviour depends
on the working directory unless the user asked for that by keeping a `./std` there.

**Success metrics**

- From `t.TempDir()`, with `AILANG_STDLIB_PATH` unset: `ailang docs std/stream`, `docs --list`,
  `docs --all-functions` and `docs prelude` all exit 0.
- The import suggestion for an undefined `length` is identical inside and outside the repo.
- `--stdlib-path X` wins over `./std`, and a single run never mixes modules from two roots.
- `grep -rn 'findStdlibDir\|func stdlibDir\|getStdlibPath\|findStdlibPath'` over `cmd/` and
  `internal/` returns only the unified resolver.

## Solution Design

### The single resolver

Add a leaf package, `internal/stdlibroot` (about 120 LOC). It imports only `config` and `std`,
so `loader`, `stdlibindex` and `cmd/ailang` can all use it without an import cycle.

```go
type Root struct {
    FS     fs.FS    // os.DirFS(Dir), or std.FS when embedded
    Dir    string   // absolute path; "" when embedded
    Source string   // "flag" | "env" | "cwd" | "binary" | "user" | "system" | "embedded"
    Tried  []string // every candidate checked, for trace output and errors
}

func Resolve(override string) (Root, error) // memoised per process for override == ""
```

**The order.** The first candidate that contains `io.ail` wins. That is the marker
`docs.go:144-151` already uses, and it stops an unrelated project `std/` directory from being
mistaken for the stdlib.

1. `override` (the `--stdlib-path` flag)
2. `AILANG_STDLIB_PATH` (a path-list, split with `os.PathListSeparator`)
3. `./std` (development and `go run`: live edits to `std/` show up immediately)
4. `<bin>/../std` (tarball layout)
5. the user data dir (`getUserDataDir` moves here unchanged)
6. `/usr/local/share/ailang/std` and `/usr/share/ailang/std` (not on Windows)
7. **the embedded `std.FS`**, which always exists and is version-matched to the binary by construction

**What changes compared with R1.** An explicit override (tiers 1 and 2) now beats `./std`.
This matches the flag's own help text and R2's existing order. If an explicit override is set
but none of its entries contains `io.ail`, the result is an **error naming the tried paths**.
It does not fall through (principle 2: a user who names a stdlib should not silently get a
different one). The implicit tiers (3 to 6) fall through to the embedded copy without noise.

**One root per process.** The loader resolves modules against `Root.FS` only, so R1b's
mixed-root loading becomes impossible. If a module is missing from the chosen root, that is an
error that names the root. It does not fall back per module.

### Consumers

- **Loader (R1).** `StdlibResolver` keeps its public surface (`ResolveStdlib`, the version
  check, `errWithSearchTrace`, the "did you mean" suggestion) but gets its candidate list and
  its FS from `stdlibroot`. The two ad-hoc `std.FS` fallbacks in `loader.go` collapse into
  "read from `Root.FS`". The `<embedded>/std/<m>.ail` display path is kept. `runner` passes
  `opts.StdlibPath` as the override instead of calling `os.Setenv`, and
  `ConfigureStdlibResolver` finally gets its caller. `--trace-loader` is wired to print
  `Root.Source` and `Root.Tried`.
- **Docs (R2).** Delete `findStdlibDir` and `isStdlibDir`. `parseModuleFile`,
  `parseExportSignatures`, `discoverModules` and `showModuleDocs` take `(fsys fs.FS, name string)`
  and use `fs.ReadFile` and `fs.ReadDir`. Resolution moves *after* the `prelude` branch. The
  terminal error now happens only for a bad explicit override, and it prints `Root.Tried`.
- **Import hints (R3).** `stdlibindex.build` scans `Root.FS`. `stdlibDir()` is deleted.
- **Dead code and callers (R4 to R6).** Delete `internal/module` (check `make test` and
  `git log` for any intended revival first; see the coding-standards rule on "unused" code).
  Drop the `AILANG_STDLIB` registry row. Remove the `Setenv` in `internal/embed/embed.go`.
  In the executor and eval harness, pass the root only if `<dir>/io.ail` exists.

### Files to modify

- `internal/stdlibroot/root.go` (new, about 120 LOC) and `root_test.go` (new, about 150 LOC)
- `internal/loader/stdlib_resolver.go` (−60 / +25): the candidate list delegates to stdlibroot, and `getUserDataDir` moves
- `internal/loader/loader.go` (−30 / +10): one read path through `Root.FS`
- `internal/runner/run.go` (±10): the override goes through `ConfigureStdlibResolver`, and `--trace-loader` is wired
- `cmd/ailang/docs.go`, `cmd/ailang/docs_signatures.go`, `cmd/ailang/docs_all_functions.go` (±60): take an `fs.FS`, and resolve after `prelude`
- `internal/stdlibindex/index.go` (−15 / +5)
- `internal/embed/embed.go` (−4), `internal/executor/environment.go` (±4)
- `internal/module/` (deleted), `internal/config/paths.go` (−3)
- `CHANGELOG.md`, and `docs/docs/reference/cli.md` (document the order and that the flag beats `./std`)

### Conflict Surface

This change touches `internal/loader`, the module-root half of the pipeline.

1. **Positions extended.** The stdlib-root precedence, and the FS that std modules are read from.
2. **Other users of those positions.** Repo-root development (`./std` must still win when no
   override is set). The eval harness passes `--stdlib-path <cwd>/std` and runs the child in
   `<cwd>/.eval_workspace/…`, where there is no `./std`. The flag already decides there today,
   so nothing changes. The cloud executor's workspace `std` goes through the env tier.
   The version-mismatch warning in `checkStdlibVersion` applies to on-disk roots only, because
   the embedded copy matches by construction.
3. **Disambiguation.** Explicit beats implicit, and implicit beats embedded. There is one root
   per process.
4. **Must still work.** `make test`, `make verify-examples` and `make test-imports` pass from
   the repo root. `ailang check std/io.ail` and `ailang iface std/io` work
   (`loader.go:184-193`). A multi-file example that imports `std/list` and `std/string` works
   from `t.TempDir()`.
5. **Deliberate changes.** (a) A shadow `./std` no longer beats `--stdlib-path` or
   `AILANG_STDLIB_PATH`. (b) A partial `./std` overlay, meaning a directory that has `io.ail`
   but lacks some module, now errors instead of taking the missing module from another root.
   (c) An explicit override that resolves nowhere now errors instead of being skipped silently.

## Milestones

### M1: `internal/stdlibroot` and the loader on top of it (0.5 day)

- [ ] `Resolve` with the seven-tier order, per-process memoisation, and `Tried`.
- [ ] The loader and runner read through it. `--stdlib-path` works as a real override, and `--trace-loader` prints the source.
- [ ] R5 and R6 fixed. R4 deleted.
- **Acceptance:** table tests pin the order. The shadow `./std/list.ail` scenario from the Audit returns the real `map` when `--stdlib-path` is given. A test asserts one root per run.

### M2: `ailang docs` and import hints (0.5 day)

- [ ] The docs readers take an `fs.FS`. `findStdlibDir` is deleted. Resolution moves after `prelude`.
- [ ] `stdlibindex` scans `Root.FS`.
- **Acceptance:** the ticket's criterion. `docs std/stream`, `--list`, `--all-functions` and `prelude` exit 0 from a temp dir. The `length` hint appears outside the repo.

### M3: Tests from outside the repo, and documentation (0.5 day)

- [ ] The out-of-repo tests below, the CHANGELOG entry, and the precedence section in `cli.md`.
- **Acceptance:** each test fails when M2 is reverted (mutation-check them).

## Tests

All of these run with `t.Chdir(t.TempDir())` and `t.Setenv("AILANG_STDLIB_PATH", "")`, so
no test depends on the repo checkout being the cwd.

1. `internal/stdlibroot`: from an empty temp dir, `Resolve("")` returns `Source=="embedded"`,
   and `fs.ReadFile(root.FS, "stream.ail")` succeeds. Also: a temp `std/io.ail` in the cwd
   gives `cwd`; env and flag each beat it; a bogus flag gives an error listing `Tried`; a
   `std/` without `io.ail` is skipped.
2. `cmd/ailang`: `showModuleDocs`, `listModules`, `buildAllFunctionsLines` and
   `renderPreludeDocs` run from a temp dir. Assert on the output (for example, `docs std/stream`
   contains an export known to be in `std/stream.ail`), not just the exit code.
3. `internal/stdlibindex`: from a temp dir, `Modules("length")` is non-empty.
4. `internal/loader`: the shadow-`std` precedence test and the one-root-per-run test.
5. Binary smoke (release lane): after `make build`, `cd "$(mktemp -d)" && ailang docs std/stream`
   exits 0. Wire it into CI (`.github/workflows/ci.yml`) or the release workflow, whichever already runs the built binary, so a release binary is covered.

## Non-Goals

- Shipping `std/` in the release tarball. The earlier doc's Ruling 2 is unnecessary once every
  reader falls back to `std.FS`. It can be revisited later if on-disk sources matter for the LSP.
- LSP go-to-definition into embedded stdlib modules (R7). That needs a file URI, for example
  by materialising files under the cache dir, and it gets its own small follow-up.
- `docs search` or embeddings. No language or semantics changes.

## Axiom compliance

This is a tooling and resolution fix. The relevant axioms, scored briefly:

- **Determinism (+1).** One root per process, and a version-matched embedded floor, remove
  cwd-dependent stdlib selection.
- **Fail loudly (+1).** Explicit overrides that do not resolve now error instead of being
  skipped. The silent empty import-hint index is gone.
- **AI-friendliness (+1).** The documented discovery tool works where agents actually run.
- **Simplicity (+1).** Five resolvers become one, one dead package is deleted, and one env var
  row is dropped.

There are no hard violations. The net score is +4.

## Quorum trigger check

Trigger 2 applies in a narrow sense: the loader's precedence changes (explicit overrides now
beat `./std`), and other lanes depend on the loader. The one lane that passes an override, the
eval harness, runs its child in `.eval_workspace/…`, which has no `./std`
(`eval_harness/runner.go:242,276`). The flag already wins there today. Triggers 1, 3 and 4 do not fire: there are no freeze items, no
cost or banking surface, and every premise can be checked in-repo. **Recommendation:** skip
the quorum unless the reviewer disagrees with deliberate change 5(b).

## Relation to the earlier doc

[m-docs-stdlib-resolution.md](../m-docs-stdlib-resolution.md) (2026-09-13, #1131) diagnosed
R2 correctly, but some of its premises were wrong or have gone stale:

- It says embedding `std/` is "new" and rules out a loader-side embedded fallback. Both
  already existed, in `std/embed.go` and `loader.go:196`. Its own sprint plan noticed this.
- It keeps the loader's per-module search, so R1a and R1b remain.
- It does not mention R3 (the import hints go silent outside the repo), `docs prelude`, the
  inert `--stdlib-path` and `--trace-loader` flags, or the dead `internal/module`.

If this doc is accepted, the earlier doc and its sprint plan should be moved to `archive/`
with a pointer here.

## Verification log

| # | Claim | Evidence |
|---|-------|----------|
| V1 | `docs std/stream`, `std/clock`, `--list`, `--all-functions` and `prelude` fail outside a repo | Run from the scratch dir with the installed v0.43.1-6 binary and with a worktree build (`AILANG dev`): each prints `Error: stdlib directory not found` |
| V2 | `run` and `check` work from the same dir | `ailang run --caps IO --entry main t.ail` (imports `std/list`) prints `[2, 3]`. `check` prints `No errors found!` |
| V3 | Docs works when given a root | `AILANG_STDLIB_PATH=<wt>/std ./ailang-wt docs std/stream` prints the `# std/stream` header |
| V4 | Import hints silently disappear outside the repo | The same `u.ail`: inside the worktree, the hint is appended; from the scratch dir, only `undefined variable: length` |
| V5 | `--stdlib-path` loses to `./std` | Shadow `prec/std/list.ail` (`map` returns `[999]`) plus `--stdlib-path <wt>/std` prints `[999]`, together with a version warning about `<wt>/std` (mixed roots) |
| V6 | `ConfigureStdlibResolver` has no non-test caller | `grep -rn ConfigureStdlibResolver` finds only `loader.go:102` and `loader_test.go:123` |
| V7 | `internal/module` has no importers | `go list -f '{{.ImportPath}}: {{join .Imports " "}}' ./... \| grep 'internal/module '` is empty |
| V8 | `--trace-loader` is inert | `runner/run.go:125-128` (`_ = opts.TraceLoader`). `run --trace-loader` from the scratch dir prints no `[trace-loader]` lines |
| V9 | No other `findStdlibDir` or `isStdlibDir` callers | `grep -rn 'findStdlibDir\|isStdlibDir'` finds only `docs.go` |
| V10 | The stdlib is embedded | `std/embed.go:10` `//go:embed *.ail`. It is used at `loader.go:203,390`, `pipeline/canonical_json.go:120` and `cmd/wasm/main.go:44` |
