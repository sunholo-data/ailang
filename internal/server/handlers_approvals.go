package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/sunholo/ailang/internal/coordinator"
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
	Summary      string `json:"summary,omitempty"` // Short summary for display
}

// mapCoordinatorApprovalToUI maps a coordinator ApprovalRequestRecord to the UI format
func (s *Server) mapCoordinatorApprovalToUI(ctx context.Context, rec *coordinator.ApprovalRequestRecord) UIApproval {
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
			if task.Cost > 0 {
				approval.EstimatedCost = task.Cost
			}
		}
	}

	return approval
}

// GET /api/approvals?status={status} - Get approvals by status
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
		Notes string `json:"notes"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Perform action - try coordinator store first, fall back to messaging store
	var err error
	ctx := r.Context()

	if s.approvalStore != nil {
		// Use coordinator store
		status := "approved"
		if action == "reject" {
			status = "rejected"
		}

		// Get the approval request to find the task ID
		req, getErr := s.approvalStore.GetApprovalRequest(ctx, approvalID)
		if getErr != nil {
			http.Error(w, fmt.Sprintf("Failed to get approval: %v", getErr), http.StatusNotFound)
			return
		}

		// Store feedback as a task event if provided (matching CLI pattern)
		if body.Notes != "" && s.coordStoreRaw != nil && req.TaskID != "" {
			// Get task to retrieve iteration number
			iteration := 1
			if task, getTaskErr := s.taskEventStore.GetTask(ctx, req.TaskID); getTaskErr == nil && task != nil {
				iteration = task.Iteration
				if iteration == 0 {
					iteration = 1
				}
			}

			// Use StoreFeedbackEvent with HumanFeedback struct (same as CLI)
			feedback := &coordinator.HumanFeedback{
				TaskID:    req.TaskID,
				Iteration: iteration,
				Feedback:  body.Notes,
				Action:    action,
				Timestamp: time.Now(),
				UserID:    "dashboard-user",
			}
			if storeErr := coordinator.StoreFeedbackEvent(ctx, s.coordStoreRaw, feedback); storeErr != nil {
				log.Printf("Warning: Failed to store feedback event: %v", storeErr)
			} else {
				log.Printf("Stored %s feedback for task %s (iteration %d)", action, req.TaskID, iteration)
			}
		}

		err = s.approvalStore.ResolveApprovalRequest(ctx, approvalID, status, "dashboard-user")

		// For merge approvals, also perform the merge (matching CLI behavior)
		if err == nil && req.Type == "merge" && action == "approve" && s.coordStoreRaw != nil {
			task, getTaskErr := s.taskEventStore.GetTask(ctx, req.TaskID)
			if getTaskErr == nil && task != nil && task.WorktreePath != "" {
				log.Printf("Dashboard approval: merging worktree for task %s", req.TaskID)

				// Store approval event for audit trail
				if auditErr := coordinator.StoreApprovalEvent(ctx, s.coordStoreRaw, req.TaskID, "dashboard-user"); auditErr != nil {
					log.Printf("Warning: Failed to store approval event: %v", auditErr)
				}

				// Perform the merge
				mergeResult, mergeErr := coordinator.MergeWorktree(ctx, task.WorktreePath, "dev")
				if mergeErr != nil {
					log.Printf("Warning: Merge failed for task %s: %v", req.TaskID, mergeErr)
					// Don't fail the approval - the approval is resolved, merge can be done manually
				} else if mergeResult != nil && mergeResult.Success {
					log.Printf("Successfully merged worktree for task %s", req.TaskID)
					// Mark task as completed
					if markErr := s.coordStoreRaw.MarkTaskCompleted(ctx, req.TaskID, &coordinator.ExecuteResult{
						Success: true,
						Output:  "Approved and merged via dashboard",
					}); markErr != nil {
						log.Printf("Warning: Failed to mark task completed: %v", markErr)
					}
				} else if mergeResult != nil && len(mergeResult.ConflictFiles) > 0 {
					log.Printf("Merge conflicts in task %s: %v", req.TaskID, mergeResult.ConflictFiles)
				}
			}
		}
	} else {
		// Fall back to messaging store for backwards compatibility
		if action == "approve" {
			err = s.store.ApproveApproval(approvalID, "user", body.Notes, 24*time.Hour)
		} else {
			err = s.store.RejectApproval(approvalID, "user", body.Notes)
		}
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
