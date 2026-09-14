package coordinator

import "time"

// Duplicate suppression policy — the ONE place that decides whether an existing
// task means an identical incoming request should be skipped.
//
// Reported 2026-09-12 by Daneel: a retry of a task that had FAILED at git push
// came back as
//
//	{"status":"deduplicated","error_msg":"Skipped: similar to recent task task-f02c283d"}
//
// The failure was the whole reason for the retry, so suppressing against it made
// that failure permanent — the only way to re-run the work was to reword it.
// Suppression is for work already in flight or already done, never for work that
// did not happen.
//
// Two other claims in that message were false and are fixed with it: the match is
// EXACT (simhash equality, both stores), not "similar", and it was unbounded in
// time, not "recent" — a fingerprint from any point in history suppressed a new
// request forever.

// DedupWindow bounds how far back a suppressing duplicate may be. Redeliveries
// and double-sends arrive within seconds to minutes; a day is generous for that
// and still lets the same request through tomorrow.
const DedupWindow = 24 * time.Hour

// DedupCandidateLimit caps how many same-fingerprint tasks a store examines
// before giving up and letting the request through.
//
// A store cannot answer "is there a SUPPRESSING duplicate" in the query, because
// the status rule lives here — so it fetches candidates and filters. SQLite takes
// the newest ones (ORDER BY created_at DESC); Firestore takes them unordered,
// since equality-plus-order-by needs a composite index that does not exist and
// the resulting error would fail the query open, which is worse than an
// unordered scan of a handful of rows.
const DedupCandidateLimit = 20

// dedupSuppressesByStatus answers, for a task already in the store, whether an
// identical new request should be skipped. A table rather than a condition, for
// the same reason as terminalByStatus: the exhaustiveness test then catches the
// next status someone adds, instead of it defaulting into a silent answer.
var dedupSuppressesByStatus = map[TaskStatus]bool{
	TaskStatusPending:         true,  // the same work is already queued
	TaskStatusQueued:          true,  //
	TaskStatusRunning:         true,  // in flight
	TaskStatusPendingApproval: true,  // done, waiting on a decision
	TaskStatusCompleted:       true,  // done, and recently
	TaskStatusNoChanges:       false, // ran and produced nothing — retrying is the point
	TaskStatusFailed:          false, // the reported bug
	TaskStatusRejected:        false, // a human refused that attempt, not every future one
	TaskStatusCancelled:       false, //
	TaskStatusDuplicate:       false, // suppressing against a suppression record chains forever
}

// DedupSuppresses reports whether an existing task in this status should
// suppress an identical new one.
//
// An UNKNOWN status does not suppress. That is the safe direction here: the cost
// of running once more is one task, and the cost of the other direction is a
// request that can never be made again.
func DedupSuppresses(s TaskStatus) bool { return dedupSuppressesByStatus[s] }

// DedupScope is the INCOMING request, as far as suppression is concerned.
//
// Content equality alone is not duplication, and treating it as such broke the
// pipeline for as long as the pipeline has existed. Measured 2026-09-14 on the
// first handoff ever to fire in production: design-doc-creator finished
// task-08032ebc, the approval dispatched sprint-planner, and the resulting task
// was suppressed ONE SECOND later against task-08032ebc itself —
//
//	{"status":"deduplicated","original_task_id":"task-08032ebc",
//	 "original_task_status":"completed"}
//
// because sendAgentHandoffMessage embeds the predecessor's request verbatim
// ("Original Request: %s"), so every handoff simhashes to its own parent, and a
// parent at handoff time is always `completed`, which suppresses. A handoff
// could therefore NEVER dispatch. That is a second, independent bug behind the
// approval-path fix of 2026-09-07 — which is why fixing that one did not make
// the pipeline run.
type DedupScope struct {
	// Since is the oldest existing task that still counts.
	Since time.Time
	// AgentID is the agent the new task is for. The same words sent to a
	// different agent are a different job — a pipeline stage, or a package
	// request fanned to several package agents — not a repeat.
	AgentID string
	// ParentTaskID is set when the new task descends from another. A stage
	// never duplicates the stage it follows.
	ParentTaskID string
}

// BlocksDuplicate reports whether this existing task suppresses the incoming
// request described by scope.
//
// A task with no CreatedAt does not suppress — an unknown age is a data defect,
// and reading it as "recent" would resurrect the unbounded behaviour.
func (t *TaskRecord) BlocksDuplicate(scope DedupScope) bool {
	if t == nil || !DedupSuppresses(t.Status) {
		return false
	}
	if t.CreatedAt.IsZero() {
		return false
	}
	if t.CreatedAt.Before(scope.Since) {
		return false
	}
	// Different agent, different work. Only decided when BOTH sides name an
	// agent: an empty side is unknown, not "matches anything", and guessing
	// either way here would silently change which requests are suppressed.
	if scope.AgentID != "" && t.AgentID != "" && t.AgentID != scope.AgentID {
		return false
	}
	// The stage this one follows.
	if scope.ParentTaskID != "" && t.ID == scope.ParentTaskID {
		return false
	}
	return true
}

// DedupSince is the cutoff for a DedupScope.
func DedupSince(now time.Time) time.Time { return now.Add(-DedupWindow) }

// DedupScopeFor builds the scope for a task about to be created.
func DedupScopeFor(t *TaskRecord, now time.Time) DedupScope {
	scope := DedupScope{Since: DedupSince(now)}
	if t != nil {
		scope.AgentID = t.AgentID
		scope.ParentTaskID = t.ParentTaskID
	}
	return scope
}
