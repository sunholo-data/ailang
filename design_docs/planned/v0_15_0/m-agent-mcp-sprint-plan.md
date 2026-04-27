# Sprint Plan: M-AGENT-MCP (Remote MCP Server for ailang.sunholo.com)

## Summary

Build, deploy, and integrate a Cloud Run-hosted MCP server (`mcp.ailang.sunholo.com`) that exposes versioned AILANG documentation/stdlib/examples/benchmarks as typed MCP tools. Includes CLI fresh-fetch fallback (M5), bootstrap skill simplification (M6), and a single anonymous write tool for public feedback (M7).

**Duration:** 8 days (7 days of milestone work + 1 day integration/buffer)
**Dependencies:** `internal/apiserver/mcp.go` (in-tree, ready), `ailang-multivac/` Cloud Run + Terraform pipeline
**Risk Level:** Medium — mostly composition of existing infrastructure, but introduces new GCP resources (Cloud Run service, Pub/Sub topic, custom domain) and new dual repo workflow (ailang + ailang-multivac)

**Design doc:** [m-agent-mcp-website.md](m-agent-mcp-website.md)

---

## Current Status Analysis

### Completed Recently (last 14 days)
- ✅ M-EXEC-PI (6 milestones, ~1100 LOC): pi executor, NDJSON parser, dockerfile, dispatcher, dashboard wiring
- ✅ post-release v0.14.2: dashboard data, axiom scorecard, gitignore intro
- ✅ M-EVAL-CROSS-HARNESS: adjusted-rate dashboards, matched-models comparison

### Velocity
- Recent average: ~150–200 LOC/day on milestone work, with reliable per-milestone checkpoints
- M-EXEC-PI shipped 6 milestones in a comparable timeframe — the sprint structure scales
- Estimated capacity for this sprint: ~1200–1500 LOC across 8 days

### Remaining from Design Doc
All 8 milestones are scoped here. Nothing is pre-built; the design doc is the full spec.

---

## Proposed Milestones

### M1 — AILANG Tool Skeleton in `mcp-tools/` (1.5 days)

**Goal:** Author the ~21 AILANG functions that introspect the AILANG corpus, each annotated `@mcp_name`/`@route`. Server boots locally and exposes the tools via `--mcp-http`.

**Estimated:** ~400 LOC AILANG (mostly thin slice-and-filter over snapshot data) + ~150 LOC Go (snapshot reader helpers in `internal/apiserver/` if needed) + ~100 LOC tests

**Tasks:**
- Day 1 (morning): Create `mcp-tools/` directory with one `.ail` file per group: `language.ail`, `examples.ail`, `design.ail`, `benchmarks.ail`, `discovery.ail`, `feedback.ail` (placeholder, fleshed out in M7)
- Day 1 (afternoon): Implement language.ail (`ailang_versions`, `prompt_get`, `stdlib_modules`, `stdlib_module`, `stdlib_search`, `limitations_list`, `effects_catalog`) reading from `/srv/snapshot/versioned/$VER/`
- Day 2 (morning): Implement examples.ail + design.ail + discovery.ail (`docs_nav`, `docs_search` against `docs.sqlite`)
- Day 2 (afternoon): `make snapshot` target writes a minimal `build/snapshot/` from current repo state; `make mcp-local` runs `ailang serve-api --mcp-http --routes-only --caps FS,Net ./mcp-tools/`

**Acceptance Criteria:**
- [ ] `ailang serve-api --mcp-http --routes-only ./mcp-tools/` boots without error
- [ ] `curl -X POST http://localhost:8080/mcp/ -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'` returns ≥20 tools with valid schemas
- [ ] Every `@mcp_name` passes `validateMCPName` regex
- [ ] `make verify-examples` passes for `mcp-tools/`
- [ ] Each tool returns `{served_for, ...}` for version-scoped reads

**Risks:**
- AILANG limitations may surface (e.g. SQLite query patterns not supported by `import std/sqlite`) — Mitigation: file an issue for each gap; fall back to a thin Go helper in `internal/apiserver/` for that one query if blocking
- Tool name collisions across modules — Mitigation: use `@mcp_name` overrides; the framework already has dedup logic (mcp_schema.go)

---

### M2 — Snapshot Builders (1 day)

**Goal:** Reproducible `make snapshot` that produces `build/snapshot/` (multi-version layout from design doc).

**Estimated:** ~300 LOC Go (two builders) + ~50 LOC Makefile/shell + ~80 LOC tests

**Tasks:**
- Day 3 (morning): `tools/build-benchmarks-sqlite/` walks `eval_results/baselines/`, writes `benchmarks.sqlite` with the columns specified in the design doc (run_id, benchmark, language, model, version, pass, cost_usd, input_tokens, output_tokens, error_category, timestamp)
- Day 3 (midday): `tools/build-docs-sqlite/` walks `docs/docs/*.md`, writes FTS5 `(version, path, title, body)` plus a `docs_nav` table from sidebar config
- Day 3 (afternoon): Wire both into `make snapshot`; add a determinism check (run twice, diff should be empty modulo `built_at` timestamp); add the additive layout (`versioned/$VER/`, `unscoped/`, `latest` symlink)

**Acceptance Criteria:**
- [ ] `make snapshot` produces `build/snapshot/` deterministically (two runs → byte-identical except `built_at`)
- [ ] `benchmarks.sqlite` has all baseline runs from `eval_results/baselines/` with correct schema
- [ ] `docs.sqlite` FTS5 index returns ≥1 hit for "effects" query
- [ ] Total snapshot size <50 MB for current repo state
- [ ] `versioned/$VER/` populated for current release; `latest` symlink points to it

**Risks:**
- `eval_results/baselines/` JSON schema variance across versions — Mitigation: validate each row against expected fields; skip+log malformed rows rather than fail the build

---

### M3 — Cloud Run Image + Deploy (1 day)

**Goal:** `mcp.ailang.sunholo.com` resolves to a Cloud Run service serving the MCP over Streamable HTTP.

**Estimated:** ~150 LOC Dockerfile/Terraform + ~100 LOC Cloud Build config

**Tasks:**
- Day 4 (morning): `ailang-multivac/docker/Dockerfile.mcp` — multi-stage build (build `ailang` from pinned tag, copy `build/snapshot/`, set entrypoint to `ailang serve-api --mcp-http --port 8080 --routes-only --caps FS,Net /srv/mcp-tools/`)
- Day 4 (midday): `ailang-multivac/terraform/cloud_run_mcp.tf` — new `google_cloud_run_v2_service` (min-instances 0, max 10, CPU 1, mem 512Mi, public ingress, VPC egress restricted to `*.googleapis.com`); service account with no secrets (Pub/Sub publisher role added in M7)
- Day 4 (afternoon): Cloud Build trigger on `mcp-tools/**` or snapshot change; Cloud Run domain mapping for `mcp.ailang.sunholo.com`; verify deploy + `tools/list` round-trip

**Acceptance Criteria:**
- [ ] `curl -X POST https://mcp.ailang.sunholo.com/mcp/ -d '{"jsonrpc":"2.0","method":"tools/list","id":1}'` returns the tool catalog
- [ ] Cloud Run service shows healthy with min-instances 0 (cold-start <3 s)
- [ ] Terraform `terraform plan` is clean (no drift) after deploy
- [ ] Custom domain SSL cert is provisioned and serving
- [ ] Cloud Build trigger fires on a test commit to `mcp-tools/`

**Risks:**
- Custom domain DNS propagation can take hours — Mitigation: provision the cert at the start of M4; in the meantime acceptance can use the `*.run.app` URL
- Cloud Run cold start with snapshot mounted from image (vs GCS) — Mitigation: image-baked is fine for v1 (snapshot is small); GCS-mount is a v2 optimization

---

### M4 — Discovery from the Docs Site (0.5 day)

**Goal:** A user visiting ailang.sunholo.com can add the MCP server to their agent in one click.

**Estimated:** ~100 LOC React component + ~80 LOC docs/markdown

**Tasks:**
- Day 5 (morning, ½): Add `<link rel="mcp" href="https://mcp.ailang.sunholo.com/mcp/">` to `docs/src/theme/Layout`; create `<AddToAgent>` component on landing page with three deeplink buttons (Claude Desktop, Cursor, Cline); update `llms.txt` header with MCP endpoint reference; new doc `docs/docs/guides/agent-mcp.md`

**Acceptance Criteria:**
- [ ] Landing page renders the three "Add to ___" buttons
- [ ] Clicking "Add to Claude Desktop" opens Claude with the MCP server pre-filled (verified manually)
- [ ] `docs/docs/guides/agent-mcp.md` documents the available tools and connection flow
- [ ] `llms.txt` mentions MCP as the preferred path for structured queries
- [ ] Docusaurus build passes

**Risks:**
- Each harness's add-MCP URL scheme may differ or change — Mitigation: link to the harness's docs as fallback if the deeplink fails

---

### M5 — CLI Falls Back to MCP for Fresh Content (1 day)

**Goal:** `ailang prompt --source auto` prefers fresh, version-locked content from the deployed MCP, with bulletproof fallback to embedded.

**Estimated:** ~200 LOC Go in `cmd/ailang/` + `internal/mcp_client/` + ~100 LOC tests including the cross-version regression test

**Tasks:**
- Day 5 (afternoon, ½): New `internal/mcp_client/` package — minimal MCP client (just `tools/call` over Streamable HTTP) with 1.5s timeout, `for_version` always passed, `served_for` always validated
- Day 6 (morning): Wire into `ailang prompt --source mcp|embedded|auto` (default `auto`), `ailang agent-prompt`, `ailang devtools-prompt`, `ailang docs search`; cache under `~/.ailang/cache/prompts/$AILANG_VERSION/` keyed by sha
- Day 6 (midday): New `ailang mcp status` command — shows CLI version, embedded sha, deployed sha for CLI's version, drift state
- Day 6 (afternoon): Cross-version regression test (install previous AILANG release, point at fresh MCP, confirm no v0.15-tagged content received); offline test (network disabled → embedded fallback succeeds silently)

**Acceptance Criteria:**
- [ ] `ailang prompt --source mcp` returns content where `served_for == $AILANG_VERSION` or fails over to embedded
- [ ] `ailang prompt --source embedded` works with no network (eval reproducibility)
- [ ] `ailang prompt --source auto` (default) returns embedded if MCP unreachable, with no error to stderr
- [ ] `ailang mcp status` distinguishes "fresher for your version" vs "only newer versions available"
- [ ] Cross-version regression test passes — vN-1 CLI never receives vN content
- [ ] Cache hit on second call within an hour (no network call)

**Risks:**
- Network flakiness in CI — Mitigation: tests use a local mock MCP server, not the real one
- Embedded vs fresh sha mismatch on first deploy (server has no content for current dev version) — Mitigation: `auto` falls through to embedded silently; this is the designed behavior

---

### M6 — Bootstrap Skill Consumes MCP (0.5 day)

**Goal:** The `ailang-bootstrap` skill (and any sister skills carrying embedded AILANG guidance) become thin shims that point agents at the MCP.

**Estimated:** ~100 LOC skill markdown rewrites + ~50 LOC migration notes

**Tasks:**
- Day 7 (morning, ½): Audit candidate skills (grep `~/.claude/skills/` for AILANG version refs / stdlib signatures / prompt content); rewrite `ailang-bootstrap` SKILL.md to instruct adding `mcp.ailang.sunholo.com` and listing the read tools to call; same treatment for any sister skills found; CHANGELOG migration note describing the swap

**Acceptance Criteria:**
- [ ] `ailang-bootstrap` SKILL.md shrinks ≥50% LoC
- [ ] Smoke test: a fresh Claude Code session bootstrapped via the new skill on an empty repo answers "what stdlib function reads a file?" using only MCP tool calls (verified by trace inspection of which tools were called)
- [ ] Migration note added to skill READMEs and `CHANGELOG.md`
- [ ] No accidental regression in skill description triggers (skill still loads on the same prompts)

**Risks:**
- The `ailang-bootstrap` skill may not exist in this repo (it's a user-side skill) — Mitigation: locate via `find ~/.claude -name SKILL.md | xargs grep -l ailang-bootstrap`; if not present, document the pattern in the new `agent-mcp.md` guide so external skill authors can adopt it

---

### M7 — Public Feedback Channel `submit_feedback` (0.5 day)

**Goal:** A single anonymous, rate-limited write tool. **Reuses the existing `ailang-messages` Pub/Sub topology** — no new topic, no new Firestore collection, no drainer. Feedback lands directly in `ailang messages list --inbox public-feedback` via the coordinator's existing push subscription.

**Reused infra** (already shipping): `internal/pubsub/client.go` publishes to topic `ailang-messages` via Application Default Credentials; coordinator push subscription routes by attribute (`inbox`, `from_agent`, `category`, `message_type`); Firestore-backed inbox is the existing moderation surface.

**Estimated:** ~80 LOC AILANG (`mcp-tools/feedback.ail` validation + envelope construction) + ~60 LOC Go (thin pubsub publish helper exposed to AILANG, OR `submit_feedback` as a Go-side MCP tool) + ~30 LOC Terraform (one IAM binding) + ~80 LOC tests = ~250 LOC total

**Tasks:**
- Day 7 (afternoon, ½): `mcp-tools/feedback.ail` — `@mcp_name("submit_feedback") @route` function. Validate inputs (≤10 KB body, ≤4 KB snippet, required fields, category enum). Construct message envelope. Call publish helper.
- Day 8 (morning, ½): Wire publish path. Two options, pick whichever is shorter to land:
  - **Option A** (preferred if a builtin slot is cheap): expose `internal/pubsub.Client.Publish` as a thin builtin `_pubsub_publish_message(topic, attrs, body)`; `feedback.ail` calls it
  - **Option B**: register `submit_feedback` as a Go-side MCP tool alongside the AILANG ones (apiserver supports this); validation logic moves to Go
  - Either way: attributes are `{inbox: "public-feedback", from_agent: "mcp-public", category: <input>, message_type: "feedback"}`
- Day 8 (midday, ½): Terraform delta = single `google_pubsub_topic_iam_member` granting MCP Cloud Run SA `roles/pubsub.publisher` on the existing `ailang-messages` topic. LB-level rate limit (5/min, 50/hour, 200/day for `submit_feedback` path; 60/min generic). Spam regression test.

**Acceptance Criteria:**
- [ ] Valid `submit_feedback` call returns `{ticket_id, queued_at}` and the item appears in `ailang messages list --inbox public-feedback` within 5 s (via existing coordinator push subscription)
- [ ] Spam test: 100 requests from one IP returns 429s after the per-minute threshold
- [ ] Oversized body returns structured `{error: "body_too_large", field: "body"}`, not generic 400
- [ ] Daily quota overflow returns `{error: "rate_limit_daily", retry_after}`, never silent drop
- [ ] `terraform plan` shows exactly one new GCP resource (the IAM binding) — no new topic, no new subscription, no new Firestore collection

**Risks:**
- Builtin/Go-tool decision (Option A vs B) may bikeshed — Mitigation: timebox to 30 minutes; if undecided, ship Option B (smaller surface, no new builtin to support). The user-visible MCP tool is identical either way.
- Existing `ailang-messages` topic schema vs feedback message shape — Mitigation: feedback message conforms to the existing envelope; we're not extending it. Verify by re-reading `internal/messages/store.go` or equivalent before implementing.
- LB-level rate limit may need Cloud Armor — Mitigation: start with Cloud Run's built-in concurrency cap + middleware-side per-IP limit; upgrade to Cloud Armor only if abuse appears

---

### M8 — Eval Harness Integration + CI + Release (1 day)

**Goal:** Sprint is verifiable in CI and the design doc moves to `implemented/`.

**Estimated:** ~150 LOC test code + ~100 LOC benchmark + CHANGELOG/docs updates

**Tasks:**
- Day 8 (afternoon, ½): Add `mcp-tools/` to `make verify-examples` (must compile + lint); end-to-end test spins up local server, calls each tool once via the official `modelcontextprotocol/go-sdk`, asserts no errors and non-empty results
- Day 9 (morning): Benchmark — fixed agent task ("write a stdlib function that uses effects") run with vs without MCP attached; capture pass-rate and token-count delta; surface in `benchmarks_for_model` for self-comparison
- Day 9 (afternoon): Update `CHANGELOG.md` with the M-AGENT-MCP entry; run `make ci`; move `m-agent-mcp-website.md` and this sprint plan to `design_docs/implemented/v0_15_x/`; update `roadmap.json`

**Acceptance Criteria:**
- [ ] `make ci` passes (build, test, lint, verify-examples, file-size check)
- [ ] End-to-end test calls every registered MCP tool successfully
- [ ] Benchmark shows ≥10% pass-rate lift OR ≥30% token reduction with MCP attached
- [ ] `CHANGELOG.md` entry references the design doc and sprint plan
- [ ] Both docs landed in `design_docs/implemented/v0_15_x/`
- [ ] `mcp.ailang.sunholo.com` is live and serving

---

## Success Metrics (sprint-level)

- **Test coverage**: All new Go code in `internal/mcp_client/` and `tools/build-*/` ≥70%
- **Examples**: `mcp-tools/*.ail` files pass `make verify-examples`
- **Documentation updated**:
  - [ ] `docs/docs/guides/agent-mcp.md` (new)
  - [ ] `docs/src/theme/Layout` (MCP discovery link)
  - [ ] `llms.txt` (MCP endpoint reference)
  - [ ] `CHANGELOG.md` (M-AGENT-MCP entry)
  - [ ] Design doc + sprint plan moved to `implemented/v0_15_x/`
- **Tests passing**: `make ci` green
- **Linting clean**: `golangci-lint` no new warnings

## Dependencies

- **Existing infra (already shipping)**:
  - `internal/apiserver/mcp.go` — MCP server framework with `HTTPHandler()` for Streamable HTTP
  - `cmd/ailang/serve_api.go` — `--mcp-http` entrypoint
  - `ailang-multivac/` — Terraform + Cloud Build pipeline
- **External (must coordinate)**:
  - `mcp.ailang.sunholo.com` DNS / GCP project access for Cloud Run domain mapping
  - GCP IAM permission to create new Cloud Run service + Pub/Sub topic + Firestore collection in the AILANG project

## Open Questions (to resolve during execution, not blockers)

1. **Workload Identity vs Cloud Run service account JSON key for Pub/Sub publish from inside AILANG?** Recommend Workload Identity; if the REST call from `std/net` is awkward, fall back to a thin Go shim (M7 risk).
2. **Skill audit scope** (M6) — Just `ailang-bootstrap`, or also `use-ailang` and any others embedding AILANG knowledge? Recommend audit-first, edit-second.
3. ~~Backfill of historical `versioned/X.Y.Z/` snapshots~~ — **Resolved: no backfill.** First launch ships current-version snapshot only; older versions populate organically as releases happen. Pre-launch CLIs receive `unknown_version` and silently fall back to embedded — designed behavior.

## Notes

- M1–M4 are sequential (M2 depends on M1, M3 depends on M2, M4 depends on M3)
- M5, M6, M7 can in principle run in parallel after M3 — sprint-executor should run them sequentially in the order above to keep checkpoints clean
- M8 must come last (verifies everything)
- This sprint introduces a **new repo dependency**: `ailang-multivac/`. Sprint-executor must be ready to commit to both repos with appropriate branches
- The `ailang-bootstrap` skill referenced in M6 may live outside this repo (`~/.claude/skills/`) — M6 may produce zero changes here and a recommendation to update the skill manually
