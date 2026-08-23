# M-COHORT-MANIFEST-BUILD-PROVENANCE: Build identity for release evidence, cache identity, and eval banking

**Status**: Planned — **QUORUM-BLOCKED at round 3 (2026-08-23, iteration 256)**; one bounded,
unanimous, reviewer-specified revision away from routable. See "Quorum revision log — round 3"
for the exact resume. NOT blocked on a human, a lane, or a clock.
**Target**: v0.34.0
**Priority**: P0 (High) — the clause-5 KPI's frozen release evidence currently records `"dev"/"dev"` provenance
**Estimated**: 3–4 days (M1: 1d · M2: 1d · M3: 0.5d · M4: 1d incl. docs)
**Dependencies**: None
**Planner-Lane**: opus-required (M2 changes the compiler-cache identity component; both wrong-key failure classes are silent)

> All measurements in this doc were made by its author in the pin worktree
> `/Users/voightkampff/.ailang-driver-pin/v1` at `origin/dev` =
> `96381331a3e0dd0682873553b79d91c7c8e27f5f`, go1.26.6 darwin/arm64, on 2026-08-23 —
> **except V17 and V22, measured first-party by the controller** in the same worktree at the same
> HEAD during quorum revision round 1, and **V23**, measured first-party by the controller during
> quorum revision round 2, all transcribed verbatim (see the Quorum revision log).
> Every claim about current behaviour carries a Verification Log row (V1–V23, bottom of doc).

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
- A `--baseline` cohort freeze from an unstamped binary **exits non-zero before any spend**, with
  remediation text; a stamped freeze records real `ailang_version`/`git_commit`.
- Unstamped and stamped-dirty builds **decline module caching entirely** — no Lookup, no Store,
  one stderr warning — rather than being assigned a substitute compiler identity; a clean-stamped
  build's module-cache key stays byte-identical to today's `version.Commit`-keyed behaviour.
  There is no cache identity for two ambiguous builds to collide on or differ by, by construction
  (V22/V23: the rejected content-hash alternative cost ~215 ms/process against a measured 0–12 ms
  cache benefit).
- An unstamped `--bank-by-version` run banks into `unstamped-<id8>/`, never into a shared `dev/`
  — via `AttributionID()`, the metadata-derived id retained solely for this bucketing use and
  never consulted by the cache.
- `internal/version` goes from **zero test files** (V11) to a tested, load-bearing package.
- Clean-stamped (`make`-built at a clean commit) binaries: zero behaviour change on all three
  surfaces. A **dirty**-stamped build changes on exactly one surface — the module cache, which
  now bypasses instead of keying on the shared `abc1234-dirty` string that used to alias every
  rebuild's different compiler bytes to one identity (Objection 1, revision round 1; Failure
  Class 1). Freeze and banking still treat `-dirty` as known.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Cohort freeze **refuses** unknown identity; no `--allow-unstamped` escape | Release evidence integrity; any override re-opens the silent path this doc closes | human | design | med |
| Cache identity gated on `Identity.CommitClean()`: a clean known commit keys the cache exactly as today; unstamped or stamped-dirty **declines to cache** (no Lookup, no Store) rather than deriving a substitute identity | A substitute identity is either a stale-blob risk (a constant key, incl. a shared `abc1234-dirty` — the v1→v3 bump class, V8) or a ~215 ms/process tax to make safe (full-content hash, V22/V23) against a measured 0–12 ms cache benefit; declining to cache is correct **by construction** — no lookup, no stale blob | human | design | high |
| Unstamped banking bucket = `unstamped-<id8>` per build (not refusal, not shared `dev/`), via `AttributionID()` | Diverges from m-eval-version-banking's per-release cadence for the unstamped case only; changes on-disk layout eval consumers see | human | design | med |
| `CommitClean()` is a pure string check on already-loaded `Identity` fields — no `os.Stat`, no `!ok` branch, nothing to fail on. The cache gate cannot itself error; only `AttributionID()` (Consumer 3 only) retains an `os.Stat`-derived `!ok` branch, on which Consumer 3 refuses `--bank-by-version` | Two failure surfaces should not share one control-flow shape: a cache gate that can itself fail invites a fallback-on-error footgun; keeping the gate to a string comparison removes that surface entirely | human | design | med |
| Knownness sentinel stays the existing `"dev"` literal; ldflags/Makefile contract untouched | Changing the stamping contract would touch CI, Makefile, release tooling | agent | design | low |
| Accessor lives in `internal/version` (single source of knownness) | Three call sites otherwise re-derive `!= "dev"` independently and drift | agent | implementation | low |

### Design Freeze

Resolved before implementation starts:

- [x] Freeze refusal has no escape flag (remediation = build with ldflags; **executed end-to-end
      in a linked worktree**, V17)
- [x] Cache gate = `Identity.CommitClean()` (clean known commit only); unstamped or stamped-dirty
      **declines to cache** entirely (no Lookup, no Store) + one stderr warning — no substitute
      identity is ever derived, and the gate itself cannot fail (`CommitClean()` is a string
      check on already-loaded fields, so there is no `!ok` branch on the caching path)
- [x] Banking degrade = `unstamped-<id8>` bucket; `releaseTag()` itself is NOT modified (its
      `"dev"→"dev"` pin in `TestReleaseTag` stays green; the fix is at the call site — the exact
      "guard the helper, miss the call site" failure mode named in `cache_key.go:20-24`, V8/V13)
- [x] Exported vars `Version`/`Commit`/`BuildTime` remain (compat: `cmd/ailang/main.go:27-28` aliases, V10)

## Deferred Decisions

Intentionally left open for the implementer:

- Exact refusal message wording (must contain the substring `unstamped` and the remediation
  command; see M4) — agent
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
func (i Identity) CommitClean() bool  // i.CommitKnown() && !strings.HasSuffix(i.Commit, "-dirty")

// Get returns the package vars as one value (post-init(), i.e. after the
// ReadBuildInfo fallback has had its chance to populate Commit).
func Get() Identity

// AttributionID returns a best-effort identifier for bucketing eval output
// (Consumer 3 only). It is an ATTRIBUTION identifier, not a correctness
// identity, and MUST NEVER be used as a cache key — Consumer 2 does not call
// it; the module cache is instead gated directly on Identity.CommitClean().
//   CommitKnown && !strings.HasSuffix(Commit, "-dirty")
//                -> the commit string
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

**`CommitKnown()` vs `CommitClean()` (the consumer-1-vs-consumer-2 distinction).**
`CommitKnown()` answers *evidence*-knownness: Consumer 1 keeps accepting a `-dirty` commit,
because `-dirty` is visible to a reviewer, which is the standard the refusal protects.
`CommitClean()` answers *bytes*-identity: the same `abc1234-dirty` string is shared by every
rebuild from that dirty tree while the compiler bytes differ, so it cannot key a cache — Consumer
2 gates the module cache on `CommitClean()` alone, and any build that is unstamped or
stamped-dirty declines to cache rather than being assigned a substitute identity. Evidence-
knownness and bytes-identity are different requirements; `Identity.CommitKnown()` answers the
first, `Identity.CommitClean()` the second.

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
directly: an unstamped or stamped-dirty build declines to cache entirely (no Lookup, no Store)
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
`id version.Identity`, V16) with two refusal branches ahead of the existing `--verify` branch:

1. `baselineSet && !id.VersionKnown()` → error: freezing release evidence from an unstamped
   build; the manifest would record `ailang_version: "dev"`.
2. `baselineSet && id.VersionKnown() && !id.CommitKnown()` → same for `git_commit`.

Both messages MUST contain the substring `unstamped` (distinguishable from `validateBaselineID`
and `--verify` errors — an `err != nil` assertion alone is decorative) and the remediation:

```
rebuild with build identity, e.g.
  go build -ldflags "-X github.com/sunholo-data/ailang/internal/version.Version=$(git describe --tags --always --dirty) \
    -X github.com/sunholo-data/ailang/internal/version.Commit=$(git rev-parse HEAD)" -o <out> ./cmd/ailang
(works in linked worktrees: the Makefile derives these the same way — Makefile:27-29)
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
manifest; a manifest that exists implies stamped provenance from M4 on.

### Consumer 2 — module cache: BYPASS ON AMBIGUOUS IDENTITY (M2)

At `pipeline_module.go:276`, replace the raw `version.Commit` with:

```go
id := version.Get()
if !id.CommitClean() {
    // Ambiguous compiler identity (unstamped, or stamped-dirty): do NOT cache
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
- **Dirty-stamped builds** (`make`-installed from a dirty tree — the commonest developer state):
  previously every rebuild shared the `abc1234-dirty` commit identity while the compiler bytes
  differed — Failure Class 1 for exactly the binaries a developer iterates on. Now `CommitClean()`
  is false for any `-dirty` suffix, so the run is **bypassed entirely** — no Lookup, no Store, one
  stderr warning — rather than being given its own substitute identity. This is an **intended
  behaviour change** on this one surface (Objection 1, revision round 1; sharpened in round 2):
  the cache benefit given up (0–12 ms, V23) is strictly cheaper than any correctness-preserving
  alternative that was priced (~215 ms/process for a content hash, V22).
- **Unstamped builds:** `CommitKnown()` is false, so `CommitClean()` is false too — same bypass as
  the dirty-stamped case. No `os.Stat`, no computed id, nothing to be wrong: the run simply does
  not participate in the module cache.
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
- `internal/version/identity_test.go` (~150 LOC) — first tests for the package: knownness truth
  table (`VersionKnown`/`CommitKnown`/`CommitClean`), `AttributionID` commit-preference incl.
  `-dirty` exclusion, stat-id determinism + rebuild-sensitivity, unstat-able-path branch
- `internal/pipeline/compiler_identity_test.go` (~80 LOC) — `CommitClean()` gate selection +
  cache-bypass
- `cmd/ailang/eval_suite_bank_test.go` (~60 LOC) — bucket helper table test

**Modified files:**
- `internal/version/version.go` (+~60 LOC) — `Identity`, `Get`, `CommitClean`, `AttributionID`,
  `attributionIDFromPath`
- `internal/pipeline/pipeline_module.go` (+~15/−3 LOC) — call-site swap + bypass branch
- `cmd/ailang/eval_suite.go` (+~15/−4 LOC) — `bankBucket` helper at the single mutation point
- `cmd/ailang/eval_suite_cohort.go` (+~25 LOC) — two refusal branches in `validateCohortFreeze`
- `cmd/ailang/eval_suite_cohort_test.go` (+~60 LOC) — new refusal tests; existing signature
  callers updated (testing policy: rewrite outdated tests, no backward-compat shims)
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
  (three predicates, not two). `TestAttributionIDPrefersCommit` MUST carry a `-dirty` row: an
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
  `TestModuleCache_BypassedWhenIdentityAmbiguous` carries three arms — dirty-stamped, unstamped,
  and clean-stamped — driven by setting the exported `version.Commit` var directly and restoring
  it with `defer` (no `sync.Once` seam needed; resolves `gemini-3-1-pro`'s testability objection,
  see the Quorum revision log, round 2). The dirty-stamped and unstamped arms must assert the
  cache directory stays EMPTY after a compile — an assertion that would also pass on a
  lookup-miss-but-store path is decorative — and the clean-stamped arm must assert caching still
  happens.
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
  proven identical, not merely "some string".
- **AC-10:** `go vet ./cmd/ailang` → rc=0.

---

### M4 — freeze refusal + docs (1 day; strictly better)

Refusal branches in `validateCohortFreeze`; CHANGELOG; this doc's status stays Planned until the
sprint ships.

**Acceptance criteria:**

- **AC-11** (one test per refusal branch + the pass branch):
  ```bash
  L=/tmp/m4a.log
  go test ./cmd/ailang -count=1 -v \
    -run '^(TestValidateCohortFreeze_RefusesUnstampedVersion|TestValidateCohortFreeze_RefusesUnstampedCommit|TestValidateCohortFreeze_PassesStampedIdentity)$' >"$L" 2>&1
  RC=$?; N=$(grep -c '^=== RUN   Test[^/]*$' "$L")
  [ "$N" -eq 0 ] && { echo "INSTRUMENT FAILURE"; exit 3; }
  [ "$N" -eq 3 ] && [ "$RC" -eq 0 ]           # EXPECTED_N=3
  ```
  Both refusal tests must assert the error message contains `unstamped` (distinguishes from
  `validateBaselineID`/`--verify` errors, which can also make `err != nil` true).
- **AC-12** (existing freeze-gate behaviour retained):
  ```bash
  L=/tmp/m4b.log
  go test ./cmd/ailang -count=1 -v \
    -run '^(TestValidateCohortFreeze|TestValidateCohortFreeze_VerifyMessageExplainsWhy|TestEvalSuiteBaselineRequiresVerify_CLI)$' >"$L" 2>&1
  RC=$?; N=$(grep -c '^=== RUN   Test[^/]*$' "$L")
  [ "$N" -eq 0 ] && { echo "INSTRUMENT FAILURE"; exit 3; }
  [ "$N" -eq 3 ] && [ "$RC" -eq 0 ]           # EXPECTED_N=3 (all three exist today, V16)
  ```
- **AC-13:** `make check-changelog` → rc=0 with the new entry present in
  `changelogs/v0.18-current.md`.
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
| 8 | M2 | Ambiguous (unstamped or stamped-dirty) identity declines module caching entirely — no Lookup, no Store, no substitute key ever computed | `if false && !id.CommitClean()` (bypass branch dead; `id` stays used) | `TestModuleCache_BypassedWhenIdentityAmbiguous` (dirty-stamped and unstamped arms: cache dir stays empty after compile — a lookup-only assertion would also pass on the store path, so the dir listing is the assertion) |
| 9 | M2 | Clean-stamped builds keep byte-identical keys | force the bypass arm on a clean commit (`clean := false; _ = clean` on the `CommitClean()` branch) | `TestCompilerCacheIdentity_StampedUsesCommit` (key equals `ModuleCacheKey(commit,…)` recomputed directly, with a clean injected commit); also the clean-stamped arm inside `TestModuleCache_BypassedWhenIdentityAmbiguous` asserting caching still happens |
| 10 | M3 | Unstamped run banks under `unstamped-` prefix | `if false && !id.VersionKnown()` in `bankBucket` | `TestBankBucket_UnstampedMarker` (asserts prefix AND that the id8 suffix is non-empty — prefix alone could be satisfied by a constant) |
| 11 | M3 | Stamped run bucket byte-identical to today | force unstamped arm (`known := false; _ = known`) | `TestBankBucket_StampedUnchanged` + existing `TestReleaseTag` (AC-8) |
| 12 | M4 | Refusal branch: unknown Version | `if false && !id.VersionKnown()` | `TestValidateCohortFreeze_RefusesUnstampedVersion` (asserts substring `unstamped`) |
| 13 | M4 | Refusal branch: unknown Commit | `if false && !id.CommitKnown()` | `TestValidateCohortFreeze_RefusesUnstampedCommit` (asserts substring `unstamped`; run with VersionKnown=true so branch 12 cannot mask it) |
| 14 | M4 | Pass branch: stamped identity + valid flags proceeds | force refusal (`|| true` on branch 12's condition) | `TestValidateCohortFreeze_PassesStampedIdentity` (asserts `err == nil`) |

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
   `"dev"`; and, pre-revision, the stamped-dirty variant: every rebuild from a dirty tree shared
   `abc1234-dirty`): a rebuilt compiler decodes blobs written by a different compiler — the exact
   class the v1→v3 format bumps were shipped to defend against, but at the identity component the
   bumps cannot see (V8). Excluded by construction under this design: any identity that is not a
   clean known commit (`CommitClean()` false) never keys the cache at all — it bypasses — so
   there is no constant substitute key left to alias different compiler bytes.
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
- Module-cache behaviour, **dirty-stamped**: `abc1234-dirty`-keyed caching (the aliasing defect
  this replaces, Failure Class 1) → **no caching at all** — same bypass as unstamped, since
  `CommitClean()` is false for any `-dirty` suffix. Each `make` from a dirty tree no longer risks
  decoding a blob written by different compiler bytes, because no rebuild from a dirty tree is
  ever cached. Freeze and banking behaviour for dirty-stamped builds is unchanged: `-dirty`
  remains known evidence via `CommitKnown()`.
- `--bank-by-version` bucket: `dev/` → `unstamped-<id8>/` (the shared `dev/` bucket stops
  accumulating; existing `dev/` data is left in place, annotated by convention as pre-fix;
  `<id8>` comes from `AttributionID()`, the attribution-only identifier — never the cache).
- `--baseline` freeze: silent `"dev"/"dev"` manifest → pre-spend refusal.

Anything else that changes is a regression, not an intention.

## Testing Strategy

**Unit tests:** knownness truth table (`VersionKnown`/`CommitKnown`/`CommitClean`); `AttributionID`
branch coverage via the `attributionIDFromPath` seam (fixture files in `t.TempDir()`,
rebuild-sensitivity via rewrite + `os.Chtimes`, no faking of `os.Executable`); `bankBucket` table
test; `validateCohortFreeze` branch tests with injected `version.Identity` values (the new
parameter is the injection seam — no global mutation, no build-tag tricks).

**Integration tests:** `TestModuleCache_BypassedWhenIdentityAmbiguous` compiles a real module
with a forced dirty-stamped or unstamped `version.Commit` (set directly, restored with `defer` —
no `sync.Once` seam needed) and asserts the cache dir stays empty; its third, clean-stamped arm
asserts caching still happens.

**Regression-surface tests:** the fixtures listed in Conflict Surface, enforced by AC-5, AC-7,
AC-8, AC-12 with enumeration floors (every `-run` criterion carries a literal EXPECTED_N and
fails loudly on N=0 — `go test -run` exits 0 on a selector that matches nothing, so the exit
code alone is green before a single test exists).

**Manual verification (worktree, both arms):**
1. `go build -o /tmp/x/ailang ./cmd/ailang` (unstamped): `--version` shows no Commit; a
   `--baseline v1.1-rc1 --verify --dry-run`-class invocation refuses with the `unstamped`
   message; `--bank-by-version` prints an `unstamped-<id8>` output path.
2. Same build with the ldflags recipe from M4's remediation text (stamped even in a worktree,
   executed-verified V17): freeze proceeds; bucket is the release tag; manifest records real
   values. Expectation check: `go version -m` on this binary still shows **0** `vcs.*` lines
   (V17) — that is the ldflags mechanism working as designed, not a failed remediation.

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
| Freeze refusal breaks an existing automation that froze unstamped | Med | That automation was producing corrupt evidence (V15); the refusal message carries the exact rebuild command; lands pre-spend so nothing is wasted |
| `validateCohortFreeze` signature change ripples through tests | Low | Three existing tests touch it (V16); testing policy is rewrite-not-shim; AC-12 pins retained behaviour |
| A future caller reads `version.Version` raw instead of `Get()` | Med | Accessor is the documented entry point; package doc comment updated in M1 to point decision-bearing consumers at `Identity`/`CommitClean`/`AttributionID`; Conflict Surface names the classification rule |

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Cache correctness no longer depends on deriving an identity for ambiguous builds at all — unstamped and dirty-stamped builds decline to cache instead of aliasing (today: every unstamped build to `"dev"`; every dirty-stamped rebuild to one `-dirty` string), and the clean-stamped path is unchanged |
| A2: Replayability | +1 | The cohort manifest becomes actually sufficient to recompute the cohort |
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

| V24 | **[CONTROLLER, executed — the row `gpt5-6-sol` demanded in round 3]** The two stamping paths put `-dirty` on **DIFFERENT FIELDS**, so a `Commit`-only dirty check is blind to every `make`/ldflags-stamped dirty build | Read-only evaluation of the exact `Makefile:27-28` inputs in a genuinely dirty tree — the main checkout `/Users/voightkampff/dev/sunholo-data/ailang` (control: `git status --porcelain` shows **3** modified tracked files + 3 untracked) — against the clean pin worktree as the control arm, plus the `debug.ReadBuildInfo()` arm from the same dirty tree (arm B of V2) | **ldflags path, dirty tree:** `git describe --tags --always --dirty` → `v0.33.1-218-gdb71d2a16-dirty` (dirtiness on **Version**), `git rev-parse HEAD` → `db71d2a1638bf1fa379945289fc90fde749819cd` (**plain SHA, no `-dirty`**). **Control, clean pin worktree:** `v0.33.1-220-g96381331a` / `96381331a3e0…` — neither field dirty. **ReadBuildInfo path, same dirty tree:** `--version` prints `AILANG dev` with `Full: db71d2a1638bf1…-dirty` (dirtiness on **Commit**). So `-dirty` reaches `Commit` ONLY via `ReadBuildInfo`, and reaches `Version` ONLY via ldflags. `CommitClean()` as specified in round 2 therefore returns **true** for every `make`-installed dirty binary — the exact population it was written to exclude |

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
