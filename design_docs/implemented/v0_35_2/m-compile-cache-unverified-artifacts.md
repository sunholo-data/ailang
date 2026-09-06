> **STATUS: IN SPRINT — M1, M2 and M3 of 4 landed; M4 remains. Banner corrected 2026-09-05, iteration 332.**
>
> **`D-55` REMAINS OPEN.** What unparked this design was not an answer to it but the row's OWN
> pre-registered default — *"(a), applied at the next iteration ... as a controller routing call
> rather than as a ruling"* — which fired at iteration 329 because `D-55` was still unanswered then.
> The loop may not resolve a ledger row on its own behalf, so the scope decision is APPLIED while the
> question stays answerable, and a later (b) or (c) answer supersedes this sprint's scope.
>
> This banner previously read *"NOT approved and MUST NOT be executed until `D-55` is answered"* and
> was left standing for three iterations while M1 (`3d7bbfad8`), M2 (`f5edd569a`) and M3
> (`d14bd42cc`) all shipped under that default — a tracked document contradicting the tree. The
> executor of M3 flagged it; it is corrected here rather than left for a fourth.
>
> The design went to a 3-reviewer quorum twice, was revised once between rounds, and the round-2
> synthesis was BLOCKED. The record below is kept verbatim: the one live objection is real and
> unanswered, and it is exactly what `D-55` asks Mark to rule on. The defect addressed is confirmed
> and public ([#1046](https://github.com/sunholo-data/ailang/issues/1046)), which stays OPEN until M4.
>
> **Quorum record** (the machine JSON lives at `.ailang/state/mission-quorum/` on the rig and is
> **gitignored**, so this table is the tracked record — `.gitignore:82` ignores `.ailang/`):
>
> | round | artifact | `gpt5-6-sol` | `gemini-3-1-pro` | `oc-glm-5-2` | controller | synthesis |
> |---|---|---|---|---|---|---|
> | 1 | `…2026-09-05T11-31-14Z.json` | **ABSENT (budget)**, re-run alone at a raised cap → **reject** | pass | pass | pass | PROCEED-at-N−1, then reject on re-run |
> | 2 | `…2026-09-05T11-43-52Z.json` | **reject** | pass | **reject** | pass | **BLOCKED** |
>
> - Designer: `codex:gpt-6-astra`. Reviewers were **pinned explicitly** to
>   `gpt5-6-sol,gemini-3-1-pro,oc-glm-5-2` to keep astra — which had become the quorum's OpenAI seat
>   the same day — from reviewing the document it had just authored. Both artifacts confirm astra
>   reviewed nothing.
> - **Round-1 → round-2 revision** answered `gpt5-6-sol`'s first objection (axiom A5 claimed on a
>   fixed file COUNT with no byte ceiling): §Explicit byte ceilings now sets 16 MiB per blob,
>   64 KiB for the stamp and 32 MiB per module, measured against a 27-module survey.
> - **`oc-glm-5-2`'s round-2 objection is CLOSED**, after the doc was written: it held that V54's
>   serve-api flags were never really verified. The iteration-328 evaluator invoked the binary —
>   `ailang serve-api --help` plus a live MCP `initialize` RPC — and all three flags (`--mcp`,
>   `--routes-only`, `--no-feedback-tool`) are real, correctly spelled and functional.
> - **The one live objection is `gpt5-6-sol`'s round-2 point**, which is `D-55`: hashes prove
>   consistency, not compiler provenance. Note it is a **pre-existing** property of HEAD that this
>   design strictly reduces — at HEAD `LoadArtifacts` gob-decodes with 0 byte ceilings and 0 hash
>   checks — and that Go's `encoding/gob` already caps preallocation against remaining input.

# M-COMPILE-CACHE-UNVERIFIED-ARTIFACTS — bind executable cache artifacts to their authorizing key

**Status**: In sprint — M1-M3 of 4 landed under `D-55`'s pre-registered default (iteration 329), M4 remains; see the banner above for what that default is and is not
**Target**: v0.35.1
**Priority**: P0 — source edits must determine the program served
**Estimated**: 4 working days, one sprint (four independently testable milestones)
**Dependencies**: None
**Created**: 2026-09-05
**Author**: V1 mission designer
**Verified baseline**: `087fbea631a0b80556baa034b499fbdae33e76d2` [V01]

## Axiom Compliance

| Axiom | Score | Justification |
|---|---:|---|
| A1: Determinism | +1 | A cache hit must represent the source snapshot being compiled. |
| A2: Replayability | +1 | Restarts must not substitute an earlier compiled program. |
| A3: Effect Legibility | 0 | No language effect change. |
| A4: Explicit Authority | +1 | The requested module and computed key authorize the bytes decoded. |
| A5: Bounded Verification | +1 | Enforce 16 MiB per blob, 64 KiB for the stamp, and 32 MiB total per module before hashing/decoding; decode only compiler-produced, hash-verified payloads under the explicit trust assumption below. |
| A6: Safe Concurrency | +1 | Mixed writes must produce a miss, never a mixed executable. |
| A7: Machines First | +1 | Stable diagnostic prefixes identify cache failures. |
| A8: Minimal Syntax | 0 | No syntax change. |
| A9: Cost Visibility | 0 | No billing or resource-accounting change. |
| A10: Composability | 0 | No caller-facing pipeline configuration change. |
| A11: Structured Failure | +1 | Unverifiable artifacts become explicit misses; optional persistence failures warn. |
| A12: System Boundary | 0 | No new capability or service. |

**Net score: +7.** No proposed violation of A1, A3, A4, or A7.

## Problem Statement

The reporter describes an MCP server registering six of seven annotated exports for roughly a month, using v0.34.0-430-g74c717b07. Its artifact directory was dated 2026-05-03 while source was dated 2026-08-25 (114 days); 28 entries lagged their sources. Copying the source to another path worked, six repeated runs at the original path failed identically, and deleting the single artifact directory restored the missing tool. These historical observations are **inherited from controller, not re-derived**; the reporter's original checkout is unavailable [V02]. The failure mechanism, rather than the historical write failure, is reproduced here [V22].

### Current behavior, verified at baseline

1. **A: manifest validity does not establish artifact validity.** The pipeline computes a content/dependency/compiler key and `Lookup` compares it with the manifest entry [V03]. `LoadArtifacts(moduleID)` instead reads from `modules/<sanitizeModuleID(moduleID)>`; it has no expected-key parameter or artifact key comparison [V04]. Its four files are two gob files and two JSON files, decoded independently [V05]. The pipeline updates the in-memory manifest entry before calling `StoreArtifacts`, discards that error, and later discards the error from saving the manifest [V06]. A successful manifest save following a failed artifact write can therefore authorize old or partially replaced blobs; successful decoding skips compilation [V07]. **An interruption merely between `Store` and `StoreArtifacts` does not itself persist the new manifest**: `Store` is in-memory, and `Save` is later [V06]. The dangerous persisted state is real, but that particular crash-only explanation would overstate the code.
2. **B: source-read errors collapse source identity.** The key site initializes source to an empty string and ignores `os.ReadFile` failure; its comment records the earlier `mod.Path` versus `mod.File.Path` recurrence [V08]. Embedded stdlib uses a synthetic `<embedded>/std/...` path and preloaded bytes locally inside `Load`; the returned `LoadedModule` does not retain those bytes [V09, V10]. A fresh temporary project reproduced identical empty-source keys for `std/option` and `std/result`, with different interface digests [V23]. Ordinary on-disk dependency edits still changed execution from 3 to 41 [V24]. This is conditional identity loss, not universally broken source invalidation.
3. **C: clearing entries leaves executable blobs.** `Clear()` replaces the manifest and saves it, without deleting `modules/` [V11]. **Command correction:** at this HEAD the dispatcher exposes `ailang cache compile-clear`; `ailang cache clear` is not a recognized case [V12]. A temporary-cache run of `compile-clear` reported four cleared entries while `modules/` remained [V25].

In the API server, `registerModule` builds exports from `loaded.Iface`, then attaches annotations from `loaded.File`; the route extractor only updates names already in that export list [V13]. Missing exported routes thus receive no diagnostic in that loop [V14]. Existing registration validation concerns whole modules dropped outside the base path [V15]. The serve-api pipeline configuration lacks `NoCache`, and its command/server sources do not read `AILANG_NO_CACHE` or expose a cache-bypass flag [V16]. The bounded MCP reproduction below produced `6 → 7 → 6 → 7` tools as source was updated, old artifacts restored under the new manifest, and that directory removed [V22].

## Goals

- Require verified module identity, cache key, and coherent artifact bytes before any compilation skip.
- Hash the same source snapshot the loader parsed, including embedded stdlib.
- Preserve successful fresh compilation when optional cache persistence fails, with visible stderr diagnostics.
- Make compilation-cache clearing remove both manifest entries and all module artifacts.
- Refuse to serve a local module whose exported, annotated route is absent from its compiled interface.
- Pin observable execution and MCP tool membership, not merely manifest strings.

## High-Impact Decisions

| Question | Decision | Consequence |
|---|---|---|
| D1: directory identity | Keep the existing directory layout; add `artifacts.json` containing format version, exact module ID, cache key, and SHA-256 of each of four fixed blob names. | Small migration; legacy blobs cannot qualify as verified hits. |
| D2: transactions | No manifest/artifact transaction or multi-directory architecture. Verify all bytes on read; publish the stamp last. | Persistence failure costs recompilation rather than correctness. |
| D3: write failure | Warn on stderr, retain the freshly compiled in-memory unit, and do not publish its manifest entry on artifact failure. | A read-only cache directory cannot make a correct source build fatal. |
| D4: source identity | Add `LoadedModule.SourceContent *string`, set from the exact parser input on every successful loader read. | No second filesystem read, no synthetic-path heuristics, explicit unknown-source state. |
| D5: migration | Bump `cacheKeyVersion` from `v3` to `v4` [V17]. | One cold compile per module; also protects same-commit/dev builds. |
| D6: clear and inspection | Repair `compile-clear`; defer `cache verify`/`cache doctor` and a `clear` alias. | No CLI namespace expansion in this sprint. |
| D7: route mismatch | Return an actionable registration error before publishing a local module, for exported `@route` functions missing from the iface. | A residual integrity failure is visible before clients receive an incomplete surface. |

These are the proposed design decisions, not self-approval. The controller owns design approval and sprint execution authorization.

## Solution Design

### 1. Artifact verification at the read boundary

Change the internal methods together with their callers:

```go
StoreArtifacts(moduleID, cacheKey string, cm *CachedModule) error
LoadArtifacts(moduleID, expectedCacheKey string) (*CachedModule, error)
```

Proposed `artifacts.json` schema:

```json
{
  "version": "v4",
  "module_id": "api/entry",
  "cache_key": "<computed key>",
  "sha256": {
    "core.gob": "<hash>",
    "coretypeinfo.gob": "<hash>",
    "iface.json": "<hash>",
    "constructors.json": "<hash>"
  }
}
```

The reader must first read the stamp through the bounded reader specified below, then parse it and require the current version, nonempty exact expected key, exact requested module ID, and exactly the four required digest entries. Unknown/missing versions, absent stamps, malformed JSON, wrong IDs/keys, missing blobs, wrong hashes, and decode errors all return an invalid-artifact error. Metadata must not supply filesystem paths: read only the four hard-coded basenames.

**Explicit byte ceilings (inclusive):** define `maxArtifactBlobBytes = 16 << 20` (16,777,216 bytes) for each of the four blobs, `maxArtifactStampBytes = 64 << 10` (65,536 bytes) for `artifacts.json`, and `maxModuleArtifactBytes = 32 << 20` (33,554,432 bytes) for the stamp plus all four blobs together. These are fixed compiler constants, not values supplied by the cache. On 2026-09-05, `ls -l` and a read-only size survey of 27 complete cached modules found `core.gob` sizes 962–8,218 bytes; maxima for CoreTI, iface, and constructors were 20,387, 5,845, and 361 bytes, respectively. The largest four-blob sum was 34,811 bytes (`std__result`) [V49, V50]. Thus the blob ceiling exceeds the largest observed blob by over 800× and the aggregate ceiling leaves over 900× the largest observed four-blob sum after reserving the full stamp budget (`(33,554,432 - 65,536) / 34,811 > 900`). The stamp has only version/module/key and four digests; 64 KiB leaves generous room for module names. This sample is not a universal module-size bound: larger legitimate sources remain compilable, but artifacts beyond these limits are ineligible for caching.

**Enforcement sequence:** use one bounded-read helper for the stamp and every blob, in fixed order (stamp, Core, CoreTI, iface, constructors). Track accepted byte lengths with `int64`; before each read compute `remaining = maxModuleArtifactBytes - acceptedBytes` and `limit = min(fileCeiling, remaining)`. Open the file once, call `f.Stat()` on that handle, require a regular file, and reject immediately if its reported size exceeds `limit`. Do not allocate using that reported size and do not use `os.ReadFile` at this boundary [V51]. If stat permits proceeding, the authoritative read is `io.ReadAll(io.LimitReader(f, limit + 1))`, followed by closing the handle. If the returned length exceeds `limit`, reject without hashing or decoding. The extra byte is an overflow sentinel: a file of exactly `limit` bytes is valid only after the underlying reader reaches EOF before returning byte `limit + 1`; EOF caused by exhausting the limiter returns that extra byte and is rejected [V52]. Non-EOF read errors are invalid artifacts, never truncated successes.

The stat is only an early rejection optimization. Growth between stat and read still encounters the sentinel; replacement of the path cannot change the already-opened handle. Shrinkage or in-place changes either yield an in-budget byte snapshot matching the stamp or fail its hash. No stat/read race can authorize uncapped input. An accepted attempt reads at most 32 MiB including the stamp; a rejected attempt reads at most that budget plus one sentinel byte, often zero payload bytes after an oversized stat. Hashing covers at most the accepted blob bytes, stamp JSON parsing receives at most 64 KiB, and raw buffering is bounded by those lengths plus the bounded allocation/copy overhead of `io.ReadAll`. This is a per-module work/memory bound, not a filesystem latency guarantee or a bound on whole-project compilation.

An over-limit stat or sentinel returns an invalid-artifact error with stable reason `ARTIFACT_TOO_LARGE`. The pipeline emits, for example, `CACHE_INVALID module=api/entry path=.../core.gob reason=ARTIFACT_TOO_LARGE scope=blob limit_bytes=16777216; recompiling`. Set `scope=stamp`, `blob`, or `module` according to the binding ceiling (`module` when remaining aggregate capacity is smaller; the file scope wins ties). Keep existing once-per-module warning behavior. This is always a verified cache **MISS**, never a compile error; fine source must still compile and execute. The proposed reason is unallocated in the current code [V53].

Read each blob once through that helper, verify SHA-256 over those bytes, and decode **those same byte slices** only after all four hashes pass. Never verify a file and then reopen it for decoding. No mtime participates in validity. A valid old stamp left beside a partial new write will fail a hash or key check; a reader interleaving with a writer either obtains a coherent validated snapshot or gets a miss. Hash equality assumes accidental corruption/concurrency, not an adversary who can rewrite both code and cache metadata.

**Decode bound and its one trust assumption:** the accepted stamp's digests must originate from this same compiler version's trusted serializer, rather than being forged or coherently rewritten together with malformed payloads. With the usual SHA-256 collision-resistance premise, verifying all capped byte slices before any payload decode means the decoders only receive the exact, finite, compiler-produced encodings; corrupt gob length/count fields cannot reach a decoder. Work and reconstructed allocations are therefore limited to those compiler-produced objects represented within the byte budgets, not arbitrary corrupt length claims. This does **not** assert that decoded memory equals serialized byte length or that a byte cap alone hardens gob: Go explicitly documents that its decoder is not hardened for adversarial inputs and its size sanity limits are not configurable [V52]. The capped stamp itself is parsed as metadata before trust is established; it cannot authorize decoder input without the key/version/identity/hash checks. Authenticated artifacts or a sandboxed hostile-input decoder remain outside the accepted accidental-corruption threat model.

The pipeline must pass its newly computed `moduleCacheKey`, never a key obtained from the artifact stamp, to `LoadArtifacts`. It must count a compilation skip only after successful verification and decoding. A manifest hit with invalid artifacts emits one `CACHE_INVALID` diagnostic per module per invocation, containing module, cache path, and reason, then recompiles. Empty caches need no warning.

**Sanitization collision decision:** `a/b` and `a__b` both map to `a__b` today [V18]. This proposal deliberately leaves that directory-name collision in place. The exact `module_id` comparison prevents cross-module consumption even if the two requested cache keys happen to be equal. Two colliding IDs can still evict each other's artifacts and cause recompilation; eliminating that storage/performance collision is a follow-up. Tests must use equal keys to prove that module identity, rather than incidental key inequality, supplies the guard.

**Migration:** v4 manifests start empty when v3 is encountered, using the existing version rejection behavior [V17]. Legacy directories remain inert until overwritten or cleared. Even a hand-advanced v4 manifest pointing at unstamped legacy artifacts must miss. No eager deletion, migration command, or second cache tree is required.

### 2. Publication, partial writes, and optional-cache failures

Serialize all four blobs to memory before modifying their directory. Check their lengths against the same blob and aggregate ceilings before hashing. Compute hashes from these serialized bytes, then serialize the stamp and check its own and the final aggregate length before any file write. An oversized freshly compiled module returns a nonfatal `StoreArtifacts` encoding-stage error (`ARTIFACT_TOO_LARGE`), so the pipeline warns with `CACHE_WRITE_FAILED`, retains fresh compilation, and publishes no new manifest entry. These checks bound cache verification, not source compilation/serialization itself. Write the four files, then write the stamp to a unique temporary file in that same module directory and rename it into place last. Check write, close, and rename errors; clean up this invocation's temporary stamp on failure. Do not reuse a shared temporary filename. If platform/filesystem behavior interrupts publication, read verification must still be sufficient to reject an incomplete record; correctness must not depend on a universal rename-atomicity promise.

Only after successful `StoreArtifacts` should the pipeline call `Store` for the new manifest entry. Leave an earlier manifest entry intact when the write fails. It either has the wrong source key or points at files that must pass the old stamp's hashes. Also surface cache creation and manifest-save failures, because otherwise a read-only directory would still fail invisibly at a different point [V06, V19].

Use a compact stderr warning such as:

```text
CACHE_WRITE_FAILED module=api/entry stage=artifacts path=...: permission denied; using fresh compilation
```

Stages must distinguish initialization, encoding, artifact publication, and manifest save. Cache warnings must not enter program stdout, MCP JSON-RPC stdout, or effect traces. Keep warning output active without `DebugCompile`. Aggregate initialization/save errors once per invocation; artifact errors once per affected module. Do not return a compile error solely because persistence failed. A verified read-only cache hit may still be used; a read-only miss must compile successfully and warn that it could not be cached.

**Why a transaction is unnecessary:** if artifact publication fails, no new entry is intentionally published, and a previously divergent manifest still cannot authorize mismatching bytes. If artifacts succeed but the manifest save fails, the next lookup may miss or see an old key; any matching lookup still must pass artifact verification. A process stopping after any individual write can leave garbage or lose a warm hit, but cannot turn that incomplete write into permission to execute stale bytes. A key-only stamp without blob hashes is insufficient: overwriting some files while an old stamp remains could otherwise admit a mixed same-key or old-key record.

No fsync durability contract, cross-process manifest locking, or directory transaction is added. Lost manifest updates and orphan temporary files may waste work; validated read correctness is the sprint's boundary.

### 3. Source snapshot ownership, including embedded stdlib

The loader already has exact bytes for both normal files and embedded stdlib immediately before lexer creation; normal disk-read failure returns an LDR001 report before parsing [V09]. Convert once to `sourceText`, give that string to the lexer, and retain `&sourceText` in the returned `LoadedModule`. The pointer distinguishes a known empty string from unavailable content. Treat this field as immutable and exclude it from executable artifact serialization.

At the cache-key site, require `mod.SourceContent != nil`, and pass `*mod.SourceContent` to `ModuleCacheKey`. Remove the opportunistic `os.ReadFile` and its empty-string default entirely. Do not use `mod.Path`, `mod.File.Path`, an AST pretty-printer, or a special `<embedded>` prefix as a substitute for source identity.

- **On-disk source:** the initial loader read must succeed; failure remains a source-loading error. A later disk edit cannot cause the key to describe different bytes from the parsed AST. A subsequent fresh pipeline invocation observes that edit.
- **Embedded stdlib:** use the same retained content field. A real file is neither required nor reread.
- **Synthetic/preloaded modules without a source snapshot:** explicitly bypass both cache lookup and cache publication for that module and emit `CACHE_SOURCE_UNAVAILABLE`. Continue compilation if its existing AST is otherwise sufficient. Never hash unknown content as `""`. This is a visible loss of an optimization, not replacement of source data.

The field is needed on the loader object at the key site. Result assembly currently creates new LoadedModule objects from compiled units [V47]; deliberately do not copy the source snapshot into those runtime objects. No runtime/Core/iface serialization change is needed. Preserve module IDs, resolved AST paths, package resolution, prelude injection, and declaration validation.

### 4. Clearing and API registration diagnostics

`CacheStore.Clear()` must save an empty current-version manifest and remove the entire `modules/` subtree under its resolved compilation-cache root, including legacy directories, v4 stamps, partial blobs, and abandoned stamp temporary files. If either operation fails, return a contextual error; the CLI must not print its success message. Empty or missing `modules/` is success. Do not delete the cache root's siblings, package caches, brain data, or another session's override directory. A failed deletion may leave bytes on disk but must not be reported as complete. Clearing during active writers is not a global barrier; operators should stop writers if they require the directory to remain empty.

The repaired user command is `ailang cache compile-clear`, honoring `AILANG_CACHE_DIR` through `NewCacheStore` [V12, V20]. Keep the current command spelling; adding `cache clear` would broaden a CLI namespace that also manages brain data [V12]. A later `compile-verify` could report stamp/key/hash failures. An artifact older than its source is only a hint: restored timestamps and content-preserving edits make age a poor validity test. A doctor command is not necessary to establish the invariant and is out of this sprint.

In `registerModule`, after the local-file filter but before the idempotent-return path and map publication, inspect `loaded.File.Funcs`. For each `IsExport` function with a `route` annotation, require `loaded.Iface.Exports[fn.Name]` to exist and be non-nil. If the iface itself is nil and an exported route exists, report the same inconsistency rather than silently returning. Return an error naming source path, module, and function, with `CACHE_ROUTE_IFACE_MISMATCH` and the compilation-cache remedy. Do not silently retry or add the missing function directly to the export list. `@nomcp` and `@noexpose` must not excuse a missing iface export; their filtering remains downstream. Private annotated functions remain outside this narrow invariant.

This check must not alter whole-module outside-base-path drop handling or its existing override [V15]. No serve-api cache-bypass flag is required to make the cache safe; bypass support is explicitly deferred.

## Conflict Surface

This is an AILANG pipeline correctness change. Proposed edits and compatibility boundaries are enumerated below; rows describing baseline behavior carry evidence IDs.

| File / surface | What can be disturbed | Boundary and validation |
|---|---|---|
| `internal/pipeline/cache_store.go` | Manifest lookup, four serializers, artifact loading, environment root selection, clear [V03–V05, V11, V20] | Keep manifest entry fields and blob encodings; add one stamp and expected-key arguments; preserve existing round-trip assertions. |
| `internal/pipeline/cache_key.go` | Cache version and compiler/source/dependency key algebra [V17] | Bump v3 → v4 only; retain key inputs and dependency sorting. |
| `internal/pipeline/pipeline_module.go` | Cache read/skip path, write order, hit accounting, warning stdout discipline, source identity [V03, V06–V08, V19] | All load failures compile; all persistence failures retain fresh units; `NoCache` continues to bypass cache work. |
| `internal/loader/loader.go` | LoadedModule layout, source loading, embedded fallback, exports/prelude construction [V09, V10, V26] | Add optional immutable source snapshot; no path or resolution-policy change. |
| `internal/loader/stdlib_resolver.go`, `std/embed.go` | Filesystem versus embedded source choice [V09, V27] | Read-only compatibility surface; do not change resolver precedence or embedded files. Pin embedded-source content in loader/pipeline tests. |
| `internal/pipeline/pipeline_module_compile.go` result assembly [V47]; `internal/runtime/`, `internal/embed/` consumers | Fresh/cached Core, type information, constructors, and iface delivered to execution | Preserve existing payload types; source snapshot is used before result assembly. No runtime or interpreter semantics edits. |
| `.ailang/cache/compile/manifest.json` / override equivalent | Manifest format compatibility [V17, V20] | Keep schema fields; version v4 rejects v3; no tracked cache fixtures. |
| `cmd/ailang/cache.go`, `cmd/ailang/cache_compile.go` | Compile-clear dispatch/error/success behavior and adjacent brain commands [V12] | Retain spelling and root selection; repair via `Clear`; CLI subprocess regression pins exit status and deletion. |
| `internal/apiserver/module_entry.go` | Registration identity, filters, idempotency, iface/AST reconciliation [V13–V15] | Add only the exported-route invariant; private functions and intentional exposure filters keep their semantics. |
| `internal/apiserver/load_project.go`, `server.go`, `routes.go`, `mcp.go`; `cmd/ailang/serve_api.go` | ModeCheck load, module preloading, route attachment, MCP membership [V13–V16, V28] | Compatibility surface except registration helper/tests; actual MCP stdio regression exercises the whole chain. No bypass/config redesign. |
| `internal/eval_harness/runner.go`, `cmd/ailang/main_run_exec.go` | Eval subprocess execution and run cache bypass [V29] | No harness edit. Keep stderr warnings out of evaluated program stdout; test source edits and `AILANG_NO_CACHE=1` with isolated roots. |
| `internal/executor/motoko/motoko.go:338`, per-task cache setup | Motoko task cache isolation [V30] | No motoko core/adapter change. Preserve `AILANG_CACHE_DIR` placement and make clear touch only that override's compile subtree. |
| `internal/pipeline/cache_store_test.go`, `cache_invalidation_test.go`, `cache_key_test.go`; `internal/loader/loader_test.go`; `internal/apiserver/module_entry_test.go`; existing `cmd/ailang/serve_api_mcp_surface_test.go` | Regression fixtures and changed internal method signatures | Update callers in the same milestone as the signature change; add behavior tests beside their owners. |

Implementation may split artifact helpers into `internal/pipeline/cache_artifacts.go` if needed to keep the existing serializer file manageable; that is the same responsibility, not another milestone. No parser, lexer, type-system, effect-runtime, or motoko core edits are authorized by this design.

## Testing Strategy

### Existing tests: exact coverage and blind spots

Both requested test files were read in full (116 and 464 lines respectively) [V31]. These are assertion-level findings, not results from running artificial mutants.

| Existing test | What it actually pins | What it does not pin | Evidence |
|---|---|---|---|
| `TestCacheKey_InvalidatesOnSourceEdit` | Runs ModeCheck twice for `42 → 99`; requires nonempty, unequal manifest `cache_key` strings. | Never reloads artifacts, observes artifact rewriting, evaluates 99, or starts MCP. Publishing a new key with old blobs survives these assertions. | V32 |
| `TestCacheStore_RoundTrip` | Persist/reload manifest; right-key hit, wrong-key/missing-module miss; iface digest and normalized JSON survive. | No artifact directory is involved. | V33 |
| `TestCacheStore_Clear` | Stores two manifest entries, saves, calls clear, observes in-memory count zero. | Creates no artifacts, does not inspect `modules/`, does not reload cleared manifest. | V34 |
| `TestCacheStore_CorruptedManifest` | Invalid JSON produces a fresh zero-entry store without constructor error. | Does not simulate valid JSON authorizing wrong blobs or legacy-format migration. | V35 |
| `TestCacheStore_Stats` | Two entry counts and 10 + 20 = 30 recorded milliseconds. | No disk artifact integrity assertion. | V36 |
| `TestCacheStore_ArtifactRoundTrip` | One successful write/read; checks Core decl count/flags/Let name and ID, export metadata, CoreTI size/int type, iface module/digest, constructor arity, record alias name, alias params, constructor type-param count. | Does not assert every nested value; no second write, expected key, identity collision, partial failure, or pipeline behavior. | V37 |
| `TestCacheStore_ArtifactRoundTrip_DiverseExprTypes` | Six decls; selected Match, VarGlobal, DictRef fields survive. | No overwrite/error injection or manifest coordination. | V38 |
| `TestNewCacheStore_HonorsEnvOverride`, `TestNewCacheStore_EmptyEnvFallsBackToProjectDir` | Manifest placement under nonempty override versus default directory. | Do not verify artifact placement or that clear preserves sibling caches. | V39 |

The seven key-unit tests separately exercise deterministic hashing, different source/compiler/dependency inputs, dependency order, commit change, and nil/empty dependency equivalence; they never connect artifact reads to authorization [V40]. `TestLoad_EmbeddedStdlibFallback` checks successful load/exports or types, missing-module failure, and loader pointer reuse, not retained source bytes or pipeline keys [V41]. Cache-focused test call-site searches found no artifact poisoning test in apiserver; the same search's positive controls find artifact round trips in pipeline and route tests in apiserver [V42]. The existing CLI MCP test exercises feedback-tool presence with a positive `status` control, but does not edit source or poison artifacts [V48]. The baseline cache test selection passes despite the independently reproduced stale tool [V43, V22].

### New tests and the specific mutations they must kill

Every listed test name is a required implementation deliverable, not a claim that it already exists. All fixtures must be source strings or temporary files created by tracked Go tests. Do not put the durable test or its expectations in a build/results directory.

| ID / proposed test | Construction and required assertions | Mutation killed; why existing coverage does not already kill it |
|---|---|---|
| T1 `TestCacheArtifacts_Authorization` | Table: valid stamp; missing/corrupt stamp; wrong version; empty/wrong key; wrong module ID with **equal key**; `a/b` versus `a__b`. Verify expected hits/misses. | Remove key/version/module checks or trust the stored key. V33/V37/V38 only use matched single writes without expected-key authorization. |
| T2 `TestCacheArtifacts_PartialWrite` | Seed version A, interrupt publication at each stage (encode each payload, each file write, stamp close/rename), and test loads with both A and B keys. Mutate one blob at a time to other well-formed serialized bytes; test missing digest entries. | Accept a key-only stamp, omit any blob hash, or decode unverified bytes. V37/V38 never fail/interrupt a write. The control must still load an untouched A snapshot. |
| T3 `TestCacheArtifacts_ReadSnapshot` | Deterministic instance-local I/O hook changes disk contents after bytes are returned to the reader; assert decoded result is exactly the verified snapshot or a miss, never later unverified bytes. | Reopen blobs after hash checks. Existing round trips have no reader/writer interleaving [V37/V38]. |
| T4 `TestCacheArtifacts_Migration` | Explicit v3 manifest plus valid legacy blobs produces cold compilation under v4; manually current-version manifest plus unstamped blobs also misses. Assert new stamp version and observable current value. | Omit version bump or accept legacy artifact fallback. Corrupt-JSON test V35 is not a version fixture. |
| T5 `TestCachePipeline_WriteFailure` | Force artifact publication failure after fresh compilation through an instance-local dependency seam; separately fail cache initialization and Save. Require fresh result/exports, stderr stage diagnostic with DebugCompile false, no new manifest authorization on artifact failure, no stdout contamination. Repeat after restoring writes and require a valid warm hit. | Discard any persistence error, publish the entry on failed artifacts, or make optional persistence fatal. Existing tests never inject failures [V32–V39]. |
| T6 `TestCacheSource_ExactSnapshot` | Loader tests compare retained text with file bytes and embedded `std.FS` bytes; pipeline test changes/removes the disk file after loading, checks key against retained text; nil snapshot bypasses reads/writes with diagnostic, known empty snapshot remains distinct. | Retain only disk text, use empty embedded text, reread disk at key time, or treat nil as empty. V32 changes readable source between complete invocations; V41 checks neither source bytes nor keys. |
| T7 `TestCachePipeline_EmbeddedKeys` | Force embedded stdlib, assert its synthetic path, compute keys from retained nonempty option/result content and actual dep digests; assert manifest keys match those computations and differ from the empty-source key. | Reintroduce the silent empty-source branch specifically at the pipeline site. Key-unit tests V40 would still pass; V41 does not inspect pipeline keys. |
| T8 `TestCachePipeline_SourceEditBehavior` | Compile/run a dependency returning 2, then 40, requiring output 3 then 41. Also require updated Core behavior and a subsequent verified hit; run with NoCache and require identical output and no persistence. | Keep an old blob while advancing its manifest, or accidentally disable all cache hits to make correctness tests pass. V32 asserts only key strings. |
| T9 `TestCacheStore_ClearArtifacts` | Create valid entries, orphan legacy dirs, a partial write, and a stamp temp; clear, reopen, require zero entries and absent modules subtree. Include override/default placement and sentinel sibling files. Inject removal and Save errors, require errors rather than success. | Restore manifest-only clear, swallow deletion failure, or delete too wide a root. V34 makes no artifact tree; V39 only checks manifest location. |
| T10 `TestCompileCacheClear_Artifacts` | CLI subprocess with isolated cache root, seeded artifacts; require exit 0, success line, absent modules, preserved sibling sentinel. Failure fixture must exit nonzero without success line. | Dispatcher/helper uses a different clear path or prints success on failure. No artifact/compile-clear subprocess test appeared in the audited search [V42]; the existing MCP subprocess test only checks feedback-tool suppression [V48]. |
| T11 `TestRegisterModule_RouteIfaceMismatch` | Local AST with exported annotated f7 and iface lacking f7 (also nil iface); require named diagnostic and no map publication, including the idempotent-registration case. Matched iface, private annotated function, outside-basePath handling, and intentional MCP exposure filtering remain controls. | Retain silent missing-name skip or bypass validation on repeat registration. Existing route tests assume matching exports; drop validation concerns a different whole-module condition [V14/V15/V42]. |
| T12 `TestServeAPI_DivergentCacheTools` | Fresh subprocesses, six routes then seven, restore six-route artifacts **including their old stamp** while retaining new manifest. MCP initialize/list must return seven exact names after repair; f7 call must execute expected body; second restart is a verified hit. Also copy a single stale blob under a fresh stamp. | Trust manifest alone or omit one blob hash; a registration error instead of successful recompilation also fails. V22 establishes the failing baseline; existing V32–V41 and the MCP surface test V48 never construct this state. |
| T13 `TestCacheArtifacts_ByteLimits` | Warm a valid module, then replace `core.gob` with a sparse file of `maxArtifactBlobBytes + 1` bytes, retaining the valid manifest/stamp. Require `LoadArtifacts` to return no module and reason `ARTIFACT_TOO_LARGE`, scope `blob`, with no payload read/decode after the oversized stat. A pipeline invocation must record a verified MISS, emit `CACHE_INVALID` with that reason, successfully compile/execute the expected source result, and repair the cache for a subsequent verified hit. Table cases also cover an oversized stamp and aggregate overflow with each blob individually within its ceiling; pad the two JSON blobs with legal whitespace and update test-stamp hashes to isolate the aggregate guard. With private instance-local smaller budgets, test exact-limit/clean EOF acceptance, limit-plus-one rejection, growth after stat caught by the sentinel, shrink/hash mismatch, and no payload decode before all hashes pass. Pin production constants separately; test over-limit publication warns and retains fresh compilation without publishing a manifest entry. | Removing the blob ceiling must turn the required reason/read-count assertions red even if the later hash check still causes recompilation; removing the aggregate or stamp ceiling fails its corresponding scope assertion. Keeping only stat fails the growth-after-stat case; using `LimitReader(limit)` without the sentinel fails the limit-plus-one case. Moving decode ahead of hashes fails the zero-decode assertion on malformed, hash-mismatching bytes. Existing round trips do not exercise oversized files [V37/V38]. |

T6 spans loader and pipeline packages with the same test name and complementary assertions. Hooks must be private, per-instance/function-argument dependencies with real filesystem defaults, not environment switches, exported test-only API, or mutable global function replacement. Permission tests should inject `os.ErrPermission` deterministically; a real read-only-directory case is supplementary because privileged test runners can bypass mode bits. For partial writes, manually constructing intermediate directories supplies interruption coverage without sleeping or killing a process nondeterministically.

## Implementation Plan

### M1 — verified artifact loads and visible publication failure (2 days)

Modify cache store/key/pipeline code and existing store tests together. Add stamp verification, four hashes, stamp-last publication, expected-key arguments, manifest-after-artifacts ordering, version v4, and stderr cache error handling. Implement T1–T5 and T13, including bounded reads, publication-size checks, and the stat/read-race and overflow diagnostics. The ceiling revision adds 0.5 day here. Preserve serializer round-trip assertions; replace only method-call signatures in those tests. The tree must compile and pass these tests at the commit boundary.

### M2 — loader-owned source identity (0.75 day)

Add immutable source snapshot retention in `internal/loader/loader.go`; consume it in the pipeline without disk rereads. Implement T6–T8, retaining the original manifest-key regression test. Embedded success, unreadable source failure, nil-source bypass, and on-disk behavioral invalidation must all be tested in this commit.

### M3 — complete compilation-cache clearing (0.5 day)

Repair `Clear()` and implement T9–T10, with failure propagation and isolated-root preservation. Place the CLI recovery test in existing `cmd/ailang/serve_api_mcp_surface_test.go`, alongside the server recovery fixture for M4 [V48]. Keep `compile-clear` command spelling and brain behavior. This commit is independent of API diagnostic work.

### M4 — route integrity diagnostic and MCP regression (0.75 day)

Add the narrow registration invariant and T11–T12; bank the subprocess regression in existing tracked path `cmd/ailang/serve_api_mcp_surface_test.go` with its CLI fixtures, and registration assertions in existing `internal/apiserver/module_entry_test.go`. Perform final package/boundary checks. Do not replace the seven-tool success expectation with merely an error expectation.

**Total: 4 days.** Tests accompany each behavior change; no intentionally red commit boundary and no separate architecture phase.

## Success Criteria — executable and non-vacuous

### Named-test gate for each milestone

Run the following command after implementation. It rejects missing/skipped test names instead of accepting Go's zero-selected-tests success. Each expected `PASS <test>` corresponds to the mutation listed in T1–T13. It is deliberately expected to fail at the current baseline because the new tests do not exist yet.

```sh
python3 - <<'PY'
import json, re, subprocess
checks = [
 ("M1", "./internal/pipeline", ["TestCacheArtifacts_Authorization", "TestCacheArtifacts_PartialWrite", "TestCacheArtifacts_ReadSnapshot", "TestCacheArtifacts_Migration", "TestCachePipeline_WriteFailure", "TestCacheArtifacts_ByteLimits"]),
 ("M2", "./internal/loader", ["TestCacheSource_ExactSnapshot"]),
 ("M2", "./internal/pipeline", ["TestCacheSource_ExactSnapshot", "TestCachePipeline_EmbeddedKeys", "TestCachePipeline_SourceEditBehavior"]),
 ("M3", "./internal/pipeline", ["TestCacheStore_ClearArtifacts"]),
 ("M3", "./cmd/ailang", ["TestCompileCacheClear_Artifacts"]),
 ("M4", "./internal/apiserver", ["TestRegisterModule_RouteIfaceMismatch"]),
 ("M4", "./cmd/ailang", ["TestServeAPI_DivergentCacheTools"]),
]
for milestone, package, names in checks:
    pattern = "^(" + "|".join(re.escape(n) for n in names) + ")$"
    p = subprocess.run(["go", "test", "-json", package, "-run", pattern,
                        "-count=1", "-timeout=150s"],
                       capture_output=True, text=True, timeout=180)
    events = [json.loads(line) for line in p.stdout.splitlines() if line.startswith("{")]
    passed = {e.get("Test") for e in events if e.get("Action") == "pass"}
    assert p.returncode == 0, p.stdout + p.stderr
    assert set(names) <= passed, (milestone, "missing or skipped", set(names) - passed)
    for name in names:
        print(milestone, "PASS", name)
PY
```

At each boundary, run just that milestone's rows plus earlier rows. Acceptance requires all 14 named passes (T6 occurs in two packages). The existing CLI surface test uses `buildAilang(t)` and a bounded MCP probe [V48]; extend that machinery for isolated cache roots, routes-only mode, and tool calls. The MCP test must use a fresh binary built from the tested tree, never a PATH-installed binary; give its build a separate bounded setup deadline so process startup is not mistaken for a cache failure.

### AC-E2E — direct reproduction against an actual CLI binary

This is a standalone reviewer command and the source for the full-restoration portion of the durable T12 fixture. The fresh-stamp/single-stale-blob partial variant is **Go-test-only**: T12 must additionally pin the hash-failure diagnostic, f7 execution, and subsequent verified-hit accounting using the test fixture, while this short script demonstrates the original full-restoration failure. Passing this script alone does not satisfy T12. It uses real MCP stdio sessions, a dedicated temporary cache, bounded RPC waits and child cleanup. The six/seven fixtures contain no `main`, so the test selects the sole module by manifest membership, not a guessed absolute-path spelling. Canonicalize the temporary root before invoking either compiler/server; that avoids symlink spelling differences between cache keyspaces. The three serve-api flag names in the script were independently verified by the controller [V54].

Build the reviewed binary (180-second bound):

```sh
python3 - <<'PY'
import subprocess
subprocess.run(["go", "build", "-o", "/tmp/ailang-cache-reviewed", "./cmd/ailang"],
               check=True, timeout=180)
PY
```

Then run the block below with `/tmp/ailang-cache-reviewed 7` after the change. To demonstrate the baseline, use the baseline binary with argument `6`; this records the bug rather than treating it as post-change success. The default invocation shown is post-change acceptance.

<!-- cache-repro-start -->
```sh
python3 - /tmp/ailang-cache-reviewed 7 <<'PY'
import json, os, pathlib, selectors, shutil, subprocess, sys, tempfile, time
binary, expected = sys.argv[1], int(sys.argv[2])
with tempfile.TemporaryDirectory(prefix="ailang-cache-repro-") as td:
    root = pathlib.Path(td).resolve()
    source = root / "entry.ail"
    env = dict(os.environ, AILANG_CACHE_DIR=str(root / "cache"))
    def write_source(n):
        source.write_text("module entry\n" + "".join(
            '@route("POST", "/f%d")\nexport pure func f%d(x: int) -> int = x + %d\n'
            % (i, i, i) for i in range(1, n + 1)))
    def manifest():
        return json.loads((root / "cache/compile/manifest.json").read_text())
    def list_tools():
        p = subprocess.Popen([binary, "serve-api", "--mcp", "--routes-only",
                              "--no-feedback-tool", str(source)], cwd=root, env=env,
                             stdin=subprocess.PIPE, stdout=subprocess.PIPE,
                             stderr=subprocess.DEVNULL, text=True, bufsize=1)
        sel = selectors.DefaultSelector()
        sel.register(p.stdout, selectors.EVENT_READ)
        def rpc(obj):
            p.stdin.write(json.dumps(obj) + "\n")
            p.stdin.flush()
            end = time.monotonic() + 15
            while time.monotonic() < end:
                if sel.select(max(0, end - time.monotonic())):
                    line = p.stdout.readline()
                    assert line, "MCP exited before replying"
                    msg = json.loads(line)
                    if msg.get("id") == obj.get("id"):
                        return msg
            raise TimeoutError("MCP response exceeded 15 seconds")
        try:
            rpc({"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": {
                "protocolVersion": "2024-11-05", "capabilities": {},
                "clientInfo": {"name": "cache-repro", "version": "1"}}})
            p.stdin.write(json.dumps({"jsonrpc": "2.0",
                                     "method": "notifications/initialized"}) + "\n")
            p.stdin.flush()
            reply = rpc({"jsonrpc": "2.0", "id": 2,
                         "method": "tools/list", "params": {}})
            return sorted(t["name"] for t in reply["result"]["tools"])
        finally:
            p.terminate()
            try:
                p.wait(timeout=3)
            except subprocess.TimeoutExpired:
                p.kill()
                p.wait(timeout=3)
            sel.close()
    names = lambda n: ["f%d" % i for i in range(1, n + 1)]
    write_source(6)
    assert list_tools() == names(6)
    old_manifest = manifest()
    assert len(old_manifest["entries"]) == 1
    module_id = next(iter(old_manifest["entries"]))
    module_dir = root / "cache/compile/modules" / module_id.replace("/", "__").replace("\\", "__")
    old_files = {p.name: p.read_bytes() for p in module_dir.iterdir() if p.is_file()}
    write_source(7)
    assert list_tools() == names(7)
    fresh_key = manifest()["entries"][module_id]["cache_key"]
    assert old_manifest["entries"][module_id]["cache_key"] != fresh_key
    # Restore ALL old files, including the old stamp after v4 is implemented.
    shutil.rmtree(module_dir)
    module_dir.mkdir()
    for name, data in old_files.items():
        (module_dir / name).write_bytes(data)
    assert manifest()["entries"][module_id]["cache_key"] == fresh_key
    observed = list_tools()
    print("divergent tools:", observed)
    assert observed == names(expected), (observed, expected)
    assert manifest()["entries"][module_id]["cache_key"] == fresh_key
    # Next restart must remain correct (T12 additionally asserts a verified hit).
    assert list_tools() == names(expected)
    shutil.rmtree(module_dir)
    assert list_tools() == names(7)
    print("manifest stayed fresh; deleting one artifact directory yields seven tools")
PY
```
<!-- cache-repro-end -->

**Expected baseline output:** `divergent tools: ['f1', 'f2', 'f3', 'f4', 'f5', 'f6']`, then the manifest/deletion confirmation. **Expected post-change output:** the same list including `'f7'`, then the confirmation. A startup error, timeout, wrong name, or six-tool result after the change is a failure. This kills trusting a manifest without verifying the artifact key. T12 must additionally assert f7 execution and the fresh-stamp/single-stale-blob variant so an implementation that only adds the registration error or stamp-key comparison cannot satisfy acceptance.

### Compatibility gates

Run, with a 10-minute outer deadline per command, `go test ./internal/pipeline ./internal/loader ./internal/apiserver ./cmd/ailang -count=1 -timeout=5m` and `make check-boundaries`. Expected: package `ok` results and boundary check exit 0. These are supplementary regression checks; the non-vacuous T1–T13 gate supplies the positive proof of new behavior. The boundary target invokes `scripts/check_boundaries.sh` [V44]. No paid model eval or unbounded motoko session is necessary for this compiler-cache change.

Durable paths: existing tracked `internal/pipeline/cache_store_test.go`, `internal/pipeline/cache_invalidation_test.go`, and `internal/apiserver/module_entry_test.go` were confirmed with `git ls-files` [V45]. Existing `cmd/ailang/serve_api_mcp_surface_test.go` is also tracked [V48]; reviewer command `git ls-files --error-unmatch cmd/ailang/serve_api_mcp_surface_test.go` must print that path and exit 0. This catches banking an ignored/generated result instead of the regression test. No git mutation belongs to the designer task.

## Verification Log

Commands below were run in the requested worktree on 2026-09-05, except explicitly controller-supplied evidence. Output excerpts preserve the observed predicates; grouped citations refer to the corresponding individual claim rows. Future behavior, proposed tests, estimates, and design reasoning are not claims about existing implementation. Absence claims include a same-path known-positive control. No original reporter filesystem state is claimed as locally measured.

| ID | Current-code / evidence claim | Command executed | Observed output |
|---|---|---|---|
| V01 | Baseline commit; initially clean worktree. | `git rev-parse HEAD`; `git status --short` (control: `git rev-parse HEAD`) | `087fbea631a0b80556baa034b499fbdae33e76d2`; status empty, rev-parse succeeds. |
| V02 | Reporter version, dates, 28 lagging entries, repeated/copy/delete elimination sequence. | No local command possible against reporter checkout. | **Inherited from controller, not re-derived.** Historical evidence only. |
| V03 | Content key and manifest comparison are wired. | `sed -n '250,280p' internal/pipeline/pipeline_module.go`; `sed -n '84,91p' internal/pipeline/cache_store.go` | `ModuleCacheKey(version.Commit, sourceContent, depDigests)`; `Lookup(..., moduleCacheKey)`; `entry.CacheKey != cacheKey`. |
| V04 | Artifact API uses module-derived path with no cache-key authorization. | `rg -n 'CacheKey' internal/pipeline/cache_store.go`; control `rg -c 'modDir' internal/pipeline/cache_store.go`; `sed -n '184,190p' internal/pipeline/cache_store.go` | Exactly `40: CacheKey ...` and `87: ... entry.CacheKey != cacheKey`; positive control `11`; `LoadArtifacts(moduleID string)` joins `sanitizeModuleID(moduleID)`. No key occurrence in artifact methods. |
| V05 | Four independently decoded files; two gob and two JSON. | `cat internal/pipeline/cache_store.go` | Store/Load sections enumerate `core.gob`, `coretypeinfo.gob`, `iface.json`, `constructors.json`; gob decoders for first two, `unmarshalIfaceFull`/`unmarshalConstructors` for latter two. |
| V06 | In-memory entry update precedes artifact write; artifact and later manifest errors discarded. | `sed -n '367,385p' internal/pipeline/pipeline_module.go`; `sed -n '95,106p' internal/pipeline/cache_store.go`; `sed -n '419,430p' internal/pipeline/pipeline_module.go` | `Store` assigns map; pipeline `Store(...)` then `_ = cacheStore.StoreArtifacts(...)`; later `_ = cacheStore.Save()`; Save uses `os.WriteFile(manifest.json, ...)`. |
| V07 | Successful artifact decoding skips compilation; load error falls through. | `sed -n '276,305p' internal/pipeline/pipeline_module.go` | Assigns cached Core/CoreTI/Iface/Constructors and `continue`; comment and debug message state recompilation on load failure. |
| V08 | Key site swallows source read errors; recurrence recorded in adjacent comment. | `sed -n '256,276p' internal/pipeline/pipeline_module.go` | `sourceContent := ""`; `if srcBytes, err := os.ReadFile(srcPath); err == nil`; comment says earlier silent failure ignored source edits. Historical occurrence beyond this comment is not independently dated here. |
| V09 | Loader has embedded bytes and synthetic path; disk read errors stop before parse. | `sed -n '150,154p' internal/loader/loader.go`; `sed -n '194,212p' internal/loader/loader.go`; `sed -n '273,296p' internal/loader/loader.go` | Local `var content []byte`; `std.FS.ReadFile(embFile)`; `content = embContent`; `<embedded>/std/`; disk `ReadFile` failure returns `errors.WrapReport(report)`; lexer consumes `string(content)`; `file.Path = fullPath`. |
| V10 | Source content is not a retained LoadedModule field. | `rg -n 'SourceContent\|SourceDigest' internal/loader/loader.go`; control `rg -n 'type LoadedModule\|Imports' internal/loader/loader.go`; `sed -n '47,59p' internal/loader/loader.go`; `sed -n '330,343p' internal/loader/loader.go` | Source-field search empty; positive control `47:type LoadedModule` and `50:Imports`; complete struct/construction show Path/File/Imports/Exports/Types/Constructors/Core/Iface/CoreTI, no source payload. |
| V11 | Clear has no artifact deletion. | `sed -n '109,116p' internal/pipeline/cache_store.go` (full method; positive control is its `cs.Save()` call) | Resets Version/Entries then `return cs.Save()`; complete body has no filesystem removal. |
| V12 | CLI name is compile-clear, dispatch invokes Clear, error exits before success; brain commands share namespace. | `cat cmd/ailang/cache_compile.go`; `rg -n 'case "clear"\|case "compile-clear"' cmd/ailang/cache.go`; `cat cmd/ailang/cache.go` | Search returns only `53:case "compile-clear"` (positive control); helper calls `NewCacheStore(".")`, `cs.Clear()`, `os.Exit(1)` on error, then `Cleared %d cached compilation entries`; dispatcher includes search/put/promote. |
| V13 | Registration takes iface exports and fresh AST annotations. | `sed -n '100,117p' internal/apiserver/module_entry.go`; `sed -n '474,498p' internal/apiserver/server.go`; `sed -n '83,124p' internal/apiserver/routes.go` | `extractModuleInfo(loaded.Iface)` ranges iface.Exports; `extractRouteAnnotations(info, loaded.File)` scans file.Funcs and matching export names. |
| V14 | Missing route name receives no error in extractor. | `sed -n '83,126p' internal/apiserver/routes.go` (full method; positive control: `log.Printf` inside matching-name branch) | Method returns no value; logging and assignment occur only inside `if ...Name == fn.Name`; no unmatched-name branch. |
| V15 | Existing drop validation concerns outside-basePath modules and has an override. | `sed -n '54,79p' internal/apiserver/module_entry.go`; `sed -n '248,305p' internal/apiserver/server.go` | Outside filter calls `recordDrop`; ValidateRegistration scans dropped module annotations, checks `AllowDropsEnvVar`, reports `dropped by under-basePath filter`. |
| V16 | Serve-api has no compile-cache bypass wiring. | `rg -n 'NoCache\|AILANG_NO_CACHE\|no-cache' internal/apiserver cmd/ailang/serve_api.go`; controls `rg -n 'pipeline.Config\|routesOnlyFlag' internal/apiserver/load_project.go cmd/ailang/serve_api.go`; `sed -n '68,75p' internal/apiserver/load_project.go` | First search empty; controls find config at line 70 and route flag at lines 33/141; config has ModeCheck and RelaxModules only. |
| V17 | Version v3 is part of hash and manifest loading rejects version mismatch. | `cat internal/pipeline/cache_key.go`; `cat internal/pipeline/cache_store.go` (load method) | `const cacheKeyVersion = "v3"`; hash prefix includes version/compiler; sorted dependency loop; `manifest.Version != cacheKeyVersion` returns error and constructor initializes fresh manifest. |
| V18 | Sanitization collides for slash and double underscore. | `cat internal/pipeline/cache_store.go` (sanitizeModuleID); `python3 -c 'print("a/b".replace("/", "__"), "a__b".replace("/", "__"))'` | Switch replaces slash/backslash with two underscores, otherwise copies bytes; evaluated output `a__b a__b`. |
| V19 | Cache constructor failure silently disables pipeline cache. | `sed -n '227,238p' internal/pipeline/pipeline_module.go` | Guard `!cfg.NoCache`; only `if cs, err := NewCacheStore(projectDir); err == nil { cacheStore = cs }`, no error branch (complete block; positive control NewCacheStore call). |
| V20 | Cache root override remains scoped to compile/. | `sed -n '60,72p' internal/pipeline/cache_store.go` | Nonempty AILANG_CACHE_DIR joins `compile`; otherwise project joins `.ailang/cache/compile`; MkdirAll failure returns contextual error. |
| V21 | Empty-source key algebra reproduces controller's explicit commit hash and local dev hash. | `python3` SHA-256 computation: `payload = f'ailang-cache:v3:{commit}\nsource:{hashlib.sha256(b"" ).hexdigest()}\n'`; print SHA-256 for `dev` and full V01 commit. | `dev → 8547154886eff974a400ae4f31701a71abd1ac46b0d5bbc8283e00e5a335f10c`; V01 commit → `b5149f5d2d7eac93707cf159b94ccdcc9f97b8d2960fe843a7eeb20c3e6f8136`. Re-derived formula, not a claim that the local binary was linked with V01 as version.Commit. |
| V22 | Actual MCP reporter-signature reproduction. | `go build -o /tmp/ailang-cache-audit-087fbea63 ./cmd/ailang` (180s outer bound); AC-E2E block with this binary and expected count `6`. | Build exit 0; v1 six, fresh seven; new manifest key differs; restored artifacts yield six; manifest remains fresh; delete one directory yields seven. The initial check/server experiment used different physical spellings and did not reproduce; canonical-root, server-only warmup is the successful controlled command. |
| V23 | Live option/result manifest keys coincide despite different interfaces. | Temporary-project probe in Appendix A, using V22 binary. | Both keys `8547154886eff974a400ae4f31701a71abd1ac46b0d5bbc8283e00e5a335f10c`; option digest `cbda6d5a1b82429a0ed24eae20170c33599f57450a4e8ef70acd56339632a3e5`; result digest `c725731a4ccf2b21fdebb1cfe1d355af22c98affbaf99516969eb500ad247a80`. Key equals independently computed dev empty-source key V21. |
| V24 | Ordinary disk dependency invalidation still works. | Appendix A probe: two `run --entry main --caps IO main.ail` invocations changing dep 2 → 40. | Both exit 0; stdout ends in `3\n`, then `41\n`. |
| V25 | CLI clear leaves artifacts on baseline. | Appendix A probe: `cache compile-clear` in that isolated project/cache. | Exit 0, `Cleared 4 cached compilation entries`; `modules remains: True`. |
| V26 | Loader prelude/export building precedes LoadedModule return. | `sed -n '309,345p' internal/loader/loader.go` | `injectEntryPreludeImports`, `buildExports`, `buildTypes`, LoadedModule literal and cache assignment. |
| V27 | Stdlib filesystem resolution occurs before embedded fallback. | `sed -n '194,214p' internal/loader/loader.go`; `sed -n '160,205p' internal/loader/stdlib_resolver.go`; `cat std/embed.go` | ResolveStdlib iterates search paths/stat; Load tries std.FS only when ResolveStdlib returns error; std/embed.go declares `//go:embed *.ail` and `var FS embed.FS`. |
| V28 | MCP registration enumerates registered exports, with exposure/nomcp filtering and optional feedback tool. | `sed -n '30,110p' internal/apiserver/mcp.go`; `cat internal/apiserver/load_project.go` | `registerTools` ranges modInfo.Exports, calls loadedExportMember, skips IsNoMCP; noFeedbackTool controls feedback registration; LoadProject preloads result.Modules before registerModule. |
| V29 | Eval uses run subprocess with inherited environment; run reads cache-disable env. | `sed -n '275,335p' internal/eval_harness/runner.go`; `rg -n 'AILANG_NO_CACHE' cmd/ailang/main_run_exec.go` | `run --entry main --quiet`, relax/stdlib flags, workspace command, `env := os.Environ()`; line 240 sets `NoCache: os.Getenv("AILANG_NO_CACHE") == "1"`. |
| V30 | Motoko allocates per-task cache before child execution. | `sed -n '337,355p' internal/executor/motoko/motoko.go`; `rg -n 'AILANG_CACHE_DIR' internal/executor/motoko/motoko.go` | Comment describes parallel gob corruption; calls `setupTaskCacheDir(sessionID)`, returns error on failure, defers cleanup; line 387 sets `AILANG_CACHE_DIR=` plus taskCacheDir. |
| V31 | Requested test files read in full. | `cat internal/pipeline/cache_invalidation_test.go`; `sed -n '1,310p' internal/pipeline/cache_store_test.go`; `sed -n '311,650p' internal/pipeline/cache_store_test.go`; final `sed -n '440,520p' internal/pipeline/cache_store_test.go` | Full source through closing brace of readCacheKey and EmptyEnvFallsBackToProjectDir; nine tests across the two files. |
| V32 | Source-edit regression checks only manifest keys. | `cat internal/pipeline/cache_invalidation_test.go` | Run results discarded as `_`; ModeCheck 42/99 fixtures; reads keyV1/keyV2 and compares; helper returns entry.CacheKey. Positive controls for absence of execution assertions: two Run calls and key inequality assertion in complete file. |
| V33 | Manifest round-trip assertions. | `sed -n '14,65p' internal/pipeline/cache_store_test.go` | Lookup correct/wrong/missing; digest/JSON comparisons; no artifact methods in complete test (positive control Store/Save/Lookup). |
| V34 | Clear test only counts entries. | `sed -n '66,93p' internal/pipeline/cache_store_test.go` | Store mod1/mod2, Save, Stats 2, Clear, Stats 0; no artifact creation/reload in full body (positive control Clear). |
| V35 | CorruptedManifest is invalid-JSON recovery only. | `sed -n '94,113p' internal/pipeline/cache_store_test.go` | Writes `not json`; expects constructor success and Stats zero. |
| V36 | Stats asserts counts and summed recorded time. | `sed -n '114,133p' internal/pipeline/cache_store_test.go` | Two entries, CompileTimeMs 10/20, expected 30. |
| V37 | ArtifactRoundTrip covers selected successful serialization fields only. | `sed -n '135,302p' internal/pipeline/cache_store_test.go` | One StoreArtifacts/LoadArtifacts pair; assertions listed in coverage table. No second write/failure in full body; positive control StoreArtifacts at line 228. |
| V38 | Diverse round trip covers selected node forms only. | `sed -n '303,405p' internal/pipeline/cache_store_test.go` | Six decls, Match exhaustive/arms, VarGlobal module, DictRef class; one successful Store/Load pair. |
| V39 | Environment tests inspect manifest placement only. | `sed -n '406,464p' internal/pipeline/cache_store_test.go` | Tests set override/nonempty then empty; Stat checks for manifest paths; no artifact/clear invocation in full bodies (positive control Save). |
| V40 | Key unit tests test key algebra, not artifacts. | `cat internal/pipeline/cache_key_test.go` | Seven named tests listed above; only ModuleCacheKey calls and string assertions. |
| V41 | Embedded fallback loader test lacks source/key assertions. | `sed -n '115,156p' internal/loader/loader_test.go` | Subtests check nonnil exports/types, missing module error, repeated pointer equality; positive controls Load calls, no key calculation in complete test. |
| V42 | Audited cache/registration tests do not construct divergent artifacts. | `rg -n 'LoadArtifacts\|StoreArtifacts\|cache/compile\|compile-clear' internal/apiserver --glob '*test.go'`; controls same pattern in `internal/pipeline --glob '*test.go'` and `rg -n '^func TestExtractRouteAnnotations' internal/apiserver/routes_annotations_test.go`; inspected module_entry_test.go test names. | Apiserver search empty; pipeline finds only store round trips and manifest path assertions; route controls at 11/81/138. Module-entry tests cover physical identity, base-path drops, idempotence, drop validation. CLI search `rg -n 'compile-clear\|StoreArtifacts\|LoadArtifacts' cmd/ailang --glob '*test.go'` was empty; positive control `case "compile-clear"` in cmd/ailang/cache.go. Coverage conclusions are structural, not an exhaustive mutant run. |
| V43 | Existing cache-focused tests pass with defect present. | `go test ./internal/pipeline -run '^(TestCacheKey_InvalidatesOnSourceEdit\|TestCacheStore_.*\|TestNewCacheStore_.*\|TestModuleCacheKey_.*)$' -count=1 -timeout=90s` (120s outer bound) | `ok github.com/sunholo-data/ailang/internal/pipeline 0.433s`, exit 0. Includes source-edit, store, environment-root, and key-unit tests. |
| V44 | Boundary check target. | `sed -n '180,188p' make/code-health.mk` | `check-boundaries:` runs `bash scripts/check_boundaries.sh`. |
| V45 | Durable existing regression paths are tracked. | `git ls-files internal/pipeline/cache_invalidation_test.go internal/pipeline/cache_store_test.go internal/apiserver/module_entry_test.go` | All three exact paths printed. |
| V46 | Design convention references were read before writing. | `cat design_docs/implemented/v0_35_0/m-pipeline-reconciliation.md`; `cat design_docs/implemented/v0_10_0/m-route-collision-guard.md`; `cat design_docs/planned/v0_35_0/m-fs-rename.md` | Headings include metadata, Axiom Compliance, Problem Statement, Goals, High-Impact Decisions, Solution Design, Success Criteria, Testing Strategy, Conflict Surface / Verification Log. Historical technical claims in those docs are not adopted here. |
| V47 | Runtime result assembly creates separate LoadedModule values from compiled units. | `sed -n '593,647p' internal/pipeline/pipeline_module_compile.go`; `rg -n 'assembleModuleResult' internal/pipeline` | Pipeline line 516 calls assembler; helper assigns Path/File/Core/Iface/CoreTI/Imports and empty compatibility maps to new LoadedModule values. |
| V48 | Existing tracked CLI MCP surface test checks feedback suppression without divergent artifacts and provides a bounded probe. | `sed -n '1,150p' cmd/ailang/serve_api_mcp_surface_test.go`; `git ls-files cmd/ailang/serve_api_mcp_surface_test.go` | Complete file: TestServeAPI_MCPToolSurface calls buildAilang(t), writes one status/helper source, checks status and submit_feedback; helper uses a 30-second context. No source edit/cache mutation in complete test; positive control status/tools-list assertions. Git prints the exact path. |
| V49 | Populated cache artifact sizes measured during the revision, rather than inferred from blob count. | `ls -l /Users/voightkampff/dev/sunholo-data/ailang/.ailang/cache/compile/modules/*/core.gob`; same command for `/Users/voightkampff/dev/sunholo-data/.probe-iter328-cache/.ailang/cache/compile/modules/*/core.gob` | First cache: scratch_ok 1,205, scratch_ver 1,325, std__io 5,600 bytes. Second: lib 962, main 1,699, std__option 7,541, std__result 8,218 bytes. Existing files were only inspected, not generated or changed in this revision. |
| V50 | Broader local sample and all four payload sizes support the chosen headroom. | Read-only `python3` / `pathlib` survey: under `/Users/voightkampff/dev/sunholo-data/{ailang,iter152probe,.probe-iter328-cache,.wt-iter163-acfixture}/.ailang/cache/compile/modules`, iterate child directories, retain those with all four fixed files, use `stat().st_size`, and print count, per-name maxima, maximum sum and its path. | 27 complete modules; core min/max 962/8,218; maxima Core 8,218, CoreTI 20,387, iface 5,845, constructors 361 bytes. Maximum sum 34,811, all four maxima from `.probe-iter328-cache/.../std__result`. This is a local sample, not a claim that no larger module exists. |
| V51 | Baseline artifact loading reads unbounded files before decode; the proposed helper replaces these read sites. | `sed -n '1,290p' internal/pipeline/cache_store.go` (complete StoreArtifacts/LoadArtifacts bodies; positive control `func (cs *CacheStore) LoadArtifacts` at line 185) | Four `os.ReadFile` calls, each followed by gob or JSON decoding; no size check or limited reader in the complete load method. StoreArtifacts supplies the matching gob encoders and JSON marshal helpers [V05]. |
| V52 | `LimitReader` EOF needs a sentinel; gob's decoder itself is not a hostile-input resource bound. | `go env GOROOT`; `sed -n '454,489p' /Users/voightkampff/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.26.6.darwin-arm64/src/io/io.go`; `sed -n '274,288p'` of that toolchain's `src/encoding/gob/doc.go` | Go 1.26.6: LimitedReader returns EOF at N <= 0 or underlying EOF and truncates each read request to N. Gob security documentation says decoder size checks are basic, limits are not configurable, and untrusted inputs can consume significant resources. These first-party sources support sentinel enforcement and the explicitly conditional compiler-origin decode argument, not a new gob hardening claim. |
| V53 | Proposed overflow reason `ARTIFACT_TOO_LARGE` and its `CACHE_INVALID` prefix are not allocated in current code. | `rg -n 'CACHE_INVALID\|ARTIFACT_TOO_LARGE\|BLOB_BYTES_EXCEEDED\|MODULE_BYTES_EXCEEDED\|STAMP_BYTES_EXCEEDED' internal cmd`; same-code positive control `rg -n 'CACHE_WRITE_FAILED\|func .*LoadArtifacts' internal/pipeline` | First search returned no matches; control returned `internal/pipeline/cache_store.go:185:func (cs *CacheStore) LoadArtifacts(...)`. This revision uses only `ARTIFACT_TOO_LARGE` with a scope field, not the other searched candidate names. |
| V54 | AC-E2E serve-api flags `--mcp`, `--routes-only`, and `--no-feedback-tool` exist as flag registrations. | **Controller-supplied first-party verification, not rerun by designer:** controller searched each flag name in `cmd/ailang/serve_api.go` and ran a known-absent control. Exact command spelling/control token was not supplied. | Controller reported each of the three names appears exactly once as a flag name; known-absent control returned 0. Records flag availability only, not new behavioral claims. |

### Appendix A — temporary-project measurements (V23–V25)

The probe below re-derives conditional source identity loss and clear behavior at baseline; expected values describe the baseline, not acceptance after implementation. Every subprocess has a 15-second deadline.

```sh
python3 - <<'PY'
import json, os, pathlib, subprocess, tempfile
binary = "/tmp/ailang-cache-audit-087fbea63"
with tempfile.TemporaryDirectory(prefix="ailang-cache-source-") as td:
    root = pathlib.Path(td).resolve()
    env = dict(os.environ, AILANG_CACHE_DIR=str(root / "cache"))
    (root / "main.ail").write_text(
        "module main\nimport dep (value)\n"
        "export func main() -> () ! {IO} = println(show(value() + 1))\n")
    for value in (2, 40):
        (root / "dep.ail").write_text(
            "module dep\nexport pure func value() -> int = %d\n" % value)
        p = subprocess.run([binary, "run", "--entry", "main", "--caps", "IO", "main.ail"],
                           cwd=root, env=env, capture_output=True, text=True, timeout=15)
        assert p.returncode == 0, p.stderr
        print("dep", value, "stdout", repr(p.stdout))
    m = json.loads((root / "cache/compile/manifest.json").read_text())
    for name in ("std/option", "std/result"):
        print(name, m["entries"][name]["cache_key"], m["entries"][name]["iface_digest"])
    p = subprocess.run([binary, "cache", "compile-clear"], cwd=root, env=env,
                       capture_output=True, text=True, timeout=15)
    print("clear:", p.returncode, p.stdout.strip(), "modules remains:",
          (root / "cache/compile/modules").exists())
PY
```

## Out of scope / follow-ups

- Eliminate sanitized-directory aliasing with hashed module-ID directories. This sprint prevents wrong-module loads but deliberately permits collision-induced misses.
- Add `serve-api --no-cache` / `AILANG_NO_CACHE` support, or a compile-cache inspection command. Correctness must not depend on an operator discovering a bypass.
- Add mtime age reporting as a troubleshooting hint; never use age as the validity oracle.
- Transactional/durable manifest replacement, cross-process locking, cache GC, distributed caches, or authenticated artifacts. Artifact verification byte ceilings are in scope. Concurrent cache clear is not a writer-quiescence mechanism.
- Audit additional key inputs such as compiler options and validation policies. This proposal binds artifacts to the existing input-key contract; it does not claim that contract includes every semantically relevant configuration option.
- Changes to unrelated source rereads for documentation extraction, broader private-route validation, or other surface/annotation inconsistencies.
- Investigate the reporter's original failed write (ENOSPC, permission, serializer error, interruption). The divergent-state reproduction proves the execution hazard, not which event created their historical directory.

## Risks & Mitigations

| Risk | Mitigation |
|---|---|
| Verification overhead on warm modules | Hash already-read bytes; decode without extra disk reads; enforce the 16 MiB blob / 64 KiB stamp / 32 MiB module ceilings and compiler-origin decode assumption. Do not weaken integrity to recover a benchmark number. |
| Source snapshot memory retention | Reuse the string given to the lexer, retain only through loader lifetime; no artifact payload duplication. |
| Noisy write failures in read-only environments | Warn once per module/stage or once per invocation for shared failures; compilation remains usable. |
| Stamp added but not coupled to actual bytes | T2/T3 exercise each blob and verify/decode interleaving. |
| Green key tests conceal stale executable behavior again | T8/T12 require observable values, exact tool names, a call result, and a subsequent verified hit. |
| Scope grows beyond 4 days | Keep storage layout and caller configuration fixed; route adjacent work to the explicit follow-ups. |

## Related Documents

- [Pipeline reconciliation](../../implemented/v0_35_0/m-pipeline-reconciliation.md) — evidence-log and decision structure [V46].
- [Route collision guard](../../implemented/v0_10_0/m-route-collision-guard.md) — neighboring API registration design, not evidence for current cache semantics [V46].
- [FS rename](../v0_35_0/m-fs-rename.md) — conflict-surface and bounded implementation structure [V46].

**Document created**: 2026-09-05. Implementation and final commits remain with the mission controller.
