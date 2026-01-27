package coordinator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/sunholo/ailang/internal/messaging"
	"github.com/sunholo/ailang/internal/observatory"
	"github.com/sunholo/ailang/internal/telemetry"
	"github.com/sunholo/ailang/internal/websocket"
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

	// Initialize Observatory sync for trace linking (M-TASK-HIERARCHY)
	// Uses the same state directory for the observatory database
	obsDBPath := filepath.Join(d.config.StateDir, "observatory.db")
	obsBackend, err := observatory.NewSQLiteBackendFromPath(obsDBPath)
	if err != nil {
		d.logger.Printf("Warning: Failed to initialize Observatory backend: %v", err)
		d.logger.Println("Observatory trace linking disabled")
	} else {
		d.observatorySync = NewObservatorySync(obsBackend, d.logger)
		d.logger.Printf("Observatory sync initialized (db: %s)", obsDBPath)
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

	// Clean up worktrees for tasks in terminal states (cancelled, completed, failed, rejected)
	// This handles worktrees left behind by RecoverStaleTasks and any other cleanup gaps
	d.cleanupWorktreesForTerminalTasks()

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

		// Set iteration: use message iteration if provided, otherwise default to 1
		iteration := msg.Iteration
		if iteration == 0 {
			iteration = 1 // First run is iteration 1
		}

		task := &TaskRecord{
			ID:            taskID,
			MessageID:     msg.ID,
			AgentID:       agentID,          // M-COORD-ARTIFACT-DISCOVERY: Set AgentID from inbox
			ParentTaskID:  msg.ParentTaskID, // M-TASK-HIERARCHY: Link to parent task for handoff chains
			Iteration:     iteration,        // M-TASK-HIERARCHY: Iteration number for feedback loops
			Title:         msg.Title,
			Content:       msg.Content,
			Type:          analyzed.Type,
			Kind:          kind,
			Priority:      CalculatePriority(analyzed),
			Status:        TaskStatusPending,
			Workspace:     workspace,
			GithubIssue:   msg.GithubIssue, // M-COORD-GITHUB-AUTO-ROUTING
			GithubRepo:    msg.GithubRepo,  // M-COORD-GITHUB-CLOSE-ON-MERGE
			CreatedAt:     msg.CreatedAt,
			Capabilities:  analyzed.Capabilities,  // M-DEPRECATE-AILANG-AGENT
			ImpactLevel:   analyzed.ImpactLevel,   // M-DEPRECATE-AILANG-AGENT
			EstimatedCost: analyzed.EstimatedCost, // M-DEPRECATE-AILANG-AGENT
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

		// Get or create a thread in collaboration.db for dashboard visibility
		// Using GetOrCreate prevents duplicate threads when message isn't marked read properly
		targetAgent := agentID
		if targetAgent == "" {
			targetAgent = "coordinator"
		}
		thread, created, err := d.msgStore.GetOrCreateThreadWithWorkspace(
			msg.Title,         // title
			"ailang_instance", // createdByType (constraint: 'human' or 'ailang_instance')
			"coordinator",     // createdByID
			targetAgent,       // targetAgent - the agent that will handle this task
			workspace,         // workspace - source project/agent
		)
		if err != nil {
			d.logger.Printf("Failed to get/create thread for task %s: %v", taskID, err)
			// Continue anyway - thread is for visibility, not required for task
		} else {
			task.ThreadID = thread.ID
			if created {
				d.logger.Printf("Created thread %s for task %s (agent: %s)", thread.ID, taskID, targetAgent)
			} else {
				d.logger.Printf("Reusing existing thread %s for task %s (agent: %s)", thread.ID, taskID, targetAgent)
			}
		}

		// Store the task
		if err := d.taskStore.CreateTask(d.ctx, task); err != nil {
			d.logger.Printf("Failed to create task for message %s: %v", msg.ID, err)
			// Still mark message as read even if task creation fails (e.g., duplicate task)
			// This prevents infinite loops when the same message keeps being processed
			if adapter := d.inboxAdapters[im.inbox]; adapter != nil {
				if markErr := adapter.MarkAsRead(msg.ID); markErr != nil {
					d.logger.Printf("Failed to mark message as read after task error: %v", markErr)
				} else {
					d.logger.Printf("Marked message %s as read (task already existed)", msg.ID)
				}
			}
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

		// Log capability detection results
		capStr := "none"
		if len(analyzed.Capabilities) > 0 {
			capTypes := make([]string, len(analyzed.Capabilities))
			for i, cap := range analyzed.Capabilities {
				capTypes[i] = string(cap.Type)
			}
			capStr = fmt.Sprintf("%v", capTypes)
		}
		d.logger.Printf("Created task %s (type: %s, priority: %d, impact: %s, caps: %s, est_cost: $%.2f, agent: %s, issue: #%d) from message %s",
			task.ID, task.Type, task.Priority, task.ImpactLevel, capStr, task.EstimatedCost, agentID, task.GithubIssue, msg.ID)

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
var coordinatorTracer = telemetry.Tracer("coordinator")

// executeTask runs a single task through the executor
func (d *Daemon) executeTask(task *TaskRecord) error {
	// Start OTEL span for task execution with a NEW trace root.
	// CRITICAL: Use context.Background() instead of d.ctx to avoid trace contamination
	// between tasks. Each task gets its own trace_id instead of all tasks sharing
	// the daemon's trace context (which caused Phase 16 trace mixing bug).
	taskCtx, span := telemetry.StartSpan(context.Background(), coordinatorTracer, "coordinator.task.execute",
		trace.WithNewRoot(), // Force new trace, don't inherit from previous tasks
		trace.WithAttributes(
			attribute.String("task.id", task.ID),
			attribute.String("task.type", string(task.Type)),
			attribute.String("task.stage", string(task.Stage)),
			attribute.String("task.title", task.Title),
			attribute.Int("task.iteration", task.Iteration), // M-TRANSCRIPT: feedback loop iteration
		),
	)
	defer span.End()

	// NOTE: Intentionally NOT mutating d.ctx to avoid trace contamination.
	// Use taskCtx for all operations within this task execution.

	d.logger.Printf("Starting execution of task %s (type: %s)", task.ID, task.Type)

	// Mark task as running
	if err := d.taskStore.MarkTaskRunning(taskCtx, task.ID, "", ""); err != nil {
		return fmt.Errorf("failed to mark task as running: %w", err)
	}

	// Post "starting" message to thread
	d.postTaskStatus(task, "running", "Starting task execution...")

	// Determine which agent should handle this task
	// Look up the thread to get the target agent
	targetAgent := "coordinator" // default
	if task.ThreadID != "" && d.msgStore != nil {
		if thread, err := d.msgStore.GetThread(task.ThreadID); err == nil && thread != nil && thread.TargetAgent != "" {
			targetAgent = thread.TargetAgent
		}
	}

	// Create analyzed task for executor
	// Use config-driven directive for GitHub-linked tasks (M-COORD-GENERIC-WORKFLOWS)
	// Look up agent config by targetAgent (from thread) or fall back to stage-based lookup
	var agentConfig *AgentConfig
	if d.agentRegistry != nil {
		// First try targetAgent (from thread - for direct messages to inboxes)
		agentConfig = d.agentRegistry.GetAgentByID(targetAgent)
		if agentConfig == nil {
			// Fall back to stage-based lookup (for workflow stages)
			agentID := stageToAgentID(task.Stage)
			d.logger.Printf("[DEBUG] Task stage: %s -> agentID: %s", task.Stage, agentID)
			if agentID != "" {
				agentConfig = d.agentRegistry.GetAgentByID(agentID)
			}
		}
		if agentConfig != nil {
			d.logger.Printf("[DEBUG] Found agent config: ID=%s, Invoke=%+v", agentConfig.ID, agentConfig.Invoke)
		} else {
			d.logger.Printf("[DEBUG] No agent config found for agent: %s", targetAgent)
		}
	}

	// Budget enforcement check (M-PER-PROVIDER-BUDGETS)
	// Check budget before executing task. If hard limit exceeded, create approval request.
	if budgetBlocked, budgetErr := d.checkBudgetBeforeExecution(taskCtx, task, agentConfig); budgetBlocked {
		if budgetErr != nil {
			d.logger.Printf("Budget check failed for task %s: %v", task.ID, budgetErr)
		}
		return nil // Task blocked, waiting for budget approval
	}

	directive := BuildDirectiveFromConfig(task, agentConfig)
	d.logger.Printf("[DEBUG] Built directive (first 500 chars): %s", truncateString(directive, 500))
	analyzed := &AnalyzedTask{
		Task: &Task{
			ID:           task.ID,
			Title:        task.Title,
			Content:      directive, // Stage-aware: design/sprint/implementation prompts
			Kind:         task.Kind,
			MessageID:    task.MessageID,
			ParentTaskID: task.ParentTaskID, // M-TASK-HIERARCHY
			CreatedAt:    task.CreatedAt,
			Iteration:    task.Iteration, // M-TRANSCRIPT: feedback loop iteration
			SessionID:    task.SessionID, // M-TRANSCRIPT: for Claude --resume
		},
		Type: task.Type,
	}

	// Check if this is a script agent (doesn't need worktree isolation)
	isScriptAgent := agentConfig != nil && agentConfig.Invoke != nil && agentConfig.Invoke.Type == "script"

	// Get the correct worktree manager for this agent
	worktreeMgr := d.worktreeManagers[targetAgent]
	if worktreeMgr == nil {
		worktreeMgr = d.worktreeMgr // Fallback to default
	}

	// Create worktree for task isolation (AI agents only)
	// Script agents run deterministic commands and don't modify files, so they don't need isolation
	var worktree *Worktree
	var workspacePath string

	if isScriptAgent {
		// Script agents use workspace directly - no worktree needed
		workspacePath = task.Workspace
		if workspacePath == "" && agentConfig != nil {
			workspacePath = agentConfig.Workspace
		}
		d.logger.Printf("Script agent %s using workspace directly: %s", targetAgent, workspacePath)
	} else if worktreeMgr != nil {
		var wtErr error
		// Get merge branch from agent config (inherits from global if not set)
		mergeBranch := ""
		if agentConfig != nil {
			mergeBranch = agentConfig.MergeBranch
		}
		worktree, wtErr = worktreeMgr.CreateWorktree(task.ID, mergeBranch)
		if wtErr != nil {
			// Check if this is a recoverable "limit reached" error
			if errors.Is(wtErr, ErrWorktreeLimitReached) {
				// Reset task back to pending - it will be retried when a worktree frees up
				// CRITICAL: Task was marked "running" at line 462, must reset or it's stuck forever!
				d.logger.Printf("WARN: Task %s worktree limit reached, resetting to pending for retry", task.ID)
				if resetErr := d.taskStore.ResetTaskToPending(taskCtx, task.ID); resetErr != nil {
					d.logger.Printf("ERROR: Failed to reset task %s to pending: %v", task.ID, resetErr)
				}
				return wtErr // Return error, task will be picked up on next poll
			}

			// CRITICAL: Do NOT continue without worktree!
			// Agent would modify main repo directly, causing data loss.
			d.logger.Printf("ERROR: Failed to create worktree for task %s (agent: %s): %v", task.ID, targetAgent, wtErr)
			failErr := fmt.Errorf("worktree creation failed: %w (check 'ailang coordinator cleanup' to free slots)", wtErr)
			if err := d.taskStore.MarkTaskFailed(taskCtx, task.ID, failErr); err != nil {
				d.logger.Printf("Warning: Failed to mark task %s as failed: %v", task.ID, err)
			}
			return failErr // Don't execute without worktree isolation
		}
		workspacePath = worktree.Path
	} else {
		// No worktree manager means we can't ensure isolation
		d.logger.Printf("ERROR: No worktree manager available for task %s (agent: %s)", task.ID, targetAgent)
		failErr := fmt.Errorf("no worktree manager available for agent %s", targetAgent)
		if err := d.taskStore.MarkTaskFailed(taskCtx, task.ID, failErr); err != nil {
			d.logger.Printf("Warning: Failed to mark task %s as failed: %v", task.ID, err)
		}
		return failErr
	}

	// Sync task to Observatory for trace linking (M-TASK-HIERARCHY)
	var obsContext *ObservatoryContext
	if d.observatorySync != nil {
		// Sync task entity
		if err := d.observatorySync.SyncTask(taskCtx, task); err != nil {
			d.logger.Printf("Warning: Failed to sync task to Observatory: %v", err)
		}

		// Create agent assignment and get ID for context propagation
		assignmentID, err := d.observatorySync.SyncAgentAssignment(taskCtx, task.ID, targetAgent, "claude")
		if err != nil {
			d.logger.Printf("Warning: Failed to sync agent assignment: %v", err)
		}

		// Get workspace ID from sync cache
		workspaceID := d.observatorySync.GetWorkspaceID(task.Workspace)

		obsContext = &ObservatoryContext{
			TaskID:       task.ID,
			AgentID:      targetAgent,
			AssignmentID: assignmentID,
			WorkspaceID:  workspaceID,
		}
		d.logger.Printf("Observatory context: task=%s agent=%s assignment=%s workspace=%s",
			task.ID, targetAgent, assignmentID, workspaceID)
	}

	opts := &ExecuteOptions{
		Timeout:            10 * time.Minute, // 10 minute timeout per task
		Workspace:          workspacePath,    // Worktree path for AI agents, direct workspace for script agents
		ObservatoryContext: obsContext,
		AgentConfig:        agentConfig, // For system prompt construction (v0.8.0+)
	}

	// Pass per-agent model override (v0.8.0+)
	// If agent config specifies a model, use it instead of the executor default
	if agentConfig != nil && agentConfig.Model != "" {
		opts.Model = agentConfig.Model
		d.logger.Printf("Task %s using agent-configured model: %s", task.ID, agentConfig.Model)
	}

	// Pass InvokeConfig for script execution (v0.6.4+)
	// This allows the TaskExecutor to route to ScriptProvider
	if agentConfig != nil && agentConfig.Invoke != nil {
		opts.InvokeConfig = agentConfig.Invoke
		if agentConfig.Invoke.Type == "script" {
			d.logger.Printf("Task %s will use script execution: %s", task.ID, agentConfig.Invoke.Command)
		}
	}

	// Create streaming event handler if broadcaster is available
	var eventHandler *CoordinatorEventHandler
	if d.eventBroadcaster != nil {
		eventHandler = NewCoordinatorEventHandler(task.ID, task.ThreadID, d.eventBroadcaster)
		opts.EventHandler = eventHandler

		// Set task context for event enrichment (workspace, directive, agent info)
		eventHandler.SetTaskContext(&TaskEventContext{
			Workspace:  task.Workspace,
			Directive:  task.Content,
			AgentID:    task.AgentID,
			SourceType: "coordinator",
		})

		// Set up event storage for historical replay
		if sqliteStore, ok := d.taskStore.(*SQLiteStore); ok {
			eventHandler.SetEventStorer(func(record *TaskEventRecord) error {
				return sqliteStore.StoreTaskEvent(taskCtx, record)
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
	resourceTracker.Start(taskCtx)

	// Execute the task
	result, err := d.executor.ExecuteWithRetry(taskCtx, analyzed, opts, 2)

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
		_ = d.taskStore.UpdateTaskMetrics(taskCtx, task.ID, finalMetrics.PeakCPU, finalMetrics.PeakMemory)
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

	// Record resource metrics in span (peak CPU/memory)
	if finalMetrics != nil {
		span.SetAttributes(
			attribute.Float64("task.peak_cpu_percent", finalMetrics.PeakCPU),
			attribute.Float64("task.peak_memory_mb", finalMetrics.PeakMemory),
			attribute.Float64("task.cpu_percent", finalMetrics.CPUPercent),
			attribute.Float64("task.memory_mb", finalMetrics.MemoryMB),
			attribute.Int("task.duration_sec", finalMetrics.DurationSec),
		)
	}

	// Update task status based on result
	if result.Success {
		// Check if this agent skips approval (e.g., script agents)
		skipApproval := agentConfig != nil && agentConfig.SkipApproval

		// PRESERVE WORKTREE - mark as pending approval, not completed
		// Human must approve/reject before worktree is cleaned up
		worktreePath := ""
		worktreeBranch := ""
		baseBranch := ""
		baseCommit := ""
		if worktree != nil {
			worktreePath = worktree.Path
			worktreeBranch = worktree.Branch
			baseBranch = worktree.BaseBranch
			baseCommit = worktree.BaseCommit

			// AUTO-COMMIT: Deterministically commit any uncommitted changes
			// Agents may forget to commit, so we do it automatically before approval
			if err := autoCommitWorktree(worktreePath, task.Title, d.logger); err != nil {
				d.logger.Printf("Warning: Auto-commit failed for task %s: %v", task.ID, err)
				// Continue anyway - some tasks may not produce file changes
			}
		}

		// Update task with worktree and agent info (needed for git diff artifact discovery)
		task.WorktreePath = worktreePath
		task.AgentID = targetAgent

		// Sync thread target_agent to match task agent_id (keeps hierarchy API in sync)
		if task.ThreadID != "" && d.msgStore != nil {
			if err := d.msgStore.SetThreadTargetAgent(task.ThreadID, targetAgent); err != nil {
				d.logger.Printf("Warning: Failed to sync thread target_agent: %v", err)
			}
		}

		if skipApproval {
			// Script agents or agents with skip_approval: mark as completed directly
			if err := d.taskStore.MarkTaskCompleted(taskCtx, task.ID, result); err != nil {
				d.logger.Printf("Warning: Failed to mark task completed: %v", err)
			}
			d.logger.Printf("Task %s completed (skip_approval=true, cost: $%.4f, tokens: %d)",
				task.ID, result.Cost, result.TokensUsed)
		} else {
			// Normal agents: mark as pending approval
			if err := d.taskStore.MarkTaskPendingApproval(taskCtx, task.ID, worktreePath, worktreeBranch, baseBranch, baseCommit, result); err != nil {
				d.logger.Printf("Warning: Failed to mark task pending approval: %v", err)
			}

			// M-COORD-GITHUB-AUTO-ROUTING: Process stage completion for GitHub-linked tasks
			// This posts the summary to GitHub and adds the appropriate approval label
			if err := d.ProcessStageCompletion(taskCtx, task, result); err != nil {
				d.logger.Printf("Warning: Failed to process stage completion: %v", err)
			}

			// Create approval request record for the CLI/dashboard to show
			approvalID := fmt.Sprintf("apr-%s", task.ID[5:]) // apr-<hash> from task-<hash>

			// Check if handoffs are pending for this agent (will be embedded in approval)
			var contextJSON string
			var handoffTargets []string
			if d.agentRegistry != nil && task.AgentID != "" {
				agent := d.agentRegistry.GetAgentByID(task.AgentID)
				if agent != nil && len(agent.TriggerOnComplete) > 0 && !agent.AutoApproveHandoffs {
					handoffTargets = agent.TriggerOnComplete
					// Build context with embedded handoff data
					handoffContext := map[string]interface{}{
						"handoff_targets": handoffTargets,
						"session_id":      result.SessionID,
						"source_agent":    task.AgentID,
					}
					if contextBytes, err := json.Marshal(handoffContext); err == nil {
						contextJSON = string(contextBytes)
					}
					d.logger.Printf("Embedding handoff to %v in merge approval for task %s", handoffTargets, task.ID)
				}
			}

			// Determine approval type based on whether handoff is embedded
			approvalType := string(ApprovalTypeMerge)
			description := fmt.Sprintf("Agent completed work on: %s", task.Title)
			if len(handoffTargets) > 0 {
				approvalType = "merge_handoff" // Combined type for UI clarity
				description = fmt.Sprintf("Agent completed work on: %s (will handoff to: %v)", task.Title, handoffTargets)
			}

			approvalReq := &ApprovalRequestRecord{
				ID:          approvalID,
				TaskID:      task.ID,
				Type:        approvalType,
				Description: description,
				ContextJSON: contextJSON,
				Status:      "pending",
				CreatedAt:   time.Now(),
			}
			if err := d.taskStore.CreateApprovalRequest(taskCtx, approvalReq); err != nil {
				d.logger.Printf("Warning: Failed to create approval request: %v", err)
			}

			// M-DASHBOARD-APPROVAL-INTEGRATION: Create inbox message for Event Queue visibility
			// This allows approvals to show up in the dashboard's event queue with pulsing animation
			if d.msgStore != nil {
				approvalMsg := &messaging.InboxMessage{
					FromAgent:     "coordinator",
					ToInbox:       "approvals",
					MessageType:   "approval_request",
					Title:         fmt.Sprintf("Approval Required: %s", task.Title),
					Payload:       fmt.Sprintf("**Approval Required**\n\nTask: %s\nAgent: %s\n\nReview and approve via dashboard or CLI:\n`ailang coordinator approve %s`", task.Title, task.AgentID, approvalID),
					CorrelationID: task.ID,
					Status:        messaging.InboxStatusUnread,
				}
				if msgErr := d.msgStore.InsertInboxMessage(approvalMsg); msgErr != nil {
					d.logger.Printf("Warning: Failed to create approval inbox message: %v", msgErr)
				}
			}

			d.logger.Printf("Task %s awaiting approval (cost: $%.4f, tokens: %d, worktree: %s)",
				task.ID, result.Cost, result.TokensUsed, worktreePath)
		}

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
		if err := d.taskStore.MarkTaskFailed(taskCtx, task.ID, fmt.Errorf("%s", result.Error)); err != nil {
			d.logger.Printf("Warning: Failed to mark task failed: %v", err)
		}
		d.logger.Printf("Task %s failed: %s", task.ID, result.Error)
		span.SetStatus(codes.Error, result.Error)
	}

	// Set span status for successful tasks
	if result.Success {
		span.SetStatus(codes.Ok, "task completed successfully")
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

// cleanupWorktreesForTerminalTasks removes worktrees for tasks in terminal states.
// This is needed because RecoverStaleTasks marks tasks as cancelled but doesn't
// clean up their worktrees, leading to "max worktrees limit reached" errors.
// It handles two cases:
// 1. Tasks with WorktreePath set in database - clean up by task ID
// 2. Orphaned worktree directories on disk for terminal tasks (WorktreePath may be empty)
func (d *Daemon) cleanupWorktreesForTerminalTasks() {
	if d.taskStore == nil {
		return
	}

	terminalStatuses := []TaskStatus{
		TaskStatusCancelled,
		TaskStatusCompleted,
		TaskStatusFailed,
		TaskStatusRejected,
	}

	cleanedTotal := 0

	// First pass: clean up by WorktreePath in database
	for _, status := range terminalStatuses {
		tasks, err := d.taskStore.ListTasks(d.ctx, &TaskFilter{Status: []TaskStatus{status}})
		if err != nil {
			continue
		}

		for _, task := range tasks {
			if task.WorktreePath == "" {
				continue
			}

			agentID := task.AgentID
			if agentID == "" {
				agentID = "coordinator"
			}

			if wm, ok := d.worktreeManagers[agentID]; ok {
				if err := wm.RemoveWorktree(task.ID); err == nil {
					cleanedTotal++
					d.logger.Printf("Cleaned up stale worktree for %s task %s", status, task.ID)
				}
			}
		}
	}

	// Second pass: scan worktree directories for orphans
	// This handles cases where WorktreePath was never set (task crashed early)
	for agentID, wm := range d.worktreeManagers {
		// Get base directory for this agent's worktrees
		baseDir := filepath.Join(d.config.StateDir, "worktrees", agentID)
		entries, err := os.ReadDir(baseDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}

			taskID := entry.Name()

			// Check if this task exists and is in a terminal state
			task, err := d.taskStore.GetTask(d.ctx, taskID)
			if err != nil {
				// Task doesn't exist in DB - this is truly orphaned, clean it up
				if err := wm.RemoveWorktree(taskID); err == nil {
					cleanedTotal++
					d.logger.Printf("Cleaned up orphaned worktree (no task record) %s/%s", agentID, taskID)
				}
				continue
			}

			// Check if task is in terminal state
			isTerminal := false
			for _, ts := range terminalStatuses {
				if task.Status == ts {
					isTerminal = true
					break
				}
			}

			if isTerminal {
				if err := wm.RemoveWorktree(taskID); err == nil {
					cleanedTotal++
					d.logger.Printf("Cleaned up orphaned worktree for %s task %s/%s", task.Status, agentID, taskID)
				}
			}
		}
	}

	if cleanedTotal > 0 {
		d.logger.Printf("Cleaned up %d stale worktree(s) for terminal tasks", cleanedTotal)
	}
}

// initHTTPBroadcaster initializes the HTTP broadcaster for streaming events to dashboard
func (d *Daemon) initHTTPBroadcaster() error {
	serverURL := DefaultServerURL()

	broadcaster := NewHTTPBroadcaster(serverURL, d.logger)

	// Check if server is reachable
	if !broadcaster.CheckServerAvailable() {
		return fmt.Errorf("collaboration hub server not available at %s", serverURL)
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

// checkBudgetBeforeExecution checks if the task can proceed within budget limits.
// Returns (blocked, error) where blocked=true means task should wait for approval.
func (d *Daemon) checkBudgetBeforeExecution(ctx context.Context, task *TaskRecord, agentConfig *AgentConfig) (bool, error) {
	// Load budget configuration
	budgetsCfg, err := LoadBudgetsConfig()
	if err != nil || budgetsCfg == nil {
		// No budget config = no enforcement
		d.logger.Printf("[DEBUG] No budget config found, skipping budget check")
		return false, nil
	}

	// Determine provider
	provider := "claude" // default
	if agentConfig != nil && agentConfig.Provider != "" {
		provider = agentConfig.Provider
	} else if d.coordConfig != nil && d.coordConfig.DefaultProvider != "" {
		provider = d.coordConfig.DefaultProvider
	}

	// Get provider-specific limits
	var dailyLimit, taskMaxLimit float64
	var hardLimit bool

	if budgetsCfg.Providers != nil {
		if providerCfg, ok := budgetsCfg.Providers[provider]; ok && providerCfg != nil {
			dailyLimit = providerCfg.DailyBudget
			taskMaxLimit = providerCfg.TaskMaxCost
			hardLimit = providerCfg.HardLimit
		}
	}

	// Fall back to global limits
	if dailyLimit == 0 && budgetsCfg.Global != nil {
		dailyLimit = budgetsCfg.Global.DailyBudget
	}
	if taskMaxLimit == 0 && budgetsCfg.Global != nil {
		taskMaxLimit = budgetsCfg.Global.TaskMaxCost
	}

	// If no limits configured, allow the task
	if dailyLimit == 0 && taskMaxLimit == 0 {
		return false, nil
	}

	// Get current spend by provider
	costByProvider, err := d.taskStore.GetCostByProvider()
	if err != nil {
		d.logger.Printf("Warning: Failed to get cost by provider: %v", err)
		return false, nil // Allow task if we can't check budget
	}

	currentSpend := costByProvider[provider]
	d.logger.Printf("[BUDGET] Provider %s: current spend $%.2f, daily limit $%.2f, hard=%v",
		provider, currentSpend, dailyLimit, hardLimit)

	// Check if already over budget
	if dailyLimit > 0 && currentSpend >= dailyLimit {
		d.logger.Printf("[BUDGET] Provider %s: daily budget exceeded ($%.2f >= $%.2f)",
			provider, currentSpend, dailyLimit)

		if hardLimit {
			// Create cost approval request
			return d.createBudgetApproval(ctx, task, provider, currentSpend, dailyLimit)
		}
		// Soft limit - log warning and continue
		d.logger.Printf("[BUDGET] WARNING: Provider %s over budget but soft limit, continuing", provider)
	}

	return false, nil
}

// createBudgetApproval creates an ApprovalTypeCost approval request for budget-blocked tasks.
func (d *Daemon) createBudgetApproval(ctx context.Context, task *TaskRecord, provider string, currentSpend, limit float64) (bool, error) {
	d.logger.Printf("[BUDGET] Creating cost approval for task %s (provider %s)", task.ID, provider)

	// Mark task as pending approval
	if err := d.taskStore.MarkTaskPendingApproval(ctx, task.ID, "", "", "", "", nil); err != nil {
		d.logger.Printf("Warning: Failed to mark task as pending approval: %v", err)
	}

	// Create approval request with type "cost"
	approvalReq := &ApprovalRequestRecord{
		ID:        fmt.Sprintf("apr-cost-%s", task.ID),
		TaskID:    task.ID,
		Type:      string(ApprovalTypeCost),
		Status:    "pending",
		CreatedAt: time.Now(),
		ContextJSON: fmt.Sprintf(`{
			"provider": %q,
			"current_spend": %.2f,
			"daily_limit": %.2f,
			"reason": "Daily budget limit exceeded for provider %s"
		}`, provider, currentSpend, limit, provider),
	}

	if err := d.taskStore.CreateApprovalRequest(ctx, approvalReq); err != nil {
		d.logger.Printf("Warning: Failed to create cost approval request: %v", err)
		return false, err
	}

	// Post status update
	d.postTaskStatus(task, "budget_blocked",
		fmt.Sprintf("Task blocked: %s budget limit exceeded ($%.2f/$%.2f). Waiting for approval.",
			provider, currentSpend, limit))

	// Broadcast event
	if d.eventBroadcaster != nil {
		d.eventBroadcaster(&websocket.TaskStreamEvent{
			TaskID:     task.ID,
			StreamType: websocket.TaskStreamStatus,
			Status:     "budget_blocked",
			Text: fmt.Sprintf("Budget limit exceeded for %s ($%.2f/$%.2f)",
				provider, currentSpend, limit),
		})
	}

	return true, nil // Task blocked
}

// autoCommitWorktree commits any uncommitted changes in the worktree.
// This ensures agent work is captured even if the agent forgot to commit.
func autoCommitWorktree(worktreePath, taskTitle string, logger *log.Logger) error {
	// Check for uncommitted changes (including untracked files)
	statusCmd := exec.Command("git", "status", "--porcelain")
	statusCmd.Dir = worktreePath
	output, err := statusCmd.Output()
	if err != nil {
		return fmt.Errorf("git status failed: %w", err)
	}

	// No changes to commit
	if strings.TrimSpace(string(output)) == "" {
		logger.Printf("No uncommitted changes in worktree: %s", worktreePath)
		return nil
	}

	logger.Printf("Auto-committing changes in worktree: %s", worktreePath)
	logger.Printf("Changes:\n%s", string(output))

	// Add all changes (including untracked files)
	addCmd := exec.Command("git", "add", "-A")
	addCmd.Dir = worktreePath
	if err := addCmd.Run(); err != nil {
		return fmt.Errorf("git add failed: %w", err)
	}

	// Create commit message
	commitMsg := fmt.Sprintf("Auto-commit: %s\n\nCommitted by AILANG coordinator after agent completion.\n\n🤖 Generated with Claude Code", taskTitle)

	// Commit the changes
	commitCmd := exec.Command("git", "commit", "-m", commitMsg)
	commitCmd.Dir = worktreePath
	commitOutput, err := commitCmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git commit failed: %w\nOutput: %s", err, string(commitOutput))
	}

	logger.Printf("Auto-commit successful: %s", strings.TrimSpace(string(commitOutput)))
	return nil
}
