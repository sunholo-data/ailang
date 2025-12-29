package coordinator

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"syscall"
	"time"

	"github.com/sunholo/ailang/internal/messaging"
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
}

// Daemon is the coordinator daemon
type Daemon struct {
	config    *Config
	logger    *log.Logger
	ctx       context.Context
	cancel    context.CancelFunc
	startedAt time.Time
	tasksRun  int

	// Task processing components
	msgStore     *messaging.Store
	msgAdapter   *InboxMessageAdapter
	analyzer     *TaskAnalyzer
	worktreeMgr  *WorktreeManager
	taskStore    Store
	executor     *TaskExecutor

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
		config: config,
		logger: logger,
		ctx:    ctx,
		cancel: cancel,
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

	// Register as an agent in the collaboration hub
	if err := d.registerAgent(); err != nil {
		d.logger.Printf("Warning: Failed to register agent: %v", err)
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

// initTaskProcessing initializes the message adapter, analyzer, and store
func (d *Daemon) initTaskProcessing() error {
	// Open collaboration database
	adapter, store, err := OpenDefaultInboxAdapter("user")
	if err != nil {
		return fmt.Errorf("failed to open inbox adapter: %w", err)
	}
	d.msgAdapter = adapter
	d.msgStore = store

	// Initialize analyzer
	d.analyzer = NewTaskAnalyzer(0.8)

	// Initialize task store
	dbPath := filepath.Join(d.config.StateDir, "coordinator.db")
	taskStore, err := NewSQLiteStore(dbPath)
	if err != nil {
		return fmt.Errorf("failed to open task store: %w", err)
	}
	d.taskStore = taskStore

	// Initialize worktree manager
	worktreeMgr, err := NewWorktreeManager("", filepath.Join(d.config.StateDir, "worktrees", "coordinator"), d.config.MaxWorktrees)
	if err != nil {
		return fmt.Errorf("failed to create worktree manager: %w", err)
	}
	d.worktreeMgr = worktreeMgr

	// Initialize task executor with available providers
	executor, err := DefaultTaskExecutor()
	if err != nil {
		d.logger.Printf("Warning: Failed to create task executor: %v", err)
		d.logger.Println("Task execution disabled - will queue tasks but not execute them")
	} else {
		d.executor = executor
		d.logger.Printf("Task executor initialized with providers: %v", executor.ListProviders())
	}

	return nil
}

// pollAndProcessTasks polls for new messages and queues them as tasks
func (d *Daemon) pollAndProcessTasks() error {
	messages, err := d.msgAdapter.ListUnread()
	if err != nil {
		return fmt.Errorf("failed to list unread messages: %w", err)
	}

	if len(messages) == 0 {
		return nil
	}

	d.logger.Printf("Found %d unread messages", len(messages))

	for _, msg := range messages {
		// Create a Task for the analyzer
		taskID := fmt.Sprintf("task-%s", msg.ID[:8])
		taskInput := &Task{
			ID:        taskID,
			Title:     msg.Title,
			Content:   msg.Content,
			MessageID: msg.ID,
			CreatedAt: msg.CreatedAt,
		}

		// Analyze the message to classify it
		analyzed := d.analyzer.Analyze(taskInput)

		// Create a task record
		task := &TaskRecord{
			ID:        taskID,
			MessageID: msg.ID,
			Title:     msg.Title,
			Content:   msg.Content,
			Type:      analyzed.Type,
			Priority:  CalculatePriority(analyzed),
			Status:    TaskStatusPending,
			CreatedAt: msg.CreatedAt,
		}

		// Check for duplicates
		fingerprint := analyzed.Fingerprint
		if fingerprint != 0 {
			if dup, _ := d.taskStore.FindDuplicateTask(d.ctx, fingerprint, 0.9); dup != nil {
				d.logger.Printf("Skipping duplicate task for message %s (similar to task %s)", msg.ID, dup.ID)
				// Mark message as read since we're skipping it
				_ = d.msgAdapter.MarkAsRead(msg.ID)
				continue
			}
		}

		// Create a thread in collaboration.db for dashboard visibility
		thread, err := d.msgStore.CreateThread(
			msg.Title,           // title
			"ailang_instance",   // createdByType (constraint: 'human' or 'ailang_instance')
			"coordinator",       // createdByID
			"coordinator",       // targetAgent
		)
		if err != nil {
			d.logger.Printf("Failed to create thread for task %s: %v", taskID, err)
			// Continue anyway - thread is for visibility, not required for task
		} else {
			task.ThreadID = thread.ID
			d.logger.Printf("Created thread %s for task %s", thread.ID, taskID)
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

		d.logger.Printf("Created task %s (type: %s, priority: %d, thread: %s) from message %s",
			task.ID, task.Type, task.Priority, task.ThreadID, msg.ID)

		// Mark message as read
		if err := d.msgAdapter.MarkAsRead(msg.ID); err != nil {
			d.logger.Printf("Failed to mark message as read: %v", err)
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
			MessageID: task.MessageID,
			CreatedAt: task.CreatedAt,
		},
		Type: task.Type,
	}

	// Get workspace from worktree manager
	var worktree *Worktree
	if d.worktreeMgr != nil {
		var wtErr error
		worktree, wtErr = d.worktreeMgr.CreateWorktree(task.ID)
		if wtErr != nil {
			d.logger.Printf("Warning: Failed to create worktree for task %s: %v", task.ID, wtErr)
			// Continue without worktree - will use current directory
		}
	}

	opts := &ExecuteOptions{
		Timeout: 10 * time.Minute, // 10 minute timeout per task
	}
	if worktree != nil {
		opts.Workspace = worktree.Path
	}

	// Execute the task
	result, err := d.executor.ExecuteWithRetry(d.ctx, analyzed, opts, 2)

	// Cleanup worktree
	if worktree != nil && d.worktreeMgr != nil {
		if rmErr := d.worktreeMgr.RemoveWorktree(task.ID); rmErr != nil {
			d.logger.Printf("Warning: Failed to remove worktree: %v", rmErr)
		}
	}

	if err != nil {
		return fmt.Errorf("executor error: %w", err)
	}

	// Update task status based on result
	if result.Success {
		if err := d.taskStore.MarkTaskCompleted(d.ctx, task.ID, result); err != nil {
			d.logger.Printf("Warning: Failed to mark task completed: %v", err)
		}
		d.logger.Printf("Task %s completed successfully (cost: $%.4f, tokens: %d)",
			task.ID, result.Cost, result.TokensUsed)
	} else {
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

// postTaskResult posts the execution result to the task's thread
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
				"✅ Provider: %s\n"+
				"⏱️ Duration: %s\n"+
				"💰 Cost: $%.4f\n"+
				"🔢 Tokens: %d\n\n"+
				"**Output:**\n```\n%s\n```",
				result.Provider, result.Duration, result.Cost, result.TokensUsed, result.Output)

			if len(result.FilesCreated) > 0 {
				content += fmt.Sprintf("\n\n**Files Created:** %v", result.FilesCreated)
			}
			if len(result.FilesModified) > 0 {
				content += fmt.Sprintf("\n\n**Files Modified:** %v", result.FilesModified)
			}
		} else {
			kind = "error"
			content = fmt.Sprintf("**Task Failed**\n\n"+
				"❌ Provider: %s\n"+
				"⏱️ Duration: %s\n\n"+
				"**Error:**\n```\n%s\n```",
				result.Provider, result.Duration, result.Error)
		}
	} else {
		kind = "error"
		content = "**Task Failed**\n\n❌ Unknown error"
	}

	// Create metadata with execution stats
	metadataJSON := ""
	if result != nil {
		metadata := map[string]interface{}{
			"execution_result": map[string]interface{}{
				"success":      result.Success,
				"duration_ms":  result.Duration.Milliseconds(),
				"cost_usd":     result.Cost,
				"total_tokens": result.TokensUsed,
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
	}
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
