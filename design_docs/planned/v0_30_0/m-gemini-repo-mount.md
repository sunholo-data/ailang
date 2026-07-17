# M-GEMINI-REPO-MOUNT — Managed Agents repository and inline-source mounts

**Status**: PARKED (needs-human-review) — **Phase-1 spike RUN (2026-07-17); core premise REFUTED.**
The documented `repository` + `inline` source mounts **do not exist** on the live Vertex endpoint;
this design cannot be implemented as written and needs a **GCS-backed redesign OR abandonment** (human
scope decision — see the Phase-1 Spike Result section and the mission-iteration-45 report on issue #399).
**Target**: v0.30.0
**Priority**: P1 — mission gap G4; upgrades Gemini from reasoning-only review to in-sandbox verification
**Estimated**: ~~~1 day (≤~250 LOC of Go)~~ — invalidated; a GCS-backed approach is a larger, unscoped lift
**Dependencies**: `managed_agents` executor / M-MANAGED-AGENTS v0.22.0

> **⛔ PHASE-1 SPIKE RESULT (mission iteration 45, 2026-07-17) — PREMISE REFUTED.** Mark authorized the
> ADC-gated live Vertex contract-discovery spike ("yep do the vertex contract spike", #399). It ran
> against the real `interactions` endpoint (project `ailang-dev`, `global`, `Api-Revision 2026-05-20`,
> 14 credential-free probes, all cheap request-validation `400`s — no sandbox provisioned). **Result: the
> live API rejects both `repository` and `inline` source types outright** — `Unsupported environment data
> source type: REPOSITORY/INLINE. Must be one of: [gcs, skill_registry]`. It further requires **network
> egress to be enabled before ANY data source is accepted** (a fresh `{"type":"remote"}` env has egress
> OFF). So every source-mount premise in this doc is refuted, and the encoder/CLI/limit design below is
> moot as written. See the **Phase-1 Spike Result** section for the full VERIFIED-LIVE record and the
> reproducible probe (`internal/executor/managed_agents/managed_agents_live_test.go`,
> `AILANG_LIVE_MANAGED_AGENTS_MOUNT=1`). **The Phase-1 gate the quorum demanded is now satisfied — and it
> returned a NEGATIVE. No Phase-2 code should be written against the refuted contract.**

> **PARK NOTE (mission iteration 44, 2026-07-17).** This doc was authored by the `codex:gpt-5.6-sol`
> designer (G3 designer-rotation live test) and passed through TWO 3-provider quorum rounds
> (`gpt5-6-sol` + `gemini-3-1-pro` + claude controller). Round 1 → BLOCKED (unverified wire contract;
> programmatic silent-fallback hole); a designer revision added the Premise Verification Log +
> contract-discovery-spike-first structure + `CapEnvironmentSources` pre-dispatch gate. Round 2 → still
> BLOCKED: the reviewers hold that a design resting on a DOC-ONLY/ASSUMED external API contract cannot be
> ratified until the **Phase-1 live contract-discovery spike is actually RUN and RECORDED** (their
> proposed fix *is* this doc's Phase 1). **Unblock path (human-authorize):** run the ADC-gated Phase-1
> spike against the live Vertex `interactions` endpoint (repository-only / inline-only / combined POSTs;
> record credential-scrubbed request+response + agent-observed filesystem evidence), update every Premise
> Verification Log row to VERIFIED-LIVE, THEN Phase 2+ may begin. **Additional unincorporated reviewer
> row to add before/with the spike** (gemini round 2, valid): "A mounted repository includes sufficient
> commit history to `git checkout <arbitrary older SHA>` (NOT a shallow `--depth=1` / default-branch-only
> clone)" — if Vertex shallow-clones, the directive must run an explicit `git clone` instead of relying on
> the provider mount. Quorum artifacts: `.ailang/state/mission-quorum/m-gemini-repo-mount-*.json`.

## Problem Statement

AILANG can assign Gemini executor, evaluator, and quorum-reviewer roles through the Vertex AI Managed
Agents executor, but the hosted agent cannot currently see the caller's repository. The interaction
request always creates an empty remote environment, so a Gemini reviewer cannot inspect the worktree,
run `ailang check`, or use repository-local tooling. It can only reason over diff text packed into the
prompt by the current diff bridge.

Current state at HEAD `36cca59a1`:

- `internal/executor/managed_agents/types.go:38` already defines
  `interactionRequest.Environment json.RawMessage` with `json:"environment"`.
- `internal/executor/managed_agents/managed_agents.go:164` hardcodes
  `envRaw := json.RawMessage(`{"type":"remote"}`)` and assigns it to `Environment` at line 170.
  This is the only place the environment payload is built. The surrounding comment at lines 153–163
  documents the `CapRemoteSandbox` no-shared-filesystem limitation and the text-output bridge.
- `internal/executor/executor.go:37` defines provider-agnostic `Task` fields, including
  `Metadata map[string]string` (“Provider-specific options”) and `ExtraEnv map[string]string`.
  There is no environment-mount field.
- `cmd/ailang/exec.go` implements `ailang exec <provider>`. `resolveAgenticExecutorName` at line 317
  maps `gemini` to `managed_agents`; `executeCLI` at line 336 builds `executor.Task` at lines 347–356.
  Flags are registered on its `flag.FlagSet` around lines 63–88, and help text is around lines 677–710.
  No `--env-repo` or `--env-inline-file` flags exist.
- `internal/eval_harness/gemini_evaluator_bridge.go` provides `BuildDiffBundle` and
  `RunGeminiEvaluator`; `internal/eval_harness/managed_agents_bridge.go` provides the existing
  cross-environment text bridge. This prompt-packed path is reasoning-only.

The Managed Agents documentation describes repository and inline sources, public-repository mounting,
per-file limits, and default outbound networking. Those claims have not yet been proven against the live
Vertex `interactions` endpoint used by this executor, and the exact inline wire encoding remains an
assumption. The first implementation phase therefore discovers and records the accepted contract before
any encoder code lands. Once verified, this is the missing plumbing for Gemini to verify the actual code
instead of reviewing only a textual representation of it.

## Premise Verification Log

`VERIFIED-LIVE` means a credential-scrubbed request/response record from the actual endpoint used by
`managed_agents`; documentation examples alone are `DOC-ONLY`. This log starts honestly below and must be
updated by the Phase-1 spike before Phase 2 begins.

**Updated by the Phase-1 spike, mission iteration 45 (2026-07-17). Every DOC-ONLY/ASSUMED row was probed
against the live endpoint; the net result is REFUTED.** The endpoint used is
`aiplatform.googleapis.com` (Vertex) — NOT the `ai.google.dev` Gemini Developer API the original rows
cited; the two contracts diverge, which is exactly the risk the quorum flagged.

| Claim | Original source | Status (post-spike) | Live evidence |
|-------|-----------------|---------------------|---------------|
| A config-object environment accepts an `environment.sources[]` array | `ai.google.dev` docs | **PARTIALLY VERIFIED-LIVE** | The `sources[]` array IS parsed and validated (the API reads each element's `type` and validates per-source `target`, error path `environment.config.sources.target`). But the *contents* below are refuted. |
| Repository sources use fields `type`, `source`, and `target` | `ai.google.dev` "Mount from a source" | **REFUTED (VERIFIED-LIVE)** | `HTTP 400: Unsupported environment data source type: `REPOSITORY`. Must be one of: [`gcs`, `skill_registry`].` There is **no repository/git source type** on this endpoint. |
| Inline sources use the documented field names and an exact content encoding | `ai.google.dev` "Mount from a source" | **REFUTED (VERIFIED-LIVE)** | `HTTP 400: Unsupported environment data source type: `INLINE`. Must be one of: [`gcs`, `skill_registry`].` There is **no inline source type**; inline patch injection is impossible via `environment`. |
| Inline content has a per-file limit described as "1 MB" | `ai.google.dev` source-type table | **N/A (VERIFIED-LIVE)** | Moot — no inline source type exists. Not probed for accept/reject size because the type itself is rejected first. |
| Equality at exactly `1 << 20` bytes is accepted and `1 << 20 + 1` is rejected | unverified | **N/A (VERIFIED-LIVE)** | Moot — no inline source type. |
| A public repository mounts without repository credentials | `ai.google.dev` public GitHub example | **N/A (VERIFIED-LIVE)** | Moot — no repository source type. GCS sources would use GCS/IAM auth, not git-repo credentials. |
| A fresh remote environment has unrestricted outbound network by default | `ai.google.dev` "Network configuration" | **REFUTED (VERIFIED-LIVE)** | `HTTP 400: Network egress is not enabled for the environment. Cannot specify data sources.` A fresh `{"type":"remote"}` env has egress **OFF**, and egress must be enabled *before* any data source is accepted. |
| **[NEW — gemini round-2 row]** A mounted repo has enough history to `git checkout` an arbitrary older SHA (not shallow) | quorum reviewer | **N/A (VERIFIED-LIVE)** | Moot — no repository mount exists to be shallow or deep. A GCS-backed redesign would ship whatever the caller uploads, so history depth becomes a tarball-contents question, not a provider-mount question. |
| **[NEW — discovered]** The only supported `sources[].type` values | live probe | **VERIFIED-LIVE** | Exactly **`gcs`** and **`skill_registry`** (per the two rejection messages). |
| **[NEW — discovered]** Each source requires a `target` | live probe | **VERIFIED-LIVE** | Bare `{"type":"gcs"}` / `{"type":"skill_registry"}` → `HTTP 400: `environment.config.sources.target` is required.` |
| **[NEW — discovered]** `environment.network` is a real config object gating egress | live probe | **VERIFIED-LIVE (name undiscovered)** | `environment.network` accepts params (errors are scoped to `environment.network`), but its egress-enable field is **not** `egress`, `egress_enabled`, `enable_egress`, `enable_internet_access`, or `egress_setting` (all → `Unknown parameter … at 'environment.network'`). The exact param name needs the Vertex Managed Agents environment proto/reference — a follow-up, not blind probing. |

## Phase-1 Spike Result (VERIFIED-LIVE, mission iteration 45, 2026-07-17)

**Authorization:** Mark, on issue #399 — "yep do the vertex contract spike".

**Method:** an env-var-guarded, in-package Go probe
(`internal/executor/managed_agents/managed_agents_live_test.go`, gated by
`AILANG_LIVE_MANAGED_AGENTS_MOUNT=1`) that reuses the executor's own `sendInteraction` + `parseSSE` so
each request is byte-identical to production. It POSTs varying `environment` payloads to the exact live
endpoint the executor uses and records the response. **14 probes were run; all were request-validation
`HTTP 400`s that reject before a sandbox is provisioned, so the spike cost was negligible** (no agent
run, no sandbox — the informative-rejection path). No credentials appear in requests (the ADC bearer
token is added inside `sendInteraction` and never printed); interaction/environment IDs are redacted.

- **Endpoint:** `POST https://aiplatform.googleapis.com/v1beta1/projects/ailang-dev/locations/global/interactions`
- **Headers:** `Api-Revision: 2026-05-20`, `Authorization: Bearer <ADC>`, `Accept: text/event-stream`.

**Verbatim probe transcript (credential-free):**

| Probe | `environment` sent (abridged) | Live response |
|-------|-------------------------------|---------------|
| A repo-only | `sources:[{type:repository, source:<git url>, target:/workspace/ailang}]` | `400 Unsupported environment data source type: `REPOSITORY`. Must be one of: [`gcs`, `skill_registry`].` |
| B inline-only | `sources:[{type:inline, target:…, content:"AILANG_INLINE_SENTINEL_v1…"}]` | `400 Unsupported environment data source type: `INLINE`. Must be one of: [`gcs`, `skill_registry`].` |
| F gcs bare | `sources:[{type:gcs}]` | `400 `environment.config.sources.target` is required.` |
| G gcs+source+target | `sources:[{type:gcs, source:gs://…, target:/workspace/x}]` | `400 Network egress is not enabled for the environment. Cannot specify data sources.` |
| H skill_registry bare | `sources:[{type:skill_registry}]` | `400 `environment.config.sources.target` is required.` |
| I–N egress-enable guesses | `network:{egress\|egress_enabled\|enable_egress\|enable_internet_access\|egress_setting: …}` | `400 Unknown parameter '<name>' at 'environment.network'.` (all six guesses rejected) |

**What is now VERIFIED-LIVE:**

1. **The documented `repository` and `inline` mounts do not exist on this endpoint.** The only accepted
   `sources[].type` values are **`gcs`** and **`skill_registry`**. This refutes the entire mount model
   this doc was built on (git-URL repository mount + inline patch injection).
2. **Data sources are gated behind network egress.** A fresh `{"type":"remote"}` environment has egress
   OFF; any `sources[]` on such an env is rejected with "Network egress is not enabled … Cannot specify
   data sources." Egress must be enabled first, via some parameter under the real `environment.network`
   object (name TBD — see #3).
3. **`environment.network` exists but its egress-enable field name is undiscovered.** Six idiomatic
   guesses were all rejected as unknown params scoped to `environment.network`. Discovering it (and the
   full `gcs` source contract) requires the actual Vertex Managed Agents environment proto/reference,
   not further blind 400-probing.
4. Each source requires a `target`; the server normalizes `environment.sources` to
   `environment.config.sources` internally.

**Implication — this design is refuted, not merely unverified.** "Mount the caller's git repo + inject
the uncommitted diff as an inline file" is not expressible against this API. The nearest real path is a
**GCS-backed mount**: enable egress, upload a repo tarball/checkout (and the diff) to a GCS bucket in
`ailang-dev`, and mount it via a `gcs` source. That is a materially different and larger design (GCS
bucket + upload/lifecycle pipeline, IAM, egress config, tarball assembly) than the ≤250-LOC estimate
here, and it still needs the egress-param + `gcs`-source contract pinned first.

### 🔴 Decision needed (Mark) — scope, not code

The spike did its job (verify-before-build) and returned a decisive negative. Three options:

- **(a) Redesign around GCS-backed mounts.** Highest value (Gemini gains real in-sandbox `ailang check`),
  but a fresh, larger design + a second contract-discovery spike (egress param + `gcs` source fields).
  Would be re-queued as a new design-doc iteration, not a resume of this one.
- **(b) Shelve G4; keep the prompt-packed diff bridge.** Gemini stays a reasoning-only reviewer (as
  today). Zero further cost. The diff bridge already works and `VerificationDegraded` is stamped honestly.
- **(c) Investigate `skill_registry`** as an alternate delivery path (unknown fit; likely for agent
  skills, not repo code — lowest confidence).

Recommendation: **(b) shelve unless mounted Gemini verification proves worth a GCS pipeline** — the
existing bridge covers the review use case, and the mission has cheaper accessibility-queue items. Reply
on #399 to choose. Until then the doc stays PARKED with the contract now honestly recorded.

## Goals

> **NOTE (iter 45):** the Goals/Decisions/Plan below describe the REFUTED repository+inline design and are
> retained only as the record of what the spike invalidated. Do not implement against them. A GCS-backed
> redesign (option a) would rewrite this section.

- Preserve byte-for-byte-equivalent default behavior: a task with no requested sources sends
  `{"type":"remote"}`.
- Add a typed task-level source configuration for one public repository and zero or more inline files.
- Accept `--env-repo <url>[@<ref>]`, `--env-target <dir>`, and repeatable
  `--env-inline-file <path>` flags on agentic `ailang exec` calls.
- Reject invalid or oversized inline files loudly before any Managed Agents request is sent.
- Reject non-empty `Task.EnvSources` centrally when the selected executor does not advertise environment-
  source support, including programmatic coordinator/eval callers.
- Expose the same source configuration at the eval-harness caller seam, allowing a Gemini evaluator or
  reviewer to mount a pinned repository revision and inject the uncommitted diff as inline content.
- Keep the existing prompt-packed diff bridge as the no-mount fallback.
- Cover environment encoding and CLI parsing with hermetic unit tests; require no ADC or live network in CI.

## High-Impact Decisions

| Decision | Choice | Rationale and tradeoff |
|----------|--------|------------------------|
| Provider contract | Run a bounded live contract-discovery spike before designing the wire encoder | Documentation is evidence for intent, not proof of the live Vertex `interactions` contract used here. Repository-only, inline-only, and combined probes must produce a credential-scrubbed request/response record and update the Premise Verification Log to `VERIFIED-LIVE`. Golden tests are written only after that record pins field names, inline encoding, limits, public-repository behavior, and network behavior. If the spike cannot run, implementation remains blocked. |
| Task transport | Add a typed `EnvSources` field to `executor.Task`; do not encode mounts in `Metadata` | A typed field gives compile-time shape, central validation, deterministic JSON, and testable size limits. `Metadata` would avoid expanding the shared task type, but stringly typed JSON or delimiter conventions would hide malformed authority until provider execution and encourage silent fallback. The shared field is justified because mounted execution environments are an executor capability, not a Gemini model parameter. Executors that do not support it must fail clearly if handed non-empty sources rather than ignore them. |
| Capability enforcement | Add `CapEnvironmentSources` plus shared `executor.ValidateTaskCapabilities`; route production dispatch through capability-validating helpers, and advertise support only from `managed_agents` | A central capability gate follows the existing `CapRemoteSandbox` feature-detection pattern and closes the Go API hole for CLI, coordinator, and eval-harness callers without duplicating ad hoc checks across Claude, Codex, OpenCode, Pi, Motoko, or future executors. Empty `EnvSources` is universally valid, so the gate is additive. Direct provider encoding still validates source contents, but unsupported executors fail before execution rather than silently ignore authority. |
| Source model | Use explicit repository and inline variants with constructors/validation | Prevents impossible combinations such as a repository with inline bytes or an inline file without a target. The Go representation may be a tagged struct to keep the patch small, but validation must switch on a closed kind set. |
| Inline-file loading | Read the local file at CLI/eval-harness boundary and place validated bytes plus sandbox target on `Task` | The caller owns local filesystem authority. The Managed Agents encoder should receive content, not reopen arbitrary paths later. This makes the request deterministic and lets the spike-confirmed per-file rule fail before network I/O. The intended bound is `1 << 20` bytes pending Phase 1. Tradeoff: task objects may carry up to the aggregate inline payload size. |
| Repository ref | Parse the final `@<ref>` only after a URL-aware split; retain it as explicit source metadata and pin with the reviewer directive's `git checkout --detach <ref>` recipe | The documented mount source has URL and target fields; this design does not invent an unsupported wire-level `ref` property. The mounted repository is followed by an explicit checkout in the directive. A missing ref is allowed for general CLI use but the eval harness must supply the reviewed SHA. |
| Targeting | Default repository target to `/workspace/ailang`; inline targets default under the repository target using a sanitized basename | Gives G4 a stable working directory while keeping simple calls concise. Absolute sandbox targets supplied by callers remain explicit. Basename collisions or traversal-like targets fail loudly. |
| Provider scope | Register flags globally on `ailang exec`, but reject non-empty environment sources unless the resolved executor advertises/supports them | Shared flag parsing stays simple and future executors can adopt the capability. Silently ignoring mount flags for Claude or API-only execution would violate the no-silent-fallback rule. |
| Fallback | Use the current diff bridge only when no environment source configuration is supplied | Existing callers stay unchanged. Requested mounts never degrade silently to prompt packing; mount construction errors are returned. |
| Environment lifecycle | Create a fresh remote environment per interaction | Matches current behavior and keeps this patch bounded. Reuse and persisted files are deferred. |

## Design Freeze Checklist

- [x] Default request contract remains `{"type":"remote"}` when `Task.EnvSources` is empty.
- [x] Repository mounts are opt-in and restricted to explicit public repository URLs.
- [x] The eval harness requires an explicit pinned ref/SHA for verifying a sprint revision.
- [x] Inline content is loaded and size-checked before the network request.
- [ ] Managed Agents environment wire contract recorded VERIFIED-LIVE by the Phase-1 spike before any encoder code lands.
- [ ] Intended per-file inline semantics are a `1 << 20`-byte maximum with equality accepted and larger
  rejected; freeze the exact unit and equality rule only after the Phase-1 live boundary probe confirms it.
- [x] Invalid source kinds, empty URLs, empty content targets, duplicate targets, and path traversal fail loudly.
- [x] No unsupported repository `ref` field is added to the external JSON contract.
- [x] No mounted-source request silently falls back to an empty sandbox or prompt packing.
- [x] No live-network test is required in CI; the contract-discovery spike is manual and env-var guarded.
- [x] Existing diff-bridge behavior remains available when mounts are absent.
- [x] No parser, lexer, type-system, evaluator-default, or AILANG syntax change is included.

## Deferred Decisions

- Reusing a Managed Agents environment across interactions and defining cleanup/retention policy.
- Aggregate inline-payload ceilings below any API request limit; after Phase 1 this patch enforces only the
  live-confirmed per-file bound, intended to be `1 << 20` bytes with equality accepted.
- Private repository authentication, credential brokering, signed URLs, or secret mounts.
- Generalizing environment sources into a cross-provider capability interface beyond the typed task field.
- Automatically generating a full changed-file overlay rather than supplying explicit inline files or a patch.
- Automatically downloading the AILANG Linux release or running `ailang check` from executor code.
- Selecting Gemini as the default evaluator or adding Gemini to designer rotation before evidence is collected.

## Solution Design

### Overview

Add a small typed environment-source model to `executor.Task`. The CLI and eval harness translate
explicit caller inputs into validated source values. The Managed Agents executor owns the final JSON
encoding because it owns the provider contract. With no values, its builder returns the existing remote
payload unchanged. With values, it returns a remote environment containing repository and/or inline
sources.

The mounted repository supplies the committed baseline. Inline files supply caller-owned content not in
that baseline, normally an uncommitted patch file such as `/workspace/ailang/.ailang-review/diff.patch`.
The reviewer directive checks out the explicit SHA, applies or reads the patch, downloads the appropriate
AILANG Linux release binary, and runs the required checks. Those commands remain directive policy, not
executor behavior.

### Architecture

1. **Provider-agnostic task contract (`internal/executor/executor.go`)**
   - Add `EnvSources []EnvironmentSource` to `Task`.
   - Define a compact tagged `EnvironmentSource` representation with repository URL, sandbox target,
     optional requested ref, inline content, and inline target.
   - Add `CapEnvironmentSources` and a shared `ValidateTaskCapabilities(exec, task)` check.
   - Make the shared production dispatch helpers run that validation before `Execute` or
     `ExecuteStreaming`; non-empty sources sent to an executor without the capability return a clear error.
   - Keep local filesystem paths out of the final task after boundary loading.

2. **Managed Agents payload builder (`internal/executor/managed_agents/`)**
   - Advertise `CapEnvironmentSources` from the Managed Agents executor only.
   - After the Phase-1 spike, add wire-only environment/source structs matching the recorded live contract.
   - Add `buildEnvironmentPayload([]executor.EnvironmentSource) (json.RawMessage, error)`.
   - Return exactly `json.RawMessage(`{"type":"remote"}`)` for an empty slice.
   - Validate every source and encode `{"type":"remote","sources":[...]}` for non-empty input.
   - Encode repository and inline sources using only the field names and inline content encoding recorded
     by the Phase-1 spike, with deterministic ordering matching the caller's source order.
   - Replace only the hardcoded assignment at `managed_agents.go:164` with the builder call and propagate
     errors before `sendInteraction`.

3. **CLI boundary (`cmd/ailang/exec.go`)**
   - Add a repeatable string flag helper for `--env-inline-file`.
   - Add `--env-repo`, `--env-target`, and their names to argument normalization and help output.
   - Parse repository URL/ref without splitting URL userinfo or ordinary path characters incorrectly.
   - Read inline files, enforce the limit, derive safe sandbox targets, and construct task sources.
   - Reject mount flags with `--api-only` or an unsupported agentic executor.
   - Extend `executeCLI` to receive the validated source slice and assign it to `executor.Task`.

4. **Eval-harness seam (`internal/eval_harness/gemini_evaluator_bridge.go`)**
   - Extend `EvalOptions` with an optional mount configuration: repository URL, pinned ref/SHA, target,
     and inline files/content.
   - When configured, pass mount flags through the injected/default runner seam rather than embedding the
     entire diff in the directive.
   - For the default command runner, append `--env-repo`, `--env-target`, and repeated
     `--env-inline-file` arguments.
   - Keep `BuildDiffBundle` and the current reasoning-only directive path unchanged for calls without mount
     configuration. Such verdicts continue to stamp `VerificationDegraded` as today.

5. **Reviewer directive recipe (documentation in this design and help text)**
   - `cd /workspace/ailang`.
   - `git checkout --detach <pinned-sha>` and verify `git rev-parse HEAD` equals the requested SHA.
   - Apply or inspect the mounted inline patch/file overlay.
   - If Phase 1 verifies default outbound access, fetch a pinned Linux AILANG release binary; otherwise the
     directive must use the recorded network configuration or fail explicitly.
   - Run `ailang check` and any bounded repository checks requested by the evaluator directive.

### Implementation Plan

#### Phase 1 — Contract-discovery spike (hard prerequisite)

- [ ] Add a bounded, manual-only live probe guarded by `AILANG_LIVE_MANAGED_AGENTS_MOUNT=1`; require
  working ADC/project configuration and never run it in default CI.
- [ ] POST repository-only, inline-only, and combined config-object `environment` payloads to the exact
  Vertex `interactions` endpoint used by `managed_agents`.
- [ ] Include bounded checks for public-repository mounting without credentials, exact inline file bytes,
  the `1 << 20`/`1 << 20 + 1` boundary, and default outbound HTTPS behavior.
- [ ] Record the accepted serialized requests and responses with authorization headers, tokens, project
  identifiers, interaction IDs, and other credentials removed.
- [ ] Update every Premise Verification Log row to `VERIFIED-LIVE` with the observed positive or negative
  result, rewriting the claim if live behavior differs; pin the exact inline field name and raw-vs-base64
  encoding, and explicitly record any rejected documented shape.
- [ ] **Gate:** If the live probe cannot be run, Phase 2 does not begin; the doc stays blocked pending a
  recorded contract.

#### Phase 2 — Typed source contract, capability gate, and encoder

- [ ] Add `EnvironmentSource` and `Task.EnvSources` in `internal/executor/executor.go`.
- [ ] Add `CapEnvironmentSources`, shared capability validation, and capability-validating production
  dispatch helpers in `internal/executor/executor.go`.
- [ ] Route CLI, coordinator, and eval-harness dispatch through the shared gate so programmatic callers
  cannot silently send non-empty `EnvSources` to a non-supporting executor.
- [ ] Advertise `CapEnvironmentSources` from `managed_agents`.
- [ ] Add Managed Agents wire structs and `buildEnvironmentPayload` in
  `internal/executor/managed_agents/types.go` or a focused `environment.go`, contingent on the recorded
  Phase-1 field names, inline encoding, and limit semantics.
- [ ] Replace the hardcoded environment construction in
  `internal/executor/managed_agents/managed_agents.go`.
- [ ] Return validation/encoding errors before network activity.

#### Phase 3 — CLI plumbing

- [ ] Register `--env-repo`, `--env-target`, and repeatable `--env-inline-file` flags.
- [ ] Add URL/ref parsing, inline file loading, target derivation, and validation helpers.
- [ ] Thread validated sources through `executeCLI` into `executor.Task`.
- [ ] Update `ailang exec` help with the opt-in mount behavior and example.

#### Phase 4 — Eval-harness seam

- [ ] Add optional environment-mount configuration to `EvalOptions` and runner input.
- [ ] Pass mount flags from `DefaultGeminiRunner` when configured.
- [ ] Preserve `BuildDiffBundle` prompt-packing when no mount is configured.
- [ ] Document that mounted verification requires the evaluator directive to pin, overlay, and check.

#### Phase 5 — Hermetic verification

- [ ] Add environment-builder table tests for empty, repository-only, inline-only, and combined payloads.
- [ ] Add rejection tests for oversized inline content and malformed/duplicate targets.
- [ ] Add a shared-dispatch unit test proving a non-supporting fake executor receives no call and returns a
  clear error when `Task.EnvSources` is non-empty; empty sources continue normally.
- [ ] Add a CLI flag-parsing test covering repository ref, target, and repeated inline files.
- [ ] Add regression tests proving no-source CLI/eval calls produce the unchanged remote environment.
- [ ] Keep the recorded Phase-1 probe opt-in and out of CI; hermetic golden expectations must derive from
  its sanitized live contract record, not from the pre-spike assumption.

## Files to Modify/Create

| Path | Change | Estimated LOC |
|------|--------|---------------|
| `internal/executor/executor.go` | Add typed environment-source contract, `Task.EnvSources`, `CapEnvironmentSources`, and shared pre-dispatch capability validation/helpers | +40–55 |
| `internal/executor/executor_test.go` | Prove non-supporting executors reject non-empty sources before execution and empty sources remain compatible | +20–30 |
| `internal/executor/managed_agents/types.go` | Add Managed Agents environment/source wire structs if kept beside request types | +20–35 |
| `internal/executor/managed_agents/managed_agents.go` | Advertise environment-source support and call the spike-confirmed payload builder instead of hardcoding `{"type":"remote"}` | +10–18 |
| `internal/executor/managed_agents/environment.go` | Optional focused builder/validation file if not placed in `types.go` | +45–70 |
| `internal/executor/managed_agents/managed_agents_test.go` | Capability advertisement, payload-builder, and unchanged-default regression tests derived from the live record | +70–100 |
| `internal/executor/managed_agents/managed_agents_live_test.go` | Env-var-guarded ADC live contract spike; manual only, excluded from default CI execution | +45–70 |
| `cmd/ailang/exec.go` | Flags, parsing/loading helpers, task plumbing, shared dispatch gate, validation, and help | +50–80 |
| `cmd/ailang/exec_test.go` | CLI parsing and no-source regression tests | +30–50 |
| `cmd/ailang/coordinator_cloud_executor.go` | Route coordinator cloud execution through the shared capability-validating dispatch helper | +1–3 |
| `internal/coordinator/provider_executor.go` | Route coordinator programmatic execution through the shared capability-validating dispatch helpers | +2–5 |
| `internal/eval_harness/agent_runner_multi.go` | Route eval-harness programmatic execution through the shared capability-validating dispatch helper | +1–3 |
| `internal/eval_harness/gemini_evaluator_bridge.go` | Optional mount configuration and default-runner argument plumbing | +25–45 |
| `internal/eval_harness/gemini_evaluator_bridge_test.go` | Mounted-argument and fallback-path tests | +25–45 |
| `design_docs/planned/v0_30_0/m-gemini-repo-mount.md` | Store the credential-scrubbed Phase-1 verification outcome in the Premise Verification Log | Documentation only |

Implementation should prefer placing the builder in `types.go` if that keeps the production patch within
the ~250 LOC Go budget; `environment.go` is an alternative, not an additional requirement. The manual
spike/test fixture and test LOC do not expand the production-code bound; the shared gate call-site changes
are deliberately one-line substitutions rather than per-executor guards.

## Examples

### Before: every Gemini interaction

```go
envRaw := json.RawMessage(`{"type":"remote"}`)
```

```json
{
  "environment": {"type": "remote"}
}
```

### After: repository plus inline patch

Illustrative only: every source-shape annotation below is **(shape pending Phase-1 spike confirmation)**.
The spike may change a field name, remove a field, or require raw text versus base64 content before this
becomes an encoder golden.

```jsonc
{
  "environment": {
    "type": "remote",
    "sources": [ // (shape pending Phase-1 spike confirmation)
      {
        "type": "repository", // (shape pending Phase-1 spike confirmation)
        "source": "https://github.com/sunholo-data/ailang.git", // (shape pending Phase-1 spike confirmation)
        "target": "/workspace/ailang" // (shape pending Phase-1 spike confirmation)
      },
      {
        "type": "inline", // (shape pending Phase-1 spike confirmation)
        "target": "/workspace/ailang/.ailang-review/diff.patch", // (shape pending Phase-1 spike confirmation)
        "content": "<raw-UTF-8-or-base64: pending spike>" // (shape pending Phase-1 spike confirmation)
      }
    ]
  }
}
```

The requested ref/SHA is not emitted as an undocumented JSON property. It is carried explicitly to the
reviewer directive, which must run `git checkout --detach 36cca59a1` and verify the resulting HEAD before
applying or inspecting `diff.patch`.

### CLI

```bash
ailang exec gemini \
  --env-repo https://github.com/sunholo-data/ailang.git@36cca59a1 \
  --env-target /workspace/ailang \
  --env-inline-file diff.patch \
  "Checkout the requested SHA, apply .ailang-review/diff.patch, fetch the pinned Linux ailang binary, and run ailang check on the changed .ail files. Report commands and results."
```

With none of these flags, the command continues to send only `{"type":"remote"}` and behaves exactly as
existing `ailang exec gemini` calls do today.

## Success Criteria

- [ ] Empty source configuration serializes exactly to `{"type":"remote"}`.
- [ ] The Phase-1 live probe records credential-scrubbed repository-only, inline-only, and combined
  request/response evidence and updates the Premise Verification Log before encoder implementation begins.
- [ ] Repository-only, inline-only, and combined requests serialize to the spike-confirmed Managed Agents JSON.
- [ ] Inline files beyond the spike-confirmed per-file boundary fail loudly before network I/O; intended
  semantics are `1 << 20` bytes accepted and `1 << 20 + 1` rejected.
- [ ] `ailang exec gemini --env-repo ... --env-inline-file ...` reaches `executor.Task.EnvSources`.
- [ ] A non-supporting executor handed non-empty EnvSources returns a clear error (no silent ignore) — covered by a unit test.
- [ ] Eval-harness callers can request a mounted public repository at a pinned SHA and inject an
  uncommitted patch/file.
- [ ] The no-mount eval path still uses the existing diff bridge and degraded-verification semantics.
- [ ] Existing `ailang exec gemini ...` invocations work unchanged.
- [ ] Default Gemini exec behavior is unchanged.
- [ ] No evaluator default changes; Sonnet remains the default evaluator.
- [ ] All tests passing.
- [ ] Docs and `ailang exec` help updated.
- [ ] No live-network or ADC-dependent test is added to default CI.

## **Conflict Surface**

This change touches `cmd/ailang/exec.go`, a compilation entry point, plus the shared `executor.Task`
contract. The shared change includes a pre-dispatch `CapEnvironmentSources` gate, not only a CLI check.
The change is intentionally additive:

- New flags default off.
- `Task.EnvSources` defaults to nil/empty.
- The capability gate accepts nil/empty `EnvSources` for every executor, so all existing programmatic
  callers are unaffected; only a newly non-empty unsupported request returns an error.
- Managed Agents advertises `CapEnvironmentSources`; other executors do not and therefore cannot silently
  discard requested mounts.
- The Managed Agents environment builder returns the existing `{"type":"remote"}` payload for nil/empty
  sources.
- Existing executor selection remains `gemini` → `managed_agents`.
- Existing API-only provider behavior is unchanged; mount flags are rejected rather than ignored.
- Existing text extraction and file-artifact bridge behavior is unchanged.

Existing programs and callers that must continue to work without modification:

- `ailang exec gemini "<directive>"`
- `ailang exec gemini "<directive>" --model antigravity-preview-05-2026`
- `ailang exec gemini --json --prompt "<directive>"`
- `ailang exec gemini ... --workspace <dir>` even though the hosted sandbox still cannot directly mount
  that local directory.
- Coordinator or factory callers constructing `executor.Task` without `EnvSources`.
- `internal/eval_harness/gemini_evaluator_bridge.go` callers using `RunGeminiEvaluator` with default
  `EvalOptions`.
- Tests and injected runners relying on the current `EvalRunner` seam.
- `internal/eval_harness/managed_agents_bridge.go` extract-out behavior for solution artifacts.
- Claude, OpenAI, Anthropic, OpenRouter, and Ollama `ailang exec` paths when no mount flags are present.

Syntactic-position analysis: **N/A — no parser/lexer/type change; the surface is the JSON request contract
+ CLI flag set.** There is no AILANG grammar ambiguity, token change, precedence change, AST change, or type
inference impact. The compatibility risks are Go compile-time call signatures, flag normalization, external
Managed Agents JSON shape, and eval-runner argument construction.

Required compile-surface precautions:

- Prefer an options/source parameter or helper that minimizes churn to existing `executeCLI` test seams.
- Update every direct Go caller of any changed function signature in the same patch.
- Preserve JSON omission/order expectations used by request tests.
- Ensure repeatable flags do not consume the provider or directive positional arguments.
- Ensure parsing `@<ref>` does not corrupt URLs containing userinfo; public unauthenticated HTTPS URLs are
  the supported G4 path.

## Testing Strategy

### Unit tests: environment payload

- Empty/nil sources return exactly `{"type":"remote"}`.
- Repository-only yields one repository source with the spike-confirmed URL/target field names.
- Inline-only yields one inline source with the spike-confirmed fields and exact recorded raw/base64 encoding.
- Repository plus multiple inline files preserves deterministic caller order.
- The live-confirmed size boundary is covered exactly; intended cases are `1 << 20` bytes accepted and
  `1 << 20 + 1` bytes returning a descriptive error, contingent on Phase 1.
- Empty URL, relative/unsafe target, unknown kind, duplicate target, and empty inline name/content source
  return errors.
- Builder errors occur before the HTTP sender is invoked.

### Unit tests: shared capability dispatch

- A fake executor without `CapEnvironmentSources` receives no `Execute`/`ExecuteStreaming` call when handed
  non-empty `Task.EnvSources`, and the shared dispatcher returns a clear capability error.
- The same fake executor accepts nil/empty `EnvSources`, proving existing programmatic callers are unchanged.
- The Managed Agents executor advertises `CapEnvironmentSources` and proceeds to provider payload validation.

### Unit tests: CLI and eval seam

- Parse `--env-repo https://github.com/sunholo-data/ailang.git@36cca59a1` into URL plus ref.
- Parse a custom `--env-target` and repeated `--env-inline-file` values in caller order.
- Load temporary inline files and propagate bytes/targets onto `executor.Task`.
- Reject mount flags with `--api-only` and unsupported executors.
- Verify no mount flags produce nil/empty `Task.EnvSources`.
- Verify default `RunGeminiEvaluator` still takes the diff-bundle fallback path.
- Verify mounted evaluator options append the expected command arguments and do not duplicate the full diff
  in the directive.

### Regression and manual verification

- Retain existing managed-agent, `exec`, and eval-bridge unit tests as regression coverage.
- Phase 1 adds an opt-in probe such as
  `AILANG_LIVE_MANAGED_AGENTS_MOUNT=1 go test ... -run LiveEnvironmentContract`.
- The live probe is manual only, requires ADC/project configuration, exercises repository-only,
  inline-only, and combined payloads plus the size/network checks, and records sanitized request/response
  evidence before any encoder golden is written.
- Default CI must skip the live probe without treating missing ADC as success or silently substituting a
  fake environment.

## Non-Goals

- No change to the default evaluator; it remains Sonnet. A Gemini-default flip requires the charter's
  ≥3-datapoint evidence rule.
- No executor-owned orchestration of `ailang check`, release-binary download, patch application, or checkout.
  Those steps belong to the reviewer/evaluator directive downstream; this work lands mount plumbing and a
  documented recipe only.
- No environment reuse or persistence across interactions.
- No private repository support or credential injection.
- No automatic write-back from the hosted sandbox to the caller's worktree.
- No replacement or deletion of the existing diff bridge.

## Timeline

| Phase | Estimate | Deliverable |
|-------|----------|-------------|
| 1. Contract-discovery spike | 1–2 hours manual | ADC-gated live repository/inline/combined evidence, sanitized record, and VERIFIED-LIVE premise updates |
| 2. Typed task, gate, and payload builder | 2–3 hours | Capability-safe dispatch and live-confirmed environment JSON with unchanged empty default |
| 3. CLI flags and task threading | 1–2 hours | Repository/ref/target and repeatable inline-file plumbing |
| 4. Eval-harness caller seam | 1 hour | Mounted verification option plus retained diff fallback |
| 5. Tests, help, and recipe | 1–2 hours | Hermetic coverage derived from the live record and bounded reviewer recipe |

Total: approximately one focused engineering day.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| Managed Agents wire shape differs from documentation/assumptions | Request rejection or false golden confidence | Phase 1 live-probes the actual endpoint and blocks Phase 2 until a sanitized accepted contract is recorded; only then are provider structs and goldens written. |
| Repository mount does not guarantee the requested revision | Reviewer checks wrong code | Require a pinned ref/SHA for eval-harness mounted verification; directive runs detached checkout and verifies `git rev-parse HEAD`. |
| Inline patch exceeds the live-confirmed per-file limit | Evaluation cannot mount the sprint delta | Fail before network I/O with path, actual size, and recorded limit; caller may split the delta into multiple explicit inline files. Never fall back silently to truncated content. |
| Target collisions or traversal overwrite mounted files unexpectedly | Incorrect or unsafe verification state | Require absolute normalized sandbox targets, reject traversal and duplicate targets, and use a dedicated `.ailang-review/` namespace by default. |
| Shared `Task` type leaks provider-specific semantics | Architectural coupling | Model the field as generic execution-environment sources; keep Managed Agents JSON and API terms inside its executor package. |
| Programmatic caller sends sources to an unsupported executor | Mount request is silently ignored | Shared pre-dispatch `CapEnvironmentSources` validation rejects non-empty sources before calling the executor; unit-test both streaming and non-streaming paths. |
| New CLI flags perturb positional directive parsing | Existing commands break | Add focused normalization/positional tests and keep all defaults empty/off. |
| Eval mount fails and bridge fallback hides it | False confidence | Fallback is selected only when no mount was requested. Any requested-mount failure is terminal and explicit. |
| Live-confirmed default outbound network increases sandbox authority | Supply-chain or data-exfiltration risk | Phase 1 records actual behavior. If unrestricted, mounts remain explicit opt-in, public-repo-only, and directives pin download URLs/versions; do not pass local secrets or credentials as inline files. If restricted, fail or configure the recorded allowlist explicitly. |
| Production patch grows beyond the one-day bound | Delayed mission gap | Keep environment reuse, automatic overlays, private repos, and check orchestration deferred; target ≤~250 LOC of Go. |

## Axiom Compliance

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Managed Agents execution is already nondeterministic; source order, content, targets, and pinned SHA are explicit and deterministically encoded. |
| A2: Replayability | +1 | Repository URL, pinned SHA, inline bytes, and targets can be captured with the task/request, making the reviewed input reconstructable. |
| A3: Effect Legibility | 0 | No AILANG effect-system change; external filesystem/network effects remain inside the executor boundary. |
| A4: Explicit Authority | +1 | Mounting grants the hosted sandbox read access to a named public repository and explicit inline content only when the caller opts in. No local workspace, secret, or credential authority is ambient; unsupported authority requests fail. |
| A5: Bounded Verification | +1 | The bounded, env-var-gated live spike must establish the external premise before implementation, and mounted Gemini checks then run against a pinned revision instead of relying solely on diff reasoning. The live-confirmed inline boundary is explicit and tested. |
| A6: Safe Concurrency | 0 | No concurrency or shared-environment reuse is introduced; each interaction still receives a fresh environment. |
| A7: Machines First | +1 | Typed source configuration and machine-executed `ailang check` replace a human-oriented prompt-only bridge as the preferred verifying path. |
| A8: Minimal Syntax | 0 | Adds only CLI flags and Go structs; no AILANG syntax. The surface is the minimum needed for repository and inline mounts. |
| A9: Cost Visibility | 0 | Existing execution usage/cost reporting remains; source mounting does not add hidden model calls. Download/runtime costs remain visible in agent execution duration. |
| A10: Composability | +1 | The same source list composes repository baseline and multiple inline overlays and can be used by CLI, evaluator, and reviewer callers. |
| A11: Structured Failure | +1 | A shared pre-dispatch capability gate rejects non-empty sources for unsupported executors across CLI and programmatic callers; invalid URLs, targets, kinds, duplicates, oversized files, unverified contracts, and JSON construction errors also fail explicitly with no silent fallback. |
| A12: System Boundary | +1 | Local file reading occurs at the caller boundary; provider JSON encoding occurs in `managed_agents`; reviewer orchestration stays in the directive. Responsibilities remain explicit. |

**Net Score: +7** → **Proceed only after the Phase-1 contract gate.** No hard-axiom violations. A4 is
strengthened because repository and inline read authority is explicit, bounded, visible in task
configuration, and off by default. A5 is stronger because external premises are established by a bounded
live probe before conclusions, and A11 is stronger because unsupported programmatic dispatch fails at the
shared executor boundary rather than relying on CLI-only validation.

## References

- [`internal/executor/managed_agents/`](../../../internal/executor/managed_agents/) — Managed Agents
  executor and the existing hardcoded remote environment.
- [`internal/executor/executor.go`](../../../internal/executor/executor.go) — shared `executor.Task` contract.
- [`cmd/ailang/exec.go`](../../../cmd/ailang/exec.go) — `ailang exec` flags and Gemini executor routing.
- [`internal/eval_harness/gemini_evaluator_bridge.go`](../../../internal/eval_harness/gemini_evaluator_bridge.go)
  — current reasoning-only diff bridge and caller seam.
- [`internal/eval_harness/managed_agents_bridge.go`](../../../internal/eval_harness/managed_agents_bridge.go)
  — existing cross-environment output bridge.
- [V1 mission gap G4](../../v1-mission.md) — repo-mount upgrade and designer-rotation prerequisite.
- [Gemini Agent Environment documentation](https://ai.google.dev/gemini-api/docs/agent-environment) —
  documentation read on 2026-07-17 for repository/inline examples, stated limits, network behavior, and
  reusable environments; DOC-ONLY, not a live verification record for this executor's endpoint.

## Future Work

- Reuse Managed Agents environments across interactions with explicit IDs, retention, cleanup, and stale-state
  controls so persisted files are safe and observable.
- Promote the manual checkout/download/`ailang check` sequence into a versioned reviewer-directive recipe,
  while keeping orchestration outside the executor.
- Add helpers to generate split inline overlays for large uncommitted changes without exceeding per-file limits.
- Collect at least three mounted-verification datapoints and compare verdict quality/cost before considering a
  default-evaluator change.
- After successful mounted reviewer/evaluator evidence, add Gemini to the design-doc creator rotation described
  by G4 and record `(designer, quorum outcome)` evidence in the mission log.
