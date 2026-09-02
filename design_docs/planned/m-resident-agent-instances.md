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

### D2 — herdr is the supervisor; the contract to the outside is A2A

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

**herdr runs headless with no controlling terminal** — confirmed. Three traps,
each of which cost time and would cost more unattended:

- `setsid herdr server` **fails silently** — no output, empty log, no server.
- `herdr status` **exits 0 whether or not the server is running**, so a
  readiness loop keyed on its exit code never waits. Probe `api snapshot`.
- `workspace list` / `pane list` return successful **empty** results with no
  server running, indistinguishable from a healthy idle one.

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
