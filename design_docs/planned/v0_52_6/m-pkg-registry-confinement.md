# M-PKG-REGISTRY-CONFINEMENT: a policy-confined agent cannot fetch from the registry, write the registry cache, or have its planted copy trusted — and hosts get a read-only package root

Refs #1607 #1608

**Status**: Implementation complete; awaiting independent sprint evaluation
**Target**: v0.52.6
**Priority**: P0 (security/soundness — #1607; #1608 is P2 on its own but is the same defect class and ships in the same fix)
**Estimated**: 3 days
**Dependencies**: None. Same class as #1552 (examples corpus confinement, fixed in `1099e49f6`) and #1554 (the policy-tool child's cache redirect); this doc does not modify those, it extends the same principle to the package registry.
**Issues**: [#1607](https://github.com/sunholo-data/ailang/issues/1607) (P0, area:pkg), [#1608](https://github.com/sunholo-data/ailang/issues/1608) (P2, area:pkg). Triage 2026-10-08 verified the defect live on origin/dev `658ff76a3`. Premises below were re-verified by code reads at `1dfd5615` (see Verification Log).

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | The lock's `content_hash` is finally checked for registry packages, so a resolved registry dependency is pinned to content, not to "whatever is in a writable cache today" (V13). |
| A2: Replayability | 0 | No trace or replay surface change. |
| A3: Effect Legibility | +1 | Registry fetch and cache extraction become *refusals with named reasons* under confinement, instead of hidden network egress + persistent write performed by a "read-only" op (V1–V3). |
| A4: Explicit Authority | +1 | The core of the fix: network egress and writes to `$HOME` are operator authority, removed from the confined agent's default-reachable surface (V1, V5). |
| A5: Bounded Verification | 0 | No change to check cost or scope (the hash recomputation is over already-present files). |
| A6: Safe Concurrency | 0 | No concurrency change. |
| A7: Machines First | +1 | The refusal messages are machine-readable named errors a confined agent can act on (ask the operator to provision), rather than a silent fetch the agent cannot distinguish from local state. |
| A8: Minimal Syntax | 0 | No language syntax; one new host-set env var and Go-internal gates. |
| A9: Cost Visibility | 0 | No resource-cost surface change. |
| A10: Composability | 0 | No effect/record composition change. |
| A11: Structured Failure | +1 | Every new refusal is a named error naming the operator remedy (provision `AILANG_PACKAGE_ROOT` or pre-warm the cache), per the no-silent-fallbacks principle. |
| A12: System Boundary | +1 | The boundary between operator-provisioned state (read-only package root) and agent-writable state (nothing, for registry packages) becomes explicit. |

**Net Score: +5** → **Decision: Move forward.** No −1 on A1/A3/A4/A7.

### Hard Violation Check

- [x] A1: no implicit nondeterminism introduced (removal of an implicit fetch *increases* determinism)
- [x] A3: no hidden side effects added; two removed
- [x] A4: ambient authority (default-allowed network + HOME write) removed, not granted
- [x] A7: no human-convenience optimization over machine analysis

## Problem Statement

Two defects, one class: **the package registry surface was never routed through the confinement model that #1552 (examples corpus) and #1554 (the policy-tool child's caches) already established.**

### #1607 — a policy-confined agent reaches the network and writes a persistent, later-trusted cache

`ailang policy-tool` admits `pkg_docs` in its **default read-only `cli` set** (`defaultCLIAllow`, `internal/policytool/cli_ops.go:79–82` — V1). The op runs `ailang pkg-docs` as a **separate, unconfined child process** (`cli_ops.go:156`, V2). When the requested package is not cached, `cmd/ailang/pkg_docs.go` calls `pkg.NewRegistryClient()`, `FetchIndex()` (line 73), `FetchPackage()` (line 81), then `os.MkdirAll` + `pkg.ExtractTarball` into the cache (lines 87–88, V3) — writing `$HOME/.ailang/cache/registry/<vendor>/<name>/<version>/`.

So a policy-confined agent (empty `net_allow`, fs sandbox, deny-write) gets, through a default-admitted "read-only" op:

1. **Network egress** to the registry (30 s-timeout HTTP client, `internal/pkg/registry.go:35`).
2. **A persistent write outside the sandbox** — `RegistryCacheDir()` derives from `os.UserHomeDir()` only (`registry.go:207–211`, V4). Notably, #1554 already redirects the child's *compile and prompt* caches to a private temp dir via `AILANG_CACHE_DIR` (`cli_ops.go:405`, V6) — the **registry cache was missed**, and `AILANG_CACHE_DIR` does not cover it by definition (`internal/config/paths.go`, `EnvCacheDir` row, V7).
3. **Cache poisoning**: a later `check`/`ai-check` (default-admitted, V1) resolves a package's imports against that agent-fetched copy — the compile pipeline loads registry dependencies from `CachedPackagePath` (`internal/pkg/loader.go:173` via `internal/pipeline/package_resolver.go`/`pipeline_module_phases.go:161`, V8), and `ValidateContentHashesFrom` **skips every non-`path` dependency** (`internal/pkg/lockfile.go:160–161`, V13) — the lock's `content_hash` for registry packages is never verified at check time. The agent's fetched copy is trusted by construction.

Live-reproduced by the filer (issue body: empty `HOME`, restricted policy, `{"op":"pkg_docs","module":"sunholo/oauth"}` → `<HOME>/.ailang/cache/registry/sunholo/oauth/0.1.0/` appears) and again by triage 2026-10-08 on origin/dev `658ff76a3`.

The same leak exists one op over: `lock` is not in the default set but **is** reachable when a policy names it in `cli_allow` (`cliSchemas` entry `cli_ops.go:107`), and its declared write ledger says only `ailang.lock` (`lockWrites`, `cli_ops.go:71`) — while `ailang lock` auto-downloads uncached registry dependencies into the HOME cache (`internal/pkg/resolver.go:262–283`, V9) and fetches the registry index to resolve transitive path-deps inside registry packages (`resolver.go:113`, V10). The declared-writes ledger is currently false for this path.

### #1608 — no way to point resolution at a read-only, operator-provisioned package root

Registry packages resolve **only** under `os.UserHomeDir()/.ailang/cache/registry` (`registry.go:207–217`), and merely *computing* that path `MkdirAll`s it — so even a pure read (any `check` on a package with registry deps, V8) is a HOME write (V4). `AILANG_CACHE_DIR` covers the compile and prompt caches only (V7). The operator policy has no package key (`internal/policy/policy.go` `Policy` struct, V11). No `AILANG_PACKAGE_ROOT` (or equivalent) exists anywhere in the repo (V12). A host that confines tools has to fake a `HOME` containing a symlink to make a read-only snapshot visible — the exact friction AILANG World row 141 hit (`w-workspace-project-layouts.md`, ailang-world repo).

**Impact:** every `ailang_only`-lane agent and every policy-tool deployment (AILANG World hosts, the eval harness's confined lanes) is affected. This is the supply-chain boundary of the language: the thing `check` type-checks your program against is currently agent-writable in the confined configuration.

## Goals

**Primary Goal:** under `AILANG_AGENT_POLICY`, the registry surface becomes read-only-from-operator-state — no network fetch, no cache write, no trust in agent-placed cache content — and hosts get an explicit, read-only, operator-provisioned package root.

**Success Metrics:**
- The #1607 repro inverts: confined `pkg_docs` for an uncached package returns a named refusal and creates **zero** entries under `$HOME/.ailang/` (acceptance test A1).
- Confined `lock` over a manifest with an uncached registry dependency fails loudly with the same named refusal; the declared-writes ledger (`ailang.lock` only) becomes true (A2).
- A confined `check` over a package whose registry deps live in a chmod-0555 `AILANG_PACKAGE_ROOT` succeeds and writes nothing anywhere (A3).
- A modified `.ail` file inside a cached registry package is detected at check time via the lock's `content_hash` (A4).
- Unconfined behaviour (`ailang install`, `pkg-docs`, `lock` without a policy) is byte-for-byte unchanged (A5).

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| D1: Gate at the choke points (`RegistryClient.FetchIndex`/`FetchPackage`, `ExtractTarball`-into-cache, `EnsureRegistryCacheDir`), not per call site | Per-call-site gating is what produced the whack-a-mole (pkg_docs fixed, lock missed); a choke point covers future callers | agent | design | med |
| D2: The confined signal is `AILANG_AGENT_POLICY` env presence (`config.AgentPolicy() != ""`), same as #1552 | One signal, already injected into every policy-tool child (`childEnv`, `cli_ops.go:377–385`); inventing a second signal splits the confinement model | agent | design | low |
| D3: When `AILANG_PACKAGE_ROOT` is set, registry packages resolve **only** from it (no HOME-cache fallback) | A fallback to a writable HOME cache under confinement silently reintroduces the poisoning path; fail-loud is the house rule | human | design | med |
| D4: Registry-package `content_hash` verification added to `ValidateContentHashesFrom`; a mismatch is a **hard error** in `check` (D5 ruled hard-fail now) | Path deps already behave this way; check currently only warns even for them (`package_resolver.go:67–69`) | agent | design | low |
| D5: Whether a registry `content_hash` mismatch fails `ailang check` (exit ≠ 0) now or after a deprecation window | Failing now can break working checkouts whose cache legitimately predates a re-lock; failing never keeps the poisoning detector soft | human | design | med |
| D6: `RegistryCacheDir()` stops `MkdirAll`-ing; an explicit `EnsureRegistryCacheDir()` is called only on genuine write paths | Makes every read path (check/pkg-docs/lock-read) write-free by construction; touches every registry-cache consumer | agent | design | med |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] D3 ratified: package-root-only resolution, no HOME fallback when `AILANG_PACKAGE_ROOT` is set — **Ruled 2026-10-08 by Mark: yes, package-root-only resolution; no HOME fallback when `AILANG_PACKAGE_ROOT` is set.**
- [x] D5 ratified: mismatch severity at `check` (loud-warning-now vs hard-fail-now) — **Ruled 2026-10-08 by Mark: hard-fail now: a registry `content_hash` mismatch makes `ailang check` exit non-zero in this release (no deprecation window).**

## Solution Design

### Overview

Three coordinated changes, all inside the existing confinement doctrine:

1. **A confinement gate at the registry choke points** (#1607): when `AILANG_AGENT_POLICY` is set, the network entry points (`FetchIndex`, `FetchPackage`) and the cache-write helpers refuse with a named error that names the operator remedies. `pkg-docs` and `lock` under confinement become cache/package-root reads that fail loudly on miss.
2. **A read-only, operator-provisioned package root** (#1608): `AILANG_PACKAGE_ROOT` — host-set like `AILANG_EXAMPLES`, never agent-set — read *before* the HOME cache by the `PackageLoader` and `pkg-docs`, never written, never created. `RegistryCacheDir` loses its `MkdirAll` (moved to an explicit ensure called only by write paths).
3. **Content-hash verification of registry packages** (poisoning detector): `ValidateContentHashesFrom` extends to `source == "registry"` entries, recomputing the hash over the resolved directory (package root first, then cache) and comparing against the lock's `content_hash`.

### Architecture

**Components:**

1. **`internal/pkg/confinement.go` (new)** — `Confined() bool` (`config.AgentPolicy() != ""`) and `RefuseIfConfined(what string) error` returning the canonical refusal text: `confined by AILANG_AGENT_POLICY: <what> is operator authority — provision AILANG_PACKAGE_ROOT or run 'ailang install' outside the sandbox`. Single source for the message so tests can pin it.
2. **Gates in `internal/pkg/registry.go`** — `FetchIndex` and `FetchPackage` call `RefuseIfConfined("fetching from the registry")` first (D1). `ExtractTarball` is *not* gated itself (it is a pure byte→dir helper also used by operator tooling on explicit paths); instead the registry-cache write path is: `EnsureRegistryCacheDir()` (new, does the `MkdirAll`) gated by `RefuseIfConfined("writing the registry cache")`, and `RegistryCacheDir()` becomes a pure computation (D6). All current write-side callers (`pkg_install.go:121–130`, `pkg_docs.go:86–88`, `resolver.go:274–277`) move to `EnsureRegistryCacheDir`.
3. **`AILANG_PACKAGE_ROOT`** — `EnvPackageRoot` registered in `internal/config/paths.go` `pathVars` (docs row: "Read-only, operator-provisioned root of registry packages, laid out `<vendor>/<name>/<version>`; read before the HOME cache by the PackageLoader and `pkg-docs`; never written or created. Set by the host, never the agent — like AILANG_EXAMPLES.") plus getter `config.PackageRoot()`. Resolution helper in `internal/pkg` — `PackageDir(name, version)`: if `config.PackageRoot() != ""`, return `<root>/<vendor>/<name>/<version>` (existing-or-not; the caller errors loudly on miss, no fallback — D3); else the `CachedPackagePath` value. `PackageLoader`'s registry case (`loader.go:173`) and `pkg_docs`'s cache read switch to it.
4. **Hash verification** — `ValidateContentHashesFrom` (`lockfile.go:158–176`) grows a registry branch: `source == "registry"` → resolve the dir with `PackageDir` → `ContentHash(dir)` → compare with `p.ContentHash`. Mismatch produces the same drift-message shape as path deps, naming the package and both hashes (D4/D5 govern severity at the `check` caller).
5. **`cmd/ailang/pkg_docs.go`** — reorder: package-root read → HOME-cache read (git-cache read stays as-is, it is read-only) → **if confined: named refusal** → else current fetch path, with `EnsureRegistryCacheDir` on the write.

**Why the choke point and not `cli_ops.go`:** the policy schema table (`cliSchemas`) already refuses unknown commands, but it cannot see *what a command does* — `pkg_docs` looks read-only and `lock` looks like one file write. The confinement model's invariant is environmental (`AILANG_AGENT_POLICY` reaches every child, V2), so the library that owns the network and the cache enforces it. This also covers the coordinator-side and tooling callers uniformly: they run unconfined (no policy env) and are unaffected (V14).

### Conflict Surface (who else touches these code paths)

This is not a parser/typechecker change (none of `internal/parser`, `internal/lexer`, `internal/ast`, `internal/types`, `internal/elaborate`, `internal/iface`, `internal/codegen`, `internal/eval`, `internal/vm`, `internal/effects`, `cmd/ailang/exec.go` is modified). The shared machinery touched is the registry cache, and its full consumer set was enumerated (D1's justification):

| Caller | File:line | Network | HOME-cache write | Confined-reachable? | Effect of gate |
|---|---|---|---|---|---|
| `pkg-docs` | `cmd/ailang/pkg_docs.go:73–88` | yes | yes | **yes — default allow** (V1) | refusal on miss (fixes #1607) |
| `ailang lock` (resolver) | `internal/pkg/resolver.go:113, 262–283` | yes | yes | yes — opt-in `cli_allow` (V9) | refusal on uncached dep |
| `check`/`ai-check`/`test` (loader read) | `internal/pkg/loader.go:173` | no | yes — `MkdirAll` in `RegistryCacheDir` (V4) | yes — default allow (V8) | read becomes write-free; package root read; hash verified |
| `ailang install` | `cmd/ailang/pkg_install.go:109–130` | yes | yes | no — not in `cliSchemas` (V15) | none (unconfined) |
| `pkg publish` / `pkg quality` / `pkg cascade status` | `pkg_publish.go:376`, `pkg_quality.go:141`, `pkg_cascade_status.go:77,170` | yes | no | no — not in `cliSchemas` (V15) | none (unconfined) |
| coordinator package agents | `internal/coordinator/package_agents.go:156` | yes | no | no — operator daemon, no policy env (V14) | none |
| `registry-validator` | `cmd/registry-validator/main.go:196` | yes | temp dir | no — operator tooling (V15) | none |

**Programs/commands that MUST still work post-change:** every row above in the unconfined case (acceptance A5), plus confined `pkg-docs`/`check`/`ai-check`/`test`/`iface`/`tree` over packages whose registry deps are pre-provisioned (A1/A3). **Deliberately changed:** confined `pkg-docs` on an uncached package (fetch → refusal), confined `lock` needing a download (silent download → refusal), and any process that relied on `RegistryCacheDir()` creating the directory as a side effect (now explicit).

### Implementation Plan

**Phase 1: the gate (#1607)** (~1 day)
- [x] `internal/pkg/confinement.go` — `Confined()`, `RefuseIfConfined`
- [x] Gates in `FetchIndex`/`FetchPackage`; `RegistryCacheDir` purity split + `EnsureRegistryCacheDir`; migrate the three write callers
- [x] `cmd/ailang/pkg_docs.go` confined branch
- [x] `internal/pkg/resolver.go`: registry-download branch and transitive-index branch refuse when confined
- [x] Acceptance tests A1, A2, A5 (below)

**Phase 2: the read-only root (#1608)** (~1 day)
- [x] `EnvPackageRoot` + `config.PackageRoot()` + `pathVars` row; regenerate/verify `docs/docs/reference/env-vars.md`
- [x] `pkg.PackageDir(name, version)` resolution helper; wire into `loader.go:173` and `pkg_docs`
- [x] Acceptance tests A3; chmod-0555 read-only fixture

**Phase 3: the hash check (poisoning detector)** (~0.5 day)
- [x] `ValidateContentHashesFrom` registry branch
- [x] Acceptance test A4; D5 ruling applied at the `check` caller
- [x] Docs: `docs/docs/reference/std-package.md` (resolution order + confinement note), `docs/docs/reference/cli.md` unchanged surface, SECURITY.md note under the confinement section

**Phase 4: sweep + gates** (~0.5 day)
- [x] Focused Go tests, `make test-core`, `make fmt`, `make lint`, `make check-boundaries`, `make check-file-sizes` (full suite runs in CI; config→pkg layering is unchanged: `internal/pkg` already imports `internal/config`, V16)
- [x] Re-run the #1607 repro from the issue body end-to-end against the built binary

### Files to Modify/Create

**New files:**
- `internal/pkg/confinement.go` — the gate helpers, ~40 LOC
- `cmd/ailang/pkg_docs_confinement_test.go` — A1/A5, ~120 LOC

**Modified files:**
- `internal/config/paths.go` — `EnvPackageRoot` + getter + `pathVars` row, ~10 LOC
- `internal/pkg/registry.go` — gates, `RegistryCacheDir` purity split, `EnsureRegistryCacheDir`, `PackageDir`, ~50 LOC
- `internal/pkg/loader.go` — registry case via `PackageDir`, ~10 LOC
- `internal/pkg/resolver.go` — confined refusals on the two fetch branches, `EnsureRegistryCacheDir`, ~15 LOC
- `internal/pkg/lockfile.go` — registry branch in `ValidateContentHashesFrom`, ~25 LOC
- `cmd/ailang/pkg_docs.go` — confined branch + package-root read, ~30 LOC
- `cmd/ailang/pkg_install.go` — `EnsureRegistryCacheDir`, ~3 LOC
- `docs/docs/reference/env-vars.md`, `docs/docs/reference/std-package.md`, `SECURITY.md` — documentation, ~40 LOC
- `internal/pkg/lockfile_test.go`, `internal/pkg/loader_test.go`, `internal/pkg/resolver_test.go` — A2/A3/A4 fixtures, ~150 LOC

## Examples

### Example 1: #1607 repro, before and after

**Before** (v0.52.5, confined child, empty `HOME`):
```
$ echo '{"op":"pkg_docs","module":"sunholo/oauth"}' | ailang policy-tool
{"ok":true,"stdout":"<AGENT.md of sunholo/oauth>…"}
$ ls $HOME/.ailang/cache/registry/sunholo/oauth/
0.1.0    ← written by the confined agent
```

**After**:
```
$ echo '{"op":"pkg_docs","module":"sunholo/oauth"}' | ailang policy-tool
{"ok":false,"refused":"pkg-docs: package sunholo/oauth is not provisioned and a
 confined agent cannot fetch it: fetching from the registry is operator authority
 — the host can set AILANG_PACKAGE_ROOT or run 'ailang install sunholo/oauth'
 outside the sandbox"}
$ ls $HOME/.ailang
ls: cannot access '$HOME/.ailang': No such file or directory
```

### Example 2: operator-provisioned read-only root (#1608)

```
# Host (unconfined), once:
$ ailang install sunholo/oauth@0.1.0
$ cp -R ~/.ailang/cache/registry /srv/ailang/packages && chmod -R a-w /srv/ailang/packages

# Launcher for the confined agent:
$ AILANG_AGENT_POLICY=policy.toml AILANG_PACKAGE_ROOT=/srv/ailang/packages \
  ailang policy-tool <<< '{"op":"check","path":"pkg/main.ail"}'
{"ok":true,…}    # resolved against /srv/ailang/packages, read-only, nothing written
```

### Example 3: poisoning detected (A4)

```
$ echo 'export func steal() = 1  -- planted' >> \
    ~/.ailang/cache/registry/sunholo/oauth/0.1.0/oauth.ail
$ ailang check pkg/main.ail
Error: dependency sunholo/oauth content changed (locked: 9ab34…, current: f00d1…)
Run 'ailang lock' to update    ← the lock's content_hash no longer matches the cache copy
```

## Success Criteria

- [x] **A1 (#1607)**: with `AILANG_AGENT_POLICY` set, empty `HOME`, and an uncached package, `pkg_docs` returns the named refusal and `$HOME/.ailang` is not created (test: temp HOME + `os.Stat` absence)
- [x] **A2**: confined `lock` over a manifest with an uncached registry dep refuses; no cache write; with all deps pre-provisioned it still writes `ailang.lock` (the declared ledger holds)
- [x] **A3 (#1608)**: confined `check` resolves registry deps from a chmod-0555 `AILANG_PACKAGE_ROOT`, succeeds, and writes nothing (test: read-only fixture + post-run tree compare)
- [x] **A4**: a modified `.ail` in a cached registry package yields a content-hash mismatch at check time
- [x] **A5**: unconfined `pkg-docs` (uncached), `install`, and `lock` behave exactly as on v0.52.5
- [x] Focused registry checks and core tests run; SQLite/CGO environment failures recorded separately. `make fmt`/`make lint`, `make check-boundaries`, and `make check-file-sizes` checked; full `make test` delegated to CI per executor constraint
- [x] Documentation updated: `env-vars.md` (new var), `std-package.md` (resolution order + confinement), `SECURITY.md`
- [x] The issue-body repro re-run by the executor against the built binary, recorded in the sprint report

## Testing Strategy

**Unit tests:**
- `RefuseIfConfined` message stability (pinned text)
- `RegistryCacheDir` no longer creates; `EnsureRegistryCacheDir` creates; both idempotent behaviours
- `PackageDir` resolution order (root set → root-only; unset → HOME cache)
- `ValidateContentHashesFrom`: registry entry match, mismatch, missing dir, lock with empty `content_hash` (already rejected by `Validate`, V17)

**Integration tests:**
- `cmd/ailang/pkg_docs_confinement_test.go`: A1/A5 using `t.Setenv(config.EnvAgentPolicy, …)` and a temp `HOME` (the #1552 test shape, `examples_corpus_test.go:88–100`)
- policy-tool level: dispatch `pkg_docs` through `Host` with an injected `run` that records the child env/argv (the existing test seam, `policytool.go:115–119`)
- loader test: registry dep behind a 0555 root

**Manual testing:**
- The issue-body repro end-to-end (empty `HOME`, restricted policy, uncached package) on a real binary
- AILANG World host shape: `AILANG_PACKAGE_ROOT` symlink-free real dir, confined agent checks a package that imports a registry package

## Deferred Decisions

- Whether `policy-tool`'s `Summary` should surface the presence/absence of `AILANG_PACKAGE_ROOT` to the model (helps the agent phrase the ask; costs a summary field) — agent may resolve.
- Whether the confined refusal should distinguish "package root set but package missing" from "no package root set" in the message tail — agent may resolve (recommend: yes, one extra clause).
- Whether `ailang install` should learn `--package-root` to *populate* the root (operator convenience) — out of scope here; the operator `cp` in Example 2 is the v0.52.6 story.

## Non-Goals

- **Not routing the registry cache through the policy-tool child's private temp `AILANG_CACHE_DIR`** (#1554's shape): registry packages are shared, operator-provisioned state; a per-child temp cache would force a network fetch per child — exactly what confinement forbids — and would destroy the poisoning detector. The explicit read-only root is the honest shape.
- **Not adding a policy-file key for the package root** (`package_root = "…"` in agent-policy.toml): the policy already travels as a path; the root is host state like `AILANG_EXAMPLES`, and `policy.Resolved` would have to be threaded into `internal/pkg` to reach the child's loader. The env var reaches every child already (V2). Revisit if a policy ever needs to *pin* a root digest.
- **Not changing `ExtractTarball` itself** — pure helper, gated at its cache-writing callers.
- **Not verifying the *tarball* hash against the registry index** (only the extracted content against the lock) — the lock is the trust anchor the compiler already ships; registry-index signing is a separate, larger design.
- **Not touching git-source dependencies** — separate cache (`~/.ailang/cache/git`), read-only in the confined paths today; if a write path emerges there it gets its own issue.

## Timeline

**Week 1** (3 days):
- Days 1–2: Phases 1–2 (gate + read-only root) with A1/A2/A3/A5 green
- Day 3: Phase 3–4 (hash check, docs, full gates, live repro re-run)

**Total: ~3 days** (2× the naive 1.5-day estimate, per house convention)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| An operator shell exports `AILANG_AGENT_POLICY` while doing unconfined package work; their `install`/`lock` now refuse | Med | The refusal names the cause and remedy ("unset AILANG_AGENT_POLICY"); same accepted tradeoff as #1552's corpus skip. Loud, not silent. |
| A caller relied on `RegistryCacheDir()` creating the directory (behaviour change, D6) | Med | Grep of all callers shows the write callers migrate to `EnsureRegistryCacheDir` (Conflict Surface table); read callers *gain* correctness. |
| Registry `content_hash` in old locks was computed from a cache copy that legitimately differs across extraction environments | Low | `ContentHash` hashes `.ail` file contents only (`hasher.go:14–16`, V18) — deterministic from a given tarball; cross-extraction differences would already have broken `lock --check` drift. |
| Breaking AILANG World hosts that fake a `HOME` symlink today | Low | That mechanism keeps working (it is just a HOME cache); the new root is additive. Announce in the changelog entry. |
| The gate message text drifts and breaks a downstream matcher | Low | Message pinned in a unit test (single source in `confinement.go`). |

## Related Documents

**Implemented (may inform design):**
- (none found by SimHash; the #1552/#1554 fixes are recorded in code comments — `cmd/ailang/examples_corpus.go:47–56`, `internal/policytool/cli_ops.go:390–395` — and their issues)

**Planned (check for overlap):**
- `design_docs/planned/m-motoko-ailang-only-lane.md` — the confined lane this gate protects
- `design_docs/planned/HANDOVER-package-agent-autoprovision.md` — package-agent provisioning, operator side
- `design_docs/planned/v0_53_0/m-test-runner-compile-once.md` — `ailang test` compile path, shares the pipeline consumer (no overlap: it changes compilation count, not resolution authority)

## References

- [Design Axioms](/docs/references/axioms)
- Issues: [#1607](https://github.com/sunholo-data/ailang/issues/1607), [#1608](https://github.com/sunholo-data/ailang/issues/1608); prior art in class: #1552 (examples corpus confinement), #1554 (policy-tool child cache redirect)
- AILANG World row 141 — `design_docs/planned/w-workspace-project-layouts.md` (ailang-world repo), PR sunholo-data/ailang-world#224

## Verification Log

| # | Claim | Evidence |
|---|-------|----------|
| V1 | `pkg_docs` is in `defaultCLIAllow` (default read-only set) | Read `internal/policytool/cli_ops.go:79–82`; schema at `:99` (`{cmd: "pkg-docs", field: "Module", kind: kindModule, required: true, needsRoot: true}`) |
| V2 | The CLI child is a separate, unconfined process carrying `AILANG_AGENT_POLICY` | `cli_ops.go:156` ("The CLI child is a separate, UNCONFINED process"), `childEnv` `:377–385` (strips then re-adds `AILANG_AGENT_POLICY=<policyPath>`), `runAilang` `:396–407` (`exec.CommandContext`, env passed through) |
| V3 | `pkg-docs` fetches and extracts on cache miss | Read `cmd/ailang/pkg_docs.go`: `FetchIndex` `:73`, `FetchPackage` `:81`, `os.MkdirAll` `:87`, `ExtractTarball` `:88` |
| V4 | `RegistryCacheDir` is HOME-only and `MkdirAll`s | Read `internal/pkg/registry.go:207–217` |
| V5 | `AILANG_CACHE_DIR` covers compile + prompt caches only | `internal/config/paths.go` `EnvCacheDir` row ("Root of the compile cache (<dir>/compile) and the prompt cache"); no registry mention anywhere in `paths.go` |
| V6 | The policy-tool child's caches are redirected, registry missed | `cli_ops.go:390–395` comment + `:405` (`config.EnvCacheDir+"="+cacheDir` temp dir) |
| V7 | 30 s HTTP client for registry fetches | `registry.go:55` (`&http.Client{Timeout: 30 * time.Second}`) |
| V8 | `check`/`ai-check` resolve registry imports from the HOME cache | `internal/pipeline/pipeline_module_phases.go:161` → `tryLoadPackageResolver` → `NewPackageLoader` → `loader.go:173` (`CachedPackagePath(locked.Name, locked.Version)`) |
| V9 | `ailang lock` auto-downloads uncached registry deps | Read `internal/pkg/resolver.go:262–283` (stat manifest → `FetchPackage` → `MkdirAll` → `ExtractTarball`) |
| V10 | Lock resolution also fetches the index for transitive path-deps inside registry packages | `resolver.go:113` (`registryClient.FetchIndex()` in the `fromRegistry && dep.Path != ""` branch) |
| V11 | The operator policy has no package key | Read `internal/policy/policy.go` `Policy` struct (fields: allowed_caps, fs_sandbox, net_allow, net_allow_http, process_allow, cli_allow, budgets, timeout_ms, max_source_bytes, ai_provider, entry, security_mode, byte ceilings, fs_deny_write) — no package/registry key |
| V12 | No `AILANG_PACKAGE_ROOT`/`PackageRoot` exists anywhere | `grep -rn "AILANG_PACKAGE_ROOT\|PackageRoot" --include="*.go" --include="*.md" .` → only this doc (negative-existence sweep, 2026-10-08) |
| V13 | Registry packages' `content_hash` is never verified at check time | Read `internal/pkg/lockfile.go:160–161` (`if p.Source != "path" \|\| p.Path == "" { continue }`); the pipeline caller only prints a warning (`internal/pipeline/package_resolver.go:67–69`); issue #1608's "accepts a placeholder hash at check time" confirmed |
| V14 | Coordinator/tooling registry callers run unconfined (no `AILANG_AGENT_POLICY` in their env) | `grep -rn "AILANG_AGENT_POLICY"` shows it set only by the executor (`internal/config/job.go:45`, `internal/executor/envbuild*`), never by the coordinator daemon or `cmd/registry-validator` |
| V15 | `install`/`publish`/`quality`/`cascade` are unreachable through policy-tool | Read the full `cliSchemas` table `cli_ops.go:87–108` — those commands have no schema, and `cliAllowed` requires one (`cli_ops.go:118–123`) |
| V16 | `internal/pkg` already imports `internal/config` (layering permits the gate) | `registry.go` imports `"github.com/sunholo-data/ailang/internal/config"` |
| V17 | A lock entry with empty `content_hash` is already rejected | `lockfile.go:126–129` (`Validate`: "package %s missing content_hash") |
| V18 | `ContentHash` is content-only, deterministic from the files | Read `internal/pkg/hasher.go:14–16` ("SHA256 hash over all .ail source files in dir") |
| V19 | The #1552 confined-signal precedent is `config.AgentPolicy() != ""` | Read `cmd/ailang/examples_corpus.go:87`; test shape at `examples_corpus_test.go:88–100` |
| V20 | Live reproduction of #1607 on current dev | Issue body (empty `HOME`, restricted policy, `pkg_docs` op → registry dir appears) and triage 2026-10-08 on origin/dev `658ff76a3`; code-read confirmed at `1dfd5615` (this checkout, shallow depth 1 — history citations #1552/#1554 are taken from the in-repo comments cited above, not from `git log`) |
| V21 | No new language-surface claims in this doc | It makes no "AILANG does/does not support X" statement — the changes are Go CLI/library behaviour; no `ailang check` of a language construct is implicated, and no new MOD/PAR/TC/EFF error code is allocated (refusals are command-level errors, not diagnostics; `grep` for the refusal strings shows them unallocated) |

## Future Work

- Registry-index/tarball signing (trust anchor beyond the lock) — only meaningful once more than one registry exists.
- `ailang install --package-root` to populate/maintain an operator root with version pinning from a lock file.
- Surfacing `AILANG_PACKAGE_ROOT` state in the policy-tool `Summary` (see Deferred Decisions).
- Git-source dependency confinement audit (out of scope here; read-only today).

---

**Document created**: 2026-10-08
**Last updated**: 2026-10-08

## Implementation evidence

See [sprint report](m-pkg-registry-confinement-sprint-report.md) for A1–A5, validation results, environment limitations, and the implementation PR body. No full `make test` was run and no branch was pushed.
