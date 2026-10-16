---
name: cloud-setup
description: Set up Claude Code cloud environments (claude.ai/code, mobile, desktop "cloud" sessions) for AILANG development — the environment's network level, env vars, secrets and setup script, plus in-session provisioning and verification. Use when configuring a new cloud environment, when `ailang` is not on PATH in a cloud session, when GCP commands fail with CREDENTIALS_MISSING, or when user says "setup cloud", "cloud environment", or mentions mobile Claude Code.
---

# Cloud Environment Setup

An Anthropic-hosted cloud session is a fresh container per session. The base image already
ships Go (matching `go.mod`), make, git, gh, node 22, python3, jq and gcloud/bq/gsutil wired
to the egress proxy. It does **not** ship the `ailang` binary on PATH, z3, or any GCP
credential — those are what this skill adds.

Two halves: the **environment config** the user enters once (only they can — it lives in the
cloud environment menu in the session title bar → Edit), and **scripts** that run per session.

## 1. Environment config (user enters once)

**Network access: Full.** The work reaches GitHub, `*.googleapis.com` (Firestore, GCS,
logging), OpenRouter, `dashboard.ailang.sunholo.com`, the registry validator and Go module
hosts; a Limited allowlist is always one host short.

**Environment variables:**

```
AILANG_STORAGE_MESSAGING=gcp
AILANG_MESSAGES_PROJECT=ailang-multivac
AILANG_CLOUD_PROJECT=ailang-multivac
GOOGLE_CLOUD_PROJECT=ailang-multivac
AILANG_AUTOPUSH=0
AILANG_AUTOREBASE=0
```

- Never `AILANG_STORAGE=gcp` — it moves all three stores, not just messaging.
- `AILANG_AUTOPUSH=0`: the Stop hook rebases and pushes `dev`. Right on the Mac; wrong from
  an ephemeral clone that may be far behind `origin/dev`. Push branches deliberately.

**Secrets** (environment settings, never pasted into chat) — only the ones you use:

| Variable | For |
|---|---|
| `GCP_SA_KEY_JSON` | Full JSON of a dedicated service-account key. `setup.sh` writes it to the ADC path and activates it for gcloud. Least privilege on `ailang-multivac`: `roles/datastore.user` (message store), `roles/logging.viewer` (cascade debugging), `roles/storage.objectViewer`. |
| `AILANG_REGISTRY_API_KEY` | `ailang publish` / unpublish only. Installing packages needs nothing. |
| `OPENROUTER_API_KEY` | Evals and benchmarks. |
| `ANTHROPIC_API_KEY`, `GEMINI_API_KEY`, `OPENAI_API_KEY` | Direct-provider evals and live tests only. |

**Setup script:**

```bash
bash /home/user/ailang/.claude/skills/cloud-setup/scripts/setup.sh || true
```

**What needs the GCP key.** The gcloud CLI is optional, but the message plane is not: the
`gcp` message store is Firestore through the Go client, which authenticates with ADC. With no
key, everything else works (build, `make test`, `ailang check`/`run`, evals with a provider
key, git and GitHub through the session's connector) and `ailang messages` reads an empty local
store. Only `roles/datastore.user` is needed for the inbox.

## 2. Per session

```bash
.claude/skills/cloud-setup/scripts/verify.sh   # read-only readiness check, exit 1 on failure
.claude/skills/cloud-setup/scripts/setup.sh    # idempotent; rerun if verify fails
```

`setup.sh`: checks Go against `go.mod`, apt-installs z3/openssh-client if missing, writes
`GCP_SA_KEY_JSON` to `~/.config/gcloud/application_default_credentials.json` and activates it,
runs `make quick-install` (version-stamped) and links `/usr/local/bin/ailang`, smoke-tests
`examples/runnable/hello.ail`. It never fails the boot.

Then confirm the message plane per CLAUDE.md: `ailang storage status` shows
`messaging gcp (AILANG_STORAGE_MESSAGING)`, and `ailang messages list --unread` prints a
`store: gcp (...)` header.

## 3. Do not reintroduce

These were in the 2025 version of this skill and are wrong for the current container:

- **`GOTOOLCHAIN=local`** — `go.mod` pins a newer Go than any old image; `local` makes that a
  hard error. Leave the default so go fetches the toolchain.
- **`GOPROXY=direct`** — hits GitHub's anonymous throttle. Use `https://proxy.golang.org|direct`.
- **Rewriting `/etc/resolv.conf`, `wget --no-check-certificate`** — egress goes through a TLS
  proxy whose CA is installed. Never disable TLS verification or unset `HTTPS_PROXY`; proxy
  problems are diagnosed with `/root/.ccr/README.md`.
- **Installing Go / gh from tarballs** — already in the image.

Details and symptoms: [resources/troubleshooting.md](resources/troubleshooting.md). The
coordinator's own cloud executor images (`docker/Dockerfile.agent-base`) are the reference for
what an AILANG container needs; their comments record the measured traps.
