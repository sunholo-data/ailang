# Observatory ListSpans Recency Truncation

**Status**: Planned
**Target**: v0.38.10 (dashboard/observatory infra — not gated to a language release)
**Priority**: P1
**Estimated**: 2 days
**Dependencies**: None

## Axiom Compliance

**Canonical reference:** [Design Axioms](/docs/references/axioms)

This is dashboard/observability infrastructure, not a language or runtime change — it does not
touch the compiler, typechecker, or execution semantics. Most axioms score 0 (no applicable
surface). The two that apply are the ones about silent-wrongness, which is exactly this bug's
shape.

### Axiom Scoring

| Axiom | Score | Justification |
|-------|-------|---------------|
| A1: Determinism | 0 | No language/runtime surface; query results become MORE deterministic (caller-selectable order) but this doesn't touch AILANG program semantics |
| A2: Replayability | 0 | No trace/replay semantics affected |
| A3: Effect Legibility | 0 | No effect system surface |
| A4: Explicit Authority | 0 | No capability/authority surface |
| A5: Bounded Verification | 0 | No type-checking surface |
| A6: Safe Concurrency | 0 | No concurrency model changes |
| A7: Machines First | 0 | Not a language ergonomics change either direction |
| A8: Minimal Syntax | 0 | No syntax involved |
| A9: Cost Visibility | +1 | A truncated read currently looks identical to a complete one; this makes "you didn't see everything" visible instead of silently absorbed |
| A10: Composability | 0 | No composition surface |
| A11: Structured Failure | +1 | Converts a silent wrong-answer (stale data mistaken for current data) into a detectable, typed condition (a `truncated` signal) — directly the project's "no silent fallbacks" principle (CLAUDE.md §2) applied to a read path instead of a config fallback |
| A12: System Boundary | 0 | No boundary-crossing surface |

**Net Score: +2** → **Decision: Move forward**

### Hard Violation Check

- [x] A1 (Determinism): No implicit nondeterminism introduced — new order param is explicit and caller-selected
- [x] A3 (Effects): No hidden side effects — read-only API, no behavior change for existing callers who don't opt in
- [x] A4 (Authority): No ambient access granted
- [x] A7 (Machines First): N/A — no human/machine ergonomics trade-off here

### Decision Thresholds

Net Score +2 → ✅ Proceed to implementation.

## Problem Statement

`ListSpans` — the shared read path behind `/api/observatory/spans` and every internal caller —
orders results `start_time ASC` and then applies a hard `Limit`, on **both** storage backends:

- Firestore (prod/dev/test): `q = q.OrderBy("start_time", firestore.Asc).OrderBy(firestore.DocumentID, firestore.Asc).Offset(opts.Offset)` then `.Limit(opts.Limit)` — [`internal/storage/firestore/observatory_spans.go:72`](../../internal/storage/firestore/observatory_spans.go#L72), verified by reading the function (lines 40-92).
- SQLite (local/rig): `query += " ORDER BY start_time ASC, id ASC"` — [`internal/observatory/store_spans.go:302`](../../internal/observatory/store_spans.go#L302), verified via `grep -n "ORDER BY" internal/observatory/store_spans*.go`.

When a caller queries a **wide time window with no other scoping filter** (no `trace_id` /
`task_id` / `agent_assignment_id`) and the window's true span count exceeds `Limit`, the response
is silently truncated to the **oldest** matching spans, not the newest. There is no signal in the
response that truncation happened — a capped page and a complete page are byte-for-byte
indistinguishable in shape. A consumer reading "the newest span in this response" is actually
reading "the newest span that fit before the limit ran out," which can be arbitrarily older than
"now."

**Current State — reproduced during incident investigation, 2026-09-16:**

- Live measurement: querying `start_after=2026-09-16T05:00:00Z` with `limit=1000` against prod
  (`https://dashboard.ailang.sunholo.com`) returned spans covering only `05:30:15Z` →
  `06:10:06Z` — the 1000-doc cap was exhausted ~40 minutes into the window, at current background
  volume of roughly 25 spans/minute (dominated by `ailang-coordinator`'s own Firestore
  instrumentation, not the payload traffic anyone actually wants to see).
- `scripts/check_pi_wire_budget.sh` queried a `-2h .. tomorrow` window with `limit=1000` to find
  the newest OpenRouter/pi span. At the volume above, that window holds well over 1000 total
  spans, so the script's watermark was structurally stuck ~20-40 minutes stale on every run. It
  read `INCONCLUSIVE` three times in a row (`no new pi span ingested within 90s`) while Cloud Run
  logs showed spans being received and **"stored span successfully"** continuously the entire
  time, including seconds before each check ran. No outage occurred; the read path could not see
  data that unquestionably existed. Fixed narrowly in commit `471ecab6f` (window shrunk to 15
  minutes) — that patch addresses only this one caller and only at today's volume; it will
  reproduce the same failure if background span volume roughly doubles.
- `cmd/ailang/dashboard_tools.go:409` (`ailang dashboard health`, function `dashboardHealthCommand`)
  queries `/api/observatory/spans?limit=1000` with **no time window at all**, then locally filters
  for `start_time > now-24h` to print a "Recent Activity (24h)" count. Verified by reading
  [`cmd/ailang/dashboard_tools.go:405-426`](../../cmd/ailang/dashboard_tools.go#L405). With no
  `start_after`, ascending order returns the **oldest 1000 spans ever stored** (bounded only by
  retention). Once a deployment has accumulated more than 1000 spans in its retention window —
  true of prod today — this command's "recent activity" count is silently wrong, not merely
  occasionally stale: it is counting how many of the oldest-surviving spans happen to fall in the
  last 24 hours, which trends toward zero as the deployment ages.
- `cmd/ailang/observatory_backfill.go:307` queries per-assignment with a wide
  `StartAfter`/`StartBefore` window and `Limit: 100000`, with an explicit comment: *"Use large
  limit to ensure we get all spans (pagination would be complex)"* — verified by reading
  [`cmd/ailang/observatory_backfill.go:300-320`](../../cmd/ailang/observatory_backfill.go#L300).
  Same failure shape, currently lower-probability only because 100,000 is a much larger margin
  than 1,000 at today's volume. Because ordering is ascending, if a window's true span count ever
  exceeds the limit, the spans silently dropped are the ones closest to `endTime` — i.e. exactly
  the most-recently-generated unlinked spans a reconciliation/backfill tool exists to catch.

**Audited and confirmed safe** (same `ListSpans`/`ListSpansLightweight`, but scoped by
`TraceID`/`TaskID`/`AgentAssignmentID`, which keeps result-set size naturally small — a single
trace or task does not accumulate thousands of spans):
`internal/observatory/hierarchy.go:132,143,175`, `internal/observatory/outliers.go:34`,
`internal/server/handlers_controlplane_task_hierarchy.go:145`, `cmd/ailang/trace_local.go:195`,
`cmd/ailang/chains_tree.go:251`, `cmd/ailang/observatory_hierarchy.go:361,419`, and the UI's
`ui/src/features/controlplane/hooks/useTraceData.ts` (always queries by `trace_id` or `task_id`).
`internal/observatory/api_enrichment.go` (`/api/observatory/spans/enriched`) shares the same
underlying risk in principle — it accepts arbitrary caller-supplied `SpanListOptions` — but every
current caller of it (the UI) scopes by `trace_id`/`task_id`, so it is not exposed today. Verified
via `grep -rn "\.ListSpans(" --include="*.go" internal cmd` and reading each call site
(2026-09-16).

**Impact:**

- Anyone diagnosing ingest health, running a "is the pipeline stuck?" check, or reading
  "recent activity" from the dashboard CLI can reach a false "it's stalled" conclusion while the
  system is healthy — as this incident demonstrates, that costs real investigation time and can
  misdirect follow-up work (this incident's `check-pi-wire-budget` failures were initially read as
  a possible OpenRouter ingest outage).
- The failure gets *worse*, not better, as the deployment matures: retention keeps historical
  spans around, background instrumentation volume only grows, and the gap between "oldest 1000
  spans in a window" and "now" widens over time. `dashboard_tools.go`'s health check is likely
  already silently near-zero on prod today, not just fragile.
- It is a trap for future tooling: any new script or dashboard view built against
  `/api/observatory/spans` with a wide window and a limit — the natural way to ask "what's
  recent?" — will reproduce this bug on first use, with no error to catch it in review or testing
  at low volume.

## Goals

**Primary Goal:** Make truncation on `ListSpans` a detectable condition instead of an invisible
one, and give callers that need "most recent N" a way to actually get it regardless of window
width or background volume.

**Success Metrics:**
- A caller of `/api/observatory/spans` (or the internal `ListSpans`) can distinguish "I saw every
  matching span in this window" from "I was capped, do not trust the tail" without guessing from
  volume.
- A caller that wants "the newest N spans matching these filters" can request that directly,
  without picking a window width narrow enough to dodge the cap (the failure mode that bit
  `check_pi_wire_budget.sh`).
- `ailang dashboard health`'s "Recent Activity (24h)" count is correct on a deployment with more
  than `limit` spans in its retention window.
- `observatory_backfill.go`'s per-assignment span lookup cannot silently drop the
  most-recently-generated spans in an oversized window.

## High-Impact Decisions

| Decision | Why High Impact | Chosen By | Deadline | Change Cost |
|----------|-----------------|-----------|----------|-------------|
| Add an opt-in `order` param (`asc` default / `desc`) to `SpanListOptions` and both backend implementations | Existing ASC-order callers use `Offset` for deterministic forward pagination (e.g. hierarchy building); changing the default would silently reorder their results. Must stay additive. | human | design | med |
| Surface truncation via an HTTP response header (e.g. `X-Observatory-Truncated: true`), not a JSON envelope change | `handleListSpans` currently returns a bare JSON array; every existing consumer (UI, CLI, this script) decodes it as `[]Span`. Wrapping it in `{spans:[...], truncated:...}` is a breaking contract change for all of them at once. | human | design | high (if envelope chosen instead) / low (if header chosen) |
| Fix `dashboard_tools.go`'s health command and `observatory_backfill.go` to use the new `desc` order (or the truncation signal) rather than leaving them on the same broken pattern | These are the two currently-live broken/at-risk callers found by the audit above; landing the primitive without using it anywhere leaves both bugs in place | agent | compile | low |

### Design Freeze

- [ ] Truncation is signaled via response header, not a JSON body shape change (rejects the
      envelope alternative — confirm before implementation, since reversing this after callers
      exist means a second migration)
- [ ] `order` defaults to `asc` (current behavior) on both backends; `desc` is opt-in only

## Solution Design

### Overview

Two additive, backward-compatible changes to the shared `ListSpans` primitive, plus fixes to the
two callers already proven broken by this bug:

1. **Detect and surface truncation.** After executing a bounded query, compare
   `len(result) == opts.Limit` (when `opts.Limit > 0`). If true, the caller asked for more spans
   than fit — that's the necessary-and-sufficient condition for "this page might not include
   everything in the window." Return that as a header on the HTTP response
   (`X-Observatory-Truncated: true`) and as a return value from the Go-level `ListSpans` callers
   that want it (e.g. a second bool return value, or a field the caller can check — kept as a
   Deferred Decision, see below).
2. **Add an opt-in descending order.** `SpanListOptions.Order` (new field: `"" | "asc" | "desc"`,
   empty = `asc` for backward compatibility). On Firestore, swap `firestore.Asc` for
   `firestore.Desc` on both the `start_time` and tiebreaker `OrderBy` clauses when `desc` is
   requested. On SQLite, swap `ORDER BY start_time ASC, id ASC` for `ORDER BY start_time DESC, id
   DESC`. `Offset` continues to mean "skip this many from the front of whichever order was
   requested" — unchanged semantics, just applied to the other direction.
3. **Fix the two broken callers.**
   - `dashboard_tools.go`'s health command: pass `StartAfter: now-24h` (bound the query instead of
     fetching unfiltered history) — this alone fixes it even without the new `order` param, since
     the existing bug requires an unbounded-or-oversized window, and 24h is itself an oversized
     window at current volume. Add `Order: "desc"` and stop once enough spans are seen, or simply
     rely on the narrower `StartAfter` bound — either resolves it; pick during implementation
     (Deferred Decision).
   - `observatory_backfill.go`: use `Order: "desc"` per per-assignment query so a hit against the
     100,000 cap drops the *oldest* spans in that assignment's window (which have presumably
     already been reconciled in a prior run) rather than the newest (which are the ones this tool
     exists to catch), and check the new truncation signal to log a warning instead of silently
     reporting completeness.

### Architecture

**Components:**
1. **`SpanListOptions.Order`** (new field, `internal/observatory/store_spans.go`) — plumbed
   through the HTTP query param `order` in `handleListSpans`
   ([`internal/observatory/api_spans.go:18-54`](../../internal/observatory/api_spans.go#L18)) and
   into both backend implementations.
2. **Truncation detection** — a single `len(result) == limit` check, added once in each backend's
   `ListSpans`, no new query logic required.
3. **Two caller fixes** — bound `dashboard_tools.go`'s window; flip `observatory_backfill.go` to
   `desc` and check the truncation signal.

### Implementation Plan

**Phase 1: Add `Order` to `SpanListOptions` and both backends** (~4 hours)
- [ ] Add `Order string` field to `SpanListOptions` (`internal/observatory/store_spans.go`)
- [ ] Firestore: honor `Order` in `ObservatoryStore.ListSpans`
      (`internal/storage/firestore/observatory_spans.go:72`)
- [ ] SQLite: honor `Order` in the equivalent query builder (`internal/observatory/store_spans.go:302`)
- [ ] Parse `order` query param in `handleListSpans` (`internal/observatory/api_spans.go`),
      validate against `{"", "asc", "desc"}`, reject anything else with 400 (no silent fallback to
      `asc` on a typo'd value)
- [ ] Unit tests: same fixture data, assert `asc` vs `desc` return reversed order; assert an
      invalid `order` value 400s

**Phase 2: Truncation signal** (~3 hours)
- [ ] Both backends: after the query, compute `truncated := opts.Limit > 0 && len(result) == opts.Limit`
- [ ] `handleListSpans`: set `X-Observatory-Truncated: true` when truncated
- [ ] Unit test: a fixture with more rows than `Limit` sets the header; fewer rows does not

**Phase 3: Fix the two broken callers** (~5 hours)
- [ ] `dashboard_tools.go` `dashboardHealthCommand`: bound the spans query with `StartAfter: now-24h`
      (stop fetching unfiltered history); verify against a local/dev deployment with >1000 total
      spans that the recent-activity count is no longer near-zero
- [ ] `observatory_backfill.go`: set `Order: "desc"` on the per-assignment `ListSpans` call; log a
      warning (not silent) when the truncation header/signal fires
- [ ] Manual verification against prod read replica or dev: confirm both commands now report
      numbers consistent with live Cloud Run logs

### Files to Modify/Create

**Modified files:**
- `internal/observatory/store_spans.go` — add `Order` field to `SpanListOptions`; SQLite `ORDER BY`
  branch on `Order`; truncation check, ~30 LOC
- `internal/storage/firestore/observatory_spans.go` — `Order`-conditional `OrderBy` direction;
  truncation check, ~20 LOC
- `internal/observatory/api_spans.go` — parse/validate `order` query param; set truncation header,
  ~20 LOC
- `cmd/ailang/dashboard_tools.go` — bound the health command's spans query, ~10 LOC
- `cmd/ailang/observatory_backfill.go` — `Order: "desc"`, truncation-aware logging, ~10 LOC
- `.claude/rules/cloud-endpoints.md` — add a short entry documenting the ASC+Limit truncation trap,
  alongside the existing OTLP/JSON ID and Firestore-backend entries, so it isn't rediscovered by
  the next tool built against this endpoint

## Examples

### Example 1: `check_pi_wire_budget.sh`-shaped check, after this lands

**Before (the actual incident):**
```
GET /api/observatory/spans?limit=1000&start_after=<now-2h>&start_before=<tomorrow>
→ 200 OK, 1000 spans, newest start_time = 05:40Z (stale — silently truncated)
→ script concludes: "no new pi span ingested within 90s" (false)
```

**After:**
```
GET /api/observatory/spans?limit=50&order=desc&start_after=<now-2h>&start_before=<tomorrow>
→ 200 OK, 50 spans, newest-first, first element IS the true newest span regardless of
  how much background volume exists in the window
→ (if the caller still wants the old asc+wide-window shape for some reason, a
  X-Observatory-Truncated: true header tells it not to trust the tail)
```

### Example 2: `ailang dashboard health`

**Before:** "Recent Activity (24h): 0" on a deployment with plenty of activity, because the
underlying query fetched the oldest 1000 spans in the server's entire retention window and none of
them happen to be recent.

**After:** the query is bounded to `start_after=now-24h`, so every returned span is actually within
the 24h window being reported on.

## Success Criteria

- [ ] `order=desc` returns spans newest-first on both Firestore and SQLite backends (unit test)
- [ ] `order` param rejects invalid values with 400, never silently falls back to `asc` (unit test)
- [ ] Truncated responses carry `X-Observatory-Truncated: true`; non-truncated responses do not (unit test)
- [ ] `ailang dashboard health`'s recent-activity count is bounded to the stated window, verified
      against a deployment with >1000 total spans
- [ ] `observatory_backfill.go` uses `desc` order and logs (does not silently swallow) a truncation hit
- [ ] All tests passing
- [ ] `.claude/rules/cloud-endpoints.md` updated with the truncation trap
- [ ] `docs/internal/dashboard-cloud-read-indexes.md` updated with the new opt-in `desc` order and truncation header, so the read contract stays authoritative
- [ ] Existing `asc`-order callers (hierarchy.go, outliers.go, chains_tree.go, etc.) unaffected —
      no behavior change when `Order` is left unset

## Testing Strategy

**Unit tests:**
- `internal/observatory/store_spans_test.go` (or nearest existing span test file): seed >`Limit`
  spans in a window, assert `asc` returns oldest-first, `desc` returns newest-first, and the
  truncation signal fires only when the result set was actually capped
- Firestore backend: same shape, using existing Firestore emulator/test harness if present in
  `internal/storage/firestore/*_test.go`

**Integration tests:**
- `internal/observatory/api_test.go`: exercise `handleListSpans` with `order=desc` and an invalid
  `order` value end-to-end through the HTTP handler

**Manual testing:**
- Against dev (`ailang-dev-dashboard`), reproduce the original incident shape (wide window, high
  background volume) with `order=asc` (confirm truncation still occurs and is now signaled), then
  repeat with `order=desc` (confirm the true newest span is returned regardless of volume)
- Run `ailang dashboard health` against a deployment known to have >1000 total spans; confirm the
  recent-activity count is no longer implausibly low

## Deferred Decisions

- Whether Go-level callers of `ListSpans` (not just the HTTP API) receive the truncation signal as
  a second return value, an exported field on a result struct, or purely as the HTTP header (only
  relevant to HTTP consumers) — agent may choose during Phase 2, guided by what's least invasive
  to the existing `(spans []*Span, err error)` signature used by ~15 call sites
- Exact wording/format of the `.claude/rules/cloud-endpoints.md` addition — agent may choose,
  following the existing entries' style (a subheading, a one-line summary, a "measured" date)
- Whether `dashboard_tools.go`'s health command also adopts `order=desc` in addition to bounding
  the window, or relies on the bound alone — agent may choose; the bound alone is sufficient to
  fix the reported bug

## Non-Goals

- **Not** changing the default order of `ListSpans` — every existing unscoped-window caller that
  relies on ascending order for pagination (`Offset`-based forward iteration) must see zero
  behavior change.
- **Not** migrating every caller of `/api/observatory/spans` to the new params — only the two
  callers proven broken by this audit (`dashboard_tools.go`, `observatory_backfill.go`) are in
  scope. The scoped (trace/task/assignment) callers are unaffected by the underlying bug and don't
  need touching.
- **Not** a cursor-based pagination redesign of the observatory read API. `Offset`+`Limit` with a
  caller-selectable order is the minimal fix that closes the actual failure mode; a full
  cursor/streaming API is a larger surface this doc does not attempt to justify.
- **Not** raising the default `Limit` or changing retention. The bug is about *silent* truncation,
  not about the existence of a limit — limits are fine as long as the caller can tell when one was
  hit.

## Timeline

**Day 1** (~7 hours): Phase 1 (Order param, both backends, tests) + Phase 2 (truncation signal, tests)

**Day 2** (~5 hours): Phase 3 (fix both broken callers, manual verification against a live
deployment, update `.claude/rules/cloud-endpoints.md`)

**Total: ~2 days**

## Risks & Mitigations

| Risk | Impact | Mitigation |
|------|--------|-----------|
| A `desc`-order Firestore query needs a composite index that doesn't exist yet (this repo has been bitten by missing Firestore indexes before, per `docs/internal/dashboard-cloud-read-indexes.md`) | Med — query would 500 in prod until an index is added via Terraform | The index Terraform lives in the **separate `ailang-multivac` repo** (`terraform/dashboard_read_indexes.tf` there, not here — verified: no such path exists in this repo; `docs/internal/dashboard-cloud-read-indexes.md:5-6` names it as "adjacent `ailang-multivac/...`"). Check that repo during Phase 1 and add the `desc` composite index alongside the existing `asc` ones before deploying past dev; this is a cross-repo dependency, flag it in the sprint plan |
| The `start_time ASC, document ID ASC` order is a **documented read contract**, not an accidental default (`docs/internal/dashboard-cloud-read-indexes.md:11-12`: "Spans and stage span pages sort by `start_time ASC, document ID ASC`") | Low — this doc's design already keeps `asc` as the unconditional default and only adds `desc` as opt-in, so the contract is preserved | Update `docs/internal/dashboard-cloud-read-indexes.md` alongside the code change to document the new opt-in `desc` mode and the truncation header, so the contract doc stays authoritative |
| Existing callers construct `SpanListOptions{}` by zero-value and might accidentally send `Order: ""` where Go's zero-value ambiguity matters | Low | Zero-value `""` is defined to mean `asc` (current behavior) — no caller needs to change to stay correct |
| Fixing `observatory_backfill.go`'s order changes what it drops on truncation, but doesn't eliminate the cap itself — a single assignment with >100,000 spans in its window is still exposed | Low (100k margin is currently ~100x current volume) | Out of scope for this doc; the truncation signal at least makes it detectable instead of silent, which is the stated goal |

## Related Documents

**Implemented (may inform design):**
- [design_docs/implemented/v0_7_0/observatory-architecture.md](../implemented/v0_7_0/observatory-architecture.md) — original observatory design; establishes the dual-backend (SQLite/Firestore) shape this doc must preserve compatibility with
- [design_docs/implemented/v0_7_0/m-task-hierarchy-sprint-plan.md](../implemented/v0_7_0/m-task-hierarchy-sprint-plan.md)
- [design_docs/implemented/v0_7_0/m-db-cleanup.md](../implemented/v0_7_0/m-db-cleanup.md)

**Planned (checked for overlap — none found relevant):**
- [design_docs/planned/m-list-accessor-api.md](m-list-accessor-api.md) — AILANG-language list
  accessors, unrelated (different "list" — language data structure vs. this HTTP API)
- [design_docs/planned/m-list-repr-spike.md](m-list-repr-spike.md) — same, unrelated
- [design_docs/planned/v0_29_0/m-cascade-observability.md](v0_29_0/m-cascade-observability.md) —
  reviewed; covers cascade-level observability concepts, not the span-list read path

## References

- [Design Axioms](/docs/references/axioms) - The 12 non-negotiable principles
- `.claude/rules/cloud-endpoints.md` — existing documented observatory traps (OTLP/JSON ID
  decoding, Firestore-vs-SQLite backend selection); this doc's fix should add an entry here
- [`docs/internal/dashboard-cloud-read-indexes.md`](../../docs/internal/dashboard-cloud-read-indexes.md) —
  the existing read contract that documents `start_time ASC, document ID ASC` as the deliberate
  sort/tiebreak for spans; confirms the default order this doc preserves is intentional, and is
  the doc to update alongside the code change (needs an entry for the new opt-in `desc` order and
  the truncation header)
- Incident evidence: prod Cloud Run logs for `ailang-dashboard` (`ailang-multivac`,
  `europe-west1`), 2026-09-16 05:00-07:05Z — spans stored successfully throughout; no deploy, no
  5xx, confirming the read path (not ingest) was the fault
- `scripts/check_pi_wire_budget.sh` commit `471ecab6f` — the narrow, single-caller patch that
  motivated this systemic doc

## Future Work

- If background instrumentation volume (currently dominated by `ailang-coordinator`'s own
  Firestore span-per-query noise) keeps growing, consider whether that noise should be sampled or
  filtered at write time rather than only worked around at read time — out of scope here since
  it's a volume/cost question, not a correctness one.

---

**Document created**: 2026-09-16
**Last updated**: 2026-09-16
