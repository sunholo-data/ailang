## M-RELEASE-GATE: Gated release pipeline — tag → test → smoke gate → prod

**Status**: PROPOSED (awaiting approval to create the prod trigger)
**Target**: v0.25.0
**Priority**: P1 — prod (public `mcp.ailang.sunholo.com` + coordinator) currently updates only via a manual, forgettable step; releases v0.20–v0.24 silently never reached prod.
**Owner**: release-manager

## Problem

`release.yml` builds only CLI binaries. The Cloud Run stack (coordinator, agent jobs,
dashboard, **MCP docs server**) updates through a separate Cloud Build pipeline that was
**broken since ~v0.19.1** (inline agent build missing `--build-arg PROJECT` →
`invalid reference format`). Even after fixing the test trigger, **prod only updates via a
manual `promote-to-prod`** — so the public MCP froze at 0.19.1 for ~3 weeks and agents got
`unknown_version` for new releases (reported by BlackMage, 2026-05-31).

Decision (user, 2026-06-05): **do NOT auto-roll prod ungated.** Add a **test gate that must
pass before the prod roll.**

## Design

One tag-triggered Cloud Build pipeline (`cloudbuild-release.yaml`, in the **ailang** repo so
a file-based trigger on the ailang repo can read it) that runs **sequentially**:

```
v* tag pushed
   │
   ├─(0) UNIT TESTS (make test)  ← runs in parallel; gates the prod roll
   ├─(1) build core images from the released source  → push to TEST registry (:latest + :${TAG})
   ├─(2) deploy TEST  (gcloud run update test services + jobs)
   ├─(3) SMOKE GATE against test  ← service-health + version safety
   │        fail → STOP. prod untouched.
   ├─(4) promote images test→prod (crane copy)         ← only if SMOKE GATE *and* UNIT TESTS passed
   └─(5) deploy PROD  (gcloud run update prod services + jobs)
```

**Two gates protect prod** (both must pass before `promote-images`):
- **CI gate** (step 0) — requires the authoritative `CI` workflow (`ci.yml`, job `test`:
  `go test ./...` + parser/coverage/golden/import checks, with the `ailang` binary installed) to
  have concluded `success` for the tagged commit. Polls the GitHub check-runs API up to ~15 min;
  red or absent CI → fail-closed. **We do NOT re-run the suite inside the deploy pipeline** —
  that proved flaky (empty `$TAG_NAME`, missing `ailang` binary, a timing-sensitive 10s-timeout
  test all failed the deploy on env issues, never code regressions). Requiring the existing CI
  result is authoritative and flake-free in the deploy path.
- **Smoke gate** (step 3) — catches deploy/runtime/version-serving problems CI can't (image
  actually deployed, MCP serves the released version, services healthy).

**Root-cause companion fix:** `ci.yml` now also runs `make check-file-sizes` (it previously
ran tests but not the size gate, which let `parser_expr.go` exceed 800 lines onto `dev`). The
remaining gap — `dev` accepts direct pushes with no merge gate, so a red CI doesn't block the
push — needs **branch protection** (require CI green before merge); that's a GitHub repo setting,
tracked as a manual follow-up.

Steps 1–2 reuse `cloudbuild-dev.yaml`'s proven build+deploy logic (already passes
`--build-arg PROJECT`). Steps 4–5 reuse `cloudbuild-promote.yaml`'s `crane copy` + deploy
logic (already exists in multivac for the manual `promote-to-prod`). The **only new code is
step 3, the gate.** Prod steps `waitFor` the gate, so a red gate leaves prod on the last
known-good revision.

### The smoke gate (step 3) — fast, deterministic, no models

Run against the just-deployed **test** services (URLs are stable Cloud Run URLs):

| Check | Call | Pass condition |
|-------|------|----------------|
| MCP serves new version | `POST $TEST_MCP/api/mcp/ailang_versions` | `result.latest == ${TAG#v}` |
| MCP docs answer the version (BlackMage's exact failing call) | `POST $TEST_MCP/api/mcp/docs_search {forVersion:"${TAG#v}", query:"module"}` | not `unknown_version`, ≥1 hit |
| Coordinator healthy | `GET $TEST_COORD/health` | HTTP 200 |
| Dashboard healthy | `GET $TEST_DASH/health` | HTTP 200 |

Rationale: these are seconds-fast, require no API keys/models, and directly guard the exact
failure class that caused this incident (a stale/broken image reaching prod, and the MCP not
serving the released version). The heavier agent-mode `--tier smoke` eval is **out of scope**
for the gate (needs model access; belongs in nightly/CI, not the release critical path).

### Trigger changes

- **New**: `ailang-core-release` — fires on ailang repo `push.tag ^v.*`, `filename:
  cloudbuild-release.yaml`, SA `sa-cloudbuild`. This is the prod-affecting trigger.
- **Remove/disable**: `ailang-core-test-release` — superseded (the new pipeline deploys test
  as step 2). Keeping both would double-deploy test on every tag.
- **Keep**: `promote-to-prod` (manual, `_SERVICE`-scoped) as a break-glass for ad-hoc
  single-service promotion.

## Out of scope / follow-up

- These Cloud Build triggers are **not in Terraform** (pure imperative/inline — the source of
  the original drift). A separate doc should bring all `ailang-*` triggers under multivac
  Terraform to kill the drift class. Tracked, not done here.
- Versioned (`:${TAG}`) image retention/rollback policy.

## Acceptance

- [ ] Pushing a `v*` tag deploys test, runs the gate, and only then rolls prod.
- [ ] A deliberately-failing gate (e.g. point MCP check at a wrong version) leaves prod on the
      prior revision (verified once in test).
- [ ] `mcp.ailang.sunholo.com` `result.latest` matches the new tag after a release.
- [ ] release-manager skill step 7.6 updated: prod is now gated-automatic; manual command kept
      as fallback.
