# Independent sprint evaluation — PR #1849, round 2

**Verdict: FAIL, 68/100 (hard correctness failure).** Reviewed exact head `217f229391896d6da70736d051c66dcd175ee3e4`; worktree `pr-1849` was read only. The original round-1 FAIL51 artifact remains untouched. Fresh exact-head CI is pending; no whole suite was rerun for this evaluation.

## Blocking finding: malformed-lock authority bypass through CallPreserveFloats

`internal/embed/embed.go:272` (`Engine.CallPreserveFloats`) does not apply the new `e.pkgAssets.err` check installed at `Load` and `CallPrepared`. When `Engine.New` finds malformed `ailang.lock`, its target map is unavailable. `CallPreserveFloats` discards `compileModule`'s error and falls through to runtime module loading, then evaluates with no lock snapshot. `assetPath` consequently selects a newer cached version.

First-party isolated consumer repro on the reviewed head:

```text
Call: value=<nil> error=cannot load package lock file .../ailang.lock: ... unexpected end of JSON input
CallPreserveFloats: value=FALLBACK_NEWER error=<nil>
```

The fixture has a valid package manifest, malformed lock text `{`, one cached `test/data@0.4.0` asset containing `FALLBACK_NEWER`, and an FS-capable `asset_fixture.read` function. It invokes both public APIs on separate engines. This is an observed unsafe fallback, not a hypothetical missing check. MCP and A2A use `CallPreserveFloats` (`internal/apiserver/mcp.go`, `mcp_ui_serve.go`, `a2a.go`), making the alternate entry relevant to served consumers.

**Required repair:** apply the same early package-discovery error guard to `CallPreserveFloats`; add a regression that calls it against malformed manifest/lock and confirms no evaluation/fallback, plus a valid locked-byte call. Preserve float argument conversion and existing compile/load behavior outside this gate. Rerun focused embed/API asset tests and exact-head CI. The independent reviewer must recheck the repaired commit.

Repro source and captured output are in scratch: `pr1849-review-repro/main.go`, `pr1849-review-repro/go.mod`, `pr1849-review-repro.log`. Run `go run -mod=mod .` from the repro directory. Its Go version matches the reviewed repository (`1.26.10`); it writes only disposable fixture state.

## Verified residual scope

- Versioned `PkgAssets` takes precedence over legacy `PkgDirs`; unresolved/deleted targets retain tombstones and versions rather than falling back.
- `WithBudget` preserves both maps; Clone preserves the immutable references; tests cover original, budget and cloned contexts.
- Runner/batch consume `pipeline.Result.PkgAssets`; tests alter the on-disk lock after compilation and still assert locked bytes.
- Embed target maps are per-engine and context replacement clones caller state; first engine retains older targets while a second engine observes a changed lock.
- Root-package/no-lock authoring and configured `AILANG_PACKAGE_ROOT` behavior are retained. The two independently found doc/JSON contradictions were corrected in `217f2293`.
- Existing MAJOR.MINOR.PATCH and prerelease-before-release ordering remains; the old dev/unknown synthetic-version policy was not restored.
- Tests exercise registry/path/git targets, transitive tombstones, malformed discovery, actual asset bytes, context replacement, API repeated calls and both runner modes.
- PR conversation, inline comments and formal reviews were read. The earlier SUPERSEDED comment is unsupported by the residual code and is not adopted.

## Gates and score

Existing completed local logs show unfiltered `make test-changed` (8 packages plus full core), all six package suites, x20 asset tests, lint, boundaries/architecture closure, sizes, formatting, changelog, stdlib49 and example gates. They establish the repair's baseline validation; they did not cover the alternate public entry point above. Exact-head remote CI remains pending, so no full-suite pass is claimed.

| Category | Score | Reason |
|---|---:|---|
| Tests pass | 0/20 | Exact-head CI pending; independent malformed-lock repro fails |
| Lint | 10/10 | Completed local gate; documentation-only latest head |
| Acceptance | 24/30 | Most runtime/plumbing criteria met; public embed failure propagation incomplete |
| Code quality | 12/15 | Small layered changes; duplicated call paths missed shared authority gate |
| Documentation | 15/15 | Reconciled deployed behavior and preserved historical FAIL51 |
| Design fidelity | 7/10 | Intended map/tombstone architecture present; fail-loud promise violated on alternate call API |
| Total | **68/100** | **Hard fail regardless of score** |

Regression-surface analysis exists in the design, and positive/negative fixtures are substantive. No implementation files, sprint artifacts or design-doc locations were changed by this evaluator.
