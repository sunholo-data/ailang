# Cloud release and promotion — the full picture

On-demand companion to `SKILL.md` §7.6–7.7. Design:
`design_docs/planned/v0_35_0/m-unified-release-model.md` (supersedes
`design_docs/implemented/v0_24_1/m-release-gate.md` steps 4–5).

## What a `v*` tag does (`cloudbuild-release.yaml`, trigger `ailang-core-release`)

```
(0) CI gate     GitHub workflow `CI`, job `test`, must be green for the tagged SHA
                (160 × 15 s = 40 min budget; CI on dev takes 19–23 min). deploy-test
                waits on it, so red CI leaves TEST untouched.
(1) build       all 18 images from the tagged tree → TEST registry, each tagged
                :vX.Y.Z AND :latest. No allowFailure anywhere (dev gives motoko and
                resident-pi a pass; a release does not — a version names the whole
                deployment). The resident image's acceptance suite is a hard gate.
(2) deploy TEST 4 core services + all 17 executor jobs, via ailang-multivac's
                scripts/cloudbuild-lib.sh (cloned at run time; the one copy of the
                roll / job→image logic, mirroring terraform/cloud_run_jobs.tf).
(3) smoke gate  MCP serves exactly the tag (GATE 1 — hence std/VERSION must equal
                it), docs_search answers, coordinator + dashboard /health.
```

It **stops there**. The 18 images: coordinator, dashboard, mcp, ntfy, agent-base,
agent, agent-go, agent-pi, agent-pi-go, agent-codex, agent-codex-go, agent-opencode,
agent-gemini, agent-gemini-go, agent-eval, agent-eval-go, agent-motoko, resident-pi.

Until 2026-09-03 this pipeline built 5 images, deployed `:latest` rather than the
version, rolled 2 of 17 jobs, and auto-promoted to prod after the smoke gate; the other
13 images were built only by ailang-multivac's branch pipelines from whatever dev head
was current, so a prod branch push could ship unreleased code (it did, 2026-09-02).

## Promote to prod (`promote-to-prod`, ailang-multivac)

```bash
scripts/release.sh promote core vX.Y.Z --dry-run   # guards + plan only
scripts/release.sh promote core vX.Y.Z             # copy, retag, roll
```

Guards, before anything is copied: a SUCCESSFUL `ailang-core-release` build exists for
the tag (images are pushed in parallel with the gates, so registry tags alone prove
nothing), and every image of the set carries `:vX.Y.Z` in test. Then `crane copy` by
version, `crane tag … latest` in prod, digest assertion, and a roll with `:latest` —
which is what Terraform pins by string, so the next apply sees no drift while the
registry records which version is live. Rollback: `promote core v<previous>`.

Sets: `core` (all 18 + 4 services + 17 jobs), `agents`, `agent`, `agent-pi`,
`coordinator`, `dashboard`, `mcp`; `docparse`/`billing` and `website-builder` are
promoted by their own repos' versions. There is no `all`.

Terraform and config travel by branch (`dev → test → prod`, the config-only triggers)
— never by promote. If a release needs new terraform, push that branch first.

## Verifying a version is live

Cloud Run reports the linux/amd64 **child** digest of a buildx image, not the OCI index
digest a tag points at. Compare a service's `status.imageDigest` against
`crane manifest <img>:vX.Y.Z | jq .manifests[].digest`, not `crane digest`. The
coordinator (buildpacks, single manifest) is the one image where the two agree. Jobs
expose the digest only on executions (`gcloud run jobs executions describe …
--format 'value(template.containers[0].image)'`).

## Break-glass caveat

`cloudbuild-dev.yaml` submitted at prod builds all 18 images and rolls 17 jobs but tags
`:latest` only, bypassing promote-by-version: the registry will not record which version
prod runs. Re-promote a real version afterwards.
