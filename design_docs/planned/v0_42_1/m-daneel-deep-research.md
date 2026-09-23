# M-DANEEL-DEEP-RESEARCH — the daneel-executor holds `sunholo/gemini_agents` (Google Deep Research on the ailang_only lane)

**Status**: Planned
**Target**: v0.42.1 (core M1) + fleet config in `ailang-multivac` (policy TOML, job secret/env, IAM) + Daneel-repo template
**Priority**: P1 — Daneel's host answers research questions today through its own ask route (daneel#152); a request *dispatched to the executor* still cannot research at all, which is exactly the asymmetry the lane exists to remove
**Estimated**: ~4–5 days (M1 core ~1.5 d, M2 fleet ~1 d, M3 template ~0.5 d, M4 probe + docs ~1 d, buffer)
**Dependencies**: [M-DANEEL-AILANG-EXECUTOR](../../implemented/v0_39_3/m-daneel-ailang-executor.md) (landed — the lane agent, policy, template, dispatch rung); [M-EXECUTOR-POLICY-HARDENING](../../implemented/v0_41_0/m-executor-policy-hardening.md) (landed — restricted mode, worker env scrubbing, CheckLanePolicy); `sunholo/gemini_agents@0.1.0` on the registry (published 2026-09-21, depends on `sunholo/gcp_auth@0.8.1`)

**Created**: 2026-09-23 · **Author**: design-doc-creator session (coordinator task `task-28db791a`, issue #1267, Daneel message `inbox_1789993952527_28db791a`)

---

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | +1 | The two new authority knobs (which env vars, how long a Net request may wait) become **policy fields** resolved before the worker starts — the same policy admits the same program with the same bounds every run. Today `--net-timeout` is caller-settable beside `--policy` (V10); after M1 the policy owns it and the caller's copy is refused by name. |
| A2: Replayability | +1 | `env_allow` and `net_timeout_ms` ride the policy digest (the digest is over the TOML bytes) and are banked in the `policy: {...}` admission line (V11) — a replayed answer program is bound by the recorded bounds, not the deployment's mood. |
| A3: Effect Legibility | +1 | The package's effect row becomes *legible through the gate* instead of unavailable: `{FS, Net, Env}` is a declared, row-subset-checked row (V1, V2). The Env effect keeps its existing typed `NotAllowed` result and snapshot semantics — no hidden reads (V5). |
| A4: Explicit Authority | +1 | This is the doc's spine. The policy names the two hosts, the two env vars, and the per-request patience; the caller cannot widen any of them; the metadata server — the one ambient authority Cloud Run offers — stays **closed** (V8), so ADC reaches the program only as an operator-mounted credential inside the sandbox, not as an ambient token tap. |
| A5: Bounded Verification | 0 | Admission stays typecheck + row-subset; no verification-path change. |
| A6: Safe Concurrency | 0 | No concurrency change; one research question = one dispatched job, polls are separate bounded runs. |
| A7: Machines First | +1 | The model's whole surface stays typed tools: the `lock` op gains the same `Path` field shape `check`/`fmt` already use (V12), the lane summary tells it exactly which env vars it may read, and the package exports typed `start/get/delete` + answer-parsing functions the template teaches verbatim. The answer rides the `ANSWER:` block, not prose. |
| A8: Minimal Syntax | +1 | No language change. Two policy TOML fields, one CLI schema field, one supervisor env splice — all reusing shapes that already exist (V3, V5, V12). |
| A9: Cost Visibility | +1 | Deep Research is the first *dollars-per-run* effect on the lane ($1–3 by Google's estimate; V16): the template requires `usageOf` (tokens, searches) in the ANSWER block, the task's existing `AILANG_MAX_COST_USD` bounds the loop, and `delete` after harvest bounds Google-side storage (the package documents that interactions are **stored until deleted**). |
| A10: Composability | +1 | Composes five existing things — the registry package, `gcp_auth`'s ADC token, the policy gate, the scratch-package flow, the completion `ANSWER:` convention — with no new machinery beyond the two policy fields. |
| A11: Structured Failure | +1 | Every new refusal is named: `NotAllowed` for an env var outside `env_allow` (the effect's existing typed error), `E_NET_DOMAIN_BLOCKED` for an unlisted host, a named resolve refusal for a policy that admits Env without naming vars, and the package's own `Err`/`transient` contract for API failures (one run in six is a transient `api_error`; V16). |
| A12: System Boundary | +1 | Mark's ruling is preserved as architecture: the research **runs in Daneel's own project** (`start("sunholo-daneel", …)`; the SA's `roles/aiplatform.user` is on `sunholo-daneel`), while the plane's executor holds only what its policy names. Daneel's host widens by nothing. |

**Net Score: +10** → **Decision: ✅ Proceed to implementation** (after the freeze items are ruled)

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism; the only timing knob moves *into* the pinned policy
- [x] A3 (Effects): no hidden side effects — Env reads are snapshot-pinned and allowlist-refused; no new effect is invented
- [x] A4 (Authority): tightens — the caller-settable `--net-timeout` beside `--policy` is *removed* (M1), and the metadata-server ambient path stays closed
- [x] A7 (Machines First): typed tools, typed refusals, banked bounds; nothing is made human-pretty at the machine's expense

---

## Problem Statement

**Current state:**

1. **Daneel's host can research; the plane's executor cannot.** `sunholo/gemini_agents@0.1.0` (registry, 2026-09-21) wraps Google's managed Deep Research agent through the Vertex AI Interactions API under ADC: `start`/`get`/`list`/`cancel`/`delete`, returning a cited report (`url_citation` sources) plus `usage`. Daneel's host uses it today as the ask route's backend (daneel#152), running as the `daneel` account in `sunholo-daneel`. The `daneel-executor` — the ailang_only lane agent built for exactly the "request exceeds the host's caps" case — has no route to it: its policy names neither the package's caps nor its hosts, so every dispatched research question is answered from the model's own weights or refused.

2. **The brief's three items are necessary but not sufficient — the package's effect row does not fit the lane today.** The measured gap (V1): the package's `interaction` module is `{FS, Net, Env}` — ADC via `sunholo/gcp_auth`, whose token path is the ADC **file** refresh exchange (hence the package's two hosts, `aiplatform.googleapis.com` and `oauth2.googleapis.com`) or the metadata server. But the lane REQUIRES a restricted policy (`executor.CheckLanePolicy`, M-EXECUTOR-POLICY-HARDENING D3), and restricted mode **cannot admit `Env` at all** — `RestrictedEffects` lists IO, FS, Net, Clock, Rand, Stream, Process, AI and *no others* (V2); a policy naming Env is refused at resolve with "no confined adapter". The parent doc's deferred item ("whether the executor gets `Env` … adding it later is a one-line policy change") was **wrong on its premise** (V3): without new machinery there is no line to add. Going `trusted_host` instead is refused for the lane by `CheckLanePolicy` (V4) — so the feature is impossible without an AILANG-core change, whatever the fleet config says.

3. **The ADC path the package needs is closed on the lane by design — and the open path needs a credential the policy must name.** The worker's process environment is scrubbed to `workerEnvAllow` + the pinned AI provider's credential vars (V6); `GOOGLE_APPLICATION_CREDENTIALS` reaches the worker only when the policy admits AI with a Google provider (V7). The metadata server — the ambient alternative — is refused by the Net effect's IP rules with **no policy grant path** (the `--net-allow-metadata` flag is refused under `--policy`: "restricted mode has no metadata grant") (V8). So the package's file path is the only viable one, and it needs (a) the `Env` cap to read `GOOGLE_APPLICATION_CREDENTIALS`/`GOOGLE_CLOUD_PROJECT`, (b) the credential file mounted **inside `fs_sandbox`** (FS reads outside are refused), and (c) both names admitted by name.

4. **Net patience is not policy-pinned and not delivered.** `--net-timeout` defaults to 30 s per request (V9); a cold project's first Interactions POST has been seen to hang 60 s (V16). The flag is caller-settable beside `--policy` today (V10) — a widening wart — but the lane's only execution route (`ailang_run`) passes no such flag and hard-caps the whole invocation at 130 s (V9), so the deployed policy cannot express "this lane may wait 90 s for a cold POST" at all.

5. **Registry dependencies cannot be materialised by the lane.** The answer program imports `pkg/sunholo/gemini_agents/…`, which requires the answer package to carry a lock naming it and the registry tarball to be in the local cache (V13). The policy CLI tool's `lock` op runs **at the sandbox root** and takes no path (V12), so it cannot lock a package in `.ailang-scratch/<task>/` (the ratified answer-package location); `ailang install` exists as a CLI command but is not in the tool's schema at all, so the agent cannot fetch the package either (V13). The lane's documented recipe ("run `ailang lock` before the first run") works for repo-root packages only — a live friction for scratch packages generally.

**Impact:** every "research this properly" request Daneel dispatches to the plane is answered without sources, or not answered. The lane — built so a fixed-capability host can lend a question to an agent with "more free reign" — currently has *less* free reign for research than the host doing the lending, which inverts the design's whole point.

---

## Goals

**Primary Goal:** the `daneel-executor` answers dispatched research questions by writing AILANG programs that drive `sunholo/gemini_agents` (start → poll → harvest cited report → delete) under a restricted policy that names every new authority — two hosts, two env vars, one credential file, one per-request timeout — with Daneel's research running in `sunholo-daneel` per Mark's ruling.

**Success Metrics:**

1. **End-to-end probe (M4):** a dispatched research Request produces, in one task, (a) a scratch answer package whose manifest pins `sunholo/gemini_agents = "0.1.0"`, locked via the CLI tool **with a path**; (b) at least one `ailang_run` admission whose banked line shows `Env` in caps, `env_allow = [GOOGLE_APPLICATION_CREDENTIALS, GOOGLE_CLOUD_PROJECT]`, both API hosts in `net_allow`, and `net_timeout_ms ≥ 90000`; (c) a completion whose `ANSWER:` block carries the report's substance, at least one `url_citation` source, and `usageOf` numbers; (d) a `delete` observed in the transcript (Google stores interactions until deleted).
2. **Boundary probe (M4):** a program reading an env var outside `env_allow` gets the typed `NotAllowed`; a program fetching an unlisted host gets `E_NET_DOMAIN_BLOCKED`; a policy naming `Env` without `env_allow` is refused at resolve with a reason naming the field — all asserted in tests, not assumed.
3. **Lane integrity (M1):** `executor.CheckLanePolicy` still refuses a non-restricted policy for the lane (regression test unchanged); the metadata server remains unreachable under every restricted policy (existing containment tests stay green); `--net-timeout` beside `--policy` is refused by name (new test).
4. **No-rebuild (M2/M3):** the whole capability is policy + job + template config plus the M1 core — no image rebuild; a template fix rides the next dispatched task.
5. **Cost visibility (M4):** the probe's answer records `usage` (tokens, searches) and the task's banked row its `cost_usd`; the template forbids open-ended "research everything" prompts in favour of scoped questions.

---

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| **D1 — Env becomes a *confined* restricted-mode admission**: `Env` enters `RestrictedEffects` **only** behind a new policy field `env_allow = [...]` (names), mirroring `net_allow`/`process_allow`: Env without `env_allow` refused, `env_allow` without the Env cap refused, worker passes exactly the named vars, and the effect's existing `NotAllowed` allowlist is pinned to them (V5, V6). Alternatives rejected: `trusted_host` (refused for the lane, V4); an AI-effect-mediated auth builtin (new machinery, duplicates `gcp_auth`) | This is the first widening of what restricted mode *can* admit since M-EXECUTOR-POLICY-HARDENING froze the list; done sloppily it reopens ambient-authority leaks (the mode exists because Env reads were unconfined) | **human (freeze)** | design | high |
| **D2 — Credential delivery**: the job mounts an SA key (Secret Manager) **inside `fs_sandbox`** (`.ailang-scratch/adc-credentials.json`, git-excluded, `fs_deny_write`-protected), the job env sets `GOOGLE_APPLICATION_CREDENTIALS` to it and `GOOGLE_CLOUD_PROJECT=sunholo-daneel`, and the policy admits exactly those two names. Residual accepted: a sandboxed program can *read* the key file and print it (it can read every file in its sandbox); mitigated by scratch-dir commit exclusion, `fs_deny_write` (no tamper), and work_tier-2 job isolation. Alternatives rejected: metadata-server grant (an ambient token tap for every program, V8); refresh-token-in-env (same exposure, worse portability) | Puts a standing credential inside the one directory the agent can read — the honest cost of running ADC without ambient authority; the residual must be consciously accepted, not discovered later | **human (freeze)** | design | med |
| **D3 — Net patience becomes policy-owned**: new field `net_timeout_ms` (int, 0 = today's 30 s default); refused when set without Net/Stream, refused when ≥ `timeout_ms`; `--net-timeout` moves into `refusedWithPolicy` (no caller widening once the policy owns it, V10); the `ailang_run` tool's 130 s kill is derived from the policy's `timeout_ms` (+10 s slack, capped) so the tool bound can never silently undercut the policy | Realizes the brief's "a `--net-timeout` above 30 s on the job": the job has **no argv route** to `ailang run` (the model's tool call is the only route, V9), so the policy is the only place the number can live; the 130 s hard cap is the same "looks identical to a dead one" failure the net timeout itself was added for | **agent** (semantics follow the existing guard pattern; values are D2-adjacent config) | compile | low |
| **D4 — Project and IAM**: the template pins the project string `"sunholo-daneel"` (the package takes the project as an argument); the job's service account is granted `roles/aiplatform.user` on `sunholo-daneel`. IAM is the real boundary: a model passing another project gets a clean authz failure | **Pre-ruled by Mark (attended, 21 Sept, task brief): "Daneel's research runs in Daneel's own project, not the plane's"** | **human — RATIFIED by the brief** | design | low |
| **D5 — Package delivery**: the answer package declares the registry dep and the CLI `lock` op gains a `Path` field (validated in-root; `cmd.Dir` = the package dir) so `ailang_cli {op: "lock", path: ".ailang-scratch/<task>"}` writes the lock **and** populates the registry cache there (lock downloads uncached registry deps, V14). `ailang install` stays out of the tool schema (fetching is lock's job). Alternatives rejected: hand-written locks (registry content hashes, fragile); vendoring the package into the daneel repo (freezes 0.1.0, forks the registry); job-image pre-install (still needs the per-package lock, so the schema change is required anyway) | The lock-op fix repairs a **live lane friction** (root-cwd lock cannot see scratch packages, V12/V15) that blocks every registry dep, not just this one; granting fetch authority was already ratified when the parent doc's D2 put `lock` in `cli_allow` | **agent** (schema change mirrors `check`/`fmt`; noted for review) | compile | low |

### Design Freeze

Before implementation begins, these must be ruled by Mark:

- [ ] **D1** — the confined Env admission (policy `env_allow`; the two-name v1 grant). This is the security-relevant widening; the doc claims it is as tight as `net_allow` — a human should agree.
- [ ] **D2** — credential delivery + the read-exfil residual. The alternative is no Deep Research on the lane until a Go-mediated ADC builtin exists (future work), so the residual is a real trade, not a detail.
- [x] **D4** — pre-ruled by Mark (task brief, 21 Sept): research runs in `sunholo-daneel`; the job's SA gets `roles/aiplatform.user` there.

(D3 and D5 are agent-owned by the table above; they proceed unless a reviewer blocks them.)

---

## Deferred Decisions

The following are intentionally left open for the implementer:

- **The `net_timeout_ms` / `timeout_ms` values** (suggested: 90 000 / 120 000 — above the measured 60 s cold POST, under the probe-able tool bound) — agent may choose within the constraint `net_timeout_ms < timeout_ms`; the probe asserts the cold-start case.
- **Whether `env_allow` values are names-only or support `NAME=literal` pins** — v1 is names-only (values come from the job env, operator-set); a literal-pin form waits for a measured need.
- **`getPid`/`getArgs` under the confined admission** — they leak nothing ambient (own pid, caller-supplied args); agent may keep them unconfined or refuse them with Env, provided the daneel use (getEnv/hasEnv only) works and a test pins the choice.
- **Cache warming** — whether the job entrypoint pre-runs `ailang install` for the two packages (one less per-task download) or every task locks cold; agent may choose after measuring lock latency.
- **A Go-mediated ADC builtin** (`std/gcp` minting tokens through the auth library, metadata server included, no key file) — the *clean* long-term answer to D2's residual; deferred until a second consumer measures it.

---

## Solution Design

### Overview

No new image, no new agent, no language change. The executor gains the package the way the lane was designed to grant everything: the **policy names it** (caps, hosts, env vars, credential path, timeout), the **template teaches it** (the package's own quick-start recipe, pinned versions, the ANSWER convention), and the **job supplies the one thing policy cannot** — the credential. The one AILANG-core change (M1) is two policy fields and the confined Env admission they require, plus the two small lane repairs the feature exposed (lock-with-path, tool-bound-from-policy). Everything else is M-EXECUTOR-POLICY-HARDENING's machinery doing exactly what it was built to do.

```
Daneel host (ask route, daneel#152)                    daneel-executor (ailang_only, restricted)
  research Request exceeds caps                           workspace = sunholo-data/daneel clone
        │                                                        ┌─ job entrypoint (before pi):
  Host.dispatch() ── message ──► coordinator ── Cloud Run Job ──┘    mount secret → ${WORKSPACE}/.ailang-scratch/adc-credentials.json
        │                                                        export GOOGLE_APPLICATION_CREDENTIALS, GOOGLE_CLOUD_PROJECT=sunholo-daneel
        ▼                                                        ┌─ the model, with typed tools only:
  completion ◄── ANSWER: report + sources + usage ◄───────────────┘  write .ailang-scratch/<task>/{ailang.toml, research.ail}
                                                                     ailang_cli {op:"lock", path:".ailang-scratch/<task>"}   ← lock + registry download
                                                                     ailang_check → ailang_run (admitted: {IO,FS,Net,Env} ⊆ caps,
                                                                       hosts ⊆ net_allow, env names ⊆ env_allow, net_timeout 90 s)
                                                                     program: start("sunholo-daneel", startBody(question, nonce))
                                                                       → poll get() across runs → report/sources/usage → delete()
```

### Architecture

**Components:**

1. **M1a — the confined Env admission** (this repo). `policy.Policy` gains `EnvAllow []string \`toml:"env_allow"\``; `policy.Resolve` admits `Env` in restricted mode **iff** `env_allow` is non-empty, with the two cross-refusals the FS/Net/Process guards already use (V3): *"policy admits Env but sets no env_allow — refusing to run Env unconfined"* and *"env_allow is set but Env is not in allowed_caps"*. Confinement is two walls that already exist, composed: the **worker wall** — `workerEnv` passes exactly `workerEnvAllow ∪ env_allow` names (values from the parent job env) so nothing ambient is ever in the snapshot; and the **effect wall** — the resolved `env_allow` pins `effCtx.EnvAllowlist`, so `getEnv` of any other name returns the typed `NotAllowed` and `hasEnv` returns false without revealing existence (V5, V6). No new effect code: the Env effect's allowlist and snapshot semantics are already implemented and tested.

2. **M1b — policy-owned net patience** (this repo). `policy.Policy` gains `NetTimeoutMs int \`toml:"net_timeout_ms"\`` (0 = the 30 s default); resolve refuses it without Net/Stream and refuses `net_timeout_ms >= timeout_ms` (a per-request bound above the whole-run bound is a policy bug — fail loudly, A11). `runPolicyResolved` carries both new fields; the `main_run.go` splice sets `*allowEnvFlag` from `env_allow` (comma-joined, the flag's own format) and `*netTimeoutFlag` from `net_timeout_ms`; `--net-timeout` joins `refusedWithPolicy` ("comes from the policy's net_timeout_ms"). The admission `policy: {...}` line banks `env_allow` and `net_timeout_ms`; `policy-tool`'s summary exposes both so the lane prompt can say *"Env is allowed only for: GOOGLE_APPLICATION_CREDENTIALS, GOOGLE_CLOUD_PROJECT"* beside the existing Net/Process lines.

3. **M1c — the two lane repairs** (this repo). The CLI `lock` schema gains `field: "Path", kind: kindPath, required: true` and the runner sets `cmd.Dir` to the validated in-root directory (mirroring how `check` validates a path, differing only in cwd — `ailang lock` operates on the CWD package, V12/V15). The `ailang_run` tool's `pi.exec` bound becomes `min(summary.timeout_ms + 10_000, 600_000)` (falling back to 130 s when no policy resolves), in **both** copies of `ailang-exec.ts` (`.pi/extensions/` and `cmd/ailang/pi_assets/`, byte-identical today, V17) and their test.

4. **M2 — the fleet config** (`ailang-multivac`, not this repo). `config/policies/daneel-executor.toml` gains:
   ```toml
   allowed_caps  = ["IO", "FS", "Clock", "Net", "AI", "Process", "Env"]   # Env: NEW, confined (D1)
   env_allow     = ["GOOGLE_APPLICATION_CREDENTIALS", "GOOGLE_CLOUD_PROJECT"]
   net_allow     = [ ...existing..., "aiplatform.googleapis.com", "oauth2.googleapis.com" ]
   net_timeout_ms = 90000
   timeout_ms    = 120000          # must exceed net_timeout_ms (M1b refuses otherwise)
   fs_deny_write = [ ...existing..., ".ailang-scratch/adc-credentials.json" ]
   # budgets: Net raised to cover start + several polls per run (e.g. 8)
   ```
   The **job** (Terraform): the service account granted `roles/aiplatform.user` on `sunholo-daneel` (D4, ruled); the SA key in Secret Manager, mounted read-only to the workspace at task start by the entrypoint (which knows `AILANG_WORKSPACE`); job env `GOOGLE_APPLICATION_CREDENTIALS` + `GOOGLE_CLOUD_PROJECT=sunholo-daneel`. The registry entry changes **nothing** (`acknowledge_only`, lane, policy path all stand).

5. **M3 — the template** (`templates/daneel-executor-task.md`, Daneel repo — rides the checkout). New section "Deep research requests": pin `'sunholo/gemini_agents' = "0.1.0"` in the scratch manifest, `lock` **with the package path**, the package's quick-start recipe (nonce discipline: the nonce is the request's identity; there is no client idempotency key), poll-across-runs (each `ailang_run` is a separate bounded run — do not loop inside one program), `ANSWER:` must carry the report's substance + at least the top sources + `usage` numbers, **`delete` after harvest** (Google stores interactions until deleted), `transient`-one-retry rule (one run in six is a transient `api_error`), and the project string **`"sunholo-daneel"`** fixed (a request naming another project is a bug, not a choice — IAM answers it).

6. **M4 — the probe.** One dispatched research Request end-to-end (Success Metrics 1, 4, 5) plus the negative probes (Metric 2) plus the M1 regression suite (Metric 3).

### Policy compatibility surface

(Not a parser/typechecker change — no Conflict Surface section required. This is the policy-shape equivalent, per the same habit:)

- **Policies without the new fields resolve identically** — `env_allow` nil ⇒ Env stays unadmitted exactly as today; `net_timeout_ms` 0 ⇒ the 30 s default. Existing lane policies (`pkg-ailang-only.toml`, `ailang-only-executor.toml`) and every non-policy run are untouched. The digest changes only for policies that adopt the fields (expected: the digest is over TOML bytes).
- **`--allow-env`/`--allow-env-file` stay caller-refused under `--policy`** (they already are, V6) — the worker sets the flag value from the resolved policy, the same splice `--net-allow-domains` uses; nothing new is caller-reachable.
- **Adding `--net-timeout` to `refusedWithPolicy`** breaks no live caller: the lane's tool passes no such flag (V9), and the grep proving it goes in the test comment, per the conflict-surface habit.
- **`CheckLanePolicy` and the containment tests are the regression fixtures**: restricted-mode admission lists, worker-env scrubbing, metadata-server blocking, redirect containment (`run_policy_containment_test.go`) and confined-git (`TestRunPolicyE2E_ConfinedGitInRestrictedMode`) must all stay green, byte-for-byte in intent.

### Implementation Plan

**M1: Core — confined Env + policy-owned net timeout + lane repairs** (~1.5 days, this repo)
- [ ] `internal/policy/policy.go`: `EnvAllow`, `NetTimeoutMs` fields (docs in the struct comment, mirroring `NetAllow`'s)
- [ ] `internal/policy/resolve.go`: `Env` in `RestrictedEffects` behind the `env_allow` guards; `net_timeout_ms` guards (no Net/Stream ⇒ refuse; ≥ `timeout_ms` ⇒ refuse); `Resolved` carries both
- [ ] `cmd/ailang/run_policy.go`: `runPolicyResolved.envAllow`/`netTimeout`; `--net-timeout` in `refusedWithPolicy`; admission line banks both
- [ ] `cmd/ailang/run_policy_supervise.go`: `workerEnv` passes `env_allow` names (values from the parent env; absent names silently omitted — a missing var is not an error)
- [ ] `cmd/ailang/main_run.go`: the splice sets `*allowEnvFlag`, `*netTimeoutFlag`
- [ ] `internal/policytool/cli_ops.go`: `lock` schema + per-op cwd; `policytool` summary + lane-prompt lines (`env_allow`, `net_timeout_ms`)
- [ ] `.pi/extensions/ailang-exec.ts` + `cmd/ailang/pi_assets/ailang-exec.ts`: exec bound from `summary.timeout_ms`; `.ailang-exec.test.ts` cases
- [ ] Tests (all named-reason assertions): the six refusals; e2e restricted run with `env_allow` (admitted var readable, outside var `NotAllowed`, `hasEnv` false); metadata still blocked; `--net-timeout`+`--policy` refused; `lock` with path writes the lock in the target dir, out-of-root path refused; `CheckLanePolicy` regression (restricted-with-Env+env_allow passes, trusted_host still refused)

**M2: Fleet config** (~1 day, `ailang-multivac`)
- [ ] Diff the deployed `daneel-executor.toml` against the parent doc's shape first (V20 — this checkout does not carry it); then apply the M2 block above
- [ ] Terraform: role grant on `sunholo-daneel`; secret + entrypoint mount; job env vars
- [ ] Re-verify the package's host list **from its source** (registry → `ailang-packages` repo) the way the parent doc enumerated the host's domains — the package's own docs are the claim, a probe is the proof (any miss surfaces as a named `E_NET_DOMAIN_BLOCKED`)

**M3: Template** (~0.5 day, Daneel repo)
- [ ] The "Deep research requests" section (component 5); re-verify the Request-record premises against `ext/abi/types.ail` at HEAD (parent doc's V14 discipline)
- [ ] `_smoke.ail`-style dry recipe in the template's examples block (start/get/delete shapes, pinned versions)

**M4: Probe + docs** (~1 day)
- [ ] Live round trip (Success Metric 1), asserted on the NDJSON (zero `bash`), the admission JSON, and the inbox completion payload
- [ ] Negative probes (Success Metric 2) — the refusals ARE the boundary working
- [ ] Docs: `agent-tool-policy.md` (the two fields, the Env row, the example policy), `docs/LIMITATIONS.md` residual-trust matrix row for the credential file, CHANGELOG entry; this doc moves to `implemented/` with the enumeration record

### Files to Modify/Create

**Modified (this repo):**
- `internal/policy/policy.go` (~+8) — two fields
- `internal/policy/resolve.go` (~+30) — admission guards, `Resolved` fields
- `cmd/ailang/run_policy.go` (~+15) — resolved fields, one refusal, admission line
- `cmd/ailang/run_policy_supervise.go` (~+8) — `workerEnv`
- `cmd/ailang/main_run.go` (~+6) — the splice
- `internal/policytool/cli_ops.go` (~+12) — lock schema, per-op cwd
- `internal/policytool/` summary host (~+8) — two fields + prompt lines
- `.pi/extensions/ailang-exec.ts` and `cmd/ailang/pi_assets/ailang-exec.ts` (~+6 each, kept byte-identical) + `.ailang-exec.test.ts` (~+40)
- Tests: `internal/policy/resolve_test.go`, `cmd/ailang/run_policy_hardening_test.go`, new `cmd/ailang/run_policy_env_test.go` (~+250 across)
- `docs/docs/guides/agent-tool-policy.md` (~+30), `docs/LIMITATIONS.md` (~+6), `changelogs/v0.32-current.md` (~+12)

**Not in this repo:**
- `ailang-multivac`: `config/policies/daneel-executor.toml` (~+8), Terraform job (role grant, secret, env) (~+20)
- `sunholo-data/daneel`: `templates/daneel-executor-task.md` (~+60)

---

## Examples

### Example 1: the whole round trip

A capability is asked: *"Research what changed in EU AI Act enforcement this quarter and cite sources."* Web research exceeds every cap the host lends.

```
Host.dispatch(inbox: "daneel-executor", payload: Request{ question, requester, ... })

daneel-executor task:
  write .ailang-scratch/research-q1/ailang.toml
      [package] name = "research-q1"
      [dependencies] 'sunholo/gemini_agents' = "0.1.0"          # registry, pinned
  ailang_cli {op: "lock", path: ".ailang-scratch/research-q1"}   # NEW (D5): lock + registry cache
  write .ailang-scratch/research-q1/research.ail   # imports pkg/sunholo/gemini_agents/{request,interaction,answer,agents}
                                                   # ! {IO, FS, Net, Env}
  ailang_run: admitted —
    policy: {"ok":true, "caps":["IO","FS","Clock","Net","AI","Process","Env"],
             "env_allow":["GOOGLE_APPLICATION_CREDENTIALS","GOOGLE_CLOUD_PROJECT"],
             "net_allow":[...,"aiplatform.googleapis.com","oauth2.googleapis.com"],
             "net_timeout_ms":90000, ...}
  program run 1: start("sunholo-daneel", startBody(question, nonce)) → Ok(id)
  program run 2..n (one per ailang_run): get("sunholo-daneel", id) → "running" → come back next run
  program run k: statusOf → "done" → rewriteCitations(reportOf(rec)),
                 sourcesBlock(sourcesOf(rec)), usageOf(rec) → delete("sunholo-daneel", id)
  final message: "ANSWER: <report's substance> [1][2]…\nSources: [1] … [2] …\nUsage: 41k tokens, 3 searches."

completion → sender inbox; payload.summary ends with the ANSWER: block
Daneel host parses it, mails the requester via lent mail()
```

### Example 2: the refusals ARE the boundary working

- A program tries `getEnv("OPENAI_API_KEY")` → the typed `NotAllowed` ("environment variable … not in allowlist"). The key never existed in the worker snapshot either — the wall is double.
- A program tries a registry the policy doesn't name → `E_NET_DOMAIN_BLOCKED: domain not in allowlist`.
- A policy author forgets `env_allow` but lists `Env` in caps → resolve refuses: *"policy admits Env but sets no env_allow — refusing to run Env unconfined"*. The lane cannot drift into ambient reads by omission.
- The model names project `"some-other-project"` → Vertex returns a clean authz failure: the SA's `roles/aiplatform.user` exists only on `sunholo-daneel` (D4's IAM-is-the-boundary).

---

## Success Criteria

- [ ] Success Metrics 1–5 (Goals), each with its measurement named (NDJSON, admission JSON, inbox payload, test names)
- [ ] M1's named-reason test suite green; `make test-core` green; `make check-boundaries` green; `make simplicity-audit` no regression (no new env-var routes — the policy rides `AILANG_AGENT_POLICY_TOML`)
- [ ] `executor.CheckLanePolicy` still refuses `trusted_host` for the lane — the D3 invariant of M-EXECUTOR-POLICY-HARDENING is unchanged by the Env admission
- [ ] The metadata server remains blocked under restricted policies (existing containment tests untouched and green)
- [ ] The two `ailang-exec.ts` copies byte-identical (drift test green)
- [ ] A clean research answer banks `status: completed` (acknowledge_only stands — no registry change)
- [ ] Docs updated (policy guide, LIMITATIONS residual row, changelog); the M2 enumeration record lands in this doc's implementation report

## Testing Strategy

**Unit (this repo, M1):** every refusal with its exact reason string (the agent reads the reason); the resolve guards; `workerEnv` composition (admitted names present, absent names omitted, provider-scrub unchanged); the lock schema (in-root path accepted, traversal refused, lock written in the target dir).

**Integration:** `ailang run --policy` end-to-end on the checkout binary: a restricted policy with `env_allow` runs an Env-cap program; `--net-timeout` beside `--policy` refused; `net_timeout_ms ≥ timeout_ms` refused; a scratch package with a registry dep locking through the policy-tool route (registry fetch stubbed or hit once, then cached).

**Regression-surface:** `run_policy_containment_test.go`, the confined-git E2E, the worker-env provider-scope test, and `CheckLanePolicy`'s test all stay green — these are the fixtures proving the admission widened *by one confined door*, not by a mode change.

**Manual / live (M2/M4):** the dispatched round trip; the negative probes; Cloud Logging by execution name for the transcript (m-chains-executor-transcripts still pending); the package's host list re-verified from its source, recorded in the implementation report.

## Non-Goals

- **Widening Daneel's host.** The host keeps its own direct ask-route usage (daneel#152); this doc adds a lane for *dispatched* questions, it does not move the host's.
- **A general ADC/credentials story for the lane.** This is one package, one policy, two env names. A `std/gcp` token builtin, env literal-pins, or a Secret-effect admission are separate measured-need decisions (see Deferred).
- **Developer-API parity** (`generativelanguage.googleapis.com`, API-key, `deep-research-max`, custom agent configs — the package names these as not-on-Vertex). Not grantable, not taught, not in `net_allow`.
- **Enforcing "research-only" usage of the hosts.** The policy admits the two hosts for any Net call an admitted program makes — same as every `net_allow` entry; IAM and budgets bound the blast radius.
- **Fixing `gcp_auth`'s metadata-server arm** (unreachable under restricted mode, V8). It stays the fallback this design deliberately does not use.
- **M-STD-WEB-SECOND-BACKEND.** Related (std/web's Ollama-only search), distinct (that is a std module backend; this is a registry package on one lane). No shared machinery is changed here.

## Timeline

| Day | Work |
|---|---|
| 1 | M1 policy fields + resolve guards + supervisor/splice wiring + first test batch · freeze items D1/D2 to Mark |
| 2 | M1 remainder (policytool summary/prompt, lock schema, extension bound, both copies + tests) · quorum optional here |
| 3 | M2 fleet config + job (role, secret, env) + host-list re-verification · M3 template |
| 4 | M4 probe + negative probes + docs + changelog · implementation report |

Realistic: 4 days assumes the M1 guards hit no surprise in `workerEnv` composition. If the probe finds the package's documented host list incomplete (a regional endpoint, say), add a day — the miss is a named `E_NET_DOMAIN_BLOCKED`, the fix is one `net_allow` line.

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| The confined Env admission is done as a bare `RestrictedEffects["Env"] = true` without the `env_allow` guard (review-slip, not design-slip) | High — reopens ambient reads on every restricted lane | The resolve refusal is the first test written; `CheckLanePolicy` regression named in Success Criteria; the freeze item asks Mark to check exactly this |
| Key-file read exfil (D2 residual): a sandboxed program prints the credential | Med — a standing SA key in a transcript | Accepted at freeze with mitigations (scratch exclusion, deny-write, job isolation, work_tier 2); the `std/gcp` builtin is the tracked escape |
| The package's host list is incomplete or moves (regional endpoints, a 0.2.0 re-pin) | Med — every research run refused at the gate, loudly | Probe asserts a live `start`; a miss is a named `E_NET_DOMAIN_BLOCKED`; the template pins `0.1.0` so a registry bump cannot drift the answer package |
| Deep Research cost runaway ($1–3/run, ~20 min) | Med | Scoped-question rule in the template; `AILANG_MAX_COST_USD` on the task; `usageOf` mandatory in the ANSWER block; the host can rate-limit per requester once measured |
| Cold-start POST exceeds even 90 s | Low | `transient`-one-retry rule in the template; `net_timeout_ms` is one policy line to raise; the probe measures the real cold latency |
| `lock` op cwd change breaks an existing user of the root-cwd shape | Low | The lane's only lock users are scratch packages (root lock fails today, V15 — the change *fixes* the documented recipe); `cli_ops_test.go` asserts both shapes |
| Delete-after-harvest forgotten → Google-side storage growth + repeated re-reads | Low | Template states it as the final step; the probe's transcript assertion includes the `delete` |

## Axiom-adjacent note — why this is NOT the trusted_host slippery slope

The one-sentence objection this doc must survive: *"you're putting a host integration (Env + a credential) behind a lane that claims no host integrations."* The answer is the difference between **ambient** and **named**: `trusted_host` grants the worker environment wholesale; this design admits exactly two operator-named variables through a policy-digested field, walls the effect behind the same kind of closed-set check as `net_allow`, and keeps every existing containment fixture green. `CheckLanePolicy`'s own docstring names Env as the host-integration counterexample — M1 turns Env from a host integration into a confined one, which is precisely the ladder Process climbed in M-EXECUTOR-POLICY-HARDENING M6 (unadmitted → confined schema). If that framing is wrong, D1's freeze is where it should be said.

## Related Documents

**Implemented (parents and constraints):**
- [m-daneel-ailang-executor.md](../../implemented/v0_39_3/m-daneel-ailang-executor.md) — the lane agent, policy shape, template, dispatch rung; its Deferred item "whether the executor gets Env" is picked up here **with a corrected premise** (not a one-line policy change)
- [m-executor-policy-hardening.md](../../implemented/v0_41_0/m-executor-policy-hardening.md) — restricted mode, `RestrictedEffects`, worker env scrubbing, `CheckLanePolicy` D3; this doc extends its M6 ladder (unadmitted → confined) to Env
- [m-agent-ailang-only-execution.md](../../implemented/v0_39_0/m-agent-ailang-only-execution.md) — the lane, the gate, the registry fields

**Planned (check for overlap):**
- [m-std-web-second-backend.md](../v0_39_2/m-std-web-second-backend.md) — related open thread (std/web search depends on Ollama alone); this doc gives *this lane* a research route independent of Ollama but changes no std/web machinery — noted distinct in Non-Goals
- [m-chains-executor-transcripts.md](../v0_39_1/m-chains-executor-transcripts.md) — transcripts for the lane; until it ships, Cloud Logging by execution name

**External:**
- `sunholo/gemini_agents@0.1.0` + `sunholo/gcp_auth@0.8.1` (registry; source: `sunholo-data/ailang-packages` packages/{gemini-agents,gcp-auth})
- `sunholo-data/daneel` — the executor's workspace, the template, the host's ask route (daneel#152)
- Issue #1267 (this task's GitHub thread); Daneel message `inbox_1789993952527_28db791a`

## References

- [Agent tool policy guide](../../../docs/docs/guides/agent-tool-policy.md) — the policy file, delivery, banked fields, what the model is told
- `internal/policy/policy.go` / `resolve.go` — the Policy struct, `RestrictedEffects`, the fine-grained guard pattern this doc mirrors
- `cmd/ailang/run_policy.go` / `run_policy_supervise.go` — the widening-flag refusal list, the splice, `workerEnv`
- `internal/effects/env.go` — the existing Env allowlist + snapshot semantics (the confinement this design composes)
- `internal/policytool/cli_ops.go` — the CLI op schemas (the `lock` change)
- `internal/pkg/loader.go` / `resolver.go` — lock-vs-cache resolution for registry deps
- `internal/ai/factory/factory.go:167-178` — the Vertex ADC lane the executor's own Gemini calls already use (the auth identity this feature shares)

## Future Work

- A `std/gcp` ADC token builtin (Go-mediated, metadata server included) — removes D2's key-file residual for every lane consumer; waits for a second consumer
- Env literal-pins (`NAME=literal`) in policies, if a measured need distinguishes naming from pinning
- `ailang install` behind a confined CLI schema (cache warming without lock), if per-task lock latency measures badly
- Extracting a "registry package as lane tool" checklist once a second package rides the lane
- m-chains-executor-transcripts, so the probe's transcript assertions move from Cloud Logging to the chain

---

## Verification Log

Two classes of row, kept honestly separate: **VERIFIED HERE** (grep/read/run against this checkout or the live registry, 2026-09-23) and **INHERITED** (the task brief, daneel#152, or the registry's published metadata — the sprint re-verifies the load-bearing ones at M2/M4 start and records the result in the implementation report).

| # | Claim | Method | Result |
|---|-------|--------|--------|
| V1 | The package's `interaction` module is `{FS, Net, Env}`; the package's ceiling is `Net, FS, Env, IO`; its own run line demands `--caps Net,FS,Env` and `--net-allow-domains aiplatform.googleapis.com,oauth2.googleapis.com` with a net timeout above 30 s; it depends on `sunholo/gcp_auth` | `ailang pkg info sunholo/gemini_agents` (Effects: Net, FS, Env, IO; Dependencies: sunholo/gcp_auth) + `ailang pkg docs sunholo/gemini_agents` (module table, quick start, the two domains, the 60 s cold POST note) — live registry, this session | **VERIFIED HERE (registry)** — the host list and caps are the package's *published claim*; M2 re-verifies from source, M4's probe is the proof |
| V2 | Restricted mode cannot admit `Env` today: `RestrictedEffects` = {IO, FS, Net, Clock, Rand, Stream, Process, AI} and nothing else; a policy naming Env is refused at resolve | read `internal/policy/resolve.go:41-46` (the map) and `:182-184` (the refusal text "no confined adapter for") | **VERIFIED HERE** — the load-bearing negative-existence premise; there is no line to add today |
| V3 | The guard pattern to mirror exists: FS needs `fs_sandbox`, Net needs `net_allow`, Process needs `process_allow`, cross-set refusals both directions; AI additionally requires a pinned provider + budget | read `internal/policy/resolve.go:192-211` + the AI admission block; changelog `v0.32-current.md` M7 entry ("Restricted mode admits `AI` when `ai_provider` is pinned **and** `[budgets] AI` states the ceiling") | **VERIFIED HERE** |
| V4 | The lane requires restricted mode; `trusted_host` is refused for it, and the refusal's own docstring names Env as the host-integration case | read `internal/executor/agent_policy.go:61-85` (`CheckLanePolicy`: "requires a restricted policy… would be claimed over a host-integration grant"; comment: "(Process, AI, Env…)") | **VERIFIED HERE** — going `trusted_host` is not an escape hatch; the confined admission is the only route |
| V5 | The Env effect already has allowlist + snapshot semantics: `getEnv` outside the allowlist → typed `NotAllowed`; `hasEnv` → false without revealing existence; snapshot pinned at start; no enumeration | read `internal/effects/env.go` in full (`envGetEnv`, `envHasEnv`, `isInAllowlist`, the ADT `EnvError = NotFound \| NotAllowed`) | **VERIFIED HERE** — the confinement mechanism is composition, not new effect code |
| V6 | `--allow-env`/`--allow-env-file`/`--env` are caller-refused under `--policy` ("environment access is not a policy field") | read `cmd/ailang/run_policy.go:41-77` (`refusedWithPolicy` entries) | **VERIFIED HERE** |
| V7 | The worker's process env is scrubbed: `workerEnvAllow` (PATH/HOME/TZ/…) plus the pinned AI provider's credential vars only; `GOOGLE_APPLICATION_CREDENTIALS` reaches the worker only when AI is admitted with a Google provider | read `cmd/ailang/run_policy_supervise.go:43-49` (`workerEnvAllow`), `:195-227` (`workerEnv`, `providerCredentialVars` — the Google case at :225) | **VERIFIED HERE** — the snapshot the Env effect reads is already policy-shaped; M1 adds the `env_allow` names to exactly this composition |
| V8 | The metadata server is unreachable through the Net effect under a policy: link-local 169.254.169.254 refused without `allowMetadata`, and `--net-allow-metadata` is itself refused under `--policy` ("restricted mode has no metadata grant"); **no policy field grants it** | read `internal/effects/net_authorize.go:216-219` (`validateIP`) + `cmd/ailang/run_policy.go:68` (the refusal) + `internal/policy/policy.go` (no metadata field in the struct) | **VERIFIED HERE** — negative-existence: ADC on the lane must ride the file path, which is what the two allowed hosts are for |
| V9 | `--net-timeout` defaults to 30 s; the lane's `ailang_run` passes no such flag and caps the whole invocation at 130 000 ms | read `cmd/ailang/help.go:262`, `cmd/ailang/main_run.go:78`, `internal/runner/handlers.go:166-176`, and `ailang-exec.ts` `register()`'s `ailang_run` (args = `["run","--policy",path,…"--args-json"…,file]`; `pi.exec(..., {timeout: 130_000})`) | **VERIFIED HERE** — the job has no argv route to `ailang run`; "on the job" can only mean "in the policy" (D3) |
| V10 | `--net-timeout` is NOT in `refusedWithPolicy` — caller-settable beside `--policy` today (a widening wart M1 removes) | read the full `refusedWithPolicy` list (`cmd/ailang/run_policy.go:41-77`): net-allow-* yes, net-timeout absent | **VERIFIED HERE** (negative-existence, by full-list read) |
| V11 | The admission `policy: {...}` stderr line exists and is banked; the policy-tool summary feeds the lane prompt (caps, sandbox, net_allow, process_allow, cli, timeout) | read `cmd/ailang/run_policy.go` `admissionLine` + `internal/policytool` summary + `ailang-exec.ts` `lanePrompt()` | **VERIFIED HERE** — both carry the two new fields in M1 |
| V12 | The CLI `lock` op takes no path and runs at the sandbox root (`needsRoot: true`, no `field`); `ailang lock` locks the CWD package and fails without an `ailang.toml` there | read `internal/policytool/cli_ops.go:82` (schema), `:264-270` (dir = root), `cmd/ailang/pkg_commands.go:460-505` (`pkgLockCommand`: `os.Getwd()`, "no ailang.toml found in %s") | **VERIFIED HERE** — the root-cwd lock cannot see scratch packages: the live friction D5 fixes |
| V13 | Registry deps must be locked AND cached before a run; the cache miss error names `ailang install`; `ailang install` exists as a CLI command but is not in the tool's schema (unreachable through the lane) | read `internal/pkg/loader.go:178-181` (the error), `cmd/ailang/commands_pkg.go:44` (the command), `internal/policytool/cli_ops.go:61-83` (`cliSchemas` — no install) | **VERIFIED HERE** (both halves) |
| V14 | `ailang lock` populates the registry cache as a side effect of resolution (checks cache, downloads + extracts the tarball when absent) — so a lock-with-path is also the fetch | read `internal/pkg/resolver.go:246-282` (the registry arm: `CachedPackagePath`, stat check, download, `ExtractTarball`) | **VERIFIED HERE** |
| V15 | The ratified answer-package shape (`.ailang-scratch/<pkg>/` with manifest + lock, excluded from wrapper commits) is the lane's documented recipe — and its lock step cannot work through the tool today | parent doc (implemented) component 1 + `cmd/ailang/coordinator_cloud_scratch.go` (ScratchDir exclusion, cited there as V6c) + V12 above | **Code VERIFIED HERE** (V12); the *recipe gap* is the design finding |
| V16 | Interactions-API behaviour: `background:true`/`store:true` mandatory; listed immediately; nonce is the only request identity; delete-after-harvest required (Google stores until deleted); 1-in-6 transient `api_error`; $1–3/run cost estimate, 22k–60k tokens measured; gcp_auth = "ADC token (file refresh exchange, or the metadata server)" | `ailang pkg docs sunholo/gemini_agents` (live registry, "What the API does (measured 21 Sept 2026)" + Cost + Dependencies) + `ailang pkg docs sunholo/gcp_auth` | **VERIFIED HERE (registry docs, package-authored measurements)** — the sprint probe re-verifies delete/transient/usage against a live record |
| V17 | The two `ailang-exec.ts` copies (`.pi/extensions/` and `cmd/ailang/pi_assets/`) are byte-identical and embedded via profile assets | `diff -q` → identical; `internal/executor/pi/profile_assets.go:32-35` (the asset map) | **VERIFIED HERE** — both must change together (drift test enforces) |
| V18 | The executor's own Gemini calls already reach Vertex under ADC via the metadata server (the Go auth library inside the worker, not the Net effect) — the auth *identity* this feature shares; only the project and hosts change | read `internal/ai/factory/factory.go:167-193` (the ProviderGoogle ADC lane, `gemini.NewVertexAIClient`, the ADC error path) + `run_policy_supervise.go`'s "ADC on Cloud Run needs none" | **VERIFIED HERE** — the task brief's "auth path is the same one" is true of the identity; the *mechanism* differs (Go library vs the package's file-exchange over Net), which is exactly why the two hosts and the key file are needed |
| V19 | Mark's ruling: research runs in `sunholo-daneel`; the job's service account gets `roles/aiplatform.user` there | task brief (attended ruling, 21 Sept) | **INHERITED — treated as ratified** (D4 checked); M2 applies it and the probe's IAM failure mode is the assertion |
| V20 | The deployed `daneel-executor.toml`'s current shape (caps, budgets, net_allow, cli_allow, security_mode after the v0.41.0 migration) | fleet config in `ailang-multivac` — **not present in this checkout**; changelog `v0.32-current.md:565` records it admits Process (confined), AI, Net in restricted mode | **INHERITED — M2 diffs the deployed file before editing** (the parent doc's M2 discipline) |
| V21 | Daneel's host uses the package today as the ask route's backend, as the `daneel` account in `sunholo-daneel`; the record reads back report + `url_citation` sources + usage | task brief citing daneel#152 (the daneel repo is not present on this authoring workspace) | **INHERITED — re-verified at M4** by reading one live interaction record through the executor |

---

## Quorum

**Triggers fired: 2 of 4** — (1) **design-freeze items D1/D2** (a restricted-mode admission widening and a standing credential inside the sandbox — both security-relevant, both human-ruled) and (4) **external systems** (fleet config, IAM, the Daneel repo, the registry package — three of the four artefacts live outside this checkout). Triggers 2 (shared machinery) and 3 (banked schema) do not fire: the M1 changes extend the existing guard/splice/schema patterns in place, and the admission line grows two fields without changing shape.

**Recommended but not run at authoring time** (the skill makes quorum optional; this doc's D1 is exactly the kind of widening a reject-by-default reviewer should see before an Opus sprint spends a day on it):

```bash
ailang design-quorum design_docs/planned/v0_42_1/m-daneel-deep-research.md \
  --reviewers gpt5-6-sol,gemini-3-1-pro \
  --controller-verdict pass --controller-note "premises V1-V18 verified in-log; D1 is a confined admission on the M6 ladder, not a mode change; D2's residual is stated, not hidden" \
  --mission-log design_docs/v1-mission-log.md
```

**D1 and D2 must be ruled by Mark before the sprint starts.** The quorum can precede, accompany or follow the freeze; the sprint plan may not start execution on an unchecked freeze item.

---

**Document created**: 2026-09-23
**Last updated**: 2026-09-23
