# Two new PR arrivals — read-only ownership review

Actual source baseline: origin/dev `65253e86e68be251eecd796d92327e4d4828c234`. The inspected registry/telemetry/run source files are byte-identical between 749045717 and 65253e86. No source, PR, comment, card, message or acknowledgment was changed. Only these two arrivals were examined; no issue backlog rerun.

| PR | Exact head / branch | Existing owner | Live approval / disposition |
|---|---|---|---|
| [2010](https://github.com/sunholo-data/ailang/pull/2010) registry record literals | `74a0f4406d21b440bfecb35e074ac70033be486d`, `coordinator/task-789e5f9f` | PROD `task-789e5f9f`, agent `ailang-core-triage`, status `pending_approval` | `apr-789e5f9f`, type `merge`, **agent_actionable:false** — operator hold |
| [2009](https://github.com/sunholo-data/ailang/pull/2009) ambient trace export | `c263d2833a05952d92a3e0a4fa0d63f7285359d4`, `coordinator/task-80a1ffa8` | PROD `task-80a1ffa8`, agent `ailang-core-triage`, status `pending_approval` | `apr-80a1ffa8`, type `merge`, **agent_actionable:false** — operator hold |

Both agents finished their triage stage; neither is an unowned implementation request or a currently running executor. Their tasks are present on production `ailang-multivac`, absent from the current test/dev active-task snapshots. Task chain IDs are respectively `bf7355e3-acdf-41a9-822f-f56a58aaadc2` and `0536710d-f949-48f5-8618-33eb5c65dc1b`. Each PR adds one small triage Markdown document, recommends **design-doc**, and includes no implementation. Both authors are sunholo-voight-kampff; both base at749045717. All three discussion endpoints were read in full: each PR has 0 top-level comments, 0 inline comments and 0 formal reviews.

The live cloud approvals response reports no controller authority and explicitly reserves both cards to the operator. Do not merge/approve indirectly through gh, dispatch another agent, or infer that merging these triage rows approves their proposed semantics. Parent can add both to the held set (22 total under its existing 20-held accounting).

## #2010 diagnostic accuracy and existing scope

Confirmed source cause: cmd/registry-validator/tool_names.go Pattern2 scans every `name: "X"` literal in all .ail source; only flat records carrying a `value` sibling are exempt. The actual std/zip entry type is `{ name:string, content:string }`, so `[Content_Types].xml` is picked up and rejected by the provider-safe name regexp. handlePublish calls this gate before compile checking (current line 251; triage cites stale 243). CLI pkg_publish.go/pkg_quality.go do not call this validator-local function, supporting the reported server/local parity gap. Its return tuple lacks file/position, so file:line needs an intentional shared diagnostic contract.

Independently read the cited official package [before/after comparison](https://github.com/sunholo-data/ailang-packages/compare/d7e2df3...65e700d): exactly 1 commit and 1 file, packages/specstore/xlsx.ail; 9 insertions / 6 deletions add the part(path,body) helper and replace the literal records. This verifies the syntactic rewrite, while the original live refusal text was not preserved and no public publish was reattempted.

This is already the root cause in planned/ailang-core-triage/registry-validator-tool-name-overmatch.md and the 2026-09-15 row in planned/ailang-core-backlog.md. Implemented v0_20_1/m-ext-author-dx.md documents the provider-safe gate's origin. The new row correctly adds dry-run/quality parity, PUB diagnostic and source position asks. Future combined design should supersede both triage rows; another shape exemption would duplicate the existing pattern. No approved implementation design/sprint was identified for this expanded fix.

## #2009 diagnostic accuracy and correction before future design

Confirmed current source: internal/config/cloud.go TraceProjectFromEnv chooses OTLP_GOOGLE_CLOUD_PROJECT then ambient GOOGLE_CLOUD_PROJECT; telemetry/config_test.go explicitly pins the ambient fallback. platform/otel/init.go passes that project into exporter initialization; Init's legacy path logs warnings without an active-destination notice. fs.path, net.url and process.command are added directly in telemetry/effect_spans.go and are not covered by collector AILANG_TRACE_VALUES redaction. Changing export selection/off precedence and recorded attributes is an environment/privacy contract change requiring the design gate.

**Substantive triage correction:** point 3 says tier-off means a plain ailang run has no spans to export. Current cmd/ailang/main_run_exec.go initializes exporters at 37–44 and **unconditionally starts the CLI root span at 71–80**, before runner/run.go resolves the collector's tier-off around 547–560. The root span name includes the filename and attribute `file.path` also contains it. AILANG_TRACE=off therefore leaves at least this path-bearing root span eligible for export; the defect is broader than hypothetical independent AI SDK spans. Future off-switch and redaction coverage must include root span names/file.path, not only collector values/fs.path. No live confidential data or telemetry egress repro was attempted for this review.

Minor accuracy: this newly created Oct 29 triage row has a stale Date 2026-09-30. Its wording calls the Cloud Trace exporter archived, but the exact official pinned 1.38.0 [README](https://github.com/GoogleCloudPlatform/opentelemetry-operations-go/blob/exporter/trace/v1.38.0/exporter/trace/README.md) says it is deprecated and **will be archived after Jan 1, 2027**; that is future relative to today. go.mod does pin Cloud Trace exporter 1.38.0 and OTLP HTTP 1.47.0. Presence of an OTLP dependency alone does not validate ADC/project endpoint migration; keep that behind design rather than treating it as an automatic config patch. No existing approved design for ambient export reversal was identified in the triage's cited closest collection-tier/cloud-resolution work.

## Evidence and safe cloud read

Raw PR bodies/full file patches/comments/inline/reviews are in new-arrivals/pr{2009,2010}*.json. Live cards: new-arrivals/prod-approvals.json; task snapshots: smt-preflight-{prod,test,dev}-active-tasks.json (fresh root/language snapshots 13:40 local). Exact package comparison and official dependency README are banked alongside them.

Supported read-only live-card command:

```sh
env AILANG_STORAGE_COORDINATOR=gcp AILANG_CLOUD_PROJECT=ailang-multivac \
  /tmp/ailang-backlog-20261029/pr-1771/bin/ailang coordinator approvals --remote gcp --json
```

`coordinator pending --remote gcp` is explicitly rejected by this binary because pending is local-only; it was not used as cloud evidence. No global configuration changes or controller override were made.
