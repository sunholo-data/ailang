package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sort"
	"strconv"
	"time"

	"github.com/sunholo/ailang/internal/coordinator"
	"github.com/sunholo/ailang/internal/display"
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

// POST /api/approvals/{id}/approve - Approve an approval request
// POST /api/approvals/{id}/reject - Reject an approval request
func (s *Server) handleApproval(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Extract approval ID and action from path
	path := r.URL.Path[len("/api/approvals/"):]
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

	// Parse request body for review notes
	var body struct {
		Notes     string `json:"notes"`
		Permanent bool   `json:"permanent"` // If true, permanent rejection (no retry)
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	ctx := r.Context()

	// Use coordinator store with unified processor if available
	if s.approvalStore != nil && s.coordStoreRaw != nil {
		// Get the approval request to find the task ID
		req, getErr := s.approvalStore.GetApprovalRequest(ctx, approvalID)
		if getErr != nil {
			http.Error(w, fmt.Sprintf("Failed to get approval: %v", getErr), http.StatusNotFound)
			return
		}

		// For handoff approvals, use the existing resolve-only path
		if req.Type == "handoff" {
			status := "approved"
			if action == "reject" {
				status = "rejected"
			}
			if err := s.approvalStore.ResolveApprovalRequest(ctx, approvalID, status, "dashboard-user"); err != nil {
				http.Error(w, fmt.Sprintf("Failed to %s handoff: %v", action, err), http.StatusInternalServerError)
				return
			}
		} else {
			// For merge approvals, use the unified ProcessApprovalRequest
			// Load agent registry for per-agent merge branch lookup
			agentRegistry, _ := coordinator.LoadAgentRegistry()

			// Create GitHub poster for issue updates
			var githubPoster *coordinator.GitHubPoster
			if poster, err := coordinator.NewGitHubPoster(); err == nil {
				githubPoster = poster
			}

			// Note: MergeBranch is resolved by processor from AgentRegistry or defaults
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
				MsgStore:          s.store, // For feedback messages
				GitHubPoster:      githubPoster,
				AgentRegistry:     agentRegistry,
			})

			if err != nil {
				log.Printf("Approval processing failed for %s: %v", req.TaskID, err)
				http.Error(w, fmt.Sprintf("Failed to %s: %v", action, err), http.StatusInternalServerError)
				return
			}

			if !result.Success && len(result.ConflictFiles) > 0 {
				// Report conflicts but don't fail - approval is resolved
				log.Printf("Merge conflicts in task %s: %v", req.TaskID, result.ConflictFiles)
			}

			log.Printf("Dashboard %s: %s", action, result.Message)
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
