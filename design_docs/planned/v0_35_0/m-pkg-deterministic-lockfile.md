# M-PKG-DETERMINISTIC-LOCKFILE: ailang.lock as a Pure Function of Its Inputs

**Status**: Planned
**Milestone**: M-PKG-DETERMINISTIC-LOCKFILE
**Target version**: v0.35.0
**Author**: Claude (via agent-to-agent message plane, correlation by message id)
**Date**: 2026-09-02

## Problem Statement

`ailang lock` stamps a wall-clock `generated_at` timestamp into `ailang.lock`
(`internal/pkg/lockfile.go:65`, `GeneratedAt: time.Now().UTC()`). The committed lockfile
is therefore **not a pure function of its inputs**: every regeneration produces a diff
even when the resolved dependency graph is unchanged.

### Measured evidence (2026-09-02)

A coordinator task asked to change nothing produced `sunholo-data/ailang-packages` PR #55,
whose **entire diff** was:

```diff
-  "generated_at": "2026-05-02T22:35:52.582637388Z",
+  "generated_at": "2026-09-02T11:38:27.11853544Z",
```

No dependency change, no version change. The agent ran a verification step that
regenerated the lock, and the timestamp alone created a reviewable PR. That PR was closed
as noise.

### Why it matters beyond one PR

- The package cascade pipeline regenerates locks routinely; each regeneration produces a
  **phantom diff**. Reviewers learn to ignore lockfile changes — which is exactly when a
  real dependency change slips through.
- "Did anything actually change?" becomes unanswerable from the diff. This is the same
  class of defect as a task status that cannot distinguish work from no-work.
- It conflicts with AILANG's stated principle of **compositional determinism** — all
  non-determinism should be explicit, and a committed build artefact should be
  reproducible.

## Consumer Audit (performed 2026-09-02)

Question from the filing task: *does any consumer actually read `generated_at`?*

Audit of the repository:

| Consumer | Reads `LockFile.GeneratedAt`? |
|---|---|
| `internal/pkg/lockfile.go` | Writes only (`NewLockFile`). No read path in `LoadLockFile`, `Validate`, `FindPackage`, `ValidateContentHashes`, or `ValidateAgainstManifest`. |
| All non-test code in `internal/` and `cmd/` | **None.** `grep -rn '\.GeneratedAt' internal/pkg cmd/ailang` (excluding `_test.go`) returns zero hits for the lockfile type. |
| `internal/pkg/lockfile_test.go:71` | Yes — but only to *neutralize* it: `lf2.GeneratedAt = lf1.GeneratedAt // use same timestamp for determinism comparison`. The test already treats the field as a determinism hazard. |
| `cmd/ailang/pkg_notify_upgrade_test.go` | Contains a `generated_at` string in a test fixture, but for the notify payload, not the lockfile. |
| External consumers (ailang-packages repo, cascade) | The cascade regenerates locks; nothing in the pipeline keys behavior on the timestamp. It is a diff-only artifact. |

Other `GeneratedAt` fields elsewhere (`internal/manifest`, `internal/observatory`,
`cmd/ailang/chains_diagnostics.go`, `cmd/ailang/examples.go`) are **out of scope**: they
live in non-committed reports/manifests where a wall-clock stamp is semantically
meaningful. Only `ailang.lock` is a committed, diff-reviewed artefact.

**Conclusion: `generated_at` in `ailang.lock` has zero readers. Removal is free.**

## Design Goals

1. `ailang lock` run twice over unchanged inputs produces a byte-identical `ailang.lock`.
2. No phantom diffs in downstream package repos or the cascade pipeline.
3. Preserve schema compatibility for loading existing lockfiles (old files with
   `generated_at` must still parse).
4. If freshness provenance is ever wanted, it must live **outside** the committed artefact.

## Options Considered

### Option A — Drop `generated_at` entirely (RECOMMENDED)

Remove the field from `LockFile` and stop writing it.

- **Pros**: Simplest; the audit shows zero readers; the existing test already works around
  the field; restores lockfile-as-pure-function in one step. JSON unmarshal of old
  lockfiles with the field still succeeds if we keep tolerant parsing (unknown fields are
  ignored by default with `encoding/json`).
- **Cons**: Loses "when was this lock generated" provenance — but that provenance is
  unreliable anyway (it records regeneration time, not resolution-decision time), and git
  history already answers "when did the lock last *change*" far better.

### Option B — Move timestamp to a non-committed sidecar

Write `generated_at` to an uncommitted artefact (e.g. `.ailang/state/lock-meta.json` or
stdout/log only).

- **Pros**: Keeps a freshness signal available to tooling.
- **Cons**: Adds a new state file and lifecycle for a field nobody reads today; violates
  YAGNI. Can be added later if a real consumer appears.

### Option C — Keep field, populate from input-derived data

E.g. derive from the newest dependency's publish time or git rev.

- **Pros**: Keeps the schema field with deterministic content.
- **Cons**: Not all sources have a meaningful timestamp (path deps); the value would be
  semantically murky; complexity for zero current consumers.

### Option D — Content-hash guard in `ailang lock`

Keep the timestamp but skip writing the file when only `generated_at` differs.

- **Pros**: No schema change.
- **Cons**: The file on disk is *still* non-deterministic (regeneration after delete
  produces a different timestamp); the purity problem is papered over, not solved; adds a
  compare-before-write path that can drift.

## Recommended Design (Option A)

1. **`internal/pkg/lockfile.go`**:
   - Remove `GeneratedAt` from `LockFile`.
   - Remove the `time.Now()` call from `NewLockFile` (drops the `time` import).
   - Loading stays backward-compatible: `encoding/json` ignores unknown fields, so old
     lockfiles containing `generated_at` parse without error. State this explicitly in
     code comments and tests.
2. **`ailang lock` CLI**: on regeneration where content is unchanged, the file is
   byte-identical — `git status` shows nothing. No special-casing needed.
3. **Tests**:
   - Update `lockfile_test.go`: remove the `lf2.GeneratedAt = lf1.GeneratedAt`
     normalization; replace with a test asserting `NewLockFile` + `Save` twice yields
     byte-identical output.
   - Add a test that a lockfile JSON containing a legacy `generated_at` field loads
     successfully (forward-compat guarantee).
4. **Docs**: note in the package-management docs that `ailang.lock` is deterministic and
   safe to diff-review; any diff is a real dependency change.

### What about `ailang_version` and `generator`?

`AILANGVersion` and `Generator` are also environment-derived. `generator` is a fixed
string per tool and stable; `ailang_version` changes only when the toolchain changes —
which *is* a real, reviewable input change and arguably belongs in the diff. This doc
keeps both but flags them: if they later prove noisy (version bumps causing lockfile
churn across the fleet), apply the same removal reasoning in a follow-up.

## Acceptance Criteria

- [ ] `ailang lock` run twice over unchanged inputs produces byte-identical `ailang.lock`.
- [ ] No `time.Now()` on the lockfile generation path.
- [ ] Old lockfiles with `generated_at` still load (compat test).
- [ ] No phantom diffs from lock regeneration in the cascade pipeline (verify by
      re-running the PR #55 scenario: regenerate → `git diff --exit-code` passes).
- [ ] Docs updated.

## Out of Scope

- `generated_at` in manifests, observatory reports, chains diagnostics, examples
  listings (non-committed artefacts where wall-clock is meaningful).
- Determinism of resolution itself (version selection, registry ordering) — the resolver
  already sorts packages, effects, and exports; this doc covers only the wall-clock leak.

## Related Documents

- `design_docs/PROGRAM.md` — determinism as a core principle.
- Filing context: agent-to-agent message-plane task, correlation by message id;
  evidence PR `sunholo-data/ailang-packages#55` (closed as noise).
