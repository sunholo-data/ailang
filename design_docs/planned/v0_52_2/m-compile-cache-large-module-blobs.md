# Large test modules never compile-cache: CoreTI blob dedup + configurable artifact ceilings

**Status**: Planned
**Target**: v0.52.2
**Priority**: P1 (DX — measured 10-minute test runs on a real project)
**Estimated**: 3 days
**Dependencies**: None hard. Builds on the unreleased cap raise at HEAD (64/128 MiB, `changelogs/unreleased/2026-10-06-mod010-package-file-cache-cap.md`); this doc covers the durable half of the same ask.

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

Every feature must align with AILANG's 12 Design Axioms. Score each axiom and verify no hard violations.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | The dedup key is each type's own gob bytes — a pure function of the value, so the encoded blob is byte-deterministic for a fixed module |
| A2: Replayability | 0 | No trace or execution-semantics change; cache hit vs miss is already outside the replayed verdict |
| A3: Effect Legibility | 0 | No effect surface touched |
| A4: Explicit Authority | +1 | Keeps the stamp authorization (4 payload digests, module ID, cache key) intact; oversized artifacts still refuse to publish |
| A5: Bounded Verification | +1 | The ceilings stay bounded reads (Stat-before-read, aggregate module budget); dedup shrinks what is read instead of unbounding it |
| A6: Safe Concurrency | 0 | No concurrency change; cache write path is unchanged |
| A7: Machines First | +1 | The failure mode stops being "a warning a human must notice for months" and becomes "works, or names the exact byte limit and the env var that moves it" |
| A8: Minimal Syntax | +1 | No language syntax change |
| A9: Cost Visibility | +1 | The warning now states the byte limit that was hit (numbers, not adjectives) |
| A10: Composability | 0 | Purely cache-layer; composes with `AILANG_CACHE_DIR` and `AILANG_NO_CACHE` unchanged |
| A11: Structured Failure | +1 | Malformed env values fail loudly with the variable named (the `FSMaxBytes` pattern), never a silent default |
| A12: System Boundary | 0 | No boundary crossing added or removed |

**Net Score: +8** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced
- [x] A3 (Effects): No hidden side effects
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): Not optimizing for human convenience over machine analysis

## Problem Statement

`ailang test --package .` in stapledons-godot's `sim` package (289 tests) takes ~10 min (9m36s and 10m0s measured 2026-10-06, M-series Studio). Every run prints:

```
CACHE_WRITE_FAILED module=sim/protocol_test stage=encoding path=sim/.ailang/cache/compile/modules/sim__protocol_test/coretypeinfo.gob: ARTIFACT_TOO_LARGE: artifact exceeds blob byte limit; using fresh compilation
```

so `sim/protocol_test` (638 lines) is recompiled from scratch on every run.

**Current State:**

- The v0.52.0 binary caps each artifact blob at 16 MiB (`maxArtifactBlobBytes = 16 << 20` at that commit). protocol_test's `coretypeinfo.gob` is 21.8 MiB, so the cache refuses to store it, every run, forever.
- HEAD (unreleased) already raises the ceilings 16/32 → 64/128 MiB (`internal/pipeline/cache_artifacts.go:29-35`, changelog fragment `2026-10-06-mod010-package-file-cache-cap.md`, refs #1328). That takes a warm `check protocol_test.ail` from 7.3 s to 0.53 s. It is necessary and ships the "raise" half of the ask — but it is a stopgap, not a fix:
  1. **The ceilings are hard-coded constants.** The next module that outgrows 64 MiB degrades to the same silent every-run recompile, with only a stderr warning. The operator has no knob.
  2. **The blob is far larger than its information content.** `CoreTypeInfo` is `map[uint64]Type`; the gob encoding transmits each node's type as a full value (gob deduplicates type *definitions* per stream, not *values*). Measured on `stapledons/sim/protocol` (the #1328 triage addendum): 11,639 entries, only 996 distinct types by `String()`, 3.0 MB of type text against 1.0 MB unique. A 638-line test module producing 21.8 MiB of type info is encoding the same ~1k types ~11k times.

**Impact:**

- Real projects with large test modules: every `ailang test` invocation pays a full recompile of the test module (7.3 s for protocol_test alone) plus loses the cache for everything downstream of it.
- The failure is silent-by-design (a warning, then fresh compilation) — correct as a *fallback*, but there is no escalation path: the warning names no byte count and no remedy.

## Goals

**Primary Goal:** Make the compile cache hold modules whose CoreTI blob exceeds today's ceilings — by shrinking the blob to its unique content and by making the ceilings operator-configurable — without weakening the bounded-read or authorization guarantees.

**Success Metrics:**

1. protocol_test's `coretypeinfo.gob` encodes to a small multiple of its unique type content (target: several-fold reduction; measured in-sprint on the real module — see Deferred Decisions for the instrument).
2. `cd sim && ailang test --package .` warm-run time drops by the protocol_test compile share (the per-test recompile of the test module itself is #1328's separate backlog row and is NOT claimed here).
3. `AILANG_CACHE_MAX_BLOB_BYTES=1G` (or any documented spelling) is honored; a malformed value fails loudly naming the variable; unset keeps the 64/128 MiB defaults.
4. An over-ceiling module's warning names the limit in bytes and the env var that moves it.
5. `make simplicity-audit` stays green: env reads stay in `internal/config`, every new var documented.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Dedup the CoreTI blob at encode time (types table + node→index map), keyed by each type's own gob bytes | This is the structural fix; chunking and gzip only postpone the same wall | agent | design | med |
| Bump `cacheKeyVersion` v5 → v6 so old-format blobs never decode into the new codec | One-time full-cache miss for every module; the existing stamp check already enforces it | agent | design | low |
| Decode shares one `Type` pointer per distinct type across node entries (instead of fresh values per entry) | Memory win, but only safe if no consumer mutates a Type in place — needs a guard test, not an assumption | agent | design | med |
| Two new env vars (`AILANG_CACHE_MAX_BLOB_BYTES`, `AILANG_CACHE_MAX_MODULE_BYTES`), parsed by `config.ParseByteSize`, read in `internal/config`; malformed values FAIL the run (typed configError), filesystem construction errors keep today's warn-and-bypass | Names and semantics become public API; the simplicity audit gates where they are read and that they are documented; the error routing changes an existing path's behavior (V16) | agent | design | low |
| Reject "chunk coretypeinfo.gob into N files" | Breaks the stamp's exactly-four-payload-digest invariant and complicates authorization for no gain once dedup lands | agent | design | low |

### Design Freeze

Before implementation begins, these must be resolved:

- [x] Dedup (not chunking, not gzip-only) is the mechanism — frozen in this doc.
- [x] Env vars are the operator knob, fail-loud, in `internal/config` — frozen.
- [x] `cacheKeyVersion` bump is accepted (one-time cache miss per module after upgrade) — frozen.
- [ ] Human approval of this design doc (the gate this document is for).

## Solution Design

### Overview

Two changes at the cache layer, neither touching the language, the type system's semantics, or the pipeline's control flow:

1. **M1 — CoreTI codec dedup (the structural fix).** `coretypeinfo.gob` stops being `gob(CoreTypeInfo)` and becomes `gob({Types []Type; Entries map[uint64]int32})`: each distinct type is encoded once, keyed losslessly by the type's own gob bytes; each node maps to a table index. The codec lives entirely behind the existing `cacheArtifactCodec.encodeCoreTI/decodeCoreTI` seam.
2. **M2 — Configurable ceilings (the operator knob).** `AILANG_CACHE_MAX_BLOB_BYTES` and `AILANG_CACHE_MAX_MODULE_BYTES` override the 64 MiB / 128 MiB defaults, parsed by `config.ParseByteSize` (one spelling for every cap), failing loudly on malformed values. The stamp cap stays fixed (it is a 64 KiB JSON document; making it configurable buys nothing).

Plus one small diagnostic (**M3**): the `ARTIFACT_TOO_LARGE` message names the byte limit and the env var, so the next module that outgrows the ceiling tells the operator exactly what to do.

### Architecture

**Components:**

1. **Dedup codec** (`internal/pipeline/cache_artifacts.go`, inside `productionCacheArtifactCodec`):
   - Encode: for each `(nodeID, typ)` in the CoreTI, gob-encode `typ` alone, use those bytes as the dedup key. Identical bytes ⇒ identical decode ⇒ merging entries is lossless. Distinct bytes ⇒ separate table slots (never merges anything). Emit `coreTIDedup{Types: table, Entries: nodeID→int32}` through one gob stream.
   - Decode: read the struct, rebuild `map[uint64]Type` with `table[entries[nodeID]]`. Entries that shared a key share one decoded pointer — this is intended (11,639 entries over 996 distinct types ≈ 12× fewer live Type values in memory after a warm load).
   - Determinism: the table order follows map iteration order of the *input* CoreTI, which is not deterministic — but the *decoded value* is, and the cache key already covers source identity. The blob bytes may differ run-to-run for the same module; the stamp digest is computed over whatever bytes were written, so verification is unaffected. (If byte-stability is wanted for tests, sort by key — implementer's choice, noted in Deferred Decisions.)
2. **Config getters** (`internal/config/compiler.go`, beside `FSMaxBytes`):
   - `CacheMaxBlobBytes() (int64, bool, error)` / `CacheMaxModuleBytes() (int64, bool, error)` following the `FSMaxBytes` shape exactly: unset → `(0, false, nil)`; malformed → error naming the variable; valid → parsed bytes.
   - `NewCacheStore` (`internal/pipeline/cache_store.go:80`) resolves them once at store construction into `artifactLimits`, replacing the direct `productionArtifactLimits()` call.
   - **Error routing (a real behavior decision, not an assumption):** today a `NewCacheStore` error does NOT fail the run — `cacheRuntime` init (`cache_runtime.go:40-44`) warns `CACHE_WRITE_FAILED stage=initialization` and bypasses the cache. That is right for filesystem errors (a broken cache dir must not break compilation), but wrong for a malformed safety cap: `ParseByteSize`'s own contract says a misconfigured cap fails loudly. The design splits the two: `NewCacheStore` wraps env-validation failures in a typed `configError`; `cacheRuntime` propagates `configError` up as a run failure (the caller already returns it through the pipeline) and keeps warn-and-bypass for every other construction error. Tests pin both arms.
   - Both variables are registered in the env registry (`AreaCompiler` — "Compiler and runtime") so `docs/docs/reference/env-vars.md` lists them and the 100%-documented simplicity gate holds.
   - Validation: module limit ≥ blob limit is NOT enforced (the binding logic already handles a smaller remaining budget — `bindingArtifactLimit` reports the `module` scope); zero or negative is rejected by `ParseByteSize`.
3. **Diagnostic** (`internal/pipeline/cache_artifacts.go:artifactTooLarge` + `cache_runtime.go:warnWrite`): the error message becomes `artifact exceeds blob byte limit 67108864 (raise AILANG_CACHE_MAX_BLOB_BYTES)` — the limit number the read path already prints (`scope=... limit_bytes=...`) now also reaches the write path's `CACHE_WRITE_FAILED` line through the existing `cacheArtifactError.LimitBytes` field. The knob is named only in the `blob` scope's message (the `stamp` and `module` scopes keep their plain forms).

**Why not chunking:** `storeArtifacts` publishes exactly four payloads authorized by `artifacts.json` (`validateArtifactDigests` requires exactly four digests); chunked CoreTI would need N digests, N bounded reads, and an aggregate check the module budget already provides. Dedup attacks the cause (repeated values) instead of the symptom (file size), so the ceiling is hit later or never.

**Why not gzip:** compression shrinks the blob without fixing the representation, adds a decompression step to a path whose security property is "bounded read before decode", and its ratio on already-repetitive gob is the same redundancy dedup removes losslessly. Can be layered later (see Future Work).

### Implementation Plan

**Phase 1: CoreTI dedup codec** (~1 day)
- [ ] Add `encodeCoreTIDedup` / `decodeCoreTIDedup` behind the codec seam; wire them into `productionCacheArtifactCodec`.
- [ ] Bump `cacheKeyVersion` to `"v6"` with a comment in the existing lineage block (`cache_key.go`) explaining the CoreTI format change.
- [ ] Unit test: round-trip a synthetic CoreTI with repeated types; assert entry-count, per-node type equality (`Equals`), and that two nodes with the same type share one pointer after decode.
- [ ] Unit test: the dedup key is lossless — build a CoreTI containing types that are `Equals` but NOT byte-identical under separate gob encodings (if constructible) and assert they are kept separate (dedup never merges more than byte-identity warrants).
- [ ] Mutation guard test: after decoding with shared pointers, perform every in-place map operation consumers do (`Set` on one node, `ApplySubstitution`) and assert other nodes' types are unchanged (see Conflict Surface).

**Phase 2: configurable ceilings** (~1 day)
- [ ] `CacheMaxBlobBytes` / `CacheMaxModuleBytes` in `internal/config/compiler.go` (the `FSMaxBytes` pattern, `ParseByteSize`); register both in the env registry with `AreaCompiler` docs text.
- [ ] `NewCacheStore` resolves the overrides into `artifactLimits`; constructor error on malformed values; `productionArtifactLimits` remains the default source.
- [ ] Tests: unset → defaults; `256MB`/`268435456` spellings → honored; `abc`/`-1` → run fails naming the variable (configError propagates, cache is NOT silently bypassed); store respects a lowered blob limit end-to-end (store refuses, `CACHE_WRITE_FAILED` printed once, no manifest entry).
- [ ] Docs: `docs/docs/reference/env-vars.md` rows (required for the 100% documented gate).

**Phase 3: diagnostic + measurement** (~0.5 day)
- [ ] `artifactTooLarge` message carries the limit and the knob name; `CACHE_WRITE_FAILED` write-path line shows it (the read path already prints `limit_bytes`).
- [ ] Measurement on the real module (stapledons-godot `sim/protocol_test.ail`): record naive vs deduped `coretypeinfo.gob` sizes and warm `check` time; put the numbers in the implementation report.
- [ ] Changelog fragment under `changelogs/unreleased/`.

### Files to Modify/Create

**New files:**
- `internal/pipeline/cache_coreti_dedup_test.go` — round-trip, sharing, and mutation-guard tests (~150 LOC)

**Modified files:**
- `internal/pipeline/cache_artifacts.go` — dedup codec inside `productionCacheArtifactCodec`, `artifactTooLarge` message (~60 LOC)
- `internal/pipeline/cache_key.go` — `cacheKeyVersion` v5 → v6 + lineage comment (~5 LOC)
- `internal/pipeline/cache_store.go` — resolve config overrides in `NewCacheStore` (~15 LOC)
- `internal/config/compiler.go` — two getters + env constants (~30 LOC)
- `docs/docs/reference/env-vars.md` — two rows (~4 LOC)
- `changelogs/unreleased/<date>-compile-cache-large-module-blobs.md` — fragment (~15 LOC)

## Examples

### Example 1: The reported failure, after

**Before (v0.52.0, and any module over the ceiling):**
```
CACHE_WRITE_FAILED module=sim/protocol_test stage=encoding path=sim/.ailang/cache/compile/modules/sim__protocol_test/coretypeinfo.gob: ARTIFACT_TOO_LARGE: artifact exceeds blob byte limit; using fresh compilation
```
(then a 7.3 s recompile of protocol_test, every run)

**After (default config):**
```
(no warning — the deduped blob fits the 64 MiB ceiling; warm check 0.53 s)
```

**After (a module over the ceiling):**
```
CACHE_WRITE_FAILED module=big/module stage=encoding path=…/coretypeinfo.gob: ARTIFACT_TOO_LARGE: artifact exceeds blob byte limit 67108864 (raise AILANG_CACHE_MAX_BLOB_BYTES); using fresh compilation
```

### Example 2: The operator knob

```bash
# One spelling for every byte cap in the CLI (ParseByteSize):
AILANG_CACHE_MAX_BLOB_BYTES=512MB ailang test --package .

# Malformed — the run fails at cache construction, naming the variable:
AILANG_CACHE_MAX_BLOB_BYTES=fifty ailang check x.ail
# error: AILANG_CACHE_MAX_BLOB_BYTES: invalid size "fifty": want an integer, optionally with a K/M/G/T suffix (e.g. 256MB, 8G)
```

## Success Criteria

- [ ] A synthetic CoreTI with N entries over D distinct types round-trips exactly (per-node `Equals`), and same-type nodes share one decoded pointer (unit test).
- [ ] Dedup never merges byte-distinct encodings (unit test; property stated as a test name).
- [ ] `cacheKeyVersion` bump: a v5-format cache directory misses once and then re-populates in the new format (regression test driving the pipeline like `TestCacheArtifacts_ByteLimits`'s `pipeline_miss_repairs_oversized_blob_then_hits` arm).
- [ ] `AILANG_CACHE_MAX_BLOB_BYTES` honored end-to-end (store refuses over-limit, warning printed, no manifest entry); malformed value fails loudly; unset keeps 64/128 MiB defaults.
- [ ] Measured sizes for `sim/protocol_test`'s blob before/after land in the implementation report.
- [ ] `make test-core` (or the focused `go test ./internal/pipeline/ ./internal/config/`) green; `make simplicity-audit` green.
- [ ] `docs/docs/reference/env-vars.md` lists both variables.
- [ ] All tests passing; documentation updated.

## Testing Strategy

**Unit tests:**
- Codec round-trip, pointer sharing, byte-key losslessness (Phase 1).
- Config getters: unset/spelled/malformed matrix (Phase 2).
- Store with lowered limits: refuse + warn once + no manifest entry (mirrors the existing `oversized_publication_is_non_authorizing` arm).

**Integration tests:**
- Pipeline-level: cache a module, corrupt/replace the CoreTI blob with a v5-format (old-codec) blob plus a forged stamp with a matching digest — the `stamp.Version != cacheKeyVersion` check must reject it (this is the authorization regression for the format change).
- The existing `TestCacheArtifacts_ByteLimits` stays green untouched (limits and bounded-read semantics unchanged).

**Manual testing:**
- On stapledons-godot `sim`: `cd sim && ailang test --package .` twice — second run has no `CACHE_WRITE_FAILED`, and `sim/.ailang/cache/compile/modules/sim__protocol_test/coretypeinfo.gob` is materially smaller than 21.8 MiB.

## Verification Log

Checked against the repo at HEAD (single-commit checkout, `07e1a89b`) on 2026-10-06. Negatives carry their instrument. No Go toolchain was available in this authoring session, so every claim below is a code read, a grep, or a citation of an already-banked measurement — no new runtime measurement is claimed.

| # | Claim | How verified | Result |
|---|-------|--------------|--------|
| V1 | The reported symptom matches the code: the blob ceiling rejects the write at the encoding stage and the run continues with fresh compilation | `internal/pipeline/cache_artifacts.go` `encodeArtifacts` → `checkArtifactSize` → `artifactTooLarge("encoding", …)`; `cache_runtime.go:141` `warnWrite` prints `CACHE_WRITE_FAILED … using fresh compilation`, once per `(stage, module)` per run | **Confirmed** |
| V2 | v0.52.0's ceilings were 16/32 MiB and HEAD's are 64/128 MiB | The user's v0.52.0 binary prints `ARTIFACT_TOO_LARGE` for a 21.8 MiB blob (issue report); `cache_artifacts.go:29-35` now says `64 << 20` / `128 << 20` with the #1328 comment; `cache_artifacts_test.go:193` pins 64/64KiB/128 | **Confirmed** (the raise is at HEAD, unreleased — fragment `changelogs/unreleased/2026-10-06-mod010-package-file-cache-cap.md`) |
| V3 | **No env var or flag configures the artifact ceilings today** | `grep -rn "os.Getenv\|LookupEnv" internal/pipeline` → only `DEBUG_*` knobs and `pipeline_module_phases.go:295` (`AILANG_NO_CACHE`, read via config); `internal/config/paths.go` env registry has no size-limit var; `artifactLimits` is populated solely from the constants via `productionArtifactLimits()` (`cache_store.go:84`) | **Confirmed (negative)** |
| V4 | The ceilings are a bounded-read guarantee: an oversized artifact is rejected by `Stat` before any payload byte is read | `readBoundedArtifact` stats, checks `info.Size() > limit`, closes, returns `artifactTooLarge` before `io.ReadAll`; `TestCacheArtifacts_ByteLimits/oversized_blob_rejected_by_stat_without_read` pins `reads == 0` | **Confirmed** — the design must not unbound this |
| V5 | The blob's redundancy is real, not assumed: gob does not share values across map entries | Banked measurement in the #1328 triage addendum (`design_docs/planned/ailang-core-triage/inline-test-per-case-recompile-1328.md`, addendum 2026-10-06): `stapledons/sim/protocol` has 11,639 CoreTI entries, 996 distinct types by `String()`, 3.0 MB of type text vs 1.0 MB unique; protocol_test's blob is 21.8 MiB | **Confirmed (measured, cited — not re-measured here; re-measurement is a Phase 3 task)** |
| V6 | Merging types by identical gob bytes is lossless | Encode/decode determinism of `encoding/gob` for a fixed registered type set: identical input bytes decode to identical values. The key is the exact encoded bytes, so the codec only ever merges entries whose encodings are byte-identical | **Confirmed by construction**; the Phase 1 test pins it |
| V7 | All `Type` implementations needed by CoreTI are gob-registered | `internal/types/gob.go` `init()` registers 14 Type implementations plus Kinds and `Scheme`; the M-TAINT-TYPES comment records what happens when one is missed | **Confirmed** |
| V8 | `Type` values are **not** uniformly immutable — `TRecord.TypeName` is mutated in place by the typechecker — but no mutation site runs over *cache-decoded* CoreTI values | `grep -rnE '\.(Name|Element|Fields|Row|TypeName|Args) =[^=]' internal/types/*.go` (non-test) → 5 hits, all `TRecord.TypeName`: `typechecker_substitution.go:155`, `unification_core.go:414,416`, `unification_records.go:73,75`. All five run during unification/substitution **of the module being compiled**, before its CoreTI is cached. The mutation sites' inputs are typechecker-constructed types, not decoded blobs | **Confirmed, with the honest caveat:** in-place mutation EXISTS in `internal/types`; the safety of decode-time sharing rests on the decoded CoreTI's consumers, not on global immutability |
| V9 | Residual sharing risk is real, not hypothetical: (a) `TRecord.Fields` is a `map[string]Type` an in-place editor would corrupt, and (b) V8 proves `TypeName` in-place mutation is an established idiom in the typechecker, so a future consumer could repeat it on decoded values | `types.go:179` `TRecord{Fields map[string]Type; Row Type; TypeName string}`; V8's grep hits | **Open risk — closed by the Phase 1 mutation-guard test over the actual decoded-CoreTI consumer operations, with a documented fallback (decode deep-copies per entry) if it ever fails** |
| V10 | Specializer mutates the CoreTI *map*, not shared Type values | `specialize_clone.go` calls `s.CoreTI.Set(cloned.ID(), substituteType(typ, typeSubst))` — new keys, substituted (new) values | **Confirmed** |
| V11 | Bumping `cacheKeyVersion` invalidates old blobs through the existing authorization check | `cache_key.go` `const cacheKeyVersion = "v5"` with a lineage of exactly such bumps (v1→v2 … v4→v5); `cache_artifacts.go:311` rejects `stamp.Version != cacheKeyVersion` as `artifactFailure("verification", …)` | **Confirmed** — no new invalidation mechanism needed |
| V12 | Chunking would break the stamp contract | `validateArtifactDigests` requires `len(digests) == 4` ("stamp must contain exactly four payload digests"); `artifactPayloadNames()` returns exactly the four payloads | **Confirmed (negative)** |
| V13 | The env-var plumbing pattern exists to copy, and the simplicity gates constrain where it may live | `internal/config/compiler.go` `FSMaxBytes() (int64, bool, error)` over `ParseByteSize` (`bytesize.go`: "THE size parser for every cap", fail-loud); `tools/simplicity_metrics.sh:364` gates `getenv_outside_config ≤ 0` (reads must live in `internal/config`) and `:366` gates `env_vars_documented_pct = 100` (must appear in the reference page) | **Confirmed** |
| V14 | The write-path warning today does not name the byte limit | `cache_runtime.go:warnWrite` prints `stage=%s path=%s: %v`; `artifactTooLarge`'s message is `"artifact exceeds %s byte limit"` — no number, no remedy; the read path does print `scope=%s limit_bytes=%d` | **Confirmed** |
| V15 | Named tests recompile the test module once per test, on a path this doc does not fix | The #1328 triage addendum: `EvaluateNamedTestBodyExprs` → per-body temp file with `TransientRoot: true`, never cached; "raising the cap does not help either" | **Confirmed — scoped out (Non-Goals), tracked in `ailang-core-backlog.md`** |
| V16 | A `NewCacheStore` construction error today does NOT fail the run — it warns and bypasses the cache | `cache_runtime.go:40-44`: `store, err := deps.newStore(projectDir)`; on error → `warnWrite("initialization", …)` and returns a store-less runtime (cache off, compilation continues) | **Confirmed — and it corrects this doc's first draft**, which assumed errors propagate. The malformed-env-value path therefore needs the typed `configError` split in M2, or a bad value would silently disable the cache with a warning |

## Conflict Surface

This change does not touch `internal/parser/`, the type system's semantics, or codegen; it changes an on-disk artifact format and a constructor's config resolution. The conflict surface that matters is the artifact/decode contract and the decoded CoreTI's consumers.

### Positions touched

1. `coretypeinfo.gob`'s byte format (one of four authorized payloads).
2. `NewCacheStore`'s limits resolution (constructor used by every cache-enabled path).
3. The `CACHE_WRITE_FAILED` warning text. Not parsed by production code (the only non-test emitter is `cache_runtime.go`), but asserted by substring in `internal/pipeline/cache_invalidation_test.go` (:449 `…module=answer stage=publication`, :521, :546) and `internal/pipeline/cache_artifacts_test.go` (`CACHE_WRITE_FAILED` + `stage=encoding`). Appending the limit after the existing `stage=… path=…: <err>` fields keeps every substring assertion green.

### What else lives in those positions

- The other three payloads (`core.gob`, `iface.json`, `constructors.json`) share the same codec seam, the same ceilings and the same stamp — untouched.
- `AILANG_NO_CACHE` and `AILANG_CACHE_DIR` also gate cache behavior — orthogonal knobs; the new vars compose (a store is still not built when `NoCache`, and `CacheDir` still relocates it).

### Disambiguation strategy

- Format change vs old blobs: the versioned stamp (`cacheKeyVersion`) + SHA-256 digests; a v5 blob under a v6 key mismatches the stamp and misses once. The authorization property (only stamp-authorized bytes decode) is unchanged.
- Config override vs default: unset/empty means default; any non-empty value is parsed or fails loudly. No precedence chain beyond that (no config-file layering).

### Programs/paths that MUST still work

- `TestCacheArtifacts_ByteLimits` (all arms: stat-before-read, aggregate module scope, write-side aggregate, limit-reader sentinel, hash-before-decode, oversized publication non-authorizing, miss-then-repair) — the ceilings' semantics are unchanged, only their source.
- The motoko session fixture (`internal/executor/motoko/testdata/session_motoko_main_success.jsonl`) embeds a `CACHE_WRITE_FAILED … ARTIFACT_TOO_LARGE` warning — its semantics (warning + fresh compile + success) must remain a possible outcome.
- Any module currently caching cleanly (std modules, largest blob 20 KB per the constant's comment) — round-trip identical modulo the one-time version miss.
- `internal/repl` and `internal/runner/vm.go` consumers of `res.Artifacts.CoreTI` — they receive the same `map[uint64]Type` shape the codec returns; only pointer identity between equal entries changes.

### What deliberately changes

- Every existing cache entry misses exactly once (v5 → v6).
- Equal-typed nodes share one decoded `Type` pointer (memory win; guarded by V8–V10 and the mutation test).
- The `CACHE_WRITE_FAILED` message gains the limit and the knob name (V14) — text change only.

## Deferred Decisions

The following are intentionally left open for the implementer:

- Whether the dedup table is emitted in a deterministic order (sort by key bytes) — affects only blob-byte stability across runs, not correctness; tests that compare blob bytes may want it sorted. **Agent may choose.**
- Whether `encodeCoreTI` streams the per-type gob encodings or buffers them (memory profile during encode of a 20k-entry module) — **agent may choose**; the seam stays.
- Whether the blob/stamp/module ceilings are also surfaced in a `ailang cache` status command — **deferred; not needed for this fix.**
- The exact dedup ratio on the real protocol_test blob (V5's re-measurement) — **sprint-executor measures and reports.**

## Non-Goals

**Not attempted in this feature:**
- **Per-test recompilation of named tests** — the #1328 addendum's batched-compile design (synthetic `__named_test_<k>` entries, isolation fallback). Even with a perfect cache, `ailang test protocol_test.ail` compiles the test module once per test because each body is a distinct transient source. That is the larger share of the 10-minute run and has its own open design decision; tracked in `ailang-core-backlog.md`.
- **Chunking coretypeinfo.gob** — rejected (V12, Design Freeze).
- **Compression (gzip/zstd) of artifacts** — a layer that can be added later on top of dedup; not needed to meet the goal.
- **Cache eviction / size budgeting of the cache directory as a whole** — different problem.
- **Making the stamp ceiling configurable** — it is a 64 KiB JSON document; no operator has a reason to move it.

## Timeline

**Week 1** (2.5 days):
- Phase 1 (codec + version bump + tests), Phase 2 (config + tests + docs), Phase 3 (diagnostic + measurement + fragment)

**Total: ~2.5–3 days across 1 week** (2× the naive estimate of 1.5 days, per the skill's rule of thumb)

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Dedup ratio on the real blob is smaller than the addendum's text-unique ratio suggests (gob per-entry overhead may be small relative to type sizes) | Med | Phase 3 measures on the real module; even a 3× cut takes protocol_test from 21.8 MiB to ~7 MiB, far under 64 MiB; the env vars cover any residual case regardless |
| A consumer mutates a decoded `Type` in place, corrupting shared entries | High (silent wrong types) | V8/V9: in-place `TRecord.TypeName` mutation is an established typechecker idiom, so the guard is a test, not an assumption. The Phase 1 mutation-guard test sweeps the decoded-CoreTI consumer operations (`Get`, `Set`, `ApplySubstitution`, lowering reads); if it finds a mutator, decode deep-copies per entry — blob size, the actual fix, is unaffected |
| Encode-time cost: one gob encode per distinct type + one map of byte keys per entry | Low | 996 encodes for the measured module; the per-entry cost is a map lookup on a string key. Benchmarked in Phase 3 alongside the size measurement |
| Operator sets an absurdly high ceiling and a corrupted cache file OOMs the reader | Med | The limit is the operator's explicit choice, documented as such in the env-var reference; `readBoundedArtifact`'s stat-before-read ordering is unchanged so the failure is still a clean `ARTIFACT_TOO_LARGE`, not a half-read |
| A malformed env value silently disables the cache instead of failing | Med | V16: the `configError` split makes validation failures run failures while keeping filesystem failures warn-and-bypass; both arms pinned by tests |
| Adding two env vars trips the simplicity audit | Low | V13: reads live in `internal/config` (`getenv_outside_config` stays 0) and both are documented in `reference/env-vars.md` (`env_vars_documented_pct` stays 100) |

## Related Documents

<!-- Auto-populated by search on "compile cache large module blobs" -->

**Implemented (may inform design):**
- [m-compile-cache-unverified-artifacts](../implemented/v0_35_2/m-compile-cache-unverified-artifacts.md) — the stamp/authorization contract (exactly-four digests, version binding) this design must not weaken

**Planned (check for overlap):**
- [inline-test-per-case-recompile-1328](../ailang-core-triage/inline-test-per-case-recompile-1328.md) — the #1328 triage doc whose 2026-10-06 addendum measured the CoreTI redundancy (11,639 entries / 996 distinct types / 3.0 MB vs 1.0 MB unique) and proposed exactly this dedup; also scopes out the named-test recompile path this doc deliberately does not fix
- [ailang-core-backlog.md](../ailang-core-backlog.md) — the named-test recompile backlog row

The SimHash/neural related-doc search returned no direct hits on this topic (nearest SimHash matches were unrelated); the documents above were located by reading the #1328 chain, and no planned or implemented doc covers cache-blob limits or CoreTI encoding — the duplicate/coverage gate passes.

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- Issue context: stapledons-godot report (inbox_1791287299418_f86a7ce4), #1328
- `changelogs/unreleased/2026-10-06-mod010-package-file-cache-cap.md` — the already-landed 64/128 MiB raise this doc builds on
- `internal/pipeline/cache_artifacts.go`, `cache_key.go`, `cache_store.go`, `cache_runtime.go` — the surfaces changed
- `internal/config/compiler.go` (`FSMaxBytes`), `internal/config/bytesize.go` (`ParseByteSize`) — the patterns copied

## Future Work

- gzip/zstd layer over the deduped blob for modules whose *unique* content itself is huge
- The named-test batched compile (#1328 addendum) — the remaining ~9 minutes of the reported 10
- Surfacing cache blob sizes and dedup ratios in `ailang cache` status output

---

**Document created**: 2026-10-06
**Last updated**: 2026-10-06
