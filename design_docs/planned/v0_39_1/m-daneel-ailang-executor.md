# M-DANEEL-AILANG-EXECUTOR — Daneel gets its own executor on the `ailang_only` lane

**Status**: Planned — design freeze ratified 2026-09-16 (D1–D4), ready for sprint planning
**Target**: v0.39.1 (the one AILANG-core item, M1) + fleet/deployment config (registry entry, policy file, task template — no image rebuild) + Daneel-repo changes (capability rung, template, teaching)
**Priority**: P1 — Daneel's host has no lane to lend a capability's question to an agent with "more free reign" (web search, Gemini, multi-step research); today that freedom would have to be granted to the *host itself*, which is exactly the boundary the Host record exists to avoid
**Estimated**: ~3 days (M1 core fix ~0.5d, M2 lane config + template ~1d, M3 Daneel-side capability ~0.5d, M4 end-to-end probe + docs ~1d)
**Dependencies**: [M-AGENT-AILANG-ONLY-EXECUTION](../../implemented/v0_39_0/m-agent-ailang-only-execution.md) (landed v0.39.0 — the lane, the gate, the registry fields); the v0.39.1 `ailang_cli`/`cli_allow`/`.ailang-scratch` batch (in tree, unreleased — this doc's M2/M4 assume it ships first or together)

**Created**: 2026-09-16 · **Author**: attended session with Mark (goal, premises, proposed shape, freeze items D1–D4)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | The executor's whole authority is a static TOML file admitted by a pure (source, policy) check. The same request program is admitted or refused identically on every run; nothing depends on image state (extensions are read from the workspace checkout per task). |
| A2: Replayability | +1 | Every run banks `tool_policy` + `policy_digest` (parent doc M1); the answer program, its manifest and its lock are reproducible artifacts under the recorded policy, and the completion rides the message plane with a correlation ID back to the request. |
| A3: Effect Legibility | +1 | The lending boundary IS an effect row: the executor's program must declare `! {Net, AI, …}` and admission is a row-subset check against `allowed_caps`. Daneel's own abi caps its capabilities at `[IO, FS, Process, Clock, Net, Env, AI]`; this lane is strictly narrower (see D2). |
| A4: Explicit Authority | +1 | This is the doc's thesis. The host does NOT hand its own credentials to the requester; it dispatches to an agent whose authority is a policy file it cannot edit (default-deny, read-only outside the sandbox, parent D4). A capability still cannot name a recipient, model, repo or domain — the *policy* names the domains and the provider, not the capability. |
| A5: Bounded Verification | +1 | Admission is typecheck + row-subset; the answer program can carry `requires`/`ensures` verified by `ai-check` (Z3 now ships in every agent image, v0.39.1). |
| A6: Safe Concurrency | 0 | No concurrency change. Daneel's host dispatches one task per question; the coordinator's per-agent `max_concurrent_tasks` bounds the rest. |
| A7: Machines First | +1 | The request arrives as a typed Request record (Daneel abi v0.4); the answer returns as the completion's `summary` field in a JSON payload a host parses, not prose a human must relay. The executor writes AILANG, whose admission result is structured JSON. |
| A8: Minimal Syntax | +1 | No language change. One CLI semantics fix (`ai_provider` actually applied), one registry entry, one policy file, one template, one Daneel capability rung. |
| A9: Cost Visibility | +1 | The completion carries `model_used`, token counts and `cost_usd`; the policy pins the model (M1) so per-question cost is attributable to a known rung, not the environment's default provider. |
| A10: Composability | +1 | Composes four existing things without new machinery: the `ailang_only` lane, the coordinator's message-triggered dispatch, Daneel's ext/* path-dep packages, and the completion's Summary field. No new plane, no new image. |
| A11: Structured Failure | +1 | A denied program returns `error_kind` JSON; a request the executor cannot satisfy uses the existing `BLOCKED:` transcript marker, which the wrapper maps to `status: blocked` with the reason (verified, coordinator_cloud.go) — distinct from `no_changes` and from `failed`. |
| A12: System Boundary | +1 | The boundary between "Daneel's fixed capabilities" and "research with more free reign" becomes one dispatch hop on the message plane, crossing at a typed Host function on one side and a policy TOML on the other — no shared credentials, no ambient authority. |

**Net Score: +11** → **Decision: ✅ Proceed to implementation** (after the four freeze items are ruled)

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism; policy and template are static config read per task
- [x] A3 (Effects): no hidden side effects — the lane's whole point is that every effect the executor exercises is declared and gated
- [x] A4 (Authority): tightens Daneel's boundary rather than widening it (the host never gains Net/AI itself)
- [x] A7 (Machines First): the answer rides a typed field, not a human inbox ritual

---

## Problem Statement

**Current state (measured premises from the 2026-09-16 attended session, re-verified in this workspace where the code lives here — see Verification Log):**

1. **Daneel's lending boundary is real and works, and it stops exactly where this doc starts.** Daneel's extensions are AILANG packages (`ext/{abi,activity,calendar,design,help,writer}/{register.ail,ailang.toml,_smoke.ail}` in sunholo-data/daneel); `ext/abi/types.ail` v0.4 defines `Authority`/`Request`/`Outcome`/`Host`/`Hooks`; a capability exports `register_with_config`; the `Host` is the lending boundary — `draft/mail/publish/dispatch/compose/project/activityReport/followup` are host-lent functions, and a capability cannot name a recipient, model, repo or domain; effects max out at `[IO, FS, Process, Clock, Net, Env, AI]` (V14, Daneel checkout — sprint re-verifies at M2 start, see below).

2. **Some capability questions need more than the host lends.** "Search the web for X", "ask Gemini about Y", "research Z across multiple sources" exceed a fixed capability's caps and, more importantly, exceed what the *host* should hold itself: giving the host a search key and a Gemini key so a capability can borrow them inverts the boundary — the host becomes the ambient-authority holder. The Host design deliberately has no `name a model` / `name a domain` escape hatch.

3. **The machinery to lend an agent instead already exists and is proven in prod.** The `ailang_only` lane is live on v0.39.0: `tool_policy: ailang_only` + `policy_path` TOML (`allowed_caps`, `fs_sandbox=${WORKSPACE}`, `net_allow`, `net_allow_http`, `process_allow`, `entry`; `cli_allow` ships in v0.39.1 with the `ailang_cli` tool, Z3 in every agent image, `ailang iface std/*` works outside the checkout, `.ailang-scratch/` excluded from wrapper commits). Agents on the lane today: `ailang-only-executor`, `pkg-sunholo-{test-pkg,email,ailang-parse}`. Registry field names: `internal/coordinator/agent_registry.go` `ToolPolicy`/`PolicyPath`; Jobs delivery via `AILANG_TOOL_POLICY` + `AILANG_AGENT_POLICY_TOML` (`executor.MaterializeAgentPolicy`, with `${WORKSPACE}` substitution — verified here, V2/V3/V8). `net_allow` is proven BOTH ways: listed domain → 559 bytes; unlisted → `E_NET_DOMAIN_BLOCKED` returned as the string value (V7, code verified). `process_allow` narrowing proven (`git push` → `NotAllowed`).

4. **One GAP blocks the lane for this use: the policy cannot pin the AI model.** `internal/policy.Policy` has `ai_provider` (policy.go:57, "`stub` or a model name; controls the AI effect handler") but `cmd/ailang/run_policy.go` never reads it: `runPolicyResolved` carries `caps/netDomains/netAllowHTTP/processAllow/sandbox/digest` and nothing else, and the `ailang_run` tool invokes `ailang run --policy <path> <file>` with no `--ai`. An AI-cap program under `--policy` therefore binds no handler from the policy at all — today it either fails (no `--ai` passed, the run path warns "caps AI requires --ai") or, wherever a default provider is configured in the environment, uses that. The lending boundary needs the *policy* to name the model (V1 — verified by reading both files).

5. **The return path exists but must be chosen.** `execute-job` publishes `pubsub.TaskCompletion` (`task_id, agent_id, status, branch, error_msg, changed_files, summary, model_used, session_id` — topics.go:60); completions surface as `Task task-X: <status>` messages in the sender's inbox with a JSON payload including `summary`. `summary` is the transcript **tail, bounded at 1200 bytes** (`completionSummary`, coordinator_cloud_summary.go — verified, V2/V3). How a text ANSWER rides this is freeze item **D1**. Two measured constraints bear on it: a no-diff run reports `no_changes` (a terminal not-success status) unless the agent is registered `acknowledge_only: true` (verified, V4) — an answer service changes no files, so this is load-bearing; and the committed-artifact path has a **live measured failure mode** (backlog 2026-09-15: two daneel design-doc runs completed with `changed_files` on a branch that does not hold the work — pushed-ref mismatch, direct-fix pending; V12).

6. **Daneel's extensions load per task with no image rebuild — but path deps need lock machinery.** The workspace clone gives the model `ext/*` on disk; imports are path dependencies (`'sunholo/daneel_ext_abi' = { path = '../abi' }`). Verified in this repo: without a lock, the self-only package loader resolves only *self-package* imports — a different package name fails with `not found in ailang.lock; run 'ailang lock'` (internal/pkg/loader.go — V6); path deps resolve on disk with no network (internal/pkg/resolver.go — offline-safe, V6b). But `ailang lock` is in the `cli_allow` **deny** list (94-subcommand audit: "plane / registry / provider / spend"), because lock fetches git/registry deps in general. And lock stores path deps as **absolute paths** (backlog 2026-09-15, resolver.go:127 `filepath.Abs()`) — so a lock committed to the Daneel repo would break across machines. The v0.39.1 wrapper excludes `.ailang-scratch/` from commits, which is exactly where a per-task, ephemeral, machine-local lock is harmless.

7. **Transcripts for the lane:** chains chat has none for pi until m-chains-executor-transcripts (planned, ratified) ships; Cloud Logging by execution name meanwhile.

**Impact:** without this lane, every "more free reign" request from a Daneel capability is either refused (the common, silent case — the capability says "I can't do that") or satisfied by widening the host, which is the one design decision Daneel's authority model exists to prevent. The measurement that motivated the lane (first `ailang_only` package-repo run, chain 687d6ebc: 41 tool calls guessing signatures) also shows the lane is *usable* by a model that has never seen the workspace — with `ailang_cli`'s `iface` answering in one call.

---

## Goals

**Primary Goal:** A registry agent (`daneel-executor`) on the `ailang_only` lane that Daneel's host dispatches to when a request exceeds its fixed capabilities, which answers by writing AILANG programs importing Daneel's `ext/*` packages and calling web search / Gemini under a policy that IS the lending boundary — Daneel gets the answer back on the message plane and mails the requester.

**Success Metrics:**

1. **End-to-end probe (M4):** a message mailed to the `daneel-executor` inbox with a well-formed Request record produces, in one task, (a) zero `bash` tool calls in the NDJSON, (b) at least one `ailang_run` admission with `effects_used ⊆ [IO, FS, Clock, Net, AI]`, (c) a completion in the sender's inbox whose `summary` carries the answer, and (d) `model_used` = the policy-pinned model, not an environment default.
2. **Boundary probe (M4):** an answer program declaring `Process` or `Env`, or fetching a domain outside `net_allow`, is refused with `error_kind: policy_violation` / `E_NET_DOMAIN_BLOCKED` — the refusal is the *measured* behaviour, asserted in a test, not assumed.
3. **Model pin (M1):** a policy with `allowed_caps` including `AI` but no `ai_provider` is refused at the gate (mirroring the `fs_sandbox`/`process_allow`/`net_allow` guards); with `ai_provider = "stub"`, an AI-cap program runs against the stub handler under `--policy` with no `--ai` flag. Both asserted in tests.
4. **No rebuild (M2):** a change to Daneel's `ext/*` (or a new `ext/` package) is picked up by the next dispatched task with no image or binary change — asserted by the probe running against a workspace checkout at a moving ref.
5. **Daneel-side (M3):** a capability whose request exceeds its caps calls the host's lent `dispatch()`, reports "dispatched" to the requester, and mails the answer when the completion arrives — measured by one live capability question round-tripped.

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **D1 — Return path for the ANSWER**: completion `summary` (transcript tail, ≤1200 bytes, template-pinned final-message convention) as primary; full transcript at `artifact_gcs_path` as the overflow/debug channel; committed answer artifact and wrapper-sent message rejected for v1 | This is the contract Daneel's host parses. Committed artifacts require the host to do git fetch work and sit on a path with a live measured bug (V12); a wrapper-sent message is new machinery crossing into the message plane from the executor | **human (freeze)** | design | high |
| **D2 — v1 policy grants**: `allowed_caps = [IO, FS, Clock, Net, AI]` (no `Process`, no `Env`); `net_allow` = exactly the domains Daneel's host already lends (Gemini `generativelanguage.googleapis.com` + the search API host — **enumerate from the host code at M2 start, do not copy from memory**); `ai_provider` pinned per D3; `cli_allow` = default set + `lock`; `fs_sandbox = ${WORKSPACE}`; answer programs + manifests + locks live under `.ailang-scratch/` | The policy IS the lending boundary: every cap/domain here is authority the executor holds that a capability does not. `Process`/`Env` are the two caps whose v1 need is unproven; `lock` must be granted for path deps (V6) and must never be committed (V6c) | **human (freeze)** | design | high |
| **D3 — Model + rung**: `glm-5.3-flash` (via the lane's existing route) for every question in v1; a stronger model per capability rung is deferred until measured | Pins cost and quality attribution (`model_used` on the completion); the lane's prompt injection already handles the route that discards the system role | **human (freeze)** | design | med |
| **D4 — Reachability**: v1 = dispatched by Daneel's host only. The executor's template treats a request without a well-formed Request record as `BLOCKED` (reason: "requests arrive via Daneel's host dispatch; this inbox is not a human surface"). No enforcement machinery is built — the message plane has no sender-allowlist and v1 does not pretend otherwise | Whether humans may mail `daneel-executor` directly decides who the template is written for and whether the Request record is load-bearing or advisory | **human (freeze)** | design | med |
| **D5 — M1 semantics**: `run --policy` binds the AI handler from `pol.AIProvider` (same path as `--ai`), refuses `--ai` combined with `--policy` (a widening flag, like `--caps`), refuses `AI` in `allowed_caps` without `ai_provider`, and refuses `ai_provider` set without the `AI` cap (mirroring the `net_allow`/`process_allow` precedent) | This closes the only gap between "the policy states the authority" and "the run enforces it". Provider-level pin (a per-call model name resolving to a different provider is already a typed error in the resolver contract); within-provider per-call latitude is a measured residual, not a v1 fix | **agent** (semantics follow the existing guard pattern; noted for review) | compile | low |

### Design Freeze

Before implementation begins, these must be ruled by Mark (triggers 1 and 4 fire — freeze items + external systems):

- [x] **D1** — **ratified by Mark (attended, 2026-09-16)**: `summary` primary + transcript artifact for overflow; committed artifact and wrapper-sent message rejected for v1
- [x] **D2** — **ratified by Mark (attended, 2026-09-16) WITH ONE CHANGE**: `allowed_caps = [IO, FS, Clock, Net, AI, Process]` — Process narrowed to `process_allow = ["git:status", "git:diff", "git:log"]` (the same read-only set as the package lane) so a question can inspect the daneel repo's history; still no Env. `net_allow` enumerated from Daneel's host; `cli_allow = default + lock`; scratch-dir confinement for answer packages
- [x] **D3** — **ratified by Mark (attended, 2026-09-16)**: `glm-5.3-flash` for v1, per-rung stronger model deferred
- [x] **D4** — **ratified by Mark (attended, 2026-09-16)**: host-dispatch only in v1; malformed Request → `BLOCKED`

---

## Solution Design

### Overview

No new image, no new binary surface beyond the M1 fix. The agent is **configuration**: an `agent-pi` container with the `ailang_only` profile, a workspace cloned from `sunholo-data/daneel`, a policy TOML that states the lending boundary, and a task template that teaches the Request/Outcome contract and where the ext/* packages live. Daneel's side gains one capability rung (or a rung inside an existing capability) whose `handle` dispatches via the host's lent `dispatch()` and reports "dispatched"; the host consumes the completion and mails the requester.

```
Daneel capability                          daneel-executor (agent-pi, ailang_only)
  request exceeds caps                       inbox: daneel-executor
        │                                           ▲ message → coordinator → Cloud Run Job
        ▼                                           │ AILANG_TOOL_POLICY=ailang_only
  Host.dispatch()  ── mail to inbox ────────────────┘ AILANG_AGENT_POLICY_TOML=<daneel-executor.toml>
  reports "dispatched"                               workspace = sunholo-data/daneel clone
                                                    template = templates/daneel-executor-task.md (in the daneel repo)
                                                      │
                                                      ▼
                                              writes .ailang-scratch/<task>/answer.ail
                                              + ailang.toml (path deps 'sunholo/daneel_ext_*' = { path = '../../ext/<pkg>' })
                                              + ailang lock (path-only ⇒ offline)
                                              imports ext/abi/types.ail, calls std/net + AI effect
                                                      │
                                                      ▼
                                              ailang_run → gate admits (row ⊆ [IO,FS,Clock,Net,AI],
                                              domains ⊆ net_allow, handler bound to ai_provider)
                                                      │
                                                      ▼
                                              final message ends with ANSWER block
                                                      │
                                     completion ┌───┴────────────────────────────┐
                                        ▼      ▼                                ▼
                              sender inbox: "Task task-X: completed"   transcript.txt at artifact_gcs_path
                              payload.summary = the ANSWER tail (≤1200 B)   (full text; debug/overflow)
                                                      │
                                              Daneel host polls the plane
                                              (tools/daneel-work.sh), parses summary,
                                              mails the requester via lent mail()
```

### Architecture

**Components:**

1. **The policy file — `policies/daneel-executor.toml`** (deployment config on the coordinator host, delivered as `AILANG_AGENT_POLICY_TOML`). It is the lending boundary, and every line of it is authority a Daneel capability does not hold:

   ```toml
   allowed_caps  = ["IO", "FS", "Clock", "Net", "AI"]   # D2; NO Process, NO Env, NO Msg in v1
                                                            # (Msg is a real runtime effect — the Cognitive OS
                                                            #  fabric, internal/effects/msg.go — and EXCLUDING it
                                                            # is what keeps the answer on the completion path: the
                                                            #  program cannot mail; only the completion mails, V18)
   fs_sandbox    = "${WORKSPACE}"                        # substituted per task by MaterializeAgentPolicy
   net_allow     = [                                     # D2: enumerated from Daneel's host at M2 —
     "generativelanguage.googleapis.com",                #   do NOT copy this list from memory; the host
     # "<search-api-host>",                              #   code is the source of truth
   ]
   net_allow_http = false
   ai_provider    = "<the D3 model>"                      # M1 makes this load-bearing
   cli_allow     = [                                     # v0.39.1 default set plus:
     "lock",                                              # path deps need lock (V6); path-only ⇒ offline (V6b)
   ]                                                      # NOTE: explicit list REPLACES the default set —
                                                          # M2 must write the full default list + "lock"
   entry         = "main"
   ```

   Two correctness notes the implementer must not miss: an explicit `cli_allow` **replaces** the documented default set (an empty list refuses everything), so the file must spell out the default set (`check ai-check iface fmt test docs:search examples builtins pkg-docs tree prompt agent-prompt devtools-prompt policy-check axioms version`) plus `lock`; and the `lock` grant is safe **only because** answer packages live under `.ailang-scratch/` (excluded from wrapper commits, V6c) — a path-dep lock stores absolute paths and must never land in the Daneel repo (V6d).

2. **The registry entry** (coordinator registry YAML — deployment config; the fields are `internal/coordinator/agent_registry.go`):

   ```yaml
   - id: daneel-executor
     label: Daneel executor (ailang_only lane)
     inbox: daneel-executor
     execution_lane: cloud
     repo: sunholo-data/daneel          # the workspace IS the daneel checkout: ext/* on disk, no rebuild
     capabilities: ["research", "answer"]
     tool_policy: ailang_only
     policy_path: <path on the coordinator host>
     acknowledge_only: true             # D1/V4: an answer service changes no files; without this a
                                         # clean answer reports no_changes (a terminal not-success)
     role: researcher                   # or explicit model per D3, once M1 pins ai_provider
     work_tier: 2                       # untrusted content by construction: the Request record is
                                         # sender-controlled message text
     invoke:
       type: prompt
       template_file: templates/daneel-executor-task.md   # resolved relative to the workspace (V10) —
                                                          # the template lives in the daneel repo, so it
                                                          # rides the same checkout the ext/* packages do
   ```

3. **The task template — `templates/daneel-executor-task.md`** (Daneel repo, so it is picked up per task from the workspace checkout and versioned beside the ext/* packages it teaches). It carries: the Request record fields (Daneel abi v0.4, pointing at `ext/abi/types.ail` as the type source of truth), the instruction to import the relevant `ext/*` packages via a scratch package with path deps and to run `ailang lock` before the first run, `skills/daneel-capability-SKILL.md` as the teaching, the **final-message convention for D1** (the message must END with an `ANSWER:` block ≤ ~1100 bytes — the completion's `summary` is the transcript tail, bounded at 1200, and whatever the answer is must survive that bound), the `BLOCKED:` convention (D4: a malformed Request, or a question whose needed domain/cap is outside the policy — the refusal is the honest outcome), and the package context rule (never `ailang check` a package file standalone — MOD010; check within the package; the filed friction's workaround until the pending ailang-core fix lands).

4. **The Daneel-side capability rung** (Daneel repo; one new rung in `ext/ask` or inside an existing capability): `handle` receives a request exceeding its caps, composes the Request record, calls the **host-lent** `dispatch()` targeting the `daneel-executor` inbox, and returns `Outcome` = "dispatched" (the capability still cannot name a model, repo or domain — it names an *inbox*, which is what `dispatch` is for). The host consumes the completion message (it already polls the plane via `tools/daneel-work.sh`) and mails the requester via lent `mail()`. The host-side completion consumer parses the payload's `summary` for the `ANSWER:` block; on `status: blocked`/`failed` it mails the reason instead.

5. **M1 — the model pin** (this repo, the only AILANG-core change): `cmd/ailang/run_policy.go` gains an `aiProvider` field on `runPolicyResolved` and the run path binds the AI handler from `pol.AIProvider` (the same `setupAIHandler` path `--ai` uses), plus three refusals mirroring the existing guards: `--ai` + `--policy` refused as a widening flag; `AI` in `allowed_caps` without `ai_provider` refused ("policy admits AI but sets no ai_provider"); `ai_provider` set without the `AI` cap refused (the `net_allow`-set-without-Net precedent). `ai_provider = "stub"` keeps its documented meaning (stub handler) so the whole thing is testable without a provider.

### Implementation Plan

**M1: Pin the model — `ai_provider` applied by `run --policy`** (~4 hours, this repo)
- [ ] `runPolicyResolved.aiProvider` + the three refusals above; splice it at the existing policy-resolution point (`cmd/ailang/main_run.go:219-228`): `"stub"` → `*aiStubFlag = true`, any other value → `*aiModelFlag = resolved.aiProvider` — the same flag-value splice the other five fields already use, so the AI handler binding in `runFile` needs no change
- [ ] Tests: (a) policy `AI` + `ai_provider = "stub"` runs an AI-cap program against the stub with no `--ai` flag; (b) policy `AI` without `ai_provider` refused, exit 1, reason names the field; (c) `ai_provider` without `AI` cap refused; (d) `--ai` with `--policy` refused as widening
- [ ] Caller audit: the only `--policy` caller today is the `ailang_run` tool (it passes no `--ai`), so the new `--ai`+`--policy` refusal breaks no live caller — assert by grep in the test's comment, per the conflict-surface habit
- [ ] Docs: `agent-tool-policy.md` policy example gains `ai_provider`; the guide's "the WHOLE authority a submitted program can have" table row for AI
- [ ] Note in the admission `policy: {...}` stderr line: the pinned provider, so the transcript states the model the policy chose

**M2: The lane config — policy file + registry entry + template** (~1 day, deployment repo + Daneel repo)
- [ ] Enumerate `net_allow` **from Daneel's host code** (the search API host and the Gemini host the host already lends); record the enumeration in the Verification Log of this doc's implementation report
- [ ] Write `policies/daneel-executor.toml` (full explicit `cli_allow` = default set + `lock`) and register the agent (registry YAML above) — deployment config, `ailang coordinator agents daneel-executor` shows declared vs effective
- [ ] Write `templates/daneel-executor-task.md` in the Daneel repo (component 3); re-verify the abi premises (V14) against `ext/abi/types.ail` at HEAD and fix this doc's implementation report if v0.4 has moved
- [ ] Verify the scratch-package shape actually resolves: a `.ailang-scratch/<pkg>/` with `ailang.toml` declaring `'sunholo/daneel_ext_abi' = { path = '../../ext/abi' }`, `ailang lock` (path-only, offline), then `ailang run --policy` admitting a program importing it — this pins premise 6 end-to-end in a container, not on paper

**M3: The Daneel side — capability rung + host completion consumer** (~4 hours, Daneel repo)
- [ ] The dispatch rung (component 4): compose Request record → lent `dispatch()` → `Outcome` "dispatched"
- [ ] Host completion consumer: parse `summary`'s `ANSWER:` block; mail the requester; handle `blocked`/`failed` by mailing the reason
- [ ] `_smoke.ail` for the rung (every ext package carries one — the convention stands)

**M4: The end-to-end probe + boundary assertions** (~1 day)
- [ ] Live round trip (Success Metrics 1–2): a real capability question dispatched, answered, mailed — asserted on the NDJSON (zero `bash`), the admission JSON, and the inbox completion payload
- [ ] Negative probes: unlisted domain → `E_NET_DOMAIN_BLOCKED` as the string value; `Process`/`Env` declaration → `policy_violation`; malformed Request → `BLOCKED` completion (D4)
- [ ] No-rebuild probe (Success Metric 4): touch an `ext/` package in the workspace between two dispatched tasks; the second task sees the change
- [ ] Docs: a short `docs/docs/guides/` note (or a section in `agent-tool-policy.md`) on lending an executor to an external host; CHANGELOG entry for M1

### Files to Modify/Create

**New files (not in this repo unless stated):**
- `policies/daneel-executor.toml` — deployment/coordinator host (~25 lines)
- registry YAML entry — deployment repo (~20 lines)
- `templates/daneel-executor-task.md` — **sunholo-data/daneel** (~80 lines)
- Daneel: the dispatch rung + host completion consumer + `_smoke.ail` (~100 lines AILANG)
- This repo: `cmd/ailang/run_policy_test.go` additions or a new `run_policy_ai_test.go` (~120 lines)

**Modified files (this repo):**
- `cmd/ailang/run_policy.go` — `aiProvider` on `runPolicyResolved`, three refusals, wiring (~+35)
- `cmd/ailang/main_run.go` — the `--policy` splice block (lines ~219-228): set `*aiStubFlag`/`*aiModelFlag` from `resolved.aiProvider` (~+6)
- `docs/docs/guides/agent-tool-policy.md` — `ai_provider` row + the lending-an-executor note (~+25)
- `changelogs/v0.32-current.md` — M1 entry (~+10)

---

## Examples

### Example 1: the whole round trip

A calendar capability is asked: *"find a 30-minute slot next week that works for everyone in the Munich office and check the weather forecast for the travel window."* Weather is outside every cap it holds, and outside what the host lends.

```
capability handle():  request exceeds caps
  → Host.dispatch(inbox: "daneel-executor",
                  payload: Request{ question, requester, authority, context })
  → Outcome "dispatched — task will mail you"

daneel-executor (Cloud Run Job, workspace = daneel clone):
  read templates/daneel-executor-task.md (workspace-resolved)
  write .ailang-scratch/weather-slot/ailang.toml   ('sunholo/daneel_ext_calendar' = { path = '../../ext/calendar' })
  ailang_cli: ["lock"]                              (path-only, offline)
  write .ailang-scratch/weather-slot/answer.ail    (imports ext/abi/types.ail, ext/calendar; ! {IO, FS, Clock, Net, AI})
  ailang_run: admitted — row ⊆ [IO,FS,Clock,Net,AI], domains ⊆ net_allow, handler = policy's ai_provider
  program: calls the search API (listed host), calls Gemini (listed host), computes the slot
  final message: "... \nANSWER: Tue 14:00–14:30 CET works for all six attendees; forecast dry, 18°C."

completion → sender inbox: "Task task-a1b2: completed"
  payload.summary = "ANSWER: Tue 14:00–14:30 CET works for all six attendees; forecast dry, 18°C."
  payload.model_used = <the D3 model>    (not an environment default — M1)
Daneel host: parses summary, mails the requester via lent mail()
```

### Example 2: the refusal IS the boundary working

The same executor is asked to also "push the updated agenda to the repo". The answer program declaring `FS` + `Process` is refused at the gate: `error_kind: policy_violation, missing_from_policy: Process` (D2 grants no Process). The model narrows — it computes the agenda content and returns it in the `ANSWER:` block; the *push* stays with whoever holds that authority. Daneel's boundary was never consulted on the executor's authority, and the executor never held Daneel's.

---

## Success Criteria

- [ ] SM1–SM5 (Goals), each with its measurement named (NDJSON, admission JSON, inbox payload, container probe)
- [ ] M1's four tests green; `ailang run --policy` with `ai_provider = "stub"` runs an AI-cap program with **no** `--ai` flag
- [ ] `ailang coordinator agents daneel-executor` shows `tool_policy: ailang_only` declared and effective, and `policy_digest` non-empty on the banked row
- [ ] A policy-domain violation and a cap violation both bank under named error categories — never `api_error`
- [ ] The registered agent is `acknowledge_only: true` and a clean answer banks `status: completed`, not `no_changes` (V4 — assert this in the probe; it is the one registry line most likely to be "obviously" omitted)
- [ ] No new env-var routes; `make simplicity-audit` unchanged (the policy arrives on the existing `AILANG_AGENT_POLICY_TOML`)
- [ ] Daneel repo: rung smoke passes; one live capability question answered end-to-end
- [ ] Docs updated (guide section + changelog); this doc moved to implemented with the enumeration record

## Testing Strategy

**Unit tests (this repo, M1):**
- The four refusal/admission cases in `cmd/ailang/run_policy_*test.go`; assert exit codes AND the named reason strings (the agent reads the reason)
- Parity guard: the policy TOML example in the guide and the test fixtures share the `ai_provider` field name (the parent doc's V8 lesson — the `resident-run`/`--caps` caps drift: one concept, two spellings)

**Integration (M2/M4, container):**
- The scratch-package resolution probe (path dep + `lock` + `run --policy` admission, inside an `agent-pi` image) — this is the premise-6 pin, run where the lane runs
- Boundary assertions: `E_NET_DOMAIN_BLOCKED` for an unlisted host; `policy_violation` for `Process`/`Env`
- `docker/test-agent-pi.sh` gains a case only if M1 changes tool-visible output (the admission stderr line)

**Manual / live:**
- The M4 end-to-end round trip, asserted on the message plane (the inbox payload), Cloud Logging by execution name for the transcript until m-chains-executor-transcripts ships

## Deferred Decisions

- **Within-provider per-call model latitude** (a program naming a different model of the *same* pinned provider) — agent may choose whether to measure it in v1; the provider-level pin is the boundary; a per-call model allowlist would be new machinery
- **Whether the executor gets `Env`** once a measured task needs it (D2 excludes it from v1; adding it later is a one-line policy change, which is the point of the policy being the boundary)
- **A stronger model per capability rung** (D3's second half) — deferred until round-trip quality is measured on `glm-5.3-flash`
- **Pi TS extensions for the executor** (per-agent `extensions:` list, workspace `.ts` files appended to `-e`) — NOT needed for v1: Daneel's extensions are AILANG packages; deferred until a measured need
- **Answer size beyond the Summary bound** — if 1100 bytes proves too tight in practice, the escalation is a *wrapper-committed answer artifact under a bounded path*, reusing `artifact_patterns`; that decision waits for the measurement, and the V12 pushed-ref bug is fixed first either way
- **Sender-allowlist on the inbox** (D4's enforcement half) — the message plane has none today; if direct human reach is ever blessed, the enforcement question returns

## Non-Goals

- **Widening Daneel's host.** The host gains no keys, no domains, no models. It gains one dispatch call it already has (`Host.dispatch` is a lent function today).
- **A pi TS extension surface for the executor.** Daneel's extensions are AILANG; the lane profile already carries the tools.
- **Enforcing Daneel's abi contract in the gate.** The gate enforces AILANG policy; the Request/Outcome record is the *template's* contract, enforced by the model and checked by the Daneel side, not by `internal/policy`.
- **Generic "external host executor" machinery.** This doc builds one named agent for one named host; extracting a general shape waits for a second host.
- **Fixing the MOD010 standalone-check friction or the lock absolute-path bug.** Both are filed (ailang-core approval pending / backlog direct-fix); this doc works around both via package-context checking and scratch-dir confinement.

## Timeline

| Day | Work |
|---|---|
| 1 | M1 (pin + tests + guide row) · freeze items ruled by Mark |
| 2 | M2 (enumerate domains from the host, policy file, registry entry, template, scratch-package probe in container) |
| 3 | M3 (Daneel rung + host consumer + smoke) · M4 (end-to-end probe, boundary assertions, docs, changelog) |

Realistic: 3 days assumes v0.39.1 (`ailang_cli`, `.ailang-scratch`) ships first or with it. If the scratch-package resolution probe finds a surprise (lock refusing path-only manifests, resolver walking out of the sandbox), add a day — that probe is deliberately first in M2.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|------------|
| The `summary` tail-capture misses the answer (model ends with prose, not the `ANSWER:` block) | High — D1's whole contract | The template states the convention in the same breath as the tools; M4's probe asserts the payload `summary` contains `ANSWER:`; if the model flakes on it, that is a template/prompt iteration, not a protocol change |
| `glm-5.3-flash` cannot write a correct multi-dep scratch package first try | Med | The lane prompt already injects caps/sandbox/domains; the template carries the exact manifest lines to copy; `ailang_cli iface` answers signature questions in one call (the 41-guessing-call measurement is the cautionary tale) |
| Path-dep + lock + sandbox interaction misbehaves in the container | Med | The M2 probe runs it before anything else; `.ailang-scratch/` is inside `fs_sandbox` by construction; the lock stays out of commits by the wrapper's existing exclusion |
| `net_allow` enumeration from memory instead of the host code | High — an unlisted domain = every research answer refused; an over-broad list = boundary widened by sloppiness | M2 task 1 makes enumeration an explicit recorded step against the host source; the negative probe asserts one unlisted domain fails |
| A capability round-trips questions so freely the lane becomes a general chatbot (cost) | Med | One question = one Cloud Run Job with banked `cost_usd`; D3's cheap-first model; the Daneel side can rate-limit per requester once measured |
| Daneel's abi moves past v0.4 and the template's field names go stale | Low | The template points at `ext/abi/types.ail` as the source of truth and the template itself rides the same checkout — a moved field is visible in the same diff |
| Completion `status: no_changes` ships because `acknowledge_only` was omitted | Med (silent misreport: a good answer read as not-success) | Success criterion asserts the completed status explicitly; V4's comment in code documents the mechanism |

## Related Documents

**Implemented (may inform design):**
- [m-agent-ailang-only-execution.md](../../implemented/v0_39_0/m-agent-ailang-only-execution.md) — the lane, the gate, the registry fields, D4 (default-deny policy), the profile; this doc is a *consumer* of all of it
- [m-agent-safe-runner.md](../v1_1_0/m-agent-safe-runner.md) — the Go admission gate this lane runs on; M3 (runner daemon over the message bus) is a later shape of the same boundary

**Planned (check for overlap):**
- [m-agent-loop-architecture.md](../v1_1_0/m-agent-loop-architecture.md) — where a multi-turn tool loop lives; distinct (that is AILANG-side loop design, this is a fleet agent on an existing loop)
- [m-chains-executor-transcripts.md](../v0_39_1/m-chains-executor-transcripts.md) — transcripts for the lane; until it ships, Cloud Logging by execution name
- [m-completion-path-parity.md](../m-completion-path-parity.md) — the completion path this doc's D1 rides; its effects (approval records etc.) are the executor's existing behaviour, unchanged here

**External:**
- sunholo-data/daneel — `ext/abi/types.ail` (v0.4), `ext/{abi,activity,calendar,design,help,writer}/`, `design_docs/daneel-calendar.md`, `authority.md`, `skills/daneel-capability-SKILL.md`, `tools/daneel-work.sh`
- memory: `project-ailang-only-lane-friction-routing` (the lane-friction routing memory)

## References

- [Agent tool policy guide](../../../docs/docs/guides/agent-tool-policy.md) — the lane, the policy file, delivery, banked fields
- `internal/policy/policy.go` — the Policy struct incl. `AIProvider`; `cmd/ailang/run_policy.go` — the derivation this doc's M1 extends
- `internal/pubsub/topics.go` — TaskCompletion incl. `Summary`; `cmd/ailang/coordinator_cloud_summary.go` — the 1200-byte tail capture
- `internal/executor/agent_policy.go` — `${WORKSPACE}` substitution, read-only materialisation outside the sandbox
- `internal/pkg/loader.go` / `internal/pkg/resolver.go` — self-only vs locked resolution (why `lock` is needed for path deps and offline-safe for path-only)
- Daneel design: `authority.md` (the Host lending boundary), `daneel-calendar.md` (a capability that will use this lane)

## Future Work

- A **question profile** (`read` + `ailang_check` only) and a **research profile** (this doc's lane) as named profiles beside `ailang_only`, if more executors of this shape follow
- Per-call model allowlisting within the pinned provider, if the measured residual matters
- The wrapper-committed answer artifact with a bounded path, if the Summary bound proves too tight
- m-agent-safe-runner M3 shape: the gate as a standing service the host posts programs to, rather than a per-task agent — the boundary stays identical, the dispatch gets cheaper
- Extracting the "external host lends an executor" pattern once a second host exists

---

## Verification Log

Two classes of row, kept honestly separate: **VERIFIED HERE** (grep/read against this checkout, 2026-09-16) and **INHERITED** (measured premises from the attended session / the task brief, whose artifacts live in the Daneel checkout or the prod store — the sprint re-verifies the Daneel-repo rows at M2 start and records the result in the implementation report).

| # | Claim | Method | Result |
|---|-------|--------|--------|
| V1 | The GAP: `policy.Policy` has `ai_provider` but the `--policy` run path never applies it | read `internal/policy/policy.go:57` (`AIProvider string \`toml:"ai_provider"\``, comment: "controls the AI effect handler") and all of `cmd/ailang/run_policy.go` — `runPolicyResolved` = {caps, netDomains, netAllowHTTP, processAllow, sandbox, digest}, no provider; read the splice point `cmd/ailang/main_run.go:219-228` — the policy resolves into `*capsFlag/*netAllowDomainsFlag/*netAllowHTTPFlag/*processAllowlistFlag` and nothing else; read the tool: `ailang-exec.ts` builds `["run", "--policy", path, …, file]` — no `--ai`; read `cmd/ailang/ai_handlers.go:31` — the handler binds from the `--ai` flag (`runFile`'s `AIHandler` closure passes `aiModel` through), warning "caps AI requires --ai" when absent | **VERIFIED HERE** — the premise holds, and is sharper than stated: under `--policy` with no `--ai` there is no handler from *anywhere* unless the environment configures one. M1's wiring point is exactly the `main_run.go` splice: `*aiModelFlag = resolved.aiProvider` (`"stub"` → `*aiStubFlag = true`) |
| V2 | A text answer can ride the completion: `summary` = transcript tail, bounded 1200 bytes | read `internal/pubsub/topics.go:60-108` (TaskCompletion fields incl. `Summary`); `cmd/ailang/coordinator_cloud.go` (`completion.Summary = completionSummary(execResult.Transcript)`); `cmd/ailang/coordinator_cloud_summary.go:16-42` (`completionSummaryMax = 1200`, tail + "full text at the artifact path") | **VERIFIED HERE** — D1's primary channel exists exactly as premised |
| V3 | The completion surfaces in the sender's inbox with the summary in a JSON payload | read `internal/coordinator/pubsub_completion_handler.go:195-229` — payload incl. `"summary"`, title `Task <id>: <status>`, `CorrelationID: task.MessageID`, posted via `InsertInboxMessage` to the sender-resolved inbox | **VERIFIED HERE** |
| V4 | A no-diff run reports `no_changes` (terminal, not-success) unless the agent is registered `acknowledge_only: true` | read `cmd/ailang/coordinator_cloud.go:261` (`expectChanges := !config.AcknowledgeOnly()`) and `internal/coordinator/agent_registry.go` `AcknowledgeOnly` comment ("probes and acknowledgement tasks") | **VERIFIED HERE** — the registry entry MUST set `acknowledge_only: true`; the most likely silent failure of the whole design is omitting this one line |
| V5 | `BLOCKED:` in the transcript maps to `status: blocked` with the reason, only when nothing changed | read `cmd/ailang/coordinator_cloud.go:270-286` + `internal/coordinator/task_blocked.go:57` (`ParseBlockedMarker`) | **VERIFIED HERE** — D4's malformed-Request path uses existing machinery |
| V6 | Path deps need lock: the self-only (lockless) package loader resolves only self-package imports; another package name fails with "not found in ailang.lock; run 'ailang lock'" | read `internal/pipeline/package_resolver.go:93-125` (authoring fallback, self-only) and `internal/pkg/loader.go:44-64` (`isSelfReference` else the lock error) | **VERIFIED HERE** — premise 6's "lock machinery is needed for path deps" is code, not folklore |
| V6b | `ailang lock` with path-only deps is offline-safe | read `internal/pkg/resolver.go:131-140` — path deps resolve relative to the package dir on disk; the registry/network fetch arm is only for non-path deps | **VERIFIED HERE** |
| V6c | A lock written under `.ailang-scratch/` never lands in the repo | read `cmd/ailang/coordinator_cloud_scratch.go` — `ScratchDir = ".ailang-scratch"`, `stageForCommit` excludes it at any depth, commits only what is staged | **VERIFIED HERE** — the wrapper excludes it; this is why the doc confines answer packages there |
| V6d | A committed path-dep lock would break across machines | backlog row 2026-09-15, `resolver.go:127 filepath.Abs()` (m-pkg-lock-portability deliberately kept path deps absolute); cited as measured | **INHERITED** (backlog, this repo) — the confinement makes it moot for this lane |
| V7 | `net_allow` is enforced at the handler and the refusal is a typed, string-valued error | read `internal/effects/net.go:79` `E_NET_DOMAIN_BLOCKED: domain not in allowlist: <host>`; `std/net.ail:53` documents it; both-direction behaviour (559 bytes listed / blocked unlisted) is the attended session's prod measurement | **Code VERIFIED HERE; live behaviour INHERITED** |
| V8 | Delivery plumbing: registry `ToolPolicy`/`PolicyPath` → `AILANG_TOOL_POLICY` + `AILANG_AGENT_POLICY_TOML` → `MaterializeAgentPolicy` (read-only outside the workspace, `${WORKSPACE}` substituted) | read `internal/coordinator/agent_registry.go:157-165`, `internal/config/job.go:36-37`, `internal/dispatch/cloudrun/dispatcher.go:291-296`, all of `internal/executor/agent_policy.go` | **VERIFIED HERE** |
| V9 | The guard pattern M1 mirrors exists: FS needs `fs_sandbox`, Process needs `process_allow`, Net needs `net_allow`, cross-set refused | read `cmd/ailang/run_policy.go` (all five refusals) | **VERIFIED HERE** |
| V10 | A `template_file` resolves relative to the workspace, so the template can live in the Daneel repo | read `internal/coordinator/agent_registry.go:43-52` (ResolveTemplate: absolute / `~/` / workspace-relative) | **VERIFIED HERE** |
| V11 | `cli_allow`'s default set excludes `lock`; an explicit list REPLACES the default; execution subcommands are refused even when listed | read `cmd/ailang/pi_assets/ailang-exec.ts:96-133` (DEFAULT_CLI list, `cliDecision`) + changelog Unreleased (94-subcommand audit: `lock` in the deny column) | **VERIFIED HERE** |
| V12 | The committed-artifact return path has a live measured failure mode | backlog row 2026-09-15: two daneel design-doc runs completed with `changed_files` on a branch that does not hold the work (pushed-ref mismatch, direct-fix pending) | **INHERITED** (backlog, this repo) — feeds D1's rejection of the artifact path for v1 |
| V13 | The lane's model route (`ollama/glm-5.3-flash:cloud`) discards the system role entirely, so the lane injects its teaching as BOTH a system-prompt section and a conversation message — D1's template convention must survive a route with no system role | read `docs/docs/guides/agent-tool-policy.md` ("What the model is told") | **VERIFIED HERE** (guide) — the template's load-bearing lines (ANSWER convention, BLOCKED convention) must be in the conversation message, not only the system prompt |
| V14 | Daneel's extensions are AILANG packages; abi v0.4 defines Authority/Request/Outcome/Host/Hooks; capabilities export `register_with_config`; host-lent functions are draft/mail/publish/dispatch/compose/project/activityReport/followup; a capability cannot name a recipient, model, repo or domain; effects max `[IO, FS, Process, Clock, Net, Env, AI]` | attended session 2026-09-16 against sunholo-data/daneel; **the Daneel checkout is not present on this authoring workspace** | **INHERITED — sprint re-verifies at M2 start** against `ext/abi/types.ail` at HEAD; if v0.4 has moved, the template (not the policy) is what changes |
| V15 | Daneel's host reads the plane via `tools/daneel-work.sh` (message-store env) and can mail/dispatch | attended session; Daneel checkout not present here | **INHERITED — re-verify at M3** |
| V16 | The lane is live in prod on v0.39.0 with the named agents; v0.39.1 (`ailang_cli`, `cli_allow`, Z3-in-image, `iface std/*` outside checkout, `.ailang-scratch/` exclusion) is in tree and unreleased | `std/VERSION` = v0.39.0; changelog Unreleased carries the v0.39.1 items; this checkout's HEAD = 207c3421, whose commit message ships the `ailang_cli` tool | **VERIFIED HERE** — the doc targets v0.39.1 and depends on that batch shipping |
| V17 | No sender-allowlist exists on inboxes (D4's enforcement half is absent by construction) | read the dispatch path: a message to an inbox triggers a task; no sender filter in `pubsub_completion_handler.go`/registry dispatch | **VERIFIED HERE** — D4 is a stance/documentation decision in v1, honestly not an enforcement one |
| V18 | "The program cannot send messages" holds because the POLICY excludes `Msg`, not because no such effect exists: the runtime has a `Msg` effect (`RegisterOp("Msg", "send", …)`, Cognitive OS fabric; native CLI binds a Store-backed handler per m-cog-runtime) — an answer program granted `Msg` could in principle write to the plane, so D2's cap list is the load-bearing exclusion | read `internal/effects/msg.go:1-40` (Msg effect, handler configuration paths) and `:312-315` (send/recv ops) | **VERIFIED HERE** — the exclusion is a policy line the freeze names, not a runtime impossibility; the D2 cap list deliberately omits `Msg` |

---

## Quorum

**Triggers fired: 2 of 4** — (1) **design-freeze items D1–D4** (return-path contract, the authority grant itself, model, reachability) and (4) **external systems** (Gemini/search domains enumerated from another repo's host code; the Daneel repo and the deployment registry are outside this checkout). Triggers 2 (shared machinery) and 3 (banked schema) do not fire: the only AILANG-core change (M1) extends an existing guard pattern in one file, and no banked-row schema changes.

**Not yet run.** Premises V14/V15 are inherited and the sprint re-verifies them at M2/M3 start; V1–V13, V16, V17 are verified in this log. Reviewers as the parent doc used: `gpt5-6-sol`, `gemini-3-1-pro`, `oc-glm-5-2`. **D1–D4 must be ruled by Mark before the sprint starts** — the quorum can precede, accompany or follow the freeze, but the sprint plan may not start execution on an unchecked freeze item.

---

**Document created**: 2026-09-16
**Last updated**: 2026-09-16