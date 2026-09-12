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

// BlocksDuplicate reports whether this task suppresses an identical request
// arriving now, given the oldest task that still counts.
//
// A task with no CreatedAt does not suppress — an unknown age is a data defect,
// and reading it as "recent" would resurrect the unbounded behaviour.
func (t *TaskRecord) BlocksDuplicate(since time.Time) bool {
	if t == nil || !DedupSuppresses(t.Status) {
		return false
	}
	if t.CreatedAt.IsZero() {
		return false
	}
	return !t.CreatedAt.Before(since)
}

// DedupSince is the cutoff to pass to FindDuplicateTask.
func DedupSince(now time.Time) time.Time { return now.Add(-DedupWindow) }
