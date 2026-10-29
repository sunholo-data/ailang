# Sprint Plan: M-DOCS-SEARCH-FALLBACK-CONFINEMENT

**Status: APPROVED — execution authorized; implementation in progress.**
Design [m-docs-search-fallback-confinement.md](m-docs-search-fallback-confinement.md), exact approved commit `096a808ac76e858143eee4e62854ceb56c0e2b74`; issue **#1965 only**. Target **v0.56.0 unreleased**, not a version/tag request.

The user first approved the design for planning (receipt observed2026-10-29T14:48:58Z, assistant observation). The later instruction “exeute all three sprints”, observed2026-10-29T20:17:42Z, explicitly approves this reviewed plan and all four implementation milestones. Binding record `/tmp/ailang-backlog-20261029/three-sprint-execution-approval.json` identifies the reviewed plan SHA25636f65d30acc3c4253f2564221a12c7dba44ec7df46a3af656a689c9ce25b35fa and initial JSON3fd34ec5a3da1cd871c90c18a54c6c28fda3d38b6c188ede792fc2f91da4e6c9. No external/provider review, operator-card mutation or unrelated CI bypass is authorized. Independent evaluation and merge remain separate.

## Summary and scope

Enforce the existing canonical host authority contract at GitHub backend construction, direct/reused Search, each actual HTTP request and direct cache write. Preserve existing unconfined authentication/repository/cache behavior and local SimHash. Add discriminating real CLI/policy-tool controls with private state and no live GitHub/credential/Cloud Trace access.

**Duration:** two attended days, four bounded 60–90-minute milestones plus CI/independent review waiting. **Risk:** medium, driven by hermetic cross-platform process proof rather than guard complexity. **Estimate:** **90 implementation + 385 tests + 50 docs = 525 changed lines**. This includes 25% uncertainty above 420 base; original design's 280–380 estimate undercounted full positive instruments, private-state snapshots and separate real policy-tool/Windows proof. Do not omit required proof to meet a line estimate.

### Approved rulings and boundaries

All design decisions are ratified, not new freeze gates:

- Whole GitHub backend is operator-only under AILANG_AGENT_POLICY, even with inherited token or valid cached result. Refuse before credentials/helper/git discovery, cache or requests.
- Reuse `pkg.Require`/`pkg.ConfinedError` with a docs-specific local-corpus/operator remedy. Do not edit package authority semantics or extract/migrate authority infrastructure.
- Default/explicit local **SimHash** remains usable. Explicit neural/embed/warmup/cache-management modes remain a separate policy scope; `internal/docsearch/search.go` algorithm stays zero-diff.
- Preserve token precedence, repo restriction/defaults, cache TTL/layout, authenticated HTTP behavior, private result schema, optional unconfined cache-write behavior and policy-tool allow-list/output contract.
- No .ail product/example edits, language/compiler changes, new transport/config/flag/schema, unauthenticated retry, live search/auth store or broader top-level migration.

No human semantic decision remains OPEN inside this approved scope. The plan/execute gate is satisfied by the explicit instruction above; execute sequentially without new approval pauses after each milestone. If a genuinely new import/authority/API policy beyond the design is necessary, **STOP: commit current authorized work, report `BLOCKED: <new decision>` citing the design's Deferred decisions/non-goals or Backend boundary, and end the run**. Do not widen scope or silently fall back.

## Current implementation and dependencies

Planning HEAD 096a808ac, design source baseline actual dev dba1c499. NewBackend currently resolves GITHUB_TOKEN/gh and git before policy; Search reads cache and invokes client.Do without a policy check; writeCache resolves HOME/mkdir/temp/rename; Search currently ignores all cache-write errors. docsSearchCommand emits auth/unset-control hints on fallback failures. Policy-tool docs_search admits json/limit, pins its policy in the real child and uses an existing response envelope.

Landed #1964 provides Require/ConfinedError. #1773 touches a different package-publish seam and is **not an integration prerequisite**; preserve its source/carrier. Root reports #1926 merged 93e4f2586 at 14:49:37 UTC, but that is **plan+JSON only**, not a runner fix; #2013 remains open 0853 with failed aggregate CI. “Merged” grants no execution or retry authorization here. Treat any later whole-suite failure from that external work as root-owned evidence/routing, not a reason to modify it.

Before any future execution, inspect clean/dirty status and bounded ownership for this seam, preserve these drafts once authorized, normally integrate actual dev on a clean branch and retain all other owners' changes. Do not import coordinator tasks, dispatch or cherry-pick unmerged #1773. Current planning does not claim a fresh live task-plane inventory beyond root's direct ownership assignment.

## Velocity and reuse evidence

The existing `analyze_velocity.sh 7` was inspected and run read-only. It exits 141 in its existing head pipeline and samples ancient changelog entries; it supplies no reliable daily rate. Raw output is `/tmp/ailang-backlog-20261029/1965-planner-velocity-script.log`.

Relevant immutable attended host-tooling checkpoints (cmd/ailang+internal/pkg only):

| Commit | Scope | Add/delete |
|---|---|---:|
|19f2933ef|#1773 staged dependency/asset tests and helper|600/56|
|f3499b71e|bounded diagnostic/quality propagation|326/7|
|f8e15bc2e|post-save warning and real CLI controls|157/0|
|**Total**|fixture-heavy execution evidence, not release/daily velocity|**1083/63**|

These are multi-agent/attended, cache-warm checkpoints with broad gate still externally blocked; their short commit intervals are **not** a single-agent throughput measure. Use a conservative capacity assumption **300 LOC/day** for two days; estimates already include25% uncertainty. Concrete counts/SHAs/timestamps are in `/tmp/ailang-backlog-20261029/1965-planner-velocity.json`.

Registry reuse gate: all milestones are `none` for runtime package dependency. This is enforcement/test/docs of an existing **host-Go trust boundary**, not a package-like new capability; an AILANG registry package cannot intercept host credentials/HTTP/cache or direct Go calls. Registry search/info/docs is not applicable here and no package is published or inbox contribution sent. Reuse inspected M-DX27, internal/pkg authority, existing GitHub backend, buildAilang/runCLIIn/RunBounded/SetHomeDir and current policy-tool contracts. Matching populated decisions are in the JSON; no placeholder audit.

## Milestones

| ID | Implementation | Tests | Docs | Total | Bound | Depends |
|---|---:|---:|---:|---:|---|---|
|M1_GITHUB_AUTHORITY_BOUNDARIES|65|140|0|205|75–90 min|none|
|M2_ACTUAL_CLI_AND_SAFE_HINTS|25|125|0|150|60–90 min|M1|
|M3_REAL_POLICY_TOOL_CHILD|0|100|0|100|60–75 min|M2|
|M4_DOCS_GATES_AND_HANDOFF|0|20|50|70|60–90 min|M3|
|**Sum**|**90**|**385**|**50**|**525**|two days|sequential|

### M1 — backend authority and direct controls

Own `internal/docsearch/github/authority.go` (new narrow helper), github.go/cache.go and new authority/confinement test companions. Existing files are166/83/164 lines; companions keep changes reviewable and below 800. Do not change SearchBackend/public JSON shape or add a transport/credential provider.

1. Capture genuine failing-first controls before guards, safely isolated from real GitHub.
2. Constructor Require(AuthNetwork) precedes token/helper/git discovery. Search rechecks before token/repo validation and cache lookup, including NewBackendForTest or manual/preconstructed Backend. Source-order review is part of proof; a cached-result refusal alone must not be misdescribed as a read spy.
3. Recheck Require at each client.Do boundary, including pagination. Test a backend made before confinement and a deterministic policy change between page 1/page 2; prove first allowed request/pagination instrument active then zero next request after confinement.
4. Direct writeCache Require(AuthHomeWrite) precedes HOME resolution/mkdir/temp/rename. Seed a valid cache unconfined, prove a hit and writer activity, then compare full private state after confined Search/write attempts. Include an invalid/missing private HOME control so home-resolution errors cannot precede typed refusal.
5. Propagate only typed confinement cache-write errors from Search, including policy change after an allowed HTTP result but before persistence. Preserve ordinary unconfined optional-cache failure as best-effort.

Acceptance (design AC2/AC3/AC6): constructor, Search/manual/test backend, stale backend, cache-hit and direct writer all typed-refuse with active positive instruments; each HTTP transition is counted; no later side effect or success fallback; unconfined pagination/repo/auth/error/cache TTL/token-safety tests stay green; #1964 authority and local/neural algorithms remain zero-diff.

Checkpoint: `go test ./internal/docsearch/github -count=1`, focused lint/boundaries. Snapshot tests use private HOME/USERPROFILE/home via SetHomeDir plus private XDG trees, names/types/content hashes and supported metadata, not only known device/cache filenames. No parallel env-mutating tests. Commit once accepted during authorized execution.

### M2 — actual CLI, local compatibility and safe hints

Own cmd/ailang/docs_search.go plus a new docs_search_confinement_test.go and small portable helper testdata. Do not grow the existing 497-line command into a broad refactor. Reuse shared buildAilang once and existing bounded subprocess/home helpers.

- Drive the fresh real CLI in a private CWD whose discovery parents contain neither docs nor design_docs. Missing token and dummy-present token must return canonical policy refusal, zero helper/request counts and unchanged private HOME/XDG.
- Portable Go fake gh/git helpers first demonstrate invocation in safe unconfined controls. Fake gh exits without a token; constructor-only/direct-library positives can exercise git without a search. Actual dummy-token CLI red/positive paths additionally use a **loopback deny-CONNECT proxy that never forwards**, so a missing guard cannot accidentally reach real GitHub.
- Set both upper/lowercase HTTP_PROXY/HTTPS_PROXY/ALL_PROXY to the counted local instrument; clear NO_PROXY/no_proxy and other inherited proxy-bypass settings. Prove the proxy really sees an unconfined attempted request before requiring zero requests confined. No real token is read or emitted.
- Strip ambient GOOGLE_CLOUD_PROJECT, OTLP_GOOGLE_CLOUD_PROJECT and OTEL exporter endpoint variables (including traces/metrics variants) in private child environments using existing bounded/home utilities. Held #2009 still has an unconditional root span: **AILANG_TRACE=off is not an export kill switch**. Do not repair that separate issue here.
- Test explicit and auto-discovered local Markdown corpus with default SimHash under confinement; compare parsed stable results and validate JSON. Corpus/HOME/XDG and helper/proxy counters stay unchanged. Missing explicit --path stays a local error without fallback.
- Preserve disabled-fallback local failure for --no-github/env disable, but use only local-path/operator remedy. Canonical confined fallback must not suggest authenticating or unsetting controls. Unconfined auth hints remain.

Acceptance (AC1/AC4/AC6): actual CLI matrix is discriminating and hermetic; positive helper/proxy instruments are live; local results/flags preserved; typed refusal/safe hints, zero side effects, Windows executes the assertions. No new JSON error envelope.

Checkpoint: `go test ./cmd/ailang -run 'TestDocsSearch' -count=1` after `make build`; include existing TestDocsSearch_StillDesignDocSimHash. Build caps and subprocess bounds stay at existing helper values (runCLIIn 30s, RunBounded caller-bound); no deadline widening. Local test source fixtures are Markdown/TOML/JSON/Go, not .ail.

### M3 — actual policy-tool dispatch and child

Add a focused cmd/ailang policy-tool docs-search process companion; production policytool code is intended zero-diff. Use a minimal private policy file/ordinary directories, no symlinks. **Do not reuse writePolicyFixture or policytool.newFixture:** both blanket-skip Windows for unrelated restricted-run/symlink reasons. Do not change those old Windows restrictions.

Drive real `ailang policy-tool --policy <privatepolicy> --request-file <privateJSON>` with docs_search query/flags. Parent AILANG_AGENT_POLICY is absent; actual child must still receive Host's pinned policy and refuse missing-corpus fallback before helpers/cache/proxy. Existing ChildEnvCarriesPolicy unit control separately checks exact-one policy path even against a conflicting parent value. No mocked h.run result is evidence for this milestone.

Preserve the two-level contract: **tool process exit 0 means response produced**; a refused child yields valid existing response OK=false/exit_code nonzero/canonical stderr. Local auto-discovered SimHash yields OK=true/child 0 and parseable stdout JSON. Do not change process exit or response schema to fit a test.

Retain the json/limit allow-list, rejection of other flags (including neural/path), existing unknown-field decoding and output byte caps. No env request facility is added; do not redefine handling of otherwise ignored known request fields. The tool schema has no --path, so allowed-local test creates a discoverable corpus in its private inherited CWD. All new fallback/local assertions execute on Windows, using .exe/path helpers and the M2 active instruments/full snapshots; no POSIX blanket skip.

Acceptance (AC5 plus AC1/AC4): actual tool→actual child path, policy propagation, failure/success envelope, no admission/output-cap changes, no HOME/XDG/corpus mutation or helper/network effect. Existing policy-tool environment/allow-list/output controls remain green.

Checkpoint: focused new real CLI tool tests plus `go test ./internal/policytool -run 'TestPolicyTool_(ChildEnvCarriesPolicy|CLIOutputCapIsThePolicys|JSONOverCapFailsLoudly|CLIPathBearingOptionsRefused)' -count=1`. Existing symlink-only tests may retain their historical skips; new authority cases must not inherit them.

### M4 — docs, final gates and independent handoff

Update only approved scoped contract in docs/docs/reference/cli.md, docs/docs/guides/semantic-search.md, SECURITY.md and `changelogs/unreleased/YYYY-MM-DD-docs-search-confinement.md` opening with `### `. Explain GitHub-only confinement, cached-result refusal, local/operator remedy, preserved local SimHash and explicit neural/cache-management limits. Do not claim universal docs-mode confinement or a new release.

Collect a single final `make build`, `make test-changed` (changed packages + core), relevant focused policy-tool controls, `make lint`, `make check-boundaries`, `make check-home-isolation`, `make check-file-sizes`, `make check-changelog`, `make fmt-check`. No full local make test, repeated broad gate or policy/timeout weakening. No examples manifest change is planned, so verify-examples/stdlib regeneration are not added without actual affected scope.

Map AC1–AC7 to test names, immutable source, real positive/negative counters, state snapshots and logs. Qualify any docs-only integration after the tested source. Independent evaluator plus separately authorized exact-head PR Linux/Windows CI supplies broad readiness; executor does not self-score. Root owns PR/merge and issue disposition; milestone commits use Refs #1965 only.

If unchanged #1926/#2013 infrastructure blocks broad CI, preserve actual failure and report to root. This sprint does not execute that owner, touch its card, retry #2013 or bypass its gate. No unrelated source repair is implied by this plan.

## Example, syntax and completion coverage

No .ail files or new language examples are planned. The approved host-Go boundary is demonstrated by controlled Markdown corpora and actual CLI/tool process fixtures. A standalone examples/runnable program cannot enforce constructor/credential/HTTP/cache boundaries and is not required by this design.

| Fixture | Contracts | Effects | Inline AIL tests |
|---|---|---|---|
|Markdown corpus + direct library controls|N/A host-Go authority, asserted by Go tests|N/A no AIL execution|N/A Go assertions inspect errors/counters/state|
|CLI/policy-tool TOML+JSON and Go helpers|N/A host protocol boundary|N/A no product .ail program|N/A real process/JSON/state assertions|

Syntax prompt gate is not applicable because no .ail creation/edit is planned; if a new .ail need appears, load current prompt and document why it belongs before adding it. Do not introduce product .ail scope to manufacture an example.

## Schedule, success and current validation

Original schedule: Day1 M1 then M2; Day2 M3 then M4, with CI/evaluation waiting separate. Estimated totals525 and sequential dependencies are unchanged; github_issues is exactly[1965]. Registry decisions are populated none, not placeholders. The JSON now records actual authorized execution and milestone results rather than the initial null planning state.

Success is every approved boundary and real process matrix with live positive instruments, no confined credential/cache/HTTP/write behavior, unchanged local/unconfined behavior, no new policy/scope, green required actual gates and independent review. Do not create a new global coverage threshold; measure named branch/control coverage and actual effects instead.

Planning validation is schema/IDs/LOC sums/dependency DAG/pending state and local documentation links, not product tests. create_sprint_json.sh was inspected by root and is deliberately **not run** because it imports/reads messages. JSON is manual from the documented schema with exact issue linking, validated by the existing sprint-executor validator. Its “ready for execution” text means schema-ready only, never user authorization. No messages, coordinator, provider review, implementation, PR or push occurred during planning.

## Execution checkpoints

- Approved draft bank: a33ee6bf0; actual dev6d0d3a9da normally integrated before source work.
- M1:9eee44912, typed constructor/Search/per-request/cache-writer guards; direct RED→GREEN and full backend compatibility.
- M2:3630ec86c, safe local/operator hints and real CLI controls; real auth/unset-hint RED→GREEN.
- M3:5a1a21267, actual policy-tool→actual child controls and unchanged compatibility contracts; pre-guard overlay RED/current GREEN.
- M4: scoped user docs and [acceptance/evidence map](m-docs-search-fallback-confinement-validation.md); actual final gate outcomes tracked in JSON/handoff. Independent acceptance and CI stay separate.

The implementation uses more fixture code than the original estimate to prove live portable helpers, never-forward proxy, full private-state snapshots, both token arms, actual process/child distinctions and late transitions. No production policy or scope was expanded to meet the tests. No .ail syntax/examples gate was triggered; no product .ail was edited.

## Execution sizing and friction

Scoped source/test/user-doc churn versus accepted dev565fc1b04 is671 lines:
47 in production Go files (including7 lines of CLI-reference prose),589 tests/portable
helper and35 user docs/changelog. Milestone totals are295/255/78/43; plan/design/state
and this validation report are recorded separately. Original estimate was525.
The difference is portable positive instrumentation and complete actual process/private-state
controls, not additional product semantics. The execution was sequential in one attended
session, with independent read-only preflight; no parallel-speedup/daily-velocity claim.

The actual final affected-package gate caught CLI reference drift after a direct mirror
edit. The same approved prose was moved to the existing authoritative generator and
`make docs-cli` regenerated its mirror. Original failure remains preserved; the corrected
focused gate passes. No gate/threshold/deadline changed. A preliminary local-result test
also exposed existing macOS /var versus /private/var spelling; compare canonical real
files while preserving production path rendering. Both are workflow/test corrections,
not changes to the approved confinement semantics.

Final local build and affected/core/structure gates pass. Final backend counters use atomic handler/test synchronization, verified by a focused package rerun; production behavior is unchanged. All four implementation milestones are complete. JSON stays in_progress pending independent acceptance and exact-head PR CI; no self-score or merge is performed.
