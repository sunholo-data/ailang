# Deterministic Interface Digests for Exported Type Schemes (#2076)

**Status**: Planned (design-scoping; no option chosen — human decisions below)
**Target**: v0.57.1
**Priority**: P1 (determinism violation in the compile-cache identity layer; no wrong verdicts observed)
**Estimated**: 3–4 days investigation+fix+regressions (conditional on option selection)
**Dependencies**: None (hard-depends on the human decisions in Design Freeze)

> **Scope of this document.** This is a DESIGN-ONLY scoping and review doc, per the
> issue-2076 task directive. It reproduces the reported nondeterminism at the pinned
> base `d9be18f1cdc462af5e6baf1bfe34193c1bd51dce`, proposes **candidate** definitions for
> alpha-equivalence and digest compatibility (explicitly human-owned, not assumed),
> enumerates solution alternatives WITHOUT selecting one, enumerates every
> published/stored interface digest surface affected, and defines appropriate
> regressions. No compiler implementation, planning, execution, or semantic approval
> is authorized by this document; no harness or artifact-serialization expansion is
> proposed as in-scope work.

## Problem Statement

`internal/iface` computes a per-module interface digest (`Iface.Digest`) by hashing,
among other fields, the **raw string rendering of each exported type scheme**
(`Builder.schemeToString`, `internal/iface/digest.go:61`). The typechecker assigns
fresh α-names (`α87`, `ρ3`, …) to quantified variables during inference, and that
assignment **permutes across processes** for at least one std module (`std/sem`).
`canonicalizeScheme` (`internal/iface/builder.go:549`) sorts the quantifier lists but
carries a TODO for body α-normalization:

```go
// TODO: Alpha-normalize the type to ensure consistent variable naming
// For now, just return with sorted quantifiers
```

The result: the same source, compiled by the same binary in two fresh processes,
produces α-equivalent exported schemes but **different interface digests**.

**Current State (all reproduced at pinned `d9be18f1`, this session — see Reproduction
Evidence):**

- 6 fresh-process compilations of `std/sem` → 4 distinct digests in one 6-run set
  (6 distinct observed across the session; the reporter saw 3 in 6 runs — small-sample
  variation of the same phenomenon, confirmed by the triage doc's V19 row at
  [m-typecheck-env-scan-cost](../../implemented/v0_57_0/m-typecheck-env-scan-cost.md) which measured
  the same thing on its own base binary).
- Full std corpus scan (49 modules × 3 runs): **only `std/sem` flaps** today.
- The permuted schemes are α-equivalent in every observed pair: only the cyclic
  assignment of fresh names to positions changes (`make_frame`'s
  `embedding/embedding_dim/ver` rotate among `α87/α88/α89`).
- The digest is **stable within a single process** (6 in-process compiles → 6 identical
  digests). The nondeterminism is a once-per-process draw, not a per-map-range
  randomization — see Reproduction Evidence for why this constrains the regression
  design.

**Impact — who is affected:**

- **Every fresh-cache compile context** mints a new `std/sem` digest: CI runs, new
  machines/checkouts, `AILANG_NO_CACHE=1` profiling runs, and — most concretely —
  motoko parallel task isolation, which sets a fresh
  `AILANG_CACHE_DIR=/tmp/motoko-task-<uuid>/cache` **per spawned task**
  (`internal/pipeline/cache_store.go` header comment). Every parallel task pays a
  different std/sem digest.
- **Dependent cache keys thrash:** a module's cache key includes the sorted
  dependency-closure digests (`cacheDepDigest` → `ModuleCacheKey`,
  `internal/pipeline/cache_dep_closure.go:48-49`, `internal/pipeline/cache_key.go`).
  A flapping std/sem digest changes the cache keys of every direct and transitive
  importer on each fresh compile → cascaded recompiles.
- **Stored digests are non-reproducible:** `CacheEntry.IfaceDigest` in
  `.ailang/cache/compile/manifest.json` and the cached `iface.json` artifact embed one
  random draw; byte-equivalence comparisons across runs (the methodology the
  typecheck-env-scan doc used for V18/V19) fail on std/sem and cannot be used as a
  regression gate for std/sem today.
- **Not affected (verified, see Verification Log V5):** the registry package identity
  (`InterfaceHashV2` → `HashProjection` over the already-canonicalized
  `InterfaceJSON`), the PUB005 skew guard, the agent-facing `ailang iface` /
  `internal-dump-iface` view (already α-canonicalized via `canonicalizeType`), and the
  builtin-freeze digest (`computeBuiltinDigest` — static schemes, no inference).
- **The CI gate that claims to cover this is vacuous:** `make test-iface-determinism`
  prints "Interface determinism verified (by construction)" and runs no test
  (`make/test.mk:240-242`). It cannot catch this bug class.

**Why it matters for the program:** AILANG's north star is a deterministic substrate;
the interface digest is the identity the compile cache relies on, and `M-XMOD-ALIAS`
work already established (via `TestXModAlias_DigestIgnoresTypeAliases`) that digest
stability is a load-bearing property — alias edits deliberately do NOT change digests
to avoid dependent-cascade rebuilds. Fresh-name permutation breaks the same property
from the other side.

## Reproduction Evidence (design-time, read-only, pinned `d9be18f1`)

Method (fully reproducible, no repo state changed):

1. Working tree verified clean at exactly `d9be18f1cdc462af5e6baf1bfe34193c1bd51dce`
   (`git status` empty; `git rev-parse HEAD` match).
2. Go toolchain 1.26.10 (`go.mod` requirement) downloaded to `/tmp` (session-scoped;
   the sandbox has no Go toolchain installed — the installed `/usr/local/bin/ailang`
   is `b23cadc-dirty`, **not** the pinned base, and was not used as evidence).
3. A **temporary, uncommitted** probe package (`tmpprobe2076/`, deleted after the
   run) inside the module called `pipeline.RunWithContext` with
   `Config{DryLink: true, PackageDir: "."}` on `std/sem.ail` and printed
   `result.Interface.Digest` plus each export's scheme rendering — the same digest
   value `publishFreshModule` stores (`internal/pipeline/pipeline_module_cache.go:170`).
   Each run got a fresh `AILANG_CACHE_DIR` so no cached iface masked the fresh
   compile. The probe binary was built from the clean tree with standard
   `-ldflags` version stamps.
4. No executor dispatch was required or performed; reproduction was local and
   read-only with respect to the source tree.

**Evidence table — 6-run std/sem reproduction at `d9be18f1`:**

| Run | digest (sha256 hex) |
|-----|---------------------|
| 1 | `cc532b08077834b5254a8422a7c512f0625b2d9c8bcc6369bb25fcc600d89cc8` |
| 2 | `f2b936fcc650bc0f7aba22ced7aae60056ce898c1aada0710ebdc4563a78f141` |
| 3 | `f7571a6b5b0e7691d1b10709300129fa1449329c583b72a8b769218fc276044d` |
| 4 | `f7571a6b5b0e7691d1b10709300129fa1449329c583b72a8b769218fc276044d` |
| 5 | `d5bf39341db4b87bc8d6a5a5f3fce8da39e91ccc1d87616650719b8489872340` |
| 6 | `f7571a6b5b0e7691d1b10709300129fa1449329c583b72a8b769218fc276044d` |

4 distinct digests in 6 runs. Across the whole session (6-run set + corpus scans)
7 distinct values were observed, with `f7571a6b…` the most frequent.

**Evidence — the permutation is pure α-renaming.** Full per-export diff between two
runs (only 3 lines of 23 differ: the digest line and two exports):

```
< make_frame: ∀α87 α88 α89. (string, string, bytes) -> { content: string,
    embedding: Option[α87], embedding_dim: α88, id: string, opaque: bytes,
    simhash: int, ts: int, ver: α89, ...{...same fields...} } ! {Clock}
> make_frame: ∀α87 α88 α89. (string, string, bytes) -> { content: string,
    embedding: Option[α89], embedding_dim: α87, id: string, opaque: bytes,
    simhash: int, ts: int, ver: α88, ...{...same fields...} } ! {Clock}
```

(`make_frame_at` permutes the same way among `α78/α79/α80`.) The quantifier lists are
already sorted (canonicalizeScheme does that); what permutes is which fresh name the
inference assigned to which position. Every observed pair of schemes is related by a
bijection on the quantified variables — i.e., α-equivalent.

**Evidence — in-process stability (critical constraint for test design).** Six
compiles of `std/sem` in one probe process → six identical digests
(`f7571a6b…`). The nondeterminism is therefore a **once-per-process** effect: some
map-derived order is captured once per process (stable for all compiles inside it)
and differs across processes. Consequences:

- An in-process N-iteration loop of the shape that caught the earlier
  `aliasDigest` bug (`internal/pipeline/cache_alias_digest_stable_test.go`, 200
  iterations) **cannot** catch this bug and must not be used as the regression.
- The precise capture point inside `internal/types`/`internal/pipeline` is NOT
  identified in this session; locating it is Phase-0 work (Open Question Q4). It is
  not `newTypeEnvWithBuiltinsImpl`'s builtin-map iteration (`BindScheme` is a plain
  map write, order-independent) and not the builtin registry (`registerFromSpecRegistry`
  sorts names before registration).

**Evidence — corpus blast radius.** All 49 `std/*.ail` modules compiled 3× each in
fresh processes: only `std/sem` produced differing digests. Matches the reporter and
V19 (`std/ + examples/` 448/449 identical, the one being std/sem). In-repo direct
importers of `std/sem` (grep, V11): `examples/reference/semantic_retrieval.ail`,
`examples/reference/sharedmem_cache.ail`, `examples/archive/broken/sem_frame_test.ail`.
External packages' import graphs cannot be enumerated from this checkout and are
structurally exposed the same way.

## Candidate Definitions (HUMAN-OWNED — not assumed by this document)

Per the issue-2076 task conditions, the following definitions are **candidates**, and
adopting them is a human decision. They are stated precisely so the decision is
cheap to make.

### D1 (candidate): Alpha-equivalence and digest α-invariance

- **Alpha-equivalence:** two type schemes `S = ∀Q. τ` and `S' = ∀Q'. τ'` are
  α-equivalent iff there is a bijection `σ : Q → Q'` (type vars to type vars, row
  vars to row vars, preserving kind) such that `σ(τ) = τ'`. This is the standard
  HM notion; the reproduction shows the flapping schemes are α-equivalent pairs.
- **Digest α-invariance property (the proposed regression oracle):**
  `Iface.Digest` must be a function of the export set modulo α-equivalence — for any
  two compilations of the same sources whose exported schemes are pairwise
  α-equivalent, the digests are equal.

**Open questions (human-owned):**

- **Q1:** Is α-invariance the correct identity relation for the *compile-cache*
  digest — i.e., may the digest deliberately distinguish two modules that differ
  only in fresh-name choice? (Today nothing consumes such a distinction; nothing in
  the codebase compares two compiled modules' raw α names for equality. V6/V7:
  instantiation freshens on every use, so importer verdicts are α-independent.)
- **Q2:** Does the stability requirement extend to the **raw cached artifact bytes**
  (the `iface.json` artifact's `MarshalScheme` output and `IfaceJSON` blob), or is
  digest-level stability sufficient? Byte-stability is what the V18/V19
  equivalence methodology needs; digest-only stability leaves that methodology
  broken for std/sem.

### D2 (candidate): Digest compatibility on algorithm change

Whichever option below changes the digest computation changes **every** module digest
that has quantified exports (std/sem's certainly, plus any module whose schemes
render differently under canonicalization). Candidate compatibility mechanisms:

- **M-bump:** one-time `cacheKeyVersion` bump (v8 → v9) in
  `internal/pipeline/cache_key.go` — the established mechanism; the v1→v8 history in
  that file documents exactly this pattern ("bumping guarantees old blobs never
  decode…"). Old entries miss once, everything recompiles once.
- **M-tag:** version the digest string itself (e.g., a schema prefix inside the
  hashed pre-image or in the stored field). More surface, no precedent.
- **M-none:** rely on compiler identity in the cache key (same commit → same
  algorithm). Rejected-by-precedent: `cache_key.go` documents that dev/worktree
  builds of the same commit must NOT silently share verdicts across algorithm
  changes — that is exactly what the version bump exists for.

**Open questions (human-owned):**

- **Q3:** Is a one-time global compile-cache invalidation acceptable? (Precedent:
  every prior cacheKeyVersion bump did exactly this; cost is one cold compile per
  project.)
- **Q3b:** Under Option A (α-normalized schemes in stored artifacts), do cached
  `iface.json` blobs from the old encoding need a decode guard beyond the
  key-version bump (the v2→v3 precedent added one for exactly this reason)?

## Solution Options (ENUMERATED, NOT CHOSEN — human decision Q0)

Per the issue-2076 task conditions, alpha-normalization must not be chosen by
assumption. The options, with what each fixes and what each costs:

| Option | Mechanism | Fixes digest | Fixes artifact bytes | Touches | Risk |
|--------|-----------|:---:|:---:|---------|------|
| **A — alpha-normalize schemes** | Close the `canonicalizeScheme` TODO: rewrite quantified vars in the scheme body (and quantifier list) to canonical `a, b, …` / row-`r` order before the scheme is stored on the `Iface` (thus before both digesting and artifact marshalling) | ✅ | ✅ | `internal/iface` (builder + digest input) | Low–med: changes the representation importers see on cache hit; importer instantiation freshens anyway (V7), but scheme bytes in every cached artifact change once |
| **B — digest-only canonicalization** | Keep raw schemes; compute the digest over a canonical rendering of each scheme (reuse the existing `canonicalizeType`/`InterfaceJSON` machinery (`internal/iface/json.go:189`) or a scheme-level equivalent) instead of raw `schemeToString` | ✅ | ❌ (V18-style byte comparisons stay broken for std/sem) | `internal/iface/digest.go` (+ a canonical-scheme renderer) | Low: smallest semantic surface; digest values change once |
| **C — digest versioning only** | Tag/document the digest as α-unstable per-process; consumers treat it as advisory | ❌ | ❌ | docs only | Does not fix cache thrash; enumerated for completeness; likely reject |
| **D — inference mint-order determinism** | Root-cause fix: locate the once-per-process map-order capture in `internal/types`/`internal/pipeline` and make fresh-name assignment deterministic | ✅ | ✅ | `internal/types` (security- and semantics-sensitive per AGENTS.md), possibly `internal/pipeline` | High: typechecker core; needs its own investigation (Phase 0) and its own design scrutiny |
| **E — layering** | B or A now for cache-digest stability; D as a separate follow-up design (it also stabilizes artifacts and every other raw-name consumer) | ✅ | partial now | as A/B + follow-up | Splits risk; recommended shape **if** the human wants byte-stability only eventually |

**Observations that should inform Q0 (not decide it):**

- The codebase already contains display-side α-canonicalization
  (`canonicalizeType`, `internal/iface/json.go:189`, "Type variables are renamed to
  a, b, c, …") and registry-side canonical hashing (`HashProjection`). Option B is
  "use the existing canonical form for the digest too"; Option A is "make the stored
  scheme itself canonical". No new canonicalization concept is needed for either.
- Option A additionally fixes Q2 in the affirmative (artifact bytes stable), which is
  what the V18/V19 equivalence methodology needs; Option B leaves that broken.
- Option D is the only one that fixes the phenomenon class at its source (any future
  module, not just std/sem, that gains a map-order-dependent mint), but the
  reproduction shows the source is subtle (process-stable, not per-range) and lives
  in the package AGENTS.md marks as security- and semantics-sensitive.
- std/sem being the only flapping std module today is **evidence about the class,
  not a bound on it**: any module whose exported scheme mints ≥2 fresh vars in a
  map-order-dependent traversal is exposed; std/sem merely is the only current one.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|----------------|-----------|----------|-------------|
| **Q0:** Option selection (A / B / C / D / E) | Determines whether artifact bytes stay nondeterministic (B/C) and whether the typechecker core is touched (D) | **human** | design | high |
| **Q1:** Adopt D1 (α-invariance as the digest identity property) | Defines what "fixed" means for every future digest change | **human** | design | med |
| **Q2:** Require raw-artifact byte stability or digest-only stability | Decides whether the V18/V19 equivalence methodology can be a permanent regression gate | **human** | design | med |
| **Q3:** Digest compatibility mechanism (M-bump vs M-tag vs M-none) | One-time global cache invalidation vs new versioned field surface | **human** | design | med |
| **Q4:** Authorize the Phase-0 root-cause investigation in `internal/types` (and whether it gates the fix or runs in parallel) | Root-cause fix risk profile vs symptom-fix timing | **human** | design | high |
| **Q5:** Regression placement: make `test-iface-determinism` real (subprocess canary) and whether a corpus-level digest-stability gate enters CI | CI runtime cost vs coverage of the actual bug class | **human** | design | low |
| Canonical-name alphabet for schemes (if A/B): reuse `a, b, …` display convention vs a distinct digest-only alphabet | Avoids confusing α-normalized stored schemes with source-level names in error messages | agent | implementation | low |

### Design Freeze

Before implementation begins, these must be resolved (all human-owned per the
issue-2076 task conditions — sprint-executor must PAUSE on any unchecked item):

- [ ] Q0 — solution option selected
- [ ] Q1 — D1 adopted (or amended)
- [ ] Q2 — artifact byte stability required or not
- [ ] Q3 — digest compatibility mechanism chosen
- [ ] Q4 — Phase-0 investigation authorized/sequenced
- [ ] Q5 — regression scope and CI placement

## Solution Design

### Overview

Whichever option is selected, the work decomposes the same way; the components below
are option-conditional and intentionally not fully specified (design-only scope).

### Architecture

**Components:**

1. **Phase 0 — root-cause investigation (conditional, Q4):** locate the
   once-per-process map-order capture that permutes fresh-name assignment for
   `make_frame`/`make_frame_at`. Constraints recorded by the reproduction
   (in-process stable; builtin registry and `BindScheme` excluded) bound the search.
   Deliverable: a written mechanism (file:line) sufficient to decide whether Option D
   is tractable in this version or its own design doc.
2. **Canonicalization (per Q0):** Option A closes the `canonicalizeScheme` TODO —
   rewrite quantified vars in the scheme body to canonical names in a deterministic
   traversal order, before the scheme reaches the `Iface` exports map; Option B adds
   a canonical scheme-rendering used only by `computeDigest`. Both MUST reuse the
   established traversal-based canonicalization pattern (`canonicalizeType` /
   `renderConstructor` in `internal/iface/json.go`) rather than inventing a second
   alphabet of machinery.
3. **Digest compatibility (per Q3):** `cacheKeyVersion` bump v8 → v9 (M-bump) with a
   comment in the established format of `cache_key.go`; plus, under Option A, a
   decode-guard decision per Q3b following the v2→v3 precedent.
4. **Regressions (per Q5):** see Testing Strategy.

### Implementation Plan (sketch — final plan is sprint-planner's, after Q0)

**Phase 1: Canonicalization** (~1 day, after Q0)
- [ ] Implement per chosen option in `internal/iface` (A: `canonicalizeScheme` +
      scheme-body rewrite; B: canonical digest input)
- [ ] Digest compatibility per Q3 (cache_key.go bump + comment)
- [ ] Unit regressions R1 (below)

**Phase 2: Cross-process regression** (~0.5 day)
- [ ] Make `make test-iface-determinism` run the real canary (R2), replacing the
      "by construction" stub
- [ ] Re-run the 6-run std/sem reproduction and the 49-module corpus scan; record
      digests in the Verification Log of the implemented doc

**Phase 3: Docs + release surface** (~0.5 day)
- [ ] Changelog fragment (digest change is a one-time invalidation — say so)
- [ ] Note in `docs/` cache documentation if it exists for cache invalidation

### Files to Modify/Create (indicative, option-dependent)

**Modified (Option A):**
- `internal/iface/builder.go` (~40 LOC) — close the TODO in `canonicalizeScheme`
- `internal/pipeline/cache_key.go` (+3 LOC) — version bump + comment

**Modified (Option B):**
- `internal/iface/digest.go` (~30 LOC) — canonical digest input
- `internal/pipeline/cache_key.go` (+3 LOC)

**New:**
- `internal/iface/digest_alpha_test.go` (~80 LOC) — R1 unit regressions
- `internal/pipeline/iface_digest_canary_test.go` (~120 LOC) — R2 subprocess canary
  (pattern: `internal/pkg/iface_subprocess_test.go` already builds `./cmd/ailang`
  and execs it — reuse that shape)

## Examples

### Before (today, at `d9be18f1`): same source, same binary, two fresh processes

```
$ ailang check … # process 1 — cache manifest for std/sem
"iface_digest": "f7571a6b5b0e7691d1b10709300129fa1449329c583b72a8b769218fc276044d"
$ ailang check … # process 2 — same command, same sources
"iface_digest": "cc532b08077834b5254a8422a7c512f0625b2d9c8bcc6369bb25fcc600d89cc8"
```

Every module importing std/sem gets a different cache key in each process →
fresh-cache contexts recompile the whole import closure.

### After (any of Options A/B): digest is a function of the α-equivalence class

```
"iface_digest": "<one value, every run, every machine>"
```

with a one-time cacheKeyVersion bump invalidating prior entries once (Q3).

## Affected Published / Stored Interface Digests (enumeration)

Every surface that stores or derives from `Iface.Digest` of a **compiled** module
(all verified by reading the code, V-log V2–V6):

| # | Surface | Location | Effect of the flap |
|---|---------|----------|--------------------|
| 1 | `Iface.Digest` itself | computed in `computeDigest` (`internal/iface/digest.go:30`), set at `builder.go:444-448` and recomputed by `SetDerivedEq` (`iface.go:155-159`) | random draw per fresh process (std/sem today) |
| 2 | Compile-cache manifest entry `iface_digest` | `CacheEntry.IfaceDigest` (`cache_store.go:45`, written `pipeline_module_cache.go:170`, serialized in `.ailang/cache/compile/manifest.json`) | stored value differs run to run |
| 3 | Cached `iface.json` artifact (per module) | `marshalIfaceFull` embeds `digest` + raw `MarshalScheme` bytes (`cache_artifacts.go:183`, `:180-215`) | artifact bytes AND digest flap; served to importers on cache hit |
| 4 | **Dependent modules' cache keys** | `cacheDepDigest` = `ifc.Digest + "+" + aliasDigest` (`cache_dep_closure.go:48-49`) folded into `ModuleCacheKey` (`cache_key.go:47-60`) | every direct/transitive importer of a flapping module gets a new key per fresh compile → cascade misses |
| 5 | In-repo importers of std/sem (today's blast radius) | `examples/reference/semantic_retrieval.ail`, `examples/reference/sharedmem_cache.ail`, `examples/archive/broken/sem_frame_test.ail` | their cache entries thrash in fresh-cache contexts |
| 6 | External packages importing a flapping module | cannot be enumerated from this checkout | structurally identical to #5 |

**Verified NOT affected** (this bounds the fix's blast radius):

- Registry package identity: `InterfaceHashV2` hashes `HashProjection(j)` over the
  canonicalized `InterfaceJSON` (`internal/pkg/hasher_v2.go:41`, `internal/iface/hash_projection.go`),
  where `j` comes from `internal-dump-iface`'s already-α-canonicalized output.
  PUB005 skew-guard behavior is unaffected by the flap.
- Agent-facing interface views: `ailang iface` / `internal-dump-iface`
  (`ToNormalizedJSON` → `canonicalizeType`, α-normalized to `a, b, c…`: `internal/iface/json.go:115`, `:189`).
- Builtin freeze golden digest (`computeBuiltinDigest`, static spec registry) —
  different code path, no inference; the one hardcoded digest golden in the test
  tree (`builtin_freeze_test.go:13`) must NOT change under any option.
- Eval/maintenance/mission digest namespaces (workspace artifacts, request digests)
  — unrelated to `Iface.Digest`.

## Success Criteria

- [ ] 6-run fresh-process canary on `std/sem` (and the two reference importers)
      produces exactly **one** distinct digest per module (acceptance: R2 below)
- [ ] Full 49-module std corpus × 3 runs: zero digest variation (re-run of this
      doc's corpus scan on the fixed build)
- [ ] R1 unit test: α-equivalent scheme pairs digest equal; non-α-equivalent pairs
      digest different
- [ ] `TestBuiltinInterfaceStability` golden unchanged; `HashProjection` output for
      a fixture module byte-identical before/after the change
- [ ] `make test-changed` (incl. test-core) green; `make verify-examples` green
- [ ] Changelog fragment documents the one-time digest/cache invalidation
- [ ] All Testing Strategy items that were approved in Q5 land in CI

## Testing Strategy

**Why the obvious test does not work (measured):** an in-process loop cannot catch a
once-per-process effect — the reproduction showed 6 identical in-process digests. The
regression MUST be cross-process for the real bug class; in-process unit tests cover
the *property* (α-invariance) instead.

**Unit (R1 — property, in-process, `internal/iface/digest_alpha_test.go`):**
- Two α-equivalent schemes (same structure, renamed quantified vars) → equal digest.
- Two structurally different schemes → different digests.
- Under Option A: a compiled `Iface`'s stored schemes are themselves canonical
  (round-trip through `marshalIfaceFull`/`unmarshalIfaceFull`).

**Integration (R2 — the actual bug class, cross-process,
`internal/pipeline/iface_digest_canary_test.go`):**
- Build `./cmd/ailang` (the established pattern in
  `internal/pkg/iface_subprocess_test.go`), exec it ≥6 times as fresh processes with
  fresh `AILANG_CACHE_DIR`s on `std/sem`, assert exactly one distinct digest.
- Wire into `make test-iface-determinism` (replacing the "verified (by construction)"
  stub at `make/test.mk:240`) — it is already part of `ci-strict`.

**Corpus (R3 — candidate, human-gated by Q5):**
- The 49-module × N-run digest scan used for this doc's evidence is currently a
  design-time manual method (temporary probe). Making it a standing target would be
  *harness expansion*, which the issue-2076 directive explicitly does not authorize;
  it is listed here as a candidate for the human to approve or reject, not as scope.

**Digest compatibility (R4):**
- Assert entries written under cacheKeyVersion v8 miss after the v9 bump (the
  existing cache-store tests cover the bump mechanics; extend with a comment-linked
  case per the file's precedent).

**Manual:**
- Re-run this doc's 6-run reproduction and corpus scan on the fixed build; record
  both in the implemented doc's Verification Log.

## Deferred Decisions

- Canonical-name alphabet details (reuse display `a, b…` vs distinct) — agent may
  choose, after checking error-message readability (deferred to implementation).
- Exact canary run count N (6 vs more) and timeout budget — agent may choose within
  CI runtime limits.
- Probe-test placement (`internal/pipeline` vs `internal/pkg` next to the existing
  subprocess-test shape) — agent may choose.
- Whether the Phase-0 writeup lands as a section of this doc's implementation
  report or a follow-up design doc — human at review, if Option D is in scope.

## Non-Goals

- **Expanding the equivalence/eval harness or artifact serialization** — explicitly
  out of scope per the issue-2076 task directive (R3 is a candidate for human
  approval only, not proposed scope).
- **Publishing interface digests to the registry** — the registry identity hash is
  unaffected and stays as-is (`HashProjection`).
- **Rewriting inference determinism wholesale (Option D) inside this design** — if
  Q4 selects D, it gets its own design scrutiny given `internal/types` is
  security- and semantics-sensitive (AGENTS.md).
- **Fixing the vacuous `test-iface-determinism` stub beyond the canary** — the stub
  gets a real test; no new CI framework.
- **Making `schemeToString` itself canonical for display** — display paths already
  canonicalize separately; only the digest path is in scope.

## Timeline

Estimates conditional on Q0/Q4; the one certain block is the regressions.

**Week 1** (~2 days):
- Human decisions Q0–Q5 (not estimated — human)
- Phase 0 investigation (if authorized) or canonicalization per chosen option
- R1 unit tests

**Week 2** (~1–2 days):
- R2 canary + make target; digest compatibility per Q3
- Corpus re-scan evidence; changelog; `make test-changed`/`verify-examples` green

**Total: ~3–4 days** (2× the naive estimate, per house rules)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Option A changes cached scheme bytes; some consumer compares raw scheme strings across modules | Med | Grep-and-audit raw `MarshalScheme`/`Scheme.String()` consumers during implementation; the reproduction pipeline re-run covers importers via the two reference examples |
| Option D (if chosen) destabilizes the typechecker core | High | Its own design doc + full `internal/types`/`internal/pipeline` cross-check + equivalence corpora before/after (V20-style) |
| Subprocess canary adds CI runtime | Low | Reuse the existing built-binary fixture; single module (std/sem), N bounded; only in `ci-strict` (already the slot) |
| One-time cache invalidation surprises users | Low | Changelog fragment + cache_key.go comment in the established format |
| Fresh α-flap discovered in another module post-fix (only std/sem known today) | Med | The canary asserts on the known case; corpus re-scan is recorded; Q2 answer decides whether a standing corpus gate is wanted |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | Restores run-to-run determinism of the compile-cache identity — the direct fix of this issue |
| A2: Replayability | 0 | No trace/execution changes |
| A3: Effect Legibility | 0 | No effect-row changes (row vars are α-renamed with the same machinery, not altered) |
| A4: Explicit Authority | 0 | No capability changes |
| A5: Bounded Verification | 0 | Digest is compile-time metadata; no verification-surface change |
| A6: Safe Concurrency | 0 | No concurrency changes |
| A7: Machines First | +1 | Machine-checkable cache identity; a subprocess canary makes the determinism claim mechanically enforced instead of asserted "by construction" |
| A8: Minimal Syntax | 0 | No syntax changes |
| A9: Cost Visibility | 0 | No resource accounting changes (one-time recompile cost documented in changelog) |
| A10: Composability | +1 | Reuses the existing canonicalization pattern (`canonicalizeType`) and the existing cacheKeyVersion mechanism rather than adding new machinery |
| A11: Structured Failure | 0 | No error-path changes |
| A12: System Boundary | 0 | No boundary changes |

**Net Score: +3** → **Decision: proceed to human decision points (Design Freeze), not implementation**

### Hard Violation Check

- [x] A1 (Determinism): removes implicit nondeterminism; introduces none
- [x] A3 (Effects): no hidden side effects
- [x] A4 (Authority): no ambient access granted
- [x] A7 (Machines First): no human-convenience tradeoff

## Verification Log

| # | Claim | Evidence | Status |
|---|-------|----------|--------|
| V1 | Working tree is clean at the pinned base | `git rev-parse HEAD` = `d9be18f1cdc462af5e6baf1bfe34193c1bd51dce`, `git status --short` empty (except this doc + the since-deleted probe) | Confirmed |
| V2 | `canonicalizeScheme` sorts quantifiers but does NOT α-normalize the body | `internal/iface/builder.go:549-567` — `// TODO: Alpha-normalize the type to ensure consistent variable naming / For now, just return with sorted quantifiers` | Confirmed |
| V3 | The digest hashes raw scheme renderings | `internal/iface/digest.go:61` (`Type: b.schemeToString(item.Type)`) → `builder.go:577-600` (`scheme.Type.String()` raw) | Confirmed |
| V4 | 6 fresh-process runs of std/sem → ≥2 distinct digests (reporter: 3 in 6; triage V19: 3 in 6) | Live reproduction at pinned base this session: 4 distinct in 6 runs (table above); 7 distinct observed overall | Confirmed (exceeds report — same phenomenon) |
| V5 | Registry identity and display paths are NOT affected | `internal/pkg/hasher_v2.go:39-56` hashes `iface.HashProjection(j)` over canonicalized `InterfaceJSON` (`hash_projection.go:20-60`, schema pinned `ailang.iface/v1`); `ToNormalizedJSON` uses `canonicalizeType` (`internal/iface/json.go:115`, `:189`, α→`a,b,c`) | Confirmed |
| V6 | Builtin freeze digest is a separate, unaffected path | `internal/iface/builtin_freeze.go:96` `computeBuiltinDigest`; only hardcoded golden is `builtin_freeze_test.go:13`; `registerFromSpecRegistry` sorts names (`internal/link/builtin_module.go:28-33`) | Confirmed |
| V7 | Importer verdicts are α-independent (schemes are instantiated fresh per use) | `generalizeWithConstraints`/instantiation docs (`internal/types/typechecker_functions.go:482-545`); no consumer compares raw α names across modules (repo-wide grep) | Confirmed |
| V8 | The digest is stable in-process, varies cross-process | Live probe: 6 in-process compiles → identical digest `f7571a6b…`; cross-process runs differ (V4) | Confirmed |
| V9 | Only `std/sem` flaps in the std corpus today | Live scan: 49 `std/*.ail` modules × 3 fresh-process runs; only `sem` differs | Confirmed |
| V10 | In-repo direct importers of std/sem | `grep -rln "import std/sem"` → `examples/reference/semantic_retrieval.ail`, `examples/reference/sharedmem_cache.ail`, `examples/archive/broken/sem_frame_test.ail` | Confirmed |
| V11 | `make test-iface-determinism` runs no test | `make/test.mk:240-242` — echo "Interface determinism verified (by construction)"; target listed in `ci-strict` (`make/ci.mk:24`) | Confirmed |
| V12 | Negative: no existing regression asserts cross-process digest stability for compiled modules | Only digest-stability tests: `TestBuiltinInterfaceStability` (builtins only) and `TestHashProjection_Deterministic` (display/canonical path); the 200-iteration `TestAliasDigest_RecordAliasStable` covers `aliasDigest`, a different function | Confirmed |
| V13 | Negative: `canonicalizeScheme`'s TODO is the only α-normalization gap in the digest path | `computeDigest` inputs: module, schema, exports (schemeToString), constructors (`Type.String()` on already-canonical constructor field types — ADT fields carry no inference-fresh α names: constructors are built from declared AST types), derivedEq; the α-flap enters only via exported scheme bodies | Confirmed |
| V14 | Prior same-class bug (map-order nondeterminism in cache identity) was fixed by canonicalization + stability test | `internal/pipeline/cache_alias_digest_stable_test.go:1-16` (Daneel, 2026-09-28: record print order gave every module a new cache key per run) | Confirmed |
| V15 | Negative: no Go toolchain in the sandbox; installed binary is NOT the pinned base | `go`/`make` absent from PATH; `/usr/local/bin/ailang --version` = commit `b23cadc…-dirty` (not in this checkout's history — `git cat-file -t b23cadc` fails). Reproduction therefore used a probe built from the clean pinned tree with go1.26.10 from `go.mod` | Confirmed |
| V16 | Negative: no executor dispatch was used for reproduction | Reproduction was local (temp probe package, deleted; evidence in this table); no `ailang` fleet/mission/eval invocation | Confirmed |

## Related Documents

**Implemented (informs design):**
- [m-registry-interface-hash-blind-to-signatures](../v0_35_0/m-registry-interface-hash-blind-to-signatures.md) — the registry identity hash already canonicalizes; the pattern to reuse for Option B
- [m-typecheck-env-scan-cost](../../implemented/v0_57_0/m-typecheck-env-scan-cost.md) — V19 row: first in-repo record of the std/sem flap, treated as pre-existing and out of scope there
- [m-crew-core-cold-compile-followup](../v0_55_0/m-crew-core-cold-compile-followup.md) — shares the env-walk/inference area Option D would touch

**Planned (check for overlap):** none found by SimHash/neural search at creation time;
manual review of the cache-related triage docs found no doc covering digest
canonicalization.

## References

- **Issue:** #2076 (reporter's 6-run std/sem reproduction with 3 digests; triage did not execute it — this doc executed it)
- **Axiom reference:** [Design Axioms](/docs/references/axioms)
- **Guardrails:** [AGENTS.md](../../../AGENTS.md) — `internal/types` is security- and semantics-sensitive (Option D risk); no-silent-fallbacks
- **Code anchors:** `internal/iface/builder.go:549` (TODO), `internal/iface/digest.go:30`, `internal/pipeline/cache_dep_closure.go:48`, `internal/pipeline/cache_key.go:20-47` (version-bump precedent), `make/test.mk:240` (stub target)

## Future Work

- Option D root-cause fix (inference mint-order determinism) as its own design, if Q0/Q4 defer it — it would additionally make raw artifact bytes reproducible.
- Standing corpus-level digest-stability gate (R3) if the human approves harness expansion separately.
- Consider folding a digest-stability assertion into the equivalence methodology used by cold-compile performance docs, once Q2 is answered.

---

**Document created**: 2026-11-03
**Last updated**: 2026-11-03
