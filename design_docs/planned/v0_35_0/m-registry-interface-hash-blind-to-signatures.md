# M-REGISTRY-INTERFACE-HASH-BLIND-TO-SIGNATURES

**Status:** planned
**Target release:** v0.35.0
**Scope:** 4 days (decomposed in Milestones)
**HEAD measured at:** `bd17d9643a527ad13261354563fed590c1baf0fa`

## Summary

`pkg.InterfaceHash` (the value persisted as `interface_hash` in the lockfile, registry
metadata, coordinator task rows and pubsub payloads) folds exactly five things into a SHA256:
package name, edition, the `ailang` version string, the sorted exported **module paths**, and
the sorted max effects. It folds **no signature, type, arity or parameter data**. As a result the
hash is invariant under adding, removing, or retyping an exported function inside an already-listed
module. The registry's own cascade therefore cannot see an API addition (it classifies it `patch` /
`A`), and — worse — cannot see a **breaking** removal or retype at all. The sibling agent
`ailang-parse` observed this in the wild: `sunholo/external_backend` 0.1.0→0.2.0 added
`export pure func describeProcessError` and the `interface_hash` was byte-identical.

This design makes the interface hash sensitive to the actual exported signatures, and — the hard
part — does so without (a) flooding every consumer with a false "interface changed" wave for
already-published packages, (b) silently suppressing that wave (a CLAUDE.md §2 violation), or
(c) letting the publisher and the registry validator disagree about a package's identity. It also
introduces a **signature set** so the cascade can classify a delta as *additive* vs *breaking*
rather than only "changed or not". The hard part — the transition window, where pre-migration
packages have no trustworthy old side — is answered with an explicit **`unknown`** class, not an
informational fallback (OBJ-1/OBJ-3).

## The Defect (verified)

- **V1.** `InterfaceHash` (`internal/pkg/hasher.go:73-100`) folds name, edition, `ailang` version,
  sorted `m.Exports.Modules`, sorted max effects. No signature/type/arity/param data.
- **V2.** `ExportConfig.Modules` is `[]string` — module paths only (`internal/pkg/manifest.go:141`).
- **V3.** Consequence by construction: hash invariant under (a) adding an export, (b) removing one,
  (c) retyping one. Only (a) observed in the wild; (b)/(c) follow from V1+V2.
- **V4.** `EmitInterfaceChangeNotice` short-circuits on hash equality
  (`internal/messaging/pkg_events.go:56-58`), so no notice fires and the release is classified
  `patch`.
- **V5.** Field report from `ailang-parse` (2026-08-31, v0.34.0-214-g746ba72d6): `external_backend`
  0.1.0→0.2.0 added `describeProcessError`; `interface_hash` byte-identical; `content_hash`
  changed; dependent notified `patch`; the new export is genuinely importable (smoke 15/15).
- **V6.** Six non-test call sites of `InterfaceHash(` across two binaries (publisher + validator).
- **V7.** `internal/iface` already models the missing data: `Iface`, `IfaceItem`, `TypeExport`,
  `ConstructorScheme`, `AddExport(name, typ, purity)`, `AddType(name, arity)`, `AddTypeAlias`,
  `ToNormalizedJSON` (canonicalized), and a `Digest` field + `computeDigest` in `builder.go`.
- **V8.** `interface_hash` persisted in lockfile, registry metadata, coordinator SQLite rows
  (`from_interface_hash`/`to_interface_hash`), the Firestore converter, and pubsub payloads.
- **V9 (REFUTED).** Round-1 claimed `ailang publish` runs `runPrePublishSmoke` (compiles the
  package) immediately before the hash computation, so compiled interface data is reachable at
  publish time. That is **refuted**: the smoke gate is a **subprocess**.
  `RunSmokeInTempDir` (`internal/pkg/publish_validator.go:57`) spawns
  `exec.CommandContext(ctx, ailangBin, args...)` (`:88`) with the binary obtained via
  `os.Executable()` (`cmd/ailang/pkg_publish.go:156`). It returns a pass/fail `SmokeResult`;
  **no interface data crosses back** into the publish process. Control: `grep -c '^func '`
  `internal/pkg/publish_validator.go` = 6 (file read).
- **V10 (VERIFIED).** The in-process path to a real `Iface` per exported module exists and is
  reachable from `ailang publish`: `outputInterface` (`cmd/ailang/check.go:466`) builds
  `pipeline.Config{DryLink: true}` and calls `pipeline.Run(cfg, src)` (`:481-490`); the pipeline
  returns `result.Interface` (an `*iface.Iface`). `iface.NewBuilder`
  (`internal/iface/builder.go:302`) and `Builder.computeDigest` (`:689`) model the
  signature-bearing data. So publish CAN obtain a real `Iface` per exported module by calling
  `pipeline.Run` with `DryLink: true` once per module in `[exports]`. This is new work, not free —
  see D7.
- **V11 (VERIFIED).** `classifyChange` (`internal/messaging/pkg_events.go:162-170`) is:
  `if old.InterfaceHash != new.InterfaceHash { return "C" }`;
  `if old.ContentHash != new.ContentHash { return "A" }`; `return "A"`. Both remaining branches
  return `"A"`, so the `ContentHash` comparison is **dead code** — there is no additive class
  today. Control: `grep -c 'return "A"' internal/messaging/pkg_events.go` = 2 (both in
  `classifyChange`). Introducing a real class set is a rewrite of a 9-line function, not delicate
  surgery.

## Root Cause

The hash is computed from the **manifest** (`PackageManifest`), which only carries module paths and
effect ceilings. The compiler's exported-interface machinery (`internal/iface`) already produces a
canonical, deterministic, signature-bearing serialization (`ToNormalizedJSON`), but the registry
hash never consults it. The two systems model the same "public surface" with different fidelity,
and the registry side is the lossy one.

## Design Decisions (the hard part)

### D1 — Versioned hash value (algorithm marker inside the value)

Change the value format so old and new are **distinguishable, not merely different**:

- Legacy (today): `sha256:<64 hex>`.
- New (v2): `sha256:ifacev2:<64 hex>`.

The marker lives inside the value, so **no persisted field name changes** — `interface_hash` in the
lockfile, `InterfHash` in `PackageMetadata`, `from_interface_hash`/`to_interface_hash` in
coordinator rows and pubsub all keep their names and JSON tags. A consumer can tell which algorithm
produced a stored value by parsing the marker. Add a helper
`InterfaceHashVersion(hash string) int` returning `0` (legacy) or `2` (v2).

Rationale: a separate `interface_hash_v2` field would require touching every persistence point and
every reader; a value-embedded marker is self-describing and requires no schema migration.

### D2 — Migration policy: backfill + an explicit `unknown` class (no informational fallback)

The false-wave problem: after the change, every already-published package's stored hash is legacy
format, so a naive old-vs-new comparison sees a delta for **every** package. Round-1 proposed an
`interface-hash-migrated` informational class for the transition window; the quorum rejected it
(OBJ-1/OBJ-3) because it silently masks a real, potentially breaking delta — the same
no-silent-fallback violation the defect itself is. The honest answer is two mechanisms:

1. **Registry backfill (durable fix, reduces the `unknown` population).** The registry validator
   stores every published tarball in GCS at `packages/<vendor>/<name>/<version>/package.tar.gz`
   (`cmd/registry-validator/main.go:260-262`). A one-time migration pass downloads each historical
   tarball, extracts it, and re-runs the pipeline with the **current** compiler to recompute the
   **v2** hash **and** the signature set for that version. Backfill is **best-effort**: re-parsing
   an old tarball with the current compiler may fail (the old package may use syntax/features the
   current compiler rejects, or the current compiler may produce a different iface than the
   original compiler did). A version that fails to backfill is marked `unbackfillable`; its old
   side is `unknown` **permanently** (see D5). Backfill is in scope as M4; if it does not fit the
   sprint, it can be split into a follow-up with the named consequence that the transition window
   (and the `unknown` population) stays longer.
2. **Explicit `unknown` class (correctness fix, always required).** For any comparison where the
   old side's signature set is absent — a legacy hash that was not backfilled, or a local lockfile
   holding a legacy hash — `classifyChange` returns the explicit class **`U` (unknown /
   indeterminate)**, **not** `additive`, **not** `migrated/informational`, and **not** a silent
   degradation to today's hash-only A/C. `U` is visible to consumers (carried in `change_class`).
   A consumer agent seeing `U` must **not** auto-apply a patch; it surfaces the delta for explicit
   review. An honest "I cannot tell whether this is breaking" is strictly better than a confident
   wrong "patch", which is the entire defect being fixed. This satisfies CLAUDE.md §2: there is no
   fallback value that affects a decision — the decision is an explicit `unknown`.

### D3 — Determinism: canonical serialization, not `scheme.Type.String()`

The digest input must be stable across machines, ailang builds and map iteration order. Use the
**canonicalized** serialization path (`ToNormalizedJSON`-style: type variables renamed to `a,b,c,…`,
all arrays sorted, deterministic field order). Do **not** reuse the existing `computeDigest`
(`builder.go:688`) as-is: it serializes via `schemeToString` → `scheme.Type.String()` and
`ft.String()`/`ResultType.String()`, whose cross-build determinism is **UNVERIFIED** (no test locks
it; see Risks). The v2 hash folds, per exported module, a canonical JSON that excludes `TypeAliases`
(see D6), sorted by module path, then the package name, edition, and sorted max effects.

### D4 — Two-binary problem (publisher vs registry-validator) — dissolved by the subcommand

`cmd/registry-validator` is a separate binary. Round-1 framed this as a two-binary drift risk and
proposed a cross-binary consistency test. FIX-1 dissolves the problem instead: the canonical JSON is
defined **once**, by the hidden subcommand `ailang internal-dump-iface` (D7c). Both binaries become
**callers** of that one subcommand via `BuildModuleIface` (D7c), not re-implementers of the
serialization — there is no second definition to drift. The cross-binary consistency test is
retained as a regression guard that both binaries invoke the same subcommand and produce identical
hashes on the same fixture (AC9), but it now guards a single shared path rather than checking that
two independent implementations agree.

### D5 — Classification: a signature set + an explicit `unknown` class

A hash is a single value and cannot say *what* changed. Introduce a **signature set**: the sorted
list of canonical `module:name:signature` strings derived from each exported module's canonical
interface (function exports → `name:type`, type exports → `name(arity)`, constructors →
`name(fieldTypes)`). Persist it alongside the hash (new `interface_signatures` field on
`PackageMetadata`; carried on `PackageVersionInfo`). Then `classifyChange` (a rewrite of the
9-line function at `pkg_events.go:162-170`, whose `ContentHash` branch is dead code — V11) returns
one of four classes:

| old side | new side | class | meaning |
|---|---|---|---|
| signatures present | signatures present, new ⊇ old | `B` | additive (`minor`) |
| signatures present | signatures present, any old removed/changed | `C` | breaking (`major`) |
| signatures present | signatures present, equal | `A` | no interface change |
| signatures **absent** (legacy, un-backfilled) | any | `U` | **unknown / indeterminate** — cannot tell if breaking |

The `U` row is the honest answer to the transition window (OBJ-1/OBJ-3): when the old side's
signature set is absent, there is no trustworthy old side, so the classifier returns an explicit
`unknown` rather than a confident wrong `A`/`C`/`B`. `U` is carried in `change_class` on the
`upgrade-available` / `interface-change-notice` envelope. A consumer agent seeing `U` must **not**
auto-apply a patch; it surfaces the delta for explicit review (a human or a review-gated agent).
The coordinator's `autonomy_router.go:61` mapping (`PkgMsgInterfaceChange`: `"major"`→C else B)
must be extended so `U` routes to review, not to auto-apply.

This directly fixes the reporter's point: a consumer pinning by `interface_hash` can now see a
breaking removal/retype, because the signature-set diff classifies it `C` and the v2 hash delta
fires the notice. And it fixes the transition window honestly: a pre-migration old side classifies
`U`, not a silent `A`/`C`.

### D6 — TypeAlias policy: excluded, consistent with the existing digest-neutrality lock

`internal/iface/xmod_alias_digest_test.go` locks the module-level digest to **ignore** `TypeAliases`
(and `AliasParams`) so adding a transparent alias does not trigger dependent-package cascade
rebuilds. The v2 package hash must be consistent with that intent: fold `Exports`, `Constructors`
and `Types`, but **exclude** `TypeAliases`. The reporter's field case is a function export, which is
folded. If a future design decides aliases are real API, that is a separate, deliberate change with
its own migration — not folded into this one.

### D7 — Verified pipeline wiring at publish time (answers OBJ-2 and its consequences)

Round-1's V9 premise (smoke gate makes interface data reachable) is refuted (V9). The real,
verified path is V10: a real `Iface` per exported module is obtainable by running `pipeline.Run`
with `DryLink: true` once per module in `[exports]`, exactly as `outputInterface` does
(`cmd/ailang/check.go:466,481-490`). Round-2's quorum (3/3) accepted this direction but blocked on
**how** untrusted package source gets compiled: all three reviewers required the compilation to stay
in an isolated subprocess (FIX-1), with context/deadline/limit discipline (FIX-2), and corrected the
publisher/validator equivalence claim (FIX-3). The design below applies all three verbatim.

**(a) Publish now type-checks every exported module — a new, stricter publish-blocking gate.**
Computing the v2 hash requires building each exported module's iface, which type-checks it. Cost:
one `pipeline.Run` (DryLink, no eval) per exported module at publish time — bounded, most packages
have a handful of exported modules. If an exported module **fails to type-check**, the v2 hash
cannot be computed, so **publish is refused** with a clear error. This is a behaviour change worth
naming: today the publisher only compiles the package if a `_smoke.ail` exists (and only that file),
so a package with a broken exported module could still publish. After this change it cannot.
**This is a new, stricter gate that the registry validator does not yet enforce** (FIX-3): the
validator's compile-failure gate is `ailang check --package .` over the **whole package** as a
subprocess (`cmd/registry-validator/main.go:200,206-211`; M3), **not** a per-exported-module
`pipeline.Run(DryLink:true)`. These are different checks with different scope. The new gate is
applied to **both** binaries in M2, so the publisher and validator come to agree on the *stricter*
definition — but that agreement is a consequence of M2, not a pre-existing symmetry. (Round-1's
"closes an asymmetry / both agree" claim is refuted by M3.)

**(b) Module path → file resolution.** `pipeline.Run` is per-file. `[exports]` lists module paths;
each maps to a file as `filepath.Join(packageDir, modulePath) + ".ail"` (the loader's rule,
`internal/loader/loader.go:156,262,268,372`; `outputInterface` uses the same `+ ".ail"` mapping).
If a declared `[exports]` module path does not resolve to a file, that is a manifest inconsistency
(a module declared but missing) → **publish is refused** with a clear error naming the missing
file. The file must also declare `module <modulePath>` (MOD010), which the type-check enforces.

**(c) Isolated subprocess + context/deadline/limit discipline (FIX-1 + FIX-2).** The registry
validator is a live HTTP server processing untrusted uploads (M1), and it already isolates
compilation in a subprocess (`exec.Command("ailang", "check", ...)` at `validate.go:76,95,116`;
M2) — that is the house style. The design therefore does **not** compile untrusted source in-process.
Instead:

- **Hidden CLI subcommand.** Add `ailang internal-dump-iface <packageDir> <modulePath>` that runs
  `pipeline.Run` with `DryLink: true` (context propagated into `pipeline.Run`), and writes the
  canonical JSON to stdout. This subcommand is the **single definition** of the canonical JSON
  (FIX-1); it dissolves D4's two-binary problem because both binaries become callers of it.
- **Shared library wrapper.** `pkg.BuildModuleIface(ctx context.Context, packageDir, modulePath
  string) (*iface.Iface, error)` in `internal/pkg` invokes the subcommand via
  `exec.CommandContext(ctx, ...)` with a strict per-module timeout, parses the returned canonical
  JSON, and returns the `*iface.Iface`. Both the publisher and the registry-validator call
  `BuildModuleIface`; neither re-implements the serialization. A compiler panic in the subprocess
  cannot take down the registry process (FIX-1).
- **Deadlines and limits (FIX-2).** Define an **overall publish deadline**, a **per-module
  deadline**, and a **maximum exported-module count**; a timeout or limit failure **explicitly
  refuses publication** (no silent partial hash). Initial values, tunable in one named place — the
  `PublishLimits` struct in `internal/pkg/iface_subprocess.go`: overall publish deadline **60s**,
  per-module deadline **10s**, maximum exported-module count **64**. These are initial values, not
gospel; the named tuning point is the `PublishLimits` struct.
- **CLI refactor.** `outputInterface` (`cmd/ailang/check.go:466`) is refactored to call
  `BuildModuleIface` and `os.Exit(1)` on the returned error (CLI error handling stays in the CLI);
  the publish path calls it and returns the error up the stack as the publish-blocking gate.

## Verification Log

All measurements at HEAD `bd17d9643a527ad13261354563fed590c1baf0fa` unless noted. The installed
binary is STALE (built from `4bd58be`); source reads are at HEAD, and the one runtime probe
(`ailang iface`) is flagged accordingly.

| # | Claim | Command | Observed output |
|---|-------|---------|-----------------|
| 1 | HEAD commit | `git rev-parse HEAD` | `bd17d9643a527ad13261354563fed590c1baf0fa` |
| 2 | V1: no signature/type/arity/param data in hasher.go | `grep -cE "Signature\|Type\|Func\|Arity\|Param" internal/pkg/hasher.go` | `0` |
| 3 | V1 control (same file, known-positive) | `grep -c "Fprintf" internal/pkg/hasher.go` | `6` |
| 4 | V2: `ExportConfig.Modules` is `[]string` | `sed -n '135,150p' internal/pkg/manifest.go` | `Modules []string \`toml:"modules"\`` |
| 5 | V4: short-circuit on hash equality | `read internal/messaging/pkg_events.go` | `if old.InterfaceHash == new.InterfaceHash { return "", nil }` |
| 6 | V6: six non-test call sites | `grep -rn "InterfaceHash(" --include='*.go' cmd/ internal/ \| grep -v "_test.go"` | 6 sites: pkg_publish.go:108, resolver.go:179/233/300, pkg_msg.go:90/106, registry-validator/main.go:214 |
| 7 | V7: iface models the missing data | `read internal/iface/iface.go, json.go, xmod_alias_digest_test.go` | `IfaceItem{Name,Type *types.Scheme,Purity}`, `TypeExport{Name,Arity}`, `ConstructorScheme`, `ToNormalizedJSON`, `computeDigest` |
| 8 | V8: persisted in lockfile | `grep -n "InterfaceHash\|interface_hash" internal/pkg/lockfile.go` | `InterfaceHash string \`json:"interface_hash,omitempty"\`` (line 37) |
| 9 | V8: persisted in registry metadata | `grep -n "InterfaceHash\|interface_hash" internal/pkg/registry_types.go` | `InterfHash string \`json:"interface_hash"\`` (line 39) |
| 10 | V8: persisted in coordinator SQLite | `grep -n "interface_hash\|InterfaceHash" internal/coordinator/store_sqlite.go` | `ALTER TABLE tasks ADD COLUMN from_interface_hash/to_interface_hash` (198-199) |
| 11 | V8: persisted in pubsub | `grep -n "InterfaceHash\|interface_hash" internal/pubsub/publisher.go` | `FromInterfaceHash`/`ToInterfaceHash` (75-76) |
| 12 | V8: persisted in Firestore converter | `grep -rn "interface_hash\|InterfHash" --include='*.go' internal/ \| grep -i "firestore\|converter"` | `internal/storage/firestore/coordinator_convert.go:70-71,144-145` |
| 13 | V9: smoke runs before hash (subprocess, pass/fail only) | `sed -n '60,160p' cmd/ailang/pkg_publish.go` | `runPrePublishSmoke(cwd, manifest)` precedes `interfaceHash := pkg.InterfaceHash(manifest)` (line 108); the smoke is a subprocess (see rows 20-22) |
| 14 | iface output is canonicalized + deterministic | `ailang iface examples/effect_budget_demo.ail` twice, `diff` | `IDENTICAL across runs` (STALE binary; source path read at HEAD) |
| 15 | `classifyChange` never returns "B" (additive) | `read internal/messaging/pkg_events.go` | returns `"C"` if interface hash differs, else `"A"`; no `"B"` branch |
| 16 | `EmitInterfaceChangeNotice` sets no ChangeClass | `sed -n '56,80p' internal/messaging/pkg_events.go \| grep -n "ChangeClass"` | `NO ChangeClass in EmitInterfaceChangeNotice (confirmed)` |
| 17 | cascade maps interface-change-notice → B unless "major" | `read internal/coordinator/autonomy_router.go` | `PkgMsgInterfaceChange`: `"major"`→C else B |
| 18 | no test locks `computeDigest` cross-build determinism | `grep -rln "computeDigest\|Digest" --include='*_test.go' internal/iface/` | only xmod_alias, builtin_freeze, constructor tests (none lock `scheme.Type.String()` determinism) |
| 19 | binary staleness | `freshness_report` | `STALE: installed binary built from 4bd58be, HEAD is bd17d9643a52` |
| 20 | V9 refuted: smoke is a subprocess | `grep -n "func RunSmokeInTempDir\|exec.CommandContext" internal/pkg/publish_validator.go` | `RunSmokeInTempDir` at :57, `exec.CommandContext` at :88 |
| 21 | V9 refuted control (same file, known-positive) | `grep -c '^func ' internal/pkg/publish_validator.go` | `6` |
| 22 | V9 refuted: binary via os.Executable | `grep -n "os.Executable" cmd/ailang/pkg_publish.go` | `:156` |
| 23 | V10: outputInterface + DryLink | `grep -n "func outputInterface" cmd/ailang/check.go; sed -n '481,490p' cmd/ailang/check.go` | `:466`; `pipeline.Config{DryLink: true}` + `pipeline.Run(cfg, src)` |
| 24 | V10: NewBuilder + computeDigest | `grep -n "func NewBuilder\|func (b \*Builder) computeDigest" internal/iface/builder.go` | `:302`, `:689` |
| 25 | V11: classifyChange dead code | `sed -n '162,170p' internal/messaging/pkg_events.go` | both remaining branches return `"A"`; `ContentHash` branch dead |
| 26 | V11 control (same file, known-positive) | `grep -c 'return "A"' internal/messaging/pkg_events.go` | `2` (both in classifyChange) |
| 27 | registry stores tarballs in GCS (backfill feasible) | `grep -n "tarballGCSPath\|uploadToGCS" cmd/registry-validator/main.go` | `packages/<vendor>/<name>/<version>/package.tar.gz` (:260-262) |
| 28 | module path → file rule | `grep -n "filepath.Join(ml.basePath, path) + \".ail\"" internal/loader/loader.go` | `:372` (also :156,262,268) |
| 29 | outputInterface os.Exit on error (CLI-shaped) | `grep -n "os.Exit" cmd/ailang/check.go` | `:477,:493` (in outputInterface) |
| 30 | M1: validator is a live HTTP server | `grep -n "ListenAndServe" cmd/registry-validator/main.go; grep -c "HandleFunc\|ListenAndServe" cmd/registry-validator/main.go` | `main.go:70 log.Fatal(http.ListenAndServe(":"+port, nil))`; 9 non-test HandleFunc/ListenAndServe sites in main.go |
| 31 | M1: compile gate rejects with HTTP 400 | `sed -n '200,211p' cmd/registry-validator/main.go` | `runAilangCheck(tempDir)` at :200; `jsonError(w, http.StatusBadRequest, "Compilation failed:\n%s", compileErr)` at :207-209 |
| 32 | M2: validator isolates compilation in a subprocess | `grep -n "exec.Command" cmd/registry-validator/validate.go` | `:76 exec.Command("ailang","check","--package",".")`; `:95 exec.Command("ailang","check",f)`; `:116 exec.Command("ailang","verify","--json",f)` |
| 33 | M2 control (same file, known-positive) | `grep -c '^func ' cmd/registry-validator/validate.go` | `10` |
| 34 | M3: validator has no in-process pipeline.Run/DryLink | `grep -rn "pipeline.Run\|DryLink" cmd/registry-validator/` | `0` in every file (6 non-test files enumerated) |
| 35 | M3 control (scope is real) | `grep -c '^func ' cmd/registry-validator/main.go` | `7` |
| 36 | FIX-3: validator's compile gate is whole-package, not per-module | `sed -n '200,211p' cmd/registry-validator/main.go` | `runAilangCheck(tempDir)` → `ailang check --package .` over the whole package as a subprocess; NOT a per-exported-module `pipeline.Run(DryLink:true)` — **REFUTES** D7a's "both agree" claim |
| 37 | FIX-3: blast radius of the new publish-blocking gate (UNVERIFIED) | `curl -s "<registry>/api/v1/packages?limit=1000" \| jq '[.[] | select(.has_smoke == false)] | length'` | **UNVERIFIED** — the live registry is not queryable from this session. A human or later iteration should run the command above (exact endpoint to be confirmed against the registry API) to bound how many published versions lack `_smoke.ail` and would newly be blocked by D7a. Honest gap, not a guessed number. |

## Conflict Surface

`interface_hash` is package identity consumed by:

- **Resolver** (`internal/pkg/resolver.go:179,233,300`) — writes `InterfaceHash` into
  `ResolvedPackage` → the lockfile. If the definition changes, every resolved dependency's stored
  hash changes; consumers comparing lockfile-vs-registry must be format-aware (D2).
- **Lockfile** (`internal/pkg/lockfile.go:37`) — persisted per-dependency. A legacy hash in a
  local lockfile vs a v2 hash in the registry is the exact transition-window case D2 handles.
- **Registry validator** (`cmd/registry-validator/main.go:214`) — writes `InterfHash` into
  `metadata.json`. If the publisher and validator disagree, a package's identity differs by who
  computed it (D4).
- **Coordinator cascade** (`internal/coordinator/store_sqlite.go:198-199`, `store.go:71-72`,
  `cloud_dispatcher.go:45-46`, `daemon_tasks_*.go`, `stage_execution.go:230-231`,
  `autonomy_router.go:61`) — `from_interface_hash`/`to_interface_hash` drive `ClassifyChange`
  (B vs C) and the cascade template's `{{.FromInterfaceHash}}`/`{{.ToInterfaceHash}}`. A format
  change here changes cascade classification for every task. The new `U` class (D5) must be
  handled by `autonomy_router.go:61` (`PkgMsgInterfaceChange`: `"major"`→C else B) so `U` routes
  to review, not auto-apply.
- **Validator compile gate vs publish gate (DIVERGENCE, FIX-3).** The validator's compile-failure
gate is `ailang check --package .` over the **whole package** as a subprocess
(`cmd/registry-validator/main.go:200,206-211`); D7's new gate is a **per-exported-module**
`pipeline.Run(DryLink:true)` via the `internal-dump-iface` subcommand. These are **different checks
with different scope**. D7a no longer claims the two binaries "agree on what is publishable" at
HEAD; it claims the new gate is **stricter** and is applied to both binaries in M2. Until M2 lands,
a package could pass the validator's whole-package check yet fail the per-module gate (or vice
versa) — that divergence is the conflict surface M2 closes.
- **`ailang publish` gate** (`cmd/ailang/pkg_publish.go`) — after D7, publish type-checks every
  exported module and refuses on failure. A package that previously published with a broken
  exported module (no smoke test) is now refused. This is a deliberate behaviour change (D7a).
- **Pubsub** (`internal/pubsub/publisher.go:75-76`, `coordinator/pubsub_adapter.go:178-179`) —
  cascade envelopes carry the hashes; the adapter forwards them to task rows.
- **Messaging schema** (`internal/messaging/pkg_schema.go:74-75,171,181`) — `upgrade-available`
  and `interface-change-notice` **require** `from_interface_hash`/`to_interface_hash`; a format
  change must not break the validator's non-empty check.
- **`ailang pkg-info` / `pkg_msg`** (`cmd/ailang/pkg_info.go:94,225,279`, `pkg_msg.go:90,106`) —
  display and re-derive hashes; a format change alters what users see.

**What breaks if the value moves:** any consumer that treats `interface_hash` as a stable,
algorithm-agnostic identity will see a delta for every package on the first resolve after the
change. That is the false-wave harm. The mitigation is D2 (registry backfill + the explicit
`unknown` class), and it must be shipped in the **same release** as the hash change — a release
that changes the hash without the migration is the exact incomplete design this doc rejects.

## Acceptance Criteria

Each AC is a command that can FAIL. "Passes on unmodified HEAD" states whether the criterion is
already green at base (a criterion already red at base is broken, not a finding).

| # | Criterion (command) | Passes on unmodified HEAD? |
|---|---------------------|----------------------------|
| AC1 | `go test ./internal/pkg/ -run TestInterfaceHashV2_SensitiveToAddedExport` — adding an export to a listed module changes the v2 hash. | **No** — test does not exist at HEAD; the underlying behavior is the defect (V3a). |
| AC2 | `go test ./internal/pkg/ -run TestInterfaceHashV2_SensitiveToRemovedExport` — removing an export changes the v2 hash. | **No** — same defect (V3b). |
| AC3 | `go test ./internal/pkg/ -run TestInterfaceHashV2_SensitiveToRetype` — changing an export's type/arity changes the v2 hash. | **No** — same defect (V3c). |
| AC4 | `go test ./internal/pkg/ -run TestInterfaceHashV2_Deterministic` — same package dir hashed twice yields identical v2 hash (map-iteration + build stability). | **No** — test does not exist. |
| AC5 | `go test ./internal/pkg/ -run TestInterfaceHashV2_OrderIndependent` — reordering `[exports]` modules and `[effects]` max does not change the v2 hash. | **No** — test does not exist (legacy order-independence is covered by `TestInterfaceHash_OrderIndependent`, but not for v2). |
| AC6 | `go test ./internal/pkg/ -run TestInterfaceHashVersion` — `InterfaceHashVersion("sha256:ifacev2:…")==2` and `InterfaceHashVersion("sha256:…")==0`. | **No** — helper does not exist. |
| AC7 | `go test ./internal/messaging/ -run TestClassifyChange_AdditiveVsBreaking` — signature-set diff classifies pure addition as `B`/`minor` and removal/retype as `C`/`major`. | **No** — `classifyChange` never returns `B` today (V-log #15). |
| AC8 | `go test ./internal/messaging/ -run TestClassifyChange_UnknownOldSide` — comparing a legacy (un-backfilled) old side to a v2 new side returns `U` (unknown), not `A`/`C`/`B` and not a silent fallback. | **No** — the `U` class does not exist. |
| AC9 | `go test ./cmd/registry-validator/...` and `go test ./cmd/ailang/...` — cross-binary consistency test asserts both binaries invoke the same `internal-dump-iface` subcommand and produce identical v2 hashes on the same fixture (D4, FIX-1). | **No** — test does not exist. |
| AC10 | `go test ./internal/iface/ -run TestXModAlias` — the existing alias-digest-neutrality lock still passes (v2 hash excludes TypeAliases). | **Yes** — passes at HEAD; must remain green. |
| AC11 | `go test ./internal/pkg/ -run TestInterfaceHash_` — the legacy `InterfaceHash` tests still pass (backward-compat of the old function, kept for the migration path). | **Yes** — passes at HEAD. |
| AC12 | `go test ./internal/pkg/ -run TestPublishBlocksOnBrokenExportedModule` — publish refuses when an exported module fails to type-check (D7a). | **No** — the gate does not exist. |
| AC13 | `go test ./internal/pkg/ -run TestBuildModuleIface_ReturnsError` — `BuildModuleIface` returns an error (does not `os.Exit`) when the module file is missing, fails to type-check, or the subprocess fails/times out (D7c). | **No** — the function does not exist. |
| AC14 | `go test ./internal/pkg/ -run TestBuildModuleIface_ModulePathResolution` — a declared `[exports]` module path that does not resolve to a file returns an error (D7b). | **No** — the function does not exist. |
| AC15 | `go test ./internal/pkg/ -run TestBuildModuleIface_Cancellation` — cancelling the context kills the `internal-dump-iface` subprocess and `BuildModuleIface` returns `ctx.Err()` (FIX-2). | **No** — the function does not exist. |
| AC16 | `go test ./internal/pkg/ -run TestPublish_EnforcesPerModuleDeadline` — a module exceeding the per-module deadline (10s) refuses publication with a timeout error (FIX-2). | **No** — the deadline does not exist. |
| AC17 | `go test ./internal/pkg/ -run TestPublish_EnforcesExportLimit` — a package declaring more than the maximum exported-module count (64) refuses publication (FIX-2). | **No** — the limit does not exist. |
| AC18 | `go test ./internal/pkg/ -run TestBackfill_BoundedBatch` — backfill processes at most N versions (50) per invocation (FIX-2). | **No** — backfill does not exist. |
| AC19 | `go test ./internal/pkg/ -run TestBackfill_CheckpointedResume` — backfill resumes from its persisted checkpoint after an interruption, without reprocessing completed versions (FIX-2). | **No** — backfill does not exist. |

## Milestones

Each ≤ 1 day, independently committable and independently testable.

- **M1 — Canonical signature serialization + v2 hash + isolated subprocess builder (1 day).**
  Add to `internal/iface` a canonical, alias-excluded serialization (reuse `ToNormalizedJSON`
  machinery minus `TypeAliases`). Add the hidden CLI subcommand `ailang internal-dump-iface
  <packageDir> <modulePath>` that runs `pipeline.Run` with `DryLink: true` (context propagated into
  `pipeline.Run`) and writes canonical JSON to stdout (D7c, FIX-1). Add
  `pkg.BuildModuleIface(ctx context.Context, packageDir, modulePath string) (*iface.Iface, error)`
  in `internal/pkg` that invokes the subcommand via `exec.CommandContext` with a strict per-module
  timeout and parses the returned JSON (D7c, FIX-1/FIX-2). Add the `PublishLimits` struct
  (overall 60s / per-module 10s / max 64 modules) in `internal/pkg/iface_subprocess.go` and
  `pkg.InterfaceHashV2(dir, manifest)` that builds each exported module's iface via
  `BuildModuleIface` and folds canonical JSON + name/edition/effects with the `ifacev2` marker. Add
  `InterfaceHashVersion`. Refactor the CLI's `outputInterface` to call `BuildModuleIface` and
  `os.Exit(1)` on its error. Tests: AC1-AC6, AC10, AC11, AC13-AC17.
- **M2 — Wire both binaries + cross-binary consistency (1 day).** Point `cmd/ailang/pkg_publish.go`
  and `cmd/registry-validator/main.go` at `InterfaceHashV2` via `BuildModuleIface`, applying the
  new per-module gate to **both** binaries (FIX-3). Add the cross-binary consistency test (AC9).
  Keep the legacy `InterfaceHash` for the migration path.
- **M3 — Signature set + classification with `unknown` (1 day).** Add `interface_signatures` to
  `PackageMetadata` and `Signatures` to `PackageVersionInfo`. Rewrite `classifyChange` (the 9-line
  function at `pkg_events.go:162-170`) to diff signature sets and return `A`/`B`/`C`/`U` (D5);
  set `ChangeClass`/`Breaking` on `EmitInterfaceChangeNotice` and `EmitUpgradeAvailable`. Extend the
  coordinator's `autonomy_router.go:61` mapping so `U` routes to review, not auto-apply. Tests:
  AC7, AC8.
- **M4 — Registry backfill + transition handling (1 day).** Add a validator migration
  command/endpoint that downloads each historical tarball from GCS
  (`packages/<vendor>/<name>/<version>/package.tar.gz`), extracts it, and recomputes the v2 hash +
  signature set with the current compiler. Backfill is a **cursor-based, resumable job** (FIX-2):
  it processes at most **N=50 versions per invocation** (initial value, tunable in the
  `PublishLimits` struct), with a **per-download deadline (30s)** and a **per-compile deadline
  (10s)**, **bounded retries (3)**, a **persisted checkpoint** (so an interrupted run resumes
  without reprocessing completed versions), and explicit **`failed` / `unbackfillable`** states.
  Mark versions that fail to re-parse `unbackfillable` (their old side stays `U` permanently). Wire
  the resolver/lockfile path so a legacy stored hash with no signature set classifies `U`, not a
  false wave. Tests: AC8 end-to-end, AC18, AC19, plus a migration dry-run on a fixture registry.

## Mutation Table

For each guard, the concrete mutation that must make a named test go RED. Mutations are chosen to
still **compile** (neuter a condition with `if false && cond`) so "the mutant does not build" cannot
masquerade as "the guard fired".

| Guard | Mutation (still compiles) | Test that must go RED |
|-------|---------------------------|-----------------------|
| v2 hash folds export signatures | In `InterfaceHashV2`, change the fold to `if false && len(moduleJSON) > 0 { … }` so module JSON is never written into the hash. | `TestInterfaceHashV2_SensitiveToAddedExport` (AC1) |
| v2 hash folds removal | Same mutation as above (removal is the same fold). | `TestInterfaceHashV2_SensitiveToRemovedExport` (AC2) |
| v2 hash folds type/arity | Same fold mutation. | `TestInterfaceHashV2_SensitiveToRetype` (AC3) |
| v2 hash is deterministic | Replace the canonical serialization with `scheme.Type.String()` (non-canonicalized) in the fold. | `TestInterfaceHashV2_Deterministic` (AC4) |
| version marker distinguishes old/new | In `InterfaceHashV2`, drop the `ifacev2` marker (`if false && …` around the marker write). | `TestInterfaceHashVersion` (AC6) |
| additive vs breaking classification | In `classifyChange`, neuter the signature-set diff: `if false && newSuperset(old) { … }` so every delta falls to the breaking branch. | `TestClassifyChange_AdditiveVsBreaking` (AC7) |
| unknown old side classifies `U`, not silent A/C | In `classifyChange`, neuter the `old.Signatures == nil` branch: `if false && old.Signatures == nil { … }` so a legacy old side falls through to the hash-only A/C path. | `TestClassifyChange_UnknownOldSide` (AC8) |
| two binaries agree | In `cmd/registry-validator/main.go`, bypass `BuildModuleIface` and call the legacy `InterfaceHash` instead of `InterfaceHashV2`. | cross-binary consistency test (AC9) |
| TypeAlias excluded from v2 hash | In the canonical serialization, re-enable the `Alias` field (`if false && …` guard removed). | `TestXModAlias_DigestIgnoresTypeAliases` (AC10) |
| publish blocks on broken exported module | In `InterfaceHashV2`, swallow the `BuildModuleIface` error (`if false && err != nil { … }`) so a broken module still produces a hash. | `TestPublishBlocksOnBrokenExportedModule` (AC12) |
| library-shaped builder returns error, not os.Exit | In `BuildModuleIface`, replace the error return with `os.Exit(1)` on the missing-file path. | `TestBuildModuleIface_ReturnsError` (AC13) |
| module path resolution | In `BuildModuleIface`, drop the `+ ".ail"` suffix (`if false && …`) so a declared module path resolves to the wrong file. | `TestBuildModuleIface_ModulePathResolution` (AC14) |
| cancellation kills the subprocess | In `BuildModuleIface`, pass a background context instead of the caller's `ctx` to `exec.CommandContext` (`if false && …`), so cancelling the caller does not kill the subprocess. | `TestBuildModuleIface_Cancellation` (AC15) |
| per-module deadline enforced | In `BuildModuleIface`, drop the per-module timeout (`if false && …` around the `context.WithTimeout`), so a hung subprocess never times out. | `TestPublish_EnforcesPerModuleDeadline` (AC16) |
| export-limit enforced | In `InterfaceHashV2`, remove the max-exported-module-count check (`if false && len(exports) > max { … }`), so an oversized package still publishes. | `TestPublish_EnforcesExportLimit` (AC17) |
| backfill batch bounded | In the backfill job, drop the N-per-invocation cap (`if false && processed >= N { … }`), so one invocation processes the whole registry. | `TestBackfill_BoundedBatch` (AC18) |
| backfill resumes from checkpoint | In the backfill job, skip persisting/reading the checkpoint (`if false && …`), so an interrupted run restarts from the beginning. | `TestBackfill_CheckpointedResume` (AC19) |

## Risks / Unverified

- **Pre-existing defect, OUT OF SCOPE (M4).** The registry validator's three compile call sites use
  `exec.Command`, **not** `exec.CommandContext` (`cmd/registry-validator/validate.go:76,95,116`):
  the server compiles untrusted uploads with **no timeout and no cancellation today, at HEAD**,
  independent of anything this document proposes. This is a pre-existing defect the controller is
  filing as its own queue row; it is **explicitly out of scope here** and this document does not grow
to fix it. It is named for honesty only. (The new `BuildModuleIface` path in D7c uses
`exec.CommandContext` with a strict timeout, so the *new* gate is not affected — but the existing
`runAilangCheck`/`runAilangVerify` paths remain un-timed until the separate queue row lands.)
- **`computeDigest` determinism (UNVERIFIED).** The existing `builder.go:688` digest serializes via
  `scheme.Type.String()` / `ft.String()` / `ResultType.String()`. No test locks cross-build
  determinism of those `String()` methods (V-log #18). The design therefore does **not** reuse
  `computeDigest`; it uses the canonicalized path (D3). If a reviewer can prove `scheme.Type.String()`
  is already canonical, M1 can reuse `computeDigest` — but that must be demonstrated, not assumed.
- **Pipeline wiring at publish time (RESOLVED, now D7).** Round-1 flagged this UNVERIFIED; the
  quorum's OBJ-2 was correct. V9 refutes the smoke-gate premise; V10 verifies the path
  (`pipeline.Run` with `DryLink: true`, per `outputInterface` at `check.go:466,481-490`). Round-2's
  quorum (3/3) then blocked on **how** untrusted source is compiled, and all three fixes are applied
  verbatim in D7: an isolated subprocess (`internal-dump-iface`) with context/deadline/limit
  discipline (FIX-1/FIX-2) and a corrected, weaker publish-gate claim (FIX-3). The consequences — a
  new, stricter publish-blocking gate (D7a), module-path→file resolution (D7b), and an isolated
  subprocess builder (D7c) — are designed in D7 and gated by AC12-AC17.
- **Backfill cost and failure (UNVERIFIED).** Recomputing v2 hashes + signature sets for all
  historical versions requires the validator to re-run the pipeline per version from the stored
  GCS tarball. M4 is therefore a **cursor-based, resumable job** (FIX-2): at most N=50 versions per
  invocation, per-download/per-compile deadlines, bounded retries, a persisted checkpoint, and
  explicit `failed`/`unbackfillable` states (AC18/AC19). I did not measure the registry's version
  count or per-version cost, and the blast radius of the new publish-blocking gate is recorded as
  **UNVERIFIED** (V-log #37) with the exact command a human or later iteration should run. Re-parsing
  an old tarball with the current compiler may fail; such versions are marked `unbackfillable` and
  their old side stays `U` permanently (D2/D5). Flag for the sprint planner.
- **`classifyChange` class set.** Today `classifyChange` returns only `"A"`/`"C"` (V-log #15) and
  its `ContentHash` branch is dead code (V11). M3 rewrites it to `A`/`B`/`C`/`U` (D5). The existing
  `EmitUpgradeAvailable` path must use the same classification, and the coordinator's
  `autonomy_router.go:61` mapping must route `U` to review, not auto-apply.

## Scope

4 days as decomposed. Answering OBJ-1/OBJ-3 honestly grew the design: the `unknown` class (D5) is
the core correctness fix and is in scope (M3); the registry backfill (D2/M4) reduces the `unknown`
population but is **best-effort** — versions that cannot be re-parsed with the current compiler are
`unbackfillable` and their old side stays `U` permanently. If the honest scope is larger (e.g. the
registry backfill turns out to need batching, or the pipeline wiring for multi-module iface build
is non-trivial), the decomposition above isolates the risk in M1/M4 so the sprint planner can
re-plan those milestones without re-opening the hash-format or classification decision. The `U`
class and the publish-blocking gate (D7a) are non-negotiable parts of the fix; backfill is the one
piece that can be split into a follow-up, with the named consequence that the transition window
(and the `unknown` population) stays longer. This document is a design, not an implementation: no
Go source is changed by this document.

---

## Quorum verification log (controller-recorded, iteration 310)

The quorum artifacts themselves live at `.ailang/state/mission-quorum/`, which `.gitignore:82`
excludes (`git check-ignore` rc=0; known-ignored control `eval_results/x` rc=0 &mdash; **note the `/x`**: the bare directory `eval_results/` returns **rc=1**, because `eval_results/.gitignore` negates subpaths (`!eval_results/baselines/`, `!eval_results/performance_tables/`). The iteration-310 evaluator caught this doc quoting the control without its path; the command actually run was the file form. The substantive claim is independently true either way; **0** quorum
artifacts are tracked on `origin/dev` against a control of 1473 tracked files under `design_docs/`).
They are therefore recorded here, in the tracked artifact, rather than force-added.

| Round | Artifact | Present / total | Synthesis |
|---|---|---|---|
| 1 | `m-registry-interface-hash-blind-to-signatures-2026-08-31T15-33-19Z.json` | 3/3 external, 0 absent | **BLOCKED** — 3 reject ($0.0735) |
| 2 | `m-registry-interface-hash-blind-to-signatures-2026-08-31T15-43-38Z.json` | 3/3 external, 0 absent | **BLOCKED** — 3 reject ($0.0929) |

Both rounds had every external reviewer PRESENT, so neither verdict is an N−1 degrade.

**Round 1 objections landed on three different surfaces** (D2 migration semantics; the unverified
pipeline-wiring premise; D5's unpopulated signature sets). **Round 2 objections localised onto ONE
surface** — how untrusted package source gets compiled — which is the decomposition signal, not a
maturity signal. No reviewer flipped to pass, so the doc was not split; instead:

**Disposition: NARROW-REFINEMENT CARVE-OUT.** After the one protocol-mandated re-quorum, all three
remaining objections (a) carried a concrete reviewer-authored `proposed_fix` and (b) disputed only
isolation, bounding and an unverified equivalence claim — **none disputed the design direction**. A
bounded third revision applied the reviewers' fixes verbatim. This satisfies the objections; it is
not a force-pass.

| Objection | Reviewer | Controller action |
|---|---|---|
| D7c runs the compiler in-process inside a live HTTP registry server (crash/DoS) | `gemini-3-1-pro` | **MEASURED, CONFIRMED.** `main.go:70` `ListenAndServe`; the existing gate already isolates via `exec.Command` at `validate.go:76/95/116` (control `^func ` = 10). Fix applied verbatim: one hidden `internal-dump-iface` subcommand, invoked as a subprocess by both binaries — which also dissolves D4. |
| Unbounded `pipeline.Run` calls and an unbounded backfill (bounded-waits axiom) | `gpt5-6-sol` | Fix applied verbatim: `ctx context.Context` propagation, per-module and overall deadlines, an exported-module cap, and a cursor-based resumable backfill with checkpoints. |
| The publisher/validator equivalence claim is asserted, not verified | `oc-glm-5-2` | **MEASURED, REFUTED — the claim was false.** `pipeline.Run` = 0 and `DryLink` = 0 across all 11 files of `cmd/registry-validator/` (control `^func ` in `main.go` = 7); the validator runs `ailang check --package .` over the whole package. The doc now makes the weaker, true claim. |

**Controller premise refuted in round 1, recorded rather than smoothed:** the controller's own
round-1 fact V9 — *"`runPrePublishSmoke` compiles the package, so interface data is plausibly
reachable at publish time"* — is **FALSE**. `RunSmokeInTempDir` (`internal/pkg/publish_validator.go:57`)
runs `exec.CommandContext(ctx, ailangBin, args...)` at `:88` against the binary from
`os.Executable()` (`cmd/ailang/pkg_publish.go:156`): a subprocess returning pass/fail, with no
interface data crossing back. The designer labelled it UNVERIFIED rather than asserting it, and
`gemini-3-1-pro` blocked on it. Both behaved correctly; the controller's hint was the defect.

**Pre-existing defect surfaced by this review, deliberately NOT absorbed:** the registry validator
compiles untrusted uploads with `exec.Command` (no context, no timeout, no cancellation) at
`validate.go:76`, `:95` and `:116`. That is true at HEAD independent of this design, so it is a
queue row, not a revision to this document.
