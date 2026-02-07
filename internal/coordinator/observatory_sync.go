// Package coordinator provides task coordination and execution.
package coordinator

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/sunholo/ailang/internal/observatory"
)

// ObservatorySync handles syncing coordinator entities to Observatory.
// This enables the full entity hierarchy: WORKSPACE → TASK → AGENT → SPANS
type ObservatorySync struct {
	backend observatory.Backend
	logger  *log.Logger

	// Cache workspace IDs to avoid repeated lookups
	workspaceCache map[string]string // path -> workspace ID
	cacheMu        sync.RWMutex
}

// NewObservatorySync creates a new ObservatorySync instance.
func NewObservatorySync(backend observatory.Backend, logger *log.Logger) *ObservatorySync {
	if logger == nil {
		logger = log.Default()
	}
	return &ObservatorySync{
		backend:        backend,
		logger:         logger,
		workspaceCache: make(map[string]string),
	}
}

// SyncTask syncs a coordinator task to Observatory.
// This should be called when a task is created or updated.
func (s *ObservatorySync) SyncTask(ctx context.Context, task *TaskRecord) error {
	if s.backend == nil {
		return nil // Observatory not configured
	}

	// Get or create workspace for this task
	workspaceID, err := s.getOrCreateWorkspace(ctx, task.Workspace)
	if err != nil {
		s.logger.Printf("Observatory sync: failed to get/create workspace for %s: %v", task.Workspace, err)
		// Continue with empty workspace ID - don't block task execution
		workspaceID = ""
	}

	// Convert coordinator task to observatory task
	obsTask := &observatory.Task{
		ID:           task.ID,
		WorkspaceID:  workspaceID,
		ParentTaskID: task.ParentTaskID, // Links to parent task for handoff chains
		Title:        task.Title,
		Description:  task.Content,
		SourceType:   s.convertSourceType(task),
		SourceRef:    s.getSourceRef(task),
		Status:       s.convertTaskStatus(task.Status),
		Priority:     s.convertPriority(task.Priority),
		CreatedAt:    task.CreatedAt,
		StartedAt:    task.StartedAt,
		CompletedAt:  task.CompletedAt,
	}

	s.logger.Printf("Observatory sync DEBUG: task.ID=%q, workspaceID=%q, title=%q", task.ID, workspaceID, task.Title)

	// Try to create first
	if err := s.backend.CreateTask(ctx, obsTask); err != nil {
		s.logger.Printf("Observatory sync DEBUG: CreateTask error: %v", err)
		// If exists, try to update
		if existingTask, getErr := s.backend.GetTask(ctx, task.ID); getErr == nil && existingTask != nil {
			return s.backend.UpdateTask(ctx, obsTask)
		}
		return fmt.Errorf("failed to sync task %s: %w", task.ID, err)
	}

	s.logger.Printf("Observatory sync: task %s synced to workspace %s", task.ID, workspaceID)
	return nil
}

// SyncAgentAssignment syncs an agent assignment to Observatory.
// This creates the link between a task and the agent executing it.
func (s *ObservatorySync) SyncAgentAssignment(ctx context.Context, taskID, agentID, provider string) (string, error) {
	if s.backend == nil {
		return "", nil // Observatory not configured
	}

	// Generate a deterministic assignment ID
	assignmentID := s.generateAssignmentID(taskID, agentID)

	assignment := &observatory.AgentAssignment{
		ID:         assignmentID,
		TaskID:     taskID,
		AgentID:    agentID,
		Provider:   s.convertProvider(provider),
		Status:     observatory.AgentStatusRunning,
		AssignedAt: time.Now(),
	}
	now := time.Now()
	assignment.StartedAt = &now

	if err := s.backend.CreateAgentAssignment(ctx, assignment); err != nil {
		s.logger.Printf("Observatory sync: failed to create agent assignment: %v", err)
		return "", err
	}

	s.logger.Printf("Observatory sync: agent assignment %s created (task=%s, agent=%s)", assignmentID, taskID, agentID)
	return assignmentID, nil
}

// CompleteAgentAssignment marks an agent assignment as completed.
func (s *ObservatorySync) CompleteAgentAssignment(ctx context.Context, assignmentID string, success bool) error {
	if s.backend == nil || assignmentID == "" {
		return nil
	}

	assignment, err := s.backend.GetAgentAssignment(ctx, assignmentID)
	if err != nil {
		return fmt.Errorf("failed to get assignment %s: %w", assignmentID, err)
	}
	if assignment == nil {
		return nil // Assignment not found, skip
	}

	now := time.Now()
	assignment.CompletedAt = &now
	if success {
		assignment.Status = observatory.AgentStatusCompleted
	} else {
		assignment.Status = observatory.AgentStatusFailed
	}

	if assignment.StartedAt != nil {
		assignment.DurationMs = now.Sub(*assignment.StartedAt).Milliseconds()
	}

	return s.backend.UpdateAgentAssignment(ctx, assignment)
}

// getOrCreateWorkspace gets an existing workspace or creates a new one.
func (s *ObservatorySync) getOrCreateWorkspace(ctx context.Context, workspacePath string) (string, error) {
	if workspacePath == "" {
		return "", nil
	}

	// Normalize path
	absPath, err := filepath.Abs(workspacePath)
	if err != nil {
		absPath = workspacePath
	}

	// Check cache first
	s.cacheMu.RLock()
	if id, ok := s.workspaceCache[absPath]; ok {
		s.cacheMu.RUnlock()
		return id, nil
	}
	s.cacheMu.RUnlock()

	// Generate deterministic workspace ID from path
	workspaceID := s.generateWorkspaceID(absPath)

	// Try to get existing workspace
	existing, err := s.backend.GetWorkspace(ctx, workspaceID)
	if err == nil && existing != nil {
		s.cacheMu.Lock()
		s.workspaceCache[absPath] = workspaceID
		s.cacheMu.Unlock()
		return workspaceID, nil
	}

	// Create new workspace
	workspace := &observatory.Workspace{
		ID:        workspaceID,
		Name:      filepath.Base(absPath),
		Path:      absPath,
		GitRemote: s.getGitRemote(absPath),
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := s.backend.CreateWorkspace(ctx, workspace); err != nil {
		return "", fmt.Errorf("failed to create workspace: %w", err)
	}

	s.cacheMu.Lock()
	s.workspaceCache[absPath] = workspaceID
	s.cacheMu.Unlock()

	s.logger.Printf("Observatory sync: workspace %s created for %s", workspaceID, absPath)
	return workspaceID, nil
}

// generateWorkspaceID creates a deterministic ID from the workspace path.
func (s *ObservatorySync) generateWorkspaceID(path string) string {
	hash := sha256.Sum256([]byte(path))
	return "ws_" + hex.EncodeToString(hash[:8])
}

// generateAssignmentID creates a deterministic assignment ID.
func (s *ObservatorySync) generateAssignmentID(taskID, agentID string) string {
	data := fmt.Sprintf("%s:%s:%d", taskID, agentID, time.Now().UnixNano())
	hash := sha256.Sum256([]byte(data))
	return "aa_" + hex.EncodeToString(hash[:8])
}

// getGitRemote returns the git remote URL for a path, or empty string if not a git repo.
func (s *ObservatorySync) getGitRemote(path string) string {
	cmd := exec.Command("git", "-C", path, "remote", "get-url", "origin")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// convertSourceType converts task metadata to observatory source type.
func (s *ObservatorySync) convertSourceType(task *TaskRecord) observatory.TaskSourceType {
	if task.GithubIssue > 0 {
		return observatory.TaskSourceGitHub
	}
	if task.MessageID != "" {
		return observatory.TaskSourceMessage
	}
	return observatory.TaskSourceManual
}

// getSourceRef returns the source reference (GitHub issue, message ID, etc.)
func (s *ObservatorySync) getSourceRef(task *TaskRecord) string {
	if task.GithubIssue > 0 {
		return fmt.Sprintf("#%d", task.GithubIssue)
	}
	return task.MessageID
}

// convertTaskStatus converts coordinator status to observatory status.
func (s *ObservatorySync) convertTaskStatus(status TaskStatus) observatory.TaskStatus {
	switch status {
	case TaskStatusPending, TaskStatusQueued:
		return observatory.TaskStatusPending
	case TaskStatusRunning:
		return observatory.TaskStatusRunning
	case TaskStatusCompleted, TaskStatusPendingApproval:
		return observatory.TaskStatusCompleted
	case TaskStatusFailed, TaskStatusRejected, TaskStatusCancelled:
		return observatory.TaskStatusFailed
	default:
		return observatory.TaskStatusPending
	}
}

// convertPriority converts numeric priority to string.
func (s *ObservatorySync) convertPriority(priority int) string {
	switch {
	case priority >= 80:
		return "critical"
	case priority >= 60:
		return "high"
	case priority >= 40:
		return "medium"
	default:
		return "low"
	}
}

// GetWorkspaceID returns the cached workspace ID for a given path.
// Returns empty string if not cached (workspace not yet synced).
func (s *ObservatorySync) GetWorkspaceID(path string) string {
	if s == nil || path == "" {
		return ""
	}

	absPath, err := filepath.Abs(path)
	if err != nil {
		absPath = path
	}

	s.cacheMu.RLock()
	defer s.cacheMu.RUnlock()
	return s.workspaceCache[absPath]
}

// convertProvider converts provider string to observatory Provider type.
// observatory.Provider is just `type Provider string`, so this is a direct cast.
// Named constants (ProviderClaude, ProviderGemini) still exist for typed comparisons
// but this function works for any executor name without needing updates.
func (s *ObservatorySync) convertProvider(provider string) observatory.Provider {
	return observatory.Provider(strings.ToLower(provider))
}
