# Sprint Plan: M-MANAGED-AGENTS — Retire Gemini CLI, Adopt Vertex Managed Agents API

## Summary

Delete the Gemini CLI executor and replace it with a Managed Agents API executor talking to Vertex AI via ADC, unblocking `gemini-3-5-flash` agent-mode for v0.22.0 evals. Source: [m-antigravity-cli-migration.md](m-antigravity-cli-migration.md).

**Duration:** 3-4 days
**Dependencies:** Vertex AI `v1beta1/...interactions` endpoint (verified live), ADC (`gcloud auth application-default login`)
**Risk Level:** Medium (new API surface, schema may drift before v0.22.0 release)
**Hard deadline:** Merge before **2026-06-10** (one week before Gemini CLI shutoff on 2026-06-18)

---

## Current Status Analysis

### Completed Recently
- ✅ M-COG-RUNTIME-BROWSER M5 (browser substrate) — 5 milestones shipped over 7 days
- ✅ `gemini-3-5-flash` added to models.yml with `max_output_tokens` plumbing fix ([f29029a3](https://github.com/sunholo-data/ailang/commit/f29029a3))
- ✅ v0.20.0 dashboard updated with gemini-3-5-flash standard-mode results ([b1ab977b](https://github.com/sunholo-data/ailang/commit/b1ab977b))

### Velocity (last 7 days)
- ~62k LOC churn across project, ~8.8k LOC/day average
- M-COG-RUNTIME-BROWSER shipped 11 day-numbered milestones — strong sustained per-day cadence
- Both heavy refactors (executor changes) and small ones (config tweaks) are in distribution

### Remaining Work (from M-MANAGED-AGENTS design doc)
- ⏳ M1: Delete gemini executor + models.yml cleanup + config-load validation hook (~50 LOC net deletion)
- ⏳ M2: New `internal/executor/managed_agents/` package (~700 LOC: HTTP client + SSE parser + executor wrapper + tests)
- ⏳ M3: 3-benchmark smoke gate (no LOC, just runs)
- ⏳ M4: Documentation + memory (~150 LOC docs)

---

## Proposed Milestones

### Milestone 1: Retire Gemini CLI Executor

**Goal:** Remove the Gemini CLI executor cleanly; reject `agent_cli: "gemini"` at config load with a clear next-step message.
**Estimated:** -250 LOC implementation (deletions) + 30 LOC validation hook = **~280 LOC net change**
**Duration:** 0.5 days

**Tasks:**
- Day 1 (AM): Delete `internal/executor/gemini/` package; remove blank import from `internal/coordinator/provider_executor.go`; strip `agent_cli: "gemini"` + `agent_model_name` from 8 models.yml entries (keep standard-mode `provider: google` config intact).
- Day 1 (AM cont.): Add validation in `internal/eval_harness/models.go::GetExecutorForModel` (or `ResolveModelName`) that returns a clear error for the literal string `"gemini"`.

**Acceptance Criteria:**
- [ ] `internal/executor/gemini/` package no longer exists (or tombstone-only `doc.go`)
- [ ] No `agent_cli: "gemini"` entries survive in `models.yml`
- [ ] `internal/coordinator/provider_executor.go` no longer imports the gemini executor
- [ ] `ailang eval-suite --agent --models gemini-3-5-flash --benchmarks fizzbuzz --langs ailang` fails fast with the prescribed error message (since M2 hasn't landed yet)
- [ ] `make ci` green
- [ ] `make test` green for `internal/eval_harness/...`

**Risks:**
- Coordinator unit tests reference `internal/executor/gemini` imports. *Mitigation*: grep and remove any test stubs.
- External tooling references `agent_cli: "gemini"` in vendored configs. *Mitigation*: error message is the documentation.

---

### Milestone 2: Managed Agents API Executor

**Goal:** Ship `internal/executor/managed_agents/` — an HTTP/SSE executor talking to `aiplatform.googleapis.com/v1beta1/.../interactions` via ADC, implementing the full `executor.Executor` interface.
**Estimated:** ~500 LOC implementation + ~200 LOC tests = **~700 LOC**
**Duration:** 2 days

**Tasks:**
- Day 1 (AM): Scaffold package skeleton (`executor.go`, `client.go`, `sse.go`, `types.go`, `README.md`, `testdata/`). Move the verified fixture from `design_docs/planned/v0_22_0/testdata/managed_agents_sse_pong.txt` into `internal/executor/managed_agents/testdata/sse_pong.txt` so it ships with the package.
- Day 1 (PM): HTTP client + ADC auth via `golang.org/x/oauth2/google.FindDefaultCredentials` (mirror `internal/ai/gemini/client.go` pattern). Build POST request with required headers (`Api-Revision: 2026-05-20`) and verified body shape (`stream: true, background: true, store: true, agent, environment: {type: "remote"}, input: [{type: "user_input", content: [{type: "text", text: <prompt>}]}]`).
- Day 2 (AM): SSE parser. Stream-line-oriented: read `event: <name>\n` followed by `data: <json>\n` blocks. Handle event types verified in fixture: `interaction.created`, `interaction.status_update`, `step.start`, `step.delta`, `step.stop`, `interaction.completed`, `done`. Unknown event names → captured into `ProviderData["managed_agents_events"]` map for forward compat.
- Day 2 (AM cont.): `Result` mapping. Accumulate `step.delta.text` into `Output`. Map `interaction.completed.usage` to `InputTokens/OutputTokens/TotalTokens` fields; compute `CostUSD` client-side from gemini-3-5-flash rates ($1.50/1M input, $9.00/1M output+thought). `Success = (interaction.completed.status == "completed")`. Capture `interaction.id` + `environment_id` into `ProviderData` for multi-turn (future use; not required for first cut).
- Day 2 (PM): Register in `internal/executor/factory.go`; add blank import to `internal/coordinator/provider_executor.go`. Unit tests against the saved fixture (replay-parser test, token/cost extraction test). Gated live test (`AILANG_MANAGED_AGENTS_LIVE=1`) that hits the real API.

**Acceptance Criteria:**
- [ ] `internal/executor/managed_agents/` package exists; implements all 7 methods of `executor.Executor` interface
- [ ] `Register()` + `init()` pattern matches other executors (see `claude/claude.go:764-773` reference)
- [ ] Unit tests pass: `go test ./internal/executor/managed_agents/...`
- [ ] Integration test (gated) hits the real Vertex `v1beta1/.../interactions` endpoint via ADC and parses an end-to-end response
- [ ] Coordinator + eval harness can resolve `agent_cli: "managed_agents"` to this executor (auto-discovery via factory)
- [ ] `make test` green
- [ ] `make lint` green

**Risks:**
- API schema drift between today (2026-05-20) and v0.22.0 release. *Mitigation*: `Api-Revision: 2026-05-20` header pins behavior; `ProviderData` map captures unknown events; lenient parsing on absent fields.
- Tool-use event types not in the fixture (the `say PONG` probe didn't trigger any tools). *Mitigation*: capture a tool-use fixture during M3 if smoke benchmarks trigger tool steps; add parser support iteratively.
- Vertex pricing differs from raw `gemini-3-5-flash` rates due to managed-sandbox premium. *Mitigation*: cost computation is from API-reported `usage`, so it's accurate; the rates we plug in just need to track public Vertex pricing. Flag any large variance in M3.

---

### Milestone 3: Wire `gemini-3-5-flash` to Managed Agents + Smoke Gate

**Goal:** Bring `gemini-3-5-flash` agent-mode back online via the new executor; verify against the standard 3-benchmark smoke set.
**Estimated:** ~10 LOC config + 0 LOC code = **~10 LOC**
**Duration:** 0.5 days

**Tasks:**
- Day 3 (AM): Add `agent_cli: "managed_agents"`, `agent_model_name: "antigravity-preview-05-2026"`, `gcp_project: "ailang-dev"`, `gcp_location: "global"`, `max_output_tokens: 65536`, and existing `budgets` to the `gemini-3-5-flash` entry in `models.yml`.
- Day 3 (AM): Run smoke gate: `ailang eval-suite --agent --models gemini-3-5-flash --benchmarks fizzbuzz,adt_option,csv_to_json_converter --langs ailang --output /tmp/smoke_managed_agents --parallel 2`. Expect 3/3 PASS.
- Day 3 (AM): If the smoke surfaces new SSE event types not in the fixture (e.g. tool-use steps), capture as a new fixture and tweak the parser as needed.

**Acceptance Criteria:**
- [ ] `gemini-3-5-flash` in models.yml has `agent_cli: "managed_agents"` and supporting config
- [ ] Smoke gate passes 3/3 on `fizzbuzz`, `adt_option`, `csv_to_json_converter` (ailang only)
- [ ] If tool-use events appeared, fixture saved and parser updated
- [ ] Smoke-test result files retained for the v0.22.0 release validation

**Risks:**
- Smoke fails not because of executor bug but because of API rate limits or quota. *Mitigation*: `--agent-parallel 2`, retry once, escalate if persistent.
- Tool-use event shape requires meaningful parser work. *Mitigation*: budget half a day extra; only fizzbuzz/adt_option/csv_to_json should reach tool-use territory if at all.

---

### Milestone 4: Documentation + Memory

**Goal:** Bring docs and memory up to date so future sessions don't redo the investigation.
**Estimated:** ~150 LOC docs + 2 memory files = **~150 LOC**
**Duration:** 0.5 days

**Tasks:**
- Day 3 (PM): Rewrite `docs/docs/guides/evaluation/harness-setup.md` — remove Gemini CLI section, add Managed Agents API setup (ADC, project/location, one-time provisioning note).
- Day 3 (PM): Update `.claude/rules/coordinator.md` executor table.
- Day 3 (PM): Write memory files:
  - `/Users/mark/.claude/projects/-Users-mark-dev-sunholo-ailang/memory/feedback_managed_agents_v1beta1_path.md` — "When probing new Vertex AI APIs, try `v1beta1` first" with the bad-probe story.
  - `/Users/mark/.claude/projects/-Users-mark-dev-sunholo-ailang/memory/reference_managed_agents_api.md` — verified endpoint + body + event schema + multi-turn handle reference.
- Day 3 (PM): Send `ailang messages send` notifications to `sunholo-demos` and `motoko_explore` agents: gemini-cli executor removed, agent-mode now via `managed_agents`.

**Acceptance Criteria:**
- [ ] `harness-setup.md` reflects new state — no Gemini CLI references
- [ ] `coordinator.md` executor table is current
- [ ] Both memory files written and indexed in `MEMORY.md`
- [ ] Cross-repo notifications sent

**Risks:** None material.

---

## Success Metrics

- Test coverage: maintained at current baseline (>50% for `internal/executor/managed_agents/`)
- Examples passing: N/A (no new AILANG language features in this sprint)
- Documentation updated: `harness-setup.md`, `coordinator.md`, two memory files
- Smoke gate: 3/3 PASS for `gemini-3-5-flash` via Managed Agents API
- All tests passing: ✅
- All linting passing: ✅

## Dependencies

- Vertex AI `v1beta1/.../interactions` endpoint live on `ailang-dev` project (verified 2026-05-20 — see [testdata/managed_agents_sse_pong.txt](testdata/managed_agents_sse_pong.txt))
- ADC configured (`gcloud auth application-default login`)
- Gemini CLI shutoff on 2026-06-18 — hard deadline gives ~21 days, sprint sized for 3-4

## Open Questions

- **Tool-use event schema**: not captured in the minimal probe. Will be captured during M3 smoke if relevant; design supports lenient parsing of unknown events.
- **Multi-turn benchmark support**: first cut provisions a fresh sandbox per benchmark. Sandbox/conversation reuse via `interaction.id` + `environment_id` is captured but unwired. Defer to a follow-up if cost optimization needed.
- **Cloud Run Jobs deprecation**: out of scope for this sprint. Once Managed Agents executor has shipped one full release cycle of green data, file a separate sprint to mark the Gemini-family Cloud Run Jobs path deprecated in multivac.

## Notes

- The first version of this sprint plan said the Vertex Managed Agents API "doesn't exist" because I probed `v1` and `v1beta` paths and missed `v1beta1`. The user pointed me at the docs page, a `v1beta1` probe immediately succeeded, and the sprint shape went from 1-2 days (retirement only) back to 3-4 days (retirement + new executor). The `feedback_managed_agents_v1beta1_path.md` memory in M4 captures this lesson.
- No Cloud Run Jobs changes in this sprint. The Managed Agents executor lands as a peer of the existing Cloud Run Jobs path for Gemini-family models. Separate v0.23.x sprint deprecates the Cloud Run Jobs Gemini path once Managed Agents has shipped one release cycle of green data.
- `gemini-3-5-flash` standard mode (88.2% AILANG in v0.20.0) is **not affected** by this sprint — it continues via direct Vertex `generateContent` calls.
