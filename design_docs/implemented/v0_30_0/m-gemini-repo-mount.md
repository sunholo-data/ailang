# M-GEMINI-REPO-MOUNT — Managed Agents repository and inline-source mounts

**Status**: **✅ IMPLEMENTED + LIVE-VERIFIED (all premises confirmed).** Phase 2 shipped in sprint `M-GEMINI-REPO-MOUNT` (mission iteration 52, opus executor, sonnet evaluator 91/100 PASS): typed `RequiresEgress`/`CapNetworkEgress` gate + shared `ValidateTaskCapabilities`, `buildEnvironment` egress wiring, `--clone-repo`/`--clone-sha` CLI flags, eval-harness clone-review with conditional evidence check + bounded-execution/deadline-degraded, and a live-gated E2E. **The last INCORPORATED-not-VERIFIED-LIVE premise** — provider support for the arbitrary-SHA shallow `git fetch --depth 1 origin <sha>` — was **LIVE-CONFIRMED in mission iteration 53** (Mark #399 "vertex git clone test granted"): `TestLiveCloneOverEgressE2E` pinned a real non-HEAD SHA (`80cbd9612…`) through the production `Executor.Execute` path — the sandbox fetched-by-SHA, checked out `FETCH_HEAD`, echoed the EXACT pinned SHA, emitted `CLONE_OK` + verdict. **PASS in 113.6s, $0.865, 527221 in / 8201 out tokens** (project `ailang-dev`, `global`). Doc moved to `implemented/v0_30_0/`. Trail below (revised by the claude-fable-5 designer per Mark's #399 "apply both fixes, ship it"; both iteration-51 re-quorum fixes + the iteration-52 re-quorum shallow-fetch fix applied): (1) typed `RequiresEgress`/`CapNetworkEgress` gate replacing the Metadata-key opt-in; (2) "Bounded execution & timeout reuse" grounded in the existing timeout/context machinery. The original `repository`/`inline` mount model was REFUTED live (iter-45), retained below as historical record; iter-46 LIVE-VERIFIED the lean replacement: an egress-enabled sandbox (`environment.network.allowlist:[{domain:"*"}]`, NO data source) in which the agent runs `git clone` of the public ailang repo end-to-end (cloned HEAD `806b3b4a4`, listed files, read `go.mod`).
**Target**: v0.30.0
**Priority**: P1 — mission gap G4; upgrades Gemini from reasoning-only review to in-sandbox verification
**Estimated**: 1–2 focused days; ≤150 LOC production Go (agent-side `git clone` + typed egress capability
gate — no encoder, no GCS, no mount, no inline)
**Dependencies**: `managed_agents` executor / M-MANAGED-AGENTS v0.22.0

> **✅ RESOLVED (Mark, #399, 2026-07-18: "apply both fixes, ship it") — formerly ⛔ PARK-NOTE (quorum
> block, mission iteration 51).** Iteration 51 decomposed Mark's approved clone-over-egress scope into the
> sprint-sized **Phase 2** below and ran the bounded design quorum (`gpt5-6-sol` + `gemini-3-1-pro` +
> claude controller). Round 1 → BLOCKED on a real HEAD-review evidence-check bug (echoed-SHA `==` empty
> `CloneSHA` would fail every HEAD review) → **FIXED** in a revision pass (evidence check is now
> conditional; new positive acceptance test). The **re-quorum** then BLOCKED on **two NEW, sound
> objections** that exceeded the one-revision bound, parking the doc for Mark's call. **Mark approved both
> fixes**, and this revision (mission iteration 52, claude-fable-5 designer) applies them. The two
> objections, retained as the changelog of this revision:
>
> 1. **Typed egress-capability gate, not a `Metadata` key (gemini-3-1-pro) — ✅ APPLIED.** The prior
>    opt-in (one provider-scoped `Task.Metadata["managed_agents.egress"]="1"` key) was a programmatic
>    silent-fallback hole on the shared `executor.Task` Go API: a non-CLI caller that set the key on a
>    non-managed_agents executor got egress *silently ignored* (the key is uninterpreted) — the same
>    no-silent-fallback class as round 1. Fix applied below: a typed `RequiresEgress bool` field on
>    `executor.Task` + a `CapNetworkEgress` capability constant in `internal/executor/executor.go` + a
>    shared pre-dispatch validation that errors loudly when `RequiresEgress` is set but the resolved
>    executor does not advertise the capability. This deliberately reverses the iteration-51 "one boolean
>    → Metadata, don't widen `Task`" scope call and re-widens the shared `executor.Task` contract —
>    exactly the architecture call Mark ratified.
> 2. **Bounded execution & timeout reuse (gpt5-6-sol) — ✅ APPLIED.** Phase 2 had the sandbox agent run a
>    (possibly full-history, for arbitrary SHAs) `git clone` + optional binary download + `ailang check`
>    with no stated deadlines/cancellation, and did not identify the existing managed_agents
>    timeout/context machinery to reuse — violating the mission's **Standing Rule 6 (every wait is
>    bounded)**. Fix applied below: a new **"Bounded execution & timeout reuse"** section grounds Phase 2
>    in the EXISTING `context.WithTimeout` → `sendInteraction` deadline and eval-bridge ctx threading
>    (file:line recorded in the premise table), makes deadline expiry a structured `VerificationDegraded`
>    result (never a retry, never a clean pass), and adds cancellation/timeout acceptance tests.
>
> **Re-quorum (iteration 52) — one further fix applied under Mark's "ship it".** The post-revision
> re-quorum ran `gpt5-6-sol` + `gemini-3-1-pro`: `gpt5-6-sol` was **budget-absent** (doc grew past its
> $0.10 cap; its own objection #2 is already applied so its absence is low-risk — degraded N−1);
> `gemini-3-1-pro` raised **one new, sound objection** (NOT one of Mark's two): the arbitrary-SHA path
> mandated a **full clone**, but Probe R only verified `--depth 1` — an unverified, potentially unbounded
> premise (a Standing-Rule-6 hole inside the bounded-execution fix itself). Its own proposed recipe
> (shallow **fetch-by-SHA**, `git fetch --depth 1 origin <sha>`) was applied verbatim (Canonical clone
> preamble + Bounded execution + premise table + risks), making BOTH clone modes shallow/bounded and
> needing no live Probe-S. Applied inline under Mark's explicit "apply both fixes, ship it" directive
> (which outranks the re-quorum-once park rule) rather than re-parking on a self-fixable refinement of an
> already-approved fix — surfaced here and in the iteration-52 mission report for Mark to veto if desired.
> This is Go plumbing with no `.ail` surface, so the design-doc-creator live `ailang check` gate is N/A.
> Quorum artifacts: `.ailang/state/mission-quorum/m-gemini-repo-mount-2026-07-18T{08,10}-*.json`.

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

> **✅ PHASE-1b UPDATE (mission iteration 46, 2026-07-17) — EGRESS PARAM FOUND; CLONE-OVER-EGRESS
> LIVE-VERIFIED.** Mark replied on #399: *"can you look at this for the vertex managed agents interaction
> with GitHub? https://www.philschmid.de/managed-agents-gh"*. That post (Gemini **Developer** API surface,
> `google-genai` SDK) demonstrates GitHub access via **network egress with header transform**, and — key
> insight — the egress param is a **structured list** `network.allowlist:[{domain, transform}]`, NOT the
> six *scalar* enable-flags iter-45 guessed (which is why iter-45 missed it). Re-probing OUR Vertex
> endpoint with that shape (probes O–R, same ADC harness): `network.allowlist:[{domain:"*"}]` is
> **accepted and provisions an egress-enabled sandbox** (specific domains + `transform` are "not supported
> now" on Vertex — wildcard only). Probe **R** then proved the money shot: an egress-only env (no data
> source at all) **cloned the public ailang repo end-to-end** — `git clone --depth 1` succeeded,
> `rev-parse HEAD` = `806b3b4a4` (current dev), file listing + `go.mod` returned. **So iter-45's "nearest
> path = GCS-backed mount, large lift" is superseded: for a PUBLIC repo the agent just clones itself over
> egress — no mount, no GCS, no inline.** See the **Phase-1b Spike Result** section and option **(d)**.

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

## Phase-1b Spike Result (VERIFIED-LIVE, mission iteration 46, 2026-07-17)

**Authorization / trigger:** Mark, on #399 — "can you look at this … https://www.philschmid.de/managed-agents-gh".

**What the blog gave us.** It targets the Gemini **Developer** API (`ai.google.dev` /
`generativelanguage.googleapis.com`, `google-genai` SDK, API-key auth) — a *different surface* from our
executor's Vertex `aiplatform.googleapis.com` (ADC). On that surface, GitHub access is done by enabling
**network egress** and injecting the PAT via a per-domain header **transform**, with the agent running
`gh`/`git` inside the sandbox. The transferable insight: the egress param is a **structured list**
`environment.network.allowlist:[{domain, transform:[{Authorization:…}]}]`, not a boolean/enum enable-flag.
(Independent our-project confirmation of the Developer-API surface was **not** possible this iteration —
the available `GOOGLE_API_KEY` is invalid even for `generateContent`; a valid Developer-API key with
interactions/preview access is a human provisioning step, parked.)

**Method:** the same env-guarded ADC probe harness as iter-45 (`managed_agents_live_test.go`, probes
O–R), against the **Vertex** endpoint the executor actually uses. Probes O–Q are cheap validation `400`s;
Q and R **provision a real sandbox** (small, bounded cost).

| Probe | `environment` sent (abridged) | Live Vertex response |
|-------|-------------------------------|----------------------|
| O top `network.allowlist` + domain+transform | `network:{allowlist:[{domain:"api.github.com",transform:[{Authorization:"Bearer X"}]}]}` + gcs src | `400 Only domain: '*' is supported now.` |
| P same under `config` | `config:{network:{allowlist:[…api.github.com…]},sources:[…]}` | `400 Only domain: '*' is supported now.` |
| Q wildcard only | `network:{allowlist:[{domain:"*"}]}` + gcs src | **`status=completed`, agent ran, replied `OK`** — egress env provisioned |
| R **egress-only clone** | `network:{allowlist:[{domain:"*"}]}` (NO data source) | **`status=completed`, 9 steps** — `git clone --depth 1` of public ailang repo OK; `rev-parse HEAD`=`806b3b4a4`; file listing + `go.mod` (`module github.com/sunholo-data/ailang`, `go 1.26.5`) returned verbatim |

**What is now VERIFIED-LIVE (superseding iter-45 #3):**

1. **The egress-enable param IS `environment.network.allowlist`** — a list of `{domain}` (with optional
   `transform`). iter-45's six scalar guesses missed it because the shape is a list, not a flag.
2. **Only `domain:"*"` (wildcard, all-egress) is accepted on Vertex today.** Specific domains and the
   blog's per-domain header `transform` are explicitly *"not supported now"* on Vertex (they work on the
   Developer-API surface). So the blog's PAT-injection trick is **not** available on Vertex yet.
3. **An egress-only sandbox (no data source) clones a PUBLIC repo end-to-end.** The agent itself runs
   `git clone` over the open egress — no `repository`/`inline`/`gcs` mount involved at all. For public
   repos this makes the entire source-mount design **unnecessary**.

**Security note.** `domain:"*"` is unrestricted outbound. For the intended use (an evaluator/reviewer
that clones a *public* repo at a given SHA and reasons over it, read-only) that is acceptable and needs
no secret — a public clone requires no auth, so no PAT ever enters the sandbox. Anything requiring a
private repo or secret would need per-domain `transform` (Vertex: not yet) or the Developer-API surface —
out of scope for the public-repo review capability.

### ✅ Decision RESOLVED (Mark, #399, 2026-07-18T06:58:06Z): option (d) APPROVED

**"clone over egress approved."** The Phase-2 decomposition below implements option (d). The option
analysis is retained as the decision record:

- **(d) Clone-over-egress (NEW — recommended).** Give the `managed_agents` executor/evaluator an
  egress-enabled env (`network.allowlist:[{domain:"*"}]`) and have the agent `git clone` the (public)
  ailang repo at the target SHA itself, then run `ailang check`/review in-sandbox. **No encoder, no GCS,
  no inline, no mount** — this is the small path the doc originally hoped for, now on solid ground. Directly
  delivers Mark's #399 "gemini can git clone the codebase" for the reviewer/evaluator role. Next step:
  decompose a small sprint (egress env wiring in the executor + a review directive + the existing
  `managed_agents_bridge` for artifact return). LIVE-VERIFIED feasible.
- **(a) GCS-backed mounts.** Still valid for *private* code / offline determinism, but now the *fallback*,
  not the primary — a larger lift only justified if clone-over-egress's public-only / wildcard-egress
  limits bite.
- **(b) Shelve; keep the prompt-packed diff bridge.** Still the zero-cost option, but (d) is now cheap
  enough that shelving forgoes a real capability gain.
- **(c) `skill_registry`.** Unchanged — lowest confidence, not pursued.

**Recommendation was (d); Mark approved (d) on #399 (2026-07-18).** The doc is now APPROVED and the
Phase-2 decomposition follows immediately below.

## Phase 2 (PLAN-READY — both quorum fixes applied, see ✅ RESOLVED note at top) — Clone-over-egress capability

This is the approved, sprint-sized decomposition of option (d). Everything here is grounded either in the
doc's recorded HEAD facts (Problem Statement, verified against the working tree 2026-07-18) or in the
VERIFIED-LIVE probe record (Phase-1/1b sections above). No mount, no GCS, no inline, no encoder.

### Overview

Give the `managed_agents` executor an **opt-in egress-enabled environment**
(`{"type":"remote","network":{"allowlist":[{"domain":"*"}]}}` — the exact shape probes Q/R proved), and
have the sandbox agent itself `git clone` the public ailang repo at the target revision, run
review/`ailang check` in-sandbox, and return a structured verdict through the **existing** text-output
bridge. The executor wires the environment; the *caller* (CLI / eval harness) owns the clone directive —
matching the executor's recorded policy-free contract (comment at `managed_agents.go:153–163`: artifact
return via text output, "The executor itself stays policy-free").

### Opt-in mechanism (DECISION: typed `RequiresEgress` field + `CapNetworkEgress` capability gate — Mark's call, #399 2026-07-18)

**Chosen:** a typed capability gate on the shared executor contract:

- `executor.Task` gains a typed field **`RequiresEgress bool`** (struct at `executor.go:37`, alongside
  `Metadata` at :60 and `ExtraEnv` at :68). Zero value `false` = today's behavior.
- A new capability constant **`CapNetworkEgress Capability = "network_egress"`** joins the existing
  constants at `executor.go:193–213` (`Capability` type at :190; `CapRemoteSandbox` at :213).
- The `managed_agents` executor advertises `CapNetworkEgress` in its `Capabilities()` (interface method
  at `executor.go:24`); **no other executor does**.
- A **shared pre-dispatch validation** errors LOUDLY — before any network I/O — when
  `task.RequiresEgress` is true but the resolved executor does not advertise `CapNetworkEgress`. This
  closes the programmatic silent-fallback hole: a non-managed_agents executor can no longer silently
  ignore an egress request (the same no-silent-fallback class as the round-1 bug).
- The managed_agents env builder emits `{"type":"remote","network":{"allowlist":[{"domain":"*"}]}}` when
  `task.RequiresEgress` is set, else the byte-identical default `{"type":"remote"}`.

Repo URL and SHA remain **caller-side directive inputs** (CLI flags / eval-harness options), not executor
inputs, because the executor is policy-free: it needs to know only "egress on/off"; what the agent does
with the egress (clone what, checkout what) is directive policy built by the caller.

**Prior proposal (superseded):** the iteration-51 draft used one `Task.Metadata["managed_agents.egress"]`
key precisely to avoid widening the shared `Task` contract (≤120-LOC bias). The re-quorum
(gemini-3-1-pro) held that the provider-scoped key was itself a programmatic silent-fallback hole, and
Mark ratified the typed gate on #399 ("apply both fixes, ship it").

**Residual: RESOLVED, not deferred.** The typed gate closes the programmatic hole outright: an egress
request is now a compile-visible field, and setting it on a non-`CapNetworkEgress` executor produces a
loud shared pre-dispatch error — never an uninterpreted string. No provider-scoping residual remains.

### Code-change surface (grounded at recorded HEAD facts)

1. **`internal/executor/executor.go`** — add `RequiresEgress bool` to `Task` (struct at :37), the
   `CapNetworkEgress` capability constant alongside the existing constants (:193–213; `Capability` type
   :190), and a small shared helper **`executor.ValidateTaskCapabilities(task, exec) error`** that
   returns a loud error when `task.RequiresEgress` is true and `exec.Capabilities()` (interface method
   :24) lacks `CapNetworkEgress`. A single shared helper — rather than inline checks at each dispatch
   site — is chosen because it gives ONE grep-able enforcement point that future capabilities extend
   (the superseded mount design independently arrived at the same shape with
   `CapEnvironmentSources`/`ValidateTaskCapabilities`); the CLI (`executeCLI`) and eval-bridge dispatch
   paths call it before `Execute`/`ExecuteStreaming`.
2. **`internal/executor/managed_agents/managed_agents.go`** — replace the hardcode at line 164
   (`envRaw := json.RawMessage(`{"type":"remote"}`)`, assigned to `Environment` at line 170) with a
   small `buildEnvironment(task) (json.RawMessage, error)`:
   - `task.RequiresEgress == false` → return **byte-identical** `{"type":"remote"}` (default unchanged,
     egress OFF).
   - `task.RequiresEgress == true` → return
     `{"type":"remote","network":{"allowlist":[{"domain":"*"}]}}` (probe Q/R shape).
   - Advertise `CapNetworkEgress` in `Capabilities()`. The package continues to read **no** Metadata
     keys at all (verified by grep, 2026-07-18 — empty; the typed field replaces the planned key).
3. **`cmd/ailang/exec.go`** — register `--clone-repo <url>` and `--clone-sha <sha>` in the `runExec`
   flag block (flags registered ~lines 63–88; help text ~lines 677–710):
   - `--clone-repo` set → validate the resolved executor is `managed_agents`
     (`resolveAgenticExecutorName`, line 317); otherwise, or with `--api-only`, **exit non-zero with a
     clear error** — never ignore. This fast CLI check is UX-layer; the shared
     `ValidateTaskCapabilities` pre-dispatch gate (item 1) is the enforcement that also covers every
     non-CLI programmatic caller.
   - `--clone-sha` without `--clone-repo` → error.
   - On success: set `task.RequiresEgress = true` on the `executor.Task` built in `executeCLI` (task
     literal at lines 347–356) and prepend the canonical clone preamble (below) to the directive.
4. **`internal/eval_harness/gemini_evaluator_bridge.go`** — extend `EvalOptions` with optional
   `CloneRepoURL` + `CloneSHA`. When set, `RunGeminiEvaluator` (line 620) builds the clone-review
   directive instead of packing the full diff; when unset, the `BuildDiffBundle` (line 131)
   prompt-packed path is **unchanged**. Verdict return rides the existing text-output parsing in
   `managed_agents_bridge.go` / the bridge's verdict parser — no new artifact channel. The directive
   requires the agent to echo `git rev-parse HEAD`; the bridge's evidence check is **conditional on
   whether a SHA was pinned**: when `CloneSHA` is non-empty, the bridge asserts the echoed HEAD
   **equals** `CloneSHA`; when `CloneSHA` is empty (HEAD review), there is nothing to compare against —
   the bridge instead asserts the echo is a syntactically-valid, non-empty 40-hex SHA (proof the agent
   actually cloned and ran in-sandbox) and **records** it as the reviewed revision. In both cases a
   missing/empty/invalid echo — or, in the pinned case, a mismatch — stamps `VerificationDegraded: true`
   with a non-empty `DegradedReason` (reusing the existing invariant at lines 551–562: degraded ⇒ reason
   non-empty; degraded is never a clean pass). A HEAD review with a valid echo is NOT degraded.

### Canonical clone preamble (review directive)

- **No SHA requested (HEAD review):** `git clone --depth 1 <public-url>` — the exact probe-R-proven
  recipe; echo `git rev-parse HEAD`. **Evidence = a syntactically-valid, non-empty 40-hex echo** (there
  is no `CloneSHA` to compare against); the bridge records the echoed SHA as the reviewed revision.
- **Arbitrary SHA requested:** **shallow fetch-by-SHA** (bounded, no full history) —
  `git init && git remote add origin <public-url> && git fetch --depth 1 origin <sha> && git checkout
  --detach FETCH_HEAD`; echo `git rev-parse HEAD`. **Evidence = the echo must equal the requested
  `CloneSHA`.** This adopts the iteration-52 re-quorum fix (gemini-3-1-pro) verbatim: the earlier draft
  mandated a **full clone** (NOT `--depth 1`) to reach an arbitrary older SHA, but Probe R only verified
  the shallow `--depth 1` path — a full clone's completion within the sandbox disk/network/interaction
  limits was an **unverified, potentially unbounded** premise (a Standing-Rule-6 hole in the very
  bounded-execution fix that motivated this section). A `git fetch --depth 1 origin <sha>` fetches
  exactly the one pinned commit shallowly (no history walk, bounded like the HEAD path), so it both
  reaches any arbitrary SHA AND stays within the probe-proven shallow envelope — no full clone, no
  Probe-S live-verification needed. M4's live-gated E2E exercises this fetch-by-SHA path to confirm the
  provider supports `fetch --depth 1 <sha>`.
- Then: run the review / `ailang check` (the agent may fetch a pinned Linux `ailang` release binary over
  the same egress) and emit the structured verdict JSON the bridge already parses.

### Bounded execution & timeout reuse (Standing Rule 6: every wait is bounded)

No new plumbing — Phase 2 reuses the timeout/context machinery that ALREADY exists on the exact call
path:

- **Per-interaction hard ceiling (already exists).**
  `internal/executor/managed_agents/managed_agents.go:178–184` computes `timeout := task.Timeout`
  (falling back to `e.timeoutSeconds`, sourced from `cfg.TimeoutSeconds` at :38) and creates
  `reqCtx, cancel := context.WithTimeout(ctx, timeout)` at :183, propagated into
  `sendInteraction(reqCtx, ...)` at :187. The entire in-sandbox clone → checkout → optional binary
  download → `ailang check` runs server-side WITHIN that single bounded interaction — the whole
  clone-review is covered by one propagated deadline.
- **Caller ctx already threaded on the eval path.** `internal/eval_harness/gemini_evaluator_bridge.go`:
  `EvalRunner` is `func(ctx context.Context, ...)` (:594), `RunGeminiEvaluator(ctx context.Context, ...)`
  (:620), and `DefaultGeminiRunner` threads ctx to the `ailang exec gemini` process (:682). No fresh
  `context.Background()` on the live path.
- **Clone-depth bound (both modes shallow — iteration-52 re-quorum fix).** HEAD review uses
  `git clone --depth 1` (the probe-R-proven recipe); the arbitrary-SHA path uses a **shallow fetch-by-SHA**
  (`git fetch --depth 1 origin <sha>` — see Canonical clone preamble), NOT a full clone. Neither mode walks
  full history, so both stay within the probe-proven shallow envelope and are bounded by construction; and
  both additionally run inside the single interaction deadline above — if a fetch cannot finish within the
  interaction deadline, the interaction times out and the result is a structured degraded/error (below),
  never a hang. This closes the full-clone unverified-premise hole gemini-3-1-pro raised in the iteration-52
  re-quorum.
- **Deadline expiry is a STRUCTURED degraded/error result — never a retry, never a clean pass.** A
  context-deadline-exceeded from `sendInteraction` stamps `VerificationDegraded: true` with a non-empty
  `DegradedReason`, reusing the existing `VerificationDegraded ⇒ non-empty DegradedReason` invariant at
  `gemini_evaluator_bridge.go:551–562`.

### No-silent-fallback compliance

- Egress is strictly opt-in; the `RequiresEgress == false` default request stays byte-identical to today.
- **Primary guard — the typed gate:** `task.RequiresEgress` set on an executor that does not advertise
  `CapNetworkEgress` → loud shared pre-dispatch error before any network I/O, covering programmatic AND
  CLI callers alike (the Metadata provider-scoping residual is gone).
- Clone flags on a non-managed_agents executor or with `--api-only` → loud CLI error, never ignored
  (fast-fail UX layered over the shared gate).
- A requested clone-review that cannot produce valid clone evidence → `VerificationDegraded` with
  reason — never silently downgraded to the prompt-packed path, never a clean pass on absent evidence.
  "Valid" is conditional on the request: pinned `CloneSHA` ⇒ echo must match it; HEAD review (empty
  `CloneSHA`) ⇒ echo must be a valid non-empty 40-hex SHA (recorded as the reviewed revision).
- Deadline expiry → `VerificationDegraded` with reason (Bounded execution section above) — never a
  silent retry, never a clean pass.

### Milestones (each ≤1 day)

| Milestone | Deliverable (one line) |
|-----------|------------------------|
| **M1** — typed egress capability gate + env wiring | `Task.RequiresEgress` + `CapNetworkEgress` + shared `ValidateTaskCapabilities` (executor.go); `buildEnvironment` replaces the `managed_agents.go:164` hardcode; golden tests assert BOTH JSON shapes + loud non-capability rejection, no live call |
| **M2** — CLI flags | `--clone-repo`/`--clone-sha` + `task.RequiresEgress` wiring + loud non-managed_agents/`--api-only` rejection + help text + parsing tests |
| **M3** — eval-harness clone-review | `EvalOptions.CloneRepoURL/CloneSHA` → clone directive + HEAD-evidence check + unchanged-fallback regression tests + timeout/cancellation tests (deadline-exceeded ⇒ degraded; caller ctx honored, no fresh background ctx) |
| **M4** — live E2E + docs | `AILANG_LIVE_MANAGED_AGENTS_MOUNT=1`-gated (CI-skipped) end-to-end clone→check→verdict run; evidence recorded in this doc |

#### Implementation status (sprint `M-GEMINI-REPO-MOUNT`, branch `sprint/g4-clone-over-egress`)

- **M1 — DONE.** `Task.RequiresEgress`, `CapNetworkEgress`, and shared
  `ValidateTaskCapabilities` in `internal/executor/executor.go`; `buildEnvironment`
  replaces the `envRaw` hardcode in `managed_agents.go` and the executor advertises
  `CapNetworkEgress`. Golden tests pin both JSON shapes byte-for-byte + the loud
  non-capability rejection (fake executor, no `Execute` call). No live call.
- **M2 — DONE.** `--clone-repo`/`--clone-sha` in `cmd/ailang/exec.go`; a single-source
  `executor.BuildClonePreamble` (shared with M3 so CLI + bridge cannot drift) and
  `executor.ValidateCloneFlags` (dispatch-independent). Non-`managed_agents`,
  `--api-only`, `--clone-sha`-without-`--clone-repo`, and invalid-SHA all exit
  non-zero. Help updated. Parsing tests cover every case.
- **M3 — DONE.** `EvalOptions.CloneRepoURL/CloneSHA` in `gemini_evaluator_bridge.go`;
  clone-review directive replaces the diff bundle when set (unset ⇒ `BuildDiffBundle`
  path unchanged, regression-tested). Conditional echoed-`git rev-parse HEAD` evidence
  check (pinned ⇒ equal; HEAD ⇒ valid 40-hex, recorded on `GeminiVerdict.ReviewedRevision`);
  missing/invalid/mismatch/deadline-exceeded ⇒ `VerificationDegraded` with reason. Caller
  ctx threaded unchanged (pre-cancelled-ctx-reaches-runner test).
- **M4 — CODE + DOCS DONE; LIVE RUN ✅ CONFIRMED (mission iteration 53).** The gated E2E
  `TestLiveCloneOverEgressE2E` (in `managed_agents_live_test.go`) exercises the fetch-by-SHA
  clone-review through the production `Execute` path. It is CI-skipped and SKIPs (never
  fails, never passes) when ADC is absent — verified both gate-off and gate-on-but-no-ADC.
  **Mark #399 ("vertex git clone test granted") authorized the live run; it PASSED** with ADC
  against `ailang-dev`/`global`, pinning a real non-HEAD SHA (`80cbd9612…`): the sandbox
  fetched-by-SHA, checked out `FETCH_HEAD`, echoed the exact pinned SHA, emitted `CLONE_OK`.
  **113.6s, $0.865, 527221 in / 8201 out tokens.** Provider support for `git fetch --depth 1 <sha>`
  is confirmed — no silent fallback to a full clone was ever taken.

### Acceptance criteria (testable)

- [x] **Default unchanged:** `RequiresEgress == false` → environment payload byte-identical
  `{"type":"remote"}` (golden test, no live call). — `TestBuildEnvironment_DefaultByteIdentical`
- [x] **Egress shape pinned:** `RequiresEgress == true` → exactly
  `{"type":"remote","network":{"allowlist":[{"domain":"*"}]}}` (unit/golden test, no live call). —
  `TestBuildEnvironment_EgressShapePinned`
- [x] `RequiresEgress` set on an executor that does NOT advertise `CapNetworkEgress` → loud shared
  pre-dispatch error before any network I/O (unit test with a fake non-capability executor; no
  `Execute`/`ExecuteStreaming` call is made). — `TestValidateTaskCapabilities_EgressOnNonCapableExecutor_LoudReject`
- [x] `ailang exec claude --clone-repo …` (any non-managed_agents resolution) and
  `ailang exec gemini --api-only --clone-repo …` → non-zero exit with a clear error (unit test). —
  `TestCloneFlagValidation_CLIWiring`
- [x] `--clone-sha` without `--clone-repo` → error. — `TestCloneFlagValidation_CLIWiring` / `TestValidateCloneFlags`
- [x] Eval bridge with clone options unset → `BuildDiffBundle` fallback path unchanged (regression test).
  — `TestRunGeminiEvaluator_CloneUnset_FallbackUnchanged` (+ existing `TestRunGeminiEvaluator_StubHappyPath`)
- [x] Eval bridge with `CloneSHA` set and a mismatched `rev-parse HEAD` echo →
  `VerificationDegraded == true` with non-empty `DegradedReason` (unit test with fake runner). —
  `TestRunGeminiEvaluator_ClonePinned_Mismatch_Degraded`
- [x] Eval bridge in HEAD review (`CloneSHA` empty) with a valid non-empty 40-hex `rev-parse HEAD`
  echo → `VerificationDegraded == false`, echoed SHA recorded as the reviewed revision (positive unit
  test — HEAD reviews must pass cleanly). — `TestRunGeminiEvaluator_CloneHEAD_ValidEchoPasses`
- [x] Eval bridge (either mode) with a missing/empty/invalid `rev-parse HEAD` echo →
  `VerificationDegraded == true` with non-empty `DegradedReason` (unit test with fake runner). —
  `TestRunGeminiEvaluator_CloneMissingEvidence_Degraded`
- [x] **Timeout is structured degraded:** a fake runner returning a `context.DeadlineExceeded`-class
  error → `VerificationDegraded == true` with non-empty `DegradedReason` (unit test; never a retry,
  never a clean pass). — `TestRunGeminiEvaluator_CloneDeadlineExceeded_Degraded`
- [x] **Caller ctx honored:** the managed_agents env-builder + clone-review path creates no fresh
  background context — the caller's ctx reaches `sendInteraction` via the existing
  `WithTimeout` propagation and the eval-bridge threading (assertion/unit test). —
  `TestRunGeminiEvaluator_CallerCtxHonored`
- [x] Live-gated E2E (`AILANG_LIVE_MANAGED_AGENTS_MOUNT=1`, skipped in default CI; missing ADC is a SKIP,
  never a pass): sandbox clones the public repo, runs the directive, returns a parsed verdict. —
  `TestLiveCloneOverEgressE2E` **PASSED LIVE (mission iteration 53, Mark #399 authorization)** with ADC
  against `ailang-dev`/`global`, pinning real SHA `80cbd9612…`: fetch-by-SHA → checkout → exact-SHA echo
  → `CLONE_OK` + verdict. 113.6s, $0.865, 527221 in / 8201 out tokens.
- [x] All tests passing; `ailang exec` help + docs updated. — `harness-setup.md` Clone-over-egress
  section + `printExecHelp` in `exec.go`.

### LOC budget

**≤150 LOC production Go** (tests excluded; up from the prior ≤120 by the Mark-approved typed-gate
widening): `internal/executor/executor.go` typed `RequiresEgress` field + `CapNetworkEgress` constant +
`ValidateTaskCapabilities` helper + its CLI/eval-bridge call sites ~15–20; `managed_agents` env builder +
capability advertisement ~30; `cmd/ailang/exec.go` flags/validation/preamble ~40; eval-bridge
options/directive/evidence-check ~55 (≈145 total). If the eval-bridge share grows past ~70, cut scope
there (directive templating stays minimal), not elsewhere.

### Conflict Surface (Phase 2)

Files/symbols touched:

- `internal/executor/executor.go` — **NOW TOUCHED** (reverses the iteration-51 scope call, per Mark
  #399): `Task.RequiresEgress` typed field (struct :37), `CapNetworkEgress` constant (with the existing
  constants :193–213, `Capability` type :190), and the shared `ValidateTaskCapabilities` helper. Every
  executor's `Capabilities()` (interface :24) is READ by the shared validation; only managed_agents
  ADDS the new cap — no other executor's code changes.
- `internal/executor/managed_agents/managed_agents.go` — `envRaw` hardcode at :164/:170 → `buildEnvironment`
  keyed off `task.RequiresEgress`; `Capabilities()` gains `CapNetworkEgress`. The package continues to
  read NO Metadata keys (grep-verified 2026-07-18, empty).
- `internal/executor/managed_agents/types.go` — **unchanged** (`interactionRequest.Environment` is already
  `json.RawMessage` at :38; the builder emits raw JSON, no new wire structs required).
- `internal/executor/managed_agents/managed_agents_test.go` — golden/rejection tests (new).
- `internal/executor/managed_agents/managed_agents_live_test.go` — extends the EXISTING probe harness
  (probes A–R) with the gated E2E; stays manual-only, out of default CI.
- `cmd/ailang/exec.go` — `runExec` flag block (~63–88), read-only use of `resolveAgenticExecutorName` (:317),
  `executeCLI`/task literal (:336/:347–356, sets `RequiresEgress = true` + calls the shared validation),
  help (~677–710).
- `internal/eval_harness/gemini_evaluator_bridge.go` — `EvalOptions`, `RunGeminiEvaluator` (:620),
  degraded-verdict invariant reuse (:551–562), ctx threading unchanged (:594/:620/:682).
- `internal/eval_harness/managed_agents_bridge.go` — **unchanged**; the existing extract-out text bridge is
  reused as-is for verdict return.

**Explicitly NOT touched:** every non-managed_agents executor's own code (their `Capabilities()` lists
are read by the shared validation but not edited), parser/lexer/AST/type-system/eval/VM (no AILANG
language surface at all), and the **motoko core** — this is executor-contract widening plus
executor/eval-harness plumbing in the extension lane; **no core-floor change** (frozen-core boundary
still holds).

Callers that must continue to work unchanged: `ailang exec gemini "<directive>"` (all current flag
combinations, no clone flags) sends byte-identical requests; coordinator/factory callers constructing
`executor.Task` without setting `RequiresEgress` (zero-value `false` — the shared validation is a no-op
for them); `RunGeminiEvaluator` with default `EvalOptions` (diff-bundle
path); injected-runner test seams; `managed_agents_bridge` extract-out behavior; all Claude/OpenAI/
Anthropic/OpenRouter/Ollama `ailang exec` paths.

Syntactic-position analysis: **N/A — no parser/lexer/type change**; the surface is one JSON request field,
one typed `Task` field + one capability constant + one shared validation helper, two CLI flags, and two
eval-option fields.

### Phase-2 premise verification (new claims only; Phase-1/1b log above is unchanged)

| Claim | Status | Evidence |
|-------|--------|----------|
| Egress-enable JSON shape `network.allowlist:[{domain:"*"}]` | VERIFIED-LIVE | Probes Q/R (Phase-1b table) |
| Egress-only sandbox clones the public repo end-to-end | VERIFIED-LIVE | Probe R (`--depth 1`, HEAD `806b3b4a4`) |
| `executor.Task` struct + `Capability` type/constants exist to extend | VERIFIED | `executor.go:37` (Task; `Timeout` :55, `Metadata` :60, `ExtraEnv` :68); `Capability` type :190, constants :193–213 (`CapRemoteSandbox` at :213); `Capabilities()` interface method :24 — re-checked 2026-07-18 |
| `managed_agents` currently reads NO Metadata keys (negative claim) | VERIFIED | grep of `internal/executor/managed_agents/` for Metadata reads, 2026-07-18 — empty |
| `envRaw` hardcode still at `managed_agents.go:164` | VERIFIED | re-checked in working tree 2026-07-18 |
| `WithTimeout` → `sendInteraction` deadline propagation exists to reuse | VERIFIED | `managed_agents.go:178–184` (`timeout := task.Timeout`, fallback `e.timeoutSeconds` from `cfg.TimeoutSeconds` :38); `context.WithTimeout(ctx, timeout)` :183 → `sendInteraction(reqCtx, ...)` :187 |
| Eval bridge threads the caller ctx (no fresh background ctx on the live path) | VERIFIED | `gemini_evaluator_bridge.go` — `EvalRunner` ctx :594; `RunGeminiEvaluator` ctx :620; `DefaultGeminiRunner` ctx :682 |
| `VerificationDegraded ⇒ DegradedReason non-empty` invariant exists to reuse | VERIFIED | `gemini_evaluator_bridge.go:551–562` |
| Arbitrary older SHA is reachable without a full clone | **VERIFIED-LIVE** | Mission iteration 53 (Mark #399 "vertex git clone test granted"): `TestLiveCloneOverEgressE2E` with `AILANG_LIVE_MA_SHA=80cbd9612d8c4f56a9391b4f65cb09249a373230` (a real non-HEAD commit) through the production `Executor.Execute` path — sandbox ran `git fetch --depth 1 origin <sha>` + `git checkout --detach FETCH_HEAD`, echoed the EXACT pinned SHA back (`git rev-parse HEAD` == pinned), emitted `CLONE_OK` + verdict JSON. PASS in 113.6s, $0.865, 527221 in / 8201 out tokens. Provider supports arbitrary-SHA shallow fetch — the last INCORPORATED premise is now live-confirmed. |

### Security

`domain:"*"` is unrestricted outbound egress — the ONLY shape Vertex accepts today (per-domain allowlists
and header `transform` are "not supported now", probes O/P). Accepted because the capability is scoped
to a **read-only reviewer/evaluator cloning a PUBLIC repo at a SHA**: a public clone needs no auth, so
**no secret, PAT, or credential ever enters the sandbox** — there is nothing to exfiltrate but the public
repo and the directive. The private-repo/PAT path (per-domain `transform`, Developer-API surface) stays
**out of scope**. Directives should pin any binary-download URLs/versions. Egress is opt-in per task and
visible in the request payload.

### Generator ≠ judge

This capability makes gemini (Google) a valid **in-sandbox, read-only evaluator/reviewer** — a judge from
a different provider than the Anthropic/OpenAI **generators/executors**, which is the mission's preferred
evaluator direction (Mark, #399). The file-EDITING executor role remains **out of scope**: the sandbox is
Google-hosted server-side (`CapRemoteSandbox`), so agent edits never land on the local worktree — this is
the reviewer/evaluator lane only. A default-evaluator flip is still gated on the charter's ≥3-datapoint
evidence rule (see Future Work).

### Phase-2 risks

| Risk | Impact | Mitigation |
|------|--------|------------|
| Wildcard egress widens sandbox authority | Outbound exfil surface | Opt-in per task; public-repo/read-only scope; no secrets in sandbox (see Security) |
| ~~Programmatic caller sets the Metadata key on another executor~~ MITIGATED by the typed gate | (was: key uninterpreted, silent) | `RequiresEgress` on a non-`CapNetworkEgress` executor → loud shared pre-dispatch error (see Opt-in mechanism) — the iteration-51 hole is closed, not deferred |
| Widening the shared `Task` contract touches all executors' Capabilities surface | Cross-executor churn | Small + additive: only managed_agents ADVERTISES `CapNetworkEgress`; other executors' `Capabilities()` are read, never edited; a compile-time typed field is strictly safer than a stringly Metadata key |
| Interaction deadline too short for the arbitrary-SHA shallow fetch-by-SHA | Review times out | Both clone modes are shallow (`--depth 1`) and bounded by construction; deadline expiry is a structured `VerificationDegraded` with reason (Bounded execution section), never a hang or silent retry; caller tunes `task.Timeout` |
| Agent claims wrong revision reviewed | Verdict on wrong code | Directive echoes `git rev-parse HEAD`; bridge check is conditional: pinned `CloneSHA` ⇒ echo must equal it (mismatch ⇒ `VerificationDegraded`); HEAD review ⇒ echo must be a valid non-empty 40-hex SHA, recorded as the reviewed revision (missing/invalid ⇒ degraded) — never a clean pass on absent evidence |
| Vertex tightens/renames the allowlist contract | Requests rejected | Shape is golden-tested + the live probe harness (A–R) re-verifies cheaply; failure mode is a loud 400, not silent |

---

## ⛔ SUPERSEDED — original mount-based design (historical record only)

**Everything from here through the "Risks & Mitigations" table describes the REFUTED
`repository`/`inline` mount design — the typed source contract, `CapEnvironmentSources` gate, inline
encoder, per-file byte limits, and `--env-repo`/`--env-inline-file` flags. It was refuted by the Phase-1
live spike (the Vertex `interactions` endpoint accepts only `gcs`/`skill_registry` source types) and is
retained ONLY as historical context and the record of what the spike invalidated. Do NOT implement
against it. The approved design is the Phase 2 — Clone-over-egress section above.**

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

#### Phase 2 — Typed source contract, capability gate, and encoder **(SUPERSEDED — do not implement)**

> **⛔ SUPERSEDED:** the typed-source/encoder design below is refuted — see the Phase-1 spike (no
> `repository`/`inline` source types exist on the live endpoint). Retained only as historical context.
> The approved replacement is **Phase 2 — Clone-over-egress capability** above.

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

## Axiom Compliance (scored against the APPROVED clone-over-egress design)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Managed Agents execution is already nondeterministic; the egress payload is a fixed golden-tested JSON shape and the reviewed revision is an explicit SHA. |
| A2: Replayability | +1 | Repo URL + pinned SHA + the clone preamble travel with the directive, and the bridge records the agent-echoed `rev-parse HEAD` — the reviewed input is reconstructable from the task record. |
| A3: Effect Legibility | 0 | No AILANG effect-system change; network/filesystem effects stay inside the provider sandbox boundary. |
| A4: Explicit Authority | 0 | Egress authority is opt-in, per-task, and visible in the request payload — but it is wildcard-wide (the only shape Vertex accepts), so it cannot be scoped to the one repo. Honest downgrade from the mount design's +1; compensated by the no-secrets/public-only scope (Security section). |
| A5: Bounded Verification | +1 | The external contract was pinned by a bounded live probe BEFORE implementation (probes Q/R), and the reviewer now runs `ailang check` against a pinned revision in-sandbox instead of reasoning over packed diff text. |
| A6: Safe Concurrency | 0 | No concurrency or environment reuse introduced; each interaction still gets a fresh environment. |
| A7: Machines First | +1 | Machine-executed in-sandbox `ailang check` on real code replaces a human-oriented prompt-packed text bridge as the verifying path. |
| A8: Minimal Syntax | 0 | Two CLI flags, one typed `Task` field + one capability constant, two eval-option fields; no AILANG syntax. Strictly smaller surface than the superseded mount design. |
| A9: Cost Visibility | 0 | Existing usage/cost reporting unchanged; clone/check time is visible in agent execution duration (probe R: 9 steps). |
| A10: Composability | +1 | The same opt-in serves `ailang exec` CLI calls, the eval-harness evaluator, and future quorum-reviewer callers without provider-specific coupling in shared types. |
| A11: Structured Failure | +1 | Every off-contract state fails loudly: clone flags on a non-supporting executor, `--api-only`, `RequiresEgress` on a non-`CapNetworkEgress` executor (shared pre-dispatch gate), SHA-without-repo, deadline expiry, and missing/mismatched clone evidence (⇒ `VerificationDegraded` with reason). No silent fallback in any direction. |
| A12: System Boundary | +1 | The executor wires only the environment (policy-free, per its recorded contract); directive policy lives at the caller; verdict parsing stays in the eval bridge. Responsibilities remain explicit and unchanged in shape. |

**Net Score: +6** → **Proceed.** No hard-axiom violations (A1/A3/A4/A7 all ≥ 0). A4 is scored 0, not +1,
because Vertex's wildcard-only egress cannot express repo-scoped authority — recorded honestly rather
than claimed away; the mitigation is the public-only/no-secrets scope, not a narrower grant.

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
