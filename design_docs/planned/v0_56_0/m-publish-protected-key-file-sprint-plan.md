# Sprint plan: M-PUBLISH-PROTECTED-KEY-FILE

## Summary and authorization

Implement approved #1900 D1–D7: explicit publisher key-file selection, validated
private macOS/Linux handle, unsupported-platform refusal, no credential fallback,
no secret access for help/dry-run, and unchanged #1964 authority ordering.

- Design: [M-PUBLISH-PROTECTED-KEY-FILE](m-publish-protected-key-file.md).
- User explicitly said **"approve the three designs"**. Root bound this to exact
  design commit `7d000322b5b3b5d81fb88ae70213012a1f79a29e`, D1–D7, in session
  `three-design-approvals.json`, observed 2026-10-29 14:48:58UTC. This is the
  assistant's observation time, not an independently supplied send timestamp.
- **Execution authorized, M0 dependency unmet.** The user said "exeute all three
  sprints", observed2026-10-29 20:17:42UTC, with presented sequencing. Exact
  approved plan/JSON drafts were banked at c9698d88e before metadata updates.
  Product edits remain forbidden until legitimate accepted #2013 integration;
  no further D1–D7 or execute approval is needed once that condition is met.
- Target v0.56.0, unreleased. Duration 3 working days, five bounded implementation
  sessions plus an initial dependency check and final independent/CI handoff.
- Estimate 650 changed product/test/user-doc lines (160 production  + 410 tests  + 80
  user docs); planning/state/approval artifacts excluded from this capacity sum.
  Risk medium-high because credential handling requires security-specific proof.
- Baseline dba1c499ca6f76b40992fd521e3dd440d9277f49, isolated worktree
  publish-key-file-1900, branch docs/publish-key-file-1900. Approved drafts are
  now banked cleanly; no new base is merged while the dependency remains unmet.

## Current status and dependency

The original planning baseline had only design approval. Plan/execution approval
is now recorded above; implementation has not started because M0 is unmet.
Current publishing
is env-only. The existing AI key-file helper checks formatting/stat size but not
owner/mode/no-follow, so it cannot directly satisfy this feature. Preserve its
separate semantics. No registry package implements this host credential boundary.

PR #2013/#1773 still owns overlapping pkg_publish.go staging/rewrite/quality work.
Last confirmed source0853c9fc15f3daf5f9e304e6e451e78c21fd899c is OPEN/unmerged,
with its independent gate failed solely by repeated unchanged #1926 runner race.
The user's subsequent "merged" referred to #1926 plan-only merge
93e4f2586b2b25e7b124cdbd43e8f2ed534239fd at14:49:37UTC; no actual runner fix or
#2013 source merge was established. Do not infer that a plan merge resolves CI.

M0 must prove legitimate #2013 integration before any product edit. Normal
integration after actual accepted source merge is preferred; an explicitly
root-reviewed combined scope is the only alternative permitted by the approved
design. No combined implementation approval exists at planning time. There are
**no open semantic/human choices among D1–D7**; do not reopen them. The remaining
integration dependency is an external-state gate, not a request to bypass it.

**Terminal behavior:** if the dependency is unmet, STOP: bank only already
approved current work, report `BLOCKED: #2013 publishing integration dependency
is not legitimately met`, and end the run. Do not repair #1926, retry CI, merge
#2013, dispatch another owner or implement on a conflicting old publish path.

## Approved rulings to preserve

1. Explicit --key-file only: no default location, new file-path env variable or
   secret-provider fallback. Track empty explicit flag separately from absence.
2. Real publish rejects nonempty file+key-env ambiguity and missing credentials;
   unsafe/invalid selected file never falls back to env/unauthenticated upload.
3. macOS/Linux: one read-only no-follow/nonblocking/close-on-exec handle,
   handle-based regular-file/effective-owner/mode 0400-or0600 validation, bounded
   16KiB stream, one nonempty trimmed line, no CR/LF/NUL/header-control bytes.
4. Windows/other unsupported platforms refuse the file lane before opening it.
   Do not invent an ACL proof or weaken protection to portable mode bits.
5. Real publish validates once before package checks/children; help and dry-run
   never load/verify the credential file. Existing confined dry-run stays refused.
6. Legacy env-only publishing remains with one non-secret inheritance warning;
   do not scrub process-global env or place the file key in any env/argv/global.
7. Publish only. Unpublish/key administration, AI readers, transport/redirect
   policy, registry schema, language semantics and account/UID deployment unchanged.

Existing pkg.Require and dispatch/transport policy run before credential access.
A key path never creates authority under confinement. File permissions reduce
inheritance/cross-user disclosure; they do not isolate malicious same-UID code.
Parent components remain operator-managed; only the opened final handle is the
protected-file promise. No chmod/chown/creation or interactive repair.

## Capacity and registry reuse

The last seven-day scoped package/publish history on this baseline has 7 non-merge
commits, 1443 insertions/171 deletions across 24 changed-file rows. It includes parallel
authority implementation/review repairs, so it is not a single-agent throughput
measurement. #1773's large regression matrix also exceeded its initial estimate;
this sprint budgets tests explicitly. Existing velocity script exited 141 in its
pipefail git-log/head pipeline; scoped statistics were used without script edits.
Logs1900-planner-velocity.log and1900-planner-scoped-velocity.txt retain evidence.
Capacity assumption: 220 changed lines/day, 3 days, 650 planned lines, with review/CI
headroom outside five roughly 45–90 minute implementation sessions.

Every milestone is host Go credential/CLI/test/documentation work, not an AILANG
package-like capability. External registry search is not applicable/authorized.
Registry reuse decision **none** per milestone: reuse pkg.Require/HTTPClient,
existing command dispatch, bounded test/process helpers, loopback registry setup,
platform syscall conventions and original publish code. Introduce no package
or secret-service dependency. JSON records each decision and its reason.

## Milestones

### M0_INTEGRATION_PREFLIGHT — 0 lines, 15–30 minutes

**Dependencies:** explicit execute instruction and legitimate #2013 source
integration as above. This milestone makes no product change.

- Recheck clean status/current owner; preserve every other worktree and draft.
- Verify actual #2013 merge/acceptance and resulting source SHA using gh/source
  evidence, or the exact root-reviewed combined integration authorization.
- Once approved planning work is banked and tree clean, normally integrate actual
  accepted dev. Read the final publish path, #1773 validation and #1964 guards.
  Do not infer source acceptance solely from ancestry or plan/card status.
- Record exact dependency result/source in own JSON, build baseline for relevant
  CLI proofs. If unmet, apply terminal STOP above before any source edit.

Acceptance:

- [ ] Actual accepted dependency source and owner recorded; no competing overwrite.
- [ ] Baseline retains #1773 shared rewrite/staging/D2/Notes/author restoration and
  #1964 authority-before-work; no runner/card/CI policy edits.

### M1_PROTECTED_HANDLE_READER — 210 lines, 60–90 minutes

**Dependencies:** M0. Budget 80 production  + 130 tests.
Files: new internal/pkg/publish_keyfile.go, publish_keyfile_unix.go,
publish_keyfile_other.go and focused test companions. Keep files below 800 lines.
Do not change internal/pkg/authority.go or AI factory readers.

- Failing-first protected-read tests, then a publishing-specific reader whose
  first operation is existing pkg.Require. Use existing syscall conventions;
  macOS/Linux open no-follow/nonblock/read-only/close-on-exec, then fstat the
  handle. Owner=current effective UID, regular file, exact0400/0600 modes only.
- Bounded16KiB+1 read from that same validated handle, close it before return.
  Reject missing/symlink/FIFO/socket/device/directory/foreign-owner/unsafe modes.
  No separate Stat+ReadFile security decision or privileged ownership changes.
- Unsupported platform companion returns named refusal before open; no fake ACL.

Acceptance:

- [ ] Real disposable0400/0600 files pass; 0644/0640/0700/special modes fail.
- [ ] Symlink/nonregular/missing paths fail; FIFO refusal is bounded; actual
  handle metadata positive plus synthetic foreign-owner negative validate owner
  policy without requiring root/chown. These synthetic checks are classified.
- [ ] Same opened-handle identity survives pathname replacement; bounded reader
  rejects growth beyond 16 KiB without an unbounded allocation or hang.
- [ ] Direct reader under confinement has zero opener calls; unsupported OS has
  zero opener calls and named refusal. No key values in any local read error.
- [ ] Focused affected tests, lint and home isolation pass; preserve red evidence.

### M2_CREDENTIAL_SELECTION_AND_FORMAT — 100 lines, 45–60 minutes

**Dependencies:** M1. Budget 30 production  + 70 tests.
Files: narrow publisher credential selection/format helpers in internal/pkg or a
cmd/ailang/pkg_publish_auth.go companion, plus unit tests. Avoid a public generic
credential framework; choose the smallest API needed for invocation-local value.

- Freeze exactly one approved source/value once. Explicit file+nonempty key-env
  refuses before read; explicit empty path refuses; absent sources refuse real
  publish. No default file probe and no later config.RegistryAPIKey reread.
- Trim surrounding whitespace; require single nonempty line, reject embedded
  CR/LF/NUL/control bytes and oversize streamed input. Error text never contains
  synthetic secret or prefix. Relative path uses original invocation cwd.
- Ensure help/dry-run mode bypasses credential selection/file inspection; do not
  turn unused credentials into an offline prerequisite.

Acceptance:

- [ ] Complete approved selection table passes, including file error+valid env
  counterfactual, explicit empty flag, no-source and no implicit default probe.
- [ ] Empty/whitespace/multiline/control/oversize synthetic contents fail; trimmed
  single-line dummy key accepted. No fallback on any read/parse failure.
- [ ] Frozen result survives path/env mutation after selection; parent env remains
  unchanged; nil/default key is never treated as successful credential selection.
- [ ] Authority/unsupported refusal precedes file opener; focused tests and lint pass.

### M3_PUBLISH_COMMAND_WIRING — 130 lines, 60–75 minutes

**Dependencies:** M2. Budget 50 production  + 80 tests.
Files: cmd/ailang/pkg_publish.go (currently 624 lines on old dev, smaller after
#2013), new pkg_publish_auth.go and pkg_publish_keyfile_test.go companions as
needed. Do not copy the old rewrite body into the integrated #1773 implementation.

- Add local --key-file flag with presence tracking and help. Both bare/pkg routes
  use same function. Retain dispatch and inner authority-before-read.
- Real publish selects credential before validation/test/smoke subprocesses, then
  passes value explicitly to upload header construction. Remove ambient reread
  from uploadTarball; update all callsites rather than adding hidden fallback.
- File value never reaches env/argv/log/attestation/tarball/message. Env-only real
  lane warns once on stderr, without value or prefix; env remains unchanged.
- Selected-key exact reflection in validator error is redacted before rendering
  existing error cause; no general logging/transport refactor.

Acceptance:

- [ ] Actual built CLI both publish spellings recognize the option; help and
  dry-run with missing/unreadable/unsupported file do not open/read it or upload.
- [ ] Confined bare/pkg real/dry-run refuse before read; help remains available.
- [ ] Real credential errors precede package subprocess/upload; env-only warns
  exactly once, file-only does not create env warning or key variable.
- [ ] Loopback header/status unit controls retain X-API-Key/validation behavior,
  reject missing key before request and redact exact dummy-key response echoes.
- [ ] Existing #1773 and authority-focused tests plus lint remain green.

### M4_REAL_CHILD_AND_LOOPBACK_PROOF — 130 lines, 60–90 minutes

**Dependencies:** M3. Budget 0 production  + 130 tests.
Files: cmd/ailang/pkg_publish_keyfile_cli_test.go; private testdata helper/fixtures
only where needed, reusing #1773 staged fixture harness and shared buildAilang.
No production test hooks, broad test-main changes, external programs or live keys.

- Actual built publisher runs synthetic package with smoke that invokes a
  compiled Go env-probe helper through existing unconfined Process effect.
  Helper checks **raw os.LookupEnv**, emits only key-present/key-absent sentinel
  and never a key. File-only arm must observe absent; legacy dummy-env arm must
  observe present using the identical helper, proving instrument is not blind.
  Do not accept Env.hasEnv=false alone as proof: an Env allow-list can hide a
  present variable. Existing smoke grants Process; no policy bypass/flag change.
- Loopback registry/validator counts actual requests. Validator asserts exact
  synthetic file-key header then returns a controlled failure (e.g.403), stopping
  before post-publish messaging/cascade. This is an intentionally failed mock
  upload, not successful live publication. Positive control proves server active.
- Package smoke can replace the dummy file after selection; received header must
  still contain frozen original value. Private HOME, explicit local mock registry,
  scrubbed telemetry/cloud settings, no real collaboration/coordinator store.
  Reuse clearMessagesEnv and testutil.SetHomeDir, then a private test-only
  isolation helper clears GOOGLE_CLOUD_PROJECT, OTLP_GOOGLE_CLOUD_PROJECT,
  AILANG_CLOUD_PROJECT, GOOGLE_APPLICATION_CREDENTIALS, inherited OTEL_* and
  AILANG_TRACE* plus GEMINI_TELEMETRY_TARGET/AILANG_OTLP_INGEST_TOKEN; retain
  AILANG_NO_METADATA=1 and no operator config. AILANG_TRACE=off alone does not
  disable the held #2009 root-span/export behavior. Do not change production
  tracing or duplicate that independently owned issue.
- Inspect author snapshots/tarball/attestation/captured errors for dummy key.
  Retain #1773 registry module+asset discrimination and D2/snapshot controls.

Acceptance:

- [ ] File-only actual child sentinel absent vs legacy env sentinel present;
  counted loopback sees the correct distinct dummy header for both arms.
- [ ] Read-once/path replacement control uses original value; parent env and
  author manifest/lock/content unchanged, no secret in captured artifacts.
- [ ] Missing/conflicting/unsafe file yields zero upload; help/dry-run zero file
  open and zero upload; loopback controlled failure emits no package messages.
- [ ] Portable helper uses bounded Go subprocess support; Windows actually tests
  file-lane unsupported and existing env/help/dry-run, not a blanket skip.
- [ ] Focused real CLI/authority/#1773 regressions and lint pass, raw reds retained.

### M5_DOCS_GATES_AND_HANDOFF — 80 lines, 60–90 minutes

**Dependencies:** M4. Budget 0 production +0 tests  + 80 user-doc lines.
Files: publishing guide, CLI reference/help source as needed, config legacy-env
reference text if changed, fragment changelogs/unreleased/YYYY-MM-DD-publish-key-file.md
opening with `### `, and own plan/state/validation records. No root audit/other plans.

- Document exact invocation, no startup export/default/fallback,0400/0600/private
  parent/same-UID limitation, Windows unsupported, dry-run does not verify auth,
  legacy env inheritance warning, publish-only flag scope.
- Verify help/reference agreement and actual examples with dummy-only offline
  fixtures. No real credential creation/read/upload or release claim.
- Final make build and make test-changed (changed packages +core), lint,
  check-boundaries, check-home-isolation, check-file-sizes, check-changelog,
  fmt-check. No full local make test; exact-head CI supplies broad Linux/Windows.
- Bank per-milestone commits/evidence after execution approval. Final authorized
  handoff goes to independent sprint-evaluator; no executor self-score or merge.

Acceptance:

- [ ] User docs/help and unreleased fragment accurately describe approved D1–D7.
- [ ] Focused/changed/core/structural gates pass with unchanged thresholds/timeouts;
  no retry/waiver hides unrelated gate failures. Pending CI remains explicit.
- [ ] Design AC1–AC8 mapped to committed tests and actual logs; exact integrated
  source provenance, red/green and platform qualifications retained.
- [ ] Independent evaluation handoff uses terminal real gates; no self-approval,
  live publish, issue closure, coordinator action or release during execution.

## Schedule and totals

| Session | Production | Tests | User docs | Total |
|---|---:|---:|---:|---:|
| M0 integration preflight | 0 | 0 | 0 | 0 |
| M1 protected handle | 80 | 130 | 0 | 210 |
| M2 source/format selection | 30 | 70 | 0 | 100 |
| M3 publish integration | 50 | 80 | 0 | 130 |
| M4 actual child/loopback proof | 0 | 130 | 0 | 130 |
| M5 docs/gates/handoff | 0 | 0 | 80 | 80 |
| **Total** | **160** | **410** | **80** | **650** |

Day 1: M0 dependency check, then M1/M2. Day 2: M3/M4. Day 3: M5/final focused
verification, review/CI headroom. A failed external M0 ends the run; no product
work begins simply to occupy the session. Each session fits a bounded executor
run; if required proof grows, split sessions without widening approved semantics.

## Prompt, examples and measurement limits

AILANG prompt version loaded **v0.16.11**, full embedded prompt saved as
1900-planner-prompt.md; active version queried. Executor reloads before writing
any `.ail` fixture. Existing std/process.ail and effect authorizer confirm
unconfined no-allowlist PATH lookup; helper path is a compiled test binary.

| Fixture | Contracts | Effects | Inline tests |
|---|---|---|---|
| Private package smoke invoking raw-env Go probe | skip: host process/env inheritance is observed by Go harness, not SMT | include: main -> () ! {Process,IO}, explicit Result handling; manifest ceiling matches | skip: actual process/loopback sentinels and headers asserted by Go integration |
| Pure package marker module | skip: not a new language contract, only publish fixture | include: pure string marker, empty row | skip: no standalone semantic change; outer Go asserts package gates/source identity |
| User-facing invocation example | skip: CLI operation, not AILANG function | skip: no AILANG effect body | skip: real offline CLI acceptance covers invocation |

No standalone runnable example needs real credentials: the documented invocation
plus private synthetic package is the example. All test secrets are disposable
sentinels; never open actual ~/.config/ailang/registry.key or external auth stores.
The M4 test must distinguish raw absence from policy-redacted access and must not
turn its Go probe into a credential printer or a generic command adapter.

## Execution checkpoint: M0 dependency unmet

Explicit execution was authorized by the attended user at observed
2026-10-29 20:17:42UTC, recorded in session three-sprint-execution-approval.json.
This binds the original approved plan SHA256
`1190cc269d0865e069cce1ba802928f0275381c752432b56078d41bbaaee1145`
and JSON SHA256
`3f06bbf28d2236c6e583da4b1a139b4e626dbd0c0dad1f0dd52aa4748b9eed9e`,
both preserved at commit c9698d88e. Observation timestamp qualification remains
as in the approval record; no independent send-time claim.

At 2026-10-29T20:20:37.463547+00:00, actual read-only gh PR checks show #2013
OPEN at0853c9fc15f3daf5f9e304e6e451e78c21fd899c and #2016 OPEN
at2ce4b2f7db93c353bca743ef0c8d2d5dd11d04cf, no merge timestamp/commit.
Existing #1773 source remains clean at0853. M0 remains incomplete and product
implementation has not started. No code tests are warranted for this metadata
checkpoint; existing sprint-JSON/link/diff validation is rerun. The STOP above
now applies; continue already-approved M1–M5 only after root confirms actual
accepted #2013 integration. Operator action, CI retry, #1926 repair and gate
bypass remain outside this agent’s authority.

## Execution resumed: legitimate integration satisfied

Root independently accepted #2013 PASS95 with all27CI terminal, then normally
merged it at96d69f05b6e84bd6666e8fa91ff948f803b67d92 (20:53:24UTC);
issue1773 closed20:53:25. The conditional execute instruction now permits
M1–M5 without a new approval question. New native execution checkout
.claude/worktrees/publish-key-file-1900-exec, branchfix/publish-protected-key-file,
normally integrated actual dev ata9aa133133f9720898c3d7ebfa12627101549211. No Go/AIL diff
from accepted source before M1, no merge conflict, new1900 docs remain planned.
Historical blocked checkpoint remains preserved above.

### M1 checkpoint

Protected same-handle reader implemented; synthetic focused tests PASS25.555s,
targeted lint0, boundaries/home isolation PASS. API-missing compile RED and
FIFO harness setup failure retained separately; the original child entered
existing CLI-build TestMain before reaching its3s bound, so the corrected
control measures the reader directly with bounded unblock teardown. No
production timer changed. Logs in session1900-execution/M1-*.log; no real
key file or external registry was used. M2 follows under existing approval.

### M2 checkpoint

Strict source/format selection passes isolated affected tests (pkg17.804s,
cmd0.768s), targeted lint0. Initial missing-API compile failure is retained
as an API red, not a legacy exploit reproduction. All subsequent Go commands
use a private outer HOME/allowlisted environment so package TestMain startup
also cannot inherit operator credential/export configuration. No fallback,
default probe, AI-reader change or environment mutation.
