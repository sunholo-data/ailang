package coordinator

import (
	"context"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"sync"
	"time"
)

// debugApprovalWatcher controls verbose logging (set via DEBUG_APPROVAL_WATCHER=1)
var debugApprovalWatcher = os.Getenv("DEBUG_APPROVAL_WATCHER") == "1"

// ApprovalLabel constants for GitHub label-based approval workflow
const (
	// Request labels - added by coordinator when work is complete
	LabelNeedsDesignApproval = "needs-design-approval"
	LabelNeedsSprintApproval = "needs-sprint-approval"
	LabelNeedsMergeApproval  = "needs-merge-approval"
	LabelNeedsRevision       = "needs-revision"

	// Approval labels - added by human to approve
	LabelDesignApproved = "design-approved"
	LabelSprintApproved = "sprint-approved"
	LabelMergeApproved  = "merge-approved"
)

// ApprovalEvent represents a detected approval or revision request
type ApprovalEvent struct {
	TaskID      string
	IssueNumber int
	Label       string
	EventType   ApprovalEventType
}

// ApprovalEventType categorizes the type of approval event
type ApprovalEventType string

const (
	ApprovalEventDesign   ApprovalEventType = "design-approved"
	ApprovalEventSprint   ApprovalEventType = "sprint-approved"
	ApprovalEventMerge    ApprovalEventType = "merge-approved"
	ApprovalEventRevision ApprovalEventType = "needs-revision"
)

// ApprovalHandler is called when an approval is detected
type ApprovalHandler func(ctx context.Context, event *ApprovalEvent) error

// ApprovalWatcher polls GitHub for approval labels and triggers pipeline stages.
// It watches issues linked to tasks and detects when humans add approval labels.
type ApprovalWatcher struct {
	mu            sync.Mutex
	poster        *GitHubPoster
	store         Store
	pollInterval  time.Duration
	handlers      map[ApprovalEventType]ApprovalHandler
	watchedIssues map[int]string // issue number -> task ID
	stopCh        chan struct{}
	running       bool
	lastPoll      time.Time // track last successful poll for status reporting
}

// NewApprovalWatcher creates a new approval watcher.
func NewApprovalWatcher(poster *GitHubPoster, store Store, pollInterval time.Duration) *ApprovalWatcher {
	if pollInterval == 0 {
		pollInterval = 60 * time.Second
	}
	return &ApprovalWatcher{
		poster:        poster,
		store:         store,
		pollInterval:  pollInterval,
		handlers:      make(map[ApprovalEventType]ApprovalHandler),
		watchedIssues: make(map[int]string),
		stopCh:        make(chan struct{}),
	}
}

// RegisterHandler registers a handler for a specific approval event type.
func (w *ApprovalWatcher) RegisterHandler(eventType ApprovalEventType, handler ApprovalHandler) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.handlers[eventType] = handler
}

// WatchIssue starts watching a GitHub issue for approval labels.
func (w *ApprovalWatcher) WatchIssue(issueNumber int, taskID string) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.watchedIssues[issueNumber] = taskID
	log.Printf("[ApprovalWatcher] Now watching issue #%d for task %s", issueNumber, taskID)
}

// UnwatchIssue stops watching a GitHub issue.
func (w *ApprovalWatcher) UnwatchIssue(issueNumber int) {
	w.mu.Lock()
	defer w.mu.Unlock()
	delete(w.watchedIssues, issueNumber)
	log.Printf("[ApprovalWatcher] Stopped watching issue #%d", issueNumber)
}

// Start begins the polling loop.
func (w *ApprovalWatcher) Start(ctx context.Context) error {
	w.mu.Lock()
	if w.running {
		w.mu.Unlock()
		return fmt.Errorf("approval watcher already running")
	}
	w.running = true
	w.stopCh = make(chan struct{})
	w.mu.Unlock()

	log.Printf("[ApprovalWatcher] Starting with poll interval %s", w.pollInterval)
	go w.pollLoop(ctx)
	return nil
}

// Stop halts the polling loop.
func (w *ApprovalWatcher) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.running {
		close(w.stopCh)
		w.running = false
	}
}

// IsRunning returns whether the watcher is currently running.
func (w *ApprovalWatcher) IsRunning() bool {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.running
}

// pollLoop continuously polls GitHub for approval labels.
func (w *ApprovalWatcher) pollLoop(ctx context.Context) {
	ticker := time.NewTicker(w.pollInterval)
	defer ticker.Stop()

	// Do an initial poll immediately
	w.pollOnce(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-w.stopCh:
			return
		case <-ticker.C:
			w.pollOnce(ctx)
		}
	}
}

// pollOnce checks all watched issues for approval labels.
func (w *ApprovalWatcher) pollOnce(ctx context.Context) {
	w.mu.Lock()
	issues := make(map[int]string, len(w.watchedIssues))
	for k, v := range w.watchedIssues {
		issues[k] = v
	}
	w.lastPoll = time.Now()
	w.mu.Unlock()

	if debugApprovalWatcher {
		log.Printf("[ApprovalWatcher] Poll cycle started (watching %d issues)", len(issues))
	}

	eventsFound := 0
	for issueNum, taskID := range issues {
		select {
		case <-ctx.Done():
			return
		default:
		}

		event := w.checkIssueLabels(ctx, issueNum, taskID)
		if event != nil {
			eventsFound++
			w.handleEvent(ctx, event)
		}
	}

	if debugApprovalWatcher {
		log.Printf("[ApprovalWatcher] Poll cycle complete (%d issues, %d events)", len(issues), eventsFound)
	}
}

// checkIssueLabels checks if an issue has any approval labels.
func (w *ApprovalWatcher) checkIssueLabels(ctx context.Context, issueNum int, taskID string) *ApprovalEvent {
	labels, err := w.poster.GetLabels(issueNum)
	if err != nil {
		log.Printf("[ApprovalWatcher] Error getting labels for issue #%d: %v", issueNum, err)
		return nil
	}

	if debugApprovalWatcher {
		log.Printf("[ApprovalWatcher] Issue #%d labels: %v", issueNum, labels)
	}

	// Check for approval labels in priority order
	for _, label := range labels {
		switch label {
		case LabelNeedsRevision:
			return &ApprovalEvent{
				TaskID:      taskID,
				IssueNumber: issueNum,
				Label:       label,
				EventType:   ApprovalEventRevision,
			}
		case LabelDesignApproved:
			return &ApprovalEvent{
				TaskID:      taskID,
				IssueNumber: issueNum,
				Label:       label,
				EventType:   ApprovalEventDesign,
			}
		case LabelSprintApproved:
			return &ApprovalEvent{
				TaskID:      taskID,
				IssueNumber: issueNum,
				Label:       label,
				EventType:   ApprovalEventSprint,
			}
		case LabelMergeApproved:
			return &ApprovalEvent{
				TaskID:      taskID,
				IssueNumber: issueNum,
				Label:       label,
				EventType:   ApprovalEventMerge,
			}
		}
	}

	return nil
}

// handleEvent processes an approval event by calling the registered handler.
// CRITICAL: This function includes panic recovery to prevent handler panics from
// killing the poll goroutine. If a handler panics, we log the error and stack
// trace but continue polling. This prevents silent failures where the watcher
// stops working with no indication of why.
func (w *ApprovalWatcher) handleEvent(ctx context.Context, event *ApprovalEvent) {
	// Panic recovery - prevents handler panics from killing the poll goroutine
	defer func() {
		if r := recover(); r != nil {
			log.Printf("[ApprovalWatcher] PANIC in handler for %s (task %s, issue #%d): %v",
				event.EventType, event.TaskID, event.IssueNumber, r)
			log.Printf("[ApprovalWatcher] Stack trace:\n%s", debug.Stack())
		}
	}()

	w.mu.Lock()
	handler, ok := w.handlers[event.EventType]
	w.mu.Unlock()

	if !ok {
		log.Printf("[ApprovalWatcher] No handler for event type: %s", event.EventType)
		return
	}

	log.Printf("[ApprovalWatcher] Processing %s for task %s (issue #%d)",
		event.EventType, event.TaskID, event.IssueNumber)

	if err := handler(ctx, event); err != nil {
		log.Printf("[ApprovalWatcher] Handler error for %s: %v", event.EventType, err)
		return
	}

	// Remove the approval label after processing (to prevent re-triggering)
	if err := w.poster.RemoveLabel(event.IssueNumber, event.Label); err != nil {
		log.Printf("[ApprovalWatcher] Failed to remove label %s from issue #%d: %v",
			event.Label, event.IssueNumber, err)
	}

	// Also remove the corresponding "needs-*" label if present
	var needsLabel string
	switch event.EventType {
	case ApprovalEventDesign:
		needsLabel = LabelNeedsDesignApproval
	case ApprovalEventSprint:
		needsLabel = LabelNeedsSprintApproval
	case ApprovalEventMerge:
		needsLabel = LabelNeedsMergeApproval
	}
	if needsLabel != "" {
		_ = w.poster.RemoveLabel(event.IssueNumber, needsLabel)
	}
}

// LoadWatchedIssuesFromStore loads all tasks with GitHub issues that are in active stages.
func (w *ApprovalWatcher) LoadWatchedIssuesFromStore(ctx context.Context) error {
	// Load tasks in design, sprint, or implementation stages
	stages := []TaskStage{TaskStageDesign, TaskStageSprint, TaskStageImplementation, TaskStageMerge}

	for _, stage := range stages {
		tasks, err := w.store.GetTasksByStage(ctx, stage)
		if err != nil {
			return fmt.Errorf("failed to load tasks for stage %s: %w", stage, err)
		}

		for _, task := range tasks {
			if task.GithubIssue > 0 {
				w.WatchIssue(task.GithubIssue, task.ID)
			}
		}
	}

	return nil
}

// WatchedIssueCount returns the number of issues being watched.
func (w *ApprovalWatcher) WatchedIssueCount() int {
	w.mu.Lock()
	defer w.mu.Unlock()
	return len(w.watchedIssues)
}

// WatcherStatus represents the current state of the ApprovalWatcher.
type WatcherStatus struct {
	Running       bool              `json:"running"`
	LastPoll      time.Time         `json:"last_poll"`
	PollInterval  time.Duration     `json:"poll_interval"`
	WatchedIssues map[int]string    `json:"watched_issues"` // issue number -> task ID
}

// GetStatus returns the current status of the watcher.
func (w *ApprovalWatcher) GetStatus() WatcherStatus {
	w.mu.Lock()
	defer w.mu.Unlock()

	// Copy watched issues map
	issues := make(map[int]string, len(w.watchedIssues))
	for k, v := range w.watchedIssues {
		issues[k] = v
	}

	return WatcherStatus{
		Running:       w.running,
		LastPoll:      w.lastPoll,
		PollInterval:  w.pollInterval,
		WatchedIssues: issues,
	}
}
