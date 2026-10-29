# M-PKG-SMOKE-STAGED-LOCK executor validation

**Final disposition: accepted and merged, v0.56.0 UNRELEASED.** Independent
round2 PASS95/100 evaluated165f87a12ceb0e20c04216c472004c11685e417b.
Actual final CI27completed:20SUCCESS/7SKIPPED, no failed/cancelled/pending.
Root normal merge96d69f05b6e84bd6666e8fa91ff948f803b67d92 at20:53:24UTC.
[Both frozen evaluations and supporting proof](evidence/m-pkg-smoke-staged-lock/README.md)
are durable; the prior FAIL75/raw failure identities remain unchanged.
The local/executor and pending-CI statements below record earlier checkpoints;
they are not the final acceptance verdict. No new test execution is claimed by archival.


Implementation complete, unreleased. Independent evaluation and exact PR broad
Linux/Windows CI remain pending. This document records executor evidence, not a
score or release approval. Design and plan remain planned until root accepts the
independent review.

## Source and authorization

- User design approval recorded2026-10-29 11:57:17UTC; explicit execute instruction
  at12:23UTC accepted plan0c3e8f406. Existing issue1773/design carrier1775 retained.
- M1 commit19f2933ef; M2 f3499b71e; M3 f8e15bc2e; independent D1 repair
  dade1527b; M4 portable CLI/tests/docs37c957b3a742aba8c966c185f39d48d086d5e794.
- Normal actual-dev integrations:65253e86 (IFC/cachev8), aca5a6ead (npm/docs
  security), dba1c499 (completion audit, merge336605f05). No IFC/cache or other
  owner implementation was edited. The audit added zero Go/AIL diff against the
  immutable tested37c source. Git inferred a directory move for this new plan;
  merge resolution deliberately retained it in planned pending independent review.
- Final metadata/link changes preserve that immutable tested Go/AIL tree.

All raw logs below are retained under `/tmp/ailang-backlog-20261029/`; they are
session evidence, not durable external URLs. Test names below are reproducible
from committed source. Initial failed controls remain preserved.

## Acceptance map

| Acceptance | Committed proof | Evidence log |
|---|---|---|
| Published manifest + regenerated registry hashes, author inputs unchanged | `TestSmokeStage_RegeneratesRegistryLock`, `TestRewritePathDeps_UsesOriginalDirectoryAndIsIdempotent` | `1773-M1-red.log`, `1773-M1-checkpoint.log` |
| Exact same-version registry module/assets, staged root asset | `TestSmokeStage_PublishedSourceAndAssetBytes` | `1773-M1-checkpoint.log` |
| Actual CLI source discriminator, unavailable exact version/newer cache refusal | `TestPkgSmokeStaging_QualityAndDryRun` | `1773-M4-cli-byte-discriminator.log` |
| Confined and exclusive-root D2 pin, no git executable/HOME writes; missing/changed/tampered refusal | `TestSmokeStage_GitCarryForward` and existing `TestResolveGitConfined_CarryForwardMatrix`/package-root controls | `1773-M1-checkpoint.log`, final changed/core |
| Malformed/schema-invalid prior metadata, missing exact target fail closed | `TestSmokeStage_FailClosedPreparation`, existing package/asset snapshot tests | `1773-M1-checkpoint.log`, final changed/core |
| Stage rErr, first diagnostic before >8KiB, UTF8 byte budgets, no schema addition | `TestSmokeOutput_*`, `TestSmokeFailureNotes_*`, `TestPkgSmokeDiagnostics_*` | `1773-M2-red.log`, `1773-M2-focused.log`, final changed/core |
| Actual child signal retained, timeout priority preserved | `TestSmokeFailureNotes_ActualSignalCause`, competing signal/child-error formatter case | `1773-M2-signal-red.log`, `1773-M2-signal-green.log` |
| Human PUB015 and JSON Notes share first cause; no-run preserved | `TestPkgSmokeDiagnostics_RealQualityNotes` and existing no-run test | `1773-M4-offline-cli-final.log`, final changed/core |
| Warning observes actual registry/git source, unchanged lock/conflicts/direct path | `TestPkgLockShadowWarning_RealCLI`, `TestPkgLockShadowWarning_ActualSourcesAndReadOnly` | `1773-M3-red.log`, `1773-M3-green.log` |
| User docs, remedies, intentional staging/tarball divergence | publishing guide, validator comments, unreleased fragment | final structure checks |

### Discriminating offline CLI proof

The CLI-specific fixture requires REGISTRY_SOURCE + REGISTRY_SOURCE_ASSET and
STAGED_ROOT. A real direct run against the author-local pair **must fail exit23**;
actual quality JSON and publish --dry-run then pass with the registry pair.
Thus passing smoke is not merely agreement between two local values. Missing
exact0.1.0 with newer0.2.0 cached fails with the canonical package-root remedy.
Author manifest, lock and content snapshots remain unchanged. The loopback mock
counts actual registry reads, rejects every non-GET, and asserts zero writes.
Child HOME is private; no message/coordinator database opens occur. Z3 is
explicitly absent and the uncontracted fixture still passes. No live package was
published and no upload/cascade/coordinator action was invoked.

### Independent signal finding and repair

Independent preflight at immutable f3499b71e found an empty-output self-killed
child reduced to generic exit-1. Original independent RED is
`1773-independent-m1m2/signal-control.log`; its README describes the unchanged
process-function extraction and deliberately stubbed stage. The real complete
stage/process regression also failed before repair (`1773-M2-signal-red.log`).
D1 retains actual negative-exit signal text, while timeout remains first, and
recognizes signal: before competing child diagnostics. Complete actual-process
GREEN and independent extracted-function GREEN are preserved in
`1773-M2-signal-green.log` and
`1773-independent-m1m2/signal-control-fixed.log`. Only the signal-specific
regression skips Windows: TerminateProcess is an ordinary exit there. Common
empty-output/start/timeout/byte-budget controls still run on Windows; no Unix
signal claim is made for Windows.

## Completed local gates

Commands used GOFLAGS=-p=1 for package serialization; existing test deadlines,
quality thresholds and subprocess limits were unchanged.

- `make build`: PASS, `1773-M4-build.log`.
- `make test-changed`: PASS on immutable37c, comparing actual devaca5a6ead;
  cmd/ailang172.825s and internal/pkg19.353s plus every core package including
  types/effects/pipeline/runtime/iface. `1773-M4-test-changed.log`.
- `make lint check-boundaries check-home-isolation check-file-sizes check-changelog fmt-check`:
  PASS, lint0 and all files <=800lines. `1773-M4-structure.log`.
- Focused milestone RED/GREEN and D2/snapshot controls passed before checkpoint
  commits; no full local make test was run. No unrelated failure was hidden or
  retried in the final gate.

Portability uses filepath helpers, os.DevNull, the shared bounded CLI builder and
Go child helper. Actual broad Windows execution remains an exact PR CI gate.
The independent evaluator must inspect source/criteria and actual terminal CI;
queued/failed checks are not executor acceptance. No self-score is assigned.

## Estimate variance

The680-line estimate was exceeded by the required cross-process, D2, source/asset,
missing-target, Unicode/signal and warning regression matrix plus approval/state
and evidence records. Production scope remains the approved shared rewrite,
stage assembly, diagnostic propagation and warning. File-set divergence was
documented; tarball policy, authority, resolution order, JSON schema, timeout
policy and language semantics remain unchanged.

## Accepted dependency integration (2026-10-29)

Root confirmed operator PR2016 normally merged at
`565fc1b0461f1408d930df3029a72beef60f7ff1` (20:30:35UTC), accepting
fixture source2ce4b2f7d. Normal merge into this PR is
`cc2c09ca808f43d5eabbb8206c83775590fc9e60`, with no conflict. All33 previous
owned files were byte-identical immediately after merge, including all21 own
Go/AIL files, and the own-delta path set against accepted dev remained equal.
Incoming Go diff is only internal/runner/worker_terminal_pty_test.go; fleet
driver/docs are preserved. Subsequent edits record this evidence only.

Fresh make build stamped fullcc2 source; selective combined-head checks passed:
internal/pkg10.222s, cmd/ailang10.820s, internal/runner19.027s. Command:

```sh
GOFLAGS=-p=1 go test ./internal/pkg ./cmd/ailang ./internal/runner \
  -run '^(TestSmokeStage_|TestPkgSmokeStaging_|TestPkgSmokeDiagnostics_|TestPkgLockShadowWarning_|TestWorkerTerminalPTYLifecycle$)' \
  -count=1
```

Logs and identity proof are retained in session
1773-normal-integration-2016/{build.log,focused.log,identity.json}. No full suite,
count20/sibling rerun, new product repair, CI rerun, policy change or self-score.
Separate immutable2ce native11cell/count20/47sibling PASS remains in
1926-native-validation/report.json. New exact-head broad CI is authoritative
and independent evaluation remains required.

The prior independent **FAIL75** at0853 from two actual unchanged #1926
PID-readiness failures remains historical and unmodified. Passing package tests
did not waive that gate. The new accepted fixture and fresh-head CI must clear
the blocker; this integration does not retroactively rewrite those failures.
