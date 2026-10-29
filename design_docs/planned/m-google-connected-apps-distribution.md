# M-GOOGLE-CONNECTED-APPS: Add Gemini and Search to AILANG Distribution

**Status**: Planned — proposed design, awaiting user approval; implementation and submission have not begun.
**Date**: 2026-10-29
**Target**: Next distribution increment after approval; release version TBD. Source baseline: v0.55.0, `65253e86e`.
**Priority**: P1 — reuse existing connector and plugin investment in a new distribution channel.
**Estimated**: 5–8 engineering days, excluding preview admission, vendor review and any separately scoped protocol changes. Sprint planner must refine after the feasibility results.
**Dependencies**: Google preview access; existing `serve-api` MCP, `sunholo/mcp_oauth`, Parse connector and hosted executor. File upload depends on the existing MCP file-handoff work, not a second implementation.
**Routing**: Distribution documentation/configuration and service/package adapters. A measured shared protocol defect routes to an AILANG fix; motoko core is outside scope ([PROGRAM](../PROGRAM.md)).

## Problem Statement

AILANG already reaches developers through Claude Code and OpenAI Codex plugins, and hosted users through MCP connectors. We also have a Gemini Enterprise executor integration. Google's Connected Apps Platform adds a related channel: the consumer Gemini app and Search, using a remote MCP service and Google's registration/review process [G1–G3].

We should reuse those investments while keeping four different deliverables explicit: an installable coding-agent plugin, a hosted MCP service, a vendor directory listing, and an A2A agent. Possessing one does not establish availability through the others.

The existing distribution umbrella covers Claude/OpenAI [R1]; this document adds the Google consumer channel. It does not replace the umbrella, redo OAuth, or absorb the Gemini Enterprise/A2A implementation [R2]. There is no measured Google Connected Apps compatibility result yet.

### Existing Channels and Complementary Roles

| Channel | Existing asset and evidence | Relationship to this proposal |
|---|---|---|
| Claude Code / OpenAI Codex development | AILANG Bootstrap; current repository README documents skills and local MCP tools [R3] | Retain local development and repository workflows. Google registration does not install these skills or hooks. |
| Parse plugin packaging | `docparse-skill/plugins/ailang-parse/plugin.json` has `com.openai` metadata; `mcp.json` points to the hosted OAuth connector [R4] | Reuse brand assets, capability descriptions, examples and the hosted service; prepare Google console fields separately. Manifest presence is not proof of directory publication. |
| Claude / ChatGPT hosted connectors | Directory-ready design, submission kit and connector guide [R1, R5] | Reuse the service's listed MCP projection, OAuth and file/UI components. Historical gap lists require revalidation. |
| Gemini Enterprise | Dev executor service and an observed successful GE connection are documented in the runbook; OAuth variant is also documented as deployed [R2] | Reuse executor tools and sandbox. GE's IAM service-agent arrangement is a distinct deployment path. |
| Hosted AILANG docs | Public docs MCP, with a documented tool catalog [R6] | Include useful docs in the AILANG developer offering; avoid a redundant third Google listing initially. |
| Google Connected Apps | New channel proposed here | Two distinct product candidates: AILANG Parse, then AILANG developer tools. Test Gemini first, then establish Search availability independently. |

**Impact:** Parse users gain another place to invoke document tooling; AILANG users can discover and use language tools without first installing a local plugin. Existing plugin users retain local workflows and confidentiality choices.

## Goals and Success Metrics

**Primary goal:** Establish a repeatable Google Connected Apps distribution path for existing AILANG services, with Parse as the first pilot and the AILANG executor as the reuse test.

1. Record a Gemini preview result for each product: actual client sign-in, discovery and three successful advertised tasks, or a specific externally blocked/rejected result.
2. Reuse existing service implementations, tool names and OAuth package. Each necessary compatibility change has a measured failure, named owner and regression evidence.
3. Publish a channel matrix that distinguishes packaged, deployed, client-tested, submitted, approved and published states, per product and Google surface.
4. Validate a complete user-document path before advertising uploaded-file support; record byte integrity and the resulting structured output on a known fixture.
5. Preserve existing Claude/Codex/ChatGPT/GE behaviour through the relevant baseline checks; Google-specific changes must not weaken shared authentication or executor policy.

Engineering completion, submission and public launch are separate outcomes. A preview-only pilot must remain labelled preview; review delays do not count as a published distribution channel.

## High-Impact Decisions

| Decision | Why high impact | Chosen by | Deadline | Change cost |
|---|---|---|---|---|
| D1: Add Google as a channel of the existing MCP distribution work; Parse pilot first, executor second | Prevents parallel connector stacks and duplicate product listings | human, proposed here | design | high |
| D2: Register distinct Parse and AILANG developer products, each with one stable hosted MCP endpoint; no docs-only third listing initially | Determines branding, category, tooling scope and review | human | design | med |
| D3: Use user-delegated OAuth for account-backed services, with existing issuer/token boundaries | User identity and quota must survive a new client; GE IAM is not a consumer auth contract | human | design | high |
| D4: Gemini preview first; Search becomes a separately validated expansion | Preview access and available categories/surfaces are external dependencies | human | design | med |
| D5: Preserve shared refresh rotation and teaching behaviour until a measured Google requirement is resolved | A broad workaround could regress security or existing clients | human, based on feasibility evidence | before compatibility changes | high |
| D6: Use MCP directly; keep A2A delegation in the existing GE design | Prevents an unverified second protocol path from blocking distribution | human | design | med |

### Design Freeze

Before sprint execution, approve D1–D4 and D6, including the account that owns the Google listings. D5 establishes a constraint, not permission to weaken shared behaviour. A proposed refresh-policy or teaching-surface change must return with evidence and a scoped design amendment before implementation.

- [ ] Approve reuse architecture and product sequence (D1).
- [ ] Approve two product identities; confirm available categories before creating entries (D2).
- [ ] Approve account-linking approach and production owner (D3).
- [ ] Approve Gemini-first rollout with Search separately tracked (D4).
- [ ] Ratify compatibility constraints and escalation boundary (D5).
- [ ] Keep A2A outside this channel's initial acceptance criteria (D6).

## Solution Design

### Architecture

```text
Local coding agents                         Hosted assistant clients
Claude Code / Codex                         Claude / ChatGPT / GE / Gemini / Search
        |                                                |
Native plugin + skills                       Remote MCP + client-appropriate auth
        |                                                |
Local CLI / local MCP                        Existing service implementations
                                             |                         |
                                      AILANG Parse               AILANG developer tools
                                      document engine            docs + check/run/eval
                                             |                         |
                                      Own account/quota          Own issuer + sandbox policy
```

Google receives the same service capabilities through remote MCP. Its Connected Console supplies the product registration, tool snapshot, tester access and publication state [G3]. Local plugin packaging remains host-specific. An OpenAI plugin ZIP or Claude marketplace manifest is useful source material, not the Google submission format.

The baseline is configuration, submission materials and tests. Do not pre-allocate a new CLI command, Google-only server fork, broad registry abstraction or new runtime semantics. If validation exposes a shared defect, amend this design with the exact change before scheduling that work.

### Product A: AILANG Parse

Candidate endpoint: `https://docparse.ailang.sunholo.com/mcp/connect/` [R4–R5]. Reuse its account linking, parser/generator engine, tool annotations, account quota and file handling. Inventory the live advertised tools before selecting the initial Google scope; the October 6 submission kit is historical evidence, not today's pass certificate.

Proposed initial capabilities: inspect document structure/annotations on verified formats; convert a supported input to a supported output; estimate processing requirements. Generated-file delivery, conversion preservation and uploads require their own end-to-end evidence. Do not repeat old claims about footnotes, formulas or benchmark superiority: the positioning audit records caveats [R7].

A confidential/local-only document stays on the existing local CLI/plugin route. A Google hosted interaction must clearly disclose when content is sent to our service; an upload must never be implied to remain local.

### Product B: AILANG Developer Tools

Reuse `executor_tools/`, the docs tools and the OAuth executor deployment described in [R2]. Keep the existing vocabulary (`ailang_prompt`, `ailang_check`, `ailang_run`, `ailang_eval`, `ailang_builtins`). Hosted arguments carry code; local plugin tools can use local paths. Tool-name reuse is not a claim that the schemas are identical.

Preserve the existing executor policy and supervision. Public rollout depends on the executor design's production authentication, per-user rate limiting and deployment readiness. A dev sign-in restricted to `sunholo.com` can establish feasibility but does not establish public availability.

Include docs lookup in this product rather than publishing a highly similar docs-only Connected App. If Google rejects execution as a category/use case, record that ruling and revise the product scope before claiming a docs-only alternative is approved.

### Protocol and Client Boundaries

| Layer | Reuse | Google-specific validation |
|---|---|---|
| Remote MCP / Streamable HTTP | `serve-api` tool discovery, schemas and invocation [R5] | Negotiate with the actual Google client; capture protocol version, initialization and required metadata. Similar transport does not prove every extension works. |
| Tool metadata | Existing titles, behaviour hints, schemas and error results | Verify synced tools, confirmation settings and model routing. Google's guidance recommends server instructions and clear domain descriptions [G2]. |
| OAuth | `sunholo/mcp_oauth`, protected-resource metadata and verifier [R5, R8] | Client registration, exact callbacks, PKCE, refresh and unlink/revoke. Google console documents CIMD/DCR and pre-registration [G3]. |
| MCP Apps / files | Existing widget/file-handoff work [R5, R9] | Widget rendering, CSP, host calls, direct upload and downloadable results. Test each claimed path. |
| A2A | Existing service/GE work retains its own contract [R2] | No A2A integration is established by the Connected Apps guides reviewed here [G1–G2]. Do not register an Agent Card as an MCP endpoint. |

### Feasibility Questions With Stop Conditions

**F1 — Access and category.** Establish preview admission, developer-account ownership, available markets and categories. Google fixes a category at app creation [G3]. The linked public category guide returned 404 during research, so category eligibility for Parse and developer tools is unresolved. Do not guess a category to advance the form.

**F2 — Account linking.** Test discovery, actual client registration, first sign-in, a protected call, two refresh cycles, reconnect and revocation. Confirm that the Google client supplies a CIMD document our issuer accepts; its mere support in Google's console is insufficient. A pre-registered-client alternative is only usable if our service supports it. Never add wildcard callbacks or log tokens to make the test pass.

**F3 — Refresh policy mismatch.** Google's persistent-linking guidance asks for a stable refresh token, while the executor adapter explicitly delegates refresh rotation to `sunholo/mcp_oauth` [G4, V4]. Measure whether Google accepts rotated tokens and ask for clarification if necessary. Retain shared rotation by default. If stable tokens prove mandatory, propose an explicit client-specific package policy with expiry, revocation and replay-risk analysis; do not silently change every client's security model. This dependency can block publication while the remainder of the pilot proceeds.

**F4 — Teaching content and routing.** Google's policy disallows system instructions in app responses and references to other tools/new rules governing them [G5]. `ailang_prompt` delivers teaching content, and the GE connector description contains workflow instructions [R2]. Review exactly what Google ingests and returns. Do not copy GE's instructions or plugin hooks wholesale. Test whether language reference material is acceptable as tool data; if unclear, obtain a policy ruling or explicitly narrow the initial tool set. A new initialize/server-instructions option is a conditional shared-framework change, not assumed to exist or required in advance (V5).

**F5 — User files and outputs.** Do not assume a Gemini attachment reaches an MCP tool, or that Google implements OpenAI's `openai/fileParams`. Start with a controlled sample and URL. Then validate the existing upload widget/file-reference path against Google. Verify bytes by digest before parsing, account ownership, size limits and cleanup; check the expected annotations/tables from a fixture. If uploads fail, label the pilot sample/URL-only and keep upload acceptance open. Never serialize a file through model-generated base64 or let a failed tool silently become an assistant-authored substitute parser.

**F6 — Tool and request budgets.** Record actual Google timeouts/output limits rather than importing Claude's limits. Exercise one bounded parse, conversion/download, check, run, named policy refusal and quota refusal as applicable. Keep Google's metadata snapshot aligned with the deployed tool set. A compatible endpoint can still fail product review if advertised tasks time out.

### Registration, Release and Channel Evidence

Create a Google submission kit alongside the existing Claude/OpenAI kit. It contains two product sections, proposed category, endpoint/issuer, ownership verification, brand assets, capability text, prompts, privacy/terms/support URLs, review-account instructions and evidence links. Google requires a domain challenge under `/.well-known/google-connected-apps`, direct HTTP 200, and explicit preview/review/publication steps [G3, G6]. A company account administrator owns program terms acceptance.

Track per-product and per-surface state as: `candidate → endpoint-verified → client-tested → submitted → approved → published`, with independent flags/reasons for preview-only or blocked. A console screenshot or test transcript is evidence for its particular step, not the next one. Record release/image/package versions, test date, tester, tool-list snapshot and source URLs. Keep secrets and reviewer credentials outside committed artifacts.

Use existing service usage logs and Google console metrics. Track successful calls, latency, errors, quota/compute consumption and link/refresh failures. Attribute the Google channel only from a verified client/registration identity, not from a model-supplied argument. Do not retain document bodies, bearer tokens or code in distribution evidence.

Roll back by disabling the Google listing/tester exposure or reverting its compatible service release. Do not disable existing vendor connectors or replace the GE IAM deployment to roll back this channel.

## Implementation Plan (After Approval)

| Phase | Deliverable | Owner | Estimate |
|---|---|---|---|
| M1: inventory and admission | Channel matrix, current manifests/versions, access/category decision, feasibility dossier F1–F6 with observed vs pending separated | AILANG distribution owner; product owners validate their services | 1–2 days |
| M2: Parse Gemini pilot | Existing connector linked in Gemini; three working tasks; file/output paths explicitly graded; compatibility fixes only after scoped amendment | `docparse` service and `ailang-parse` product owners | 1.5–2.5 days |
| M3: executor reuse test | Existing OAuth executor linked; docs/check/run workflows and sandbox/quota refusals verified; production readiness delegated to its existing design | AILANG executor owner; `ailang-multivac` owns infra | 1–1.5 days |
| M4: submission and rollout | Google kit, public setup/troubleshooting docs, evidence-backed matrix, authorized review submission and independently validated Search status | Distribution owner + company account operator | 1.5–2 days |

Search availability is an expansion checkpoint, not a reason to call a Gemini preview public. Preview admission and vendor review can pause the rollout without preventing completion of preparatory artifacts. A material OAuth, framework or file-transfer gap requires a revised estimate and design amendment rather than an unbounded M2.

### Files and Repositories

| Location | Proposed work | Approximate size |
|---|---|---|
| `ailang/docs/docs/guides/mcp-connectors.md` | Google setup/compatibility and evidence labels | +80–140 lines |
| `ailang/docs/docs/guides/getting-started.mdx` | Link channel matrix; retain native plugin instructions | +10–20 lines |
| `ailang/docs/internal/gemini-enterprise-executor-mcp.md` | Cross-link consumer runbook; document distinct auth path | +10–20 lines |
| `ailang/design_docs/planned/m-google-connected-apps-submission-kit.md` (new, during implementation) | Two-product checklist, source dates, proposed copy and evidence references | 150–250 lines |
| `ailang/docs/docs/guides/distribution-channels.md` (new) | Product × host × package/endpoint/status matrix and canonical setup links | 70–120 lines |
| `docparse-skill` | Reuse existing assets/manifests; add Google channel links to product docs after validation | +10–30 lines; no Google plugin manifest presumed |
| `ailang_bootstrap` | Link hosted option after validation; preserve local plugin/extension workflows | +10–30 lines |
| `docparse`, `ailang-parse`, `ailang-multivac` | Verification route, product docs/listing assets and infra configuration as required | Scope after M1 inventory; owners approve in their repos |

No Go or AILANG language changes are baseline deliverables. Existing `ailang mcp check --target both` is a useful cross-vendor baseline, **not a Google certification** (V6). Add a Google-specific checker target only if M1 identifies stable, machine-checkable requirements worth maintaining; that is a scoped amendment, not a new default obligation.

## Examples and Acceptance Criteria

These are proposed test prompts, not claims of existing availability. Source documents and exact expected results belong in the submission kit.

| Journey | Prompt shape | Required evidence |
|---|---|---|
| Parse structure | `@AILANG Parse list the tables and reviewer comments in this sample document.` | Actual `mcpParse` call and expected structure/annotations, not just a plausible model answer |
| Parse conversion | `@AILANG Parse convert the document at this test URL to Markdown.` | Actual tool result; downloadable artifact if the claimed workflow produces a file |
| Parse planning | `Estimate processing requirements for this PDF using AILANG Parse.` | Actual estimate and clear distinction between deterministic/AI processing where applicable |
| AILANG authoring | `@AILANG help me write and check a small AILANG program.` | Reference lookup/teaching path accepted by Google, actual check result and bounded run |
| AILANG errors | `@AILANG explain and fix the type error in this program.` | Named checker diagnostic, repaired code and successful recheck |
| AILANG evaluation | `Evaluate a simple arithmetic expression using AILANG.` | Actual `ailang_eval` result from the hosted executor |

- [ ] AC1: Channel inventory distinguishes local plugins, hosted MCP, A2A and directory state; links to existing Claude/OpenAI work and the GE runbook.
- [ ] AC2: Each product has a dated Google access/category determination and F1–F6 results, including unresolved external dependencies.
- [ ] AC3: Parse passes three advertised journeys in Gemini preview; user-file support is separately passed or explicitly withheld from listing copy.
- [ ] AC4: Executor passes three advertised journeys using the existing engine and sandbox; invalid-auth, policy and quota refusals are demonstrated without executing forbidden work.
- [ ] AC5: Account linking, two refresh cycles, disconnect/revocation and reconnect are verified per product; any refresh-policy change has separate approved scope.
- [ ] AC6: Existing affected-client checks pass. Metadata/auth changes retain the relevant Claude/OpenAI checks; GE keeps its IAM path. All tests required by any implementation changes pass.
- [ ] AC7: Submission kit and documentation are updated; every advertised capability has an evidence link. Draft, submitted, approved and published states are never conflated.
- [ ] AC8: Search has an independent availability/test result, or an explicit pending/unsupported status with its blocker. A Google-wide claim requires a successful Search result.
- [ ] AC9: Before public rollout, production owners confirm quota/rate limits, retention, support and rollback. Actual publication is recorded separately and requires authorization for the selected products/surfaces.

## Testing Strategy

Use MCP Inspector and Gemini Developer Mode for actual host behaviour [G2]. Reuse existing `executor_tools/smoke.sh` and `executor_oauth/test_oauth.sh`; read their coverage before adapting them. The executor smoke script invokes execution and must target an approved test environment. It deliberately avoids the feedback write tool; do not send feedback as a read-only check.

Maintain a small, deterministic fixture set with known document annotations/tables and known checker/evaluation outputs. Test invalid tokens and wrong-account file references as well as success. For every shared code change, add only the regression tests justified by the measured mismatch; rerun relevant existing auth, schema and policy tests. Documentation-only work requires link/metadata validation rather than a full compiler test run.

Capture sanitized client/server evidence and exact versions. An MCP Inspector pass does not prove Gemini compatibility, and a Gemini result does not prove Search compatibility. Do not claim live success from this design session: no client or deployment smoke was run.

## Verification Log (2026-10-29)

| ID | Claim / check | Evidence and result |
|---|---|---|
| V1 | Existing Claude/Codex distribution is reusable context | Read current `README.md` Quick Start; inspected local `docparse-skill` plugin and MCP manifests. Packaging exists; publication/client behaviour was not tested. |
| V2 | GE executor is an existing asset | Read GE runbook and design [R2]. Runbook records successful IAM calls and deployed OAuth dev variant; this session did not independently re-probe cloud state. Older design header lags the OAuth runbook. |
| V3 | Listed MCP, auth and UI serving have existing implementations | Read connector guide [R5] and `internal/apiserver/mcp.go` constructor: separate listed projection and UI registration. Citation supports reuse; no new `.ail` syntax is proposed. |
| V4 | Refresh rotation mismatch is real enough to require a spike | `executor_oauth/oauth.ail` header delegates rotation to the package; token route calls `refreshTokens`. Google guidance [G4] describes stable tokens for persistent linking. Acceptance by Google remains untested. |
| V5 | Server-instruction metadata is an investigation item | Read `newMCPServer` in `internal/apiserver/mcp.go:63`: shown options set capabilities; no instruction field is set in that constructor. Other host surfaces were not exhaustively audited; this is not a claim that a shared extension is absent everywhere. |
| V6 | Existing readiness command is not Google-specific | `ailang mcp check --help` lists `anthropic`, `openai`, `both`; command warned the installed binary may be stale. Reconfirm against source/built binary before changing it. |
| V7 | Scope distinct from existing distribution designs | Read directory-ready umbrella, kit, connector guide, GE design and Parse positioning. Searched `design_docs`, `docs`, README for `connected.apps`, `connected-apps`, `connected console`: no matches before creation. Their existing concerns are cited rather than redesigned. |
| V8 | Related-doc search cannot certify duplication by neural score | Ran both streams with `ailang docs search --neural --timeout 15s --limit 5 --json`. Both explicitly report `embedding_model: fallback-simhash`, zero embeddings and irrelevant ranked matches. Read top planned/implemented results; use targeted search and related-doc review instead of treating fallback rank values as neural thresholds. Scaffold script repeated the fallback. |
| V9 | External requirements are source-backed, not live compatibility | Read G1–G6 during this conversation. Public category-guide link returned 404. Console admission, tool execution, uploads, OAuth and publication remain untested. |

## Risks and Mitigations

| Risk | Mitigation |
|---|---|
| Preview/category access is unavailable | Finish inventory and kit; record exact blocked state; do not promise review dates or guaranteed distribution. |
| Google-specific auth behaviour weakens existing clients | Test existing package first, isolate any justified client policy, and require approval for a semantic auth change. |
| Consumer client cannot deliver files or teaching content as assumed | Make F4/F5 explicit feasibility results; limit advertised tools/workflows to observed successes. |
| Published tool snapshot drifts from deployment | Record snapshot/release identities; version and re-review changes per Google's workflow. |
| New distribution brings unbounded compute or hidden data transfer | Keep account quotas and executor bounds; production gate belongs to service owners; disclose hosted processing and retain local alternatives. |

## Axiom Compliance

Canonical reference: [Design Axioms](../../docs/docs/references/axioms.mdx).

| Axiom | Score | Justification |
|---|---|---|
| A1: Determinism | 0 | Existing service semantics retained; Google routing is outside deterministic execution claims. |
| A2: Replayability | +1 | Versioned fixtures, tool snapshots and sanitized outcome evidence make compatibility results reproducible. |
| A3: Effect Legibility | 0 | Existing declared effects preserved; hosted file transfer is disclosed. |
| A4: Explicit Authority | 0 | Existing token verification, account/file ownership and executor policy remain required boundaries. |
| A5: Bounded Verification | +1 | Small per-product journeys and explicit failure cases establish channel-specific acceptance. |
| A6: Safe Concurrency | 0 | No concurrency semantics changes. |
| A7: Machines First | +1 | Reuses typed tool schemas and structured results across clients. |
| A8: Minimal Syntax | 0 | No language syntax added. |
| A9: Cost Visibility | +1 | Per-account quotas, processing estimates and channel usage are rollout evidence. |
| A10: Composability | +1 | One service participates in multiple host-native distribution channels. |
| A11: Structured Failure | +1 | Unsupported, blocked, refused and untested states are explicit; no silent parsing/auth fallback. |
| A12: System Boundary | 0 | External Google review/client behaviour remains an explicit boundary. |

**Net score: +6.** Hard constraints: A1/A3/A4/A7 have no negative scores. This supports design review, not implementation approval.

## Non-Goals and Deferred Decisions

No new parser/type/effect/runtime semantics, motoko core changes, Android app/AppFunctions implementation, generic Google agent platform, new A2A gateway, unified vendor packaging standard, or automatic approval/submission pipeline. This session creates the design only; it does not register apps, accept terms, send messages, deploy services or publish listings.

The implementer may choose fixture organization, screenshot filenames and submission-kit layout. Proposed wording can be refined within observed capabilities. Company account ownership, category choice, refresh-policy changes, public execution access and publication scope remain explicit operator decisions. Quorum is recommended for the external auth/policy premises before planning; it has not been run.

## Related Documents and Sources

### Existing Work (Reuse and Scope Boundaries)

- **R1:** [M-SERVEAPI-DIRECTORY-READY](v0_51_0/m-serveapi-directory-ready.md) and [Claude/OpenAI submission kit](v0_51_0/m-serveapi-directory-ready-submission-kit.md). Existing shared distribution foundation; this proposal adds a Google-specific channel and validation kit.
- **R2:** [Gemini Enterprise executor design](v0_55_0/m-ailang-executor-gemini-enterprise.md) and [setup runbook](../../docs/internal/gemini-enterprise-executor-mcp.md). Own executor infrastructure and A2A work; this proposal reuses those services and adds consumer distribution.
- **R3:** [Repository Quick Start](../../README.md#quick-start), [AILANG Bootstrap](https://github.com/sunholo-data/ailang_bootstrap). Local plugin channel; local sibling README differed from the main repo's newer installation guidance, so re-read the canonical release before editing plugin docs.
- **R4:** [Parse plugin repository](https://github.com/sunholo-data/docparse-skill), specifically `plugins/ailang-parse/plugin.json` and `mcp.json`, inspected locally. Existing OpenAI metadata and hosted endpoint are reuse inputs, not evidence of public listing.
- **R5:** [MCP connector guide](../../docs/docs/guides/mcp-connectors.md). Reference for existing auth, projection, file and MCP Apps serving features.
- **R6:** [Hosted docs MCP](../../docs/docs/guides/agent-mcp.md). Existing developer knowledge surface.
- **R7:** [Parse positioning audit](v0_53_0/ailang-parse-positioning.md). Scope and claim caveats; does not cover this new Google channel.
- **R8:** [Implemented OAuth package](../implemented/v0_52_0/m-mcp-oauth-package.md). Owns shared account-linking logic; no copied OAuth implementation here.
- **R9:** [File handoff](v0_53_0/m-mcp-file-handoff.md) and [MCP files friction](v0_53_0/m-mcp-files-friction.md). Own file-transfer functionality; this design tests its Google client compatibility.

### Google Primary Sources (Read 2026-10-29; Recheck Before Submission)

- **G1:** [Announcement, 2026-10-28](https://blog.google/innovation-and-ai/technology/developers-tools/connected-apps-one-place/) and [platform overview](https://developers.google.com/connected-apps): phased preview, Gemini/Search distribution and supported integration paths.
- **G2:** [MCP developer guide](https://developers.google.com/connected-apps/mcp): remote service, metadata, MCP Apps and Gemini Developer Mode.
- **G3:** [Register, test and publish](https://developers.google.com/connected-apps/gpc/perform-the-steps): categories, registration, OAuth options, tool snapshots and release states.
- **G4:** [OAuth implementation guide](https://developers.google.com/connected-apps/mcp/oauth): discovery, PKCE, refresh and persistent-linking behaviour. Some registration wording differs from the newer G3 console guide; actual client behaviour must settle compatibility.
- **G5:** [Connected Apps policy](https://developers.google.com/connected-apps/connected-apps-policy): accurate scope, distinct product value, data handling and response-instruction restrictions.
- **G6:** [Prerequisites](https://developers.google.com/connected-apps/gpc/prerequisites): HTTPS/Streamable HTTP, developer access, ownership challenge and submission assets.
