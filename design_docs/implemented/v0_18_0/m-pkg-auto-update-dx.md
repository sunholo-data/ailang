# M-PKG-AUTO-UPDATE-DX: Package Auto-Update Bus DX Fixes

**Status**: Implemented
**Target**: v0.18.0
**Priority**: P2 (Blocking autonomy story; not a production regression)
**Estimated**: 1 day (~6-8 hours)
**Dependencies**: None hard. Compose with M-MOTOKO-EXECUTOR-ADAPTER (same release).
**Author**: Claude + Mark
**Created**: 2026-05-08
**Source messages**: 4a16ab61 (motoko_explore), 202d5eef (sprint-executor), 72d25729 (motoko_explore)

---

## Executive Summary

Three independent agents stress-tested the package auto-update bus during the motoko-ext-* publish session (2026-05-07) and hit four DX gaps that together make the autonomy story — where motoko self-modifies an extension and fires a cascade — structurally impossible without manual workarounds. None are regressions against existing tests; all were latent design gaps exposed by first production use.

**Four issues (escalating severity):**

1. **DISCOVERABILITY** — `ailang publish` absent from `ailang pkg` help; operators reach `notify-upgrade` first, hit wall #2, give up.
2. **PATH-DEP REJECTION** — `ailang pkg notify-upgrade` fails with `"requires from_interface_hash and to_interface_hash"` for any path-based dependency. The entire motoko-ext-* fleet is consumed via path deps, so the auto-update bus cannot fire for any motoko extension today.
3. **WHITESPACE-FRAGILE path-dep rewrite** — `rewritePathDepsForPublish` matches `"name" = { path = "..." }` with single-space exact string only; aligned TOML (padded for column alignment) silently fails to match, and the published tarball ships with unresolvable paths.
4. **SILENT RATCHET** — bumping `[package].version` in `ailang.toml` without running `ailang publish` causes `ailang lock` in consumers to silently re-resolve the new version: no announcement, no cascade-test, no interface-hash verification. This is the failure mode M-PKG-AUTONOMOUS-CASCADE-SAFE was designed to prevent, but it only prevents it if the publisher uses the `ailang publish` path.

---

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No semantic change; tooling only |
| A2: Replayability | +1 | Regex path-dep rewrite produces deterministic output regardless of TOML formatting |
| A3: Effect Legibility | 0 | No effect-system change |
| A4: Explicit Authority | +2 | Ratchet guard makes the "publish is required" authority boundary explicit |
| A5: Bounded Verification | +1 | path-dep hash support closes verification gap for local packages |
| A6: Safe Concurrency | 0 | None |
| A7: Machines First | +1 | notify-upgrade for path deps enables autonomous cascade-bus participation |
| A8: Minimal Syntax | 0 | No new syntax |
| A9: Cost Visibility | 0 | None |
| A10: Composability | +1 | Any agent workspace (not just motoko) benefits from path-dep notify-upgrade |
| A11: Structured Failure | +2 | Issues 2 and 3 replace silent/misleading failures with actionable errors |
| A12: System Boundary | +2 | Ratchet guard makes the ailang.toml↔registry boundary explicit |

**Net Score: +10** → **Decision: ✅ Proceed to implementation**

---

## Issue 1: Discoverability — `ailang publish` missing from `ailang pkg` help

### Current behaviour

```
$ ailang pkg
Usage: ailang pkg <command>

Commands:
  info <vendor/name>          Show detailed package information
  versions <vendor/name>      List all versions with hashes
  stats                       Show ecosystem-wide statistics
  provenance <pkg>@<ver>      Show provenance chain for a version
  history <pkg>@<ver>         Show version history timeline
  notify-upgrade <pkg>@<ver>  Emit upgrade-available message
  affected-by <pkg>           List workspaces depending on a package
```

`ailang publish` is a top-level command (`ailang publish`, not `ailang pkg publish`), but the `ailang pkg` help never mentions it. An operator looking for "how do I announce an upgrade?" sees `notify-upgrade` first and never discovers that `ailang publish` is the correct path that does notify + cascade-topic + AI-escalation routing.

### Fix

Add a cross-reference at the bottom of `ailang pkg` help:

```
  notify-upgrade <pkg>@<ver>  Emit upgrade-available message (manual fallback)
  affected-by <pkg>           List workspaces depending on a package

To publish a new version and fire the full cascade bus:
  ailang publish              (preferred — wraps notify-upgrade + cascade-topic)
```

Also add `ailang publish` to the top-level `ailang --help` command listing under a "Packages" section if it is not already there.

**File**: `cmd/ailang/main.go:393-407` (the `ailang pkg` help block).

---

## Issue 2: PATH-DEP REJECTION in `ailang pkg notify-upgrade`

### Root cause

`pkgNotifyUpgradeCommand` (`cmd/ailang/pkg_msg.go`) computes `toInterfaceHash` only inside this branch:

```go
if manifest.Package.Name == pkgName {
    // We're the package being upgraded — compute current hashes
    resolved, resolveErr := pkg.ResolveDependencies(manifest, cwd)
    for _, r := range resolved {
        if r.Name == pkgName { ... }  // package won't find itself here
    }
}
```

Two problems:

1. **Wrong branch when run from consumer**: If the user runs `notify-upgrade sunholo/motoko_ext_a2a@0.1.1` from `motoko_agent/` (the host), `manifest.Package.Name` is `motoko_agent/src`, so the branch never executes and `toInterfaceHash` stays `""`.

2. **Wrong branch when run from the package itself**: If run from `motoko_ext_a2a/`, `manifest.Package.Name == pkgName` matches, but `ResolveDependencies` resolves the package's *dependencies*, not the package itself — so the loop over `resolved` never finds `pkgName`, and `toInterfaceHash` stays `""`.

`pkg.InterfaceHash(m)` already accepts a `*PackageManifest` and computes the hash deterministically from exports + effects (`internal/pkg/hasher.go:73`). The fix is straightforward: when run from inside the package directory, call `pkg.InterfaceHash(manifest)` directly. When run from a consumer that has a path dep, resolve the path dep manifest and call `InterfaceHash` on it.

### Fix

In `pkgNotifyUpgradeCommand`, replace the `toInterfaceHash` computation block:

```go
// Current manifest IS the package being upgraded — compute hash directly.
if manifest.Package.Name == pkgName {
    toInterfaceHash = pkg.InterfaceHash(manifest)
    if fromInterfaceHash != "" && toInterfaceHash != fromInterfaceHash {
        changeClass = "C"
    }
} else {
    // We're a consumer — look for a path dep pointing at this package.
    for depName, dep := range manifest.Dependencies {
        if depName != pkgName || dep.Path == "" {
            continue
        }
        depDir := dep.Path
        if !filepath.IsAbs(depDir) {
            depDir = filepath.Join(cwd, dep.Path)
        }
        depManifest, err := pkg.LoadManifest(depDir)
        if err != nil {
            return fmt.Errorf("cannot load path dep %s at %s: %w", pkgName, dep.Path, err)
        }
        toInterfaceHash = pkg.InterfaceHash(depManifest)
        if fromInterfaceHash != "" && toInterfaceHash != fromInterfaceHash {
            changeClass = "C"
        }
        break
    }
}
```

If `toInterfaceHash` is still empty after this block (package not found in manifest, not the current package, and not a path dep), fall back to the existing registry-resolver path. The validation check `"requires from_interface_hash and to_interface_hash"` should remain — but it should only fire when the package cannot be located at all, not for the common path-dep case.

**Files**: `cmd/ailang/pkg_msg.go` (pkgNotifyUpgradeCommand).

---

## Issue 3: WHITESPACE-FRAGILE path-dep rewrite in `ailang publish`

### Current behaviour

`rewritePathDepsForPublish` (`cmd/ailang/pkg_publish.go:125`) does:

```go
old := fmt.Sprintf(`"%s" = { path = "%s" }`, depName, dep.Path)
if strings.Contains(content, old) {
    content = strings.Replace(content, old, replacement, 1)
}
```

This matches only the exact string `"name" = { path = "value" }`. TOML authors commonly align multi-dep blocks for readability:

```toml
"motoko-ext-abi"     = { path = "../motoko-ext-abi" }
"motoko-ext-compose" = { path = "../motoko-ext-compose" }
```

The padded form silently fails to match. The dep is not rewritten. The published tarball ships the raw path dep. The registry validator returns a `-32600` parse error at install time.

### Fix

Replace the exact string match with a regex match that tolerates any whitespace around `=`, `{`, and between keys:

```go
import "regexp"

// Build a pattern that matches any whitespace variant of:
//   "depName" = { path = "depPath" }
// Capture groups let us identify the match for replacement.
escapedName := regexp.QuoteMeta(depName)
escapedPath := regexp.QuoteMeta(dep.Path)
pattern := fmt.Sprintf(
    `"%s"\s*=\s*\{\s*path\s*=\s*"%s"\s*\}`,
    escapedName, escapedPath,
)
re, err := regexp.Compile(pattern)
if err != nil {
    return false, fmt.Errorf("internal: bad rewrite pattern for %s: %w", depName, err)
}
if re.MatchString(content) {
    content = re.ReplaceAllLiteralString(content, replacement)
    fmt.Printf("  %s Rewrote dep %s: path %q → registry %s\n", cyan("→"), depName, dep.Path, version)
    rewritten = true
}
```

`regexp.QuoteMeta` escapes any special characters in the dep name or path, so the pattern is safe for all real-world values.

**Files**: `cmd/ailang/pkg_publish.go` (rewritePathDepsForPublish).

**Test to add**: A test case in `cmd/ailang/pkg_publish_test.go` (or a new test file) that verifies rewriting succeeds for:
- single-space format: `"name" = { path = "val" }`
- aligned format: `"name"   = { path = "val" }`
- extra spaces inside braces: `"name" = {  path = "val"  }`

---

## Issue 4: SILENT RATCHET — version bump without `ailang publish`

### Problem

When a developer edits `[package].version` in `ailang.toml` and pushes (as happened in commit `c2d5b47` for `motoko-ext-a2a`), the next `ailang lock` in any consumer silently re-resolves to the new version:

- No `upgrade-available` message in the package inbox
- No cascade-test triggered
- No interface-hash comparison
- No PR required

This defeats the purpose of M-PKG-AUTONOMOUS-CASCADE-SAFE. The constraint was: "publish is the gate." Without enforcement, the gate is optional.

### Fix options

Three candidates, ordered by implementation cost:

#### (a) `ailang lock` warning — LOW COST, INCOMPLETE

When `ailang lock` resolves a path dep at a higher version than the lockfile records, check whether an `upgrade-available` message exists in the package inbox for that version. If not, emit a warning:

```
⚠  sunholo/motoko_ext_a2a resolved to 0.1.1 (was 0.1.0 in lockfile)
   No upgrade-available message found. Run 'ailang publish' in the package
   directory to announce this change and trigger cascade tests.
```

This is best-effort: it requires the inbox DB to be reachable. Good for developer feedback; does not enforce.

#### (b) `ailang-packages` pre-commit hook — MEDIUM COST, ENFORCES AT SOURCE

A pre-commit hook in `ailang-packages/` that detects `[package].version` changed since the last commit and fails if no `ailang publish` run is recorded in `~/.ailang/state/publish_log.db` for that (package, version) pair:

```
Error: [package].version changed to 0.1.1 in packages/motoko-ext-a2a/ailang.toml
       but 'ailang publish sunholo/motoko_ext_a2a@0.1.1' was not run.
       Run 'ailang publish' before committing a version bump.
```

**Requires**: `ailang publish` writes a record to `~/.ailang/state/publish_log.db` (currently it does not persist anything locally).

#### (c) `ailang publish` verifies version is fresh — MEDIUM COST, ENFORCES AT GATE

Before publishing, `ailang publish` checks whether the lockfile in any detected dependent workspace already records this `(package, version)` without a corresponding `upgrade-available` message. If so:

```
Error: sunholo/motoko_ext_a2a@0.1.1 is already present in motoko_agent/ailang.lock
       without a matching upgrade-available message. This indicates a silent ratchet.
       Reset version to 0.1.0, run 'ailang publish', then update consumers.
```

This catches the case after the fact (publish time) rather than at commit time.

### Recommendation

Ship **(a) lock warning** in this milestone (simple, non-breaking, immediately useful). File **(b) pre-commit hook** as a follow-on spike (requires instrumenting `ailang publish` local history). Skip (c) — it enforces the wrong invariant (re-publishing is valid; the constraint is on the consumer side).

**Files for (a)**: `cmd/ailang/pkg_commands.go` (lockCommand), `internal/pkg/resolver.go` (where version changes are detected).

---

## Out of Scope (Related — Tracked Separately)

**Motoko decision framework encoding** (msg 72d25729): Where should ext-vs-script, modify-vs-create, and publish-discipline rules live in motoko's prompt/tool surface? Options: (B) `SYSTEM.md` addendum + (C) `motoko-ext-precreate-lint` tool. This is a motoko-side design question, not an AILANG-core fix. Track as `design_docs/planned/` in motoko_agent repo if motoko_explore proceeds.

---

## Milestones

### M1 — Discoverability fix (30 min)

- Add `ailang publish` cross-reference to `ailang pkg` help text.
- Verify `ailang --help` lists publish under packages.

**Acceptance**: `ailang pkg` output contains "ailang publish" with a one-line description.

### M2 — Path-dep notify-upgrade fix (2-3 hours)

- Replace `toInterfaceHash` computation in `pkgNotifyUpgradeCommand` with path-dep-aware logic.
- Add unit test: run `notify-upgrade sunholo/motoko_ext_a2a@0.1.1` from a consumer workspace that has `motoko-ext-a2a` as a path dep; verify message sends without error and `toInterfaceHash` is non-empty.
- Add unit test: run from inside the package itself; same verification.
- Repro from msg 4a16ab61 #2 must pass: `ailang pkg notify-upgrade sunholo/motoko_ext_a2a@0.1.1 --dry-run` succeeds in `motoko_agent/`.

**Acceptance**: Both test cases pass. No validation error for path deps.

### M3 — Whitespace-robust path-dep rewrite (1-2 hours)

- Replace exact-string match with `regexp`-based replacement in `rewritePathDepsForPublish`.
- Add table-driven tests covering single-space, aligned, and extra-space TOML formats.
- Verify motoko-ext-compose (the original failing case) publishes correctly.

**Acceptance**: All three whitespace variants produce a correct `"name" = "version"` rewrite. No silent no-op.

### M4 — Lock warning for silent ratchet (1-2 hours)

- In `lockCommand`, after resolving path deps, check if any resolved version is higher than the recorded lockfile version.
- If so, check messaging store for an `upgrade-available` message for `(pkgName, newVersion)`.
- If absent, print warning (not error) to stderr.
- Update CHANGELOG.

**Acceptance**: Running `ailang lock` after a silent version bump prints the warning. Running after a proper `ailang publish` does not. Existing `make test` passes.

---

## CHANGELOG entry (draft)

```markdown
### Fixed
- `ailang pkg notify-upgrade` now works for path-based dependencies;
  previously failed with "requires from_interface_hash" for the entire
  motoko-ext-* fleet (msg 4a16ab61 M2)
- `ailang publish` path-dep rewrite now handles TOML-aligned formatting;
  previously silently skipped column-padded deps, shipping broken tarballs
  (msg 202d5eef M3)

### Improved
- `ailang pkg` help now cross-references `ailang publish` as the preferred
  publish path (msg 4a16ab61 M1)
- `ailang lock` warns when a path dep resolves to a higher version with no
  matching upgrade-available message (silent ratchet detection, msg 4a16ab61 M4)
```
