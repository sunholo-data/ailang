# Sprint Plan: M-PKG-AUTONOMOUS-CASCADE-SAFE

## Summary

Validate the v0.10.0 autonomous package cascade end-to-end + harden the cascade-trigger channel via topic-IAM separation + add observability. This is the precondition for publicly claiming "AILANG has an autonomous package update system" and the precondition for the broader [M-PKG-TRUSTED-AUTONOMOUS-EVOLUTION](m-pkg-trusted-autonomous-evolution.md) v0.11.0 sprint.

**Design doc:** [m-pkg-autonomous-cascade-safe.md](m-pkg-autonomous-cascade-safe.md)
**Duration:** 2 working days, ~18 hours
**Total LOC:** ~510 (150 Go code + 80 bash + 30 Terraform + 250 docs)
**Risk Level:** Low-medium — most infrastructure already exists (cascade scheduler, autonomy router, provenance, pkg history) and was tested in M-PKG-AUTONOMOUS-UPDATES; this sprint wires + validates + adds one IAM boundary.

---

## Current Status Analysis

### Completed (precursor)
- ✅ M-PKG-AUTONOMOUS-UPDATES (v0.10.0): cascade scheduler with topo sort + circuit breaker, autonomy router (class A/B/C), 13 cloud package agents, provenance + AGENT.md, FindDependents, `ailang pkg history` CLI — but NEVER observed end-to-end in cloud
- ✅ M-PKG-FEEDBACK-LOOP M2 (yesterday, in dev/test/prod): cloud-dispatch templating fix (commit `228d5c0a`) — was the silent blocker preventing `pkg-update.md` from reaching cloud agents
- ✅ M-CI-BUILD-SPEED (yesterday): 60s config-only deploys, ~6min full-pipeline cache-warm — makes this sprint's iteration cycle 25× faster than it would have been

### Velocity (last 2 days)
- M-PKG-FEEDBACK-LOOP M2 + cloud-dispatch fix + test-pkg agent + drift fixes + docs: shipped in ~3h with ~750 LOC across two repos
- M-CI-BUILD-SPEED 6 milestones: shipped in ~3h with ~310 LOC across two repos
- **Sustained velocity ~150-200 LOC/hour for CI/infra work** — this sprint's 510 LOC ÷ ~18h is conservative

### Why aggressive 2-day timeline is realistic
- Most components already exist; we wire them, not build them
- M-CI-BUILD-SPEED makes deploy iteration ~60s (config-only) or ~6min (full) instead of 25min
- Today's M-PKG-FEEDBACK-LOOP M2 already proved the cloud-dispatch templating works end-to-end (we saw real GitHub issues created by cloud agents)

---

## Proposed Milestones

All 6 milestones map 1:1 to the design doc's implementation phases. They are **sequentially dependent** (each builds on the prior).

### M1 — Topic + IAM separation (~4h, ~70 LOC)

**Goal:** Create the `ailang-{env}-cascade` Pub/Sub topic with coordinator-SA-only publish permission. Surface `source_topic` attribute through the pubsub adapter so agents can distinguish cascade messages from public-MCP-routed messages.

**Tasks:**
- New Terraform: `ailang-multivac/terraform/pubsub_cascade.tf` — `google_pubsub_topic` + `google_pubsub_topic_iam_binding` (publisher = coordinator SA) + agent SA subscriber bindings (~30 LOC)
- Modify `internal/coordinator/pubsub_adapter.go` to surface `source_topic` attribute on inbound messages (~10 LOC + tests)
- Modify `ailang-multivac/config/templates/pkg-update.md` to add the guard "only act on bump if `source_topic == cascade`" (~10 LOC)
- Modify `ailang-multivac/config/config.cloud.yaml` so all 13 pkg-* agents subscribe to the new cascade topic (~5 LOC × 13 = ~65 LOC; technically 13 small edits)
- Apply Terraform to dev via M-CI-BUILD-SPEED config-only fast path (60s)

**Acceptance:**
- `gcloud pubsub topics describe ailang-dev-cascade` returns the new topic
- `gcloud pubsub topics get-iam-policy ailang-dev-cascade` shows only the coordinator SA has `roles/pubsub.publisher`
- A test publish as a non-coordinator SA fails with permission denied
- Existing tests pass; pubsub_adapter unit test covers `source_topic` attribute extraction
- 13 agent subscriptions present in `gcloud pubsub subscriptions list --project=ailang-multivac-dev | grep cascade`

**Risks:** Low. Mostly Terraform + one small Go change. Worst case: if `source_topic` attribute isn't natively in the Pub/Sub message envelope, we add it explicitly in the publisher (Phase 2).

---

### M2 — Coordinator publishes to cascade topic (~3h, ~30 LOC)

**Goal:** `cmd/ailang/pkg_publish.go::emitDependentNotifications` publishes cascade messages on the new `ailang-cascade` topic with `source=cascade` attribute, instead of (or in addition to, for backward-compat) writing to `pkg:*` inboxes.

**Tasks:**
- Modify `cmd/ailang/pkg_publish.go::emitDependentNotifications` to publish on cascade topic (~20 LOC)
- Dual-write to legacy `pkg:*` inbox for one-release backward-compat (deprecate in v0.17)
- Unit test: assert correct topic + attribute (~10 LOC)
- Deploy via M-CI-BUILD-SPEED full pipeline (~6 min cache-warm)
- Verify in dev: a test `ailang publish` writes to `ailang-dev-cascade` with the right attributes (manual `gcloud pubsub topics list-subscriptions` + a tail script)

**Dependencies:** M1 (topic must exist)

**Acceptance:**
- `emitDependentNotifications` publishes to `ailang-{env}-cascade` topic
- Message attributes include `source=cascade` and `root_package=<vendor>/<name>@<version>`
- Unit test asserts both via mock publisher
- Backward-compat: legacy `pkg:*` inbox write still happens (verified by `ailang messages list --inbox pkg:sunholo/test_pkg`)

**Risks:** Low. The publisher API already exists in the coordinator; this is a target-topic change.

---

### M3 — Per-cascade budget cap (~2h, ~55 LOC)

**Goal:** `ailang.toml` gains an optional `[cascade] max_cost_usd = 1.0` section. Cascade scheduler enforces the cap across the cascade DAG (cumulative); per-task hard ceiling stays unchanged. When cap exceeded, scheduler aborts further dispatches with a structured error.

**Tasks:**
- Add `[cascade]` section parsing to `internal/pkg/manifest.go` (~25 LOC + tests)
- Modify `internal/coordinator/cascade_scheduler.go` to honor per-cascade cap (~15 LOC + tests)
- Unit test: cap enforcement; default = $1.0 if section absent
- Integration test: forced-fail cascade with $0.10 cap aborts after first task (~15 LOC bash)
- Default of $1.0 USD; configurable per-package via the package's `ailang.toml`

**Dependencies:** M1, M2 (cascade messages need to actually flow first)

**Acceptance:**
- `ailang.toml` parses `[cascade] max_cost_usd = 0.50` (override) or defaults to 1.0 if absent
- Cascade scheduler accumulates per-task cost across DAG; aborts when cap exceeded
- Aborted cascade emits structured event (visible in `ailang chains view <cascade-id>`)
- Unit test + integration test (in scripts/integration/) both pass

**Risks:** Low. Coordinator already has per-task budget enforcement infrastructure.

---

### M4 — `ailang pkg cascade status` CLI command (~3h, ~120 LOC)

**Goal:** New CLI subcommand `ailang pkg cascade status <vendor>/<name>@<version>` reads existing chain + provenance data and prints: cascade DAG, per-node status (queued/running/pr_opened/failed/budget_exceeded), accumulated cost, PR URLs.

**Tasks:**
- New `cmd/ailang/pkg_cascade_status.go` (~80 LOC) reading from existing `chains` queries + `pkg history`
- Register subcommand in `cmd/ailang/main.go` (~5 LOC)
- New `cmd/ailang/pkg_cascade_status_test.go` (~40 LOC) — unit test on synthetic cascade record
- Help text + examples in `--help`

**Dependencies:** M1, M2, M3 (needs real cascade data to query)

**Acceptance:**
- `ailang pkg cascade status sunholo/test_pkg@0.0.4` returns the cascade DAG with statuses
- Output includes accumulated cost, budget remaining, PR URLs
- Unit test covers: empty cascade, single-node cascade, multi-node cascade with one failed
- Help text + examples accessible via `--help`

**Risks:** Low-med. The chain + provenance APIs exist but may need small tweaks to expose the right joined view.

---

### M5 — Smoke tests + dependent package fixture (~3h, ~150 LOC)

**Goal:** First end-to-end observation. Create the `sunholo/test_pkg_consumer` fixture package (depends on `sunholo/test_pkg`). Run a real `ailang publish` on test_pkg in test env. Observe the cascade fire and open a real PR on test_pkg_consumer in `ailang-packages` repo. Plus a negative test that public MCP cannot trigger a publish.

**Tasks:**
- Create `sunholo/test_pkg_consumer` package fixture in `ailang-packages` repo (~30 LOC `.ail` + `ailang.toml` + `AGENT.md`)
- `scripts/integration/test_cascade_e2e.sh` (~80 LOC bash):
  - Bumps `sunholo/test_pkg` version
  - Runs `ailang publish`
  - Polls for cascade fire (timeout 60s)
  - Polls for dependent agent run (timeout 5min)
  - Asserts a PR opened on test_pkg_consumer with the right metadata
  - Asserts provenance chain via `ailang pkg history`
  - Idempotent cleanup: deletes test PRs + branches + reverts version bump
- `scripts/integration/test_cascade_negative.sh` (~40 LOC bash):
  - Submits feedback via MCP with `auto_dispatch=true, package=sunholo/test_pkg, body=<crafted bump request>`
  - Polls for any new published version (timeout 5min)
  - Asserts NO publish happened (agent files an issue per `pkg-feedback.md`)

**Dependencies:** M1, M2, M3, M4 (need everything wired before testing end-to-end)

**Acceptance:**
- `test_cascade_e2e.sh` runs successfully against test env
- A real PR is opened on `sunholo-data/ailang-packages` for test_pkg_consumer
- `ailang pkg history sunholo/test_pkg_consumer` shows the cascade root
- `ailang pkg cascade status sunholo/test_pkg@<version>` shows the chain
- `test_cascade_negative.sh` confirms public MCP cannot trigger a publish
- Both scripts cleanup their test artifacts (PRs labeled `[smoke-test, do-not-merge]` deleted at end)

**Risks:** Med. This is the first observation of the v0.10.0 cascade infrastructure end-to-end. May surface latent bugs in `emitDependentNotifications` that have been silent. Budget 1h within the milestone for fixing whatever it surfaces.

---

### M6 — Promote + observe in prod + docs (~3h, ~280 LOC)

**Goal:** Promote dev → test → prod. Observe one real cascade in prod (no-op patch bump on `sunholo/test_pkg`). Write the developer guide. CHANGELOG entry. Move design doc + sprint plan to `implemented/v0_16_x/`.

**Tasks:**
- Promote multivac dev → test → prod (~6min × 2 envs via M-CI-BUILD-SPEED full pipeline; can run in parallel)
- Promote ailang dev (CLI changes) — automatic via ailang-core-dev trigger after merge
- Run `test_cascade_e2e.sh` in test env (must pass)
- Run `test_cascade_e2e.sh` in prod env (must pass) — ONE real prod cascade observation
- Write `docs/guides/autonomous-package-updates.md` (~200 LOC markdown) with annotated logs from the prod observation, including a "what happens, what doesn't" boundary diagram
- Update `cmd/ailang/prompts/v0.16.0.md` with cascade flow story (~30 LOC)
- CHANGELOG entry under v0.16.x section (~15 LOC) referencing this doc + parent + sister
- Move design doc + sprint plan from `design_docs/planned/v0_16_0/` to `design_docs/implemented/v0_16_x/`
- Update sprint JSON status to `completed`

**Dependencies:** M5 (smoke tests must pass before promotion)

**Acceptance:**
- Test env smoke test passes
- Prod env smoke test passes (one real cascade observed)
- `docs/guides/autonomous-package-updates.md` exists and walks through the prod observation
- CHANGELOG entry under v0.16.x
- Design doc + sprint plan moved to `implemented/v0_16_x/`
- Sprint JSON status = `completed`

**Risks:** Low. By the time we get here, everything has been validated in dev + test. Prod observation is the formality.

---

## Success Metrics (sprint-level)

- **All 6 milestones acceptance criteria pass**
- **One real cascade observed in prod** with full provenance trail and a real PR opened on `sunholo-data/ailang-packages`
- **Public MCP cannot trigger a publish** — verified by negative test
- **Per-cascade budget cap enforced** — verified by forced-fail test
- **Developer guide published** at `docs/guides/autonomous-package-updates.md`
- **All existing tests pass** — no regressions to M-PKG-FEEDBACK-LOOP M2 or M-CI-BUILD-SPEED
- **CHANGELOG updated** under v0.16.x section

## Dependencies

**Hard preconditions (already met):**
- ✅ M-PKG-AUTONOMOUS-UPDATES infrastructure (cascade scheduler, autonomy router, provenance) — v0.10.0
- ✅ M-PKG-FEEDBACK-LOOP M2 cloud dispatch templating fix — yesterday in dev/test/prod
- ✅ M-CI-BUILD-SPEED fast deploys — yesterday in dev/test/prod

**Soft dependencies (helpful, not blocking):**
- Cloud Build SA permissions on the new cascade topic — already has admin (no IAM change needed for CI itself)
- 13 pkg-* agents in cloud config — exists, just needs subscription update

## Open Questions

1. **Should the negative test run automatically in CI?** Recommend YES for `test_cascade_negative.sh` (cheap, runs in <30s, prevents regressions in the IAM separation). Recommend NO for `test_cascade_e2e.sh` (creates real PRs, expensive, needs cleanup) — leave as a manual smoke test triggered before each release.
2. **Whether to drop the legacy `pkg:*` inbox write in this sprint or in v0.17.** Defer to v0.17 to give one release of dual-write observation.
3. **Cleanup behavior of test_cascade_e2e.sh on failure.** Recommend: still attempt cleanup but don't mask the test failure. Print PR URLs that need manual cleanup if the script's cleanup itself fails.

## Notes

- All 6 milestones can run in dependency-ordered sequence. No parallelization opportunity (each builds on the prior).
- The 13 pkg-* agent YAML edits in M1 can be done with a `sed`/`yq` script — they're identical across agents.
- The smoke-test PR needs a `[smoke-test, do-not-merge]` GitHub label on `sunholo-data/ailang-packages` repo + a CI rule that blocks merge of any PR with that label. Add this in M5 setup.
- Prod cascade observation in M6 is the first time AILANG has demonstrated this functionality end-to-end. The CHANGELOG entry can lead with that framing.
- After this sprint, the next P0 follow-up is the AILANG core → packages cascade (separate sprint, larger scope, depends on this work being stable in prod for ≥7 days).
