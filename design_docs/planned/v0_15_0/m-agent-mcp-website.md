# M-AGENT-MCP: Remote MCP Server for ailang.sunholo.com

**Status**: Planned
**Target**: v0.15.x
**Priority**: P1 (Strategic — directly serves Axiom A7 "Machines First")
**Estimated**: 7.5 days
**Dependencies**: `internal/apiserver/mcp.go` (already built, in-tree), `ailang serve-api --mcp-http` (already shipping), `ailang-multivac/` Cloud Run pipeline

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is an **agent-facing surface area** change. It exposes existing structured data through a protocol AI agents already speak (MCP), rather than asking them to scrape Markdown.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | Read-only introspection of static corpora; no runtime change |
| A2: Replayability | **+1** | Tool calls are pure functions over versioned snapshots; replays are byte-identical |
| A3: Effect Legibility | 0 | Server runs AILANG modules with declared `IO`/`Net` caps only |
| A4: Explicit Authority | **+1** | Tools are exposed via `@route`/`@mcp_name` — no ambient access; allowlisted by `--routes-only`. The single write tool (`submit_feedback`) writes only to a Pub/Sub topic via narrowly-scoped service account (`roles/pubsub.publisher` on one topic) |
| A5: Bounded Verification | 0 | No type system change |
| A6: Safe Concurrency | 0 | Reuses per-request evaluator isolation already in `serve-api` |
| A7: Machines First | **+3** | The whole point — agents stop scraping HTML and start calling typed tools |
| A8: Minimal Syntax | 0 | No syntax change |
| A9: Cost Visibility | **+1** | Benchmark introspection tool surfaces per-model cost/token data |
| A10: Composability | **+2** | New introspection tools = new `.ail` files; auto-registered as MCP tools — no Go change required to extend |
| A11: Structured Failure | 0 | Reuses MCP error envelope |
| A12: System Boundary | **+1** | Each tool has a typed signature; agents see schemas, not free-form prose |

**Net Score: +9** → **Decision: Move forward**

### Hard Violation Check

- [x] A1: Tools are pure reads over versioned data; no nondeterminism
- [x] A3: AILANG modules declare `! {IO, FS, Net}` explicitly; capability budget enforced
- [x] A4: `--routes-only` filtering means only `@route`/`@mcp_name`-annotated functions are exposed
- [x] A7: This is the strongest "machines first" win available — turns the whole docs site into a typed API

---

## Problem Statement

### Current State

`ailang.sunholo.com` is a Docusaurus site optimized for **humans browsing Markdown**. The supporting structured data is excellent:

- **52 doc pages** (`docs/docs/`) — guides, reference, architecture
- **30 stdlib modules** (`stdlib/std/*.ail`) — typed signatures + docstrings
- **162 examples** (`examples/*.ail`) — passing 99% with metadata in `examples_report.json`
- **39 baseline benchmark folders** (`eval_results/baselines/`) — per-model cost, pass-rate, error categories
- **928 design docs** (`design_docs/`) — planned/implemented/rejected with rationale
- **22 teaching prompt versions** (`prompts/`) — paired with language versions
- **`llms.txt`** at site root — flat dump of everything

But an AI agent investigating AILANG today has to:
1. Scrape `llms.txt` (huge, no filter — burns context),
2. Or guess URLs from the sidebar and fetch one Markdown page at a time,
3. Without ever knowing which prompt pairs with v0.14.2, which examples cover effects, or which models historically fail at pattern matching.

### Insight: We Already Have the Server

[`internal/apiserver/mcp.go`](../../../internal/apiserver/mcp.go) implements a full MCP server that **auto-registers any exposed AILANG function as a typed MCP tool**. It supports both stdio (for Claude Desktop / Cursor / Cline) and **Streamable HTTP** (`HTTPHandler()`, line 258) — exactly the transport remote MCP needs.

The user-facing entry point already exists: `ailang serve-api --mcp-http ./api/` ([cmd/ailang/serve_api.go:270](../../../cmd/ailang/serve_api.go)).

So the task is **not** "build an agent API". It is:
1. Write a small set of AILANG modules that introspect the AILANG corpus
2. Bundle them with the docs/benchmark snapshots into a Cloud Run image
3. Deploy as `mcp.ailang.sunholo.com`
4. Advertise it from the Docusaurus site

This dogfoods AILANG (the agent API is written in AILANG), and any new tool is a new `.ail` file — zero Go work to extend.

### The Bigger Insight: MCP as Canonical Docs Source

Once we have a versioned, queryable, always-fresh MCP endpoint, the **CLI's embedded prompts become a fallback**, not the source of truth.

Today, [`cmd/ailang/main.go:19`](../../../cmd/ailang/main.go#L19) does `//go:embed all:prompts` — all 50+ prompt versions are baked into every binary. Consequences:

- A user on `ailang v0.10` gets v0.10's view of best practices forever, even if v0.14 ships materially better guidance.
- `ailang prompt` returns whatever was true at compile time. There is no path for "we learned this last week" to reach existing installs short of reinstall.
- The same staleness applies to anything else baked in: stdlib type signatures, example metadata, axiom scorecard, devtools reference, error-message hints.
- External tooling that depends on AILANG knowledge — the `ailang-bootstrap` skill and any equivalents in other harnesses — has the same staleness problem, only worse, because skill files are edited by hand and drift independently.

With a Cloud Run MCP server that ships on every release:

- `ailang prompt` can call `prompt_get` and **prefer fresh content** when the network is up, falling back to the embedded copy when offline. The embedded copy becomes a safety net, not the canonical version.
- The `ailang-bootstrap` skill (and any harness-side equivalent) stops embedding lengthy AILANG guidance — it becomes a thin shim that adds `mcp.ailang.sunholo.com` to the agent's MCP config. Updates to AILANG guidance ship via redeploy, not via every user editing their `SKILL.md`.
- Documentation improvements reach the field the day they're shipped, across every CLI version still in use.
- The website's `llms.txt` shrinks dramatically — instead of dumping everything, it tells agents to query the MCP for what they need, when they need it.

This reframes the project from "agent helper" to **"single live source of truth for AILANG guidance, served over a protocol every coding agent already speaks."** Determinism is preserved because each MCP image is pinned to a specific AILANG release; users who need reproducibility pin to a versioned subdomain (e.g. `mcp-v0-15.ailang.sunholo.com`).

### Why MCP, Not REST

- **MCP is what agents already speak.** Claude Desktop, Cursor, Cline, Continue, Claude Code all support remote MCP. A user can add `mcp.ailang.sunholo.com` once and every agent on every harness gets it for free.
- **Tool discovery is built-in.** The agent calls `tools/list` and gets typed schemas — no OpenAPI to host separately.
- **We already validated tool-name compliance** (`validateMCPName`, Claude Desktop strict regex).
- **Streamable HTTP transport** matches Cloud Run's request/response model cleanly.

---

## Proposed Tool Surface

What an agent investigating AILANG actually wants to ask. Each row is one `@mcp_name` AILANG function.

### Version Scoping (read this first)

The MCP corpus is **multi-version**. The image carries `snapshot/v0.10.0/`, `snapshot/v0.11.4/`, … `snapshot/v0.15.x/` side-by-side, each with that release's stdlib, examples, prompts, limitations, and docs.

Every "language facts" tool below takes an optional **`for_version`** parameter:

- **Omitted** → server returns content for `latest` AND tags the response with the version actually used (`{served_for: "0.15.0", available_versions: [...]}`)
- **Provided** → server returns content for that version exactly; if it doesn't have it, returns `{error: "unknown_version", nearest: "0.14.2"}` so the caller can fall back without retry guessing
- **CLI behavior** — `ailang prompt --source auto` always passes `for_version=$AILANG_VERSION`. The CLI never receives prompts for builtins it doesn't have.

This keeps "fresh" honest: an agent on v0.10 gets v0.10's content (which may have been *re-rendered* with later doc fixes — same builtins, better wording), never v0.15's content referencing builtins that don't exist in their binary.

Tools that return **historical / observational data** (benchmarks, design docs, changelog) are unscoped — they describe history, not the current language.

| Category | Version-scoped? |
|---|---|
| Language reference (stdlib, prompts, effects, limitations) | **Yes** — `for_version` |
| Examples & patterns | **Yes** — `for_version` (filtered to those that compile under that version) |
| Docs & navigation | **Yes** — sidebar/content as of that release's docs site |
| Design docs, roadmap, changelog, benchmarks | **No** — historical record |

### Language Reference (version-scoped, read-only, pure)

| Tool | Purpose | Inputs | Output shape |
|------|---------|--------|--------------|
| `ailang_versions` | List all versions the server can answer for | — | `{latest, available: ["0.10.0", "0.11.4", ...]}` |
| `prompt_get` | Fetch the canonical teaching prompt | `{for_version?, kind: "agent"\|"devtools"\|"compact"}` | `{markdown, sha, prompt_version, served_for}` |
| `stdlib_modules` | List stdlib modules with one-line summaries | `{for_version?}` | `{served_for, modules: [{name, summary, function_count}]}` |
| `stdlib_module` | Full module: exports + signatures + docstrings | `{module, for_version?}` | `{served_for, name, exports: [...], imports}` |
| `stdlib_search` | Find stdlib functions by name/signature/keyword | `{query, for_version?, limit?}` | `{served_for, hits: [...]}` |
| `limitations_list` | Known limitations (Y-combinator, etc.) by category | `{category?, for_version?}` | `{served_for, limitations: [...]}` |
| `effects_catalog` | All declared effects with capability mapping | `{for_version?}` | `{served_for, effects: [{effect, capabilities, since_version}]}` |

### Examples & Patterns (version-scoped)

| Tool | Purpose | Inputs | Output shape |
|------|---------|--------|--------------|
| `examples_list` | Filter examples that PASS under `for_version` | `{features?, status?, for_version?, limit?}` | `{served_for, examples: [...]}` |
| `example_get` | Full source + metadata for one example | `{path, for_version?}` | `{source, features, expected_output, last_passed_version, served_for}` |
| `example_for_concept` | Best-fit example for a concept query | `{concept, for_version?}` | `{path, source, why_relevant, served_for}` |

### Design & Roadmap

| Tool | Purpose | Inputs | Output shape |
|------|---------|--------|--------------|
| `design_docs_list` | List planned/implemented/rejected docs | `{state, version?}` | `[{title, version, status, summary}]` |
| `design_doc` | Full design doc by id | `{slug}` | `{markdown, axiom_score, status, related_docs}` |
| `roadmap` | What is shipping next, what is rejected and why | — | `{next_version, planned: [...], rejected: [...]}` |

### Benchmarks (the killer app for agents picking a model)

| Tool | Purpose | Inputs | Output shape |
|------|---------|--------|--------------|
| `benchmarks_list` | List benchmark suites and runs | `{since_version?}` | `[{benchmark, language, model, pass_rate, version}]` |
| `benchmark_run` | One run's full result | `{run_id}` | `{code, compile_ok, runtime_ok, stdout_ok, cost_usd, tokens, error_category}` |
| `benchmarks_for_model` | Where does a given model pass/fail? | `{model, version?}` | `{passes: [...], fails_by_category: {...}, total_cost}` |
| `benchmarks_compare` | Pairwise model/version comparison | `{model_a, model_b, version?}` | `{a_only_passes, b_only_passes, both, neither}` |

### Discovery & Navigation

| Tool | Purpose | Inputs | Output shape |
|------|---------|--------|--------------|
| `docs_nav` | Sidebar tree as JSON (replaces scraping the sidebar) | `{for_version?}` | `{served_for, sections: [{title, path, children}]}` |
| `docs_search` | Full-text search across doc Markdown | `{query, for_version?, limit?}` | `{served_for, hits: [...]}` |
| `changelog_for_version` | What changed in a release | `{version}` | `{added, changed, fixed, breaking}` |

### Public Feedback (the only write tool in v1)

| Tool | Purpose | Inputs | Output shape |
|------|---------|--------|--------------|
| `submit_feedback` | Anonymous bug report / feature request / docs gap, queued for human review | `{title, body, category: "bug"\|"feature"\|"docs"\|"limitation", ailang_version, snippet?, agent_context?: {model, harness}, contact?: {email, github}}` | `{ticket_id, queued_at, view_url?}` |

Designed so an agent can file a useful report **mid-session** the moment it hits a wall ("I tried `std/sqlite`, error message was confusing, here's the snippet"). Anonymous is fine — the value is the signal, not the identity. `contact` is optional and used only if the reporter wants a follow-up.

Spam mitigation (no auth needed):
- Per-IP rate limit at the LB: 5/min, 50/hour, 200/day
- Body cap 10 KB, snippet cap 4 KB
- Daily topic-level cap (overflow → 429, never silent drop)
- All submissions land in a Pub/Sub queue → Firestore moderation list, **never** direct-write into anything user-visible
- Reviewed items become GitHub issues or `ailang messages` entries via existing tooling

That's **~21 tools**, twenty thin read-only filters over a JSON snapshot (≤30 LoC of AILANG each) plus one write tool that publishes to Pub/Sub. The `examples_report.json`, `eval_results/baselines/`, and `prompts/` directories are already structured — the AILANG modules are thin readers.

---

## Architecture

```
┌──────────────────────────────────────────────────────┐
│  ailang.sunholo.com (Docusaurus, GitHub Pages)       │
│  - <link rel="mcp" href="https://mcp.../mcp/">       │
│  - "Add to Claude" button on landing page            │
│  - llms.txt now points at MCP for richer queries     │
└────────────────┬─────────────────────────────────────┘
                 │ (agent reads link, adds remote MCP)
                 ▼
┌──────────────────────────────────────────────────────┐
│  mcp.ailang.sunholo.com (Cloud Run)                  │
│                                                      │
│  Image (multi-version snapshot):                     │
│    /usr/local/bin/ailang   (binary, latest only)     │
│    /srv/mcp-tools/         (AILANG @mcp_name funcs)  │
│    /srv/snapshot/                                    │
│      ├── versioned/                                  │
│      │   ├── 0.10.0/{stdlib,examples,prompts,docs,  │
│      │   │           limitations,effects.json}       │
│      │   ├── 0.11.4/...                              │
│      │   ├── 0.14.2/...                              │
│      │   ├── 0.15.0/...                              │
│      │   └── latest -> 0.15.0  (symlink)             │
│      ├── unscoped/                                   │
│      │   ├── design_docs/                            │
│      │   ├── benchmarks.sqlite                       │
│      │   ├── changelog/                              │
│      │   └── roadmap.json                            │
│      └── docs.sqlite (FTS5; partitioned by version)  │
│                                                      │
│  Entrypoint:                                         │
│    ailang serve-api --mcp-http \                     │
│      --port 8080 \                                   │
│      --routes-only \                                 │
│      --caps FS \                                     │
│      /srv/mcp-tools/                                 │
└────────────────┬─────────────────────────────────────┘
                 │ Streamable HTTP /mcp/
                 ▼
┌──────────────────────────────────────────────────────┐
│  Agent (Claude Desktop, Cursor, Cline, Claude Code)  │
│  - tools/list → 20 typed tools                       │
│  - tools/call → benchmark, doc, stdlib, example data │
└──────────────────────────────────────────────────────┘
```

### Why a New Cloud Run Service (not piggyback on an existing one)

The `ailang-multivac/` repo today runs three Cloud Run services: `coordinator` (task dispatch + GitHub webhooks + KMS-wrapped secrets), `dashboard` (telemetry viewer), and `website-builder` (form-driven site generation). None is a natural host:

| Dimension | Existing services | MCP service |
|---|---|---|
| **Binary** | Custom Go (`coordinator`, `dashboard`, `website-builder`) | `ailang serve-api --mcp-http` (dogfoods AILANG) |
| **Image rebuild trigger** | Code change in `cmd/...` | New AILANG release (snapshot rebake) |
| **Auth posture** | API keys, GitHub token, KMS | Anonymous public read |
| **Secrets** | Many (`COORDINATOR_API_KEY`, `GITHUB_WEBHOOK_SECRET`, KMS) | None |
| **Caps** | Full `IO`/`Net` | `--caps FS` only |
| **Blast radius if it breaks** | Coordinator down = no eval runs | Docs MCP down = agents fall back to scraping |

Cloud Run services are free at idle (min-instances 0). The benefits — independent rollback, no shared-secret blast radius, clean "this image == AILANG vX.Y.Z snapshot" semantics, and a simpler Terraform diff per release — outweigh the marginal cost of one more `google_cloud_run_v2_service` resource.

### Caps & Security

- `--caps FS,Net` — `FS` for snapshot reads, `Net` exclusively for the `submit_feedback` tool to publish to Pub/Sub
- VPC egress: locked to `*.googleapis.com` only (Cloud Run egress restriction). The MCP service can't reach arbitrary internet endpoints even if a tool tried to.
- `--routes-only` — only `@mcp_name` exports register; nothing else leaks even if a helper is exported
- **No authentication** for read tools (this is public docs data). Per-IP rate limit at the LB applies to all tools, with stricter quotas for `submit_feedback`.
- Service account has exactly two GCP permissions: `roles/pubsub.publisher` on the feedback topic, and storage read on the snapshot bucket (if the snapshot is mounted from GCS instead of baked in)
- Snapshot is **immutable per image** — each release rebuilds + redeploys. Read-tool replies are deterministic for a given image SHA.

### Snapshot Build (CI) — Additive Across Versions

A new `make snapshot` target — runs in the existing release pipeline ([release-manager](../../../.claude/skills/release-manager/SKILL.md)). The snapshot is **additive**: each release adds a new `versioned/X.Y.Z/` directory and updates the `latest` symlink. Old version directories are never overwritten or removed.

For release `vX.Y.Z`:
1. Create `build/snapshot/versioned/X.Y.Z/` from the current working tree's `stdlib/`, `examples/` (filtered to those passing on this version per `examples_report.json`), `prompts/vX.Y.Z*.md`, `docs/docs/`, plus a generated `effects.json` and `limitations.json`
2. Append/refresh `build/snapshot/unscoped/`: rebuild `benchmarks.sqlite` from all of `eval_results/baselines/` (one row per run, indexed by model + ailang_version + benchmark); copy `design_docs/` and `CHANGELOG.md`; regenerate `roadmap.json`
3. Rebuild `docs.sqlite` FTS5 with a `version` column so `docs_search` can filter by `for_version`
4. Bake the entire `build/snapshot/` (all historical versions + unscoped data) into the Cloud Run image alongside the **latest** `ailang` binary

**No backfill on first launch.** The initial deploy ships with snapshots only for the current AILANG release. Older versions get added organically — every future release just appends one `versioned/X.Y.Z/` directory. CLIs on pre-launch versions will receive `{error: "unknown_version"}` from `for_version`-scoped tools, and (per M5 design) silently fall back to their embedded prompts. That's the correct behavior: the embedded copy was canonical when their binary was built; not having a fresh re-render available simply means no improvement to apply yet.

Why one image with all versions, not one image per version + Cloud Run traffic splitting:
- Snapshots are tiny (~MB per version, mostly Markdown + small SQLite)
- Single image = single deploy, single rollback, single URL — agents/CLIs never have to pick the right server
- Version negotiation happens **inside** tool calls, where it belongs (the data layer), not at the network edge

If the multi-version image ever grows uncomfortable (>500 MB), we can prune `versioned/X.Y.Z/` for end-of-life releases — but only by formal deprecation policy, not casually.

---

## Implementation Plan (7 days)

### M1 — Tool Skeleton in AILANG (1.5 days)

- New directory `mcp-tools/` (top of repo) — one `.ail` file per tool group:
  `language.ail`, `examples.ail`, `design.ail`, `benchmarks.ail`, `discovery.ail`
- Each function uses `@mcp_name("name")` and `@route` (so it also works via `--routes-only` plain HTTP)
- Read-only — `import std/file` for snapshot reads, `import std/sqlite` for the two indices
- Add `make snapshot` (writes `build/snapshot/` from repo state)
- Add `make mcp-local` (runs `ailang serve-api --mcp-http --routes-only ./mcp-tools/`)

**Acceptance**: `ailang serve-api --mcp-http ./mcp-tools/` boots locally; `curl POST /mcp/ tools/list` returns ≥20 tools with valid schemas.

### M2 — Snapshot Builders (1 day)

- `tools/build-benchmarks-sqlite/` (Go, reuses `internal/eval_harness/`) — walks `eval_results/baselines/`, writes `benchmarks.sqlite` with columns: `run_id, benchmark, language, model, version, pass, cost_usd, input_tokens, output_tokens, error_category, timestamp`
- `tools/build-docs-sqlite/` — walks `docs/docs/*.md`, writes FTS5 table `(path, title, body)` plus `docs_nav` table from sidebar config
- Wire into `make snapshot`

**Acceptance**: `make snapshot` produces a deterministic `build/snapshot/` (modulo timestamps); SQLite files are <50 MB.

### M3 — Cloud Run Image + Deploy (1 day)

- New `Dockerfile.mcp` in `ailang-multivac/docker/` — multi-stage: build `ailang` from pinned tag, copy snapshot, set entrypoint to `ailang serve-api --mcp-http --port 8080 --routes-only --caps FS /srv/mcp-tools/`
- New Cloud Run service in Terraform (`ailang-multivac/terraform/`) — min-instances 0, max 10, CPU 1, mem 512Mi, public ingress
- Cloud Build trigger: rebuild on `mcp-tools/**` or `make snapshot` output change
- Custom domain `mcp.ailang.sunholo.com` via Cloud Run domain mapping

**Acceptance**: `curl https://mcp.ailang.sunholo.com/mcp/ -X POST -d '{"method":"tools/list"}'` returns the tool catalog.

### M4 — Discovery from the Docs Site (0.5 day)

- Add `<link rel="mcp" href="https://mcp.ailang.sunholo.com/mcp/">` to `docs/src/theme/Layout`
- Add an "Add to Claude / Cursor / Cline" component on the landing page (3 deeplink buttons using each harness's add-MCP URL scheme)
- Update `llms.txt` header to mention the MCP endpoint as the preferred path for structured queries
- New doc `docs/docs/guides/agent-mcp.md` — how agents connect, available tools, examples

**Acceptance**: Landing page renders the buttons; clicking "Add to Claude Desktop" opens Claude with the MCP server pre-filled.

### M5 — CLI Falls Back to MCP for Fresh Content (1 day)

This is what turns the MCP from "agent helper" into "canonical source of truth".

- New flag `ailang prompt --fresh` (and `--source mcp|embedded|auto`, default `auto`)
- **Version-locked fetch**: every MCP call from the CLI passes `for_version=$AILANG_VERSION` (the binary's compile-time version). The CLI never receives content tagged for a different version.
- `auto` behavior, in order:
  1. Call `prompt_get(for_version=$AILANG_VERSION, kind=...)`
  2. If response `served_for == $AILANG_VERSION` AND its `sha` differs from the embedded copy → use fresh, log "fresher prompt available for your version (sha abc123)" to stderr
  3. If response `served_for != $AILANG_VERSION` (server backfilled forward but not for this version) OR network failure OR timeout → use embedded silently
  4. Never fall through to a different version's content. Embedded for v0.10 beats fresh for v0.11.
- Same treatment for `ailang agent-prompt`, `ailang devtools-prompt`, and `ailang docs search`
- New `ailang mcp status` — shows: CLI version, embedded prompt sha, deployed prompt sha **for the CLI's version**, and whether the deployed server even knows about this CLI's version (warns clearly if not)
- HTTP timeout 1.5 s; cache the fresh copy under `~/.ailang/cache/prompts/$AILANG_VERSION/` keyed by `sha` so subsequent calls in the same hour are offline-fast
- Determinism guarantee: `--source embedded` is always available and is what reproducible eval runs MUST use; CHANGELOG calls this out

**Acceptance**: `ailang prompt --source mcp` returns content tagged `served_for == $AILANG_VERSION` or fails over cleanly to embedded; `ailang mcp status` correctly distinguishes "server has fresher content for your version" from "server has only newer versions you can't use"; offline runs still succeed via the embedded fallback; an end-to-end test installs the previous AILANG release, points it at a fresh MCP, and confirms it never receives v0.15-tagged content.

### M6 — Bootstrap Skill (and harness equivalents) Consume MCP (0.5 day)

- Update `ailang-bootstrap` skill (and any sister skills found by grepping for embedded AILANG guidance, e.g. `use-ailang`) so that the SKILL.md no longer carries lengthy version-pinned guidance; instead it instructs the agent to add `mcp.ailang.sunholo.com` as a remote MCP and call the relevant introspection tools
- Add a one-time migration note to `CHANGELOG.md` and the skill READMEs describing the swap from "embedded knowledge" to "live MCP queries"
- Smoke test: a Claude Code session bootstrapped against the new skill on an empty repo can answer "what stdlib function reads a file?" purely via MCP, without any locally embedded reference

**Acceptance**: Skill files shrink by ≥50%; bootstrapped agent answers a stdlib lookup question correctly using only MCP tool calls (verified by trace inspection).

### M7 — Public Feedback Channel (`submit_feedback`) (0.5 day)

The only write tool. Anonymous, rate-limited, **reuses the existing `ailang-messages` Pub/Sub pipeline** — no new topic, no new Firestore collection, no drainer.

**Existing infra we're reusing** (in [`internal/pubsub/`](../../../internal/pubsub/) + [ailang-multivac/terraform/pubsub.tf](../../../../ailang-multivac/terraform/pubsub.tf)):
- `ailang-messages` Pub/Sub topic with attribute-based routing (`inbox`, `workspace`, `from_agent`, `category`, `message_type`)
- Coordinator push subscription already drains it into Firestore (the `ailang messages` backing store)
- `pubsub.NewClient(ctx, projectID, prefix)` already uses Application Default Credentials — Cloud Run service accounts auth automatically

**What M7 actually adds**:
- New `mcp-tools/feedback.ail` with `@mcp_name("submit_feedback") @route` function. Validates inputs (≤10 KB body, ≤4 KB snippet, category enum, required fields) and returns structured `{error, field}` on bad input.
- The publish step calls a thin builtin/helper that wraps `internal/pubsub.Client.Topic("messages").Publish(...)` with attributes `{inbox: "public-feedback", from_agent: "mcp-public", category: "<from-input>", message_type: "feedback"}` — this is the only Go change needed (we either expose the existing pubsub client as a builtin to AILANG, or register `submit_feedback` as a Go-side MCP tool alongside the AILANG ones)
- Terraform delta: **just one IAM binding** — MCP Cloud Run service account gets `roles/pubsub.publisher` on the existing `ailang-messages` topic. No new topic, no new subscription, no new Firestore collection.
- LB-level per-IP rate limit: 5/min, 50/hour, 200/day for `submit_feedback`; generic 60/min for read tools.
- Daily quota guard returns `{error: "rate_limit_daily", retry_after}`, never silent drop.

**Result**: feedback shows up in `ailang messages list --inbox public-feedback` directly. No daily drain job, no second moderation queue — the existing inbox IS the moderation queue.

**Acceptance**: a valid `submit_feedback` call returns `{ticket_id, queued_at}` and the message appears in `ailang messages list --inbox public-feedback` within 5 s (via the existing coordinator push subscription); a spam test (100 requests from one IP) returns 429s after the per-minute threshold; oversized body returns a structured field-level error; the only new GCP resource provisioned is the IAM binding (verifiable in `terraform plan`).

### M8 — Eval Harness Integration + CI (1 day)

- Add `mcp-tools/` to `make verify-examples` (must compile + lint)
- Add an end-to-end test: spin up the local server, call each tool once via the official `mcp` Go SDK, assert no errors and non-empty results
- Add a benchmark: agent run with vs without the MCP server attached on a fixed task ("write a stdlib function that uses effects") — measures whether the MCP tools actually help
- Update CHANGELOG, move design doc to `implemented/v0_15_x/`

**Acceptance**: `make ci` passes; benchmark shows ≥10% pass-rate lift OR ≥30% token reduction with MCP attached.

---

## Risks & Tradeoffs

1. **Snapshot staleness** — The MCP server only knows what was bundled at release time. Mitigation: pair every release with a redeploy (already part of `release-manager`); doc-only changes (e.g. a typo fix in a guide) can trigger a snapshot-only rebuild without a full release. Out of scope: live querying GitHub.
1.1. **Determinism vs freshness conflict** — `ailang prompt --source auto` defaults to fresh, but reproducible eval runs need pinned content. Mitigation: M5 documents that `eval_results/` runs MUST set `--source embedded`; CI gates the eval harness on this. (Even `--source mcp` is technically reproducible against an immutable image SHA, but `embedded` requires no network and removes a moving part.)
1.2. **Cross-version content leak** — fresh content for a newer language version reaches a CLI that can't use it. Mitigation: enforced at protocol level — every CLI MCP call passes `for_version`; the response is tagged `served_for`; the CLI hard-rejects any response where `served_for != $AILANG_VERSION` and falls back to embedded silently. Backed by a regression test that installs vN-1 against an MCP serving up to vN.
2. **Cost** — Cloud Run min-instances 0 + low traffic = pennies/month; the only real cost is build minutes. No risk.
3. **MCP protocol churn** — The Go SDK we use (`modelcontextprotocol/go-sdk`) is already pinned in `go.mod`; we own the upgrade cadence.
4. **Discoverability** — A docs-site link is passive. Most agents will find it via the user manually adding it. This is acceptable for v1; an `.well-known/mcp.json` discovery file is a v2 candidate once the spec stabilizes.
5. **Dogfooding pressure** — Writing the introspection logic in AILANG forces us to confirm the language is fit for this kind of "thin-server-over-data" workload. If an AILANG limitation blocks a tool, that's signal for the language roadmap, not a reason to fall back to Go. This is the design intent of A7 + A10.

## Out of Scope (for v1)

- **Authentication & paid tier** — see "Future: Paid Tier" below
- Write tools that mutate user-visible state directly (`submit_feedback` is the **only** write tool, and it writes only to a moderation queue — never directly to issues, docs, or DBs)
- Live editing of AILANG code via MCP (the existing `internal/executor/` is the right place for that, not the docs MCP)
- A separate `mcp.ailang.sunholo.com/playground/` REPL (could be a v2; users can already use the local `ailang repl`)
- Agent-Agent (A2A) protocol — orthogonal; the apiserver already supports `--a2a` if/when needed

## Future: Paid Tier (Not v1, but Don't Block It)

A paid/authenticated tier is on the roadmap but explicitly out of scope for this sprint. The v1 design must avoid choices that close that door.

**What we know we want eventually**:
- Login-gated tools (e.g. higher rate limits, per-user usage history, project-scoped queries, write/feedback tools, neural search if we ever ship an embedding service)
- Per-user telemetry on which tools are actually called (read-only logs are fine for v1; per-user attribution needs auth)

**What v1 must NOT do that would block this**:
- Don't design tool names that assume "everything is public" (e.g. avoid `public_stdlib_modules` — just `stdlib_modules`)
- Don't bake the unauth posture into Cloud Run config in a way that's hard to layer auth on (use IAP-friendly Cloud Run settings)
- Don't ship a custom URL convention that conflicts with `/auth/*` later

**What's already free**:
- The `apiserver` supports `--api-key-header` / `--api-key-env` flags ([serve_api.go:272](../../../cmd/ailang/serve_api.go#L272)). Adding auth later = setting two env vars on a separate Cloud Run service.
- The `@noexpose` annotation already lets us mark tools as private. A future paid tier is "spin up a second Cloud Run service with auth flags + a superset of `mcp-tools/` including the gated `.ail` files". No protocol redesign required.
- The CLI's `--source auto` logic in M5 already supports a configured endpoint — adding `AILANG_MCP_TOKEN` is a small extension, not a rewrite.

So v1 ships fully public, fully anonymous, with clean naming and no architectural lock-in. When the paid tier is greenlit, it gets its own design doc and its own Cloud Run service (e.g. `mcp-pro.ailang.sunholo.com`); the public service keeps running unchanged.

## Open Questions

1. **`mcp.ailang.sunholo.com` vs `ailang.sunholo.com/mcp/`** — Subdomain is cleaner for Cloud Run + custom domain mapping; same-origin is friendlier to browser-side MCP clients (none exist yet, but might). Recommend subdomain for v1.
2. **Should `docs_search` proxy `ailang docs search` (which already does neural search) or use FTS5?** FTS5 has zero deploy footprint; neural needs an embedding service. Recommend FTS5 for v1, swap to neural when we have an embedding service deployed anyway.
3. **Tool versioning** (the *protocol-level* tool surface, not content) — When a tool's signature changes, do we keep the old name? MCP doesn't have native versioning. Recommend "additive only" rule: never remove or change an existing tool, only add new ones. (Distinct from content versioning, which is handled in-tool via `for_version`.)
4. **End-of-life policy for old `versioned/X.Y.Z/` snapshots** — Keep forever, prune at 18 months past last download, or prune at next major (v1.0)? Recommend "keep forever until proven painful" — snapshots are small, removing them silently breaks CLIs in the wild.

---

## Success Metrics

- **Adoption**: ≥50 unique IPs/week within 30 days of launch (Cloud Run logs)
- **Quality lift**: ≥10% pass-rate improvement OR ≥30% token reduction in M8 benchmark
- **Feedback signal**: ≥1 actionable item/week landing in `ailang messages list --inbox public-feedback` within 60 days of launch (proves the channel works, not vanity volume)
- **Maintenance cost**: ≤30 min/release to refresh the snapshot (target: zero — fully automated)
- **Tool authoring cost**: New tool = 1 AILANG file, ≤30 LoC, no Go change
- **Staleness elimination**: documentation/prompt fixes reach `ailang prompt --source auto` users within 24 h of merge to `dev` (vs. "next time they reinstall the CLI" today)
- **Skill simplification**: `ailang-bootstrap` SKILL.md (and equivalents) shrink by ≥50% LoC after M6

---

## References

- [internal/apiserver/mcp.go](../../../internal/apiserver/mcp.go) — MCP server framework (already built)
- [cmd/ailang/serve_api.go](../../../cmd/ailang/serve_api.go) — `--mcp-http` entrypoint
- [docs/docs/guides/serve-api.md](../../../docs/docs/guides/serve-api.md) — annotation reference
- [.claude/rules/api-server.md](../../../.claude/rules/api-server.md) — annotation sync rules
- [ailang-multivac/](../../../../ailang-multivac/) — existing Cloud Run infrastructure
- [MCP spec — Streamable HTTP transport](https://modelcontextprotocol.io/specification/transports#streamable-http)
