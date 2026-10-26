# Sprint Plan: M-PKG-AUTHORITY-CHOKEPOINT (#1720)

**Design doc**: [m-pkg-authority-chokepoint.md](m-pkg-authority-chokepoint.md) (Design Freeze D2–D4 ruled 2026-10-26, Mark, attended; not re-litigated here)
**Sprint JSON**: `.ailang/state/sprints/sprint_M-PKG-AUTHORITY-CHOKEPOINT.json`
**Branch**: `p0/pkg-authority-chokepoint` (based on origin/dev `8e634480f`)
**Target**: v0.54.19 · **Priority**: P0 (confinement/security) · **Risk**: medium

## Goal

Under `AILANG_AGENT_POLICY`, no code reachable from a package verb or the package
loader opens a network connection or writes under the user's home. One gate per
authority (`internal/pkg/authority.go`), and three layers that make a bypass fail
CI: the gating `RoundTripper`, an AST boundary test, and an end-to-end matrix whose
rows are generated from `pkgSubcommands()`.

## Current status (re-verified on 8e634480f)

- `RefuseIfConfined` appears only in `confinement.go`, `registry.go` (3 fetches) and
  `pkg_docs.go` — the audit's 14 guarded / 22 unguarded split still holds.
- The `scratchpad/dd3` repros are the baseline (`repro.out`, `repro_http.out`,
  `repro_home.out`).
- `internal/pkg` files are all under 600 lines; `resolver.go` (419) grows with the
  D2 branch, so the confined git path goes in a companion `resolver_git.go`.

## Velocity

Recent sprints of this shape (security fix in `internal/pkg` + `cmd/ailang`
binary-level tests) land at ~300–400 LOC/day including tests. Estimate ~1,450 LOC
(≈ 650 impl + 800 tests) → 4 days, matching the design doc's 2× estimate.

## Registry reuse gate

| Milestone | Package | Action | Reason |
|---|---|---|---|
| all | — | none | This is Go host code inside the `ailang` binary (package manager confinement); no AILANG registry package can implement it. |

## AILANG syntax gate

No milestone writes showcase `.ail` code. Test fixtures use the one-line module form
already used by the dd3 repro (`module x/y` + `export func f() -> int = 1`), verified
with `ailang check` by the tests themselves.

## Milestones

### M1 — ✅ Authority chokepoint (~260 LOC: 150 impl + 110 tests)

- `internal/pkg/authority.go`: `Authority` (`AuthNetwork`, `AuthHomeWrite`),
  `Require(a, what, remedy...)`, `HTTPClient(what, timeout)` with a gating
  `RoundTripper`, `Git(dir, args...)` (hardened prefix `-c protocol.ext.allow=never`,
  `GIT_TERMINAL_PROMPT=0`; network subcommands need `AuthNetwork`; any non-read-only
  invocation in a dir under `~/.ailang` needs `AuthHomeWrite`), `UserDir(sub)` (pure)
  and `EnsureUserDir(sub)` (gated). `confinement.go` folded in; `RefuseIfConfined` and
  `EnsureRegistryCacheDir` stay as thin wrappers with the pinned text unchanged (AC9).
- Unit tests: message pinning, transport hit-count confined=0 / unconfined=1, Git
  classification, `UserDir` purity, `EnsureUserDir` refusal.

**Acceptance**: `go test ./internal/pkg -run 'Authority|Confinement'` green; pinned
registry text byte-identical.

### M2 — ✅ Git path closed (rows 9–16, 19, 36) + D2 + D3 (~330 LOC: 190 impl + 140 tests)

- `gitcache.go`: base from `UserDir("cache/git")`, `NewGitCache` creates nothing;
  `EnsureUserDir` only inside `ensureClone`; every git call through `pkg.Git`, `--`
  before the URL (AC10).
- Resolver: `ResolveDependenciesWithPrev(manifest, root, prev *LockFile)`; under
  confinement a git dep is carried forward only when the prior entry matches
  (`git_url`, `tag`/`rev`, `subdir`) and the provisioned dir hashes to its
  `content_hash`; otherwise `Require(AuthNetwork, "resolving git dependency <name>")`
  with the remedy "run 'ailang lock' outside the sandbox or provision
  AILANG_PACKAGE_ROOT". `ResolveLock`/`CheckLock`/`EnsureLock`/`pkgLockCommand` pass
  the previous lock.
- Shared `gitPackageDir(locked)`: `PackageDir` under `AILANG_PACKAGE_ROOT` (D3,
  exclusive, no `$HOME` fallback) else the URL-keyed clone; loader uses it (no
  `MkdirAll`). `pkg_docs` git-cache read uses `UserDir` (pure) and is skipped when the
  root is set.
- Manifest validator rejects a `git` value beginning with `-`.
- Tests: D2 matrix (match → carry; content mismatch, URL/tag/subdir change, no prior
  lock → refusal), loader non-creation, D3 root resolution + "not provisioned".

**Acceptance**: confined `lock`/`lock --check`/`install --path`/`check` over a
`file://` git dep write nothing under `HOME`; existing `TestGitCache_*`, lock and
install tests green (tests asserting `NewGitCache` creates its dir are rewritten).

### M3 — ✅ HTTP, shims, messaging, Pub/Sub (rows 17, 18, 20, 22, 24, 26, 28–33) (~180 LOC: 110 impl + 70 tests)

- `RegistryClient.httpClient`, `registryAPIGet`, `pkgHistoryCommand`,
  `uploadTarball`, `pkgUnpublishCommand`, `registryAdminRequest` → `pkg.HTTPClient`.
- `DefaultBinDir` → `UserDir("bin")`; `WriteShim`/`RemoveShim` → `Require(AuthHomeWrite)`.
- `openPkgMsgStore(mode)`: write mode requires `AuthHomeWrite`; best-effort mode returns
  `errSkippedConfined`; `warnSilentRatchet` only opens the store when a path dep's
  version actually changed and prints `note: silent-ratchet check skipped (confined)`
  under confinement (D7).
- `newCascadePublisher` → `Require(AuthNetwork, "publishing cascade notifications")`.

**Acceptance**: confined `pkg info/versions/stats/history` make 0 requests; no
`collaboration.db` appears for confined `lock` with a prior lock.

### M4 — ✅ Command-table authority (D5) (~110 LOC: 60 impl + 50 tests)

- `Command.Authority []pkg.Authority`; operator verbs (`publish`, `unpublish`, `key`,
  `install`, `add`, `notify-upgrade`, `bin`, `cascade`) declare it; `pkgGroupCommand`
  and the generated legacy top-level rows refuse at dispatch when confined.
  `bin list` stays read-only (the declaration on `bin` refuses only mutating
  subcommands — `list` is pure); `cascade status` was already gated by I/O and stays
  allowed read-free.
- Completeness test: every `pkgSubcommands()` entry appears in an explicit
  authority table (empty = none), so a new verb must declare.

**Acceptance**: confined `ailang add …`/`ailang pkg add …` both refuse before any work.

### M5 — ✅ Git trust (T1/T2, D4) (~120 LOC: 50 impl + 70 tests)

- `ValidateContentHashesFrom` gains a git branch via `gitPackageDir`; mismatch or a
  missing dir returns a `RegistryContentError`-class (fatal) error naming the package
  and both hashes.

**Acceptance**: a tampered `.ail` in a provisioned git dep, and a different tag checked
out in the shared clone, both make `ailang check` exit ≠ 0 with a drift message (AC5).

### M6 — ✅ Proof, docs, changelog (~450 LOC tests + ~60 docs)

- `internal/pkg/authority_boundary_test.go` (AC3): AST scan of `internal/pkg` +
  the `cmd/ailang` files that define a `pkgSubcommands()` `Run` or the
  `openPkgMsgStore`/`newCascadePublisher` helpers (computed from the AST).
- `cmd/ailang/pkg_confinement_matrix_test.go` (AC1, AC2a, AC2c, AC4, AC6): built
  binary, temp `HOME`, loopback recorder, `file://` + `http://127.0.0.1` git deps,
  `PATH`-first git shim, outside canary; rows generated from `pkgSubcommands()` plus
  `lock --check`, `install --path .`, `check`, `ai-check`, `test`, `iface`, `run`; a verb
  with no row fails the test. Unconfined arm must see ≥1 request/clone ("instrument
  blind" otherwise). Policy-tool `lock` route through `policytool.Host`.
- Mutation evidence (AC2b) on scratchpad copies — never `git checkout <file>`.
- Re-run `scratchpad/dd3/repro*.sh` against the built binary (AC8).
- Docs: `SECURITY.md` confinement section, `docs/docs/reference/std-package.md`, the
  `config` env row text for `AILANG_PACKAGE_ROOT`; changelog fragment
  `changelogs/unreleased/2026-10-26-pkg-authority-chokepoint.md` (### Fixed — security),
  closing the v0.53.0 "known gap" note.

## Validation (every milestone)

`make test-changed` · `make lint` · `make test-imports` · `make check-file-sizes` ·
`make check-boundaries` · `make check-changelog`. Never full `make test` locally.
No new environment variable (A8) — the forbidigo Registry rule therefore has nothing
to register. Tests never touch the real `$HOME` or the network.

## Risks

| Risk | Mitigation |
|---|---|
| D4 breaks unconfined users who share one git URL at two tags | Error names the package and says `run 'ailang lock'`; changelog calls it out |
| Matrix is slow (binary build + git) | Reuse `buildAilang`; file-local git; bounded per-row timeouts |
| AST test too strict | Allowlist with a reason string per entry |

## Execution record (2026-10-26)

All six milestones landed on `p0/pkg-authority-chokepoint` (≈2,050 Go lines added incl. tests).

**Guarded-site count after:** all 36 audit rows guarded (22 previously unguarded rows closed);
`TestPkgAuthorityBoundary` reports zero raw network/git/home/messaging/Pub/Sub primitives in
the package surface outside `internal/pkg/authority.go`.

**Acceptance**

- [x] AC1 matrix — 28 confined rows generated from `pkgSubcommands()` + loader entry points; all 0 requests / 0 network git / byte-identical HOME, canary, sandbox.
- [x] AC2a unconfined control — 7 rows each light the recorder or the git shim.
- [x] AC2b mutation — table below.
- [x] AC2c / AC5 / AC6 provisioned control — confined check/lock/lock --check/tree exit 0 from a read-only `AILANG_PACKAGE_ROOT`; tamper and missing package fail loudly.
- [x] AC3 AST boundary test (+ a probe proving it detects all 7 construct kinds).
- [x] AC4 policy-tool `lock` route refuses, no writes.
- [x] AC7 unconfined unchanged — existing pkg/lock/install/publish/gitcache tests green; the one test asserting "git deps are skipped by hash validation" was replaced per the testing policy (D4).
- [x] AC8 repro re-run (`scratchpad/dd3/repro*_after.out`): confined arms 0 HOME entries, 0 requests; unconfined control still 1 request.
- [x] AC9 pinned text; AC10 `--` before URLs + manifest rejects leading `-`.

**Mutation evidence (AC2b)** — each applied to a scratchpad copy of the repo, never `git checkout`:

| Mutation | Red tests |
|---|---|
| A: `Require` removed from `pkg.Git` network branch + `ensureClone` pre-check | `TestGitConfinedRefusals` (matrix stays green: D2 carry-forward and the home-write gate still hold — layering) |
| B: A + home-write gates + D2 bypass (`gitFromProvisionedOnly` ignores `Confined()`) | `TestGitConfinedRefusals`, `TestGitCacheConfinedResolve`, `TestResolveGitConfined_CarryForwardMatrix`, matrix `lock`/`pkg lock`/`pkg lock --check`, `TestPkgConfinementPolicyToolLock` |
| C: `HTTPClient` transport gate removed | `TestHTTPClientTransportGate`, matrix `pkg info`/`versions`/`stats`/`history` |
| D: `MkdirAll` re-added to `NewGitCache` | `TestLoaderGitCaseCreatesNothing`, `TestGitCacheConfinedResolve`, matrix `check`/`ai-check`/`run`/`lock`/`pkg lock`/`lock --check`/`quality`, policy-tool lock |
| E: raw `http.Get` inserted in `pkg_info.go` | `TestPkgAuthorityBoundary` |
| F: git hash check removed (D4) | `TestValidateGitContentHashes`, `TestPkgConfinementMatrix_ProvisionedControl` |
| G: `warnSilentRatchet` calls `messaging.OpenStore` directly | `TestPkgAuthorityBoundary` |
| H: a verb's matrix row deleted | `TestPkgConfinementMatrix` ("has no confinement matrix row") |
| I: `add` drops its `Authority` declaration | `TestPkgVerbAuthorityComplete`, `TestPkgOperatorVerbsRefusedAtDispatch` |

**Gates:** `make lint` 0 issues · `make test-imports` ✓ · `make check-file-sizes` ✓ · `make check-boundaries` ✓ ·
`make check-changelog` ✓ · `make test-changed`: `internal/pkg` ok, `internal/config` ok, `cmd/ailang` one failure
`TestMemprobeDebugLogDoesNotAccumulate` (peak-RSS test, run while the mutation job loaded the machine; passes
alone: delta 13 MB ≤ 30 MB) — unrelated to this change.

**Deferred (agent-resolvable, per design):** `config.Confined()` shared with `examples_corpus.go` (one concept, two
definitions remains); extending the `Command.Authority` column and matrix to non-pkg top-level commands; T3
(read confinement of lock-supplied paths) and a rev-keyed git cache remain Future Work.
