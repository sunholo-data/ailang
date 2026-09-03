# M-RESIDENT-AGENT-INSTANCES: a coding agent that keeps working when the laptop closes

**Status**: M1 (image), M2 (A2A surface) and M2b (platform-side client) **built and green**; M3 (terraform + instance script) **written and validated, not applied**. Phase 0 gate **passed on measurement** 2026-09-02.
**Scope**: `docker/resident/` (image, boot, A2A server), `ailang-multivac/terraform/resident_agents.tf`, `ailang-multivac/scripts/resident-instance.sh`. Runs in `ailang-multivac-dev`, region `europe-west4`.
**Target**: v0.35.0
**Priority**: P2 — nothing autonomous is blocked on it. It replaces a laptop, not a pipeline.
**Estimated**: ~6.5d total; ~4d spent (M0–M2b).
**Dependencies**: herdr v0.8.2 (Apache-2.0, pinned by sha256). Cloud Run **instances** (Preview, 2026-09). Consumed by Aitana platform via A2A — see that repo's `docs/design/v6.40.0/resident-agent-instances.md`, which is a *consuming* doc in the same shape as its browser-sessions one.

---

## Problem statement

AILANG's agent work runs on Mark's laptop. Claude Code and herdr sessions stop
when the lid closes, on a machine outside the project's IAM, its observability
and its cost accounting. The Cloud Run **Jobs** executors this estate already
runs (`terraform/cloud_run_jobs.tf`) solve the opposite problem — bounded,
dispatched, one-shot work — and cannot hold a session across days.

Two things changed in 2026-09. Google shipped **Cloud Run instances**: a
singleton that runs up to 7 days, keeps a stable HTTPS URL across restarts, and
stops and resumes on command, at about **$5.70/month** for 1 vCPU running
continuously. And **herdr** turned out to be the missing supervisor — a
headless server that keeps coding agents working with no client attached and
classifies each as working, blocked, done or idle.

## Decisions

### D1 — One agent per instance

Not several sharing a host. An instance is a singleton with its own image,
service account, volume config and URL, so one agent each means every agent
gets its own identity, its own `only-dir` home prefix, its own credentials and
its own blast radius, with no in-container multi-tenancy. Breadth of agents is
served by *more instances*, not bigger ones. Measurement backs it: 1 GiB holds
one Node agent (§ Phase 0).

Start with two variants, both already published in
`europe-west1-docker.pkg.dev/ailang-multivac/ailang/`:

| Variant | CLI | Model |
|---|---|---|
| `agent` | Claude Code | Claude |
| `agent-pi` | pi | `z-ai/glm-5.3-flash` via OpenRouter |

Neither carries the Go toolchain, so neither pays the compile tax below. The
`-go` variants come later and either size up or delegate builds to Jobs.

### D1b — The runtime is deployed twice; keep it configuration-driven

Decided with Mark 2026-09-02. The same image runs in two estates:

- **Internal** (here, `ailang-multivac-dev`): AILANG's own code work, pinned
  `z-ai/glm-5.3-flash`, no customer content.
- **Customer-facing** (Aitana estate): reads a user's documents over MCP under
  that user's identity, with an **EU-resident model resolved through Aitana's
  tier system** — their `MODEL_RESIDENCY_POLICY` defaults to `eu-strict` and
  raises on a non-EU model, so our pinned GLM id would fail closed there, which
  is the policy working.

Consequences for this image, all of them "do not hardcode":

- **No model may be baked in.** Already true — the model is a per-run parameter
  and the registry is materialised at boot from `MODELS_JSON`.
- **The MCP endpoint and any credential arrive as configuration**, never as
  image content.
- **Nothing may assume the AILANG estate** — no hardcoded bucket, project or
  registry path.

The subject-identity design for the customer-facing case is Aitana's to own and
is security-critical: the agent must NOT be able to name the user it acts for.
See their doc's Decision 5.

### D1c — Two execution modes, and streaming is required

Decided with Mark 2026-09-02. Output must reach a user live, which the current
build does not do: it captures the pane as an artifact on state change and
serves `tasks/transcript` on demand, so a caller polls and sees nothing until
something moves.

**Do not stream the terminal.** Forwarding `agent.read` as it changes streams a
TUI — ANSI, redraws, a cursor repainting text already sent. It would look like
streaming and be unusable; `strip_ansi` cannot fix a screen that repaints
itself. The unit of streamed content is an *agent event*, never a terminal
frame.

`pi --mode json` emits NDJSON, and this repo already normalises that stream in
`internal/executor/pi/pi.go`. That parser is the streaming source — reuse it
rather than writing a second one.

**The two modes are genuinely different shapes and both are needed:**

| | `interactive` (today) | `stream` (`pi --mode json`) |
|---|---|---|
| herdr sees | a TUI it can classify | a process with no TUI |
| `blocked` detection | works — the one state herdr classifies positively | **lost** |
| Human attach (`herdr --remote`) | the reason herdr was adopted | pointless |
| Output | terminal text only | clean NDJSON events |
| Fits | sessions a person will join | programmatic runs whose output must stream |

So mode is a **per-run parameter**, like the model: the A2A request selects it,
defaulting to `interactive`. This doubles the execution paths inside
`messageSend`, which is a real cost and the reason it is written down rather
than drifted into.

⚠️ **Consequence to design for, not discover:** a `stream` run cannot report
`input-required`, because that state comes from herdr's TUI detection. A run
that needs to ask a question must either be `interactive`, or the NDJSON stream
must carry its own equivalent — check what pi's event schema exposes before
assuming the second.

### D2 — A2A is the contract; herdr is optional and off the task path

⚠️ **Revised 2026-09-02 after live testing.** herdr was originally the
supervisor. It no longer is, because driving pi as a TUI does not work headless:

- `agent.prompt` enters text without submitting it — context stayed at
  `0.0%/1.3M`, so no model call was made. Passing the documented `wait` object
  did not fix it.
- Completion is undetectable: `idle`/`done` are UI-coupled, so a task sits at
  `working` forever.
- `session-protocol-gate` branches on `ctx.hasUI`; a TUI makes that true, so it
  demands a human keypress that never comes.

**Execution is now `pi --mode json` spawned directly** — the same invocation
`internal/executor/pi/pi.go` uses — whose NDJSON carries an explicit `agent_end`
and therefore a real terminal state. herdr stays installed for human attach
(`herdr --remote`) but is **not started by default**
(`RESIDENT_ENABLE_HERDR=1`), and the pi extension suite is moved aside unless
`RESIDENT_PI_EXTENSIONS=1` — it is built for a developer in this repo, and
`session-protocol-gate` would block a resident's tools.

If herdr returns for attach, it belongs in **its own image** rather than beside
the agent: co-locating them cost a process, memory and a class of silent
failures for a feature the task path never used.

The outward contract is unchanged. Only what happens inside the container did.

### D2-original (superseded) — herdr as supervisor

herdr supervises the pane and classifies agent state. It stays **inside** the
container: the outside world talks **A2A**, not a bespoke REST API wrapping
herdr's vocabulary. The specification already carries everything needed —
`TASK_STATE_INPUT_REQUIRED` is exactly herdr's `blocked`, follow-up messages on
the same task id are exactly "answer the agent", and push notification configs
are exactly "the caller disconnected hours ago".

This is what makes the estate question tractable: because the instance is an
A2A peer, **which project hosts it is not an architectural question**. The
Aitana platform reaches it over an authenticated protocol boundary, not by
sharing identity.

### D7 — Session persistence: the resident is not yet persistent

**This is the gap between what exists and what the design promises.**

`pi --mode json --no-session` is *stateless per call*: every `message/send`
spawns a fresh pi with no memory of the last. So today there is a persistent
**host** and an ephemeral **agent** — the box survives, the conversation does
not. "Long-running context" was the premise of this design, and stream mode as
first built discards it.

`--no-session` is deliberate in the job executor, documented there as
*"ephemeral run (avoids ~/.pi/sessions/ pollution)"* — correct for a one-shot
job, wrong for a resident. pi keeps sessions in `~/.pi/sessions/` and emits a
`session` event carrying the id, which `pi.go` already reads.

**Design:**

- Drop `--no-session` for resident runs; capture the session id from the first
  `session` event and store it against the **agent**, not the task — the agent
  is the thing with continuity.
- Resume on subsequent calls. **Verify pi's resume flag against the installed
  version** rather than assuming one exists in the shape we want.
- ⚠️ **Sessions live on local disk, which does not survive the 7-day restart.**
  Use the stage-in/stage-out pattern D3 already establishes for the workspace:
  restore `~/.pi/sessions/` from `/agent-home` at boot, sync back after each
  run. They must NOT be written directly to the mount — gcsfuse has no POSIX
  locking and pi writes them actively, which is precisely the corruption case
  D3 exists to avoid.
- One session per agent per instance follows from D1.

### D3 — Code on local disk; the git remote is durability

gcsfuse has no POSIX locking, so a `.git` directory on the mount corrupts. The
workspace is local ephemeral disk, re-cloned at boot; `/agent-home` holds
transcripts, herdr session state and notes. A lost instance costs the unpushed
working tree — the same exposure a laptop has.

`~/.pi` also stays on local disk: the materialised model registry carries the
live provider key, and the home is a bucket.

### D4 — The model is a per-run parameter, and must be registered

pi takes `--model` per invocation, so swapping models needs no rebuild. But pi
has **no max-tokens flag**: for a model absent from `~/.pi/agent/models.json`
it silently falls back to `maxTokens` 16384, `contextWindow` 128000,
`reasoning` false. Passing `z-ai/glm-5.3-flash` unregistered would discard 90%
of its 1.31M context with no error. The A2A server therefore **refuses**
unregistered models, naming the fallback and listing what is registered.

### D8 — The tool policy is pi's, and `bash` is outside Decision 6

pi ships `read`, `bash`, `edit` and `write` enabled, and the resident passed no
`--tools` restriction, so all four were live and **nothing said so**. Two
consequences, one cosmetic and one not.

The AILANG program allowlist (Decision 6) governs what `resident-run` will
execute. **pi's `bash` tool does not go through `resident-run`.** So while
`bash` is enabled the allowlist is a convenience and not a containment
boundary, and the doc should not imply otherwise. What actually bounds this
agent is the **container** and the **gcsfuse `only-dir` mount** — which is a
real boundary, just a coarser one than Decision 6 sounded like.

The policy is therefore explicit (`RESIDENT_TOOLS`, default
`read,edit,write,bash` — what pi already did), logged at spawn, and reported in
`/health`. Narrowing it is now a deployment choice rather than a code change.
Extensions matter here too: they add tools, so an extension is a change to this
policy and not only to behaviour.

**Extensions now have a blast radius they lacked.** Under `--no-session` a
misbehaving extension could ruin one call. With D7's sessions it rewrites the
user's *conversation*, and that rewrite is staged to GCS. Compaction is the
sharp case — it is genuinely wanted for long threads, and it edits history by
design. Hence one rotated restore point (`$AGENT_HOME/sessions.prev`) per boot,
and extensions still off by default.

### D9 — Delegated authority: the agent must act AS the user, and the platform must stay the policy point

Requirement (Mark, 2026-09-03): the resident should be able to drive the ADK
platform through the `aiplatform` CLI **with the same permissions as the user**,
to copilot for them.

The trap is the confused deputy. If the agent calls the platform as **itself**,
its rights are either broader than the user's — so a prompt-injected agent
reads documents its user cannot — or narrower, so it is useless. Neither is
"the same permissions as the user".

Three ways to get there:

| | Mechanism | Problem |
|---|---|---|
| a | Pass the user's Firebase ID token into the run | A live user credential in an agent's environment, one prompt injection away from exfiltration, and hard to keep out of the transcript |
| b | Token exchange: platform mints a short-lived delegation token naming (user, agent, scope) | Better and revocable, but a new credential type to build and audit |
| c | **No credential leaves the platform.** The agent calls back over MCP/A2A carrying only its `contextId`; the platform resolves that to the user it already knows and applies that user's permissions | The agent holds a capability handle, not an identity |

**(c) is the recommendation.** The platform already owns the `contextId → user`
mapping — `tools/resident_agent.py` derives the contextId from the ADK
invocation precisely so the model cannot name someone else's thread — so the
inbound side can resolve the caller without the agent ever holding a user
credential. Permissions are then enforced by the platform's existing authz
rather than reimplemented agent-side, and revocation is a platform decision.

It also decides where the CLI sits: the resident does not run `aiplatform` with
the user's token. It calls the platform, and the platform acts for the user.

### D10 — The fleet's pi is abandoned and eleven versions behind; the resident moved alone

Found while chasing D7: the resident could not hold a conversation because its
pi had no `--session-id`. The version was 0.73.1, and the reason is structural
rather than a stale bump.

`docker/Dockerfile.agent-pi` (and `Dockerfile.agent-eval`) ran
`npm install -g @mariozechner/pi-coding-agent` **unpinned**. That package is
**abandoned at 0.73.1**; development moved to `@earendil-works/pi-coding-agent`,
0.84.4 as of 2026-09-03. So the fleet was floating *and* reproducible only by
the accident that its package stopped moving. Publish a 0.73.2 and every image
changes silently. herdr, in the same directory, is pinned by version **and**
sha256 precisely to prevent this.

Since pi is the fleet default, this covers `agent-pi`, `agent-pi-go`,
`agent-eval` and everything downstream — **including every benchmark number
measured to date**.

**Done now (no behaviour change):** both Dockerfiles pin `0.73.1` explicitly.
The image is what it always was; a rebuild now produces it twice.

**Not done — a deliberate decision, not a side effect:** moving the fleet to
`@earendil-works`. Three reasons it is not a drop-in.

1. `internal/executor/pi/pi.go`'s NDJSON parser was written against 0.73.1, and
   an unrecognised event is *skipped*, so a schema change degrades **silently** —
   the failure mode this repo's NEVER SILENT rule exists to forbid.
2. `ailang pi install` pushes the binary's embedded extension suite across an
   extension API that may have moved in eleven minor versions.
3. **Eval comparability.** Changing the harness mid-stream breaks comparison
   with every historical result. For a benchmarking repo that is a methodology
   decision, not an infrastructure one.

**Evidence that lowers the risk of (1):** `docker/resident/lib/pi.mjs` is
written from ailang's own event vocabulary — `session`, `turn_start`,
`message_update.assistantMessageEvent.text_delta`, `tool_execution_start`,
`message_end`, `agent_end` — and it parsed **0.84.4** correctly live on
2026-09-03: 35 events, clean text, a real terminal state, and a resumed session
across two calls. The schema held for exactly the fields ailang reads. A
migration should still re-baseline the evals rather than assume it.

The resident moved alone because its requirement is different in kind: it needs
`--session-id` to exist at all, whereas the job executor is a one-shot that
`--no-session` suits and that has 0.73.1 numbers behind it.

### D11 — The singleton IS the unit of tenancy; one instance per user, not one instance for users

Asked directly (Mark, 2026-09-03): what happens when several people connect,
does the GCS mount take care of it, and does it autoscale? Answers, in order:
they collide, no, and no.

**It does not autoscale, and that is the product.** Cloud Run *instances* are
singletons — exactly one container, no replicas. That property is what buys the
stable URL, the stop/start, and a box that can hold state at all. Autoscaling
belongs to Cloud Run *services*, which is the thing this design deliberately
did not choose. So the ceiling is fixed and real.

**The mount does not isolate users.** The live spec reads
`only-dir=homes/mark` — ONE prefix for the whole instance, not one per caller.
Everyone shares a home, a workspace and a `~/.pi/sessions/`. `contextId`
separates transcripts *logically*, but the agent holds `read` and `bash`
(D8), so it can read any session file in that directory. **Several users on one
instance are not isolated, and the tool must not be offered to a skill handling
one user's confidential material on behalf of another until they are.**

**Concurrency was unbounded.** The per-session chain serialises turns within one
conversation and does nothing across conversations, and every `message/send`
spawns its own pi process. Nothing refused the Nth caller; it OOMed — and M0
measured the cgroup killing the CHILD while the container survives, so the
arriving caller silently killed somebody else's run. Now capped by
`RESIDENT_MAX_CONCURRENT_RUNS` (default 3, a guess pending M6's measurement on
a 4 GiB box, given M0 found 1 GiB hosted one agent), refused with a message
naming the ceiling, and the live count is in `/health`.

**Therefore the unit of tenancy is the instance.** One per user, started on
dispatch and stopped when idle (M4) — which is affordable precisely because D7
put the conversation on the GCS home, so a stopped instance loses nothing.
Per-user isolation then comes free from the mechanism already in place: the
`only-dir` prefix becomes the user's, not the deployment's.

Two numbers that bound this: **~$25-30/month** per always-RUNNING instance at
this size versus **≤$6** idle-stopped (M6's target), and a **quota of 100
instances per project** (`run.googleapis.com/instances`, checked on
`ailang-multivac-dev` 2026-09-03) — a hard ceiling on users per project before
a quota increase or sharding.

*(The delegated-authority follow-up Mark deferred — the agent acting on the
platform as the user — is D12, not D11.)*

### Environments

Only **dev** has a resident instance (`resident-pi-ailang` in
`ailang-multivac-dev`, europe-west4). `ailang-multivac-test` and
`ailang-multivac` have none. The consuming URL is derivable rather than
literal — `https://<instance>-<projectNumber>.<region>.run.app` — so the
platform's Terraform should COMPUTE it per environment from that env's project
number, not carry a pasted string per trigger.

## Phase 0 findings (2026-09-02) — measured, not assumed

**Compute.** `go build -a std` on live `europe-west4` instances:

| Configuration | Time |
|---|---|
| Laptop, 16 cores (M4 Max) | 3.6s |
| Laptop, one core | 27.6s |
| **1 vCPU** (Xeon 2.60GHz) | **465.4s**, then 480.2s |
| 1 vCPU (Xeon 2.00GHz) | 563.2s |
| **4 vCPU** (Xeon 2.80GHz) | **121.3s** — 3.84× for 4× |

**No burst cliff.** Repeated rebuilds varied 3.2% and instances tracked their
clock. A vCPU is small — roughly 1/17 of an M4 Max core — but honest and
purchasable. Hardware is heterogeneous: two instances minutes apart got 2.60 and
2.00 GHz Xeons.

**Memory is the tighter constraint.** `MemTotal` and Node's `os.totalmem()`
both report host memory (3916 MiB) rather than the limit, so anything sizing
from them over-commits — but V8 *is* container-aware (`heap_size_limit` 524 MiB
on a 1 GiB instance). Allocation was OOM-killed at ~800 MiB, and **the
container survived**: the cgroup killed the child, not the supervisor. cgroup
files are unreadable under the sandbox, so limits must be passed in as env.

**herdr runs headless with no controlling terminal** — confirmed. Five traps,
each of which cost time and would cost more unattended:

- `setsid herdr server` **fails silently** — no output, empty log, no server.
- `herdr status` **exits 0 whether or not the server is running**, so a
  readiness loop keyed on its exit code never waits. Probe `api snapshot`.
- `workspace list` / `pane list` return successful **empty** results with no
  server running, indistinguishable from a healthy idle one.
- **`agent.start` returns before the agent can accept input.** Prompting
  immediately fails with "not an active named agent" while herdr simultaneously
  reports `agents: 1`. `AgentInfo` carries `interactive_ready` and
  `launch_pending` for exactly this — wait on them, never on a guessed sleep.
- **`agent.prompt` without a `wait` object does not reliably submit.** Observed
  live: the prompt sat in pi's input box with context at `0.0%/1.3M`, so no
  model call was made and the task hung at `working` indefinitely. The
  socket-api docs are explicit that passing `wait` (`{until, timeout_ms}`) is
  what "submits the prompt and starts the wait in one request". The same
  paragraph documents a refusal that must not be swallowed: if the agent is
  already `blocked`, `agent.prompt` returns **`agent_blocked` without sending
  input** — ignoring that response is how a prompt silently vanishes.

`HERDR_SOCKET_PATH`, not `HOME`, is the isolation lever.

**`only-dir` works on a live mount** — the scoped prefix appears at the mount
root, so per-agent home isolation is infrastructural. **IAP and IAM are
first-class** on the resource. **GPU is supported** via
`nodeSelector.accelerator`. **`europe-west1` cannot host instances at all**
(`REGION_CAPACITY_EXHAUSTED`, absent from the API's available list), which is
why this is the estate's first split-region deployment.

## Implementation

`docker/resident/` layers onto the published `agent-pi` image: pinned herdr, a
restart-idempotent `boot.sh` that fails closed on a missing model registry, and
a Node A2A server (`server.mjs` + `lib/`) speaking herdr's **socket API**
directly — `herdr api schema --json` is authoritative where CLI flag spellings
would be guesswork, and reading it prevented three wrong-field bugs.

38 acceptance assertions run in Cloud Build against the built image, including
that `POST /panes` returns **404** — so the "no bespoke protocol" decision
cannot silently regress.

`ailang-multivac/terraform/resident_agents.tf` declares identity, IAM, bucket
and secret access; there is **no `google_cloud_run_v2_instance` resource** in
the provider, so the instance object itself comes from
`scripts/resident-instance.sh` over REST, with `--validate-only` for CI. The
IAM half never leaves terraform.

## Self-verification

Per `design_docs/README.md` §6 every design names the one command that proves
the deployed thing works. For the resident agent:

```bash
scripts/resident-instance.sh verify --name resident-pi-ailang \
  --project ailang-multivac-dev
```

It runs from a developer machine or from CI, needs **no platform backend, no
Firestore and no CI runner**, and mints its identity by impersonating the
instance's own runtime service account — read off the instance rather than
passed in, since that account is by construction on its own allowlist. That
isolation is the point: a failure here is the instance's, not four systems'
between us and it.

**Positive assertions**

| | Proves |
|---|---|
| instance reports `CONDITION_SUCCEEDED` | it is running, not merely defined |
| `/livez` serves | the process is up |
| authenticated `/health` is healthy | a caller on the allowlist is admitted |
| agent card advertises `<base>/a2a` | the A2A contract is published, not a bespoke route |
| `message/send` → `tasks/get` reaches `completed` with a non-empty artifact | **the agent can actually think** |

**Negative assertions** — the failures that would otherwise look healthy:

| | Catches |
|---|---|
| unauthenticated `/health` is refused | the 2026-09-02 stale image that served 200 to anonymous callers for 96 seconds. Expects **401 when `invokerIamDisabled`** (the app owns auth) and **403 otherwise** (the edge does), read from the live spec — asserting a constant would report config drift as a test failure |
| an unregistered model is refused | pi has no max-tokens flag, so an unknown model runs silently at 16384/128000/no-reasoning. "It worked" is exactly what this bug looks like |
| the artifact contains no ANSI escapes | forwarding the TUI instead of text (D1c's trap). Asserted before streaming lands, not after |

Everything above the last row of the positive table can pass on an agent that
cannot think, which is why the round trip is in the suite and why it is the
assertion that has failed most often.

**What a green run does not prove**: durability across the 7-day restart,
context retention between calls (D7 — it is currently `--no-session`, so a green
run says nothing about memory), cost, or behaviour under concurrent callers.

**Where it runs**: operator-invoked today; the post-deploy gate once M4 lands.
The image's own 40+ assertions run in Cloud Build separately — those prove the
build, this proves the deployment.

### Why the boot proves pi runs headless

`boot.sh` runs `pi --version </dev/null` and logs the result. On 2026-09-03 the
round trip failed for fifteen minutes at a time with **no error and a healthy
`/health`**: pi had been spawned with Node's default stdin, an open pipe that
was never written to, where Go's `exec.Cmd` leaves `Stdin` nil and gets
`/dev/null`. The invocations matched flag for flag; they differed in the one
thing nobody had written down. One second at boot, with the same stdin
discipline the executor uses, would have named it immediately.

## Open questions

1. **Terminal detection.** `blocked` is the only state herdr classifies
   positively; `idle`/`done` are UI-coupled and degenerate headless. Completion
   currently leans on the caller's staleness sweep. `events.subscribe` in the
   socket API is a better basis than chaining `agent.wait`.
2. **Provider Terraform resource** for Cloud Run instances — retire the script
   when it lands.
3. **SSH** is "coming soon" for instances; until then debugging is logs plus
   the A2A surface. `herdr --remote` over Tailscale is the escape hatch if that
   proves too thin.
