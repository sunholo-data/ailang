# M-PUBLISH-PROTECTED-KEY-FILE

**Status**: Proposed — attended design approval pending; no implementation authorization
**Target**: v0.56.0, unreleased
**Priority**: P1
**Estimated**: 2–3 days, approximately 450–650 changed lines including portable tests/docs; re-estimate during approved planning
**Issue**: [#1900](https://github.com/sunholo-data/ailang/issues/1900)
**Baseline**: actual dev `dba1c499ca6f76b40992fd521e3dd440d9277f49` (2026-10-29)
**Integration dependency**: #2013/#1773 owns overlapping publish code and is not merged; see Integration below

## Problem statement and threat boundary

Publishing currently reads `AILANG_REGISTRY_API_KEY` only. The publishing guide
recommends exporting it for a shell session. The reporter exported it from
`.zshenv`, causing unrelated descendants to inherit registry publishing authority.
Their workaround binds a private file's contents to one command's environment;
that still gives that command's test/smoke descendants an inherited key.

Add an explicitly selected protected-file lane so the publishing process reads
the key into private memory and passes it only as the validator request header.
Do not export the file contents to process environment, argv, package metadata,
attestations, messages or a child command. Do not automatically discover a key
for every process running as the operator.

This reduces accidental environment inheritance. A mode 0600 file does **not**
isolate malicious programs running under the same user ID, and a command-line
path is not a capability boundary. Existing UID-separation designs own that
larger isolation problem. This feature must not claim that same-UID agents can
no longer read an operator's files or invoke an unconfined publishing binary.
Land #1964's confinement guarantee before credential access, not after it.

No real credential file was opened or written while preparing this design.
All verification below is source-based; future tests use synthetic dummy keys
and loopback services, never the live registry.

## Coverage and existing conventions

The current-owner refresh has both #1900 and #1965 open, without comments; no
matching owner in 24 open PRs or 18 active production coordinator tasks. It is a
bounded current snapshot, not an exhaustive historical ownership claim.
Repository keyword and local scaffold searches found no registry protected-file
design. Neural maxima were 0.33 implemented/0.28 planned, below duplicate gates.

Existing conventions were read before proposing policy:

- `factory.ReadKeyFile` supports AI credentials, trims surrounding whitespace,
  requires one non-empty line and bounds stat size at 16 KiB. It uses Stat followed
  by ReadFile; it does **not** enforce owner/mode/no-follow or bound a growing
  stream. Reusing it unchanged cannot satisfy a protected-file promise. Do not
  silently strengthen all AI credential readers in this package change.
- Existing AI flag/file precedence is flag over environment-file source; a
  selected invalid file errors rather than falling back. This design follows
  explicit selection/failure but proposes rejecting a simultaneous *secret*
  environment variable to avoid retaining inherited authority during file use.
- Activation uses O_NOFOLLOW and fail-closed platform declarations. Fileguard
  refuses an unsupported confinement platform. Accessapproval skips Unix mode
  checks on Windows; that skip is not evidence of Windows ACL protection.
- `internal/secrets` resolves 1Password/cloud secrets and can invoke external
  providers. It is not a local protected-file reader and must not become a
  fallback for a missing publisher key.
- Registry key consumers also include unpublish/key-management. This request
  adds the file lane to **publish only**; those commands retain their current
  environment lane and need no unrelated refactor.

## Goals

1. Explicit `ailang publish --key-file PATH` (and equivalent `ailang pkg publish`)
   authenticates from a validated private file without putting its contents in
   environment/argv or publisher-spawned processes.
2. Selected missing/unsafe/invalid files fail loudly; they never fall back to an
   environment key, a default location or an unauthenticated upload.
3. Preserve confinement before all file access and request construction.
4. Help/dry-run require no secret access; document that dry-run does not verify
   publishing credentials.
5. Retain legacy per-call environment publishing with a non-secret inheritance
   warning, and replace shell-startup-export advice with explicit invocation.

## High-impact decisions requiring approval

All rows are proposals, not an attended ruling.

| ID | Proposed choice | Why it matters | Decider | Change cost |
|---|---|---|---|---|
| D1 selection | Explicit `--key-file` only; no automatic default file and no new file-path env variable | Default discovery would grant ambient publish authority again; a path must be deliberately named per invocation | Human approval pending | Medium |
| D2 ambiguity/absence | Real publish rejects file+nonempty registry-key env together; file failure never falls back; no file+no env fails before upload | Defines authority/compatibility; callers must remove the inherited env key to choose the file lane | Human approval pending | Medium |
| D3 Unix protection | Initially macOS/Linux: same effective UID, regular non-symlink file, mode 0400 or0600 only; read through one validated no-follow handle | Prevents misleading mode-only/symlink/TOCTOU checks while allowing read-only secrets | Human approval pending | Medium |
| D4 other platforms | New file lane refuses on Windows/other unsupported platforms; do not treat os.FileMode as Windows ACL evidence | Avoids an unverified security claim. Native Windows DACL support would be a separately approved expansion; legacy env lane remains available | Human approval pending | High |
| D5 timing | Real publish resolves/validates once before package checks can spawn children; help/dry-run never open a key file | Controls secret lifetime, failure timing and offline behavior | Human approval pending | Medium |
| D6 existing env | Preserve env-only real publishing, with one stderr warning about inheritance; do not silently scrub process-global environment | Retains automation compatibility while explaining the existing exposure | Human approval pending | Medium |
| D7 scope | Publish only; no key-file lane for unpublish/key mint/revoke in this change | Keeps destructive/admin credentials outside the requested authoring change | Human approval pending | Medium |

D2 intentionally changes a real publish with no credentials from a remote
unauthenticated attempt to a local configuration error. D4 is an explicit
unsupported-feature refusal, not a fallback to the environment when a file was
selected. If full Windows protected-file support is required, amend D4 and its
ACL acceptance matrix before planning; do not improvise a permissions bypass.

### Design freeze

- [ ] Attended user approves D1–D7, including the Windows limitation.
- [ ] #2013 integration condition is met or root explicitly reviews a combined scope.
- [ ] Sprint-planner creates the approved plan; user separately says execute sprint.

## Solution design

### Credential selection

Parse `--key-file` locally to publish. Track explicit flag presence, so an empty
`--key-file=` is an error for real publish rather than indistinguishable from
absence. Resolve relative paths against the invocation's original cwd; do not
read from the rewritten/staged package. No implicit `~` expansion: shell-expanded
absolute paths and ordinary relative paths use normal CLI path conventions.

Real publish selects exactly one source:

| Explicit file option | Registry-key env | Result |
|---|---|---|
| Nonempty | Nonempty | Refuse ambiguous/inherited authority; ask operator to unset env |
| Nonempty | Unset/empty | Validate and read selected file once; error has no fallback |
| Empty explicit flag | Any | Refuse empty path |
| Absent | Nonempty | Existing env source, with inheritance warning |
| Absent | Unset/empty | Refuse missing publishing credentials before upload |

No key-derived source is written to config. Never call os.Setenv or install a
process-global credential cache. The selected value is an invocation-local
argument to upload/auth header construction, replacing uploadTarball's ambient
config.RegistryAPIKey reread. This also freezes the selected source/value across
quality checks and upload rather than consulting a changing environment later.
Call-site/API details are planner latitude, not a new public credential API.

### Authority and ordering

The existing command table declares publish AuthNetwork/AuthHomeWrite, and both
bare/group dispatch run through gatedRun before command work. Help is exempt.
Retain this behavior, including current refusal of confined dry-run. A file flag
must not turn publish into an admitted policy-tool operation.

Add a defensive existing `pkg.Require` check at the credential selection/read
boundary so direct internal function tests/callers cannot read a publisher key
before discovering they are confined. Treat publisher credential access as part
of the operator publishing operation; do not add a secret-read grant or bypass
through AILANG_PACKAGE_ROOT. Do not change `pkg.Require` semantics or the
Command.Authority table. Actual HTTP remains on existing `pkg.HTTPClient`.

For unconfined real publish: parse/help handling, authority check, credential
selection/read once, then existing package validation/quality/tarball flow and
upload with the selected value. This can fail credentials earlier than current
upload-time env reading, as intentionally approved in D5. Test/smoke processes
receive no new key variable. A legacy env key remains inherited because the
operator chose that existing lane; warn once without echoing its value.

For help/dry-run: do not inspect the credential file, test its existence, read its
contents or resolve an operator's default HOME. Dry-run still runs existing
package/quality checks and can fail those independently. Explicitly state that
credentials were not loaded/verified; no claim that dry-run verifies file safety
or remote authorization. No key-bearing request or publish side effect occurs.

### Protected local reader

Use a small publishing-specific reader under `internal/pkg` with build-tagged
platform validation. Do not import the AI/provider factory to access its helper,
change AI file semantics, or introduce a general secret-provider framework.

On macOS/Linux:

1. Check existing authority before filesystem access.
2. Open the explicitly named final component read-only with no-follow,
   close-on-exec and nonblocking flags. No-follow rejects final symlinks and
   nonblocking avoids hanging on a FIFO before file type can be inspected.
3. Validate **the opened handle**, not an earlier Stat result: regular file,
   owner equals effective UID, no group/other permissions or execute/special
   bits, permission mode 0400 or0600. Reject a directory, FIFO/socket/device,
   symlink, foreign owner or unavailable ownership metadata.
4. Read a bounded stream (maximum 16 KiB, detect byte 16 KiB + 1), then close before
   spawning package checks. Do not trust stat size alone. Trim surrounding
   whitespace, require one non-empty line, reject embedded CR/LF, NUL and other
   header-control bytes. Never include contents in a validation error.
5. Never create/chmod/chown/repair the file or its parent. No interactive prompt,
   keychain lookup, secret-manager invocation or fallback.

The promise concerns the opened final file/handle. Parent path components may
resolve through normal operator-managed mounts/symlinks; this is not a sandbox
path resolver or protection against an attacker controlling the operator's
ancestors or same UID. Advise a private operator-controlled parent directory.
Read once from that validated handle: path replacement after open cannot switch
the bytes being read through a second pathname open. It does not make immutable
memory or defend against concurrent writes by that same authorized UID.

For unsupported platforms, the file reader returns a named unsupported-platform
error before opening anything; it must not inspect a potentially protected file
and then claim ACL verification was successful. Windows CI must assert this
refusal, not blanket-skip the feature's tests. Other publishing behavior remains.
Existing syscall/O_NOFOLLOW and x/sys dependency conventions can be reused;
exact low-level API is implementation latitude only within these guarantees.

### Output, requests and documentation

Authentication/validation errors name the source/path/reason, never the secret
or its prefix. Avoid logging request headers or marshaling the selected key into
an attestation, tarball, collaboration message or trace. If a loopback validator
reflects the exact selected key in a response error, redact that selected value
before rendering the existing response text; keep the actual validation cause.
No general logging framework or unrelated token redaction policy is changed.

Use existing X-API-Key for the same configured validator endpoint. No new URL,
transport, retry, redirect policy, token shape or registry schema is introduced.
The existing validator/transport trust model remains; this feature addresses
credential source/inheritance and local file protection, not general transport
hardening. Keep existing post-publish message/cascade behavior; tests must isolate
it without contacting real services.

Update publish help at its source, CLI reference and publishing guide:

```sh
# Operator provisions this file privately; do not commit the key.
ailang publish --key-file "$HOME/.config/ailang/registry.key"

# Legacy automation: scope the value to this command, never a startup export.
AILANG_REGISTRY_API_KEY='<scoped-key>' ailang publish
```

These are documentation examples, not commands executed while designing. On
macOS/Linux document mode 0400/0600, effective owner, private parent, no final
symlink, and the same-UID limitation. Warn against shell-startup exports and
explain that per-call env still reaches publish's descendants. Windows docs
state the file lane is unavailable pending native protection support. The guide's
maintainer key-mint examples are separate existing env consumers; do not imply
that this flag supplies credentials to key management.

## Integration with current work

PR [#2013](https://github.com/sunholo-data/ailang/pull/2013), exact head
`0853c9fc15f3daf5f9e304e6e451e78c21fd899c`, owns staged published smoke, shared
rewrite extraction and diagnostics in pkg_publish.go/package tests. It remains
clean and **unmerged** after the allowed build retry repeated unchanged #1926
runner PID readiness failure; this design does not bypass its evaluator gate.

Future execution requires #2013's legitimate gate resolution/normal merge, or
an explicitly reviewed combined integration scope. Do not copy the older dev
pkg_publish.go over its changes or create a competing rewrite implementation.
Recheck actual merged dev and current owner before planning/execution. The new
credential seam must preserve staged registry source+asset proof, D2 pins,
bounded Notes, dry-run and original author-input restoration. #1965's docs
fallback proposal remains in a separate worktree and is not a dependency.

## Implementation outline and acceptance

This is a design, not a sprint plan. Likely files: new publishing credential
reader plus Unix/unsupported companions and tests in internal/pkg; narrow
pkg_publish auth/flag changes and CLI test companion; publishing help/reference,
guide and unreleased fragment. No AI/effects/types/parser/cache-epoch changes.

- [ ] **AC1 source selection:** test every row of the matrix, no implicit default
  file, invalid selected file never falls back to a dummy env key, explicit
  empty flag fails, missing real credentials produces no upload.
- [ ] **AC2 protection:** synthetic temporary files cover 0400/0600 success,
  0644/0640/0700/special-bit refusal, final symlink, directory/FIFO, foreign owner
  metadata, empty/multiline/control/oversize and a bounded growing stream. Prove
  the reader reads only its opened validated handle. Use bounded helpers and
  disposable dummy files; no privileged ownership mutation or real key.
- [ ] **AC3 authority ordering:** actual bare and pkg publish under confinement
  refuse before an instrumented file opener, regardless of file/env presence.
  Direct reader/selection calls also refuse. Existing help remains available;
  dry-run's current confinement refusal remains. No authority table weakening.
- [ ] **AC4 no secret inheritance:** an actual synthetic package quality/smoke
  child can observe only a sentinel for absent registry-key env, while a counted
  loopback upload receives exactly the dummy file key. Parent env is unchanged;
  key is absent from child argv/output, tarball/attestation/logs/messages/errors.
  Instrumented upload must be proven active by a positive control, not assumed.
- [ ] **AC5 lifetime:** real publish reads once before package subprocesses, closes
  the handle, and uses frozen selected value even if the pathname/env changes
  later. Help/dry-run with missing/unreadable/unsupported-platform file option
  opens nothing and makes no upload. Ordinary package gate failure still fails.
- [ ] **AC6 legacy compatibility:** env-only upload still uses same X-API-Key and
  warns once without secret text. Unpublish/key-management remain unchanged.
  Mock validator body reflecting the dummy key cannot echo it through an error;
  existing status/validation semantics remain recognizable.
- [ ] **AC7 platforms:** macOS/Linux run full protected-handle assertions;
  Windows/other-platform implementation executes a named fail-closed file-lane
  refusal and existing env/help/dry-run controls, no fake mode/ACL success.
- [ ] **AC8 integration/docs/gates:** after authorized #2013 integration, actual
  offline CLI published-source/asset, D2, diagnostics and input-restoration
  controls remain green. Run build, focused tests, make test-changed, lint,
  boundaries, home isolation, sizes, fmt, help/reference consistency and changelog
  checks. No full local make test; broad exact-head CI and independent sprint
  evaluation remain required. No live publish, external credential or auth-store
  access in validation.

Estimate includes platform/control tests and combined publish validation, not
just one flag. Reuse existing bounded build/process/loopback helpers; do not add
an ad-hoc shell publishing wrapper or a global default credential resolver.

## Verification log

| ID | Source-verified premise | Evidence |
|---|---|---|
| V1 | Publish is env-only with no key-file flag | Read pkg_publish.go flag declarations and uploadTarball:269; config.RegistryAPIKey:44 returns only EnvRegistryAPIKey |
| V2 | Upload currently reads env after package checks/dry-run | Read entire pkgPublishCommand and uploadTarball call; dry-run returns before upload; quality executes attested children |
| V3 | Existing authority gates precede both command spellings, help exempt | Read commands_pkg.go publish Authority row/gatedRun:186ff and commands_groups.go generated legacy row; HTTPClient transport checks Require |
| V4 | Existing AI reader is not permission-safe and performs separate pathname reads | Read internal/ai/factory/keyfile.go and TestReadKeyFile body; Stat/ReadFile,16KiB stat cap, trim/single-line; no owner/mode/no-follow checks |
| V5 | Reusable platform conventions exist, but not a verified generic protected-file reader | Read activation lock_unix.go/privatedir_other.go, fileguard unsupported platform declarations, accessapproval service mode checks; searched protection/API helpers across internal/cmd |
| V6 | Windows mode skip is not ACL evidence | accessapproval/service.go explicitly skips mode test on Windows; activation documents lack of equivalent mode guarantee; no DACL reader in scoped internal/cmd source search |
| V7 | Same-UID file isolation is outside this feature | Read m-mac-agent-uid threat table M3 and executor-env-hardening D2; cite existing design rather than reading actual operator credentials |
| V8 | Other env consumers exist and need explicit scope separation | Full repository RegistryAPIKey call search finds pkg_publish, pkg_unpublish and pkg_keys only, plus config definition |
| V9 | Existing guide recommends shell export | Read package-publishing.md quickstart/API-key/maintainer sections and publish help source |
| V10 | #2013 overlaps publishing and is not a safe assumed ancestor | Root's terminal independent gate classification; branch0853 source inspection versus dev; shared rewrite extraction/quality changes are owned and preserved |
| V11 | Current owner/design coverage is bounded and no matching active owner found | Saved 24-open-PR/18-active-production-task inventory plus issue1900 body/comments; local scaffold neural maxima 0.33/0.28; manually read AI/UID/authority/explicit-host credential conventions |
| V12 | Existing raw validator responses can be returned in errors | Read uploadTarball status handling:300ff; new exact-selected-key redaction must cover reflection without replacing validation causes |

No new language support/limitation, code allocation, live registry behavior or
Windows ACL implementation is claimed. Exact file protection is a proposed
contract to prove during execution, not functionality already present.

## Risks and mitigations

| Risk | Mitigation |
|---|---|
| Automatic default merely moves ambient authority from env to disk | Explicit flag only, no default discovery; state same-UID limitation |
| Symlink/racy read or FIFO stalls | No-follow/nonblocking opened handle, handle metadata, bounded stream, close before child |
| Windows branch silently treats mode bits as ACL proof | Named unsupported file-lane refusal unless a separate native ACL design is approved |
| File lane still carries an inherited environment key | Refuse simultaneous sources; warning on intentionally retained legacy env lane |
| Secret leaks into package processes or reflected errors | Invocation-local value only, no Setenv, actual child/loopback/error controls |
| Integration overwrites unmerged smoke work or bypasses its gate | Explicit #2013 dependency/reviewed combined-scope requirement |

## Axiom compliance

| Axiom | Score | Reason |
|---|---|---|
| A1 Determinism | 0 | No language result/semantic change |
| A2 Replayability | 0 | No replay change |
| A3 Effect legibility | +1 | Publishing credential source is explicitly selected |
| A4 Explicit authority | +1 | Removes need to grant secret through every descendant environment |
| A5 Bounded verification | +1 | Finite source/protection/platform/loopback matrix |
| A6 Safe concurrency | 0 | No concurrency model claim; one invocation-local read |
| A7 Machines first | +1 | Missing/unsafe/ambiguous sources fail clearly without fallback |
| A8 Minimal syntax | 0 | One CLI flag, no language syntax |
| A9 Cost visibility | 0 | No cost-accounting change |
| A10 Composability | 0 | Existing publishing/auth header preserved |
| A11 Structured failure | +1 | Existing confinement error plus specific unsafe/unsupported source errors |
| A12 System boundary | +1 | Secret read belongs to operator publish, not package child processes |

**Net score: +6.** No negative A1/A3/A4/A7 score. Design approval and execution
remain separate; this score does not authorize product changes.

## Related documents

- [Package authority chokepoint](../v0_54_19/m-pkg-authority-chokepoint.md): landed
  guard ordering, transport and dispatch constraints, not a credential-file lane.
- [Executor environment hardening](../v0_49_1/m-executor-env-hardening.md): separates
  child credentials; this proposal changes author CLI input, not fleet identity.
- [macOS agent UID isolation](../v0_54_13/m-mac-agent-uid.md): explicit same-UID
  threat limitation; no account/permission deployment is part of this feature.
- [Worker policy and lazy AI](../../implemented/v0_54_5/m-worker-policy-and-lazy-ai.md):
  credentials are read when their operation is needed; do not change AI behavior.
- [Explicit host credentials](../../implemented/v0_44_0/m-serveapi-websocket-bridge.md):
  named sources without ambient fallback, for a different consumer.

## Deferred decisions and non-goals

After approval, agent latitude covers helper/type names, exact safe error wording
and test layout. No fallback, chmod repair, default secret file, native Windows
ACL implementation, key-management scope extension, global credential framework,
AI reader semantics change, UID deployment, registry protocol/transport change,
runner #1926 repair or #2013 gate bypass is authorized by this design.
