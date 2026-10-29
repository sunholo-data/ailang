# Sprint Plan: M-PKG-SMOKE-STAGED-LOCK

## Summary and authorization

Make isolated package smoke test the published manifest and a regenerated registry
lock, preserving existing confined git-pin carry-forward, locked asset snapshots,
and warning-only root-path shadowing. Surface bounded, useful PUB015 diagnostics.

- Design: [existing approved design](../v0_54_7/m-pkg-smoke-staged-lock.md).
- Design approval: attended user explicitly approved the refreshed design on
  2026-10-29, relayed by root; approval was recorded at 11:57:17 UTC.
- This exact plan at `0c3e8f406` was **explicitly accepted for execution** by the
  attended user, relayed by root at2026-10-29 12:23UTC. `execution_authorized=true`. No
  coordinator task, message import/ack, cloud approval or executor dispatch is
  part of planning. Commit/push/merge of a coordinator carrier is not substituted
  for the attended execute instruction.
- Target: v0.56.0, unreleased. Original design location/target v0.54.7 stays as
  provenance; no version/tag, release or live package publication is authorized.
- Baseline: actual dev `749045717c484299b3103008023bf27a826df844` (#2008 merged).
  Planning branch normally integrated it at `82d04388e3a2160f180619aa1031b21d325af3c5`
  after banking approved design commit `e8ff97280`. Product source is unchanged.
- Duration: two working days, four bounded executor sessions (about 5–6 active
  hours plus verification/review headroom). Budget 680 changed lines including
  tests/docs; moved rewrite code is counted conservatively. Risk: medium.

## Current status and planning basis

Execution started under the recorded approval; milestone acceptance remains
unpassed until verified. The initial baseline `RunSmokeInTempDir` copies the
relative author lock; quality discards process output and staging `rErr`. Root
path B can be shadowed by transitive registry B without a warning. Existing host
APIs already supply D2 carry-forward and per-compilation asset targets.

Read-only verification on actual dev 80793 is banked in session
`1773-readonly-repro/{results.json,extra-results.json,*.log}`: local source+asset
markers pass, copied stage fails missing sibling, actual quality has compile=true
and PUB015-only/no Notes, manually regenerated stage uses distinct registry
source+asset bytes, and root path shadowing remains silent. `--no-run` remains
not-run. These are premise checks, not completed implementation acceptance tests.

Recent seven-day scoped git history has nine non-merge package commits,
2,184 insertions and 198 deletions. This includes parallel agents and cannot be
used as single-agent daily throughput. The recent alias repair banked 246 lines
of regression companions around a 15-line production diff; verification, not
raw edits, dominated its session. The legacy velocity script exits 141 because
its `pipefail` log pipeline feeds `head`; scoped read-only git statistics were
used instead, with no script repair. Planning assumption: 340 changed lines/day,
680 total over two days, with the compatibility/verification work budgeted in
advance. This is an estimate, not a measured throughput guarantee.

## Approved rulings and boundaries

There are **no open human semantic decisions** inside the approved scope:

1. Test published registry versions, not staged sibling source. Rebuild registry
   hashes from the rewritten staged manifest; never relabel local lock entries.
2. Preserve approved D2: validate previous lock before removal, then use existing
   `ResolveLockFrom` for unchanged provisioned git pins. Keep missing, changed,
   malformed and tampered-pin refusal. No new authority path or changed precedence.
3. Preserve #1849 snapshots: root assets from the stage, dependency code/assets
   from the same resolved directory, missing-target tombstones and exclusive
   PACKAGE_ROOT. Do not add a second asset resolver.
4. Failed assembly/run remains failed smoke/PUB015 with existing Notes/schema;
   `NotRun` remains reserved for deliberate no-run. Do not change exit policy.
5. Root path shadowing adds stderr warning after successful normal lock write;
   source/version/order/lock data and exit codes remain unchanged. Optional new
   `lock --check` warning is deferred; existing drift semantics stay unchanged.
6. Design Phase4 defaults to documenting current staging/tarball differences.
   Do not refactor or expand the published tarball file set.

New authority grants, root-path-wins resolution, validator rewrite-to-latest
changes or a need to widen this scope: **STOP**, commit only approved work,
report `BLOCKED: <specific new scope/decision>`, and end the run. Do not implement
an alternative while waiting for a ruling. This is a scope guard, not a reopened
human decision. Keep other agents' work and root's completion-audit branch intact.

## Registry reuse and architecture

All four milestones are host Go tooling/tests/docs, not package-like AILANG
runtime capabilities. External registry search is not applicable or authorized
for planning. The reuse audit is `none` for each milestone: reuse existing
`internal/pkg` resolver/lock/authority/loader and quality APIs instead. Mock and
preprovisioned registries are test fixtures, not new packages to publish.

One necessary API refinement preserves current output behavior: the extracted
rewrite helper is **silent** and returns rewrite facts (e.g. an internal Go
`[]PathDepRewrite` result). The existing `rewritePathDepsForPublish` wrapper
converts that to its bool result and renders publish progress. Moving its
current `fmt.Printf` into staging would corrupt `pkg quality --json`; assert
that JSON remains parseable. This is not a new JSON field or policy choice.

## M1_STAGED_PUBLISHED_VIEW — 260 lines, 75–90 minutes

**Goal:** one staged manifest/lock view with D2 and source/asset consistency.
Dependencies: none after explicit execute authorization.

Files: extract existing rewrite into `internal/pkg/publish_rewrite.go`; staging
assembly in `internal/pkg/publish_stage.go`; integrate
`internal/pkg/publish_validator.go`; retain thin render wrapper in
`cmd/ailang/pkg_publish.go`. Tests in new companions
`publish_rewrite_test.go`, `publish_stage_test.go`, and
`cmd/ailang/pkg_smoke_staging_test.go`, reusing existing smoke/registry fixtures.
Estimate: 100 implementation/moved lines + 160 test/fixture lines.

- First capture failing dependency-aware smoke and silent JSON regressions.
- Resolve original path declarations against original absolute packageDir, not
  the temp dir. Rewrite staged manifest to exact sibling-declared version.
- Read/validate copied prior lock, distinguish genuinely missing from unreadable,
  malformed or schema-invalid input, remove it, invoke current
  `ResolveLockFrom(stagedManifest,stage,generator,validatedPrevious)`, then Save
  only the new lock. Do not launch smoke after any preparation failure.
- Previous path entries never become the execution lock. Previous git metadata
  is only the existing matched URL/tag/rev/subdir/hash/interface carry-forward;
  missing/changed/tampered pins retain canonical refusal. No git/HTTP/home bypass.

Acceptance:

- [x] Same-version local B and registry B have distinct module **and asset** bytes.
  Direct real `RunSmokeInTempDir`, real `pkg quality --json` and real
  `publish --dry-run` select registry markers. Helper inspection proves registry
  source and registry content/interface hashes in the newly saved staged lock.
- [x] Staged self/root assets resolve inside stage; dependency code+assets use
  one target; another cached version cannot rescue a missing locked target.
- [x] Author manifest/lock/content snapshots are unchanged after every path,
  including publish's existing save/restore; rewriting already-registry manifest
  is idempotent, and whitespace variants from existing publish tests remain valid.
- [x] Valid prior git pins under both confinement and PACKAGE_ROOT pass without
  git/HTTP/home writes. Nil, malformed, changed and tampered prior cases refuse;
  use existing D2 fixtures and no-git PATH instrumentation, not shell shortcuts.
  These assertions cover stage assembly/resolution; the child's existing runtime
  capability/FS-sandbox policy still applies and may refuse execution as before.
- [x] Registry missing-version, missing-provision and forbidden-fetch cases fail
  before child execution and preserve actual authority/remedy, without raw-lock
  fallback. No-dependency/registry-only controls remain valid.
- [x] Rewritten quality JSON parses; publish progress still belongs to the CLI.

Checkpoint: targeted `go test ./internal/pkg ./cmd/ailang -run
'TestRewritePathDeps|TestSmokeStage|TestRunSmokeInTempDir|TestPkgSmokeStaging|TestResolveGitConfined|TestGitPackageRoot' -count=1`, lint and boundaries. Rebuild
with `make build` before tests that discover the real binary; pair its own stdlib.

M1 checkpoint: PASS (2026-10-29). Real registry source/asset markers and author
input snapshots pass; D2 stage carry-forward/malformed/missing controls pass;
quality JSON and dry-run have no upload POST. Existing smoke/whitespace and
authority controls pass without skips; lint0/boundaries/sizes/fmt pass. Logs
`1773-M1-*` retain failing-first and final real-binary evidence.

## M2_BOUNDED_SMOKE_DIAGNOSTICS — 205 lines, 60–75 minutes

**Goal:** retain the cause of failed staging/process execution in <=2048-byte
Notes, including before the existing 8KiB tail-only truncation loses it.
Dependencies: M1.

Files: `internal/pkg/publish_output.go` and `_test.go` companions;
`publish_validator.go`, `internal/pkg/quality.go`,
`cmd/ailang/pkg_quality.go`, and new
`cmd/ailang/pkg_smoke_diagnostics_test.go`. No new JSON schema or unbounded
production trace store. Estimate: 75 implementation + 130 test lines.

Concrete diagnostic selection contract (applied to full captured output before
lossy truncation; shared by 8KiB result and 2KiB Notes summarization):

1. For failure summarization, normalize CRLF/CR and ANSI decoration; replace malformed UTF-8 with a
   printable replacement before byte budgeting. Preserve raw failure logs in the
   test/session evidence; sanitizing Notes does not claim output was absent.
2. Infrastructure `rErr` takes precedence, with its first meaningful error as
   cause even when no SmokeResult exists. Non-ExitError child-start failure must
   retain the actual executable/start error, not empty output/only exit -1.
3. Timeout prefixes `timed out after <requested duration>` and may append the
   selected captured diagnostic. Timeout is never `NotRun`; keep current timer
   placement (resolution outside subprocess timeout) and proctree cleanup.
4. Otherwise scan lines in order for a trimmed error diagnostic: case-insensitive
   `Error:`, `panic:`, `fatal error:`, `FAIL:` or a line beginning a `PAR_` parser
   diagnostic. Prefer these over earlier progress/warning/stdout noise. Stack
   frames (`at ...`, `goroutine ...`, indented Go file:line) are context, not the
   diagnostic. If none exists, select first nonblank non-warning/progress/
   INFO/DEBUG/TRACE/stack line; if none, state the nonzero exit/signal cause.
5. Put that cause first, optional bounded final context afterward. UTF-8-safe
   truncation reserves space for a truncation marker and all prefixes/newlines.
   If one diagnostic exceeds the cap, keep a bounded prefix plus marker. Total
   SmokeResult output <=8192 bytes; total failed Notes <=2048 bytes, not runes.
   Successful short smoke output remains unchanged. Reuse existing
   `internal/strutil.TruncateBytes` after `strings.ToValidUTF8`, reserving all
   prefixes/markers first and omitting optional context when its budget is too
   small for the marker; do not duplicate or change strutil truncation policy.
6. Human PUB015 uses Notes' same first bounded cause, JSON exposes existing
   smoke.notes, and `NotRun`/absent-smoke/attested-upload schema stay unchanged.

Acceptance:

- [x] Stage copy/rewrite/lock `rErr`, refused resolution, actual failing child,
  child-start failure, signal/timeout and empty output each retain a useful cause.
- [x] Leading progress/warnings and following trace frames do not displace Error;
  a real diagnostic before >8KiB trailing output is retained. Long non-ASCII,
  malformed UTF-8 and a >2048-byte single error line respect exact byte caps and
  parseable JSON. Timeout marker fits inside cap, too.
- [x] Human and JSON PUB015 agree on first cause; no new report field/code and
  no lost failure through `rErr != nil`. No-run still badges rather than fails.
- [x] Existing fail/pass/timeout/isolation controls stay valid; no CI timeout,
  output-gate or authority policy is relaxed.

Checkpoint: targeted pkg/CLI diagnostic and quality/no-run tests, `gofmt`/lint.
No repeated full packages already green without an affected change/failure.

M2 checkpoint: PASS. Red/green cause retention, exact byte bounds and UTF-8,
setup/start/parser/noise/trace/timeout cases and actual quality/no-run controls
pass; lint0/boundaries/sizes/fmt pass. Actual merged #1771 dev652 was normally
integrated before this milestone; its cache/IFC source was not edited. Raw
`1773-M2-*` evidence retains the original failures.

## M3_ROOT_PATH_SHADOW_WARNING — 105 lines, 45–60 minutes

**Goal:** make discarded root path intent visible without changing resolution.
Dependencies: M1; execute sequentially to avoid overlapping package CLI edits.

Files: new `cmd/ailang/pkg_lock_warnings.go` and `_test.go`; add one call after
successful existing resolve/save in `pkg_commands.go`. Estimate: 30 implementation
+ 75 tests. Existing file is 669 lines; companion keeps it well under 800.

Acceptance:

- [x] Consumer root path A/B, A registry-pins same-version B: actual CLI lock exits
  0, writes identical resolved B source/version/hash/path data, then warns on
  stderr that local B was not selected. Version conflicts still fail unchanged.
- [x] Actual non-path source appears in warning (registry or git); no false
  registry wording for git. Direct path-selected B and unrelated deps do not warn.
- [x] No re-resolution, network request, new lock write, precedence/visitation or
  exit-code change is introduced by warning. `--check` retains existing drift
  behavior; this optional extra warning is deferred.

Checkpoint: focused warning/lock/path-resolution tests and lint; compare lock
semantics excluding existing generation metadata, not a new timestamp policy.

M3 checkpoint: PASS. Actual CLI warns only for discarded local path intent;
lock semantics, conflict handling and unsuccessful-write behavior are unchanged.
Actual registry/git names and nonmutation controls pass; CLI22.586s and
lint0/boundaries/sizes/fmt pass. Logs `1773-M3-*` preserve red/green evidence.

## M4_INTEGRATION_DOCS_HANDOFF — 110 lines, 60–90 minutes

**Goal:** bank real offline CLI proof, focused gates and reviewable documentation.
Dependencies: M1, M2, M3. Estimate: 50 harness/assertion + 60 docs/evidence lines.

Files: final `pkg_smoke_staging_test.go` integration cases;
`docs/docs/guides/package-publishing.md`,
`changelogs/unreleased/YYYY-MM-DD-pkg-smoke-staged-lock.md` (starts `### `),
this plan, existing design and sprint JSON. Document staging/tarball divergence
(lock regenerated for smoke; not shipped; existing CHANGELOG difference), with no
shared-predicate/tarball refactor. Preserve unrelated metadata and code.

Acceptance/gates:

- [ ] One portable Go harness uses a preprovisioned PACKAGE_ROOT and/or httptest
  mirror with distinct same-version local/registry code+asset markers. Invoke
  actual built CLI quality(JSON/human) and publish **--dry-run** plus direct real
  RunSmoke. Handlers count reads and reject/count every publish POST; assert no
  upload/cascade/message path occurred. No live registry endpoint or user HOME
  state is used. Quality's registry-overlap metadata uses mock index, not a
  hidden external request. Author inputs remain unchanged.
- [ ] Authority boundary and existing D2/hash/snapshot controls pass; no privileged
  client/git constructor is added. Missing locked target and malformed-lock
  failures are retained. Evidence maps every acceptance item to a test/log.
- [ ] `make test-changed` (changed packages + core), `make lint`,
  `make check-boundaries`, `make check-home-isolation`,
  `make check-file-sizes`, `make check-changelog` and `make fmt-check` pass. `make build`
  precedes binary-discovery cases. **Do not run full make test locally**; exact PR
  CI supplies full Linux/Windows/broad gates. Keep existing timeouts, quality
  thresholds/baselines and failure evidence; no force/admin bypass.
- [ ] Windows portability: use filepath helpers, forward-slash persisted lock
  paths, os.DevNull and existing bounded Go subprocess helpers, no POSIX-only
  fixture executable or exact OS-error wording. Reuse shared buildAilang once;
  existing 120s binary build and 60s per-CLI subprocess bounds remain. Smoke's
  default 30s/manifest 1–300s and short existing timeout test remain unchanged.
- [ ] Documentation describes published-view behavior, exact-version unavailable
  and canonical authority remedies, bounded PUB015 Notes and warning-only shadow.
  No release, live consumer source edit or publication is claimed.
- [ ] Implementation executor supplies source/commit/CI/evidence to an independent
  sprint-evaluator. No executor self-score; failed/queued checks remain explicit.

After explicit execute, preserve bot-auth/account rules before any separately
authorized push. Planning creates no PR or coordinator carrier. Root owns review,
merge and any issue disposition; use Refs #1773 during milestone work.

## Fixtures, syntax and example coverage

AILANG prompt version loaded: **v0.16.11**, full embedded prompt loaded during
planning; `prompt --version-active` confirms it. Executor reloads current prompt
before writing `.ail` fixtures. Proposed private examples:
`internal/pkg/testdata/smoke_staged_lock/{author_a,local_b,registry_b}/` with
manifests, model.ail, _smoke.ail and assets/marker.txt; use Go literals instead if
that better reuses existing fixture helpers without duplicated source.

| Fixture/demo | Contracts | Effects | Inline tests |
|--------------|-----------|---------|--------------|
| Author A smoke reading B and own staged assets | skip: cross-process staging/authority contract asserted by Go harness, not function SMT | include: main -> () ! {IO,FS}, assetPath/readFile/println and fail on wrong bytes | skip: Go harness asserts actual CLI outcomes, lock bytes and stderr |
| Local/registry B marker modules | skip: deliberately different source fixtures, Go asserts exact returned marker | include: exported pure marker -> string | skip: Go source+asset equality/discrimination assertions are the test |
| Failure/output/timeout fixtures | skip: intentional error/timeout cases are external observation tests | include: explicit IO/FS/Clock as each case requires | skip: enclosing Go tests assert failure/byte bound/deadline |

No standalone examples/runnable entry is added: a faithful demo requires a
controlled two-package registry/layout and belongs in the offline integration
fixture. The package-publishing guide provides the user-facing walkthrough.

## Execution schedule, risks and resumption

Day 1: M1 then M2. Day 2: M3 then M4, independent evaluator handoff. Each session is
bounded as above; checkpoint/commit at each milestone, do not widen a blocked
stage. Reuse the banked read-only fixture as a starting point, not as proof of
new implementation. Inventory ownership again before execution; no coordinator
import/ack/send is needed.

Before execution, normally integrate actual dev when clean; preserve planning
commits first. Root's separate completion audit `da3e5d54c` is not in this
baseline and is root-owned. Current design links to the landed asset design at
planned/v0_54_7 remain valid; after the actual audit merge, update only these own
links to implemented/v0_56_0. Do not edit or cherry-pick the audit branch.

Main risks: accidental raw path-lock reuse; dropping D2 provenance; helper stdout
corrupting JSON; first diagnostic lost before summarizing; timeout/UTF-8 bounds;
flaky shell fixtures on Windows. The explicit tests above instrument each risk.
No parser/type/effect semantics, new registry service, tarball file policy,
resolver precedence or broad cleanup is part of this sprint.
