# M-CACHE-MODULE-ID-ENCODING — replace `sanitizeModuleID`'s broken, non-injective mapping

**Status**: PARKED — needs-human-review; blocked pending D-57
**Created**: 2026-09-06
**Mission**: V1, iteration 334; iteration-336 design phase parked after round-2 quorum
**Supersedes queue rows**: `m-cache-sanitize-module-id-windows-colon` (iter-333),
`m-cache-sanitize-module-id-collision` (iter-328)

## Problem

`sanitizeModuleID` (`internal/pipeline/cache_store.go:161`) is the function that turns a
compiled module's ID into the single directory component holding that module's compile
artifacts under `<cache>/compile/modules/`. It replaces only `/` and `\` with `__` and passes
every other byte through unchanged. Its sole production consumer is
`moduleArtifactDir` (`internal/pipeline/cache_artifacts.go:402-404`).

Two distinct, separately-filed defects, both on this one function, with different severities:

**Defect 1 — Windows: drive-letter IDs produce illegal artifact directory components.**
For a drive-letter module ID such as `C:/Users/runneradmin/AppData/...`, the old encoder
produces **one** path component `C:__Users__runneradmin__AppData__...` containing a **colon**.
Windows forbids `:` in a path component (`< > : " / \ | ? *`, plus ASCII control chars,
trailing dot/space, and the reserved `CON PRN AUX NUL COM1-9 LPT1-9` names are all illegal —
and a reserved name stays reserved when followed by an extension: `con.txt` has basename `con`
and is just as illegal as bare `con`). Publication for such an ID therefore targets an
illegal component. This is a deduction about the encoding, not a captured Windows execution.

**Reconstructed diagnostic, not a Windows CI transcript:** the following illustration combines
the verified separator-only encoding and publication diagnostic format with Windows' colon
restriction. No Windows CI log excerpt of this error was captured. The fetched baseline log
had no `CACHE_WRITE_FAILED` match (see iteration-336 Verification Log); it does not establish
how many Windows modules encounter this path or a platform-wide outage.

```text
CACHE_WRITE_FAILED module=C:/Users/... stage=publication
path=...\compile\modules\C:__Users__runneradmin__... : ARTIFACT_INVALID
```

The proposed M3 Windows publication test must establish runtime behavior for the drive-letter
fixture. The earlier claim that this text was observed on the CI runner, and the associated
claim of a measured total-platform outage, are withdrawn. The separator-only encoder itself
predates this change.

Two tests in `cmd/ailang/serve_api_mcp_surface_test.go` (the callers of
`requireCompileArtifactCache` at lines 177 and 287) are gated behind a skip that fires on
Windows **and** an observed-empty manifest (skip branch at line 572). That skip is a temporary
accommodation; section Conflict Surface and M3 say how it is retired.

**Defect 2 — collision: the mapping is not injective (STORAGE / RECOMPILATION inefficiency).**
`sanitizeModuleID("a/b")` and `sanitizeModuleID("a__b")` both yield `a__b`. Two distinct module
IDs therefore share one artifact directory and evict each other, forcing a recompile. The repo
already *documents* this as expected: the subtest
`sanitized_collision_uses_exact_module_id` (`internal/pipeline/cache_artifacts_test.go:64-73`)
asserts the collision exists and then proves the correctness backstop holds — the artifact
stamp records the exact `module_id`, and `LoadArtifacts` refuses any stamp whose `module_id`
does not match the requested ID (`internal/pipeline/cache_artifacts.go:305`), so a collision
costs a recompile, never a wrong program. This is a **storage/recompilation defect, not a
correctness one**. State plainly: it is a performance/efficiency gap.

Different severities, one cause: a non-canonical, non-injective, non-legal directory-name
encoding.

### Q1c pre-check — case collisions (the trap neither filed row mentions)

macOS APFS and Windows NTFS are case-INSENSITIVE by default. The loader keys modules by
`module.Path` (`internal/loader/loader.go:513`); that assignment does not fold case. Separately,
`validateModuleName` (`internal/loader/stdlib_resolver.go:25`) allows `[a-zA-Z0-9_/-]` for
stdlib-style names (scope verified below). The encoder must therefore handle byte-distinct
case-variant IDs without relying on case-sensitive directory comparison. On a case-insensitive
filesystem a byte-injective-but-case-preserving mapping would still collide at the directory
level. The chosen scheme must preserve its collision resistance after filesystem case folding.
The stamp `module_id` backstop still protects correctness; this is a recompilation-churn source
that the proposed scheme reduces.

## Non-goals

- **Not adversarial cache hardening.** We are not defending against a hostile peer who can
  write arbitrary paths under the cache root. That is the separate row
  `m-cache-artifact-adversarial-decode`. `validateModuleName` rejects `..`, null bytes,
  and restricts names to `[a-zA-Z0-9_/-]` for stdlib resolution: its sole production caller
  is `StdlibResolver.ResolveStdlib`. This does not establish validation of all absolute-path
  module IDs; defence in depth for those remains the adversarial row's job.
- **No change to the artifact stamp's correctness role.** The exact-`module_id` comparison in
  `LoadArtifacts` stays the ultimate backstop for every collision (case- or otherwise). This
  design reduces collisions; it does not move the correctness boundary.
- **No automated old-scheme garbage collection.** Stale directories are swept by the existing
  `Clear()` (see Migration); an opportunistic age-based GC of orphan dirs is out of scope.
- **No drive-letter / path canonicalisation on Windows.** We fix the encoding; we do not change
  how module IDs are produced by the loader (that is a separate, cross-cutting change).

## Design options considered

Constraint bundle every option must satisfy — every output must be:
(a) a legal single path component on Windows/macOS/Linux (no `< > : " / \ | ? *`, ASCII
    control, trailing dot/space; and the basename **before the first dot** is not a reserved
    device name — `con.txt` is as illegal as `con`);
(b) collision-resistant, with foreign-module stamps rejected by the existing backstop;
(c) retain that collision resistance after a case-insensitive filesystem folds case;
(d) bounded to 57 ASCII bytes per component (the whole path still depends on the cache root);
(e) deterministic across runs and platforms.

### Option A — widen the replacement set to all forbidden characters (REJECTED)
Replace every forbidden Windows byte with `__`, keep everything else. This is the smallest
diff and is *wrong* for three reasons: (1) it does not remove the reserved device-name hazard
(a module whose sanitized name is `CON`, or on a case-insensitive FS `con`, or `con.txt` —
reserved basename before the dot — still names an illegal component); (2) it makes **collisions strictly worse** — the separator `_` and
`.` are already collision-prone and every newly-mapped character shares one replacement
symbol, so `a:b`, `a"b`, `a*b` … all collapse; (3) it does nothing about Windows' 255-char
component limit because output length scales with input length, and module IDs are absolute
paths. Rejected.

### Option B — pure content hash (fixed-length, legal, collision-resistant)
`encode(id) = hex(sha256(id))[:N]` lowercase hex characters. Pros: constant output length (independent of a
long `C:/Users/…` input), inherently legal (hex alphabet), collision resistance dependent
on `N`, and a lowercase-hex alphabet stable under filesystem case folding. Case variants
are hashed as different inputs; hash collisions remain possible. Cons: the cache directory becomes
*unreadable to a human* — you cannot tell `std/list` from `std/map` at a glance, which makes
debugging on-disk state materially harder. Pure hash alone would discard all debuggability for
no correctness gain over a hybrid with the same hash width and stamp backstop.

### Option C — hybrid: truncated human-readable slug + hash-suffix over the FULL module ID (CHOSEN)
Keep a shortened, sanitised, truncated slug for human debuggability and append a short
lowercase-hex digest derived from the **full** module ID for collision resistance. The slug
is a readability aid that may also separate some hash collisions, but is lossy; it cannot
guarantee uniqueness. The existing exact-module-ID stamp check remains the correctness
authority. Detailed in Chosen design.

**Why C over B at the same 64-bit hash width:** the hybrid keeps a readable module hint
while preserving the hash-based collision resistance and lowercase alphabet. It costs a slug
pass and up to 41 additional bytes; the total remains within the chosen 57-byte component cap.
This is a readability/space tradeoff, not a claim of mathematical injectivity or equivalence
to a full 256-bit hash.

### Contested alternative from iteration-336 quorum (not adopted; awaiting re-quorum)

Sol proposed a full 256-bit digest and a new `CACHE_DIRECTORY_COLLISION` diagnostic, followed
by recompilation or selection of a bounded alternate directory. Its objection to the former
injectivity claim is accepted: no bounded encoding over arbitrary-length IDs can guarantee
uniqueness, including a full digest. A full digest would improve collision resistance, but
64 lowercase hex characters already exceed this design's 57-byte component cap before adding
the prefix or slug. The new diagnostic/alternate-directory behavior would also add production
semantics beyond the bounded revision. Neither proposal is silently treated as approved or
unnecessary: they remain contested alternatives, not adopted in the current scope, for the
independent re-quorum to assess against the corrected contract.

The retained 64-bit scheme explicitly accepts theoretical directory contention/recompilation.
When an existing artifact stamp belongs to another module, the existing `stamp.ModuleID !=
moduleID` check rejects it; `cacheRuntime.load` invokes `warnInvalid` (the existing
`CACHE_INVALID …; recompiling` diagnostic) and the pipeline takes its ordinary compile path.
This detects foreign-module artifacts at load time; it is not a promise to prevent future
writes from contending for the same directory. No collision-specific error code or alternate
path is introduced here. M1's inherited test name `TestEncodeModuleDirName_InjectivityAndDeterminism`
means separation of its finite fixture set, not a proof of mathematical injectivity.

## Chosen design

Replace `sanitizeModuleID` with an `encodeModuleDirName(string) string` (new function) computed
as `"m-" + slug(id) + "-" + hex(sha256(id))[:16]` (16 hex characters, not 16 digest bytes):

```go
// encodeModuleDirName maps a module ID to a legal, bounded, collision-resistant
// single directory component whose lowercase alphabet is stable under filesystem case folding.
// The unconditional "m-" prefix is the Windows reserved-name guard (see legality).
// The slug is a truncated readability aid only; the 16-hex-char SHA-256 prefix over
// the FULL module ID is the collision-resistant discriminator.
func encodeModuleDirName(moduleID string) string {
    sum := sha256.Sum256([]byte(moduleID))
    suffix := hex.EncodeToString(sum[:])[:16]     // 64 bits, lowercase alpha -> case-insensitive-FS safe
    return "m-" + slug(moduleID) + "-" + suffix
}

// slug processes raw bytes: fold ASCII A-Z to a-z; keep [a-z0-9_-]; replace each
// other byte with exactly one '_'. Preserve runs. Trim outer '_' BEFORE the 38-byte
// cut; do not trim again after cutting. Never decode or lowercase Unicode runes.
func slug(id string) string { /* ASCII byte-map; trim outer '_'; cut at 38 */ }
```

**Normative slug algorithm (iteration 336 clarification):**

1. Iterate the original Go string by byte index. For each byte in ASCII `A`–`Z`, add
   32 to obtain `a`–`z`. Preserve ASCII `a`–`z`, `0`–`9`, `_`, and `-`.
   Replace every other byte, including `.`, with exactly one ASCII `_`.
2. Preserve all resulting underscore runs, including adjacent replacements and original
   underscores. There is no run collapse: `:/` becomes `__`; the two UTF-8 bytes of `é`
   also become `__`. Do not decode runes, apply Unicode case folding, or normalize the input.
3. Trim all leading and trailing `_` bytes from the complete mapped string.
4. Take the first `min(38, length)` bytes of that trimmed string. Do not trim again:
   truncation can expose a trailing `_`, which is allowed. An empty slug is allowed and
   produces `m--<16hex>`; no placeholder is substituted.
5. Compute the suffix independently over the full, original string bytes, before any of
   these slug transformations. Append the first 16 lowercase hex characters of SHA-256.

This resolves the inherited design/plan ambiguity in favor of the sprint plan's per-byte,
runs-allowed mapping. The hybrid shape, prefix, hash width, and 57-byte bound are unchanged.
The reference calculation in the iteration-336 Verification Log pins the order and fixtures.

- **Output shape**: `m-<slug>-<16hex>`, e.g. `m-std_list-d9997702a41d1e11`.
- **Max component length**: 2 (`m-`) + 38 (slug cap) + 1 (dash) + 16 (suffix) = **57 chars**,
  far under the 255-char Windows component limit. Independent of input length, so an absolute
  `C:/Users/runneradmin/…` path stays ≤ 57 chars.
- **(a) legality — the argument is the PREFIX, not the suffix.** Round 1 of this doc claimed
  "the output always ends in `-<hex>` so it is never a reserved device name". **That claim was
  false**: Windows reserves the basename *before the first dot*, so `con.txt-<hex>` (slug with
  `.` preserved) still has basename `con` and is illegal; a suffix after a dot protects nothing.
  The corrected argument: every output begins with the literal `m-`, so the basename before any
  dot is `m` + `-` + (the slug's first segment) — it always starts with `m-` and therefore can
  never equal `CON`, `PRN`, `AUX`, `NUL`, `COM1-9` or `LPT1-9`, whatever the module ID was.
  Second, and independently: the slug alphabet **drops `.`** (mapped to `_`, alphabet is
  `[a-z0-9_-]`), so no output contains a dot at all and the "basename before the first dot" is
  the whole component, which starts with `m-`. **Choice**: we drop `.` rather than keep it. The
  prefix alone would suffice, but dropping `.` (i) removes the hazard a second, independent
  way, (ii) makes the trailing-dot rule moot (nothing to trim), and (iii) costs nothing in
  readability — `validateModuleName` permits no `.` in stdlib-style names, and dots in absolute-path
  IDs are directory-name noise. The suffix alphabet is `[0-9a-f]`; no forbidden byte survives
  either part. The slug never ends in `.` or space because neither is in its alphabet.
- **(b) collision resistance, not injectivity**: distinct module IDs map to distinct 16-hex-char (64-bit)
  suffixes with high probability under the non-adversarial, uniformly distributed digest
  model. Birthday bound: collision probability ≈
  n² / 2⁶⁵; at n = 10⁶ directory entries that is ≈ 3e-8. If two IDs ever did collide the
  suffix would agree, and different final slugs can still distinguish that pair. Distinct input
  prefixes need not yield distinct slugs: case folding, replacement, trimming, and truncation
  are lossy. A collision, even if one occurred, degrades to the existing stamp-backstop
  recompile, never to a wrong program (Q3). For arbitrary-length inputs, a bounded output
  space cannot be injective. A theoretical hash collision together with equal final slugs
  can still cause directory contention and repeated recompilation. This proposal reduces
  that risk; it does not eliminate it or introduce collision-specific diagnostics.
- **(c) case-fold-stable alphabet**: `slug` is lowercased and the suffix is lowercase hex over the
  **original bytes**. `Foo` and `foo` produce different lowercase-hex suffixes
  (`1cbec737f863e492` vs `2c26b46b68ffc68f`), so the two directories remain distinct even on
  an APFS/NTFS case-insensitive volume, because the *alphabet* of the differing part has no
  cases to fold.
- **(d) bounded**: 57 chars max.
- **(e) deterministic**: a pure function of `moduleID`; identical on every run and platform.

Worked examples (recomputed in iteration 336 using the normative byte algorithm above;
suffix = first 16 hex of SHA-256 over the raw module ID; the `m-` prefix does not enter the hash):

| module ID | slug (38 cap, lower, mapped) | resulting directory component |
|---|---|---|
| `std/list` | `std_list` | `m-std_list-d9997702a41d1e11` |
| `a/b` | `a_b` | `m-a_b-c14cddc033f64b9d` |
| `a__b` | `a__b` | `m-a__b-63e5c1c455d01d5c` |
| `C:/Users/runneradmin/x` | `c__users_runneradmin_x` | `m-c__users_runneradmin_x-81fb5218f110e3cc` (no colon ⇒ legal on Windows) |
| `con` | `con` | `m-con-1143da2bc54c495c` (basename `m-con-…`, never `con`) |
| `con.txt` | `con_txt` | `m-con_txt-d3bde286fd271ed6` (legal: no dot survives, and the basename before any dot starts with `m-`, not `con`) |
| `CON.txt` | `con_txt` (same slug as `con.txt`) | `m-con_txt-09c8cc7edcae01ac` (legal for the same reason; distinct from `con.txt` via the suffix) |
| `nul.log` | `nul_log` | `m-nul_log-c0294fbf8537502a` (legal: basename before any dot is `m-nul_log-…`, never `nul`) |
| `COM1.any` | `com1_any` | `m-com1_any-bdd82f44de519430` (legal: basename before any dot is `m-com1_any-…`, never `COM1`) |
| `Foo` / `foo` | `foo` / `foo` (same slug) | `m-foo-1cbec737f863e492` / `m-foo-2c26b46b68ffc68f` (distinct even case-folded) |

Expected Defect 1 encoding correction: the colon is gone (every component is `m-` + `[a-z0-9_-]` + `-` + lowercase
hex) and no reserved basename can appear before a dot because there is no dot and the
component starts with `m-`. The measured Defect 2 pair `a/b` and `a__b` maps to distinct
directories, as does the case-variant pair `Foo`/`foo`. These are exact fixture results,
not proof that all possible module IDs map to distinct directories.

## Conflict Surface

Everything that must move in the same change set:

- `internal/pipeline/cache_store.go:161-174` — `sanitizeModuleID` is replaced by
  `encodeModuleDirName`. (Keeping a `slug` helper here or beside it is uncontroversial.)
- `internal/pipeline/cache_artifacts.go:402-404` — `moduleArtifactDir` is the **sole**
  production call site; it switches to `encodeModuleDirName`.
- `internal/pipeline/cache_artifacts_test.go:64-73` — the subtest
  `sanitized_collision_uses_exact_module_id` asserts the *old collision* (line 67 compares
  `sanitizeModuleID("a/b")` vs `sanitizeModuleID("a__b")`). It must be rewritten: first assert
  the **new** mapping separates `a/b` and `a__b`, then keep the stamp-backstop assertion that
  loading a colliding ID is rejected. Line 67's fixture assertion changes exactly here.
- `cmd/ailang/serve_api_mcp_surface_test.go:177,287,538-586` — the two callers of
  `requireCompileArtifactCache` and the helper itself, including the comment lines 538-549 and
  the Windows skip at 572. M3 retires the skip branch.
- The manifest (`CacheManifest`, `internal/pipeline/cache_store.go`) is keyed by `moduleID`
  and is **independent of the directory encoding**; it needs no schema change.
- `Clear()` (`internal/pipeline/cache_store.go:118`, removeAll at line 126) removes
  `<cs.dir>/modules` wholesale and therefore sweeps both old- and new-scheme directories
  together; verified (Verification Log, `Clear()` rows).

- `internal/pipeline/cache_runtime.go:78` — references `moduleArtifactDir` in the diagnostic
  path (Verification Log). Verified by `sed -n '66,92p' internal/pipeline/cache_runtime.go`
  that this line **merely passes the returned path string to `artifactErrorPath(err, …)` for a
  `warnWrite` diagnostic**; it treats the path as an opaque string and never splits, parses or
  pattern-matches it, so it **requires no change** under the new encoding.
- `cmd/ailang/serve_api_mcp_surface_test.go:602` — **a hand-rolled DUPLICATE of the encoding in
  another package**: `name := strings.NewReplacer("/", "__", "\\", "__").Replace(moduleID)`.
  It reimplements `sanitizeModuleID` rather than calling it, so it is invisible to a
  `grep sanitizeModuleID` and to the "outside `internal/pipeline`" framing, and it will compute
  the WRONG directory the moment the encoding changes. **It must move in the same change set**
  (M2 owns it): export a test-visible encoder or have the fixture call the real function. Found
  by the controller while measuring this objection; neither the author nor round 1 found it.

No other package **calls** `moduleArtifactDir` — a repo-wide grep returns hits only in
`internal/pipeline/{cache_artifacts.go,cache_runtime.go,cache_artifacts_test.go}` (Verification
Log). That is a claim about CALL SITES, not about consumers: the two bullets above are the
consumers that grep cannot see.

## Migration and compatibility

**Is renaming the scheme safe?** Yes. The artifact **stamp** is keyed by `module_id` and the
**manifest** is keyed by `moduleID`; neither depends on the directory encoding. When the
encoding changes, `moduleArtifactDir` computes a name that does not exist, `LoadArtifacts`
fails to find the stamp and returns an error, and the caller (`cacheRuntime.load`,
`internal/pipeline/cache_runtime.go:52-66`) logs `CACHE_INVALID`/`MISS` and **recompiles** the
module through the normal verified path. Evidence (Verification Log rows "Miss path
recompiles" and "**Non-verified path recompiles**"): `load` returns `(nil, entry, false)` on a
`LoadArtifacts` error, and in `internal/pipeline/pipeline_module.go:269-296` the **only** early
exit is the `continue` inside `verified && cached != nil`; the `else` branch on the miss side
consists entirely of `cacheMisses++` plus an optional `[CACHE] … INVALID, recompiling` /
`MISS` debug line, after which control falls out of the `if/else` into the ordinary compile
path. There is no error return, no panic and no silent-fallback branch on the miss side.
(Round 1 cited `269-294`; the measured range is `269-296` — the `load` call is at 269, the
verified branch runs to ~287 and the else-branch to ~296.) So an old-scheme directory is never
read as if it were a new hit — a "miss" is a clean recompile, never a wrong program.

**Do stale old-scheme directories leak?** Old-scheme directories remain until `Clear()`.
The declared 32 MiB artifact limit has not been demonstrated to bound their on-disk footprint,
including metadata and publication leftovers; this design makes no such guarantee. Aggregate
retained storage depends on the existing cache contents. Automated reclamation remains out of
scope, and users can reclaim these directories with `ailang cache clear`.
`Clear()` removes `<cs.dir>/modules` wholesale (verified: `internal/pipeline/cache_store.go:126`),
so both schemes are swept together and no scheme-specific cleanup code is needed.

> Round-2 note (gpt6-astra, applied verbatim under the narrow-refinement carve-out): round 2 of
> this doc asserted a **32 MiB per-module** orphan bound from the constant
> `maxModuleArtifactBytes`. The reviewer's catch stands — declaring and unit-testing a constant
> does not establish a total on-disk directory bound, and says nothing about stamps, temporary
> files, failure-path leftovers, or the aggregate across retained old-scheme directories. The
> numeric guarantee is **withdrawn**, not re-argued.

**Do we need a `cacheKeyVersion` bump?** No. `cacheKeyVersion` (`internal/pipeline/cache_key.go:28`,
currently `"v4"`) guards the **on-disk blob/stamp format** across format changes (each historical
bump in that file is a blob-decodability guard). This change alters only *directory names*, not
the gob/json payloads or the `artifactStamp` schema, and the miss path already recompiles
cleanly. The rename is **self-invalidating**: every existing entry is automatically a miss
under the new scheme. Bumping the version would be harmless but redundant; we do not bump.

## Q3 — what the fix must not break (confirmed first-party)

- `moduleArtifactDir` is the **only** production consumer of the encoded name (see Conflict
  Surface grep); switching it switches the whole cache.
- The stamp's exact-`module_id` comparison (`internal/pipeline/cache_artifacts.go:305`,
  `stamp.ModuleID != moduleID`) remains the correctness backstop and is **unchanged**; the
  stamp schema (`artifactStamp`, `cache_artifacts.go:35-40`: `Version`, `ModuleID`, `CacheKey`,
  `SHA256`) keeps the same fields and does **not** move.
- The `cmd/ailang` test fixture **does** construct the `modules/<component>` layout:
  `compileArtifactDir` at `serve_api_mcp_surface_test.go:601-604` duplicates the old encoder,
  and callers at lines 179 and 289 use its path. M2 must migrate this fixture to the real
  encoder, as already required by the Conflict Surface and sprint plan. The call-site grep
  for `moduleArtifactDir` does not establish the absence of consumers outside the package.
  The earlier blanket outside-pipeline claim is withdrawn; the intended compatibility
  boundary preserves stamp/manifest formats while changing this on-disk directory layout.

## Milestones

Each is independently committable and testable; each acceptance test below names the exact
production-code mutation it kills.

### M1 — introduce `encodeModuleDirName` + pure unit tests (no production wiring yet)
Add the new function and a `_test.go` table test exercising only the *function*: determinism,
exact separation over the worked-example set (`std/list`, `a/b` vs `a__b`, `Foo` vs `foo`), and
bounded length (≤ 57).
- Acceptance test `TestEncodeModuleDirName_InjectivityAndDeterminism` (unit, table-driven).
- **Mutation killed**: delete the `-<hex>` suffix computation (return `"m-" + slug(id)`, retaining the prefix).
  This table instantly turns red (`Foo` and `foo` become equal; `a/b` and `a__b` retain
  distinct slugs under the runs-preserved algorithm). This
  mutation is killed by M1's own code — M1's test is not vacuous for M1 because M1 introduces
  both.

### M2 — wire the encoding and rewrite the collision test
In `cache_artifacts.go`, point `moduleArtifactDir` at `encodeModuleDirName`; delete
`sanitizeModuleID`. **Also fix the hand-rolled duplicate encoder at
`cmd/ailang/serve_api_mcp_surface_test.go:602`** (`strings.NewReplacer("/", "__", "\\", "__")`),
which reimplements the old scheme in another package: have that fixture call the real encoder
rather than a copy, so the two can never diverge again. Leaving it is a silent break, not a
cosmetic one — the fixture would compute a directory the production code no longer uses. Rewrite `sanitized_collision_uses_exact_module_id`
(`internal/pipeline/cache_artifacts_test.go:64-73`) to (a) assert the new mapping separates
`a/b` and `a__b`, then (b) keep the stamp-backstop assertion that a stamp from one module is
refused under the other.
- Acceptance test: the rewritten subtest.
- **Mutation killed**: revert `moduleArtifactDir` to call `sanitizeModuleID`
  (reintroduce the collision). Then (a) fails — the two IDs map to the same directory.
  Critically, M1's pure unit test does NOT wire the production path, so M2's integration test
  is the first to kill this wiring mutation; it is **not vacuous**.

### M3 — Windows legality + retire the Windows skip
Add `TestEncodeModuleDirName_AllLegalOnWindows`: for a representative set of module IDs —
including a drive-letter path `C:/Users/runneradmin/x`, the bare `con`, the
**extension-carrying reserved basenames `CON.txt`, `nul.log`, `COM1.any`**, and one with a
trailing dot/space source — assert the produced component contains **none** of the forbidden
bytes `< > : " / \ | ? *`, contains no ASCII control char, does not end in `.`/space, and that
**the basename before the first dot** (i.e. `component[:index(".")]`, or the whole component
when there is no dot), upper-cased, is not in `CON PRN AUX NUL COM1-9 LPT1-9`. The test must
assert on that pre-dot basename, **not** on the whole component — checking only bare names is
the round-1 mistake. Additionally, guarded by `runtime.GOOS == "windows"`, the same table
performs an **actual `os.MkdirAll` of each component under `t.TempDir()`** and requires it to
succeed, so the Windows CI runner proves the legality claim against the real filesystem, not
just against our own rule list. Then delete the skip branch in
`requireCompileArtifactCache` (`cmd/ailang/serve_api_mcp_surface_test.go:572`) and update the
stale comment block (538-549). Once Windows publication actually works, an empty manifest is a
real failure on every platform, so the helper must fail (not skip).
- Acceptance test: the Windows-legality table (the string assertions run on a Linux/macOS dev
  box; the directory-creation leg runs on the Windows CI runner) + the two
  `requireCompileArtifactCache` callers now exercising real publication on the Windows CI runner.
- **Mutation killed** (two): (1) revert `encodeModuleDirName`'s colon/forbidden handling (e.g.
  replace the slug map with the old pass-through for those bytes) — the table turns red on a
  `:` or path byte. (2) **Drop the `m-` prefix and restore `.` to the slug alphabet** — the
  `CON.txt`/`nul.log`/`COM1.any` rows turn red because the pre-dot basename becomes `con`/`nul`/
  `com1`, and on the Windows runner the `MkdirAll` leg fails outright. This legality table is
  added by M3 and not asserted at M1/M2, so it is **not vacuous** — M1's table deliberately
  omits the Windows legality enumeration so this milestone owns it.

  > **Superseded by the sprint plan's stricter reading (judge finding, iteration 334).** This
  > paragraph uses "not vacuous" to mean *not redundant test coverage* — M1's table does not
  > exercise this input class, so M3's table adds something. The sprint plan applies the
  > mission's actual mutation-testing rule — a test belongs to a milestone only when reverting
  > THAT milestone's own production hunk turns it red — and by that rule M3's table is
  > **VACUOUS for M3's own diff**, because the hunk it kills landed in M1. The plan's verdict
  > governs; M3 ships test coverage and a skip deletion, and no new production behaviour. The
  > same concession applies to M4. Read the plan's non-vacuity ledger, not this sentence.

### M4 — stale-scheme sweep regression test + docs
Add `TestClear_SweepsArtifactDirectories`: populate artifact dirs (old-scheme and new-scheme
names) + a manifest entry, call `Clear()`, and assert both dirs and the manifest are gone.
Tie the design decision to a note in `cache_key.go` near `cacheKeyVersion` so a future engineer
sees why a version bump was *not* needed for the encoding change.
- Acceptance test: the `Clear()` sweep test.
- **Mutation killed**: revert `Clear()` to *not* remove `<cs.dir>/modules`
  (drop the `artifactIO.removeAll` at `cache_store.go:126`). The test then finds stale
  artifact dirs after `Clear()` and goes red. Note this asserts *existing* `Clear()` behaviour
  and is a regression guard, not a behavioural change.

## Test plan

| Test name | File | What it asserts | Mutation it kills |
|---|---|---|---|
| `TestEncodeModuleDirName_InjectivityAndDeterminism` | `internal/pipeline/cache_artifacts_test.go` (new unit) | deterministic; `a/b`≠`a__b` and `Foo`≠`foo` under new encoding; length ≤ 57 | deleting the `-<hex>` suffix (M1) |
| `TestCacheArtifacts.../sanitized_collision_uses_exact_module_id` (rewritten) | `internal/pipeline/cache_artifacts_test.go:64-73` | new mapping separates `a/b`/`a__b`; stamp-backstop still refuses a foreign stamp | reverting `moduleArtifactDir` to `sanitizeModuleID` (M2) |
| `TestEncodeModuleDirName_AllLegalOnWindows` | `internal/pipeline/cache_artifacts_test.go` (new) | output has no Windows-forbidden byte, no control char, no trailing dot/space; the **basename before the first dot** is not a reserved device name; fixtures incl. `C:/Users/…`, `con`, `CON.txt`, `nul.log`, `COM1.any`; on `GOOS=windows` also `os.MkdirAll` of every component succeeds | reverting the forbidden-byte mapping; dropping the `m-` prefix + restoring `.` to the slug (M3) |
| `requireCompileArtifactCache` callers (177, 287) | `cmd/ailang/serve_api_mcp_surface_test.go` | real publication on all platforms after skip deleted | reintroducing the colon leak (M3, on the Windows runner) |
| `TestClear_SweepsArtifactDirectories` | `internal/pipeline/cache_store_test.go` (new) | `Clear()` removes old- and new-scheme dirs and empties the manifest | reverting `Clear()`'s `modules` removeAll (M4) |

## Verification Log

### Iteration-336 source-attribution proof (post-round-2; no design revision)

Sol's alternative proof was reproduced exactly for its eight named source files. The original
worktree `/Users/voightkampff/dev/sunholo-data/.wt-v1-iter334` returned
`c2a9d8fb4abfadb472a5c05461f10a506f4a8013` from `git rev-parse HEAD`.
In `/Users/voightkampff/.ailang-driver-pin/.wt-v1-iter336`, `pwd`, `git rev-parse HEAD`, and
`git status --short` returned, respectively:

```text
/Users/voightkampff/.ailang-driver-pin/.wt-v1-iter336
e30904f71d59b8a6b93f10c3b8d77bc28bce4f48
 M design_docs/planned/v0_36_0/m-cache-module-id-encoding.md
```

Exact source-comparison command, run in the iteration-336 worktree:

```sh
git diff --exit-code c2a9d8fb4abfadb472a5c05461f10a506f4a8013..e30904f71d59b8a6b93f10c3b8d77bc28bce4f48 -- internal/pipeline/cache_store.go internal/pipeline/cache_artifacts.go internal/pipeline/cache_runtime.go internal/pipeline/pipeline_module.go internal/pipeline/cache_key.go internal/loader/loader.go internal/loader/stdlib_resolver.go cmd/ailang/serve_api_mcp_surface_test.go
```

Observed: **exit 0, no output**. These eight files are identical between the recorded commits;
the current status additionally shows the only working-tree change is this design document.
The positive identity controls are the two successful commit resolutions and the current
`pwd`/status output. This ties inherited observations about the eight named files to the
reviewed source snapshot; it does not claim equivalence for other files or convert round 2
into a pass. The historical Verification Log remains intact below.


Every codebase claim below was produced by the command shown; controls run where required per
R2. `$R` = repo root `/Users/voightkampff/dev/sunholo-data/.wt-v1-iter334`.

| Claim | Command (cwd `$R`) | Observed output |
|---|---|---|
| `sanitizeModuleID` replaces only `/` and `\` with `__` | `sed -n '160,175p' internal/pipeline/cache_store.go` | `// sanitizeModuleID converts a module ID …` `func sanitizeModuleID(moduleID string) string {` `// Replace path separators with double underscores` `case '/', '\\': result = append(result, '_', '_')` |
| Sole production call site is `moduleArtifactDir` | `grep -rn "sanitizeModuleID" --include=*.go .` | hits only in `cache_store.go:161-162` (def) and `cache_artifacts.go:403` (call), plus 1 test fixture at `cache_artifacts_test.go:67` and comment refs at `serve_api_mcp_surface_test.go:542,549,572` |
| `moduleArtifactDir` joins `modules` + encoded name | `sed -n '401,404p' internal/pipeline/cache_artifacts.go` | `return filepath.Join(cs.dir, "modules", sanitizeModuleID(moduleID))` |
| Collision `sanitizeModuleID("a/b")==sanitizeModuleID("a__b")` is asserted in the existing subtest | `sed -n '56,75p' internal/pipeline/cache_artifacts_test.go` | `if sanitizeModuleID("a/b") != sanitizeModuleID("a__b") { t.Fatal("fixture does not exercise the known directory collision") }` |
| Stamp backstop: `LoadArtifacts` refuses a mismatched `module_id` | `sed -n '/func (cs \*CacheStore) loadArtifacts/,/^}/p' internal/pipeline/cache_artifacts.go` | `if expectedCacheKey == "" \|\| stamp.Version != cacheKeyVersion \|\| stamp.ModuleID != moduleID \|\| stamp.CacheKey != expectedCacheKey { return nil, artifactFailure("verification", stampPath, …mismatch)) }` |
| Miss path recompiles (an artifact miss is a clean recompile, not wrong output) | `sed -n '52,66p' internal/pipeline/cache_runtime.go` | `cached, err := runtime.store.LoadArtifacts(moduleID, expectedKey)` `if err != nil { runtime.warnInvalid(moduleID, err); return nil, entry, false }` |
| `Clear()` removes `<cs.dir>/modules` wholesale | `sed -n '/func (cs \*CacheStore) Clear/,/^}/p' internal/pipeline/cache_store.go` | `if err := cs.artifactIO.removeAll(filepath.Join(cs.dir, "modules")); err != nil { return fmt.Errorf("remove compilation cache artifacts: %w", err) }` |
| `Clear()` line numbers (round 1 cited `118-132` and `129`; corrected) | `grep -n 'func (cs \*CacheStore) Clear\|removeAll(filepath.Join(cs.dir, "modules"))' internal/pipeline/cache_store.go` | `118:func (cs *CacheStore) Clear() error {` `126:	if err := cs.artifactIO.removeAll(filepath.Join(cs.dir, "modules")); err != nil {` |
| **Non-verified path recompiles** (migration safety; measured by the controller, transcribed) | `sed -n '260,300p' internal/pipeline/pipeline_module.go` | `moduleCacheKey = ModuleCacheKey(version.Commit, *mod.SourceContent, depDigests)` `cached, entry, verified := moduleCache.load(string(modID), moduleCacheKey)` `if verified { cacheHits++ // M-INCREMENTAL-TYPECHECK: Skip only after the artifact stamp and all payloads verify. if cached != nil { … compiledUnits[string(modID)] = unit; continue } } else { cacheMisses++; if cfg.DebugCompile { if entry != nil { fmt.Fprintf(os.Stderr, "[CACHE] %s: INVALID, recompiling (compiled %s ago)\n", …) } else { fmt.Fprintf(os.Stderr, "[CACHE] %s: MISS\n", modID) } } }` — the only early exit is the `continue` inside `verified && cached != nil`; every other path falls out of the `if/else` into the ordinary compile path. `load` call at line 269; verified branch to ~287; else-branch to ~296. |
| **A per-module WRITE cap exists; it does NOT bound the aggregate orphan footprint** — `maxModuleArtifactBytes` is exactly 32 MiB and test-pinned, but see the round-2 note in Migration: this row is evidence about a *write-path limit*, never about total retained on-disk bytes across old-scheme directories, stamps, or failure-path leftovers. Round 2 of this doc cited this row for the stronger claim and the stronger claim is **withdrawn** (measured by the controller, transcribed) | `grep -rn "maxModuleArtifactBytes" --include='*.go' internal/pipeline/` | `internal/pipeline/cache_artifacts.go:29:	maxModuleArtifactBytes int64 = 32 << 20` `internal/pipeline/cache_artifacts_test.go:193:	if maxArtifactBlobBytes != 16<<20 \|\| maxArtifactStampBytes != 64<<10 \|\| maxModuleArtifactBytes != 32<<20 {` — known-present control in the same breath (R2): `grep -rn "cacheKeyVersion" --include='*.go' internal/pipeline/ \| wc -l` → `14` |
| Stamp compare is at line 305 | `grep -n 'stamp.ModuleID != moduleID' internal/pipeline/cache_artifacts.go` | `305:	if expectedCacheKey == "" \|\| stamp.Version != cacheKeyVersion \|\| stamp.ModuleID != moduleID \|\| stamp.CacheKey != expectedCacheKey {` |
| `artifactStamp` schema is at lines 35-40 with fields `Version`, `ModuleID`, `CacheKey`, `SHA256` (round 1 cited `36-41`; corrected) | `grep -n 'type artifactStamp struct' internal/pipeline/cache_artifacts.go` then `sed -n '35,40p' …` | `35:type artifactStamp struct {` then `Version string \`json:"version"\`` `ModuleID string \`json:"module_id"\`` `CacheKey string \`json:"cache_key"\`` `SHA256 map[string]string \`json:"sha256"\`` `}` |
| `cacheKeyVersion` is declared at line 28 | `grep -n 'cacheKeyVersion = ' internal/pipeline/cache_key.go` | `28:const cacheKeyVersion = "v4"` |
| `CACHE_WRITE_FAILED … stage=publication` is a real diagnostic string in this repo (the Problem section quotes it) | `grep -rn 'CACHE_WRITE_FAILED' --include='*.go' internal/ cmd/ \| head -5` | `internal/pipeline/cache_invalidation_test.go:380: … "CACHE_WRITE_FAILED module=answer stage=publication" … "using fresh compilation"`, plus `stage=initialization` (422), `stage=manifest_save` (444), `stage=encoding` (`cache_artifacts_test.go:348`) |
| Worked-example suffixes for the four new reserved-basename fixtures | `for id in 'con.txt' 'CON.txt' 'nul.log' 'COM1.any'; do printf '%s' "$id" \| shasum -a 256 \| cut -c1-16; done` | `con.txt → d3bde286fd271ed6`, `CON.txt → 09c8cc7edcae01ac`, `nul.log → c0294fbf8537502a`, `COM1.any → bdd82f44de519430` (round-1 suffixes were not recomputed; the `m-` prefix is outside the hash input) |
| Nothing outside `internal/pipeline` **calls** `moduleArtifactDir` (a CALL-SITE claim, not a consumer claim — see the two Conflict Surface bullets) | `grep -rln "moduleArtifactDir" --include='*.go' .` | `internal/pipeline/cache_artifacts_test.go`, `internal/pipeline/cache_runtime.go`, `internal/pipeline/cache_artifacts.go` — 3 files, all in `internal/pipeline`. Known-present control in the same breath: `grep -rln "sanitizeModuleID" --include='*.go' .` → 4 files INCLUDING `cmd/ailang/serve_api_mcp_surface_test.go`, so the instrument does reach `cmd/`. |
| **R2/oc-glm-5-2**: `cache_runtime.go:78` treats the encoded path as an opaque string (measured by the controller, transcribed) | `sed -n '66,92p' internal/pipeline/cache_runtime.go` | `runtime.warnWrite(stage, moduleID, artifactErrorPath(err, runtime.store.moduleArtifactDir(moduleID)), err)` — the value is passed straight into a diagnostic; no `Split`, `TrimPrefix`, `Replace` or pattern match anywhere in the function. Requires no change. |
| **R2/controller, found while measuring oc-glm-5-2's objection**: a hand-rolled duplicate of the encoding exists in ANOTHER package and does not call `sanitizeModuleID` | `grep -rn 'Split.*"__"\|TrimPrefix.*modules\|Replace.*"__"' --include='*.go' internal/ cmd/` | `cmd/ailang/serve_api_mcp_surface_test.go:602:	name := strings.NewReplacer("/", "__", "\\\\", "__").Replace(moduleID)` (plus two unrelated Firestore-id hits in `internal/server/auth/workspace.go`, which are the known-present control that the pattern fires). This is a SECOND implementation of the directory encoding and breaks silently when the encoding changes. |
| **R2/gemini-3-1-pro**: `Clear()` DOES reset the manifest, so M4's test assertion is sound (measured by the controller, transcribed) | `sed -n '117,132p' internal/pipeline/cache_store.go` | `cs.manifest = &CacheManifest{ Version: cacheKeyVersion, Entries: make(map[string]*CacheEntry) }` then `if err := cs.Save(); err != nil { return fmt.Errorf("save empty compilation cache manifest: %w", err) }` then `cs.artifactIO.removeAll(filepath.Join(cs.dir, "modules"))`. The reviewer's conditional ("if `Clear()` does not actually clear the manifest") is REFUTED: it resets the in-memory manifest AND persists it, before removing the artifact tree. M4's test needs no weakening. |
| `cacheKeyVersion` guards blob format, currently `v4` | `sed -n '1,35p' internal/pipeline/cache_key.go` | `const cacheKeyVersion = "v4"`; header comment: "bumped when the cache format changes, invalidating all entries" |
| `validateModuleName` allows mixed-case `[a-zA-Z0-9_/-]` for stdlib resolution | `sed -n '25,79p' internal/loader/stdlib_resolver.go` | `validPattern := regexp.MustCompile(\`^[a-zA-Z0-9_/-]+$\`)`; suspicious list also rejects `c:`, `C:`, UNC |
| Modules keyed by canonical `module.Path` | `sed -n '486,513p' internal/loader/loader.go` | `// Store with canonical ID (module.Path), not input path` `modules[module.Path] = module` |
| Windows skip branch + its two callers exist in serve test | `grep -n "requireCompileArtifactCache\|sanitizeModuleID" cmd/ailang/serve_api_mcp_surface_test.go` | callers at lines 177 and 287; `sanitizeModuleID` comment refs at 542, 549; skip at 572 |
| **Negative control** (R2): a fabricated symbol is absent, with a known-present control in the same breath | `grep -rn "fabricatedSymbolDefinitelyAbsentXYZ" --include=*.go . \| wc -l` then `grep -rln "sanitizeModuleID" --include=*.go .` | first output `0`; positive control returns `./cmd/ailang/serve_api_mcp_surface_test.go`, `./internal/pipeline/cache_store.go`, `./internal/pipeline/cache_artifacts_test.go`, `./internal/pipeline/cache_artifacts.go` (4 files) — the same 4-file set shown in the call-site row, so the search instrument is known-good. No admissible "found nothing" claim was made without this control. |

### Iteration 336 clarification — measured 2026-09-06

Commands in this subsection ran from
`/Users/voightkampff/.ailang-driver-pin/.wt-v1-iter336`. These measurements clarify the
proposed algorithm; they are not evidence that the production encoder is implemented.

| Claim | Command | Observed output |
|---|---|---|
| Q3 has an outside-pipeline layout consumer requiring the already-planned M2 migration | `sed -n '601,604p' cmd/ailang/serve_api_mcp_surface_test.go` and `rg -n 'compileArtifactDir' cmd/ailang/serve_api_mcp_surface_test.go` | Body: `name := strings.NewReplacer("/", "__", "\\", "__").Replace(moduleID)` followed by `return filepath.Join(cacheRoot, "compile", "modules", name)`. Callers at 179 and 289; definition at 601. This positive evidence replaces the former negative claim. |
| M2 already owns the real-encoder migration | `sed -n '175,195p' design_docs/planned/v0_36_0/m-cache-module-id-encoding-sprint-plan.md` | Work items explicitly expose the real encoder through a narrow test-visible API and replace `compileArtifactDir`'s `strings.NewReplacer` copy with a call to it; a matching copy is insufficient. |
| All worked examples, ASCII/UTF-8 byte handling, empty slug, and trim-before-cut order | Reference command below | Full output below: all 11 inherited suffixes match; the Windows slug has **two** underscores after `c`; the trim/cut fixture ends its 38-byte slug with `_` and produces a 57-byte component. |

Additional revision measurements after iteration-336 quorum round 1:

| Claim | Command | Observed output |
|---|---|---|
| `CacheEntry` and `CacheManifest` have no dedicated directory-name/path fields | `sed -n '/type CacheEntry struct/,/}/p' internal/pipeline/cache_store.go` and `sed -n '/type CacheManifest struct/,/}/p' internal/pipeline/cache_store.go` | Complete entry fields: `CacheKey string`, `IfaceDigest string`, `IfaceJSON []byte`, `CompileTimeMs int64`, `Timestamp time.Time`; manifest fields: `Version string`, `Entries map[string]*CacheEntry`. The complete bodies are the negative evidence; the present `CacheKey` and `Entries` fields are positive controls proving the reads reached both schemas. |
| `validateModuleName` is invoked by stdlib resolution; it is not established as a guard for every absolute-path ID | `rg -n 'validateModuleName\(' . --glob '*.go'` and `sed -n '147,170p' internal/loader/stdlib_resolver.go` | Production definition at `stdlib_resolver.go:25`, sole non-test call at `:163`, inside `StdlibResolver.ResolveStdlib(moduleName string)` before removing the `std/` prefix. Remaining hits are in `stdlib_resolver_test.go` and `stdlib_entry_path_test.go`; the definition and production call are positive controls for the absence of other production callers. |
| Loader map assignment uses the stored module path directly | `sed -n '486,513p' internal/loader/loader.go` | `// Store with canonical ID (module.Path), not input path` followed by `modules[module.Path] = module`. This verifies the assignment, not a universal claim about all upstream path normalization. |
| Validator restrictions apply to that stdlib-style name input | `sed -n '25,79p' internal/loader/stdlib_resolver.go` | Body rejects `strings.Contains(name, "..")` and null bytes, uses `^[a-zA-Z0-9_/-]+$`, then checks `filepath.IsAbs(name)` and suspicious patterns. |
| Captured baseline log does not establish the illustrated Windows diagnostic | `rg -n 'CACHE_WRITE_FAILED' /tmp/v1-iter336-windows-base.log` then positive control `rg -n -m 1 'Runner Image' /tmp/v1-iter336-windows-base.log` | First query: no matches. Positive control: `2:2026-09-06T01:49:01.8360088Z ##[group]Runner Image Provisioner`. Controller supplied this log from job `101410083473`; no runtime error transcript is claimed. |
| Existing foreign-module stamp rejection routes to the existing invalid-cache diagnostic and recompilation | `sed -n '295,310p' internal/pipeline/cache_artifacts.go`, `sed -n '52,66p' internal/pipeline/cache_runtime.go`, `sed -n '82,115p' internal/pipeline/cache_runtime.go`, and `sed -n '269,298p' internal/pipeline/pipeline_module.go` | Stamp comparison includes `stamp.ModuleID != moduleID` and returns `artifactFailure("verification", stampPath, fmt.Errorf("stamp authorization mismatch"))`. Load error calls `runtime.warnInvalid(moduleID, err)` and returns `(nil, entry, false)`. `warnInvalid` prints `CACHE_INVALID module=%s path=%s reason=%s` and `; recompiling` (deduplicated per module). The pipeline's unverified branch increments misses and falls through; the cached-unit `continue` is inside the verified branch. |

Reference command (stdlib-only calculation, no repository files written):

```sh
python3 - <<'PYTHON'
import hashlib
ids = ['std/list', 'a/b', 'a__b', 'C:/Users/runneradmin/x', 'con', 'con.txt', 'CON.txt', 'nul.log', 'COM1.any', 'Foo', 'foo', '', '///', '__A__', 'aéB', '__' + 'A' * 37 + '/B__']
for module_id in ids:
    raw = module_id.encode('utf-8')
    lowered = bytes(b + 32 if 65 <= b <= 90 else b for b in raw)
    mapped = bytes(b if 97 <= b <= 122 or 48 <= b <= 57 or b in (95, 45) else 95 for b in lowered)
    slug = mapped.strip(b'_')[:38].decode('ascii')
    component = 'm-' + slug + '-' + hashlib.sha256(raw).hexdigest()[:16]
    print(repr(module_id), repr(slug), component, len(component))
PYTHON
```

Observed output (input, slug, component, component byte length):

```text
'std/list' 'std_list' m-std_list-d9997702a41d1e11 27
'a/b' 'a_b' m-a_b-c14cddc033f64b9d 22
'a__b' 'a__b' m-a__b-63e5c1c455d01d5c 23
'C:/Users/runneradmin/x' 'c__users_runneradmin_x' m-c__users_runneradmin_x-81fb5218f110e3cc 41
'con' 'con' m-con-1143da2bc54c495c 22
'con.txt' 'con_txt' m-con_txt-d3bde286fd271ed6 26
'CON.txt' 'con_txt' m-con_txt-09c8cc7edcae01ac 26
'nul.log' 'nul_log' m-nul_log-c0294fbf8537502a 26
'COM1.any' 'com1_any' m-com1_any-bdd82f44de519430 27
'Foo' 'foo' m-foo-1cbec737f863e492 22
'foo' 'foo' m-foo-2c26b46b68ffc68f 22
'' '' m--e3b0c44298fc1c14 19
'///' '' m--732c4e9711639ed1 19
'__A__' 'a' m-a-6530635bd09a76b6 20
'aéB' 'a__b' m-a__b-a65e5ade07551d5e 23
'__AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA/B__' 'aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa_' m-aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa_-753374029f5ccda0 57
```

## Quorum verification log

### Iteration 336 round 2 — BLOCKED; design phase PARKED pending D-57

Artifact read in full with `cat /tmp/v1-iter336-quorum-r2/*.json`:
`/tmp/v1-iter336-quorum-r2/m-cache-module-id-encoding-2026-09-06T05-47-18Z.json`.
Observed synthesis: **blocked**, two rejects, one pass, no absent reviewers; total cost
**$0.13907585**. The controller recorded an in-session pass; this did not override the
reviewer rejections. Round 1 plus round 2 cost **$0.28068117**. No third design revision or
third quorum is being performed in this phase.

| Reviewer | Recorded verdict / request | Disposition |
|---|---|---|
| `gpt5-6-sol` | **reject**: inherited verification comes from another worktree without source-identity proof; proposed either remeasurement or the exact eight-file commit diff. | Reproduced the proposed eight-file diff: exit 0, no output; both commit hashes and current worktree status are recorded at the start of Verification Log. This applies only the requested attribution proof and does not change the blocked verdict. |
| `gemini-3-1-pro` | **pass**, with a request to make M2's artificial mismatched-stamp fixture explicit now that the natural pair separates. | Preserved as review advice for any authorized continuation; no implementation instruction is changed in this parking disposition. |
| `oc-glm-5-2` | **reject**: disputes the full-prefix slug's readability value; requests either a basename-plus-parent redesign with new fixtures or dropping the slug for pure hashing. | This disputes the design direction. Neither alternative is adopted here, and the narrow-refinement carve-out cannot close it. Escalated to D-57; encoder direction remains unchanged and implementation is parked. |

Two factual qualifications to GLM's reasoning are recorded without treating them as approval:
under the normative byte algorithm, **one** `/` or `\` byte maps to **one** `_`; two separators
map to `__`, as do the two UTF-8 bytes of `é`. Its single-separator-to-two-underscores premise
is incorrect. Also, lossy mapping does not logically imply zero readability: the measured
`C:/Users/runneradmin/x` fixture retains `c__users_runneradmin_x`. This illustrates a limited
readable prefix, not demonstrated usability for the production path distribution. Neither
qualification satisfies GLM's requested alternatives or overrides its direction rejection.

**D-57 decision requested:** keep the hybrid with explicitly limited readable-prefix value
(recommended), choose pure hashing, or redesign around basename plus parent directory.
The default is **PARK**: no production code, no M1 execution, and no direction change until
a human ruling. This paragraph records the pending decision, not a human answer. The earlier
round-1 text referring to an upcoming re-quorum is retained as history; this round-2 parking
status supersedes it. The controller owns the decision ledger and further routing.


### Iteration 336 round 1 — blocked; bounded revision pending re-quorum

Artifact read in full:
`/tmp/v1-iter336-quorum-r1/m-cache-module-id-encoding-2026-09-06T05-39-39Z.json`
(command: `cat /tmp/v1-iter336-quorum-r1/*.json`). Observed synthesis: **blocked**, two
rejects, one pass, no absent reviewers; total cost **$0.14160532**. The controller's recorded
in-session pass did not override either rejection. This revision does not claim a new verdict.

| Reviewer | Recorded verdict and objection | Bounded revision / remaining disagreement |
|---|---|---|
| `gpt5-6-sol` | **reject**: bounded lossy slug plus 64-bit suffix cannot satisfy injectivity; requests verification of Windows observation, full SHA-256 and explicit collision handling. | Replaced impossible uniqueness guarantees with collision resistance and finite-fixture separation; explicitly accepts theoretical contention/recompilation. Withdrew captured-Windows-outage claim. Full-256/new-diagnostic proposal is recorded as a contested alternative above, **not adopted**, awaiting independent re-quorum. |
| `gemini-3-1-pro` | **reject**: missing `CacheEntry` schema inspection leaves the migration premise unverified. | Read both complete structs and logged fields with positive controls; neither stores a dedicated directory-name/path field. |
| `oc-glm-5-2` | **pass**, with objection to unverified captured Windows diagnostic and catch about validator scope. | Adopted its option (b): label diagnostic an inferred reconstruction with no captured CI excerpt; fetched baseline had no diagnostic match. Verified the sole production validator caller is stdlib resolution and qualified all associated scope claims. |

The normative M1 suffix-removal mutation now returns `"m-" + slug(id)`, retaining the prefix.
The algorithm, 38-byte slug cap, 16-hex-character suffix, and 57-byte total cap are unchanged.
No production code or new diagnostic is introduced by this revision.

### Historical iteration-334 quorum record

**Round 1 (2026-09-06): 3 reviewers, 3 rejects. Design direction (hybrid slug + hash) undisputed
by all three; blocked on one substantive defect and one shared verification debt.**

| Reviewer | Strongest objection (one line) | Change made in round 2 |
|---|---|---|
| gpt6-astra | The encoding still yields illegal Windows names: `con.txt` → `con.txt-<hex>` keeps reserved basename `con` before the dot; "ends in `-<hex>`" is a false legality argument and M3 tested only bare names. | Output shape is now `m-<slug>-<16hex>` (unconditional safe prefix); slug cap 40 → 38 so the 57-char bound holds; `.` dropped from the slug alphabet (mapped to `_`) as a second, independent guard; legality bullet (a) rewritten around the prefix and states explicitly that the round-1 argument was false; worked examples gain `con.txt`, `CON.txt`, `nul.log`, `COM1.any` with measured suffixes; M3 and its test-plan row now assert on the basename **before the first dot**, carry those fixtures, add a real `os.MkdirAll` leg on the Windows runner, and name the prefix/dot mutation they kill. Hex suffixes of existing rows unchanged (the prefix is outside the hash input). |
| gemini-3-1-pro | `pipeline_module.go:269-294` ("treats non-verified as a recompile") and `maxModuleArtifactBytes` (`~32 MiB`) appear nowhere in the Verification Log, yet carry the migration-safety and no-GC decisions. | Both premises measured first-party by the controller and transcribed as Verification Log rows with exact commands and outputs; Migration section now cites them, corrects the range to `269-296`, and says "32 MiB" (exact, `32 << 20`, test-pinned at `cache_artifacts_test.go:193`), not "~32 MiB". |
| oc-glm-5-2 | Same point: the no-`cacheKeyVersion`-bump conclusion rests on an unverified claim that the miss path recompiles rather than errors, panics or silently falls back. | Same rows as above; the Migration text now states from the quoted excerpt that the only early exit is the `continue` under `verified && cached != nil`, and that the else-branch is `cacheMisses++` plus a debug line with no error return, panic or fallback. |

**Sweep for the shared root cause** (a sentence reasoning about a file not opened): every
remaining line-number pointer in the doc was re-measured and either confirmed or corrected in
new Verification Log rows — `Clear()` removeAll is at line 126 (round 1 said 129),
`artifactStamp` is at 35-40 (round 1 said 36-41), stamp compare at 305 and `cacheKeyVersion`
at 28 confirmed, the quoted `CACHE_WRITE_FAILED … stage=publication` string confirmed present.
No claim about repository behaviour now lacks a log row.

### Round 2 (2026-09-06, after the revision above) — BLOCKED 3/3, all three answered under the narrow-refinement carve-out

All three reviewers were PRESENT this round (`absent_reviewers` empty; round 1 had `gpt6-astra`
absent on `budget`, re-run alone with a raised cap and folded in before acting). Every round-2
objection landed on a DIFFERENT surface, each carried a concrete reviewer-authored
`proposed_fix`, and none disputed the design direction — so the controller applied the
reviewers' own fixes rather than parking (Gate-2 narrow-refinement carve-out).

| Reviewer | Round-2 strongest objection | Disposition |
|---|---|---|
| gpt6-astra | Declaring and unit-testing `maxModuleArtifactBytes` does not establish a total on-disk bound for orphan directories (no accounting for stamps, temp files, failure-path leftovers, or aggregate across retained dirs). | Reviewer's replacement paragraph applied **VERBATIM**; the numeric guarantee is withdrawn, not re-argued. |
| gemini-3-1-pro | M4 asserts `Clear()` "empties the manifest" on an unverified premise; if it only sweeps `modules/`, M4's test fails on its own assertions. | **Premise MEASURED first-party by the controller and REFUTED in the doc's favour**: `Clear()` resets the in-memory manifest and `Save()`s it before `removeAll`. Verification Log row added; M4 unchanged. |
| oc-glm-5-2 | Conflict Surface conflates "no other package" with "no other call site"; `cache_runtime.go:78` consumes the encoded name and is never listed or dismissed. | Reviewer's bullet template applied and filled from measurement: `:78` treats the path as an opaque diagnostic string and needs no change. **The objection also led the controller to a defect nobody had found** — `cmd/ailang/serve_api_mcp_surface_test.go:602` hand-rolls the old encoding in another package and is invisible to every grep the doc ran. Added to the Conflict Surface and assigned to M2. |
