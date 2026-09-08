# Dashboard cloud read rollout contract

Status: code repair on dev, 2026-09-08; production verification pending.
Infrastructure source: adjacent `ailang-multivac/terraform/firestore.tf`.
This document specifies query requirements; it does not provision indexes.

## Read contract

Chains sort by `created_at DESC, document ID DESC`; SQLite uses `id` as
its matching tie breaker. `CreatedAfter` is exclusive. Spans and stage span
pages sort by `start_time ASC, document ID ASC`; span time bounds are inclusive.
Filters precede paging. Offsets refer to a fixed cohort, not a snapshot under
concurrent writes. Chain backend default page size is 50; CLI retains 20.

| HTTP `/api/chains` | `ailang chains list` |
|---|---|
| `status` | `--status` |
| `source_type` | `--source` |
| `agent_id` | `--agent` |
| `workspace_id` | `--workspace` |
| `github_repo` | `--repo` |
| `limit`, `offset` | `--limit`, `--offset` |
| `since` (integer hours) | `--since` (CLI duration/date syntax) |

Example: `ailang chains list --remote gcp --limit 20 --offset 20 --json`.
Invalid supplied paging arguments fail before a backend read. Stage chat/spans
verify that the stage belongs to the chain in the URL. These chain endpoints
return structured errors: missing record 404, permission 403, authentication
401, index/precondition 503 (`query_not_ready`), transient storage 503, other
query failures 500. A 503 requires operational investigation; retries alone
cannot create an index. Empty successful lists serialize as arrays.

## Index requirements

Each row below is a collection-scope composite index. Directions are explicit;
Firestore appends the document-name tie breaker in the final ordering direction.
Confirm the generated index matches the query before rollout.

| Collection | Fields, in index order | Purpose |
|---|---|---|
| `obs_spans` | `stage_id ASC`, `start_time ASC` | Stage span pages |
| `obs_chat_messages` | `task_id ASC`, `timestamp ASC` | Task transcript |
| `obs_chat_messages` | `session_id ASC`, `timestamp ASC` | Session transcript and time bounds |
| `obs_chains` | `status ASC`, `created_at DESC` | Status chain pages; existing Terraform index |
| `obs_chains` | `source_type ASC`, `created_at DESC` | Source chain pages |
| `obs_chains` | `workspace_id ASC`, `created_at DESC` | Workspace chain pages |
| `obs_chains` | `github_repo ASC`, `created_at DESC` | Repository chain pages |
| `obs_spans` | each of `trace_id`, `task_id`, `agent_assignment_id`, `provider`, `model`, `status` ASC separately, followed by `start_time ASC` | Filtered span listing |

Concrete Terraform form for the first missing stage index, using the existing
infrastructure resource conventions:

```hcl
resource "google_firestore_index" "obs_spans_stage_time" {
  provider   = google-beta
  project    = var.project_id
  database   = google_firestore_database.default.name
  collection = "obs_spans"
  fields {
    field_path = "stage_id"
    order      = "ASCENDING"
  }
  fields {
    field_path = "start_time"
    order      = "ASCENDING"
  }
}
```

The transcript indexes use the same resource form with the collection and field
pairs above. Multiple simultaneous equality filters may use index merging, but
this table does not certify every combination. Validate the actual production
filter combinations and range bounds against READY indexes. See the official
[Firestore index overview](https://firebase.google.com/docs/firestore/query-data/index-overview).
The existing `parent_span_id ASC, start_time DESC` index does not substitute for
a stage or ascending span query. Index definitions were inspected, not applied.

## Verification before declaring production repaired

1. Review Terraform plan in the infrastructure repository; create missing indexes
   through its normal workflow and wait for READY.
2. Deploy the capture and read fixes; record exact coordinator/dashboard revisions.
3. On a fixed time-bounded cohort, compare CLI/API page IDs and filters, including
   tied timestamps, agent matches beyond page one and empty pages.
4. For a known linked task, inspect its chain, stage spans and transcript. Verify
   wrong-chain stage URLs fail and index/permission failures do not look empty.
5. Check fresh runs end to end, cost normalization, lineage completeness, missing
   evidence, approval execution and query latency/read count before deleting more
   evidence views.

## Remaining gaps

- SDK tests use in-memory RPC fixtures with preordered rows. They assert request
  filters/order/paging and compare fixed-cohort IDs to SQLite; they do not emulate
  Firestore's query engine or index readiness.
- Agent filtering can scan many stage and chain documents. Stage span totals still
  read all matching documents. Offset pages are not scalable cursor pagination.
- Firestore span workspace filters explicitly fail until a join contract exists.
- Stage transcript pagination is not implemented. Legacy `/api/observatory/spans`
  still has permissive parsing and a different error response contract.
- Cloud chain summary stage counts/agent flow and SQLite agent-filtered aggregates
  have inherited row-shape parity gaps. ID/page parity does not certify summaries.
- Capture deployment, normalized costs, aggregate timeout/empty results, historical
  provenance gaps and approval delivery remain unverified in production.
