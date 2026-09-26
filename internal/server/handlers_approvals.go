package server

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/sunholo-data/ailang/internal/coordinator"
	"github.com/sunholo-data/ailang/internal/display"
)

// UIApproval is the format expected by the frontend Approval interface
type UIApproval struct {
	ID              string  `json:"id"`
	ThreadID        string  `json:"thread_id"`         // Maps to task_id
	ThreadTitle     string  `json:"thread_title"`      // Task title
	InstanceID      string  `json:"instance_id"`       // Agent ID
	CreatedAt       int64   `json:"created_at"`        // Unix timestamp
	EffectDeltaJSON string  `json:"effect_delta_json"` // Empty for coordinator approvals
	Proposal        string  `json:"proposal"`          // Description
	Impact          string  `json:"impact"`            // low/medium/high
	EstimatedCost   float64 `json:"estimated_cost"`    // From task if available
	Status          string  `json:"status"`            // pending/approved/rejected
	ReviewedBy      string  `json:"reviewed_by,omitempty"`
	ReviewedAt      *int64  `json:"reviewed_at,omitempty"`
	ReviewNotes     string  `json:"review_notes,omitempty"`
	// Multi-channel fields
	RequestType  string `json:"request_type,omitempty"` // merge/handoff
	TaskID       string `json:"task_id,omitempty"`      // Direct task reference
	WorktreePath string `json:"worktree_path,omitempty"`
	BranchName   string `json:"branch_name,omitempty"`
	Workspace    string `json:"workspace,omitempty"` // Source workspace (e.g., "/Users/mark/dev/sunholo/stapledons_voyage")
	Summary      string `json:"summary,omitempty"`   // Short summary for display
	// Display info for consistent rendering
	StatusDisplay *display.StatusDisplay `json:"status_display,omitempty"`

	// #921 — the fields the card exists to show. Evaluation is the sprint-
	// evaluator verdict from the approval record (PASS/FAIL/UNAVAILABLE, may
	// carry a "(late…)" suffix). DiffStat and ChangedFiles come from the diff
	// the OWNING coordinator persisted at completion — the dashboard cannot
	// stat a worktree on another machine, and rendered "Files (0)" for every
	// local-lane task until it stopped trying.
	Evaluation   string   `json:"evaluation,omitempty"`
	DiffStat     string   `json:"diff_stat,omitempty"`
	ChangedFiles []string `json:"changed_files,omitempty"`
}

// mapCoordinatorApprovalToUI maps a coordinator ApprovalRequestRecord to the UI format
func (s *Server) mapCoordinatorApprovalToUI(ctx context.Context, rec *coordinator.ApprovalRequestRecord) UIApproval {
	statusDisplay := display.ApprovalStatusDisplay(rec.Status)
	approval := UIApproval{
		ID:              rec.ID,
		ThreadID:        rec.TaskID,
		InstanceID:      "coordinator",
		CreatedAt:       rec.CreatedAt.UnixMilli(),
		EffectDeltaJSON: "{}",
		Proposal:        rec.Description,
		Impact:          "medium",
		Status:          rec.Status,
		RequestType:     rec.Type,
		TaskID:          rec.TaskID,
		Summary:         rec.Description,
		StatusDisplay:   &statusDisplay,
	}

	// Set impact based on type
	if rec.Type == "handoff" {
		approval.Impact = "low"
	} else if rec.Type == "merge" {
		approval.Impact = "high"
	}

	approval.Evaluation = rec.Evaluation
	if rec.ContextJSON != "" {
		var cctx struct {
			DiffStat     string   `json:"diff_stat"`
			ChangedFiles []string `json:"changed_files"`
		}
		if err := json.Unmarshal([]byte(rec.ContextJSON), &cctx); err == nil {
			approval.DiffStat = cctx.DiffStat
			approval.ChangedFiles = cctx.ChangedFiles
		}
	}

	if rec.ResolvedBy != "" {
		approval.ReviewedBy = rec.ResolvedBy
	}
	if rec.ResolvedAt != nil {
		ts := rec.ResolvedAt.UnixMilli()
		approval.ReviewedAt = &ts
	}

	// Enrich with task info if available
	if s.taskEventStore != nil {
		task, err := s.taskEventStore.GetTask(ctx, rec.TaskID)
		if err == nil && task != nil {
			approval.ThreadTitle = task.Title
			approval.InstanceID = task.AgentID
			if task.WorktreePath != "" {
				approval.WorktreePath = task.WorktreePath
			}
			if task.Workspace != "" {
				approval.Workspace = task.Workspace
			}
			if task.Cost > 0 {
				approval.EstimatedCost = task.Cost
			}
		}
	}

	return approval
}

// ApprovalsResponse wraps approval list with metadata for the unified endpoint
type ApprovalsResponse struct {
	Approvals []UIApproval `json:"approvals"`
	Total     int          `json:"total"`
	Pending   int          `json:"pending_count"`
	Approved  int          `json:"approved_count"`
	Rejected  int          `json:"rejected_count"`
}

// GET /api/approvals?status={status} - Get approvals by status
// GET /api/approvals?status=all - Get all approvals sorted by time (merged history)
// Now uses coordinator store instead of messaging store
func (s *Server) handleApprovals(w http.ResponseWriter, r *http.Request) {
	// POST /api/approvals — secret-approval intake from a CloudSecretApprover
	// (M-SECRET-REMOTE-APPROVAL-WIRING). GET continues to list approvals.
	if r.Method == http.MethodPost {
		s.handleSecretApprovalIntake(w, r)
		return
	}
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	status := r.URL.Query().Get("status")
	if status == "" {
		status = "pending"
	}

	// Check if coordinator approval store is available
	if s.approvalStore == nil {
		// Fall back to messaging store for backwards compatibility
		approvals, err := s.store.GetApprovalsByStatus(status, 50)
		if err != nil {
			http.Error(w, "Failed to get approvals", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(approvals); err != nil {
			log.Printf("Failed to encode approvals response: %v", err)
		}
		return
	}

	ctx := r.Context()
	var result []UIApproval

	if status == "all" {
		// Unified history: get all approvals (pending + resolved), sorted by time
		pending, err := s.approvalStore.ListPendingApprovals(ctx)
		if err != nil {
			log.Printf("Failed to list pending approvals: %v", err)
			http.Error(w, "Failed to get approvals", http.StatusInternalServerError)
			return
		}
		for _, rec := range pending {
			result = append(result, s.mapCoordinatorApprovalToUI(ctx, rec))
		}

		resolved, err := s.approvalStore.ListResolvedApprovals(ctx, 100)
		if err != nil {
			log.Printf("Failed to list resolved approvals: %v", err)
			http.Error(w, "Failed to get approvals", http.StatusInternalServerError)
			return
		}
		for _, rec := range resolved {
			result = append(result, s.mapCoordinatorApprovalToUI(ctx, rec))
		}

		// Sort by time (most recent first)
		sort.Slice(result, func(i, j int) bool {
			// Use ReviewedAt if available (for resolved), otherwise CreatedAt
			iTime := result[i].CreatedAt
			jTime := result[j].CreatedAt
			if result[i].ReviewedAt != nil {
				iTime = *result[i].ReviewedAt
			}
			if result[j].ReviewedAt != nil {
				jTime = *result[j].ReviewedAt
			}
			return iTime > jTime // Most recent first
		})

		// Return wrapped response with counts
		pendingCount := 0
		approvedCount := 0
		rejectedCount := 0
		for _, a := range result {
			switch a.Status {
			case "pending":
				pendingCount++
			case "approved":
				approvedCount++
			case "rejected":
				rejectedCount++
			}
		}

		if result == nil {
			result = []UIApproval{}
		}

		response := ApprovalsResponse{
			Approvals: result,
			Total:     len(result),
			Pending:   pendingCount,
			Approved:  approvedCount,
			Rejected:  rejectedCount,
		}

		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("Failed to encode approvals response: %v", err)
		}
		return
	}

	// Original behavior for specific status
	if status == "pending" {
		pending, err := s.approvalStore.ListPendingApprovals(ctx)
		if err != nil {
			log.Printf("Failed to list pending approvals: %v", err)
			http.Error(w, "Failed to get approvals", http.StatusInternalServerError)
			return
		}
		for _, rec := range pending {
			result = append(result, s.mapCoordinatorApprovalToUI(ctx, rec))
		}
	} else {
		// For approved/rejected, get recent resolved
		resolved, err := s.approvalStore.ListResolvedApprovals(ctx, 50)
		if err != nil {
			log.Printf("Failed to list resolved approvals: %v", err)
			http.Error(w, "Failed to get approvals", http.StatusInternalServerError)
			return
		}
		for _, rec := range resolved {
			if rec.Status == status {
				result = append(result, s.mapCoordinatorApprovalToUI(ctx, rec))
			}
		}
	}

	// Ensure we return an empty array, not null
	if result == nil {
		result = []UIApproval{}
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("Failed to encode approvals response: %v", err)
	}
}

// approvalDecisionBody is the optional JSON body both approve/reject surfaces
// accept. Optional because ntfy action buttons and the documented curl
// examples POST with no body at all.
type approvalDecisionBody struct {
	Notes     string `json:"notes"`
	Permanent bool   `json:"permanent"` // If true, permanent rejection (no retry)
}

// decodeApprovalDecisionBody reads the optional decision body. An empty body
// (io.EOF) is not an error; a malformed one is, and has already been answered
// with 400 when ok is false.
func decodeApprovalDecisionBody(w http.ResponseWriter, r *http.Request) (approvalDecisionBody, bool) {
	var body approvalDecisionBody
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil && err != io.EOF {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return body, false
	}
	return body, true
}

// processTaskApproval is the ONE place the server turns a task-approval
// decision into coordinator.ProcessApprovalRequest — the single path that
// resolves the record, merges the worktree, updates GitHub and fires the
// handoffs that were waiting on the approval.
//
// Both HTTP surfaces call this: /api/approvals/{id}/{action} (the React UI)
// and /api/coordinator/{action}/{id} (the documented curl/script route). The
// second used to resolve the record by hand and return — so an approval on
// that route recorded "approved", dispatched nothing, and left the task
// pending_approval forever, a week after the CLI path had been fixed for the
// same bug (2026-09-07). Two wirings of one decision is how that happens; this
// is the only one now.
func (s *Server) processTaskApproval(ctx context.Context, req *coordinator.ApprovalRequestRecord, action string, body approvalDecisionBody) (*coordinator.ApprovalResult, error) {
	if s.coordStoreRaw == nil {
		return nil, fmt.Errorf("coordinator store not configured")
	}
	if req == nil {
		return nil, fmt.Errorf("approval request is nil")
	}

	// A registry that failed to load is not "no agents": it means every handoff
	// this approval owes will be silently skipped, which is the exact failure
	// this helper exists to end. Say so in the log rather than discarding it.
	agentRegistry, err := coordinator.LoadAgentRegistry()
	if err != nil {
		log.Printf("Approval %s: agent registry unavailable, handoffs cannot fire: %v", req.ID, err)
	}

	// Create GitHub poster for issue updates
	var githubPoster *coordinator.GitHubPoster
	if poster, err := coordinator.NewGitHubPoster(); err == nil {
		githubPoster = poster
	}

	result, err := coordinator.ProcessApprovalRequest(ctx, &coordinator.ApprovalParams{
		TaskID:            req.TaskID,
		Action:            action,
		ApprovedBy:        "dashboard-user",
		Channel:           "dashboard",
		Feedback:          body.Notes,
		SkipMerge:         false,
		KeepWorktree:      false,
		RetriggerOnReject: !body.Permanent, // false = permanent rejection, true = retry with feedback
		Store:             s.coordStoreRaw,
		MsgStore:          s.store, // For feedback messages and handoff delivery
		GitHubPoster:      githubPoster,
		AgentRegistry:     agentRegistry,
		ObsBackend:        s.obsBackend, // M-CHAINS-SIMPLIFY: For chain status updates
	})
	if err != nil {
		log.Printf("Approval processing failed for %s: %v", req.TaskID, err)
		return nil, err
	}

	if !result.Success && len(result.ConflictFiles) > 0 {
		// Report conflicts but don't fail - approval is resolved
		log.Printf("Merge conflicts in task %s: %v", req.TaskID, result.ConflictFiles)
	}

	log.Printf("Dashboard %s: %s", action, result.Message)
	return result, nil
}

// POST /api/approvals/{id}/approve - Approve an approval request
// POST /api/approvals/{id}/reject - Reject an approval request
func (s *Server) handleApproval(w http.ResponseWriter, r *http.Request) {
	// Extract approval ID and action from path
	path := r.URL.Path[len("/api/approvals/"):]

	// GET /api/approvals/{id} — status poll for a single approval
	// (M-SECRET-REMOTE-APPROVAL-WIRING; used by CloudSecretApprover). A bare id
	// with no trailing /action is a status query.
	if r.Method == http.MethodGet {
		if path != "" && !strings.Contains(path, "/") {
			s.handleSecretApprovalStatus(w, r, path)
			return
		}
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var approvalID, action string

	// Parse path: {id}/approve or {id}/reject
	for i, ch := range path {
		if ch == '/' {
			approvalID = path[:i]
			action = path[i+1:]
			break
		}
	}

	if approvalID == "" || (action != "approve" && action != "reject") {
		http.Error(w, "Invalid path", http.StatusBadRequest)
		return
	}

	// Secret-approval token auth (M-SECRET-EFFECT): the iPhone ntfy action
	// buttons authenticate with a signed single-use ?token= rather than IAM.
	// If a token is present (and token auth is enabled) it must be valid for
	// this exact approval+action; a present-but-invalid token is rejected
	// rather than falling through to another auth path.
	if handled, ok := s.checkSecretApprovalToken(r, approvalID, action); handled && !ok {
		http.Error(w, "Invalid or expired approval token", http.StatusUnauthorized)
		return
	} else if !handled {
		// No secret-approval token: this is a dashboard/browser action, so it
		// must meet the same bar as /api/coordinator/approve/ — an
		// authenticated session with the Approver role (#920). Without this
		// gate the endpoint approved as "dashboard-user" for ANY unauthenticated
		// caller. When Firebase auth is not configured the gate passes through,
		// consistent with requireApprover; the single-use ntfy token above
		// remains the only no-session way to resolve an approval.
		if !s.approverSessionAuthorized(w, r) {
			return
		}
	}

	body, ok := decodeApprovalDecisionBody(w, r)
	if !ok {
		return
	}

	ctx := r.Context()

	// Secret approvals resolve directly via the store, WITHOUT the task
	// merge/handoff machinery (ProcessApprovalRequest) — there is no worktree or
	// PR behind a secret request (M-SECRET-REMOTE-APPROVAL-WIRING).
	if s.approvalStore != nil {
		if rec, err := s.approvalStore.GetApprovalRequest(ctx, approvalID); err == nil && rec != nil && rec.Type == "secret" {
			s.resolveSecretApproval(w, r, rec, action, "operator")
			return
		}
	}

	// Use coordinator store with unified processor if available
	if s.approvalStore != nil && s.coordStoreRaw != nil {
		// Get the approval request to find the task ID
		req, getErr := s.approvalStore.GetApprovalRequest(ctx, approvalID)
		if getErr != nil {
			http.Error(w, fmt.Sprintf("Failed to get approval: %v", getErr), http.StatusNotFound)
			return
		}

		if _, err := s.processTaskApproval(ctx, req, action, body); err != nil {
			http.Error(w, fmt.Sprintf("Failed to %s: %v", action, err), http.StatusInternalServerError)
			return
		}

		// Success response
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]string{
			"status":  "success",
			"action":  action,
			"message": fmt.Sprintf("Approval %s successfully", action+"d"),
		}); err != nil {
			log.Printf("Failed to encode approval response: %v", err)
		}
		return
	}

	// Fall back to messaging store for backwards compatibility
	var err error
	if action == "approve" {
		err = s.store.ApproveApproval(approvalID, "user", body.Notes, 24*time.Hour)
	} else {
		err = s.store.RejectApproval(approvalID, "user", body.Notes)
	}

	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to %s approval: %v", action, err), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(map[string]string{
		"status":  "success",
		"action":  action,
		"message": fmt.Sprintf("Approval %s successfully", action+"d"),
	}); err != nil {
		log.Printf("Failed to encode approval response: %v", err)
	}
}

// handleApprovalHistory returns approval history entries
// GET /api/approvals/history?thread_id={id}&limit={n}
func (s *Server) handleApprovalHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	threadID := r.URL.Query().Get("thread_id")
	limitStr := r.URL.Query().Get("limit")
	limit := 100 // Default limit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	entries, err := s.store.GetApprovalHistory(threadID, limit)
	if err != nil {
		log.Printf("Failed to get approval history: %v", err)
		http.Error(w, "Failed to get approval history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(entries); err != nil {
		log.Printf("Failed to encode approval history response: %v", err)
	}
}
