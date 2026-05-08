# Sprint Plan: M-PKG-AUTO-UPDATE-DX

**Status**: Planned
**Target**: v0.18.0
**Estimated**: 1 day (~6-8 hours)
**Source-of-truth design**: [m-pkg-auto-update-dx.md](m-pkg-auto-update-dx.md)

> This plan drives execution against the design doc. All architectural decisions, axiom scoring, risks, and rationale live there. This file is the milestone-by-milestone schedule.

---

## Why this sprint, why now

The motoko-ext-* publish session (2026-05-07) exposed four latent DX gaps that together make the autonomy story — where motoko self-modifies an extension and fires a cascade — structurally impossible without manual workarounds. These are all tooling-layer fixes with no semantic changes to the language. Risk surface is tiny; value is high.

---

## Velocity calibration

| Milestone | LOC impl | LOC tests | Total | Notes |
|-----------|----------|-----------|-------|-------|
| M1 (help text) | 10 | 0 | 10 | Pure text change |
| M2 (path-dep notify-upgrade) | 60 | 80 | 140 | Core logic fix + unit tests |
| M3 (whitespace-robust rewrite) | 25 | 60 | 85 | Regex swap + table tests |
| M4 (lock warning) | 50 | 40 | 90 | Warning emission + test |
| **Total** | **145** | **180** | **325** | |

Reference: M-MOTOKO-EXECUTOR-ADAPTER shipped ~500 LOC in 2 sessions. This sprint is smaller.

---

## Milestones

### M1 — Discoverability fix

**Estimated**: 30 min, ~10 LOC
**Files**: `cmd/ailang/main.go`

**Tasks:**
1. In the `ailang pkg` help block (`main.go:393-407`), add a blank line after the last command, then:
   ```
   To publish a new version and fire the full cascade bus:
     ailang publish              (preferred — wraps notify-upgrade + cascade-topic)
   ```
2. Annotate `notify-upgrade` line with `(manual fallback)`.
3. Verify `ailang pkg` (no args) prints the new text.

**Acceptance criteria:**
- `ailang pkg` output contains "ailang publish" with description.
- `notify-upgrade` line notes it's a manual fallback.
- No other behaviour changed.

---

### M2 — Path-dep support in `notify-upgrade`

**Estimated**: 2-3 hours, ~140 LOC
**Files**: `cmd/ailang/pkg_msg.go`

**Tasks:**
1. Replace the `toInterfaceHash` computation block in `pkgNotifyUpgradeCommand` with the two-branch logic from the design doc:
   - Branch A: `manifest.Package.Name == pkgName` → call `pkg.InterfaceHash(manifest)` directly.
   - Branch B: scan `manifest.Dependencies` for a path dep matching `pkgName` → load that manifest, call `pkg.InterfaceHash`.
   - Branch C (fallback): existing registry-resolver path.
2. Remove the dead loop over `resolved` that searched for the package in its own deps (it can never find itself there).
3. Keep the validation error for the case where the package cannot be located at all.

**Tests** (`cmd/ailang/pkg_msg_test.go` or new `pkg_notify_upgrade_test.go`):
- Run from consumer workspace that has the package as a path dep: verify `--dry-run` succeeds and `toInterfaceHash` is non-empty.
- Run from inside the package directory itself: same.
- Run with a package name that doesn't appear in manifest at all: verify the validation error fires.

**Acceptance criteria:**
- `ailang pkg notify-upgrade sunholo/motoko_ext_a2a@0.1.1 --dry-run` succeeds when run from `motoko_agent/` (consumer with path dep).
- `toInterfaceHash` in the output is a valid `sha256:...` string.
- All three test cases pass.

---

### M3 — Whitespace-robust path-dep rewrite

**Estimated**: 1-2 hours, ~85 LOC
**Files**: `cmd/ailang/pkg_publish.go`

**Tasks:**
1. In `rewritePathDepsForPublish`, replace the `strings.Contains` / `strings.Replace` match with a `regexp`-based replacement:
   - Pattern: `"<depName>"\s*=\s*\{\s*path\s*=\s*"<depPath>"\s*\}` (with `regexp.QuoteMeta` for both values).
   - Use `re.ReplaceAllLiteralString` for the replacement.
2. Add `"regexp"` import.

**Tests** (`cmd/ailang/pkg_publish_test.go` or new file):
Table-driven test for `rewritePathDepsForPublish` covering:
- Single-space format: `"name" = { path = "val" }` ✓
- Aligned format: `"name"   = { path = "val" }` (the original failing case) ✓
- Extra inner spaces: `"name" = {  path = "val"  }` ✓
- Non-matching dep name: no rewrite, no error ✓

**Acceptance criteria:**
- All four whitespace table cases pass.
- `rewritten = true` returned for all matching cases.
- No silent no-op for aligned TOML.

---

### M4 — Lock warning for silent ratchet

**Estimated**: 1-2 hours, ~90 LOC
**Files**: `cmd/ailang/pkg_commands.go` (lockCommand), `internal/pkg/resolver.go` or `cmd/ailang/pkg_commands.go`

**Tasks:**
1. After `ailang lock` resolves dependencies, compare each path dep's resolved version against the previous lockfile:
   - If `resolvedVersion > lockedVersion` (string compare is fine; semver ordering is a stretch goal), check the messaging store for an `upgrade-available` message for `(pkgName, resolvedVersion)`.
   - If absent, print to stderr:
     ```
     ⚠  <pkgName> resolved to <newVer> (was <oldVer> in lockfile)
        No upgrade-available message found. Run 'ailang publish' in the
        package directory to announce this change and trigger cascade tests.
     ```
2. This is a warning, not an error. `ailang lock` still succeeds and writes the lock file.
3. Open the messaging store read-only; if it cannot be opened (no DB yet), skip the check silently.

**Tests:**
- Construct a scenario where a path dep's manifest version is higher than the lockfile: verify warning fires.
- Construct a scenario where an `upgrade-available` message exists for the new version: verify no warning.
- Construct a scenario where no previous lockfile exists: verify no warning (first lock run).

**Acceptance criteria:**
- Warning appears on stderr when a path dep resolves to a higher version with no `upgrade-available` message.
- No warning when message exists.
- `ailang lock` exit code is 0 in both cases.
- Existing `make test` green.

---

### M5 — CHANGELOG + design doc move

**Estimated**: 15 min
**Files**: `changelogs/v0.10-current.md`, `design_docs/`

**Tasks:**
1. Add v0.18.x entry to `changelogs/v0.10-current.md` with the four fixes.
2. Move `design_docs/planned/v0_18_0/m-pkg-auto-update-dx.md` to `design_docs/implemented/v0_18_0/` and set `**Status**: Implemented`.
3. Ack message `e234c455` if present (or any sprint-evaluator message for this sprint).

**Acceptance criteria:**
- Changelog has entry.
- Design doc in `implemented/`.

---

## Execution order

```
M1 (30 min) → M2 (2-3h) → M3 (1-2h) → M4 (1-2h) → M5 (15 min)
```

No parallelism needed — files are non-overlapping.

---

## Success metrics

- [ ] `ailang pkg` help cross-references `ailang publish`
- [ ] `ailang pkg notify-upgrade <path-dep-pkg>@<ver> --dry-run` succeeds for any path dep
- [ ] `rewritePathDepsForPublish` handles all whitespace variants
- [ ] `ailang lock` warns on silent version bump with no announcement
- [ ] `make test` green
- [ ] CHANGELOG updated
- [ ] Design doc moved to `implemented/`
