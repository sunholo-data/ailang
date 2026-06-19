// Package coordinator provides approval checkpoints for task execution.
package coordinator

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ApprovalType defines the kind of approval being requested
type ApprovalType string

const (
	ApprovalTypeMerge   ApprovalType = "merge"   // Request to merge worktree changes
	ApprovalTypeDestroy ApprovalType = "destroy" // Request to destroy worktree with changes
	ApprovalTypeExecute ApprovalType = "execute" // Request to execute a destructive operation
	ApprovalTypeCost    ApprovalType = "cost"    // Cost threshold exceeded
	ApprovalTypeHandoff ApprovalType = "handoff" // Request to hand off work to another agent
	ApprovalTypeSecret  ApprovalType = "secret"  // Request to resolve a secret reference (M-SECRET-EFFECT)
)

// ApprovalStatus represents the state of an approval request
type ApprovalStatus string

const (
	ApprovalStatusPending  ApprovalStatus = "pending"
	ApprovalStatusApproved ApprovalStatus = "approved"
	ApprovalStatusRejected ApprovalStatus = "rejected"
	ApprovalStatusTimeout  ApprovalStatus = "timeout"
)

// ApprovalRequest represents a pending approval request
type ApprovalRequest struct {
	ID           string         `json:"id"`
	TaskID       string         `json:"task_id"`
	ThreadID     string         `json:"thread_id,omitempty"`
	Type         ApprovalType   `json:"type"`
	Status       ApprovalStatus `json:"status"`
	Title        string         `json:"title"`
	Description  string         `json:"description"`
	DiffSummary  string         `json:"diff_summary,omitempty"`
	FilesChanged []string       `json:"files_changed,omitempty"`
	CreatedAt    time.Time      `json:"created_at"`
	ResolvedAt   *time.Time     `json:"resolved_at,omitempty"`
	ResolvedBy   string         `json:"resolved_by,omitempty"`
	Timeout      time.Duration  `json:"timeout"`
	AutoReject   bool           `json:"auto_reject"` // Reject on timeout instead of approve

	// Handoff-specific fields (used when Type == ApprovalTypeHandoff)
	SourceAgentID string `json:"source_agent_id,omitempty"` // Agent that completed the task
	TargetAgentID string `json:"target_agent_id,omitempty"` // Agent to hand off to
	SessionID     string `json:"session_id,omitempty"`      // Claude Code/Gemini CLI session for continuity
	HandoffData   string `json:"handoff_data,omitempty"`    // Additional context for handoff

	// Secret-specific fields (used when Type == ApprovalTypeSecret, M-SECRET-EFFECT).
	// SecretRef is the op:// reference (safe to store/display); the resolved
	// VALUE is never part of an approval request.
	SecretRef     string `json:"secret_ref,omitempty"`     // op:// reference being requested
	SecretPurpose string `json:"secret_purpose,omitempty"` // human-readable intent
	AgentID       string `json:"agent_id,omitempty"`       // agent requesting the secret
}

// ApprovalCallback is called when an approval is resolved
type ApprovalCallback func(request *ApprovalRequest)

// ApprovalCheckpoint manages approval requests for a coordinator.
// It provides blocking wait for human decisions and timeout handling.
type ApprovalCheckpoint struct {
	mu       sync.RWMutex
	requests map[string]*ApprovalRequest
	waiters  map[string]chan ApprovalStatus
	callback ApprovalCallback

	// Default timeout for approval requests
	defaultTimeout time.Duration
}

// NewApprovalCheckpoint creates a new approval checkpoint manager
func NewApprovalCheckpoint(defaultTimeout time.Duration) *ApprovalCheckpoint {
	if defaultTimeout == 0 {
		defaultTimeout = 1 * time.Hour // Default to 1 hour
	}
	return &ApprovalCheckpoint{
		requests:       make(map[string]*ApprovalRequest),
		waiters:        make(map[string]chan ApprovalStatus),
		defaultTimeout: defaultTimeout,
	}
}

// SetCallback sets the callback for approval resolution events
func (ac *ApprovalCheckpoint) SetCallback(callback ApprovalCallback) {
	ac.mu.Lock()
	ac.callback = callback
	ac.mu.Unlock()
}

// RequestApproval creates a new approval request and blocks until resolved.
// Returns the approval status (approved, rejected, or timeout).
func (ac *ApprovalCheckpoint) RequestApproval(ctx context.Context, request *ApprovalRequest) (ApprovalStatus, error) {
	if request.ID == "" {
		request.ID = fmt.Sprintf("approval-%d", time.Now().UnixNano())
	}
	if request.CreatedAt.IsZero() {
		request.CreatedAt = time.Now()
	}
	if request.Timeout == 0 {
		request.Timeout = ac.defaultTimeout
	}
	request.Status = ApprovalStatusPending

	// Create wait channel
	waitCh := make(chan ApprovalStatus, 1)

	ac.mu.Lock()
	ac.requests[request.ID] = request
	ac.waiters[request.ID] = waitCh
	ac.mu.Unlock()

	// Start timeout goroutine
	timeoutCtx, cancel := context.WithTimeout(ctx, request.Timeout)
	defer cancel()

	go func() {
		<-timeoutCtx.Done()
		if timeoutCtx.Err() == context.DeadlineExceeded {
			ac.handleTimeout(request.ID)
		}
	}()

	// Wait for resolution
	select {
	case status := <-waitCh:
		return status, nil
	case <-ctx.Done():
		// Context cancelled externally
		ac.cleanup(request.ID)
		return ApprovalStatusRejected, ctx.Err()
	}
}

// Approve approves an approval request
func (ac *ApprovalCheckpoint) Approve(requestID string, resolvedBy string) error {
	return ac.resolve(requestID, ApprovalStatusApproved, resolvedBy)
}

// Reject rejects an approval request
func (ac *ApprovalCheckpoint) Reject(requestID string, resolvedBy string) error {
	return ac.resolve(requestID, ApprovalStatusRejected, resolvedBy)
}

// resolve resolves an approval request with the given status
func (ac *ApprovalCheckpoint) resolve(requestID string, status ApprovalStatus, resolvedBy string) error {
	ac.mu.Lock()
	request, ok := ac.requests[requestID]
	if !ok {
		ac.mu.Unlock()
		return fmt.Errorf("approval request not found: %s", requestID)
	}

	now := time.Now()
	request.Status = status
	request.ResolvedAt = &now
	request.ResolvedBy = resolvedBy

	waitCh := ac.waiters[requestID]
	callback := ac.callback
	ac.mu.Unlock()

	// Notify waiter
	if waitCh != nil {
		select {
		case waitCh <- status:
		default:
		}
	}

	// Call callback
	if callback != nil {
		callback(request)
	}

	return nil
}

// handleTimeout handles approval request timeout
func (ac *ApprovalCheckpoint) handleTimeout(requestID string) {
	ac.mu.Lock()
	request, ok := ac.requests[requestID]
	if !ok || request.Status != ApprovalStatusPending {
		ac.mu.Unlock()
		return
	}

	status := ApprovalStatusTimeout
	if request.AutoReject {
		status = ApprovalStatusRejected
	}

	now := time.Now()
	request.Status = status
	request.ResolvedAt = &now
	request.ResolvedBy = "system:timeout"

	waitCh := ac.waiters[requestID]
	callback := ac.callback
	ac.mu.Unlock()

	// Notify waiter
	if waitCh != nil {
		select {
		case waitCh <- status:
		default:
		}
	}

	// Call callback
	if callback != nil {
		callback(request)
	}
}

// cleanup removes an approval request from tracking
func (ac *ApprovalCheckpoint) cleanup(requestID string) {
	ac.mu.Lock()
	delete(ac.requests, requestID)
	delete(ac.waiters, requestID)
	ac.mu.Unlock()
}

// GetRequest returns an approval request by ID
func (ac *ApprovalCheckpoint) GetRequest(requestID string) *ApprovalRequest {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ac.requests[requestID]
}

// GetPendingRequests returns all pending approval requests
func (ac *ApprovalCheckpoint) GetPendingRequests() []*ApprovalRequest {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	var pending []*ApprovalRequest
	for _, req := range ac.requests {
		if req.Status == ApprovalStatusPending {
			pending = append(pending, req)
		}
	}
	return pending
}

// GetRequestByTask returns the approval request for a task
func (ac *ApprovalCheckpoint) GetRequestByTask(taskID string) *ApprovalRequest {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	for _, req := range ac.requests {
		if req.TaskID == taskID && req.Status == ApprovalStatusPending {
			return req
		}
	}
	return nil
}

// HasPendingApproval checks if a task has a pending approval request
func (ac *ApprovalCheckpoint) HasPendingApproval(taskID string) bool {
	return ac.GetRequestByTask(taskID) != nil
}

// ApproveByTask approves the pending request for a task
func (ac *ApprovalCheckpoint) ApproveByTask(taskID string, resolvedBy string) error {
	req := ac.GetRequestByTask(taskID)
	if req == nil {
		return fmt.Errorf("no pending approval for task: %s", taskID)
	}
	return ac.Approve(req.ID, resolvedBy)
}

// RejectByTask rejects the pending request for a task
func (ac *ApprovalCheckpoint) RejectByTask(taskID string, resolvedBy string) error {
	req := ac.GetRequestByTask(taskID)
	if req == nil {
		return fmt.Errorf("no pending approval for task: %s", taskID)
	}
	return ac.Reject(req.ID, resolvedBy)
}

// Count returns the number of pending approval requests
func (ac *ApprovalCheckpoint) Count() int {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	count := 0
	for _, req := range ac.requests {
		if req.Status == ApprovalStatusPending {
			count++
		}
	}
	return count
}

// Clear removes all requests (for testing/cleanup)
func (ac *ApprovalCheckpoint) Clear() {
	ac.mu.Lock()
	defer ac.mu.Unlock()

	// Close all wait channels
	for _, ch := range ac.waiters {
		close(ch)
	}

	ac.requests = make(map[string]*ApprovalRequest)
	ac.waiters = make(map[string]chan ApprovalStatus)
}

// ApprovalStore interface for persistence
type ApprovalStore interface {
	CreateApprovalRequest(ctx context.Context, req *ApprovalRequestRecord) error
	GetApprovalRequest(ctx context.Context, id string) (*ApprovalRequestRecord, error)
	GetApprovalRequestByTask(ctx context.Context, taskID string) (*ApprovalRequestRecord, error)
	ListPendingApprovals(ctx context.Context) ([]*ApprovalRequestRecord, error)
	ResolveApprovalRequest(ctx context.Context, id string, status string, resolvedBy string) error
}

// StoreBackedApprovalCheckpoint wraps ApprovalCheckpoint with SQLite persistence.
// This allows CLI commands to approve/reject requests that the daemon is waiting on.
type StoreBackedApprovalCheckpoint struct {
	*ApprovalCheckpoint
	store        ApprovalStore
	pollInterval time.Duration
}

// NewStoreBackedApprovalCheckpoint creates a store-backed approval checkpoint
func NewStoreBackedApprovalCheckpoint(store ApprovalStore, defaultTimeout time.Duration) *StoreBackedApprovalCheckpoint {
	return &StoreBackedApprovalCheckpoint{
		ApprovalCheckpoint: NewApprovalCheckpoint(defaultTimeout),
		store:              store,
		pollInterval:       2 * time.Second,
	}
}

// RequestApproval creates an approval request and waits for resolution.
// It persists the request to the store and polls for status changes.
func (sac *StoreBackedApprovalCheckpoint) RequestApproval(ctx context.Context, request *ApprovalRequest) (ApprovalStatus, error) {
	if request.ID == "" {
		request.ID = fmt.Sprintf("approval-%d", time.Now().UnixNano())
	}
	if request.CreatedAt.IsZero() {
		request.CreatedAt = time.Now()
	}
	if request.Timeout == 0 {
		request.Timeout = sac.defaultTimeout
	}
	request.Status = ApprovalStatusPending

	// Persist to store
	timeoutAt := request.CreatedAt.Add(request.Timeout)
	record := &ApprovalRequestRecord{
		ID:          request.ID,
		TaskID:      request.TaskID,
		Type:        string(request.Type),
		Description: request.Description,
		Status:      string(ApprovalStatusPending),
		CreatedAt:   request.CreatedAt,
		TimeoutAt:   &timeoutAt,
		AutoReject:  request.AutoReject,
	}

	if sac.store != nil {
		if err := sac.store.CreateApprovalRequest(ctx, record); err != nil {
			return ApprovalStatusRejected, fmt.Errorf("failed to persist approval request: %w", err)
		}
	}

	// Also store in memory for in-process resolution
	waitCh := make(chan ApprovalStatus, 1)
	sac.mu.Lock()
	sac.requests[request.ID] = request
	sac.waiters[request.ID] = waitCh
	sac.mu.Unlock()

	// Start timeout goroutine
	timeoutCtx, cancel := context.WithTimeout(ctx, request.Timeout)
	defer cancel()

	// Start polling for store changes
	pollTicker := time.NewTicker(sac.pollInterval)
	defer pollTicker.Stop()

	for {
		select {
		case status := <-waitCh:
			// Resolved in-process
			return status, nil

		case <-pollTicker.C:
			// Poll store for status changes
			if sac.store != nil {
				record, err := sac.store.GetApprovalRequest(ctx, request.ID)
				if err == nil && record != nil {
					if record.Status == "approved" {
						sac.cleanup(request.ID)
						return ApprovalStatusApproved, nil
					} else if record.Status == "rejected" {
						sac.cleanup(request.ID)
						return ApprovalStatusRejected, nil
					}
				}
			}

		case <-timeoutCtx.Done():
			if timeoutCtx.Err() == context.DeadlineExceeded {
				sac.handleTimeout(request.ID)
				status := ApprovalStatusTimeout
				if request.AutoReject {
					status = ApprovalStatusRejected
				}
				// Update store
				if sac.store != nil {
					_ = sac.store.ResolveApprovalRequest(ctx, request.ID, string(status), "system:timeout")
				}
				return status, nil
			}
			// Context cancelled externally
			sac.cleanup(request.ID)
			return ApprovalStatusRejected, ctx.Err()
		}
	}
}
