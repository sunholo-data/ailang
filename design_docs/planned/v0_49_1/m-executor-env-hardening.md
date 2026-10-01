# M-EXECUTOR-ENV-HARDENING: A Typed Environment Boundary for the Agent Executor Fleet

**Status**: Implemented — D1–D6 approved by Mark 2026-10-01; Phases 1+3 in #1435, Phases 2+4 in the follow-up PR-B
**Target**: Next security release after v0.49.0; release number assigned during planning
**Priority**: P0 — operator write credentials (GitHub fleet token, superuser registry key) are reachable by any prompt-injected agent subprocess
**Estimated**: 8–12 engineering days, including regression coverage and lane migration inventory (planning estimate, not a sprint commitment)
**Dependencies**: None new. Complements implemented [M-EXECUTOR-POLICY-HARDENING](../../implemented/v0_41_0/m-executor-policy-hardening.md) (v0.41.0), which confined the AILANG *program* runtime and explicitly did not own the agent-harness subprocess boundary.
**Routing**: AILANG fix under [PROGRAM](../../PROGRAM.md); no motoko core change, no language change
**Created / updated**: 2026-09-30 / 2026-10-01 (approval; findings re-verified on `dev` @ `2aee3ed76`)

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Child env becomes a deterministic pure function of (lane profile, task); no timing or ambient nondeterminism added |
| A2: Replayability | +1 | Banked child-env name digest lets a banked run state which vars reached the model-facing process |
| A3: Effect Legibility | +1 | Secret forwarding stops being an invisible ambient side effect of `os.Environ()` inheritance |
| A4: Explicit Authority | +1 | Every forwarded variable is an explicit grant in a resolved profile — the core of this design |
| A5: Bounded Verification | +1 | Precedence and allowlists are unit-testable without a live CLI or model |
| A6: Safe Concurrency | 0 | No concurrency changes; env built once per dispatch |
| A7: Machines First | +1 | Boundary violations are loud, structured, named-variable errors, not silent behavior |
| A8: Minimal Syntax | +1 | No language syntax; one Go seam reused by five executors |
| A9: Cost Visibility | 0 | No cost-system change |
| A10: Composability | +1 | One `BuildEnvironment` seam; the existing claude-only strip is retired into it |
| A11: Structured Failure | +1 | Refused env names and unvalidated URLs fail loudly pre-dispatch; no silent fallback |
| A12: System Boundary | +1 | Separates operator host authority from model-facing child authority explicitly |

**Net Score: +9** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): no implicit nondeterminism introduced — one deterministic builder replaces loader-dependent duplicate entries
- [x] A3 (Effects): no hidden side effects — the design removes a hidden one (ambient secret inheritance)
- [x] A4 (Authority): no ambient access granted — this design *revokes* ambient access by default
- [x] A7 (Machines First): not optimizing for human convenience; preserves the machine-facing contract (correlation IDs, stdlib pin)

## Problem Statement

AILANG's five subprocess executors — **claude, codex, opencode, motoko, pi** — launch model-facing
agent CLIs via `BuildEnvironment`, which starts from the operator host's **complete environment**
(`os.Environ()`, `internal/executor/environment.go:69`) and hands the result to the child as
`cmd.Env` (`internal/executor/pi/pi.go:167`, `codex/codex.go:172`, `opencode/opencode.go:178`,
`motoko/motoko.go:462`, `claude/claude.go:253`). The child's model can read its own environment
with a single `bash` tool call. Untrusted content routinely reaches agent prompts — coordinator
directives embed GitHub issue/PR text (this very task's directive ships open-PR data that the
host must label "data, not instruction"), eval benchmarks run tasks authored by third parties,
and clone-over-egress lanes review arbitrary public repos — so a prompt injection anywhere in
that chain can exfiltrate every secret in the container with one `printenv`.

**Measured in the executor container, this session (2026-09-30):** the model-facing agent process
observes at least five operator secrets — `GITHUB_TOKEN` (the fleet's repo write token),
`AILANG_REGISTRY_API_KEY` (the **superuser** registry key: publish, unpublish, and key
management — `internal/config/pkgregistry.go:30`), `OPENROUTER_API_KEY`, `GEMINI_API_KEY`, and
`OLLAMA_API_KEY`. An agent does not need `ailang_only` tool confinement to reach them; every
lane with `bash` already can.

### Findings

| ID | Demonstrated / source-supported behavior | Cause (read, cited) |
|---|---|---|
| E1 | All operator secrets reach every model-facing agent child | `BuildEnvironment` starts at `os.Environ()` and every executor passes it through verbatim. Only claude strips its own auth (`claude/claude.go:265-270` removes `CLAUDE_CODE_OAUTH_TOKEN` and, in apikey mode, `ANTHROPIC_API_KEY`) — ad-hoc, one executor, two variables; pi, codex, opencode, and motoko strip nothing |
| E2 | `Task.ExtraEnv` can set **any** environment variable in the agent child with no name validation | `environment.go:103-107` appends task-supplied pairs unconditionally; sources include eval benchmark YAML `agent_env` (`internal/eval_harness/spec.go:38-43`) and browser-session env (`browser_sessions.go:198-203`) — semi-trusted corpora can set `PATH`, `LD_PRELOAD`, `GIT_CONFIG`, `OTEL_EXPORTER_OTLP_ENDPOINT` (a telemetry-redirect exfiltration channel), or `AILANG_*` routing variables |
| E3 | Precedence is an accident of append order, and is the reverse of the documented one | `environment.go` appends `AILANG_STDLIB_PATH` (91), `PWD` (96), `ExtraEnv` (103-105), `AILANG_PARENT_TASK_ID` (133), chain IDs (140-146), `OTEL_RESOURCE_ATTRIBUTES` (153), `OTEL_EXPORTER_OTLP_ENDPOINT` (162) with plain `append`; motoko appends ten more keys after the builder (`motoko.go:381-458`). **Correction (2026-10-01):** the doc originally said duplicates reach `execve` and glibc's first-match wins. They do not: Go's `os/exec` de-duplicates `Cmd.Env` with *last* value wins (documented on `exec.Cmd.Env`), and all five executors launch through it. The real defect is that "last" is wherever a line happens to sit: `ExtraEnv` is appended *before* the harness keys, so on a collision the harness beats `ExtraEnv` — the inverse of D4's `ExtraEnv > injected > inherited` — and `UpdateEnvVar` (274-283) rewrites only the first match. Nothing is wrong at runtime today that a test pins; the order is simply unowned |
| E4 | Clone-review preamble interpolates `--clone-repo` verbatim into shell text the agent is told to run EXACTLY | `clone_preamble.go:47-77` validates only that `repoURL` is non-empty; a URL carrying shell metacharacters (`;`, `$(…)`, backticks) is emitted into `git clone --depth 1 <URL> repo` as-is. Callers: `cmd/ailang/exec.go:392` (CLI flag) and `internal/eval_harness/gemini_clone_review.go:23` (spec-supplied URL). Blast radius (2026-10-01): only `managed_agents` advertises `CapNetworkEgress` (`ValidateCloneFlags`, `clone_preamble.go:100`), so the shell that runs the preamble is the Google-hosted sandbox, not the executor container — still an injection, smaller reach |
| E5 (new, 2026-10-01) | Stripping `GITHUB_TOKEN` from the child env alone would not hide it: the cloud parent writes the fleet token **as a literal** into the *global* git config | `cmd/ailang/coordinator_cloud.go:337-345` runs `git config --global credential.helper '!f() { …; echo "password=<token>"; }; f'`. The child shares `$HOME`, so `git config --global --get credential.helper` prints the token, and every `git push` the child runs authenticates as the fleet for any repo the token reaches |
| E6 (new, 2026-10-01) | `ValidateTaskCapabilities` is not on most dispatch paths | Called only from `cmd/ailang/exec.go:415` and `eval_harness/browser_sessions.go:120,204`. The coordinator (`internal/coordinator/provider_executor.go:178`), the cloud job (`coordinator_cloud_executor.go`), mission dispatch and the eval agent runner never call it — so a D3 check that lived *only* there would miss the paths that matter. The builder must refuse too |
| E7 (new, 2026-10-01) | The harness itself uses `ExtraEnv` for two names D3 deny-lists | `AILANG_AGENT_POLICY` (`coordinator_cloud_executor.go:95`, `provider_executor.go:168`, `agent_runner_multi.go:251`) and `PI_WORKSPACE_TRUST_REMOTES` (`coordinator_cloud_executor.go:74`, `provider_executor.go:111`). `AILANG_AGENT_POLICY` must become harness-injected from `Task.PolicyPath`; `ExtraEnv` may carry it only when it equals that path |

**Impact:** every agent in the fleet — coordinators, eval runners, reviewers, mission stages —
currently trusts prompt-injected content not to run `printenv` and post the result. The blast
radius is not read exfiltration alone: the stolen registry key can publish malicious packages
(and notify dependent inboxes, kicking off repair PRs that execute further downstream), and the
fleet GitHub token can write to every repo it can reach. A jailbroken/looping agent on a
benchmark lane has the same reach.

## Goals

**Primary Goal:** no operator secret reaches a model-facing agent subprocess unless a named,
resolved lane decision explicitly grants it, and the child environment is a deterministic,
auditable function of the task.

**Success Metrics:**
- Secret-bearing variables visible to a default-lane agent child: **5+ → 0** (measured with the
  same `env`-inspection used to find them).
- Duplicate keys in any child `execve` environment: **0**, pinned by a cross-executor test.
- `ExtraEnv` injection of a deny-listed name (`LD_*`, `PATH`, `GIT_*`, `OTEL_*`,
  `AILANG_AGENT_POLICY*`, …) without lane opt-in → **loud pre-dispatch error** through the
  existing `ValidateTaskCapabilities` seam, naming the variable.
- Clone preamble rejects any URL that is not a plain `https` git URL → regression test with
  metacharacter fixtures.
- The banked `Result` records the child env **name set digest** (names only, never values), so
  the boundary is visible in the data the way `ToolPolicy` and `PolicyDigest` already are.

## Relationship to M-SEC2 (added 2026-10-01, attended triage)

M-SEC2 (ailang-multivac `internal-docs/M-SEC2-cloud-executor-hardening.md`) hardens the **container**
boundary: Phase 1 (multivac `aac1c82`, 2026-09-30) split executors into internal / external / eval
lanes with per-lane service accounts and secret-level IAM; SEC2.2 completion binding landed in
this repo as #1426. Its post-Phase-1 secret table still injects **`github-token` and
`registry-api-key` into all three lanes, external included** — and per-run GitHub App tokens are
an M-SEC2 non-goal. So after M-SEC2 those two secrets remain in every executor container, and
nothing stops the model-facing child from reading them. This design owns that remaining
**in-container** half (host process → agent child). The two do not overlap; planning should treat
M-SEC2's lane split as the given and grant per lane on top of it. Findings E1–E4 were re-checked
against `dev` on 2026-10-01: E1, E2 and E4 hold as written; E3 holds in substance but its mechanism
was wrong and is corrected above; E5–E7 are new.

### Relationship to the 2026-10-01 executor audit (H-4) and H-6

This design **is** the audit's H-4 ("env allowlist for AI CLI subprocesses",
ailang-multivac `internal-docs/SECURITY-AUDIT-2026-10-01-ailang-executor-escape-hardening.md`,
finding F-A5). M-SEC2 Phase 1 decides which secrets a *lane holds*; this design decides which of
those the *model-facing child sees*. Neither is a boundary against a same-UID adversary on its own:
the CLI child runs as the same user as the `ailang` parent, so it can read the parent's
`/proc/<ppid>/environ` and any file the parent can read. Removing secrets from the child env and from
`~/.gitconfig` takes them out of `printenv`, crash dumps, MCP-server and tool subprocess inheritance,
and casual exfiltration — real exposure reduction — while the hard boundary is H-6 (UID split) plus
the egress lock (F-A4). This doc says "defence in depth" where it means it.

## Threat Model and Scope

**Untrusted inputs:** GitHub issue/PR text delivered in coordinator directives, benchmark task
files and `agent_env`, cloned repository content, MCP server output, model output. The agent CLI
process is model-facing: its entire environment is one `printenv` away from the prompt.

**Trusted components:** the coordinator and launcher, operator config and policy sources, the
executor package itself, the pinned toolchain.

**Explicitly outside this design's attacker model:** a compromised kernel or host administrator;
the *AILANG program runtime* confinement, which [M-EXECUTOR-POLICY-HARDENING](../../implemented/v0_41_0/m-executor-policy-hardening.md)
already repaired (root-anchored FS, redirect re-authorization, mediated tools, operator budgets);
and the future safe-runner execution service ([m-agent-safe-runner](../v1_1_0/m-agent-safe-runner.md)),
which owns "subprocess invocation … with fixed env (no inheritance from runner shell)" for its
own typed channel. This design owns the **existing agent-executor fleet's** subprocess boundary.

**Necessary nuance — the child genuinely needs some secrets:** the model-facing CLI must
authenticate to its inference provider (pi on OpenRouter needs `OPENROUTER_API_KEY`; claude uses
its credentials file). The deliverable is therefore not "strip everything" but a **typed profile**:
each lane names exactly which credentials its child needs, and everything else is dropped by
default. What cannot survive in any default profile: the fleet `GITHUB_TOKEN` and the superuser
`AILANG_REGISTRY_API_KEY` — neither is an inference credential.

## High-Impact Decisions

All six were approved by Mark on 2026-10-01 as written; the implementation notes below record
where the inventory refined *how* D2 is delivered, not *what* it decides.

| ID | Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|---|---|---|---|---|---|
| D1 | One typed `EnvPolicy` in `BuildEnvironment`: default-deny allowlist + per-executor required set + per-lane grants; all five executors route through it | Turns an ambient-inheritance boundary into an explicit-authority one; single seam prevents five divergent copies | human | design | high |
| D2 | Secret handling: credential-shaped variables (`GITHUB_TOKEN`, `AILANG_REGISTRY_API_KEY`, `GEMINI_API_KEY`, `CLAUDE_CODE_OAUTH_TOKEN`, `ANTHROPIC_API_KEY`, …) are dropped from agent children unless the lane's profile grants them; provider auth moves to file-based delivery where the CLI supports it (claude already does); lanes that `git push` get a **scoped per-task token via a git credential file outside the workspace**, never the fleet token in env | Removes the two highest-impact stealables; changes fleet push mechanics, so migration inventory is required | human | design | high |
| D3 | `ExtraEnv` validation: name syntax check plus deny-list (`LD_*`, `PATH`, `HOME`, `SHELL`, `GIT_*`, `OTEL_*`, `AILANG_AGENT_POLICY*`, `HTTP*_PROXY` unless granted); violations are loud pre-dispatch errors | Semi-trusted corpora must not gain loader/routing authority over the child | human | design | med |
| D4 | One canonical env builder: ordered construction, every key set exactly once (update-or-append semantics for all keys), documented precedence `ExtraEnv > harness-injected > allowlisted inherited`; duplicate entries structurally impossible | Fixes E3; makes "override" mean what the code comment already promises | agent | design | med |
| D5 | Clone preamble URL validation in both `ValidateCloneFlags` and `BuildClonePreamble`: `https` scheme, host present, no shell metacharacters; invalid URL is a named error (no silent fallback) | Closes E4's shell-text injection at the boundary | agent | design | low |
| D6 | Bank the child env name set digest on `Result` (names only, values never), mirroring `ToolPolicy`/`PolicyDigest` | Makes the boundary measurable in the banked corpus instead of asserted in docs | human | design | med |

### Design Freeze

- [x] Approve D1/D2: the default-deny profile, the per-lane grant mechanism, and the
      fleet-token migration (inventory of lanes that `git push` before cutover). — Approved by Mark 2026-10-01
- [x] Approve D3: the deny-list and which lanes may opt out of which entries. — Approved by Mark 2026-10-01
- [x] Approve D6: banking the env name digest (and confirm names-only, never values). — Approved by Mark 2026-10-01

D4/D5 are agent-decided implementation mechanics with no user trade-off; they ride along.

## Push-Lane Inventory (2026-10-01)

Which lanes `git push`, and with what credential. Read from `dev` @ `2aee3ed76` and
ailang-multivac `terraform/cloud_run_jobs.tf`.

| Lane | Who pushes | Credential today | What the CLI child needs |
|---|---|---|---|
| **Cloud, every job** (15 jobs: internal claude/claude-go/codex/codex-go/pi/pi-go/opencode/motoko, external `*-apikey` ×5, eval/eval-go) — all run `ailang coordinator execute-job` (`Dockerfile.agent-base:111-112`) | **The parent.** `executeCloudTask` clones, commits leftovers, pushes (`coordinator_cloud.go:690`), verifies the remote ref, and opens the PR — all after the CLI exits | `GITHUB_TOKEN` (Secret Manager, all three lanes) written into the global credential helper (E5); or a per-agent SSH deploy key (`coordinator_cloud_sshkey.go`, prod daneel-writer) | Nothing for the parent's push. *Optional:* some agents push their own branch mid-run (`coordinator_cloud.go:643-650` handles "the agent pushed it itself") and may `git fetch` a private task repo. No `gh` in any agent image (`docker/` has no gh install) |
| **Local coordinator daemon** (`internal/coordinator/provider_executor.go`) | Nobody — the local flow never pushes (no `push` in `internal/coordinator`) | — | Nothing |
| **Mission / Mac rig** (pi, codex, claude stages via `internal/mission/dispatch`) | **The child** — e.g. pi's `prepush-gate.ts` intercepts the agent's own `git push` / `gh pr create` | The rig's git uses `gh auth git-credential` (keyring), not an env token; no launchd env file sets `GITHUB_TOKEN` | Nothing from env — `gh` falls back to its stored login when `GH_TOKEN`/`GITHUB_TOKEN` are absent |

**Consequence for D2.** No lane needs the fleet token *in the child's environment*. The parent keeps
`GITHUB_TOKEN` in its own env for its own git and PR calls, and its global credential helper now
reads the token from the environment at call time (`$GITHUB_TOKEN`) instead of embedding it — so the
same helper yields nothing inside the child. For the optional mid-run push/fetch, the parent writes a
per-task git credential file (0600, in a 0700 dir outside the workspace) **scoped by URL to the task's
own repository** and points the child's git at it with `GIT_CONFIG_*` entries; it is removed when the
task ends. `AILANG_CHILD_GIT_CREDENTIALS=none` withholds even that (the parent still pushes). That
switch exists because a private-repo `git fetch` mid-run is the one behaviour the inventory could not
rule out; once a probe task per lane shows nobody needs it, flip the default.

## Per-Executor Default Profiles (D1/D2 implementation)

Inherited names pass only when on the allowlist **and** not credential-shaped; credential-shaped
names (`*_KEY`, `*_TOKEN`, `*SECRET*`, `*PASSWORD*`, `*CREDENTIAL*`, `SSH_AUTH_SOCK`, `AWS_*`, …)
pass only as an explicit grant. Unknown names are dropped.

| Executor | Granted credential(s) | Source in the cloud job | Notes |
|---|---|---|---|
| claude | `ANTHROPIC_API_KEY` **only** under `AILANG_AUTH_MODE=apikey` | external `*-apikey` jobs (per-execution override, KMS-decrypted by the parent) | OAuth lanes authenticate from `~/.claude/.credentials.json` written by the parent; `CLAUDE_CODE_OAUTH_TOKEN` never reaches the child (the old ad-hoc strip, now the seam's default) |
| codex | `OPENAI_API_KEY`, `CODEX_API_KEY` | eval jobs only; internal codex reads `~/.codex/auth.json` installed by the parent from `codex-auth-json` | `AILANG_CODEX_AUTH_SECRET` (a secret *name*) is withheld — the child does not fetch it |
| pi | the model's provider key, from the `provider/` prefix: `openrouter`→`OPENROUTER_API_KEY`, `google`→`GEMINI_API_KEY`+`GOOGLE_API_KEY`, `openai`→`OPENAI_API_KEY`, `anthropic`→`ANTHROPIC_API_KEY`, `ollama`→`OLLAMA_API_KEY`; plus, when the task carries an agent policy, the credential vars its restricted worker is allowed (pinned AI provider, std/web backend) | internal pi: GEMINI, OPENROUTER, OLLAMA; pi-go: OPENAI, OPENROUTER | The `ailang_only` lane's worker selects its credentials from pi's env (`run_policy_supervise.go` `workerEnv`), so the policy-derived grant is what keeps it working |
| opencode | same provider-prefix rule as pi | OPENROUTER | |
| motoko | `OPENROUTER_API_KEY`, `OPENAI_API_KEY`, `GEMINI_API_KEY` | OPENROUTER, GEMINI | the ratified motoko set (`motoko.go:26-29`); never `ANTHROPIC_API_KEY` |

Never granted by default to any executor: `GITHUB_TOKEN`, `GH_TOKEN`, `AILANG_REGISTRY_API_KEY`,
`CLAUDE_CODE_OAUTH_TOKEN`, `AILANG_KMS_KEY`, `AILANG_CODEX_AUTH_SECRET`, `AILANG_SSH_KEY_SECRET`,
`GOOGLE_APPLICATION_CREDENTIALS` (unless a policy-pinned Google provider asks for it).
Operator escape hatch: `AILANG_EXECUTOR_ENV_INHERIT` (comma-separated names, set on the *host*, never
from a task) forwards additional names — including a credential, which is then an explicit operator
grant visible in the banked name digest. It is the rollback lever if a lane turns out to need a
variable the profile drops.

## Solution Design

### Overview

Move child-env construction from "inherit everything, append on top" to one typed, deterministic
builder with an explicit authority source. Nothing changes for the model-facing contract the
harness relies on (correlation IDs, stdlib pin, telemetry wiring survive as explicit grants).

### Architecture

**Components:**

1. **`EnvPolicy` (new `internal/executor/envpolicy.go`)** — a resolved, immutable per-task
   profile: an `Inherit` allowlist (names forwarded from the host), an `ExtraAllowed` set
   (names `ExtraEnv` may set, defaulting to the lane's declared needs), and `Granted` entries
   (secrets the lane explicitly needs, e.g. `OPENROUTER_API_KEY` for an OpenRouter-pinned pi
   lane). Resolution is per executor + lane, decoded once at dispatch; the digest is banked.
   Default profile for every executor: locale, `PATH` (built from the executor's known binary
   dirs, not inherited raw), `HOME`, `TMPDIR`, the harness-injected correlation/telemetry/stdlib
   variables, and the lane's single inference credential. Absent profile fields mean **deny**,
   matching the project's default-deny principle.

2. **Canonical builder (rework of `BuildEnvironment`)** — construct from empty, apply in order:
   allowlisted inherited → harness-injected (stdlib, PWD, correlation IDs, chain IDs, OTEL,
   GCP pins, rig lease) → validated `ExtraEnv` → executor-required set. Every key is set through
   update-or-append so a key exists at most once; duplicates become structurally impossible and
   precedence is deterministic (fixes E3). The claude executor's ad-hoc `RemoveEnvVar` strip is
   retired into the central seam.

3. **Validation (`ExtraEnv`, clone URL)** — `ExtraEnv` names are checked against
   `EnvPolicy.ExtraAllowed` plus a hard deny-list; violations fail pre-dispatch via the existing
   `ValidateTaskCapabilities` path, naming the variable and the rule. Clone URLs are validated
   (plain `https` URL, host present, no metacharacters) in both entry points, with the same
   error style as the existing `--clone-sha` rule.

4. **Auditability** — `Result.EnvNamesDigest` (sha256 over the sorted child env **names**) is
   banked with the run; a focused test asserts no secret name ever appears in the digest input
   list for default lanes, and `docs/docs/guides/agent-tool-policy.md` (or a sibling guide)
   documents the boundary and the lane opt-in.

### Implementation Plan

**Phase 1: Canonical builder + precedence (~2 days)**
- [x] Rework `BuildEnvironment` to the ordered, dedup-by-construction builder; keep the exported
      shape so all five executors port in one commit each
- [x] Cross-executor test: no duplicate keys, precedence pinned (`ExtraEnv` > injected > inherited)
- [x] Existing env tests (`environment_test.go`, per-executor env assertions) stay green

**Phase 2: `EnvPolicy` + secret stripping (~3 days)**
- [x] `envpolicy.go`: type, resolution, default profiles per executor, lane grant mechanism
- [x] Credential-shaped vars dropped by default; claude's strip retired into the seam
- [x] Scoped per-task git credential file (outside workspace, 0600) for push-capable lanes (D2)
- [x] Adversarial tests: default-lane child env contains no `*_API_KEY`/`*TOKEN` name beyond the
      lane's single inference credential

**Phase 3: Validation + clone URL (~2 days)**
- [x] `ExtraEnv` name validation wired into `ValidateTaskCapabilities` (loud, named errors)
- [x] URL validation in `ValidateCloneFlags` + `BuildClonePreamble`; metacharacter fixtures
- [x] Eval-harness `agent_env` and browser-session flows re-tested end to end

**Phase 4: Banking, migration, docs (~2 days)**
- [x] `Result.EnvNamesDigest` + test that values are never banked
- [x] Lane migration inventory: which deployed lanes read `GITHUB_TOKEN` / registry key today;
      explicit grants added where a human ratifies them
- [x] Guide updated; example lane profile committed; release notes drafted

### Files to Modify/Create

**New files:**
- `internal/executor/envpolicy.go` — `EnvPolicy` type, resolution, deny-list (~200 LOC)
- `internal/executor/envpolicy_test.go` — allowlist/denylist/precedence/adversarial tests (~300 LOC)

**Modified files:**
- `internal/executor/environment.go` — canonical builder, `ExtraEnv` validation hook (~150 changed LOC)
- `internal/executor/executor.go` — `Result.EnvNamesDigest` field; `Task` doc updates (~20 LOC)
- `internal/executor/{pi/pi.go, codex/codex.go, claude/claude.go, opencode/opencode.go, motoko/motoko.go}` — per-executor required sets; claude strip retired (~60 LOC)
- `internal/executor/clone_preamble.go` + `clone_preamble_test.go` — URL validation (~40 LOC)
- `internal/eval_harness/{agent_runner_multi.go, browser_sessions.go}` — route `ExtraEnv` through validation (~30 LOC)
- `docs/docs/guides/` — boundary documentation (~1 page)

## Conflict Surface

This design touches `internal/executor` (not a parser/typechecker/codegen package — no
syntactic positions change — but the surface is enumerated anyway because five executors share
the seam):

| Existing behavior | Preservation requirement / intentional change |
|---|---|
| Mission dispatch constants (`internal/mission/dispatch/run.go:243-251`: messaging pins, `AILANG_MISSION_STAGE`) | MUST survive verbatim as harness-injected grants — read and cited; pinned by an existing dispatch test |
| Eval `agent_env` forwarding (`agent_runner_multi.go:248-262`, e.g. `MOTOKO_AST_AUTOREAD`) | Preserved for non-deny-listed names; deny-listed names now require lane opt-in |
| Browser session env (`browser_sessions.go:198-203`) | Preserved as explicit `ExtraAllowed` entries for the browser lane |
| Claude credentials-file flow (`claude_auth.go:65-120`) | Preserved; the env var stays available to the *parent* and is stripped from the child centrally instead of ad-hoc |
| Agents that `git push` using ambient `GITHUB_TOKEN` | **Intentional change** (D2): scoped per-task credential file; migration inventory before cutover |
| `ailang_only` lane (`MaterializeAgentPolicy`, `AILANG_AGENT_POLICY` forwarding) | Preserved; the policy path stays a harness-injected grant, never an `ExtraEnv`-overridable name |
| OTEL/trace wiring (`TRACEPARENT`, correlation IDs, resource attributes) | Preserved as explicit injected values; `OTEL_*` joins the `ExtraEnv` deny-list (values still injected by the harness itself) |

## Examples

### Example 1: default pi lane, before and after

**Before (measured in the executor container):**
```text
$ printenv | grep -iE 'key|token$'   # runnable by any bash-capable agent
GITHUB_TOKEN=gho_…                    # fleet write token
AILANG_REGISTRY_API_KEY=…              # superuser publish/key-management key
OPENROUTER_API_KEY=sk-or-…             # needed (inference)
GEMINI_API_KEY=AIza…                   # not needed by this lane
OLLAMA_API_KEY=…                       # not needed by this lane
```

**After:**
```text
$ printenv | grep -iE 'key|token$'
OPENROUTER_API_KEY=sk-or-…             # the lane's single ratified inference credential
```

### Example 2: benchmark `agent_env` injection, before and after

**Before:** a benchmark YAML sets `agent_env: {LD_PRELOAD: "/tmp/x.so"}` — silently forwarded
into the agent child (E2).

**After:**
```text
pre-dispatch error: ExtraEnv "LD_PRELOAD" is deny-listed for lane "eval/standard";
grant it in the lane's EnvPolicy or remove it from the benchmark spec
```

## Success Criteria

- [ ] AC1: default-lane agent child env contains no credential-bearing variable other than the
      lane's ratified inference credential (test enumerates `*_API_KEY`, `*TOKEN*`, `*SECRET*`
      name patterns against the built env, per executor)
- [ ] AC2: the builder never emits a duplicate key (so the slice handed to `exec.Cmd` already is the
      child env, with no reliance on `os/exec`'s last-wins de-duplication); precedence
      `ExtraEnv > injected > inherited` pinned by test
- [ ] AC3: deny-listed `ExtraEnv` names fail loudly pre-dispatch, naming the variable (via
      `ValidateTaskCapabilities` path); mission-dispatch constants and eval `agent_env`
      fixtures still pass end to end
- [ ] AC4: `BuildClonePreamble`/`ValidateCloneFlags` reject metacharacter and non-https URLs with
      named errors; the two legitimate modes (HEAD clone, fetch-by-SHA) keep their exact output
- [ ] AC5: `Result.EnvNamesDigest` banked; test asserts values are never banked and the digest is
      stable for identical tasks (determinism)
- [ ] AC6: push-capable lanes migrate to scoped per-task git credentials; inventory of affected
      deployed lanes is in the implementation report
- [ ] AC7: all existing executor tests, `make test`, `make lint`, `make check-boundaries` green;
      docs updated; example lane profile committed

## Testing Strategy

**Unit tests:**
- `EnvPolicy` resolution: default-deny, lane grants, deny-list precedence
- Canonical builder: dedup, ordering, update-or-append; inherited-vs-injected fixtures
- URL validation table: valid HEAD/SHA URLs, `;`, `$(…)`, backticks, non-https, empty host

**Integration tests:**
- Per-executor: launch-path tests (as in `environment_test.go`) assert the built env's name set
- Eval-harness `agent_env` and browser-session flows run end to end against the seam
- Adversarial: "what CAN an injected prompt read?" — a fixture child dumps its env; the test
  asserts the secret-name set is empty for default lanes

**Manual testing:**
- One real coordinator dispatch in a disposable container; inspect the banked
  `EnvNamesDigest` against a manual `printenv` of the same container

## Deferred Decisions

- Exact default `Inherit` allowlist contents beyond the obvious (locale, `PATH`, `HOME`, `TMPDIR`)
  — agent may choose, documented in the profile
- Whether `HTTP(S)_PROXY` forwarding becomes a lane grant (cloud lanes may need it) — agent may
  propose; human ratifies per lane
- `EnvNamesDigest` vs. full sorted name list on `Result` — agent may choose, digest preferred
  for size; names never values either way
- Whether a future `ailang env-audit` subcommand is worth adding for operators — deferred until
  the banked digest proves insufficient

## Non-Goals

- **Not attempted:** re-designing AILANG *program* runtime confinement — already shipped as
  M-EXECUTOR-POLICY-HARDENING; this design cites and composes with it, and does not reopen it
- The safe-runner execution service (m-agent-safe-runner owns the message-channel boundary)
- OS-level sandboxing (gVisor/Firecracker), container hardening, or egress network policy —
  different layers; the executor container's own hardening is a cloud-infra topic
- Restricting the agent CLIs' own file-system access (tool_policy already provides the
  `ailang_only` lane; ambient bash lanes are an operator choice)
- Redaction of secrets in logs/traces beyond the new "never bank env values" rule

## Timeline

**Week 1** (~4 days): Phase 1 (canonical builder + precedence) and Phase 2 start (`EnvPolicy`).
**Week 2** (~4 days): finish Phase 2 (secret stripping, scoped git credentials, migration
inventory), Phase 3 (validation + clone URL).
**Week 3** (~2-4 days): Phase 4 (banking, docs), full `make test`/`lint`/`check-boundaries`,
migration cutover review.

**Total: ~8-12 days across ~3 weeks** (planning estimate at 2× the raw implementation guess).

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| A lane silently depends on an ambient secret (`GITHUB_TOKEN` for pushes, registry key for `ailang publish`) and breaks at cutover | High | Migration inventory BEFORE cutover (AC6); explicit lane grants where a human ratifies; loud pre-dispatch refusal otherwise (no silent fallback) |
| Stripping `PATH` inheritance breaks executor CLIs that shell out to tools | Med | `PATH` is built from the executor's known binary dirs plus a small allowlisted prefix set, not inherited raw; per-executor health checks/canaries already exist to catch breakage |
| Provider auth moves to files and a CLI rejects it | Med | Claude already proves the file path; pi/codex/opencode/motoko keep their single inference credential in env (the ratified grant) — no new delivery mechanism is forced on them |
| The profile grows per-executor exceptions ("five divergent copies" again) | Med | One `EnvPolicy` type, one builder; exceptions are data in profiles, not code; drift test pins the shared seam |
| `ExtraEnv` deny-list breaks a live benchmark | Low | Deny-list is narrow (loader/routing/policy names); eval fixtures run end to end in Phase 3; breakage surfaces as a named pre-dispatch error, not a wrong run |

## Related Documents

<!-- Auto-populated by search on "executor env hardening"; manual coverage analysis below -->

**Implemented (may inform design):**
- [M-EXECUTOR-POLICY-HARDENING](../../implemented/v0_41_0/m-executor-policy-hardening.md)
  (v0.41.0) — the prior executor security hardening. Distinct ownership: it repaired
  enforcement *beneath* the AILANG program runtime (root-anchored FS, redirect authorization,
  mediated tools, operator budgets); its restricted-worker env allowlist governs the
  `ailang_only` lane's program worker, not the agent-executor subprocesses this design covers.
  Its section-4 principle ("keep provider credentials in the model-facing host, not the worker")
  is the same direction, applied there to a different boundary.
- [M-AGENT-AILANG-ONLY-EXECUTION](../../implemented/v0_39_0/m-agent-ailang-only-execution.md)
  (v0.39.0) — tool profiles and admission plumbing; this design's `ExtraEnv` deny-list composes
  with its tool lanes.

**Planned (check for overlap):**
- [m-agent-safe-runner](../v1_1_0/m-agent-safe-runner.md) — the future execution *service* with a
  typed message channel and "fixed env (no inheritance from runner shell)". Distinct artifact:
  a new service; this design hardens the executors that run today. Its checkbox 160 states the
  same principle this design brings to the current fleet.
- [m-exec-multi-executor-support](../../implemented/v0_6_1/m-exec-multi-executor-support.md)
  (implemented) — created the shared executor layer this seam lives in.

**Coverage search note:** `ailang docs search --neural` reported `fallback-simhash` (0 embeddings)
in this environment, so neural thresholds could not be evaluated mechanically; the coverage
decision above rests on reading the two closest documents and their explicit scope statements,
per the duplicate/coverage gate.

## Verification Log

| ID | Claim checked | Instrument / evidence | Result |
|---|---|---|---|
| V1 | Operator secrets are visible to the model-facing agent process in the executor container | `env \| grep -iE "key\|token\|secret\|auth\|credential"` in this coordinator-spawned session (values redacted in this doc) | 5 secret-bearing vars present: `GITHUB_TOKEN`, `AILANG_REGISTRY_API_KEY`, `OPENROUTER_API_KEY`, `GEMINI_API_KEY`, `OLLAMA_API_KEY` |
| V2 | `BuildEnvironment` inherits the full host env and all five subprocess executors pass it through | Read `environment.go:69`; `pi/pi.go:160-167`; `codex/codex.go:165-172`; `opencode/opencode.go:171-178`; `motoko/motoko.go:373,462` (moved by #1429); `claude/claude.go:253` | Confirmed; single shared seam (good for one fix) |
| V3 | Only claude strips auth secrets, ad-hoc | `grep -rn "RemoveEnvVar(" internal/executor/` (non-test) | Only `claude.go:265,270` (its own two vars) and `environment.go:75,81` (`CLAUDECODE`, rig lease) — no central secret seam exists |
| V4 | No env sanitization/allowlist seam exists anywhere in the executor layer | `grep -rniE "sanitiz\|allowlist\|denylist\|redact\|scrub" internal/executor/*.go */*.go` (non-test) | Only tool-policy and isolation comments; none govern env |
| V5 | `ExtraEnv` is forwarded unvalidated, from semi-trusted sources | Read `environment.go:99-107`; `eval_harness/spec.go:38-43` (`agent_env` from benchmark YAML); `browser_sessions.go:198-203`; `mission/dispatch/run.go:243-251` (constants) | Confirmed; mission dispatch is constants-only today, eval/benchmark sources are data files |
| V6 | Duplicate env entries occur and "last wins" is not portable | `environment.go` appends keys at 91,96,104,133,140-146,153,162,166 with plain `append`. **Re-checked 2026-10-01:** Go's `os/exec` de-duplicates `Cmd.Env` (last wins) before `execve`, so no child sees a duplicate | Original claim **corrected** — see E3: the defect is unowned precedence (harness beats `ExtraEnv`), not duplicates at `execve` |
| V7 | Clone preamble interpolates an unvalidated URL into shell text | Read `clone_preamble.go:47-77` (only non-empty checked); callers `cmd/ailang/exec.go:151,392`, `eval_harness/gemini_clone_review.go:23` | Confirmed; contrast: `--clone-sha` IS validated (SHA40) |
| V8 | Registry key is a write credential worth naming | Read `internal/config/pkgregistry.go:30` — "API key sent as X-API-Key to the validator for **publish, unpublish and key management**" | Confirmed; superuser key, not a read credential |
| V9 | Duplicate/coverage gate: no planned or implemented doc owns this boundary | `ailang docs search --neural` (fell back to simhash, 0.45-threshold unevaluable — recorded, per the gate's honesty requirement); filename/content search over `design_docs/` for `secur/sandbox/hardening`; read the two closest docs | `m-executor-policy-hardening` (implemented v0.41.0) and `m-agent-safe-runner` (planned v1_1_0) both state explicit, disjoint scope; topic is genuinely distinct |
| V10 | Conflict fixtures exist and assert what this doc claims | Read `mission/dispatch/run.go:243-251` (ExtraEnv constants), `agent_runner_multi.go:248-262` (agent_env + `AILANG_AGENT_POLICY` path), `claude_auth.go:65-120` (credentials file) | All cited fixtures verified present with the stated behavior |
| V11 | Toolpin on env: no other Go toolchain claims | `which go` in the executor container; `ailang --version` = v0.47.1 (repo at v0.49.0) | Go unavailable in-container: verification used code reading + live env + the installed binary; no `ailang check` language claims are made by this doc (it proposes no language change) |

## Notes on This Request's Provenance

The request reads "Security hardening for AILANG with gym 5.3 / Use GLM 5.3 for security
hardening of executors" (iPhone-dictated). "gym 5.3" is read as **GLM 5.3**, the model this
session runs on (`AILANG_MODEL=openrouter/z-ai/glm-5.3`) — i.e., the routing instruction is
satisfied by the session itself; the designable content is *security hardening of the
executors*. The open PRs listed in the directive (#1414 mission-charter docs, #1415 a different
Daneel design, #1416 fleet rotate-log design) were checked by title and none covers executor
security.
