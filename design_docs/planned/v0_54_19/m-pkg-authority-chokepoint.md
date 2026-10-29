# M-PKG-AUTHORITY-CHOKEPOINT: route every package network and `$HOME` write through one confinement gate, and make a bypass fail a test

Tracks: #1720

**Status**: Implemented on `p0/pkg-authority-chokepoint` (awaiting independent sprint evaluation); sprint record in [m-pkg-authority-chokepoint-sprint-plan.md](m-pkg-authority-chokepoint-sprint-plan.md)
**Target**: v0.54.19
**Priority**: P0. This is a security/soundness bug: a confined agent gets network egress and a persistent write outside its sandbox, and later checks resolve against that write.
**Estimated**: 4 days (2× a naive 2-day estimate, per house convention)
**Dependencies**: None. This doc completes [M-PKG-REGISTRY-CONFINEMENT](m-pkg-registry-confinement.md) (#1607/#1608, shipped in v0.53.0) and replaces its per-function gates with a per-authority gate. It does not change the confinement *signal*.
**Issues**: [#1720](https://github.com/sunholo-data/ailang/issues/1720) (P0, area:pkg). Premises were re-verified by code reads and live runs on origin/dev `772703831` on 2026-10-26 (Verification Log).

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Today a git dependency is read from a URL-keyed checkout whose working tree is whatever revision was checked out last (V16), and its `content_hash` is never checked (V15). After this change the lock's hash pins it. |
| A2: Replayability | 0 | No trace or replay surface change. |
| A3: Effect Legibility | +1 | `lock`, `lock --check`, `install --path`, `check` and five registry-API verbs stop performing hidden egress or `$HOME` writes under confinement. Each becomes a named refusal (V3–V9). |
| A4: Explicit Authority | +1 | Network and `$HOME` writes become two named authorities, each with one grant point. Today authority is ambient: every package verb gets it unless it happens to call one of three guarded functions. |
| A5: Bounded Verification | 0 | The git hash check is over files that are already present; nothing extra is fetched. |
| A6: Safe Concurrency | 0 | No concurrency change. |
| A7: Machines First | +1 | One canonical refusal text an agent can act on, with a pinned test. The boundary test makes "is this code path gated?" a mechanical question rather than one for a reviewer to remember. |
| A8: Minimal Syntax | 0 | No language syntax, and no new environment variable. |
| A9: Cost Visibility | 0 | No cost surface change. |
| A10: Composability | 0 | No effect or record change. |
| A11: Structured Failure | +1 | Every new refusal is a named error that states the operator remedy. A best-effort warning that is skipped under confinement says so on stderr instead of being skipped silently (D7). |
| A12: System Boundary | +1 | The boundary is between what the operator provisions (registry and git package roots, the bin dir, the state DB) and what an agent may write (only the op's declared ledger). It covers every transport, not only registry HTTP. |

**Net Score: +6**, so the decision is to move forward. No axiom among A1/A3/A4/A7 scores −1.

### Hard Violation Check

- [x] A1: no implicit nondeterminism introduced. An unpinned git checkout becomes hash-pinned.
- [x] A3: no hidden side effects added; twenty-two are gated.
- [x] A4: ambient authority is removed, not granted.
- [x] A7: there is no human-convenience optimisation.

## Problem Statement

### What "confined" means today

The process is confined when **`AILANG_AGENT_POLICY` is set to a non-empty value**: `pkg.Confined()` is `config.AgentPolicy() != ""` (`internal/pkg/confinement.go:11`; env row `internal/config/job.go:57,107`). Two parties set it:

- the executor launcher, from `Task.PolicyPath` (`internal/executor/environment.go:163`);
- `ailang policy-tool`, which pins it on every CLI child (`childEnv`, `internal/policytool/cli_ops.go:394–403`).

The CLI child is a **separate, unconfined OS process** (`cli_ops.go:156` comment, `runAilang` `:413–437`), so the only thing standing between a confined agent and the network or `$HOME` is that the child code checks the variable.

Lineage of the model:

- [M-AGENT-AILANG-ONLY-EXECUTION](../../implemented/v0_39_0/m-agent-ailang-only-execution.md) introduced the `ailang_only` lane and the policy-path environment variable.
- [M-EXECUTOR-POLICY-HARDENING](../../implemented/v0_41_0/m-executor-policy-hardening.md) (implemented, M4 `70393ba17`) made `AILANG_AGENT_POLICY` the materialised policy path and built `policy-tool` and its CLI schema table.
- #1552 (examples corpus) and #1554 (the child's compile and prompt caches) applied the variable to two leaks.
- [M-PKG-REGISTRY-CONFINEMENT](../v0_52_6/m-pkg-registry-confinement.md) (#1607/#1608, merged in PR #1704 `be6ca8b43`, released in v0.53.0) added `RefuseIfConfined` and `AILANG_PACKAGE_ROOT`. That doc still sits in `planned/v0_52_6/` with status "Implementation complete; awaiting independent sprint evaluation". It was never moved to `implemented/`.

The v0.53.0 changelog records the gap this doc closes (`changelogs/v0.32-current.md:1561`): *"Known gap: a confined `ailang lock` with a git dependency still clones into `$HOME/.ailang/cache/git`, and `pkg info`/provenance skip the confinement guard; tracked in #1720."*

### The defect, measured

All three runs below used a binary built from origin/dev `772703831`, with `AILANG_AGENT_POLICY` set, an empty temporary `HOME`, and a manifest with one `file://` git dependency (`scratchpad/dd3/repro.sh`, V3):

```
== confined `ailang lock` (no prior lock)            rc=0
Cloning into '$HOME/.ailang/cache/git/366de57b0bf5f04f'...
✓ Generated ailang.lock (1 packages)
$HOME/.ailang/cache/git/366de57b0bf5f04f              ← clone written by the confined agent

== confined `ailang check main.ail` (git dep), fresh HOME
$HOME/.ailang/cache/git                               ← the loader's NewGitCache MkdirAll

== confined `ailang lock` with a prior lock, fresh HOME   rc=0
$HOME/.ailang/cache/git/366de57b0bf5f04f
$HOME/.ailang/state/collaboration.db                  ← warnSilentRatchet opens the messaging SQLite store
```

A loopback-registry run counted HTTP requests made under confinement (`scratchpad/dd3/repro_http.sh`, V7):

```
pkg info test/lib            requests=1  GET /api/packages/test/lib     ← unguarded
pkg versions test/lib        requests=1  GET /api/packages/test/lib     ← unguarded
pkg stats                    requests=1  GET /api/stats                 ← unguarded
pkg history test/lib@0.1.0   requests=1  GET /packages/test/lib/0.1.0/history.json  ← unguarded
pkg provenance test/lib@0.1.0 requests=0                                ← guarded (FetchMetadata)
pkg search / install / docs / cascade status   requests=0               ← guarded
-- unconfined control: pkg info requests=1, pkg search requests=1       ← the instrument sees a positive
```

The issue body says provenance skips the guard. The measurement says **`pkg provenance` is guarded**, because it goes through `FetchMetadata`. The unguarded sibling is **`pkg history`**, which calls a raw `http.Get` at `pkg_provenance.go:89`.

### Why this is P0

- **Egress and a write outside the sandbox.** `lock` is reachable from a confined agent whenever a policy's `cli_allow` names it (`cliSchemas`, `cli_ops.go:107`). Its declared write ledger says `ailang.lock` only (`lockWrites`, `cli_ops.go:71`), and that ledger is false (V3). An https git URL means outbound network. A `file://` URL copies any git repository the host user can read into a cache that later checks resolve against, which is a read escape as well as a write.
- **The cache poisoning class is the same as #1607.** The loader resolves git dependencies from that cache (`loader.go:148–170`), and `ValidateContentHashesFrom` never verifies git `content_hash` (`lockfile.go:163`; `registry_hash.go:21` covers `registry` only, V15). Whatever the confined agent caused to be cloned is trusted when later checks compile against it.
- **The default-allowed surface writes too.** `check`, `test` and `iface` are in `defaultCLIAllow` (`cli_ops.go:79–82`). For any project with a git dependency they `MkdirAll $HOME/.ailang/cache/git` (V4). The write is small, but it breaks the "writes nothing anywhere" contract of #1704's A3.

### Root cause

1. **The gate surrounds a client, not an authority.** #1704's D1 made the registry client's three fetch methods the choke point. The things that need gating are *network egress* and *writes under the user's home*, and those happen through at least five other transports in the same package surface: the git subprocess, the validator HTTP API, raw `http.Get`, Pub/Sub, and the messaging SQLite store. None of them pass through the registry client, so none were gated.
2. **An unverified negative premise.** #1704's Non-Goals said git dependencies were "read-only in the confined paths today". No Verification Log row backed that claim, and it was false: `resolver.go:192` clones from the same `lock` that the doc gated for registry dependencies.
3. **No mechanism makes the next bypass fail.** The #1704 tests enumerate *known functions*: `TestRegistryConfinement` lists three fetches (`confinement_test.go:23–27`). Nothing enumerates *entry points* or forbids a new raw I/O call, so a new site is unguarded by default. That is how `pkg history`, `pkg info` and `pkg key` stayed open in the same files #1704 edited.

## Audit: every network / `$HOME`-write site reachable from `ailang pkg …` and the loader

Scope:

- every verb in `pkgSubcommands()` (`cmd/ailang/commands_pkg.go:21–121`), 19 verbs. The bare top-level spellings (`lock`, `install`, `add`, `docs`, …) run the *same* functions (`pkgLegacyTopLevel`, `commands_groups.go:269–279`), so they are covered;
- `lock --check`;
- `install --path`;
- every caller of `PackageLoader.packageDir`: `check`, `ai-check`, `test` and `iface` through `pipeline/package_resolver.go:77`; `run` through `runner/pkg_dirs.go:25`; AI-provider init through `cmd/ailang/configdriven_init.go:104`.

There is no `pkg update` verb (V12). "Policy-tool reach" means **default** (in `defaultCLIAllow`), **opt-in** (in `cliSchemas`, so admitted when a policy's `cli_allow` names it), or **none** (no schema). Code that is not reachable through policy-tool is still reachable from an `ailang_only` launcher shell or a future `cli_allow`, and the fix gates it anyway: the gate is environmental, not per-schema.

| # | Entry → site | File:line | Authority | Policy-tool reach | Status |
|---|---|---|---|---|---|
| 1 | registry index fetch | `internal/pkg/registry.go:62–63` `FetchIndex` | network | via 7,16,19,22,24,27,28 | **guarded** |
| 2 | registry tarball fetch | `internal/pkg/registry.go:121–122` `FetchPackage` | network | via 8,22 | **guarded** |
| 3 | registry metadata fetch | `internal/pkg/registry.go:149–150` `FetchMetadata` | network | via 22,24,27 | **guarded** |
| 4 | `pkg search` → `SearchPackages` | `registry.go:184`, `pkg_search.go:40–42` | network | none | **guarded** (transitive, measured 0 req) |
| 5 | `install` → `ResolveLatestVersion` | `registry.go:242`, `pkg_install.go:74` | network | none | **guarded** (transitive) |
| 6 | registry cache creation | `confinement.go:23–31` `EnsureRegistryCacheDir` | `$HOME` write | via 8,22 | **guarded** |
| 7 | `lock` → transitive path-dep index lookup | `internal/pkg/resolver.go:110–115` | network | opt-in | **guarded** (via 1) |
| 8 | `lock` → registry download + extract | `resolver.go:270–285` | network + `$HOME` write | opt-in | **guarded** (via 2, 6) |
| 9 | `lock` → **git dependency** | `resolver.go:190–199` (`NewGitCache` + `Resolve`) | network + `$HOME` write | **opt-in** | **UNGUARDED** (measured) |
| 10 | git cache root creation | `internal/pkg/gitcache.go:20–29` `NewGitCache` `MkdirAll ~/.ailang/cache/git` | `$HOME` write | via 9,11,12,14,15,19 | **UNGUARDED** |
| 11 | `git clone` | `gitcache.go:82–100` `ensureClone` | network + `$HOME` write | via 9 | **UNGUARDED** (measured, `file://`) |
| 12 | `git fetch` on a cached clone | `gitcache.go:104–122` `fetch` | network + `$HOME` write | via 9 | **UNGUARDED** |
| 13 | `git checkout` in the cache | `gitcache.go:125–133` `checkout` | `$HOME` write | via 9 | **UNGUARDED** |
| 14 | loader git case (`check`/`ai-check`/`test`/`iface`/`run`/AI-provider init) | `internal/pkg/loader.go:148–156` | `$HOME` write (`MkdirAll`) | **default** | **UNGUARDED** (measured) |
| 15 | `lock --check` → `CheckLock` → 9 | `internal/pkg/lockcheck.go:43–55` | network + `$HOME` write | none (flag not in schema) | **UNGUARDED** (measured) |
| 16 | `install --path` → `EnsureLock` → 9 | `internal/pkg/bin.go:116–126`, `pkg_install.go:45`, `pkg_bin.go:81` | network + `$HOME` write | none | **UNGUARDED** (measured clone) |
| 17 | `install` → `[bin]` shim write | `bin.go:138–139` `DefaultBinDir`, `:170–184` `WriteShim`; `pkg_bin.go:104` | `$HOME` write (`~/.ailang/bin`) | none | **UNGUARDED** |
| 18 | `bin uninstall` → shim delete | `bin.go:314–326` `RemoveShim`; `pkg_bin.go:210` | `$HOME` write | none | **UNGUARDED** |
| 19 | `add --git` | `cmd/ailang/pkg_commands.go:116–122` | network + `$HOME` write | none | **UNGUARDED** |
| 20 | `lock` (prior lock exists) → `warnSilentRatchet` | `pkg_commands.go:531`, `:611` `messaging.OpenStore` → `~/.ailang/state/collaboration.db` | `$HOME` write | **opt-in** | **UNGUARDED** (measured) |
| 21 | `pkg docs` / `pkg-docs` registry fallback | `cmd/ailang/pkg_docs.go:80–109` | network + `$HOME` write | default | **guarded** |
| 22 | `pkg info` / `versions` / `stats` → `registryAPIGet` | `cmd/ailang/pkg_info.go:23–38` (calls `:60`, `:209`, `:313`) | network | none | **UNGUARDED** (measured 1 req each) |
| 23 | `pkg provenance` | `cmd/ailang/pkg_provenance.go:25–26` | network | none | **guarded** (measured 0 req) |
| 24 | `pkg history` | `pkg_provenance.go:86–89` raw `http.Get` | network | none | **UNGUARDED** (measured) |
| 25 | `install <vendor/name>` (registry) incl. its shim step `pkg_install.go:142` | `pkg_install.go:68–142` | network + `$HOME` write | none | **guarded** (`FetchMetadata` at `:90` refuses first; measured 0 req) |
| 26 | `install --path` entry (shim + lock) | `pkg_install.go:45` → `pkg_bin.go:42–104` | network + `$HOME` write | none | **UNGUARDED** |
| 27 | `publish` registry reads | `cmd/ailang/pkg_publish.go:380–381`, `:450` | network | none | **guarded** |
| 28 | `publish` tarball upload | `pkg_publish.go:148`, `:239–286` (`http.Client` POST) | network | none | **UNGUARDED** |
| 29 | `publish` → `emitPublishMessages` | `pkg_publish.go:155`, `:315` `openPkgMsgStore` | `$HOME` write | none | **UNGUARDED** |
| 30 | `publish` → cascade Pub/Sub | `pkg_publish.go:420`, `:516`, `:604–617` | network | none | **UNGUARDED** |
| 31 | `unpublish` | `cmd/ailang/pkg_unpublish.go:85–92` (DELETE) | network | none | **UNGUARDED** |
| 32 | `pkg key create\|list\|revoke` | `cmd/ailang/pkg_keys.go:141–150` `registryAdminRequest` | network | none | **UNGUARDED** |
| 33 | `pkg notify-upgrade` | `cmd/ailang/pkg_msg.go:166`, `:266–268` `openPkgMsgStore` | `$HOME` write | none | **UNGUARDED** |
| 34 | `pkg cascade status` | `cmd/ailang/pkg_cascade_status.go:71–77`, `:150`, `:170` | network | none | **guarded** (measured 0 req) |
| 35 | `pkg quality` overlap check | `cmd/ailang/pkg_quality.go:142` | network | none | **guarded** (error folded into the report) |
| 36 | `pkg docs` git-cache *read* | `pkg_docs.go:66–78` | read of a cache rows 9–13 can write | default | **UNGUARDED** (trust gap; closed by gating 9–13) |

**Count: 36 sites. 14 guarded (rows 1–8, 21, 23, 25, 27, 34, 35; rows 4, 5, 7, 8 and 25 are guarded transitively). 22 unguarded (rows 9–20, 22, 24, 26, 28–33, 36).** Eight of the unguarded sites are reachable through policy-tool:

- rows 9–13 and 20 through `lock` (opt-in);
- row 14 through `check`, `test` and `iface` (default);
- row 36 through `pkg-docs` (default).

**Sites with no network or `$HOME` authority** were measured or read and excluded:

- `pkg tree` (`resolver.go:348–383` reads manifests only; measured no `$HOME` change);
- `pkg affected-by` (scans lockfiles; measured no change);
- `pkg bin list` (read-only; measured);
- `init package` (writes in cwd only, `pkg_init.go:149`);
- `pkg quality`'s `exec ailang test` (`pkg_quality.go:188`) and `iface_subprocess.go` / `publish_validator.go` execs. These run local `ailang` children that inherit `AILANG_AGENT_POLICY`, so they are confined by the same gate.

**Related trust gaps** (not I/O sites, but the same poisoning class):

- **T1**: git `content_hash` is never verified at check time (`lockfile.go:163` skips non-`path`; `registry_hash.go:21` is `registry` only).
- **T2**: the git cache directory is keyed by URL only (`gitcache.go:76–79`), and the loader ignores `GitRev`. Two projects pinning different tags of one URL share one working tree, so the loader compiles against whichever was checked out last (V16).
- **T3**: the loader falls back to a lock-supplied absolute `Path` for git and registry entries (`loader.go:160–164`, `:176–179`). This is a *read* escape via an edited `ailang.lock`. It is out of scope here; see Non-Goals.

## Goals

**Primary goal:** under `AILANG_AGENT_POLICY`, no code reachable from a package verb or the package loader can open a network connection or write under the user's home. This is guaranteed by construction, through one gate per authority, and a new I/O site that bypasses the gate fails CI.

**Success metrics:**

- All 22 unguarded rows become refusals or write-free reads. The three V3 repros invert: zero entries are created under `$HOME`, and the git shim logs no `clone`/`fetch`.
- A table-driven matrix runs **every** `pkgSubcommands()` verb, plus `lock --check`, `install --path`, `check`, `test`, `iface` and `run`, under confinement with a git dependency. It asserts zero requests at a loopback recorder, zero network-capable git invocations, and a byte-identical `$HOME`, an outside canary dir, and sandbox apart from the op's ledger. A new verb added without a matrix row fails the test.
- An AST boundary test fails if any non-test file in `internal/pkg/` or the package-verb files uses raw `net/http` clients, `exec.Command("git", …)`, `os.UserHomeDir`, `messaging.OpenStore` or `pubsub.NewClient` outside the chokepoint.
- A tampered file in a provisioned git dependency fails `ailang check` (T1/T2).
- Unconfined behaviour is unchanged. Existing `TestPkgRegistryUnconfinedWorkflows`, gitcache, lock and install tests stay green.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Gate by **authority** (network, home-write) in one file, `internal/pkg/authority.go`. The HTTP gate sits in the `RoundTripper`, so every request through a package client is checked even at a call site that forgot. | This replaces #1704's per-function gating, which is the root cause. Every package I/O site migrates to it. | agent | design | med |
| D2: A confined `lock` with a git dependency **refuses**. The one exception is an entry already pinned in the existing `ailang.lock` (same `git_url` + `tag`/`rev` + `subdir`) whose provisioned directory hashes to the locked `content_hash`. That entry is **carried forward with no git invocation**. | Pure refusal makes `lock` unusable in any confined project that has a git dependency, even when only a path dependency changed. Carry-forward keeps it usable without network or writes, but adds a resolver mode. | human | design | med |
| D3: When `AILANG_PACKAGE_ROOT` is set, git dependencies also resolve **only** from it, at `<root>/<vendor>/<name>/<version>` (the registry layout), verified by `content_hash`. There is no `$HOME` git-cache fallback. | This extends #1704's D3 ruling (exclusive root) to git. Today git ignores the root, so a host that sets it still has confined `check` reading the `$HOME` git cache. Choosing the same layout means one provisioning recipe. | human | design | med |
| D4: A git `content_hash` mismatch at `check` is a **hard error**, like registry under #1704 D5, not a warning like path dependencies. | It closes T1/T2. It can break an unconfined checkout that today compiles silently against the wrong revision of a shared URL-keyed clone, which is the correct outcome but visible to users. | human | design | med |
| D5: Operator-only verbs also declare `Authority` in the `Command` table and are refused at dispatch when confined: `publish`, `unpublish`, `key`, `install`, `add`, `notify-upgrade`, `bin` (uninstall), `cascade`. | This gives an early, uniform refusal before any partial work, and a completeness test forces every verb to declare a value. The I/O gate (D1) is still the guarantee; this column is a usability layer, not the boundary. | agent | design | low |
| D6: The bypass detector is a scoped Go AST test, not a `forbidigo` rule. | `forbidigo` patterns are repo-wide. `http.Get` and `exec.Command("git")` are legitimate in the coordinator, the eval harness and elsewhere, so a repo-wide ban would need dozens of path exclusions. The test scans exactly the package surface, and that file set is derived from the command table. | agent | design | low |
| D7: Best-effort messaging writes reachable from `lock` (`warnSilentRatchet`) are **skipped under confinement with a one-line stderr note** rather than refused. The operator verbs that write messages (`publish`, `notify-upgrade`) refuse via D5. | The silent-ratchet warning is advisory, and failing `lock` over it would punish the agent for an operator-side feature. Skipping it without a note would be a silent fallback (CLAUDE.md principle 2). | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] D2 (**Ruled 2026-10-26 (Mark, attended): accepted as recommended.**): is a confined `lock` with git dependencies a pure refusal, or does it carry forward entries already pinned and provisioned? (Recommendation: carry forward, because it is write-free and network-free by construction and keeps `lock` useful.)
- [x] D3 (**Ruled 2026-10-26 (Mark, attended): accepted as recommended.**): do git dependencies resolve exclusively from `AILANG_PACKAGE_ROOT` when it is set, using the `<vendor>/<name>/<version>` layout? (Recommendation: yes, consistent with the #1704 D3 ruling of 2026-10-08.)
- [x] D4 (**Ruled 2026-10-26 (Mark, attended): accepted as recommended.**): is a git `content_hash` mismatch at `check` a hard failure now, or a warning during a deprecation window? (Recommendation: hard-fail now, consistent with the #1704 D5 ruling.)

## Solution Design

### Overview

There is one gate per authority, owned by one file, and three mechanisms prevent drift:

1. **`internal/pkg/authority.go` (the chokepoint).** It is the only place in the package surface allowed to:
   - build an HTTP client;
   - run `git`;
   - resolve or create anything under `~/.ailang`;
   - open the messaging store or a Pub/Sub publisher on behalf of a package verb.

   Each primitive calls `Require(authority, what)` before acting.
2. **An AST boundary test** that fails when a non-test file in the package surface performs one of those operations any other way.
3. **An end-to-end confinement matrix** over the command table. It fails when any verb, including one added later, produces a request, a network git invocation, or a byte of change outside the allowed roots.

### Architecture

**`internal/pkg/authority.go`** (new, about 180 LOC). It absorbs `confinement.go`, which keeps `Confined`, `RefuseIfConfined` and `EnsureRegistryCacheDir` as thin wrappers so the pinned message text and the existing tests stay valid.

```go
type Authority int
const (
    AuthNetwork   Authority = iota // any outbound connection: HTTP, git transport, Pub/Sub, Firestore
    AuthHomeWrite                  // create, modify or delete anything under the user's ~/.ailang
)

// Require returns the canonical refusal when confined. Single source of the text.
func Require(a Authority, what string) error

// HTTPClient is the ONLY *http.Client package code may hold. Its Transport wraps
// http.DefaultTransport and calls Require(AuthNetwork, what) in RoundTrip, so a
// request built at a new call site is gated without the site knowing.
func HTTPClient(what string, timeout time.Duration) *http.Client

// Git runs git with a fixed hardened prefix
// (-c protocol.ext.allow=never, GIT_TERMINAL_PROMPT=0, `--` before positional URLs).
// clone/fetch/ls-remote/pull/submodule require AuthNetwork; any invocation whose Dir is
// under UserDir(...) and is not in the read-only set {rev-parse, cat-file} requires
// AuthHomeWrite. Under confinement the resolver never calls it (D2), so the read-only
// set is for unconfined use only.
func Git(dir string, args ...string) ([]byte, error)

// UserDir computes ~/.ailang/<sub> without creating it.
// EnsureUserDir requires AuthHomeWrite, then MkdirAll.
func UserDir(sub string) (string, error)
func EnsureUserDir(sub string) (string, error)
```

**Migrations** (each removes a raw primitive from a call site):

| Site rows | Change |
|---|---|
| 1–3, 22, 24, 28, 31, 32 | `RegistryClient.httpClient`, `registryAPIGet`, `pkgHistoryCommand`, `uploadTarball`, `pkgUnpublishCommand` and `registryAdminRequest` obtain clients from `pkg.HTTPClient`. The `FetchX` pre-checks stay, because they give a better message before work starts; the transport is the backstop. |
| 9–13, 19 | `GitCache` gets its base from `UserDir("cache/git")` (pure) and calls `EnsureUserDir` only inside `ensureClone`. `ensureClone`, `fetch`, `checkout` and `revParse` call `pkg.Git`. `NewGitCache` no longer creates anything, mirroring #1704's D6 split for the registry cache. |
| 9, 15, 16 | The resolver's git branch, when confined, takes the D2 path. If a prior lock entry matches and the provisioned dir (`PackageDir` under `AILANG_PACKAGE_ROOT` per D3, else the existing `$HOME` clone) hashes to the locked `content_hash`, it emits the carried entry. Otherwise it returns `Require(AuthNetwork, "resolving git dependency <name>")` with the remedy "run 'ailang lock' outside the sandbox, or provision AILANG_PACKAGE_ROOT". `ResolveDependencies` gains the previous lock as an input. `CheckLock` and `EnsureLock` already have it, and `pkgLockCommand` loads it at `pkg_commands.go:498`. |
| 14 | The loader's git case uses `GitCache.CacheDir` from a non-creating cache (or `PackageDir` when the root is set, D3) and never calls `MkdirAll`. |
| 17, 18, 26 | `DefaultBinDir` → `UserDir("bin")`. `WriteShim` and `RemoveShim` call `EnsureUserDir("bin")` and `Require(AuthHomeWrite, …)` respectively. |
| 20, 29, 33 | `openPkgMsgStore` becomes the single opener, and `warnSilentRatchet` calls it instead of `messaging.OpenStore`. It takes a mode. Write mode requires `AuthHomeWrite`. Best-effort mode returns `(nil, errSkippedConfined)` under confinement, and the caller prints `note: silent-ratchet check skipped (confined)` (D7). |
| 30 | `newCascadePublisher` calls `Require(AuthNetwork, "publishing cascade notifications")` before `pubsub.NewClient`. |
| — | `cmd/ailang` `Command` gains `Authority []pkg.Authority`. `pkgGroupCommand` and the legacy top-level rows refuse a declared verb at dispatch when confined (D5). |

**Trust fixes** (T1/T2): `ValidateContentHashesFrom` grows a `git` branch next to `validateRegistryContentHashes`. It resolves the directory exactly as the loader does (shared helper `gitPackageDir(locked)`, so the two cannot diverge), hashes it, and returns a `RegistryContentError`-class error on mismatch, which the pipeline already treats as fatal (`pipeline/package_resolver.go:68–72`) per D4. A different revision checked out in the shared URL-keyed clone now surfaces as drift ("run 'ailang lock'") instead of a silent wrong compile.

**Why the transport and the AST test, not more `RefuseIfConfined` calls:** #1704 added guards at the three sites it knew about and wrote tests for those three. This doc's audit found 22 more unguarded sites. A `RoundTripper`-level gate makes the *HTTP* class closed by construction. The AST test closes the *bypass* class: a raw client, a raw `git`, a raw home dir. The matrix closes the *entry-point* class, because its row set is computed from the command table. Each layer catches what the others cannot. The AST test cannot see a gated function called at the wrong time. The matrix cannot see a code path its fixture does not exercise. The transport cannot see `exec` or file writes.

### Alternatives considered

| Alternative | Verdict | Why |
|---|---|---|
| **A. Add `RefuseIfConfined` at the 22 sites** (the #1704 shape) | Rejected | It fixes today's list, and nothing makes site #37 fail. It is the anti-pattern this issue is an instance of. |
| **B. Authority chokepoint + transport gate + AST boundary test + command-table matrix** | **Chosen** | One grant point per authority. Bypasses and new verbs fail CI. Unconfined behaviour is unchanged. |
| **C. Process-wide kill switch when confined**: replace `http.DefaultTransport` at startup and strip `git` from `PATH` | Rejected as the primary mechanism | It does not cover `$HOME` writes, Pub/Sub's gRPC transport, or clients with their own `Transport`. It would also break `design_quorum`, which is reachable under confinement via `cli_allow` (`cli_ops.go:108`) and legitimately calls model APIs. It also adds global mutable state. Its useful idea, gating at the transport, is kept but scoped to package clients (D1). |
| **D. OS-level sandbox of the policy-tool child** (sandbox-exec / Landlock + network namespace / gVisor) | Deferred (Future Work) | It is the strongest option, because it catches compile-cache and any future writes. But it is platform-specific, Cloud Run's runtime constrains it, and it is a multi-week effort. M-EXECUTOR-POLICY-HARDENING chose in-process enforcement (`os.Root`, a single authorising dialer in `internal/effects/net_authorize.go`) for the same reasons, and this doc follows that precedent. |
| **E. Policy-tool side only**: drop `lock` from `cliSchemas`, or widen `lockWrites` to name `$HOME` | Rejected | Policy-tool cannot see what a child does (#1704 rationale). Dropping `lock` leaves the default-allowed `check` write (row 14) and every non-policy-tool confined caller. Widening the ledger legitimises the hole. |

### Conflict Surface (shared machinery touched)

This change touches none of the parser, lexer, AST, type checker, elaborator, iface, codegen, evaluator, VM, effects or `exec.go` code. The shared machinery is the package resolver, loader and HTTP clients:

| Consumer | File:line | Confined? | Effect |
|---|---|---|---|
| `registry-validator` | `cmd/registry-validator/validate.go:58`, `main.go:196` | no (operator service, never sets the variable) | none; it uses the same chokepoint unconfined |
| coordinator package agents | `internal/coordinator/package_agents.go:156` | no (daemon) | none |
| runner asset dirs | `internal/runner/pkg_dirs.go:25` | possibly | loader read becomes write-free (row 14) |
| AI-provider init | `cmd/ailang/configdriven_init.go:104` | possibly | same |
| `ailang examples` | `cmd/ailang/examples_corpus.go:87` | yes | none (it reads the same signal through its own `config.AgentPolicy() != ""` check; see Deferred Decisions) |
| policy-tool `lock` op | `internal/policytool/cli_ops.go:71,107` | yes | the `ailang.lock`-only ledger becomes true |

**Must still work:**

- unconfined `lock`, `add --git`, `install`, `install --path` and `publish` (the existing tests `TestPkgRegistryUnconfinedWorkflows` `cmd/ailang/pkg_docs_confinement_test.go`, `TestGitCache_*` `internal/pkg/gitcache_test.go`, `pkg_lock_*_test.go`, `pkg_install_test.go`, `pkg_publish_test.go`);
- confined `check` over a provisioned git dependency;
- confined `pkg-docs` over a provisioned package;
- #1704's A1–A5.

**Deliberately changes:**

- the 22 unguarded rows above under confinement;
- `NewGitCache` no longer creates its directory;
- the git hash drift becomes fatal at `check` (D4).

### Implementation Plan

**Phase 1: chokepoint + git gate (closes the P0)** (~1.5 days)

- [ ] `internal/pkg/authority.go`: `Require`, `HTTPClient` (gating `RoundTripper`), `Git`, `UserDir`/`EnsureUserDir`. Fold in `confinement.go`.
- [ ] `gitcache.go` purity split. All git invocations go through `pkg.Git`, with `--` before the URL.
- [ ] Resolver git branch: confined → D2 carry-forward or refusal. Thread the previous lock into `ResolveDependencies`, `ResolveLock` and `CheckLock`.
- [ ] Loader git case: non-creating.
- [ ] Bin shims and messaging opener (D7). Cascade publisher gate.

**Phase 2: HTTP migration + command-table authority** (~1 day)

- [ ] Migrate rows 22, 24, 28, 31 and 32 and `RegistryClient` to `pkg.HTTPClient`.
- [ ] Add the `Command.Authority` column, a dispatch refusal, and a completeness test (every `pkgSubcommands()` entry declares a value, empty meaning "none").

**Phase 3: trust + package root for git** (~0.5 day)

- [ ] Git branch in `ValidateContentHashesFrom` via a shared `gitPackageDir`, made fatal (D4).
- [ ] `AILANG_PACKAGE_ROOT` resolution for git (D3). Update the `config/paths.go` doc row to say "registry and git packages".

**Phase 4: proof** (~1 day)

- [ ] AST boundary test, confinement matrix and policy-tool route test (Testing Strategy).
- [ ] Mutation-test both new tests (AC2) via scratchpad copies.
- [ ] Re-run `scratchpad/dd3/repro*.sh` against the built binary and attach the output to the sprint report.
- [ ] Gates: `make test-core`, focused `go test ./internal/pkg/... ./internal/policytool/... ./cmd/ailang/ -run 'Confin|Git|Lock|Install|Pkg'`, `make lint`, `make check-boundaries`, `make check-file-sizes`.
- [ ] Docs: `SECURITY.md` confinement section, `docs/docs/reference/std-package.md` (git resolution order, hash check), env-vars row text. Write the changelog fragment `changelogs/unreleased/2026-10-2x-pkg-authority-chokepoint.md`.

### Files to Modify/Create

**New files:**

- `internal/pkg/authority.go`: the chokepoint, ~180 LOC (absorbs `confinement.go`).
- `internal/pkg/authority_boundary_test.go`: AST bypass detector, ~150 LOC.
- `internal/pkg/authority_test.go`: unit tests for `Require`, the transport, `Git`'s network classification and `UserDir` purity, ~150 LOC.
- `cmd/ailang/pkg_confinement_matrix_test.go`: end-to-end matrix plus controls, ~350 LOC.

**Modified files:**

- `internal/pkg/confinement.go`: reduced to wrappers, or deleted with its callers moved, ~−20 LOC.
- `internal/pkg/gitcache.go`: purity split, `pkg.Git`, `--`, ~40 LOC.
- `internal/pkg/resolver.go`: confined git path (D2), previous-lock input, ~60 LOC.
- `internal/pkg/lockcheck.go`: pass the previous lock through, ~10 LOC.
- `internal/pkg/loader.go`: non-creating git dir, shared `gitPackageDir`, D3, ~25 LOC.
- `internal/pkg/lockfile.go`: git hash branch (or `registry_hash.go`), ~30 LOC.
- `internal/pkg/bin.go`: `UserDir`/`EnsureUserDir`/`Require` in shim write and remove, ~10 LOC.
- `internal/pkg/registry.go`: `httpClient` from `HTTPClient`; `RegistryCacheDir` via `UserDir`, ~10 LOC.
- `cmd/ailang/pkg_info.go`: `registryAPIGet` via `pkg.HTTPClient`, ~5 LOC.
- `cmd/ailang/pkg_provenance.go`: history via `pkg.HTTPClient`, ~5 LOC.
- `cmd/ailang/pkg_publish.go`: upload client, store opener mode, cascade gate, ~15 LOC.
- `cmd/ailang/pkg_unpublish.go`: client, ~3 LOC.
- `cmd/ailang/pkg_keys.go`: client, ~3 LOC.
- `cmd/ailang/pkg_msg.go`: opener with mode, ~10 LOC.
- `cmd/ailang/pkg_commands.go`: `warnSilentRatchet` via the opener (D7); `lock` passes the previous lock, ~15 LOC.
- `cmd/ailang/commands_pkg.go`: `Authority` declarations and dispatch refusal, ~40 LOC.
- `cmd/ailang/commands_groups.go`: legacy rows inherit the declaration, ~5 LOC.
- `internal/config/paths.go`: `EnvPackageRoot` doc text, ~1 LOC.
- `SECURITY.md`, `docs/docs/reference/std-package.md`: documentation, ~40 LOC.

## Examples

### Example 1: the #1720 repro, before and after

**Before** (origin/dev `772703831`, measured):

```
$ AILANG_AGENT_POLICY=p.toml HOME=$EMPTY ailang lock
Cloning into '$EMPTY/.ailang/cache/git/366de57b0bf5f04f'...
✓ Generated ailang.lock (1 packages)
```

**After** (no prior lock):

```
$ AILANG_AGENT_POLICY=p.toml HOME=$EMPTY ailang lock
Error: dependency resolution failed: confined by AILANG_AGENT_POLICY: resolving git
dependency test/gitlib is operator authority — run 'ailang lock' outside the sandbox
or provision AILANG_PACKAGE_ROOT
$ ls -A $EMPTY          # nothing
```

**After** (D2: prior lock pins the entry and the provisioned clone hashes to it):

```
$ AILANG_AGENT_POLICY=p.toml ailang lock
✓ Generated ailang.lock (2 packages)
  → test/gitlib@0.1.0 (git, carried forward: provisioned content matches lock)
  → test/shared@0.2.0 (path: ../shared)
```

### Example 2: a bypass caught at review time

```go
// someone adds to cmd/ailang/pkg_info.go:
resp, err := http.Get(registryValidatorURL() + "/api/trending")
```

```
--- FAIL: TestPkgAuthorityBoundary
    cmd/ailang/pkg_info.go:412: http.Get outside internal/pkg/authority.go —
    use pkg.HTTPClient(what, timeout) so the request is gated under AILANG_AGENT_POLICY
```

## Success Criteria

- [ ] **AC1: confinement matrix** (`cmd/ailang/pkg_confinement_matrix_test.go`, built binary via the existing `buildAilang` helper, `main_test.go:365`). Fixture:
  - a temp `HOME`;
  - a sandbox project whose manifest has (a) a `file://` git dependency and (b) an `http://127.0.0.1:<port>/x.git` git dependency served by a loopback recorder;
  - `AILANG_REGISTRY` and `AILANG_REGISTRY_VALIDATOR` pointed at the same recorder;
  - an `AILANG_CACHE_DIR` temp dir (an allowed root, mirroring `runAilang`);
  - an *outside* canary dir;
  - a `PATH`-first `git` shim that logs argv and then execs the real git.

  Rows are **generated** from `pkgSubcommands()` (each verb with a fixture argv table), plus `lock --check`, `install --path .`, `check`, `ai-check`, `test`, `iface` and `run`. A verb in `pkgSubcommands()` with no argv entry fails the test. For each row under `AILANG_AGENT_POLICY`, the test asserts:
  - (i) the recorder logged **0 requests**;
  - (ii) the git shim logged **no `clone`/`fetch`/`ls-remote`/`pull`/`submodule`**;
  - (iii) a path+mode+size+sha256 snapshot of `HOME`, the outside canary and the sandbox is **byte-identical**, except for paths in that op's declared ledger (`ailang.lock` for a successful `lock`);
  - (iv) a refusing row's stderr contains `confined by AILANG_AGENT_POLICY`, and an allowed row (`check`/`tree`/`docs` over provisioned deps) exits 0.
- [ ] **AC2: negative controls.** These prove the instruments can see a positive.
  - (a) **Unconfined arm.** The same fixture with `AILANG_AGENT_POLICY` unset, for `lock`, `add --git`, `install --path`, `pkg info`, `pkg history` and `lock` with a prior lock, **must** register ≥1 recorder request or git-shim `clone`, and a non-empty `$HOME` diff. If the unconfined arm is quiet, the test fails as "instrument blind".
  - (b) **Mutation.** Recorded in the sprint report, done on scratchpad copies, never `git checkout <file>`:
    - removing `Require` from `pkg.Git`'s network branch turns the `lock`/`lock --check`/`install --path` rows red;
    - removing it from the `HTTPClient` transport turns `pkg info`/`history` red;
    - re-adding `MkdirAll` to `NewGitCache` turns the `check` row red;
    - inserting a raw `http.Get` into `pkg_info.go` turns AC3 red.
  - (c) **Positive confined control.** An operator-provisioned git dependency (pre-cloned by an unconfined `lock` in a separate `HOME`, then copied under `AILANG_PACKAGE_ROOT`) gives confined `check` exit 0 and zero diff, and confined `lock` exit 0 with only `ailang.lock` changed (D2).
- [ ] **AC3: boundary test** (`internal/pkg/authority_boundary_test.go`). It parses every non-test `.go` file in `internal/pkg/`, plus every `cmd/ailang` file that declares a function referenced as `Run` by `pkgSubcommands()` or that the `openPkgMsgStore`/`newCascadePublisher` helpers live in. The `cmd` file set is computed from the AST, not hard-coded. It fails on any of the following outside `authority.go`:
  - `http.Get`/`Post`/`Head`/`PostForm`;
  - `http.DefaultClient`;
  - an `http.Client{…}` composite literal;
  - `exec.Command`/`CommandContext` whose first argument is the string `"git"`;
  - `os.UserHomeDir`;
  - `messaging.OpenStore` outside `openPkgMsgStore`;
  - `pubsub.NewClient` outside `newCascadePublisher`.

  An allowlist names the local-exec sites (`iface_subprocess.go`, `publish_validator.go`, `pkg_quality.go:188`), each with a one-line reason.
- [ ] **AC4: policy-tool route.** A test dispatches `{"op":"lock"}` through `policytool.Host` with `cli_allow = ["lock"]` and the real binary. The response is a refusal and the AC1 snapshot diff is empty.
- [ ] **AC5: git trust (T1/T2).** Two cases:
  - appending to a `.ail` in a provisioned git dependency makes `ailang check` exit ≠ 0 with a drift message naming the package and both hashes;
  - checking out a different tag in the shared URL-keyed clone does the same.
- [ ] **AC6: package root for git (D3).** With `AILANG_PACKAGE_ROOT` set (chmod 0555 fixture), confined `check` resolves the git dependency from `<root>/<vendor>/<name>/<version>` and never stats `$HOME/.ailang/cache/git`. A missing package gives the named "not provisioned" error.
- [ ] **AC7: unconfined unchanged.** The existing tests listed under Conflict Surface "Must still work" pass unmodified, apart from tests that assert `NewGitCache` creates its directory, which are rewritten per the testing policy.
- [ ] **AC8: live repro inverted.** `repro.sh`, `repro_http.sh` and `repro_home.sh` (the V3/V7 scripts) re-run against the built binary show zero `$HOME` entries and zero requests in the confined arms.
- [ ] **AC9: canonical message.** The `Require` text is pinned by a unit test. The existing pinned `RefuseIfConfined("writing the registry cache")` text is unchanged.
- [ ] **AC10: hardening.** Every git argv built by `pkg.Git` has `--` before the URL, and the manifest validator rejects a `git` value beginning with `-`. This is defence in depth only: measured at git 2.54, a leading-dash URL is not exploitable today (V14).
- [ ] All gates in Phase 4 pass. Documentation and the changelog fragment are written.

## Testing Strategy

**Unit tests:**

- `Require` message pinning.
- The gating `RoundTripper`, using an `httptest.Server` and counting hits: confined gives 0, unconfined gives 1.
- `pkg.Git` classification: the network subcommand set, and the `$HOME`-dir write set versus read-only.
- `UserDir` purity: creates nothing. `EnsureUserDir` refuses when confined.
- Resolver D2 matrix: prior entry matches → carry; content mismatch → refusal; URL, tag or subdir changed → refusal; no prior lock → refusal.
- `ValidateContentHashesFrom` git branch: match, mismatch, missing dir.

**Integration tests:** AC1–AC6. The shape follows `cmd/ailang/run_policy_containment_test.go` (`buildAilang`, `newContainmentE2E`, a git config with `GIT_CONFIG_GLOBAL=/dev/null`) and `cmd/ailang/pkg_docs_confinement_test.go` (temp `HOME` + `os.Stat` absence).

**Manual testing:** the three `scratchpad/dd3` repro scripts, plus the AILANG World host shape: an `AILANG_PACKAGE_ROOT` holding one registry and one git package, and a confined agent checking a project that imports both.

## Deferred Decisions

- Whether `Confined()` moves to `internal/config` (`config.Confined()`) so that `examples_corpus.go:87` and `pkg` share one definition of "confined". This is one concept with two implementations today. Agent may resolve; recommended yes.
- The exact refusal tail for git ("run 'ailang lock' outside the sandbox …") versus registry ("… run 'ailang install' …"). `Require` takes a remedy argument. Agent may resolve, provided the AC9 pinned registry text is unchanged.
- Whether the boundary test also scans `cmd/registry-validator`. It is operator-only today. Agent may resolve; recommended no, to keep the scope at the package surface reachable by an agent.
- Whether the D5 `Authority` column extends beyond the `pkg` group to every top-level command. Agent may resolve; recommended to land it for `pkg` now and file a follow-up for the rest.

## Non-Goals

- **Read confinement of lock-supplied paths (T3).** An edited `ailang.lock` can point a `path` entry, or a `git`/`registry` entry through the legacy `Path` fallback (`loader.go:160–164`, `:176–179`), at any readable host directory. That is a *read* escape. Fixing it changes path-dependency semantics (`../shared` is legitimate) and needs its own design. File as a follow-up issue.
- **Re-keying the git cache by revision** (`<urlhash>/<rev>`). T2 becomes *detected* by the D4 hash check rather than *prevented*. Prevention is Future Work.
- **OS-level sandboxing of the CLI child** (Alternative D).
- **Gating non-package network** reachable under confinement, such as `design_quorum`'s model calls. That network is deliberate and policy-admitted.
- **`pkg-docs` matching any vendor's `packages/<name>/AGENT.md` in the git cache** (`pkg_docs.go:72`). This is a lookup-precision issue, not an authority issue. The read becomes trustworthy once rows 9–13 are gated.

## Timeline

- **Day 1–1.5:** Phase 1. Rows 9–20 closed, and the repro scripts are clean.
- **Day 2.5:** Phase 2. Rows 22, 24, 28–33 closed, plus the command-table authority column.
- **Day 3:** Phase 3. Git hash check and package root.
- **Day 4:** Phase 4. Matrix, boundary test, mutation evidence, docs and gates.

**Total: ~4 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| D4 hard-fail breaks unconfined users whose two projects share a git URL at different tags | Med | They are silently compiling against the wrong revision today. The error names the package and says `run 'ailang lock'`. Call it out in the changelog. Rev-keyed cache is Future Work. |
| An operator shell exports `AILANG_AGENT_POLICY` and its `publish`/`lock` start refusing | Med | This is the same accepted trade-off as #1552 and #1704. The refusal names the variable and the remedy. |
| `ResolveDependencies` gains a parameter, and `registry-validator` calls it | Low | Pass `nil` (no prior lock). It is unconfined, so behaviour is unchanged. |
| The matrix is slow or flaky (binary build, git, loopback server) | Low | Reuse `buildAilang` (built once per package run) and file-local git. There is no real network. Use bounded per-row timeouts. |
| The AST test is too strict for a legitimate future site | Low | The allowlist requires a reason string, and the review sees it. That visibility is the point. |
| `git checkout` in the `$HOME` clone during an unconfined `lock` races a concurrent confined `check` reading it | Low | The D4 hash check turns a torn read into a loud drift error, not a silent wrong compile. |

## Related Documents

**Implemented (inform the design):**

- [M-EXECUTOR-POLICY-HARDENING](../../implemented/v0_41_0/m-executor-policy-hardening.md): origin of `AILANG_AGENT_POLICY` as the materialised policy path and of `policy-tool`. Its single authorising dialer (`internal/effects/net_authorize.go`) is the in-process chokepoint precedent this doc follows.
- [M-AGENT-AILANG-ONLY-EXECUTION](../../implemented/v0_39_0/m-agent-ailang-only-execution.md): the confined lane.
- [m-pkg-lock-portability](../../implemented/v0_10_0/m-pkg-lock-portability.md): why git lock entries omit `Path` and resolve at runtime from `GitURL` (the URL-keyed cache in T2).

**Planned (overlap checked):**

- [M-PKG-REGISTRY-CONFINEMENT](../v0_52_6/m-pkg-registry-confinement.md) (#1607/#1608): the direct predecessor. This doc supersedes its D1 (per-function gates) with per-authority gates. It keeps its D2 (signal), D3 (exclusive root, extended to git here) and D5 (hard-fail hashes, extended to git). It also corrects its false Non-Goal premise about git. Neural similarity 0.41: distinct, because that doc gates the registry transport and this one gates every transport.
- [m-pkg-asset-lock-resolution](../../implemented/v0_56_0/m-pkg-asset-lock-resolution-sprint-plan.md): shares `LockedPackageDirs` (`runner/pkg_dirs.go`). It does not overlap, because it changes which dirs are read, not authority.

## References

- [Design Axioms](/docs/references/axioms)
- Issue #1720; predecessor issues #1607, #1608, #1552, #1554; PR #1704.
- `changelogs/v0.32-current.md:1561`: the recorded known gap.

## Verification Log

All reads are at origin/dev `772703831` (detached worktree). All runs used a binary built from it (`go build -o scratchpad/dd3/ailang ./cmd/ailang`), with scratch fixtures under `scratchpad/dd3/`.

| # | Claim | Evidence |
|---|-------|----------|
| V1 | "Confined" = `AILANG_AGENT_POLICY` non-empty; set by the executor and by policy-tool | `internal/pkg/confinement.go:11`; `internal/config/job.go:57,117`; `internal/executor/environment.go:163`; `internal/policytool/cli_ops.go:394–403` |
| V2 | The guard today covers exactly the registry client, the registry cache ensure, and `pkg_docs` | `grep -rn RefuseIfConfined --include=*.go` gives `confinement.go:14,24`, `registry.go:63,122,150`, `pkg_docs.go:80`, and nothing else |
| V3 | Confined `lock` with a `file://` git dependency clones into `$HOME/.ailang/cache/git`; with a prior lock it also creates `$HOME/.ailang/state/collaboration.db` | Live: `scratchpad/dd3/repro.sh`, output `repro.out` (cases A and C) |
| V4 | Confined `check` on a project with a git dependency creates `$HOME/.ailang/cache/git` | Live: `repro.out` case B; mechanism `loader.go:151` `NewGitCache` → `gitcache.go:26` `MkdirAll`; reached via `pipeline/package_resolver.go:77` → `loadDepManifest` → `LoadManifestByName` (`loader.go:246`) |
| V5 | `lock --check` and `install --path` clone under confinement | Live: `scratchpad/dd3/repro_home.sh`, output `repro_home.out`, plus the `h.*.out` logs ("Cloning into …") |
| V6 | `warnSilentRatchet` opens the SQLite store unconditionally when a prior lock exists, and `sqliteopen` creates the parent dir | `pkg_commands.go:529–531`, `:609–611`; `internal/messaging/store.go:84–90` (`statedir.Path("collaboration.db")`); `internal/sqliteopen/sqliteopen.go:81` `MkdirAll` |
| V7 | `pkg info/versions/stats/history` make requests under confinement; `provenance/search/install/docs/cascade status` do not; the unconfined control does | Live: `scratchpad/dd3/repro_http.sh`, output `repro_http.out` (loopback recorder on 127.0.0.1:18723) |
| V8 | `publish` upload, `unpublish`, `pkg key` and the cascade Pub/Sub have no guard | Code read: `pkg_publish.go:239–286`, `:604–617`; `pkg_unpublish.go:85–92`; `pkg_keys.go:141–150`. Not executed (it would need credentials); no `RefuseIfConfined` or `Confined` in these files (V2 grep) |
| V9 | `WriteShim`/`RemoveShim`/`DefaultBinDir` write `~/.ailang/bin` unguarded; `notify-upgrade` and `publish` open the messaging store unguarded | Code read: `bin.go:138–139,170–184,314–326`; `pkg_msg.go:166,266–268`; `pkg_publish.go:155,315`. The live `notify-upgrade` probe failed message validation *before* the store open (`repro_home.out`), so it is code-read only |
| V10 | `lock` is reachable through policy-tool via `cli_allow`; its ledger is `ailang.lock` only; `check/test/iface/tree/pkg-docs` are default | `cli_ops.go:71`, `:79–82`, `:107` |
| V11 | Bare top-level `lock/install/add/docs/…` run the same functions as `pkg <verb>` | `commands_groups.go:264–279` (`pkgLegacyTopLevel` → the same `Run` from `pkgSubcommands()`) |
| V12 | There is no `pkg update` verb | `pkgSubcommands()` (`commands_pkg.go:21–121`) lists 19 verbs, none named `update`; `grep '"update"' commands_pkg.go commands_groups.go` is empty |
| V13 | No `authority.go`/`egress*` or `HTTPClient`/`pkg.Git` chokepoint exists today; no git argv hardening exists | `ls internal/pkg/authority* internal/pkg/egress*` gives no such file; `grep -rn "protocol.allow\|GIT_TERMINAL_PROMPT" internal cmd` is empty |
| V14 | A leading-dash git URL is **not** exploitable as an option injection at git 2.54 with the current argv | Live: `git clone "--upload-pack=touch …/PWNED" <dir>` and the `--branch` variant both fail with "repository … does not exist", and no `PWNED` file appeared (`scratchpad/dd3/gitprobe`). So AC10 is hardening, not a fix |
| V15 | Git `content_hash` is never verified at check time | `lockfile.go:158–176` (path only after `validateRegistryContentHashes`); `registry_hash.go:19–23` (`if p.Source != "registry" { continue }`) |
| V16 | The git cache is keyed by URL only, and the loader ignores `GitRev` | `gitcache.go:76–79` (`sha256(gitURL)[:16]`); `loader.go:148–170` reads `cache.CacheDir(locked.GitURL)` with no rev check |
| V17 | The #1704 Non-Goal premise "git is read-only in the confined paths" was unlogged and false | `m-pkg-registry-confinement.md` Non-Goals ("Not touching git-source dependencies … read-only in the confined paths today"); its Verification Log V1–V21 has no git row; V3 refutes it |
| V18 | #1704's tests enumerate functions, not entry points | `internal/pkg/confinement_test.go:23–27` (three fetch closures); `cmd/ailang/pkg_docs_confinement_test.go` covers `pkg-docs` and the unconfined docs/install/lock only |
| V19 | `forbidigo` is configured repo-wide by pattern, with path exclusions | `.golangci.yml:19–34`, `:58–70` |
| V20 | A binary-level containment test harness exists to reuse | `cmd/ailang/main_test.go:365` `buildAilang`; `cmd/ailang/run_policy_containment_test.go:276–300` (confined git e2e, `GIT_CONFIG_GLOBAL=/dev/null`) |
| V21 | `design_quorum` is reachable under confinement and uses the network legitimately (which is why Alternative C is rejected) | `cli_ops.go:108` (`design_quorum` schema with `max-cost-usd`) |
| V22 | No new language-surface claim; no diagnostic code allocated | Go CLI and library behaviour only. Refusals are command-level errors, not `MOD/PAR/TC/EFF` codes |
| V23 | The originating registry-confinement doc was never moved to `implemented/` | `find design_docs -iname "*confine*"` finds only `planned/v0_52_6/m-pkg-registry-confinement{,-sprint-plan}.md`; header status "Implementation complete; awaiting independent sprint evaluation" |

**Quorum:** trigger 1 fires (Design Freeze items D2–D4). `ailang design-quorum` was **skipped at the requester's explicit direction** for this authoring pass. Run it before sprint planning (`--max-cost-usd 0.30`).

## Future Work

- An OS-level sandbox for the policy-tool CLI child (Alternative D), which would make every authority enforced by the kernel.
- A rev-keyed git cache (`<urlhash>/<rev>`, via `git worktree` or `git archive`), so T2 is prevented rather than detected.
- Read confinement of lock-supplied paths (T3).
- `ailang install --package-root` to populate an operator root from a lock (registry and git), carried over from #1704 Future Work.
- Extend the `Command.Authority` column and the matrix to every top-level command.

---

**Document created**: 2026-10-26
**Last updated**: 2026-10-26
