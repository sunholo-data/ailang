# Dashboard rollout and data audit — 2026-09-08

Status: v0.35.3 deployed to production; query/function smoke checks pass.
Global task data completeness is not yet certified.

## Changes and release identity

- Application repair: `95955af6b` on ailang/dev.
- Simplified React UI: `bd06e6534` (11,206 net source lines removed).
- Release preparation: `3f30144ae`, v0.35.3, includes the current dev release contents.
- Infrastructure: `f32693c` on ailang-multivac, promoted through existing dev/test/prod
  branches; adds 12 query indexes in `terraform/dashboard_read_indexes.tf`.
- Read-only Terraform plans for each environment: 12 add, 0 change, 0 destroy.
  Existing infrastructure/config had no branch drift before this promotion.

## Verification evidence

| Check | Result |
|---|---|
| Backend sprint independent round2 | Scoped PASS 93/100 |
| Prior local test blockers | All four explicitly pass in hosted unfiltered Go tests |
| Dev chain pages | Distinct IDs at offsets 0 and 2; out-of-range page is `[]` |
| Dev malformed since | 400 |
| Dev agent pages | Three distinct pages; agent membership independently checked in each chain's stages |
| Dev indexed filters | Workspace, repository and combined query: 503 before indexes, 200 after |
| Dev stage ownership | Wrong chain: 404 |
| Dev stage spans and transcript | 503 before indexes, 200 after |
| Prod sampled stage spans and transcript | 500 before indexes, 200 after |
| Evidence in two sampled prod tasks | Zero stage spans, zero transcript messages, zero direct task-ID spans |
| UI assets | Production now serves the simplified assets: JS 720,858 bytes; CSS 203,195 bytes |
| Versioned test release | SUCCESS; all 18 images, CI gate and smoke gates pass |
| Production promotion | SUCCESS; public MCP reports 0.35.3 |
| Production input validation | Malformed since/negative offset: 400; wrong-chain stage: 404 |
| Production standalone standard tracing | 18 spans, zero function spans |
| Production standalone deep tracing | 40 spans, 22 function spans; factorial(10) args/result are 10/3628800 |
| Production trace-detail API | Retrieves all 40 spans of the deep trace |
| Generated project cache refresh | 65 syntax chunks, 342 builtins, 217 examples; active prompt v0.16.6 retrieved |

The dev agent page reads took 0.124–0.208 seconds across three calls. This is a
small live smoke check, not a load or latency distribution benchmark.

Hosted evidence for the scoped repair:
[CI/test at dd0a96ed8](https://github.com/sunholo-data/ailang/actions/runs/34231976270/job/102080121553).
Release gate for the exact v0.35.3 preparation commit:
[CI/test at 3f30144ae](https://github.com/sunholo-data/ailang/actions/runs/34239945243/job/102107234857).

## Deployment records

| Operation | Cloud Build ID |
|---|---|
| Dev application containing read fixes | `b1dea44b-dab7-4a5a-a999-5e33bd74ce69` |
| Dev indexes | `c11c41e5-01ef-4242-9269-6dcc776b2fd5` |
| Test indexes | `daef0038-7802-4c67-a246-838a2413c4b7` |
| Prod indexes | `294744cd-404c-4d92-b1d4-8b54aeb32e8b` |
| Versioned test release | `54d7c992-fc88-46d9-97f5-380029b738a6` |
| Production dry run | `6b1d4973-7213-4f20-b990-21a878db36d1` |
| Production promotion | `8448ab58-dab7-4644-a147-5dca0afda42f` |

Builds live in `ailang-multivac-deploy`, region `europe-west3`. Service region is
`europe-west1`. Production before application promotion: coordinator
`ailang-coordinator-00077-tjr`, dashboard `ailang-dashboard-00044-469`.
Prior v0.35.2 release build `15cb5633-0f9e-4448-bd7a-b9ff1e8e4e14` succeeded and
is the versioned rollback reference through the existing promotion workflow.

The image release gate requires the hosted `CI/test` job, a complete release image
set and smoke checks. At the earlier audited source SHA, separate launchd hook
and SonarCloud checks failed (new coverage 77.3%, threshold 80%; security C).
Those findings are not conditions enforced by the image promotion pipeline;
this audit does not dismiss or resolve them.

## Remaining foundation work before more evidence-view deletion

1. **Read-only CLI construction.** `openChainsReadBackend` calls
   `storage.NewGCPBackends`, which initializes coordinator cost synchronization.
   Missing cost metadata causes a task scan and marks counters dirty for writing.
   Add an observatory-only read constructor and test that chain reads never open
   coordinator/messaging lifecycles. CLI startup also runs local retention health
   checks; isolate those from read-only inspection. No live cloud chain-read CLI command was used
   in this audit.
2. **Reported cost normalization.** `OTLPReceiver.convertSpan` recognizes
   `gen_ai.usage.cost`, `ailang.cost.usd`, `ai.cost_usd`, `task.cost_usd`, but not
   the sampled provider's `gen_ai.usage.total_cost`. Add provider-reported cost
   with explicit precedence; distinguish missing cost from explicitly reported
   zero before token-price estimation. Test actual captured attribute shapes.
   Do not sum provider and internal duplicate representations as separate spend.
   Historical backfill requires its own reviewed scope.
3. **Fresh provenance acceptance.** After image rollout, verify newly completed
   tasks across coordinator, executor, chain/stage, messages and spans, with and
   without function tracing. Historical zero evidence is not fixed by indexes.
4. **Remaining query/shape contract.** Canonical chain summary counts/agent flow,
   transcript paging, legacy span endpoint validation, bounded aggregates and
   cursor paging remain open. Agent membership and span totals still scan.
5. **Deployment identity.** `docker/Dockerfile.dashboard` builds with only
   `-ldflags="-s -w"`; the release pipeline tags the image but does not stamp the
   binary version. Thus `/api/version` can still say `dev` on a released image.
   Propagate version/commit/build-time into the binary and test the image response.
   For this rollout, verify Cloud Build source SHA, image digests and UI assets.
6. **Approval delivery.** Verify outcome linkage and execution separately from
   queue visibility. Operator approvals and inbox state were left untouched.

No additional React evidence view should be deleted solely because these read
smoke checks pass. The next deletion needs the corresponding data acceptance.

## Final production identity and reproducible evidence

All three infrastructure builds and the application promotion completed
successfully. The twelve new indexes are READY in all three environments.

- Coordinator: `ailang-coordinator-00079-zkq`, image digest
  `sha256:f6ab8c8f1db29fbf91e2af2a6fa79321fdc62d42a0dd7fc9acc11d16eb025fdc`.
- Dashboard: `ailang-dashboard-00045-rnt`, image digest
  `sha256:6e23621db8be396aeb8253d1762bd69bef7c025f0f8d21a789c7d9bc8099df0f`.
- MCP: `ailang-mcp-00053-8sr`, versions API returns `0.35.3`.
- [Published release and signed platform assets](https://github.com/sunholo-data/ailang/releases/tag/v0.35.3).
  The downloaded Darwin ARM64 archive matched both its checksum file and GitHub's
  asset digest: `b23eafc0bcf0cf061f3bfb987c31fe539de638903867ad99648259f708a75a82`.
  That binary reports v0.35.3, commit `3f30144aedededc863ee38989cc0d8c48a2ef015`.

The existing factorial example was run with the published binary and explicit
resource label `ailang.audit_id=dashboard-production-v0.35.3`; these are standalone
smoke traces, not evidence attached to existing coordinator tasks:

- Standard trace: `5998c6541a1478bd27af8428a2860b42`.
- [Deep trace via production API](https://dashboard.ailang.sunholo.com/api/observatory/traces/70173b93d0ed34912c1217ea89b0ebe8).

Both executions returned the expected factorial results, and all spans were read
back from production storage. The deep trace includes actual serialized args and
results, not just empty field names. This verifies the standalone function path;
fresh coordinator task-to-delivery acceptance remains open. Historical evidence
in the two earlier sampled tasks remains absent. Inbox state and operator
approvals were not changed; release broadcasts/imports were not run.
