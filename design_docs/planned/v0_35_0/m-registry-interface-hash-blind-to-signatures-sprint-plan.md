# Sprint Plan — M-REGISTRY-INTERFACE-HASH-BLIND-TO-SIGNATURES

**Design doc:** [`m-registry-interface-hash-blind-to-signatures.md`](m-registry-interface-hash-blind-to-signatures.md)
**Planner:** sprint-planner, mission iteration 310
**Planned at:** worktree `/Users/voightkampff/.ailang-driver-pin/.planner-wt-iter310`, detached HEAD `66add3b4865116ba93a669ac1e2c690fe0a9860c`
**Verdict:** **PROCEED WITH RE-SCOPE.** The design direction is sound and quorum-cleared. The
*decomposition and the acceptance table are not shippable as written* — six defects measured below,
three of them blocking (D1 makes 16 of 19 gates un-failable; D2 makes M1's central API
unimplementable; D3 makes the classifier stall every cascade on the day it lands).

**Honest estimate: 9 days (not the doc's 4).** Split into a 4-day shippable Sprint 1 (all
reversible, zero registry writes) and a 5-day Sprint 2 (everything irreversible).

---

## 0. Measured base state

Every number below was produced in this worktree at `66add3b48`, unmodified, with exit codes
captured **without a pipe** (`cmd > /tmp/out 2>&1; rc=$?`) because `${PIPESTATUS[0]}` is silently
empty in zsh.

| Probe | Command | Observed |
|---|---|---|
| Worktree clean | `git status --porcelain \| wc -l` | `0` |
| `InterfaceHash` is signature-blind | `grep -cE "Signature\|Type\|Func\|Arity\|Param" internal/pkg/hasher.go` | `0` — **control, same file:** `grep -c Fprintf internal/pkg/hasher.go` = `6` |
| `classifyChange` dead branch | `sed -n '162,170p' internal/messaging/pkg_events.go` | both non-`C` branches `return "A"` — confirmed |
| 6 non-test call sites | `grep -rn "InterfaceHash(" --include='*.go' cmd/ internal/ \| grep -v _test.go` | 7 lines = 6 call sites + the `func` definition at `hasher.go:73`. Controller's "six call sites" confirmed. |
| Validator has no in-process pipeline | `grep -rn "pipeline.Run\|DryLink" cmd/registry-validator/ \| wc -l` | `0` — **control:** `grep -c '^func ' cmd/registry-validator/main.go` = `7` |
| `.ailang/` is gitignored | `git check-ignore -v .ailang/state/sprints/x.json` | `rc=0`, `.gitignore:82` |
| **`go build ./...` is RED at base** | `go build ./... > /tmp/o 2>&1; rc=$?` | **`rc=1`** — `cmd/wasm runtime.main_main·f: function main is undeclared`. **Never use as a gate.** |
| Narrow build is GREEN at base | `go build ./internal/pkg/... ./internal/messaging/... ./internal/iface/... ./cmd/ailang ./cmd/registry-validator` | `rc=0` — this is the build assertion the plan uses |
| `go vet` (same scope) GREEN | `go vet <same 5 targets>` | `rc=0` |
| `internal/pkg` suite GREEN | `go test ./internal/pkg/` | `rc=0`, 1.255s |
| `internal/messaging` suite GREEN | `go test ./internal/messaging/` | `rc=0`, 5.222s |
| `internal/iface` alias lock GREEN | `go test ./internal/iface/ -run '^TestXModAlias' -v` | `rc=0`, `--- PASS: TestXModAlias` × **2** |
| `cmd/registry-validator` suite GREEN | `go test ./cmd/registry-validator/...` | `rc=0`, 3.117s |
| **`cmd/ailang` suite GREEN but SLOW** | `go test ./cmd/ailang/...` | `rc=0`, **311.052s** |
| Architecture boundaries GREEN | `bash scripts/check_boundaries.sh` | `rc=0`, "no architecture boundary violations" |
| `internal/pkg` → core already legal | `grep -rho "internal/[a-z_]*" internal/pkg/*.go \| sort -u` | already imports `internal/builtins` (a CORE pkg) and boundaries pass ⇒ `internal/pkg` → `internal/iface` is legal |
| `internal-dump-iface` name free | `grep -rn "internal-dump-iface" .` | only the design doc — **control:** `grep -c iface cmd/ailang/main.go` = `7` |

---

## 1. What I refuted (read this before executing)

### D1 — BLOCKING. 16 of the doc's 19 acceptance commands are GREEN at base and **can never go red**.

The doc's AC table says AC1–AC9 and AC12–AC19 "**Pass on unmodified HEAD? No** — test does not
exist." That is false. `go test -run <regex>` with a regex that matches nothing **exits 0**:

```
$ go test ./internal/pkg/ -run TestInterfaceHashV2_SensitiveToAddedExport > /tmp/ac1.txt 2>&1; rc=$?
$ echo $rc; cat /tmp/ac1.txt
0
ok  	github.com/sunholo-data/ailang/internal/pkg	0.530s [no tests to run]
```

Control, same package, a test that *does* exist:

```
$ go test ./internal/pkg/ -run TestInterfaceHash_ > /tmp/ac11.txt 2>&1; echo $?
0
ok  	github.com/sunholo-data/ailang/internal/pkg	0.241s
```

Both are `rc=0`. The gate is not measuring anything. An executor could ship M1 with zero tests
written and every AC would report green.

**This is compounded by a second silent-green path.** The subprocess tests (AC13–AC17) need an
`ailang` binary. The house helper is `internal/testutil.FindAilangBinary(t)`
(`internal/testutil/ailangbin.go:49`), which calls **`t.Skipf`** when every candidate binary is
older than the newest Go source (`:74-79`). A skipped test also exits 0. So AC13–AC17 would be
green-by-skip on any machine with a stale `bin/ailang` — which is the normal state of this repo.

**Fix, used by every acceptance command in this plan:** assert on the `--- PASS:` line, not on the
exit code. Measured both directions in one command:

```
# nonexistent test -> RED (correct)
go test ./internal/pkg/ -run '^TestInterfaceHashV2_SensitiveToAddedExport$' -v > /tmp/x.txt 2>&1; rc=$?
grep -q -- '--- PASS: TestInterfaceHashV2_SensitiveToAddedExport' /tmp/x.txt && [ $rc -eq 0 ] \
  && echo GREEN || echo RED
# observed: RED (go rc=0)

# existing test -> GREEN (control, same form, same session)
go test ./internal/messaging/ -run '^TestClassifyChange$' -v > /tmp/y.txt 2>&1; rc=$?
grep -q -- '--- PASS: TestClassifyChange' /tmp/y.txt && [ $rc -eq 0 ] && echo GREEN || echo RED
# observed: GREEN (go rc=0)
```

This form also fails a `t.Skipf`, closing the stale-binary hole in the same stroke.

### D2 — BLOCKING. `BuildModuleIface(...) (*iface.Iface, error)` is not implementable as specified.

D7c says `BuildModuleIface` "parses the returned canonical JSON, and returns the `*iface.Iface`".
It cannot. The canonical JSON renders each export's type as a **string**
(`FuncJSON.Type string`, `internal/iface/json.go:33`), while `iface.IfaceItem.Type` is a
`*types.Scheme` (`internal/iface/iface.go:29`). Reconstructing the `Iface` needs a
string→`types.Scheme` parser. There is none:

```
$ grep -rn "func.*FromNormalizedJSON\|func.*ParseIface\|func.*UnmarshalIface" internal/iface/
(no output)
$ grep -rn "func (i \*Iface) ToNormalizedJSON" internal/iface/json.go     # control: the one-way direction exists
internal/iface/json.go:46:func (i *Iface) ToNormalizedJSON() ([]byte, error) {
$ grep -rn "func ParseType\|func ParseScheme" internal/types/ internal/parser/
(no output)
$ grep -rc "func " internal/types/types.go                                # control: the file is non-empty
41
```

**Fix (adopted by this plan):** the subprocess wrapper returns the *canonical JSON view*, not the
compiler's `Iface`:

```go
func BuildModuleIface(ctx context.Context, packageDir, modulePath string) (*iface.InterfaceJSON, error)
```

Nothing downstream needs a `*types.Scheme` — the v2 hash and the signature set are both pure
functions of the canonical JSON. This is strictly simpler than the design and removes a whole
deserializer from scope.

### D3 — BLOCKING. D5's classification table stalls **every** cascade the day M3 lands.

D5's table reads: *old side signatures absent → `U`, for any new side*. Before the backfill (M4)
and before either binary is wired (M2), **no** package has a signature set. So every
`old`/`new` pair is legacy-vs-legacy, and every comparison returns `U`. `U` routes to review and
forbids auto-apply (D5, and the `autonomy_router.go:61` change). Net effect: the coordinator stops
auto-applying anything, for every package, until backfill completes. That is the mirror image of
the harm the doc set out to avoid — it replaces a confident-wrong `patch` with a total review stall.

**Fix (adopted):** the classifier is a 2×2 on *which sides carry signatures*, not a 1-D test on the
old side.

| old side | new side | class | rationale |
|---|---|---|---|
| legacy (no sigs) | legacy (no sigs) | **today's hash-only `A`/`C`** | Pre-migration steady state. Nothing about the world changed; behaving differently would be a regression, not honesty. Not a fallback — it is the unchanged legacy path for a wholly-legacy comparison. |
| legacy (no sigs) | v2 (sigs) | **`U`** | The genuine transition window. No trustworthy old side ⇒ explicit unknown. This is the D5 case that matters. |
| v2 (sigs) | v2 (sigs) | `A` / `B` / `C` | The signature-set diff, as designed. |
| v2 (sigs) | legacy (no sigs) | **`U`** | Regression in producer fidelity; also indeterminate. |

This preserves D5's non-negotiable (`U` is explicit, never a silent `A`) while bounding the blast
radius to packages that have actually crossed the migration boundary.

### D4 — Non-blocking, but M1's `outputInterface` refactor would regress `ailang iface --compact`.

D6 requires the v2 hash to **exclude** type aliases. But `ToNormalizedJSON` deliberately **emits**
them (`internal/iface/json.go:29,91-92` — the `alias` field), and `ailang iface --compact` reads
that field back (`cmd/ailang/check.go:533`); it exists because of M-IFACE-RECORD-FIELDS, so an
agent can build/destructure a record type from the compact interface instead of cat-ing the source.
If `internal-dump-iface` emits alias-excluded JSON and `outputInterface` is refactored onto it (as
M1 instructs), that agent-facing capability is silently deleted.

**Fix (adopted):** `internal-dump-iface` emits the **full** normalized JSON, byte-identical to
`ailang iface`. Alias exclusion happens one layer later, in a pure deterministic projection
`iface.HashProjection(InterfaceJSON) []byte` that drops `types[].alias`. FIX-1's "single definition
of the canonical JSON" is preserved. And `outputInterface` is **left alone** — see D5 below.

### D5 — Non-blocking. Refactoring `outputInterface` onto `BuildModuleIface` makes `ailang iface` fork a copy of itself.

`BuildModuleIface` is a subprocess wrapper. If `outputInterface` calls it, then every
`ailang iface foo.ail` spawns an `ailang internal-dump-iface` child — a per-invocation process
fork on a user-facing, hot, agent-facing command, for zero safety benefit (the local user's own
source is not untrusted; the isolation requirement, FIX-1, is about the **registry server**).
It is also a recursion hazard if the subcommand is ever wired through the same helper.

**Fix (adopted):** two functions, one definition.
`iface.BuildCanonicalJSON(ctx, packageDir, modulePath) ([]byte, error)` is the **in-process**
single definition. `internal-dump-iface` is a thin `main`-side wrapper that prints it.
`pkg.BuildModuleIface` is the **subprocess** wrapper used by publish and the validator.
`outputInterface` calls `BuildCanonicalJSON` directly. FIX-1 is satisfied — there is still exactly
one serialization — without the fork.

### D6 — Non-blocking, but AC9 cannot catch the failure mode it is named for.

The publisher resolves its helper binary via `os.Executable()` (`cmd/ailang/pkg_publish.go:156`);
the registry validator resolves via bare `exec.Command("ailang", …)` from `PATH`
(`cmd/registry-validator/validate.go:76,95,116`). Those are **different binaries at different
versions** in production: the validator runs a deployed container image, the publisher runs
whatever the user installed. FIX-1 removes *code* duplication; it does not remove *version skew*.
Two `ailang` versions can legitimately emit different canonical JSON for the same source (any type
renderer change does it), and then the publisher and validator disagree about package identity —
exactly the D4 harm the doc believes it dissolved.

AC9 as written (`go test ./cmd/registry-validator/... && go test ./cmd/ailang/...`) builds both
binaries from one tree at one commit, so it is structurally incapable of observing skew.

**Fix (adopted, M7):** the canonical JSON carries the producing binary's version, and the validator
**compares** the publisher-submitted v2 hash against its own recomputation and **rejects the upload
with HTTP 400 on mismatch** (loud failure, CLAUDE.md §2) instead of silently overwriting it as
`main.go:214` does today. `TestValidatorRejectsHashMismatch` is the guard.

### D7 — Non-blocking scope note. `go test ./cmd/ailang/...` is a **311-second** gate.

Measured: `rc=0`, `311.052s`. AC9 names it unscoped. Every acceptance command in this plan scopes
`cmd/ailang` with `-run '^Test…$'`.

### D8 — Non-blocking. The doc's AC10 mutation is invalid.

The mutation table says: mutate *the new canonical serialization* to re-enable `Alias`, and
`TestXModAlias_DigestIgnoresTypeAliases` must go RED. It will not. That test calls
`b.computeDigest(...)` directly (`internal/iface/xmod_alias_digest_test.go:32,36`) — the **builder's**
digest, which this sprint does not touch. It is untouched by any change to the new hash projection.
The alias-exclusion guard must be a **new** test on the v2 hash (`TestInterfaceHashV2_IgnoresTypeAliases`,
M4). `TestXModAlias*` stays as a regression lock only.

### Two design claims I checked and **confirmed** (no refutation)

- `pipeline.RunWithContext(ctx, cfg, src)` exists (`internal/pipeline/pipeline.go:154`), so FIX-2's
  "context propagated into `pipeline.Run`" is implementable as written.
- `pipeline.Config.PackageDir` exists (`internal/pipeline/pipeline.go:94-96`) and overrides where
  `ailang.toml`/`ailang.lock` are found, so D7b's module-path→file resolution has a real home;
  running the child with `cmd.Dir = packageDir` (the pattern `runAilangCheck` already uses) covers
  the loader's CWD-relative basePath.

---

## 2. Sprint 1 — shippable, fully reversible, zero registry writes (4 days)

Ordering rule applied: **riskiest reversible thing first, irreversible last.** Everything in
Sprint 1 is additive and *dark* — no binary emits a v2 hash, no registry record changes, no
published package's identity moves. Sprint 1 can be reverted with a single `git revert` at any
milestone boundary. Every milestone leaves the tree green and is independently committable.

Shared build/vet assertion for **every** milestone (measured GREEN at base, `rc=0`):

```
go build ./internal/pkg/... ./internal/messaging/... ./internal/iface/... ./cmd/ailang ./cmd/registry-validator > /tmp/b.txt 2>&1; rc=$?; echo "build rc=$rc"
go vet   ./internal/pkg/... ./internal/messaging/... ./internal/iface/... ./cmd/ailang ./cmd/registry-validator > /tmp/v.txt 2>&1; rc=$?; echo "vet rc=$rc"
bash scripts/check_boundaries.sh > /tmp/cb.txt 2>&1; echo "boundaries rc=$?"
```

Do **not** use `go build ./...` (measured `rc=1` at base — `cmd/wasm`).

---

### M1 — Alias-excluded hash projection over the canonical interface JSON (0.5 day, ~120 LOC + ~90 test LOC)

**Ships:** `internal/iface/hash_projection.go` — `HashProjection(*InterfaceJSON) ([]byte, error)`,
a pure, deterministic, alias-dropping projection of the normalized JSON (D4). Sorted arrays,
canonical type variables (inherited from `ToNormalizedJSON`), `types[].alias` omitted,
`types[].ctors` / `funcs[]` retained. Plus `SignatureSet(*InterfaceJSON) []string` producing the
sorted `module:name:signature` strings D5 needs.

**Why first:** it is a pure function over a struct. No compiler, no subprocess, no fixtures, no
binary. Fastest possible feedback on the one decision the whole sprint rests on (what exactly gets
hashed), and it is trivially revertible.

**Acceptance (measured base state in brackets):**

| # | Command | Base |
|---|---|---|
| M1-A1 | `go test ./internal/iface/ -run '^TestHashProjection_ExcludesAlias$' -v > /tmp/o 2>&1; rc=$?; grep -q -- '--- PASS: TestHashProjection_ExcludesAlias' /tmp/o && [ $rc -eq 0 ]` | **RED** (measured: `-v` output has no `--- PASS` line; `go rc=0` with `[no tests to run]`) |
| M1-A2 | same form, `^TestHashProjection_Deterministic$` (100 iterations over a map-bearing input) | **RED** (same measurement) |
| M1-A3 | same form, `^TestSignatureSet_SortedAndStable$` | **RED** (same measurement) |
| M1-A4 | `go test ./internal/iface/ -run '^TestXModAlias' -v > /tmp/o 2>&1; rc=$?; [ "$(grep -c -- '--- PASS: TestXModAlias' /tmp/o)" -eq 2 ] && [ $rc -eq 0 ]` | **GREEN at base** (measured: 2 PASS lines, `rc=0`) — regression lock, must stay green |
| M1-A5 | shared build/vet/boundaries block | **GREEN at base** (`0`/`0`/`0`) |

**Mutations (must still compile — anchored to the whole M1 diff, not only the alias rule):**

| Hunk shipped | Mutation | Test that must go RED |
|---|---|---|
| alias exclusion | in `HashProjection`, `if false && t.Alias != "" { t.Alias = "" }` — alias survives into the projection | `TestHashProjection_ExcludesAlias` |
| array sorting (supporting hunk) | `if false && true { sort.Strings(sigs) }` — drop the sort in `SignatureSet` | `TestSignatureSet_SortedAndStable` |
| ctor retention (supporting hunk) | `if false && len(t.Ctors) > 0 { out.Ctors = t.Ctors }` — constructors silently dropped | `TestHashProjection_Deterministic` (fixture carries an ADT; projection changes) |

---

### M2 — `iface.BuildCanonicalJSON` + the hidden `internal-dump-iface` subcommand (0.5 day, ~110 LOC + ~70 test LOC)

**Ships:** `iface.BuildCanonicalJSON(ctx, packageDir, modulePath) ([]byte, error)` — resolves
`filepath.Join(packageDir, modulePath) + ".ail"` (the loader rule, `internal/loader/loader.go:372`),
runs `pipeline.RunWithContext(ctx, pipeline.Config{DryLink: true, PackageDir: packageDir}, src)`,
returns `result.Interface.ToNormalizedJSON()`. Returns **errors**, never `os.Exit` (D7c's
library-shaped requirement, now satisfied one layer lower). Plus the `case "internal-dump-iface":`
arm in `cmd/ailang/main.go`'s dispatch switch, deliberately absent from `printHelp` (`cmd/ailang/help.go:231` — measured: there is no `printUsage`).
`outputInterface` is repointed at `BuildCanonicalJSON` **in-process** (D5) — same bytes as today,
`os.Exit(1)` handling stays in the CLI.

**Why second:** it is the single definition FIX-1 demands. Landing it before the subprocess wrapper
means the wrapper has something real to shell out to, and `ailang iface` regressions surface
immediately (M2-A4).

**Acceptance:**

| # | Command | Base |
|---|---|---|
| M2-A1 | `-run '^TestBuildCanonicalJSON_ReturnsErrorNotExit$'` in `./internal/iface/`, `--- PASS` form | **RED** |
| M2-A2 | `-run '^TestBuildCanonicalJSON_ModulePathResolution$'` (declared path with no `.ail` file ⇒ error naming the missing file) | **RED** |
| M2-A3 | `-run '^TestBuildCanonicalJSON_ContextCancelled$'` (cancelled ctx ⇒ `ctx.Err()`) | **RED** |
| M2-A4 | `go test ./cmd/ailang/ -run '^TestOutputInterface' -v` — **scoped** (`./cmd/ailang/...` unscoped is 311s, D7) | **GREEN-or-absent at base**: measured `go test ./cmd/ailang/... ` = `rc=0` in 311.052s. Executor must add `TestOutputInterface_ByteIdenticalToPreRefactor` with a golden file captured from the **pre-refactor** binary; the gate is that golden. |
| M2-A5 | `./bin/ailang internal-dump-iface <fixture-dir> <mod> > /tmp/d.json 2>&1; rc=$?` then `./bin/ailang iface <fixture-dir>/<mod>.ail > /tmp/i.json 2>&1; diff /tmp/d.json /tmp/i.json` — the subcommand and the public command must agree byte-for-byte (D4) | **RED** (subcommand does not exist; measured `grep -rn internal-dump-iface .` hits only the design doc, control `grep -c iface cmd/ailang/main.go` = 7) |
| M2-A6 | shared build/vet/boundaries block | **GREEN at base** |

**Mutations:**

| Hunk shipped | Mutation | Test RED |
|---|---|---|
| `+ ".ail"` resolution | `if false && !strings.HasSuffix(f, ".ail") { f += ".ail" }` | `TestBuildCanonicalJSON_ModulePathResolution` |
| ctx propagation | pass `context.Background()` instead of `ctx` to `RunWithContext` | `TestBuildCanonicalJSON_ContextCancelled` |
| error-not-exit shape | replace the missing-file `return nil, err` with `os.Exit(1)` | `TestBuildCanonicalJSON_ReturnsErrorNotExit` |
| **switch-arm registration (supporting hunk — the one that ships unpinned)** | delete `case "internal-dump-iface":` from `cmd/ailang/main.go` | M2-A5 (subcommand/public-command diff) |
| `PackageDir` wiring (supporting hunk) | `if false && packageDir != "" { cfg.PackageDir = packageDir }` — intra-package imports stop resolving | `TestBuildCanonicalJSON_ResolvesIntraPackageImport` (fixture with a module importing a sibling) |

---

### M3 — `pkg.BuildModuleIface` subprocess wrapper + `PublishLimits` (1 day, ~180 LOC + ~200 test LOC)

**Ships:** `internal/pkg/iface_subprocess.go` —
`BuildModuleIface(ctx, packageDir, modulePath) (*iface.InterfaceJSON, error)` (**signature
corrected per D2**), invoking `ailang internal-dump-iface` via `exec.CommandContext` with
`cmd.Dir = packageDir`, its own process group (mirroring `publish_validator.go`'s
`setProcessGroup`), and a per-module `context.WithTimeout`. Plus the `PublishLimits` struct
(overall 60s / per-module 10s / max 64 exported modules) as the one named tuning point.
Binary resolution: `os.Executable()` for the publisher, `exec.LookPath("ailang")` for the
validator, injectable for tests.

**Why here, and why it is a full day:** this is the highest-risk *reversible* item in the sprint.
It needs package fixtures on disk, a real `ailang` binary, process-group kill semantics, and four
timing/cancellation tests. The doc folds this plus M1, M2 and M4 into one 1-day milestone; that is
the single largest estimate error in the doc.

**Timeout tests must not sleep 10s.** `PublishLimits` is injectable; `TestBuildModuleIface_PerModuleDeadline`
sets the per-module deadline to 50ms against a fixture that blocks, so the suite stays fast.

**Acceptance** (all in `./internal/pkg/`, `--- PASS` form; all **RED at base** — measured, the
package has no such tests and the corrected form reports RED where the doc's form reported
`rc=0 [no tests to run]`):

- M3-A1 `^TestBuildModuleIface_ReturnsError$` — missing file / type error / non-zero child ⇒ error, never `os.Exit`
- M3-A2 `^TestBuildModuleIface_Cancellation$` — cancelling the caller kills the child and returns `ctx.Err()`
- M3-A3 `^TestBuildModuleIface_PerModuleDeadline$` — injected 50ms deadline ⇒ typed timeout error
- M3-A4 `^TestBuildModuleIface_ExportLimit$` — >`MaxExportedModules` ⇒ refusal
- M3-A5 `^TestBuildModuleIface_MatchesInProcess$` — subprocess bytes == `iface.BuildCanonicalJSON` bytes
- M3-A6 `go test ./internal/pkg/ > /tmp/o 2>&1; rc=$?` — whole package still green. **GREEN at base** (measured `rc=0`, 1.255s)
- M3-A7 shared build/vet/boundaries. **GREEN at base**

> **Executor note (silent-green trap):** do **not** gate these on `internal/testutil.FindAilangBinary`,
> which `t.Skipf`s on a stale binary (`internal/testutil/ailangbin.go:74-79`) — a skip is `rc=0`.
> Build the child once in `TestMain` via `go build -o <t.TempDir()>/ailang ./cmd/ailang` and
> `t.Fatal` if that build fails. The `--- PASS:` assertion in every command above also catches a skip.

**Mutations:**

| Hunk shipped | Mutation | Test RED |
|---|---|---|
| per-module timeout | `if false && lim.PerModule > 0 { ctx, cancel = context.WithTimeout(ctx, lim.PerModule) }` | `TestBuildModuleIface_PerModuleDeadline` |
| ctx into the child | `exec.Command(...)` instead of `exec.CommandContext(ctx, ...)` | `TestBuildModuleIface_Cancellation` |
| export cap | `if false && len(mods) > lim.MaxExportedModules { return err }` | `TestBuildModuleIface_ExportLimit` |
| **`cmd.Dir` (supporting hunk)** | `if false && packageDir != "" { cmd.Dir = packageDir }` — child runs in the wrong CWD, imports break | `TestBuildModuleIface_MatchesInProcess` |
| **process-group kill (supporting hunk)** | drop `setProcessGroup(cmd)` | `TestBuildModuleIface_Cancellation` (orphan child outlives cancel) |
| **`PublishLimits` defaults (supporting hunk)** | zero the struct literal's defaults | `TestBuildModuleIface_PerModuleDeadline` |

---

### M4 — `InterfaceHashV2` + `InterfaceHashVersion` (1 day, ~150 LOC + ~230 test LOC)

**Ships:** `pkg.InterfaceHashV2(ctx, packageDir, m *PackageManifest) (string, []string, error)`
returning `sha256:ifacev2:<64hex>` **and** the signature set; folds, per exported module (sorted by
path), `iface.HashProjection` of that module's canonical JSON, then name / edition / sorted max
effects — the legacy fields kept so a v2 hash is a strict superset of what legacy folded. Plus
`InterfaceHashVersion(string) int` → `0` | `2`. Legacy `InterfaceHash` is **untouched** (AC11).
Nothing calls `InterfaceHashV2` yet — this milestone is pure library.

**Acceptance** (all `./internal/pkg/`, `--- PASS` form, all **RED at base**, measured):

- M4-A1 `^TestInterfaceHashV2_SensitiveToAddedExport$` — the reporter's exact field case (V5)
- M4-A2 `^TestInterfaceHashV2_SensitiveToRemovedExport$`
- M4-A3 `^TestInterfaceHashV2_SensitiveToRetype$`
- M4-A4 `^TestInterfaceHashV2_Deterministic$` — same dir hashed 10× ⇒ identical
- M4-A5 `^TestInterfaceHashV2_OrderIndependent$` — reordering `[exports]` / `[effects] max` ⇒ identical
- M4-A6 `^TestInterfaceHashVersion$` — `"sha256:ifacev2:…"`→2, `"sha256:…"`→0, garbage→0
- M4-A7 `^TestInterfaceHashV2_IgnoresTypeAliases$` — **the real D6 guard** (the doc's AC10 mutation is invalid, D8)
- M4-A8 `go test ./internal/pkg/ -run '^TestInterfaceHash_' -v` — legacy tests still pass. **GREEN at base** (measured `rc=0`)
- M4-A9 `go test ./internal/iface/ -run '^TestXModAlias' -v` (2 PASS lines). **GREEN at base**
- M4-A10 shared build/vet/boundaries. **GREEN at base**

**Mutations:**

| Hunk shipped | Mutation | Test RED |
|---|---|---|
| fold the module projection | `if false && len(proj) > 0 { h.Write(proj) }` | `TestInterfaceHashV2_SensitiveToAddedExport` (+A2, +A3) |
| canonical (not raw) projection | fold `ToNormalizedJSON` output instead of `HashProjection` | `TestInterfaceHashV2_IgnoresTypeAliases` |
| `ifacev2` marker | `if false && true { prefix = "sha256:ifacev2:" }` | `TestInterfaceHashVersion` |
| **sort of `[exports]` (supporting hunk)** | `if false && true { sort.Strings(exports) }` | `TestInterfaceHashV2_OrderIndependent` |
| **error propagation (supporting hunk)** | `if false && err != nil { return "", nil, err }` — a broken module silently yields a hash | `TestInterfaceHashV2_RefusesOnBrokenModule` |
| **effects fold retained (supporting hunk)** | drop the `effect:` writes | `TestInterfaceHashV2_SensitiveToEffectCeiling` |

---

### M5 — Signature-set classification with the `U` class (1 day, ~160 LOC + ~180 test LOC)

**Ships:** `Signatures []string` on `messaging.PackageVersionInfo` (`pkg_events.go:13-20`);
`classifyChange` rewritten from the 9-line dead-branch function (`pkg_events.go:162-170`) to the
**2×2 in D3** returning `A`/`B`/`C`/`U`; `ChangeClass`/`Breaking` set on
`EmitInterfaceChangeNotice` and `EmitUpgradeAvailable`; `autonomy_router.go:61`
(`PkgMsgInterfaceChange`) extended so `U` routes to review (`ChangeClassC`), not auto-apply.
The existing `TestClassifyChange` (`pkg_events_test.go:191`) is **replaced**, not extended — per
`.claude/rules/coding-standards.md`, out-of-date tests are removed. Its "content only"/"no change"
cases encode the dead-code behaviour and must not survive.

**Why last in Sprint 1:** it changes cascade routing. Under the D3 2×2 the change is *dark* while
no producer emits signatures (legacy-vs-legacy is byte-identical to today's behaviour), so it is
safe to land — but it is the milestone whose blast radius is hardest to reason about, so it goes
after the machinery it depends on is green.

**Acceptance:**

| # | Command | Base |
|---|---|---|
| M5-A1 | `./internal/messaging/ -run '^TestClassifyChange_BothLegacy_UnchangedBehaviour$'` — **the D3 guard**: legacy/legacy still yields today's `A`/`C` | **RED** |
| M5-A2 | `-run '^TestClassifyChange_AdditiveVsBreaking$'` — v2/v2, new ⊇ old ⇒ `B`; removal/retype ⇒ `C` | **RED** |
| M5-A3 | `-run '^TestClassifyChange_UnknownOldSide$'` — legacy old + v2 new ⇒ `U` | **RED** |
| M5-A4 | `-run '^TestClassifyChange_UnknownNewSide$'` — v2 old + legacy new ⇒ `U` | **RED** |
| M5-A5 | `./internal/coordinator/ -run '^TestAutonomyRouter_UnknownRoutesToReview$'` | **RED** |
| M5-A6 | `go test ./internal/messaging/ > /tmp/o 2>&1; rc=$?` — package green. **GREEN at base** (measured `rc=0`, 5.222s) | **GREEN** |
| M5-A7 | `grep -c 'content only' internal/messaging/pkg_events_test.go` must be `0` — the stale case is deleted, not left passing. Control at base: `grep -c 'content only' …` = **1** (measured, `pkg_events_test.go:197`) | **RED** (currently 1) |
| M5-A8 | shared build/vet/boundaries | **GREEN at base** |

**Mutations:**

| Hunk shipped | Mutation | Test RED |
|---|---|---|
| `U` on legacy-old + v2-new | `if false && old.Signatures == nil && new.Signatures != nil { return "U" }` | `TestClassifyChange_UnknownOldSide` |
| legacy/legacy stays hash-only | `if false && old.Signatures == nil && new.Signatures == nil { return legacyClassify(old,new) }` — falls into `U` | `TestClassifyChange_BothLegacy_UnchangedBehaviour` |
| superset ⇒ `B` | `if false && isSuperset(new.Signatures, old.Signatures) { return "B" }` | `TestClassifyChange_AdditiveVsBreaking` |
| **`ChangeClass` actually set on the envelope (supporting hunk)** | `if false && cls != "" { env.Package.ChangeClass = cls }` | `TestEmitInterfaceChangeNotice_CarriesChangeClass` |
| **router case (supporting hunk)** | delete the `case "U":` arm in `autonomy_router.go` | `TestAutonomyRouter_UnknownRoutesToReview` |

---

## 3. Sprint 2 — everything irreversible (5 days, DEFERRED)

Not planned in day-granularity here; it should be re-planned against Sprint 1's measured outcome,
because M9's size depends on registry facts nobody in this loop can see (V-log #37 is UNVERIFIED).
Named so the deferral is explicit, not silent.

| # | Milestone | Est. | Why it is Sprint 2 |
|---|---|---|---|
| M6 | Wire `cmd/ailang/pkg_publish.go` to `InterfaceHashV2`; publish refuses on a broken exported module (D7a) | 0.5d | First user-visible behaviour change. Refuses publishes that previously succeeded, with an **unmeasured blast radius** (V-log #37). Land behind `AILANG_PKG_IFACE_V2=1`, default off, until the radius is measured. |
| M7 | Wire `cmd/registry-validator/main.go:214`; cross-binary consistency test; **version-skew guard** (D6) — validator rejects HTTP 400 on hash mismatch instead of silently overwriting | 1d | **Irreversible: registry writes.** The moment this lands, newly published packages carry a v2 identity. |
| M8 | Persist `interface_signatures` on `PackageMetadata` / `PackageVersionInfo`; thread through lockfile, pubsub, coordinator rows, Firestore converter | 1d | **Irreversible: schema on published records.** Touches 8 persistence points enumerated in the doc's Conflict Surface. |
| M9 | Registry backfill: GCS tarball download → extract → recompute → checkpoint → resume; bounded batch N=50, per-download 30s, per-compile 10s, 3 retries, `failed`/`unbackfillable` states | **2.5d** | **Irreversible: rewrites historical registry metadata.** The doc budgets 1 day. A resumable, checkpointed, retrying batch job over an object store with two distinct failure taxonomies is not a 1-day item; the doc's own FIX-2 requirements (AC18/AC19) each imply persistence and a resume path. |

**Precondition on M6/M9 (blocking, cannot be satisfied from this session):** run the V-log #37
command against the live registry to bound how many published versions would newly be refused by
the D7a gate. It is recorded UNVERIFIED in the design and stays UNVERIFIED here — I could not reach
the registry. Shipping M6 without that number is shipping an unbounded regression.

---

## 4. Honest estimate

| | Doc | This plan | Delta |
|---|---|---|---|
| Canonical serialization + subcommand + subprocess builder + v2 hash (doc's M1) | 1 day | **3 days** (M1+M2+M3+M4) | +2 |
| Wire both binaries (doc's M2) | 1 day | **1.5 days** (M6+M7, +version-skew guard) | +0.5 |
| Signature set + classification (doc's M3) | 1 day | **1 day** (M5) | 0 |
| Persistence threading | *(folded into M3)* | **1 day** (M8) | +1 |
| Backfill (doc's M4) | 1 day | **2.5 days** (M9) | +1.5 |
| **Total** | **4 days** | **9 days** | **+5** |

Where the doc's estimate goes wrong: its M1 is four separable pieces of work (a pure projection, an
in-process builder + CLI arm, a subprocess wrapper with process-group and timing semantics, and the
hash itself) carrying 14 of its 19 acceptance criteria; and its M4 treats a checkpointed, resumable,
retrying object-store batch job as a one-day item.

**Genuinely shippable first sprint: M1–M5 (4 days).** It is coherent on its own — `ailang` gains a
signature-sensitive v2 hash and signature set computable for any package, an isolated subprocess
builder with real deadlines, and a classifier that can say `unknown` honestly — while remaining
completely dark: no binary emits a v2 hash, no registry record moves, no published package's
identity changes, and `git revert` at any milestone boundary is clean.

---

## 5. Handoff notes

- **The sprint JSON is at `.ailang/state/sprints/sprint_M-REGISTRY-INTERFACE-HASH-BLIND-TO-SIGNATURES.json`,
  and `.ailang/` is gitignored** (`git check-ignore -v` → `rc=0`, `.gitignore:82`). A plain
  `git add` will **not** pick it up. The controller must `git add -f` it or accept that only this
  markdown is committed. Flagging so it is not silently lost.
- No git write command was run by the planner. Both deliverables are untracked/modified files in
  the worktree.
- The design doc itself is **not** edited by this plan. D1–D8 are recorded here; if the controller
  wants them folded back into the design, that is a separate doc revision (and D3 arguably warrants
  one, since it changes D5's table).
