# M-COHORT-MANIFEST-BUILD-PROVENANCE: Build identity for release evidence, cache identity, and eval banking

**Status**: Planned — round 4 came back **BLOCKED 2/2**; both round-4 objections plus a
**systemic sweep of every gate and acceptance assertion** (the four rounds' objections are one
class: a predicate weaker than the purpose it is cited for) have been **APPLIED in round 5
(2026-08-23, iteration 257)**; the doc now awaits the round-5 quorum (the controller's step).
**Quorum has NOT passed.** See "Quorum revision log — round 5". NOT blocked on a human, a lane,
or a clock.
**Target**: v0.34.0
**Priority**: P0 (High) — the clause-5 KPI's frozen release evidence currently records `"dev"/"dev"` provenance
**Estimated**: 3–4 days (M1: 1d · M2: 1d · M3: 0.5d · M4: 1d incl. docs)
**Dependencies**: None
**Planner-Lane**: opus-required (M2 changes the compiler-cache identity component; both wrong-key failure classes are silent)

> All measurements in this doc were made by its author in the pin worktree
> `/Users/voightkampff/.ailang-driver-pin/v1` at `origin/dev` =
> `96381331a3e0dd0682873553b79d91c7c8e27f5f`, go1.26.6 darwin/arm64, on 2026-08-23 —
> **except V17 and V22, measured first-party by the controller** in the same worktree at the same
> HEAD during quorum revision round 1, **V23**, measured first-party by the controller during
> quorum revision round 2, **V24**, measured first-party by the controller in round 3 at
> `db71d2a16` and **re-derived first-party in round 4** at the then-current
> `ad6d08050b5fabc42f8510780466295646000a05` (numbers in the row are the round-4 re-derivation,
> reproduced read-only by the round-4 reviser in the same session), **V25**, measured by the
> round-4 reviser at `ad6d08050`, and **V26–V28**, measured by the round-5 reviser at
> `ad6d08050` — all transcribed verbatim (see the Quorum revision log).
> Every claim about current behaviour carries a Verification Log row (V1–V28, bottom of doc).

## Related Documents

Duplicate/coverage gate: a grep sweep of `design_docs/planned/` and recent `implemented/` for
`buildvcs | vcs.revision | unstamped | build provenance | ldflags` found no doc covering
build-identity provenance (V18). The neural search was not run (needs rig ollama); the grep gate
was decisive. Nearest neighbours, each genuinely distinct:

- [m-eval-version-banking](../implemented/v0_26_1/m-eval-version-banking.md) (implemented v0.26.1)
  — created `--bank-by-version` and deliberately chose **per-RELEASE** granularity ("no churn on
  every dev commit"). This doc fixes its degenerate case — an unstamped build banks into one
  shared `dev/` bucket forever — while preserving the per-release cadence for stamped builds.
- [m-cost-per-success-kpi M4a](../implemented/v1_0_0/m-cost-per-success-kpi-m4a-sprint-plan.md)
  (implemented v1.0.0) — created `cohort_manifest.json` as the reproducibility artifact. This doc
  makes its `ailang_version`/`git_commit` fields real instead of `"dev"`.
- `internal/pipeline/cache_key.go` v1→v2→v3 history (M-PERF6, M-LAMBDA-OPEN-RECORD-PATTERN,
  M-XMOD-ALIAS-POLY) — bumped the FORMAT version to defend against same-version dev/worktree
  builds; this doc fixes the IDENTITY component those comments only mention in passing (V8).
- Decision **D-30** (open, human) — harness↔`ai-check` version *skew* via PATH-resolved child
  processes. Different defect, same neighbourhood: D-30 is about two binaries disagreeing; this
  doc is about one binary not knowing what it is. Referenced only; not designed here.

## Problem Statement

`internal/version.Version` and `internal/version.Commit` both default to the literal `"dev"`.
`Commit` has a runtime fallback (`init()` reads `vcs.revision` from `debug.ReadBuildInfo()`);
`Version` has none — only `-ldflags` (the Makefile) ever sets it (V7, V3).

**Go does not stamp VCS info when building from a linked git worktree, and `-buildvcs=true`
silently produces an unstamped binary instead of erroring** (V1, V4, V5). The pin worktree's
`.git` is a *file* (`gitdir: …/.git/worktrees/v1`), not a directory (V6), which is consistent
with the Go toolchain's repo-root detection not recognising linked worktrees. This is not a
detached-HEAD artifact: a branch-attached linked worktree is equally unstamped (V5).

This mission's own rules mandate building in worktrees (never the shared main tree) with a
scratch `go build` (never `make quick-install`, to protect `~/go/bin` for concurrent agents). So
**every binary this loop builds is unstamped by construction**, `version.Commit` silently
degrades to `"dev"`, and `version.Version` is `"dev"` even on stamped-VCS main-checkout builds
(V2, V3).

**Current state (measured):**
- The live v1.0 release-evidence artifact
  `design_docs/implemented/v1_0_0/m-cost-per-success-kpi-baseline-v1.0-cohort-manifest.json`
  records `ailang_version = "dev"`, `git_commit = "dev"` while its sibling fields (`frozen_at`,
  `chain_id`, `cohort_hash`) are populated (V15). Its stated purpose — a reviewer independently
  recomputing the cohort — is unmet.
- Every worktree-built `ailang` shares one module-cache compiler identity (`"dev"`), so the
  documented invalidation — "rebuilding ailang at a new commit invalidates cache" — never
  happens for loop-built binaries (V8).
- Every worktree-built eval run with `--bank-by-version` banks into one shared `dev/` bucket:
  `releaseTag("dev")` returns `"dev"` unchanged (V9, V13). This repo has shipped a fix for
  cross-version result pooling before (m-eval-version-banking); unstamped builds silently
  re-create it.

**Impact:** release evidence (clause-5 KPI), compiler-cache correctness, and eval data integrity
— all three are CLAUDE.md §2 surfaces where a plausible fallback value is forbidden.

## The full consumer surface (systemic sweep, CLAUDE.md §3)

Enumerated by **import path**, not symbol grep — a symbol grep for `version\.Version` misses two
call sites that alias the import as `versionpkg` (V10). Seven non-test files import
`internal/version`:

| # | Site | Consumes | Role | Classification |
|---|------|----------|------|----------------|
| 1 | `cmd/ailang/eval_suite_manifest.go:212-213` | `Version`+`Commit` | frozen cohort manifest (release evidence) | **decision-bearing → M4 (refuse at freeze)** |
| 2 | `internal/pipeline/pipeline_module.go:276` | `Commit` | module-cache compiler identity | **decision-bearing → M2 (bypass on ambiguous identity)** |
| 3 | `cmd/ailang/eval_suite.go:191` | `Version` | `--bank-by-version` output bucket | **decision-bearing → M3 (explicit unstamped marker)** |
| 4 | `cmd/ailang/main.go:27-28` | both (aliases) | `--version` display | display-grade; unchanged (Non-Goals) |
| 5 | `cmd/ailang/prompt.go:94` (`versionpkg`) | `Version` | `LoadPromptFresh` callerVersion → MCP `forVersion` + prompt cache key | service-grade; degrades to embedded fallback by design; unchanged, residual noted |
| 6 | `cmd/ailang/mcp_status.go:84` (`versionpkg`) | `Version` | same path as #5 | unchanged |
| 7 | `cmd/ailang-microrag-mcp/main.go:54,63` | `Version` | MCP server metadata + startup log | display-grade; unchanged |

Consumers 1–3 make decisions (what evidence says, what gets recomputed, where data pools) from
an unverified value; 4–7 only label output. The fix is one shared accessor plus three explicit,
reviewable call-site decisions — not three independent patches and not a blanket startup refusal
(`go run`/`go test` builds are legitimately unstamped, and the whole test suite runs through
them).

## Goals

**Primary Goal:** No decision-bearing consumer of build identity ever acts on `"dev"` without
either refusing loudly (evidence), declining the action entirely rather than guessing (cache), or
substituting a *deterministic, still-differentiating, explicitly-marked* identity (banking).

**Success Metrics:**
- A `--baseline` cohort freeze from an unstamped OR dirty-stamped binary (`-dirty` on either
  field, V24) **exits non-zero before any spend**, with remediation text; a **clean**-stamped
  freeze records real `ailang_version`/`git_commit`. The freeze gate is
  `VersionKnown() && CommitClean()` (round 5): a `-dirty` suffix is *visible* evidence but not
  *reproducible* evidence — the manifest records no source diff and no compiler-content
  identity (V27), so a dirty freeze can never satisfy the manifest's own recompute-the-cohort
  purpose.
- Unstamped and stamped-dirty builds (a `-dirty` suffix on **either** `Commit` or `Version` —
  the two stamping paths put it on different fields, V24) **decline module caching entirely** —
  no Lookup, no Store, one stderr warning — rather than being assigned a substitute compiler
  identity; a clean-stamped build's module-cache key stays byte-identical to today's
  `version.Commit`-keyed behaviour.
  There is no cache identity for two ambiguous builds to collide on or differ by, by construction
  (V22/V23: the rejected content-hash alternative cost ~215 ms/process against a measured 0–12 ms
  cache benefit).
- An unstamped `--bank-by-version` run banks into `unstamped-<id8>/`, never into a shared `dev/`
  — via `AttributionID()`, the metadata-derived id retained solely for this bucketing use and
  never consulted by the cache.
- `internal/version` goes from **zero test files** (V11) to a tested, load-bearing package.
- Clean-stamped (`make`-built at a clean commit) binaries: zero behaviour change on all three
  surfaces. A **dirty**-stamped build changes on exactly two surfaces — the module cache, which
  now bypasses instead of keying on an identity that aliases different compiler bytes: the
  shared `abc1234-dirty` string on the ReadBuildInfo path (Objection 1, revision round 1), or
  the **plain commit SHA** on the `make`/ldflags path, where `-dirty` rides on `Version` only
  (round 3, V24) — Failure Class 1 both ways; and the cohort freeze, which now refuses dirty
  identity on either field (round 5). Banking alone still treats `-dirty` as known:
  `releaseTag` collapses `-dirty` into the release bucket *deliberately*, pinned by an existing
  test row (V28), and dirty-tree drift is a subset of the intra-release drift the per-release
  cadence already pools by design.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Cohort freeze **refuses** unknown OR dirty identity — gate = `VersionKnown() && CommitClean()`; no `--allow-unstamped`/`--allow-dirty` escape | Release evidence integrity; any override re-opens the silent path this doc closes. Round 5 widened the gate from knownness to cleanliness: a `-dirty` stamp is visible but not reproducible (the manifest carries no source diff or compiler-content identity, V27), and the strict gate is satisfiable by the doc's own recipe from a committed tree at the same commit (V26) — refusing dirty relocates the once-per-release freeze onto a clean tree, which is what deterministic release evidence means | human (unknown, rounds 0–4) + agent (dirty, round 5, per `gpt5-6-sol`) | design | med |
| Cache identity gated on `Identity.CommitClean()`: a known commit with `-dirty` on **neither** `Commit` nor `Version` keys the cache exactly as today; unstamped, or `-dirty`-stamped on either field, **declines to cache** (no Lookup, no Store) rather than deriving a substitute identity | A substitute identity is either a stale-blob risk (a constant key, incl. a shared `abc1234-dirty` — the v1→v3 bump class, V8) or a ~215 ms/process tax to make safe (full-content hash, V22/V23) against a measured 0–12 ms cache benefit; declining to cache is correct **by construction** — no lookup, no stale blob. Both fields must be checked because the two stamping paths put `-dirty` on DIFFERENT fields (V24) | human | design | high |
| `Version == "dev"` + clean known `Commit` **still caches** under the widened gate — an unknown `Version` is treated as "not dirty" | The only in-contract population with this shape is the plain `go build`/`go run`/`go test` in a real `.git` directory, where dirtiness lands on the **`Commit`** arm (ReadBuildInfo appends `-dirty` to `Commit`, V24/V25); a clean real SHA fully identifies the bytes, and refusing it would gut caching for every clean-tree `go test` build for zero correctness gain | agent (round 4) | design | low |
| Unstamped banking bucket = `unstamped-<id8>` per build (not refusal, not shared `dev/`), via `AttributionID()` | Diverges from m-eval-version-banking's per-release cadence for the unstamped case only; changes on-disk layout eval consumers see | human | design | med |
| `CommitClean()` is a pure string check on already-loaded `Identity` fields — no `os.Stat`, no `!ok` branch, nothing to fail on. The cache gate cannot itself error; only `AttributionID()` (Consumer 3 only) retains an `os.Stat`-derived `!ok` branch, on which Consumer 3 refuses `--bank-by-version` | Two failure surfaces should not share one control-flow shape: a cache gate that can itself fail invites a fallback-on-error footgun; keeping the gate to a string comparison removes that surface entirely | human | design | med |
| Knownness sentinel stays the existing `"dev"` literal; ldflags/Makefile contract untouched | Changing the stamping contract would touch CI, Makefile, release tooling | agent | design | low |
| Accessor lives in `internal/version` (single source of knownness) | Three call sites otherwise re-derive `!= "dev"` independently and drift | agent | implementation | low |

### Design Freeze

Resolved before implementation starts:

- [x] Freeze refusal has no escape flag, and covers dirty-stamped identity on either field as
      well as unstamped (round 5; gate = `VersionKnown() && CommitClean()`). Remediation = build
      with ldflags **from a committed-clean tree**: the recipe itself was executed end-to-end in
      a linked worktree (V17), and its inputs stamp clean exactly when `git status --porcelain`
      is empty (V26)
- [x] Cache gate = `Identity.CommitClean()` (known commit AND no `-dirty` suffix on either
      `Commit` or `Version` — both stamping paths covered, V24); unstamped or `-dirty`-stamped
      on either field **declines to cache** entirely (no Lookup, no Store) + one stderr warning — no substitute
      identity is ever derived, and the gate itself cannot fail (`CommitClean()` is a string
      check on already-loaded fields, so there is no `!ok` branch on the caching path)
- [x] Banking degrade = `unstamped-<id8>` bucket; `releaseTag()` itself is NOT modified (its
      `"dev"→"dev"` pin in `TestReleaseTag` stays green; the fix is at the call site — the exact
      "guard the helper, miss the call site" failure mode named in `cache_key.go:20-24`, V8/V13)
- [x] Exported vars `Version`/`Commit`/`BuildTime` remain (compat: `cmd/ailang/main.go:27-28` aliases, V10)

## Deferred Decisions

Intentionally left open for the implementer:

- Exact refusal message wording (branches 1–2 must contain the substring `unstamped`, branch 3
  the substring `dirty` and not `unstamped`; all three carry the remediation command; see M4)
  — agent
- `AttributionID` prefix/length and exact stat-field composition (`exe-` + ≥12 hex chars of
  sha256 over a composition that includes at least `dev`, `inode`, `size`, `mtimeUnixNano` — or
  an equivalent meeting the three requirements in the API section) — agent
- Whether `--version` gains a cosmetic "(build identity: unstamped)" line — agent (out of the
  acceptance gate either way)
- Test file naming/organization within the touched packages — agent
- Where the one-shot cache-bypass warning is emitted (call site vs helper) — agent

## Solution Design

### Overview

Add a knownness-aware accessor to `internal/version`; leave the ldflags/init() contract exactly
as-is. Each decision-bearing consumer then makes an explicit choice — and they legitimately
differ, because their failure costs differ: guessed *evidence* is unrecoverable (→ refuse),
a guessed *cache key* is silently wrong in both directions (→ decline to cache rather than
guess), and refused *eval data* would just be lost (→ bank it under an explicit unstamped
marker, using a metadata-derived attribution id that is never used as a cache key).

### API (internal/version)

```go
// Identity is the build identity together with whether each component is known.
type Identity struct {
    Version   string // "v0.33.1" or "dev"
    Commit    string // full SHA, possibly "-dirty"-suffixed, or "dev"
    BuildTime string
}

func (i Identity) VersionKnown() bool // i.Version != "" && i.Version != "dev"
func (i Identity) CommitKnown() bool  // i.Commit  != "" && i.Commit  != "dev"
func (i Identity) CommitClean() bool  // i.CommitKnown() && !strings.HasSuffix(i.Commit, "-dirty") && !strings.HasSuffix(i.Version, "-dirty")

// Get returns the package vars as one value (post-init(), i.e. after the
// ReadBuildInfo fallback has had its chance to populate Commit).
func Get() Identity

// AttributionID returns a best-effort identifier for bucketing eval output
// (Consumer 3 only). It is an ATTRIBUTION identifier, not a correctness
// identity, and MUST NEVER be used as a cache key — Consumer 2 does not call
// it; the module cache is instead gated directly on Identity.CommitClean().
//   CommitKnown && !strings.HasSuffix(Commit, "-dirty")
//                -> the commit string
//                   (narrower than CommitClean() by design: on this function's
//                   only call path Version == "dev" — Consumer 3 calls it only
//                   when !VersionKnown() — so a Version arm would be vacuous;
//                   see the round-4 revision log, NOTED item 1)
//   else         -> "exe-" + hex(sha256(fmt.Sprintf("%d:%d:%d:%d",
//                       dev, inode, size, mtimeUnixNano)))[:12]
//                   (or an equivalent stat-derived composition), from os.Stat
//                   on os.Executable(), computed at most once per process
//                   (sync.Once)
// ok == false (id == "") only when the executable cannot be located or Stat'ed.
// Callers MUST fail safe on !ok — Consumer 3 refuses --bank-by-version rather
// than guess a bucket.
func AttributionID() (id string, ok bool)
```

**`CommitKnown()` vs `CommitClean()`.**
`CommitKnown()` answers *knownness* — the value is a real stamp, not the `"dev"` sentinel.
Through round 4, Consumer 1 accepted a `-dirty` commit on the argument that `-dirty` is visible
to a reviewer; round 5 retires that argument, because the standard Consumer 1 must protect is
*reproducibility*, not visibility — the manifest records no source diff and no compiler-content
identity (V27), so a reviewer holding `abc1234-dirty` knows the bytes were modified but cannot
recover them. Consumer 1 therefore gates on `VersionKnown() && CommitClean()`, refusing dirty
on either field; `CommitKnown()` survives as a component predicate whose own refusal branch
yields the distinct `unstamped` message (M4).
`CommitClean()` answers *bytes*-identity, and it must check BOTH fields because the two stamping
paths put `-dirty` on DIFFERENT fields (V24): the `ReadBuildInfo` fallback appends `-dirty` to
`Commit` when `vcs.modified` is true — and never touches `Version` at all (V25) — while the
Makefile/ldflags path stamps `Commit=$(git rev-parse HEAD)`, a **plain SHA even in a dirty
tree**, and carries dirtiness only on `Version=$(git describe --tags --always --dirty)`
(`Makefile:27-28`, V24). A dirty identity on either path aliases different compiler bytes under
one string — the shared `abc1234-dirty` on the ReadBuildInfo path; the plain commit SHA,
indistinguishable from the CLEAN build at that commit, on the ldflags path — so neither can key
a cache. Consumer 2 gates the module cache on `CommitClean()` alone, and any build that is
unstamped or `-dirty`-stamped on either field declines to cache rather than being assigned a
substitute identity. Knownness and bytes-identity are different requirements;
`Identity.CommitKnown()` answers the first, `Identity.CommitClean()` the second — and from
round 5 the freeze gate composes both (`VersionKnown() && CommitClean()`).

**The `Version == "dev"` + clean-`Commit` case, and the asymmetry it creates (decided in round
4; see High-Impact Decisions).** The predicate's `Version` arm rejects a `-dirty` *suffix*, not
an unknown `Version`: `!strings.HasSuffix("dev", "-dirty")` is true, so a build with
`Version == "dev"` and a clean known `Commit` **still caches**, keyed on that commit. That is
the right answer: this shape is the plain `go build`/`go run`/`go test` in a real `.git`
*directory*, where only the ReadBuildInfo fallback runs — it populates `Commit` (appending
`-dirty` iff `vcs.modified`, V25) and never assigns `Version` (V7/V25) — so a clean real SHA in
`Commit` fully identifies the compiler bytes, and `Version` being unknown says nothing about
dirtiness. The asymmetry, stated out loud: an unknown `Version` is treated as "not dirty", so a
*dirty* tree under plain `go build` in a real `.git` directory is caught by the **`Commit`** arm
(ReadBuildInfo stamps `-dirty` there), not by the `Version` arm — while a dirty tree under
`make`/ldflags is caught by the **`Version`** arm, not the `Commit` arm. The two arms cover
different populations and neither alone is sufficient; that is the whole point of the round-4
widening. Residual, named for reviewers: an off-contract build that hand-stamps only `Commit`
(a plain SHA from a dirty tree, without the Makefile's `Version` stamp) evades both arms — no
string predicate can detect a clean-*looking* stamp that lies; the documented Makefile and
remediation contract stamps both fields together (V17), and changing that contract is already a
settled non-goal (knownness-sentinel decision, High-Impact Decisions).

**Requirements on the non-commit branch** (all three are binding; the composition above is one
way to meet them; this branch now feeds only `AttributionID()` — the module cache no longer
derives a substitute identity at all, see below):

1. **Deterministic across processes** for the same binary file — same `dev`/`inode`/`size`/
   `mtime` → same id (a per-process or per-call component would defeat the point of an
   attribution bucket that is supposed to identify one build).
2. **Differs across rebuilds** — `go build -o X` and `make install` both rewrite `X`, giving a
   new mtime and normally a new inode, so two different unstamped builds bank into different
   buckets.
3. **Cost is a `stat`, not a read** — no 96 MB hash. The rejected full-content-hash alternative
   measured **~215 ms per process** on the current binary (V22/V23) — unacceptable even for the
   attribution-only path, let alone the compile-hot cache path it was originally proposed for.

**The cache does not use this branch at all.** Consumer 2 gates on `Identity.CommitClean()`
directly: an unstamped or `-dirty`-stamped (either field) build declines to cache entirely (no Lookup, no Store)
rather than being assigned a stat-derived or any other substitute identity. So the
over-invalidation-vs-under-invalidation question that applied to a cache identity does not arise
on that surface — there is no cache identity for an ambiguous build to get right or wrong.
`AttributionID()`'s stat-derived fallback remains metadata-derived, not content-addressed; that
is acceptable because bucketing eval output (Consumer 3) is an attribution problem, not a
correctness one — a wrong bucket mislabels data, it cannot serve a stale compiled artifact.

Internal seam for tests: `attributionIDFromPath(path string) (string, error)` — stat-derives the
id for an arbitrary file, so the fallback and error branches are testable without faking
`os.Executable`. No existing executable-identity helper exists to reuse (V19).

### Consumer 1 — cohort freeze: REFUSE (M4)

Extend `validateCohortFreeze` (currently `(baselineSet, baselineID, verify)` → add
`id version.Identity`, V16) with three refusal branches ahead of the existing `--verify` branch
— the gate is `VersionKnown() && CommitClean()`:

1. `baselineSet && !id.VersionKnown()` → error: freezing release evidence from an unstamped
   build; the manifest would record `ailang_version: "dev"`.
2. `baselineSet && id.VersionKnown() && !id.CommitKnown()` → same for `git_commit`.
3. `baselineSet && id.VersionKnown() && id.CommitKnown() && !id.CommitClean()` → error:
   freezing release evidence from a **dirty-stamped** build — `-dirty` on `Commit`
   (ReadBuildInfo path) or on `Version` (ldflags path; the two paths differ, V24). The manifest
   records no source diff or compiler-content identity (V27), so a `-dirty` value names bytes
   no reviewer can reconstruct (round 5, per `gpt5-6-sol`). The branch reuses `CommitClean()`,
   whose per-arm behaviour is already pinned by `TestIdentityKnownness` and mutation row 8b.

Branch 1–2 messages MUST contain the substring `unstamped`; branch 3 messages MUST contain the
substring `dirty` and MUST NOT contain `unstamped` (each branch distinguishable from the others
and from `validateBaselineID`/`--verify` errors — an `err != nil` assertion alone is
decorative). All three carry the remediation:

```
rebuild with build identity, e.g.
  go build -ldflags "-X github.com/sunholo-data/ailang/internal/version.Version=$(git describe --tags --always --dirty) \
    -X github.com/sunholo-data/ailang/internal/version.Commit=$(git rev-parse HEAD)" -o <out> ./cmd/ailang
(works in linked worktrees: the Makefile derives these the same way — Makefile:27-29;
 the tree must be COMMITTED-CLEAN first: with uncommitted changes `git describe` stamps
 `…-dirty` and the freeze refuses again — commit or stash, then rebuild; V26)
```

The remediation is **executed-verified, not inferred**: run from this linked worktree (detached
at `96381331a`), the ldflags build stamps both `Version` and `Commit` exactly, matching
`git describe` and `git rev-parse HEAD` (V17). One expectation to pre-empt: the remediated
binary still shows **zero** `vcs.*` build settings under `go version -m` (V17: 0 `vcs\.` lines;
11 `build` settings vs 10 for the unstamped build — the extra one is the recorded `-ldflags`).
The ldflags mechanism sets the package vars directly and does **not** restore Go's VCS
stamping. That is sufficient, because all three decision-bearing consumers read
`version.Version`/`version.Commit`, never `debug.ReadBuildInfo()` — do not conclude from
`go version -m` that the remediation failed.

This lands **pre-spend** by construction: `validateCohortFreeze` already runs before the rig
lock, the API-key check, and any benchmark (V16), and the manifest is only ever written on the
freeze path (`if baselineSet`, V12) — so non-freeze runs are untouched, satisfying the "refusal
scoped to published/frozen evidence, not binary startup" constraint. No schema change to the
manifest; a manifest that exists implies **clean**-stamped provenance from M4 on.

### Consumer 2 — module cache: BYPASS ON AMBIGUOUS IDENTITY (M2)

At `pipeline_module.go:276`, replace the raw `version.Commit` with:

```go
id := version.Get()
if !id.CommitClean() {
    // Ambiguous compiler identity (unstamped, or -dirty-stamped on either
    // Commit or Version -- the two stamping paths differ, V24): do NOT cache
    // this run — no Lookup, no Store. One stderr warning. Recompiling is
    // always safe; serving a blob under an identity that cannot distinguish
    // compiler bytes is the failure class this doc exists to close.
} else {
    moduleCacheKey = ModuleCacheKey(id.Commit, sourceContent, depDigests)
    ...
}
```

Clean-stamped builds are **byte-identical to today** (the overwhelmingly common CI/release
path); unstamped and dirty-stamped builds lose module caching, whose measured benefit on the
examples corpus is **0–12 ms** (V22/V23), against a rejected content-hash alternative costing
**~215 ms per process** (V22). This is correct **by construction**, not by argument: a stale
blob cannot be served if no lookup ever happens.

- **Clean-stamped builds:** `CommitClean()` is true, so `moduleCacheKey` is keyed on `id.Commit`
  — byte-identical to today; zero change.
- **Dirty-stamped builds — BOTH stamping paths** (the round-3 finding, V24). On the
  `make`/ldflags path (the commonest developer state), `Commit` is a **plain SHA even from a
  dirty tree** and dirtiness rides on `Version` (`…-dirty` from `git describe`), so under the
  pre-round-4 predicate these builds kept caching under the pure SHA — aliasing dirty compiler
  bytes with the *clean* build at that commit; the **`Version` arm** now catches them. On the
  plain-`go build` (ReadBuildInfo) path, dirtiness rides on `Commit` (`abc1234-dirty`, shared by
  every rebuild from that dirty tree while the bytes differ); the **`Commit` arm** catches
  those. Either way `CommitClean()` is false and the run is **bypassed entirely** — no Lookup,
  no Store, one stderr warning — rather than being given its own substitute identity. This is an
  **intended behaviour change** on this one surface (Objection 1, revision round 1; sharpened in
  rounds 2 and 4): the cache benefit given up (0–12 ms, V23) is strictly cheaper than any
  correctness-preserving alternative that was priced (~215 ms/process for a content hash, V22).
- **Unstamped builds:** `CommitKnown()` is false, so `CommitClean()` is false too — same bypass as
  the dirty-stamped case. No `os.Stat`, no computed id, nothing to be wrong: the run simply does
  not participate in the module cache.
- **`Version == "dev"` + clean-`Commit` builds** (plain `go build`/`go test` in a real `.git`
  directory at a clean tree): `CommitClean()` is true — `Commit` is a clean real SHA and `"dev"`
  does not end in `-dirty` — so these **still cache**, keyed on `id.Commit`. Correct by the arm
  asymmetry documented in the API section: on this build path dirtiness would have landed on
  `Commit` itself (V25), so a clean `Commit` is a sufficient bytes-identity and the unknown
  `Version` carries no dirtiness signal.
- **`go test` binaries:** also unstamped in worktrees, so also bypassed for the whole run; pipeline
  tests use temp cache dirs regardless, so this has no practical effect on test behaviour.

`ModuleCacheKey` itself, `cacheKeyVersion`, source hashing, and dep digests are untouched; all
seven existing `TestModuleCacheKey_*` tests (V14) must pass unmodified.

### Consumer 3 — eval banking: EXPLICIT UNSTAMPED MARKER (M3)

At `eval_suite.go:191`, extract a helper (e.g. `bankBucket(id version.Identity) (string, error)`):

- `id.VersionKnown()` → `releaseTag(id.Version)` — today's behaviour, byte-identical, including
  the per-release no-churn cadence m-eval-version-banking chose deliberately.
- else → `"unstamped-" + <AttributionID first 8>` (e.g. `unstamped-exe-a1b2c3d4` collapses to
  `unstamped-<id8>` — implementer picks the exact composition; it must start with `unstamped-`
  and must not collide with the `dev` literal).
- else-and-`!ok` → **refuse `--bank-by-version`** (exit non-zero, same remediation text as M4):
  with no attributable identity at all, banking anywhere either pools or lies.

Why a marker rather than refusal: unstamped eval runs are this loop's legitimate daily business;
their data must land *attributably*, not be refused (refusal would push operators to ad-hoc
`--output` dirs and lose the version-consistency of the pipeline). Why per-build granularity is
acceptable here despite m-eval-version-banking's per-release rule: an unstamped build has no
release identity to pool under — pooling unrelated builds is the defect itself — and the nightly
rotation runs `make`-installed (stamped) binaries, so the no-churn property is preserved exactly
where it was designed. The single-mutation-point property holds: the bucket is joined into
`*outputDir` once at flag-parse time, and all downstream readers (`cleanResults`, the
`--skip-existing` glob, the queue and parallel runners, the metrics writer) read `*outputDir`
(V12), so one edit keeps the whole pipeline consistent.

**Scope of the `gpt5-6-sol` objection.** That objection was scoped to the *cache identity* ("Do
not use stat metadata as the cache identity") — this surface is attribution, not correctness: a
wrong bucket mislabels data, it cannot produce a wrong compile. `AttributionID()`'s stat-derived
fallback (metadata-derived, not content-addressed) is retained here on that basis. The controller
made this scoping call — it is the one judgment in the carve-out revision beyond the reviewers'
own proposed fixes, and it is offered to the reviewers to attack, not asserted as settled.

### Consumers 4–7 — display/service surfaces: UNCHANGED

`--version` output, the microrag MCP server label, and the prompt/MCP `forVersion` path keep
reading the raw vars. For #5/#6 an unstamped `"dev"` means the MCP prompt fetch version-matches
`"dev"` or falls back to embedded content and the on-disk prompt cache keys under `dev/` — a
service-grade degradation that is already the designed fallback (V10). Listed as a residual, not
designed here.

## Files to Modify/Create

**New files:**
- `internal/version/identity_test.go` (~170 LOC) — first tests for the package: knownness truth
  table (`VersionKnown`/`CommitKnown`/`CommitClean`, incl. the two cross-field rows — plain-SHA
  `Commit` + `-dirty` `Version` → not clean, V24; `"dev"` `Version` + clean `Commit` → clean),
  `AttributionID` commit-preference incl.
  `-dirty` exclusion, stat-id determinism + rebuild-sensitivity, unstat-able-path branch
- `internal/pipeline/compiler_identity_test.go` (~120 LOC) — `CommitClean()` gate selection +
  cache-bypass (five behavioural arms incl. dummy-artifact pre-population, see AC-6)
- `cmd/ailang/eval_suite_bank_test.go` (~60 LOC) — bucket helper table test

**Modified files:**
- `internal/version/version.go` (+~60 LOC) — `Identity`, `Get`, `CommitClean`, `AttributionID`,
  `attributionIDFromPath`
- `internal/pipeline/pipeline_module.go` (+~15/−3 LOC) — call-site swap + bypass branch
- `cmd/ailang/eval_suite.go` (+~15/−4 LOC) — `bankBucket` helper at the single mutation point
- `cmd/ailang/eval_suite_cohort.go` (+~35 LOC) — three refusal branches in `validateCohortFreeze`
- `cmd/ailang/eval_suite_cohort_test.go` (+~90 LOC) — new refusal tests (five shapes, AC-11);
  existing signature callers updated (testing policy: rewrite outdated tests, no backward-compat
  shims)
- `changelogs/v0.18-current.md` — entry under Fixed/Changed

## Milestones

Ordered so every intermediate state is behaviour-identical or strictly better; no intermediate
commit ships a regression. M1 is a pure addition; M2/M3/M4 each depend only on M1 and are
independently shippable.

---

### M1 — `internal/version`: knowable identity + first tests (1 day; behaviour-identical)

Adds `Identity`/`Get`/`CommitClean`/`AttributionID`/`attributionIDFromPath`. No call site
changes. The package currently has ZERO test files (V11) and is about to become load-bearing —
the tests are a deliverable, not a garnish, and they also pin the *existing* init() fallback
semantics (Commit-only, Version never).

**Acceptance criteria** (all commands baselined rc=0 on pristine dev — V20; `go build ./...` is
rc=1 on pristine dev (V21) and is banned from every criterion in this doc):

- **AC-1:** `go build ./internal/version/... && go vet ./internal/version/...` → rc=0.
- **AC-2** (enumeration floor, shape from m-list-accessor-api):
  ```bash
  L=/tmp/m1.log
  go test ./internal/version/ -count=1 -v \
    -run '^(TestIdentityKnownness|TestAttributionIDPrefersCommit|TestAttributionIDExeStatDeterministic|TestAttributionIDUnstatablePath)$' >"$L" 2>&1
  RC=$?
  N=$(grep -c '^=== RUN   Test[^/]*$' "$L")   # top-level only; excludes subtests
  [ "$N" -eq 0 ] && { echo "INSTRUMENT FAILURE: selector matched nothing"; exit 3; }
  [ "$N" -eq 4 ] && [ "$RC" -eq 0 ]           # EXPECTED_N=4, literal
  ```
  `TestIdentityKnownness` MUST cover `CommitClean()` as well as `VersionKnown()`/`CommitKnown()`
  (three predicates, not two), and its `CommitClean()` rows MUST include the two cross-field
  cases: **plain-SHA `Commit` + `-dirty` `Version` → false** (the direct regression pin for the
  round-3 defect — this row fails against the round-2/3 predicate, V24) and **`"dev"` `Version`
  + clean-SHA `Commit` → true** (the documented still-caches case, High-Impact Decisions).
  `TestAttributionIDPrefersCommit` MUST carry a `-dirty` row: an
  injected `abc1234-dirty` commit must NOT come back as the id (the id must carry the `exe-`
  prefix) — a clean-commit-only test would pass on the pre-revision aliasing design (mutation
  row 4).
- **AC-3:** `go test ./internal/version/... -count=1` output contains `ok ` and does NOT contain
  `[no test files]` (positive-string assertion — the pre-change output IS `[no test files]` with
  rc=0, V11, so rc alone proves nothing).
- **AC-4:** `gofmt -l internal/version` → 0 lines.

---

### M2 — module-cache compiler identity (1 day; strictly better)

Call-site swap in `pipeline_module.go` to gate on `Identity.CommitClean()`, with bypass on
ambiguous identity, per Solution Design.

**Acceptance criteria:**

- **AC-5** (regression floor — the seven existing key tests, unmodified):
  ```bash
  L=/tmp/m2a.log
  go test ./internal/pipeline/ -count=1 -v \
    -run '^(TestModuleCacheKey_Deterministic|TestModuleCacheKey_DifferentSource|TestModuleCacheKey_DifferentVersion|TestModuleCacheKey_DifferentDep|TestModuleCacheKey_DepOrderIndependent|TestModuleCacheKey_CommitChange|TestModuleCacheKey_NoDeps)$' >"$L" 2>&1
  RC=$?; N=$(grep -c '^=== RUN   Test[^/]*$' "$L")
  [ "$N" -eq 0 ] && { echo "INSTRUMENT FAILURE"; exit 3; }
  [ "$N" -eq 7 ] && [ "$RC" -eq 0 ]           # EXPECTED_N=7
  ```
- **AC-6** (new behaviour floor):
  ```bash
  L=/tmp/m2b.log
  go test ./internal/pipeline/ -count=1 -v \
    -run '^(TestCompilerCacheIdentity_StampedUsesCommit|TestModuleCache_BypassedWhenIdentityAmbiguous)$' >"$L" 2>&1
  RC=$?; N=$(grep -c '^=== RUN   Test[^/]*$' "$L")
  [ "$N" -eq 0 ] && { echo "INSTRUMENT FAILURE"; exit 3; }
  [ "$N" -eq 2 ] && [ "$RC" -eq 0 ]           # EXPECTED_N=2
  ```
  `TestModuleCache_BypassedWhenIdentityAmbiguous` carries five arms — ReadBuildInfo-dirty
  (`Commit` = `abc1234-dirty`), **ldflags-dirty** (`Commit` = plain SHA, `Version` =
  `v0.33.1-…-dirty` — the behavioural regression pin for the round-3 defect, V24; this arm
  FAILS against the round-2/3 predicate), unstamped, clean-stamped, and `"dev"`-`Version` with
  clean-SHA `Commit` — driven by setting the exported `version.Commit`/`version.Version` vars
  directly and restoring them with `defer` (no `sync.Once` seam needed; resolves
  `gemini-3-1-pro`'s testability objection, see the Quorum revision log, round 2). The three
  bypass arms must pre-populate the cache directory with a dummy artifact keyed by the
  ambiguous identity, and assert that the compiler executes anyway (proving Lookup bypass) and
  that no new files are written to the cache (proving Store bypass). Asserting only that the
  directory stays empty in a fresh environment leaves Lookup bypass unverified (round 5, per
  `gemini-3-1-pro`): a mutant that still performs `Lookup` on the ambiguous key simply misses
  in a fresh directory, skips `Store`, and leaves it empty — Failure Class 1 intact, test
  green (mutation row 8c). The two caching arms (clean-stamped; `"dev"`-`Version` + clean
  `Commit`) must assert caching still happens.
- **AC-7:** `go build ./internal/pipeline/... && go test ./internal/pipeline/ -count=1` → rc=0
  (full package suite; baselined rc=0 pristine, V20).

---

### M3 — banking bucket marker (0.5 day; strictly better)

`bankBucket` helper at `eval_suite.go:191`; `releaseTag` untouched.

**Acceptance criteria:**

- **AC-8** (existing pins prove `releaseTag` untouched, incl. its `"dev"→"dev"` row):
  ```bash
  L=/tmp/m3a.log
  go test ./cmd/ailang -count=1 -v -run '^TestReleaseTag$' >"$L" 2>&1
  RC=$?; N=$(grep -c '^=== RUN   Test[^/]*$' "$L")
  [ "$N" -eq 0 ] && { echo "INSTRUMENT FAILURE"; exit 3; }
  [ "$N" -eq 1 ] && [ "$RC" -eq 0 ]           # EXPECTED_N=1
  ```
- **AC-9** (new behaviour floor):
  ```bash
  L=/tmp/m3b.log
  go test ./cmd/ailang -count=1 -v \
    -run '^(TestBankBucket_UnstampedMarker|TestBankBucket_StampedUnchanged)$' >"$L" 2>&1
  RC=$?; N=$(grep -c '^=== RUN   Test[^/]*$' "$L")
  [ "$N" -eq 0 ] && { echo "INSTRUMENT FAILURE"; exit 3; }
  [ "$N" -eq 2 ] && [ "$RC" -eq 0 ]           # EXPECTED_N=2
  ```
  `TestBankBucket_StampedUnchanged` must assert byte-equality with `releaseTag`'s output for a
  stamped version string (e.g. `v0.26.0-26-g9249a66bf` → `v0.26.0`), so the stamped path is
  proven identical, not merely "some string". `TestBankBucket_UnstampedMarker` must assert the
  bucket equals `"unstamped-"` + the first 8 characters of the id returned by a direct
  `AttributionID()` call in the same process — asserting only the prefix plus a non-empty
  suffix is satisfiable by a constant such as `unstamped-deadbeef`, which would silently
  re-pool every unstamped build into one shared bucket, i.e. re-create the exact defect this
  consumer exists to close (round-5 sweep; same class as the AC-6 hole).
- **AC-10:** `go vet ./cmd/ailang` → rc=0.

---

### M4 — freeze refusal + docs (1 day; strictly better)

Refusal branches in `validateCohortFreeze`; CHANGELOG; this doc's status stays Planned until the
sprint ships.

**Acceptance criteria:**

- **AC-11** (one test per refusal shape + the pass branch):
  ```bash
  L=/tmp/m4a.log
  go test ./cmd/ailang -count=1 -v \
    -run '^(TestValidateCohortFreeze_RefusesUnstampedVersion|TestValidateCohortFreeze_RefusesUnstampedCommit|TestValidateCohortFreeze_RefusesDirtyCommit|TestValidateCohortFreeze_RefusesDirtyVersion|TestValidateCohortFreeze_PassesStampedIdentity)$' >"$L" 2>&1
  RC=$?; N=$(grep -c '^=== RUN   Test[^/]*$' "$L")
  [ "$N" -eq 0 ] && { echo "INSTRUMENT FAILURE"; exit 3; }
  [ "$N" -eq 5 ] && [ "$RC" -eq 0 ]           # EXPECTED_N=5
  ```
  The two unstamped-refusal tests must assert the error message contains `unstamped`. The two
  dirty-refusal tests (round 5) cover BOTH stamping shapes pre-spend, per `gpt5-6-sol`:
  `_RefusesDirtyCommit` injects `Commit = "abc1234-dirty"` + a clean known `Version` (the
  ReadBuildInfo shape) and `_RefusesDirtyVersion` injects `Version = "v0.34.0-dirty"` + a
  plain-SHA `Commit` (the ldflags shape, V24); each must assert the message contains `dirty`
  and does NOT contain `unstamped` (distinguishes each branch from the others and from
  `validateBaselineID`/`--verify` errors, which can also make `err != nil` true).
  `_PassesStampedIdentity` must use a clean-stamped identity of the exact shape the prescribed
  recipe produces in a clean worktree (`git describe`-style `Version` with no `-dirty` + a
  plain-SHA `Commit`, V26) and assert `err == nil`.
- **AC-12** (existing freeze-gate behaviour retained):
  ```bash
  L=/tmp/m4b.log
  go test ./cmd/ailang -count=1 -v \
    -run '^(TestValidateCohortFreeze|TestValidateCohortFreeze_VerifyMessageExplainsWhy|TestEvalSuiteBaselineRequiresVerify_CLI)$' >"$L" 2>&1
  RC=$?; N=$(grep -c '^=== RUN   Test[^/]*$' "$L")
  [ "$N" -eq 0 ] && { echo "INSTRUMENT FAILURE"; exit 3; }
  [ "$N" -eq 3 ] && [ "$RC" -eq 0 ]           # EXPECTED_N=3 (all three exist today, V16)
  ```
- **AC-13:** `make check-changelog` → rc=0, AND the entry is positively asserted:
  `grep -c 'cohort-manifest-build-provenance' changelogs/v0.18-current.md` ≥ 1 (the entry must
  name this doc's slug). `make check-changelog` alone was rc=0 on pristine dev BEFORE any entry
  existed (V20), so its exit code cannot prove the new entry is present — the same rc-only hole
  AC-3 closes for `[no test files]` (round-5 sweep).
- **AC-14:** `make check-file-sizes && make check-boundaries` → rc=0.

**Total: 4 milestones, 14 acceptance criteria.**

## Mutation table

One row per shipped behaviour. Mutations keep every import used (`if false && cond`, forced
booleans) so "mutant does not compile" never masquerades as "guard fired". Per assertion the
authors asked: *what else could produce this observed value?* — noted where it shaped the test.

| # | Milestone | Behaviour | Mutation (keeps imports) | Killed by |
|---|-----------|-----------|--------------------------|-----------|
| 1 | M1 | `VersionKnown()` false for `"dev"` | `return true` in `VersionKnown` | `TestIdentityKnownness` (truth table incl. `"dev"`, `""`, `"v0.33.1"`) |
| 2 | M1 | `CommitKnown()` false for `"dev"` | `return true` in `CommitKnown` | `TestIdentityKnownness` |
| 3 | M1 | `AttributionID` prefers the commit when known AND clean | `if false && i.CommitKnown()` | `TestAttributionIDPrefersCommit` (injected clean commit must come back verbatim) |
| 4 | M1 | `AttributionID` excludes `-dirty` commits from the commit branch | `if false && strings.HasSuffix(i.Commit, "-dirty")` (`strings` import stays used) | `TestAttributionIDPrefersCommit` (`-dirty` row: injected `abc1234-dirty` must NOT come back; id must carry the `exe-` prefix) |
| 5 | M1 | `AttributionID` stat-id fallback is deterministic across calls | append a per-call counter into the stat composition (`fmt.Appendf`, import stays used) | `TestAttributionIDExeStatDeterministic` (two calls on the same fixture file must be equal) |
| 6 | M1 | `AttributionID` stat-id changes when the file is rewritten | replace `mtimeUnixNano` in the composition with the constant `0` | `TestAttributionIDExeStatDeterministic` (rewrite the fixture + `os.Chtimes` bump → id must differ; same-size rewrite is the hard case, which is why mtime is load-bearing) |
| 7 | M1 | `!ok` on unstat-able executable path, id empty (`AttributionID`, Consumer 3 only) | `ok = true` on the error branch | `TestAttributionIDUnstatablePath` (asserts `ok==false` AND `id==""` — `id==""` alone is also the zero value, so both are asserted) |
| 8 | M2 | Ambiguous (unstamped, or `-dirty` on either field) identity declines module caching entirely — no Lookup, no Store, no substitute key ever computed | `if false && !id.CommitClean()` (bypass branch dead; `id` stays used) | `TestModuleCache_BypassedWhenIdentityAmbiguous` (all three bypass arms — ReadBuildInfo-dirty, ldflags-dirty, unstamped: the pre-populated dummy artifact keyed by the ambiguous identity is never read — the compile executes anyway — and no new cache files appear; round 5 replaced the empty-dir assertion, see AC-6). NOTE: this mutant reds regardless of which arm selected the bypass, so it does NOT discriminate the round-4 `Version` arm — row 8b pins that |
| 8b | M1/M2 | The `Version` arm of `CommitClean()` is load-bearing: a plain-SHA `Commit` with a `-dirty` `Version` (every `make`/ldflags dirty build, V24) must NOT be clean | drop only the `Version` arm — `return i.CommitKnown() && !strings.HasSuffix(i.Commit, "-dirty")` (compiles; `strings` stays used; this IS the round-2/3 predicate, i.e. the round-3 defect re-introduced) | `TestIdentityKnownness` (cross-field row: plain-SHA `Commit` + `-dirty` `Version` must yield `CommitClean() == false`) AND `TestModuleCache_BypassedWhenIdentityAmbiguous` (ldflags-dirty arm: compile executes despite the pre-populated dummy; no new cache files). The symmetric Commit-arm-drop mutant is killed by the same tests' pre-existing ReadBuildInfo-dirty rows/arms |
| 8c | M2 | The bypass is a **Lookup** bypass, not only a Store bypass — an ambiguous identity must never consult the cache at all (Failure Class 1: reading a stale blob) | hoist the lookup above the gate: compute `moduleCacheKey` and call `Lookup` unconditionally; keep only `Store` under `CommitClean()` (compiles; all imports stay used) | The **round-5-strengthened AC-6 arms**: the pre-populated dummy artifact keyed by the ambiguous identity makes the mutant's `Lookup` HIT, so the "compiler executes anyway" assertion fails — the stale dummy is served instead of a fresh compile. The **pre-round-5 empty-directory assertion does NOT kill this mutant**: in a fresh directory the mutant's `Lookup` misses, `Store` is skipped, and the directory stays empty — test green, Failure Class 1 intact. That contrast is what proves the AC-6 strengthening is load-bearing, not decorative (round 5, per `gemini-3-1-pro`) |
| 9 | M2 | Clean-stamped builds keep byte-identical keys | force the bypass arm on a clean commit (`clean := false; _ = clean` on the `CommitClean()` branch) | `TestCompilerCacheIdentity_StampedUsesCommit` (key equals `ModuleCacheKey(commit,…)` recomputed directly, with a clean injected commit); also the clean-stamped arm inside `TestModuleCache_BypassedWhenIdentityAmbiguous` asserting caching still happens |
| 10 | M3 | Unstamped run banks under `unstamped-<id8>` derived from `AttributionID()` — per-build, never a shared constant | `if false && !id.VersionKnown()` in `bankBucket`; also the constant-suffix mutant: `return "unstamped-deadbeef"` on the unstamped arm | `TestBankBucket_UnstampedMarker` (round 5: asserts the bucket EQUALS `"unstamped-"` + first 8 chars of a direct `AttributionID()` call in the same process — the constant-suffix mutant, which would silently re-pool every unstamped build into one bucket, survives the pre-round-5 assertion of "prefix + non-empty suffix" and dies to the equality) |
| 11 | M3 | Stamped run bucket byte-identical to today | force unstamped arm (`known := false; _ = known`) | `TestBankBucket_StampedUnchanged` + existing `TestReleaseTag` (AC-8) |
| 12 | M4 | Refusal branch: unknown Version | `if false && !id.VersionKnown()` | `TestValidateCohortFreeze_RefusesUnstampedVersion` (asserts substring `unstamped`) |
| 13 | M4 | Refusal branch: unknown Commit | `if false && !id.CommitKnown()` | `TestValidateCohortFreeze_RefusesUnstampedCommit` (asserts substring `unstamped`; run with VersionKnown=true so branch 12 cannot mask it) |
| 14 | M4 | Pass branch: **clean**-stamped identity + valid flags proceeds | force refusal (`|| true` on branch 12's condition) | `TestValidateCohortFreeze_PassesStampedIdentity` (asserts `err == nil` for a clean-stamped identity of the V26 recipe shape — this row also guards against an over-eager round-5 gate that refuses everything) |
| 15 | M4 | Refusal branch 3: dirty-stamped identity, either field (round 5) | `if false && !id.CommitClean()` on branch 3 (branch dead; imports stay used) | `TestValidateCohortFreeze_RefusesDirtyCommit` AND `TestValidateCohortFreeze_RefusesDirtyVersion` (each asserts substring `dirty` and absence of `unstamped`; the two tests cover the two stamping shapes from V24, so a mutant surviving one shape dies to the other; per-arm discrimination inside `CommitClean()` itself is already pinned by row 8b) |

## Conflict Surface

This design touches no parser/typechecker position, but it touches **compiler cache machinery**
(`internal/pipeline`) and the eval pipeline's output layout — both places where a wrong value is
silent.

### What else touches the module cache key

- `cacheKeyVersion` (`cache_key.go:24`, currently `"v3"`) — the FORMAT version, bumped when the
  blob shape changes (v1→v2 gob `RecordPattern.Rest`; v2→v3 `Iface.AliasParams`). Orthogonal to
  this change and untouched; both components still feed the same SHA-256 preimage.
- `sourceContent` — read from `mod.File.Path` (the `mod.Path` collision bug is documented at the
  call site, `pipeline_module.go:262-268`); untouched.
- `depDigests` — sorted dependency interface digests; untouched.
- The cache store (`Lookup`/`Store`/`LoadArtifacts`) keyed by `(modID, key)`; untouched.
- `ailang clean cache` / manual nukes — unaffected escape hatch either way.

### What a wrong key would do (the three failure classes)

1. **Constant across compiler builds** (today's defect, pre-fix: every worktree build shares
   `"dev"`; pre-revision, the ReadBuildInfo-dirty variant: every rebuild from a dirty tree
   shared `abc1234-dirty`; and pre-round-4, the ldflags-dirty variant: every `make` from a dirty
   tree cached under the **plain commit SHA**, aliasing dirty bytes with the clean build at that
   commit — V24): a rebuilt compiler decodes blobs written by a different compiler — the exact
   class the v1→v3 format bumps were shipped to defend against, but at the identity component the
   bumps cannot see (V8). Excluded by construction under this design: any identity that is not a
   known commit clean on BOTH fields (`CommitClean()` false) never keys the cache at all — it
   bypasses — so there is no constant substitute key left to alias different compiler bytes.
2. **Varying per process** (a bad fix): permanent 0% hit rate. Not a risk here: the only value
   that ever keys the cache is `id.Commit`, a static string already loaded once per process by
   `version.Get()` — there is no computed, potentially-unstable component on the caching path.
3. **Colliding across modules** (historical `mod.Path` bug): wrong artifacts reused. Not touched
   by this change (source/dep components unchanged), pinned by the existing
   `cache_invalidation_test.go`.

### What else reads the banking bucket

`*outputDir` is mutated ONCE at flag-parse (`eval_suite.go:187-197`) and read by `cleanResults`
(:575), the `--skip-existing` glob (:626-632), the queue runner (:690), the parallel runner
(:764), and the metrics writer (:770) — verified by grep (V12). The M3 edit stays inside that
single mutation point, so consistency is structural, not per-reader.

### What else feeds the freeze gate

`validateCohortFreeze` runs before the rig lock, API-key check, and any benchmark (V16). Its
existing error surfaces (`validateBaselineID` charset errors, the `--verify` refusal) share the
`err != nil` observable with the new refusals — which is why AC-11 asserts distinguishing
message text, not just non-nil.

### Programs/tests that MUST still work (regression fixtures)

- `internal/pipeline`: all seven `TestModuleCacheKey_*` (V14) and `cache_invalidation_test.go`,
  unmodified — AC-5, AC-7.
- `cmd/ailang`: `TestReleaseTag` (including its `"dev" → "dev"` row — `releaseTag` itself must
  not change) — AC-8.
- `cmd/ailang`: `TestValidateCohortFreeze`, `TestValidateCohortFreeze_VerifyMessageExplainsWhy`,
  `TestEvalSuiteBaselineRequiresVerify_CLI` (V16) — AC-12 (signature-change updates allowed;
  asserted behaviour retained).
- `cmd/ailang`: `TestBuildCohortManifest_RecordsEveryPlannedField`,
  `TestCohortHash_StableAcrossBuilds` (V16) — manifest schema and hash untouched.
- A stamped (`make`-built) binary end-to-end: `--version`, module compile with warm cache,
  `--bank-by-version` bucket path — all byte-identical to today.

### What deliberately changes (unstamped and dirty-stamped builds only)

- Module-cache behaviour, unstamped: `"dev"`-keyed caching (today's defect) → **no caching at
  all** for unstamped runs (`CommitClean()` is false; no Lookup, no Store, one stderr warning).
  This is a strict removal of the caching benefit on this surface, not a substitute identity —
  see Consumer 2 for the measured 0–12 ms cost (V22/V23).
- Module-cache behaviour, **dirty-stamped (both paths)**: `abc1234-dirty`-keyed caching on the
  ReadBuildInfo path, and plain-SHA-keyed caching on the `make`/ldflags path (dirtiness rides on
  `Version` only there — the round-3 finding, V24) → **no caching at all** — same bypass as
  unstamped, since `CommitClean()` is false for a `-dirty` suffix on either field. Each `make`
  from a dirty tree no longer risks decoding a blob written by different compiler bytes — or
  storing its own dirty blob under the plain SHA where a later CLEAN build at that commit would
  find it — because no rebuild from a dirty tree is ever cached. **Banking** behaviour for
  dirty-stamped builds is unchanged: a `-dirty` identity remains known via `VersionKnown()`,
  and `releaseTag` collapses it into the release bucket deliberately (existing pinned test row,
  V28). **Freeze** behaviour for dirty-stamped builds CHANGES in round 5: `-dirty` on either
  field is refused pre-spend (branch 3), because visible-but-unreproducible provenance fails
  the manifest's recompute purpose (V27).
- `--bank-by-version` bucket: `dev/` → `unstamped-<id8>/` (the shared `dev/` bucket stops
  accumulating; existing `dev/` data is left in place, annotated by convention as pre-fix;
  `<id8>` comes from `AttributionID()`, the attribution-only identifier — never the cache).
- `--baseline` freeze: silent `"dev"/"dev"` manifest → pre-spend refusal; and (round 5) a
  dirty-stamped freeze, previously accepted, → pre-spend refusal with the `dirty` message.

Anything else that changes is a regression, not an intention.

## Testing Strategy

**Unit tests:** knownness truth table (`VersionKnown`/`CommitKnown`/`CommitClean`, incl. the two
cross-field rows: plain-SHA `Commit` + `-dirty` `Version` → not clean, V24; `"dev"` `Version` +
clean `Commit` → clean); `AttributionID`
branch coverage via the `attributionIDFromPath` seam (fixture files in `t.TempDir()`,
rebuild-sensitivity via rewrite + `os.Chtimes`, no faking of `os.Executable`); `bankBucket` table
test; `validateCohortFreeze` branch tests with injected `version.Identity` values (the new
parameter is the injection seam — no global mutation, no build-tag tricks) — five shapes:
unstamped `Version`, unstamped `Commit`, dirty `Commit` (ReadBuildInfo shape), dirty `Version`
over a plain-SHA `Commit` (ldflags shape, V24), and the clean-stamped pass shape (V26).

**Integration tests:** `TestModuleCache_BypassedWhenIdentityAmbiguous` compiles a real module
with a forced ambiguous identity — dirty `version.Commit` (ReadBuildInfo path), dirty
`version.Version` over a plain-SHA `version.Commit` (ldflags path — the round-3 arm, V24), or
unstamped (both exported vars set directly, restored with `defer` — no `sync.Once` seam needed)
— pre-populates the cache directory with a dummy artifact keyed by the ambiguous identity, and
asserts the compile executes anyway (Lookup bypass) and that no new cache files appear (Store
bypass) — round 5; the empty-fresh-dir form left Lookup bypass unverified (mutation row 8c).
Its two caching arms (clean-stamped, and `"dev"` `Version` + clean `Commit`) assert caching
still happens.

**Regression-surface tests:** the fixtures listed in Conflict Surface, enforced by AC-5, AC-7,
AC-8, AC-12 with enumeration floors (every `-run` criterion carries a literal EXPECTED_N and
fails loudly on N=0 — `go test -run` exits 0 on a selector that matches nothing, so the exit
code alone is green before a single test exists).

**Manual verification (worktree, both arms):**
1. `go build -o /tmp/x/ailang ./cmd/ailang` (unstamped): `--version` shows no Commit; a
   `--baseline v1.1-rc1 --verify --dry-run`-class invocation refuses with the `unstamped`
   message; `--bank-by-version` prints an `unstamped-<id8>` output path.
2. Same build with the ldflags recipe from M4's remediation text, run in a COMMITTED-CLEAN tree
   (stamped even in a worktree, executed-verified V17; clean inputs verified V26): freeze
   proceeds; bucket is the release tag; manifest records real values. Expectation check:
   `go version -m` on this binary still shows **0** `vcs.*` lines (V17) — that is the ldflags
   mechanism working as designed, not a failed remediation.
3. Same recipe run in a DIRTY tree (uncommitted changes): `Version` stamps `…-dirty` while
   `Commit` stays a plain SHA (V24/V26), and the freeze refuses with the `dirty` message — the
   remediation text's committed-clean line is what prevents a refusal loop here (round 5).

## Non-Goals

**Not in this doc**, each with rationale:

- **Re-running or re-freezing the banked v1.0 cohort** — this doc changes what *future*
  artifacts record; a published baseline is never rewritten (the manifest's own rule: a
  re-freeze publishes as a NEW baseline id).
- **`eval_results/` being git-ignored** — already handled by convention (decision-bearing
  artifacts are copied to a tracked `design_docs/implemented/` path, as the v1.0 manifest was)
  plus a controller-side gate rule; on review the convention is sufficient for this defect class
  because the artifact that must survive IS tracked — the residual (a reviewer must know to look
  in `design_docs/`, not `eval_results/`) is real but belongs to a docs/gate follow-up queue
  row, not this design.
- **The process half of the root cause** (the mission loop's scratch-build recipe not carrying
  the Makefile's `-ldflags`) — a mission-skill change, routed separately; see Routing note.
- **D-30** (harness↔`ai-check` PATH-resolved version skew) — open human decision, different
  defect; referenced in Related Documents only.
- **Display/service surfaces** (consumers 4–7): `--version`, microrag MCP metadata, prompt/MCP
  `forVersion` — `"dev"` only labels output there; the prompt path's embedded fallback is its
  designed degradation (V10). Residual noted, unchanged here.
- **A startup-time or blanket refusal** — explicitly rejected: `go run`/`go test` builds are
  legitimately unstamped and the whole test suite runs through them.

## Routing note

The controller routes the companion *process* fix separately: the mission verification
protocol's scratch-build recipe should adopt the same `-ldflags` derivation the Makefile uses
(the full recipe was executed end-to-end in this linked worktree and stamped both vars exactly —
V17; V2 says nothing about worktrees, it was measured in the main checkout). That skill
change removes the day-to-day *trigger*; this doc removes the *defect class* (any future
unstamped binary, from any build path, is either refused at the evidence gate or explicitly
marked). The two do not overlap: nothing in this doc edits mission skills, and the skill fix
edits no Go code.

## Timeline

- **Day 1:** M1 (accessor + package's first tests)
- **Day 2:** M2 (cache identity + bypass + tests)
- **Day 2.5:** M3 (bucket marker + tests)
- **Day 3–3.5:** M4 (freeze refusal + CHANGELOG + manual two-arm verification)
- Buffer to 4 days for review churn on the refusal wording and the two-arm manual check.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| Cache-gate cost on hot compile paths | None | `CommitClean()` is a string check on already-loaded `Identity` fields — no `os.Stat`, no hashing, nothing computed on the compile path; the full-content-hash alternative was rejected outright for its measured ~215 ms/process on the default cache-on path (V22), which is why the design declines to cache ambiguous builds rather than paying that cost |
| Stat-metadata aliasing in `AttributionID()` (preserved-mtime restore onto a reused inode) | Low, and scoped to attribution only | `AttributionID()` is never used as a cache key (Consumer 2 gates on `CommitClean()` alone); a collision here means Consumer 3 mislabels a bucket, not that a stale compiled artifact is served. Needs deliberate archive-style restoration to trigger at all; see Consumer 3's "Scope of the `gpt5-6-sol` objection" paragraph |
| Unstamped bucket proliferation in `eval_results/` | Low | Only ad-hoc unstamped runs create them; rotation lanes run stamped binaries; buckets are self-labelling (`unstamped-`) |
| Freeze refusal breaks an existing automation that froze unstamped or dirty | Med | That automation was producing corrupt (unstamped, V15) or unreproducible (dirty, V27) evidence; the refusal message carries the exact rebuild command including the committed-clean requirement; lands pre-spend so nothing is wasted; the strict gate is satisfiable by the doc's own recipe from a committed tree at the same commit (V26), so no freeze is ever impossible — at worst it is one commit/stash away |
| `validateCohortFreeze` signature change ripples through tests | Low | Three existing tests touch it (V16); testing policy is rewrite-not-shim; AC-12 pins retained behaviour |
| A future caller reads `version.Version` raw instead of `Get()` | Med | Accessor is the documented entry point; package doc comment updated in M1 to point decision-bearing consumers at `Identity`/`CommitClean`/`AttributionID`; Conflict Surface names the classification rule |

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Cache correctness no longer depends on deriving an identity for ambiguous builds at all — unstamped and dirty-stamped builds decline to cache instead of aliasing (today: every unstamped build to `"dev"`; every ReadBuildInfo-dirty rebuild to one `-dirty` string; every ldflags-dirty rebuild to the plain commit SHA, V24), and the clean-stamped path is unchanged |
| A2: Replayability | +1 | The cohort manifest becomes actually sufficient to recompute the cohort — round 5 closes the last gap by refusing dirty-stamped freezes, whose bytes no manifest field could reconstruct (V27) |
| A3: Effect Legibility | 0 | No effect-system change |
| A4: Explicit Authority | 0 | No capability change |
| A5: Bounded Verification | 0 | No change |
| A6: Safe Concurrency | 0 | No change |
| A7: Machines First | 0 | No change |
| A8: Minimal Syntax | 0 | No language surface |
| A9: Cost Visibility | +1 | Freeze refusal lands before metered spend; unstamped data is banked attributably instead of silently pooled |
| A10: Composability | 0 | No change |
| A11: Structured Failure | +1 | Silent `"dev"` substitution becomes a typed refusal with remediation, or an explicit marker |
| A12: System Boundary | 0 | No change |

**Net Score: +4** — proceed.

### Hard Violation Check

- [x] A1: no implicit nondeterminism introduced (the cache gate, `CommitClean()`, is a pure
      string check on already-loaded fields; `AttributionID()`'s stat-derived fallback, used only
      for attribution, is a pure function of the executable file's stat identity, memoized, and
      its non-content-addressed, metadata-only nature is named, not hidden)
- [x] A3: no hidden side effects (one explicit stderr warning on cache bypass)
- [x] A4: no ambient authority granted
- [x] A7: optimizes for machine-checkable provenance, not human convenience

## Verification Log

All rows measured by this doc's author (the DESIGNER), 2026-08-23, in
`/Users/voightkampff/.ailang-driver-pin/v1` at `origin/dev` = `96381331a`, go1.26.6
darwin/arm64, unless another directory is named — **except V17 and V22, measured first-party by
the CONTROLLER** in the same worktree at the same HEAD (quorum revision round 1), and **V23**,
measured first-party by the CONTROLLER (quorum revision round 2), transcribed verbatim here, per
the mission rule that a premise is measured, not forwarded. Empty/negative results are paired
with a known-positive control in the same call, scoped to the same path; scopes asserted with
`test -d`.

| # | Claim | Command | Observed |
|---|-------|---------|----------|
| V1 | Pin-worktree build is UNSTAMPED | `go build -o /tmp/i256v/wt-ailang ./cmd/ailang; go version -m /tmp/i256v/wt-ailang \| grep -c 'vcs'` · control `grep -c $'\tbuild\t'` | build rc=0; vcs count **0**; control **10** build settings |
| V2 | Main-checkout build IS stamped | same from `/Users/voightkampff/dev/sunholo-data/ailang` | **4** vcs lines: `vcs=git`, `vcs.revision=db71d2a1638bf1fa379945289fc90fde749819cd`, `vcs.time=2026-08-23T01:30:17Z`, `vcs.modified=true` |
| V3 | `Version` has NO runtime fallback; `Commit` has | `/tmp/i256v/main-ailang --version` (the stamped arm) | `AILANG dev` + `Commit: db71d2a` + `Full: db71d2a…-dirty` — stamping fixed Commit only |
| V4 | `-buildvcs=true` neither rescues nor errors | `go build -buildvcs=true -o /tmp/i256v/wt-vcs-ailang ./cmd/ailang`; `go version -m … \| grep -c 'vcs\.'` | rc=**0**, binary produced, **0** `vcs.` lines, `--version` = `AILANG dev`, no Commit line. Instrument note: `grep -c 'vcs'` (undotted) returns **1** — the match is the output header containing the binary's own filename (`/tmp/i256v/wt-vcs-ailang: go1.26.6`); the dotted pattern is the correct instrument |
| V5 | It is the WORKTREE, not the detached HEAD | same build from `/Users/voightkampff/dev/sunholo-data/.wt-iter216-record` (linked worktree ON branch `docs/mission-iter216-record`) | rc=0; **0** `vcs.` lines; control **10** build settings |
| V6 | Linked worktree `.git` is a FILE (mechanism) | `cat /Users/voightkampff/.ailang-driver-pin/v1/.git` | `gitdir: /Users/voightkampff/dev/sunholo-data/ailang/.git/worktrees/v1` |
| V7 | `internal/version` = one 56-line file; `init()` populates Commit/BuildTime only, never Version; no knownness/AttributionID API exists (negative-existence) | full read of `internal/version/version.go`; `ls internal/version/` | `version.go` only; `init()` switches on `vcs.revision`/`vcs.time`/`vcs.modified`; `Version` never reassigned; no accessor/exe-hash API |
| V8 | Consumer 2 + the "guard the helper, miss the call site" precedent | read `internal/pipeline/pipeline_module.go:260-280`, `internal/pipeline/cache_key.go:1-50` | `:276` `ModuleCacheKey(version.Commit, sourceContent, depDigests)`; `cacheKeyVersion = "v3"`; v1→v3 comments cite "same-version dev/worktree build" as FORMAT-bump rationale, identity component unguarded |
| V9 | Consumer 3: `releaseTag("dev") == "dev"`, banks shared bucket | read `cmd/ailang/eval_suite.go:30-50,185-197`; `head -30 cmd/ailang/release_tag_test.go` | `:191` `ver := releaseTag(version.Version)`; doc comment + `TestReleaseTag` pin `"dev" -> "dev"` (`// non-ldflags build unchanged`) |
| V10 | Full importer surface = **7** non-test files, 2 of them ALIASED (invisible to symbol grep) | `test -d cmd internal` (rc=0); `grep -rln 'sunholo-data/ailang/internal/version"' --include='*.go' cmd/ internal/ tools/ \| grep -v _test.go` · negative control `grep -rn 'version\.ZzNoSuchSymbolI256' …` → 0 | 7 files: `eval_suite_manifest.go`, `pipeline_module.go`, `eval_suite.go`, `main.go`, `prompt.go`, `mcp_status.go`, `cmd/ailang-microrag-mcp/main.go`. `prompt.go:10`/`mcp_status.go:22` import as `versionpkg` and pass `versionpkg.Version` into `LoadPromptFresh` (`prompt.go:94`, `mcp_status.go:84`) — a `grep 'version\.Version'` cannot see them. Symbol grep reproduced: `version\.Commit` → **5** hits (2 are comments, 1 the main.go alias) |
| V11 | `internal/version` has ZERO test files | `go test ./internal/version/... -count=1` · control (same instrument, package with tests): `go test ./internal/pipeline/ -count=1 -run 'TestModuleCacheKey' -v` | `? … [no test files]`, rc=**0** · control: **7** top-level `=== RUN` lines, rc=0 |
| V12 | Manifest written ONLY on freeze path; bucket mutated once, read by all downstream stages | read `eval_suite.go:510-528` (`if baselineSet { freezeCohortManifest(*outputDir, …) }`); `grep -n 'outputDir' cmd/ailang/eval_suite.go` | freeze gated on `baselineSet`; `*outputDir` set at `:187-197`, read at `:521,:575,:626-632,:690,:764,:770` |
| V13 | Consumer 1 writes raw vars into the manifest | read `cmd/ailang/eval_suite_manifest.go:187-230` | `:212` `AILANGVersion: version.Version`, `:213` `GitCommit: version.Commit` |
| V14 | Existing cache-key test surface (regression floor for AC-5) | `grep -rn 'func Test' internal/pipeline/cache_key_test.go` | exactly 7: `TestModuleCacheKey_{Deterministic,DifferentSource,DifferentVersion,DifferentDep,DepOrderIndependent,CommitChange,NoDeps}` |
| V15 | The live v1.0 artifact carries the defect | `python3` json read of `design_docs/implemented/v1_0_0/m-cost-per-success-kpi-baseline-v1.0-cohort-manifest.json` | `ailang_version='dev'`, `git_commit='dev'`; controls populated: `frozen_at='2026-08-22T19:56:27.504667Z'`, `chain_id='219b1fbc-…'`, `cohort_hash='526fe724…'` |
| V16 | Freeze-gate seam + existing freeze/manifest tests (fixtures for AC-11/12; behaviour claims from reading test bodies' names + `validateCohortFreeze` body) | read `cmd/ailang/eval_suite_cohort.go:70-100`; `grep -rn 'func Test' cmd/ailang/eval_suite_cohort_test.go cmd/ailang/eval_suite_manifest_test.go` | `validateCohortFreeze(baselineSet, baselineID, verify)` runs "before the rig lock, before the API-key check"; 11 cohort tests + 16 manifest tests exist, incl. `TestValidateCohortFreeze`, `…_VerifyMessageExplainsWhy`, `TestEvalSuiteBaselineRequiresVerify_CLI`, `TestCohortHash_StableAcrossBuilds` |
| V17 | **[CONTROLLER, executed]** The ldflags remediation works when actually run in a linked worktree — AND it does NOT restore `vcs.*` build settings | From `/Users/voightkampff/.ailang-driver-pin/v1` (detached at `origin/dev` = `96381331a3e0dd0682873553b79d91c7c8e27f5f`): `VERSION=$(git describe --tags --always --dirty); COMMIT=$(git rev-parse HEAD); BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S'); go build -ldflags "-X …version.Version=$VERSION -X …version.Commit=$COMMIT -X …version.BuildTime=$BUILD_TIME" -o /tmp/i256_ldf/ailang ./cmd/ailang; /tmp/i256_ldf/ailang --version; go version -m /tmp/i256_ldf/ailang \| grep -c 'vcs\.'` | build rc=**0**; inputs `VERSION=v0.33.1-220-g96381331a`, `COMMIT=96381331a3e0dd0682873553b79d91c7c8e27f5f`; `--version` prints `AILANG v0.33.1-220-g96381331a`, `Commit: 9638133`, `Full: 96381331a3e0dd0682873553b79d91c7c8e27f5f`, `Built: 2026-08-23_03:56:43` — **both vars stamped, matching `git describe`/`git rev-parse HEAD` exactly**. `vcs\.` lines = **0** (control: **11** `build` settings vs **10** for the un-stamped worktree build — the extra one is the recorded `-ldflags`): ldflags sets the package vars directly, it does not restore Go's VCS stamping; sufficient because all three decision-bearing consumers read the vars, never `debug.ReadBuildInfo()`. Supersedes the round-0 V17, which only *read* `Makefile:26-30` and leaned on V2 — a main-checkout measurement that says nothing about worktrees |
| V18 | No existing design doc covers build provenance (duplicate gate; negative + positive control in same call) | `grep -rln 'buildvcs\|vcs\.revision\|unstamped\|build provenance\|ldflags' design_docs/planned/ design_docs/implemented/v1_0_0/ design_docs/implemented/v1_1_0/` · control: `grep -rln 'M-EVAL-VERSION-BANKING\|bank-by-version' design_docs/` | first grep: **0** files; control: **5+** files incl. `design_docs/implemented/v0_26_1/m-eval-version-banking.md` (instrument sees positives in the same tree) |
| V19 | No existing exe-hash helper to reuse (negative-existence for the M1 API) | `grep -rn 'os\.Executable' --include='*.go' internal/ cmd/ \| grep -v _test.go` (positive: 9+ hits, all path-resolution) · `grep -rn 'sha256' --include='*.go' internal/ \| grep -i 'file\|exec'` | `os.Executable` used only for path resolution (module/loader/help/replay/…); no site hashes the executable |
| V20 | Baselined rc=0 commands on pristine dev (everything used in AC-1..AC-14) | `go build ./internal/version/... ; go build ./internal/pipeline/... ; go build ./cmd/ailang ; go vet ./internal/version/... ; go vet ./cmd/ailang ; gofmt -l internal/version internal/pipeline cmd/ailang ; make check-changelog ; make check-file-sizes ; make check-boundaries ; go test ./internal/pipeline/ -count=1 ; go test ./cmd/ailang -count=1 -run '^TestReleaseTag$' -v` | ALL rc=0; `gofmt -l` → **0** lines; full pipeline suite rc=0; `TestReleaseTag`: 1 `=== RUN`, `ok` |
| V21 | `go build ./...` is rc=1 on pristine dev — BANNED from acceptance criteria | `go build ./... ; echo $?` | rc=**1** — `# github.com/sunholo-data/ailang/cmd/wasm … function main is undeclared in the main package` |
| V22 | **[CONTROLLER, executed]** Full-executable hashing is a ~215 ms/process tax on the DEFAULT path — the reason the cache no longer falls back to a computed identity at all (it declines to cache instead) | binary size via `ls -l $(which ailang)`; `shasum -a 256` over it, 3 runs, with a 2-byte-file control; cache-default read of `internal/pipeline/pipeline_module.go` | `ailang` binary = **96,201,490 bytes**; `shasum -a 256`: **232 / 215 / 214 ms** (control: 2-byte file **29 ms** — the figure is hash+IO work, not process startup); module caching is ON BY DEFAULT (`if !cfg.NoCache`, `pipeline_module.go:230` — the directive cited :229; DESIGNER re-verified the line as **:230** at this HEAD). With `sync.Once` that is ~215 ms added to every unstamped `ailang` process that compiles any module — the state of every worktree/`go test` binary this mission builds — and Objection 1 would have moved stamped-dirty builds onto the same hash; controller measurement V23 (round 2) priced the actual cache benefit against this cost and selected bypass over any substitute identity for ambiguous builds |
| V23 | **[CONTROLLER, executed]** Module-cache's measured benefit is small relative to the rejected hash cost — the reason round-2 selected cache bypass over any substitute identity for ambiguous builds | Using `/tmp/i256_ldf/ailang` (the ldflags-stamped worktree build, `v0.33.1-220-g96381331a`, V17): three runs each of `ailang run examples/asset_path.ail` and `ailang run examples/regex_log_orchestration.ail` (heaviest example, 6 imports) with cache ON (default) vs OFF (`AILANG_NO_CACHE=1` — the switch at `cmd/ailang/main_run_exec.go:240`) | `examples/asset_path.ail`: **33/34/33 ms** ON vs **46/46/44 ms** OFF (≈12 ms benefit); `examples/regex_log_orchestration.ail`: **39/38/39 ms** ON vs **39/39/38 ms** OFF (no measurable difference). Against the ~215 ms/process content-hash cost (V22), paying to protect a ≤12 ms saving is strictly worse than not caching at all — this measurement, quoted in round 2, selected bypass over any substitute cache identity |
| V24 | **[CONTROLLER, executed — the row `gpt5-6-sol` demanded in round 3; RE-DERIVED FIRST-PARTY in round 4 at the current HEAD `ad6d08050`, superseding the round-3 numbers taken at `db71d2a16`; reproduced read-only, command-for-command, by the round-4 reviser in the same session]** The two stamping paths put `-dirty` on **DIFFERENT FIELDS**, so a `Commit`-only dirty check is blind to every `make`/ldflags-stamped dirty build | Read-only evaluation of the exact `Makefile:27-28` inputs (`VERSION := $(shell git describe --tags --always --dirty …)` at :27, `COMMIT := $(shell git rev-parse HEAD …)` at :28 — line numbers re-checked by `grep -n` this round) in a genuinely dirty tree — the main checkout `/Users/voightkampff/dev/sunholo-data/ailang` (control: `git status --porcelain` = **6** lines — **3** modified tracked + **3** untracked) — against the clean pin worktree `/Users/voightkampff/.ailang-driver-pin/v1` (detached at `origin/dev` = `ad6d08050b5fabc42f8510780466295646000a05`, porcelain **0** lines) as the control arm; ReadBuildInfo arm from the `internal/version/version.go` `init()` re-read (V25) plus the live dirty-tree binary of V2/V3 | **ldflags path, dirty tree:** `git describe --tags --always --dirty` → `v0.33.1-221-gad6d08050-dirty` (dirtiness on **Version**), `git rev-parse HEAD` → `ad6d08050b5fabc42f8510780466295646000a05` (**plain SHA, no `-dirty`**). **Control, clean pin worktree:** `v0.33.1-221-gad6d08050` (no `-dirty`) / the identical SHA `ad6d08050b5fabc42f8510780466295646000a05` — neither field dirty, same commit. **ReadBuildInfo path:** `-dirty` is appended to **`Commit`** only, never `Version` (mechanism: V25; observed live in V2/V3's dirty-tree binary — `Full: …-dirty`). So `-dirty` reaches `Commit` ONLY via `ReadBuildInfo`, and reaches `Version` ONLY via ldflags. `CommitClean()` as specified in round 2 therefore returned **true** for every `make`-installed dirty binary — the exact population it was written to exclude; the round-4 predicate adds the `Version` arm to close exactly this |
| V25 | **[ROUND-4 REVISER, executed]** The `ReadBuildInfo` fallback appends `-dirty` to `Commit` — never to `Version`, which has NO runtime assignment at all (mechanism arm of V24; re-derived this iteration, confirming V7) | Full re-read: `cat internal/version/version.go` in the pin worktree at `ad6d08050` | `init()` returns early if `Commit != "dev"` (ldflags wins); otherwise it iterates `debug.ReadBuildInfo()` settings: `vcs.revision` → `Commit = s.Value`; `vcs.modified` → `if s.Value == "true" && Commit != "dev" { Commit = Commit + "-dirty" }`. `Version` is assigned exactly once, at its `var` declaration (`"dev"`); no statement in the package ever reassigns it |
| V26 | **[ROUND-5 REVISER, executed]** The strict freeze gate (`VersionKnown() && CommitClean()`) is SATISFIABLE by the doc's own prescribed recipe from a committed tree — the Makefile's inputs stamp clean values exactly when `git status --porcelain` is empty, at the same commit a dirty tree sits on | Pin worktree at `git rev-parse HEAD` = `ad6d08050b5fabc42f8510780466295646000a05`: `git describe --tags --always` · `git describe --tags --always --dirty` · `git status --porcelain`; contrast arm: the same three commands in the main checkout `/Users/voightkampff/dev/sunholo-data/ailang` | Pin worktree: base describe **`v0.33.1-221-gad6d08050`** (no `-dirty`); the `--dirty` arm returns `v0.33.1-221-gad6d08050-dirty` because porcelain = **1** line — ` M design_docs/planned/m-cohort-manifest-build-provenance.md`, i.e. the only dirty file is this doc's own in-flight round-5 revision. The `--dirty` flag only ever APPENDS a suffix to the same base string, so the base value IS what a committed tree stamps. Main checkout at the SAME commit: `v0.33.1-221-gad6d08050-dirty`, porcelain **6** lines; `git rev-parse HEAD` = the identical plain SHA in both trees — dirtiness never reaches `Commit` on the ldflags path (re-confirms V24). So the strict gate never makes freezing impossible: the clean state is always one commit/stash away at the same commit. (Round-5 framing correction, named per protocol: the controller's clean-arm measurement — porcelain 0, clean describe — was true when taken but is no longer reproducible in-place, because the file being revised is itself the worktree's one dirty file) |
| V27 | **[ROUND-5 REVISER, executed]** The live release-evidence manifest has NO source-diff and NO compiler-content-identity field — a `-dirty` stamp would be visible but unreconstructable — and `cohort_hash` is over cohort COMPOSITION, not compiler bytes, so it cannot stand in for one | `jq -r 'keys[]'`, `jq -r 'keys \| length'`, `jq -r '.cohort_hash'` on `design_docs/implemented/v1_0_0/m-cost-per-success-kpi-baseline-v1.0-cohort-manifest.json`; preimage read: `sed -n 118,145p cmd/ailang/eval_suite_manifest.go` | Exactly **20** top-level keys: `ailang_version, baseline_id, benchmarks, chain_id, cohort_hash, conditions, eval_mode, executors, frozen_at, git_commit, languages, model_suite, models, prompt_version, run_window, seed, source_ref_prefix, trials, verify, verify_timeout` — none is a diff or a content hash of the compiler. `cohort_hash` = `526fe7240112bb16238f91a077487dc9fbf0be5c0fbca723c2a21e6a52bd0f40`; its preimage `cohortIdentity` = `{eval_mode, languages, conditions, models, benchmarks, seed, prompt_version, trials}`, with the source comment stating `frozen_at / git_commit / chain_id / ailang_version / baseline_id are all absent` — a deliberate composition identity that excludes build provenance entirely |
| V28 | **[ROUND-5 REVISER, executed]** `releaseTag` collapses `-dirty` into the release bucket DELIBERATELY — pinned by an existing test row — so Consumer 3's `VersionKnown()` arm pooling dirty builds into the release bucket is designed width, not an unexamined gap (systemic-sweep verdict: SOUND) | `grep -n 'func releaseTag' -A 3 cmd/ailang/eval_suite.go`; `sed -n 1,25p cmd/ailang/release_tag_test.go` | `releaseTag` = `strings.TrimSuffix(gitDescribeSuffix.ReplaceAllString(v, ""), "-dirty")` (`eval_suite.go:42-44`); `TestReleaseTag` carries the row `"v0.26.0-26-g9249a66bf-dirty": "v0.26.0"` commented `// dirty dev build -> its release`, beside `"v0.26.0-26-g9249a66bf": "v0.26.0"` — dirty-tree drift pools into the same bucket as 26 commits of post-release drift, which the per-release cadence pools BY DESIGN (m-eval-version-banking). Banking is attribution, not release evidence: this width matches its purpose, unlike the freeze gate's |

### Quorum revision log — round 1 (2026-08-23)

Round-0 verdict: **BLOCKED**, full-strength (both external reviewers present + controller; 3/3
reject). Artifact:
`.ailang/state/mission-quorum/m-cohort-manifest-build-provenance-2026-08-23T03-54-48Z.json`.
The design direction (knowness-aware accessor + per-consumer refuse/degrade/marker) was accepted
by all three and is unchanged. Fixes applied:

| Objection | Verdict on it | Fix applied in this revision |
|-----------|---------------|------------------------------|
| **O1** (`gemini-3-1-pro`): `BuildID()` returning `-dirty` commits aliases different compiler bytes under one cache identity — Failure Class 1 for stamped dirty builds | Accepted — the round's most valuable finding | At the time, `BuildID()`'s commit branch was made to require `CommitKnown && !strings.HasSuffix(Commit, "-dirty")`, with dirty commits taking a stat-derived fallback branch. The consumer-1-vs-consumer-2 distinction (dirty = acceptable *evidence* via `Identity.CommitKnown()`, unsafe *bytes-identity* for cache keys) was stated in the API section. `TestBuildIDPrefersCommit` (since renamed `TestAttributionIDPrefersCommit` — round 2, see below) gained a mandatory `-dirty` row (AC-2 note, mutation row 4). Success Metrics, Consumer 2, Conflict Surface, and Axiom A1 were updated to carry the intended dirty-stamped cache-identity change. **Superseded in round 2**: the cache no longer derives any substitute identity for a dirty commit — it bypasses instead; `BuildID()` was renamed `AttributionID()` and is no longer part of the cache story at all |
| **O2** (controller, measured): the full-executable sha256 fallback costs ~215 ms per process on the default cache-on path (V22), made worse by O1 | Accepted — resolved jointly with O1 | At the time, the fallback was made a stat-derived id (`"exe-" + hex(sha256(dev:inode:size:mtimeNano))[:12]` or equivalent), `os.Stat`-only, memoized, with four binding requirements stated in the API section (cross-process determinism, rebuild sensitivity, stat-not-read cost, over-invalidation-only errors) plus a named residual (metadata aliasing under archive-style restore). **Superseded in round 2**: round-2's own measurement (V23) showed the cache benefit this fallback was protecting is only 0–12 ms, so full-content hashing was never the real alternative on the table — cache bypass is, and that is what the stat-derived id was replaced by on the caching surface. It survives, unchanged in mechanism, only as `AttributionID()`'s fallback for Consumer 3's attribution bucket, where the four requirements reduce to three (the over-invalidation requirement was cache-specific and no longer applies) |
| **O3** (`gpt5-6-sol`): round-0 V17 was inferential — the ldflags remediation had never been executed in a linked worktree, and prose leaned on V2 (a main-checkout measurement) | Accepted — controller executed the reviewer's command; it **passes** | V17 replaced with the controller's executed measurement (transcribed verbatim, scope: linked worktree, detached at `96381331a`), including the unanticipated finding that ldflags does **not** restore `vcs.*` build settings (0 `vcs\.` lines; 11 vs 10 `build` settings) and why that is sufficient — now stated in Consumer 1's remediation section so `go version -m` is not misread as remediation failure. V2-implies-worktree prose corrected in the Routing note |

No milestones added or removed; the milestone shape, scope-outs, and the `go build ./...` ban
all stand as in round 0.

### Quorum revision log — round 2 (2026-08-23)

Round-2 verdict: **BLOCKED**. `gpt5-6-sol` was **ABSENT on `budget`** in the round-2 quorum run
and was restored by a separate `ailang design-review --max-cost-usd 0.30` run (cost
**$0.09095**, verdict **reject**) — so the round was **not** decided at N−1; its objection is a
first-class blocking vote, not a courtesy read.

| Objection | Quoted | Controller measurement | Fix applied in this revision |
|-----------|--------|--------------------------|-------------------------------|
| **`gpt5-6-sol`**: the round-1 stat-derived id is unsafe as a **cache identity** — the doc simultaneously claims it "errs toward over-invalidation, never under" and then names a residual where different executable bytes keep the same stat identity; those two statements contradict | "Replace the cache fallback with a correctness-preserving policy: clean known commit -> commit key; unknown or dirty identity -> either compute and memoize a content hash of the executable, or bypass cache with one explicit warning. Do not use stat metadata as the cache identity. … If hashing cost remains unacceptable, route unstamped/dirty builds to cache bypass until build-time provenance is embedded." | Full-executable sha256: **232 / 215 / 214 ms** per process on the 96,201,490-byte binary (control: a 2-byte file hashes in 29 ms, V22); the module cache's whole measured benefit on the examples corpus with a stamped build is **0–12 ms** (`examples/asset_path.ail`: 33/34/33 ms ON vs 46/46/44 ms OFF ≈ 12 ms; `examples/regex_log_orchestration.ail`, 6 imports: 39/38/39 ms ON vs 39/39/38 ms OFF — no measurable difference; V23) | The reviewer's own conditional selects **cache bypass**: paying ~215 ms to protect a ≤12 ms saving is strictly worse than not caching at all. `Identity.CommitClean()` now gates the cache; unstamped or stamped-dirty declines to cache entirely (no Lookup, no Store, one stderr warning). `BuildID()` is removed from the cache story; it survives only as `AttributionID()` for Consumer 3 (eval-output bucketing), explicitly documented as NOT a correctness identity and never usable as a cache key |
| **`gemini-3-1-pro`**: `TestModuleCache_BypassedWhenIdentityUnknown` is un-implementable — `BuildID()` is memoized with `sync.Once` and has no cross-package test seam, so an `internal/pipeline` test cannot force the bypass branch | (testability objection on the round-1 design) | — | Dissolves under the bypass design rather than needing a seam: the bypass branch is driven by `version.Commit`, an exported package-level `var`, so an `internal/pipeline` test sets it directly (e.g. `"abc1234-dirty"` and `"dev"`) and restores it with `defer` — no `sync.Once`, no exported override helper, no injected function. Test renamed `TestModuleCache_BypassedWhenIdentityAmbiguous`, with dirty-stamped and unstamped arms plus a clean-stamped arm asserting caching still happens |

Fixes were applied by the **controller under the narrow-refinement carve-out** (Fable-diet
compliance: the unit is one bounded DOC, not one bounded RUN) — not by commissioning a third
designer run. No milestones added or removed; the milestone shape, scope-outs, and the
`go build ./...` ban all stand as in round 0 and round 1.

### Quorum revision log — round 3 (2026-08-23) — **BLOCKED**

Round 3 was a CONFIRMING quorum on the controller's carve-out revision — more conservative than
the carve-out requires (it permits routing straight to sprint-planner after a bounded 2nd
revision). Running it is what caught the defect below. **Both external reviewers PRESENT**
(`absent_reviewers` empty — a full-strength 2/2 reject, not the degraded pass the N−1 rule warns
about); controller verdict `pass`. Artifact:
`.ailang/state/mission-quorum/m-cohort-manifest-build-provenance-2026-08-23T04-27-47Z.json`.

**Both reviewers independently named the SAME defect, and it is in the predicate the CONTROLLER
authored in round 2 — not in anything the designer wrote.**

| objection | verdict |
|---|---|
| **`gpt5-6-sol`**: "The cache bypass does not reliably detect the document's claimed dirty-stamped case. `CommitClean()` checks only whether `Commit` ends in `-dirty`, but the documented ldflags/Makefile contract stamps `Commit=$(git rev-parse HEAD)`, which is a plain SHA even for a dirty tree; dirtiness is instead carried by `Version=$(git describe --tags --always --dirty)`. Such a dirty ldflags-built binary will therefore pass `CommitClean()` and continue using the shared commit cache key, leaving Failure Class 1 unresolved." | **REAL — measured and confirmed, see V24** |
| **`gemini-3-1-pro`**: same defect, independently — "the doc's remediation command (V17) and the Makefile derive `Commit` via `git rev-parse HEAD`, which outputs a raw SHA and never appends `-dirty`. Consequently, dirty stamped builds will evaluate `CommitClean() == true` and incorrectly cache their ambiguous compiler artifacts under the pure SHA, reopening Failure Class 1." | **REAL** |

`gpt5-6-sol`'s *catch* was that the round-2 claim ("`make`-installed dirty builds have an
`abc1234-dirty` Commit") was **never verified** and that no verification row builds from a dirty
tree. Correct: the evidence to refute it was sitting in this doc's own V2 and V17 rows the whole
time — V2 measured `Commit`-carries-`-dirty` on the **ReadBuildInfo** path, V17 measured the
**ldflags** path in a **clean** worktree — and nobody joined them. The controller measured it
(V24) rather than forwarding the objection.

**NOT APPLIED IN THIS ITERATION — the doc is left BLOCKED, deliberately.** The carve-out permits
ONE bounded controller revision and that was round 2. Rounds 2 and 3 each blocked on a defect
introduced by the *previous* fix, and round 3's was the controller's own; applying a third
controller revision unreviewed is precisely what round 3 just punished. The remaining work is
also not pure transcription — `gpt5-6-sol` additionally asks for "explicit handling documented for
`Version == \"dev\"` plus a clean runtime-stamped commit", i.e. the plain-`go build`-in-a-real-
`.git`-directory case, where `Version` is unknown while `Commit` is a clean real SHA. That case
should probably still cache, but it needs stating, and stating it is a (small) design decision.

**Resume — bounded, unanimous, and machine-checkable.** Both reviewers gave the same verbatim
fix; the next iteration applies it and re-quorums ONCE:

1. Widen the cache predicate to cover **both** stamping paths, per `gemini-3-1-pro`'s text:
   `CommitClean() := CommitKnown() && !strings.HasSuffix(i.Commit, "-dirty") && !strings.HasSuffix(i.Version, "-dirty")`
   (`gpt5-6-sol` proposes the same under the name `CacheCommitKnown()`; the predicate is identical).
2. Document the `Version == "dev"` + clean-`Commit` case explicitly.
3. Add V24 (already present) as the dirty-tree verification row, and add the test arms both
   reviewers asked for: a plain-SHA `Commit` with a `-dirty` `Version` **must** select bypass.
4. Re-quorum once. Nothing here is blocked on a human, a lane, or a clock — see the mission log.

### Quorum revision log — round 4 (2026-08-23)

Round 4 applies the round-3 resume — items 1–3, exactly and only — via a fresh designer run
(iteration 257), not a second controller carve-out (rounds 2 and 3 each blocked on a defect
introduced by the previous round's fix, and round 3's was the controller's own). The resume's
step 4 (re-quorum ONCE) is the controller's and is pending: **quorum has NOT passed.** V24 was
re-derived first-party at the current HEAD (`ad6d08050`) rather than carried forward from
`db71d2a16`. Both round-3 objections name the same defect, so one predicate fix discharges both;
`gpt5-6-sol`'s additional ask is handled as its own row.

| Objection (verbatim, from the round-3 table) | What changed | Where |
|---|---|---|
| **`gpt5-6-sol`**: "The cache bypass does not reliably detect the document's claimed dirty-stamped case. `CommitClean()` checks only whether `Commit` ends in `-dirty`, but the documented ldflags/Makefile contract stamps `Commit=$(git rev-parse HEAD)`, which is a plain SHA even for a dirty tree; dirtiness is instead carried by `Version=$(git describe --tags --always --dirty)`. Such a dirty ldflags-built binary will therefore pass `CommitClean()` and continue using the shared commit cache key, leaving Failure Class 1 unresolved." | Predicate widened to the reviewers' verbatim text: `CommitClean() := CommitKnown() && !strings.HasSuffix(i.Commit, "-dirty") && !strings.HasSuffix(i.Version, "-dirty")`. Name kept `CommitClean()` (`gpt5-6-sol`'s `CacheCommitKnown()` is the identical predicate). `CommitKnown()` deliberately UNCHANGED — the evidence-knownness vs bytes-identity distinction is settled design, and a dirty commit remains acceptable *evidence* for Consumer 1; only the cache predicate widens | API section (predicate line + rewritten both-fields paragraph); High-Impact Decisions (cache-gate row); Design Freeze; Success Metrics; Consumer 2 (snippet comment + rewritten dirty-stamped bullet); Conflict Surface Failure Class 1; "What deliberately changes"; Testing Strategy; Axiom A1; V24 (re-derived) |
| **`gemini-3-1-pro`**: "the doc's remediation command (V17) and the Makefile derive `Commit` via `git rev-parse HEAD`, which outputs a raw SHA and never appends `-dirty`. Consequently, dirty stamped builds will evaluate `CommitClean() == true` and incorrectly cache their ambiguous compiler artifacts under the pure SHA, reopening Failure Class 1." | Same defect, found independently — same predicate fix as the row above. Additionally, the test arms both reviewers asked for: `TestIdentityKnownness` gains two cross-field rows (plain-SHA `Commit` + `-dirty` `Version` → false — fails against the round-2/3 predicate; `"dev"` `Version` + clean `Commit` → true); `TestModuleCache_BypassedWhenIdentityAmbiguous` grows from three to five arms (new: ldflags-dirty bypass arm asserting the cache dir stays empty; `"dev"`-`Version`+clean-`Commit` arm asserting caching still happens); mutation row **8b** pins the `Version` arm by mutating the predicate back to the round-2/3 form and naming the tests that must die | AC-2; AC-6; Mutation table rows 8 (note) + 8b (new); Testing Strategy; Files to Modify LOC (~150→~170, ~80→~100) |
| **`gpt5-6-sol`** (additional ask, quoted in the round-3 narrative): "explicit handling documented for `Version == \"dev\"` plus a clean runtime-stamped commit" | Decided (the one small design decision in this round): this case **still caches**. `!strings.HasSuffix("dev", "-dirty")` is true and `Commit` is a clean real SHA; on the only in-contract path producing this shape (plain `go build` in a real `.git` directory) dirtiness would have landed on `Commit` itself (V25), so a clean `Commit` is a sufficient bytes-identity and an unknown `Version` says nothing about dirtiness. The asymmetry is stated out loud — unknown `Version` is treated as "not dirty"; plain-`go build` dirty trees are caught by the `Commit` arm, `make`/ldflags dirty trees by the `Version` arm; the arms cover different populations and neither alone suffices. The off-contract Commit-only hand-stamp evasion is named as a residual | API section (new "`Version == \"dev\"` + clean-`Commit`" paragraph); Consumer 2 (new bullet); new High-Impact Decisions row (attackable); AC-2 + AC-6 caching arms |

**NOTED-NOT-CHANGED** (observed while editing; deliberately left alone — outside the round-3
mandate; listed for the round-4 reviewers):

1. `AttributionID()`'s commit-preference branch keeps the narrower condition
   `CommitKnown && !strings.HasSuffix(Commit, "-dirty")` — NOT widened. Its only specified
   caller (Consumer 3's `bankBucket`) reaches it only when `!VersionKnown()`, i.e.
   `Version == "dev"`, where a `Version` arm is vacuous (`"dev"` never ends in `-dirty`);
   widening it would imply a reachable behaviour change that does not exist. A clarifying
   comment was added at the API-comment branch so the two predicates are not confused.
2. No M1 mutation row enumerates a whole-`return true` mutant of `CommitClean()` (rows 1–2
   cover only `VersionKnown`/`CommitKnown`). Such a mutant dies to `TestIdentityKnownness`'s
   dirty rows (pre-existing and new), so coverage exists; only the table enumeration is
   missing. Left as-is.
3. Formatting only: the V24 row was previously separated from the Verification Log table by a
   blank line, rendering as a header-less table fragment; the blank line was removed while
   updating V24 (no content decision).

### Quorum revision log — round 5 (2026-08-23)

Round-4 verdict: **BLOCKED 2/2**. Both round-4 objections target code paths round 4 never
touched — they are pre-existing holes rounds 1–3 walked past. Four consecutive rounds have now
each found one instance of **one class**: *a gate or assertion whose satisfying-state set is
wider than the purpose it is cited for* (round 3: `CommitClean()` blind to ldflags-dirty;
round 4a: the freeze gate accepting dirty; round 4b: AC-6's empty-dir assertion blind to
`Lookup`). Round 5 therefore ran a **systemic sweep** (CLAUDE.md §3: one unified fix, not
case-by-case patches) over every gate/predicate and every acceptance assertion in the doc
BEFORE applying the two named fixes. **The sweep found two additional TOO-WIDE instances the
reviewers had not named** (AC-9's unstamped-marker assertion; AC-13's changelog assertion) plus
one consequential hole the round-5 fix itself would have introduced (the remediation text would
loop dirty-tree operators back into refusal) — all fixed in this round.

#### Per-objection table

| Reviewer | Objection (verbatim) | What changed | Where |
|---|---|---|---|
| `gpt5-6-sol` | "The freeze gate deliberately accepts dirty-stamped builds even though the manifest records no source diff or compiler-content identity. A commit SHA plus a visible '-dirty' suffix cannot let a reviewer reproduce the compiler bytes, contradicting the document's claim that the cohort manifest becomes sufficient to recompute the cohort and violating deterministic release evidence." (Catch: "Consumer 1 uses VersionKnown()/CommitKnown() rather than a clean-build predicate, so both ReadBuildInfo-dirty and ldflags-dirty binaries may freeze authoritative evidence with unrecoverable local modifications.") | **Alternative (a) of the reviewer's proposed fix selected** (see decision below): Consumer 1's gate is now `VersionKnown() && CommitClean()`, with a third refusal branch for dirty-stamped identity on either field, distinct `dirty` message text, and pre-spend tests for BOTH dirty stamping shapes (`Commit="abc1234-dirty"`; `Version="v0.34.0-dirty"` + plain-SHA `Commit`). Remediation text gains the committed-clean requirement so its own output cannot re-fail the gate. Both premises measured first-party: the manifest's 20 keys carry no diff/content identity and `cohort_hash` is composition-only (V27); the strict gate is satisfiable by the doc's own recipe from a committed tree (V26) | Success Metrics; High-Impact Decisions (freeze row); Design Freeze; API section (`CommitKnown()` vs `CommitClean()` rewritten); Consumer 1 / M4 (branch 3 + message contract + remediation); AC-11 (EXPECTED_N 3→5); mutation rows 14 (updated) + 15 (new); Conflict Surface "What deliberately changes"; Manual verification arm 3 (new); Risks; Axiom A2; V26/V27 |
| `gemini-3-1-pro` | "AC-6 contains a critical testing gap: asserting the cache directory stays EMPTY after a compile proves that 'Store' was bypassed, but it fails to prove 'Lookup' was bypassed. In a fresh test directory, an implementation that incorrectly performs a 'Lookup' on an ambiguous key will simply miss, skip the 'Store', and leave the directory empty, passing the test while leaving the compiler vulnerable to reading stale blobs (Failure Class 1)." | The reviewer's proposed text applied to AC-6 verbatim: the three bypass arms must pre-populate the cache directory with a dummy artifact keyed by the ambiguous identity, assert the compiler executes anyway (Lookup bypass), and assert no new files are written (Store bypass). New mutation row **8c** pins the contrast: a mutant that gates only `Store` and still performs `Lookup` is killed by the strengthened arms (the pre-populated dummy makes its `Lookup` HIT, failing the compiles-anyway assertion) and is NOT killed by the old empty-directory assertion (fresh dir → miss → no Store → empty → green) — that contrast is what proves the strengthening is load-bearing | AC-6; mutation rows 8 + 8b (assertion wording) + 8c (new); Testing Strategy (integration paragraph); Files to Modify (`compiler_identity_test.go` ~100→~120 LOC) |

#### The (a)-vs-(b) decision on dirty freezes

`gpt5-6-sol` offered two alternatives: **(a)** refuse unless `VersionKnown()`, `CommitKnown()`,
and neither field ends in `-dirty`; **(b)** support dirty freezes by extending the manifest with
a content-addressed source/compiler identity sufficient to reconstruct the exact dirty build.
**Chosen: (a).** Grounds:

- **(a) is satisfiable by the doc's own recipe, so it forecloses nothing.** From a committed
  tree the Makefile's own inputs stamp a clean `Version` and a clean `Commit` (V26), and the
  clean state is reachable from any dirty tree at the same commit by one commit/stash. The gate
  relocates a once-per-release operation onto the only state that is reproducible — which is
  the definition of deterministic release evidence, not a cost.
- **(a) is the only option under which the existing 20-key manifest schema remains sufficient
  for its stated purpose** (V27) — no schema change, no new verify surface.
- **(b) is deselected NOT on the V22/V23 cost measurement.** That ~215 ms figure was taken on
  the compile hot path with caching on by default; a once-per-release freeze would pay it once,
  which is negligible — citing it here would be a scope-mismatched citation. (b) is deselected
  on three other grounds: **(i)** a content address lets a reviewer *verify* bytes they already
  possess, not *reconstruct* bytes they do not — reconstruction requires embedding the full
  uncommitted source diff in the manifest, a schema change plus a new end-to-end verify path;
  **(ii)** that diff is a leak surface — uncommitted local modifications (scratch keys, WIP)
  serialized into tracked, published release evidence, against the never-print-secrets rule;
  **(iii)** it legitimizes evidence whose provenance depends on un-versioned state, the exact
  opposite of what the artifact exists to provide, for a workflow (dirty release freezes) that
  no lane of this mission needs.

#### Systemic sweep table

Every gate/predicate and acceptance assertion, asked: *what is this cited for, what is the full
set of states satisfying it, and is that set larger than the purpose requires?* For ACs, also:
*what else, other than the mechanism under test, could produce the observed value?*

| Item | Cited purpose | Satisfying-state set | Verdict | Fix |
|---|---|---|---|---|
| `VersionKnown()` | knownness component (freeze branch 1; banking arm select) | any non-empty non-`"dev"` string, incl. `-dirty` strings | SOUND as a *component* — knownness is all it is cited for; consumers needing cleanliness must compose (freeze now does) | none |
| `CommitKnown()` | knownness component (freeze branch 2; `AttributionID` commit branch) | any non-empty non-`"dev"` string, incl. `abc1234-dirty` | SOUND as a component; **was TOO-WIDE as the freeze's terminal gate** — the round-4a instance | freeze gate composes `CommitClean()` (round 5) |
| `CommitClean()` | bytes-identity (cache gate; round 5: freeze gate too) | known `Commit`, no `-dirty` on either field | SOUND post-round-4; named residual stands (off-contract `Commit`-only hand-stamp — no string predicate can detect a clean-looking stamp that lies) | none (residual already named) |
| `AttributionID()` commit branch | attribution id for Consumer 3 | `CommitKnown` ∧ `Commit` not `-dirty`; `Version` arm vacuous on the only call path (`!VersionKnown()` ⇒ `"dev"`) | SOUND — narrowing documented (round-4 NOTED 1); contract forbids cache use | none |
| `AttributionID()` stat branch + `!ok` | deterministic per-build fallback; fail-safe on unstat-able exe | stat-derived id; `!ok` ⇒ Consumer 3 refuses | SOUND — mutation row 7 asserts `ok==false` AND `id==""` (zero-value hole already closed) | none |
| Consumer-1 freeze branches | manifest sufficient to recompute the cohort | pre-round-5: any KNOWN pair, **including dirty on either field** | **TOO-WIDE — round 4a** | branch 3 + gate `VersionKnown() && CommitClean()`; alternative (a) |
| Consumer-2 cache bypass gate | never Lookup/Store on ambiguous identity | `!CommitClean()` ⇒ bypass | predicate SOUND (post round 4); its ASSERTION (AC-6) was the too-wide half | AC-6 fix (round 4b) |
| Consumer-3 `VersionKnown()` arm (`releaseTag`) | per-release attributable bucketing | all builds whose `Version` parses to a release tag — incl. dirty and N-commits-past-tag builds | SOUND — the width is *deliberate and pinned*: `releaseTag` collapses `-dirty` by an existing test row (V28), and dirty-tree drift ⊂ intra-release drift (`v0.26.0-26-g…` → `v0.26.0`) which the per-release cadence pools by design; banking is attribution, not release evidence | none |
| Consumer-3 unstamped arm | per-build distinct bucket | `unstamped-` + `AttributionID` first 8 | predicate SOUND; its ASSERTION (AC-9) was too-wide — see AC-9 row | AC-9 fix |
| Consumer-3 `!ok` arm | refuse rather than guess a bucket | `ok == false` | SOUND | none |
| AC-1 / AC-4 / AC-7 / AC-10 / AC-14 | build/vet/fmt/suite regression floors | a green tree (all baselined rc=0, V20) | SOUND *as floors* — cited only as regression floors; no behaviour claim rests on them alone | none |
| AC-2 | M1 truth table exists and passes | N=4 + rc=0 with mandated rows | SOUND — N=0 instrument guard, literal EXPECTED_N, mandated rows backed by mutation rows 1–7 | none |
| AC-3 | package genuinely gained tests | output contains `ok ` and not `[no test files]` | SOUND — built against the rc-only hole (rc=0 pre-change, V11) | none |
| AC-5 / AC-8 / AC-12 | named regression pins with enumeration floors | the named tests present + green | SOUND | none |
| AC-6 | ambiguous identity neither Looks up nor Stores | pre-round-5: empty dir in a fresh env — **also satisfied by lookup-miss-then-skip-Store** | **TOO-WIDE — round 4b** | pre-populated dummy + compiles-anyway + no-new-files; mutation row 8c |
| AC-9 (`_UnstampedMarker`) | distinct bucket per unstamped build | pre-round-5: prefix + non-empty suffix — **also satisfied by a constant `unstamped-deadbeef`, i.e. by the shared-bucket defect itself** | **TOO-WIDE — sweep-found, NOT named by any reviewer** | assert bucket == `unstamped-` + first 8 of a direct `AttributionID()` call; mutation row 10 updated |
| AC-9 (`_StampedUnchanged`) | stamped path byte-identical to today | byte-equality with `releaseTag` output | SOUND | none |
| AC-11 | refusal branches distinguishable + pass branch proceeds | pre-round-5: 3 tests, `unstamped` substring — no dirty coverage at all (the test-side face of the round-4a hole) | **TOO-WIDE — round 4a, test side** | 5 tests; dirty tests assert `dirty` ∧ ¬`unstamped`; pass test pinned to the V26 clean shape |
| AC-13 | changelog entry present | pre-round-5: `make check-changelog` rc=0 — **baselined rc=0 BEFORE any entry existed (V20)**, so the no-entry state satisfies it; "with the new entry present" was prose, not an assertion | **TOO-WIDE — sweep-found, NOT named by any reviewer** | positive `grep -c` on the doc slug added |
| M4 remediation text | tells a refused operator how to satisfy the gate | pre-round-5 recipe (`git describe … --dirty`): in a dirty tree its own output now RE-FAILS the round-5 gate — a refusal loop | **consequential hole the round-5 gate change itself would have opened — sweep-found in the same round** | committed-clean requirement added to the remediation + manual arm 3 |
| Consumer-2 stderr warning | bypass observability | unasserted by any AC | NOTED-NOT-CHANGED (display-grade; emission site is a deferred decision; not a gate) | none |

#### NOTED-NOT-CHANGED (round 5)

1. Consumer 2's one-shot stderr bypass warning has no killing test — display-grade
   observability, emission site deliberately deferred to the implementer; not a gate.
2. The freeze's pre-spend property rests on call order (`validateCohortFreeze` before rig
   lock/API-key/benchmarks, V16), pinned only indirectly via
   `TestEvalSuiteBaselineRequiresVerify_CLI` — structural, unchanged.
3. Banking's dirty-collapse (`releaseTag` `-dirty` row, V28) — evaluated by the sweep and found
   SOUND, not changed: excluding dirty builds from release buckets would alter the stamped path
   (contra AC-8's `TestReleaseTag` pin and m-eval-version-banking's no-churn design) to protect
   an attribution surface that pools intra-release drift by design.
4. Every round-4 decision stands unmodified: the widened `CommitClean()`, the
   `Version == "dev"` + clean-`Commit` still-caches decision, V24/V25, and mutation row 8b —
   the sweep found none of them wrong.

## Instrument corrections vs. the task framing (measured this iteration)

Named explicitly, per the directive:

1. **C4's "0 vcs lines" needs the dotted pattern.** With `grep -c 'vcs'` the observed count is
   **1**, because `go version -m`'s output header contains the binary's own path and my test
   binary was named `wt-vcs-ailang` (V4). The substance of C4 fully stands (`grep -c 'vcs\.'` →
   0, rc=0, no Commit in `--version`); the lesson is that the acceptance-facing instrument must
   be `vcs\.`, and AC criteria in this doc use only test-level assertions, not that grep.
2. **The "three consumers" enumeration was incomplete, and the instrument could not have seen
   it.** Enumerating by import path instead of by `version\.Version` symbol grep surfaces **7**
   importing files, including two call sites (`prompt.go:94`, `mcp_status.go:84`) that alias the
   import as `versionpkg` and are invisible to the symbol grep (V10). The three decision-bearing
   consumers and the systemic-fix conclusion survive unchanged; the four additional surfaces are
   display/service-grade and are classified (not skipped) in the sweep table.
3. Everything else in the framing reproduced exactly: C1, C2, C3, C5, C6, C7, the 5-hit
   `version.Commit` grep, `[no test files]`, and the `go build ./...` rc=1 ban (V1–V3, V5, V9,
   V11, V15, V21). The framing's remediation claim initially rested on a read-only V17; quorum
   round 1 established that was inferential, and V17 is now the controller's executed
   in-worktree measurement (see the Quorum revision log).

**DESIGN_DOC_PATH**: `design_docs/planned/m-cohort-manifest-build-provenance.md`
