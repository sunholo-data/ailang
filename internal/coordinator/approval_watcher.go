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
	// Feedback contains human comments harvested from GitHub (may be empty)
	Feedback string
	// FeedbackAuthor is the author of the most recent human comment
	FeedbackAuthor string
	// Channel indicates where the approval came from (github, dashboard, cli)
	Channel string
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
//
// The watcher supports two modes:
// 1. Legacy hardcoded labels (design-approved, sprint-approved, merge-approved)
// 2. Config-driven labels (from AgentConfig.Approval)
//
// needs-revision is universal and works with both modes.
type ApprovalWatcher struct {
	mu            sync.Mutex
	poster        *GitHubPoster
	store         Store
	pollInterval  time.Duration
	handlers      map[ApprovalEventType]ApprovalHandler
	watchedIssues map[int]string    // issue number -> task ID
	lastPollTime  map[int]time.Time // issue number -> last poll time (for comment harvesting)
	stopCh        chan struct{}
	running       bool
	lastPoll      time.Time // track last successful poll for status reporting

	// Config-driven approval support (M-COORD-GENERIC-WORKFLOWS)
	agentRegistry  *AgentRegistry             // registry of agents with approval configs
	agentByLabel   map[string]*AgentConfig    // approved_label -> agent config
	customHandlers map[string]ApprovalHandler // approved_label -> handler
}

// NewApprovalWatcher creates a new approval watcher.
func NewApprovalWatcher(poster *GitHubPoster, store Store, pollInterval time.Duration) *ApprovalWatcher {
	if pollInterval == 0 {
		pollInterval = 60 * time.Second
	}
	return &ApprovalWatcher{
		poster:         poster,
		store:          store,
		pollInterval:   pollInterval,
		handlers:       make(map[ApprovalEventType]ApprovalHandler),
		watchedIssues:  make(map[int]string),
		lastPollTime:   make(map[int]time.Time),
		stopCh:         make(chan struct{}),
		agentByLabel:   make(map[string]*AgentConfig),
		customHandlers: make(map[string]ApprovalHandler),
	}
}

// SetAgentRegistry sets the agent registry for config-driven approval lookup.
func (w *ApprovalWatcher) SetAgentRegistry(registry *AgentRegistry) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.agentRegistry = registry
}

// RegisterAgentApproval registers an agent's approval configuration.
// This enables config-driven approval labels for the agent.
//
// When the approved_label is detected on a watched issue, the handler will be called.
// The handler receives an ApprovalEvent with the label and associated task.
//
// Example:
//
//	watcher.RegisterAgentApproval(agent, func(ctx context.Context, event *ApprovalEvent) error {
//	    // Handle custom approval
//	    return nil
//	})
func (w *ApprovalWatcher) RegisterAgentApproval(agent *AgentConfig, handler ApprovalHandler) error {
	if agent == nil {
		return fmt.Errorf("agent config is nil")
	}
	if agent.Approval == nil {
		return fmt.Errorf("agent %s has no approval config", agent.ID)
	}
	if agent.Approval.ApprovedLabel == "" {
		return fmt.Errorf("agent %s has empty approved_label", agent.ID)
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	label := agent.Approval.ApprovedLabel
	w.agentByLabel[label] = agent
	if handler != nil {
		w.customHandlers[label] = handler
	}

	log.Printf("[ApprovalWatcher] Registered approval label %q for agent %s", label, agent.ID)
	return nil
}

// GetAgentByLabel returns the agent config for a given approval label.
// Returns nil if no agent is registered for that label.
func (w *ApprovalWatcher) GetAgentByLabel(label string) *AgentConfig {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.agentByLabel[label]
}

// GetRegisteredLabels returns all registered custom approval labels.
func (w *ApprovalWatcher) GetRegisteredLabels() []string {
	w.mu.Lock()
	defer w.mu.Unlock()
	labels := make([]string, 0, len(w.agentByLabel))
	for label := range w.agentByLabel {
		labels = append(labels, label)
	}
	return labels
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
	delete(w.lastPollTime, issueNumber)
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
	pollTime := time.Now()
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

		// Update last poll time for this issue (for comment harvesting window)
		w.mu.Lock()
		w.lastPollTime[issueNum] = pollTime
		w.mu.Unlock()
	}

	if debugApprovalWatcher {
		log.Printf("[ApprovalWatcher] Poll cycle complete (%d issues, %d events)", len(issues), eventsFound)
	}
}

// checkIssueLabels checks if an issue has any approval labels.
// Checks both legacy hardcoded labels and config-driven labels.
// needs-revision is always checked first (universal across all workflows).
// When an approval label is found, also harvests recent human comments as feedback.
func (w *ApprovalWatcher) checkIssueLabels(ctx context.Context, issueNum int, taskID string) *ApprovalEvent {
	labels, err := w.poster.GetLabels(issueNum)
	if err != nil {
		log.Printf("[ApprovalWatcher] Error getting labels for issue #%d: %v", issueNum, err)
		return nil
	}

	if debugApprovalWatcher {
		log.Printf("[ApprovalWatcher] Issue #%d labels: %v", issueNum, labels)
	}

	// Helper to create event with harvested comments
	createEventWithComments := func(label string, eventType ApprovalEventType) *ApprovalEvent {
		event := &ApprovalEvent{
			TaskID:      taskID,
			IssueNumber: issueNum,
			Label:       label,
			EventType:   eventType,
			Channel:     "github",
		}

		// Harvest human comments since last poll
		w.mu.Lock()
		sinceTime := w.lastPollTime[issueNum]
		if sinceTime.IsZero() {
			// First time polling this issue - look back 5 minutes
			sinceTime = time.Now().Add(-5 * time.Minute)
		}
		w.mu.Unlock()

		comments, err := w.poster.GetRecentHumanComments(issueNum, sinceTime)
		if err != nil {
			log.Printf("[ApprovalWatcher] Error harvesting comments for issue #%d: %v", issueNum, err)
		} else if len(comments) > 0 {
			event.Feedback = ExtractFeedbackFromComments(comments)
			event.FeedbackAuthor = comments[len(comments)-1].Author
			if debugApprovalWatcher {
				log.Printf("[ApprovalWatcher] Harvested %d comments from issue #%d (feedback length: %d)",
					len(comments), issueNum, len(event.Feedback))
			}
		}

		return event
	}

	// First pass: Check for needs-revision (universal, highest priority)
	for _, label := range labels {
		if label == LabelNeedsRevision {
			return createEventWithComments(label, ApprovalEventRevision)
		}
	}

	// Second pass: Check config-driven labels (if any are registered)
	w.mu.Lock()
	customLabels := make(map[string]*AgentConfig, len(w.agentByLabel))
	for k, v := range w.agentByLabel {
		customLabels[k] = v
	}
	w.mu.Unlock()

	for _, label := range labels {
		if agent, ok := customLabels[label]; ok {
			return createEventWithComments(label, ApprovalEventType(fmt.Sprintf("custom:%s", agent.ID)))
		}
	}

	// Third pass: Check legacy hardcoded labels (for backwards compatibility)
	for _, label := range labels {
		switch label {
		case LabelDesignApproved:
			return createEventWithComments(label, ApprovalEventDesign)
		case LabelSprintApproved:
			return createEventWithComments(label, ApprovalEventSprint)
		case LabelMergeApproved:
			return createEventWithComments(label, ApprovalEventMerge)
		}
	}

	return nil
}

// handleEvent processes an approval event by calling the registered handler.
// CRITICAL: This function includes panic recovery to prevent handler panics from
// killing the poll goroutine. If a handler panics, we log the error and stack
// trace but continue polling. This prevents silent failures where the watcher
// stops working with no indication of why.
//
// Supports both legacy ApprovalEventType handlers and custom label handlers.
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
	// First check for custom handler by label (config-driven)
	customHandler, isCustom := w.customHandlers[event.Label]
	// Then check for legacy handler by event type
	handler, ok := w.handlers[event.EventType]
	w.mu.Unlock()

	// Prefer custom handler if available
	if isCustom {
		log.Printf("[ApprovalWatcher] Processing custom label %q for task %s (issue #%d)",
			event.Label, event.TaskID, event.IssueNumber)

		if err := customHandler(ctx, event); err != nil {
			log.Printf("[ApprovalWatcher] Custom handler error for %s: %v", event.Label, err)
			return
		}
	} else if ok {
		log.Printf("[ApprovalWatcher] Processing %s for task %s (issue #%d)",
			event.EventType, event.TaskID, event.IssueNumber)

		if err := handler(ctx, event); err != nil {
			log.Printf("[ApprovalWatcher] Handler error for %s: %v", event.EventType, err)
			return
		}
	} else {
		log.Printf("[ApprovalWatcher] No handler for event type: %s (label: %s)", event.EventType, event.Label)
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
	default:
		// For custom labels, check if the agent has a needs_label configured
		w.mu.Lock()
		if agent, ok := w.agentByLabel[event.Label]; ok && agent.Approval != nil {
			needsLabel = agent.Approval.NeedsLabel
		}
		w.mu.Unlock()
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
	Running       bool           `json:"running"`
	LastPoll      time.Time      `json:"last_poll"`
	PollInterval  time.Duration  `json:"poll_interval"`
	WatchedIssues map[int]string `json:"watched_issues"` // issue number -> task ID
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
