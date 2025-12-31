package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/sunholo/ailang/internal/messaging"
	"github.com/sunholo/ailang/internal/websocket"
)

// Config holds daemon configuration
type Config struct {
	PollInterval time.Duration // How often to check for new messages
	MaxWorktrees int           // Maximum concurrent worktrees
	LogFile      string        // Path to log file
	PIDFile      string        // Path to PID file
	StateDir     string        // Directory for state files
}

// DefaultConfig returns sensible defaults
func DefaultConfig() *Config {
	homeDir, _ := os.UserHomeDir()
	stateDir := filepath.Join(homeDir, ".ailang", "state")
	logsDir := filepath.Join(homeDir, ".ailang", "logs")

	return &Config{
		PollInterval: 30 * time.Second,
		MaxWorktrees: 3,
		LogFile:      filepath.Join(logsDir, "coordinator.log"),
		PIDFile:      filepath.Join(stateDir, "coordinator.pid"),
		StateDir:     stateDir,
	}
}

// Status represents the daemon's current state
type Status struct {
	Running   bool      `json:"running"`
	PID       int       `json:"pid,omitempty"`
	StartedAt time.Time `json:"started_at,omitempty"`
	Uptime    string    `json:"uptime,omitempty"`
	TasksRun  int       `json:"tasks_run"`
	// Extended stats from database
	PendingTasks int     `json:"pending_tasks,omitempty"`
	RunningTasks int     `json:"running_tasks,omitempty"`
	FailedTasks  int     `json:"failed_tasks,omitempty"`
	TotalCost    float64 `json:"total_cost,omitempty"`
	TotalTokens  int     `json:"total_tokens,omitempty"`
}

// Daemon is the coordinator daemon
type Daemon struct {
	config    *Config
	logger    *log.Logger
	logFile   *os.File // Underlying log file handle (for cleanup on Windows)
	ctx       context.Context
	cancel    context.CancelFunc
	startedAt time.Time
	tasksRun  int

	// Task processing components
	msgStore        *messaging.Store
	msgAdapter      *InboxMessageAdapter        // Legacy: single adapter (for backwards compat)
	inboxAdapters   map[string]*InboxMessageAdapter // Key: inbox name
	worktreeManagers map[string]*WorktreeManager     // Key: agent ID
	analyzer        *TaskAnalyzer
	worktreeMgr     *WorktreeManager              // Legacy: default worktree manager
	taskStore       Store
	executor        *TaskExecutor

	// Agent configuration
	agentRegistry *AgentRegistry
	coordConfig   *CoordinatorConfig

	// Approval workflow
	approvalCheckpoint *ApprovalCheckpoint

	// Event broadcasting for real-time updates
	eventBroadcaster EventBroadcaster

	// Resource tracking for running tasks
	resourceRegistry *ResourceTrackerRegistry

	// History tracking
	instanceID string
}

// NewDaemon creates a new daemon instance
func NewDaemon(config *Config) (*Daemon, error) {
	if config == nil {
		config = DefaultConfig()
	}

	// Ensure directories exist
	if err := os.MkdirAll(filepath.Dir(config.LogFile), 0755); err != nil {
		return nil, fmt.Errorf("failed to create log directory: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(config.PIDFile), 0755); err != nil {
		return nil, fmt.Errorf("failed to create state directory: %w", err)
	}

	// Open log file
	logFile, err := os.OpenFile(config.LogFile, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to open log file: %w", err)
	}

	logger := log.New(logFile, "[coordinator] ", log.LstdFlags|log.Lshortfile)

	ctx, cancel := context.WithCancel(context.Background())

	return &Daemon{
		config:           config,
		logger:           logger,
		logFile:          logFile,
		ctx:              ctx,
		cancel:           cancel,
		resourceRegistry: NewResourceTrackerRegistry(),
	}, nil
}

// Start starts the daemon
func (d *Daemon) Start() error {
	// Check if already running
	if status, _ := d.Status(); status.Running {
		return fmt.Errorf("daemon already running with PID %d", status.PID)
	}

	// Write PID file
	if err := d.writePIDFile(); err != nil {
		return fmt.Errorf("failed to write PID file: %w", err)
	}

	d.startedAt = time.Now()
	d.logger.Printf("Daemon starting (PID: %d)", os.Getpid())

	// Set up signal handling
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		sig := <-sigChan
		d.logger.Printf("Received signal: %v, shutting down...", sig)
		d.cancel()
	}()

	// Run main loop
	return d.Run()
}

// Run is the main daemon loop
func (d *Daemon) Run() error {
	ticker := time.NewTicker(d.config.PollInterval)
	defer ticker.Stop()

	// Initialize task processing components
	if err := d.initTaskProcessing(); err != nil {
		d.logger.Printf("Warning: Failed to initialize task processing: %v", err)
		d.logger.Println("Daemon running in standby mode (no message processing)")
	} else {
		d.logger.Println("Daemon running, polling for tasks...")
	}

	// Initialize HTTP broadcaster for real-time streaming to dashboard
	if err := d.initHTTPBroadcaster(); err != nil {
		d.logger.Printf("Warning: HTTP broadcaster not available: %v", err)
		d.logger.Println("Task streaming to dashboard disabled - events will be logged only")
	} else {
		d.logger.Println("HTTP broadcaster connected to Collaboration Hub")
	}

	// Register as an agent in the collaboration hub
	if err := d.registerAgent(); err != nil {
		d.logger.Printf("Warning: Failed to register agent: %v", err)
	}

	// Start GitHub sync if enabled
	if d.coordConfig != nil && d.coordConfig.GitHubSync != nil && d.coordConfig.GitHubSync.Enabled {
		go d.runGitHubSync()
	}

	for {
		select {
		case <-d.ctx.Done():
			d.logger.Println("Daemon shutting down gracefully")
			d.cleanup()
			return nil
		case <-ticker.C:
			d.logger.Println("Checking for new tasks...")
			if d.msgAdapter != nil {
				if err := d.pollAndProcessTasks(); err != nil {
					d.logger.Printf("Error processing tasks: %v", err)
				}
			}

			// Execute pending tasks
			if d.executor != nil {
				if err := d.executeTaskQueue(); err != nil {
					d.logger.Printf("Error executing tasks: %v", err)
				}
			}
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

// runGitHubSync runs periodic GitHub issue import in the background.
// This imports GitHub issues as messages, which then trigger task creation.
func (d *Daemon) runGitHubSync() {
	cfg := d.coordConfig.GitHubSync
	interval := time.Duration(cfg.IntervalSecs) * time.Second
	if interval < time.Minute {
		interval = 5 * time.Minute // Minimum 5 minutes to avoid rate limits
	}

	d.logger.Printf("GitHub sync started (interval: %v, labels: %v, target: %s)",
		interval, cfg.WatchLabels, cfg.TargetInbox)

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	// Run immediately on startup
	d.syncGitHubIssues()

	for {
		select {
		case <-d.ctx.Done():
			d.logger.Println("GitHub sync stopping")
			return
		case <-ticker.C:
			d.syncGitHubIssues()
		}
	}
}

// syncGitHubIssues imports GitHub issues as messages using the ailang CLI.
func (d *Daemon) syncGitHubIssues() {
	cfg := d.coordConfig.GitHubSync
	d.logger.Println("Running GitHub issue sync...")

	// Build the command: ailang messages import-github [--labels label1,label2]
	args := []string{"messages", "import-github"}
	if len(cfg.WatchLabels) > 0 {
		labels := ""
		for i, label := range cfg.WatchLabels {
			if i > 0 {
				labels += ","
			}
			labels += label
		}
		args = append(args, "--labels", labels)
	}

	// Use exec to run the ailang command
	cmd := exec.CommandContext(d.ctx, "ailang", args...)
	output, err := cmd.CombinedOutput()

	if err != nil {
		d.logger.Printf("GitHub sync error: %v\nOutput: %s", err, string(output))
		return
	}

	// Log result (trimmed to avoid log spam)
	result := string(output)
	if len(result) > 200 {
		result = result[:200] + "..."
	}
	if result != "" {
		d.logger.Printf("GitHub sync: %s", result)
	} else {
		d.logger.Println("GitHub sync complete (no new issues)")
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
			ID:        taskID,
			MessageID: msg.ID,
			Title:     msg.Title,
			Content:   msg.Content,
			Type:      analyzed.Type,
			Kind:      kind,
			Priority:  CalculatePriority(analyzed),
			Status:    TaskStatusPending,
			Workspace: workspace,
			CreatedAt: msg.CreatedAt,
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
			msg.Title,           // title
			"ailang_instance",   // createdByType (constraint: 'human' or 'ailang_instance')
			"coordinator",       // createdByID
			targetAgent,         // targetAgent - the agent that will handle this task
			workspace,           // workspace - source project/agent
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

		d.logger.Printf("Created task %s (type: %s, priority: %d, thread: %s, agent: %s) from message %s",
			task.ID, task.Type, task.Priority, task.ThreadID, agentID, msg.ID)

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

// executeTask runs a single task through the executor
func (d *Daemon) executeTask(task *TaskRecord) error {
	d.logger.Printf("Starting execution of task %s (type: %s)", task.ID, task.Type)

	// Mark task as running
	if err := d.taskStore.MarkTaskRunning(d.ctx, task.ID, "", ""); err != nil {
		return fmt.Errorf("failed to mark task as running: %w", err)
	}

	// Post "starting" message to thread
	d.postTaskStatus(task, "running", "Starting task execution...")

	// Create analyzed task for executor
	analyzed := &AnalyzedTask{
		Task: &Task{
			ID:        task.ID,
			Title:     task.Title,
			Content:   task.Content,
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
			d.logger.Printf("Warning: Failed to create worktree for task %s (agent: %s): %v", task.ID, targetAgent, wtErr)
			// Continue without worktree - will use current directory
		}
	}

	opts := &ExecuteOptions{
		Timeout: 10 * time.Minute, // 10 minute timeout per task
	}
	if worktree != nil {
		opts.Workspace = worktree.Path
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
		// Cleanup worktree on error (no work to preserve)
		if worktree != nil && d.worktreeMgr != nil {
			if rmErr := d.worktreeMgr.RemoveWorktree(task.ID); rmErr != nil {
				d.logger.Printf("Warning: Failed to remove worktree: %v", rmErr)
			}
		}
		return fmt.Errorf("executor error: %w", err)
	}

	// Update task status based on result
	if result.Success {
		// PRESERVE WORKTREE - mark as pending approval, not completed
		// Human must approve/reject before worktree is cleaned up
		worktreePath := ""
		if worktree != nil {
			worktreePath = worktree.Path
		}
		if err := d.taskStore.MarkTaskPendingApproval(d.ctx, task.ID, worktreePath, result); err != nil {
			d.logger.Printf("Warning: Failed to mark task pending approval: %v", err)
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

// handleAgentHandoffs checks if the completed task should trigger handoffs to other agents.
// This implements the agent-to-agent messaging with optional approval gates.
func (d *Daemon) handleAgentHandoffs(task *TaskRecord, result *ExecuteResult) error {
	if d.agentRegistry == nil {
		return nil // No agent registry configured
	}

	// Determine which agent handled this task
	sourceAgentID := "coordinator" // default
	if task.ThreadID != "" && d.msgStore != nil {
		if thread, err := d.msgStore.GetThread(task.ThreadID); err == nil && thread != nil && thread.TargetAgent != "" {
			sourceAgentID = thread.TargetAgent
		}
	}

	// Look up the source agent's configuration
	sourceAgent := d.agentRegistry.GetAgentByID(sourceAgentID)
	if sourceAgent == nil {
		d.logger.Printf("Agent %s not found in registry, skipping handoffs", sourceAgentID)
		return nil
	}

	// Check if this agent has any trigger_on_complete targets
	if len(sourceAgent.TriggerOnComplete) == 0 {
		return nil // No handoffs configured
	}

	d.logger.Printf("Task %s completed by agent %s, checking handoffs to: %v",
		task.ID, sourceAgentID, sourceAgent.TriggerOnComplete)

	// Process each handoff target
	for _, targetAgentID := range sourceAgent.TriggerOnComplete {
		targetAgent := d.agentRegistry.GetAgentByID(targetAgentID)
		if targetAgent == nil {
			d.logger.Printf("Warning: Handoff target agent %s not found in registry", targetAgentID)
			continue
		}

		// Get session ID from the result for continuity
		sessionID := ""
		if result != nil {
			sessionID = result.SessionID
		}

		// Build handoff message
		handoffMessage := fmt.Sprintf("**Handoff from %s**\n\n"+
			"Task: %s\n"+
			"Original Request: %s\n\n"+
			"Result: %s\n\n"+
			"Please continue this work.",
			sourceAgentID, task.ID, task.Content, result.Output)

		if sourceAgent.AutoApproveHandoffs {
			// Auto-approve: send message directly to target agent's inbox
			if err := d.sendHandoffMessage(targetAgent, task, handoffMessage, sessionID); err != nil {
				d.logger.Printf("Warning: Failed to send handoff to %s: %v", targetAgentID, err)
				continue
			}
			d.logger.Printf("Auto-approved handoff from %s to %s for task %s",
				sourceAgentID, targetAgentID, task.ID)
		} else {
			// Require approval: create approval request
			if err := d.requestHandoffApproval(sourceAgent, targetAgent, task, result, handoffMessage, sessionID); err != nil {
				d.logger.Printf("Warning: Failed to create handoff approval request: %v", err)
				continue
			}
			d.logger.Printf("Created handoff approval request from %s to %s for task %s",
				sourceAgentID, targetAgentID, task.ID)
		}
	}

	return nil
}

// sendHandoffMessage sends a message to the target agent's inbox
func (d *Daemon) sendHandoffMessage(targetAgent *AgentConfig, task *TaskRecord, message, sessionID string) error {
	if d.msgStore == nil {
		return fmt.Errorf("message store not available")
	}

	// Include session ID in metadata for continuity
	metadata := ""
	if sessionID != "" {
		metadataMap := map[string]interface{}{
			"session_id":     sessionID,
			"handoff_source": task.ID,
		}
		if data, err := json.Marshal(metadataMap); err == nil {
			metadata = string(data)
		}
	}

	// Create message in the target agent's inbox
	// We create a new thread for the handoff
	_, err := d.msgStore.CreateMessage(
		"",                                       // New thread (empty ThreadID)
		"ailang_instance", "coordinator",         // from
		targetAgent.Inbox, targetAgent.ID,        // to (inbox and agent)
		"handoff",                                // kind
		message,
		metadata,
	)

	return err
}

// requestHandoffApproval creates an approval request for a handoff
func (d *Daemon) requestHandoffApproval(
	sourceAgent, targetAgent *AgentConfig,
	task *TaskRecord,
	result *ExecuteResult,
	message, sessionID string,
) error {
	if d.approvalCheckpoint == nil {
		return fmt.Errorf("approval checkpoint not available")
	}

	// Create the approval request (non-blocking - we don't wait for it)
	request := &ApprovalRequest{
		ID:            fmt.Sprintf("handoff-%s-%s-%d", sourceAgent.ID, targetAgent.ID, time.Now().UnixNano()),
		TaskID:        task.ID,
		ThreadID:      task.ThreadID,
		Type:          ApprovalTypeHandoff,
		Title:         fmt.Sprintf("Handoff: %s → %s", sourceAgent.Label, targetAgent.Label),
		Description:   message,
		SourceAgentID: sourceAgent.ID,
		TargetAgentID: targetAgent.ID,
		SessionID:     sessionID,
		Timeout:       24 * time.Hour, // Allow 24 hours for human approval
		AutoReject:    true,           // Reject on timeout (don't auto-approve handoffs)
	}

	// Store the request for dashboard visibility
	// Note: We don't block waiting for approval here - the approval is handled
	// asynchronously when the human approves via CLI/dashboard
	d.approvalCheckpoint.mu.Lock()
	d.approvalCheckpoint.requests[request.ID] = request
	d.approvalCheckpoint.mu.Unlock()

	// Broadcast the approval request event for dashboard
	if d.eventBroadcaster != nil {
		d.eventBroadcaster(&websocket.TaskStreamEvent{
			TaskID:     task.ID,
			ThreadID:   task.ThreadID,
			StreamType: websocket.TaskStreamStatus,
			Status:     "approval_requested",
			Text:       fmt.Sprintf("Handoff approval needed: %s → %s", sourceAgent.ID, targetAgent.ID),
		})
	}

	// Also post to the task's thread so it shows up in the message UI
	if task.ThreadID != "" && d.msgStore != nil {
		content := fmt.Sprintf("**Approval Required: Handoff to %s**\n\n%s\n\n"+
			"Use `ailang coordinator approve %s` to approve or `reject` to reject.",
			targetAgent.Label, message, request.ID)

		_, _ = d.msgStore.CreateMessage(
			task.ThreadID,
			"ailang_instance", "coordinator",
			"human", "user",
			"approval_request",
			content,
			"",
		)
	}

	return nil
}

// HandleApproval processes an approved task - merges worktree changes to main branch.
func (d *Daemon) HandleApproval(ctx context.Context, taskID, approvedBy string) error {
	// Get the task
	task, err := d.taskStore.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// Verify task is pending approval
	if task.Status != TaskStatusPendingApproval {
		return fmt.Errorf("task %s is not pending approval (status: %s)", taskID, task.Status)
	}

	// Get worktree path
	if task.WorktreePath == "" {
		return fmt.Errorf("task %s has no worktree path", taskID)
	}

	d.logger.Printf("Processing approval for task %s (worktree: %s)", taskID, task.WorktreePath)

	// Attempt to merge worktree changes
	mergeResult, err := MergeWorktree(ctx, task.WorktreePath, "main")
	if err != nil {
		return fmt.Errorf("merge failed: %w", err)
	}

	// Handle merge conflicts
	if !mergeResult.Success {
		if len(mergeResult.ConflictFiles) > 0 {
			// Mark task as still pending but with conflict info
			d.logger.Printf("Merge conflicts in task %s: %v", taskID, mergeResult.ConflictFiles)

			// Notify via message
			if task.ThreadID != "" && d.msgStore != nil {
				content := fmt.Sprintf("**Merge Conflicts Detected**\n\n"+
					"The following files have conflicts:\n%s\n\n"+
					"Please resolve conflicts manually in the worktree at:\n`%s`\n\n"+
					"Then retry the approval.",
					strings.Join(mergeResult.ConflictFiles, "\n"), task.WorktreePath)

				_, _ = d.msgStore.CreateMessage(
					task.ThreadID,
					"ailang_instance", "coordinator",
					"human", "user",
					"merge_conflict",
					content,
					"",
				)
			}

			return fmt.Errorf("merge conflicts: %v", mergeResult.ConflictFiles)
		}
		return fmt.Errorf("merge failed: %s", mergeResult.Error)
	}

	// Merge succeeded - update task status
	if err := d.taskStore.MarkTaskCompleted(ctx, taskID, &ExecuteResult{
		Success: true,
		Output:  fmt.Sprintf("Merged to main (commit: %s)", mergeResult.CommitHash),
	}); err != nil {
		d.logger.Printf("Warning: Failed to update task status: %v", err)
	}

	// Clean up worktree (changes are now in main)
	if d.worktreeMgr != nil {
		if rmErr := d.worktreeMgr.RemoveWorktree(taskID); rmErr != nil {
			d.logger.Printf("Warning: Failed to remove worktree: %v", rmErr)
		}
	}

	// Notify success
	if task.ThreadID != "" && d.msgStore != nil {
		content := fmt.Sprintf("**Changes Merged Successfully**\n\n"+
			"Commit: `%s`\n"+
			"Files: %s\n"+
			"Approved by: %s",
			mergeResult.CommitHash,
			strings.Join(mergeResult.MergedFiles, ", "),
			approvedBy)

		_, _ = d.msgStore.CreateMessage(
			task.ThreadID,
			"ailang_instance", "coordinator",
			"human", "user",
			"merge_complete",
			content,
			"",
		)
	}

	// Broadcast event
	if d.eventBroadcaster != nil {
		d.eventBroadcaster(&websocket.TaskStreamEvent{
			TaskID:     taskID,
			ThreadID:   task.ThreadID,
			StreamType: websocket.TaskStreamStatus,
			Status:     "merged",
			Text:       fmt.Sprintf("Changes merged to main (commit: %s)", mergeResult.CommitHash[:8]),
		})
	}

	d.logger.Printf("Task %s approved and merged by %s (commit: %s)",
		taskID, approvedBy, mergeResult.CommitHash)

	// Check for pending handoff approvals for this task
	if d.approvalCheckpoint != nil {
		if req := d.approvalCheckpoint.GetRequestByTask(taskID); req != nil {
			if req.Type == ApprovalTypeHandoff {
				// Auto-approve the handoff now that the work is merged
				if err := d.processHandoffApproval(ctx, req); err != nil {
					d.logger.Printf("Warning: Failed to process handoff: %v", err)
				}
			}
		}
	}

	return nil
}

// HandleRejection processes a rejected task - preserves worktree and marks task rejected.
func (d *Daemon) HandleRejection(ctx context.Context, taskID, rejectedBy, reason string) error {
	// Get the task
	task, err := d.taskStore.GetTask(ctx, taskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}
	if task == nil {
		return fmt.Errorf("task not found: %s", taskID)
	}

	// Verify task is pending approval
	if task.Status != TaskStatusPendingApproval {
		return fmt.Errorf("task %s is not pending approval (status: %s)", taskID, task.Status)
	}

	d.logger.Printf("Processing rejection for task %s (worktree preserved: %s)", taskID, task.WorktreePath)

	// Mark task as rejected - worktree is preserved for reference
	if err := d.taskStore.MarkTaskRejected(ctx, taskID); err != nil {
		return fmt.Errorf("failed to mark task rejected: %w", err)
	}

	// Notify via message
	if task.ThreadID != "" && d.msgStore != nil {
		content := fmt.Sprintf("**Task Rejected**\n\n"+
			"Rejected by: %s\n"+
			"Reason: %s\n\n"+
			"The worktree is preserved at:\n`%s`\n\n"+
			"You can review the changes or delete the worktree manually.",
			rejectedBy, reason, task.WorktreePath)

		_, _ = d.msgStore.CreateMessage(
			task.ThreadID,
			"ailang_instance", "coordinator",
			"human", "user",
			"task_rejected",
			content,
			"",
		)
	}

	// Broadcast event
	if d.eventBroadcaster != nil {
		d.eventBroadcaster(&websocket.TaskStreamEvent{
			TaskID:     taskID,
			ThreadID:   task.ThreadID,
			StreamType: websocket.TaskStreamStatus,
			Status:     "rejected",
			Text:       fmt.Sprintf("Task rejected by %s", rejectedBy),
		})
	}

	// Reject any pending handoff approvals for this task
	if d.approvalCheckpoint != nil {
		if req := d.approvalCheckpoint.GetRequestByTask(taskID); req != nil {
			_ = d.approvalCheckpoint.Reject(req.ID, rejectedBy)
		}
	}

	d.logger.Printf("Task %s rejected by %s (worktree preserved)", taskID, rejectedBy)
	return nil
}

// processHandoffApproval sends a handoff message after approval.
func (d *Daemon) processHandoffApproval(ctx context.Context, req *ApprovalRequest) error {
	if d.agentRegistry == nil || d.msgStore == nil {
		return fmt.Errorf("agent registry or message store not available")
	}

	targetAgent := d.agentRegistry.GetAgentByID(req.TargetAgentID)
	if targetAgent == nil {
		return fmt.Errorf("target agent %s not found", req.TargetAgentID)
	}

	// Get the task for context
	task, err := d.taskStore.GetTask(ctx, req.TaskID)
	if err != nil {
		return fmt.Errorf("failed to get task: %w", err)
	}

	// Create the handoff message
	message := fmt.Sprintf("**Approved Handoff from %s**\n\n%s",
		req.SourceAgentID, req.Description)

	metadata := ""
	if req.SessionID != "" {
		metadataMap := map[string]interface{}{
			"session_id":     req.SessionID,
			"handoff_source": req.TaskID,
		}
		if data, err := json.Marshal(metadataMap); err == nil {
			metadata = string(data)
		}
	}

	// Send to target agent's inbox
	_, err = d.msgStore.CreateMessage(
		"",                                // New thread
		"ailang_instance", "coordinator",  // from
		targetAgent.Inbox, targetAgent.ID, // to
		"handoff",
		message,
		metadata,
	)
	if err != nil {
		return fmt.Errorf("failed to send handoff message: %w", err)
	}

	d.logger.Printf("Handoff approved: %s → %s (task: %s)",
		req.SourceAgentID, req.TargetAgentID, task.ID)

	return nil
}

// Stop stops a running daemon
func (d *Daemon) Stop() error {
	status, err := d.Status()
	if err != nil {
		return err
	}

	if !status.Running {
		return fmt.Errorf("daemon is not running")
	}

	// Send SIGTERM to the process
	process, err := os.FindProcess(status.PID)
	if err != nil {
		return fmt.Errorf("failed to find process: %w", err)
	}

	if err := process.Signal(syscall.SIGTERM); err != nil {
		return fmt.Errorf("failed to send signal: %w", err)
	}

	// Wait for process to exit (with timeout)
	for i := 0; i < 10; i++ {
		time.Sleep(500 * time.Millisecond)
		if !d.isProcessRunning(status.PID) {
			return nil
		}
	}

	// Force kill if still running
	if err := process.Kill(); err != nil {
		return fmt.Errorf("failed to kill process: %w", err)
	}

	return nil
}

// Close releases resources held by the daemon.
// This must be called when done with the daemon to release file handles.
// On Windows, this is required before the log file can be deleted.
func (d *Daemon) Close() error {
	if d.logFile != nil {
		if err := d.logFile.Close(); err != nil {
			return fmt.Errorf("failed to close log file: %w", err)
		}
		d.logFile = nil
	}
	return nil
}

// Status returns the current daemon status
func (d *Daemon) Status() (*Status, error) {
	status := &Status{
		Running:  false,
		TasksRun: d.tasksRun,
	}

	// Read PID file
	pid, err := d.readPIDFile()
	if err != nil {
		// No PID file means not running
		return status, nil
	}

	// Check if process is running
	if d.isProcessRunning(pid) {
		status.Running = true
		status.PID = pid

		// Try to get start time from process (simplified - just use current time minus uptime estimate)
		if !d.startedAt.IsZero() {
			status.StartedAt = d.startedAt
			status.Uptime = time.Since(d.startedAt).Round(time.Second).String()
		}
	} else {
		// Stale PID file, clean up
		_ = os.Remove(d.config.PIDFile)
	}

	// Get actual task stats from database (in-memory counter resets on restart)
	store, err := NewSQLiteStore(filepath.Join(d.config.StateDir, "coordinator.db"))
	if err == nil {
		defer store.Close()
		stats, err := store.GetTaskStats(context.Background())
		if err == nil {
			status.TasksRun = stats.CompletedTasks
			status.PendingTasks = stats.PendingTasks
			status.RunningTasks = stats.RunningTasks
			status.FailedTasks = stats.FailedTasks
			status.TotalCost = stats.TotalCost
			status.TotalTokens = stats.TotalTokens
		}
	}

	return status, nil
}

// StatusJSON returns status as JSON string
func (d *Daemon) StatusJSON() (string, error) {
	status, err := d.Status()
	if err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(status, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// writePIDFile writes the current process ID to the PID file
func (d *Daemon) writePIDFile() error {
	pid := os.Getpid()
	return os.WriteFile(d.config.PIDFile, []byte(strconv.Itoa(pid)), 0644)
}

// readPIDFile reads the PID from the PID file
func (d *Daemon) readPIDFile() (int, error) {
	data, err := os.ReadFile(d.config.PIDFile)
	if err != nil {
		return 0, err
	}

	pid, err := strconv.Atoi(string(data))
	if err != nil {
		return 0, fmt.Errorf("invalid PID file content: %w", err)
	}

	return pid, nil
}

// isProcessRunning checks if a process with the given PID is running
func (d *Daemon) isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}

	if runtime.GOOS == "windows" {
		// On Windows, FindProcess succeeds even for non-existent PIDs.
		// We check by trying to terminate with signal 0 via Windows API
		// or by running tasklist. The simplest approach is to use tasklist.
		cmd := exec.Command("tasklist", "/FI", fmt.Sprintf("PID eq %d", pid), "/NH")
		output, err := cmd.Output()
		if err != nil {
			return false
		}
		// If process exists, output contains the process info
		// If not, it contains "INFO: No tasks are running..."
		return !strings.Contains(string(output), "INFO:")
	}

	// On Unix, FindProcess always succeeds, so we need to send signal 0
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

// cleanup removes PID file and performs other cleanup
func (d *Daemon) cleanup() {
	// Mark agent as idle in dashboard
	if err := d.unregisterAgent(); err != nil {
		d.logger.Printf("Failed to unregister agent: %v", err)
	}

	if err := os.Remove(d.config.PIDFile); err != nil && !os.IsNotExist(err) {
		d.logger.Printf("Failed to remove PID file: %v", err)
	}
	d.logger.Println("Cleanup complete")
}

// registerAgent registers the coordinator as an agent in the collaboration hub
func (d *Daemon) registerAgent() error {
	if d.msgStore == nil {
		return nil // No store available
	}

	db := d.msgStore.DB()
	if db == nil {
		return nil
	}

	now := time.Now().Unix()

	// Register/update agent
	_, err := db.Exec(`
		INSERT INTO agents (id, label, status, created_at, updated_at, last_active_at, config_json)
		VALUES ('coordinator', 'Coordinator Daemon', 'running', ?, ?, ?, '{}')
		ON CONFLICT(id) DO UPDATE SET status='running', updated_at=?, last_active_at=?
	`, now, now, now, now, now)

	if err != nil {
		return err
	}

	// Create instance history entry
	d.instanceID = fmt.Sprintf("coord_%d", now)
	_, err = db.Exec(`
		INSERT INTO instance_history (id, agent_id, instance_id, started_at)
		VALUES (?, 'coordinator', ?, ?)
	`, d.instanceID, d.instanceID, now)

	if err != nil {
		d.logger.Printf("Warning: Failed to record instance history: %v", err)
	}

	d.logger.Println("Registered as agent in collaboration hub")
	return nil
}

// unregisterAgent marks the coordinator as idle in the collaboration hub
func (d *Daemon) unregisterAgent() error {
	if d.msgStore == nil {
		return nil
	}

	db := d.msgStore.DB()
	if db == nil {
		return nil
	}

	now := time.Now().Unix()

	// Update agent status
	_, err := db.Exec(`
		UPDATE agents SET status='idle', updated_at=? WHERE id='coordinator'
	`, now)

	// Complete instance history entry
	if d.instanceID != "" {
		_, histErr := db.Exec(`
			UPDATE instance_history SET ended_at=?, exit_code=0
			WHERE id=?
		`, now, d.instanceID)
		if histErr != nil {
			d.logger.Printf("Warning: Failed to update instance history: %v", histErr)
		}
	}

	return err
}

// IncrementTasksRun increments the tasks run counter
func (d *Daemon) IncrementTasksRun() {
	d.tasksRun++
}

// GetLogger returns the daemon's logger
func (d *Daemon) GetLogger() *log.Logger {
	return d.logger
}

// GetContext returns the daemon's context
func (d *Daemon) GetContext() context.Context {
	return d.ctx
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

// GetActiveTaskMetrics returns metrics for all currently running tasks
func (d *Daemon) GetActiveTaskMetrics() []*ResourceMetrics {
	if d.resourceRegistry == nil {
		return nil
	}
	return d.resourceRegistry.GetAllMetrics()
}

// GetResourceRegistry returns the resource tracker registry for external access
func (d *Daemon) GetResourceRegistry() *ResourceTrackerRegistry {
	return d.resourceRegistry
}
