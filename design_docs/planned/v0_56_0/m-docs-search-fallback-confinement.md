# M-DOCS-SEARCH-FALLBACK-CONFINEMENT

**Status**: Design and sprint execution approved; implementation awaiting independent acceptance
**Target**: v0.56.0 (unreleased; no version/tag change)
**Priority**: P1
**Estimated**: 1–2 days, approximately 280–380 changed lines including tests/docs; planner must re-estimate after approval
**Issue**: [#1965](https://github.com/sunholo-data/ailang/issues/1965)
**Baseline**: actual dev `dba1c499ca6f76b40992fd521e3dd440d9277f49` (2026-10-29)
**Dependencies**: reuse landed #1964 package authority refusal contract; preserve M-DX27 local/backend separation

## Attended approval and current stage (2026-10-29)

The user said “approve the three designs”. Root's binding record is
`/tmp/ailang-backlog-20261029/three-design-approvals.json`, approving exactly design
commit `096a808ac76e858143eee4e62854ceb56c0e2b74` for **sprint planning only**.
Receipt observed at 2026-10-29T14:48:58Z is an assistant-recorded observation, not
an independently supplied user send timestamp. The approved scope is whole
GitHub backend refusal under confinement, preserving local SimHash and leaving
explicit neural/cache-management separate. No third external review, plan/execute,
implementation, operator-card action or #2013 CI bypass is inferred.

The later user instruction “exeute all three sprints” approved the reviewed
[sprint plan](m-docs-search-fallback-confinement-sprint-plan.md) and execution,
bound by `/tmp/ailang-backlog-20261029/three-sprint-execution-approval.json`,
observed2026-10-29T20:17:42Z (assistant receipt observation). All four milestones
are authorized; independent evaluation and merge remain separate. The JSON is
`.ailang/state/sprints/sprint_M-DOCS-SEARCH-FALLBACK-CONFINEMENT.json`; the
[validation map](m-docs-search-fallback-confinement-validation.md) records proof.
This update changes stage metadata, not the approved semantics.

## Problem statement

When no local documentation directory is found, `ailang docs search` constructs
an authenticated GitHub backend even with `AILANG_AGENT_POLICY` set. Construction
may run `gh auth token`; searching may issue HTTP and write a result cache under
HOME. Policy-tool exposes `docs_search` and pins the policy variable in its CLI
child, so the child reaches this path despite the host's confinement intent.

#1964 did not resolve this: its primitive scan and dispatch declarations cover
package verbs, and its design explicitly defers expansion to other commands.
The docs-search GitHub backend is outside that scan. This follow-up closes that
specific fallback boundary without changing package authority or migrating all
top-level commands.

Fresh offline probes on actual dev and #1773's existing binary both reach a
compiled fake `gh` under confinement when local docs are absent. The fake writes
a private marker and exits without returning a token: no real credential or
network request is used. Local SimHash and `--no-github` controls avoid the helper.
This verifies external-helper reachability; the reported real `gh` device-id
write is issue evidence, not a newly repeated measurement. HTTP/cache risks are
verified from source (V2–V4).

## Ownership and coverage gate

On 2026-10-29 both #1965 and #1900 are open with no comments. A fresh `gh pr list`
inventory contains 24 open PRs, with no matching #1965/#1900 implementation or
design owner. The live production coordinator read-only list has 18 active
pending/queued/running/approval tasks, with no matching issue/topic owner. This
is a bounded current-owner snapshot, not a claim of no historical task.

The existing M-DX27 design/plan cover authenticated fallback, repository
restriction, cache and disable controls. They contain no confinement gate. The
package authority and registry-confinement designs cover separate package paths.
A fresh scaffold search found maximum neural scores 0.30 implemented and 0.28
planned, below the duplicate thresholds; keyword/manual coverage searches found
and read those concrete related designs. Their existing behavior is a constraint,
not work to implement again. No second fallback implementation is proposed.

#1900 remains a separate unimplemented key-file feature: current publishing reads
only `config.RegistryAPIKey()` from the environment, with no publish key-file
flag. It needs independent decisions about credential-source precedence, file
protection and Windows ACLs. It is not combined into this confinement design.

## Goals

1. Under confinement, refuse the GitHub docs-search fallback before credential
   lookup, child execution, GitHub cache access or network requests.
2. Keep local SimHash documentation search usable, with existing results/flags.
3. Return a named, actionable confinement error; do not suggest authenticating
   or re-enabling GitHub as the remedy for a confined process.
4. Preserve unconfined authentication, repository restriction, cache TTL and
   secret-safe errors, plus the existing backend separation.
5. Prove the boundary with actual CLI/policy-tool and direct-library controls
   using private HOME, fake processes and loopback transports only.

## Approved high-impact decisions

| Decision | Approved choice | Why it matters | Chosen by | Change cost |
|---|---|---|---|---|
| No local corpus under confinement | Fail with a canonical typed confinement refusal; require a local corpus or operator execution | Determines behavior for installed agent binaries, rather than silently returning an empty result | Attended user approval recorded above (planning only) | Medium |
| Existing GitHub result cache under confinement | Refuse before reading it, including a fresh cache hit | Makes GitHub backend entirely operator-only and avoids credential resolution before a purported offline cache hit | Attended user approval recorded above (planning only) | Medium |
| Scope | Gate only GitHub fallback/library cache; keep local search and explicit neural/cache-management policy separate | Avoids claiming a universal top-level confinement policy that this bounded change cannot prove | Attended user approval recorded above (planning only) | Medium |
| Shared authority representation | Reuse `pkg.Require`/`pkg.ConfinedError` with a docs-specific remedy; do not create a second policy variable or refactor the authority package | Keeps confinement detection/error type consistent with #1964 | Approved narrow reuse; no authority-semantic change | Low |

### Design freeze

- [x] Attended user approved the refusal/cache/scope choices above for planning.
- [x] Sprint-planner prepared exact draft milestones/acceptance after design approval; plan approval is pending.
- [ ] Attended user explicitly says execute sprint before product edits.

Only the explicitly recorded design approval and draft planning are complete.
No implementation or agent/coordinator handoff is authorized; the execute box
remains open until the user explicitly approves execution of a concrete plan.

## Solution design

### Backend boundary

Keep `docsearch.SearchBackend` and local selection unchanged. Add a small private
authority helper within `internal/docsearch/github` that delegates to existing
`pkg.Require` for network/home-write operations. Use a docs-specific remedy:
`use local documentation with --path, or run docs search outside the sandbox`.
No new error code, environment variable or policy schema is needed.

1. `NewBackend` calls the helper **before** `config.GitHubToken`, `gh auth token`
   or repository discovery. The guard is unconditional with respect to token
   presence: an inherited token cannot unlock fallback under confinement.
2. `Backend.Search` refuses before token validation/cache lookup. A backend
   constructed earlier, a manually populated exported Backend, or the test
   constructor must not bypass confinement. Recheck authority at the actual HTTP
   call boundary (including pagination); do not rely solely on CLI selection or
   constructor-time environment state.
3. `writeCache` checks `AuthHomeWrite` before resolving HOME, creating a directory,
   creating a temp file or renaming it. Direct callers must also refuse. A typed
   confinement error is propagated rather than discarded by Search's current
   best-effort cache write. Ordinary unconfined cache failure remains best-effort
   as before; do not turn optional cache persistence into a new failure policy.
4. Keep existing HTTP/repository/authentication behavior for unconfined callers.
   No new transport, credential provider, token precedence or cache layout is
   introduced. All GitHub requests continue through the existing backend; do not
   add a separate CLI HTTP path or an unauthenticated retry.

`pkg.Require` is already independent of docsearch: `go list -deps ./internal/pkg`
contains no docsearch package (V8). This narrow reuse requires no authority
extraction or compiler dependency change. Run existing architecture gates after
implementation; a newly discovered import conflict must be reported before
expanding scope.

### CLI selection and diagnostics

`docsSearchCommand` continues choosing LocalBackend when an explicit valid
`--path` or discovered corpus exists. Do not put unconditional authority on the
entire docs command: that would refuse the local operation this fix must retain.

When fallback construction returns `pkg.IsConfinedError`, preserve its canonical
message and emit only the local-corpus/operator remedy. Suppress the current
`set GITHUB_TOKEN / gh auth login` hint for that error. In the already-disabled
`--no-github`/`AILANG_NO_GITHUB_SEARCH` branch under confinement, preserve the local
failure but avoid advising the caller to unset a control that cannot grant
operator authority. Unconfined diagnostics retain existing authentication hints.
No successful empty result, silent backend change or new JSON error envelope.

### Scope and adjacent paths

The audit also found that **explicit local neural search** uses an embedding
cache under HOME (`internal/docsearch/embed.go`) and an embedder. Policy-tool's
`docs_search` schema admits only json/limit flags, not neural/rebuild/cleanup.
This proposal does not certify every docs mode as filesystem/network-free and
does not change explicit neural, warmup or cache-management behavior. Those are
a distinct policy decision if broader top-level confinement is requested.

The default local **SimHash** control is the preserved path proven here. The
existing `internal/docsearch/search.go` algorithm remains zero-diff. No generic
Command.Authority migration, package cache/asset change, registry key handling,
policy-tool allow-list expansion or operator permission grant is included.

## Compatibility and intentional change

| Situation | After implementation |
|---|---|
| Confined, no local docs, token absent | Typed refusal before fake/real gh invocation |
| Confined, no local docs, token present | Same refusal, zero requests; token never used |
| Confined, pre-existing GitHub cache hit | Same refusal before cache use |
| Confined, explicit local corpus/default discovered corpus, SimHash | Existing successful local search |
| Confined, explicit missing/invalid --path | Existing local path error; no fallback |
| Confined, fallback disabled | Local failure, safe local/operator remedy |
| Unconfined, authenticated fallback | Existing authenticated requests/cache/repo validation |
| Unconfined, missing/rejected credentials | Existing loud authentication failure, no unauthenticated retry |

This intentionally removes confined access to the GitHub backend even when a
token/cache exists. It does not change language syntax, type/effect semantics,
engine execution or module lookup; no AILANG capability claim or fixture is
needed for this host-Go boundary. No parser/typechecker Conflict Surface applies.

## Implementation outline (not an execution plan)

1. Add direct backend/constructor/cache refusal tests that fail first, preserving
   existing unconfined API/cache tests. Add the narrow guards and typed-error
   propagation. Estimated 70–100 production lines including a small helper.
2. Add actual CLI/policy-tool process matrix and safe hints; preserve local
   preference and no-GitHub controls. Estimated 160–220 test lines.
3. Update existing CLI reference, semantic-search guide, SECURITY.md and one
   unreleased changelog fragment. Estimated 40–60 lines; no new script/service.

Likely files: `internal/docsearch/github/{authority.go,github.go,cache.go,github_test.go}`,
`cmd/ailang/docs_search.go`, a focused CLI confinement test companion,
`internal/policytool` integration tests if needed for the real child path,
`docs/docs/reference/cli.md`, `docs/docs/guides/semantic-search.md`, `SECURITY.md`.
Do not edit `internal/pkg/authority.go` semantics or the #1773 source branch.
Planner should reconcile the 280–380 estimate against required portable process
harness reuse; budget 1–2 days including review rather than treating guard edits
as the full effort.

## Acceptance criteria and verification strategy

- [ ] **AC1 actual CLI:** outside corpus discovery, policy set, no token and fake
  gh/git child helpers: exit nonzero with canonical policy refusal, zero helper
  invocations, unchanged private HOME/XDG state. Helpers must first be proved
  active by an unconfined control; absence alone is not a valid instrument.
- [ ] **AC2 token + direct library:** inherited dummy token, direct `NewBackend`
  and injected `Backend.Search` via `NewBackendForTest` all refuse. A counted
  loopback HTTP server receives zero requests confined and a valid unconfined
  request. No real token, GitHub egress or auth-store access.
- [ ] **AC3 cached/direct writer:** seed a valid cache unconfined, prove its hit,
  then confined Search refuses; direct writeCache returns typed home-write
  refusal before any mutation. Snapshot all private HOME/XDG bytes before/after,
  not just the currently known device-id/cache filenames.
- [ ] **AC4 local compatibility:** actual CLI explicit and auto-discovered local
  SimHash search succeed under confinement with unchanged corpus/HOME and no
  helper/request. Include JSON parsing, invalid --path, --no-github and env
  disable controls. No auth/unset-controls hint on a confined failure.
- [ ] **AC5 policy-tool:** exercise actual docs_search dispatch and its actual
  child binary, not only a mocked runner. Prove pinned child policy still
  refuses the fallback; local SimHash remains allowed. Retain flag allow-list
  (json/limit) and existing output bounds. No new credential admission.
- [ ] **AC6 unconfined compatibility:** existing authenticated pagination/request,
  repo-default/repo-refusal, error-token-safety and cache expiry/isolation tests
  remain green. Tests use loopback and private HOME; no live GitHub search.
- [ ] **AC7 gates/docs:** make build, focused affected tests, make test-changed,
  lint, boundaries, home isolation, sizes, fmt and changelog pass. No full local
  make test; exact-head PR CI supplies broad Linux/Windows gates. Use bounded
  portable Go child helpers rather than shell scripts; Windows must execute the
  authority assertions without POSIX-only blanket skips. Docs scope remains
  GitHub fallback, not a universal docs-mode confinement claim.

Independent sprint evaluation follows implementation and real terminal gates;
no executor self-score. Design approval and draft planning are recorded above;
execution, PR and live publication remain separate unauthorized steps.

## Verification log

| ID | Verified claim | Evidence |
|---|---|---|
| V1 | Current confined CLI reaches external credential helper; local/disable controls avoid it | Fresh actual-dev build plus #1773 existing binary, eight offline arms in session `1965-probe/results.json`; compiled fake gh exits1 after marker, no real tokens/network |
| V2 | Construction uses token then gh/repository detection without policy gate | Read `internal/docsearch/github/github.go` NewBackend34–56; config.GitHubToken resolves only GITHUB_TOKEN |
| V3 | Search can request HTTP and read cache without confinement check | Read Backend.Search66–137, direct client.Do and pre-request cache read |
| V4 | Cache writer creates HOME directories/temp files/renames | Read `internal/docsearch/github/cache.go` cachePath/writeCache, no Require/Confined reference; Search currently ignores write error |
| V5 | Package scan does not cover this backend; #1964 deferred broader authority | Read `internal/pkg/authority_boundary_test.go` authoritySurfaceFiles and authority design Deferred Decisions/Future Work |
| V6 | docs_search is admitted and child policy pinned; neural flags absent | Read `internal/policytool/cli_ops.go` cliSchemas docs_search and childEnv/runAilang; read existing ChildEnvCarriesPolicy test body |
| V7 | Existing fallback tests cover unconfined behavior, not this new confinement contract | Read all `internal/docsearch/github/github_test.go` bodies and `cmd/ailang/docs_search_guard_test.go` local SimHash regression |
| V8 | Reusing pkg authority does not introduce a docsearch import cycle | Actual go list -deps ./internal/pkg saved `1965-pkg-import-closure.txt`, zero docsearch entries; read authority.Require/ConfinedError definitions |
| V9 | No currently matching owner/design found within stated bounds | gh openPR24 + live gcp activeTasks18 saved in session; exact issue/topic searches; read M-DX27, package authority and registry confinement docs; scaffold neural maxima0.30/0.28 |
| V10 | #1773 does not change the affected docs/key-source seam | git diff actual dev..fix/pkg-smoke-staged-lock over docs_search.go, internal/docsearch and config/pkgregistry.go is empty |
| V11 | Local neural has separate I/O; no universal docs-mode guarantee is valid | Read internal/docsearch/embed.go NewEmbeddingCache49/write112 and local Search neural branch; policy-tool flags only json/limit |
| V12 | #1900 key-file lane still absent | Read all publish flag declarations/upload config.RegistryAPIKey call and config/pkgregistry.go; current publishing help/docs provide no key-file option; no secret-file access |

Evidence directory: `/tmp/ailang-backlog-20261029/`. The read-only refresh report
`1965-1900-current-dev-refresh.md` and saved JSON/logs are session provenance;
committed source paths and future regression tests are reproducible evidence.
No current website, law, external product behavior or language limitation is
asserted by this design. No new diagnostic code is proposed.

## Risks and mitigations

| Risk | Mitigation |
|---|---|
| Guard only at CLI misses direct/reused backend | Constructor, Search, HTTP and cache-write boundary tests, with direct injected backend controls |
| Cached results or dummy token quietly bypass policy | Refuse before credentials/cache and explicitly test both lanes |
| Generic authentication hints encourage the wrong remedy | Typed refusal handling and actual stderr negative assertions |
| Unconfined best-effort cache behavior changes | Propagate only typed confinement refusal; preserve other optional persistence behavior |
| Overclaiming local neural containment or all CLI commands | Explicit narrow scope and separate adjacent-mode decision, no generic authority refactor |
| HOME/process test silently inactive or platform-specific | Unconfined positive instrumentation, full private state snapshots, portable Go helpers and Windows CI |

## Axiom compliance

| Axiom | Score | Justification |
|---|---|---|
| A1 Determinism | 0 | No language/runtime result change |
| A2 Replayability | 0 | No trace/replay change |
| A3 Effect legibility | +1 | An automatic network/credential fallback becomes a named refusal |
| A4 Explicit authority | +1 | Ambient token/helper/cache cannot grant operator fallback under confinement |
| A5 Bounded verification | +1 | Finite actual-CLI/library matrix with counted local instruments |
| A6 Safe concurrency | 0 | No concurrency model change |
| A7 Machines first | +1 | Existing typed refusal supports consistent host/tool handling |
| A8 Minimal syntax | 0 | No syntax/flag/schema addition |
| A9 Cost visibility | 0 | No new cost accounting |
| A10 Composability | 0 | Existing backend API retained |
| A11 Structured failure | +1 | Reuses ConfinedError rather than silent results/auth confusion |
| A12 System boundary | +1 | Enforces the credential/HTTP/cache boundary at its owning backend |

**Net score: +6.** No negative A1/A3/A4/A7 score. Design approval covers planning
only; this arithmetic is not implementation authorization.

## Related documents

- [M-DX27 fallback design](../v0_29_0/m-dx27-docs-search-github-fallback.md) and
  [plan](../m-dx27-docs-search-github-fallback-sprint-plan.md): existing implemented
  source despite stale planned metadata; retain authenticated fallback/local
  separation, not duplicate implementation.
- [Package authority chokepoint](../v0_54_19/m-pkg-authority-chokepoint.md): reuse
  its authority/refusal contract; this design closes one explicitly deferred
  non-package path rather than reopening package semantics.
- [Registry confinement](../v0_52_6/m-pkg-registry-confinement.md): adjacent package
  cache/network boundary, distinct from GitHub docs-search credentials/cache.
- [SECURITY.md](../../../SECURITY.md): existing guarantee is package-specific;
  add the scoped docs fallback guarantee only after verified implementation.

## Deferred decisions and non-goals

Agent latitude after approval: private helper filename, assertion organization,
exact safe hint wording, and portable helper reuse. No new JSON fields or codes.

Not included: #1900 protected publishing keys; generic top-level authority
migration; explicit local neural/cache-management containment; moving authority
out of internal/pkg; changing token precedence, repo policy, cache TTL or local
search algorithm; external security disclosure; coordinator/card actions; live
GitHub/API probes. A need for any of these requires a separately approved scope.
