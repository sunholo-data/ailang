---
title: Autonomous Package Updates
description: How AILANG packages auto-bump downstream dependents in cloud — what fires the cascade, who's authorized to trigger it, and how to observe one in flight.
---

# Autonomous Package Updates

When you publish a new version of a package with `ailang publish`, AILANG can automatically open update PRs on every downstream package that depends on it. The cascade runs in the cloud (Cloud Run agents per package), each PR is opened against `sunholo-data/ailang-packages` for human review, and the whole flow is auditable end-to-end via `ailang pkg cascade status`.

This guide covers the **golden path** as it works today and the **safety boundary** that makes it safe to point at production packages.

## The 30-second mental model

```
You: cd packages/my-pkg && ailang publish
   │
   ▼
ailang publish detects 3 downstream dependents from the registry
   │
   ▼
Coordinator publishes 3 messages on ailang-cascade Pub/Sub topic
(IAM-restricted: ONLY the coordinator service account can publish)
   │
   ▼
Each dependent's package agent (Cloud Run Job) wakes up,
sees `Source: cascade` attribute, follows pkg-update.md template
   │
   ▼
Agent: bumps dep version → runs tests → opens PR
   │
   ▼
Human (you) reviews + merges PR
```

**v1 is always-PR**. No package version is auto-merged or auto-published downstream. The cascade does the boring work; you keep the merge call.

## Two modes of update propagation

| Update kind | Path | Verified |
|---|---|---|
| **Cascade-triggered bump** (you ran `ailang publish`) | Coordinator → `ailang-cascade` topic → dependent's package agent → PR | ✅ |
| **Public feedback** (someone files via `mcp.ailang.sunholo.com/submit_feedback`) | MCP → `pkg:vendor/name` inbox → package agent → **stops at filing an issue** | ✅ |

The crucial distinction: only cascade-topic messages count as authoritative bump triggers. Public MCP feedback can never publish a new version, no matter what the body asks for. The IAM layer enforces this before any agent code runs (see [Security model](#security-model) below).

## Doing it: a real cascade

### Prerequisites

- `ailang` CLI ≥ v0.16.0 installed (`ailang --version`)
- Push access to `sunholo-data/ailang-packages` (you'll review the cascade-driven PR)
- Your package is registered (`ailang search vendor/name` returns it)

### Step 1 — bump and publish

Edit your package's `ailang.toml`:

```toml
[package]
name = "vendor/my-pkg"
version = "0.2.1"  # was 0.2.0
```

Then publish:

```bash
cd packages/my-pkg
ailang publish
```

You'll see (annotated):

```
✓ Built vendor/my-pkg v0.2.1
✓ Pushed to registry: vendor/my-pkg@0.2.1
✓ Found 3 dependents: vendor/dependent-a, vendor/dependent-b, vendor/dependent-c
→ Notified dependent vendor/dependent-a (ID: msg_…, class: patch, legacy inbox)
→ Cascade-topic notification published for vendor/dependent-a (root=vendor/my-pkg@0.2.1)
→ Notified dependent vendor/dependent-b (ID: msg_…, class: patch, legacy inbox)
→ Cascade-topic notification published for vendor/dependent-b (root=vendor/my-pkg@0.2.1)
→ Notified dependent vendor/dependent-c (ID: msg_…, class: patch, legacy inbox)
→ Cascade-topic notification published for vendor/dependent-c (root=vendor/my-pkg@0.2.1)

Cascade ID: <chain_id from provenance>
Watch with: ailang pkg cascade status vendor/my-pkg@0.2.1
```

### Step 2 — watch the cascade

In another terminal:

```bash
ailang pkg cascade status vendor/my-pkg@0.2.1
```

After ~3 minutes (typical for a 3-dependent cascade with Sonnet agents):

```
Cascade: vendor/my-pkg@0.2.1
Chain ID: chain_2026…
Published: 2026-04-30T14:02:11Z

DAG (3 dependent package(s)):
  vendor/my-pkg@0.2.1 (root)
  ├── vendor/dependent-a
  │      status: merged
  │      PR:     https://github.com/sunholo-data/ailang-packages/pull/47
  │      cost:   $0.18
  ├── vendor/dependent-b
  │      status: merged
  │      PR:     https://github.com/sunholo-data/ailang-packages/pull/48
  │      cost:   $0.21
  └── vendor/dependent-c
         status: queued
         notes:  no cascade-driven version yet
```

### Step 3 — review and merge

Each cascade PR carries a `[smoke-test]`-free, normal commit message of the form:

> `[cascade] bump vendor/my-pkg → 0.2.1 (cascade chain_…)`

PR contents: bumped dependency in `ailang.toml`, regenerated lock file, no behavioural changes. Review as you would any dep bump and merge.

`ailang pkg history vendor/dependent-a` will show the new version with provenance pointing at the cascade root.

## Security model

### Threat model: a stranger via the public MCP

The MCP server `mcp.ailang.sunholo.com` accepts `submit_feedback` calls from anyone, including with `package="vendor/your-pkg", auto_dispatch=true`. Without further mitigation that would route a "please publish v999" body straight to a package agent's inbox.

**Mitigation: separate Pub/Sub topic, IAM-restricted publish.**

| Topic | Purpose | Publish IAM |
|---|---|---|
| `ailang-messages` | All public-facing message traffic, including MCP `submit_feedback` | Open (public) |
| `ailang-cascade` | Authoritative package-update triggers from `ailang publish` | **Coordinator service account only** |

Each pkg-* agent's `pkg-update.md` template guards on the `Source` attribute carried by the inbound message. The agent acts on bump requests **only when `Source: cascade`**. Anything else (`Source: messages`, empty, unknown) — even with `auto_dispatch=true` and a polite body — gets filed as a GitHub issue and the agent stops.

The IAM enforcement runs **at the GCP layer before any agent code executes**. A stranger via the public MCP literally cannot publish to the cascade topic — the platform refuses the publish with `PERMISSION_DENIED`. Even if a future template change loosened the source check, or a template-routing bug let a malicious body reach the bump branch, the cascade message simply does not exist.

To verify in your environment:

```bash
gcloud pubsub topics get-iam-policy ailang-cascade --project=ailang-multivac
# Expected: only ONE serviceAccount with roles/pubsub.publisher,
# and it's the coordinator SA (e.g. ailang-coordinator@…)
```

### Threat model: compromised dependent in the graph

A package in the cascade graph could be compromised — its agent might try to bump siblings into a cycle, exfiltrate secrets via `Net`, or runaway-spend on Sonnet calls.

Three defenses already in the box:
- **Circuit breaker.** N consecutive failures across the cascade abort further dispatches. (`internal/coordinator/cascade_scheduler.go::CascadeCircuitBreaker`)
- **Per-cascade cumulative budget cap.** Each package owner sets `[cascade] max_cost_usd` in their `ailang.toml` (default $1.00). When the cumulative cost across the cascade DAG approaches the cap, the scheduler aborts further dispatches with a structured event. Per-task hard ceilings (existing budget machinery) are unchanged.
- **Always-PR.** No package agent auto-merges. A compromised agent can open noisy PRs but cannot land code without a human reviewing.

Future complement: in-language `Net<external>` taint sinks via [M-TAINT-EFFECTS](/design_docs/planned/v0_16_0/m-taint-effects).

### Threat model: drift between intent and act

What if the agent's interpretation of the bump differs from what you wanted? Two safeguards:

1. The PR carries a `[cascade]` title prefix and links back to the cascade root via the `cascade-id` commit trailer. You see exactly which publish triggered it.
2. The agent operates inside its package subdirectory (`subdirectory:` in `config.cloud.yaml`); it cannot touch siblings.

## Configuration cheat-sheet

In your package's `ailang.toml`:

```toml
[package]
name = "vendor/your-pkg"
version = "0.1.0"

[dependencies]
"vendor/some-dep" = "0.5.0"

# Optional — defaults to $1.00 if absent
[cascade]
max_cost_usd = 0.50  # Lower for low-budget fixtures, higher for hot-path libs
```

In `ailang-multivac/config/config.cloud.yaml`, your package agent (created by the M-PKG-AUTONOMOUS-UPDATES rollout):

```yaml
- id: pkg-vendor-your-pkg
  inbox: pkg:vendor/your-pkg
  workspace: sunholo-data/ailang-packages
  subdirectory: packages/your-pkg
  invoke:
    type: prompt
    template_file: /etc/ailang-config/templates/pkg-update.md
  auto_merge: false  # Always-PR for v1
```

## Diagnostics

### Cascade fired but no PR?

```bash
# Was the cascade message actually published?
gcloud pubsub subscriptions pull ailang-cascade-coordinator \
  --project=ailang-multivac --limit=5 --auto-ack=false

# Did the coordinator dispatch the agent?
gcloud logging read 'resource.type="cloud_run_revision"
  AND resource.labels.service_name="ailang-coordinator"
  AND textPayload:"vendor/your-dependent"' \
  --project=ailang-multivac --limit=20 --freshness=10m

# Did the agent run actually start?
ailang chains view <cascade-id>  # uses local SQLite chain store
```

### PR opened but no provenance link?

The dependent's published version may not yet have its `provenance.correlation_ids` populated — the registry validator updates this on the NEXT publish. Check `ailang pkg provenance vendor/dependent@<new-version>` after the cascade lands.

### Wanted to ship a quick fix but cascade flow blocks me?

For a one-off direct edit, skip the cascade entirely: open a manual PR against the dependent. The package agent only fires on cascade-topic messages, so it won't conflict with manual work.

## What's planned next

This guide describes M-PKG-AUTONOMOUS-CASCADE-SAFE. Two larger sprints build on it:

- **AILANG core → packages cascade** (separate sprint). When AILANG itself ships a new version, all 13 package agents rebuild against the new compiler and decide their bump class. Reuses the cascade scheduler from this work.
- **M-PKG-TRUSTED-AUTONOMOUS-EVOLUTION** (v0.11.0+ planned, 4-week sprint). Adds signed publishing, admission policy, effect ceilings — the full guardians-style trust model. This work is its precondition.

## References

- [M-PKG-AUTONOMOUS-CASCADE-SAFE design doc](/design_docs/implemented/v0_16_x/m-pkg-autonomous-cascade-safe)
- [M-PKG-AUTONOMOUS-UPDATES design doc](/design_docs/implemented/v0_10_0/m-pkg-autonomous-updates) — the v0.10.0 work this validates and secures
- [pkg-update.md template](https://github.com/sunholo-data/ailang-multivac/blob/main/config/templates/pkg-update.md) — what the package agent reads
- [pkg-feedback.md template](https://github.com/sunholo-data/ailang-multivac/blob/main/config/templates/pkg-feedback.md) — what the agent reads for public feedback messages
