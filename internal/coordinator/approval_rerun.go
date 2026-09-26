package coordinator

// Telling a REPLAY apart from a RE-RUN.
//
// The approval id is deterministic — apr-<task hash> — so a redelivered
// completion addresses the same row, and CreateApprovalIfAbsent makes that
// replay a no-op. That is correct and it is the whole point.
//
// But it also means two DIFFERENT executions of the same task address the same
// row, and those are not the same event. Measured 2026-09-15: eleven
// ailang-core-triage tasks were rejected, re-dispatched, ran again and produced
// new work — and their completions found an approval record already marked
// `rejected` from the previous attempt. CreateApprovalIfAbsent reported "not
// created", applyApproval read that as superseded, and each task settled into
// `pending_approval` behind a resolved record. Invisible to
// `coordinator approvals`, and unapprovable: approve refuses an already-resolved
// approval. Eleven finished pieces of work with no way to accept them.
//
// The distinguishing evidence is the WORK, not the task. A replay carries the
// same diff; a re-run carries a different one. So the approval records a work
// id, and a collision is only superseded when the work matches.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sort"
	"strings"
)

// WorkIDForApproval identifies what a completion is asking approval FOR.
//
// Built from the changed files and the diffstat rather than the patch: the
// patch embeds commit noise that differs between identical re-runs, and the
// question here is "is this the same change?", not "is this the same commit?".
//
// An empty work id means the executor produced no diff source. Those must NOT
// compare equal to each other — two no-diff completions are not evidence of the
// same work — so the caller treats "" as unknown rather than as a match.
func WorkIDForApproval(changedFiles []string, diffStat string) string {
	if len(changedFiles) == 0 && strings.TrimSpace(diffStat) == "" {
		return ""
	}
	files := append([]string(nil), changedFiles...)
	sort.Strings(files) // order varies between runs; the set does not
	h := sha256.Sum256([]byte(strings.Join(files, "\n") + "\x00" + strings.TrimSpace(diffStat)))
	return hex.EncodeToString(h[:8])
}

// ApprovalCollision is what to do when the approval row already exists.
type ApprovalCollision int

const (
	// CollisionReplay: the same work, already recorded. Do nothing — and never
	// reopen, because a human may have decided it.
	CollisionReplay ApprovalCollision = iota
	// CollisionNewWork: a later execution produced a DIFFERENT change, and the
	// standing decision was about the old one. It needs a fresh decision.
	CollisionNewWork
	// CollisionUnknown: not enough evidence to tell them apart. Treated as a
	// replay, because reopening a decision on a guess is the worse error.
	CollisionUnknown
	// CollisionStaleCard: the approval is still PENDING, but it describes the
	// previous execution's work. Nobody has decided anything, so there is no
	// decision to reopen — the evidence just has to be replaced before someone
	// reads it.
	CollisionStaleCard
)

// ClassifyApprovalCollision decides what an existing approval row means.
//
// existingWorkID is read from the stored approval; newWorkID from the
// completion now being finalised.
func ClassifyApprovalCollision(existing *ApprovalRequestRecord, newWorkID string) ApprovalCollision {
	if existing == nil {
		return CollisionUnknown
	}
	existingWorkID := workIDFromContext(existing.ContextJSON)
	sameWork := existingWorkID != "" && newWorkID != "" && existingWorkID == newWorkID
	noEvidence := existingWorkID == "" || newWorkID == ""

	// A PENDING row was read as "needs nothing: the decision has not been made,
	// and the work it describes is about to be judged either way". The second
	// half of that is false. The card is the evidence the decision is made ON,
	// and a re-run replaces the work WITHOUT replacing the card — so the
	// operator judges run 1 and merges run 2.
	//
	// Measured 2026-09-15, task-c0ca6301: the card said
	// `design_docs/planned/ailang-core-backlog.md | 7 +++++++`; what merged was
	// `design_docs/planned/ailang-core-triage/coordinator-completion-wrong-ref.md`
	// (+11), a file the card never named. Nothing was broken at the merge — the
	// branch was correct throughout. The approval simply described an execution
	// that had been superseded two minutes after it finished.
	if existing.Status == "pending" {
		if sameWork || noEvidence {
			return CollisionReplay
		}
		return CollisionStaleCard
	}

	// Either side unknown: no evidence, so do not disturb a recorded decision.
	if noEvidence {
		return CollisionUnknown
	}
	if sameWork {
		return CollisionReplay
	}
	return CollisionNewWork
}

// WorkIDFromApprovalContext is workIDFromContext for other packages (the
// Firestore store reads it to make its decision marks conditional).
func WorkIDFromApprovalContext(contextJSON string) string { return workIDFromContext(contextJSON) }

// workIDFromContext reads the work id out of a stored approval context.
//
// Absent on every approval written before 2026-09-15, which is why an unknown
// id must mean "no evidence" rather than "no work": an old record compared
// against a new completion would otherwise look like new work every time and
// reopen decisions that were correctly made.
func workIDFromContext(contextJSON string) string {
	if contextJSON == "" {
		return ""
	}
	var obj struct {
		WorkID string `json:"work_id"`
	}
	if err := json.Unmarshal([]byte(contextJSON), &obj); err != nil {
		return ""
	}
	return obj.WorkID
}
