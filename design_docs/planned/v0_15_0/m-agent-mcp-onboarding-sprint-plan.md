# Sprint Plan: M-AGENT-MCP-ONBOARDING (Fix broken indexes + install/onboarding tools)

## Summary

Fix three broken MCP tools that currently return `snapshot_read_failed` in prod (`docs_search`, `example_for_concept`, `stdlib_search`) and add two new tools agents need most: `install_guide(harness, platform)` and `onboarding_guide(role)`. Bonus: fix the `:latest` SHA Cloud Run quirk that forced manual `gcloud run services update` after every prod deploy in M-AGENT-MCP.

**Duration:** 2 days (1.5 days planned + 0.5 day buffer for the deploy chain)
**Dependencies:** M-AGENT-MCP shipped to prod (all 21 tools live; we're extending the existing infrastructure)
**Risk Level:** Low — pure additions to a well-rehearsed path; no new GCP resources, no new caps, no IAM changes

**Design doc:** [m-agent-mcp-onboarding.md](m-agent-mcp-onboarding.md)

---

## Current Status Analysis

### Completed Recently (M-AGENT-MCP)
- ✅ M1–M8 + Json fix + M7.1 publish wire all live in prod
- ✅ `https://mcp.ailang.sunholo.com/mcp/` serving 21 tools (3 of which are broken — fixed in this sprint)
- ✅ `submit_feedback` writing real messages to Firestore (verified end-to-end)
- ✅ Claude Code + Claude Desktop both wired to the MCP
- ✅ `ailang_bootstrap` all 3 branches (dev/preview/stable) synced

### Velocity (last session)
- M-AGENT-MCP: ~2660 LOC across 8 milestones, ~6 hours of focused execution
- The dev → test → prod deploy dance is now well-rehearsed (~25 min per env)
- Forces redeploys via `gcloud run services update --image=:latest` are needed twice per sprint (per env) — this sprint adds the automation

### Remaining from Design Doc
All 4 milestones are scoped here. No prior partial work.

---

## Proposed Milestones

### M1 — Fix 3 broken indexes + `:latest` SHA automation (0.5 day)

**Goal:** All 21 currently-advertised MCP tools return real data in prod, and future deploys don't need manual `gcloud run services update`.

**Estimated:** ~220 LOC Go (3 index generators) + ~40 LOC AILANG (substring matching in tool implementations) + ~30 LOC YAML (cloudbuild deploy step) + tests

**Tasks:**
- Day 1 morning: extend `tools/build-snapshot/main.go` with three new generators:
  - `writeDocsSearchIndex()` — walks `docs/docs/**/*.{md,mdx}`, strips frontmatter, extracts title + headings + body, writes `versioned/<ver>/docs_search_index.json` as `[{path, title, headings, body, body_lower}]` (lowercased body for case-insensitive matching)
  - `writeExampleConceptIndex()` — reads `examples/examples_report.json` (handle JSONL vs JSON ambiguity surfaced in M-AGENT-MCP M2) + parses each `.ail` file's leading `--` comment block for tags. Writes `[{path, title, concepts: [...], why}]`
  - `writeStdlibSearchIndex()` — extends existing stdlib_summary builder to also emit per-export `[{module, name, signature, doc, keywords}]` rows
- Day 1 midday: update `mcp_tools/discovery.ail` (docs_search) and `mcp_tools/language.ail` (stdlib_search) and `mcp_tools/examples.ail` (example_for_concept) — replace stub passthroughs with actual substring matching against the loaded indexes. Limit results to top-5 per query, weighted title > heading > body.
- Day 1 afternoon: add deploy automation. In `ailang-multivac/cloudbuild.yaml`, append a new `deploy-mcp-services` step after `terraform-apply` that runs `gcloud run services update <svc> --image=:latest` for `ailang-${prefix}-mcp` so the new `:latest` SHA is picked up automatically. (We hit this twice in M-AGENT-MCP — both prod deploys needed manual force-redeploy.)

**Acceptance Criteria:**
- [ ] `tools/call docs_search query="install" forVersion=""` returns ≥1 hit pointing at `getting-started.mdx`
- [ ] `tools/call example_for_concept concept="effects" forVersion=""` returns a real example path (not error envelope)
- [ ] `tools/call stdlib_search query="readFile" forVersion=""` returns `{module: "std/fs", name: "readFile", signature: "...", ...}`
- [ ] `make verify-mcp-tools` passes for all 7 mcp_tools/*.ail files
- [ ] After prod deploy, no manual `gcloud run services update` needed — service shows new image SHA automatically

**Risks:**
- `examples_report.json` format (JSONL vs JSON wrapped) was a footgun in M-AGENT-MCP M2 — Mitigation: defensive parser that handles both
- Substring match noise (e.g. `docs_search("install")` could hit any page mentioning "install") — Mitigation: top-5 limit + weighted scoring; user feedback via `submit_feedback` if signal-to-noise is bad

---

### M2 — `install_guide(harness, platform)` tool + index (0.5 day)

**Goal:** Agents can ask "how do I install AILANG via Claude Code?" and get the right `/plugin` commands without scraping the website.

**Estimated:** ~120 LOC Go (snapshot generator + hand-curated overrides JSON) + ~50 LOC AILANG (`mcp_tools/install.ail`) + 30 LOC tests

**Tasks:**
- Day 1 evening / Day 2 morning: create `tools/build-snapshot/install_guide_overrides.json` — hand-curated map of `harness → {commands, what_you_get, verify, post_install}` derived from `getting-started.mdx` + `ailang_bootstrap/README.md`. Covers 4 harnesses (claude-code, gemini-cli, codex-cli, manual) and platform variants for manual install
- Day 2 morning: extend `tools/build-snapshot/main.go` with `writeInstallGuideIndex()` — copies the overrides JSON into `unscoped/install_guide.json` (unscoped because install commands rarely change between AILANG releases, and when they do it's via plugin updates not language releases)
- Day 2 morning: new `mcp_tools/install.ail` with `@mcp_name("install_guide") @route` function that reads `unscoped/install_guide.json` and filters by harness/platform
- Day 2 morning: register in `make verify-mcp-tools`; add `make verify-install-guide` that diffs the overrides file's commands against the actual canonical sources (`getting-started.mdx` headings + `ailang_bootstrap/README.md` install sections) — fails CI on drift

**Acceptance Criteria:**
- [ ] `tools/call install_guide harness="claude-code" platform=""` returns the `/plugin marketplace add ...` commands and `what_you_get` list
- [ ] `tools/call install_guide harness=""` returns ALL harnesses
- [ ] `tools/call install_guide harness="manual" platform="darwin-arm64"` returns the right tarball URL pattern
- [ ] `tools/call install_guide harness="bogus"` returns structured `{error: "unknown_harness", available: [...]}`
- [ ] `make verify-install-guide` passes against current canonical sources

**Risks:**
- Hand-curated overrides go stale if `ailang_bootstrap/README.md` changes — Mitigation: the `make verify-install-guide` drift-check runs in CI
- Platform detection fragile — Mitigation: only require platform for `manual` harness; others auto-handle in their plugin

---

### M3 — `onboarding_guide(role)` tool + index (0.5 day)

**Goal:** A new agent (any harness) can call one tool and get a complete 8-step flow for using AILANG, including the syntax pitfalls we learned the hard way during M-AGENT-MCP.

**Estimated:** ~80 LOC Go (snapshot generator + overrides) + ~50 LOC AILANG (`mcp_tools/onboarding.ail`) + 30 LOC tests

**Tasks:**
- Day 2 midday: create `tools/build-snapshot/onboarding_guide_overrides.json` — `agent` and `developer` roles with stepwise flows. The `common_pitfalls` section pre-populated with the gotchas hit during M-AGENT-MCP execution: string `++` is list-only, `module` is reserved, hyphens forbidden in module paths, flag ordering matters
- Day 2 midday: extend `tools/build-snapshot/main.go` with `writeOnboardingGuideIndex()` — copies the overrides into `unscoped/onboarding_guide.json` (unscoped — flow doesn't change per release)
- Day 2 midday: new `mcp_tools/onboarding.ail` with `@mcp_name("onboarding_guide") @route` function that reads `unscoped/onboarding_guide.json` and filters by role
- Day 2 midday: bake an `mcp_setup` reference into step 2 of the agent flow (instructions for adding `mcp.ailang.sunholo.com` to the agent's MCP config — proof the agent is using it)

**Acceptance Criteria:**
- [ ] `tools/call onboarding_guide role="agent"` returns the 8-step flow
- [ ] `tools/call onboarding_guide role=""` defaults to `agent`
- [ ] `tools/call onboarding_guide role="developer"` returns developer-specific flow (less MCP-focused, more CLI-focused)
- [ ] `common_pitfalls` includes ≥4 specific syntax gotchas with example/correction pairs
- [ ] Tool registered in `tools/list` after deploy (catalog shows 23 total: 21 existing + install_guide + onboarding_guide)

**Risks:**
- Onboarding opinionation could be wrong for some harnesses — Mitigation: ship `agent` + `developer` in v1; iterate on actual feedback via `submit_feedback`
- Pitfalls list could grow unwieldy — Mitigation: cap at top-10, sort by frequency-of-fail observed in agent traces (future work)

---

### M4 — CHANGELOG + push dev → test → prod + verify end-to-end (0.5 day)

**Goal:** All four pieces (3 fixed indexes + 2 new tools + cloud-run automation) live in prod, verified by real tool calls.

**Estimated:** ~50 LOC CHANGELOG + ~20 LOC sprint summary docs + push/deploy/verify activity

**Tasks:**
- Day 2 afternoon: CHANGELOG entry under M-AGENT-MCP-ONBOARDING describing all 4 fixes with example tool-call shapes
- Day 2 afternoon: push to ailang/dev → wait for ailang-core-dev rebuild → push to ailang-multivac/dev → wait for multivac-dev deploy (now auto-deploys via the M1 fix)
- Day 2 afternoon: smoke-test dev MCP — call all 4 fixed/new tools, verify real responses
- Day 2 afternoon: promote dev → test → prod (well-rehearsed dance from M-AGENT-MCP)
- Day 2 evening: smoke-test prod at `https://mcp.ailang.sunholo.com/mcp/` — call `docs_search`, `example_for_concept`, `stdlib_search`, `install_guide`, `onboarding_guide`. Verify `tools/list` returns 23 tools
- Day 2 evening: open a follow-up `submit_feedback` from inside the prod MCP saying "M-AGENT-MCP-ONBOARDING shipped — please test the new install_guide and onboarding_guide tools and report back" so it shows in the public-feedback inbox as a self-test of the closing-the-loop pattern

**Acceptance Criteria:**
- [ ] `make ci` passes (build, test, lint, verify-examples, verify-mcp-tools, verify-install-guide)
- [ ] `https://mcp.ailang.sunholo.com/mcp/` returns 23 tools in `tools/list`
- [ ] All 4 broken-now-fixed tools return real responses (no `snapshot_read_failed`)
- [ ] No manual `gcloud run services update --image=:latest` needed during prod deploy (M1 automation works)
- [ ] CHANGELOG entry references the design doc + this sprint plan
- [ ] Self-test `submit_feedback` from prod URL lands in `inbox_messages` Firestore collection

---

## Success Metrics (sprint-level)

- **Tests passing**: `make ci` green; `make verify-install-guide` (new) green
- **Tool count in prod**: 23 tools at `tools/list` (21 existing + install_guide + onboarding_guide)
- **Three regressions closed**: docs_search, example_for_concept, stdlib_search all serving real data
- **Deploy automation works**: prod deploy doesn't need manual force-redeploy
- **Documentation updated**:
  - [ ] `docs/docs/guides/agent-mcp.md` — add install_guide + onboarding_guide to the tool catalog table
  - [ ] CHANGELOG (`changelogs/v0.10-current.md`) — M-AGENT-MCP-ONBOARDING entry
  - [ ] No design doc move — stays in `planned/v0_15_0/` until v0.15.0 release

## Dependencies

- M-AGENT-MCP shipped to prod (✅ done)
- Existing infrastructure: `internal/apiserver/mcp.go`, `tools/build-snapshot/`, `mcp_tools/` directory
- GCP IAM/secrets: no changes (reuses existing MCP service account from M-AGENT-MCP M3)
- DNS: no changes (`mcp.ailang.sunholo.com` already provisioned with cert)

## Open Questions (resolve during execution)

1. **JSON shape for examples_report**: M-AGENT-MCP M2 hit this — need defensive parsing. Recommend: try `json.NewDecoder().Decode(&result)` first; if `result` is a slice fall back to JSONL line-by-line.
2. **install_guide platform field**: should it be required or optional? Recommend: optional, defaults to "all platforms" for manual install; ignored for plugin-managed harnesses.
3. **Self-test `submit_feedback` from M4**: Should this be permanent (every release does a self-test from prod) or one-time? Recommend: one-time for M4 acceptance; future releases can do it via the post-release skill if desired.

## Notes

- M1, M2, M3 can technically run in parallel after the snapshot generator scaffolding is in place (M1 first task) — but for sprint-executor running sequentially, the order above keeps checkpoints clean
- M4 must come last (verifies everything end-to-end including the prod deploy)
- This sprint introduces NO new GCP resources, NO Terraform changes (except the cloudbuild.yaml addition for auto-redeploy), NO IAM changes, NO custom domain changes — pure additions to the existing service
- The dev → test → prod chain is well-rehearsed; ETA per env is ~15 min build + ~5 min deploy
