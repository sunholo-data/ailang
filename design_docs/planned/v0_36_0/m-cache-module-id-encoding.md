# M-CACHE-MODULE-ID-ENCODING — replace `sanitizeModuleID`'s broken, non-injective mapping

**Status**: Planned
**Created**: 2026-09-06
**Mission**: V1, iteration 334
**Supersedes queue rows**: `m-cache-sanitize-module-id-windows-colon` (iter-333),
`m-cache-sanitize-module-id-collision` (iter-328)

## Problem

`sanitizeModuleID` (`internal/pipeline/cache_store.go:161`) is the function that turns a
compiled module's ID into the single directory component holding that module's compile
artifacts under `<cache>/compile/modules/`. It replaces only `/` and `\` with `__` and passes
every other byte through unchanged. Its sole production consumer is
`moduleArtifactDir` (`internal/pipeline/cache_artifacts.go:402-404`).

Two distinct, separately-filed defects, both on this one function, with different severities:

**Defect 1 — Windows: the artifact cache publishes nothing at all (TOTAL PLATFORM OUTAGE).**
On Windows a module ID is the canonical absolute source path and begins with a drive letter,
e.g. `C:/Users/runneradmin/AppData/...`. After `sanitizeModuleID` that whole string becomes
**one** path component `C:__Users__runneradmin__AppData__...` — which contains a **colon**.
Windows forbids `:` in a path component (`< > : " / \ | ? *`, plus ASCII control chars,
trailing dot/space, and the reserved `CON PRN AUX NUL COM1-9 LPT1-9` names are all illegal —
and a reserved name stays reserved when followed by an extension: `con.txt` has basename `con`
and is just as illegal as bare `con`).
So `StoreArtifacts` fails on every publication, and the M1 diagnostic from
`m-compile-cache-unverified-artifacts` reports it verbatim on the CI runner:

```
CACHE_WRITE_FAILED module=C:/Users/... stage=publication
path=...\compile\modules\C:__Users__runneradmin__... : ARTIFACT_INVALID
```

The whole compile-artifact cache is therefore **non-functional on Windows**. This is
PRE-EXISTING: `sanitizeModuleID` predates the sprint; the M1 diagnostic only made it audible.
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

macOS APFS and Windows NTFS are case-INSENSITIVE by default. Module IDs are **not**
case-normalized anywhere: the loader keys modules by the canonical `module.Path`
(`internal/loader/loader.go:513`) and `validateModuleName`
(`internal/loader/stdlib_resolver.go:25`) explicitly allows `[a-zA-Z0-9_/-]`. So two module IDs
differing only in case are byte-distinct and currently legal. On a case-insensitive filesystem
a byte-injective-but-case-preserving mapping would still collide at the directory level. Any
chosen scheme must be distinct even after the *filesystem* has case-folded. (Not a correctness
defect either — the stamp `module_id` backstop still protects — but it is a silent recompile
churn source that a robust scheme should remove.)

## Non-goals

- **Not adversarial cache hardening.** We are not defending against a hostile peer who can
  write arbitrary paths under the cache root. That is the separate row
  `m-cache-artifact-adversarial-decode`. (Note: `validateModuleName` already rejects `..`,
  null bytes, and only permits `[a-zA-Z0-9_/-]`, but module IDs are resolved absolute paths —
  see Conflict Surface — so defence in depth remains the adversarial row's job.)
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
(b) injective over distinct module IDs;
(c) distinct even after a case-insensitive filesystem folds case;
(d) bounded well under Windows' 255-char component / 260-char `MAX_PATH` limits;
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

### Option B — pure content hash (fixed-length, always legal, injective-in-practice) 
`encode(id) = sha256(id)[:N]` lowercase hex. Pros: constant output length (independent of a
long `C:/Users/…` input), inherently legal (hex alphabet), injective in practice (see collision
argument), and a lowercase-hex alphabet is stable even under a case-insensitive filesystem, so
two case-variant IDs get distinct lowercase-hex suffixes. Cons: the cache directory becomes
*unreadable to a human* — you cannot tell `std/list` from `std/map` at a glance, which makes
debugging on-disk state materially harder. Pure hash alone would discard all debuggability for
no correctness gain over a hybrid.

### Option C — hybrid: truncated human-readable slug + hash-suffix over the FULL module ID (CHOSEN)
Keep a shortened, sanitised, truncated slug for human debuggability and append a short
lowercase-hex digest derived from the **full** module ID for guaranteed distinctness. The slug
is purely cosmetic: uniqueness must never rest on it (it is truncated and case-folded, so it
can collide); the suffix is the authority. Detailed in Chosen design.

**Why C over B:** identical injectivity, identical legality, identical case-safety, identical
bounded length — and you keep the ability to read which module lives in which directory.
The cost is a tiny slug-splitting pass that a one-line helper performs. C dominates B.

## Chosen design

Replace `sanitizeModuleID` with an `encodeModuleDirName(string) string` (new function) computed
as `"m-" + slug(id) + "-" + hex(sha256(id)[:16])`:

```go
// encodeModuleDirName maps a module ID to a legal, injective, case-insensitive-FS-safe
// single directory component. The unconditional "m-" prefix is the Windows reserved-name
// guard (see legality). The slug is a truncated readability aid only; the 16-hex-char
// SHA-256 prefix over the FULL module ID is the uniqueness authority.
func encodeModuleDirName(moduleID string) string {
    sum := sha256.Sum256([]byte(moduleID))
    suffix := hex.EncodeToString(sum[:])[:16]     // 64 bits, lowercase alpha -> case-insensitive-FS safe
    return "m-" + slug(moduleID) + "-" + suffix
}

// slug lowercases and keeps [a-z0-9_-], mapping every other byte (and any run of them,
// INCLUDING '.') to '_'. Then it trims leading/trailing '_' so no component ends in an
// underscore run. Finally it truncates to 38 bytes (ASCII-only => safe byte cut).
func slug(id string) string { /* lower; rune-map; trim; cut at 38 */ }
```

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
  readability — `validateModuleName` permits no `.` in module names, and dots in absolute-path
  IDs are directory-name noise. The suffix alphabet is `[0-9a-f]`; no forbidden byte survives
  either part. The slug never ends in `.` or space because neither is in its alphabet.
- **(b) injectivity in practice**: distinct module IDs map to distinct 16-hex-char (64-bit)
  suffixes with overwhelmingly high probability. Birthday bound: collision probability ≈
  n² / 2⁶⁵; at n = 10⁶ directory entries that is ≈ 3e-8. If two IDs ever did collide the
  suffix would agree, but the slug additionally differs for every pair that differs within
  their first 38 characters, so the real distinguishing width is larger than 64 bits for
  realistic IDs. A collision, even if one occurred, degrades to the existing stamp-backstop
  recompile, never to a wrong program (Q3).
- **(c) case-collision safe**: `slug` is lowercased and the suffix is lowercase hex over the
  **original bytes**. `Foo` and `foo` produce different lowercase-hex suffixes
  (`1cbec737f863e492` vs `2c26b46b68ffc68f`), so the two directories remain distinct even on
  an APFS/NTFS case-insensitive volume, because the *alphabet* of the differing part has no
  cases to fold.
- **(d) bounded**: 57 chars max.
- **(e) deterministic**: a pure function of `moduleID`; identical on every run and platform.

Worked examples (suffix = first 16 hex of SHA-256 over the raw module ID; the `m-` prefix
does not enter the hash, so the suffixes are unchanged from round 1):

| module ID | slug (38 cap, lower, mapped) | resulting directory component |
|---|---|---|
| `std/list` | `std_list` | `m-std_list-d9997702a41d1e11` |
| `a/b` | `a_b` | `m-a_b-c14cddc033f64b9d` |
| `a__b` | `a__b` | `m-a__b-63e5c1c455d01d5c` |
| `C:/Users/runneradmin/x` | `c_users_runneradmin_x` | `m-c_users_runneradmin_x-81fb5218f110e3cc` (no colon ⇒ legal on Windows) |
| `con` | `con` | `m-con-1143da2bc54c495c` (basename `m-con-…`, never `con`) |
| `con.txt` | `con_txt` | `m-con_txt-d3bde286fd271ed6` (legal: no dot survives, and the basename before any dot starts with `m-`, not `con`) |
| `CON.txt` | `con_txt` (same slug as `con.txt`) | `m-con_txt-09c8cc7edcae01ac` (legal for the same reason; distinct from `con.txt` via the suffix) |
| `nul.log` | `nul_log` | `m-nul_log-c0294fbf8537502a` (legal: basename before any dot is `m-nul_log-…`, never `nul`) |
| `COM1.any` | `com1_any` | `m-com1_any-bdd82f44de519430` (legal: basename before any dot is `m-com1_any-…`, never `COM1`) |
| `Foo` / `foo` | `foo` / `foo` (same slug) | `m-foo-1cbec737f863e492` / `m-foo-2c26b46b68ffc68f` (distinct even case-folded) |

Defect 1 fixed: the colon is gone (every component is `m-` + `[a-z0-9_-]` + `-` + lowercase
hex) and no reserved basename can appear before a dot because there is no dot and the
component starts with `m-`. Defect 2 fixed: `a/b` and `a__b` now map to distinct dirs.
Case trap closed: `Foo`/`foo` separate.

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
- Nothing outside `internal/pipeline` reads the `modules/<component>` directory layout
  (verified by grep). No external contract changes.

## Milestones

Each is independently committable and testable; each acceptance test below names the exact
production-code mutation it kills.

### M1 — introduce `encodeModuleDirName` + pure unit tests (no production wiring yet)
Add the new function and a `_test.go` table test exercising only the *function*: determinism,
injectivity over the worked-example set (`std/list`, `a/b` vs `a__b`, `Foo` vs `foo`), and
bounded length (≤ 57).
- Acceptance test `TestEncodeModuleDirName_InjectivityAndDeterminism` (unit, table-driven).
- **Mutation killed**: delete the `-<hex>` suffix computation (make output just `slug(id)`).
  This table instantly turns red (`a/b` vs `a__b`, and `Foo` vs `foo`, become equal). This
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
| `validateModuleName` allows mixed-case `[a-zA-Z0-9_/-]` (case not normalised) | `sed -n '25,79p' internal/loader/stdlib_resolver.go` | `validPattern := regexp.MustCompile(\`^[a-zA-Z0-9_/-]+$\`)`; suspicious list also rejects `c:`, `C:`, UNC |
| Modules keyed by canonical `module.Path` | `sed -n '486,513p' internal/loader/loader.go` | `// Store with canonical ID (module.Path), not input path` `modules[module.Path] = module` |
| Windows skip branch + its two callers exist in serve test | `grep -n "requireCompileArtifactCache\|sanitizeModuleID" cmd/ailang/serve_api_mcp_surface_test.go` | callers at lines 177 and 287; `sanitizeModuleID` comment refs at 542, 549; skip at 572 |
| **Negative control** (R2): a fabricated symbol is absent, with a known-present control in the same breath | `grep -rn "fabricatedSymbolDefinitelyAbsentXYZ" --include=*.go . \| wc -l` then `grep -rln "sanitizeModuleID" --include=*.go .` | first output `0`; positive control returns `./cmd/ailang/serve_api_mcp_surface_test.go`, `./internal/pipeline/cache_store.go`, `./internal/pipeline/cache_artifacts_test.go`, `./internal/pipeline/cache_artifacts.go` (4 files) — the same 4-file set shown in the call-site row, so the search instrument is known-good. No admissible "found nothing" claim was made without this control. |

## Quorum verification log

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
