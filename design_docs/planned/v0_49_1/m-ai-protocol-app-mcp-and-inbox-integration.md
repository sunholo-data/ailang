# M-AI-PROTOCOL-APP: GCP docs MCP hookup + AILANG inbox messaging for the AI protocol app

**Status**: Planned (draft written for forwarding — see "Provenance")
**Target**: v0.50.x (AILANG-side milestones); app-side work is unversioned (external repo)
**Priority**: P2 (requested by Mark 2026-09-30; blocked on an inbox that does not exist yet, by the requester's own note)
**Estimated**: 1–2 weeks AILANG-side (M1 ~0.5d docs · M2 ~1–2d · M3 ~2–3d · M4 ~1–2d; 2× realism applied)
**Dependencies**: M3's end-to-end flow depends on M2 (app inbox provisioning). No language/compiler dependencies — this doc touches no parser/typechecker/codegen surface.

## Provenance

Forwarded request (Mark, 2026-09-30, subject "Design doc for the ai protocol app", quoted with
typos intact):

> I think we need an inbox for this so won't work yet but looking for hooning up the GCP
> documentation MCP to the app would be cool for documentation, and add a tool so the app can send
> messages to AILANG inboxes itself - make a document for now that we will forward.

Open-PR coverage check at request time: the only open PR listed was
**#1414** (`docs(motoko): reset the mission charter for motoko main; loop stays paused`) — a
motoko mission-charter reset that does not cover this request. No in-repo doc covers this topic
(dual SimHash + neural related-doc search returned nothing above the noise floor; see Related
Documents for the nearest neighbours that were read instead).

Per the requester: this doc is **a document for now** — nothing here is implemented, and
implementation still needs the normal sprint gate. The doc is written to be **forwarded** to the
app side; every milestone states which side owns it.

## Interpretation & Assumptions (explicit — correct these when forwarding)

| # | Assumption | Basis / alternative |
|---|---|---|
| A1 | "the ai protocol app" is an **external app** in the sunholo ecosystem, not a component of this repo. | Repo-wide grep for `protocol app` / `ai-protocol` / `aiproto` returns zero hits (V8). If it IS meant to be something in-repo, this doc's Part A/B still applies but ownership lines change. |
| A2 | "the GCP documentation MCP" = **AILANG's hosted documentation MCP**, `mcp.ailang.sunholo.com` — the docs/knowledge MCP that runs on GCP Cloud Run in the `ailang-multivac` project (`europe-west1`, no Terraform, Cloud Build imperative deploys — m-secret-effect-remote-approval.md §deployment). | Alternative reading: Google's own GCP-documentation MCP servers. If that is what was meant, Part A becomes app-side-only config against a Google-published endpoint and this repo owes nothing to it — flagged as **Open Question Q1**. |
| A3 | "we need an inbox for this so won't work yet" = the app has **no inbox identity in the AILANG message plane**, so a two-way messaging flow cannot land end-to-end today. The requester acknowledges this; the doc designs the inbox (Part B) as the prerequisite, not as a surprise. | The send direction (Part C) technically only needs a *target* inbox, but without the app's own inbox there is no reply path, no provenance identity, and no way to distinguish app traffic from anonymous spam — so the inbox is treated as a hard dependency for M3. |
| A4 | The app is an **AI agent app with a tool surface** (hence "add a tool so the app can send messages … itself"), so MCP is the natural integration protocol for both halves. | If the app instead wants a plain REST path, the coordinator daemon's `/api/messages` GET/POST (fail-closed API key, `internal/coordinator/daemon_http.go:72`) is the existing precedent — noted in Part C as a rejected-for-now alternative. |

## Problem Statement

The app currently has (a) no live AILANG documentation access — an agent inside it would scrape
stale markdown or guess URLs, exactly the failure mode the hosted docs MCP exists to remove
(docs/docs/guides/agent-mcp.md: "Agent fetches `llms.txt` (huge, no filter)… Agent guesses URLs
from the sidebar") — and (b) no way to participate in the AILANG message plane: it cannot file
work, feedback, or questions into AILANG inboxes, and AILANG agents and humans cannot reply to
it, because it has no inbox and no supported write surface.

**Current state (all verified, see Verification Log):**

- The hosted docs MCP is **public, anonymous, read-only** knowledge with exactly **one** write
  tool: `submit_feedback`, which routes to the `public-feedback` inbox (or a `pkg:<vendor>/<name>`
  inbox) via Firestore + Pub/Sub (`internal/apiserver/feedback_tool.go`,
  `internal/feedback/publisher.go:2-3,46`). There is no general "send a message to an inbox"
  tool (V11).
- The message plane's typed inbox addresses are `pkg:`, `workspace:`, `team:`, and `plain` —
  there is **no `app:` type** (`internal/messaging/pkg_routing.go:16-21`, V2). Typed prefixes
  are load-bearing, not cosmetic (publisher.go:60-62: a minted-but-unwatched inbox is a
  measurable failure class).
- The only proven anonymous-internet → message-plane write path is the feedback pipeline, which
  is deliberately **human-triage by default** and keeps public MCP traffic away from autonomous
  dispatch (Topic-IAM separation, m-pkg-autonomous-cascade-safe: "Public MCP traffic cannot
  trigger a publish — enforced at the GCP layer before any code runs").

**Impact:** the app cannot answer AILANG questions authoritatively, cannot contribute
observations back into AILANG's loop, and — because it has no inbox — any ad-hoc integration
someone bolts on would be indistinguishable from anonymous internet traffic. Doing this without
a design is how a minted-but-unwatched inbox (or worse, an anonymous dispatch trigger) gets
created.

## Goals

**Primary goal:** give the AI protocol app (1) live, version-pinned AILANG documentation via the
hosted GCP docs MCP, and (2) an authenticated, supported tool to send messages into AILANG
inboxes — with a provisioned app inbox so the flow is two-way.

**Success metrics:**

1. An agent in the app answers an AILANG docs/stdlib question via MCP tools (`docs_nav`,
   `stdlib_search`, `prompt_get`, …) without scraping markdown.
2. A message sent by the app via the new tool appears in
   `AILANG_STORAGE_MESSAGING=gcp ailang messages list --inbox <target>` on the prod Firestore
   store, with a server-pinned `from` identity.
3. A reply sent to the app's inbox is readable by the app (M4) or by a human via `ailang
   messages list --inbox app:ai-protocol`.
4. Zero new anonymous-internet write surface: the public/anonymous MCP deployment advertises NO
   inbox tools (negative-tested).
5. No message sent by the app can trigger autonomous agent dispatch (routing stays human-triage
   by default).

## Solution Design

Three parts. **Part A is app-side config only** (nothing to build here). **Part B and Part C are
AILANG-repo milestones** and are the parts that need a sprint gate before implementation.

### Part A — Hook the app to the hosted GCP documentation MCP (app-side, M1)

The hosted docs MCP is a remote MCP server, transport `streamable-http`:

```json
{
  "mcpServers": {
    "ailang-docs": {
      "url": "https://mcp.ailang.sunholo.com/mcp/",
      "transport": "streamable-http"
    }
  }
}
```

Notes to forward with the config:

- Version scoping: version-scoped tools take `for_version`; empty string resolves to "latest".
  The app should pin to the AILANG version it targets so it never serves cross-version content
  (responses carry a `{served_for, data}` envelope for leak detection — agent-mcp.md).
- The server is read-only knowledge; it **cannot run the app's code**. The local execution MCP
  (`ailang_bootstrap`) is a separate, opt-in surface — out of scope here.
- Cost note (A9): the server deliberately advertises NO `listChanged` capability because
  MCP 2025-11-25 clients otherwise hold subscription streams open and a Cloud Run instance can
  never go idle (measured 2026-09-21 on docparse prod: 86,400 instance-seconds/day;
  `internal/apiserver/mcp.go` capabilities comment). The app's MCP client must not attempt to
  hold listen/subscription streams against this endpoint.

### Part B — Provision the app's inbox (AILANG-side, M2 — the acknowledged blocker)

The requester's "we need an inbox for this so won't work yet" is this milestone.

**Decision B1 — inbox shape.** Two options:

| Option | Description | Cost | Assessment |
|---|---|---|---|
| (a) Plain inbox `ai-protocol-app` | Zero code — `ParseInboxAddress` already treats un-prefixed addresses as `plain`. | ~0 | Rejected as the primary shape: plain inboxes are the user/personal namespace; a machine participant would be invisible in typed-inbox listings (`ListPackageInboxes`-style prefix queries) and untyped at every routing decision. |
| (b) New typed prefix `app:ai-protocol` | Mirror the existing `pkg:`/`workspace:`/`team:` pattern: add `InboxAddrApp` to `internal/messaging/pkg_routing.go`, a `FormatAppInbox` helper, and prefix-listing support. | ~1–2 days, small diff + tests | **Recommended.** Typed prefixes are the message plane's identity system and are load-bearing (publisher.go:60-62 — a minted-but-unwatched inbox is a measured failure class). Typing the app makes `app:%` queries, coordinator routing decisions, and trust boundaries all explicit (A4/A12). |

**Decision B2 — who reads it, initially.** The app's inbox is initially read by (i) humans via
`ailang messages list --inbox app:ai-protocol` and (ii) the app itself (M4 read tools). It is
**not** registered for coordinator auto-dispatch in this milestone — dispatch registration would
reuse the package-agent registry machinery and requires the Topic-IAM separation ruling from
m-pkg-autonomous-cascade-safe first (that is explicitly out of scope; see Open Question Q3).

**Decision B3 — store.** The inbox lives in the prod Firestore message plane (`ailang-multivac`)
like every other inbox. The app's runtime must read/write through supported surfaces only (the
MCP tools in Part C, or the CLI with `AILANG_STORAGE_MESSAGING=gcp` +
`AILANG_MESSAGES_PROJECT=ailang-multivac`); the retired `AILANG_MESSAGES_STORE` and the
`AILANG_STORAGE=gcp` full-plane switch are hard errors for a reason — never work around them
(CLAUDE.md §session-start).

### Part C — `inbox_send`: a tool so the app can send messages to AILANG inboxes (AILANG-side, M3; read tools M4)

**Shape.** A new Go-side MCP tool `inbox_send`, registered alongside `submit_feedback` in
`internal/apiserver` (the feedback-tool pattern: Go-side registration because it needs
Firestore + Pub/Sub access; the existing AILANG-side `mcp_tools/*.ail` file stays a documentation
reference hidden via `@noexpose`).

**Auth is the load-bearing decision (Decision C1).** The hosted MCP is anonymous today. An
anonymous `inbox_send` would be an arbitrary-internet → any-inbox write, i.e. spam and
injection into autonomous-agent routing — exactly what m-pkg-autonomous-cascade-safe closed at
the GCP layer ("Public MCP traffic cannot trigger a publish — enforced at the GCP layer before
any code runs") and what the 2026-08-31 amplification incident (m-message-plane-trust: one
legitimate message → 591 self-addressed notifications) shows the plane must never be exposed to
without provenance. Therefore:

- `inbox_send` is served on an **authenticated deployment**: the same `internal/apiserver` code
  already supports API-key header auth (`APIKeyHeader` / `APIKeyEnv`, `server.go:177-178`,
  `auth.go`, exercised by `mcp_header_auth_test.go`) and Cloud Run can additionally front it
  with an OIDC/IAM ingress requiring the app's identity (Open Question Q4). The anonymous
  public deployment must not register the tool — fail-closed, negative-tested (Success
  Criterion 4).
- The `from` identity is **pinned server-side** from the authenticated identity. The client
  may not set `from`; a `from` argument in the input schema is rejected at registration time.

**Input schema (draft):**

| Field | Type | Notes |
|---|---|---|
| `to` | string (required) | Target inbox. **Allowlist**, not free-form: initially `user`, `public-feedback`, and `app:ai-protocol`. `pkg:*` targets are excluded from v1 because package inboxes feed autonomous agents; adding them requires the auto-dispatch ruling (Q3) and the `auto:` category-tagging discipline `submit_feedback` already uses. |
| `title` | string (required) | Short title (mirrors `messages send --title`). |
| `body` | string (required) | ≤10 KB (feedback-tool precedent). |
| `category` | string (optional) | Same taxonomy as `submit_feedback` (bug/feature/docs/limitation) for triage symmetry. |
| `envelope_code` / `envelope_context` | string (optional) | Semantic-envelope fields, mirroring `messages send --envelope-*` (V10) so app messages are searchable/triage-clusterable like CLI-sent ones. |

**Behaviour guards (each has a named precedent):**

- **Per-key rate limit** — reuse `internal/apiserver/ratelimit.go`; `submit_feedback` already
  rate-limits per-IP from the rightmost `X-Forwarded-For` (feedback_tool.go handler). Per-key is
  strictly stronger.
- **Structured error envelopes, fail-loud on misconfig** — the feedback tool's lazy client
  init returns a structured error on first call when storage is not configured; no silent
  fallback (CLAUDE.md §2). Same pattern for `inbox_send`.
- **Human-triage default routing** — messages land and wait for a reader, exactly like
  `submit_feedback`'s default (publisher.go:86: "feedback files in the inbox for human/agent
  triage"). No `auto_dispatch`-equivalent argument in v1.
- **Publishes via the same path as `submit_feedback`** (Firestore write + Pub/Sub notification)
  so the app's messages get notify-daemon delivery, `messages health` visibility, and backstop
  semantics for free — NOT a new bespoke writer. (m-message-plane-trust's lesson: seams that
  report success and discard errors are the failure class; reusing the proven path avoids
  minting a new seam.)

**M4 — app-side read tools (optional, gated on Part B):** `inbox_list_unread` /
`inbox_read` scoped **strictly to the authenticated identity's own inbox** (`app:ai-protocol`),
same authenticated deployment. Public read of arbitrary inboxes is never offered. These reuse
the store's listing/reading paths (`messages list --unread` semantics) rather than raw
Firestore queries (CLAUDE.md §1: look the question up, not the tool).

**Rejected alternative (recorded):** exposing the coordinator daemon's `/api/messages`
(`internal/coordinator/daemon_http.go:72`, fail-closed key auth) directly to the app. The
daemon is a local/ops surface on a specific machine's coordinator, not a hosted API; pointing
an external app at it couples the app to one machine's daemon. The hosted-MCP route reuses the
same store with none of that coupling. Revisit only if the app deploys inside the same trust
domain.

## Examples

**App sends a question into the user inbox (end state):**

```
# inside the app's agent, as a tool call:
inbox_send(to="user", title="AILANG v0.49 effect-row question",
           body="Can row-polymorphic records omit an effect label…",
           category="docs", envelope_context="reading effects reference")
→ { "status": "ok", "message_id": "…", "to": "user", "from": "app:ai-protocol" }

# on any AILANG machine:
export AILANG_STORAGE_MESSAGING=gcp AILANG_MESSAGES_PROJECT=ailang-multivac
ailang messages list --unread        # shows the message from app:ai-protocol
```

**Reply path:** a human or agent answers with
`ailang messages send app:ai-protocol "…" --title "Re: …"`; the app reads it via the M4 read
tools (or a human checks it with `ailang messages list --inbox app:ai-protocol`).

**Docs access (Part A, day one, no AILANG-side work):** the app's agent calls
`stdlib_search("file")`, `docs_nav()`, `prompt_get(kind="agent", for_version="v0.49.0")` and
gets structured JSON — the workflow already documented in
[docs/docs/guides/agent-mcp.md](../../docs/docs/guides/agent-mcp.md).

## Success Criteria

- [ ] M1: the forwarded doc carries the app-side MCP config snippet; the app connects to
      `mcp.ailang.sunholo.com` and answers a docs question via tools (verified by the app team).
- [ ] M2: `app:` is a typed inbox address — `ParseInboxAddress("app:ai-protocol")` round-trips,
      `FormatAppInbox` exists, prefix listing (`app:%`) works; unit tests added.
- [ ] M2: a message sent to `app:ai-protocol` is visible via
      `AILANG_STORAGE_MESSAGING=gcp ailang messages list --inbox app:ai-protocol` on prod.
- [ ] M3: `inbox_send` is registered ONLY on the authenticated deployment; a negative test
      asserts the anonymous deployment advertises no inbox tools.
- [ ] M3: `from` is server-pinned; an input-schema `from` field is rejected.
- [ ] M3: send → target inbox visible in gcp store; rate limit, 10 KB body cap, structured
      error envelope, and fail-loud misconfig behaviour all tested.
- [ ] M3: no `pkg:*` target accepted in v1 (test), no `auto_dispatch` argument exists.
- [ ] M4 (if in scope): app can list/read ONLY its own inbox via the read tools (test asserts
      cross-inbox reads are rejected).
- [ ] Documentation updated: agent-mcp.md (tool), agent-messaging.md (app inbox), this doc's
      status.
- [ ] All tests passing; `make check-boundaries` clean; no new package dependencies (the
      tool reuses existing `internal/apiserver` + `internal/feedback`/`internal/messaging`).

## Timeline

| Week | Milestone | Owner |
|---|---|---|
| 1 | M1 (config snippet forwarded; app connects) + M2 (`app:` inbox type + tests + prod verification) | app team / AILANG |
| 2 | M3 (`inbox_send` on authenticated deployment, guards + negative tests) | AILANG |
| 2–3 | M4 (own-inbox read tools) — cut line if scope must shrink | AILANG |

## Related Documents

- [docs/docs/guides/agent-mcp.md](../../docs/docs/guides/agent-mcp.md) — the hosted GCP docs MCP:
  public/anonymous/read-only, tool catalog, connecting, `submit_feedback`.
- [design_docs/planned/m-message-plane-trust.md](m-message-plane-trust.md) — the message plane's
  measured seam-failure history; why `inbox_send` must reuse the proven publish path.
- [design_docs/implemented/v0_16_0/m-pkg-autonomous-cascade-safe.md](../implemented/v0_16_0/m-pkg-autonomous-cascade-safe.md)
  — Topic-IAM separation: why public MCP traffic must never reach autonomous dispatch, the
  trust model `inbox_send` inherits.
- [design_docs/implemented/v0_15_0/m-mcp-edge-throttle.md](../implemented/v0_15_0/m-mcp-edge-throttle.md)
  — per-IP rate limiting on the hosted MCP write tool (the limiter `inbox_send` reuses).
- [docs/internal/message-plane-topology.md](../../docs/internal/message-plane-topology.md) —
  the message-plane map (stores, projects, who reads what).
- [design_docs/implemented/v0_30_0/m-public-feedback-delivery-audit.md](../implemented/v0_30_0/m-public-feedback-delivery-audit.md)
  — the `public-feedback` delivery audit; precedent for auditing a new inbound surface after
  it lands.

No existing planned or implemented doc covers this topic (duplicate/coverage gate: no related-doc
match ≥ 0.45 neural / ≥ 0.75 SimHash; the search's top hits were keyword noise on unrelated
docs). Neural embeddings are currently on the `fallback-simhash` path (0 embeddings computed), so
the search was keyword-only — the human-read pass above is the real coverage check.

## Verification Log

| # | Claim | Observed |
|---|---|---|
| V1 | The hosted docs MCP is public, anonymous, read-only knowledge; `submit_feedback` is its one write tool | `docs/docs/guides/agent-mcp.md` ("public, anonymous, read-only… There's one write tool (`submit_feedback`)"); `internal/apiserver/mcp.go:70` — `registerFeedbackTool()` is the only Go-side tool registration besides module-export tools |
| V2 | Typed inbox addresses are exactly `pkg:`/`workspace:`/`team:`/`plain`; **no `app:` type exists** | `internal/messaging/pkg_routing.go:16-21` (`InboxAddrPackage…InboxAddrPlain`); grep for an app address type: zero hits (negative-existence, first-party read) |
| V3 | The feedback pipeline writes Firestore + Pub/Sub, default inbox `public-feedback`, human-triage default | `internal/feedback/publisher.go:2-3,46,79-89` |
| V4 | Per-IP rate limiting for the hosted MCP write tool exists and is config-driven | `internal/apiserver/ratelimit.go`; `feedback_tool.go:17-19` (`AILANG_RATELIMIT_RPM`/`AILANG_RATELIMIT_BURST`, RPM=0 disables) |
| V5 | `internal/apiserver` already supports API-key header auth for MCP traffic (C1's mechanism exists) | `server.go:83,177-178,258-259` (`APIKeyHeader`/`APIKeyEnv`), `auth.go`, `mcp_header_auth_test.go` |
| V6 | The coordinator daemon exposes an authenticated `/api/messages` GET/POST (rejected alternative) | `internal/coordinator/daemon_http.go:59,72` (fail-closed since M-V1-SIMPLIFY-S3 M5) |
| V7 | Cloud Run deployment topology (projects, region, no Terraform) | `design_docs/implemented/v0_26_0/m-secret-effect-remote-approval.md:285` |
| V8 | "the ai protocol app" has no counterpart in this repo | repo-wide grep `protocol app` / `ai-protocol` / `aiproto` over `.go`/`.md`/`.ail`: zero hits |
| V9 | Open PR #1414 does not cover this request | PR title/body as supplied with the task: `docs(motoko): reset the mission charter for motoko main; loop stays paused` |
| V10 | `ailang messages send` supports `--from`, `--title`, and semantic envelope fields (the schema `inbox_send` mirrors) | `.agents/skills/agent-inbox/SKILL.md` quick start; `cmd/ailang/messages_send.go` flag set |
| V11 | No `inbox_send`/`send_message` MCP tool exists today (negative-existence) | `grep -rn "AddTool(" internal/apiserver/*.go` (non-test): exactly two sites — `feedback_tool.go:59` (`submit_feedback`) and `mcp.go:213` (module exports) |
| V12 | The MCP server advertises NO `listChanged` (the Cloud Run stream-cost fix) | `internal/apiserver/mcp.go` ServerCapabilities comment (docparse prod, measured 2026-09-21) |
| V13 | Open PR coverage + duplicate-doc gate ran before creation | Only #1414 open (task header); `create_planned_doc.sh` search (SimHash + neural) returned no on-topic match — note: the script itself then died on a `set -euo pipefail` bug (see Notes) |

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No language/runtime semantics touched; message content is user data |
| A2: Replayability | 0 | No effect on traces |
| A3: Effect Legibility | +1 | The write is an explicit, named, authenticated boundary operation with structured error envelopes — not an ambient side channel |
| A4: Explicit Authority | +1 | Typed `app:` identity, server-pinned `from`, allowlisted targets, authenticated deployment; the anonymous deployment is fail-closed |
| A5: Bounded Verification | 0 | No verification-surface change |
| A6: Safe Concurrency | 0 | Reuses existing Firestore/Pub/Sub paths |
| A7: Machines First | +1 | Typed MCP tool schema + typed inbox address; machine-decidable routing for app traffic |
| A8: Minimal Syntax | +1 | No new AILANG syntax; one typed address prefix and one Go-side tool |
| A9: Cost Visibility | +1 | Reuses the measured Cloud Run stream/cost lesson (V12); per-key rate limiting bounds write cost |
| A10: Composability | +1 | Composes the existing message plane, feedback publisher, rate limiter, and auth middleware — no new seam |
| A11: Structured Failure | +1 | Fail-loud on misconfig (feedback-tool precedent), structured error envelopes, negative tests for the anonymous deployment |
| A12: System Boundary | +1 | App↔AILANG crossing is explicit, authenticated, and typed on both ends |

**Net Score: +9** → **Decision: Move forward** (once Open Questions Q1–Q4 are answered by the
forwarded doc's recipient)

### Hard Violation Check

- [x] A1: No implicit nondeterminism introduced
- [x] A3: No hidden side effects (the only new write is an authenticated, named tool)
- [x] A4: No ambient access granted (anonymous surface explicitly gains NOTHING)
- [x] A7: Not optimizing for human convenience over machine analysis

## Open Questions (to answer when forwarding)

| # | Question | Default if unanswered |
|---|---|---|
| Q1 | Which "GCP documentation MCP" — AILANG's hosted docs MCP (assumed, A2) or Google's GCP-documentation MCP servers? | Assumption A2 stands; Part A is app-side config either way |
| Q2 | Inbox name: `app:ai-protocol` is a placeholder — what is the app's canonical identity? | `app:ai-protocol` |
| Q3 | Must app messages ever trigger autonomous dispatch into coordinator agents? | **No** in v1 — human-triage only; dispatch needs Topic-IAM separation (m-pkg-autonomous-cascade-safe) and a separate ruling |
| Q4 | What auth can the app hold: static API key (C1's mechanism) or a GCP service account for Cloud Run IAM ingress? | API key first (`APIKeyHeader`/`APIKeyEnv` exists today, V5) |
| Q5 | Does the app read replies itself (M4 read tools) or does a human relay them? | M4 in scope but last in line to cut |

## Notes

- **Quorum:** `ailang design-quorum` was not run on this doc (optional step). Because the doc
  is a forwarding draft whose premises are externally answerable (Q1–Q4), the right moment for
  quorum is after the app side confirms the assumptions — before sprint planning, not now.
- **Script friction (for the record, M-DX-PI-HARNESS doctrine):**
  `create_planned_doc.sh` dies with exit 2 whenever both related-doc searches return no
  matches — under `set -euo pipefail`, `merge_results`'s internal `grep -E "^[0-9]+\."`
  returns 1 on empty input, and the `IMPLEMENTED=$(merge_results …)` assignment kills the
  script. This doc was scaffolded by hand from the script's template as a workaround. The fix
  is one line (`… | grep -E "^[0-9]+\." || true` inside `merge_results`); not applied here
  because skill-script edits need their own gate.