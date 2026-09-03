## M-UNIFIED-RELEASE-MODEL: push → dev, tag → test, promote by version → prod

**Status**: 🚧 Implemented, awaiting validation on v0.35.0 (the first tag through the new pipeline)
**Target**: v0.35.0
**Priority**: P1 — prod received unreleased code on 2026-09-02 through a branch push, and most executor images were never versioned at all
**Owner**: release-manager
**Supersedes**: `design_docs/implemented/v0_24_1/m-release-gate.md` steps (4)–(5) and its "Trigger changes"
**Companion repo**: `sunholo-data/ailang-multivac` (pipelines, shared library, triggers-as-code, docs)

---

## Problem

Deployment was a half-finished migration with both models live at once:

| | dev | test | prod |
|---|---|---|---|
| coordinator, dashboard, mcp, agent, agent-go | push ailang `dev` | `v*` tag **and** push multivac `test` | tag auto-promote **and** push multivac `prod` **and** manual promote |
| pi, codex, opencode, gemini, eval, their -go flavours, motoko, resident-pi | push **multivac** `dev` | push multivac `test` only | push multivac `prod` or manual promote |

- A push to the multivac `test`/`prod` branch ran `cloudbuild.yaml`: a full rebuild of every image from ailang's dev head *at that moment*. The release skill called those triggers "Terraform only". On 2026-09-02 that route put unreleased code into prod, dev was pushed straight to prod, two prod builds were cancelled mid-flight, and prod only became consistent through a manual `promote-to-prod _SERVICE=all`.
- The tag pipeline built 5 of 18 images, deployed `:latest` rather than the version, had no GitHub token (so its agent-base clone would now fail under the throttle measured 2026-09-02), rolled 2 of 17 jobs, and auto-promoted to prod after a smoke gate that never executes an agent job.
- The 13 executor images belonged to no version: they existed only in the multivac pipelines, so "prod is on v0.34.0" described the coordinator and said nothing about the image 32 of 35 prod agents run.
- Test and prod had been frozen on the tag path since v0.34.0 (2026-08-26) while dev took 366 commits.
- Four documents gave four different lists of valid promote targets; the trigger setup script could not recreate the live trigger set.

## Decision (2026-09-03)

1. **push → dev.** `cloudbuild-dev.yaml` builds all 18 images on every ailang `dev` push and rolls the 4 core services and all 17 executor jobs.
2. **`v*` tag → test.** `cloudbuild-release.yaml` builds the same 18 images from the tagged tree, tagged `:vX.Y.Z` **and** `:latest`, deploys test (services + 17 jobs), and runs the CI gate and the smoke gate. It **stops there**. No step may fail: a version names the whole deployment.
3. **promote by version → prod.** `promote-to-prod` (ailang-multivac, manual) takes `_SERVICE` and `_VERSION`, refuses unless a SUCCESSFUL release build exists for the tag **and** every image of the set carries `:vX.Y.Z` in test, copies exactly those images to prod, moves prod's `:latest` to the same digests, and rolls services and jobs with `:latest` — which is what Terraform pins by string, so the next apply sees no drift while the registry records which version is live.
4. **Branches are for infrastructure only.** The multivac `dev`/`test`/`prod` branches keep the config-only triggers (terraform + config, roll coordinator/mcp). The three full-pipeline branch triggers are retired. Infra first, then tag: a release that needs new terraform pushes the branch before the tag.
5. **One copy of the deploy logic.** `ailang-multivac/scripts/cloudbuild-lib.sh` (POSIX sh, sourced by all four pipelines after a shallow clone) holds the job→image map mirroring `terraform/cloud_run_jobs.tf`, the roll/retry logic, and the promote guards. Tested offline against stubbed gcloud/crane (`make test-lib`).
6. **Triggers as code.** `ailang-multivac/terraform-triggers/` declares the 12 file-based/manual triggers (imported 2026-09-03; the 6 inline-config triggers are a follow-up).

## The 18 images and their DAG

```
coordinator (buildpacks)   dashboard   mcp   ntfy
agent-base ─┬─ agent ───────── agent-go
            ├─ agent-pi ────── agent-pi-go
            │                └─ resident-pi (BASE build-arg; context docker/resident)
            ├─ agent-codex ─── agent-codex-go
            ├─ agent-gemini ── agent-gemini-go
            ├─ agent-eval ──── agent-eval-go
            ├─ agent-opencode
            └─ agent-motoko
```

Dev keeps `allowFailure` on motoko and resident-pi (externally maintained / Preview); the release pipeline does not.

## Promote vocabulary

| `_SERVICE` | copies test→prod `:vX` + moves `:latest` | rolls | version series |
|---|---|---|---|
| `core` | all 18 | coordinator (5 attempts), dashboard, mcp, ntfy-if-exists, 17 jobs | ailang `vX` |
| `agents` | the 14 executor images | 17 jobs | ailang |
| `agent` / `agent-pi` | that family | its jobs | ailang |
| `coordinator` / `dashboard` / `mcp` | one image | one service | ailang |
| `docparse` / `billing` | prefixed image | one service | docparse `vX` |
| `website-builder` | one image | one service | ailang-demos `vX` |

`all` was removed: docparse/billing and website-builder are versioned by their own repos' tags, so one `_VERSION` cannot name them together with ailang's.

## Acceptance (v0.35.0)

After the tag build: one SUCCESS `ailang-core-release` build for `TAG_NAME=v0.35.0`; 18 images carry `:v0.35.0` in test with `crane digest :v0.35.0 == :latest`; test services serve digests that are children of the `:v0.35.0` OCI indexes (Cloud Run reports the linux/amd64 child, not the index); all 17 test jobs updated inside the deploy-test window and an execution resolves to a `:v0.35.0` child; MCP `ailang_versions.latest == 0.35.0`; `promote core v0.34.0 --dry-run` still refused.

After `promote core v0.35.0`: the same digests in prod, prod services on them, 17 prod jobs updated, `mcp.ailang.sunholo.com` on 0.35.0, docparse/billing/website-builder revisions untouched.

## Follow-ups

- Pin `*_image_tag` per environment in the multivac tfvars so a promotion becomes a reviewed tfvars change.
- Convert the 6 inline-config triggers (docparse, demos, registry validator) to file-based configs in their repos and import them.
- `BASE_TAG` build-arg so release children `FROM :vX` rather than the parent's `:latest` (hermetic DAG; resident-pi already does this).
- Artifact Registry cleanup policy (keep `^v`, `latest`, `cache`; delete untagged >14d).
- The demos repo has never been tagged; website-builder cannot be promoted by version until it is.
- Add an executor-job execution to the smoke gate.
