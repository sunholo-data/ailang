package coordinator

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/sunholo/ailang/internal/messaging"
	"github.com/sunholo/ailang/internal/websocket"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// stageToAgentID maps task stages to agent IDs for config lookup.
// Returns empty string for unknown stages (fallback to legacy behavior).
func stageToAgentID(stage TaskStage) string {
	switch stage {
	case TaskStageDesign:
		return "design-doc-creator"
	case TaskStageSprint:
		return "sprint-planner"
	case TaskStageImplementation:
		return "sprint-executor"
	default:
		return ""
	}
}

// initTaskProcessing initializes the message adapter, analyzer, and store
func (d *Daemon) initTaskProcessing() error {
	// Load agent configuration from ~/.ailang/config.yaml
	coordConfig, err := LoadCoordinatorConfig()
	if err != nil {
		return fmt.Errorf("failed to load coordinator config: %w", err)
	}
	d.coordConfig = coordConfig

	// Build agent registry
	d.agentRegistry = NewAgentRegistry()
	for _, agent := range coordConfig.Agents {
		if regErr := d.agentRegistry.Register(agent); regErr != nil {
			d.logger.Printf("Warning: Failed to register agent %q: %v", agent.ID, regErr)
		}
	}
	d.logger.Printf("Agent registry initialized with %d agent(s): %v",
		d.agentRegistry.Count(), d.agentRegistry.ListInboxes())

	// Validate agent references
	if issues := d.agentRegistry.Validate(); len(issues) > 0 {
		for _, issue := range issues {
			d.logger.Printf("Warning: Agent config issue: %s", issue)
		}
	}

	// Initialize inbox adapters for each configured agent
	d.inboxAdapters = make(map[string]*InboxMessageAdapter)
	d.worktreeManagers = make(map[string]*WorktreeManager)

	// Open shared message store
	adapter, store, err := OpenDefaultInboxAdapter("coordinator")
	if err != nil {
		return fmt.Errorf("failed to open inbox adapter: %w", err)
	}
	d.msgAdapter = adapter // Keep for backwards compatibility
	d.msgStore = store

	// Create adapters and worktree managers for each agent
	for _, agent := range coordConfig.Agents {
		// Create inbox adapter
		inboxAdapter := &InboxMessageAdapter{
			store: store,
			inbox: agent.Inbox,
		}
		d.inboxAdapters[agent.Inbox] = inboxAdapter
		d.logger.Printf("Watching inbox %q for agent %q", agent.Inbox, agent.ID)

		// Create worktree manager with agent's workspace
		workspace := agent.Workspace
		if workspace == "" || workspace == "." {
			// Use current working directory
			workspace, _ = os.Getwd()
		}
		worktreeBase := filepath.Join(d.config.StateDir, "worktrees", agent.ID)
		worktreeMgr, wmErr := NewWorktreeManager(workspace, worktreeBase, d.config.MaxWorktrees)
		if wmErr != nil {
			d.logger.Printf("Warning: Failed to create worktree manager for agent %q: %v", agent.ID, wmErr)
			continue
		}
		d.worktreeManagers[agent.ID] = worktreeMgr
		d.logger.Printf("Worktree manager ready for agent %q (base: %s)", agent.ID, worktreeBase)
	}

	// Initialize analyzer
	d.analyzer = NewTaskAnalyzer(0.8)

	// Initialize task store
	dbPath := filepath.Join(d.config.StateDir, "coordinator.db")
	taskStore, err := NewSQLiteStore(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open task store: %w", err)
	}
	d.taskStore = taskStore

	// Legacy worktree manager (for coordinator agent fallback)
	d.worktreeMgr = d.worktreeManagers["coordinator"]
	if d.worktreeMgr == nil {
		// Create default if coordinator wasn't configured
		worktreeMgr, wmErr := NewWorktreeManager("", filepath.Join(d.config.StateDir, "worktrees", "coordinator"), d.config.MaxWorktrees)
		if wmErr != nil {
			return fmt.Errorf("failed to create default worktree manager: %w", wmErr)
		}
		d.worktreeMgr = worktreeMgr
	}

	// Initialize task executor with available providers
	executor, err := DefaultTaskExecutor()
	if err != nil {
		d.logger.Printf("Warning: Failed to create task executor: %v", err)
		d.logger.Println("Task execution disabled - will queue tasks but not execute them")
	} else {
		d.executor = executor
		d.logger.Printf("Task executor initialized with providers: %v", executor.ListProviders())
	}

	// Initialize approval checkpoint for human-in-the-loop workflow
	d.approvalCheckpoint = NewApprovalCheckpoint(24 * time.Hour) // 24h default timeout
	d.approvalCheckpoint.SetCallback(func(request *ApprovalRequest) {
		d.logger.Printf("Approval request %s resolved: %s (task: %s)", request.ID, request.Status, request.TaskID)
	})
	d.logger.Println("Approval checkpoint initialized")

	// Initialize GitHub-driven approval workflow (M-COORD-GITHUB-AUTO-ROUTING)
	githubPoster, err := NewGitHubPoster()
	if err != nil {
		d.logger.Printf("Warning: GitHub poster not available: %v", err)
		d.logger.Println("GitHub-based approval workflow disabled")
	} else {
		d.githubPoster = githubPoster

		// Create approval watcher with poll interval from config (default 60s)
		pollInterval := d.config.ApprovalPollInterval
		if pollInterval == 0 {
			pollInterval = 60 * time.Second
		}
		d.approvalWatcher = NewApprovalWatcher(githubPoster, d.taskStore, pollInterval)

		// Create task chain and register approval handlers
		d.taskChain = NewTaskChain(githubPoster, d.taskStore, d.approvalWatcher)

		// Register approval event handlers
		d.approvalWatcher.RegisterHandler(ApprovalEventDesign, d.taskChain.OnDesignApproved)
		d.approvalWatcher.RegisterHandler(ApprovalEventSprint, d.taskChain.OnSprintApproved)
		d.approvalWatcher.RegisterHandler(ApprovalEventMerge, d.taskChain.OnMergeApproved)
		d.approvalWatcher.RegisterHandler(ApprovalEventRevision, d.taskChain.OnNeedsRevision)

		// Load existing watched issues from store
		if err := d.approvalWatcher.LoadWatchedIssuesFromStore(d.ctx); err != nil {
			d.logger.Printf("Warning: Failed to load watched issues: %v", err)
		} else {
			d.logger.Printf("GitHub approval watcher initialized (watching %d issue(s), poll interval: %v)",
				d.approvalWatcher.WatchedIssueCount(), pollInterval)
		}
	}

	// Recover stale tasks from previous daemon runs
	// Tasks that were running/queued when daemon crashed are marked cancelled
	if d.taskStore != nil {
		staleThreshold := 5 * time.Minute // Tasks idle for >5 min are considered stale
		recovered, err := d.taskStore.RecoverStaleTasks(d.ctx, staleThreshold)
		if err != nil {
			d.logger.Printf("Warning: Failed to recover stale tasks: %v", err)
		} else if recovered > 0 {
			d.logger.Printf("Recovered %d stale task(s) from previous daemon run", recovered)
		}
	}

	return nil
}

// pollAndProcessTasks polls for new messages and queues them as tasks
func (d *Daemon) pollAndProcessTasks() error {
	// Collect messages from all inbox adapters
	type inboxMessage struct {
		inbox   string
		agentID string
		msg     *Message // coordinator.Message type
	}
	var allMessages []inboxMessage

	for inbox, adapter := range d.inboxAdapters {
		messages, err := adapter.ListUnread()
		if err != nil {
			d.logger.Printf("Warning: Failed to list messages from inbox %q: %v", inbox, err)
			continue
		}
		agent := d.agentRegistry.GetAgentForInbox(inbox)
		agentID := ""
		if agent != nil {
			agentID = agent.ID
		}
		for _, msg := range messages {
			allMessages = append(allMessages, inboxMessage{
				inbox:   inbox,
				agentID: agentID,
				msg:     msg,
			})
		}
	}

	if len(allMessages) == 0 {
		return nil
	}

	d.logger.Printf("Found %d unread messages across %d inboxes", len(allMessages), len(d.inboxAdapters))

	for _, im := range allMessages {
		msg := im.msg
		agentID := im.agentID

		// Create a Task for the analyzer
		taskID := fmt.Sprintf("task-%s", msg.ID[:8])

		// Determine kind - use message kind if set, otherwise infer from type
		kind := msg.Kind
		if kind == "" {
			if msg.Type == "question" || msg.Type == "research" {
				kind = "question"
			} else {
				kind = "directive"
			}
		}

		taskInput := &Task{
			ID:        taskID,
			Title:     msg.Title,
			Content:   msg.Content,
			Kind:      kind,
			MessageID: msg.ID,
			CreatedAt: msg.CreatedAt,
		}

		// Analyze the message to classify it
		analyzed := d.analyzer.Analyze(taskInput)

		// Create a task record
		// Get the agent's workspace path from the registry
		workspace := ""
		if agent := d.agentRegistry.GetAgentByID(agentID); agent != nil && agent.Workspace != "" {
			workspace = agent.Workspace
		} else {
			// Fallback to current directory - warn as skills may not be available
			workspace, _ = os.Getwd()
			d.logger.Printf("WARNING: Agent %q has no workspace configured, using current directory: %s (skills may not be available)", agentID, workspace)
		}

		task := &TaskRecord{
			ID:          taskID,
			MessageID:   msg.ID,
			AgentID:     agentID, // M-COORD-ARTIFACT-DISCOVERY: Set AgentID from inbox
			Title:       msg.Title,
			Content:     msg.Content,
			Type:        analyzed.Type,
			Kind:        kind,
			Priority:    CalculatePriority(analyzed),
			Status:      TaskStatusPending,
			Workspace:   workspace,
			GithubIssue: msg.GithubIssue, // M-COORD-GITHUB-AUTO-ROUTING
			CreatedAt:   msg.CreatedAt,
		}

		// Check for duplicates
		fingerprint := analyzed.Fingerprint
		if fingerprint != 0 {
			if dup, _ := d.taskStore.FindDuplicateTask(d.ctx, fingerprint, 0.9); dup != nil {
				d.logger.Printf("Skipping duplicate task for message %s (similar to task %s)", msg.ID, dup.ID)
				// Mark message as read since we're skipping it
				if adapter := d.inboxAdapters[im.inbox]; adapter != nil {
					_ = adapter.MarkAsRead(msg.ID)
				}
				continue
			}
		}

		// Create a thread in collaboration.db for dashboard visibility
		targetAgent := agentID
		if targetAgent == "" {
			targetAgent = "coordinator"
		}
		thread, err := d.msgStore.CreateThreadWithWorkspace(
			msg.Title,         // title
			"ailang_instance", // createdByType (constraint: 'human' or 'ailang_instance')
			"coordinator",     // createdByID
			targetAgent,       // targetAgent - the agent that will handle this task
			workspace,         // workspace - source project/agent
		)
		if err != nil {
			d.logger.Printf("Failed to create thread for task %s: %v", taskID, err)
			// Continue anyway - thread is for visibility, not required for task
		} else {
			task.ThreadID = thread.ID
			d.logger.Printf("Created thread %s for task %s (agent: %s)", thread.ID, taskID, targetAgent)
		}

		// Store the task
		if err := d.taskStore.CreateTask(d.ctx, task); err != nil {
			d.logger.Printf("Failed to create task for message %s: %v", msg.ID, err)
			continue
		}

		// Set fingerprint for deduplication
		if fingerprint != 0 {
			_ = d.taskStore.SetTaskFingerprint(d.ctx, task.ID, fingerprint)
		}

		// M-COORD-GITHUB-AUTO-ROUTING: Initialize GitHub-linked tasks
		if task.GithubIssue > 0 && d.taskChain != nil {
			// Start the task chain (posts "working" comment to GitHub)
			if err := d.taskChain.StartTask(d.ctx, task.ID, task.GithubIssue); err != nil {
				d.logger.Printf("Warning: Failed to start task chain for issue #%d: %v", task.GithubIssue, err)
			} else {
				d.logger.Printf("Started GitHub pipeline for task %s (issue #%d)", task.ID, task.GithubIssue)
			}

			// Start watching for approval labels
			if d.approvalWatcher != nil {
				d.approvalWatcher.WatchIssue(task.GithubIssue, task.ID)
			}
		}

		d.logger.Printf("Created task %s (type: %s, priority: %d, thread: %s, agent: %s, issue: #%d) from message %s",
			task.ID, task.Type, task.Priority, task.ThreadID, agentID, task.GithubIssue, msg.ID)

		// Mark message as read using the correct inbox adapter
		if adapter := d.inboxAdapters[im.inbox]; adapter != nil {
			if err := adapter.MarkAsRead(msg.ID); err != nil {
				d.logger.Printf("Failed to mark message as read: %v", err)
			}
		}

		d.tasksRun++
	}

	return nil
}

// executeTaskQueue picks up pending tasks and executes them
func (d *Daemon) executeTaskQueue() error {
	if d.executor == nil {
		return nil // No executor available
	}

	// Get pending tasks, ordered by priority (highest first)
	filter := &TaskFilter{
		Status:    []TaskStatus{TaskStatusPending},
		OrderBy:   "priority",
		OrderDesc: true, // Higher priority first
		Limit:     1,    // Process one at a time for now
	}

	tasks, err := d.taskStore.ListTasks(d.ctx, filter)
	if err != nil {
		return fmt.Errorf("failed to list pending tasks: %w", err)
	}

	if len(tasks) == 0 {
		return nil
	}

	for _, task := range tasks {
		if err := d.executeTask(task); err != nil {
			d.logger.Printf("Failed to execute task %s: %v", task.ID, err)
			// Mark task as failed
			_ = d.taskStore.MarkTaskFailed(d.ctx, task.ID, err)
			// Post failure message to thread
			d.postTaskResult(task, nil, err)
		}
	}

	return nil
}

// coordinatorTracer returns the tracer for coordinator instrumentation.
var coordinatorTracer = otel.Tracer("coordinator")

// executeTask runs a single task through the executor
func (d *Daemon) executeTask(task *TaskRecord) error {
	// Start OTEL span for task execution
	ctx, span := coordinatorTracer.Start(d.ctx, "coordinator.task.execute",
		trace.WithAttributes(
			attribute.String("task.id", task.ID),
			attribute.String("task.type", string(task.Type)),
			attribute.String("task.stage", string(task.Stage)),
			attribute.String("task.title", task.Title),
		),
	)
	defer span.End()

	// Use traced context for the rest of the function
	d.ctx = ctx

	d.logger.Printf("Starting execution of task %s (type: %s)", task.ID, task.Type)

	// Mark task as running
	if err := d.taskStore.MarkTaskRunning(d.ctx, task.ID, "", ""); err != nil {
		return fmt.Errorf("failed to mark task as running: %w", err)
	}

	// Post "starting" message to thread
	d.postTaskStatus(task, "running", "Starting task execution...")

	// Create analyzed task for executor
	// Use config-driven directive for GitHub-linked tasks (M-COORD-GENERIC-WORKFLOWS)
	// Look up agent config for this stage to use proper skill/markers
	var agentConfig *AgentConfig
	if d.agentRegistry != nil {
		agentID := stageToAgentID(task.Stage)
		d.logger.Printf("[DEBUG] Task stage: %s -> agentID: %s", task.Stage, agentID)
		if agentID != "" {
			agentConfig = d.agentRegistry.GetAgentByID(agentID)
			if agentConfig != nil {
				d.logger.Printf("[DEBUG] Found agent config: ID=%s, Invoke=%+v", agentConfig.ID, agentConfig.Invoke)
			} else {
				d.logger.Printf("[DEBUG] No agent config found for ID: %s", agentID)
			}
		}
	}
	directive := BuildDirectiveFromConfig(task, agentConfig)
	d.logger.Printf("[DEBUG] Built directive (first 500 chars): %s", truncateString(directive, 500))
	analyzed := &AnalyzedTask{
		Task: &Task{
			ID:        task.ID,
			Title:     task.Title,
			Content:   directive, // Stage-aware: design/sprint/implementation prompts
			Kind:      task.Kind,
			MessageID: task.MessageID,
			CreatedAt: task.CreatedAt,
		},
		Type: task.Type,
	}

	// Determine which agent should handle this task
	// Look up the thread to get the target agent
	targetAgent := "coordinator" // default
	if task.ThreadID != "" && d.msgStore != nil {
		if thread, err := d.msgStore.GetThread(task.ThreadID); err == nil && thread != nil && thread.TargetAgent != "" {
			targetAgent = thread.TargetAgent
		}
	}

	// Get the correct worktree manager for this agent
	worktreeMgr := d.worktreeManagers[targetAgent]
	if worktreeMgr == nil {
		worktreeMgr = d.worktreeMgr // Fallback to default
	}

	// Create worktree for task isolation
	var worktree *Worktree
	if worktreeMgr != nil {
		var wtErr error
		worktree, wtErr = worktreeMgr.CreateWorktree(task.ID)
		if wtErr != nil {
			// Check if this is a recoverable "limit reached" error
			if errors.Is(wtErr, ErrWorktreeLimitReached) {
				// Leave task in pending state - it will be retried when a worktree frees up
				d.logger.Printf("INFO: Task %s waiting for worktree (limit reached), will retry", task.ID)
				return wtErr // Return error but don't mark as failed
			}

			// CRITICAL: Do NOT continue without worktree!
			// Agent would modify main repo directly, causing data loss.
			d.logger.Printf("ERROR: Failed to create worktree for task %s (agent: %s): %v", task.ID, targetAgent, wtErr)
			failErr := fmt.Errorf("worktree creation failed: %w (check 'ailang coordinator cleanup' to free slots)", wtErr)
			if err := d.taskStore.MarkTaskFailed(d.ctx, task.ID, failErr); err != nil {
				d.logger.Printf("Warning: Failed to mark task %s as failed: %v", task.ID, err)
			}
			return failErr // Don't execute without worktree isolation
		}
	} else {
		// No worktree manager means we can't ensure isolation
		d.logger.Printf("ERROR: No worktree manager available for task %s (agent: %s)", task.ID, targetAgent)
		failErr := fmt.Errorf("no worktree manager available for agent %s", targetAgent)
		if err := d.taskStore.MarkTaskFailed(d.ctx, task.ID, failErr); err != nil {
			d.logger.Printf("Warning: Failed to mark task %s as failed: %v", task.ID, err)
		}
		return failErr
	}

	opts := &ExecuteOptions{
		Timeout:   10 * time.Minute, // 10 minute timeout per task
		Workspace: worktree.Path,    // Always have worktree (fail fast above)
	}

	// Create streaming event handler if broadcaster is available
	var eventHandler *CoordinatorEventHandler
	if d.eventBroadcaster != nil {
		eventHandler = NewCoordinatorEventHandler(task.ID, task.ThreadID, d.eventBroadcaster)
		opts.EventHandler = eventHandler

		// Set up event storage for historical replay
		if sqliteStore, ok := d.taskStore.(*SQLiteStore); ok {
			eventHandler.SetEventStorer(func(record *TaskEventRecord) error {
				return sqliteStore.StoreTaskEvent(d.ctx, record)
			})
		}

		// Emit starting status
		eventHandler.EmitStatus("running")
	}

	// Create resource tracker for monitoring
	// Use current PID as initial tracker - executor will spawn subprocess
	resourceTracker := NewResourceTracker(task.ID, task.ThreadID, 0)
	d.resourceRegistry.Register(task.ID, resourceTracker)

	// Set up update callback for WebSocket broadcasting
	if d.eventBroadcaster != nil {
		resourceTracker.SetUpdateCallback(func(metrics *ResourceMetrics) {
			// Broadcast metrics update via event handler status
			if eventHandler != nil {
				eventHandler.UpdateMetrics(metrics.TokensIn, metrics.TokensOut, metrics.Cost)
			}
		})
	}

	// Start resource tracking
	resourceTracker.Start(d.ctx)

	// Execute the task
	result, err := d.executor.ExecuteWithRetry(d.ctx, analyzed, opts, 2)

	// Stop resource tracking and get final metrics
	resourceTracker.Stop()
	finalMetrics := resourceTracker.GetMetrics()
	d.resourceRegistry.Unregister(task.ID)

	// Emit completion status via event handler
	if eventHandler != nil {
		if err != nil {
			eventHandler.OnError(err)
			eventHandler.EmitStatus("failed")
		} else if result != nil {
			eventHandler.UpdateMetrics(result.InputTokens, result.OutputTokens, result.Cost)
			if result.Success {
				eventHandler.EmitStatus("completed")
			} else {
				eventHandler.EmitStatus("failed")
			}
		}
	}

	// Store peak resource metrics in task record
	if finalMetrics != nil {
		_ = d.taskStore.UpdateTaskMetrics(d.ctx, task.ID, finalMetrics.PeakCPU, finalMetrics.PeakMemory)
	}

	if err != nil {
		// Record error in span
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())

		// Cleanup worktree on error (no work to preserve)
		if worktree != nil && d.worktreeMgr != nil {
			if rmErr := d.worktreeMgr.RemoveWorktree(task.ID); rmErr != nil {
				d.logger.Printf("Warning: Failed to remove worktree: %v", rmErr)
			}
		}
		return fmt.Errorf("executor error: %w", err)
	}

	// Record result metrics in span
	if result != nil {
		span.SetAttributes(
			attribute.Int64("task.tokens_in", int64(result.InputTokens)),
			attribute.Int64("task.tokens_out", int64(result.OutputTokens)),
			attribute.Float64("task.cost_usd", result.Cost),
			attribute.Bool("task.success", result.Success),
		)
	}

	// Update task status based on result
	if result.Success {
		// PRESERVE WORKTREE - mark as pending approval, not completed
		// Human must approve/reject before worktree is cleaned up
		worktreePath := ""
		worktreeBranch := ""
		if worktree != nil {
			worktreePath = worktree.Path
			worktreeBranch = worktree.Branch
		}
		if err := d.taskStore.MarkTaskPendingApproval(d.ctx, task.ID, worktreePath, worktreeBranch, result); err != nil {
			d.logger.Printf("Warning: Failed to mark task pending approval: %v", err)
		}

		// Update task with worktree and agent info before ProcessStageCompletion
		// (needed for git diff artifact discovery)
		task.WorktreePath = worktreePath
		task.AgentID = targetAgent

		// M-COORD-GITHUB-AUTO-ROUTING: Process stage completion for GitHub-linked tasks
		// This posts the summary to GitHub and adds the appropriate approval label
		if err := d.ProcessStageCompletion(d.ctx, task, result); err != nil {
			d.logger.Printf("Warning: Failed to process stage completion: %v", err)
		}

		// Create approval request record for the CLI/dashboard to show
		approvalID := fmt.Sprintf("apr-%s", task.ID[5:]) // apr-<hash> from task-<hash>
		approvalReq := &ApprovalRequestRecord{
			ID:          approvalID,
			TaskID:      task.ID,
			Type:        string(ApprovalTypeMerge),
			Description: fmt.Sprintf("Agent completed work on: %s", task.Title),
			Status:      "pending",
			CreatedAt:   time.Now(),
		}
		if err := d.taskStore.CreateApprovalRequest(d.ctx, approvalReq); err != nil {
			d.logger.Printf("Warning: Failed to create approval request: %v", err)
		}

		d.logger.Printf("Task %s awaiting approval (cost: $%.4f, tokens: %d, worktree: %s)",
			task.ID, result.Cost, result.TokensUsed, worktreePath)

		// Check for agent-to-agent handoffs
		// Find the agent that handled this task and check for trigger_on_complete
		if err := d.handleAgentHandoffs(task, result); err != nil {
			d.logger.Printf("Warning: Failed to process agent handoffs: %v", err)
		}
	} else {
		// Failed tasks: cleanup worktree (no useful work to preserve)
		if worktree != nil && d.worktreeMgr != nil {
			if rmErr := d.worktreeMgr.RemoveWorktree(task.ID); rmErr != nil {
				d.logger.Printf("Warning: Failed to remove worktree: %v", rmErr)
			}
		}
		if err := d.taskStore.MarkTaskFailed(d.ctx, task.ID, fmt.Errorf("%s", result.Error)); err != nil {
			d.logger.Printf("Warning: Failed to mark task failed: %v", err)
		}
		d.logger.Printf("Task %s failed: %s", task.ID, result.Error)
	}

	// Post result to thread
	d.postTaskResult(task, result, nil)

	return nil
}

// postTaskStatus posts a status update to the task's thread
func (d *Daemon) postTaskStatus(task *TaskRecord, status, message string) {
	if task.ThreadID == "" || d.msgStore == nil {
		return
	}

	content := fmt.Sprintf("**Status: %s**\n\n%s", status, message)
	_, err := d.msgStore.CreateMessage(
		task.ThreadID,
		"ailang_instance", "coordinator", // from
		"human", "user", // to (for visibility)
		"status",
		content,
		"",
	)
	if err != nil {
		d.logger.Printf("Failed to post status to thread %s: %v", task.ThreadID, err)
	}
}

// postTaskResult posts the execution result to the task's thread and records metrics
func (d *Daemon) postTaskResult(task *TaskRecord, result *ExecuteResult, execErr error) {
	if task.ThreadID == "" || d.msgStore == nil {
		return
	}

	var content string
	var kind string

	if execErr != nil {
		kind = "error"
		content = fmt.Sprintf("**Task Failed**\n\n❌ Error: %v", execErr)
	} else if result != nil {
		if result.Success {
			kind = "result"
			content = fmt.Sprintf("**Task Completed Successfully**\n\n"+
				"- **Provider:** %s\n"+
				"- **Duration:** %s\n"+
				"- **Cost:** $%.4f\n"+
				"- **Tokens:** %d (in: %d, out: %d)\n\n"+
				"---\n\n%s",
				result.Provider, result.Duration, result.Cost,
				result.TokensUsed, result.InputTokens, result.OutputTokens, result.Output)

			if len(result.FilesCreated) > 0 {
				content += fmt.Sprintf("\n\n**Files Created:** %v", result.FilesCreated)
			}
			if len(result.FilesModified) > 0 {
				content += fmt.Sprintf("\n\n**Files Modified:** %v", result.FilesModified)
			}
		} else {
			kind = "error"
			content = fmt.Sprintf("**Task Failed**\n\n"+
				"- **Provider:** %s\n"+
				"- **Duration:** %s\n\n"+
				"**Error:** %s",
				result.Provider, result.Duration, result.Error)
		}
	} else {
		kind = "error"
		content = "**Task Failed**\n\n❌ Unknown error"
	}

	// Create metadata with execution_stats format expected by metrics system
	metadataJSON := ""
	if result != nil {
		metadata := map[string]interface{}{
			"execution_stats": map[string]interface{}{
				"duration_ms":    result.Duration.Milliseconds(),
				"input_tokens":   result.InputTokens,
				"output_tokens":  result.OutputTokens,
				"cost":           result.Cost, // In dollars
				"files_created":  result.FilesCreated,
				"files_modified": result.FilesModified,
			},
		}
		if data, err := json.Marshal(metadata); err == nil {
			metadataJSON = string(data)
		}
	}

	_, err := d.msgStore.CreateMessage(
		task.ThreadID,
		"ailang_instance", "coordinator",
		"human", "user",
		kind,
		content,
		metadataJSON,
	)
	if err != nil {
		d.logger.Printf("Failed to post result to thread %s: %v", task.ThreadID, err)
		return
	}

	// Record metrics at global, agent, and thread levels
	if result != nil {
		stats := &messaging.MessageExecutionStats{
			DurationMS:   int(result.Duration.Milliseconds()),
			InputTokens:  result.InputTokens,
			OutputTokens: result.OutputTokens,
			CostCents:    int(result.Cost * 100), // Convert dollars to cents
			FilesCreated: result.FilesCreated,
		}
		if err := d.msgStore.RecordMetrics(task.ThreadID, "coordinator", stats); err != nil {
			d.logger.Printf("Failed to record metrics: %v", err)
		} else {
			d.logger.Printf("Recorded metrics: thread=%s, tokens=%d, cost=$%.4f",
				task.ThreadID, stats.InputTokens+stats.OutputTokens, result.Cost)
		}
	}
}

// syncWorktreeState syncs worktree manager in-memory state with actual git worktrees.
// This handles cases where worktrees are removed externally (e.g., via CLI rejection/cleanup).
func (d *Daemon) syncWorktreeState() {
	for agentID, wm := range d.worktreeManagers {
		cleaned, err := wm.CleanupOrphaned()
		if err != nil {
			d.logger.Printf("Warning: Failed to sync worktrees for agent %s: %v", agentID, err)
		} else if cleaned > 0 {
			d.logger.Printf("Synced worktrees for agent %s: removed %d orphaned slot(s)", agentID, cleaned)
		}
	}
}

// initHTTPBroadcaster initializes the HTTP broadcaster for streaming events to dashboard
func (d *Daemon) initHTTPBroadcaster() error {
	serverURL := DefaultServerURL()

	broadcaster := NewHTTPBroadcaster(serverURL, d.logger)

	// Check if server is reachable
	if !broadcaster.CheckServerAvailable() {
		return fmt.Errorf("Collaboration Hub server not available at %s", serverURL)
	}

	// Set the broadcaster
	d.SetEventBroadcaster(broadcaster.BroadcastFunc())
	d.logger.Printf("HTTP broadcaster initialized, streaming to %s", serverURL)

	return nil
}

// SetEventBroadcaster sets the event broadcaster for real-time task updates.
// When set, task execution events will be streamed via the broadcaster.
func (d *Daemon) SetEventBroadcaster(broadcaster EventBroadcaster) {
	d.eventBroadcaster = broadcaster
}

// CreateWebSocketBroadcaster creates an EventBroadcaster that broadcasts to a WebSocket server.
// This is used when the coordinator is connected to the collaboration hub server.
func CreateWebSocketBroadcaster(wsServer interface {
	BroadcastTaskEvent(stream *websocket.TaskStreamEvent)
}) EventBroadcaster {
	return func(event *websocket.TaskStreamEvent) {
		wsServer.BroadcastTaskEvent(event)
	}
}
